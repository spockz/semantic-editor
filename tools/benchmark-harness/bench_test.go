// Package main verifies the benchmark task parsing, extraction, and oracle behavior.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCodexToolCallOutcomes(t *testing.T) {
	t.Parallel()

	lines := []string{
		`{"type":"item.started","item":{"id":"verify","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","status":"in_progress"}}`,
		`{"type":"item.completed","item":{"id":"verify","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","status":"failed","result":{"content":[{"type":"text","text":"language could not be detected"}]},"error":null}}`,
		`{"type":"item.completed","item":{"id":"rename","type":"mcp_tool_call","server":"semedit","tool":"semantic_rename","status":"completed","result":{"content":[{"type":"text","text":"renamed"}],"structuredContent":{"metrics":{"schema_version":1,"total_ms":1250,"phases":{"verification.before_diagnostics":{"count":1,"duration_ms":700}}}}},"error":null}}`,
		`{"type":"item.completed","item":{"id":"unavailable","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","status":"failed","result":null,"error":{"message":"MCP server unavailable"}}}`,
	}

	res := &RunResult{}
	tracker := codexToolTracker{}
	for _, line := range lines {
		var event CodexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("unmarshal Codex event: %v", err)
		}
		tracker.observe(res, event.Item)
	}

	if len(res.ToolCalls) != 3 {
		t.Fatalf("tool calls = %d, want 3", len(res.ToolCalls))
	}
	if got := res.ToolCalls[0]; got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusFailed || !strings.Contains(got.Failure, "language could not be detected") {
		t.Errorf("semantic_verify outcome = %#v, want transport success and functional failure", got)
	}
	if got := res.ToolCalls[1]; got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusSucceeded {
		t.Errorf("semantic_rename outcome = %#v, want full success", got)
	}
	if got := res.ToolCalls[1].MCPMetrics; got == nil || got.TotalMS != 1250 || got.Phases["verification.before_diagnostics"].DurationMS != 700 {
		t.Errorf("semantic_rename MCP metrics = %#v, want structured server timing", got)
	}
	if got := res.ToolCalls[2]; got.TransportStatus != ToolCallStatusFailed || got.FunctionalStatus != ToolCallStatusUnknown || !strings.Contains(got.Failure, "unavailable") {
		t.Errorf("unavailable semantic_verify outcome = %#v, want transport failure and unknown function result", got)
	}
	if !res.MCPVerified {
		t.Error("MCPVerified = false, want true after a semedit transport success")
	}
}

