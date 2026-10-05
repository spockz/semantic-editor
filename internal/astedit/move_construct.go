// Package astedit provides zero-token declaration construct relocation.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"semedit/internal/gocache"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"

	"slices"

	"golang.org/x/tools/go/packages"

	"golang.org/x/tools/go/ast/astutil"
)

// MoveConstructOptions configures declaration relocation.
type MoveConstructOptions struct {
	Placement           Placement
	TargetSymbol        string
	DestinationFile     string
	AutoOrganizeImports bool
	DeferFormatting     bool
}

// GoMoveConstructKinds lists declaration kinds supported by Go's move handler.
var GoMoveConstructKinds = []ConstructKind{ConstructFunction, ConstructMethod, ConstructType, ConstructConst, ConstructVar}

func validateMoveConstructPlacement(opts MoveConstructOptions) error {
	if opts.TargetSymbol == "" || opts.Placement == PlacementBeforeSymbol || opts.Placement == PlacementAfterSymbol {
		return nil
	}
	return fmt.Errorf("target_symbol requires before_symbol or after_symbol placement")
}

// MoveConstruct relocates an existing declaration construct to a new placement within filePath.
func MoveConstruct(ctx context.Context, filePath, symbolQuery string, opts MoveConstructOptions) (string, error) {
	if err := validateMoveConstructPlacement(opts); err != nil {
		return "", err
	}
	cleanPath := filepath.Clean(filePath)
	destinationPath := cleanPath
	if opts.DestinationFile != "" {
		destinationPath = filepath.Clean(opts.DestinationFile)
	}
	if destinationPath != cleanPath {
		return moveConstructTo(ctx, cleanPath, destinationPath, symbolQuery, opts)
	}
	snapshot, err := readMoveSource(ctx, cleanPath)
	if err != nil {
		return "", err
	}
	formatted, err := planSameFileMove(snapshot, filePath, symbolQuery, opts)
	if err != nil || formatted == nil {
		return "", err
	}
	return publishSameFileMove(ctx, snapshot, symbolQuery, opts, formatted)
}

type moveSourceSnapshot struct {
	path    string
	content []byte
	file    *ast.File
	set     *token.FileSet
}

func readMoveSource(ctx context.Context, path string) (moveSourceSnapshot, error) {
	content, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return moveSourceSnapshot{}, fmt.Errorf("read target file: %w", err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return moveSourceSnapshot{}, &SyntaxError{File: path, Pos: extractSyntaxPosition(set, err), Cause: err, Err: ErrSyntax}
	}
	return moveSourceSnapshot{path: path, content: content, file: file, set: set}, nil
}

func planSameFileMove(snapshot moveSourceSnapshot, displayPath, symbolQuery string, opts MoveConstructOptions) ([]byte, error) {
	declaration := findMoveDeclaration(snapshot.file, symbolQuery)
	if declaration == nil {
		return nil, &symbol.SymbolError{Op: "move_construct", File: displayPath, Symbol: symbolQuery, Err: symbol.ErrNotFound}
	}
	if group, ok := declaration.(*ast.GenDecl); ok && len(group.Specs) > 1 {
		return nil, fmt.Errorf("cannot move one member of a grouped declaration; split the group first")
	}
	if opts.TargetSymbol == symbolQuery && (opts.Placement == PlacementBeforeSymbol || opts.Placement == PlacementAfterSymbol) {
		return nil, nil
	}
	cut := cutMoveDeclaration(snapshot, declaration)
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, snapshot.path, cut.remaining, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse file after cutting %s: %w", symbolQuery, err)
	}
	insertOffset, err := calculateMoveOffset(set, file, cut.remaining, opts)
	if err != nil {
		return nil, err
	}
	formatted, err := format.Source(spliceMoveDeclaration(cut.remaining, cut.source, insertOffset))
	if err != nil {
		return nil, fmt.Errorf("format moved construct: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), snapshot.path, formatted, parser.ParseComments); err != nil {
		return nil, fmt.Errorf("verify moved syntax: %w", err)
	}
	return formatted, nil
}

type moveDeclarationCut struct {
	remaining []byte
	source    string
}

