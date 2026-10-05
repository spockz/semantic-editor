// OpenCode process tests prove task usage is complete and diagnostic questions cannot change solving measurements.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeUsageProcessAccounting(t *testing.T) {
	t.Setenv(openRouterAPIKeyEnv, "token-to-redact")
	bin := t.TempDir()
	program := "#!/bin/sh\nprintf '%s\\n' \"$SEMEDIT_OPENCODE_EVENTS\"\n"
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	first := `{"type":"step_finish","part":{"type":"step-finish","id":"finish-1","messageID":"message-1","reason":"tool-calls","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}}`
	duplicate := `{"type":"message.part.updated","properties":{"part":{"type":"step-finish","id":"finish-1","messageID":"message-1","reason":"tool-calls","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}}}`
	second := `{"type":"step_finish","part":{"type":"step-finish","id":"finish-2","messageID":"message-1","reason":"stop","tokens":{"input":11,"output":13,"reasoning":17,"cache":{"read":19,"write":23}}}}`
	zero := `{"type":"step_finish","part":{"type":"step-finish","id":"zero","reason":"stop","tokens":{"input":0,"output":0,"reasoning":0,"cache":{"read":0,"write":0}}}}`
	for _, test := range []struct {
		name      string
		events    string
		wantError string
		prompt    int
		cached    int
		uncached  int
		output    int
		reasoning int
		initial   int
	}{
		{name: "distinct steps and duplicate part", events: strings.Join([]string{first, duplicate, second}, "\n"), prompt: 68, cached: 24, uncached: 44, output: 17, reasoning: 19, initial: 15},
		{name: "late duplicate part", events: strings.Join([]string{first, second, duplicate}, "\n"), prompt: 68, cached: 24, uncached: 44, output: 17, reasoning: 19, initial: 15},
		{name: "conflicting duplicate usage", events: first + "\n" + strings.Replace(first, `"input":3`, `"input":4`, 1) + "\n" + second, wantError: "conflicting duplicate usage"},
		{name: "conflicting duplicate reason", events: first + "\n" + strings.Replace(first, `"reason":"tool-calls"`, `"reason":"stop"`, 1), wantError: "conflicting duplicate usage"},
		{name: "reported zero", events: zero},
		{name: "missing finish", events: `{"type":"step_start","part":{"type":"step-start","id":"start"}}`, wantError: "complete terminal token usage"},
		{name: "unfinished tool loop", events: first, wantError: "complete terminal token usage"},
		{name: "missing usage", events: `{"type":"step_finish","part":{"type":"step-finish","id":"finish","reason":"stop"}}`, wantError: "lacks required"},
		{name: "null counter", events: strings.Replace(zero, `"input":0`, `"input":null`, 1), wantError: "lacks required"},
		{name: "negative counter", events: strings.Replace(zero, `"input":0`, `"input":-1`, 1), wantError: "negative counter"},
		{name: "fractional counter", events: strings.Replace(zero, `"input":0`, `"input":0.5`, 1), wantError: "decode OpenCode token usage"},
		{name: "provider failure after usage", events: zero + "\n" + `{"type":"error","error":{"message":"provider unavailable"}}`, wantError: "provider unavailable"},
		{name: "credential redaction", events: zero + "\n" + `{"type":"error","error":{"data":{"message":"service rejected token-to-redact"},"request":{"secret":"token-to-redact"}}}`, wantError: "service rejected [REDACTED]"},
		{name: "missing intermediate usage", events: strings.Join([]string{`{"type":"step_start","part":{"type":"step-start","id":"start-1"}}`, `{"type":"step_start","part":{"type":"step-start","id":"start-2"}}`, zero}, "\n"), wantError: "complete terminal token usage"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SEMEDIT_OPENCODE_EVENTS", `{"type":"session.created","properties":{"info":{"id":"usage-session"}}}`+"\n"+test.events)
			result := &RunResult{Arm: ArmBaseline}
			session, err := NewRunner(t.TempDir()).runOpenCode(t.Context(), t.TempDir(), Target{Harness: string(HarnessOpenCode), Model: openCodeDefaultModel}, "solve", result, "")
			if test.wantError != "" {
				if err != nil && strings.Contains(err.Error(), "token-to-redact") {
					t.Fatal("credential leaked into provider error")
				}
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if session != "usage-session" || result.PromptTokens != test.prompt || result.CachedPromptTokens != test.cached || result.UncachedPromptTokens != test.uncached || result.OutputTokens != test.output || result.ReasoningTokens != test.reasoning || result.InitialContextTokens != test.initial {
				t.Fatalf("incorrect process usage: session=%s result=%+v", session, result)
			}
		})
	}
}

