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

	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
)

const (
	semanticTimeoutDiagnosticReflectionPrompt  = "The benchmark task stopped because its execution timed out. Do not use tools, edit files, or retry the task. Based only on the conversation so far, identify the last action underway, where time appears to have gone (model reasoning, tool wait, repeated verification or undo, exploration, or another cause), and the strongest evidence for that diagnosis. Clearly separate observed evidence from inference, and answer in a few concise sentences."
	semanticTimeoutIntegrationReflectionPrompt = "The benchmark task stopped because its execution timed out. For benchmark analysis only, do not use tools, edit files, or retry the task. In one sentence, say whether semedit's tools helped or hindered the work before the timeout, grounded in what actually happened. If you could not assess their value, say so."
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

const HarnessCopilot HarnessType = "copilot"

func shouldRetainWorkDir(ctx context.Context, runErr error) bool {
	if ctx.Err() != nil || errors.Is(runErr, context.DeadlineExceeded) {
		return true
	}
	var exitErr *exec.ExitError
	return errors.As(runErr, &exitErr) && exitErr.ExitCode() == -1
}

type agentTaskTimeoutKey struct{}

func withAgentTaskTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, agentTaskTimeoutKey{}, timeout)
}

func agentTaskContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout, ok := ctx.Value(agentTaskTimeoutKey{}).(time.Duration)
	if !ok || timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func appendResultError(existing string, err error) string {
	for _, component := range errorComponents(err) {
		if strings.TrimSpace(component.Error()) == "" || containsErrorMessage(existing, component.Error()) {
			continue
		}
		if strings.TrimSpace(existing) == "" {
			existing = component.Error()
			continue
		}
		existing += "\n" + component.Error()
	}
	return existing
}

func joinDistinctErrors(existing, next error) error {
	missing := make([]error, 0)
	for _, component := range errorComponents(next) {
		if existing == nil || !sameError(existing, component) {
			missing = append(missing, component)
		}
	}
	return errors.Join(append([]error{existing}, missing...)...)
}

func containsErrorMessage(messages, candidate string) bool {
	return messages == candidate || strings.HasPrefix(messages, candidate+"\n") || strings.HasSuffix(messages, "\n"+candidate) || strings.Contains(messages, "\n"+candidate+"\n")
}

func sameError(existing, candidate error) bool {
	return errors.Is(existing, candidate) || errors.Is(candidate, existing) || existing.Error() == candidate.Error()
}

func errorComponents(err error) []error {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var components []error
		for _, child := range joined.Unwrap() {
			components = append(components, errorComponents(child)...)
		}
		return components
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return errorComponents(wrapped)
	}
	return []error{err}
}

func (r *Runner) ExecuteAgentDriver(ctx context.Context, task *Task, target Target, arm ArmType, variant string) (*RunResult, error) {
	execution, err := ResolveAgentExecution(task, target, arm, variant, r.semeditArmRestriction)
	if err != nil {
		return nil, err
	}
	return r.ExecuteAgent(ctx, execution)
}

