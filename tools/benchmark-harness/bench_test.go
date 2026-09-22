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

	"semedit/internal/mcp"
)

func TestCodexToolCallOutcomes(t *testing.T) {
	t.Parallel()

	lines := []string{
		`{"type":"item.started","item":{"id":"verify","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","arguments":{"file":"api/server.go"},"status":"in_progress"}}`,
		`{"type":"item.completed","item":{"id":"verify","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","status":"failed","result":{"content":[{"type":"text","text":"language could not be detected"}]},"error":null}}`,
		`{"type":"item.completed","item":{"id":"rename","type":"mcp_tool_call","server":"semedit","tool":"semantic_rename","status":"completed","result":{"content":[{"type":"text","text":"renamed"}],"structuredContent":{"metrics":{"schema_version":1,"total_ms":1250,"phases":{"verification.before_diagnostics":{"count":1,"duration_ms":700}}},"session_metrics":{"server_start_to_initialize_ms":125,"initialize_to_first_semantic_call_ms":875}}},"error":null}}`,
		`{"type":"item.completed","item":{"id":"unavailable","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","status":"failed","result":null,"error":{"message":"MCP server unavailable"}}}`,
	}

	res := &RunResult{}
	tracker := codexToolTracker{}
	for _, line := range lines {
		var event CodexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("unmarshal Codex event: %v", err)
		}
		tracker.observe(res, event.Item, time.Now())
	}

	if len(res.ToolCalls) != 3 {
		t.Fatalf("tool calls = %d, want 3", len(res.ToolCalls))
	}
	if got := res.ToolCalls[0]; got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusFailed || !strings.Contains(got.Failure, "language could not be detected") {
		t.Errorf("semantic_verify outcome = %#v, want transport success and functional failure", got)
	}
	if got, want := string(res.ToolCalls[0].Arguments), `{"file":"api/server.go"}`; got != want {
		t.Errorf("semantic_verify arguments = %s, want %s", got, want)
	}
	if got := res.ToolCalls[1]; got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusSucceeded {
		t.Errorf("semantic_rename outcome = %#v, want full success", got)
	}
	if got := res.ToolCalls[1].MCPMetrics; got == nil || got.TotalMS != 1250 || got.Phases["verification.before_diagnostics"].DurationMS != 700 {
		t.Errorf("semantic_rename MCP metrics = %#v, want structured server timing", got)
	}
	if got := res.MCPInitializeToFirstSemanticCall; got == nil || *got != 875*time.Millisecond {
		t.Errorf("MCP initialize-to-first-semantic timing = %v, want 875ms", got)
	}
	if got := res.ToolCalls[2]; got.TransportStatus != ToolCallStatusFailed || got.FunctionalStatus != ToolCallStatusUnknown || !strings.Contains(got.Failure, "unavailable") {
		t.Errorf("unavailable semantic_verify outcome = %#v, want transport failure and unknown function result", got)
	}
	if !res.MCPVerified {
		t.Error("MCPVerified = false, want true after a semedit transport success")
	}
}

