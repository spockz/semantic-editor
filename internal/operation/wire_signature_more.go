// WHY: Implements bounded Go signature expansions on the shared typed edit and receipt contracts.

package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"strings"
)

const wireSourceIndex = "source_index"

const signatureParameters = "parameters"

func signatureExpansionDef(key, cliName, mcpName, summary string, params []ParameterContract, example map[string]any, handler func(context.Context, CallContext, map[string]any) (structuralPlanResult, error)) Def[map[string]any, structuralPlanResult] {
	return Def[map[string]any, structuralPlanResult]{
		Key: key, Summary: summary + automaticVerificationGuidance, Params: params, Level: LevelWorkspace, CLIName: cliName, MCPName: mcpName,
		Batchable: true, BufferedBatch: true, ExampleRaw: example, Output: func() *OperationOutputContract { output := signatureOutputContract(); return &output }(),
		Parse: func(raw map[string]any) (map[string]any, error) {
			if err := CheckParams(raw, params); err != nil {
				return nil, err
			}
			return maps.Clone(raw), nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, map[string]any) (structuralPlanResult, error){backend.LanguageGo: handler},
		Format:   func(result structuralPlanResult) (string, error) { return formatFileEdit(result.Edit) },
		FinalizeOutput: func(cc CallContext, _ map[string]any, result structuralPlanResult) (structuralPlanResult, error) {
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			result.Draft.Operation = key
			receipt, err := FinalizeEditReceipt(ctx, cc.WorkDir, cc.InBatch, result.Draft, result.Resolve)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize signature expansion receipt: %w", err)
			}
			result.Receipt = receipt
			return result, nil
		},
		ProjectOutput: func(result structuralPlanResult) (any, error) {
			out := map[string]any{"result": result.Edit.Detail, signatureEditReceiptField: result.Receipt}
			if len(result.Receipt.Continuations) > 0 {
				target := result.Receipt.Continuations[0]
				out["file"], out["symbol"] = target.File, target.Symbol
			}
			if result.Edit.Diff != "" {
				out["diff"] = result.Edit.Diff
			}
			return out, nil
		},
	}
}

func introduceParameterDef() Def[map[string]any, structuralPlanResult] {
	mapping := ParameterMappingSchema()
	minimum := 1
	mappings := ValueSchema{Kind: SchemaArray, Items: &mapping, MinItems: &minimum}
	params := []ParameterContract{
		structuralStringParam("file", wireTargetGoFileDescription, true, true, RoleTarget, "semantic_lookup.file", "semantic_change_signature.result.edit_receipt.continuations[].file", "semantic_introduce_parameter.result.edit_receipt.continuations[].file", "semantic_introduce_parameter_object.result.edit_receipt.continuations[].file"),
		structuralStringParam("symbol", "Target: private Go free function whose body and direct callers will be updated. Copy the qualified symbol from semantic_lookup or a live signature continuation.", true, true, RoleTarget, "semantic_lookup.symbol", "semantic_change_signature.result.edit_receipt.continuations[].symbol", "semantic_introduce_parameter.result.edit_receipt.continuations[].symbol", "semantic_introduce_parameter_object.result.edit_receipt.continuations[].symbol"),
		{Name: signatureParameters, JSONName: signatureParameters, CLIName: signatureParameters, Type: ParamArray, Required: true, Role: RoleValue, Description: "Replacement: complete ordered parameter mapping; preserve existing parameters by source_index and append one parameter with its closed constant argument.", Schema: &mappings},
		structuralStringParam(wireDiscriminator, "Selector: exact source text of one compile-time constant expression in the target function body; it must match the supplied caller constant in typed value and evaluation timing.", true, false, RoleSelector),
		structuralStringParam(wireExpectedRevision, "Precondition: opaque selected-file revision copied from a live continuation for this file and symbol; stale body selections are refused.", false, true, RoleTarget, "semantic_change_signature.result.edit_receipt.continuations[].expected_revision", "semantic_introduce_parameter.result.edit_receipt.continuations[].expected_revision", "semantic_introduce_parameter_object.result.edit_receipt.continuations[].expected_revision"),
	}
	return signatureExpansionDef("introduce_parameter", "introduce-parameter", "semantic_introduce_parameter", "Use this tool instead of manually replacing one compile-time constant expression in a private Go free function and updating its callers. It appends a parameter and updates every proven direct caller using the same closed constant value. Mutable reads, side effects, methods, exported or generic functions, function values, and incomplete call sets are refused. The edit receipt contains a live continuation for the surviving function.", params,
		map[string]any{"file": "service.go", "symbol": "process", signatureParameters: []any{map[string]any{wireSourceIndex: 0, "parameter": map[string]any{"name": "id", "type": "int"}}, map[string]any{"parameter": map[string]any{"name": "mode", "type": "int"}, "argument": "3"}}, wireDiscriminator: "3"}, runIntroduceParameter)
}

