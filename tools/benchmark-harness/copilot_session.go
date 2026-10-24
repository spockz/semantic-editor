// Keeps one Copilot ACP process and its MCP children alive across benchmark turns while validating protocol evidence.

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type copilotProvider struct {
	runner  *Runner
	runtime *copilotPreparedRuntime
	session *copilotSession
	config  adapterSessionConfig
}

type copilotWireMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

type copilotRead struct {
	message copilotWireMessage
	raw     json.RawMessage
	err     error
}

type copilotDiagnosticBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

type copilotUsage struct {
	Input      *int `json:"inputTokens"`
	Output     *int `json:"outputTokens"`
	CacheRead  *int `json:"cacheReadTokens"`
	CacheWrite *int `json:"cacheWriteTokens"`
	Reasoning  *int `json:"reasoningTokens"`
}

type copilotSession struct {
	command      *exec.Cmd
	stdin        io.WriteCloser
	events       chan copilotRead
	done         chan struct{}
	stop         chan struct{}
	waitErr      error
	stderr       copilotDiagnosticBuffer
	sessionID    string
	agentName    string
	agentVersion string
	nextID       int
	workDir      string
	target       Target
	arm          ArmType
	startedAt    time.Time
	totals       copilotUsage
	failed       error
	closed       bool
}

type copilotSessionStartError struct {
	startedAt time.Time
	stderr    string
	err       error
}

func (e *copilotSessionStartError) Error() string { return e.err.Error() }

func (e *copilotSessionStartError) Unwrap() error { return e.err }

type copilotTurn struct {
	result        *RunResult
	sessionID     string
	expectedModel string
	usage         copilotUsage
	calls         map[string]int
	completed     map[string]bool
	standardCalls map[string]bool
	usageCalls    int
	firstEvent    time.Time
	tracker       codexToolTracker
	err           error
}

func (b *copilotDiagnosticBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(data)
	const limit = 64 * 1024
	if len(data) > limit {
		data = data[len(data)-limit:]
		b.data.Reset()
	}
	if excess := b.data.Len() + len(data) - limit; excess > 0 {
		b.data.Next(excess)
	}
	_, err := b.data.Write(data)
	return original, err
}

func (b *copilotDiagnosticBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.String()
}

func (p *copilotProvider) run(ctx context.Context, workDir string, target Target, prompt string, result *RunResult, resumeID string) (string, error) {
	if p.session == nil {
		if resumeID != "" {
			return "", fmt.Errorf("cannot resume Copilot session without its live process")
		}
		command, err := p.runner.newCopilotCommand(ctx, workDir, target, result.Arm, p.runtime)
		if err != nil {
			return "", err
		}
		p.session, err = startCopilotSession(ctx, command, workDir, target, result.Arm)
		if err != nil {
			if startErr, ok := errors.AsType[*copilotSessionStartError](err); ok {
				result.processStartedAt = startErr.startedAt
			}
			return "", err
		}
	}
	s := p.session
	if s.workDir != workDir || s.target != target || s.arm != result.Arm {
		return s.sessionID, fmt.Errorf("copilot session workspace, target, or arm changed")
	}
	if resumeID != "" && resumeID != s.sessionID {
		return s.sessionID, fmt.Errorf("copilot session identity changed")
	}
	return s.sessionID, s.run(ctx, prompt, result)
}

func (p *copilotProvider) close() error {
	if p.session == nil {
		return nil
	}
	return p.session.close()
}

func (p *copilotProvider) prepare(ctx context.Context, input adapterPreparationInput) (adapterStageResult, error) {
	started := time.Now()
	p.config = input.Session
	err := p.prepareFixture(ctx, input.Session.WorkDir, input.Session.Target, input.Session.Arm, input.Session.External)
	return completeAdapterStage(adapterStage(adapterPreparationStage, started, time.Now()), adapterPreparationStage, started, err), err
}