func TestRunCodexCapturesBoundedDiagnostics(t *testing.T) {
	cases := []struct {
		name, stderr, stdout string
		exit, wantStatus     int
		wantErr              bool
	}{
		{name: "success with stderr", stderr: "MCP startup warning\n", stdout: "not-json\n"},
		{name: "success empty stderr", stdout: "not-json\n"},
		{name: "oversized stderr", stderr: strings.Repeat("x", maxCodexStderrBytes+100), stdout: "not-json\n"},
		{name: "failure retains stderr", stderr: "server failed to start", exit: 7, wantErr: true, wantStatus: 7},
		{name: "malformed stdout", stderr: "diagnostic", stdout: "not-json\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			script := filepath.Join(binDir, "codex")
			content := fmt.Sprintf("#!/bin/sh\nprintf '%%s' %q >&2\nprintf '%%s' %q\nexit %d\n", tc.stderr, tc.stdout, tc.exit)
			if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			// #nosec G302 -- the temporary test command must be executable.
			if err := os.Chmod(script, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			res := &RunResult{}
			err := NewRunner(t.TempDir()).runCodex(context.Background(), t.TempDir(), Target{}, "prompt", res)
			if (err != nil) != tc.wantErr {
				t.Fatalf("runCodex error = %v, want error=%t", err, tc.wantErr)
			}
			if res.CodexExitCode == nil || *res.CodexExitCode != tc.wantStatus {
				t.Fatalf("exit status = %v, want %d", res.CodexExitCode, tc.wantStatus)
			}
			if tc.stderr != "" && !strings.Contains(res.CodexStderr, strings.TrimSpace(tc.stderr)[:min(len(strings.TrimSpace(tc.stderr)), 20)]) {
				t.Fatalf("stderr = %q, missing diagnostic", res.CodexStderr)
			}
			if len(res.CodexStderr) > maxCodexStderrBytes+64 {
				t.Fatalf("stderr retained %d bytes, expected bounded output", len(res.CodexStderr))
			}
			if tc.name == "oversized stderr" {
				if !strings.Contains(res.CodexStderr, "[stderr truncated; retained first 65536 bytes]") {
					t.Fatal("oversized stderr missing truncation marker")
				}
				if !strings.HasPrefix(res.CodexStderr, strings.Repeat("x", maxCodexStderrBytes)) {
					t.Fatal("oversized stderr payload was not retained up to the boundary")
				}
			}
			if tc.name == "malformed stdout" && res.Turns != 1 {
				t.Fatalf("malformed stdout changed turns to %d", res.Turns)
			}
			if tc.name == "malformed stdout" && (res.ToolCount != 0 || res.OutputTokens != 0 || res.ReasoningTokens != 0 || res.PromptTokens != 0) {
				t.Fatalf("malformed stdout contaminated metrics: %#v", res)
			}
			if tc.name == "success with stderr" {
				data, marshalErr := json.Marshal(res)
				if marshalErr != nil || !strings.Contains(string(data), `"codex_exit_code":0`) {
					t.Fatalf("successful exit status was not serialized: %s (%v)", data, marshalErr)
				}
			}
			if tc.name == "failure retains stderr" {
				data, marshalErr := json.Marshal(res)
				if marshalErr != nil || !strings.Contains(string(data), `"codex_exit_code":7`) {
					t.Fatalf("failed exit status was not serialized: %s (%v)", data, marshalErr)
				}
			}
		})
	}
}

func TestAgyToolCallOutcomes(t *testing.T) {
	t.Parallel()

	transcript := []byte(`{"type":"PLANNER_RESPONSE","status":"DONE","tool_calls":[{"name":"call_mcp_tool","args":{"ServerName":"\"semedit\"","ToolName":"\"semantic_rename\""}}]}
{"type":"GENERIC","status":"DONE","content":"rename result"}
{"type":"PLANNER_RESPONSE","status":"DONE","tool_calls":[{"name":"call_mcp_tool","args":{"ServerName":"\"semedit\"","ToolName":"\"semantic_verify\""}}]}
{"type":"GENERIC","status":"FAILED","content":"server rejected the request"}`)

	res := &RunResult{}
	parseAgyTranscript(transcript, res)

	if got, want := res.ToolCount, 2; got != want {
		t.Fatalf("tool count = %d, want %d", got, want)
	}
	if got := res.ToolCalls[0]; got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusSucceeded {
		t.Errorf("semantic_rename outcome = %#v, want full success", got)
	}
	if got := res.ToolCalls[1]; got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusFailed || !strings.Contains(got.Failure, "rejected") {
		t.Errorf("semantic_verify outcome = %#v, want transport success and functional failure", got)
	}
	if !res.MCPVerified {
		t.Error("MCPVerified = false, want true after an Agy semedit transport success")
	}
}

func TestMCPVerificationRequiresSemanticToolIdentity(t *testing.T) {
	t.Parallel()

	res := &RunResult{ToolCalls: []ToolCall{{
		Name:            `/bin/zsh -lc "sed -n '1,80p' ../run_codex_semedit/main.go"`,
		TransportStatus: ToolCallStatusSucceeded,
	}}}
	refreshMCPVerified(res)
	if res.MCPVerified {
		t.Fatal("MCPVerified = true for a shell command that only mentions semedit")
	}
}

