// Package operation verifies typed organize-imports outcomes at the operation boundary.
package operation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestJavaOrganizeImportsFailureReportsAlreadyPublishedSource(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.java")
	if err := os.WriteFile(file, []byte("class Widget {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := []byte("class Widget {}\n")
	if err := os.WriteFile(file, []byte("import java.util.List;\nclass Widget {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := javaOrganizeImportsFailure(context.Background(), NewCallContext(root, backend.ProjectContext{}), file, before, context.DeadlineExceeded)
	var failure *organizeImportsError
	if !errors.As(err, &failure) {
		t.Fatalf("failure type = %T, want organizeImportsError", err)
	}
	if failure.ApplicationState != string(EditPartiallyPublished) || !reflect.DeepEqual(failure.ChangedFiles, []string{"Widget.java"}) {
		t.Fatalf("partial Java result = %#v", failure)
	}
}

func TestAddBuildDependencyParsesJavaCoordinateAndTrust(t *testing.T) {
	req, err := parseAddBuildDependency(map[string]any{"language": "java", "trust_workspace": true, "dependency": "org.example:codec:2.1:runtime"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Project.Language != backend.LanguageJava || !req.Project.WorkspaceTrust.Trusted || req.Dependency != "org.example:codec:2.1:runtime" {
		t.Fatalf("parsed Java dependency request = %#v", req)
	}
}

func TestAddJavaBuildDependencyUpdatesStandalonePOM(t *testing.T) {
	root := t.TempDir()
	pom := "<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><modelVersion>4.0.0</modelVersion><groupId>org.example</groupId><artifactId>app</artifactId><version>1</version></project>\n"
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(pom), 0o600); err != nil {
		t.Fatal(err)
	}
	project := backend.ProjectContext{RootDir: root, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	call := NewCallContext(root, project)
	result, err := runAddJavaBuildDependency(context.Background(), call, AddBuildDependencyReq{Project: project, Dependency: "org.example:codec:2.1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.POMs) != 1 || result.Dependency != "org.example:codec:2.1" {
		t.Fatalf("result = %#v", result)
	}
	updated, err := os.ReadFile(filepath.Join(root, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "<artifactId>codec</artifactId>") || !strings.Contains(string(updated), "<version>2.1</version>") {
		t.Fatalf("updated POM = %s", updated)
	}
}