func cutMoveDeclaration(snapshot moveSourceSnapshot, declaration ast.Decl) moveDeclarationCut {
	start, end := moveDeclarationSourceOffsets(snapshot.set, declaration)
	cutEnd := end
	for cutEnd < len(snapshot.content) && (snapshot.content[cutEnd] == ' ' || snapshot.content[cutEnd] == '\t' || snapshot.content[cutEnd] == '\r') {
		cutEnd++
	}
	if cutEnd < len(snapshot.content) && snapshot.content[cutEnd] == '\n' {
		cutEnd++
	}
	remaining := make([]byte, 0, len(snapshot.content)-(cutEnd-start))
	remaining = append(remaining, snapshot.content[:start]...)
	remaining = append(remaining, snapshot.content[cutEnd:]...)
	return moveDeclarationCut{remaining: remaining, source: string(snapshot.content[start:end])}
}

func moveDeclarationSourceOffsets(set *token.FileSet, declaration ast.Decl) (int, int) {
	startPos, endPos := declaration.Pos(), declaration.End()
	switch item := declaration.(type) {
	case *ast.FuncDecl:
		if item.Doc != nil {
			startPos = item.Doc.Pos()
		}
	case *ast.GenDecl:
		if item.Doc != nil {
			startPos = item.Doc.Pos()
		}
	}
	return set.Position(startPos).Offset, set.Position(endPos).Offset
}

func spliceMoveDeclaration(remaining []byte, declaration string, insertOffset int) []byte {
	var result bytes.Buffer
	result.Write(remaining[:insertOffset])
	if insertOffset > 0 && !bytes.HasSuffix(result.Bytes(), []byte("\n\n")) {
		if bytes.HasSuffix(result.Bytes(), []byte("\n")) {
			result.WriteString("\n")
		} else {
			result.WriteString("\n\n")
		}
	}
	rest := remaining[insertOffset:]
	result.WriteString(declaration)
	if len(bytes.TrimLeft(rest, "\r\n")) == 0 {
		result.WriteByte('\n')
	} else {
		result.WriteString("\n\n")
	}
	result.Write(bytes.TrimLeft(rest, "\r\n"))
	return result.Bytes()
}

func publishSameFileMove(ctx context.Context, snapshot moveSourceSnapshot, symbolQuery string, opts MoveConstructOptions, formatted []byte) (string, error) {
	if err := verifyMoveFileUnchanged(ctx, snapshot.path, snapshot.content); err != nil {
		return "", err
	}
	if err := pipeline.WriteAtomicContext(ctx, snapshot.path, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if err := finalizeMovedFile(ctx, snapshot.path, opts); err != nil {
		return "", err
	}
	finalContent, err := sourcebuf.ReadFile(ctx, snapshot.path)
	if err != nil {
		return "", fmt.Errorf("read moved file after formatting: %w", err)
	}
	return UnifiedDiff(symbolQuery, string(snapshot.content), string(finalContent)), nil
}

func finalizeMovedFile(ctx context.Context, filePath string, opts MoveConstructOptions) error {
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(filePath), filePath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(filePath), filePath)
			return fmt.Errorf("organize imports: %w", err)
		}
	} else if !opts.DeferFormatting {
		if err := pipeline.Format(ctx, filepath.Dir(filePath), filePath); err != nil {
			return fmt.Errorf("format file: %w", err)
		}
	}
	return nil
}

func calculateMoveOffset(fset *token.FileSet, fileNode *ast.File, content []byte, opts MoveConstructOptions) (int, error) {
	switch opts.Placement {
	case PlacementFileStart:
		return offsetAfterImports(fset, fileNode), nil

	case PlacementFileEnd, "":
		return len(content), nil

	case PlacementBeforeSymbol, PlacementAfterSymbol:
		return findSymbolBoundaryOffset(fset, fileNode, opts.TargetSymbol, opts.Placement)

	case PlacementPublicStart, PlacementPublicEnd, PlacementPrivateStart, PlacementPrivateEnd:
		return calculateInsertionOffset(fset, fileNode, content, Options{
			Placement:    opts.Placement,
			TargetSymbol: opts.TargetSymbol,
		})

	default:
		return 0, fmt.Errorf("unsupported placement %q for move_construct", opts.Placement)
	}
}

