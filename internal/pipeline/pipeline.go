// Package pipeline orchestrates atomic writes, code formatting, and compiler diagnostic reporting.
package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"semedit/internal/gocache"
	"semedit/internal/telemetry"

	"golang.org/x/tools/imports"
)

var (
	// ErrAtomicWrite indicates a failure during atomic file staging, sync, or rename.
	ErrAtomicWrite = errors.New("atomic write failed")
	// ErrFormatFailed indicates gofmt execution failed.
	ErrFormatFailed = errors.New("gofmt execution failed")
	// ErrOrganizeImportsFailed indicates import organization failed.
	ErrOrganizeImportsFailed = errors.New("organize imports failed")

	reMissingPackage = regexp.MustCompile(`cannot find package "([^"]+)"`)
	reNoModule       = regexp.MustCompile(`no required module provides package ([^;:\s]+)`)
)

// WriteAtomic writes data to a file atomically via sibling temp file, fsync, and rename.
func WriteAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	var oldInfo os.FileInfo
	if info, statErr := os.Stat(targetPath); statErr == nil {
		oldInfo = info
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("%w: stat target: %w", ErrAtomicWrite, statErr)
	}
	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("%w: create temp file: %w", ErrAtomicWrite, err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("%w: write temp file: %w", ErrAtomicWrite, err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("%w: sync temp file: %w", ErrAtomicWrite, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("%w: close temp file: %w", ErrAtomicWrite, err)
	}

	if oldInfo != nil {
		if err := os.Chmod(tmpName, oldInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("%w: preserve target permissions: %w", ErrAtomicWrite, err)
		}
		// Ensure advancing mtime before renaming (ADR-0010).
		newTime := time.Now()
		if minimum := oldInfo.ModTime().Add(time.Millisecond); newTime.Before(minimum) {
			newTime = minimum
		}
		if err := os.Chtimes(tmpName, newTime, newTime); err != nil {
			return fmt.Errorf("%w: set target mtime: %w", ErrAtomicWrite, err)
		}
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("%w: rename to target: %w", ErrAtomicWrite, err)
	}

	return nil
}

// Format runs gofmt on the specified paths or directories.
func Format(ctx context.Context, workDir string, paths ...string) error {
	finish := telemetry.Start(ctx, telemetry.PhaseFormattingGofmt)
	defer finish()

	files, err := GoFiles(workDir, paths...)
	if err != nil {
		return fmt.Errorf("%w: select Go files: %w", ErrFormatFailed, err)
	}
	for _, file := range files {
		formatted, err := runGofmt(ctx, workDir, "", []string{file})
		if err != nil {
			return err
		}
		target := file
		if !filepath.IsAbs(target) && workDir != "" {
			target = filepath.Join(workDir, target)
		}
		target, err = filepath.EvalSymlinks(target)
		if err != nil {
			return fmt.Errorf("%w: resolve formatted Go file %q: %w", ErrFormatFailed, file, err)
		}
		before, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("%w: read formatted Go file %q: %w", ErrFormatFailed, file, err)
		}
		if bytes.Equal(before, []byte(formatted)) {
			continue
		}
		if err := WriteAtomic(target, []byte(formatted)); err != nil {
			return fmt.Errorf("%w: write formatted Go file %q: %w", ErrFormatFailed, file, err)
		}
	}
	return nil
}

