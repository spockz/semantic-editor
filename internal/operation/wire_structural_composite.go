// This file exposes one typed keyed-element append with a final-source selector continuation.

package operation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"slices"

	"golang.org/x/tools/go/packages"
)

const insertCompositeElementToolName = "semantic_insert_composite_element"

type compositeSelectorContinuation struct {
	File             string `json:"file"`
	InDeclaration    string `json:"in_declaration"`
	Discriminator    string `json:"discriminator"`
	ExpectedRevision string `json:"expected_revision"`
}

type compositeSelectorCandidate struct {
	File          string
	Declaration   string
	ElementSource string
}

type compositePlanResult struct {
	Structural           structuralPlanResult
	Candidate            compositeSelectorCandidate
	SelectorContinuation *compositeSelectorContinuation
}

func insertCompositeElementStructuralDef() Def[map[string]any, compositePlanResult] {
	placement := structuralStringParam("placement", "Placement: append exactly one keyed element to the selected literal.", false, false, RolePolicy)
	placement.Schema.Enum = []any{"append"}
	placement.Default = "append"
	params := []ParameterContract{
		structuralStringParam("file", wireTargetGoFileDescription, true, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].file",
			insertCompositeElementToolName+".result.selector_continuations[].file"),
		structuralStringParam(wireInDeclaration, "Selector: exact top-level declaration that owns the selected literal; copy from an inspection record or this tool's finalized selector record.", true, true, RoleSelector,
			"semantic_inspect_symbol.result.continuations[].in_declaration",
			insertCompositeElementToolName+".result.selector_continuations[].in_declaration"),
		structuralStringParam(wireDiscriminator, "Selector: exact current source text of one unique typed composite literal; after an edit, copy the refreshed discriminator from this tool's finalized selector record.", true, true, RoleSelector,
			insertCompositeElementToolName+".result.selector_continuations[].discriminator"),
		structuralStringParam("source", "Value: one caller-authored keyed element in key: value form, without braces, commas, or comments.", true, false, RoleValue),
		placement,
		structuralStringParam("expected_revision", "Precondition: optional opaque full-file revision copied verbatim from the same inspection or finalized selector record as file and in_declaration.", false, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].expected_revision",
			insertCompositeElementToolName+".result.selector_continuations[].expected_revision"),
	}
	output := structuralOutputContract(false)
	selectorItem := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file":               {Kind: SchemaString},
		wireInDeclaration:    {Kind: SchemaString},
		wireDiscriminator:    {Kind: SchemaString},
		wireExpectedRevision: {Kind: SchemaString},
	}, Required: []string{"file", wireInDeclaration, wireDiscriminator, "expected_revision"}}
	output.ResultSchema.Properties["selector_continuations"] = ValueSchema{Kind: SchemaArray, Items: &selectorItem}
	for _, field := range []string{"file", wireInDeclaration, wireDiscriminator, "expected_revision"} {
		output.Continuations = append(output.Continuations, OutputContinuation{
			Path: "result.selector_continuations[]." + field, Schema: selectorItem.Properties[field],
			Role: compositeContinuationRole(field), Consumers: []string{insertCompositeElementToolName},
		})
	}
	spec := structuralPlanSpec{
		Key: "insert_composite_element", CLIName: "insert-composite-element", MCPName: insertCompositeElementToolName,
		Summary: "Use this tool instead of replacing a declaration or literal to append one keyed element to a unique typed Go struct or map literal owned by `in_declaration`. Select the exact literal with its full current source in `discriminator`; supply one key: value element in `source`. Repeated literals, positional elements, duplicate or unknown struct fields, duplicate or non-constant map keys, malformed elements, comments, and non-struct/map literal types are refused. A successful standalone result includes a refreshed selector record only when the final source still proves one live literal; batch results do not include selector continuations. Appending is structural and can change runtime behavior, including evaluation of the inserted value.",
		Params:  params, ExampleRaw: map[string]any{"file": "routes.go", wireInDeclaration: "routes", wireDiscriminator: `map[string]Handler{"health": health}`, "source": `"metrics": metrics`},
		Plan: func(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
			if placement := stringParam(raw, "placement"); placement != "" && placement != "append" {
				return astedit.SourcePlan{}, fmt.Errorf("placement must be append")
			}
			context, err := structuralTypedCompositeContext(snapshot, stringParam(raw, wireInDeclaration), stringParam(raw, wireDiscriminator), stringParam(raw, "source"))
			if err != nil {
				return astedit.SourcePlan{}, err
			}
			return astedit.PlanInsertCompositeElement(snapshot.Sources[snapshot.File], astedit.InsertElementIntent{
				Declaration: stringParam(raw, wireInDeclaration), Discriminator: stringParam(raw, wireDiscriminator),
				ExpectedRevision: rawOptionalString(raw, "expected_revision"), Source: stringParam(raw, "source"), TypeContext: context,
			})
		},
		Result: func(_ CallContext, _ map[string]any, _ astedit.SourcePlan, _ string) FileEditRes {
			return FileEditRes{Detail: "Appended one keyed composite element."}
		},
	}
	return compositeElementDef(spec, output)
}