func findSymbolBoundaryOffset(fset *token.FileSet, fileNode *ast.File, targetSymbol string, placement Placement) (int, error) {
	if targetSymbol == "" {
		return 0, &PlacementError{
			Strategy: placement,
			Err:      ErrSymbolNotFound,
		}
	}
	targetRecv, targetName, _ := symbol.ParseIdentifier(targetSymbol)
	for _, decl := range fileNode.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok {
			boundary := functionSymbolBoundary(fset, function, targetRecv, targetName, placement)
			if boundary.found {
				return boundary.offset, nil
			}
		}
		if group, ok := decl.(*ast.GenDecl); ok && generalDeclarationHasName(group, targetSymbol) {
			return generalDeclarationBoundary(fset, group, placement), nil
		}
	}
	return 0, fmt.Errorf("target symbol %q not found: %w", targetSymbol, ErrSymbolNotFound)
}

type moveSymbolBoundaryResult struct {
	offset int
	found  bool
}

func functionSymbolBoundary(fset *token.FileSet, function *ast.FuncDecl, receiver, name string, placement Placement) moveSymbolBoundaryResult {
	if function.Name.Name != name || extractReceiverName(function.Recv) != receiver {
		return moveSymbolBoundaryResult{}
	}
	if placement == PlacementAfterSymbol {
		return moveSymbolBoundaryResult{offset: fset.Position(function.End()).Offset, found: true}
	}
	if function.Doc != nil {
		return moveSymbolBoundaryResult{offset: fset.Position(function.Doc.Pos()).Offset, found: true}
	}
	return moveSymbolBoundaryResult{offset: fset.Position(function.Pos()).Offset, found: true}
}

