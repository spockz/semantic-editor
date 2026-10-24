// This file exposes one typed keyed-element append with a final-source selector continuation.

package operation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/symbol"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/tools/go/packages"
)

type compositeSelectorContinuation struct {
	File             string `json:"file"`
	Kind             string `json:"kind"`
	InDeclaration    string `json:"in_declaration"`
	Discriminator    string `json:"discriminator"`
	ExpectedRevision string `json:"expected_revision"`
}

type compositeSelectorCandidate struct {
	File          string
	Declaration   string
	ElementSource string
}

func runInsertElementConstruct(ctx context.Context, cc CallContext, req InsertConstructReq) (FileEditRes, error) {
	raw := insertElementConstructRaw(req)
	result, err := runStructuralPlan(ctx, cc, raw, insertElementPlanSpec())
	if err != nil {
		return FileEditRes{}, err
	}
	canonical, err := canonicalCompositeElement(req.Source)
	if err != nil {
		return FileEditRes{}, err
	}
	result.Edit.ReceiptDraft = result.Draft
	result.Edit.Resolve = result.Resolve
	result.Edit.ElementCandidate = &compositeSelectorCandidate{
		File: result.Edit.Display, Declaration: req.InDeclaration, ElementSource: canonical,
	}
	return result.Edit, nil
}

func runInsertElementConstructJava(ctx context.Context, cc CallContext, req InsertConstructReq) (FileEditRes, error) {
	project := effectiveProject(cc, req.Project)
	var expectedRevision *string
	if req.ExpectedRevision != "" {
		expectedRevision = &req.ExpectedRevision
	}
	root, file, before, after, err := runJavaInspectedSourceEdit(ctx, cc, project, req.File, req.InDeclaration, "element insertion", expectedRevision, true, func(declaration []byte) ([]byte, error) {
		return javacst.AppendJavaElement(declaration, req.Discriminator, []byte(req.Source))
	})
	if err != nil {
		return FileEditRes{}, err
	}
	display := displayWorkPath(root, req.File)
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("derive Java element receipt path for %q: %w", file, err)
	}
	if !filepath.IsLocal(relative) {
		return FileEditRes{}, fmt.Errorf("derive Java element receipt path for %q: path escapes workspace root", file)
	}
	relative = filepath.ToSlash(relative)
	return FileEditRes{
		File: file, Display: display, Symbol: req.InDeclaration,
		Diff:             astedit.UnifiedDiff(display, string(before), string(after)),
		Detail:           "Appended one element to the selected Java initializer.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
		ReceiptDraft: EditReceiptDraft{Operation: operationKeyInsertConstruct, Status: EditPersisted,
			ReadSet:        []FileRevision{{Path: relative, Revision: structuralRevision(before)}},
			PlannedChanges: []FileRevision{{Path: relative, Revision: structuralRevision(after)}}},
	}, nil
}

func insertElementPlanSpec() structuralPlanSpec {
	return structuralPlanSpec{
		Key: operationKeyInsertConstruct,
		Plan: func(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
			if placement := stringParam(raw, "placement"); placement != "" && placement != "append" {
				return astedit.SourcePlan{}, fmt.Errorf("placement must be append")
			}
			typeContext, err := structuralTypedCompositeContext(snapshot, stringParam(raw, wireInDeclaration), stringParam(raw, wireDiscriminator), stringParam(raw, "source"))
			if err != nil {
				return astedit.SourcePlan{}, err
			}
			return astedit.PlanInsertCompositeElement(snapshot.Sources[snapshot.File], astedit.InsertElementIntent{
				Declaration: stringParam(raw, wireInDeclaration), Discriminator: stringParam(raw, wireDiscriminator),
				ExpectedRevision: rawOptionalString(raw, wireExpectedRevision), Source: stringParam(raw, "source"), TypeContext: typeContext,
			})
		},
		Result: func(_ CallContext, _ map[string]any, _ astedit.SourcePlan, _ string) FileEditRes {
			return FileEditRes{Detail: "Appended one keyed composite element."}
		},
	}
}

