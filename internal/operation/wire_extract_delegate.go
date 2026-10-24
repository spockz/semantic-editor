// This file implements the bounded Go field-and-method delegate extraction.
// WHY: Method relocation needs a separate wrapper contract from field-only extraction.

package operation

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"golang.org/x/tools/go/packages"
)

const extractDelegateToolName = "refactor_extract_delegate"

func extractDelegateStructuralDef() Def[map[string]any, structuralPlanResult] {
	member := MemberTargetSchema()
	minimum := 1
	params := []ParameterContract{
		structuralStringParam("file", "Target: existing Go source file containing the private owner, fields, and method; copy from a live source continuation.", true, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].file"),
		structuralStringParam("symbol", "Target: fresh private simple name for the generated delegate type.", true, false, RoleTarget),
		structuralStringParam("in_type", "Selector: existing private, non-generic struct declared in file; copy from the matching type inspection continuation.", true, true, RoleSelector,
			"semantic_inspect_symbol.result.continuations[].in_type"),
		{Name: "members", JSONName: "members", CLIName: "members", Type: ParamArray, Description: "Selector: non-empty contiguous run of exact {file, symbol} field identities listed in declaration order; each selects one direct private field on in_type in file.", Required: true, Role: RoleSelector,
			Schema: &ValueSchema{Kind: SchemaArray, Items: &member, MinItems: &minimum}},
		structuralStringParam("field", "Replacement: fresh private simple composition field name on the original owner.", true, false, RoleValue),
		structuralStringParam("method", "Selector: exact direct method name on in_type to move into the delegate.", true, true, RoleSelector),
		structuralStringParam("expected_revision", "Precondition: optional opaque full-file revision from the same live inspection continuation as file and in_type; stale values are refused.", false, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].expected_revision"),
	}
	output := structuralOutputContract(false)
	properties := SymbolTargetSchema().Properties
	output.Continuations = append(output.Continuations,
		OutputContinuation{Path: editReceiptContinuationFilePath, Schema: properties["file"], Role: RoleTarget, Consumers: []string{semanticInspectSymbolToolName}},
		OutputContinuation{Path: editReceiptContinuationSymbolPath, Schema: properties["symbol"], Role: RoleTarget, Consumers: []string{semanticInspectSymbolToolName}},
	)
	return structuralPlanDef(structuralPlanSpec{
		Key: "refactor_extract_delegate", CLIName: "refactor-extract-delegate", MCPName: extractDelegateToolName,
		Summary: "Use this tool instead of manually moving one direct pointer-receiver method and a contiguous declaration-ordered run of direct private fields from a private Go struct into a same-file delegate. The original method remains as a wrapper, preserving its signature and method sets. The wrapper passes a nil delegate for a nil owner so the moved method retains its original lazy field-access behavior. Receiver uses must be direct selectors of selected fields. Recover, receiver escape, unsupported signatures, generated or inactive sources, external uses, keyed or address-taken uses, tags, comments, embedding, generic or exported owners, and reflection-sensitive packages are refused. The complete overlay is typechecked before one buffered edit.",
		Params:  params, Output: &output,
		ExampleRaw: map[string]any{"file": "counter.go", "symbol": "counterState", "in_type": "counter", "members": []any{map[string]any{"file": "counter.go", "symbol": "counter.value"}}, "field": "state", "method": "Add"},
		Plan:       planExtractDelegate,
		Result: func(_ CallContext, raw map[string]any, _ astedit.SourcePlan, _ string) FileEditRes {
			return FileEditRes{Symbol: stringParam(raw, "symbol"), InType: stringParam(raw, "in_type"), InFunction: stringParam(raw, "method"), Detail: "Extracted selected fields and method into a composed Go delegate type."}
		},
		Continuations:   extractTypeContinuations,
		ResolveWithRoot: resolveExtractTypeTarget,
	})
}

