package astedit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
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

func TestWorkspaceOutlineGoScopedModuleOwnership(t *testing.T) {
	root := t.TempDir()
	writeOutlineFile(t, root, "go.mod", "// Root module docs.\n// More details.\nmodule example.com/root\n\ngo 1.24\n")
	writeOutlineFile(t, root, "root.go", "// Package root is the root package.\npackage root\n")
	writeOutlineFile(t, root, "outer/pkg.go", "package pkg\n")
	writeOutlineFile(t, root, "outer/nested/go.mod", "module example.com/root/nested\n\ngo 1.24\n")
	writeOutlineFile(t, root, "outer/nested/pkg/pkg.go", "package pkg\n")
	writeOutlineFile(t, root, "outer/nested/child/go.mod", "module example.com/root/nested/child\n")
	writeOutlineFile(t, root, "outer/nested/child/pkg.go", "package child\n")

	rootResult, err := astedit.WorkspaceOutlineGo(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootResult.Modules) != 3 {
		t.Fatalf("root modules = %#v", rootResult.Modules)
	}
	if rootResult.Modules[0].Doc == nil || *rootResult.Modules[0].Doc != "Root module docs.\nMore details." {
		t.Fatalf("root module doc = %#v", rootResult.Modules[0].Doc)
	}
	if rootResult.Modules[0].Synopsis != "Root module docs. More details." {
		t.Fatalf("root module synopsis = %q", rootResult.Modules[0].Synopsis)
	}
	assertModulePackageCounts(t, rootResult)
	for _, pkg := range rootResult.Packages {
		want := "example.com/root"
		if pkg.Path == "outer/nested/pkg" {
			want = "example.com/root/nested"
		}
		if pkg.Path == "outer/nested/child" {
			want = "example.com/root/nested/child"
		}
		if pkg.Module != want || pkg.Language != "go" {
			t.Errorf("package %q owner/language = %q/%q, want %q/go", pkg.Path, pkg.Module, pkg.Language, want)
		}
	}

	scoped, err := astedit.WorkspaceOutlineGo(root, "outer/nested/pkg", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped.Modules) != 1 || scoped.Modules[0].Path != "outer/nested" {
		t.Fatalf("scoped nested modules = %#v", scoped.Modules)
	}
	if len(scoped.Packages) != 1 || scoped.Packages[0].Module != "example.com/root/nested" {
		t.Fatalf("scoped nested packages = %#v", scoped.Packages)
	}
	if scoped.Modules[0].PackagesCount != 1 {
		t.Fatalf("scoped package count = %d", scoped.Modules[0].PackagesCount)
	}

	rootModuleScope, err := astedit.WorkspaceOutlineGo(root, "outer", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootModuleScope.Modules) != 3 || !rootModuleScope.Modules[0].IsRoot {
		t.Fatalf("root-module subtree modules = %#v", rootModuleScope.Modules)
	}
	assertModulePackageCounts(t, rootModuleScope)

	nestedScope, err := astedit.WorkspaceOutlineGo(root, "outer/nested", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(nestedScope.Modules) != 2 {
		t.Fatalf("nested scope modules = %#v", nestedScope.Modules)
	}
	if nestedScope.Modules[0].Path != "outer/nested" || nestedScope.Modules[1].Path != "outer/nested/child" {
		t.Fatalf("nested scope module paths = %#v", nestedScope.Modules)
	}
}

func TestWorkspaceOutlineGoWorkIncludesExternalUsesWithoutWalkingTheirPackages(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	external := filepath.Join(parent, "external")
	writeOutlineFile(t, root, "go.mod", "module example.com/root\n")
	writeOutlineFile(t, root, "root.go", "package root\n")
	writeOutlineFile(t, root, "local/go.mod", "module example.com/local\n")
	writeOutlineFile(t, root, "local/local.go", "package local\n")
	writeOutlineFile(t, external, "go.mod", "module example.com/external\n")
	writeOutlineFile(t, external, "external.go", "package external\n")
	if err := os.Symlink(external, filepath.Join(root, "external-link")); err != nil {
		t.Fatalf("create external module symlink: %v", err)
	}
	writeOutlineFile(t, root, "go.work", "go 1.24\nuse (\n ./local\n ./external-link\n)\n")

	result, err := astedit.WorkspaceOutlineGo(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 3 {
		t.Fatalf("modules = %#v", result.Modules)
	}
	paths := map[string]backend.WorkspaceModule{}
	for _, module := range result.Modules {
		paths[module.Path] = module
	}
	if paths["../external"].ModulePath != "example.com/external" || paths["../external"].IsRoot {
		t.Errorf("external module = %#v", paths["../external"])
	}
	if paths["../external"].PackagesCount != 0 {
		t.Errorf("external module package count = %d, want 0", paths["../external"].PackagesCount)
	}
	if _, ok := paths["external-link"]; ok {
		t.Errorf("module descriptor retained symlink path: %#v", paths["external-link"])
	}
	if paths["local"].ModulePath != "example.com/local" {
		t.Errorf("local module = %#v", paths["local"])
	}
	if len(result.Packages) != 2 {
		t.Fatalf("packages include external-use contents: %#v", result.Packages)
	}
	assertModulePackageCounts(t, result)

}

func TestWorkspaceOutlineGoMalformedWorkFileIsAnError(t *testing.T) {
	root := t.TempDir()
	writeOutlineFile(t, root, "go.mod", "module example.com/root\n")
	writeOutlineFile(t, root, "go.work", "use (\n ./unfinished\n")
	if _, err := astedit.WorkspaceOutlineGo(root, "", false); err == nil {
		t.Fatal("malformed go.work was silently ignored")
	}
}

func TestWorkspaceOutlineGoMalformedModuleFilesAreErrors(t *testing.T) {
	t.Run("root scope", func(t *testing.T) {
		root := t.TempDir()
		writeOutlineFile(t, root, "go.mod", "module (\n")
		if _, err := astedit.WorkspaceOutlineGo(root, "", false); err == nil {
			t.Fatal("malformed root go.mod was silently accepted")
		}
	})
	t.Run("scoped target", func(t *testing.T) {
		root := t.TempDir()
		writeOutlineFile(t, root, "go.mod", "module example.com/root\n")
		writeOutlineFile(t, root, "nested/go.mod", "module (\n")
		writeOutlineFile(t, root, "nested/pkg.go", "package nested\n")
		if _, err := astedit.WorkspaceOutlineGo(root, "nested", false); err == nil {
			t.Fatal("malformed scoped go.mod was silently accepted")
		}
	})
}

func TestWorkspaceOutlineGoRejectsTargetsOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside-link")); err != nil {
		t.Fatalf("create outside symlink: %v", err)
	}
	for _, target := range []string{"../outside", outside, "outside-link"} {
		t.Run(target, func(t *testing.T) {
			if _, err := astedit.WorkspaceOutlineGo(root, target, false); err == nil {
				t.Fatalf("outside target %q was accepted", target)
			}
		})
	}
}

