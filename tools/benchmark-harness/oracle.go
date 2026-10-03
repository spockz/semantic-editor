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

	"gopkg.in/yaml.v3"
)

// TaskMetadata models the structured frontmatter embedded within each txtar benchmark archive.
type TaskMetadata struct {
	TaskID                  string            `json:"task_id"`
	Category                string            `json:"category"`
	Instruction             string            `json:"instruction"`
	VerificationConstraint  string            `json:"verification_constraint,omitempty"`
	Contexts                []string          `json:"contexts,omitempty"`
	PromptVariants          map[string]string `json:"prompt_variants,omitempty"`
	InteractiveFollowups    []string          `json:"interactive_followups,omitempty"`
	InteractiveMode         string            `json:"interactive_mode,omitempty"`
	StagedInitialTotalEdits int               `json:"staged_initial_total_edits,omitempty"`
	StagedFollowups         []StagedFollowup  `json:"staged_followups,omitempty"`
	Oracle                  OracleConfig      `json:"oracle"`
}

type StagedFollowup struct {
	TotalEdits  int          `json:"total_edits" yaml:"total_edits"`
	Instruction string       `json:"instruction" yaml:"instruction"`
	Oracle      OracleConfig `json:"oracle"      yaml:"oracle"`
}

// OracleConfig specifies the validation criteria across evaluation levels.
type OracleConfig struct {
	MutationPolicy          MutationPolicyConfig `json:"level_1_mutation_policy"             yaml:"level_1_mutation_policy"`
	AST                     ASTConfig            `json:"level_2_ast"                         yaml:"level_2_ast"`
	Build                   BuildConfig          `json:"level_3_build"                       yaml:"level_3_build"`
	Test                    TestConfig           `json:"level_4_test"                        yaml:"level_4_test"`
	DiagnosticExpectedTools []string             `json:"diagnostic_expected_tools,omitempty"`
}

// MutationPolicyConfig defines file modification constraints.
type MutationPolicyConfig struct {
	DisallowedFiles []string `json:"disallowed_files" yaml:"disallowed_files"`
}

// SymbolAdjacency ensures relative declaration ordering.
type SymbolAdjacency struct {
	First  string `json:"first"  yaml:"first"`
	Second string `json:"second" yaml:"second"`
}

// ASTConfig specifies AST structural and symbol invariants.
type ASTConfig struct {
	File                  string          `json:"file"                     yaml:"file"`
	MustContainSymbols    []string        `json:"must_contain_symbols"     yaml:"must_contain_symbols"`
	MustNotContainSymbols []string        `json:"must_not_contain_symbols" yaml:"must_not_contain_symbols"`
	MustCallFunctions     []string        `json:"must_call_functions"      yaml:"must_call_functions"`
	MustContainImports    []string        `json:"must_contain_imports"     yaml:"must_contain_imports"`
	MustNotContainImports []string        `json:"must_not_contain_imports" yaml:"must_not_contain_imports"`
	SymbolBefore          SymbolAdjacency `json:"symbol_before"            yaml:"symbol_before"`
}

// BuildConfig specifies compiler requirements.
type BuildConfig struct {
	CleanCompile bool `json:"clean_compile" yaml:"clean_compile"`
}

