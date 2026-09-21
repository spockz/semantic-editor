// Package backend_test exercises the public standalone Haskell lookup contract with fake sessions.
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

type fakeHaskellSession struct {
	symbols    json.RawMessage
	methods    []string
	initialize map[string]any
	cancel     bool
}

func (f *fakeHaskellSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
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

func (f *fakeHaskellSession) Notify(_ context.Context, method string, _ any) error {
	f.methods = append(f.methods, method)
	return nil
}

func (f *fakeHaskellSession) Close() error { return nil }

func haskellFixture(t *testing.T, source string) (string, string) {
	t.Helper()
	root := t.TempDir()
	file := filepath.Join(root, "Thing.hs")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func trustedHaskellProject(root, file string) backend.ProjectContext {
	return backend.ProjectContext{
		RootDir: root, File: file, Language: backend.LanguageHaskell,
		HaskellStandalone: true, WorkspaceTrust: backend.NewWorkspaceTrust(root, true),
	}
}

func TestHaskellLookupInitializesUTF16AndHierarchy(t *testing.T) {
	root, file := haskellFixture(t, "module M where\nvalue = 😀\ndata Type = Constructor { field :: Int }\n")
	session := &fakeHaskellSession{symbols: json.RawMessage(`[{
  "name":"M","kind":2,"range":{"start":{"line":0,"character":0},"end":{"line":2,"character":39}},"selectionRange":{"start":{"line":0,"character":7},"end":{"line":0,"character":8}},
  "children":[{"name":"value","kind":12,"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":10}},"selectionRange":{"start":{"line":1,"character":0},"end":{"line":1,"character":5}}},{"name":"emoji","kind":12,"range":{"start":{"line":1,"character":8},"end":{"line":1,"character":10}},"selectionRange":{"start":{"line":1,"character":8},"end":{"line":1,"character":10}}},{"name":"Type","kind":5,"range":{"start":{"line":2,"character":0},"end":{"line":2,"character":39}},"selectionRange":{"start":{"line":2,"character":5},"end":{"line":2,"character":9}},"children":[{"name":"Constructor","kind":9,"range":{"start":{"line":2,"character":12},"end":{"line":2,"character":23}},"selectionRange":{"start":{"line":2,"character":12},"end":{"line":2,"character":23}},"children":[{"name":"field","kind":8,"range":{"start":{"line":2,"character":26},"end":{"line":2,"character":39}},"selectionRange":{"start":{"line":2,"character":26},"end":{"line":2,"character":31}}}]}]}]}]`)}
	underTest := backend.NewHaskellBackendWithFactory(func(context.Context, string, backend.HaskellConfig) (backend.HaskellSession, error) {
		return session, nil
	})
	result, err := underTest.Lookup(context.Background(), trustedHaskellProject(root, file), "M.Type.Constructor.field")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result.Kind != "field" || result.Location.Range.Start.Character != 26 || result.Offset == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	emoji, err := underTest.Lookup(context.Background(), trustedHaskellProject(root, file), "emoji")
	if err != nil || emoji.Location.Range.Start.Character != 8 || emoji.Location.Range.End.Character != 10 || emoji.Offset != 23 {
		t.Fatalf("non-BMP UTF-16 result=%+v err=%v", emoji, err)
	}
	if len(session.methods) < 4 || session.methods[0] != "initialize" || session.methods[1] != "initialized" || session.methods[2] != "workspace/didChangeConfiguration" || session.methods[3] != "textDocument/didOpen" {
		t.Fatalf("session lifecycle = %v", session.methods)
	}
	caps := session.initialize["capabilities"].(map[string]any)
	if caps["general"].(map[string]any)["positionEncodings"].([]string)[0] != "utf-16" {
		t.Fatalf("missing UTF-16 capability: %#v", caps)
	}
	settings := session.initialize["settings"].(map[string]any)["haskell"].(map[string]any)
	if settings["checkProject"] != false || settings["checkParents"] != "NeverCheck" {
		t.Fatalf("unsafe HLS settings: %#v", settings)
	}
}

func TestHaskellLookupRejectsTrustBeforeSessionFactory(t *testing.T) {
	root, file := haskellFixture(t, "module M where\n")
	started := false
	underTest := backend.NewHaskellBackendWithFactory(func(context.Context, string, backend.HaskellConfig) (backend.HaskellSession, error) {
		started = true
		return nil, nil
	})
	_, err := underTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageHaskell, HaskellStandalone: true}, "M")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) || started {
		t.Fatalf("err=%v started=%v", err, started)
	}
}

