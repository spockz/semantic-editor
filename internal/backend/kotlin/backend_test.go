// Package kotlin tests the observable trust and receipt boundaries of the Kotlin adapter.
package kotlin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	neutralbackend "semedit/internal/backend"
	"semedit/internal/lsp"
	"semedit/internal/pipeline"
)

func TestWaitDiagnosticsRequiresMatchingURIReceipt(t *testing.T) {
	root := t.TempDir()
	selectedURI := "file://" + filepath.Join(root, "request-1.kt")
	session := &kotlinProcessSession{root: root, reports: make(map[string]kotlinDiagnosticReceipt), wake: make(chan struct{}, 1), selectedURI: selectedURI}
	wrongParams, _ := json.Marshal(map[string]any{"uri": "file:///outside/Widget.kt", "diagnostics": []any{}})
	session.record(lsp.Notification{Method: "textDocument/publishDiagnostics", Params: wrongParams})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := session.WaitDiagnostics(ctx, selectedURI, 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wrong-URI diagnostics returned error %v, want bounded deadline", err)
	}
}

func TestWaitDiagnosticsAcceptsVersionlessExplicitEmptyReport(t *testing.T) {
	root := t.TempDir()
	selectedURI := "file://" + filepath.Join(root, "request-1.kt")
	session := &kotlinProcessSession{root: root, reports: make(map[string]kotlinDiagnosticReceipt), wake: make(chan struct{}, 1), selectedURI: selectedURI}
	params, _ := json.Marshal(map[string]any{"uri": selectedURI, "diagnostics": []any{}})
	session.record(lsp.Notification{Method: "textDocument/publishDiagnostics", Params: params})

	diagnostics, err := session.WaitDiagnostics(context.Background(), selectedURI, 1)
	if err != nil {
		t.Fatalf("WaitDiagnostics returned error: %v", err)
	}
	if diagnostics == nil || len(diagnostics) != 0 {
		t.Fatalf("WaitDiagnostics returned %#v, want explicit empty diagnostic list", diagnostics)
	}
}

type outlineTestSession struct {
	response string
	uri      string
	text     string
	closed   bool
}

type receiptTestSession struct {
	uri        string
	text       string
	closed     bool
	requestErr error
	closeErr   error
}

func (s *receiptTestSession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	if method == "initialize" && s.requestErr != nil {
		return nil, s.requestErr
	}
	if method == "textDocument/documentSymbol" {
		return json.RawMessage("[]"), nil
	}
	return json.RawMessage("{}"), nil
}

func (s *receiptTestSession) Notify(_ context.Context, method string, params any) error {
	if method != "textDocument/didOpen" {
		return nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var opened struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(encoded, &opened); err != nil {
		return err
	}
	s.uri, s.text = opened.TextDocument.URI, opened.TextDocument.Text
	return nil
}

func (s *receiptTestSession) WaitDiagnostics(_ context.Context, uri string, _ int) ([]neutralbackend.Diagnostic, error) {
	if uri != s.uri {
		return nil, fmt.Errorf("waited for %q, opened %q", uri, s.uri)
	}
	return []neutralbackend.Diagnostic{}, nil
}
func (s *receiptTestSession) Close() error { s.closed = true; return s.closeErr }

func TestRepeatedVerifyUsesFreshSessionAndSourceURI(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.kt")
	if err := pipeline.WriteAtomic(file, []byte("class Widget {}\n")); err != nil {
		t.Fatal(err)
	}
	var sessions []*receiptTestSession
	factory := func(_ context.Context, _ string, _ neutralbackend.KotlinConfig) (KotlinSession, error) {
		session := &receiptTestSession{}
		sessions = append(sessions, session)
		return session, nil
	}
	backend := NewKotlinBackend(WithKotlinSessionFactory(factory))
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	if _, err := backend.Verify(context.Background(), neutralbackend.VerifyRequest{Project: project}); err != nil {
		t.Fatal(err)
	}
	changed := []byte("class Widget { fun changed() {} }\n")
	if err := pipeline.WriteAtomic(file, changed); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Verify(context.Background(), neutralbackend.VerifyRequest{Project: project}); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("created %d sessions, want 2", len(sessions))
	}
	if sessions[0].uri == sessions[1].uri {
		t.Fatalf("reused diagnostics URI %q", sessions[0].uri)
	}
	if sessions[0].text == sessions[1].text || sessions[1].text != string(changed) {
		t.Fatalf("verify source snapshots were %q then %q", sessions[0].text, sessions[1].text)
	}
	if !sessions[0].closed || !sessions[1].closed {
		t.Fatal("verify left a request session open")
	}
}

