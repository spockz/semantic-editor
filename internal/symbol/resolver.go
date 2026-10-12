// Package symbol provides AST symbol resolution and coordinate translation for Go source files.
package symbol

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"semedit/internal/sourcebuf"
	"slices"
	"strings"
)

// Symbol represents an identified symbol with exact source coordinates.
type Symbol struct {
	Name          string `json:"name"`
	Receiver      string `json:"receiver,omitempty"`
	QualifiedName string `json:"qualified_name,omitempty"`
	Kind          string `json:"kind"`
	File          string `json:"file"`
	Line          int    `json:"line"`
	Column        int    `json:"column"`
	Offset        int    `json:"offset"`
}

// BuildQualifiedName returns the formatted name including receiver if present.
func (s *Symbol) BuildQualifiedName() string {
	if s.Receiver != "" {
		return s.Receiver + "." + s.Name
	}
	return s.Name
}

// LookupResult holds the resolved symbol information or ambiguous candidate options.
type LookupResult struct {
	Definition *Symbol `json:"definition,omitempty"`
	// Usages are syntactic local binding declarations, not semantic references.
	Usages     []*Symbol `json:"usages,omitempty"`
	Candidates []*Symbol `json:"candidates,omitempty"`
}

// IsAmbiguous reports whether lookup found multiple candidate declarations.
func (result *LookupResult) IsAmbiguous() bool {
	return result != nil && len(result.Candidates) >= 2
}

// ParseIdentifier decomposes a query identifier into receiver and symbol name.
func ParseIdentifier(raw string) (receiver string, name string, err error) {
	_, receiver, name, err = parseIdentifier(raw)
	return receiver, name, err
}

func parseIdentifier(raw string) (pkg string, receiver string, name string, err error) {
	clean := normalizeInput(raw)
	if clean == "" {
		return "", "", "", fmt.Errorf("%w: identifier cannot be empty", ErrInvalidIdentifier)
	}
	parts := strings.Split(clean, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", "", "", fmt.Errorf("%w: %q contains an empty selector component", ErrInvalidIdentifier, raw)
		}
	}
	switch len(parts) {
	case 1:
		return "", "", parts[0], nil
	case 2:
		pkg, receiver, name = "", cleanReceiver(parts[0]), parts[1]
	case 3:
		pkg, receiver, name = parts[0], cleanReceiver(parts[1]), parts[2]
	default:
		return "", "", "", fmt.Errorf("%w: %q (expected [Package.][Receiver.]Name)", ErrInvalidIdentifier, raw)
	}
	if strings.TrimSpace(receiver) == "" {
		return "", "", "", fmt.Errorf("%w: %q contains an empty receiver", ErrInvalidIdentifier, raw)
	}
	return pkg, receiver, name, nil
}

func cleanReceiver(raw string) string {
	recv := strings.TrimPrefix(raw, "*")
	recv = strings.TrimPrefix(recv, "(")
	recv = strings.TrimSuffix(recv, ")")
	return strings.TrimPrefix(recv, "*")
}

type sourceSnapshotReader struct {
	ctx   context.Context
	files map[string][]byte
}