// DiagnosticDelta records compiler diagnostic shifts across an edit (ADR-0004, RQ-0006).
type DiagnosticDelta struct {
	Before      []string `json:"before"`
	After       []string `json:"after"`
	NetDelta    int      `json:"net_delta"`
	Introduced  []string `json:"introduced"`
	Resolved    []string `json:"resolved"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// ComputeDelta calculates introduced and resolved diagnostics between two states.
func ComputeDelta(before []string, after []string) DiagnosticDelta {
	beforeMap := make(map[string]bool, len(before))
	for _, b := range before {
		beforeMap[b] = true
	}

	afterMap := make(map[string]bool, len(after))
	for _, a := range after {
		afterMap[a] = true
	}

	var introduced []string
	for _, a := range after {
		if !beforeMap[a] {
			introduced = append(introduced, a)
		}
	}

	var resolved []string
	for _, b := range before {
		if !afterMap[b] {
			resolved = append(resolved, b)
		}
	}

	var suggestions []string
	seenSuggestion := make(map[string]bool)
	for _, intro := range introduced {
		if m := reMissingPackage.FindStringSubmatch(intro); len(m) > 1 {
			pkg := m[1]
			sug := fmt.Sprintf("Run 'go get %s' or use semantic_add_build_dependency to install the missing dependency.", pkg)
			if !seenSuggestion[sug] {
				suggestions = append(suggestions, sug)
				seenSuggestion[sug] = true
			}
		} else if m := reNoModule.FindStringSubmatch(intro); len(m) > 1 {
			pkg := m[1]
			sug := fmt.Sprintf("Run 'go get %s' or use semantic_add_build_dependency to install the missing dependency.", pkg)
			if !seenSuggestion[sug] {
				suggestions = append(suggestions, sug)
				seenSuggestion[sug] = true
			}
		}
	}

	return DiagnosticDelta{
		Before:      before,
		After:       after,
		NetDelta:    len(after) - len(before),
		Introduced:  introduced,
		Resolved:    resolved,
		Suggestions: suggestions,
	}
}

// ImportOptions configures explicit import additions and removals.
type ImportOptions struct {
	Add    []string
	Remove []string
}

// OrganizeImports adjusts imports and formats the given paths using golang.org/x/tools/imports.
func OrganizeImports(ctx context.Context, workDir string, paths ...string) error {
	return OrganizeImportsWithOptions(ctx, workDir, ImportOptions{}, paths...)
}

// OrganizeImportsWithOptions adjusts imports and formats using explicit additions/removals and imports.Process.
func OrganizeImportsWithOptions(ctx context.Context, workDir string, opts ImportOptions, paths ...string) error {
	finish := telemetry.Start(ctx, telemetry.PhaseFormattingImports)
	defer finish()

	if len(paths) == 0 {
		paths = []string{"."}
	}

	var goFiles []string
	for _, p := range paths {
		target := p
		if workDir != "" && !filepath.IsAbs(target) {
			target = filepath.Join(workDir, target)
		}
		fi, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("stat path %s: %w", p, err)
		}
		if fi.IsDir() {
			err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					name := info.Name()
					if name == ".git" || name == ".scratch" || name == "vendor" || name == "node_modules" {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasSuffix(info.Name(), ".go") {
					goFiles = append(goFiles, path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("walk dir %s: %w", target, err)
			}
		} else if strings.HasSuffix(target, ".go") {
			goFiles = append(goFiles, target)
		}
	}

	for _, file := range goFiles {
		cleanFile := filepath.Clean(file)
		data, err := os.ReadFile(cleanFile)
		if err != nil {
			return fmt.Errorf("read file %s: %w", file, err)
		}

		modifiedData, err := applyExplicitImports(cleanFile, data, opts)
		if err != nil {
			return fmt.Errorf("apply explicit imports for %s: %w", file, err)
		}

		res, err := imports.Process(cleanFile, modifiedData, nil)
		if err != nil {
			return fmt.Errorf("%w for %s: %w", ErrOrganizeImportsFailed, file, err)
		}

		if !bytes.Equal(data, res) {
			if err := WriteAtomic(cleanFile, res); err != nil {
				return fmt.Errorf("write organized file %s: %w", file, err)
			}
		}
	}
	return nil
}

func parseImportEntry(entry string) (alias string, path string) {
	trimmed := strings.TrimSpace(entry)
	if idx := strings.Index(trimmed, " "); idx > 0 {
		alias = strings.TrimSpace(trimmed[:idx])
		path = strings.Trim(strings.TrimSpace(trimmed[idx+1:]), `"'`)
		return alias, path
	}
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		alias = strings.TrimSpace(trimmed[:idx])
		path = strings.Trim(strings.TrimSpace(trimmed[idx+1:]), `"'`)
		return alias, path
	}
	return "", strings.Trim(trimmed, `"'`)
}

