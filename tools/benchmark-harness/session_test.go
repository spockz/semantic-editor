// This file exercises session boundaries because diagnostic turns must remain auditable without changing benchmark task measurements.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
  1) printf 'package main\nfunc Old() {}\n' > main.go
     printf '{"type":"item.completed","item":{"id":"lookup","type":"mcp_tool_call","server":"semedit","tool":"semantic_lookup","status":"completed"}}\n'
     printf '{"type":"item.completed","item":{"id":"unnamed","type":"mcp_tool_call","status":"completed"}}\n' ;;
  2) printf 'package main\nfunc Done() {}\n' > main.go
     printf '{"type":"item.completed","item":{"id":"replace","type":"mcp_tool_call","server":"semedit","tool":"semantic_replace_body","status":"failed"}}\n' ;;
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
			Oracle:               OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"Done"}}, DiagnosticExpectedTools: []string{"semantic_lookup", "semantic_replace_body", "semantic_rename"}},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Old() {}\n")),
	}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmSemedit, Variant: "small", Prompt: "original", Followups: []string{"Correct the implementation so Done exists."}, Policy: SemeditArmRestrictWrite}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Turns != 2 || result.PromptTokens != 12 || result.OutputTokens != 4 || result.ToolCount != 2 || result.DiagnosticToolCoverage == nil || result.DiagnosticToolCoverage.Complete || result.DiagnosticToolCoverage.ObservationState != ToolObservationPartial || result.DiagnosticToolCoverage.ExpectedSatisfied || result.DiagnosticToolCoverage.Missing != nil || !slices.Equal(result.DiagnosticToolCoverage.Observed, []string{"semantic_lookup", "semantic_replace_body"}) {
		t.Fatalf("task aggregates include non-task telemetry or missed correction: %+v", result)
	}
	if slices.Contains(result.DiagnosticToolCoverage.Observed, "semantic_insert_construct") {
		t.Fatal("diagnostic reflection tool leaked into measured tool coverage")
	}
	if result.ToolCalls[1].FunctionalStatus != ToolCallStatusFailed {
		t.Fatalf("failed-call outcome was lost: coverage=%#v call=%#v", result.DiagnosticToolCoverage, result.ToolCalls[1])
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
	if !strings.Contains(string(encoded), `"diagnostic_tool_coverage"`) || !strings.Contains(string(encoded), `"observation_state":"partial"`) || !strings.Contains(string(encoded), `"missing":null`) {
		t.Fatalf("diagnostic tool coverage was not serialized: %s", encoded)
	}
	for _, privateMetric := range []string{"100000", "diagnostic-tool"} {
		if strings.Contains(string(encoded), privateMetric) {
			t.Fatalf("diagnostic telemetry leaked into measured result: %s", encoded)
		}
	}
}