func TestWorkspaceOutlineGoReportsUnreadableManifestErrors(t *testing.T) {
	t.Run("go.mod", func(t *testing.T) {
		root := t.TempDir()
		manifest := filepath.Join(root, "go.mod")
		if err := os.WriteFile(manifest, []byte("module example.com/root\n"), 0o000); err != nil {
			t.Fatal(err)
		}
		if _, err := os.ReadFile(manifest); err == nil {
			t.Skip("current user can read mode-000 go.mod")
		}
		if _, err := astedit.WorkspaceOutlineGo(root, "", false); err == nil {
			t.Fatal("unreadable go.mod was silently ignored")
		}
	})
	t.Run("go.work", func(t *testing.T) {
		root := t.TempDir()
		writeOutlineFile(t, root, "go.mod", "module example.com/root\n")
		manifest := filepath.Join(root, "go.work")
		if err := os.WriteFile(manifest, []byte("use ./local\n"), 0o000); err != nil {
			t.Fatal(err)
		}
		if _, err := os.ReadFile(manifest); err == nil {
			t.Skip("current user can read mode-000 go.work")
		}
		if _, err := astedit.WorkspaceOutlineGo(root, "", false); err == nil {
			t.Fatal("unreadable go.work was silently ignored")
		}
	})
}

func TestWorkspaceOutlineGoFallsBackToRootPackageDoc(t *testing.T) {
	root := t.TempDir()
	writeOutlineFile(t, root, "go.mod", "module example.com/fallback\n")
	writeOutlineFile(t, root, "root.go", "// Package fallback documents the module.\npackage fallback\n")
	result, err := astedit.WorkspaceOutlineGo(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 1 || result.Modules[0].Doc == nil || *result.Modules[0].Doc != "Package fallback documents the module." {
		t.Fatalf("module package-doc fallback = %#v", result.Modules)
	}
	if result.Modules[0].Synopsis != "Package fallback documents the module." {
		t.Fatalf("module synopsis = %q", result.Modules[0].Synopsis)
	}
}

func assertModulePackageCounts(t *testing.T, result *backend.WorkspaceOutlineResult) {
	t.Helper()
	for _, module := range result.Modules {
		count := 0
		for _, pkg := range result.Packages {
			if pkg.Module == module.ModulePath {
				count++
			}
		}
		if module.PackagesCount != count {
			t.Errorf("module %q package count = %d, want %d", module.Path, module.PackagesCount, count)
		}
	}
}

func writeOutlineFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
