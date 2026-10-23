// Package astedit isolates whole-file relocation because it moves declaration sets and their test companion as one typed module edit.
package astedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// MoveFileOptions controls companion-test relocation and formatting.
type MoveFileOptions struct {
	MoveMatchingTests   bool
	AutoOrganizeImports bool
	DeferFormatting     bool
}

type moveFileDecl struct {
	decl  ast.Decl
	start int
	end   int
	names map[string]bool
}

type moveFileObjectKey struct {
	path   string
	offset int
	name   string
}

// MoveFile relocates a production file's declarations and its exact sibling test file as one checked module edit.
func MoveFile(ctx context.Context, sourcePath, destinationPath string, opts MoveFileOptions) (string, error) {
	if sourcebuf.FromContext(ctx) == nil {
		root, err := sourceBufferRoot(sourcePath)
		if err != nil {
			return "", fmt.Errorf("locate move module: %w", err)
		}
		var diff string
		err = sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
			var moveErr error
			diff, moveErr = MoveFile(editCtx, sourcePath, destinationPath, opts)
			return moveErr
		})
		return diff, err
	}
	state, snapshotErr := prepareMoveFileSnapshot(ctx, sourcePath, destinationPath, opts)
	if snapshotErr != nil {
		return "", snapshotErr
	}
	root, sourcePath, destinationPath := state.root, state.sourcePath, state.destinationPath
	files, companionSource, companionDestination := state.files, state.companionSource, state.companionDestination
	analysis, analysisErr := analyzeMoveFile(ctx, state)
	if analysisErr != nil {
		return "", analysisErr
	}
	changes, expectations, planErr := planMoveFileChanges(moveFilePlanInput{
		packages: analysis.packages, files: files, sourcePath: sourcePath, destinationPath: destinationPath,
		companionSource: companionSource, companionDestination: companionDestination,
		sourcePackage: analysis.sourcePackage, destinationPackage: analysis.destinationPackage, sourceSyntax: analysis.sourceSyntax,
		declarations: analysis.declarations, objects: analysis.objects,
	})
	if planErr != nil {
		return "", planErr
	}
	post, typecheckErr := typecheckMoveOverlay(ctx, root, changes)
	if typecheckErr != nil {
		return "", typecheckErr
	}
	if err := validateMoveFileReferences(post, expectations); err != nil {
		return "", err
	}
	if err := validateMoveFileInitialization(analysis.packages, post, sourcePath, destinationPath, companionSource, companionDestination); err != nil {
		return "", err
	}
	return applyMoveFileChanges(ctx, files, changes)
}

type moveFileAnalysis struct {
	packages                          []*packages.Package
	sourcePackage, destinationPackage *packages.Package
	sourceSyntax, destinationSyntax   *ast.File
	declarations                      []moveFileDecl
	objects                           map[moveFileObjectKey]types.Object
}

func analyzeMoveFile(ctx context.Context, state moveFileSnapshot) (moveFileAnalysis, error) {
	pkgs, loadErr := loadMovePackages(ctx, state.root, nil)
	if loadErr != nil {
		return moveFileAnalysis{}, fmt.Errorf("load Go module packages: %w", loadErr)
	}
	if err := validateLoadedMovePackages(pkgs, state.files, ""); err != nil {
		return moveFileAnalysis{}, err
	}
	sourcePkg, destinationPkg := packageForMoveFile(pkgs, state.sourcePath), packageForMoveFile(pkgs, state.destinationPath)
	if sourcePkg == nil || destinationPkg == nil || sourcePkg.PkgPath == destinationPkg.PkgPath || !sameMoveMainModule(sourcePkg, destinationPkg) {
		return moveFileAnalysis{}, fmt.Errorf("move_file requires active files in distinct packages of the same main module")
	}
	sourceAST, destinationAST := syntaxForMoveFile(sourcePkg, state.sourcePath), syntaxForMoveFile(destinationPkg, state.destinationPath)
	if sourceAST == nil || destinationAST == nil {
		return moveFileAnalysis{}, fmt.Errorf("move_file requires active source and destination files")
	}
	if sourcePkg.Name == "main" || destinationPkg.Name == "main" {
		return moveFileAnalysis{}, fmt.Errorf("move_file refuses package main entrypoints")
	}
	if moveFileHasLinkageDirective(sourceAST) || moveFileHasLinkageDirective(destinationAST) {
		return moveFileAnalysis{}, fmt.Errorf("move_file refuses compiler linkage directives in selected files")
	}
	for _, spec := range sourceAST.Imports {
		if spec.Name != nil && (spec.Name.Name == "." || spec.Name.Name == "_") {
			return moveFileAnalysis{}, fmt.Errorf("move_file refuses dot or blank imports in the source file")
		}
	}
	decls, objects, err := collectMoveFileDeclarations(sourcePkg, sourceAST, state.files[state.sourcePath])
	if err != nil {
		return moveFileAnalysis{}, err
	}
	if len(decls) == 0 {
		return moveFileAnalysis{}, fmt.Errorf("source file has no movable declarations")
	}
	if err := validateMoveFileClosure(pkgs, state, sourcePkg, sourceAST, destinationAST, decls, objects); err != nil {
		return moveFileAnalysis{}, err
	}
	if err := rejectMoveFileBuiltinRebinding(pkgs, state, sourcePkg, destinationPkg, decls); err != nil {
		return moveFileAnalysis{}, err
	}
	return moveFileAnalysis{packages: pkgs, sourcePackage: sourcePkg, destinationPackage: destinationPkg, sourceSyntax: sourceAST, destinationSyntax: destinationAST, declarations: decls, objects: objects}, nil
}

func rejectMoveFileBuiltinRebinding(pkgs []*packages.Package, state moveFileSnapshot, sourcePkg, destinationPkg *packages.Package, decls []moveFileDecl) error {
	check := func(pkg *packages.Package, node ast.Node) error {
		destinationScopes := moveFileDestinationBuiltinScopes(pkgs, pkg, destinationPkg)
		var bindingErr error
		ast.Inspect(node, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			obj := pkg.TypesInfo.Uses[id]
			if obj == nil || obj.Parent() != types.Universe {
				return true
			}
			shadow, target := moveFileBuiltinShadow(destinationScopes, obj.Name())
			if shadow == nil {
				return true
			}
			use := pkg.Fset.Position(id.Pos())
			declaration := target.Fset.Position(shadow.Pos())
			bindingErr = fmt.Errorf("move_file refuses builtin %q at %s:%d because destination declaration at %s changes its binding", obj.Name(), filepath.Clean(use.Filename), use.Line, filepath.Clean(declaration.Filename))
			return false
		})
		return bindingErr
	}
	for _, decl := range decls {
		if err := check(sourcePkg, decl.decl); err != nil {
			return err
		}
	}
	if state.companionSource == "" {
		return nil
	}
	for _, pkg := range pkgs {
		file := syntaxForMoveFile(pkg, state.companionSource)
		if file == nil || pkg.PkgPath != sourcePkg.PkgPath && pkg.Name != sourcePkg.Name+"_test" {
			continue
		}
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
				continue
			}
			if err := check(pkg, decl); err != nil {
				return err
			}
		}
	}
	return nil
}

