// Package rust captures a bounded rust-analyzer inline-local assist.
// WHY: Native inline edits can duplicate effects or alter inference, so only a proven literal substitution is published.
package rust

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
)

const rustRefactorInline = "refactor.inline"

func (b *RustBackend) applyRustInlineVariable(ctx context.Context, request backend.CodeActionRequest) (*backend.CodeActionResult, error) {
	if request.Operation != backend.OperationInlineVariable || request.Kind != rustRefactorInline || !isSimpleRustIdentifier(request.Symbol) {
		return nil, fmt.Errorf("unsupported rust inline-variable request")
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
		return nil, fmt.Errorf("rust inline source revision changed")
	}
	if rustHasUnresolvedNameExpansion(source) {
		return nil, fmt.Errorf("rust inline refuses files with unresolved imported or macro-expanded names because equivalence cannot be proven")
	}
	expected, err := prepareRustInlineCandidate(source, request.Range, request.Symbol)
	if err != nil {
		return nil, err
	}
	updated, err := executeRustInlineLifecycle(ctx, b, root, file, source, request, func(ctx context.Context, session RustSession, file string, source []byte, raw json.RawMessage) ([]byte, error) {
		return resolveRustInlineAction(ctx, session, file, source, raw, expected)
	})
	if err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: source, After: updated, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(updated)}}}, nil
}

func prepareRustInlineCandidate(source []byte, selected backend.Range, name string) ([]byte, error) {
	start, err := rustByteOffset(source, rustPosition(selected.Start))
	if err != nil {
		return nil, fmt.Errorf("resolve rust inline binding: %w", err)
	}
	end, err := rustByteOffset(source, rustPosition(selected.End))
	if err != nil || end-start != len(name) || string(source[start:end]) != name {
		return nil, fmt.Errorf("rust inline selection must identify the binding name")
	}
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := len(source)
	if n := bytes.IndexByte(source[start:], '\n'); n >= 0 {
		lineEnd = start + n
	}
	line := source[lineStart:lineEnd]
	indentEnd := len(line) - len(bytes.TrimLeft(line, " \t"))
	indent := line[:indentEnd]
	declaration := string(line[indentEnd:])
	prefix := "let " + name + " = "
	if !strings.HasPrefix(declaration, prefix) || !strings.HasSuffix(declaration, ";") || strings.Contains(declaration, "//") || strings.Contains(declaration, "/*") {
		return nil, fmt.Errorf("rust inline currently requires an untyped immutable let binding")
	}
	if start != lineStart+indentEnd+len("let ") {
		return nil, fmt.Errorf("rust inline selection is not the declaration binding")
	}
	value := strings.TrimSuffix(strings.TrimPrefix(declaration, prefix), ";")
	if !rustSuffixedIntegerLiteral(value) {
		return nil, fmt.Errorf("rust inline currently requires one explicitly suffixed integer literal")
	}
	useEnd, useLine, err := validateRustInlineUse(source, start, lineEnd, name)
	if err != nil {
		return nil, err
	}
	return rustInlineExpectedResult(source, lineStart, indent, useEnd, useLine, name, value), nil
}

func validateRustInlineUse(source []byte, start, lineEnd int, name string) (int, string, error) {
	if lineEnd == len(source) || lineEnd+1 >= len(source) {
		return 0, "", fmt.Errorf("rust inline requires one immediately following use")
	}
	useEnd := len(source)
	if n := bytes.IndexByte(source[lineEnd+1:], '\n'); n >= 0 {
		useEnd = lineEnd + 1 + n
	}
	useLine := string(source[lineEnd+1 : useEnd])
	if _, err := validateRustInlineCall(useLine, name, indentOfRustLine(source, start)); err != nil {
		return 0, "", err
	}
	for i := 0; i < len(source); {
		at := bytes.Index(source[i:], []byte(name))
		if at < 0 {
			break
		}
		at += i
		after := at + len(name)
		if (at == 0 || !isRustIdentifierByte(source[at-1])) && (after == len(source) || !isRustIdentifierByte(source[after])) && at != start && (at < lineEnd+1 || after > useEnd) {
			return 0, "", fmt.Errorf("rust inline refuses additional, shadowed, or non-local uses")
		}
		i = after
	}
	return useEnd, useLine, nil
}

