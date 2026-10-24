// Package java implements Java language-server operations and source transformations.
// WHY: This file validates JDT LS code actions before publishing their source edits.
package java

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const javaMoveResourceKind = "moveResource"
const javaMakeStaticOperation backend.Operation = "make_static"
const javaMakeStaticActionKind = backend.CodeActionInlineKind
const javaMakeStaticActionTitle = "Make Static"

type javaCodeAction struct {
	Title    string          `json:"title"`
	Kind     string          `json:"kind"`
	Edit     json.RawMessage `json:"edit"`
	Disabled json.RawMessage `json:"disabled"`
	Command  *javaCommand    `json:"command"`
}

type javaCommand struct {
	Command   string            `json:"command"`
	Arguments []json.RawMessage `json:"arguments"`
}

type javaApplyEditRequester interface {
	RequestWithApplyEdit(context.Context, string, any, func(json.RawMessage) (bool, error)) (json.RawMessage, error)
}

type javaCodeActionRollback struct {
	buffer     *sourcebuf.Buffer
	ctx        context.Context
	checkpoint sourcebuf.Checkpoint
	files      []string
	staged     bool
}

func validateJavaCodeActionRequest(request backend.CodeActionRequest) error {
	if request.Operation == javaMakeTypeExplicitOperation {
		if request.Kind != "refactor" || request.Symbol == "" || request.EnclosingSymbol == "" || request.ExpectedRevision == "" {
			return fmt.Errorf("java make-type-explicit requires a local, owner, fresh revision, and refactor kind")
		}
		return javacst.ValidateJavaIdentifier(request.Symbol)
	}
	wantKind := map[backend.Operation]string{
		backend.OperationExtractFunction:          "refactor.extract.function",
		backend.OperationExtractVariable:          "refactor.extract.variable",
		backend.OperationExtractConstant:          "refactor.extract.constant",
		backend.OperationInlineVariable:           "refactor.inline",
		backend.OperationInlineFunction:           "refactor.inline",
		backend.OperationImplementContract:        "source.overrideMethods",
		backend.OperationChangeSignature:          "refactor.change.signature",
		backend.OperationIntroduceParameter:       "refactor.change.signature",
		backend.OperationIntroduceParameterObject: "refactor.change.signature",
		backend.OperationMoveConstruct:            "moveStaticMember",
		javaMakeStaticOperation:                   javaMakeStaticActionKind,
	}[request.Operation]
	if !validJavaCodeActionKind(request, wantKind) || javaCodeActionNeedsName(request.Operation) && strings.TrimSpace(request.NewName) == "" {
		return fmt.Errorf("unsupported Java code action request")
	}
	if request.Operation == backend.OperationMoveConstruct && (request.DestinationFile == "" || request.DestinationType == "") {
		return fmt.Errorf("java move_construct requires a destination file and type")
	}
	if request.Operation == backend.OperationIntroduceParameter && (request.Expression == "" || request.ParameterName == "") {
		return fmt.Errorf("java introduce_parameter requires an exact expression and parameter name")
	}
	if request.Operation == backend.OperationIntroduceParameterObject && (request.ParameterObjectType == "" || request.ParameterObjectMethod == "" || len(request.ParameterObjectFields) == 0 || len(request.Parameters) != 1) {
		return fmt.Errorf("java introduce_parameter_object requires a generated type, field mapping, one replacement parameter, and resolved call sites")
	}
	return nil
}

func javaCodeActionNeedsName(operation backend.Operation) bool {
	switch operation {
	case backend.OperationInlineVariable, backend.OperationInlineFunction, backend.OperationImplementContract,
		backend.OperationChangeSignature, backend.OperationIntroduceParameter, backend.OperationIntroduceParameterObject, backend.OperationMoveConstruct:
		return false
	case javaMakeStaticOperation:
		return false
	default:
		return true
	}
}

func (b *JavaBackend) rollbackJavaCodeAction(ctx context.Context, rollback *javaCodeActionRollback) error {
	if !rollback.staged || rollback.buffer == nil {
		return nil
	}
	rollback.buffer.Rollback(rollback.checkpoint)
	var failures []error
	for _, path := range rollback.files {
		content, err := sourcebuf.ReadFile(rollback.ctx, path)
		if err != nil {
			failures = append(failures, fmt.Errorf("read Java refactoring rollback content %s: %w", path, err))
			continue
		}
		if err := b.syncJavaDocumentLocked(ctx, path, content); err != nil {
			failures = append(failures, fmt.Errorf("restore Java refactoring overlay %s: %w", path, err))
			continue
		}
		if sourcebuf.FromContext(ctx) != nil {
			state := b.documents[pathutil.FileURI(path)]
			state.pendingSave = true
			b.documents[pathutil.FileURI(path)] = state
		}
	}
	return errors.Join(failures...)
}

// ApplyCodeAction applies a JDT LS refactoring through validated workspace edits.
func (b *JavaBackend) ApplyCodeAction(ctx context.Context, request backend.CodeActionRequest) (result *backend.CodeActionResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateJavaCodeActionRequest(request); err != nil {
		return nil, err
	}
	root, file, source, err := resolveJavaProject(ctx, request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: request.Operation, Language: backend.LanguageJava, Workspace: root}
	}
	if preimageErr := validateJavaCodeActionPreimage(source, request); preimageErr != nil {
		return nil, preimageErr
	}

	config := effectiveJavaConfig(request.Project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	rollback := &javaCodeActionRollback{}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, b.rollbackJavaCodeAction(ctx, rollback))
		}
	}()
	if sessionErr := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); sessionErr != nil {
		return nil, sessionErr
	}
	session := b.session
	buffer := sourcebuf.FromContext(ctx)
	if request.Operation == backend.OperationMoveConstruct && request.Kind == javaMoveResourceKind && buffer != nil {
		return nil, ErrJavaBatchResourceRenameUnsupported
	}
	if syncErr := b.syncJavaCodeActionDocuments(ctx, buffer, file, source); syncErr != nil {
		return nil, syncErr
	}
	uri := pathutil.FileURI(file)
	params, actionDiagnostics, err := javaCodeActionRequestParams(ctx, session, b.documents[uri].version, uri, file, request)
	if err != nil {
		return nil, err
	}
	var workspaceEdit json.RawMessage
	var resourceReferences []string
	workspaceEdit, resourceReferences, err = requestJavaCodeActionEdit(ctx, session, root, uri, params, request, actionDiagnostics)
	if err != nil {
		return nil, err
	}
	workspace, err := prepareJavaCodeActionWorkspace(ctx, root, file, uri, source, workspaceEdit, resourceReferences, buffer, request)
	if err != nil {
		return nil, err
	}
	workspace.organizeImports = len(workspace.renames) == 0
	workingBuffer, workingCtx, afterAction, ownedBuffer, err := b.stageJavaCodeAction(ctx, root, file, uri, source, buffer, session, &workspace, request, rollback)
	if err != nil {
		return nil, err
	}
	return b.finishJavaCodeAction(ctx, workingCtx, file, source, afterAction, session, workingBuffer, ownedBuffer, &workspace, rollback)
}

