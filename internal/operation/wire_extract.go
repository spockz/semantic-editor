// Package operation wires bounded Go extraction into the shared CLI and MCP registry.
package operation

import (
	"context"
	"fmt"

	"semedit/internal/astedit"
	"semedit/internal/backend"
)

// ExtractFunctionReq selects a contiguous statement region and names its helper.
type ExtractFunctionReq struct {
	Project    backend.ProjectContext
	File       string
	Symbol     string
	InFunction string
	Kind       astedit.ExtractKind
	StartLine  int
	EndLine    int
}

// GetProjectContext returns the request project for registry dispatch.
func (r ExtractFunctionReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ExtractFunctionReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var extractFunctionParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Go source file or Java source file containing the selected method", Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh simple identifier for the helper declaration to create", Required: true},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified owner function or method containing the statements", Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Helper declaration form", Required: true, Enums: []string{"function", "method"}},
	{Name: "start_line", CLIName: "start-line", JSONName: "start_line", Type: ParamInteger, Description: "First selected statement's starting line; must match a direct statement boundary", Required: true},
	{Name: "end_line", CLIName: "end-line", JSONName: "end_line", Type: ParamInteger, Description: "Last selected statement's ending line; must match a direct statement boundary", Required: true},
}, readProjectParams...)

func parseExtractFunction(raw map[string]any) (ExtractFunctionReq, error) {
	var req ExtractFunctionReq
	if err := CheckParams(raw, extractFunctionParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		req.Project.Language = backend.LanguageGo
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return req, err
	}
	kind, err := ParseEnum(raw, "kind", "kind", []string{"function", "method"}, true, "")
	if err != nil {
		return req, err
	}
	req.Kind = astedit.ExtractKind(kind)
	if req.StartLine, err = ParseInteger(raw, "start_line", "start-line", true); err != nil {
		return req, err
	}
	if req.EndLine, err = ParseInteger(raw, "end_line", "end-line", true); err != nil {
		return req, err
	}
	return req, nil
}

func runExtractFunction(ctx context.Context, cc CallContext, req ExtractFunctionReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractFunction(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractOptions{
		Kind: req.Kind, StartLine: req.StartLine, EndLine: req.EndLine,
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Diff: diff, Detail: "Extracted statements into a " + string(req.Kind) + ".",
		Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func extractFunctionDef() Def[ExtractFunctionReq, ExtractionResult] {
	return Def[ExtractFunctionReq, ExtractionResult]{
		Key: "extract_function", Summary: "Use this tool instead of copying statements to extract a safe contiguous top-level call region from a Go function or method into a local helper. Parameters are inferred from named owner parameters and the receiver. One leading := declaration may return multiple distinct named live-out values when every result expression is a basic literal, a boolean literal, a named owner parameter, or a supported unary or parenthesized form of those expressions. Effectful, call, compound, and unsupported result-type expressions are refused before writing. Pass the returned live file and helper symbol to semantic_rename or semantic_replace_body. Address-taken values, closure-captured inputs, direct recover calls, selector calls on forwarded values, other local data flow, branching, loops, comments, composite literals, generic owners, and variadic parameters are refused." + automaticVerificationGuidance,
		Params: extractFunctionParams, Level: LevelFile, CLIName: "extract-function", MCPName: semanticExtractFunctionToolName,
		Parse:    parseExtractFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ExtractFunctionReq) (ExtractionResult, error){backend.LanguageGo: runExtractFunctionWithReceipt, backend.LanguageJava: runExtractFunctionJavaWithReceipt},
		Format:   formatExtractionResult,
		Output:   extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName),
		FinalizeOutput: func(cc CallContext, req ExtractFunctionReq, result ExtractionResult) (ExtractionResult, error) {
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractionOutput(cc, ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol, InFunction: req.InFunction}, result)
			}
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "writeSummary", wireInFunction: "Server.Serve", "kind": "function", "start_line": 12, "end_line": 12},
		BufferedBatch: true, Batchable: true,
	}
}