func TestExecuteAgentSkipsPrewarmWithoutGoTargetAndTimesCodexProcess(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(bin, "codex")
	program := `#!/bin/sh
sleep 0.03
printf '{"type":"thread.started","thread_id":"fake-session"}\n'
printf '{"type":"turn.started"}\n'
printf '{"type":"item.completed","item":{"id":"verify","type":"mcp_tool_call","server":"semedit","tool":"semantic_verify","status":"completed"}}\n'
printf '{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"hi"}}\n'
printf '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}\n'
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	data, err := os.ReadFile("../../testdata/bench/task_00_hi_overhead.txtar")
	if err != nil {
		t.Fatal(err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatal(err)
	}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmSemedit, Variant: "small", Prompt: task.Metadata.Instruction, Policy: SemeditArmRestrictWrite}
	result, err := NewRunner(filepath.Join(root, "scratch"), WithSemeditPrewarmVerify(true)).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("empty fixture run failed: %s", result.Error)
	}
	if result.SemeditPrewarmVerify || result.Provenance["semedit_prewarm_skipped"] == "" {
		t.Fatalf("prewarm should skip a fixture without a Go target: %+v", result)
	}
	if result.processStartedAt.IsZero() || result.WallClock < 30*time.Millisecond {
		t.Fatalf("wall clock must include Codex execution from its process start: start=%s duration=%s", result.processStartedAt, result.WallClock)
	}
	if result.WallClock > time.Since(result.processStartedAt) {
		t.Fatalf("wall clock includes time before Codex process start: measured=%s elapsed=%s", result.WallClock, time.Since(result.processStartedAt))
	}
}

func TestExecuteAgentRecordsNoWallTimeBeforeCodexStarts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", root)
	data, err := os.ReadFile("../../testdata/bench/task_00_hi_overhead.txtar")
	if err != nil {
		t.Fatal(err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatal(err)
	}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmBaseline, Variant: "small", Prompt: task.Metadata.Instruction, Policy: SemeditArmRestrictWrite}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if !result.processStartedAt.IsZero() {
		t.Fatalf("missing Codex executable started a process at %s", result.processStartedAt)
	}
	if result.WallClock != 0 || len(result.InteractionSteps) != 1 || result.InteractionSteps[0].WallClock != 0 {
		t.Fatalf("failed Codex launch recorded prelaunch wall time: %+v", result)
	}
	if result.Error == "" {
		t.Fatal("missing Codex executable produced no error")
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

func TestDiagnosticToolCoverageUsesAllTaskToolSelectionsAndKeepsBatchExplicit(t *testing.T) {
	coverage := diagnosticToolCoverage([]string{
		"mcp__semedit__semantic_lookup",
		"semantic_replace_body",
		"semantic_batch",
	}, []ToolCall{
		{Name: "semedit/semantic_lookup", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed},
		{Name: "semantic_insert_construct", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusSucceeded},
	}, ToolObservationComplete, "")
	if coverage == nil {
		t.Fatal("configured diagnostic coverage is nil")
	}
	if !slices.Equal(coverage.Expected, []string{"semantic_batch", "semantic_lookup", "semantic_replace_body"}) {
		t.Fatalf("expected tools = %v", coverage.Expected)
	}
	if !slices.Equal(coverage.Observed, []string{"semantic_insert_construct", "semantic_lookup"}) {
		t.Fatalf("observed tool selections = %v", coverage.Observed)
	}
	if !slices.Equal(coverage.Missing, []string{"semantic_batch", "semantic_replace_body"}) || !coverage.Complete || coverage.ExpectedSatisfied {
		t.Fatalf("missing/capture-complete/expected-satisfied = %v/%t/%t", coverage.Missing, coverage.Complete, coverage.ExpectedSatisfied)
	}
	incomplete := diagnosticToolCoverage([]string{"semantic_lookup", "semantic_replace_body"}, []ToolCall{{Name: "semantic_lookup"}}, ToolObservationUnknown, "terminal observation unavailable")
	if incomplete.Complete || incomplete.ExpectedSatisfied || incomplete.Missing != nil || incomplete.CompletenessReason == "" {
		t.Fatalf("incomplete capture implied missing tools: %#v", incomplete)
	}
	incompleteSatisfied := diagnosticToolCoverage([]string{"semantic_lookup", "semantic_replace_body"}, []ToolCall{{Name: "semantic_lookup"}, {Name: "semantic_replace_body"}}, ToolObservationUnknown, "terminal observation unavailable")
	if incompleteSatisfied.Complete || !incompleteSatisfied.ExpectedSatisfied || incompleteSatisfied.Missing != nil {
		t.Fatalf("positive expected observations were lost for incomplete capture: %#v", incompleteSatisfied)
	}
	encoded, err := json.Marshal(coverage)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"complete":true`) || !strings.Contains(string(encoded), `"semantic_lookup"`) {
		t.Fatalf("diagnostic coverage not serialized: %s", encoded)
	}
}

func TestExecuteAgentSkipsDiagnosticReflectionAfterFailedFinalOracle(t *testing.T) {
	for _, followups := range [][]string{nil, {"retry once"}} {
		name := "zero-followups"
		if len(followups) > 0 {
			name = "exhausted-followups"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			if err := os.Mkdir(bin, 0o750); err != nil {
				t.Fatal(err)
			}
			countPath := filepath.Join(root, "count")
			script := filepath.Join(bin, "codex")
			program := `#!/bin/sh
count=0
if [ -f "$SEMEDIT_REFLECTION_COUNT" ]; then count=$(cat "$SEMEDIT_REFLECTION_COUNT"); fi
count=$((count + 1))
printf '%s' "$count" > "$SEMEDIT_REFLECTION_COUNT"
if [ "$count" -gt "$SEMEDIT_EXPECTED_TASK_ATTEMPTS" ]; then while :; do :; done; fi
printf '{"type":"thread.started","thread_id":"failed-session"}\n'
printf '{"type":"turn.started"}\n'
printf '{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"DONE"}}\n'
printf '{"type":"turn.completed","usage":{"input_tokens":2,"output_tokens":1}}\n'
`
			if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("SEMEDIT_REFLECTION_COUNT", countPath)
			t.Setenv("SEMEDIT_EXPECTED_TASK_ATTEMPTS", strconv.Itoa(1+len(followups)))
			task := &Task{
				Metadata: TaskMetadata{
					TaskID:      "failed-final-oracle",
					Instruction: "Add Missing.",
					Oracle:      OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"Missing"}}},
				},
				Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Existing() {}\n")),
			}
			execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmSemedit, Variant: "small", Prompt: "original", Followups: followups, Policy: SemeditArmRestrictWrite}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(ctx, execution)
			if err != nil {
				t.Fatal(err)
			}
			if ctx.Err() != nil {
				t.Fatalf("task returned only after its context ended: %v", ctx.Err())
			}
			if result.Oracle == nil || result.Oracle.Passed || result.Oracle.FailureStage != "level_2_ast" {
				t.Fatalf("final failed oracle was not preserved: %#v", result.Oracle)
			}
			if result.DiagnosticReflectionSkipped != "final task oracle did not pass" {
				t.Fatalf("diagnostic skip reason = %q", result.DiagnosticReflectionSkipped)
			}
			if result.SemanticToolReflection != nil || result.SemanticBatchReflection != nil {
				t.Fatalf("failed final oracle triggered diagnostic reflection: %#v", result)
			}
			count, err := os.ReadFile(countPath)
			if err != nil || string(count) != strconv.Itoa(1+len(followups)) {
				t.Fatalf("provider attempts = %q, err=%v; want %d task attempts and no reflection", count, err, 1+len(followups))
			}
			if len(result.InteractionSteps) != 1+len(followups) {
				t.Fatalf("interaction steps = %d, want %d task attempts", len(result.InteractionSteps), 1+len(followups))
			}
		})
	}
}

