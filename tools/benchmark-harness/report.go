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
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			if err := convertBenchmarkDurationsToMilliseconds(item); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, item := range value {
			if err := convertBenchmarkDurationsToMilliseconds(item); err != nil {
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
			if raw, ok := value[key]; ok {
				milliseconds, err := durationNanosecondsToMilliseconds(raw)
				if err != nil {
					return fmt.Errorf("convert %s: %w", key, err)
				}
				value[key] = milliseconds
			}
		}
		if _, isOracle := value["level_1_policy"]; isOracle {
			if raw, ok := value["duration_ms"]; ok {
				milliseconds, err := durationNanosecondsToMilliseconds(raw)
				if err != nil {
					return fmt.Errorf("convert oracle duration_ms: %w", err)
				}
				value["duration_ms"] = milliseconds
			}
		}
	}
	return nil
}

func durationNanosecondsToMilliseconds(raw any) (int64, error) {
	nanoseconds, ok := raw.(float64)
	if !ok || nanoseconds != float64(int64(nanoseconds)) {
		return 0, fmt.Errorf("expected integral duration, got %T (%v)", raw, raw)
	}
	return int64(nanoseconds) / int64(time.Millisecond), nil
}

// ComparisonSummary bundles Baseline vs MCP runs for a specific Task and Target across variants.
type ComparisonSummary struct {
	TaskID                string                   `json:"task_id"`
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
	LargeBaseline         *RunResult               `json:"large_baseline,omitempty"`
	LargeSemedit          *RunResult               `json:"large_semedit,omitempty"`
	LargeVerifiedBaseline *RunResult               `json:"large_verified_baseline,omitempty"`
	LargeVerifiedSemedit  *RunResult               `json:"large_verified_semedit,omitempty"`
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
	type key struct {
		baseTask              string
		target                string
		promptVariant         string
		mcpServerInstructions MCPServerInstructionMode
	}

	grouped := make(map[key]*ComparisonSummary)

	for _, r := range runs {
		baseTask := normalizeTaskBase(r.TaskID)
		mcpServerInstructions := normalizeMCPServerInstructions(r.MCPServerInstructions)
		k := key{baseTask: baseTask, target: r.Target.String(), promptVariant: r.PromptVariant, mcpServerInstructions: mcpServerInstructions}
		comp, exists := grouped[k]
		if !exists {
			comp = &ComparisonSummary{
				TaskID:                baseTask,
				PromptVariant:         r.PromptVariant,
				MCPServerInstructions: mcpServerInstructions,
				Provenance:            r.Provenance.Clone(),
				Target:                r.Target,
				BeforeState:           r.BeforeState,
			}
			grouped[k] = comp
		} else {
			comp.Provenance = commonProvenance(comp.Provenance, r.Provenance)
		}

		if comp.BeforeState == "" && r.BeforeState != "" {
			comp.BeforeState = r.BeforeState
		}

		variant := strings.ToLower(r.Variant)
		if idx := strings.Index(variant, ":"); idx >= 0 {
			variant = variant[:idx]
		}
		if variant == "" {
			if strings.Contains(r.TaskID, "large") {
				variant = "large"
			} else {
				variant = "small"
			}
		}

		isLarge := strings.Contains(variant, "large")
		isVerified := strings.Contains(variant, "verified")

		switch r.Arm {
		case ArmBaseline:
			if isVerified {
				if comp.VanillaVerifiedPrompt == "" && r.Prompt != "" {
					comp.VanillaVerifiedPrompt = r.Prompt
				}
				if isLarge {
					comp.LargeVerifiedBaseline = r
				} else {
					comp.SmallVerifiedBaseline = r
				}
			} else {
				if comp.VanillaPrompt == "" && r.Prompt != "" {
					comp.VanillaPrompt = r.Prompt
				}
				if isLarge {
					comp.LargeBaseline = r
				} else {
					comp.SmallBaseline = r
				}
			}
		case ArmSemedit, ArmControl:
			if isVerified {
				if comp.MCPVerifiedPrompt == "" && r.Prompt != "" {
					comp.MCPVerifiedPrompt = r.Prompt
				}
				if isLarge {
					comp.LargeVerifiedSemedit = r
				} else {
					comp.SmallVerifiedSemedit = r
				}
			} else {
				if comp.MCPPrompt == "" && r.Prompt != "" {
					comp.MCPPrompt = r.Prompt
				}
				if isLarge {
					comp.LargeSemedit = r
				} else {
					comp.SmallSemedit = r
				}
			}
		}
	}

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
		if result[i].PromptVariant != result[j].PromptVariant {
			return result[i].PromptVariant < result[j].PromptVariant
		}
		if result[i].MCPServerInstructions != result[j].MCPServerInstructions {
			return result[i].MCPServerInstructions < result[j].MCPServerInstructions
		}
		return false
	})

	return result
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

		targetStr := comp.Target.String()
		if targetStr != currentTarget {
			fmt.Fprintf(&sb, "### Target: `%s`\n\n", targetStr)
			currentTarget = targetStr
		}
		fmt.Fprintf(&sb, "#### Configuration: %s\n\n", formatConfiguration(comp))
		if provenance := comp.Provenance.String(); provenance != "" {
			fmt.Fprintf(&sb, "* **Run Provenance**: `%s`\n\n", provenance)
		}

		switch {
		case comp.TxtarProvenance != "" && strings.HasPrefix(comp.TxtarProvenance, "http"):
			fmt.Fprintf(&sb, "* **Fixture**: [%s](%s)\n\n", comp.TxtarPath, comp.TxtarProvenance)
		case comp.TxtarProvenance != "":
			fmt.Fprintf(&sb, "* **Fixture**: `%s` (`%s`)\n\n", comp.TxtarPath, comp.TxtarProvenance)
		case comp.TxtarPath != "":
			fmt.Fprintf(&sb, "* **Fixture**: `%s`\n\n", comp.TxtarPath)
		}

		if comp.VanillaPrompt != "" && comp.MCPPrompt != "" && comp.VanillaPrompt == comp.MCPPrompt {
			writePrompt(&sb, "LLM Prompt", comp.VanillaPrompt)
		} else {
			if comp.VanillaPrompt != "" {
				writePrompt(&sb, "Vanilla LLM Prompt", comp.VanillaPrompt)
			}
			if comp.MCPPrompt != "" {
				writePrompt(&sb, "Semedit MCP Prompt", comp.MCPPrompt)
			}
		}

		hasVerified := comp.SmallVerifiedBaseline != nil || comp.SmallVerifiedSemedit != nil ||
			comp.LargeVerifiedBaseline != nil || comp.LargeVerifiedSemedit != nil

		if hasVerified {
			if comp.VanillaVerifiedPrompt != "" && comp.MCPVerifiedPrompt != "" && comp.VanillaVerifiedPrompt == comp.MCPVerifiedPrompt {
				writePrompt(&sb, "Verified Prompt", comp.VanillaVerifiedPrompt)
			} else {
				if comp.VanillaVerifiedPrompt != "" {
					writePrompt(&sb, "Vanilla (Verified) Prompt", comp.VanillaVerifiedPrompt)
				}
				if comp.MCPVerifiedPrompt != "" {
					writePrompt(&sb, "Semedit MCP (Verified) Prompt", comp.MCPVerifiedPrompt)
				}
			}
		}

		if comp.BeforeState != "" {
			sb.WriteString("<details><summary><b>Initial Workspace State (Before Edit)</b></summary>\n\n```go\n")
			sb.WriteString(comp.BeforeState)
			sb.WriteString("\n```\n</details>\n\n")
		}

		// 1. Standard Comparison Table (Unverified)
		hasStandard := comp.SmallBaseline != nil || comp.SmallSemedit != nil ||
			comp.LargeBaseline != nil || comp.LargeSemedit != nil

		if hasStandard {
			renderComparisonTable(&sb, comp.SmallBaseline, comp.SmallSemedit, comp.LargeBaseline, comp.LargeSemedit)
			renderDiffSection(&sb, 5, "Standard vs Semedit in Small Context", comp.SmallBaseline, comp.SmallSemedit)
			renderDiffSection(&sb, 5, "Standard vs Semedit in Large Context", comp.LargeBaseline, comp.LargeSemedit)
			renderSemanticToolReflection(&sb, "Standard vs Semedit in Small Context", comp.SmallSemedit)
			renderSemanticToolReflection(&sb, "Standard vs Semedit in Large Context", comp.LargeSemedit)
			renderSemanticBatchReflection(&sb, "Standard vs Semedit in Small Context", comp.SmallSemedit)
			renderSemanticBatchReflection(&sb, "Standard vs Semedit in Large Context", comp.LargeSemedit)
		}

		// 2. Verified Directive Comparison Table
		if hasVerified {
			sb.WriteString("##### Verified Directive Comparison (+Self-Correction Loop)\n\n")
			renderComparisonTable(&sb, comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit, comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
			renderDiffSection(&sb, 6, "Verified vs Semedit in Small Context", comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit)
			renderDiffSection(&sb, 6, "Verified vs Semedit in Large Context", comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
			renderSemanticToolReflection(&sb, "Verified vs Semedit in Small Context", comp.SmallVerifiedSemedit)
			renderSemanticToolReflection(&sb, "Verified vs Semedit in Large Context", comp.LargeVerifiedSemedit)
			renderSemanticBatchReflection(&sb, "Verified vs Semedit in Small Context", comp.SmallVerifiedSemedit)
			renderSemanticBatchReflection(&sb, "Verified vs Semedit in Large Context", comp.LargeVerifiedSemedit)
		}
	}

	return sb.String()
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
	return fmt.Sprintf("%s prompt · %s MCP instructions", promptVariant, normalizeMCPServerInstructions(comp.MCPServerInstructions))
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
	sb.WriteString("<details><summary>Session reflection</summary>\n\n")
	fmt.Fprintf(sb, "<p><strong>Prompt:</strong></p><pre>%s</pre>\n", html.EscapeString(reflection.Prompt))
	if reflection.Error != "" {
		fmt.Fprintf(sb, "<p><strong>Capture error:</strong> %s</p>\n", html.EscapeString(reflection.Error))
	}
	fmt.Fprintf(sb, "<p>Reflection wall-clock: %.2fs; turns: %d; tool calls: %d.</p>\n", reflection.WallClock.Seconds(), reflection.Turns, len(reflection.ToolCalls))
	sb.WriteString("</details>\n\n")
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
	sb.WriteString("<details><summary>Session reflection</summary>\n\n")
	fmt.Fprintf(sb, "<p><strong>Prompt:</strong></p><pre>%s</pre>\n", html.EscapeString(reflection.Prompt))
	if reflection.Error != "" {
		fmt.Fprintf(sb, "<p><strong>Capture error:</strong> %s</p>\n", html.EscapeString(reflection.Error))
	}
	fmt.Fprintf(sb, "<p>Reflection wall-clock: %.2fs; turns: %d; tool calls: %d.</p>\n", reflection.WallClock.Seconds(), reflection.Turns, len(reflection.ToolCalls))
	sb.WriteString("</details>\n\n")
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

	// 1. Wall-Clock Latency
	sb.WriteString(formatMetricRowDuration("Wall-Clock Latency",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) time.Duration { return r.WallClock }))

	sb.WriteString(formatMetricRowOptionalDuration("Process Start → First Event",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) *time.Duration { return r.ProcessStartToFirstEvent }))

	sb.WriteString(formatMetricRowOptionalDuration("First Event → First Tool Call",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) *time.Duration { return r.FirstEventToFirstToolCall }))

	sb.WriteString(formatMetricRowOptionalDuration("MCP Initialize → First Semantic Call",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) *time.Duration { return r.MCPInitializeToFirstSemanticCall }))

	sb.WriteString(formatMetricRowOptionalDuration("MCP Server Start → Initialize",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) *time.Duration { return r.MCPServerStartToInitialize }))

	// 2. Interaction Turns & Cycles Breakdown
	sb.WriteString(formatMetricRowInt("Top-Level User Turns",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.Turns }))

	sb.WriteString(formatMetricRowInt("Internal Tool Cycles",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.InternalTurns }))

	sb.WriteString(formatMetricRowInt("Initial Load / Discovery Turns",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.InitialLoadTurns }))

	sb.WriteString(formatMetricRowInt("MCP Discovery / Schema Turns",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.MCPLoadTurns }))

	sb.WriteString(formatMetricRowInt("Total Tool Invocations",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.ToolCount }))

	// 3. Output Tokens
	sb.WriteString(formatMetricRowInt("Output Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.OutputTokens }))

	// 4. Reasoning / Thinking Tokens
	sb.WriteString(formatMetricRowInt("Reasoning / Thinking Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.ReasoningTokens }))

	// 5. Total Input Tokens
	sb.WriteString(formatMetricRowInt("Total Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.PromptTokens }))

	// 6. Cached Input Tokens
	sb.WriteString(formatMetricRowIntWithDirection("Cached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.CachedPromptTokens }, deltaHigherIsBetter))

	// 7. Uncached Input Tokens
	sb.WriteString(formatMetricRowInt("Uncached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.UncachedPromptTokens }))

	sb.WriteString(formatCachedToUncachedRatioRow(sbRun, smRun, lbRun, lmRun))

	// 8. Oracle L1: Mutation Policy
	sb.WriteString(formatOracleRow("Oracle L1: Mutation Policy",
		sbRun, smRun, lbRun, lmRun,
		func(o *OracleResult) bool { return o.Level1Policy }))

	// 9. Oracle L2: AST Invariants
	sb.WriteString(formatOracleRow("Oracle L2: AST Invariants",
		sbRun, smRun, lbRun, lmRun,
		func(o *OracleResult) bool { return o.Level2AST }))

	// 10. Oracle L3: Clean Build
	sb.WriteString(formatOracleRow("Oracle L3: Clean Build",
		sbRun, smRun, lbRun, lmRun,
		func(o *OracleResult) bool { return o.Level3Build }))

	// 11. Oracle L4: Verification Test
	sb.WriteString(formatOracleRow("Oracle L4: Verification Test",
		sbRun, smRun, lbRun, lmRun,
		func(o *OracleResult) bool { return o.Level4Test }))

	// 12. MCP Tool Invocation Verified
	sb.WriteString(formatMCPVerifiedRow("MCP Tools Invocation Verified",
		sbRun, smRun, lbRun, lmRun))

	sb.WriteString("\n")
}

func mcpColumnHeader(context string, run *RunResult) string {
	label := fmt.Sprintf("MCP (%s)", context)
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
		if err := os.WriteFile(outJSONPath, jsonData, 0o600); err != nil {
			return fmt.Errorf("write report json: %w", err)
		}
	}

	if outMDPath != "" {
		if err := os.MkdirAll(filepath.Dir(outMDPath), 0o750); err != nil {
			return fmt.Errorf("mkdir md: %w", err)
		}
		md := report.RenderMarkdown()
		if err := os.WriteFile(outMDPath, []byte(md), 0o600); err != nil {
			return fmt.Errorf("write report md: %w", err)
		}
	}

	return nil
}
