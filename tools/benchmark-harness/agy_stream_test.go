// agy_stream_test.go proves Agy continuations reuse one live stream process and do not count cumulative usage twice.

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type agyMultiTurnFixture struct {
	workspace string
	pidLog    string
	promptLog string
	argsLog   string
	session   *agySession
}

func newAgyMultiTurnFixture(t *testing.T) *agyMultiTurnFixture {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	pidLog := filepath.Join(root, "pids")
	promptLog := filepath.Join(root, "prompts")
	argsLog := filepath.Join(root, "args")
	transcript := filepath.Join(os.Getenv("HOME"), ".gemini", "antigravity-cli", "brain", "agy-stream-test", ".system_generated", "logs", "transcript.jsonl")
	script := `#!/bin/sh
printf '%s\n' "$$" >> "$SEMEDIT_AGY_PID_LOG"
printf '%s\n' "$*" > "$SEMEDIT_AGY_ARGS_LOG"
count=0
while IFS= read -r prompt; do
  count=$((count + 1))
  printf '%s\n' "$prompt" >> "$SEMEDIT_AGY_PROMPT_LOG"
  if [ "$count" -eq 1 ]; then
    printf '{"event":"init","conversation_id":"agy-stream-test"}\n'
    input=10; output=3; thinking=1; cache=4; total=13
    tool=semantic_lookup
  else
    if [ "$count" -eq 2 ]; then
      input=18; output=8; thinking=2; cache=8; total=26
      tool=semantic_replace_body
    else
      input=24; output=12; thinking=3; cache=12; total=39
      tool=
    fi
  fi
  mkdir -p "$(dirname "$SEMEDIT_AGY_TRANSCRIPT")"
  printf '{"type":"PLANNER_RESPONSE"}\n' >> "$SEMEDIT_AGY_TRANSCRIPT"
  if [ -n "$tool" ]; then
    printf '{"type":"GENERIC","status":"done","tool_calls":[{"name":"%s","args":{"symbol":"x"}}]}\n' "$tool" >> "$SEMEDIT_AGY_TRANSCRIPT"
  fi
  printf '{"event":"step_update","step_update":{"conversation_id":"agy-stream-test","step_index":%s,"state":"DONE","step_type":"agent_response"}}\n' "$count"
  printf '{"event":"result","result":{"conversation_id":"agy-stream-test","status":"SUCCESS","response":"turn-%s","duration_seconds":%s,"num_turns":%s,"usage":{"input_tokens":%s,"output_tokens":%s,"thinking_tokens":%s,"cache_read_tokens":%s,"total_tokens":%s}}}\n' "$count" "$count" "$count" "$input" "$output" "$thinking" "$cache" "$total"
done
`
	agy := filepath.Join(root, "agy")
	if err := os.WriteFile(agy, []byte(script), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEMEDIT_AGY_PID_LOG", pidLog)
	t.Setenv("SEMEDIT_AGY_PROMPT_LOG", promptLog)
	t.Setenv("SEMEDIT_AGY_ARGS_LOG", argsLog)
	t.Setenv("SEMEDIT_AGY_TRANSCRIPT", transcript)
	runner := NewRunner(filepath.Join(root, "scratch"))
	runner.agyExecutable = agy
	session := newAgySession(runner, workspace, Target{Harness: string(HarnessAgy), Model: "model-a", Effort: "high"}, ArmSemedit)
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Errorf("close Agy session: %v", err)
		}
	})
	return &agyMultiTurnFixture{workspace: workspace, pidLog: pidLog, promptLog: promptLog, argsLog: argsLog, session: session}
}

