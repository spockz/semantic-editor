// opencode_session_test.go proves attached turns reuse one server and reject lost or changed sessions.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenCodeAttachHelperProcess(t *testing.T) {
	t.Helper()
	if os.Getenv("SEMEDIT_OPENCODE_HELPER") != "1" {
		return
	}
	args := os.Args
	separator := slices.Index(args, "--")
	if separator < 0 {
		os.Exit(2)
	}
	args = args[separator+1:]
	if os.Getenv("SEMEDIT_OPENCODE_HELPER_MODE") == "usage" {
		runOpenCodeUsageHelper(args)
		return
	}
	if os.Getenv("SEMEDIT_OPENCODE_HELPER_MODE") == "cancel" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	attachURL := ""
	sessionID := "ses-123"
	for i, arg := range args {
		if arg == "--attach" && i+1 < len(args) {
			attachURL = args[i+1]
		}
		if arg == "--session" && i+1 < len(args) {
			sessionID = args[i+1]
		}
	}
	if os.Getenv("SEMEDIT_OPENCODE_HELPER_MODE") != "eof" {
		response, err := http.Post(attachURL+"/harness/turn", "application/json", strings.NewReader(strings.Join(args, "\n")))
		if err != nil {
			os.Exit(3)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			os.Exit(4)
		}
		mode := os.Getenv("SEMEDIT_OPENCODE_HELPER_MODE")
		if mode == "changed" || mode == "mixed" {
			sessionID = "ses-changed"
		}
		fmt.Printf("{\"type\":\"session.created\",\"properties\":{\"info\":{\"id\":%q}}}\n", sessionID)
		if mode == "unfinished" {
			fmt.Printf("{\"type\":\"message.part.updated\",\"properties\":{\"sessionID\":%q,\"part\":{\"id\":\"tool-1\",\"sessionID\":%q,\"messageID\":\"message-1\",\"type\":\"tool\",\"tool\":\"semantic_lookup\",\"state\":{\"status\":\"running\",\"input\":{\"symbol\":\"target\"}}}}}\n", sessionID, sessionID)
			fmt.Printf("{\"type\":\"step_start\",\"sessionID\":%q,\"part\":{\"id\":\"start-1\",\"messageID\":\"message-1\",\"type\":\"step-start\"}}\n", sessionID)
			os.Exit(0)
		}
		if mode == "mixed" {
			fmt.Println(`{"type":"session.updated","properties":{"info":{"id":"ses-123"}}}`)
		}
		fmt.Println(`{"type":"step_finish","part":{"type":"step-finish","id":"finish-1","reason":"stop","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}}`)
	}
	os.Exit(0)
}

func TestOpenCodeAttachedTurnsReuseServerAndSession(t *testing.T) {
	env := openCodeFakeClient(t, "")
	server, observation := newOpenCodeAttachedTestServer(t)
	defer server.Close()
	workDir := t.TempDir()
	startedAt := time.Now().Add(-time.Second)
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: workDir,
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline,
		startedAt: startedAt, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	for i, prompt := range []string{"first task", "corrective task", "diagnostic reflection"} {
		result := &RunResult{Arm: ArmBaseline}
		sessionID, err := session.run(context.Background(), prompt, result, session.sessionID)
		if err != nil {
			t.Fatalf("run %q: %v", prompt, err)
		}
		if sessionID != "ses-123" {
			t.Fatalf("run %q session ID = %q, want ses-123", prompt, sessionID)
		}
		if i == 0 && !result.processStartedAt.Equal(startedAt) {
			t.Fatalf("initial process origin = %s, want serve launch %s", result.processStartedAt, startedAt)
		}
		if i > 0 && !result.processStartedAt.IsZero() {
			t.Fatalf("follow-up process origin = %s, want per-turn origin assigned by the session", result.processStartedAt)
		}
	}
	assertOpenCodeAttachedTurns(t, observation, server.URL, workDir)
}

