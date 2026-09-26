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

func (r *Runner) ExecuteAgentDriver(ctx context.Context, task *Task, target Target, arm ArmType, variant string) (*RunResult, error) {
	execution, err := ResolveAgentExecution(task, target, arm, variant, r.semeditArmRestriction)
	if err != nil {
		return nil, err
	}
	return r.ExecuteAgent(ctx, execution)
}

func (r *Runner) ExecuteAgent(ctx context.Context, execution AgentExecution) (result *RunResult, executeErr error) {
	task, target, arm, variant := execution.Task, execution.Target, execution.Arm, execution.Variant
	if task == nil {
		return nil, fmt.Errorf("execute agent: task is nil")
	}
	adapter, err := selectProvider(r, target)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	runID := fmt.Sprintf("run_%s_%s_%s_%s_%d", target.Harness, arm, safePathFragment(variant), task.Metadata.TaskID, time.Now().UnixNano())
	workDir := filepath.Join(r.baseScratchDir, runID)
	// #nosec G703,G301 -- benchmark work directory is isolated under the configured scratch root.
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	session, err := newAgentSession(adapter, r.baseScratchDir, runID, workDir)
	if err != nil {
		// #nosec G703 -- cleanup isolated workspace when session initialization fails.
		removeErr := os.RemoveAll(workDir)
		return nil, errors.Join(err, removeErr)
	}
	session.execution = execution
	session.started = started
	defer func() {
		if closeErr := session.close(); closeErr != nil {
			if session.result != nil {
				session.result.Success = false
				if strings.TrimSpace(session.result.Error) == "" {
					session.result.Error = closeErr.Error()
				} else {
					session.result.Error = session.result.Error + "\n" + closeErr.Error()
				}
			}
			executeErr = errors.Join(executeErr, closeErr)
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
	session.beforeFiles, err = snapshotWorkspaceFiles(workDir)
	if err != nil {
		return nil, fmt.Errorf("snapshot benchmark workspace: %w", err)
	}
	targetFile := task.Metadata.Oracle.AST.File
	if targetFile == "" {
		targetFile = "api/server.go"
	}
	session.initialPath = filepath.Join(workDir, filepath.FromSlash(targetFile))
	// #nosec G304,G703 -- initialPath is derived from the selected fixture oracle path.
	if data, err := os.ReadFile(session.initialPath); err == nil {
		session.beforeContent = strings.TrimSpace(string(data))
	}
	mcpInstructions := MCPServerInstructionsNone
	if target.Harness == string(HarnessCodex) || target.Harness == string(HarnessOpenCode) {
		mcpInstructions = r.mcpServerInstructionsMode
	}
	res := &RunResult{
		TaskID: task.Metadata.TaskID, Variant: variant,
		PromptVariant:                promptVariantFromExecution(execution),
		SemeditArmRestrict:           execution.Policy,
		SemeditArmRestrictionApplied: arm == ArmSemedit,
		MCPServerInstructions:        mcpInstructions,
		Provenance:                   r.provenanceFor(), Target: target, Arm: arm, BeforeState: session.beforeContent,
	}
	session.result = res
	if strings.HasPrefix(target.Model, "openrouter/") {
		if res.Provenance == nil {
			res.Provenance = make(ProvenanceSet)
		}
		res.Provenance["openrouter_auth"] = "environment"
		if strings.HasSuffix(target.Model, ":free") {
			res.Provenance["openrouter_catalog_as_of"] = openRouterFreeCatalogAsOf
			res.Provenance["openrouter_catalog_source"] = openRouterFreeCatalogURL
		}
	}
	prompt := execution.Prompt
	if prompt == "" {
		return nil, fmt.Errorf("planned agent execution has empty initial prompt")
	}
	res.Prompt = prompt
	if res.Provenance == nil {
		res.Provenance = make(ProvenanceSet)
	}
	res.Provenance["session_transcript"] = session.transcriptPath
	err = session.runTurn(ctx, target, prompt, "task", res)
	if err != nil {
		res.WallClock = time.Since(started)
		session.retainWorkDir = shouldRetainWorkDir(ctx, err)
		res.Error = fmt.Sprintf("%s execution: %v", target.Harness, err)
		res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
		return res, nil
	}
	res.WallClock = time.Since(started)
	if err := session.evaluate(ctx, res); err != nil {
		res.WallClock = time.Since(started)
		session.retainWorkDir = shouldRetainWorkDir(ctx, err)
		res.Error = err.Error()
		res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
		return res, nil
	}
	res.InteractionSteps = append(res.InteractionSteps, interactionStep(1, prompt, res))
	_, session.retainWorkDir = session.runFollowups(ctx)
	sessionID := session.resumeID
	res.WallClock = time.Since(started)
	if shouldRequestSemanticBatchReflection(arm, sessionID, res) {
		res.SemanticBatchReflection = r.requestSemanticToolReflection(ctx, session, target, semanticBatchReflectionPrompt)
	} else if shouldRequestSemanticToolReflection(arm, sessionID, res) {
		res.SemanticToolReflection = r.requestSemanticToolReflection(ctx, session, target, semanticToolReflectionPrompt)
	}
	return res, nil
}

func prepareAgentPromptWithPolicy(task *Task, arm ArmType, variant string, res *RunResult, policy ...SemeditArmRestriction) (string, error) {
	effectivePolicy := SemeditArmRestrictWrite
	if len(policy) > 1 {
		return "", fmt.Errorf("prepare agent prompt accepts at most one semedit arm restriction")
	}
	if len(policy) == 1 {
		effectivePolicy = policy[0]
	}
	baseInstruction := task.Metadata.Instruction
	promptVarName := ""
	if _, after, ok := strings.Cut(variant, ":"); ok {
		promptVarName = after
	}
	if promptVarName != "" && len(task.Metadata.PromptVariants) > 0 {
		if customInstruction, ok := task.Metadata.PromptVariants[promptVarName]; ok {
			baseInstruction = customInstruction
		}
	}
	res.PromptVariant = promptVarName
	if strings.Contains(strings.ToLower(variant), "verified") && task.Metadata.VerificationConstraint != "" {
		baseInstruction = fmt.Sprintf("%s %s", baseInstruction, task.Metadata.VerificationConstraint)
	}
	baseInstruction = withMutationPolicyGuidance(baseInstruction, task, false)
	if arm == ArmSemedit {
		steering, err := semeditRestrictionSteering(effectivePolicy)
		if err != nil {
			return "", err
		}
		baseInstruction = strings.TrimSpace(baseInstruction + " " + steering)
		return fmt.Sprintf("%s When done, output DONE.", baseInstruction), nil
	}
	if arm == ArmBaseline {
		return fmt.Sprintf("%s Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.", baseInstruction), nil
	}
	return "", fmt.Errorf("unsupported arm for agent driver: %s", arm)
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

func (r *Runner) requestSemanticToolReflection(ctx context.Context, session *agentSession, target Target, prompt string) *SemanticToolReflection {
	turn := &RunResult{Target: target, Arm: ArmSemedit, Prompt: prompt}
	started := time.Now()
	runErr := session.runTurn(ctx, target, prompt, "diagnostic_reflection", turn)
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
	total.MCPVerified = total.MCPVerified || turn.MCPVerified
	if turn.Oracle != nil {
		total.Oracle, total.Success = turn.Oracle, turn.Success
	}
	if turn.Error != "" {
		total.Error = turn.Error
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
