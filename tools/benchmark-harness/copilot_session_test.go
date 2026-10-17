// Exercises continued Copilot sessions through a real helper process and verifies observable accounting failures.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCopilotPersistentProcess(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestCopilotProtocolHelper$")
	command.Env = append(os.Environ(), "SEMEDIT_COPILOT_HELPER=normal")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := startCopilotSession(ctx, command, t.TempDir(), Target{Model: "test-model"}, ArmBaseline)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Error(err)
		}
	})
	pid := session.command.Process.Pid
	for index := 1; index <= 2; index++ {
		t.Run(fmt.Sprintf("turn-%d", index), func(t *testing.T) {
			result := &RunResult{}
			if err := session.run(ctx, "continue", result); err != nil {
				t.Fatal(err)
			}
			if session.command.Process.Pid != pid || session.sessionID != "test-session" {
				t.Fatal("session identity changed")
			}
			if result.PromptTokens != 10 || result.OutputTokens != 2 || result.InitialContextTokens != 10 || result.CachedPromptTokens != 0 {
				t.Fatalf("incorrect per-turn usage: %+v", result)
			}
			if result.toolObservationState != ToolObservationComplete || len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "semantic_lookup" || result.ToolCalls[0].FunctionalStatus != ToolCallStatusSucceeded {
				t.Fatalf("incorrect tool outcome: %+v", result)
			}
			if result.agentResponse != "DONE" || len(result.rawEvents) == 0 {
				t.Fatal("response or raw transcript missing")
			}
		})
	}
}

func TestCopilotProtocolFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		message string
	}{{"missing", "lacks required"}, {"mismatch", "cumulative"}, {"omitted", "omitted"}, {"identity", "session identity"}, {"tool-missing", "no completion"}, {"provider-error", "session.error"}, {"negative", "negative"}, {"cache-overflow", "cached tokens"}, {"fractional", "decode Copilot usage"}} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestCopilotProtocolHelper$")
			command.Env = append(os.Environ(), "SEMEDIT_COPILOT_HELPER="+test.name)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			session, err := startCopilotSession(ctx, command, t.TempDir(), Target{Model: "test-model"}, ArmBaseline)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if closeErr := session.close(); closeErr != nil {
					t.Error(closeErr)
				}
			})
			result := &RunResult{}
			err = session.run(ctx, "test", result)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error=%v, want %s", err, test.message)
			}
			if result.toolObservationState != ToolObservationPartial {
				t.Fatal("invalid stream marked complete")
			}
			if err := session.run(ctx, "retry", &RunResult{}); err == nil || !strings.Contains(err.Error(), "cannot continue") {
				t.Fatalf("failed session continued: %v", err)
			}
		})
	}
}

func TestCopilotUsageZeroAndAbsent(t *testing.T) {
	for _, test := range []struct {
		name   string
		wire   string
		absent bool
	}{{"explicit zero", `{"inputTokens":0,"outputTokens":0,"cacheReadTokens":0,"cacheWriteTokens":0,"reasoningTokens":0}`, false}, {"absent optional", `{"inputTokens":0,"outputTokens":0}`, true}} {
		t.Run(test.name, func(t *testing.T) {
			var usage copilotUsage
			if err := json.Unmarshal([]byte(test.wire), &usage); err != nil {
				t.Fatal(err)
			}
			result := &RunResult{}
			turn := &copilotTurn{result: result, usageCalls: 1}
			if err := turn.addUsage(usage); err != nil {
				t.Fatal(err)
			}
			if result.PromptTokens != 0 || result.OutputTokens != 0 {
				t.Fatal("zero counters changed")
			}
			if (result.Provenance["copilot_usage_cache_read"] == "unavailable") != test.absent {
				t.Fatalf("missing/zero distinction lost: %+v", result.Provenance)
			}
		})
	}
}

func TestCopilotProtocolHelper(_ *testing.T) {
	mode := os.Getenv("SEMEDIT_COPILOT_HELPER")
	if mode == "" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	turn := 0
	selected := false
	emit := func(value any) {
		if err := encoder.Encode(value); err != nil {
			os.Exit(2)
		}
	}
	for scanner.Scan() {
		var request struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(3)
		}
		var result any
		switch request.Method {
		case "initialize":
			result = copilotHelperNegotiationResult(request.Params, mode)
		case "session/new":
			result = copilotHelperSessionCatalog(mode)
		case "session/set_config_option":
			selected = copilotHelperSelectModel(request.Params, mode, emit, request.ID)
			if !selected {
				continue
			}
			result = copilotHelperSelectionResult(mode)
		case "session/prompt":
			if !selected || mode == "cloud-no-catalog" || mode == "cloud-unadvertised" || mode == "cloud-selection-not-applied" {
				os.Exit(11)
			}
			turn++
			result = copilotHelperPrompt(mode, turn, emit)
		default:
			os.Exit(6)
		}
		emit(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}
	if scanner.Err() != nil {
		os.Exit(7)
	}
	os.Exit(0)
}

