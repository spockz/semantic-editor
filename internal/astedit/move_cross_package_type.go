// Package astedit isolates type-group movement because nominal type identity and method sets need stronger proofs than a free-function move.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

type crossPackageTypeMethod struct {
	declaration *ast.FuncDecl
	object      *types.Func
	pointer     bool
}

type crossPackageTypeGroup struct {
	packages           []*packages.Package
	sourcePackage      *packages.Package
	destinationPackage *packages.Package
	sourceFile         *ast.File
	destinationFile    *ast.File
	typeDeclaration    *ast.GenDecl
	typeSpec           *ast.TypeSpec
	typeObject         *types.TypeName
	typePosition       token.Position
	named              *types.Named
	methods            []crossPackageTypeMethod
	declarationStart   int
	declarationEnd     int
	declarationText    []byte
	memberSpans        [][2]int
	neededImports      *ast.File
}

func resolveCrossPackageTypeGroup(snapshot crossPackageMoveSnapshot, symbolQuery string, selected crossPackagePackages) (*crossPackageTypeGroup, bool, error) {
	typeDeclaration, typeSpec := findCrossPackageTypeSpec(selected.sourceFile, symbolQuery)
	if typeSpec == nil || typeSpec.Assign.IsValid() {
		return nil, false, nil
	}
	typeObject, named, resolveErr := resolveCrossPackageTypeDefinition(snapshot, symbolQuery, selected, typeDeclaration, typeSpec)
	if resolveErr != nil {
		return nil, true, resolveErr
	}
	group := &crossPackageTypeGroup{packages: selected.all, sourcePackage: selected.source, destinationPackage: selected.destination, sourceFile: selected.sourceFile, destinationFile: selected.destFile, typeDeclaration: typeDeclaration, typeSpec: typeSpec, typeObject: typeObject, typePosition: selected.source.Fset.Position(typeObject.Pos()), named: named}
	methods, methodErr := reconcileCrossPackageTypeMethods(snapshot, group)
	if methodErr != nil {
		return nil, true, methodErr
	}
	group.methods = methods
	if err := validateCrossPackageTypeReferences(snapshot, symbolQuery, group); err != nil {
		return nil, true, err
	}
	if err := setCrossPackageTypeSourceSpans(snapshot, selected.source, group); err != nil {
		return nil, true, err
	}
	imports, importErr := moveImportsForTypeGroup(group)
	if importErr != nil {
		return nil, true, importErr
	}
	group.neededImports = imports
	if err := rejectCrossPackageTypeDependencies(group); err != nil {
		return nil, true, err
	}
	return group, true, nil
}

func findCrossPackageTypeSpec(file *ast.File, name string) (*ast.GenDecl, *ast.TypeSpec) {
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, raw := range group.Specs {
			spec, ok := raw.(*ast.TypeSpec)
			if ok && spec.Name.Name == name {
				return group, spec
			}
		}
	}
	return nil, nil
}

func resolveCrossPackageTypeDefinition(snapshot crossPackageMoveSnapshot, name string, selected crossPackagePackages, declaration *ast.GenDecl, spec *ast.TypeSpec) (*types.TypeName, *types.Named, error) {
	if err := requireCrossPackageMovePackageInfo(selected.all); err != nil {
		return nil, nil, err
	}
	if len(declaration.Specs) != 1 || spec.Assign.IsValid() {
		return nil, nil, fmt.Errorf("cross-package type move requires one standalone defined type declaration")
	}
	if spec.TypeParams != nil && len(spec.TypeParams.List) != 0 {
		return nil, nil, fmt.Errorf("cross-package type move refuses generic types")
	}
	typeObject, ok := selected.source.TypesInfo.Defs[spec.Name].(*types.TypeName)
	if !ok || typeObject == nil || !typeObject.Exported() || typeObject.Pkg() == nil || typeObject.Pkg().Path() != selected.source.PkgPath || filepath.Clean(selected.source.Fset.Position(typeObject.Pos()).Filename) != snapshot.sourcePath {
		return nil, nil, fmt.Errorf("cross-package type move requires an exported defined type in the selected source file")
	}
	named, ok := typeObject.Type().(*types.Named)
	if !ok || named.TypeParams() != nil && named.TypeParams().Len() != 0 {
		return nil, nil, fmt.Errorf("cross-package type move requires a non-generic defined type")
	}
	if err := validateCrossPackageTypeShape(named); err != nil {
		return nil, nil, err
	}
	if selected.destination.Types.Scope().Lookup(name) != nil {
		return nil, nil, fmt.Errorf("destination already declares %q", name)
	}
	if err := validateCrossPackageTypeSource(snapshot, selected.sourceFile, selected.destFile, name); err != nil {
		return nil, nil, err
	}
	return typeObject, named, nil
}

func requireCrossPackageMovePackageInfo(packages []*packages.Package) error {
	for _, pkg := range packages {
		if pkg == nil || pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil || pkg.Module == nil || !pkg.Module.Main {
			return fmt.Errorf("cross-package type move cannot establish complete package information for every module package")
		}
	}
	return nil
}

func validateCrossPackageTypeShape(named *types.Named) error {
	switch underlying := named.Underlying().(type) {
	case *types.Basic:
		return nil
	case *types.Struct:
		for field := range underlying.Fields() {
			if field.Embedded() {
				return fmt.Errorf("cross-package type move refuses embedded fields")
			}
		}
		return nil
	default:
		return fmt.Errorf("cross-package type move supports only scalar and non-embedded struct types")
	}
}

func validateCrossPackageTypeReferences(snapshot crossPackageMoveSnapshot, symbolQuery string, group *crossPackageTypeGroup) error {
	if err := rejectCrossPackageAnonymousPrivateStructFields(group); err != nil {
		return err
	}
	if err := validateCrossPackageTestReferences(snapshot, symbolQuery, group.methods); err != nil {
		return err
	}
	if err := validateCrossPackageTypeReferenceIdentities(group); err != nil {
		return err
	}
	if err := validateCrossPackageModuleReferences(snapshot, symbolQuery, group); err != nil {
		return err
	}
	return rejectCrossPackageTypeExternalConsumers(group)
}

func validateCrossPackageTestReferences(snapshot crossPackageMoveSnapshot, typeName string, methods []crossPackageTypeMethod) error {
	names := make([]string, 1, 1+len(methods))
	names[0] = typeName
	for _, method := range methods {
		names = append(names, method.object.Name())
	}
	for path, content := range snapshot.files {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, name := range names {
			if containsMoveIdentifier(content, name) {
				return fmt.Errorf("cross-package type move refuses test-file reference to %q in %s", name, path)
			}
		}
	}
	return nil
}

func validateCrossPackageModuleReferences(snapshot crossPackageMoveSnapshot, typeName string, group *crossPackageTypeGroup) error {
	names := make([]string, 1, 1+len(group.methods))
	names[0] = typeName
	for _, method := range group.methods {
		names = append(names, method.object.Name())
	}
	for _, name := range names {
		for path, content := range snapshot.files {
			if err := validateMoveModuleSource(path, content, snapshot.sourcePath, snapshot.destinationPath, name); err != nil {
				return err
			}
		}
		if err := rejectUntypedMoveReferences(snapshot.files, group.packages, name); err != nil {
			return err
		}
	}
	return nil
}

func setCrossPackageTypeSourceSpans(snapshot crossPackageMoveSnapshot, source *packages.Package, group *crossPackageTypeGroup) error {
	start, end := moveDeclOffsets(source.Fset, group.typeDeclaration, group.sourceFile, snapshot.files[snapshot.sourcePath])
	if start < 0 || end < start || end > len(snapshot.files[snapshot.sourcePath]) {
		return fmt.Errorf("invalid source declaration bounds for type %q", group.typeObject.Name())
	}
	spans := [][2]int{{start, end}}
	for _, method := range group.methods {
		methodStart, methodEnd := moveDeclOffsets(source.Fset, method.declaration, group.sourceFile, snapshot.files[snapshot.sourcePath])
		if methodStart < 0 || methodEnd < methodStart || methodEnd > len(snapshot.files[snapshot.sourcePath]) {
			return fmt.Errorf("invalid source declaration bounds for method %q", method.object.Name())
		}
		spans = append(spans, [2]int{methodStart, methodEnd})
	}
	slices.SortFunc(spans, func(left, right [2]int) int { return left[0] - right[0] })
	for index := 1; index < len(spans); index++ {
		if spans[index-1][1] > spans[index][0] {
			return fmt.Errorf("cross-package type move has overlapping declaration comments")
		}
	}
	group.declarationStart = spans[0][0]
	group.declarationEnd = spans[len(spans)-1][1]
	group.memberSpans = spans
	var text bytes.Buffer
	for index, span := range spans {
		if index > 0 {
			text.WriteString("\n\n")
		}
		text.Write(snapshot.files[snapshot.sourcePath][span[0]:span[1]])
	}
	group.declarationText = text.Bytes()
	return nil
}

