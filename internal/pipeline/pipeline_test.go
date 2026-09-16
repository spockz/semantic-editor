// Package pipeline_test validates atomic file write and formatting pipeline routines.
package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
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
