// Package mcp tests cache selection, which depends on Server configuration rather than workspace path policy.
package mcp

import (
	"path/filepath"
	"testing"
)

func TestGoBaseDirForRootSharesDefaultSubdirectoriesAndIsolatesWorktrees(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
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
