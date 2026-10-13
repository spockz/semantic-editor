// Package operation declares retained MCP staging controls separately from source mutations.
package operation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"semedit/internal/backend"
)

// StageLimits are immutable server policy limits reported with each receipt.
type StageLimits struct {
	Steps            int           `json:"steps"`
	RequestBytes     int64         `json:"request_bytes"`
	DirtySourceBytes int64         `json:"dirty_source_bytes"`
	MaxAge           time.Duration `json:"max_age"`
}

// StageReceipt describes the accepted volatile revision without exposing child continuations.
type StageReceipt struct {
	Status           string      `json:"status"`
	StageID          string      `json:"stage_id"`
	StepID           string      `json:"step_id,omitempty"`
	Steps            []StageStep `json:"steps"`
	StepCount        int         `json:"step_count"`
	RequestBytes     int64       `json:"request_bytes"`
	DirtySourceBytes int64       `json:"dirty_source_bytes"`
	Limits           StageLimits `json:"limits"`
}

// StageStep identifies one remaining stable step in a retained staging session.
type StageStep struct {
	StepID string `json:"step_id"`
}

// StageCommitRow preserves exact evidence for one accepted stage step.
type StageCommitRow struct {
	StepID string `json:"step_id"`
	Tool   string `json:"tool"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// StageDiagnosticCandidate identifies steps whose changed paths may relate to a diagnostic.
type StageDiagnosticCandidate struct {
	Diagnostic       string   `json:"diagnostic"`
	CandidateStepIDs []string `json:"candidate_step_ids"`
}

// StageDiagnosticDelta reports diagnostics observed after retained publication.
type StageDiagnosticDelta struct {
	Introduced []string                   `json:"introduced"`
	Resolved   []string                   `json:"resolved"`
	NetDelta   int                        `json:"net_delta"`
	Candidates []StageDiagnosticCandidate `json:"candidates"`
}

// StageCommitResult reports final publication and one verification boundary.
type StageCommitResult struct {
	Status           string                `json:"status"`
	ApplicationState string                `json:"application_state,omitempty"`
	WorkspaceRoot    string                `json:"workspace_root"`
	Results          []StageCommitRow      `json:"results"`
	PublishedFiles   int                   `json:"published_files"`
	PublicationError string                `json:"publication_error,omitempty"`
	DiagnosticDelta  *StageDiagnosticDelta `json:"diagnostic_delta,omitempty"`
	Verification     *VerifyRes            `json:"verification,omitempty"`
	FinalDiff        string                `json:"final_diff"`
}

// StageCommitError preserves the completed publication outcome when final staging cannot report success.
type StageCommitError struct {
	Result StageCommitResult
	Err    error
}

// Error reports the final stage lifecycle failure.
func (e *StageCommitError) Error() string {
	if e == nil || e.Err == nil {
		return "staged commit failed"
	}
	return e.Err.Error()
}

// Unwrap returns the underlying publication or verification failure.
func (e *StageCommitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type stageStartReq struct{}
type stageDropReq struct{ StepID string }
type stageAbortReq struct{}
type stageCommitReq struct {
	TrustWorkspace bool
	SkipHooks      bool
}

func stageRuntime(cc CallContext) (StageRuntime, error) {
	if cc.StageRuntime == nil {
		return nil, fmt.Errorf("staging requires a retained MCP session; use semedit mcp")
	}
	return cc.StageRuntime, nil
}
func stageCommitSchema() *ValueSchema {
	text := ValueSchema{Kind: SchemaString}
	integer := ValueSchema{Kind: SchemaInteger}
	diagnostic := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"diagnostic":         text,
		"candidate_step_ids": {Kind: SchemaArray, Items: &text},
	}, Required: []string{"diagnostic", "candidate_step_ids"}}
	delta := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		introducedField: {Kind: SchemaArray, Items: &text},
		"resolved":      {Kind: SchemaArray, Items: &text},
		"net_delta":     integer,
		"candidates":    {Kind: SchemaArray, Items: &diagnostic},
	}, Required: []string{introducedField, "resolved", "net_delta", "candidates"}}
	row := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"step_id": text, "tool": text, "status": text, "error": text}, Required: []string{"step_id", "tool", "status"}}
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"status":            text,
		"workspace_root":    text,
		"results":           {Kind: SchemaArray, Items: &row},
		"published_files":   integer,
		"publication_error": text,
		"diagnostic_delta":  delta,
		"verification":      *NativeVerifyOutputSchema(),
		"final_diff":        text,
	}, Required: []string{"status", "workspace_root", "results", "published_files", "final_diff"}}
}

func stageCommitErrorSchema() *ValueSchema {
	schema := stageCommitSchema()
	schema.Properties["status"] = ValueSchema{Kind: SchemaString, Enum: []any{"failed", "failed_partial", "applied_with_diagnostics"}}
	schema.Properties["application_state"] = ValueSchema{Kind: SchemaString, Enum: []any{"not_applied", "partially_published", "persisted"}}
	schema.Required = append(schema.Required, "application_state")
	return schema
}

func projectStageCommitError(err error) (any, bool, error) {
	var failure *StageCommitError
	if !errors.As(err, &failure) {
		return nil, false, nil
	}
	return failure.Result, true, nil
}

func registerStageOps(r *Registry) error {
	start := stageStatelessControl("stage_start", ("Use this tool instead of applying Go or Java edits immediately when a retained sequence needs review before publication. Accepted edits remain in an in-memory overlay until semantic_commit; no workspace files are written by this control."), "stage", "semantic_stage", func(raw map[string]any) (stageStartReq, error) {
		return stageStartReq{}, CheckParams(raw, nil)
	}, func(rt StageRuntime, ctx context.Context, cc CallContext, _ stageStartReq) (StageReceipt, error) {
		return rt.Start(ctx, cc.WorkDir)
	})
	if err := Register(r, start); err != nil {
		return err
	}
	dropParams := []ParameterContract{{Name: "step_id", CLIName: "step-id", JSONName: "step_id", Type: ParamString, Description: "Opaque step ID from a prior stage receipt. Dropping it rebuilds the retained overlay without writing workspace files.", Required: true, Role: RoleSelector, SourceFields: StageDropSourceFields(r.All())}}
	if err := Register(r, Def[stageDropReq, StageReceipt]{Key: "stage_drop", Summary: ("Use this tool instead of restarting the entire staging session when removing one accepted step. The surviving steps are replayed into a new in-memory overlay and no workspace files are written."), Level: LevelWorkspace, CLIName: "stage-drop", MCPName: "semantic_stage_drop", Params: dropParams, Parse: func(raw map[string]any) (stageDropReq, error) {
		if e := CheckParams(raw, dropParams); e != nil {
			return stageDropReq{}, e
		}
		v, e := ParseString(raw, "step_id", "step-id", true)
		return stageDropReq{v}, e
	}, Handlers: map[backend.LanguageID]func(context.Context, CallContext, stageDropReq) (StageReceipt, error){backend.LanguageGo: func(ctx context.Context, cc CallContext, q stageDropReq) (StageReceipt, error) {
		rt, e := stageRuntime(cc)
		if e != nil {
			return StageReceipt{}, e
		}
		return rt.Drop(ctx, cc.WorkDir, q.StepID)
	}, backend.LanguageJava: func(ctx context.Context, cc CallContext, q stageDropReq) (StageReceipt, error) {
		rt, e := stageRuntime(cc)
		if e != nil {
			return StageReceipt{}, e
		}
		return rt.Drop(ctx, cc.WorkDir, q.StepID)
	}}, Format: func(v StageReceipt) (string, error) { return v.Status, nil }, ProjectOutput: func(v StageReceipt) (any, error) { return v, nil }, Output: &OperationOutputContract{ResultSchema: StageReceiptSchema(), Continuations: StageDropContinuations()}, ExampleRaw: map[string]any{"step_id": "00112233445566778899aabbccddeeff"}}); err != nil {
		return err
	}
	abort := stageStatelessControl("stage_abort", ("Use this tool instead of undo when discarding an uncommitted retained Go or Java staging session. This removes its in-memory overlay and writes no workspace files."), "stage-abort", "semantic_stage_abort", func(raw map[string]any) (stageAbortReq, error) {
		return stageAbortReq{}, CheckParams(raw, nil)
	}, func(rt StageRuntime, ctx context.Context, cc CallContext, _ stageAbortReq) (StageReceipt, error) {
		return rt.Abort(ctx, cc.WorkDir)
	})
	if err := Register(r, abort); err != nil {
		return err
	}
	commitParams := []ParameterContract{
		{Name: "trust_workspace", CLIName: "trust-workspace", JSONName: "trust_workspace", Type: ParamBoolean, Description: "Trust configured verification hooks for the active workspace during final verification.", Default: false},
		{Name: wireSkipHooks, CLIName: "skip-hooks", JSONName: wireSkipHooks, Type: ParamBoolean, Description: "Skip configured verification hooks after publication; formatting and diagnostic synchronization still run.", Default: false},
	}
	return Register(r, Def[stageCommitReq, StageCommitResult]{Key: "stage_commit", Summary: ("Use this tool instead of semantic_batch when publishing an existing retained Go or Java staging session. It normalizes the staged edits and reports zero, partial, or full publication and the final verification outcome."), Level: LevelWorkspace, CLIName: "commit", MCPName: "semantic_commit", Params: commitParams, Parse: func(raw map[string]any) (stageCommitReq, error) {
		if e := CheckParams(raw, commitParams); e != nil {
			return stageCommitReq{}, e
		}
		trusted, e := ParseBool(raw, "trust_workspace", "trust-workspace", false)
		if e != nil {
			return stageCommitReq{}, e
		}
		skipHooks, e := ParseBool(raw, wireSkipHooks, "skip-hooks", false)
		return stageCommitReq{TrustWorkspace: trusted, SkipHooks: skipHooks}, e
	}, Handlers: map[backend.LanguageID]func(context.Context, CallContext, stageCommitReq) (StageCommitResult, error){backend.LanguageGo: func(ctx context.Context, cc CallContext, q stageCommitReq) (StageCommitResult, error) {
		rt, e := stageRuntime(cc)
		if e != nil {
			return StageCommitResult{}, e
		}
		return rt.Commit(ctx, cc.WorkDir, q.TrustWorkspace || cc.Project.WorkspaceTrust.Trusted, q.SkipHooks)
	}, backend.LanguageJava: func(ctx context.Context, cc CallContext, q stageCommitReq) (StageCommitResult, error) {
		rt, e := stageRuntime(cc)
		if e != nil {
			return StageCommitResult{}, e
		}
		return rt.Commit(ctx, cc.WorkDir, q.TrustWorkspace || cc.Project.WorkspaceTrust.Trusted, q.SkipHooks)
	}}, Format: func(v StageCommitResult) (string, error) { return v.Status, nil }, ProjectOutput: func(v StageCommitResult) (any, error) { return v, nil }, Output: &OperationOutputContract{ResultSchema: stageCommitSchema()}, ErrorOutput: &OperationErrorOutputContract{ResultSchema: stageCommitErrorSchema()}, ProjectErrorOutput: projectStageCommitError, ExampleRaw: map[string]any{"trust_workspace": false, wireSkipHooks: false}})
}

func stageStatelessControl[Request any](key, summary, cliName, mcpName string, parse func(map[string]any) (Request, error), invoke func(StageRuntime, context.Context, CallContext, Request) (StageReceipt, error)) Def[Request, StageReceipt] {
	return Def[Request, StageReceipt]{
		Key: key, Summary: summary, Level: LevelWorkspace, CLIName: cliName, MCPName: mcpName, Parse: parse,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, Request) (StageReceipt, error){backend.LanguageGo: func(ctx context.Context, cc CallContext, request Request) (StageReceipt, error) {
			runtime, err := stageRuntime(cc)
			if err != nil {
				return StageReceipt{}, err
			}
			return invoke(runtime, ctx, cc, request)
		}, backend.LanguageJava: func(ctx context.Context, cc CallContext, request Request) (StageReceipt, error) {
			runtime, err := stageRuntime(cc)
			if err != nil {
				return StageReceipt{}, err
			}
			return invoke(runtime, ctx, cc, request)
		}},
		Format:        func(v StageReceipt) (string, error) { return v.Status, nil },
		ProjectOutput: func(v StageReceipt) (any, error) { return v, nil },
		Output:        &OperationOutputContract{ResultSchema: StageReceiptSchema()},
		ExampleRaw:    map[string]any{},
	}
}
