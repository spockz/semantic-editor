// Transport tests keep the persistent Codex session contract auditable without calling the provider.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexAppServerProcessHelper(t *testing.T) {
	if os.Getenv("SEMEDIT_TEST_PROCESS") != "app-server" {
		return
	}
	runFakeCodexAppServer(t)
}

func installCodexAppServerFake(t *testing.T, root, mode string) (string, string) {
	t.Helper()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "requests.jsonl")
	processPath := filepath.Join(root, "processes")
	quotedExecutable := "'" + strings.ReplaceAll(executable, "'", "'\\''") + "'"
	script := "#!/bin/sh\nexec " + quotedExecutable + " -test.run=^TestCodexAppServerProcessHelper$ -- \"$@\"\n"
	provider := filepath.Join(bin, "codex")
	if err := pipeline.WriteAtomic(provider, []byte(script)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(provider, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_TEST_PROCESS", "app-server")
	t.Setenv("SEMEDIT_APP_SERVER_MODE", mode)
	t.Setenv("SEMEDIT_APP_SERVER_LOG", logPath)
	t.Setenv("SEMEDIT_APP_SERVER_PROCESSES", processPath)
	return logPath, processPath
}

func TestCodexAppServerKeepsOneProcessAndThreadAcrossTurns(t *testing.T) {
	root := t.TempDir()
	logPath, processPath := installCodexAppServerFake(t, root, "")
	target := Target{Harness: string(HarnessCodex), Model: "test-model", Effort: "high"}
	server, serverErr := newCodexAppServer(NewRunner(filepath.Join(root, "scratch")), root, target, ArmSemedit)
	if serverErr != nil {
		t.Fatal(serverErr)
	}
	defer func() {
		if closeErr := server.close(); closeErr != nil {
			t.Errorf("close app-server: %v", closeErr)
		}
	}()
	first := &RunResult{}
	threadID, err := server.run(context.Background(), "first prompt", first)
	if err != nil {
		t.Fatal(err)
	}
	if threadID != "persistent-thread" || first.Turns != 1 || first.PromptTokens != 10 || first.CachedPromptTokens != 2 || first.UncachedPromptTokens != 8 || first.OutputTokens != 3 || first.ReasoningTokens != 1 {
		t.Fatalf("first turn result = thread %q, %+v", threadID, first)
	}
	second := &RunResult{}
	secondThreadID, err := server.run(context.Background(), "follow-up prompt", second)
	if err != nil {
		t.Fatal(err)
	}
	if secondThreadID != threadID || second.Turns != 1 || second.PromptTokens != 10 || second.CachedPromptTokens != 2 || second.UncachedPromptTokens != 8 || second.OutputTokens != 3 || second.ReasoningTokens != 1 {
		t.Fatalf("follow-up result = thread %q, %+v", secondThreadID, second)
	}
	if closeErr := server.close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	requests, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(requests), "\"method\":\"initialize\""); got != 1 {
		t.Errorf("initialize requests = %d, want one; log: %s", got, requests)
	}
	if got := strings.Count(string(requests), "\"method\":\"thread/start\""); got != 1 {
		t.Errorf("thread/start requests = %d, want one; log: %s", got, requests)
	}
	assertFakeCodexTurnRequestsUseThread(t, requests, threadID, 2)
	processes, err := os.ReadFile(processPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(processes), "process"); got != 1 {
		t.Errorf("app-server processes = %d, want one", got)
	}
	metrics, err := os.ReadFile(benchmarkMCPMetricsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(metrics), "\"session_metrics\""); got != 1 {
		t.Errorf("session metrics records = %d, want one; metrics: %s", got, metrics)
	}
}

