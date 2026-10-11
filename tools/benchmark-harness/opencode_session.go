// opencode_session.go owns one sandboxed OpenCode server across all turns in a benchmark session.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type openCodePreparedRuntime struct {
	workDir    string
	target     Target
	arm        ArmType
	configPath string
	configData []byte
	env        []string
}

type openCodeSession struct {
	mu           sync.Mutex
	waitMu       sync.Mutex
	cmd          *exec.Cmd
	done         chan struct{}
	waitErr      error
	cleanupErr   error
	serverCancel context.CancelFunc
	stdout       *boundedOpenCodeBuffer
	stderr       *boundedOpenCodeBuffer
	serverURL    string
	workDir      string
	target       Target
	arm          ArmType
	configPath   string
	configData   []byte
	env          []string
	sessionID    string
	startedAt    time.Time
	closed       bool
	sandboxPath  string
	profilePath  string
}

type boundedOpenCodeBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	limit  int
}

func (b *boundedOpenCodeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		_, _ = b.buffer.Write(p[:remaining])
	}
	return len(p), nil
}

func (b *boundedOpenCodeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func newOpenCodeSession(ctx context.Context, runner *Runner, workDir string, target Target, arm ArmType) (*openCodeSession, error) {
	runtime, err := prepareOpenCodeRuntime(ctx, runner, workDir, target, arm)
	if err != nil {
		return nil, err
	}
	return newOpenCodeSessionWithPreparedRuntime(ctx, runner, workDir, target, arm, runtime)
}

func prepareOpenCodeRuntime(ctx context.Context, runner *Runner, workDir string, target Target, arm ArmType) (openCodePreparedRuntime, error) {
	configPath, env, err := runner.openCodeEnvironment(ctx, workDir, arm, target)
	if err != nil {
		return openCodePreparedRuntime{}, fmt.Errorf("prepare OpenCode environment: %w", err)
	}
	model := openCodeProviderModel(target.Model)
	if strings.HasPrefix(model, "openrouter/") && strings.TrimSpace(environmentLookup(env, openRouterAPIKeyEnv)) == "" {
		return openCodePreparedRuntime{}, fmt.Errorf("%s is not set; OpenCode benchmarks require OpenRouter authentication", openRouterAPIKeyEnv)
	}
	// #nosec G304 -- configPath is generated inside the isolated fixture runtime directory.
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return openCodePreparedRuntime{}, fmt.Errorf("snapshot OpenCode configuration: %w", err)
	}
	return openCodePreparedRuntime{
		workDir: workDir, target: target, arm: arm, configPath: configPath,
		configData: append([]byte(nil), configData...), env: append([]string(nil), env...),
	}, nil
}

func newOpenCodeSessionWithPreparedRuntime(ctx context.Context, runner *Runner, workDir string, target Target, arm ArmType, runtime openCodePreparedRuntime) (*openCodeSession, error) {
	if runtime.workDir != workDir || runtime.target != target || runtime.arm != arm {
		return nil, fmt.Errorf("prepared OpenCode runtime does not match session workspace, target, or arm")
	}
	configData, err := os.ReadFile(runtime.configPath)
	if err != nil {
		return nil, fmt.Errorf("snapshot OpenCode configuration: %w", err)
	}
	if !bytes.Equal(configData, runtime.configData) {
		return nil, fmt.Errorf("OpenCode configuration changed before server startup")
	}
	model := openCodeProviderModel(target.Model)
	if strings.HasPrefix(model, "openrouter/") && strings.TrimSpace(environmentLookup(runtime.env, openRouterAPIKeyEnv)) == "" {
		return nil, fmt.Errorf("%s is not set; OpenCode benchmarks require OpenRouter authentication", openRouterAPIKeyEnv)
	}
	openCodePath, err := exec.LookPath("opencode")
	if err != nil {
		return nil, fmt.Errorf("locate OpenCode executable: %w", err)
	}
	address, port, err := openCodeLoopbackAddress(ctx)
	if err != nil {
		return nil, err
	}
	modelURL, err := runner.openCodeModelEndpointURL(target)
	if err != nil {
		return nil, err
	}
	sandboxPath, profilePath, err := runner.openCodeSessionSandboxProfile(workDir, openCodePath, modelURL, port)
	if err != nil {
		return nil, fmt.Errorf("prepare OpenCode filesystem containment: %w", err)
	}
	serverCtx, serverCancel := context.WithCancel(context.Background())
	// #nosec G204 -- executable paths are resolved locally; server arguments are fixed except for an allocated loopback port.
	cmd := exec.CommandContext(serverCtx, sandboxPath, "-f", profilePath, openCodePath, "serve", "--hostname", "127.0.0.1", "--port", port)
	cmd.Cancel = func() error { return signalOpenCodeServerProcess(cmd, false) }
	cmd.WaitDelay = 5 * time.Second
	cmd.Dir = workDir
	cmd.Env = append([]string(nil), runtime.env...)
	server := &openCodeSession{
		cmd: cmd, done: make(chan struct{}), stdout: &boundedOpenCodeBuffer{limit: maxOpenCodeStderrBytes},
		stderr: &boundedOpenCodeBuffer{limit: maxOpenCodeStderrBytes}, serverURL: "http://" + address,
		workDir: workDir, target: target, arm: arm, configPath: runtime.configPath, configData: append([]byte(nil), runtime.configData...), env: append([]string(nil), runtime.env...),
		serverCancel: serverCancel, sandboxPath: sandboxPath, profilePath: profilePath,
	}
	configureOpenCodeServerProcess(cmd)
	cmd.Stdout = server.stdout
	cmd.Stderr = server.stderr
	if err := cmd.Start(); err != nil {
		serverCancel()
		return nil, fmt.Errorf("start sandboxed OpenCode server using %s: %w", runtime.configPath, err)
	}
	server.startedAt = time.Now()
	go server.waitForServerExit()
	if err := server.waitReady(ctx); err != nil {
		return server, errors.Join(err, server.close())
	}
	return server, nil
}

