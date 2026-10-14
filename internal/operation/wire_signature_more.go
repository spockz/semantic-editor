// WHY: Implements bounded Go signature expansions on the shared typed edit and receipt contracts.

package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
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
			receipt, err := finalizeCallReceipt(ctx, cc, cc.WorkDir, result.Draft, result.Resolve)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize signature expansion receipt: %w", err)
			}
			result.Receipt = receipt
			functionTarget, err := finalizedSignatureFunctionTarget(ctx, cc, result)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize signature function target: %w", err)
			}
			if functionTarget.Proven {
				result.FunctionTarget = &functionTarget.Target
			}
			return result, nil
		},
		ProjectOutput: func(result structuralPlanResult) (any, error) { return projectSignatureExpansion(result), nil },
	}
}

func introduceParameterDef() Def[map[string]any, structuralPlanResult] {
	mapping := ParameterMappingSchema()
	minimum := 1
	mappings := ValueSchema{Kind: SchemaArray, Items: &mapping, MinItems: &minimum}
	params := make([]ParameterContract, 0, 11)
	params = append(params, []ParameterContract{
		structuralStringParam("file", "Target source file: Go file or Java source in a trusted JDT LS workspace.", true, true, RoleTarget, append([]string{"semantic_lookup.file"}, signatureTargetSourceFields("file")...)...),
		structuralStringParam("symbol", "Target: private Go free function or signature-qualified Java method whose body expression and resolved callers will be updated.", true, true, RoleTarget, append([]string{"semantic_lookup.symbol"}, signatureTargetSourceFields("symbol")...)...),
		{Name: signatureParameters, JSONName: signatureParameters, CLIName: signatureParameters, Type: ParamArray, Required: true, Role: RoleValue, Description: "Replacement: complete ordered parameter mapping; preserve existing parameters by source_index and append one parameter with its closed argument expression.", Schema: &mappings},
		structuralStringParam(wireDiscriminator, "Selector: exact source text of the expression to replace with the new parameter inside the target function or method.", true, false, RoleSelector),
		structuralStringParam(wireConstructPath, "Selector: inspected Java construct path when the expression discriminator repeats; stale expected_revision values are refused.", false, false, RoleSelector),
		structuralStringParam(wireExpectedRevision, "Precondition: opaque selected-file revision copied from the same finalized function_target record as file and symbol; stale body selections are refused.", false, true, RoleTarget, signatureTargetSourceFields(wireExpectedRevision)...),
	}...)
	params = append(params, readProjectParams[:5]...)
	definition := signatureExpansionDef("introduce_parameter", "introduce-parameter", "semantic_introduce_parameter", "Use this tool instead of manually replacing one exact expression with a new parameter and updating its callers. Java delegates signature and call-site changes to JDT LS; Go requires a private free function and proves the closed constant argument and complete call set. Methods, exported or generic Go functions, function values, and ambiguous Java expressions are refused. Standalone results expose a finalized function_target; receipt continuations remain available for inspection.", params,
		map[string]any{"file": "service.go", "symbol": "process", signatureParameters: []any{map[string]any{wireSourceIndex: 0, "parameter": map[string]any{"name": "id", "type": "int"}}, map[string]any{"parameter": map[string]any{"name": "mode", "type": "int"}, "argument": "3"}}, wireDiscriminator: "3"}, runIntroduceParameter)
	definition.Handlers[backend.LanguageJava] = runIntroduceParameterJava
	return definition
}

