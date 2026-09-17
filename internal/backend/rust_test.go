// Package backend_test exercises the hermetic Rust lookup transport boundary and its trust invariants.
package backend_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/backend"
)

type fakeRustSession struct {
	symbols json.RawMessage
	calls   []string
	init    map[string]any
	closed  bool
	err     error
}

func (f *fakeRustSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	f.calls = append(f.calls, method)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if method == "initialize" {
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(`{"capabilities":{}}`), json.Unmarshal(encoded, &f.init)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.symbols, nil
}

func (f *fakeRustSession) Notify(ctx context.Context, method string, _ any) error {
	f.calls = append(f.calls, method)
	return ctx.Err()
}

func (f *fakeRustSession) Close() error {
	f.closed = true
	return nil
}

func rustProject(t *testing.T, source string) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "lib.rs")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func TestRustLookupStreamsInitializeOpenAndSymbolsWithUTF16(t *testing.T) {
	root, file := rustProject(t, "fn 😀value() {}\n")
	session := &fakeRustSession{symbols: json.RawMessage(`[{"name":"😀value","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":12}},"selectionRange":{"start":{"line":0,"character":3},"end":{"line":0,"character":10}},"children":[]}]`)}
	backendUnderTest := backend.NewRustBackendWithFactory(func(context.Context, string) (backend.RustSession, error) {
		return session, nil
	})
	result, err := backendUnderTest.Lookup(context.Background(), backend.ProjectContext{
		RootDir:        root,
		File:           file,
		WorkspaceTrust: backend.NewWorkspaceTrust(root, true),
	}, "😀value")
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != 3 || result.Line != 1 || result.Column != 4 || result.Kind != "function" {
		t.Fatalf("lookup result = %+v", result)
	}
	if len(session.calls) != 5 || session.calls[0] != "initialize" || session.calls[1] != "initialized" || session.calls[2] != "workspace/didChangeConfiguration" || session.calls[3] != "textDocument/didOpen" || session.calls[4] != "textDocument/documentSymbol" {
		t.Fatalf("LSP call sequence = %#v", session.calls)
	}
	if session.init["initializationOptions"] == nil {
		t.Fatal("initialize omitted rust-analyzer settings")
	}
}

func TestRustLookupRejectsTrustBeforeStartingTransport(t *testing.T) {
	root, file := rustProject(t, "fn value() {}\n")
	started := false
	backendUnderTest := backend.NewRustBackendWithFactory(func(context.Context, string) (backend.RustSession, error) {
		started = true
		return nil, nil
	})
	_, err := backendUnderTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file}, "value")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) || started {
		t.Fatalf("trust error = %v, transport started = %v", err, started)
	}
}

func TestRustServiceTrustUsesDiscoveredCargoRoot(t *testing.T) {
	root, file := rustProject(t, "fn value() {}\n")
	session := &fakeRustSession{symbols: json.RawMessage(`[{"name":"value","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}},"selectionRange":{"start":{"line":0,"character":3},"end":{"line":0,"character":8}}}]`)}
	rendered := backend.NewRustBackendWithFactory(func(context.Context, string) (backend.RustSession, error) {
		return session, nil
	})
	registry, err := backend.NewRegistry(rendered)
	if err != nil {
		t.Fatal(err)
	}
	service := backend.NewService(registry)
	result, err := service.Lookup(context.Background(), backend.ProjectContext{
		File:           file,
		Language:       backend.LanguageRust,
		WorkspaceTrust: backend.NewWorkspaceTrust(root, true),
	}, "value")
	if err != nil || result == nil {
		t.Fatalf("service lookup = %+v, %v", result, err)
	}
}

func TestRustLookupReturnsAllHierarchicalAmbiguityCandidates(t *testing.T) {
	root, file := rustProject(t, "mod a {}\nmod b {}\n")
	session := &fakeRustSession{symbols: json.RawMessage(`[{"name":"a","kind":2,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":8}},"selectionRange":{"start":{"line":0,"character":4},"end":{"line":0,"character":5}},"children":[{"name":"run","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":8}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}}}]},{"name":"b","kind":2,"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":8}},"selectionRange":{"start":{"line":1,"character":4},"end":{"line":1,"character":5}},"children":[{"name":"run","kind":12,"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":8}},"selectionRange":{"start":{"line":1,"character":0},"end":{"line":1,"character":3}}}]}]`)}
	backendUnderTest := backend.NewRustBackendWithFactory(func(context.Context, string) (backend.RustSession, error) {
		return session, nil
	})
	result, err := backendUnderTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "run")
	if err != nil || !result.Ambiguous || len(result.Candidates) != 2 {
		t.Fatalf("ambiguity result = %+v, %v", result, err)
	}
	qualified, err := backendUnderTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "a::run")
	if err != nil || qualified.Ambiguous || qualified.Symbol != "a::run" {
		t.Fatalf("qualified result = %+v, %v", qualified, err)
	}
}

func TestRustLookupRejectsMalformedAndOutOfRootResponses(t *testing.T) {
	root, file := rustProject(t, "fn value() {}\n")
	for name, testCase := range map[string]struct {
		response json.RawMessage
		want     error
	}{
		"flat":        {response: json.RawMessage(`[{"name":"value","kind":12,"location":{}}]`), want: backend.ErrRustUnsupportedResponse},
		"out-of-root": {response: json.RawMessage(`[{"name":"value","kind":12,"uri":"file:///tmp/out.rs","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}}}]`), want: backend.ErrRustMalformedResponse},
	} {
		t.Run(name, func(t *testing.T) {
			backendUnderTest := backend.NewRustBackendWithFactory(func(context.Context, string) (backend.RustSession, error) {
				return &fakeRustSession{symbols: testCase.response}, nil
			})
			_, err := backendUnderTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "value")
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
		})
	}
}
