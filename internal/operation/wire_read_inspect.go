// Package operation defines the public request schemas and read-only handlers for Go inspect and outline.
package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/capability"
)

// OutlineReq requests a filtered Go source outline.
type OutlineReq struct {
	Project           backend.ProjectContext
	Path              string
	Kinds             []string
	IncludeUnexported bool
	IncludeTests      bool
}

// GetProjectContext returns the project selected for outline dispatch.
func (r OutlineReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the effective project selected by registry dispatch.
func (r *OutlineReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// InspectReq requests exact source metadata for one Go symbol.
type InspectReq struct {
	Project backend.ProjectContext
	Symbol  string
}

// GetProjectContext returns the project selected for inspect dispatch.
func (r InspectReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the effective project selected by registry dispatch.
func (r *InspectReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var outlineReadParams = readRequestParams(
	ParameterContract{Name: "path", CLIName: "path", JSONName: "path", Description: "Source file or directory to outline", Type: ParamString, Required: true},
	ParameterContract{Name: "kinds", CLIName: "kind", JSONName: "kinds", Description: "Optional kinds to include: type, interface, function, method, field, constant, variable, package, class, constructor, enum, record, module, property, object, enum_member, trait, target; known kinds absent from a backend produce empty symbols", Type: ParamStringSlice},
	ParameterContract{Name: "include_unexported", CLIName: "include-unexported", JSONName: "include_unexported", Description: "Include unexported Go declarations and members; external backends require true (default true)", Type: ParamBoolean, Default: true},
	ParameterContract{Name: "include_tests", CLIName: "include-tests", JSONName: "include_tests", Description: "Include _test.go files for Go directory scopes (default false)", Type: ParamBoolean, Default: false},
)

var inspectReadParams = readRequestParams(
	ParameterContract{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: qualified Go symbol name to inspect", Type: ParamString, Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, sourceReplaceBodySymbol}},
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Description: "Selector: optional source file that constrains symbol search scope", Type: ParamString, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile, sourceReplaceConstructFile}},
)

func parseInspectRead(raw map[string]any) (InspectReq, error) {
	if err := CheckParams(raw, inspectReadParams); err != nil {
		return InspectReq{}, err
	}
	query, err := ParseString(raw, "symbol", "symbol", true)
	if err != nil {
		return InspectReq{}, err
	}
	file, err := ParseString(raw, "file", "file", false)
	if err != nil {
		return InspectReq{}, err
	}
	project, err := parseReadProject(raw, file)
	if err != nil {
		return InspectReq{}, err
	}
	return InspectReq{Project: project, Symbol: query}, nil
}

func parseOutlineRead(raw map[string]any) (OutlineReq, error) {
	if err := CheckParams(raw, outlineReadParams); err != nil {
		return OutlineReq{}, err
	}
	path, err := ParseString(raw, "path", "path", true)
	if err != nil {
		return OutlineReq{}, err
	}
	kinds, err := ParseStringSlice(raw, "kinds", "kind")
	if err != nil {
		return OutlineReq{}, err
	}
	includeUnexported, err := ParseBool(raw, "include_unexported", "include-unexported", true)
	if err != nil {
		return OutlineReq{}, err
	}
	includeTests, err := ParseBool(raw, "include_tests", "include-tests", false)
	if err != nil {
		return OutlineReq{}, err
	}
	project, err := parseReadProject(raw, path)
	if err != nil {
		return OutlineReq{}, err
	}
	return OutlineReq{Project: project, Path: path, Kinds: kinds, IncludeUnexported: includeUnexported, IncludeTests: includeTests}, nil
}

func inspectReadRun(ctx context.Context, cc CallContext, request InspectReq) (*backend.InspectResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, request.Project)
	return service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: request.Symbol})
}

func outlineReadRun(ctx context.Context, cc CallContext, request OutlineReq) (*backend.OutlineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, request.Project)
	return service.Outline(ctx, backend.OutlineRequest{
		Project: project, Path: request.Path, Kinds: request.Kinds,
		IncludeUnexported: request.IncludeUnexported, IncludeTests: request.IncludeTests,
	})
}

func formatInspectRead(result *backend.InspectResult) (string, error) {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format inspect result: %w", err)
	}
	return string(output), nil
}

func formatOutlineRead(result *backend.OutlineResult) (string, error) {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format outline result: %w", err)
	}
	return string(output), nil
}

func inspectReadDef() Def[InspectReq, *backend.InspectResult] {
	return Def[InspectReq, *backend.InspectResult]{
		Key: capability.OpInspect, Summary: "Use this tool instead of reading source with shell commands when you need a declaration's exact source, metadata, and snapshot revision.",
		Params: inspectReadParams, Level: LevelSymbol, CLIName: "inspect-symbol", MCPName: "semantic_inspect_symbol",
		ReadOnly: true, Parse: parseInspectRead,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InspectReq) (*backend.InspectResult, error){
			backend.LanguageGo:   inspectReadRun,
			backend.LanguageJava: inspectReadRun,
			backend.LanguageMake: inspectReadRun,
		},
		Format:     formatInspectRead,
		ExampleRaw: map[string]any{"symbol": "Server.Start", "language": "go", wireTrustWorkspace: false},
	}
}

func outlineReadDef() Def[OutlineReq, *backend.OutlineResult] {
	return Def[OutlineReq, *backend.OutlineResult]{
		Key: capability.OpOutline, Summary: "Use this tool instead of scanning source files with shell commands when you need a declaration outline for a selected source scope; directory scopes are Go-only.",
		Params: outlineReadParams, Level: LevelSymbol, CLIName: "outline", MCPName: "semantic_outline",
		ReadOnly: true, Parse: parseOutlineRead, PrepareProject: prepareOutlineProject,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, OutlineReq) (*backend.OutlineResult, error){
			backend.LanguageGo:     outlineReadRun,
			backend.LanguageJava:   outlineReadRun,
			backend.LanguageBash:   outlineReadRun,
			backend.LanguageMake:   outlineReadRun,
			backend.LanguageKotlin: outlineReadRun,
			backend.LanguageRust:   outlineReadRun,
			backend.LanguageScala:  outlineReadRun,
		},
		Format:     formatOutlineRead,
		ExampleRaw: map[string]any{"path": "internal/astedit", "language": "go", "include_unexported": true, "include_tests": false},
	}
}

func prepareOutlineProject(project backend.ProjectContext) (backend.ProjectContext, error) {
	root, err := filepath.Abs(project.RootDir)
	if err != nil {
		return backend.ProjectContext{}, fmt.Errorf("resolve outline root %q: %w", project.RootDir, err)
	}
	target := project.File
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return backend.ProjectContext{}, fmt.Errorf("resolve outline path %q: %w", project.File, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return backend.ProjectContext{}, fmt.Errorf("stat outline path %q: %w", target, err)
	}
	if info.IsDir() {
		project.File = ""
	}
	return project, nil
}

func registerReadInspectOps(registry *Registry) error {
	if err := Register(registry, inspectReadDef()); err != nil {
		return err
	}
	return Register(registry, outlineReadDef())
}
