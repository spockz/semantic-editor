// Package scala owns bounded Metals refactorings over explicitly trusted workspaces.
// WHY: each advertised edit must validate the complete compiler response before staging source.
package scala

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

type scalaInlineShape struct {
	selection scalaRange
	delete    scalaRange
	uses      []scalaRange
	value     string
	callee    string
}

type scalaInlineCommand struct {
	Title     string            `json:"title"`
	Command   string            `json:"command"`
	Arguments []json.RawMessage `json:"arguments"`
}

type scalaApplyEditRequester interface {
	RequestWithApplyEdit(context.Context, string, any, func(json.RawMessage) (bool, error)) (json.RawMessage, error)
}

func (b *ScalaBackend) applyScalaInlineValue(parent context.Context, request backend.CodeActionRequest, root, file string, source []byte, shape scalaInlineShape) (result *backend.CodeActionResult, returnErr error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	if b.factory == nil {
		return nil, &ScalaError{Op: "start", Workspace: root, Err: ErrMetalsUnavailable}
	}
	session, err := b.factory(ctx, root, request.Project.Scala)
	if err != nil || session == nil {
		if err == nil {
			err = ErrMetalsUnavailable
		}
		return nil, &ScalaError{Op: "start", Workspace: root, Err: err}
	}
	closed, opened := false, false
	uri := pathutil.FileURI(file)
	defer func() {
		if opened {
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
			closeErr := session.Notify(closeCtx, "textDocument/didClose", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}})
			closeCancel()
			returnErr = errors.Join(returnErr, closeErr)
		}
		if !closed {
			returnErr = errors.Join(returnErr, session.Close())
		}
		if returnErr != nil {
			result = nil
		}
	}()
	err = initializeScalaSession(ctx, session, root, true)
	if err != nil {
		return nil, &ScalaError{Op: "initialize", Workspace: root, Err: err}
	}
	err = session.Notify(ctx, "textDocument/didOpen", map[string]any{scalaTextDocumentKey: map[string]any{
		"uri": uri, "languageId": "scala", "version": 1, "text": string(source),
	}})
	if err != nil {
		return nil, &ScalaError{Op: "didOpen", File: file, Err: err}
	}
	opened = true
	captured, err := captureScalaInlineValue(ctx, session, request, root, file, uri, source, shape)
	if err != nil {
		return nil, err
	}
	closeErr := session.Notify(ctx, "textDocument/didClose", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}})
	opened = false
	sessionErr := session.Close()
	closed = true
	if err := errors.Join(closeErr, sessionErr); err != nil {
		return nil, &ScalaError{Op: "close", File: file, Err: err}
	}
	if err := stageScalaExtraction(ctx, root, file, source, captured); err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: bytes.Clone(source), After: captured, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(captured)}}}, nil
}

func captureScalaInlineValue(ctx context.Context, session ScalaSession, request backend.CodeActionRequest, root, file, uri string, source []byte, shape scalaInlineShape) ([]byte, error) {
	// documentSymbol is a parse-readiness barrier; Metals may not offer actions immediately after didOpen.
	symbolsRaw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}})
	if err != nil {
		return nil, &ScalaError{Op: "documentSymbol", File: file, Err: err}
	}
	symbols, err := decodeScalaDocumentSymbols(symbolsRaw, root, source)
	if err != nil || len(symbols) == 0 || !scalaInlineOwnerPresent(symbols, request.EnclosingSymbol) {
		if err == nil {
			err = fmt.Errorf("metals did not report the selected Scala owner after parsing")
		}
		return nil, &ScalaError{Op: "documentSymbol", File: file, Symbol: request.EnclosingSymbol, Err: err}
	}
	selection := shape.selection
	actionsRaw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{
		scalaTextDocumentKey: map[string]string{"uri": uri}, "range": selection,
		"context": map[string]any{"diagnostics": []any{}, "only": []string{scalaRefactorInline}},
	})
	if err != nil {
		return nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
	}
	command, err := selectScalaInlineCommand(actionsRaw, request.Symbol, uri, selection.Start)
	if err != nil {
		return nil, &ScalaError{Op: scalaOpCodeAction, File: file, Symbol: request.Symbol, Err: err}
	}
	requester, ok := session.(scalaApplyEditRequester)
	if !ok {
		return nil, &ScalaError{Op: "executeCommand", File: file, Err: fmt.Errorf("metals session does not support correlated apply-edit capture")}
	}
	want, err := deriveScalaInlinePostimage(source, shape)
	if err != nil {
		return nil, err
	}
	var captured []byte
	commandParams := map[string]any{"command": command.Command, "arguments": command.Arguments}
	if _, err := requester.RequestWithApplyEdit(ctx, "workspace/executeCommand", commandParams, func(raw json.RawMessage) (bool, error) {
		candidate, decodeErr := decodeScalaInlineEdit(file, source, raw, shape)
		if decodeErr != nil {
			return false, decodeErr
		}
		if !bytes.Equal(candidate, want) {
			return false, fmt.Errorf("metals inline-value edit differs from the independently derived postimage")
		}
		captured = bytes.Clone(candidate)
		return true, nil
	}); err != nil {
		return nil, &ScalaError{Op: "executeCommand", File: file, Symbol: request.Symbol, Err: err}
	}
	if !bytes.Equal(captured, want) {
		return nil, &ScalaError{Op: "executeCommand", File: file, Symbol: request.Symbol, Err: fmt.Errorf("metals inline-value completed without the expected captured edit")}
	}
	return captured, nil
}

