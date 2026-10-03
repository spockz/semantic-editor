// Package mcp_test verifies MCP protocol server and batch execution functionality.
package mcp_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/mcp"
	"semedit/internal/telemetry"
)

func TestBatch_SuccessfulExecution(t *testing.T) {
	fixture := createSuccessfulBatchFixture(t)
	run := executeSuccessfulBatch(t, fixture.Root)
	assertSuccessfulBatchResponse(t, run.Response, run.Metrics)
	assertSuccessfulBatchSources(t, fixture)
}

type successfulBatchFixture struct {
	Root       string
	FirstFile  string
	SecondFile string
}

type successfulBatchRun struct {
	Response *mcp.BatchResponse
	Metrics  telemetry.Snapshot
}

func createSuccessfulBatchFixture(t *testing.T) successfulBatchFixture {
	t.Helper()
	root := t.TempDir()
	writeBatchFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/batch\n\ngo 1.23\n")
	first := filepath.Join(root, "file1.go")
	writeBatchFixtureFile(t, first, "package testpkg\n\nfunc Foo() string {\n\treturn \"old\"\n}\n")
	second := filepath.Join(root, "file2.go")
	writeBatchFixtureFile(t, second, "package testpkg\n\nfunc Dispatch(op string) string {\n\tswitch op {\n\tdefault:\n\t\treturn \"none\"\n\t}\n}\n")
	return successfulBatchFixture{Root: root, FirstFile: first, SecondFile: second}
}

func writeBatchFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func executeSuccessfulBatch(t *testing.T, root string) successfulBatchRun {
	t.Helper()
	edits := successfulBatchEdits()
	metrics := telemetry.NewMetrics()
	response, err := mcp.NewServer("full", root, &bytes.Buffer{}).ExecuteBatch(telemetry.WithMetrics(context.Background(), metrics), edits, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}
	return successfulBatchRun{Response: response, Metrics: metrics.Snapshot(0)}
}

