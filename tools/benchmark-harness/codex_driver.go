// codex_driver.go adapts Codex event streams and process runs into benchmark telemetry.
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
	"semedit/internal/gocache"
	"semedit/internal/mcp"
	"strconv"
	"strings"
	"time"
)

const maxCodexStderrBytes = 64 * 1024

const benchmarkAGENTSOverride = `# Benchmark fixture instruction override

This directory is a synthetic benchmark fixture and is the complete benchmark
workspace.

Follow the benchmark user prompt. You may inspect and use any file in this
directory, including documentation. Do not inspect, read, or use paths outside
this directory. Run only checks that the user prompt or files in this directory
require.

Do not modify this AGENTS.override.md file.
`

// CodexEvent represents a line from codex exec --json.
type CodexEvent struct {
	Type     string     `json:"type"`
	ThreadID string     `json:"thread_id,omitempty"`
	Item     *codexItem `json:"item,omitempty"`
	Usage    *struct {
		InputTokens           int `json:"input_tokens"`
		CachedInputTokens     int `json:"cached_input_tokens"`
		OutputTokens          int `json:"output_tokens"`
		ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	} `json:"usage,omitempty"`
}

type codexItem struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type"`
	Command   string          `json:"command,omitempty"`
	Text      string          `json:"text,omitempty"`
	Server    string          `json:"server,omitempty"`
	Tool      string          `json:"tool,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Status    string          `json:"status,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     json.RawMessage `json:"error,omitempty"`
}

type codexToolTracker struct {
	calls             map[string]int
	internalTurns     int
	initialLoadTurns  int
	mcpLoadTurns      int
	mutatingSeen      bool
	firstToolCallAt   time.Time
	eventDecodeErrors int
	observationErrors int
}

func (t *codexToolTracker) observe(res *RunResult, item *codexItem, observedAt time.Time) {
	if item == nil {
		return
	}
	if !isCodexToolItem(item.Type) {
		return
	}
	name := codexToolName(item)
	if name == "" {
		t.observationErrors++
		return
	}
	if item.Status == "in_progress" {
		if _, alreadyRecorded := t.calls[item.ID]; alreadyRecorded {
			return
		}
		t.record(res, item.ID, ToolCall{
			Name:             name,
			Server:           item.Server,
			Arguments:        item.Arguments,
			TransportStatus:  ToolCallStatusUnknown,
			FunctionalStatus: ToolCallStatusUnknown,
		}, observedAt)
		return
	}
	call := codexToolOutcome(name, item.Server, item.Arguments, item.Status, item.Result, item.Error)
	if callIndex, ok := t.calls[item.ID]; ok {
		if len(call.Arguments) == 0 {
			call.Arguments = res.ToolCalls[callIndex].Arguments
		}
		res.ToolCalls[callIndex] = call
		applyMCPStartupMetrics(res, call.MCPStartupMetrics)
		refreshMCPVerified(res)
		return
	}
	t.record(res, item.ID, call, observedAt)
}

func (t *codexToolTracker) record(res *RunResult, id string, call ToolCall, observedAt time.Time) {
	if t.firstToolCallAt.IsZero() {
		t.firstToolCallAt = observedAt
	}
	applyMCPStartupMetrics(res, call.MCPStartupMetrics)
	callIndex := appendToolCall(res, call)
	if id != "" {
		if t.calls == nil {
			t.calls = make(map[string]int)
		}
		t.calls[id] = callIndex
	}
	t.internalTurns++
	if isSemanticTool(call.Name, call.Server) || isMutatingTool(call.Name) {
		t.mutatingSeen = true
		return
	}
	if !t.mutatingSeen {
		t.initialLoadTurns++
		if call.Server != "" || isMCPDiscoveryTool(call.Name) {
			t.mcpLoadTurns++
		}
	}
}

func isCodexToolItem(itemType string) bool {
	switch itemType {
	case "tool_call", "mcp_tool_call", "function_call", "command_execution", "file_change":
		return true
	default:
		return false
	}
}

