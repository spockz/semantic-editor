// This file proves field insertion against the exact non-generic Go type and its complete same-package literal source set.

package operation

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
	"strings"

	"golang.org/x/tools/go/packages"
)

func structuralFieldOwner(snapshot *structuralSnapshot, selector string) (*packages.Package, *types.Named, string, error) {
	parts := strings.Split(selector, ".")
	if len(parts) < 1 || len(parts) > 2 {
		return nil, nil, "", fmt.Errorf("in_type %q must be a local Go type name, optionally prefixed by its package name", selector)
	}
	typeName := parts[len(parts)-1]
	if !token.IsIdentifier(typeName) || token.Lookup(typeName).IsKeyword() {
		return nil, nil, "", fmt.Errorf("in_type %q has an invalid type name", selector)
	}
	var owner *packages.Package
	var named *types.Named
	for _, pkg := range snapshot.Packages {
		contains, err := packageContainsStructuralFile(pkg, snapshot.File)
		if err != nil {
			return nil, nil, "", err
		}
		if !contains || pkg.ID != pkg.PkgPath || strings.Contains(pkg.PkgPath, " [") || pkg.Types == nil || pkg.Types.Path() != pkg.PkgPath || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		if parts[0] != typeName && pkg.Name != parts[0] {
			continue
		}
		if len(pkg.Errors) > 0 || pkg.IllTyped {
			message := "package is ill-typed"
			if len(pkg.Errors) > 0 {
				message = pkg.Errors[0].Msg
			}
			return nil, nil, "", fmt.Errorf("selected package %s is not fully typechecked: %s", pkg.PkgPath, message)
		}
		var selectedFile *ast.File
		for _, file := range pkg.Syntax {
			path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
			if err != nil {
				return nil, nil, "", fmt.Errorf("resolve selected package syntax: %w", err)
			}
			if path == snapshot.File {
				if selectedFile != nil {
					return nil, nil, "", fmt.Errorf("selected Go source is ambiguous in package %s", pkg.PkgPath)
				}
				selectedFile = file
			}
		}
		if selectedFile == nil {
			continue
		}
		for _, declaration := range selectedFile.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, spec := range group.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				object, ok := pkg.TypesInfo.Defs[typeSpec.Name].(*types.TypeName)
				if !ok || object.IsAlias() || typeSpec.Assign.IsValid() || typeSpec.TypeParams != nil && len(typeSpec.TypeParams.List) > 0 {
					return nil, nil, "", fmt.Errorf("in_type %q must be a directly declared non-generic struct", selector)
				}
				resolved, ok := types.Unalias(object.Type()).(*types.Named)
				if !ok || resolved.Obj() != object {
					return nil, nil, "", fmt.Errorf("in_type %q does not resolve to its exact declared type", selector)
				}
				if _, ok := resolved.Underlying().(*types.Struct); !ok {
					return nil, nil, "", fmt.Errorf("in_type %q is not a struct", selector)
				}
				if owner != nil {
					return nil, nil, "", fmt.Errorf("in_type %q is ambiguous in the selected package", selector)
				}
				owner, named = pkg, resolved
			}
		}
	}
	if owner == nil || named == nil {
		return nil, nil, "", fmt.Errorf("in_type %q was not found in the selected Go file", selector)
	}
	if ast.IsExported(typeName) {
		return nil, nil, "", fmt.Errorf("cannot prove positional-literal compatibility for exported owner type %q", selector)
	}
	return owner, named, typeName, nil
}

func structuralFieldSourceSet(snapshot *structuralSnapshot, owner *packages.Package, named *types.Named, typeName, newField string) error {
	directory := filepath.Dir(snapshot.File)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("enumerate field owner source directory %q: %w", directory, err)
	}
	files := make(map[string]*ast.File)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve field owner source %q: %w", path, err)
		}
		source, err := sourcebuf.ReadFile(snapshot.Ctx, absolute)
		if err != nil {
			return fmt.Errorf("capture field owner source %q: %w", path, err)
		}
		if previous, ok := snapshot.Sources[absolute]; ok && !bytes.Equal(previous, source) {
			return fmt.Errorf("field owner source %q differs from the analyzed snapshot", path)
		}
		snapshot.Sources[absolute] = bytes.Clone(source)
		file, err := parser.ParseFile(token.NewFileSet(), absolute, source, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return fmt.Errorf("parse field owner source %q: %w", path, err)
		}
		if file.Name.Name != owner.Name {
			continue
		}
		files[absolute] = file
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Name.Name != newField {
				continue
			}
			if structuralReceiverType(function.Recv) == typeName {
				return fmt.Errorf("field %q collides with a method on %s", newField, typeName)
			}
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("field owner package %s has no source files", owner.PkgPath)
	}
	if method := types.NewMethodSet(types.NewPointer(named)).Lookup(owner.Types, newField); method != nil {
		return fmt.Errorf("field %q collides with a method in the complete owner method set", newField)
	}
	aliases := map[string]bool{typeName: true}
	changed := true
	for changed {
		changed = false
		for _, file := range files {
			for _, declaration := range file.Decls {
				group, ok := declaration.(*ast.GenDecl)
				if !ok || group.Tok != token.TYPE {
					continue
				}
				for _, raw := range group.Specs {
					spec, ok := raw.(*ast.TypeSpec)
					if !ok || !spec.Assign.IsValid() || aliases[spec.Name.Name] {
						continue
					}
					base, ok := spec.Type.(*ast.Ident)
					if ok && aliases[base.Name] {
						aliases[spec.Name.Name] = true
						changed = true
					}
				}
			}
		}
	}
	for path, file := range files {
		var positional string
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok || len(literal.Elts) == 0 {
				return true
			}
			name, ok := literal.Type.(*ast.Ident)
			if !ok || !aliases[name.Name] {
				return true
			}
			for _, element := range literal.Elts {
				if _, ok := element.(*ast.KeyValueExpr); !ok {
					positional = name.Name
					return false
				}
			}
			return true
		})
		if positional != "" {
			return fmt.Errorf("cannot add field while source contains unkeyed %s literals (%s)", positional, path)
		}
	}
	return nil
}

func structuralReceiverType(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) != 1 {
		return ""
	}
	var receiverType func(ast.Expr) string
	receiverType = func(expression ast.Expr) string {
		switch node := expression.(type) {
		case *ast.Ident:
			return node.Name
		case *ast.StarExpr:
			return receiverType(node.X)
		case *ast.ParenExpr:
			return receiverType(node.X)
		}
		return ""
	}
	return receiverType(receiver.List[0].Type)
}

func resolveStructuralFieldTarget(ctx context.Context, root string, candidate SymbolTarget) (SymbolTarget, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(candidate.File))
	source, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("read final field target %q: %w", path, err)
	}
	separator := strings.LastIndex(candidate.Symbol, ".")
	if separator <= 0 || separator == len(candidate.Symbol)-1 {
		return SymbolTarget{}, false, nil
	}
	ownerSelector, fieldName := candidate.Symbol[:separator], candidate.Symbol[separator+1:]
	ownerType := ownerSelector[strings.LastIndex(ownerSelector, ".")+1:]
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("parse final field target %q: %w", path, err)
	}
	found := 0
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, raw := range group.Specs {
			spec, ok := raw.(*ast.TypeSpec)
			if !ok || spec.Name.Name != ownerType || spec.Assign.IsValid() {
				continue
			}
			structure, ok := spec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				for _, name := range field.Names {
					if name.Name == fieldName {
						found++
					}
				}
			}
		}
	}
	return candidate, found == 1, nil
}