func generalDeclarationHasName(group *ast.GenDecl, name string) bool {
	for _, spec := range group.Specs {
		switch item := spec.(type) {
		case *ast.TypeSpec:
			if item.Name.Name == name {
				return true
			}
		case *ast.ValueSpec:
			for _, identifier := range item.Names {
				if identifier.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func generalDeclarationBoundary(fset *token.FileSet, declaration *ast.GenDecl, placement Placement) int {
	if placement == PlacementAfterSymbol {
		return fset.Position(declaration.End()).Offset
	}
	if declaration.Doc != nil {
		return fset.Position(declaration.Doc.Pos()).Offset
	}
	return fset.Position(declaration.Pos()).Offset
}

func moveConstructTo(ctx context.Context, sourcePath, destinationPath, symbolQuery string, opts MoveConstructOptions) (string, error) {
	sourcePath, destinationPath = filepath.Clean(sourcePath), filepath.Clean(destinationPath)
	if filepath.Dir(sourcePath) != filepath.Dir(destinationPath) {
		return moveConstructAcrossPackages(ctx, sourcePath, destinationPath, symbolQuery, opts)
	}
	snapshot, err := readMovePair(ctx, sourcePath, destinationPath)
	if err != nil {
		return "", err
	}
	plan, err := planSamePackageMove(ctx, snapshot, symbolQuery, opts)
	if err != nil {
		return "", err
	}
	return publishSamePackageMove(ctx, snapshot, plan, symbolQuery, opts)
}

type movePairSnapshot struct {
	sourcePath      string
	destinationPath string
	source          []byte
	destination     []byte
	sourceSet       *token.FileSet
	destinationSet  *token.FileSet
	sourceAST       *ast.File
	destinationAST  *ast.File
}

type movePairPlan struct {
	source      []byte
	destination []byte
}

func readMovePair(ctx context.Context, sourcePath, destinationPath string) (movePairSnapshot, error) {
	source, err := sourcebuf.ReadFile(ctx, sourcePath)
	if err != nil {
		return movePairSnapshot{}, fmt.Errorf("read source file: %w", err)
	}
	destination, err := sourcebuf.ReadFile(ctx, destinationPath)
	if err != nil {
		return movePairSnapshot{}, fmt.Errorf("read destination file: %w", err)
	}
	sourceSet := token.NewFileSet()
	sourceAST, err := parser.ParseFile(sourceSet, sourcePath, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return movePairSnapshot{}, fmt.Errorf("parse source file: %w", err)
	}
	destinationSet := token.NewFileSet()
	destinationAST, err := parser.ParseFile(destinationSet, destinationPath, destination, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return movePairSnapshot{}, fmt.Errorf("parse destination file: %w", err)
	}
	if sourceAST.Name.Name != destinationAST.Name.Name {
		return movePairSnapshot{}, fmt.Errorf("source and destination must belong to the same Go package")
	}
	return movePairSnapshot{
		sourcePath:      sourcePath,
		destinationPath: destinationPath,
		source:          source,
		destination:     destination,
		sourceSet:       sourceSet,
		destinationSet:  destinationSet,
		sourceAST:       sourceAST,
		destinationAST:  destinationAST,
	}, nil
}

func planSamePackageMove(ctx context.Context, snapshot movePairSnapshot, symbolQuery string, opts MoveConstructOptions) (movePairPlan, error) {
	declaration := findMoveDeclaration(snapshot.sourceAST, symbolQuery)
	if err := validateMovePairTarget(snapshot, declaration, symbolQuery); err != nil {
		return movePairPlan{}, err
	}
	cut, err := cutCrossFileMove(snapshot, declaration)
	if err != nil {
		return movePairPlan{}, err
	}
	insertion, err := insertCrossFileMove(snapshot, cut.declaration, opts)
	if err != nil {
		return movePairPlan{}, err
	}
	imports, err := moveImportsForDeclaration(ctx, filepath.Dir(snapshot.sourcePath), declaration, snapshot.sourceAST)
	if err != nil {
		return movePairPlan{}, err
	}
	if mergeErr := mergeMoveImports(insertion.file, imports); mergeErr != nil {
		return movePairPlan{}, mergeErr
	}
	destination, err := formatMoveDestination(insertion.file, insertion.set)
	if err != nil {
		return movePairPlan{}, err
	}
	source, err := format.Source(cut.remaining)
	if err != nil {
		return movePairPlan{}, fmt.Errorf("format source after move: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), snapshot.destinationPath, destination, parser.ParseComments|parser.AllErrors); err != nil {
		return movePairPlan{}, fmt.Errorf("validate destination after move: %w", err)
	}
	return movePairPlan{source: source, destination: destination}, nil
}

func validateMovePairTarget(snapshot movePairSnapshot, declaration ast.Decl, symbolQuery string) error {
	if declaration == nil {
		return &symbol.SymbolError{Op: "move_construct", File: snapshot.sourcePath, Symbol: symbolQuery, Err: symbol.ErrNotFound}
	}
	if group, ok := declaration.(*ast.GenDecl); ok && len(group.Specs) > 1 {
		return fmt.Errorf("cannot move one member of a grouped declaration; split the group first")
	}
	if declarationNameCollision(snapshot.destinationAST, declaration) {
		return fmt.Errorf("destination already declares a name from %q", symbolQuery)
	}
	if moveBuildConstraint(snapshot.sourceAST)+moveFilenameBuildConstraint(snapshot.sourcePath) != moveBuildConstraint(snapshot.destinationAST)+moveFilenameBuildConstraint(snapshot.destinationPath) {
		return fmt.Errorf("source and destination have different Go build constraints")
	}
	return nil
}

type crossFileMoveCut struct {
	remaining   []byte
	declaration string
}

func cutCrossFileMove(snapshot movePairSnapshot, declaration ast.Decl) (crossFileMoveCut, error) {
	start, end := moveDeclOffsets(snapshot.sourceSet, declaration, snapshot.sourceAST, snapshot.source)
	declarationText := string(snapshot.source[start:end])
	remaining := append(append([]byte(nil), snapshot.source[:start]...), snapshot.source[end:]...)
	if len(remaining) > 0 && remaining[len(remaining)-1] != '\n' {
		remaining = append(remaining, '\n')
	}
	if _, err := parser.ParseFile(token.NewFileSet(), snapshot.sourcePath, remaining, parser.ParseComments|parser.AllErrors); err != nil {
		return crossFileMoveCut{}, fmt.Errorf("validate source after move: %w", err)
	}
	return crossFileMoveCut{remaining: remaining, declaration: declarationText}, nil
}

type moveDestinationInsertion struct {
	file *ast.File
	set  *token.FileSet
}

func insertCrossFileMove(snapshot movePairSnapshot, declaration string, opts MoveConstructOptions) (moveDestinationInsertion, error) {
	offset, err := calculateMoveOffset(snapshot.destinationSet, snapshot.destinationAST, snapshot.destination, opts)
	if err != nil {
		return moveDestinationInsertion{}, err
	}
	updated := spliceMoveDeclaration(snapshot.destination, declaration, offset)
	formatted, err := format.Source(updated)
	if err != nil {
		return moveDestinationInsertion{}, fmt.Errorf("format destination after move: %w", err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, snapshot.destinationPath, formatted, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return moveDestinationInsertion{}, fmt.Errorf("parse destination after declaration insertion: %w", err)
	}
	return moveDestinationInsertion{file: file, set: set}, nil
}

func formatMoveDestination(file *ast.File, set *token.FileSet) ([]byte, error) {
	var output bytes.Buffer
	if err := format.Node(&output, set, file); err != nil {
		return nil, fmt.Errorf("format destination imports: %w", err)
	}
	return output.Bytes(), nil
}

func publishSamePackageMove(ctx context.Context, snapshot movePairSnapshot, plan movePairPlan, symbolQuery string, opts MoveConstructOptions) (string, error) {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		cleanSource, err := cleanupMoveImports(ctx, snapshot.sourcePath, plan.source)
		if err != nil {
			return "", err
		}
		return applyBufferedMove(ctx, map[string][]byte{snapshot.sourcePath: snapshot.source, snapshot.destinationPath: snapshot.destination}, map[string][]byte{snapshot.sourcePath: cleanSource, snapshot.destinationPath: plan.destination})
	}
	if err := verifyMoveFileUnchanged(ctx, snapshot.sourcePath, snapshot.source); err != nil {
		return "", err
	}
	if err := verifyMoveFileUnchanged(ctx, snapshot.destinationPath, snapshot.destination); err != nil {
		return "", err
	}
	if err := writeSamePackageMove(ctx, snapshot, plan); err != nil {
		return "", err
	}
	if opts.AutoOrganizeImports {
		if err := organizeMovedFiles(ctx, snapshot); err != nil {
			return "", err
		}
	}
	return readMovedFileDiff(ctx, snapshot, symbolQuery)
}

func writeSamePackageMove(ctx context.Context, snapshot movePairSnapshot, plan movePairPlan) error {
	if err := pipeline.WriteAtomicContext(ctx, snapshot.destinationPath, plan.destination); err != nil {
		return fmt.Errorf("write destination atomically: %w", err)
	}
	if err := pipeline.WriteAtomicContext(ctx, snapshot.sourcePath, plan.source); err != nil {
		return fmt.Errorf("destination %s was written but source %s remains unchanged: %w", snapshot.destinationPath, snapshot.sourcePath, err)
	}
	return nil
}

func organizeMovedFiles(ctx context.Context, snapshot movePairSnapshot) error {
	for _, path := range []string{snapshot.sourcePath, snapshot.destinationPath} {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(path), path); err != nil {
			return fmt.Errorf("organize imports in %s after move: %w", path, err)
		}
	}
	return nil
}

func readMovedFileDiff(ctx context.Context, snapshot movePairSnapshot, symbolQuery string) (string, error) {
	finalSource, err := sourcebuf.ReadFile(ctx, snapshot.sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source after move: %w", err)
	}
	finalDestination, err := sourcebuf.ReadFile(ctx, snapshot.destinationPath)
	if err != nil {
		return "", fmt.Errorf("read destination after move: %w", err)
	}
	return UnifiedDiff(symbolQuery, string(snapshot.source), string(finalSource)) + "\n" + UnifiedDiff(symbolQuery, string(snapshot.destination), string(finalDestination)), nil
}

func findMoveDeclaration(file *ast.File, query string) ast.Decl {
	recv, name, _ := symbol.ParseIdentifier(query)
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == recv {
			return fn
		}
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					if item.Name.Name == query {
						return gen
					}
				case *ast.ValueSpec:
					for _, ident := range item.Names {
						if ident.Name == query {
							return gen
						}
					}
				}
			}
		}
	}
	return nil
}

