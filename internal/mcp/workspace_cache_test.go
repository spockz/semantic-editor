// Package mcp tests cache selection, which depends on Server configuration rather than workspace path policy.
package mcp

import (
	"path/filepath"
	"testing"
)

func TestGoBaseDirForRootRequiresExplicitIsolation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ambientServer := &Server{workDir: root}
	for _, selected := range []string{root, filepath.Join(root, "module", "nested"), filepath.Join(root, ".scratch", "worktrees", "other")} {
		if got := ambientServer.goBaseDirForRoot(selected); got != "" {
			t.Errorf("ambient goBaseDirForRoot(%q) = %q, want empty", selected, got)
		}
	}

	configuredCache := filepath.Join(root, "private-cache")
	server := &Server{workDir: root, goBaseDir: configuredCache}
	for _, selected := range []string{root, filepath.Join(root, "module", "nested")} {
		if got := server.goBaseDirForRoot(selected); got != configuredCache {
			t.Errorf("goBaseDirForRoot(%q) = %q, want configured cache %q", selected, got, configuredCache)
		}
	}
	worktree := filepath.Join(root, ".scratch", "worktrees", "other")
	if got, want := server.goBaseDirForRoot(worktree), filepath.Join(worktree, ".scratch", "go"); got != want {
		t.Errorf("goBaseDirForRoot(worktree) = %q, want isolated cache %q", got, want)
	}
}
