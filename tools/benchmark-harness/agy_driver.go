// agy_driver.go adapts Agy process responses and transcripts into benchmark telemetry.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
	"strings"
	"time"
)

// AgyResponse represents the JSON output from agy --output-format json -p.
type AgyResponse struct {
	ConversationID  string  `json:"conversation_id"`
	Status          string  `json:"status"`
	Response        string  `json:"response"`
	Error           string  `json:"error,omitempty"`
	DurationSeconds float64 `json:"duration_seconds"`
	NumTurns        int     `json:"num_turns"`
	Usage           struct {
		InputTokens     int `json:"input_tokens"`
		OutputTokens    int `json:"output_tokens"`
		ThinkingTokens  int `json:"thinking_tokens"`
		CacheReadTokens int `json:"cache_read_tokens"`
		TotalTokens     int `json:"total_tokens"`
	} `json:"usage"`
}

func agyToolFunctionalStatus(status string) ToolCallStatus {
	switch strings.ToLower(status) {
	case "done", "success", "completed":
		return ToolCallStatusSucceeded
	case "failed", "error", "cancelled", "canceled":
		return ToolCallStatusFailed
	default:
		return ToolCallStatusUnknown
	}
}

func recordAgyResponseMetrics(res *RunResult, resp AgyResponse) error {
	res.Turns = resp.NumTurns
	res.PromptTokens = resp.Usage.InputTokens
	res.CachedPromptTokens = resp.Usage.CacheReadTokens
	uncached := resp.Usage.InputTokens - resp.Usage.CacheReadTokens
	if uncached < 0 {
		uncached = resp.Usage.InputTokens
	}
	res.UncachedPromptTokens = uncached
	res.OutputTokens = resp.Usage.OutputTokens
	res.ReasoningTokens = resp.Usage.ThinkingTokens
	res.agentResponse = strings.TrimSpace(resp.Response)
	if resp.ConversationID == "" {
		return fmt.Errorf("agy response did not include a conversation ID")
	}
	return extractAgyTools(resp.ConversationID, res)
}

func (r *Runner) runAgy(ctx context.Context, workDir string, target Target, prompt string, res *RunResult, resumeID string) (string, error) {
	agyBin := configuredAgyExecutable(r.agyExecutable)

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve Agy workspace: %w", err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Dir(filepath.Dir(r.baseScratchDir)))
	if err != nil {
		return "", fmt.Errorf("resolve repository root for Agy MCP: %w", err)
	}
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return "", fmt.Errorf("prepare Agy Go environment: %w", err)
	}
	res.MCPServerInstructions = r.mcpServerInstructionsMode
	if err := writeAgyMCPConfig(absWorkDir, repositoryRoot, res.Arm, res.MCPServerInstructions, env); err != nil {
		return "", fmt.Errorf("configure Agy MCP server: %w", err)
	}
	configPath := filepath.Join(absWorkDir, ".agents", "mcp_config.json")
	defer func() {
		_ = os.Remove(configPath)
	}()

	args := buildAgyArguments(absWorkDir, target, prompt, resumeID)

	// #nosec G204 -- agyBin is one of two fixed executable names; args are passed directly without shell interpretation.
	cmd := exec.CommandContext(ctx, agyBin, args...)
	cmd.Dir = workDir
	cmd.Env = env
	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start Agy: %w", err)
	}
	res.processStartedAt = time.Now()
	runErr := cmd.Wait()
	out := output.Bytes()
	var resp AgyResponse
	decodeErr := json.Unmarshal(out, &resp)
	if decodeErr == nil {
		if recordErr := recordAgyResponseMetrics(res, resp); recordErr != nil {
			decodeErr = recordErr
		}
	}
	var responseErr error
	if runErr != nil {
		responseErr = fmt.Errorf("agy exec error: %w (output: %s)", runErr, string(out))
	}
	if decodeErr != nil {
		responseErr = errors.Join(responseErr, fmt.Errorf("decode Agy response and transcript: %w (raw: %s)", decodeErr, string(out)))
	}
	if decodeErr == nil && resp.Status != "SUCCESS" {
		responseErr = errors.Join(responseErr, fmt.Errorf("agy error status (%s): %s", resp.Status, resp.Error))
	}
	if resp.ConversationID == "" && decodeErr == nil {
		responseErr = errors.Join(responseErr, fmt.Errorf("agy did not emit a conversation ID"))
	}
	if responseErr != nil {
		res.toolObservationState = ToolObservationPartial
		res.toolObservationReason = responseErr.Error()
	} else {
		res.toolObservationState = ToolObservationUnknown
		res.toolObservationReason = "Agy adapter has no validated terminal transcript marker"
	}
	if responseErr != nil {
		return resp.ConversationID, responseErr
	}
	return resp.ConversationID, nil
}

