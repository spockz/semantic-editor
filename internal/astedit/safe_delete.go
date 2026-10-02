// Package astedit isolates compiler-backed checks for narrowly safe Go declaration deletion.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type safeDeleteTarget struct {
	name string
	kind string
	pos  token.Pos
	end  token.Pos
}

func classifySafeDeleteTarget(file *ast.File, selected ast.Node) (safeDeleteTarget, string) {
	switch node := selected.(type) {
	case *ast.FuncDecl:
		if node.Recv != nil || node.Name == nil || ast.IsExported(node.Name.Name) || node.Name.Name == "main" || node.Name.Name == "init" {
			return safeDeleteTarget{}, "safe_delete supports only unexported free functions other than main and init"
		}
		return safeDeleteTarget{name: node.Name.Name, kind: "function", pos: node.Name.Pos(), end: node.End()}, ""
	case *ast.ValueSpec:
		if len(node.Names) != 1 {
			return safeDeleteTarget{}, "safe_delete refuses grouped or multi-name variable declarations"
		}
		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				if spec != node {
					continue
				}
				switch gen.Tok {
				case token.CONST:
					if ast.IsExported(node.Names[0].Name) {
						return safeDeleteTarget{}, "safe_delete refuses exported declarations"
					}
					return safeDeleteTarget{name: node.Names[0].Name, kind: "constant", pos: node.Names[0].Pos(), end: node.End()}, ""
				case token.VAR:
					if ast.IsExported(node.Names[0].Name) {
						return safeDeleteTarget{}, "safe_delete refuses exported declarations"
					}
					return safeDeleteTarget{name: node.Names[0].Name, kind: "value", pos: node.Names[0].Pos(), end: node.End()}, ""
				default:
					return safeDeleteTarget{}, "safe_delete supports only constant and variable declarations"
				}
			}
		}
		return safeDeleteTarget{}, "safe_delete requires a standalone declaration"
	case ast.Spec:
		return safeDeleteTarget{}, "safe_delete supports only standalone declarations"
	default:
		return safeDeleteTarget{}, "safe_delete supports only unexported standalone constants, variables, and free functions"
	}
}

func verifySafeDeleteTarget(ctx context.Context, root, targetPath string, target safeDeleteTarget) (string, error) {
	pkgs, err := loadMovePackages(ctx, root, nil)
	if err != nil {
		return "", err
	}
	if problem := safeDeletePackageProblem(pkgs, root); problem != "" {
		return "cannot prove safe deletion across active Go packages: " + problem, nil
	}
	targetObject, targetPkg, initializerRefusal := safeDeleteDefinition(pkgs, root, targetPath, target)
	if initializerRefusal != "" {
		return initializerRefusal, nil
	}
	if targetObject == nil {
		return "target is not present in an active, successfully type-checked package", nil
	}
	if target.kind == "function" {
		if _, ok := targetObject.(*types.Func); !ok {
			return "target does not resolve to a free function", nil
		}
	}
	if safeDeleteHasReference(pkgs, root, targetPath, target, targetObject, targetPkg) {
		return fmt.Sprintf("safe_delete found a compiler-resolved reference to %q", target.name), nil
	}
	return "", nil
}

func safeDeletePackageProblem(pkgs []*packages.Package, root string) string {
	problem := ""
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.Module == nil || !pkg.Module.Main || !safeDeleteWithin(root, pkg.Module.Dir) {
			return true
		}
		for _, item := range pkg.Errors {
			if problem == "" {
				problem = item.Error()
			}
		}
		for _, imported := range pkg.Imports {
			if imported.PkgPath == "reflect" || imported.PkgPath == "plugin" || imported.PkgPath == "unsafe" {
				problem = "module imports reflect, plugin, or unsafe, so dynamic references cannot be excluded"
			}
		}
		return true
	}, nil)
	return problem
}

func safeDeleteDefinition(pkgs []*packages.Package, root, targetPath string, target safeDeleteTarget) (types.Object, string, string) {
	var selected types.Object
	packagePath := ""
	refusal := ""
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.Module == nil || !pkg.Module.Main || !safeDeleteWithin(root, pkg.Module.Dir) {
			return true
		}
		object, effectful := safeDeleteDefinitionInPackage(pkg, targetPath, target)
		if effectful {
			refusal = "variable initializer is not proven to be effect-free"
		}
		if object != nil {
			selected, packagePath = object, pkg.PkgPath
		}
		return true
	}, nil)
	return selected, packagePath, refusal
}

func safeDeleteDefinitionInPackage(pkg *packages.Package, targetPath string, target safeDeleteTarget) (types.Object, bool) {
	for index, compiled := range pkg.CompiledGoFiles {
		if filepath.Clean(compiled) != filepath.Clean(targetPath) || index >= len(pkg.Syntax) {
			continue
		}
		for _, declaration := range pkg.Syntax[index].Decls {
			object, effectful := safeDeleteDefinitionInDecl(pkg, declaration, target)
			if object != nil || effectful {
				return object, effectful
			}
		}
	}
	return nil, false
}

func safeDeleteDefinitionInDecl(pkg *packages.Package, declaration ast.Decl, target safeDeleteTarget) (types.Object, bool) {
	switch node := declaration.(type) {
	case *ast.FuncDecl:
		if target.kind == "function" && node.Name.Name == target.name && pkg.Fset.Position(node.Name.Pos()).Offset == pkg.Fset.Position(target.pos).Offset {
			return pkg.TypesInfo.Defs[node.Name], false
		}
	case *ast.GenDecl:
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || target.kind != "value" {
				continue
			}
			for _, name := range value.Names {
				if name.Name != target.name || pkg.Fset.Position(name.Pos()).Offset != pkg.Fset.Position(target.pos).Offset {
					continue
				}
				object := pkg.TypesInfo.Defs[name]
				variable, ok := object.(*types.Var)
				if !ok {
					return nil, false
				}
				return object, !safeDeleteVariableInitializer(pkg, node, value, variable)
			}
		}
	}
	return nil, false
}

func safeDeleteHasReference(pkgs []*packages.Package, root, targetPath string, target safeDeleteTarget, targetObject types.Object, targetPkg string) bool {
	foundUse := false
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.Module == nil || !pkg.Module.Main || !safeDeleteWithin(root, pkg.Module.Dir) {
			return true
		}
		for ident, object := range pkg.TypesInfo.Uses {
			if object == nil || object.Name() != target.name || object.Pkg() == nil || object.Pkg().Path() != targetPkg {
				continue
			}
			if object.Pos() == targetObject.Pos() && (filepath.Clean(pkg.Fset.Position(ident.Pos()).Filename) != filepath.Clean(targetPath) || ident.Pos() < target.pos || ident.End() > target.end) {
				foundUse = true
				return false
			}
			_ = ident
		}
		return !foundUse
	}, nil)
	return foundUse
}

func safeDeleteWithin(root, directory string) bool {
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func safeDeleteVariableInitializer(pkg *packages.Package, declaration *ast.GenDecl, value *ast.ValueSpec, variable *types.Var) bool {
	if declaration.Tok != token.VAR || declaration.Lparen.IsValid() || len(declaration.Specs) != 1 || len(value.Names) != 1 || ast.IsExported(variable.Name()) {
		return false
	}
	if _, ok := types.Unalias(variable.Type()).Underlying().(*types.Basic); !ok {
		return false
	}
	if len(value.Values) == 0 {
		return true
	}
	if len(value.Values) != 1 {
		return false
	}
	typed, ok := pkg.TypesInfo.Types[value.Values[0]]
	return ok && typed.Value != nil
}