func (r *Runner) ExecuteAgent(ctx context.Context, execution AgentExecution) (result *RunResult, executeErr error) {
	ctx, err := prepareAgentStorageContext(ctx, r)
	if err != nil {
		return nil, err
	}

	task, target, arm, variant := execution.Task, execution.Target, execution.Arm, execution.Variant
	if task == nil {
		return nil, fmt.Errorf("execute agent: task is nil")
	}
	selectedAdapter, selectErr := selectProvider(r, target)
	if selectErr != nil {
		return nil, selectErr
	}
	runID := fmt.Sprintf("run_%s_%s_%s_%s_%d", target.Harness, arm, safePathFragment(variant), task.Metadata.TaskID, time.Now().UnixNano())
	workDir, err := createBenchmarkWorkspace(r.fixtureRoot, runID)
	if err != nil {
		return nil, err
	}
	session, sessionErr := newAgentSession(selectedAdapter, r.baseScratchDir, r.fixtureRoot, runID, workDir)
	if sessionErr != nil {
		// #nosec G703 -- cleanup isolated workspace when session initialization fails.
		return nil, errors.Join(sessionErr, os.RemoveAll(workDir))
	}
	session.execution = execution
	session.runner = r
	defer func() {
		if closeErr := session.close(); closeErr != nil {
			if session.result != nil {
				session.result.Success = false
				session.result.Error = appendResultError(session.result.Error, closeErr)
			}
			executeErr = joinDistinctErrors(executeErr, closeErr)
		}
	}()
	workspace, workspaceErr := prepareAgentWorkspace(ctx, r, task, target, arm, variant, workDir, session)
	if workspaceErr != nil {
		return nil, workspaceErr
	}
	mcpInstructions := MCPServerInstructionsNone
	if target.Harness == string(HarnessCodex) || target.Harness == string(HarnessOpenCode) || target.Harness == string(HarnessCopilot) {
		mcpInstructions = r.mcpServerInstructionsMode
	}
	res := newAgentRunResult(r, task, agentRunResultInput{
		target: target, arm: arm, variant: variant, execution: execution, session: session, workspace: workspace, mcpInstructions: mcpInstructions,
	})
	session.result = res
	if _, _, local := localRelayJobModel(target); local && r.localModelLimits != nil && r.localModelLimits.Context > 0 {
		ensureProvenance(res)
		res.Provenance["local_model_context_tokens"] = fmt.Sprint(r.localModelLimits.Context)
		res.Provenance["local_model_output_tokens"] = fmt.Sprint(r.localModelLimits.Output)
		res.Provenance["local_model_limits_source"] = "llama.cpp-props"
	}
	prompt, err := prepareAgentRunMetadata(execution, session, res, r.semeditPrewarmVerify)
	if err != nil {
		return nil, err
	}
	session.execute(ctx, prompt)
	return res, session.executionErr
}

type agentWorkspaceSetup struct {
	prewarmApplied  bool
	prewarmDuration time.Duration
}