func (p *copilotProvider) warmup(_ context.Context) (adapterStageResult, error) {
	return adapterNoopStage(adapterWarmupStage), nil
}

func (p *copilotProvider) runTurn(ctx context.Context, input adapterInvocation) (adapterTurnResult, error) {
	started := time.Now()
	turn := &RunResult{Target: p.config.Target, Arm: p.config.Arm}
	sessionID, err := p.run(ctx, p.config.WorkDir, p.config.Target, input.Prompt, turn, input.ResumeID)
	stage := adapterStage(adapterTurnStage, started, time.Now())
	stage.ProcessStartedAt = cloneTimePtr(nonzeroTime(turn.processStartedAt))
	stage.SessionID = sessionID
	return adapterTurnResult{Stage: completeAdapterStage(stage, adapterTurnStage, started, err), Observation: observationFromRunResult(turn)}, err
}

func (p *copilotProvider) shutdown() adapterShutdownResult {
	started := time.Now()
	err := p.close()
	stage := completeAdapterStage(adapterStage(adapterShutdownStage, started, time.Now()), adapterShutdownStage, started, err)
	shutdown := adapterShutdownFromError(stage, err)
	if p.session != nil {
		shutdown.Stderr = p.session.stderr.String()
	}
	return shutdown
}

func startCopilotSession(ctx context.Context, command *exec.Cmd, workDir string, target Target, arm ArmType) (*copilotSession, error) {
	s := &copilotSession{command: command, events: make(chan copilotRead, 256), done: make(chan struct{}), stop: make(chan struct{}), workDir: workDir, target: target, arm: arm, startedAt: time.Now()}
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open Copilot stdin: %w", err)
	}
	s.stdin = stdin
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open Copilot stdout: %w", err), stdin.Close())
	}
	command.Stderr = &s.stderr
	if err = command.Start(); err != nil {
		return nil, errors.Join(fmt.Errorf("start Copilot: %w", err), stdin.Close(), stdout.Close())
	}
	s.startedAt = time.Now()
	go s.read(stdout)
	events := []string{"assistant.usage", "tool.execution_start", "tool.execution_complete", "session.error", "model.call_failure"}
	initializeRaw, err := s.request(ctx, "initialize", map[string]any{"protocolVersion": 1, "clientInfo": map[string]string{"name": "semedit-benchmark", "version": "1"}, "clientCapabilities": map[string]any{"_meta": map[string]any{"github.com/copilot": map[string]any{"events": events}}}}, nil)
	if err == nil {
		var initialized struct {
			ProtocolVersion int `json:"protocolVersion"`
			AgentInfo       struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"agentInfo"`
		}
		if decodeErr := json.Unmarshal(initializeRaw, &initialized); decodeErr != nil {
			err = fmt.Errorf("decode Copilot initialize: %w", decodeErr)
		} else if initialized.ProtocolVersion != 1 {
			err = fmt.Errorf("unsupported Copilot ACP protocol version %d", initialized.ProtocolVersion)
		} else {
			s.agentName = initialized.AgentInfo.Name
			s.agentVersion = initialized.AgentInfo.Version
		}
	}
	if err == nil {
		err = s.createAndSelectModel(ctx, workDir)
	}
	if err != nil {
		closeErr := s.close()
		return nil, &copilotSessionStartError{startedAt: s.startedAt, stderr: s.stderr.String(), err: errors.Join(s.diagnosticError(err), closeErr)}
	}
	return s, nil
}

func (s *copilotSession) read(stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		raw := append(json.RawMessage(nil), scanner.Bytes()...)
		var message copilotWireMessage
		err := json.Unmarshal(raw, &message)
		if err != nil {
			err = fmt.Errorf("decode Copilot protocol message: %w", err)
		}
		select {
		case s.events <- copilotRead{message: message, raw: raw, err: err}:
		case <-s.stop:
		}
		if err != nil {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case s.events <- copilotRead{err: fmt.Errorf("read Copilot protocol: %w", err)}:
		case <-s.stop:
		}
	}
	s.waitErr = s.command.Wait()
	close(s.done)
	close(s.events)
}

func (s *copilotSession) write(value any) error {
	if err := json.NewEncoder(s.stdin).Encode(value); err != nil {
		return fmt.Errorf("write Copilot protocol: %w", err)
	}
	return nil
}

func (s *copilotSession) request(ctx context.Context, method string, params any, turn *copilotTurn) (json.RawMessage, error) {
	s.nextID++
	id := s.nextID
	if err := s.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		read, readErr := s.receive(ctx, method)
		if readErr != nil {
			return nil, readErr
		}
		message := read.message
		if turn != nil {
			turn.result.rawEvents = append(turn.result.rawEvents, read.raw)
		}
		if message.Method != "" {
			if len(message.ID) > 0 {
				if answerErr := s.answerRequest(message); answerErr != nil {
					return nil, answerErr
				}
			} else if turn != nil {
				turn.observe(message)
			}
			continue
		}
		var responseID int
		if decodeErr := json.Unmarshal(message.ID, &responseID); decodeErr != nil {
			return nil, fmt.Errorf("decode Copilot response ID: %w", decodeErr)
		}
		if responseID != id {
			return nil, fmt.Errorf("copilot returned unexpected response ID %d", responseID)
		}
		if hasJSONValue(message.Error) {
			return nil, fmt.Errorf("copilot %s: %s", method, message.Error)
		}
		return message.Result, nil
	}
}

func (s *copilotSession) answerRequest(message copilotWireMessage) error {
	var response any
	if message.Method == "session/request_permission" {
		var params struct {
			SessionID string `json:"sessionId"`
			Options   []struct {
				OptionID string `json:"optionId"`
				Kind     string `json:"kind"`
			} `json:"options"`
		}
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return fmt.Errorf("decode Copilot permission: %w", err)
		}
		if params.SessionID != s.sessionID {
			return fmt.Errorf("copilot permission has mismatched session identity")
		}
		for _, option := range params.Options {
			if option.Kind == "allow_once" {
				response = map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": option.OptionID}}
				break
			}
		}
		if response == nil {
			return fmt.Errorf("copilot permission lacks allow_once option")
		}
	} else {
		return s.write(map[string]any{"jsonrpc": "2.0", "id": message.ID, "error": map[string]any{"code": -32601, "message": "unsupported benchmark client request"}})
	}
	return s.write(map[string]any{"jsonrpc": "2.0", "id": message.ID, "result": response})
}

func (s *copilotSession) run(ctx context.Context, prompt string, result *RunResult) error {
	if s.failed != nil {
		return fmt.Errorf("cannot continue failed Copilot process: %w", s.failed)
	}
	if s.closed {
		return fmt.Errorf("copilot session is closed")
	}
	result.processStartedAt = time.Now()
	if s.totals.Input == nil {
		result.processStartedAt = s.startedAt
	}
	if result.Provenance == nil {
		result.Provenance = make(ProvenanceSet)
	}
	result.Provenance["copilot_agent_name"] = s.agentName
	result.Provenance["copilot_agent_version"] = s.agentVersion
	result.Provenance["copilot_protocol_version"] = "1"
	result.Provenance["copilot_model_selection"] = "acp_session_set_config_option"
	turn := &copilotTurn{result: result, sessionID: s.sessionID, expectedModel: copilotWireModel(s.target), calls: map[string]int{}, completed: map[string]bool{}, standardCalls: map[string]bool{}}
	response, err := s.request(ctx, "session/prompt", map[string]any{"sessionId": s.sessionID, "prompt": []map[string]string{{"type": "text", "text": prompt}}}, turn)
	if err == nil {
		err = turn.finish(response, s)
	}
	err = errors.Join(err, turn.err)
	if !turn.firstEvent.IsZero() {
		value := turn.firstEvent.Sub(result.processStartedAt)
		result.ProcessStartToFirstEvent = &value
		if !turn.tracker.firstToolCallAt.IsZero() {
			toolValue := turn.tracker.firstToolCallAt.Sub(turn.firstEvent)
			result.FirstEventToFirstToolCall = &toolValue
		}
	}
	refreshToolCallCounts(result)
	result.Turns = 1
	result.InternalTurns = turn.tracker.internalTurns
	result.InitialLoadTurns = turn.tracker.initialLoadTurns
	result.MCPLoadTurns = turn.tracker.mcpLoadTurns
	if strings.Contains(s.stderr.String(), "raw event notifications") {
		err = errors.Join(err, fmt.Errorf("copilot dropped raw event notifications"))
	}
	if err != nil {
		err = s.diagnosticError(err)
		s.failed = err
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = err.Error()
	} else {
		result.toolObservationState = ToolObservationComplete
	}
	return err
}

func (t *copilotTurn) observe(message copilotWireMessage) {
	var params struct {
		SessionID   string          `json:"sessionId"`
		Type        string          `json:"type"`
		Data        json.RawMessage `json:"data"`
		DataOmitted string          `json:"dataOmitted"`
		Update      struct {
			SessionUpdate string          `json:"sessionUpdate"`
			ToolCallID    string          `json:"toolCallId"`
			Content       json.RawMessage `json:"content"`
		} `json:"update"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		t.err = errors.Join(t.err, fmt.Errorf("decode Copilot event: %w", err))
		return
	}
	if params.SessionID == "" || params.SessionID != t.sessionID {
		t.err = errors.Join(t.err, fmt.Errorf("copilot event has missing or mismatched session identity"))
		return
	}
	if message.Method == "session/update" {
		if params.Update.SessionUpdate == "agent_message_chunk" {
			t.appendResponseChunk(params.Update.Content)
		}
		if params.Update.SessionUpdate == "tool_call" {
			t.standardCalls[params.Update.ToolCallID] = true
		}
		return
	}
	if message.Method != "github.com/copilot/sessionEvent" {
		return
	}
	if params.DataOmitted != "" {
		t.err = errors.Join(t.err, fmt.Errorf("copilot omitted %s payload: %s", params.Type, params.DataOmitted))
		return
	}
	if t.firstEvent.IsZero() {
		t.firstEvent = time.Now()
	}
	switch params.Type {
	case "assistant.usage":
		var identity struct {
			Model        string `json:"model"`
			FinishReason string `json:"finishReason"`
		}
		if err := json.Unmarshal(params.Data, &identity); err != nil {
			t.err = errors.Join(t.err, fmt.Errorf("decode Copilot usage identity: %w", err))
			return
		}
		if t.expectedModel != "" && identity.Model != t.expectedModel {
			t.err = errors.Join(t.err, fmt.Errorf("copilot reported model %q instead of selected %q", identity.Model, t.expectedModel))
		}
		switch identity.FinishReason {
		case "length", "content_filter", "refusal", "error":
			t.err = errors.Join(t.err, fmt.Errorf("copilot model call stopped with %q", identity.FinishReason))
		}
		var usage copilotUsage
		if err := json.Unmarshal(params.Data, &usage); err != nil {
			t.err = errors.Join(t.err, fmt.Errorf("decode Copilot usage: %w", err))
			return
		}
		t.usageCalls++
		if err := t.addUsage(usage); err != nil {
			t.err = errors.Join(t.err, err)
		}
	case "tool.execution_start", "tool.execution_complete":
		t.observeTool(params.Type, params.Data)
	case "session.error", "model.call_failure":
		t.err = errors.Join(t.err, fmt.Errorf("copilot %s: %s", params.Type, params.Data))
	}
}

