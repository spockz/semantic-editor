// Package main defines resolved benchmark plans and provider-neutral execution scheduling.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type JobExecutor func(context.Context, Job) (*RunResult, error)

type JobOutcome struct {
	Job    Job
	Result *RunResult
	Err    error
}

type BenchmarkPlan struct {
	Jobs       []Job
	Exclusions []FixtureExclusion
	Options    PlanOptions
}

type FixtureExclusion struct {
	TaskID    string
	Path      string
	Reason    string
	Requested []string
	Available []string
}

type Job struct {
	ID                    string
	PairID                string
	TaskID                string
	FixturePath           string
	Context               string
	PromptVariant         string
	Target                Target
	Arm                   ArmType
	Repeat                int
	Policy                SemeditArmRestriction
	Task                  *Task
	Execution             AgentExecution
	MCPServerInstructions MCPServerInstructionMode
	SemeditSkill          bool
}

type PlanOptions struct {
	BenchDir              string
	TaskIDs               []string
	Targets               []Target
	Variants              []string
	Repeats               int
	Concurrency           int
	Timeout               time.Duration
	OutDir                string
	RunID                 string
	Resume                bool
	SemeditSkill          bool
	OutJSON               string
	OutMD                 string
	MCPServerInstructions []MCPServerInstructionMode
	SemeditArmRestriction SemeditArmRestriction
	SemeditPrewarmVerify  bool
	RAMFixtureRoot        string
	ModuleCache           string
	Provenance            ProvenanceSet
}

