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
	selectedAdapter, selectErr := selectProvider(r, target)
	if selectErr != nil {
		return nil, selectErr
	}
	runID := fmt.Sprintf("run_%s_%s_%s_%s_%d", target.Harness, arm, safePathFragment(variant), task.Metadata.TaskID, time.Now().UnixNano())
	workDir := filepath.Join(r.baseScratchDir, runID)
	// #nosec G703,G301 -- benchmark work directory is isolated under the configured scratch root.
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	session, sessionErr := newAgentSession(selectedAdapter, r.baseScratchDir, runID, workDir)
	if sessionErr != nil {
		// #nosec G703 -- cleanup isolated workspace when session initialization fails.
		removeErr := os.RemoveAll(workDir)
		return nil, errors.Join(sessionErr, removeErr)
	}
	session.execution = execution
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
	workspace, workspaceErr := prepareAgentWorkspace(ctx, r, task, target, arm, variant, workDir, session)
	if workspaceErr != nil {
		return nil, workspaceErr
	}
	mcpInstructions := MCPServerInstructionsNone
	if target.Harness == string(HarnessCodex) || target.Harness == string(HarnessOpenCode) {
		mcpInstructions = r.mcpServerInstructionsMode
	}
	res := newAgentRunResult(r, task, agentRunResultInput{
		target: target, arm: arm, variant: variant, execution: execution,
		session: session, workspace: workspace, mcpInstructions: mcpInstructions,
	})
	session.result = res
	prompt, err := prepareAgentRunMetadata(execution, session, res, r.semeditPrewarmVerify)
	if err != nil {
		return nil, err
	}
	initialTask := runInitialAgentTask(ctx, session, target, task, prompt, res)
	if initialTask.stopEarly {
		return res, nil
	}
	r.runAgentFollowups(ctx, session, target, arm, initialTask.started, res)
	if isSemeditArm(arm) && session.resumeID != "" && res.Success {
		res.SemanticIntegrationReflection = r.requestSemanticToolReflection(ctx, session, target, semanticIntegrationReflectionPrompt)
	}
	return res, nil
}

type agentWorkspaceSetup struct {
	prewarmApplied  bool
	prewarmDuration time.Duration
}

func prepareAgentWorkspace(ctx context.Context, runner *Runner, task *Task, target Target, arm ArmType, variant, workDir string, session *agentSession) (agentWorkspaceSetup, error) {
	if err := task.ExtractVariantTo(workDir, variant); err != nil {
		return agentWorkspaceSetup{}, fmt.Errorf("extract task fixture: %w", err)
	}
	if target.Harness == string(HarnessOpenCode) {
		if err := initializeOpenCodeFixtureGit(ctx, workDir); err != nil {
			return agentWorkspaceSetup{}, err
		}
	}
	if target.Harness == string(HarnessCodex) {
		if err := writeBenchmarkAGENTSOverride(workDir); err != nil {
			return agentWorkspaceSetup{}, fmt.Errorf("prepare Codex fixture instructions: %w", err)
		}
	}
	setup := agentWorkspaceSetup{}
	if isSemeditArm(arm) && runner.semeditPrewarmVerify && target.Harness == string(HarnessCodex) && task.Metadata.Oracle.AST.File != "" {
		binaryPath := filepath.Join(filepath.Dir(filepath.Dir(runner.baseScratchDir)), "bin", "semedit-next")
		var err error
		setup.prewarmDuration, err = runSemeditVerifyWarmup(ctx, workDir, task.Metadata.Oracle.AST.File, binaryPath)
		if err != nil {
			return agentWorkspaceSetup{}, err
		}
		setup.prewarmApplied = true
	}
	beforeFiles, err := snapshotWorkspaceFiles(workDir)
	if err != nil {
		return agentWorkspaceSetup{}, fmt.Errorf("snapshot benchmark workspace: %w", err)
	}
	session.beforeFiles = beforeFiles
	if err := setAgentInitialFile(session, task, workDir); err != nil {
		return agentWorkspaceSetup{}, err
	}
	return setup, nil
}

func setAgentInitialFile(session *agentSession, task *Task, workDir string) error {
	targetFile := task.Metadata.Oracle.AST.File
	if targetFile == "" {
		targetFile = "api/server.go"
	}
	session.initialPath = filepath.Join(workDir, filepath.FromSlash(targetFile))
	// #nosec G304,G703 -- initialPath is derived from the selected fixture oracle path.
	data, err := os.ReadFile(session.initialPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read initial benchmark file: %w", err)
	}
	session.beforeContent = strings.TrimSpace(string(data))
	return nil
}

