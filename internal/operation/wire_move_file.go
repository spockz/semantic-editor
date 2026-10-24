// Package operation keeps whole-file selection separate from single-symbol placement so callers get one checked declaration-set plan.
package operation

import (
	"context"
	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
)

import (
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type moveFileError struct {
	State        string
	ChangedFiles []string
	Err          error
}

func (e *moveFileError) Error() string {
	return fmt.Sprintf("move_file %s; changed paths %v: %v", e.State, e.ChangedFiles, e.Err)
}
func (e *moveFileError) Unwrap() error { return e.Err }

func handleMoveFile(ctx context.Context, cc CallContext, req MoveFileReq) (FileEditRes, error) {
	handler := bufferedFileEditHandler("move_file", func(req MoveFileReq) string { return req.File }, runMoveFile)
	result, err := handler(ctx, cc, req)
	if err == nil {
		return result, nil
	}
	failure := &moveFileError{State: moveFileNotApplied, ChangedFiles: []string{}, Err: err}
	if publication, ok := errors.AsType[*sourcebuf.PublicationError](err); ok {
		for _, path := range publication.Published {
			failure.ChangedFiles = append(failure.ChangedFiles, displayWorkPath(cc.WorkDir, path))
		}
		if len(publication.Published) > 0 {
			failure.State = string(EditPartiallyPublished)
		}
	}
	return result, failure
}

func projectMoveFileOutput(result FileEditRes) (any, error) {
	projected, err := projectFileEditOutput(result, "file", "diff")
	if err != nil {
		return nil, err
	}
	fields := projected.(map[string]any)
	fields[wireApplicationState] = result.ApplicationState
	fields["changed_files"] = result.ChangedFiles
	return fields, nil
}

func projectMoveFileError(err error) (any, bool, error) {
	var failure *moveFileError
	if !errors.As(err, &failure) {
		return nil, false, nil
	}
	return map[string]any{wireApplicationState: failure.State, "changed_files": failure.ChangedFiles, "refusal": failure.Err.Error()}, true, nil
}

const moveFileDestinationField = "destination_file"

const moveFileNotApplied = "not_applied"

// MoveFileReq selects one closed Go declaration set and its explicit destination.
type MoveFileReq struct {
	Project           backend.ProjectContext
	File              string
	DestinationFile   string
	MoveMatchingTests bool
}

// GetProjectContext returns the project used for registry dispatch.
func (r MoveFileReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the project during dispatch merge.
func (r *MoveFileReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var moveFileParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Existing Go production file whose complete declaration set will move; the source becomes a package-only shell.", Required: true},
	{Name: moveFileDestinationField, CLIName: "destination-file", JSONName: moveFileDestinationField, Type: ParamString, Role: RoleTarget, Description: "Existing production file in another existing package of the same module; declarations are appended and collisions refuse.", Required: true},
	{Name: "move_matching_tests", CLIName: "move-matching-tests", JSONName: "move_matching_tests", Type: ParamBoolean, Description: "Move the exact sibling _test.go when its resolved dependencies fit the selected declarations and test file; defaults true and refuses unsafe test dependencies.", Default: true},
}, readProjectParams[:5]...)

func parseMoveFile(raw map[string]any) (MoveFileReq, error) {
	var req MoveFileReq
	if err := CheckParams(raw, moveFileParams); err != nil {
		return req, err
	}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.DestinationFile, err = ParseString(raw, (moveFileDestinationField), "destination-file", true); err != nil {
		return req, err
	}
	if req.MoveMatchingTests, err = ParseBool(raw, "move_matching_tests", "move-matching-tests", true); err != nil {
		return req, err
	}
	req.Project, err = parseReadProject(raw, req.File)
	return req, err
}

func runMoveFile(ctx context.Context, cc CallContext, req MoveFileReq) (FileEditRes, error) {
	source := resolveWorkPath(cc.WorkDir, req.File)
	destination := resolveWorkPath(cc.WorkDir, req.DestinationFile)
	if cc.InBatch {
		if err := validateBatchMoveTargets(ctx, cc.WorkDir, source, destination); err != nil {
			return FileEditRes{}, err
		}
	}
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.MoveFile(ctx, source, destination, astedit.MoveFileOptions{MoveMatchingTests: req.MoveMatchingTests, AutoOrganizeImports: effectiveAutoOrganize(cc, true), DeferFormatting: cc.DeferImports})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, status := finishDelta()
	return FileEditRes{File: destination, Display: displayWorkPath(cc.WorkDir, req.DestinationFile), DestinationFile: destination, DestinationDisplay: displayWorkPath(cc.WorkDir, req.DestinationFile), Diff: diff, Detail: "Successfully moved file declarations.", Delta: delta, DiagnosticStatus: status, HasDelta: true}, nil
}

func moveFileDef() Def[MoveFileReq, FileEditRes] {
	return Def[MoveFileReq, FileEditRes]{
		Key: "refactor_move_file", CLIName: "refactor-move-file", MCPName: "refactor_move_file", Level: LevelWorkspace,
		Summary: "Use this tool instead of sequential symbol moves to relocate a complete closed Go production-file declaration set to an explicit existing file in another package of the same module. Supports private helpers, constants, types and methods together, and dependency-closed matching tests by default. Leaves package-only source shells. Refuses package variables, init, outside dependencies, unsafe test helpers, fixtures, generated/build-constrained files and unsupported initialization effects before publication. Other backends are unsupported.",
		Params:  moveFileParams, Parse: parseMoveFile, Handlers: map[backend.LanguageID]func(context.Context, CallContext, MoveFileReq) (FileEditRes, error){backend.LanguageGo: handleMoveFile},
		Format: formatMoveFile, ExampleRaw: map[string]any{"file": "old/value.go", moveFileDestinationField: "api/value.go", "move_matching_tests": true}, Batchable: true, BufferedBatch: true,
		Output:         fileEditOutputContract(map[string]ValueSchema{"file": {Kind: SchemaString}, "diff": {Kind: SchemaString}, wireApplicationState: {Kind: SchemaString}, "changed_files": {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}}}, []OutputContinuation{outputContinuation("file", RoleTarget, fileOutputConsumers)}),
		ErrorOutput:    &OperationErrorOutputContract{ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{wireApplicationState: {Kind: SchemaString, Enum: []any{moveFileNotApplied, string(EditPartiallyPublished)}}, "changed_files": {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}}, "refusal": {Kind: SchemaString}}, Required: []string{wireApplicationState, "changed_files", "refusal"}}},
		FinalizeOutput: finalizeMoveFileOutput, ProjectOutput: projectMoveFileOutput, ProjectErrorOutput: projectMoveFileError,
	}
}

