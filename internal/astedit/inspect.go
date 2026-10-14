// Package astedit extracts exact declarations selected by the shared Go symbol resolver.
package astedit

import (
	"cmp"
	"context"
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/symbol"
	"go/ast"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
)

// InspectGo returns exact declaration snapshots for symbols selected by the resolver.
func InspectGo(root, file, query string) (*backend.InspectResult, error) {
	return InspectGoWithContext(context.Background(), root, file, query)
}

// InspectGoWithContext resolves exact declaration snapshots from the request's source view.
func InspectGoWithContext(ctx context.Context, root, file, query string) (*backend.InspectResult, error) {
	inspection, err := resolveGoInspectSnapshot(ctx, root, file, query)
	if err != nil {
		return nil, err
	}
	result := &backend.InspectResult{
		Scope: backend.ReadScope{
			Kind:     "workspace_syntax",
			Path:     ".",
			Complete: true,
		},
		Matches: make([]backend.InspectedSymbol, 0, len(inspection.candidates)),
	}
	if inspection.selectedFile != "" {
		result.Scope.Kind = "selected_file"
		result.Scope.Path = displayGoPath(inspection.root, inspection.selectedFile)
	}
	for _, candidate := range inspection.candidates {
		inspected, err := inspectGoCandidate(inspection, candidate)
		if err != nil {
			return nil, err
		}
		result.Matches = append(result.Matches, inspected)
	}
	return result, nil
}

type goInspectSnapshot struct {
	root         string
	selectedFile string
	files        map[string][]byte
	candidates   []*symbol.Symbol
}

func resolveGoInspectSnapshot(ctx context.Context, root, file, query string) (goInspectSnapshot, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return goInspectSnapshot{}, fmt.Errorf("resolve Go workspace %q: %w", root, err)
	}
	selectedFile := file
	if selectedFile != "" && !filepath.IsAbs(selectedFile) {
		selectedFile = filepath.Join(absoluteRoot, selectedFile)
	}
	snapshot, err := symbol.ResolveSnapshotWithContext(ctx, absoluteRoot, selectedFile, query)
	if err != nil {
		return goInspectSnapshot{}, err
	}
	candidates := inspectSymbolCandidates(snapshot.Result)
	if len(candidates) == 0 {
		return goInspectSnapshot{}, fmt.Errorf("symbol resolver returned no inspection candidates for %q", query)
	}
	slices.SortFunc(candidates, func(left, right *symbol.Symbol) int {
		if order := strings.Compare(left.File, right.File); order != 0 {
			return order
		}
		return cmp.Compare(left.Offset, right.Offset)
	})
	return goInspectSnapshot{root: absoluteRoot, selectedFile: selectedFile, files: snapshot.Files, candidates: candidates}, nil
}

func inspectSymbolCandidates(result *symbol.LookupResult) []*symbol.Symbol {
	candidates := make([]*symbol.Symbol, 0, 1)
	if result.IsAmbiguous() {
		return append(candidates, result.Candidates...)
	}
	if result.Definition != nil {
		candidates = append(candidates, result.Definition)
	}
	return candidates
}

func inspectGoCandidate(snapshot goInspectSnapshot, candidate *symbol.Symbol) (backend.InspectedSymbol, error) {
	candidatePath := candidate.File
	if !filepath.IsAbs(candidatePath) {
		candidatePath = filepath.Join(snapshot.root, candidatePath)
	}
	candidatePath, err := filepath.Abs(candidatePath)
	if err != nil {
		return backend.InspectedSymbol{}, fmt.Errorf("resolve inspection candidate path %q: %w", candidate.File, err)
	}
	source, ok := snapshot.files[filepath.Clean(candidatePath)]
	if !ok {
		return backend.InspectedSymbol{}, fmt.Errorf("inspection snapshot omitted resolved source %s", candidatePath)
	}
	parsed, err := parseGoReadSource(snapshot.root, candidatePath, source)
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	target, err := findGoInspectTarget(parsed, candidate)
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	return buildInspectedGoSymbol(parsed, source, target)
}

