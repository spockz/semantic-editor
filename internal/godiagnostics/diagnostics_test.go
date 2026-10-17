// Package godiagnostics_test verifies Go diagnostics through real gopls sessions.

package godiagnostics_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/telemetry"
)

func TestManagedDiagnosticsMalformedFingerprintFreshness(t *testing.T) {
	clean := "package sample\n\nimport _ \"embed\"\n\n//go:embed first.txt\nvar Content string\n\nfunc Value() int { return len(Content) }\n"
	fixture := newCacheInputFixture(t, map[string]string{"value.go": clean, "first.txt": "one"})
	checkCachedInputsMatchFull(t, fixture, "")
	for _, broken := range []string{
		"package sample\n\nimport _ \"embed\"\n\n//go:embed first.txt\nvar Content string\n\nfunc Value() int { return 1 + }\n",
		"package sample\n\nimport _ \"embed\"\n\n//go:embed first.txt\nvar Content string\n\nfunc Value() int { return 2 + }\n",
	} {
		writeManagedFixture(t, fixture.root, "value.go", broken)
		metrics := checkCachedInputsMatchFull(t, fixture, "expected")
		if metrics.Phases["go.packages.load_scope"].Count == 0 {
			t.Fatal("malformed source change did not refresh structural discovery")
		}
	}
	writeManagedFixture(t, fixture.root, "value.go", clean)
	checkCachedInputsMatchFull(t, fixture, "")
}

func TestManagedDiagnosticsEmbedFingerprintFreshness(t *testing.T) {
	source := func(pattern string) string {
		return "package sample\n\nimport \"embed\"\n\n//go:embed " + pattern + "\nvar Content embed.FS\n\nfunc Value() int { return 1 }\n"
	}
	fixture := newCacheInputFixture(t, map[string]string{"value.go": source("*.txt"), "first.txt": "one", "first.dat": "one"})
	checkCachedInputsMatchFull(t, fixture, "")
	writeManagedFixture(t, fixture.root, "value.go", source("*.dat"))
	checkCachedInputsMatchFull(t, fixture, "")
	writeManagedFixture(t, fixture.root, "first.dat", "changed embedded bytes")
	changed := checkCachedInputsMatchFull(t, fixture, "")
	if changed.Phases["go.packages.load_scope"].Count == 0 {
		t.Fatal("changed selected embed bytes reused structural discovery")
	}
	writeManagedFixture(t, fixture.root, "second.dat", "additional match")
	added := checkCachedInputsMatchFull(t, fixture, "")
	if added.Phases["go.packages.load_scope"].Count == 0 {
		t.Fatal("new selected embed file reused structural discovery")
	}
	if err := os.Remove(filepath.Join(fixture.root, "second.dat")); err != nil {
		t.Fatal(err)
	}
	checkCachedInputsMatchFull(t, fixture, "")
	warm := checkCachedInputsMatchFull(t, fixture, "")
	if got := warm.Phases["go.packages.load_scope"].Count; got != 0 {
		t.Fatalf("unchanged embed inputs loaded scope %d times", got)
	}
}

func TestManagedDiagnosticsAtomicEditSkipsSaveNotifications(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{"value.go": "package sample\n\nfunc Value() int { return 1 }\n"})
	checkCachedInputsMatchFull(t, fixture, "")
	for _, change := range []struct{ source, want string }{
		{"package sample\n\nfunc Value() int { return missingAtomic }\n", "missingAtomic"},
		{"package sample\n\nfunc Value() int { return 2 }\n", ""},
	} {
		writeManagedFixture(t, fixture.root, "value.go", change.source)
		metrics := checkCachedInputsMatchFull(t, fixture, change.want)
		for phase, want := range map[string]int{"gopls.notify.textDocument/didChange": 1, "gopls.notify.textDocument/didSave": 0, "gopls.notify.workspace/didChangeWatchedFiles": 0, "gopls.request.initialize": 0} {
			if got := metrics.Phases[phase].Count; got != want {
				t.Errorf("%s=%d want %d", phase, got, want)
			}
		}
	}
}

