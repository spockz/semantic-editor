// Package rust validates rust-analyzer's bounded local-to-constant assist.
// WHY: The native assist can change name resolution and drop timing, so this file proves the exact eligible source shape and complete postimage before staging.
package rust

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const (
	rustPromoteLocalToConstKind = "promote_local_to_const"
)

var rustPromoteActionID = regexp.MustCompile("^promote_local_to_const:Refactor:[0-9]+:$")

type rustPromotePlan struct {
	rangeValue backend.Range
	expected   []byte
	edits      []rustEditSpan
}

func (b *RustBackend) applyRustPromoteLocalToConst(ctx context.Context, request backend.CodeActionRequest) (*backend.CodeActionResult, error) {
	if request.Operation != backend.OperationPromoteLocalToConst || request.Kind != rustPromoteLocalToConstKind || !isSimpleRustIdentifier(request.Symbol) || request.EnclosingSymbol == "" {
		return nil, fmt.Errorf("unsupported rust promote-local-to-const request")
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
		return nil, fmt.Errorf("rust promote-local-to-const source revision changed")
	}
	plan, err := planRustPromoteLocalToConst(source, request.EnclosingSymbol, request.Symbol)
	if err != nil {
		return nil, err
	}
	if request.Range != plan.rangeValue {
		return nil, fmt.Errorf("rust promote-local-to-const binding selection is stale or ambiguous")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureRustSessionLocked(ctx, root); sessionErr != nil {
		return nil, sessionErr
	}
	session := b.session
	updated, err := runRustDocumentLifecycle(ctx, b, session, file, source, func(ctx context.Context, uri string) ([]byte, error) {
		return resolveAndValidateRustPromoteLocalToConst(ctx, session, file, uri, source, plan)
	})
	if err != nil {
		return nil, err
	}
	if err := sourcebuf.ApplyAndPublish(ctx, root, "Rust promote-local-to-const", []sourcebuf.Change{{Path: file, Before: source, After: updated}}); err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: source, After: updated, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(updated)}}}, nil
}

func planRustPromoteLocalToConst(source []byte, owner, name string) (rustPromotePlan, error) {
	if rustHasPromoteUncertainty(source) {
		return rustPromotePlan{}, fmt.Errorf("rust promote-local-to-const refuses macros, attributes, imports, comments, strings, and custom implementation contexts")
	}
	if !isSimpleRustIdentifier(owner) || !rustPromoteLowerSnakeName(name) {
		return rustPromotePlan{}, fmt.Errorf("rust promote-local-to-const requires a lowercase ASCII local in a simple function")
	}
	generated := strings.ToUpper(name)
	if rustIdentifierPresent(source, generated) {
		return rustPromotePlan{}, fmt.Errorf("generated Rust constant name %q already appears in the selected source", generated)
	}
	lines := bytes.SplitAfter(source, []byte("\n"))
	ownerLine, err := findRustPromoteOwnerLine(lines, owner)
	if err != nil {
		return rustPromotePlan{}, err
	}
	return planRustPromoteStatements(source, lines, ownerLine, name, generated)
}

func findRustPromoteOwnerLine(lines [][]byte, owner string) (int, error) {
	ownerHeader := "fn " + owner + "() {"
	publicHeader := "pub " + ownerHeader
	ownerLine := -1
	for index, line := range lines {
		text := strings.TrimSpace(strings.TrimSuffix(string(line), "\n"))
		if text != ownerHeader && text != publicHeader {
			continue
		}
		if ownerLine >= 0 {
			return -1, fmt.Errorf("rust promote-local-to-const owner is ambiguous")
		}
		ownerLine = index
	}
	if ownerLine < 0 {
		return -1, fmt.Errorf("rust promote-local-to-const requires one simple zero-argument function")
	}
	return ownerLine, nil
}