func TestCodexAppServerReturnsOnMalformedEOFApprovalAndCancellation(t *testing.T) {
	for _, mode := range []string{"malformed", "invalid-version", "eof", "approval", "hang"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			installCodexAppServerFake(t, root, mode)
			server, err := newCodexAppServer(NewRunner(filepath.Join(root, "scratch")), root, Target{Harness: string(HarnessCodex)}, ArmBaseline)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = server.close() }()
			timeout := time.Second
			if mode == "hang" {
				timeout = 50 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			started := time.Now()
			_, err = server.run(ctx, "prompt", &RunResult{})
			if err == nil {
				t.Fatal("broken or cancelled app-server turn unexpectedly succeeded")
			}
			if time.Since(started) > 2*time.Second {
				t.Fatalf("app-server run did not return promptly: %s", time.Since(started))
			}
			if mode == "hang" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("cancelled turn error = %v, want deadline exceeded", err)
			}
			if mode == "approval" && !strings.Contains(err.Error(), "unexpected Codex app-server request") {
				t.Fatalf("unexpected approval request error = %v", err)
			}
		})
	}
}

func runFakeCodexAppServer(t *testing.T) {
	t.Helper()
	enabled, serverArgs := fakeCodexAppServerCLIOverrides(os.Args)
	state := &fakeCodexAppServerState{mode: os.Getenv("SEMEDIT_APP_SERVER_MODE"), threadID: "persistent-thread", enabled: enabled, serverArgs: serverArgs}
	encoder := json.NewEncoder(os.Stdout)
	write := func(value any) {
		if message, ok := value.(map[string]any); ok {
			delete(message, "jsonrpc")
		}
		if err := encoder.Encode(value); err != nil {
			os.Exit(3)
		}
	}
	appendFakeCodexAppServerProcess(os.Getenv("SEMEDIT_APP_SERVER_PROCESSES"))
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var request fakeCodexAppServerRequest
		if err := json.Unmarshal(line, &request); err != nil {
			os.Exit(4)
		}
		logFakeCodexAppServerRequest(os.Getenv("SEMEDIT_APP_SERVER_LOG"), line)
		handleFakeCodexAppServerRequest(state, request, write)
	}
	if err := scanner.Err(); err != nil {
		os.Exit(15)
	}
	writeFakeCodexMetrics(state)
	os.Exit(0)
}

type fakeCodexAppServerState struct {
	mode                string
	threadID            string
	turnCount           int
	enabled             bool
	serverArgs          []string
	metricsPath         string
	finalTaskCompletion time.Time
}

type fakeCodexAppServerRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func appendFakeCodexAppServerProcess(path string) {
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(2)
	}
	_, writeErr := file.WriteString("process\\n")
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Exit(2)
	}
}

func logFakeCodexAppServerRequest(path string, line []byte) {
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(5)
	}
	_, writeErr := file.Write(append(line, '\n'))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Exit(6)
	}
}

func handleFakeCodexAppServerRequest(state *fakeCodexAppServerState, request fakeCodexAppServerRequest, write func(any)) {
	switch request.Method {
	case "initialize":
		if state.mode == "malformed" {
			_, _ = os.Stdout.WriteString("not-json\n")
			os.Exit(0)
		}
		if state.mode == "invalid-version" {
			if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"jsonrpc": "1.0", "id": request.ID, "result": map[string]any{}}); err != nil {
				os.Exit(17)
			}
			os.Exit(0)
		}
		write(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{}})
	case "initialized":
	case "thread/start":
		handleFakeCodexThreadStart(request, state, write)
	case "turn/start":
		handleFakeCodexTurnStart(request, state, write)
	case "turn/interrupt":
		if state.mode == "hang" {
			write(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{}})
		}
	case "thread/read":
		write(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"thread": map[string]any{"id": state.threadID}}})
	}
}

