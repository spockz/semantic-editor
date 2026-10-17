//go:build tmpfs_integration

// Optional native coverage proves SWE warmup fills RAM caches before any provider is launched.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestSWEWorkspaceWarmupOnTmpfs(t *testing.T) {
	ramRoot, moduleCache := os.Getenv("SEMEDIT_TEST_TMPFS_ROOT"), os.Getenv("SEMEDIT_TEST_SWE_MODCACHE")
	if ramRoot == "" || moduleCache == "" {
		t.Skip("set SEMEDIT_TEST_TMPFS_ROOT and SEMEDIT_TEST_SWE_MODCACHE with a cached grpc-go fixture and built semedit-next")
	}
	if mounted, err := isTmpfsMount(ramRoot); err != nil || !mounted {
		t.Fatalf("invalid tmpfs root: %v", err)
	}
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := readSWEWarmupFixture(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatal(err)
	}
	if task.Metadata.Oracle.AST.File != "" {
		t.Fatal("SWE regression unexpectedly has an AST target")
	}
	workDir, err := os.MkdirTemp(ramRoot, "semedit-swe-warmup-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := os.RemoveAll(workDir); cleanupErr != nil {
			t.Error(cleanupErr)
		}
	})
	scratch := filepath.Join(projectRoot, ".scratch", "benchmarks")
	session, err := newAgentSession(&behavioralSessionProvider{}, scratch, ramRoot, filepath.Base(workDir), workDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := session.close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	ctx := gocache.WithModuleCache(t.Context(), moduleCache)
	runner := NewRunner(scratch, WithModuleCache(moduleCache), WithSemeditPrewarmVerify(true))
	setup, err := prepareAgentWorkspace(ctx, runner, task, Target{Harness: "codex"}, ArmSemedit, "small", workDir, session)
	if err != nil {
		t.Fatal(err)
	}
	if !setup.prewarmApplied || setup.prewarmDuration <= 0 {
		t.Fatal("SWE prewarm did not run")
	}
	for _, name := range []string{"build", "gopls"} {
		entries, readErr := os.ReadDir(filepath.Join(workDir, ".scratch", "go", name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(entries) == 0 {
			t.Fatalf("prewarm left %s cache empty", name)
		}
	}
	if _, err := os.Stat(filepath.Join(workDir, ".scratch", "go", "mod")); !os.IsNotExist(err) {
		t.Fatalf("private module cache exists: %v", err)
	}
	t.Logf("SWE target with no AST assertion prewarmed in %s using %s", setup.prewarmDuration, moduleCache)
}

func readSWEWarmupFixture(projectRoot string) ([]byte, error) {
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("open benchmark project: %w", err)
	}
	data, readErr := root.ReadFile("testdata/bench/grpc__grpc-go-2951.txtar")
	if err := errors.Join(readErr, root.Close()); err != nil {
		return nil, fmt.Errorf("read SWE warmup fixture: %w", err)
	}
	return data, nil
}
