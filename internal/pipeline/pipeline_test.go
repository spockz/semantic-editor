// Package pipeline_test validates atomic file write and formatting pipeline routines.
package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bytes"

	"semedit/internal/pipeline"
)

func TestWriteAtomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "test.txt")

	data1 := []byte("first content")
	if err := pipeline.WriteAtomic(target, data1); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	data2 := []byte("second content")
	if err := pipeline.WriteAtomic(target, data2); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	read, err := os.ReadFile(filepath.Clean(target))
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !bytes.Equal(read, data2) {
		t.Errorf("got %q, want %q", string(read), string(data2))
	}
}

func TestWriteAtomicPreservesPermissionsAndAdvancesMtime(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	// #nosec G302 -- the test deliberately uses a non-default mode to verify preservation.
	if err := os.Chmod(target, 0o640); err != nil {
		t.Fatalf("chmod initial file: %v", err)
	}
	oldInfo, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat initial file: %v", err)
	}
	if err := pipeline.WriteAtomic(target, []byte("new")); err != nil {
		t.Fatalf("atomic write failed: %v", err)
	}
	newInfo, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat updated file: %v", err)
	}
	if got, want := newInfo.Mode().Perm(), oldInfo.Mode().Perm(); got != want {
		t.Fatalf("permissions = %o, want %o", got, want)
	}
	if !newInfo.ModTime().After(oldInfo.ModTime()) {
		t.Fatalf("mtime = %v, want after %v", newInfo.ModTime(), oldInfo.ModTime())
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "unformatted.go")
	unformatted := []byte("package   main\n\nfunc   main(  )   {}\n")
	if err := os.WriteFile(target, unformatted, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx := context.Background()
	if err := pipeline.Format(ctx, dir, target); err != nil {
		t.Fatalf("format failed: %v", err)
	}

	formatted, err := os.ReadFile(filepath.Clean(target))
	if err != nil {
		t.Fatalf("read formatted: %v", err)
	}

	expected := "package main\n\nfunc main() {}\n"
	if string(formatted) != expected {
		t.Errorf("got %q, want %q", string(formatted), expected)
	}
}

func TestComputeDelta(t *testing.T) {
	t.Parallel()

	before := []string{"err1: syntax", "err2: unused variable"}
	after := []string{"err1: syntax", "err3: type mismatch"}

	delta := pipeline.ComputeDelta(before, after)

	if delta.NetDelta != 0 {
		t.Errorf("expected NetDelta 0, got %d", delta.NetDelta)
	}
	if len(delta.Introduced) != 1 || delta.Introduced[0] != "err3: type mismatch" {
		t.Errorf("unexpected Introduced: %v", delta.Introduced)
	}
	if len(delta.Resolved) != 1 || delta.Resolved[0] != "err2: unused variable" {
		t.Errorf("unexpected Resolved: %v", delta.Resolved)
	}
}

func TestFindModuleRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subDir := filepath.Join(dir, "pkg", "sub")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	modFile := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(modFile, []byte("module test\n"), 0o600); err != nil {
		t.Fatalf("write mod failed: %v", err)
	}

	found := pipeline.FindModuleRoot(subDir)
	if found != dir {
		t.Errorf("FindModuleRoot(%q) = %q, want %q", subDir, found, dir)
	}
}

func TestComputeDelta_Suggestions(t *testing.T) {
	t.Parallel()

	before := []string{}
	after := []string{
		`server.go:4:2: cannot find package "github.com/google/uuid" in any of:`,
		`main.go:5:2: no required module provides package github.com/stretchr/testify/assert; to add it:`,
	}

	delta := pipeline.ComputeDelta(before, after)

	if len(delta.Suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %v", len(delta.Suggestions), delta.Suggestions)
	}
	expected0 := "Run 'go get github.com/google/uuid' or use semantic_add_build_dependency to install the missing dependency."
	if delta.Suggestions[0] != expected0 {
		t.Errorf("got suggestion[0] %q, want %q", delta.Suggestions[0], expected0)
	}
}

func TestOrganizeImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	// Source with unused import and missing import for fmt.Println
	unorganized := `package main

import (
	"bytes"
)

func main() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(file, []byte(unorganized), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	ctx := context.Background()
	if err := pipeline.OrganizeImports(ctx, dir, file); err != nil {
		t.Fatalf("OrganizeImports failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"fmt"`) {
		t.Errorf("expected fmt to be imported, got:\n%s", content)
	}
	if strings.Contains(content, `"bytes"`) {
		t.Errorf("expected unused bytes import to be removed, got:\n%s", content)
	}
}

