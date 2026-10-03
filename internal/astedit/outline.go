// Package astedit builds file-centric declaration outlines from exact Go source snapshots.
package astedit

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"strings"
)

// OutlineGo produces deterministic Go declarations from selected source files.
//
//nolint:gocognit,funlen // One file walk preserves the ordering and scope of each declaration projection.
func OutlineGo(root, path string, kinds []string, includeUnexported, includeTests bool) (*backend.OutlineResult, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("outline path is required")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Go workspace %q: %w", root, err)
	}
	target := path
	if !filepath.IsAbs(target) {
		target = filepath.Join(absoluteRoot, target)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve outline path %q: %w", path, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("stat outline path %q: %w", path, err)
	}
	directory := info.IsDir()
	if linkInfo, linkErr := os.Lstat(target); linkErr == nil && linkInfo.Mode()&os.ModeSymlink != 0 && directory {
		return nil, fmt.Errorf("outline directory path %q is a symlink", path)
	}
	if !directory && (!info.Mode().IsRegular() || filepath.Ext(target) != ".go") {
		return nil, fmt.Errorf("outline path %q is not a Go source file or directory", path)
	}
	kindSet := make(map[string]bool, len(kinds))
	for _, kind := range kinds {
		if !backend.IsReadOutlineKind(kind) {
			return nil, fmt.Errorf("unknown outline kind %q", kind)
		}
		kindSet[kind] = true
	}
	files, err := pipeline.GoFiles(absoluteRoot, target)
	if err != nil {
		return nil, fmt.Errorf("select Go outline files under %q: %w", path, err)
	}
	if !directory && len(files) == 0 {
		return nil, fmt.Errorf("outline path %q contains no Go source file", path)
	}
	result := &backend.OutlineResult{
		Scope: backend.ReadScope{
			Kind:     "selected_file",
			Path:     displayGoPath(absoluteRoot, target),
			Complete: true,
		},
		Files: make([]backend.OutlineFile, 0, len(files)),
	}
	if directory {
		result.Scope.Kind = "directory_syntax"
		result.Scope.Path = displayGoPath(absoluteRoot, target)
	}
	matchesKind := func(kind string) bool {
		return len(kindSet) == 0 || kindSet[kind]
	}
	for _, filePath := range files {
		if directory && !includeTests && strings.HasSuffix(filepath.Base(filePath), "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Clean(filePath)) // #nosec G304 -- GoFiles returns the bounded selected source set.
		if err != nil {
			return nil, fmt.Errorf("read Go outline source %s: %w", filePath, err)
		}
		parsed, err := parseGoReadSource(absoluteRoot, filePath, source)
		if err != nil {
			return nil, err
		}
		doc, err := goReadComment(source, parsed.fset, parsed.file.Doc)
		if err != nil {
			return nil, err
		}
		imports, err := goReadImports(parsed.file)
		if err != nil {
			return nil, err
		}
		fileResult := backend.OutlineFile{
			File:     parsed.displayPath,
			Language: backend.LanguageGo,
			Package:  parsed.file.Name.Name,
			Revision: goReadRevision(source),
			Doc:      doc,
			Imports:  imports,
			Symbols:  make([]backend.ReadSymbol, 0),
		}
		for _, declaration := range parsed.file.Decls {
			switch decl := declaration.(type) {
			case *ast.FuncDecl:
				kind := "function"
				if decl.Recv != nil {
					kind = "method"
				}
				receiver := ""
				if decl.Recv != nil {
					receiver = goReceiverName(decl.Recv)
				}
				visible := includeUnexported || ast.IsExported(decl.Name.Name)
				if receiver != "" && !includeUnexported && !ast.IsExported(receiver) {
					visible = false
				}
				if !matchesKind(kind) || !visible {
					continue
				}
				signatureDecl := *decl
				signatureDecl.Doc = nil
				signatureDecl.Body = nil
				signature, err := goReadFormattedNode(parsed.fset, &signatureDecl)
				if err != nil {
					return nil, err
				}
				doc, err := goReadComment(source, parsed.fset, decl.Doc)
				if err != nil {
					return nil, err
				}
				qualifiedName := decl.Name.Name
				if receiver != "" {
					qualifiedName = receiver + "." + decl.Name.Name
				}
				readSymbol, err := makeOutlineSymbol(parsed, decl.Name.Name, qualifiedName, kind, decl, decl.Name, &signature, doc, nil, nil, nil)
				if err != nil {
					return nil, err
				}
				fileResult.Symbols = append(fileResult.Symbols, readSymbol)
			case *ast.GenDecl:
				for _, rawSpec := range decl.Specs {
					switch spec := rawSpec.(type) {
					case *ast.TypeSpec:
						typeKind := "type"
						if _, ok := spec.Type.(*ast.InterfaceType); ok {
							typeKind = "interface"
						}
						if !includeUnexported && !ast.IsExported(spec.Name.Name) {
							continue
						}
						children, childMatch, err := outlineTypeChildren(parsed, spec, kindSet, includeUnexported)
						if err != nil {
							return nil, err
						}
						if !matchesKind(typeKind) && !childMatch {
							continue
						}
						signature, err := outlineFilteredTypeSignature(parsed, decl, spec, kindSet, includeUnexported)
						if err != nil {
							return nil, err
						}
						docGroup := spec.Doc
						if docGroup == nil {
							docGroup = decl.Doc
						}
						doc, err := goReadComment(source, parsed.fset, docGroup)
						if err != nil {
							return nil, err
						}
						readSymbol, err := makeOutlineSymbol(parsed, spec.Name.Name, spec.Name.Name, typeKind, spec, spec.Name, &signature, doc, nil, nil, children)
						if err != nil {
							return nil, err
						}
						fileResult.Symbols = append(fileResult.Symbols, readSymbol)
					case *ast.ValueSpec:
						kind := "variable"
						prefix := "var "
						if decl.Tok == token.CONST {
							kind = "constant"
							prefix = "const "
						}
						if !matchesKind(kind) {
							continue
						}
						docGroup := spec.Doc
						if docGroup == nil {
							docGroup = decl.Doc
						}
						doc, err := goReadComment(source, parsed.fset, docGroup)
						if err != nil {
							return nil, err
						}
						var typeText *string
						if spec.Type != nil {
							text, err := goReadNodeText(source, parsed.fset, spec.Type)
							if err != nil {
								return nil, err
							}
							typeText = &text
						}
						signatureText, err := goReadFormattedNode(parsed.fset, spec)
						if err != nil {
							return nil, err
						}
						if kind == "variable" {
							names := make([]string, len(spec.Names))
							for i, name := range spec.Names {
								names[i] = name.Name
							}
							signatureText = prefix + strings.Join(names, ", ")
							if typeText != nil {
								signatureText += " " + *typeText
							}
						} else {
							signatureText = prefix + signatureText
						}
						for nameIndex, name := range spec.Names {
							if !includeUnexported && !ast.IsExported(name.Name) {
								continue
							}
							nameSignature := signatureText
							if kind == "variable" && len(spec.Names) > 1 {
								nameSignature = prefix + name.Name
								if typeText != nil {
									nameSignature += " " + *typeText
								}
							}
							if kind == "constant" && len(spec.Names) > 1 {
								nameSignature = prefix + name.Name
								if typeText != nil {
									nameSignature += " " + *typeText
								}
								if nameIndex < len(spec.Values) {
									valueText, err := goReadNodeText(source, parsed.fset, spec.Values[nameIndex])
									if err != nil {
										return nil, err
									}
									nameSignature += " = " + valueText
								}
							}
							readSymbol, err := makeOutlineSymbol(parsed, name.Name, name.Name, kind, spec, name, &nameSignature, doc, typeText, nil, nil)
							if err != nil {
								return nil, err
							}
							fileResult.Symbols = append(fileResult.Symbols, readSymbol)
						}
					}
				}
			}
		}
		result.Files = append(result.Files, fileResult)
	}
	return result, nil
}

