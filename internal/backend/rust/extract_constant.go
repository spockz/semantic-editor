// Package rust validates rust-analyzer's local constant extraction assist.
// WHY: The assist preserves an existing local binding while inserting a typed const, so its exact edit and rename must be proven before publication.
package rust

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
)

const (
	rustExtractConstantKind          = "constant"
	rustExtractConstantActionID      = "extract_constant:RefactorExtract:1:"
	rustMethodTextDocumentCodeAction = "textDocument/codeAction"
)

func isSafeRustConstantIdentifier(value string) bool {
	return isSimpleRustIdentifier(value) && value == strings.ToUpper(value)
}

func (b *RustBackend) applyRustExtractConstant(ctx context.Context, request backend.CodeActionRequest) (*backend.CodeActionResult, error) {
	if request.Operation != backend.OperationExtractVariable || request.Kind != rustExtractConstantKind || !isSafeRustConstantIdentifier(request.NewName) || request.Range.End.Line < request.Range.Start.Line || request.Range.End.Line == request.Range.Start.Line && request.Range.End.Character <= request.Range.Start.Character {
		return nil, fmt.Errorf("unsupported Rust extract-constant request")
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
		return nil, fmt.Errorf("rust extract-constant source revision changed")
	}
	if rustHasUnresolvedNameExpansion(source) {
		return nil, fmt.Errorf("rust extract-constant refuses files with unresolved imported or macro-expanded names")
	}
	if rustIdentifierPresent(source, request.NewName) {
		return nil, fmt.Errorf("requested Rust constant name %q already appears in the selected file", request.NewName)
	}
	selectionStart, selectionEnd, insertionStart, indent, localName, valueType, err := resolveRustConstantSelection(source, request.Range, request.Expression)
	if err != nil {
		return nil, err
	}
	if rustSourceHasGenericTypeParameter(source, valueType) {
		return nil, fmt.Errorf("rust extract-constant refuses possibly shadowed primitive type %q", valueType)
	}
	if rustSourceMayShadowPrimitiveType(source, valueType) {
		return nil, fmt.Errorf("rust extract-constant refuses possibly shadowed primitive type %q", valueType)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureRustSessionLocked(ctx, root); sessionErr != nil {
		return nil, sessionErr
	}
	session := b.session
	updated, err := runRustDocumentLifecycle(ctx, b, session, file, source, func(ctx context.Context, uri string) ([]byte, error) {
		edits, editErr := requestRustExtractConstantEdits(ctx, session, file, uri, request.Range)
		if editErr != nil {
			return nil, editErr
		}
		candidate, position, useSpan, bindingSpan, candidateErr := prepareRustExtractConstantCandidate(source, edits, selectionStart, selectionEnd, insertionStart, indent, localName, request.Expression, valueType)
		if candidateErr != nil {
			return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("validate extract-constant workspace edit: %w", candidateErr)}
		}
		if notifyErr := session.Notify(ctx, "textDocument/didChange", map[string]any{
			lspTextDocumentKey: map[string]any{"uri": uri, "version": 2},
			"contentChanges":   []map[string]string{{"text": string(candidate)}},
		}); notifyErr != nil {
			return nil, &RustError{Op: "didChange", File: file, Err: notifyErr}
		}
		return renameRustExtractBinding(ctx, session, file, uri, candidate, position, useSpan, bindingSpan, request.NewName)
	})
	if err != nil {
		return nil, err
	}
	if stageErr := stageRustExtraction(ctx, root, file, source, updated); stageErr != nil {
		return nil, stageErr
	}
	return &backend.CodeActionResult{File: file, Before: source, After: updated, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(updated)}}}, nil
}

func resolveRustConstantSelection(source []byte, selected backend.Range, expression string) (int, int, int, []byte, string, string, error) {
	start, err := rustByteOffset(source, rustPosition(selected.Start))
	if err != nil {
		return 0, 0, 0, nil, "", "", fmt.Errorf("resolve Rust constant selection: %w", err)
	}
	end, err := rustByteOffset(source, rustPosition(selected.End))
	if err != nil || end <= start || string(source[start:end]) != expression {
		return 0, 0, 0, nil, "", "", fmt.Errorf("rust constant selection does not match the requested expression")
	}
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := len(source)
	if offset := bytes.IndexByte(source[start:], '\n'); offset >= 0 {
		lineEnd = start + offset
	}
	if start < lineStart || end > lineEnd {
		return 0, 0, 0, nil, "", "", fmt.Errorf("rust extract-constant currently requires a single-line expression")
	}
	line := source[lineStart:lineEnd]
	indentLength := len(line) - len(bytes.TrimLeft(line, " \t"))
	indent := bytes.Clone(line[:indentLength])
	prefix := string(line[indentLength : start-lineStart])
	if !strings.HasPrefix(prefix, "let ") || !strings.HasSuffix(prefix, " = ") || strings.ContainsAny(prefix, "(){}!") {
		return 0, 0, 0, nil, "", "", fmt.Errorf("rust extract-constant currently requires a direct local initializer")
	}
	localName := strings.TrimSuffix(strings.TrimPrefix(prefix, "let "), " = ")
	if !isSimpleRustIdentifier(localName) || string(bytes.TrimSpace(line[end-lineStart:])) != ";" || bytes.Contains(line, []byte("//")) || bytes.Contains(line, []byte("/*")) {
		return 0, 0, 0, nil, "", "", fmt.Errorf("rust extract-constant currently requires a comment-free local initializer statement")
	}
	valueType, ok := rustSuffixedIntegerExpressionType(expression)
	if !ok {
		return 0, 0, 0, nil, "", "", fmt.Errorf("rust extract-constant currently requires a suffixed integer literal expression")
	}
	return start, end, lineStart + indentLength, indent, localName, valueType, nil
}