func successfulBatchEdits() []mcp.BatchEntry {
	return []mcp.BatchEntry{
		{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"file1.go","symbol":"Foo","body":"return \"new\""}`)},
		{Tool: "semantic_insert_construct", Params: json.RawMessage(`{"file":"file2.go","kind":"case","in_function":"Dispatch","discriminator":"op","source":"case \"ping\":\n\treturn \"pong\""}`)},
		{Tool: "semantic_insert_function", Params: json.RawMessage(`{"file":"file1.go","source":"func Added() int { return 3 }"}`)},
		{Tool: "semantic_scaffold_file", Params: json.RawMessage(`{"file":"generated.go","package":"testpkg"}`)},
	}
}

func assertSuccessfulBatchResponse(t *testing.T, response *mcp.BatchResponse, metrics telemetry.Snapshot) {
	t.Helper()
	if response.Status != "ok" {
		t.Errorf("expected status ok, got %s", response.Status)
	}
	if len(response.Results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(response.Results))
	}
	if response.DiagnosticDelta == nil {
		t.Fatal("successful batch omitted diagnostic delta")
	}
	if !strings.Contains(response.FinalDiff, "file1.go") || !strings.Contains(response.FinalDiff, `return "new"`) || !strings.Contains(response.FinalDiff, "file2.go") {
		t.Fatalf("final workspace diff omitted batch edits: %s", response.FinalDiff)
	}
	if before, diagnostics := metrics.Phases[string(telemetry.PhaseVerificationBefore)].Count, metrics.Phases[string(telemetry.PhaseVerificationDiagnostics)].Count; before != 1 || diagnostics != 1 {
		t.Fatalf("diagnostic invocation counts = before:%d diagnostics:%d, want one baseline and one outer verification", before, diagnostics)
	}
	assertSuccessfulBatchContinuations(t, response.Results)
}

func assertSuccessfulBatchContinuations(t *testing.T, results []mcp.BatchResult) {
	t.Helper()
	for _, result := range results {
		if result.Status != "ok" {
			t.Errorf("expected all results ok: %+v", results)
			return
		}
	}
	if results[0].File != "file1.go" || results[0].Symbol != "Foo" || results[0].DiagnosticStatus != "deferred" {
		t.Errorf("replace_body batch continuation = %+v", results[0])
	}
	if results[1].File != "file2.go" || results[1].Symbol != "" || results[1].InFunction != "Dispatch" || results[1].Discriminator != "op" {
		t.Errorf("insert_case batch continuation = %+v", results[1])
	}
	if results[2].File != "file1.go" || results[2].Symbol != "Added" {
		t.Errorf("insert_function batch continuation = %+v", results[2])
	}
	if results[3].File != "generated.go" || results[3].Package != "testpkg" || results[3].Symbol != "" {
		t.Errorf("scaffold_file batch continuation = %+v", results[3])
	}
}

func assertSuccessfulBatchSources(t *testing.T, fixture successfulBatchFixture) {
	t.Helper()
	first, err := os.ReadFile(filepath.Clean(fixture.FirstFile))
	if err != nil {
		t.Fatalf("read file1: %v", err)
	}
	if !strings.Contains(string(first), `return "new"`) {
		t.Errorf("file1 not updated:\n%s", string(first))
	}
	second, err := os.ReadFile(filepath.Clean(fixture.SecondFile))
	if err != nil {
		t.Fatalf("read file2: %v", err)
	}
	if !strings.Contains(string(second), `case "ping":`) {
		t.Errorf("file2 not updated:\n%s", string(second))
	}
}

func TestExecuteBatchForWorkspaceEnforcesAndNormalizesFileScope(t *testing.T) {
	configuredRoot := t.TempDir()
	worktree := filepath.Join(configuredRoot, ".scratch", "worktrees", "selected")
	if err := os.MkdirAll(worktree, 0o700); err != nil {
		t.Fatal(err)
	}
	mainFile := filepath.Join(configuredRoot, "main.go")
	mainSource := []byte("package main\n\nfunc Value() int { return 1 }\n")
	if err := os.WriteFile(mainFile, mainSource, 0o600); err != nil {
		t.Fatal(err)
	}
	worktreeFile := filepath.Join(worktree, "value.go")
	if err := os.WriteFile(filepath.Join(worktree, "go.mod"), []byte("module example.com/worktree\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(worktreeFile, []byte("package worktree\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := mcp.NewServer("full", configuredRoot, &bytes.Buffer{})
	var response *mcp.BatchResponse
	var err error

	_, err = server.ExecuteBatchForWorkspace(context.Background(), worktree, false, []mcp.BatchEntry{{
		Tool:   "semantic_replace_body",
		Params: json.RawMessage(`{"file":"../main.go","symbol":"Value","body":"return 2"}`),
	}}, false)
	if err == nil || !strings.Contains(err.Error(), "outside workspace root") {
		t.Fatalf("outside-file batch error = %v", err)
	}
	if got, readErr := os.ReadFile(mainFile); readErr != nil || !bytes.Equal(got, mainSource) {
		t.Fatalf("rejected batch changed configured-root file: got %q, err %v", got, readErr)
	}

	_, err = server.ExecuteBatchForWorkspace(context.Background(), worktree, false, []mcp.BatchEntry{{
		Tool:   "semantic_replace_body",
		Params: json.RawMessage(`{"workspace_root":".","file":"value.go","symbol":"Value","body":"return 2"}`),
	}}, false)
	if err == nil || !strings.Contains(err.Error(), "differing from batch root") {
		t.Fatalf("differing child-root batch error = %v", err)
	}
	if got, readErr := os.ReadFile(worktreeFile); readErr != nil || string(got) != "package worktree\n\nfunc Value() int { return 1 }\n" {
		t.Fatalf("refused direct batch changed selected-worktree file: got %q, err %v", got, readErr)
	}

	response, err = server.ExecuteBatchForWorkspace(context.Background(), worktree, false, []mcp.BatchEntry{{
		Tool:   "semantic_replace_body",
		Params: json.RawMessage(fmt.Sprintf(`{"file":%q,"symbol":"Value","body":"return 2"}`, worktreeFile)),
	}}, false)
	if err != nil || response.Status != "ok" || len(response.Results) != 1 {
		t.Fatalf("contained absolute-file batch = %+v, err %v", response, err)
	}
	if response.Results[0].File != "value.go" {
		t.Fatalf("absolute file continuation = %q, want normalized worktree-relative path", response.Results[0].File)
	}
	if got, readErr := os.ReadFile(mainFile); readErr != nil || !bytes.Equal(got, mainSource) {
		t.Fatalf("valid worktree batch changed configured-root file: got %q, err %v", got, readErr)
	}
}

func TestBatchExpectedRevisionUsesCurrentWorkingOverlay(t *testing.T) {
	initial := []byte("package sample\n\nfunc Value() int {\n\treturn 1\n}\n")
	current := []byte("package sample\n\nfunc Value() int {\n\treturn 2\n}\n")
	staleDigest := sha256.Sum256(initial)
	currentDigest := sha256.Sum256(current)

	tests := []struct {
		name          string
		revision      string
		wantStatus    string
		wantBody      string
		wantErrorText string
	}{
		{name: "obsolete pre-batch revision refuses", revision: hex.EncodeToString(staleDigest[:]), wantStatus: "error", wantBody: "return 2", wantErrorText: "expected_revision does not match captured source"},
		{name: "current overlay revision succeeds", revision: hex.EncodeToString(currentDigest[:]), wantStatus: "ok", wantBody: "return 3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertBatchExpectedRevisionCase(t, initial, test.revision, test.wantStatus, test.wantBody, test.wantErrorText)
		})
	}
}

func assertBatchExpectedRevisionCase(t *testing.T, initial []byte, revision, wantStatus, wantBody, wantErrorText string) {
	t.Helper()
	root := t.TempDir()
	writeBatchFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/revisionbatch\n\ngo 1.24\n")
	path := filepath.Join(root, "main.go")
	writeBatchFixtureFile(t, path, string(initial))
	response := executeRevisionGuardBatch(t, root, revision)
	assertRevisionGuardResult(t, response, wantStatus, wantErrorText)
	assertRevisionGuardSource(t, path, wantStatus, wantBody)
}

func executeRevisionGuardBatch(t *testing.T, root, revision string) *mcp.BatchResponse {
	t.Helper()
	edits := []mcp.BatchEntry{
		{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"main.go","symbol":"Value","body":"return 2"}`)},
		{Tool: "semantic_replace_body", Params: json.RawMessage(fmt.Sprintf(`{"file":"main.go","symbol":"Value","expected_revision":%q,"body":"return 3"}`, revision))},
	}
	response, err := mcp.NewServer("full", root, &bytes.Buffer{}).ExecuteBatch(context.Background(), edits, false)
	if err != nil {
		t.Fatalf("execute batch: %v", err)
	}
	return response
}

