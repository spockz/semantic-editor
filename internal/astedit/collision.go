// Package astedit checks package declarations before insertion so collisions
// are rejected before a write can create an invalid package.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

type declarationCollision struct{ name, file string }

func findPackageDeclarationCollisions(targetPath string, target *ast.File, incoming []ast.Decl) ([]declarationCollision, error) {
	wanted := make(map[string]struct{})
	for _, decl := range incoming {
		for _, name := range extractDeclNames(decl) {
			if name != "_" {
				wanted[name] = struct{}{}
			}
		}
	}
	if len(wanted) == 0 {
		return nil, nil
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve target path: %w", err)
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(absTarget), "*.go"))
	if err != nil {
		return nil, fmt.Errorf("list package files: %w", err)
	}
	if target == nil {
		return nil, fmt.Errorf("cannot check package declarations without a parsed target file")
	}
	fset := token.NewFileSet()
	var collisions []declarationCollision
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve package file %s: %w", path, err)
		}
		var file *ast.File
		if absPath == absTarget {
			file = target
		} else {
			file, err = parser.ParseFile(fset, absPath, nil, parser.ParseComments|parser.AllErrors)
			if err != nil {
				return nil, fmt.Errorf("parse package file %s while checking collisions: %w", absPath, err)
			}
		}
		if file.Name.Name != target.Name.Name {
			continue
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
				continue
			}
			for _, name := range extractDeclNames(decl) {
				if _, ok := wanted[name]; ok {
					collisions = append(collisions, declarationCollision{name: name, file: absPath})
				}
			}
		}
	}
	return collisions, nil
}

func handleInsertDeclCollisions(ctx context.Context, path, source string, declarations []ast.Decl, file *ast.File, opts DeclOptions) (bool, error) {
	seen := make(map[string]struct{})
	for _, decl := range declarations {
		for _, name := range extractDeclNames(decl) {
			if _, exists := seen[name]; exists {
				return false, fmt.Errorf("incoming declarations repeat symbol %q: %w", name, ErrDeclCollision)
			}
			seen[name] = struct{}{}
		}
	}
	collisions, err := findPackageDeclarationCollisions(path, file, declarations)
	if err != nil {
		return false, err
	}
	if len(collisions) == 0 {
		return false, nil
	}
	if !opts.Overwrite {
		return false, fmt.Errorf("declaration %q already exists in %s: %w", collisions[0].name, collisions[0].file, ErrDeclCollision)
	}
	if len(declarations) != 1 || len(collisions) != 1 {
		return false, fmt.Errorf("overwrite requires one declaration with one colliding symbol")
	}
	targetAbs, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("resolve target path: %w", err)
	}
	if collisions[0].file != targetAbs {
		return false, fmt.Errorf("cannot overwrite %q: it is declared in %s, not target file %s", collisions[0].name, collisions[0].file, targetAbs)
	}
	if _, err := ReplaceDecl(ctx, path, collisions[0].name, source, ReplaceDeclOptions{AutoOrganizeImports: opts.AutoOrganizeImports}); err != nil {
		return false, err
	}
	return true, nil
}
