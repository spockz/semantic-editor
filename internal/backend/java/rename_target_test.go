// WHY: Verifies that native Java rename eligibility uses JDT LS's complete safe-edit preparation without publishing source.
package java

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type renameTargetSession struct {
	uri            string
	extraURI       string
	ambiguity      bool
	typeRename     bool
	renameRequests int
}

func (s *renameTargetSession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	switch method {
	case "textDocument/documentSymbol":
		if s.typeRename {
			return json.Marshal([]any{map[string]any{
				"name": "Service", "kind": 5,
				"range":          javaRange{Start: javaPosition{}, End: javaPosition{Line: 0, Character: 21}},
				"selectionRange": javaRange{Start: javaPosition{Line: 0, Character: 13}, End: javaPosition{Line: 0, Character: 20}},
			}})
		}
		children := []any{map[string]any{
			"name": "send()", "kind": 6,
			"range":          javaRange{Start: javaPosition{Line: 0, Character: 16}, End: javaPosition{Line: 0, Character: 30}},
			"selectionRange": javaRange{Start: javaPosition{Line: 0, Character: 21}, End: javaPosition{Line: 0, Character: 25}},
		}}
		if s.ambiguity {
			children = append(children, map[string]any{
				"name": "send(int)", "kind": 6,
				"range":          javaRange{Start: javaPosition{Line: 0, Character: 16}, End: javaPosition{Line: 0, Character: 30}},
				"selectionRange": javaRange{Start: javaPosition{Line: 0, Character: 21}, End: javaPosition{Line: 0, Character: 25}},
			})
		}
		return json.Marshal([]any{map[string]any{
			"name": "Service", "kind": 5,
			"range":          javaRange{Start: javaPosition{}, End: javaPosition{Line: 0, Character: 32}},
			"selectionRange": javaRange{Start: javaPosition{}, End: javaPosition{Line: 0, Character: 13}},
			"children":       children,
		}})
	case "textDocument/prepareRename":
		if s.typeRename {
			return json.Marshal(javaRange{Start: javaPosition{Line: 0, Character: 13}, End: javaPosition{Line: 0, Character: 20}})
		}
		return json.Marshal(javaRange{Start: javaPosition{Line: 0, Character: 21}, End: javaPosition{Line: 0, Character: 25}})
	case "textDocument/rename":
		s.renameRequests++
		if s.typeRename {
			return json.Marshal(map[string]any{"changes": map[string]any{s.uri: []any{map[string]any{
				"range": javaRange{Start: javaPosition{Line: 0, Character: 13}, End: javaPosition{Line: 0, Character: 20}}, "newText": "Widget",
			}}}})
		}
		if s.extraURI != "" {
			return json.Marshal(map[string]any{"documentChanges": []any{
				map[string]any{"textDocument": map[string]any{"uri": s.uri, "version": 1}, "edits": []any{map[string]any{
					"range": javaRange{Start: javaPosition{Line: 0, Character: 21}, End: javaPosition{Line: 0, Character: 25}}, "newText": "publish",
				}}},
				map[string]any{"textDocument": map[string]any{"uri": s.extraURI, "version": 2}, "edits": []any{map[string]any{
					"range": javaRange{Start: javaPosition{Line: 0, Character: 15}, End: javaPosition{Line: 0, Character: 22}}, "newText": "Widget",
				}}},
			}})
		}
		return json.Marshal(map[string]any{"changes": map[string]any{s.uri: []any{map[string]any{
			"range": javaRange{Start: javaPosition{Line: 0, Character: 21}, End: javaPosition{Line: 0, Character: 25}}, "newText": "publish",
		}}}})
	default:
		return nil, errors.New("unexpected Java session request: " + method)
	}
}

func (*renameTargetSession) Notify(context.Context, string, any) error { return nil }
func (*renameTargetSession) WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error) {
	return nil, nil
}
func (*renameTargetSession) Close() error { return nil }

