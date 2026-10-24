// agy_stream.go owns one persistent Agy stream-json process so every task continuation stays in one live conversation and keeps its MCP server session.

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
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type agySession struct {
	runner              *Runner
	workDir             string
	target              Target
	arm                 ArmType
	process             *agyProcess
	cancel              context.CancelFunc
	transcriptOffset    int64
	configPath          string
	configData          []byte
	closed              bool
	failure             error
	preparedEnvironment []string
}

type agyStreamEvent struct {
	Event          string          `json:"event"`
	ConversationID string          `json:"conversation_id"`
	Result         *AgyResponse    `json:"result"`
	StepUpdate     json.RawMessage `json:"step_update"`
	Raw            json.RawMessage `json:"-"`
}

type agyProcess struct {
	cmd            *exec.Cmd
	startedAt      time.Time
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	waitErr        error
	events         <-chan agyStreamEvent
	readErr        <-chan error
	stderr         *bytes.Buffer
	conversationID string
	lastUsage      AgyResponse
	initialized    bool
	closed         bool
	configPath     string
}

func newAgySession(runner *Runner, workDir string, target Target, arm ArmType) *agySession {
	return &agySession{runner: runner, workDir: workDir, target: target, arm: arm}
}

func buildAgyStreamArguments(workDir string, target Target) []string {
	args := []string{"--input-format", "stream-json", "--output-format", "stream-json", "--dangerously-skip-permissions", "--add-dir", workDir}
	if target.Model != "" {
		args = append(args, "--model", target.Model)
	}
	if target.Effort != "" {
		args = append(args, "--effort", target.Effort)
	}
	return args
}

func prepareAgyMCPConfig(ctx context.Context, runner *Runner, workDir string, arm ArmType) (string, string, []byte, []string, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("resolve Agy workspace: %w", err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Dir(filepath.Dir(runner.baseScratchDir)))
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("resolve repository root for Agy MCP: %w", err)
	}
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("prepare Agy Go environment: %w", err)
	}
	err = writeAgyMCPConfigForRunner(runner, absWorkDir, repositoryRoot, arm, runner.mcpServerInstructionsMode, env)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("configure Agy MCP server: %w", err)
	}
	configPath := filepath.Join(absWorkDir, ".agents", "mcp_config.json")
	configData, err := os.ReadFile(configPath) // #nosec G304 -- path is fixed beneath the resolved benchmark workspace.
	if err != nil {
		removeErr := os.Remove(configPath)
		return "", "", nil, nil, errors.Join(fmt.Errorf("read Agy MCP configuration: %w", err), removeErr)
	}
	return absWorkDir, configPath, configData, env, nil
}

func launchAgyStream(executable, workDir, configPath string, args, env []string) (*agyProcess, context.CancelFunc, error) {
	lifecycleCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(lifecycleCtx, executable, args...) // #nosec G204 -- executable and args are derived from the benchmark's configured harness target.
	cmd.Dir = workDir
	cmd.Env = env
	configureCodexAppServerProcess(cmd)
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("open Agy stdout pipe: %w", err)
	}
	cmd.Stdout = stdoutWriter
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		readerErr := stdoutReader.Close()
		writerErr := stdoutWriter.Close()
		return nil, nil, errors.Join(fmt.Errorf("open Agy stdin: %w", err), readerErr, writerErr)
	}
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr
	err = cmd.Start()
	if err != nil {
		cancel()
		stdinErr := stdin.Close()
		readerErr := stdoutReader.Close()
		writerErr := stdoutWriter.Close()
		return nil, nil, errors.Join(fmt.Errorf("start Agy stream: %w", err), stdinErr, readerErr, writerErr)
	}
	startedAt := time.Now()
	err = stdoutWriter.Close()
	if err != nil {
		stdinErr := stdin.Close()
		killErr := cmd.Process.Kill()
		waitErr := cmd.Wait()
		cancel()
		readerErr := stdoutReader.Close()
		return nil, nil, errors.Join(fmt.Errorf("close parent Agy stdout pipe: %w", err), stdinErr, killErr, waitErr, readerErr)
	}
	events := make(chan agyStreamEvent, 8)
	readErr := make(chan error, 1)
	process := &agyProcess{
		cmd: cmd, startedAt: startedAt, stdin: stdin, stdout: stdoutReader,
		events: events, readErr: readErr, stderr: stderr, configPath: configPath,
	}
	go readAgyStream(lifecycleCtx, stdoutReader, events, readErr)
	return process, cancel, nil
}