func (b *JavaBackend) syncJavaCodeActionDocuments(ctx context.Context, buffer *sourcebuf.Buffer, file string, source []byte) error {
	if buffer != nil {
		for _, change := range buffer.Changes() {
			if strings.EqualFold(filepath.Ext(change.Path), ".java") {
				if err := b.syncJavaDocumentLocked(ctx, change.Path, change.After); err != nil {
					return err
				}
			}
		}
	}
	if err := b.syncJavaDocumentLocked(ctx, file, source); err != nil {
		return err
	}
	return b.saveCommittedJavaDocumentsLocked(ctx)
}

func (b *JavaBackend) finishJavaCodeAction(ctx, workingCtx context.Context, file string, source, afterAction []byte, session JavaSession, buffer *sourcebuf.Buffer, ownedBuffer bool, workspace *javaWorkspaceEdit, rollback *javaCodeActionRollback) (*backend.CodeActionResult, error) {
	resultFiles := make([]backend.CodeActionFileEdit, 0, len(workspace.files))
	var selectedAfter []byte
	diagnostics := make([]backend.Diagnostic, 0)
	for _, fileEdit := range workspace.files {
		current := fileEdit.after
		if fileEdit.file == file {
			current = afterAction
		}
		verified, err := b.verifyJavaCodeActionFile(workingCtx, session, buffer, fileEdit, current, workspace.organizeImports)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, verified.diagnostics...)
		resultFiles = append(resultFiles, backend.CodeActionFileEdit{File: fileEdit.file, Before: bytes.Clone(fileEdit.before), After: bytes.Clone(verified.content)})
		if fileEdit.file == file {
			selectedAfter = bytes.Clone(verified.content)
		}
	}
	if selectedAfter == nil {
		return nil, fmt.Errorf("java code action did not retain selected-file output")
	}
	if err := b.publishJavaCodeAction(ctx, session, buffer, ownedBuffer, workspace, resultFiles, rollback); err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: source, After: selectedAfter, Files: resultFiles, Diagnostics: diagnostics}, nil
}

type javaCodeActionFileVerification struct {
	content     []byte
	diagnostics []backend.Diagnostic
}

func (b *JavaBackend) verifyJavaCodeActionFile(ctx context.Context, session JavaSession, buffer *sourcebuf.Buffer, fileEdit javaWorkspaceFileEdit, current []byte, organizeImports bool) (javaCodeActionFileVerification, error) {
	verified := javaVerifiedSource{content: bytes.Clone(current), version: b.documents[fileEdit.uri].version}
	if err := applyJavaFormattingRequest(ctx, session, fileEdit.file, verified.content, &verified); err != nil {
		return javaCodeActionFileVerification{}, fmt.Errorf("format Java refactoring result for %s: %w", fileEdit.file, err)
	}
	// JDT LS still indexes the old package until the resource rename is published.
	if organizeImports {
		if err := b.applyJavaOrganizeRequest(ctx, session, fileEdit.file, current, &verified); err != nil {
			return javaCodeActionFileVerification{}, fmt.Errorf("organize imports after Java refactoring for %s: %w", fileEdit.file, err)
		}
	}
	version, err := b.publishJavaVerifyEdits(ctx, session, fileEdit.file, current, verified, buffer)
	if err != nil {
		return javaCodeActionFileVerification{}, err
	}
	diagnostics, err := waitJavaVerifyDiagnostics(ctx, session, fileEdit.file, version)
	if err != nil {
		return javaCodeActionFileVerification{}, fmt.Errorf("verify Java refactoring result for %s: %w", fileEdit.file, err)
	}
	return javaCodeActionFileVerification{content: verified.content, diagnostics: diagnostics}, nil
}

func (b *JavaBackend) publishJavaCodeAction(ctx context.Context, session JavaSession, buffer *sourcebuf.Buffer, ownedBuffer bool, workspace *javaWorkspaceEdit, resultFiles []backend.CodeActionFileEdit, rollback *javaCodeActionRollback) error {
	if !ownedBuffer {
		return nil
	}
	if err := buffer.Flush(ctx); err != nil {
		return fmt.Errorf("publish Java code action workspace edit: %w", err)
	}
	rollback.staged = false
	if err := b.publishJavaCodeActionRenames(ctx, session, workspace, resultFiles); err != nil {
		return err
	}
	for _, fileEdit := range resultFiles {
		if err := b.syncJavaDocumentLocked(ctx, fileEdit.File, fileEdit.After); err != nil {
			return err
		}
	}
	return nil
}

func (b *JavaBackend) publishJavaCodeActionRenames(ctx context.Context, session JavaSession, workspace *javaWorkspaceEdit, resultFiles []backend.CodeActionFileEdit) error {
	for _, rename := range workspace.renames {
		resourceEdit := javaRenameEdit{files: workspace.files}
		if err := b.publishJavaResourceRename(ctx, session, rename, &resourceEdit); err != nil {
			return fmt.Errorf("publish Java resource move: %w", err)
		}
		workspace.files = resourceEdit.files
		for index := range resultFiles {
			if resultFiles[index].File == rename.oldPath {
				resultFiles[index].File = rename.newPath
			}
		}
	}
	return nil
}

func (b *JavaBackend) stageJavaCodeAction(ctx context.Context, root, file, uri string, source []byte, buffer *sourcebuf.Buffer, session JavaSession, workspace *javaWorkspaceEdit, request backend.CodeActionRequest, rollback *javaCodeActionRollback) (*sourcebuf.Buffer, context.Context, []byte, bool, error) {
	workingBuffer := buffer
	workingCtx := ctx
	ownedBuffer := false
	if workingBuffer == nil {
		var err error
		workingBuffer, err = sourcebuf.New(ctx, root)
		if err != nil {
			return nil, nil, nil, false, fmt.Errorf("create Java code action buffer: %w", err)
		}
		if err := workingBuffer.AuthorizeWrites(ctx, root); err != nil {
			return nil, nil, nil, false, fmt.Errorf("authorize Java code action buffer: %w", err)
		}
		workingCtx = sourcebuf.WithBuffer(ctx, workingBuffer)
		ownedBuffer = true
	}
	rollback.buffer = workingBuffer
	rollback.ctx = workingCtx
	rollback.checkpoint = workingBuffer.Checkpoint()
	rollback.files = make([]string, 0, len(workspace.files))
	changes := make([]sourcebuf.Change, 0, len(workspace.files))
	var afterAction []byte
	for _, fileEdit := range workspace.files {
		rollback.files = append(rollback.files, fileEdit.file)
		if err := b.syncJavaDocumentLocked(workingCtx, fileEdit.file, fileEdit.before); err != nil {
			return nil, nil, nil, false, err
		}
		current := b.documents[fileEdit.uri]
		if fileEdit.version != nil && current.version != *fileEdit.version {
			return nil, nil, nil, false, fmt.Errorf("java code action document version %d is stale; current version is %d", *fileEdit.version, current.version)
		}
		changes = append(changes, sourcebuf.Change{Path: fileEdit.file, Before: fileEdit.before, After: fileEdit.after})
		if fileEdit.file == file {
			afterAction = bytes.Clone(fileEdit.after)
		}
	}
	if err := workingBuffer.Apply(workingCtx, changes); err != nil {
		return nil, nil, nil, false, fmt.Errorf("stage Java code action workspace edit: %w", err)
	}
	rollback.staged = true
	for _, fileEdit := range workspace.files {
		if !bytes.Equal(fileEdit.before, fileEdit.after) {
			state := b.documents[fileEdit.uri]
			state.pendingSave = true
			b.documents[fileEdit.uri] = state
		}
	}
	if afterAction == nil {
		return nil, nil, nil, false, fmt.Errorf("java code action workspace edit omitted the selected file")
	}
	for _, fileEdit := range workspace.files {
		if err := b.syncJavaDocumentLocked(workingCtx, fileEdit.file, fileEdit.after); err != nil {
			return nil, nil, nil, false, err
		}
	}
	afterAction, err := b.postProcessJavaCodeAction(ctx, workingCtx, root, file, uri, source, afterAction, session, workingBuffer, workspace, request)
	if err != nil {
		return nil, nil, nil, false, err
	}
	return workingBuffer, workingCtx, afterAction, ownedBuffer, nil
}

