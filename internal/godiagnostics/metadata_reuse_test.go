// WHY: This file checks that fresh package discovery is reused without losing diagnostic behavior when imports change the graph.

package godiagnostics_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/telemetry"
)

func TestManagedDiagnosticsFreshDiscoveryMatchesFullForImportFailures(t *testing.T) {
	t.Run("missing import create and repair", func(t *testing.T) {
		fixture := newCacheInputFixture(t, map[string]string{
			"provider/value.go": "package provider\nfunc Value() int { return 1 }\n",
			"consumer/value.go": "package consumer\nimport api \"example.com/cacheinputs/provider\"\nvar Value = api.Value()\n",
		})
		checkCachedInputsMatchFull(t, fixture, "")
		warm := checkCachedInputsMatchFull(t, fixture, "")
		assertMetadataCacheHit(t, warm)

		writeManagedFixture(t, fixture.root, "consumer/value.go", "package consumer\nimport api \"example.invalid/not-installed\"\nvar Value = api.Value()\n")
		created := checkRetainedImportResultMatchesFresh(t, fixture, "example.invalid/not-installed")
		assertFreshDiscoveryOnly(t, created)
		assertWorkspaceFolderRefreshCount(t, created, 2, "missing-import creation")

		writeManagedFixture(t, fixture.root, "consumer/value.go", "package consumer\nimport api \"example.com/cacheinputs/provider\"\nvar Value = api.Value()\n")
		repaired := checkRetainedImportResultMatchesFresh(t, fixture, "")
		assertFreshDiscoveryOnly(t, repaired)
		assertWorkspaceFolderRefreshCount(t, repaired, 2, "missing-import repair")
	})

	t.Run("import cycle create and repair", func(t *testing.T) {
		fixture := newCacheInputFixture(t, map[string]string{
			"alpha/value.go": "package alpha\nfunc Value() int { return 1 }\n",
			"beta/value.go":  "package beta\nimport \"example.com/cacheinputs/alpha\"\nfunc Other() int { return alpha.Value() }\n",
		})
		checkCachedInputsMatchFull(t, fixture, "")
		warm := checkCachedInputsMatchFull(t, fixture, "")
		assertMetadataCacheHit(t, warm)

		writeManagedFixture(t, fixture.root, "alpha/value.go", "package alpha\nimport \"example.com/cacheinputs/beta\"\nfunc Value() int { return beta.Other() }\n")
		createdRetained, createdFresh, created := checkRetainedAndFresh(t, fixture)
		assertFreshDiscoveryOnly(t, created)
		assertCycleFindingPresent(t, createdRetained, "retained cyclic capture")
		assertCycleFindingPresent(t, createdFresh, "fresh cyclic capture")
		assertWorkspaceFolderRefreshCount(t, created, 2, "cycle creation")

		writeManagedFixture(t, fixture.root, "alpha/value.go", "package alpha\nfunc Value() int { return 1 }\n")
		repairedRetained, repairedFresh, repaired := checkRetainedAndFresh(t, fixture)
		assertFreshDiscoveryOnly(t, repaired)
		assertNoFindings(t, repairedRetained, "retained repaired capture")
		assertNoFindings(t, repairedFresh, "fresh repaired capture")
		assertWorkspaceFolderRefreshCount(t, repaired, 2, "cycle repair")
	})
}

