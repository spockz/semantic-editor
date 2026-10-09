package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"semedit/internal/pipeline"
)

// BenchmarkReport captures empirical comparison results across targets, tasks, variants, and arms.
type BenchmarkReport struct {
	FormatVersion int                  `json:"format_version"`
	DurationUnit  string               `json:"duration_unit"`
	Timestamp     time.Time            `json:"timestamp"`
	Runs          []*RunResult         `json:"runs"`
	Comparisons   []*ComparisonSummary `json:"comparisons,omitempty"`
}

const (
	benchmarkReportFormatVersion = 2
	benchmarkReportDurationUnit  = "milliseconds"
)

// MarshalJSON writes benchmark durations as milliseconds, matching their public JSON field names.
func (rep BenchmarkReport) MarshalJSON() ([]byte, error) {
	type reportAlias BenchmarkReport
	wire := reportAlias(rep)
	wire.FormatVersion = benchmarkReportFormatVersion
	wire.DurationUnit = benchmarkReportDurationUnit

	data, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if err := convertBenchmarkDurationsToMilliseconds(value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func convertBenchmarkDurationsToMilliseconds(value any) error {
	switch typed := value.(type) {
	case []any:
		return convertDurationChildren(typed)
	case map[string]any:
		return convertDurationFields(typed)
	default:
		return nil
	}
}

func durationNanosecondsToMilliseconds(raw any) (int64, error) {
	nanoseconds, ok := raw.(float64)
	if !ok || nanoseconds != float64(int64(nanoseconds)) {
		return 0, fmt.Errorf("expected integral duration, got %T (%v)", raw, raw)
	}
	return int64(nanoseconds) / int64(time.Millisecond), nil
}

func convertDurationChildren(items []any) error {
	for _, item := range items {
		if err := convertBenchmarkDurationsToMilliseconds(item); err != nil {
			return err
		}
	}
	return nil
}

func convertDurationFields(fields map[string]any) error {
	for _, value := range fields {
		if err := convertBenchmarkDurationsToMilliseconds(value); err != nil {
			return err
		}
	}
	if err := convertDurationMetricFields(fields); err != nil {
		return err
	}
	if _, isOracle := fields["level_1_policy"]; isOracle {
		return convertOracleDuration(fields)
	}
	return nil
}

func convertDurationMetricFields(fields map[string]any) error {
	for _, key := range []string{
		"wall_clock_ms",
		"process_start_to_first_event_ms",
		"first_event_to_first_tool_call_ms",
		"mcp_initialize_to_first_semantic_call_ms",
		"mcp_server_start_to_initialize_ms",
	} {
		if raw, exists := fields[key]; exists {
			milliseconds, err := durationNanosecondsToMilliseconds(raw)
			if err != nil {
				return fmt.Errorf("convert %s: %w", key, err)
			}
			fields[key] = milliseconds
		}
	}
	return nil
}

func convertOracleDuration(fields map[string]any) error {
	raw, exists := fields["duration_ms"]
	if !exists {
		return nil
	}
	milliseconds, err := durationNanosecondsToMilliseconds(raw)
	if err != nil {
		return fmt.Errorf("convert oracle duration_ms: %w", err)
	}
	fields["duration_ms"] = milliseconds
	return nil
}

// ComparisonSummary bundles Baseline vs MCP runs for a specific Task and Target across variants.
type ComparisonSummary struct {
	Complexity            *ComplexityProfile       `json:"complexity,omitempty"`
	SemeditArmRestrict    SemeditArmRestriction    `json:"semedit_arm_restrict,omitempty"`
	TaskID                string                   `json:"task_id"`
	Repeat                int                      `json:"repeat,omitempty"`
	PromptVariant         string                   `json:"prompt_variant,omitempty"`
	MCPServerInstructions MCPServerInstructionMode `json:"mcp_server_instructions,omitempty"`
	Provenance            ProvenanceSet            `json:"provenance,omitempty"`
	TxtarPath             string                   `json:"txtar_path,omitempty"`
	TxtarProvenance       string                   `json:"txtar_provenance,omitempty"`
	Target                Target                   `json:"target"`
	VanillaPrompt         string                   `json:"vanilla_prompt,omitempty"`
	MCPPrompt             string                   `json:"mcp_prompt,omitempty"`
	VanillaVerifiedPrompt string                   `json:"vanilla_verified_prompt,omitempty"`
	MCPVerifiedPrompt     string                   `json:"mcp_verified_prompt,omitempty"`
	BeforeState           string                   `json:"before_state,omitempty"`
	SmallBaseline         *RunResult               `json:"small_baseline,omitempty"`
	SmallSemedit          *RunResult               `json:"small_semedit,omitempty"`
	SmallVerifiedBaseline *RunResult               `json:"small_verified_baseline,omitempty"`
	SmallVerifiedSemedit  *RunResult               `json:"small_verified_semedit,omitempty"`
	SmallSemeditExplicit  *RunResult               `json:"small_semedit_explicit,omitempty"`
	SmallVerifiedExplicit *RunResult               `json:"small_verified_semedit_explicit,omitempty"`
	LargeBaseline         *RunResult               `json:"large_baseline,omitempty"`
	LargeSemedit          *RunResult               `json:"large_semedit,omitempty"`
	LargeVerifiedBaseline *RunResult               `json:"large_verified_baseline,omitempty"`
	LargeVerifiedSemedit  *RunResult               `json:"large_verified_semedit,omitempty"`
	LargeSemeditExplicit  *RunResult               `json:"large_semedit_explicit,omitempty"`
	LargeVerifiedExplicit *RunResult               `json:"large_verified_semedit_explicit,omitempty"`
}

var noSemanticToolsAvailablePattern = regexp.MustCompile(`(?i)\bno\b(?:\W+\w+)*\W+\btools?\b(?:\W+\w+)*\W+\bavailable\b`)

const benchmarkReportCSS = `<style>
table.benchmark-tool-calls {
  width: 100%;
  table-layout: fixed;
}

table.benchmark-tool-calls th:first-child,
table.benchmark-tool-calls td:first-child {
  width: 3rem;
}

table.benchmark-tool-calls td {
  min-width: 0;
}

table.benchmark-tool-calls pre.benchmark-shell-command,
table.benchmark-tool-calls pre.benchmark-tool-arguments {
  width: 100%;
  max-width: 32rem;
  margin: 0.5rem 0 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
  overflow-x: auto;
}

table.benchmark-tool-calls pre.benchmark-shell-command code {
  white-space: inherit;
}

.benchmark-delta-positive {
  color: var(--bs-success, #198754);
  font-weight: 700;
}

.benchmark-delta-negative {
  color: var(--bs-danger, #dc3545);
  font-weight: 700;
}
</style>

`

// normalizeTaskBase maps variant names (e.g. task-01b-rename-local-large-context) to their logical base name (task-01-rename-local).
func normalizeTaskBase(taskID string) string {
	base := strings.ReplaceAll(taskID, "_", "-")
	base = strings.Replace(base, "task-01b", "task-01", 1)
	base = strings.TrimSuffix(base, "-large-context")
	return base
}

// BuildComparisons pairs corresponding baseline and semedit runs into structured comparison sets.
func BuildComparisons(runs []*RunResult) []*ComparisonSummary {
	grouped := make(map[comparisonKey]*ComparisonSummary)
	sharedBaseline, sharedBaselineNeeded := indexSharedPromptBaselines(runs)
	for _, r := range runs {
		if r == nil || r.ExecutionMode == "ground_truth_fix_patch_replay" || isSharedPromptBaseline(r, sharedBaselineNeeded) {
			continue
		}
		baseTask := comparisonTaskIdentity(r)
		mcpServerInstructions := normalizeMCPServerInstructions(r.MCPServerInstructions)
		k := comparisonKey{baseTask: baseTask, target: r.Target.String(), repeat: r.Repeat, promptVariant: r.PromptVariant, mcpServerInstructions: mcpServerInstructions, semeditArmRestrict: r.SemeditArmRestrict}
		comp, exists := grouped[k]
		if !exists {
			comp = &ComparisonSummary{
				Complexity: r.Complexity, TaskID: baseTask, Repeat: k.repeat,
				PromptVariant: r.PromptVariant, MCPServerInstructions: mcpServerInstructions,
				SemeditArmRestrict: r.SemeditArmRestrict, Provenance: r.Provenance.Clone(),
				Target: r.Target, BeforeState: r.BeforeState,
			}
			grouped[k] = comp
		} else {
			comp.Provenance = commonProvenance(comp.Provenance, r.Provenance)
		}
		if comp.BeforeState == "" && r.BeforeState != "" {
			comp.BeforeState = r.BeforeState
		}
		assignComparisonRun(comp, r)
	}
	assignSharedPromptBaselines(grouped, sharedBaseline)
	result := make([]*ComparisonSummary, 0, len(grouped))
	for _, comp := range grouped {
		result = append(result, comp)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TaskID != result[j].TaskID {
			return result[i].TaskID < result[j].TaskID
		}
		if result[i].Target.String() != result[j].Target.String() {
			return result[i].Target.String() < result[j].Target.String()
		}
		if result[i].Repeat != result[j].Repeat {
			return result[i].Repeat < result[j].Repeat
		}
		if result[i].PromptVariant != result[j].PromptVariant {
			return result[i].PromptVariant < result[j].PromptVariant
		}
		if result[i].MCPServerInstructions != result[j].MCPServerInstructions {
			return result[i].MCPServerInstructions < result[j].MCPServerInstructions
		}
		return result[i].SemeditArmRestrict < result[j].SemeditArmRestrict
	})
	return result
}

type comparisonKey struct {
	baseTask              string
	target                string
	repeat                int
	promptVariant         string
	mcpServerInstructions MCPServerInstructionMode
	semeditArmRestrict    SemeditArmRestriction
}

func indexSharedPromptBaselines(runs []*RunResult) (map[comparisonBaselineKey][]*RunResult, map[comparisonBaselineKey]bool) {
	baselines := make(map[comparisonBaselineKey][]*RunResult)
	needed := make(map[comparisonBaselineKey]bool)
	for _, run := range runs {
		if run == nil || run.ExecutionMode == "ground_truth_fix_patch_replay" {
			continue
		}
		key := comparisonBaselineKeyFor(run)
		if isSemeditArm(run.Arm) && (run.PromptVariant == "implicit" || run.PromptVariant == "explicit") {
			needed[key] = true
		}
		if run.Arm == ArmBaseline && run.PromptVariant == "" {
			baselines[key] = append(baselines[key], run)
		}
	}
	return baselines, needed
}

func isSharedPromptBaseline(run *RunResult, needed map[comparisonBaselineKey]bool) bool {
	return run.Arm == ArmBaseline && run.PromptVariant == "" && needed[comparisonBaselineKeyFor(run)]
}

func assignSharedPromptBaselines(grouped map[comparisonKey]*ComparisonSummary, baselines map[comparisonBaselineKey][]*RunResult) {
	variants := []comparisonRunVariant{{}, {large: true}, {verified: true}, {large: true, verified: true}}
	for key, comparison := range grouped {
		if key.promptVariant != "implicit" && key.promptVariant != "explicit" {
			continue
		}
		for _, variant := range variants {
			if !comparisonHasSemeditVariant(comparison, variant) {
				continue
			}
			baselineKey := comparisonBaselineKey{task: comparison.TaskID, target: key.target, context: comparisonContextName(variant), verified: variant.verified, repeat: key.repeat, mcpServerInstructions: key.mcpServerInstructions, semeditArmRestrict: key.semeditArmRestrict}
			for _, baseline := range baselines[baselineKey] {
				assignBaselineComparisonRun(comparison, baseline, variant)
				comparison.Provenance = commonProvenance(comparison.Provenance, baseline.Provenance)
				if comparison.BeforeState == "" {
					comparison.BeforeState = baseline.BeforeState
				}
			}
		}
	}
}

type comparisonBaselineKey struct {
	task                  string
	target                string
	context               string
	verified              bool
	repeat                int
	mcpServerInstructions MCPServerInstructionMode
	semeditArmRestrict    SemeditArmRestriction
}

func comparisonBaselineKeyFor(run *RunResult) comparisonBaselineKey {
	variant := comparisonRunVariantFor(run)
	return comparisonBaselineKey{task: comparisonTaskIdentity(run), target: run.Target.String(), context: comparisonContextName(variant), verified: variant.verified, repeat: run.Repeat, mcpServerInstructions: normalizeMCPServerInstructions(run.MCPServerInstructions), semeditArmRestrict: run.SemeditArmRestrict}
}

func comparisonContextName(variant comparisonRunVariant) string {
	if variant.large {
		return "large"
	}
	return "small"
}

func comparisonHasSemeditVariant(comp *ComparisonSummary, variant comparisonRunVariant) bool {
	switch {
	case variant.large && variant.verified:
		return comp.LargeVerifiedSemedit != nil || comp.LargeVerifiedExplicit != nil
	case variant.large:
		return comp.LargeSemedit != nil || comp.LargeSemeditExplicit != nil
	case variant.verified:
		return comp.SmallVerifiedSemedit != nil || comp.SmallVerifiedExplicit != nil
	default:
		return comp.SmallSemedit != nil || comp.SmallSemeditExplicit != nil
	}
}

// commonProvenance retains only technical context shared by all runs in a comparison.
func commonProvenance(current, next ProvenanceSet) ProvenanceSet {
	if len(current) == 0 || len(next) == 0 {
		return nil
	}
	common := make(ProvenanceSet)
	for key, value := range current {
		if next[key] == value {
			common[key] = value
		}
	}
	return common
}

func normalizeMCPServerInstructions(mode MCPServerInstructionMode) MCPServerInstructionMode {
	switch mode {
	case "", MCPServerInstructionsNone:
		return MCPServerInstructionsNone
	case "directive":
		return MCPServerInstructionsPrescriptive
	default:
		return mode
	}
}

// ResolveTxtarProvenance resolves the provenance link for a benchmark fixture file.
// If committed and clean in git, it returns the commit-anchored GitHub link.
// If uncommitted or untracked, it returns a sha256 hash fallback string.
func ResolveTxtarProvenance(repoRoot, txtarRelPath string) string {
	fullPath := filepath.Join(repoRoot, filepath.FromSlash(txtarRelPath))
	// #nosec G304 -- reading benchmark fixture for provenance checksum
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	hashShort := fmt.Sprintf("sha256:%.12x (uncommitted)", hash)

	ctx := context.Background()
	// #nosec G204 -- inspecting git status of txtar fixture
	cmdStatus := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", txtarRelPath)
	cmdStatus.Dir = repoRoot
	outStatus, err := cmdStatus.Output()
	if err == nil && len(bytes.TrimSpace(outStatus)) == 0 {
		// #nosec G204 -- retrieving git commit hash of txtar fixture
		cmdLog := exec.CommandContext(ctx, "git", "log", "-n", "1", "--pretty=format:%H", "--", txtarRelPath)
		cmdLog.Dir = repoRoot
		outLog, err := cmdLog.Output()
		sha := strings.TrimSpace(string(outLog))
		if err == nil && len(sha) == 40 {
			return fmt.Sprintf("https://github.com/spockz/semantic-editor/blob/%s/%s", sha, filepath.ToSlash(txtarRelPath))
		}
	}

	return hashShort
}

// RenderMarkdown renders the structured side-by-side comparison tables.
func (rep *BenchmarkReport) RenderMarkdown() string {
	var sb strings.Builder
	sb.WriteString("# Empirical Benchmark Report: Vanilla LLM vs. Semedit MCP\n\n")
	sb.WriteString(benchmarkReportCSS)
	fmt.Fprintf(&sb, "* **Date**: %s\n\n", rep.Timestamp.Format("2006-01-02 15:04:05 MST"))

	currentTaskID := ""
	currentTarget := ""
	for _, comp := range rep.Comparisons {
		if comp.TaskID != currentTaskID {
			if currentTaskID != "" {
				sb.WriteString("---\n\n")
			}
			fmt.Fprintf(&sb, "## Test case: `%s`\n\n", comp.TaskID)
			currentTaskID = comp.TaskID
			currentTarget = ""
		}
		targetStr := comparisonTargetLabel(comp)
		if targetStr != currentTarget {
			fmt.Fprintf(&sb, "### Target: `%s`\n\n", targetStr)
			currentTarget = targetStr
		}
		renderComparisonMarkdown(&sb, comp)
	}

	return sb.String()
}

func (rep *BenchmarkReport) UnmarshalJSON(data []byte) error {
	if rep == nil {
		return fmt.Errorf("benchmark report receiver is nil")
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := restoreBenchmarkDurationsFromMilliseconds(value); err != nil {
		return err
	}
	restored, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("restore benchmark durations: %w", err)
	}
	type reportAlias BenchmarkReport
	var decoded reportAlias
	if err := json.Unmarshal(restored, &decoded); err != nil {
		return err
	}
	*rep = BenchmarkReport(decoded)
	return nil
}

func comparisonTargetLabel(comp *ComparisonSummary) string {
	target := comp.Target.String()
	if comp.Repeat > 0 {
		return fmt.Sprintf("%s (repeat %d)", target, comp.Repeat)
	}
	return target
}

func renderComparisonMarkdown(sb *strings.Builder, comp *ComparisonSummary) {
	renderComparisonIdentity(sb, comp)
	renderComparisonPrompts(sb, comp)
	renderComparisonState(sb, comp)
	renderStandardSections(sb, comp)
	renderVerifiedSections(sb, comp)
}

func renderComparisonIdentity(sb *strings.Builder, comp *ComparisonSummary) {
	fmt.Fprintf(sb, "#### Configuration: %s\n\n", formatConfiguration(comp))
	if provenance := comp.Provenance.String(); provenance != "" {
		fmt.Fprintf(sb, "* **Run Provenance**: `%s`\n\n", provenance)
	}
	switch {
	case comp.TxtarProvenance != "" && strings.HasPrefix(comp.TxtarProvenance, "http"):
		fmt.Fprintf(sb, "* **Fixture**: [%s](%s)\n\n", comp.TxtarPath, comp.TxtarProvenance)
	case comp.TxtarProvenance != "":
		fmt.Fprintf(sb, "* **Fixture**: `%s` (`%s`)\n\n", comp.TxtarPath, comp.TxtarProvenance)
	case comp.TxtarPath != "":
		fmt.Fprintf(sb, "* **Fixture**: `%s`\n\n", comp.TxtarPath)
	}
}

func renderComparisonPrompts(sb *strings.Builder, comp *ComparisonSummary) {
	if comp.VanillaPrompt != "" && comp.MCPPrompt != "" && comp.VanillaPrompt == comp.MCPPrompt {
		writePrompt(sb, "LLM Prompt", comp.VanillaPrompt)
	} else {
		if comp.VanillaPrompt != "" {
			writePrompt(sb, "Vanilla LLM Prompt", comp.VanillaPrompt)
		}
		if comp.MCPPrompt != "" {
			writePrompt(sb, "Semedit MCP Prompt", comp.MCPPrompt)
		}
	}
	if !hasVerifiedComparisons(comp) {
		return
	}
	if comp.VanillaVerifiedPrompt != "" && comp.MCPVerifiedPrompt != "" && comp.VanillaVerifiedPrompt == comp.MCPVerifiedPrompt {
		writePrompt(sb, "Verified Prompt", comp.VanillaVerifiedPrompt)
		return
	}
	if comp.VanillaVerifiedPrompt != "" {
		writePrompt(sb, "Vanilla (Verified) Prompt", comp.VanillaVerifiedPrompt)
	}
	if comp.MCPVerifiedPrompt != "" {
		writePrompt(sb, "Semedit MCP (Verified) Prompt", comp.MCPVerifiedPrompt)
	}
}

func renderComparisonState(sb *strings.Builder, comp *ComparisonSummary) {
	if comp.BeforeState == "" {
		return
	}
	sb.WriteString("<details><summary><b>Initial Workspace State (Before Edit)</b></summary>\n\n```go\n")
	sb.WriteString(comp.BeforeState)
	sb.WriteString("\n```\n</details>\n\n")
}

func hasVerifiedComparisons(comp *ComparisonSummary) bool {
	return comp.SmallVerifiedBaseline != nil || comp.SmallVerifiedSemedit != nil ||
		comp.LargeVerifiedBaseline != nil || comp.LargeVerifiedSemedit != nil || comp.SmallVerifiedExplicit != nil || comp.LargeVerifiedExplicit != nil
}

func renderStandardSections(sb *strings.Builder, comp *ComparisonSummary) {
	if comp.SmallBaseline == nil && comp.SmallSemedit == nil && comp.LargeBaseline == nil && comp.LargeSemedit == nil && comp.SmallSemeditExplicit == nil && comp.LargeSemeditExplicit == nil {
		return
	}
	if comp.SmallSemedit != nil || comp.LargeSemedit != nil {
		renderComparisonTable(sb, comp.SmallBaseline, comp.SmallSemedit, comp.LargeBaseline, comp.LargeSemedit)
		renderDiffSection(sb, 5, "Baseline vs Semedit Agentic in Small Context", comp.SmallBaseline, comp.SmallSemedit)
		renderDiffSection(sb, 5, "Baseline vs Semedit Agentic in Large Context", comp.LargeBaseline, comp.LargeSemedit)
	}
	if comp.SmallSemeditExplicit != nil || comp.LargeSemeditExplicit != nil {
		renderComparisonTable(sb, comp.SmallBaseline, comp.SmallSemeditExplicit, comp.LargeBaseline, comp.LargeSemeditExplicit)
		renderDiffSection(sb, 5, "Baseline vs Semedit Explicit in Small Context", comp.SmallBaseline, comp.SmallSemeditExplicit)
		renderDiffSection(sb, 5, "Baseline vs Semedit Explicit in Large Context", comp.LargeBaseline, comp.LargeSemeditExplicit)
	}
	renderSemanticToolReflection(sb, "Standard vs Semedit in Small Context", comp.SmallSemedit)
	renderSemanticToolReflection(sb, "Standard vs Semedit in Large Context", comp.LargeSemedit)
	renderSemanticBatchReflection(sb, "Standard vs Semedit in Small Context", comp.SmallSemedit)
	renderSemanticIntegrationReflection(sb, "Standard vs Semedit in Small Context", comp.SmallSemedit)
	renderSemanticBatchReflection(sb, "Standard vs Semedit in Large Context", comp.LargeSemedit)
	renderSemanticIntegrationReflection(sb, "Standard vs Semedit in Large Context", comp.LargeSemedit)
}

func renderVerifiedSections(sb *strings.Builder, comp *ComparisonSummary) {
	if !hasVerifiedComparisons(comp) {
		return
	}
	sb.WriteString("##### Verified Directive Comparison (+Self-Correction Loop)\n\n")
	if comp.SmallVerifiedSemedit != nil || comp.LargeVerifiedSemedit != nil {
		renderComparisonTable(sb, comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit, comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
		renderDiffSection(sb, 6, "Verified vs Semedit Agentic in Small Context", comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit)
		renderDiffSection(sb, 6, "Verified vs Semedit Agentic in Large Context", comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
	}
	if comp.SmallVerifiedExplicit != nil || comp.LargeVerifiedExplicit != nil {
		renderComparisonTable(sb, comp.SmallVerifiedBaseline, comp.SmallVerifiedExplicit, comp.LargeVerifiedBaseline, comp.LargeVerifiedExplicit)
		renderDiffSection(sb, 6, "Verified vs Semedit Explicit in Small Context", comp.SmallVerifiedBaseline, comp.SmallVerifiedExplicit)
		renderDiffSection(sb, 6, "Verified vs Semedit Explicit in Large Context", comp.LargeVerifiedBaseline, comp.LargeVerifiedExplicit)
	}
	renderSemanticToolReflection(sb, "Verified vs Semedit in Small Context", comp.SmallVerifiedSemedit)
	renderSemanticToolReflection(sb, "Verified vs Semedit in Large Context", comp.LargeVerifiedSemedit)
	renderSemanticBatchReflection(sb, "Verified vs Semedit in Small Context", comp.SmallVerifiedSemedit)
	renderSemanticIntegrationReflection(sb, "Verified vs Semedit in Small Context", comp.SmallVerifiedSemedit)
	renderSemanticBatchReflection(sb, "Verified vs Semedit in Large Context", comp.LargeVerifiedSemedit)
	renderSemanticIntegrationReflection(sb, "Verified vs Semedit in Large Context", comp.LargeVerifiedSemedit)
}

func writePrompt(sb *strings.Builder, label, prompt string) {
	fmt.Fprintf(sb, "**%s**:\n%s\n\n", label, formatPromptBlockquote(prompt))
}

func formatPromptBlockquote(prompt string) string {
	lines := strings.Split(prompt, "\n")
	for index, line := range lines {
		if line == "" {
			lines[index] = ">"
			continue
		}
		lines[index] = "> " + line
	}
	return strings.Join(lines, "\n")
}

func formatConfiguration(comp *ComparisonSummary) string {
	promptVariant := strings.TrimSpace(comp.PromptVariant)
	if promptVariant == "" {
		promptVariant = "default"
	}
	policy := string(comp.SemeditArmRestrict)
	if policy == "" {
		policy = "unspecified"
	}
	return fmt.Sprintf("%s prompt · %s MCP instructions · %s semedit restriction", promptVariant, normalizeMCPServerInstructions(comp.MCPServerInstructions), policy)
}

func renderSemanticToolReflection(sb *strings.Builder, title string, run *RunResult) {
	if run == nil || run.SemanticToolReflection == nil {
		return
	}
	reflection := run.SemanticToolReflection
	fmt.Fprintf(sb, "#### %s: Semedit Tool-Use Reflection\n\n", title)
	sb.WriteString("No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.\n\n")
	if reflection.Response != "" {
		renderSemanticToolCallout(sb, reflection.Response)
	}
	renderSemanticReflectionDetails(sb, reflection)
}

func renderSemanticBatchReflection(sb *strings.Builder, title string, run *RunResult) {
	if run == nil || run.SemanticBatchReflection == nil {
		return
	}
	reflection := run.SemanticBatchReflection
	fmt.Fprintf(sb, "#### %s: Semedit Batch-Use Reflection\n\n", title)
	sb.WriteString("Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.\n\n")
	if reflection.Response != "" {
		renderSemanticBatchCallout(sb, reflection.Response)
	}
	renderSemanticReflectionDetails(sb, reflection)
}

func renderSemanticToolCallout(sb *strings.Builder, response string) {
	calloutClass, icon := "callout-warning", "⚠"
	if noSemanticToolsAvailablePattern.MatchString(response) {
		calloutClass, icon = "callout-error", "✕"
	}
	fmt.Fprintf(sb, "<div class=\"callout %s\"><div class=\"callout-title\"><span>%s</span> Why no semantic edit tool was used</div><div class=\"callout-desc\">%s</div></div>\n\n", calloutClass, icon, html.EscapeString(response))
}

func renderSemanticBatchCallout(sb *strings.Builder, response string) {
	fmt.Fprintf(sb, "<div class=\"callout callout-warning\"><div class=\"callout-title\"><span>⚠</span> Why semantic edits were not batched</div><div class=\"callout-desc\">%s</div></div>\n\n", html.EscapeString(response))
}

func renderComparisonTable(sb *strings.Builder, sbRun, smRun, lbRun, lmRun *RunResult) {
	fmt.Fprintf(sb, "| Metric | Vanilla (Small) | %s | Δ (Small) | Vanilla (Large) | %s | Δ (Large) |\n", mcpColumnHeader("Small", smRun), mcpColumnHeader("Large", lmRun))
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
	runs := benchmarkComparisonRuns{smallBaseline: sbRun, smallSemedit: smRun, largeBaseline: lbRun, largeSemedit: lmRun}
	renderTimingComparisonRows(sb, runs)
	renderInteractionComparisonRows(sb, runs)
	renderTokenComparisonRows(sb, runs)
	renderOracleComparisonRows(sb, runs)
	sb.WriteString("\n")
}

type benchmarkComparisonRuns struct {
	smallBaseline *RunResult
	smallSemedit  *RunResult
	largeBaseline *RunResult
	largeSemedit  *RunResult
}

func renderTimingComparisonRows(sb *strings.Builder, runs benchmarkComparisonRuns) {
	sb.WriteString(formatMetricRowDuration("Wall-Clock Latency",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) time.Duration { return r.WallClock }))
	sb.WriteString(formatMetricRowOptionalDuration("Process Start → First Event",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) *time.Duration { return r.ProcessStartToFirstEvent }))
	sb.WriteString(formatMetricRowOptionalDuration("First Event → First Tool Call",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) *time.Duration { return r.FirstEventToFirstToolCall }))
	sb.WriteString(formatMetricRowOptionalDuration("MCP Initialize → First Semantic Call",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) *time.Duration { return r.MCPInitializeToFirstSemanticCall }))
	sb.WriteString(formatMetricRowOptionalDuration("MCP Server Start → Initialize",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) *time.Duration { return r.MCPServerStartToInitialize }))
}