func introduceParameterObjectDef() Def[map[string]any, structuralPlanResult] {
	fieldMapping := ParameterToFieldMappingSchema()
	minimum := 1
	fields := ValueSchema{Kind: SchemaArray, Items: &fieldMapping, MinItems: &minimum}
	parameter := ParameterSpecSchema()
	params := make([]ParameterContract, 0, 11)
	params = append(params, []ParameterContract{
		structuralStringParam("file", "Target source file: Go source or Java source in a trusted JDT LS workspace.", true, true, RoleTarget, append([]string{"semantic_lookup.file"}, signatureTargetSourceFields("file")...)...),
		structuralStringParam("symbol", "Target: private Go free function or signature-qualified Java method whose complete parameter list, body bindings, and resolved direct callers will be updated.", true, true, RoleTarget, append([]string{"semantic_lookup.symbol"}, signatureTargetSourceFields("symbol")...)...),
		structuralStringParam("destination_file", "Destination: existing same-file location for the new private struct or package-private Java parameter object.", true, true, RoleTarget),
		{Name: "parameter", JSONName: "parameter", CLIName: "parameter", Type: ParamObject, Required: true, Role: RoleValue, Description: "Creation choice: new parameter name and private Go struct or package-private Java type name.", Schema: &parameter},
		{Name: "fields", JSONName: "fields", CLIName: "fields", Type: ParamArray, Required: true, Role: RoleValue, Description: "Replacement: map every original parameter exactly once in source_index order to a field with its original type; callers retain argument order and evaluation.", Schema: &fields},
		structuralStringParam(wireExpectedRevision, "Precondition: opaque selected-file revision copied from the same finalized function_target record as file and symbol; stale targets are refused.", false, true, RoleTarget, signatureTargetSourceFields(wireExpectedRevision)...),
	}...)
	params = append(params, readProjectParams[:5]...)
	definition := signatureExpansionDef("introduce_parameter_object", "introduce-parameter-object", "semantic_introduce_parameter_object", "Use this tool instead of manually replacing a complete parameter list with one value object, rewriting bound body references, and updating every proven direct caller. Go requires a private free function; Java requires one same-package method with direct invocation references. Partial mappings, unresolved call sites, cross-package Java callers, and unsupported Java parameter shapes are refused.", params,
		map[string]any{"file": "write.go", "symbol": "write", "destination_file": "write.go", "parameter": map[string]any{"name": "options", "type": "writeOptions"}, "fields": []any{map[string]any{wireSourceIndex: 0, "field": map[string]any{"name": "path", "type": "string"}}}}, runIntroduceParameterObject)
	definition.Handlers[backend.LanguageJava] = runIntroduceParameterObjectJava
	return definition
}

