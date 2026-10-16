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

	"github.com/spockz/semantic-editor/internal/atomicfile"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/telemetry"

	"github.com/spockz/semantic-editor/internal/sourcebuf"

	"golang.org/x/tools/imports"
)

// ImportOutcomeError retains the known source-view changes when a later import step fails.
type ImportOutcomeError struct {
	Report ImportReport
	Err    error
}

func (e *ImportOutcomeError) Error() string {
	return e.Err.Error()
}

func (e *ImportOutcomeError) Unwrap() error {
	return e.Err
}

// ImportReport records canonical paths changed in the active source view.
type ImportReport struct {
	ChangedFiles []string
	Buffered     bool
}

var (
	// ErrAtomicWrite indicates a failure during atomic file staging, sync, or rename.
	ErrAtomicWrite = atomicfile.ErrAtomicWrite
	// ErrFormatFailed indicates gofmt execution failed.
	ErrFormatFailed = errors.New("gofmt execution failed")
	// ErrOrganizeImportsFailed indicates import organization failed.
	ErrOrganizeImportsFailed = errors.New("organize imports failed")

	reMissingPackage = regexp.MustCompile(`cannot find package "([^"]+)"`)
	reNoModule       = regexp.MustCompile(`no required module provides package ([^;:\s]+)`)
)

// WriteAtomic writes data to a file atomically via sibling temp file, fsync, and rename.
func WriteAtomic(targetPath string, data []byte) error {
	return atomicfile.WriteAtomic(targetPath, data)
}

// Format runs gofmt on the specified paths or directories.
func Format(ctx context.Context, workDir string, paths ...string) error {
	if sourcebuf.FromContext(ctx) != nil {
		return formatBuffered(ctx, workDir, paths...)
	}

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
	Before      []string `json:"-"`
	After       []string `json:"-"`
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
	_, err := OrganizeImportsWithOptionsReport(ctx, workDir, opts, paths...)
	return err
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

	removeExplicitImports(fileNode, opts.Remove)
	addExplicitImports(fileNode, opts.Add)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, fileNode); err != nil {
		return nil, fmt.Errorf("format explicit import edits: %w", err)
	}
	return buf.Bytes(), nil
}

func removeExplicitImports(fileNode *ast.File, paths []string) {
	if len(paths) == 0 {
		return
	}
	newDecls := make([]ast.Decl, 0, len(fileNode.Decls))
	for _, decl := range fileNode.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			newDecls = append(newDecls, decl)
			continue
		}
		gen.Specs = filterImportSpecs(gen.Specs, paths)
		if len(gen.Specs) > 0 {
			newDecls = append(newDecls, gen)
		}
	}
	fileNode.Decls = newDecls
}

func filterImportSpecs(specs []ast.Spec, paths []string) []ast.Spec {
	kept := make([]ast.Spec, 0, len(specs))
	for _, spec := range specs {
		imp, ok := spec.(*ast.ImportSpec)
		if !ok || !slices.Contains(paths, strings.Trim(imp.Path.Value, `"'`)) {
			kept = append(kept, spec)
		}
	}
	return kept
}

func addExplicitImports(fileNode *ast.File, entries []string) {
	for _, entry := range entries {
		alias, pkgPath := parseImportEntry(entry)
		if pkgPath == "" || updateExistingImport(fileNode, alias, pkgPath) {
			continue
		}
		insertExplicitImport(fileNode, alias, pkgPath)
	}
}

func updateExistingImport(fileNode *ast.File, alias, pkgPath string) bool {
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
			return true
		}
	}
	return false
}

func insertExplicitImport(fileNode *ast.File, alias, pkgPath string) {
	newSpec := &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(pkgPath)}}
	if alias != "" {
		newSpec.Name = ast.NewIdent(alias)
	}
	for _, decl := range fileNode.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			gen.Specs = append(gen.Specs, newSpec)
			return
		}
	}
	anchor := fileNode.Name.End()
	newSpec.Path.ValuePos = anchor
	if newSpec.Name != nil {
		newSpec.Name.NamePos = anchor
	}
	newDecl := &ast.GenDecl{TokPos: anchor, Tok: token.IMPORT, Specs: []ast.Spec{newSpec}}
	fileNode.Decls = slices.Insert(fileNode.Decls, 0, ast.Decl(newDecl))
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