func handleFakeCodexThreadStart(request fakeCodexAppServerRequest, state *fakeCodexAppServerState, write func(any)) {
	var params map[string]any
	if err := json.Unmarshal(request.Params, &params); err != nil {
		os.Exit(7)
	}
	config, _ := params["config"].(map[string]any)
	cwd, _ := params["cwd"].(string)
	model, _ := params["model"].(string)
	if model == "" {
		model = "test-model"
	}
	result := map[string]any{
		"thread":         (map[string]any{"id": state.threadID, "ephemeral": params["ephemeral"]}),
		"cwd":            cwd,
		"model":          model,
		"approvalPolicy": "never",
		"sandbox":        map[string]any{"type": "workspaceWrite", "networkAccess": false},
	}
	if state.mode == "persistent" {
		result["thread"] = map[string]any{"id": state.threadID, "ephemeral": false}
	}
	if effort, ok := config["model_reasoning_effort"].(string); ok {
		result["reasoningEffort"] = effort
	}
	if state.enabled {
		state.metricsPath = benchmarkMCPMetricsPath(cwd)
	}
	write(map[string]any{"id": request.ID, "result": result})
}

func handleFakeCodexTurnStart(request fakeCodexAppServerRequest, state *fakeCodexAppServerState, write func(any)) {
	if state.mode == "eof" {
		os.Exit(0)
	}
	state.turnCount++
	turnID := fmt.Sprintf("turn-%d", state.turnCount)
	prompt := fakeCodexPrompt(request.Params)
	if state.mode == "feedback" && state.turnCount > 1 && os.Getenv("SEMEDIT_FEEDBACK_ERROR") == "true" {
		write(map[string]any{"id": request.ID, "error": map[string]any{"code": -32000, "message": "fake reflection failure"}})
		return
	}
	recordFakeCodexEvidence(prompt, state)
	write(map[string]any{"id": request.ID, "result": map[string]any{"turn": map[string]any{"id": turnID, "status": "inProgress"}}})
	write(map[string]any{"method": "turn/started", "params": map[string]any{"threadId": state.threadID, "turn": map[string]any{"id": turnID, "status": "inProgress"}}})
	if state.mode == "approval" {
		write(map[string]any{"id": 999, "method": "item/commandExecution/requestApproval", "params": map[string]any{"threadId": state.threadID, "turnId": turnID, "itemId": "approval-item", "startedAtMs": time.Now().UnixMilli(), "command": "echo fixture"}})
		return
	}
	if state.mode == "hang" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(11)
	}
	emitFakeCodexTurnBody(state, turnID, cwd, write)
	writeFakeCodexTurnResult(state, turnID, write)
	advanceFakeCodexTaskCompletion(state)
}

func fakeCodexPrompt(raw json.RawMessage) string {
	var params struct {
		Input []struct {
			Text string `json:"text"`
		} `json:"input"`
	}
	_ = json.Unmarshal(raw, &params)
	if len(params.Input) == 0 {
		return ""
	}
	return params.Input[0].Text
}

func recordFakeCodexEvidence(prompt string, state *fakeCodexAppServerState) {
	evidence := os.Getenv("SEMEDIT_TEST_EVIDENCE")
	if evidence == "" {
		return
	}
	plan, _ := os.ReadFile(os.Getenv("SEMEDIT_TEST_OUTPUT"))
	cwd, _ := os.Getwd()
	args := make([]string, 0, 1+len(state.serverArgs)+1)
	args = append(args, "mcp_servers.semedit.enabled=false")
	if state.enabled {
		args[0] = "mcp_servers.semedit.enabled=true"
	}
	args = append(args, state.serverArgs...)
	args = append(args, "threadId="+state.threadID)
	invocation := cliProviderInvocation{Args: args, Prompt: prompt, Plan: string(plan), Directory: cwd}
	data, err := json.Marshal(invocation)
	if err != nil {
		os.Exit(8)
	}
	path := filepath.Join(evidence, fmt.Sprintf("%d.json", time.Now().UnixNano()))
	if err := pipeline.WriteAtomic(path, data); err != nil {
		os.Exit(9)
	}
	if prompt != semanticIntegrationReflectionPrompt {
		if err := pipeline.WriteAtomic(filepath.Join(cwd, "main.go"), []byte("package main\nfunc Done() {}\nfunc main() {}\n")); err != nil {
			os.Exit(10)
		}
	}
}

