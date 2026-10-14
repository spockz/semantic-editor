// Package operation groups import organization and dependency insertion handlers.
package operation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/adapters/golang"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/maven/pomedit"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

var organizeImportsParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Selector: optional Go file or directory path, relative to the active workspace root. Omit it to organize imports across the entire workspace (Go may write multiple files). Java requires one selected .java file."},
	{Name: "add", CLIName: "add", JSONName: "add", Type: ParamStringSlice, Description: "Optional imports to explicitly add. Go supports 'path', 'alias path', or '_ path'; Java accepts fully qualified type names."},
	{Name: "remove", CLIName: "remove", JSONName: "remove", Type: ParamStringSlice, Description: "Optional imports to explicitly remove. Java accepts fully qualified type names."},
}, readProjectParams...)

func parseOrganizeImports(raw map[string]any) (OrganizeImportsReq, error) {
	var req OrganizeImportsReq
	if err := CheckParams(raw, organizeImportsParams); err != nil {
		return req, err
	}
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
	req.Project, err = parseReadProject(raw, req.File)
	if err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		req.Project.Language = backend.LanguageGo
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

func runOrganizeImportsJava(ctx context.Context, cc CallContext, req OrganizeImportsReq) (FileEditRes, error) {
	if strings.TrimSpace(req.File) == "" || !strings.EqualFold(filepath.Ext(req.File), ".java") {
		return FileEditRes{}, fmt.Errorf("java import organization requires one selected .java file")
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	before, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("read Java import target: %w", err)
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	if _, verifyErr := service.Verify(ctx, backend.VerifyRequest{
		Project: project, Path: file, FormatSelectedFile: true, OrganizeImports: true,
		AddImports: req.Add, RemoveImports: req.Remove,
	}); verifyErr != nil {
		return FileEditRes{}, javaOrganizeImportsFailure(ctx, cc, file, before, verifyErr)
	}
	after, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return FileEditRes{}, javaOrganizeImportsFailure(ctx, cc, file, before, fmt.Errorf("read organized Java source: %w", err))
	}
	state := EditPersisted
	changedFiles := []string{}
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		state = EditAppliedWorking
	}
	if !bytes.Equal(before, after) {
		relative, relErr := filepath.Rel(cc.WorkDir, file)
		if relErr != nil {
			return FileEditRes{}, fmt.Errorf("make Java import path relative: %w", relErr)
		}
		changedFiles = append(changedFiles, filepath.ToSlash(relative))
	} else {
		state = EditNoOp
	}
	diagnosticStatus := diagnosticStatusUnavailable
	if cc.DeferVerification {
		diagnosticStatus = "deferred"
	}
	return FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, file), Detail: "Organized imports.",
		ChangedFiles: changedFiles, ApplicationState: string(state), DiagnosticStatus: diagnosticStatus,
		HasDelta: true,
	}, nil
}

func javaOrganizeImportsFailure(ctx context.Context, cc CallContext, file string, before []byte, cause error) error {
	state := EditStatus("unknown")
	changedFiles := []string{}
	diagnosticStatus := diagnosticStatusUnavailable
	if cc.DeferVerification {
		diagnosticStatus = "deferred"
	}
	if after, err := sourcebuf.ReadFile(ctx, file); err == nil && !bytes.Equal(before, after) {
		state = EditPartiallyPublished
		if sourcebuf.FromContext(ctx) != nil {
			state = EditAppliedWorking
		}
		if relative, relErr := filepath.Rel(cc.WorkDir, file); relErr == nil {
			changedFiles = append(changedFiles, filepath.ToSlash(relative))
		} else {
			cause = errors.Join(cause, fmt.Errorf("make Java import failure path relative: %w", relErr))
		}
	}
	return &organizeImportsError{ApplicationState: string(state), ChangedFiles: changedFiles, DiagnosticStatus: diagnosticStatus, Err: cause}
}

