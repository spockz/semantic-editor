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
		switch decl := declaration.(type) {
		case *ast.FuncDecl:
			receiver := extractReceiver(decl.Recv)
			if decl.Name.Name == targetName && (targetRecv == "" || targetRecv == receiver) {
				position := fset.Position(decl.Name.Pos())
				kind := "function"
				if receiver != "" {
					kind = "method"
				}
				results = append(results, &Symbol{Name: decl.Name.Name, Receiver: receiver, Kind: kind, File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset})
			}
		case *ast.GenDecl:
			for _, raw := range decl.Specs {
				switch spec := raw.(type) {
				case *ast.TypeSpec:
					if targetRecv != "" && spec.Name.Name == targetRecv {
						members, err := findTypeMembers(fset, filePath, spec, targetName, index)
						if err != nil {
							return nil, err
						}
						results = append(results, members...)
					}
					if targetRecv != "" {
						continue
					}
					if spec.Name.Name == targetName {
						position := fset.Position(spec.Name.Pos())
						results = append(results, &Symbol{Name: spec.Name.Name, Kind: "type", File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset})
					}
				case *ast.ValueSpec:
					if targetRecv != "" {
						continue
					}
					for _, ident := range spec.Names {
						if ident.Name == targetName {
							position := fset.Position(ident.Pos())
							results = append(results, &Symbol{Name: ident.Name, Kind: "value", File: filePath, Line: position.Line, Column: position.Column, Offset: position.Offset})
						}
					}
				}
			}
		}
	}
	return results, nil
}

func scanFunctionVariables(fset *token.FileSet, node *ast.File, filePath, targetName string) []*Symbol {
	var results []*Symbol
	for _, decl := range node.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		appendVariables := func(fields *ast.FieldList, kind string) {
			if fields == nil {
				return
			}
			for _, field := range fields.List {
				for _, ident := range field.Names {
					appendLocalVariable(&results, fset, filePath, ident, targetName, kind)
				}
			}
		}
		appendVariables(function.Recv, "receiver")
		appendVariables(function.Type.Params, "parameter")
		appendVariables(function.Type.Results, "result")
		if function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			switch declaration := node.(type) {
			case *ast.AssignStmt:
				if declaration.Tok == token.DEFINE {
					for _, lhs := range declaration.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok {
							appendLocalVariable(&results, fset, filePath, ident, targetName, "variable")
						}
					}
				}
			case *ast.RangeStmt:
				if declaration.Tok == token.DEFINE {
					for _, expr := range []ast.Expr{declaration.Key, declaration.Value} {
						if ident, ok := expr.(*ast.Ident); ok {
							appendLocalVariable(&results, fset, filePath, ident, targetName, "variable")
						}
					}
				}
			case *ast.ValueSpec:
				for _, ident := range declaration.Names {
					appendLocalVariable(&results, fset, filePath, ident, targetName, "variable")
				}
			}
			return true
		})
	}

	return results
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

func loadPackageTypeIndex(filePath, packageName string) (*packageTypeIndex, error) {
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
		// #nosec G304 -- path is bounded to one regular Go file in the selected package directory.
		source, err := os.ReadFile(path)
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