func renderInteractionComparisonRows(sb *strings.Builder, runs benchmarkComparisonRuns) {
	sb.WriteString(formatMetricRowInt("Top-Level User Turns",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.Turns }))
	sb.WriteString(formatMetricRowInt("Internal Tool Cycles",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.InternalTurns }))
	sb.WriteString(formatMetricRowInt("Initial Load / Discovery Turns",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.InitialLoadTurns }))
	sb.WriteString(formatMetricRowInt("MCP Discovery / Schema Turns",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.MCPLoadTurns }))
	sb.WriteString(formatMetricRowInt("Total Tool Invocations",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.ToolCount }))
}

func renderTokenComparisonRows(sb *strings.Builder, runs benchmarkComparisonRuns) {
	sb.WriteString(formatMetricRowInt("Output Tokens",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.OutputTokens }))
	sb.WriteString(formatMetricRowInt("Reasoning / Thinking Tokens",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.ReasoningTokens }))
	sb.WriteString(formatMetricRowInt("Total Input Tokens",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.PromptTokens }))
	sb.WriteString(formatMetricRowIntWithDirection("Cached Input Tokens",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.CachedPromptTokens }, deltaHigherIsBetter))
	sb.WriteString(formatMetricRowInt("Uncached Input Tokens",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(r *RunResult) int { return r.UncachedPromptTokens }))
	sb.WriteString(formatCachedToUncachedRatioRow(runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit))
}

