// Package golang validates gopls rename edits at the Go mutation boundary before publishing disk changes.
package golang

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/godiagnostics"
	"semedit/internal/sourcebuf"
	"slices"
	"strings"
	"unicode/utf8"
)

type renameSpan struct {
	start int
	end   int
	text  string
}

type renameOutput struct {
	document godiagnostics.Document
	content  []byte
}

type renamePosition struct {
	Line      *int `json:"line"`
	Character *int `json:"character"`
}

type renameRange struct {
	Start *renamePosition `json:"start"`
	End   *renamePosition `json:"end"`
}

type renameTextEdit struct {
	Range   *renameRange `json:"range"`
	NewText *string      `json:"newText"`
}

type renameDocumentEdit struct {
	TextDocument *struct {
		URI     string `json:"uri"`
		Version *int   `json:"version"`
	} `json:"textDocument"`
	Edits []renameTextEdit `json:"edits"`
}

type renameWorkspaceEdit struct {
	DocumentChanges   []renameDocumentEdit `json:"documentChanges"`
	Changes           json.RawMessage      `json:"changes"`
	ChangeAnnotations json.RawMessage      `json:"changeAnnotations"`
}

type renamePlanRequest struct {
	session      *godiagnostics.Session
	selected     godiagnostics.Document
	selectedFrom int
	selectedTo   int
	oldName      string
	newName      string
}

type renameDocumentPlan struct {
	output         renameOutput
	selectedEdited bool
}

type renameSpanPlan struct {
	spans          []renameSpan
	selectedEdited bool
}

type preparedRename struct {
	params  map[string]any
	start   int
	end     int
	oldName string
}

func renameInSession(ctx context.Context, session *godiagnostics.Session, file string, line, column int, newName string) error {
	return renameInSessionFiles(ctx, session, file, line, column, newName, nil)
}

func renameOffsets(source []byte, span *renameRange) (int, int, error) {
	if span == nil {
		return 0, 0, errors.New("rename edit lacks a range")
	}
	start, err := renameByteOffset(source, span.Start)
	if err != nil {
		return 0, 0, err
	}
	end, err := renameByteOffset(source, span.End)
	if err != nil {
		return 0, 0, err
	}
	if end < start {
		return 0, 0, errors.New("reversed rename range")
	}
	return start, end, nil
}

func renameByteOffset(source []byte, position *renamePosition) (int, error) {
	if position == nil || position.Line == nil || position.Character == nil || *position.Line < 0 || *position.Character < 0 || !utf8.Valid(source) {
		return 0, errors.New("invalid UTF-16 rename position")
	}
	offset := 0
	for range *position.Line {
		next := bytes.IndexByte(source[offset:], '\n')
		if next < 0 {
			return 0, errors.New("rename position line is outside source")
		}
		offset += next + 1
	}
	units := 0
	for offset < len(source) && source[offset] != '\n' && source[offset] != '\r' {
		if units == *position.Character {
			return offset, nil
		}
		runeValue, size := utf8.DecodeRune(source[offset:])
		width := 1
		if runeValue > 0xffff {
			width = 2
		}
		units += width
		offset += size
		if units > *position.Character {
			return 0, errors.New("rename position splits a UTF-16 surrogate pair")
		}
	}
	if units != *position.Character {
		return 0, errors.New("rename position character is outside source")
	}
	return offset, nil
}

func planRenameEdits(session *godiagnostics.Session, raw json.RawMessage, selected godiagnostics.Document, selectedStart, selectedEnd int, oldName, newName string) ([]renameOutput, error) {
	var edit renameWorkspaceEdit
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&edit); err != nil {
		return nil, fmt.Errorf("decode Go rename edit: %w", err)
	}
	if len(edit.Changes) > 0 || len(edit.ChangeAnnotations) > 0 || len(edit.DocumentChanges) == 0 {
		return nil, errors.New("rename requires nonempty versioned document changes without annotations")
	}
	request := renamePlanRequest{
		session: session, selected: selected, selectedFrom: selectedStart, selectedTo: selectedEnd, oldName: oldName, newName: newName,
	}
	selectedEdited := false
	seen := make(map[string]bool)
	outputs := make([]renameOutput, 0, len(edit.DocumentChanges))
	for _, change := range edit.DocumentChanges {
		plan, err := planRenameDocument(request, change, seen)
		if err != nil {
			return nil, err
		}
		selectedEdited = selectedEdited || plan.selectedEdited
		outputs = append(outputs, plan.output)
	}
	if !selectedEdited {
		return nil, errors.New("rename edit does not include the prepared target")
	}
	slices.SortFunc(outputs, func(a, b renameOutput) int { return strings.Compare(a.document.Path, b.document.Path) })
	return outputs, nil
}