func openCodeLoopbackAddress(ctx context.Context) (string, string, error) {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		return "", "", fmt.Errorf("reserve OpenCode loopback port: %w", err)
	}
	address := listener.Addr().String()
	closeErr := listener.Close()
	if closeErr != nil {
		return "", "", fmt.Errorf("release reserved OpenCode loopback port: %w", closeErr)
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", "", fmt.Errorf("parse reserved OpenCode loopback address %q: %w", address, err)
	}
	return address, port, nil
}

func (s *openCodeSession) waitReady(ctx context.Context) error {
	client := &http.Client{Timeout: 250 * time.Millisecond}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-s.done:
			if waitErr := s.serverWaitError(); waitErr != nil {
				return fmt.Errorf("OpenCode server exited before becoming ready: %w: %s", waitErr, s.diagnostic())
			}
			return fmt.Errorf("OpenCode server exited before becoming ready: %s", s.diagnostic())
		default:
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.serverURL+"/global/health", nil)
		if err == nil {
			response, requestErr := client.Do(request)
			if requestErr == nil {
				_ = response.Body.Close()
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for OpenCode server readiness: %w", ctx.Err())
		case <-deadline.C:
			return fmt.Errorf("OpenCode server did not become ready at %s: %s", s.serverURL, s.diagnostic())
		case <-ticker.C:
		}
	}
}

func (s *openCodeSession) diagnostic() string {
	message := strings.TrimSpace(s.stderr.String())
	if message == "" {
		message = strings.TrimSpace(s.stdout.String())
	}
	return sanitizeOpenRouterDiagnostic(message, s.env)
}

func (s *openCodeSession) run(ctx context.Context, prompt string, result *RunResult, resumeID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expectedSessionID, err := s.validateRun(ctx, resumeID)
	if err != nil {
		return resumeID, err
	}
	workDir, err := filepath.Abs(s.workDir)
	if err != nil {
		return resumeID, fmt.Errorf("resolve OpenCode fixture directory: %w", err)
	}
	return s.runAttached(ctx, prompt, result, expectedSessionID, workDir)
}

func (s *openCodeSession) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	select {
	case <-s.done:
		if s.serverCancel != nil {
			s.serverCancel()
		}
		if waitErr := s.serverWaitError(); waitErr != nil {
			return fmt.Errorf("OpenCode server exited before session cleanup: %w", waitErr)
		}
		return nil
	default:
	}
	var shutdownErr error
	if err := s.disposeInstance(); err != nil {
		shutdownErr = errors.Join(shutdownErr, err)
	}
	if s.serverCancel != nil {
		s.serverCancel()
	} else if err := signalOpenCodeServerProcess(s.cmd, false); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("signal OpenCode server shutdown: %w", err))
	}
	select {
	case <-s.done:
		return errors.Join(shutdownErr, s.serverCleanupError())
	case <-time.After(5 * time.Second):
	}
	if err := signalOpenCodeServerProcess(s.cmd, true); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("force stop OpenCode server process group: %w", err))
	}
	select {
	case <-s.done:
		return errors.Join(shutdownErr, s.serverCleanupError())
	case <-time.After(5 * time.Second):
		return errors.Join(shutdownErr, fmt.Errorf("OpenCode server process group did not exit after force stop"))
	}
}

