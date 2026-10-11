// Package main provides a persistent Codex app-server adapter for benchmark turns.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func (w appServerStderrWriter) Write(data []byte) (int, error) {
	originalLength := len(data)
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := maxCodexStderrBytes - w.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = w.buffer.Write(data)
	}
	return originalLength, nil
}

const appServerTurnCompletedMethod = "turn/completed"

type appServerThreadStartResponse struct {
	Thread struct {
		ID        string `json:"id"`
		Ephemeral bool   `json:"ephemeral"`
	} `json:"thread"`
	Cwd             string  `json:"cwd"`
	Model           string  `json:"model"`
	ReasoningEffort *string `json:"reasoningEffort"`
	ApprovalPolicy  string  `json:"approvalPolicy"`
	Sandbox         struct {
		Type          string `json:"type"`
		NetworkAccess bool   `json:"networkAccess"`
	} `json:"sandbox"`
}

type appServerStderrWriter struct {
	mu     *sync.Mutex
	buffer *bytes.Buffer
}

func (s *codexAppServer) run(ctx context.Context, prompt string, result *RunResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("codex app-server requires a result")
	}
	if s.closed {
		return "", fmt.Errorf("codex app-server is already closed")
	}
	firstMeasuredTurn := s.turnCount == 0
	if err := s.ensureStarted(ctx, result); err != nil {
		result.CodexStderr = s.stderrDiagnostic()
		if captureErr := s.capturePendingEvents(result); captureErr != nil {
			err = errors.Join(err, captureErr)
		}
		setCodexObservationState(result, err, false)
		return "", err
	}
	if s.threadID == "" {
		err := fmt.Errorf("codex app-server thread ID is missing")
		setCodexObservationState(result, err, false)
		return "", err
	}
	if err := s.processThreadNotifications(result); err != nil {
		result.CodexStderr = s.stderrDiagnostic()
		if captureErr := s.capturePendingEvents(result); captureErr != nil {
			err = errors.Join(err, captureErr)
		}
		setCodexObservationState(result, err, false)
		return s.threadID, err
	}
	err := s.startTurn(ctx, prompt, result)
	if captureErr := s.capturePendingEvents(result); captureErr != nil {
		err = errors.Join(err, captureErr)
	}
	result.CodexStderr = s.stderrDiagnostic()
	if firstMeasuredTurn {
		s.pendingMu.Lock()
		if !s.firstEventAt.IsZero() {
			value := s.firstEventAt.Sub(s.startedAt)
			result.ProcessStartToFirstEvent = &value
		}
		s.pendingMu.Unlock()
		if !s.toolTracker.firstToolCallAt.IsZero() && !s.firstEventAt.IsZero() {
			value := s.toolTracker.firstToolCallAt.Sub(s.firstEventAt)
			result.FirstEventToFirstToolCall = &value
		}
	}
	setCodexObservationState(result, err, err == nil)
	if err != nil {
		return s.threadID, err
	}
	return s.threadID, nil
}

func (s *codexAppServer) close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	defer s.processCancel()
	if s.stdin != nil {
		if err := s.stdin.Close(); err != nil {
			s.closeErr = fmt.Errorf("close Codex app-server stdin: %w", err)
		}
	}
	if s.cmd == nil || s.cmd.Process == nil {
		return s.closeErr
	}
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			s.closeErr = errors.Join(s.closeErr, fmt.Errorf("wait for Codex app-server: %w", err))
		}
	case <-timer.C:
		_ = signalCodexAppServerProcess(s.cmd.Process.Pid, false)
		killTimer := time.NewTimer(time.Second)
		defer killTimer.Stop()
		select {
		case err := <-done:
			if err != nil {
				s.closeErr = errors.Join(s.closeErr, fmt.Errorf("stop Codex app-server: %w", err))
			}
		case <-killTimer.C:
			_ = signalCodexAppServerProcess(s.cmd.Process.Pid, true)
			if err := <-done; err != nil {
				s.closeErr = errors.Join(s.closeErr, fmt.Errorf("kill Codex app-server: %w", err))
			}
		}
	}
	select {
	case <-s.readerDone:
	case <-time.After(time.Second):
		if s.stdout != nil {
			_ = s.stdout.Close()
		}
		select {
		case <-s.readerDone:
		case <-time.After(time.Second):
			s.closeErr = errors.Join(s.closeErr, fmt.Errorf("codex app-server reader did not stop"))
		}
	}
	return s.closeErr
}