func rejectCrossPackageTypeExternalConsumers(group *crossPackageTypeGroup) error {
	for _, pkg := range group.packages {
		if pkg.Module == nil || pkg.Module.Main || pkg.TypesInfo == nil {
			continue
		}
		for ident := range pkg.TypesInfo.Uses {
			if sameCrossPackageTypeName(pkg, pkg.TypesInfo.Uses[ident], group.typeObject, group.typePosition) {
				return fmt.Errorf("cross-package type move refuses known external consumer in %s", pkg.PkgPath)
			}
		}
		for selector, selection := range pkg.TypesInfo.Selections {
			if selection != nil && crossPackageTypeGroupMethod(selection.Obj(), group) {
				return fmt.Errorf("cross-package type move refuses known external method consumer in %s", pkg.Fset.Position(selector.Pos()).Filename)
			}
		}
	}
	return nil
}

func validateCrossPackageTypeSource(snapshot crossPackageMoveSnapshot, sourceFile, destinationFile *ast.File, name string) error {
	for _, selected := range []struct {
		path string
		file *ast.File
	}{{snapshot.sourcePath, sourceFile}, {snapshot.destinationPath, destinationFile}} {
		if err := validateCrossPackageSelectedFile(selected.path, selected.file); err != nil {
			return err
		}
	}
	return rejectCrossPackageSiblingMethods(snapshot, name)
}

func validateCrossPackageSelectedFile(path string, file *ast.File) error {
	if err := validateCrossPackageImports(path, file); err != nil {
		return err
	}
	return rejectCrossPackageDirectives(path, file)
}

func validateCrossPackageImports(path string, file *ast.File) error {
	for _, spec := range file.Imports {
		importPath, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("decode import in %s: %w", path, err)
		}
		if importPath == "C" {
			return fmt.Errorf("cross-package move refuses cgo source or destination %s", path)
		}
	}
	return nil
}

func rejectCrossPackageDirectives(path string, file *ast.File) error {
	for _, comment := range file.Comments {
		for _, item := range comment.List {
			if strings.HasPrefix(item.Text, "//go:") {
				return fmt.Errorf("cross-package move refuses compiler directives in %s", path)
			}
		}
	}
	return nil
}

func rejectCrossPackageSiblingMethods(snapshot crossPackageMoveSnapshot, name string) error {
	for path, content := range snapshot.files {
		if filepath.Clean(filepath.Dir(path)) != filepath.Clean(filepath.Dir(snapshot.sourcePath)) || filepath.Clean(path) == snapshot.sourcePath {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return fmt.Errorf("cross-package type move cannot inspect sibling source %s: %w", path, err)
		}
		for _, declaration := range parsed.Decls {
			method, ok := declaration.(*ast.FuncDecl)
			if !ok || method.Recv == nil || len(method.Recv.List) != 1 {
				continue
			}
			base := receiverBaseName(method.Recv.List[0].Type)
			if base == name {
				return fmt.Errorf("cross-package type move refuses direct or build-variant method %q outside the selected source file", method.Name.Name)
			}
		}
	}
	return nil
}

func receiverBaseName(expression ast.Expr) string {
	switch receiver := expression.(type) {
	case *ast.Ident:
		return receiver.Name
	case *ast.StarExpr:
		ident, _ := receiver.X.(*ast.Ident)
		if ident != nil {
			return ident.Name
		}
	}
	return ""
}

func reconcileCrossPackageTypeMethods(snapshot crossPackageMoveSnapshot, group *crossPackageTypeGroup) ([]crossPackageTypeMethod, error) {
	declarations := map[*types.Func]crossPackageTypeMethod{}
	for _, declaration := range group.sourceFile.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil {
			continue
		}
		if len(method.Recv.List) != 1 || len(method.Recv.List[0].Names) > 1 {
			return nil, fmt.Errorf("cross-package type move refuses unsupported receiver declaration")
		}
		receiver := method.Recv.List[0].Type
		pointer := false
		if star, isPointerReceiver := receiver.(*ast.StarExpr); isPointerReceiver {
			pointer = true
			receiver = star.X
		}
		base, ok := receiver.(*ast.Ident)
		if !ok || base.Name != group.typeObject.Name() {
			continue
		}
		obj, ok := group.sourcePackage.TypesInfo.Defs[method.Name].(*types.Func)
		if !ok || obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != group.sourcePackage.PkgPath || filepath.Clean(group.sourcePackage.Fset.Position(obj.Pos()).Filename) != snapshot.sourcePath {
			return nil, fmt.Errorf("cross-package type move cannot identify direct method %q", method.Name.Name)
		}
		if !obj.Exported() {
			return nil, fmt.Errorf("cross-package type move refuses private method %q", obj.Name())
		}
		if _, exists := declarations[obj]; exists {
			return nil, fmt.Errorf("cross-package type move found ambiguous method identity %q", obj.Name())
		}
		declarations[obj] = crossPackageTypeMethod{declaration: method, object: obj, pointer: pointer}
	}
	if group.named.NumMethods() != len(declarations) {
		return nil, fmt.Errorf("cross-package type move cannot reconcile every direct method with source declarations")
	}
	methods := make([]crossPackageTypeMethod, 0, len(declarations))
	for method := range group.named.Methods() {
		found, ok := declarations[method]
		if !ok {
			return nil, fmt.Errorf("cross-package type move cannot reconcile method %q with the selected source file", method.Name())
		}
		methods = append(methods, found)
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("cross-package type move requires at least one direct method")
	}
	slices.SortFunc(methods, func(left, right crossPackageTypeMethod) int {
		return group.sourcePackage.Fset.Position(left.declaration.Pos()).Offset - group.sourcePackage.Fset.Position(right.declaration.Pos()).Offset
	})
	return methods, nil
}

func rejectCrossPackageTypeDependencies(group *crossPackageTypeGroup) error {
	allowed := map[types.Object]bool{group.typeObject: true}
	for _, method := range group.methods {
		allowed[method.object] = true
	}
	declarations := make([]ast.Node, 1, 1+len(group.methods))
	declarations[0] = group.typeDeclaration
	for _, method := range group.methods {
		declarations = append(declarations, method.declaration)
	}
	for _, declaration := range declarations {
		var dependency string
		ast.Inspect(declaration, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			object := group.sourcePackage.TypesInfo.Uses[ident]
			if object == nil || object.Pkg() == nil || object.Pkg().Path() != group.sourcePackage.PkgPath || object.Parent() != group.sourcePackage.Types.Scope() || allowed[object] {
				return true
			}
			if _, ok := object.(*types.PkgName); ok {
				return true
			}
			dependency = object.Name()
			return false
		})
		if dependency != "" {
			return fmt.Errorf("cross-package type move refuses source-package dependency %q", dependency)
		}
	}
	return nil
}

func rejectCrossPackageAnonymousPrivateStructFields(group *crossPackageTypeGroup) error {
	declarations := make([]ast.Node, 1, 1+len(group.methods))
	declarations[0] = group.typeDeclaration
	for _, method := range group.methods {
		declarations = append(declarations, method.declaration)
	}
	for _, declaration := range declarations {
		var found bool
		ast.Inspect(declaration, func(node ast.Node) bool {
			structure, ok := node.(*ast.StructType)
			if !ok || structure == group.typeSpec.Type {
				return true
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				for _, name := range field.Names {
					if !ast.IsExported(name.Name) {
						found = true
						return false
					}
				}
			}
			return !found
		})
		if found {
			return fmt.Errorf("cross-package type move refuses anonymous struct with unexported fields")
		}
	}
	return nil
}

func moveImportsForTypeGroup(group *crossPackageTypeGroup) (*ast.File, error) {
	used := map[string]bool{}
	declarations := make([]ast.Node, 1, 1+len(group.methods))
	declarations[0] = group.typeDeclaration
	for _, method := range group.methods {
		declarations = append(declarations, method.declaration)
	}
	for _, declaration := range declarations {
		ast.Inspect(declaration, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok {
				if ident, ok := selector.X.(*ast.Ident); ok {
					used[ident.Name] = true
				}
			}
			return true
		})
	}
	imports := &ast.File{}
	for _, spec := range group.sourceFile.Imports {
		path, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("decode source import: %w", err)
		}
		if spec.Name != nil && spec.Name.Name == "." {
			return nil, fmt.Errorf("cross-package type move refuses a dot import")
		}
		if spec.Name != nil && spec.Name.Name == "_" {
			continue
		}
		var local string
		var alias *ast.Ident
		if spec.Name != nil {
			local, alias = spec.Name.Name, ast.NewIdent(spec.Name.Name)
		} else if imported, ok := group.sourcePackage.TypesInfo.Implicits[spec].(*types.PkgName); ok {
			local = imported.Imported().Name()
		} else {
			return nil, fmt.Errorf("cannot resolve source import name for %q", path)
		}
		if used[local] {
			imports.Imports = append(imports.Imports, &ast.ImportSpec{Name: alias, Path: &ast.BasicLit{Kind: token.STRING, Value: strconvQuote(path)}})
		}
	}
	return imports, nil
}

type crossPackageTypeMarkerKind uint8

const (
	crossPackageTypeReference crossPackageTypeMarkerKind = iota + 1
	crossPackageTypeImport
	crossPackageMethodSelection
)

type crossPackageTypeMarker struct {
	sentinel  string
	name      string
	kind      crossPackageTypeMarkerKind
	method    *types.Func
	selection *types.Selection
	offset    int
}

type crossPackageTypeReferenceKey struct {
	packageID string
	filePath  string
}

type crossPackageTypeMovePlan struct {
	changes  map[string][]byte
	markers  map[crossPackageTypeReferenceKey][]crossPackageTypeMarker
	sequence int
}