type agentRunResultInput struct {
	target          Target
	arm             ArmType
	variant         string
	execution       AgentExecution
	session         *agentSession
	workspace       agentWorkspaceSetup
	mcpInstructions MCPServerInstructionMode
}

func newAgentRunResult(r *Runner, task *Task, input agentRunResultInput) *RunResult {
	return &RunResult{
		TaskID: task.Metadata.TaskID, Variant: input.variant,
		InteractiveMode:              task.Metadata.InteractiveMode,
		PromptVariant:                promptVariantFromExecution(input.execution),
		SemeditArmRestrict:           input.execution.Policy,
		SemeditArmRestrictionApplied: isSemeditArm(input.arm),
		SemeditPrewarmVerify:         input.workspace.prewarmApplied,
		SemeditPrewarmDurationMS:     input.workspace.prewarmDuration.Milliseconds(),
		MCPServerInstructions:        input.mcpInstructions,
		Provenance:                   r.provenanceFor(), Target: input.target, Arm: input.arm, BeforeState: input.session.beforeContent,
	}
}

func prepareAgentRunMetadata(execution AgentExecution, session *agentSession, result *RunResult, prewarmConfigured bool) (string, error) {
	if strings.HasPrefix(execution.Target.Model, "openrouter/") {
		ensureProvenance(result)
		result.Provenance["openrouter_auth"] = "environment"
		if strings.HasSuffix(execution.Target.Model, ":free") {
			result.Provenance["openrouter_catalog_as_of"] = openRouterFreeCatalogAsOf
			result.Provenance["openrouter_catalog_source"] = openRouterFreeCatalogURL
		}
	}
	if execution.Prompt == "" {
		return "", fmt.Errorf("planned agent execution has empty initial prompt")
	}
	result.Prompt = execution.Prompt
	ensureProvenance(result)
	result.Provenance["session_transcript"] = session.transcriptPath
	if result.SemeditPrewarmVerify {
		result.Provenance["semedit_prewarm"] = "verify_cli_check_only"
		result.Provenance["go_cache_initial_state"] = "empty"
	} else if isSemeditArm(execution.Arm) && prewarmConfigured {
		result.Provenance["semedit_prewarm_skipped"] = "no_go_target_or_unsupported_provider"
	}
	return execution.Prompt, nil
}

func ensureProvenance(result *RunResult) {
	if result.Provenance == nil {
		result.Provenance = make(ProvenanceSet)
	}
}

type initialAgentTaskResult struct {
	started   time.Time
	stopEarly bool
}

func runInitialAgentTask(ctx context.Context, session *agentSession, target Target, task *Task, prompt string, result *RunResult) initialAgentTaskResult {
	started := time.Now()
	session.started = started
	runErr := session.runTurn(ctx, target, prompt, "task", result)
	if result.processStartedAt.IsZero() {
		started = time.Time{}
	} else {
		started = result.processStartedAt
	}
	session.started = started
	if runErr != nil {
		result.WallClock = elapsedTaskTime(started)
		session.retainWorkDir = shouldRetainWorkDir(ctx, runErr)
		result.Error = fmt.Sprintf("%s execution: %v", target.Harness, runErr)
		result.InteractionSteps = append(result.InteractionSteps, initialInteractionStep(task, prompt, result))
		return initialAgentTaskResult{started: started, stopEarly: true}
	}
	result.WallClock = elapsedTaskTime(started)
	if err := session.evaluate(ctx, result); err != nil {
		result.WallClock = elapsedTaskTime(started)
		session.retainWorkDir = shouldRetainWorkDir(ctx, err)
		result.Error = err.Error()
		result.InteractionSteps = append(result.InteractionSteps, initialInteractionStep(task, prompt, result))
		return initialAgentTaskResult{started: started, stopEarly: true}
	}
	if task.Metadata.InteractiveMode == "staged" {
		result.WallClock = elapsedTaskTime(started)
	}
	result.InteractionSteps = append(result.InteractionSteps, initialInteractionStep(task, prompt, result))
	return initialAgentTaskResult{started: started}
}