func moveFileDestinationBuiltinScopes(pkgs []*packages.Package, source *packages.Package, destination *packages.Package) []*packages.Package {
	targetName := destination.Name
	if strings.HasSuffix(source.Name, "_test") {
		targetName += "_test"
	}
	scopes := make([]*packages.Package, 0, 2)
	for _, pkg := range pkgs {
		if pkg.Name != targetName || pkg.Types == nil {
			continue
		}
		if pkg.PkgPath == destination.PkgPath || pkg.PkgPath == destination.PkgPath+"_test" {
			scopes = append(scopes, pkg)
		}
	}
	return scopes
}

func moveFileBuiltinShadow(scopes []*packages.Package, name string) (types.Object, *packages.Package) {
	for _, pkg := range scopes {
		if shadow := pkg.Types.Scope().Lookup(name); shadow != nil {
			return shadow, pkg
		}
	}
	return nil, nil
}

func validateMoveFileClosure(pkgs []*packages.Package, state moveFileSnapshot, sourcePkg *packages.Package, sourceAST, destinationAST *ast.File, declarations []moveFileDecl, objects map[moveFileObjectKey]types.Object) error {
	for name := range moveFileObjectNames(objects) {
		for path, content := range state.files {
			if err := validateMoveFileModuleSource(pkgs, path, content, state.sourcePath, state.destinationPath, sourcePkg.PkgPath, sourcePkg.Name, name, objects); err != nil {
				return err
			}
		}
		if err := rejectUntypedMoveReferences(state.files, pkgs, name); err != nil {
			return err
		}
	}
	for _, check := range []func() error{
		func() error { return rejectMoveFilePackageEffects(pkgs, state.sourcePath, state.destinationPath) },
		func() error { return rejectMoveFileCollisions(destinationAST, declarations) },
		func() error { return rejectMoveFileDependencies(sourcePkg, objects, declarations) },
		func() error {
			return rejectMoveFileExternalPrivateReferences(pkgs, sourcePkg, objects, state.sourcePath, state.companionSource)
		},
		func() error { return rejectMoveFilePathEffects(pkgs, sourceAST, state.companionSource, state.files) },
	} {
		if err := check(); err != nil {
			return err
		}
	}
	if state.companionSource != "" {
		if _, err := validateMoveTestCompanion(pkgs, state.companionSource, sourcePkg); err != nil {
			return err
		}
	}
	return nil
}

func validateMoveFileModuleSource(pkgs []*packages.Package, path string, content []byte, sourcePath, destinationPath, sourceImport, sourceName, name string, objects map[moveFileObjectKey]types.Object) error {
	cleanPath := filepath.Clean(path)
	text := string(content)
	generated := strings.Contains(text, "Code generated ") && strings.Contains(text, "DO NOT EDIT.")
	constrained := moveFilenameBuildConstraint(path) != "" || strings.Contains(text, "//go:build ") || strings.Contains(text, "// +build ")
	if (cleanPath == filepath.Clean(sourcePath) || cleanPath == filepath.Clean(destinationPath)) && (generated || constrained) {
		return fmt.Errorf("cross-package move refuses generated or build-constrained source or destination %s", path)
	}
	if !generated && !constrained {
		return nil
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if filepath.Clean(pkg.Fset.Position(file.Pos()).Filename) != cleanPath {
				continue
			}
			reference := moveFileMovedReferenceInFile(pkg, file, name, objects)
			if reference != nil {
				position := pkg.Fset.Position(reference.Pos())
				return fmt.Errorf("cross-package move refuses reference to %q in generated or build-constrained file %s:%d", name, path, position.Line)
			}
			return nil
		}
	}
	if moveFileUnloadedMayReference(content, path, sourcePath, destinationPath, sourceImport, sourceName, name) {
		return fmt.Errorf("cross-package move cannot prove whether %q in generated or build-constrained file %s refers to the moved declaration", name, path)
	}
	return nil
}

func moveFileUnloadedMayReference(content []byte, path, sourcePath, destinationPath, sourceImport, sourceName, name string) bool {
	dir := filepath.Dir(filepath.Clean(path))
	if dir == filepath.Dir(filepath.Clean(sourcePath)) || dir == filepath.Dir(filepath.Clean(destinationPath)) {
		return containsMoveIdentifier(content, name)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.AllErrors)
	if err != nil {
		return containsMoveIdentifier(content, name)
	}
	aliases := map[string]bool{}
	dotImport := false
	for _, spec := range file.Imports {
		importPath, err := strconvUnquote(spec.Path.Value)
		if err != nil || importPath != sourceImport {
			continue
		}
		if spec.Name != nil && spec.Name.Name == "." {
			dotImport = true
			continue
		}
		alias := sourceName
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		aliases[alias] = true
	}
	if dotImport && containsMoveIdentifier(content, name) {
		return true
	}
	mayReference := false
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return true
		}
		qualifier, ok := selector.X.(*ast.Ident)
		if ok && aliases[qualifier.Name] {
			mayReference = true
			return false
		}
		return true
	})
	return mayReference
}

func moveFileMovedReferenceInFile(pkg *packages.Package, file *ast.File, name string, objects map[moveFileObjectKey]types.Object) *ast.Ident {
	var reference *ast.Ident
	ast.Inspect(file, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok || id.Name != name {
			return true
		}
		obj := pkg.TypesInfo.Uses[id]
		if obj == nil {
			return true
		}
		pos := pkg.Fset.Position(obj.Pos())
		if _, moving := objects[moveFileObjectKey{filepath.Clean(pos.Filename), pos.Offset, obj.Name()}]; moving {
			reference = id
			return false
		}
		return true
	})
	return reference
}

type moveFileSnapshot struct {
	root, sourcePath, destinationPath     string
	companionSource, companionDestination string
	files                                 map[string][]byte
}

func prepareMoveFileSnapshot(ctx context.Context, sourcePath, destinationPath string, opts MoveFileOptions) (moveFileSnapshot, error) {
	var state moveFileSnapshot
	var pathErr error
	state.sourcePath, pathErr = filepath.Abs(filepath.Clean(sourcePath))
	if pathErr != nil {
		return moveFileSnapshot{}, fmt.Errorf("resolve source path: %w", pathErr)
	}
	state.destinationPath, pathErr = filepath.Abs(filepath.Clean(destinationPath))
	if pathErr != nil {
		return moveFileSnapshot{}, fmt.Errorf("resolve destination path: %w", pathErr)
	}
	if state.sourcePath == state.destinationPath || strings.HasSuffix(state.sourcePath, "_test.go") || strings.HasSuffix(state.destinationPath, "_test.go") {
		return moveFileSnapshot{}, fmt.Errorf("move_file requires distinct production Go files")
	}
	if filepath.Ext(state.sourcePath) != ".go" || filepath.Ext(state.destinationPath) != ".go" {
		return moveFileSnapshot{}, fmt.Errorf("move_file requires Go source paths")
	}
	state.root, pathErr = moveModuleRoot(state.sourcePath)
	if pathErr != nil {
		return moveFileSnapshot{}, pathErr
	}
	if err := validateMoveFileSnapshotPaths(ctx, state.root, state.sourcePath, state.destinationPath); err != nil {
		return moveFileSnapshot{}, err
	}
	state.root, pathErr = filepath.EvalSymlinks(state.root)
	if pathErr != nil {
		return moveFileSnapshot{}, fmt.Errorf("canonicalize module root: %w", pathErr)
	}
	state.sourcePath, pathErr = sourcebuf.CanonicalPath(ctx, state.sourcePath)
	if pathErr != nil {
		return moveFileSnapshot{}, fmt.Errorf("canonicalize source path: %w", pathErr)
	}
	state.destinationPath, pathErr = sourcebuf.CanonicalPath(ctx, state.destinationPath)
	if pathErr != nil {
		return moveFileSnapshot{}, fmt.Errorf("canonicalize destination path: %w", pathErr)
	}
	if !moveWithin(state.root, state.sourcePath) || !moveWithin(state.root, state.destinationPath) {
		return moveFileSnapshot{}, fmt.Errorf("move_file requires source and destination in the same Go module")
	}
	if state.sourcePath == state.destinationPath {
		return moveFileSnapshot{}, fmt.Errorf("move_file requires distinct production Go files")
	}
	state.companionSource = strings.TrimSuffix(state.sourcePath, ".go") + "_test.go"
	state.companionDestination = strings.TrimSuffix(state.destinationPath, ".go") + "_test.go"
	if !opts.MoveMatchingTests {
		state.companionSource, state.companionDestination = "", ""
	}
	state.files, pathErr = inspectMoveModuleFiles(ctx, state.root, state.sourcePath, state.destinationPath, "")
	if pathErr != nil {
		return moveFileSnapshot{}, pathErr
	}
	if err := validateMoveFileSnapshotContents(ctx, &state); err != nil {
		return moveFileSnapshot{}, err
	}
	return state, nil
}