func scalaInlineOwnerPresent(symbols []scalaDocumentSymbol, qualified string) bool {
	parts := strings.Split(qualified, ".")
	wanted := parts[len(parts)-1]
	var visit func([]scalaDocumentSymbol) bool
	visit = func(items []scalaDocumentSymbol) bool {
		for _, item := range items {
			if item.Name == wanted && (item.Kind == 6 || item.Kind == 12) {
				return true
			}
			if visit(item.Children) {
				return true
			}
		}
		return false
	}
	return visit(symbols)
}

func deriveScalaInlinePostimage(source []byte, shape scalaInlineShape) ([]byte, error) {
	declarationStart, err := scalaByteOffset(source, shape.delete.Start)
	if err != nil {
		return nil, err
	}
	declarationEnd, err := scalaByteOffset(source, shape.delete.End)
	if err != nil {
		return nil, err
	}
	edits := []scalaOffsetEdit{{start: declarationStart, end: declarationEnd}}
	for _, use := range shape.uses {
		start, err := scalaByteOffset(source, use.Start)
		if err != nil {
			return nil, err
		}
		end, err := scalaByteOffset(source, use.End)
		if err != nil {
			return nil, err
		}
		edits = append(edits, scalaOffsetEdit{start: start, end: end, text: shape.value})
	}
	return applyScalaOffsetEdits(source, edits)
}

