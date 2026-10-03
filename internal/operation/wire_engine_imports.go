// Package operation groups import organization and dependency insertion handlers.
package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
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

type organizeImportsError struct {
	ApplicationState string
	ChangedFiles     []string
	DiagnosticStatus string
	Err              error
}

const (
	organizeImportsApplicationState = "application_state"
	organizeImportsChangedFiles     = "changed_files"
	organizeImportsDiagnosticStatus = "diagnostic_status"
	organizeImportsDiagnosticDelta  = "diagnostic_delta"
)

var organizeImportsParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Selector: optional file to constrain search scope. Optional Go file or directory path, relative to the active semedit workspace root. Omit it to organize imports across the entire workspace, which may write multiple files"},
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
	report, err := pipeline.OrganizeImportsWithOptionsReport(ctx, cc.WorkDir, pipeline.ImportOptions{
		Add:    req.Add,
		Remove: req.Remove,
	}, paths...)
	diagnosticStatus := diagnosticStatusUnavailable
	if report.Buffered {
		diagnosticStatus = "deferred"
	}
	changedFiles := make([]string, 0, len(report.ChangedFiles))
	for _, changedPath := range report.ChangedFiles {
		relative, relErr := filepath.Rel(cc.WorkDir, changedPath)
		if relErr != nil {
			failure := fmt.Errorf("make changed import path relative %q: %w", changedPath, relErr)
			return FileEditRes{ChangedFiles: changedFiles, ApplicationState: string(EditStatus("unknown"))}, &organizeImportsError{ApplicationState: string(EditStatus("unknown")), ChangedFiles: changedFiles, DiagnosticStatus: diagnosticStatus, Err: failure}
		}
		changedFiles = append(changedFiles, filepath.ToSlash(relative))
	}
	if err != nil {
		state := EditStatus("unknown")
		if len(changedFiles) > 0 {
			if report.Buffered {
				state = EditAppliedWorking
			} else {
				state = EditPartiallyPublished
			}
		}
		return FileEditRes{ChangedFiles: changedFiles, ApplicationState: string(state)}, &organizeImportsError{ApplicationState: string(state), ChangedFiles: changedFiles, DiagnosticStatus: diagnosticStatus, Err: err}
	}
	state := EditPersisted
	if report.Buffered {
		state = EditAppliedWorking
	}
	if len(changedFiles) == 0 {
		state = EditNoOp
	}
	delta, diagnosticStatus := finishDelta()
	written := cc.WorkDir
	if len(paths) > 0 {
		written = paths[0]
	}
	display := ""
	if info, statErr := os.Stat(written); statErr == nil && info.Mode().IsRegular() {
		display = displayWorkPath(cc.WorkDir, written)
	}
	return FileEditRes{File: written, Display: display, Detail: "Organized imports.", ChangedFiles: changedFiles, ApplicationState: string(state), Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}, nil
}

