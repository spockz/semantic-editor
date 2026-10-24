// Package scala captures the bounded Metals extract-method action.
// WHY: Callable extraction must accept only the versioned, same-file edits and local rename proven by the configured server.
package scala

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
)

var scalaDirectCallStatement = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\((?:\s*(?:[0-9]+(?:\s*[+*-]\s*[0-9]+)*|"(?:[^"\\]|\\.)*")\s*(?:,\s*(?:[0-9]+(?:\s*[+*-]\s*[0-9]+)*|"(?:[^"\\]|\\.)*")\s*)*)?\)$`)

type scalaExtractMethodData struct {
	Param struct {
		URI string `json:"uri"`
	} `json:"param"`
	Range           scalaRange    `json:"range"`
	ExtractPosition scalaPosition `json:"extractPosition"`
	CodeActionName  string        `json:"codeActionName"`
}

func scalaCallableStatement(statement string) bool {
	lines := strings.Split(strings.TrimSpace(statement), "\n")
	if len(lines) == 0 {
		return false
	}
	callee := ""
	for _, line := range lines {
		match := scalaDirectCallStatement.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 2 || !scalaIdentifier(match[1]) || (callee != "" && match[1] != callee) {
			return false
		}
		callee = match[1]
	}
	return true
}

func scalaCallableSelectionIsWholeLine(source []byte, start, end int) bool {
	if start < 0 || end <= start || end > len(source) {
		return false
	}
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := len(source)
	if offset := bytes.IndexByte(source[end:], '\n'); offset >= 0 {
		lineEnd = end + offset
	}
	return strings.TrimSpace(string(source[lineStart:start])) == "" && strings.TrimSpace(string(source[end:lineEnd])) == "" && scalaCallableStatement(string(source[start:end]))
}

func (b *ScalaBackend) applyScalaExtractCallableLocked(ctx context.Context, request backend.CodeActionRequest, root, file string, source []byte, statement string) (result *backend.CodeActionResult, returnErr error) {
	return b.runScalaExtraction(ctx, root, file, source, func(ctx context.Context, session ScalaSession, uri string) ([]byte, func(context.Context) ([]byte, error), error) {
		selection := scalaRange{Start: scalaPosition(request.Range.Start), End: scalaPosition(request.Range.End)}
		raw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}, "range": selection, "context": map[string]any{"diagnostics": []any{}, "only": []string{scalaRefactorExtract}}})
		if err != nil {
			return nil, nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
		}
		action, err := selectScalaExtractMethodAction(raw, uri, selection, request.EnclosingSymbol)
		if err != nil {
			return nil, nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
		}
		resolvedRaw, err := session.Request(ctx, "codeAction/resolve", action)
		if err != nil {
			return nil, nil, &ScalaError{Op: "codeAction/resolve", File: file, Err: err}
		}
		resolved, err := validateScalaResolvedExtractMethod(resolvedRaw, action)
		if err != nil {
			return nil, nil, &ScalaError{Op: "codeAction/resolve", File: file, Err: err}
		}
		indent := scalaLineIndent(source, scalaPosition(request.Range.Start))
		edits, generated, err := decodeScalaExtractMethodEdits(file, source, resolved.Edit, selection, statement, indent)
		if err != nil {
			return nil, nil, &ScalaError{Op: "codeAction/resolve", File: file, Err: err}
		}
		candidate, err := applyScalaOffsetEdits(source, edits)
		if err != nil {
			return nil, nil, &ScalaError{Op: "codeAction/resolve", File: file, Err: err}
		}
		definitionStart, callStart, err := scalaExtractedMethodSpans(candidate, generated)
		if err != nil {
			return nil, nil, err
		}
		return candidate, func(ctx context.Context) ([]byte, error) {
			renamed, err := renameScalaExtractedMethod(ctx, session, uri, candidate, generated, request.NewName, definitionStart, callStart)
			if err != nil {
				return nil, &ScalaError{Op: "rename", File: file, Err: err}
			}
			return renamed, nil
		}, nil
	})
}

func selectScalaExtractMethodAction(raw json.RawMessage, uri string, selected scalaRange, owner string) (scalaAction, error) {
	var actions []scalaAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		return scalaAction{}, fmt.Errorf("malformed Metals extract-method actions: %w", err)
	}
	ownerName := owner
	if dot := strings.LastIndex(ownerName, "."); dot >= 0 {
		ownerName = ownerName[dot+1:]
	}
	wantTitle := "Extract selection as method in method `" + ownerName + "`"
	var chosen *scalaAction
	for index := range actions {
		candidate := &actions[index]
		if candidate.Title != wantTitle || candidate.Kind != scalaRefactorExtract {
			continue
		}
		var data scalaExtractMethodData
		if err := json.Unmarshal(candidate.Data, &data); err != nil || data.Param.URI != uri || data.Range != selected || data.ExtractPosition != selected.Start || data.CodeActionName != "ExtractMethodCodeAction" {
			return scalaAction{}, fmt.Errorf("metals extract-method action does not match the selected method and range")
		}
		if candidate.Edit != nil && string(candidate.Edit) != "null" || candidate.Command != nil && string(candidate.Command) != "null" || candidate.Disabled != nil && string(candidate.Disabled) != "null" {
			return scalaAction{}, fmt.Errorf("metals extract-method action must be lazy and enabled")
		}
		if chosen != nil {
			return scalaAction{}, fmt.Errorf("multiple matching Metals extract-method actions are available")
		}
		chosen = candidate
	}
	if chosen == nil {
		return scalaAction{}, fmt.Errorf("metals did not return one method-local ExtractMethodCodeAction")
	}
	return *chosen, nil
}

func validateScalaResolvedExtractMethod(raw json.RawMessage, original scalaAction) (scalaAction, error) {
	var resolved scalaAction
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&resolved); err != nil {
		return scalaAction{}, fmt.Errorf("malformed resolved Metals extract-method action: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return scalaAction{}, fmt.Errorf("resolved Metals extract-method action has trailing data")
	}
	if resolved.Title != original.Title || resolved.Kind != original.Kind || !bytes.Equal(bytes.TrimSpace(resolved.Data), bytes.TrimSpace(original.Data)) || len(resolved.Edit) == 0 || string(resolved.Edit) == "null" || len(resolved.Command) > 0 && string(resolved.Command) != "null" || len(resolved.Disabled) > 0 && string(resolved.Disabled) != "null" {
		return scalaAction{}, fmt.Errorf("resolved Metals action changed identity or did not return one direct workspace edit")
	}
	return resolved, nil
}

func decodeScalaExtractMethodEdits(file string, source []byte, raw json.RawMessage, selected scalaRange, statement, indent string) ([]scalaOffsetEdit, string, error) {
	edits, err := decodeScalaExtractionFileEdits(file, raw, "extract-method", 2)
	if err != nil {
		return nil, "", err
	}
	selectedStart, err := scalaByteOffset(source, selected.Start)
	if err != nil {
		return nil, "", err
	}
	selectedEnd, err := scalaByteOffset(source, selected.End)
	if err != nil || selectedEnd <= selectedStart {
		return nil, "", fmt.Errorf("invalid Metals extract-method selection")
	}
	decoded := make([]scalaOffsetEdit, 0, 2)
	var replacement, insertion *scalaOffsetEdit
	for _, edit := range edits {
		start, startErr := scalaByteOffset(source, edit.Range.Start)
		end, endErr := scalaByteOffset(source, edit.Range.End)
		if startErr != nil || endErr != nil || end < start {
			return nil, "", fmt.Errorf("invalid Metals extract-method edit range")
		}
		decodedEdit := scalaOffsetEdit{start: start, end: end, text: edit.NewText}
		decoded = append(decoded, decodedEdit)
		switch {
		case start == selectedStart && end == selectedEnd:
			replacement = &decoded[len(decoded)-1]
		case start == selectedStart && end == selectedStart:
			insertion = &decoded[len(decoded)-1]
		default:
			return nil, "", fmt.Errorf("metals extract-method edit is outside the selected statement")
		}
	}
	if replacement == nil || insertion == nil {
		return nil, "", fmt.Errorf("metals extract-method edit must replace the statement and insert one local method at its start")
	}
	generated, err := validateScalaExtractMethodInsertion(insertion.text, statement, indent)
	if err != nil {
		return nil, "", fmt.Errorf("metals extract-method edits do not match one no-argument Unit helper: %w", err)
	}
	if replacement.text != generated+"()" {
		return nil, "", fmt.Errorf("metals extract-method replacement does not call the generated helper")
	}
	insertion.text = normalizedScalaHelperInsertion(generated, statement, indent)
	if bytes.Contains(source, []byte(generated)) {
		return nil, "", fmt.Errorf("metals generated helper name %q already appears in the source", generated)
	}
	return decoded, generated, nil
}

func normalizedScalaHelperInsertion(name, statement, indent string) string {
	lines := strings.Split(strings.TrimSpace(statement), "\n")
	for index := range lines {
		lines[index] = strings.TrimSpace(lines[index])
	}
	if len(lines) > 1 {
		return "def " + name + "(): Unit = {\n" + indent + "  " + strings.Join(lines, "\n"+indent+"  ") + "\n" + indent + "}\n\n" + indent
	}
	return "def " + name + "(): Unit =\n" + indent + "  " + strings.Join(lines, "\n"+indent+"  ") + "\n\n" + indent
}

func validateScalaExtractMethodInsertion(insertion, statement, indent string) (string, error) {
	statementLines := strings.Split(strings.TrimSpace(statement), "\n")
	for index := range statementLines {
		statementLines[index] = strings.TrimSpace(statementLines[index])
	}
	lines := strings.Split(strings.Trim(insertion, "\n"), "\n")
	header := regexp.MustCompile(`^def ([A-Za-z_][A-Za-z0-9_]*)\(\): Unit =$`).FindStringSubmatch(lines[0])
	if len(lines) != len(statementLines)+3 || len(header) != 2 || !scalaIdentifier(header[1]) || strings.TrimSpace(lines[len(lines)-2]) != "" || strings.TrimSpace(lines[len(lines)-1]) != "" {
		return "", fmt.Errorf("metals helper insertion has unexpected parameters, body, or separator")
	}
	for index, expected := range statementLines {
		line := lines[index+1]
		if !strings.HasPrefix(line, indent) || strings.TrimSpace(line) != expected {
			return "", fmt.Errorf("metals helper insertion has an unexpected statement body")
		}
	}
	return header[1], nil
}

func applyScalaOffsetEdits(source []byte, edits []scalaOffsetEdit) ([]byte, error) {
	candidate := bytes.Clone(source)
	ordered := append([]scalaOffsetEdit(nil), edits...)
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].start != ordered[right].start {
			return ordered[left].start > ordered[right].start
		}
		return ordered[left].end > ordered[right].end
	})
	for _, edit := range ordered {
		if edit.start < 0 || edit.end > len(candidate) || edit.start > edit.end {
			return nil, fmt.Errorf("metals text edit lies outside the source")
		}
		candidate = append(append(bytes.Clone(candidate[:edit.start]), []byte(edit.text)...), candidate[edit.end:]...)
	}
	return candidate, nil
}

func scalaLineIndent(source []byte, start scalaPosition) string {
	lineStart, _ := scalaByteOffset(source, scalaPosition{Line: start.Line})
	if lineStart < 0 || lineStart > len(source) {
		return ""
	}
	lineEnd := len(source)
	if offset := bytes.IndexByte(source[lineStart:], '\n'); offset >= 0 {
		lineEnd = lineStart + offset
	}
	line := source[lineStart:lineEnd]
	return string(line[:len(line)-len(bytes.TrimLeft(line, " \t"))])
}

func scalaExtractedMethodSpans(candidate []byte, generated string) (int, int, error) {
	declaration := []byte("def " + generated + "(): Unit")
	declarationAt := bytes.Index(candidate, declaration)
	if declarationAt < 0 || bytes.Contains(candidate[declarationAt+len(declaration):], declaration) {
		return 0, 0, fmt.Errorf("metals did not produce one local helper declaration")
	}
	definitionStart := declarationAt + len("def ")
	call := []byte(generated + "()")
	callAt := declarationAt + len(declaration) + bytes.Index(candidate[declarationAt+len(declaration):], call)
	if callAt < declarationAt+len(declaration) || bytes.Contains(candidate[callAt+len(call):], call) {
		return 0, 0, fmt.Errorf("metals did not produce one helper call after its declaration")
	}
	return definitionStart, callAt, nil
}

func renameScalaExtractedMethod(ctx context.Context, session ScalaSession, uri string, candidate []byte, generated, requested string, definitionStart, callStart int) ([]byte, error) {
	return renameScalaExtractedSymbol(ctx, session, uri, candidate, generated, requested, scalaRenameTarget{
		position:    definitionStart,
		definition:  definitionStart,
		reference:   callStart,
		description: "extracted method",
	})
}