func validateMoveFileSnapshotPaths(ctx context.Context, root, sourcePath, destinationPath string) error {
	if !moveWithin(root, destinationPath) {
		return fmt.Errorf("move_file requires source and destination in the same Go module")
	}
	if err := rejectMoveSymlinkPath(root, sourcePath); err != nil {
		return err
	}
	if err := rejectMoveSymlinkPath(root, destinationPath); err != nil {
		return err
	}
	return sourcebuf.FromContext(ctx).AuthorizeWrites(ctx, root)
}

func validateMoveFileSnapshotContents(ctx context.Context, state *moveFileSnapshot) error {
	if _, ok := state.files[state.sourcePath]; !ok {
		return fmt.Errorf("source must be an existing Go file in the selected module: %s", state.sourcePath)
	}
	if _, ok := state.files[state.destinationPath]; !ok {
		return fmt.Errorf("destination must be an existing Go file in the selected module: %s", state.destinationPath)
	}
	if state.companionSource == "" {
		return nil
	}
	if _, ok := state.files[state.companionSource]; !ok {
		state.companionSource, state.companionDestination = "", ""
		return nil
	}
	if _, err := sourcebuf.ReadFile(ctx, state.companionDestination); err == nil {
		return fmt.Errorf("matching test destination already exists or cannot be inspected: %s", state.companionDestination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect matching test destination %s: %w", state.companionDestination, err)
	}
	return nil
}

func collectMoveFileDeclarations(pkg *packages.Package, file *ast.File, source []byte) ([]moveFileDecl, map[moveFileObjectKey]types.Object, error) {
	decls := make([]moveFileDecl, 0, len(file.Decls))
	objects := map[moveFileObjectKey]types.Object{}
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		item, movedObjects, err := collectOneMoveFileDeclaration(pkg, file, source, decl)
		if err != nil {
			return nil, nil, err
		}
		decls = append(decls, item)
		maps.Copy(objects, movedObjects)
	}
	return decls, objects, nil
}

func collectOneMoveFileDeclaration(pkg *packages.Package, file *ast.File, source []byte, decl ast.Decl) (moveFileDecl, map[moveFileObjectKey]types.Object, error) {
	if err := validateMoveFileDeclarationKind(decl); err != nil {
		return moveFileDecl{}, nil, err
	}
	if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil && !moveFileReceiverDeclaredHere(pkg, fn) {
		return moveFileDecl{}, nil, fmt.Errorf("move_file refuses method on a type declared outside the source file: %s", fn.Name.Name)
	}
	start, end := moveDeclOffsets(pkg.Fset, decl, file, source)
	if start < 0 || end < start || end > len(source) {
		return moveFileDecl{}, nil, fmt.Errorf("invalid source declaration bounds")
	}
	item := moveFileDecl{decl: decl, start: start, end: end, names: moveFileDeclNames(decl)}
	objects, err := moveFileDeclarationObjects(pkg, decl)
	if err != nil {
		return moveFileDecl{}, nil, err
	}
	return item, objects, nil
}

func validateMoveFileDeclarationKind(decl ast.Decl) error {
	switch node := decl.(type) {
	case *ast.GenDecl:
		if node.Tok == token.VAR {
			return fmt.Errorf("move_file refuses package variables because their initialization effects may change")
		}
		if node.Tok != token.CONST && node.Tok != token.TYPE {
			return fmt.Errorf("move_file encountered unsupported declaration kind %s", node.Tok)
		}
	case *ast.FuncDecl:
		if node.Name.Name == "init" {
			return fmt.Errorf("move_file refuses init functions")
		}
	default:
		return fmt.Errorf("move_file encountered unsupported top-level declaration")
	}
	return nil
}

func moveFileDeclarationObjects(pkg *packages.Package, decl ast.Decl) (map[moveFileObjectKey]types.Object, error) {
	objects := map[moveFileObjectKey]types.Object{}
	if fn, ok := decl.(*ast.FuncDecl); ok {
		obj := pkg.TypesInfo.Defs[fn.Name]
		if obj == nil {
			return nil, fmt.Errorf("cannot resolve declaration %q", fn.Name.Name)
		}
		pos := pkg.Fset.Position(obj.Pos())
		objects[moveFileObjectKey{filepath.Clean(pos.Filename), pos.Offset, obj.Name()}] = obj
	} else if gen, ok := decl.(*ast.GenDecl); ok {
		for _, spec := range gen.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if obj := pkg.TypesInfo.Defs[s.Name]; obj != nil {
					p := pkg.Fset.Position(obj.Pos())
					objects[moveFileObjectKey{filepath.Clean(p.Filename), p.Offset, obj.Name()}] = obj
				}
			case *ast.ValueSpec:
				for _, name := range s.Names {
					if obj := pkg.TypesInfo.Defs[name]; obj != nil {
						p := pkg.Fset.Position(obj.Pos())
						objects[moveFileObjectKey{filepath.Clean(p.Filename), p.Offset, obj.Name()}] = obj
					}
				}
			}
		}
	}
	return objects, nil
}

func moveFileDeclNames(decl ast.Decl) map[string]bool {
	names := map[string]bool{}
	switch n := decl.(type) {
	case *ast.FuncDecl:
		if n.Recv == nil {
			names[n.Name.Name] = true
		}
	case *ast.GenDecl:
		for _, sp := range n.Specs {
			switch s := sp.(type) {
			case *ast.TypeSpec:
				names[s.Name.Name] = true
			case *ast.ValueSpec:
				for _, id := range s.Names {
					names[id.Name] = true
				}
			}
		}
	}
	return names
}

func moveFileReceiverDeclaredHere(pkg *packages.Package, fn *ast.FuncDecl) bool {
	name := moveReceiverTypeName(fn.Recv)
	if name == "" {
		return false
	}
	obj := pkg.TypesInfo.Uses[astReceiverIdent(fn.Recv)]
	if obj == nil {
		return false
	}
	p := pkg.Fset.Position(obj.Pos())
	return filepath.Clean(p.Filename) == filepath.Clean(pkg.Fset.Position(fn.Pos()).Filename) && p.Offset >= 0
}