func TestRunCodexCapturesBoundedDiagnostics(t *testing.T) {
	const threadStarted = "{\"type\":\"thread.started\",\"thread_id\":\"thread-123\"}\\n"
	cases := []struct {
		name, stderr, stdout string
		exit, wantStatus     int
		wantErr              bool
	}{
		{name: "success with stderr", stderr: "MCP startup warning\n", stdout: threadStarted},
		{name: "success empty stderr", stdout: threadStarted},
		{name: "oversized stderr", stderr: strings.Repeat("x", maxCodexStderrBytes+100), stdout: threadStarted},
		{name: "failure retains stderr", stderr: "server failed to start", stdout: threadStarted, exit: 7, wantErr: true, wantStatus: 7},
		{name: "malformed stdout", stderr: "diagnostic", stdout: "not-json\n" + threadStarted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			script := filepath.Join(binDir, "codex")
			content := fmt.Sprintf("#!/bin/sh\nprintf '%%b' %q >&2\nprintf '%%b' %q\nexit %d\n", tc.stderr, tc.stdout, tc.exit)
			if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			// #nosec G302 -- the temporary test command must be executable.
			if err := os.Chmod(script, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			res := &RunResult{Arm: ArmBaseline}
			_, err := NewRunner(t.TempDir()).runCodex(context.Background(), t.TempDir(), Target{}, "prompt", res, "")
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
			if tc.name == "malformed stdout" && (res.ToolCount != 0 || res.OutputTokens != 0 || res.ReasoningTokens != 0 || res.PromptTokens != 0 || res.agentResponse != "") {
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

func TestRunOpenCodeCapturesToolOutcomeAndSession(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "test-key")
	binDir := t.TempDir()
	script := filepath.Join(binDir, "opencode")
	stdout := strings.Join([]string{
		`{"type":"session.created","properties":{"info":{"id":"ses-123"}}}`,
		`{"type":"message.part.updated","properties":{"part":{"id":"tool-1","type":"tool","tool":"semantic_rename","server":"semedit","state":{"status":"running","input":{"symbol":"old"}}}}}`,
		`{"type":"message.part.updated","properties":{"part":{"id":"tool-1","type":"tool","tool":"semantic_rename","server":"semedit","state":{"status":"completed","input":{"symbol":"old"},"output":{"structuredContent":{"metrics":{"schema_version":1,"total_ms":17}}}}}}}`,
		`{"type":"message.part.updated","properties":{"part":{"type":"text","text":"DONE"}}}`,
	}, "\n") + "\n"
	content := fmt.Sprintf("#!/bin/sh\nprintf '%%b' %q\n", stdout)
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- the temporary test command must be executable.
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	workDir := t.TempDir()
	res := &RunResult{Arm: ArmSemedit}
	sessionID, err := NewRunner(t.TempDir()).runOpenCode(t.Context(), workDir, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free"}, "prompt", res, "")
	if err != nil {
		t.Fatal(err)
	}
	if sessionID != "ses-123" {
		t.Fatalf("session ID = %q, want ses-123", sessionID)
	}
	if res.OpenCodeExitCode == nil || *res.OpenCodeExitCode != 0 {
		t.Fatalf("OpenCode exit status = %v, want 0", res.OpenCodeExitCode)
	}
	if got := len(res.ToolCalls); got != 1 {
		t.Fatalf("tool calls = %d, want 1", got)
	}
	call := res.ToolCalls[0]
	if call.TransportStatus != ToolCallStatusSucceeded || call.FunctionalStatus != ToolCallStatusSucceeded {
		t.Errorf("tool outcome = %#v, want full success", call)
	}
	if call.MCPMetrics == nil || call.MCPMetrics.TotalMS != 17 {
		t.Errorf("MCP metrics = %#v, want total_ms=17", call.MCPMetrics)
	}
	if !res.MCPVerified {
		t.Error("MCPVerified = false, want true")
	}
	if got := res.agentResponse; got != "DONE" {
		t.Errorf("agent response = %q, want DONE", got)
	}
}

func TestOpenCodeConfigurationIsFixtureScoped(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "secret-test-key")
	workDir := t.TempDir()
	runner := NewRunner(filepath.Join(t.TempDir(), "benchmarks"), WithMCPServerInstructions(MCPServerInstructionsPrescriptive))
	configPath, env, err := runner.openCodeEnvironment(t.Context(), workDir, ArmSemedit, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free"})
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- config path is constructed by the harness inside the temporary fixture.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Provider map[string]struct {
			Options map[string]string `json:"options"`
		} `json:"provider"`
		MCP struct {
			Servers map[string]struct {
				Command     []string          `json:"command"`
				CWD         string            `json:"cwd"`
				Environment map[string]string `json:"environment"`
				CodeMode    bool              `json:"codemode"`
			} `json:"servers"`
		} `json:"mcp"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if got := config.Provider["openrouter"].Options["baseURL"]; got != openRouterBaseURL {
		t.Errorf("OpenRouter base URL = %q, want %q", got, openRouterBaseURL)
	}
	if got := config.Provider["openrouter"].Options["apiKey"]; got != "{env:"+openRouterAPIKeyEnv+"}" {
		t.Errorf("OpenRouter API key config = %q, want environment reference", got)
	}
	if strings.Contains(string(data), "secret-test-key") {
		t.Fatal("OpenRouter credential was serialized into OpenCode configuration")
	}
	server, found := config.MCP.Servers["semedit"]
	if !found {
		t.Fatal("semedit MCP server is absent")
	}
	if got, want := server.Command[len(server.Command)-2:], []string{"--instructions", mcp.PrescriptiveInstructions}; !slices.Equal(got, want) {
		t.Errorf("MCP command suffix = %#v, want %#v", got, want)
	}
	if server.CWD != workDir || server.CodeMode {
		t.Errorf("MCP server = %#v, want fixture cwd and direct tool exposure", server)
	}
	if got := server.Environment["GOCACHE"]; got != filepath.Join(workDir, ".scratch", "go", "build") {
		t.Errorf("MCP GOCACHE = %q, want fixture-local cache", got)
	}
	if got := environmentValue(env, "HOME"); got == "" || !strings.HasPrefix(got, filepath.Join(workDir, ".scratch", "opencode")) {
		t.Errorf("OpenCode HOME = %q, want fixture-local state", got)
	}
	if got := environmentValue(env, "OPENCODE_DISABLE_PROJECT_CONFIG"); got != "1" {
		t.Errorf("OPENCODE_DISABLE_PROJECT_CONFIG = %q, want 1", got)
	}
	if got := environmentValue(env, "OPENCODE_CONFIG"); got != configPath {
		t.Errorf("OPENCODE_CONFIG = %q, want %q", got, configPath)
	}

	baselinePath, _, err := runner.openCodeEnvironment(t.Context(), t.TempDir(), ArmBaseline, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free"})
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- config path is constructed by the harness inside the temporary fixture.
	baseline, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(baseline), `"mcp"`) {
		t.Errorf("baseline config unexpectedly enables MCP: %s", baseline)
	}
}

func TestSemanticToolReflectionOnlyFollowsUnverifiedSemanticRuns(t *testing.T) {
	t.Parallel()

	if !shouldRequestSemanticToolReflection(ArmSemedit, "thread-123", &RunResult{}) {
		t.Fatal("unverified semantic run should request a reflection")
	}
	for _, test := range []struct {
		name      string
		arm       ArmType
		sessionID string
		result    *RunResult
	}{
		{name: "baseline", arm: ArmBaseline, sessionID: "thread-123", result: &RunResult{}},
		{name: "no session", arm: ArmSemedit, result: &RunResult{}},
		{name: "semantic confirmed", arm: ArmSemedit, sessionID: "thread-123", result: &RunResult{MCPVerified: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if shouldRequestSemanticToolReflection(test.arm, test.sessionID, test.result) {
				t.Fatal("unexpected semantic-tool reflection")
			}
		})
	}
	for _, expected := range []string{"do not make further file changes", "do not run tools", "semantic_*", "Do not retry"} {
		if !strings.Contains(semanticToolReflectionPrompt, expected) {
			t.Errorf("reflection prompt missing %q", expected)
		}
	}

	var event CodexEvent
	if err := json.Unmarshal([]byte(`{"type":"item.completed","item":{"type":"agent_message","text":"Ordinary editing seemed simpler."}}`), &event); err != nil {
		t.Fatal(err)
	}
	result := &RunResult{}
	appendCodexAgentResponse(result, event.Item)
	if got, want := result.agentResponse, "Ordinary editing seemed simpler."; got != want {
		t.Errorf("captured reflection = %q, want %q", got, want)
	}
}

func TestSemanticBatchReflectionRequiresConsecutiveUnbatchedCalls(t *testing.T) {
	t.Parallel()

	result := &RunResult{MCPVerified: true, ToolCalls: []ToolCall{
		{Name: "semantic_rename", Server: "semedit"},
		{Name: "semantic_insert_function", Server: "semedit"},
	}}
	if !shouldRequestSemanticBatchReflection(ArmSemedit, "thread-123", result) {
		t.Fatal("consecutive unbatched semantic calls should request a reflection")
	}
	if !shouldRequestSemanticBatchReflection(ArmSemedit, "thread-123", &RunResult{ToolCalls: result.ToolCalls}) {
		t.Fatal("attempted consecutive semantic calls should request a reflection even when transport was not confirmed")
	}
	for _, test := range []struct {
		name   string
		arm    ArmType
		result *RunResult
	}{
		{name: "baseline", arm: ArmBaseline, result: result},
		{name: "separated", arm: ArmSemedit, result: &RunResult{MCPVerified: true, ToolCalls: []ToolCall{{Name: "semantic_rename", Server: "semedit"}, {Name: "shell", Server: "local"}, {Name: "semantic_insert_function", Server: "semedit"}}}},
		{name: "batch", arm: ArmSemedit, result: &RunResult{MCPVerified: true, ToolCalls: []ToolCall{{Name: "semantic_rename", Server: "semedit"}, {Name: "semantic_insert_function", Server: "semedit"}, {Name: "mcp__semedit__semantic_batch"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if shouldRequestSemanticBatchReflection(test.arm, "thread-123", test.result) {
				t.Fatal("unexpected semantic batch reflection")
			}
		})
	}
	if !strings.Contains(semanticBatchReflectionPrompt, "semantic_batch") || !strings.Contains(semanticBatchReflectionPrompt, "do not run tools") {
		t.Errorf("batch reflection prompt lacks its constraints: %q", semanticBatchReflectionPrompt)
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
	if got := string(res.ToolCalls[0].Arguments); !strings.Contains(got, `"ToolName":"\"semantic_rename\""`) {
		t.Errorf("semantic_rename arguments = %s, want the recorded Agy arguments", got)
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
		MCPServerInstructionsNone:         "mcp_servers.semedit.args=[\"mcp\",\"--profile\",\"full\"]",
		MCPServerInstructionsDescriptive:  "Semedit semantic tools are available",
		MCPServerInstructionsPrescriptive: "Inspect the complete tool inventory",
	} {
		runner := NewRunner(t.TempDir(), WithMCPServerInstructions(mode))
		override, err := runner.codexMCPServerInstructionsOverride()
		if err != nil {
			t.Fatalf("build %s override: %v", mode, err)
		}
		parts := []string{"mcp_servers.semedit.args=", want}
		if mode != MCPServerInstructionsNone {
			parts = append(parts, "--instructions")
		}
		for _, part := range parts {
			if !strings.Contains(override, part) {
				t.Errorf("%s Codex MCP override %q missing %q", mode, override, part)
			}
		}
	}
	if got, err := NewRunner(t.TempDir(), WithMCPServerInstructions(MCPServerInstructionsPrescriptive)).codexMCPOverride(ArmBaseline); err != nil || got != "mcp_servers.semedit.enabled=false" {
		t.Errorf("baseline Codex MCP override = %q, %v; want semedit disabled", got, err)
	}
	for _, key := range []string{"GOENV", "GOCACHE", "GOMODCACHE", "GOTMPDIR", "GOBIN", "GOPATH", "GOFLAGS", "GOWORK"} {
		if !strings.Contains(codexMCPGoEnvironmentOverride(), `"`+key+`"`) {
			t.Errorf("Codex MCP environment override missing %s", key)
		}
	}
	runner := NewRunner(t.TempDir())
	if got, want := runner.codexMCPBinaryOverride(), `mcp_servers.semedit.command="`+filepath.Join(filepath.Dir(filepath.Dir(runner.baseScratchDir)), "bin", "semedit-next")+`"`; got != want {
		t.Errorf("Codex MCP binary override = %q, want %q", got, want)
	}
	if got, want := codexMCPEnabledOverride(), "mcp_servers.semedit.enabled=true"; got != want {
		t.Errorf("Codex MCP enabled override = %q, want %q", got, want)
	}
}

func TestFixtureGoEnvironmentIsWorkspaceLocal(t *testing.T) {
	root := t.TempDir()
	for _, key := range []string{"GOENV", "GOCACHE", "GOMODCACHE", "GOTMPDIR", "GOBIN", "GOPATH", "GOFLAGS", "GOWORK"} {
		t.Setenv(key, filepath.Join(t.TempDir(), key))
	}
	env, err := fixtureGoEnvironment(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"GOENV":      filepath.Join(root, ".scratch", "go", "env"),
		"GOCACHE":    filepath.Join(root, ".scratch", "go", "build"),
		"GOMODCACHE": filepath.Join(root, ".scratch", "go", "mod"),
		"GOTMPDIR":   filepath.Join(root, ".scratch", "go", "tmp"),
		"GOBIN":      filepath.Join(root, ".scratch", "go", "bin"),
		"GOPATH":     filepath.Join(root, ".scratch", "go"),
		"GOFLAGS":    "",
		"GOWORK":     "off",
	}
	for key, path := range want {
		got := environmentValue(env, key)
		if got != path {
			t.Errorf("%s = %q, want %q", key, got, path)
		}
		if key == "GOCACHE" || key == "GOMODCACHE" || key == "GOTMPDIR" || key == "GOBIN" || key == "GOPATH" {
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s directory was not created: %v", key, err)
			}
		}
	}
}

func TestWriteBenchmarkAGENTSOverride(t *testing.T) {
	workDir := t.TempDir()
	if err := writeBenchmarkAGENTSOverride(workDir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workDir, "AGENTS.override.md")
	// #nosec G304 -- path is a harness-generated file inside the test temp directory
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != benchmarkAGENTSOverride {
		t.Errorf("fixture AGENTS override = %q, want %q", got, benchmarkAGENTSOverride)
	}
	if err := writeBenchmarkAGENTSOverride(workDir); err == nil {
		t.Fatal("second fixture AGENTS override write unexpectedly succeeded")
	}
}

func environmentValue(env []string, key string) string {
	for _, entry := range env {
		candidate, value, found := strings.Cut(entry, "=")
		if found && candidate == key {
			return value
		}
	}
	return ""
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

func TestTask11MixedSinkControlPassesOracle(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_11_mixed_sink_api_migration.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	hiddenTestDir, err := filepath.Abs(filepath.Join("..", "..", task.Metadata.Oracle.Test.HiddenTestDir))
	if err != nil {
		t.Fatalf("resolve hidden test directory: %v", err)
	}
	task.Metadata.Oracle.Test.HiddenTestDir = hiddenTestDir

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	result, err := NewRunner(t.TempDir()).ExecuteControl(ctx, task)
	if err != nil {
		t.Fatalf("execute control: %v", err)
	}
	if !result.Success {
		t.Fatalf("task-11 control oracle = %#v, error = %q, want passing oracle", result.Oracle, result.Error)
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

	if len(task.Metadata.PromptVariants) != 3 {
		t.Fatalf("expected 3 prompt variants, got %d (%v)", len(task.Metadata.PromptVariants), task.Metadata.PromptVariants)
	}
	if !strings.Contains(task.Metadata.PromptVariants["crypto_rand"], "crypto/rand") {
		t.Errorf("expected crypto/rand in crypto_rand variant, got: %s", task.Metadata.PromptVariants["crypto_rand"])
	}
	if got := task.Metadata.PromptVariants["prefer_discover_semedit"]; !strings.Contains(got, "complete available tool inventory") || !strings.Contains(got, "semantic editing tools") {
		t.Errorf("prefer-discover-semedit prompt = %q, want discovery and semantic-tool guidance", got)
	}
}

func TestDiscoveryPromptVariantsRequireToolInventoryInspection(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		filepath.Join("..", "..", "testdata", "scripts", "generate_template_main.txtar"),
		filepath.Join("..", "..", "testdata", "bench", "task_11_mixed_sink_api_migration.txtar"),
	}
	for _, fixturePath := range fixtures {
		// #nosec G304 -- fixture path comes from the static list above.
		data, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("read fixture %s: %v", fixturePath, err)
		}
		task, err := ParseTask(data)
		if err != nil {
			t.Fatalf("parse fixture %s: %v", fixturePath, err)
		}
		prompt := task.Metadata.PromptVariants["prefer_discover_semedit"]
		for _, phrase := range []string{"complete available tool inventory", "semantic editing tools"} {
			if !strings.Contains(prompt, phrase) {
				t.Errorf("%s discovery prompt = %q, missing %q", fixturePath, prompt, phrase)
			}
		}
	}
}

func TestInteractiveFollowupLaddersAreParsed(t *testing.T) {
	t.Parallel()

	tests := map[string]int{
		"task_01_rename_local.txtar":             3,
		"task_04_insert_public.txtar":            3,
		"task_09_composite_refactor.txtar":       4,
		"task_11_mixed_sink_api_migration.txtar": 4,
	}
	for fixture, wantSteps := range tests {
		// #nosec G304 -- fixture name comes from this static test map.
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", fixture))
		if err != nil {
			t.Fatalf("read %s: %v", fixture, err)
		}
		task, err := ParseTask(data)
		if err != nil {
			t.Fatalf("parse %s: %v", fixture, err)
		}
		if got := len(task.Metadata.InteractiveFollowups); got != wantSteps {
			t.Errorf("%s follow-up steps = %d, want %d", fixture, got, wantSteps)
		}
	}
}

func TestMutationPolicyGuidance(t *testing.T) {
	t.Parallel()

	task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{MutationPolicy: MutationPolicyConfig{
		DisallowedFiles: []string{"go.sum", "api/server_test.go", "go.mod"},
	}}}}
	const prompt = "Continue the refactor."

	initial := withMutationPolicyGuidance(prompt, task, false)
	if want := "Do not edit tests. You are forbidden to modify protected files: \"go.mod\", \"go.sum\".\n\nContinue the refactor."; initial != want {
		t.Fatalf("initial policy guidance = %q, want %q", initial, want)
	}
	followup := withMutationPolicyGuidance(prompt, task, true)
	if !strings.Contains(followup, "If an earlier turn changed any protected file, restore its original contents before continuing.") {
		t.Fatalf("follow-up policy guidance does not request remediation: %q", followup)
	}

	withoutPolicy := &Task{}
	if got := withMutationPolicyGuidance(prompt, withoutPolicy, true); got != prompt {
		t.Fatalf("guidance without protected files = %q, want original prompt", got)
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

func TestCreateBenchmarkRunDir(t *testing.T) {
	root := t.TempDir()
	runDir, err := createBenchmarkRunDir(root, "run-20260922T120000Z")
	if err != nil {
		t.Fatalf("create benchmark run directory: %v", err)
	}
	if want := filepath.Join(root, "run-20260922T120000Z"); runDir != want {
		t.Fatalf("run directory = %q, want %q", runDir, want)
	}
	if _, err := os.Stat(runDir); err != nil {
		t.Fatalf("stat run directory: %v", err)
	}
	if _, err := createBenchmarkRunDir(root, "run-20260922T120000Z"); err == nil {
		t.Fatal("duplicate run id unexpectedly overwrote an existing run")
	}
	for _, runID := range []string{"", ".", "..", "run/next", "run next", "run:next"} {
		if _, err := createBenchmarkRunDir(root, runID); err == nil {
			t.Errorf("invalid run id %q unexpectedly accepted", runID)
		}
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
			if len(b.PromptVariants) != 3 {
				t.Errorf("expected 3 prompt variants for task-07-generate-template-main, got %d", len(b.PromptVariants))
			}
		}
		if b.TaskID == "task-11-mixed-sink-api-migration" {
			foundMixedSinkMigration = true
			if got, want := b.PromptVariants, []string{"default", "prefer_discover_semedit"}; !slices.Equal(got, want) {
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