func copilotAddCounter(total **int, value *int) error {
	if value == nil {
		return nil
	}
	if *value < 0 {
		return fmt.Errorf("copilot usage contains negative tokens")
	}
	if *total == nil {
		initial := 0
		*total = &initial
	}
	if *value > int(^uint(0)>>1)-**total {
		return fmt.Errorf("copilot usage token overflow")
	}
	**total += *value
	return nil
}

func (t *copilotTurn) addUsage(usage copilotUsage) error {
	if usage.Input == nil || usage.Output == nil {
		return fmt.Errorf("copilot usage lacks required input or output tokens")
	}
	for _, counter := range []*int{usage.Input, usage.Output, usage.CacheRead, usage.CacheWrite, usage.Reasoning} {
		if counter != nil && *counter < 0 {
			return fmt.Errorf("copilot usage contains negative tokens")
		}
	}
	if usage.CacheRead != nil && *usage.CacheRead > *usage.Input {
		return fmt.Errorf("copilot cached tokens exceed input tokens")
	}
	if usage.CacheRead != nil && usage.CacheWrite != nil && *usage.CacheWrite > *usage.Input-*usage.CacheRead {
		return fmt.Errorf("copilot cached tokens exceed input tokens")
	}
	for _, pair := range []struct {
		total **int
		value *int
	}{{&t.usage.Input, usage.Input}, {&t.usage.Output, usage.Output}, {&t.usage.CacheRead, usage.CacheRead}, {&t.usage.CacheWrite, usage.CacheWrite}, {&t.usage.Reasoning, usage.Reasoning}} {
		if err := copilotAddCounter(pair.total, pair.value); err != nil {
			return err
		}
	}
	if t.usageCalls == 1 {
		t.result.InitialContextTokens = *usage.Input
	}
	t.result.PromptTokens = *t.usage.Input
	t.result.OutputTokens = *t.usage.Output
	if t.usage.CacheRead != nil {
		t.result.CachedPromptTokens = *t.usage.CacheRead
	}
	if usage.CacheRead != nil && usage.CacheWrite != nil {
		t.result.UncachedPromptTokens += *usage.Input - *usage.CacheRead - *usage.CacheWrite
	}
	if t.usage.Reasoning != nil {
		t.result.ReasoningTokens = *t.usage.Reasoning
	}
	if t.result.Provenance == nil {
		t.result.Provenance = make(ProvenanceSet)
	}
	for _, counter := range []struct {
		name  string
		value *int
	}{{"cache_read", usage.CacheRead}, {"cache_write", usage.CacheWrite}, {"reasoning", usage.Reasoning}} {
		if counter.value == nil {
			t.result.Provenance["copilot_usage_"+counter.name] = "unavailable"
		}
	}
	if usage.CacheRead == nil || usage.CacheWrite == nil {
		t.result.Provenance["copilot_usage_uncached"] = "unavailable"
	}
	return nil
}

