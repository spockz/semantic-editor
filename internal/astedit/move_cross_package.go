// Package astedit isolates cross-package relocation here because its module-wide preflight and reference rewrite have different safety requirements from same-package moves.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"errors"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"semedit/internal/gocache"
	"semedit/internal/sourcebuf"
	"slices"
)

// moveConstructAcrossPackages moves one exported free function inside a module
// and rewrites references only when go/packages can prove their identities.
//
//nolint:funlen,gocognit // Ordered compiler preflight and publication share one immutable plan with explicit refusal points.
func moveConstructAcrossPackages(ctx context.Context, sourcePath, destinationPath, symbolQuery string, opts MoveConstructOptions) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	sourcePath, destinationPath = filepath.Clean(sourcePath), filepath.Clean(destinationPath)
	if strings.Contains(symbolQuery, ".") || strings.HasPrefix(symbolQuery, "(") || !token.IsIdentifier(symbolQuery) || !ast.IsExported(symbolQuery) {
		return "", fmt.Errorf("cross-package move requires an exported free function name")
	}
	moduleRoot, err := moveModuleRoot(sourcePath)
	if err != nil {
		return "", err
	}
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}
	destinationPath, err = filepath.Abs(destinationPath)
	if err != nil {
		return "", fmt.Errorf("resolve destination path: %w", err)
	}
	if !moveWithin(moduleRoot, destinationPath) {
		return "", fmt.Errorf("cross-package move requires source and destination in the same Go module")
	}
	if pathErr := rejectMoveSymlinkPath(moduleRoot, sourcePath); pathErr != nil {
		return "", pathErr
	}
	if pathErr := rejectMoveSymlinkPath(moduleRoot, destinationPath); pathErr != nil {
		return "", pathErr
	}
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if authorizationErr := buffer.AuthorizeWrites(ctx, moduleRoot); authorizationErr != nil {
			return "", authorizationErr
		}
	}
	original, err := inspectMoveModuleFiles(ctx, moduleRoot, sourcePath, destinationPath, symbolQuery)
	if err != nil {
		return "", err
	}
	for _, path := range []string{sourcePath, destinationPath} {
		if _, ok := original[path]; !ok {
			return "", fmt.Errorf("source and destination must be existing Go files in the selected module: %s", path)
		}
	}

	pkgs, err := loadMovePackages(ctx, moduleRoot, nil)
	if err != nil {
		return "", fmt.Errorf("load Go module packages: %w", err)
	}
	if len(pkgs) == 0 {
		return "", fmt.Errorf("load Go module packages: no packages found")
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			return "", fmt.Errorf("cannot establish complete Go references: package %s: %s", pkg.PkgPath, pkg.Errors[0].Msg)
		}
	}
	if referenceErr := rejectUntypedMoveReferences(original, pkgs, symbolQuery); referenceErr != nil {
		return "", referenceErr
	}
	sourcePkg, destinationPkg := packageForMoveFile(pkgs, sourcePath), packageForMoveFile(pkgs, destinationPath)
	if sourcePkg == nil || destinationPkg == nil {
		return "", fmt.Errorf("source and destination must be active Go files in the selected module")
	}
	if sourcePkg.ID == destinationPkg.ID || sourcePkg.PkgPath == destinationPkg.PkgPath {
		return "", fmt.Errorf("cross-package move requires distinct Go packages")
	}
	if sourcePkg.Module == nil || destinationPkg.Module == nil || !sourcePkg.Module.Main || !destinationPkg.Module.Main || sourcePkg.Module.Path != destinationPkg.Module.Path {
		return "", fmt.Errorf("cross-package move requires source and destination in the same Go module")
	}
	if strings.HasSuffix(sourcePath, "_test.go") || strings.HasSuffix(destinationPath, "_test.go") {
		return "", fmt.Errorf("cross-package move does not support test files")
	}
	sourceFile := syntaxForMoveFile(sourcePkg, sourcePath)
	destinationFile := syntaxForMoveFile(destinationPkg, destinationPath)
	if sourceFile == nil || destinationFile == nil {
		return "", fmt.Errorf("source and destination must be active Go files")
	}
	decl, targetObj := findExportedMoveFunction(sourcePkg, sourceFile, sourcePath, symbolQuery)
	if decl == nil || targetObj == nil {
		return "", &symbol.SymbolError{Op: "move_construct", File: sourcePath, Symbol: symbolQuery, Err: symbol.ErrNotFound}
	}
	if decl.Recv != nil {
		return "", fmt.Errorf("cross-package move does not support receiver methods")
	}
	if !ast.IsExported(decl.Name.Name) {
		return "", fmt.Errorf("cross-package move requires an exported function")
	}
	if declarationNameCollision(destinationFile, decl) {
		return "", fmt.Errorf("destination already declares %q", symbolQuery)
	}
	if dependencyErr := rejectSourceLocalMoveDependencies(sourcePkg, decl); dependencyErr != nil {
		return "", dependencyErr
	}
	start, end := moveDeclOffsets(sourcePkg.Fset, decl, sourceFile, original[sourcePath])
	if start < 0 || end < start || end > len(original[sourcePath]) {
		return "", fmt.Errorf("invalid source declaration bounds for %q", symbolQuery)
	}
	declText := original[sourcePath][start:end]
	neededImports, err := moveImportsForTypedDeclaration(decl, sourceFile, sourcePkg)
	if err != nil {
		return "", err
	}
	targetPos := sourcePkg.Fset.Position(targetObj.Pos())
	refs, destinationRefs, err := rewriteMoveReferences(pkgs, sourcePkg.PkgPath, targetObj.Name(), sourcePath, targetPos.Offset, start, end, destinationPath, destinationPkg.PkgPath, destinationPkg.Name)
	if err != nil {
		return "", err
	}
	changes := refs
	destinationBase := original[destinationPath]
	if rewritten, ok := changes[destinationPath]; ok {
		destinationBase = rewritten
	}
	destinationFset := token.NewFileSet()
	destinationNode, err := parser.ParseFile(destinationFset, destinationPath, destinationBase, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse destination after reference rewrites: %w", err)
	}
	insertOffset, err := calculateMoveOffset(destinationFset, destinationNode, destinationBase, opts)
	if err != nil {
		return "", err
	}
	var destinationBuffer bytes.Buffer
	destinationBuffer.Write(destinationBase[:insertOffset])
	if insertOffset > 0 && !bytes.HasSuffix(destinationBuffer.Bytes(), []byte("\n\n")) {
		if bytes.HasSuffix(destinationBuffer.Bytes(), []byte("\n")) {
			destinationBuffer.WriteByte('\n')
		} else {
			destinationBuffer.WriteString("\n\n")
		}
	}
	destinationBuffer.Write(declText)
	destinationRest := bytes.TrimLeft(destinationBase[insertOffset:], "\r\n")
	if len(destinationRest) == 0 {
		destinationBuffer.WriteByte('\n')
	} else {
		destinationBuffer.WriteString("\n\n")
	}
	destinationBuffer.Write(destinationRest)
	formattedDestination, err := format.Source(destinationBuffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("format destination after move: %w", err)
	}
	destinationFset = token.NewFileSet()
	destinationNode, err = parser.ParseFile(destinationFset, destinationPath, formattedDestination, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse destination after declaration insertion: %w", err)
	}
	if importErr := mergeMoveImports(destinationNode, neededImports); importErr != nil {
		return "", importErr
	}
	var mergedDestination bytes.Buffer
	if formatErr := format.Node(&mergedDestination, destinationFset, destinationNode); formatErr != nil {
		return "", fmt.Errorf("format destination imports: %w", formatErr)
	}
	formattedDestination, err = format.Source(mergedDestination.Bytes())
	if err != nil {
		return "", fmt.Errorf("format destination imports: %w", err)
	}
	if _, sourceParseErr := parser.ParseFile(token.NewFileSet(), sourcePath, changes[sourcePath], parser.ParseComments|parser.AllErrors); sourceParseErr != nil {
		return "", fmt.Errorf("validate source after move: %w", sourceParseErr)
	}
	if _, destinationParseErr := parser.ParseFile(token.NewFileSet(), destinationPath, formattedDestination, parser.ParseComments|parser.AllErrors); destinationParseErr != nil {
		return "", fmt.Errorf("validate destination after move: %w", destinationParseErr)
	}
	changes[destinationPath] = formattedDestination
	expectedDestinationReferences := map[moveReferenceKey][]moveReferenceExpectation{}
	for key, markers := range destinationRefs {
		content := changes[key.filePath]
		for _, marker := range markers {
			offset := bytes.Index(content, []byte(marker.sentinel))
			if offset < 0 {
				return "", fmt.Errorf("locate rewritten destination reference in %s", key.filePath)
			}
			expectedDestinationReferences[key] = append(expectedDestinationReferences[key], moveReferenceExpectation{offset: offset, name: marker.replacement, kind: marker.kind})
			content = bytes.Replace(content, []byte(marker.sentinel), []byte(marker.replacement), 1)
		}
		changes[key.filePath] = content
	}
	for path := range changes {
		if _, ok := original[path]; !ok {
			return "", fmt.Errorf("reference file is outside the preflight module snapshot: %s", path)
		}
	}
	overlayPackages, err := typecheckMoveOverlay(ctx, moduleRoot, changes)
	if err != nil {
		return "", err
	}
	if err := validateMoveDestinationReferences(overlayPackages, destinationPkg.PkgPath, destinationPath, targetObj.Name(), expectedDestinationReferences); err != nil {
		return "", err
	}
	paths := make([]string, 0, len(changes))
	for path := range changes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if sourcebuf.FromContext(ctx) != nil {
		return applyBufferedMove(ctx, original, changes)
	}
	// Compare all snapshots immediately before publication. New declaration first,
	// references second, and source removal last keeps every partial state explicit.
	for path := range changes {
		before := original[path]
		// #nosec G304 -- each changed path was resolved from the preflight module snapshot.
		current, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("recheck file before move publication %s: %w", path, readErr)
		}
		if !bytes.Equal(current, before) {
			return "", fmt.Errorf("file changed during move preflight: %s", path)
		}
	}
	order := publicationOrder(paths, destinationPath, sourcePath)
	written := make([]string, 0, len(order))
	for _, path := range order {
		if err := pipeline.WriteAtomic(path, changes[path]); err != nil {
			if len(written) != 0 {
				return "", fmt.Errorf("partial cross-package move: wrote %s; failed writing %s: %w", strings.Join(written, ", "), path, err)
			}
			return "", fmt.Errorf("write move file %s atomically: %w", path, err)
		}
		written = append(written, path)
	}
	var diffs []string
	for _, path := range paths {
		persisted, readErr := os.ReadFile(path) // #nosec G304 -- path is a validated in-module Go file from the preflight snapshot.
		if readErr != nil {
			return "", fmt.Errorf("cross-package move wrote %s but could not read persisted result %s: %w", strings.Join(written, ", "), path, readErr)
		}
		diff := UnifiedDiff(path, string(original[path]), string(persisted))
		if diff != "" {
			diffs = append(diffs, diff)
		}
	}
	return strings.Join(diffs, "\n"), nil
}