func planRenameDocument(request renamePlanRequest, change renameDocumentEdit, seen map[string]bool) (renameDocumentPlan, error) {
	document, err := snapshotRenameDocument(request.session, change, seen)
	if err != nil {
		return renameDocumentPlan{}, err
	}
	spanPlan, err := collectRenameSpans(request, document, change.Edits)
	if err != nil {
		return renameDocumentPlan{}, err
	}
	content := applyRenameSpans(document.Content, spanPlan.spans)
	if _, err := parser.ParseFile(token.NewFileSet(), document.Path, content, parser.AllErrors); err != nil {
		return renameDocumentPlan{}, fmt.Errorf("rename produced invalid Go source %q: %w", document.Path, err)
	}
	return renameDocumentPlan{output: renameOutput{document: document, content: content}, selectedEdited: spanPlan.selectedEdited}, nil
}

func snapshotRenameDocument(session *godiagnostics.Session, change renameDocumentEdit, seen map[string]bool) (godiagnostics.Document, error) {
	if change.TextDocument == nil || change.TextDocument.Version == nil || len(change.Edits) == 0 {
		return godiagnostics.Document{}, errors.New("rename edit lacks a versioned document or text edits")
	}
	path, err := safeRenameDocumentPath(session, change.TextDocument.URI)
	if err != nil {
		return godiagnostics.Document{}, err
	}
	if seen[path] {
		return godiagnostics.Document{}, fmt.Errorf("duplicate rename document %q", path)
	}
	seen[path] = true
	document, err := session.Snapshot(path)
	if err != nil {
		return godiagnostics.Document{}, err
	}
	if *change.TextDocument.Version != document.Version {
		return godiagnostics.Document{}, fmt.Errorf("stale rename document version for %q", path)
	}
	return document, nil
}

func safeRenameDocumentPath(session *godiagnostics.Session, documentURI string) (string, error) {
	uri, err := url.Parse(documentURI)
	if err != nil || uri.Scheme != "file" || uri.Host != "" || uri.RawQuery != "" || uri.Fragment != "" || !filepath.IsAbs(uri.Path) {
		return "", errors.New("rename edit has an invalid local file URI")
	}
	path := filepath.Clean(uri.Path)
	canonical, err := session.CanonicalSourcePath(path)
	if err != nil {
		return "", fmt.Errorf("resolve rename file: %w", err)
	}
	if path != canonical || !pathutil.PathWithin(session.Root, path) || filepath.Ext(path) != ".go" {
		return "", fmt.Errorf("unsafe rename file %q", path)
	}
	info, err := os.Stat(canonical)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat rename file: %w", err)
	}
	if err == nil && !info.Mode().IsRegular() {
		return "", fmt.Errorf("rename file %q is not regular", path)
	}
	return canonical, nil
}

func collectRenameSpans(request renamePlanRequest, document godiagnostics.Document, edits []renameTextEdit) (renameSpanPlan, error) {
	plan := renameSpanPlan{spans: make([]renameSpan, 0, len(edits))}
	for _, textEdit := range edits {
		span, err := validateRenameSpan(request, document, textEdit)
		if err != nil {
			return renameSpanPlan{}, err
		}
		plan.selectedEdited = plan.selectedEdited || document.Path == request.selected.Path && span.start == request.selectedFrom && span.end == request.selectedTo
		plan.spans = append(plan.spans, span)
	}
	if err := validateRenameTokens(document.Content, plan.spans, request.oldName); err != nil {
		return renameSpanPlan{}, err
	}
	slices.SortFunc(plan.spans, compareRenameSpans)
	if err := rejectOverlappingRenameSpans(plan.spans); err != nil {
		return renameSpanPlan{}, err
	}
	return plan, nil
}

