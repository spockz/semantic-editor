// This file verifies disposable session cleanup handles read-only cache directories without following workspace symlinks.

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentSessionSeparatesPersistentEvidenceFromFixtureCleanup(t *testing.T) {
	persistentDir := t.TempDir()
	fixtureDir := t.TempDir()
	workDir := filepath.Join(fixtureDir, "run-1")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	session, err := newAgentSession(&behavioralSessionProvider{}, persistentDir, fixtureDir, "split-root", workDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.runTurn(t.Context(), "task", sessionTaskTurn, &RunResult{}); err != nil {
		t.Fatal(err)
	}
	if err := session.close(); err != nil {
		t.Fatalf("close split-root session: %v", err)
	}
	if _, err := os.Stat(session.transcriptPath); err != nil {
		t.Fatalf("persistent transcript was not retained: %v", err)
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("fixture work directory survived cleanup: %v", err)
	}
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Fatalf("fixture root was removed with its session: %v", err)
	}
}

func TestAgentSessionCloseRemovesReadonlyCacheWithoutFollowingSymlinks(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	cacheDir := filepath.Join(workDir, ".scratch", "go", "mod", "pgregory.net", "rapid@v1.0.0")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		t.Fatal(err)
	}
	cachedFile := filepath.Join(cacheDir, "source.go")
	if err := os.WriteFile(cachedFile, []byte("package rapid\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cacheDir, 0o500); err != nil {
		t.Fatal(err)
	}
	outsideDir := filepath.Join(root, "outside")
	if err := os.Mkdir(outsideDir, 0o500); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(workDir, "outside-cache")
	if err := os.Symlink(outsideDir, link); err != nil {
		t.Fatal(err)
	}
	session, sessionErr := newAgentSession(&behavioralSessionProvider{}, root, root, "readonly-cache", workDir)
	if sessionErr != nil {
		t.Fatal(sessionErr)
	}
	capturedWorkDir := filepath.Join(root, "captured-workspace")
	if err := os.Rename(workDir, capturedWorkDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, workDir); err != nil {
		t.Fatal(err)
	}
	if err := session.close(); err != nil {
		t.Fatalf("close session after workspace path became a symlink: %v", err)
	}
	if _, err := os.Lstat(workDir); !os.IsNotExist(err) {
		t.Fatalf("workspace symlink cleanup = %v, want link removed", err)
	}
	info, err := os.Stat(outsideDir)
	if err != nil {
		t.Fatalf("outside symlink target was changed or removed: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o500 {
		t.Fatalf("outside directory mode = %04o, want 0500", got)
	}
	entries, err := os.ReadDir(capturedWorkDir)
	if err != nil {
		t.Fatalf("read captured workspace after cleanup: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("captured workspace retained entries after cleanup: %v", entries)
	}
	if err := os.Remove(capturedWorkDir); err != nil {
		t.Fatalf("remove empty captured workspace: %v", err)
	}
}

func TestAgentSessionCloseRetainsReadonlyWorkspace(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	readonlyDir := filepath.Join(workDir, ".scratch", "go", "mod")
	if err := os.MkdirAll(readonlyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readonlyDir, 0o500); err != nil {
		t.Fatal(err)
	}
	session, sessionErr := newAgentSession(&behavioralSessionProvider{}, root, root, "retained-readonly", workDir)
	if sessionErr != nil {
		t.Fatal(sessionErr)
	}
	session.execution = AgentExecution{Task: &Task{}}
	session.retainWorkDir = true
	session.result = &RunResult{}
	if err := session.close(); err != nil {
		t.Fatalf("close retained session: %v", err)
	}
	if got := session.result.Provenance["retained_working_directory"]; got != workDir {
		t.Fatalf("retained directory provenance = %q, want %q", got, workDir)
	}
	info, err := os.Stat(readonlyDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o500 {
		t.Fatalf("retained cache directory mode = %04o, want 0500", got)
	}
}

func TestExecuteAgentCleansReadonlyModuleCacheAfterPassingOracle(t *testing.T) {
	root := t.TempDir()
	installCodexAppServerFake(t, root, "readonly-cache")
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:      "readonly-module-cache",
			Instruction: "Leave the existing declaration intact.",
			Oracle:      OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"Old"}}},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Old() {}\n")),
	}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmBaseline, Variant: "small", Prompt: "DONE"}
	scratchDir := filepath.Join(root, "scratch")
	result, err := NewRunner(scratchDir).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatalf("execute agent with readonly module cache: %v", err)
	}
	if !result.Success || result.Oracle == nil || !result.Oracle.Passed {
		t.Fatalf("readonly cache cleanup changed a passing task result: success=%t oracle=%+v error=%q", result.Success, result.Oracle, result.Error)
	}
	entries, err := os.ReadDir(scratchDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "run_") {
			t.Fatalf("disposable session workspace was retained: %s", entry.Name())
		}
	}
}

func TestAgentSessionRetainsWorkspaceWithoutDisposableCaches(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	cacheDirs := []string{
		".scratch/go/build/aa",
		".scratch/go/gopls",
		".scratch/go/tmp",
		".scratch/copilot/cache",
		".scratch/copilot/tmp/node-compile-cache",
		".scratch/copilot/home/Library/Caches/copilot/pkg",
		".scratch/opencode/cache",
		".scratch/opencode/tmp",
	}
	for _, relative := range append(cacheDirs, ".scratch/go/mod") {
		if err := os.MkdirAll(filepath.Join(workDir, relative), 0o750); err != nil {
			t.Fatal(err)
		}
		if relative == ".scratch/go/mod" {
			continue
		}
		if err := os.WriteFile(filepath.Join(workDir, relative, "cache-entry"), []byte("generated"), 0o400); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(workDir, ".scratch", "go", "build", "aa"), 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(workDir, ".scratch", "go", "mod"), 0o500); err != nil {
		t.Fatal(err)
	}
	session, err := newAgentSession(&behavioralSessionProvider{}, root, root, "retained-cache-prune", workDir)
	if err != nil {
		t.Fatal(err)
	}
	session.execution = AgentExecution{Task: &Task{}}
	session.retainWorkDir = true
	session.result = &RunResult{}
	if closeErr := session.close(); closeErr != nil {
		t.Fatalf("close retained session: %v", closeErr)
	}
	for _, relative := range cacheDirs {
		if _, statErr := os.Stat(filepath.Join(workDir, relative)); !os.IsNotExist(statErr) {
			t.Errorf("retained cache %q still exists: %v", relative, statErr)
		}
	}
	modInfo, err := os.Stat(filepath.Join(workDir, ".scratch", "go", "mod"))
	if err != nil {
		t.Fatalf("retained module cache was removed: %v", err)
	}
	if got := modInfo.Mode().Perm(); got != 0o500 {
		t.Fatalf("retained module cache mode = %04o, want 0500", got)
	}
	if got := session.result.Provenance["retained_working_directory"]; got != workDir {
		t.Fatalf("retained directory provenance = %q, want %q", got, workDir)
	}
}
