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
	"path/filepath"
	"semedit/internal/sourcebuf"
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
	Symbol     string  `json:"symbol,omitempty"`
	File       string  `json:"file,omitempty"`
	Line       int     `json:"line,omitempty"`
	Column     int     `json:"column,omitempty"`
	Offset     int     `json:"offset,omitempty"`
	Kind       string  `json:"kind,omitempty"`
	Receiver   string  `json:"receiver,omitempty"`
	Definition *Symbol `json:"definition,omitempty"`
	// Usages are syntactic local binding declarations, not semantic references.
	Usages     []*Symbol `json:"usages,omitempty"`
	Ambiguous  bool      `json:"ambiguous,omitempty"`
	Candidates []*Symbol `json:"candidates,omitempty"`
}

// ParseIdentifier decomposes a query identifier into receiver and symbol name.
func ParseIdentifier(raw string) (receiver string, name string, err error) {
	clean := normalizeInput(raw)
	if clean == "" {
		return "", "", fmt.Errorf("%w: identifier cannot be empty", ErrInvalidIdentifier)
	}

	parts := strings.Split(clean, ".")
	switch len(parts) {
	case 1:
		return "", parts[0], nil
	case 2:
		recv := strings.TrimPrefix(parts[0], "*")
		recv = strings.TrimPrefix(recv, "(")
		recv = strings.TrimSuffix(recv, ")")
		recv = strings.TrimPrefix(recv, "*")
		return recv, parts[1], nil
	default:
		return "", "", fmt.Errorf("%w: %q (expected [Receiver.]Name)", ErrInvalidIdentifier, raw)
	}
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

func scanFile(filePath string, targetRecv string, targetName string, packageIndexes map[string]*packageTypeIndex, reader *sourceSnapshotReader) ([]*Symbol, error) {
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

func resolveTargetFiles(rootDir, filePath string) ([]string, error) {
	if filePath != "" {
		resolvedPath := filePath
		if !filepath.IsAbs(resolvedPath) && rootDir != "" {
			resolvedPath = filepath.Join(rootDir, filePath)
		}
		return []string{resolvedPath}, nil
	}
	searchRoot := rootDir
	if searchRoot == "" {
		searchRoot = "."
	}
	rootPath := filepath.Clean(searchRoot)
	var targetFiles []string
	err := filepath.WalkDir(searchRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if filepath.Clean(path) != rootPath {
				base := entry.Name()
				if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			targetFiles = append(targetFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return targetFiles, nil
}

func isLocalBinding(kind string) bool {
	switch kind {
	case "receiver", "parameter", "result", "variable":
		return true
	default:
		return false
	}
}

func setLegacyDefinition(result *LookupResult, definition *Symbol) {
	result.Symbol = definition.BuildQualifiedName()
	result.File = definition.File
	result.Line = definition.Line
	result.Column = definition.Column
	result.Offset = definition.Offset
	result.Kind = definition.Kind
	result.Receiver = definition.Receiver
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
	recv, name, err := ParseIdentifier(query)
	if err != nil {
		return nil, &SymbolError{Op: "resolve", File: filePath, Symbol: query, Err: err}
	}

	targetFiles, err := resolveTargetFiles(rootDir, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	reader := newSourceSnapshotReader()
	reader.ctx = ctx
	packageIndexes := make(map[string]*packageTypeIndex)
	var declarations, usages []*Symbol
	for _, file := range targetFiles {
		symbols, scanErr := scanFile(file, recv, name, packageIndexes, reader)
		if scanErr != nil {
			return nil, scanErr
		}
		for _, candidate := range symbols {
			candidate.QualifiedName = candidate.BuildQualifiedName()
			if rootDir != "" {
				if relative, relErr := filepath.Rel(rootDir, candidate.File); relErr == nil && !strings.HasPrefix(relative, "..") {
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

	result := &LookupResult{Usages: usages}
	if len(declarations) > 0 {
		if len(declarations) > 1 {
			result.Ambiguous = true
			result.Candidates = declarations
			return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
		}
		result.Definition = declarations[0]
		setLegacyDefinition(result, result.Definition)
		return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
	}

	if len(usages) == 0 {
		return nil, &SymbolError{Op: "resolve", File: filePath, Symbol: query, Err: ErrNotFound}
	}
	if len(usages) > 1 {
		result.Ambiguous = true
		result.Candidates = usages
		return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
	}
	result.Definition = usages[0]
	setLegacyDefinition(result, result.Definition)
	return &ResolutionSnapshot{Result: result, Files: reader.snapshot()}, nil
}