func TestCopilotNegotiationAndModelEvidence(t *testing.T) {
	for _, test := range []struct {
		mode        string
		wantError   string
		uncached    int
		unavailable bool
	}{{"normal", "", 10, false}, {"cloud-no-catalog", "catalog unavailable", 0, false}, {"cloud-unadvertised", "not advertised", 0, false}, {"cloud-selection-not-applied", "did not confirm", 0, false}, {"protocol-version", "protocol version", 0, false}, {"selection-rejected", "model unavailable", 0, false}, {"wrong-model", "reported model", 0, false}, {"truncated", "stopped with", 0, false}, {"cache-write", "", 5, false}, {"optional", "", 0, true}} {
		t.Run(test.mode, func(t *testing.T) {
			result, err := executeCopilotHelperScenario(t, test.mode)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error=%v, want %s", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Provenance["copilot_agent_version"] != "test-version" || result.Provenance["copilot_protocol_version"] != "1" {
				t.Fatalf("missing negotiation evidence: %+v", result.Provenance)
			}
			if result.UncachedPromptTokens != test.uncached || (result.Provenance["copilot_usage_uncached"] == "unavailable") != test.unavailable {
				t.Fatalf("incorrect uncached usage: tokens=%d provenance=%+v", result.UncachedPromptTokens, result.Provenance)
			}
		})
	}
}

func copilotHelperUsage(mode string, turn int) map[string]any {
	usage := map[string]any{"model": "test-model", "inputTokens": 10, "outputTokens": 2, "cacheReadTokens": 0, "cacheWriteTokens": 0, "reasoningTokens": 0}
	switch mode {
	case "cache-write":
		usage["cacheReadTokens"] = 2
		usage["cacheWriteTokens"] = 3
	case "optional":
		delete(usage, "cacheWriteTokens")
		delete(usage, "reasoningTokens")
	case "wrong-model":
		usage["model"] = "unexpected-model"
	case "truncated":
		usage["finishReason"] = "length"
	case "missing":
		delete(usage, "inputTokens")
	case "negative":
		usage["outputTokens"] = -1
	case "cache-overflow":
		usage["cacheReadTokens"] = 11
	case "fractional":
		usage["inputTokens"] = 0.5
	}
	if mode == "later-optional-missing" && turn > 1 {
		delete(usage, "cacheReadTokens")
		delete(usage, "cacheWriteTokens")
		delete(usage, "reasoningTokens")
	}
	if mode == "diagnostic-metrics" && turn > 2 {
		usage["inputTokens"] = 100000
		usage["outputTokens"] = 100000
		usage["cacheReadTokens"] = 30000
		usage["cacheWriteTokens"] = 20000
		usage["reasoningTokens"] = 10000
	}
	return usage
}