func introduceParameterObjectDef() Def[map[string]any, structuralPlanResult] {
	fieldMapping := ParameterToFieldMappingSchema()
	minimum := 1
	fields := ValueSchema{Kind: SchemaArray, Items: &fieldMapping, MinItems: &minimum}
	parameter := ParameterSpecSchema()
	params := []ParameterContract{
		structuralStringParam("file", wireTargetGoFileDescription, true, true, RoleTarget, "semantic_lookup.file", "semantic_change_signature.result.edit_receipt.continuations[].file", "semantic_introduce_parameter.result.edit_receipt.continuations[].file", "semantic_introduce_parameter_object.result.edit_receipt.continuations[].file"),
		structuralStringParam("symbol", "Target: private Go free function whose complete parameter list, body bindings, and direct callers will be updated. Copy the qualified symbol from semantic_lookup or a live signature continuation.", true, true, RoleTarget, "semantic_lookup.symbol", "semantic_change_signature.result.edit_receipt.continuations[].symbol", "semantic_introduce_parameter.result.edit_receipt.continuations[].symbol", "semantic_introduce_parameter_object.result.edit_receipt.continuations[].symbol"),
		structuralStringParam("destination_file", "Destination: existing same-file location for the new private struct; release one requires the target file.", true, true, RoleTarget, "semantic_lookup.file"),
		{Name: "parameter", JSONName: "parameter", CLIName: "parameter", Type: ParamObject, Required: true, Role: RoleValue, Description: "Creation choice: new private parameter name and private struct type name; choose both names for this conversion.", Schema: &parameter},
		{Name: "fields", JSONName: "fields", CLIName: "fields", Type: ParamArray, Required: true, Role: RoleValue, Description: "Replacement: map every original parameter exactly once, in source_index order, to a private field with the same Go type; caller expressions become keyed fields in original evaluation order.", Schema: &fields},
		structuralStringParam(wireExpectedRevision, "Precondition: opaque selected-file revision copied from a live continuation for this file and symbol; stale targets are refused.", false, true, RoleTarget, "semantic_change_signature.result.edit_receipt.continuations[].expected_revision", "semantic_introduce_parameter.result.edit_receipt.continuations[].expected_revision", "semantic_introduce_parameter_object.result.edit_receipt.continuations[].expected_revision"),
	}
	return signatureExpansionDef("introduce_parameter_object", "introduce-parameter-object", "semantic_introduce_parameter_object", "Use this tool instead of manually replacing all parameters of one private Go free function with a private same-file struct. It rewrites bound body references and converts every proven direct caller to a keyed literal that preserves original argument evaluation order. Partial mappings, methods, exported or generic functions, function values, and incomplete call sets are refused. The receipt contains live continuations for the function and new type.", params,
		map[string]any{"file": "write.go", "symbol": "write", "destination_file": "write.go", "parameter": map[string]any{"name": "options", "type": "writeOptions"}, "fields": []any{map[string]any{wireSourceIndex: 0, "field": map[string]any{"name": "path", "type": "string"}}}}, runIntroduceParameterObject)
}

