// Package golang tests cache selection at the subprocess boundary so rename tooling remains usable in restricted workspaces.
package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/gocache"
)

func TestCommandEnvUsesWorkspaceLocalGoState(t *testing.T) {
	root := t.TempDir()
	for key, value := range map[string]string{
		"GOENV":      filepath.Join(t.TempDir(), "env"),
		"GOCACHE":    t.TempDir(),
		"GOMODCACHE": t.TempDir(),
		"GOTMPDIR":   t.TempDir(),
		"GOBIN":      t.TempDir(),
		"GOPATH":     t.TempDir(),
		"GOFLAGS":    "-mod=readonly",
		"GOWORK":     filepath.Join(t.TempDir(), "go.work"),
	} {
		t.Setenv(key, value)
	}
	env, err := gocache.Environment(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"GOENV":      filepath.Join(root, ".scratch", "go", "env"),
		"GOCACHE":    filepath.Join(root, ".scratch", "go", "build"),
		"GOMODCACHE": filepath.Join(root, ".scratch", "go", "mod"),
		"GOTMPDIR":   filepath.Join(root, ".scratch", "go", "tmp"),
		"GOBIN":      filepath.Join(root, ".scratch", "go", "bin"),
		"GOPATH":     filepath.Join(root, ".scratch", "go"),
		"GOFLAGS":    "",
		"GOWORK":     "off",
	}
	for key, path := range want {
		if got := envValue(env, key); got != path {
			t.Errorf("%s = %q, want %q", key, got, path)
		}
	}
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOTMPDIR", "GOBIN", "GOPATH"} {
		if _, err := os.Stat(want[key]); err != nil {
			t.Errorf("%s directory was not created: %v", key, err)
		}
	}
}

func TestCommandEnvRequiresWorkspace(t *testing.T) {
	if _, err := gocache.Environment(t.Context(), ""); err == nil {
		t.Fatal("Environment(\"\") succeeded, want error")
	}
}

func TestCommandEnvUsesConfiguredBaseDir(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(t.TempDir(), "go-state")
	env, err := gocache.Environment(gocache.WithBaseDir(t.Context(), baseDir), root)
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "GOCACHE"); got != filepath.Join(baseDir, "build") {
		t.Errorf("GOCACHE = %q, want %q", got, filepath.Join(baseDir, "build"))
	}
}

func envValue(env []string, key string) string {
	for _, value := range env {
		candidate, value, found := strings.Cut(value, "=")
		if found && candidate == key {
			return value
		}
	}
	return ""
}