func (t *copilotTurn) observeTool(kind string, raw json.RawMessage) {
	var data struct {
		ToolCallID    string          `json:"toolCallId"`
		ToolName      string          `json:"toolName"`
		MCPToolName   string          `json:"mcpToolName"`
		MCPServerName string          `json:"mcpServerName"`
		Arguments     json.RawMessage `json:"arguments"`
		Result        json.RawMessage `json:"result"`
		Error         json.RawMessage `json:"error"`
		Success       *bool           `json:"success"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.err = errors.Join(t.err, fmt.Errorf("decode Copilot tool event: %w", err))
		return
	}
	if data.ToolCallID == "" {
		t.err = errors.Join(t.err, fmt.Errorf("copilot tool event omitted call ID"))
		return
	}
	index, exists := t.calls[data.ToolCallID]
	if kind == "tool.execution_start" {
		if exists {
			t.err = errors.Join(t.err, fmt.Errorf("duplicate Copilot tool start %s", data.ToolCallID))
			return
		}
		name := data.ToolName
		if data.MCPToolName != "" {
			name = data.MCPToolName
		}
		if name == "" {
			t.err = errors.Join(t.err, fmt.Errorf("copilot tool omitted name"))
			return
		}
		t.tracker.record(t.result, data.ToolCallID, ToolCall{Name: name, Server: data.MCPServerName, Arguments: data.Arguments, TransportStatus: ToolCallStatusUnknown, FunctionalStatus: ToolCallStatusUnknown}, time.Now())
		t.calls[data.ToolCallID] = len(t.result.ToolCalls) - 1
		return
	}
	if !exists || t.completed[data.ToolCallID] || data.Success == nil {
		t.err = errors.Join(t.err, fmt.Errorf("unmatched or invalid Copilot tool completion %s", data.ToolCallID))
		return
	}
	t.completed[data.ToolCallID] = true
	call := t.result.ToolCalls[index]
	status := "failed"
	if *data.Success {
		status = "completed"
	}
	call = codexToolOutcome(call.Name, call.Server, call.Arguments, status, data.Result, data.Error)
	t.result.ToolCalls[index] = call
	applyMCPStartupMetrics(t.result, call.MCPStartupMetrics)
}

func (t *copilotTurn) finish(raw json.RawMessage, s *copilotSession) error {
	var response struct {
		StopReason string `json:"stopReason"`
		Usage      struct {
			Input  *int `json:"inputTokens"`
			Output *int `json:"outputTokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return fmt.Errorf("decode Copilot prompt result: %w", err)
	}
	if response.StopReason != "end_turn" {
		return fmt.Errorf("copilot prompt stopped with %q", response.StopReason)
	}
	if t.usageCalls == 0 || t.usage.Input == nil || t.usage.Output == nil || response.Usage.Input == nil || response.Usage.Output == nil {
		return fmt.Errorf("copilot prompt lacks complete terminal usage")
	}
	beforeInput, beforeOutput := 0, 0
	if s.totals.Input != nil {
		beforeInput = *s.totals.Input
	}
	if s.totals.Output != nil {
		beforeOutput = *s.totals.Output
	}
	if *response.Usage.Input-beforeInput != *t.usage.Input || *response.Usage.Output-beforeOutput != *t.usage.Output {
		return fmt.Errorf("copilot raw usage does not match cumulative prompt usage")
	}
	s.totals.Input = response.Usage.Input
	s.totals.Output = response.Usage.Output
	for id := range t.calls {
		if !t.completed[id] {
			return fmt.Errorf("copilot tool %s has no completion", id)
		}
	}
	for id := range t.standardCalls {
		if _, ok := t.calls[id]; !ok {
			return fmt.Errorf("copilot tool %s lacks raw observation", id)
		}
	}
	return nil
}

