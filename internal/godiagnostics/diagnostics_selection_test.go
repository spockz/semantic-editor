// Package godiagnostics verifies package selection rejects incomplete coverage.
package godiagnostics

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/telemetry"
	"golang.org/x/tools/go/packages"
)

func TestSelectPackageFilesRejectsErrorsWithoutGoFiles(t *testing.T) {
	root := t.TempDir()
	healthy := filepath.Join(root, "healthy.go")
	for _, id := range []string{"orphan", "./.scratch/go/build/92"} {
		for _, policy := range []struct {
			name     string
			explicit bool
		}{{"implicit", false}, {"explicit", true}} {
			t.Run(policy.name+"/"+id, func(t *testing.T) {
				loaded := []*packages.Package{
					{ID: "healthy", GoFiles: []string{healthy}},
					{ID: id, Errors: []packages.Error{{Msg: "build constraints exclude all Go files"}}},
				}
				_, _, err := selectPackageFilesWithPolicy(root, loaded, filepath.Join(root, ".scratch", "go", "build"), policy.explicit)
				if err == nil || !strings.Contains(err.Error(), "incomplete Go package coverage") {
					t.Fatalf("package coverage error = %v, want incomplete package coverage", err)
				}
			})
		}
	}
}

func TestAssembleDiagnosticsScopeCanonicalizesSymlinkedGoFiles(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "source.go")
	if err := os.WriteFile(file, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, aliasRoot); err != nil {
		t.Fatal(err)
	}

	scope, err := assembleDiagnosticsScope(context.Background(), root, []*packages.Package{{
		Module:  &packages.Module{Dir: root},
		GoFiles: []string{filepath.Join(aliasRoot, "source.go")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope.files) != 1 || scope.files[0] != canonical {
		t.Fatalf("scope files = %v, want canonical path %q", scope.files, canonical)
	}
}

func TestSelectPackageFilesKeepsFindingsForUsableFiles(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "broken.go")
	loaded := []*packages.Package{{
		ID:      "broken",
		GoFiles: []string{file},
		Errors:  []packages.Error{{Msg: "missing import"}, {Msg: "missing import"}},
	}}
	files, findings, err := selectPackageFiles(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != file {
		t.Fatalf("files = %v, want %q", files, file)
	}
	if len(findings) != 1 || findings[0].Severity != 1 || findings[0].Source != "package" || !strings.Contains(findings[0].String(), "[package] missing import") {
		t.Fatalf("findings = %+v, want deduplicated package severity 1", findings)
	}
}

func TestAffectedPackagePatternsInvalidatesRetargetedGoSymlink(t *testing.T) {
	fixture := newRetargetedSymlinkGraph(t)
	metrics := telemetry.NewMetrics()
	metricsCtx := telemetry.WithMetrics(fixture.ctx, metrics)

	assertAffectedPackageCoverage(metricsCtx, t, fixture.root)
	assertScopeLoads(t, metrics, 0)
	retargetGoSymlink(t, fixture.alias, fixture.secondTarget)
	assertAffectedPackageCoverage(metricsCtx, t, fixture.root)
	assertScopeLoads(t, metrics, 1)
	assertAffectedPackageCoverage(metricsCtx, t, fixture.root)
	assertScopeLoads(t, metrics, 1)
}

type retargetedSymlinkGraph struct {
	root         string
	ctx          context.Context
	alias        string
	secondTarget string
}

func newRetargetedSymlinkGraph(t *testing.T) retargetedSymlinkGraph {
	t.Helper()
	root := t.TempDir()
	external := t.TempDir()
	module := "module example.com/symlink\n\ngo 1.24\n"
	if writeErr := os.WriteFile(filepath.Join(root, "go.mod"), []byte(module), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(root, "main.go"), []byte("package sample\n\nfunc Main() {}\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	firstTarget := filepath.Join(external, "v1.go")
	secondTarget := filepath.Join(external, "v2.go")
	for _, target := range []string{firstTarget, secondTarget} {
		if writeErr := os.WriteFile(target, []byte("package sample\n\nfunc Linked() int { return 1 }\n"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	alias := filepath.Join(root, "alias.go")
	if linkErr := os.Symlink(firstTarget, alias); linkErr != nil {
		t.Fatal(linkErr)
	}
	executable, findErr := FindGopls()
	if findErr != nil {
		t.Fatal(findErr)
	}
	environment, envErr := gocache.Environment(t.Context(), root)
	if envErr != nil {
		t.Fatal(envErr)
	}
	manager := NewSessionManager()
	t.Cleanup(func() {
		if closeErr := manager.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	ctx := WithSessionManager(t.Context(), manager)
	if useErr := UseWithContext(ctx, root, executable, environment, func(context.Context, *Session) error { return nil }); useErr != nil {
		t.Fatal(useErr)
	}
	return retargetedSymlinkGraph{
		root: root, ctx: ctx, alias: alias, secondTarget: secondTarget,
	}
}

func retargetGoSymlink(t *testing.T, alias, target string) {
	t.Helper()
	if removeErr := os.Remove(alias); removeErr != nil {
		t.Fatal(removeErr)
	}
	if linkErr := os.Symlink(target, alias); linkErr != nil {
		t.Fatal(linkErr)
	}
}

func assertAffectedPackageCoverage(ctx context.Context, t *testing.T, root string) {
	t.Helper()
	patterns, err := AffectedPackagePatternsWithContext(ctx, root, []string{"main.go"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(patterns, ","); got != "./." && got != "./..." {
		t.Fatalf("affected patterns = %v, want root-package coverage", patterns)
	}
}

func assertScopeLoads(t *testing.T, metrics *telemetry.Metrics, want int) {
	t.Helper()
	if got := metrics.Snapshot(0).Phases["go.packages.load_scope"].Count; got != want {
		t.Fatalf("graph lookup loaded package scope %d times, want %d", got, want)
	}
}
