// agy_driver.go adapts Agy process responses and transcripts into benchmark telemetry.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
	"strings"
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

func recordAgyResponseMetrics(res *RunResult, resp AgyResponse) {
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
	if resp.ConversationID != "" {
		extractAgyTools(resp.ConversationID, res)
	}
}

func (r *Runner) runAgy(ctx context.Context, workDir string, target Target, prompt string, res *RunResult, resumeID string) (string, error) {
	agyBin := "/Users/alessandro/.local/bin/agy"
	if _, err := os.Stat(agyBin); err != nil {
		agyBin = "agy"
	}

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

	args := []string{"--output-format", "json", "--dangerously-skip-permissions", "--add-dir", absWorkDir}
	if resumeID != "" {
		args = append(args, "--conversation", resumeID)
	}
	if target.Model != "" {
		args = append(args, "--model", target.Model)
	}
	if target.Effort != "" {
		args = append(args, "--effort", target.Effort)
	}

	fullPrompt := fmt.Sprintf("Working directory is %s.\n%s", absWorkDir, prompt)
	args = append(args, "-p", fullPrompt)

	cmd := exec.CommandContext(ctx, agyBin, args...)
	cmd.Dir = workDir
	cmd.Env = env

	out, runErr := cmd.Output()
	var resp AgyResponse
	decodeErr := json.Unmarshal(out, &resp)
	if decodeErr == nil {
		recordAgyResponseMetrics(res, resp)
	}
	if runErr != nil {
		return "", fmt.Errorf("agy exec error: %w (output: %s)", runErr, string(out))
	}
	if decodeErr != nil {
		return "", fmt.Errorf("unmarshal agy json response: %w (raw: %s)", decodeErr, string(out))
	}
	if resp.Status != "SUCCESS" && resp.Error != "" {
		return "", fmt.Errorf("agy error status (%s): %s", resp.Status, resp.Error)
	}
	if resp.ConversationID == "" {
		return "", fmt.Errorf("agy did not emit a conversation ID")
	}
	return resp.ConversationID, nil
}

type agyTranscriptStep struct {
	Type      string `json:"type"`
	Status    string `json:"status"`
	Content   string `json:"content"`
	ToolCalls []struct {
		Name string `json:"name"`
		Args any    `json:"args"`
	} `json:"tool_calls"`
}

func isMutatingTool(name string) bool {
	clean := strings.ToLower(strings.Trim(name, "\""))
	if strings.Contains(clean, "write") || strings.Contains(clean, "replace") ||
		strings.Contains(clean, "edit") || strings.Contains(clean, "patch") ||
		strings.Contains(clean, "insert") || strings.Contains(clean, "delete") ||
		strings.Contains(clean, "rename") || strings.Contains(clean, "semantic_") {
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

func extractAgyTools(convID string, res *RunResult) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	candidates := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs", "transcript.jsonl"),
		filepath.Join(home, ".gemini", "antigravity", "brain", convID, ".system_generated", "logs", "transcript.jsonl"),
	}

	for _, transcriptPath := range candidates {
		// #nosec G304 -- reading agy transcript within user home directory
		data, err := os.ReadFile(transcriptPath)
		if err != nil {
			continue
		}
		parseAgyTranscript(data, res)
		break
	}
}

func parseAgyTranscript(data []byte, res *RunResult) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	internalTurns := 0
	initialLoadTurns := 0
	mcpLoadTurns := 0
	mutatingSeen := false
	var pending []int

	for scanner.Scan() {
		var step agyTranscriptStep
		if err := json.Unmarshal(scanner.Bytes(), &step); err != nil {
			continue
		}
		if step.Type == "PLANNER_RESPONSE" {
			internalTurns++
		}
		for _, tc := range step.ToolCalls {
			toolName := strings.Trim(strings.TrimSpace(tc.Name), "\"")
			serverName := ""
			isMCPCall := toolName == "call_mcp_tool"
			if isMCPCall {
				if argsMap, ok := tc.Args.(map[string]any); ok {
					if t, ok := argsMap["ToolName"].(string); ok {
						toolName = strings.Trim(strings.TrimSpace(t), "\"")
					}
					if s, ok := argsMap["ServerName"].(string); ok {
						serverName = strings.Trim(strings.TrimSpace(s), "\"")
					}
				}
			}
			cleanName := strings.Trim(toolName, "\"")
			arguments, err := json.Marshal(tc.Args)
			if err != nil {
				arguments = nil
			}
			callIndex := appendToolCall(res, ToolCall{Name: cleanName, Server: serverName, Arguments: arguments, TransportStatus: ToolCallStatusUnknown, FunctionalStatus: ToolCallStatusUnknown})
			pending = append(pending, callIndex)

			isMutating := isSemanticTool(cleanName, serverName) || isMutatingTool(cleanName)
			if isMutating {
				mutatingSeen = true
			} else if !mutatingSeen {
				initialLoadTurns++
				if isMCPCall || isMCPDiscoveryTool(cleanName) {
					mcpLoadTurns++
				}
			}
		}
		if step.Type == "GENERIC" && len(pending) > 0 {
			for _, callIndex := range pending {
				res.ToolCalls[callIndex].TransportStatus = ToolCallStatusSucceeded
				res.ToolCalls[callIndex].FunctionalStatus = agyToolFunctionalStatus(step.Status)
				if res.ToolCalls[callIndex].FunctionalStatus == ToolCallStatusFailed {
					res.ToolCalls[callIndex].Failure = compactToolFailure(step.Content)
				}
			}
			pending = nil
			refreshMCPVerified(res)
		}
	}

	if internalTurns > 0 {
		res.InternalTurns = internalTurns
	}
	res.InitialLoadTurns = initialLoadTurns
	res.MCPLoadTurns = mcpLoadTurns
	res.ToolCount = len(res.ToolCalls)
	refreshMCPVerified(res)
}

func writeAgyMCPConfig(workDir, repositoryRoot string, arm ArmType, mode MCPServerInstructionMode, env []string) error {
	configDir := filepath.Join(workDir, ".agents")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return fmt.Errorf("create Agy MCP config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "mcp_config.json")
	if _, err := os.Lstat(configPath); err == nil {
		return fmt.Errorf("Agy fixture already contains %s", configPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect Agy MCP config path: %w", err)
	}

	args := []string{"mcp", "--profile", "full"}
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
