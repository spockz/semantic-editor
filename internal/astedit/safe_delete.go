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
	if problem := safeDeleteImportProblem(pkgs, targetPath, target); problem != "" {
		return problem, nil
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
		moduleSource := false
		for _, file := range pkg.GoFiles {
			if safeDeleteModuleSource(root, file) {
				moduleSource = true
				break
			}
		}
		if !moduleSource {
			return true
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
		if target.kind == "function" && node.Name.Name == target.name {
			return pkg.TypesInfo.Defs[node.Name], false
		}
	case *ast.GenDecl:
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || (target.kind != "value" && target.kind != "constant") {
				continue
			}
			for _, name := range value.Names {
				if name.Name != target.name {
					continue
				}
				object := pkg.TypesInfo.Defs[name]
				if target.kind == "constant" {
					if _, ok := object.(*types.Const); ok {
						return object, false
					}
					return nil, false
				}
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
	if targetObject == nil || targetObject.Pkg() == nil || targetObject.Pkg().Path() != targetPkg {
		return true
	}
	foundUse := false
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.Module == nil || !pkg.Module.Main || !safeDeleteWithin(root, pkg.Module.Dir) {
			return true
		}
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			foundUse = true
			return false
		}
		var selected ast.Decl
		var selectedObject types.Object
		for _, file := range pkg.Syntax {
			if filepath.Clean(pkg.Fset.Position(file.Pos()).Filename) != filepath.Clean(targetPath) {
				continue
			}
			for _, declaration := range file.Decls {
				object, matched := safeDeleteDefinitionInDecl(pkg, declaration, target)
				if matched && object != nil {
					selected = declaration
					selectedObject = object
				}
			}
		}
		if selectedObject == nil {
			return true
		}
		for ident, object := range pkg.TypesInfo.Uses {
			if object != selectedObject {
				continue
			}
			if ident.Pos() < selected.Pos() || ident.End() > selected.End() {
				foundUse = true
				return false
			}
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

func safeDeleteImportProblem(pkgs []*packages.Package, targetPath string, target safeDeleteTarget) string {
	problem := ""
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if problem != "" {
			return false
		}
		problem = safeDeletePackageImportProblem(pkg, targetPath, target)
		return problem == ""
	}, nil)
	return problem
}

func safeDeletePackageImportProblem(pkg *packages.Package, targetPath string, target safeDeleteTarget) string {
	if pkg.TypesInfo == nil || pkg.Fset == nil {
		return ""
	}
	for index, path := range pkg.CompiledGoFiles {
		if filepath.Clean(path) != filepath.Clean(targetPath) || index >= len(pkg.Syntax) {
			continue
		}
		file := pkg.Syntax[index]
		selected := safeDeleteSelectedDeclaration(pkg, file, target)
		if selected == nil {
			continue
		}
		if problem := safeDeleteSelectedImportProblem(pkg, file, selected); problem != "" {
			return problem
		}
	}
	return ""
}

func safeDeleteSelectedDeclaration(pkg *packages.Package, file *ast.File, target safeDeleteTarget) ast.Node {
	for _, declaration := range file.Decls {
		object, _ := safeDeleteDefinitionInDecl(pkg, declaration, target)
		if object != nil {
			return declaration
		}
	}
	return nil
}

func safeDeleteSelectedImportProblem(pkg *packages.Package, file *ast.File, selected ast.Node) string {
	for _, imported := range file.Imports {
		if imported.Name != nil && imported.Name.Name == "." {
			return "safe_delete cannot prove import initialization with a dot import"
		}
		importedName := safeDeleteImportObject(pkg.TypesInfo, imported)
		if importedName == nil || importedName.Name() == "_" {
			continue
		}
		uses := safeDeleteImportUses(pkg.TypesInfo, file, selected, importedName)
		if uses.inside && !uses.outside {
			return "safe_delete would remove the final use of an import and change package initialization; retain initialization explicitly before deleting"
		}
	}
	return ""
}

func safeDeleteImportObject(info *types.Info, imported *ast.ImportSpec) types.Object {
	if imported.Name != nil {
		return info.Defs[imported.Name]
	}
	return info.Implicits[imported]
}

type safeDeleteImportUseSet struct {
	inside  bool
	outside bool
}

func safeDeleteImportUses(info *types.Info, file *ast.File, selected ast.Node, imported types.Object) safeDeleteImportUseSet {
	uses := safeDeleteImportUseSet{}
	for identifier, object := range info.Uses {
		if object != imported || identifier.Pos() < file.Pos() || identifier.End() > file.End() {
			continue
		}
		if identifier.Pos() >= selected.Pos() && identifier.End() <= selected.End() {
			uses.inside = true
		} else {
			uses.outside = true
		}
	}
	return uses
}

func safeDeleteModuleSource(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return false
	}
	for directory := range strings.SplitSeq(filepath.Dir(relative), string(filepath.Separator)) {
		if directory != "." && (strings.HasPrefix(directory, ".") || strings.HasPrefix(directory, "_") || directory == "vendor") {
			return false
		}
	}
	return true
}