func insertConstructOutputContract() *OperationOutputContract {
	output := fileEditOutputContract(map[string]ValueSchema{
		"file": {Kind: SchemaString}, "symbol": {Kind: SchemaString}, "in_type": {Kind: SchemaString},
		wireInFunction: {Kind: SchemaString}, wireDiscriminator: {Kind: SchemaString},
		"line": {Kind: SchemaInteger}, "diff": {Kind: SchemaString},
	}, []OutputContinuation{
		outputContinuation("file", RoleTarget, fileOutputConsumers),
		outputContinuation("symbol", RoleTarget, symbolOutputConsumers),
		outputContinuation("in_type", RoleSelector, []string{insertFieldToolName, extractTypeToolName, semanticInsertConstructToolName}),
		outputContinuation(wireInFunction, RoleSelector, functionOutputConsumers),
		outputContinuation(wireDiscriminator, RoleSelector, discriminatorOutputConsumers),
	})
	selectorItem := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file": {Kind: SchemaString}, "kind": {Kind: SchemaString, Enum: []any{string(backend.ConstructElement)}},
		wireInDeclaration: {Kind: SchemaString}, wireDiscriminator: {Kind: SchemaString}, wireExpectedRevision: {Kind: SchemaString},
	}, Required: []string{"file", "kind", wireInDeclaration, wireDiscriminator, wireExpectedRevision}}
	output.ResultSchema.Properties["selector_continuations"] = ValueSchema{Kind: SchemaArray, Items: &selectorItem}
	for _, field := range []string{"file", wireInDeclaration, wireDiscriminator, wireExpectedRevision} {
		output.Continuations = append(output.Continuations, OutputContinuation{
			Path: "result.selector_continuations[]." + field, Schema: selectorItem.Properties[field],
			Role: compositeContinuationRole(field), Consumers: []string{semanticInsertConstructToolName},
		})
	}
	return output
}

func compositeContinuationRole(field string) ParameterRole {
	if field == "file" || field == wireExpectedRevision {
		return RoleTarget
	}
	return RoleSelector
}

func finalizeInsertConstructOutput(cc CallContext, result FileEditRes) (FileEditRes, error) {
	result, err := finalizeFileEditOutput(cc, result)
	if err != nil || result.ElementCandidate == nil || result.SuppressContinuations {
		return result, err
	}
	if result.Receipt.Status != EditPersisted && result.Receipt.Status != EditPersistedWithDiagnostics {
		return result, nil
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	continuation, live, err := resolveCompositeSelectorContinuation(ctx, cc.WorkDir, *result.ElementCandidate, result.Receipt)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("resolve composite selector continuation: %w", err)
	}
	if live {
		result.SelectorContinuation = &continuation
	}
	return result, nil
}

func projectInsertConstructOutput(result FileEditRes) (any, error) {
	projected, err := projectFileEditOutput(result, "file", "symbol", "in_type", wireInFunction, wireDiscriminator, "line", "diff")
	if err != nil {
		return nil, err
	}
	object, ok := projected.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("project insert construct: typed result has type %T: %w", projected, ErrInvalidDefinition)
	}
	if result.SelectorContinuation != nil && !result.SuppressContinuations {
		object["selector_continuations"] = []compositeSelectorContinuation{*result.SelectorContinuation}
	}
	return object, nil
}

func structuralTypedCompositeContext(snapshot *structuralSnapshot, declaration, discriminator, source string) (astedit.CompositeTypeContext, error) {
	selected, err := selectCompositePackage(snapshot)
	if err != nil {
		return astedit.CompositeTypeContext{}, err
	}
	owner, err := structuralDeclarationOwner(selected.file, declaration)
	if err != nil {
		return astedit.CompositeTypeContext{}, err
	}
	literal, err := selectedCompositeLiteral(snapshot, selected.pkg, owner, discriminator)
	if err != nil {
		return astedit.CompositeTypeContext{}, err
	}
	keySource, err := astedit.CompositeElementKeySource(source)
	if err != nil {
		return astedit.CompositeTypeContext{}, err
	}
	typeAndValue, ok := selected.pkg.TypesInfo.Types[literal]
	if !ok || typeAndValue.Type == nil {
		return astedit.CompositeTypeContext{}, fmt.Errorf("selected literal has no typed object")
	}
	return typedCompositeContext(snapshot, selected.pkg, literal, typeAndValue.Type.Underlying(), keySource)
}

type compositePackageSelection struct {
	pkg  *packages.Package
	file *ast.File
}

