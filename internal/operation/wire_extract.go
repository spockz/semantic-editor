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
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh simple identifier for the helper declaration to create", Required: true},
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
		Key: "extract_function", Summary: "Use this tool instead of copying statements to extract a safe contiguous top-level call region from a Go function or method into a local helper. Parameters are inferred from named owner parameters and the receiver. One leading := declaration may return multiple distinct named live-out values when every result expression is a basic literal, a boolean literal, a named owner parameter, or a supported unary or parenthesized form of those expressions. Effectful, call, compound, and unsupported result-type expressions are refused before writing. Pass the returned live file and helper symbol to semantic_rename or semantic_replace_body. Address-taken values, closure-captured inputs, direct recover calls, selector calls on forwarded values, other local data flow, branching, loops, comments, composite literals, generic owners, and variadic parameters are refused." + automaticVerificationGuidance,
		Params: extractFunctionParams, Level: LevelFile, CLIName: "extract-function", MCPName: "semantic_extract_function",
		Parse:         parseExtractFunction,
		Handlers:      map[backend.LanguageID]func(context.Context, CallContext, ExtractFunctionReq) (FileEditRes, error){backend.LanguageGo: runExtractFunction},
		Format:        formatFileEdit,
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

// ExtractConstantReq selects a constant expression and names its new package constant.
type ExtractConstantReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	Discriminator    string
	ExpectedRevision string
}

// GetProjectContext returns the requested language for dispatch.
func (r ExtractConstantReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the language selected by request dispatch.
func (r *ExtractConstantReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var extractConstantParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh package-level constant identifier to create; choose a new simple name", Required: true},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified function or method containing the expression", Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: exact source text of one current compile-time constant expression", Required: true},
	{Name: "expected_revision", CLIName: "expected-revision", JSONName: "expected_revision", Type: ParamString, Description: "Opaque SHA-256 revision of the selected Go source file; stale revisions are refused"},
}

func parseExtractConstant(raw map[string]any) (ExtractConstantReq, error) {
	var req ExtractConstantReq
	if err := CheckParams(raw, extractConstantParams); err != nil {
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
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, true); err != nil {
		return req, err
	}
	if req.ExpectedRevision, err = ParseString(raw, "expected_revision", "expected-revision", false); err != nil {
		return req, err
	}
	if value, present := rawLookup(raw, "expected_revision", "expected-revision"); present && value != nil && req.ExpectedRevision == "" {
		return req, fmt.Errorf("param %q must not be empty when supplied: %w", "expected_revision", ErrInvalidParams)
	}
	return req, nil
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

func extractConstantDef() Def[ExtractConstantReq, FileEditRes] {
	return Def[ExtractConstantReq, FileEditRes]{
		Key: "extract_constant", Summary: "Use this tool instead of copying an expression and replacing its uses to extract one exact Go constant expression into a package-level constant. The expression must have resolved compile-time type and value facts; untyped constants stay untyped, and supported typed basic constants retain their type. It refuses iota, local constant dependencies, comments inside the selection, collisions, duplicate matches, and unsupported types. On success, the returned file and symbol identify the new live declaration; pass them to semantic_rename or semantic_replace_decl. Select only an expression whose source text identifies one current occurrence." + automaticVerificationGuidance,
		Params: extractConstantParams, Level: LevelFile, CLIName: "extract-constant", MCPName: "semantic_extract_constant",
		Parse:    parseExtractConstant,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ExtractConstantReq) (FileEditRes, error){backend.LanguageGo: runExtractConstant},
		Format:   formatFileEdit, ExampleRaw: map[string]any{"file": wireExampleFile, "symbol": "retryWindow", wireInFunction: "Retry", wireDiscriminator: "3 * 60"},
		BufferedBatch: true, Batchable: true,
	}
}

// ExtractVariableReq selects one exact expression and names its new local.
type ExtractVariableReq struct {
	Project       backend.ProjectContext
	File          string
	Symbol        string
	InFunction    string
	Discriminator string
	ConstructPath string
}

// GetProjectContext returns the request project for registry dispatch.
func (r ExtractVariableReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ExtractVariableReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var extractVariableParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh simple local identifier to create; choose a new name not already visible in the owner", Required: true},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified owner function or method containing the selected call", Required: true, SourceFields: append(append([]string{}, sourceInFunctionContinuations...), "semantic_extract_variable.in_function")},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: exact source text of the first argument expression in a direct call to an unshadowed same-file package function", Required: true},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic; each path is the zero-based direct statement index in the owner body"},
}

func parseExtractVariable(raw map[string]any) (ExtractVariableReq, error) {
	var req ExtractVariableReq
	if err := CheckParams(raw, extractVariableParams); err != nil {
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
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, true); err != nil {
		return req, err
	}
	if req.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return req, err
	}
	return req, nil
}

func runExtractVariable(ctx context.Context, cc CallContext, req ExtractVariableReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractVariable(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractVariableOptions{
		Discriminator: req.Discriminator, ConstructPath: req.ConstructPath,
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

func extractVariableDef() Def[ExtractVariableReq, FileEditRes] {
	return Def[ExtractVariableReq, FileEditRes]{
		Key: "extract_variable", Summary: "Use this tool instead of copying expressions and replacing uses to extract one exact, side-effect-free typed expression from the first argument of a direct Go call into a preceding local. The call target must be an unshadowed same-file package function with no unresolved import binding; callable owner parameters/locals, builtins, and functions declared only in other files are refused. Later arguments retain their evaluation order. The owner must contain no comments. Expressions with calls, receives, address-taking, composite literals, only constants, or shifts with an untyped left operand are refused; expressions in returns, nested expressions, and receiver calls are unsupported. On success, pass the live file and in_function to semantic_inline_variable when its binding/use contract supports the result." + automaticVerificationGuidance,
		Params: extractVariableParams, Level: LevelFile, CLIName: "extract-variable", MCPName: "semantic_extract_variable",
		Parse:         parseExtractVariable,
		Handlers:      map[backend.LanguageID]func(context.Context, CallContext, ExtractVariableReq) (FileEditRes, error){backend.LanguageGo: runExtractVariable},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "value", wireInFunction: "PrintValue", wireDiscriminator: "count + 1"},
		BufferedBatch: true, Batchable: true,
	}
}
