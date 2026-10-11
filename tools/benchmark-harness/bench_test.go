// Package main verifies the benchmark task parsing, extraction, and oracle behavior.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"semedit/internal/mcp"
	"semedit/internal/pipeline"
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
	cases := []codexDiagnosticCase{
		{name: "success with stderr", stderr: "MCP startup warning\n", stdout: threadStarted},
		{name: "success empty stderr", stdout: threadStarted},
		{name: "oversized stderr", stderr: strings.Repeat("x", maxCodexStderrBytes+100), stdout: threadStarted},
		{name: "failure retains stderr", stderr: "server failed to start", stdout: threadStarted, exit: 7, wantErr: true, wantStatus: 7},
		{name: "malformed stdout", stderr: "diagnostic", stdout: "not-json\n" + threadStarted, wantErr: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			runCodexDiagnosticCase(t, testCase)
		})
	}
}

type codexDiagnosticCase struct {
	name       string
	stderr     string
	stdout     string
	exit       int
	wantStatus int
	wantErr    bool
}

func runCodexDiagnosticCase(t *testing.T, testCase codexDiagnosticCase) {
	t.Helper()
	binDir := t.TempDir()
	script := filepath.Join(binDir, "codex")
	content := fmt.Sprintf("#!/bin/sh\nprintf '%%b' %q >&2\nprintf '%%b' %q\nexit %d\n", testCase.stderr, testCase.stdout, testCase.exit)
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- the temporary test command must be executable.
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	result := &RunResult{Arm: ArmBaseline}
	threadID, runErr := NewRunner(t.TempDir()).runCodex(context.Background(), t.TempDir(), Target{}, "prompt", result, "")
	if threadID != "thread-123" {
		t.Fatalf("runCodex thread ID = %q, want thread-123", threadID)
	}
	assertCodexRunStatus(t, testCase, result, runErr)
	assertCodexStderr(t, testCase, result)
	assertCodexMalformedOutput(t, testCase, result)
	assertCodexExitCodeSerialization(t, testCase, result)
}

func assertCodexRunStatus(t *testing.T, testCase codexDiagnosticCase, result *RunResult, runErr error) {
	t.Helper()
	if (runErr != nil) != testCase.wantErr {
		t.Fatalf("runCodex error = %v, want error=%t", runErr, testCase.wantErr)
	}
	if result.CodexExitCode == nil || *result.CodexExitCode != testCase.wantStatus {
		t.Fatalf("exit status = %v, want %d", result.CodexExitCode, testCase.wantStatus)
	}
}

func assertCodexStderr(t *testing.T, testCase codexDiagnosticCase, result *RunResult) {
	t.Helper()
	if testCase.stderr != "" && !strings.Contains(result.CodexStderr, strings.TrimSpace(testCase.stderr)[:min(len(strings.TrimSpace(testCase.stderr)), 20)]) {
		t.Fatalf("stderr = %q, missing diagnostic", result.CodexStderr)
	}
	if len(result.CodexStderr) > maxCodexStderrBytes+64 {
		t.Fatalf("stderr retained %d bytes, expected bounded output", len(result.CodexStderr))
	}
	if testCase.name == "oversized stderr" {
		assertCodexStderrTruncation(t, result.CodexStderr)
	}
}

func assertCodexStderrTruncation(t *testing.T, stderr string) {
	t.Helper()
	if !strings.Contains(stderr, "[stderr truncated; retained first 65536 bytes]") {
		t.Fatal("oversized stderr missing truncation marker")
	}
	if !strings.HasPrefix(stderr, strings.Repeat("x", maxCodexStderrBytes)) {
		t.Fatal("oversized stderr payload was not retained up to the boundary")
	}
}

func assertCodexMalformedOutput(t *testing.T, testCase codexDiagnosticCase, result *RunResult) {
	t.Helper()
	if testCase.name != "malformed stdout" {
		return
	}
	if result.Turns != 0 {
		t.Fatalf("malformed event stream manufactured task turns: %d", result.Turns)
	}
	if result.ToolCount != 0 || result.OutputTokens != 0 || result.ReasoningTokens != 0 || result.PromptTokens != 0 || result.agentResponse != "" {
		t.Fatalf("malformed stdout contaminated metrics: %#v", result)
	}
}