func rejectUntypedMoveReferences(snapshot map[string][]byte, pkgs []*packages.Package, symbolName string) error {
	loaded := map[string]bool{}
	for _, pkg := range pkgs {
		for _, path := range append(append([]string(nil), pkg.GoFiles...), pkg.CompiledGoFiles...) {
			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("resolve loaded Go file %s: %w", path, err)
			}
			loaded[filepath.Clean(abs)] = true
		}
	}
	for path, content := range snapshot {
		if loaded[filepath.Clean(path)] || !containsMoveIdentifier(content, symbolName) {
			continue
		}
		return fmt.Errorf("cross-package move cannot prove reference completeness: untyped Go file %s mentions %q", path, symbolName)
	}
	return nil
}

func moveModuleRoot(sourcePath string) (string, error) {
	abs, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}
	for dir := filepath.Dir(abs); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect module root: %w", err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("cross-package move requires a Go module")
}

func moveWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func containsMoveIdentifier(source []byte, name string) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("move-scan.go", -1, len(source))
	var scanner scanner.Scanner
	scanner.Init(file, source, nil, 0)
	for {
		_, tok, literal := scanner.Scan()
		if tok == token.EOF {
			return false
		}
		if tok == token.IDENT && literal == name {
			return true
		}
	}
}