func assertRevisionGuardResult(t *testing.T, response *mcp.BatchResponse, wantStatus, wantError string) {
	t.Helper()
	if response.Status != wantStatus {
		t.Fatalf("batch status = %q, want %q; results=%+v", response.Status, wantStatus, response.Results)
	}
	if wantError == "" {
		return
	}
	if len(response.Results) < 2 || !strings.Contains(response.Results[1].Error, wantError) {
		t.Fatalf("guard failure = %+v, want %q", response.Results, wantError)
	}
}

func assertRevisionGuardSource(t *testing.T, path string, wantStatus, wantBody string) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read published batch result: %v", err)
	}
	if !strings.Contains(string(actual), wantBody) {
		t.Fatalf("published source = %s, want %q", actual, wantBody)
	}
	if wantStatus == "error" && strings.Contains(string(actual), "return 3") {
		t.Fatalf("guarded child wrote after stale revision refusal: %s", actual)
	}
}

func TestBatchToolSuccessIncludesStructuredBatchResponse(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/batchstructured\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(tmpDir, "file.go")
	if err := os.WriteFile(file, []byte("package batchstructured\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_replace_body","params":{"file":"file.go","symbol":"Value","body":"return 2"}}]}}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", tmpDir, &out).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent struct {
				Result  mcp.BatchResponse `json:"result"`
				Metrics map[string]any    `json:"metrics"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.IsError {
		t.Fatalf("batch call failed: %#v", response.Result.Content)
	}
	if response.Result.StructuredContent.Result.Status != "ok" {
		t.Fatalf("structured batch status = %q", response.Result.StructuredContent.Result.Status)
	}
	if len(response.Result.StructuredContent.Result.Results) != 1 || response.Result.StructuredContent.Result.Results[0].Status != "ok" {
		t.Fatalf("structured batch results = %#v", response.Result.StructuredContent.Result.Results)
	}
	if response.Result.StructuredContent.Result.DiagnosticDelta == nil {
		t.Fatal("structured batch response omitted diagnostic_delta")
	}
	if !strings.Contains(response.Result.StructuredContent.Result.FinalDiff, "file.go") {
		t.Fatalf("structured batch response omitted final_diff: %+v", response.Result.StructuredContent.Result)
	}
	if response.Result.StructuredContent.Metrics != nil {
		t.Fatal("structured batch response unexpectedly includes inline metrics")
	}
	if len(response.Result.Content) != 1 || !strings.Contains(response.Result.Content[0].Text, "diagnostic_delta") {
		t.Fatalf("batch content lost human-readable response: %#v", response.Result.Content)
	}
}

func TestBatchValidationIdentifiesEditAndParamsShape(t *testing.T) {
	tmpDir := t.TempDir()
	var out bytes.Buffer
	server := mcp.NewServer("full", tmpDir, &out)
	response, err := server.ExecuteBatch(context.Background(), []mcp.BatchEntry{{
		Tool: "semantic_replace_body",
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != "error" || len(response.Results) != 1 {
		t.Fatalf("response = %#v", response)
	}
	message := response.Results[0].Error
	if !strings.Contains(message, "edit 0 params for semantic_replace_body") || !strings.Contains(message, `"params"`) || !strings.Contains(message, `"symbol":"Old"`) {
		t.Fatalf("error = %q, want indexed params-shape guidance and example", message)
	}
}

func TestBatchToolIncludesEmptyFinalDiff(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/batchemptydiff\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file.go"), []byte("package batchemptydiff\n\nfunc Value() int {\n\treturn 1\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_replace_body","params":{"file":"file.go","symbol":"Value","body":"return 1"}}]}}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", tmpDir, &out).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Result struct {
					Status    string  `json:"status"`
					FinalDiff *string `json:"final_diff"`
				} `json:"result"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	result := response.Result.StructuredContent.Result
	if response.Result.IsError || result.Status != "ok" {
		t.Fatalf("batch response failed: error=%v status=%q output=%s", response.Result.IsError, result.Status, out.String())
	}
	if result.FinalDiff == nil {
		t.Fatalf("successful no-change batch omitted final_diff: %s", out.String())
	}
	if *result.FinalDiff != "" {
		t.Fatalf("final_diff = %q, want empty diff", *result.FinalDiff)
	}
}

