// Test mutation policies through Evaluate and real workspace snapshots so created, changed, and deleted test files remain observable.
package main

import (
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"path/filepath"
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

func TestQualifiedForbiddenMethodSymbolUsesReceiverIdentity(t *testing.T) {
	data, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "task_14_scoped_homonym_rename.txtar"))
	if readErr != nil {
		t.Fatalf("read task 14 fixture: %v", readErr)
	}
	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatalf("parse task 14 fixture: %v", parseErr)
	}
	workDir := t.TempDir()
	if extractErr := task.ExtractTo(workDir); extractErr != nil {
		t.Fatalf("extract task 14 fixture: %v", extractErr)
	}

	typesPath := filepath.Join(workDir, "transport", "types.go")
	types, readErr := os.ReadFile(typesPath)
	if readErr != nil {
		t.Fatalf("read types.go: %v", readErr)
	}
	types = []byte(strings.Replace(string(types), "func (c *Client) Close()", "func (c *Client) Shutdown()", 1))
	if writeErr := pipeline.WriteAtomic(typesPath, types); writeErr != nil {
		t.Fatalf("write renamed types.go: %v", writeErr)
	}
	callsPath := filepath.Join(workDir, "transport", "client_calls.go")
	calls, readErr := os.ReadFile(callsPath)
	if readErr != nil {
		t.Fatalf("read client_calls.go: %v", readErr)
	}
	calls = []byte(strings.Replace(string(calls), "c.Close()", "c.Shutdown()", 1))
	if writeErr := pipeline.WriteAtomic(callsPath, calls); writeErr != nil {
		t.Fatalf("write renamed client_calls.go: %v", writeErr)
	}

	result, evaluateErr := Evaluate(t.Context(), task, workDir, nil)
	if evaluateErr != nil {
		t.Fatalf("evaluate renamed fixture: %v", evaluateErr)
	}
	if !result.Passed {
		t.Fatalf("renamed fixture result = %+v; want pass despite decoy Close methods", result)
	}

	types = append(types, []byte("\nfunc (c *Client) Close() string { return \"legacy\" }\n")...)
	if writeErr := pipeline.WriteAtomic(typesPath, types); writeErr != nil {
		t.Fatalf("restore stale Client.Close: %v", writeErr)
	}
	result, evaluateErr = Evaluate(t.Context(), task, workDir, nil)
	if evaluateErr != nil {
		t.Fatalf("evaluate stale declaration: %v", evaluateErr)
	}
	if result.Passed || result.FailureStage != "level_2_ast" {
		t.Fatalf("stale declaration result = %+v; want level 2 AST failure", result)
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