func BuildBenchmarkPlan(options PlanOptions) (*BenchmarkPlan, error) {
	options, err := normalizePlanOptions(options)
	if err != nil {
		return nil, err
	}
	fixtures, err := discoverPlanFixtures(options.BenchDir, options.TaskIDs)
	if err != nil {
		return nil, err
	}
	if len(fixtures) == 0 {
		return nil, fmt.Errorf("no benchmark fixtures selected")
	}
	plan := &BenchmarkPlan{Options: options}
	seenJobs := make(map[string]bool)
	for _, fixture := range fixtures {
		if err := expandFixtureJobs(plan, fixture, seenJobs); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func RenderBenchmarkPlan(w io.Writer, plan *BenchmarkPlan) error {
	if plan == nil {
		return fmt.Errorf("benchmark plan is nil")
	}
	if w == nil {
		return fmt.Errorf("benchmark plan output writer is nil")
	}
	if _, err := fmt.Fprintf(w, "Resolved benchmark plan: %d jobs\n", len(plan.Jobs)); err != nil {
		return err
	}
	options := plan.Options
	if _, err := fmt.Fprintf(w, "Settings: timeout=%s concurrency=%d repeats=%d MCP-instructions=%s semedit-arm-restrict=%s semedit-skill=%t semedit-prewarm-verify=%t out-dir=%q run-id=%q resume=%t out-json=%q out-md=%q\n", options.Timeout, options.Concurrency, options.Repeats, strings.Join(mcpInstructionModeStrings(options.MCPServerInstructions), ","), options.SemeditArmRestriction, options.SemeditSkill, options.SemeditPrewarmVerify, options.OutDir, options.RunID, options.Resume, options.OutJSON, options.OutMD); err != nil {
		return err
	}
	for index, job := range plan.Jobs {
		promptVariant := emptyAsDash(job.PromptVariant)
		policy := string(options.SemeditArmRestriction)
		applies := isSemeditArm(job.Arm)
		if job.Arm == ArmControl {
			policy = "-"
			applies = false
		}
		if _, err := fmt.Fprintf(w, "  %d. %s pair=%s task=%s fixture=%s target=%s arm=%s context=%s prompt-variant=%s repeat=%d MCP-instructions=%s policy=%s policy-applies=%t semedit-skill=%t\n", index+1, job.ID, emptyAsDash(job.PairID), job.TaskID, job.FixturePath, job.Target.String(), job.Arm, job.Context, promptVariant, job.Repeat, job.MCPServerInstructions, policy, applies, job.SemeditSkill); err != nil {
			return err
		}
		if job.Execution.Prompt != "" {
			if _, err := fmt.Fprintf(w, "     resolved-prompt=%q\n", job.Execution.Prompt); err != nil {
				return err
			}
		}
		for followupIndex, followup := range job.Execution.Followups {
			if _, err := fmt.Fprintf(w, "     followup-%d=%q\n", followupIndex+1, followup); err != nil {
				return err
			}
		}
		for stageIndex, stage := range job.Execution.StagedFollowups {
			if _, err := fmt.Fprintf(w, "     staged-followup-%d total-edits=%d prompt=%q\n", stageIndex+1, stage.TotalEdits, stage.Instruction); err != nil {
				return err
			}
		}
	}
	for _, exclusion := range plan.Exclusions {
		if _, err := fmt.Fprintf(w, "Excluded fixture: task=%s path=%s requested=%s available=%s reason=%s\n", exclusion.TaskID, exclusion.Path, strings.Join(exclusion.Requested, ","), strings.Join(exclusion.Available, ","), exclusion.Reason); err != nil {
			return err
		}
	}
	if len(plan.Jobs) == 0 {
		if _, err := fmt.Fprintln(w, "No benchmark jobs are scheduled."); err != nil {
			return err
		}
	}
	return nil
}

func ExecutePlan(ctx context.Context, plan *BenchmarkPlan, output io.Writer, execute JobExecutor) ([]JobOutcome, error) {
	return ExecutePlanWithCompletion(ctx, plan, output, execute, nil)
}

func ExecutePlanWithCompletion(ctx context.Context, plan *BenchmarkPlan, output io.Writer, execute JobExecutor, onComplete func(JobOutcome) error) ([]JobOutcome, error) {
	if plan == nil || len(plan.Jobs) == 0 {
		return nil, nil
	}
	if output == nil {
		output = io.Discard
	}
	concurrency := min(max(plan.Options.Concurrency, 1), len(plan.Jobs))
	completed := runPlanWorkers(ctx, plan, concurrency, execute)
	return collectPlanOutcomesWithCompletion(output, plan, completed, onComplete)
}

type indexedPlanOutcome struct {
	index   int
	outcome JobOutcome
}

func runPlanWorkers(ctx context.Context, plan *BenchmarkPlan, concurrency int, execute JobExecutor) <-chan indexedPlanOutcome {
	jobs := make(chan int, len(plan.Jobs))
	completed := make(chan indexedPlanOutcome, len(plan.Jobs))
	for index := range plan.Jobs {
		jobs <- index
	}
	close(jobs)
	var workers sync.WaitGroup
	for range concurrency {
		workers.Go(func() {
			for index := range jobs {
				job := plan.Jobs[index]
				completed <- indexedPlanOutcome{index: index, outcome: executePlanJob(ctx, plan, job, execute)}
			}
		})
	}
	go func() { workers.Wait(); close(completed) }()
	return completed
}

func executePlanJob(ctx context.Context, plan *BenchmarkPlan, job Job, execute JobExecutor) JobOutcome {
	var result *RunResult
	var err error
	switch {
	case ctx.Err() != nil:
		err = ctx.Err()
	case execute == nil:
		err = fmt.Errorf("job executor is nil")
	default:
		jobCtx, cancel := jobExecutionContext(ctx, plan.Options.Timeout, job)
		result, err = execute(jobCtx, job)
		cancel()
		if result == nil && err == nil {
			err = fmt.Errorf("job executor returned nil result without an error")
		}
	}
	if result == nil {
		result = &RunResult{}
	}
	populatePlanResult(plan, job, result, err)
	return JobOutcome{Job: job, Result: result, Err: err}
}

func jobExecutionContext(ctx context.Context, timeout time.Duration, job Job) (context.Context, context.CancelFunc) {
	if job.Arm == ArmControl {
		return context.WithTimeout(ctx, timeout)
	}
	return withAgentTaskTimeout(ctx, timeout), func() {}
}

func populatePlanResult(plan *BenchmarkPlan, job Job, result *RunResult, err error) {
	result.TaskID = job.TaskID
	if job.Task != nil && job.Task.Metadata.Repository.URL != "" {
		profile := job.Task.Metadata.Complexity
		result.Complexity = &profile
	}
	result.JobID = job.ID
	result.ComparisonPairID = job.PairID
	result.Variant = job.Context
	if job.PromptVariant != "" {
		result.Variant += ":" + job.PromptVariant
	}
	result.Repeat = job.Repeat
	result.PromptVariant = job.PromptVariant
	result.Target = job.Target
	result.Arm = job.Arm
	result.SemeditSkill = job.SemeditSkill
	result.SemeditArmRestrict = plan.Options.SemeditArmRestriction
	result.SemeditArmRestrictionApplied = isSemeditArm(job.Arm)
	result.Prompt = job.Execution.Prompt
	result.MCPServerInstructions = job.MCPServerInstructions
	if job.Arm == ArmControl {
		result.SemeditArmRestrict = ""
		result.SemeditArmRestrictionApplied = false
		result.Prompt = ""
	}
	if result.Provenance == nil {
		result.Provenance = make(ProvenanceSet)
	}
	for key, value := range plan.Options.Provenance {
		if _, exists := result.Provenance[key]; !exists {
			result.Provenance[key] = value
		}
	}
	if err != nil {
		result.Success = false
		result.Error = appendResultError(result.Error, err)
	} else if result.Error != "" {
		result.Success = false
	}
}

func writePlanProgress(output io.Writer, plan *BenchmarkPlan, done int, outcome JobOutcome) error {
	status := "PASS"
	if outcome.Err != nil {
		if errors.Is(outcome.Err, context.Canceled) || errors.Is(outcome.Err, context.DeadlineExceeded) {
			status = "CANCELLED"
		} else {
			status = "ERROR"
		}
	} else if !outcome.Result.Success {
		status = "FAIL"
	}
	line := fmt.Sprintf("[%d/%d completed · %d remaining] %s run_id=%s job=%s pair=%s task=%s target=%s arm=%s", done, len(plan.Jobs), len(plan.Jobs)-done, status, emptyAsDash(plan.Options.RunID), outcome.Job.ID, emptyAsDash(outcome.Job.PairID), outcome.Job.TaskID, outcome.Job.Target.String(), outcome.Job.Arm)
	if outcome.Err != nil {
		line += " error=" + compactBenchmarkProgressError(outcome.Err.Error())
	} else if outcome.Result.Error != "" {
		line += " error=" + compactBenchmarkProgressError(outcome.Result.Error)
	}
	_, err := fmt.Fprintln(output, line)
	return err
}

type variantSelection struct {
	Context        string
	PromptName     string
	ExplicitPrompt bool
}

func normalizeVariantSelections(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"small", "large"}, nil
	}
	var variants []string
	seen := make(map[string]bool)
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			contextName, promptName, compound := strings.Cut(part, ":")
			contextName = strings.ToLower(strings.TrimSpace(contextName))
			promptName = strings.TrimSpace(promptName)
			baseContext, validContext := normalizedContextBase(contextName)
			if !validContext {
				return nil, fmt.Errorf("invalid context variant %q (want small or large, optionally verified)", part)
			}
			if strings.HasSuffix(contextName, "+verified") || strings.HasSuffix(contextName, "-verified") || strings.HasSuffix(contextName, "_verified") {
				contextName = baseContext + "+verified"
			} else {
				contextName = baseContext
			}
			if strings.Contains(promptName, ":") {
				return nil, fmt.Errorf("invalid compound variant %q: multiple prompt separators", part)
			}
			if compound && promptName == "" {
				return nil, fmt.Errorf("invalid compound variant %q: prompt name is empty", part)
			}
			part = contextName
			if compound {
				part += ":" + promptName
			}
			if seen[part] {
				continue
			}
			seen[part] = true
			variants = append(variants, part)
		}
	}
	if len(variants) == 0 {
		return nil, fmt.Errorf("at least one context variant is required")
	}
	return variants, nil
}