func (s *codexAppServer) ensureStarted(ctx context.Context, result *RunResult) error {
	if s.processStarted {
		return s.fatal()
	}
	override, err := s.runner.codexMCPOverride(s.arm, s.workDir)
	if err != nil {
		return fmt.Errorf("resolve Codex MCP configuration: %w", err)
	}
	if err := s.startProcess(ctx, result, override); err != nil {
		return err
	}
	return s.startThread(ctx)
}

func (s *codexAppServer) startTurn(ctx context.Context, prompt string, result *RunResult) error {
	s.toolTracker = codexToolTracker{}
	result.Turns = 1
	if err := s.turnSettingsMatch(); err != nil {
		return err
	}
	beforeUsage := s.lastUsage
	var response struct {
		Turn struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"turn"`
	}
	params := map[string]any{"threadId": s.threadID, "input": []any{map[string]any{"type": "text", "text": prompt}}}
	if err := s.rpc(ctx, "turn/start", params, &response); err != nil {
		return fmt.Errorf("start Codex turn: %w", err)
	}
	if response.Turn.ID == "" {
		return fmt.Errorf("codex turn/start response omitted turn ID")
	}
	if response.Turn.Status == "failed" || response.Turn.Status == "interrupted" {
		return fmt.Errorf("codex turn %s started with terminal status %q", response.Turn.ID, response.Turn.Status)
	}
	return s.awaitTurn(ctx, response.Turn.ID, result, beforeUsage)
}

func (s *codexAppServer) observeMessage(msg appServerMessage, result *RunResult, before appServerTokenUsage, turnID string) error {
	if err := s.fatal(); err != nil {
		return err
	}
	switch {
	case msg.Method == "thread/started":
		return s.observeThreadStarted(msg.Params)
	case msg.Method == "mcpServerStatus/updated":
		return s.observeMCPStatus(msg.Params)
	case msg.Method == "thread/tokenUsage/updated":
		return s.observeTokenUsage(msg.Params, result, before, turnID)
	case msg.Method == "turn/started" || msg.Method == appServerTurnCompletedMethod:
		return s.observeTurnEvent(msg, result, turnID)
	case strings.HasPrefix(msg.Method, "item/"):
		return s.observeItemEvent(msg, result, turnID)
	default:
		return nil
	}
}

func (s *codexAppServer) appendAppServerItems(result *RunResult, items []codexItem) {
	for index := range items {
		item := items[index]
		switch item.Type {
		case "agentMessage":
			item.Type = "agent_message"
			if item.Status == "inProgress" {
				continue
			}
			itemKey := "agent:" + item.ID
			if item.ID != "" {
				if s.toolTracker.calls == nil {
					s.toolTracker.calls = make(map[string]int)
				}
				if _, seen := s.toolTracker.calls[itemKey]; seen {
					continue
				}
				s.toolTracker.calls[itemKey] = -1
			} else if strings.TrimSpace(result.agentResponse) == strings.TrimSpace(item.Text) {
				continue
			}
			appendCodexAgentResponse(result, &item)
		case "mcpToolCall":
			item.Type = "mcp_tool_call"
		case "commandExecution":
			item.Type = "command_execution"
		case "fileChange":
			item.Type = codexFileChangeItemType
		case "toolCall":
			item.Type = "tool_call"
		}
		if item.Status == "" {
			item.Status = "completed"
		}
		if item.Status == "inProgress" {
			item.Status = "in_progress"
		}
		if item.Status == "declined" {
			item.Status = "failed"
		}
		s.toolTracker.observe(result, &item, time.Now())
	}
	result.Turns = 1
	result.InternalTurns = s.toolTracker.internalTurns
	result.InitialLoadTurns = s.toolTracker.initialLoadTurns
	result.MCPLoadTurns = s.toolTracker.mcpLoadTurns
	result.ToolCount = len(result.ToolCalls)
	refreshMCPVerified(result)
}

func (s *codexAppServer) rpc(ctx context.Context, method string, params any, response any) error {
	id := strconv.FormatUint(s.nextID.Add(1), 10)
	resultCh := make(chan appServerMessage, 1)
	s.pendingMu.Lock()
	s.pending[id] = resultCh
	s.pendingMu.Unlock()
	defer func() { s.pendingMu.Lock(); delete(s.pending, id); s.pendingMu.Unlock() }()
	if err := s.writeMessage(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.readerDone:
		if err := s.fatal(); err != nil {
			return err
		}
		return fmt.Errorf("codex app-server stdout closed")
	case msg := <-resultCh:
		if msg.Error != nil {
			return fmt.Errorf("codex app-server %s failed (%d): %s", method, msg.Error.Code, msg.Error.Message)
		}
		if response != nil && hasJSONValue(msg.Result) {
			if err := json.Unmarshal(msg.Result, response); err != nil {
				return fmt.Errorf("decode Codex app-server %s response: %w", method, err)
			}
		}
		return nil
	}
}

func (s *codexAppServer) notify(method string, params any) error {
	return s.writeMessage(map[string]any{"method": method, "params": params})
}

func (s *codexAppServer) writeMessage(message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode Codex app-server message: %w", err)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.stdin == nil {
		return fmt.Errorf("codex app-server stdin is unavailable")
	}
	if _, err = s.stdin.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write Codex app-server message: %w", err)
	}
	return nil
}

func (s *codexAppServer) readLoop() {
	defer close(s.readerDone)
	scanner := bufio.NewScanner(s.stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		s.recordFirstEvent()
		message, err := s.decodeAppServerMessage(scanner.Bytes())
		if err != nil {
			s.setFatal(err)
			return
		}
		if err := s.dispatchAppServerMessage(message); err != nil {
			s.setFatal(err)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		s.setFatal(fmt.Errorf("read Codex app-server output: %w", err))
	}
}

func (s *codexAppServer) setFatal(err error) {
	if err == nil {
		return
	}
	s.fatalMu.Lock()
	if s.fatalErr == nil {
		s.fatalErr = err
	}
	s.fatalMu.Unlock()
}

func (s *codexAppServer) fatal() error {
	s.fatalMu.Lock()
	defer s.fatalMu.Unlock()
	return s.fatalErr
}

func (s *codexAppServer) stderrDiagnostic() string {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return boundedDiagnostic(s.stderr.Bytes(), maxCodexStderrBytes)
}

func (s *codexAppServer) capturePendingEvents(result *RunResult) error {
	s.captureServerRequests(result)
	for {
		select {
		case msg := <-s.notifications:
			result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), msg.Raw...))
			if isTurnScopedAppServerNotification(msg.Method) {
				return fmt.Errorf("unexpected Codex turn notification %q after completion barrier", msg.Method)
			}
			if msg.Method == "mcpServerStatus/updated" {
				if err := s.observeMCPStatus(msg.Params); err != nil {
					return err
				}
			}
		default:
			return nil
		}
	}
}

func (s *codexAppServer) interruptAndDrain(turnID string, result *RunResult, before appServerTokenUsage) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_ = s.rpc(ctx, "turn/interrupt", map[string]any{"threadId": s.threadID, "turnId": turnID}, nil)
	cancel()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.readerDone:
			return
		case <-timer.C:
			if s.cmd != nil && s.cmd.Process != nil {
				_ = signalCodexAppServerProcess(s.cmd.Process.Pid, false)
			}
			return
		case msg := <-s.notifications:
			result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), msg.Raw...))
			if err := s.observeMessage(msg, result, before, turnID); err != nil {
				s.setFatal(err)
				return
			}
			if msg.Method == appServerTurnCompletedMethod {
				var event struct {
					ThreadID string `json:"threadId"`
					Turn     struct {
						ID     string `json:"id"`
						Status string `json:"status"`
					} `json:"turn"`
				}
				if err := json.Unmarshal(msg.Params, &event); err != nil {
					s.setFatal(fmt.Errorf("decode interrupted Codex turn completion: %w", err))
					return
				}
				if event.ThreadID != s.threadID || event.Turn.ID != turnID {
					s.setFatal(fmt.Errorf("interrupted Codex turn completion ID mismatch"))
					return
				}
				if event.Turn.Status == "completed" || event.Turn.Status == "interrupted" || event.Turn.Status == "failed" {
					return
				}
			}
		}
	}
}

func (s *codexAppServer) processThreadNotifications(result *RunResult) error {
	if err := s.fatal(); err != nil {
		return err
	}
	for {
		select {
		case msg := <-s.notifications:
			result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), msg.Raw...))
			if isTurnScopedAppServerNotification(msg.Method) {
				return fmt.Errorf("unexpected Codex turn notification %q outside an active turn", msg.Method)
			}
			if err := s.validateStartupNotification(msg); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func (s *codexAppServer) startProcess(ctx context.Context, result *RunResult, override string) error {
	args := []string{"app-server", "-c", "sandbox_workspace_write.network_access=false"}
	if override != "" {
		args = append(args, "-c", override)
	}
	if isSemeditArm(s.arm) {
		args = append(args,
			"-c", s.runner.codexMCPBinaryOverride(),
			"-c", codexMCPEnabledOverride(),
			"-c", codexMCPGoEnvironmentOverride(),
			"-c", "mcp_servers.semedit.required=true",
			"-c", `mcp_servers.semedit.default_tools_approval_mode="approve"`,
		)
	}
	cmd := exec.CommandContext(s.processCtx, "codex", args...)
	cmd.Dir = s.workDir
	env, err := fixtureGoEnvironment(ctx, s.workDir)
	if err != nil {
		return fmt.Errorf("prepare Codex Go environment: %w", err)
	}
	cmd.Env = env
	configureCodexAppServerProcess(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open Codex app-server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open Codex app-server stdout: %w", err)
	}
	cmd.Stderr = appServerStderrWriter{mu: &s.writeMu, buffer: &s.stderr}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Codex app-server: %w", err)
	}
	s.cmd, s.stdin, s.stdout, s.processStarted, s.startedAt = cmd, stdin, stdout, true, time.Now()
	result.processStartedAt = s.startedAt
	go s.readLoop()
	return nil
}

func (s *codexAppServer) startThread(ctx context.Context) error {
	if err := s.rpc(ctx, "initialize", map[string]any{"clientInfo": map[string]any{"name": "semedit-benchmark", "version": "1"}}, nil); err != nil {
		return fmt.Errorf("initialize Codex app-server: %w", err)
	}
	if err := s.notify("initialized", map[string]any{}); err != nil {
		return fmt.Errorf("notify Codex initialization: %w", err)
	}
	params := map[string]any{"cwd": s.workDir, "approvalPolicy": "never", "sandbox": "workspace-write"}
	params["ephemeral"] = true
	if s.target.Model != "" {
		params["model"] = s.target.Model
	}
	config := map[string]any{"sandbox_workspace_write": map[string]any{"network_access": false}}
	if s.target.Effort != "" {
		config["model_reasoning_effort"] = s.target.Effort
	}
	params["config"] = config
	var response appServerThreadStartResponse
	if err := s.rpc(ctx, "thread/start", params, &response); err != nil {
		return fmt.Errorf("start Codex thread: %w", err)
	}
	if err := validateAppServerThreadStart(response, s.workDir, s.target); err != nil {
		return err
	}
	s.threadID, s.threadModel = response.Thread.ID, response.Model
	if response.ReasoningEffort != nil {
		s.threadEffort = *response.ReasoningEffort
	}
	return nil
}

func (s *codexAppServer) awaitTurn(ctx context.Context, turnID string, result *RunResult, before appServerTokenUsage) error {
	for {
		select {
		case <-ctx.Done():
			s.interruptAndDrain(turnID, result, before)
			return ctx.Err()
		case <-s.readerDone:
			if err := s.fatal(); err != nil {
				return err
			}
			return fmt.Errorf("codex app-server stdout closed before turn %s completed: %s", turnID, s.stderrDiagnostic())
		case msg := <-s.notifications:
			completed, err := s.handleTurnNotification(msg, turnID, result, before)
			if err != nil {
				return err
			}
			if completed {
				return nil
			}
		}
	}
}

func (s *codexAppServer) handleTurnNotification(msg appServerMessage, turnID string, result *RunResult, before appServerTokenUsage) (bool, error) {
	result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), msg.Raw...))
	if err := s.observeMessage(msg, result, before, turnID); err != nil {
		return false, err
	}
	if msg.Method != appServerTurnCompletedMethod {
		return false, nil
	}
	var event struct {
		ThreadID string `json:"threadId"`
		Turn     struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(msg.Params, &event); err != nil {
		return false, fmt.Errorf("decode Codex turn/completed: %w", err)
	}
	if event.ThreadID != s.threadID {
		return false, fmt.Errorf("codex turn/completed thread ID %q does not match %q", event.ThreadID, s.threadID)
	}
	if event.Turn.ID != turnID {
		return false, fmt.Errorf("codex turn/completed ID %q does not match requested turn %q", event.Turn.ID, turnID)
	}
	if event.Turn.Status != "completed" {
		return false, fmt.Errorf("codex turn %s ended with status %q", event.Turn.ID, event.Turn.Status)
	}
	if err := s.flushTurnNotifications(turnID, result, before); err != nil {
		return false, err
	}
	s.turnCount++
	return true, nil
}

func (s *codexAppServer) flushTurnNotifications(turnID string, result *RunResult, before appServerTokenUsage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var response map[string]json.RawMessage
	if err := s.rpc(ctx, "thread/read", map[string]any{"threadId": s.threadID}, &response); err != nil {
		return fmt.Errorf("flush Codex turn notifications: %w", err)
	}
	for {
		select {
		case msg := <-s.notifications:
			result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), msg.Raw...))
			if err := s.observeMessage(msg, result, before, turnID); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func (s *codexAppServer) turnSettingsMatch() error {
	if (s.target.Model != "" && s.threadModel != s.target.Model) || (s.target.Effort != "" && s.threadEffort != s.target.Effort) {
		return fmt.Errorf("codex app-server target changed during thread: %s/%s to %s/%s", s.threadModel, s.threadEffort, s.target.Model, s.target.Effort)
	}
	return nil
}

func (s *codexAppServer) observeThreadStarted(params json.RawMessage) error {
	var started struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(params, &started); err != nil {
		return fmt.Errorf("decode Codex thread/started: %w", err)
	}
	if started.Thread.ID == "" {
		return fmt.Errorf("codex thread/started omitted thread ID")
	}
	if s.threadID != "" && started.Thread.ID != s.threadID {
		return fmt.Errorf("codex thread ID %q does not match started thread %q", s.threadID, started.Thread.ID)
	}
	return nil
}

func (s *codexAppServer) observeMCPStatus(params json.RawMessage) error {
	var status struct {
		Name     string `json:"name"`
		State    string `json:"status"`
		Error    string `json:"error"`
		ThreadID string `json:"threadId"`
	}
	if err := json.Unmarshal(params, &status); err != nil {
		return fmt.Errorf("decode Codex MCP status: %w", err)
	}
	if status.ThreadID != "" && status.ThreadID != s.threadID {
		return fmt.Errorf("codex MCP status thread ID %q does not match %q", status.ThreadID, s.threadID)
	}
	if status.Name == "semedit" && (status.State == "failed" || status.State == "cancelled") {
		return fmt.Errorf("codex semedit MCP server %s: %s", status.State, status.Error)
	}
	return nil
}

func (s *codexAppServer) observeTokenUsage(params json.RawMessage, result *RunResult, before appServerTokenUsage, turnID string) error {
	var usage struct {
		ThreadID   string `json:"threadId"`
		TurnID     string `json:"turnId"`
		TokenUsage struct {
			Total appServerTokenUsage `json:"total"`
		} `json:"tokenUsage"`
	}
	if err := json.Unmarshal(params, &usage); err != nil {
		return fmt.Errorf("decode Codex token usage: %w", err)
	}
	if usage.ThreadID != s.threadID {
		return fmt.Errorf("codex token usage thread ID %q does not match %q", usage.ThreadID, s.threadID)
	}
	if usage.TurnID != turnID {
		return fmt.Errorf("codex token usage turn ID %q does not match %q", usage.TurnID, turnID)
	}
	total := usage.TokenUsage.Total
	if total.InputTokens < s.lastUsage.InputTokens || total.CachedInputTokens < s.lastUsage.CachedInputTokens || total.OutputTokens < s.lastUsage.OutputTokens || total.ReasoningOutputTokens < s.lastUsage.ReasoningOutputTokens {
		return fmt.Errorf("codex cumulative token usage regressed")
	}
	if total.CachedInputTokens > total.InputTokens {
		return fmt.Errorf("codex cached input tokens exceed total input tokens")
	}
	s.lastUsage = total
	result.PromptTokens = total.InputTokens - before.InputTokens
	result.CachedPromptTokens = total.CachedInputTokens - before.CachedInputTokens
	result.UncachedPromptTokens = result.PromptTokens - result.CachedPromptTokens
	result.OutputTokens = total.OutputTokens - before.OutputTokens
	result.ReasoningTokens = total.ReasoningOutputTokens - before.ReasoningOutputTokens
	return nil
}

func (s *codexAppServer) observeTurnEvent(msg appServerMessage, result *RunResult, turnID string) error {
	var event struct {
		ThreadID string `json:"threadId"`
		Turn     struct {
			ID     string      `json:"id"`
			Status string      `json:"status"`
			Items  []codexItem `json:"items"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(msg.Params, &event); err != nil {
		return fmt.Errorf("decode Codex %s: %w", msg.Method, err)
	}
	if event.ThreadID != s.threadID {
		return fmt.Errorf("codex %s thread ID %q does not match %q", msg.Method, event.ThreadID, s.threadID)
	}
	if event.Turn.ID != turnID {
		return fmt.Errorf("codex %s ID %q does not match %q", msg.Method, event.Turn.ID, turnID)
	}
	if msg.Method == "turn/started" && event.Turn.Status != "" && event.Turn.Status != "inProgress" {
		return fmt.Errorf("codex turn started with unexpected status %q", event.Turn.Status)
	}
	if msg.Method == appServerTurnCompletedMethod && event.Turn.Status == "" {
		return fmt.Errorf("codex turn/completed omitted status")
	}
	result.Turns = 1
	if msg.Method == appServerTurnCompletedMethod {
		s.appendAppServerItems(result, event.Turn.Items)
	}
	return nil
}

func (s *codexAppServer) observeItemEvent(msg appServerMessage, result *RunResult, turnID string) error {
	var event struct {
		ThreadID string     `json:"threadId"`
		TurnID   string     `json:"turnId"`
		Item     *codexItem `json:"item"`
	}
	if err := json.Unmarshal(msg.Params, &event); err != nil {
		return fmt.Errorf("decode Codex %s: %w", msg.Method, err)
	}
	if event.ThreadID != s.threadID {
		return fmt.Errorf("codex item notification thread ID %q does not match %q", event.ThreadID, s.threadID)
	}
	if event.TurnID != turnID {
		return fmt.Errorf("codex item notification turn ID %q does not match %q", event.TurnID, turnID)
	}
	if event.Item == nil {
		return nil
	}
	if msg.Method == "item/started" && event.Item.Type == "agentMessage" {
		return nil
	}
	var envelope struct {
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(msg.Params, &envelope); err != nil {
		return fmt.Errorf("decode Codex item payload: %w", err)
	}
	normalizeAppServerItem(envelope.Item, event.Item)
	s.appendAppServerItems(result, []codexItem{*event.Item})
	return nil
}

func (s *codexAppServer) decodeAppServerMessage(raw []byte) (appServerMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return appServerMessage{}, fmt.Errorf("decode Codex app-server JSON-RPC message: %w", err)
	}
	if rawVersion, exists := fields["jsonrpc"]; exists {
		var version string
		if err := json.Unmarshal(rawVersion, &version); err != nil || version != "2.0" {
			return appServerMessage{}, fmt.Errorf("codex app-server message has invalid JSON-RPC version")
		}
	}
	message := appServerMessage{Raw: append(json.RawMessage(nil), raw...)}
	if err := decodeAppServerField(fields, "id", &message.ID); err != nil {
		return appServerMessage{}, err
	}
	if err := decodeAppServerField(fields, "method", &message.Method); err != nil {
		return appServerMessage{}, err
	}
	if err := decodeAppServerField(fields, "params", &message.Params); err != nil {
		return appServerMessage{}, err
	}
	if err := decodeAppServerField(fields, "result", &message.Result); err != nil {
		return appServerMessage{}, err
	}
	if rawError, exists := fields["error"]; exists {
		var rpcErr struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(rawError, &rpcErr); err != nil {
			return appServerMessage{}, fmt.Errorf("decode Codex app-server RPC error: %w", err)
		}
		message.Error = &appServerRPCError{Code: rpcErr.Code, Message: rpcErr.Message, Data: rpcErr.Data}
	}
	if len(message.ID) > 0 && message.Method == "" {
		_, hasResult := fields["result"]
		_, hasError := fields["error"]
		if hasResult == hasError {
			return appServerMessage{}, fmt.Errorf("codex app-server response must contain exactly one of result or error")
		}
	}
	return message, nil
}

func (s *codexAppServer) dispatchAppServerResponse(message appServerMessage) error {
	var id string
	if err := json.Unmarshal(message.ID, &id); err != nil {
		return fmt.Errorf("decode Codex app-server response ID: %w", err)
	}
	s.pendingMu.Lock()
	ch := s.pending[id]
	s.pendingMu.Unlock()
	if ch == nil {
		return fmt.Errorf("codex app-server response has unmatched ID %q", id)
	}
	ch <- message
	return nil
}

func (s *codexAppServer) dispatchAppServerRequest(message appServerMessage) error {
	s.pendingMu.Lock()
	s.serverRequests = append(s.serverRequests, append(json.RawMessage(nil), message.Raw...))
	s.pendingMu.Unlock()
	if rejection, known := appServerRequestRejection(message.Method); known {
		if err := s.writeMessage(map[string]any{"id": message.ID, "result": rejection}); err != nil {
			return fmt.Errorf("decline Codex app-server request %q: %w", message.Method, err)
		}
	} else if err := s.writeMessage(map[string]any{"id": message.ID, "error": map[string]any{"code": -32601, "message": "client request method is unsupported"}}); err != nil {
		return fmt.Errorf("reject Codex app-server request %q: %w", message.Method, err)
	}
	return fmt.Errorf("unexpected Codex app-server request %q", message.Method)
}

func (s *codexAppServer) dispatchAppServerNotification(message appServerMessage) error {
	select {
	case s.notifications <- message:
		return nil
	case <-s.stopReader:
		return nil
	}
}

func (s *codexAppServer) recordFirstEvent() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.firstEventAt.IsZero() {
		s.firstEventAt = time.Now()
	}
}