func validateRenameSpan(request renamePlanRequest, document godiagnostics.Document, edit renameTextEdit) (renameSpan, error) {
	if edit.NewText == nil || !utf8.ValidString(*edit.NewText) {
		return renameSpan{}, errors.New("rename edit lacks valid replacement text")
	}
	start, end, err := renameOffsets(document.Content, edit.Range)
	if err != nil {
		return renameSpan{}, err
	}
	if start == end || string(document.Content[start:end]) != request.oldName || *edit.NewText != request.newName {
		return renameSpan{}, errors.New("rename edit does not replace the selected identifier with the requested name")
	}
	return renameSpan{start: start, end: end, text: *edit.NewText}, nil
}

func compareRenameSpans(a, b renameSpan) int {
	if a.start != b.start {
		return cmp.Compare(a.start, b.start)
	}
	return cmp.Compare(a.end, b.end)
}

func rejectOverlappingRenameSpans(spans []renameSpan) error {
	for index := 1; index < len(spans); index++ {
		if spans[index].start < spans[index-1].end || spans[index].start == spans[index-1].start {
			return errors.New("overlapping rename edits")
		}
	}
	return nil
}

func applyRenameSpans(source []byte, spans []renameSpan) []byte {
	content := slices.Clone(source)
	for _, span := range slices.Backward(spans) {
		updated := make([]byte, 0, len(content)+len(span.text)-(span.end-span.start))
		updated = append(updated, content[:span.start]...)
		updated = append(updated, span.text...)
		updated = append(updated, content[span.end:]...)
		content = updated
	}
	return content
}

func validateRenameTokens(source []byte, spans []renameSpan, oldName string) error {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(source))
	var lexer scanner.Scanner
	lexer.Init(file, source, nil, scanner.ScanComments)
	matched := make(map[int]bool, len(spans))
	for {
		position, kind, literal := lexer.Scan()
		if kind == token.EOF {
			break
		}
		start := file.Offset(position)
		if kind == token.IDENT && literal == oldName {
			for index, span := range spans {
				if span.start == start && span.end == start+len(literal) {
					matched[index] = true
				}
			}
		}
		if kind == token.COMMENT {
			for index, span := range spans {
				if span.start >= start && span.end <= start+len(literal) {
					matched[index] = true
				}
			}
		}
	}
	if len(matched) != len(spans) {
		return errors.New("rename edit targets source other than an identifier or documentation comment")
	}
	return nil
}

func renameInSessionFiles(ctx context.Context, session *godiagnostics.Session, file string, line, column int, newName string, record func([]string)) error {
	document, err := session.Snapshot(file)
	if err != nil {
		return err
	}
	if !pathutil.PathWithin(session.Root, document.Path) {
		return errors.New("rename target is outside the module")
	}
	offset, err := selectedRenameOffset(document.Content, line, column, newName)
	if err != nil {
		return err
	}
	prepared, err := prepareSelectedRename(ctx, session, document, offset)
	if err != nil {
		return err
	}
	prepared.params["newName"] = newName
	raw, err := session.Request(ctx, "textDocument/rename", prepared.params)
	if err != nil {
		return fmt.Errorf("request Go rename: %w", err)
	}
	outputs, err := planRenameEdits(session, raw, document, prepared.start, prepared.end, prepared.oldName, newName)
	if err != nil {
		return err
	}
	if err := session.AssertCurrent(); err != nil {
		return err
	}
	return publishRenameOutputs(ctx, session, outputs, record)
}

func selectedRenameOffset(source []byte, line, column int, newName string) (int, error) {
	if !token.IsIdentifier(newName) || newName == "_" {
		return 0, fmt.Errorf("invalid rename identifier %q", newName)
	}
	if line < 1 || column < 1 {
		return 0, errors.New("invalid rename coordinate")
	}
	offset := 0
	for n := 1; n < line; n++ {
		next := bytes.IndexByte(source[offset:], '\n')
		if next < 0 {
			return 0, errors.New("rename line is outside source")
		}
		offset += next + 1
	}
	lineStart := offset
	offset += column - 1
	lineEnd := bytes.IndexByte(source[lineStart:], '\n')
	if lineEnd < 0 {
		lineEnd = len(source) - lineStart
	}
	if offset > lineStart+lineEnd || !utf8.Valid(source[:offset]) {
		return 0, errors.New("rename column is outside a UTF-8 source boundary")
	}
	return offset, nil
}