type fixtureSpec struct {
	Path string
	Task *Task
}

func selectFixtureVariants(requested, available []string) ([]variantSelection, []string) {
	availableSet := make(map[string]bool, len(available))
	for _, name := range available {
		availableSet[name] = true
	}
	var selected []variantSelection
	var invalid []string
	for _, raw := range requested {
		contextName, promptName, explicitPrompt := strings.Cut(raw, ":")
		baseContext, valid := normalizedContextBase(contextName)
		if !valid || !availableSet[baseContext] {
			invalid = append(invalid, contextName)
			continue
		}
		selected = append(selected, variantSelection{Context: contextName, PromptName: promptName, ExplicitPrompt: explicitPrompt})
	}
	return selected, slices.Compact(invalid)
}

func stablePlanID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return parts[0] + "-" + hex.EncodeToString(sum[:8])
}

func supportedTaskContexts(task *Task) []string {
	if task == nil {
		return nil
	}
	contexts := task.Metadata.Contexts
	if len(contexts) == 0 {
		contexts = []string{"small"}
	}
	out := make([]string, 0, len(contexts))
	for _, value := range contexts {
		out = append(out, strings.ToLower(strings.TrimSpace(value)))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func discoverPlanFixtures(benchDir string, taskIDs []string) ([]fixtureSpec, error) {
	byID := make(map[string]fixtureSpec)
	for _, source := range fixtureDirectories(benchDir) {
		if err := addFixtureDirectory(byID, source); err != nil {
			return nil, err
		}
	}
	return selectPlanFixtures(byID, taskIDs)
}

type fixtureDirectory struct {
	path       string
	optional   bool
	skipBadTxt bool
}

func fixtureDirectories(benchDir string) []fixtureDirectory {
	directories := []fixtureDirectory{{path: benchDir}}
	scriptsDir := filepath.Join(filepath.Dir(benchDir), "scripts")
	if filepath.Clean(scriptsDir) != filepath.Clean(benchDir) {
		directories = append(directories, fixtureDirectory{path: scriptsDir, optional: true, skipBadTxt: true})
	}
	return directories
}

func addFixtureDirectory(byID map[string]fixtureSpec, source fixtureDirectory) error {
	entries, err := os.ReadDir(source.path)
	if err != nil {
		if source.optional && os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read fixture directory %s: %w", source.path, err)
	}
	for _, entry := range entries {
		if err := addFixtureEntry(byID, source, entry); err != nil {
			return err
		}
	}
	return nil
}

func addFixtureEntry(byID map[string]fixtureSpec, source fixtureDirectory, entry os.DirEntry) error {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
		return nil
	}
	path := filepath.Join(source.path, entry.Name())
	// #nosec G304 -- path is selected from the configured benchmark directories by os.ReadDir.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read fixture %s: %w", path, err)
	}
	task, err := ParseTask(data)
	if err != nil {
		if source.skipBadTxt {
			return nil
		}
		return fmt.Errorf("parse fixture %s: %w", path, err)
	}
	if _, exists := byID[task.Metadata.TaskID]; !exists {
		byID[task.Metadata.TaskID] = fixtureSpec{Path: path, Task: task}
	}
	return nil
}