func planExtractDelegate(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
	inType, newType := stringParam(raw, "in_type"), stringParam(raw, "symbol")
	fieldName, methodName := stringParam(raw, "field"), stringParam(raw, "method")
	selection, err := prepareExtractTypeSelection(snapshot, raw, inType, newType, fieldName, false)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	method, syntax, err := resolveExtractDelegateMethod(selection.Owner, selection.TypeName, methodName, snapshot.File)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	if err := validateExtractDelegateMethod(snapshot, selection, method, syntax); err != nil {
		return astedit.SourcePlan{}, err
	}
	return buildExtractDelegateSourcePlan(snapshot, selection, method, syntax)
}

func resolveExtractDelegateMethod(owner *packages.Package, typeName, methodName, selectedFile string) (*types.Func, *ast.FuncDecl, error) {
	obj, index, _ := types.LookupFieldOrMethod(types.NewPointer(owner.Types.Scope().Lookup(typeName).Type()), true, owner.Types, methodName)
	method, ok := obj.(*types.Func)
	if !ok || len(index) != 1 {
		return nil, nil, fmt.Errorf("method %q must be declared directly on %q", methodName, typeName)
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil, nil, fmt.Errorf("method %q has no direct receiver", methodName)
	}
	receiverType, ok := types.Unalias(signature.Recv().Type()).(*types.Pointer)
	if !ok || receiverType.Elem() != owner.Types.Scope().Lookup(typeName).Type() {
		return nil, nil, fmt.Errorf("method %q must have a direct pointer receiver on %q", methodName, typeName)
	}
	for _, syntax := range owner.Syntax {
		path, err := filepath.Abs(owner.Fset.Position(syntax.Pos()).Filename)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve method source path: %w", err)
		}
		if path != selectedFile {
			continue
		}
		for _, declaration := range syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && owner.TypesInfo.Defs[function.Name] == method {
				return method, function, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("method %q must be declared in the selected file", methodName)
}

func validateExtractDelegateMethod(snapshot *structuralSnapshot, selection extractTypeSelection, method *types.Func, syntax *ast.FuncDecl) error {
	if err := validateExtractDelegateSignature(method, syntax); err != nil {
		return err
	}
	for _, file := range selection.Owner.Syntax {
		path, err := filepath.Abs(selection.Owner.Fset.Position(file.Pos()).Filename)
		if err != nil {
			return fmt.Errorf("resolve method comment source path: %w", err)
		}
		if path != snapshot.File {
			continue
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if comment.Pos() >= syntax.Pos() && comment.End() <= syntax.End() {
					return fmt.Errorf("method %q contains comments that cannot be relocated safely", method.Name())
				}
			}
		}
	}
	if functionScope := selection.Owner.TypesInfo.Scopes[syntax.Type]; functionScope != nil {
		if _, object := functionScope.LookupParent("nil", syntax.Body.Pos()); object != nil && object.Parent() != types.Universe {
			return fmt.Errorf("method %q shadows nil required by the generated wrapper", method.Name())
		}
	}
	if err := validateExtractDelegateReceiverUses(selection, method, syntax); err != nil {
		return err
	}
	if extractDelegateUsesRecover(selection, syntax) {
		return fmt.Errorf("cannot extract method %q because moving builtin recover changes deferred-panic behavior", method.Name())
	}
	if err := rejectGeneratedExtractFile(snapshot, selection.Owner); err != nil {
		return err
	}
	return rejectStructuralBuildVariants(snapshot, selection.Owner)
}

func validateExtractDelegateSignature(method *types.Func, syntax *ast.FuncDecl) error {
	signature := method.Type().(*types.Signature)
	if syntax.Doc != nil || syntax.Body == nil || signature.TypeParams() != nil && signature.TypeParams().Len() != 0 || signature.Variadic() {
		return fmt.Errorf("method %q must be a direct, non-generic, non-variadic method without a declaration comment", method.Name())
	}
	if syntax.Recv == nil || len(syntax.Recv.List) != 1 || len(syntax.Recv.List[0].Names) != 1 || syntax.Recv.List[0].Names[0].Name == "_" {
		return fmt.Errorf("method %q requires one named, non-blank receiver", method.Name())
	}
	for _, field := range syntax.Type.Params.List {
		if len(field.Names) != 1 || field.Names[0].Name == "_" {
			return fmt.Errorf("method %q requires each parameter to have one non-blank name", method.Name())
		}
	}
	if syntax.Type.Results != nil {
		for _, field := range syntax.Type.Results.List {
			if len(field.Names) != 0 {
				return fmt.Errorf("method %q requires unnamed results", method.Name())
			}
		}
	}
	return nil
}

