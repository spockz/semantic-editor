// This file provides an opt-in live check that successive Codex turns retain one semedit MCP instance.

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"semedit/internal/pipeline"
	"strings"
	"testing"
	"time"
)

func TestLiveCodexAppServerKeepsMCPAcrossHints(t *testing.T) {
	if os.Getenv("SEMEDIT_LIVE_CODEX_CONTINUITY") != "1" {
		t.Skip("requires explicit live provider validation")
	}
	root, rootErr := filepath.Abs(filepath.Join("..", ".."))
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	base := filepath.Join(root, ".scratch", "codex-continuity-live")
	if err := os.MkdirAll(base, 0o750); err != nil {
		t.Fatal(err)
	}
	workDir, createErr := os.MkdirTemp(base, "fixture-")
	if createErr != nil {
		t.Fatal(createErr)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(workDir); err != nil {
			t.Error(err)
		}
	})
	for name, content := range map[string]string{
		"go.mod":  "module example.com/continuity\n\ngo 1.23\n",
		"main.go": "package main\n\nfunc Existing() {}\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(workDir, name), []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	provider := &codexProvider{runner: NewRunner(base)}
	t.Cleanup(func() {
		if err := provider.close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	target := Target{Harness: string(HarnessCodex), Model: "gpt-6-luna", Effort: "medium"}
	threadID := ""
	for index, prompt := range []string{
		"Call semedit semantic_rename once to rename function Existing in main.go to Renamed. Do not run shell commands or call other tools. Then answer done.",
		"Follow-up: call semedit semantic_rename once to rename function Renamed in main.go back to Existing. Do not run shell commands or call other tools. Then answer done.",
	} {
		result := &RunResult{Target: target, Arm: ArmSemeditAgentic}
		id, runErr := provider.run(ctx, workDir, target, prompt, result, threadID)
		if runErr != nil {
			t.Fatalf("turn %d: %v; stderr: %s", index+1, runErr, result.CodexStderr)
		}
		if id == "" || (threadID != "" && threadID != id) {
			t.Fatalf("thread changed: %q to %q", threadID, id)
		}
		threadID = id
		if !result.MCPVerified || result.toolObservationState != ToolObservationComplete {
			t.Fatalf("turn %d missing complete MCP evidence: verified=%t observation=%s tools=%v", index+1, result.MCPVerified, result.toolObservationState, result.ToolCalls)
		}
		t.Logf("turn %d: thread=%s tools=%d tokens=%d", index+1, id, result.ToolCount, result.PromptTokens+result.OutputTokens)
	}
	if err := provider.close(); err != nil {
		t.Fatal(err)
	}
	assertLiveCodexMCPPersistence(t, workDir)
}

func assertLiveCodexMCPPersistence(t *testing.T, workDir string) {
	t.Helper()
	source, readErr := os.ReadFile(filepath.Join(workDir, "main.go"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(source), "func Existing()") || strings.Contains(string(source), "Renamed") {
		t.Fatalf("successive semantic renames did not round trip: %s", source)
	}
	data, metricsErr := os.ReadFile(benchmarkMCPMetricsPath(workDir))
	if metricsErr != nil {
		t.Fatal(metricsErr)
	}
	var records []struct {
		Sequence       int             `json:"sequence"`
		SessionMetrics json.RawMessage `json:"session_metrics"`
	}
	if decodeErr := json.Unmarshal(data, &records); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(records) < 2 {
		t.Fatalf("semantic calls=%d, want at least two", len(records))
	}
	startups, previous := 0, 0
	for _, record := range records {
		if record.Sequence <= previous {
			t.Fatalf("MCP sequence restarted: %d after %d", record.Sequence, previous)
		}
		previous = record.Sequence
		if hasJSONValue(record.SessionMetrics) {
			startups++
		}
	}
	if startups != 1 {
		t.Fatalf("MCP startups=%d, want exactly one over both turns", startups)
	}
	t.Logf("one MCP startup across %d semantic calls", len(records))
}
