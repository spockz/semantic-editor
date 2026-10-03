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
	packageName, typeName, err := parseStructuralFieldSelector(selector)
	if err != nil {
		return nil, nil, "", err
	}
	var selected structuralFieldSelection
	for _, pkg := range snapshot.Packages {
		candidate, found, err := structuralFieldPackageOwner(snapshot, pkg, packageName, typeName, selector)
		if err != nil {
			return nil, nil, "", err
		}
		if !found {
			continue
		}
		if selected.Package != nil {
			return nil, nil, "", fmt.Errorf("in_type %q is ambiguous in the selected package", selector)
		}
		selected = candidate
	}
	if selected.Package == nil || selected.Named == nil {
		return nil, nil, "", fmt.Errorf("in_type %q was not found in the selected Go file", selector)
	}
	if ast.IsExported(typeName) {
		return nil, nil, "", fmt.Errorf("cannot prove positional-literal compatibility for exported owner type %q", selector)
	}
	return selected.Package, selected.Named, typeName, nil
}

type structuralFieldSelection struct {
	Package *packages.Package
	Named   *types.Named
}

func parseStructuralFieldSelector(selector string) (string, string, error) {
	parts := strings.Split(selector, ".")
	if len(parts) != 1 && len(parts) != 2 {
		return "", "", fmt.Errorf("in_type %q must be a local Go type name, optionally prefixed by its package name", selector)
	}
	typeName := parts[len(parts)-1]
	if !token.IsIdentifier(typeName) || token.Lookup(typeName).IsKeyword() {
		return "", "", fmt.Errorf("in_type %q has an invalid type name", selector)
	}
	packageName := ""
	if len(parts) == 2 {
		packageName = parts[0]
	}
	return packageName, typeName, nil
}

func structuralFieldPackageOwner(snapshot *structuralSnapshot, pkg *packages.Package, packageName, typeName, selector string) (structuralFieldSelection, bool, error) {
	contains, err := packageContainsStructuralFile(pkg, snapshot.File)
	if err != nil || !contains || !validStructuralFieldPackage(pkg) || packageName != "" && pkg.Name != packageName {
		return structuralFieldSelection{}, false, err
	}
	if len(pkg.Errors) > 0 || pkg.IllTyped {
		message := "package is ill-typed"
		if len(pkg.Errors) > 0 {
			message = pkg.Errors[0].Msg
		}
		return structuralFieldSelection{}, false, fmt.Errorf("selected package %s is not fully typechecked: %s", pkg.PkgPath, message)
	}
	file, err := structuralFieldSelectedFile(pkg, snapshot.File)
	if err != nil || file == nil {
		return structuralFieldSelection{}, false, err
	}
	named, found, err := structuralFieldDeclaredType(pkg, file, typeName, selector)
	if err != nil || !found {
		return structuralFieldSelection{}, false, err
	}
	return structuralFieldSelection{Package: pkg, Named: named}, true, nil
}

func validStructuralFieldPackage(pkg *packages.Package) bool {
	return pkg.ID == pkg.PkgPath && !strings.Contains(pkg.PkgPath, " [") && pkg.Types != nil && pkg.Types.Path() == pkg.PkgPath && pkg.TypesInfo != nil && pkg.Fset != nil
}

func structuralFieldSelectedFile(pkg *packages.Package, selectedPath string) (*ast.File, error) {
	var selected *ast.File
	for _, file := range pkg.Syntax {
		path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
		if err != nil {
			return nil, fmt.Errorf("resolve selected package syntax: %w", err)
		}
		if path != selectedPath {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("selected Go source is ambiguous in package %s", pkg.PkgPath)
		}
		selected = file
	}
	return selected, nil
}

func structuralFieldDeclaredType(pkg *packages.Package, file *ast.File, typeName, selector string) (*types.Named, bool, error) {
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, spec := range group.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			named, err := validatedStructuralFieldType(pkg, typeSpec, selector)
			return named, true, err
		}
	}
	return nil, false, nil
}

func validatedStructuralFieldType(pkg *packages.Package, spec *ast.TypeSpec, selector string) (*types.Named, error) {
	object, ok := pkg.TypesInfo.Defs[spec.Name].(*types.TypeName)
	if !ok || object.IsAlias() || spec.Assign.IsValid() || spec.TypeParams != nil && len(spec.TypeParams.List) > 0 {
		return nil, fmt.Errorf("in_type %q must be a directly declared non-generic struct", selector)
	}
	resolved, ok := types.Unalias(object.Type()).(*types.Named)
	if !ok || resolved.Obj() != object {
		return nil, fmt.Errorf("in_type %q does not resolve to its exact declared type", selector)
	}
	if _, ok := resolved.Underlying().(*types.Struct); !ok {
		return nil, fmt.Errorf("in_type %q is not a struct", selector)
	}
	return resolved, nil
}