func prepareJavaCodeActionWorkspace(ctx context.Context, root, file, uri string, source []byte, workspaceEdit json.RawMessage, resourceReferences []string, buffer *sourcebuf.Buffer, request backend.CodeActionRequest) (javaWorkspaceEdit, error) {
	if request.Operation == javaMakeTypeExplicitOperation {
		if err := validateJavaExplicitTypeEdit(source, uri, request, workspaceEdit); err != nil {
			return javaWorkspaceEdit{}, err
		}
	}
	moveResource := request.Operation == backend.OperationMoveConstruct && request.Kind == javaMoveResourceKind
	workspace, err := prepareJavaWorkspaceEdits(ctx, root, file, source, workspaceEdit, "", "", moveResource && buffer == nil)
	if err != nil {
		return javaWorkspaceEdit{}, err
	}
	if moveResource {
		if err := prepareJavaCodeActionResourceMove(ctx, root, file, uri, resourceReferences, request, &workspace); err != nil {
			return javaWorkspaceEdit{}, err
		}
	}
	if len(workspace.renames) == 0 {
		return workspace, nil
	}
	if !moveResource || len(workspace.renames) != 1 {
		return javaWorkspaceEdit{}, fmt.Errorf("%w: unsupported Java resource operation", ErrJavaRenameInvalidEdit)
	}
	if filepath.Dir(workspace.renames[0].newPath) != filepath.Dir(request.DestinationFile) {
		return javaWorkspaceEdit{}, fmt.Errorf("%w: JDT LS moved the Java type outside the requested destination package", ErrJavaRenameInvalidEdit)
	}
	return workspace, nil
}

func prepareJavaCodeActionResourceMove(ctx context.Context, root, file, uri string, references []string, request backend.CodeActionRequest, workspace *javaWorkspaceEdit) error {
	if request.DestinationPackage == "" {
		return fmt.Errorf("java top-level type move requires a resolved destination package")
	}
	if err := prepareJavaResourceMoveFiles(ctx, root, file, request.DestinationPackage, references, workspace); err != nil {
		return err
	}
	if len(workspace.renames) == 0 {
		newPath := filepath.Join(filepath.Dir(request.DestinationFile), filepath.Base(file))
		rename := javaResourceRename{oldPath: file, newPath: newPath, oldURI: uri, newURI: pathutil.FileURI(newPath)}
		if err := validateJavaResourceMove(root, file, workspace.files, rename); err != nil {
			return err
		}
		workspace.renames = append(workspace.renames, rename)
	}
	if len(workspace.renames) != 1 || workspace.renames[0].newPath != filepath.Join(filepath.Dir(request.DestinationFile), filepath.Base(file)) {
		return fmt.Errorf("JDT LS Java resource move returned an unexpected resource rename")
	}
	return nil
}

func javaCodeActionRequestParams(ctx context.Context, session JavaSession, version int, uri, file string, request backend.CodeActionRequest) (map[string]any, []backend.Diagnostic, error) {
	var diagnostics []backend.Diagnostic
	if request.Operation == backend.OperationImplementContract {
		var err error
		diagnostics, err = waitJavaVerifyDiagnostics(ctx, session, file, version)
		if err != nil {
			return nil, nil, fmt.Errorf("wait for Java contract diagnostics before generating methods: %w", err)
		}
	}
	lspDiagnostics := make([]map[string]any, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		item := map[string]any{"message": diagnostic.Message, "severity": diagnostic.Severity}
		if diagnostic.Location != nil {
			item["range"] = diagnostic.Location.Range
		}
		lspDiagnostics = append(lspDiagnostics, item)
	}
	contextParams := map[string]any{"diagnostics": lspDiagnostics}
	if request.Operation != backend.OperationImplementContract {
		contextParams["only"] = []string{request.Kind}
	}
	return map[string]any{
		lspTextDocumentKey: map[string]any{"uri": uri, "version": version},
		"range":            request.Range,
		"context":          contextParams,
	}, diagnostics, nil
}

func (b *JavaBackend) postProcessJavaCodeAction(ctx, workingCtx context.Context, root, file, uri string, source, afterAction []byte, session JavaSession, buffer *sourcebuf.Buffer, workspace *javaWorkspaceEdit, request backend.CodeActionRequest) ([]byte, error) {
	if request.Operation == backend.OperationIntroduceParameter {
		updated, err := b.replaceJavaIntroducedParameterExpression(ctx, root, file, afterAction, request)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(updated, afterAction) {
			if err := buffer.Apply(workingCtx, []sourcebuf.Change{{Path: file, Before: afterAction, After: updated}}); err != nil {
				return nil, fmt.Errorf("stage Java introduced parameter expression: %w", err)
			}
			afterAction = updated
			updateJavaWorkspaceFile(workspace, file, updated)
			if err := b.syncJavaDocumentLocked(workingCtx, file, updated); err != nil {
				return nil, err
			}
		}
	}
	if request.Operation == backend.OperationIntroduceParameterObject {
		updated, err := b.rewriteJavaParameterObjectCallers(ctx, workingCtx, root, file, afterAction, buffer, workspace, request)
		if err != nil {
			return nil, err
		}
		afterAction = updated
	}
	if isJavaExtractionOperation(request.Operation) {
		updated, err := b.renameJavaExtractedValue(ctx, workingCtx, root, file, uri, source, afterAction, session, buffer, workspace, request)
		if err != nil {
			return nil, err
		}
		afterAction = updated
	}
	return afterAction, nil
}

func updateJavaWorkspaceFile(workspace *javaWorkspaceEdit, file string, content []byte) {
	for index := range workspace.files {
		if workspace.files[index].file == file {
			workspace.files[index].after = bytes.Clone(content)
			return
		}
	}
}

func isJavaExtractionOperation(operation backend.Operation) bool {
	return operation == backend.OperationExtractFunction || operation == backend.OperationExtractVariable || operation == backend.OperationExtractConstant
}

