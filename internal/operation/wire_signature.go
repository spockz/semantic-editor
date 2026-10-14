// WHY: Exposes proven Go signature transformations through typed, buffered operations.

package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const signatureChangeConsumer = "semantic_change_signature"

const signatureReplaceBodyConsumer = "semantic_replace_body"

const signatureEditReceiptField = editReceiptField

const introduceParameterConsumer = "semantic_introduce_parameter"

const introduceParameterObjectConsumer = "semantic_introduce_parameter_object"

func signatureTargetSourceFields(field string) []string {
	return []string{
		signatureChangeConsumer + ".result.function_target." + field,
		introduceParameterConsumer + ".result.function_target." + field,
		introduceParameterObjectConsumer + ".result.function_target." + field,
	}
}

func signatureReceiptSourceFields(field string) []string {
	return []string{
		signatureChangeConsumer + ".result.edit_receipt.continuations[]." + field,
		introduceParameterConsumer + ".result.edit_receipt.continuations[]." + field,
		introduceParameterObjectConsumer + ".result.edit_receipt.continuations[]." + field,
	}
}

type signatureCallTarget struct {
	packagePath string
	file        string
	offset      int
}

func signatureOutputContract() OperationOutputContract {
	target := SymbolTargetSchema().Properties
	inspectors := []string{"semantic_inspect_symbol"}
	consumers := []string{signatureChangeConsumer, "semantic_introduce_parameter", "semantic_introduce_parameter_object", signatureReplaceBodyConsumer}
	functionTarget := ValueSchema{Kind: SchemaObject, Properties: target, Required: []string{"file", "symbol", wireExpectedRevision}}
	result := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"result": {Kind: SchemaString}, "file": {Kind: SchemaString}, "symbol": {Kind: SchemaString}, "diff": {Kind: SchemaString}, "function_target": functionTarget, signatureEditReceiptField: EditReceiptSchema(),
	}, Required: []string{"result", signatureEditReceiptField}}
	return OperationOutputContract{ResultSchema: &result, Continuations: []OutputContinuation{
		{Path: "result.edit_receipt.continuations[].file", Schema: target["file"], Role: RoleTarget, Consumers: inspectors},
		{Path: "result.edit_receipt.continuations[].symbol", Schema: target["symbol"], Role: RoleTarget, Consumers: inspectors},
		{Path: "result.function_target.file", Schema: target["file"], Role: RoleTarget, Consumers: consumers, Optional: true},
		{Path: "result.function_target.symbol", Schema: target["symbol"], Role: RoleTarget, Consumers: consumers, Optional: true},
		{Path: "result.function_target.expected_revision", Schema: target[wireExpectedRevision], Role: RoleTarget, Consumers: consumers, Optional: true},
	}}
}

type signatureFunctionTargetProof struct {
	Target SymbolTarget
	Proven bool
}

func finalizedSignatureFunctionTarget(ctx context.Context, cc CallContext, result structuralPlanResult) (signatureFunctionTargetProof, error) {
	if cc.InBatch || result.Receipt.Status == EditAppliedWorking || result.Edit.Symbol == "" {
		return signatureFunctionTargetProof{}, nil
	}
	candidate, found := uniqueSignatureFunctionCandidate(result)
	if !found || candidate.ExpectedRevision == nil {
		return signatureFunctionTargetProof{}, nil
	}
	live, err := signatureFunctionCandidateSurvives(ctx, cc.WorkDir, candidate, result.Edit.Symbol)
	if err != nil {
		return signatureFunctionTargetProof{}, err
	}
	return signatureFunctionTargetProof{Target: candidate, Proven: live}, nil
}

func uniqueSignatureFunctionCandidate(result structuralPlanResult) (SymbolTarget, bool) {
	var candidate SymbolTarget
	matches := 0
	for _, continuation := range result.Receipt.Continuations {
		if continuation.File == result.Edit.Display && continuation.Symbol == result.Edit.Symbol {
			candidate = continuation
			matches++
		}
	}
	return candidate, matches == 1
}

