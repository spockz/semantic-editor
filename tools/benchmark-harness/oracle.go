// Package main implements the evaluation harness and multi-level correctness oracle for semedit benchmarks.
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/pipeline"
	"slices"
	"strconv"
	"strings"
	"time"
)

// TaskMetadata models the structured frontmatter embedded within each txtar benchmark archive.
type TaskMetadata struct {
	TaskID                 string            `json:"task_id"`
	Category               string            `json:"category"`
	Instruction            string            `json:"instruction"`
	VerificationConstraint string            `json:"verification_constraint,omitempty"`
	Contexts               []string          `json:"contexts,omitempty"`
	PromptVariants         map[string]string `json:"prompt_variants,omitempty"`
	InteractiveFollowups   []string          `json:"interactive_followups,omitempty"`
	Oracle                 OracleConfig      `json:"oracle"`
}

// OracleConfig specifies the validation criteria across evaluation levels.
type OracleConfig struct {
	MutationPolicy          MutationPolicyConfig `json:"level_1_mutation_policy"`
	AST                     ASTConfig            `json:"level_2_ast"`
	Build                   BuildConfig          `json:"level_3_build"`
	Test                    TestConfig           `json:"level_4_test"`
	DiagnosticExpectedTools []string             `json:"diagnostic_expected_tools,omitempty"`
}

// MutationPolicyConfig defines file modification constraints.
type MutationPolicyConfig struct {
	DisallowedFiles []string `json:"disallowed_files"`
}

// SymbolAdjacency ensures relative declaration ordering.
type SymbolAdjacency struct {
	First  string `json:"first"`
	Second string `json:"second"`
}

// ASTConfig specifies AST structural and symbol invariants.
type ASTConfig struct {
	File                  string          `json:"file"`
	MustContainSymbols    []string        `json:"must_contain_symbols"`
	MustNotContainSymbols []string        `json:"must_not_contain_symbols"`
	MustContainImports    []string        `json:"must_contain_imports"`
	MustNotContainImports []string        `json:"must_not_contain_imports"`
	SymbolBefore          SymbolAdjacency `json:"symbol_before"`
}

// BuildConfig specifies compiler requirements.
type BuildConfig struct {
	CleanCompile bool `json:"clean_compile"`
}

// TestConfig specifies test suite requirements.
type TestConfig struct {
	PassTests     bool   `json:"pass_tests"`
	HiddenTestDir string `json:"hidden_test_dir"`
}