func applyExplicitImports(filePath string, src []byte, opts ImportOptions) ([]byte, error) {
	if len(opts.Add) == 0 && len(opts.Remove) == 0 {
		return src, nil
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse source for import edits: %w", err)
	}

	if len(opts.Remove) > 0 {
		var newDecls []ast.Decl
		for _, decl := range fileNode.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.IMPORT {
				newDecls = append(newDecls, decl)
				continue
			}

			var newSpecs []ast.Spec
			for _, spec := range gen.Specs {
				imp, ok := spec.(*ast.ImportSpec)
				if !ok {
					newSpecs = append(newSpecs, spec)
					continue
				}
				impPath := strings.Trim(imp.Path.Value, `"'`)
				if slices.Contains(opts.Remove, impPath) {
					continue
				}
				newSpecs = append(newSpecs, spec)
			}
			gen.Specs = newSpecs
			if len(gen.Specs) > 0 {
				newDecls = append(newDecls, gen)
			}
		}
		fileNode.Decls = newDecls
	}

	if len(opts.Add) > 0 {
		for _, entry := range opts.Add {
			alias, pkgPath := parseImportEntry(entry)
			if pkgPath == "" {
				continue
			}
			found := false
			for _, decl := range fileNode.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.IMPORT {
					continue
				}
				for _, spec := range gen.Specs {
					imp, ok := spec.(*ast.ImportSpec)
					if !ok || strings.Trim(imp.Path.Value, `"'`) != pkgPath {
						continue
					}
					if alias != "" {
						imp.Name = ast.NewIdent(alias)
					}
					found = true
					break
				}
				if found {
					break
				}
			}
			if found {
				continue
			}

			newSpec := &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(pkgPath),
				},
			}
			if alias != "" {
				newSpec.Name = ast.NewIdent(alias)
			}

			var firstImportDecl *ast.GenDecl
			for _, decl := range fileNode.Decls {
				if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
					firstImportDecl = gen
					break
				}
			}

			if firstImportDecl != nil {
				firstImportDecl.Specs = append(firstImportDecl.Specs, newSpec)
			} else {
				newGen := &ast.GenDecl{
					Tok:   token.IMPORT,
					Specs: []ast.Spec{newSpec},
				}
				fileNode.Decls = slices.Insert(fileNode.Decls, 0, ast.Decl(newGen))
			}
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, fileNode); err != nil {
		return nil, fmt.Errorf("format explicit import edits: %w", err)
	}
	return buf.Bytes(), nil
}

// FindModuleRoot locates the nearest enclosing directory containing go.mod.
func FindModuleRoot(dir string) string {
	cur := filepath.Clean(dir)
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return dir
}

// CheckDiagnostics collects compiler/linter diagnostics without rolling back intermediate states (ADR-0004).
func CheckDiagnostics(ctx context.Context, workDir string) ([]string, error) {
	finish := telemetry.Start(ctx, telemetry.PhaseVerificationDiagnostics)
	defer finish()

	effectiveDir := FindModuleRoot(workDir)

	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	if effectiveDir != "" {
		cmd.Dir = effectiveDir
	}
	var err error
	cmd.Env, err = gocache.Environment(ctx, effectiveDir)
	if err != nil {
		return nil, fmt.Errorf("prepare go diagnostics environment: %w", err)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	_ = cmd.Run() // Exit status may be non-zero on compiler errors, which is normal for diagnostic reporting.

	output := out.String()
	if output == "" {
		return nil, nil
	}

	var diagnostics []string
	for line := range bytes.SplitSeq([]byte(output), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			diagnostics = append(diagnostics, string(trimmed))
		}
	}
	return diagnostics, nil
}

// GoFiles selects source files using shared formatting and verification exclusions. Explicit file symlinks are honored; recursive discovery skips all symlinks.
func GoFiles(workDir string, paths ...string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	base := workDir
	if base == "" {
		base = "."
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return nil, fmt.Errorf("resolve Go workspace %q: %w", workDir, err)
	}
	cacheRoots, err := configuredGoCacheRoots(base)
	if err != nil {
		return nil, err
	}

	selected := make(map[string]string)
	for _, source := range paths {
		if source == "" {
			return nil, fmt.Errorf("empty Go source path")
		}
		target := source
		if !filepath.IsAbs(target) {
			target = filepath.Join(base, target)
		}
		target, err = filepath.Abs(target)
		if err != nil {
			return nil, fmt.Errorf("resolve Go source path %q: %w", source, err)
		}
		info, err := os.Lstat(target)
		if err != nil {
			return nil, fmt.Errorf("stat Go source path %q: %w", source, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			targetInfo, err := os.Stat(target)
			if err != nil {
				return nil, fmt.Errorf("stat Go source symlink %q: %w", source, err)
			}
			if targetInfo.IsDir() {
				continue
			}
			if targetInfo.Mode().IsRegular() {
				addGoFile(selected, target, source)
			}
			continue
		}
		if !info.IsDir() {
			if info.Mode().IsRegular() {
				addGoFile(selected, target, source)
			}
			continue
		}
		walkCacheRoots, err := cacheRootsForGoDirectory(target, cacheRoots)
		if err != nil {
			return nil, err
		}
		if err := collectGoDirectory(base, target, source, walkCacheRoots, selected); err != nil {
			return nil, err
		}
	}

	files := make([]string, 0, len(selected))
	for _, path := range selected {
		files = append(files, path)
	}
	slices.Sort(files)
	return files, nil
}

// NeedsFormatting reports whether selected Go files differ from gofmt output.
func NeedsFormatting(ctx context.Context, workDir string, paths ...string) (bool, error) {
	files, err := GoFiles(workDir, paths...)
	if err != nil {
		return false, err
	}
	if len(files) == 0 {
		return false, nil
	}
	output, err := runGofmt(ctx, workDir, "-l", files)
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace([]byte(output))) > 0, nil
}