// TestConfig specifies test suite requirements.
type TestConfig struct {
	PassTests     bool   `json:"pass_tests"      yaml:"pass_tests"`
	HiddenTestDir string `json:"hidden_test_dir" yaml:"hidden_test_dir"`
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

type stagedFollowupYAML struct {
	TotalEdits  int    `yaml:"total_edits"`
	Instruction string `yaml:"instruction"`
	Oracle      struct {
		MutationPolicy MutationPolicyConfig `yaml:"level_1_mutation_policy"`
		AST            ASTConfig            `yaml:"level_2_ast"`
		Build          struct {
			CleanCompile *bool `yaml:"clean_compile"`
		} `yaml:"level_3_build"`
		Test struct {
			PassTests     *bool  `yaml:"pass_tests"`
			HiddenTestDir string `yaml:"hidden_test_dir"`
		} `yaml:"level_4_test"`
	} `yaml:"oracle"`
}

// parseYAMLFrontmatter provides zero-dependency YAML parsing for benchmark task specs.
func parseYAMLFrontmatter(comment []byte) (TaskMetadata, error) {
	parser := taskFrontmatterParser{}
	scanner := bufio.NewScanner(bytes.NewReader(comment))
	for scanner.Scan() {
		if err := parser.parseLine(scanner.Text()); err != nil {
			return TaskMetadata{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return TaskMetadata{}, err
	}
	if !parser.cleanCompileSeen {
		return TaskMetadata{}, errors.New("missing required oracle.level_3_build.clean_compile")
	}
	if !parser.passTestsSeen {
		return TaskMetadata{}, errors.New("missing required oracle.level_4_test.pass_tests")
	}
	return parseStagedFollowups(comment, parser.metadata)
}

type taskFrontmatterParser struct {
	metadata         TaskMetadata
	currentPath      []string
	cleanCompileSeen bool
	passTestsSeen    bool
}

func (parser *taskFrontmatterParser) parseLine(rawLine string) error {
	trimmed := strings.TrimSpace(rawLine)
	if trimmed == "" || parser.isIgnoredComment(&rawLine, &trimmed) {
		return nil
	}
	level := frontmatterIndentLevel(rawLine)
	if item, ok := strings.CutPrefix(trimmed, "- "); ok {
		parser.appendListItem(strings.Trim(item, `"'`))
		return nil
	}
	key, value := parseFrontmatterKeyValue(trimmed)
	parser.replacePathAtLevel(level, key)
	return parser.setScalar(strings.Join(parser.currentPath, "."), value)
}

func (parser *taskFrontmatterParser) isIgnoredComment(rawLine, trimmed *string) bool {
	after, isComment := strings.CutPrefix(*trimmed, "#")
	if !isComment {
		return false
	}
	*trimmed = strings.TrimSpace(after)
	if *trimmed == "" || strings.HasPrefix(*trimmed, "exec ") || strings.HasPrefix(*trimmed, "cmp ") {
		return true
	}
	if index := strings.Index(*rawLine, "#"); index >= 0 {
		*rawLine = strings.TrimPrefix((*rawLine)[index+1:], " ")
	}
	return false
}

func frontmatterIndentLevel(rawLine string) int {
	indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))
	return indent / 2
}

func parseFrontmatterKeyValue(line string) (string, string) {
	parts := strings.SplitN(line, ":", 2)
	key := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return key, ""
	}
	value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	return key, value
}

func (parser *taskFrontmatterParser) replacePathAtLevel(level int, key string) {
	if level < len(parser.currentPath) {
		parser.currentPath = parser.currentPath[:level]
	}
	parser.currentPath = append(parser.currentPath, key)
}

func (parser *taskFrontmatterParser) appendListItem(item string) {
	metadata := &parser.metadata
	switch strings.Join(parser.currentPath, ".") {
	case "contexts":
		metadata.Contexts = append(metadata.Contexts, item)
	case "interactive_followups":
		metadata.InteractiveFollowups = append(metadata.InteractiveFollowups, item)
	case "oracle.diagnostic_expected_tools":
		metadata.Oracle.DiagnosticExpectedTools = append(metadata.Oracle.DiagnosticExpectedTools, item)
	case "oracle.level_1_mutation_policy.disallowed_files":
		metadata.Oracle.MutationPolicy.DisallowedFiles = append(metadata.Oracle.MutationPolicy.DisallowedFiles, item)
	case "oracle.level_2_ast.must_contain_symbols":
		metadata.Oracle.AST.MustContainSymbols = append(metadata.Oracle.AST.MustContainSymbols, item)
	case "oracle.level_2_ast.must_not_contain_symbols":
		metadata.Oracle.AST.MustNotContainSymbols = append(metadata.Oracle.AST.MustNotContainSymbols, item)
	case "oracle.level_2_ast.must_call_functions":
		metadata.Oracle.AST.MustCallFunctions = append(metadata.Oracle.AST.MustCallFunctions, item)
	case "oracle.level_2_ast.must_contain_imports":
		metadata.Oracle.AST.MustContainImports = append(metadata.Oracle.AST.MustContainImports, item)
	case "oracle.level_2_ast.must_not_contain_imports":
		metadata.Oracle.AST.MustNotContainImports = append(metadata.Oracle.AST.MustNotContainImports, item)
	}
}

func (parser *taskFrontmatterParser) setScalar(path, value string) error {
	if strings.HasPrefix(path, "prompt_variants.") && value != "" {
		name := strings.TrimPrefix(path, "prompt_variants.")
		if parser.metadata.PromptVariants == nil {
			parser.metadata.PromptVariants = make(map[string]string)
		}
		parser.metadata.PromptVariants[name] = value
	}
	return parser.setKnownScalar(path, value)
}

