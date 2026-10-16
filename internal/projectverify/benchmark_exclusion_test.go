// Verification must stage source files without copying benchmark archives into each normalization workspace.
package projectverify_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/atomicfile"
	"github.com/spockz/semantic-editor/internal/projectverify"
)

func TestNormalizationExcludesBenchmarkData(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	archive := filepath.Join(root, "data", "benchmarks", "archive", "Archived.java")
	if err := os.MkdirAll(filepath.Dir(archive), 0o750); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(root, "main.go"): "package main\n",
		archive:                        "class Archived {}\n",
	} {
		if err := atomicfile.WriteAtomic(path, []byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := projectverify.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Sources) != 1 || plan.Sources[0].Path != "main.go" {
		t.Fatalf("verification sources include benchmark artifacts: %+v", plan.Sources)
	}
	plan.Normalize = []projectverify.Hook{{
		ID: "format", Languages: []string{"go"},
		Shell: &projectverify.ShellSpec{Executable: "/bin/sh", Args: []string{"-c"},
			Script: "test ! -e data/benchmarks && printf 'package main\\n// formatted\\n' > main.go"},
	}}
	report, err := projectverify.RunPhase(t.Context(), plan, projectverify.PhaseNormalize, projectverify.Scope{Paths: []string{"main.go"}}, nil)
	if err != nil || !report.Completed {
		t.Fatalf("normalize without staging archive: %+v, %v", report, err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil || string(contents) != "package main\n// formatted\n" {
		t.Fatalf("normalized source = %q, %v", contents, err)
	}
}