func astReceiverIdent(field *ast.FieldList) *ast.Ident {
	if field == nil || len(field.List) == 0 {
		return nil
	}
	e := field.List[0].Type
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id
	}
	if sel, ok := e.(*ast.SelectorExpr); ok {
		return sel.Sel
	}
	return nil
}
func moveReceiverTypeName(field *ast.FieldList) string {
	id := astReceiverIdent(field)
	if id == nil {
		return ""
	}
	return id.Name
}

func rejectMoveFilePackageEffects(pkgs []*packages.Package, sourcePath, destinationPath string) error {
	for _, pkg := range pkgs {
		if !moveFilePackageHasAffectedFile(pkg, sourcePath, destinationPath) {
			continue
		}
		if err := rejectMoveFilePackageVariantEffects(pkg); err != nil {
			return err
		}
	}
	return nil
}

func moveFilePackageHasAffectedFile(pkg *packages.Package, sourcePath, destinationPath string) bool {
	sourceDir, destinationDir := filepath.Dir(filepath.Clean(sourcePath)), filepath.Dir(filepath.Clean(destinationPath))
	for _, file := range pkg.Syntax {
		dir := filepath.Dir(filepath.Clean(pkg.Fset.Position(file.Pos()).Filename))
		if dir == sourceDir || dir == destinationDir {
			return true
		}
	}
	return false
}

func rejectMoveFilePackageVariantEffects(pkg *packages.Package) error {
	for _, file := range pkg.Syntax {
		for _, spec := range file.Imports {
			if spec.Name != nil && (spec.Name.Name == "." || spec.Name.Name == "_") {
				return fmt.Errorf("move_file refuses dot or blank imports in affected package %s", pkg.PkgPath)
			}
		}
		for _, decl := range file.Decls {
			fn, isFunction := decl.(*ast.FuncDecl)
			if isFunction && fn.Name.Name == "init" {
				return fmt.Errorf("move_file refuses package init effects in %s", pkg.PkgPath)
			}
			if isFunction && fn.Name.Name == "TestMain" {
				path := filepath.Clean(pkg.Fset.Position(fn.Pos()).Filename)
				return fmt.Errorf("move_file refuses TestMain in affected package %s at %s", pkg.PkgPath, path)
			}
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
				return fmt.Errorf("move_file refuses package variable effects in %s", pkg.PkgPath)
			}
		}
	}
	return nil
}

func moveFileObjectNames(objects map[moveFileObjectKey]types.Object) map[string]bool {
	names := map[string]bool{}
	for _, obj := range objects {
		names[obj.Name()] = true
	}
	return names
}

func moveFileMethodObject(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	return ok && sig.Recv() != nil
}

func rejectMoveFileCollisions(dest *ast.File, decls []moveFileDecl) error {
	names := map[string]bool{}
	for _, d := range decls {
		for n := range d.names {
			names[n] = true
		}
	}
	for _, decl := range dest.Decls {
		for name := range moveFileDeclNames(decl) {
			if names[name] {
				return fmt.Errorf("destination already declares %q", name)
			}
		}
	}
	return nil
}

func rejectMoveFileDependencies(pkg *packages.Package, objects map[moveFileObjectKey]types.Object, decls []moveFileDecl) error {
	moved := map[types.Object]bool{}
	for _, o := range objects {
		moved[o] = true
	}
	for _, d := range decls {
		var bad string
		ast.Inspect(d.decl, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok || bad != "" {
				return true
			}
			obj := pkg.TypesInfo.Uses[id]
			if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == pkg.PkgPath && obj.Parent() == pkg.Types.Scope() && !moved[obj] {
				if _, ok := obj.(*types.PkgName); !ok {
					bad = fmt.Sprintf("%s at %s", obj.Name(), filepath.Clean(pkg.Fset.Position(obj.Pos()).Filename))
				}
			}
			return bad == ""
		})
		if bad != "" {
			return fmt.Errorf("move_file refuses source-package dependency %q outside the moved declaration set", bad)
		}
	}
	return nil
}

func rejectMoveFileExternalPrivateReferences(pkgs []*packages.Package, source *packages.Package, objects map[moveFileObjectKey]types.Object, sourcePath, companionPath string) error {
	for _, pkg := range pkgs {
		if pkg.PkgPath != source.PkgPath {
			continue
		}
		for _, file := range pkg.Syntax {
			path := filepath.Clean(pkg.Fset.Position(file.Pos()).Filename)
			if path == sourcePath || path == companionPath {
				continue
			}
			if err := rejectMoveFileSourceFileDependencies(pkg, file, path, objects); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectMoveFileSourceFileDependencies(pkg *packages.Package, file *ast.File, path string, objects map[moveFileObjectKey]types.Object) error {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil && moveFileTypeIsMoved(pkg, fn, objects) {
			return fmt.Errorf("move_file refuses method %q declared outside the moved source file", fn.Name.Name)
		}
	}
	var dependency string
	ast.Inspect(file, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		obj := pkg.TypesInfo.Uses[id]
		if obj == nil {
			return true
		}
		pos := pkg.Fset.Position(obj.Pos())
		if _, moving := objects[moveFileObjectKey{filepath.Clean(pos.Filename), pos.Offset, obj.Name()}]; moving && !obj.Exported() {
			dependency = obj.Name()
			return false
		}
		return true
	})
	if dependency != "" {
		return fmt.Errorf("move_file refuses source-package dependency %q left in %s", dependency, path)
	}
	return nil
}

func moveFileTypeIsMoved(pkg *packages.Package, fn *ast.FuncDecl, objects map[moveFileObjectKey]types.Object) bool {
	id := astReceiverIdent(fn.Recv)
	if id == nil {
		return false
	}
	obj := pkg.TypesInfo.Uses[id]
	if obj == nil {
		return false
	}
	pos := pkg.Fset.Position(obj.Pos())
	_, ok := objects[moveFileObjectKey{filepath.Clean(pos.Filename), pos.Offset, obj.Name()}]
	return ok
}

func rejectMoveFilePathEffects(pkgs []*packages.Package, file *ast.File, companion string, files map[string][]byte) error {
	if moveFileHasEmbedDirective(file) {
		return fmt.Errorf("move_file refuses embed paths because their resolution would change")
	}
	if companion == "" {
		return nil
	}
	for path := range files {
		if path == companion {
			if err := rejectMoveFileTestPathEffects(pkgs, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func moveFileHasEmbedDirective(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(strings.TrimSpace(comment.Text), "//go:embed") {
				return true
			}
		}
	}
	return false
}

func moveFileHasLinkageDirective(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if strings.HasPrefix(text, "//go:linkname") || strings.HasPrefix(text, "//go:cgo_") || strings.HasPrefix(text, "//go:wasm") {
				return true
			}
		}
	}
	return false
}

func rejectMoveFileTestPathEffects(pkgs []*packages.Package, path string) error {
	var testPackage *packages.Package
	var testFile *ast.File
	for _, candidate := range pkgs {
		if file := syntaxForMoveFile(candidate, path); file != nil {
			testPackage = candidate
			testFile = file
			break
		}
	}
	if testPackage == nil || testFile == nil {
		return fmt.Errorf("move_file cannot identify matching test package for %s", path)
	}
	if moveFileHasEmbedDirective(testFile) {
		return fmt.Errorf("move_file refuses embed paths in matching tests")
	}
	if moveFileHasLinkageDirective(testFile) {
		return fmt.Errorf("move_file refuses compiler linkage directives in matching tests")
	}
	pathLiteral := moveFileTestPathReference(testPackage, testFile)
	if pathLiteral != "" {
		return fmt.Errorf("move_file refuses matching tests with relative fixture path %q in %s", pathLiteral, path)
	}
	return nil
}

func moveFileTestPathReference(pkg *packages.Package, file *ast.File) string {
	checkedFunctions := map[ast.Expr]bool{}
	var pathLiteral string
	ast.Inspect(file, func(node ast.Node) bool {
		if found := moveFileTestPathNodeReference(pkg, node, checkedFunctions); found != "" {
			pathLiteral = found
			return false
		}
		return true
	})
	return pathLiteral
}

func moveFileTestPathNodeReference(pkg *packages.Package, node ast.Node, checkedFunctions map[ast.Expr]bool) string {
	call, ok := node.(*ast.CallExpr)
	if ok {
		indices, recognized := moveFilePathCallArguments(pkg, call.Fun)
		if moveFileReadsWorkingDirectory(pkg, call.Fun, call.Args) {
			return "working directory environment read via " + moveFileCallName(call.Fun)
		}
		if recognized {
			checkedFunctions[call.Fun] = true
			if len(indices) == 0 && moveFileCallName(call.Fun) == "Getwd" {
				return "working directory read via Getwd"
			}
			for _, index := range indices {
				if index >= len(call.Args) {
					continue
				}
				lit, ok := call.Args[index].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return "dynamic path passed to " + moveFileCallName(call.Fun)
				}
				value, err := strconvUnquote(lit.Value)
				if err == nil && value != "" && !filepath.IsAbs(value) {
					return value
				}
			}
		}
	}
	if selector, ok := node.(*ast.SelectorExpr); ok && !checkedFunctions[selector] && moveFilePathSelector(pkg, selector) {
		return "filesystem function value " + selector.Sel.Name
	}
	return ""
}