func (parser *taskFrontmatterParser) setKnownScalar(path, value string) error {
	metadata := &parser.metadata
	switch path {
	case "task_id":
		metadata.TaskID = value
	case "category":
		metadata.Category = value
	case "instruction":
		metadata.Instruction = value
	case "verification_constraint":
		metadata.VerificationConstraint = value
	case "interactive_mode":
		metadata.InteractiveMode = value
	case "staged_initial_total_edits":
		count, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("staged_initial_total_edits: %w", err)
		}
		metadata.StagedInitialTotalEdits = count
	case "oracle.level_2_ast.file":
		metadata.Oracle.AST.File = value
	case "oracle.level_2_ast.symbol_before.first":
		metadata.Oracle.AST.SymbolBefore.First = value
	case "oracle.level_2_ast.symbol_before.second":
		metadata.Oracle.AST.SymbolBefore.Second = value
	case "oracle.level_3_build.clean_compile":
		return parser.setCleanCompile(value)
	case "oracle.level_4_test.pass_tests":
		return parser.setPassTests(value)
	case "oracle.level_4_test.hidden_test_dir":
		metadata.Oracle.Test.HiddenTestDir = value
	}
	return nil
}

func (parser *taskFrontmatterParser) setCleanCompile(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("oracle.level_3_build.clean_compile: %w", err)
	}
	parser.cleanCompileSeen = true
	parser.metadata.Oracle.Build.CleanCompile = parsed
	return nil
}

func (parser *taskFrontmatterParser) setPassTests(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("oracle.level_4_test.pass_tests: %w", err)
	}
	parser.passTestsSeen = true
	parser.metadata.Oracle.Test.PassTests = parsed
	return nil
}

func parseStagedFollowups(comment []byte, meta TaskMetadata) (TaskMetadata, error) {
	if meta.InteractiveMode != "" && meta.InteractiveMode != "staged" {
		return TaskMetadata{}, fmt.Errorf("unsupported interactive_mode %q", meta.InteractiveMode)
	}
	if meta.InteractiveMode != "staged" {
		if bytes.Contains(comment, []byte("staged_followups:")) {
			return TaskMetadata{}, errors.New("staged_followups requires interactive_mode: staged")
		}
		return meta, nil
	}
	if len(meta.InteractiveFollowups) > 0 {
		return TaskMetadata{}, errors.New("staged_followups cannot be combined with interactive_followups")
	}
	if meta.StagedInitialTotalEdits < 1 {
		return TaskMetadata{}, errors.New("staged_initial_total_edits must be positive")
	}
	var staged struct {
		Followups []stagedFollowupYAML `yaml:"staged_followups"`
	}
	if err := yaml.Unmarshal(comment, &staged); err != nil {
		return TaskMetadata{}, fmt.Errorf("parse staged followups: %w", err)
	}
	if len(staged.Followups) == 0 {
		return TaskMetadata{}, errors.New("staged mode requires staged_followups")
	}
	previous := meta.StagedInitialTotalEdits
	for index, followup := range staged.Followups {
		if followup.TotalEdits <= previous {
			return TaskMetadata{}, fmt.Errorf("staged followup %d total_edits must exceed %d", index+1, previous)
		}
		if strings.TrimSpace(followup.Instruction) == "" {
			return TaskMetadata{}, fmt.Errorf("staged followup %d has empty instruction", index+1)
		}
		if followup.Oracle.Build.CleanCompile == nil || followup.Oracle.Test.PassTests == nil {
			return TaskMetadata{}, fmt.Errorf("staged followup %d requires explicit build and test settings", index+1)
		}
		if followup.Oracle.AST.File == "" {
			return TaskMetadata{}, fmt.Errorf("staged followup %d requires an AST file", index+1)
		}
		meta.StagedFollowups = append(meta.StagedFollowups, StagedFollowup{
			TotalEdits:  followup.TotalEdits,
			Instruction: followup.Instruction,
			Oracle: OracleConfig{
				MutationPolicy: followup.Oracle.MutationPolicy,
				AST:            followup.Oracle.AST,
				Build:          BuildConfig{CleanCompile: *followup.Oracle.Build.CleanCompile},
				Test:           TestConfig{PassTests: *followup.Oracle.Test.PassTests, HiddenTestDir: followup.Oracle.Test.HiddenTestDir},
			},
		})
		previous = followup.TotalEdits
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
	res := &OracleResult{}
	if failure := checkMutationPolicy(task.Metadata.Oracle.MutationPolicy, modifiedFiles); failure.failed {
		applyOracleFailure(res, failure, start)
		return res, nil
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
	if err := evaluateAST(task.Metadata.Oracle.AST, evalDir); err != nil {
		applyOracleFailure(res, oracleFailure{failed: true, stage: "level_2_ast", message: err.Error()}, start)
		return res, nil
	}
	res.Level2AST = true
	if err := evaluateOracleBuild(ctx, task, evalDir); err != nil {
		applyOracleFailure(res, oracleFailure{failed: true, stage: "level_3_build", message: err.Error()}, start)
		return res, nil
	}
	res.Level3Build = true
	if err := evaluateOracleTests(ctx, task, evalDir); err != nil {
		applyOracleFailure(res, oracleFailure{failed: true, stage: "level_4_test", message: err.Error()}, start)
		return res, nil
	}
	res.Level4Test = true
	res.Passed = true
	res.Duration = time.Since(start)
	return res, nil
}

type oracleFailure struct {
	failed  bool
	stage   string
	message string
}

func checkMutationPolicy(policy MutationPolicyConfig, modifiedFiles []string) oracleFailure {
	if slices.Contains(policy.DisallowedFiles, "*") && len(modifiedFiles) > 0 {
		return oracleFailure{failed: true, stage: "level_1_mutation_policy", message: fmt.Sprintf("workspace must remain unchanged; modified: %s", strings.Join(modifiedFiles, ", "))}
	}
	for _, modified := range modifiedFiles {
		relative := filepath.ToSlash(modified)
		if slices.Contains(policy.DisallowedFiles, relative) {
			return oracleFailure{failed: true, stage: "level_1_mutation_policy", message: fmt.Sprintf("unauthorized modification of protected file: %s", relative)}
		}
	}
	return oracleFailure{}
}

func applyOracleFailure(result *OracleResult, failure oracleFailure, started time.Time) {
	result.FailureStage = failure.stage
	result.ErrorMessage = failure.message
	result.Duration = time.Since(started)
}

func evaluateOracleBuild(ctx context.Context, task *Task, workDir string) error {
	if !task.Metadata.Oracle.Build.CleanCompile {
		return nil
	}
	if output, err := runOracleGoCommand(ctx, workDir, "build"); err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, output)
	}
	return nil
}