func configuredAgyExecutable(configured string) string {
	if configured != "" {
		return configured
	}
	const defaultExecutable = "/Users/alessandro/.local/bin/agy"
	if _, err := os.Stat(defaultExecutable); err == nil {
		return defaultExecutable
	}
	return "agy"
}

func buildAgyArguments(workDir string, target Target, prompt, resumeID string) []string {
	args := []string{"--output-format", "json", "--dangerously-skip-permissions", "--add-dir", workDir}
	if resumeID != "" {
		args = append(args, "--conversation", resumeID)
	}
	if target.Model != "" {
		args = append(args, "--model", target.Model)
	}
	if target.Effort != "" {
		args = append(args, "--effort", target.Effort)
	}
	fullPrompt := fmt.Sprintf("Working directory is %s.\n%s", workDir, prompt)
	return append(args, "-p", fullPrompt)
}

type agyTranscriptStep struct {
	Type      string                  `json:"type"`
	Status    string                  `json:"status"`
	Content   string                  `json:"content"`
	ToolCalls []agyTranscriptToolCall `json:"tool_calls"`
}

type agyTranscriptToolCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

func isMutatingTool(name string) bool {
	clean := strings.ToLower(strings.Trim(name, "\""))
	if strings.Contains(clean, "write") || strings.Contains(clean, "replace") ||
		strings.Contains(clean, "edit") || strings.Contains(clean, "patch") ||
		strings.Contains(clean, "insert") || strings.Contains(clean, "delete") ||
		strings.Contains(clean, "rename") || strings.Contains(clean, "semantic_") ||
		clean == "file_change" {
		return true
	}
	return false
}

func isMCPDiscoveryTool(name string) bool {
	clean := strings.ToLower(strings.Trim(name, "\""))
	return strings.Contains(clean, "mcp") ||
		strings.Contains(clean, "list_resources") ||
		strings.Contains(clean, "read_resource") ||
		strings.Contains(clean, "get_server_capabilities") ||
		strings.Contains(clean, "describe")
}

func extractAgyTools(convID string, res *RunResult) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve Agy transcript home: %w", err)
	}
	candidates := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs", "transcript.jsonl"),
		filepath.Join(home, ".gemini", "antigravity", "brain", convID, ".system_generated", "logs", "transcript.jsonl"),
	}
	var lastErr error
	for _, transcriptPath := range candidates {
		// #nosec G304 -- transcript paths use Agy’s fixed home log layout and provider-issued conversation ID.
		data, err := os.ReadFile(transcriptPath)
		if err != nil {
			lastErr = err
			continue
		}
		if err := parseAgyTranscript(data, res); err != nil {
			return fmt.Errorf("parse Agy transcript %s: %w", transcriptPath, err)
		}
		return nil
	}
	return fmt.Errorf("read Agy transcript for conversation %q: %w", convID, lastErr)
}

func parseAgyTranscript(data []byte, res *RunResult) error {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	state := agyTranscriptState{}
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		state.lineNumber++
		if err := processAgyTranscriptLine(line, res, &state); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan Agy transcript: %w", err)
	}
	if state.internalTurns > 0 {
		res.InternalTurns = state.internalTurns
	}
	res.InitialLoadTurns = state.initialLoadTurns
	res.MCPLoadTurns = state.mcpLoadTurns
	res.ToolCount = len(res.ToolCalls)
	refreshMCPVerified(res)
	return nil
}

type agyTranscriptState struct {
	internalTurns    int
	initialLoadTurns int
	mcpLoadTurns     int
	mutatingSeen     bool
	pending          []int
	lineNumber       int
}

func processAgyTranscriptLine(line []byte, result *RunResult, state *agyTranscriptState) error {
	var step agyTranscriptStep
	if err := json.Unmarshal(line, &step); err != nil {
		return fmt.Errorf("decode transcript line %d: %w", state.lineNumber, err)
	}
	result.rawEvents = append(result.rawEvents, json.RawMessage(line))
	if step.Type == "PLANNER_RESPONSE" {
		state.internalTurns++
	}
	if err := appendAgyTranscriptCalls(step.ToolCalls, result, state); err != nil {
		return err
	}
	if step.Type == "GENERIC" {
		completeAgyTranscriptCalls(step, result, state)
	}
	return nil
}