func codexToolName(item *codexItem) string {
	if item.Type == "file_change" {
		return "file_change"
	}
	for _, name := range []string{item.Tool, item.Command, item.Text} {
		if clean := strings.TrimSpace(name); clean != "" {
			return clean
		}
	}
	return ""
}

func codexToolOutcome(name, server string, arguments json.RawMessage, status string, result, toolErr json.RawMessage) ToolCall {
	call := ToolCall{
		Name:              name,
		Server:            server,
		Arguments:         arguments,
		TransportStatus:   ToolCallStatusUnknown,
		FunctionalStatus:  ToolCallStatusUnknown,
		MCPMetrics:        codexMCPMetrics(result),
		MCPStartupMetrics: codexMCPStartupMetrics(result),
	}
	hasResult := hasJSONValue(result)
	hasError := hasJSONValue(toolErr)

	switch strings.ToLower(status) {
	case "completed":
		if hasError {
			call.TransportStatus = ToolCallStatusFailed
			call.Failure = jsonToolFailure(toolErr)
			return call
		}
		call.TransportStatus = ToolCallStatusSucceeded
		call.FunctionalStatus = ToolCallStatusSucceeded
	case "failed":
		if hasResult {
			call.TransportStatus = ToolCallStatusSucceeded
			call.FunctionalStatus = ToolCallStatusFailed
			call.Failure = jsonToolFailure(result)
			return call
		}
		if hasError {
			call.TransportStatus = ToolCallStatusFailed
			call.Failure = jsonToolFailure(toolErr)
			return call
		}
		call.FunctionalStatus = ToolCallStatusFailed
	default:
		if hasError {
			call.TransportStatus = ToolCallStatusFailed
			call.Failure = jsonToolFailure(toolErr)
		}
	}
	return call
}

func codexMCPMetrics(result json.RawMessage) *MCPMetrics {
	if !hasJSONValue(result) {
		return nil
	}
	var envelope struct {
		StructuredContent struct {
			Metrics *MCPMetrics `json:"metrics"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		return nil
	}
	return envelope.StructuredContent.Metrics
}

func codexMCPStartupMetrics(result json.RawMessage) *MCPStartupMetrics {
	if !hasJSONValue(result) {
		return nil
	}
	var envelope struct {
		StructuredContent struct {
			SessionMetrics *MCPStartupMetrics `json:"session_metrics"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		return nil
	}
	return envelope.StructuredContent.SessionMetrics
}

func appendToolCall(res *RunResult, call ToolCall) int {
	res.ToolCalls = append(res.ToolCalls, call)
	res.ToolsUsed = append(res.ToolsUsed, call.Name)
	return len(res.ToolCalls) - 1
}

func applyMCPStartupMetrics(res *RunResult, metrics *MCPStartupMetrics) {
	if metrics == nil {
		return
	}
	if res.MCPServerStartToInitialize == nil {
		value := time.Duration(metrics.ServerStartToInitializeMS) * time.Millisecond
		res.MCPServerStartToInitialize = &value
	}
	if res.MCPInitializeToFirstSemanticCall == nil {
		value := time.Duration(metrics.InitializeToFirstSemanticCallMS) * time.Millisecond
		res.MCPInitializeToFirstSemanticCall = &value
	}
}

func refreshMCPVerified(res *RunResult) {
	res.MCPVerified = false
	for _, call := range res.ToolCalls {
		if isSemanticTool(call.Name, call.Server) && call.TransportStatus == ToolCallStatusSucceeded {
			res.MCPVerified = true
			return
		}
	}
}

func isSemanticTool(name, server string) bool {
	cleanServer := strings.ToLower(strings.Trim(server, "\""))
	return cleanServer == "semedit" || strings.HasPrefix(semanticToolBaseName(name), "semantic_")
}

func hasJSONValue(raw json.RawMessage) bool {
	value := strings.TrimSpace(string(raw))
	return value != "" && value != "null"
}

func jsonToolFailure(raw json.RawMessage) string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return compactToolFailure(string(raw))
	}
	return compactToolFailure(toolFailureText(value))
}