func buildCrossPackageTypeMoveChanges(ctx context.Context, snapshot crossPackageMoveSnapshot, group *crossPackageTypeGroup, opts MoveConstructOptions) (crossPackageMoveChangeSet, error) {
	plan := crossPackageTypeMovePlan{changes: map[string][]byte{}, markers: map[crossPackageTypeReferenceKey][]crossPackageTypeMarker{}}
	for _, pkg := range group.packages {
		if !isMoveReferencePackage(pkg) {
			continue
		}
		for _, file := range pkg.Syntax {
			path := filepath.Clean(pkg.Fset.Position(file.Pos()).Filename)
			if !moveReferenceFileSelected(pkg, path, moveReferenceContext{sourcePath: snapshot.sourcePath, destinationPath: snapshot.destinationPath}) {
				continue
			}
			filePlan, err := planCrossPackageTypeReferenceFile(pkg, file, path, snapshot, group, &plan.sequence)
			if err != nil {
				return crossPackageMoveChangeSet{}, err
			}
			if filePlan.changed {
				key := crossPackageTypeReferenceKey{packageID: pkg.ID, filePath: path}
				plan.markers[key] = append(plan.markers[key], filePlan.markers...)
				plan.changes[path] = filePlan.content
			}
		}
	}
	if _, exists := plan.changes[snapshot.sourcePath]; !exists {
		return crossPackageMoveChangeSet{}, fmt.Errorf("cross-package type move did not stage its source file")
	}
	destinationBase := snapshot.files[snapshot.destinationPath]
	if rewritten, exists := plan.changes[snapshot.destinationPath]; exists {
		destinationBase = rewritten
	}
	sourceKey := crossPackageTypeReferenceKey{packageID: group.sourcePackage.ID, filePath: snapshot.sourcePath}
	markedDeclaration, groupMarkers, remainingMarkers, err := applyMovedTypeGroupMarkers(group, snapshot.files[snapshot.sourcePath], plan.markers[sourceKey])
	if err != nil {
		return crossPackageMoveChangeSet{}, err
	}
	plan.markers[sourceKey] = remainingMarkers
	if len(groupMarkers) == 0 {
		return crossPackageMoveChangeSet{}, fmt.Errorf("cross-package type move could not capture moved declaration references")
	}
	destination, err := insertMovedDeclaration(snapshot.destinationPath, destinationBase, markedDeclaration, group.neededImports, opts)
	if err != nil {
		return crossPackageMoveChangeSet{}, err
	}
	plan.changes[snapshot.destinationPath] = destination
	destinationKey := crossPackageTypeReferenceKey{packageID: group.destinationPackage.ID, filePath: snapshot.destinationPath}
	plan.markers[destinationKey] = append(plan.markers[destinationKey], groupMarkers...)
	if err := expandCrossPackageTypeMarkers(plan.changes, plan.markers); err != nil {
		return crossPackageMoveChangeSet{}, err
	}
	if err := validateCrossPackageTypeMoveOverlay(ctx, snapshot, group, plan.changes, plan.markers); err != nil {
		return crossPackageMoveChangeSet{}, err
	}
	return crossPackageMoveChangeSet{contents: plan.changes}, nil
}

type crossPackageTypeFilePlan struct {
	content []byte
	markers []crossPackageTypeMarker
	changed bool
}

func planCrossPackageTypeReferenceFile(pkg *packages.Package, file *ast.File, path string, snapshot crossPackageMoveSnapshot, group *crossPackageTypeGroup, sequence *int) (crossPackageTypeFilePlan, error) {
	removeGroup := filepath.Clean(path) == filepath.Clean(snapshot.sourcePath)
	moveGroupRanges := crossPackageTypeGroupRanges(group, pkg, path)
	markers, changed := markCrossPackageTypeReferences(pkg, file, group, moveGroupRanges, sequence)
	methodMarkers, methodsChanged := markCrossPackageMethodSelections(pkg, file, group, moveGroupRanges, sequence)
	markers = append(markers, methodMarkers...)
	changed = changed || methodsChanged
	if removeGroup {
		removeCrossPackageTypeDeclarations(file, group, pkg.Fset)
		changed = true
	}
	if !changed {
		return crossPackageTypeFilePlan{}, nil
	}
	if err := prepareCrossPackageTypeReferenceImports(pkg, file, path, markers, group); err != nil {
		return crossPackageTypeFilePlan{}, err
	}
	content, err := formatMoveReferenceFile(pkg, file, path)
	if err != nil {
		return crossPackageTypeFilePlan{}, err
	}
	return crossPackageTypeFilePlan{content: content, markers: markers, changed: true}, nil
}

func markCrossPackageTypeReferences(pkg *packages.Package, file *ast.File, group *crossPackageTypeGroup, moveGroupRanges [][2]int, sequence *int) ([]crossPackageTypeMarker, bool) {
	var markers []crossPackageTypeMarker
	changed := false
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		selector, isSelector := cursor.Node().(*ast.SelectorExpr)
		if isSelector && isCrossPackageTypeReference(pkg, selector.Sel, group) {
			marker := nextCrossPackageTypeMarker(file, sequence)
			markerRecord := crossPackageTypeMarker{sentinel: marker, name: group.typeObject.Name(), kind: crossPackageTypeReference, offset: pkg.Fset.Position(selector.Sel.Pos()).Offset}
			if inRanges(pkg.Fset.Position(selector.Pos()).Offset, moveGroupRanges) {
				markers = append(markers, markerRecord)
				return true
			}
			if pkg.PkgPath == group.destinationPackage.PkgPath {
				cursor.Replace(ast.NewIdent(marker))
			} else {
				alias := moveImportAlias(file, group.destinationPackage.PkgPath, group.destinationPackage.Name)
				importMarker := nextCrossPackageTypeMarker(file, sequence)
				markers = append(markers, crossPackageTypeMarker{sentinel: importMarker, name: alias, kind: crossPackageTypeImport, offset: pkg.Fset.Position(selector.X.Pos()).Offset}, markerRecord)
				selector.X = ast.NewIdent(importMarker)
				selector.Sel.Name = marker
			}
			changed = true
			return false
		}
		ident, ok := cursor.Node().(*ast.Ident)
		if !ok || !isCrossPackageTypeReference(pkg, ident, group) {
			return true
		}
		if parent, isSelector := cursor.Parent().(*ast.SelectorExpr); isSelector && parent.Sel == ident {
			return true
		}
		marker := nextCrossPackageTypeMarker(file, sequence)
		markerRecord := crossPackageTypeMarker{sentinel: marker, name: group.typeObject.Name(), kind: crossPackageTypeReference, offset: pkg.Fset.Position(ident.Pos()).Offset}
		if inRanges(pkg.Fset.Position(ident.Pos()).Offset, moveGroupRanges) {
			markers = append(markers, markerRecord)
			return true
		}
		if pkg.PkgPath == group.destinationPackage.PkgPath {
			cursor.Replace(ast.NewIdent(marker))
		} else {
			alias := moveImportAlias(file, group.destinationPackage.PkgPath, group.destinationPackage.Name)
			importMarker := nextCrossPackageTypeMarker(file, sequence)
			markers = append(markers, crossPackageTypeMarker{sentinel: importMarker, name: alias, kind: crossPackageTypeImport, offset: pkg.Fset.Position(ident.Pos()).Offset}, markerRecord)
			cursor.Replace(&ast.SelectorExpr{X: ast.NewIdent(importMarker), Sel: ast.NewIdent(marker)})
		}
		changed = true
		return false
	}, nil)
	return markers, changed
}

func markCrossPackageMethodSelections(pkg *packages.Package, file *ast.File, group *crossPackageTypeGroup, moveGroupRanges [][2]int, sequence *int) ([]crossPackageTypeMarker, bool) {
	var markers []crossPackageTypeMarker
	changed := false
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		selector, ok := cursor.Node().(*ast.SelectorExpr)
		if !ok {
			return true
		}
		selection := pkg.TypesInfo.Selections[selector]
		if selection == nil || !crossPackageTypeGroupMethod(selection.Obj(), group) {
			return true
		}
		marker := nextCrossPackageTypeMarker(file, sequence)
		markerRecord := crossPackageTypeMarker{sentinel: marker, name: selector.Sel.Name, kind: crossPackageMethodSelection, method: selection.Obj().(*types.Func), selection: selection, offset: pkg.Fset.Position(selector.Sel.Pos()).Offset}
		markers = append(markers, markerRecord)
		if inRanges(pkg.Fset.Position(selector.Pos()).Offset, moveGroupRanges) {
			return true
		}
		selector.Sel.Name = marker
		changed = true
		return true
	}, nil)
	return markers, changed
}

func prepareCrossPackageTypeReferenceImports(pkg *packages.Package, file *ast.File, path string, markers []crossPackageTypeMarker, group *crossPackageTypeGroup) error {
	if pkg.PkgPath != group.destinationPackage.PkgPath {
		edits := moveReferenceEdits{changed: true}
		for _, marker := range markers {
			if marker.kind == crossPackageTypeImport {
				edits.markers = append(edits.markers, moveReferenceMarker{replacement: marker.name, kind: moveReferenceImport})
				if edits.alias == "" {
					edits.alias = marker.name
				}
			}
		}
		request := moveReferenceContext{destinationImport: group.destinationPackage.PkgPath, destinationName: group.destinationPackage.Name}
		if err := prepareMoveReferenceImports(pkg, file, path, edits, request); err != nil {
			return err
		}
	} else if err := removeUnusedMoveImports(file, pkg.TypesInfo, group.destinationPackage.PkgPath, group.destinationPackage.Name, nil); err != nil {
		return err
	}
	return nil
}