func (s *openCodeSession) verifyLiveSession(ctx context.Context) error {
	if s.sessionID == "" {
		return fmt.Errorf("OpenCode server session ID is missing")
	}
	requestURL := s.serverURL + "/session/" + url.PathEscape(s.sessionID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build OpenCode session check: %w", err)
	}
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return fmt.Errorf("check original OpenCode session %q: %w", s.sessionID, err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("original OpenCode session %q is unavailable: HTTP %d", s.sessionID, response.StatusCode)
	}
	return nil
}

func (s *openCodeSession) disposeInstance() error {
	workDir, err := filepath.Abs(s.workDir)
	if err != nil {
		return fmt.Errorf("resolve OpenCode fixture directory for instance disposal: %w", err)
	}
	requestURL := s.serverURL + "/instance/dispose?directory=" + url.QueryEscape(workDir)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build OpenCode instance disposal request: %w", err)
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		return fmt.Errorf("dispose OpenCode fixture instance: %w", err)
	}
	_ = response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("dispose OpenCode fixture instance: HTTP %d", response.StatusCode)
	}
	return nil
}

func (s *openCodeSession) verifyConfiguration() error {
	if s.configPath == "" {
		return nil
	}
	current, err := os.ReadFile(s.configPath)
	if err != nil {
		return fmt.Errorf("read original OpenCode configuration: %w", err)
	}
	if !bytes.Equal(current, s.configData) {
		return fmt.Errorf("OpenCode configuration changed during the live session")
	}
	return nil
}

func (s *openCodeSession) serverWaitError() error {
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	if s.cleanupErr != nil {
		return errors.Join(s.waitErr, fmt.Errorf("reap OpenCode server process group: %w", s.cleanupErr))
	}
	return s.waitErr
}

func (s *openCodeSession) serverCleanupError() error {
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	if s.cleanupErr == nil {
		return nil
	}
	return fmt.Errorf("reap OpenCode server process group: %w", s.cleanupErr)
}

func (s *openCodeSession) validateRun(ctx context.Context, resumeID string) (string, error) {
	if s.closed {
		return resumeID, fmt.Errorf("cannot continue closed OpenCode server session")
	}
	if err := s.verifyConfiguration(); err != nil {
		return resumeID, err
	}
	select {
	case <-s.done:
		if waitErr := s.serverWaitError(); waitErr != nil {
			return resumeID, fmt.Errorf("OpenCode server exited before task turn: %w: %s", waitErr, s.diagnostic())
		}
		return resumeID, fmt.Errorf("OpenCode server exited before task turn: %s", s.diagnostic())
	default:
	}
	if s.sessionID == "" && resumeID != "" {
		return resumeID, fmt.Errorf("cannot continue OpenCode session %q without its original live server session", resumeID)
	}
	if s.sessionID != "" && (resumeID == "" || resumeID != s.sessionID) {
		return resumeID, fmt.Errorf("OpenCode follow-up session %q does not match original session %q", resumeID, s.sessionID)
	}
	if s.sessionID != "" {
		if err := s.verifyLiveSession(ctx); err != nil {
			return resumeID, err
		}
	}
	return s.sessionID, nil
}