func TestWaitDiagnosticsRejectsMissingAndNullArrays(t *testing.T) {
	for _, raw := range []string{`{"uri":"MATCH_URI"}`, `{"uri":"MATCH_URI","diagnostics":null}`} {
		t.Run(raw, func(t *testing.T) {
			root := t.TempDir()
			uri := "file://" + filepath.Join(root, "request-1.kt")
			params := []byte(strings.ReplaceAll(raw, "MATCH_URI", uri))
			session := &kotlinProcessSession{root: root, reports: make(map[string]kotlinDiagnosticReceipt), wake: make(chan struct{}, 1), selectedURI: uri}
			session.record(lsp.Notification{Method: "textDocument/publishDiagnostics", Params: params})
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			if _, err := session.WaitDiagnostics(ctx, uri, 1); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("WaitDiagnostics error = %v, want deadline", err)
			}
		})
	}
}

type stalledInitializeSession struct {
	closed bool
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func (s *stalledInitializeSession) Request(ctx context.Context, method string, _ any) (json.RawMessage, error) {
	if method == "initialize" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return json.RawMessage("{}"), nil
}
func (*stalledInitializeSession) Notify(context.Context, string, any) error { return nil }
func (*stalledInitializeSession) WaitDiagnostics(context.Context, string, int) ([]neutralbackend.Diagnostic, error) {
	return nil, errors.New("unexpected diagnostics wait")
}
func (s *stalledInitializeSession) Close() error { s.closed = true; return nil }

func TestLookupBoundsStalledInitializationAndCleansScratch(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.kt")
	if err := pipeline.WriteAtomic(file, []byte("class Widget {}\n")); err != nil {
		t.Fatal(err)
	}
	session := &stalledInitializeSession{}
	backend := NewKotlinBackend(WithKotlinSessionFactory(func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error) { return session, nil }))
	backend.operationTimeout = 30 * time.Millisecond
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	_, err := backend.Lookup(context.Background(), project, "Widget")
	if !errors.Is(err, ErrKotlinOperationTimeout) {
		t.Fatalf("Lookup error = %v, want operation timeout", err)
	}
	if !session.closed {
		t.Fatal("timed-out initialize left its session open")
	}
	entries, err := os.ReadDir(filepath.Join(root, ".scratch"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("timed-out initialize left scratch entries: %v", entries)
	}
}

func TestInitializeCleanupErrorsAreJoined(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.kt")
	if err := pipeline.WriteAtomic(file, []byte("class Widget {}\n")); err != nil {
		t.Fatal(err)
	}
	requestErr := errors.New("initialize request failed")
	closeErr := errors.New("session close failed")
	session := &receiptTestSession{requestErr: requestErr, closeErr: closeErr}
	backend := NewKotlinBackend(WithKotlinSessionFactory(func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error) { return session, nil }))
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	_, err := backend.Lookup(context.Background(), project, "Widget")
	if !errors.Is(err, requestErr) || !errors.Is(err, closeErr) {
		t.Fatalf("Lookup error = %v, want initialize and close errors", err)
	}
}

func (s *outlineTestSession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	if method == "textDocument/documentSymbol" {
		return json.RawMessage(strings.ReplaceAll(s.response, "SELECTED_URI", s.uri)), nil
	}
	return json.RawMessage("{}"), nil
}

func (s *outlineTestSession) Notify(_ context.Context, method string, params any) error {
	if method != "textDocument/didOpen" {
		return nil
	}
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var opened struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(data, &opened); err != nil {
		return err
	}
	s.uri, s.text = opened.TextDocument.URI, opened.TextDocument.Text
	return nil
}

func (*outlineTestSession) WaitDiagnostics(context.Context, string, int) ([]neutralbackend.Diagnostic, error) {
	return nil, errors.New("unexpected diagnostics wait")
}

func (s *outlineTestSession) Close() error {
	s.closed = true
	return nil
}