func moveFilePathCallArguments(pkg *packages.Package, fun ast.Expr) ([]int, bool) {
	var object *types.Func
	var name string
	switch function := fun.(type) {
	case *ast.SelectorExpr:
		object, _ = pkg.TypesInfo.Uses[function.Sel].(*types.Func)
		name = function.Sel.Name
	case *ast.Ident:
		object, _ = pkg.TypesInfo.Uses[function].(*types.Func)
		name = function.Name
	default:
		return nil, false
	}
	if object == nil || object.Pkg() == nil {
		return nil, false
	}
	packagePath := object.Pkg().Path()
	if packagePath == "testing" && name == "Chdir" {
		return []int{0}, true
	}
	if packagePath != "os" && packagePath != "path/filepath" && packagePath != "path" && packagePath != "io/ioutil" && packagePath != "io/fs" && packagePath != "syscall" {
		return nil, false
	}
	pathArguments := map[string][]int{
		"Abs": {0}, "Chdir": {0}, "Create": {0}, "DirFS": {0}, "EvalSymlinks": {0}, "Glob": {0}, "Getwd": {},
		"Lstat": {0}, "Mkdir": {0}, "MkdirAll": {0}, "Open": {0}, "OpenFile": {0},
		"OpenRoot": {0}, "ReadDir": {0}, "ReadFile": {0}, "Remove": {0}, "RemoveAll": {0}, "Rename": {0, 1},
		"Rmdir": {0}, "Stat": {0}, "Symlink": {0, 1}, "Truncate": {0}, "Unlink": {0},
		"Walk": {0}, "WalkDir": {0}, "WriteFile": {0}, "Sub": {1},
	}
	indices, ok := pathArguments[name]
	if !ok {
		return nil, false
	}
	if packagePath == "io/fs" {
		switch name {
		case "ReadFile", "Stat", "ReadDir", "Glob", "WalkDir":
			indices = []int{1}
		}
	}
	if packagePath != "os" && name == "DirFS" {
		return nil, false
	}
	if packagePath == "io/fs" && name == "Sub" {
		indices = []int{1}
	}
	if packagePath == "os" && name == "DirFS" {
		indices = []int{0}
	}
	return indices, true
}

func moveFileReadsWorkingDirectory(pkg *packages.Package, fun ast.Expr, args []ast.Expr) bool {
	id, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	object, ok := pkg.TypesInfo.Uses[id.Sel].(*types.Func)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != "os" || object.Name() != "Getenv" && object.Name() != "LookupEnv" {
		return false
	}
	if len(args) == 0 {
		return true
	}
	key, ok := args[0].(*ast.BasicLit)
	if !ok || key.Kind != token.STRING {
		return true
	}
	value, err := strconvUnquote(key.Value)
	return err != nil || value == "PWD"
}

func moveFilePathSelector(pkg *packages.Package, selector *ast.SelectorExpr) bool {
	_, ok := moveFilePathCallArguments(pkg, selector)
	return ok
}

func moveFileCallName(fun ast.Expr) string {
	if selector, ok := fun.(*ast.SelectorExpr); ok {
		return selector.Sel.Name
	}
	return "filesystem API"
}

func validateMoveTestCompanion(pkgs []*packages.Package, path string, source *packages.Package) (*ast.File, error) {
	for _, pkg := range pkgs {
		if pkg.ID == pkg.PkgPath {
			continue
		}
		if f := syntaxForMoveFile(pkg, path); f != nil {
			if err := validateMoveTestCompanionFile(pkg, f, path, source); err != nil {
				return nil, err
			}
			return f, nil
		}
	}
	return nil, fmt.Errorf("matching test companion is not an active test file")
}

func validateMoveTestCompanionFile(pkg *packages.Package, file *ast.File, path string, source *packages.Package) error {
	if file.Name.Name != source.Name && file.Name.Name != source.Name+"_test" {
		return fmt.Errorf("matching test companion has incompatible package clause")
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "TestMain" && pkg.PkgPath == source.PkgPath {
			return fmt.Errorf("move_file refuses a shared TestMain in the source package")
		}
	}
	var helper string
	ast.Inspect(file, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		obj := pkg.TypesInfo.Uses[id]
		if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != pkg.PkgPath {
			return true
		}
		pos := pkg.Fset.Position(obj.Pos())
		if !strings.HasSuffix(pos.Filename, "_test.go") || filepath.Clean(pos.Filename) == filepath.Clean(path) {
			return true
		}
		helper = fmt.Sprintf("%s at %s", obj.Name(), filepath.Clean(pos.Filename))
		return false
	})
	if helper != "" {
		return fmt.Errorf("move_file refuses matching tests that depend on test helper %s", helper)
	}
	return nil
}

type moveFileExpectedReference struct {
	packageID, path                    string
	offset                             int
	name                               string
	marker                             string
	destinationImport, destinationPath string
}

type moveFilePlanInput struct {
	packages                              []*packages.Package
	files                                 map[string][]byte
	sourcePath, destinationPath           string
	companionSource, companionDestination string
	sourcePackage, destinationPackage     *packages.Package
	sourceSyntax                          *ast.File
	declarations                          []moveFileDecl
	objects                               map[moveFileObjectKey]types.Object
}