func runIntroduceParameter(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file, symbol := stringParam(raw, "file"), stringParam(raw, "symbol")
	snapshot, err := prepareStructuralSnapshot(ctx, cc, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	before := snapshot.Sources[snapshot.File]
	if err := validateSignatureExpectedRevision(raw, before); err != nil {
		return structuralPlanResult{}, err
	}
	mappings, err := decodeSignatureMappings(raw[signatureParameters])
	if err != nil {
		return structuralPlanResult{}, err
	}
	added, err := appendedSignatureMapping(mappings)
	if err != nil {
		return structuralPlanResult{}, err
	}
	proof, err := collectSignatureCallSetWithName(snapshot, symbol, added.Name)
	if err != nil {
		return structuralPlanResult{}, err
	}
	expression, ok := raw[wireDiscriminator].(string)
	if !ok || strings.TrimSpace(expression) == "" {
		return structuralPlanResult{}, fmt.Errorf("discriminator must be a non-empty expression source")
	}
	evidence, err := proveIntroduceParameterConstant(snapshot, symbol, expression, added.Type, added.Argument)
	if err != nil {
		return structuralPlanResult{}, err
	}
	request := astedit.IntroduceParameterRequest{Signature: astedit.SignatureChangeRequest{File: snapshot.File, Symbol: symbol, Parameters: mappings}, Expression: expression, ParameterName: added.Name, Constant: evidence}
	if expected, ok := raw[wireExpectedRevision].(string); ok {
		request.Signature.ExpectedRevision = expected
	}
	plan, err := astedit.PlanIntroduceParameter(request, proof)
	if err != nil {
		return structuralPlanResult{}, err
	}
	return applySignatureExpansion(ctx, cc, snapshot, file, symbol, "Replaced the selected constant expression with a parameter and updated every proven direct caller.", plan)
}

func runIntroduceParameterObject(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file, symbol := stringParam(raw, "file"), stringParam(raw, "symbol")
	if stringParam(raw, "destination_file") != file {
		return structuralPlanResult{}, fmt.Errorf("destination_file must equal file for the same-file release")
	}
	snapshot, err := prepareStructuralSnapshot(ctx, cc, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	before := snapshot.Sources[snapshot.File]
	if err := validateSignatureExpectedRevision(raw, before); err != nil {
		return structuralPlanResult{}, err
	}
	parameter, err := decodeParameterSpec(raw["parameter"])
	if err != nil {
		return structuralPlanResult{}, err
	}
	fields, err := decodeParameterObjectFields(raw["fields"])
	if err != nil {
		return structuralPlanResult{}, err
	}
	proof, err := collectSignatureCallSetWithName(snapshot, symbol, parameter.Name)
	if err != nil {
		return structuralPlanResult{}, err
	}
	bindings, err := collectSignatureBodyBindings(snapshot, symbol)
	if err != nil {
		return structuralPlanResult{}, err
	}
	astProof := astedit.ParameterObjectProof{Complete: true, Files: proof.Files, Calls: proof.Calls, BodyBindings: bindings}
	request := astedit.ParameterObjectRequest{File: snapshot.File, Symbol: symbol, ParameterName: parameter.Name, TypeName: parameter.Type, Fields: fields}
	plan, err := astedit.PlanIntroduceParameterObject(request, astProof)
	if err != nil {
		return structuralPlanResult{}, err
	}
	typeRevision := ""
	for _, planned := range plan.Files {
		if planned.File == snapshot.File {
			typeRevision = structuralRevision(planned.After)
		}
	}
	if typeRevision == "" {
		return structuralPlanResult{}, fmt.Errorf("planned parameter object output omits target file")
	}
	result, err := applySignatureExpansion(ctx, cc, snapshot, file, symbol, "Replaced all function parameters with a private object and updated every proven direct caller.", plan)
	if err != nil {
		return structuralPlanResult{}, err
	}
	typeTarget := SymbolTarget{File: displayWorkPath(cc.WorkDir, file), Symbol: parameter.Type}
	typeTarget.ExpectedRevision = &typeRevision
	result.Draft.Continuations = append(result.Draft.Continuations, typeTarget)
	return result, nil
}

func validateSignatureExpectedRevision(raw map[string]any, before []byte) error {
	value, supplied := raw[wireExpectedRevision]
	if !supplied {
		return nil
	}
	revision, ok := value.(string)
	if !ok || revision == "" {
		return fmt.Errorf("expected_revision must be a non-empty string")
	}
	if revision != structuralRevision(before) {
		return fmt.Errorf("expected_revision does not match the selected full-file revision")
	}
	return nil
}

func appendedSignatureMapping(mappings []astedit.SignatureParameterMapping) (astedit.SignatureParameterMapping, error) {
	var added *astedit.SignatureParameterMapping
	for index := range mappings {
		if mappings[index].SourceIndex != nil {
			continue
		}
		if added != nil {
			return astedit.SignatureParameterMapping{}, fmt.Errorf("only one appended parameter is supported")
		}
		added = &mappings[index]
	}
	if added == nil || added.Argument == "" {
		return astedit.SignatureParameterMapping{}, fmt.Errorf("one appended parameter with an argument is required")
	}
	return *added, nil
}

func proveIntroduceParameterConstant(snapshot *structuralSnapshot, symbol, expression, parameterType, argument string) (astedit.SignatureConstantEvidence, error) {
	type expressionOccurrence struct {
		file  string
		start int
		end   int
	}
	selectedByOccurrence := make(map[expressionOccurrence]types.TypeAndValue)
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.Fset == nil || len(pkg.Errors) != 0 {
			continue
		}
		for _, file := range pkg.Syntax {
			position := pkg.Fset.Position(file.Pos())
			path, err := filepath.Abs(position.Filename)
			if err != nil {
				return astedit.SignatureConstantEvidence{}, fmt.Errorf("resolve selected expression file: %w", err)
			}
			if filepath.Clean(path) != snapshot.File {
				continue
			}
			source := snapshot.Sources[snapshot.File]
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Name.Name != symbol || function.Body == nil {
					continue
				}
				ast.Inspect(function.Body, func(node ast.Node) bool {
					expr, ok := node.(ast.Expr)
					if !ok {
						return true
					}
					start, end := pkg.Fset.Position(expr.Pos()).Offset, pkg.Fset.Position(expr.End()).Offset
					if start >= 0 && end > start && end <= len(source) && string(source[start:end]) == strings.TrimSpace(expression) {
						occurrence := expressionOccurrence{file: snapshot.File, start: start, end: end}
						evidence := pkg.TypesInfo.Types[expr]
						if previous, found := selectedByOccurrence[occurrence]; found {
							if previous.Value == nil || previous.Type == nil || evidence.Value == nil || evidence.Type == nil || previous.Value.Kind() != evidence.Value.Kind() || !constant.Compare(previous.Value, token.EQL, evidence.Value) || !types.Identical(previous.Type, evidence.Type) {
								selectedByOccurrence[occurrence] = types.TypeAndValue{}
							}
						} else {
							selectedByOccurrence[occurrence] = evidence
						}
					}
					return true
				})
			}
		}
	}
	if len(selectedByOccurrence) != 1 {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("selected body expression must resolve uniquely to a compile-time typed constant")
	}
	var selected types.TypeAndValue
	for _, evidence := range selectedByOccurrence {
		selected = evidence
	}
	if selected.Value == nil || selected.Type == nil {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("selected body expression must resolve uniquely to a compile-time typed constant with consistent evidence across package variants")
	}
	if err := validateClosedSignatureArgument(argument); err != nil {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("validate constant caller argument: %w", err)
	}
	var argumentValue constant.Value
	var argumentType types.Type
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Fset == nil {
			continue
		}
		value, err := types.Eval(pkg.Fset, pkg.Types, token.NoPos, argument)
		if err == nil && value.Value != nil {
			argumentValue, argumentType = value.Value, value.Type
			break
		}
	}
	if argumentValue == nil || argumentType == nil || selected.Value.Kind() != argumentValue.Kind() || !constant.Compare(selected.Value, token.EQL, argumentValue) {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("caller argument must be a compile-time constant with the selected expression's value")
	}
	var targetType types.Type
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Fset == nil {
			continue
		}
		value, err := types.Eval(pkg.Fset, pkg.Types, token.NoPos, parameterType)
		if err == nil && value.IsType() {
			targetType = value.Type
			break
		}
	}
	if targetType == nil || !types.AssignableTo(selected.Type, targetType) || !types.AssignableTo(argumentType, targetType) {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("selected and caller constants must both be assignable to the appended parameter type")
	}
	return astedit.SignatureConstantEvidence{Type: parameterType, SelectedValue: selected.Value.String(), ArgumentValue: argumentValue.String(), ArgumentSource: argument}, nil
}