func TestManagedDiagnosticsPreservedMtimeNotificationFallback(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{"value.go": "package sample\n\nfunc Value() int { return 1 }\n"})
	path := filepath.Join(fixture.root, "value.go")
	stamp := time.Now().Add(-10 * time.Second)
	if stampErr := os.Chtimes(path, stamp, stamp); stampErr != nil {
		t.Fatal(stampErr)
	}
	checkCachedInputsMatchFull(t, fixture, "")
	for _, change := range []struct{ source, want string }{
		{"package sample\n\nfunc Value() int { return missingExternal }\n", "missingExternal"},
		{"package sample\n\nfunc Value() int { return 2 }\n", ""},
	} {
		before, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := file.WriteString(change.source)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatalf("external edit: write=%v close=%v", writeErr, closeErr)
		}
		if stampErr := os.Chtimes(path, stamp, stamp); stampErr != nil {
			t.Fatal(stampErr)
		}
		after, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
			t.Fatal("fixture did not preserve inode and mtime")
		}
		metrics := checkCachedInputsMatchFull(t, fixture, change.want)
		for phase, want := range map[string]int{"gopls.notify.textDocument/didChange": 1, "gopls.notify.textDocument/didSave": 1, "gopls.request.initialize": 0} {
			if got := metrics.Phases[phase].Count; got != want {
				t.Errorf("%s=%d want %d", phase, got, want)
			}
		}
	}
	writeManagedFixture(t, fixture.root, "reload.go", "package sample\n\nimport \"strconv\"\n\nfunc Reload() string { return strconv.Itoa(Value()) }\n")
	checkCachedInputsMatchFull(t, fixture, "")
}

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
		metrics := telemetry.NewMetrics()
		checkCtx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(harness.ctx, metrics))
		checkManagedFixtureDiagnostics(checkCtx, t, harness.manager, *harness.scanRoot, harness.wrapper, harness.environment, harness.counter, wantStarts, wantDiagnostic)
		harness.metrics = append(harness.metrics, metrics.Snapshot(0))
	}
	harness.write = write
	harness.check = check
	runManagedDiagnosticsSequence(ctx, t, harness)
	assertFreshScopeMetadataReuse(t, harness.metrics)
	if got := harness.metrics[1].Phases["gopls.request.textDocument/diagnostic"].Count; got != 0 {
		t.Fatalf("unchanged diagnostics requested from gopls %d times, want 0", got)
	}
	if got := harness.metrics[1].Phases["go.packages.load_scope"].Count; got != 0 {
		t.Fatalf("unchanged scope metadata loaded %d times, want 0", got)
	}
}

func assertFreshScopeMetadataReuse(t *testing.T, metrics []telemetry.Snapshot) {
	t.Helper()
	for _, index := range []int{7, 9} {
		if got := metrics[index].Phases["go.packages.load_scope"].Count; got != 1 {
			t.Errorf("structural scope change at check %d loaded metadata %d times, want 1", index, got)
		}
		if got := metrics[index].Phases["go.packages.load_diagnostics"].Count; got != 0 {
			t.Errorf("fresh scope metadata at check %d triggered %d diagnostic package loads, want 0", index, got)
		}
	}
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
	metrics                               []telemetry.Snapshot
}

func runManagedDiagnosticsSequence(ctx context.Context, t *testing.T, harness managedDiagnosticsHarness) {
	t.Helper()
	runManagedDiagnosticsBasics(t, harness)
	runManagedDiagnosticsDependencies(ctx, t, harness)
	runManagedDiagnosticsFailures(ctx, t, harness)
}

