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

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

func TestDiagnosticsLifecycleAndFailures(t *testing.T) {
	gopls, goplsErr := godiagnostics.FindGopls()
	if goplsErr != nil {
		t.Fatalf("FindGopls() error = %v", goplsErr)
	}
	fixture := createDiagnosticsFixture(t)
	check := func(ctx context.Context, executable string) ([]string, error) {
		return godiagnostics.Check(ctx, fixture.root, executable, os.Environ())
	}
	t.Run("real gopls lifecycle and diagnostics", func(t *testing.T) {
		assertRealDiagnosticsLifecycle(t, check, gopls, fixture.root, fixture.secondSource)
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

type diagnosticsFixture struct {
	root         string
	secondSource string
}

func createDiagnosticsFixture(t *testing.T) diagnosticsFixture {
	t.Helper()
	root := t.TempDir()
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
	return diagnosticsFixture{root: root, secondSource: secondSource}
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
	harness := managedDiagnosticsHarness{root: root, module: module, clean: clean, wrapper: wrapper, counter: counter, ctx: ctx, manager: manager, environment: environment, scanRoot: &scanRoot}
	check := func(wantStarts int, wantDiagnostic string) {
		checkManagedFixtureDiagnostics(harness.ctx, t, harness.manager, *harness.scanRoot, harness.wrapper, harness.environment, harness.counter, wantStarts, wantDiagnostic)
	}
	harness.write = write
	harness.check = check
	runManagedDiagnosticsSequence(ctx, t, harness)
}

func TestManagedDiagnosticsRetainsSessionAcrossFileMembershipChanges(t *testing.T) {
	gopls, err := godiagnostics.FindGopls()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	module := "module example.com/membership\n\ngo 1.24\n"
	writeManagedFixture(t, root, "go.mod", module)
	writeManagedFixture(t, root, "value.go", "package sample\n\nfunc Value() int { return 1 }\n")
	counter := filepath.Join(root, "gopls-starts")
	wrapper := filepath.Join(root, "gopls-wrapper")
	writeManagedFixture(t, root, "gopls-wrapper", "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n")
	if chmodErr := os.Chmod(wrapper, 0o700); chmodErr != nil {
		t.Fatal(chmodErr)
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
		if closeErr := manager.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	check := func(wantDiagnostic string) {
		t.Helper()
		checkManagedFixtureDiagnostics(ctx, t, manager, root, wrapper, environment, counter, 1, wantDiagnostic)
	}
	check("")
	sourcePath := filepath.Join(root, "extra.go")
	writeManagedFixture(t, root, "extra.go", "package sample\n\nfunc Extra() int { return missingValue }\n")
	check("missingValue")
	writeManagedFixture(t, root, "extra.go", "package sample\n\nfunc Extra() int { return Value() }\n")
	check("")
	if err := os.Remove(sourcePath); err != nil {
		t.Fatal(err)
	}
	check("")
	testPath := filepath.Join(root, "membership_test.go")
	writeManagedFixture(t, root, "membership_test.go", "package sample\n\nimport \"testing\"\n\nfunc TestMembership(t *testing.T) { _ = missingValue }\n")
	check("missingValue")
	writeManagedFixture(t, root, "membership_test.go", "package sample\n\nimport \"testing\"\n\nfunc TestMembership(t *testing.T) { _ = Value() }\n")
	check("")
	if err := os.Remove(testPath); err != nil {
		t.Fatal(err)
	}
	check("")
}

func TestManagedDiagnosticsSessionSurvivesMoveFile(t *testing.T) {
	gopls, err := godiagnostics.FindGopls()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeManagedFixture(t, root, "go.mod", "module example.com/move\n\ngo 1.24\n")
	writeManagedFixture(t, root, "old/value.go", "package old\n\nfunc Value() int { return 1 }\n")
	writeManagedFixture(t, root, "old/value_test.go", "package old\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) { if Value() != 1 { t.Fatal(\"bad value\") } }\n")
	writeManagedFixture(t, root, "new/destination.go", "package new\n")
	counter := filepath.Join(root, "gopls-starts")
	wrapper := filepath.Join(root, "gopls-wrapper")
	writeManagedFixture(t, root, "gopls-wrapper", "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n")
	if chmodErr := os.Chmod(wrapper, 0o700); chmodErr != nil {
		t.Fatal(chmodErr)
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
		if closeErr := manager.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	checkManagedFixtureDiagnostics(ctx, t, manager, root, wrapper, environment, counter, 1, "")
	if _, err := astedit.MoveFile(ctx, filepath.Join(root, "old/value.go"), filepath.Join(root, "new/destination.go"), astedit.MoveFileOptions{MoveMatchingTests: true}); err != nil {
		t.Fatalf("MoveFile() error = %v", err)
	}
	for path, want := range map[string]string{
		"old/value.go":            "package old",
		"old/value_test.go":       "package old",
		"new/destination.go":      "func Value() int",
		"new/destination_test.go": "func TestValue",
	} {
		content, readErr := os.ReadFile(filepath.Join(root, path))
		if readErr != nil {
			t.Fatalf("read %s after MoveFile: %v", path, readErr)
		}
		if !strings.Contains(string(content), want) {
			t.Errorf("%s after MoveFile = %q, want it to contain %q", path, content, want)
		}
	}
	checkManagedFixtureDiagnostics(ctx, t, manager, root, wrapper, environment, counter, 1, "")
}

type managedDiagnosticsHarness struct {
	root, module, clean, wrapper, counter string
	ctx                                   context.Context
	manager                               *godiagnostics.SessionManager
	environment                           []string
	write                                 func(string, string)
	check                                 func(int, string)
	scanRoot                              *string
}

func runManagedDiagnosticsSequence(ctx context.Context, t *testing.T, harness managedDiagnosticsHarness) {
	t.Helper()
	check := harness.check
	check(1, "")
	check(1, "")
	harness.write("value.go", "package sample\n\nfunc Value() int { return missingValue }\n")
	check(1, "missingValue")
	harness.write("value.go", harness.clean)
	check(1, "")
	harness.write("go.mod", harness.module+"\n// changed module configuration\n")
	check(2, "")
	harness.write("extra.go", "package sample\n\nfunc Extra() int { return Value() }\n")
	check(2, "")
	harness.write("../dependency/go.mod", "module example.com/dependency\n\ngo 1.24\n")
	dependencySource := "package dependency\n\nfunc Value() int { return 1 }\n"
	harness.write("../dependency/value.go", dependencySource)
	harness.write("go.mod", harness.module+"\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n")
	harness.write("a/value.go", "package a\n\nimport \"example.com/dependency\"\n\nvar Value int = dependency.Value()\n")
	harness.write("b/value.go", "package b\n\nimport \"example.com/manageddiagnostics/a\"\n\nvar Value int = a.Value\n")
	*harness.scanRoot = filepath.Join(harness.root, "a")
	check(3, "")
	*harness.scanRoot = filepath.Join(harness.root, "b")
	check(3, "")
	harness.write("../dependency/value.go", "package dependency\n\nfunc Value() string { return \"changed\" }\n")
	*harness.scanRoot = filepath.Join(harness.root, "a")
	check(3, "cannot use")
	*harness.scanRoot = filepath.Join(harness.root, "b")
	check(3, "")
	harness.write("../dependency/value.go", dependencySource)
	*harness.scanRoot = filepath.Join(harness.root, "a")
	check(3, "")
	harness.write("go.mod", harness.module+"\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n// parent changed\n")
	check(4, "")
	harness.write("gopls-wrapper", "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n# executable replaced\n")
	check(5, "")
	validManifest, manifestErr := os.ReadFile(filepath.Join(harness.root, "go.mod"))
	if manifestErr != nil {
		t.Fatal(manifestErr)
	}
	harness.write("go.mod", "this is not a module manifest\n")
	if _, err := harness.manager.CheckDetailed(ctx, *harness.scanRoot, harness.wrapper, harness.environment); err == nil {
		t.Fatal("invalid module configuration did not fail")
	}
	harness.write("go.mod", string(validManifest))
	check(6, "")
	if err := harness.manager.Close(); err != nil {
		t.Fatal(err)
	}
	_, closedErr := harness.manager.CheckDetailed(ctx, harness.root, harness.wrapper, harness.environment)
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