func TestExecuteAgentOpenCodeExcludesDiagnosticUsage(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	countPath := filepath.Join(root, "count")
	configPath := filepath.Join(root, "captured-config.json")
	program := `#!/bin/sh
set -eu
count=0
if [ -f "$SEMEDIT_SESSION_COUNT" ]; then count=$(cat "$SEMEDIT_SESSION_COUNT"); fi
count=$((count + 1))
printf '%s' "$count" > "$SEMEDIT_SESSION_COUNT"
cp "$OPENCODE_CONFIG" "$SEMEDIT_CAPTURE_CONFIG"
model=''
resume=''
while [ "$#" -gt 0 ]; do
 case "$1" in
  --model) shift; model=$1 ;;
  --session) shift; resume=$1 ;;
  --variant) exit 78 ;;
 esac
 shift
done
[ "$model" = 'amdbeast/qwen36-iq4-xs' ] || exit 79
if [ "$count" -gt 1 ]; then [ "$resume" = 'fake-session' ] || exit 80; else [ -z "$resume" ] || exit 81; fi
printf '{"type":"step_start","sessionID":"fake-session","part":{"id":"start-%s","type":"step-start"}}\n' "$count"
case "$count" in
 1) printf 'package main\nfunc Old() {}\n' > main.go
    input=5; read=3; write=1; output=2; reasoning=1; tool=semantic_lookup ;;
 2) printf 'package main\nfunc Done() {}\n' > main.go
    input=7; read=4; write=2; output=2; reasoning=2; tool=semantic_replace_body ;;
 3|4) sleep 0.15
    input=100000; read=100000; write=100000; output=100000; reasoning=100000; tool=semantic_insert_construct ;;
 *) exit 82 ;;
esac
printf '{"type":"tool_use","part":{"id":"tool-%s","type":"tool","tool":"%s","server":"semedit","state":{"status":"completed","input":{},"output":"ok"}}}\n' "$count" "$tool"
printf '{"type":"text","part":{"type":"text","text":"I would ask my owner to add semedit because it helped this task."}}\n'
printf '{"type":"step_finish","part":{"id":"finish-%s","type":"step-finish","reason":"stop","tokens":{"input":%s,"output":%s,"reasoning":%s,"cache":{"read":%s,"write":%s}}}}\n' "$count" "$input" "$output" "$reasoning" "$read" "$write"
`
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_SESSION_COUNT", countPath)
	t.Setenv("SEMEDIT_CAPTURE_CONFIG", configPath)
	task := &Task{Metadata: TaskMetadata{TaskID: "opencode-session", Instruction: "Add Done.", Oracle: OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"Done"}}}}, Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Old() {}\n"))}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessOpenCode), Model: openCodeDefaultModel}, Arm: ArmSemedit, Variant: "small", Prompt: "original", Followups: []string{"Correct the implementation so Done exists."}, Policy: SemeditArmRestrictWrite}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(t.Context(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Turns != 2 || result.PromptTokens != 22 || result.CachedPromptTokens != 7 || result.UncachedPromptTokens != 15 || result.OutputTokens != 4 || result.ReasoningTokens != 3 || result.ToolCount != 2 || result.InitialContextTokens != 9 {
		t.Fatalf("solving metrics include diagnostic usage or missed a correction: %+v", result)
	}
	assertOpenCodeDiagnosticEvidence(t, result, countPath, configPath)
}

func assertOpenCodeDiagnosticEvidence(t *testing.T, result *RunResult, countPath, configPath string) {
	t.Helper()
	if got := result.Target.String(); got != "opencode/qwen36-iq4-xs" {
		t.Fatalf("benchmark target = %q, want opencode/qwen36-iq4-xs", got)
	}
	if result.SemanticBatchReflection == nil || result.SemanticIntegrationReflection == nil || !strings.Contains(result.SemanticIntegrationReflection.Response, "ask my owner") {
		t.Fatalf("missing diagnostic answers: %+v", result)
	}
	count, err := os.ReadFile(countPath)
	if err != nil || string(count) != "4" {
		t.Fatalf("process count=%q error=%v", count, err)
	}
	data, err := os.ReadFile(result.Provenance["session_transcript"])
	if err != nil {
		t.Fatal(err)
	}
	var turns []sessionTurn
	if decodeErr := json.Unmarshal(data, &turns); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(turns) != 4 || turns[0].Classification != "task" || turns[1].Classification != "task" || turns[2].Classification != "diagnostic_reflection" || turns[3].Classification != "diagnostic_reflection" {
		t.Fatalf("incorrect turn classifications: %+v", turns)
	}
	if result.processStartedAt.IsZero() || result.WallClock <= 0 || result.WallClock > turns[2].StartedAt.Sub(result.processStartedAt) {
		t.Fatalf("task clock extends into diagnostic questioning: %s", result.WallClock)
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Tools map[string]bool `json:"tools"`
	}
	if decodeErr := json.Unmarshal(configData, &config); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	enabled, present := config.Tools["task"]
	if !present || enabled {
		t.Fatal("delegation must be disabled because child-session usage is absent from OpenCode JSON")
	}
}

