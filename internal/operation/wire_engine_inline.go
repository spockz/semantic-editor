// Package operation wires local Go refactorings into the shared CLI and MCP registry.
package operation

import (
	"context"

	"semedit/internal/astedit"
	"semedit/internal/backend"
)

// InlineVariableReq requests one bounded local-variable inline operation.
type InlineVariableReq struct {
	Project    backend.ProjectContext
	File       string
	Symbol     string
	InFunction string
}

// GetProjectContext returns the project selector used for registry dispatch.
func (r InlineVariableReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the project selector after ambient context merging.
func (r *InlineVariableReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var inlineVariableParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Go source file containing the local binding; path is relative to the workspace root", Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Description: "Target: live local binding in the selected owner; copy a compatible local identity from semantic_lookup. The binding is removed on success.", Required: true, SourceFields: []string{sourceLookupSymbol}},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified Go function or method containing the local binding", Required: true, SourceFields: sourceInFunctionContinuations},
}

func parseInlineVariable(raw map[string]any) (InlineVariableReq, error) {
	var req InlineVariableReq
	if err := CheckParams(raw, inlineVariableParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return req, err
	}
	return req, nil
}

func runInlineVariable(ctx context.Context, cc CallContext, req InlineVariableReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.InlineVariable(ctx, targetPath, req.InFunction, req.Symbol)
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File),
		Symbol: req.Symbol, InFunction: req.InFunction, Diff: diff,
		Detail: "Inlined local literal.", Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func inlineVariableDef() Def[InlineVariableReq, ExtractionResult] {
	return Def[InlineVariableReq, ExtractionResult]{
		Key:     "inline_variable",
		Summary: "Use this tool instead of replacing local uses by text to inline one Go local initialized by a literal into all direct argument or return-value uses in the immediately following statement. Literal substitution preserves the binding's value and does not duplicate runtime evaluation. The successful edit removes the binding and returns its surviving owner function as a live target; use semantic_lookup to select fresh constructs or bindings. The removed name is historical evidence and is not a follow-up target. Expressions, reassignment, shadowing, intervening statements, and unsupported use shapes are refused. Call inlining is not supported.",
		Params:  inlineVariableParams, Level: LevelFile, CLIName: "inline-variable", MCPName: semanticInlineVariableToolName,
		Parse:    parseInlineVariable,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InlineVariableReq) (ExtractionResult, error){backend.LanguageGo: runInlineVariableWithReceipt},
		Format:   formatExtractionResult,
		Output:   extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName),
		FinalizeOutput: func(cc CallContext, _ InlineVariableReq, result ExtractionResult) (ExtractionResult, error) {
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "limit", wireInFunction: "Server.Stop"},
		BufferedBatch: true, Batchable: true,
	}
}
