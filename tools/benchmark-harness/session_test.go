// This file exercises session boundaries because diagnostic turns must remain auditable without changing benchmark task measurements.
package main

import (
	"context"
	"encoding/json"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"os/exec"
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
	session, sessionErr := newAgentSession(provider, root, "session-test", workDir)
	if sessionErr != nil {
		t.Fatal(sessionErr)
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
	if decodeErr := json.Unmarshal(transcript, &turns); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(turns) != 3 || turns[0].Classification != "task" || turns[1].Classification != "task" || turns[2].Classification != "diagnostic_reflection" {
		t.Fatalf("unexpected explicit transcript turns: %+v", turns)
	}
}

func TestAgentSessionRetainsTaskMCPMetricsAfterCleanup(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.MkdirAll(filepath.Join(workDir, ".scratch"), 0o750); err != nil {
		t.Fatal(err)
	}
	session, sessionErr := newAgentSession(&behavioralSessionProvider{}, root, "metrics-session", workDir)
	if sessionErr != nil {
		t.Fatal(sessionErr)
	}
	session.execution = AgentExecution{Task: &Task{}, Arm: ArmSemedit}
	session.result = &RunResult{}
	taskFinishedAt := time.Now()
	session.turns = []sessionTurn{
		{Classification: "task", FinishedAt: taskFinishedAt},
		{Classification: "diagnostic_reflection", FinishedAt: taskFinishedAt.Add(time.Second)},
	}
	records := []map[string]any{
		{"sequence": 1, "tool": "semantic_rename", "completed_at": taskFinishedAt.Add(-time.Millisecond), "metrics": map[string]any{"total_ms": 1250}},
		{"sequence": 2, "tool": "semantic_lookup", "completed_at": taskFinishedAt.Add(time.Millisecond), "metrics": map[string]any{"total_ms": 10}},
	}
	data, marshalErr := json.Marshal(records)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if err := pipeline.WriteAtomic(benchmarkMCPMetricsPath(workDir), data); err != nil {
		t.Fatal(err)
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("workspace cleanup = %v, want removed workspace", err)
	}
	metricsPath := session.result.Provenance["mcp_metrics_file"]
	if metricsPath == "" {
		t.Fatal("result provenance omitted retained MCP metrics path")
	}
	retainedData, readErr := os.ReadFile(metricsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var retained []struct {
		Tool string `json:"tool"`
	}
	if err := json.Unmarshal(retainedData, &retained); err != nil {
		t.Fatal(err)
	}
	if len(retained) != 1 || retained[0].Tool != "semantic_rename" {
		t.Fatalf("retained task metrics = %+v, want task call only", retained)
	}
}

func TestParseAgyTranscriptRejectsMalformedEvent(t *testing.T) {
	result := &RunResult{}
	if err := parseAgyTranscript([]byte("{not-json}\n"), result); err == nil {
		t.Fatal("malformed transcript unexpectedly succeeded")
	}
	if err := parseAgyTranscript([]byte("\n{not-json}\n"), result); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("malformed event after blank line error = %v, want physical line 2", err)
	}
}

func TestPrepareAgentWorkspaceReportsInitialFileReadErrors(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "workspace")
	task := &Task{
		Metadata: TaskMetadata{TaskID: "initial-file-read", Oracle: OracleConfig{AST: ASTConfig{File: "nested"}}},
		Archive:  ParseArchive([]byte("-- nested/child.txt --\ncontent\n")),
	}
	_, err := prepareAgentWorkspace(context.Background(), &Runner{}, task, Target{Harness: "fake"}, ArmBaseline, "small", workDir, &agentSession{})
	if err == nil || !strings.Contains(err.Error(), "read initial benchmark file") {
		t.Fatalf("prepare workspace error = %v, want initial file read failure", err)
	}
}

