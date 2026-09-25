// driver.go coordinates benchmark task setup, provider dispatch, and result evaluation.
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"semedit/internal/pipeline"
)

func boundedDiagnostic(data []byte, limit int) string {
	if len(data) <= limit {
		return string(data)
	}
	return string(data[:limit]) + fmt.Sprintf("\n[stderr truncated; retained first %d bytes]", limit)
}

// HarnessType represents the execution harness (codex, agy, control).
type HarnessType string

const (
	HarnessCodex    HarnessType = "codex"
	HarnessAgy      HarnessType = "agy"
	HarnessOpenCode HarnessType = "opencode"
	HarnessControl  HarnessType = "control"
)

func shouldRetainWorkDir(ctx context.Context, runErr error) bool {
	if ctx.Err() != nil || errors.Is(runErr, context.DeadlineExceeded) {
		return true
	}
	var exitErr *exec.ExitError
	return errors.As(runErr, &exitErr) && exitErr.ExitCode() == -1
}

// ExecuteAgentDriver runs a task via an external harness (codex or agy) and captures telemetry.
func (r *Runner) ExecuteAgentDriver(ctx context.Context, task *Task, target Target, arm ArmType, variant string) (*RunResult, error) {
	start := time.Now()
	runID := fmt.Sprintf("run_%s_%s_%s_%s_%d", target.Harness, arm, safePathFragment(variant), task.Metadata.TaskID, time.Now().UnixNano())
	workDir := filepath.Join(r.baseScratchDir, runID)

	// #nosec G703,G301 -- benchmark work directory is isolated under the configured scratch root.
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	var result *RunResult
	retainWorkDir := false
	defer func() {
		if !retainWorkDir {
			// #nosec G703 -- cleanup ephemeral test harness work directory
			_ = os.RemoveAll(workDir)
			return
		}
		if result != nil {
			if result.Provenance == nil {
				result.Provenance = make(ProvenanceSet)
			}
			result.Provenance["retained_working_directory"] = workDir
		}
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
	if target.Harness == string(HarnessCodex) || target.Harness == string(HarnessOpenCode) {
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
	result = res

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
		prompt = fmt.Sprintf("%s Do not read source code with shell or terminal commands, including `sed`, `cat`, or equivalent commands. Use built-in code inspection tools to inspect source and semedit semantic operations for applicable edits. Use `semantic_lookup` to locate target symbols whenever supported and available. Do not silently fall back to shell-based source reads. If you cannot use `semantic_lookup` because it is unavailable, unsupported for the target, blocked by an unmet precondition, or fails, state the specific reason in your response, then finish with DONE.", baseInstruction)
	case ArmBaseline:
		prompt = fmt.Sprintf("%s Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.", baseInstruction)
	default:
		return nil, fmt.Errorf("unsupported arm for agent driver: %s", arm)
	}
	res.Prompt = prompt

	var sessionID string
	switch target.Harness {
	case string(HarnessCodex):
		sessionID, err = r.runCodex(ctx, workDir, target, prompt, res, "")
		if err != nil {
			res.WallClock = time.Since(start)
			retainWorkDir = shouldRetainWorkDir(ctx, err)
			res.Error = fmt.Sprintf("codex execution: %v", err)
			res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
			return res, nil
		}
	case string(HarnessAgy):
		sessionID, err = r.runAgy(ctx, workDir, target, prompt, res, "")
		if err != nil {
			res.WallClock = time.Since(start)
			retainWorkDir = shouldRetainWorkDir(ctx, err)
			res.Error = fmt.Sprintf("agy execution: %v", err)
			res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
			return res, nil
		}
	case string(HarnessOpenCode):
		sessionID, err = r.runOpenCode(ctx, workDir, target, prompt, res, "")
		if err != nil {
			res.WallClock = time.Since(start)
			retainWorkDir = shouldRetainWorkDir(ctx, err)
			res.Error = fmt.Sprintf("opencode execution: %v", err)
			res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
			return res, nil
		}
	default:
		return nil, fmt.Errorf("unsupported harness: %s", target.Harness)
	}
	res.WallClock = time.Since(start)

	if err := evaluateAgentResult(ctx, task, workDir, beforeFiles, initialPath, beforeContent, res); err != nil {
		res.WallClock = time.Since(start)
		retainWorkDir = shouldRetainWorkDir(ctx, err)
		res.Error = err.Error()
		res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
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
		case string(HarnessOpenCode):
			sessionID, runErr = r.runOpenCode(ctx, workDir, target, followup, turn, sessionID)
		}
		if runErr != nil {
			turn.WallClock = time.Since(turnStarted)
			turn.Error = runErr.Error()
			res.Error = fmt.Sprintf("interactive step %d: %v", index+2, runErr)
			res.WallClock = time.Since(start)
			retainWorkDir = shouldRetainWorkDir(ctx, runErr)
			mergeTurn(res, turn)
			res.InteractionSteps = append(res.InteractionSteps, interactionStep(index+2, followup, turn))
			break
		}
		turn.WallClock = time.Since(turnStarted)
		if err := evaluateAgentResult(ctx, task, workDir, beforeFiles, initialPath, beforeContent, turn); err != nil {
			turn.Error = err.Error()
			res.Error = err.Error()
			res.WallClock = time.Since(start)
			retainWorkDir = shouldRetainWorkDir(ctx, err)
			mergeTurn(res, turn)
			res.InteractionSteps = append(res.InteractionSteps, interactionStep(index+2, followup, turn))
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
	case string(HarnessOpenCode):
		_, runErr = r.runOpenCode(ctx, workDir, target, prompt, turn, sessionID)
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
	slices.Sort(protected)
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
	if turn.Oracle != nil {
		total.Oracle, total.Success = turn.Oracle, turn.Success
	}
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
	modified := slices.Sorted(maps.Keys(paths))
	return modified, nil
}

func formatDiffSummary(filename, before, after string) string {
	bLines := strings.Split(before, "\n")
	aLines := strings.Split(after, "\n")
	return fmt.Sprintf("File %s modified (%d lines -> %d lines)", filename, len(bLines), len(aLines))
}
