// WHY: Applies proven constant extraction plans to an existing Go source overlay.

package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"

	"semedit/internal/gocache"

	"strings"

	"golang.org/x/tools/go/packages"
)

// ExtractConstantOptions selects one exact compile-time expression.
type ExtractConstantOptions struct {
	Discriminator    string
	ExpectedRevision string
}

// ExtractConstant plans and applies one package-level constant extraction.
func ExtractConstant(ctx context.Context, filePath, owner, name string, opts ExtractConstantOptions) (string, error) {
	if opts.Discriminator == "" {
		return "", fmt.Errorf("discriminator must contain the exact source text of one constant expression")
	}
	cleanPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("resolve constant extraction target: %w", err)
	}
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read constant extraction target: %w", err)
	}
	if opts.ExpectedRevision != "" && goReadRevision(content) != opts.ExpectedRevision {
		return "", fmt.Errorf("selected source revision mismatch: expected %q, current %q", opts.ExpectedRevision, goReadRevision(content))
	}
	if err := sourcebuf.ObserveTree(ctx, filepath.Dir(cleanPath)); err != nil {
		return "", fmt.Errorf("capture package source set for constant extraction: %w", err)
	}
	overlay := sourcebuf.Overlay(ctx)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[cleanPath] = bytes.Clone(content)
	loaded, err := loadConstantPackage(ctx, filepath.Dir(cleanPath), overlay)
	if err != nil {
		return "", err
	}
	var fset *token.FileSet
	var syntax *ast.File
	var info *types.Info
	var scope *types.Scope
	for _, pkg := range loaded {
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for index, path := range pkg.CompiledGoFiles {
			absolute, absErr := filepath.Abs(path)
			if absErr != nil || filepath.Clean(absolute) != cleanPath {
				continue
			}
			fset, syntax, info, scope = pkg.Fset, pkg.Syntax[index], pkg.TypesInfo, pkg.Types.Scope()
			break
		}
	}
	if syntax == nil {
		return "", fmt.Errorf("constant extraction target %q is not in a buildable Go package", cleanPath)
	}
	receiver, ownerName, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return "", fmt.Errorf("parse in_function selector: %w", err)
	}
	target := findExtractOwner(syntax, ownerName, receiver)
	if target == nil || target.Body == nil {
		return "", fmt.Errorf("constant extraction owner %q was not found or has no body", owner)
	}
	selected := make([]ast.Expr, 0, 1)
	ast.Inspect(target.Body, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		start, end := fset.Position(expr.Pos()).Offset, fset.Position(expr.End()).Offset
		if start >= 0 && end >= start && end <= len(content) && string(content[start:end]) == opts.Discriminator {
			selected = append(selected, expr)
		}
		return true
	})
	if len(selected) != 1 {
		return "", fmt.Errorf("constant expression selection matched %d current expressions; choose exact source text that identifies one expression", len(selected))
	}
	plan, err := PlanConstantExtraction(content, fset, syntax, info, scope, selected[0], name)
	if err != nil {
		return "", fmt.Errorf("plan constant extraction: %w", err)
	}
	updated := make([]byte, 0, len(content)+len(plan.Declaration)+8)
	updated = append(updated, content[:plan.Start]...)
	updated = append(updated, plan.Replacement...)
	updated = append(updated, content[plan.End:]...)
	if len(updated) > 0 && updated[len(updated)-1] != '\n' {
		updated = append(updated, '\n')
	}
	updated = append(updated, '\n')
	updated = append(updated, []byte(plan.Declaration)...)
	updated = append(updated, '\n')
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, updated, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: plan.Declaration, Cause: err, Err: ErrSyntax}
	}
	validated := sourcebuf.Overlay(ctx)
	if validated == nil {
		validated = make(map[string][]byte)
	}
	validated[cleanPath] = updated
	if _, err := loadConstantPackage(ctx, filepath.Dir(cleanPath), validated); err != nil {
		return "", fmt.Errorf("validate extracted constant source: %w", err)
	}
	checkPreimage := func() error {
		current, readErr := sourcebuf.ReadFile(ctx, cleanPath)
		if readErr != nil {
			return fmt.Errorf("re-read selected source: %w", readErr)
		}
		if !bytes.Equal(current, content) {
			return fmt.Errorf("selected source changed while planning constant extraction")
		}
		return nil
	}
	if err := sourcebuf.WriteFileChecked(ctx, cleanPath, updated, checkPreimage); err != nil {
		return "", fmt.Errorf("apply constant extraction: %w", err)
	}
	return UnifiedDiff(filepath.Base(cleanPath), string(content), string(updated)), nil
}

func loadConstantPackage(ctx context.Context, dir string, overlay map[string][]byte) ([]*packages.Package, error) {
	environment, err := gocache.Environment(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("prepare constant extraction Go environment: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read constant extraction package directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return nil, fmt.Errorf("inspect constant extraction package source %q: %w", path, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("constant extraction refuses symlinked Go package source %q", path)
		}
	}
	config := &packages.Config{Context: ctx, Dir: dir, Env: environment, BuildFlags: []string{"-mod=readonly"}, Tests: true, Overlay: overlay, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports}
	loaded, err := packages.Load(config, ".")
	if err != nil {
		return nil, fmt.Errorf("load Go package for constant extraction: %w", err)
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("load Go package for constant extraction: no package found")
	}
	primary := make([]*packages.Package, 0, 1)
	for _, pkg := range loaded {
		for _, item := range pkg.Errors {
			return nil, fmt.Errorf("load Go package for constant extraction: %s", item.Error())
		}
		if !hasConstantTestFiles(pkg.CompiledGoFiles) {
			primary = append(primary, pkg)
		}
	}
	if len(primary) == 0 {
		return nil, fmt.Errorf("load Go package for constant extraction: no primary package found")
	}
	return primary, nil
}

func hasConstantTestFiles(files []string) bool {
	for _, path := range files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			return true
		}
	}
	return false
}
