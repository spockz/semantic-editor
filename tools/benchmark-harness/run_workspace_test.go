// run_workspace_test.go checks each benchmark execution receives an exclusive workspace.

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBenchmarkWorkspaceIsExclusive(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing-parent", "fixtures")
	first, err := createBenchmarkWorkspace(root, "run-one")
	if err != nil {
		t.Fatal(err)
	}
	second, err := createBenchmarkWorkspace(root, "run-two")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("concurrent workspaces share a path: %s", first)
	}
	sentinel := filepath.Join(first, "owner")
	if err := os.WriteFile(sentinel, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createBenchmarkWorkspace(root, "run-one"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate run ID error = %v, want exclusive-create collision", err)
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "first" {
		t.Fatalf("existing workspace changed after collision: data=%q err=%v", data, err)
	}
}

func TestCreateBenchmarkWorkspaceRejectsUnsafeRunID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fixtures")
	for _, runID := range []string{"../outside", "nested/run", "."} {
		if workDir, err := createBenchmarkWorkspace(root, runID); err == nil || workDir != "" {
			t.Errorf("create workspace with run ID %q = (%q, %v), want local-name error", runID, workDir, err)
		}
	}
}