func signatureFunctionCandidateSurvives(ctx context.Context, root string, candidate SymbolTarget, symbol string) (bool, error) {
	actual, exists, err := receiptFileRevision(ctx, root, candidate.File, false)
	if err != nil {
		return false, fmt.Errorf("verify finalized signature function target: %w", err)
	}
	if !exists || actual.Revision != *candidate.ExpectedRevision {
		return false, nil
	}
	source, err := sourcebuf.ReadFile(ctx, resolveWorkPath(root, candidate.File))
	if err != nil {
		return false, fmt.Errorf("read finalized signature function target: %w", err)
	}
	if structuralRevision(source) != actual.Revision {
		return false, nil
	}
	liveFunction, err := signatureSourceHasFreeFunction(source, candidate.File, symbol)
	if err != nil {
		return false, fmt.Errorf("parse finalized signature function target: %w", err)
	}
	return liveFunction, nil
}

func signatureSourceHasFreeFunction(source []byte, filename, symbol string) (bool, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return false, err
	}
	functionCount := 0
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == symbol && !token.IsExported(function.Name.Name) {
			functionCount++
		}
	}
	return functionCount == 1, nil
}

func projectSignatureExpansion(result structuralPlanResult) any {
	out := map[string]any{"result": result.Edit.Detail, signatureEditReceiptField: result.Receipt}
	if result.FunctionTarget != nil {
		out["function_target"] = *result.FunctionTarget
		out["file"], out["symbol"] = result.FunctionTarget.File, result.FunctionTarget.Symbol
	}
	if result.Edit.Diff != "" {
		out["diff"] = result.Edit.Diff
	}
	return out
}