func prepareAgentWorkspace(ctx context.Context, runner *Runner, task *Task, target Target, arm ArmType, variant, workDir string, session *agentSession) (agentWorkspaceSetup, error) {
	if task.Metadata.Repository.URL != "" {
		if err := hydrateExternalRepository(ctx, task, runner.baseScratchDir, workDir); err != nil {
			return agentWorkspaceSetup{}, fmt.Errorf("hydrate external repository fixture: %w", err)
		}
	}
	if err := task.ExtractVariantTo(workDir, variant); err != nil {
		return agentWorkspaceSetup{}, fmt.Errorf("extract task fixture: %w", err)
	}
	if runner.moduleCache != "" {
		if err := prepareSharedGoDependencies(ctx, task, workDir); err != nil {
			return agentWorkspaceSetup{}, err
		}
	}
	if session.execution.SemeditSkill {
		if err := installSemeditSkill(workDir, runner.baseScratchDir); err != nil {
			return agentWorkspaceSetup{}, fmt.Errorf("install semedit benchmark skill: %w", err)
		}
	}
	if err := session.prepareAdapter(ctx, target, arm, task.Metadata.Repository.URL != ""); err != nil {
		return agentWorkspaceSetup{}, err
	}

	setup := agentWorkspaceSetup{}
	if isSemeditArm(arm) && runner.semeditPrewarmVerify {
		file, targetErr := fixtureGoWarmupFile(ctx, task, workDir)
		if targetErr != nil {
			return agentWorkspaceSetup{}, targetErr
		}
		if file != "" {
			binaryPath := filepath.Join(filepath.Dir(filepath.Dir(runner.baseScratchDir)), "bin", "semedit-next")
			var warmupErr error
			setup.prewarmDuration, warmupErr = runSemeditVerifyWarmup(ctx, workDir, file, binaryPath)
			if warmupErr != nil {
				return agentWorkspaceSetup{}, warmupErr
			}
			setup.prewarmApplied = true
		}
	}
	if err := prepareFixtureGoDependencies(ctx, workDir); err != nil {
		return agentWorkspaceSetup{}, err
	}
	if err := session.warmupAdapter(ctx); err != nil {
		return agentWorkspaceSetup{}, err
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

func installSemeditSkill(workDir, scratchRoot string) error {
	repositoryRoot := filepath.Dir(filepath.Dir(scratchRoot))
	sourcePath := filepath.Join(repositoryRoot, "skills", "semedit", "SKILL.md")
	// #nosec G304 -- the source is the repository-owned semedit skill.
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source skill %s: %w", sourcePath, err)
	}
	return prepareRuntimeLayout(workDir, nil, map[string][]byte{filepath.Join(".agents", "skills", "semedit", "SKILL.md"): content})
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
		ExecutionMode: "agent", SemeditSkill: input.execution.SemeditSkill,
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
	result.Provenance["fixture_dependency_preparation"] = "go-list-readonly-deps-test"
	if result.Provenance["module_cache"] != "" {
		result.Provenance["module_cache_preparation"] = "go-mod-download-all-before-offline-execution"
	}
	result.Provenance["session_transcript"] = session.transcriptPath
	result.Provenance["adapter_stage_transcript"] = session.stageTranscriptPath
	if transport := providerContinuationTransport(execution.Target); transport != "" {
		result.Provenance["provider_transport"] = transport
	}
	if result.SemeditPrewarmVerify {
		result.Provenance["semedit_prewarm"] = "verify_cli_check_only"
		result.Provenance["go_cache_initial_state"] = "empty"
	} else if isSemeditArm(execution.Arm) && prewarmConfigured {
		result.Provenance["semedit_prewarm_skipped"] = ("no_go_target")
	}
	return execution.Prompt, nil
}

func ensureProvenance(result *RunResult) {
	if result.Provenance == nil {
		result.Provenance = make(ProvenanceSet)
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
	if arm == ArmSemeditExplicit {
		explicitInstruction, ok := task.Metadata.PromptVariants["explicit"]
		if !ok || strings.TrimSpace(explicitInstruction) == "" {
			return "", fmt.Errorf("explicit semedit prompt variant is required and must be nonblank")
		}
		baseInstruction = explicitInstruction
	} else if promptVarName != "" {
		promptName := promptVarName
		if promptName == "implicit" || promptName == "explicit" {
			promptName = "default"
		}
		if customInstruction, ok := task.Metadata.PromptVariants[promptName]; ok {
			baseInstruction = customInstruction
		}
	}
	res.PromptVariant = promptVarName
	if task.Metadata.Repository.URL != "" {
		if arm == ArmBaseline {
			return baseInstruction, nil
		}
		if arm != ArmSemedit && arm != ArmSemeditAgentic && arm != ArmSemeditExplicit {
			return "", fmt.Errorf("unsupported external-repository task arm %s", arm)
		}
		steering, err := semeditRestrictionSteering(effectivePolicy)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s When done, output DONE.", strings.TrimSpace(baseInstruction), steering), nil
	}
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

const semanticToolReflectionPrompt = "The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task."

const semanticBatchReflectionPrompt = "The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task."

const semanticIntegrationReflectionPrompt = "The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit's tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit's tools or not. This sentence will be the first thing a potential new user sees about semedit's usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits."

func shouldRequestSemanticToolReflection(arm ArmType, sessionID string, result *RunResult) bool {
	return isSemeditArm(arm) && sessionID != "" && result != nil && result.Success && result.Error == "" && !result.MCPVerified
}

func shouldRequestSemanticBatchReflection(arm ArmType, sessionID string, result *RunResult) bool {
	return isSemeditArm(arm) && sessionID != "" && result != nil && result.Success && result.Error == "" && hasConsecutiveUnbatchedSemanticToolCalls(result.ToolCalls)
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
	for _, prefix := range []string{"semedit/", "semedit-", "mcp__semedit__"} {
		if after, found := strings.CutPrefix(clean, prefix); found {
			return after
		}
	}
	if after, found := strings.CutPrefix(clean, "semedit_semantic_"); found {
		return "semantic_" + after
	}
	return clean
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
			if entry.Name() == ".git" || entry.Name() == ".scratch" || sourcefiles.IsBenchmarkDataDirectory(path) {
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

func prepareAgentStorageContext(ctx context.Context, runner *Runner) (context.Context, error) {
	if err := runner.validateFixtureRoot(); err != nil {
		return nil, err
	}
	if runner.moduleCache != "" {
		ctx = gocache.WithModuleCache(ctx, runner.moduleCache)
	}
	return ctx, nil
}
