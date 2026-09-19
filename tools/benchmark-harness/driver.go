package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HarnessType represents the execution harness (codex, agy, control).
type HarnessType string

const (
	HarnessCodex   HarnessType = "codex"
	HarnessAgy     HarnessType = "agy"
	HarnessControl HarnessType = "control"
)

// CodexEvent represents a line from codex exec --json.
type CodexEvent struct {
	Type string `json:"type"`
	Item *struct {
		Type    string `json:"type"`
		Command string `json:"command,omitempty"`
		Text    string `json:"text,omitempty"`
	} `json:"item,omitempty"`
	Usage *struct {
		InputTokens           int `json:"input_tokens"`
		CachedInputTokens     int `json:"cached_input_tokens"`
		OutputTokens          int `json:"output_tokens"`
		ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	} `json:"usage,omitempty"`
}

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

// ExecuteAgentDriver runs a task via an external harness (codex or agy) and captures telemetry.
func (r *Runner) ExecuteAgentDriver(ctx context.Context, task *Task, target Target, arm ArmType, variant string) (*RunResult, error) {
	start := time.Now()
	runID := fmt.Sprintf("run_%s_%s_%s_%s_%d", target.Harness, arm, variant, task.Metadata.TaskID, time.Now().UnixNano())
	workDir := filepath.Join(r.baseScratchDir, runID)

	// #nosec G703,G301 -- ephemeral test harness work directory
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	defer func() {
		// #nosec G703 -- cleanup ephemeral test harness work directory
		_ = os.RemoveAll(workDir)
	}()

	if err := task.ExtractVariantTo(workDir, variant); err != nil {
		return nil, fmt.Errorf("extract task fixture: %w", err)
	}

	targetFile := task.Metadata.Oracle.AST.File
	if targetFile == "" {
		targetFile = "api/server.go"
	}

	beforeContent := ""
	initialPath := filepath.Join(workDir, filepath.FromSlash(targetFile))
	// #nosec G304,G703 -- reading initial state file inside isolated benchmark workspace
	if data, err := os.ReadFile(initialPath); err == nil {
		beforeContent = strings.TrimSpace(string(data))
	}

	res := &RunResult{
		TaskID:      task.Metadata.TaskID,
		Variant:     variant,
		Target:      target,
		Arm:         arm,
		BeforeState: beforeContent,
	}

	baseInstruction := task.Metadata.Instruction

	// Check if a specific prompt variant is requested via variant name (e.g. "small:crypto_rand", "small+verified:crypto_rand")
	promptVarName := ""
	if _, after, ok := strings.Cut(variant, ":"); ok {
		promptVarName = after
	}
	if promptVarName != "" && len(task.Metadata.PromptVariants) > 0 {
		if customInstr, ok := task.Metadata.PromptVariants[promptVarName]; ok {
			baseInstruction = customInstr
		}
	}
	res.PromptVariant = promptVarName

	if strings.Contains(strings.ToLower(variant), "verified") && task.Metadata.VerificationConstraint != "" {
		baseInstruction = fmt.Sprintf("%s %s", baseInstruction, task.Metadata.VerificationConstraint)
	}

	var prompt string
	switch arm {
	case ArmSemedit:
		prompt = fmt.Sprintf("%s Prefer using semantic editor operations if applicable. When done, output DONE.", baseInstruction)
	case ArmBaseline:
		prompt = fmt.Sprintf("%s Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.", baseInstruction)
	default:
		return nil, fmt.Errorf("unsupported arm for agent driver: %s", arm)
	}
	res.Prompt = prompt

	switch target.Harness {
	case string(HarnessCodex):
		if err := r.runCodex(ctx, workDir, target, prompt, res); err != nil {
			res.Error = fmt.Sprintf("codex execution: %v", err)
			return res, nil
		}
	case string(HarnessAgy):
		if err := r.runAgy(ctx, workDir, target, prompt, res); err != nil {
			res.Error = fmt.Sprintf("agy execution: %v", err)
			return res, nil
		}
	default:
		return nil, fmt.Errorf("unsupported harness: %s", target.Harness)
	}

	res.WallClock = time.Since(start)

	// Capture resulting diff
	afterContent := ""
	// #nosec G304,G703 -- reading final state file inside isolated benchmark workspace
	if data, err := os.ReadFile(initialPath); err == nil {
		afterContent = strings.TrimSpace(string(data))
	}
	if beforeContent != "" && afterContent != "" && beforeContent != afterContent {
		res.Diff = formatDiffSummary(targetFile, beforeContent, afterContent)
	}

	oracleRes, err := Evaluate(ctx, task, workDir, []string{targetFile})
	if err != nil {
		res.Error = fmt.Sprintf("oracle evaluation: %v", err)
		return res, nil
	}
	res.Oracle = oracleRes
	res.Success = oracleRes.Passed

	return res, nil
}

func formatDiffSummary(filename, before, after string) string {
	bLines := strings.Split(before, "\n")
	aLines := strings.Split(after, "\n")
	return fmt.Sprintf("File %s modified (%d lines -> %d lines)", filename, len(bLines), len(aLines))
}

