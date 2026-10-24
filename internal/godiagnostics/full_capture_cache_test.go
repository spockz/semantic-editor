// WHY: This file verifies that full diagnostic captures seed safe incremental reuse without losing scoped or buffered findings.

package godiagnostics_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/telemetry"
)

func assertCaptureEqualsFresh(t *testing.T, root string, fixture cacheInputFixture, got []godiagnostics.Diagnostic, want string) {
	t.Helper()
	fresh, err := godiagnostics.CheckDetailedWithContext(context.Background(), root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("fresh full capture for %q: %v", root, err)
	}
	gotMessages := strings.Join(godiagnostics.Messages(got), "\n")
	freshMessages := strings.Join(godiagnostics.Messages(fresh), "\n")
	if gotMessages != freshMessages {
		t.Fatalf("capture for %q returned %q, fresh full capture returned %q", root, gotMessages, freshMessages)
	}
	if want != "" && !strings.Contains(gotMessages, want) {
		t.Fatalf("capture for %q returned %q, want finding containing %q", root, gotMessages, want)
	}
}

func TestManagedFullCaptureSeedsIncrementalDiagnostics(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"a/value.go": "package a\nfunc Value() int { return 1 }\n",
		"b/value.go": "package b\nfunc Value() int { return missingValue }\n",
	})
	fullMetrics := telemetry.NewMetrics()
	fullCtx := telemetry.WithMetrics(fixture.ctx, fullMetrics)
	full, err := godiagnostics.CheckDetailedWithContext(fullCtx, fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("full capture: %v", err)
	}
	if got := fullMetrics.Snapshot(0).Phases["gopls.request.textDocument/diagnostic"].Count; got != 2 {
		t.Fatalf("full capture pulled diagnostics for %d files, want both files", got)
	}
	assertCaptureEqualsFresh(t, fixture.root, fixture, full, "missingValue")

	incrementalMetrics := telemetry.NewMetrics()
	incrementalCtx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(fixture.ctx, incrementalMetrics))
	incremental, err := godiagnostics.CheckDetailedWithContext(incrementalCtx, fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("unchanged incremental capture after full capture: %v", err)
	}
	assertCaptureEqualsFresh(t, fixture.root, fixture, incremental, "missingValue")
	snapshot := incrementalMetrics.Snapshot(0)
	for _, phase := range []string{"gopls.request.textDocument/diagnostic", "go.packages.load_diagnostics", "go.packages.load_scope"} {
		if got := snapshot.Phases[phase].Count; got != 0 {
			t.Errorf("unchanged incremental capture after full capture ran %s %d times, want 0", phase, got)
		}
	}
}

func TestManagedSubtreeFullCaptureDoesNotReplaceOtherDiagnosticCache(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"a/value.go": "package a\nimport api \"example.invalid/oldmissing\"\nvar Value = api.Value()\n",
		"b/value.go": "package b\nfunc Value() int { return 1 }\n",
	})
	rootA := filepath.Join(fixture.root, "a")
	rootB := filepath.Join(fixture.root, "b")
	initialCtx := godiagnostics.WithIncrementalDiagnostics(fixture.ctx)
	initial, err := godiagnostics.CheckDetailedWithContext(initialCtx, rootA, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("initial incremental capture for package A: %v", err)
	}
	if !strings.Contains(strings.Join(godiagnostics.Messages(initial), "\n"), "example.invalid/oldmissing") {
		t.Fatalf("initial package A capture has no missing-import finding: %q", godiagnostics.Messages(initial))
	}

	writeManagedFixture(t, rootA, "value.go", "package a\nimport api \"example.invalid/newmissing\"\nvar Value = api.Value()\n")
	fullB, err := godiagnostics.CheckDetailedWithContext(fixture.ctx, rootB, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("full capture for package B: %v", err)
	}
	if len(fullB) != 0 {
		t.Fatalf("full capture for package B returned unrelated findings: %+v", fullB)
	}

	metrics := telemetry.NewMetrics()
	incrementalCtx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(fixture.ctx, metrics))
	retainedA, err := godiagnostics.CheckDetailedWithContext(incrementalCtx, rootA, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("incremental capture for package A after full B capture: %v", err)
	}
	assertCaptureEqualsFresh(t, rootA, fixture, retainedA, "example.invalid/newmissing")
	if got := metrics.Snapshot(0).Phases["gopls.request.textDocument/diagnostic"].Count; got == 0 {
		t.Fatal("package A reused a diagnostic baseline after its import changed")
	}
}