func isCrossPackageTypeReference(pkg *packages.Package, ident *ast.Ident, group *crossPackageTypeGroup) bool {
	object := pkg.TypesInfo.Uses[ident]
	return sameCrossPackageTypeName(pkg, object, group.typeObject, group.typePosition)
}

func sameCrossPackageTypeName(pkg *packages.Package, left, right types.Object, rightPosition token.Position) bool {
	if left != nil && left == right {
		return true
	}
	if left == nil || right == nil || left.Name() != right.Name() || left.Pkg() == nil || right.Pkg() == nil || left.Pkg().Path() != right.Pkg().Path() {
		return false
	}
	leftPosition := pkg.Fset.Position(left.Pos())
	return leftPosition.IsValid() && rightPosition.IsValid() && filepath.Clean(leftPosition.Filename) == filepath.Clean(rightPosition.Filename) && leftPosition.Offset == rightPosition.Offset
}

func crossPackageTypeGroupMethod(object types.Object, group *crossPackageTypeGroup) bool {
	method, ok := object.(*types.Func)
	if !ok {
		return false
	}
	for _, candidate := range group.methods {
		position := group.sourcePackage.Fset.Position(candidate.object.Pos())
		if sameCrossPackageTypeName(group.sourcePackage, method, candidate.object, position) {
			return true
		}
	}
	return false
}

