package backend_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"semedit/internal/backend"
	scalabackend "semedit/internal/backend/scala"
)

type fakeScalaSession struct {
	symbols    json.RawMessage
	methods    []string
	initialize map[string]any
	cancel     bool
}

func (f *fakeScalaSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
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
func (f *fakeScalaSession) Notify(_ context.Context, method string, _ any) error {
	f.methods = append(f.methods, method)
	return nil
}
func (f *fakeScalaSession) Close() error { return nil }

func scalaFixture(t *testing.T, source string) (string, string) {
	t.Helper()
	root := t.TempDir()
	file := filepath.Join(root, "Thing.scala")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func trustedScalaProject(root, file string) backend.ProjectContext {
	return backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageScala, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
}

func TestScalaLookupInitializesUTF16AndHierarchicalSymbols(t *testing.T) {
	root, file := scalaFixture(t, "package p;\nclass 😀Thing {\n  int field;\n  void run() {}\n}\n")
	session := &fakeScalaSession{symbols: json.RawMessage(`[{"name":"p","kind":4,"range":{"start":{"line":0,"character":0},"end":{"line":4,"character":1}},"selectionRange":{"start":{"line":0,"character":8},"end":{"line":0,"character":9}},"children":[{"name":"😀Thing","kind":5,"range":{"start":{"line":1,"character":0},"end":{"line":4,"character":1}},"selectionRange":{"start":{"line":1,"character":6},"end":{"line":1,"character":14}},"children":[{"name":"field","kind":8,"range":{"start":{"line":2,"character":2},"end":{"line":2,"character":12}},"selectionRange":{"start":{"line":2,"character":6},"end":{"line":2,"character":11}}}]}]}]`)}
	underTest := scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		return session, nil
	})
	result, err := underTest.Lookup(context.Background(), trustedScalaProject(root, file), "p.😀Thing.field")
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

func TestScalaLookupRejectsTrustBeforeSessionFactory(t *testing.T) {
	root, file := scalaFixture(t, "class Thing {}\n")
	started := false
	underTest := scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		started = true
		return nil, nil
	})
	_, err := underTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageScala}, "Thing")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) || started {
		t.Fatalf("err=%v started=%v", err, started)
	}
}

func TestScalaStandaloneRequiresExplicitRootWhenProjectMarkerExists(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "src", "Thing.scala")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("class Thing {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	session := &fakeScalaSession{symbols: json.RawMessage(`[]`)}
	service, err := backend.NewRegistry(scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		return session, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	_ = service
	if _, err := scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		return session, nil
	}).Lookup(context.Background(), backend.ProjectContext{File: file, Language: backend.LanguageScala, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "Thing"); !errors.Is(err, scalabackend.ErrScalaProjectMarkersRequireExplicitRoot) {
		t.Fatalf("marker error = %v", err)
	}
}