func collectSignatureBodyBindings(snapshot *structuralSnapshot, symbol string) ([]astedit.SignatureBodyBinding, error) {
	var bindings []astedit.SignatureBodyBinding
	seen := make(map[astedit.SignatureBodyBinding]struct{})
	found := false
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.Fset == nil || len(pkg.Errors) != 0 {
			continue
		}
		for _, file := range pkg.Syntax {
			position := pkg.Fset.Position(file.Pos())
			path, err := filepath.Abs(position.Filename)
			if err != nil {
				return nil, fmt.Errorf("resolve parameter binding file: %w", err)
			}
			if filepath.Clean(path) != snapshot.File {
				continue
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Name.Name != symbol || function.Body == nil {
					continue
				}
				found = true
				var parameterObjects []types.Object
				for _, field := range function.Type.Params.List {
					for _, name := range field.Names {
						parameterObjects = append(parameterObjects, pkg.TypesInfo.Defs[name])
					}
				}
				objectIndex := make(map[types.Object]int, len(parameterObjects))
				for index, object := range parameterObjects {
					if object == nil {
						return nil, fmt.Errorf("parameter %d has no resolved Go object", index)
					}
					objectIndex[object] = index
				}
				bodyStart, bodyEnd := pkg.Fset.Position(function.Body.Pos()).Offset, pkg.Fset.Position(function.Body.End()).Offset
				ast.Inspect(function.Body, func(node ast.Node) bool {
					identifier, ok := node.(*ast.Ident)
					if !ok {
						return true
					}
					object := pkg.TypesInfo.Uses[identifier]
					index, parameter := objectIndex[object]
					if !parameter {
						return true
					}
					start, end := pkg.Fset.Position(identifier.Pos()).Offset, pkg.Fset.Position(identifier.End()).Offset
					binding := astedit.SignatureBodyBinding{File: snapshot.File, Start: start, End: end, SourceIndex: index}
					if _, ok := seen[binding]; !ok {
						seen[binding] = struct{}{}
						bindings = append(bindings, binding)
					}
					return true
				})
				if bodyStart < 0 || bodyEnd <= bodyStart {
					return nil, fmt.Errorf("function body positions are unavailable")
				}
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("private function %q was not found in the selected file", symbol)
	}
	return bindings, nil
}

func decodeParameterSpec(value any) (ParameterSpec, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ParameterSpec{}, fmt.Errorf("encode parameter object target: %w", err)
	}
	var parameter ParameterSpec
	if err := json.Unmarshal(encoded, &parameter); err != nil {
		return ParameterSpec{}, fmt.Errorf("decode parameter object target: %w", err)
	}
	if err := ValidateParameterSpec(parameter); err != nil {
		return ParameterSpec{}, fmt.Errorf("validate parameter object target: %w", err)
	}
	if parameter.Variadic != nil && *parameter.Variadic {
		return ParameterSpec{}, fmt.Errorf("parameter object argument cannot be variadic")
	}
	return parameter, nil
}

