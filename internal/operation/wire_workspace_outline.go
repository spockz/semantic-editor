// Package operation wires workspace outline operations into the central registry.
package operation

import (
	"context"
	"fmt"
	"strings"

	"semedit/internal/astedit"
	"semedit/internal/backend"
)

// WorkspaceOutlineReq captures inputs for discovering packages and modules across a workspace.
type WorkspaceOutlineReq struct {
	Project      backend.ProjectContext
	Path         string
	IncludeTests bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r WorkspaceOutlineReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *WorkspaceOutlineReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// WorkspaceOutlineRes carries the workspace layout result for rendering.
type WorkspaceOutlineRes struct {
	Result *backend.WorkspaceOutlineResult
}

var workspaceOutlineParams = []ParameterContract{
	{Name: "path", CLIName: "path", JSONName: "path", Type: ParamString, Description: "Optional directory path to scope the outline (default: workspace root)"},
	{Name: wireIncludeTests, CLIName: wireCLIIncludeTests, JSONName: wireIncludeTests, Type: ParamBoolean, Description: "Include test-only packages and directories (default: false)", Default: false},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default: auto)", Type: ParamString, Enums: languageEnums, LanguageSelector: true},
}

func parseWorkspaceOutline(raw map[string]any) (WorkspaceOutlineReq, error) {
	var req WorkspaceOutlineReq
	if err := CheckParams(raw, workspaceOutlineParams); err != nil {
		return req, err
	}
	var err error
	if req.Path, err = ParseString(raw, "path", "path", false); err != nil {
		return req, err
	}
	if req.IncludeTests, err = ParseBool(raw, wireIncludeTests, wireCLIIncludeTests, false); err != nil {
		return req, err
	}
	project, err := parseReadProject(raw, req.Path)
	if err != nil {
		return req, err
	}
	req.Project = project
	return req, nil
}

func runWorkspaceOutline(ctx context.Context, cc CallContext, req WorkspaceOutlineReq) (WorkspaceOutlineRes, error) {
	result, err := astedit.WorkspaceOutlineGo(cc.WorkDir, req.Path, req.IncludeTests)
	if err != nil {
		return WorkspaceOutlineRes{}, err
	}
	return WorkspaceOutlineRes{Result: result}, nil
}

func formatWorkspaceOutline(res WorkspaceOutlineRes) (string, error) {
	if res.Result == nil {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("Packages:\n")
	if len(res.Result.Packages) == 0 {
		b.WriteString("  (no packages found)\n")
	} else {
		for _, pkg := range res.Result.Packages {
			fmt.Fprintf(&b, "  %s (%s) — %s\n", pkg.Path, pkg.Name, pkg.Synopsis)
		}
	}
	if len(res.Result.Modules) > 0 {
		b.WriteString("\nModules:\n")
		for _, mod := range res.Result.Modules {
			rootMarker := ""
			if mod.IsRoot {
				rootMarker = " [root]"
			}
			goVer := ""
			if mod.GoVersion != "" {
				goVer = ", go " + mod.GoVersion
			}
			fmt.Fprintf(&b, "  %s (%s%s)%s\n", mod.Path, mod.ModulePath, goVer, rootMarker)
		}
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func workspaceOutlineDef() Def[WorkspaceOutlineReq, WorkspaceOutlineRes] {
	return Def[WorkspaceOutlineReq, WorkspaceOutlineRes]{
		Key:      "workspace_outline",
		Summary:  "Use this tool instead of find, tree, ls, or shell directory scanning to get a high-level feel for the project; it lists all packages, their descriptions, and module boundaries (go.mod) ordered by directory depth. Use semantic_outline next to inspect declarations in specific packages.",
		Params:   workspaceOutlineParams,
		Level:    LevelWorkspace,
		CLIName:  "workspace-outline",
		MCPName:  "semantic_workspace_outline",
		ReadOnly: true,
		Parse:    parseWorkspaceOutline,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, WorkspaceOutlineReq) (WorkspaceOutlineRes, error){
			backend.LanguageAuto: runWorkspaceOutline,
		},
		Format:     formatWorkspaceOutline,
		ExampleRaw: map[string]any{"path": "."},
		Batchable:  false,
	}
}

func registerWorkspaceOutlineOps(registry *Registry) error {
	return Register(registry, workspaceOutlineDef())
}