func moveDeclOffsets(fset *token.FileSet, decl ast.Decl, file *ast.File, source []byte) (int, int) {
	startPos := decl.Pos()
	endPos := decl.End()
	switch item := decl.(type) {
	case *ast.FuncDecl:
		if item.Doc != nil {
			startPos = item.Doc.Pos()
		}
	case *ast.GenDecl:
		if item.Doc != nil {
			startPos = item.Doc.Pos()
		}
	}
	start := fset.Position(startPos).Offset
	end := fset.Position(endPos).Offset
	endLine := fset.Position(endPos).Line
	for _, group := range file.Comments {
		groupStart := fset.Position(group.Pos())
		groupEnd := fset.Position(group.End()).Offset
		if groupStart.Offset < end || groupStart.Line != endLine {
			continue
		}
		if len(bytes.TrimSpace(source[end:groupStart.Offset])) == 0 {
			end = groupEnd
		}
	}
	return start, end
}

func declarationNameCollision(file *ast.File, moved ast.Decl) bool {
	return declarationNameCollisionSet(file, movedDeclarationNames(moved))
}

func movedDeclarationNames(declaration ast.Decl) map[string]bool {
	names := map[string]bool{}
	switch decl := declaration.(type) {
	case *ast.FuncDecl:
		names[decl.Name.Name] = true
	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			switch item := spec.(type) {
			case *ast.TypeSpec:
				names[item.Name.Name] = true
			case *ast.ValueSpec:
				for _, name := range item.Names {
					names[name.Name] = true
				}
			}
		}
	}
	return names
}