func executeCopilotHelperScenario(t *testing.T, mode string) (*RunResult, error) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestCopilotProtocolHelper$")
	command.Env = append(os.Environ(), "SEMEDIT_COPILOT_HELPER="+mode)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result := &RunResult{}
	session, err := startCopilotSession(ctx, command, t.TempDir(), Target{Model: "test-model"}, ArmBaseline)
	if err != nil {
		return result, err
	}
	t.Cleanup(func() {
		if closeErr := session.close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	return result, session.run(ctx, "test", result)
}

func TestCopilotFollowupUsageAvailability(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestCopilotProtocolHelper$")
	command.Env = append(os.Environ(), "SEMEDIT_COPILOT_HELPER=later-optional-missing")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := startCopilotSession(ctx, command, t.TempDir(), Target{Model: "test-model"}, ArmBaseline)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := session.close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	first, second := &RunResult{}, &RunResult{}
	if runErr := session.run(ctx, "first", first); runErr != nil {
		t.Fatal(runErr)
	}
	if first.Provenance["copilot_usage_cache_read"] == "unavailable" {
		t.Fatal("first turn lost explicit cache-read zero")
	}
	if runErr := session.run(ctx, "second", second); runErr != nil {
		t.Fatal(runErr)
	}
	mergeTurn(first, second)
	for _, name := range []string{"cache_read", "cache_write", "reasoning", "uncached"} {
		if first.Provenance["copilot_usage_"+name] != "unavailable" {
			t.Errorf("aggregate lost unavailable %s: %+v", name, first.Provenance)
		}
	}
	if first.PromptTokens != 20 || first.OutputTokens != 4 {
		t.Fatalf("aggregate token totals: prompt=%d output=%d", first.PromptTokens, first.OutputTokens)
	}
}

func emitCopilotHelperEvent(emit func(any), mode, kind string, data any) {
	sessionID := "test-session"
	if mode == "identity" {
		sessionID = "other-session"
	}
	params := map[string]any{"sessionId": sessionID, "type": kind, "data": data}
	if mode == "omitted" && kind == "assistant.usage" {
		params["dataOmitted"] = "too-large"
	}
	emit(map[string]any{"jsonrpc": "2.0", "method": "github.com/copilot/sessionEvent", "params": params})
}

func copilotHelperNegotiationResult(raw json.RawMessage, mode string) any {
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		os.Exit(4)
	}
	caps, ok := params["clientCapabilities"].(map[string]any)
	if !ok || caps["_meta"] == nil {
		os.Exit(5)
	}
	version := 1
	if mode == "protocol-version" {
		version = 2
	}
	return map[string]any{"protocolVersion": version, "agentInfo": map[string]string{"name": "Copilot", "version": "test-version"}}
}

func TestCopilotSelectsLocalWireModelBeforePrompt(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestCopilotProtocolHelper$")
	command.Env = append(os.Environ(), "SEMEDIT_COPILOT_HELPER=local-no-catalog")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := startCopilotSession(ctx, command, t.TempDir(), Target{Harness: string(HarnessCopilot), Model: "local/test-model"}, ArmBaseline)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := session.close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	result := &RunResult{}
	if runErr := session.run(ctx, "test", result); runErr != nil {
		t.Fatal(runErr)
	}
	if !result.processStartedAt.Equal(session.startedAt) {
		t.Fatal("first turn omitted process initialization time")
	}
	if result.Provenance["copilot_model_selection"] != "acp_session_set_config_option" {
		t.Fatal("missing explicit model selection evidence")
	}
}

func emitCopilotHelperText(mode string, emit func(any)) {
	if mode == "tool-update-array" {
		emit(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{"sessionId": "test-session", "update": map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "call", "content": []any{map[string]any{"type": "content", "content": map[string]string{"type": "text", "text": "tool output"}}}}}})
	}
	parts := []string{"DO", "NE"}
	if mode == "large-response" {
		parts = []string{strings.Repeat("A", 20000), strings.Repeat("B", 20000)}
	}
	for _, part := range parts {
		emit(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{"sessionId": "test-session", "update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]string{"type": "text", "text": part}}}})
	}
}

func TestCopilotLargeStandardResponse(t *testing.T) {
	result, err := executeCopilotHelperScenario(t, "large-response")
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.Repeat("A", 20000) + strings.Repeat("B", 20000)
	if result.agentResponse != expected {
		t.Fatalf("standard response length=%d, want %d complete ordered bytes", len(result.agentResponse), len(expected))
	}
	if result.toolObservationState != ToolObservationComplete {
		t.Fatal("complete standard response lost tool observation evidence")
	}
}

func copilotHelperSelectModel(params json.RawMessage, mode string, emit func(any), id int) bool {
	var selection struct {
		ConfigID  string `json:"configId"`
		Value     string `json:"value"`
		SessionID string `json:"sessionId"`
	}
	if decodeErr := json.Unmarshal(params, &selection); decodeErr != nil {
		os.Exit(9)
	}
	if selection.ConfigID != "model" || selection.Value != "test-model" || selection.SessionID != "test-session" {
		os.Exit(10)
	}
	if mode == "selection-rejected" {
		emit(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32602, "message": "model unavailable"}})
		return false
	}
	return true
}