func registerExtractOps(registry *Registry) error {
	if err := Register(registry, extractConstantDef()); err != nil {
		return fmt.Errorf("register constant extraction operation: %w", err)
	}
	if err := Register(registry, extractFunctionDef()); err != nil {
		return fmt.Errorf("register extraction operation: %w", err)
	}
	if err := Register(registry, extractVariableDef()); err != nil {
		return fmt.Errorf("register variable extraction operation: %w", err)
	}
	return nil
}

const expectedRevisionJSONName = "expected_revision"

// ExtractConstantReq selects a constant expression and names its new package constant.
type ExtractConstantReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	Discriminator    string
	ConstructPath    string
	ExpectedRevision string
}

// GetProjectContext returns the requested language for dispatch.
func (r ExtractConstantReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the language selected by request dispatch.
func (r *ExtractConstantReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var extractConstantParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: source file containing the selected Go or Java method", Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh package-level constant identifier to create; choose a new simple name", Required: true},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified function or method containing the expression", Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: exact source text of one current compile-time constant expression", Required: true},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Role: RoleSelector, Description: "Selector: candidate path from an ambiguity diagnostic; repeated expressions require its matching expected_revision"},
	{Name: expectedRevisionJSONName, CLIName: "expected-revision", JSONName: expectedRevisionJSONName, Type: ParamString, Role: RoleTarget, Description: "Guard: file revision copied from the ambiguity diagnostic; stale revisions are refused", SourceFields: []string{sourceInspectExpectedRevision}},
}, readProjectParams...)

func parseExtractConstant(raw map[string]any) (ExtractConstantReq, error) {
	selection, err := parseExtractionSelection(raw, extractConstantParams)
	if err != nil {
		return ExtractConstantReq{}, err
	}
	return ExtractConstantReq(selection), nil
}

type extractionSelection struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	Discriminator    string
	ConstructPath    string
	ExpectedRevision string
}

func parseExtractionSelection(raw map[string]any, params []ParameterContract) (extractionSelection, error) {
	var selection extractionSelection
	if err := CheckParams(raw, params); err != nil {
		return selection, err
	}
	selection.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if selection.File, err = ParseString(raw, "file", "file", true); err != nil {
		return selection, err
	}
	if selection.Project, err = parseReadProject(raw, selection.File); err != nil {
		return selection, err
	}
	if selection.Project.Language == "" {
		selection.Project.Language = backend.LanguageGo
	}
	if selection.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return selection, err
	}
	if selection.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return selection, err
	}
	if selection.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, true); err != nil {
		return selection, err
	}
	if selection.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return selection, err
	}
	if selection.ExpectedRevision, err = ParseString(raw, expectedRevisionJSONName, "expected-revision", false); err != nil {
		return selection, err
	}
	if err := validateExtractionRevision(raw, selection); err != nil {
		return selection, err
	}
	return selection, nil
}

func validateExtractionRevision(raw map[string]any, selection extractionSelection) error {
	if value, present := rawLookup(raw, expectedRevisionJSONName, "expected-revision"); present && value != nil && selection.ExpectedRevision == "" {
		return fmt.Errorf("param %q must not be empty when supplied: %w", expectedRevisionJSONName, ErrInvalidParams)
	}
	if selection.ConstructPath != "" && selection.ExpectedRevision == "" {
		return fmt.Errorf("param %q is required when construct_path is supplied: %w", expectedRevisionJSONName, ErrInvalidParams)
	}
	return nil
}

