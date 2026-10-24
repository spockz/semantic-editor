// Package scala captures one bounded Metals extraction and requested local rename.
// WHY: The adapter must validate the compiler action and every unpublished edit before staging any Scala source.
package scala

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type scalaAction struct {
	Title       string            `json:"title"`
	Kind        string            `json:"kind"`
	Edit        json.RawMessage   `json:"edit,omitempty"`
	Command     json.RawMessage   `json:"command,omitempty"`
	Disabled    json.RawMessage   `json:"disabled,omitempty"`
	Data        json.RawMessage   `json:"data,omitempty"`
	Diagnostics []json.RawMessage `json:"diagnostics,omitempty"`
}

const (
	scalaOpCodeAction    = "codeAction"
	scalaRefactorExtract = "refactor.extract"
	scalaRefactorInline  = backend.CodeActionInlineKind
)

type scalaEdit struct {
	Range        scalaRange `json:"range"`
	NewText      string     `json:"newText"`
	AnnotationID string     `json:"annotationId"`
}

type scalaVersionedEdits struct {
	TextDocument struct {
		URI     string `json:"uri"`
		Version *int   `json:"version"`
	} `json:"textDocument"`
	Edits []scalaEdit `json:"edits"`
}

type scalaExtractShape struct {
	start, end         int
	openStart, openEnd int
	callEnd            int
	callee             string
	indent             string
}

var scalaIntegerArithmetic = regexp.MustCompile(`^[0-9]+(?:\s*[+*-]\s*[0-9]+)+$`)

// ApplyCodeAction validates and captures the verified Metals ExtractValue action.
func (b *ScalaBackend) ApplyCodeAction(ctx context.Context, request backend.CodeActionRequest) (result *backend.CodeActionResult, returnErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateScalaCodeActionRequest(request); err != nil {
		return nil, err
	}
	root, file, source, err := resolveScalaProject(ctx, request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: request.Operation, Language: backend.LanguageScala, Workspace: root}
	}
	digest := sha256.Sum256(source)
	if request.ExpectedRevision != "" && request.ExpectedRevision != hex.EncodeToString(digest[:]) {
		return nil, fmt.Errorf("scala extraction source revision changed")
	}
	if request.Operation == backend.OperationInlineVariable {
		shape, shapeErr := validateScalaInlineShape(source, request)
		if shapeErr != nil {
			return nil, shapeErr
		}
		return b.applyScalaInlineValue(ctx, request, root, file, source, shape)
	}
	if regexp.MustCompile(`\b` + regexp.QuoteMeta(request.NewName) + `\b`).Match(source) {
		return nil, fmt.Errorf("requested Scala local name %q already appears in the selected file", request.NewName)
	}
	expression, shape, err := validateScalaExtractionSelection(source, request)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureScalaSessionLocked(ctx, root, request.Project.Scala, scalaOpCodeAction); err != nil {
		return nil, err
	}
	if request.Operation == backend.OperationExtractFunction {
		return b.applyScalaExtractCallableLocked(ctx, request, root, file, source, expression)
	}
	return b.applyScalaExtractVariableLocked(ctx, request, root, file, source, expression, shape)
}

func validateScalaCodeActionRequest(request backend.CodeActionRequest) error {
	switch request.Operation {
	case backend.OperationInlineVariable:
		if request.Kind != scalaRefactorInline || !scalaIdentifier(request.Symbol) || request.EnclosingSymbol == "" {
			return fmt.Errorf("unsupported scala code action request")
		}
	case backend.OperationExtractVariable:
		if request.Kind != scalaRefactorExtract || !scalaIdentifier(request.NewName) {
			return fmt.Errorf("unsupported scala code action request")
		}
		if !scalaIdentifier(request.Symbol) {
			return fmt.Errorf("unsupported scala code action request")
		}
	case backend.OperationExtractFunction:
		if request.Kind != scalaRefactorExtract || !scalaIdentifier(request.NewName) {
			return fmt.Errorf("unsupported scala code action request")
		}
		if request.EnclosingSymbol == "" || !scalaCallableStatement(request.Expression) {
			return fmt.Errorf("scala callable extraction requires one supported direct-call statement and owner")
		}
	default:
		return fmt.Errorf("unsupported scala code action request")
	}
	return nil
}