func changeSignatureDef() Def[map[string]any, structuralPlanResult] {
	mapping := ParameterMappingSchema()
	mappings := ValueSchema{Kind: SchemaArray, Items: &mapping}
	params := make([]ParameterContract, 0, 9)
	params = append(params, []ParameterContract{
		structuralStringParam("file", "Target source file: Go file in the active module or Java file in a trusted JDT LS workspace.", true, true, RoleTarget, append([]string{"semantic_inspect_symbol.result.continuations[].file", "semantic_lookup.file"}, signatureTargetSourceFields("file")...)...),
		structuralStringParam("symbol", "Target: exact private Go free-function identity or signature-qualified Java method; copy it from semantic_lookup or semantic_inspect_symbol.", true, true, RoleTarget, append([]string{"semantic_inspect_symbol.result.continuations[].symbol", "semantic_lookup.symbol"}, signatureTargetSourceFields("symbol")...)...),
		{Name: signatureParameters, JSONName: signatureParameters, CLIName: signatureParameters, Type: ParamArray, Required: true, Role: RoleValue, Description: "Replacement: complete ordered parameter mapping. Preserve Go's supported append-only proof. For Java, map existing parameters by source_index and give each added parameter a closed argument expression used at every existing call site.", Schema: &mappings},
		structuralStringParam(wireExpectedRevision, "Precondition: full-file revision copied from semantic_inspect_symbol or the same finalized function_target record for this file and symbol; stale revisions are refused.", false, true, RoleTarget, append([]string{"semantic_inspect_symbol.result.continuations[].expected_revision"}, signatureTargetSourceFields(wireExpectedRevision)...)...),
	}...)
	params = append(params, readProjectParams[:5]...)
	return Def[map[string]any, structuralPlanResult]{
		Key: "change_signature", Summary: "Use this tool instead of manually changing a Java method signature and its resolved callers through JDT LS, or a supported private Go free-function signature and its proven direct callers. Supply the complete ordered parameter mapping. New Java parameters require a default argument expression used at every existing call site; Go keeps its stricter append-only proof. Standalone results expose a finalized function_target; receipt continuations remain available for inspection. Buffered results report applied_working without claiming disk publication." + automaticVerificationGuidance,
		Params: params, Level: LevelWorkspace, CLIName: "change-signature", MCPName: "semantic_change_signature", Batchable: true, BufferedBatch: true,
		ExampleRaw: map[string]any{"file": "service.go", "symbol": "process", signatureParameters: []any{
			map[string]any{wireSourceIndex: 0, "parameter": map[string]any{"name": "id", "type": "int"}},
			map[string]any{"parameter": map[string]any{"name": "label", "type": "string"}, "argument": "\"ready\""},
		}},
		Output: func() *OperationOutputContract { output := signatureOutputContract(); return &output }(),
		Parse: func(raw map[string]any) (map[string]any, error) {
			if err := CheckParams(raw, params); err != nil {
				return nil, err
			}
			return maps.Clone(raw), nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, map[string]any) (structuralPlanResult, error){backend.LanguageGo: runChangeSignature, backend.LanguageJava: runChangeSignatureJava},
		Format:   func(result structuralPlanResult) (string, error) { return formatFileEdit(result.Edit) },
		FinalizeOutput: func(cc CallContext, _ map[string]any, result structuralPlanResult) (structuralPlanResult, error) {
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			receipt, err := finalizeCallReceipt(ctx, cc, cc.WorkDir, result.Draft, result.Resolve)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize change-signature receipt: %w", err)
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
		ProjectOutput: func(result structuralPlanResult) (any, error) {
			return projectSignatureExpansion(result), nil
		},
	}
}

type signatureChangeIntent struct {
	Symbol           string
	Mappings         []astedit.SignatureParameterMapping
	ExpectedRevision string
	NewParameterName string
}

func runChangeSignature(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file := stringParam(raw, "file")
	snapshot, err := prepareStructuralSnapshot(ctx, cc, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	intent, err := parseSignatureChangeIntent(raw, snapshot.Sources[snapshot.File])
	if err != nil {
		return structuralPlanResult{}, err
	}
	plan, err := planSignatureChange(snapshot, intent)
	if err != nil {
		return structuralPlanResult{}, err
	}
	return applySignatureChange(cc, file, snapshot, intent, plan)
}

func parseSignatureChangeIntent(raw map[string]any, before []byte) (signatureChangeIntent, error) {
	intent := signatureChangeIntent{Symbol: stringParam(raw, "symbol")}
	if expected, supplied := raw[wireExpectedRevision]; supplied {
		revision, ok := expected.(string)
		if !ok || revision == "" {
			return signatureChangeIntent{}, fmt.Errorf("expected_revision must be a non-empty string")
		}
		if revision != structuralRevision(before) {
			return signatureChangeIntent{}, fmt.Errorf("expected_revision does not match the selected full-file revision")
		}
		intent.ExpectedRevision = revision
	}
	mappings, err := decodeSignatureMappings(raw[signatureParameters])
	if err != nil {
		return signatureChangeIntent{}, err
	}
	intent.Mappings = mappings
	for _, mapping := range mappings {
		if mapping.SourceIndex != nil {
			continue
		}
		if intent.NewParameterName != "" {
			return signatureChangeIntent{}, fmt.Errorf("only one new parameter is supported")
		}
		intent.NewParameterName = mapping.Name
	}
	return intent, nil
}

func planSignatureChange(snapshot *structuralSnapshot, intent signatureChangeIntent) (astedit.SignaturePlan, error) {
	proof, err := collectSignatureCallSetWithName(snapshot, intent.Symbol, intent.NewParameterName)
	if err != nil {
		return astedit.SignaturePlan{}, err
	}
	request := astedit.SignatureChangeRequest{File: snapshot.File, Symbol: intent.Symbol, Parameters: intent.Mappings, ExpectedRevision: intent.ExpectedRevision}
	plan, err := astedit.PlanChangeSignature(request, proof)
	if err != nil {
		return astedit.SignaturePlan{}, err
	}
	if err := validateSignatureOverlay(snapshot, plan); err != nil {
		return astedit.SignaturePlan{}, err
	}
	return plan, nil
}

func applySignatureChange(cc CallContext, file string, snapshot *structuralSnapshot, intent signatureChangeIntent, plan astedit.SignaturePlan) (structuralPlanResult, error) {
	before := snapshot.Sources[snapshot.File]
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
	edit := FileEditRes{File: snapshot.File, Display: displayWorkPath(cc.WorkDir, file), Symbol: intent.Symbol,
		Detail: "Updated the function signature and every proven direct caller.", Diff: strings.Join(diffs, "\n"), Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}
	if !changed {
		edit.Detail = "The requested signature is already present."
	}
	if applyErr != nil {
		edit.Detail = "Signature output publication did not complete: " + applyErr.Error()
	}
	draft := EditReceiptDraft{Operation: "change_signature", Status: EditPersisted, ReadSet: readSet}
	if !changed {
		draft.Status = EditNoOp
	} else if diagnosticStatus == "changed" {
		draft.Status = EditPersistedWithDiagnostics
	}
	planned := make(map[string][]byte, len(plan.Files))
	for _, item := range plan.Files {
		planned[item.File] = item.After
		relative, err := filepath.Rel(cc.WorkDir, item.File)
		if err != nil || !filepath.IsLocal(relative) {
			return structuralPlanResult{}, fmt.Errorf("planned file %q is outside workspace", item.File)
		}
		draft.PlannedChanges = append(draft.PlannedChanges, FileRevision{Path: filepath.ToSlash(relative), Revision: structuralRevision(item.After)})
	}
	finalSource := before
	if changedSource, ok := planned[snapshot.File]; ok {
		finalSource = changedSource
	}
	continuationRevision := structuralRevision(finalSource)
	draft.Continuations = []SymbolTarget{{File: edit.Display, Symbol: intent.Symbol, ExpectedRevision: &continuationRevision}}
	resolve := func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		return resolveSignatureContinuation(resolveCtx, cc.WorkDir, candidate)
	}
	return structuralPlanResult{Edit: edit, Draft: draft, Resolve: resolve}, nil
}

func decodeSignatureMappings(value any) ([]astedit.SignatureParameterMapping, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode parameter mappings: %w", err)
	}
	var mappings []ParameterMapping
	if err := json.Unmarshal(encoded, &mappings); err != nil {
		return nil, fmt.Errorf("decode parameter mappings: %w", err)
	}
	if err := ValidateParameterMappings(mappings); err != nil {
		return nil, fmt.Errorf("validate parameter mappings: %w", err)
	}
	converted := make([]astedit.SignatureParameterMapping, 0, len(mappings))
	for index, mapping := range mappings {
		if mapping.Parameter.Variadic != nil && *mapping.Parameter.Variadic {
			return nil, fmt.Errorf("parameter mapping %d cannot be variadic", index)
		}
		argument := ""
		if mapping.Argument != nil {
			argument = *mapping.Argument
		}
		converted = append(converted, astedit.SignatureParameterMapping{SourceIndex: mapping.SourceIndex, Name: mapping.Parameter.Name,
			Type: mapping.Parameter.Type, Argument: argument})
	}
	return converted, nil
}

