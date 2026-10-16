// Report discovery must never descend into captured workspaces or publish archived evidence as results.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBenchmarkDiscoveryPrunesArtifactTrees(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	files := map[string]string{
		"report.json":                           "{}",
		"run/task/result.json":                  "{}",
		"manifest.json":                         "{",
		"run/smoke-summary.json":                "{",
		"archive/old/report.json":               "{",
		"fixture-validation/task/report.json":   "{",
		"run/evidence/report.json":              "{",
		"run/jobs/checkpoint.json":              "{",
		"run/workspaces/report.json":            "{",
		"run/.scratch/report.json":              "{",
		"run/task/captured-project/report.json": "{",
	}
	for relative, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := writeGeneratedFile(path, []byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	// A read of this tree would fail before JSON parsing, proving directory pruning.
	blocked := filepath.Join(root, "archive")
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(blocked, 0o750); err != nil {
			t.Error(err)
		}
	})
	sources, err := loadBenchmarkSources(root)
	if err != nil {
		t.Fatalf("load published reports without reading artifacts: %v", err)
	}
	if len(sources) != 2 || sources[0].RunID != "legacy" || sources[1].RunID != "run" {
		t.Fatalf("published sources = %+v, want only root report and run/task report", sources)
	}
}