func normalizeInput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) >= 2 {
		first, last := trimmed[0], trimmed[len(trimmed)-1]
		if (first == '\'' || first == '"') && first == last {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}

// Resolve locates a symbol within a target file or workspace directory.
func Resolve(rootDir string, filePath string, query string) (*LookupResult, error) {
	snapshot, err := ResolveSnapshot(rootDir, filePath, query)
	if err != nil {
		return nil, err
	}
	return snapshot.Result, nil
}

func scanFile(filePath string, targetPackage string, targetRecv string, targetName string, packageIndexes map[string]*packageTypeIndex, reader *sourceSnapshotReader) ([]*Symbol, error) {
	cleanPath := filepath.Clean(filePath)
	src, err := reader.read(cleanPath)
	if err != nil {
		return nil, &SymbolError{Op: "read", File: cleanPath, Symbol: targetName, Err: err}
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		pos := extractPosition(fset, err)
		return nil, &SymbolError{Op: "parse", File: filePath, Symbol: targetName, Pos: pos, Err: err}
	}
	if targetPackage != "" && node.Name.Name != targetPackage {
		return nil, nil
	}

	indexedReceiver, err := receiverTypeIndex(filePath, targetRecv, node, packageIndexes, reader)
	if err != nil {
		return nil, fmt.Errorf("index interface declarations for %s: %w", filePath, err)
	}
	index := indexedReceiver.index

	results, err := scanTopLevelDeclarations(fset, node, filePath, targetRecv, targetName, index)
	if err != nil {
		return nil, fmt.Errorf("resolve %s.%s in %s: %w", targetRecv, targetName, filePath, err)
	}
	if targetRecv == "" {
		results = append(results, scanFunctionVariables(fset, node, filePath, targetName)...)
	}
	return results, nil
}

type receiverIndexResult struct {
	index *packageTypeIndex
	found bool
}

func receiverTypeIndex(filePath, targetRecv string, file *ast.File, packageIndexes map[string]*packageTypeIndex, reader *sourceSnapshotReader) (receiverIndexResult, error) {
	if targetRecv == "" {
		return receiverIndexResult{}, nil
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		result, err := receiverIndexInDeclaration(filePath, targetRecv, file.Name.Name, general, packageIndexes, reader)
		if err != nil {
			return receiverIndexResult{}, err
		}
		if result.found {
			return result, nil
		}
	}
	return receiverIndexResult{}, nil
}

func receiverIndexInDeclaration(filePath, targetRecv, packageName string, declaration *ast.GenDecl, packageIndexes map[string]*packageTypeIndex, reader *sourceSnapshotReader) (receiverIndexResult, error) {
	for _, raw := range declaration.Specs {
		spec, ok := raw.(*ast.TypeSpec)
		if !ok || spec.Name.Name != targetRecv || !supportsIndexedReceiver(spec.Type) {
			continue
		}
		key := receiverIndexKey(filePath, packageName)
		index := packageIndexes[key]
		if index != nil {
			return receiverIndexResult{index: index, found: true}, nil
		}
		index, err := loadPackageTypeIndex(filePath, packageName, reader)
		if err != nil {
			return receiverIndexResult{}, err
		}
		packageIndexes[key] = index
		return receiverIndexResult{index: index, found: true}, nil
	}
	return receiverIndexResult{}, nil
}

func supportsIndexedReceiver(expression ast.Expr) bool {
	if _, isInterface := expression.(*ast.InterfaceType); isInterface {
		return true
	}
	return embeddedInterfaceName(expression) != ""
}

func receiverIndexKey(filePath, packageName string) string {
	key := filepath.Clean(filepath.Dir(filePath)) + "\x00" + packageName
	if strings.HasSuffix(filePath, "_test.go") {
		key += "\x00tests"
	}
	return key
}

func appendLocalVariable(results *[]*Symbol, fset *token.FileSet, filePath string, ident *ast.Ident, targetName string, kind string) {
	if ident == nil || ident.Name == "_" || ident.Name != targetName {
		return
	}
	pos := fset.Position(ident.Pos())
	*results = append(*results, &Symbol{
		Name:   ident.Name,
		Kind:   kind,
		File:   filePath,
		Line:   pos.Line,
		Column: pos.Column,
		Offset: pos.Offset,
	})
}

func findTypeMembers(fset *token.FileSet, filePath string, spec *ast.TypeSpec, targetName string, index *packageTypeIndex) ([]*Symbol, error) {
	switch typeNode := spec.Type.(type) {
	case *ast.StructType:
		var results []*Symbol
		for _, field := range typeNode.Fields.List {
			for _, ident := range field.Names {
				if ident.Name != targetName {
					continue
				}
				position := fset.Position(ident.Pos())
				results = append(results, &Symbol{Name: ident.Name, Receiver: spec.Name.Name, Kind: "field", File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset})
			}
		}
		return results, nil
	case *ast.InterfaceType:
		root := indexedTypeDeclaration{spec: spec, source: packageSourceFile{path: filePath, fset: fset}}
		return findInterfaceMethods(index, root, targetName)
	default:
		if embeddedInterfaceName(spec.Type) == "" {
			return nil, nil
		}
		root := indexedTypeDeclaration{spec: spec, source: packageSourceFile{path: filePath, fset: fset}}
		return findInterfaceMethods(index, root, targetName)
	}
}

func extractReceiver(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	for {
		switch node := t.(type) {
		case *ast.StarExpr:
			t = node.X
		case *ast.IndexExpr:
			t = node.X
		case *ast.IndexListExpr:
			t = node.X
		case *ast.Ident:
			return node.Name
		default:
			return ""
		}
	}
}

func extractPosition(fset *token.FileSet, err error) token.Position {
	var errList scanner.ErrorList
	if errors.As(err, &errList) && len(errList) > 0 {
		if fset != nil {
			if f := fset.File(token.Pos(1)); f != nil {
				return fset.Position(f.Pos(errList[0].Pos.Offset))
			}
		}
		return errList[0].Pos
	}
	if sErr, ok := errors.AsType[*scanner.Error](err); ok {
		if fset != nil {
			if f := fset.File(token.Pos(1)); f != nil {
				return fset.Position(f.Pos(sErr.Pos.Offset))
			}
		}
		return sErr.Pos
	}
	return token.Position{}
}

func resolveTargetFiles(ctx context.Context, rootDir, filePath string) ([]string, error) {
	if filePath != "" {
		return resolveSelectedTargetFiles(ctx, rootDir, filePath)
	}
	return resolveWorkspaceTargetFiles(ctx, rootDir)
}

func resolveSelectedTargetFiles(ctx context.Context, rootDir, filePath string) ([]string, error) {
	resolvedPath := filePath
	if !filepath.IsAbs(resolvedPath) && rootDir != "" {
		resolvedPath = filepath.Join(rootDir, filePath)
	}
	info, err := os.Stat(resolvedPath)
	if err == nil && info.IsDir() {
		return resolveDirectoryTargetFiles(ctx, rootDir, resolvedPath)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("inspect symbol target %q: %w", resolvedPath, err)
	}
	if errors.Is(err, fs.ErrNotExist) {
		files, found, directoryErr := resolveVirtualDirectoryTargetFiles(ctx, rootDir, resolvedPath)
		if directoryErr != nil {
			return nil, directoryErr
		}
		if found {
			return files, nil
		}
	}
	return []string{resolvedPath}, nil
}

func resolveWorkspaceTargetFiles(ctx context.Context, rootDir string) ([]string, error) {
	searchRoot := rootDir
	if searchRoot == "" {
		searchRoot = "."
	}
	rootPath := filepath.Clean(searchRoot)
	targetFiles := make(map[string]bool)
	err := filepath.WalkDir(searchRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if filepath.Clean(path) != rootPath && ignoredSourceDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if resolvableSourceName(entry.Name()) {
			targetFiles[path] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := addOverlayTargetFiles(ctx, rootPath, rootPath, false, targetFiles); err != nil {
		return nil, err
	}
	return sortedTargetFiles(targetFiles), nil
}

func ignoredSourceDirectory(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata"
}

func resolvableSourceName(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func isLocalBinding(kind string) bool {
	switch kind {
	case "receiver", "parameter", "result", "variable":
		return true
	default:
		return false
	}
}

// ResolutionSnapshot contains a lookup result and the exact source bytes read while resolving it.
type ResolutionSnapshot struct {
	Result *LookupResult
	Files  map[string][]byte
}

func newSourceSnapshotReader() *sourceSnapshotReader {
	return &sourceSnapshotReader{files: make(map[string][]byte)}
}

func (reader *sourceSnapshotReader) read(path string) ([]byte, error) {
	cleanPath := filepath.Clean(path)
	absolutePath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("resolve source path %s: %w", cleanPath, err)
	}
	key := filepath.Clean(absolutePath)
	if source, ok := reader.files[key]; ok {
		return source, nil
	}
	// #nosec G304 -- caller supplies a Go source path discovered by the resolver.
	source, err := sourcebuf.ReadFile(reader.ctx, cleanPath)
	if err != nil {
		return nil, err
	}
	reader.files[key] = source
	return source, nil
}

func (reader *sourceSnapshotReader) snapshot() map[string][]byte {
	files := make(map[string][]byte, len(reader.files))
	maps.Copy(files, reader.files)
	return files
}

// ResolveSnapshot resolves a symbol and returns the exact source files read during lookup.
func ResolveSnapshot(rootDir string, filePath string, query string) (*ResolutionSnapshot, error) {
	return ResolveSnapshotWithContext(context.Background(), rootDir, filePath, query)
}

// ResolveSnapshotWithContext resolves symbols against the current batch source revision.
func ResolveSnapshotWithContext(ctx context.Context, rootDir string, filePath string, query string) (*ResolutionSnapshot, error) {
	pkg, recv, name, err := parseIdentifier(query)
	if err != nil {
		return nil, &SymbolError{Op: "resolve", File: filePath, Symbol: query, Err: err}
	}

	targetFiles, err := resolveTargetFiles(ctx, rootDir, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	reader := newSourceSnapshotReader()
	reader.ctx = ctx
	packageIndexes := make(map[string]*packageTypeIndex)
	declarations, usages, err := scanResolutionCandidates(rootDir, targetFiles, pkg, recv, name, packageIndexes, reader)
	if err != nil {
		return nil, err
	}
	return resolveCandidates(filePath, query, declarations, usages, reader)
}

func scanResolutionCandidates(rootDir string, targetFiles []string, pkg, recv, name string, packageIndexes map[string]*packageTypeIndex, reader *sourceSnapshotReader) ([]*Symbol, []*Symbol, error) {
	var declarations, usages []*Symbol
	scan := func(targetPackage, targetRecv string) error {
		for _, file := range targetFiles {
			symbols, err := scanFile(file, targetPackage, targetRecv, name, packageIndexes, reader)
			if err != nil {
				return err
			}
			for _, candidate := range symbols {
				candidate.QualifiedName = candidate.BuildQualifiedName()
				if rootDir != "" {
					if relative, err := filepath.Rel(rootDir, candidate.File); err == nil && !strings.HasPrefix(relative, "..") {
						candidate.File = relative
					}
				}
				if isLocalBinding(candidate.Kind) {
					usages = append(usages, candidate)
				} else {
					declarations = append(declarations, candidate)
				}
			}
		}
		return nil
	}
	if err := scan(pkg, recv); err != nil {
		return nil, nil, err
	}
	if pkg == "" && recv != "" && len(declarations) == 0 && len(usages) == 0 {
		if err := scan(recv, ""); err != nil {
			return nil, nil, err
		}
	}
	return declarations, usages, nil
}

func resolveCandidates(filePath, query string, declarations, usages []*Symbol, reader *sourceSnapshotReader) (*ResolutionSnapshot, error) {
	result := &LookupResult{Usages: usages}
	if len(declarations) > 0 {
		if len(declarations) > 1 {
			result.Candidates = declarations
			return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
		}
		result.Definition = declarations[0]
		return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
	}
	if len(usages) == 0 {
		return nil, &SymbolError{Op: "resolve", File: filePath, Symbol: query, Err: ErrNotFound}
	}
	if len(usages) > 1 {
		result.Candidates = usages
		return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
	}
	result.Definition = usages[0]
	return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
}

func pathWithinDirectory(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func resolveDirectoryTargetFiles(ctx context.Context, rootDir, directory string) ([]string, error) {
	boundary := rootDir
	if boundary == "" {
		boundary = "."
	}
	boundary, err := filepath.Abs(boundary)
	if err != nil {
		return nil, fmt.Errorf("resolve symbol workspace %q: %w", rootDir, err)
	}
	boundary, err = filepath.EvalSymlinks(boundary)
	if err != nil {
		return nil, fmt.Errorf("canonicalize symbol workspace %q: %w", rootDir, err)
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve Go package directory: %w", err)
	}
	canonicalDirectory, err := canonicalTargetDirectory(ctx, directory)
	if err != nil {
		return nil, fmt.Errorf("canonicalize Go package directory: %w", err)
	}
	if !pathWithinDirectory(boundary, canonicalDirectory) {
		return nil, fmt.Errorf("package directory %q resolves outside the workspace", directory)
	}
	entries, err := os.ReadDir(directory)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read Go package directory %q: %w", directory, err)
	}
	targetFiles := make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		candidate := filepath.Join(directory, name)
		candidateInfo, statErr := os.Stat(candidate)
		if statErr != nil {
			return nil, fmt.Errorf("inspect Go source candidate %q: %w", candidate, statErr)
		}
		if !candidateInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("source candidate %q is not a regular file", candidate)
		}
		canonicalCandidate, canonicalErr := filepath.EvalSymlinks(candidate)
		if canonicalErr != nil {
			return nil, fmt.Errorf("canonicalize Go source candidate %q: %w", candidate, canonicalErr)
		}
		if !pathWithinDirectory(boundary, canonicalCandidate) {
			return nil, fmt.Errorf("source candidate %q resolves outside the workspace", candidate)
		}
		targetFiles[candidate] = true
	}
	if err := addOverlayTargetFiles(ctx, boundary, canonicalDirectory, true, targetFiles); err != nil {
		return nil, err
	}
	if len(targetFiles) == 0 {
		return nil, fmt.Errorf("no Go source files found in directory %q", directory)
	}
	return sortedTargetFiles(targetFiles), nil
}

func resolveVirtualDirectoryTargetFiles(ctx context.Context, rootDir, directory string) ([]string, bool, error) {
	if _, err := canonicalTargetDirectory(ctx, directory); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("canonicalize virtual Go package directory %q: %w", directory, err)
	}
	files, err := resolveDirectoryTargetFiles(ctx, rootDir, directory)
	if err != nil {
		return nil, true, err
	}
	return files, true, nil
}

func canonicalTargetDirectory(ctx context.Context, directory string) (string, error) {
	canonical, err := filepath.EvalSymlinks(directory)
	if err == nil {
		return canonical, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	for path := range sourcebuf.Overlay(ctx) {
		canonicalPath, canonicalErr := sourcebuf.CanonicalPath(ctx, path)
		if canonicalErr != nil {
			return "", fmt.Errorf("canonicalize buffered package source %q: %w", path, canonicalErr)
		}
		candidate := filepath.Join(directory, filepath.Base(canonicalPath))
		candidatePath, candidateErr := sourcebuf.CanonicalPath(ctx, candidate)
		if candidateErr == nil && candidatePath == canonicalPath {
			return filepath.Dir(canonicalPath), nil
		}
	}
	return "", err
}

func addOverlayTargetFiles(ctx context.Context, boundary, directory string, directOnly bool, targetFiles map[string]bool) error {
	canonicalBoundary, err := sourcebuf.CanonicalPath(ctx, boundary)
	if err != nil {
		return fmt.Errorf("canonicalize symbol workspace %q: %w", boundary, err)
	}
	canonicalDirectory, err := canonicalTargetDirectory(ctx, directory)
	if err != nil {
		return fmt.Errorf("canonicalize symbol target directory %q: %w", directory, err)
	}
	for path := range sourcebuf.Overlay(ctx) {
		canonical, err := sourcebuf.CanonicalPath(ctx, path)
		if err != nil {
			return fmt.Errorf("canonicalize buffered symbol source %q: %w", path, err)
		}
		if _, err := os.Lstat(canonical); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect buffered symbol source %q: %w", path, err)
		}
		if !pathWithinDirectory(canonicalBoundary, canonical) || (directOnly && filepath.Dir(canonical) != canonicalDirectory) || !isResolvableGoSource(canonicalBoundary, canonical) {
			continue
		}
		targetFiles[canonical] = true
	}
	return nil
}

func isResolvableGoSource(root, path string) bool {
	if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
		return false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return false
	}
	for directory := filepath.Dir(relative); directory != "."; directory = filepath.Dir(directory) {
		base := filepath.Base(directory)
		if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") {
			return false
		}
	}
	return true
}

func sortedTargetFiles(files map[string]bool) []string {
	targets := make([]string, 0, len(files))
	for path := range files {
		targets = append(targets, path)
	}
	slices.Sort(targets)
	return targets
}