func appendAgyTranscriptCalls(toolCalls []agyTranscriptToolCall, result *RunResult, state *agyTranscriptState) error {
	for _, toolCall := range toolCalls {
		name, server, isMCPCall := resolveAgyToolCall(toolCall)
		arguments, err := json.Marshal(toolCall.Args)
		if err != nil {
			return fmt.Errorf("encode arguments on transcript line %d: %w", state.lineNumber, err)
		}
		index := appendToolCall(result, ToolCall{Name: name, Server: server, Arguments: arguments, TransportStatus: ToolCallStatusUnknown, FunctionalStatus: ToolCallStatusUnknown})
		state.pending = append(state.pending, index)
		observeAgyToolCall(name, server, isMCPCall, state)
	}
	return nil
}

func resolveAgyToolCall(toolCall agyTranscriptToolCall) (string, string, bool) {
	name := strings.Trim(strings.TrimSpace(toolCall.Name), "\"")
	server := ""
	isMCPCall := name == "call_mcp_tool"
	argsMap, ok := toolCall.Args.(map[string]any)
	if !isMCPCall || !ok {
		return name, server, isMCPCall
	}
	if toolName, ok := argsMap["ToolName"].(string); ok {
		name = strings.Trim(strings.TrimSpace(toolName), "\"")
	}
	if serverName, ok := argsMap["ServerName"].(string); ok {
		server = strings.Trim(strings.TrimSpace(serverName), "\"")
	}
	return strings.Trim(name, "\""), server, isMCPCall
}

func observeAgyToolCall(name, server string, isMCPCall bool, state *agyTranscriptState) {
	if isSemanticTool(name, server) || isMutatingTool(name) {
		state.mutatingSeen = true
		return
	}
	if state.mutatingSeen {
		return
	}
	state.initialLoadTurns++
	if isMCPCall || isMCPDiscoveryTool(name) {
		state.mcpLoadTurns++
	}
}

func completeAgyTranscriptCalls(step agyTranscriptStep, result *RunResult, state *agyTranscriptState) {
	if len(state.pending) == 0 {
		return
	}
	for _, index := range state.pending {
		result.ToolCalls[index].TransportStatus = ToolCallStatusSucceeded
		result.ToolCalls[index].FunctionalStatus = agyToolFunctionalStatus(step.Status)
		if result.ToolCalls[index].FunctionalStatus == ToolCallStatusFailed {
			result.ToolCalls[index].Failure = compactToolFailure(step.Content)
		}
	}
	state.pending = nil
	refreshMCPVerified(result)
}

func writeAgyMCPConfig(workDir, repositoryRoot string, arm ArmType, mode MCPServerInstructionMode, env []string) error {
	configDir := filepath.Join(workDir, ".agents")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return fmt.Errorf("create Agy MCP config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "mcp_config.json")
	if _, err := os.Lstat(configPath); err == nil {
		return fmt.Errorf("agy fixture already contains %s", configPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect Agy MCP config path: %w", err)
	}

	expectedRoot, err := canonicalFixtureRoot(workDir)
	if err != nil {
		return fmt.Errorf("resolve Agy MCP expected root: %w", err)
	}
	args := []string{"mcp", "--profile", "full", "--expected-root", expectedRoot, "--metrics-file", benchmarkMCPMetricsPath(workDir)}
	disabled := arm == ArmBaseline
	if !disabled {
		switch mode {
		case MCPServerInstructionsNone:
		case MCPServerInstructionsDescriptive:
			args = append(args, "--instructions", mcp.DescriptiveInstructions)
		case MCPServerInstructionsPrescriptive:
			args = append(args, "--instructions", mcp.PrescriptiveInstructions)
		default:
			return fmt.Errorf("unsupported MCP server instruction mode %q", mode)
		}
	}

	server := map[string]any{
		"command":  filepath.Join(repositoryRoot, "bin", "semedit-next"),
		"args":     args,
		"cwd":      workDir,
		"env":      environmentMap(env),
		"disabled": disabled,
	}
	data, err := json.MarshalIndent(map[string]any{
		"mcpServers": map[string]any{"semedit": server},
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Agy MCP config: %w", err)
	}
	if err := pipeline.WriteAtomic(configPath, data); err != nil {
		return fmt.Errorf("write Agy MCP config: %w", err)
	}
	return nil
}