// CheckDiagnostics requests root-wide gopls diagnostics under the selected project root.
func CheckDiagnostics(ctx context.Context, workDir string) ([]string, error) {
	diagnostics, err := CheckDiagnosticDetails(ctx, workDir)
	return godiagnostics.Messages(diagnostics), err
}

// CheckDiagnosticDetails returns root-wide typed gopls diagnostics under the selected project root.
func CheckDiagnosticDetails(ctx context.Context, workDir string) ([]godiagnostics.Diagnostic, error) {
	return CheckDiagnosticDetailsForPatterns(godiagnostics.WithIncrementalDiagnostics(ctx), workDir)
}

// CheckDiagnosticDetailsForPatterns returns diagnostics for explicit Go package patterns; no patterns preserves workspace-wide behavior.
func CheckDiagnosticDetailsForPatterns(ctx context.Context, workDir string, patterns ...string) ([]godiagnostics.Diagnostic, error) {
	finish := telemetry.Start(ctx, telemetry.PhaseVerificationDiagnostics)
	defer finish()

	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	effectiveDir, err := filepath.Abs(filepath.Clean(workDir))
	if err != nil {
		return nil, fmt.Errorf("resolve Go diagnostics root %q: %w", workDir, err)
	}
	goplsPath, err := godiagnostics.FindGopls()
	if err != nil {
		return nil, fmt.Errorf("find gopls for Go diagnostics: %w", err)
	}
	environment, err := gocache.Environment(ctx, effectiveDir)
	if err != nil {
		return nil, fmt.Errorf("prepare gopls diagnostics environment: %w", err)
	}
	diagnostics, err := godiagnostics.CheckDetailedWithPatternsWithContext(ctx, effectiveDir, goplsPath, environment, patterns)
	if err != nil {
		return nil, fmt.Errorf("check Go diagnostics with gopls: %w", err)
	}
	return diagnostics, nil
}

// GoFiles selects source files using shared formatting and verification exclusions. Explicit file symlinks are honored; recursive discovery skips all symlinks.
func GoFiles(workDir string, paths ...string) ([]string, error) {
	return GoFilesContext(context.Background(), workDir, paths...)
}

// GoFilesContext selects source files from the active disk or buffered source view.
func GoFilesContext(ctx context.Context, workDir string, paths ...string) ([]string, error) {
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
	cacheRoots, err := configuredGoCacheRoots(ctx, base)
	if err != nil {
		return nil, err
	}

	selected := make(map[string]string)
	for _, source := range paths {
		if err := collectGoPath(ctx, base, source, cacheRoots, selected); err != nil {
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

func collectGoPath(ctx context.Context, base, source string, cacheRoots []string, selected map[string]string) error {
	if source == "" {
		return fmt.Errorf("empty Go source path")
	}
	target := source
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, target)
	}
	var err error
	target, err = filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve Go source path %q: %w", source, err)
	}
	info, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			canonical, canonicalErr := sourcebuf.CanonicalPath(ctx, target)
			if canonicalErr == nil {
				if filepath.Ext(canonical) == ".go" {
					addGoFile(selected, canonical, source)
				}
				return nil
			}
		}
		return fmt.Errorf("stat Go source path %q: %w", source, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return collectExplicitSymlink(selected, target, source)
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() {
			addGoFile(selected, target, source)
		}
		return nil
	}
	walkCacheRoots, err := cacheRootsForGoDirectory(target, cacheRoots)
	if err != nil {
		return err
	}
	if err := sourcebuf.ObserveTree(ctx, target, ".go"); err != nil {
		return fmt.Errorf("observe selected Go sources: %w", err)
	}
	return collectGoDirectory(base, target, source, walkCacheRoots, selected)
}

func collectExplicitSymlink(selected map[string]string, target, source string) error {
	targetInfo, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("stat Go source symlink %q: %w", source, err)
	}
	if !targetInfo.IsDir() && targetInfo.Mode().IsRegular() {
		addGoFile(selected, target, source)
	}
	return nil
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

// WriteAtomicContext routes source writes through the active batch buffer.
func WriteAtomicContext(ctx context.Context, targetPath string, data []byte) error {
	return sourcebuf.WriteFile(ctx, targetPath, data)
}