func (r *Runner) runAgentFollowups(ctx context.Context, session *agentSession, target Target, arm ArmType, started time.Time, result *RunResult) {
	if session.execution.Task.Metadata.InteractiveMode == "staged" {
		_, session.retainWorkDir = session.runStagedFollowups(ctx)
	} else {
		_, session.retainWorkDir = session.runFollowups(ctx)
	}
	result.WallClock = elapsedTaskTime(started)
	if shouldRequestSemanticBatchReflection(arm, session.resumeID, result) {
		result.SemanticBatchReflection = r.requestSemanticToolReflection(ctx, session, target, semanticBatchReflectionPrompt)
	} else if shouldRequestSemanticToolReflection(arm, session.resumeID, result) {
		result.SemanticToolReflection = r.requestSemanticToolReflection(ctx, session, target, semanticToolReflectionPrompt)
	}
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
	if promptVarName != "" {
		promptName := promptVarName
		switch promptName {
		case "implicit":
			promptName = "default"
		case "explicit":
			if arm != ArmSemeditExplicit {
				promptName = "default"
			}
		}
		if customInstruction, ok := task.Metadata.PromptVariants[promptName]; ok {
			baseInstruction = customInstruction
		}
	}
	res.PromptVariant = promptVarName
	if strings.Contains(strings.ToLower(variant), "verified") && task.Metadata.VerificationConstraint != "" {
		baseInstruction = fmt.Sprintf("%s %s", baseInstruction, task.Metadata.VerificationConstraint)
	}
	baseInstruction = withMutationPolicyGuidance(baseInstruction, task, false)
	if arm == ArmSemeditExplicit {
		baseInstruction = strings.TrimSpace(baseInstruction + " Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently.")
	}
	if isSemeditArm(arm) {
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

const semanticIntegrationReflectionPrompt = "The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit's tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit's tools or not. This sentence will be the first thing a potential new user sees about semedit's usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits."

func shouldRequestSemanticToolReflection(arm ArmType, sessionID string, result *RunResult) bool {
	return isSemeditArm(arm) && sessionID != "" && result != nil && result.Success && !result.MCPVerified
}

func shouldRequestSemanticBatchReflection(arm ArmType, sessionID string, result *RunResult) bool {
	return isSemeditArm(arm) && sessionID != "" && result != nil && result.Success && hasConsecutiveUnbatchedSemanticToolCalls(result.ToolCalls)
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
	if after, found := strings.CutPrefix(clean, "semedit_semantic_"); found {
		return "semantic_" + after
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
	if slices.Contains(protected, "*") {
		return fmt.Sprintf("Do not modify any workspace files.\n\n%s", prompt)
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
	return InteractionStep{
		Step:                             step,
		Prompt:                           prompt,
		WallClock:                        res.WallClock,
		Turns:                            res.Turns,
		PromptTokens:                     res.PromptTokens,
		CachedPromptTokens:               res.CachedPromptTokens,
		UncachedPromptTokens:             res.UncachedPromptTokens,
		OutputTokens:                     res.OutputTokens,
		ReasoningTokens:                  res.ReasoningTokens,
		InitialLoadTurns:                 res.InitialLoadTurns,
		MCPLoadTurns:                     res.MCPLoadTurns,
		InternalTurns:                    res.InternalTurns,
		ProcessStartToFirstEvent:         res.ProcessStartToFirstEvent,
		MCPInitializeToFirstSemanticCall: res.MCPInitializeToFirstSemanticCall,
		ToolCalls:                        res.ToolCalls,
		Oracle:                           res.Oracle,
		Error:                            res.Error,
	}
}

func stagedInteractionStep(step int, totalEdits int, prompt string, res *RunResult) InteractionStep {
	result := interactionStep(step, prompt, res)
	result.TotalEdits = totalEdits
	return result
}

func initialInteractionStep(task *Task, prompt string, res *RunResult) InteractionStep {
	if task.Metadata.InteractiveMode == "staged" {
		return stagedInteractionStep(1, task.Metadata.StagedInitialTotalEdits, prompt, res)
	}
	return interactionStep(1, prompt, res)
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
		relativePath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("relativize %s: %w", path, relErr)
		}
		data, readErr := workspaceFS.ReadFile(relativePath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		files[filepath.ToSlash(relativePath)] = sha256.Sum256(data)
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
