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

var extractFunctionParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: simple identifier for the helper declaration created by extraction", Required: true, SourceFields: sourceSymbolContinuations},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified owner function or method containing the statements", Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Helper declaration form", Required: true, Enums: []string{"function", "method"}},
	{Name: "start_line", CLIName: "start-line", JSONName: "start_line", Type: ParamInteger, Description: "First selected statement's starting line; must match a direct statement boundary", Required: true},
	{Name: "end_line", CLIName: "end-line", JSONName: "end_line", Type: ParamInteger, Description: "Last selected statement's ending line; must match a direct statement boundary", Required: true},
}

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

func extractFunctionDef() Def[ExtractFunctionReq, FileEditRes] {
	return Def[ExtractFunctionReq, FileEditRes]{
		Key: "extract_function", Summary: "Use this tool instead of manually copying statements when extracting a safe contiguous top-level call region from a Go function or method into a local helper. Parameters are inferred from named owner parameters and the receiver. One leading single-variable declaration may be returned when its type is supported; address-taken inputs or results, closure-captured inputs, direct recover calls, selector calls on forwarded values, other local data flow, branching, loops, comments, composite literals, generic owners, and variadic parameters are refused." + automaticVerificationGuidance,
		Params: extractFunctionParams, Level: LevelFile, CLIName: "extract-function", MCPName: "semantic_extract_function",
		Parse:      parseExtractFunction,
		Handlers:   map[backend.LanguageID]func(context.Context, CallContext, ExtractFunctionReq) (FileEditRes, error){backend.LanguageGo: runExtractFunction},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "symbol": "writeSummary", wireInFunction: "Server.Serve", "kind": "function", "start_line": 12, "end_line": 12},
		Batchable:  true,
	}
}

func registerExtractOps(registry *Registry) error {
	if err := Register(registry, extractFunctionDef()); err != nil {
		return fmt.Errorf("register extraction operation: %w", err)
	}
	return nil
}