func planMoveFileChanges(input moveFilePlanInput) (map[string][]byte, []moveFileExpectedReference, error) {
	files := input.files
	sourcePath, destinationPath := input.sourcePath, input.destinationPath
	companionSource, companionDestination := input.companionSource, input.companionDestination
	sourcePkg, destPkg, sourceAST := input.sourcePackage, input.destinationPackage, input.sourceSyntax
	decls := input.declarations
	changes := map[string][]byte{}
	expectations := []moveFileExpectedReference{}
	nextMarker := 0
	newMarker := func(name string, packageID string, path string, offset int) string {
		marker := fmt.Sprintf("semeditMoveRef%d", nextMarker)
		nextMarker++
		expectations = append(expectations, moveFileExpectedReference{packageID: packageID, path: path, offset: offset, name: name, marker: marker})
		return marker
	}
	if err := planMoveFilePackageReferences(input, changes, newMarker); err != nil {
		return nil, nil, err
	}
	prod, err := buildMoveFileDestination(destinationPath, files[destinationPath], decls, files[sourcePath], sourceAST, sourcePkg, destPkg, newMarker)
	if err != nil {
		return nil, nil, err
	}
	changes[destinationPath] = prod
	shell, err := moveFilePackageShell(sourceAST.Name.Name, sourceAST, sourcePkg.Fset, files[sourcePath], decls)
	if err != nil {
		return nil, nil, err
	}
	changes[sourcePath] = shell
	if companionSource != "" {
		if err := planMoveFileCompanion(input, changes); err != nil {
			return nil, nil, err
		}
	}
	if err := validateMoveFileOutputPaths(changes, files, companionDestination); err != nil {
		return nil, nil, err
	}
	if err := finalizeMoveFileMarkers(changes, expectations, destPkg.PkgPath, destinationPath, companionSource, companionDestination); err != nil {
		return nil, nil, err
	}
	return changes, expectations, nil
}

func planMoveFilePackageReferences(input moveFilePlanInput, changes map[string][]byte, newMarker func(name, packageID, path string, offset int) string) error {
	pkgs := input.packages
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if err := planMoveFilePackageFile(input, pkg, file, changes, newMarker); err != nil {
				return err
			}
		}
	}
	return nil
}

func planMoveFilePackageFile(input moveFilePlanInput, pkg *packages.Package, file *ast.File, changes map[string][]byte, newMarker func(name, packageID, path string, offset int) string) error {
	files := input.files
	sourcePath, companionSource, companionDestination := input.sourcePath, input.companionSource, input.companionDestination
	sourcePkg, destPkg, objects := input.sourcePackage, input.destinationPackage, input.objects
	path := filepath.Clean(pkg.Fset.Position(file.Pos()).Filename)
	if _, ok := files[path]; !ok || path == sourcePath {
		return nil
	}
	if path == companionSource && file.Name.Name == sourcePkg.Name {
		var edits []moveFileTextEdit
		for _, id := range moveFileTypedReferenceIdents(pkg, file, objects) {
			start := pkg.Fset.Position(id.Pos()).Offset
			end := pkg.Fset.Position(id.End()).Offset
			marker := newMarker(id.Name, pkg.ID, companionDestination, start)
			edits = append(edits, moveFileTextEdit{start, end, marker})
		}
		output, err := rewriteMoveFileReferences(path, files[path], pkg, edits, "", "", "")
		if err != nil {
			return err
		}
		changes[path] = output
		return nil
	}
	var edits []moveFileTextEdit
	ast.Inspect(file, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if edit, ok := planMoveFileReferenceEdit(pkg, file, id, path, objects, sourcePkg, destPkg, newMarker); ok {
			edits = append(edits, edit)
			return false
		}
		return true
	})
	if len(edits) == 0 {
		return nil
	}
	alias := moveImportAlias(file, destPkg.PkgPath, destPkg.Name)
	output, err := rewriteMoveFileReferences(path, files[path], pkg, edits, sourcePkg.PkgPath, destPkg.PkgPath, alias)
	if err != nil {
		return err
	}
	if path == companionSource && file.Name.Name == sourcePkg.Name+"_test" && moveFileHasImport(output, sourcePkg.PkgPath) {
		return fmt.Errorf("move_file refuses matching external tests that retain a source-package dependency in %s", path)
	}
	if prior, ok := changes[path]; ok && !bytes.Equal(prior, output) {
		return fmt.Errorf("move_file found inconsistent typed test variants for %s", path)
	}
	changes[path] = output
	return nil
}

func planMoveFileCompanion(input moveFilePlanInput, changes map[string][]byte) error {
	files, companionSource, companionDestination := input.files, input.companionSource, input.companionDestination
	destPkg := input.destinationPackage
	testContent := files[companionSource]
	if rewritten, ok := changes[companionSource]; ok {
		testContent = rewritten
	}
	testFset := token.NewFileSet()
	testNode, parseErr := parser.ParseFile(testFset, companionSource, testContent, parser.ParseComments|parser.AllErrors)
	if parseErr != nil {
		return fmt.Errorf("parse matching test companion: %w", parseErr)
	}
	testPackageName := destPkg.Name
	if strings.HasSuffix(testNode.Name.Name, "_test") {
		testPackageName += "_test"
	}
	testNode.Name = ast.NewIdent(testPackageName)
	if testNode.Doc != nil {
		doc := testNode.Doc
		testNode.Comments = slices.DeleteFunc(testNode.Comments, func(group *ast.CommentGroup) bool { return group == doc })
		testNode.Doc = nil
	}
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), testNode); err != nil {
		return fmt.Errorf("format moved test companion: %w", err)
	}
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		return fmt.Errorf("format moved test companion: %w", err)
	}
	changes[companionDestination] = formatted
	originalTestFset := token.NewFileSet()
	originalTestNode, parseErr := parser.ParseFile(originalTestFset, companionSource, files[companionSource], parser.ParseComments|parser.AllErrors)
	if parseErr != nil {
		return fmt.Errorf("parse matching test companion: %w", parseErr)
	}
	allTestDecls := make([]moveFileDecl, 0, len(originalTestNode.Decls))
	for _, decl := range originalTestNode.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		start, end := moveDeclOffsets(originalTestFset, decl, originalTestNode, files[companionSource])
		allTestDecls = append(allTestDecls, moveFileDecl{start: start, end: end})
	}
	changes[companionSource], err = moveFilePackageShell(testPackageNameForSource(files[companionSource], companionSource), originalTestNode, originalTestFset, files[companionSource], allTestDecls)
	return err
}

func validateMoveFileOutputPaths(changes map[string][]byte, files map[string][]byte, companionDestination string) error {
	for path := range changes {
		if _, ok := files[path]; !ok && path != companionDestination {
			return fmt.Errorf("move_file output is outside module snapshot: %s", path)
		}
	}
	return nil
}

func finalizeMoveFileMarkers(changes map[string][]byte, expectations []moveFileExpectedReference, destinationImport, destinationPath, companionSource, companionDestination string) error {
	for i := range expectations {
		expectations[i].destinationImport = destinationImport
		expectations[i].destinationPath = destinationPath
		if expectations[i].path == companionSource {
			expectations[i].path = companionDestination
		}
	}
	for path, content := range changes {
		for i := range expectations {
			item := &expectations[i]
			if item.marker == "" || item.path != path {
				continue
			}
			at := bytes.Index(content, []byte(item.marker))
			if at < 0 || bytes.Contains(content[at+len(item.marker):], []byte(item.marker)) {
				return fmt.Errorf("move_file cannot uniquely locate rewritten reference %q in %s", item.name, path)
			}
			item.offset = at
			content = bytes.Replace(content, []byte(item.marker), []byte(item.name), 1)
			item.marker = ""
		}
		changes[path] = content
	}
	return nil
}