func TestMCPServerInstructionModeAndCodexOverride(t *testing.T) {
	t.Parallel()

	if got, err := ParseMCPServerInstructionMode("descriptive"); err != nil || got != MCPServerInstructionsDescriptive {
		t.Fatalf("parse descriptive mode = %q, %v", got, err)
	}
	if got, err := ParseMCPServerInstructionMode("DIRECTIVE"); err != nil || got != MCPServerInstructionsPrescriptive {
		t.Fatalf("parse legacy directive mode = %q, %v", got, err)
	}
	if _, err := ParseMCPServerInstructionMode("unknown"); err == nil {
		t.Fatal("unknown instruction mode unexpectedly parsed")
	}

	for mode, want := range map[MCPServerInstructionMode]string{
		MCPServerInstructionsDescriptive:  "Semedit semantic tools are available",
		MCPServerInstructionsPrescriptive: "inspect the complete tool inventory",
	} {
		runner := NewRunner(t.TempDir(), WithMCPServerInstructions(mode))
		override, err := runner.codexMCPServerInstructionsOverride()
		if err != nil {
			t.Fatalf("build %s override: %v", mode, err)
		}
		for _, part := range []string{"mcp_servers.semedit.args=", "--instructions", want} {
			if !strings.Contains(override, part) {
				t.Errorf("%s Codex MCP override %q missing %q", mode, override, part)
			}
		}
	}
}

func TestProvenanceSetSnapshotsExecutionContext(t *testing.T) {
	t.Parallel()

	var provenance ProvenanceSet
	if err := provenance.Set("parser-limit=16MiB"); err != nil {
		t.Fatal(err)
	}
	if err := provenance.Set("benchmark-suite=semantic-routing"); err != nil {
		t.Fatal(err)
	}
	if got, want := provenance.String(), "benchmark-suite=semantic-routing,parser-limit=16MiB"; got != want {
		t.Errorf("provenance string = %q, want %q", got, want)
	}
	if err := provenance.Set("parser-limit=32MiB"); err == nil {
		t.Fatal("conflicting provenance unexpectedly accepted")
	}

	runner := NewRunner(t.TempDir(), WithMCPServerInstructions(MCPServerInstructionsPrescriptive), WithProvenance(provenance))
	if got, want := runner.provenanceFor()["parser-limit"], "16MiB"; got != want {
		t.Errorf("run provenance = %q, want %q", got, want)
	}
	if got := runner.provenanceFor()["mcp-server-instructions"]; got != "" {
		t.Errorf("MCP instruction mode leaked into provenance as %q", got)
	}
}

func TestParseAllBenchmarkFixtures(t *testing.T) {
	benchDir := filepath.Join("..", "..", "testdata", "bench")
	entries, err := os.ReadDir(benchDir)
	if err != nil {
		t.Fatalf("read bench dir: %v", err)
	}

	expectedTasks := 11
	foundTasks := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txtar" {
			continue
		}

		foundTasks++
		cleanPath := filepath.Clean(filepath.Join(benchDir, entry.Name()))
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			t.Fatalf("read fixture %s: %v", entry.Name(), err)
		}

		task, err := ParseTask(data)
		if err != nil {
			t.Fatalf("parse fixture %s: %v", entry.Name(), err)
		}

		if task.Metadata.TaskID == "" {
			t.Errorf("fixture %s missing task_id", entry.Name())
		}
		if task.Metadata.Instruction == "" {
			t.Errorf("fixture %s missing instruction", entry.Name())
		}
		if task.Metadata.Category == "" {
			t.Errorf("fixture %s missing category", entry.Name())
		}

		// Verify no want/ files inside archive (crucial invariant)
		for _, file := range task.Archive.Files {
			normalized := filepath.ToSlash(file.Name)
			if strings.HasPrefix(normalized, "want/") {
				t.Errorf("fixture %s contains golden want/ file %s: violates anti-leakage invariant", entry.Name(), file.Name)
			}
		}

		// Verify extraction to temp dir
		tmpDir := t.TempDir()
		if err := task.ExtractTo(tmpDir); err != nil {
			t.Fatalf("extract task %s: %v", entry.Name(), err)
		}

		// Verify go.mod was extracted
		if _, err := os.Stat(filepath.Join(tmpDir, "go.mod")); err != nil {
			t.Errorf("task %s failed to extract go.mod: %v", entry.Name(), err)
		}
	}

	if foundTasks < expectedTasks {
		t.Errorf("expected at least %d benchmark tasks, found %d", expectedTasks, foundTasks)
	}
}