func validateCrossPackageTypeReferenceIdentities(group *crossPackageTypeGroup) error {
	for _, pkg := range group.packages {
		for _, file := range pkg.Syntax {
			filePath := filepath.Clean(pkg.Fset.Position(file.Pos()).Filename)
			if err := validateCrossPackageTypeNameReferences(pkg, file, filePath, group); err != nil {
				return err
			}
			if err := validateCrossPackageMethodIdentities(pkg, filePath, group); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCrossPackageTypeNameReferences(pkg *packages.Package, file *ast.File, filePath string, group *crossPackageTypeGroup) error {
	var err error
	ast.Inspect(file, func(node ast.Node) bool {
		if err != nil {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok || ident.Name != group.typeObject.Name() {
			return true
		}
		object, ok := pkg.TypesInfo.Uses[ident].(*types.TypeName)
		if !ok || object.Pkg() == nil || object.Pkg().Path() != group.sourcePackage.PkgPath {
			return true
		}
		if !sameCrossPackageTypeName(pkg, object, group.typeObject, group.typePosition) {
			err = fmt.Errorf("cross-package type move cannot map type reference in %s", filePath)
			return false
		}
		return true
	})
	return err
}

func validateCrossPackageMethodIdentities(pkg *packages.Package, filePath string, group *crossPackageTypeGroup) error {
	for _, selection := range pkg.TypesInfo.Selections {
		method, ok := selection.Obj().(*types.Func)
		if !ok || !crossPackageTypeReceiverMethod(method, group) {
			continue
		}
		if !crossPackageTypeGroupMethod(method, group) {
			return fmt.Errorf("cross-package type move cannot map method selection %q in %s", method.Name(), filePath)
		}
	}
	return nil
}

func crossPackageTypeReceiverMethod(method *types.Func, group *crossPackageTypeGroup) bool {
	if method == nil || method.Pkg() == nil || method.Pkg().Path() != group.sourcePackage.PkgPath {
		return false
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiver := signature.Recv().Type()
	if pointer, isPointerReceiver := receiver.(*types.Pointer); isPointerReceiver {
		receiver = pointer.Elem()
	}
	named, ok := receiver.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == group.sourcePackage.PkgPath && named.Obj().Name() == group.typeObject.Name()
}

func nextCrossPackageTypeMarker(file *ast.File, sequence *int) string {
	for {
		candidate := fmt.Sprintf("semeditMovedType%d", *sequence)
		*sequence++
		used := false
		ast.Inspect(file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && ident.Name == candidate {
				used = true
				return false
			}
			return true
		})
		if !used {
			return candidate
		}
	}
}

func crossPackageTypeGroupRanges(group *crossPackageTypeGroup, pkg *packages.Package, path string) [][2]int {
	if filepath.Clean(path) != filepath.Clean(pkg.Fset.Position(group.typeSpec.Pos()).Filename) {
		return nil
	}
	ranges := make([][2]int, 0, len(group.methods)+1)
	return append(ranges, group.memberSpans...)
}

func inRanges(offset int, ranges [][2]int) bool {
	for _, span := range ranges {
		if offset >= span[0] && offset < span[1] {
			return true
		}
	}
	return false
}

func removeCrossPackageTypeDeclarations(file *ast.File, group *crossPackageTypeGroup, fset *token.FileSet) {
	removedMethods := map[*ast.FuncDecl]bool{}
	for _, method := range group.methods {
		removedMethods[method.declaration] = true
	}
	declarations := file.Decls[:0]
	for _, declaration := range file.Decls {
		if declaration == group.typeDeclaration || removedMethods[funcDecl(declaration)] {
			continue
		}
		declarations = append(declarations, declaration)
	}
	file.Decls = declarations
	spans := group.memberSpans
	comments := file.Comments[:0]
	for _, comment := range file.Comments {
		start, end := fset.Position(comment.Pos()).Offset, fset.Position(comment.End()).Offset
		if inRanges(start, spans) || inRanges(end-1, spans) {
			continue
		}
		comments = append(comments, comment)
	}
	file.Comments = comments
}

func funcDecl(declaration ast.Decl) *ast.FuncDecl {
	method, _ := declaration.(*ast.FuncDecl)
	return method
}

func applyMovedTypeGroupMarkers(group *crossPackageTypeGroup, source []byte, markers []crossPackageTypeMarker) ([]byte, []crossPackageTypeMarker, []crossPackageTypeMarker, error) {
	moved, remaining := partitionCrossPackageTypeMarkers(group.memberSpans, markers)
	if len(moved) == 0 {
		return group.declarationText, moved, remaining, nil
	}
	marked := append([]byte(nil), group.declarationText...)
	type replacement struct {
		start int
		end   int
		text  []byte
	}
	replacements := make([]replacement, 0, len(moved))
	for _, marker := range moved {
		item, err := crossPackageMovedMarkerReplacement(group.memberSpans, source, marker)
		if err != nil {
			return nil, nil, nil, err
		}
		replacements = append(replacements, replacement{start: item[0], end: item[1], text: []byte(marker.sentinel)})
	}
	slices.SortFunc(replacements, func(left, right replacement) int { return right.start - left.start })
	for _, edit := range replacements {
		if edit.start < 0 || edit.end > len(marked) || edit.end < edit.start {
			return nil, nil, nil, fmt.Errorf("moved type proof marker has invalid source range")
		}
		marked = append(marked[:edit.start], append(edit.text, marked[edit.end:]...)...)
	}
	return marked, moved, remaining, nil
}

func partitionCrossPackageTypeMarkers(spans [][2]int, markers []crossPackageTypeMarker) ([]crossPackageTypeMarker, []crossPackageTypeMarker) {
	var moved, remaining []crossPackageTypeMarker
	for _, marker := range markers {
		if crossPackageOffsetInSpans(marker.offset, spans) {
			moved = append(moved, marker)
		} else {
			remaining = append(remaining, marker)
		}
	}
	return moved, remaining
}

func crossPackageOffsetInSpans(offset int, spans [][2]int) bool {
	for _, span := range spans {
		if offset >= span[0] && offset < span[1] {
			return true
		}
	}
	return false
}

func crossPackageMovedMarkerReplacement(spans [][2]int, source []byte, marker crossPackageTypeMarker) ([2]int, error) {
	prefix := 0
	for index, span := range spans {
		if marker.offset >= span[0] && marker.offset < span[1] {
			localOffset := marker.offset - span[0]
			end := localOffset + len(marker.name)
			original := source[span[0]:span[1]]
			if end > len(original) || string(original[localOffset:end]) != marker.name {
				return [2]int{}, fmt.Errorf("moved type proof marker does not match source identifier %q", marker.name)
			}
			return [2]int{prefix + localOffset, prefix + end}, nil
		}
		prefix += span[1] - span[0]
		if index+1 < len(spans) {
			prefix += 2
		}
	}
	return [2]int{}, fmt.Errorf("cannot map moved type proof marker to its declaration")
}

func expandCrossPackageTypeMarkers(changes map[string][]byte, markers map[crossPackageTypeReferenceKey][]crossPackageTypeMarker) error {
	for key, fileMarkers := range markers {
		content, exists := changes[key.filePath]
		if !exists {
			continue
		}
		type replacement struct {
			marker crossPackageTypeMarker
			offset int
		}
		items := make([]replacement, 0, len(fileMarkers))
		for _, marker := range fileMarkers {
			offset := bytes.Index(content, []byte(marker.sentinel))
			if offset < 0 || bytes.Contains(content[offset+len(marker.sentinel):], []byte(marker.sentinel)) {
				return fmt.Errorf("locate unique cross-package type proof marker in %s", key.filePath)
			}
			items = append(items, replacement{marker: marker, offset: offset})
		}
		slices.SortFunc(items, func(left, right replacement) int { return right.offset - left.offset })
		for _, item := range items {
			start, end := item.offset, item.offset+len(item.marker.sentinel)
			content = append(content[:start], append([]byte(item.marker.name), content[end:]...)...)
		}
		for index := range items {
			finalOffset := items[index].offset
			for _, earlier := range items {
				if earlier.offset < items[index].offset {
					finalOffset += len(earlier.marker.name) - len(earlier.marker.sentinel)
				}
			}
			items[index].marker.offset = finalOffset
		}
		fileMarkers = fileMarkers[:0]
		for _, item := range items {
			fileMarkers = append(fileMarkers, item.marker)
		}
		markers[key] = fileMarkers
		changes[key.filePath] = content
	}
	for path, content := range changes {
		if strings.Contains(string(content), "semeditMovedType") {
			return fmt.Errorf("cross-package type move could not expand all proof markers in %s", path)
		}
	}
	return nil
}

func validateCrossPackageTypeMoveOverlay(ctx context.Context, snapshot crossPackageMoveSnapshot, group *crossPackageTypeGroup, changes map[string][]byte, markers map[crossPackageTypeReferenceKey][]crossPackageTypeMarker) error {
	for path := range changes {
		if _, exists := snapshot.files[path]; !exists {
			return fmt.Errorf("type move output is outside the preflight module snapshot: %s", path)
		}
	}
	if err := validateMoveSourceAndDestination(changes, snapshot.sourcePath, snapshot.destinationPath); err != nil {
		return err
	}
	overlay, err := typecheckMoveOverlay(ctx, snapshot.moduleRoot, changes)
	if err != nil {
		return err
	}
	destination := packageForMoveFile(overlay, snapshot.destinationPath)
	if destination == nil || destination.PkgPath != group.destinationPackage.PkgPath {
		return fmt.Errorf("cross-package type move cannot identify destination package in the candidate overlay")
	}
	movedType, movedNamed, movedMethods, err := resolveCrossPackageMovedDeclarations(destination, snapshot.destinationPath, group)
	if err != nil {
		return err
	}
	if err := validateCrossPackageMovedDeclarationBindings(group, destination, movedType, movedMethods); err != nil {
		return err
	}
	if err := validateCrossPackageMovedMethodSignatures(group, movedNamed, movedMethods); err != nil {
		return err
	}
	if err := validateCrossPackageInitializationGraph(group.packages, overlay); err != nil {
		return err
	}
	if err := validateCrossPackageTypeMarkers(overlay, destination, group, movedType, movedNamed, movedMethods, markers); err != nil {
		return err
	}
	return validateCrossPackageInterfaceRelations(group, movedNamed, group.packages, overlay)
}

func resolveCrossPackageMovedDeclarations(destination *packages.Package, destinationPath string, group *crossPackageTypeGroup) (*types.TypeName, *types.Named, map[string]*types.Func, error) {
	movedType, ok := destination.Types.Scope().Lookup(group.typeObject.Name()).(*types.TypeName)
	if !ok || movedType.Pkg() == nil || movedType.Pkg().Path() != destination.PkgPath || filepath.Clean(destination.Fset.Position(movedType.Pos()).Filename) != destinationPath {
		return nil, nil, nil, fmt.Errorf("cross-package type move cannot identify moved type %q in destination package", group.typeObject.Name())
	}
	movedNamed, ok := movedType.Type().(*types.Named)
	if !ok || movedNamed.NumMethods() != len(group.methods) {
		return nil, nil, nil, fmt.Errorf("cross-package type move cannot identify the complete moved method group")
	}
	movedMethods := make(map[string]*types.Func, len(group.methods))
	for method := range movedNamed.Methods() {
		if !method.Exported() || method.Pkg() == nil || method.Pkg().Path() != destination.PkgPath || filepath.Clean(destination.Fset.Position(method.Pos()).Filename) != destinationPath {
			return nil, nil, nil, fmt.Errorf("cross-package type move cannot identify moved method %q in destination package", method.Name())
		}
		movedMethods[method.Name()] = method
	}
	return movedType, movedNamed, movedMethods, nil
}

func validateCrossPackageMovedMethodSignatures(group *crossPackageTypeGroup, movedNamed *types.Named, movedMethods map[string]*types.Func) error {
	for _, method := range group.methods {
		moved := movedMethods[method.object.Name()]
		if moved == nil {
			return fmt.Errorf("cross-package type move cannot map method %q to the destination type", method.object.Name())
		}
		oldSig, oldOK := method.object.Type().(*types.Signature)
		newSig, newOK := moved.Type().(*types.Signature)
		if !oldOK || !newOK || oldSig.Recv() == nil || newSig.Recv() == nil || isPointerType(oldSig.Recv().Type()) != method.pointer || isPointerType(newSig.Recv().Type()) != method.pointer {
			return fmt.Errorf("cross-package type move changed receiver form for method %q", method.object.Name())
		}
		if !crossPackageMoveTypeEqual(oldSig, newSig, group.named, movedNamed) {
			return fmt.Errorf("cross-package type move changed method signature for %q", method.object.Name())
		}
	}
	return nil
}

func validateCrossPackageTypeMarkers(overlay []*packages.Package, destination *packages.Package, group *crossPackageTypeGroup, movedType *types.TypeName, movedNamed *types.Named, movedMethods map[string]*types.Func, markers map[crossPackageTypeReferenceKey][]crossPackageTypeMarker) error {
	for key, expected := range markers {
		pkg := packageByID(overlay, key.packageID)
		if pkg == nil {
			return fmt.Errorf("cross-package type move cannot validate references in package %s", key.packageID)
		}
		file := syntaxFileForReference(pkg, key.filePath)
		if file == nil {
			return fmt.Errorf("cross-package type move cannot validate references in %s", key.filePath)
		}
		for _, marker := range expected {
			ident := identifierAtOffset(pkg, file, marker.offset, marker.name)
			if ident == nil {
				return fmt.Errorf("cross-package type move cannot locate proof reference %q in %s", marker.name, key.filePath)
			}
			switch marker.kind {
			case crossPackageTypeReference:
				if pkg.TypesInfo.Uses[ident] != movedType {
					return fmt.Errorf("cross-package type move refuses reference %q in %s because it resolves to a different declaration", marker.name, key.filePath)
				}
			case crossPackageTypeImport:
				packageName, valid := pkg.TypesInfo.Uses[ident].(*types.PkgName)
				if !valid || packageName.Imported().Path() != destination.PkgPath {
					return fmt.Errorf("cross-package type move refuses destination import in %s because it resolves to a different package", key.filePath)
				}
			case crossPackageMethodSelection:
				if err := validateCrossPackageMethodSelection(pkg, file, marker, movedMethods, group.named, movedNamed); err != nil {
					return err
				}
			default:
				return fmt.Errorf("cross-package type move encountered an unsupported proof marker")
			}
		}
	}
	return nil
}

func validateCrossPackageInitializationGraph(beforePackages []*packages.Package, after []*packages.Package) error {
	afterByID := make(map[string]*packages.Package, len(after))
	for _, pkg := range after {
		afterByID[pkg.ID] = pkg
	}
	changed, err := crossPackageImportsChanged(beforePackages, afterByID)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return validateCrossPackagePackageInitialization(beforePackages, after, map[*packages.Package]bool{})
}

func crossPackageImportsChanged(beforePackages []*packages.Package, afterByID map[string]*packages.Package) (bool, error) {
	changed := false
	for _, before := range beforePackages {
		if before.Module == nil || !before.Module.Main {
			continue
		}
		current := afterByID[before.ID]
		if current == nil {
			return false, fmt.Errorf("cross-package move cannot pair package %s for initialization proof", before.ID)
		}
		if !sameCrossPackageImportPaths(before, current) {
			changed = true
		}
	}
	return changed, nil
}

func validateCrossPackagePackageInitialization(before, after []*packages.Package, seen map[*packages.Package]bool) error {
	for _, pkg := range before {
		if err := inspectCrossPackageInitializationDependencies(pkg, seen); err != nil {
			return err
		}
	}
	for _, pkg := range after {
		if pkg.Module != nil && pkg.Module.Main {
			if err := inspectCrossPackageInitializationDependencies(pkg, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func inspectCrossPackageInitializationDependencies(pkg *packages.Package, seen map[*packages.Package]bool) error {
	if pkg == nil || seen[pkg] {
		return nil
	}
	seen[pkg] = true
	if len(pkg.Syntax) == 0 || pkg.TypesInfo == nil {
		return fmt.Errorf("cross-package move cannot prove initialization safety for dependency %s", pkg.PkgPath)
	}
	if err := validateCrossPackagePackageInitializers(pkg); err != nil {
		return err
	}
	for _, dependency := range pkg.Imports {
		if err := inspectCrossPackageInitializationDependencies(dependency, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateCrossPackagePackageInitializers(pkg *packages.Package) error {
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			if err := validateCrossPackageInitializerDeclaration(pkg, declaration); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCrossPackageInitializerDeclaration(pkg *packages.Package, declaration ast.Decl) error {
	switch declaration := declaration.(type) {
	case *ast.FuncDecl:
		if declaration.Recv == nil && declaration.Name.Name == "init" {
			return fmt.Errorf("cross-package move refuses package initialization function in %s", pkg.PkgPath)
		}
	case *ast.GenDecl:
		return validateCrossPackageVariableInitializers(pkg, declaration)
	}
	return nil
}

func validateCrossPackageVariableInitializers(pkg *packages.Package, declaration *ast.GenDecl) error {
	if declaration.Tok != token.VAR {
		return nil
	}
	for _, spec := range declaration.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, expression := range valueSpec.Values {
			if !pureCrossPackagePackageInitializer(expression, pkg.TypesInfo) {
				return fmt.Errorf("cross-package move refuses dynamic package initialization in %s", pkg.PkgPath)
			}
		}
	}
	return nil
}

func sameCrossPackageImportPaths(left, right *packages.Package) bool {
	leftPaths := make([]string, 0, len(left.Imports))
	for _, dependency := range left.Imports {
		leftPaths = append(leftPaths, dependency.PkgPath)
	}
	rightPaths := make([]string, 0, len(right.Imports))
	for _, dependency := range right.Imports {
		rightPaths = append(rightPaths, dependency.PkgPath)
	}
	slices.Sort(leftPaths)
	slices.Sort(rightPaths)
	return slices.Equal(leftPaths, rightPaths)
}

func pureCrossPackagePackageInitializer(expression ast.Expr, info *types.Info) bool {
	if info == nil {
		return false
	}
	if value, ok := info.Types[expression]; ok && value.Value != nil {
		return true
	}
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name == "nil"
	case *ast.SelectorExpr:
		selection := info.Selections[expression]
		receiver, hasType := info.Types[expression.X]
		return selection != nil && selection.Kind() == types.MethodExpr && hasType && receiver.IsType()
	default:
		return false
	}
}

func validateCrossPackageMovedDeclarationBindings(group *crossPackageTypeGroup, destination *packages.Package, movedType *types.TypeName, movedMethods map[string]*types.Func) error {
	objects := map[types.Object]types.Object{group.typeObject: movedType}
	for _, method := range group.methods {
		objects[method.object] = movedMethods[method.object.Name()]
	}
	newNamed, newOK := movedType.Type().(*types.Named)
	if !newOK {
		return fmt.Errorf("cross-package type move changed the defined type shape")
	}
	if err := pairCrossPackageMovedTypeFields(group.named, newNamed, objects); err != nil {
		return err
	}
	pairs, err := crossPackageMovedDeclarationPairs(group, destination, movedType, movedMethods)
	if err != nil {
		return err
	}
	identifiers, err := pairCrossPackageMovedIdentifiers(group, destination, objects, pairs)
	if err != nil {
		return err
	}
	return validateCrossPackageMovedIdentifierBindings(group, destination, objects, newNamed, identifiers)
}

type crossPackageMovedDeclarationPair struct {
	old  ast.Node
	new  ast.Node
	name string
}

type crossPackageMovedIdentifierPair struct {
	old, current *ast.Ident
	declaration  string
}

func pairCrossPackageMovedTypeFields(oldNamed, newNamed *types.Named, objects map[types.Object]types.Object) error {
	oldStruct, oldIsStruct := oldNamed.Underlying().(*types.Struct)
	newStruct, newIsStruct := newNamed.Underlying().(*types.Struct)
	if oldIsStruct != newIsStruct {
		return fmt.Errorf("cross-package type move changed the defined type shape")
	}
	if !oldIsStruct {
		return nil
	}
	if oldStruct.NumFields() != newStruct.NumFields() {
		return fmt.Errorf("cross-package type move changed struct field identity")
	}
	for index := range oldStruct.NumFields() {
		oldField, newField := oldStruct.Field(index), newStruct.Field(index)
		if oldField.Name() != newField.Name() || oldField.Embedded() != newField.Embedded() {
			return fmt.Errorf("cross-package type move changed struct field identity")
		}
		objects[oldField] = newField
	}
	return nil
}

func crossPackageMovedDeclarationPairs(group *crossPackageTypeGroup, destination *packages.Package, movedType *types.TypeName, movedMethods map[string]*types.Func) ([]crossPackageMovedDeclarationPair, error) {
	pairs := make([]crossPackageMovedDeclarationPair, 0, 1+len(group.methods))
	newTypeDecl := movedTypeSpecDeclaration(destination, movedType)
	if newTypeDecl == nil {
		return nil, fmt.Errorf("cross-package type move cannot locate moved type declaration for binding proof")
	}
	pairs = append(pairs, crossPackageMovedDeclarationPair{group.typeDeclaration, newTypeDecl, group.typeObject.Name()})
	for _, method := range group.methods {
		newDecl := movedMethodDeclaration(destination, movedMethods[method.object.Name()])
		if newDecl == nil {
			return nil, fmt.Errorf("cross-package type move cannot locate moved method %q for binding proof", method.object.Name())
		}
		pairs = append(pairs, crossPackageMovedDeclarationPair{method.declaration, newDecl, method.object.Name()})
	}
	return pairs, nil
}

func pairCrossPackageMovedIdentifiers(group *crossPackageTypeGroup, destination *packages.Package, objects map[types.Object]types.Object, pairs []crossPackageMovedDeclarationPair) ([]crossPackageMovedIdentifierPair, error) {
	allIdentifiers := []crossPackageMovedIdentifierPair{}
	for _, pair := range pairs {
		oldIDs, newIDs := moveIdentifierSequence(pair.old), moveIdentifierSequence(pair.new)
		if len(oldIDs) != len(newIDs) {
			return nil, fmt.Errorf("cross-package type move cannot pair identifier bindings in moved declaration %q", pair.name)
		}
		for index := range oldIDs {
			oldIdent, newIdent := oldIDs[index], newIDs[index]
			if oldIdent.Name != newIdent.Name {
				return nil, fmt.Errorf("cross-package type move cannot pair identifier %q in moved declaration %q", oldIdent.Name, pair.name)
			}
			allIdentifiers = append(allIdentifiers, crossPackageMovedIdentifierPair{old: oldIdent, current: newIdent, declaration: pair.name})
			oldObject := group.sourcePackage.TypesInfo.Defs[oldIdent]
			newObject := destination.TypesInfo.Defs[newIdent]
			if oldObject != nil || newObject != nil {
				if oldObject == nil || newObject == nil {
					return nil, fmt.Errorf("cross-package type move cannot pair definition binding for %q", oldIdent.Name)
				}
				if variable, ok := oldObject.(*types.Var); !ok || !variable.IsField() || variable.Exported() {
					objects[oldObject] = newObject
				}
			}
		}
	}
	return allIdentifiers, nil
}

func validateCrossPackageMovedIdentifierBindings(group *crossPackageTypeGroup, destination *packages.Package, objects map[types.Object]types.Object, newNamed *types.Named, identifiers []crossPackageMovedIdentifierPair) error {
	for _, identifier := range identifiers {
		oldIdent, newIdent := identifier.old, identifier.current
		oldObject, oldDef := group.sourcePackage.TypesInfo.Defs[oldIdent]
		if oldObject == nil {
			oldObject = group.sourcePackage.TypesInfo.Uses[oldIdent]
		}
		newObject, newDef := destination.TypesInfo.Defs[newIdent]
		if newObject == nil {
			newObject = destination.TypesInfo.Uses[newIdent]
		}
		if oldIdent.Name == "_" && newIdent.Name == "_" {
			continue
		}
		if oldObject == nil && newObject == nil {
			oldSelection := crossPackageSelectionForIdentifier(group.sourcePackage, oldIdent)
			newSelection := crossPackageSelectionForIdentifier(destination, newIdent)
			if oldSelection != nil && newSelection != nil && oldSelection.Kind() == newSelection.Kind() && slices.Equal(oldSelection.Index(), newSelection.Index()) && oldSelection.Indirect() == newSelection.Indirect() && crossPackageMovedBindingEqual(oldSelection.Obj(), newSelection.Obj(), objects, group.named, newNamed) && crossPackageMoveTypeEqual(oldSelection.Type(), newSelection.Type(), group.named, newNamed) {
				continue
			}
		}
		if oldDef != newDef || oldObject == nil || newObject == nil || !crossPackageMovedBindingEqual(oldObject, newObject, objects, group.named, newNamed) {
			return fmt.Errorf("cross-package type move changes identifier binding for %q in moved declaration %q", oldIdent.Name, identifier.declaration)
		}
	}
	return nil
}

func crossPackageSelectionForIdentifier(pkg *packages.Package, ident *ast.Ident) *types.Selection {
	var selection *types.Selection
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok && selector.Sel == ident {
				selection = pkg.TypesInfo.Selections[selector]
				return false
			}
			return selection == nil
		})
		if selection != nil {
			return selection
		}
	}
	return nil
}

func moveIdentifierSequence(node ast.Node) []*ast.Ident {
	identifiers := []*ast.Ident{}
	ast.Inspect(node, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			identifiers = append(identifiers, ident)
		}
		return true
	})
	return identifiers
}

func movedTypeSpecDeclaration(pkg *packages.Package, target *types.TypeName) *ast.GenDecl {
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, spec := range group.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && pkg.TypesInfo.Defs[typeSpec.Name] == target {
					return group
				}
			}
		}
	}
	return nil
}

func movedMethodDeclaration(pkg *packages.Package, target *types.Func) *ast.FuncDecl {
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			method, ok := declaration.(*ast.FuncDecl)
			if ok && pkg.TypesInfo.Defs[method.Name] == target {
				return method
			}
		}
	}
	return nil
}