func compiledGoFiles(pkgs []*packages.Package) (map[string]bool, error) {
	compiled := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, filename := range pkg.CompiledGoFiles {
			absolute, err := filepath.Abs(filename)
			if err != nil {
				return nil, fmt.Errorf("resolve compiled Go file: %w", err)
			}
			compiled[filepath.Clean(absolute)] = true
		}
	}
	return compiled, nil
}

func checkSignatureVariantFile(snapshot *structuralSnapshot, absolute, symbol string) error {
	source, err := sourcebuf.ReadFile(snapshot.Ctx, absolute)
	if err != nil {
		return fmt.Errorf("read inactive signature variant %q: %w", absolute, err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), absolute, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse inactive signature variant %q: %w", absolute, err)
	}
	mentions := false
	ast.Inspect(file, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && identifier.Name == symbol {
			mentions = true
			return false
		}
		return true
	})
	snapshot.Sources[absolute] = bytes.Clone(source)
	if mentions {
		return fmt.Errorf("function %q may have a caller or alternate definition in inactive source variant %s", symbol, absolute)
	}
	return nil
}

func rejectSignatureVariantReferences(snapshot *structuralSnapshot, symbol string) error {
	compiled, err := compiledGoFiles(snapshot.Packages)
	if err != nil {
		return err
	}
	packageDir := filepath.Dir(snapshot.File)
	return filepath.WalkDir(packageDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan signature package variant %q: %w", path, walkErr)
		}
		if entry.IsDir() {
			if path != packageDir && (strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") || entry.Name() == "testdata" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve signature variant path: %w", err)
		}
		if compiled[filepath.Clean(absolute)] {
			return nil
		}
		return checkSignatureVariantFile(snapshot, absolute, symbol)
	})
}

func validateSignatureOverlay(snapshot *structuralSnapshot, plan astedit.SignaturePlan) error {
	extra := make(map[string][]byte, len(plan.Files))
	for _, file := range plan.Files {
		before, ok := snapshot.Sources[file.File]
		if !ok || !bytes.Equal(before, file.Before) {
			return fmt.Errorf("signature preimage for %q differs from the analyzed read set", file.File)
		}
		extra[file.File] = bytes.Clone(file.After)
	}
	if len(extra) == 0 {
		return nil
	}
	loaded, _, err := loadStructuralPackages(snapshot.Ctx, snapshot.Root, extra)
	if err != nil {
		return fmt.Errorf("typecheck planned signature output: %w", err)
	}
	for _, pkg := range loaded {
		if len(pkg.Errors) != 0 {
			return fmt.Errorf("signature output fails Go type analysis in %s: %s", pkg.PkgPath, pkg.Errors[0].Msg)
		}
	}
	return nil
}