// OracleResult records evaluation outcomes across all validation levels.
type OracleResult struct {
	Passed       bool          `json:"passed"`
	Level1Policy bool          `json:"level_1_policy"`
	Level2AST    bool          `json:"level_2_ast"`
	Level3Build  bool          `json:"level_3_build"`
	Level4Test   bool          `json:"level_4_test"`
	FailureStage string        `json:"failure_stage,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
}

// Task represents an unpacked benchmark scenario ready for agent execution.
type Task struct {
	Metadata       TaskMetadata
	Archive        *Archive
	contextVariant string
}

// ParseTask parses a txtar archive containing YAML frontmatter in its initial comment.
func ParseTask(data []byte) (*Task, error) {
	ar := ParseArchive(data)
	if len(ar.Comment) == 0 {
		return nil, errors.New("txtar benchmark fixture missing frontmatter comment")
	}

	meta, err := parseYAMLFrontmatter(ar.Comment)
	if err != nil {
		return nil, fmt.Errorf("unmarshal frontmatter yaml: %w", err)
	}

	if meta.TaskID == "" {
		return nil, errors.New("benchmark fixture missing task_id")
	}

	return &Task{
		Metadata: meta,
		Archive:  ar,
	}, nil
}

// parseYAMLFrontmatter provides zero-dependency YAML parsing for benchmark task specs.
func parseYAMLFrontmatter(comment []byte) (TaskMetadata, error) {
	var meta TaskMetadata
	scanner := bufio.NewScanner(bytes.NewReader(comment))

	var currentPath []string
	var cleanCompileSeen, passTestsSeen bool

	for scanner.Scan() {
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			continue
		}

		if after, ok := strings.CutPrefix(trimmed, "#"); ok {
			trimmed = strings.TrimSpace(after)
			if trimmed == "" || strings.HasPrefix(trimmed, "exec ") || strings.HasPrefix(trimmed, "cmp ") {
				continue
			}
			if hashIdx := strings.Index(rawLine, "#"); hashIdx >= 0 {
				rawLine = strings.TrimPrefix(rawLine[hashIdx+1:], " ")
			}
		}

		indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))
		level := indent / 2

		if after, ok := strings.CutPrefix(trimmed, "- "); ok {
			itemVal := strings.Trim(after, `"'`)
			targetSlice := strings.Join(currentPath, ".")
			switch targetSlice {
			case "contexts":
				meta.Contexts = append(meta.Contexts, itemVal)
			case "interactive_followups":
				meta.InteractiveFollowups = append(meta.InteractiveFollowups, itemVal)
			case "oracle.diagnostic_expected_tools":
				meta.Oracle.DiagnosticExpectedTools = append(meta.Oracle.DiagnosticExpectedTools, itemVal)
			case "oracle.level_1_mutation_policy.disallowed_files":
				meta.Oracle.MutationPolicy.DisallowedFiles = append(meta.Oracle.MutationPolicy.DisallowedFiles, itemVal)
			case "oracle.level_2_ast.must_contain_symbols":
				meta.Oracle.AST.MustContainSymbols = append(meta.Oracle.AST.MustContainSymbols, itemVal)
			case "oracle.level_2_ast.must_not_contain_symbols":
				meta.Oracle.AST.MustNotContainSymbols = append(meta.Oracle.AST.MustNotContainSymbols, itemVal)
			case "oracle.level_2_ast.must_contain_imports":
				meta.Oracle.AST.MustContainImports = append(meta.Oracle.AST.MustContainImports, itemVal)
			case "oracle.level_2_ast.must_not_contain_imports":
				meta.Oracle.AST.MustNotContainImports = append(meta.Oracle.AST.MustNotContainImports, itemVal)
			}
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		key := strings.TrimSpace(parts[0])
		val := ""
		if len(parts) > 1 {
			val = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		}

		if level < len(currentPath) {
			currentPath = currentPath[:level]
		}
		currentPath = append(currentPath, key)

		fullPath := strings.Join(currentPath, ".")
		if strings.HasPrefix(fullPath, "prompt_variants.") && val != "" {
			varName := strings.TrimPrefix(fullPath, "prompt_variants.")
			if meta.PromptVariants == nil {
				meta.PromptVariants = make(map[string]string)
			}
			meta.PromptVariants[varName] = val
		}

		switch fullPath {
		case "task_id":
			meta.TaskID = val
		case "category":
			meta.Category = val
		case "instruction":
			meta.Instruction = val
		case "verification_constraint":
			meta.VerificationConstraint = val
		case "oracle.level_2_ast.file":
			meta.Oracle.AST.File = val
		case "oracle.level_2_ast.symbol_before.first":
			meta.Oracle.AST.SymbolBefore.First = val
		case "oracle.level_2_ast.symbol_before.second":
			meta.Oracle.AST.SymbolBefore.Second = val
		case "oracle.level_3_build.clean_compile":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return TaskMetadata{}, fmt.Errorf("oracle.level_3_build.clean_compile: %w", err)
			}
			cleanCompileSeen = true
			meta.Oracle.Build.CleanCompile = b
		case "oracle.level_4_test.pass_tests":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return TaskMetadata{}, fmt.Errorf("oracle.level_4_test.pass_tests: %w", err)
			}
			passTestsSeen = true
			meta.Oracle.Test.PassTests = b
		case "oracle.level_4_test.hidden_test_dir":
			meta.Oracle.Test.HiddenTestDir = val
		}
	}

	if err := scanner.Err(); err != nil {
		return TaskMetadata{}, err
	}
	if !cleanCompileSeen {
		return TaskMetadata{}, errors.New("missing required oracle.level_3_build.clean_compile")
	}
	if !passTestsSeen {
		return TaskMetadata{}, errors.New("missing required oracle.level_4_test.pass_tests")
	}
	return meta, nil
}

// ExtractTo extracts all non-golden archive files to target directory.
func (t *Task) ExtractTo(targetDir string) error {
	contexts := supportedTaskContexts(t)
	variant := t.contextVariant
	if variant == "" {
		variant = "small"
		if !slices.Contains(contexts, variant) && len(contexts) > 0 {
			variant = contexts[0]
		}
	}
	return t.ExtractVariantTo(targetDir, variant)
}