func runManagedDiagnosticsBasics(t *testing.T, harness managedDiagnosticsHarness) {
	check := harness.check
	check(1, "")
	check(1, "")
	harness.write("value.go", "package sample\n\nfunc Value() int { return missingValue }\n")
	check(1, "missingValue")
	check(1, "missingValue")
	harness.write("value.go", harness.clean)
	check(1, "")
	check(1, "")
	harness.write("go.mod", harness.module+"\n// changed module configuration\n")
	check(2, "")
	harness.write("extra.go", "package sample\n\nfunc Extra() int { return Value() }\n")
	check(2, "")
	if err := os.Remove(filepath.Join(harness.root, "extra.go")); err != nil {
		t.Fatal(err)
	}
	check(2, "")
}

func runManagedDiagnosticsDependencies(ctx context.Context, t *testing.T, harness managedDiagnosticsHarness) {
	check := harness.check
	harness.write("../dependency/go.mod", "module example.com/dependency\n\ngo 1.24\n")
	dependencySource := "package dependency\n\nfunc Value() int { return 1 }\n"
	harness.write("../dependency/value.go", dependencySource)
	harness.write("go.mod", harness.module+"\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n")
	harness.write("a/value.go", "package a\n\nimport \"example.com/dependency\"\n\nvar Value int = dependency.Value()\n")
	harness.write("b/value.go", "package b\n\nimport \"example.com/manageddiagnostics/a\"\n\nvar Value int = a.Value\n")
	harness.write("c/value.go", "package c\n\nimport \"example.com/manageddiagnostics/b\"\n\nvar Value int = b.Value\n")
	harness.write("a/value_test.go", "package a_test\n\nimport (\"testing\"; \"example.com/manageddiagnostics/a\")\n\nfunc TestValue(t *testing.T) { if a.Value == 0 { t.Fatal(a.Value) } }\n")
	*harness.scanRoot = filepath.Join(harness.root, "a")
	check(3, "")
	patterns, graphErr := godiagnostics.AffectedPackagePatternsWithContext(godiagnostics.WithSessionManager(ctx, harness.manager), harness.root, []string{"a/value.go"})
	if graphErr != nil {
		t.Fatal(graphErr)
	}
	if strings.Join(patterns, ",") != "./a,./b,./c" {
		t.Fatalf("affected package patterns = %v, want ./a, ./b, ./c", patterns)
	}
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
}

func runManagedDiagnosticsFailures(ctx context.Context, t *testing.T, harness managedDiagnosticsHarness) {
	check := harness.check
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

type cacheInputFixture struct {
	root        string
	executable  string
	ctx         context.Context
	environment []string
	manager     *godiagnostics.SessionManager
}

func newCacheInputFixture(t *testing.T, files map[string]string) cacheInputFixture {
	t.Helper()
	root := t.TempDir()
	writeManagedFixture(t, root, "go.mod", "module example.com/cacheinputs\n\ngo 1.24\n")
	for name, content := range files {
		writeManagedFixture(t, root, name, content)
	}
	executable, err := godiagnostics.FindGopls()
	if err != nil {
		t.Fatal(err)
	}
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if closeErr := manager.Close(); closeErr != nil {
			t.Errorf("close retained diagnostics: %v", closeErr)
		}
	})
	ctx := godiagnostics.WithSessionManager(context.Background(), manager)
	environment, err := gocache.Environment(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	return cacheInputFixture{root: root, executable: executable, ctx: ctx, environment: environment, manager: manager}
}