func validateExtractDelegateReceiverUses(selection extractTypeSelection, method *types.Func, syntax *ast.FuncDecl) error {
	selected := make(map[*types.Var]bool, len(selection.Selected))
	for _, field := range selection.Selected {
		selected[field.object] = true
	}
	receiver := selection.Owner.TypesInfo.Defs[syntax.Recv.List[0].Names[0]]
	if receiver == nil {
		return fmt.Errorf("method %q receiver has no compiler identity", method.Name())
	}
	allowed := make(map[*ast.Ident]bool)
	ast.Inspect(syntax.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || selection.Owner.TypesInfo.Uses[ident] != receiver {
			return true
		}
		resolved := selection.Owner.TypesInfo.Selections[selector]
		if resolved == nil {
			return true
		}
		field, ok := resolved.Obj().(*types.Var)
		if ok && selected[field] && len(resolved.Index()) == 1 {
			allowed[ident] = true
		}
		return true
	})
	var invalid bool
	ast.Inspect(syntax.Body, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && selection.Owner.TypesInfo.Uses[ident] == receiver && !allowed[ident] {
			invalid = true
			return false
		}
		return !invalid
	})
	if invalid {
		return fmt.Errorf("cannot extract method %q: receiver uses must be direct selectors of selected fields", method.Name())
	}
	return nil
}

func extractDelegateUsesRecover(selection extractTypeSelection, syntax *ast.FuncDecl) bool {
	var found bool
	ast.Inspect(syntax.Body, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		builtin, ok := selection.Owner.TypesInfo.Uses[ident].(*types.Builtin)
		found = found || ok && builtin.Name() == "recover"
		return !found
	})
	return found
}

func buildExtractDelegateSourcePlan(snapshot *structuralSnapshot, selection extractTypeSelection, method *types.Func, syntax *ast.FuncDecl) (astedit.SourcePlan, error) {
	source := snapshot.Sources[snapshot.File]
	fset, file, err := parsePlanExtractSource(snapshot.File, source)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	ownerGroup, err := findExtractTypeOwner(file, selection.TypeName)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	if err = rejectExtractTypeTrailingComment(fset, file, ownerGroup, selection.TypeName); err != nil {
		return astedit.SourcePlan{}, err
	}
	if err = bindExtractTypeFieldDeclarations(selection.Selected, ownerGroup, selection.TypeName); err != nil {
		return astedit.SourcePlan{}, err
	}
	fieldEdits, fieldLines, newline, err := extractTypeFieldEdits(source, fset, selection)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	methodStart := selection.Owner.Fset.Position(syntax.Pos()).Offset
	methodEnd := selection.Owner.Fset.Position(syntax.End()).Offset
	methodText := append([]byte(nil), source[methodStart:methodEnd]...)
	moved, err := movedExtractDelegateMethod(methodText, methodStart, selection, method, syntax)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	wrapper, err := extractDelegateWrapper(source, syntax, selection, method)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	useEdits, err := extractDelegateReferenceEdits(snapshot, selection, syntax)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	fieldEdits = append(fieldEdits, useEdits...)
	fieldEdits = append(fieldEdits, sourceEdit{start: methodStart, end: methodEnd, text: wrapper})
	fields := renderExtractTypeFields(fieldLines, newline)
	insertAt := fset.Position(ownerGroup.End()).Offset
	typeText := []byte(newline + newline + "type " + selection.NewType + " struct {" + newline + fields + "}" + newline + newline + string(moved))
	fieldEdits = append(fieldEdits, sourceEdit{start: insertAt, end: insertAt, text: typeText})
	updated, err := applyExtractTypeEdits(source, fieldEdits)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	return astedit.SourcePlan{Source: updated, Selection: astedit.PlanSelection{Kind: "type", Owner: selection.TypeName, Target: selection.NewType}, RequiredProofs: []astedit.ProofRequirement{astedit.ProofCompleteSourceSet, astedit.ProofTypedMemberSet, astedit.ProofUnkeyedLiteralSet}, Changed: !bytes.Equal(source, updated)}, nil
}

