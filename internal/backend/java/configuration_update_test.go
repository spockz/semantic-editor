// WHY: Keeps Maven refresh notifications aligned with the JDT LS extension request shape.
package java

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

type configurationUpdateSession struct {
	notifications []struct {
		method string
		params any
	}
}

func (*configurationUpdateSession) Request(context.Context, string, any) (json.RawMessage, error) {
	return nil, nil
}

func (s *configurationUpdateSession) Notify(_ context.Context, method string, params any) error {
	s.notifications = append(s.notifications, struct {
		method string
		params any
	}{method: method, params: params})
	return nil
}

func (*configurationUpdateSession) WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error) {
	return nil, nil
}

func (*configurationUpdateSession) Close() error { return nil }

func TestNotifyProjectConfigurationUpdateUsesPOMDocumentIdentifiers(t *testing.T) {
	root := t.TempDir()
	root = backend.CanonicalWorkspaceRoot(root)
	session := &configurationUpdateSession{}
	underTest := &JavaBackend{root: root, session: session}
	changed := []string{filepath.Join(root, "pom.xml"), filepath.Join(root, "core", "pom.xml")}
	project := backend.ProjectContext{RootDir: root, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	if err := underTest.NotifyProjectConfigurationUpdate(context.Background(), project, changed); err != nil {
		t.Fatal(err)
	}
	if len(session.notifications) != 3 || session.notifications[0].method != "workspace/didChangeWatchedFiles" {
		t.Fatalf("notifications = %#v", session.notifications)
	}
	for index, path := range changed {
		got := session.notifications[index+1]
		if got.method != "java/projectConfigurationUpdate" {
			t.Fatalf("notification %d method = %q", index+1, got.method)
		}
		params, ok := got.params.(map[string]string)
		if !ok || params["uri"] != pathutil.FileURI(path) {
			t.Fatalf("notification %d params = %#v, want document URI %q", index+1, got.params, pathutil.FileURI(path))
		}
	}
}