func validateScalaInlineShape(source []byte, request backend.CodeActionRequest) (scalaInlineShape, error) {
	owner := request.EnclosingSymbol
	if dot := strings.LastIndex(owner, "."); dot >= 0 {
		owner = owner[dot+1:]
	}
	name := request.Symbol
	pattern := regexp.MustCompile(`(?s)^\s*object\s+[A-Za-z_][A-Za-z0-9_]*\s*\{\s*\n[ \t]*def\s+` + regexp.QuoteMeta(owner) + `\s*\(\s*\)\s*:\s*Unit\s*=\s*\{\s*\n([ \t]*)val\s+` + regexp.QuoteMeta(name) + `\s*=\s*([0-9]+)\s*\n[ \t]*([A-Za-z_][A-Za-z0-9_]*)\(([A-Za-z_][A-Za-z0-9_]*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*)\)\s*\n[ \t]*\}\s*\n[ \t]*def\s+([A-Za-z_][A-Za-z0-9_]*)\(([^)]*)\)\s*:\s*Unit\s*=\s*\(\s*\)\s*\n\}\s*$`)
	match := pattern.FindSubmatchIndex(source)
	if len(match) < 14 {
		return scalaInlineShape{}, fmt.Errorf("scala inline-value currently requires one object with a zero-argument Unit owner, an untyped immutable decimal Int local, and one direct call")
	}
	callee := string(source[match[6]:match[7]])
	calleeDecl := string(source[match[10]:match[11]])
	params := strings.Split(string(source[match[12]:match[13]]), ",")
	args := strings.Split(string(source[match[8]:match[9]]), ",")
	if callee == owner || callee != calleeDecl || len(params) != len(args) || len(args) == 0 {
		return scalaInlineShape{}, fmt.Errorf("scala inline-value requires one same-object direct callee with matching strict Int parameters")
	}
	for _, parameter := range params {
		if !regexp.MustCompile(`^\s*[A-Za-z_][A-Za-z0-9_]*\s*:\s*Int\s*$`).MatchString(parameter) {
			return scalaInlineShape{}, fmt.Errorf("scala inline-value requires strict Int callee parameters")
		}
	}
	for _, argument := range args {
		if strings.TrimSpace(argument) != name {
			return scalaInlineShape{}, fmt.Errorf("scala inline-value requires every use to be a direct call argument")
		}
	}
	for _, literal := range regexp.MustCompile(`[0-9]+`).FindAllString(string(source[match[4]:match[5]]), -1) {
		if _, err := strconv.ParseUint(literal, 10, 31); err != nil {
			return scalaInlineShape{}, fmt.Errorf("scala inline-value requires a nonnegative Int literal")
		}
	}
	// The action range is the declaration identifier; the captured edit must delete its whole declaration line.
	declarationStart := match[2]
	declarationEnd := bytes.IndexByte(source[declarationStart:], '\n') + declarationStart + 1
	if declarationEnd <= declarationStart {
		return scalaInlineShape{}, fmt.Errorf("scala inline-value declaration must end with a newline")
	}
	valStart := bytes.Index(source[declarationStart:declarationEnd], []byte("val"))
	if valStart < 0 {
		return scalaInlineShape{}, fmt.Errorf("scala inline-value binding is missing")
	}
	nameStart := declarationStart + valStart + len("val")
	for nameStart < declarationEnd && (source[nameStart] == ' ' || source[nameStart] == '\t') {
		nameStart++
	}
	if !bytes.HasPrefix(source[nameStart:declarationEnd], []byte(name)) {
		return scalaInlineShape{}, fmt.Errorf("scala inline-value binding is missing")
	}
	callStart := match[8]
	uses := make([]scalaRange, 0, len(args))
	for _, argument := range args {
		trimmed := strings.TrimSpace(argument)
		if trimmed != name {
			return scalaInlineShape{}, fmt.Errorf("scala inline-value call arguments changed during validation")
		}
		argStart := bytes.Index(source[callStart:], []byte(name)) + callStart
		if argStart < callStart {
			return scalaInlineShape{}, fmt.Errorf("scala inline-value use is missing")
		}
		uses = append(uses, scalaRange{Start: scalaPositionAt(source, argStart), End: scalaPositionAt(source, argStart+len(name))})
		callStart = argStart + len(name)
	}
	if countScalaInlineIdentifier(source, name) != 1+len(uses) {
		return scalaInlineShape{}, fmt.Errorf("scala inline-value refuses additional, shadowed, or non-call uses")
	}
	return scalaInlineShape{
		selection: scalaRange{Start: scalaPositionAt(source, nameStart), End: scalaPositionAt(source, nameStart+len(name))},
		delete:    scalaRange{Start: scalaPositionAt(source, declarationStart), End: scalaPositionAt(source, declarationEnd)},
		uses:      uses,
		value:     string(source[match[4]:match[5]]),
		callee:    callee,
	}, nil
}

func countScalaInlineIdentifier(source []byte, name string) int {
	count := 0
	for offset := 0; offset < len(source); {
		if !scalaInlineIdentifierStart(source[offset]) {
			offset++
			continue
		}
		start := offset
		offset++
		for offset < len(source) && scalaInlineIdentifierPart(source[offset]) {
			offset++
		}
		if string(source[start:offset]) == name {
			count++
		}
	}
	return count
}

func scalaInlineIdentifierStart(char byte) bool {
	return char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z'
}

func scalaInlineIdentifierPart(char byte) bool {
	return scalaInlineIdentifierStart(char) || char >= '0' && char <= '9'
}