func planMoveFileReferenceEdit(pkg *packages.Package, file *ast.File, id *ast.Ident, path string, objects map[moveFileObjectKey]types.Object, sourcePkg, destPkg *packages.Package, newMarker func(name, packageID, path string, offset int) string) (moveFileTextEdit, bool) {
	obj := pkg.TypesInfo.Uses[id]
	if obj == nil || !ast.IsExported(obj.Name()) || moveFileMethodObject(obj) {
		return moveFileTextEdit{}, false
	}
	pos := pkg.Fset.Position(obj.Pos())
	if _, moving := objects[moveFileObjectKey{filepath.Clean(pos.Filename), pos.Offset, obj.Name()}]; !moving {
		return moveFileTextEdit{}, false
	}
	if pkg.PkgPath == destPkg.PkgPath {
		selector, ok := parentMoveSelector(file, id)
		if !ok || selector.Sel != id {
			return moveFileTextEdit{}, false
		}
		start, end := pkg.Fset.Position(selector.Pos()).Offset, pkg.Fset.Position(selector.End()).Offset
		return moveFileTextEdit{start, end, newMarker(obj.Name(), pkg.ID, path, start)}, true
	}
	if pkg.PkgPath == sourcePkg.PkgPath {
		start, end := pkg.Fset.Position(id.Pos()).Offset, pkg.Fset.Position(id.End()).Offset
		if selector, ok := parentMoveSelector(file, id); ok && selector.Sel == id {
			start, end = pkg.Fset.Position(selector.Pos()).Offset, pkg.Fset.Position(selector.End()).Offset
		}
		alias := moveImportAlias(file, destPkg.PkgPath, destPkg.Name)
		return moveFileTextEdit{start, end, alias + "." + newMarker(obj.Name(), pkg.ID, path, start)}, true
	}
	selector, ok := parentMoveSelector(file, id)
	if !ok || selector.Sel != id {
		return moveFileTextEdit{}, false
	}
	start, end := pkg.Fset.Position(selector.Pos()).Offset, pkg.Fset.Position(selector.End()).Offset
	alias := moveImportAlias(file, destPkg.PkgPath, destPkg.Name)
	return moveFileTextEdit{start, end, alias + "." + newMarker(obj.Name(), pkg.ID, path, start)}, true
}

func moveFileHasImport(content []byte, importPath string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "matching_test.go", content, parser.ImportsOnly)
	if err != nil {
		return true
	}
	for _, spec := range file.Imports {
		path, err := strconvUnquote(spec.Path.Value)
		if err == nil && path == importPath {
			return true
		}
	}
	return false
}

func moveFileTypedReferenceIdents(pkg *packages.Package, file *ast.File, objects map[moveFileObjectKey]types.Object) []*ast.Ident {
	var refs []*ast.Ident
	ast.Inspect(file, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		obj := pkg.TypesInfo.Uses[id]
		if obj == nil {
			return true
		}
		if moveFileMethodObject(obj) {
			return true
		}
		pos := pkg.Fset.Position(obj.Pos())
		if _, moving := objects[moveFileObjectKey{filepath.Clean(pos.Filename), pos.Offset, obj.Name()}]; moving {
			refs = append(refs, id)
		}
		return true
	})
	return refs
}

func testPackageNameForSource(content []byte, path string) string {
	file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.PackageClauseOnly)
	if err != nil || file.Name == nil {
		return ""
	}
	return file.Name.Name
}

type moveFileTextEdit struct {
	start, end int
	text       string
}

func parentMoveSelector(file *ast.File, id *ast.Ident) (*ast.SelectorExpr, bool) {
	var found *ast.SelectorExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok && s.Sel == id {
			found = s
			return false
		}
		return found == nil
	})
	return found, found != nil
}
func rewriteMoveFileReferences(path string, content []byte, pkg *packages.Package, edits []moveFileTextEdit, sourceImport, destImport, destAlias string) ([]byte, error) {
	for _, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > len(content) {
			return nil, fmt.Errorf("invalid typed reference bounds in %s", path)
		}
	}
	// Apply byte edits in descending order so original go/types positions remain valid.
	slices.SortFunc(edits, func(a, b moveFileTextEdit) int { return b.start - a.start })
	out := bytes.Clone(content)
	last := len(out) + 1
	for _, e := range edits {
		if e.end > last {
			return nil, fmt.Errorf("overlapping typed move references in %s", path)
		}
		out = append(append(bytes.Clone(out[:e.start]), []byte(e.text)...), out[e.end:]...)
		last = e.start
	}
	fset := token.NewFileSet()
	node, parseErr := parser.ParseFile(fset, path, out, parser.ParseComments|parser.AllErrors)
	if parseErr != nil {
		return nil, fmt.Errorf("parse rewritten references in %s: %w", path, parseErr)
	}
	if destImport != "" && pkg.PkgPath != destImport {
		if err := setMoveImport(node, destImport, destAlias, destAlias); err != nil {
			return nil, fmt.Errorf("add destination import in %s: %w", path, err)
		}
	}
	if err := removeUnusedMoveFileImport(node, pkg, sourceImport); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := format.Node(&b, fset, node); err != nil {
		return nil, fmt.Errorf("format references in %s: %w", path, err)
	}
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format references in %s: %w", path, err)
	}
	return formatted, nil
}

func removeUnusedMoveFileImport(file *ast.File, pkg *packages.Package, importPath string) error {
	for _, spec := range slices.Clone(file.Imports) {
		path, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return err
		}
		if path != importPath {
			continue
		}
		alias := ""
		if spec.Name != nil {
			alias = spec.Name.Name
		} else if imported := pkg.Imports[importPath]; imported != nil {
			alias = imported.Name
		}
		if alias == "" {
			return fmt.Errorf("cannot resolve imported package name for %q", importPath)
		}
		if !moveFileImportAliasUsed(file, alias) {
			if err := deleteMoveFileImport(file, importPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func moveFileImportAliasUsed(file *ast.File, alias string) bool {
	used := false
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := selector.X.(*ast.Ident)
		if ok && id.Name == alias {
			used = true
			return false
		}
		return !used
	})
	return used
}

func deleteMoveFileImport(file *ast.File, importPath string) error {
	decls := file.Decls[:0]
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.IMPORT {
			decls = append(decls, decl)
			continue
		}
		specs := group.Specs[:0]
		for _, item := range group.Specs {
			spec, ok := item.(*ast.ImportSpec)
			if !ok {
				return fmt.Errorf("unexpected non-import specification")
			}
			path, err := strconvUnquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if path != importPath {
				specs = append(specs, item)
			}
		}
		group.Specs = specs
		if len(group.Specs) > 0 {
			decls = append(decls, group)
		}
	}
	file.Decls = decls
	rebuildMoveImportIndex(file)
	return nil
}

func buildMoveFileDestination(path string, base []byte, decls []moveFileDecl, source []byte, sourceAST *ast.File, sourcePkg, destPkg *packages.Package, newMarker func(name, packageID, path string, offset int) string) ([]byte, error) {
	var b bytes.Buffer
	b.Write(base)
	if len(base) > 0 && !bytes.HasSuffix(base, []byte("\n")) {
		b.WriteByte('\n')
	}
	for _, d := range decls {
		b.WriteByte('\n')
		declaration, err := rewriteMoveFileDestinationSelfReferences(d, source, sourcePkg, destPkg, path, newMarker)
		if err != nil {
			return nil, err
		}
		b.Write(declaration)
		if !bytes.HasSuffix(declaration, []byte("\n")) {
			b.WriteByte('\n')
		}
	}
	fset := token.NewFileSet()
	node, parseErr := parser.ParseFile(fset, path, b.Bytes(), parser.ParseComments|parser.AllErrors)
	if parseErr != nil {
		return nil, fmt.Errorf("parse moved destination declarations: %w", parseErr)
	}
	if err := moveFileRetainUsedImports(node, decls, sourceAST, sourcePkg, destPkg); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, node); err != nil {
		return nil, fmt.Errorf("format moved destination: %w", err)
	}
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format moved destination: %w", err)
	}
	return formatted, nil
}

