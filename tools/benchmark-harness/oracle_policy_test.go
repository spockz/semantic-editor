// Test mutation policies through Evaluate and real workspace snapshots so created, changed, and deleted test files remain observable.
package main

import (
	"os"
	"path/filepath"
	"semedit/internal/pipeline"
	"strings"
	"testing"
)

func TestEvaluateForbidsTestFileChangesFromWorkspaceSnapshot(t *testing.T) {
	task := loadTask12MutationPolicyFixture(t)
	if !task.Metadata.Oracle.MutationPolicy.ForbidTestEdits {
		t.Fatal("task 12 fixture did not enable test-file protection")
	}
	scenarios := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "modify", mutate: modifyRequestTestFile},
		{name: "create", mutate: createRequestTestFile},
		{name: "delete", mutate: deleteRequestTestFile},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			assertEvaluateRejectsTestFileMutation(t, task, scenario.mutate)
		})
	}
}

func loadTask12MutationPolicyFixture(t *testing.T) *Task {
	t.Helper()
	fixturePath := filepath.Join("..", "..", "testdata", "bench", "task_12_request_identity_migration.txtar")
	data, readErr := os.ReadFile(fixturePath)
	if readErr != nil {
		t.Fatalf("read task 12 fixture: %v", readErr)
	}
	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatalf("parse task 12 fixture: %v", parseErr)
	}
	return task
}

func modifyRequestTestFile(t *testing.T, workDir string) {
	t.Helper()
	path := filepath.Join(workDir, "request", "request_test.go")
	contents, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read existing test file: %v", readErr)
	}
	contents = append(contents, []byte("\n// changed by mutation-policy test\n")...)
	if writeErr := pipeline.WriteAtomic(path, contents); writeErr != nil {
		t.Fatalf("modify test file: %v", writeErr)
	}
}

func createRequestTestFile(t *testing.T, workDir string) {
	t.Helper()
	path := filepath.Join(workDir, "request", "policy_test.go")
	if writeErr := pipeline.WriteAtomic(path, []byte("package request\n")); writeErr != nil {
		t.Fatalf("create test file: %v", writeErr)
	}
}

func deleteRequestTestFile(t *testing.T, workDir string) {
	t.Helper()
	path := filepath.Join(workDir, "request", "request_test.go")
	if removeErr := os.Remove(path); removeErr != nil {
		t.Fatalf("delete test file: %v", removeErr)
	}
}

func assertEvaluateRejectsTestFileMutation(t *testing.T, task *Task, mutate func(*testing.T, string)) {
	t.Helper()
	workDir := t.TempDir()
	if extractErr := task.ExtractTo(workDir); extractErr != nil {
		t.Fatalf("extract task fixture: %v", extractErr)
	}
	before, snapshotErr := snapshotWorkspaceFiles(workDir)
	if snapshotErr != nil {
		t.Fatalf("snapshot initial workspace: %v", snapshotErr)
	}
	mutate(t, workDir)
	modified, changedErr := changedWorkspaceFiles(workDir, before)
	if changedErr != nil {
		t.Fatalf("compute changed workspace files: %v", changedErr)
	}
	result, evaluateErr := Evaluate(t.Context(), task, workDir, modified)
	if evaluateErr != nil {
		t.Fatalf("evaluate test-file mutation: %v", evaluateErr)
	}
	if result.Passed || result.FailureStage != "level_1_mutation_policy" {
		t.Fatalf("Evaluate() = %+v; want level 1 policy failure", result)
	}
	if !strings.Contains(result.ErrorMessage, "test file edits are disallowed") {
		t.Fatalf("Evaluate() error = %q; want test-file policy failure", result.ErrorMessage)
	}
	if !strings.Contains(result.ErrorMessage, "_test.go") {
		t.Fatalf("Evaluate() error = %q; want changed test path", result.ErrorMessage)
	}
}