func (b *JavaBackend) rewriteJavaParameterObjectCallers(ctx, workingCtx context.Context, root, file string, afterAction []byte, buffer *sourcebuf.Buffer, workspace *javaWorkspaceEdit, request backend.CodeActionRequest) ([]byte, error) {
	callSites, err := javaParameterObjectCallSiteMap(request)
	if err != nil {
		return nil, err
	}
	updates := make([]sourcebuf.Change, 0, len(workspace.files))
	for index := range workspace.files {
		fileEdit := &workspace.files[index]
		updated, err := b.rewriteJavaParameterObjectFile(ctx, root, file, fileEdit, callSites, request)
		if err != nil {
			return nil, err
		}
		if fileEdit.file == file {
			afterAction = updated
		}
		if !bytes.Equal(updated, fileEdit.after) {
			updates = append(updates, sourcebuf.Change{Path: fileEdit.file, Before: fileEdit.after, After: updated})
			fileEdit.after = bytes.Clone(updated)
		}
	}
	if len(callSites) != 0 {
		return nil, fmt.Errorf("JDT LS omitted %d resolved Java parameter-object caller files", len(callSites))
	}
	if err := applyJavaParameterObjectUpdates(workingCtx, buffer, updates); err != nil {
		return nil, err
	}
	for _, update := range updates {
		if err := b.syncJavaDocumentLocked(workingCtx, update.Path, update.After); err != nil {
			return nil, err
		}
	}
	return afterAction, nil
}

func javaParameterObjectCallSiteMap(request backend.CodeActionRequest) (map[string][][]string, error) {
	callSites := make(map[string][][]string, len(request.ParameterObjectCallSites))
	for _, callSite := range request.ParameterObjectCallSites {
		if callSite.File == "" || len(callSite.Arguments) != len(request.ParameterObjectFields) {
			return nil, fmt.Errorf("java parameter-object call site has an invalid file or argument count")
		}
		file := filepath.Clean(callSite.File)
		callSites[file] = append(callSites[file], callSite.Arguments)
	}
	return callSites, nil
}

func (b *JavaBackend) rewriteJavaParameterObjectFile(ctx context.Context, root, selectedFile string, fileEdit *javaWorkspaceFileEdit, callSites map[string][][]string, request backend.CodeActionRequest) ([]byte, error) {
	updated := bytes.Clone(fileEdit.after)
	file := filepath.Clean(fileEdit.file)
	if arguments, exists := callSites[file]; exists {
		var err error
		updated, err = javacst.ReplaceJavaMethodCallsWithParameterObject(updated, request.ParameterObjectMethod, request.ParameterObjectType, arguments)
		if err != nil {
			return nil, fmt.Errorf("rewrite Java parameter-object callers in %s: %w", fileEdit.file, err)
		}
		delete(callSites, file)
	}
	if fileEdit.file == selectedFile {
		var err error
		updated, err = b.replaceJavaParameterObjectReferences(ctx, root, selectedFile, updated, request)
		if err != nil {
			return nil, err
		}
	}
	if request.Operation == backend.OperationIntroduceParameterObject {
		restored, err := restoreJavaImportsFromPreimage(updated, fileEdit.before)
		if err != nil {
			return nil, fmt.Errorf("restore Java parameter-object imports in %s: %w", fileEdit.file, err)
		}
		updated = restored
	}
	return updated, nil
}

func applyJavaParameterObjectUpdates(ctx context.Context, buffer *sourcebuf.Buffer, updates []sourcebuf.Change) error {
	if len(updates) == 0 {
		return nil
	}
	if err := buffer.Apply(ctx, updates); err != nil {
		return fmt.Errorf("stage Java parameter-object references and callers: %w", err)
	}
	return nil
}

func (b *JavaBackend) renameJavaExtractedValue(ctx, workingCtx context.Context, root, file, uri string, source, afterAction []byte, session JavaSession, buffer *sourcebuf.Buffer, workspace *javaWorkspaceEdit, request backend.CodeActionRequest) ([]byte, error) {
	start, end, err := javaExtractionRange(source, request.Range)
	if err != nil {
		return nil, err
	}
	expression := source[start:end]
	generatedName, nameStart, err := javaExtractedValueName(afterAction, expression, request.Operation)
	if err != nil {
		return nil, err
	}
	if generatedName == request.NewName {
		return afterAction, nil
	}
	return b.applyJavaExtractedValueRename(ctx, workingCtx, root, file, uri, afterAction, generatedName, nameStart, session, buffer, workspace, request.NewName)
}

func javaExtractionRange(source []byte, selection backend.Range) (int, int, error) {
	start, err := javaByteOffset(source, javaPosition{Line: selection.Start.Line, Character: selection.Start.Character})
	if err != nil {
		return 0, 0, err
	}
	end, err := javaByteOffset(source, javaPosition{Line: selection.End.Line, Character: selection.End.Character})
	if err != nil || end <= start {
		return 0, 0, fmt.Errorf("invalid Java extraction expression range")
	}
	return start, end, nil
}

func javaExtractedValueName(afterAction, expression []byte, operation backend.Operation) (string, int, error) {
	var name string
	var start int
	var err error
	if operation == backend.OperationExtractFunction {
		name, start, _, err = javacst.ExtractedMethodIdentifier(afterAction, expression)
	} else {
		name, start, _, err = javacst.ExtractedLocalIdentifier(afterAction, expression)
	}
	if err != nil {
		return "", 0, fmt.Errorf("locate JDT LS extracted value: %w", err)
	}
	return name, start, nil
}

