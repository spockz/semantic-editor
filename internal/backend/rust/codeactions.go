// Package rust captures the bounded rust-analyzer extraction assist without invoking client commands.
// WHY: This file resolves one local extraction and composes its requested rename before publishing.
package rust

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const rustLSPDiagnosticsKey = "diagnostics"

type rustCodeAction struct {
	Title    string          `json:"title"`
	Kind     string          `json:"kind"`
	Disabled json.RawMessage `json:"disabled"`
	Edit     json.RawMessage `json:"edit"`
	Data     json.RawMessage `json:"data"`
	Command  json.RawMessage `json:"command"`
}

type rustCodeActionData struct {
	ID string `json:"id"`
}

type rustCodeActionDocument struct {
	TextDocument struct {
		URI     string `json:"uri"`
		Version *int   `json:"version"`
	} `json:"textDocument"`
	Edits []json.RawMessage `json:"edits"`
}

// ApplyCodeAction validates a bounded rust-analyzer extraction assist before staging its source edit.
func (b *RustBackend) ApplyCodeAction(ctx context.Context, request backend.CodeActionRequest) (*backend.CodeActionResult, error) {
	if request.Operation == backend.OperationPromoteLocalToConst {
		return b.applyRustPromoteLocalToConst(ctx, request)
	}
	if request.Operation == backend.OperationExtractVariable && request.Kind == "constant" {
		return b.applyRustExtractConstant(ctx, request)
	}
	if request.Operation == backend.OperationInlineFunction {
		return b.applyRustInlineFunction(ctx, request)
	}
	if request.Operation == backend.OperationExtractFunction {
		return b.applyRustExtractFunction(ctx, request)
	}
	if request.Operation == backend.OperationInlineVariable {
		return b.applyRustInlineVariable(ctx, request)
	}
	return b.applyRustExtractVariable(ctx, request)
}

func runRustDocumentLifecycle(ctx context.Context, b *RustBackend, session RustSession, file string, source []byte, transform func(context.Context, string) ([]byte, error)) (updated []byte, returnErr error) {
	uri := pathutil.FileURI(file)
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{lspTextDocumentKey: map[string]any{"uri": uri, rustLanguageIDKey: rustLanguageID, "version": 1, "text": string(source)}}); err != nil {
		return nil, &RustError{Op: "didOpen", File: file, Err: err}
	}
	opened := true
	defer func() {
		if opened {
			if err := session.Notify(context.WithoutCancel(ctx), "textDocument/didClose", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}}); err != nil {
				returnErr = errors.Join(returnErr, &RustError{Op: "didClose", File: file, Err: err}, b.closeSessionLocked())
			}
		}
	}()
	if statusSource, ok := session.(rustStatusNotificationSource); ok {
		if err := waitForRustAnalyzerReady(ctx, statusSource.LatestServerStatus, statusSource.ServerStatuses()); err != nil {
			return nil, &RustError{Op: "workspaceStatus", File: file, Err: err}
		}
	}
	updated, returnErr = transform(ctx, uri)
	if returnErr != nil {
		return nil, returnErr
	}
	if err := session.Notify(context.WithoutCancel(ctx), "textDocument/didClose", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}}); err != nil {
		opened = false
		return nil, errors.Join(&RustError{Op: "didClose", File: file, Err: err}, b.closeSessionLocked())
	}
	opened = false
	return updated, nil
}

func (b *RustBackend) applyRustExtractVariable(ctx context.Context, request backend.CodeActionRequest) (result *backend.CodeActionResult, returnErr error) {
	if err := validateRustExtractRequest(request); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveRustProject(ctx, request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: request.Operation, Language: backend.LanguageRust, Workspace: root}
	}
	if request.ExpectedRevision != "" && request.ExpectedRevision != rustSourceRevision(source) {
		return nil, fmt.Errorf("rust extraction source revision changed")
	}
	if rustHasUnresolvedNameExpansion(source) {
		return nil, fmt.Errorf("rust extraction refuses files with unresolved imported or macro-expanded names because introduced-name conflicts cannot be proven")
	}
	if rustIdentifierPresent(source, request.NewName) {
		return nil, fmt.Errorf("requested rust local name %q already appears in the selected file; refusing possible shadowing conflict", request.NewName)
	}
	selectionStart, selectionEnd, insertionStart, err := resolveRustExtractSelection(source, request.Range)
	if err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureRustSessionLocked(ctx, root); sessionErr != nil {
		return nil, sessionErr
	}
	session := b.session
	updated, err := runRustDocumentLifecycle(ctx, b, session, file, source, func(ctx context.Context, uri string) ([]byte, error) {
		return computeRustExtraction(ctx, session, file, uri, source, request, selectionStart, selectionEnd, insertionStart)
	})
	if err != nil {
		return nil, err
	}
	if err := stageRustExtraction(ctx, root, file, source, updated); err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: source, After: updated, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(updated)}}, Diagnostics: nil}, nil
}