func organizeImportsDef() Def[OrganizeImportsReq, FileEditRes] {
	return Def[OrganizeImportsReq, FileEditRes]{
		Key:     operationKeyOrganizeImports,
		Summary: "Use this tool instead of text editing import blocks or shelling out to goimports when adding, removing, or formatting Go imports. For Java, select one trusted .java file; explicit additions and removals use fully qualified type names before JDT LS formats and organizes imports." + automaticVerificationGuidance,
		Params:  organizeImportsParams,
		Level:   LevelFile,
		CLIName: "imports",
		MCPName: "semantic_organize_imports",
		Parse:   parseOrganizeImports,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, OrganizeImportsReq) (FileEditRes, error){
			backend.LanguageGo:   runOrganizeImports,
			backend.LanguageJava: runOrganizeImportsJava,
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
	semanticReplaceBodyToolName, semanticReplaceConstructToolName, semanticMoveConstructToolName,
	semanticRemoveConstructToolName, semanticInsertCaseToolName, semanticInsertFunctionToolName,
	semanticInsertTypeToolName, semanticInsertDeclToolName, semanticInsertDeclarationToolName,
	semanticInsertConstructToolName, semanticReplaceDeclToolName, semanticExtractFunctionToolName,
	semanticExtractConstantToolName,
	semanticExtractVariableToolName, semanticInlineVariableToolName,
	semanticFindReferencesToolName, semanticInspectSymbolToolName, semanticInspectConstructToolName, semanticAssertionModeToolName,
}

// AddBuildDependencyReq fetches one external module into the build (go get + go mod tidy).
type AddBuildDependencyReq struct {
	Project    backend.ProjectContext
	Package    string
	Dependency string
	File       string
}

// GetProjectContext returns the request project for registry dispatch.
func (r AddBuildDependencyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *AddBuildDependencyReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

// BuildDependencyRes reports a build-file mutation for response rendering and batch post-processing.
type BuildDependencyRes struct {
	Package    string
	GoMod      string
	Dependency string
	POMs       []string
}

func (e *organizeImportsError) Error() string {
	return e.Err.Error()
}

func (e *organizeImportsError) Unwrap() error {
	return e.Err
}

var addBuildDependencyParams = append([]ParameterContract{
	{Name: "package", CLIName: "package", JSONName: "package", Type: ParamString, Description: "Go module path to fetch (e.g. github.com/google/uuid@latest); updates go.mod/go.sum, runs go get and go mod tidy, and may access the network."},
	{Name: operationKeyDependency, CLIName: operationKeyDependency, JSONName: operationKeyDependency, Type: ParamString, Description: "Java Maven coordinate groupId:artifactId[:version][:scope][:classifier]."},
	{Name: "file", CLIName: "file", JSONName: "file", InputAliases: []string{"path"}, Type: ParamString, Description: "Optional Java module directory or pom.xml path in a Maven reactor; defaults to the workspace root POM."},
}, readProjectParams...)

func parseAddBuildDependency(raw map[string]any) (AddBuildDependencyReq, error) {
	var req AddBuildDependencyReq
	if err := CheckParams(raw, addBuildDependencyParams); err != nil {
		return req, err
	}
	var err error
	if req.Package, err = ParseString(raw, "package", "package", false); err != nil {
		return req, err
	}
	if req.Dependency, err = ParseString(raw, operationKeyDependency, operationKeyDependency, false); err != nil {
		return req, err
	}
	if req.File, err = ParseString(raw, "file", "file", false); err != nil {
		return req, err
	}
	if req.File == "" {
		if req.File, err = ParseString(raw, "path", "path", false); err != nil {
			return req, err
		}
	}
	req.Project, err = parseReadProject(raw, req.File)
	if err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		if req.Dependency != "" {
			req.Project.Language = backend.LanguageJava
		} else {
			req.Project.Language = backend.LanguageGo
		}
	}
	switch req.Project.Language {
	case backend.LanguageJava:
		if req.Dependency == "" {
			return req, fmt.Errorf("param %q is required: %w", operationKeyDependency, ErrInvalidParams)
		}
		if req.Package != "" {
			return req, fmt.Errorf("java requires dependency and does not accept package")
		}
	case backend.LanguageGo:
		if req.Package == "" {
			return req, fmt.Errorf("param %q is required: %w", "package", ErrInvalidParams)
		}
		if req.Dependency != "" || req.File != "" {
			return req, fmt.Errorf("go requires package and does not accept dependency or file")
		}
	default:
		return req, fmt.Errorf("add_build_dependency supports Go and Java")
	}
	return req, nil
}

func runAddBuildDependency(ctx context.Context, cc CallContext, req AddBuildDependencyReq) (BuildDependencyRes, error) {
	if err := golang.AddDependency(ctx, cc.WorkDir, req.Package); err != nil {
		return BuildDependencyRes{}, err
	}
	return BuildDependencyRes{Package: req.Package, GoMod: filepath.Join(cc.WorkDir, "go.mod")}, nil
}

func runAddJavaBuildDependency(ctx context.Context, cc CallContext, req AddBuildDependencyReq) (BuildDependencyRes, error) {
	project := effectiveProject(cc, req.Project)
	root, err := filepath.Abs(project.RootDir)
	if err != nil {
		return BuildDependencyRes{}, fmt.Errorf("resolve Java workspace root: %w", err)
	}
	project.RootDir = root
	if !project.WorkspaceTrust.Allows(root) {
		return BuildDependencyRes{}, &backend.WorkspaceTrustError{Operation: backend.OperationAddBuildDependency, Language: backend.LanguageJava, Workspace: root}
	}
	if sourcebuf.FromContext(ctx) != nil {
		return BuildDependencyRes{}, fmt.Errorf("java Maven dependency edits are not supported in buffered batches")
	}
	dependency, err := pomedit.ParseCoordinate(req.Dependency)
	if err != nil {
		return BuildDependencyRes{}, err
	}
	plan, err := pomedit.Plan(root, req.File, dependency)
	if err != nil {
		return BuildDependencyRes{}, err
	}
	written := make([]pomedit.FileEdit, 0, len(plan))
	for _, edit := range plan {
		original, readErr := os.ReadFile(edit.Path)
		if readErr != nil {
			return BuildDependencyRes{}, fmt.Errorf("read POM before update: %w", readErr)
		}
		if writeErr := pipeline.WriteAtomicContext(ctx, edit.Path, edit.Data); writeErr != nil {
			rollbackErr := error(nil)
			for _, w := range slices.Backward(written) {
				rollbackErr = errors.Join(rollbackErr, pipeline.WriteAtomicContext(context.WithoutCancel(ctx), w.Path, w.Data))
			}
			return BuildDependencyRes{}, errors.Join(fmt.Errorf("write Maven POM %q: %w", edit.Path, writeErr), rollbackErr)
		}
		written = append(written, pomedit.FileEdit{Path: edit.Path, Data: original})
	}
	poms := make([]string, 0, len(plan))
	for _, edit := range plan {
		poms = append(poms, edit.Path)
	}
	if len(poms) > 0 && cc.Service != nil {
		if err := cc.Service.NotifyProjectConfigurationUpdate(ctx, project, poms); err != nil {
			return BuildDependencyRes{Dependency: req.Dependency, POMs: poms}, fmt.Errorf("maven POMs updated but JDT LS refresh failed: %w", err)
		}
	}
	return BuildDependencyRes{Dependency: req.Dependency, POMs: poms}, nil
}

func addBuildDependencyDef() Def[AddBuildDependencyReq, BuildDependencyRes] {
	return Def[AddBuildDependencyReq, BuildDependencyRes]{
		Key:     "add_build_dependency",
		Summary: "Use this tool instead of editing go.mod/go.sum or Maven POM files manually to add a build dependency. Go runs go get and go mod tidy and may access the network; Java updates the selected Maven reactor using dependencyManagement inheritance and refreshes JDT LS. Java requires explicit workspace trust. Use semantic_organize_imports for source imports.",
		Params:  addBuildDependencyParams,
		Level:   LevelBuild,
		CLIName: "add-build-dependency",
		MCPName: "semantic_add_build_dependency",
		Parse:   parseAddBuildDependency,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, AddBuildDependencyReq) (BuildDependencyRes, error){
			backend.LanguageGo:   runAddBuildDependency,
			backend.LanguageJava: runAddJavaBuildDependency,
		},
		Format: func(res BuildDependencyRes) (string, error) {
			if res.Dependency != "" {
				return fmt.Sprintf("Successfully added Maven dependency %s to %d POM(s).", res.Dependency, len(res.POMs)), nil
			}
			return fmt.Sprintf("Successfully added dependency %s and tidied go.mod.", res.Package), nil
		},
		Output:        &OperationOutputContract{ResultSchema: dependencyOutputSchema()},
		ProjectOutput: projectDependencyOutput,
		ExampleRaw:    map[string]any{"package": "github.com/google/uuid@latest"},
		Batchable:     true,
	}
}
