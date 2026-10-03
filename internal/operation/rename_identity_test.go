// Package operation tests rename continuation identity against the real Go backend.
package operation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/backend"
	"semedit/internal/backends"
)

func TestRenameContinuationMatchesFreshInspection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/renameproof\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "server.go")
	source := "package sample\n\ntype Server struct{}\nfunc (Server) Start() {}\n\ntype Other struct{}\nfunc (Other) Serve() {}\n"
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageGo}
	service := backends.NewDefaultService()
	call := NewCallContext(root, project)
	call.Service = service
	result, err := renameRun(context.Background(), call, RenameReq{Project: project, Symbol: "Server.Start", To: "Serve"})
	if err != nil {
		t.Fatalf("rename method: %v", err)
	}
	if result.Lookup == nil || result.Revision == "" {
		t.Fatalf("rename did not prove a final identity: %#v", result)
	}
	projected, err := projectRenameOutput(result)
	if err != nil {
		t.Fatalf("project rename: %v", err)
	}
	output := projected.(map[string]any)
	if output["symbol"] != "Server.Serve" || output["file"] != "server.go" {
		t.Fatalf("continuation resolved the wrong method: %#v", output)
	}
	inspection, err := service.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "Server.Serve"})
	if err != nil {
		t.Fatalf("fresh inspect: %v", err)
	}
	if inspection == nil {
		t.Fatal("fresh inspect returned no result")
	}
	var freshRevision string
	for _, match := range inspection.Matches {
		if match.QualifiedName == "Server.Serve" && sameWorkspacePath(root, match.File, file) {
			if freshRevision != "" {
				t.Fatal("fresh inspect returned duplicate target matches")
			}
			freshRevision = match.Revision
		}
	}
	if freshRevision == "" || output[wireExpectedRevision] != freshRevision {
		t.Fatalf("rename revision %v does not match fresh inspection %q", output[wireExpectedRevision], freshRevision)
	}
}
