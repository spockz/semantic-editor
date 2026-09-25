// Package kotlin tests the observable trust and receipt boundaries of the Kotlin adapter.
package kotlin

import (
	"context"
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