func rejectMoveSymlinkPath(root, path string) error {
	if !moveWithin(root, path) {
		return fmt.Errorf("cross-package move path escapes module: %s", path)
	}
	for current := path; moveWithin(root, current); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect move path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("cross-package move refuses symlink path %s", current)
		}
		if current == root {
			break
		}
	}
	return nil
}

func packageForMoveFile(pkgs []*packages.Package, path string) *packages.Package {
	for _, pkg := range pkgs {
		if pkg.Module == nil || !pkg.Module.Main || pkg.ID != pkg.PkgPath {
			continue
		}
		for _, file := range pkg.CompiledGoFiles {
			abs, _ := filepath.Abs(file)
			if filepath.Clean(abs) == filepath.Clean(path) {
				return pkg
			}
		}
	}
	return nil
}

func syntaxForMoveFile(pkg *packages.Package, path string) *ast.File {
	for _, file := range pkg.Syntax {
		if pkg.Fset.Position(file.Pos()).Filename == path {
			return file
		}
	}
	return nil
}

func findExportedMoveFunction(pkg *packages.Package, file *ast.File, path, name string) (*ast.FuncDecl, *types.Func) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != name {
			continue
		}
		obj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func)
		if ok && obj.Pkg() != nil && obj.Pkg().Path() == pkg.PkgPath && pkg.Fset.Position(obj.Pos()).Filename == path {
			return fn, obj
		}
	}
	return nil, nil
}

func rejectSourceLocalMoveDependencies(pkg *packages.Package, decl *ast.FuncDecl) error {
	var reason string
	ast.Inspect(decl, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok || reason != "" {
			return true
		}
		obj := pkg.TypesInfo.Uses[ident]
		if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != pkg.PkgPath || obj.Parent() != pkg.Types.Scope() {
			return true
		}
		if obj == pkg.TypesInfo.Defs[decl.Name] {
			return true
		}
		if _, ok := obj.(*types.PkgName); ok {
			return true
		}
		reason = fmt.Sprintf("cross-package move refuses source-package dependency %q", obj.Name())
		return false
	})
	if reason != "" {
		return fmt.Errorf("%s", reason)
	}
	return nil
}