func renderOracleComparisonRows(sb *strings.Builder, runs benchmarkComparisonRuns) {
	sb.WriteString(formatOracleRow("Oracle L1: Mutation Policy",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(o *OracleResult) bool { return o.Level1Policy }))
	sb.WriteString(formatOracleRow("Oracle L2: AST Invariants",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(o *OracleResult) bool { return o.Level2AST }))
	sb.WriteString(formatOracleRow("Oracle L3: Clean Build",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(o *OracleResult) bool { return o.Level3Build }))
	sb.WriteString(formatOracleRow("Oracle L4: Verification Test",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit,
		func(o *OracleResult) bool { return o.Level4Test }))
	sb.WriteString(formatMCPVerifiedRow("MCP Tools Invocation Verified",
		runs.smallBaseline, runs.smallSemedit, runs.largeBaseline, runs.largeSemedit))
}

func mcpColumnHeader(context string, run *RunResult) string {
	armLabel := "MCP"
	if run != nil {
		switch run.Arm {
		case ArmBaseline, ArmSemedit, ArmControl:
		case ArmSemeditAgentic:
			armLabel = "Semedit Agentic"
		case ArmSemeditExplicit:
			armLabel = "Semedit Explicit"
		}
	}
	label := fmt.Sprintf("%s (%s)", armLabel, context)
	if run == nil || run.MCPVerified {
		return label
	}
	return fmt.Sprintf("<span role=\"img\" aria-label=\"Semantic tool invocation not verified\" title=\"Semantic tool invocation not verified\">⚠</span> %s", label)
}