func (s *codexAppServer) dispatchAppServerMessage(message appServerMessage) error {
	if len(message.ID) > 0 && message.Method == "" {
		return s.dispatchAppServerResponse(message)
	}
	if message.Method != "" && len(message.ID) > 0 {
		return s.dispatchAppServerRequest(message)
	}
	if message.Method != "" {
		return s.dispatchAppServerNotification(message)
	}
	return fmt.Errorf("codex app-server message omitted both request ID and notification method")
}

func (s *codexAppServer) validateStartedNotification(params json.RawMessage) error {
	var event struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(params, &event); err != nil {
		return fmt.Errorf("decode Codex thread/started: %w", err)
	}
	if event.Thread.ID == "" || event.Thread.ID != s.threadID {
		return fmt.Errorf("codex thread/started ID %q does not match %q", event.Thread.ID, s.threadID)
	}
	return nil
}

func (s *codexAppServer) validateStartupMCPStatus(params json.RawMessage) error {
	var event struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		Error    string `json:"error"`
		ThreadID string `json:"threadId"`
	}
	if err := json.Unmarshal(params, &event); err != nil {
		return fmt.Errorf("decode Codex MCP startup status: %w", err)
	}
	if event.ThreadID != "" && event.ThreadID != s.threadID {
		return fmt.Errorf("codex MCP status thread ID %q does not match %q", event.ThreadID, s.threadID)
	}
	if event.Name == "semedit" && (event.Status == "failed" || event.Status == "cancelled") {
		return fmt.Errorf("codex semedit MCP server %s: %s", event.Status, event.Error)
	}
	return nil
}

