// Package mcp tests scoped Go batch verification.
// WHY: Batch scope selection must use cached package closures or conservatively cover each edited module.
package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
)

func TestSelectBatchGoPatternsIncludesTransitiveAndTestOnlyImporters(t *testing.T) {
	root := t.TempDir()
	write := func(relative, contents string) {
		t.Helper()
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/scope\n\ngo 1.23\n")
	write("lib/lib.go", "package lib\n\nfunc Value() int { return 1 }\n")
	write("consumer/consumer.go", "package consumer\n\nimport \"example.test/scope/lib\"\n\nfunc Value() int { return lib.Value() }\n")
	write("testconsumer/anchor.go", "package testconsumer\n")
	write("testconsumer/anchor_test.go", "package testconsumer_test\n\nimport (\n \"testing\"\n \"example.test/scope/lib\"\n)\n\nfunc TestValue(t *testing.T) { if lib.Value() != 1 { t.Fatal(\"value\") } }\n")
	write("unrelated/unrelated.go", "package unrelated\n\nimport _ \"example.test/scope/missing\"\n")

	ctx := coldBatchGoScope(t)
	patterns, reason := selectBatchGoPatterns(ctx, root, []BatchEntry{{Params: []byte(`{"file":"lib/lib.go"}`)}}, []string{root})
	if !strings.Contains(reason, "edited module roots") {
		t.Fatalf("selectBatchGoPatterns reason = %q, want scoped module detail", reason)
	}
	want := []string{"./consumer", "./lib", "./testconsumer"}
	if !slices.Equal(patterns[root], want) {
		t.Fatalf("patterns = %v, want %v", patterns[root], want)
	}
}

func coldBatchGoScope(t *testing.T) context.Context {
	t.Helper()
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if closeErr := manager.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	return godiagnostics.WithSessionManager(t.Context(), manager)
}

func TestSelectBatchGoPatternsFallsBackWithoutCachedGraph(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/scope\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package scope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	patterns, reason := selectBatchGoPatterns(context.Background(), root, []BatchEntry{{Params: []byte(`{"file":"main.go"}`)}}, []string{root})
	if !strings.Contains(reason, "cached Go package scope was unavailable") || !slices.Equal(patterns[root], []string{"./..."}) {
		t.Fatalf("patterns = %v, reason = %q, want conservative full-module fallback", patterns, reason)
	}
}

func TestSelectBatchGoPatternsOmitsUneditedNestedModule(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	root := t.TempDir()
	write := func(relative, contents string) {
		t.Helper()
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/main\n\ngo 1.23\n")
	write("app/app.go", "package app\n\nfunc Value() int { return 1 }\n")
	write("tools/goctl/go.mod", "module example.test/goctl\n\ngo 1.23\n\nrequire example.invalid/missing v0.0.1\n")
	write("tools/goctl/main.go", "package goctl\n\nimport _ \"example.invalid/missing\"\n")
	nested := filepath.Join(root, "tools", "goctl")
	ctx := cacheBatchGoScope(t, root)
	patterns, reason := selectBatchGoPatterns(ctx, root, []BatchEntry{{Params: []byte(`{"file":"app/app.go"}`)}}, []string{root, nested})
	if reason == "" || !strings.Contains(reason, ".") {
		t.Fatalf("selectBatchGoPatterns reason = %q, want selected module detail", reason)
	}
	if !slices.Equal(patterns[root], []string{"./app"}) {
		t.Fatalf("main module patterns = %v, want ./app", patterns[root])
	}
	if _, selected := patterns[nested]; selected {
		t.Fatalf("unrelated nested module was selected: %v", patterns)
	}
}

func TestSelectBatchGoPatternsSelectsEditedNestedModuleReverseImporters(t *testing.T) {
	root := t.TempDir()
	write := func(relative, contents string) {
		t.Helper()
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/main\n\ngo 1.23\n")
	write("main.go", "package main\n")
	write("tools/goctl/go.mod", "module example.test/goctl\n\ngo 1.23\n")
	write("tools/goctl/lib/lib.go", "package lib\n\nfunc Value() int { return 1 }\n")
	write("tools/goctl/consumer/consumer.go", "package consumer\n\nimport \"example.test/goctl/lib\"\n\nfunc Value() int { return lib.Value() }\n")
	nested := filepath.Join(root, "tools", "goctl")
	ctx := cacheBatchGoScope(t, nested)
	patterns, reason := selectBatchGoPatterns(ctx, root, []BatchEntry{{Params: []byte(`{"file":"tools/goctl/lib/lib.go"}`)}}, []string{root, nested})
	if reason == "" {
		t.Fatal("selectBatchGoPatterns did not record selected nested module scope")
	}
	if !slices.Equal(patterns[nested], []string{"./consumer", "./lib"}) {
		t.Fatalf("nested module patterns = %v, want consumer and lib", patterns[nested])
	}
	if _, selected := patterns[root]; selected {
		t.Fatalf("unrelated parent module was selected: %v", patterns)
	}
}

func cacheBatchGoScope(t *testing.T, root string) context.Context {
	t.Helper()
	executable, err := godiagnostics.FindGopls()
	if errors.Is(err, godiagnostics.ErrGoplsNotFound) {
		t.Skip("gopls not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if closeErr := manager.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	ctx := godiagnostics.WithSessionManager(t.Context(), manager)
	environment, err := gocache.Environment(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CheckDetailed(ctx, root, executable, environment); err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestReconcilePublishedGoScopeFallsBackWhenPackageInputsChange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/scope\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package scope\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotBatchWorkspace(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package scope\n\nimport \"fmt\"\n\nfunc Value() int { fmt.Println(1); return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseline := batchVerificationBaseline{GoPatterns: map[string][]string{root: {"."}}, ModuleRoots: []string{root}}
	reconcilePublishedGoScope(t.Context(), root, before, &baseline)
	if baseline.OmittedReason == "" || !slices.Equal(baseline.GoPatterns[root], []string{"./..."}) {
		t.Fatalf("baseline = %+v, want build-input fallback", baseline)
	}
}

func TestReconcilePublishedGoScopeKeepsExplicitScaffoldTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/scope\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotBatchWorkspace(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(root, "generated", "model.go")
	if err := os.MkdirAll(filepath.Dir(generated), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(generated, []byte("package scope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseline := batchVerificationBaseline{
		GoPatterns:        map[string][]string{root: {"./..."}},
		ModuleRoots:       []string{root},
		ScaffoldGoTargets: map[string]string{generated: root},
	}
	reconcilePublishedGoScope(t.Context(), root, before, &baseline)
	if baseline.OmittedReason != "" {
		t.Fatalf("baseline = %+v, want explicit scaffold target to preserve comparable coverage", baseline)
	}
}