func TestManagedDiagnosticsSameAliasImportSwitchRefreshesReverseClosure(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"providerint/value.go":    "package api\nfunc Value() int { return 1 }\n",
		"providerstring/value.go": "package api\nfunc Value() string { return \"one\" }\n",
		"consumer/value.go":       "package consumer\nimport api \"example.com/cacheinputs/providerint\"\nvar Value = api.Value()\n",
		"transitive/value.go":     "package transitive\nimport \"example.com/cacheinputs/consumer\"\nvar Value int = consumer.Value\n",
	})
	checkCachedInputsMatchFull(t, fixture, "")
	warm := checkCachedInputsMatchFull(t, fixture, "")
	assertMetadataCacheHit(t, warm)

	writeManagedFixture(t, fixture.root, "consumer/value.go", "package consumer\nimport api \"example.com/cacheinputs/providerstring\"\nvar Value = api.Value()\n")
	changed := checkCachedInputsMatchFull(t, fixture, "cannot use")
	assertFreshDiscoveryOnly(t, changed)
	assertWorkspaceFolderRefreshCount(t, changed, 0, "clean provider switch")
	if got := changed.Phases["gopls.request.textDocument/diagnostic"].Count; got != 2 {
		t.Fatalf("same-alias import switch refreshed %d files, want consumer and transitive dependent", got)
	}

	writeManagedFixture(t, fixture.root, "consumer/value.go", "package consumer\nimport api \"example.com/cacheinputs/providerint\"\nvar Value = api.Value()\n")
	repaired := checkCachedInputsMatchFull(t, fixture, "")
	assertFreshDiscoveryOnly(t, repaired)
	assertWorkspaceFolderRefreshCount(t, repaired, 0, "clean provider switch repair")
	if got := repaired.Phases["gopls.request.textDocument/diagnostic"].Count; got != 2 {
		t.Fatalf("repaired same-alias import switch refreshed %d files, want consumer and transitive dependent", got)
	}
}

func TestManagedDiagnosticsUnchangedCaptureAndBodyFindingRefresh(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"value.go": "package sample\nfunc Value() int { return 1 }\n",
	})
	checkCachedInputsMatchFull(t, fixture, "")
	warm := checkCachedInputsMatchFull(t, fixture, "")
	assertMetadataCacheHit(t, warm)

	writeManagedFixture(t, fixture.root, "value.go", "package sample\nfunc Value() int { return bodyMissing }\n")
	changed := checkCachedInputsMatchFull(t, fixture, "bodyMissing")
	if got := changed.Phases["go.packages.load_scope"].Count; got != 0 {
		t.Fatalf("body-only edit reloaded structural metadata %d times, want 0", got)
	}
	if got := changed.Phases["go.packages.load_diagnostics"].Count; got != 1 {
		t.Fatalf("body-only edit refreshed package findings %d times, want 1", got)
	}
	if got := changed.Phases["gopls.request.textDocument/diagnostic"].Count; got != 1 {
		t.Fatalf("body-only edit pulled diagnostics for %d files, want 1", got)
	}
}