func TestAgyStreamSessionReusesProcessAndDeltasUsage(t *testing.T) {
	fixture := newAgyMultiTurnFixture(t)
	session := fixture.session
	first := &RunResult{}
	conversationID, err := session.run(context.Background(), "first prompt", first, "")
	if err != nil {
		t.Fatal(err)
	}
	if conversationID != "agy-stream-test" || first.PromptTokens != 10 || first.OutputTokens != 3 || first.Turns != 1 || len(first.ToolCalls) != 1 || first.ToolCalls[0].Name != "semantic_lookup" {
		t.Fatalf("first turn mismatch: conversation=%q result=%+v", conversationID, first)
	}
	second := &RunResult{}
	continuedID, err := session.run(context.Background(), "second prompt", second, conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if continuedID != conversationID || second.PromptTokens != 8 || second.OutputTokens != 5 || second.ReasoningTokens != 1 || second.CachedPromptTokens != 4 || second.Turns != 1 || len(second.ToolCalls) != 1 || second.ToolCalls[0].Name != "semantic_replace_body" {
		t.Fatalf("second turn must contain usage deltas: conversation=%q result=%+v", continuedID, second)
	}
	reflection := &RunResult{}
	reflectionID, err := session.run(context.Background(), "diagnostic reflection", reflection, conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if reflectionID != conversationID || reflection.PromptTokens != 6 || reflection.OutputTokens != 4 || reflection.ReasoningTokens != 1 || reflection.CachedPromptTokens != 4 || reflection.Turns != 1 || reflection.InternalTurns != 1 || len(reflection.ToolCalls) != 0 {
		t.Fatalf("reflection turn left the persistent session or duplicated usage: conversation=%q result=%+v", reflectionID, reflection)
	}
	if closeErr := session.close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	pids, err := os.ReadFile(fixture.pidLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(string(pids))) != 1 {
		t.Fatalf("Agy process count = %q, want one process", pids)
	}
	args, err := os.ReadFile(fixture.argsLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--input-format stream-json") || !strings.Contains(string(args), "--output-format stream-json") || !strings.Contains(string(args), "--model model-a") || !strings.Contains(string(args), "--effort high") {
		t.Fatalf("Agy stream arguments lost session settings: %s", args)
	}
	prompts, err := os.ReadFile(fixture.promptLog)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(prompts)), "\n"); len(lines) != 3 || !strings.Contains(lines[0], "first prompt") || !strings.Contains(lines[1], "second prompt") || !strings.Contains(lines[2], "diagnostic reflection") {
		t.Fatalf("Agy prompt stream order = %q", prompts)
	}
	if _, err := os.Stat(filepath.Join(fixture.workspace, ".agents", "mcp_config.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Agy MCP config should be removed after process close, stat err=%v", err)
	}
}