func (s *agySession) start(ctx context.Context) error {
	if s.closed {
		return fmt.Errorf("agy session is already closed")
	}
	if s.process != nil {
		return nil
	}
	if s.runner == nil {
		return fmt.Errorf("agy session has no runner")
	}
	if err := s.prepare(ctx); err != nil {
		return err
	}
	absWorkDir, err := filepath.Abs(s.workDir)
	if err != nil {
		return fmt.Errorf("resolve Agy workspace: %w", err)
	}
	args := buildAgyStreamArguments(absWorkDir, s.target)
	process, cancel, err := launchAgyStream(configuredAgyExecutable(s.runner.agyExecutable), s.workDir, s.configPath, args, s.preparedEnvironment)
	if err != nil {
		return err
	}
	s.process = process
	s.cancel = cancel
	return nil
}

func readAgyStream(ctx context.Context, stdout io.Reader, events chan<- agyStreamEvent, readErr chan<- error) {
	defer close(events)
	defer close(readErr)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var event agyStreamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			select {
			case readErr <- fmt.Errorf("decode Agy stream event: %w", err):
			case <-ctx.Done():
			}
			return
		}
		event.Raw = append(json.RawMessage(nil), line...)
		select {
		case events <- event:
		case <-ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case readErr <- fmt.Errorf("read Agy stream: %w", err):
		case <-ctx.Done():
		}
	}
}

func validAgyConversationID(id string) bool {
	return id != "" && id != "." && filepath.Base(id) == id && !strings.ContainsAny(id, `/\\`)
}

func (p *agyProcess) handleEvent(event agyStreamEvent, result *RunResult) (*AgyResponse, bool, error) {
	result.rawEvents = append(result.rawEvents, append(json.RawMessage(nil), event.Raw...))
	switch event.Event {
	case "init":
		if p.initialized || !validAgyConversationID(event.ConversationID) {
			return nil, false, fmt.Errorf("agy emitted a missing, unsafe, or duplicate stream identity")
		}
		p.conversationID = event.ConversationID
		p.initialized = true
	case "step_update":
		if !p.initialized {
			return nil, false, fmt.Errorf("agy emitted a step before stream initialization")
		}
		var update struct {
			ConversationID string `json:"conversation_id"`
		}
		if err := json.Unmarshal(event.StepUpdate, &update); err != nil {
			return nil, false, fmt.Errorf("decode Agy step update: %w", err)
		}
		if update.ConversationID != "" && update.ConversationID != p.conversationID {
			return nil, false, fmt.Errorf("agy conversation identity changed in a step update")
		}
	case "result":
		if !p.initialized || event.Result == nil {
			return nil, false, fmt.Errorf("agy emitted a result before stream initialization or without payload")
		}
		if event.Result.ConversationID != p.conversationID {
			return nil, false, fmt.Errorf("agy conversation identity changed or was missing in the result")
		}
		if event.Result.Status != "SUCCESS" {
			return event.Result, true, fmt.Errorf("agy turn status %q: %s", event.Result.Status, event.Result.Error)
		}
		return event.Result, true, nil
	default:
		return nil, false, fmt.Errorf("agy emitted unsupported stream event %q", event.Event)
	}
	return nil, false, nil
}

