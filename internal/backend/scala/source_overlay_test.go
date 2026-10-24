// Package scala tests that language-server input reflects the active semantic source overlay.
package scala

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	neutralbackend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type overlayScalaSession struct{ opened []string }

func (s *overlayScalaSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	if method == "textDocument/documentSymbol" {
		return json.Marshal([]any{map[string]any{
			"name": "Thing", "kind": 5,
			"range":          map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 0, "character": 30}},
			"selectionRange": map[string]any{"start": map[string]int{"line": 0, "character": 7}, "end": map[string]int{"line": 0, "character": 12}},
		}})
	}
	return json.RawMessage(`{}`), nil
}

func (s *overlayScalaSession) Notify(_ context.Context, method string, params any) error {
	if method == "textDocument/didOpen" {
		var request struct {
			TextDocument struct {
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		encoded, err := json.Marshal(params)
		if err != nil {
			return err
		}
		if decodeErr := json.Unmarshal(encoded, &request); decodeErr != nil {
			return decodeErr
		}
		s.opened = append(s.opened, request.TextDocument.Text)
	}
	return nil
}

func (*overlayScalaSession) Close() error { return nil }

func TestResolveScalaProjectReadsBufferedSource(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "Thing.scala")
	disk := []byte("object Thing { val value = 1 }\n")
	if writeErr := os.WriteFile(file, disk, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if authorizeErr := buffer.AuthorizeWrites(context.Background(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	updated := []byte("object Thing { val value = 2 }\n")
	if applyErr := buffer.Apply(context.Background(), []sourcebuf.Change{{Path: file, Before: disk, After: updated}}); applyErr != nil {
		t.Fatal(applyErr)
	}
	_, _, got, err := resolveScalaProject(sourcebuf.WithBuffer(context.Background(), buffer), neutralbackend.ProjectContext{RootDir: root, File: file})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(updated) {
		t.Fatalf("resolveScalaProject() source = %q, want overlay %q", got, updated)
	}
}

func TestLookupAndOutlineUseBufferedSourceAndReportItsView(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "Thing.scala")
	disk := []byte("object Thing { val value = 1 }\n")
	if writeErr := os.WriteFile(file, disk, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if authorizeErr := buffer.AuthorizeWrites(context.Background(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	updated := []byte("object Thing { val value = 2 }\n")
	if applyErr := buffer.Apply(context.Background(), []sourcebuf.Change{{Path: file, Before: disk, After: updated}}); applyErr != nil {
		t.Fatal(applyErr)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	session := &overlayScalaSession{}
	b := NewScalaBackendWithFactory(func(context.Context, string, neutralbackend.ScalaConfig) (ScalaSession, error) { return session, nil })
	lookup, err := b.Lookup(ctx, project, "Thing")
	if err != nil || lookup == nil || lookup.SourceView != "working" {
		t.Fatalf("Lookup() result = %#v, err %v; want working source", lookup, err)
	}
	outline, err := b.Outline(ctx, neutralbackend.OutlineRequest{Project: project, Path: file, IncludeUnexported: true})
	if err != nil || outline == nil || len(outline.Files) != 1 || outline.Files[0].SourceView != "working" {
		t.Fatalf("Outline() result = %#v, err %v; want one working-source file", outline, err)
	}
	if len(session.opened) != 2 || session.opened[0] != string(updated) || session.opened[1] != string(updated) {
		t.Fatalf("LSP didOpen text = %#v, want overlay %q twice", session.opened, updated)
	}
	afterDisk, err := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
	if err != nil || string(afterDisk) != string(disk) {
		t.Fatalf("disk source = %q, err %v; want unchanged %q", afterDisk, err, disk)
	}
}

func TestScalaLookupAndExtractionRefuseDifferentSessionRoots(t *testing.T) {
	writeProject := func(source string) neutralbackend.ProjectContext {
		t.Helper()
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(root, "Thing.scala")
		if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		return neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	}
	firstProject := writeProject("object Thing { val value = 1 }\n")
	source := []byte("object Thing {\n  def run(): Int = consume(1 + 2)\n  def consume(value: Int): Int = value\n}\n")
	secondProject := writeProject(string(source))
	session := &overlayScalaSession{}
	factoryCalls := 0
	adapter := NewScalaBackendWithFactory(func(context.Context, string, neutralbackend.ScalaConfig) (ScalaSession, error) {
		factoryCalls++
		return session, nil
	})
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := adapter.Lookup(t.Context(), firstProject, "Thing"); err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "1 + 2")
	selection := neutralbackend.Range{Start: neutralbackend.Position(scalaPositionAt(source, start)), End: neutralbackend.Position(scalaPositionAt(source, start+len("1 + 2")))}
	cases := []struct {
		name      string
		operation string
		run       func() error
	}{
		{name: "lookup", operation: "lookup", run: func() error {
			_, err := adapter.Lookup(t.Context(), secondProject, "Thing")
			return err
		}},
		{name: "extraction", operation: scalaOpCodeAction, run: func() error {
			_, err := adapter.ApplyCodeAction(t.Context(), neutralbackend.CodeActionRequest{Operation: neutralbackend.OperationExtractVariable, Project: secondProject, Range: selection, Kind: scalaRefactorExtract, Symbol: "run", NewName: "total"})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			var scalaErr *ScalaError
			if !errors.Is(err, ErrScalaSessionConflict) || !errors.As(err, &scalaErr) || scalaErr.Op != tc.operation {
				t.Fatalf("operation error = %v, want %s session conflict", err, tc.operation)
			}
		})
	}
	if factoryCalls != 1 || len(session.opened) != 1 {
		t.Fatalf("factory calls = %d, opened documents = %d; want one original session/document", factoryCalls, len(session.opened))
	}
	after, err := os.ReadFile(secondProject.File) // #nosec G304 -- selected source is an owned temporary fixture.
	if err != nil || string(after) != string(source) {
		t.Fatalf("refused source = %q, err %v; want unchanged %q", after, err, source)
	}
}