func runIntroduceParameter(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file, symbol := stringParam(raw, "file"), stringParam(raw, "symbol")
	snapshot, err := prepareStructuralSnapshot(ctx, cc, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	before := snapshot.Sources[snapshot.File]
	if revisionErr := validateSignatureExpectedRevision(raw, before); revisionErr != nil {
		return structuralPlanResult{}, revisionErr
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
	if revisionErr := validateSignatureExpectedRevision(raw, before); revisionErr != nil {
		return structuralPlanResult{}, revisionErr
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

type signatureExpressionOccurrence struct {
	file  string
	start int
	end   int
}

type typedSignatureExpression struct {
	typeAndValue types.TypeAndValue
	ownerPackage *types.Package
	ownerPos     token.Pos
	fset         *token.FileSet
}

func proveIntroduceParameterConstant(snapshot *structuralSnapshot, symbol, expression, parameterType, argument string) (astedit.SignatureConstantEvidence, error) {
	selected, err := selectedSignatureConstant(snapshot, symbol, expression)
	if err != nil {
		return astedit.SignatureConstantEvidence{}, err
	}
	selectedOwner := selected.ownerPackage
	selectedFset := selected.fset
	selectedPosition := selected.ownerPos
	selectedValue := selected.typeAndValue
	if err := validateClosedSignatureArgument(argument); err != nil {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("validate constant caller argument: %w", err)
	}
	argumentResult, argumentErr := types.Eval(selectedFset, selectedOwner, selectedPosition, argument)
	if argumentErr != nil {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("evaluate caller constant in selected function scope: %w", argumentErr)
	}
	argumentValue, argumentType := argumentResult.Value, argumentResult.Type
	if argumentValue == nil || argumentType == nil || selectedValue.Value.Kind() != argumentValue.Kind() || !constant.Compare(selectedValue.Value, token.EQL, argumentValue) {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("caller argument must be a compile-time constant with the selected expression's value")
	}
	targetResult, targetErr := types.Eval(selectedFset, selectedOwner, selectedPosition, parameterType)
	if targetErr != nil {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("evaluate parameter type in selected function scope: %w", targetErr)
	}
	var targetType types.Type
	if targetResult.IsType() {
		targetType = targetResult.Type
	}
	if targetType == nil || !types.AssignableTo(selectedValue.Type, targetType) || !types.AssignableTo(argumentType, targetType) {
		return astedit.SignatureConstantEvidence{}, fmt.Errorf("selected and caller constants must both be assignable to the appended parameter type")
	}
	return astedit.SignatureConstantEvidence{Type: parameterType, SelectedValue: selectedValue.Value.String(), ArgumentValue: argumentValue.String(), ArgumentSource: argument}, nil
}

func selectedSignatureConstant(snapshot *structuralSnapshot, symbol, expression string) (typedSignatureExpression, error) {
	selected := make(map[signatureExpressionOccurrence]typedSignatureExpression)
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.Fset == nil || len(pkg.Errors) != 0 {
			continue
		}
		for _, file := range pkg.Syntax {
			if err := collectSignatureExpressionEvidence(snapshot, pkg, file, symbol, expression, selected); err != nil {
				return typedSignatureExpression{}, err
			}
		}
	}
	if len(selected) != 1 {
		return typedSignatureExpression{}, fmt.Errorf("selected body expression must resolve uniquely to a compile-time typed constant")
	}
	for _, evidence := range selected {
		if !validTypedSignatureExpression(evidence) {
			return typedSignatureExpression{}, fmt.Errorf("selected body expression must resolve uniquely to a compile-time typed constant with consistent evidence across package variants")
		}
		return evidence, nil
	}
	return typedSignatureExpression{}, fmt.Errorf("selected body expression evidence is unavailable")
}

func collectSignatureExpressionEvidence(snapshot *structuralSnapshot, pkg *packages.Package, file *ast.File, symbol, expression string, selected map[signatureExpressionOccurrence]typedSignatureExpression) error {
	path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
	if err != nil {
		return fmt.Errorf("resolve selected expression file: %w", err)
	}
	path = filepath.Clean(path)
	if path != snapshot.File {
		return nil
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
			if start < 0 || end <= start || end > len(source) || string(source[start:end]) != strings.TrimSpace(expression) {
				return true
			}
			key := signatureExpressionOccurrence{file: path, start: start, end: end}
			evidence := typedSignatureExpression{typeAndValue: pkg.TypesInfo.Types[expr], ownerPackage: pkg.Types, ownerPos: function.Type.Pos(), fset: pkg.Fset}
			if previous, found := selected[key]; found && !equivalentTypedSignatureExpression(previous, evidence) {
				selected[key] = typedSignatureExpression{}
			} else if !found {
				selected[key] = evidence
			}
			return true
		})
	}
	return nil
}

func equivalentTypedSignatureExpression(left, right typedSignatureExpression) bool {
	return left.typeAndValue.Value != nil && left.typeAndValue.Type != nil && right.typeAndValue.Value != nil && right.typeAndValue.Type != nil && left.ownerPackage != nil && right.ownerPackage != nil && left.ownerPackage.Path() == right.ownerPackage.Path() && left.typeAndValue.Value.Kind() == right.typeAndValue.Value.Kind() && constant.Compare(left.typeAndValue.Value, token.EQL, right.typeAndValue.Value) && equivalentSignatureVariantType(left.typeAndValue.Type, left.fset, right.typeAndValue.Type, right.fset)
}

func validTypedSignatureExpression(evidence typedSignatureExpression) bool {
	return evidence.typeAndValue.Value != nil && evidence.typeAndValue.Type != nil && evidence.ownerPackage != nil && evidence.ownerPos.IsValid() && evidence.fset != nil
}

func equivalentSignatureVariantType(left types.Type, leftFset *token.FileSet, right types.Type, rightFset *token.FileSet) bool {
	if left == nil || right == nil {
		return false
	}
	if types.Identical(left, right) {
		return true
	}
	leftNamed, leftOK := types.Unalias(left).(*types.Named)
	rightNamed, rightOK := types.Unalias(right).(*types.Named)
	if !leftOK || !rightOK || !sameSignatureVariantTypeName(leftNamed.Obj(), leftFset, rightNamed.Obj(), rightFset) || !types.Identical(leftNamed.Underlying(), rightNamed.Underlying()) {
		return false
	}
	leftArgs, rightArgs := leftNamed.TypeArgs(), rightNamed.TypeArgs()
	if leftArgs.Len() != rightArgs.Len() {
		return false
	}
	for index := range leftArgs.Len() {
		if !types.Identical(leftArgs.At(index), rightArgs.At(index)) {
			return false
		}
	}
	return true
}

func sameSignatureVariantTypeName(left *types.TypeName, leftFset *token.FileSet, right *types.TypeName, rightFset *token.FileSet) bool {
	if left == nil || right == nil || left.Name() != right.Name() || left.Pkg() == nil || right.Pkg() == nil || left.Pkg().Path() != right.Pkg().Path() || leftFset == nil || rightFset == nil {
		return false
	}
	leftFile, rightFile := leftFset.File(left.Pos()), rightFset.File(right.Pos())
	if leftFile == nil || rightFile == nil {
		return false
	}
	leftPath, err := filepath.Abs(leftFile.Name())
	if err != nil {
		return false
	}
	rightPath, err := filepath.Abs(rightFile.Name())
	if err != nil {
		return false
	}
	return filepath.Clean(leftPath) == filepath.Clean(rightPath) && leftFile.Offset(left.Pos()) == rightFile.Offset(right.Pos())
}

func collectSignatureBodyBindings(snapshot *structuralSnapshot, symbol string) ([]astedit.SignatureBodyBinding, error) {
	var bindings []astedit.SignatureBodyBinding
	seen := make(map[astedit.SignatureBodyBinding]struct{})
	found := false
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.Fset == nil || len(pkg.Errors) != 0 {
			continue
		}
		matched, err := collectSignaturePackageBindings(snapshot, pkg, symbol, seen, &bindings)
		if err != nil {
			return nil, err
		}
		found = found || matched
	}
	if !found {
		return nil, fmt.Errorf("private function %q was not found in the selected file", symbol)
	}
	return bindings, nil
}

