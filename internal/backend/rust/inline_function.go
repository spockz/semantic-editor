// Package rust captures the bounded rust-analyzer inline-call assist.
// WHY: Only a typed literal-return substitution has a locally provable result without rewriting other callers.
package rust

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
)

var rustInlineIntegerFunction = regexp.MustCompile(`^fn ([A-Za-z_][A-Za-z0-9_]*)\(\)\s*->\s*(i8|i16|i32|i64|i128|isize|u8|u16|u32|u64|u128|usize)\s*\{\s*([0-9][0-9_]*_(?:i8|i16|i32|i64|i128|isize|u8|u16|u32|u64|u128|usize))\s*\}$`)

func (b *RustBackend) applyRustInlineFunction(ctx context.Context, request backend.CodeActionRequest) (*backend.CodeActionResult, error) {
	if err := validateRustInlineFunctionRequest(request); err != nil {
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
		return nil, fmt.Errorf("rust inline-call source revision changed")
	}
	expected, callStart, callEnd, err := expectedRustInlineCall(source, request)
	if err != nil {
		return nil, err
	}
	updated, err := executeRustInlineLifecycle(ctx, b, root, file, source, request, func(ctx context.Context, session RustSession, file string, source []byte, raw json.RawMessage) ([]byte, error) {
		return resolveRustInlineFunctionAction(ctx, session, file, source, raw, request.Symbol, callStart, callEnd, expected)
	})
	if err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: source, After: updated, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(updated)}}}, nil
}

func validateRustInlineFunctionRequest(request backend.CodeActionRequest) error {
	if request.Operation != backend.OperationInlineFunction || request.Kind != rustRefactorInline || !isSimpleRustIdentifier(request.Symbol) || !isSimpleRustIdentifier(request.EnclosingSymbol) || request.Expression == "" || !rustInlineIntegerFunction.MatchString("fn value() -> "+rustInlineLiteralSuffix(request.Expression)+" { "+request.Expression+" }") {
		return fmt.Errorf("unsupported rust inline-call request")
	}
	if request.Range.Start.Line != request.Range.End.Line || request.Range.Start.Character < 0 || request.Range.End.Character-request.Range.Start.Character != len(request.Symbol) {
		return fmt.Errorf("rust inline-call selection must identify one function name")
	}
	return nil
}

func rustInlineLiteralSuffix(literal string) string {
	_, after, ok := strings.CutLast(literal, "_")
	if !ok {
		return ""
	}
	return after
}

func expectedRustInlineCall(source []byte, request backend.CodeActionRequest) ([]byte, int, int, error) {
	start, err := rustByteOffset(source, rustPosition(request.Range.Start))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("resolve rust inline-call selection: %w", err)
	}
	end, err := rustByteOffset(source, rustPosition(request.Range.End))
	if err != nil || end-start != len(request.Symbol) || string(source[start:end]) != request.Symbol {
		return nil, 0, 0, fmt.Errorf("rust inline-call selection does not match the selected callee")
	}
	callEnd := end + 2
	if callEnd > len(source) || string(source[end:callEnd]) != "()" || start > 0 && isRustIdentifierByte(source[start-1]) {
		return nil, 0, 0, fmt.Errorf("rust inline-call selection must identify a direct zero-argument call")
	}
	if request.Expression == "" || request.Expression != strings.TrimSpace(request.Expression) {
		return nil, 0, 0, fmt.Errorf("rust inline-call replacement must be one exact typed literal")
	}
	updated := make([]byte, 0, len(source)-callEnd+start+len(request.Expression))
	updated = append(updated, source[:start]...)
	updated = append(updated, request.Expression...)
	updated = append(updated, source[callEnd:]...)
	return updated, start, callEnd, nil
}

func resolveRustInlineFunctionAction(ctx context.Context, session RustSession, file string, source []byte, raw json.RawMessage, name string, callStart, callEnd int, expected []byte) ([]byte, error) {
	action, encoded, err := selectRustInlineFunctionAction(raw, name)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	if rustCodeActionDisabled(action.Disabled) || hasRustCommand(action.Command) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("rust-analyzer inline-call action is disabled or command-based")}
	}
	if len(action.Edit) == 0 || string(action.Edit) == "null" {
		if len(action.Data) == 0 || string(action.Data) == "null" {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		resolvedRaw, resolveErr := session.Request(ctx, rustOpResolveAction, encoded)
		if resolveErr != nil {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: resolveErr}
		}
		var resolved rustCodeAction
		if json.Unmarshal(resolvedRaw, &resolved) != nil || resolved.Title != action.Title || resolved.Kind != action.Kind || len(resolved.Edit) == 0 || string(resolved.Edit) == "null" || rustCodeActionDisabled(resolved.Disabled) || hasRustCommand(resolved.Command) {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		if len(resolved.Data) != 0 && string(resolved.Data) != "null" && !sameRustCodeActionIdentity(resolved.Data, action.Data, "inline_call:RefactorInline:") {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		action = resolved
	}
	var editObject map[string]json.RawMessage
	if json.Unmarshal(action.Edit, &editObject) != nil || editObject == nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	if _, versioned := editObject["documentChanges"]; !versioned {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("rust inline-call requires a versioned single-document edit")}
	}
	edits, err := decodeRustWorkspaceEdits(file, action.Edit)
	if err != nil || len(edits) != 1 {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	editStart, startErr := rustByteOffset(source, edits[0].Range.Start)
	editEnd, endErr := rustByteOffset(source, edits[0].Range.End)
	if startErr != nil || endErr != nil || editStart != callStart || editEnd != callEnd {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("rust inline-call edit does not replace the selected call")}
	}
	spans, err := validateRustEditSpans(source, edits, string(source[callStart:callEnd]))
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	actual := applyRustEditSpans(source, spans)
	if !bytes.Equal(actual, expected) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("rust inline-call edit differs from the proven whole-file substitution")}
	}
	return actual, nil
}

func selectRustInlineFunctionAction(raw json.RawMessage, name string) (rustCodeAction, json.RawMessage, error) {
	wantTitle := "Inline `" + name + "`"
	return selectUniqueRustCodeAction(raw, "inline-call", func(action rustCodeAction) bool {
		var data rustCodeActionData
		return action.Title == wantTitle && action.Kind == rustRefactorInline &&
			json.Unmarshal(action.Data, &data) == nil && strings.HasPrefix(data.ID, "inline_call:RefactorInline:")
	})
}

func rustSourceRevision(source []byte) string {
	digest := sha256.Sum256(source)
	return hex.EncodeToString(digest[:])
}