func validateRustInlineCall(line, name string, indent []byte) (int, error) {
	trimmed := strings.TrimSpace(line)
	if strings.ContainsAny(trimmed, "!&") || !strings.HasSuffix(trimmed, ");") || !bytes.HasPrefix([]byte(line), indent) {
		return 0, fmt.Errorf("rust inline currently requires a direct argument use in the immediately following ordinary call")
	}
	call := strings.TrimSuffix(trimmed, ");")
	callee, args, hasArgs := strings.Cut(call, "(")
	if !hasArgs || strings.ContainsAny(args, "()") || !isSimpleRustIdentifier(callee) {
		return 0, fmt.Errorf("rust inline currently requires one ordinary function call with simple arguments")
	}
	uses := 0
	for argument := range strings.SplitSeq(args, ",") {
		argument = strings.TrimSpace(argument)
		if !isSimpleRustIdentifier(argument) {
			return 0, fmt.Errorf("rust inline currently requires simple identifier call arguments")
		}
		if argument == name {
			uses++
		}
	}
	if uses == 0 {
		return 0, fmt.Errorf("rust inline binding has no direct use")
	}
	return uses, nil
}

func indentOfRustLine(source []byte, offset int) []byte {
	lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
	line := source[lineStart:offset]
	return line[:len(line)-len(bytes.TrimLeft(line, " \t"))]
}

func rustInlineExpectedResult(source []byte, lineStart int, indent []byte, useEnd int, useLine, name, value string) []byte {
	updated := bytes.Clone(source)
	lineEnd := lineStart + len(bytes.SplitN(source[lineStart:], []byte("\n"), 2)[0])
	updated = append(updated[:lineStart], updated[lineEnd+1:]...)
	useStart := lineStart + len(indent)
	updatedUseLine := rewriteRustInlineArguments(strings.TrimPrefix(useLine, string(indent)), name, value)
	useLineLength := useEnd - lineEnd - 1
	updated = append(updated[:useStart], append(updatedUseLine, updated[lineStart+useLineLength:]...)...)
	return updated
}

func rewriteRustInlineArguments(line, name, value string) []byte {
	trimmed := strings.TrimSpace(line)
	prefixLength := len(line) - len(strings.TrimLeft(line, " \t"))
	trailingLength := len(line) - len(strings.TrimRight(line, " \t"))
	call := strings.TrimSuffix(trimmed, ");")
	callee, args, _ := strings.Cut(call, "(")
	parts := strings.Split(args, ",")
	for i, part := range parts {
		leading := len(part) - len(strings.TrimLeft(part, " \t"))
		trailing := len(part) - len(strings.TrimRight(part, " \t"))
		if strings.TrimSpace(part) == name {
			end := len(part) - trailing
			parts[i] = part[:leading] + value + part[end:]
		}
	}
	rewritten := callee + "(" + strings.Join(parts, ",") + ");"
	return []byte(line[:prefixLength] + rewritten + line[len(line)-trailingLength:])
}

func rustSuffixedIntegerLiteral(value string) bool {
	value = strings.ReplaceAll(value, "_", "")
	for _, suffix := range []string{"i8", "i16", "i32", "i64", "i128", "isize", "u8", "u16", "u32", "u64", "u128", "usize"} {
		if before, ok := strings.CutSuffix(value, suffix); ok {
			number := before
			if number == "" {
				return false
			}
			for _, r := range number {
				if r < '0' || r > '9' {
					return false
				}
			}
			return true
		}
	}
	return false
}

func resolveRustInlineAction(ctx context.Context, session RustSession, file string, source []byte, raw json.RawMessage, expected []byte) ([]byte, error) {
	action, err := selectRustInlineAction(ctx, session, file, raw)
	if err != nil {
		return nil, err
	}
	return verifyRustInlineEdits(file, source, action.Edit, expected)
}

