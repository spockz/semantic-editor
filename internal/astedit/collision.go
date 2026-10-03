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

type packageSourceFiles struct {
	paths      []string
	targetPath string
}

type parsedCollisionSource struct {
	path string
	file *ast.File
}

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
	wanted := declarationNames(incoming)
	if len(wanted) == 0 {
		return nil, nil
	}
	sources, err := packageSourcePaths(targetPath)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("cannot check package declarations without a parsed target file")
	}
	fset := token.NewFileSet()
	var collisions []declarationCollision
	for _, path := range sources.paths {
		parsed, err := parseCollisionFile(ctx, fset, path, sources.targetPath, target)
		if err != nil {
			return nil, err
		}
		if parsed.file.Name.Name != target.Name.Name {
			continue
		}
		collisions = append(collisions, fileDeclarationCollisions(parsed.file, parsed.path, wanted)...)
	}
	return collisions, nil
}

func declarationNames(declarations []ast.Decl) map[string]struct{} {
	wanted := make(map[string]struct{})
	for _, decl := range declarations {
		for _, name := range extractDeclNames(decl) {
			if name != "_" {
				wanted[name] = struct{}{}
			}
		}
	}
	return wanted
}

func packageSourcePaths(targetPath string) (packageSourceFiles, error) {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return packageSourceFiles{}, fmt.Errorf("resolve target path: %w", err)
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(absTarget), "*.go"))
	if err != nil {
		return packageSourceFiles{}, fmt.Errorf("list package files: %w", err)
	}
	return packageSourceFiles{paths: paths, targetPath: absTarget}, nil
}

func parseCollisionFile(ctx context.Context, fset *token.FileSet, path, absTarget string, target *ast.File) (parsedCollisionSource, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return parsedCollisionSource{}, fmt.Errorf("resolve package file %s: %w", path, err)
	}
	if absPath == absTarget {
		return parsedCollisionSource{path: absPath, file: target}, nil
	}
	file, err := sourcebuf.ParseFile(ctx, fset, absPath, nil, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return parsedCollisionSource{}, fmt.Errorf("parse package file %s while checking collisions: %w", absPath, err)
	}
	return parsedCollisionSource{path: absPath, file: file}, nil
}

func fileDeclarationCollisions(file *ast.File, path string, wanted map[string]struct{}) []declarationCollision {
	var collisions []declarationCollision
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
			continue
		}
		for _, name := range extractDeclNames(decl) {
			if _, ok := wanted[name]; ok {
				collisions = append(collisions, declarationCollision{name: name, file: path})
			}
		}
	}
	return collisions
}

func checkConstructDeclarationCollision(ctx context.Context, targetPath string, target *ast.File, incoming ast.Decl) error {
	if fn, ok := incoming.(*ast.FuncDecl); ok && fn.Recv != nil {
		return checkMethodCollision(ctx, targetPath, target, fn)
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

func checkMethodCollision(ctx context.Context, targetPath string, target *ast.File, incoming *ast.FuncDecl) error {
	sources, err := packageSourcePaths(targetPath)
	if err != nil {
		return err
	}
	if target == nil {
		return fmt.Errorf("cannot check package declarations without a parsed target file")
	}
	fset := token.NewFileSet()
	for _, path := range sources.paths {
		parsed, err := parseCollisionFile(ctx, fset, path, sources.targetPath, target)
		if err != nil {
			return err
		}
		if parsed.file.Name.Name != target.Name.Name {
			continue
		}
		if methodExists(parsed.file, incoming) {
			receiverType := extractReceiverTypeName(incoming.Recv)
			return fmt.Errorf("method %s.%s already exists in %s: %w", receiverType, incoming.Name.Name, parsed.path, ErrDeclCollision)
		}
	}
	return nil
}

func methodExists(file *ast.File, incoming *ast.FuncDecl) bool {
	receiverType := extractReceiverTypeName(incoming.Recv)
	for _, decl := range file.Decls {
		existing, ok := decl.(*ast.FuncDecl)
		if ok && existing.Recv != nil && existing.Name.Name == incoming.Name.Name && extractReceiverTypeName(existing.Recv) == receiverType {
			return true
		}
	}
	return false
}