func selectCompositePackage(snapshot *structuralSnapshot) (compositePackageSelection, error) {
	var selectedPackage *packages.Package
	var selectedFile *ast.File
	for _, pkg := range snapshot.Packages {
		if !eligibleCompositePackage(pkg) {
			continue
		}
		contains, err := packageContainsStructuralFile(pkg, snapshot.File)
		if err != nil {
			return compositePackageSelection{}, err
		}
		if !contains {
			continue
		}
		if len(pkg.Errors) != 0 || pkg.IllTyped {
			return compositePackageSelection{}, fmt.Errorf("selected package %s is not fully typechecked", pkg.PkgPath)
		}
		file, err := compositeFileInPackage(pkg, snapshot.File)
		if err != nil {
			return compositePackageSelection{}, err
		}
		if file != nil {
			if selectedFile != nil {
				return compositePackageSelection{}, fmt.Errorf("selected Go source has ambiguous package syntax")
			}
			selectedFile = file
			selectedPackage = pkg
		}
	}
	if selectedPackage == nil || selectedFile == nil {
		return compositePackageSelection{}, fmt.Errorf("selected Go file has no complete current-overlay type information")
	}
	return compositePackageSelection{pkg: selectedPackage, file: selectedFile}, nil
}

func eligibleCompositePackage(pkg *packages.Package) bool {
	return pkg.ID == pkg.PkgPath && pkg.Types != nil && pkg.Types.Path() == pkg.PkgPath && pkg.TypesInfo != nil && pkg.Fset != nil
}

func compositeFileInPackage(pkg *packages.Package, selectedPath string) (*ast.File, error) {
	var selected *ast.File
	for _, file := range pkg.Syntax {
		path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
		if err != nil {
			return nil, fmt.Errorf("resolve selected syntax file: %w", err)
		}
		if path != selectedPath {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("selected Go source has ambiguous package syntax")
		}
		selected = file
	}
	return selected, nil
}

func selectedCompositeLiteral(snapshot *structuralSnapshot, pkg *packages.Package, owner ast.Node, discriminator string) (*ast.CompositeLit, error) {
	var matches []*ast.CompositeLit
	ast.Inspect(owner, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		start, end := pkg.Fset.Position(literal.Pos()).Offset, pkg.Fset.Position(literal.End()).Offset
		if start >= 0 && end <= len(snapshot.Sources[snapshot.File]) && string(snapshot.Sources[snapshot.File][start:end]) == discriminator {
			matches = append(matches, literal)
		}
		return true
	})
	if len(matches) != 1 {
		return nil, fmt.Errorf("literal selector matched %d composites", len(matches))
	}
	return matches[0], nil
}

func typedCompositeContext(snapshot *structuralSnapshot, pkg *packages.Package, literal *ast.CompositeLit, underlying types.Type, keySource string) (astedit.CompositeTypeContext, error) {
	context := astedit.CompositeTypeContext{}
	switch typed := underlying.(type) {
	case *types.Struct:
		context.Family = astedit.LiteralStruct
		for field := range typed.Fields() {
			context.StructFields = append(context.StructFields, astedit.FieldSpec{Name: field.Name(), Type: types.TypeString(field.Type(), nil)})
		}
	case *types.Map:
		context.Family = astedit.LiteralMap
		return typedMapCompositeContext(snapshot, pkg, literal, typed, keySource, context)
	default:
		return astedit.CompositeTypeContext{}, fmt.Errorf("only typed keyed struct and map literals are supported")
	}
	return context, nil
}

func typedMapCompositeContext(snapshot *structuralSnapshot, pkg *packages.Package, literal *ast.CompositeLit, typed *types.Map, keySource string, context astedit.CompositeTypeContext) (astedit.CompositeTypeContext, error) {
	context.MapKeyType = types.TypeString(typed.Key(), nil)
	for _, element := range literal.Elts {
		keyed, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return astedit.CompositeTypeContext{}, fmt.Errorf("map literal contains a positional element")
		}
		evidence, ok := pkg.TypesInfo.Types[keyed.Key]
		if !ok || evidence.Value == nil {
			return astedit.CompositeTypeContext{}, fmt.Errorf("map literal has a key without constant identity")
		}
		start, end := pkg.Fset.Position(keyed.Key.Pos()).Offset, pkg.Fset.Position(keyed.Key.End()).Offset
		context.ExistingMapKeys = append(context.ExistingMapKeys, astedit.TypedKeyEvidence{Expression: string(snapshot.Sources[snapshot.File][start:end]), Identity: evidence.Value.String()})
	}
	newKey, err := types.Eval(pkg.Fset, pkg.Types, literal.Pos(), keySource)
	if err != nil || newKey.Type == nil || newKey.Value == nil || !types.AssignableTo(newKey.Type, typed.Key()) {
		return astedit.CompositeTypeContext{}, fmt.Errorf("new map key must be a constant assignable to %s", context.MapKeyType)
	}
	context.NewMapKey = astedit.TypedKeyEvidence{Expression: keySource, Identity: newKey.Value.String()}
	return context, nil
}