func toolFailureText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		for _, item := range typed {
			if text := toolFailureText(item); text != "" {
				return text
			}
		}
	case map[string]any:
		for _, key := range []string{"error", "message", "text", "content", "result"} {
			if item, ok := typed[key]; ok {
				if text := toolFailureText(item); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func compactToolFailure(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

func (r *Runner) runCodex(ctx context.Context, workDir string, target Target, prompt string, res *RunResult, resumeID string) (string, error) {
	args := []string{"exec"}
	if resumeID == "" {
		args = append(args, "--json", "-s", "workspace-write", "-C", workDir)
	} else {
		args = append(args, "resume", "--json", resumeID)
	}
	if target.Model != "" {
		args = append(args, "--model", target.Model)
	}
	if target.Effort != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", target.Effort))
	}
	if override, err := r.codexMCPOverride(res.Arm, workDir); err != nil {
		return "", err
	} else if override != "" {
		args = append(args, "-c", override)
	}
	if res.Arm == ArmSemedit {
		args = append(args, "-c", r.codexMCPBinaryOverride())
		args = append(args, "-c", codexMCPEnabledOverride())
		args = append(args, "-c", codexMCPGoEnvironmentOverride())
	}
	args = append(args, prompt)

	// #nosec G204 -- external driver invocation controlled by benchmark harness
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Dir = workDir
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return "", fmt.Errorf("prepare Codex Go environment: %w", err)
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("open codex stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start codex: %w", err)
	}
	processStartedAt := time.Now()
	res.processStartedAt = processStartedAt

	stream := readCodexEventStream(stdout, res, resumeID)
	waitErr := cmd.Wait()
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		res.CodexExitCode = &code
	}
	res.CodexStderr = boundedDiagnostic(stderr.Bytes(), maxCodexStderrBytes)
	applyCodexStreamMetrics(res, stream, processStartedAt)
	providerErr := codexStreamFailure(ctx, stream, waitErr, cmd.ProcessState, res.CodexStderr)
	setCodexObservationState(res, providerErr, stream.sawTerminalTurn)
	if providerErr == nil && stream.sawTerminalTurn && stream.tracker.observationErrors > 0 {
		res.toolObservationState = ToolObservationPartial
		res.toolObservationReason = fmt.Sprintf("Codex observed %d malformed tool call records", stream.tracker.observationErrors)
	}
	if providerErr != nil {
		return stream.threadID, providerErr
	}
	if stream.validEvents == 0 {
		return stream.threadID, fmt.Errorf("codex emitted no valid task events")
	}
	if stream.threadID == "" {
		return "", fmt.Errorf("codex did not emit a thread ID")
	}
	return stream.threadID, nil
}

type codexEventStream struct {
	turns           int
	sawTerminalTurn bool
	validEvents     int
	firstEventAt    time.Time
	threadID        string
	scanErr         error
	tracker         codexToolTracker
}

func readCodexEventStream(stdout io.Reader, result *RunResult, resumeID string) codexEventStream {
	stream := codexEventStream{threadID: resumeID}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		consumeCodexEvent(&stream, result, append([]byte(nil), scanner.Bytes()...))
	}
	stream.scanErr = scanner.Err()
	return stream
}