func TestAgyStreamCancellationTerminatesProcessAndSession(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
while IFS= read -r prompt; do
  printf '{"event":"init","conversation_id":"agy-cancel-test"}\n'
done
`
	agy := filepath.Join(root, "agy")
	if err := os.WriteFile(agy, []byte(script), 0o750); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(filepath.Join(root, "scratch"))
	runner.agyExecutable = agy
	session := newAgySession(runner, workspace, Target{Harness: string(HarnessAgy)}, ArmBaseline)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := session.run(ctx, "wait forever", &RunResult{}, "")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled turn error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("canceled Agy process shutdown took %s", elapsed)
	}
	if session.process == nil || session.process.cmd.ProcessState == nil {
		t.Fatal("canceled Agy process was not reaped")
	}
	if _, err := session.run(context.Background(), "must not restart", &RunResult{}, session.process.conversationID); err == nil {
		t.Fatal("failed Agy session accepted another turn")
	}
}

func TestAgyStreamRejectsMCPConfigMutationWithoutAnotherPrompt(t *testing.T) {
	fixture := newAgyMultiTurnFixture(t)
	session := fixture.session
	conversationID, err := session.run(context.Background(), "first prompt", &RunResult{}, "")
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(fixture.workspace, ".agents", "mcp_config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if configWriteErr := os.WriteFile(configPath, append(configData, '\n'), 0o600); configWriteErr != nil {
		t.Fatal(configWriteErr)
	}
	continuedID, err := session.run(context.Background(), "must not reach Agy", &RunResult{}, conversationID)
	if err == nil || !strings.Contains(err.Error(), "agy MCP configuration changed") {
		t.Fatalf("mutated Agy config error = %v", err)
	}
	if continuedID != conversationID {
		t.Fatalf("Agy conversation identity changed after config mutation: got %q want %q", continuedID, conversationID)
	}
	if _, runErr := session.run(context.Background(), "must not restart", &RunResult{}, conversationID); runErr == nil {
		t.Fatal("failed Agy session accepted another turn")
	}
	pids, err := os.ReadFile(fixture.pidLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(string(pids))) != 1 {
		t.Fatalf("Agy config mutation created %d processes, want one", len(strings.Fields(string(pids))))
	}
	prompts, err := os.ReadFile(fixture.promptLog)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(prompts)), "\n"); len(lines) != 1 || !strings.Contains(lines[0], "first prompt") {
		t.Fatalf("prompt reached Agy after frozen-config mutation: %q", prompts)
	}
}

func TestAgyStreamEOFBeforeResultFailsAndReaps(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
printf '{"event":"init","conversation_id":"agy-eof-test"}\n'
exit 17
`
	agy := filepath.Join(root, "agy")
	if err := os.WriteFile(agy, []byte(script), 0o750); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(filepath.Join(root, "scratch"))
	runner.agyExecutable = agy
	session := newAgySession(runner, workspace, Target{Harness: string(HarnessAgy)}, ArmBaseline)
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Errorf("close Agy session: %v", err)
		}
	})
	started := time.Now()
	_, err := session.run(context.Background(), "exit before result", &RunResult{}, "")
	if err == nil || !strings.Contains(err.Error(), "agy stream ended before a turn result") {
		t.Fatalf("Agy EOF error = %v, want missing result", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("Agy EOF handling took %s, want bounded completion", elapsed)
	}
	if session.process == nil || session.process.cmd.ProcessState == nil {
		t.Fatal("Agy EOF failure did not reap its process")
	}
}

func TestAgyStreamRejectsQueuedDuplicateResultWithoutRestart(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(home, ".gemini", "antigravity-cli", "brain", "agy-duplicate-test", ".system_generated", "logs", "transcript.jsonl")
	pidLog := filepath.Join(root, "pids")
	script := `#!/bin/sh
printf '%s\n' "$$" >> "$SEMEDIT_AGY_PID_LOG"
while IFS= read -r prompt; do
  printf '{"event":"init","conversation_id":"agy-duplicate-test"}\n'
  mkdir -p "$(dirname "$SEMEDIT_AGY_TRANSCRIPT")"
  printf '{"type":"PLANNER_RESPONSE"}\n' >> "$SEMEDIT_AGY_TRANSCRIPT"
  result='{"event":"result","result":{"conversation_id":"agy-duplicate-test","status":"SUCCESS","response":"first","num_turns":1,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}'
  printf '%s\n' "$result"
  printf '%s\n' "$result"
done
`
	agy := filepath.Join(root, "agy")
	if err := os.WriteFile(agy, []byte(script), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEMEDIT_AGY_PID_LOG", pidLog)
	t.Setenv("SEMEDIT_AGY_TRANSCRIPT", transcript)
	runner := NewRunner(filepath.Join(root, "scratch"))
	runner.agyExecutable = agy
	session := newAgySession(runner, workspace, Target{Harness: string(HarnessAgy)}, ArmBaseline)
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Errorf("close Agy session: %v", err)
		}
	})
	conversationID, err := session.run(context.Background(), "first", &RunResult{}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.run(context.Background(), "second", &RunResult{}, conversationID)
	if err == nil || !strings.Contains(err.Error(), "num_turns did not advance from 1 to 1") {
		t.Fatalf("queued duplicate result was not rejected: %v", err)
	}
	if _, runErr := session.run(context.Background(), "must not restart", &RunResult{}, conversationID); runErr == nil {
		t.Fatal("Agy session restarted after a stale result")
	}
	pids, err := os.ReadFile(pidLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(string(pids))) != 1 {
		t.Fatalf("stale result created %d Agy processes, want one", len(strings.Fields(string(pids))))
	}
	if session.process == nil || session.process.cmd.ProcessState == nil {
		t.Fatal("failed Agy session process was not reaped")
	}
}