func crossPackageMovedBindingEqual(old, current types.Object, moved map[types.Object]types.Object, oldNamed, newNamed *types.Named) bool {
	if mapped := moved[old]; mapped != nil {
		return current == mapped
	}
	if reflect.TypeOf(old) != reflect.TypeOf(current) || old.Name() != current.Name() {
		return false
	}
	if oldPkg, ok := old.(*types.PkgName); ok {
		newPkg, valid := current.(*types.PkgName)
		return valid && oldPkg.Imported().Path() == newPkg.Imported().Path()
	}
	if old.Pkg() == nil {
		if old.Parent() == types.Universe {
			return old == current
		}
		return current.Pkg() == nil && crossPackageMoveTypeEqual(old.Type(), current.Type(), oldNamed, newNamed)
	}
	if current.Pkg() == nil || old.Pkg().Path() != current.Pkg().Path() {
		return false
	}
	return crossPackageMoveTypeEqual(old.Type(), current.Type(), oldNamed, newNamed)
}

func isPointerType(typ types.Type) bool {
	_, ok := typ.(*types.Pointer)
	return ok
}

func identifierAtOffset(pkg *packages.Package, file *ast.File, offset int, name string) *ast.Ident {
	var found *ast.Ident
	ast.Inspect(file, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && ident.Name == name && pkg.Fset.Position(ident.Pos()).Offset == offset {
			found = ident
			return false
		}
		return found == nil
	})
	return found
}