func structuralFieldSourceSet(snapshot *structuralSnapshot, owner *packages.Package, named *types.Named, typeName, newField string) error {
	files, err := captureStructuralFieldFiles(snapshot, owner, typeName, newField)
	if err != nil {
		return err
	}
	if method := types.NewMethodSet(types.NewPointer(named)).Lookup(owner.Types, newField); method != nil {
		return fmt.Errorf("field %q collides with a method in the complete owner method set", newField)
	}
	aliases := structuralFieldAliases(files, typeName)
	return validateNoPositionalFieldLiterals(files, aliases)
}

func captureStructuralFieldFiles(snapshot *structuralSnapshot, owner *packages.Package, typeName, newField string) (map[string]*ast.File, error) {
	directory := filepath.Dir(snapshot.File)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("enumerate field owner source directory %q: %w", directory, err)
	}
	files := make(map[string]*ast.File)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		file, absolute, err := captureStructuralFieldFile(snapshot, owner, directory, entry.Name(), typeName, newField)
		if err != nil {
			return nil, err
		}
		if file == nil {
			continue
		}
		files[absolute] = file
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("field owner package %s has no source files", owner.PkgPath)
	}
	return files, nil
}

func captureStructuralFieldFile(snapshot *structuralSnapshot, owner *packages.Package, directory, filename, typeName, newField string) (*ast.File, string, error) {
	path := filepath.Join(directory, filename)
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve field owner source %q: %w", path, err)
	}
	source, err := sourcebuf.ReadFile(snapshot.Ctx, absolute)
	if err != nil {
		return nil, "", fmt.Errorf("capture field owner source %q: %w", path, err)
	}
	if previous, ok := snapshot.Sources[absolute]; ok && !bytes.Equal(previous, source) {
		return nil, "", fmt.Errorf("field owner source %q differs from the analyzed snapshot", path)
	}
	snapshot.Sources[absolute] = bytes.Clone(source)
	file, err := parser.ParseFile(token.NewFileSet(), absolute, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, "", fmt.Errorf("parse field owner source %q: %w", path, err)
	}
	if file.Name.Name != owner.Name {
		return nil, absolute, nil
	}
	if err := rejectStructuralFieldMethod(file, typeName, newField); err != nil {
		return nil, "", err
	}
	return file, absolute, nil
}

func rejectStructuralFieldMethod(file *ast.File, typeName, newField string) error {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv != nil && function.Name.Name == newField && structuralReceiverType(function.Recv) == typeName {
			return fmt.Errorf("field %q collides with a method on %s", newField, typeName)
		}
	}
	return nil
}

func structuralFieldAliases(files map[string]*ast.File, typeName string) map[string]bool {
	aliases := map[string]bool{typeName: true}
	for extendStructuralFieldAliases(files, aliases) {
	}
	return aliases
}

func extendStructuralFieldAliases(files map[string]*ast.File, aliases map[string]bool) bool {
	changed := false
	for _, file := range files {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, raw := range group.Specs {
				spec, ok := raw.(*ast.TypeSpec)
				if ok && structuralAliasExtends(spec, aliases) {
					changed = true
				}
			}
		}
	}
	return changed
}

func structuralAliasExtends(spec *ast.TypeSpec, aliases map[string]bool) bool {
	if !spec.Assign.IsValid() || aliases[spec.Name.Name] {
		return false
	}
	base, ok := spec.Type.(*ast.Ident)
	if !ok || !aliases[base.Name] {
		return false
	}
	aliases[spec.Name.Name] = true
	return true
}

func validateNoPositionalFieldLiterals(files map[string]*ast.File, aliases map[string]bool) error {
	for path, file := range files {
		positional := positionalStructuralLiteral(file, aliases)
		if positional != "" {
			return fmt.Errorf("cannot add field while source contains unkeyed %s literals (%s)", positional, path)
		}
	}
	return nil
}

func positionalStructuralLiteral(file *ast.File, aliases map[string]bool) string {
	var positional string
	ast.Inspect(file, func(node ast.Node) bool {
		if positional != "" {
			return false
		}
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
	return positional
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