func validateScalaExtractionSelection(source []byte, request backend.CodeActionRequest) (string, scalaExtractShape, error) {
	start, err := scalaByteOffset(source, scalaPosition(request.Range.Start))
	if err != nil {
		return "", scalaExtractShape{}, fmt.Errorf("resolve scala extraction selection: %w", err)
	}
	end, err := scalaByteOffset(source, scalaPosition(request.Range.End))
	if err != nil || end <= start {
		return "", scalaExtractShape{}, fmt.Errorf("invalid scala extraction selection")
	}
	expression := string(source[start:end])
	if request.Operation == backend.OperationExtractFunction {
		if expression != request.Expression || !scalaCallableSelectionIsWholeLine(source, start, end) {
			return "", scalaExtractShape{}, fmt.Errorf("scala callable extraction requires the exact complete statement on one line")
		}
		return expression, scalaExtractShape{}, nil
	}
	shape, err := validateScalaExtractShape(source, request.Symbol, expression, start, end)
	if err != nil {
		return "", scalaExtractShape{}, err
	}
	if shape.start != start || shape.end != end {
		return "", scalaExtractShape{}, fmt.Errorf("scala extraction range does not match the selected owner and expression")
	}
	return expression, shape, nil
}

func validateScalaExtractShape(source []byte, owner, expression string, selectedStart, selectedEnd int) (scalaExtractShape, error) {
	if !scalaIntegerArithmetic.MatchString(expression) {
		return scalaExtractShape{}, fmt.Errorf("scala extraction currently accepts only integer literal arithmetic with +, -, or *")
	}
	for _, literal := range regexp.MustCompile(`[0-9]+`).FindAllString(expression, -1) {
		value, err := strconv.ParseInt(literal, 10, 32)
		if err != nil || value > 2147483647 {
			return scalaExtractShape{}, fmt.Errorf("scala extraction requires Int-range integer literals")
		}
	}
	wholeFile := regexp.MustCompile(`(?s)^\s*object\s+[A-Za-z_][A-Za-z0-9_]*\s*\{\s*\n([ \t]*)def\s+` + regexp.QuoteMeta(owner) + `\s*\(\s*\)\s*:\s*Int\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\(\s*` + regexp.QuoteMeta(expression) + `\s*\)\s*\n[ \t]*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*Int\s*\)\s*:\s*Int\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\s*\n\}\s*$`)
	match := wholeFile.FindSubmatchIndex(source)
	if len(match) < 12 || string(source[match[4]:match[5]]) != string(source[match[6]:match[7]]) || string(source[match[8]:match[9]]) != string(source[match[10]:match[11]]) {
		return scalaExtractShape{}, fmt.Errorf("scala extraction requires one unambiguous object with a zero-argument Int owner and one same-object strict Int callee")
	}
	lineStart := bytes.LastIndex(source[:selectedStart], []byte("\n")) + 1
	lineEnd := len(source)
	if offset := bytes.IndexByte(source[selectedEnd:], '\n'); offset >= 0 {
		lineEnd = selectedEnd + offset
	}
	line := string(source[lineStart:lineEnd])
	callee := string(source[match[4]:match[5]])
	callPrefix := callee + "("
	callStart := strings.Index(line, callPrefix)
	if callStart < 0 || strings.Contains(line[callStart+len(callPrefix):], callPrefix) {
		return scalaExtractShape{}, fmt.Errorf("scala extraction selection is not the unique selected direct-call argument")
	}
	argStart := callStart + len(callPrefix)
	if line[argStart:argStart+len(expression)] != expression || lineStart+argStart != selectedStart || lineStart+argStart+len(expression) != selectedEnd {
		return scalaExtractShape{}, fmt.Errorf("scala extraction selection is not the unique selected direct-call argument")
	}
	ownerPrefix := line[:callStart]
	equals := strings.LastIndex(ownerPrefix, "=")
	if equals < 0 || strings.TrimSpace(ownerPrefix[equals+1:]) != "" {
		return scalaExtractShape{}, fmt.Errorf("scala extraction requires whitespace between the method result type and direct call")
	}
	indent := string(source[match[2]:match[3]])
	return scalaExtractShape{start: selectedStart, end: selectedEnd, openStart: lineStart + equals + 1, openEnd: lineStart + callStart, callEnd: lineEnd, callee: callee, indent: indent}, nil
}

