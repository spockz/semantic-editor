// WHY: Exercises the public Java code-action lifecycle so overlay synchronization, versioned diagnostics, and save publication do not regress.
package java

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

type lifecycleJavaSession struct {
	file      string
	uri       string
	original  []byte
	action    []byte
	formatted []byte
	final     []byte

	overlay       []byte
	lastVersion   int
	didOpen       int
	didChange     int
	didSave       int
	formatSeen    bool
	organizeSeen  bool
	diagnosticsAt int
}

func (s *lifecycleJavaSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	switch method {
	case "initialize":
		return json.RawMessage(`{}`), nil
	case "textDocument/formatting":
		if err := s.expectOverlayAndDisk("formatting", s.action, s.original); err != nil {
			return nil, err
		}
		s.formatSeen = true
		return json.Marshal([]any{map[string]any{
			"range":   javaRange{Start: javaPosition{}, End: javaPosition{Line: 4, Character: 0}},
			"newText": string(s.formatted),
		}})
	case "textDocument/codeAction":
		kind, version, err := lifecycleCodeActionRequest(params)
		if err != nil {
			return nil, err
		}
		switch kind {
		case "refactor.inline":
			if version != 1 {
				return nil, fmt.Errorf("initial code action version = %d, want 1", version)
			}
			return lifecycleCodeActionResponse(s.uri, version, kind, javaRange{
				Start: javaPosition{Line: 2, Character: 21}, End: javaPosition{Line: 2, Character: 22},
			}, "2")
		case "source.organizeImports":
			if err := s.expectOverlayAndDisk("organize imports", s.formatted, s.original); err != nil {
				return nil, err
			}
			if version != 3 {
				return nil, fmt.Errorf("organize imports version = %d, want 3", version)
			}
			s.organizeSeen = true
			return lifecycleCodeActionResponse(s.uri, version, kind, javaRange{
				Start: javaPosition{Line: 0, Character: 17}, End: javaPosition{Line: 0, Character: 26},
			}, "List")
		default:
			return nil, fmt.Errorf("unexpected code action kind %q", kind)
		}
	default:
		return nil, fmt.Errorf("unexpected request %q", method)
	}
}

func (s *lifecycleJavaSession) Notify(_ context.Context, method string, params any) error {
	switch method {
	case "textDocument/didOpen":
		version, content, err := lifecycleDocumentUpdate(params, true)
		if err != nil {
			return err
		}
		if version != 1 || !bytes.Equal(content, s.original) {
			return fmt.Errorf("didOpen version=%d content=%q, want original version 1", version, content)
		}
		s.overlay, s.lastVersion, s.didOpen = content, version, s.didOpen+1
	case "textDocument/didChange":
		version, content, err := lifecycleDocumentUpdate(params, false)
		if err != nil {
			return err
		}
		if version != s.lastVersion+1 {
			return fmt.Errorf("didChange version = %d, want %d", version, s.lastVersion+1)
		}
		if err := s.expectDisk("didChange", s.original); err != nil {
			return err
		}
		s.overlay, s.lastVersion, s.didChange = content, version, s.didChange+1
	case "textDocument/didSave":
		if err := s.expectOverlayAndDisk("didSave", s.final, s.final); err != nil {
			return err
		}
		s.didSave++
	}
	return nil
}

func (s *lifecycleJavaSession) WaitDiagnostics(_ context.Context, uri string, version int) ([]backend.Diagnostic, error) {
	if uri != s.uri {
		return nil, fmt.Errorf("diagnostics URI = %q, want %q", uri, s.uri)
	}
	if version != s.lastVersion || version != 4 {
		return nil, fmt.Errorf("diagnostics version = %d, current version = %d, want 4", version, s.lastVersion)
	}
	if err := s.expectOverlayAndDisk("diagnostics", s.final, s.original); err != nil {
		return nil, err
	}
	s.diagnosticsAt = version
	return nil, nil
}

func (*lifecycleJavaSession) Close() error { return nil }

func (s *lifecycleJavaSession) expectOverlayAndDisk(stage string, overlay, disk []byte) error {
	if !bytes.Equal(s.overlay, overlay) {
		return fmt.Errorf("%s overlay = %q, want %q", stage, s.overlay, overlay)
	}
	return s.expectDisk(stage, disk)
}