func applySignaturePlan(snapshot *structuralSnapshot, plan astedit.SignaturePlan) (bool, []string, error) {
	changes := make([]sourcebuf.Change, 0, len(plan.Files))
	diffs := make([]string, 0, len(plan.Files))
	for _, file := range plan.Files {
		current, err := sourcebuf.ReadFile(snapshot.Ctx, file.File)
		if err != nil {
			return false, nil, fmt.Errorf("revalidate signature output %q: %w", file.File, err)
		}
		if !bytes.Equal(current, file.Before) {
			return false, nil, fmt.Errorf("signature output preimage changed for %q", file.File)
		}
		changes = append(changes, sourcebuf.Change{Path: file.File, Before: file.Before, After: file.After})
		diffs = append(diffs, astedit.UnifiedDiff(displayWorkPath(snapshot.Root, file.File), string(file.Before), string(file.After)))
	}
	if len(changes) == 0 {
		return false, nil, nil
	}
	if err := snapshot.Buffer.AssertDisk(); err != nil {
		return false, nil, fmt.Errorf("revalidate signature read set: %w", err)
	}
	if err := snapshot.Buffer.Apply(snapshot.Ctx, changes); err != nil {
		return false, nil, fmt.Errorf("apply complete signature output set: %w", err)
	}
	if snapshot.owned {
		if err := snapshot.Buffer.Flush(snapshot.Ctx); err != nil {
			return true, diffs, fmt.Errorf("publish signature output set: %w", err)
		}
	}
	return true, diffs, nil
}

func resolveSignatureContinuation(ctx context.Context, root string, candidate SymbolTarget) (SymbolTarget, bool, error) {
	{
		path := resolveWorkPath(root, candidate.File)
		source, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return SymbolTarget{}, false, fmt.Errorf("read continuation candidate %q: %w", path, err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return SymbolTarget{}, false, fmt.Errorf("parse continuation candidate %q: %w", path, err)
		}
		for _, declaration := range file.Decls {
			switch value := declaration.(type) {
			case *ast.FuncDecl:
				if value.Recv == nil && value.Name.Name == candidate.Symbol && !token.IsExported(value.Name.Name) {
					return candidate, true, nil
				}
			case *ast.GenDecl:
				for _, spec := range value.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && typeSpec.Name.Name == candidate.Symbol && !token.IsExported(typeSpec.Name.Name) {
						return candidate, true, nil
					}
				}
			}
		}
		return SymbolTarget{}, false, nil
	}
}

func collectSignatureCallSetWithName(snapshot *structuralSnapshot, symbol, newParameterName string) (astedit.SignatureCallSetProof, error) {
	target, err := resolveSignatureCallTarget(snapshot, symbol, newParameterName)
	if err != nil {
		return astedit.SignatureCallSetProof{}, err
	}
	if variantErr := rejectSignatureVariantReferences(snapshot, symbol); variantErr != nil {
		return astedit.SignatureCallSetProof{}, variantErr
	}
	proof := astedit.SignatureCallSetProof{Complete: true}
	paths, err := signatureWorkspacePaths(snapshot.Root, snapshot.Sources)
	if err != nil {
		return astedit.SignatureCallSetProof{}, err
	}
	for _, path := range paths {
		if filepath.Ext(path) == ".go" {
			proof.Files = append(proof.Files, astedit.SignatureSourceFile{File: path, Source: snapshot.Sources[path]})
		}
	}
	seen := make(map[string]bool)
	for _, pkg := range snapshot.Packages {
		calls, err := collectSignaturePackageCalls(pkg, target, symbol, seen)
		if err != nil {
			return astedit.SignatureCallSetProof{}, err
		}
		proof.Calls = append(proof.Calls, calls...)
	}
	return proof, nil
}