func checkCachedInputsMatchFull(t *testing.T, fixture cacheInputFixture, want string) telemetry.Snapshot {
	t.Helper()
	metrics := telemetry.NewMetrics()
	ctx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(fixture.ctx, metrics))
	retained, err := godiagnostics.CheckDetailedWithContext(ctx, fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := godiagnostics.CheckDetailedWithContext(context.Background(), fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(godiagnostics.Messages(retained), "\n")
	expected := strings.Join(godiagnostics.Messages(fresh), "\n")
	if got != expected {
		t.Fatalf("retained diagnostics %q differ from fresh full diagnostics %q", got, expected)
	}
	if (want == "" && got != "") || (want != "" && !strings.Contains(got, want)) {
		t.Fatalf("diagnostics = %q, want %q", got, want)
	}
	return metrics.Snapshot(0)
}

func TestManagedDiagnosticsCompilerInputMembership(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{"value.go": "package sample\nfunc Value() int { return 1 }\n"})
	checkCachedInputsMatchFull(t, fixture, "")
	warm := checkCachedInputsMatchFull(t, fixture, "")
	if warm.Phases["go.packages.load_scope"].Count != 0 || warm.Phases["gopls.request.textDocument/diagnostic"].Count != 0 {
		t.Fatalf("unchanged capture did not reuse metadata and diagnostics: %+v", warm)
	}
	writeManagedFixture(t, fixture.root, "bad.c", "this is not C\n")
	checkCachedInputsMatchFull(t, fixture, "C source files not allowed")
	if err := os.Remove(filepath.Join(fixture.root, "bad.c")); err != nil {
		t.Fatal(err)
	}
	checkCachedInputsMatchFull(t, fixture, "")
}

func TestManagedDiagnosticsCompilerInputSymlinkRetarget(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{"value.go": "package sample\nfunc Value() int { return 1 }\n"})
	checkCachedInputsMatchFull(t, fixture, "")
	targets := t.TempDir()
	writeManagedFixture(t, targets, "ignored.txt", "//go:build never\n\nthis is not C\n")
	writeManagedFixture(t, targets, "active.txt", "this is not C\n")
	link := filepath.Join(fixture.root, "bad.c")
	if err := os.Symlink(filepath.Join(targets, "ignored.txt"), link); err != nil {
		t.Fatal(err)
	}
	checkCachedInputsMatchFull(t, fixture, "")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(targets, "active.txt"), link); err != nil {
		t.Fatal(err)
	}
	checkCachedInputsMatchFull(t, fixture, "C source files not allowed")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	checkCachedInputsMatchFull(t, fixture, "")
}

func checkRetainedGraphMatchesFresh(t *testing.T, fixture cacheInputFixture, want string) telemetry.Snapshot {
	t.Helper()
	metrics := telemetry.NewMetrics()
	patterns, err := godiagnostics.AffectedPackagePatternsWithContext(telemetry.WithMetrics(fixture.ctx, metrics), fixture.root, []string{"a/a.go"})
	if err != nil {
		t.Fatal(err)
	}
	fresh := godiagnostics.NewSessionManager()
	defer func() {
		if closeErr := fresh.Close(); closeErr != nil {
			t.Errorf("close fresh graph: %v", closeErr)
		}
	}()
	freshPatterns, err := godiagnostics.AffectedPackagePatternsWithContext(godiagnostics.WithSessionManager(context.Background(), fresh), fixture.root, []string{"a/a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(patterns, ",") != want || strings.Join(patterns, ",") != strings.Join(freshPatterns, ",") {
		t.Fatalf("retained patterns %v, fresh %v, want %s", patterns, freshPatterns, want)
	}
	return metrics.Snapshot(0)
}

func TestManagedGraphUsesCurrentEffectiveEnvironment(t *testing.T) {
	for _, setting := range []string{"GOFLAGS", "GOENV"} {
		t.Run(setting, func(t *testing.T) {
			t.Setenv("GOFLAGS", "")
			config := filepath.Join(t.TempDir(), "go-env")
			writeManagedFixture(t, filepath.Dir(config), filepath.Base(config), "GOFLAGS=\n")
			t.Setenv("GOENV", config)
			fixture := newCacheInputFixture(t, map[string]string{
				"a/a.go":     "package a\nfunc Value() int { return 1 }\n",
				"b/b.go":     "package b\n",
				"b/extra.go": "//go:build extra\n\npackage b\nimport \"example.com/cacheinputs/a\"\nfunc Extra() int { return a.Value() }\n",
			})
			checkCachedInputsMatchFull(t, fixture, "")
			warm := checkRetainedGraphMatchesFresh(t, fixture, "./a")
			if warm.Phases["go.packages.load_scope"].Count != 0 {
				t.Fatalf("unchanged graph did not reuse scope: %+v", warm)
			}
			if setting == "GOFLAGS" {
				t.Setenv("GOFLAGS", "-tags=extra")
			} else {
				writeManagedFixture(t, filepath.Dir(config), filepath.Base(config), "GOFLAGS=-tags=extra\n")
			}
			checkRetainedGraphMatchesFresh(t, fixture, "./a,./b")
			if setting == "GOFLAGS" {
				t.Setenv("GOFLAGS", "")
			} else {
				writeManagedFixture(t, filepath.Dir(config), filepath.Base(config), "GOFLAGS=\n")
			}
			checkRetainedGraphMatchesFresh(t, fixture, "./a")
		})
	}
}

func TestPublishedOverlayFlushSynchronizesFromDisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell wrappers require a Unix-like test host")
	}
	harness := newPublishedOverlayHarness(t)
	checkDirtyPublishedOverlay(t, harness)
	syncSnapshot := publishAndReloadPublishedOverlay(t, harness)
	assertPublishedOverlaySync(t, harness, syncSnapshot)
}