func (b *JavaBackend) applyJavaExtractedValueRename(ctx, workingCtx context.Context, root, file, uri string, afterAction []byte, generatedName string, nameStart int, session JavaSession, buffer *sourcebuf.Buffer, workspace *javaWorkspaceEdit, newName string) ([]byte, error) {
	position := backend.PositionFromByteOffset(afterAction, nameStart)
	prepared, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}, "position": position})
	if err != nil || !validJavaPrepareRename(prepared) {
		if err == nil {
			err = ErrJavaRenameInvalidEdit
		}
		return nil, fmt.Errorf("prepare Java extracted-value rename: %w", err)
	}
	renameRaw, err := session.Request(ctx, "textDocument/rename", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}, "position": position, "newName": newName})
	if err != nil {
		return nil, fmt.Errorf("rename Java extracted value: %w", err)
	}
	workspaceRename, err := prepareJavaWorkspaceEdits(workingCtx, root, file, afterAction, renameRaw, generatedName, newName, false)
	if err != nil {
		return nil, fmt.Errorf("validate Java extracted-value rename: %w; workspace edit: %s", err, renameRaw)
	}
	if len(workspaceRename.files) != 1 || workspaceRename.files[0].file != file {
		return nil, fmt.Errorf("java extracted-value rename modified unexpected files")
	}
	updated := bytes.Clone(workspaceRename.files[0].after)
	if err := buffer.Apply(workingCtx, []sourcebuf.Change{{Path: file, Before: afterAction, After: updated}}); err != nil {
		return nil, fmt.Errorf("stage Java extracted-value rename: %w", err)
	}
	updateJavaWorkspaceFile(workspace, file, updated)
	if err := b.syncJavaDocumentLocked(workingCtx, file, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func requestJavaCodeActionEdit(ctx context.Context, session JavaSession, root, uri string, params map[string]any, request backend.CodeActionRequest, diagnostics []backend.Diagnostic) (json.RawMessage, []string, error) {
	var workspaceEdit json.RawMessage
	var resourceReferences []string
	switch request.Operation {
	case backend.OperationMoveConstruct:
		moveDestination := any(request.DestinationType)
		if request.Kind == javaMoveResourceKind {
			var err error
			moveDestination, err = javaMoveResourcePackageNode(root, filepath.Dir(request.DestinationFile))
			if err != nil {
				return nil, nil, err
			}
			resourceReferences, err = javaMoveReferenceFiles(ctx, session, root, uri, request.Range.Start)
			if err != nil {
				return nil, nil, err
			}
		}
		moveRequest := map[string]any{"moveKind": request.Kind, "updateReferences": true, "destination": moveDestination}
		if request.Kind == javaMoveResourceKind {
			moveRequest["sourceUris"] = []string{uri}
		} else {
			moveRequest["params"] = params
		}
		raw, err := session.Request(ctx, "java/move", moveRequest)
		if err != nil {
			return nil, nil, fmt.Errorf("request Java move refactoring: %w", err)
		}
		var response struct {
			Edit         json.RawMessage `json:"edit"`
			ErrorMessage string          `json:"errorMessage"`
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, nil, fmt.Errorf("decode Java move refactoring response: %w", err)
		}
		if response.ErrorMessage != "" {
			return nil, nil, fmt.Errorf("JDT LS move refactoring: %s", response.ErrorMessage)
		}
		workspaceEdit = response.Edit
	case backend.OperationImplementContract:
		raw, err := session.Request(ctx, "java/listOverridableMethods", params)
		if err != nil {
			return nil, nil, fmt.Errorf("list Java contract methods: %w", err)
		}
		workspaceEdit, err = prepareJavaOverrideMethodsEdit(ctx, session, params, raw)
		if err != nil {
			messages := make([]string, 0, len(diagnostics))
			for _, diagnostic := range diagnostics {
				messages = append(messages, diagnostic.Message)
			}
			return nil, nil, fmt.Errorf("%w; Java diagnostics: %s", err, strings.Join(messages, "; "))
		}
	default:
		var err error
		workspaceEdit, err = requestJavaRefactoringEdit(ctx, session, params, request)
		if err != nil {
			return nil, nil, err
		}
	}
	return workspaceEdit, resourceReferences, nil
}

func requestJavaRefactoringEdit(ctx context.Context, session JavaSession, params map[string]any, request backend.CodeActionRequest) (json.RawMessage, error) {
	raw, err := session.Request(ctx, "textDocument/codeAction", params)
	if err != nil {
		return nil, fmt.Errorf("request Java %s code action: %w", request.Kind, err)
	}
	if request.Operation == javaMakeTypeExplicitOperation {
		return selectJavaExplicitTypeEdit(raw, request)
	}
	var action javaCodeAction
	if request.Operation == javaMakeStaticOperation {
		action, err = selectJavaCodeActionTitle(raw, javaMakeStaticActionKind, javaMakeStaticActionTitle)
	} else {
		action, err = selectJavaCodeAction(raw, request.Kind)
	}
	if err != nil {
		return nil, err
	}
	if len(action.Edit) != 0 && string(action.Edit) != "null" {
		return action.Edit, nil
	}
	if request.Operation == backend.OperationChangeSignature || request.Operation == backend.OperationIntroduceParameterObject {
		return prepareJavaChangeSignatureEdit(ctx, session, params, action, request.Parameters)
	}
	if action.Command == nil || action.Command.Command == "" {
		return nil, fmt.Errorf("JDT LS code action has neither an edit nor a command")
	}
	requester, ok := session.(javaApplyEditRequester)
	if !ok {
		return nil, fmt.Errorf("JDT LS session cannot capture workspace/applyEdit")
	}
	var captured json.RawMessage
	commandResult, err := requester.RequestWithApplyEdit(ctx, "workspace/executeCommand", map[string]any{
		"command": action.Command.Command, "arguments": action.Command.Arguments,
	}, func(edit json.RawMessage) (bool, error) {
		if len(captured) != 0 {
			return false, fmt.Errorf("JDT LS code action emitted more than one workspace edit")
		}
		captured = bytes.Clone(edit)
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("execute Java %s code action: %w", request.Kind, err)
	}
	var status struct {
		Applied       bool   `json:"applied"`
		FailureReason string `json:"failureReason"`
	}
	if len(commandResult) > 0 && json.Unmarshal(commandResult, &status) == nil && !status.Applied && status.FailureReason != "" {
		return nil, fmt.Errorf("JDT LS rejected workspace/applyEdit: %s", status.FailureReason)
	}
	if len(captured) == 0 {
		return nil, fmt.Errorf("JDT LS code action completed without a workspace/applyEdit")
	}
	return captured, nil
}

func javaMoveResourcePackageNode(root, destinationDir string) (json.RawMessage, error) {
	root = backend.CanonicalWorkspaceRoot(root)
	destinationDir = backend.CanonicalWorkspaceRoot(destinationDir)
	if !pathutil.PathWithin(root, destinationDir) {
		return nil, fmt.Errorf("java move package %q is outside workspace root %q", destinationDir, root)
	}
	relative, err := filepath.Rel(root, destinationDir)
	if err != nil || !filepath.IsLocal(relative) {
		return nil, fmt.Errorf("derive Java move package path for %q: %w", destinationDir, err)
	}
	projectName := filepath.Base(root)
	for projectDir := destinationDir; pathutil.PathWithin(root, projectDir); projectDir = filepath.Dir(projectDir) {
		if _, err := os.Stat(filepath.Join(projectDir, "pom.xml")); err == nil {
			projectName = filepath.Base(projectDir)
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect Java move package project %q: %w", projectDir, err)
		}
		if projectDir == root {
			break
		}
	}
	return json.Marshal(map[string]any{
		"displayName":            filepath.Base(destinationDir),
		"uri":                    pathutil.FileURI(destinationDir),
		"path":                   "/" + filepath.ToSlash(relative),
		"project":                projectName,
		"isDefaultPackage":       false,
		"isParentOfSelectedFile": false,
	})
}

func javaMoveReferenceFiles(ctx context.Context, session JavaSession, root, sourceURI string, position backend.Position) ([]string, error) {
	response, err := session.Request(ctx, "textDocument/references", map[string]any{
		lspTextDocumentKey: map[string]string{"uri": sourceURI},
		"position":         position,
		"context":          map[string]any{"includeDeclaration": false},
	})
	if err != nil {
		return nil, fmt.Errorf("find Java type references before move: %w", err)
	}
	var locations []struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(response, &locations); err != nil {
		return nil, fmt.Errorf("decode Java type references before move: %w", err)
	}
	files := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		path, err := pathutil.FilePathFromURI(location.URI)
		if err != nil {
			return nil, fmt.Errorf("decode Java type reference URI %q: %w", location.URI, err)
		}
		path = backend.CanonicalWorkspaceRoot(path)
		if !pathutil.PathWithin(root, path) || !strings.EqualFold(filepath.Ext(path), ".java") {
			return nil, fmt.Errorf("java type reference %q is outside the workspace or is not Java", path)
		}
		files[path] = struct{}{}
	}
	result := make([]string, 0, len(files))
	for path := range files {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func prepareJavaResourceMoveFiles(ctx context.Context, root, sourceFile, destinationPackage string, referenceFiles []string, workspace *javaWorkspaceEdit) error {
	selected := findJavaWorkspaceFile(workspace, sourceFile)
	if selected == nil {
		return fmt.Errorf("java resource move edit omitted selected file %q", sourceFile)
	}
	oldPackage, err := javacst.JavaPackageName(selected.before)
	if err != nil {
		return fmt.Errorf("read Java source package before move: %w", err)
	}
	if destinationPackage == oldPackage {
		return fmt.Errorf("java resource move destination resolves to the source package %q", oldPackage)
	}
	selected.after, err = javacst.ReplaceJavaPackageName(selected.after, destinationPackage)
	if err != nil {
		return fmt.Errorf("update Java source package for move: %w", err)
	}
	typeName := strings.TrimSuffix(filepath.Base(sourceFile), filepath.Ext(sourceFile))
	move := javaTypeMoveImports{
		typeName: typeName, oldPackage: oldPackage, newPackage: destinationPackage,
		oldName: qualifiedJavaTypeName(oldPackage, typeName), newName: qualifiedJavaTypeName(destinationPackage, typeName),
	}
	paths, err := javaMoveReferencePaths(root, referenceFiles, workspace.files)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if path == sourceFile {
			continue
		}
		if err := updateJavaMoveReference(ctx, path, workspace, move); err != nil {
			return err
		}
	}
	sort.Slice(workspace.files, func(i, j int) bool { return workspace.files[i].file < workspace.files[j].file })
	return nil
}

type javaTypeMoveImports struct {
	typeName   string
	oldPackage string
	newPackage string
	oldName    string
	newName    string
}

func findJavaWorkspaceFile(workspace *javaWorkspaceEdit, file string) *javaWorkspaceFileEdit {
	for index := range workspace.files {
		if workspace.files[index].file == file {
			return &workspace.files[index]
		}
	}
	return nil
}

func qualifiedJavaTypeName(packageName, typeName string) string {
	if packageName == "" {
		return typeName
	}
	return packageName + "." + typeName
}

func javaMoveReferencePaths(root string, referenceFiles []string, files []javaWorkspaceFileEdit) ([]string, error) {
	paths := make(map[string]struct{}, len(referenceFiles)+len(files))
	for _, path := range referenceFiles {
		paths[path] = struct{}{}
	}
	for _, fileEdit := range files {
		paths[fileEdit.file] = struct{}{}
	}
	allJavaFiles, err := javaSourceFiles(root, root)
	if err != nil {
		return nil, fmt.Errorf("list Java files for moved-type references: %w", err)
	}
	for _, path := range allJavaFiles {
		paths[path] = struct{}{}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	return ordered, nil
}

func updateJavaMoveReference(ctx context.Context, path string, workspace *javaWorkspaceEdit, move javaTypeMoveImports) error {
	existing := findJavaWorkspaceFile(workspace, path)
	var before, current []byte
	if existing == nil {
		var err error
		before, err = sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return fmt.Errorf("read Java type reference file %q: %w", path, err)
		}
		current = before
	} else {
		before, current = existing.before, existing.after
	}
	callerPackage, err := javacst.JavaPackageName(current)
	if err != nil {
		return fmt.Errorf("read Java caller package in %q: %w", path, err)
	}
	if !javaFileReferencesMovedType(current, callerPackage, move) && !javaFileReferencesMovedType(before, callerPackage, move) {
		return nil
	}
	addImports, removeImports, err := javaMoveImportDelta(path, callerPackage, move)
	if err != nil {
		return err
	}
	updated, err := applyJavaExplicitImports(current, addImports, removeImports)
	if err != nil {
		return fmt.Errorf("update Java moved-type import in %q: %w", path, err)
	}
	if existing != nil {
		existing.after = updated
		return nil
	}
	if bytes.Equal(before, updated) {
		return nil
	}
	workspace.files = append(workspace.files, javaWorkspaceFileEdit{file: path, uri: pathutil.FileURI(path), before: bytes.Clone(before), after: updated})
	return nil
}

func javaFileReferencesMovedType(source []byte, callerPackage string, move javaTypeMoveImports) bool {
	return move.oldPackage != "" && (callerPackage == move.oldPackage || bytes.Contains(source, []byte("import "+move.oldName+";")))
}

func javaMoveImportDelta(path, callerPackage string, move javaTypeMoveImports) ([]string, []string, error) {
	addImports := make([]string, 0, 1)
	removeImports := make([]string, 0, 1)
	if move.oldPackage != "" {
		removeImports = append(removeImports, move.oldName)
	}
	if callerPackage == move.newPackage {
		return addImports, removeImports, nil
	}
	if move.newPackage == "" {
		return nil, nil, fmt.Errorf("cannot add a reference to default-package type %q from %q", move.typeName, path)
	}
	return append(addImports, move.newName), removeImports, nil
}

func (b *JavaBackend) replaceJavaIntroducedParameterExpression(ctx context.Context, root, file string, source []byte, request backend.CodeActionRequest) ([]byte, error) {
	if err := javacst.ValidateJavaIdentifier(request.ParameterName); err != nil {
		return nil, fmt.Errorf("invalid Java parameter name: %w", err)
	}
	methodRange, err := b.javaChangedMethodSourceRange(ctx, root, file, source, request.Symbol, len(request.Parameters), "", "resolve Java method after signature change", "java introduce_parameter resolves to %d changed method candidates", "java introduce_parameter method range is invalid")
	if err != nil {
		return nil, err
	}
	methodSource := source[methodRange.start:methodRange.end]
	candidate, _, err := javacst.ResolveExpression(methodSource, request.Expression, request.ExpressionPath)
	if err != nil {
		return nil, fmt.Errorf("resolve Java expression to replace with the new parameter: %w", err)
	}
	result := bytes.Clone(source)
	absoluteStart, absoluteEnd := methodRange.start+int(candidate.StartByte), methodRange.start+int(candidate.EndByte)
	result = append(result[:absoluteStart], append([]byte(request.ParameterName), result[absoluteEnd:]...)...)
	return result, nil
}

func (b *JavaBackend) replaceJavaParameterObjectReferences(ctx context.Context, root, file string, source []byte, request backend.CodeActionRequest) ([]byte, error) {
	methodRange, err := b.javaChangedMethodSourceRange(ctx, root, file, source, request.Symbol, 1, request.ParameterObjectType, "resolve Java method after parameter-object signature change", "java introduce_parameter_object resolves to %d changed method candidates for parameter type %q", "java parameter-object method range is invalid")
	if err != nil {
		return nil, err
	}
	fields := make([]javacst.JavaParameterField, 0, len(request.ParameterObjectFields))
	for _, field := range request.ParameterObjectFields {
		fields = append(fields, javacst.JavaParameterField{SourceIndex: field.SourceIndex, ParameterName: field.ParameterName, Name: field.Name, Type: field.Type})
	}
	updatedMethod, err := javacst.RewriteJavaParameterReferences(source[methodRange.start:methodRange.end], request.Parameters[0].Name, fields)
	if err != nil {
		return nil, fmt.Errorf("rewrite Java parameter-object body references: %w", err)
	}
	updated := make([]byte, 0, len(source)-methodRange.end+methodRange.start+len(updatedMethod))
	updated = append(updated, source[:methodRange.start]...)
	updated = append(updated, updatedMethod...)
	updated = append(updated, source[methodRange.end:]...)
	return updated, nil
}

type javaChangedMethodSourceRange struct{ start, end int }

func (b *JavaBackend) javaChangedMethodSourceRange(ctx context.Context, root, file string, source []byte, query string, parameterCount int, parameterType, resolveContext, countError, rangeError string) (javaChangedMethodSourceRange, error) {
	symbols, err := b.readJavaSymbolsLocked(ctx, root, file, source)
	if err != nil {
		return javaChangedMethodSourceRange{}, fmt.Errorf("%s: %w", resolveContext, err)
	}
	readSymbols, err := mapJavaReadSymbols(symbols, nil, file)
	if err != nil {
		return javaChangedMethodSourceRange{}, err
	}
	parts, _, _, err := parseJavaSymbolQuery(query)
	if err != nil {
		return javaChangedMethodSourceRange{}, err
	}
	candidates, err := matchJavaReadSymbols(strings.Join(parts, "."), readSymbols)
	if err != nil {
		return javaChangedMethodSourceRange{}, err
	}
	return selectJavaChangedMethodSourceRange(source, candidates, parameterCount, parameterType, countError, rangeError)
}

func selectJavaChangedMethodSourceRange(source []byte, candidates []backend.ReadSymbol, parameterCount int, parameterType, countError, rangeError string) (javaChangedMethodSourceRange, error) {
	var target backend.ReadSymbol
	count := 0
	for _, candidate := range candidates {
		if candidate.Kind != "method" || javaSignatureParameterCount(candidate.Signature) != parameterCount {
			continue
		}
		if parameterType != "" && !javaSignatureHasParameterType(candidate.Signature, parameterType) {
			continue
		}
		target, count = candidate, count+1
	}
	if count != 1 {
		if strings.Contains(countError, "%q") {
			return javaChangedMethodSourceRange{}, fmt.Errorf(countError, count, parameterType)
		}
		return javaChangedMethodSourceRange{}, fmt.Errorf(countError, count)
	}
	start, err := javacst.ByteOffsetFromPosition(source, target.Range.Start)
	if err != nil {
		return javaChangedMethodSourceRange{}, err
	}
	end, err := javacst.ByteOffsetFromPosition(source, target.Range.End)
	if err != nil || end <= start {
		return javaChangedMethodSourceRange{}, fmt.Errorf("%s", rangeError)
	}
	return javaChangedMethodSourceRange{start: start, end: end}, nil
}

func javaSignatureParameterCount(signature *string) int {
	if signature == nil || strings.TrimSpace(*signature) == "" {
		return 0
	}
	depth, count := 0, 1
	for _, r := range *signature {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

func javaSignatureHasParameterType(signature *string, parameterType string) bool {
	if signature == nil || parameterType == "" {
		return false
	}
	for parameter := range strings.SplitSeq(strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(*signature), ")"), "(")), ",") {
		fields := strings.Fields(strings.TrimSpace(parameter))
		if len(fields) == 0 {
			continue
		}
		typeName := strings.TrimSuffix(fields[0], "...")
		if typeName == parameterType || strings.HasSuffix(typeName, "."+parameterType) {
			return true
		}
	}
	return false
}

func prepareJavaOverrideMethodsEdit(ctx context.Context, session JavaSession, codeActionParams map[string]any, raw json.RawMessage) (json.RawMessage, error) {
	var response struct {
		Type    string `json:"type"`
		Methods []struct {
			BindingKey    string   `json:"bindingKey"`
			Name          string   `json:"name"`
			Parameters    []string `json:"parameters"`
			Unimplemented bool     `json:"unimplemented"`
		} `json:"methods"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode JDT LS overridable methods: %w", err)
	}
	selected := make([]map[string]any, 0, len(response.Methods))
	for _, method := range response.Methods {
		if !method.Unimplemented {
			continue
		}
		if method.BindingKey == "" || method.Name == "" {
			return nil, fmt.Errorf("JDT LS returned an unimplemented method without a binding key or name")
		}
		selected = append(selected, map[string]any{
			"bindingKey": method.BindingKey, "name": method.Name, "parameters": method.Parameters,
			"unimplemented": method.Unimplemented,
		})
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("JDT LS found no unimplemented methods for Java type %q: %s", response.Type, raw)
	}
	edit, err := session.Request(ctx, "java/addOverridableMethods", map[string]any{
		"context": codeActionParams, "overridableMethods": selected,
	})
	if err != nil {
		return nil, fmt.Errorf("generate Java contract methods: %w", err)
	}
	if len(edit) == 0 || string(edit) == "null" {
		return nil, fmt.Errorf("JDT LS returned no edit for Java contract methods")
	}
	return edit, nil
}

func selectJavaCodeAction(raw json.RawMessage, kind string) (javaCodeAction, error) {
	var actions []javaCodeAction
	if json.Unmarshal(raw, &actions) != nil {
		return javaCodeAction{}, fmt.Errorf("malformed JDT LS code action response")
	}
	matches, disabledTitles := javaCodeActionMatches(actions, kind)
	if selected, ok := preferredJavaCodeAction(matches); ok {
		return selected, nil
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 && len(disabledTitles) > 0 {
		return javaCodeAction{}, fmt.Errorf("JDT LS returned only disabled actions of kind %q: %s", kind, strings.Join(disabledTitles, ", "))
	}
	if len(actions) == 0 {
		return javaCodeAction{}, fmt.Errorf("JDT LS returned no code actions")
	}
	if len(matches) == 0 {
		return javaCodeAction{}, fmt.Errorf("JDT LS returned no enabled actions of kind %q; available actions: %s", kind, strings.Join(javaEnabledActionTitles(actions), ", "))
	}
	return javaCodeAction{}, fmt.Errorf("JDT LS returned %d enabled actions of kind %q: %s", len(matches), kind, strings.Join(javaActionTitles(matches), ", "))
}

func selectJavaCodeActionTitle(raw json.RawMessage, kind, title string) (javaCodeAction, error) {
	var actions []javaCodeAction
	if json.Unmarshal(raw, &actions) != nil {
		return javaCodeAction{}, fmt.Errorf("malformed JDT LS code action response")
	}
	var matches []javaCodeAction
	var disabled bool
	for _, action := range actions {
		if action.Title != title || action.Kind != kind {
			continue
		}
		if javaCodeActionDisabled(action) {
			disabled = true
			continue
		}
		matches = append(matches, action)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return javaCodeAction{}, fmt.Errorf("JDT LS returned %d enabled actions titled %q of kind %q", len(matches), title, kind)
	}
	if disabled {
		return javaCodeAction{}, fmt.Errorf("JDT LS returned only disabled actions titled %q of kind %q", title, kind)
	}
	return javaCodeAction{}, fmt.Errorf("JDT LS returned no enabled action titled %q of kind %q", title, kind)
}

func javaCodeActionMatches(actions []javaCodeAction, kind string) ([]javaCodeAction, []string) {
	var matches []javaCodeAction
	var disabledTitles []string
	for _, action := range actions {
		if action.Kind != kind {
			continue
		}
		if javaCodeActionDisabled(action) {
			disabledTitles = append(disabledTitles, action.Title)
			continue
		}
		matches = append(matches, action)
	}
	return matches, disabledTitles
}

func preferredJavaCodeAction(matches []javaCodeAction) (javaCodeAction, bool) {
	if len(matches) <= 1 {
		return javaCodeAction{}, false
	}
	var selected javaCodeAction
	count := 0
	for _, action := range matches {
		if strings.Contains(strings.ToLower(action.Title), "replace all occurrences") {
			continue
		}
		selected = action
		count++
	}
	return selected, count == 1
}

func javaCodeActionDisabled(action javaCodeAction) bool {
	disabled := bytes.TrimSpace(action.Disabled)
	return len(disabled) > 0 && !bytes.Equal(disabled, []byte("null"))
}

func javaActionTitles(actions []javaCodeAction) []string {
	titles := make([]string, 0, len(actions))
	for _, action := range actions {
		titles = append(titles, fmt.Sprintf("%s (%s)", action.Title, action.Kind))
	}
	return titles
}

func javaEnabledActionTitles(actions []javaCodeAction) []string {
	var enabled []javaCodeAction
	for _, action := range actions {
		if !javaCodeActionDisabled(action) {
			enabled = append(enabled, action)
		}
	}
	return javaActionTitles(enabled)
}

type javaChangeSignatureInfo struct {
	MethodIdentifier string `json:"methodIdentifier"`
	MethodName       string `json:"methodName"`
	Modifier         string `json:"modifier"`
	ReturnType       string `json:"returnType"`
	Parameters       []struct {
		OriginalIndex int    `json:"originalIndex"`
		Name          string `json:"name"`
		Type          string `json:"type"`
	} `json:"parameters"`
	Exceptions []string `json:"exceptions"`
}

func prepareJavaChangeSignatureEdit(ctx context.Context, session JavaSession, contextParams map[string]any, action javaCodeAction, requested []backend.CodeActionParameter) (json.RawMessage, error) {
	if action.Command == nil || action.Command.Command == "" || len(requested) == 0 {
		return nil, fmt.Errorf("JDT LS change-signature action is missing its command or ordered parameters")
	}
	var info javaChangeSignatureInfo
	if len(action.Command.Arguments) > 2 && string(action.Command.Arguments[2]) != "null" {
		if err := json.Unmarshal(action.Command.Arguments[2], &info); err != nil {
			return nil, fmt.Errorf("decode JDT LS change-signature information: %w", err)
		}
	} else {
		raw, err := session.Request(ctx, "java/getChangeSignatureInfo", contextParams)
		if err != nil {
			return nil, fmt.Errorf("request JDT LS change-signature information: %w", err)
		}
		if err := json.Unmarshal(raw, &info); err != nil {
			return nil, fmt.Errorf("decode JDT LS change-signature information: %w", err)
		}
	}
	if info.MethodIdentifier == "" || info.MethodName == "" || info.ReturnType == "" {
		return nil, fmt.Errorf("JDT LS returned incomplete change-signature information")
	}
	parameters, err := javaChangeSignatureParameters(info, requested)
	if err != nil {
		return nil, err
	}
	commandName := "changeSignature"
	if len(action.Command.Arguments) > 0 {
		var provided string
		if json.Unmarshal(action.Command.Arguments[0], &provided) == nil && provided != "" {
			commandName = provided
		}
	}
	params := map[string]any{
		"command":          commandName,
		"context":          contextParams,
		"options":          map[string]any{"tabSize": 4, "insertSpaces": true},
		"commandArguments": []any{info.MethodIdentifier, false, info.MethodName, info.Modifier, info.ReturnType, parameters, info.Exceptions, false},
	}
	raw, err := session.Request(ctx, "java/getRefactorEdit", params)
	if err != nil {
		return nil, fmt.Errorf("request JDT LS change-signature edit: %w", err)
	}
	var response struct {
		Edit         json.RawMessage `json:"edit"`
		ErrorMessage string          `json:"errorMessage"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode JDT LS change-signature edit: %w", err)
	}
	if response.ErrorMessage != "" {
		return nil, fmt.Errorf("JDT LS rejected change-signature edit: %s", response.ErrorMessage)
	}
	if len(response.Edit) == 0 || string(response.Edit) == "null" {
		return nil, fmt.Errorf("JDT LS returned no change-signature workspace edit")
	}
	return response.Edit, nil
}

func javaChangeSignatureParameters(info javaChangeSignatureInfo, requested []backend.CodeActionParameter) ([]map[string]any, error) {
	existing := make(map[int]bool, len(info.Parameters))
	for _, parameter := range info.Parameters {
		existing[parameter.OriginalIndex] = true
	}
	parameters := make([]map[string]any, 0, len(requested))
	used := make(map[int]bool, len(requested))
	nextIndex := len(info.Parameters)
	for index, parameter := range requested {
		if strings.TrimSpace(parameter.Name) == "" || strings.TrimSpace(parameter.Type) == "" {
			return nil, fmt.Errorf("change-signature parameter %d requires a name and type", index)
		}
		originalIndex := nextIndex
		entry := map[string]any{"name": parameter.Name, "type": parameter.Type}
		if parameter.SourceIndex != nil {
			originalIndex = *parameter.SourceIndex
			if !existing[originalIndex] || used[originalIndex] {
				return nil, fmt.Errorf("change-signature source_index %d is missing or duplicated", originalIndex)
			}
			used[originalIndex] = true
		} else {
			if strings.TrimSpace(parameter.Argument) == "" {
				return nil, fmt.Errorf("new change-signature parameter %q requires an argument expression", parameter.Name)
			}
			entry["defaultValue"] = parameter.Argument
			nextIndex++
		}
		entry["originalIndex"] = originalIndex
		parameters = append(parameters, entry)
	}
	return parameters, nil
}

func validJavaCodeActionKind(request backend.CodeActionRequest, expected string) bool {
	if expected == "" {
		return false
	}
	if request.Operation == backend.OperationMoveConstruct {
		return request.Kind == "moveStaticMember" || request.Kind == "moveResource"
	}
	if request.Operation == javaMakeStaticOperation {
		return request.Kind == javaMakeStaticActionKind
	}
	return request.Kind == expected
}

func validateJavaCodeActionPreimage(source []byte, request backend.CodeActionRequest) error {
	if request.ExpectedRevision != "" {
		digest := sha256.Sum256(source)
		if request.ExpectedRevision != hex.EncodeToString(digest[:]) {
			return fmt.Errorf("java code action source revision changed")
		}
	}
	if request.Operation == javaMakeTypeExplicitOperation {
		if _, selectionErr := javacst.JavaInferredLocalAt(source, request.Symbol, request.Range); selectionErr != nil {
			return fmt.Errorf("resolve Java inferred local before native action: %w", selectionErr)
		}
	}
	return nil
}