func movedExtractDelegateMethod(source []byte, methodStart int, selection extractTypeSelection, method *types.Func, syntax *ast.FuncDecl) ([]byte, error) {
	_ = method
	receiverType := syntax.Recv.List[0].Type
	start := selection.Owner.Fset.Position(receiverType.Pos()).Offset - methodStart
	end := selection.Owner.Fset.Position(receiverType.End()).Offset - methodStart
	if start < 0 || end > len(source) || start >= end {
		return nil, fmt.Errorf("method receiver type range is invalid")
	}
	return applyExtractTypeEdits(source, []sourceEdit{{start: start, end: end, text: []byte("*" + selection.NewType)}})
}

func extractDelegateWrapper(source []byte, syntax *ast.FuncDecl, selection extractTypeSelection, method *types.Func) ([]byte, error) {
	bodyStart := selection.Owner.Fset.Position(syntax.Body.Pos()).Offset
	methodStart := selection.Owner.Fset.Position(syntax.Pos()).Offset
	if methodStart < 0 || bodyStart > len(source) || methodStart >= bodyStart {
		return nil, fmt.Errorf("method signature source range is invalid")
	}
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	receiver := syntax.Recv.List[0].Names[0].Name
	local := "delegate"
	used := map[string]bool{receiver: true, "nil": true}
	for _, field := range syntax.Type.Params.List {
		used[field.Names[0].Name] = true
	}
	ast.Inspect(syntax.Body, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok {
			used[ident.Name] = true
		}
		return true
	})
	for suffix := 1; used[local]; suffix++ {
		local = fmt.Sprintf("delegate%d", suffix)
	}
	var body strings.Builder
	body.WriteString("{" + newline + "\tvar " + local + " *" + selection.NewType + newline + "\tif " + receiver + " != nil {" + newline + "\t\t" + local + " = &" + receiver + "." + selection.FieldName + newline + "\t}" + newline + "\t")
	if syntax.Type.Results != nil && len(syntax.Type.Results.List) > 0 {
		body.WriteString("return ")
	}
	body.WriteString(local + "." + method.Name() + "(" + extractDelegateMethodArguments(syntax) + ")")
	body.WriteString(newline + "}")
	return append(append([]byte(nil), source[methodStart:bodyStart]...), []byte(body.String())...), nil
}

func extractDelegateReferenceEdits(snapshot *structuralSnapshot, selection extractTypeSelection, moved *ast.FuncDecl) ([]sourceEdit, error) {
	methodStart := selection.Owner.Fset.Position(moved.Pos()).Offset
	methodEnd := selection.Owner.Fset.Position(moved.End()).Offset
	edits := make([]sourceEdit, 0)
	seen := make(map[int]bool)
	for _, syntax := range selection.Owner.Syntax {
		path, err := filepath.Abs(selection.Owner.Fset.Position(syntax.Pos()).Filename)
		if err != nil {
			return nil, fmt.Errorf("resolve delegate reference source path: %w", err)
		}
		if path != snapshot.File {
			continue
		}
		ast.Inspect(syntax, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			position := selection.Owner.Fset.Position(selector.Pos()).Offset
			if position >= methodStart && position < methodEnd {
				return true
			}
			resolved := selection.Owner.TypesInfo.Selections[selector]
			if resolved == nil {
				return true
			}
			field, ok := resolved.Obj().(*types.Var)
			if !ok || !isSelectedExtractField(selection.Selected, field) {
				return true
			}
			offset := selection.Owner.Fset.Position(selector.X.End()).Offset
			if !seen[offset] {
				edits = append(edits, sourceEdit{start: offset, end: offset, text: []byte("." + selection.FieldName)})
				seen[offset] = true
			}
			return true
		})
	}
	return edits, nil
}

func extractDelegateMethodArguments(syntax *ast.FuncDecl) string {
	args := make([]string, 0, len(syntax.Type.Params.List))
	for _, field := range syntax.Type.Params.List {
		args = append(args, field.Names[0].Name)
	}
	return strings.Join(args, ", ")
}