func planRustPromoteStatements(source []byte, lines [][]byte, ownerLine int, name, generated string) (rustPromotePlan, error) {
	var empty rustPromotePlan
	if ownerLine < 0 || ownerLine+3 >= len(lines) {
		return empty, fmt.Errorf("rust promote-local-to-const requires one simple zero-argument function")
	}
	bodyLine := strings.TrimSuffix(strings.TrimSuffix(string(lines[ownerLine+1]), "\n"), "\r")
	callLine := strings.TrimSuffix(strings.TrimSuffix(string(lines[ownerLine+2]), "\n"), "\r")
	closeLine := strings.TrimSpace(strings.TrimSuffix(string(lines[ownerLine+3]), "\n"))
	indent := leadingRustWhitespace(bodyLine)
	body := strings.TrimSpace(bodyLine)
	call := strings.TrimSpace(callLine)
	if indent == "" || strings.TrimSpace(callLine) == "" || leadingRustWhitespace(callLine) != indent || closeLine != "}" {
		return empty, fmt.Errorf("rust promote-local-to-const requires adjacent direct statements in one simple function")
	}
	prefix := "let " + name + ": "
	if !strings.HasPrefix(body, prefix) || !strings.HasSuffix(body, ";") {
		return empty, fmt.Errorf("rust promote-local-to-const requires an immutable explicitly typed local initializer")
	}
	equals := strings.Index(body, " = ")
	if equals < len(prefix) || body[equals+3:len(body)-1] == "" {
		return empty, fmt.Errorf("rust promote-local-to-const requires one direct initializer")
	}
	typeName := strings.TrimSpace(body[len(prefix):equals])
	value := body[equals+3 : len(body)-1]
	if !rustFixedIntegerType(typeName) || !rustDecimalLiteralFits(value, typeName) {
		return empty, fmt.Errorf("rust promote-local-to-const requires a range-valid decimal fixed-width primitive integer literal")
	}
	if rustSourceHasGenericTypeParameter(source, typeName) || rustSourceMayShadowPrimitiveType(source, typeName) {
		return empty, fmt.Errorf("rust promote-local-to-const cannot prove primitive integer type %q is unshadowed", typeName)
	}
	callName, arguments, ok := rustPromoteCall(call)
	if !ok || callName == name || len(arguments) != 1 || arguments[0] != name {
		return empty, fmt.Errorf("rust promote-local-to-const requires one immediately following direct call with the local as its sole value argument")
	}
	if rustIdentifierOccurrences(source, name) != 2 {
		return empty, fmt.Errorf("rust promote-local-to-const refuses extra, shadowed, or non-call uses of the local")
	}
	keywordStart, nameStart, callArgStart, err := rustPromoteBindingOffsets(source, lines, ownerLine, name)
	if err != nil {
		return empty, err
	}
	edits := []rustEditSpan{
		{start: keywordStart, end: keywordStart + len("let"), text: "const"},
		{start: nameStart, end: nameStart + len(name), text: generated},
		{start: callArgStart, end: callArgStart + len(name), text: generated},
	}
	sortRustEditSpans(edits)
	selection, err := byteOffsetsToBackendRange(source, nameStart, nameStart+len(name))
	if err != nil {
		return empty, fmt.Errorf("resolve Rust promote-local-to-const binding: %w", err)
	}
	return rustPromotePlan{rangeValue: selection, expected: applyRustEditSpans(source, edits), edits: edits}, nil
}

func rustPromoteBindingOffsets(source []byte, lines [][]byte, ownerLine int, name string) (keywordStart, nameStart, callArgStart int, returnErr error) {
	declLine := ownerLine + 1
	for index := range declLine {
		keywordStart += len(lines[index])
	}
	declContent := strings.TrimSuffix(strings.TrimSuffix(string(lines[declLine]), "\n"), "\r")
	keywordStart += len(declContent) - len(strings.TrimLeft(declContent, " \t"))
	nameStart = keywordStart + len("let ")
	callLine := ownerLine + 2
	callStart := 0
	for index := range callLine {
		callStart += len(lines[index])
	}
	callContent := strings.TrimSuffix(strings.TrimSuffix(string(lines[callLine]), "\n"), "\r")
	openParen := strings.Index(callContent, "(")
	callArgumentText := callContent[openParen+1:]
	callArgStart = callStart + openParen + 1 + len(callArgumentText) - len(strings.TrimLeft(callArgumentText, " \t"))
	if callArgStart < 0 || callArgStart+len(name) > len(source) || string(source[callArgStart:callArgStart+len(name)]) != name {
		return 0, 0, 0, fmt.Errorf("rust promote-local-to-const could not resolve the direct argument")
	}
	return keywordStart, nameStart, callArgStart, nil
}