func assertCodexExitCodeSerialization(t *testing.T, testCase codexDiagnosticCase, result *RunResult) {
	t.Helper()
	var wantCode string
	switch testCase.name {
	case "success with stderr":
		wantCode = `"codex_exit_code":0`
	case "failure retains stderr":
		wantCode = `"codex_exit_code":7`
	default:
		return
	}
	data, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(data), wantCode) {
		t.Fatalf("exit status %s was not serialized: %s (%v)", wantCode, data, err)
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
		`{"type":"step_finish","part":{"type":"step-finish","id":"finish-1","reason":"stop","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}}`,
	}, "\n") + "\n"
	content := fmt.Sprintf("#!/bin/sh\nprintf '%%b' %q\n", stdout)
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- the temporary test command must be executable.
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatal(err)
	}
	installPassthroughOpenCodeSandbox(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	workDir := t.TempDir()
	res := &RunResult{Arm: ArmSemedit}
	sessionID, err := NewRunner(t.TempDir()).runOpenCode(t.Context(), workDir, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free"}, "prompt", res, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.processStartedAt.IsZero() || res.measuredWallClock(time.Now()) <= 0 {
		t.Fatalf("OpenCode task was not timed from process start: %s", res.processStartedAt)
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
	configPath, env, configErr := runner.openCodeEnvironment(t.Context(), workDir, ArmSemedit, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free"})
	if configErr != nil {
		t.Fatal(configErr)
	}
	// #nosec G304 -- config path is constructed by the harness inside the temporary fixture.
	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var config struct {
		Provider map[string]map[string]any `json:"provider"`
		MCP      struct {
			Servers map[string]struct {
				Command     []string          `json:"command"`
				CWD         string            `json:"cwd"`
				Environment map[string]string `json:"environment"`
				CodeMode    bool              `json:"codemode"`
			} `json:"servers"`
		} `json:"mcp"`
	}
	if decodeErr := json.Unmarshal(data, &config); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	openRouterProvider, providerFound := config.Provider["openrouter"]
	if !providerFound {
		t.Fatal("OpenRouter provider is absent")
	}
	if _, foundNPM := openRouterProvider["npm"]; foundNPM {
		t.Error("OpenRouter config overrides the native runtime with npm")
	}
	if _, foundOptions := openRouterProvider["options"]; foundOptions {
		t.Error("OpenRouter config overrides native endpoint or credentials")
	}
	models, modelsFound := openRouterProvider["models"].(map[string]any)
	if !modelsFound {
		t.Fatal("OpenRouter model registration is absent")
	}
	if _, freeModelFound := models["free"]; !freeModelFound {
		t.Errorf("OpenRouter model registration = %#v, want free", models)
	}
	if strings.Contains(string(data), "secret-test-key") {
		t.Fatal("OpenRouter credential was serialized into OpenCode configuration")
	}
	server, found := config.MCP.Servers["semedit"]
	if !found {
		t.Fatal("semedit MCP server is absent")
	}
	expectedRoot, err := canonicalFixtureRoot(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(server.Command, "--metrics-file") || !slices.Contains(server.Command, benchmarkMCPMetricsPath(workDir)) {
		t.Errorf("MCP command = %#v, want fixture-local metrics file", server.Command)
	}
	if !slices.Contains(server.Command, "--expected-root") || !slices.Contains(server.Command, expectedRoot) {
		t.Errorf("MCP command = %#v, want expected root %q", server.Command, expectedRoot)
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

	assertOpenCodeBaselineConfiguration(t, runner)
	assertLegacyOpenCodeConfiguration(t, runner)
}

func assertOpenCodeBaselineConfiguration(t *testing.T, runner *Runner) {
	t.Helper()
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

func assertLegacyOpenCodeConfiguration(t *testing.T, runner *Runner) {
	t.Helper()
	const relayURL = "http://127.0.0.1:43123/v1"
	runner = NewRunner(runner.baseScratchDir, WithOpenCodeRelayURL(relayURL), WithOpenCodeModelLimits(openCodeModelLimits{Context: 65536, Output: 8192}))
	configPath, _, err := runner.openCodeEnvironment(t.Context(), t.TempDir(), ArmBaseline, Target{Harness: string(HarnessOpenCode), Model: "qwen36-iq4-xs"})
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- the path is returned by the fixture-scoped harness setup.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Provider map[string]struct {
			Options map[string]string `json:"options"`
			Models  map[string]struct {
				ID    string `json:"id"`
				Limit struct {
					Context int `json:"context"`
					Output  int `json:"output"`
				} `json:"limit"`
			} `json:"models"`
		} `json:"provider"`
	}
	if decodeErr := json.Unmarshal(data, &config); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if got := config.Provider["amdbeast"].Options["baseURL"]; got != relayURL {
		t.Errorf("AMD Beast base URL = %q, want %q", got, relayURL)
	}
	if got := config.Provider["amdbeast"].Models["qwen36-iq4-xs"].ID; got != "qwen36-iq4-xs" {
		t.Errorf("AMD Beast API model = %q, want qwen36-iq4-xs", got)
	}
	if got := config.Provider["amdbeast"].Models["qwen36-iq4-xs"].Limit; got.Context != 65536 || got.Output != 8192 {
		t.Errorf("AMD Beast limits = %+v, want context=65536 output=8192", got)
	}
	if _, found := config.Provider["openrouter"]; found {
		t.Error("AMD Beast config unexpectedly includes OpenRouter")
	}
}

func TestOpenCodeAMDConfigRequiresRunRelay(t *testing.T) {
	target := Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}
	if _, err := NewRunner(t.TempDir()).openCodeConfig(t.TempDir(), ArmBaseline, target, nil); err == nil || !strings.Contains(err.Error(), "run-owned OpenCode relay") {
		t.Fatalf("config without relay error = %v, want explicit relay requirement", err)
	}
	const relayURL = "http://127.0.0.1:43123/v1"
	config, err := NewRunner(t.TempDir(), WithOpenCodeRelayURL(relayURL), WithOpenCodeModelLimits(openCodeModelLimits{Context: 65536, Output: 8192})).openCodeConfig(t.TempDir(), ArmBaseline, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := config["provider"].(map[string]any)["amdbeast"].(map[string]any)
	options := provider["options"].(map[string]any)
	if options["baseURL"] != relayURL {
		t.Fatalf("AMD Beast config base URL = %v, want %q", options["baseURL"], relayURL)
	}
	profile, err := renderOpenCodeSeatbeltProfile(t.TempDir(), "43123", "/bin/sh", "/bin/cat", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(profile, `(allow network-outbound (remote tcp "localhost:43123"))`) || strings.Contains(profile, "192.168.1.122") {
		t.Fatalf("profile does not restrict outbound access to the same relay port: %s", profile)
	}
	openRouterPort, err := openCodeSandboxRelayPort("https://openrouter.ai/api/v1")
	if err != nil || openRouterPort != "" {
		t.Fatalf("OpenRouter outbound port = %q, error = %v; want no provider egress", openRouterPort, err)
	}
	openRouterProfile, err := renderOpenCodeSeatbeltProfile(t.TempDir(), openRouterPort, "/bin/sh", "/bin/cat", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(openRouterProfile, "network-outbound") {
		t.Fatalf("OpenRouter profile unexpectedly permits outbound traffic: %s", openRouterProfile)
	}
}

func TestSemanticToolReflectionOnlyFollowsUnverifiedSemanticRuns(t *testing.T) {
	t.Parallel()

	if !shouldRequestSemanticToolReflection(ArmSemedit, "thread-123", &RunResult{Success: true}) {
		t.Fatal("unverified semantic run should request a reflection")
	}
	for _, test := range []struct {
		name      string
		arm       ArmType
		sessionID string
		result    *RunResult
	}{
		{name: "baseline", arm: ArmBaseline, sessionID: "thread-123", result: &RunResult{Success: true}},
		{name: "no session", arm: ArmSemedit, result: &RunResult{Success: true}},
		{name: "semantic confirmed", arm: ArmSemedit, sessionID: "thread-123", result: &RunResult{MCPVerified: true, Success: true}},
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
	result := &RunResult{Success: true}
	appendCodexAgentResponse(result, event.Item)
	if got, want := result.agentResponse, "Ordinary editing seemed simpler."; got != want {
		t.Errorf("captured reflection = %q, want %q", got, want)
	}
}

func TestSemanticBatchReflectionRequiresConsecutiveUnbatchedCalls(t *testing.T) {
	t.Parallel()

	result := &RunResult{MCPVerified: true, Success: true, ToolCalls: []ToolCall{
		{Name: "semantic_rename", Server: "semedit"},
		{Name: "semantic_insert_function", Server: "semedit"},
	}}
	if !shouldRequestSemanticBatchReflection(ArmSemedit, "thread-123", result) {
		t.Fatal("consecutive unbatched semantic calls should request a reflection")
	}
	if !shouldRequestSemanticBatchReflection(ArmSemedit, "thread-123", &RunResult{Success: true, ToolCalls: result.ToolCalls}) {
		t.Fatal("attempted consecutive semantic calls should request a reflection even when transport was not confirmed")
	}
	for _, test := range []struct {
		name   string
		arm    ArmType
		result *RunResult
	}{
		{name: "baseline", arm: ArmBaseline, result: result},
		{name: "separated", arm: ArmSemedit, result: &RunResult{MCPVerified: true, Success: true, ToolCalls: []ToolCall{{Name: "semantic_rename", Server: "semedit"}, {Name: "shell", Server: "local"}, {Name: "semantic_insert_function", Server: "semedit"}}}},
		{name: "batch", arm: ArmSemedit, result: &RunResult{MCPVerified: true, Success: true, ToolCalls: []ToolCall{{Name: "semantic_rename", Server: "semedit"}, {Name: "semantic_insert_function", Server: "semedit"}, {Name: "mcp__semedit__semantic_batch"}}}},
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
	if err := parseAgyTranscript(transcript, res); err != nil {
		t.Fatalf("parse Agy transcript: %v", err)
	}

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

	fixtureRoot := t.TempDir()
	expectedRoot, err := canonicalFixtureRoot(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	for mode, want := range map[MCPServerInstructionMode]string{
		MCPServerInstructionsNone:         "",
		MCPServerInstructionsDescriptive:  "Semedit semantic tools are available",
		MCPServerInstructionsPrescriptive: "Inspect the complete tool inventory",
	} {
		runner := NewRunner(t.TempDir(), WithMCPServerInstructions(mode))
		override, err := runner.codexMCPServerInstructionsOverride(fixtureRoot)
		if err != nil {
			t.Fatalf("build %s override: %v", mode, err)
		}
		parts := []string{"mcp_servers.semedit.args=", "--expected-root", expectedRoot, "--metrics-file", benchmarkMCPMetricsPath(fixtureRoot)}
		if want != "" {
			parts = append(parts, want, "--instructions")
		}
		for _, part := range parts {
			if !strings.Contains(override, part) {
				t.Errorf("%s Codex MCP override %q missing %q", mode, override, part)
			}
		}
	}
	if got, err := NewRunner(t.TempDir(), WithMCPServerInstructions(MCPServerInstructionsPrescriptive)).codexMCPOverride(ArmBaseline, fixtureRoot); err != nil || got != "mcp_servers.semedit.enabled=false" {
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

func TestSemeditVerifyWarmupUsesEmptyFixtureLocalCaches(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "fixture")
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module example.com/warmup\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package warmup\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEMEDIT_TEST_WORKDIR", workDir)
	binary := filepath.Join(root, "fake-semedit")
	script := "#!/bin/sh\n" +
		"test \"$1\" = verify && test \"$2\" = --file && test \"$3\" = main.go && test \"$4\" = --language && test \"$5\" = go && test \"$6\" = --check-only || exit 9\n" +
		"test \"$GOMODCACHE\" = \"$SEMEDIT_TEST_WORKDIR/.scratch/go/mod\" || exit 10\n" +
		"test \"$GOCACHE\" = \"$SEMEDIT_TEST_WORKDIR/.scratch/go/build\" || exit 11\n" +
		"mkdir \"$GOMODCACHE/seed\"\n"
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := runSemeditVerifyWarmup(t.Context(), workDir, "main.go", binary); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".scratch", "go", "mod", "seed")); err != nil {
		t.Fatalf("warmup did not use fixture-local cache: %v", err)
	}
	if _, err := runSemeditVerifyWarmup(t.Context(), workDir, "main.go", binary); err == nil {
		t.Fatal("warmup accepted a non-empty module cache")
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
		if task.Metadata.Repository.URL != "" {
			continue
		}

		// Verify extraction to temp dir
		tmpDir := t.TempDir()
		if err := task.ExtractTo(tmpDir); err != nil {
			t.Fatalf("extract task %s: %v", entry.Name(), err)
		}

		// Verify go.mod was extracted
		if len(task.Archive.Files) > 0 {
			if _, err := os.Stat(filepath.Join(tmpDir, "go.mod")); err != nil {
				t.Errorf("task %s failed to extract go.mod: %v", entry.Name(), err)
			}
		}
	}

	if foundTasks < expectedTasks {
		t.Errorf("expected at least %d benchmark tasks, found %d", expectedTasks, foundTasks)
	}
}

func TestEmptyWorkspaceFixtureRejectsChanges(t *testing.T) {
	data, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "task_00_hi_overhead.txtar"))
	if readErr != nil {
		t.Fatalf("read no-op fixture: %v", readErr)
	}
	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatalf("parse no-op fixture: %v", parseErr)
	}
	if len(task.Archive.Files) != 0 {
		t.Fatalf("empty fixture contains %d files", len(task.Archive.Files))
	}
	workDir := t.TempDir()
	if err := task.ExtractTo(workDir); err != nil {
		t.Fatalf("extract no-op fixture: %v", err)
	}
	passed, evaluateErr := Evaluate(context.Background(), task, workDir, nil)
	if evaluateErr != nil {
		t.Fatalf("evaluate unchanged workspace: %v", evaluateErr)
	}
	if !passed.Passed {
		t.Fatalf("unchanged workspace failed: %s", passed.ErrorMessage)
	}
	changed, changedErr := Evaluate(context.Background(), task, workDir, []string{"unexpected.txt"})
	if changedErr != nil {
		t.Fatalf("evaluate changed workspace: %v", changedErr)
	}
	if changed.Passed || changed.FailureStage != "level_1_mutation_policy" {
		t.Fatalf("changed workspace result = %+v", changed)
	}
}

func TestOracleMutationPolicyEnforcement(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_01_rename_local.txtar"))
	data, readErr := os.ReadFile(fixturePath)
	if readErr != nil {
		t.Fatalf("read fixture: %v", readErr)
	}

	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatalf("parse task: %v", parseErr)
	}

	tmpDir := t.TempDir()
	if err := task.ExtractTo(tmpDir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Case 1: Unauthorized change to go.mod
	res, evaluateErr := Evaluate(context.Background(), task, tmpDir, []string{"go.mod"})
	if evaluateErr != nil {
		t.Fatalf("evaluate: %v", evaluateErr)
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
	data, readErr := os.ReadFile(fixturePath)
	if readErr != nil {
		t.Fatalf("read fixture: %v", readErr)
	}

	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatalf("parse task: %v", parseErr)
	}

	tmpDir := t.TempDir()
	if err := task.ExtractTo(tmpDir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Unmodified initial fixture fails Level 2 AST (newName missing and oldName present)
	res, evaluateErr := Evaluate(context.Background(), task, tmpDir, nil)
	if evaluateErr != nil {
		t.Fatalf("evaluate: %v", evaluateErr)
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

	before, snapshotErr := snapshotWorkspaceFiles(root)
	if snapshotErr != nil {
		t.Fatalf("snapshot workspace: %v", snapshotErr)
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

	got, changedErr := changedWorkspaceFiles(root, before)
	if changedErr != nil {
		t.Fatalf("collect workspace changes: %v", changedErr)
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

func TestTask02LargeContextExtractsCompilingWorkspace(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_02_rename_cross_pkg.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	workDir := t.TempDir()
	if err := task.ExtractVariantTo(workDir, "large"); err != nil {
		t.Fatalf("extract large fixture: %v", err)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build large fixture: %v\n%s", err, output)
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
		"explicit":     "Call semantic tools in order.",
		"empty-prompt": " ",
	}}}
	if got, want := matrixPromptVariants(withPrompts, "small"), []string{"small:default"}; !slices.Equal(got, want) {
		t.Fatalf("unexpected expanded prompt variants: got %v, want %v", got, want)
	}
	if got, want := matrixPromptVariants(withPrompts, "large:default"), []string{"large:default"}; !slices.Equal(got, want) {
		t.Fatalf("unexpected explicit prompt variant: got %v, want %v", got, want)
	}
	if got, want := matrixPromptVariants(withPrompts, "small:explicit"), []string{"small:explicit"}; !slices.Equal(got, want) {
		t.Fatalf("unexpected explicit arm selection: got %v, want %v", got, want)
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
			if got, want := b.PromptVariants, []string{"default", "explicit", "prefer_discover_semedit"}; !slices.Equal(got, want) {
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

func TestCollectAvailableBenchmarksReturnsErrorForMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := CollectAvailableBenchmarks(missing); err == nil {
		t.Fatal("CollectAvailableBenchmarks accepted a missing required directory")
	}
}

func TestCollectAvailableBenchmarksReturnsErrorForMalformedFixture(t *testing.T) {
	benchDir := t.TempDir()
	fixture := filepath.Join(benchDir, "malformed.txtar")
	if err := os.WriteFile(fixture, []byte("not a benchmark fixture"), 0o600); err != nil {
		t.Fatalf("write malformed fixture: %v", err)
	}
	if _, err := CollectAvailableBenchmarks(benchDir); err == nil {
		t.Fatal("CollectAvailableBenchmarks accepted a malformed fixture")
	}
}

func TestOpenCodeOpenRouterSmallModelUsesBenchmarkTarget(t *testing.T) {
	target := Target{Harness: string(HarnessOpenCode), Model: "openrouter/nvidia/nemotron-3-ultra:free"}
	config, err := NewRunner(t.TempDir()).openCodeConfig(t.TempDir(), ArmBaseline, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := config["model"]; got != target.Model {
		t.Errorf("default model = %v, want %q", got, target.Model)
	}
	if got := config["small_model"]; got != target.Model {
		t.Errorf("small model = %v, want %q", got, target.Model)
	}
}

func TestParseTaskRejectsMalformedAndMissingOracleSettings(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "task_01_rename_local.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement, field string }{
		{"invalid clean_compile", "clean_compile: true", "clean_compile: tru", "oracle.level_3_build.clean_compile"},
		{"invalid pass_tests", "pass_tests: true", "pass_tests: tru", "oracle.level_4_test.pass_tests"},
		{"missing clean_compile", "    clean_compile: true\n", "", "oracle.level_3_build.clean_compile"},
		{"missing pass_tests", "    pass_tests: true\n", "", "oracle.level_4_test.pass_tests"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := strings.Replace(string(fixture), tc.old, tc.replacement, 1)
			if data == string(fixture) {
				t.Fatalf("fixture did not contain %q", tc.old)
			}
			_, err := ParseTask([]byte(data))
			if err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("ParseTask error = %v, want field %s", err, tc.field)
			}
		})
	}
}

func TestParseTaskValidatesOracleGoTestCommand(t *testing.T) {
	fixture := func(argument string) []byte {
		return fmt.Appendf(nil, `task_id: oracle-test-command
category: test
instruction: Validate the oracle test command.
oracle:
  level_3_build:
    clean_compile: false
  level_4_test:
    pass_tests: true
    command:
      - "-race"
      - %q
      - "./..."
-- go.mod --
module example.com/oracle-test-command
go 1.23
`, argument)
	}

	task, err := ParseTask(fixture("-gcflags=all=-l"))
	if err != nil {
		t.Fatalf("ParseTask accepted command error = %v", err)
	}
	if want := []string{"-race", "-gcflags=all=-l", "./..."}; !slices.Equal(task.Metadata.Oracle.Test.Command, want) {
		t.Fatalf("oracle test command = %q, want %q", task.Metadata.Oracle.Test.Command, want)
	}

	for _, argument := range []string{
		"-gcflags",
		"all=-l",
		"-gcflags=all=-N",
		"-gcflags=./...=-l",
		"-gcflags=all=-l -N",
		"-ldflags=all=-s",
		"../...",
		"./pkg/../...",
		"./pkg/..",
	} {
		t.Run(argument, func(t *testing.T) {
			_, err := ParseTask(fixture(argument))
			if err == nil || !strings.Contains(err.Error(), "unsupported Go test argument") || !strings.Contains(err.Error(), argument) {
				t.Fatalf("ParseTask error = %v, want rejection of %q", err, argument)
			}
		})
	}
}

func TestEvaluateUsesProjectLocalGoEnvironment(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(root, "go-environment")
	fakeGo := filepath.Join(binDir, "go")
	script := `#!/bin/sh
set -eu
test "$GOENV" = "$PWD/.scratch/go/env"
test "$GOCACHE" = "$PWD/.scratch/go/build"
test "$GOMODCACHE" = "$PWD/.scratch/go/mod"
test "$GOTMPDIR" = "$PWD/.scratch/go/tmp"
test "$GOPATH" = "$PWD/.scratch/go"
test "$GOFLAGS" = ""
test "$GOWORK" = "off"
printf ok > "$SEMEDIT_TEST_CAPTURE"
`
	if err := os.WriteFile(fakeGo, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_TEST_CAPTURE", capture)
	for _, key := range []string{"GOENV", "GOCACHE", "GOMODCACHE", "GOTMPDIR", "GOPATH", "GOFLAGS", "GOWORK"} {
		t.Setenv(key, filepath.Join(root, "inherited", key))
	}
	task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{Build: BuildConfig{CleanCompile: true}}}}
	result, err := Evaluate(t.Context(), task, workDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || !result.Level3Build {
		t.Fatalf("oracle result = %+v, want passing isolated build", result)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Fatalf("fake Go environment validation = %q, want ok", data)
	}
}

type oracleCleanupCase struct {
	name           string
	failOperation  string
	wantPassed     bool
	wantStage      string
	wantOperations []string
}

type oracleCleanupFixture struct {
	workDir       string
	originalCache string
	commandLog    string
	task          *Task
}

func TestEvaluateCleansReadOnlyPrivateGoCaches(t *testing.T) {
	cases := []oracleCleanupCase{
		{name: "build and test pass", wantPassed: true, wantOperations: []string{"build", "test"}},
		{name: "build fails", failOperation: "build", wantStage: "level_3_build", wantOperations: []string{"build"}},
		{name: "test fails", failOperation: "test", wantStage: "level_4_test", wantOperations: []string{"build", "test"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := createOracleCleanupFixture(t, testCase.failOperation)
			result, err := Evaluate(t.Context(), fixture.task, fixture.workDir, nil)
			if err != nil {
				t.Fatalf("Evaluate returned cleanup or execution error: %v", err)
			}
			assertOracleCleanupResult(t, testCase, result)
			assertOracleCleanupArtifacts(t, testCase, fixture)
		})
	}
}

func TestEvaluateCancellationIsExecutionError(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte("#!/bin/sh\nsleep 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := Evaluate(ctx, &Task{Metadata: TaskMetadata{Oracle: OracleConfig{Build: BuildConfig{CleanCompile: true}}}}, workDir, nil)
	if result != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Evaluate cancellation = result=%+v err=%v, want execution deadline error", result, err)
	}
}

func createOracleCleanupFixture(t *testing.T, failOperation string) oracleCleanupFixture {
	t.Helper()
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	hiddenTests := filepath.Join(root, "hidden")
	binDir := filepath.Join(root, "bin")
	for _, dir := range []string{workDir, hiddenTests, binDir, filepath.Join(workDir, ".scratch")} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	originalCache := filepath.Join(workDir, ".scratch", "owner-cache")
	if err := os.WriteFile(originalCache, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenTests, "acceptance_test.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commandLog := filepath.Join(root, "commands")
	program := oracleCleanupFakeGoProgram()
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(program), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_ORACLE_COMMAND_LOG", commandLog)
	t.Setenv("SEMEDIT_ORACLE_FAIL_OPERATION", failOperation)
	task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{
		Build: BuildConfig{CleanCompile: true},
		Test:  TestConfig{PassTests: true, HiddenTestDir: hiddenTests},
	}}}
	return oracleCleanupFixture{workDir: workDir, originalCache: originalCache, commandLog: commandLog, task: task}
}

func oracleCleanupFakeGoProgram() string {
	return `#!/bin/sh
set -eu
operation="$1"
test "$GOENV" = "$PWD/.scratch/go/env"
test "$GOCACHE" = "$PWD/.scratch/go/build"
test "$GOMODCACHE" = "$PWD/.scratch/go/mod"
test "$GOTMPDIR" = "$PWD/.scratch/go/tmp"
cache="$GOMODCACHE/$operation/nested"
mkdir -p "$cache"
printf module > "$cache/module.zip"
chmod 0444 "$cache/module.zip"
chmod 0555 "$cache"
printf '%s\t%s\n' "$operation" "$PWD" >> "$SEMEDIT_ORACLE_COMMAND_LOG"
if [ "$operation" = build ]; then test ! -f acceptance_test.go; else test -f acceptance_test.go; fi
if [ "$SEMEDIT_ORACLE_FAIL_OPERATION" = "$operation" ]; then exit 9; fi
`
}

func assertOracleCleanupResult(t *testing.T, testCase oracleCleanupCase, result *OracleResult) {
	t.Helper()
	if result.Passed != testCase.wantPassed || result.FailureStage != testCase.wantStage {
		t.Fatalf("oracle result = %+v, want passed=%t stage=%q", result, testCase.wantPassed, testCase.wantStage)
	}
	if testCase.wantPassed && (!result.Level3Build || !result.Level4Test) {
		t.Fatalf("successful build and test levels were not retained: %+v", result)
	}
}

func assertOracleCleanupArtifacts(t *testing.T, testCase oracleCleanupCase, fixture oracleCleanupFixture) {
	t.Helper()
	if data, err := os.ReadFile(fixture.originalCache); err != nil || string(data) != "keep" {
		t.Fatalf("preexisting workspace scratch changed: data=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(fixture.workDir, "acceptance_test.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("hidden tests reached original workspace: %v", err)
	}
	data, err := os.ReadFile(fixture.commandLog)
	if err != nil {
		t.Fatalf("read fake Go command log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(testCase.wantOperations) {
		t.Fatalf("fake Go operations = %q, want %v", lines, testCase.wantOperations)
	}
	for index, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 2 || fields[0] != testCase.wantOperations[index] {
			t.Fatalf("fake Go command %d = %q, want operation %q and private path", index, line, testCase.wantOperations[index])
		}
		if _, err := os.Stat(fields[1]); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("private oracle workspace remains after cleanup: %s (%v)", fields[1], err)
		}
	}
}

func codexMCPArgsFromOverride(t *testing.T, override string) []string {
	t.Helper()
	_, rawArgs, ok := strings.Cut(override, "=")
	if !ok {
		t.Fatalf("Codex override has no args assignment: %s", override)
	}
	var args []string
	if decodeErr := json.Unmarshal([]byte(rawArgs), &args); decodeErr != nil {
		t.Fatalf("decode Codex args from %q: %v", rawArgs, decodeErr)
	}
	return args
}

func TestEvaluateKeepsHiddenTestsOutOfAgentWorkspace(t *testing.T) {
	for _, pass := range []bool{true, false} {
		t.Run(fmt.Sprintf("oracle_passes_%v", pass), func(t *testing.T) {
			workDir, hiddenDir := t.TempDir(), t.TempDir()
			if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module private-oracle\n\ngo 1.23\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			testBody := "package main\nimport \"testing\"\nfunc TestHidden(t *testing.T) { t.Fatal(\"hidden-failure\") }\n"
			if pass {
				testBody = "package main\nimport \"testing\"\nfunc TestHidden(t *testing.T) { if Value() != 1 { t.Fatal(\"wrong value\") } }\n"
			}
			if err := os.WriteFile(filepath.Join(hiddenDir, "acceptance_test.go"), []byte(testBody), 0o600); err != nil {
				t.Fatal(err)
			}
			task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{Test: TestConfig{PassTests: true, HiddenTestDir: hiddenDir}}}}
			result, err := Evaluate(context.Background(), task, workDir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if result.Passed != pass {
				t.Fatalf("oracle passed = %v, want %v (%+v)", result.Passed, pass, result)
			}
			if _, err := os.Stat(filepath.Join(workDir, "acceptance_test.go")); !os.IsNotExist(err) {
				t.Fatalf("hidden test visible to resumed agent: stat error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(workDir, ".scratch")); !os.IsNotExist(err) {
				t.Fatalf("private oracle workspace remains before resumed turn: stat error = %v", err)
			}
		})
	}
}

func TestDiagnosticExpectedToolsAreParsedFromOracleMetadata(t *testing.T) {
	data := []byte(`task_id: diagnostic-tools
oracle:
  diagnostic_expected_tools:
    - "mcp__semedit__semantic_lookup"
    - "semantic_replace_body"
  level_3_build:
    clean_compile: true
  level_4_test:
    pass_tests: true
`)
	metadata, err := parseYAMLFrontmatter(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"mcp__semedit__semantic_lookup", "semantic_replace_body"}
	if !slices.Equal(metadata.Oracle.DiagnosticExpectedTools, want) {
		t.Fatalf("diagnostic expected tools = %v, want %v", metadata.Oracle.DiagnosticExpectedTools, want)
	}
}

func TestCodexToolObservationRequiresTerminalTurnAndNamedToolEvents(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(bin, "codex")
	program := `#!/bin/sh
printf '{"type":"thread.started","thread_id":"capture-test"}\n'
case "$SEMEDIT_CODEX_CAPTURE_CASE" in
  unfinished)
    printf '{"type":"turn.started"}\n'
    printf '{"type":"turn.completed"}\n'
    printf '{"type":"turn.started"}\n'
    ;;
  malformed_tool)
    printf '{"type":"turn.started"}\n'
    printf '{"type":"item.completed","item":{"type":"mcp_tool_call","status":"completed"}}\n'
    printf '{"type":"turn.completed"}\n'
    ;;
  complete)
    printf '{"type":"turn.started"}\n'
    printf '{"type":"turn.completed"}\n'
    ;;
esac
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := NewRunner(filepath.Join(root, "scratch"))
	for _, test := range []struct {
		name       string
		state      ToolObservationState
		wantErr    bool
		wantReason string
	}{
		{name: "unfinished latest turn", state: ToolObservationPartial, wantReason: "terminal turn.completed"},
		{name: "unnamed selected tool", state: ToolObservationPartial, wantReason: "malformed tool call records"},
		{name: "terminal capture", state: ToolObservationComplete},
	} {
		t.Run(test.name, func(t *testing.T) {
			mode := "complete"
			if strings.Contains(test.name, "unfinished") {
				mode = "unfinished"
			} else if strings.Contains(test.name, "unnamed") {
				mode = "malformed_tool"
			}
			t.Setenv("SEMEDIT_CODEX_CAPTURE_CASE", mode)
			result := &RunResult{Arm: ArmBaseline}
			_, err := runner.runCodex(t.Context(), root, Target{Harness: string(HarnessCodex)}, "prompt", result, "")
			if (err != nil) != test.wantErr {
				t.Fatalf("runCodex error = %v, want error=%t", err, test.wantErr)
			}
			if result.toolObservationState != test.state || (test.wantReason != "" && !strings.Contains(result.toolObservationReason, test.wantReason)) {
				t.Fatalf("observation state/reason = %q/%q", result.toolObservationState, result.toolObservationReason)
			}
		})
	}
}

func TestCodexTerminationOrigin(t *testing.T) {
	t.Parallel()
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	deadlineCtx, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	cases := []struct {
		name     string
		ctx      context.Context
		signaled bool
		want     string
	}{
		{name: "normal exit", ctx: context.Background(), want: ""},
		{name: "outside signal", ctx: context.Background(), signaled: true, want: "outside_harness"},
		{name: "harness cancellation", ctx: canceledCtx, signaled: true, want: "harness_cancellation"},
		{name: "harness timeout", ctx: deadlineCtx, signaled: true, want: "harness_timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexTerminationOrigin(tc.ctx, tc.signaled); got != tc.want {
				t.Fatalf("codexTerminationOrigin() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAgyMCPConfigSetsFixtureExpectedRoot(t *testing.T) {
	workDir := t.TempDir()
	repositoryRoot := t.TempDir()
	env, envErr := fixtureGoEnvironment(t.Context(), workDir)
	if envErr != nil {
		t.Fatal(envErr)
	}
	if configErr := writeAgyMCPConfig(workDir, repositoryRoot, ArmSemedit, MCPServerInstructionsNone, env); configErr != nil {
		t.Fatal(configErr)
	}
	data, readErr := os.ReadFile(filepath.Join(workDir, ".agents", "mcp_config.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	var config struct {
		Servers map[string]struct {
			Args []string `json:"args"`
		} `json:"mcpServers"`
	}
	if decodeErr := json.Unmarshal(data, &config); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	server, ok := config.Servers["semedit"]
	if !ok {
		t.Fatalf("Agy MCP config missing semedit: %s", data)
	}
	expectedRoot, rootErr := canonicalFixtureRoot(workDir)
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	if !slices.Contains(server.Args, "--metrics-file") || !slices.Contains(server.Args, benchmarkMCPMetricsPath(workDir)) {
		t.Errorf("Agy MCP args = %#v, want fixture-local metrics file", server.Args)
	}
	for i := range len(server.Args) {
		if server.Args[i] == "--expected-root" && i+1 < len(server.Args) && server.Args[i+1] == expectedRoot {
			return
		}
	}
	t.Errorf("Agy MCP args = %#v, want --expected-root %q", server.Args, expectedRoot)
}

func TestParseMCPServerInstructionModesCanonicalizesAndDeduplicates(t *testing.T) {
	modes, err := ParseMCPServerInstructionModes("NONE, descriptive, directive, prescriptive")
	if err != nil {
		t.Fatal(err)
	}
	want := []MCPServerInstructionMode{MCPServerInstructionsNone, MCPServerInstructionsDescriptive, MCPServerInstructionsPrescriptive}
	if !slices.Equal(modes, want) {
		t.Fatalf("modes = %v, want %v", modes, want)
	}
	for _, raw := range []string{"", ",none", "none,", "none,,descriptive", "unsupported"} {
		if _, err := ParseMCPServerInstructionModes(raw); err == nil {
			t.Errorf("ParseMCPServerInstructionModes(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestCodexFileChangeCapturedAsToolCall(t *testing.T) {
	t.Parallel()

	lines := []string{
		`{"type":"item.started","item":{"id":"edit","type":"file_change","changes":[{"path":"/tmp/api/server.go","kind":"update"}],"status":"in_progress"}}`,
		`{"type":"item.completed","item":{"id":"edit","type":"file_change","changes":[{"path":"/tmp/api/server.go","kind":"update"}],"status":"completed"}}`,
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

	if len(res.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(res.ToolCalls))
	}
	got := res.ToolCalls[0]
	if got.Name != "file_change" || got.TransportStatus != ToolCallStatusSucceeded || got.FunctionalStatus != ToolCallStatusSucceeded {
		t.Fatalf("file change call = %#v, want captured successful file_change", got)
	}
	if !isMutatingTool(got.Name) {
		t.Fatalf("file_change classified mutating = false, want true")
	}
	if tracker.initialLoadTurns != 0 {
		t.Fatalf("initial load turns = %d, want 0", tracker.initialLoadTurns)
	}
}

func TestStagedRenameFixtureEnforcesEachCumulativeCheckpoint(t *testing.T) {
	data, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "task_13_rename_followup_scale.txtar"))
	if readErr != nil {
		t.Fatalf("read staged fixture: %v", readErr)
	}
	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatalf("parse staged fixture: %v", parseErr)
	}
	if task.Metadata.InteractiveMode != "staged" || task.Metadata.StagedInitialTotalEdits != 1 || len(task.Metadata.StagedFollowups) != 4 {
		t.Fatalf("staged metadata = %+v", task.Metadata)
	}
	workDir := t.TempDir()
	if err := task.ExtractTo(workDir); err != nil {
		t.Fatalf("extract staged fixture: %v", err)
	}
	plan, planReadErr := os.ReadFile(filepath.Join(workDir, "rename-plan.md"))
	if planReadErr != nil {
		t.Fatalf("read rename plan: %v", planReadErr)
	}
	var oldNames, newNames []string
	for line := range strings.SplitSeq(string(plan), "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		parts := strings.Split(line, "`")
		if len(parts) >= 4 {
			oldNames = append(oldNames, parts[1])
			newNames = append(newNames, parts[3])
		}
	}
	if len(oldNames) != 16 {
		t.Fatalf("rename plan has %d mappings, want 16", len(oldNames))
	}
	sourcePath := filepath.Join(workDir, "pipeline", "workflow.go")
	source, sourceReadErr := os.ReadFile(sourcePath)
	if sourceReadErr != nil {
		t.Fatalf("read workflow: %v", sourceReadErr)
	}
	current := string(source)
	previous := 0
	for index, count := range []int{1, 2, 4, 8, 16} {
		for edit := previous; edit < count; edit++ {
			current = strings.ReplaceAll(current, oldNames[edit], newNames[edit])
		}
		if err := pipeline.WriteAtomic(sourcePath, []byte(current)); err != nil {
			t.Fatalf("write stage %d: %v", count, err)
		}
		stage := *task
		if index > 0 {
			stage.Metadata.Oracle = task.Metadata.StagedFollowups[index-1].Oracle
			if task.Metadata.StagedFollowups[index-1].TotalEdits != count {
				t.Fatalf("stage %d total edits = %d", index, task.Metadata.StagedFollowups[index-1].TotalEdits)
			}
		}
		result, err := Evaluate(context.Background(), &stage, workDir, []string{"pipeline/workflow.go"})
		if err != nil || !result.Passed {
			t.Fatalf("stage %d oracle = %+v, err=%v", count, result, err)
		}
		previous = count
	}
	early, err := Evaluate(context.Background(), task, workDir, []string{"pipeline/workflow.go"})
	if err != nil || early.Passed {
		t.Fatalf("first-stage oracle accepted all 16 edits: %+v, err=%v", early, err)
	}
}

func TestStagedFollowupsRejectNonIncreasingCounts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "task_13_rename_followup_scale.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	malformed := strings.Replace(string(data), "  - total_edits: 2", "  - total_edits: 1", 1)
	if _, err := ParseTask([]byte(malformed)); err == nil || !strings.Contains(err.Error(), "total_edits must exceed") {
		t.Fatalf("non-increasing stage count accepted: %v", err)
	}
	malformed = strings.Replace(string(data), "interactive_mode: staged", "interactive_mode: corrective", 1)
	if _, err := ParseTask([]byte(malformed)); err == nil || !strings.Contains(err.Error(), "unsupported interactive_mode") {
		t.Fatalf("unsupported stage mode accepted: %v", err)
	}
}

func TestParseTaskLoadsPrivateExplicitRecipe(t *testing.T) {
	fixture := []byte("task_id: explicit-private\nproblem_statement_file: .benchmark/issue.md\noracle:\n  level_3_build:\n    clean_compile: false\n  level_4_test:\n    pass_tests: false\n-- .benchmark/issue.md --\nOriginal issue\n-- .benchmark/explicit.md --\nCall semantic_batch with exact ordered edits.\n")
	task, err := ParseTask(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if task.Metadata.Instruction != "Original issue\n" || task.Metadata.PromptVariants["default"] != "Original issue\n" {
		t.Fatalf("issue prompt changed: %+v", task.Metadata)
	}
	if task.Metadata.PromptVariants["explicit"] != "Call semantic_batch with exact ordered edits.\n" {
		t.Fatalf("explicit recipe missing: %+v", task.Metadata.PromptVariants)
	}
	workDir := t.TempDir()
	if err := task.ExtractTo(workDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".benchmark", "explicit.md")); !os.IsNotExist(err) {
		t.Fatalf("private recipe extracted into workspace: %v", err)
	}
}

func TestMCPInstructionAdaptersPassEmptyInstructionsForNone(t *testing.T) {
	fixtureRoot := t.TempDir()
	runner := NewRunner(t.TempDir(), WithMCPServerInstructions(MCPServerInstructionsNone))
	codexOverride, err := runner.codexMCPServerInstructionsOverride(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	codexArgs := codexMCPArgsFromOverride(t, codexOverride)
	if index := slices.Index(codexArgs, "--instructions"); index < 0 || index+1 >= len(codexArgs) || codexArgs[index+1] != "" {
		t.Errorf("Codex none arguments = %#v, want --instructions followed by an empty value", codexArgs)
	}

	openCodeArgs, err := runner.openCodeMCPArguments(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	if index := slices.Index(openCodeArgs, "--instructions"); index < 0 || index+1 >= len(openCodeArgs) || openCodeArgs[index+1] != "" {
		t.Errorf("OpenCode none arguments = %#v, want --instructions followed by an empty value", openCodeArgs)
	}

	env, err := fixtureGoEnvironment(t.Context(), fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot := t.TempDir()
	if configErr := writeAgyMCPConfig(fixtureRoot, repositoryRoot, ArmSemedit, MCPServerInstructionsNone, env); configErr != nil {
		t.Fatal(configErr)
	}
	data, err := os.ReadFile(filepath.Join(fixtureRoot, ".agents", "mcp_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Servers map[string]struct {
			Args []string `json:"args"`
		} `json:"mcpServers"`
	}
	if decodeErr := json.Unmarshal(data, &config); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	server := config.Servers["semedit"]
	if index := slices.Index(server.Args, "--instructions"); index < 0 || index+1 >= len(server.Args) || server.Args[index+1] != "" {
		t.Errorf("Agy none arguments = %#v, want --instructions followed by an empty value", server.Args)
	}
}

func TestRunnerInstructionModesReachCodexMCPArgs(t *testing.T) {
	cases := []struct {
		name    string
		options []RunnerOption
		want    string
	}{
		{name: "omitted", want: mcp.PrescriptiveInstructions},
		{name: "empty", options: []RunnerOption{WithMCPServerInstructions("")}, want: mcp.PrescriptiveInstructions},
		{name: "none", options: []RunnerOption{WithMCPServerInstructions(MCPServerInstructionsNone)}},
		{name: "descriptive", options: []RunnerOption{WithMCPServerInstructions(MCPServerInstructionsDescriptive)}, want: mcp.DescriptiveInstructions},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runner := NewRunner(t.TempDir(), test.options...)
			override, err := runner.codexMCPServerInstructionsOverride(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			args := codexMCPArgsFromOverride(t, override)
			index := slices.Index(args, "--instructions")
			if index < 0 || index+1 >= len(args) || args[index+1] != test.want {
				t.Errorf("Codex MCP args = %#v, want --instructions followed by %q", args, test.want)
			}
		})
	}
}