func TestExecuteAgentUsesOneProcessSessionAndExcludesReflectionMetrics(t *testing.T) {
	root := t.TempDir()
	requestLog, processLog := installCodexAppServerFake(t, root, "session")
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:               "session-process",
			Instruction:          "Add Done.",
			InteractiveFollowups: []string{"Correct the implementation so Done exists."},
			Oracle: OracleConfig{
				AST:                     ASTConfig{File: "main.go", MustContainSymbols: []string{"Done"}},
				DiagnosticExpectedTools: []string{"semantic_lookup", "semantic_replace_body", "semantic_rename"},
			},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Old() {}\n")),
	}
	execution := AgentExecution{
		Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmSemedit,
		Variant: "small", Prompt: "original", Followups: []string{"Correct the implementation so Done exists."},
		Policy: SemeditArmRestrictWrite,
	}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	assertSessionProcessTaskResult(t, result)
	assertSessionProcessWorkspace(t, processLog, requestLog)
	assertSessionProcessTranscript(t, result)
}

func TestExecuteAgentEvaluatesOracleAfterAgentDeadline(t *testing.T) {
	root := t.TempDir()
	installCodexAppServerFake(t, root, "")
	bin := filepath.Join(root, "bin")
	if err := pipeline.WriteAtomic(filepath.Join(bin, "go"), []byte("#!/bin/sh\nsleep 0.9\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(bin, "go"), 0o700); err != nil {
		t.Fatal(err)
	}
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:      "oracle-after-agent-deadline",
			Instruction: "Leave the existing declaration intact.",
			Oracle: OracleConfig{
				AST:   ASTConfig{File: "main.go", MustContainSymbols: []string{"Existing"}},
				Build: BuildConfig{CleanCompile: true},
			},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Existing() {}\n")),
	}
	ctx := withAgentTaskTimeout(context.Background(), 750*time.Millisecond)
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(ctx, AgentExecution{Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"}, Arm: ArmBaseline, Variant: "small", Prompt: task.Metadata.Instruction})
	if err != nil {
		t.Fatalf("ExecuteAgent returned an oracle deadline error: %v", err)
	}
	if !result.Success || result.Oracle == nil || !result.Oracle.Passed {
		t.Fatalf("oracle did not run after the agent deadline: %+v", result)
	}
}

func assertSessionProcessTaskResult(t *testing.T, result *RunResult) {
	t.Helper()
	if !result.Success || result.Turns != 2 || result.PromptTokens != 12 || result.OutputTokens != 4 || result.ToolCount != 2 || result.DiagnosticToolCoverage == nil || !result.DiagnosticToolCoverage.Complete || result.DiagnosticToolCoverage.ObservationState != ToolObservationComplete || result.DiagnosticToolCoverage.ExpectedSatisfied || !slices.Equal(result.DiagnosticToolCoverage.Missing, []string{"semantic_rename"}) || !slices.Equal(result.DiagnosticToolCoverage.Observed, []string{"semantic_lookup", "semantic_replace_body"}) {
		t.Fatalf("task aggregates include non-task telemetry or missed correction: %+v", result)
	}
	if slices.Contains(result.DiagnosticToolCoverage.Observed, "semantic_insert_construct") {
		t.Fatal("diagnostic reflection tool leaked into measured tool coverage")
	}
	if result.ToolCalls[1].FunctionalStatus != ToolCallStatusFailed {
		t.Fatalf("failed-call outcome was lost: coverage=%#v call=%#v", result.DiagnosticToolCoverage, result.ToolCalls[1])
	}
}

func assertSessionProcessWorkspace(t *testing.T, countPath, workLog string) {
	t.Helper()
	processes, err := os.ReadFile(countPath)
	if err != nil || strings.Count(string(processes), "process") != 1 {
		t.Fatalf("app-server process count = %q, err=%v; want one process", processes, err)
	}
	data, err := os.ReadFile(workLog)
	if err != nil {
		t.Fatal(err)
	}
	type requestRecord struct {
		Method string `json:"method"`
		Params struct {
			ThreadID string `json:"threadId"`
			CWD      string `json:"cwd"`
		} `json:"params"`
	}
	var requests []requestRecord
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var request requestRecord
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			t.Fatalf("decode app-server request %q: %v", line, err)
		}
		requests = append(requests, request)
	}
	count := func(method string) int {
		total := 0
		for _, request := range requests {
			if request.Method == method {
				total++
			}
		}
		return total
	}
	if count("initialize") != 1 || count("thread/start") != 1 || count("turn/start") != 4 {
		t.Fatalf("app-server lifecycle requests: initialize=%d thread/start=%d turn/start=%d", count("initialize"), count("thread/start"), count("turn/start"))
	}
	var cwd string
	for _, request := range requests {
		if request.Method == "thread/start" {
			cwd = request.Params.CWD
		}
		if request.Method == "turn/start" && request.Params.ThreadID != "persistent-thread" {
			t.Fatalf("turn used thread %q, want original thread persistent-thread", request.Params.ThreadID)
		}
	}
	if cwd == "" || !filepath.IsAbs(cwd) {
		t.Fatalf("thread/start cwd = %q, want absolute fixture directory", cwd)
	}
}