func prepareSelectedRename(ctx context.Context, session *godiagnostics.Session, document godiagnostics.Document, offset int) (preparedRename, error) {
	params := map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": backend.PositionFromByteOffset(document.Content, offset)}
	raw, err := session.Request(ctx, "textDocument/prepareRename", params)
	if err != nil {
		return preparedRename{}, fmt.Errorf("prepare Go rename: %w", err)
	}
	var prepared struct {
		Range       *renameRange `json:"range"`
		Placeholder string       `json:"placeholder"`
	}
	if decodeErr := json.Unmarshal(raw, &prepared); decodeErr != nil {
		return preparedRename{}, fmt.Errorf("decode prepared rename: %w", decodeErr)
	}
	if prepared.Range == nil {
		return preparedRename{}, errors.New("gopls declined rename")
	}
	begin, finish, err := renameOffsets(document.Content, prepared.Range)
	if err != nil {
		return preparedRename{}, fmt.Errorf("prepared rename range: %w", err)
	}
	if offset < begin || offset >= finish || !token.IsIdentifier(string(document.Content[begin:finish])) || prepared.Placeholder != string(document.Content[begin:finish]) {
		return preparedRename{}, errors.New("prepared rename does not match the selected identifier")
	}
	return preparedRename{params: params, start: begin, end: finish, oldName: prepared.Placeholder}, nil
}

func publishRenameOutputs(ctx context.Context, session *godiagnostics.Session, outputs []renameOutput, record func([]string)) error {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		return publishBufferedRename(ctx, buffer, outputs, record)
	}
	return publishDiskRename(ctx, session, outputs, record)
}

func publishBufferedRename(ctx context.Context, buffer *sourcebuf.Buffer, outputs []renameOutput, record func([]string)) error {
	changes := make([]sourcebuf.Change, 0, len(outputs))
	for _, output := range outputs {
		changes = append(changes, sourcebuf.Change{Path: output.document.Path, Before: output.document.Content, After: output.content})
	}
	if err := buffer.Apply(ctx, changes); err != nil {
		return err
	}
	if record != nil {
		record(renameOutputFiles(outputs))
	}
	return nil
}

func publishDiskRename(ctx context.Context, session *godiagnostics.Session, outputs []renameOutput, record func([]string)) error {
	published := make(map[string][]byte, len(outputs))
	for index, output := range outputs {
		if err := writeRenameOutput(ctx, session, published, output); err != nil {
			return fmt.Errorf("rename published %d of %d files: %w", index, len(outputs), err)
		}
		published[output.document.Path] = output.content
	}
	if record != nil {
		record(renameOutputFiles(outputs))
	}
	return nil
}

func writeRenameOutput(ctx context.Context, session *godiagnostics.Session, published map[string][]byte, output renameOutput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := session.AssertRevision(published); err != nil {
		return err
	}
	current, err := session.Snapshot(output.document.Path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current.Content, output.document.Content) {
		return errors.New("source changed")
	}
	return sourcebuf.WriteFileChecked(ctx, output.document.Path, output.content, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := session.AssertRevision(published); err != nil {
			return err
		}
		staged, err := session.Snapshot(output.document.Path)
		if err != nil {
			return err
		}
		if staged.Path != output.document.Path || !os.SameFile(staged.Info, output.document.Info) || !bytes.Equal(staged.Content, output.document.Content) {
			return errors.New("rename source changed during staging")
		}
		return nil
	})
}

func renameOutputFiles(outputs []renameOutput) []string {
	paths := make([]string, 0, len(outputs))
	for _, output := range outputs {
		if !bytes.Equal(output.document.Content, output.content) {
			paths = append(paths, output.document.Path)
		}
	}
	slices.Sort(paths)
	return paths
}
