// Harness snapshots and oracle copies must not traverse captured benchmark workspaces.
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/atomicfile"
)

func TestBenchmarkWorkspaceExcludesCapturedArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, relative := range []string{"main.go", "src/benchmarks/value.go", "data/benchmarks/archive/value.go"} {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := atomicfile.WriteAtomic(path, []byte("package example\n")); err != nil {
			t.Fatal(err)
		}
	}
	archive := filepath.Join(root, "data", "benchmarks", "archive")
	if err := os.Chmod(archive, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(archive, 0o750); err != nil {
			t.Error(err)
		}
	})
	files, snapshotErr := snapshotWorkspaceFiles(root)
	if snapshotErr != nil || len(files) != 2 {
		t.Fatalf("snapshot traversed artifact data: %v, %v", files, snapshotErr)
	}
	target, cleanup, copyErr := copyWorkspaceForOracle(root)
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Error(err)
		}
	})
	if err := installHiddenTests(target, root); err != nil {
		t.Fatalf("hidden-test installation entered benchmark data: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "data", "benchmarks")); !os.IsNotExist(err) {
		t.Fatalf("oracle copied benchmark artifacts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "src", "benchmarks", "value.go")); err != nil {
		t.Fatalf("ordinary benchmark source was excluded: %v", err)
	}
}
