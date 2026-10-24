// WHY: Proves the Java backend accepts only the enabled Make Static action and applies its captured edit.
package java

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

type makeStaticTestSession struct {
	workspaceCodeActionSession

	response json.RawMessage
}

func (s *makeStaticTestSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if method == "textDocument/codeAction" {
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		var request struct {
			Context struct {
				Only []string `json:"only"`
			} `json:"context"`
		}
		if err := json.Unmarshal(encoded, &request); err != nil {
			return nil, err
		}
		if len(request.Context.Only) == 1 && request.Context.Only[0] == "source.organizeImports" {
			return json.RawMessage(`[]`), nil
		}
		return s.response, nil
	}
	return s.workspaceCodeActionSession.Request(ctx, method, params)
}

func TestSelectJavaMakeStaticActionRequiresOneExactEnabledAction(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  string
	}{
		{name: "exact", response: `[{"title":"Make Static","kind":"refactor.inline","edit":{}}]`},
		{name: "other title", response: `[{"title":"Make static","kind":"refactor.inline","edit":{}}]`, wantErr: `no enabled action titled "Make Static"`},
		{name: "wrong kind", response: `[{"title":"Make Static","kind":"quickfix","edit":{}}]`, wantErr: `no enabled action titled "Make Static"`},
		{name: "disabled", response: `[{"title":"Make Static","kind":"refactor.inline","disabled":{"reason":"not applicable"}}]`, wantErr: `only disabled actions titled "Make Static"`},
		{name: "ambiguous", response: `[{"title":"Make Static","kind":"refactor.inline","edit":{}},{"title":"Make Static","kind":"refactor.inline","edit":{}}]`, wantErr: `2 enabled actions titled "Make Static"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := selectJavaCodeActionTitle(json.RawMessage(test.response), javaMakeStaticActionKind, javaMakeStaticActionTitle)
			if test.wantErr == "" && err != nil {
				t.Fatalf("selectJavaCodeActionTitle() error = %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("selectJavaCodeActionTitle() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestApplyCodeActionMakeStaticCapturesAndPublishesWorkspaceEdit(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(root, "Service.java")
	source := []byte("package example;\nclass Service {\n  int count() { return 1; }\n}\n")
	if err := os.WriteFile(selected, source, 0o600); err != nil {
		t.Fatal(err)
	}
	uri := pathutil.FileURI(selected)
	session := &makeStaticTestSession{selectedURI: uri, response: json.RawMessage(`[{"title":"Make Static","kind":"refactor.inline","edit":{"changes":{"` + uri + `":[{"range":{"start":{"line":2,"character":2},"end":{"line":2,"character":2}},"newText":"static "}]}}}]`)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(backend.ProjectContext{RootDir: root, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}))
	if err != nil {
		t.Fatal(err)
	}
	service := &JavaBackend{root: root, fingerprint: fingerprint, session: session, documents: make(map[string]javaDocumentState)}
	digest := sha256.Sum256(source)
	project := backend.ProjectContext{RootDir: root, File: selected, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	result, err := service.ApplyCodeAction(context.Background(), backend.CodeActionRequest{
		Operation: javaMakeStaticOperation, Project: project, File: selected,
		Range: backend.Range{Start: backend.Position{Line: 2, Character: 2}, End: backend.Position{Line: 2, Character: 7}},
		Kind:  javaMakeStaticActionKind, ExpectedRevision: hex.EncodeToString(digest[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Files) != 1 || !strings.Contains(string(result.After), "static int count()") {
		t.Fatalf("make-static result = %#v, want one published static method edit", result)
	}
	got, err := os.ReadFile(selected)
	if err != nil || !strings.Contains(string(got), "static int count()") {
		t.Fatalf("published source = %q, err = %v", got, err)
	}
}
