// Package operation wires local Go refactorings into the shared CLI and MCP registry.
package operation

import (
	"context"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
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

var inlineVariableParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: source file containing the selected local", Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Description: "Target: live local binding in the selected owner; copy a compatible local identity from semantic_lookup. The binding is removed on success.", Required: true, SourceFields: []string{sourceLookupSymbol}},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified function or method containing the local binding", Required: true, SourceFields: sourceInFunctionContinuations},
}, readProjectParams...)

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
		Detail: ("Inlined local value."), Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func inlineVariableDef() Def[InlineVariableReq, ExtractionResult] {
	return Def[InlineVariableReq, ExtractionResult]{
		Key:     (canonicalInlineValueKey),
		Summary: ("Use this tool instead of text replacement to inline one local variable. Go supports basic literals at proven same-block direct uses, plus a pure scalar expression over local values/constants or a keyed scalar-only same-package named struct literal at its sole immediate direct return after resolved binding and type checks; ADR-0105 also permits a pure scalar expression at the sole argument of an immediate direct call statement to a resolved function or builtin, ADR-0113 permits its complete RHS use in the immediately following one-target simple assignment to a distinct existing identifier, ADR-0114 permits its complete RHS use in the immediately following single-name short declaration, ADR-0119 permits its complete initializer use in the immediately following ungrouped local var declaration with one name and one value, ADR-0127 permits its complete pure scalar expression use as an immediate if condition with no init statement, ADR-0128 permits its complete pure scalar expression use as an immediate expression switch tag with no init statement, ADR-0129 permits its sole argument use in an immediately following direct defer call to a resolved function or builtin while preserving argument evaluation at defer registration, and ADR-0130 permits its sole argument use in an immediately following direct go call to a resolved function or builtin while preserving argument evaluation before goroutine start. Candidate checks preserve target identity, inferred expression type, and initializer inputs. Java delegates to JDT LS refactor.inline; Rust delegates only a proven suffixed-integer literal substitution to rust-analyzer. The successful edit removes the binding and returns its surviving owner as a live target; use semantic_lookup to select fresh constructs or bindings. The removed name is historical evidence and is not a follow-up target. Effectful expressions, member/index access, mutable package-variable inputs, non-immediate computed uses, compound/tuple/selector/index assignments, grouped or multi-name/multi-value var declarations, multi-name short declarations, reassignment, unproven shadowed inputs, and unsupported server actions are refused."),
		Params:  inlineVariableParams, Level: LevelFile, CLIName: "refactor-inline-value", MCPName: semanticInlineVariableToolName,
		Parse: parseInlineVariable,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InlineVariableReq) (ExtractionResult, error){
			backend.LanguageGo:   runInlineVariableWithReceipt,
			backend.LanguageJava: runInlineVariableJavaWithReceipt,
			backend.LanguageRust: runInlineVariableRustWithReceipt,
		},
		Format: formatExtractionResult,
		Output: extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName),
		FinalizeOutput: func(cc CallContext, req InlineVariableReq, result ExtractionResult) (ExtractionResult, error) {
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractionOutput(cc, ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol, InFunction: req.InFunction}, result)
			}
			if req.Project.Language == backend.LanguageRust {
				return finalizeRustExtractionOutput(cc, ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol, InFunction: req.InFunction}, result)
			}
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "limit", wireInFunction: "Server.Stop"},
		BufferedBatch: true, Batchable: true,
	}
}
