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
		return classifySafeDeleteFunction(node)
	case *ast.ValueSpec:
		return classifySafeDeleteValue(file, node)
	case ast.Spec:
		return safeDeleteTarget{}, "safe_delete supports only standalone declarations"
	default:
		return safeDeleteTarget{}, "safe_delete supports only unexported standalone constants, variables, and free functions"
	}
}

func classifySafeDeleteFunction(function *ast.FuncDecl) (safeDeleteTarget, string) {
	if function.Recv != nil || function.Name == nil || ast.IsExported(function.Name.Name) || function.Name.Name == "main" || function.Name.Name == "init" {
		return safeDeleteTarget{}, "safe_delete supports only unexported free functions other than main and init"
	}
	return safeDeleteTarget{name: function.Name.Name, kind: "function", pos: function.Name.Pos(), end: function.End()}, ""
}

func classifySafeDeleteValue(file *ast.File, value *ast.ValueSpec) (safeDeleteTarget, string) {
	if len(value.Names) != 1 {
		return safeDeleteTarget{}, "safe_delete refuses grouped or multi-name variable declarations"
	}
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if ok {
			classification := classifySafeDeleteValueGroup(gen, value)
			if classification.found {
				return classification.target, classification.refusal
			}
		}
	}
	return safeDeleteTarget{}, "safe_delete requires a standalone declaration"
}

type safeDeleteValueClassification struct {
	target  safeDeleteTarget
	refusal string
	found   bool
}

func classifySafeDeleteValueGroup(group *ast.GenDecl, value *ast.ValueSpec) safeDeleteValueClassification {
	for _, spec := range group.Specs {
		if spec != value {
			continue
		}
		if group.Tok != token.CONST && group.Tok != token.VAR {
			return safeDeleteValueClassification{refusal: "safe_delete supports only constant and variable declarations", found: true}
		}
		name := value.Names[0]
		if ast.IsExported(name.Name) {
			return safeDeleteValueClassification{refusal: "safe_delete refuses exported declarations", found: true}
		}
		kind := "constant"
		if group.Tok == token.VAR {
			kind = "value"
		}
		return safeDeleteValueClassification{target: safeDeleteTarget{name: name.Name, kind: kind, pos: name.Pos(), end: value.End()}, found: true}
	}
	return safeDeleteValueClassification{}
}

type safeDeleteTargetDefinition struct {
	object             types.Object
	packagePath        string
	initializerRefusal string
}

func safeDeleteDefinition(pkgs []*packages.Package, root, targetPath string, target safeDeleteTarget) safeDeleteTargetDefinition {
	definition := safeDeleteTargetDefinition{}
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.Module == nil || !pkg.Module.Main || !safeDeleteWithin(root, pkg.Module.Dir) {
			return true
		}
		result := safeDeleteDefinitionInPackage(pkg, targetPath, target)
		if result.effectfulInitializer {
			definition.initializerRefusal = "variable initializer is not proven to be effect-free"
		}
		if result.object != nil {
			definition.object, definition.packagePath = result.object, pkg.PkgPath
		}
		return true
	}, nil)
	return definition
}

func safeDeleteDefinitionInPackage(pkg *packages.Package, targetPath string, target safeDeleteTarget) safeDeleteDefinitionResult {
	for index, compiled := range pkg.CompiledGoFiles {
		if filepath.Clean(compiled) != filepath.Clean(targetPath) || index >= len(pkg.Syntax) {
			continue
		}
		for _, declaration := range pkg.Syntax[index].Decls {
			result := safeDeleteDefinitionInDecl(pkg, declaration, target)
			if result.matched {
				return result
			}
		}
	}
	return safeDeleteDefinitionResult{}
}

type safeDeleteDefinitionResult struct {
	object               types.Object
	effectfulInitializer bool
	matched              bool
}

func safeDeleteDefinitionInDecl(pkg *packages.Package, declaration ast.Decl, target safeDeleteTarget) safeDeleteDefinitionResult {
	switch node := declaration.(type) {
	case *ast.FuncDecl:
		return safeDeleteFunctionDefinition(pkg.TypesInfo, node, target)
	case *ast.GenDecl:
		return safeDeleteGroupDefinition(pkg, node, target)
	}
	return safeDeleteDefinitionResult{}
}

func safeDeleteFunctionDefinition(info *types.Info, function *ast.FuncDecl, target safeDeleteTarget) safeDeleteDefinitionResult {
	if target.kind == "function" && function.Name.Name == target.name {
		return safeDeleteDefinitionResult{object: info.Defs[function.Name], matched: true}
	}
	return safeDeleteDefinitionResult{}
}

func safeDeleteGroupDefinition(pkg *packages.Package, group *ast.GenDecl, target safeDeleteTarget) safeDeleteDefinitionResult {
	if target.kind != "value" && target.kind != "constant" {
		return safeDeleteDefinitionResult{}
	}
	for _, spec := range group.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		result := safeDeleteValueDefinition(pkg, group, value, target)
		if result.matched {
			return result
		}
	}
	return safeDeleteDefinitionResult{}
}

