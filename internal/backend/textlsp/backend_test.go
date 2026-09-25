// Package textlsp tests selected-file lookup, trust, and diagnostics receipt contracts.
package textlsp

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/pipeline"
)

type fakeSession struct {
	response json.RawMessage
	opened   string
}

func (f *fakeSession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	if method == "textDocument/documentSymbol" {
		return f.response, nil
	}
	return json.RawMessage(`{"capabilities":{}}`), nil
}
func (f *fakeSession) Notify(_ context.Context, method string, params any) error {
	if method == "textDocument/didOpen" {
		raw, _ := json.Marshal(params)
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		_ = json.Unmarshal(raw, &p)
		f.opened = p.TextDocument.URI
	}
	return nil
}
func (*fakeSession) WaitDiagnostics(ctx context.Context, _ string, _ int) ([]backend.Diagnostic, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (*fakeSession) Close() error { return nil }

func TestLookupRequiresTrustBeforeStartingServer(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.sh")
	if err := pipeline.WriteAtomic(file, []byte("f() {}\n")); err != nil {
		t.Fatal(err)
	}
	started := false
	a := &Adapter{Config: Config{Language: backend.LanguageBash, Extensions: []string{".sh"}}, Factory: func(context.Context, string, string, []string) (Session, error) {
		started = true
		return nil, errors.New("unexpected")
	}}
	_, err := a.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file}, "f")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) {
		t.Fatalf("Lookup error=%v, want trust required", err)
	}
	if started {
		t.Fatal("server started before request trust")
	}
}

func TestLookupUsesSelectedFileAndUTF16Ranges(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.sh")
	source := "# 😀f\n"
	if err := pipeline.WriteAtomic(file, []byte(source)); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{response: json.RawMessage(`[{"name":"f","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"selectionRange":{"start":{"line":0,"character":4},"end":{"line":0,"character":5}}}]`)}
	a := &Adapter{Config: Config{Language: backend.LanguageBash, Extensions: []string{".sh"}, LanguageID: "shell"}, Factory: func(context.Context, string, string, []string) (Session, error) { return session, nil }}
	project := backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	got, err := a.Lookup(context.Background(), project, "f")
	if err != nil {
		t.Fatal(err)
	}
	canonicalFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		t.Fatal(err)
	}
	if got.Offset != 6 || got.Line != 1 || got.Column != 5 || got.Location.URI != pathutil.FileURI(canonicalFile) {
		t.Fatalf("lookup result=%+v", got)
	}
	if got.File != "a.sh" {
		t.Fatalf("result file=%q, want selected path relative to root", got.File)
	}
}