func collectSignaturePackageBindings(snapshot *structuralSnapshot, pkg *packages.Package, symbol string, seen map[astedit.SignatureBodyBinding]struct{}, bindings *[]astedit.SignatureBodyBinding) (bool, error) {
	found := false
	for _, file := range pkg.Syntax {
		path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
		if err != nil {
			return false, fmt.Errorf("resolve parameter binding file: %w", err)
		}
		if filepath.Clean(path) != snapshot.File {
			continue
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Name.Name != symbol || function.Body == nil {
				continue
			}
			functionBindings, err := signatureFunctionBindings(snapshot.File, pkg, function)
			if err != nil {
				return false, err
			}
			found = true
			for _, binding := range functionBindings {
				if _, exists := seen[binding]; exists {
					continue
				}
				seen[binding] = struct{}{}
				*bindings = append(*bindings, binding)
			}
		}
	}
	return found, nil
}

func signatureFunctionBindings(file string, pkg *packages.Package, function *ast.FuncDecl) ([]astedit.SignatureBodyBinding, error) {
	objectIndex := make(map[types.Object]int)
	for index, object := range signatureParameterObjects(pkg, function) {
		if object == nil {
			return nil, fmt.Errorf("parameter %d has no resolved Go object", index)
		}
		objectIndex[object] = index
	}
	bodyStart, bodyEnd := pkg.Fset.Position(function.Body.Pos()).Offset, pkg.Fset.Position(function.Body.End()).Offset
	if bodyStart < 0 || bodyEnd <= bodyStart {
		return nil, fmt.Errorf("function body positions are unavailable")
	}
	var bindings []astedit.SignatureBodyBinding
	ast.Inspect(function.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		index, parameter := objectIndex[pkg.TypesInfo.Uses[identifier]]
		if !parameter {
			return true
		}
		bindings = append(bindings, astedit.SignatureBodyBinding{
			File: file, Start: pkg.Fset.Position(identifier.Pos()).Offset,
			End: pkg.Fset.Position(identifier.End()).Offset, SourceIndex: index,
		})
		return true
	})
	return bindings, nil
}

func signatureParameterObjects(pkg *packages.Package, function *ast.FuncDecl) []types.Object {
	var objects []types.Object
	for _, field := range function.Type.Params.List {
		for _, name := range field.Names {
			objects = append(objects, pkg.TypesInfo.Defs[name])
		}
	}
	return objects
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
