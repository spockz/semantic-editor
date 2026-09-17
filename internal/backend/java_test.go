package backend_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"semedit/internal/backend"
)

type fakeJavaSession struct {
	symbols    json.RawMessage
	methods    []string
	initialize map[string]any
	cancel     bool
}

func (f *fakeJavaSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	f.methods = append(f.methods, method)
	if method == "initialize" {
		f.initialize = params.(map[string]any)
		return json.RawMessage(`{}`), nil
	}
	if f.cancel {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.symbols, nil
}
func (f *fakeJavaSession) Notify(_ context.Context, method string, _ any) error {
	f.methods = append(f.methods, method)
	return nil
}
func (f *fakeJavaSession) Close() error { return nil }

func javaFixture(t *testing.T, source string) (string, string) {
	t.Helper()
	root := t.TempDir()
	file := filepath.Join(root, "Thing.java")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func trustedJavaProject(root, file string) backend.ProjectContext {
	return backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
}

func TestJavaLookupInitializesUTF16AndHierarchicalSymbols(t *testing.T) {
	root, file := javaFixture(t, "package p;\nclass 😀Thing {\n  int field;\n  void run() {}\n}\n")
	session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"p","kind":4,"range":{"start":{"line":0,"character":0},"end":{"line":4,"character":1}},"selectionRange":{"start":{"line":0,"character":8},"end":{"line":0,"character":9}},"children":[{"name":"😀Thing","kind":5,"range":{"start":{"line":1,"character":0},"end":{"line":4,"character":1}},"selectionRange":{"start":{"line":1,"character":6},"end":{"line":1,"character":14}},"children":[{"name":"field","kind":8,"range":{"start":{"line":2,"character":2},"end":{"line":2,"character":12}},"selectionRange":{"start":{"line":2,"character":6},"end":{"line":2,"character":11}}}]}]}]`)}
	underTest := backend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (backend.JavaSession, error) { return session, nil })
	result, err := underTest.Lookup(context.Background(), trustedJavaProject(root, file), "p.😀Thing.field")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result.Kind != "field" || result.Offset == 0 || result.Location.Range.Start.Character != 6 || result.Location.Range.End.Character != 11 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(session.methods) < 4 || session.methods[0] != "initialize" || session.methods[1] != "initialized" || session.methods[2] != "workspace/didChangeConfiguration" || session.methods[3] != "textDocument/didOpen" {
		t.Fatalf("session lifecycle = %v", session.methods)
	}
	caps := session.initialize["capabilities"].(map[string]any)
	general := caps["general"].(map[string]any)
	if general["positionEncodings"].([]string)[0] != "utf-16" {
		t.Fatalf("missing UTF-16 capability: %#v", general)
	}
	documentSymbol := caps["textDocument"].(map[string]any)["documentSymbol"].(map[string]any)
	if documentSymbol["hierarchicalDocumentSymbolSupport"] != true {
		t.Fatalf("missing hierarchy capability: %#v", documentSymbol)
	}
}

func TestJavaLookupRejectsTrustBeforeSessionFactory(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	started := false
	underTest := backend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (backend.JavaSession, error) {
		started = true
		return nil, nil
	})
	_, err := underTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava}, "Thing")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) || started {
		t.Fatalf("err=%v started=%v", err, started)
	}
}

func TestJavaServiceDiscoversNearestRootAndRejectsSameLevelAmbiguity(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "src", "Thing.java")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("class Thing {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	session := &fakeJavaSession{symbols: json.RawMessage(`[]`)}
	service, err := backend.NewRegistry(backend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (backend.JavaSession, error) { return session, nil }))
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.NewService(service).Lookup(context.Background(), backend.ProjectContext{File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "Thing")
	if err == nil || result != nil || !errors.Is(err, backend.ErrJavaSymbolNotFound) {
		t.Fatalf("nearest root lookup = %#v, %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(root, "build.gradle"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = backend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (backend.JavaSession, error) { return session, nil }).Lookup(context.Background(), backend.ProjectContext{File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "Thing")
	if !errors.Is(err, backend.ErrJavaWorkspaceAmbiguous) {
		t.Fatalf("ambiguity error = %v", err)
	}
}

func TestJavaLookupRejectsMalformedAndOutOfRootResponses(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	for name, test := range map[string]struct {
		raw  json.RawMessage
		want error
	}{
		"flat":        {raw: json.RawMessage(`[{"name":"Thing","kind":5,"location":{}}]`), want: backend.ErrJavaUnsupportedResponse},
		"out-of-root": {raw: json.RawMessage(`[{"name":"Thing","kind":5,"uri":"file:///tmp/out.java","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}}}]`), want: backend.ErrJavaMalformedResponse},
	} {
		t.Run(name, func(t *testing.T) {
			underTest := backend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (backend.JavaSession, error) {
				return &fakeJavaSession{symbols: test.raw}, nil
			})
			_, err := underTest.Lookup(context.Background(), trustedJavaProject(root, file), "Thing")
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestJavaLookupPropagatesCancellation(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	session := &fakeJavaSession{cancel: true}
	underTest := backend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (backend.JavaSession, error) { return session, nil })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := underTest.Lookup(ctx, trustedJavaProject(root, file), "Thing")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation error = %v", err)
	}
}
