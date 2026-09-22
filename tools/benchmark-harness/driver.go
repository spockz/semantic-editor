package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"semedit/internal/gocache"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
)

const maxCodexStderrBytes = 64 * 1024

func boundedDiagnostic(data []byte, limit int) string {
	if len(data) <= limit {
		return string(data)
	}
	return string(data[:limit]) + fmt.Sprintf("\n[stderr truncated; retained first %d bytes]", limit)
}

// HarnessType represents the execution harness (codex, agy, control).
type HarnessType string

const (
	HarnessCodex   HarnessType = "codex"
	HarnessAgy     HarnessType = "agy"
	HarnessControl HarnessType = "control"
)

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
	runID := fmt.Sprintf("run_%s_%s_%s_%s_%d", target.Harness, arm, safePathFragment(variant), task.Metadata.TaskID, time.Now().UnixNano())
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
	if target.Harness == string(HarnessCodex) {
		if err := writeBenchmarkAGENTSOverride(workDir); err != nil {
			return nil, fmt.Errorf("prepare Codex fixture instructions: %w", err)
		}
	}
	beforeFiles, err := snapshotWorkspaceFiles(workDir)
	if err != nil {
		return nil, fmt.Errorf("snapshot benchmark workspace: %w", err)
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

	mcpServerInstructions := MCPServerInstructionsNone
	if target.Harness == string(HarnessCodex) {
		mcpServerInstructions = r.mcpServerInstructionsMode
	}

	res := &RunResult{
		TaskID:                task.Metadata.TaskID,
		Variant:               variant,
		MCPServerInstructions: mcpServerInstructions,
		Provenance:            r.provenanceFor(),
		Target:                target,
		Arm:                   arm,
		BeforeState:           beforeContent,
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
	baseInstruction = withMutationPolicyGuidance(baseInstruction, task, false)

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

	var sessionID string
	switch target.Harness {
	case string(HarnessCodex):
		var runErr error
		sessionID, runErr = r.runCodex(ctx, workDir, target, prompt, res, "")
		if runErr != nil {
			res.Error = fmt.Sprintf("codex execution: %v", runErr)
			return res, nil
		}
	case string(HarnessAgy):
		var runErr error
		sessionID, runErr = r.runAgy(ctx, workDir, target, prompt, res, "")
		if runErr != nil {
			res.Error = fmt.Sprintf("agy execution: %v", runErr)
			return res, nil
		}
	default:
		return nil, fmt.Errorf("unsupported harness: %s", target.Harness)
	}
	res.WallClock = time.Since(start)

	if err := evaluateAgentResult(ctx, task, workDir, beforeFiles, initialPath, beforeContent, res); err != nil {
		res.Error = err.Error()
		return res, nil
	}
	res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
	for index, followup := range task.Metadata.InteractiveFollowups {
		if res.Success {
			break
		}
		followup = withMutationPolicyGuidance(followup, task, true)
		turn := &RunResult{Target: target, Arm: arm, Prompt: followup}
		turnStarted := time.Now()
		var runErr error
		switch target.Harness {
		case string(HarnessCodex):
			sessionID, runErr = r.runCodex(ctx, workDir, target, followup, turn, sessionID)
		case string(HarnessAgy):
			sessionID, runErr = r.runAgy(ctx, workDir, target, followup, turn, sessionID)
		}
		if runErr != nil {
			res.Error = fmt.Sprintf("interactive step %d: %v", index+2, runErr)
			break
		}
		turn.WallClock = time.Since(turnStarted)
		if err := evaluateAgentResult(ctx, task, workDir, beforeFiles, initialPath, beforeContent, turn); err != nil {
			res.Error = err.Error()
			break
		}
		res.InteractionSteps = append(res.InteractionSteps, interactionStep(index+2, followup, turn))
		mergeTurn(res, turn)
	}
	res.WallClock = time.Since(start)
	if shouldRequestSemanticBatchReflection(arm, sessionID, res) {
		res.SemanticBatchReflection = r.requestSemanticToolReflection(ctx, workDir, target, sessionID, semanticBatchReflectionPrompt)
	} else if shouldRequestSemanticToolReflection(arm, sessionID, res) {
		res.SemanticToolReflection = r.requestSemanticToolReflection(ctx, workDir, target, sessionID, semanticToolReflectionPrompt)
	}

	return res, nil
}

// writeBenchmarkAGENTSOverride confines inherited Codex instructions to the synthetic fixture.
func writeBenchmarkAGENTSOverride(workDir string) error {
	path := filepath.Join(workDir, "AGENTS.override.md")
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("fixture already defines reserved %s", filepath.Base(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect fixture instruction override: %w", err)
	}
	if err := pipeline.WriteAtomic(path, []byte(benchmarkAGENTSOverride)); err != nil {
		return fmt.Errorf("write fixture instruction override: %w", err)
	}
	return nil
}

const semanticToolReflectionPrompt = "The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task."

const semanticBatchReflectionPrompt = "The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task."

func shouldRequestSemanticToolReflection(arm ArmType, sessionID string, result *RunResult) bool {
	return arm == ArmSemedit && sessionID != "" && result != nil && !result.MCPVerified
}

func shouldRequestSemanticBatchReflection(arm ArmType, sessionID string, result *RunResult) bool {
	return arm == ArmSemedit && sessionID != "" && result != nil && hasConsecutiveUnbatchedSemanticToolCalls(result.ToolCalls)
}

func hasConsecutiveUnbatchedSemanticToolCalls(calls []ToolCall) bool {
	for _, call := range calls {
		if semanticToolBaseName(call.Name) == "semantic_batch" {
			return false
		}
	}

	consecutive := 0
	for _, call := range calls {
		if isSemanticTool(call.Name, call.Server) {
			consecutive++
			if consecutive >= 2 {
				return true
			}
			continue
		}
		consecutive = 0
	}
	return false
}

func semanticToolBaseName(name string) string {
	clean := strings.ToLower(strings.Trim(name, "\""))
	for _, prefix := range []string{"semedit/", "mcp__semedit__"} {
		if after, found := strings.CutPrefix(clean, prefix); found {
			return after
		}
	}
	return clean
}

func (r *Runner) requestSemanticToolReflection(ctx context.Context, workDir string, target Target, sessionID, prompt string) *SemanticToolReflection {
	turn := &RunResult{Target: target, Arm: ArmSemedit, Prompt: prompt}
	started := time.Now()
	var runErr error
	switch target.Harness {
	case string(HarnessCodex):
		_, runErr = r.runCodex(ctx, workDir, target, prompt, turn, sessionID)
	case string(HarnessAgy):
		_, runErr = r.runAgy(ctx, workDir, target, prompt, turn, sessionID)
	default:
		runErr = fmt.Errorf("unsupported harness: %s", target.Harness)
	}
	reflection := &SemanticToolReflection{
		Prompt:    prompt,
		Response:  turn.agentResponse,
		WallClock: time.Since(started),
		Turns:     turn.Turns,
		ToolCalls: turn.ToolCalls,
	}
	if runErr != nil {
		reflection.Error = runErr.Error()
	} else if reflection.Response == "" {
		reflection.Error = "harness did not expose an assistant reflection response"
	}
	return reflection
}

func safePathFragment(value string) string {
	return strings.NewReplacer(":", "-", "/", "-", `\`, "-").Replace(value)
}

func withMutationPolicyGuidance(prompt string, task *Task, remediate bool) string {
	protected := append([]string(nil), task.Metadata.Oracle.MutationPolicy.DisallowedFiles...)
	if len(protected) == 0 {
		return prompt
	}
	sort.Strings(protected)
	hasTests := false
	otherFiles := make([]string, 0, len(protected))
	for _, path := range protected {
		if strings.HasSuffix(path, "_test.go") {
			hasTests = true
			continue
		}
		otherFiles = append(otherFiles, strconv.Quote(path))
	}
	parts := make([]string, 0, 2)
	if hasTests {
		parts = append(parts, "Do not edit tests.")
	}
	if len(otherFiles) > 0 {
		parts = append(parts, fmt.Sprintf("You are forbidden to modify protected files: %s.", strings.Join(otherFiles, ", ")))
	}
	guidance := strings.Join(parts, " ")
	if remediate {
		guidance += " If an earlier turn changed any protected file, restore its original contents before continuing."
	}
	return fmt.Sprintf("%s\n\n%s", guidance, prompt)
}

func evaluateAgentResult(ctx context.Context, task *Task, workDir string, before workspaceSnapshot, initialPath, beforeContent string, res *RunResult) error {
	// #nosec G304 -- initialPath is derived from the selected fixture oracle path.
	if data, err := os.ReadFile(initialPath); err == nil && beforeContent != "" && strings.TrimSpace(string(data)) != beforeContent {
		res.Diff = formatDiffSummary(task.Metadata.Oracle.AST.File, beforeContent, strings.TrimSpace(string(data)))
	}
	modified, err := changedWorkspaceFiles(workDir, before)
	if err != nil {
		return fmt.Errorf("snapshot benchmark workspace changes: %w", err)
	}
	oracle, err := Evaluate(ctx, task, workDir, modified)
	if err != nil {
		return fmt.Errorf("oracle evaluation: %w", err)
	}
	res.Oracle, res.Success = oracle, oracle.Passed
	return nil
}

func interactionStep(step int, prompt string, res *RunResult) InteractionStep {
	return InteractionStep{Step: step, Prompt: prompt, WallClock: res.WallClock, Turns: res.Turns, ToolCalls: res.ToolCalls, Oracle: res.Oracle, Error: res.Error}
}

func mergeTurn(total, turn *RunResult) {
	total.Turns += turn.Turns
	total.InitialLoadTurns += turn.InitialLoadTurns
	total.MCPLoadTurns += turn.MCPLoadTurns
	total.InternalTurns += turn.InternalTurns
	total.ToolCalls = append(total.ToolCalls, turn.ToolCalls...)
	total.ToolsUsed = append(total.ToolsUsed, turn.ToolsUsed...)
	total.ToolCount = len(total.ToolCalls)
	total.PromptTokens += turn.PromptTokens
	total.CachedPromptTokens += turn.CachedPromptTokens
	total.UncachedPromptTokens += turn.UncachedPromptTokens
	total.OutputTokens += turn.OutputTokens
	total.ReasoningTokens += turn.ReasoningTokens
	total.Oracle, total.Success = turn.Oracle, turn.Success
}

type workspaceSnapshot map[string][sha256.Size]byte

func snapshotWorkspaceFiles(root string) (workspaceSnapshot, error) {
	files := make(workspaceSnapshot)
	workspaceFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open benchmark workspace: %w", err)
	}
	defer func() {
		_ = workspaceFS.Close()
	}()

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".scratch" {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relativize %s: %w", path, err)
		}
		data, err := workspaceFS.ReadFile(rel)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		files[filepath.ToSlash(rel)] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func changedWorkspaceFiles(root string, before workspaceSnapshot) ([]string, error) {
	after, err := snapshotWorkspaceFiles(root)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]struct{}, len(before)+len(after))
	for path, hash := range before {
		if afterHash, found := after[path]; !found || afterHash != hash {
			paths[path] = struct{}{}
		}
	}
	for path := range after {
		if _, found := before[path]; !found {
			paths[path] = struct{}{}
		}
	}
	modified := make([]string, 0, len(paths))
	for path := range paths {
		modified = append(modified, path)
	}
	sort.Strings(modified)
	return modified, nil
}

func formatDiffSummary(filename, before, after string) string {
	bLines := strings.Split(before, "\n")
	aLines := strings.Split(after, "\n")
	return fmt.Sprintf("File %s modified (%d lines -> %d lines)", filename, len(bLines), len(aLines))
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

func (r *Runner) runAgy(ctx context.Context, workDir string, target Target, prompt string, res *RunResult, resumeID string) (string, error) {
	agyBin := "/Users/alessandro/.local/bin/agy"
	if _, err := os.Stat(agyBin); err != nil {
		agyBin = "agy"
	}

	absWorkDir, _ := filepath.Abs(workDir)
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

	// #nosec G204 -- external driver invocation controlled by benchmark harness
	cmd := exec.CommandContext(ctx, agyBin, args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("agy exec error: %w (output: %s)", err, string(out))
	}

	var resp AgyResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("unmarshal agy json response: %w (raw: %s)", err, string(out))
	}

	if resp.Status != "SUCCESS" && resp.Error != "" {
		return "", fmt.Errorf("agy error status (%s): %s", resp.Status, resp.Error)
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
	res.agentResponse = strings.TrimSpace(resp.Response)

	// Inspect transcript for tools used
	if resp.ConversationID != "" {
		extractAgyTools(resp.ConversationID, res)
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