// ExtractVariantTo extracts archive files and dynamically injects overlay files for large contexts.
func (t *Task) ExtractVariantTo(targetDir, variant string) error {
	contexts := supportedTaskContexts(t)
	variantContext, _, _ := strings.Cut(variant, ":")
	baseContext, valid := normalizedContextBase(variantContext)
	if !valid || !slices.Contains(contexts, baseContext) {
		return fmt.Errorf("task %s does not support context %q (available: %s)", t.Metadata.TaskID, variantContext, strings.Join(contexts, ","))
	}
	for _, file := range t.Archive.Files {
		if strings.HasPrefix(file.Name, "want/") || strings.HasPrefix(file.Name, "want\\") {
			continue
		}
		destPath := filepath.Join(targetDir, filepath.FromSlash(file.Name))
		// #nosec G304 G703 -- archive paths are trusted fixture contents and extraction is constrained to a caller-owned work directory.
		if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return fmt.Errorf("create dir for %s: %w", file.Name, err)
		}
		// #nosec G304 G703 -- archive paths are trusted fixture contents and extraction is constrained to a caller-owned work directory.
		if err := pipeline.WriteAtomic(destPath, file.Data); err != nil {
			return fmt.Errorf("write file %s: %w", file.Name, err)
		}
	}
	hasSmall := slices.Contains(contexts, "small")
	hasLarge := slices.Contains(contexts, "large")
	if baseContext == "large" && hasLarge && hasSmall {
		for relPath, content := range LargeContextOverlayFiles(t.Metadata.TaskID) {
			destPath := filepath.Join(targetDir, filepath.FromSlash(relPath))
			// #nosec G304 G703 -- fixture overlay destinations are fixed by harness code.
			if _, err := os.Stat(destPath); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat overlay destination %s: %w", relPath, err)
			}
			// #nosec G304 G703 -- fixture overlay destinations are fixed by harness code.
			if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
				return fmt.Errorf("create dir for %s: %w", relPath, err)
			}
			// #nosec G304 G703 -- fixture overlay destinations are fixed by harness code.
			if err := pipeline.WriteAtomic(destPath, []byte(content)); err != nil {
				return fmt.Errorf("write overlay file %s: %w", relPath, err)
			}
		}
	}
	return nil
}

// Evaluate runs the multi-level correctness oracle against the target workspace directory.
func Evaluate(ctx context.Context, task *Task, workDir string, modifiedFiles []string) (result *OracleResult, retErr error) {
	start := time.Now()
	res := &OracleResult{
		Passed: false,
	}

	// Level 1: Mutation Policy
	if slices.Contains(task.Metadata.Oracle.MutationPolicy.DisallowedFiles, "*") && len(modifiedFiles) > 0 {
		res.FailureStage = "level_1_mutation_policy"
		res.ErrorMessage = fmt.Sprintf("workspace must remain unchanged; modified: %s", strings.Join(modifiedFiles, ", "))
		res.Duration = time.Since(start)
		return res, nil
	}
	if len(task.Metadata.Oracle.MutationPolicy.DisallowedFiles) > 0 {
		for _, mod := range modifiedFiles {
			relMod := filepath.ToSlash(mod)
			if slices.Contains(task.Metadata.Oracle.MutationPolicy.DisallowedFiles, relMod) {
				res.FailureStage = "level_1_mutation_policy"
				res.ErrorMessage = fmt.Sprintf("unauthorized modification of protected file: %s", relMod)
				res.Duration = time.Since(start)
				return res, nil
			}
		}
	}
	res.Level1Policy = true

	evalDir, cleanup, err := copyWorkspaceForOracle(workDir)
	if err != nil {
		return nil, fmt.Errorf("prepare private oracle workspace: %w", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("cleanup private oracle workspace: %w", err))
		}
	}()

	// Level 2: AST Invariants
	if err := evaluateAST(task.Metadata.Oracle.AST, evalDir); err != nil {
		res.FailureStage = "level_2_ast"
		res.ErrorMessage = err.Error()
		res.Duration = time.Since(start)
		return res, nil
	}
	res.Level2AST = true

	// Level 3: Isolated Compilation
	if task.Metadata.Oracle.Build.CleanCompile {
		cmd := exec.CommandContext(ctx, "go", "build", "./...")
		cmd.Dir = evalDir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			res.FailureStage = "level_3_build"
			res.ErrorMessage = fmt.Sprintf("go build failed: %v\n%s", err, string(out))
			res.Duration = time.Since(start)
			return res, nil
		}
	}
	res.Level3Build = true

	// Level 4: Hidden Tests
	if task.Metadata.Oracle.Test.PassTests {
		if err := installHiddenTests(evalDir, task.Metadata.Oracle.Test.HiddenTestDir); err != nil {
			res.FailureStage = "level_4_test"
			res.ErrorMessage = fmt.Sprintf("install hidden tests: %v", err)
			res.Duration = time.Since(start)
			return res, nil
		}
		cmd := exec.CommandContext(ctx, "go", "test", "./...")
		cmd.Dir = evalDir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			res.FailureStage = "level_4_test"
			res.ErrorMessage = fmt.Sprintf("go test failed: %v\n%s", err, string(out))
			res.Duration = time.Since(start)
			return res, nil
		}
	}
	res.Level4Test = true
	res.Passed = true
	res.Duration = time.Since(start)
	return res, nil
}