func moveImportsForTypedDeclaration(decl ast.Decl, source *ast.File, pkg *packages.Package) (*ast.File, error) {
	used := map[string]bool{}
	ast.Inspect(decl, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := selector.X.(*ast.Ident); ok {
				used[ident.Name] = true
			}
		}
		return true
	})
	imports := &ast.File{}
	for _, spec := range source.Imports {
		path, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("decode source import: %w", err)
		}
		if spec.Name != nil && spec.Name.Name == "." {
			return nil, fmt.Errorf("cannot move declaration from a file with a dot import")
		}
		if spec.Name != nil && spec.Name.Name == "_" {
			continue
		}
		var local string
		var alias *ast.Ident
		if spec.Name != nil {
			local, alias = spec.Name.Name, ast.NewIdent(spec.Name.Name)
		} else if imported, ok := pkg.TypesInfo.Implicits[spec].(*types.PkgName); ok {
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

type moveReferenceKey struct {
	packageID string
	filePath  string
}

type moveReferenceKind uint8

const (
	moveReferenceFunction moveReferenceKind = iota + 1
	moveReferenceImport
)

type moveReferenceMarker struct {
	sentinel    string
	replacement string
	kind        moveReferenceKind
}

type moveReferenceExpectation struct {
	offset int
	name   string
	kind   moveReferenceKind
}

type moveReferenceEdits struct {
	alias   string
	markers []moveReferenceMarker
	changed bool
}

//nolint:gocognit // Typed selector and dot-import references need distinct rewrites under the same package-scope plan.
func rewriteMoveReferences(pkgs []*packages.Package, sourceImport, targetName, sourcePath string, targetOffset, declarationStart, declarationEnd int, destinationPath, destinationImport, destinationName string) (map[string][]byte, map[moveReferenceKey][]moveReferenceMarker, error) {
	changes := map[string][]byte{}
	destinationReferences := map[moveReferenceKey][]moveReferenceMarker{}
	sentinelSequence := 0
	for _, pkg := range pkgs {
		if pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		for _, file := range pkg.Syntax {
			path := filepath.Clean(pkg.Fset.Position(file.Pos()).Filename)
			if pkg.Module.Dir == "" || !moveWithin(pkg.Module.Dir, path) {
				continue
			}
			if !strings.HasSuffix(path, "_test.go") && pkg.ID != pkg.PkgPath {
				continue
			}
			if (path == sourcePath || path == destinationPath) && pkg.ID != pkg.PkgPath {
				continue
			}
			fileChanged := false
			if path == sourcePath {
				removeMovedFunction(file, pkg.Fset, targetName, targetOffset, declarationStart, declarationEnd)
				fileChanged = true
			}
			edits := collectMoveReferenceEdits(pkg, file, sourceImport, targetName, sourcePath, targetOffset, destinationImport, destinationName, &sentinelSequence)
			alias, referenceMarkers := edits.alias, edits.markers
			fileChanged = fileChanged || edits.changed
			if !fileChanged {
				continue
			}
			if alias != "" {
				if err := setMoveImport(file, destinationImport, alias, destinationName); err != nil {
					return nil, nil, fmt.Errorf("update imports in %s: %w", path, err)
				}
			}
			reserved := map[string]bool{}
			for _, marker := range referenceMarkers {
				if marker.kind == moveReferenceImport {
					reserved[marker.replacement] = true
				}
			}
			if err := removeUnusedMoveImports(file, pkg.TypesInfo, destinationImport, destinationName, reserved); err != nil {
				return nil, nil, fmt.Errorf("clean imports in %s: %w", path, err)
			}
			var out bytes.Buffer
			if err := format.Node(&out, pkg.Fset, file); err != nil {
				return nil, nil, fmt.Errorf("format references in %s: %w", path, err)
			}
			formatted, err := format.Source(out.Bytes())
			if err != nil {
				return nil, nil, fmt.Errorf("format reference file %s: %w", path, err)
			}
			destinationReferences[moveReferenceKey{packageID: pkg.ID, filePath: path}] = append(destinationReferences[moveReferenceKey{packageID: pkg.ID, filePath: path}], referenceMarkers...)
			changes[path] = formatted
		}
	}
	return changes, destinationReferences, nil
}

func collectMoveReferenceEdits(pkg *packages.Package, file *ast.File, sourceImport, targetName, sourcePath string, targetOffset int, destinationImport, destinationName string, sentinelSequence *int) moveReferenceEdits {
	var edits moveReferenceEdits
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if selector, ok := cursor.Node().(*ast.SelectorExpr); ok && isTargetMoveReference(pkg, selector.Sel, sourceImport, targetName, sourcePath, targetOffset) {
			if pkg.PkgPath == destinationImport {
				sentinel := nextMoveReferenceSentinel(file, sentinelSequence)
				edits.markers = append(edits.markers, moveReferenceMarker{sentinel: sentinel, replacement: targetName, kind: moveReferenceFunction})
				cursor.Replace(ast.NewIdent(sentinel))
			} else {
				if edits.alias == "" {
					edits.alias = moveImportAlias(file, destinationImport, destinationName)
				}
				sentinel := nextMoveReferenceSentinel(file, sentinelSequence)
				edits.markers = append(edits.markers, moveReferenceMarker{sentinel: sentinel, replacement: edits.alias, kind: moveReferenceImport})
				selector.X = ast.NewIdent(sentinel)
			}
			edits.changed = true
			return false
		}
		ident, ok := cursor.Node().(*ast.Ident)
		if !ok || !isTargetMoveReference(pkg, ident, sourceImport, targetName, sourcePath, targetOffset) {
			return true
		}
		if _, selectorName := cursor.Parent().(*ast.SelectorExpr); selectorName || pkg.PkgPath == destinationImport {
			return true
		}
		if edits.alias == "" {
			edits.alias = moveImportAlias(file, destinationImport, destinationName)
		}
		sentinel := nextMoveReferenceSentinel(file, sentinelSequence)
		edits.markers = append(edits.markers, moveReferenceMarker{sentinel: sentinel, replacement: edits.alias, kind: moveReferenceImport})
		cursor.Replace(&ast.SelectorExpr{X: ast.NewIdent(sentinel), Sel: ast.NewIdent(ident.Name)})
		edits.changed = true
		return false
	}, nil)
	return edits
}