func selectRustInlineAction(ctx context.Context, session RustSession, file string, raw json.RawMessage) (rustCodeAction, error) {
	var actions []json.RawMessage
	if json.Unmarshal(raw, &actions) != nil || len(actions) == 0 {
		return rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	var selected rustCodeAction
	var selectedRaw json.RawMessage
	for _, encoded := range actions {
		var action rustCodeAction
		if json.Unmarshal(encoded, &action) != nil {
			return rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		if action.Kind != rustRefactorInline || action.Title != "Inline variable" {
			continue
		}
		var data rustCodeActionData
		if len(action.Data) > 0 && string(action.Data) != "null" && json.Unmarshal(action.Data, &data) != nil {
			return rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		if data.ID == "inline_local_variable" || strings.HasPrefix(data.ID, "inline_local_variable:") {
			if selectedRaw != nil {
				return rustCodeAction{}, fmt.Errorf("multiple Rust inline-variable actions are available")
			}
			selected, selectedRaw = action, encoded
		}
	}
	if selectedRaw == nil || rustCodeActionDisabled(selected.Disabled) || hasRustCommand(selected.Command) {
		return rustCodeAction{}, fmt.Errorf("rust-analyzer did not provide an enabled inline-local action")
	}
	if len(selected.Edit) == 0 || string(selected.Edit) == "null" {
		return resolveRustInlineCodeAction(ctx, session, file, selected, selectedRaw)
	}
	return selected, nil
}

func resolveRustInlineCodeAction(ctx context.Context, session RustSession, file string, original rustCodeAction, encoded json.RawMessage) (rustCodeAction, error) {
	if len(original.Data) == 0 || string(original.Data) == "null" {
		return rustCodeAction{}, ErrRustRenameInvalidEdit
	}
	resolved, err := session.Request(ctx, rustOpResolveAction, encoded)
	if err != nil {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: err}
	}
	var action rustCodeAction
	if err := json.Unmarshal(resolved, &action); err != nil {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: fmt.Errorf("decode resolved inline action: %w", err)}
	}
	if rustCodeActionDisabled(action.Disabled) || action.Kind != rustRefactorInline || action.Title != "Inline variable" || hasRustCommand(action.Command) || len(action.Edit) == 0 || string(action.Edit) == "null" {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	// rust-analyzer omits the client-only action data in resolve responses, so correlate by the request and validate the full postimage.
	if len(action.Data) > 0 && string(action.Data) != "null" {
		var got, want rustCodeActionData
		if err := json.Unmarshal(action.Data, &got); err != nil {
			return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: fmt.Errorf("decode resolved inline action identity: %w", err)}
		}
		if err := json.Unmarshal(original.Data, &want); err != nil || got.ID == "" || got.ID != want.ID {
			return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
	}
	return action, nil
}

func verifyRustInlineEdits(file string, source []byte, raw json.RawMessage, expected []byte) ([]byte, error) {
	edits, err := decodeRustSingleFileWorkspaceEdit(file, raw, 1)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	spans := make([]rustEditSpan, len(edits))
	for i, edit := range edits {
		start, err := rustByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, ErrRustRenameInvalidEdit
		}
		end, err := rustByteOffset(source, edit.Range.End)
		if err != nil || end < start {
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
	actual := applyRustEditSpans(source, spans)
	if !bytes.Equal(actual, expected) {
		return nil, fmt.Errorf("rust-analyzer inline edit did not match the proven literal substitution")
	}
	return actual, nil
}

func executeRustInlineLifecycle(ctx context.Context, b *RustBackend, root, file string, source []byte, request backend.CodeActionRequest, resolve func(context.Context, RustSession, string, []byte, json.RawMessage) ([]byte, error)) (updated []byte, returnErr error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureRustSessionLocked(ctx, root); err != nil {
		return nil, err
	}
	session := b.session
	updated, err := runRustDocumentLifecycle(ctx, b, session, file, source, func(ctx context.Context, uri string) ([]byte, error) {
		raw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{
			lspTextDocumentKey: map[string]string{"uri": uri},
			"range":            request.Range,
			"context":          map[string]any{rustLSPDiagnosticsKey: []any{}, "only": []string{request.Kind}},
		})
		if err != nil {
			return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
		}
		return resolve(ctx, session, file, source, raw)
	})
	if err != nil {
		return nil, err
	}
	if err := stageRustExtraction(ctx, root, file, source, updated); err != nil {
		return nil, err
	}
	return updated, nil
}