func evaluateOracleTests(ctx context.Context, task *Task, workDir string) error {
	if !task.Metadata.Oracle.Test.PassTests {
		return nil
	}
	if err := installHiddenTests(workDir, task.Metadata.Oracle.Test.HiddenTestDir); err != nil {
		return fmt.Errorf("install hidden tests: %w", err)
	}
	if output, err := runOracleGoCommand(ctx, workDir, "test"); err != nil {
		return fmt.Errorf("go test failed: %w\n%s", err, output)
	}
	return nil
}

func runOracleGoCommand(ctx context.Context, workDir, operation string) ([]byte, error) {
	var command *exec.Cmd
	switch operation {
	case "build":
		command = exec.CommandContext(ctx, "go", "build", "./...")
	case "test":
		command = exec.CommandContext(ctx, "go", "test", "./...")
	default:
		return nil, fmt.Errorf("unsupported oracle Go command %q", operation)
	}
	command.Dir = workDir
	command.Env = append(os.Environ(), "CGO_ENABLED=0")
	return command.CombinedOutput()
}

func copyWorkspaceForOracle(workDir string) (string, func() error, error) {
	sourceRoot, resolveErr := filepath.Abs(workDir)
	if resolveErr != nil {
		return "", nil, fmt.Errorf("resolve workspace: %w", resolveErr)
	}
	sourceFS, sourceOpenErr := os.OpenRoot(sourceRoot)
	if sourceOpenErr != nil {
		return "", nil, fmt.Errorf("open workspace: %w", sourceOpenErr)
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
	target, createTargetErr := os.MkdirTemp(scratchDir, "oracle-*")
	if createTargetErr != nil {
		return "", nil, errors.Join(fmt.Errorf("create private workspace: %w", createTargetErr), sourceFS.Close(), removeScratch())
	}
	targetFS, targetOpenErr := os.OpenRoot(target)
	if targetOpenErr != nil {
		return "", nil, errors.Join(fmt.Errorf("open private workspace: %w", targetOpenErr), sourceFS.Close(), os.RemoveAll(target), removeScratch())
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
	walkErr := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}
		rel, relErr := filepath.Rel(sourceRoot, path)
		if relErr != nil {
			return fmt.Errorf("relativize %s: %w", path, relErr)
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
		data, readErr := sourceFS.ReadFile(rel)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rel, readErr)
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("stat %s: %w", rel, infoErr)
		}
		if err := targetFS.MkdirAll(filepath.Dir(rel), 0o750); err != nil {
			return fmt.Errorf("create parent for %s: %w", rel, err)
		}
		if err := targetFS.WriteFile(rel, data, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		return nil
	})
	if walkErr != nil {
		return "", nil, errors.Join(fmt.Errorf("copy workspace: %w", walkErr), cleanup())
	}
	return target, cleanup, nil
}

