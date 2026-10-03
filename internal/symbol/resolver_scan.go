// Package symbol keeps AST symbol collection here separate from file loading and parsing.
package symbol

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func scanTopLevelDeclarations(fset *token.FileSet, node *ast.File, filePath, targetRecv, targetName string, index *packageTypeIndex) ([]*Symbol, error) {
	var results []*Symbol
	for _, declaration := range node.Decls {
		found, err := scanTopLevelDeclaration(fset, declaration, filePath, targetRecv, targetName, index)
		if err != nil {
			return nil, err
		}
		results = append(results, found...)
	}
	return results, nil
}

func scanTopLevelDeclaration(fset *token.FileSet, declaration ast.Decl, filePath, targetRecv, targetName string, index *packageTypeIndex) ([]*Symbol, error) {
	switch decl := declaration.(type) {
	case *ast.FuncDecl:
		return scanFunctionDeclaration(fset, decl, filePath, targetRecv, targetName), nil
	case *ast.GenDecl:
		return scanGeneralDeclaration(fset, decl, filePath, targetRecv, targetName, index)
	default:
		return nil, nil
	}
}

func scanFunctionDeclaration(fset *token.FileSet, declaration *ast.FuncDecl, filePath, targetRecv, targetName string) []*Symbol {
	receiver := extractReceiver(declaration.Recv)
	if declaration.Name.Name != targetName || targetRecv != "" && targetRecv != receiver {
		return nil
	}
	kind := "function"
	if receiver != "" {
		kind = "method"
	}
	position := fset.Position(declaration.Name.Pos())
	return []*Symbol{{Name: declaration.Name.Name, Receiver: receiver, Kind: kind, File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset}}
}

func scanGeneralDeclaration(fset *token.FileSet, declaration *ast.GenDecl, filePath, targetRecv, targetName string, index *packageTypeIndex) ([]*Symbol, error) {
	var results []*Symbol
	for _, spec := range declaration.Specs {
		found, err := scanGeneralSpec(fset, spec, filePath, targetRecv, targetName, index)
		if err != nil {
			return nil, err
		}
		results = append(results, found...)
	}
	return results, nil
}

func scanGeneralSpec(fset *token.FileSet, raw ast.Spec, filePath, targetRecv, targetName string, index *packageTypeIndex) ([]*Symbol, error) {
	switch spec := raw.(type) {
	case *ast.TypeSpec:
		return scanTypeSpec(fset, spec, filePath, targetRecv, targetName, index)
	case *ast.ValueSpec:
		return scanValueSpec(fset, spec, filePath, targetRecv, targetName), nil
	default:
		return nil, nil
	}
}

func scanTypeSpec(fset *token.FileSet, spec *ast.TypeSpec, filePath, targetRecv, targetName string, index *packageTypeIndex) ([]*Symbol, error) {
	if targetRecv != "" {
		if spec.Name.Name != targetRecv {
			return nil, nil
		}
		return findTypeMembers(fset, filePath, spec, targetName, index)
	}
	if spec.Name.Name != targetName {
		return nil, nil
	}
	position := fset.Position(spec.Name.Pos())
	return []*Symbol{{Name: spec.Name.Name, Kind: "type", File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset}}, nil
}

func scanValueSpec(fset *token.FileSet, spec *ast.ValueSpec, filePath, targetRecv, targetName string) []*Symbol {
	if targetRecv != "" {
		return nil
	}
	var results []*Symbol
	for _, ident := range spec.Names {
		if ident.Name != targetName {
			continue
		}
		position := fset.Position(ident.Pos())
		results = append(results, &Symbol{Name: ident.Name, Kind: "value", File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset})
	}
	return results
}

func scanFunctionVariables(fset *token.FileSet, node *ast.File, filePath, targetName string) []*Symbol {
	var results []*Symbol
	for _, decl := range node.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		results = appendFunctionVariables(results, fset, filePath, targetName, function)
	}
	return results
}

func appendFunctionVariables(results []*Symbol, fset *token.FileSet, filePath, targetName string, function *ast.FuncDecl) []*Symbol {
	appendFieldVariables(&results, fset, filePath, targetName, function.Recv, "receiver")
	appendFieldVariables(&results, fset, filePath, targetName, function.Type.Params, "parameter")
	appendFieldVariables(&results, fset, filePath, targetName, function.Type.Results, "result")
	if function.Body == nil {
		return results
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		appendDeclaredLocal(&results, fset, filePath, targetName, node)
		return true
	})
	return results
}

func appendFieldVariables(results *[]*Symbol, fset *token.FileSet, filePath, targetName string, fields *ast.FieldList, kind string) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, ident := range field.Names {
			appendLocalVariable(results, fset, filePath, ident, targetName, kind)
		}
	}
}

