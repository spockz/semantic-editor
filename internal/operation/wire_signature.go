// WHY: Exposes proven Go signature transformations through typed, buffered operations.

package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"path/filepath"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/sourcebuf"
	"strings"

	"golang.org/x/tools/go/packages"
)

const signatureChangeConsumer = "semantic_change_signature"

const signatureReplaceBodyConsumer = "semantic_replace_body"

const signatureEditReceiptField = "edit_receipt"

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

func finalizedSignatureFunctionTarget(ctx context.Context, cc CallContext, result structuralPlanResult) (*SymbolTarget, error) {
	if cc.InBatch || result.Receipt.Status == EditAppliedWorking || result.Edit.Symbol == "" {
		return nil, nil
	}
	var candidate SymbolTarget
	matches := 0
	for _, continuation := range result.Receipt.Continuations {
		if continuation.File == result.Edit.Display && continuation.Symbol == result.Edit.Symbol {
			candidate = continuation
			matches++
		}
	}
	if matches != 1 || candidate.ExpectedRevision == nil {
		return nil, nil
	}
	actual, exists, err := receiptFileRevision(ctx, cc.WorkDir, candidate.File, false)
	if err != nil {
		return nil, fmt.Errorf("verify finalized signature function target: %w", err)
	}
	if !exists || actual.Revision != *candidate.ExpectedRevision {
		return nil, nil
	}
	source, err := sourcebuf.ReadFile(ctx, resolveWorkPath(cc.WorkDir, candidate.File))
	if err != nil {
		return nil, fmt.Errorf("read finalized signature function target: %w", err)
	}
	if structuralRevision(source) != actual.Revision {
		return nil, nil
	}
	liveFunction, err := signatureSourceHasFreeFunction(source, candidate.File, result.Edit.Symbol)
	if err != nil {
		return nil, fmt.Errorf("parse finalized signature function target: %w", err)
	}
	if !liveFunction {
		return nil, nil
	}
	return &candidate, nil
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
	params := []ParameterContract{
		structuralStringParam("file", wireTargetGoFileDescription, true, true, RoleTarget, append([]string{"semantic_inspect_symbol.result.continuations[].file", "semantic_lookup.file"}, signatureTargetSourceFields("file")...)...),
		structuralStringParam("symbol", "Target: private Go free function whose direct call arguments will be updated; copy its exact identity from semantic_lookup or a finalized function_target.", true, true, RoleTarget, append([]string{"semantic_inspect_symbol.result.continuations[].symbol", "semantic_lookup.symbol"}, signatureTargetSourceFields("symbol")...)...),
		{Name: signatureParameters, JSONName: signatureParameters, CLIName: signatureParameters, Type: ParamArray, Required: true, Role: RoleValue, Description: "Replacement: complete ordered parameter mapping; preserve every existing parameter by source_index and append one new mapping with a closed argument expression.", Schema: &mappings},
		structuralStringParam(wireExpectedRevision, "Precondition: full-file revision copied from semantic_inspect_symbol or the same finalized function_target record for this file and symbol; stale revisions are refused.", false, true, RoleTarget, append([]string{"semantic_inspect_symbol.result.continuations[].expected_revision"}, signatureTargetSourceFields(wireExpectedRevision)...)...),
	}
	return Def[map[string]any, structuralPlanResult]{
		Key: "change_signature", Summary: "Use this tool instead of manually editing one private, non-generic, non-variadic Go free-function signature and its callers. Preserve existing parameter order, names, types, and results, then append one parameter whose literal, nil, or Boolean argument is safe at every direct call. The complete active module call set includes same-package test variants. Function values, methods, exported functions, build-constrained references, and unresolved packages are refused. Standalone results expose a finalized function_target; receipt continuations remain available for inspection. Buffered results report applied_working without claiming disk publication." + automaticVerificationGuidance,
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
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, map[string]any) (structuralPlanResult, error){backend.LanguageGo: runChangeSignature},
		Format:   func(result structuralPlanResult) (string, error) { return formatFileEdit(result.Edit) },
		FinalizeOutput: func(cc CallContext, _ map[string]any, result structuralPlanResult) (structuralPlanResult, error) {
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			receipt, err := FinalizeEditReceipt(ctx, cc.WorkDir, cc.InBatch, result.Draft, result.Resolve)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize change-signature receipt: %w", err)
			}
			result.Receipt = receipt
			result.FunctionTarget, err = finalizedSignatureFunctionTarget(ctx, cc, result)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize signature function target: %w", err)
			}
			return result, nil
		},
		ProjectOutput: func(result structuralPlanResult) (any, error) {
			return projectSignatureExpansion(result), nil
		},
	}
}