func displayGoPath(root, path string) string {
	relativePath, err := filepath.Rel(root, path)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relativePath)
}

func makeOutlineSymbol(parsed *goReadSource, name, qualifiedName, kind string, node, selection ast.Node, signature, doc, typeValue, tag *string, children []backend.ReadSymbol) (backend.ReadSymbol, error) {
	rng, err := goReadRange(parsed.source, parsed.fset, node.Pos(), node.End())
	if err != nil {
		return backend.ReadSymbol{}, err
	}
	selectionRange, err := goReadRange(parsed.source, parsed.fset, selection.Pos(), selection.End())
	if err != nil {
		return backend.ReadSymbol{}, err
	}
	readSymbol := backend.ReadSymbol{
		Name:           name,
		QualifiedName:  qualifiedName,
		Kind:           kind,
		File:           parsed.displayPath,
		Range:          rng,
		SelectionRange: selectionRange,
		Signature:      signature,
		Doc:            doc,
		Type:           typeValue,
		Tag:            tag,
	}
	if len(children) > 0 {
		readSymbol.Children = children
	}
	return readSymbol, nil
}

func outlineTypeChildren(parsed *goReadSource, spec *ast.TypeSpec, kinds map[string]bool, includeUnexported bool) ([]backend.ReadSymbol, bool, error) {
	switch node := spec.Type.(type) {
	case *ast.StructType:
		children, err := outlineStructChildren(parsed, spec, node, kinds, includeUnexported)
		return children, len(children) > 0, err
	case *ast.InterfaceType:
		children, err := outlineInterfaceChildren(parsed, spec, node, kinds, includeUnexported)
		return children, len(children) > 0, err
	default:
		return make([]backend.ReadSymbol, 0), false, nil
	}
}

