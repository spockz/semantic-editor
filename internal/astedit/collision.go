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
	"semedit/internal/sourcebuf"
)

type declarationCollision struct{ name, file string }

func checkInsertDeclCollisions(ctx context.Context, path string, declarations []ast.Decl, file *ast.File) error {
	seen := make(map[string]struct{})
	for _, decl := range declarations {
		for _, name := range extractDeclNames(decl) {
			if _, exists := seen[name]; exists {
				return fmt.Errorf("incoming declarations repeat symbol %q: %w", name, ErrDeclCollision)
			}
			seen[name] = struct{}{}
		}
	}
	collisions, err := findPackageDeclarationCollisions(ctx, path, file, declarations)
	if err != nil {
		return err
	}
	if len(collisions) == 0 {
		return nil
	}
	return fmt.Errorf("declaration %q already exists in %s: %w", collisions[0].name, collisions[0].file, ErrDeclCollision)
}

func findPackageDeclarationCollisions(ctx context.Context, targetPath string, target *ast.File, incoming []ast.Decl) ([]declarationCollision, error) {
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
			file, err = sourcebuf.ParseFile(ctx, fset, absPath, nil, parser.ParseComments|parser.AllErrors)
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

func checkConstructDeclarationCollision(ctx context.Context, targetPath string, target *ast.File, incoming ast.Decl) error {
	if fn, ok := incoming.(*ast.FuncDecl); ok && fn.Recv != nil {
		receiverType := extractReceiverTypeName(fn.Recv)
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("resolve target path: %w", err)
		}
		paths, err := filepath.Glob(filepath.Join(filepath.Dir(absTarget), "*.go"))
		if err != nil {
			return fmt.Errorf("list package files: %w", err)
		}
		fset := token.NewFileSet()
		for _, path := range paths {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("resolve package file %s: %w", path, err)
			}
			file := target
			if absPath != absTarget {
				file, err = sourcebuf.ParseFile(ctx, fset, absPath, nil, parser.ParseComments|parser.AllErrors)
				if err != nil {
					return fmt.Errorf("parse package file %s while checking collisions: %w", absPath, err)
				}
			}
			if file.Name.Name != target.Name.Name {
				continue
			}
			for _, decl := range file.Decls {
				existing, ok := decl.(*ast.FuncDecl)
				if !ok || existing.Recv == nil {
					continue
				}
				if existing.Name.Name == fn.Name.Name && extractReceiverTypeName(existing.Recv) == receiverType {
					return fmt.Errorf("method %s.%s already exists in %s: %w", receiverType, fn.Name.Name, absPath, ErrDeclCollision)
				}
			}
		}
		return nil
	}

	collisions, err := findPackageDeclarationCollisions(ctx, targetPath, target, []ast.Decl{incoming})
	if err != nil {
		return err
	}
	if len(collisions) > 0 {
		return fmt.Errorf("declaration %q already exists in %s: %w", collisions[0].name, collisions[0].file, ErrDeclCollision)
	}
	return nil
}