func assertSessionProcessTranscript(t *testing.T, result *RunResult) {
	t.Helper()
	transcriptPath := result.Provenance["session_transcript"]
	transcript, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatalf("session evidence missing after cleanup: %v", err)
	}
	if strings.Count(string(transcript), "diagnostic_reflection") != 2 || strings.Count(string(transcript), "persistent-thread") < 4 {
		t.Fatalf("transcript did not retain task and diagnostic events: %s", transcript)
	}
	if result.SemanticIntegrationReflection == nil || result.SemanticIntegrationReflection.Prompt != semanticIntegrationReflectionPrompt || !strings.Contains(result.SemanticIntegrationReflection.Response, "ask my owner") {
		t.Fatalf("integration feedback missing or incorrectly captured: %#v", result.SemanticIntegrationReflection)
	}
	var turns []sessionTurn
	if decodeErr := json.Unmarshal(transcript, &turns); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(turns) != 4 || turns[3].Classification != "diagnostic_reflection" || turns[3].Prompt != semanticIntegrationReflectionPrompt {
		t.Fatalf("integration question did not resume the measured session: %#v", turns)
	}
	if result.WallClock > turns[2].StartedAt.Sub(result.processStartedAt) {
		t.Fatal("integration reflection time leaked into measured wall clock")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"diagnostic_tool_coverage"`) || !strings.Contains(string(encoded), `"observation_state":"complete"`) || !strings.Contains(string(encoded), `"missing":["semantic_rename"]`) {
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
	installCodexAppServerFake(t, root, "")
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
	if result.processStartedAt.IsZero() || result.WallClock <= 0 {
		t.Fatalf("wall clock must include Codex execution from process start: start=%s duration=%s", result.processStartedAt, result.WallClock)
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
	if err == nil {
		t.Fatal("missing Codex executable returned no execution error")
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
	assertLaunchFailureTranscript(t, result)
}

func TestAgentLaunchFailureHasZeroTaskTimeForBothArms(t *testing.T) {
	root := t.TempDir()
	goPath, lookupErr := exec.LookPath("go")
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	if symlinkErr := os.Symlink(goPath, filepath.Join(root, "go")); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	t.Setenv("PATH", root)
	data, err := os.ReadFile("../../testdata/bench/task_00_hi_overhead.txtar")
	if err != nil {
		t.Fatal(err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, harness := range []HarnessType{HarnessOpenCode, HarnessAgy} {
		for _, arm := range []ArmType{ArmBaseline, ArmSemedit} {
			t.Run(string(harness)+"/"+string(arm), func(t *testing.T) {
				runner := NewRunner(filepath.Join(root, string(harness), string(arm), "scratch"), WithOpenCodeModelLimits(openCodeModelLimits{Context: 65536, Output: 8192}), WithOpenCodeRelayURL("http://127.0.0.1:12345/v1"))
				runner.agyExecutable = filepath.Join(root, "missing-agy")
				target := Target{Harness: string(harness), Model: "amdbeast/fake"}
				execution := AgentExecution{Task: task, Target: target, Arm: arm, Variant: "small", Prompt: task.Metadata.Instruction, Policy: SemeditArmRestrictWrite}
				result, err := runner.ExecuteAgent(t.Context(), execution)
				if err == nil {
					t.Fatal("missing provider executable returned no execution error")
				}
				if result.Error == "" || !result.processStartedAt.IsZero() || result.WallClock != 0 {
					t.Fatalf("failed launch recorded task time: %+v", result)
				}
				if len(result.InteractionSteps) != 1 || result.InteractionSteps[0].WallClock != 0 {
					t.Fatalf("failed launch recorded step time: %+v", result.InteractionSteps)
				}
				assertLaunchFailureTranscript(t, result)
			})
		}
	}
}

func assertLaunchFailureTranscript(t *testing.T, result *RunResult) {
	t.Helper()
	transcriptPath := result.Provenance["session_transcript"]
	if transcriptPath == "" {
		t.Fatal("failed launch omitted its session transcript")
	}
	if _, err := os.Stat(transcriptPath); err != nil {
		t.Fatalf("failed launch transcript is not retained: %v", err)
	}
}

func TestAgyResumeFailureKeepsOnlyNewTranscriptMetrics(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	logs := filepath.Join(home, ".gemini", "antigravity-cli", "brain", "conversation", ".system_generated", "logs")
	if err := os.MkdirAll(logs, 0o750); err != nil {
		t.Fatal(err)
	}
	transcriptPath := filepath.Join(logs, "transcript.jsonl")
	pidPath := filepath.Join(root, "pid")
	program := `#!/bin/sh
printf '%s\n' "$$" > "$SEMEDIT_AGY_PID"
count=0
while IFS= read -r prompt; do
  count=$((count + 1))
  if [ "$count" -eq 1 ]; then
    printf '{"event":"init","conversation_id":"conversation"}\n'
    printf '{"type":"PLANNER_RESPONSE"}\n{"type":"GENERIC","status":"done","tool_calls":[{"name":"semantic_lookup","args":{}},{"name":"semantic_replace_body","args":{}}]}\n' >> "$SEMEDIT_AGY_TRANSCRIPT"
    input=5; output=2; turns=1; status=SUCCESS
  else
    printf '{bad' > "$SEMEDIT_AGY_TRANSCRIPT"
    input=7; output=3; turns=2; status=SUCCESS
  fi
  printf '{"event":"result","result":{"conversation_id":"conversation","status":"%s","num_turns":%s,"usage":{"input_tokens":%s,"output_tokens":%s}}}\n' "$status" "$turns" "$input" "$output"
done
`
	script := filepath.Join(root, "agy")
	if err := os.WriteFile(script, []byte(program), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEMEDIT_AGY_PID", pidPath)
	t.Setenv("SEMEDIT_AGY_TRANSCRIPT", transcriptPath)
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(filepath.Join(root, "scratch"), WithOpenCodeRelayURL("http://127.0.0.1:43123/v1"), WithOpenCodeModelLimits(openCodeModelLimits{Context: 65536, Output: 8192}))
	runner.agyExecutable = script
	provider := &agyProvider{runner: runner}
	t.Cleanup(func() {
		if provider.session != nil {
			if err := provider.session.close(); err != nil {
				t.Errorf("close Agy provider session: %v", err)
			}
		}
	})
	first := &RunResult{Arm: ArmBaseline}
	conversationID, err := provider.run(context.Background(), workDir, Target{Harness: string(HarnessAgy)}, "first", first, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.processStartedAt.IsZero() || first.measuredWallClock(time.Now()) <= 0 {
		t.Fatalf("Agy task was not timed from process start: %s", first.processStartedAt)
	}
	second := &RunResult{Arm: ArmBaseline}
	continuedID, err := provider.run(context.Background(), workDir, Target{Harness: string(HarnessAgy)}, "followup", second, conversationID)
	if err == nil || !strings.Contains(err.Error(), "agy cumulative transcript regressed") {
		t.Fatalf("expected cumulative transcript regression, got %v", err)
	}
	if continuedID != conversationID {
		t.Fatalf("Agy continuation identity changed: got %q want %q", continuedID, conversationID)
	}
	if first.InternalTurns != 1 || len(first.ToolCalls) != 2 {
		t.Fatalf("first transcript totals mismatch: %+v", first)
	}
	if second.InternalTurns != 0 || len(second.ToolCalls) != 0 || second.ToolCount != 0 || second.MCPVerified {
		t.Fatalf("unreliable resumed transcript metrics were not withheld: %+v", second)
	}
	if second.PromptTokens != 2 || second.OutputTokens != 1 || second.Turns != 1 {
		t.Fatalf("valid per-turn usage delta should survive transcript failure: %+v", second)
	}
	pids, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(string(pids))) != 1 {
		t.Fatalf("Agy continuation used %d processes, want one", len(strings.Fields(string(pids))))
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
	sandboxOption := installPassthroughOpenCodeSandbox(t, bin)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := NewRunner(filepath.Join(root, "scratch"), sandboxOption, WithOpenCodeRelayURL("http://127.0.0.1:43123/v1"), WithOpenCodeModelLimits(openCodeModelLimits{Context: 65536, Output: 8192}))
	for _, mode := range []string{"empty", "malformed", "null", "object"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("SEMEDIT_OPENCODE_STREAM", mode)
			result := &RunResult{Arm: ArmBaseline, Turns: 17}
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
			if mode == "empty" && result.Turns != 0 {
				t.Fatalf("empty stream retained prior turn count: %d", result.Turns)
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
	cases := []struct {
		name      string
		followups []string
	}{
		{name: "zero-followups"},
		{name: "exhausted-followups", followups: []string{"retry once"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			runFailedFinalOracleScenario(t, testCase.followups)
		})
	}
}

func runFailedFinalOracleScenario(t *testing.T, followups []string) {
	t.Helper()
	root := t.TempDir()
	requestLog, processLog := installCodexAppServerFake(t, root, "")
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
	if result.SemanticToolReflection != nil || result.SemanticBatchReflection != nil || result.SemanticIntegrationReflection != nil {
		t.Fatalf("failed final oracle triggered diagnostic reflection: %#v", result)
	}
	processes, err := os.ReadFile(processLog)
	if err != nil || strings.Count(string(processes), "process") != 1 {
		t.Fatalf("app-server processes = %q, err=%v; want one task session", processes, err)
	}
	requests, err := os.ReadFile(requestLog)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(requests), "\"method\":\"turn/start\""); got != 1+len(followups) {
		t.Fatalf("task turns = %d; want %d", got, 1+len(followups))
	}
	if len(result.InteractionSteps) != 1+len(followups) {
		t.Fatalf("interaction steps = %d, want %d task attempts", len(result.InteractionSteps), 1+len(followups))
	}
}

func TestExecuteAgentStagedFollowupsKeepSessionAndStageCosts(t *testing.T) {
	root := t.TempDir()
	requestLog, processLog := installCodexAppServerFake(t, root, "staged")
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
	processes, err := os.ReadFile(processLog)
	if err != nil || strings.Count(string(processes), "process") != 1 {
		t.Fatalf("app-server processes = %q, err=%v", processes, err)
	}
	requests, err := os.ReadFile(requestLog)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(requests), "\"method\":\"turn/start\""); got != 2 {
		t.Fatalf("staged app-server turns = %d, want two", got)
	}
	if !strings.Contains(string(requests), "\"threadId\":\"persistent-thread\"") {
		t.Fatalf("staged follow-up did not retain its original thread: %s", requests)
	}
}

func TestExecuteAgentIntegrationFeedbackFollowsOnlySemeditTasks(t *testing.T) {
	for _, testCase := range integrationFeedbackCases() {
		t.Run(testCase.name, func(t *testing.T) {
			runIntegrationFeedbackCase(t, testCase)
		})
	}
}

func TestWorkspacePrewarmIsIndependentOfHarness(t *testing.T) {
	for _, harness := range []HarnessType{HarnessCodex, HarnessAgy, HarnessOpenCode} {
		t.Run(string(harness), func(t *testing.T) {
			root := t.TempDir()
			scratch := filepath.Join(root, ".scratch", "benchmarks")
			workDir := filepath.Join(scratch, "fixture")
			if err := os.MkdirAll(workDir, 0o750); err != nil {
				t.Fatal(err)
			}
			binary := filepath.Join(root, "bin", "semedit-next")
			if err := os.MkdirAll(filepath.Dir(binary), 0o750); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\ntest \"$1\" = verify && test \"$3\" = main.go && test \"$6\" = --check-only || exit 9\nprintf warmed > warmup-marker\n"
			if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			task, err := ParseTask([]byte("task_id: warmup\ncategory: test\ninstruction: warmup\ncontexts: [small]\noracle:\n  level_2_ast:\n    file: main.go\n  level_3_build:\n    clean_compile: false\n  level_4_test:\n    pass_tests: false\n-- main.go --\npackage fixture\n"))
			if err != nil {
				t.Fatal(err)
			}
			task.Metadata.Oracle.AST.File = "main.go"
			session, err := newAgentSession(&behavioralSessionProvider{}, scratch, "warmup", workDir)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if closeErr := session.close(); closeErr != nil {
					t.Error(closeErr)
				}
			}()
			setup, err := prepareAgentWorkspace(t.Context(), NewRunner(scratch, WithSemeditPrewarmVerify(true)), task, Target{Harness: string(harness)}, ArmSemedit, "small", workDir, session)
			if err != nil {
				t.Fatal(err)
			}
			if !setup.prewarmApplied {
				t.Fatal("prewarm skipped for selected harness")
			}
			if _, err := os.Stat(filepath.Join(workDir, "warmup-marker")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type integrationFeedbackCase struct {
	name          string
	arm           ArmType
	useTools      bool
	feedbackError bool
	calls         int
}

func integrationFeedbackCases() []integrationFeedbackCase {
	return []integrationFeedbackCase{
		{name: "baseline", arm: ArmBaseline, calls: 1},
		{name: "semantic-use", arm: ArmSemedit, useTools: true, calls: 2},
		{name: "semantic-non-use", arm: ArmSemedit, calls: 3},
		{name: "feedback-error", arm: ArmSemedit, useTools: true, feedbackError: true, calls: 2},
	}
}

func runIntegrationFeedbackCase(t *testing.T, testCase integrationFeedbackCase) {
	t.Helper()
	root := t.TempDir()
	requestLog, processLog := installCodexAppServerFake(t, root, "feedback")
	t.Setenv("SEMEDIT_FEEDBACK_USE_TOOLS", strconv.FormatBool(testCase.useTools))
	t.Setenv("SEMEDIT_FEEDBACK_ERROR", strconv.FormatBool(testCase.feedbackError))
	task := integrationFeedbackTask()
	execution := AgentExecution{
		Task: task, Target: Target{Harness: string(HarnessCodex), Model: "fake"},
		Arm: testCase.arm, Variant: "small", Prompt: "Inspect Existing.", Policy: SemeditArmRestrictWrite,
	}
	result, err := NewRunner(filepath.Join(root, "scratch")).ExecuteAgent(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	assertIntegrationFeedbackResult(t, testCase, result, requestLog)
	processes, err := os.ReadFile(processLog)
	if err != nil || strings.Count(string(processes), "process") != 1 {
		t.Fatalf("app-server process count = %q, err=%v", processes, err)
	}
}

func integrationFeedbackTask() *Task {
	return &Task{
		Metadata: TaskMetadata{
			TaskID:      "integration-feedback",
			Instruction: "Inspect Existing.",
			Oracle: OracleConfig{AST: ASTConfig{
				File: "main.go", MustContainSymbols: []string{"Existing"},
			}},
		},
		Archive: ParseArchive([]byte("-- main.go --\npackage main\nfunc Existing() {}\n")),
	}
}

func assertIntegrationFeedbackResult(t *testing.T, testCase integrationFeedbackCase, result *RunResult, countPath string) {
	t.Helper()
	if !result.Success || result.Error != "" || result.Turns != 1 || result.PromptTokens != 5 || result.OutputTokens != 2 {
		t.Fatalf("feedback changed the task outcome or metrics: %+v", result)
	}
	requests, err := os.ReadFile(countPath)
	if err != nil || strings.Count(string(requests), "\"method\":\"turn/start\"") != testCase.calls {
		t.Fatalf("app-server task/feedback turns = %q, err=%v; want %d", requests, err, testCase.calls)
	}
	if testCase.arm == ArmBaseline {
		if result.SemanticIntegrationReflection != nil {
			t.Fatal("baseline received integration feedback")
		}
		return
	}
	feedback := result.SemanticIntegrationReflection
	if feedback == nil || feedback.Prompt != semanticIntegrationReflectionPrompt {
		t.Fatalf("missing integration question: %#v", feedback)
	}
	if !testCase.useTools && result.SemanticToolReflection == nil {
		t.Fatal("integration question replaced the non-use question")
	}
	assertIntegrationFeedbackCapture(t, testCase, feedback)
}

func assertIntegrationFeedbackCapture(t *testing.T, testCase integrationFeedbackCase, feedback *SemanticToolReflection) {
	t.Helper()
	if testCase.feedbackError {
		if feedback.Error == "" {
			t.Fatal("feedback failure was silently swallowed")
		}
	} else if !strings.Contains(feedback.Response, "would not ask my owner") || feedback.Error != "" {
		t.Fatalf("negative feedback was not preserved: %#v", feedback)
	}
}
