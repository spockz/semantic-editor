// Package golang tests cache selection at the subprocess boundary so rename tooling remains usable in restricted workspaces.
package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/gocache"
)

func TestCommandEnvPreservesUsableCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOCACHE", dir)
	env, err := gocache.Environment(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !containsEnv(env, "GOCACHE="+dir) {
		t.Fatalf("environment did not preserve cache: %v", env)
	}
}

func TestCommandEnvFallsBackToWorkspaceScratch(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(bad, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOCACHE", bad)
	env, err := gocache.Environment(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".scratch", "go-cache")
	if !containsEnv(env, "GOCACHE="+want) {
		t.Fatalf("environment cache = %v, want %q", env, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("fallback cache was not created: %v", err)
	}
}

func containsEnv(env []string, want string) bool {
	for _, value := range env {
		if strings.HasPrefix(value, "GOCACHE=") {
			return value == want
		}
	}
	return false
}