func consumeCodexEvent(stream *codexEventStream, result *RunResult, raw []byte) {
	var event CodexEvent
	if err := json.Unmarshal(raw, &event); err != nil || event.Type == "" {
		stream.tracker.eventDecodeErrors++
		encoded, _ := json.Marshal(string(raw))
		result.rawEvents = append(result.rawEvents, json.RawMessage(encoded))
		return
	}
	stream.validEvents++
	result.rawEvents = append(result.rawEvents, json.RawMessage(raw))
	observedAt := time.Now()
	if stream.firstEventAt.IsZero() {
		stream.firstEventAt = observedAt
	}
	if event.Type == "turn.started" {
		stream.turns++
		stream.sawTerminalTurn = false
	}
	if event.Type == "turn.completed" {
		stream.sawTerminalTurn = true
	}
	if event.Type == "thread.started" && event.ThreadID != "" {
		stream.threadID = event.ThreadID
	}
	appendCodexAgentResponse(result, event.Item)
	stream.tracker.observe(result, event.Item, observedAt)
	if event.Usage != nil {
		result.PromptTokens = event.Usage.InputTokens
		result.CachedPromptTokens = event.Usage.CachedInputTokens
		result.UncachedPromptTokens = event.Usage.InputTokens - event.Usage.CachedInputTokens
		result.OutputTokens = event.Usage.OutputTokens
		result.ReasoningTokens = event.Usage.ReasoningOutputTokens
	}
}

func applyCodexStreamMetrics(result *RunResult, stream codexEventStream, processStartedAt time.Time) {
	result.Turns = stream.turns
	result.InternalTurns = stream.tracker.internalTurns
	result.InitialLoadTurns = stream.tracker.initialLoadTurns
	result.MCPLoadTurns = stream.tracker.mcpLoadTurns
	result.ToolCount = len(result.ToolCalls)
	if !stream.firstEventAt.IsZero() {
		value := stream.firstEventAt.Sub(processStartedAt)
		result.ProcessStartToFirstEvent = &value
	}
	if !stream.firstEventAt.IsZero() && !stream.tracker.firstToolCallAt.IsZero() {
		value := stream.tracker.firstToolCallAt.Sub(stream.firstEventAt)
		result.FirstEventToFirstToolCall = &value
	}
}

func codexStreamFailure(ctx context.Context, stream codexEventStream, waitErr error, state *os.ProcessState, stderr string) error {
	var providerErr error
	if stream.scanErr != nil {
		providerErr = fmt.Errorf("scan codex output: %w", stream.scanErr)
	}
	if stream.tracker.eventDecodeErrors > 0 {
		providerErr = errors.Join(providerErr, fmt.Errorf("decode %d malformed Codex event records", stream.tracker.eventDecodeErrors))
	}
	if waitErr != nil {
		providerErr = errors.Join(providerErr, codexWaitFailure(ctx, waitErr, state != nil && state.ExitCode() == -1, stderr))
	}
	return providerErr
}

func appendCodexAgentResponse(res *RunResult, item *codexItem) {
	if res == nil || item == nil || item.Type != "agent_message" {
		return
	}
	text := strings.TrimSpace(item.Text)
	if text == "" {
		return
	}
	if res.agentResponse != "" {
		res.agentResponse += "\n"
	}
	res.agentResponse += text
}

func (r *Runner) codexMCPOverride(arm ArmType, workDir string) (string, error) {
	switch arm {
	case ArmBaseline:
		return "mcp_servers.semedit.enabled=false", nil
	case ArmSemedit:
		return r.codexMCPServerInstructionsOverride(workDir)
	default:
		return "", fmt.Errorf("unsupported Codex benchmark arm %q", arm)
	}
}

func fixtureGoEnvironment(ctx context.Context, workDir string) ([]string, error) {
	return gocache.Environment(gocache.WithBaseDir(ctx, filepath.Join(workDir, ".scratch", "go")), workDir)
}