func selectScalaInlineCommand(raw json.RawMessage, name, uri string, position scalaPosition) (scalaInlineCommand, error) {
	var actions []scalaAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		return scalaInlineCommand{}, fmt.Errorf("malformed Metals inline-value actions: %w", err)
	}
	wantTitle := "Inline `" + name + "`"
	var selected *scalaAction
	for index := range actions {
		candidate := &actions[index]
		if candidate.Title == wantTitle && candidate.Kind == scalaRefactorInline {
			if selected != nil {
				return scalaInlineCommand{}, fmt.Errorf("multiple matching Metals inline-value actions are available")
			}
			selected = candidate
		}
	}
	if selected == nil || selected.Edit != nil && string(selected.Edit) != "null" || selected.Disabled != nil && string(selected.Disabled) != "null" || selected.Data != nil && string(selected.Data) != "null" {
		return scalaInlineCommand{}, fmt.Errorf("metals did not return one enabled command-based inline-value action")
	}
	var commandFields map[string]json.RawMessage
	if err := json.Unmarshal(selected.Command, &commandFields); err != nil || len(commandFields) != 3 || commandFields["title"] == nil || commandFields["command"] == nil || commandFields["arguments"] == nil {
		return scalaInlineCommand{}, fmt.Errorf("metals inline-value command contains an unsupported envelope")
	}
	var command scalaInlineCommand
	if err := json.Unmarshal(selected.Command, &command); err != nil || command.Title != "Inline value" || command.Command != "inline-value" || len(command.Arguments) != 1 {
		return scalaInlineCommand{}, fmt.Errorf("metals inline-value action has an unsupported command")
	}
	var argument struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position scalaPosition `json:"position"`
	}
	if err := json.Unmarshal(command.Arguments[0], &argument); err != nil || argument.TextDocument.URI != uri || argument.Position != position {
		return scalaInlineCommand{}, fmt.Errorf("metals inline-value command target does not match the selected binding")
	}
	return command, nil
}

func decodeScalaInlineEdit(file string, source []byte, raw json.RawMessage, shape scalaInlineShape) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope) != 1 || envelope["changes"] == nil {
		return nil, fmt.Errorf("metals inline-value requires one unversioned changes map")
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(envelope["changes"], &changes); err != nil || len(changes) != 1 {
		return nil, fmt.Errorf("metals inline-value requires edits for exactly one file")
	}
	var edits []json.RawMessage
	if err := json.Unmarshal(changes[pathutil.FileURI(file)], &edits); err != nil || len(edits) != 1+len(shape.uses) {
		return nil, fmt.Errorf("metals inline-value returned an unexpected edit count")
	}
	offsets, err := decodeScalaInlineEdits(source, edits, shape)
	if err != nil {
		return nil, err
	}
	return applyScalaOffsetEdits(source, offsets)
}

func decodeScalaInlineEdits(source []byte, edits []json.RawMessage, shape scalaInlineShape) ([]scalaOffsetEdit, error) {
	allowed := map[string]scalaRange{"delete": shape.delete}
	for index, use := range shape.uses {
		allowed[fmt.Sprintf("use-%d", index)] = use
	}
	offsets := make([]scalaOffsetEdit, 0, len(edits))
	used := make(map[string]bool, len(edits))
	for _, encoded := range edits {
		var rawEdit map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &rawEdit); err != nil {
			return nil, fmt.Errorf("decode metals inline-value edit: %w", err)
		}
		if len(rawEdit) != 2 || rawEdit["range"] == nil || rawEdit["newText"] == nil {
			return nil, fmt.Errorf("metals inline-value edit contains unsupported fields")
		}
		var edit scalaEdit
		if err := json.Unmarshal(encoded, &edit); err != nil {
			return nil, err
		}
		start, err := scalaByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, err
		}
		end, err := scalaByteOffset(source, edit.Range.End)
		if err != nil || end < start {
			return nil, fmt.Errorf("metals inline-value edit has an invalid range")
		}
		key := ""
		for candidate, expected := range allowed {
			expectedStart, _ := scalaByteOffset(source, expected.Start)
			expectedEnd, _ := scalaByteOffset(source, expected.End)
			if start == expectedStart && end == expectedEnd {
				key = candidate
				break
			}
		}
		if key == "" || used[key] {
			return nil, fmt.Errorf("metals inline-value edit does not match one expected declaration or use")
		}
		used[key] = true
		newText := shape.value
		if key == "delete" {
			newText = ""
		}
		if edit.NewText != newText {
			return nil, fmt.Errorf("metals inline-value edit has an unexpected replacement")
		}
		offsets = append(offsets, scalaOffsetEdit{start: start, end: end, text: edit.NewText})
	}
	if len(used) != len(allowed) {
		return nil, fmt.Errorf("metals inline-value omitted a declaration or use edit")
	}
	return offsets, nil
}
