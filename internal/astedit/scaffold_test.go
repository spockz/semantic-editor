// Package astedit_test verifies AST editing functionality.
package astedit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
	"semedit/internal/sourcebuf"
)

func TestScaffoldFile_ExplicitPackage(t *testing.T) {
	t.Parallel()
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

func TestScaffoldFileBuffersNewFileUntilPublication(t *testing.T) {
	root := t.TempDir()
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	targetPath := filepath.Join(root, "generated", "model.go")

	if _, scaffoldErr := astedit.ScaffoldFile(ctx, targetPath, "models", astedit.ScaffoldOptions{}); scaffoldErr != nil {
		t.Fatalf("ScaffoldFile: %v", scaffoldErr)
	}
	if _, statErr := os.Stat(filepath.Dir(targetPath)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("staged scaffold created parent directory: %v", statErr)
	}
	content, err := sourcebuf.ReadFile(ctx, targetPath)
	if err != nil {
		t.Fatalf("read staged scaffold: %v", err)
	}
	if string(content) != "package models\n" {
		t.Fatalf("staged scaffold = %q", content)
	}
	if _, scaffoldErr := astedit.ScaffoldFile(ctx, targetPath, "models", astedit.ScaffoldOptions{}); !errors.Is(scaffoldErr, astedit.ErrFileExists) {
		t.Fatalf("duplicate staged scaffold error = %v, want ErrFileExists", scaffoldErr)
	}
}

func TestScaffoldFile_InferSibling(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestScaffoldFile_InferPreservesSiblingReadFailure(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sibling := filepath.Join(tmpDir, "broken.go")
	if err := os.Symlink(filepath.Join(tmpDir, "missing.go"), sibling); err != nil {
		t.Fatalf("create broken sibling symlink: %v", err)
	}
	targetPath := filepath.Join(tmpDir, "generated.go")
	_, err := astedit.ScaffoldFile(context.Background(), targetPath, "infer", astedit.ScaffoldOptions{})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want wrapped os.ErrNotExist", err)
	}
	if _, statErr := os.Stat(targetPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("scaffold target stat error = %v, want target unchanged", statErr)
	}
}

func TestScaffoldFile_PurposeHeaderPreservedForExplicitAndInferredPackages(t *testing.T) {
	t.Parallel()
	const header = "// WHY: This file groups the service adapters.\n// Keep this wording exactly."
	tests := []struct {
		name        string
		packageName string
		wantPackage string
	}{
		{name: "explicit", packageName: "models", wantPackage: "models"},
		{name: "inferred", packageName: "infer", wantPackage: "service"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if test.packageName == "infer" {
				if err := os.WriteFile(filepath.Join(tmpDir, "service.go"), []byte("package service\n"), 0o600); err != nil {
					t.Fatalf("write sibling: %v", err)
				}
			}
			targetPath := filepath.Join(tmpDir, "generated.go")
			pkg, err := astedit.ScaffoldFile(context.Background(), targetPath, test.packageName, astedit.ScaffoldOptions{PurposeHeader: header})
			if err != nil {
				t.Fatalf("ScaffoldFile failed: %v", err)
			}
			if pkg != test.wantPackage {
				t.Fatalf("package = %q, want %q", pkg, test.wantPackage)
			}
			content, err := os.ReadFile(targetPath)
			if err != nil {
				t.Fatalf("read scaffolded file: %v", err)
			}
			want := header + "\n\npackage " + test.wantPackage + "\n"
			if string(content) != want {
				t.Errorf("content = %q, want %q", content, want)
			}
		})
	}
}

func TestScaffoldFile_InvalidPurposeHeaderDoesNotCreateFile(t *testing.T) {
	t.Parallel()
	targetPath := filepath.Join(t.TempDir(), "nested", "generated.go")
	_, err := astedit.ScaffoldFile(context.Background(), targetPath, "models", astedit.ScaffoldOptions{PurposeHeader: "type Invalid struct {"})
	if !errors.Is(err, astedit.ErrSyntax) {
		t.Fatalf("error = %v, want ErrSyntax", err)
	}
	if _, err := os.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat scaffolded file error = %v, want file not created", err)
	}
	if _, err := os.Stat(filepath.Dir(targetPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat parent directory error = %v, want parent not created", err)
	}
}

func TestScaffoldFileWrapsPlainPurposeTextWithoutChangingPackage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   []string
	}{
		{
			name:   "plain prose",
			header: "Provides application configuration primitives and parsing.",
			want:   []string{"// Provides application configuration primitives and parsing.", "package config"},
		},
		{
			name:   "mixed existing comments and prose",
			header: "// WHY: Describes the configuration package.\nOwns parsing and validation.",
			want:   []string{"// WHY: Describes the configuration package.", "// Owns parsing and validation.", "package config"},
		},
		{
			name:   "block comment remains intact",
			header: "/*\nPurpose for the configuration package.\n*/",
			want:   []string{"/*\nPurpose for the configuration package.\n*/", "package config"},
		},
		{
			name:   "prose starting with Go keywords",
			header: "package owners can configure typed values.\nfunc describes the public entry point.",
			want:   []string{"// package owners can configure typed values.", "// func describes the public entry point.", "package config"},
		},
		{
			name:   "backticks and comment terminators remain comments",
			header: "Purpose \u0060package\u0060 uses */ safely.\nThe package handles <config>.",
			want:   []string{"// Purpose \u0060package\u0060 uses */ safely.", "// The package handles <config>.", "package config"},
		},
		{
			name:   "leading BOM is treated as encoding marker",
			header: "\ufeffPurpose of this file.",
			want:   []string{"// Purpose of this file.", "package config"},
		},
		{
			name:   "BOM and whitespace means absent header",
			header: "\ufeff \n\t",
			want:   []string{"package config"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.go")
			if _, err := astedit.ScaffoldFile(context.Background(), path, "config", astedit.ScaffoldOptions{PurposeHeader: test.header}); err != nil {
				t.Fatalf("ScaffoldFile: %v", err)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read scaffold: %v", err)
			}
			for _, want := range test.want {
				if !strings.Contains(string(content), want) {
					t.Errorf("scaffold does not contain %q:\n%s", want, content)
				}
			}
		})
	}
}

func TestScaffoldFileRejectsPurposeDeclarationsAndDirectives(t *testing.T) {
	for _, header := range []string{
		"type Config struct{}",
		"type Invalid struct {",
		"package other",
		"package",
		"func Invalid() {",
		"const Invalid =",
		"import (",
		"//go:build linux",
		"Prose \ufeff with interior BOM",
	} {
		t.Run(header, func(t *testing.T) {
			targetPath := filepath.Join(t.TempDir(), "nested", "config.go")
			_, err := astedit.ScaffoldFile(context.Background(), targetPath, "config", astedit.ScaffoldOptions{PurposeHeader: header})
			if !errors.Is(err, astedit.ErrSyntax) {
				t.Fatalf("ScaffoldFile error = %v, want ErrSyntax", err)
			}
			if _, statErr := os.Stat(targetPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("stat scaffolded file error = %v, want file not created", statErr)
			}
		})
	}
}