func computeRustExtraction(ctx context.Context, session RustSession, file, uri string, source []byte, request backend.CodeActionRequest, selectionStart, selectionEnd, insertionStart int) ([]byte, error) {
	edits, err := requestRustExtractEdits(ctx, session, file, uri, request.Range)
	if err != nil {
		return nil, err
	}
	candidate, renamePosition, useSpan, bindingSpan, err := prepareRustExtractCandidate(source, edits, selectionStart, selectionEnd, insertionStart, request.Expression)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("validate extraction workspace edit: %w", err)}
	}
	if err := session.Notify(ctx, "textDocument/didChange", map[string]any{
		lspTextDocumentKey: map[string]any{"uri": uri, "version": 2},
		"contentChanges":   []map[string]string{{"text": string(candidate)}},
	}); err != nil {
		return nil, &RustError{Op: "didChange", File: file, Err: err}
	}
	return renameRustExtractBinding(ctx, session, file, uri, candidate, renamePosition, useSpan, bindingSpan, request.NewName)
}

func validateRustExtractRequest(request backend.CodeActionRequest) error {
	if request.Operation != backend.OperationExtractVariable || request.Kind != rustRefactorExtract || !isSafeRustBindingIdentifier(request.NewName) || request.Range.End.Line < request.Range.Start.Line || request.Range.End.Line == request.Range.Start.Line && request.Range.End.Character <= request.Range.Start.Character {
		return fmt.Errorf("unsupported rust code action request")
	}
	return nil
}

func resolveRustExtractSelection(source []byte, selected backend.Range) (int, int, int, error) {
	selectionStart, err := rustByteOffset(source, rustPosition(selected.Start))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("resolve rust extraction selection: %w", err)
	}
	selectionEnd, err := rustByteOffset(source, rustPosition(selected.End))
	if err != nil || selectionEnd <= selectionStart {
		return 0, 0, 0, fmt.Errorf("invalid rust extraction selection")
	}
	insertionStart, safe := rustSelectionIsSafeExpressionStatement(source, selectionStart, selectionEnd)
	if !safe {
		return 0, 0, 0, fmt.Errorf("rust extraction currently requires a suffixed integer expression as the sole argument of a direct single-argument call statement")
	}
	return selectionStart, selectionEnd, insertionStart, nil
}

func stageRustExtraction(ctx context.Context, root, file string, before, after []byte) error {
	return sourcebuf.ApplyAndPublish(ctx, root, "rust extraction", []sourcebuf.Change{{Path: file, Before: before, After: after}})
}

func requestRustExtractEdits(ctx context.Context, session RustSession, file, uri string, selected backend.Range) ([]rustTextEdit, error) {
	actionRaw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{
		lspTextDocumentKey: map[string]string{"uri": uri},
		"range":            selected,
		"context":          map[string]any{rustLSPDiagnosticsKey: []any{}, "only": []string{rustRefactorExtract}},
	})
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	action, encoded, err := selectRustExtractVariableAction(actionRaw)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	if rustCodeActionDisabled(action.Disabled) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("rust-analyzer marked extract-variable action disabled")}
	}
	if hasRustCommand(action.Command) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("command-based rust extraction actions are unsupported")}
	}
	if len(action.Edit) == 0 || string(action.Edit) == "null" {
		if len(action.Data) == 0 || string(action.Data) == "null" {
			return nil, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		resolved, resolveErr := session.Request(ctx, rustOpResolveAction, encoded)
		if resolveErr != nil {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: resolveErr}
		}
		original := action
		if decodeErr := json.Unmarshal(resolved, &action); decodeErr != nil {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		if rustCodeActionDisabled(action.Disabled) {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: fmt.Errorf("rust-analyzer resolved extract-variable action as disabled")}
		}
		if action.Kind != original.Kind || !sameRustCodeActionIdentity(action.Data, original.Data, "extract_variable:") || hasRustCommand(action.Command) || len(action.Edit) == 0 || string(action.Edit) == "null" {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
	}
	edits, err := decodeRustSingleFileWorkspaceEdit(file, action.Edit, 1)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("decode extraction workspace edit: %w", err)}
	}
	return edits, nil
}