func (b *ScalaBackend) applyScalaExtractVariableLocked(ctx context.Context, request backend.CodeActionRequest, root, file string, source []byte, expression string, shape scalaExtractShape) (result *backend.CodeActionResult, returnErr error) {
	return b.runScalaExtraction(ctx, root, file, source, func(ctx context.Context, session ScalaSession, uri string) ([]byte, func(context.Context) ([]byte, error), error) {
		selectionRaw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}, "range": request.Range, "context": map[string]any{"diagnostics": []any{}, "only": []string{scalaRefactorExtract}}})
		if err != nil {
			return nil, nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
		}
		action, err := selectScalaExtractAction(selectionRaw, expression)
		if err != nil {
			return nil, nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
		}
		editOffsets, err := decodeScalaExtractEdits(file, source, action.Edit, shape, expression)
		if err != nil {
			return nil, nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
		}
		candidate, generatedName, bindingStart, useStart, err := applyScalaExtractEdits(source, editOffsets, shape, expression)
		if err != nil {
			return nil, nil, &ScalaError{Op: scalaOpCodeAction, File: file, Err: err}
		}
		return candidate, func(ctx context.Context) ([]byte, error) {
			renamed, err := renameScalaExtractedLocal(ctx, session, uri, candidate, generatedName, request.NewName, bindingStart, useStart)
			if err != nil {
				return nil, &ScalaError{Op: "rename", File: file, Err: err}
			}
			return renamed, nil
		}, nil
	})
}

func (b *ScalaBackend) runScalaExtraction(ctx context.Context, root, file string, source []byte, capture func(context.Context, ScalaSession, string) ([]byte, func(context.Context) ([]byte, error), error)) (result *backend.CodeActionResult, returnErr error) {
	session := b.session
	uri := pathutil.FileURI(file)
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{scalaTextDocumentKey: map[string]any{"uri": uri, "languageId": "scala", "version": 1, "text": string(source)}}); err != nil {
		return nil, &ScalaError{Op: "didOpen", File: file, Err: errors.Join(err, b.closeSessionLocked())}
	}
	opened := true
	defer func() {
		if opened {
			if closeErr := session.Notify(context.Background(), "textDocument/didClose", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}}); closeErr != nil {
				returnErr = errors.Join(returnErr, &ScalaError{Op: "didClose", File: file, Err: closeErr}, b.closeSessionLocked())
			}
		}
	}()
	candidate, rename, err := capture(ctx, session, uri)
	if err != nil {
		return nil, err
	}
	if changeErr := session.Notify(ctx, "textDocument/didChange", map[string]any{scalaTextDocumentKey: map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]string{{"text": string(candidate)}}}); changeErr != nil {
		return nil, &ScalaError{Op: "didChange", File: file, Err: changeErr}
	}
	renamed, err := rename(ctx)
	if err != nil {
		return nil, err
	}
	if closeErr := session.Notify(ctx, "textDocument/didClose", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}}); closeErr != nil {
		opened = false
		return nil, errors.Join(&ScalaError{Op: "didClose", File: file, Err: closeErr}, b.closeSessionLocked())
	}
	opened = false
	if stageErr := stageScalaExtraction(ctx, root, file, source, renamed); stageErr != nil {
		return nil, stageErr
	}
	return &backend.CodeActionResult{File: file, Before: bytes.Clone(source), After: renamed, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(renamed)}}}, nil
}

func selectScalaExtractAction(raw json.RawMessage, expression string) (scalaAction, error) {
	var actions []scalaAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		return scalaAction{}, fmt.Errorf("malformed metals code actions: %w", err)
	}
	want := "Extract `" + expression + "` as value"
	var selected *scalaAction
	for index := range actions {
		if actions[index].Kind == scalaRefactorExtract && actions[index].Title == want {
			if selected != nil {
				return scalaAction{}, fmt.Errorf("multiple matching metals extract-value actions are available")
			}
			selected = &actions[index]
		}
	}
	if selected == nil || len(selected.Edit) == 0 || string(selected.Edit) == "null" || len(selected.Command) > 0 && string(selected.Command) != "null" || len(selected.Disabled) > 0 && string(selected.Disabled) != "null" {
		return scalaAction{}, fmt.Errorf("metals did not return one direct edit for the selected extract-value action")
	}
	return *selected, nil
}