func rustSuffixedIntegerExpressionType(expression string) (string, bool) {
	parts := strings.Split(expression, " + ")
	if len(parts) == 0 {
		return "", false
	}
	var resultType string
	sum := new(big.Int)
	for _, part := range parts {
		if !rustSuffixedIntegerLiteral(part) {
			return "", false
		}
		underscore := strings.LastIndexByte(part, '_')
		if underscore < 0 || underscore == len(part)-1 {
			return "", false
		}
		literalType := part[underscore+1:]
		if resultType != "" && resultType != literalType {
			return "", false
		}
		resultType = literalType
		number := strings.ReplaceAll(part[:underscore], "_", "")
		value, ok := new(big.Int).SetString(number, 10)
		if !ok {
			return "", false
		}
		sum.Add(sum, value)
	}
	maximum, ok := rustIntegerMaximum(resultType)
	return resultType, resultType != "" && ok && sum.Cmp(maximum) <= 0
}

func rustIntegerMaximum(suffix string) (*big.Int, bool) {
	if suffix == "isize" || suffix == "usize" {
		return nil, false
	}
	signed := strings.HasPrefix(suffix, "i")
	width, err := strconv.Atoi(suffix[1:])
	if err != nil || width < 8 || width > 128 || width%8 != 0 {
		return nil, false
	}
	bits := uint(width)
	if signed {
		bits--
	}
	maximum := new(big.Int).Lsh(big.NewInt(1), bits)
	maximum.Sub(maximum, big.NewInt(1))
	return maximum, true
}

func requestRustExtractConstantEdits(ctx context.Context, session RustSession, file, uri string, selected backend.Range) ([]rustTextEdit, error) {
	actionRaw, err := session.Request(ctx, rustMethodTextDocumentCodeAction, map[string]any{
		lspTextDocumentKey: map[string]string{"uri": uri},
		"range":            selected,
		"context":          map[string]any{rustLSPDiagnosticsKey: []any{}, "only": []string{rustRefactorExtract}},
	})
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	action, encoded, err := selectRustExtractConstantAction(actionRaw)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("select extract-constant action from %d-byte response: %w", len(actionRaw), err)}
	}
	if action.Title != "Extract into constant" || rustCodeActionDisabled(action.Disabled) || hasRustCommand(action.Command) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("unsupported rust-analyzer extract-constant action (title %q, disabled %t, command %t)", action.Title, rustCodeActionDisabled(action.Disabled), hasRustCommand(action.Command))}
	}
	if len(action.Edit) == 0 || string(action.Edit) == "null" {
		if len(action.Data) == 0 || string(action.Data) == "null" {
			return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("%w (extract-constant action data length %d)", ErrRustRenameInvalidEdit, len(action.Data))}
		}
		resolved, resolveErr := session.Request(ctx, rustOpResolveAction, encoded)
		if resolveErr != nil {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: resolveErr}
		}
		original := action
		if decodeErr := json.Unmarshal(resolved, &action); decodeErr != nil || rustCodeActionDisabled(action.Disabled) || action.Title != original.Title || action.Kind != original.Kind || !rustResolvedExtractConstantIdentity(action.Data, original.Data) || hasRustCommand(action.Command) || len(action.Edit) == 0 || string(action.Edit) == "null" {
			return nil, &RustError{Op: rustOpResolveAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
	}
	return decodeRustSingleFileWorkspaceEdit(file, action.Edit, 1)
}

func selectRustExtractConstantAction(raw json.RawMessage) (rustCodeAction, json.RawMessage, error) {
	return selectUniqueRustCodeAction(raw, "extract-constant", func(action rustCodeAction) bool {
		var data rustCodeActionData
		return action.Kind == rustRefactorExtract && json.Unmarshal(action.Data, &data) == nil && data.ID == rustExtractConstantActionID
	})
}