func TestOutlineMapsExactSnapshotAndCanonicalKinds(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.kt")
	source := []byte("package p // 😀\rclass Widget {\r  fun smile() {}\r}\r")
	if err := pipeline.WriteAtomic(file, source); err != nil {
		t.Fatal(err)
	}
	response := `[
		{"name":"p","kind":4,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":15}},"selectionRange":{"start":{"line":0,"character":8},"end":{"line":0,"character":9}},"uri":"SELECTED_URI"},
		{"name":"Widget","kind":5,"detail":"class detail","range":{"start":{"line":1,"character":0},"end":{"line":3,"character":1}},"selectionRange":{"start":{"line":1,"character":6},"end":{"line":1,"character":12}},"children":[{"name":"smile","kind":12,"detail":"fun smile()","range":{"start":{"line":2,"character":2},"end":{"line":2,"character":16}},"selectionRange":{"start":{"line":2,"character":6},"end":{"line":2,"character":11}}}]},
		{"name":"ACTIVE","kind":22,"range":{"start":{"line":2,"character":6},"end":{"line":2,"character":11}},"selectionRange":{"start":{"line":2,"character":6},"end":{"line":2,"character":11}}}
	]`
	var session *outlineTestSession
	var scratch string
	underTest := NewKotlinBackend(WithKotlinSessionFactory(func(_ context.Context, root string, _ neutralbackend.KotlinConfig) (KotlinSession, error) {
		scratch = root
		session = &outlineTestSession{response: response}
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			return nil, fmt.Errorf("scratch workspace was not empty before source copy: %v, %w", entries, err)
		}
		return session, nil
	}))
	request := neutralbackend.OutlineRequest{
		Project: neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)},
		Path:    file, IncludeUnexported: true,
	}
	result, err := underTest.Outline(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if session == nil || !session.closed || session.text != string(source) || session.uri == "" {
		t.Fatalf("session did not receive and close over exact snapshot: %#v", session)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".scratch"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("request scratch was not cleaned: %v, %v", entries, err)
	}
	if _, err := os.Stat(scratch); !os.IsNotExist(err) {
		t.Fatalf("isolated server workspace still exists: %q, %v", scratch, err)
	}
	if len(result.Files) != 1 || result.Files[0].File != "Widget.kt" || result.Files[0].Symbols == nil {
		t.Fatalf("outline file metadata = %#v", result.Files)
	}
	sum := sha256.Sum256(source)
	if result.Files[0].Revision != fmt.Sprintf("%x", sum) {
		t.Fatalf("revision = %q, want exact source hash %x", result.Files[0].Revision, sum)
	}
	symbols := result.Files[0].Symbols
	if len(symbols) != 3 || symbols[0].Kind != "package" || symbols[1].Kind != "class" || symbols[2].Kind != "enum_member" {
		t.Fatalf("canonical Kotlin symbols = %#v", symbols)
	}
	if symbols[1].ServerDetail == nil || *symbols[1].ServerDetail != "class detail" {
		t.Fatalf("server detail was not preserved: %#v", symbols[1].ServerDetail)
	}
	if len(symbols[1].Children) != 1 || symbols[1].Children[0].QualifiedName != "Widget.smile" || symbols[1].Children[0].Kind != "function" {
		t.Fatalf("hierarchical children = %#v", symbols[1].Children)
	}
	filtered, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: request.Project, Path: file, Kinds: []string{"trait"}, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Files[0].Symbols == nil || len(filtered.Files[0].Symbols) != 0 {
		t.Fatalf("known absent kind should return an empty symbol array, got %#v", filtered.Files[0].Symbols)
	}
	if !bytes.Equal(mustReadFile(t, file), source) {
		t.Fatal("outline changed the original file")
	}
}