func outlineStructChildren(parsed *goReadSource, spec *ast.TypeSpec, structure *ast.StructType, kinds map[string]bool, includeUnexported bool) ([]backend.ReadSymbol, error) {
	if !outlineKindMatches(kinds, "field") {
		return make([]backend.ReadSymbol, 0), nil
	}
	children := make([]backend.ReadSymbol, 0)
	for _, field := range structure.Fields.List {
		items, err := outlineStructField(parsed, spec.Name.Name, field, includeUnexported)
		if err != nil {
			return nil, err
		}
		children = append(children, items...)
	}
	return children, nil
}

func outlineStructField(parsed *goReadSource, owner string, field *ast.Field, includeUnexported bool) ([]backend.ReadSymbol, error) {
	typeValue, err := goReadNodeText(parsed.source, parsed.fset, field.Type)
	if err != nil {
		return nil, err
	}
	tag, err := outlineFieldTag(parsed, field.Tag)
	if err != nil {
		return nil, err
	}
	doc, err := goReadComment(parsed.source, parsed.fset, field.Doc)
	if err != nil {
		return nil, err
	}
	signature, err := outlineFieldSignature(parsed, field)
	if err != nil {
		return nil, err
	}
	if len(field.Names) == 0 {
		return outlineEmbeddedStructField(parsed, owner, field, typeValue, signature, doc, tag.text, includeUnexported)
	}
	return outlineNamedStructFields(parsed, owner, field, typeValue, doc, tag.text, includeUnexported)
}

type outlineFieldTagValue struct {
	text *string
}

func outlineFieldTag(parsed *goReadSource, tagNode *ast.BasicLit) (outlineFieldTagValue, error) {
	if tagNode == nil {
		return outlineFieldTagValue{}, nil
	}
	tag, err := goReadNodeText(parsed.source, parsed.fset, tagNode)
	if err != nil {
		return outlineFieldTagValue{}, err
	}
	return outlineFieldTagValue{text: &tag}, nil
}

func outlineFieldSignature(parsed *goReadSource, field *ast.Field) (string, error) {
	start := field.Type.Pos()
	if len(field.Names) > 0 {
		start = field.Names[0].Pos()
	}
	return goReadSpan(parsed.source, parsed.fset, start, field.End())
}