func validateCrossPackageMethodSelection(pkg *packages.Package, file *ast.File, marker crossPackageTypeMarker, movedMethods map[string]*types.Func, oldNamed, newNamed *types.Named) error {
	var found *types.Selection
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != marker.name || pkg.Fset.Position(selector.Sel.Pos()).Offset != marker.offset {
			return true
		}
		found = pkg.TypesInfo.Selections[selector]
		return false
	})
	if found == nil {
		return fmt.Errorf("cross-package type move cannot locate method selection %q in %s", marker.name, pkg.Fset.Position(file.Pos()).Filename)
	}
	moved := movedMethods[marker.method.Name()]
	if moved == nil || found.Obj() != moved {
		return fmt.Errorf("cross-package type move refuses method selection %q because it resolves to a different declaration", marker.name)
	}
	if found.Kind() != marker.selection.Kind() || !slices.Equal(found.Index(), marker.selection.Index()) || found.Indirect() != marker.selection.Indirect() {
		return fmt.Errorf("cross-package type move changed method selection kind, index, or indirection for %q", marker.name)
	}
	if !crossPackageMoveTypeEqual(marker.selection.Type(), found.Type(), oldNamed, newNamed) {
		return fmt.Errorf("cross-package type move changed selected method signature for %q", marker.name)
	}
	return nil
}

func validateCrossPackageInterfaceRelations(group *crossPackageTypeGroup, movedNamed *types.Named, before, after []*packages.Package) error {
	oldInterfaces, err := collectCrossPackageModuleInterfaces(before, after, true)
	if err != nil {
		return err
	}
	newInterfaces, err := collectCrossPackageModuleInterfaces(after, nil, false)
	if err != nil {
		return err
	}
	return compareCrossPackageInterfaceRelations(group.named, movedNamed, oldInterfaces, newInterfaces)
}

func collectCrossPackageModuleInterfaces(packages, counterpart []*packages.Package, requirePair bool) ([]*types.Interface, error) {
	var interfaces []*types.Interface
	for _, pkg := range packages {
		if pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		if requirePair && packageByID(counterpart, pkg.ID) == nil {
			return nil, fmt.Errorf("cross-package type move cannot pair package %s for interface proof", pkg.ID)
		}
		found, err := crossPackageInterfaces(pkg)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, found...)
	}
	return interfaces, nil
}

func compareCrossPackageInterfaceRelations(oldNamed, movedNamed *types.Named, oldInterfaces, newInterfaces []*types.Interface) error {
	oldClasses := uniqueInterfaceShapes(oldInterfaces, oldNamed, oldNamed)
	newClasses := uniqueInterfaceShapes(newInterfaces, movedNamed, movedNamed)
	if len(oldClasses) != len(newClasses) {
		return fmt.Errorf("cross-package type move cannot pair all interface shapes across captured module packages")
	}
	matched := make([]bool, len(newClasses))
	for _, oldInterface := range oldClasses {
		match := -1
		for index, newInterface := range newClasses {
			if crossPackageMoveTypeEqual(oldInterface, newInterface, oldNamed, movedNamed) {
				if match >= 0 {
					return fmt.Errorf("cross-package type move found ambiguous interface pairing across captured module packages")
				}
				match = index
			}
		}
		if match < 0 || matched[match] {
			return fmt.Errorf("cross-package type move cannot pair an interface relation across captured module packages")
		}
		matched[match] = true
		newInterface := newClasses[match]
		if types.Implements(oldNamed, oldInterface) != types.Implements(movedNamed, newInterface) || types.Implements(types.NewPointer(oldNamed), oldInterface) != types.Implements(types.NewPointer(movedNamed), newInterface) {
			return fmt.Errorf("cross-package type move changes value or pointer interface satisfaction")
		}
	}
	return nil
}

func crossPackageInterfaces(pkg *packages.Package) ([]*types.Interface, error) {
	if pkg == nil || pkg.Types == nil || pkg.TypesInfo == nil {
		return nil, fmt.Errorf("cross-package type move cannot establish interface proof for incomplete package information")
	}
	collector := crossPackageInterfaceCollector{pkg: pkg, seen: map[types.Type]bool{}}
	for typ := range pkg.TypesInfo.Types {
		if err := collector.visit(pkg.TypesInfo.Types[typ].Type); err != nil {
			return nil, err
		}
	}
	for _, name := range pkg.Types.Scope().Names() {
		if object := pkg.Types.Scope().Lookup(name); object != nil {
			if err := collector.visit(object.Type()); err != nil {
				return nil, err
			}
		}
	}
	return collector.interfaces, nil
}

type crossPackageInterfaceCollector struct {
	pkg        *packages.Package
	seen       map[types.Type]bool
	interfaces []*types.Interface
}

func (collector *crossPackageInterfaceCollector) visit(typ types.Type) error {
	if typ == nil || collector.seen[typ] {
		return nil
	}
	collector.seen[typ] = true
	switch item := typ.(type) {
	case *types.Alias:
		return collector.visit(types.Unalias(item))
	case *types.Named:
		return collector.visitNamed(item)
	case *types.Interface:
		return collector.visitInterface(item)
	case *types.Pointer:
		return collector.visit(item.Elem())
	case *types.Array:
		return collector.visit(item.Elem())
	case *types.Slice:
		return collector.visit(item.Elem())
	case *types.Map:
		return collector.visitTypes([]types.Type{item.Key(), item.Elem()})
	case *types.Chan:
		return collector.visit(item.Elem())
	case *types.Tuple:
		return collector.visitTuple(item)
	case *types.Signature:
		return collector.visitTypes([]types.Type{item.Params(), item.Results()})
	case *types.TypeParam:
		return collector.visit(item.Constraint())
	case *types.Struct:
		return collector.visitStruct(item)
	}
	return nil
}

func (collector *crossPackageInterfaceCollector) visitNamed(named *types.Named) error {
	if _, ok := named.Underlying().(*types.Interface); !ok {
		return nil
	}
	if named.TypeParams() != nil && named.TypeParams().Len() != 0 {
		return fmt.Errorf("cross-package type move refuses generic interface relation in %s", collector.pkg.PkgPath)
	}
	return collector.visit(named.Underlying())
}

func (collector *crossPackageInterfaceCollector) visitInterface(item *types.Interface) error {
	item.Complete()
	if !item.IsMethodSet() {
		return fmt.Errorf("cross-package type move refuses non-method-set interface relation in %s", collector.pkg.PkgPath)
	}
	collector.interfaces = append(collector.interfaces, item)
	for method := range item.Methods() {
		if err := collector.visit(method.Type()); err != nil {
			return err
		}
	}
	return nil
}

func (collector *crossPackageInterfaceCollector) visitTuple(tuple *types.Tuple) error {
	for variable := range tuple.Variables() {
		if err := collector.visit(variable.Type()); err != nil {
			return err
		}
	}
	return nil
}

func (collector *crossPackageInterfaceCollector) visitStruct(structure *types.Struct) error {
	for field := range structure.Fields() {
		if err := collector.visit(field.Type()); err != nil {
			return err
		}
	}
	return nil
}

func (collector *crossPackageInterfaceCollector) visitTypes(types []types.Type) error {
	for _, typ := range types {
		if err := collector.visit(typ); err != nil {
			return err
		}
	}
	return nil
}

func uniqueInterfaceShapes(interfaces []*types.Interface, fromNamed, toNamed *types.Named) []*types.Interface {
	unique := []*types.Interface{}
	for _, candidate := range interfaces {
		duplicate := false
		for _, existing := range unique {
			if crossPackageMoveTypeEqual(candidate, existing, fromNamed, toNamed) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			unique = append(unique, candidate)
		}
	}
	return unique
}

