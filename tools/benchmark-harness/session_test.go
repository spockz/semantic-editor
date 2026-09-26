// This file exercises session boundaries because diagnostic turns must remain auditable without changing benchmark task measurements.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAgentSessionSeparatesDiagnosticTurnsFromTaskTelemetry(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	provider := &behavioralSessionProvider{}
	session, err := newAgentSession(provider, root, "session-test", workDir)
	if err != nil {
		t.Fatal(err)
	}
	taskResult := &RunResult{}
	taskStarted := time.Now()
	for _, prompt := range []string{"original", "correction"} {
		turn := &RunResult{}
		if err := session.runTurn(context.Background(), Target{}, prompt, "task", turn); err != nil {
			t.Fatal(err)
		}
		mergeTurn(taskResult, turn)
	}
	taskResult.WallClock = time.Since(taskStarted)
	frozenWallClock := taskResult.WallClock
	reflection := &RunResult{}
	if err := session.runTurn(context.Background(), Target{}, "diagnostic", "diagnostic_reflection", reflection); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 3 || provider.workDirs[0] != provider.workDirs[1] || provider.workDirs[1] != provider.workDirs[2] {
		t.Fatalf("provider did not retain one workspace/session: calls=%d workdirs=%v", provider.calls, provider.workDirs)
	}
	if provider.resumeIDs[0] != "" || provider.resumeIDs[1] != "session-id" || provider.resumeIDs[2] != "session-id" {
		t.Fatalf("resume state not carried through turns: %v", provider.resumeIDs)
	}
	if taskResult.Turns != 2 || taskResult.PromptTokens != 14 || taskResult.OutputTokens != 6 || taskResult.ToolCount != 2 || taskResult.WallClock != frozenWallClock {
		t.Fatalf("diagnostic telemetry affected task measurements: %+v", taskResult)
	}
	if reflection.Turns != 1000 || reflection.PromptTokens != 100000 || len(reflection.ToolCalls) != 500 {
		t.Fatalf("fake diagnostic did not emit large telemetry: %+v", reflection)
	}
	if err := os.RemoveAll(workDir); err != nil {
		t.Fatal(err)
	}
	transcript, err := os.ReadFile(session.transcriptPath)
	if err != nil {
		t.Fatalf("transcript did not survive workspace cleanup: %v", err)
	}
	var turns []sessionTurn
	if err := json.Unmarshal(transcript, &turns); err != nil {
		t.Fatal(err)
	}
	if len(turns) != 3 || turns[0].Classification != "task" || turns[1].Classification != "task" || turns[2].Classification != "diagnostic_reflection" {
		t.Fatalf("unexpected explicit transcript turns: %+v", turns)
	}
}

func TestParseAgyTranscriptRejectsMalformedEvent(t *testing.T) {
	result := &RunResult{}
	if err := parseAgyTranscript([]byte("{not-json}\n"), result); err == nil {
		t.Fatal("malformed transcript unexpectedly succeeded")
	}
}