func structuralDeclarationOwner(file *ast.File, name string) (ast.Node, error) {
	var owner ast.Node
	for _, declaration := range file.Decls {
		candidate, err := structuralDeclarationInNode(declaration, name)
		if err != nil {
			return nil, err
		}
		if !candidate.Found {
			continue
		}
		if owner != nil {
			return nil, fmt.Errorf("declaration %q is ambiguous: %w", name, &symbol.SymbolError{Op: wireInDeclaration, Symbol: name, Err: symbol.ErrAmbiguous})
		}
		owner = candidate.Node
	}
	if owner == nil {
		return nil, fmt.Errorf("declaration %q not found: %w", name, &symbol.SymbolError{Op: wireInDeclaration, Symbol: name, Err: symbol.ErrNotFound})
	}
	return owner, nil
}

type structuralDeclarationMatch struct {
	Node  ast.Node
	Found bool
}

func structuralDeclarationInNode(declaration ast.Decl, name string) (structuralDeclarationMatch, error) {
	switch node := declaration.(type) {
	case *ast.FuncDecl:
		if structuralFunctionMatches(node, name) {
			return structuralDeclarationMatch{Node: node, Found: true}, nil
		}
	case *ast.GenDecl:
		return structuralSpecOwner(node.Specs, name)
	}
	return structuralDeclarationMatch{}, nil
}

func structuralFunctionMatches(function *ast.FuncDecl, name string) bool {
	if function.Recv == nil {
		return function.Name.Name == name
	}
	if len(function.Recv.List) != 1 {
		return false
	}
	receiver := function.Recv.List[0].Type
	if name == qualifiedGoTarget(receiverName(receiver), function.Name.Name) {
		return true
	}
	pointer, ok := receiver.(*ast.StarExpr)
	if !ok {
		return false
	}
	identifier, ok := pointer.X.(*ast.Ident)
	return ok && name == "(*"+identifier.Name+")."+function.Name.Name
}

func structuralSpecOwner(specs []ast.Spec, name string) (structuralDeclarationMatch, error) {
	var selected structuralDeclarationMatch
	for _, spec := range specs {
		match := structuralSpecMatches(spec, name)
		if match.SharedValueSpec {
			return structuralDeclarationMatch{}, fmt.Errorf("declaration %q shares a ValueSpec", name)
		}
		if match.Found {
			if selected.Found {
				return structuralDeclarationMatch{}, fmt.Errorf("declaration %q is ambiguous: %w", name, &symbol.SymbolError{Op: wireInDeclaration, Symbol: name, Err: symbol.ErrAmbiguous})
			}
			selected = structuralDeclarationMatch{Node: spec, Found: true}
		}
	}
	return selected, nil
}

type structuralSpecMatch struct {
	Found           bool
	SharedValueSpec bool
}

func structuralSpecMatches(spec ast.Spec, name string) structuralSpecMatch {
	switch item := spec.(type) {
	case *ast.TypeSpec:
		return structuralSpecMatch{Found: item.Name.Name == name}
	case *ast.ValueSpec:
		if slices.ContainsFunc(item.Names, func(identifier *ast.Ident) bool { return identifier.Name == name }) {
			return structuralSpecMatch{Found: len(item.Names) == 1, SharedValueSpec: len(item.Names) != 1}
		}
	}
	return structuralSpecMatch{}
}

func canonicalCompositeElement(source string) (string, error) {
	keySource, err := astedit.CompositeElementKeySource(source)
	if err != nil {
		return "", err
	}
	const prefix = "package p\nvar _ = map[any]any{"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "element.go", []byte(prefix+source+"}\n"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse inserted composite element: %w", err)
	}
	var selected *ast.KeyValueExpr
	if len(file.Decls) == 1 {
		if declaration, ok := file.Decls[0].(*ast.GenDecl); ok && len(declaration.Specs) == 1 {
			if value, ok := declaration.Specs[0].(*ast.ValueSpec); ok && len(value.Values) == 1 {
				if literal, ok := value.Values[0].(*ast.CompositeLit); ok && len(literal.Elts) == 1 {
					selected, _ = literal.Elts[0].(*ast.KeyValueExpr)
				}
			}
		}
	}
	if selected == nil || keySource == "" {
		return "", fmt.Errorf("inserted composite element has no key/value syntax")
	}
	var output bytes.Buffer
	if err := format.Node(&output, fset, selected); err != nil {
		return "", fmt.Errorf("format inserted composite element identity: %w", err)
	}
	return output.String(), nil
}