func (s *copilotSession) close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.stop)
	closeErr := s.stdin.Close()
	if errors.Is(closeErr, os.ErrClosed) {
		closeErr = nil
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.done:
			return errors.Join(closeErr, s.waitErr, s.cleanupProcessGroup())
		case <-s.events:
		case <-timer.C:
			var killErr error
			if s.command.SysProcAttr != nil && s.command.SysProcAttr.Setpgid {
				killErr = s.cleanupProcessGroup()
			} else {
				killErr = s.command.Process.Kill()
			}
			if errors.Is(killErr, os.ErrProcessDone) {
				killErr = nil
			}
			select {
			case <-s.done:
				return errors.Join(closeErr, killErr, s.waitErr, s.cleanupProcessGroup())
			case <-time.After(5 * time.Second):
				return errors.Join(closeErr, killErr, fmt.Errorf("copilot did not exit after termination"))
			}
		}
	}
}

func (s *copilotSession) diagnosticError(err error) error {
	if err == nil {
		return nil
	}
	diagnostic := strings.TrimSpace(s.stderr.String())
	if diagnostic == "" {
		return err
	}
	if len(diagnostic) > 4096 {
		diagnostic = diagnostic[len(diagnostic)-4096:]
	}
	return fmt.Errorf("%w; Copilot stderr: %s", err, diagnostic)
}