func finalizeMoveFileOutput(cc CallContext, _ MoveFileReq, result FileEditRes) (FileEditRes, error) {
	final, err := finalizeFileEditOutput(cc, result)
	if err != nil {
		return FileEditRes{}, err
	}
	final.ApplicationState = string(final.Receipt.Status)
	final.ChangedFiles = make([]string, 0, len(final.Receipt.Changes))
	for _, change := range final.Receipt.Changes {
		final.ChangedFiles = append(final.ChangedFiles, change.Path)
	}
	return final, nil
}

func formatMoveFile(result FileEditRes) (string, error) {
	result.Diff = ""
	if result.HasDelta {
		result.Detail += " Diagnostics: " + result.DiagnosticStatus + "."
		if result.DiagnosticStatus == "unchanged" || result.DiagnosticStatus == "changed" {
			result.Detail += fmt.Sprintf(" Captured introduced %d, resolved %d.", len(result.Delta.Introduced), len(result.Delta.Resolved))
		}
		result.HasDelta = false
	}
	if len(result.ChangedFiles) > 10 {
		result.Detail += fmt.Sprintf(" Changed paths: %d total; first 10 shown. Inspect the MCP structured result or git diff --name-only for all paths.", len(result.ChangedFiles))
		result.ChangedFiles = result.ChangedFiles[:10]
	}
	return formatFileEdit(result)
}