func runChangeSignature(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file := stringParam(raw, "file")
	symbol := stringParam(raw, "symbol")
	snapshot, err := prepareStructuralSnapshot(ctx, cc, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	before := snapshot.Sources[snapshot.File]
	expected, supplied := raw[wireExpectedRevision]
	if supplied {
		revision, ok := expected.(string)
		if !ok || revision == "" {
			return structuralPlanResult{}, fmt.Errorf("expected_revision must be a non-empty string")
		}
		if revision != structuralRevision(before) {
			return structuralPlanResult{}, fmt.Errorf("expected_revision does not match the selected full-file revision")
		}
	}
	mappings, err := decodeSignatureMappings(raw[signatureParameters])
	if err != nil {
		return structuralPlanResult{}, err
	}
	newParameterName := ""
	for _, mapping := range mappings {
		if mapping.SourceIndex == nil {
			if newParameterName != "" {
				return structuralPlanResult{}, fmt.Errorf("only one new parameter is supported")
			}
			newParameterName = mapping.Name
		}
	}
	proof, err := collectSignatureCallSetWithName(snapshot, symbol, newParameterName)
	if err != nil {
		return structuralPlanResult{}, err
	}
	request := astedit.SignatureChangeRequest{File: snapshot.File, Symbol: symbol, Parameters: mappings}
	if supplied {
		request.ExpectedRevision = expected.(string)
	}
	plan, err := astedit.PlanChangeSignature(request, proof)
	if err != nil {
		return structuralPlanResult{}, err
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
	edit := FileEditRes{File: snapshot.File, Display: displayWorkPath(cc.WorkDir, file), Symbol: symbol,
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
	draft.Continuations = []SymbolTarget{{File: edit.Display, Symbol: symbol, ExpectedRevision: &continuationRevision}}
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

func rejectSignatureVariantReferences(snapshot *structuralSnapshot, symbol string) error {
	compiled := make(map[string]bool)
	for _, pkg := range snapshot.Packages {
		for _, filename := range pkg.CompiledGoFiles {
			absolute, err := filepath.Abs(filename)
			if err != nil {
				return fmt.Errorf("resolve compiled Go file: %w", err)
			}
			compiled[filepath.Clean(absolute)] = true
		}
	}
	packageDir := filepath.Dir(snapshot.File)
	return filepath.WalkDir(packageDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan signature package variant %q: %w", path, walkErr)
		}
		if entry.IsDir() {
			if path != packageDir && entry.Name() == "testdata" {
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
	if err := rejectSignatureVariantReferences(snapshot, symbol); err != nil {
		return astedit.SignatureCallSetProof{}, err
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
	var target signatureCallTarget
	found := false
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			return signatureCallTarget{}, fmt.Errorf("package %s has incomplete Go type information", pkg.PkgPath)
		}
		if pkg.ID != pkg.PkgPath {
			continue
		}
		for _, file := range pkg.Syntax {
			position := pkg.Fset.Position(file.Pos())
			absolute, err := filepath.Abs(position.Filename)
			if err != nil {
				return signatureCallTarget{}, fmt.Errorf("resolve analyzed Go file: %w", err)
			}
			if absolute != snapshot.File {
				continue
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Name.Name != symbol {
					continue
				}
				if token.IsExported(symbol) || function.Type.TypeParams != nil && len(function.Type.TypeParams.List) != 0 {
					return signatureCallTarget{}, fmt.Errorf("change signature supports only private non-generic free functions")
				}
				object, ok := pkg.TypesInfo.Defs[function.Name].(*types.Func)
				if !ok || object == nil || object.Pkg() == nil {
					return signatureCallTarget{}, fmt.Errorf("target %q has no resolved Go function object", symbol)
				}
				if err := signatureNameCollision(pkg, function.Body, newParameterName); err != nil {
					return signatureCallTarget{}, err
				}
				signature, ok := object.Type().(*types.Signature)
				if !ok || signature.Variadic() {
					return signatureCallTarget{}, fmt.Errorf("variadic or unresolved target signatures are unsupported")
				}
				declarationPosition := pkg.Fset.Position(object.Pos())
				if found {
					return signatureCallTarget{}, fmt.Errorf("target %q is ambiguous across loaded packages", symbol)
				}
				target = signatureCallTarget{packagePath: pkg.PkgPath, file: filepath.Clean(declarationPosition.Filename), offset: declarationPosition.Offset}
				found = true
			}
		}
	}
	if !found {
		return signatureCallTarget{}, fmt.Errorf("private function %q was not found in the selected file", symbol)
	}
	return target, nil
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
		var parents []ast.Node
		var visitErr error
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				if len(parents) > 0 {
					parents = parents[:len(parents)-1]
				}
				return false
			}
			identifier, ok := node.(*ast.Ident)
			if ok && signatureObjectMatches(pkg, pkg.TypesInfo.Uses[identifier], target, symbol) {
				var parent ast.Node
				if len(parents) > 0 {
					parent = parents[len(parents)-1]
				}
				call, direct := parent.(*ast.CallExpr)
				if !direct || call.Fun != identifier {
					visitErr = fmt.Errorf("function %q has a non-direct use at %s; function values and indirect callers are unsupported", symbol, pkg.Fset.Position(identifier.Pos()))
					return false
				}
				start, end := pkg.Fset.Position(call.Pos()), pkg.Fset.Position(call.End())
				path, err := filepath.Abs(start.Filename)
				if err != nil {
					visitErr = fmt.Errorf("resolve caller path: %w", err)
					return false
				}
				key := fmt.Sprintf("%s:%d:%d", path, start.Offset, end.Offset)
				if !seen[key] {
					seen[key] = true
					calls = append(calls, astedit.SignatureCallSite{File: path, Start: start.Offset, End: end.Offset})
				}
			}
			parents = append(parents, node)
			return true
		})
		if visitErr != nil {
			return nil, visitErr
		}
	}
	return calls, nil
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