func nextMoveReferenceSentinel(file *ast.File, sequence *int) string {
	for {
		candidate := fmt.Sprintf("semeditMoveReference%d", *sequence)
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

func isTargetMoveReference(pkg *packages.Package, ident *ast.Ident, sourceImport, targetName, sourcePath string, targetOffset int) bool {
	obj := pkg.TypesInfo.Uses[ident]
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != sourceImport || obj.Name() != targetName {
		return false
	}
	position := pkg.Fset.Position(obj.Pos())
	return filepath.Clean(position.Filename) == filepath.Clean(sourcePath) && position.Offset == targetOffset
}

func removeMovedFunction(file *ast.File, fset *token.FileSet, name string, nameOffset, declarationStart, declarationEnd int) {
	decls := file.Decls[:0]
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == name && fset.Position(fn.Name.Pos()).Offset == nameOffset {
			continue
		}
		decls = append(decls, declaration)
	}
	file.Decls = decls
	comments := file.Comments[:0]
	for _, group := range file.Comments {
		start := fset.Position(group.Pos()).Offset
		end := fset.Position(group.End()).Offset
		if start >= declarationStart && end <= declarationEnd {
			continue
		}
		comments = append(comments, group)
	}
	file.Comments = comments
}

func moveImportAlias(file *ast.File, importPath, packageName string) string {
	for _, spec := range file.Imports {
		path, err := strconvUnquote(spec.Path.Value)
		if err == nil && path == importPath {
			if spec.Name == nil {
				return packageName
			}
			return spec.Name.Name
		}
	}
	used := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			used[ident.Name] = true
		}
		return true
	})
	alias := packageName
	if !token.IsIdentifier(alias) || alias == "" {
		alias = "movedpkg"
	}
	for n := 2; used[alias]; n++ {
		alias = fmt.Sprintf("%s%d", packageName, n)
	}
	return alias
}

func setMoveImport(file *ast.File, importPath, alias, packageName string) error {
	for _, spec := range file.Imports {
		path, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return err
		}
		if path == importPath {
			if spec.Name != nil && spec.Name.Name == "." {
				return fmt.Errorf("dot import cannot be updated safely")
			}
			if spec.Name != nil && spec.Name.Name == "_" {
				return fmt.Errorf("blank import cannot be used for a moved reference")
			}
			return nil
		}
	}
	var spec *ast.ImportSpec
	if alias == packageName && filepath.Base(importPath) == packageName {
		spec = &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: strconvQuote(importPath)}}
	} else {
		spec = &ast.ImportSpec{Name: ast.NewIdent(alias), Path: &ast.BasicLit{Kind: token.STRING, Value: strconvQuote(importPath)}}
	}
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			gen.Specs = append(gen.Specs, spec)
			file.Imports = append(file.Imports, spec)
			return nil
		}
	}
	gen := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}
	file.Decls = append([]ast.Decl{gen}, file.Decls...)
	file.Imports = append(file.Imports, spec)
	return nil
}

func removeUnusedMoveImports(file *ast.File, info *types.Info, destinationImport, destinationName string, reserved map[string]bool) error {
	used, err := moveImportUsage(file, info, reserved)
	if err != nil {
		return err
	}
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		kept, err := retainedMoveImports(gen.Specs, info, destinationImport, destinationName, used)
		if err != nil {
			return err
		}
		gen.Specs = kept
	}
	rebuildMoveImportIndex(file)
	return nil
}