func TestManagedExplicitPatternDoesNotAdvanceSiblingDiagnosticBaseline(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"a/value.go": "package a\nfunc Value() int { return 1 }\n",
		"b/value.go": "package b\nfunc Value() int { return initialMissing }\n",
	})
	initial := godiagnostics.WithIncrementalDiagnostics(fixture.ctx)
	if _, err := godiagnostics.CheckDetailedWithContext(initial, fixture.root, fixture.executable, fixture.environment); err != nil {
		t.Fatalf("initial incremental capture: %v", err)
	}
	writeManagedFixture(t, fixture.root, "b/value.go", "package b\nfunc Value() int { return changedSiblingMissing }\n")

	explicitCtx := godiagnostics.WithIncrementalDiagnostics(fixture.ctx)
	explicit, err := godiagnostics.CheckDetailedWithPatternsWithContext(explicitCtx, fixture.root, fixture.executable, fixture.environment, []string{"./a"})
	if err != nil {
		t.Fatalf("explicit package capture: %v", err)
	}
	if len(explicit) != 0 {
		t.Fatalf("explicit package A capture returned sibling findings: %+v", explicit)
	}

	metrics := telemetry.NewMetrics()
	incrementalCtx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(fixture.ctx, metrics))
	retained, err := godiagnostics.CheckDetailedWithContext(incrementalCtx, fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("incremental root capture after explicit pattern: %v", err)
	}
	assertCaptureEqualsFresh(t, fixture.root, fixture, retained, "changedSiblingMissing")
	if got := metrics.Snapshot(0).Phases["gopls.request.textDocument/diagnostic"].Count; got == 0 {
		t.Fatal("root capture reused an unverified sibling diagnostic after an explicit pattern request")
	}
}

func TestManagedBufferedFullCaptureDoesNotSeedDiskDiagnostics(t *testing.T) {
	diskSource := "package sample\nimport api \"example.invalid/not-installed\"\nvar Value = api.Value()\n"
	workingSource := "package sample\nfunc Value() int { return 1 }\n"
	fixture := newCacheInputFixture(t, map[string]string{"value.go": diskSource})
	workingFixture := newCacheInputFixture(t, map[string]string{"value.go": workingSource})
	working, err := godiagnostics.CheckDetailedWithContext(context.Background(), workingFixture.root, workingFixture.executable, workingFixture.environment)
	if err != nil {
		t.Fatalf("fresh working-copy oracle: %v", err)
	}

	buffer, err := sourcebuf.New(fixture.ctx, fixture.root)
	if err != nil {
		t.Fatalf("create source buffer: %v", err)
	}
	valuePath, err := sourcebuf.CanonicalPath(fixture.ctx, filepath.Join(fixture.root, "value.go"))
	if err != nil {
		t.Fatalf("canonicalize buffered source: %v", err)
	}
	if applyErr := buffer.Apply(fixture.ctx, []sourcebuf.Change{{Path: valuePath, Before: []byte(diskSource), After: []byte(workingSource)}}); applyErr != nil {
		t.Fatalf("apply working source: %v", applyErr)
	}
	bufferedCtx := sourcebuf.WithBuffer(fixture.ctx, buffer)
	full, err := godiagnostics.CheckDetailedWithContext(bufferedCtx, fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("buffered full capture: %v", err)
	}
	if strings.Join(godiagnostics.Messages(full), "\n") != strings.Join(godiagnostics.Messages(working), "\n") {
		t.Fatalf("buffered full capture returned %q, working-copy oracle returned %q", godiagnostics.Messages(full), godiagnostics.Messages(working))
	}

	metrics := telemetry.NewMetrics()
	incrementalCtx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(bufferedCtx, metrics))
	incremental, err := godiagnostics.CheckDetailedWithContext(incrementalCtx, fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("incremental capture with same buffer: %v", err)
	}
	if strings.Join(godiagnostics.Messages(incremental), "\n") != strings.Join(godiagnostics.Messages(working), "\n") {
		t.Fatalf("buffered incremental capture returned %q, working-copy oracle returned %q", godiagnostics.Messages(incremental), godiagnostics.Messages(working))
	}
	if got := metrics.Snapshot(0).Phases["gopls.request.textDocument/diagnostic"].Count; got == 0 {
		t.Fatal("buffered full capture seeded reusable diagnostic reports")
	}

	disk, err := godiagnostics.CheckDetailedWithContext(context.Background(), fixture.root, fixture.executable, fixture.environment)
	if err != nil {
		t.Fatalf("fresh disk oracle: %v", err)
	}
	diskMessages := strings.Join(godiagnostics.Messages(disk), "\n")
	if !strings.Contains(diskMessages, "example.invalid/not-installed") {
		t.Fatalf("disk oracle returned %q, want the disk missing-import finding", diskMessages)
	}
}