func copyWorkspaceForOracle(workDir string) (string, func() error, error) {
	sourceRoot, err := filepath.Abs(workDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolve workspace: %w", err)
	}
	sourceFS, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return "", nil, fmt.Errorf("open workspace: %w", err)
	}
	scratchDir := filepath.Join(sourceRoot, ".scratch")
	scratchCreated := false
	if info, statErr := os.Lstat(scratchDir); errors.Is(statErr, os.ErrNotExist) {
		if err := os.Mkdir(scratchDir, 0o700); err != nil {
			_ = sourceFS.Close()
			return "", nil, fmt.Errorf("create workspace scratch directory: %w", err)
		}
		scratchCreated = true
	} else if statErr != nil {
		_ = sourceFS.Close()
		return "", nil, fmt.Errorf("inspect workspace scratch directory: %w", statErr)
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		_ = sourceFS.Close()
		return "", nil, errors.New("workspace .scratch is not a regular directory")
	}
	removeScratch := func() error {
		if scratchCreated {
			if err := os.Remove(scratchDir); !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return nil
	}
	target, err := os.MkdirTemp(scratchDir, "oracle-*")
	if err != nil {
		return "", nil, errors.Join(fmt.Errorf("create private workspace: %w", err), sourceFS.Close(), removeScratch())
	}
	targetFS, err := os.OpenRoot(target)
	if err != nil {
		return "", nil, errors.Join(fmt.Errorf("open private workspace: %w", err), sourceFS.Close(), os.RemoveAll(target), removeScratch())
	}
	cleanup := func() error {
		closeSourceErr, closeTargetErr := sourceFS.Close(), targetFS.Close()
		removeErr := os.RemoveAll(target)
		var scratchErr error
		if scratchCreated {
			scratchErr = os.Remove(scratchDir)
			if errors.Is(scratchErr, os.ErrNotExist) {
				scratchErr = nil
			}
		}
		return errors.Join(closeSourceErr, closeTargetErr, removeErr, scratchErr)
	}
	err = filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return fmt.Errorf("relativize %s: %w", path, err)
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".scratch" {
				return filepath.SkipDir
			}
			return targetFS.MkdirAll(rel, 0o750)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := sourceFS.ReadFile(rel)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		if err := targetFS.MkdirAll(filepath.Dir(rel), 0o750); err != nil {
			return fmt.Errorf("create parent for %s: %w", rel, err)
		}
		if err := targetFS.WriteFile(rel, data, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		return nil
	})
	if err != nil {
		return "", nil, errors.Join(fmt.Errorf("copy workspace: %w", err), cleanup())
	}
	return target, cleanup, nil
}

// installHiddenTests copies task-owned acceptance tests only after the agent has finished.
func installHiddenTests(workDir, sourceDir string) error {
	if sourceDir == "" {
		return nil
	}
	sourceRoot, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("resolve hidden test directory: %w", err)
	}
	sourceFS, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return fmt.Errorf("open hidden test directory: %w", err)
	}
	defer func() {
		_ = sourceFS.Close()
	}()

	workFS, err := os.OpenRoot(workDir)
	if err != nil {
		return fmt.Errorf("open benchmark workspace: %w", err)
	}
	defer func() {
		_ = workFS.Close()
	}()

	return filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return fmt.Errorf("relativize hidden test %s: %w", path, err)
		}
		if err := workFS.MkdirAll(filepath.Dir(rel), 0o750); err != nil {
			return fmt.Errorf("create hidden test directory for %s: %w", rel, err)
		}
		data, err := sourceFS.ReadFile(rel)
		if err != nil {
			return fmt.Errorf("read hidden test %s: %w", rel, err)
		}
		if err := workFS.WriteFile(rel, data, 0o600); err != nil {
			return fmt.Errorf("write hidden test %s: %w", rel, err)
		}
		return nil
	})
}