func TestOracleMutationPolicyEnforcement(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_01_rename_local.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse task: %v", err)
	}

	tmpDir := t.TempDir()
	if err := task.ExtractTo(tmpDir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Case 1: Unauthorized change to go.mod
	res, err := Evaluate(context.Background(), task, tmpDir, []string{"go.mod"})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if res.Passed {
		t.Errorf("expected failure on disallowed go.mod change")
	}
	if res.FailureStage != "level_1_mutation_policy" {
		t.Errorf("expected level_1_mutation_policy failure, got %s", res.FailureStage)
	}
}

func TestOracleAdversarialCheating(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_01_rename_local.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse task: %v", err)
	}

	tmpDir := t.TempDir()
	if err := task.ExtractTo(tmpDir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Unmodified initial fixture fails Level 2 AST (newName missing and oldName present)
	res, err := Evaluate(context.Background(), task, tmpDir, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if res.Passed {
		t.Errorf("expected unmodified fixture to fail oracle")
	}
	if res.FailureStage != "level_2_ast" {
		t.Errorf("expected level_2_ast failure, got %s", res.FailureStage)
	}
}

func TestWorkspaceSnapshotTracksAllChangedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "changed.go"), []byte("before"), 0o600); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "deleted.go"), []byte("before"), 0o600); err != nil {
		t.Fatalf("write deleted file: %v", err)
	}

	before, err := snapshotWorkspaceFiles(root)
	if err != nil {
		t.Fatalf("snapshot workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "changed.go"), []byte("after"), 0o600); err != nil {
		t.Fatalf("modify changed file: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "deleted.go")); err != nil {
		t.Fatalf("remove deleted file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "created.go"), []byte("after"), 0o600); err != nil {
		t.Fatalf("write created file: %v", err)
	}

	got, err := changedWorkspaceFiles(root, before)
	if err != nil {
		t.Fatalf("collect workspace changes: %v", err)
	}
	if want := []string{"changed.go", "created.go", "deleted.go"}; !slices.Equal(got, want) {
		t.Errorf("changed files = %v, want %v", got, want)
	}
}

func TestInstallHiddenTestsCopiesOnlyAfterAgentExecution(t *testing.T) {
	root := t.TempDir()
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o750); err != nil {
		t.Fatalf("create hidden test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "acceptance_test.go"), []byte("package nested\n"), 0o600); err != nil {
		t.Fatalf("write hidden test: %v", err)
	}
	if err := installHiddenTests(root, source); err != nil {
		t.Fatalf("install hidden tests: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "nested", "acceptance_test.go")); err != nil {
		t.Fatalf("installed hidden test missing: %v", err)
	}
}

func TestInstallHiddenTestsRejectsDestinationSymlink(t *testing.T) {
	workDir := t.TempDir()
	sourceDir := t.TempDir()
	escapedDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sourceDir, "nested"), 0o750); err != nil {
		t.Fatalf("create hidden test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "nested", "acceptance_test.go"), []byte("package nested\n"), 0o600); err != nil {
		t.Fatalf("write hidden test: %v", err)
	}
	if err := os.Symlink(escapedDir, filepath.Join(workDir, "nested")); err != nil {
		t.Fatalf("create workspace symlink: %v", err)
	}

	err := installHiddenTests(workDir, sourceDir)
	if err == nil {
		t.Fatal("install hidden tests unexpectedly followed workspace symlink")
	}
	if _, statErr := os.Stat(filepath.Join(escapedDir, "acceptance_test.go")); !os.IsNotExist(statErr) {
		t.Errorf("hidden test escaped workspace, stat error = %v", statErr)
	}
}

func TestTask11MixedSinkFixtureExtractsCompilingSmallAndLargeWorkspaces(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_11_mixed_sink_api_migration.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if got, want := task.Metadata.PromptVariants["default"], task.Metadata.Instruction; got != want {
		t.Errorf("default prompt = %q, want %q", got, want)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	for _, variant := range []string{"small", "large"} {
		workDir := t.TempDir()
		if err := task.ExtractVariantTo(workDir, variant); err != nil {
			t.Fatalf("extract %s fixture: %v", variant, err)
		}
		cmd := exec.CommandContext(ctx, "go", "build", "./...")
		cmd.Dir = workDir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s fixture: %v\n%s", variant, err, output)
		}
		cmd = exec.CommandContext(ctx, "go", "test", "./...")
		cmd.Dir = workDir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("visible tests %s fixture: %v\n%s", variant, err, output)
		}
		if _, err := os.Stat(filepath.Join(workDir, "audit", "normalize_hidden_test.go")); !os.IsNotExist(err) {
			t.Errorf("%s fixture unexpectedly contains hidden acceptance test: %v", variant, err)
		}

		_, err := os.Stat(filepath.Join(workDir, "audit", "fanout.go"))
		if variant == "large" && err != nil {
			t.Errorf("large fixture missing related audit overlay: %v", err)
		}
		if variant == "small" && !os.IsNotExist(err) {
			t.Errorf("small fixture unexpectedly contains related audit overlay: %v", err)
		}
	}
}

func TestPromptVariantsParsing(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "scripts", "generate_template_main.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse task: %v", err)
	}

	if len(task.Metadata.PromptVariants) != 2 {
		t.Fatalf("expected 2 prompt variants, got %d (%v)", len(task.Metadata.PromptVariants), task.Metadata.PromptVariants)
	}
	if !strings.Contains(task.Metadata.PromptVariants["crypto_rand"], "crypto/rand") {
		t.Errorf("expected crypto/rand in crypto_rand variant, got: %s", task.Metadata.PromptVariants["crypto_rand"])
	}
}