func declarationNameCollisionSet(file *ast.File, names map[string]bool) bool {
	for _, decl := range file.Decls {
		if fileDeclarationHasName(decl, names) {
			return true
		}
	}
	return false
}

func fileDeclarationHasName(declaration ast.Decl, names map[string]bool) bool {
	for name := range movedDeclarationNames(declaration) {
		if names[name] {
			return true
		}
	}
	return false
}

func moveImportsForDeclaration(ctx context.Context, workingDir string, decl ast.Decl, source *ast.File) (*ast.File, error) {
	used := map[string]bool{}
	ast.Inspect(decl, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok {
			if name, ok := selector.X.(*ast.Ident); ok {
				used[name.Name] = true
			}
		}
		return true
	})
	imports := &ast.File{}
	for _, spec := range source.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("decode source import: %w", err)
		}
		if spec.Name != nil && spec.Name.Name == "." {
			return nil, fmt.Errorf("cannot move declaration from a file with a dot import")
		}
		var local string
		if spec.Name != nil {
			local = spec.Name.Name
		} else {
			resolved, resolveErr := resolveMoveImportName(ctx, workingDir, pathValue)
			if resolveErr != nil {
				return nil, resolveErr
			}
			local = resolved
		}
		if used[local] {
			imports.Imports = append(imports.Imports, spec)
		}
	}
	return imports, nil
}

func moveFilenameBuildConstraint(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), ".go")
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return ""
	}
	known := map[string]bool{"aix": true, "android": true, "darwin": true, "dragonfly": true, "freebsd": true, "illumos": true, "ios": true, "js": true, "linux": true, "netbsd": true, "openbsd": true, "plan9": true, "solaris": true, "wasip1": true, "windows": true, "386": true, "amd64": true, "arm": true, "arm64": true, "loong64": true, "mips": true, "mipsle": true, "mips64": true, "mips64le": true, "ppc64": true, "ppc64le": true, "riscv64": true, "s390x": true, "wasm": true}
	if !known[parts[len(parts)-1]] {
		return ""
	}
	last := parts[len(parts)-1]
	if len(parts) > 2 && known[parts[len(parts)-2]] {
		return parts[len(parts)-2] + "_" + last
	}
	return last
}

func moveBuildConstraint(file *ast.File) string {
	var goBuild, plusBuild []string
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if after, ok := strings.CutPrefix(text, "//go:build "); ok {
				goBuild = append(goBuild, after)
			}
			if after, ok := strings.CutPrefix(text, "// +build "); ok {
				plusBuild = append(plusBuild, after)
			}
		}
	}
	if len(goBuild) > 0 {
		return strings.Join(goBuild, "\n")
	}
	if len(plusBuild) > 0 {
		return strings.Join(plusBuild, "\n")
	}
	return ""
}