func renderDiffSection(sb *strings.Builder, headingLevel int, title string, base, mcp *RunResult) {
	if (base == nil || (base.Diff == "" && len(base.ToolsUsed) == 0 && len(base.ToolCalls) == 0)) && (mcp == nil || (mcp.Diff == "" && len(mcp.ToolsUsed) == 0 && len(mcp.ToolCalls) == 0)) {
		return
	}
	fmt.Fprintf(sb, "%s %s\n", strings.Repeat("#", headingLevel), title)
	if base != nil {
		if base.Diff != "" {
			fmt.Fprintf(sb, "* **Vanilla Edit**: %s\n", base.Diff)
		}
	}
	if mcp != nil {
		if mcp.Diff != "" {
			fmt.Fprintf(sb, "* **MCP Edit**: %s\n", mcp.Diff)
		}
	}
	sb.WriteString("\n")
	renderToolCallComparison(sb, base, mcp)
	sb.WriteString("\n")
}

func renderToolCallComparison(sb *strings.Builder, base, mcp *RunResult) {
	if runNeedsHTMLTable(base) || runNeedsHTMLTable(mcp) {
		renderToolCallHTMLTable(sb, base, mcp)
		return
	}

	sb.WriteString("| # | Vanilla | Semedit MCP |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")
	rows := max(toolCallCount(base), toolCallCount(mcp), 1)
	for index := range rows {
		fmt.Fprintf(sb, "| %d | %s | %s |\n", index+1, toolCallAt(base, index), toolCallAt(mcp, index))
	}
}