func TestExecuteAgentStagedFollowupsKeepSessionAndStageCosts(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	countPath := filepath.Join(root, "count")
	workLog := filepath.Join(root, "workdirs")
	argsLog := filepath.Join(root, "args")
	script := filepath.Join(bin, "codex")
	program := `#!/bin/sh
count=0
if [ -f "$SEMEDIT_SESSION_COUNT" ]; then count=$(cat "$SEMEDIT_SESSION_COUNT"); fi
count=$((count + 1))
printf '%s' "$count" > "$SEMEDIT_SESSION_COUNT"
pwd >> "$SEMEDIT_SESSION_WORKDIRS"
printf '%s\n' "$*" >> "$SEMEDIT_SESSION_ARGS"
printf '{"type":"thread.started","thread_id":"fake-stage"}\n'
printf '{"type":"turn.started"}\n'
if [ "$count" -eq 1 ]; then
  printf 'package main\nfunc B() {}\n' > main.go
  printf '{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":2}}\n'
else
  printf 'package main\nfunc A() {}\nfunc B() {}\n' > main.go
  printf '{"type":"turn.completed","usage":{"input_tokens":7,"output_tokens":3}}\n'
fi
printf '{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"DONE"}}\n'
`
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_SESSION_COUNT", countPath)
	t.Setenv("SEMEDIT_SESSION_WORKDIRS", workLog)
	t.Setenv("SEMEDIT_SESSION_ARGS", argsLog)
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:                  "staged-session",
			Instruction:             "Stage one",
			InteractiveMode:         "staged",
			StagedInitialTotalEdits: 1,
			Oracle:                  OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"A"}, MustNotContainSymbols: []string{"B"}}},
			StagedFollowups:         []StagedFollowup{{TotalEdits: 2, Instruction: "Stage two", Oracle: OracleConfig{AST: ASTConfig{File: "main.go", MustContainSymbols: []string{"A", "B"}}}}},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Old() {}\n")),
	}
	execution := AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmBaseline, Variant: "small", Prompt: "Stage one", StagedFollowups: task.Metadata.StagedFollowups, Policy: SemeditArmRestrictWrite}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Oracle == nil || !result.Oracle.Passed || len(result.InteractionSteps) != 2 {
		t.Fatalf("staged result lost first-stage failure: %+v", result)
	}
	first, second := result.InteractionSteps[0], result.InteractionSteps[1]
	if first.TotalEdits != 1 || second.TotalEdits != 2 || first.Oracle == nil || first.Oracle.Passed || second.Oracle == nil || !second.Oracle.Passed {
		t.Fatalf("stage checkpoints = %+v", result.InteractionSteps)
	}
	if first.PromptTokens != 5 || second.PromptTokens != 7 || result.PromptTokens != 12 || first.OutputTokens != 2 || second.OutputTokens != 3 || result.OutputTokens != 5 {
		t.Fatalf("per-stage token accounting = %+v, aggregate = %+v", result.InteractionSteps, result)
	}
	count, err := os.ReadFile(countPath)
	if err != nil || string(count) != "2" {
		t.Fatalf("provider calls = %q, err=%v", count, err)
	}
	workdirs, err := os.ReadFile(workLog)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(workdirs))
	if len(lines) != 2 || lines[0] != lines[1] {
		t.Fatalf("staged turns changed workspace: %q", workdirs)
	}
	args, err := os.ReadFile(argsLog)
	if err != nil || !strings.Contains(string(args), "resume --json fake-stage") {
		t.Fatalf("staged follow-up did not resume provider session: %q, err=%v", args, err)
	}
}