func TestExecuteAgentUsesOneProcessSessionAndExcludesReflectionMetrics(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	countPath := filepath.Join(root, "count")
	workLog := filepath.Join(root, "workdirs")
	script := filepath.Join(bin, "codex")
	program := `#!/bin/sh
count=0
if [ -f "$SEMEDIT_SESSION_COUNT" ]; then count=$(cat "$SEMEDIT_SESSION_COUNT"); fi
count=$((count + 1))
printf '%s' "$count" > "$SEMEDIT_SESSION_COUNT"
pwd >> "$SEMEDIT_SESSION_WORKDIRS"
printf '{"type":"thread.started","thread_id":"fake-session"}\n'
printf '{"type":"turn.started"}\n'
case "$count" in
  1) printf 'package main\nfunc Old() {}\n' > main.go ;;
  2) printf 'package main\nfunc Done() {}\n' > main.go ;;
  3) sleep 0.05 ;;
esac
if [ "$count" -eq 3 ]; then
  printf '{"type":"item.completed","item":{"id":"diagnostic-tool","type":"mcp_tool_call","server":"semedit","tool":"semantic_insert_construct","status":"completed"}}\n'
fi
printf '{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"DONE"}}\n'
case "$count" in
  1) printf '{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":2}}\n' ;;
  2) printf '{"type":"turn.completed","usage":{"input_tokens":7,"output_tokens":2}}\n' ;;
  3) printf '{"type":"turn.completed","usage":{"input_tokens":100000,"output_tokens":100000}}\n' ;;
esac
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_SESSION_COUNT", countPath)
	t.Setenv("SEMEDIT_SESSION_WORKDIRS", workLog)
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:               "session-process",
			Instruction:          "Add Done.",
			InteractiveFollowups: []string{"Correct the implementation so Done exists."},
			Oracle:               OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"Done"}}},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Old() {}\n")),
	}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmSemedit, Variant: "small", Prompt: "original", Followups: []string{"Correct the implementation so Done exists."}, Policy: SemeditArmRestrictWrite}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Turns != 2 || result.PromptTokens != 12 || result.OutputTokens != 4 || result.ToolCount != 0 {
		t.Fatalf("task aggregates include non-task telemetry or missed correction: %+v", result)
	}
	count, err := os.ReadFile(countPath)
	if err != nil || string(count) != "3" {
		t.Fatalf("expected initial, followup, and reflection process calls; count=%q err=%v", count, err)
	}
	workdirs, err := os.ReadFile(workLog)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(workdirs))
	if len(lines) != 3 || lines[0] != lines[1] || lines[1] != lines[2] {
		t.Fatalf("provider process changed work directory across turns: %q", workdirs)
	}
	transcriptPath := result.Provenance["session_transcript"]
	transcript, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatalf("session evidence missing after cleanup: %v", err)
	}
	if strings.Count(string(transcript), "diagnostic_reflection") != 1 || strings.Count(string(transcript), "fake-session") < 3 {
		t.Fatalf("transcript did not retain task and diagnostic events: %s", transcript)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateMetric := range []string{"100000", "diagnostic-tool"} {
		if strings.Contains(string(encoded), privateMetric) {
			t.Fatalf("diagnostic telemetry leaked into measured result: %s", encoded)
		}
	}
}

func TestAgyResumeFailureKeepsOnlyNewTranscriptMetrics(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	bin := filepath.Join(root, "bin")
	logs := filepath.Join(home, ".gemini", "antigravity-cli", "brain", "conversation", ".system_generated", "logs")
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logs, 0o750); err != nil {
		t.Fatal(err)
	}
	countPath := filepath.Join(root, "count")
	transcriptPath := filepath.Join(logs, "transcript.jsonl")
	program := `#!/bin/sh
count=0
if [ -f "$SEMEDIT_AGY_COUNT" ]; then count=$(cat "$SEMEDIT_AGY_COUNT"); fi
count=$((count + 1))
printf '%s' "$count" > "$SEMEDIT_AGY_COUNT"
if [ "$count" -eq 1 ]; then
  printf '{"type":"PLANNER_RESPONSE"}\n{"type":"GENERIC","status":"done","tool_calls":[{"name":"semantic_lookup","args":{}},{"name":"semantic_replace_body","args":{}}]}\n' >> "$SEMEDIT_AGY_TRANSCRIPT"
  printf '{"conversation_id":"conversation","status":"SUCCESS","num_turns":1,"usage":{"input_tokens":5,"output_tokens":2}}\n'
else
  printf '{"type":"PLANNER_RESPONSE"}\n{"type":"GENERIC","status":"done","tool_calls":[{"name":"semantic_lookup","args":{}}]}\n{malformed' > "$SEMEDIT_AGY_TRANSCRIPT"
  printf '{"conversation_id":"conversation","status":"ERROR","error":"","num_turns":1,"usage":{"input_tokens":7,"output_tokens":3}}\n'
fi
`
	script := filepath.Join(bin, "agy")
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_AGY_COUNT", countPath)
	t.Setenv("SEMEDIT_AGY_TRANSCRIPT", transcriptPath)
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(filepath.Join(root, "scratch"))
	runner.agyExecutable = script
	provider := &agyProvider{runner: runner}
	first := &RunResult{Arm: ArmBaseline}
	if _, err := provider.run(context.Background(), workDir, Target{Harness: string(HarnessAgy)}, "first", first, ""); err != nil {
		t.Fatal(err)
	}
	second := &RunResult{Arm: ArmBaseline}
	_, err := provider.run(context.Background(), workDir, Target{Harness: string(HarnessAgy)}, "followup", second, "conversation")
	if err == nil || !strings.Contains(err.Error(), "decode Agy response and transcript") || !strings.Contains(err.Error(), "cumulative observations regressed") {
		t.Fatalf("expected parse and normalization errors, got %v", err)
	}
	if first.InternalTurns != 1 || len(first.ToolCalls) != 2 {
		t.Fatalf("first cumulative transcript totals mismatch: %+v", first)
	}
	if second.InternalTurns != 0 || len(second.ToolCalls) != 0 || second.ToolCount != 0 || second.MCPVerified {
		t.Fatalf("unreliable resumed transcript metrics were not withheld: %+v", second)
	}
	if second.PromptTokens != 7 || second.OutputTokens != 3 {
		t.Fatalf("response usage should remain independent of transcript parsing: %+v", second)
	}
}

func TestAgentSessionReportsTranscriptWriteFailure(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	provider := &behavioralSessionProvider{}
	session, err := newAgentSession(provider, root, "session-write-failure", workDir)
	if err != nil {
		t.Fatal(err)
	}
	session.transcriptPath = root
	err = session.runTurn(context.Background(), Target{}, "task", "task", &RunResult{})
	if err == nil || !strings.Contains(err.Error(), "record session transcript") {
		t.Fatalf("transcript write failure was not returned: %v", err)
	}
}

func TestCodexRejectsEmptyAndMalformedResumedStreams(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(bin, "codex")
	program := `#!/bin/sh
case "$SEMEDIT_CODEX_STREAM" in
  malformed) printf 'not-json\n' ;;
  null) printf 'null\n' ;;
  object) printf '{}\n' ;;
