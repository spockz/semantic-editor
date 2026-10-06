// Package mcp tests request-root resolution and state binding at MCP ingress.
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestRootAndTrustRejectWrongTypes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, raw := range []map[string]any{
		{"workspace_root": nil},
		{"workspace_root": 7},
		{"trust_workspace": nil},
		{"trust_workspace": "true"},
	} {
		if _, err := prepareWorkspaceArguments(root, raw); err == nil {
			t.Errorf("prepareWorkspaceArguments(%#v) succeeded", raw)
		}
	}
	if _, err := resolveRequestRoot(root, t.TempDir()); err == nil || !strings.Contains(err.Error(), "outside configured root") {
		t.Errorf("resolveRequestRoot accepted an external workspace: %v", err)
	}
	if got, err := resolveRequestRoot(root, filepath.Join(root, ".scratch", "worktrees", "selected")); err == nil || !strings.Contains(err.Error(), "canonicalize workspace root") {
		t.Errorf("resolveRequestRoot accepted a missing in-root worktree: root=%q err=%v", got, err)
	}
}

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

func TestRequestPathsRejectTraversalAndSymlinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside.go", filepath.Join(root, "linked", "outside.go")} {
		if err := requestPathWithinRoot(root, path); err == nil || !strings.Contains(err.Error(), "outside workspace root") && !strings.Contains(err.Error(), "symlink") {
			t.Errorf("requestPathWithinRoot(%q) error = %v", path, err)
		}
	}
	if _, err := resolveRequestRoot(root, t.TempDir()); err == nil || !strings.Contains(err.Error(), "outside configured root") {
		t.Errorf("resolveRequestRoot accepted an external workspace: %v", err)
	}
	if got, err := resolveRequestRoot(root, filepath.Join(root, ".scratch", "worktrees", "selected")); err == nil || !strings.Contains(err.Error(), "canonicalize workspace root") {
		t.Errorf("resolveRequestRoot accepted a missing in-root worktree: root=%q err=%v", got, err)
	}
	for _, test := range []struct {
		name string
		raw  map[string]any
	}{
		{name: "call site", raw: map[string]any{"call_site": map[string]any{"file": "../outside.go"}}},
		{name: "contract file", raw: map[string]any{"contract_file": "../outside.go"}},
		{name: "member file", raw: map[string]any{"members": []any{map[string]any{"file": "../outside.go"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateRequestPaths(root, test.raw); err == nil {
				t.Fatalf("validateRequestPaths(%#v) succeeded", test.raw)
			}
		})
	}
}
