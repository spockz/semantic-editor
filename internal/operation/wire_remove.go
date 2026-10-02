// This file isolates the exact-node removal request schema and handler from replacement wire contracts.

package operation

import (
	"context"
	"errors"
	"fmt"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"strings"
)

// RemoveReq selects a Go declaration or case for exact structural removal.
type RemoveReq struct {
	Project             backend.ProjectContext
	File                string
	Kind                string
	Symbol              string
	InFunction          string
	Discriminator       string
	TargetCase          string
	ConstructPath       string
	SafeDelete          bool
	AutoOrganizeImports bool
}

// SafeDeleteRefusalError reports an incomplete or failed preflight as an operation error.
type SafeDeleteRefusalError struct {
	File          string
	Symbol        string
	InFunction    string
	Discriminator string
	TargetCase    string
	ConstructPath string
	ChecksRun     []string
	Reason        string
}

// Error describes why safe deletion refused to mutate the workspace.
func (e *SafeDeleteRefusalError) Error() string {
	return fmt.Sprintf("safe_delete refused: %s (checks run: %s)", e.Reason, strings.Join(e.ChecksRun, ", "))
}

// RemoveRes reports the removal outcome, checks, diagnostics, and continuation selectors.
type RemoveRes struct {
	File             string                   `json:"file,omitempty"`
	Display          string                   `json:"-"`
	Symbol           string                   `json:"symbol,omitempty"`
	InFunction       string                   `json:"in_function,omitempty"`
	Discriminator    string                   `json:"discriminator,omitempty"`
	TargetCase       string                   `json:"target_case,omitempty"`
	ConstructPath    string                   `json:"construct_path,omitempty"`
	SafeDelete       bool                     `json:"safe_delete"`
	Status           string                   `json:"status"`
	ChecksRun        []string                 `json:"checks_run,omitempty"`
	Refusal          string                   `json:"refusal,omitempty"`
	Diff             string                   `json:"diff,omitempty"`
	DiagnosticStatus string                   `json:"diagnostic_status,omitempty"`
	Delta            pipeline.DiagnosticDelta `json:"diagnostic_delta"`
	HasDelta         bool                     `json:"-"`
}

