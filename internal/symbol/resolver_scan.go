// Package symbol keeps AST symbol collection here separate from file loading and parsing.
package symbol

import (
	"go/ast"
	"go/token"
)

func scanTopLevelDeclarations(fset *token.FileSet, node *ast.File, filePath, targetRecv, targetName string) []*Symbol {
	var results []*Symbol
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			recvName := extractReceiver(d.Recv)
			if d.Name.Name == targetName {
				if targetRecv == "" || targetRecv == recvName {
					pos := fset.Position(d.Name.Pos())
					kind := "function"
					if recvName != "" {
						kind = "method"
					}
					results = append(results, &Symbol{
						Name:     d.Name.Name,
						Receiver: recvName,
						Kind:     kind,
						File:     filePath,
						Line:     pos.Line,
						Column:   pos.Column,
						Offset:   pos.Offset,
					})
				}
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if targetRecv != "" && s.Name.Name == targetRecv {
						results = append(results, findStructFields(fset, filePath, s, targetName)...)
					}
					if targetRecv != "" {
						continue
					}
					if s.Name.Name == targetName {
						pos := fset.Position(s.Name.Pos())
						results = append(results, &Symbol{
							Name:   s.Name.Name,
							Kind:   "type",
							File:   filePath,
							Line:   pos.Line,
							Column: pos.Column,
							Offset: pos.Offset,
						})
					}
				case *ast.ValueSpec:
					if targetRecv != "" {
						continue
					}
					for _, ident := range s.Names {
						if ident.Name == targetName {
							pos := fset.Position(ident.Pos())
							results = append(results, &Symbol{
								Name:   ident.Name,
								Kind:   "value",
								File:   filePath,
								Line:   pos.Line,
								Column: pos.Column,
								Offset: pos.Offset,
							})
						}
					}
				}
			}
		}
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