func (r *Runner) runCodex(ctx context.Context, workDir string, target Target, prompt string, res *RunResult) error {
	args := []string{"exec", "--json", "--ephemeral", "-s", "workspace-write", "-C", workDir}
	if target.Model != "" {
		args = append(args, "--model", target.Model)
	}
	if target.Effort != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", target.Effort))
	}
	args = append(args, prompt)

	// #nosec G204 -- external driver invocation controlled by benchmark harness
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start codex: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	turns := 0
	internalTurns := 0
	initialLoadTurns := 0
	mcpLoadTurns := 0
	mutatingSeen := false

	for scanner.Scan() {
		line := scanner.Bytes()
		var ev CodexEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.Type == "turn.started" {
			turns++
		}
		if ev.Item != nil && ev.Item.Type == "tool_call" {
			internalTurns++
			toolName := ev.Item.Command
			if toolName == "" {
				toolName = ev.Item.Text
			}
			if toolName != "" {
				res.ToolsUsed = append(res.ToolsUsed, toolName)
				isMCP := strings.HasPrefix(toolName, "semantic_") || strings.Contains(toolName, "semedit")
				if isMCP {
					res.MCPVerified = true
				}

				isMutating := isMCP || isMutatingTool(toolName)
				if isMutating {
					mutatingSeen = true
				} else if !mutatingSeen {
					initialLoadTurns++
					if isMCPDiscoveryTool(toolName) {
						mcpLoadTurns++
					}
				}
			}
		}
		if ev.Usage != nil {
			res.PromptTokens = ev.Usage.InputTokens
			res.CachedPromptTokens = ev.Usage.CachedInputTokens
			res.UncachedPromptTokens = ev.Usage.InputTokens - ev.Usage.CachedInputTokens
			res.OutputTokens = ev.Usage.OutputTokens
			res.ReasoningTokens = ev.Usage.ReasoningOutputTokens
		}
	}

	res.Turns = 1
	if turns > 0 {
		res.Turns = turns
	}
	res.InternalTurns = internalTurns
	res.InitialLoadTurns = initialLoadTurns
	res.MCPLoadTurns = mcpLoadTurns
	res.ToolCount = len(res.ToolsUsed)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait codex: %w", err)
	}
	return nil
}

func (r *Runner) runAgy(ctx context.Context, workDir string, target Target, prompt string, res *RunResult) error {
	agyBin := "/Users/alessandro/.local/bin/agy"
	if _, err := os.Stat(agyBin); err != nil {
		agyBin = "agy"
	}

	absWorkDir, _ := filepath.Abs(workDir)
	args := []string{"--output-format", "json", "--dangerously-skip-permissions", "--add-dir", absWorkDir}
	if target.Model != "" {
		args = append(args, "--model", target.Model)
	}
	if target.Effort != "" {
		args = append(args, "--effort", target.Effort)
	}

	fullPrompt := fmt.Sprintf("Working directory is %s.\n%s", absWorkDir, prompt)
	args = append(args, "-p", fullPrompt)

	// #nosec G204 -- external driver invocation controlled by benchmark harness
	cmd := exec.CommandContext(ctx, agyBin, args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("agy exec error: %w (output: %s)", err, string(out))
	}

	var resp AgyResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return fmt.Errorf("unmarshal agy json response: %w (raw: %s)", err, string(out))
	}

	if resp.Status != "SUCCESS" && resp.Error != "" {
		return fmt.Errorf("agy error status (%s): %s", resp.Status, resp.Error)
	}

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

	// Inspect transcript for tools used
	if resp.ConversationID != "" {
		extractAgyTools(resp.ConversationID, res)
	}

	return nil
}

type agyTranscriptStep struct {
	Type      string `json:"type"`
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
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		internalTurns := 0
		initialLoadTurns := 0
		mcpLoadTurns := 0
		mutatingSeen := false

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
				isMCPCall := toolName == "call_mcp_tool"
				if isMCPCall {
					if argsMap, ok := tc.Args.(map[string]any); ok {
						if t, ok := argsMap["ToolName"].(string); ok {
							toolName = strings.Trim(strings.TrimSpace(t), "\"")
						}
					}
				}
				res.ToolsUsed = append(res.ToolsUsed, toolName)
				cleanName := strings.Trim(toolName, "\"")
				if strings.HasPrefix(cleanName, "semantic_") || strings.Contains(cleanName, "semedit") {
					res.MCPVerified = true
				}

				isMutating := strings.HasPrefix(cleanName, "semantic_") || strings.Contains(cleanName, "semedit") || isMutatingTool(cleanName)
				if isMutating {
					mutatingSeen = true
				} else if !mutatingSeen {
					initialLoadTurns++
					if isMCPCall || isMCPDiscoveryTool(cleanName) {
						mcpLoadTurns++
					}
				}
			}
		}

		if internalTurns > 0 {
			res.InternalTurns = internalTurns
		}
		res.InitialLoadTurns = initialLoadTurns
		res.MCPLoadTurns = mcpLoadTurns
		res.ToolCount = len(res.ToolsUsed)
		break
	}
}