func organizeImportsDef() Def[OrganizeImportsReq, FileEditRes] {
	return Def[OrganizeImportsReq, FileEditRes]{
		Key:     "organize_imports",
		Summary: "Use this tool instead of text editing import blocks or shelling out to goimports when adding, removing, or formatting Go source imports. Set `file` to limit writes to one file or directory; omit it only when workspace-wide import changes are intended. Supports explicit package additions (including aliases and blank imports) and removals." + automaticVerificationGuidance,
		Params:  organizeImportsParams,
		Level:   LevelFile,
		CLIName: "imports",
		MCPName: "semantic_organize_imports",
		Parse:   parseOrganizeImports,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, OrganizeImportsReq) (FileEditRes, error){
			backend.LanguageGo: runOrganizeImports,
		},
		Format: func(res FileEditRes) (string, error) {
			res.Detail = "Successfully organized imports."
			return formatFileEdit(res)
		},
		Output: &OperationOutputContract{ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
			"status":                        {Kind: SchemaString, Enum: []any{"organized"}},
			"file":                          nonEmptyStringSchema(),
			organizeImportsApplicationState: {Kind: SchemaString, Enum: []any{"persisted", "applied_working", "no_op"}},
			organizeImportsChangedFiles:     {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
			organizeImportsDiagnosticStatus: {Kind: SchemaString, Enum: []any{"changed", "unchanged", "deferred", diagnosticStatusUnavailable}},
			organizeImportsDiagnosticDelta:  diagnosticDeltaSchema(),
		}, Required: []string{"status", organizeImportsApplicationState, organizeImportsChangedFiles, organizeImportsDiagnosticStatus, organizeImportsDiagnosticDelta}}, Continuations: []OutputContinuation{
			{Field: "file", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: organizeImportsFileConsumers, Optional: true},
		}},
		ErrorOutput: &OperationErrorOutputContract{ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
			organizeImportsApplicationState: {Kind: SchemaString, Enum: []any{"partially_published", "applied_working", "unknown"}},
			organizeImportsChangedFiles:     {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
			organizeImportsDiagnosticStatus: {Kind: SchemaString, Enum: []any{"deferred", diagnosticStatusUnavailable}},
		}, Required: []string{organizeImportsApplicationState, organizeImportsChangedFiles, organizeImportsDiagnosticStatus}}},
		ProjectOutput: func(res FileEditRes) (any, error) {
			if res.ApplicationState == "" {
				return nil, fmt.Errorf("organize imports application state is unknown")
			}
			files := res.ChangedFiles
			if files == nil {
				files = []string{}
			}
			result := map[string]any{"status": "organized", organizeImportsApplicationState: res.ApplicationState, organizeImportsChangedFiles: files, organizeImportsDiagnosticStatus: res.DiagnosticStatus, organizeImportsDiagnosticDelta: diagnosticDeltaOutput(res.Delta)}
			if res.Display != "" {
				result["file"] = res.Display
			}
			return result, nil
		},
		ProjectErrorOutput: func(err error) (any, bool, error) {
			var failure *organizeImportsError
			if !errors.As(err, &failure) {
				return nil, false, nil
			}
			files := failure.ChangedFiles
			if files == nil {
				files = []string{}
			}
			return map[string]any{organizeImportsApplicationState: failure.ApplicationState, organizeImportsChangedFiles: files, organizeImportsDiagnosticStatus: failure.DiagnosticStatus}, true, nil
		},
		ExampleRaw:    map[string]any{"file": wireExampleFile},
		BufferedBatch: true, Batchable: true,
	}
}

var organizeImportsFileConsumers = []string{
	semanticLookupToolName, semanticRenameToolName, "semantic_verify",
	semanticReplaceBodyToolName, "semantic_replace_construct", semanticMoveConstructToolName,
	"semantic_remove_construct", "semantic_insert_case", "semantic_insert_function",
	"semantic_insert_type", "semantic_insert_decl", "semantic_insert_declaration",
	"semantic_insert_structure", semanticReplaceDeclToolName, semanticExtractFunctionToolName,
	semanticExtractVariableToolName, semanticInlineVariableToolName,
	"semantic_find_references", "semantic_inspect_symbol", "semantic_assertion_mode",
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

func (e *organizeImportsError) Error() string {
	return e.Err.Error()
}

func (e *organizeImportsError) Unwrap() error {
	return e.Err
}

var addBuildDependencyParams = []ParameterContract{
	{Name: "package", CLIName: "package", JSONName: "package", Type: ParamString, Description: "Module path to fetch (e.g. github.com/google/uuid@latest); runs go get and go mod tidy, can use the network, and may update go.mod and go.sum without editing source files", Required: true},
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
		Summary: "Use this tool instead of editing go.mod/go.sum or shelling out to go get or go mod tidy when adding an external Go module dependency. It runs go get and go mod tidy and may access the network. This changes build dependencies; use semantic_organize_imports for source import statements.",
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