func runSemeditVerifyWarmup(ctx context.Context, workDir, file, binaryPath string) (time.Duration, error) {
	if !filepath.IsLocal(filepath.FromSlash(file)) {
		return 0, fmt.Errorf("prewarm verify file %q must be fixture-local", file)
	}
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return 0, fmt.Errorf("prepare prewarm Go environment: %w", err)
	}
	for _, subdir := range []string{"build", "mod"} {
		cacheDir := filepath.Join(workDir, ".scratch", "go", subdir)
		entries, readErr := os.ReadDir(cacheDir)
		if readErr != nil {
			return 0, fmt.Errorf("read prewarm %s cache: %w", subdir, readErr)
		}
		if len(entries) != 0 {
			return 0, fmt.Errorf("prewarm %s cache is not empty in %s", subdir, cacheDir)
		}
	}
	// #nosec G204 -- binaryPath is the repository-owned semedit-next binary selected by the runner.
	cmd := exec.CommandContext(ctx, binaryPath, "verify", "--file", filepath.FromSlash(file), "--language", "go", "--check-only")
	cmd.Dir = workDir
	cmd.Env = env
	started := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(started)
	if err != nil {
		return duration, fmt.Errorf("prewarm semedit verify: %w: %s", err, boundedDiagnostic(output, 2048))
	}
	return duration, nil
}

func canonicalFixtureRoot(workDir string) (string, error) {
	absoluteRoot, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve fixture root %q: %w", workDir, err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("canonicalize fixture root %q: %w", absoluteRoot, err)
	}
	return filepath.Clean(canonicalRoot), nil
}

func codexMCPGoEnvironmentOverride() string {
	return `mcp_servers.semedit.env_vars=["GOENV","GOCACHE","GOMODCACHE","GOTMPDIR","GOBIN","GOPATH","GOFLAGS","GOWORK"]`
}

func codexMCPEnabledOverride() string {
	return "mcp_servers.semedit.enabled=true"
}

func (r *Runner) codexMCPBinaryOverride() string {
	repositoryRoot := filepath.Dir(filepath.Dir(r.baseScratchDir))
	return "mcp_servers.semedit.command=" + strconv.Quote(filepath.Join(repositoryRoot, "bin", "semedit-next"))
}

func (r *Runner) codexMCPServerInstructionsOverride(workDir string) (string, error) {
	root, err := canonicalFixtureRoot(workDir)
	if err != nil {
		return "", err
	}
	args := []string{"mcp", "--profile", "full", "--expected-root", root, "--metrics-file", benchmarkMCPMetricsPath(workDir)}
	switch r.mcpServerInstructionsMode {
	case MCPServerInstructionsNone:
	case MCPServerInstructionsDescriptive:
		args = append(args, "--instructions", mcp.DescriptiveInstructions)
	case MCPServerInstructionsPrescriptive:
		args = append(args, "--instructions", mcp.PrescriptiveInstructions)
	default:
		return "", fmt.Errorf("unsupported MCP server instruction mode %q", r.mcpServerInstructionsMode)
	}
	quoted := make([]string, len(args))
	for index, arg := range args {
		quoted[index] = strconv.Quote(arg)
	}
	return "mcp_servers.semedit.args=[" + strings.Join(quoted, ",") + "]", nil
}

func setCodexObservationState(result *RunResult, providerErr error, terminalTurn bool) {
	switch {
	case providerErr != nil:
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = providerErr.Error()
	case terminalTurn:
		result.toolObservationState = ToolObservationComplete
	default:
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = "Codex stream ended without a terminal turn.completed event"
	}
}

func codexTerminationOrigin(ctx context.Context, signaled bool) string {
	if !signaled {
		return ""
	}
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return "harness_timeout"
	case context.Canceled:
		return "harness_cancellation"
	default:
		return "outside_harness"
	}
}

func codexWaitFailure(ctx context.Context, waitErr error, signaled bool, stderr string) error {
	origin := codexTerminationOrigin(ctx, signaled)
	terminationDetails := ""
	if origin != "" {
		terminationDetails = fmt.Sprintf(" [termination_origin=%s]", origin)
	}
	if diagnostic := strings.TrimSpace(stderr); diagnostic != "" {
		return fmt.Errorf("run codex: %w%s: %s", waitErr, terminationDetails, diagnostic)
	}
	return fmt.Errorf("run codex: %w%s", waitErr, terminationDetails)
}