func compositeContinuationRole(field string) ParameterRole {
	if field == "file" || field == "expected_revision" {
		return RoleTarget
	}
	return RoleSelector
}

func compositeElementDef(spec structuralPlanSpec, output OperationOutputContract) Def[map[string]any, compositePlanResult] {
	return Def[map[string]any, compositePlanResult]{
		Key: spec.Key, Summary: spec.Summary, Params: spec.Params, Level: LevelFile,
		CLIName: spec.CLIName, MCPName: spec.MCPName, Batchable: true, BufferedBatch: true,
		ExampleRaw: spec.ExampleRaw, Output: &output,
		Parse: func(raw map[string]any) (map[string]any, error) { return maps.Clone(raw), nil },
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, map[string]any) (compositePlanResult, error){
			backend.LanguageGo: func(ctx context.Context, cc CallContext, raw map[string]any) (compositePlanResult, error) {
				result, err := runStructuralPlan(ctx, cc, raw, spec)
				if err != nil {
					return compositePlanResult{}, err
				}
				canonical, err := canonicalCompositeElement(stringParam(raw, "source"))
				if err != nil {
					return compositePlanResult{}, err
				}
				return compositePlanResult{Structural: result, Candidate: compositeSelectorCandidate{
					File: result.Edit.Display, Declaration: stringParam(raw, wireInDeclaration), ElementSource: canonical,
				}}, nil
			},
		},
		Format: func(result compositePlanResult) (string, error) { return formatFileEdit(result.Structural.Edit) },
		FinalizeOutput: func(cc CallContext, _ map[string]any, result compositePlanResult) (compositePlanResult, error) {
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			receipt, err := FinalizeEditReceipt(ctx, cc.WorkDir, cc.InBatch, result.Structural.Draft, result.Structural.Resolve)
			if err != nil {
				return compositePlanResult{}, fmt.Errorf("finalize composite element edit receipt: %w", err)
			}
			result.Structural.Receipt = receipt
			if receipt.Status != EditPersisted && receipt.Status != EditPersistedWithDiagnostics {
				return result, nil
			}
			continuation, live, err := resolveCompositeSelectorContinuation(ctx, cc.WorkDir, result.Candidate, receipt)
			if err != nil {
				return compositePlanResult{}, fmt.Errorf("resolve composite selector continuation: %w", err)
			}
			if live {
				result.SelectorContinuation = &continuation
			}
			return result, nil
		},
		ProjectOutput: func(result compositePlanResult) (any, error) {
			projected := map[string]any{"result": result.Structural.Edit.Detail, "edit_receipt": result.Structural.Receipt}
			if result.Structural.Receipt.Status != EditAppliedWorking {
				if result.Structural.Edit.Display != "" {
					projected["file"] = result.Structural.Edit.Display
				}
			}
			if result.Structural.Edit.Diff != "" {
				projected["diff"] = result.Structural.Edit.Diff
			}
			if result.SelectorContinuation != nil {
				projected["selector_continuations"] = []compositeSelectorContinuation{*result.SelectorContinuation}
			}
			return projected, nil
		},
	}
}