func renameRustExtractBinding(ctx context.Context, session RustSession, file, uri string, candidate []byte, position backend.Position, useSpan, bindingSpan rustEditSpan, newName string) ([]byte, error) {
	preparedRaw, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}, "position": position})
	if err != nil {
		return nil, &RustError{Op: rustOpPrepareRename, File: file, Err: err}
	}
	prepared, ok := rustPrepareRenameRange(preparedRaw)
	if !ok || prepared.Start.Line != position.Line || prepared.Start.Character != position.Character || prepared.End != (rustPosition{Line: position.Line, Character: position.Character + useSpan.end - useSpan.start}) {
		return nil, &RustError{Op: rustOpPrepareRename, File: file, Err: ErrRustRenameInvalidEdit}
	}
	nameStart, err := rustByteOffset(candidate, prepared.Start)
	if err != nil {
		return nil, &RustError{Op: rustOpPrepareRename, File: file, Err: ErrRustRenameInvalidEdit}
	}
	nameEnd, err := rustByteOffset(candidate, prepared.End)
	if err != nil || nameStart != useSpan.start || nameEnd != useSpan.end || nameEnd <= nameStart {
		return nil, &RustError{Op: rustOpPrepareRename, File: file, Err: ErrRustRenameInvalidEdit}
	}
	oldName := string(candidate[nameStart:nameEnd])
	if !isSimpleRustIdentifier(oldName) {
		return nil, &RustError{Op: rustOpPrepareRename, File: file, Err: ErrRustRenameInvalidEdit}
	}
	if oldName == newName {
		return candidate, nil
	}
	renameRaw, err := session.Request(ctx, "textDocument/rename", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}, "position": position, "newName": newName})
	if err != nil {
		return nil, &RustError{Op: "rename", File: file, Err: err}
	}
	renameEdits, decodeErr := decodeRustSingleFileWorkspaceEdit(file, renameRaw, 2)
	containsUse := containsRustTextEdit(renameEdits, prepared, oldName, newName)
	containsBinding := containsRustBindingRename(candidate, renameEdits, bindingSpan, oldName, newName)
	if decodeErr != nil || len(renameEdits) != 2 || !containsUse || !containsBinding {
		return nil, &RustError{Op: "rename", File: file, Err: fmt.Errorf("%w (decode: %w, edit count: %d, use edit: %t, binding edit: %t)", ErrRustRenameInvalidEdit, decodeErr, len(renameEdits), containsUse, containsBinding)}
	}
	updated, err := applyRustTextEdits(candidate, renameEdits, oldName)
	if err != nil {
		return nil, &RustError{Op: "rename", File: file, Err: err}
	}
	return updated, nil
}

type rustStatusNotificationSource interface {
	ServerStatuses() <-chan json.RawMessage
	LatestServerStatus() json.RawMessage
}

func waitForRustAnalyzerReady(ctx context.Context, latest func() json.RawMessage, statuses <-chan json.RawMessage) error {
	readyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if ready, err := parseRustAnalyzerStatus(latest()); ready || err != nil {
		return err
	}
	for {
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("waiting for rust-analyzer workspace readiness: %w", readyCtx.Err())
		case _, ok := <-statuses:
			if !ok {
				return fmt.Errorf("rust-analyzer server-status stream closed before workspace readiness")
			}
			ready, err := parseRustAnalyzerStatus(latest())
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		}
	}
}