func runNeedsHTMLTable(run *RunResult) bool {
	if run == nil {
		return false
	}
	return slices.ContainsFunc(run.ToolCalls, func(call ToolCall) bool {
		return toolCallIsShellCommand(call) || len(call.Arguments) > 0
	})
}

func renderToolCallHTMLTable(sb *strings.Builder, base, mcp *RunResult) {
	sb.WriteString("<table class=\"benchmark-tool-calls\">\n<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>\n<tbody>\n")
	rows := max(toolCallCount(base), toolCallCount(mcp), 1)
	for index := range rows {
		fmt.Fprintf(sb, "<tr><td>%d</td><td>%s</td><td>%s</td></tr>\n", index+1, toolCallHTMLCell(base, index), toolCallHTMLCell(mcp, index))
	}
	sb.WriteString("</tbody>\n</table>\n")
}

func toolCallCount(run *RunResult) int {
	if run == nil {
		return 0
	}
	return max(len(run.ToolCalls), len(run.ToolsUsed))
}

func toolCallAt(run *RunResult, index int) string {
	if run == nil {
		return "not published"
	}
	if index < len(run.ToolCalls) {
		call := run.ToolCalls[index]
		if toolCallIsShellCommand(call) {
			return formatShellCommandCell(call)
		}
		server := ""
		if call.Server != "" {
			server = call.Server + "/"
		}
		entry := fmt.Sprintf("`%s%s` (%s)%s", server, call.Name, toolCallStatusSummary(call), html.EscapeString(toolCallDetails(call)))
		return escapeToolCallTableCell(entry)
	}
	if index < len(run.ToolsUsed) {
		return fmt.Sprintf("`%s` (outcome unavailable)", run.ToolsUsed[index])
	}
	if toolCallCount(run) == 0 {
		return "none recorded"
	}
	return "—"
}