func TestBatch_FailFast(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/batch\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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
			Tool:   "semantic_insert_construct",
			Params: json.RawMessage(`{"file":"file2.go","kind":"case","in_function":"Bar","source":"case 1:"}`),
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
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/batch\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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

func TestBatchPostProcessFailureReturnsStructuredPartialDiff(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(goPath, filepath.Join(bin, "go")); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "gofmt-wrapper.sh")
	script := "#!/bin/sh\necho injected-gofmt-failure >&2\nexit 1\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	// The test needs an executable formatter shim to fail only during deferred processing.
	// #nosec G302
	if err := os.Chmod(wrapper, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wrapper, filepath.Join(bin, "gofmt")); err != nil {
		t.Fatal(err)
	}
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/batchfailure\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "file.go"), []byte("package batchfailure\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", bin); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("PATH", originalPath) }()
	var out bytes.Buffer
	srv := mcp.NewServer("full", module, &out)
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_replace_body","params":{"file":"file.go","symbol":"Value","body":"return 2"}}]}}}` + "\n"
	if err := srv.Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Result mcp.BatchResponse `json:"result"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	response := envelope.Result.StructuredContent.Result
	if !envelope.Result.IsError || response.Status != "error" || len(response.Results) != 2 || response.Results[1].Tool != "semantic_verify" {
		t.Fatalf("partial response = %+v; wire output: %s", envelope.Result, out.String())
	}
	if !strings.Contains(response.FinalDiff, "return 2") {
		t.Fatalf("partial final_diff omitted applied edit: %s", response.FinalDiff)
	}
	if !strings.Contains(response.Results[1].Error, "injected-gofmt-failure") {
		t.Fatalf("post-process error = %+v", response.Results[1])
	}
}

func TestBatchRejectsUnknownParamsBeforeExecutingAnyEdit(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.go")
	original := []byte("package testpkg\n\nfunc Foo() int { return 1 }\n")
	if err := os.WriteFile(file, original, 0o600); err != nil {
		t.Fatal(err)
	}
	srv := mcp.NewServer("full", root, &bytes.Buffer{})
	response, err := srv.ExecuteBatch(context.Background(), []mcp.BatchEntry{
		{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"file.go","symbol":"Foo","body":"return 2"}`)},
		{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"file.go","symbol":"Foo","body":"return 2","unexpected":true}`)},
	}, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}
	if response.Status != "error" || len(response.Results) != 1 || !strings.Contains(response.Results[0].Error, "supported keys") || !strings.Contains(response.Results[0].Error, "file") {
		t.Fatalf("batch response = %+v, want unknown parameter error with supported keys", response)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("batch changed file before rejecting invalid params:\n%s", got)
	}
}

func TestBatchMoveConstructTracksBothFilesAndNormalizesTogether(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/batchmove\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"source.go": `package batchmove