func safeDeleteValueDefinition(pkg *packages.Package, group *ast.GenDecl, value *ast.ValueSpec, target safeDeleteTarget) safeDeleteDefinitionResult {
	for _, name := range value.Names {
		if name.Name != target.name {
			continue
		}
		object := pkg.TypesInfo.Defs[name]
		if target.kind == "constant" {
			if _, ok := object.(*types.Const); ok {
				return safeDeleteDefinitionResult{object: object, matched: true}
			}
			return safeDeleteDefinitionResult{matched: true}
		}
		variable, ok := object.(*types.Var)
		if !ok {
			return safeDeleteDefinitionResult{matched: true}
		}
		return safeDeleteDefinitionResult{object: object, effectfulInitializer: !safeDeleteVariableInitializer(pkg, group, value, variable), matched: true}
	}
	return safeDeleteDefinitionResult{}
}

func safeDeleteHasReference(pkgs []*packages.Package, root, targetPath string, target safeDeleteTarget, targetObject types.Object, targetPkg string) bool {
	if targetObject == nil || targetObject.Pkg() == nil || targetObject.Pkg().Path() != targetPkg {
		return true
	}
	referenced := false
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		scan := safeDeletePackageReference(pkg, root, targetPath, target)
		referenced = referenced || scan.referenced
		return !scan.stop
	}, nil)
	return referenced
}

type safeDeleteReferenceScanResult struct {
	referenced bool
	stop       bool
}

func safeDeletePackageReference(pkg *packages.Package, root, targetPath string, target safeDeleteTarget) safeDeleteReferenceScanResult {
	if pkg.Module == nil || !pkg.Module.Main || !safeDeleteWithin(root, pkg.Module.Dir) {
		return safeDeleteReferenceScanResult{}
	}
	if pkg.TypesInfo == nil || pkg.Fset == nil {
		return safeDeleteReferenceScanResult{referenced: true, stop: true}
	}
	selection := safeDeletePackageReferenceTarget(pkg, targetPath, target)
	if selection.object == nil {
		return safeDeleteReferenceScanResult{}
	}
	for identifier, object := range pkg.TypesInfo.Uses {
		if object == selection.object && (identifier.Pos() < selection.declaration.Pos() || identifier.End() > selection.declaration.End()) {
			return safeDeleteReferenceScanResult{referenced: true, stop: true}
		}
	}
	return safeDeleteReferenceScanResult{}
}

type safeDeleteReferenceTarget struct {
	declaration ast.Decl
	object      types.Object
}

func safeDeletePackageReferenceTarget(pkg *packages.Package, targetPath string, target safeDeleteTarget) safeDeleteReferenceTarget {
	var selected safeDeleteReferenceTarget
	for _, file := range pkg.Syntax {
		if filepath.Clean(pkg.Fset.Position(file.Pos()).Filename) != filepath.Clean(targetPath) {
			continue
		}
		for _, declaration := range file.Decls {
			result := safeDeleteDefinitionInDecl(pkg, declaration, target)
			if result.matched && result.object != nil {
				selected = safeDeleteReferenceTarget{declaration: declaration, object: result.object}
			}
		}
	}
	return selected
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
		result := safeDeleteDefinitionInDecl(pkg, declaration, target)
		if result.matched && result.object != nil {
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

func safeDeletePackageProblemWithImportPolicy(pkgs []*packages.Package, root string, ignoreModuleDynamicImports bool) string {
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
		if !ignoreModuleDynamicImports {
			for _, imported := range pkg.Imports {
				if imported.PkgPath == "reflect" || imported.PkgPath == "plugin" || imported.PkgPath == "unsafe" {
					problem = "module imports reflect, plugin, or unsafe, so dynamic references cannot be excluded"
				}
			}
		}
		return true
	}, nil)
	return problem
}

func verifySafeDeleteTargetWithImportPolicy(ctx context.Context, root, targetPath string, target safeDeleteTarget, ignoreModuleDynamicImports bool) (string, error) {
	pkgs, err := loadMovePackages(ctx, root, nil)
	if err != nil {
		return "", err
	}
	if problem := safeDeletePackageProblemWithImportPolicy(pkgs, root, ignoreModuleDynamicImports); problem != "" {
		return "cannot prove safe deletion across active Go packages: " + problem, nil
	}
	definition := safeDeleteDefinition(pkgs, root, targetPath, target)
	if definition.initializerRefusal != "" {
		return definition.initializerRefusal, nil
	}
	if definition.object == nil {
		return "target is not present in an active, successfully type-checked package", nil
	}
	if target.kind == "function" {
		if _, ok := definition.object.(*types.Func); !ok {
			return "target does not resolve to a free function", nil
		}
	}
	if safeDeleteHasReference(pkgs, root, targetPath, target, definition.object, definition.packagePath) {
		return fmt.Sprintf("safe_delete found a compiler-resolved reference to %q", target.name), nil
	}
	if problem := safeDeleteImportProblem(pkgs, targetPath, target); problem != "" {
		return problem, nil
	}
	return "", nil
}