func (s *codexAppServer) validateStartupNotification(msg appServerMessage) error {
	switch msg.Method {
	case "thread/started":
		return s.validateStartedNotification(msg.Params)
	case "mcpServerStatus/updated":
		return s.validateStartupMCPStatus(msg.Params)
	default:
		return nil
	}
}

func (s *codexAppServer) captureServerRequests(result *RunResult) {
	s.pendingMu.Lock()
	requests := s.serverRequests
	s.serverRequests = nil
	s.pendingMu.Unlock()
	for _, raw := range requests {
		result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), raw...))
	}
}

type appServerTokenUsage struct {
	InputTokens           int
	CachedInputTokens     int
	OutputTokens          int
	ReasoningOutputTokens int
}

type appServerRPCError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

type appServerMessage struct {
	Raw    json.RawMessage
	ID     json.RawMessage
	Method string
	Params json.RawMessage
	Result json.RawMessage
	Error  *appServerRPCError
}

type codexAppServer struct {
	runner         *Runner
	workDir        string
	target         Target
	arm            ArmType
	processCtx     context.Context
	processCancel  context.CancelFunc
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stderr         bytes.Buffer
	writeMu        sync.Mutex
	pendingMu      sync.Mutex
	pending        map[string]chan appServerMessage
	serverRequests []json.RawMessage
	notifications  chan appServerMessage
	readerDone     chan struct{}
	stopReader     chan struct{}
	nextID         atomic.Uint64
	startedAt      time.Time
	processStarted bool
	threadID       string
	threadModel    string
	threadEffort   string
	closed         bool
	closeMu        sync.Mutex
	closeErr       error
	fatalMu        sync.Mutex
	fatalErr       error
	toolTracker    codexToolTracker
	firstEventAt   time.Time
	turnCount      int
	lastUsage      appServerTokenUsage
}