func TestMatrixPromptVariantsRequiresDeclaredPrompts(t *testing.T) {
	t.Parallel()

	withoutPrompts := &Task{Metadata: TaskMetadata{Instruction: "Metadata alone is not runnable."}}
	if got := matrixPromptVariants(withoutPrompts, "small"); got != nil {
		t.Fatalf("expected no runnable variants without prompt_variants, got %v", got)
	}

	withPrompts := &Task{Metadata: TaskMetadata{PromptVariants: map[string]string{
		"default":      "Rename the symbol.",
		"empty-prompt": " ",
	}}}
	if got, want := matrixPromptVariants(withPrompts, "small"), []string{"small:default"}; !slices.Equal(got, want) {
		t.Fatalf("unexpected expanded prompt variants: got %v, want %v", got, want)
	}
	if got, want := matrixPromptVariants(withPrompts, "large:default"), []string{"large:default"}; !slices.Equal(got, want) {
		t.Fatalf("unexpected explicit prompt variant: got %v, want %v", got, want)
	}
	if got := matrixPromptVariants(withPrompts, "large:missing"); got != nil {
		t.Fatalf("expected undeclared prompt variant to be skipped, got %v", got)
	}
}

func TestCollectAvailableBenchmarks(t *testing.T) {
	benchDir := filepath.Clean(filepath.Join("..", "..", "testdata", "bench"))
	benchmarks, err := CollectAvailableBenchmarks(benchDir)
	if err != nil {
		t.Fatalf("CollectAvailableBenchmarks failed: %v", err)
	}

	if len(benchmarks) < 10 {
		t.Fatalf("expected at least 10 benchmarks, got %d", len(benchmarks))
	}

	foundGenerateTemplate := false
	foundMixedSinkMigration := false
	for _, b := range benchmarks {
		if b.TaskID == "task-07-generate-template-main" {
			foundGenerateTemplate = true
			if len(b.PromptVariants) != 2 {
				t.Errorf("expected 2 prompt variants for task-07-generate-template-main, got %d", len(b.PromptVariants))
			}
		}
		if b.TaskID == "task-11-mixed-sink-api-migration" {
			foundMixedSinkMigration = true
			if got, want := b.PromptVariants, []string{"default"}; !slices.Equal(got, want) {
				t.Errorf("unexpected task-11 prompt variants: got %v, want %v", got, want)
			}
		}
	}
	if !foundGenerateTemplate {
		t.Errorf("expected task-07-generate-template-main in collected benchmarks")
	}
	if !foundMixedSinkMigration {
		t.Error("expected task-11-mixed-sink-api-migration in collected benchmarks")
	}

	var buf strings.Builder
	PrintBenchmarksList(&buf, benchmarks)
	output := buf.String()
	if !strings.Contains(output, "task-01-rename-local") {
		t.Errorf("expected task-01-rename-local in formatted output")
	}
}