// GetProjectContext returns the request project for registry dispatch.
func (r RemoveReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *RemoveReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var (
	removeFileSources          = sourceFileContinuations
	removeSymbolSources        = sourceSymbolContinuations
	removeDiscriminatorSources = sourceDiscriminatorContinuations
	removeParams               = []ParameterContract{
		{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: removeFileSources},
		{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Go node kind to remove: function, method, type, constant, variable, or case", Required: true, Enums: []string{"function", "method", "type", "constant", "variable", "case"}},
		{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: qualified top-level declaration to remove; required for function, method, type, constant, or variable, and never an owner selector", SourceFields: removeSymbolSources},
		{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: owner function or method containing the case target; required only for kind=case", SourceFields: sourceInFunctionContinuations},
		{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: switch discriminant expression or type-switch assignment to narrow case selection", SourceFields: removeDiscriminatorSources},
		{Name: "target_case", CLIName: "target-case", JSONName: "target_case", Type: ParamString, Description: "Selector: case value to remove (for example nil or \"foo\"); use default for a default clause"},
		{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic (for example 0 or 0.1)"},
		{Name: "safe_delete", CLIName: "safe-delete", JSONName: "safe_delete", Type: ParamBoolean, Description: "Request reference and resource preflight. Safe deletion currently supports only unexported standalone constants after a complete Go source scan; other requests fail closed and return checks run.", Default: false},
		{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically remove imports made unused by this edit (default true)", Default: true},
	}
)

func parseRemove(raw map[string]any) (RemoveReq, error) {
	var req RemoveReq
	if err := CheckParams(raw, removeParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Kind, err = ParseString(raw, "kind", "kind", true); err != nil {
		return req, err
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", false); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, false); err != nil {
		return req, err
	}
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.TargetCase, err = ParseString(raw, "target_case", "target-case", false); err != nil {
		return req, err
	}
	if req.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return req, err
	}
	if req.SafeDelete, err = ParseBool(raw, "safe_delete", "safe-delete", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	switch req.Kind {
	case "function", "method", "type", "constant", "variable":
		if req.Symbol == "" {
			return req, fmt.Errorf("symbol is required for kind=%s", req.Kind)
		}
		if req.InFunction != "" || req.Discriminator != "" || req.TargetCase != "" || req.ConstructPath != "" {
			return req, fmt.Errorf("case selectors are only valid for kind=case")
		}
	case "case":
		if req.InFunction == "" {
			return req, fmt.Errorf("in_function is required for kind=case")
		}
		if req.Symbol != "" {
			return req, fmt.Errorf("symbol is a declaration target and cannot select a case; use in_function and target_case")
		}
	default:
		return req, fmt.Errorf("unsupported removal kind %q", req.Kind)
	}
	return req, nil
}

func runRemove(ctx context.Context, cc CallContext, req RemoveReq) (RemoveRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, checks, refusal, err := astedit.Remove(ctx, targetPath, astedit.RemoveOptions{
		Kind: req.Kind, Symbol: req.Symbol, InFunction: req.InFunction, Discriminator: req.Discriminator,
		TargetCase: req.TargetCase, ConstructPath: req.ConstructPath, SafeDelete: req.SafeDelete,
	}, effectiveAutoOrganize(cc, req.AutoOrganizeImports))
	if err != nil {
		if snapshotErr, ok := errors.AsType[*astedit.SafeDeleteSnapshotError](err); ok {
			return RemoveRes{}, &SafeDeleteRefusalError{
				File: req.File, Symbol: req.Symbol, InFunction: req.InFunction,
				Discriminator: req.Discriminator, TargetCase: req.TargetCase,
				ConstructPath: req.ConstructPath, ChecksRun: checks, Reason: snapshotErr.Error(),
			}
		}
		return RemoveRes{}, err
	}
	if refusal != "" {
		return RemoveRes{}, &SafeDeleteRefusalError{
			File: req.File, Symbol: req.Symbol, InFunction: req.InFunction,
			Discriminator: req.Discriminator, TargetCase: req.TargetCase,
			ConstructPath: req.ConstructPath, ChecksRun: checks, Reason: refusal,
		}
	}
	delta, diagnosticStatus := finishDelta()
	status := "removed"
	return RemoveRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Discriminator: req.Discriminator, TargetCase: req.TargetCase,
		ConstructPath: req.ConstructPath, SafeDelete: req.SafeDelete, Status: status,
		ChecksRun: checks, Refusal: refusal, Diff: diff, DiagnosticStatus: diagnosticStatus,
		Delta: delta, HasDelta: true,
	}, nil
}

func removeDef() Def[RemoveReq, RemoveRes] {
	return Def[RemoveReq, RemoveRes]{
		Key:      "remove_construct",
		Summary:  "Use this tool instead of replace_file_content to remove exactly one Go top-level declaration or switch case. Top-level kinds are function, method, type, constant, and variable, selected by symbol; case uses in_function plus optional discriminator, target_case, and construct_path selectors. Syntax is validated and diagnostics are reported. Structural removal does not prove behavioral equivalence. safe_delete=true can remove only unexported standalone constants after scanning all Go-loader source directories across build-tag variants; other requests fail closed and return checks run.",
		Params:   removeParams,
		Level:    LevelFile,
		CLIName:  "remove-construct",
		MCPName:  "semantic_remove_construct",
		Parse:    parseRemove,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, RemoveReq) (RemoveRes, error){backend.LanguageGo: runRemove},
		Format: func(res RemoveRes) (string, error) {
			if res.Status == "refused" {
				return fmt.Sprintf("Safe deletion refused for %s: %s (checks run: %s).", res.Display, res.Refusal, strings.Join(res.ChecksRun, ", ")), nil
			}
			text := fmt.Sprintf("Removed selected node from %s.\n%s", res.Display, res.Diff)
			if res.SafeDelete {
				text += fmt.Sprintf("\nSafe deletion checks run: %s.", strings.Join(res.ChecksRun, ", "))
			}
			return AppendDiagnosticDelta(text, res.Delta), nil
		},
		ExampleRaw:    map[string]any{"file": wireExampleFile, "kind": "function", "symbol": "obsoleteHelper"},
		BufferedBatch: true, Batchable: true,
	}
}