func structuralTypedCompositeContext(snapshot *structuralSnapshot, declaration, discriminator, source string) (astedit.CompositeTypeContext, error) {
	var selectedPackage *packages.Package
	var selectedFile *ast.File
	for _, pkg := range snapshot.Packages {
		if pkg.ID != pkg.PkgPath || pkg.Types == nil || pkg.Types.Path() != pkg.PkgPath || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		contains, err := packageContainsStructuralFile(pkg, snapshot.File)
		if err != nil {
			return astedit.CompositeTypeContext{}, err
		}
		if !contains {
			continue
		}
		if len(pkg.Errors) != 0 || pkg.IllTyped {
			return astedit.CompositeTypeContext{}, fmt.Errorf("selected package %s is not fully typechecked", pkg.PkgPath)
		}
		for _, file := range pkg.Syntax {
			path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
			if err != nil {
				return astedit.CompositeTypeContext{}, fmt.Errorf("resolve selected syntax file: %w", err)
			}
			if path == snapshot.File {
				if selectedFile != nil {
					return astedit.CompositeTypeContext{}, fmt.Errorf("selected Go source has ambiguous package syntax")
				}
				selectedFile = file
			}
		}
		selectedPackage = pkg
	}
	if selectedPackage == nil || selectedFile == nil {
		return astedit.CompositeTypeContext{}, fmt.Errorf("selected Go file has no complete current-overlay type information")
	}
	owner, err := structuralDeclarationOwner(selectedFile, declaration)
	if err != nil {
		return astedit.CompositeTypeContext{}, err
	}
	keySource, err := astedit.CompositeElementKeySource(source)
	if err != nil {
		return astedit.CompositeTypeContext{}, err
	}
	var matches []*ast.CompositeLit
	ast.Inspect(owner, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		start, end := selectedPackage.Fset.Position(literal.Pos()).Offset, selectedPackage.Fset.Position(literal.End()).Offset
		if start >= 0 && end <= len(snapshot.Sources[snapshot.File]) && string(snapshot.Sources[snapshot.File][start:end]) == discriminator {
			matches = append(matches, literal)
		}
		return true
	})
	if len(matches) != 1 {
		return astedit.CompositeTypeContext{}, fmt.Errorf("literal selector matched %d composites", len(matches))
	}
	literal := matches[0]
	typeAndValue, ok := selectedPackage.TypesInfo.Types[literal]
	if !ok || typeAndValue.Type == nil {
		return astedit.CompositeTypeContext{}, fmt.Errorf("selected literal has no typed object")
	}
	context := astedit.CompositeTypeContext{}
	switch underlying := typeAndValue.Type.Underlying().(type) {
	case *types.Struct:
		context.Family = astedit.LiteralStruct
		for field := range underlying.Fields() {
			context.StructFields = append(context.StructFields, astedit.FieldSpec{Name: field.Name(), Type: types.TypeString(field.Type(), nil)})
		}
	case *types.Map:
		context.Family = astedit.LiteralMap
		context.MapKeyType = types.TypeString(underlying.Key(), nil)
		for _, element := range literal.Elts {
			keyed, ok := element.(*ast.KeyValueExpr)
			if !ok {
				return astedit.CompositeTypeContext{}, fmt.Errorf("map literal contains a positional element")
			}
			evidence, ok := selectedPackage.TypesInfo.Types[keyed.Key]
			if !ok || evidence.Value == nil {
				return astedit.CompositeTypeContext{}, fmt.Errorf("map literal has a key without constant identity")
			}
			start, end := selectedPackage.Fset.Position(keyed.Key.Pos()).Offset, selectedPackage.Fset.Position(keyed.Key.End()).Offset
			context.ExistingMapKeys = append(context.ExistingMapKeys, astedit.TypedKeyEvidence{Expression: string(snapshot.Sources[snapshot.File][start:end]), Identity: evidence.Value.String()})
		}
		newKey, err := types.Eval(selectedPackage.Fset, selectedPackage.Types, literal.Pos(), keySource)
		if err != nil || newKey.Type == nil || newKey.Value == nil || !types.AssignableTo(newKey.Type, underlying.Key()) {
			return astedit.CompositeTypeContext{}, fmt.Errorf("new map key must be a constant assignable to %s", context.MapKeyType)
		}
		context.NewMapKey = astedit.TypedKeyEvidence{Expression: keySource, Identity: newKey.Value.String()}
	default:
		return astedit.CompositeTypeContext{}, fmt.Errorf("only typed keyed struct and map literals are supported")
	}
	return context, nil
}