func decodeParameterObjectFields(value any) ([]astedit.ParameterObjectField, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode parameter-to-field mappings: %w", err)
	}
	var mappings []ParameterToFieldMapping
	if err := json.Unmarshal(encoded, &mappings); err != nil {
		return nil, fmt.Errorf("decode parameter-to-field mappings: %w", err)
	}
	if err := ValidateParameterToFieldMappings(mappings); err != nil {
		return nil, fmt.Errorf("validate parameter-to-field mappings: %w", err)
	}
	fields := make([]astedit.ParameterObjectField, 0, len(mappings))
	for _, mapping := range mappings {
		tag := ""
		if mapping.Field.Tag != nil {
			tag = *mapping.Field.Tag
		}
		fields = append(fields, astedit.ParameterObjectField{SourceIndex: mapping.SourceIndex, Name: mapping.Field.Name, Type: mapping.Field.Type, Tag: tag})
	}
	return fields, nil
}

func applySignatureExpansion(ctx context.Context, cc CallContext, snapshot *structuralSnapshot, file, symbol, detail string, plan astedit.SignaturePlan) (structuralPlanResult, error) {
	if ctx == nil {
		ctx = snapshot.Ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return structuralPlanResult{}, fmt.Errorf("apply signature expansion: %w", err)
	}
	if err := validateSignatureOverlay(snapshot, plan); err != nil {
		return structuralPlanResult{}, err
	}
	readSet, err := signatureReadSet(cc.WorkDir, snapshot)
	if err != nil {
		return structuralPlanResult{}, err
	}
	finishDelta := surroundingDelta(snapshot.Ctx, cc.WorkDir, cc.DeferVerification)
	changed, diffs, applyErr := applySignaturePlan(snapshot, plan)
	if applyErr != nil && !changed {
		return structuralPlanResult{}, applyErr
	}
	delta, diagnosticStatus := finishDelta()
	if !changed {
		detail = "The requested signature transformation is already present."
	}
	if applyErr != nil {
		detail = "Signature output publication did not complete: " + applyErr.Error()
	}
	edit := FileEditRes{File: snapshot.File, Display: displayWorkPath(cc.WorkDir, file), Symbol: symbol, Detail: detail, Diff: strings.Join(diffs, "\n"), Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}
	draft := EditReceiptDraft{Status: EditPersisted, ReadSet: readSet}
	for _, planned := range plan.Files {
		relative, err := filepath.Rel(cc.WorkDir, planned.File)
		if err != nil || !filepath.IsLocal(relative) {
			return structuralPlanResult{}, fmt.Errorf("planned signature file %q is outside workspace", planned.File)
		}
		draft.PlannedChanges = append(draft.PlannedChanges, FileRevision{Path: filepath.ToSlash(relative), Revision: structuralRevision(planned.After)})
	}
	if !changed {
		draft.Status = EditNoOp
	} else if diagnosticStatus == "changed" {
		draft.Status = EditPersistedWithDiagnostics
	}
	continuationRevision := structuralRevision(snapshot.Sources[snapshot.File])
	for _, planned := range plan.Files {
		if planned.File == snapshot.File {
			continuationRevision = structuralRevision(planned.After)
		}
	}
	draft.Continuations = []SymbolTarget{{File: edit.Display, Symbol: symbol, ExpectedRevision: &continuationRevision}}
	resolver := func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		return resolveSignatureContinuation(resolveCtx, cc.WorkDir, candidate)
	}
	return structuralPlanResult{Edit: edit, Draft: draft, Resolve: resolver}, nil
}

func registerSignatureExpansionOps(registry *Registry) error {
	if err := Register(registry, introduceParameterDef()); err != nil {
		return fmt.Errorf("register introduce-parameter: %w", err)
	}
	if err := Register(registry, introduceParameterObjectDef()); err != nil {
		return fmt.Errorf("register introduce-parameter-object: %w", err)
	}
	return nil
}

func validateClosedSignatureArgument(source string) error {
	node, err := parser.ParseExpr(source)
	if err != nil {
		return fmt.Errorf("parse expression: %w", err)
	}
	switch value := node.(type) {
	case *ast.BasicLit:
		return nil
	case *ast.Ident:
		if value.Name == "true" || value.Name == "false" || value.Name == "nil" {
			return nil
		}
	}
	return fmt.Errorf("only a literal, nil, true, or false is accepted")
}