func selectPlanFixtures(byID map[string]fixtureSpec, taskIDs []string) ([]fixtureSpec, error) {
	ids := taskIDs
	if len(ids) == 0 || slices.Contains(ids, "all") || slices.Contains(ids, "*") {
		ids = slices.Sorted(maps.Keys(byID))
	}
	selected := make([]fixtureSpec, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, requested := range ids {
		requested = strings.TrimSpace(requested)
		var found *fixtureSpec
		for taskID, fixture := range byID {
			if taskID == requested || strings.TrimSuffix(filepath.Base(fixture.Path), ".txtar") == strings.ReplaceAll(requested, "-", "_") {
				copy := fixture
				found = &copy
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("benchmark fixture %q was not found", requested)
		}
		if !seen[found.Task.Metadata.TaskID] {
			selected = append(selected, *found)
			seen[found.Task.Metadata.TaskID] = true
		}
	}
	slices.SortFunc(selected, func(a, b fixtureSpec) int { return strings.Compare(a.Task.Metadata.TaskID, b.Task.Metadata.TaskID) })
	return selected, nil
}

func validatePlanTargets(targets []Target) error {
	if len(targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	seen := make(map[string]bool, len(targets))
	for _, target := range targets {
		target.Harness = strings.ToLower(strings.TrimSpace(target.Harness))
		switch target.Harness {
		case string(HarnessControl), string(HarnessCodex), string(HarnessAgy), string(HarnessOpenCode):
		default:
			return fmt.Errorf("unsupported target harness %q", target.Harness)
		}
		if target.Harness == string(HarnessControl) && (target.Model != "" || target.Effort != "") {
			return fmt.Errorf("control target does not accept a model or effort")
		}
		key := target.String()
		if seen[key] {
			return fmt.Errorf("duplicate target %q", key)
		}
		seen[key] = true
	}
	return nil
}

func emptyAsDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func validateTaskContexts(task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	seen := make(map[string]bool, len(task.Metadata.Contexts))
	for _, raw := range task.Metadata.Contexts {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value != "small" && value != "large" {
			return fmt.Errorf("unsupported context %q (want small or large)", raw)
		}
		if seen[value] {
			return fmt.Errorf("duplicate context %q", value)
		}
		seen[value] = true
	}
	return nil
}

func normalizedContextBase(contextName string) (string, bool) {
	base := contextName
	switch {
	case strings.HasSuffix(base, "+verified"):
		base = strings.TrimSuffix(base, "+verified")
	case strings.HasSuffix(base, "-verified"):
		base = strings.TrimSuffix(base, "-verified")
	case strings.HasSuffix(base, "_verified"):
		base = strings.TrimSuffix(base, "_verified")
	}
	return base, base == "small" || base == "large"
}

func normalizePlanOptions(options PlanOptions) (PlanOptions, error) {
	if options.BenchDir == "" {
		options.BenchDir = "testdata/bench"
	}
	if options.Repeats == 0 {
		options.Repeats = 1
	}
	if options.Repeats < 1 {
		return PlanOptions{}, fmt.Errorf("repeats must be at least 1")
	}
	if options.Concurrency == 0 {
		options.Concurrency = 4
	}
	if options.Concurrency < 1 {
		return PlanOptions{}, fmt.Errorf("concurrency must be at least 1")
	}
	if options.Timeout == 0 {
		options.Timeout = 5 * time.Minute
	}
	if options.Timeout < 0 {
		return PlanOptions{}, fmt.Errorf("timeout must be positive")
	}
	if options.OutDir != "" {
		if err := validateBenchmarkRunID(options.RunID); err != nil {
			return PlanOptions{}, err
		}
	}
	if options.SemeditArmRestriction == "" {
		options.SemeditArmRestriction = SemeditArmRestrictWrite
	}
	policy, err := ParseSemeditArmRestriction(string(options.SemeditArmRestriction))
	if err != nil {
		return PlanOptions{}, fmt.Errorf("invalid semedit arm restriction: %w", err)
	}
	options.SemeditArmRestriction = policy
	if len(options.MCPServerInstructions) == 0 {
		options.MCPServerInstructions = []MCPServerInstructionMode{MCPServerInstructionsPrescriptive}
	} else {
		modes, parseModesErr := ParseMCPServerInstructionModes(strings.Join(mcpInstructionModeStrings(options.MCPServerInstructions), ","))
		if parseModesErr != nil {
			return PlanOptions{}, fmt.Errorf("invalid MCP server instruction modes: %w", parseModesErr)
		}
		options.MCPServerInstructions = modes
	}
	options.Variants, err = normalizeVariantSelections(options.Variants)
	if err != nil {
		return PlanOptions{}, err
	}
	if len(options.Targets) == 0 {
		options.Targets = []Target{{Harness: string(HarnessControl)}}
	}
	options.Targets = expandOpenRouterFreeTargets(options.Targets)
	if err := validatePlanTargets(options.Targets); err != nil {
		return PlanOptions{}, err
	}
	return options, nil
}

func expandFixtureJobs(plan *BenchmarkPlan, fixture fixtureSpec, seenJobs map[string]bool) error {
	task := fixture.Task
	if err := validateTaskContexts(task); err != nil {
		return fmt.Errorf("fixture %s contexts: %w", fixture.Path, err)
	}
	taskID := task.Metadata.TaskID
	available := supportedTaskContexts(task)
	selectedVariants, invalidContexts := selectFixtureVariants(plan.Options.Variants, available)
	if len(invalidContexts) > 0 {
		plan.Exclusions = append(plan.Exclusions, FixtureExclusion{TaskID: taskID, Path: fixture.Path, Reason: fmt.Sprintf("requested contexts %s are unavailable; available contexts: %s", strings.Join(invalidContexts, ","), strings.Join(available, ",")), Requested: invalidContexts, Available: available})
	}
	if len(selectedVariants) == 0 {
		return nil
	}
	promptNames := declaredPromptVariants(task)
	hasAgentTarget := false
	for _, target := range plan.Options.Targets {
		hasAgentTarget = hasAgentTarget || target.Harness != string(HarnessControl)
	}
	if hasAgentTarget && len(promptNames) == 0 {
		plan.Exclusions = append(plan.Exclusions, FixtureExclusion{TaskID: taskID, Path: fixture.Path, Reason: "fixture declares no non-empty prompt variants", Requested: append([]string(nil), plan.Options.Variants...)})
	}
	for _, selected := range selectedVariants {
		for _, target := range plan.Options.Targets {
			if err := appendTargetJobs(plan, fixture, selected, target, promptNames, seenJobs); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendTargetJobs(plan *BenchmarkPlan, fixture fixtureSpec, selected variantSelection, target Target, promptNames []string, seenJobs map[string]bool) error {
	isControl := target.Harness == string(HarnessControl)
	if !isControl && len(promptNames) == 0 {
		return nil
	}
	targetPrompts := []string{""}
	if !isControl {
		if selected.ExplicitPrompt {
			if !slices.Contains(promptNames, selected.PromptName) {
				return fmt.Errorf("fixture %s does not declare prompt variant %q", fixture.Task.Metadata.TaskID, selected.PromptName)
			}
			targetPrompts = []string{selected.PromptName}
		} else {
			targetPrompts = make([]string, 0, len(promptNames))
			for _, promptName := range promptNames {
				if promptName != "explicit" {
					targetPrompts = append(targetPrompts, promptName)
				}
			}
		}
	}
	for _, promptName := range targetPrompts {
		if !isControl && promptName == "explicit" {
			if err := appendExplicitPromptJobs(plan, fixture, selected, target, seenJobs); err != nil {
				return err
			}
			continue
		}
		if err := appendPromptJobs(plan, fixture, selected, target, promptName, seenJobs); err != nil {
			return err
		}
	}
	return nil
}

func appendExplicitPromptJobs(plan *BenchmarkPlan, fixture fixtureSpec, selected variantSelection, target Target, seenJobs map[string]bool) error {
	for repeat := 1; repeat <= plan.Options.Repeats; repeat++ {
		specification := Job{TaskID: fixture.Task.Metadata.TaskID, FixturePath: fixture.Path, Context: selected.Context, Target: target, Repeat: repeat, Task: fixture.Task}
		jobs, err := makeExplicitPromptJobs(specification, plan.Options)
		if err != nil {
			return fmt.Errorf("resolve explicit prompt experiment for fixture %s: %w", fixture.Task.Metadata.TaskID, err)
		}
		for _, job := range jobs {
			if seenJobs[job.ID] {
				return fmt.Errorf("duplicate resolved benchmark job %s; remove overlapping context or prompt variants", job.ID)
			}
			seenJobs[job.ID] = true
			plan.Jobs = append(plan.Jobs, job)
		}
	}
	return nil
}

func makeExplicitPromptJobs(specification Job, options PlanOptions) ([]Job, error) {
	if _, ok := specification.Task.Metadata.PromptVariants["explicit"]; !ok {
		return nil, fmt.Errorf("fixture %s has no explicit prompt", specification.Task.Metadata.TaskID)
	}
	modes := options.MCPServerInstructions
	if len(modes) == 0 {
		modes = []MCPServerInstructionMode{MCPServerInstructionsPrescriptive}
	}
	conditions := []struct {
		arm     ArmType
		variant string
	}{
		{arm: ArmBaseline},
		{arm: ArmSemedit, variant: "implicit"},
		{arm: ArmSemeditExplicit, variant: "explicit"},
	}
	jobs := make([]Job, 0, len(modes)*len(conditions))
	for _, mode := range modes {
		pairParts := []string{"pair", specification.TaskID, specification.Target.String(), specification.Context, fmt.Sprint(specification.Repeat), string(options.SemeditArmRestriction), string(mode), "semedit-prompt-experiment"}
		if options.SemeditPrewarmVerify {
			pairParts = append(pairParts, "prewarm-verify")
		}
		pairParts = append(pairParts, fmt.Sprintf("semedit-skill=%t", options.SemeditSkill))
		pairID := stablePlanID(pairParts...)
		for offset := range conditions {
			condition := conditions[(offset+specification.Repeat-1)%len(conditions)]
			job := specification
			job.PromptVariant = condition.variant
			job.SemeditSkill = options.SemeditSkill && isSemeditArm(condition.arm)
			job.ID = stablePlanID("job", specification.TaskID, specification.Target.String(), string(condition.arm), specification.Context, condition.variant, fmt.Sprint(specification.Repeat), string(mode), string(options.SemeditArmRestriction), fmt.Sprintf("semedit-skill=%t", job.SemeditSkill))
			job.PairID = pairID
			job.Arm = condition.arm
			job.MCPServerInstructions = mode
			if isSemeditArm(condition.arm) {
				job.Policy = options.SemeditArmRestriction
			}
			variant := specification.Context
			if condition.variant != "" {
				variant += ":" + condition.variant
			}
			execution, err := ResolveAgentExecution(job.Task, job.Target, condition.arm, variant, options.SemeditArmRestriction)
			if err != nil {
				return nil, err
			}
			execution.SemeditSkill = job.SemeditSkill
			job.Execution = execution
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func appendPromptJobs(plan *BenchmarkPlan, fixture fixtureSpec, selected variantSelection, target Target, promptName string, seenJobs map[string]bool) error {
	task := fixture.Task
	for repeat := 1; repeat <= plan.Options.Repeats; repeat++ {
		specification := Job{TaskID: task.Metadata.TaskID, FixturePath: fixture.Path, Context: selected.Context, PromptVariant: promptName, Target: target, Repeat: repeat, Task: task}
		jobs, err := makePlannedJobs(specification, plan.Options)
		if err != nil {
			return fmt.Errorf("resolve job for fixture %s: %w", fixture.Path, err)
		}
		for _, job := range jobs {
			if seenJobs[job.ID] {
				return fmt.Errorf("duplicate resolved benchmark job %s; remove overlapping context or prompt variants", job.ID)
			}
			seenJobs[job.ID] = true
			plan.Jobs = append(plan.Jobs, job)
		}
	}
	return nil
}

func makePlannedJobs(specification Job, options PlanOptions) ([]Job, error) {
	if specification.Task.Metadata.Repository.URL != "" && specification.Target.Harness != string(HarnessControl) {
		return makeExternalRepositoryJobs(specification, options)
	}
	variant := specification.Context
	if specification.PromptVariant != "" {
		variant += ":" + specification.PromptVariant
	}
	isControl := specification.Target.Harness == string(HarnessControl)
	policy := options.SemeditArmRestriction
	modes := options.MCPServerInstructions
	if isControl {
		modes = []MCPServerInstructionMode{MCPServerInstructionsNone}
	}
	jobs := make([]Job, 0, len(modes)*2)
	for _, mode := range modes {
		arms := []ArmType{ArmControl}
		pairID := ""
		if !isControl {
			arms = []ArmType{ArmBaseline, ArmSemedit}
			pairParts := []string{"pair", specification.TaskID, specification.Target.String(), variant, fmt.Sprint(specification.Repeat), string(policy), string(mode), fmt.Sprintf("semedit-skill=%t", options.SemeditSkill)}
			if options.SemeditPrewarmVerify {
				pairParts = append(pairParts, "prewarm-verify")
			}
			pairID = stablePlanID(pairParts...)
		}
		for _, arm := range arms {
			job := specification
			job.SemeditSkill = options.SemeditSkill && isSemeditArm(arm)
			idParts := []string{"job", specification.TaskID, specification.Target.String(), string(arm), variant, fmt.Sprint(specification.Repeat), string(mode), fmt.Sprintf("semedit-skill=%t", job.SemeditSkill)}
			if arm != ArmControl {
				idParts = append(idParts, string(policy))
				if options.SemeditPrewarmVerify {
					idParts = append(idParts, "prewarm-verify")
				}
			}
			job.ID = stablePlanID(idParts...)
			job.PairID = pairID
			job.Arm = arm
			job.MCPServerInstructions = mode
			if isSemeditArm(arm) {
				job.Policy = policy
			}
			if arm != ArmControl {
				execution, err := ResolveAgentExecution(job.Task, job.Target, arm, variant, policy)
				if err != nil {
					return nil, err
				}
				execution.SemeditSkill = job.SemeditSkill
				job.Execution = execution
			}
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func makeExternalRepositoryJobs(specification Job, options PlanOptions) ([]Job, error) {
	variant := specification.Context
	if specification.PromptVariant != "" {
		variant += ":" + specification.PromptVariant
	}
	policy := options.SemeditArmRestriction
	skillEnabled := options.SemeditSkill
	modes := options.MCPServerInstructions
	if len(modes) == 0 {
		modes = []MCPServerInstructionMode{MCPServerInstructionsPrescriptive}
	}
	arms := []ArmType{ArmBaseline, ArmSemeditAgentic, ArmSemeditExplicit}
	jobs := make([]Job, 0, len(modes)*len(arms))
	for _, mode := range modes {
		pairParts := []string{"pair", specification.TaskID, specification.Target.String(), variant, fmt.Sprint(specification.Repeat), string(policy), string(mode), fmt.Sprintf("semedit-skill=%t", skillEnabled)}
		if options.SemeditPrewarmVerify {
			pairParts = append(pairParts, "prewarm-verify")
		}
		pairID := stablePlanID(pairParts...)
		for offset := range arms {
			arm := arms[(offset+specification.Repeat-1)%len(arms)]
			job := specification
			job.Arm = arm
			job.PairID = pairID
			job.MCPServerInstructions = mode
			job.PromptVariant = ""
			job.SemeditSkill = skillEnabled && isSemeditArm(arm)
			if isSemeditArm(arm) {
				job.Policy = policy
			}
			idParts := []string{"job", specification.TaskID, specification.Target.String(), string(arm), variant, fmt.Sprint(specification.Repeat), string(mode), fmt.Sprintf("semedit-skill=%t", job.SemeditSkill), string(policy)}
			if options.SemeditPrewarmVerify {
				idParts = append(idParts, "prewarm-verify")
			}
			executionVariant := specification.Context
			switch arm {
			case ArmSemeditAgentic:
				job.PromptVariant = "implicit"
				executionVariant += ":implicit"
			case ArmSemeditExplicit:
				job.PromptVariant = "explicit"
				executionVariant += ":explicit"
				idParts = append(idParts, "explicit-agent-execution-v1")
			case ArmBaseline, ArmSemedit, ArmControl:
			}
			job.ID = stablePlanID(idParts...)
			execution, err := ResolveAgentExecution(job.Task, job.Target, arm, executionVariant, policy)
			if err != nil {
				return nil, err
			}
			execution.SemeditSkill = job.SemeditSkill
			job.Execution = execution
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func compactBenchmarkProgressError(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	const maxLength = 512
	runes := []rune(message)
	if len(runes) > maxLength {
		return string(runes[:maxLength]) + "…"
	}
	return message
}

func mcpInstructionModeStrings(modes []MCPServerInstructionMode) []string {
	values := make([]string, len(modes))
	for index, mode := range modes {
		values[index] = string(mode)
	}
	return values
}

func collectPlanOutcomesWithCompletion(output io.Writer, plan *BenchmarkPlan, completed <-chan indexedPlanOutcome, onComplete func(JobOutcome) error) ([]JobOutcome, error) {
	ordered := make([]JobOutcome, len(plan.Jobs))
	var progressErr error
	done := 0
	for item := range completed {
		done++
		ordered[item.index] = item.outcome
		if onComplete != nil {
			if err := onComplete(item.outcome); err != nil {
				progressErr = errors.Join(progressErr, fmt.Errorf("persist job %s result: %w", item.outcome.Job.ID, err))
			}
		}
		if err := writePlanProgress(output, plan, done, item.outcome); err != nil {
			progressErr = errors.Join(progressErr, fmt.Errorf("write job progress: %w", err))
		}
	}
	return ordered, progressErr
}
