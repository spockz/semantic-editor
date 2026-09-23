// codex_driver.go adapts Codex event streams and process runs into benchmark telemetry.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	calls            map[string]int
	internalTurns    int
	initialLoadTurns int
	mcpLoadTurns     int
	mutatingSeen     bool
	firstToolCallAt  time.Time
}

func (t *codexToolTracker) observe(res *RunResult, item *codexItem, observedAt time.Time) {
	if item == nil || !isCodexToolItem(item.Type) {
		return
	}

	name := codexToolName(item)
	if name == "" {
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
	case "tool_call", "mcp_tool_call", "function_call", "command_execution":
		return true
	default:
		return false
	}
}

func codexToolName(item *codexItem) string {
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
	cleanName := strings.ToLower(strings.Trim(name, "\""))
	cleanServer := strings.ToLower(strings.Trim(server, "\""))
	return cleanServer == "semedit" || strings.HasPrefix(cleanName, "semantic_") || strings.HasPrefix(cleanName, "semedit/") || strings.HasPrefix(cleanName, "mcp__semedit__")
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
	if override, err := r.codexMCPOverride(res.Arm); err != nil {
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
	processStartedAt := time.Now()
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start codex: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	turns := 0
	tracker := codexToolTracker{}
	var firstEventAt time.Time
	threadID := resumeID
	for scanner.Scan() {
		var ev CodexEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		observedAt := time.Now()
		if firstEventAt.IsZero() {
			firstEventAt = observedAt
		}
		if ev.Type == "turn.started" {
			turns++
		}
		if ev.Type == "thread.started" && ev.ThreadID != "" {
			threadID = ev.ThreadID
		}
		appendCodexAgentResponse(res, ev.Item)
		tracker.observe(res, ev.Item, observedAt)
		if ev.Usage != nil {
			res.PromptTokens = ev.Usage.InputTokens
			res.CachedPromptTokens = ev.Usage.CachedInputTokens
			res.UncachedPromptTokens = ev.Usage.InputTokens - ev.Usage.CachedInputTokens
			res.OutputTokens = ev.Usage.OutputTokens
			res.ReasoningTokens = ev.Usage.ReasoningOutputTokens
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		res.CodexExitCode = &code
	}
	res.CodexStderr = boundedDiagnostic(stderr.Bytes(), maxCodexStderrBytes)
	res.Turns = 1
	if turns > 0 {
		res.Turns = turns
	}
	res.InternalTurns = tracker.internalTurns
	res.InitialLoadTurns = tracker.initialLoadTurns
	res.MCPLoadTurns = tracker.mcpLoadTurns
	res.ToolCount = len(res.ToolCalls)
	if !firstEventAt.IsZero() {
		value := firstEventAt.Sub(processStartedAt)
		res.ProcessStartToFirstEvent = &value
	}
	if !firstEventAt.IsZero() && !tracker.firstToolCallAt.IsZero() {
		value := tracker.firstToolCallAt.Sub(firstEventAt)
		res.FirstEventToFirstToolCall = &value
	}
	if scanErr != nil {
		return threadID, fmt.Errorf("scan codex output: %w", scanErr)
	}
	if waitErr != nil {
		if diagnostic := strings.TrimSpace(res.CodexStderr); diagnostic != "" {
			return threadID, fmt.Errorf("run codex: %w: %s", waitErr, diagnostic)
		}
		return threadID, fmt.Errorf("run codex: %w", waitErr)
	}
	if threadID == "" {
		return "", fmt.Errorf("codex did not emit a thread ID")
	}
	return threadID, nil
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

func (r *Runner) codexMCPOverride(arm ArmType) (string, error) {
	switch arm {
	case ArmBaseline:
		return "mcp_servers.semedit.enabled=false", nil
	case ArmSemedit:
		return r.codexMCPServerInstructionsOverride()
	default:
		return "", fmt.Errorf("unsupported Codex benchmark arm %q", arm)
	}
}

func fixtureGoEnvironment(ctx context.Context, workDir string) ([]string, error) {
	return gocache.Environment(gocache.WithBaseDir(ctx, filepath.Join(workDir, ".scratch", "go")), workDir)
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

func (r *Runner) codexMCPServerInstructionsOverride() (string, error) {
	switch r.mcpServerInstructionsMode {
	case MCPServerInstructionsNone:
		return `mcp_servers.semedit.args=["mcp","--profile","full"]`, nil
	case MCPServerInstructionsDescriptive, MCPServerInstructionsPrescriptive:
		instructions := mcp.DescriptiveInstructions
		if r.mcpServerInstructionsMode == MCPServerInstructionsPrescriptive {
			instructions = mcp.PrescriptiveInstructions
		}
		args := []string{"mcp", "--profile", "full", "--instructions", instructions}
		quoted := make([]string, len(args))
		for index, arg := range args {
			quoted[index] = strconv.Quote(arg)
		}
		return "mcp_servers.semedit.args=[" + strings.Join(quoted, ",") + "]", nil
	default:
		return "", fmt.Errorf("unsupported MCP server instruction mode %q", r.mcpServerInstructionsMode)
	}
}