func resolveCompositeSelectorContinuation(ctx context.Context, root string, candidate compositeSelectorCandidate, receipt EditReceipt) (compositeSelectorContinuation, bool, error) {
	if err := ctx.Err(); err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("composite selector resolution canceled: %w", err)
	}
	if !filepath.IsLocal(filepath.FromSlash(candidate.File)) || candidate.Declaration == "" || candidate.ElementSource == "" {
		return compositeSelectorContinuation{}, false, nil
	}
	expectedRevision := expectedCompositeRevision(candidate.File, receipt.Changes)
	if expectedRevision == "" {
		return compositeSelectorContinuation{}, false, nil
	}
	location, exists, err := resolveCompositeSourceLocation(root, candidate.File)
	if err != nil || !exists {
		return compositeSelectorContinuation{}, false, err
	}
	source, current, err := readCompositeContinuationSource(location.File, expectedRevision)
	if err != nil || !current {
		return compositeSelectorContinuation{}, false, err
	}
	owner, ownerLive := structuralContinuationOwner(source.File, candidate.Declaration)
	if !ownerLive {
		return compositeSelectorContinuation{}, false, nil
	}
	selected, matched := findCompositeLiteral(owner, source, candidate.ElementSource)
	if !matched {
		return compositeSelectorContinuation{}, false, nil
	}
	return compositeSelectorContinuation{
		File: filepath.ToSlash(location.Relative), Kind: string(backend.ConstructElement), InDeclaration: candidate.Declaration,
		Discriminator: string(source.Content[selected.Start:selected.End]), ExpectedRevision: expectedRevision,
	}, true, nil
}

type compositeSourceLocation struct {
	File     string
	Relative string
}

type compositeSource struct {
	Content []byte
	Fset    *token.FileSet
	File    *ast.File
}

type compositeLiteralSpan struct {
	Start int
	End   int
}

func expectedCompositeRevision(path string, changes []FileRevision) string {
	for _, change := range changes {
		if change.Path == path {
			return change.Revision
		}
	}
	return ""
}

func resolveCompositeSourceLocation(root, file string) (compositeSourceLocation, bool, error) {
	abs, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return compositeSourceLocation{}, false, fmt.Errorf("resolve composite source path: %w", err)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return compositeSourceLocation{}, false, fmt.Errorf("resolve composite workspace root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return compositeSourceLocation{}, false, fmt.Errorf("canonicalize composite workspace root: %w", err)
	}
	canonicalFile, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return compositeSourceLocation{}, false, nil
		}
		return compositeSourceLocation{}, false, fmt.Errorf("canonicalize final composite source: %w", err)
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalFile)
	if err != nil {
		return compositeSourceLocation{}, false, fmt.Errorf("derive final composite source path: %w", err)
	}
	if !filepath.IsLocal(relative) {
		return compositeSourceLocation{}, false, nil
	}
	return compositeSourceLocation{File: canonicalFile, Relative: relative}, true, nil
}

func readCompositeContinuationSource(path, expectedRevision string) (compositeSource, bool, error) {
	content, err := os.ReadFile(path) // #nosec G304 -- the caller canonicalizes this path beneath the selected workspace root.
	if err != nil {
		return compositeSource{}, false, fmt.Errorf("read final composite source: %w", err)
	}
	if structuralRevision(content) != expectedRevision {
		return compositeSource{}, false, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return compositeSource{}, false, fmt.Errorf("parse final composite source: %w", err)
	}
	return compositeSource{Content: content, Fset: fset, File: file}, true, nil
}

func findCompositeLiteral(owner ast.Node, source compositeSource, element string) (compositeLiteralSpan, bool) {
	var matches []compositeLiteralSpan
	ast.Inspect(owner, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, child := range literal.Elts {
			if canonicalASTNode(source.Fset, child) == element {
				matches = append(matches, compositeLiteralSpan{
					Start: source.Fset.Position(literal.Pos()).Offset,
					End:   source.Fset.Position(literal.End()).Offset,
				})
				break
			}
		}
		return true
	})
	if len(matches) != 1 {
		return compositeLiteralSpan{}, false
	}
	selected := matches[0]
	if selected.Start < 0 || selected.End > len(source.Content) || selected.End <= selected.Start {
		return compositeLiteralSpan{}, false
	}
	return selected, true
}

func structuralContinuationOwner(file *ast.File, declaration string) (ast.Node, bool) {
	owner, err := structuralDeclarationOwner(file, declaration)
	return owner, err == nil
}

func canonicalASTNode(fset *token.FileSet, node ast.Node) string {
	var output bytes.Buffer
	if err := format.Node(&output, fset, node); err != nil {
		return ""
	}
	return output.String()
}

//go:fix inline