func TestManagedDiagnosticsSequentialEditsReuseBaselineAndClosure(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"provider/value.go":   "package provider\nfunc Value() int { return 1 }\n",
		"consumer/value.go":   "package consumer\nimport \"example.com/cacheinputs/provider\"\nvar Value = provider.Value()\n",
		"transitive/value.go": "package transitive\nimport \"example.com/cacheinputs/consumer\"\nvar Value int = consumer.Value\n",
		"unrelated/value.go":  "package unrelated\nfunc Value() int { return 1 }\n",
	})
	warm := checkCachedInputsMatchFull(t, fixture, "")
	if got := warm.Phases["gopls.request.textDocument/diagnostic"].Count; got != 4 {
		t.Fatalf("initial capture pulled %d files, want all four packages", got)
	}
	currentDiagnosticExpectation := ""
	for _, edit := range []struct {
		name, path, source, want string
		pulls                    int
	}{
		{"first clean edit", "provider/value.go", "package provider\nfunc Value() int { return 2 }\n", "", 3},
		{"second clean edit", "provider/value.go", "package provider\nfunc Value() int { return 3 }\n", "", 3},
		{"unrelated finding", "unrelated/value.go", "package unrelated\nfunc Value() int { return unrelatedMissing }\n", "unrelatedMissing", 1},
		{"retain unrelated finding", "provider/value.go", "package provider\nfunc Value() int { return 4 }\n", "unrelatedMissing", 3},
		{"dependent compiler error", "provider/value.go", "package provider\nfunc Value() string { return \"changed\" }\n", "cannot use", 3},
		{"repair dependent error", "provider/value.go", "package provider\nfunc Value() int { return 5 }\n", "unrelatedMissing", 3},
	} {
		t.Run(edit.name, func(t *testing.T) {
			baseline := checkCachedInputsMatchFull(t, fixture, currentDiagnosticExpectation)
			if got := baseline.Phases["gopls.request.textDocument/diagnostic"].Count; got != 0 {
				t.Errorf("unchanged baseline pulled %d files, want cached findings", got)
			}
			writeManagedFixture(t, fixture.root, edit.path, edit.source)
			after := checkCachedInputsMatchFull(t, fixture, edit.want)
			if got := after.Phases["gopls.request.textDocument/diagnostic"].Count; got != edit.pulls {
				t.Errorf("post-edit diagnostic pulls=%d, want %d", got, edit.pulls)
			}
			if got := after.Phases["gopls.request.initialize"].Count; got != 0 {
				t.Errorf("edit restarted gopls: initialize=%d", got)
			}
			currentDiagnosticExpectation = edit.want
		})
	}
}

