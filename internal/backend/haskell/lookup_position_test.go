// These lookup tests keep standalone HLS source coordinates aligned with the shared snapshot rules.

package haskell

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	neutralbackend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"path/filepath"
	"testing"
)

func (s *lookupPositionSession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	switch method {
	case "initialize":
		return json.RawMessage("{}"), nil
	case "textDocument/documentSymbol":
		return s.response, nil
	default:
		return nil, fmt.Errorf("unexpected request %q", method)
	}
}

func (*lookupPositionSession) Notify(context.Context, string, any) error { return nil }

func (*lookupPositionSession) Close() error { return nil }

func TestLookupValidatesUTF16SourcePositions(t *testing.T) {
	validRange := neutralbackend.Range{Start: neutralbackend.Position{}, End: neutralbackend.Position{Line: 1, Character: 9}}
	selection := neutralbackend.Range{Start: neutralbackend.Position{Line: 1}, End: neutralbackend.Position{Line: 1, Character: 5}}
	cases := []struct {
		name        string
		source      []byte
		declaration neutralbackend.Range
		selection   neutralbackend.Range
		wantError   bool
	}{
		{name: "LF", source: []byte("-- 😀\nvalue = 1\n"), declaration: validRange, selection: selection},
		{name: "CRLF", source: []byte("-- 😀\r\nvalue = 1\r\n"), declaration: validRange, selection: selection},
		{name: "CR", source: []byte("-- 😀\rvalue = 1\r"), declaration: validRange, selection: selection},
		{name: "after astral rune", source: []byte("-- 😀\nvalue = 1\n"), declaration: neutralbackend.Range{Start: neutralbackend.Position{Character: 5}, End: validRange.End}, selection: selection},
		{name: "split surrogate", source: []byte("-- 😀\nvalue = 1\n"), declaration: neutralbackend.Range{Start: neutralbackend.Position{Character: 4}, End: validRange.End}, selection: selection, wantError: true},
		{name: "CRLF carriage return is outside line", source: []byte("-- 😀\r\nvalue = 1\r\n"), declaration: neutralbackend.Range{Start: neutralbackend.Position{Character: 6}, End: validRange.End}, selection: selection, wantError: true},
		{name: "negative line", source: []byte("-- 😀\nvalue = 1\n"), declaration: validRange, selection: neutralbackend.Range{Start: neutralbackend.Position{Line: -1}, End: selection.End}, wantError: true},
		{name: "negative character", source: []byte("-- 😀\nvalue = 1\n"), declaration: validRange, selection: neutralbackend.Range{Start: neutralbackend.Position{Line: 1, Character: -1}, End: selection.End}, wantError: true},
		{name: "line outside source", source: []byte("-- 😀\nvalue = 1\n"), declaration: validRange, selection: neutralbackend.Range{Start: neutralbackend.Position{Line: 3}, End: neutralbackend.Position{Line: 3, Character: 5}}, wantError: true},
		{name: "character outside line", source: []byte("-- 😀\nvalue = 1\n"), declaration: validRange, selection: neutralbackend.Range{Start: selection.Start, End: neutralbackend.Position{Line: 1, Character: 10}}, wantError: true},
		{name: "reversed range", source: []byte("-- 😀\nvalue = 1\n"), declaration: validRange, selection: neutralbackend.Range{Start: selection.End, End: selection.Start}, wantError: true},
		{name: "invalid UTF-8 after selected declaration", source: []byte("-- 😀\nvalue = 1\n\xff"), declaration: validRange, selection: selection, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runHaskellLookupPositionCase(t, tc)
		})
	}
}

func runHaskellLookupPositionCase(t *testing.T, tc struct {
	name        string
	source      []byte
	declaration neutralbackend.Range
	selection   neutralbackend.Range
	wantError   bool
}) {
	t.Helper()
	root := t.TempDir()
	file := filepath.Join(root, "Value.hs")
	if err := pipeline.WriteAtomic(file, tc.source); err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal([]map[string]any{{"name": "value", "kind": 12, "range": tc.declaration, "selectionRange": tc.selection}})
	if err != nil {
		t.Fatal(err)
	}
	session := &lookupPositionSession{response: response}
	underTest := NewHaskellBackend(WithHaskellSessionFactory(func(context.Context, string) (HaskellSession, error) { return session, nil }))
	defer func() {
		if closeErr := underTest.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	project := neutralbackend.ProjectContext{RootDir: root, File: file, Haskell: neutralbackend.HaskellConfig{Standalone: true}, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	got, lookupErr := underTest.Lookup(context.Background(), project, "value")
	assertHaskellLookupPositionResult(t, got, lookupErr, tc, file)
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, tc.source) {
		t.Fatal("Lookup changed original source bytes")
	}
}

func assertHaskellLookupPositionResult(t *testing.T, got *neutralbackend.LookupResult, lookupErr error, tc struct {
	name        string
	source      []byte
	declaration neutralbackend.Range
	selection   neutralbackend.Range
	wantError   bool
}, file string) {
	t.Helper()
	if tc.wantError {
		if !errors.Is(lookupErr, ErrHaskellMalformedResponse) {
			t.Fatalf("Lookup error = %v, want malformed response", lookupErr)
		}
		if got != nil {
			t.Fatalf("Lookup returned a result for invalid source positions: %#v", got)
		}
		return
	}
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	if got.Definition == nil || got.Definition.Name != "value" || got.Definition.Location.Range != tc.selection || got.Definition.Location.URI != pathutil.FileURI(neutralbackend.CanonicalWorkspaceRoot(file)) {
		t.Fatalf("Lookup definition = %#v, want selected original-file range %#v", got.Definition, tc.selection)
	}
	if got.Definition.DeclarationRange == nil || *got.Definition.DeclarationRange != tc.declaration {
		t.Fatalf("Lookup declaration range = %#v, want %#v", got.Definition.DeclarationRange, tc.declaration)
	}
}

type lookupPositionSession struct{ response json.RawMessage }