func TestValidateRenameTargetDoesNotPublishAndCanRetry(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	file := filepath.Join(root, "Service.java")
	source := []byte("class Service { void send() {} }\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatal(err)
	}
	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(project))
	if err != nil {
		t.Fatal(err)
	}
	session := &renameTargetSession{uri: pathutil.FileURI(file)}
	underTest := &JavaBackend{root: root, fingerprint: fingerprint, session: session, documents: make(map[string]javaDocumentState)}
	request := backend.RenameRequest{Project: project, Symbol: "Service.send()", To: "publish"}

	for attempt := range 2 {
		if err := underTest.ValidateRenameTarget(context.Background(), request); err != nil {
			t.Fatalf("validation attempt %d: %v", attempt+1, err)
		}
		published, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(published, source) {
			t.Fatalf("validation attempt %d published source = %q, want %q", attempt+1, published, source)
		}
	}
	if session.renameRequests != 2 {
		t.Fatalf("rename requests = %d, want 2 safe workspace-edit validations", session.renameRequests)
	}
}

func TestValidateRenameTargetReportsTypedAmbiguity(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	file := filepath.Join(root, "Service.java")
	source := []byte("class Service { void send() {} }\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatal(err)
	}
	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(project))
	if err != nil {
		t.Fatal(err)
	}
	underTest := &JavaBackend{root: root, fingerprint: fingerprint, session: &renameTargetSession{uri: pathutil.FileURI(file), ambiguity: true}, documents: make(map[string]javaDocumentState)}
	_, err = underTest.FindReferences(context.Background(), backend.ReferencesRequest{Project: project, Symbol: "Service.send"})
	if !errors.Is(err, ErrJavaSymbolAmbiguous) {
		t.Fatalf("reference error = %v, want ErrJavaSymbolAmbiguous", err)
	}
	err = underTest.ValidateRenameTarget(context.Background(), backend.RenameRequest{Project: project, Symbol: "Service.send", To: "publish"})
	if !errors.Is(err, ErrJavaSymbolAmbiguous) {
		t.Fatalf("validation error = %v, want ErrJavaSymbolAmbiguous", err)
	}
	if published, readErr := os.ReadFile(file); readErr != nil || !bytes.Equal(published, source) {
		t.Fatalf("ambiguity validation changed source = %q, read error = %v", published, readErr)
	}
}

func TestValidateRenameTargetRejectsStaleWorkspaceDocumentVersion(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	file := filepath.Join(root, "Service.java")
	caller := filepath.Join(root, "Caller.java")
	source := []byte("class Service { void send() {} }\n")
	callerSource := []byte("class Caller { Service value; }\n")
	for path, contents := range map[string][]byte{file: source, caller: callerSource} {
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(project))
	if err != nil {
		t.Fatal(err)
	}
	underTest := &JavaBackend{root: root, fingerprint: fingerprint, session: &renameTargetSession{uri: pathutil.FileURI(file), extraURI: pathutil.FileURI(caller)}, documents: make(map[string]javaDocumentState)}
	err = underTest.ValidateRenameTarget(context.Background(), backend.RenameRequest{Project: project, Symbol: "Service.send()", To: "publish"})
	if !errors.Is(err, ErrJavaRenameStaleEdit) {
		t.Fatalf("validation error = %v, want ErrJavaRenameStaleEdit", err)
	}
	for path, want := range map[string][]byte{file: source, caller: callerSource} {
		got, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(got, want) {
			t.Fatalf("stale validation published %s = %q, read error = %v", path, got, readErr)
		}
	}
}

func TestValidateRenameTargetRejectsBufferedResourceRename(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	file := filepath.Join(root, "Service.java")
	source := []byte("public class Service {}\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatal(err)
	}
	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(project))
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	underTest := &JavaBackend{root: root, fingerprint: fingerprint, session: &renameTargetSession{uri: pathutil.FileURI(file), typeRename: true}, documents: make(map[string]javaDocumentState)}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	err = underTest.ValidateRenameTarget(ctx, backend.RenameRequest{Project: project, Symbol: "Service", To: "Widget"})
	if !errors.Is(err, ErrJavaBatchResourceRenameUnsupported) {
		t.Fatalf("validation error = %v, want ErrJavaBatchResourceRenameUnsupported", err)
	}
	if buffer.DirtySourceBytes() != 0 {
		t.Fatalf("resource rename validation staged %d bytes", buffer.DirtySourceBytes())
	}
	if got, readErr := os.ReadFile(file); readErr != nil || !bytes.Equal(got, source) {
		t.Fatalf("resource rename validation published source = %q, read error = %v", got, readErr)
	}
}
