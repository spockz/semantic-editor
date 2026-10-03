// Package operation verifies typed organize-imports outcomes at the operation boundary.
package operation

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"semedit/internal/backend"
)

func TestOrganizeImportsErrorProjectionRetainsPublishedPathsAndDiagnosticStatus(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/importoutcome\n\ngo 1.23\n",
		"a.go":   "package p\n\nfunc First() { fmt.Println(\"first\") }\n",
		"b.go":   "package p\n\nfunc Broken( {\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	call := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageGo})
	_, err := runOrganizeImports(context.Background(), call, OrganizeImportsReq{Project: call.Project, File: "."})
	if err == nil {
		t.Fatal("organize imports succeeded despite a later syntax error")
	}
	entry, ok := DefaultRegistry().LookupMCP("semantic_organize_imports")
	if !ok {
		t.Fatal("organize imports operation missing")
	}
	projected, known, projectionErr := entry.ProjectErrorOutput(err)
	if projectionErr != nil || !known {
		t.Fatalf("project partial organize-imports outcome: known=%v err=%v", known, projectionErr)
	}
	output := projected.(map[string]any)
	if output[organizeImportsApplicationState] != string(EditPartiallyPublished) || output[organizeImportsDiagnosticStatus] != "unavailable" {
		t.Fatalf("partial organize-imports state = %#v", output)
	}
	if got := output[organizeImportsChangedFiles]; !reflect.DeepEqual(got, []any{"a.go"}) {
		t.Fatalf("changed_files = %#v, want only the successfully published file", got)
	}
}

func TestOrganizeImportsOutputRejectsMissingApplicationState(t *testing.T) {
	entry, ok := DefaultRegistry().LookupMCP("semantic_organize_imports")
	if !ok {
		t.Fatal("organize imports operation missing")
	}
	if _, err := entry.ProjectOutput(FileEditRes{}); err == nil {
		t.Fatal("organize imports projected a missing state as a no-op")
	}
}