func buildInspectedGoSymbol(parsed *goReadSource, source []byte, target *goInspectTarget) (backend.InspectedSymbol, error) {
	start, end := target.start, target.end
	if !start.IsValid() {
		start = target.node.Pos()
	}
	if !end.IsValid() {
		end = target.node.End()
	}
	sourceText, err := goReadSpan(source, parsed.fset, start, end)
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	rng, err := goReadRange(source, parsed.fset, start, end)
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	selectionRange, err := goReadRange(source, parsed.fset, target.selection.Pos(), target.selection.End())
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	doc, err := goReadComment(source, parsed.fset, target.doc)
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	imports, err := goReadImports(parsed.file)
	if err != nil {
		return backend.InspectedSymbol{}, err
	}
	qualifiedName := target.qualified
	if qualifiedName == "" {
		qualifiedName = target.name
	}
	return backend.InspectedSymbol{
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
	}, nil
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

type goInspectQuery struct {
	parsed    *goReadSource
	candidate *symbol.Symbol
	qualified string
}

type goInspectTargetMatch struct {
	target *goInspectTarget
}

func findGoInspectTarget(parsed *goReadSource, candidate *symbol.Symbol) (*goInspectTarget, error) {
	qualified := candidate.QualifiedName
	if qualified == "" {
		qualified = candidate.BuildQualifiedName()
	}
	query := goInspectQuery{parsed: parsed, candidate: candidate, qualified: qualified}
	match, err := query.findTopLevelTarget()
	if err != nil || match.target != nil {
		return match.target, err
	}
	match, err = query.findFunctionTarget()
	if err != nil || match.target != nil {
		return match.target, err
	}
	return nil, fmt.Errorf("project resolved symbol %s at %s:%d but its snapshot declaration was not found", candidate.Name, candidate.File, candidate.Offset)
}

func (query goInspectQuery) matches(ident *ast.Ident) bool {
	return ident != nil && query.parsed.fset.PositionFor(ident.Pos(), false).Offset == query.candidate.Offset && ident.Name == query.candidate.Name
}

func (query goInspectQuery) findTopLevelTarget() (goInspectTargetMatch, error) {
	for _, declaration := range query.parsed.file.Decls {
		target, err := query.topLevelDeclarationTarget(declaration)
		if err != nil || target.target != nil {
			return target, err
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) topLevelDeclarationTarget(declaration ast.Decl) (goInspectTargetMatch, error) {
	switch decl := declaration.(type) {
	case *ast.FuncDecl:
		return query.functionDeclarationTarget(decl)
	case *ast.GenDecl:
		return query.generalDeclarationTarget(decl)
	default:
		return goInspectTargetMatch{}, nil
	}
}

func (query goInspectQuery) functionDeclarationTarget(decl *ast.FuncDecl) (goInspectTargetMatch, error) {
	if !query.matches(decl.Name) {
		return goInspectTargetMatch{}, nil
	}
	kind := "function"
	if decl.Recv != nil {
		kind = "method"
	}
	signatureDecl := *decl
	signatureDecl.Doc = nil
	signatureDecl.Body = nil
	signature, err := goReadFormattedNode(query.parsed.fset, &signatureDecl)
	if err != nil {
		return goInspectTargetMatch{}, err
	}
	return goInspectTargetMatch{target: &goInspectTarget{name: decl.Name.Name, qualified: query.qualified, kind: kind, node: decl, start: decl.Type.Func, end: decl.End(), selection: decl.Name, signature: &signature, doc: decl.Doc}}, nil
}

func (query goInspectQuery) generalDeclarationTarget(decl *ast.GenDecl) (goInspectTargetMatch, error) {
	for _, rawSpec := range decl.Specs {
		target, err := query.generalSpecTarget(decl, rawSpec)
		if err != nil || target.target != nil {
			return target, err
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) generalSpecTarget(decl *ast.GenDecl, rawSpec ast.Spec) (goInspectTargetMatch, error) {
	switch spec := rawSpec.(type) {
	case *ast.TypeSpec:
		return query.typeSpecTarget(decl, spec)
	case *ast.ValueSpec:
		return query.valueSpecTarget(decl, spec)
	default:
		return goInspectTargetMatch{}, nil
	}
}

func (query goInspectQuery) typeSpecTarget(decl *ast.GenDecl, spec *ast.TypeSpec) (goInspectTargetMatch, error) {
	if query.matches(spec.Name) {
		return query.namedTypeTarget(decl, spec)
	}
	if query.candidate.Kind == "field" {
		if structure, ok := spec.Type.(*ast.StructType); ok {
			for _, field := range structure.Fields.List {
				for _, ident := range field.Names {
					if query.matches(ident) {
						target, err := goInspectFieldTarget(query.parsed, query.qualified, field, ident)
						return goInspectTargetMatch{target: target}, err
					}
				}
			}
		}
	}
	if query.candidate.Kind == "method" {
		if iface, ok := spec.Type.(*ast.InterfaceType); ok {
			return query.interfaceMethodTarget(iface)
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) namedTypeTarget(decl *ast.GenDecl, spec *ast.TypeSpec) (goInspectTargetMatch, error) {
	singleDecl := *decl
	singleDecl.Doc = nil
	singleDecl.Specs = []ast.Spec{spec}
	singleDecl.Lparen = token.NoPos
	signature, err := goReadFormattedNode(query.parsed.fset, &singleDecl)
	if err != nil {
		return goInspectTargetMatch{}, err
	}
	doc := spec.Doc
	if doc == nil {
		doc = decl.Doc
	}
	kind := "type"
	if _, ok := spec.Type.(*ast.InterfaceType); ok {
		kind = "interface"
	}
	return goInspectTargetMatch{target: &goInspectTarget{name: spec.Name.Name, qualified: query.qualified, kind: kind, node: decl, start: decl.TokPos, end: decl.End(), selection: spec.Name, signature: &signature, doc: doc}}, nil
}

func (query goInspectQuery) interfaceMethodTarget(iface *ast.InterfaceType) (goInspectTargetMatch, error) {
	for _, field := range iface.Methods.List {
		for _, ident := range field.Names {
			if !query.matches(ident) {
				continue
			}
			signature, err := goReadSpan(query.parsed.source, query.parsed.fset, field.Names[0].Pos(), field.End())
			if err != nil {
				return goInspectTargetMatch{}, err
			}
			return goInspectTargetMatch{target: &goInspectTarget{name: ident.Name, qualified: query.qualified, kind: "method", node: field, start: field.Names[0].Pos(), end: field.End(), selection: ident, signature: &signature, doc: field.Doc}}, nil
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) valueSpecTarget(decl *ast.GenDecl, spec *ast.ValueSpec) (goInspectTargetMatch, error) {
	for _, ident := range spec.Names {
		if !query.matches(ident) {
			continue
		}
		valueKind := valueDeclarationKind(decl.Tok)
		kind, prefix := valueKind.kind, valueKind.prefix
		specText, err := goReadFormattedNode(query.parsed.fset, spec)
		if err != nil {
			return goInspectTargetMatch{}, err
		}
		signature := prefix + specText
		var typeValue *string
		if spec.Type != nil {
			text, err := goReadNodeText(query.parsed.source, query.parsed.fset, spec.Type)
			if err != nil {
				return goInspectTargetMatch{}, err
			}
			typeValue = &text
		}
		if kind == "variable" {
			signature = prefix + ident.Name
			if typeValue != nil {
				signature += " " + *typeValue
			}
		}
		doc := spec.Doc
		if doc == nil {
			doc = decl.Doc
		}
		return goInspectTargetMatch{target: &goInspectTarget{name: ident.Name, qualified: query.qualified, kind: kind, node: decl, start: decl.TokPos, end: decl.End(), selection: ident, signature: &signature, doc: doc, typeValue: typeValue}}, nil
	}
	return goInspectTargetMatch{}, nil
}

type goValueDeclarationKind struct {
	kind   string
	prefix string
}

func valueDeclarationKind(tokenKind token.Token) goValueDeclarationKind {
	if tokenKind == token.CONST {
		return goValueDeclarationKind{kind: "constant", prefix: "const "}
	}
	return goValueDeclarationKind{kind: "variable", prefix: "var "}
}

func (query goInspectQuery) findFunctionTarget() (goInspectTargetMatch, error) {
	for _, declaration := range query.parsed.file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		target, err := query.functionFieldTarget(fn)
		if err != nil || target.target != nil {
			return target, err
		}
		if fn.Body == nil {
			continue
		}
		target, err = query.functionBodyTarget(fn)
		if err != nil || target.target != nil {
			return target, err
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) functionFieldTarget(fn *ast.FuncDecl) (goInspectTargetMatch, error) {
	for _, fields := range []struct {
		list *ast.FieldList
		kind string
	}{{fn.Recv, "receiver"}, {fn.Type.Params, "parameter"}, {fn.Type.Results, "result"}} {
		if fields.list == nil {
			continue
		}
		for _, field := range fields.list.List {
			for _, ident := range field.Names {
				if !query.matches(ident) {
					continue
				}
				typeValue, err := goReadNodeText(query.parsed.source, query.parsed.fset, field.Type)
				if err != nil {
					return goInspectTargetMatch{}, err
				}
				start := field.Names[0].Pos()
				signature, err := goReadSpan(query.parsed.source, query.parsed.fset, start, field.End())
				if err != nil {
					return goInspectTargetMatch{}, err
				}
				return goInspectTargetMatch{target: &goInspectTarget{name: ident.Name, qualified: query.qualified, kind: fields.kind, node: field, start: start, end: field.End(), selection: ident, signature: &signature, doc: field.Doc, typeValue: &typeValue}}, nil
			}
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) functionBodyTarget(fn *ast.FuncDecl) (goInspectTargetMatch, error) {
	var target *goInspectTarget
	var targetErr error
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if target != nil || targetErr != nil {
			return false
		}
		match, err := query.localNodeTarget(node)
		target, targetErr = match.target, err
		return target == nil && targetErr == nil
	})
	return goInspectTargetMatch{target: target}, targetErr
}

func (query goInspectQuery) localNodeTarget(node ast.Node) (goInspectTargetMatch, error) {
	switch local := node.(type) {
	case *ast.GenDecl:
		return query.localValueTarget(local)
	case *ast.AssignStmt:
		return goInspectTargetMatch{target: query.assignmentTarget(local)}, nil
	case *ast.RangeStmt:
		return goInspectTargetMatch{target: query.rangeTarget(local)}, nil
	default:
		return goInspectTargetMatch{}, nil
	}
}

func (query goInspectQuery) localValueTarget(decl *ast.GenDecl) (goInspectTargetMatch, error) {
	for _, rawSpec := range decl.Specs {
		spec, ok := rawSpec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		match, err := query.valueSpecTarget(decl, spec)
		if err != nil || match.target != nil {
			return match, err
		}
	}
	return goInspectTargetMatch{}, nil
}

func (query goInspectQuery) assignmentTarget(assign *ast.AssignStmt) *goInspectTarget {
	for _, expression := range assign.Lhs {
		if ident, ok := expression.(*ast.Ident); ok && query.matches(ident) {
			return &goInspectTarget{name: ident.Name, qualified: query.qualified, kind: "variable", node: assign, start: assign.Pos(), end: assign.End(), selection: ident}
		}
	}
	return nil
}

func (query goInspectQuery) rangeTarget(statement *ast.RangeStmt) *goInspectTarget {
	for _, expression := range []ast.Expr{statement.Key, statement.Value} {
		if ident, ok := expression.(*ast.Ident); ok && query.matches(ident) {
			return &goInspectTarget{name: ident.Name, qualified: query.qualified, kind: "variable", node: statement, start: statement.Pos(), end: statement.End(), selection: ident}
		}
	}
	return nil
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