func rustHasPromoteUncertainty(source []byte) bool {
	text := string(source)
	return !bytes.Equal([]byte(text), bytes.ToValidUTF8(source, []byte{})) ||
		strings.ContainsAny(text, "#!\\\"'") || strings.Contains(text, "//") || strings.Contains(text, "/*") || strings.Contains(text, "*/") ||
		rustIdentifierPresent(source, "use") || rustIdentifierPresent(source, "impl") || strings.Contains(text, "include")
}

func rustFixedIntegerType(name string) bool {
	switch name {
	case "i8", "i16", "i32", "i64", "i128", "u8", "u16", "u32", "u64", "u128":
		return true
	default:
		return false
	}
}

func rustDecimalLiteralFits(text, typeName string) bool {
	if !rustPlainDecimalLiteral(text) {
		return false
	}
	value, ok := new(big.Int).SetString(strings.ReplaceAll(text, "_", ""), 10)
	if !ok {
		return false
	}
	width := 0
	if _, err := fmt.Sscanf(typeName[1:], "%d", &width); err != nil {
		return false
	}
	if strings.HasPrefix(typeName, "i") {
		width--
	}
	maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(width)), big.NewInt(1))
	return value.Sign() >= 0 && value.Cmp(maximum) <= 0
}

func rustPlainDecimalLiteral(value string) bool {
	if value == "" || value[0] < '0' || value[0] > '9' || strings.HasPrefix(value, "_") || strings.HasSuffix(value, "_") || strings.Contains(value, "__") {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func rustPromoteCall(text string) (string, []string, bool) {
	text = strings.TrimSpace(strings.TrimSuffix(text, ";"))
	open, close := strings.IndexByte(text, '('), strings.LastIndexByte(text, ')')
	if open <= 0 || close != len(text)-1 || close < open || strings.ContainsAny(text[:open], " ()") {
		return "", nil, false
	}
	callee := text[:open]
	for component := range strings.SplitSeq(callee, "::") {
		if !isSimpleRustIdentifier(component) {
			return "", nil, false
		}
	}
	args := strings.TrimSpace(text[open+1 : close])
	if args == "" || strings.Contains(args, ",") {
		return "", nil, false
	}
	return callee, []string{args}, true
}

func rustPromoteLowerSnakeName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' || strings.HasSuffix(name, "_") || strings.Contains(name, "__") {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func rustIdentifierOccurrences(source []byte, name string) int {
	count := 0
	for offset := 0; offset < len(source); {
		index := bytes.Index(source[offset:], []byte(name))
		if index < 0 {
			break
		}
		start := offset + index
		end := start + len(name)
		if (start == 0 || !isRustIdentifierByte(source[start-1])) && (end == len(source) || !isRustIdentifierByte(source[end])) {
			count++
		}
		offset = start + 1
	}
	return count
}

func leadingRustWhitespace(text string) string {
	return text[:len(text)-len(strings.TrimLeft(text, " \t"))]
}

func byteOffsetsToBackendRange(source []byte, start, end int) (backend.Range, error) {
	startPosition, err := rustPositionAtOffset(source, start)
	if err != nil {
		return backend.Range{}, err
	}
	endPosition, err := rustPositionAtOffset(source, end)
	if err != nil {
		return backend.Range{}, err
	}
	return backend.Range{Start: backend.Position{Line: startPosition.Line, Character: startPosition.Character}, End: backend.Position{Line: endPosition.Line, Character: endPosition.Character}}, nil
}

func resolveAndValidateRustPromoteLocalToConst(ctx context.Context, session RustSession, file, uri string, source []byte, plan rustPromotePlan) ([]byte, error) {
	raw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{
		lspTextDocumentKey: map[string]string{"uri": uri},
		"range":            plan.rangeValue,
		"context":          map[string]any{rustLSPDiagnosticsKey: []any{}, "only": []string{"refactor"}},
	})
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	action, err := selectRustPromoteAction(ctx, session, file, raw)
	if err != nil {
		return nil, err
	}
	return validateRustPromoteActionEdit(file, source, plan, action.Edit)
}