func moveImportUsage(file *ast.File, info *types.Info, reserved map[string]bool) (map[string]bool, error) {
	used := maps.Clone(reserved)
	if used == nil {
		used = make(map[string]bool)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if _, isPackage := info.Uses[ident].(*types.PkgName); isPackage || reserved[ident.Name] {
					used[ident.Name] = true
				}
			}
		}
		return true
	})
	for _, spec := range file.Imports {
		if spec.Name != nil && spec.Name.Name == "." {
			return nil, fmt.Errorf("dot import is ambiguous")
		}
	}
	return used, nil
}

func retainedMoveImports(specs []ast.Spec, info *types.Info, destinationImport, destinationName string, used map[string]bool) ([]ast.Spec, error) {
	kept := make([]ast.Spec, 0, len(specs))
	for _, item := range specs {
		spec, ok := item.(*ast.ImportSpec)
		if !ok {
			return nil, fmt.Errorf("unexpected non-import specification in import declaration")
		}
		path, err := strconvUnquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		name, err := moveImportName(spec, path, info, destinationImport, destinationName)
		if err != nil {
			return nil, err
		}
		if name == "_" || used[name] {
			kept = append(kept, spec)
		}
	}
	return kept, nil
}

func moveImportName(spec *ast.ImportSpec, path string, info *types.Info, destinationImport, destinationName string) (string, error) {
	if spec.Name != nil {
		return spec.Name.Name, nil
	}
	if path == destinationImport {
		return destinationName, nil
	}
	if imported, ok := info.Implicits[spec].(*types.PkgName); ok {
		return imported.Imported().Name(), nil
	}
	return "", fmt.Errorf("cannot resolve package name for import %q", path)
}

func rebuildMoveImportIndex(file *ast.File) {
	file.Imports = file.Imports[:0]
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			for _, spec := range gen.Specs {
				file.Imports = append(file.Imports, spec.(*ast.ImportSpec))
			}
		}
	}
}

func typecheckMoveOverlay(ctx context.Context, root string, changes map[string][]byte) ([]*packages.Package, error) {
	pkgs, err := loadMovePackages(ctx, root, changes)
	if err != nil {
		return nil, fmt.Errorf("preflight moved package types: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("preflight moved package types: no packages found")
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			return nil, fmt.Errorf("cross-package move failed type preflight in %s: %s", pkg.PkgPath, pkg.Errors[0].Msg)
		}
	}
	return pkgs, nil
}

func validateMoveDestinationReferences(pkgs []*packages.Package, destinationImport, destinationPath, targetName string, expected map[moveReferenceKey][]moveReferenceExpectation) error {
	for key, references := range expected {
		if err := validateMoveReferenceGroup(pkgs, destinationImport, destinationPath, targetName, key, references); err != nil {
			return err
		}
	}
	return nil
}

func validateMoveReferenceGroup(pkgs []*packages.Package, destinationImport, destinationPath, targetName string, key moveReferenceKey, references []moveReferenceExpectation) error {
	pkg := packageByID(pkgs, key.packageID)
	if pkg == nil {
		return fmt.Errorf("cross-package move cannot validate rewritten destination references in package %s", key.packageID)
	}
	moved, err := movedFunctionForReferences(pkg, destinationImport, destinationPath, targetName, key, references)
	if err != nil {
		return err
	}
	syntax := syntaxFileForReference(pkg, key.filePath)
	if syntax == nil {
		return fmt.Errorf("cross-package move cannot validate rewritten destination references in %s", key.filePath)
	}
	return inspectMoveReferenceResolutions(pkg, syntax, destinationImport, targetName, key, references, moved.function)
}

func packageByID(pkgs []*packages.Package, id string) *packages.Package {
	for _, pkg := range pkgs {
		if pkg.ID == id {
			return pkg
		}
	}
	return nil
}

type movedFunctionResolution struct {
	function *types.Func
}

func movedFunctionForReferences(pkg *packages.Package, destinationImport, destinationPath, targetName string, key moveReferenceKey, references []moveReferenceExpectation) (movedFunctionResolution, error) {
	for _, reference := range references {
		if reference.kind != moveReferenceFunction {
			continue
		}
		if pkg.PkgPath != destinationImport {
			return movedFunctionResolution{}, fmt.Errorf("cross-package move cannot validate package-local reference in %s", key.filePath)
		}
		moved, ok := pkg.Types.Scope().Lookup(targetName).(*types.Func)
		if !ok || moved.Pkg() == nil || moved.Pkg().Path() != destinationImport || filepath.Clean(pkg.Fset.Position(moved.Pos()).Filename) != filepath.Clean(destinationPath) {
			return movedFunctionResolution{}, fmt.Errorf("cross-package move cannot identify moved function %q in destination package", targetName)
		}
		return movedFunctionResolution{function: moved}, nil
	}
	return movedFunctionResolution{}, nil
}