func TestManagedDiagnosticsSingleFilePullDoesNotCoverSibling(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"samepkg/a.go": "package samepkg\nfunc A() int { return 1 }\n",
		"samepkg/b.go": "package samepkg\nfunc B() int { return siblingMissing }\n",
	})
	checkCachedInputsMatchFull(t, fixture, "siblingMissing")

	var raw json.RawMessage
	err := godiagnostics.UseWithContext(fixture.ctx, fixture.root, fixture.executable, fixture.environment, func(ctx context.Context, session *godiagnostics.Session) error {
		document, snapshotErr := session.Snapshot(filepath.Join(fixture.root, "samepkg/a.go"))
		if snapshotErr != nil {
			return snapshotErr
		}
		var requestErr error
		raw, requestErr = session.Request(ctx, "textDocument/diagnostic", map[string]any{
			"textDocument": map[string]string{"uri": document.URI},
		})
		return requestErr
	})
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Items            []json.RawMessage          `json:"items"`
		RelatedDocuments map[string]json.RawMessage `json:"relatedDocuments"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode single-file diagnostic report: %v", err)
	}
	if report.Items == nil || len(report.Items) != 0 {
		t.Fatalf("diagnostic report for clean a.go = %s, want an empty single-file report", raw)
	}
	if len(report.RelatedDocuments) != 0 {
		t.Fatalf("single-file diagnostic report unexpectedly included sibling reports: %s", raw)
	}
	full, checkErr := godiagnostics.CheckDetailedWithContext(fixture.ctx, fixture.root, fixture.executable, fixture.environment)
	if checkErr != nil {
		t.Fatal(checkErr)
	}
	for _, diagnostic := range full {
		if diagnostic.Path == "samepkg/b.go" && strings.Contains(diagnostic.Message, "siblingMissing") {
			return
		}
	}
	t.Fatalf("full package capture did not include b.go finding: %+v", full)
}

func TestManagedDiagnosticsParseMemoFreshness(t *testing.T) {
	goSource := func(pattern, body string) string {
		return "package sample\n\nimport _ \"embed\"\n\n//go:embed " + pattern + "\nvar Content string\n\nfunc Value() int { return len(Content) + " + body + " }\n"
	}
	clean := goSource("first.txt", "1")
	fixture := newCacheInputFixture(t, map[string]string{
		"value.go":  clean,
		"other.go":  "package sample\nfunc Other() int { return 2 }\n",
		"first.txt": "first embed bytes",
		"other.txt": "other embed bytes",
	})
	initial := checkCachedInputsMatchFull(t, fixture, "")
	if got := initial.Phases["go.scope.parse"].Count; got == 0 {
		t.Fatal("initial capture did not parse structural Go inputs")
	}
	unchanged := checkCachedInputsMatchFull(t, fixture, "")
	if got := unchanged.Phases["go.scope.parse"].Count; got != 0 {
		t.Fatalf("unchanged capture parsed %d Go sources, want 0", got)
	}

	valuePath := filepath.Join(fixture.root, "value.go")
	writePreservingSourceIdentity(t, valuePath, goSource("first.txt", "x"))
	bodyEdit := checkCachedInputsMatchFull(t, fixture, "undefined: x")
	assertParseCount(t, bodyEdit, 1, "same-length body edit")

	writeManagedFixture(t, fixture.root, "value.go", goSource("other.txt", "x"))
	directiveEdit := checkCachedInputsMatchFull(t, fixture, "undefined: x")
	assertParseCount(t, directiveEdit, 1, "embed directive edit")

	writeManagedFixture(t, fixture.root, "other.txt", "changed embed bytes")
	embedEdit := checkCachedInputsMatchFull(t, fixture, "undefined: x")
	if got := embedEdit.Phases["go.packages.load_scope"].Count; got != 1 {
		t.Fatalf("embed content change refreshed scope metadata %d times, want 1", got)
	}
	assertParseCount(t, embedEdit, 0, "embed content edit")
	if got := embedEdit.Phases["gopls.request.textDocument/diagnostic"].Count; got == 0 {
		t.Fatal("embed content change reused all diagnostic reports, want pulls")
	}

	for _, operator := range []string{"+", "-"} {
		broken := "package sample\n\nimport _ \"embed\"\n\n//go:embed other.txt\nvar Content string\n\nfunc Value() int { return len(Content) " + operator + " }\n"
		writeManagedFixture(t, fixture.root, "value.go", broken)
		malformed := checkCachedInputsMatchFull(t, fixture, "expected")
		assertParseCount(t, malformed, 1, "malformed source variant")
	}
	writeManagedFixture(t, fixture.root, "value.go", goSource("other.txt", "1"))
	repaired := checkCachedInputsMatchFull(t, fixture, "")
	assertParseCount(t, repaired, 1, "malformed source repair")
}

func TestManagedDiagnosticsMetadataErrorGraphRefreshesAllFiles(t *testing.T) {
	clean := "package alpha\nfunc Value() int { return 1 }\n"
	fixture := newCacheInputFixture(t, map[string]string{
		"alpha/value.go":     clean,
		"beta/value.go":      "package beta\nimport \"example.com/cacheinputs/alpha\"\nfunc Other() int { return alpha.Value() }\n",
		"unrelated/value.go": "package unrelated\nfunc Value() int { return 3 }\n",
	})
	checkCachedInputsMatchFull(t, fixture, "")
	assertMetadataCacheHit(t, checkCachedInputsMatchFull(t, fixture, ""))
	for _, step := range []struct{ name, source, want string }{
		{"create cycle", "package alpha\nimport \"example.com/cacheinputs/beta\"\nfunc Value() int { return beta.Other() }\n", "import cycle"},
		{"repair cycle", clean, ""},
		{"create missing import", "package alpha\nimport api \"example.invalid/not-installed\"\nfunc Value() int { return api.Value() }\n", "example.invalid/not-installed"},
		{"repair missing import", clean, ""},
	} {
		t.Run(step.name, func(t *testing.T) {
			writeManagedFixture(t, fixture.root, "alpha/value.go", step.source)
			var metrics telemetry.Snapshot
			var retained, fresh []godiagnostics.Diagnostic
			if step.name == "create cycle" || step.name == "repair cycle" {
				retained, fresh, metrics = checkRetainedAndFresh(t, fixture)
				if step.want == "" {
					assertNoFindings(t, retained, "retained repaired capture")
					assertNoFindings(t, fresh, "fresh repaired capture")
				} else {
					assertCycleFindingPresent(t, retained, "retained cyclic capture")
					assertCycleFindingPresent(t, fresh, "fresh cyclic capture")
				}
			} else {
				metrics = checkCachedInputsMatchFull(t, fixture, step.want)
			}
			assertFreshDiscoveryOnly(t, metrics)
			assertWorkspaceFolderRefreshCount(t, metrics, 2, step.name)
			if got := metrics.Phases["gopls.request.textDocument/diagnostic"].Count; got != 3 {
				t.Fatalf("metadata-error graph transition pulled %d files, want all 3 including unrelated package", got)
			}
			var warm telemetry.Snapshot
			if step.name == "create cycle" || step.name == "repair cycle" {
				var warmRetained, warmFresh []godiagnostics.Diagnostic
				warmRetained, warmFresh, warm = checkRetainedAndFresh(t, fixture)
				if step.want == "" {
					assertNoFindings(t, warmRetained, "unchanged retained capture")
					assertNoFindings(t, warmFresh, "unchanged fresh capture")
				} else {
					assertCycleFindingPresent(t, warmRetained, "unchanged retained capture")
					assertCycleFindingPresent(t, warmFresh, "unchanged fresh capture")
				}
			} else {
				warm = checkCachedInputsMatchFull(t, fixture, step.want)
			}
			assertMetadataCacheHit(t, warm)
		})
	}
}

func writePreservingSourceIdentity(t *testing.T, path, content string) {
	t.Helper()
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != len(original) {
		t.Fatalf("same-size fixture edit changed length from %d to %d", len(original), len(content))
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString(content)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("same-size fixture edit: write=%v close=%v", writeErr, closeErr)
	}
	if touchErr := os.Chtimes(path, before.ModTime(), before.ModTime()); touchErr != nil {
		t.Fatal(touchErr)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("same-size fixture edit did not preserve inode and mtime")
	}
}

func assertParseCount(t *testing.T, metrics telemetry.Snapshot, want int, change string) {
	t.Helper()
	if got := metrics.Phases["go.scope.parse"].Count; got != want {
		t.Fatalf("%s parsed %d Go sources, want %d", change, got, want)
	}
}

func assertMetadataCacheHit(t *testing.T, metrics telemetry.Snapshot) {
	t.Helper()
	for _, phase := range []string{"go.packages.load_scope", "go.packages.load_diagnostics", "gopls.request.textDocument/diagnostic", "gopls.notify.workspace/didChangeWorkspaceFolders"} {
		if got := metrics.Phases[phase].Count; got != 0 {
			t.Errorf("unchanged capture ran %s %d times, want 0", phase, got)
		}
	}
}

func assertFreshDiscoveryOnly(t *testing.T, metrics telemetry.Snapshot) {
	t.Helper()
	if got := metrics.Phases["go.packages.load_scope"].Count; got != 1 {
		t.Errorf("fresh discovery loaded package metadata %d times, want 1", got)
	}
	if got := metrics.Phases["go.packages.load_diagnostics"].Count; got != 0 {
		t.Errorf("fresh discovery triggered duplicate diagnostic package loads %d times, want 0", got)
	}
	if got := metrics.Phases["gopls.request.initialize"].Count; got != 0 {
		t.Errorf("fresh discovery restarted gopls %d times, want 0", got)
	}
}

func assertWorkspaceFolderRefreshCount(t *testing.T, metrics telemetry.Snapshot, want int, transition string) {
	t.Helper()
	if got := metrics.Phases["gopls.notify.workspace/didChangeWorkspaceFolders"].Count; got != want {
		t.Errorf("%s sent %d workspace-folder notifications, want %d", transition, got, want)
	}
}

func checkRetainedImportResultMatchesFresh(t *testing.T, fixture cacheInputFixture, want string) telemetry.Snapshot {
	t.Helper()
	metrics := telemetry.NewMetrics()
	ctx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(fixture.ctx, metrics))
	retained, retainedErr := godiagnostics.CheckDetailedWithContext(ctx, fixture.root, fixture.executable, fixture.environment)
	fresh, freshErr := godiagnostics.CheckDetailedWithContext(context.Background(), fixture.root, fixture.executable, fixture.environment)
	if (retainedErr == nil) != (freshErr == nil) || (retainedErr != nil && retainedErr.Error() != freshErr.Error()) {
		t.Fatalf("retained error %v differs from fresh error %v", retainedErr, freshErr)
	}
	result := strings.Join(godiagnostics.Messages(retained), "\n")
	if retainedErr != nil {
		result = retainedErr.Error()
	}
	if (want == "" && (retainedErr != nil || result != "")) || (want != "" && !strings.Contains(result, want)) {
		t.Fatalf("retained result = %q, error = %v, want substring %q", result, retainedErr, want)
	}
	if retainedErr == nil && freshErr == nil && strings.Join(godiagnostics.Messages(retained), "\n") != strings.Join(godiagnostics.Messages(fresh), "\n") {
		t.Fatalf("retained diagnostics %q differ from fresh full diagnostics %q", godiagnostics.Messages(retained), godiagnostics.Messages(fresh))
	}
	return metrics.Snapshot(0)
}

func checkRetainedAndFresh(t *testing.T, fixture cacheInputFixture) ([]godiagnostics.Diagnostic, []godiagnostics.Diagnostic, telemetry.Snapshot) {
	t.Helper()
	metrics := telemetry.NewMetrics()
	ctx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithMetrics(fixture.ctx, metrics))
	retained, retainedErr := godiagnostics.CheckDetailedWithContext(ctx, fixture.root, fixture.executable, fixture.environment)
	if retainedErr != nil {
		t.Fatalf("retained diagnostics: %v", retainedErr)
	}
	fresh, freshErr := godiagnostics.CheckDetailedWithContext(context.Background(), fixture.root, fixture.executable, fixture.environment)
	if freshErr != nil {
		t.Fatalf("fresh diagnostics: %v", freshErr)
	}
	return retained, fresh, metrics.Snapshot(0)
}

func assertCycleFindingPresent(t *testing.T, findings []godiagnostics.Diagnostic, capture string) {
	t.Helper()
	for _, finding := range findings {
		if strings.Contains(finding.Message, "import cycle") {
			return
		}
	}
	t.Fatalf("%s returned %+v, want an import-cycle finding", capture, findings)
}

func assertNoFindings(t *testing.T, findings []godiagnostics.Diagnostic, capture string) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("%s returned findings after repair: %+v", capture, findings)
	}
}
