// Package astedit extracts exact declarations selected by the shared Go symbol resolver.
package astedit

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/symbol"
	"slices"
	"strings"
)

// InspectGo returns exact declaration snapshots for symbols selected by the resolver.
func InspectGo(root, file, query string) (*backend.InspectResult, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Go workspace %q: %w", root, err)
	}
	selectedFile := file
	if selectedFile != "" && !filepath.IsAbs(selectedFile) {
		selectedFile = filepath.Join(absoluteRoot, selectedFile)
	}
	snapshot, err := symbol.ResolveSnapshot(absoluteRoot, selectedFile, query)
	if err != nil {
		return nil, err
	}
	candidates := make([]*symbol.Symbol, 0, 1)
	if snapshot.Result.Ambiguous {
		candidates = append(candidates, snapshot.Result.Candidates...)
	} else if snapshot.Result.Definition != nil {
		candidates = append(candidates, snapshot.Result.Definition)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("symbol resolver returned no inspection candidates for %q", query)
	}
	slices.SortFunc(candidates, func(left, right *symbol.Symbol) int {
		if order := strings.Compare(left.File, right.File); order != 0 {
			return order
		}
		return cmp.Compare(left.Offset, right.Offset)
	})
	result := &backend.InspectResult{
		Scope: backend.ReadScope{
			Kind:     "workspace_syntax",
			Path:     ".",
			Complete: true,
		},
		Matches: make([]backend.InspectedSymbol, 0, len(candidates)),
	}
	if selectedFile != "" {
		result.Scope.Kind = "selected_file"
		result.Scope.Path = displayGoPath(absoluteRoot, selectedFile)
	}
	for _, candidate := range candidates {
		candidatePath := candidate.File
		if !filepath.IsAbs(candidatePath) {
			candidatePath = filepath.Join(absoluteRoot, candidatePath)
		}
		candidatePath, err = filepath.Abs(candidatePath)
		if err != nil {
			return nil, fmt.Errorf("resolve inspection candidate path %q: %w", candidate.File, err)
		}
		source, ok := snapshot.Files[filepath.Clean(candidatePath)]
		if !ok {
			return nil, fmt.Errorf("inspection snapshot omitted resolved source %s", candidatePath)
		}
		parsed, err := parseGoReadSource(absoluteRoot, candidatePath, source)
		if err != nil {
			return nil, err
		}
		target, err := findGoInspectTarget(parsed, candidate)
		if err != nil {
			return nil, err
		}
		start := target.start
		if !start.IsValid() {
			start = target.node.Pos()
		}
		end := target.end
		if !end.IsValid() {
			end = target.node.End()
		}
		sourceText, err := goReadSpan(source, parsed.fset, start, end)
		if err != nil {
			return nil, err
		}
		rng, err := goReadRange(source, parsed.fset, start, end)
		if err != nil {
			return nil, err
		}
		selectionRange, err := goReadRange(source, parsed.fset, target.selection.Pos(), target.selection.End())
		if err != nil {
			return nil, err
		}
		doc, err := goReadComment(source, parsed.fset, target.doc)
		if err != nil {
			return nil, err
		}
		imports, err := goReadImports(parsed.file)
		if err != nil {
			return nil, err
		}
		qualifiedName := target.qualified
		if qualifiedName == "" {
			qualifiedName = target.name
		}
		result.Matches = append(result.Matches, backend.InspectedSymbol{
			Name:           target.name,
			QualifiedName:  qualifiedName,
			Kind:           target.kind,
			File:           parsed.displayPath,
			Range:          rng,
			SelectionRange: selectionRange,
			Signature:      target.signature,
			Doc:            doc,
			Type:           target.typeValue,
			Tag:            target.tag,
			Source:         sourceText,
			Revision:       goReadRevision(source),
			SourceExtent:   "declaration",
			Package:        parsed.file.Name.Name,
			Imports:        imports,
		})
	}
	return result, nil
}

type goInspectTarget struct {
	name      string
	qualified string
	kind      string
	node      ast.Node
	start     token.Pos
	end       token.Pos
	selection *ast.Ident
	signature *string
	doc       *ast.CommentGroup
	typeValue *string
	tag       *string
}