func moveFileRetainUsedImports(destination *ast.File, decls []moveFileDecl, sourceAST *ast.File, sourcePkg, destPkg *packages.Package) error {
	used := moveFileImportQualifierUses(decls)
	for _, spec := range sourceAST.Imports {
		imp, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return err
		}
		if imp == destPkg.PkgPath {
			continue
		}
		local := spec.Name
		if local != nil && (local.Name == "." || local.Name == "_") {
			if used[local.Name] {
				return fmt.Errorf("move_file refuses dot or blank imports used by moved declarations")
			}
			continue
		}
		name := ""
		if local != nil {
			name = local.Name
		} else if pkgName, ok := sourcePkg.TypesInfo.Implicits[spec].(*types.PkgName); ok {
			name = pkgName.Imported().Name()
		}
		if name != "" && used[name] {
			if err := setMoveImport(destination, imp, name, name); err != nil {
				return err
			}
		}
	}
	return nil
}

func moveFileImportQualifierUses(decls []moveFileDecl) map[string]bool {
	used := map[string]bool{}
	for _, decl := range decls {
		ast.Inspect(decl.decl, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := selector.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
			return true
		})
	}
	return used
}

func rewriteMoveFileDestinationSelfReferences(declaration moveFileDecl, source []byte, sourcePkg, destPkg *packages.Package, path string, newMarker func(name, packageID, path string, offset int) string) ([]byte, error) {
	var edits []moveFileTextEdit
	ast.Inspect(declaration.decl, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		obj := sourcePkg.TypesInfo.Uses[id]
		if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != destPkg.PkgPath || !ast.IsExported(obj.Name()) {
			return true
		}
		var selector *ast.SelectorExpr
		ast.Inspect(declaration.decl, func(n ast.Node) bool {
			candidate, ok := n.(*ast.SelectorExpr)
			if ok && candidate.Sel == id {
				selector = candidate
				return false
			}
			return selector == nil
		})
		if selector == nil {
			return true
		}
		if _, ok := selector.X.(*ast.Ident); !ok {
			return true
		}
		start := sourcePkg.Fset.Position(selector.Pos()).Offset - declaration.start
		end := sourcePkg.Fset.Position(selector.End()).Offset - declaration.start
		marker := newMarker(obj.Name(), destPkg.ID, path, start)
		edits = append(edits, moveFileTextEdit{start: start, end: end, text: marker})
		return true
	})
	content := bytes.Clone(source[declaration.start:declaration.end])
	slices.SortFunc(edits, func(a, b moveFileTextEdit) int { return b.start - a.start })
	last := len(content) + 1
	for _, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > len(content) || edit.end > last {
			return nil, fmt.Errorf("invalid or overlapping destination self-reference in %s", path)
		}
		content = append(append(bytes.Clone(content[:edit.start]), []byte(edit.text)...), content[edit.end:]...)
		last = edit.start
	}
	return content, nil
}

func moveFilePackageShell(name string, file *ast.File, fset *token.FileSet, source []byte, moved []moveFileDecl) ([]byte, error) {
	if !token.IsIdentifier(name) {
		return nil, fmt.Errorf("invalid package name in move_file source")
	}
	var shell bytes.Buffer
	if file != nil && file.Doc != nil {
		start, end := fset.Position(file.Doc.Pos()).Offset, fset.Position(file.Doc.End()).Offset
		if start >= 0 && end >= start && end <= len(source) {
			shell.Write(source[start:end])
			shell.WriteByte('\n')
		}
	}
	shell.WriteString("package " + name + "\n")
	if file != nil {
		for _, group := range file.Comments {
			if group == file.Doc {
				continue
			}
			start, end := fset.Position(group.Pos()).Offset, fset.Position(group.End()).Offset
			included := false
			for _, decl := range moved {
				if start >= decl.start && end <= decl.end {
					included = true
					break
				}
			}
			if included || start < 0 || end < start || end > len(source) {
				continue
			}
			shell.WriteByte('\n')
			shell.Write(source[start:end])
			shell.WriteByte('\n')
		}
	}
	return shell.Bytes(), nil
}
func validateMoveFileReferences(pkgs []*packages.Package, expected []moveFileExpectedReference) error {
	for _, item := range expected {
		pkg := packageByID(pkgs, item.packageID)
		var syntax *ast.File
		if pkg != nil {
			syntax = syntaxFileForReference(pkg, item.path)
		}
		if syntax == nil {
			for _, candidate := range pkgs {
				if candidateSyntax := syntaxFileForReference(candidate, item.path); candidateSyntax != nil {
					pkg, syntax = candidate, candidateSyntax
					break
				}
			}
		}
		if pkg == nil || syntax == nil {
			return fmt.Errorf("move_file cannot validate references in %s", item.path)
		}
		var matched *ast.Ident
		ast.Inspect(syntax, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok || pkg.Fset.Position(id.Pos()).Offset != item.offset || id.Name != item.name {
				return true
			}
			matched = id
			return true
		})
		if matched == nil {
			return fmt.Errorf("move_file refuses rewritten reference %q in %s because its exact location changed", item.name, item.path)
		}
		resolved := pkg.TypesInfo.Uses[matched]
		if resolved == nil || resolved.Name() != item.name || resolved.Pkg() == nil || resolved.Pkg().Path() != item.destinationImport || resolved.Pkg().Scope().Lookup(item.name) != resolved || filepath.Clean(pkg.Fset.Position(resolved.Pos()).Filename) != filepath.Clean(item.destinationPath) {
			return fmt.Errorf("move_file refuses rewritten reference %q in %s because it no longer resolves to the moved declaration", item.name, item.path)
		}
	}
	return nil
}

func applyMoveFileChanges(ctx context.Context, original, changes map[string][]byte) (string, error) {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return "", fmt.Errorf("move_file requires a source buffer")
	}
	paths := slices.Sorted(maps.Keys(changes))
	var diff strings.Builder
	batch := make([]sourcebuf.Change, 0, len(changes))
	for _, path := range paths {
		before, ok := original[path]
		if !ok {
			batch = append(batch, sourcebuf.Change{Path: path, After: changes[path], Create: true})
			if d := UnifiedDiff(path, "", string(changes[path])); d != "" {
				if diff.Len() > 0 {
					diff.WriteByte('\n')
				}
				diff.WriteString(d)
			}
			continue
		}
		if !bytes.Equal(before, changes[path]) {
			batch = append(batch, sourcebuf.Change{Path: path, Before: before, After: changes[path]})
			if d := UnifiedDiff(path, string(before), string(changes[path])); d != "" {
				if diff.Len() > 0 {
					diff.WriteByte('\n')
				}
				diff.WriteString(d)
			}
		}
	}
	if err := buffer.Apply(ctx, batch); err != nil {
		return "", fmt.Errorf("apply move_file changes: %w", err)
	}
	return diff.String(), nil
}