type publishedOverlayHarness struct {
	ctx           context.Context
	root          string
	gopls         string
	environment   []string
	manager       *godiagnostics.SessionManager
	diskSource    []byte
	overlaySource []byte
	goArgsLog     string
	goplsStarts   string
	shared        *publishedOverlayShared
}

func newPublishedOverlayHarness(t *testing.T) publishedOverlayHarness {
	t.Helper()
	goplsExecutable, goplsLookupErr := godiagnostics.FindGopls()
	if goplsLookupErr != nil {
		t.Fatal(goplsLookupErr)
	}
	goExecutable, goLookupErr := exec.LookPath("go")
	if goLookupErr != nil {
		t.Fatal(goLookupErr)
	}
	root := t.TempDir()
	writeManagedFixture(t, root, "go.mod", "module example.com/publishedoverlay\n\ngo 1.24\n")
	diskSource := []byte("package sample\n\nfunc Value() int { return 1 }\n")
	overlaySource := []byte("package sample\n\nimport \"strings\"\n\nfunc Value() int { _ = strings.TrimSpace(\"\"); return 2 }\n")
	writeManagedFixture(t, root, "value.go", string(diskSource))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	environment, goArgsLog, goplsStarts := publishedOverlayEnvironment(ctx, t, root, goExecutable, goplsExecutable)
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		closeErr := manager.Close()
		if closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	operationCtx := godiagnostics.WithSessionManager(ctx, manager)
	baseMetrics := telemetry.NewMetrics()
	baseCtx := telemetry.WithMetrics(operationCtx, baseMetrics)
	if diagnostics, warmErr := manager.CheckDetailed(baseCtx, root, filepath.Join(root, "gopls-wrapper"), environment); warmErr != nil {
		t.Fatalf("warm clean managed session: %v", warmErr)
	} else if len(diagnostics) != 0 {
		t.Fatalf("clean source diagnostics = %v, want none", diagnostics)
	}
	return publishedOverlayHarness{
		ctx: operationCtx, root: root, gopls: filepath.Join(root, "gopls-wrapper"),
		environment: environment, manager: manager, diskSource: diskSource, overlaySource: overlaySource,
		goArgsLog: goArgsLog, goplsStarts: goplsStarts, shared: &publishedOverlayShared{},
	}
}

func publishedOverlayEnvironment(ctx context.Context, t *testing.T, root, goExecutable, goplsExecutable string) ([]string, string, string) {
	t.Helper()
	environment, environmentErr := gocache.Environment(ctx, root)
	if environmentErr != nil {
		t.Fatal(environmentErr)
	}
	goBin := filepath.Join(root, "test-bin")
	if mkdirErr := os.MkdirAll(goBin, 0o700); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	goArgsLog := filepath.Join(root, "go-args.log")
	goWrapper := "#!/bin/sh\nif [ \"$SEMEDIT_GOPLS_CHILD\" = \"1\" ]; then printf '%s\\n' BEGIN \"$@\" END >> \"$SEMEDIT_GO_ARGS\"; fi\nexec \"$SEMEDIT_REAL_GO\" \"$@\"\n"
	writeManagedFixture(t, root, filepath.Join("test-bin", "go"), goWrapper)
	if chmodGoErr := os.Chmod(filepath.Join(goBin, "go"), 0o700); chmodGoErr != nil {
		t.Fatal(chmodGoErr)
	}
	goplsStarts := filepath.Join(root, "gopls-starts")
	goplsWrapper := filepath.Join(root, "gopls-wrapper")
	wrapperSource := "#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nexport SEMEDIT_GOPLS_CHILD=1\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n"
	writeManagedFixture(t, root, "gopls-wrapper", wrapperSource)
	if chmodGoplsErr := os.Chmod(goplsWrapper, 0o700); chmodGoplsErr != nil {
		t.Fatal(chmodGoplsErr)
	}
	path := os.Getenv("PATH")
	filteredEnvironment := make([]string, 0, len(environment)+4)
	for _, value := range environment {
		key, _, _ := strings.Cut(value, "=")
		if key == "PATH" {
			path = strings.TrimPrefix(value, "PATH=")
			continue
		}
		filteredEnvironment = append(filteredEnvironment, value)
	}
	return append(filteredEnvironment,
		"PATH="+goBin+string(os.PathListSeparator)+path,
		"SEMEDIT_GO_ARGS="+goArgsLog,
		"SEMEDIT_REAL_GO="+goExecutable,
		"SEMEDIT_GOPLS_COUNT="+goplsStarts,
		"SEMEDIT_REAL_GOPLS="+goplsExecutable,
	), goArgsLog, goplsStarts
}