func outlineEmbeddedStructField(parsed *goReadSource, owner string, field *ast.Field, typeValue, signature string, doc *string, tag *string, includeUnexported bool) ([]backend.ReadSymbol, error) {
	name := typeValue
	selection := ast.Node(field.Type)
	if base := goEmbeddedBaseIdent(field.Type); base != nil {
		name = base.Name
		selection = base
	}
	if !includeUnexported && !embeddedGoNameExported(field.Type) {
		return make([]backend.ReadSymbol, 0), nil
	}
	readSymbol, err := makeOutlineSymbol(parsed, name, owner+"."+name, "field", field, selection, &signature, doc, &typeValue, tag, nil)
	if err != nil {
		return nil, err
	}
	return []backend.ReadSymbol{readSymbol}, nil
}

func outlineNamedStructFields(parsed *goReadSource, owner string, field *ast.Field, typeValue string, doc *string, tag *string, includeUnexported bool) ([]backend.ReadSymbol, error) {
	children := make([]backend.ReadSymbol, 0, len(field.Names))
	for _, ident := range field.Names {
		if !includeUnexported && !ast.IsExported(ident.Name) {
			continue
		}
		signature := ident.Name + " " + typeValue
		if tag != nil {
			signature += " " + *tag
		}
		readSymbol, err := makeOutlineSymbol(parsed, ident.Name, owner+"."+ident.Name, "field", field, ident, &signature, doc, &typeValue, tag, nil)
		if err != nil {
			return nil, err
		}
		children = append(children, readSymbol)
	}
	return children, nil
}

func outlineInterfaceChildren(parsed *goReadSource, spec *ast.TypeSpec, interfaceType *ast.InterfaceType, kinds map[string]bool, includeUnexported bool) ([]backend.ReadSymbol, error) {
	children := make([]backend.ReadSymbol, 0)
	for _, field := range interfaceType.Methods.List {
		var items []backend.ReadSymbol
		var err error
		if len(field.Names) > 0 {
			if !outlineKindMatches(kinds, "method") {
				continue
			}
			items, err = outlineInterfaceMethods(parsed, spec.Name.Name, field, includeUnexported)
		} else {
			if !outlineKindMatches(kinds, "type") {
				continue
			}
			items, err = outlineEmbeddedInterfaceType(parsed, spec.Name.Name, field, includeUnexported)
		}
		if err != nil {
			return nil, err
		}
		children = append(children, items...)
	}
	return children, nil
}

func outlineInterfaceMethods(parsed *goReadSource, owner string, field *ast.Field, includeUnexported bool) ([]backend.ReadSymbol, error) {
	doc, err := goReadComment(parsed.source, parsed.fset, field.Doc)
	if err != nil {
		return nil, err
	}
	typeText, err := goReadNodeText(parsed.source, parsed.fset, field.Type)
	if err != nil {
		return nil, err
	}
	children := make([]backend.ReadSymbol, 0, len(field.Names))
	for _, ident := range field.Names {
		if !includeUnexported && !ast.IsExported(ident.Name) {
			continue
		}
		signature := ident.Name + " " + typeText
		readSymbol, err := makeOutlineSymbol(parsed, ident.Name, owner+"."+ident.Name, "method", field, ident, &signature, doc, nil, nil, nil)
		if err != nil {
			return nil, err
		}
		children = append(children, readSymbol)
	}
	return children, nil
}

func outlineEmbeddedInterfaceType(parsed *goReadSource, owner string, field *ast.Field, includeUnexported bool) ([]backend.ReadSymbol, error) {
	typeValue, err := goReadNodeText(parsed.source, parsed.fset, field.Type)
	if err != nil {
		return nil, err
	}
	if !includeUnexported && !embeddedGoNameExported(field.Type) {
		return make([]backend.ReadSymbol, 0), nil
	}
	doc, err := goReadComment(parsed.source, parsed.fset, field.Doc)
	if err != nil {
		return nil, err
	}
	name := typeValue
	selection := ast.Node(field.Type)
	if base := goEmbeddedBaseIdent(field.Type); base != nil {
		name = base.Name
		selection = base
	}
	signature := typeValue
	readSymbol, err := makeOutlineSymbol(parsed, name, owner+"."+name, "type", field, selection, &signature, doc, &typeValue, nil, nil)
	if err != nil {
		return nil, err
	}
	return []backend.ReadSymbol{readSymbol}, nil
}