func emitFakeCodexTurnBody(state *fakeCodexAppServerState, turnID, cwd string, write func(any)) {
	switch state.mode {
	case "session":
		emitFakeCodexSessionTurn(state, turnID, cwd, write)
	case "staged":
		source := "package main\nfunc A() {}\nfunc B() {}\n"
		if state.turnCount == 1 {
			source = "package main\nfunc B() {}\n"
		}
		_ = os.WriteFile(filepath.Join(cwd, "main.go"), []byte(source), 0o600)
	case "feedback":
		if state.turnCount == 1 && os.Getenv("SEMEDIT_FEEDBACK_USE_TOOLS") == "true" {
			emitFakeCodexToolCall(state, turnID, write, "semantic_verify")
		}
	case "readonly-cache":
		cache := filepath.Join(cwd, ".scratch", "go", "mod", "pgregory.net", "rapid@v1.0.0")
		if err := os.MkdirAll(cache, 0o700); err != nil {
			os.Exit(18)
		}
		if err := os.WriteFile(filepath.Join(cache, "source.go"), []byte("package rapid\n"), 0o600); err != nil {
			os.Exit(19)
		}
		if err := os.Chmod(cache, 0o500); err != nil {
			os.Exit(20)
		}
	default:
		if state.enabled {
			emitFakeCodexToolCall(state, turnID, write, "semantic_insert_construct")
		}
	}
}

func emitFakeCodexSessionTurn(state *fakeCodexAppServerState, turnID, cwd string, write func(any)) {
	switch state.turnCount {
	case 1:
		_ = os.WriteFile(filepath.Join(cwd, "main.go"), []byte("package main\nfunc Old() {}\n"), 0o600)
		emitFakeCodexToolCall(state, turnID, write, "semantic_lookup")
	case 2:
		_ = os.WriteFile(filepath.Join(cwd, "main.go"), []byte("package main\nfunc Done() {}\n"), 0o600)
		emitFakeCodexToolCall(state, turnID, write, "semantic_replace_body", "failed")
	default:
		emitFakeCodexToolCall(state, turnID, write, "semantic_insert_construct")
	}
}