func TestHaskellStandaloneRejectsProjectMarkersAndNonHSFiles(t *testing.T) {
	root, file := haskellFixture(t, "module M where\n")
	if err := os.WriteFile(filepath.Join(root, "package.yaml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	underTest := backend.NewHaskellBackendWithFactory(func(context.Context, string, backend.HaskellConfig) (backend.HaskellSession, error) { return nil, nil })
	_, err := underTest.Lookup(context.Background(), trustedHaskellProject(root, file), "M")
	if !errors.Is(err, backend.ErrHaskellProjectUnsupported) {
		t.Fatalf("marker error = %v", err)
	}
	for _, name := range []string{"Thing.lhs", "Thing.hs-boot"} {
		bad := filepath.Join(root, name)
		if err := os.WriteFile(bad, []byte(""), 0o600); err != nil {
			t.Fatal(err)
		}
		project := trustedHaskellProject(root, bad)
		if _, err := underTest.Lookup(context.Background(), project, "M"); !errors.Is(err, backend.ErrHaskellFileRequired) {
			t.Fatalf("%s error = %v", name, err)
		}
	}
}

func TestHaskellLookupReturnsAmbiguityAndRejectsMalformedResponses(t *testing.T) {
	root, file := haskellFixture(t, "module M where\nfirst = 1\nsecond = 2\n")
	raw := json.RawMessage(`[{
 "name":"first","kind":12,"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":9}},"selectionRange":{"start":{"line":1,"character":0},"end":{"line":1,"character":5}}
},{"name":"first","kind":12,"range":{"start":{"line":2,"character":0},"end":{"line":2,"character":10}},"selectionRange":{"start":{"line":2,"character":0},"end":{"line":2,"character":5}}}]`)
	underTest := backend.NewHaskellBackendWithFactory(func(context.Context, string, backend.HaskellConfig) (backend.HaskellSession, error) {
		return &fakeHaskellSession{symbols: raw}, nil
	})
	result, err := underTest.Lookup(context.Background(), trustedHaskellProject(root, file), "first")
	if err != nil || !result.Ambiguous || len(result.Candidates) != 2 || result.Candidates[0].Location.Range.Start.Character != 0 || result.Candidates[0].SelectionRange == nil {
		t.Fatalf("ambiguity result=%+v err=%v", result, err)
	}
	underTest = backend.NewHaskellBackendWithFactory(func(context.Context, string, backend.HaskellConfig) (backend.HaskellSession, error) {
		return &fakeHaskellSession{symbols: json.RawMessage(`[ {"name":"first","kind":12,"location":{} } ]`)}, nil
	})
	if _, err := underTest.Lookup(context.Background(), trustedHaskellProject(root, file), "first"); !errors.Is(err, backend.ErrHaskellUnsupportedResponse) {
		t.Fatalf("flat response error = %v", err)
	}
}

func TestHaskellLookupPropagatesCancellationAndClose(t *testing.T) {
	root, file := haskellFixture(t, "module M where\n")
	session := &fakeHaskellSession{cancel: true}
	underTest := backend.NewHaskellBackendWithFactory(func(context.Context, string, backend.HaskellConfig) (backend.HaskellSession, error) {
		return session, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := underTest.Lookup(ctx, trustedHaskellProject(root, file), "M"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation error = %v", err)
	}
	if err := underTest.Close(); err != nil {
		t.Fatalf("close error = %v", err)
	}
}

func TestHaskellLookupRejectsUnsupportedOperations(t *testing.T) {
	underTest := backend.NewHaskellBackend()
	if _, err := underTest.Rename(context.Background(), backend.RenameRequest{Project: backend.ProjectContext{}, To: "Other"}); !errors.Is(err, backend.ErrUnsupportedOperation) {
		t.Fatalf("rename error = %v", err)
	}
	if _, err := underTest.Verify(context.Background(), backend.VerifyRequest{}); !errors.Is(err, backend.ErrUnsupportedOperation) {
		t.Fatalf("verify error = %v", err)
	}
}

func TestHaskellRequiresExplicitStandaloneSelection(t *testing.T) {
	root, file := haskellFixture(t, "module M where\n")
	service := backend.NewDefaultService()
	project := backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	if _, err := service.Lookup(context.Background(), project, "M"); !errors.Is(err, backend.ErrLanguageUndetected) {
		t.Fatalf("auto Haskell selection error = %v", err)
	}
	project.Language = backend.LanguageHaskell
	if _, err := service.Lookup(context.Background(), project, "M"); !errors.Is(err, backend.ErrHaskellStandaloneRequired) {
		t.Fatalf("implicit standalone error = %v", err)
	}
}
