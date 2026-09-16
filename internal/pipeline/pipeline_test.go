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
