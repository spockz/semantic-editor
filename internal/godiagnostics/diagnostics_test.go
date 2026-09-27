// Package godiagnostics_test verifies Go diagnostics through real gopls sessions.

package godiagnostics_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/godiagnostics"
)

func TestDiagnosticsLifecycleAndFailures(t *testing.T) {
	gopls, goplsErr := godiagnostics.FindGopls()
	if goplsErr != nil {
		t.Fatalf("FindGopls() error = %v", goplsErr)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module diagnostics.test\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(root, "first.go")
	second := filepath.Join(root, "second.go")
	if err := os.WriteFile(first, []byte("package sample\n\nfunc First() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondSource := "package sample\n\nfunc Second() int { return 2 }\n"
	if err := os.WriteFile(second, []byte(secondSource), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unit_test.go"), []byte("package sample\n\nimport \"testing\"\n\nfunc TestFirst(t *testing.T) { if First() != 1 { t.Fatal(\"bad result\") } }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "testdata"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "testdata", "oracle.go"), []byte("package ignored\nfunc Broken() int { return \"ignored\" }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "build"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "generated.go"), []byte("package ignored\nfunc Broken() int { return \"ignored\" }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tagged.go"), []byte("//go:build never\n\npackage sample\n\nfunc Broken() int { return \"ignored\" }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	check := func(ctx context.Context, executable string) ([]string, error) {
		return godiagnostics.Check(ctx, root, executable, os.Environ())
	}

	t.Run("real gopls lifecycle and diagnostics", func(t *testing.T) {
		got, err := check(context.Background(), gopls)
		if err != nil || len(got) != 0 {
			t.Fatalf("clean diagnostics = %v, error = %v", got, err)
		}
		broken := "package sample\n\nfunc Second() int { return \"broken\" }\n"
		if err := os.WriteFile(second, []byte(broken), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err = check(context.Background(), gopls)
		if err != nil {
			t.Fatalf("broken diagnostics: %v", err)
		}
		if len(got) != 1 || !strings.Contains(got[0], "second.go:") || !strings.Contains(got[0], "[error]") {
			t.Fatalf("broken diagnostics = %v, want only second.go error", got)
		}
		if err := os.WriteFile(second, []byte(secondSource), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err = check(context.Background(), gopls)
		if err != nil || len(got) != 0 {
			t.Fatalf("fixed diagnostics = %v, error = %v", got, err)
		}

		warning := "package sample\n\nimport \"fmt\"\n\nfunc Warning() { fmt.Printf(\"%d\", \"text\") }\n"
		if err := os.WriteFile(filepath.Join(root, "warning.go"), []byte(warning), 0o600); err != nil {
			t.Fatal(err)
		}
		typed, err := godiagnostics.CheckDetailed(context.Background(), root, gopls, os.Environ())
		if err != nil {
			t.Fatalf("warning diagnostics: %v", err)
		}
		if len(typed) != 1 || typed[0].Severity != 2 || !strings.Contains(typed[0].String(), "warning.go:") || !strings.Contains(typed[0].String(), "[warning]") {
			t.Fatalf("typed diagnostics = %+v, want printf analyzer warning severity 2", typed)
		}

		missingImport := "package sample\n\nimport missing \"diagnostics.invalid/not-installed\"\n\nfunc Missing() { missing.Call() }\n"
		if err := os.WriteFile(filepath.Join(root, "missing.go"), []byte(missingImport), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err = check(context.Background(), gopls)
		if err != nil {
			t.Fatalf("missing import diagnostics: %v", err)
		}
		foundMissingImport := false
		for _, finding := range got {
			foundMissingImport = foundMissingImport || strings.Contains(finding, "diagnostics.invalid/not-installed")
		}
		if !foundMissingImport {
			t.Fatalf("diagnostics = %v, want missing import diagnostic", got)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := check(ctx, filepath.Join(t.TempDir(), "unused-gopls"))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Check() error = %v, want context canceled", err)
		}
	})

	t.Run("unavailable executable", func(t *testing.T) {
		_, err := check(context.Background(), filepath.Join(t.TempDir(), "missing-gopls"))
		if err == nil || !strings.Contains(err.Error(), "start gopls diagnostics session") {
			t.Fatalf("Check() error = %v, want unavailable executable error", err)
		}
	})
}

func TestNoGoFilesIsIncompleteCoverage(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module empty.test\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := godiagnostics.Check(context.Background(), root, filepath.Join(t.TempDir(), "unused-gopls"), os.Environ())
	if err == nil || !strings.Contains(err.Error(), "no Go source files found") {
		t.Fatalf("Check() error = %v, want incomplete source coverage error", err)
	}
}