func crossPackageMoveTypeEqual(left, right types.Type, fromNamed, toNamed *types.Named) bool {
	return crossPackageMoveTypeEqualSeen(left, right, fromNamed, toNamed, map[[2]types.Type]bool{})
}

func crossPackageMoveTypeEqualSeen(left, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	if left == fromNamed && right == toNamed {
		return true
	}
	if left == nil || right == nil {
		return left == right
	}
	pair := [2]types.Type{left, right}
	if seen[pair] {
		return true
	}
	seen[pair] = true
	if alias, ok := left.(*types.Alias); ok {
		left = types.Unalias(alias)
	}
	if alias, ok := right.(*types.Alias); ok {
		right = types.Unalias(alias)
	}
	if left == fromNamed && right == toNamed {
		return true
	}
	switch a := left.(type) {
	case *types.Basic:
		return crossPackageBasicEqual(a, right)
	case *types.Pointer:
		return crossPackageUnaryTypeEqual(a.Elem(), right, fromNamed, toNamed, seen, func(typ types.Type) types.Type {
			item, _ := typ.(*types.Pointer)
			if item == nil {
				return nil
			}
			return item.Elem()
		})
	case *types.Slice:
		return crossPackageUnaryTypeEqual(a.Elem(), right, fromNamed, toNamed, seen, func(typ types.Type) types.Type {
			item, _ := typ.(*types.Slice)
			if item == nil {
				return nil
			}
			return item.Elem()
		})
	case *types.Array:
		return crossPackageArrayEqual(a, right, fromNamed, toNamed, seen)
	case *types.Map:
		return crossPackageMapEqual(a, right, fromNamed, toNamed, seen)
	case *types.Chan:
		return crossPackageChanEqual(a, right, fromNamed, toNamed, seen)
	case *types.Named:
		return crossPackageNamedEqual(a, right, fromNamed, toNamed, seen)
	case *types.Tuple:
		return crossPackageTupleEqual(a, right, fromNamed, toNamed, seen)
	case *types.Signature:
		return crossPackageSignatureEqual(a, right, fromNamed, toNamed, seen)
	case *types.Interface:
		return crossPackageInterfaceEqual(a, right, fromNamed, toNamed, seen)
	case *types.Struct:
		return crossPackageStructEqual(a, right, fromNamed, toNamed, seen)
	default:
		return types.Identical(left, right)
	}
}

func crossPackageBasicEqual(left *types.Basic, right types.Type) bool {
	other, ok := right.(*types.Basic)
	return ok && left.Kind() == other.Kind()
}

func crossPackageUnaryTypeEqual(left types.Type, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool, element func(types.Type) types.Type) bool {
	other := element(right)
	return other != nil && crossPackageMoveTypeEqualSeen(left, other, fromNamed, toNamed, seen)
}

func crossPackageArrayEqual(left *types.Array, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Array)
	return ok && left.Len() == other.Len() && crossPackageMoveTypeEqualSeen(left.Elem(), other.Elem(), fromNamed, toNamed, seen)
}

func crossPackageMapEqual(left *types.Map, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Map)
	return ok && crossPackageMoveTypeEqualSeen(left.Key(), other.Key(), fromNamed, toNamed, seen) && crossPackageMoveTypeEqualSeen(left.Elem(), other.Elem(), fromNamed, toNamed, seen)
}

func crossPackageChanEqual(left *types.Chan, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Chan)
	return ok && left.Dir() == other.Dir() && crossPackageMoveTypeEqualSeen(left.Elem(), other.Elem(), fromNamed, toNamed, seen)
}

func crossPackageNamedEqual(left *types.Named, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Named)
	if !ok || left.Obj().Name() != other.Obj().Name() || packagePath(left.Obj()) != packagePath(other.Obj()) {
		return false
	}
	return crossPackageMoveTypeEqualSeen(left.Underlying(), other.Underlying(), fromNamed, toNamed, seen) && crossPackageMoveTypeListsEqual(left.TypeArgs(), other.TypeArgs(), fromNamed, toNamed, seen)
}

func crossPackageTupleEqual(left *types.Tuple, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Tuple)
	if !ok || left.Len() != other.Len() {
		return false
	}
	for index := range left.Len() {
		if !crossPackageMoveTypeEqualSeen(left.At(index).Type(), other.At(index).Type(), fromNamed, toNamed, seen) {
			return false
		}
	}
	return true
}

func crossPackageSignatureEqual(left *types.Signature, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Signature)
	return ok && left.Variadic() == other.Variadic() && crossPackageMoveTypeEqualSeen(left.Params(), other.Params(), fromNamed, toNamed, seen) && crossPackageMoveTypeEqualSeen(left.Results(), other.Results(), fromNamed, toNamed, seen)
}

func crossPackageInterfaceEqual(left *types.Interface, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Interface)
	if !ok {
		return false
	}
	left.Complete()
	other.Complete()
	if left.NumMethods() != other.NumMethods() {
		return false
	}
	for index := range left.NumMethods() {
		leftMethod, rightMethod := left.Method(index), other.Method(index)
		if leftMethod.Name() != rightMethod.Name() || (!leftMethod.Exported() && packagePath(leftMethod) != packagePath(rightMethod)) || !crossPackageMoveTypeEqualSeen(leftMethod.Type(), rightMethod.Type(), fromNamed, toNamed, seen) {
			return false
		}
	}
	return true
}

func crossPackageStructEqual(left *types.Struct, right types.Type, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	other, ok := right.(*types.Struct)
	if !ok || left.NumFields() != other.NumFields() {
		return false
	}
	for index := range left.NumFields() {
		leftField, rightField := left.Field(index), other.Field(index)
		if leftField.Name() != rightField.Name() || (!leftField.Exported() && packagePath(leftField) != packagePath(rightField)) || leftField.Embedded() != rightField.Embedded() || left.Tag(index) != other.Tag(index) || !crossPackageMoveTypeEqualSeen(leftField.Type(), rightField.Type(), fromNamed, toNamed, seen) {
			return false
		}
	}
	return true
}

func packagePath(object types.Object) string {
	if object == nil || object.Pkg() == nil {
		return ""
	}
	return object.Pkg().Path()
}

func crossPackageMoveTypeListsEqual(left, right *types.TypeList, fromNamed, toNamed *types.Named, seen map[[2]types.Type]bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.Len() != right.Len() {
		return false
	}
	for index := range left.Len() {
		if !crossPackageMoveTypeEqualSeen(left.At(index), right.At(index), fromNamed, toNamed, seen) {
			return false
		}
	}
	return true
}

func resolveCrossPackageTypeAlias(snapshot crossPackageMoveSnapshot, name string, selected crossPackagePackages) (crossPackageMoveTarget, bool, error) {
	declaration, spec := findCrossPackageTypeSpec(selected.sourceFile, name)
	if spec == nil || !spec.Assign.IsValid() {
		return crossPackageMoveTarget{}, false, nil
	}
	if err := requireCrossPackageMovePackageInfo(selected.all); err != nil {
		return crossPackageMoveTarget{}, true, err
	}
	if len(declaration.Specs) != 1 || spec.TypeParams != nil && len(spec.TypeParams.List) != 0 {
		return crossPackageMoveTarget{}, true, fmt.Errorf("cross-package type alias move requires one standalone non-generic alias")
	}
	typeObject, ok := selected.source.TypesInfo.Defs[spec.Name].(*types.TypeName)
	if !ok || typeObject == nil || !typeObject.IsAlias() || !typeObject.Exported() || typeObject.Pkg() == nil || typeObject.Pkg().Path() != selected.source.PkgPath || filepath.Clean(selected.source.Fset.Position(typeObject.Pos()).Filename) != snapshot.sourcePath {
		return crossPackageMoveTarget{}, true, fmt.Errorf("cross-package type alias move requires an exported alias in the selected source file")
	}
	if err := validateCrossPackagePredeclaredTypeExpression(selected.source, spec.Type); err != nil {
		return crossPackageMoveTarget{}, true, err
	}
	if err := validateCrossPackageTestReferences(snapshot, name, nil); err != nil {
		return crossPackageMoveTarget{}, true, err
	}
	if selected.destination.Types.Scope().Lookup(name) != nil {
		return crossPackageMoveTarget{}, true, fmt.Errorf("destination already declares %q", name)
	}
	if err := validateCrossPackageTypeSource(snapshot, selected.sourceFile, selected.destFile, name); err != nil {
		return crossPackageMoveTarget{}, true, err
	}
	start, end := moveDeclOffsets(selected.source.Fset, declaration, selected.sourceFile, snapshot.files[snapshot.sourcePath])
	if start < 0 || end < start || end > len(snapshot.files[snapshot.sourcePath]) {
		return crossPackageMoveTarget{}, true, fmt.Errorf("invalid source declaration bounds for type alias %q", name)
	}
	return crossPackageMoveTarget{
		packages: selected.all, sourcePackage: selected.source, destinationPackage: selected.destination,
		targetObject: typeObject, declarationStart: start, declarationEnd: end,
		declarationText: append([]byte(nil), snapshot.files[snapshot.sourcePath][start:end]...),
		neededImports:   &ast.File{}, typeAlias: true, aliasType: types.Unalias(typeObject.Type()),
	}, true, nil
}