func (p *agyProcess) runTurn(ctx context.Context, prompt string, result *RunResult) (*AgyResponse, error) {
	if p == nil || p.closed {
		return nil, fmt.Errorf("agy stream is closed")
	}
	if result == nil {
		return nil, fmt.Errorf("agy turn has no result recorder")
	}
	message := struct {
		Event   string `json:"event"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}{Event: "user"}
	message.Message.Content = prompt
	if err := json.NewEncoder(p.stdin).Encode(message); err != nil {
		return nil, fmt.Errorf("send Agy prompt: %w", err)
	}
	readErr := p.readErr
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for Agy turn: %w", ctx.Err())
		case event, ok := <-p.events:
			if !ok {
				select {
				case err, hasErr := <-readErr:
					if hasErr && err != nil {
						return nil, err
					}
				default:
				}
				return nil, fmt.Errorf("agy stream ended before a turn result")
			}
			response, complete, err := p.handleEvent(event, result)
			if err != nil {
				return response, err
			}
			if complete {
				return response, nil
			}
		case err, ok := <-readErr:
			if !ok {
				readErr = nil
			} else if err != nil {
				return nil, err
			}
		}
	}
}

func cumulativeAgyCounter(current, previous int, name string) (int, error) {
	if current < previous {
		return 0, fmt.Errorf("agy cumulative %s regressed", name)
	}
	return current - previous, nil
}

func agyUsageDelta(current, previous AgyResponse) (AgyResponse, error) {
	delta := current
	if current.NumTurns <= previous.NumTurns {
		return AgyResponse{}, fmt.Errorf("agy cumulative num_turns did not advance from %d to %d", previous.NumTurns, current.NumTurns)
	}
	if current.DurationSeconds < previous.DurationSeconds {
		return AgyResponse{}, fmt.Errorf("agy cumulative duration regressed")
	}
	delta.NumTurns = current.NumTurns - previous.NumTurns
	delta.DurationSeconds = current.DurationSeconds - previous.DurationSeconds
	var err error
	if delta.Usage.InputTokens, err = cumulativeAgyCounter(current.Usage.InputTokens, previous.Usage.InputTokens, "input tokens"); err != nil {
		return AgyResponse{}, err
	}
	if delta.Usage.OutputTokens, err = cumulativeAgyCounter(current.Usage.OutputTokens, previous.Usage.OutputTokens, "output tokens"); err != nil {
		return AgyResponse{}, err
	}
	if delta.Usage.ThinkingTokens, err = cumulativeAgyCounter(current.Usage.ThinkingTokens, previous.Usage.ThinkingTokens, "thinking tokens"); err != nil {
		return AgyResponse{}, err
	}
	if delta.Usage.CacheReadTokens, err = cumulativeAgyCounter(current.Usage.CacheReadTokens, previous.Usage.CacheReadTokens, "cache read tokens"); err != nil {
		return AgyResponse{}, err
	}
	if delta.Usage.TotalTokens, err = cumulativeAgyCounter(current.Usage.TotalTokens, previous.Usage.TotalTokens, "total tokens"); err != nil {
		return AgyResponse{}, err
	}
	return delta, nil
}

func readAgyTranscriptDelta(convID string, offset int64) (string, []byte, int64, error) {
	if !validAgyConversationID(convID) {
		return "", nil, offset, fmt.Errorf("invalid Agy conversation identity %q", convID)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil, offset, fmt.Errorf("resolve Agy transcript home: %w", err)
	}
	candidates := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs", "transcript.jsonl"),
		filepath.Join(home, ".gemini", "antigravity", "brain", convID, ".system_generated", "logs", "transcript.jsonl"),
	}
	var lastErr error
	for _, path := range candidates {
		data, readErr := os.ReadFile(path) // #nosec G304 -- candidate paths are constrained to the fixed Agy transcript directory and validated conversation ID.
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if int64(len(data)) < offset {
			return path, nil, offset, fmt.Errorf("agy cumulative transcript regressed from %d to %d bytes", offset, len(data))
		}
		if int64(len(data)) == offset {
			return path, nil, offset, fmt.Errorf("agy transcript added no records for the completed turn")
		}
		return path, append([]byte(nil), data[offset:]...), int64(len(data)), nil
	}
	return "", nil, offset, fmt.Errorf("read Agy transcript for conversation %q: %w", convID, lastErr)
}

func recordAgyStreamTurn(result *RunResult, cumulative, previous AgyResponse, transcript []byte) error {
	delta, err := agyUsageDelta(cumulative, previous)
	if err != nil {
		return err
	}
	result.Turns = delta.NumTurns
	result.PromptTokens = delta.Usage.InputTokens
	result.CachedPromptTokens = delta.Usage.CacheReadTokens
	uncached := delta.Usage.InputTokens - delta.Usage.CacheReadTokens
	if uncached < 0 {
		uncached = delta.Usage.InputTokens
	}
	result.UncachedPromptTokens = uncached
	result.OutputTokens = delta.Usage.OutputTokens
	result.ReasoningTokens = delta.Usage.ThinkingTokens
	result.agentResponse = strings.TrimSpace(cumulative.Response)
	observed := &RunResult{}
	if err := parseAgyTranscript(transcript, observed); err != nil {
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = err.Error()
		return fmt.Errorf("parse Agy transcript turn: %w", err)
	}
	result.rawEvents = append(result.rawEvents, observed.rawEvents...)
	result.ToolCalls = append(result.ToolCalls, observed.ToolCalls...)
	refreshToolCallCounts(result)
	result.InternalTurns = observed.InternalTurns
	result.InitialLoadTurns = observed.InitialLoadTurns
	result.MCPLoadTurns = observed.MCPLoadTurns
	result.MCPVerified = observed.MCPVerified
	result.toolObservationState = ToolObservationUnknown
	result.toolObservationReason = "Agy stream transcript is not a validated terminal coverage contract"
	return nil
}

func waitAgyShutdown(waitDone, readerDone <-chan struct{}) bool {
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	waited := false
	readerClosed := false
	for !waited || !readerClosed {
		select {
		case <-waitDone:
			waited = true
			waitDone = nil
		case <-readerDone:
			readerClosed = true
			readerDone = nil
		case <-timer.C:
			return false
		}
	}
	return true
}

func signalAgyProcessGroup(p *agyProcess, force bool) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("agy stream has no process to signal")
	}
	if err := signalCodexAppServerProcess(p.cmd.Process.Pid, force); err != nil &&
		!errors.Is(err, syscall.ESRCH) && !errors.Is(err, os.ErrProcessDone) {
		signal := "TERM"
		if force {
			signal = "KILL"
		}
		return fmt.Errorf("send %s to Agy process group: %w", signal, err)
	}
	return nil
}

func (p *agyProcess) terminateAfterTimeout(waitDone, readerDone <-chan struct{}) error {
	var terminateErr error
	if err := signalAgyProcessGroup(p, false); err != nil {
		terminateErr = errors.Join(terminateErr, err)
	}
	if waitAgyShutdown(waitDone, readerDone) {
		return terminateErr
	}
	if err := signalAgyProcessGroup(p, true); err != nil {
		terminateErr = errors.Join(terminateErr, err)
	}
	if waitAgyShutdown(waitDone, readerDone) {
		return terminateErr
	}
	terminateErr = errors.Join(terminateErr, fmt.Errorf("agy stream process or output reader did not stop after process-group termination"))
	if p.stdout != nil {
		if err := p.stdout.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			terminateErr = errors.Join(terminateErr, fmt.Errorf("close Agy stream output: %w", err))
		}
	}
	if !waitAgyShutdown(waitDone, readerDone) {
		terminateErr = errors.Join(terminateErr, fmt.Errorf("agy stream process could not be reaped"))
	}
	return terminateErr
}

func (p *agyProcess) shutdown() error {
	var shutdownErr error
	if p.stdin != nil {
		if err := p.stdin.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			shutdownErr = fmt.Errorf("close Agy stream input: %w", err)
		}
	}
	readerDone := make(chan struct{})
	go func() {
		for range p.events {
		}
		close(readerDone)
	}()
	waitDone := make(chan struct{})
	go func() {
		p.waitErr = p.cmd.Wait()
		close(waitDone)
	}()
	if !waitAgyShutdown(waitDone, readerDone) {
		shutdownErr = errors.Join(shutdownErr, p.terminateAfterTimeout(waitDone, readerDone))
	}
	if p.stdout != nil {
		if err := p.stdout.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("close Agy stream output: %w", err))
		}
	}
	select {
	case <-waitDone:
		if p.waitErr != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("wait for Agy stream: %w (stderr: %s)", p.waitErr, strings.TrimSpace(p.stderr.String())))
		}
		if err := signalAgyProcessGroup(p, true); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("stop residual Agy process-group children: %w", err))
		}
	default:
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("agy stream process could not be reaped"))
	}
	return shutdownErr
}

func (s *agySession) close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	var closeErr error
	if s.process != nil {
		s.process.closed = true
		closeErr = s.process.shutdown()
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.configPath != "" {
		if err := os.Remove(s.configPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			closeErr = errors.Join(closeErr, fmt.Errorf("remove Agy MCP configuration: %w", err))
		}
	}
	return closeErr
}

func (s *agySession) run(ctx context.Context, prompt string, result *RunResult, resumeID string) (string, error) {
	if s == nil || result == nil {
		return "", fmt.Errorf("run Agy turn: session and result are required")
	}
	if s.closed {
		return "", fmt.Errorf("run Agy turn: session is closed")
	}
	if s.failure != nil {
		return "", fmt.Errorf("run Agy turn: persistent Agy session failed: %w", s.failure)
	}
	if s.process == nil {
		if resumeID != "" {
			err := fmt.Errorf("run Agy turn: cannot resume conversation %q without its live process", resumeID)
			return "", s.fail(err)
		}
		if err := s.start(ctx); err != nil {
			return "", s.fail(err)
		}
		result.processStartedAt = s.process.startedAt
	} else if resumeID == "" || resumeID != s.process.conversationID {
		err := fmt.Errorf("run Agy turn: continuation identity %q does not match live conversation %q", resumeID, s.process.conversationID)
		return "", s.fail(err)
	}
	configData, err := os.ReadFile(s.configPath)
	if err != nil || !bytes.Equal(configData, s.configData) {
		if err == nil {
			err = fmt.Errorf("agy MCP configuration changed during the live session")
		} else {
			err = fmt.Errorf("read frozen Agy MCP configuration: %w", err)
		}
		return s.process.conversationID, s.fail(err)
	}
	absWorkDir, err := filepath.Abs(s.workDir)
	if err != nil {
		return s.process.conversationID, s.fail(fmt.Errorf("resolve Agy prompt workspace: %w", err))
	}
	turnPrompt := fmt.Sprintf("Working directory is %s.\n%s", absWorkDir, prompt)
	cumulative, turnErr := s.process.runTurn(ctx, turnPrompt, result)
	if cumulative != nil {
		previous := s.process.lastUsage
		if _, err := agyUsageDelta(*cumulative, previous); err != nil {
			return s.process.conversationID, s.fail(err)
		}
		if err := recordAgyStreamUsage(result, *cumulative, previous); err != nil {
			return s.process.conversationID, s.fail(err)
		}
		s.process.lastUsage = *cumulative
		transcriptPath, transcript, newOffset, transcriptErr := readAgyTranscriptDelta(s.process.conversationID, s.transcriptOffset)
		if transcriptErr != nil {
			result.toolObservationState = ToolObservationPartial
			result.toolObservationReason = transcriptErr.Error()
			return s.process.conversationID, s.fail(transcriptErr)
		}
		if err := recordAgyStreamTurn(result, *cumulative, previous, transcript); err != nil {
			return s.process.conversationID, s.fail(fmt.Errorf("record Agy transcript %s: %w", transcriptPath, err))
		}
		s.transcriptOffset = newOffset
	}
	if turnErr != nil {
		return s.process.conversationID, s.fail(turnErr)
	}
	return s.process.conversationID, nil
}

func recordAgyStreamUsage(result *RunResult, cumulative, previous AgyResponse) error {
	delta, err := agyUsageDelta(cumulative, previous)
	if err != nil {
		return err
	}
	result.Turns = delta.NumTurns
	result.PromptTokens = delta.Usage.InputTokens
	result.CachedPromptTokens = delta.Usage.CacheReadTokens
	uncached := delta.Usage.InputTokens - delta.Usage.CacheReadTokens
	if uncached < 0 {
		uncached = delta.Usage.InputTokens
	}
	result.UncachedPromptTokens = uncached
	result.OutputTokens = delta.Usage.OutputTokens
	result.ReasoningTokens = delta.Usage.ThinkingTokens
	result.agentResponse = strings.TrimSpace(cumulative.Response)
	return nil
}

func (s *agySession) fail(err error) error {
	if err == nil {
		return nil
	}
	s.failure = err
	return errors.Join(err, s.close())
}

func (s *agySession) prepare(ctx context.Context) error {
	if s.configPath != "" {
		return nil
	}
	_, configPath, data, env, err := prepareAgyMCPConfig(ctx, s.runner, s.workDir, s.arm)
	if err != nil {
		return err
	}
	s.configPath = configPath
	s.configData = append([]byte(nil), data...)
	s.preparedEnvironment = env
	return nil
}