func outlineKindMatches(kinds map[string]bool, kind string) bool {
	return len(kinds) == 0 || kinds[kind]
}

func embeddedGoNameExported(expression ast.Expr) bool {
	base := goEmbeddedBaseIdent(expression)
	return base != nil && ast.IsExported(base.Name)
}

func goReceiverName(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) == 0 {
		return ""
	}
	expression := receiver.List[0].Type
	for {
		switch typed := expression.(type) {
		case *ast.StarExpr:
			expression = typed.X
		case *ast.IndexExpr:
			expression = typed.X
		case *ast.IndexListExpr:
			expression = typed.X
		case *ast.Ident:
			return typed.Name
		case *ast.SelectorExpr:
			return typed.Sel.Name
		default:
			return ""
		}
	}
}

func outlineFilteredTypeSignature(parsed *goReadSource, declaration *ast.GenDecl, spec *ast.TypeSpec, kinds map[string]bool, includeUnexported bool) (string, error) {
	filtered := *spec
	filtered.Type = filteredOutlineType(spec.Type, kinds, includeUnexported)
	copyDecl := *declaration
	copyDecl.Doc = nil
	copyDecl.Specs = []ast.Spec{&filtered}
	copyDecl.Lparen = token.NoPos
	return goReadFormattedNode(parsed.fset, &copyDecl)
}

func filteredOutlineType(node ast.Expr, kinds map[string]bool, includeUnexported bool) ast.Expr {
	switch value := node.(type) {
	case *ast.StructType:
		copyType := *value
		copyFields := *value.Fields
		copyFields.List = filteredStructFields(value.Fields.List, kinds, includeUnexported)
		copyType.Fields = &copyFields
		return &copyType
	case *ast.InterfaceType:
		copyType := *value
		copyMethods := *value.Methods
		copyMethods.List = filteredInterfaceMethods(value.Methods.List, kinds, includeUnexported)
		copyType.Methods = &copyMethods
		return &copyType
	default:
		return node
	}
}

func filteredStructFields(fields []*ast.Field, kinds map[string]bool, includeUnexported bool) []*ast.Field {
	filtered := make([]*ast.Field, 0, len(fields))
	if !outlineKindMatches(kinds, "field") {
		return filtered
	}
	for _, field := range fields {
		copyField := *field
		if len(field.Names) > 0 {
			copyField.Names = filteredOutlineNames(field.Names, includeUnexported)
			if len(copyField.Names) == 0 {
				continue
			}
		} else if !includeUnexported && !embeddedGoNameExported(field.Type) {
			continue
		}
		filtered = append(filtered, &copyField)
	}
	return filtered
}

func filteredInterfaceMethods(fields []*ast.Field, kinds map[string]bool, includeUnexported bool) []*ast.Field {
	filtered := make([]*ast.Field, 0, len(fields))
	for _, field := range fields {
		copyField := *field
		if len(field.Names) > 0 {
			if !outlineKindMatches(kinds, "method") {
				continue
			}
			copyField.Names = filteredOutlineNames(field.Names, includeUnexported)
			if len(copyField.Names) == 0 {
				continue
			}
		} else if !outlineKindMatches(kinds, "type") || !includeUnexported && !embeddedGoNameExported(field.Type) {
			continue
		}
		filtered = append(filtered, &copyField)
	}
	return filtered
}

func filteredOutlineNames(names []*ast.Ident, includeUnexported bool) []*ast.Ident {
	filtered := make([]*ast.Ident, 0, len(names))
	for _, name := range names {
		if includeUnexported || ast.IsExported(name.Name) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
