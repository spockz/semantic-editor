// Package operation wires assertion failure-mode rewrites into the central operation registry.
package operation

import (
	"context"
	"fmt"
	"go/format"

	"semedit/internal/assertmode"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
)

// AssertionModeReq requests a safe fail-fast or continue-on-failure assertion rewrite.
type AssertionModeReq struct {
	Project        backend.ProjectContext
	File           string
	Mode           string
	DryRun         bool
	TrustWorkspace bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r AssertionModeReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *AssertionModeReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// AssertionModeRes reports the mechanical changes considered or written.
type AssertionModeRes struct {
	File         string
	Swapped      int
	DryRun       bool
	WorkingView  bool
	ReceiptDraft EditReceiptDraft
	Receipt      EditReceipt
}

var assertionModeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Go test file to rewrite; path is relative to the workspace root", Required: true, SourceFields: sourceFileContinuations},
	{Name: "mode", CLIName: "mode", JSONName: "mode", Type: ParamString, Description: "Rewrite direction", Required: true, Enums: []string{"relax", "restrict"}},
	{Name: "dry_run", CLIName: "dry-run", JSONName: "dry_run", Type: ParamBoolean, Description: "Preview the assertion rewrite without writing; trust_workspace is not required when true", Default: false},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Type: ParamBoolean, Description: "Must be true for a non-dry-run request that writes this workspace; consent applies to this request only", Default: false},
}

func parseAssertionMode(raw map[string]any) (AssertionModeReq, error) {
	var req AssertionModeReq
	if err := CheckParams(raw, assertionModeParams); err != nil {
		return req, err
	}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Mode, err = ParseEnum(raw, "mode", "mode", []string{"relax", "restrict"}, true, ""); err != nil {
		return req, err
	}
	if req.DryRun, err = ParseBool(raw, "dry_run", "dry-run", false); err != nil {
		return req, err
	}
	if req.TrustWorkspace, err = ParseBool(raw, wireTrustWorkspace, wireCLITrustWorkspace, false); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo, WorkspaceTrust: backend.WorkspaceTrust{Trusted: req.TrustWorkspace}}
	return req, nil
}

func assertionModeDef() Def[AssertionModeReq, AssertionModeRes] {
	return Def[AssertionModeReq, AssertionModeRes]{
		Key:      "assertion_mode",
		Summary:  "Use this tool instead of text replacing t.Fatal/t.Error calls when converting supported Go test assertions between fail-fast and continue-on-failure modes. Set trust_workspace=true before a write; set dry_run=true to preview without trust or file writes.",
		Params:   assertionModeParams,
		Level:    LevelFile,
		CLIName:  "assertion-mode",
		MCPName:  semanticAssertionModeToolName,
		Parse:    parseAssertionMode,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, AssertionModeReq) (AssertionModeRes, error){backend.LanguageGo: bufferedAssertionModeHandler},
		Format: func(res AssertionModeRes) (string, error) {
			if res.DryRun {
				return fmt.Sprintf("%d swapped (dry run)", res.Swapped), nil
			}
			if res.Swapped == 0 {
				return fmt.Sprintf("No assertions changed in %s.", res.File), nil
			}
			return fmt.Sprintf("%d swapped; wrote %s", res.Swapped, res.File), nil
		},
		Output: &OperationOutputContract{ResultSchema: assertionModeOutputSchema()},
		FinalizeOutput: func(cc CallContext, _ AssertionModeReq, result AssertionModeRes) (AssertionModeRes, error) {
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			receipt, err := FinalizeEditReceipt(ctx, cc.WorkDir, result.WorkingView, cc.InBatch || result.WorkingView, result.ReceiptDraft, nil)
			if err != nil {
				return AssertionModeRes{}, err
			}
			result.Receipt = receipt
			return result, nil
		},
		ProjectOutput: projectAssertionModeOutput,
		ExampleRaw:    map[string]any{"file": "api/api_test.go", "mode": "relax", "dry_run": true},
		BufferedBatch: true, Batchable: true,
	}
}

func bufferedAssertionModeHandler(ctx context.Context, cc CallContext, req AssertionModeReq) (AssertionModeRes, error) {
	if req.DryRun {
		result, err := runAssertionMode(ctx, cc, req)
		if err != nil {
			return AssertionModeRes{}, err
		}
		result.ReceiptDraft = EditReceiptDraft{Operation: "assertion_mode", Status: EditNoOp}
		return result, nil
	}
	target := resolveWorkPath(cc.WorkDir, req.File)
	root, err := structuralModuleRoot(target)
	if err != nil {
		return AssertionModeRes{}, fmt.Errorf("prepare assertion source buffer: %w", err)
	}
	var result AssertionModeRes
	err = sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		buffer := sourcebuf.FromContext(editCtx)
		before := buffer.Checkpoint()
		var editErr error
		result, editErr = runAssertionMode(editCtx, cc, req)
		if editErr != nil {
			return editErr
		}
		result.WorkingView = sourcebuf.FromContext(cc.Ctx) != nil
		fileEdit := FileEditRes{Display: req.File, DiagnosticStatus: diagnosticStatusUnavailable}
		result.ReceiptDraft, editErr = bufferedReceiptDraft(buffer, before, cc, "assertion_mode", fileEdit)
		return editErr
	})
	return result, err
}

func runAssertionMode(ctx context.Context, cc CallContext, req AssertionModeReq) (AssertionModeRes, error) {
	project := effectiveProject(cc, req.Project)
	if !req.DryRun && !project.WorkspaceTrust.Allows(project.RootDir) {
		return AssertionModeRes{}, &backend.WorkspaceTrustError{Operation: "assertion_mode", Language: backend.LanguageGo, Workspace: project.RootDir}
	}
	path := resolveWorkPath(cc.WorkDir, req.File)
	// #nosec G304 -- path is the operation's explicit workspace-relative file target.
	data, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return AssertionModeRes{}, fmt.Errorf("read assertion source %s: %w", req.File, err)
	}
	formatted, err := format.Source(data)
	if err != nil {
		return AssertionModeRes{}, fmt.Errorf("format assertion source %s: %w", req.File, err)
	}
	rewritten, result, err := assertmode.Apply(path, formatted, assertmode.Mode(req.Mode))
	if err != nil {
		return AssertionModeRes{}, err
	}
	if !req.DryRun && result.Swapped > 0 {
		if err := pipeline.WriteAtomicContext(ctx, path, rewritten); err != nil {
			return AssertionModeRes{}, fmt.Errorf("write assertion source %s: %w", req.File, err)
		}
	}
	return AssertionModeRes{File: req.File, Swapped: result.Swapped, DryRun: req.DryRun}, nil
}
