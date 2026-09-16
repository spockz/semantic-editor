// Package astedit_test verifies AST editing functionality.
package astedit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/astedit"
)

func TestScaffoldFile_ExplicitPackage(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "model.go")

	pkg, err := astedit.ScaffoldFile(context.Background(), targetPath, "models", astedit.ScaffoldOptions{})
	if err != nil {
		t.Fatalf("ScaffoldFile failed: %v", err)
	}

	if pkg != "models" {
		t.Errorf("expected package models, got %s", pkg)
	}

	content, err := os.ReadFile(filepath.Clean(targetPath))
	if err != nil {
		t.Fatalf("read scaffolded file: %v", err)
	}
	expected := "package models\n"
	if string(content) != expected {
		t.Errorf("got %q, want %q", string(content), expected)
	}
}

func TestScaffoldFile_InferSibling(t *testing.T) {
	tmpDir := t.TempDir()
	sibling := filepath.Join(tmpDir, "existing.go")
	if err := os.WriteFile(sibling, []byte("package service\n\ntype Service struct{}\n"), 0o600); err != nil {
		t.Fatalf("write sibling: %v", err)
	}

	// Add a test file with package service_test to ensure it's skipped
	testSibling := filepath.Join(tmpDir, "existing_test.go")
	if err := os.WriteFile(testSibling, []byte("package service_test\n"), 0o600); err != nil {
		t.Fatalf("write test sibling: %v", err)
	}

	targetPath := filepath.Join(tmpDir, "handler.go")
	pkg, err := astedit.ScaffoldFile(context.Background(), targetPath, "infer", astedit.ScaffoldOptions{})
	if err != nil {
		t.Fatalf("ScaffoldFile infer failed: %v", err)
	}

	if pkg != "service" {
		t.Errorf("expected inferred package 'service', got %q", pkg)
	}

	content, err := os.ReadFile(filepath.Clean(targetPath))
	if err != nil {
		t.Fatalf("read scaffolded file: %v", err)
	}
	if string(content) != "package service\n" {
		t.Errorf("content mismatch: %q", string(content))
	}
}

func TestScaffoldFile_InferOnlyTestSiblings(t *testing.T) {
	tmpDir := t.TempDir()
	testSibling := filepath.Join(tmpDir, "foo_test.go")
	if err := os.WriteFile(testSibling, []byte("package foo_test\n"), 0o600); err != nil {
		t.Fatalf("write test sibling: %v", err)
	}

	targetPath := filepath.Join(tmpDir, "foo.go")
	_, err := astedit.ScaffoldFile(context.Background(), targetPath, "infer", astedit.ScaffoldOptions{})
	if err == nil {
		t.Fatalf("expected ErrInferNoSiblings when only test siblings present")
	}
	if !errors.Is(err, astedit.ErrInferNoSiblings) {
		t.Fatalf("expected ErrInferNoSiblings, got: %v", err)
	}
}

func TestScaffoldFile_FileExistsWithoutOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "foo.go")
	if err := os.WriteFile(targetPath, []byte("package foo\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.ScaffoldFile(context.Background(), targetPath, "bar", astedit.ScaffoldOptions{
		Overwrite: false,
	})
	if err == nil {
		t.Fatalf("expected error when file exists and Overwrite is false")
	}
	if !errors.Is(err, astedit.ErrFileExists) {
		t.Fatalf("expected ErrFileExists, got: %v", err)
	}
}

func TestScaffoldFile_FileExistsWithOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "foo.go")
	if err := os.WriteFile(targetPath, []byte("package foo\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	pkg, err := astedit.ScaffoldFile(context.Background(), targetPath, "bar", astedit.ScaffoldOptions{
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("expected overwrite to succeed, got %v", err)
	}
	if pkg != "bar" {
		t.Errorf("expected package bar, got %s", pkg)
	}

	content, err := os.ReadFile(filepath.Clean(targetPath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(content) != "package bar\n" {
		t.Errorf("expected package bar, got %s", string(content))
	}
}

func TestScaffoldFile_EmptyDirInfer(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "empty", "new.go")

	_, err := astedit.ScaffoldFile(context.Background(), targetPath, "infer", astedit.ScaffoldOptions{})
	if err == nil {
		t.Fatalf("expected ErrInferNoSiblings for empty directory")
	}
	if !errors.Is(err, astedit.ErrInferNoSiblings) {
		t.Fatalf("expected ErrInferNoSiblings, got: %v", err)
	}
}
