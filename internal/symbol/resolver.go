// Package symbol provides AST symbol resolution and coordinate translation for Go source files.
package symbol

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Symbol represents an identified symbol in source code with exact coordinates.
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
	recv, name, err := ParseIdentifier(query)
	if err != nil {
		return nil, &SymbolError{Op: "resolve", File: filePath, Symbol: query, Err: err}
	}

	targetFiles, err := resolveTargetFiles(rootDir, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	packageIndexes := make(map[string]*packageTypeIndex)
	var declarations, usages []*Symbol
	for _, file := range targetFiles {
		symbols, scanErr := scanFile(file, recv, name, packageIndexes)
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
			return result, nil
		}
		result.Definition = declarations[0]
		setLegacyDefinition(result, result.Definition)
		return result, nil
	}

	if len(usages) == 0 {
		return nil, &SymbolError{Op: "resolve", File: filePath, Symbol: query, Err: ErrNotFound}
	}
	if len(usages) > 1 {
		result.Ambiguous = true
		result.Candidates = usages
		return result, nil
	}
	result.Definition = usages[0]
	setLegacyDefinition(result, result.Definition)
	return result, nil
}

func scanFile(filePath string, targetRecv string, targetName string, packageIndexes map[string]*packageTypeIndex) ([]*Symbol, error) {
	cleanPath := filepath.Clean(filePath)
	// #nosec G304 -- reading verified target go source files
	src, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, &SymbolError{Op: "read", File: cleanPath, Symbol: targetName, Err: err}
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		pos := extractPosition(fset, err)
		return nil, &SymbolError{Op: "parse", File: filePath, Symbol: targetName, Pos: pos, Err: err}
	}

	var index *packageTypeIndex
	if targetRecv != "" {
		for _, declaration := range node.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, raw := range general.Specs {
				spec, ok := raw.(*ast.TypeSpec)
				if !ok || spec.Name.Name != targetRecv {
					continue
				}
				_, isInterface := spec.Type.(*ast.InterfaceType)
				if !isInterface && embeddedInterfaceName(spec.Type) == "" {
					continue
				}
				key := filepath.Clean(filepath.Dir(filePath)) + "\x00" + node.Name.Name
				if strings.HasSuffix(filePath, "_test.go") {
					key += "\x00tests"
				}
				index = packageIndexes[key]
				if index == nil {
					index, err = loadPackageTypeIndex(filePath, node.Name.Name)
					if err != nil {
						return nil, fmt.Errorf("index interface declarations for %s: %w", filePath, err)
					}
					packageIndexes[key] = index
				}
				break
			}
			if index != nil {
				break
			}
		}
	}

	results, err := scanTopLevelDeclarations(fset, node, filePath, targetRecv, targetName, index)
	if err != nil {
		return nil, fmt.Errorf("resolve %s.%s in %s: %w", targetRecv, targetName, filePath, err)
	}
	if targetRecv == "" {
		results = append(results, scanFunctionVariables(fset, node, filePath, targetName)...)
	}
	return results, nil
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