//nolint:dupl // Keep the Scala backend response contract explicit beside its fixture.
func TestScalaLookupRejectsMalformedAndOutOfRootResponses(t *testing.T) {
	root, file := scalaFixture(t, "class Thing {}\n")
	for name, test := range map[string]struct {
		raw  json.RawMessage
		want error
	}{
		"flat":        {raw: json.RawMessage(`[{"name":"Thing","kind":5,"location":{}}]`), want: scalabackend.ErrScalaUnsupportedResponse},
		"out-of-root": {raw: json.RawMessage(`[{"name":"Thing","kind":5,"uri":"file:///tmp/out.scala","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}}}]`), want: scalabackend.ErrScalaMalformedResponse},
	} {
		t.Run(name, func(t *testing.T) {
			underTest := scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
				return &fakeScalaSession{symbols: test.raw}, nil
			})
			_, err := underTest.Lookup(context.Background(), trustedScalaProject(root, file), "Thing")
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestScalaLookupPropagatesCancellation(t *testing.T) {
	root, file := scalaFixture(t, "class Thing {}\n")
	session := &fakeScalaSession{cancel: true}
	underTest := scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		return session, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := underTest.Lookup(ctx, trustedScalaProject(root, file), "Thing")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestScalaLookupRejectsUnsupportedOperations(t *testing.T) {
	backendUnderTest := scalabackend.NewScalaBackend()
	if _, err := backendUnderTest.Rename(context.Background(), backend.RenameRequest{Project: backend.ProjectContext{}, To: "Other"}); !errors.Is(err, backend.ErrUnsupportedOperation) {
		t.Fatalf("rename error = %v", err)
	}
	if _, err := backendUnderTest.Verify(context.Background(), backend.VerifyRequest{}); !errors.Is(err, backend.ErrUnsupportedOperation) {
		t.Fatalf("verify error = %v", err)
	}
}

func TestScalaStandaloneFileWithoutMarkersUsesContainingRoot(t *testing.T) {
	root, file := scalaFixture(t, "object Thing {}\n")
	session := &fakeScalaSession{symbols: json.RawMessage(`[{"name":"Thing","kind":2,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":7},"end":{"line":0,"character":12}}}]`)}
	underTest := scalabackend.NewScalaBackendWithFactory(func(_ context.Context, gotRoot string, _ backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		if gotRoot != backend.CanonicalWorkspaceRoot(root) {
			t.Fatalf("standalone root = %q, want %q", gotRoot, root)
		}
		return session, nil
	})
	if _, err := underTest.Lookup(context.Background(), backend.ProjectContext{File: file, Language: backend.LanguageScala, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "Thing"); err != nil {
		t.Fatalf("standalone lookup failed: %v", err)
	}
}

type statefulScalaSession struct {
	methods     []string
	open        map[string]string
	openedTexts []string
	responses   map[string]json.RawMessage
}

func (f *statefulScalaSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	f.methods = append(f.methods, method)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if method == "initialize" {
		return json.RawMessage("{}"), nil
	}
	var request struct {
		TextDocument struct {
			URI string
		}
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &request); err != nil {
		return nil, err
	}
	source, ok := f.open[request.TextDocument.URI]
	if !ok {
		return nil, errors.New("documentSymbol requested without an open document")
	}
	response, ok := f.responses[source]
	if !ok {
		return nil, errors.New("no symbol response for opened source")
	}
	return response, nil
}

func (f *statefulScalaSession) Notify(ctx context.Context, method string, params any) error {
	f.methods = append(f.methods, method)
	if err := ctx.Err(); err != nil {
		return err
	}
	if method != "textDocument/didOpen" && method != "textDocument/didClose" {
		return nil
	}
	var notification struct {
		TextDocument struct {
			URI  string
			Text string
		}
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &notification); err != nil {
		return err
	}
	if method == "textDocument/didOpen" {
		if _, exists := f.open[notification.TextDocument.URI]; exists {
			return errors.New("document opened twice")
		}
		f.open[notification.TextDocument.URI] = notification.TextDocument.Text
		f.openedTexts = append(f.openedTexts, notification.TextDocument.Text)
		return nil
	}
	if _, exists := f.open[notification.TextDocument.URI]; !exists {
		return errors.New("closed document was not open")
	}
	delete(f.open, notification.TextDocument.URI)
	return nil
}

func (f *statefulScalaSession) Close() error { return nil }

func TestScalaLookupThenOutlineUsesFreshMatchedDocumentSnapshot(t *testing.T) {
	root, file := scalaFixture(t, "class Alpha {}\n")
	sourceA := "class Alpha {}\n"
	sourceB := "class Bravo {}\n"
	responseA := json.RawMessage("[{\"name\":\"Alpha\",\"kind\":5,\"range\":{\"start\":{\"line\":0,\"character\":0},\"end\":{\"line\":0,\"character\":14}},\"selectionRange\":{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":0,\"character\":11}}}]")
	responseB := json.RawMessage("[{\"name\":\"Bravo\",\"kind\":5,\"range\":{\"start\":{\"line\":0,\"character\":0},\"end\":{\"line\":0,\"character\":14}},\"selectionRange\":{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":0,\"character\":11}},\"detail\":\"class Bravo\"}]")
	session := &statefulScalaSession{open: map[string]string{}, responses: map[string]json.RawMessage{sourceA: responseA, sourceB: responseB}}
	factoryCalls := 0
	b := scalabackend.NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (scalabackend.ScalaSession, error) {
		factoryCalls++
		return session, nil
	})
	project := trustedScalaProject(root, file)
	if _, err := b.Lookup(context.Background(), project, "Alpha"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(sourceB), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := b.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: file, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 1 || len(session.openedTexts) != 2 || session.openedTexts[0] != sourceA || session.openedTexts[1] != sourceB || len(session.open) != 0 {
		t.Fatalf("factory calls %d, opened texts %#v, open docs %#v", factoryCalls, session.openedTexts, session.open)
	}
	wantCalls := []string{"initialize", "initialized", "workspace/didChangeConfiguration", "textDocument/didOpen", "textDocument/documentSymbol", "textDocument/didClose", "textDocument/didOpen", "textDocument/documentSymbol", "textDocument/didClose"}
	if !reflect.DeepEqual(session.methods, wantCalls) {
		t.Fatalf("LSP calls = %#v, want %#v", session.methods, wantCalls)
	}
	digest := sha256.Sum256([]byte(sourceB))
	if len(result.Files) != 1 || len(result.Files[0].Symbols) != 1 || result.Files[0].Symbols[0].Name != "Bravo" || result.Files[0].Symbols[0].ServerDetail == nil || *result.Files[0].Symbols[0].ServerDetail != "class Bravo" || result.Files[0].Revision != hex.EncodeToString(digest[:]) {
		t.Fatalf("outline result = %+v", result)
	}
}