func TestOpenCodePrefixedSemanticEventsDriveReflectionChoices(t *testing.T) {
	parse := func(names ...string) *RunResult {
		events := make([]string, 0, len(names))
		for index, name := range names {
			events = append(events, openCodeSemanticTerminalEvent(index, name))
		}
		result := &RunResult{Success: true}
		stream := readOpenCodeEventStream(strings.NewReader(strings.Join(events, "\n")), result, "session-123")
		if stream.decodeErrors != 0 || len(result.ToolCalls) != len(names) {
			t.Fatalf("OpenCode adapter parsed %d calls with %d decode errors: %+v", len(result.ToolCalls), stream.decodeErrors, result.ToolCalls)
		}
		refreshMCPVerified(result)
		return result
	}

	t.Run("successful semantic event prevents non-use reflection", func(t *testing.T) {
		result := parse("semedit_semantic_replace_body")
		if !result.MCPVerified {
			t.Fatal("OpenCode semedit semantic event did not verify MCP use")
		}
		if shouldRequestSemanticToolReflection(ArmSemedit, "session-123", result) {
			t.Fatal("successful prefixed semantic call triggered the non-use reflection")
		}
		if shouldRequestSemanticBatchReflection(ArmSemedit, "session-123", result) {
			t.Fatal("single semantic call triggered the batch reflection")
		}
	})

	t.Run("consecutive prefixed semantic events trigger batch reflection", func(t *testing.T) {
		result := parse("semedit_semantic_rename", "semedit_semantic_insert_function")
		if !result.MCPVerified || !shouldRequestSemanticBatchReflection(ArmSemedit, "session-123", result) {
			t.Fatalf("consecutive OpenCode semantic calls were not recognized for batching: %+v", result)
		}
		if shouldRequestSemanticToolReflection(ArmSemedit, "session-123", result) {
			t.Fatal("recognized semantic calls triggered the non-use reflection")
		}
	})

	t.Run("explicit prefixed batch suppresses batch reflection", func(t *testing.T) {
		result := parse("semedit_semantic_rename", "semedit_semantic_insert_function", "semedit_semantic_batch")
		if !result.MCPVerified || shouldRequestSemanticBatchReflection(ArmSemedit, "session-123", result) {
			t.Fatalf("explicit OpenCode semantic_batch did not suppress batch reflection: %+v", result)
		}
	})
}

func openCodeSemanticTerminalEvent(index int, name string) string {
	event := map[string]any{
		"type": "tool_use",
		"part": map[string]any{
			"id":   fmt.Sprintf("call-%d", index),
			"type": "tool",
			"tool": name,
			"state": map[string]any{
				"status": "completed",
				"input":  map[string]any{},
				"output": "ok",
			},
		},
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		panic("encode OpenCode semantic terminal event: " + err.Error())
	}
	return string(encoded)
}

func TestSemanticToolProviderPrefixesNormalizeConsistently(t *testing.T) {
	for _, test := range []struct {
		caseName, name, server, want string
	}{
		{caseName: "CLI semantic name", name: "semantic_rename", want: "semantic_rename"},
		{caseName: "slash server prefix", name: "semedit/semantic_rename", want: "semantic_rename"},
		{caseName: "MCP server prefix", name: "mcp__semedit__semantic_rename", want: "semantic_rename"},
		{caseName: "OpenCode prefix", name: "semedit_semantic_rename", want: "semantic_rename"},
		{caseName: "server identity", name: "lookup", server: "semedit", want: "lookup"},
	} {
		t.Run(test.caseName, func(t *testing.T) {
			if got := semanticToolBaseName(test.name); got != test.want {
				t.Fatalf("base name = %q, want %q", got, test.want)
			}
			if !isSemanticTool(test.name, test.server) {
				t.Fatalf("isSemanticTool(%q, %q) = false", test.name, test.server)
			}
		})
	}
	if isSemanticTool("shell", "local") {
		t.Fatal("ordinary shell tool was classified as semantic")
	}
}
