// Package operation wires workspace outline operations into the central registry.
package operation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	javabackend "semedit/internal/backend/java"
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
	project := effectiveProject(cc, req.Project)
	requestedLanguage := req.Project.Language
	if requestedLanguage == "" {
		requestedLanguage = cc.Project.Language
	}
	if requestedLanguage == backend.LanguageGo {
		result, err := astedit.WorkspaceOutlineGo(project.RootDir, req.Path, req.IncludeTests)
		if err != nil {
			return WorkspaceOutlineRes{}, err
		}
		return WorkspaceOutlineRes{Result: result}, nil
	}
	if requestedLanguage == backend.LanguageJava {
		result, err := javabackend.WorkspaceOutline(project.RootDir, req.Path, req.IncludeTests)
		if err != nil {
			return WorkspaceOutlineRes{}, err
		}
		return WorkspaceOutlineRes{Result: result}, nil
	}

	isGo, err := astedit.LooksLikeGoWorkspace(project.RootDir, req.Path)
	if err != nil {
		return WorkspaceOutlineRes{}, fmt.Errorf("detect Go workspace outline: %w", err)
	}
	combined := &backend.WorkspaceOutlineResult{
		Scope: backend.ReadScope{
			Kind:     "workspace",
			Path:     req.Path,
			Complete: true,
		},
		Modules:  make([]backend.WorkspaceModule, 0),
		Packages: make([]backend.WorkspacePackage, 0),
	}
	if combined.Scope.Path == "" {
		combined.Scope.Path = "."
	}
	if isGo {
		goResult, outlineErr := astedit.WorkspaceOutlineGo(project.RootDir, req.Path, req.IncludeTests)
		if outlineErr != nil {
			return WorkspaceOutlineRes{}, fmt.Errorf("discover Go workspace outline: %w", outlineErr)
		}
		if goResult != nil {
			combined.Scope = goResult.Scope
			combined.Modules = append(combined.Modules, goResult.Modules...)
			combined.Packages = append(combined.Packages, goResult.Packages...)
		}
	}
	isJava, err := javabackend.LooksLikeJavaWorkspace(project.RootDir, req.Path)
	if err != nil {
		return WorkspaceOutlineRes{}, fmt.Errorf("detect Java workspace outline: %w", err)
	}
	if isJava {
		javaResult, outlineErr := javabackend.WorkspaceOutline(project.RootDir, req.Path, req.IncludeTests)
		if outlineErr != nil {
			return WorkspaceOutlineRes{}, fmt.Errorf("discover Java workspace outline: %w", outlineErr)
		}
		if javaResult != nil {
			if !isGo {
				combined.Scope = javaResult.Scope
			}
			combined.Modules = append(combined.Modules, javaResult.Modules...)
			combined.Packages = append(combined.Packages, javaResult.Packages...)
		}
	}
	slices.SortFunc(combined.Modules, func(a, b backend.WorkspaceModule) int {
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		return strings.Compare(a.Language, b.Language)
	})
	slices.SortFunc(combined.Packages, func(a, b backend.WorkspacePackage) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		return strings.Compare(a.Language, b.Language)
	})
	return WorkspaceOutlineRes{Result: combined}, nil
}

func formatWorkspaceOutline(res WorkspaceOutlineRes) (string, error) {
	if res.Result == nil {
		return "", nil
	}
	var b strings.Builder
	polyglot := len(workspaceOutlineLanguages(res.Result)) > 1
	writeWorkspacePackageLines(&b, res.Result.Packages, polyglot)
	writeWorkspaceModuleLines(&b, res.Result.Modules, polyglot)
	return strings.TrimRight(b.String(), "\n"), nil
}

func workspaceOutlineLanguages(result *backend.WorkspaceOutlineResult) map[string]struct{} {
	languages := make(map[string]struct{})
	for _, mod := range result.Modules {
		if mod.Language != "" {
			languages[mod.Language] = struct{}{}
		}
	}
	for _, pkg := range result.Packages {
		if pkg.Language != "" {
			languages[pkg.Language] = struct{}{}
		}
	}
	return languages
}

func writeWorkspacePackageLines(b *strings.Builder, packages []backend.WorkspacePackage, polyglot bool) {
	b.WriteString("Packages:\n")
	if len(packages) == 0 {
		b.WriteString("  (no packages found)\n")
		return
	}
	for _, pkg := range packages {
		language := ""
		if polyglot && pkg.Language != "" {
			language = " [" + pkg.Language + "]"
		}
		fmt.Fprintf(b, "  %s (%s)%s — %s\n", pkg.Path, pkg.Name, language, pkg.Synopsis)
	}
}

func writeWorkspaceModuleLines(b *strings.Builder, modules []backend.WorkspaceModule, polyglot bool) {
	if len(modules) == 0 {
		return
	}
	b.WriteString("\nModules:\n")
	for _, mod := range modules {
		rootMarker := workspaceModuleMarker(mod, polyglot)
		if mod.ModulePath != "" {
			goVer := ""
			if mod.GoVersion != "" {
				goVer = ", go " + mod.GoVersion
			}
			fmt.Fprintf(b, "  %s (%s%s)%s\n", mod.Path, mod.ModulePath, goVer, rootMarker)
			continue
		}
		coordinates := strings.Join([]string{mod.GroupID, mod.ArtifactID, mod.Version}, ":")
		packaging := ""
		if mod.Packaging != "" {
			packaging = ", " + mod.Packaging
		}
		fmt.Fprintf(b, "  %s (%s%s)%s\n", mod.Path, coordinates, packaging, rootMarker)
	}
}

func workspaceModuleMarker(module backend.WorkspaceModule, polyglot bool) string {
	marker := ""
	if module.IsRoot {
		marker = " [root]"
	}
	if polyglot && module.Language != "" {
		marker += " [" + module.Language + "]"
	}
	if module.Synopsis != "" {
		marker += " — " + module.Synopsis
	}
	return marker
}

func workspaceOutlineDef() Def[WorkspaceOutlineReq, WorkspaceOutlineRes] {
	return Def[WorkspaceOutlineReq, WorkspaceOutlineRes]{
		Key:      "workspace_outline",
		Summary:  "Use this tool instead of find, tree, ls, or shell directory scanning to orient yourself in a Go or Java workspace; it lists packages, synopses, and Go module or Maven reactor boundaries ordered by depth. Use semantic_outline next to inspect declarations.",
		Params:   workspaceOutlineParams,
		Level:    LevelWorkspace,
		CLIName:  "workspace-outline",
		MCPName:  "semantic_workspace_outline",
		ReadOnly: true,
		Parse:    parseWorkspaceOutline,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, WorkspaceOutlineReq) (WorkspaceOutlineRes, error){
			backend.LanguageAuto: runWorkspaceOutline,
			backend.LanguageGo:   runWorkspaceOutline,
			backend.LanguageJava: runWorkspaceOutline,
		},
		Format:        formatWorkspaceOutline,
		Output:        &OperationOutputContract{ResultSchema: workspaceOutlineOutputSchema()},
		ProjectOutput: projectWorkspaceOutlineOutput,
		ExampleRaw:    map[string]any{"path": "."},
		Batchable:     false,
	}
}

func registerWorkspaceOutlineOps(registry *Registry) error {
	return Register(registry, workspaceOutlineDef())
}