func TestCopilotStandardToolUpdateContent(t *testing.T) {
	result, err := executeCopilotHelperScenario(t, "tool-update-array")
	if err != nil {
		t.Fatal(err)
	}
	if result.agentResponse != "DONE" {
		t.Fatalf("tool-update content corrupted assistant response: %q", result.agentResponse)
	}
	if len(result.ToolCalls) != 1 || result.toolObservationState != ToolObservationComplete {
		t.Fatal("array-valued standard tool update broke observation capture")
	}
}

func copilotHelperSessionCatalog(mode string) map[string]any {
	result := map[string]any{"sessionId": "test-session"}
	if mode == "cloud-no-catalog" || mode == "local-no-catalog" {
		return result
	}
	model := "test-model"
	if mode == "cloud-unadvertised" {
		model = "another-model"
	}
	result["models"] = map[string]any{"currentModelId": model, "availableModels": []any{map[string]string{"modelId": model, "name": model}}}
	return result
}

func copilotHelperSelectionResult(mode string) map[string]any {
	if mode == "local-no-catalog" {
		return map[string]any{}
	}
	model := "test-model"
	if mode == "cloud-selection-not-applied" {
		model = "another-model"
	}
	return map[string]any{"configOptions": []any{map[string]string{"id": "model", "currentValue": model}}}
}

func copilotHelperPrompt(mode string, turn int, emit func(any)) map[string]any {
	if mode == "diagnostic-crash" && turn > 2 {
		os.Exit(17)
	}
	count := 1
	if mode == "diagnostic-metrics" && turn > 2 {
		time.Sleep(30 * time.Millisecond)
		count = 3
	}
	for index := range count {
		id := "call"
		if index > 0 {
			id = fmt.Sprintf("call-%d", index)
		}
		emitCopilotHelperTool(mode, id, emit)
	}
	emitCopilotHelperEvent(emit, mode, "assistant.usage", copilotHelperUsage(mode, turn))
	emitCopilotHelperText(mode, emit)
	if mode == "provider-error" {
		emitCopilotHelperEvent(emit, mode, "session.error", map[string]any{"message": "unavailable"})
	}
	input, output := turn*10, turn*2
	if mode == "diagnostic-metrics" && turn > 2 {
		input = 20 + (turn-2)*100000
		output = 4 + (turn-2)*100000
	}
	if mode == "mismatch" {
		input++
	}
	return map[string]any{"stopReason": "end_turn", "usage": map[string]any{"inputTokens": input, "outputTokens": output}}
}

func emitCopilotHelperTool(mode, id string, emit func(any)) {
	emit(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{"sessionId": "test-session", "update": map[string]any{"sessionUpdate": "tool_call", "toolCallId": id}}})
	emitCopilotHelperEvent(emit, mode, "tool.execution_start", map[string]any{"toolCallId": id, "toolName": "semedit-semantic_lookup", "mcpToolName": "semantic_lookup", "mcpServerName": "semedit", "arguments": map[string]any{"symbol": "Foo"}})
	if mode != "tool-missing" {
		emitCopilotHelperEvent(emit, mode, "tool.execution_complete", map[string]any{"toolCallId": id, "success": true, "result": map[string]any{"content": "found"}})
	}
}