//nolint:gocognit,funlen // The resolver offset disambiguates AST declaration shapes in one pass.
func findGoInspectTarget(parsed *goReadSource, candidate *symbol.Symbol) (*goInspectTarget, error) {
	matchesOffset := func(ident *ast.Ident) bool {
		return ident != nil && parsed.fset.PositionFor(ident.Pos(), false).Offset == candidate.Offset && ident.Name == candidate.Name
	}
	qualified := candidate.QualifiedName
	if qualified == "" {
		qualified = candidate.BuildQualifiedName()
	}
	for _, declaration := range parsed.file.Decls {
		switch decl := declaration.(type) {
		case *ast.FuncDecl:
			if !matchesOffset(decl.Name) {
				continue
			}
			kind := "function"
			if decl.Recv != nil {
				kind = "method"
			}
			signatureDecl := *decl
			signatureDecl.Doc = nil
			signatureDecl.Body = nil
			signature, err := goReadFormattedNode(parsed.fset, &signatureDecl)
			if err != nil {
				return nil, err
			}
			return &goInspectTarget{
				name: decl.Name.Name, qualified: qualified, kind: kind, node: decl,
				start: decl.Type.Func, end: decl.End(), selection: decl.Name, signature: &signature, doc: decl.Doc,
			}, nil
		case *ast.GenDecl:
			for _, rawSpec := range decl.Specs {
				switch spec := rawSpec.(type) {
				case *ast.TypeSpec:
					if !matchesOffset(spec.Name) {
						if typeNode, ok := spec.Type.(*ast.StructType); ok {
							for _, field := range typeNode.Fields.List {
								for _, ident := range field.Names {
									if matchesOffset(ident) && candidate.Kind == "field" {
										return goInspectFieldTarget(parsed, qualified, field, ident)
									}
								}
							}
						}
						if interfaceNode, ok := spec.Type.(*ast.InterfaceType); ok {
							for _, field := range interfaceNode.Methods.List {
								for _, ident := range field.Names {
									if matchesOffset(ident) && candidate.Kind == "method" {
										signature, err := goReadSpan(parsed.source, parsed.fset, field.Names[0].Pos(), field.End())
										if err != nil {
											return nil, err
										}
										return &goInspectTarget{
											name: ident.Name, qualified: qualified, kind: "method", node: field,
											start: field.Names[0].Pos(), end: field.End(), selection: ident,
											signature: &signature, doc: field.Doc,
										}, nil
									}
								}
							}
						}
						continue
					}
					singleDecl := *decl
					singleDecl.Doc = nil
					singleDecl.Specs = []ast.Spec{spec}
					singleDecl.Lparen = token.NoPos
					signature, err := goReadFormattedNode(parsed.fset, &singleDecl)
					if err != nil {
						return nil, err
					}
					doc := spec.Doc
					if doc == nil {
						doc = decl.Doc
					}
					typeKind := "type"
					if _, ok := spec.Type.(*ast.InterfaceType); ok {
						typeKind = "interface"
					}
					return &goInspectTarget{
						name: spec.Name.Name, qualified: qualified, kind: typeKind, node: decl,
						start: decl.TokPos, end: decl.End(), selection: spec.Name,
						signature: &signature, doc: doc,
					}, nil
				case *ast.ValueSpec:
					for _, ident := range spec.Names {
						if !matchesOffset(ident) {
							continue
						}
						kind := "variable"
						prefix := "var "
						if decl.Tok == token.CONST {
							kind = "constant"
							prefix = "const "
						}
						specText, err := goReadFormattedNode(parsed.fset, spec)
						if err != nil {
							return nil, err
						}
						signature := prefix + specText
						var typeValue *string
						if spec.Type != nil {
							text, err := goReadNodeText(parsed.source, parsed.fset, spec.Type)
							if err != nil {
								return nil, err
							}
							typeValue = &text
						}
						if kind == "variable" {
							signature = "var " + ident.Name
							if typeValue != nil {
								signature += " " + *typeValue
							}
						}
						doc := spec.Doc
						if doc == nil {
							doc = decl.Doc
						}
						return &goInspectTarget{
							name: ident.Name, qualified: qualified, kind: kind, node: decl,
							start: decl.TokPos, end: decl.End(), selection: ident,
							signature: &signature, doc: doc, typeValue: typeValue,
						}, nil
					}
				}
			}
		}
	}
	var target *goInspectTarget
	for _, declaration := range parsed.file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		var localErr error
		checkFields := func(fields *ast.FieldList, kind string) bool {
			if fields == nil {
				return false
			}
			for _, field := range fields.List {
				for _, ident := range field.Names {
					if !matchesOffset(ident) {
						continue
					}
					typeValue, err := goReadNodeText(parsed.source, parsed.fset, field.Type)
					if err != nil {
						localErr = err
						return false
					}
					start := field.Names[0].Pos()
					signature, err := goReadSpan(parsed.source, parsed.fset, start, field.End())
					if err != nil {
						localErr = err
						return false
					}
					target = &goInspectTarget{
						name: ident.Name, qualified: qualified, kind: kind, node: field,
						start: start, end: field.End(), selection: ident, signature: &signature, doc: field.Doc,
						typeValue: &typeValue,
					}
					return true
				}
			}
			return false
		}
		if checkFields(fn.Recv, "receiver") || checkFields(fn.Type.Params, "parameter") || checkFields(fn.Type.Results, "result") {
			return target, nil
		}
		if localErr != nil {
			return nil, localErr
		}
		if fn.Body == nil {
			continue
		}
		var found bool
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if found {
				return false
			}
			switch local := node.(type) {
			case *ast.GenDecl:
				for _, rawSpec := range local.Specs {
					valueSpec, ok := rawSpec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, ident := range valueSpec.Names {
						if !matchesOffset(ident) {
							continue
						}
						kind, prefix := "variable", "var "
						if local.Tok == token.CONST {
							kind, prefix = "constant", "const "
						}
						signature, err := goReadFormattedNode(parsed.fset, valueSpec)
						if err != nil {
							localErr, found = err, true
							return false
						}
						signature = prefix + signature
						var typeValue *string
						if valueSpec.Type != nil {
							text, err := goReadNodeText(parsed.source, parsed.fset, valueSpec.Type)
							if err != nil {
								localErr, found = err, true
								return false
							}
							typeValue = &text
						}
						if kind == "variable" {
							signature = prefix + ident.Name
							if typeValue != nil {
								signature += " " + *typeValue
							}
						}
						doc := valueSpec.Doc
						if doc == nil {
							doc = local.Doc
						}
						target = &goInspectTarget{
							name: ident.Name, qualified: qualified, kind: kind, node: local,
							start: local.TokPos, end: local.End(), selection: ident,
							signature: &signature, doc: doc, typeValue: typeValue,
						}
						found = true
						return false
					}
				}
			case *ast.AssignStmt:
				for _, expression := range local.Lhs {
					if ident, ok := expression.(*ast.Ident); ok && matchesOffset(ident) {
						target = &goInspectTarget{name: ident.Name, qualified: qualified, kind: "variable", node: local, start: local.Pos(), end: local.End(), selection: ident}
						found = true
						return false
					}
				}
			case *ast.RangeStmt:
				for _, expression := range []ast.Expr{local.Key, local.Value} {
					if ident, ok := expression.(*ast.Ident); ok && matchesOffset(ident) {
						target = &goInspectTarget{name: ident.Name, qualified: qualified, kind: "variable", node: local, start: local.Pos(), end: local.End(), selection: ident}
						found = true
						return false
					}
				}
			}
			return true
		})
		if found {
			if localErr != nil {
				return nil, localErr
			}
			return target, nil
		}
	}
	return nil, fmt.Errorf("project resolved symbol %s at %s:%d but its snapshot declaration was not found", candidate.Name, candidate.File, candidate.Offset)
}

func goInspectFieldTarget(parsed *goReadSource, qualified string, field *ast.Field, selection *ast.Ident) (*goInspectTarget, error) {
	if selection == nil {
		return nil, fmt.Errorf("resolver selected an unsupported embedded field declaration")
	}
	typeValue, err := goReadNodeText(parsed.source, parsed.fset, field.Type)
	if err != nil {
		return nil, err
	}
	start := field.Type.Pos()
	if len(field.Names) > 0 {
		start = field.Names[0].Pos()
	}
	signature, err := goReadSpan(parsed.source, parsed.fset, start, field.End())
	if err != nil {
		return nil, err
	}
	var tag *string
	if field.Tag != nil {
		value, err := goReadNodeText(parsed.source, parsed.fset, field.Tag)
		if err != nil {
			return nil, err
		}
		tag = &value
	}
	name := selection.Name
	return &goInspectTarget{
		name: name, qualified: qualified, kind: "field", node: field,
		start: start, end: field.End(), selection: selection,
		signature: &signature, doc: field.Doc, typeValue: &typeValue, tag: tag,
	}, nil
}