func newCodexAppServer(runner *Runner, workDir string, target Target, arm ArmType) (*codexAppServer, error) {
	if runner == nil {
		return nil, fmt.Errorf("codex app-server requires a runner")
	}
	if target.Harness != string(HarnessCodex) {
		return nil, fmt.Errorf("codex app-server received harness %q", target.Harness)
	}
	root, err := canonicalFixtureRoot(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve Codex fixture workspace: %w", err)
	}
	processCtx, processCancel := context.WithCancel(context.Background())
	return &codexAppServer{runner: runner, workDir: root, target: target, arm: arm, processCtx: processCtx, processCancel: processCancel, pending: make(map[string]chan appServerMessage), notifications: make(chan appServerMessage, 256), readerDone: make(chan struct{}), stopReader: make(chan struct{})}, nil
}

func appServerRequestRejection(method string) (any, bool) {
	switch method {
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval", "applyPatchApproval", "execCommandApproval":
		return map[string]any{"decision": "decline"}, true
	case "item/permissions/requestApproval":
		return map[string]any{"permissions": map[string]any{}}, true
	case "item/tool/requestUserInput":
		return map[string]any{"answers": map[string]any{}}, true
	case "mcpServer/elicitation/request":
		return map[string]any{"action": "decline"}, true
	default:
		return nil, false
	}
}