esac
exit 0
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := NewRunner(filepath.Join(root, "scratch"))
	for _, mode := range []string{"empty", "malformed", "null", "object"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("SEMEDIT_CODEX_STREAM", mode)
			result := &RunResult{Arm: ArmBaseline}
			_, err := runner.runCodex(context.Background(), root, Target{Harness: string(HarnessCodex)}, "followup", result, "resumed-thread")
			if err == nil {
				t.Fatal("empty or malformed resumed event stream unexpectedly succeeded")
			}
			if mode != "empty" && !strings.Contains(err.Error(), "malformed Codex event records") {
				t.Fatalf("malformed record diagnostic missing: %v", err)
			}
			if mode == "empty" && !strings.Contains(err.Error(), "no valid task events") {
				t.Fatalf("empty stream diagnostic missing: %v", err)
			}
		})
	}
}

func TestOpenCodeRejectsEmptyAndMalformedResumedStreams(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(bin, "opencode")
	program := `#!/bin/sh
case "$SEMEDIT_OPENCODE_STREAM" in
  malformed) printf 'not-json\n' ;;
  null) printf 'null\n' ;;
  object) printf '{}\n' ;;
esac
exit 0
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := NewRunner(filepath.Join(root, "scratch"))
	for _, mode := range []string{"empty", "malformed", "null", "object"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("SEMEDIT_OPENCODE_STREAM", mode)
			result := &RunResult{Arm: ArmBaseline}
			_, err := runner.runOpenCode(context.Background(), root, Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, "followup", result, "resumed-session")
			if err == nil {
				t.Fatal("empty or malformed resumed event stream unexpectedly succeeded")
			}
			if mode != "empty" && !strings.Contains(err.Error(), "malformed OpenCode event records") {
				t.Fatalf("malformed record diagnostic missing: %v", err)
			}
			if mode == "empty" && !strings.Contains(err.Error(), "no valid task events") {
				t.Fatalf("empty stream diagnostic missing: %v", err)
			}
		})
	}
}

type behavioralSessionProvider struct {
	calls     int
	workDirs  []string
	resumeIDs []string
}

func (p *behavioralSessionProvider) run(_ context.Context, workDir string, _ Target, prompt string, result *RunResult, resumeID string) (string, error) {
	p.calls++
	p.workDirs = append(p.workDirs, workDir)
	p.resumeIDs = append(p.resumeIDs, resumeID)
	result.Turns = 1
	result.PromptTokens = 7
	result.OutputTokens = 3
	result.ToolCalls = []ToolCall{{Name: "semantic_replace_body", TransportStatus: ToolCallStatusSucceeded}}
	result.ToolCount = 1
	result.agentResponse = prompt
	if prompt == "diagnostic" {
		result.Turns = 1000
		result.PromptTokens = 100000
		result.OutputTokens = 100000
		result.ToolCalls = make([]ToolCall, 500)
		time.Sleep(15 * time.Millisecond)
	}
	return "session-id", nil
}
