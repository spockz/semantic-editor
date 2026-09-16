// Package pipeline_test validates atomic file write and formatting pipeline routines.
package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if string(read) != string(data2) {
		t.Errorf("got %q, want %q", string(read), string(data2))
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
	expected0 := "Run 'go get github.com/google/uuid' or use semantic_add_dependency to install the missing dependency."
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