func (s *openCodeSession) runAttached(ctx context.Context, prompt string, result *RunResult, expectedSessionID, workDir string) (string, error) {
	args := []string{"run", "--format", "json", "--auto", "--dir", workDir, "--attach", s.serverURL}
	if s.sessionID != "" {
		args = append(args, "--session", s.sessionID)
	}
	args = append(args, "--model", openCodeProviderModel(s.target.Model))
	if s.target.Effort != "" {
		args = append(args, "--variant", s.target.Effort)
	}
	args = append(args, prompt)
	openCodePath, err := exec.LookPath("opencode")
	if err != nil {
		return expectedSessionID, fmt.Errorf("locate OpenCode executable for attached turn: %w", err)
	}
	if s.sandboxPath == "" || s.profilePath == "" {
		return expectedSessionID, fmt.Errorf("OpenCode attached turn has no sandbox profile")
	}
	priorMessages, err := s.snapshotAssistantMessages(ctx, workDir, expectedSessionID)
	if err != nil {
		return expectedSessionID, err
	}
	// #nosec G204 -- OpenCode is resolved from PATH and receives fixed flags plus the benchmark prompt.
	cmd := exec.CommandContext(ctx, s.sandboxPath, append([]string{"-f", s.profilePath, openCodePath}, args...)...)
	cmd.Dir = workDir
	cmd.Env = s.env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return expectedSessionID, fmt.Errorf("open attached OpenCode stdout: %w", err)
	}
	stderr := &boundedOpenCodeBuffer{limit: maxOpenCodeStderrBytes}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return expectedSessionID, fmt.Errorf("start sandboxed attached OpenCode turn: %w", err)
	}
	return s.readAttachedTurn(ctx, cmd, stdout, stderr, result, expectedSessionID, workDir, priorMessages)
}

func (s *openCodeSession) readAttachedTurn(ctx context.Context, cmd *exec.Cmd, stdout io.Reader, stderr *boundedOpenCodeBuffer, result *RunResult, expectedSessionID, workDir string, priorMessages map[string]bool) (string, error) {
	stream := readOpenCodeEventStream(stdout, result, s.sessionID)
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		waitErr = errors.Join(waitErr, ctx.Err())
	}
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		result.OpenCodeExitCode = &code
	}
	result.OpenCodeStderr = sanitizeOpenRouterDiagnostic(boundedDiagnostic([]byte(stderr.String()), maxOpenCodeStderrBytes), s.env)
	if waitErr == nil && stream.sessionID != "" && len(stream.usage.starts) > 0 && openCodeStreamNeedsTerminalRecovery(stream) {
		if err := s.recoverAttachedTurn(ctx, workDir, stream.sessionID, priorMessages, &stream, result); err != nil {
			waitErr = err
		}
	}
	metricsStartedAt := time.Time{}
	if expectedSessionID == "" {
		metricsStartedAt = s.startedAt
		if result.processStartedAt.IsZero() {
			result.processStartedAt = s.startedAt
		}
	}
	applyOpenCodeStreamMetrics(result, stream, metricsStartedAt)
	providerErr := openCodeStreamFailure(stream, waitErr, result.OpenCodeStderr)
	if providerErr != nil {
		providerErr = &openCodeDiagnosticError{cause: providerErr, message: sanitizeOpenRouterDiagnostic(providerErr.Error(), s.env)}
	}
	setOpenCodeObservationState(result, providerErr, stream)
	if expectedSessionID != "" && stream.sessionID != expectedSessionID {
		return expectedSessionID, fmt.Errorf("OpenCode attached turn returned session %q, want original session %q", stream.sessionID, expectedSessionID)
	}
	if stream.sessionID != "" {
		s.sessionID = stream.sessionID
	}
	if providerErr != nil {
		return stream.sessionID, providerErr
	}
	if stream.sessionID == "" {
		return "", fmt.Errorf("OpenCode attached turn did not emit a session ID")
	}
	if stream.validEvents == 0 {
		return stream.sessionID, fmt.Errorf("OpenCode attached turn emitted no valid task events")
	}
	return stream.sessionID, nil
}

func openCodeStreamNeedsTerminalRecovery(stream openCodeEventStream) bool {
	return stream.validEvents > 0 && (!stream.usage.finished || len(stream.usage.starts) > len(stream.usage.steps))
}

func (s *openCodeSession) snapshotAssistantMessages(ctx context.Context, workDir, sessionID string) (map[string]bool, error) {
	if sessionID == "" {
		return map[string]bool{}, nil
	}
	messages, err := s.openCodeSessionMessages(ctx, workDir, sessionID)
	if err != nil {
		return nil, fmt.Errorf("snapshot OpenCode session %q messages before attached turn: %w", sessionID, err)
	}
	seen := make(map[string]bool, len(messages))
	for _, message := range messages {
		if message.Info.Role == "assistant" && message.Info.ID != "" {
			seen[message.Info.ID] = true
		}
	}
	return seen, nil
}