func structuralDeclarationOwner(file *ast.File, name string) (ast.Node, error) {
	var owner ast.Node
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			matches := value.Name.Name == name && value.Recv == nil
			if value.Recv != nil && len(value.Recv.List) == 1 {
				receiver := receiverName(value.Recv.List[0].Type)
				matches = name == qualifiedGoTarget(receiver, value.Name.Name)
				if pointer, ok := value.Recv.List[0].Type.(*ast.StarExpr); ok {
					if identifier, ok := pointer.X.(*ast.Ident); ok {
						matches = matches || name == "(*"+identifier.Name+")."+value.Name.Name
					}
				}
			}
			if matches {
				if owner != nil {
					return nil, fmt.Errorf("declaration %q is ambiguous", name)
				}
				owner = value
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				matched := false
				switch item := spec.(type) {
				case *ast.TypeSpec:
					matched = item.Name.Name == name
				case *ast.ValueSpec:
					if len(item.Names) != 1 && slices.ContainsFunc(item.Names, func(identifier *ast.Ident) bool { return identifier.Name == name }) {
						return nil, fmt.Errorf("declaration %q shares a ValueSpec", name)
					}
					matched = len(item.Names) == 1 && item.Names[0].Name == name
				}
				if matched {
					if owner != nil {
						return nil, fmt.Errorf("declaration %q is ambiguous", name)
					}
					owner = spec
				}
			}
		}
	}
	if owner == nil {
		return nil, fmt.Errorf("declaration %q not found", name)
	}
	return owner, nil
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
	var expectedRevision string
	for _, change := range receipt.Changes {
		if change.Path == candidate.File {
			expectedRevision = change.Revision
			break
		}
	}
	if expectedRevision == "" {
		return compositeSelectorContinuation{}, false, nil
	}
	absolute, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(candidate.File)))
	if err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("resolve composite source path: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("resolve composite workspace root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("canonicalize composite workspace root: %w", err)
	}
	canonicalFile, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return compositeSelectorContinuation{}, false, nil
		}
		return compositeSelectorContinuation{}, false, fmt.Errorf("canonicalize final composite source: %w", err)
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalFile)
	if err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("derive final composite source path: %w", err)
	}
	if !filepath.IsLocal(relative) {
		return compositeSelectorContinuation{}, false, nil
	}
	content, err := os.ReadFile(canonicalFile)
	if err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("read final composite source: %w", err)
	}
	if structuralRevision(content) != expectedRevision {
		return compositeSelectorContinuation{}, false, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, canonicalFile, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return compositeSelectorContinuation{}, false, fmt.Errorf("parse final composite source: %w", err)
	}
	owner, ownerLive := structuralContinuationOwner(file, candidate.Declaration)
	if !ownerLive {
		return compositeSelectorContinuation{}, false, nil
	}
	type literalMatch struct {
		literal *ast.CompositeLit
		start   int
		end     int
	}
	var matches []literalMatch
	ast.Inspect(owner, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range literal.Elts {
			if canonicalASTNode(fset, element) == candidate.ElementSource {
				matches = append(matches, literalMatch{literal: literal, start: fset.Position(literal.Pos()).Offset, end: fset.Position(literal.End()).Offset})
				break
			}
		}
		return true
	})
	if len(matches) != 1 {
		return compositeSelectorContinuation{}, false, nil
	}
	selected := matches[0]
	if selected.start < 0 || selected.end > len(content) || selected.end <= selected.start {
		return compositeSelectorContinuation{}, false, nil
	}
	return compositeSelectorContinuation{
		File: filepath.ToSlash(relative), InDeclaration: candidate.Declaration,
		Discriminator: string(content[selected.start:selected.end]), ExpectedRevision: expectedRevision,
	}, true, nil
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
