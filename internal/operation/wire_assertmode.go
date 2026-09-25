// Package operation wires assertion failure-mode rewrites into the central operation registry.
package operation

import (
	"context"
	"fmt"
	"go/format"
	"os"

	"semedit/internal/assertmode"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
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
	File    string
	Swapped int
	DryRun  bool
}

var assertionModeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Go test file to rewrite", Required: true},
	{Name: "mode", CLIName: "mode", JSONName: "mode", Type: ParamString, Description: "Rewrite direction", Required: true, Enums: []string{"relax", "restrict"}},
	{Name: "dry_run", CLIName: "dry-run", JSONName: "dry_run", Type: ParamBoolean, Description: "Preview changes without writing the file", Default: false},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Type: ParamBoolean, Description: "Explicitly authorize writing this workspace", Default: false},
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
	req.Project = backend.ProjectContext{Language: backend.LanguageGo, WorkspaceTrust: backend.NewWorkspaceTrust(".", req.TrustWorkspace)}
	return req, nil
}

func runAssertionMode(_ context.Context, cc CallContext, req AssertionModeReq) (AssertionModeRes, error) {
	project := effectiveProject(cc, req.Project)
	if !req.DryRun && !project.WorkspaceTrust.Allows(project.RootDir) {
		return AssertionModeRes{}, &backend.WorkspaceTrustError{Operation: "assertion_mode", Language: backend.LanguageGo, Workspace: project.RootDir}
	}
	path := resolveWorkPath(cc.WorkDir, req.File)
	// #nosec G304 -- path is the operation's explicit workspace-relative file target.
	data, err := os.ReadFile(path)
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
		if err := pipeline.WriteAtomic(path, rewritten); err != nil {
			return AssertionModeRes{}, fmt.Errorf("write assertion source %s: %w", req.File, err)
		}
	}
	return AssertionModeRes{File: req.File, Swapped: result.Swapped, DryRun: req.DryRun}, nil
}

func assertionModeDef() Def[AssertionModeReq, AssertionModeRes] {
	return Def[AssertionModeReq, AssertionModeRes]{
		Key:      "assertion_mode",
		Summary:  "Use this tool instead of text replacing t.Fatal/t.Error calls when converting supported Go test assertions between fail-fast and continue-on-failure modes. Requires workspace trust before writes; use dry_run to preview.",
		Params:   assertionModeParams,
		Level:    LevelFile,
		CLIName:  "assertion-mode",
		MCPName:  "semantic_assertion_mode",
		Parse:    parseAssertionMode,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, AssertionModeReq) (AssertionModeRes, error){backend.LanguageGo: runAssertionMode},
		Format: func(res AssertionModeRes) (string, error) {
			if res.DryRun {
				return fmt.Sprintf("%d swapped (dry run)", res.Swapped), nil
			}
			return fmt.Sprintf("%d swapped; wrote %s", res.Swapped, res.File), nil
		},
		ExampleRaw: map[string]any{"file": "api/api_test.go", "mode": "relax", "dry_run": true},
		Batchable:  true,
	}
}
