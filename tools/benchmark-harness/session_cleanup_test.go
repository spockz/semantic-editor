// This file verifies disposable session cleanup handles read-only cache directories without following workspace symlinks.

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	session, sessionErr := newAgentSession(&behavioralSessionProvider{}, root, "readonly-cache", workDir)
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
	session, sessionErr := newAgentSession(&behavioralSessionProvider{}, root, "retained-readonly", workDir)
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
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(bin, "codex")
	program := `#!/bin/sh
set -eu
cache="$PWD/.scratch/go/mod/pgregory.net/rapid@v1.0.0"
mkdir -p "$cache"
printf 'package rapid\n' > "$cache/source.go"
chmod 500 "$cache"
printf '{"type":"thread.started","thread_id":"readonly-cache"}\n'
printf '{"type":"turn.started"}\n'
printf '{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"DONE"}}\n'
printf '{"type":"turn.completed","usage":{"input_tokens":2,"output_tokens":1}}\n'
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
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