func selectRustPromoteAction(ctx context.Context, session RustSession, file string, raw json.RawMessage) (rustCodeAction, error) {
	action, encoded, err := selectUniqueRustCodeAction(raw, "promote-local-to-const", func(action rustCodeAction) bool {
		var data rustCodeActionData
		return action.Title == "Promote local to constant" && action.Kind == "refactor" &&
			json.Unmarshal(action.Data, &data) == nil && rustPromoteActionID.MatchString(data.ID)
	})
	if err != nil {
		return rustCodeAction{}, err
	}
	var selectedData rustCodeActionData
	if json.Unmarshal(action.Data, &selectedData) != nil || !rustPromoteActionID.MatchString(selectedData.ID) {
		return rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	if rustCodeActionDisabled(action.Disabled) || hasRustCommand(action.Command) {
		return rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	if len(action.Edit) > 0 && string(action.Edit) != "null" {
		return action, nil
	}
	return resolveRustPromoteAction(ctx, session, file, encoded, selectedData.ID)
}

func resolveRustPromoteAction(ctx context.Context, session RustSession, file string, encoded json.RawMessage, selectedID string) (rustCodeAction, error) {
	resolved, err := session.Request(ctx, rustOpResolveAction, encoded)
	if err != nil {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: err}
	}
	var action rustCodeAction
	if err := json.Unmarshal(resolved, &action); err != nil {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: fmt.Errorf("decode resolved Rust promote-local-to-const action: %w", err)}
	}
	if action.Title != "Promote local to constant" || action.Kind != "refactor" || rustCodeActionDisabled(action.Disabled) || hasRustCommand(action.Command) || len(action.Edit) == 0 || string(action.Edit) == "null" {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: fmt.Errorf("resolved Rust promote-local-to-const action has an unexpected shape: %w", ErrRustRenameInvalidEdit)}
	}
	if len(action.Data) == 0 || string(bytes.TrimSpace(action.Data)) == "null" {
		return action, nil
	}
	var data rustCodeActionData
	if json.Unmarshal(action.Data, &data) != nil || data.ID != selectedID {
		return rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: fmt.Errorf("resolved Rust promote-local-to-const identity changed: %w", ErrRustRenameInvalidEdit)}
	}
	return action, nil
}

func validateRustPromoteActionEdit(file string, source []byte, plan rustPromotePlan, raw json.RawMessage) ([]byte, error) {
	edits, err := decodeRustSingleFileWorkspaceEdit(file, raw, 1)
	if err != nil || len(edits) != len(plan.edits) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("invalid Rust promote-local-to-const edit (decode: %w, edit count: %d): %w", err, len(edits), ErrRustRenameInvalidEdit)}
	}
	actual := make([]rustEditSpan, len(edits))
	for index, edit := range edits {
		start, startErr := rustByteOffset(source, edit.Range.Start)
		end, endErr := rustByteOffset(source, edit.Range.End)
		if startErr != nil || endErr != nil || end < start {
			return nil, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
		}
		actual[index] = rustEditSpan{start: start, end: end, text: edit.NewText}
	}
	sortRustEditSpans(actual)
	for index := range actual {
		if actual[index] != plan.edits[index] {
			return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("unexpected Rust promote-local-to-const edit %d: got %+v want %+v: %w", index, actual[index], plan.edits[index], ErrRustRenameInvalidEdit)}
		}
	}
	candidate := applyRustEditSpans(source, actual)
	if !bytes.Equal(candidate, plan.expected) {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("rust promote-local-to-const result did not match the independently derived postimage (got %q want %q): %w", candidate, plan.expected, ErrRustRenameInvalidEdit)}
	}
	return candidate, nil
}