func runExtractConstant(ctx context.Context, cc CallContext, req ExtractConstantReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractConstant(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractConstantOptions{
		Discriminator: req.Discriminator, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Diff: diff, Detail: "Extracted a compile-time constant expression.",
		Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func extractConstantDef() Def[ExtractConstantReq, ExtractionResult] {
	return Def[ExtractConstantReq, ExtractionResult]{
		Key: "extract_constant", Summary: "Use this tool instead of manually moving expressions into fields to extract one exact compile-time constant expression. Go preserves resolved constant type and value facts; Java delegates to JDT LS refactor.extract.constant and renames the generated field. Supply symbol, in_function, and exact-source discriminator; use construct_path with its matching expected_revision when the expression is ambiguous." + automaticVerificationGuidance,
		Params: extractConstantParams, Level: LevelFile, CLIName: "extract-constant", MCPName: "semantic_extract_constant",
		Parse:    parseExtractConstant,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ExtractConstantReq) (ExtractionResult, error){backend.LanguageGo: runExtractConstantWithReceipt, backend.LanguageJava: runExtractConstantJavaWithReceipt},
		Format:   formatExtractionResult,
		Output:   extractionReceiptOutput(semanticRenameToolName, semanticReplaceDeclToolName),
		FinalizeOutput: func(cc CallContext, req ExtractConstantReq, result ExtractionResult) (ExtractionResult, error) {
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractionOutput(cc, ExtractVariableReq(req), result)
			}
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "retryWindow", wireInFunction: "Retry", wireDiscriminator: "3 * 60"},
		BufferedBatch: true, Batchable: true,
	}
}

// ExtractVariableReq selects one exact expression and names its new local.
type ExtractVariableReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	Discriminator    string
	ConstructPath    string
	ExpectedRevision string
}

// GetProjectContext returns the request project for registry dispatch.
func (r ExtractVariableReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ExtractVariableReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var extractVariableParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: source file containing the selected Go or Java method", Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh simple local identifier to create; choose a new name not already visible in the owner", Required: true},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified owner function or method containing the selected call", Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: exact source text of the expression to extract; Java expressions are selected inside the named method", Required: true},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic; a supplied path requires its matching expected_revision"},
	{Name: expectedRevisionJSONName, CLIName: "expected-revision", JSONName: expectedRevisionJSONName, Type: ParamString, Role: RoleTarget, Description: "Guard: selected-file SHA-256 revision copied from semantic_inspect_symbol; required with a construct_path from the ambiguity diagnostic", SourceFields: []string{sourceInspectExpectedRevision}},
}, readProjectParams...)

func parseExtractVariable(raw map[string]any) (ExtractVariableReq, error) {
	selection, err := parseExtractionSelection(raw, extractVariableParams)
	if err != nil {
		return ExtractVariableReq{}, err
	}
	return ExtractVariableReq(selection), nil
}

func runExtractVariable(ctx context.Context, cc CallContext, req ExtractVariableReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractVariable(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractVariableOptions{
		Discriminator: req.Discriminator, ConstructPath: req.ConstructPath, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Discriminator: req.Discriminator, Diff: diff,
		Detail: "Extracted the call argument into a local variable.", Delta: delta,
		DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func extractVariableDef() Def[ExtractVariableReq, ExtractionResult] {
	return Def[ExtractVariableReq, ExtractionResult]{
		Key: "extract_variable", Summary: "Use this tool instead of manually rewriting a call to extract one exact expression into a local variable. Go supports one side-effect-free typed first argument of a direct call under strict data-flow checks; Java delegates extraction to a matching JDT LS refactor action, then renames the generated local to the requested symbol. Use construct_path and its matching selected-file expected_revision when the expression is ambiguous." + automaticVerificationGuidance,
		Params: extractVariableParams, Level: LevelFile, CLIName: "extract-variable", MCPName: semanticExtractVariableToolName,
		Parse: parseExtractVariable,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ExtractVariableReq) (ExtractionResult, error){
			backend.LanguageGo:   runExtractVariableWithReceipt,
			backend.LanguageJava: runExtractVariableJavaWithReceipt,
		},
		Format: formatExtractionResult,
		Output: extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName),
		FinalizeOutput: func(cc CallContext, req ExtractVariableReq, result ExtractionResult) (ExtractionResult, error) {
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractionOutput(cc, req, result)
			}
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "value", wireInFunction: "PrintValue", wireDiscriminator: "count + 1"},
		BufferedBatch: true, Batchable: true,
	}
}
