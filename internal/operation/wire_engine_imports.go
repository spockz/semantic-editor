// Package operation groups import organization and dependency insertion handlers.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"semedit/internal/adapters/golang"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
)

// OrganizeImportsReq manages import statements of a file or workspace scope.
type OrganizeImportsReq struct {
	Project backend.ProjectContext
	File    string
	Add     []string
	Remove  []string
}

// GetProjectContext returns the request project for registry dispatch.
func (r OrganizeImportsReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *OrganizeImportsReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var organizeImportsParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Optional file or directory path to process (defaults to entire workspace)"},
	{Name: "add", CLIName: "add", JSONName: "add", Type: ParamStringSlice, Description: "Optional list of import paths to explicitly add. Supports 'path', 'alias path', or '_ path'."},
	{Name: "remove", CLIName: "remove", JSONName: "remove", Type: ParamStringSlice, Description: "Optional list of import paths to explicitly remove."},
}

func parseOrganizeImports(raw map[string]any) (OrganizeImportsReq, error) {
	var req OrganizeImportsReq
	if err := CheckParams(raw, organizeImportsParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", false); err != nil {
		return req, err
	}
	if req.Add, err = ParseStringSlice(raw, "add", "add"); err != nil {
		return req, err
	}
	if req.Remove, err = ParseStringSlice(raw, "remove", "remove"); err != nil {
		return req, err
	}
	return req, nil
}

func runOrganizeImports(ctx context.Context, cc CallContext, req OrganizeImportsReq) (FileEditRes, error) {
	var paths []string
	if req.File != "" {
		paths = []string{resolveWorkPath(cc.WorkDir, req.File)}
	}
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	err := pipeline.OrganizeImportsWithOptions(ctx, cc.WorkDir, pipeline.ImportOptions{
		Add:    req.Add,
		Remove: req.Remove,
	}, paths...)
	if err != nil {
		return FileEditRes{}, err
	}
	written := cc.WorkDir
	if len(paths) > 0 {
		written = paths[0]
	}
	return FileEditRes{File: written, Display: req.File, Symbol: req.File, Detail: "Successfully organized imports.", Delta: finishDelta(), HasDelta: true}, nil
}

func organizeImportsDef() Def[OrganizeImportsReq, FileEditRes] {
	return Def[OrganizeImportsReq, FileEditRes]{
		Key:     "organize_imports",
		Summary: "Use this tool to format imports, resolve missing package imports, and strip unused imports across specified files or the workspace. Supports explicit package additions (including aliases and blank imports) and explicit removals." + automaticVerificationGuidance,
		Params:  organizeImportsParams,
		Level:   LevelFile,
		CLIName: "imports",
		MCPName: "semantic_organize_imports",
		Parse:   parseOrganizeImports,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, OrganizeImportsReq) (FileEditRes, error){
			backend.LanguageGo: runOrganizeImports,
		},
		Format: func(res FileEditRes) (string, error) {
			return AppendDiagnosticDelta("Successfully organized imports.", res.Delta), nil
		},
		ExampleRaw: map[string]any{"file": wireExampleFile},
		Batchable:  true,
	}
}

// AddBuildDependencyReq fetches one external module into the build (go get + go mod tidy).
type AddBuildDependencyReq struct {
	Project backend.ProjectContext
	Package string
}

// GetProjectContext returns the request project for registry dispatch.
func (r AddBuildDependencyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *AddBuildDependencyReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

// BuildDependencyRes reports a build-file mutation for response rendering and batch post-processing.
type BuildDependencyRes struct {
	Package string
	GoMod   string
}

// WrittenFile returns the mutated go.mod path for batch post-processing.
func (r BuildDependencyRes) WrittenFile() string { return r.GoMod }

var addBuildDependencyParams = []ParameterContract{
	{Name: "package", CLIName: "package", JSONName: "package", Type: ParamString, Description: "Module path to fetch into the build (e.g. github.com/google/uuid@latest); mutates go.mod, not source files", Required: true},
}

func parseAddBuildDependency(raw map[string]any) (AddBuildDependencyReq, error) {
	var req AddBuildDependencyReq
	if err := CheckParams(raw, addBuildDependencyParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.Package, err = ParseString(raw, "package", "package", true); err != nil {
		return req, err
	}
	return req, nil
}

func runAddBuildDependency(ctx context.Context, cc CallContext, req AddBuildDependencyReq) (BuildDependencyRes, error) {
	if err := golang.AddDependency(ctx, cc.WorkDir, req.Package); err != nil {
		return BuildDependencyRes{}, err
	}
	return BuildDependencyRes{Package: req.Package, GoMod: filepath.Join(cc.WorkDir, "go.mod")}, nil
}

func addBuildDependencyDef() Def[AddBuildDependencyReq, BuildDependencyRes] {
	return Def[AddBuildDependencyReq, BuildDependencyRes]{
		Key:     "add_build_dependency",
		Summary: "Add an external module to the build (runs 'go get <package>' and 'go mod tidy') without manual shell execution. Build scope (go.mod); contrast with organize_imports, which edits import statements within source files.",
		Params:  addBuildDependencyParams,
		Level:   LevelBuild,
		CLIName: "add-build-dependency",
		MCPName: "semantic_add_build_dependency",
		Parse:   parseAddBuildDependency,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, AddBuildDependencyReq) (BuildDependencyRes, error){
			backend.LanguageGo: runAddBuildDependency,
		},
		Format: func(res BuildDependencyRes) (string, error) {
			return fmt.Sprintf("Successfully added dependency %s and tidied go.mod.", res.Package), nil
		},
		ExampleRaw: map[string]any{"package": "github.com/google/uuid@latest"},
		Batchable:  true,
	}
}