func rustResolvedExtractConstantIdentity(resolved, original json.RawMessage) bool {
	if len(resolved) == 0 || string(resolved) == "null" {
		return true
	}
	return sameRustCodeActionIdentity(resolved, original, rustExtractConstantActionID)
}

func prepareRustExtractConstantCandidate(source []byte, edits []rustTextEdit, selectionStart, selectionEnd, insertionStart int, indent []byte, localName, expression, valueType string) ([]byte, backend.Position, rustEditSpan, rustEditSpan, error) {
	if len(edits) != 4 || !isSimpleRustIdentifier(localName) {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: edit count or local identifier", ErrRustRenameInvalidEdit)
	}
	var letEdit, nameEdit, typeEdit, aliasEdit *rustTextEdit
	var letSpan, nameSpan, typeSpan, aliasSpan rustEditSpan
	for i := range edits {
		start, err := rustByteOffset(source, edits[i].Range.Start)
		if err != nil {
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: edit start position", ErrRustRenameInvalidEdit)
		}
		end, err := rustByteOffset(source, edits[i].Range.End)
		if err != nil || end < start {
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: edit end position", ErrRustRenameInvalidEdit)
		}
		switch {
		case start == insertionStart && end == insertionStart+len("let"):
			letEdit = &edits[i]
			letSpan = rustEditSpan{start: start, end: end, text: edits[i].NewText}
		case start == insertionStart+len("let ") && end == start+len(localName):
			nameEdit = &edits[i]
			nameSpan = rustEditSpan{start: start, end: end, text: edits[i].NewText}
		case start == insertionStart+len("let ")+len(localName)+1 && end == start:
			typeEdit = &edits[i]
			typeSpan = rustEditSpan{start: start, end: end, text: edits[i].NewText}
		case start == end && start > selectionEnd:
			aliasEdit = &edits[i]
			aliasSpan = rustEditSpan{start: start, end: end, text: edits[i].NewText}
		default:
			return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: unexpected edit range", ErrRustRenameInvalidEdit)
		}
	}
	if letEdit == nil || nameEdit == nil || typeEdit == nil || aliasEdit == nil || letEdit.NewText != "const" || typeEdit.NewText != valueType+" " {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: expected edit text", ErrRustRenameInvalidEdit)
	}
	generatedName := strings.TrimSuffix(nameEdit.NewText, ":")
	if !isSafeRustConstantIdentifier(generatedName) || nameEdit.NewText != generatedName+":" || string(source[nameSpan.start:nameSpan.end]) != localName {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: generated constant name", ErrRustRenameInvalidEdit)
	}
	if !rustIsLineIndentationOffset(source, aliasSpan.start, indent) || aliasSpan.start < selectionEnd || aliasEdit.NewText != "let "+localName+" = "+generatedName+";\n"+string(indent) || !rustAliasInsertionBeforeFirstUse(source, selectionEnd, aliasSpan.start, localName) || string(source[selectionStart:selectionEnd]) != expression || selectionEnd >= len(source) || source[selectionEnd] != ';' {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: alias insertion or initializer shape", ErrRustRenameInvalidEdit)
	}
	updated := applyRustEditSpans(source, []rustEditSpan{aliasSpan, typeSpan, nameSpan, letSpan})
	bindingStart := insertionStart + len("const ")
	aliasCandidateStart := aliasSpan.start + len("const") - len("let") + len(nameEdit.NewText) - len(localName) + len(typeEdit.NewText)
	useStart := aliasCandidateStart + len("let "+localName+" = ")
	binding := rustEditSpan{start: bindingStart, end: bindingStart + len(generatedName)}
	use := rustEditSpan{start: useStart, end: useStart + len(generatedName)}
	if use.end > len(updated) || string(updated[binding.start:binding.end]) != generatedName || string(updated[use.start:use.end]) != generatedName {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("%w: renamed spans", ErrRustRenameInvalidEdit)
	}
	position, err := rustPositionAtOffset(updated, use.start)
	if err != nil {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, err
	}
	return updated, position, use, binding, nil
}

func rustIsLineIndentationOffset(source []byte, offset int, indent []byte) bool {
	if offset < 0 || offset > len(source) {
		return false
	}
	lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
	return offset == lineStart+len(indent) && bytes.Equal(source[lineStart:offset], indent)
}

func rustAliasInsertionBeforeFirstUse(source []byte, after int, insertion int, name string) bool {
	for _, token := range rustLexSource(source) {
		if token.start < after || token.text != name {
			continue
		}
		lineStart := bytes.LastIndexByte(source[:token.start], '\n') + 1
		line := source[lineStart:token.start]
		indentLength := len(line) - len(bytes.TrimLeft(line, " \t"))
		return insertion == lineStart+indentLength
	}
	return false
}