func syntaxFileForReference(pkg *packages.Package, path string) *ast.File {
	for _, file := range pkg.Syntax {
		if filepath.Clean(pkg.Fset.Position(file.Pos()).Filename) == path {
			return file
		}
	}
	return nil
}

func inspectMoveReferenceResolutions(pkg *packages.Package, syntax *ast.File, destinationImport, targetName string, key moveReferenceKey, references []moveReferenceExpectation, moved *types.Func) error {
	wanted := make(map[int]moveReferenceExpectation, len(references))
	for _, reference := range references {
		wanted[reference.offset] = reference
	}
	misresolved := false
	ast.Inspect(syntax, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		offset := pkg.Fset.Position(ident.Pos()).Offset
		reference, expectedReference := wanted[offset]
		if !expectedReference || ident.Name != reference.name {
			return true
		}
		valid := validMoveReferenceResolution(pkg.TypesInfo.Uses[ident], reference.kind, destinationImport, moved)
		wanted[offset] = moveReferenceExpectation{kind: 0}
		if !valid {
			misresolved = true
			return false
		}
		return true
	})
	if misresolved {
		return fmt.Errorf("cross-package move refuses rewritten reference %q in %s because it resolves to a different declaration", targetName, key.filePath)
	}
	for offset, expectation := range wanted {
		if expectation.kind != 0 {
			return fmt.Errorf("cross-package move cannot prove rewritten reference %q in %s at byte %d resolves to the moved function or its destination package", expectation.name, key.filePath, offset)
		}
	}
	return nil
}

func validMoveReferenceResolution(resolved types.Object, kind moveReferenceKind, destinationImport string, moved *types.Func) bool {
	switch kind {
	case moveReferenceFunction:
		return resolved == moved
	case moveReferenceImport:
		packageName, ok := resolved.(*types.PkgName)
		return ok && packageName.Imported().Path() == destinationImport
	default:
		return false
	}
}

func publicationOrder(paths []string, destination, source string) []string {
	order := make([]string, 0, len(paths))
	order = append(order, destination)
	for _, path := range paths {
		if path != source && path != destination {
			order = append(order, path)
		}
	}
	order = append(order, source)
	return order
}

func strconvUnquote(s string) (string, error) { return strconv.Unquote(s) }
func strconvQuote(s string) string            { return strconv.Quote(s) }

func loadMovePackages(ctx context.Context, root string, changes map[string][]byte) ([]*packages.Package, error) {
	return loadMovePackageScope(ctx, root, changes, packages.NeedName|packages.NeedFiles|packages.NeedCompiledGoFiles|packages.NeedSyntax|packages.NeedTypes|packages.NeedTypesInfo|packages.NeedImports|packages.NeedDeps|packages.NeedModule|packages.NeedEmbedFiles, []string{"./..."})
}

func bindMoveInputs(ctx context.Context, root string, pkgs []*packages.Package, bound map[string]bool) (bool, error) {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return false, nil
	}
	roots := map[string]bool{root: true}
	auxiliary := make(map[string]bool)
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		module := pkg.Module
		if module == nil {
			return true
		}
		if module.Replace != nil {
			module = module.Replace
		}
		if module.Version != "" || module.Dir == "" {
			return true
		}
		roots[module.Dir] = true
		for _, path := range append(slices.Clone(pkg.OtherFiles), pkg.EmbedFiles...) {
			auxiliary[path] = true
		}
		return true
	}, nil)
	added := false
	for _, tree := range slices.Sorted(maps.Keys(roots)) {
		if !bound[tree] {
			if err := sourcebuf.ObserveTree(ctx, tree); err != nil {
				return false, err
			}
			bound[tree] = true
			added = true
		}
		for _, name := range []string{"go.mod", "go.sum"} {
			if _, err := sourcebuf.ReadFile(ctx, filepath.Join(tree, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return false, fmt.Errorf("bind move manifest: %w", err)
			}
		}
	}
	for _, path := range slices.Sorted(maps.Keys(auxiliary)) {
		if !bound[path] {
			bound[path] = true
			added = true
		}
		if _, err := sourcebuf.ReadFile(ctx, path); err != nil {
			return false, fmt.Errorf("bind move input: %w", err)
		}
	}
	if err := buffer.AssertDisk(); err != nil {
		return false, err
	}
	return added, nil
}

func applyBufferedMove(ctx context.Context, original, changes map[string][]byte) (string, error) {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return "", fmt.Errorf("buffered move requires a source buffer")
	}
	plan := make([]sourcebuf.Change, 0, len(changes))
	diffs := make([]string, 0, len(changes))
	for _, path := range slices.Sorted(maps.Keys(changes)) {
		before, exists := original[path]
		if !exists {
			return "", fmt.Errorf("move output is outside the input snapshot: %s", path)
		}
		plan = append(plan, sourcebuf.Change{Path: path, Before: before, After: changes[path]})
		if diff := UnifiedDiff(path, string(before), string(changes[path])); diff != "" {
			diffs = append(diffs, diff)
		}
	}
	if err := buffer.Apply(ctx, plan); err != nil {
		return "", fmt.Errorf("apply buffered move: %w", err)
	}
	return strings.Join(diffs, "\n"), nil
}