func signatureNameCollision(pkg *packages.Package, body *ast.BlockStmt, name string) error {
	if name == "" || body == nil {
		return nil
	}
	var collision *ast.Ident
	ast.Inspect(body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok || identifier.Name != name {
			return true
		}
		if pkg.TypesInfo.Uses[identifier] != nil || pkg.TypesInfo.Defs[identifier] != nil {
			collision = identifier
			return false
		}
		return true
	})
	if collision != nil {
		return fmt.Errorf("new parameter %q conflicts with an existing body binding at %s", name, pkg.Fset.Position(collision.Pos()))
	}
	return nil
}

func resolveSignatureCallTarget(snapshot *structuralSnapshot, symbol, newParameterName string) (signatureCallTarget, error) {
	var result signatureTargetSearch
	for _, pkg := range snapshot.Packages {
		candidate, err := signatureTargetInPackage(snapshot, pkg, symbol, newParameterName)
		if err != nil {
			return signatureCallTarget{}, err
		}
		if !candidate.Found {
			continue
		}
		if result.Found {
			return signatureCallTarget{}, fmt.Errorf("target %q is ambiguous across loaded packages", symbol)
		}
		result = candidate
	}
	if !result.Found {
		return signatureCallTarget{}, fmt.Errorf("private function %q was not found in the selected file", symbol)
	}
	return result.Target, nil
}

type signatureTargetSearch struct {
	Target signatureCallTarget
	Found  bool
}

func signatureTargetInPackage(snapshot *structuralSnapshot, pkg *packages.Package, symbol, newParameterName string) (signatureTargetSearch, error) {
	if pkg.TypesInfo == nil || pkg.Fset == nil {
		return signatureTargetSearch{}, fmt.Errorf("package %s has incomplete Go type information", pkg.PkgPath)
	}
	if pkg.ID != pkg.PkgPath {
		return signatureTargetSearch{}, nil
	}
	var result signatureTargetSearch
	for _, file := range pkg.Syntax {
		absolute, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
		if err != nil {
			return signatureTargetSearch{}, fmt.Errorf("resolve analyzed Go file: %w", err)
		}
		if absolute != snapshot.File {
			continue
		}
		candidate, err := signatureTargetInFile(pkg, file, symbol, newParameterName)
		if err != nil {
			return signatureTargetSearch{}, err
		}
		if !candidate.Found {
			continue
		}
		if result.Found {
			return signatureTargetSearch{}, fmt.Errorf("target %q is ambiguous in the selected file", symbol)
		}
		result = candidate
	}
	return result, nil
}

func signatureTargetInFile(pkg *packages.Package, file *ast.File, symbol, newParameterName string) (signatureTargetSearch, error) {
	var result signatureTargetSearch
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != symbol {
			continue
		}
		candidate, err := signatureTargetForFunction(pkg, function, symbol, newParameterName)
		if err != nil {
			return signatureTargetSearch{}, err
		}
		if result.Found {
			return signatureTargetSearch{}, fmt.Errorf("target %q is ambiguous in the selected file", symbol)
		}
		result = candidate
	}
	return result, nil
}

