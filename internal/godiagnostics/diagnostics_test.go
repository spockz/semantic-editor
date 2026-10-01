// Package godiagnostics_test verifies Go diagnostics through real gopls sessions.

package godiagnostics_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"semedit/internal/gocache"
	"semedit/internal/godiagnostics"
	"semedit/internal/pipeline"
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

func TestManagedDiagnosticsReuseAndRefresh(t *testing.T) {
	gopls, err := godiagnostics.FindGopls()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "consumer")
	write := func(name, source string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := pipeline.WriteAtomic(filepath.Join(root, name), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	module := "module example.com/manageddiagnostics\n\ngo 1.24\n"
	clean := "package sample\n\nfunc Value() int { return 1 }\n"
	write("go.mod", module)
	write("value.go", clean)
	counter := filepath.Join(root, "gopls-starts")
	wrapper := filepath.Join(root, "gopls-wrapper")
	write("gopls-wrapper", "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n")
	if err := os.Chmod(wrapper, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	environment, err := gocache.Environment(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	environment = append(environment, "SEMEDIT_GOPLS_COUNT="+counter, "SEMEDIT_REAL_GOPLS="+gopls)
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	scanRoot := root
	check := func(wantStarts int, wantDiagnostic string) {
		t.Helper()
		diagnostics, err := godiagnostics.CheckDetailedWithContext(godiagnostics.WithSessionManager(ctx, manager), scanRoot, wrapper, environment)
		if err != nil {
			t.Fatal(err)
		}
		if wantDiagnostic == "" {
			if len(diagnostics) != 0 {
				t.Fatalf("diagnostics = %+v, want clean", diagnostics)
			}
		} else {
			found := false
			for _, diagnostic := range diagnostics {
				found = found || strings.Contains(diagnostic.Message, wantDiagnostic)
			}
			if !found {
				t.Fatalf("diagnostics = %+v, want %q", diagnostics, wantDiagnostic)
			}
		}
		starts, err := os.ReadFile(counter)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(starts), "start\n"); got != wantStarts {
			t.Fatalf("gopls starts = %d, want %d", got, wantStarts)
		}
	}
	check(1, "")
	check(1, "")
	write("value.go", "package sample\n\nfunc Value() int { return missingValue }\n")
	check(1, "missingValue")
	write("value.go", clean)
	check(1, "")
	write("go.mod", module+"\n// changed module configuration\n")
	check(2, "")
	write("extra.go", "package sample\n\nfunc Extra() int { return Value() }\n")
	check(3, "")
	write("../dependency/go.mod", "module example.com/dependency\n\ngo 1.24\n")
	dependencySource := "package dependency\n\nfunc Value() int { return 1 }\n"
	write("../dependency/value.go", dependencySource)
	write("go.mod", module+"\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n")
	write("a/value.go", "package a\n\nimport \"example.com/dependency\"\n\nvar Value int = dependency.Value()\n")
	write("b/value.go", "package b\n\nimport \"example.com/manageddiagnostics/a\"\n\nvar Value int = a.Value\n")
	scanRoot = filepath.Join(root, "a")
	check(4, "")
	scanRoot = filepath.Join(root, "b")
	check(4, "")
	write("../dependency/value.go", "package dependency\n\nfunc Value() string { return \"changed\" }\n")
	scanRoot = filepath.Join(root, "a")
	check(4, "cannot use")
	scanRoot = filepath.Join(root, "b")
	check(4, "")
	write("../dependency/value.go", dependencySource)
	scanRoot = filepath.Join(root, "a")
	check(4, "")
	write("go.mod", module+"\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n// parent changed\n")
	check(5, "")
	write("gopls-wrapper", "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n# executable replaced\n")
	check(6, "")
	validManifest, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	write("go.mod", "this is not a module manifest\n")
	if _, err := manager.CheckDetailed(ctx, scanRoot, wrapper, environment); err == nil {
		t.Fatal("invalid module configuration did not fail")
	}
	write("go.mod", string(validManifest))
	check(7, "")
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = manager.CheckDetailed(ctx, root, wrapper, environment)
	if err == nil || !strings.Contains(err.Error(), "manager is closed") {
		t.Fatalf("closed check error = %v", err)
	}
}
