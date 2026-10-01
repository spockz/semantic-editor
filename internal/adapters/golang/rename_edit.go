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
	selectedEdited := false
	seen := make(map[string]bool)
	outputs := make([]renameOutput, 0, len(edit.DocumentChanges))
	for _, change := range edit.DocumentChanges {
		if change.TextDocument == nil || change.TextDocument.Version == nil || len(change.Edits) == 0 {
			return nil, errors.New("rename edit lacks a versioned document or text edits")
		}
		uri, err := url.Parse(change.TextDocument.URI)
		if err != nil || uri.Scheme != "file" || uri.Host != "" || uri.RawQuery != "" || uri.Fragment != "" || !filepath.IsAbs(uri.Path) {
			return nil, errors.New("rename edit has an invalid local file URI")
		}
		path := filepath.Clean(uri.Path)
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("resolve rename file: %w", err)
		}
		if path != canonical || !pathutil.PathWithin(session.Root, path) || filepath.Ext(path) != ".go" {
			return nil, fmt.Errorf("unsafe rename file %q", path)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat rename file: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("rename file %q is not regular", path)
		}
		if seen[path] {
			return nil, fmt.Errorf("duplicate rename document %q", path)
		}
		seen[path] = true
		document, err := session.Snapshot(path)
		if err != nil {
			return nil, err
		}
		if *change.TextDocument.Version != document.Version {
			return nil, fmt.Errorf("stale rename document version for %q", path)
		}
		spans := make([]renameSpan, 0, len(change.Edits))
		for _, textEdit := range change.Edits {
			if textEdit.NewText == nil || !utf8.ValidString(*textEdit.NewText) {
				return nil, errors.New("rename edit lacks valid replacement text")
			}
			start, end, err := renameOffsets(document.Content, textEdit.Range)
			if err != nil {
				return nil, err
			}
			if start == end || string(document.Content[start:end]) != oldName || *textEdit.NewText != newName {
				return nil, errors.New("rename edit does not replace the selected identifier with the requested name")
			}
			if document.Path == selected.Path && start == selectedStart && end == selectedEnd {
				selectedEdited = true
			}
			spans = append(spans, renameSpan{start: start, end: end, text: *textEdit.NewText})
		}
		if err := validateRenameTokens(document.Content, spans, oldName); err != nil {
			return nil, err
		}
		slices.SortFunc(spans, func(a, b renameSpan) int {
			if a.start != b.start {
				return cmp.Compare(a.start, b.start)
			}
			return cmp.Compare(a.end, b.end)
		})
		for index := 1; index < len(spans); index++ {
			if spans[index].start < spans[index-1].end || spans[index].start == spans[index-1].start {
				return nil, errors.New("overlapping rename edits")
			}
		}
		content := slices.Clone(document.Content)
		for _, span := range slices.Backward(spans) {

			updated := make([]byte, 0, len(content)+len(span.text)-(span.end-span.start))
			updated = append(updated, content[:span.start]...)
			updated = append(updated, span.text...)
			updated = append(updated, content[span.end:]...)
			content = updated
		}
		_, err = parser.ParseFile(token.NewFileSet(), path, content, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("rename produced invalid Go source %q: %w", path, err)
		}
		outputs = append(outputs, renameOutput{document: document, content: content})
	}
	if !selectedEdited {
		return nil, errors.New("rename edit does not include the prepared target")
	}
	slices.SortFunc(outputs, func(a, b renameOutput) int { return strings.Compare(a.document.Path, b.document.Path) })
	return outputs, nil
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
	if !token.IsIdentifier(newName) || newName == "_" {
		return fmt.Errorf("invalid rename identifier %q", newName)
	}
	if line < 1 || column < 1 {
		return errors.New("invalid rename coordinate")
	}
	offset := 0
	for n := 1; n < line; n++ {
		next := bytes.IndexByte(document.Content[offset:], '\n')
		if next < 0 {
			return errors.New("rename line is outside source")
		}
		offset += next + 1
	}
	start := offset
	offset += column - 1
	end := bytes.IndexByte(document.Content[start:], '\n')
	if end < 0 {
		end = len(document.Content) - start
	}
	if offset > start+end || !utf8.Valid(document.Content[:offset]) {
		return errors.New("rename column is outside a UTF-8 source boundary")
	}
	position := backend.PositionFromByteOffset(document.Content, offset)
	params := map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": position}
	raw, err := session.Request(ctx, "textDocument/prepareRename", params)
	if err != nil {
		return fmt.Errorf("prepare Go rename: %w", err)
	}
	var prepared struct {
		Range       *renameRange `json:"range"`
		Placeholder string       `json:"placeholder"`
	}
	if err := json.Unmarshal(raw, &prepared); err != nil {
		return fmt.Errorf("decode prepared rename: %w", err)
	}
	if prepared.Range == nil {
		return errors.New("gopls declined rename")
	}
	begin, finish, err := renameOffsets(document.Content, prepared.Range)
	if err != nil {
		return fmt.Errorf("prepared rename range: %w", err)
	}
	if offset < begin || offset >= finish || !token.IsIdentifier(string(document.Content[begin:finish])) || prepared.Placeholder != string(document.Content[begin:finish]) {
		return errors.New("prepared rename does not match the selected identifier")
	}
	params["newName"] = newName
	raw, err = session.Request(ctx, "textDocument/rename", params)
	if err != nil {
		return fmt.Errorf("request Go rename: %w", err)
	}
	outputs, err := planRenameEdits(session, raw, document, begin, finish, prepared.Placeholder, newName)
	if err != nil {
		return err
	}
	if err := session.AssertCurrent(); err != nil {
		return err
	}
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
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
	published := make(map[string][]byte, len(outputs))
	for index, output := range outputs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("rename published %d of %d files: %w", index, len(outputs), err)
		}
		if err := session.AssertRevision(published); err != nil {
			return fmt.Errorf("rename published %d of %d files: %w", index, len(outputs), err)
		}
		current, err := session.Snapshot(output.document.Path)
		if err != nil {
			return fmt.Errorf("rename published %d of %d files: %w", index, len(outputs), err)
		}
		if !bytes.Equal(current.Content, output.document.Content) {
			return fmt.Errorf("rename published %d of %d files: source changed", index, len(outputs))
		}
		if err := sourcebuf.WriteFileChecked(ctx, output.document.Path, output.content, func() error {
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
			if current.Path != output.document.Path || !os.SameFile(current.Info, output.document.Info) || !bytes.Equal(current.Content, output.document.Content) {
				return errors.New("rename source changed during staging")
			}
			return nil
		}); err != nil {
			return fmt.Errorf("rename published %d of %d files: %w", index, len(outputs), err)
		}
		published[output.document.Path] = output.content
	}
	if record != nil {
		record(renameOutputFiles(outputs))
	}
	return nil
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