func TestOpenCodeAttachedTurnRecoversTerminalUsageFromServer(t *testing.T) {
	env := openCodeFakeClient(t, "unfinished")
	var statusRequests, messageRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/harness/turn":
			w.WriteHeader(http.StatusOK)
		case "/session/status":
			statusRequests.Add(1)
			_, _ = io.WriteString(w, `{}`)
		case "/session/ses-123/message":
			messageRequests.Add(1)
			_, _ = io.WriteString(w, `[{"info":{"id":"message-1","role":"assistant"},"parts":[{"id":"tool-1","sessionID":"ses-123","messageID":"message-1","type":"tool","tool":"semantic_lookup","state":{"status":"completed","input":{"symbol":"target"},"output":{"ok":true}}},{"id":"finish-1","sessionID":"ses-123","messageID":"message-1","type":"step-finish","reason":"stop","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}]}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	result := &RunResult{Arm: ArmBaseline}
	sessionID, err := session.run(t.Context(), "solve", result, "")
	if err != nil {
		t.Fatal(err)
	}
	if sessionID != "ses-123" {
		t.Fatalf("session ID = %q, want ses-123", sessionID)
	}
	if result.OpenCodeExitCode == nil || *result.OpenCodeExitCode != 0 {
		t.Fatalf("OpenCode exit code = %v, want clean exit", result.OpenCodeExitCode)
	}
	if result.PromptTokens != 15 || result.CachedPromptTokens != 5 || result.UncachedPromptTokens != 10 || result.OutputTokens != 4 || result.ReasoningTokens != 2 {
		t.Fatalf("recovered usage = %+v", result)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].FunctionalStatus != ToolCallStatusSucceeded {
		t.Fatalf("recovered tool state = %+v, want completed call", result.ToolCalls)
	}
	if statusRequests.Load() == 0 || messageRequests.Load() == 0 {
		t.Fatalf("server recovery requests: status=%d messages=%d", statusRequests.Load(), messageRequests.Load())
	}
}

func TestOpenCodeAttachedTurnRecoveryPreservesProviderError(t *testing.T) {
	env := append(openCodeFakeClient(t, "unfinished"), "OPENROUTER_API_KEY=recovery-secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/harness/turn":
			w.WriteHeader(http.StatusOK)
		case "/session/status":
			_, _ = io.WriteString(w, `{}`)
		case "/session/ses-123/message":
			_, _ = io.WriteString(w, `[{"info":{"id":"message-1","role":"assistant","error":{"data":{"message":"provider unavailable: recovery-secret"}}},"parts":[{"id":"finish-1","sessionID":"ses-123","messageID":"message-1","type":"step-finish","reason":"stop","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}]}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	_, err := session.run(t.Context(), "solve", &RunResult{Arm: ArmBaseline}, "")
	if err == nil || !strings.Contains(err.Error(), "provider unavailable") {
		t.Fatalf("recovered provider error = %v", err)
	}
	if strings.Contains(err.Error(), "recovery-secret") {
		t.Fatalf("recovered provider error leaked credential: %v", err)
	}
}

func TestOpenCodeAttachedTurnRecoveryExcludesPriorAssistantUsage(t *testing.T) {
	env := openCodeFakeClient(t, "unfinished")
	var messageRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/ses-123", "/harness/turn":
			w.WriteHeader(http.StatusOK)
		case "/session/status":
			_, _ = io.WriteString(w, `{}`)
		case "/session/ses-123/message":
			if messageRequests.Add(1) == 1 {
				_, _ = io.WriteString(w, `[{"info":{"id":"previous","role":"assistant"},"parts":[{"id":"previous-finish","sessionID":"ses-123","messageID":"previous","type":"step-finish","reason":"stop","tokens":{"input":100,"output":200,"reasoning":300,"cache":{"read":400,"write":500}}}]}]`)
				return
			}
			_, _ = io.WriteString(w, `[{"info":{"id":"previous","role":"assistant"},"parts":[{"id":"previous-finish","sessionID":"ses-123","messageID":"previous","type":"step-finish","reason":"stop","tokens":{"input":100,"output":200,"reasoning":300,"cache":{"read":400,"write":500}}}]},{"info":{"id":"message-1","role":"assistant"},"parts":[{"id":"finish-1","sessionID":"ses-123","messageID":"message-1","type":"step-finish","reason":"stop","tokens":{"input":3,"output":4,"reasoning":2,"cache":{"read":5,"write":7}}}]}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(), sessionID: "ses-123",
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	result := &RunResult{Arm: ArmBaseline}
	_, err := session.run(t.Context(), "solve", result, "ses-123")
	if err != nil {
		t.Fatal(err)
	}
	if result.PromptTokens != 15 || result.OutputTokens != 4 || result.ReasoningTokens != 2 {
		t.Fatalf("recovered current-turn usage = %+v", result)
	}
	if messageRequests.Load() != 2 {
		t.Fatalf("message requests = %d, want snapshot and recovery", messageRequests.Load())
	}
}

func TestOpenCodeAttachedTurnRecoveryRejectsIdleWithoutTerminalUsage(t *testing.T) {
	env := openCodeFakeClient(t, "unfinished")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/harness/turn":
			w.WriteHeader(http.StatusOK)
		case "/session/status":
			_, _ = io.WriteString(w, `{}`)
		case "/session/ses-123/message":
			_, _ = io.WriteString(w, `[]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	result := &RunResult{Arm: ArmBaseline}
	_, err := session.run(t.Context(), "solve", result, "")
	if err == nil || !strings.Contains(err.Error(), "reached idle without complete terminal token usage") {
		t.Fatalf("idle recovery error = %v", err)
	}
	if result.toolObservationState != ToolObservationPartial || result.PromptTokens != 0 || result.OutputTokens != 0 {
		t.Fatalf("idle recovery result = %+v", result)
	}
}

type openCodeAttachObservation struct {
	mu            sync.Mutex
	turnRequests  int
	sessionChecks int
	argsByTurn    [][]string
}

func openCodeFakeClient(t *testing.T, mode string) []string {
	t.Helper()
	binDir := t.TempDir()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nexec \"$SEMEDIT_HELPER_BINARY\" -test.run=TestOpenCodeAttachHelperProcess -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "opencode"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	return []string{
		"PATH=" + binDir,
		"SEMEDIT_HELPER_BINARY=" + binary,
		"SEMEDIT_OPENCODE_HELPER=1",
		"SEMEDIT_OPENCODE_HELPER_MODE=" + mode,
	}
}

func installOpenCodeSessionSandbox(t *testing.T, session *openCodeSession) {
	t.Helper()
	binDir := t.TempDir()
	installPassthroughOpenCodeSandbox(t, binDir)
	session.sandboxPath = filepath.Join(binDir, "sandbox-exec")
	session.profilePath = filepath.Join(t.TempDir(), "session.sb")
	if err := os.WriteFile(session.profilePath, []byte("(version 1)\n(deny default)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeAttachedSessionFailureIsExplicit(t *testing.T) {
	t.Run("missing session", TestOpenCodeMissingSessionFails)
	t.Run("changed session", openCodeChangedSessionFails)
	t.Run("mixed session IDs", openCodeMixedSessionIDsFail)
	t.Run("empty stream", TestOpenCodeEmptyStreamFails)
	t.Run("canceled process", TestOpenCodeCanceledTurnFails)
}

func TestOpenCodeSessionLoopbackRuleIsPortScoped(t *testing.T) {
	profile, err := appendOpenCodeSessionLoopbackRule([]byte("(version 1)\n(deny default)\n"), "40123")
	if err != nil {
		t.Fatal(err)
	}
	text := string(profile)
	if !strings.Contains(text, `(allow network-bind (local tcp "localhost:40123"))`) ||
		!strings.Contains(text, `(allow network-inbound (local tcp "localhost:40123"))`) ||
		!strings.Contains(text, `(allow network-outbound (remote tcp "localhost:40123"))`) {
		t.Fatalf("profile does not grant the selected local TCP server port: %s", text)
	}
	if strings.Contains(text, `(remote tcp "localhost:*")`) || strings.Contains(text, `(remote ip "*")`) || strings.Contains(text, `(remote tcp "192.168.1.122:1234")`) || strings.Contains(text, "(allow network-outbound)") {
		t.Fatalf("profile widened network access: %s", text)
	}
	for _, invalid := range []string{"", "0", "65536", "abc"} {
		if _, err := appendOpenCodeSessionLoopbackRule(nil, invalid); err == nil {
			t.Errorf("port %q unexpectedly accepted", invalid)
		}
	}
}

func TestOpenCodeConfigChangesFailBeforeAttachedTurn(t *testing.T) {
	env := openCodeFakeClient(t, "")
	configPath := filepath.Join(t.TempDir(), "opencode.json")
	original := []byte(`{"model":"amdbeast/fake"}`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("changed config should fail before contacting OpenCode")
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline,
		configPath: configPath, configData: original, env: env,
	}
	if err := os.WriteFile(configPath, []byte(`{"model":"changed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := session.run(context.Background(), "follow-up", &RunResult{Arm: ArmBaseline}, "")
	if err == nil || !strings.Contains(err.Error(), "configuration changed") {
		t.Fatalf("changed configuration error = %v, want explicit frozen-config failure", err)
	}
}

func newOpenCodeAttachedTestServer(t *testing.T) (*httptest.Server, *openCodeAttachObservation) {
	t.Helper()
	observation := &openCodeAttachObservation{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observation.mu.Lock()
		defer observation.mu.Unlock()
		switch r.URL.Path {
		case "/session/ses-123":
			observation.sessionChecks++
			w.WriteHeader(http.StatusOK)
		case "/session/ses-123/message":
			_, _ = io.WriteString(w, `[]`)
		case "/harness/turn":
			observation.turnRequests++
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read fake attach request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			observation.argsByTurn = append(observation.argsByTurn, strings.Split(string(body), "\n"))
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	return server, observation
}

func assertOpenCodeAttachedTurns(t *testing.T, observation *openCodeAttachObservation, serverURL, workDir string) {
	t.Helper()
	workDirAbs, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatal(err)
	}
	observation.mu.Lock()
	defer observation.mu.Unlock()
	if observation.turnRequests != 3 || len(observation.argsByTurn) != 3 {
		t.Fatalf("attached turns = %d (%d argument sets), want 3", observation.turnRequests, len(observation.argsByTurn))
	}
	if observation.sessionChecks != 2 {
		t.Fatalf("live session checks = %d, want 2", observation.sessionChecks)
	}
	for i, args := range observation.argsByTurn {
		if !slices.Contains(args, serverURL) || !slices.Contains(args, workDirAbs) {
			t.Errorf("turn %d did not reuse the original server and fixture: %v", i+1, args)
		}
		if i == 0 && slices.Contains(args, "ses-123") {
			t.Errorf("initial turn unexpectedly used a prior session ID: %v", args)
		}
		if i > 0 && (!slices.Contains(args, "--session") || !slices.Contains(args, "ses-123")) {
			t.Errorf("turn %d did not continue the original session: %v", i+1, args)
		}
	}
}

func TestOpenCodeMissingSessionFails(t *testing.T) {
	env := openCodeFakeClient(t, "")
	var turnRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/harness/turn" {
			turnRequests.Add(1)
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline,
		env: env, sessionID: "ses-missing",
	}
	_, err := session.run(context.Background(), "follow-up", &RunResult{Arm: ArmBaseline}, "ses-missing")
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("missing session error = %v, want explicit unavailable-session failure", err)
	}
	if turnRequests.Load() != 0 {
		t.Fatalf("attached turns = %d, want no call after session lookup failed", turnRequests.Load())
	}
}

func openCodeChangedSessionFails(t *testing.T) {
	assertOpenCodeSessionIDFailure(t, "changed")
}

func TestOpenCodeEmptyStreamFails(t *testing.T) {
	env := openCodeFakeClient(t, "eof")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	_, err := session.run(context.Background(), "initial", &RunResult{Arm: ArmBaseline}, "")
	if err == nil || !strings.Contains(err.Error(), "session ID") {
		t.Fatalf("empty stream error = %v, want explicit missing-session EOF failure", err)
	}
}

func TestOpenCodeCanceledTurnFails(t *testing.T) {
	env := openCodeFakeClient(t, "cancel")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline, env: env,
	}
	installOpenCodeSessionSandbox(t, session)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := session.run(ctx, "initial", &RunResult{Arm: ArmBaseline}, "")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled attached process error = %v, want context deadline error", err)
	}
}

func openCodeMixedSessionIDsFail(t *testing.T) {
	assertOpenCodeSessionIDFailure(t, "mixed")
}

func TestOpenCodeCloseDisposesExactInstanceBeforeReap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group shutdown is Unix-specific")
	}
	eventPath := filepath.Join(t.TempDir(), "shutdown.log")
	readyPath := filepath.Join(t.TempDir(), "server.ready")
	fixtureDir := t.TempDir()
	fixtureAbs, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/instance/dispose" {
			t.Errorf("dispose request = %s %s, want POST /instance/dispose", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		file, openErr := os.OpenFile(eventPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if openErr != nil {
			t.Errorf("open shutdown event log: %v", openErr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, writeErr := fmt.Fprintf(file, "dispose:%s\n", r.URL.Query().Get("directory"))
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Errorf("write disposal event: %v %v", writeErr, closeErr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	script := `trap 'printf "term\n" >> "$EVENT_FILE"; exit 0' TERM
printf "ready\n" > "$READY_FILE"
while :; do sleep 1; done`
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Env = append(os.Environ(), "EVENT_FILE="+eventPath, "READY_FILE="+readyPath)
	configureOpenCodeServerProcess(cmd)
	if startErr := cmd.Start(); startErr != nil {
		t.Fatal(startErr)
	}
	waitUntil := time.Now().Add(time.Second)
	for {
		if _, statErr := os.Stat(readyPath); statErr == nil {
			break
		}
		if time.Now().After(waitUntil) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatal("fake server did not install its TERM handler")
		}
		time.Sleep(time.Millisecond)
	}
	session := &openCodeSession{
		cmd: cmd, done: make(chan struct{}), serverURL: server.URL, workDir: fixtureDir,
		stdout: &boundedOpenCodeBuffer{limit: 128}, stderr: &boundedOpenCodeBuffer{limit: 128},
	}
	go session.waitForServerExit()
	if closeErr := session.close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	<-session.done
	data, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "dispose:" + fixtureAbs + "\nterm\n"
	if string(data) != want {
		t.Fatalf("shutdown events = %q, want %q", data, want)
	}
	if session.serverWaitError() != nil {
		t.Fatalf("server wait error = %v, want clean reap", session.serverWaitError())
	}
}

func TestOpenCodeServerCrashKillsResidualProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group shutdown is Unix-specific")
	}
	marker := filepath.Join(t.TempDir(), "child-survived")
	script := `(sleep 1; printf survived > "$MARKER_PATH") &
exit 23`
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Env = append(os.Environ(), "MARKER_PATH="+marker)
	configureOpenCodeServerProcess(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	session := &openCodeSession{cmd: cmd, done: make(chan struct{})}
	go session.waitForServerExit()
	select {
	case <-session.done:
	case <-time.After(5 * time.Second):
		t.Fatal("crashed OpenCode server was not reaped")
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("residual child survived the crashed OpenCode server")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat child marker: %v", err)
	}
	if err := session.serverWaitError(); err == nil {
		t.Fatal("server crash was not retained as an explicit session failure")
	}
}

func TestOpenCodeProviderReusesPreparedConfigurationAtLaunch(t *testing.T) {
	root := t.TempDir()
	goPath, lookupErr := exec.LookPath("go")
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	executableDir := filepath.Join(root, "bin")
	if mkdirErr := os.Mkdir(executableDir, 0o750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	if symlinkErr := os.Symlink(goPath, filepath.Join(executableDir, "go")); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	t.Setenv("PATH", executableDir)
	workspace := filepath.Join(root, "fixture")
	if mkdirErr := os.Mkdir(workspace, 0o750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	runner := NewRunner(filepath.Join(root, ".scratch", "benchmarks"), WithLocalModelLimits(localModelLimits{Context: 65536, Output: 8192}), WithLocalModelRelayURL("http://127.0.0.1:12345/v1"))
	provider := &openCodeProvider{runner: runner}
	target := Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}
	if prepErr := provider.prepareFixture(t.Context(), workspace, target, ArmBaseline, false); prepErr != nil {
		t.Fatal(prepErr)
	}
	configPath := filepath.Join(workspace, ".scratch", "opencode", "opencode.json")
	before, beforeErr := os.Stat(configPath)
	if beforeErr != nil {
		t.Fatal(beforeErr)
	}
	if _, runErr := provider.run(t.Context(), workspace, target, "hi", &RunResult{Arm: ArmBaseline}, ""); runErr == nil || !strings.Contains(runErr.Error(), "locate OpenCode executable") {
		t.Fatalf("expected launch failure after preparation, got %v", runErr)
	}
	after, afterErr := os.Stat(configPath)
	if afterErr != nil {
		t.Fatal(afterErr)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("provider recreated prepared configuration during task turn")
	}
}

func TestOpenCodeLocalCompactionThreshold(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		context, output, threshold int
	}{
		{"saluki", 65536, 8192, 49152},
		{"different_context", 131072, 8192, 98304},
		{"fractional_threshold", 65537, 8192, 49152},
		{"output_requires_earlier_compaction", 16384, 8192, 8192},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := NewRunner(t.TempDir(), WithLocalModelLimits(localModelLimits{Context: tc.context, Output: tc.output}), WithLocalModelRelayURL("http://127.0.0.1:12345/v1"))
			config, err := runner.openCodeConfig(t.TempDir(), ArmBaseline, Target{Harness: string(HarnessOpenCode), Model: "amdbeast/saluki"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			compaction := config["compaction"].(map[string]any)
			if compaction["auto"] != true {
				t.Fatal("automatic compaction is disabled")
			}
			providers := config["provider"].(map[string]any)
			models := providers["amdbeast"].(map[string]any)["models"].(map[string]any)
			limit := models["saluki"].(map[string]any)["limit"].(map[string]int)
			// OpenCode 1.18.30 uses input-reserved; without input it ignores reserved.
			threshold := limit["input"] - compaction["reserved"].(int)
			if threshold != tc.threshold {
				t.Fatalf("compaction threshold = %d, want %d", threshold, tc.threshold)
			}
			if limit["context"] != tc.context || limit["output"] != tc.output {
				t.Fatalf("model capacity changed: %v", limit)
			}
			if threshold+limit["output"] > limit["context"] {
				t.Fatal("compaction leaves insufficient room for output")
			}
		})
	}
}

func assertOpenCodeSessionIDFailure(t *testing.T, mode string) {
	t.Helper()
	env := openCodeFakeClient(t, mode)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/ses-123" || r.URL.Path == "/harness/turn" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	session := &openCodeSession{
		done: make(chan struct{}), serverURL: server.URL, workDir: t.TempDir(),
		target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/fake"}, arm: ArmBaseline,
		env: env, sessionID: "ses-123",
	}
	installOpenCodeSessionSandbox(t, session)
	_, err := session.run(context.Background(), "follow-up", &RunResult{Arm: ArmBaseline}, "ses-123")
	if err == nil {
		t.Fatal("session identity change unexpectedly succeeded")
	}
	if session.sessionID != "ses-123" {
		t.Fatalf("session ID changed to %q after rejected response", session.sessionID)
	}
}

func openCodeUsageCount(path string) int {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(11)
	}
	count := len(data) + 1
	if err := os.WriteFile(path, []byte(strings.Repeat("x", count)), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(12)
	}
	return count
}

func serveOpenCodeUsageHelper(args []string) {
	port := ""
	for index, arg := range args {
		if arg == "--port" && index+1 < len(args) {
			port = args[index+1]
			break
		}
	}
	if port == "" {
		os.Exit(13)
	}
	openCodeUsageCount(os.Getenv("SEMEDIT_USAGE_SERVER_COUNT"))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/global/health":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/message"):
			_, _ = io.WriteString(w, `[]`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/session/"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && (r.URL.Path == "/instance/dispose" || r.URL.Path == "/harness/turn"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
	if err := http.ListenAndServe("127.0.0.1:"+port, handler); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(14)
	}
}

func runOpenCodeUsageTurn(args []string) {
	attachURL, workDir, model, sessionID := "", "", "", ""
	for index, arg := range args {
		if index+1 >= len(args) {
			continue
		}
		switch arg {
		case "--attach":
			attachURL = args[index+1]
		case "--dir":
			workDir = args[index+1]
		case "--model":
			model = args[index+1]
		case "--session":
			sessionID = args[index+1]
		}
	}
	count := openCodeUsageCount(os.Getenv("SEMEDIT_USAGE_TURN_COUNT"))
	if attachURL == "" || workDir == "" || model != "amdbeast/qwen36-iq4-xs" || count == 1 && sessionID != "" || count > 1 && sessionID != "fake-session" {
		os.Exit(15)
	}
	config, err := os.ReadFile(os.Getenv("OPENCODE_CONFIG"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(16)
	}
	if writeErr := os.WriteFile(os.Getenv("SEMEDIT_CAPTURE_CONFIG"), config, 0o600); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(17)
	}
	response, err := http.Post(attachURL+"/harness/turn", "application/json", strings.NewReader(strings.Join(args, "\n")))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(18)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		os.Exit(19)
	}
	input, read, write, output, reasoning, tool := usageTurnTokens(count)
	if count == 3 || count == 4 {
		time.Sleep(150 * time.Millisecond)
	}
	if count == 1 {
		if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main\nfunc Old() {}\n"), 0o600); err != nil {
			os.Exit(20)
		}
	}
	if count == 2 {
		if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main\nfunc Done() {}\n"), 0o600); err != nil {
			os.Exit(21)
		}
	}
	fmt.Printf("{\"type\":\"session.created\",\"properties\":{\"info\":{\"id\":\"fake-session\"}}}\n")
	fmt.Printf("{\"type\":\"step_start\",\"sessionID\":\"fake-session\",\"part\":{\"id\":\"start-%d\",\"type\":\"step-start\"}}\n", count)
	fmt.Printf("{\"type\":\"tool_use\",\"part\":{\"id\":\"tool-%d\",\"type\":\"tool\",\"tool\":\"%s\",\"server\":\"semedit\",\"state\":{\"status\":\"completed\",\"input\":{},\"output\":\"ok\"}}}\n", count, tool)
	fmt.Println(`{"type":"text","part":{"type":"text","text":"Improvement: make diagnostic summaries surface the remaining problem count before paging details."}}`)
	fmt.Printf("{\"type\":\"step_finish\",\"part\":{\"id\":\"finish-%d\",\"type\":\"step-finish\",\"reason\":\"stop\",\"tokens\":{\"input\":%d,\"output\":%d,\"reasoning\":%d,\"cache\":{\"read\":%d,\"write\":%d}}}}\n", count, input, output, reasoning, read, write)
	os.Exit(0)
}

func usageTurnTokens(count int) (input, read, write, output, reasoning int, tool string) {
	switch count {
	case 1:
		return 5, 3, 1, 2, 1, "semantic_lookup"
	case 2:
		return 7, 4, 2, 2, 2, "semantic_replace_body"
	case 3, 4:
		return 100000, 100000, 100000, 100000, 100000, "semantic_insert_construct"
	default:
		os.Exit(22)
		return 0, 0, 0, 0, 0, ""
	}
}

func runOpenCodeUsageHelper(args []string) {
	if len(args) == 0 {
		os.Exit(23)
	}
	switch args[0] {
	case "serve":
		serveOpenCodeUsageHelper(args[1:])
	case "run":
		runOpenCodeUsageTurn(args[1:])
	default:
		os.Exit(24)
	}
}