func checkDirtyPublishedOverlay(t *testing.T, harness publishedOverlayHarness) {
	t.Helper()
	buffer, bufferErr := sourcebuf.New(harness.ctx, harness.root)
	if bufferErr != nil {
		t.Fatal(bufferErr)
	}
	valuePath := filepath.Join(harness.root, "value.go")
	changeErr := buffer.Apply(harness.ctx, []sourcebuf.Change{{Path: valuePath, Before: harness.diskSource, After: harness.overlaySource}})
	if changeErr != nil {
		t.Fatal(changeErr)
	}
	harness.shared.buffer = buffer
	if resetErr := os.WriteFile(harness.goArgsLog, nil, 0o600); resetErr != nil {
		t.Fatalf("reset dirty-overlay Go invocation log: %v", resetErr)
	}
	bufferedCtx := sourcebuf.WithBuffer(harness.ctx, buffer)
	metrics := telemetry.NewMetrics()
	queryCtx := telemetry.WithMetrics(bufferedCtx, metrics)
	diagnostics, queryErr := harness.manager.CheckDetailed(queryCtx, harness.root, harness.gopls, harness.environment)
	if queryErr != nil {
		t.Fatalf("query dirty overlay: %v", queryErr)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("dirty overlay diagnostics = %v, want none", diagnostics)
	}
	args, argsErr := os.ReadFile(harness.goArgsLog)
	if argsErr != nil {
		t.Fatalf("read dirty-overlay Go invocation log: %v", argsErr)
	}
	if !strings.Contains(string(args), "list") {
		t.Fatalf("positive control: dirty query caused no gopls child go list invocation; captured args: %s", args)
	}
	snapshot := metrics.Snapshot(0)
	if got := snapshot.Phases["gopls.notify.textDocument/didChange"].Count; got != 1 {
		t.Errorf("dirty overlay didChange=%d want 1", got)
	}
	if got := snapshot.Phases["gopls.notify.textDocument/didSave"].Count; got != 0 {
		t.Errorf("dirty overlay didSave=%d want 0", got)
	}
	if got := snapshot.Phases["gopls.request.textDocument/diagnostic"].Count; got == 0 {
		t.Error("dirty overlay query made no diagnostic request to gopls")
	}
}