func TestOrganizeImportsWithOptions_ExplicitAddAndRemove(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")

	src := `package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("hello")
	var _ = crand.Reader
}

`
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	ctx := context.Background()
	opts := pipeline.ImportOptions{
		Add: []string{
			"crand crypto/rand",
			"_ net/http/pprof",
		},
		Remove: []string{
			"net/http",
		},
	}

	if err := pipeline.OrganizeImportsWithOptions(ctx, dir, opts, file); err != nil {
		t.Fatalf("OrganizeImportsWithOptions failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `crand "crypto/rand"`) {
		t.Errorf("expected crand alias import, got:\n%s", content)
	}
	if !strings.Contains(content, `_ "net/http/pprof"`) {
		t.Errorf("expected blank pprof import, got:\n%s", content)
	}
	if strings.Contains(content, `"net/http"`) && !strings.Contains(content, `"net/http/pprof"`) {
		t.Errorf("expected removed net/http to be absent, got:\n%s", content)
	}
}

func TestOrganizeImportsWithOptions_UpdatesExistingAlias(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	src := `package main

import old "strings"

func main() {
	_ = stringutil.TrimSpace(" x ")
}
`
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	opts := pipeline.ImportOptions{Add: []string{`stringutil "strings"`}}
	if err := pipeline.OrganizeImportsWithOptions(context.Background(), dir, opts, file); err != nil {
		t.Fatalf("OrganizeImportsWithOptions: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `stringutil "strings"`) || strings.Contains(content, `old "strings"`) {
		t.Fatalf("existing import alias was not updated:\n%s", content)
	}
}

func TestGoFilesWorkspaceSelectionAndFormat(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOTMPDIR", filepath.Dir(root))
	unformatted := []byte("package   main\n\nfunc   main(  )   {}\n")
	invalid := []byte("package invalid {\n")
	fixtures := map[string][]byte{
		"valid.go":                      unformatted,
		".scratch/cache.go":             invalid,
		".git/generated.go":             invalid,
		"vendor/module/dependency.go":   invalid,
		"module-cache/example/cache.go": invalid,
		"mod/build/ordinary.go":         unformatted,
	}
	for name, data := range fixtures {
		target := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatalf("create fixture directory %q: %v", name, err)
		}
		if err := os.WriteFile(target, data, 0o640); err != nil {
			t.Fatalf("write fixture %q: %v", name, err)
		}
	}
	cacheRoot := filepath.Join(root, "module-cache")
	t.Setenv("GOMODCACHE", cacheRoot)

	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "outside.go")
	if err := os.WriteFile(outsideFile, unformatted, 0o640); err != nil {
		t.Fatalf("write outside source: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-directory")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(root, "linked.go")); err != nil {
		t.Skipf("file symlinks are unavailable: %v", err)
	}

	relativeFiles, err := pipeline.GoFiles(root, ".")
	if err != nil {
		t.Fatalf("select relative workspace root: %v", err)
	}
	if got, want := strings.Join(relativeFiles, ","), "mod/build/ordinary.go,valid.go"; got != want {
		t.Fatalf("selected relative files = %q, want %q", got, want)
	}
	absoluteFiles, err := pipeline.GoFiles(root, root)
	if err != nil {
		t.Fatalf("select absolute workspace root: %v", err)
	}
	if len(absoluteFiles) != len(relativeFiles) || !filepath.IsAbs(absoluteFiles[0]) {
		t.Fatalf("selected absolute files = %v, want two absolute source paths", absoluteFiles)
	}
	explicitScratch, err := pipeline.GoFiles(root, filepath.Join(".scratch", "cache.go"))
	if err != nil {
		t.Fatalf("select explicit scratch file: %v", err)
	}
	if len(explicitScratch) != 1 {
		t.Fatalf("explicit scratch selection = %v, want its source file", explicitScratch)
	}
	emptyDir := filepath.Join(root, "empty")
	if err := os.Mkdir(emptyDir, 0o700); err != nil {
		t.Fatalf("create empty directory: %v", err)
	}
	emptyFiles, err := pipeline.GoFiles(root, emptyDir)
	if err != nil {
		t.Fatalf("select empty directory: %v", err)
	}
	if len(emptyFiles) != 0 {
		t.Fatalf("empty directory selected files: %v", emptyFiles)
	}
	if err := pipeline.Format(context.Background(), root); err != nil {
		t.Fatalf("format without paths: %v", err)
	}
	if err := pipeline.Format(context.Background(), root, emptyDir); err != nil {
		t.Fatalf("format empty directory: %v", err)
	}
	if _, err := pipeline.GoFiles(root, filepath.Join(root, "missing")); err == nil {
		t.Fatal("selecting a missing path succeeded")
	}
	if err := pipeline.Format(context.Background(), root, filepath.Join(root, "missing")); err == nil {
		t.Fatal("formatting a missing path succeeded")
	}

	needsFormatting, err := pipeline.NeedsFormatting(context.Background(), root, ".")
	if err != nil {
		t.Fatalf("check workspace formatting: %v", err)
	}
	if !needsFormatting {
		t.Fatal("workspace with unformatted sources was reported formatted")
	}
	if err := pipeline.Format(context.Background(), root, "."); err != nil {
		t.Fatalf("format selected workspace sources: %v", err)
	}
	needsFormatting, err = pipeline.NeedsFormatting(context.Background(), root, ".")
	if err != nil {
		t.Fatalf("recheck workspace formatting: %v", err)
	}
	if needsFormatting {
		t.Fatal("workspace remains unformatted after formatting")
	}

	for name, want := range fixtures {
		target := filepath.Join(root, name)
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read fixture %q: %v", name, err)
		}
		if name == "valid.go" || name == "mod/build/ordinary.go" {
			if string(got) != "package main\n\nfunc main() {}\n" {
				t.Errorf("formatted fixture %q = %q", name, got)
			}
			info, err := os.Stat(target)
			if err != nil {
				t.Fatalf("stat formatted fixture %q: %v", name, err)
			}
			if gotMode := info.Mode().Perm(); gotMode != 0o640 {
				t.Errorf("formatted fixture %q mode = %o, want %o", name, gotMode, 0o640)
			}
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("excluded fixture %q changed: got %q, want %q", name, got, want)
		}
	}
	gotOutside, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("read outside source: %v", err)
	}
	if !bytes.Equal(gotOutside, unformatted) {
		t.Errorf("source reachable through symlinks changed: %q", gotOutside)
	}
	directoryLink, err := pipeline.GoFiles(root, filepath.Join(root, "linked-directory"))
	if err != nil {
		t.Fatalf("select explicit directory symlink: %v", err)
	}
	if len(directoryLink) != 0 {
		t.Fatalf("directory symlink selected files: %v", directoryLink)
	}
	explicitLink, err := pipeline.GoFiles(root, filepath.Join(root, "linked.go"))
	if err != nil {
		t.Fatalf("select explicit file symlink: %v", err)
	}
	if len(explicitLink) != 1 {
		t.Fatalf("explicit file symlink selection = %v, want one source", explicitLink)
	}
}