func (s *copilotSession) cleanupProcessGroup() error {
	if s.command.SysProcAttr == nil || !s.command.SysProcAttr.Setpgid || s.command.Process == nil || s.command.Process.Pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-s.command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("terminate Copilot process group: %w", err)
	}
	return nil
}

func (s *copilotSession) receive(ctx context.Context, method string) (copilotRead, error) {
	select {
	case <-ctx.Done():
		return copilotRead{}, fmt.Errorf("wait for Copilot %s: %w", method, ctx.Err())
	case read, ok := <-s.events:
		if !ok {
			return copilotRead{}, fmt.Errorf("copilot exited during %s: %w", method, errors.Join(s.waitErr, io.ErrUnexpectedEOF))
		}
		if read.err != nil {
			return copilotRead{}, read.err
		}
		return read, nil
	}
}

func (t *copilotTurn) appendResponseChunk(raw json.RawMessage) {
	var content struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &content); err != nil {
		t.err = errors.Join(t.err, fmt.Errorf("decode Copilot response chunk: %w", err))
		return
	}
	if content.Type == "text" {
		t.result.agentResponse += content.Text
	}
}

type copilotModelCatalog struct {
	CurrentModelID  string                   `json:"currentModelId"`
	AvailableModels []copilotAdvertisedModel `json:"availableModels"`
}