func decodeScalaExtractionFileEdits(file string, raw json.RawMessage, description string, expectedCount int) ([]scalaEdit, error) {
	var workspaceEdit struct {
		Changes           map[string][]scalaEdit `json:"changes"`
		DocumentChanges   json.RawMessage        `json:"documentChanges"`
		ChangeAnnotations json.RawMessage        `json:"changeAnnotations"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&workspaceEdit); err != nil {
		return nil, fmt.Errorf("malformed Metals %s workspace edit: %w", description, err)
	}
	if err := decoder.Decode(new(any)); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("malformed Metals %s workspace edit trailing data: %w", description, err)
	} else if err == nil {
		return nil, fmt.Errorf("malformed Metals %s workspace edit trailing data", description)
	}
	if len(workspaceEdit.DocumentChanges) != 0 && string(workspaceEdit.DocumentChanges) != "null" || len(workspaceEdit.ChangeAnnotations) != 0 && string(workspaceEdit.ChangeAnnotations) != "null" {
		return nil, fmt.Errorf("metals %s workspace edit includes unsupported document changes or annotations", description)
	}
	uri := pathutil.FileURI(file)
	edits, ok := workspaceEdit.Changes[uri]
	if !ok || len(workspaceEdit.Changes) != 1 || len(edits) != expectedCount {
		return nil, fmt.Errorf("metals %s edit must contain exactly %d edits in the selected file", description, expectedCount)
	}
	for _, edit := range edits {
		if edit.AnnotationID != "" {
			return nil, fmt.Errorf("metals %s edit includes an unsupported annotation", description)
		}
	}
	return edits, nil
}

func decodeScalaExtractEdits(file string, source []byte, raw json.RawMessage, shape scalaExtractShape, expression string) ([]scalaOffsetEdit, error) {
	edits, err := decodeScalaExtractionFileEdits(file, raw, "extract-value", 3)
	if err != nil {
		return nil, err
	}
	decoded := make([]scalaOffsetEdit, 0, len(edits))
	for _, edit := range edits {
		start, err := scalaByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, fmt.Errorf("decode metals edit start: %w", err)
		}
		end, err := scalaByteOffset(source, edit.Range.End)
		if err != nil || end < start {
			return nil, fmt.Errorf("invalid metals extract edit range")
		}
		decoded = append(decoded, scalaOffsetEdit{start: start, end: end, text: edit.NewText})
	}
	var replacement, opening, closing *scalaOffsetEdit
	for index := range decoded {
		edit := &decoded[index]
		switch {
		case edit.start == shape.start && edit.end == shape.end:
			replacement = edit
		case edit.start == shape.openStart && edit.end == shape.openEnd:
			opening = edit
		case edit.start == shape.callEnd && edit.end == shape.callEnd:
			closing = edit
		default:
			return nil, fmt.Errorf("metals extract-value edit is outside the verified direct-call shape")
		}
	}
	if replacement == nil || opening == nil || closing == nil || replacement.text == "" || strings.ContainsAny(replacement.text, " \t\r\n") {
		return nil, fmt.Errorf("metals extract-value edit does not match the verified three-edit shape")
	}
	if !scalaIdentifier(replacement.text) {
		return nil, fmt.Errorf("metals generated an invalid local identifier")
	}
	wantOpen := " {\n" + shape.indent + "  val " + replacement.text + " = " + expression + "\n" + shape.indent + "  "
	if opening.text != wantOpen || closing.text != "\n"+shape.indent+"}" {
		return nil, fmt.Errorf("metals extract-value edits do not preserve the verified method body shape")
	}
	return decoded, nil
}

type scalaOffsetEdit struct {
	start, end int
	text       string
}

func applyScalaExtractEdits(source []byte, edits []scalaOffsetEdit, shape scalaExtractShape, expression string) ([]byte, string, int, int, error) {
	var generated string
	for _, edit := range edits {
		if edit.start == shape.start && edit.end == shape.end {
			generated = edit.text
			break
		}
	}
	if generated == "" {
		return nil, "", 0, 0, fmt.Errorf("metals extraction did not replace the selected expression")
	}
	candidate, err := applyScalaOffsetEdits(source, edits)
	if err != nil {
		return nil, "", 0, 0, err
	}
	bindingText := []byte("val " + generated + " = " + expression)
	binding := bytes.Index(candidate, bindingText)
	if binding < 0 || bytes.Contains(candidate[binding+len(bindingText):], bindingText) {
		return nil, "", 0, 0, fmt.Errorf("metals extraction did not create one verified local binding")
	}
	callText := []byte(shape.callee + "(" + generated + ")")
	call := bytes.Index(candidate, callText)
	if call < 0 || bytes.Contains(candidate[call+len(callText):], callText) || call < binding {
		return nil, "", 0, 0, fmt.Errorf("metals extraction did not preserve one unambiguous direct call after the generated binding")
	}
	return candidate, generated, binding + len("val "), call + len(shape.callee) + 1, nil
}

func renameScalaExtractedLocal(ctx context.Context, session ScalaSession, uri string, candidate []byte, generated, requested string, bindingStart, useStart int) ([]byte, error) {
	return renameScalaExtractedSymbol(ctx, session, uri, candidate, generated, requested, scalaRenameTarget{
		position:    useStart,
		definition:  bindingStart,
		reference:   useStart,
		description: "extracted local",
	})
}

func scalaIdentifier(value string) bool {
	if value == "" || (value[0] != '_' && (value[0] < 'A' || value[0] > 'Z') && (value[0] < 'a' || value[0] > 'z')) {
		return false
	}
	for _, char := range value[1:] {
		if char != '_' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	switch value {
	case "abstract", "case", "catch", "class", "def", "do", "else", "enum", "export", "extends", "false", "final", "finally", "for", "forSome", "given", "if", "implicit", "import", "lazy", "match", "new", "null", "object", "override", "package", "private", "protected", "return", "sealed", "super", "then", "this", "throw", "trait", "true", "try", "type", "val", "var", "while", "with", "yield", "_":
		return false
	default:
		return true
	}
}

func scalaPositionAt(source []byte, offset int) scalaPosition {
	line := 0
	lineStart := 0
	for index := range offset {
		if source[index] == '\n' {
			line++
			lineStart = index + 1
		}
	}
	return scalaPosition{Line: line, Character: len(utf16.Encode([]rune(string(source[lineStart:offset]))))}
}

func stageScalaExtraction(ctx context.Context, root, file string, before, after []byte) error {
	return sourcebuf.ApplyAndPublish(ctx, root, "scala extraction", []sourcebuf.Change{{Path: file, Before: before, After: after}})
}

var _ backend.CodeActionBackend = (*ScalaBackend)(nil)

type scalaRenameTarget struct {
	position    int
	definition  int
	reference   int
	description string
}

func renameScalaExtractedSymbol(ctx context.Context, session ScalaSession, uri string, candidate []byte, generated, requested string, target scalaRenameTarget) ([]byte, error) {
	prepare, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}, "position": scalaPositionAt(candidate, target.position)})
	if err != nil {
		return nil, err
	}
	var prepared struct {
		Start scalaPosition `json:"start"`
		End   scalaPosition `json:"end"`
	}
	if err = json.Unmarshal(prepare, &prepared); err != nil {
		return nil, fmt.Errorf("invalid Metals %s prepareRename response: %w", target.description, err)
	}
	start, startErr := scalaByteOffset(candidate, prepared.Start)
	end, endErr := scalaByteOffset(candidate, prepared.End)
	if startErr != nil || endErr != nil || start != target.position || end != target.position+len(generated) {
		return nil, fmt.Errorf("metals prepareRename did not select only the %s name", target.description)
	}
	raw, err := session.Request(ctx, "textDocument/rename", map[string]any{scalaTextDocumentKey: map[string]string{"uri": uri}, "position": scalaPositionAt(candidate, target.position), "newName": requested})
	if err != nil {
		return nil, err
	}
	var workspaceEdit struct {
		DocumentChanges   []scalaVersionedEdits `json:"documentChanges"`
		Changes           json.RawMessage       `json:"changes"`
		ChangeAnnotations json.RawMessage       `json:"changeAnnotations"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&workspaceEdit); err != nil || len(workspaceEdit.DocumentChanges) != 1 || len(workspaceEdit.Changes) != 0 && string(workspaceEdit.Changes) != "null" || len(workspaceEdit.ChangeAnnotations) != 0 && string(workspaceEdit.ChangeAnnotations) != "null" {
		return nil, fmt.Errorf("metals %s rename must return one versioned selected-file edit", target.description)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("metals %s rename returned trailing workspace edit data", target.description)
	}
	change := workspaceEdit.DocumentChanges[0]
	if change.TextDocument.URI != uri || change.TextDocument.Version == nil || *change.TextDocument.Version != 2 || len(change.Edits) != 2 {
		return nil, fmt.Errorf("metals %s rename must contain exactly two version-2 local edits", target.description)
	}
	want := map[[2]int]bool{
		{target.definition, target.definition + len(generated)}: true,
		{target.reference, target.reference + len(generated)}:   true,
	}
	edits := make([]scalaOffsetEdit, 0, 2)
	for _, edit := range change.Edits {
		if edit.AnnotationID != "" {
			return nil, fmt.Errorf("metals %s rename includes an unsupported annotation", target.description)
		}
		start, startErr := scalaByteOffset(candidate, edit.Range.Start)
		end, endErr := scalaByteOffset(candidate, edit.Range.End)
		span := [2]int{start, end}
		if startErr != nil || endErr != nil || edit.NewText != requested || !want[span] {
			return nil, fmt.Errorf("metals %s rename edited an unexpected source range", target.description)
		}
		delete(want, span)
		edits = append(edits, scalaOffsetEdit{start: start, end: end, text: edit.NewText})
	}
	if len(want) != 0 {
		return nil, fmt.Errorf("metals %s rename omitted one of its name occurrences", target.description)
	}
	return applyScalaOffsetEdits(candidate, edits)
}