func resolveMoveImportName(ctx context.Context, workingDir, importPath string) (string, error) {
	if sourcebuf.FromContext(ctx) != nil {
		root, err := moveModuleRoot(filepath.Join(workingDir, "source.go"))
		if err != nil {
			return "", err
		}
		loaded, err := loadMovePackageScope(ctx, root, nil, packages.NeedName|packages.NeedFiles|packages.NeedModule|packages.NeedImports|packages.NeedDeps|packages.NeedEmbedFiles, []string{importPath})
		if err != nil {
			return "", err
		}
		var selected *packages.Package
		for _, pkg := range loaded {
			if pkg.PkgPath != importPath || pkg.ID != pkg.PkgPath {
				continue
			}
			if selected != nil {
				return "", fmt.Errorf("ambiguous package name for import %q", importPath)
			}
			selected = pkg
		}
		if selected == nil || selected.Name == "" || !token.IsIdentifier(selected.Name) {
			return "", fmt.Errorf("cannot resolve package name for import %q", importPath)
		}
		if len(selected.Errors) != 0 {
			return "", fmt.Errorf("cannot resolve package name for import %q: %s", importPath, selected.Errors[0].Msg)
		}
		return selected.Name, nil
	}

	// #nosec G204 -- executable is fixed to go; importPath is one parsed import literal.
	command := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-f", "{{.Name}}", importPath)
	command.Dir = workingDir
	environment, err := gocache.Environment(ctx, workingDir)
	if err != nil {
		return "", fmt.Errorf("configure Go state for package-name lookup: %w", err)
	}
	command.Env = environment
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("cannot resolve package name for import %q; refusing move: %w", importPath, err)
	}
	name := strings.TrimSpace(string(output))
	if name == "" || !token.IsIdentifier(name) {
		return "", fmt.Errorf("go list returned an invalid package name %q for import %q; refusing move", name, importPath)
	}
	return name, nil
}

func mergeMoveImports(destination, source *ast.File) error {
	imports := make(map[string]string, len(destination.Imports))
	names := make(map[string]string, len(destination.Imports))
	for _, spec := range destination.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("decode destination import: %w", err)
		}
		local := filepath.Base(pathValue)
		if spec.Name != nil {
			local = spec.Name.Name
		}
		imports[pathValue], names[local] = local, pathValue
	}
	var additions []ast.Spec
	for _, spec := range source.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("decode source import: %w", err)
		}
		local := filepath.Base(pathValue)
		var alias *ast.Ident
		if spec.Name != nil {
			local = spec.Name.Name
			alias = &ast.Ident{Name: local}
		}
		if existing, exists := imports[pathValue]; exists {
			if existing != local {
				return fmt.Errorf("cannot move declaration because destination imports %q as %q, source uses %q", pathValue, existing, local)
			}
			continue
		}
		if existing, exists := names[local]; exists && existing != pathValue {
			return fmt.Errorf("cannot move declaration because import name %q is already used for %q", local, existing)
		}
		additions = append(additions, &ast.ImportSpec{Name: alias, Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(pathValue)}})
		imports[pathValue], names[local] = local, pathValue
	}
	if len(additions) == 0 {
		return nil
	}
	for _, declaration := range destination.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			gen.Specs = append(gen.Specs, additions...)
			return nil
		}
	}
	destination.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Lparen: token.Pos(1), Specs: additions}}, destination.Decls...)
	return nil
}

func verifyMoveFileUnchanged(ctx context.Context, path string, expected []byte) error {

	current, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return fmt.Errorf("recheck %s before move publication: %w", path, err)
	}
	if !bytes.Equal(current, expected) {
		return fmt.Errorf("%s changed while move was being prepared; refusing to overwrite concurrent edits", path)
	}
	return nil
}

func cleanupMoveImports(ctx context.Context, path string, content []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse move imports: %w", err)
	}
	for _, spec := range slices.Clone(file.Imports) {
		importPath, decodeErr := strconv.Unquote(spec.Path.Value)
		if decodeErr != nil {
			return nil, fmt.Errorf("decode move import: %w", decodeErr)
		}
		if importPath == "C" {
			continue
		}
		var name string
		if spec.Name != nil {
			name = spec.Name.Name
			if name == "_" || name == "." {
				continue
			}
		} else {
			name, err = resolveMoveImportName(ctx, filepath.Dir(path), importPath)
			if err != nil {
				return nil, err
			}
		}
		originalName := spec.Name
		spec.Name = ast.NewIdent(name)
		used := astutil.UsesImport(file, importPath)
		spec.Name = originalName
		if !used {
			astutil.DeleteImport(fset, file, importPath)
		}
	}
	var out bytes.Buffer
	if formatErr := format.Node(&out, fset, file); formatErr != nil {
		return nil, fmt.Errorf("format move imports: %w", formatErr)
	}
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format move imports: %w", err)
	}
	return formatted, nil
}