func emitFakeCodexToolCall(state *fakeCodexAppServerState, turnID string, write func(any), tool string, status ...string) {
	resultStatus := "completed"
	if len(status) > 0 {
		resultStatus = status[0]
	}
	write(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": state.threadID, "turnId": turnID, "item": map[string]any{"id": tool, "type": "mcpToolCall", "server": "semedit", "tool": tool, "status": resultStatus}}})
}

func writeFakeCodexTurnResult(state *fakeCodexAppServerState, turnID string, write func(any)) {
	answer := fakeCodexAnswer(state.mode, state.turnCount)
	write(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": state.threadID, "turnId": turnID, "item": map[string]any{"id": "answer", "type": "agentMessage", "text": answer}}})
	inputTokens, outputTokens, reasoningTokens := fakeCodexTokenUsage(state.mode, state.turnCount)
	write(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": state.threadID, "turn": map[string]any{"id": turnID, "status": "completed"}}})
	write(map[string]any{"method": "thread/tokenUsage/updated", "params": map[string]any{"threadId": state.threadID, "turnId": turnID, "tokenUsage": map[string]any{"total": map[string]any{"inputTokens": inputTokens, "cachedInputTokens": 2 * state.turnCount, "outputTokens": outputTokens, "reasoningOutputTokens": reasoningTokens}}}})
}

func fakeCodexAnswer(mode string, turnCount int) string {
	switch mode {
	case "session":
		return "I would ask my owner to add semedit because its precise edits eased this task."
	case "feedback":
		if turnCount > 1 {
			return "I would not ask my owner to add semedit because I could not assess its value in this task."
		}
		return "Task completed."
	default:
		return "DONE"
	}
}

func fakeCodexTokenUsage(mode string, turnCount int) (int, int, int) {
	inputTokens, outputTokens, reasoningTokens := 10*turnCount, 3*turnCount, turnCount
	switch mode {
	case "session":
		switch turnCount {
		case 1:
			inputTokens, outputTokens = 5, 2
		case 2:
			inputTokens, outputTokens = 12, 4
		default:
			inputTokens, outputTokens = 100000, 100000
		}
	case "staged":
		switch turnCount {
		case 1:
			inputTokens, outputTokens = 5, 2
		default:
			inputTokens, outputTokens = 12, 5
		}
	case "feedback":
		switch turnCount {
		case 1:
			inputTokens, outputTokens = 5, 2
		default:
			inputTokens, outputTokens = 100004, 100002
		}
	}
	return inputTokens, outputTokens, reasoningTokens
}

func advanceFakeCodexTaskCompletion(state *fakeCodexAppServerState) {
	if state.mode == "feedback" && state.turnCount == 1 {
		state.finalTaskCompletion = time.Now()
		return
	}
	if state.mode != "session" || state.turnCount == 2 {
		state.finalTaskCompletion = time.Now()
	}
}

func writeFakeCodexMetrics(state *fakeCodexAppServerState) {
	if state.metricsPath == "" || state.finalTaskCompletion.IsZero() {
		return
	}
	if err := os.MkdirAll(filepath.Dir(state.metricsPath), 0o750); err != nil {
		os.Exit(12)
	}
	record := []map[string]any{{"completed_at": state.finalTaskCompletion, "session_metrics": map[string]any{"server_start_to_initialize_ms": 125, "initialize_to_first_semantic_call_ms": 875}}}
	data, err := json.Marshal(record)
	if err != nil {
		os.Exit(13)
	}
	if err := pipeline.WriteAtomic(state.metricsPath, data); err != nil {
		os.Exit(14)
	}
}

func assertFakeCodexTurnRequestsUseThread(t *testing.T, requests []byte, wantThread string, wantTurns int) {
	t.Helper()
	count := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(requests)), "\n") {
		var request fakeCodexAppServerRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			t.Fatalf("decode app-server request %q: %v", line, err)
		}
		if request.Method != "turn/start" {
			continue
		}
		var params struct {
			ThreadID string `json:"threadId"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil {
			t.Fatalf("decode turn/start params: %v", err)
		}
		if params.ThreadID != wantThread {
			t.Errorf("turn/start used thread %q, want %q", params.ThreadID, wantThread)
		}
		count++
	}
	if count != wantTurns {
		t.Errorf("turn/start requests = %d, want %d", count, wantTurns)
	}
}

func fakeCodexAppServerCLIOverrides(args []string) (bool, []string) {
	enabled := false
	var serverArgs []string
	for index := 0; index+1 < len(args); index++ {
		if args[index] != "-c" {
			continue
		}
		override := args[index+1]
		switch {
		case override == "mcp_servers.semedit.enabled=true":
			enabled = true
		case strings.HasPrefix(override, "mcp_servers.semedit.args="):
			rawArgs := strings.TrimPrefix(override, "mcp_servers.semedit.args=")
			if err := json.Unmarshal([]byte(rawArgs), &serverArgs); err != nil {
				os.Exit(16)
			}
		case strings.HasPrefix(override, "mcp_servers.semedit.default_tools_approval_mode="):
			serverArgs = append(serverArgs, override)
		}
	}
	return enabled, serverArgs
}

func TestCodexAppServerRejectsPersistentThread(t *testing.T) {
	root := t.TempDir()
	installCodexAppServerFake(t, root, "persistent")
	server, err := newCodexAppServer(NewRunner(filepath.Join(root, "scratch")), root, Target{Harness: string(HarnessCodex), Model: "test-model", Effort: "medium"}, ArmBaseline)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := server.close(); closeErr != nil {
			t.Errorf("close app-server: %v", closeErr)
		}
	}()
	_, err = server.run(context.Background(), "prompt", &RunResult{})
	if err == nil || !strings.Contains(err.Error(), "thread is not ephemeral") {
		t.Fatalf("run error = %v, want rejection of persistent thread", err)
	}
}