func normalizeAppServerItem(rawItem json.RawMessage, item *codexItem) {
	if item == nil || item.Type != "commandExecution" {
		return
	}
	var command struct {
		ExitCode         *int   `json:"exitCode"`
		AggregatedOutput string `json:"aggregatedOutput"`
	}
	if err := json.Unmarshal(rawItem, &command); err != nil {
		item.Status = "failed"
		item.Error = json.RawMessage(`"could not decode Codex command execution result"`)
		return
	}
	if command.ExitCode != nil && *command.ExitCode != 0 {
		item.Status = "failed"
		item.Error, _ = json.Marshal(command.AggregatedOutput)
	}
}

func validateAppServerThreadStart(response appServerThreadStartResponse, workDir string, target Target) error {
	if !response.Thread.Ephemeral {
		return fmt.Errorf("codex benchmark thread is not ephemeral")
	}
	if response.Thread.ID == "" {
		return fmt.Errorf("codex app-server thread/start response omitted thread ID")
	}
	if response.Cwd != workDir {
		return fmt.Errorf("codex thread cwd %q does not match fixture %q", response.Cwd, workDir)
	}
	if response.Model == "" {
		return fmt.Errorf("codex thread/start response omitted model")
	}
	if target.Model != "" && response.Model != target.Model {
		return fmt.Errorf("codex thread model %q does not match requested %q", response.Model, target.Model)
	}
	if target.Effort != "" && (response.ReasoningEffort == nil || *response.ReasoningEffort != target.Effort) {
		actual := ""
		if response.ReasoningEffort != nil {
			actual = *response.ReasoningEffort
		}
		return fmt.Errorf("codex thread reasoning effort %q does not match requested %q", actual, target.Effort)
	}
	if response.ApprovalPolicy != "never" {
		return fmt.Errorf("codex thread approval policy %q does not match never", response.ApprovalPolicy)
	}
	if response.Sandbox.Type != "workspaceWrite" || response.Sandbox.NetworkAccess {
		return fmt.Errorf("codex thread sandbox policy is not workspace-write with network disabled")
	}
	return nil
}

func decodeAppServerField(fields map[string]json.RawMessage, name string, destination any) error {
	value, exists := fields[name]
	if !exists {
		return nil
	}
	if err := json.Unmarshal(value, destination); err != nil {
		return fmt.Errorf("decode Codex app-server %s field: %w", name, err)
	}
	return nil
}

func isTurnScopedAppServerNotification(method string) bool {
	return method == "thread/tokenUsage/updated" || strings.HasPrefix(method, "turn/") || strings.HasPrefix(method, "item/")
}