func inspectMoveModuleFiles(ctx context.Context, root, sourcePath, destinationPath, symbolName string) (map[string][]byte, error) {
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open Go module root: %w", err)
	}
	snapshot := map[string][]byte{}
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		return inspectMoveModuleEntry(ctx, root, path, entry, err, rootFS, sourcePath, destinationPath, symbolName, snapshot)
	})
	closeErr := rootFS.Close()
	if walkErr != nil && closeErr != nil {
		return nil, errors.Join(walkErr, fmt.Errorf("close Go module root: %w", closeErr))
	}
	if walkErr != nil {
		return nil, walkErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Go module root: %w", closeErr)
	}
	return snapshot, nil
}

func inspectMoveModuleEntry(ctx context.Context, root, path string, entry fs.DirEntry, entryErr error, rootFS *os.Root, sourcePath, destinationPath, symbolName string, snapshot map[string][]byte) error {
	if entryErr != nil {
		return fmt.Errorf("inspect module file %s: %w", path, entryErr)
	}
	if path == root {
		return nil
	}
	if entry.Type()&fs.ModeSymlink != 0 {
		return fmt.Errorf("cross-package move refuses symlink ambiguity at %s", path)
	}
	if entry.IsDir() {
		return moveDirectoryTraversal(path, entry.Name())
	}
	if !strings.HasSuffix(entry.Name(), ".go") {
		return nil
	}
	data, err := readMoveModuleFile(ctx, root, path, rootFS)
	if err != nil {
		return err
	}
	snapshot[filepath.Clean(path)] = data
	return validateMoveModuleSource(path, data, sourcePath, destinationPath, symbolName)
}

func moveDirectoryTraversal(path, name string) error {
	if name == ".git" || name == "vendor" || name == ".scratch" {
		return filepath.SkipDir
	}
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return filepath.SkipDir
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect nested module boundary in %s: %w", path, err)
	}
	return nil
}

func readMoveModuleFile(ctx context.Context, root, path string, rootFS *os.Root) ([]byte, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, fmt.Errorf("resolve module-relative Go file %s: %w", path, err)
	}
	var data []byte
	if sourcebuf.FromContext(ctx) != nil {
		data, err = sourcebuf.ReadFile(ctx, path)
	} else {
		data, err = rootFS.ReadFile(relative)
	}
	if err != nil {
		return nil, fmt.Errorf("read Go file during move preflight %s: %w", path, err)
	}
	return data, nil
}

func validateMoveModuleSource(path string, data []byte, sourcePath, destinationPath, symbolName string) error {
	text := string(data)
	generated := strings.Contains(text, "Code generated ") && strings.Contains(text, "DO NOT EDIT.")
	constrained := moveFilenameBuildConstraint(path) != "" || strings.Contains(text, "//go:build ") || strings.Contains(text, "// +build ")
	selected := filepath.Clean(path) == filepath.Clean(sourcePath) || filepath.Clean(path) == filepath.Clean(destinationPath)
	if selected && (generated || constrained) {
		return fmt.Errorf("cross-package move refuses generated or build-constrained source or destination %s", path)
	}
	if (generated || constrained) && containsMoveIdentifier(data, symbolName) {
		return fmt.Errorf("cross-package move refuses possible reference to %q in generated or build-constrained file %s", symbolName, path)
	}
	return nil
}

func loadMovePackageScope(ctx context.Context, root string, changes map[string][]byte, mode packages.LoadMode, patterns []string) ([]*packages.Package, error) {
	environment, err := gocache.Environment(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("prepare move environment: %w", err)
	}
	buffer := sourcebuf.FromContext(ctx)
	if buffer != nil {
		if err := buffer.AuthorizeWrites(ctx, root); err != nil {
			return nil, err
		}
	}
	bound := make(map[string]bool)
	if _, err := bindMoveInputs(ctx, root, nil, bound); err != nil {
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("load move packages: %w", err)
		}
		overlay := sourcebuf.Overlay(ctx)
		if overlay == nil {
			overlay = make(map[string][]byte)
		}
		maps.Copy(overlay, changes)
		cfg := &packages.Config{Context: ctx, Dir: root, Env: environment, BuildFlags: []string{"-mod=readonly"}, Mode: mode, Tests: true, Overlay: overlay}
		loaded, err := packages.Load(cfg, patterns...)
		if err != nil {
			return nil, fmt.Errorf("load move packages: %w", err)
		}
		added, err := bindMoveInputs(ctx, root, loaded, bound)
		if err != nil {
			return nil, err
		}
		if !added {
			return loaded, nil
		}
	}
}