func (s *lifecycleJavaSession) expectDisk(stage string, want []byte) error {
	got, err := os.ReadFile(s.file)
	if err != nil {
		return fmt.Errorf("read disk during %s: %w", stage, err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("disk during %s = %q, want %q", stage, got, want)
	}
	return nil
}

func lifecycleCodeActionRequest(params any) (string, int, error) {
	encoded, err := json.Marshal(params)
	if err != nil {
		return "", 0, fmt.Errorf("encode code action request: %w", err)
	}
	var request struct {
		TextDocument struct {
			Version int `json:"version"`
		} `json:"textDocument"`
		Context struct {
			Only []string `json:"only"`
		} `json:"context"`
	}
	if err := json.Unmarshal(encoded, &request); err != nil {
		return "", 0, fmt.Errorf("decode code action request: %w", err)
	}
	if len(request.Context.Only) != 1 {
		return "", 0, fmt.Errorf("code action kinds = %v, want one kind", request.Context.Only)
	}
	return request.Context.Only[0], request.TextDocument.Version, nil
}

func lifecycleCodeActionResponse(uri string, version int, kind string, editRange javaRange, newText string) (json.RawMessage, error) {
	return json.Marshal([]any{map[string]any{
		"title": "lifecycle edit",
		"kind":  kind,
		"edit": map[string]any{"documentChanges": []any{map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": version},
			"edits":        []any{map[string]any{"range": editRange, "newText": newText}},
		}}},
	}})
}

func lifecycleDocumentUpdate(params any, open bool) (int, []byte, error) {
	encoded, err := json.Marshal(params)
	if err != nil {
		return 0, nil, fmt.Errorf("encode document notification: %w", err)
	}
	var notification struct {
		TextDocument struct {
			Version int    `json:"version"`
			Text    string `json:"text"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(encoded, &notification); err != nil {
		return 0, nil, fmt.Errorf("decode document notification: %w", err)
	}
	if open {
		return notification.TextDocument.Version, []byte(notification.TextDocument.Text), nil
	}
	if len(notification.ContentChanges) != 1 {
		return 0, nil, fmt.Errorf("didChange content changes = %d, want 1", len(notification.ContentChanges))
	}
	return notification.TextDocument.Version, []byte(notification.ContentChanges[0].Text), nil
}

func TestApplyCodeActionKeepsJavaCSTEditInOverlayUntilFormattedSave(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	file := filepath.Join(root, "Thing.java")
	original := []byte("import java.util.List;\nclass Thing {\nint value() { return 1; }\n}\n")
	action := []byte("import java.util.List;\nclass Thing {\nint value() { return 2; }\n}\n")
	formatted := []byte("import java.util.ArrayList;\nclass Thing {\n    int value() {\n        return 2;\n    }\n}\n")
	final := []byte("import java.util.List;\nclass Thing {\n    int value() {\n        return 2;\n    }\n}\n")
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, original, 0o600); err != nil {
		t.Fatal(err)
	}
	session := &lifecycleJavaSession{file: file, uri: pathutil.FileURI(file), original: original, action: action, formatted: formatted, final: final}
	service := NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (JavaSession, error) {
		return session, nil
	})
	digest := sha256.Sum256(original)
	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	result, err := service.ApplyCodeAction(context.Background(), backend.CodeActionRequest{
		Operation: backend.OperationInlineVariable,
		Project:   project, File: file, Kind: "refactor.inline",
		Range:            backend.Range{Start: backend.Position{Line: 2, Character: 21}, End: backend.Position{Line: 2, Character: 22}},
		ExpectedRevision: hex.EncodeToString(digest[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || !bytes.Equal(result.After, final) {
		t.Fatalf("result = %#v, want final normalized source", result)
	}
	if got, err := os.ReadFile(file); err != nil || !bytes.Equal(got, final) {
		t.Fatalf("published source = %q, err=%v, want %q", got, err, final)
	}
	if session.didOpen != 1 || session.didChange != 3 || session.didSave != 1 {
		t.Fatalf("document lifecycle didOpen=%d didChange=%d didSave=%d, want 1, 3, 1", session.didOpen, session.didChange, session.didSave)
	}
	if !session.formatSeen || !session.organizeSeen || session.diagnosticsAt != 4 {
		t.Fatalf("format=%v organize=%v diagnosticsVersion=%d, want true, true, 4", session.formatSeen, session.organizeSeen, session.diagnosticsAt)
	}
}