// installHiddenTests copies task-owned acceptance tests only after the agent has finished.
func installHiddenTests(workDir, sourceDir string) error {
	if sourceDir == "" {
		return nil
	}
	sourceRoot, rootErr := filepath.Abs(sourceDir)
	if rootErr != nil {
		return fmt.Errorf("resolve hidden test directory: %w", rootErr)
	}
	sourceFS, sourceOpenErr := os.OpenRoot(sourceRoot)
	if sourceOpenErr != nil {
		return fmt.Errorf("open hidden test directory: %w", sourceOpenErr)
	}
	defer func() {
		_ = sourceFS.Close()
	}()

	workFS, workOpenErr := os.OpenRoot(workDir)
	if workOpenErr != nil {
		return fmt.Errorf("open benchmark workspace: %w", workOpenErr)
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
		rel, relErr := filepath.Rel(sourceRoot, path)
		if relErr != nil {
			return fmt.Errorf("relativize hidden test %s: %w", path, relErr)
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

	declaredSymbols := collectDeclaredSymbols(fileNode)

	for _, sym := range cfg.MustContainSymbols {
		if _, found := declaredSymbols[sym]; !found {
			if !astContainsIdent(fileNode, sym) {
				return fmt.Errorf("required symbol %q not found in %s", sym, cfg.File)
			}
		}
	}

	for _, sym := range cfg.MustNotContainSymbols {
		if _, found := declaredSymbols[sym]; found {
			return fmt.Errorf("disallowed symbol %q remains declared in %s", sym, cfg.File)
		}
		if astContainsExactIdent(fileNode, sym) {
			return fmt.Errorf("disallowed symbol reference %q persists in %s", sym, cfg.File)
		}
	}

	if err := validateRequiredCalls(cfg, fileNode); err != nil {
		return err
	}

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

func collectDeclaredSymbols(fileNode *ast.File) map[string]int {
	declaredSymbols := make(map[string]int)
	for index, declaration := range fileNode.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			collectFunctionSymbol(declaredSymbols, typed, index)
		case *ast.GenDecl:
			collectGeneratedSymbols(declaredSymbols, typed, index)
		}
	}
	return declaredSymbols
}

func collectFunctionSymbol(symbols map[string]int, function *ast.FuncDecl, declarationIndex int) {
	symbol := function.Name.Name
	if function.Recv != nil && len(function.Recv.List) > 0 {
		if receiver := receiverTypeName(function.Recv.List[0].Type); receiver != "" {
			symbol = receiver + "." + function.Name.Name
		}
	}
	symbols[symbol] = declarationIndex
}

func collectGeneratedSymbols(symbols map[string]int, declaration *ast.GenDecl, declarationIndex int) {
	for _, specification := range declaration.Specs {
		switch typed := specification.(type) {
		case *ast.TypeSpec:
			collectTypeSymbols(symbols, typed, declarationIndex)
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				symbols[name.Name] = declarationIndex
			}
		}
	}
}

func collectTypeSymbols(symbols map[string]int, specification *ast.TypeSpec, declarationIndex int) {
	symbols[specification.Name.Name] = declarationIndex
	interfaceType, ok := specification.Type.(*ast.InterfaceType)
	if !ok || interfaceType.Methods == nil {
		return
	}
	for _, method := range interfaceType.Methods.List {
		for _, name := range method.Names {
			symbols[specification.Name.Name+"."+name.Name] = declarationIndex
		}
	}
}

func validateRequiredCalls(cfg ASTConfig, fileNode *ast.File) error {
	for _, requirement := range cfg.MustCallFunctions {
		functionName, calleeName, ok := strings.Cut(requirement, "->")
		if !ok || functionName == "" || calleeName == "" {
			return fmt.Errorf("invalid function-call assertion %q (want Function->callee)", requirement)
		}
		found := false
		for _, decl := range fileNode.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Name.Name != functionName || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee, ok := call.Fun.(*ast.Ident)
				if ok && callee.Name == calleeName {
					found = true
				}
				return true
			})
		}
		if !found {
			return fmt.Errorf("required call %s -> %s not found in %s", functionName, calleeName, cfg.File)
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
