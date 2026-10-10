package astedit_test

import (
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/astedit"
)

func TestWorkspaceOutlineGo(t *testing.T) {
	tempDir := t.TempDir()

	// Root module and package
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/testroot\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "root.go"), []byte("// Package root is the root package.\npackage root\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Level 1 package with doc.go
	pkg1 := filepath.Join(tempDir, "pkg1")
	if err := os.MkdirAll(pkg1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg1, "doc.go"), []byte("// Package pkg1 provides level 1 utilities.\npackage pkg1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg1, "foo.go"), []byte("package pkg1\nfunc Foo() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Level 2 package without doc.go
	pkg2 := filepath.Join(tempDir, "pkg1", "sub")
	if err := os.MkdirAll(pkg2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg2, "sub.go"), []byte("// Package sub provides nested utilities.\npackage sub\nfunc Sub() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Submodule in tools/harness
	submod := filepath.Join(tempDir, "tools", "harness")
	if err := os.MkdirAll(submod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(submod, "go.mod"), []byte("module example.com/testroot/tools/harness\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(submod, "harness.go"), []byte("// Package harness is the harness tool.\npackage harness\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := astedit.WorkspaceOutlineGo(tempDir, "", false)
	if err != nil {
		t.Fatalf("WorkspaceOutlineGo failed: %v", err)
	}

	if len(res.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(res.Modules))
	}
	if !res.Modules[0].IsRoot || res.Modules[0].ModulePath != "example.com/testroot" {
		t.Errorf("unexpected root module: %+v", res.Modules[0])
	}
	if res.Modules[1].IsRoot || res.Modules[1].ModulePath != "example.com/testroot/tools/harness" {
		t.Errorf("unexpected sub-module: %+v", res.Modules[1])
	}

	if len(res.Packages) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(res.Packages))
	}

	// Check depth ordering: depth 0 should be first
	if res.Packages[0].Path != "." || res.Packages[0].Depth != 0 {
		t.Errorf("expected package at depth 0 to be '.', got %+v", res.Packages[0])
	}
	if res.Packages[0].Synopsis != "Package root is the root package." {
		t.Errorf("unexpected synopsis for root: %q", res.Packages[0].Synopsis)
	}

	// Depth 1 should come next
	if res.Packages[1].Path != "pkg1" || res.Packages[1].Depth != 1 {
		t.Errorf("expected package at depth 1 to be 'pkg1', got %+v", res.Packages[1])
	}
	if res.Packages[1].Synopsis != "Package pkg1 provides level 1 utilities." {
		t.Errorf("unexpected synopsis for pkg1: %q", res.Packages[1].Synopsis)
	}

	// Check scoped outline
	scopedRes, err := astedit.WorkspaceOutlineGo(tempDir, "pkg1", false)
	if err != nil {
		t.Fatalf("WorkspaceOutlineGo scoped failed: %v", err)
	}
	if len(scopedRes.Packages) != 2 {
		t.Fatalf("expected 2 scoped packages, got %d", len(scopedRes.Packages))
	}
}