func parseRustAnalyzerStatus(raw json.RawMessage) (bool, error) {
	if len(raw) == 0 {
		return false, nil
	}
	var status struct {
		Health    string `json:"health"`
		Quiescent bool   `json:"quiescent"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		return false, fmt.Errorf("decode rust-analyzer server status: %w", err)
	}
	if status.Health == "error" || status.Quiescent && status.Health != "ok" {
		if status.Message != "" {
			return false, fmt.Errorf("rust-analyzer workspace is not healthy: %s", status.Message)
		}
		return false, fmt.Errorf("rust-analyzer workspace is not healthy (health %q)", status.Health)
	}
	return status.Health == "ok" && status.Quiescent, nil
}

func (b *RustBackend) ensureRustSessionLocked(ctx context.Context, root string) error {
	if b.session != nil {
		if b.root != root {
			return &RustError{Op: rustOpCodeAction, Workspace: root, Err: ErrRustSessionConflict}
		}
		return nil
	}
	if b.factory == nil {
		return &RustError{Op: "start", Workspace: root, Err: ErrRustAnalyzerUnavailable}
	}
	session, err := b.factory(ctx, root)
	if err != nil {
		return &RustError{Op: "start", Workspace: root, Err: err}
	}
	if session == nil {
		return &RustError{Op: "start", Workspace: root, Err: ErrRustAnalyzerUnavailable}
	}
	b.session, b.root = session, root
	if err := initializeRustSession(ctx, session, root); err != nil {
		closeErr := b.closeSessionLocked()
		return &RustError{Op: "initialize", Workspace: root, Err: errors.Join(err, closeErr)}
	}
	return nil
}

func selectRustExtractVariableAction(raw json.RawMessage) (rustCodeAction, json.RawMessage, error) {
	return selectUniqueRustCodeAction(raw, "extract-variable", func(action rustCodeAction) bool {
		var data rustCodeActionData
		return json.Unmarshal(action.Data, &data) == nil &&
			action.Kind == rustRefactorExtract &&
			strings.HasPrefix(data.ID, "extract_variable:")
	})
}

func hasRustCommand(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func rustCodeActionDisabled(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func decodeRustSingleFileWorkspaceEdit(file string, raw json.RawMessage, expectedVersion int) ([]rustTextEdit, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, ErrRustRenameInvalidEdit
	}
	if err := validateRustChangeAnnotations(object); err != nil {
		return nil, err
	}
	for _, forbidden := range []string{"resourceOperations"} {
		if _, exists := object[forbidden]; exists {
			return nil, ErrRustRenameInvalidEdit
		}
	}
	_, hasChanges := object["changes"]
	documentsRaw, hasDocuments := object["documentChanges"]
	if hasChanges == hasDocuments {
		return nil, ErrRustRenameInvalidEdit
	}
	if hasChanges {
		if expectedVersion != 0 {
			return nil, ErrRustRenameInvalidEdit
		}
		var changes map[string]json.RawMessage
		if json.Unmarshal(object["changes"], &changes) != nil || len(changes) != 1 {
			return nil, ErrRustRenameInvalidEdit
		}
		for uri, editsRaw := range changes {
			path, err := pathutil.FilePathFromURI(uri)
			if err != nil || path != file {
				return nil, ErrRustRenameInvalidEdit
			}
			return decodeRustTextEdits(editsRaw)
		}
	}
	var documents []rustCodeActionDocument
	if json.Unmarshal(documentsRaw, &documents) != nil || len(documents) != 1 || documents[0].TextDocument.URI == "" || documents[0].TextDocument.Version == nil || *documents[0].TextDocument.Version != expectedVersion {
		return nil, ErrRustRenameInvalidEdit
	}
	path, err := pathutil.FilePathFromURI(documents[0].TextDocument.URI)
	if err != nil || path != file {
		return nil, ErrRustRenameInvalidEdit
	}
	encoded, err := json.Marshal(documents[0].Edits)
	if err != nil {
		return nil, fmt.Errorf("encode Rust code action edits: %w", err)
	}
	return decodeRustTextEdits(encoded)
}

func prepareRustExtractCandidate(source []byte, edits []rustTextEdit, selectionStart, selectionEnd, expectedInsertionStart int, expression string) ([]byte, backend.Position, rustEditSpan, rustEditSpan, error) {
	if len(edits) != 2 {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
	}
	if candidate, position, useSpan, bindingSpan, matched, err := prepareRustStatementRewriteCandidate(source, edits, selectionStart, selectionEnd, expectedInsertionStart, expression); matched {
		return candidate, position, useSpan, bindingSpan, err
	}
	var selected, insertion *rustTextEdit
	var selectedSpan, insertSpan rustEditSpan
	for i := range edits {
		start, err := rustByteOffset(source, edits[i].Range.Start)
		if err != nil {
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
		}
		end, err := rustByteOffset(source, edits[i].Range.End)
		if err != nil || end < start {
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
		}
		switch {
		case start == selectionStart && end == selectionEnd:
			selected, selectedSpan = &edits[i], rustEditSpan{start: start, end: end, text: edits[i].NewText}
		case start == end:
			insertion, insertSpan = &edits[i], rustEditSpan{start: start, end: end, text: edits[i].NewText}
		default:
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
		}
	}
	if selected == nil || insertion == nil || strings.TrimSpace(expression) == "" {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
	}
	generatedName := selected.NewText
	if !isSimpleRustIdentifier(generatedName) {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
	}
	prefix := "let " + generatedName + " = "
	inserted := insertion.NewText
	if !strings.HasPrefix(inserted, prefix) {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
	}
	initializer := strings.TrimLeft(inserted[len(prefix):], " \t\r\n")
	if !strings.HasPrefix(initializer, expression) || strings.TrimSpace(strings.TrimPrefix(initializer, expression)) != ";" {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
	}
	if insertSpan.start != expectedInsertionStart || insertSpan.start >= selectionStart {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, ErrRustRenameInvalidEdit
	}
	insertionStart := insertSpan.start
	updated := applyRustEditSpans(source, []rustEditSpan{selectedSpan, insertSpan})
	selectedOffset := selectionStart + len(insertSpan.text)
	position, err := rustPositionAtOffset(updated, selectedOffset)
	if err != nil {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, err
	}
	bindingStart := insertionStart + len("let ")
	binding := rustEditSpan{start: bindingStart, end: bindingStart + len(generatedName)}
	use := rustEditSpan{start: selectedOffset, end: selectedOffset + len(generatedName)}
	return updated, position, use, binding, nil
}

func prepareRustStatementRewriteCandidate(source []byte, edits []rustTextEdit, selectionStart, selectionEnd, statementStart int, expression string) ([]byte, backend.Position, rustEditSpan, rustEditSpan, bool, error) {
	if selectionEnd+2 > len(source) || string(source[selectionEnd:selectionEnd+2]) != ");" || statementStart < 0 || statementStart > selectionStart {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, false, nil
	}
	var replacement, insertion *rustTextEdit
	var replacementStart, replacementEnd, insertionStart, insertionEnd int
	for i := range edits {
		start, err := rustByteOffset(source, edits[i].Range.Start)
		if err != nil {
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
		}
		end, err := rustByteOffset(source, edits[i].Range.End)
		if err != nil || end < start {
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
		}
		if start == statementStart && end == selectionEnd+1 {
			replacement, replacementStart, replacementEnd = &edits[i], start, end
		} else if start == selectionEnd+2 && end == selectionEnd+2 {
			insertion, insertionStart, insertionEnd = &edits[i], start, end
		}
	}
	if replacement == nil || insertion == nil {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, false, nil
	}
	generatedName := strings.TrimPrefix(replacement.NewText, "let ")
	generatedName, hasAssignment := strings.CutSuffix(generatedName, " = "+expression)
	if !hasAssignment || !isSimpleRustIdentifier(generatedName) || string(source[statementStart:selectionStart])+expression+")" != string(source[replacementStart:replacementEnd]) {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
	}
	lineStart := bytes.LastIndexByte(source[:statementStart], '\n') + 1
	indent := source[lineStart:statementStart]
	if strings.TrimSpace(string(indent)) != "" {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
	}
	callPrefix := strings.TrimSpace(string(source[statementStart:selectionStart]))
	if !strings.HasSuffix(callPrefix, "(") {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
	}
	callee := strings.TrimSpace(strings.TrimSuffix(callPrefix, "("))
	if callee == "" || insertion.NewText != "\n"+string(indent)+callee+"("+generatedName+");" {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
	}
	updated := applyRustEditSpans(source, []rustEditSpan{
		{start: insertionStart, end: insertionEnd, text: insertion.NewText},
		{start: replacementStart, end: replacementEnd, text: replacement.NewText},
	})
	bindingStart := statementStart + len("let ")
	useStart := statementStart + len(replacement.NewText) + 1 + len("\n") + len(indent) + len(callee) + 1
	binding := rustEditSpan{start: bindingStart, end: bindingStart + len(generatedName)}
	use := rustEditSpan{start: useStart, end: useStart + len(generatedName)}
	if string(updated[binding.start:binding.end]) != generatedName || string(updated[use.start:use.end]) != generatedName {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, ErrRustRenameInvalidEdit
	}
	position, err := rustPositionAtOffset(updated, use.start)
	if err != nil {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, true, err
	}
	return updated, position, use, binding, true, nil
}

func applyRustTextEdits(source []byte, edits []rustTextEdit, oldName string) ([]byte, error) {
	spans := make([]rustEditSpan, len(edits))
	for i, edit := range edits {
		start, err := rustByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, ErrRustRenameInvalidEdit
		}
		end, err := rustByteOffset(source, edit.Range.End)
		if err != nil || end <= start || string(source[start:end]) != oldName {
			return nil, ErrRustRenameInvalidEdit
		}
		spans[i] = rustEditSpan{start: start, end: end, text: edit.NewText}
	}
	for i := range spans {
		for j := i + 1; j < len(spans); j++ {
			if spans[i].start < spans[j].end && spans[j].start < spans[i].end {
				return nil, ErrRustRenameInvalidEdit
			}
		}
	}
	for i := 1; i < len(spans); i++ {
		for j := i; j > 0 && spans[j].start > spans[j-1].start; j-- {
			spans[j], spans[j-1] = spans[j-1], spans[j]
		}
	}
	return applyRustEditSpans(source, spans), nil
}

func rustPositionAtOffset(source []byte, offset int) (backend.Position, error) {
	if offset < 0 || offset > len(source) || !utf8.Valid(source[:offset]) {
		return backend.Position{}, ErrRustRenameInvalidEdit
	}
	line, character := 0, 0
	for _, r := range string(source[:offset]) {
		switch {
		case r == '\n':
			line++
			character = 0
		case r > 0xffff:
			character += 2
		default:
			character++
		}
	}
	return backend.Position{Line: line, Character: character}, nil
}

func rustPrepareRenameRange(raw json.RawMessage) (rustRange, bool) {
	var direct rustRange
	if json.Unmarshal(raw, &direct) == nil && validPositionRangeShape(direct) {
		return direct, true
	}
	var wrapped struct {
		Range rustRange `json:"range"`
	}
	if json.Unmarshal(raw, &wrapped) == nil && validPositionRangeShape(wrapped.Range) {
		return wrapped.Range, true
	}
	return rustRange{}, false
}

func containsRustTextEdit(edits []rustTextEdit, expected rustRange, oldName, newName string) bool {
	for _, edit := range edits {
		if edit.Range == expected && edit.NewText == newName && oldName != newName {
			return true
		}
	}
	return false
}

func containsRustBindingRename(source []byte, edits []rustTextEdit, inserted rustEditSpan, oldName, newName string) bool {
	for _, edit := range edits {
		start, err := rustByteOffset(source, edit.Range.Start)
		end, endErr := rustByteOffset(source, edit.Range.End)
		if err == nil && endErr == nil && start == inserted.start && end == inserted.end && string(source[start:end]) == oldName && edit.NewText == newName {
			return true
		}
	}
	return false
}

func isSimpleRustIdentifier(value string) bool {
	if value == "" || value == "_" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				return false
			}
			continue
		}
		if r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func isSafeRustBindingIdentifier(value string) bool {
	if !isSimpleRustIdentifier(value) || value[0] >= 'A' && value[0] <= 'Z' {
		return false
	}
	switch value {
	case "as", "async", "await", "break", "const", "continue", "crate", "dyn", "else", "enum", "extern", "false", "fn", "for", "gen", "if", "impl", "in", "let", "loop", "match", "mod", "move", "mut", "pub", "ref", "return", "self", "static", "struct", "super", "trait", "true", "type", "union", "unsafe", "use", "where", "while", "abstract", "become", "box", "do", "final", "macro", "override", "priv", "typeof", "unsized", "virtual", "yield":
		return false
	default:
		return true
	}
}

func rustIdentifierPresent(source []byte, name string) bool {
	source = []byte(norm.NFC.String(string(source)))
	name = norm.NFC.String(name)
	for offset := 0; offset+len(name) <= len(source); {
		index := bytes.Index(source[offset:], []byte(name))
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(name)
		if (start == 0 || !isRustIdentifierByte(source[start-1])) && (end == len(source) || !isRustIdentifierByte(source[end])) {
			return true
		}
		offset = start + 1
	}
	return false
}

func rustHasUnresolvedNameExpansion(source []byte) bool {
	text := string(source)
	return strings.Contains(text, "!") || strings.Contains(text, "#") || strings.Contains(text, "include") || strings.Contains(text, "use") && strings.Contains(text, "*")
}

func isRustIdentifierByte(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func rustSelectionIsSafeExpressionStatement(source []byte, start, end int) (int, bool) {
	if start < 0 || end <= start || end > len(source) {
		return 0, false
	}
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := len(source)
	if newline := bytes.IndexByte(source[end:], '\n'); newline >= 0 {
		lineEnd = end + newline
	}
	before := strings.TrimSpace(string(source[lineStart:start]))
	after := strings.TrimSpace(string(source[end:lineEnd]))
	expression := strings.TrimSpace(string(source[start:end]))
	if !strings.HasSuffix(before, "(") || after != ");" || !rustSuffixedIntegerExpression(expression) {
		return 0, false
	}
	callee := strings.TrimSpace(strings.TrimSuffix(before, "("))
	if callee == "" {
		return 0, false
	}
	for component := range strings.SplitSeq(callee, "::") {
		if !isSimpleRustIdentifier(component) {
			return 0, false
		}
	}
	statementStart := lineStart
	for statementStart < len(source) && (source[statementStart] == ' ' || source[statementStart] == '\t') {
		statementStart++
	}
	return statementStart, true
}

func rustSuffixedIntegerExpression(expression string) bool {
	expectsOperand := true
	depth := 0
	for index := 0; index < len(expression); {
		value := expression[index]
		switch value {
		case ' ', '\t', '\r', '\n':
			index++
			continue
		}
		if expectsOperand {
			switch value {
			case '+', '-':
				index++
				continue
			case '(':
				depth++
				index++
				continue
			}
			next, ok := scanRustSuffixedIntegerLiteral(expression, index)
			if !ok {
				return false
			}
			index = next
			expectsOperand = false
			continue
		}
		switch value {
		case ')':
			if depth == 0 {
				return false
			}
			depth--
			index++
		case '+', '-', '*', '/', '%':
			expectsOperand = true
			index++
		default:
			return false
		}
	}
	return !expectsOperand && depth == 0
}

func scanRustSuffixedIntegerLiteral(expression string, start int) (int, bool) {
	if expression[start] < '0' || expression[start] > '9' {
		return start, false
	}
	end := start
	for end < len(expression) && (expression[end] == '_' || expression[end] >= '0' && expression[end] <= '9' || expression[end] >= 'a' && expression[end] <= 'z' || expression[end] >= 'A' && expression[end] <= 'Z') {
		end++
	}
	parts := strings.Split(expression[start:end], "_")
	if len(parts) != 2 || parts[0] == "" || !rustIntegerSuffix(parts[1]) {
		return start, false
	}
	for _, digit := range parts[0] {
		if digit < '0' || digit > '9' {
			return start, false
		}
	}
	return end, true
}

func rustIntegerSuffix(suffix string) bool {
	switch suffix {
	case "i8", "i16", "i32", "i64", "i128", "isize", "u8", "u16", "u32", "u64", "u128", "usize":
		return true
	default:
		return false
	}
}

var _ backend.CodeActionBackend = (*RustBackend)(nil)

func sameRustCodeActionIdentity(actual, initial json.RawMessage, prefix string) bool {
	var actualData, initialData rustCodeActionData
	return prefix != "" &&
		json.Unmarshal(actual, &actualData) == nil &&
		json.Unmarshal(initial, &initialData) == nil &&
		actualData.ID != "" &&
		actualData.ID == initialData.ID &&
		strings.HasPrefix(actualData.ID, prefix)
}

func selectUniqueRustCodeAction(raw json.RawMessage, label string, matches func(rustCodeAction) bool) (rustCodeAction, json.RawMessage, error) {
	var encodedActions []json.RawMessage
	if json.Unmarshal(raw, &encodedActions) != nil || len(encodedActions) == 0 {
		return rustCodeAction{}, nil, ErrRustRenameInvalidEdit
	}
	var selected rustCodeAction
	var selectedRaw json.RawMessage
	for _, encoded := range encodedActions {
		var action rustCodeAction
		if json.Unmarshal(encoded, &action) != nil {
			return rustCodeAction{}, nil, ErrRustRenameInvalidEdit
		}
		if !matches(action) {
			continue
		}
		if selectedRaw != nil {
			return rustCodeAction{}, nil, fmt.Errorf("multiple Rust %s actions are available", label)
		}
		selected, selectedRaw = action, encoded
	}
	if selectedRaw == nil {
		return rustCodeAction{}, nil, fmt.Errorf("rust-analyzer did not return an %s action", label)
	}
	return selected, selectedRaw, nil
}