func toolCallIsShellCommand(call ToolCall) bool {
	return call.Server == "" && (strings.HasPrefix(call.Name, "/bin/") || strings.HasPrefix(call.Name, "bash ") || strings.HasPrefix(call.Name, "zsh "))
}

func formatShellCommandCell(call ToolCall) string {
	command := html.EscapeString(formatShellCommand(call.Name))
	return fmt.Sprintf("%s%s<pre class=\"benchmark-shell-command\"><code class=\"language-shell\">%s</code></pre>%s", toolCallStatusSummary(call), html.EscapeString(toolCallDetails(call)), command, formatToolArguments(call.Arguments))
}

func toolCallHTMLCell(run *RunResult, index int) string {
	if run == nil {
		return "not published"
	}
	if index < len(run.ToolCalls) {
		call := run.ToolCalls[index]
		if toolCallIsShellCommand(call) {
			return formatShellCommandCell(call)
		}
		name := call.Name
		if call.Server != "" {
			name = call.Server + "/" + name
		}
		return fmt.Sprintf("<code>%s</code> (%s)%s%s", html.EscapeString(name), toolCallStatusSummary(call), html.EscapeString(toolCallDetails(call)), formatToolArguments(call.Arguments))
	}
	if index < len(run.ToolsUsed) {
		return fmt.Sprintf("<code>%s</code> (outcome unavailable)", html.EscapeString(run.ToolsUsed[index]))
	}
	if toolCallCount(run) == 0 {
		return "none recorded"
	}
	return "—"
}

func formatToolArguments(arguments json.RawMessage) string {
	arguments = bytes.TrimSpace(arguments)
	if len(arguments) == 0 || bytes.Equal(arguments, []byte("null")) {
		return ""
	}

	var formatted bytes.Buffer
	if err := json.Indent(&formatted, arguments, "", "  "); err != nil {
		formatted.Write(arguments)
	}
	return fmt.Sprintf("<pre class=\"benchmark-tool-arguments\"><code class=\"language-json\">%s</code></pre>", html.EscapeString(formatted.String()))
}

func toolCallStatusSummary(call ToolCall) string {
	return fmt.Sprintf("%s %s", formatToolCallStatus("Transport", call.TransportStatus), formatToolCallStatus("Functional", call.FunctionalStatus))
}

func toolCallDetails(call ToolCall) string {
	entry := ""
	if call.Failure != "" {
		entry += fmt.Sprintf(": %s", call.Failure)
	}
	return entry + formatMCPMetrics(call.MCPMetrics)
}

func formatToolCallStatus(kind string, status ToolCallStatus) string {
	normalized := strings.ToLower(strings.TrimSpace(string(status)))
	glyph := "?"
	switch normalized {
	case "succeeded", "success", "completed":
		glyph = "✓"
	case "failed", "failure", "error":
		glyph = "✗"
	case "timeout", "timed_out", "timed out":
		glyph = "⏱"
	}
	if normalized == "" {
		normalized = "unknown"
	}
	description := kind + " " + normalized
	return fmt.Sprintf("<span role=\"img\" aria-label=\"%s\" title=\"%s\">%s</span>", html.EscapeString(description), html.EscapeString(description), glyph)
}

func formatShellCommand(command string) string {
	for _, operator := range []string{"&&", "||"} {
		parts := strings.Split(command, operator)
		if len(parts) == 1 {
			continue
		}
		var formatted strings.Builder
		formatted.WriteString(parts[0])
		for _, part := range parts[1:] {
			formatted.WriteString(operator + " \\\n")
			formatted.WriteString(strings.TrimLeft(part, " \t"))
		}
		command = formatted.String()
	}
	return command
}

func formatMCPMetrics(metrics *MCPMetrics) string {
	if metrics == nil {
		return ""
	}
	var verificationMS, formattingMS int64
	for phase, metric := range metrics.Phases {
		switch {
		case strings.HasPrefix(phase, "verification."):
			verificationMS += metric.DurationMS
		case strings.HasPrefix(phase, "formatting."):
			formattingMS += metric.DurationMS
		}
	}
	entry := fmt.Sprintf("; server: %s", time.Duration(metrics.TotalMS)*time.Millisecond)
	if verificationMS > 0 || formattingMS > 0 {
		entry += " ("
		parts := make([]string, 0, 2)
		if verificationMS > 0 {
			parts = append(parts, fmt.Sprintf("verification: %s", time.Duration(verificationMS)*time.Millisecond))
		}
		if formattingMS > 0 {
			parts = append(parts, fmt.Sprintf("formatting: %s", time.Duration(formattingMS)*time.Millisecond))
		}
		entry += strings.Join(parts, "; ") + ")"
	}
	return entry
}

func escapeToolCallTableCell(value string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(value), " "), "|", "\\|")
}

func formatMetricRowInt(name string, sb, sm, lb, lm *RunResult, get func(*RunResult) int) string {
	return formatMetricRowIntWithDirection(name, sb, sm, lb, lm, get, deltaLowerIsBetter)
}