type copilotAdvertisedModel struct {
	ModelID string `json:"modelId"`
}

func (s *copilotSession) createAndSelectModel(ctx context.Context, workDir string) error {
	raw, err := s.request(ctx, "session/new", map[string]any{"cwd": workDir, "mcpServers": []any{}}, nil)
	if err != nil {
		return err
	}
	var created struct {
		SessionID string               `json:"sessionId"`
		Models    *copilotModelCatalog `json:"models"`
	}
	if decodeErr := json.Unmarshal(raw, &created); decodeErr != nil {
		return fmt.Errorf("decode Copilot session: %w", decodeErr)
	}
	if strings.TrimSpace(created.SessionID) == "" {
		return fmt.Errorf("copilot omitted session ID")
	}
	s.sessionID = created.SessionID
	model := copilotWireModel(s.target)
	if model == "" {
		return fmt.Errorf("copilot target model is missing")
	}
	_, local := copilotLocalModel(s.target)
	if !local {
		if catalogErr := requireCopilotCloudModel(created.Models, model); catalogErr != nil {
			return catalogErr
		}
	}
	return s.selectModel(ctx, model, !local)
}

func requireCopilotCloudModel(catalog *copilotModelCatalog, model string) error {
	if catalog == nil || len(catalog.AvailableModels) == 0 {
		return fmt.Errorf("copilot cloud model catalog unavailable; verify that the cloud credential has Copilot access and the requested model entitlement")
	}
	for _, available := range catalog.AvailableModels {
		if available.ModelID == model {
			return nil
		}
	}
	return fmt.Errorf("copilot cloud model %q is not advertised for this credential; select an available model or verify its entitlement", model)
}

func (s *copilotSession) selectModel(ctx context.Context, model string, cloud bool) error {
	raw, err := s.request(ctx, "session/set_config_option", map[string]any{"sessionId": s.sessionID, "configId": "model", "value": model}, nil)
	if err != nil {
		return err
	}
	if !cloud {
		return nil
	}
	var response struct {
		ConfigOptions []struct {
			ID           string          `json:"id"`
			CurrentValue json.RawMessage `json:"currentValue"`
		} `json:"configOptions"`
	}
	if decodeErr := json.Unmarshal(raw, &response); decodeErr != nil {
		return fmt.Errorf("decode Copilot model selection: %w", decodeErr)
	}
	for _, option := range response.ConfigOptions {
		if option.ID != "model" {
			continue
		}
		var current string
		if decodeErr := json.Unmarshal(option.CurrentValue, &current); decodeErr != nil {
			return fmt.Errorf("decode Copilot selected model: %w", decodeErr)
		}
		if current == model {
			return nil
		}
		break
	}
	return fmt.Errorf("copilot cloud model selection did not confirm requested model %q; no prompt was sent", model)
}