// OrganizeImportsWithOptionsReport applies import changes and reports paths written in the current workspace view.
func OrganizeImportsWithOptionsReport(ctx context.Context, workDir string, opts ImportOptions, paths ...string) (ImportReport, error) {
	finish := func(report ImportReport, err error) (ImportReport, error) {
		slices.Sort(report.ChangedFiles)
		report.ChangedFiles = slices.Compact(report.ChangedFiles)
		if err != nil {
			err = &ImportOutcomeError{Report: report, Err: err}
		}
		return report, err
	}
	report := ImportReport{Buffered: sourcebuf.FromContext(ctx) != nil}
	finishTelemetry := telemetry.Start(ctx, telemetry.PhaseFormattingImports)
	defer finishTelemetry()

	goFiles, err := selectImportFiles(ctx, workDir, paths)
	if err != nil {
		return finish(report, err)
	}

	for _, file := range goFiles {
		changed, err := organizeImportFile(ctx, file, opts)
		if err != nil {
			return finish(report, err)
		}
		if changed != "" {
			report.ChangedFiles = append(report.ChangedFiles, changed)
		}
	}
	return finish(report, nil)
}

func selectImportFiles(ctx context.Context, workDir string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	files, err := GoFilesContext(ctx, workDir, paths...)
	if err != nil {
		return nil, err
	}
	root := workDir
	if root == "" {
		root = "."
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve import workspace %q: %w", workDir, err)
	}
	for index, file := range files {
		if !filepath.IsAbs(file) {
			files[index] = filepath.Join(root, file)
		}
	}
	return files, nil
}

func organizeImportFile(ctx context.Context, file string, opts ImportOptions) (string, error) {
	cleanFile := filepath.Clean(file)
	data, err := sourcebuf.ReadFile(ctx, cleanFile)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", file, err)
	}
	modifiedData, err := applyExplicitImports(cleanFile, data, opts)
	if err != nil {
		return "", fmt.Errorf("apply explicit imports for %s: %w", file, err)
	}
	result, err := imports.Process(cleanFile, modifiedData, nil)
	if err != nil {
		return "", fmt.Errorf("%w for %s: %w", ErrOrganizeImportsFailed, file, err)
	}
	if result, err = applyExplicitImports(cleanFile, result, ImportOptions{Add: opts.Add}); err != nil {
		return "", fmt.Errorf("preserve explicit imports for %s: %w", file, err)
	}
	if bytes.Equal(data, result) {
		return "", nil
	}
	return publishOrganizedImports(ctx, file, cleanFile, result)
}

func publishOrganizedImports(ctx context.Context, file, cleanFile string, result []byte) (string, error) {
	canonical, err := filepath.Abs(cleanFile)
	if err != nil {
		return "", fmt.Errorf("resolve organized file path %s: %w", cleanFile, err)
	}
	if err := WriteAtomicContext(ctx, cleanFile, result); err != nil {
		return "", fmt.Errorf("write organized file %s: %w", file, err)
	}
	return canonical, nil
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

func configuredGoCacheRoots(ctx context.Context, base string) ([]string, error) {
	filter, err := sourcebuf.DirectoryFilterFor(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("resolve Go source-selection environment: %w", err)
	}
	return filter.GeneratedRoots(), nil
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

func formatBuffered(ctx context.Context, workDir string, paths ...string) error {
	files, err := GoFilesContext(ctx, workDir, paths...)
	if err != nil {
		return fmt.Errorf("%w: select buffered Go files: %w", ErrFormatFailed, err)
	}
	for _, file := range files {
		target := file
		if !filepath.IsAbs(target) {
			target = filepath.Join(workDir, target)
		}
		before, err := sourcebuf.ReadFile(ctx, target)
		if err != nil {
			return fmt.Errorf("%w: read buffered source: %w", ErrFormatFailed, err)
		}
		formatted, err := format.Source(before)
		if err != nil {
			return fmt.Errorf("%w: format buffered source: %w", ErrFormatFailed, err)
		}
		if bytes.Equal(before, formatted) {
			continue
		}
		if err := WriteAtomicContext(ctx, target, formatted); err != nil {
			return fmt.Errorf("%w: buffer formatted source: %w", ErrFormatFailed, err)
		}
	}
	return nil
}