func TestGoFilesSkipsCanonicalGoCacheAlias(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "module-cache")
	cacheFile := filepath.Join(cacheRoot, "example", "cache.go")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o700); err != nil {
		t.Fatalf("create module cache: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("package cached {\n"), 0o600); err != nil {
		t.Fatalf("write module cache source: %v", err)
	}
	aliasRoot := t.TempDir()
	alias := filepath.Join(aliasRoot, "go-module-cache")
	if err := os.Symlink(cacheRoot, alias); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	t.Setenv("GOMODCACHE", alias)

	files, err := pipeline.GoFiles(root, ".")
	if err != nil {
		t.Fatalf("select workspace with aliased module cache: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("module cache through canonical path was selected: %v", files)
	}
}

func TestFormatTreatsDashAsExplicitFilename(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "-")
	if err := os.WriteFile(target, []byte("package   main\n\nfunc   main(  )   {}\n"), 0o600); err != nil {
		t.Fatalf("write dash-named source: %v", err)
	}
	if err := pipeline.Format(context.Background(), root, "-"); err != nil {
		t.Fatalf("format explicit dash-named source: %v", err)
	}
	formatted, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read dash-named source: %v", err)
	}
	if string(formatted) != "package main\n\nfunc main() {}\n" {
		t.Fatalf("dash-named source = %q", formatted)
	}
}

func TestFormatExplicitFileSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "source.go")
	if err := os.WriteFile(target, []byte("package   main\n\nfunc   main(  )   {}\n"), 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	link := filepath.Join(root, "selected.go")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if err := pipeline.Format(context.Background(), root, link); err != nil {
		t.Fatalf("format explicitly selected source symlink: %v", err)
	}
	formatted, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read symlink target: %v", err)
	}
	if string(formatted) != "package main\n\nfunc main() {}\n" {
		t.Fatalf("formatted symlink target = %q", formatted)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("stat selected symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("formatting selected symlink replaced the link")
	}
}

func TestCheckDiagnosticsUsesGopls(t *testing.T) {
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module diagnostics.test\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "source.go"), []byte("package sample\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := pipeline.CheckDiagnostics(context.Background(), moduleDir)
	if err != nil && strings.Contains(err.Error(), "gopls executable not found") {
		t.Skipf("gopls is unavailable: %v", err)
	}
	if err != nil {
		t.Fatalf("CheckDiagnostics() error = %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("CheckDiagnostics() diagnostics = %v, want clean result", diagnostics)
	}
}

func TestCheckDiagnosticDetailsRespectsSelectedSubproject(t *testing.T) {
	moduleDir := t.TempDir()
	selectedDir := filepath.Join(moduleDir, "selected")
	siblingDir := filepath.Join(moduleDir, "broken-sibling")
	for _, dir := range []string{selectedDir, siblingDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module diagnostics.test\n\ngo 1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(selectedDir, "source.go"), []byte("package selected\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siblingDir, "broken.go"), []byte("package broken\n\nfunc Broken( {\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := pipeline.CheckDiagnosticDetails(context.Background(), selectedDir)
	if err != nil && strings.Contains(err.Error(), "gopls executable not found") {
		t.Skipf("gopls is unavailable: %v", err)
	}
	if err != nil {
		t.Fatalf("CheckDiagnosticDetails() error = %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("selected project diagnostics = %v, want no findings from the clean selected subtree", diagnostics)
	}
}