func evaluateAST(cfg ASTConfig, workDir string) error {
	if cfg.File == "" {
		return nil
	}

	targetPath := filepath.Join(workDir, filepath.FromSlash(cfg.File))
	src, err := os.ReadFile(filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("read target file for AST validation %s: %w", cfg.File, err)
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, targetPath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse AST of %s: %w", cfg.File, err)
	}

	// Collect declared symbols and functions
	declaredSymbols := make(map[string]int) // symbol -> declaration order index

	for idx, decl := range fileNode.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			symName := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recvType := receiverTypeName(d.Recv.List[0].Type)
				if recvType != "" {
					symName = recvType + "." + d.Name.Name
				}
			}
			declaredSymbols[symName] = idx

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					declaredSymbols[s.Name.Name] = idx
					if iface, ok := s.Type.(*ast.InterfaceType); ok && iface.Methods != nil {
						for _, m := range iface.Methods.List {
							for _, id := range m.Names {
								interfaceMethod := s.Name.Name + "." + id.Name
								declaredSymbols[interfaceMethod] = idx
							}
						}
					}
				case *ast.ValueSpec:
					for _, id := range s.Names {
						declaredSymbols[id.Name] = idx
					}
				}
			}
		}
	}

	// Must contain symbols
	for _, sym := range cfg.MustContainSymbols {
		if _, found := declaredSymbols[sym]; !found {
			// Check if symbol appears anywhere in AST (e.g. struct field or selector)
			if !astContainsIdent(fileNode, sym) {
				return fmt.Errorf("required symbol %q not found in %s", sym, cfg.File)
			}
		}
	}

	// Must not contain symbols
	for _, sym := range cfg.MustNotContainSymbols {
		if _, found := declaredSymbols[sym]; found {
			return fmt.Errorf("disallowed symbol %q remains declared in %s", sym, cfg.File)
		}
		if astContainsExactIdent(fileNode, sym) {
			return fmt.Errorf("disallowed symbol reference %q persists in %s", sym, cfg.File)
		}
	}

	// Imports validation
	currentImports := make(map[string]bool)
	for _, imp := range fileNode.Imports {
		path := strings.Trim(imp.Path.Value, `"'`)
		currentImports[path] = true
	}

	for _, reqImp := range cfg.MustContainImports {
		if !currentImports[reqImp] {
			return fmt.Errorf("required import %q absent from %s", reqImp, cfg.File)
		}
	}

	for _, disImp := range cfg.MustNotContainImports {
		if currentImports[disImp] {
			return fmt.Errorf("disallowed import %q present in %s", disImp, cfg.File)
		}
	}

	// Symbol ordering validation
	if cfg.SymbolBefore.First != "" && cfg.SymbolBefore.Second != "" {
		firstIdx, firstOk := declaredSymbols[cfg.SymbolBefore.First]
		secondIdx, secondOk := declaredSymbols[cfg.SymbolBefore.Second]
		if !firstOk {
			return fmt.Errorf("ordering validation failed: first symbol %q not found in declarations", cfg.SymbolBefore.First)
		}
		if !secondOk {
			return fmt.Errorf("ordering validation failed: second symbol %q not found in declarations", cfg.SymbolBefore.Second)
		}
		if firstIdx >= secondIdx {
			return fmt.Errorf("ordering validation failed: %q (idx %d) must precede %q (idx %d)", cfg.SymbolBefore.First, firstIdx, cfg.SymbolBefore.Second, secondIdx)
		}
	}

	return nil
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}

func astContainsIdent(node ast.Node, sym string) bool {
	parts := strings.Split(sym, ".")
	targetIdent := parts[len(parts)-1]

	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if id, ok := n.(*ast.Ident); ok && id.Name == targetIdent {
			found = true
			return false
		}
		return true
	})
	return found
}

func astContainsExactIdent(node ast.Node, sym string) bool {
	parts := strings.Split(sym, ".")
	targetIdent := parts[len(parts)-1]

	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if id, ok := n.(*ast.Ident); ok && id.Name == targetIdent {
			found = true
			return false
		}
		return true
	})
	return found
}