func (s *openCodeSession) recoverAttachedTurn(ctx context.Context, workDir, sessionID string, priorMessages map[string]bool, stream *openCodeEventStream, result *RunResult) error {
	if err := s.waitForOpenCodeSessionIdle(ctx, workDir, sessionID); err != nil {
		return err
	}
	messages, err := s.openCodeSessionMessages(ctx, workDir, sessionID)
	if err != nil {
		return fmt.Errorf("read completed OpenCode session %q messages: %w", sessionID, err)
	}
	for _, message := range messages {
		if message.Info.Role != "assistant" || message.Info.ID == "" || priorMessages[message.Info.ID] {
			continue
		}
		if message.Info.Error != nil {
			recovered, err := json.Marshal(map[string]any{"type": "error", "error": message.Info.Error})
			if err != nil {
				return fmt.Errorf("encode recovered OpenCode provider error: %w", err)
			}
			consumeOpenCodeEvent(stream, result, recovered)
		}
		for _, part := range message.Parts {
			if openCodeRecoveredPartAlreadyObserved(part, *stream) {
				continue
			}
			recovered := map[string]any{"type": "message.part.updated", "properties": map[string]any{"sessionID": sessionID, "part": part}}
			encoded, err := json.Marshal(recovered)
			if err != nil {
				return fmt.Errorf("encode recovered OpenCode message part: %w", err)
			}
			consumeOpenCodeEvent(stream, result, encoded)
		}
	}
	if openCodeStreamNeedsTerminalRecovery(*stream) {
		return fmt.Errorf("OpenCode session %q reached idle without complete terminal token usage", sessionID)
	}
	return nil
}

func openCodeRecoveredPartAlreadyObserved(part map[string]any, stream openCodeEventStream) bool {
	// Tool parts retain their ID while their state changes, so replay them to upgrade
	// a streamed running call to the completed server state.
	if stringValue(part["type"]) == "tool" {
		return false
	}
	id := stringValue(part["id"])
	return id != "" && stream.partIDs[id]
}

type openCodeSessionMessage struct {
	Info struct {
		ID    string `json:"id"`
		Role  string `json:"role"`
		Error any    `json:"error"`
	} `json:"info"`
	Parts []map[string]any `json:"parts"`
}

func (s *openCodeSession) openCodeSessionMessages(ctx context.Context, workDir, sessionID string) (messages []openCodeSessionMessage, returnErr error) {
	endpoint, err := url.Parse(s.serverURL + "/session/" + url.PathEscape(sessionID) + "/message")
	if err != nil {
		return nil, fmt.Errorf("build OpenCode session messages request: %w", err)
	}
	query := endpoint.Query()
	query.Set("directory", workDir)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build OpenCode session messages request: %w", err)
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("read OpenCode session messages: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close OpenCode session messages response: %w", closeErr))
		}
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("read OpenCode session messages: HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("decode OpenCode session messages: %w", err)
	}
	return messages, nil
}

func (s *openCodeSession) waitForOpenCodeSessionIdle(ctx context.Context, workDir, sessionID string) error {
	endpoint, err := url.Parse(s.serverURL + "/session/status")
	if err != nil {
		return fmt.Errorf("build OpenCode session status request: %w", err)
	}
	query := endpoint.Query()
	query.Set("directory", workDir)
	endpoint.RawQuery = query.Encode()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return fmt.Errorf("build OpenCode session status request: %w", err)
		}
		response, err := (&http.Client{}).Do(request)
		if err != nil {
			return fmt.Errorf("read OpenCode session status: %w", err)
		}
		var statuses map[string]struct {
			Type string `json:"type"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&statuses)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("read OpenCode session status: HTTP %d", response.StatusCode)
		}
		if decodeErr != nil {
			return fmt.Errorf("decode OpenCode session status: %w", decodeErr)
		}
		if status, found := statuses[sessionID]; found && status.Type == "idle" {
			return nil
		}
		// OpenCode v1.18.30 removes idle sessions from this map. Recovery only
		// starts after a streamed step-start, so an omitted entry here means the
		// observed turn returned to idle rather than that its prompt has not begun.
		if _, found := statuses[sessionID]; !found {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *openCodeSession) waitForServerExit() {
	waitErr := s.cmd.Wait()
	cleanupErr := signalOpenCodeServerProcess(s.cmd, true)
	s.waitMu.Lock()
	s.waitErr = waitErr
	s.cleanupErr = cleanupErr
	s.waitMu.Unlock()
	close(s.done)
}