func signatureTargetForFunction(pkg *packages.Package, function *ast.FuncDecl, symbol, newParameterName string) (signatureTargetSearch, error) {
	if token.IsExported(symbol) || function.Type.TypeParams != nil && len(function.Type.TypeParams.List) != 0 {
		return signatureTargetSearch{}, fmt.Errorf("change signature supports only private non-generic free functions")
	}
	object, ok := pkg.TypesInfo.Defs[function.Name].(*types.Func)
	if !ok || object == nil || object.Pkg() == nil {
		return signatureTargetSearch{}, fmt.Errorf("target %q has no resolved Go function object", symbol)
	}
	if err := signatureNameCollision(pkg, function.Body, newParameterName); err != nil {
		return signatureTargetSearch{}, err
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Variadic() {
		return signatureTargetSearch{}, fmt.Errorf("variadic or unresolved target signatures are unsupported")
	}
	position := pkg.Fset.Position(object.Pos())
	target := signatureCallTarget{packagePath: pkg.PkgPath, file: filepath.Clean(position.Filename), offset: position.Offset}
	return signatureTargetSearch{Target: target, Found: true}, nil
}

func signatureObjectMatches(pkg *packages.Package, object types.Object, target signatureCallTarget, symbol string) bool {
	if object == nil || object.Pkg() == nil || object.Pkg().Path() != target.packagePath || object.Name() != symbol {
		return false
	}
	position := pkg.Fset.Position(object.Pos())
	absolute, err := filepath.Abs(position.Filename)
	return err == nil && filepath.Clean(absolute) == target.file && position.Offset == target.offset
}

func collectSignaturePackageCalls(pkg *packages.Package, target signatureCallTarget, symbol string, seen map[string]bool) ([]astedit.SignatureCallSite, error) {
	if pkg.TypesInfo == nil || pkg.Fset == nil {
		return nil, fmt.Errorf("package %s has incomplete call analysis", pkg.PkgPath)
	}
	if len(pkg.Errors) != 0 {
		return nil, fmt.Errorf("cannot prove complete call set: package %s: %s", pkg.PkgPath, pkg.Errors[0].Msg)
	}
	calls := make([]astedit.SignatureCallSite, 0)
	for _, file := range pkg.Syntax {
		fileCalls, err := collectSignatureFileCalls(pkg, file, target, symbol, seen)
		if err != nil {
			return nil, err
		}
		calls = append(calls, fileCalls...)
	}
	return calls, nil
}

func collectSignatureFileCalls(pkg *packages.Package, file *ast.File, target signatureCallTarget, symbol string, seen map[string]bool) ([]astedit.SignatureCallSite, error) {
	var parents []ast.Node
	var calls []astedit.SignatureCallSite
	var visitErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			if len(parents) > 0 {
				parents = parents[:len(parents)-1]
			}
			return false
		}
		if visitErr != nil {
			return false
		}
		identifier, ok := node.(*ast.Ident)
		if ok && signatureObjectMatches(pkg, pkg.TypesInfo.Uses[identifier], target, symbol) {
			call, err := directSignatureCall(pkg, identifier, parents, symbol)
			if err != nil {
				visitErr = err
				return false
			}
			if call != nil {
				key := fmt.Sprintf("%s:%d:%d", call.File, call.Start, call.End)
				if !seen[key] {
					seen[key] = true
					calls = append(calls, *call)
				}
			}
		}
		parents = append(parents, node)
		return true
	})
	if visitErr != nil {
		return nil, visitErr
	}
	return calls, nil
}

func directSignatureCall(pkg *packages.Package, identifier *ast.Ident, parents []ast.Node, symbol string) (*astedit.SignatureCallSite, error) {
	var parent ast.Node
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	}
	call, direct := parent.(*ast.CallExpr)
	if !direct || call.Fun != identifier {
		return nil, fmt.Errorf("function %q has a non-direct use at %s; function values and indirect callers are unsupported", symbol, pkg.Fset.Position(identifier.Pos()))
	}
	start, end := pkg.Fset.Position(call.Pos()), pkg.Fset.Position(call.End())
	path, err := filepath.Abs(start.Filename)
	if err != nil {
		return nil, fmt.Errorf("resolve caller path: %w", err)
	}
	return &astedit.SignatureCallSite{File: path, Start: start.Offset, End: end.Offset}, nil
}

func signatureWorkspacePaths(workDir string, sources map[string][]byte) ([]string, error) {
	paths := make([]string, 0, len(sources))
	for path := range sources {
		relative, err := filepath.Rel(workDir, path)
		if err != nil {
			return nil, fmt.Errorf("resolve structural source path %q: %w", path, err)
		}
		if !filepath.IsLocal(relative) {
			continue
		}
		paths = append(paths, path)
	}
	for index := 1; index < len(paths); index++ {
		for position := index; position > 0 && strings.Compare(paths[position], paths[position-1]) < 0; position-- {
			paths[position], paths[position-1] = paths[position-1], paths[position]
		}
	}
	return paths, nil
}

func signatureReadSet(workDir string, snapshot *structuralSnapshot) ([]FileRevision, error) {
	paths, err := signatureWorkspacePaths(workDir, snapshot.Sources)
	if err != nil {
		return nil, err
	}
	readSet := make([]FileRevision, 0, len(paths))
	for _, path := range paths {
		relative, err := filepath.Rel(workDir, path)
		if err != nil || !filepath.IsLocal(relative) {
			return nil, fmt.Errorf("signature read-set path %q is outside workspace %q", path, workDir)
		}
		readSet = append(readSet, FileRevision{Path: filepath.ToSlash(relative), Revision: structuralRevision(snapshot.Sources[path])})
	}
	return readSet, nil
}