func TestOutlinePreservesWhitespacePathAndRejectsUnsupportedScopeBeforeLaunch(t *testing.T) {
	root := t.TempDir()
	spaced := filepath.Join(root, " Thing.kt")
	plain := filepath.Join(root, "Thing.kt")
	spacedSource := []byte("class Spaced {}\n")
	plainSource := []byte("class Wrong {}\n")
	if err := pipeline.WriteAtomic(spaced, spacedSource); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(plain, plainSource); err != nil {
		t.Fatal(err)
	}
	calls := 0
	var sessions []*outlineTestSession
	underTest := NewKotlinBackend(WithKotlinSessionFactory(func(_ context.Context, _ string, _ neutralbackend.KotlinConfig) (KotlinSession, error) {
		calls++
		session := &outlineTestSession{response: "[]"}
		sessions = append(sessions, session)
		return session, nil
	}))
	project := neutralbackend.ProjectContext{RootDir: root, File: spaced, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	selectedPath := " Thing.kt"
	result, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: selectedPath, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || sessions[0].text != string(spacedSource) || result.Files[0].File != selectedPath {
		t.Fatalf("outline did not preserve selected filename and source: calls=%d text=%q file=%q", calls, sessions[0].text, result.Files[0].File)
	}
	sum := sha256.Sum256(spacedSource)
	if result.Files[0].Revision != fmt.Sprintf("%x", sum) {
		t.Fatalf("revision = %q, want spaced-file hash %x", result.Files[0].Revision, sum)
	}
	filtered, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: selectedPath, Kinds: []string{"trait"}, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || sessions[0] == sessions[1] || sessions[0].uri == sessions[1].uri || !sessions[0].closed || !sessions[1].closed {
		t.Fatalf("outline operations did not use fresh closed sessions and URIs: %#v", sessions)
	}
	if filtered.Files[0].Symbols == nil || len(filtered.Files[0].Symbols) != 0 {
		t.Fatalf("known absent kind should return an empty symbol array, got %#v", filtered.Files[0].Symbols)
	}
	before := calls
	if _, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: root, IncludeUnexported: true}); err == nil || !strings.Contains(err.Error(), "selected files only") {
		t.Fatalf("directory outline error = %v, want unsupported selected-file scope", err)
	}
	if calls != before {
		t.Fatalf("directory outline launched %d additional sessions", calls-before)
	}
	if _, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: selectedPath}); err == nil || !strings.Contains(err.Error(), "visibility filtering") {
		t.Fatalf("visibility=false error = %v, want explicit unsupported filter", err)
	}
	if calls != before {
		t.Fatalf("unsupported visibility request launched %d additional sessions", calls-before)
	}
	untrusted := project
	untrusted.WorkspaceTrust = neutralbackend.WorkspaceTrust{}
	if _, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: untrusted, Path: selectedPath, IncludeUnexported: true}); err == nil {
		t.Fatal("untrusted outline succeeded")
	}
	if calls != before {
		t.Fatalf("untrusted outline launched %d additional sessions", calls-before)
	}
	if !bytes.Equal(mustReadFile(t, spaced), spacedSource) || !bytes.Equal(mustReadFile(t, plain), plainSource) {
		t.Fatal("outline changed one of the original files")
	}
}

func TestOutlineRejectsForeignURIInFilteredChild(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.kt")
	if err := pipeline.WriteAtomic(file, []byte("class Widget { fun child() {} }\n")); err != nil {
		t.Fatal(err)
	}
	response := `[{"name":"Widget","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":31}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}},"children":[{"name":"child","kind":12,"range":{"start":{"line":0,"character":15},"end":{"line":0,"character":29}},"selectionRange":{"start":{"line":0,"character":19},"end":{"line":0,"character":24}},"uri":"file:///outside/Widget.kt"}]}]`
	session := &outlineTestSession{response: response}
	underTest := NewKotlinBackend(WithKotlinSessionFactory(func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error) { return session, nil }))
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	_, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: file, Kinds: []string{"variable"}, IncludeUnexported: true})
	if !errors.Is(err, ErrKotlinMalformedResponse) {
		t.Fatalf("filtered foreign child error = %v, want malformed response", err)
	}
	if !session.closed {
		t.Fatal("foreign child response did not clean up the session")
	}
}

func TestOutlineBoundsInitializationAndRejectsUnknownKinds(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.kt")
	if err := pipeline.WriteAtomic(file, []byte("class Widget {}\n")); err != nil {
		t.Fatal(err)
	}
	stalled := &stalledInitializeSession{}
	underTest := NewKotlinBackend(WithKotlinSessionFactory(func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error) { return stalled, nil }))
	underTest.operationTimeout = 30 * time.Millisecond
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true)}
	_, err := underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: file, IncludeUnexported: true})
	if !errors.Is(err, ErrKotlinOperationTimeout) || !stalled.closed {
		t.Fatalf("timeout error = %v, session closed=%v", err, stalled.closed)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".scratch"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("timeout left request scratch entries: %v, %v", entries, err)
	}
	unknown := `[{"name":"Widget","kind":99,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":15}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}}]`
	session := &outlineTestSession{response: unknown}
	underTest = NewKotlinBackend(WithKotlinSessionFactory(func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error) { return session, nil }))
	_, err = underTest.Outline(context.Background(), neutralbackend.OutlineRequest{Project: project, Path: file, IncludeUnexported: true})
	if !errors.Is(err, ErrKotlinUnsupportedResponse) {
		t.Fatalf("unknown kind error = %v, want unsupported response", err)
	}
}
