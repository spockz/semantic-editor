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
	args := append([]string{"-w"}, paths...)
	// #nosec G204 -- canonical formatter invocation
	cmd := exec.CommandContext(ctx, "gofmt", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = os.Environ()

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrFormatFailed, strings.TrimSpace(errBuf.String()), err)
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
func OrganizeImportsWithOptions(_ context.Context, workDir string, opts ImportOptions, paths ...string) error {
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
		return src, nil
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
			if len(newSpecs) > 0 {
				gen.Specs = newSpecs
				newDecls = append(newDecls, gen)
			}
		}
		fileNode.Decls = newDecls
	}

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
				if !ok {
					continue
				}
				if strings.Trim(imp.Path.Value, `"'`) == pkgPath {
					found = true
					if alias != "" {
						imp.Name = ast.NewIdent(alias)
					}
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
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
		return src, nil
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
	effectiveDir := FindModuleRoot(workDir)

	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	if effectiveDir != "" {
		cmd.Dir = effectiveDir
	}
	var err error
	cmd.Env, err = gocache.Environment(effectiveDir)
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