func formatMetricRowIntWithDirection(name string, sb, sm, lb, lm *RunResult, get func(*RunResult) int, direction deltaDirection) string {
	valSB, valSM, deltaS := computeIntDelta(sb, sm, get, direction)
	valLB, valLM, deltaL := computeIntDelta(lb, lm, get, direction)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatCachedToUncachedRatioRow(sb, sm, lb, lm *RunResult) string {
	valSB, valSM, deltaS := computeCachedToUncachedRatioDelta(sb, sm)
	valLB, valLM, deltaL := computeCachedToUncachedRatioDelta(lb, lm)
	return fmt.Sprintf("| **Cached vs Uncached Token Ratio** | %s | %s | %s | %s | %s | %s |\n", valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatMetricRowDuration(name string, sb, sm, lb, lm *RunResult, get func(*RunResult) time.Duration) string {
	valSB, valSM, deltaS := computeDurationDelta(sb, sm, get, deltaLowerIsBetter)
	valLB, valLM, deltaL := computeDurationDelta(lb, lm, get, deltaLowerIsBetter)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatMetricRowOptionalDuration(name string, sb, sm, lb, lm *RunResult, get func(*RunResult) *time.Duration) string {
	format := func(run *RunResult) string {
		if run == nil || get(run) == nil {
			return "—"
		}
		return fmt.Sprintf("%.2fs", get(run).Seconds())
	}
	return fmt.Sprintf("| **%s** | %s | %s | — | %s | %s | — |\n", name, format(sb), format(sm), format(lb), format(lm))
}

func formatOracleRow(name string, sb, sm, lb, lm *RunResult, get func(*OracleResult) bool) string {
	render := func(r *RunResult) string {
		if r == nil || r.Oracle == nil {
			return "—"
		}
		if get(r.Oracle) {
			return "✅ PASS"
		}
		return "❌ FAIL"
	}

	return fmt.Sprintf("| **%s** | %s | %s | — | %s | %s | — |\n",
		name, render(sb), render(sm), render(lb), render(lm))
}

func formatMCPVerifiedRow(name string, sb, sm, lb, lm *RunResult) string {
	render := func(r *RunResult) string {
		if r == nil {
			return "—"
		}
		if r.Arm == ArmBaseline {
			if r.MCPVerified {
				return "❌ UNEXPECTED"
			}
			return "✅ N/A (Vanilla)"
		}
		if r.MCPVerified {
			return "✅ YES"
		}
		return "⚠️ NO (Fallback)"
	}

	return fmt.Sprintf("| **%s** | %s | %s | — | %s | %s | — |\n",
		name, render(sb), render(sm), render(lb), render(lm))
}

type deltaDirection bool

const (
	deltaLowerIsBetter  deltaDirection = false
	deltaHigherIsBetter deltaDirection = true
)

func computeIntDelta(base, mcp *RunResult, get func(*RunResult) int, direction deltaDirection) (string, string, string) {
	if base == nil && mcp == nil {
		return "—", "—", "—"
	}
	if base == nil {
		return "—", fmt.Sprintf("%d", get(mcp)), "—"
	}
	if mcp == nil {
		return fmt.Sprintf("%d", get(base)), "—", "—"
	}

	bVal := get(base)
	mVal := get(mcp)
	bStr := fmt.Sprintf("%d", bVal)
	mStr := fmt.Sprintf("%d", mVal)

	if bVal == 0 {
		if mVal == 0 {
			return bStr, mStr, "0%"
		}
		return bStr, mStr, formatDelta(100, float64(mVal), direction, mcp.MCPVerified)
	}

	diff := mVal - bVal
	pct := float64(diff) / float64(bVal) * 100.0
	return bStr, mStr, formatDelta(pct, float64(diff), direction, mcp.MCPVerified)
}

func computeCachedToUncachedRatioDelta(base, mcp *RunResult) (string, string, string) {
	if base == nil && mcp == nil {
		return "—", "—", "—"
	}
	if base == nil {
		_, value, _ := cachedToUncachedRatio(mcp)
		return "—", value, "—"
	}
	if mcp == nil {
		_, value, _ := cachedToUncachedRatio(base)
		return value, "—", "—"
	}

	baseRatio, baseText, baseComparable := cachedToUncachedRatio(base)
	mcpRatio, mcpText, mcpComparable := cachedToUncachedRatio(mcp)
	if !baseComparable || !mcpComparable || baseRatio == 0 {
		return baseText, mcpText, "—"
	}
	diff := mcpRatio - baseRatio
	pct := diff / baseRatio * 100
	return baseText, mcpText, formatDelta(pct, diff, deltaHigherIsBetter, mcp.MCPVerified)
}

func cachedToUncachedRatio(run *RunResult) (float64, string, bool) {
	if run.UncachedPromptTokens <= 0 {
		if run.CachedPromptTokens > 0 {
			return 0, "∞:1", false
		}
		return 0, "—", false
	}
	ratio := float64(run.CachedPromptTokens) / float64(run.UncachedPromptTokens)
	return ratio, fmt.Sprintf("%.2f:1", ratio), true
}

func computeDurationDelta(base, mcp *RunResult, get func(*RunResult) time.Duration, direction deltaDirection) (string, string, string) {
	if base == nil && mcp == nil {
		return "—", "—", "—"
	}
	if base == nil {
		return "—", fmt.Sprintf("%.2fs", get(mcp).Seconds()), "—"
	}
	if mcp == nil {
		return fmt.Sprintf("%.2fs", get(base).Seconds()), "—", "—"
	}

	bSec := get(base).Seconds()
	mSec := get(mcp).Seconds()
	bStr := fmt.Sprintf("%.2fs", bSec)
	mStr := fmt.Sprintf("%.2fs", mSec)

	if bSec == 0 {
		return bStr, mStr, "—"
	}

	diff := mSec - bSec
	pct := (diff / bSec) * 100.0
	return bStr, mStr, formatDelta(pct, diff, direction, mcp.MCPVerified)
}

func formatDelta(pct, diff float64, direction deltaDirection, mcpVerified bool) string {
	if !mcpVerified {
		return "N/A"
	}
	if diff == 0 {
		return "0%"
	}
	delta := fmt.Sprintf("%+.1f%%", pct)
	mcpBenefits := (direction == deltaLowerIsBetter && diff < 0) || (direction == deltaHigherIsBetter && diff > 0)
	if mcpBenefits {
		return fmt.Sprintf("<span class=\"benchmark-delta-positive\">%s</span>", delta)
	}
	if !mcpBenefits {
		return fmt.Sprintf("<span class=\"benchmark-delta-negative\">%s</span>", delta)
	}
	return delta
}

func assignComparisonRun(comp *ComparisonSummary, r *RunResult) {
	variant := comparisonRunVariantFor(r)
	switch r.Arm {
	case ArmBaseline:
		assignBaselineComparisonRun(comp, r, variant)
	case ArmSemeditExplicit:
		assignExplicitComparisonRun(comp, r, variant)
	case ArmSemedit, ArmSemeditAgentic, ArmControl:
		assignSemeditComparisonRun(comp, r, variant)
	}
}

type comparisonRunVariant struct {
	large    bool
	verified bool
}

func comparisonRunVariantFor(run *RunResult) comparisonRunVariant {
	variant := strings.ToLower(run.Variant)
	if index := strings.Index(variant, ":"); index >= 0 {
		variant = variant[:index]
	}
	if variant == "" {
		variant = "small"
		if strings.Contains(run.TaskID, "large") {
			variant = "large"
		}
	}
	return comparisonRunVariant{
		large:    strings.Contains(variant, "large"),
		verified: strings.Contains(variant, "verified"),
	}
}

func assignBaselineComparisonRun(comparison *ComparisonSummary, run *RunResult, variant comparisonRunVariant) {
	if variant.verified {
		assignComparisonPrompt(&comparison.VanillaVerifiedPrompt, run.Prompt)
		assignComparisonRunSlot(&comparison.SmallVerifiedBaseline, &comparison.LargeVerifiedBaseline, run, variant.large)
		return
	}
	assignComparisonPrompt(&comparison.VanillaPrompt, run.Prompt)
	assignComparisonRunSlot(&comparison.SmallBaseline, &comparison.LargeBaseline, run, variant.large)
}

func assignSemeditComparisonRun(comparison *ComparisonSummary, run *RunResult, variant comparisonRunVariant) {
	if variant.verified {
		assignComparisonPrompt(&comparison.MCPVerifiedPrompt, run.Prompt)
		assignComparisonRunSlot(&comparison.SmallVerifiedSemedit, &comparison.LargeVerifiedSemedit, run, variant.large)
		return
	}
	assignComparisonPrompt(&comparison.MCPPrompt, run.Prompt)
	assignComparisonRunSlot(&comparison.SmallSemedit, &comparison.LargeSemedit, run, variant.large)
}

func assignExplicitComparisonRun(comparison *ComparisonSummary, run *RunResult, variant comparisonRunVariant) {
	if variant.large {
		if variant.verified {
			comparison.LargeVerifiedExplicit = run
		} else {
			comparison.LargeSemeditExplicit = run
		}
		return
	}
	if variant.verified {
		comparison.SmallVerifiedExplicit = run
	} else {
		comparison.SmallSemeditExplicit = run
	}
}

func assignComparisonPrompt(current *string, prompt string) {
	if *current == "" && prompt != "" {
		*current = prompt
	}
}

func assignComparisonRunSlot(small, large **RunResult, run *RunResult, isLarge bool) {
	if isLarge {
		*large = run
		return
	}
	*small = run
}

func comparisonTaskIdentity(run *RunResult) string {
	if run.JobID != "" {
		return run.TaskID
	}
	return normalizeTaskBase(run.TaskID)
}

func renderSemanticReflectionDetails(sb *strings.Builder, reflection *SemanticToolReflection) {
	sb.WriteString("<details><summary>Session reflection</summary>\n\n")
	fmt.Fprintf(sb, "<p><strong>Prompt:</strong></p><pre>%s</pre>\n", html.EscapeString(reflection.Prompt))
	if reflection.Error != "" {
		fmt.Fprintf(sb, "<p><strong>Capture error:</strong> %s</p>\n", html.EscapeString(reflection.Error))
	}
	sb.WriteString("</details>\n\n")
}

func renderSemanticIntegrationReflection(sb *strings.Builder, title string, run *RunResult) {
	if run == nil || run.SemanticIntegrationReflection == nil {
		return
	}
	reflection := run.SemanticIntegrationReflection
	fmt.Fprintf(sb, "#### %s: Semedit Integration Feedback\n\n", title)
	sb.WriteString("Agent opinion based on this run; excluded from benchmark measurements.\n\n")
	if reflection.Response != "" {
		fmt.Fprintf(sb, "<blockquote>%s</blockquote>\n\n", html.EscapeString(reflection.Response))
	}
	renderSemanticReflectionDetails(sb, reflection)
}

func restoreBenchmarkDurationsFromMilliseconds(value any) error {
	switch typed := value.(type) {
	case []any:
		for _, child := range typed {
			if err := restoreBenchmarkDurationsFromMilliseconds(child); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, child := range typed {
			if err := restoreBenchmarkDurationsFromMilliseconds(child); err != nil {
				return err
			}
		}
		for _, key := range []string{
			"wall_clock_ms",
			"process_start_to_first_event_ms",
			"first_event_to_first_tool_call_ms",
			"mcp_initialize_to_first_semantic_call_ms",
			"mcp_server_start_to_initialize_ms",
		} {
			if err := restoreDurationField(typed, key); err != nil {
				return err
			}
		}
		if _, isOracle := typed["level_1_policy"]; isOracle {
			if err := restoreDurationField(typed, "duration_ms"); err != nil {
				return err
			}
		}
	}
	return nil
}

func restoreDurationField(fields map[string]any, key string) error {
	raw, exists := fields[key]
	if !exists || raw == nil {
		return nil
	}
	milliseconds, ok := raw.(float64)
	if !ok || milliseconds != float64(int64(milliseconds)) {
		return fmt.Errorf("restore %s: expected integral milliseconds, got %T (%v)", key, raw, raw)
	}
	const maximumMilliseconds = float64((1<<63 - 1) / int64(time.Millisecond))
	if milliseconds > maximumMilliseconds || milliseconds < -maximumMilliseconds {
		return fmt.Errorf("restore %s: duration is out of range: %v milliseconds", key, milliseconds)
	}
	fields[key] = int64(milliseconds) * int64(time.Millisecond)
	return nil
}

// SaveReport writes telemetry stats as JSON and a rendered Markdown table.
func SaveReport(report *BenchmarkReport, outJSONPath, outMDPath string) error {
	if report.Comparisons == nil {
		report.Comparisons = BuildComparisons(report.Runs)
	}

	if outJSONPath != "" {
		if err := os.MkdirAll(filepath.Dir(outJSONPath), 0o750); err != nil {
			return fmt.Errorf("mkdir json: %w", err)
		}
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal report json: %w", err)
		}
		if err := pipeline.WriteAtomic(outJSONPath, jsonData); err != nil {
			return fmt.Errorf("write report json: %w", err)
		}
	}

	if outMDPath != "" {
		if err := os.MkdirAll(filepath.Dir(outMDPath), 0o750); err != nil {
			return fmt.Errorf("mkdir md: %w", err)
		}
		md := report.RenderMarkdown()
		if err := pipeline.WriteAtomic(outMDPath, []byte(md)); err != nil {
			return fmt.Errorf("write report md: %w", err)
		}
	}

	return nil
}
