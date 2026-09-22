// Package mcp_test verifies MCP protocol server and batch execution functionality.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/mcp"
	"semedit/internal/telemetry"
)

func TestBatch_SuccessfulExecution(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.go")
	if err := os.WriteFile(file1, []byte("package testpkg\n\nfunc Foo() string {\n\treturn \"old\"\n}\n"), 0o600); err != nil {
		t.Fatalf("write file1: %v", err)
	}

	file2 := filepath.Join(tmpDir, "file2.go")
	if err := os.WriteFile(file2, []byte("package testpkg\n\nfunc Dispatch(op string) string {\n\tswitch op {\n\tdefault:\n\t\treturn \"none\"\n\t}\n}\n"), 0o600); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	var out bytes.Buffer
	srv := mcp.NewServer("full", tmpDir, &out)
	metrics := telemetry.NewMetrics()

	edits := []mcp.BatchEntry{
		{
			Tool:   "semantic_replace_body",
			Params: json.RawMessage(`{"file":"file1.go","symbol":"Foo","body":"return \"new\""}`),
		},
		{
			Tool:   "semantic_insert_case",
			Params: json.RawMessage(`{"file":"file2.go","func":"Dispatch","switch_on":"op","case":"case \"ping\":\n\treturn \"pong\""}`),
		},
	}

	resp, err := srv.ExecuteBatch(telemetry.WithMetrics(context.Background(), metrics), edits, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.DiagnosticDelta == nil {
		t.Fatal("successful batch omitted diagnostic delta")
	}
	snapshot := metrics.Snapshot(0)
	if before, after := snapshot.Phases[string(telemetry.PhaseVerificationBefore)].Count, snapshot.Phases[string(telemetry.PhaseVerificationAfter)].Count; before != 1 || after != 1 {
		t.Fatalf("diagnostic invocation counts = before:%d after:%d, want 1 each", before, after)
	}
	if resp.Results[0].Status != "ok" || resp.Results[1].Status != "ok" {
		t.Errorf("expected all results ok: %+v", resp.Results)
	}

	c1, err := os.ReadFile(filepath.Clean(file1))
	if err != nil {
		t.Fatalf("read file1: %v", err)
	}
	if !strings.Contains(string(c1), `return "new"`) {
		t.Errorf("file1 not updated:\n%s", string(c1))
	}

	c2, err := os.ReadFile(filepath.Clean(file2))
	if err != nil {
		t.Fatalf("read file2: %v", err)
	}
	if !strings.Contains(string(c2), `case "ping":`) {
		t.Errorf("file2 not updated:\n%s", string(c2))
	}
}

func TestBatch_FailFast(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.go")
	if err := os.WriteFile(file1, []byte("package testpkg\n\nfunc Foo() string {\n\treturn \"old\"\n}\n"), 0o600); err != nil {
		t.Fatalf("write file1: %v", err)
	}

	file2 := filepath.Join(tmpDir, "file2.go")
	initialFile2 := "package testpkg\n\nfunc Bar() {}\n"
	if err := os.WriteFile(file2, []byte(initialFile2), 0o600); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	var out bytes.Buffer
	srv := mcp.NewServer("full", tmpDir, &out)

	edits := []mcp.BatchEntry{
		{
			// Edit 1: succeeds
			Tool:   "semantic_replace_body",
			Params: json.RawMessage(`{"file":"file1.go","symbol":"Foo","body":"return \"new\""}`),
		},
		{
			// Edit 2: fails (symbol not found)
			Tool:   "semantic_replace_body",
			Params: json.RawMessage(`{"file":"file2.go","symbol":"NonExistent","body":"return 1"}`),
		},
		{
			// Edit 3: should be skipped
			Tool:   "semantic_insert_case",
			Params: json.RawMessage(`{"file":"file2.go","func":"Bar","case":"case 1:"}`),
		},
	}

	resp, err := srv.ExecuteBatch(context.Background(), edits, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned unexpected error: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("expected status error, got %s", resp.Status)
	}
	if resp.DiagnosticDelta != nil {
		t.Errorf("failed batch unexpectedly reported diagnostic delta: %+v", resp.DiagnosticDelta)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results (first ok, second error, third skipped), got %d", len(resp.Results))
	}
	if resp.Results[0].Status != "ok" {
		t.Errorf("expected result 0 ok, got %s", resp.Results[0].Status)
	}
	if resp.Results[1].Status != "error" {
		t.Errorf("expected result 1 error, got %s", resp.Results[1].Status)
	}

	// File 1 must be updated
	c1, err := os.ReadFile(filepath.Clean(file1))
	if err != nil {
		t.Fatalf("read file1: %v", err)
	}
	if !strings.Contains(string(c1), `return "new"`) {
		t.Errorf("expected file1 to be modified by successful first edit")
	}

	// File 2 must be untouched
	c2, err := os.ReadFile(filepath.Clean(file2))
	if err != nil {
		t.Fatalf("read file2: %v", err)
	}
	if string(c2) != initialFile2 {
		t.Errorf("expected file2 to be untouched, got:\n%s", string(c2))
	}
}

func TestBatch_SameFileSequential(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "calc.go")
	initial := `package calc

func Add(a, b int) int {
	return 0
}

func Sub(a, b int) int {
	return 0
}
`
	if err := os.WriteFile(file1, []byte(initial), 0o600); err != nil {
		t.Fatalf("write calc.go: %v", err)
	}

	var out bytes.Buffer
	srv := mcp.NewServer("full", tmpDir, &out)

	edits := []mcp.BatchEntry{
		{
			Tool:   "semantic_replace_body",
			Params: json.RawMessage(`{"file":"calc.go","symbol":"Add","body":"return a + b"}`),
		},
		{
			Tool:   "semantic_replace_body",
			Params: json.RawMessage(`{"file":"calc.go","symbol":"Sub","body":"return a - b"}`),
		},
	}

	resp, err := srv.ExecuteBatch(context.Background(), edits, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	content, err := os.ReadFile(filepath.Clean(file1))
	if err != nil {
		t.Fatalf("read calc.go: %v", err)
	}

	str := string(content)
	if !strings.Contains(str, "return a + b") || !strings.Contains(str, "return a - b") {
		t.Errorf("expected both bodies replaced in calc.go, got:\n%s", str)
	}
}

func TestBatch_RenameDefersWorkspacePostProcess(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/batch\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	file := filepath.Join(tmpDir, "calc.go")
	if err := os.WriteFile(file, []byte("package calc\n\nfunc Add() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write calc.go: %v", err)
	}

	srv := mcp.NewServer("full", tmpDir, nil)
	metrics := telemetry.NewMetrics()
	resp, err := srv.ExecuteBatch(telemetry.WithMetrics(context.Background(), metrics), []mcp.BatchEntry{
		{Tool: "semantic_rename", Params: json.RawMessage(`{"file":"calc.go","symbol":"Add","to":"Sum","auto_organize_imports":true}`)},
		{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"calc.go","symbol":"Sum","body":"return 2"}`)},
	}, true)
	if err != nil || resp.Status != "ok" {
		t.Fatalf("batch rename failed: resp=%+v err=%v", resp, err)
	}
	if resp.DiagnosticDelta == nil {
		t.Fatal("successful batch omitted diagnostic delta")
	}
	if got := metrics.Snapshot(0).Phases[string(telemetry.PhaseFormattingImports)].Count; got != 1 {
		t.Fatalf("workspace import post-process count = %d, want 1", got)
	}
	content, err := os.ReadFile(file) // #nosec G304 -- test reads its own TempDir fixture.
	if err != nil {
		t.Fatalf("read calc.go: %v", err)
	}
	if !strings.Contains(string(content), "func Sum()") || !strings.Contains(string(content), "return 2") {
		t.Fatalf("rename and replacement not applied:\n%s", content)
	}
}