import "strings"

// Normalize trims surrounding space before the value leaves the source file.
func Normalize(value string) string { return strings.TrimSpace(value) }

// Collapse joins fields into a stable display value.
func Collapse(value string) string { return strings.Join(strings.Fields(value), " ") }

func Local() int { return 1 }
`,
		"first.go":     "package batchmove\n\nfunc First() int { return 1 }\n",
		"second.go":    "package batchmove\n\nfunc Second() int { return 2 }\n",
		"untouched.go": "//go:build go1.23\n// +build go1.23\n\n\npackage batchmove\n\nfunc Unchanged( ){}\n",
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	metrics := telemetry.NewMetrics()
	response, err := mcp.NewServer("full", root, &bytes.Buffer{}).ExecuteBatch(telemetry.WithMetrics(context.Background(), metrics), []mcp.BatchEntry{
		{Tool: "semantic_move_construct", Params: json.RawMessage(`{"file":"source.go","destination_file":"first.go","symbol":"Normalize"}`)},
		{Tool: "semantic_move_construct", Params: json.RawMessage(`{"file":"source.go","destination_file":"second.go","symbol":"Collapse"}`)},
		{Tool: "semantic_move_construct", Params: json.RawMessage(`{"file":"source.go","destination_file":"source.go","symbol":"Local","placement":"file_start"}`)},
	}, true)
	if err != nil || response.Status != "ok" {
		t.Fatalf("move batch = %+v, err = %v", response, err)
	}
	if len(response.Results) != 3 || response.Results[0].File != "first.go" || response.Results[0].OriginFile != "source.go" || response.Results[0].DestinationFile != "first.go" || response.Results[1].File != "second.go" || response.Results[1].OriginFile != "source.go" || response.Results[1].DestinationFile != "second.go" || response.Results[2].File != "source.go" || response.Results[2].OriginFile != "source.go" || response.Results[2].DestinationFile != "source.go" {
		t.Fatalf("ordered move receipts = %+v", response.Results)
	}
	for _, result := range response.Results {
		if result.Status != "ok" || result.DiagnosticStatus != "deferred" {
			t.Errorf("move result = %+v, want successful deferred receipt", result)
		}
	}
	if got := metrics.Snapshot(0).Phases[string(telemetry.PhaseFormattingImports)].Count; got != 1 {
		t.Fatalf("deferred import organization calls = %d, want one for the write set", got)
	}
	for _, name := range []string{"source.go", "first.go", "second.go"} {
		if !strings.Contains(response.FinalDiff, name) {
			t.Errorf("final_diff omitted %s: %s", name, response.FinalDiff)
		}
	}
	read := func(name string) string {
		content, readErr := os.ReadFile(filepath.Join(root, name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		return string(content)
	}
	source := read("source.go")
	first := read("first.go")
	second := read("second.go")
	if strings.Contains(source, "import \"strings\"") || strings.Contains(source, "strings.") {
		t.Fatalf("source retained moved import: %s", source)
	}
	if !strings.Contains(source, "func Local()") {
		t.Fatalf("same-file move lost its declaration: %s", source)
	}
	untouched, err := os.ReadFile(filepath.Join(root, "untouched.go"))
	if err != nil || string(untouched) != files["untouched.go"] {
		t.Fatalf("batch normalized an unchanged build header: content=%s err=%v", untouched, err)
	}
	for name, content := range map[string]string{"first.go": first, "second.go": second} {
		if !strings.Contains(content, "\"strings\"") || !strings.Contains(content, "strings.") {
			t.Errorf("%s lost the moved declaration's required import: %s", name, content)
		}
	}
	if !strings.Contains(first, "Normalize trims surrounding space") || !strings.Contains(second, "Collapse joins fields") {
		t.Fatalf("declaration comments were not preserved: first=%s second=%s", first, second)
	}
}

func TestBatchMoveConstructFailureKeepsReceiptsAndPersistedDiff(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/batchmovefailure\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"source.go": "package batchmovefailure\n\n// MoveMe stays documented.\nfunc MoveMe() int { return 3 }\n",
		"first.go":  "package batchmovefailure\n",
		"second.go": "package batchmovefailure\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	response, err := mcp.NewServer("full", root, &bytes.Buffer{}).ExecuteBatch(context.Background(), []mcp.BatchEntry{
		{Tool: "semantic_move_construct", Params: json.RawMessage(`{"file":"source.go","destination_file":"first.go","symbol":"MoveMe"}`)},
		{Tool: "semantic_move_construct", Params: json.RawMessage(`{"file":"source.go","destination_file":"second.go","symbol":"Missing"}`)},
	}, false)
	if err != nil || response.Status != "error" {
		t.Fatalf("failed move batch = %+v, err = %v", response, err)
	}
	if response.DiagnosticDelta != nil {
		t.Fatalf("failed batch reported a misleading final diagnostic delta: %+v", response.DiagnosticDelta)
	}
	if len(response.Results) != 2 || response.Results[0].Status != "ok" || response.Results[0].DiagnosticStatus != "deferred" || response.Results[1].Status != "error" {
		t.Fatalf("ordered partial move receipts = %+v", response.Results)
	}
	if failed := response.Results[1]; failed.File != "source.go" || failed.DestinationFile != "second.go" || failed.Symbol != "Missing" {
		t.Fatalf("failed move receipt lost target selectors: %+v", failed)
	}
	if !strings.Contains(response.FinalDiff, "source.go") || !strings.Contains(response.FinalDiff, "first.go") {
		t.Fatalf("final_diff does not represent the persisted first move: %s", response.FinalDiff)
	}
	second, err := os.ReadFile(filepath.Join(root, "second.go"))
	if err != nil || strings.Contains(string(second), "func Missing") || strings.Contains(string(second), "func MoveMe") {
		t.Fatalf("failed move changed its destination: content=%s err=%v", second, err)
	}
	first, err := os.ReadFile(filepath.Join(root, "first.go"))
	if err != nil || !strings.Contains(string(first), "func MoveMe") {
		t.Fatalf("first move was not persisted: content=%s err=%v", first, err)
	}
}

func TestBatchMoveConstructRejectsDestinationCollision(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/batchmovescope\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "other"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"source.go":     "package batchmovescope\n\nfunc MoveMe() int { return 3 }\n",
		"dest.go":       "package batchmovescope\n",
		"other/dest.go": "package other\nfunc MoveMe() int {return 8}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	response, err := mcp.NewServer("full", root, &bytes.Buffer{}).ExecuteBatch(context.Background(), []mcp.BatchEntry{{
		Tool:   "semantic_move_construct",
		Params: json.RawMessage(`{"file":"source.go","destination_file":"other/dest.go","symbol":"MoveMe"}`),
	}}, false)
	if err != nil || response.Status != "error" || len(response.Results) != 1 {
		t.Fatalf("cross-directory move response = %+v, err = %v", response, err)
	}
	if !strings.Contains(response.Results[0].Error, "destination already declares") || strings.Contains(response.FinalDiff, "func MoveMe") {
		t.Fatalf("cross-directory refusal was not pre-write: %+v diff=%s", response.Results[0], response.FinalDiff)
	}
	if response.Results[0].File != "source.go" || response.Results[0].DestinationFile != "other/dest.go" {
		t.Fatalf("cross-directory refusal omitted requested paths: %+v", response.Results[0])
	}
}

func TestBatchMoveConstructRejectsSymlinkFileAndDirectoryPaths(t *testing.T) {
	for _, test := range []moveSymlinkCase{
		{name: "file", source: "source-link.go", destination: "dest.go"},
		{name: "directory", source: "source.go", destination: "linked/dest.go"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertMoveSymlinkRefused(t, test)
		})
	}
}

type moveSymlinkCase struct {
	name        string
	source      string
	destination string
}

func assertMoveSymlinkRefused(t *testing.T, test moveSymlinkCase) {
	t.Helper()
	root := createMoveSymlinkWorkspace(t)
	files := snapshotMoveFiles(t, root)
	response, err := executeMoveSymlinkCase(root, test)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink move error = %v, response = %+v", err, response)
	}
	for _, file := range files {
		assertFixtureBytesUnchanged(t, file.path, file.content, file.label)
	}
}

type moveFileSnapshot struct {
	path    string
	label   string
	content []byte
}

func snapshotMoveFiles(t *testing.T, root string) []moveFileSnapshot {
	t.Helper()
	paths := []moveFileSnapshot{
		{path: filepath.Join(root, "source.go"), label: "source"},
		{path: filepath.Join(root, "dest.go"), label: "file-symlink destination"},
		{path: filepath.Join(root, "real", "dest.go"), label: "directory-symlink destination"},
	}
	for index := range paths {
		paths[index].content = readFixtureFile(t, paths[index].path)
	}
	return paths
}

func createMoveSymlinkWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for path, content := range map[string]string{
		"go.mod":       "module example.com/batchmovesymlink\n\ngo 1.23\n",
		"source.go":    "package batchmovesymlink\n\nfunc MoveMe() int { return 3 }\n",
		"dest.go":      "package batchmovesymlink\n",
		"real/dest.go": "package batchmovesymlink\n",
	} {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "source.go"), filepath.Join(root, "source-link.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	return root
}

func executeMoveSymlinkCase(root string, test moveSymlinkCase) (*mcp.BatchResponse, error) {
	params, err := json.Marshal(map[string]string{
		"file":             test.source,
		"destination_file": test.destination,
		"symbol":           "MoveMe",
	})
	if err != nil {
		return nil, fmt.Errorf("encode symlink move test parameters: %w", err)
	}
	return mcp.NewServer("full", root, &bytes.Buffer{}).ExecuteBatch(context.Background(), []mcp.BatchEntry{{
		Tool:   "semantic_move_construct",
		Params: params,
	}}, false)
}

func assertFixtureBytesUnchanged(t *testing.T, path string, before []byte, label string) {
	t.Helper()
	if after := readFixtureFile(t, path); !bytes.Equal(before, after) {
		t.Fatalf("symlink refusal changed %s: %q", label, after)
	}
}

func readFixtureFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
