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
	root, _, secondSource := createDiagnosticsFixture(t)
	check := func(ctx context.Context, executable string) ([]string, error) {
		return godiagnostics.Check(ctx, root, executable, os.Environ())
	}
	t.Run("real gopls lifecycle and diagnostics", func(t *testing.T) {
		assertRealDiagnosticsLifecycle(t, check, gopls, root, secondSource)
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

func createDiagnosticsFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	second := filepath.Join(root, "second.go")
	secondSource := "package sample\n\nfunc Second() int { return 2 }\n"
	files := map[string]string{
		"go.mod":             "module diagnostics.test\n\ngo 1.24\n",
		"first.go":           "package sample\n\nfunc First() int { return 1 }\n",
		"second.go":          secondSource,
		"unit_test.go":       "package sample\n\nimport \"testing\"\n\nfunc TestFirst(t *testing.T) { if First() != 1 { t.Fatal(\"bad result\") } }\n",
		"testdata/oracle.go": "package ignored\nfunc Broken() int { return \"ignored\" }\n",
		"build/generated.go": "package ignored\nfunc Broken() int { return \"ignored\" }\n",
		"tagged.go":          "//go:build never\n\npackage sample\n\nfunc Broken() int { return \"ignored\" }\n",
	}
	for relative, content := range files {
		writeDiagnosticsFixture(t, root, relative, content)
	}
	return root, second, secondSource
}

func writeDiagnosticsFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertRealDiagnosticsLifecycle(t *testing.T, check func(context.Context, string) ([]string, error), gopls, root, secondSource string) {
	t.Helper()
	got, diagnosticErr := check(context.Background(), gopls)
	if diagnosticErr != nil || len(got) != 0 {
		t.Fatalf("clean diagnostics = %v, error = %v", got, diagnosticErr)
	}
	writeDiagnosticsFixture(t, root, "second.go", "package sample\n\nfunc Second() int { return \"broken\" }\n")
	got, diagnosticErr = check(context.Background(), gopls)
	if diagnosticErr != nil {
		t.Fatalf("broken diagnostics: %v", diagnosticErr)
	}
	if len(got) != 1 || !strings.Contains(got[0], "second.go:") || !strings.Contains(got[0], "[error]") {
		t.Fatalf("broken diagnostics = %v, want only second.go error", got)
	}
	writeDiagnosticsFixture(t, root, "second.go", secondSource)
	got, diagnosticErr = check(context.Background(), gopls)
	if diagnosticErr != nil || len(got) != 0 {
		t.Fatalf("fixed diagnostics = %v, error = %v", got, diagnosticErr)
	}
	assertTypedWarningDiagnostic(t, gopls, root)
	assertMissingImportDiagnostic(t, check, gopls, root)
}

func assertTypedWarningDiagnostic(t *testing.T, gopls, root string) {
	t.Helper()
	warning := "package sample\n\nimport \"fmt\"\n\nfunc Warning() { fmt.Printf(\"%d\", \"text\") }\n"
	writeDiagnosticsFixture(t, root, "warning.go", warning)
	typed, err := godiagnostics.CheckDetailed(context.Background(), root, gopls, os.Environ())
	if err != nil {
		t.Fatalf("warning diagnostics: %v", err)
	}
	if len(typed) != 1 || typed[0].Severity != 2 || !strings.Contains(typed[0].String(), "warning.go:") || !strings.Contains(typed[0].String(), "[warning]") {
		t.Fatalf("typed diagnostics = %+v, want printf analyzer warning severity 2", typed)
	}
}

func assertMissingImportDiagnostic(t *testing.T, check func(context.Context, string) ([]string, error), gopls, root string) {
	t.Helper()
	missingImport := "package sample\n\nimport missing \"diagnostics.invalid/not-installed\"\n\nfunc Missing() { missing.Call() }\n"
	writeDiagnosticsFixture(t, root, "missing.go", missingImport)
	got, err := check(context.Background(), gopls)
	if err != nil {
		t.Fatalf("missing import diagnostics: %v", err)
	}
	for _, finding := range got {
		if strings.Contains(finding, "diagnostics.invalid/not-installed") {
			return
		}
	}
	t.Fatalf("diagnostics = %v, want missing import diagnostic", got)
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
	gopls, goplsErr := godiagnostics.FindGopls()
	if goplsErr != nil {
		t.Fatal(goplsErr)
	}
	root := filepath.Join(t.TempDir(), "consumer")
	write := func(name, source string) {
		writeManagedFixture(t, root, name, source)
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
	environment, environmentErr := gocache.Environment(ctx, root)
	if environmentErr != nil {
		t.Fatal(environmentErr)
	}
	environment = append(environment, "SEMEDIT_GOPLS_COUNT="+counter, "SEMEDIT_REAL_GOPLS="+gopls)
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if closeErr := manager.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	scanRoot := root
	check := func(wantStarts int, wantDiagnostic string) {
		checkManagedFixtureDiagnostics(ctx, t, manager, scanRoot, wrapper, environment, counter, wantStarts, wantDiagnostic)
	}
	runManagedDiagnosticsSequence(ctx, t, root, module, clean, wrapper, manager, environment, write, check, &scanRoot)
}

func runManagedDiagnosticsSequence(ctx context.Context, t *testing.T, root, module, clean, wrapper string, manager *godiagnostics.SessionManager, environment []string, write func(string, string), check func(int, string), scanRoot *string) {
	t.Helper()
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
	*scanRoot = filepath.Join(root, "a")
	check(4, "")
	*scanRoot = filepath.Join(root, "b")
	check(4, "")
	write("../dependency/value.go", "package dependency\n\nfunc Value() string { return \"changed\" }\n")
	*scanRoot = filepath.Join(root, "a")
	check(4, "cannot use")
	*scanRoot = filepath.Join(root, "b")
	check(4, "")
	write("../dependency/value.go", dependencySource)
	*scanRoot = filepath.Join(root, "a")
	check(4, "")
	write("go.mod", module+"\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n// parent changed\n")
	check(5, "")
	write("gopls-wrapper", "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n# executable replaced\n")
	check(6, "")
	validManifest, manifestErr := os.ReadFile(filepath.Join(root, "go.mod"))
	if manifestErr != nil {
		t.Fatal(manifestErr)
	}
	write("go.mod", "this is not a module manifest\n")
	if _, err := manager.CheckDetailed(ctx, *scanRoot, wrapper, environment); err == nil {
		t.Fatal("invalid module configuration did not fail")
	}
	write("go.mod", string(validManifest))
	check(7, "")
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	_, closedErr := manager.CheckDetailed(ctx, root, wrapper, environment)
	if closedErr == nil || !strings.Contains(closedErr.Error(), "manager is closed") {
		t.Fatalf("closed check error = %v", closedErr)
	}
}

func writeManagedFixture(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(path, []byte(source)); err != nil {
		t.Fatal(err)
	}
}

func checkManagedFixtureDiagnostics(ctx context.Context, t *testing.T, manager *godiagnostics.SessionManager, scanRoot, wrapper string, environment []string, counter string, wantStarts int, wantDiagnostic string) {
	t.Helper()
	diagnostics, err := godiagnostics.CheckDetailedWithContext(godiagnostics.WithSessionManager(ctx, manager), scanRoot, wrapper, environment)
	if err != nil {
		t.Fatal(err)
	}
	assertManagedDiagnostic(t, diagnostics, wantDiagnostic)
	starts, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(starts), "start\n"); got != wantStarts {
		t.Fatalf("gopls starts = %d, want %d", got, wantStarts)
	}
}

func assertManagedDiagnostic(t *testing.T, diagnostics []godiagnostics.Diagnostic, want string) {
	t.Helper()
	if want == "" {
		if len(diagnostics) != 0 {
			t.Fatalf("diagnostics = %+v, want clean", diagnostics)
		}
		return
	}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return
		}
	}
	t.Fatalf("diagnostics = %+v, want %q", diagnostics, want)
}