func appendDeclaredLocal(results *[]*Symbol, fset *token.FileSet, filePath, targetName string, node ast.Node) {
	switch declaration := node.(type) {
	case *ast.AssignStmt:
		if declaration.Tok == token.DEFINE {
			appendExpressionIdentifiers(results, fset, filePath, targetName, declaration.Lhs)
		}
	case *ast.RangeStmt:
		if declaration.Tok == token.DEFINE {
			appendExpressionIdentifiers(results, fset, filePath, targetName, []ast.Expr{declaration.Key, declaration.Value})
		}
	case *ast.ValueSpec:
		for _, ident := range declaration.Names {
			appendLocalVariable(results, fset, filePath, ident, targetName, "variable")
		}
	}
}

func appendExpressionIdentifiers(results *[]*Symbol, fset *token.FileSet, filePath, targetName string, expressions []ast.Expr) {
	for _, expression := range expressions {
		if ident, ok := expression.(*ast.Ident); ok {
			appendLocalVariable(results, fset, filePath, ident, targetName, "variable")
		}
	}
}

func findInterfaceMethods(index *packageTypeIndex, root indexedTypeDeclaration, targetName string) ([]*Symbol, error) {
	var results []*Symbol
	seenDeclarations := make(map[string]struct{})
	// Persistent identity memoization breaks cycles and bounds diamond traversal.
	visited := make(map[string]bool)
	var visit func(indexedTypeDeclaration, string)
	visit = func(current indexedTypeDeclaration, queriedName string) {
		declarationID := fmt.Sprintf("%s:%d", current.source.path, current.source.fset.Position(current.spec.Pos()).Offset)
		if visited[declarationID] {
			return
		}
		visited[declarationID] = true
		interfaceType, isInterface := current.spec.Type.(*ast.InterfaceType)
		if !isInterface {
			underlyingName := embeddedInterfaceName(current.spec.Type)
			for _, target := range index.types[underlyingName] {
				visit(target, queriedName)
			}
			return
		}
		for _, field := range interfaceType.Methods.List {
			if len(field.Names) > 0 {
				for _, method := range field.Names {
					if method.Name != targetName {
						continue
					}
					position := current.source.fset.Position(method.Pos())
					methodID := fmt.Sprintf("%s:%d", current.source.path, position.Offset)
					if _, exists := seenDeclarations[methodID]; exists {
						continue
					}
					seenDeclarations[methodID] = struct{}{}
					results = append(results, &Symbol{Name: method.Name, Receiver: queriedName, Kind: "method", File: current.source.path, Line: position.Line, Column: position.Column, Offset: position.Offset})
				}
				continue
			}
			embeddedName := embeddedInterfaceName(field.Type)
			if embeddedName == "" {
				continue
			}
			for _, embedded := range index.types[embeddedName] {
				visit(embedded, queriedName)
			}
		}
	}
	visit(root, root.spec.Name.Name)
	return results, nil
}

func embeddedInterfaceName(expression ast.Expr) string {
	switch node := expression.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.IndexExpr:
		if name, ok := node.X.(*ast.Ident); ok {
			return name.Name
		}
	case *ast.IndexListExpr:
		if name, ok := node.X.(*ast.Ident); ok {
			return name.Name
		}
	}
	return ""
}

type packageSourceFile struct {
	path string
	fset *token.FileSet
}

type indexedTypeDeclaration struct {
	spec   *ast.TypeSpec
	source packageSourceFile
}

type packageTypeIndex struct {
	types map[string][]indexedTypeDeclaration
}

func loadPackageTypeIndex(filePath, packageName string, reader *sourceSnapshotReader) (*packageTypeIndex, error) {
	selectedIsTest := strings.HasSuffix(filePath, "_test.go")
	entries, err := os.ReadDir(filepath.Dir(filePath))
	if err != nil {
		return nil, fmt.Errorf("read package directory for %s: %w", filePath, err)
	}
	index := &packageTypeIndex{types: make(map[string][]indexedTypeDeclaration)}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(name, ".go") || (!selectedIsTest && strings.HasSuffix(name, "_test.go")) {
			continue
		}
		path := filepath.Join(filepath.Dir(filePath), name)
		source, err := reader.read(path)
		if err != nil {
			return nil, fmt.Errorf("read package source %s: %w", path, err)
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, source, 0)
		if file != nil && file.Name.Name != packageName {
			continue
		}
		if parseErr != nil {
			return nil, fmt.Errorf("parse package source %s: %w", path, parseErr)
		}
		if file == nil {
			return nil, fmt.Errorf("parse package source %s: parser returned no syntax tree", path)
		}
		sourceFile := packageSourceFile{path: path, fset: fset}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, raw := range general.Specs {
				spec, ok := raw.(*ast.TypeSpec)
				if !ok {
					continue
				}
				index.types[spec.Name.Name] = append(index.types[spec.Name.Name], indexedTypeDeclaration{spec: spec, source: sourceFile})
			}
		}
	}
	return index, nil
}