func publishAndReloadPublishedOverlay(t *testing.T, harness publishedOverlayHarness) telemetry.Snapshot {
	t.Helper()
	if harness.shared.buffer == nil {
		t.Fatal("dirty query did not retain its source buffer")
	}
	valuePath := filepath.Join(harness.root, "value.go")
	if flushErr := harness.shared.buffer.Flush(harness.ctx); flushErr != nil {
		t.Fatalf("flush published bytes: %v", flushErr)
	}
	published, readErr := os.ReadFile(valuePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(published, harness.overlaySource) {
		t.Fatalf("published source = %q, want %q", published, harness.overlaySource)
	}
	writeManagedFixture(t, harness.root, "extra.go", "package sample\n\nimport \"strings\"\n\nfunc UsesStrings() int { return len(strings.TrimSpace(\"x\")) }\n")
	if resetErr := os.WriteFile(harness.goArgsLog, nil, 0o600); resetErr != nil {
		t.Fatalf("reset Go invocation log: %v", resetErr)
	}
	metrics := telemetry.NewMetrics()
	diskCtx := telemetry.WithMetrics(sourcebuf.WithoutBuffer(harness.ctx), metrics)
	if syncErr := godiagnostics.SynchronizeWithContext(diskCtx); syncErr != nil {
		t.Fatalf("synchronize published disk state: %v", syncErr)
	}
	diagnostics, queryErr := harness.manager.CheckDetailed(diskCtx, harness.root, harness.gopls, harness.environment)
	if queryErr != nil {
		t.Fatalf("query published disk state after package membership change: %v", queryErr)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("published source diagnostics = %v, want none", diagnostics)
	}
	return metrics.Snapshot(0)
}

func assertPublishedOverlaySync(t *testing.T, harness publishedOverlayHarness, sync telemetry.Snapshot) {
	t.Helper()
	if got := sync.Phases["gopls.notify.textDocument/didChange"].Count; got != 1 {
		t.Errorf("published sync didChange=%d want 1", got)
	}
	if got := sync.Phases["gopls.notify.textDocument/didSave"].Count; got != 0 {
		t.Errorf("published sync didSave=%d want 0", got)
	}
	if got := sync.Phases["go.packages.load_scope"].Count; got == 0 {
		t.Error("published metadata change caused no package scope reload")
	}
	postPublicationArgs, argsErr := os.ReadFile(harness.goArgsLog)
	if argsErr != nil {
		t.Fatalf("read post-publication Go invocation log: %v", argsErr)
	}
	if !strings.Contains(string(postPublicationArgs), "list") {
		t.Errorf("post-publication reload made no observable gopls go list invocation: %s", postPublicationArgs)
	}
	if strings.Contains(string(postPublicationArgs), "-overlay") {
		t.Errorf("post-publication gopls go list still used temporary overlay input: %s", postPublicationArgs)
	}
	starts, startsErr := os.ReadFile(harness.goplsStarts)
	if startsErr != nil {
		t.Fatalf("read gopls process start log: %v", startsErr)
	}
	if got := strings.Count(string(starts), "start\n"); got != 1 {
		t.Errorf("gopls process starts=%d want 1, session was not retained", got)
	}
}

type publishedOverlayShared struct {
	buffer *sourcebuf.Buffer
}

func TestAffectedPackagePatternsExcludeProjectLocalBuildCache(t *testing.T) {
	for _, location := range []string{"inside", "outside"} {
		t.Run(location, func(t *testing.T) {
			fixture := newCacheInputFixture(t, map[string]string{
				"value.go":             "package sample\n\nfunc Value() int { return 1 }\n",
				"value_test.go":        "package sample\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) { if Value() != 1 { t.Fatal(\"value\") } }\n",
				"consumer/consumer.go": "package consumer\n\nimport sample \"example.com/cacheinputs\"\n\nfunc Value() int { return sample.Value() }\n",
			})
			base := t.TempDir()
			if location == "inside" {
				base = filepath.Join(fixture.root, ".scratch", "go")
			}
			ctx := gocache.WithBaseDir(godiagnostics.WithSessionManager(t.Context(), fixture.manager), base)
			environment, err := gocache.Environment(ctx, fixture.root)
			if err != nil {
				t.Fatal(err)
			}
			diagnostics, err := fixture.manager.CheckDetailed(ctx, fixture.root, fixture.executable, environment)
			if err != nil {
				t.Fatal(err)
			}
			assertManagedDiagnostic(t, diagnostics, "")
			patterns, err := godiagnostics.AffectedPackagePatternsWithContext(ctx, fixture.root, []string{"value.go"})
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(patterns, ","); got != "./.,./consumer" {
				t.Fatalf("affected package patterns = %q, want source packages ./.,./consumer", got)
			}
			diagnostics, err = godiagnostics.CheckDetailedWithPatternsWithContext(ctx, fixture.root, fixture.executable, environment, patterns)
			if err != nil {
				t.Fatal(err)
			}
			assertManagedDiagnostic(t, diagnostics, "")
		})
	}
}