// runGofmt invokes gofmt in bounded argument batches and never starts with no file arguments.
func runGofmt(ctx context.Context, workDir, flag string, files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	const maxArgumentBytes = 16 * 1024
	var output strings.Builder
	for start := 0; start < len(files); {
		end := start
		argumentBytes := len(flag) + len("--") + 2
		for end < len(files) {
			nextBytes := argumentBytes + len(files[end]) + 1
			if nextBytes > maxArgumentBytes {
				break
			}
			argumentBytes = nextBytes
			end++
		}
		if end == start {
			return "", fmt.Errorf("%w: gofmt file path exceeds safe argument size", ErrFormatFailed)
		}

		args := make([]string, 0, end-start+2)
		if flag != "" {
			args = append(args, flag)
		}
		args = append(args, "--")
		args = append(args, files[start:end]...)
		// #nosec G204 -- fixed gofmt invocation with validated, selected source paths.
		cmd := exec.CommandContext(ctx, "gofmt", args...)
		if workDir != "" {
			cmd.Dir = workDir
		}
		cmd.Env = os.Environ()
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("%w: %s: %w", ErrFormatFailed, strings.TrimSpace(stderr.String()), err)
		}
		output.Write(stdout.Bytes())
		start = end
	}
	return output.String(), nil
}

func configuredGoCacheRoots(base string) ([]string, error) {
	roots := make([]string, 0, 8)
	add := func(name, path string) error {
		if path == "" {
			return nil
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		path, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve Go cache path %q: %w", path, err)
		}
		roots = append(roots, filepath.Clean(path))
		canonical, err := filepath.EvalSymlinks(path)
		if err == nil {
			roots = append(roots, filepath.Clean(canonical))
			return nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("resolve configured Go cache %s %q: %w", name, path, err)
	}

	for _, key := range []string{"GOMODCACHE", "GOCACHE", "GOTMPDIR"} {
		if err := add(key, os.Getenv(key)); err != nil {
			return nil, err
		}
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve default Go path: %w", err)
		}
		gopath = filepath.Join(home, "go")
	}
	for _, path := range filepath.SplitList(gopath) {
		if err := add("GOPATH module cache", filepath.Join(path, "pkg", "mod")); err != nil {
			return nil, err
		}
	}
	return roots, nil
}

func goPathWithin(path, parent string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func isConfiguredGoCache(path string, roots []string) (bool, error) {
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false, fmt.Errorf("resolve traversed Go directory %q: %w", path, err)
	}
	for _, root := range roots {
		if goPathWithin(path, root) || goPathWithin(canonical, root) {
			return true, nil
		}
	}
	return false, nil
}

func skipGoDirectory(path string, entry os.DirEntry, cacheRoots []string) (bool, error) {
	switch entry.Name() {
	case ".scratch", ".git", "vendor":
		return true, nil
	}
	return isConfiguredGoCache(path, cacheRoots)
}

func addGoFile(selected map[string]string, path, argument string) {
	clean := filepath.Clean(path)
	if previous, ok := selected[clean]; !ok || argument < previous {
		selected[clean] = argument
	}
}

func collectGoDirectory(base, target, source string, cacheRoots []string, selected map[string]string) error {
	return filepath.WalkDir(target, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk Go source path %q: %w", source, walkErr)
		}
		if current != target && entry.IsDir() {
			skip, err := skipGoDirectory(current, entry, cacheRoots)
			if err != nil {
				return err
			}
			if skip {
				return filepath.SkipDir
			}
		}
		if entry.IsDir() || !entry.Type().IsRegular() || filepath.Ext(current) != ".go" {
			return nil
		}
		argument := current
		if !filepath.IsAbs(source) {
			var err error
			argument, err = filepath.Rel(base, current)
			if err != nil {
				return fmt.Errorf("make Go source path relative to workspace: %w", err)
			}
		}
		addGoFile(selected, current, argument)
		return nil
	})
}

func cacheRootsForGoDirectory(target string, roots []string) ([]string, error) {
	canonical, err := filepath.EvalSymlinks(target)
	if err != nil {
		return nil, fmt.Errorf("resolve selected Go directory %q: %w", target, err)
	}
	nested := make([]string, 0, len(roots))
	for _, root := range roots {
		if goPathWithin(target, root) || goPathWithin(canonical, root) {
			continue
		}
		nested = append(nested, root)
	}
	return nested, nil
}