func newCopilotDiagnosticTestSession(t *testing.T, mode string) (context.Context, *agentSession, *RunResult) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	base := t.TempDir()
	workDir := base + "/workspace"
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	target := Target{Harness: string(HarnessCopilot), Model: "local/test-model"}
	command := exec.Command(os.Args[0], "-test.run=^TestCopilotProtocolHelper$")
	command.Env = append(os.Environ(), "SEMEDIT_COPILOT_HELPER="+mode)
	live, err := startCopilotSession(ctx, command, workDir, target, ArmSemeditExplicit)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := live.close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	provider := &copilotProvider{session: live, config: adapterSessionConfig{WorkDir: workDir, Target: target, Arm: ArmSemeditExplicit}}
	session, err := newAgentSession(provider, base, base, "copilot-diagnostics", workDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := session.closeRoots(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	aggregate := &RunResult{Target: target, Arm: ArmSemeditExplicit}
	session.execution = AgentExecution{Task: &Task{}, Target: target, Arm: ArmSemeditExplicit}
	session.result = aggregate
	session.adapterPrepared = true
	session.adapterWarmed = true
	for index := range 2 {
		turn := aggregate
		if index > 0 {
			turn = &RunResult{Target: target, Arm: ArmSemeditExplicit}
		}
		if runErr := session.runTurn(ctx, "work", sessionTaskTurn, turn); runErr != nil {
			t.Fatal(runErr)
		}
		if index > 0 {
			mergeTurn(aggregate, turn)
		}
	}
	aggregate.Success = true
	aggregate.WallClock = time.Since(aggregate.processStartedAt)
	return ctx, session, aggregate
}

func TestCopilotDiagnosticMetricsStaySeparate(t *testing.T) {
	ctx, session, aggregate := newCopilotDiagnosticTestSession(t, "diagnostic-metrics")
	frozen, err := json.Marshal(aggregate)
	if err != nil {
		t.Fatal(err)
	}
	diagnosticBoundary := time.Now()
	for range 2 {
		reflection := session.requestSemanticToolReflection(ctx, "diagnose")
		if reflection.Error != "" {
			t.Fatal(reflection.Error)
		}
		if reflection.WallClock < 30*time.Millisecond || len(reflection.ToolCalls) != 3 {
			t.Fatalf("diagnostic reflection measurements missing: %+v", reflection)
		}
	}
	after, err := json.Marshal(aggregate)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(frozen) {
		t.Fatal("diagnostic turns changed frozen work result")
	}
	if aggregate.WallClock > diagnosticBoundary.Sub(aggregate.processStartedAt) {
		t.Fatal("work wall clock includes diagnostic time")
	}
	if aggregate.PromptTokens != 20 || aggregate.OutputTokens != 4 || aggregate.ToolCount != 2 || aggregate.CachedPromptTokens != 0 || aggregate.ReasoningTokens != 0 {
		t.Fatalf("work metrics contaminated: %+v", aggregate)
	}
	if closeErr := session.close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	data, err := os.ReadFile(session.transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	var turns []sessionTurn
	if decodeErr := json.Unmarshal(data, &turns); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(turns) != 4 {
		t.Fatalf("retained %d turns, want4", len(turns))
	}
	for index, turn := range turns[2:] {
		t.Run(fmt.Sprintf("diagnostic-%d", index+1), func(t *testing.T) { assertCopilotDiagnosticMetrics(t, turn) })
	}
}

func assertCopilotDiagnosticMetrics(t *testing.T, turn sessionTurn) {
	t.Helper()
	observation := turn.Observation
	if turn.Classification != "diagnostic_reflection" {
		t.Fatalf("diagnostic metrics missing: %+v", turn)
	}
	if observation.PromptTokens != 100000 || observation.OutputTokens != 100000 || observation.CachedPromptTokens != 30000 || observation.UncachedPromptTokens != 50000 || observation.ReasoningTokens != 10000 || len(observation.ToolCalls) != 3 {
		t.Fatalf("diagnostic accounting lost: %+v", observation)
	}
	if turn.WallClock < 30*time.Millisecond {
		t.Fatalf("diagnostic delay lost: %v", turn.WallClock)
	}
	if len(observation.RawEvents) == 0 || observation.Response != "DONE" {
		t.Fatal("diagnostic transcript evidence incomplete")
	}
}

func TestCopilotDiagnosticCrashPreservesWorkVerdict(t *testing.T) {
	ctx, session, aggregate := newCopilotDiagnosticTestSession(t, "diagnostic-crash")
	frozenPrompt, frozenOutput, frozenWall := aggregate.PromptTokens, aggregate.OutputTokens, aggregate.WallClock
	reflection := session.requestSemanticToolReflection(ctx, "diagnose")
	if reflection.Error == "" {
		t.Fatal("diagnostic subprocess crash was accepted")
	}
	if err := session.close(); err != nil {
		t.Fatalf("diagnostic exit contaminated cleanup: %v", err)
	}
	if !aggregate.Success || aggregate.Error != "" || aggregate.PromptTokens != frozenPrompt || aggregate.OutputTokens != frozenOutput || aggregate.WallClock != frozenWall {
		t.Fatalf("diagnostic crash changed work verdict or metrics: %+v", aggregate)
	}
	data, err := os.ReadFile(session.transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	var turns []sessionTurn
	if decodeErr := json.Unmarshal(data, &turns); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(turns) != 3 {
		t.Fatalf("retained %d turns, want3", len(turns))
	}
	diagnosticTurn := turns[2]
	if diagnosticTurn.Classification != "diagnostic_reflection" || diagnosticTurn.ProviderError == "" || !strings.Contains(diagnosticTurn.ProviderExit, "17") {
		t.Fatalf("diagnostic crash evidence lost: %+v", diagnosticTurn)
	}
}
