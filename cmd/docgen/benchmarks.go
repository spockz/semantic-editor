// Package main synthesizes code-derived capability documentation and empirical benchmark results into a Hugo source tree.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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

// BenchTarget identifies the execution harness and model for a benchmark.
type BenchTarget struct {
	Harness string `json:"harness"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
}

func (t BenchTarget) String() string {
	parts := []string{t.Harness}
	if t.Model != "" {
		parts = append(parts, t.Model)
	}
	if t.Effort != "" {
		parts = append(parts, t.Effort)
	}
	return strings.Join(parts, "/")
}

// BenchRunResult captures telemetry for a single trial.
type BenchRunResult struct {
	TaskID                           string                       `json:"task_id"`
	Variant                          string                       `json:"variant,omitempty"`
	PromptVariant                    string                       `json:"prompt_variant,omitempty"`
	MCPServerInstructions            string                       `json:"mcp_server_instructions,omitempty"`
	LegacyMCPInstructions            string                       `json:"mcp_instructions,omitempty"`
	Provenance                       map[string]string            `json:"provenance,omitempty"`
	LegacyClassifiers                map[string]string            `json:"classifiers,omitempty"`
	Target                           BenchTarget                  `json:"target"`
	Arm                              string                       `json:"arm"`
	Success                          bool                         `json:"success"`
	Turns                            int                          `json:"turns"`
	InitialLoadTurns                 int                          `json:"initial_load_turns"`
	MCPLoadTurns                     int                          `json:"mcp_load_turns"`
	InternalTurns                    int                          `json:"internal_turns"`
	ToolCount                        int                          `json:"tool_count"`
	WallClock                        time.Duration                `json:"wall_clock_ms"`
	ProcessStartToFirstEvent         *time.Duration               `json:"process_start_to_first_event_ms,omitempty"`
	FirstEventToFirstToolCall        *time.Duration               `json:"first_event_to_first_tool_call_ms,omitempty"`
	MCPInitializeToFirstSemanticCall *time.Duration               `json:"mcp_initialize_to_first_semantic_call_ms,omitempty"`
	MCPServerStartToInitialize       *time.Duration               `json:"mcp_server_start_to_initialize_ms,omitempty"`
	InitialContextTokens             int                          `json:"initial_context_tokens"`
	PromptTokens                     int                          `json:"prompt_tokens"`
	CachedPromptTokens               int                          `json:"cached_prompt_tokens"`
	UncachedPromptTokens             int                          `json:"uncached_prompt_tokens"`
	OutputTokens                     int                          `json:"output_tokens"`
	ReasoningTokens                  int                          `json:"reasoning_tokens"`
	Oracle                           *BenchOracleResult           `json:"oracle"`
	Prompt                           string                       `json:"prompt,omitempty"`
	BeforeState                      string                       `json:"before_state,omitempty"`
	Diff                             string                       `json:"diff,omitempty"`
	ToolsUsed                        []string                     `json:"tools_used,omitempty"`
	ToolCalls                        []BenchToolCall              `json:"tool_calls,omitempty"`
	InteractionSteps                 []BenchInteractionStep       `json:"interaction_steps,omitempty"`
	SemanticToolReflection           *BenchSemanticToolReflection `json:"semantic_tool_reflection,omitempty"`
	MCPVerified                      bool                         `json:"mcp_verified"`
	Error                            string                       `json:"error,omitempty"`
}

// BenchToolCall captures a tool outcome from a benchmark report without importing the harness package.
type BenchToolCall struct {
	Name             string           `json:"name"`
	Server           string           `json:"server,omitempty"`
	Arguments        json.RawMessage  `json:"arguments,omitempty"`
	TransportStatus  string           `json:"transport_status"`
	FunctionalStatus string           `json:"functional_status"`
	Failure          string           `json:"failure,omitempty"`
	MCPMetrics       *BenchMCPMetrics `json:"mcp_metrics,omitempty"`
}

// BenchInteractionStep preserves the tool-call order from one continued reasoning turn.
type BenchInteractionStep struct {
	Step      int             `json:"step"`
	ToolCalls []BenchToolCall `json:"tool_calls,omitempty"`
}

// BenchSemanticToolReflection records the diagnostic reason an unverified semantic run did not use a semantic edit tool.
type BenchSemanticToolReflection struct {
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

var noSemanticToolsAvailablePattern = regexp.MustCompile(`(?i)\bno\b(?:\W+\w+)*\W+\btools?\b(?:\W+\w+)*\W+\bavailable\b`)

// BenchMCPMetrics records MCP server latency without importing the benchmark harness package.
type BenchMCPMetrics struct {
	SchemaVersion int                            `json:"schema_version"`
	TotalMS       int64                          `json:"total_ms"`
	Phases        map[string]BenchMCPPhaseMetric `json:"phases,omitempty"`
}

// BenchMCPPhaseMetric aggregates an instrumented MCP server phase.
type BenchMCPPhaseMetric struct {
	Count      int   `json:"count"`
	DurationMS int64 `json:"duration_ms"`
}

const benchmarkToolCallCSS = `<style>
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

table.benchmark-tool-calls .benchmark-code-block {
	width: 100%;
	max-width: 32rem;
	margin: 0.5rem 0 0;
	min-width: 0;
	overflow: hidden;
}

table.benchmark-tool-calls .benchmark-code-block pre {
	white-space: pre-wrap;
	overflow-wrap: anywhere;
	word-break: break-word;
	overflow-x: auto;
}

table.benchmark-tool-calls .benchmark-code-block code {
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

const benchmarkCodeShortcode = `{{- $lang := .Get "lang" | default "text" -}}
{{- $encoded := .Get "content" -}}
{{- $content := $encoded | base64Decode -}}
<div class="benchmark-code-block">
  <div class="hextra-code-block hx:relative hx:mt-6 hx:first:mt-0 hx:group/code">
    {{- partial "components/codeblock" (dict "lang" $lang "content" $content "options" (dict)) -}}
    {{- if or (eq site.Params.highlight.copy.enable nil) (site.Params.highlight.copy.enable) -}}
      {{- partialCached "components/codeblock-copy-button" (dict "filename" "") "" -}}
    {{- end -}}
  </div>
</div>
`

// BenchOracleResult records oracle outcomes across evaluation levels.
type BenchOracleResult struct {
	Passed       bool          `json:"passed"`
	Level1Policy bool          `json:"level_1_policy"`
	Level2AST    bool          `json:"level_2_ast"`
	Level3Build  bool          `json:"level_3_build"`
	Level4Test   bool          `json:"level_4_test"`
	FailureStage string        `json:"failure_stage,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
}

// BenchComparisonSummary bundles Baseline vs MCP runs for a task and target.
type BenchComparisonSummary struct {
	TaskID                string            `json:"task_id"`
	PromptVariant         string            `json:"prompt_variant,omitempty"`
	MCPServerInstructions string            `json:"mcp_server_instructions,omitempty"`
	LegacyMCPInstructions string            `json:"mcp_instructions,omitempty"`
	Provenance            map[string]string `json:"provenance,omitempty"`
	LegacyClassifiers     map[string]string `json:"classifiers,omitempty"`
	TxtarPath             string            `json:"txtar_path,omitempty"`
	TxtarProvenance       string            `json:"txtar_provenance,omitempty"`
	Target                BenchTarget       `json:"target"`
	VanillaPrompt         string            `json:"vanilla_prompt,omitempty"`
	MCPPrompt             string            `json:"mcp_prompt,omitempty"`
	VanillaVerifiedPrompt string            `json:"vanilla_verified_prompt,omitempty"`
	MCPVerifiedPrompt     string            `json:"mcp_verified_prompt,omitempty"`
	BeforeState           string            `json:"before_state,omitempty"`
	SmallBaseline         *BenchRunResult   `json:"small_baseline,omitempty"`
	SmallSemedit          *BenchRunResult   `json:"small_semedit,omitempty"`
	SmallVerifiedBaseline *BenchRunResult   `json:"small_verified_baseline,omitempty"`
	SmallVerifiedSemedit  *BenchRunResult   `json:"small_verified_semedit,omitempty"`
	LargeBaseline         *BenchRunResult   `json:"large_baseline,omitempty"`
	LargeSemedit          *BenchRunResult   `json:"large_semedit,omitempty"`
	LargeVerifiedBaseline *BenchRunResult   `json:"large_verified_baseline,omitempty"`
	LargeVerifiedSemedit  *BenchRunResult   `json:"large_verified_semedit,omitempty"`
}

// BenchReport captures an empirical report file.
type BenchReport struct {
	Timestamp   time.Time                 `json:"timestamp"`
	Runs        []*BenchRunResult         `json:"runs"`
	Comparisons []*BenchComparisonSummary `json:"comparisons,omitempty"`
}

func loadAllBenchmarkComparisons(rootDir string) ([]*BenchComparisonSummary, error) {
	resultsDir := filepath.Join(rootDir, "data", "benchmarks", "results")
	if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var comparisons []*BenchComparisonSummary
	err := filepath.WalkDir(resultsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// #nosec G304,G122 -- reading benchmark result JSON
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read benchmark file %s: %w", path, err)
		}

		var report BenchReport
		if err := json.Unmarshal(data, &report); err != nil {
			return fmt.Errorf("unmarshal benchmark json %s: %w", path, err)
		}

		for _, comp := range report.Comparisons {
			comp = filterPublishableBenchmarkComparison(comp)
			if comp == nil {
				continue
			}
			if comp.TxtarPath != "" {
				comp.TxtarProvenance = resolveTxtarProvenance(rootDir, comp.TxtarPath)
			}
			comparisons = append(comparisons, comp)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(comparisons, func(i, j int) bool {
		if comparisons[i].TaskID != comparisons[j].TaskID {
			return comparisons[i].TaskID < comparisons[j].TaskID
		}
		if comparisons[i].Target.String() != comparisons[j].Target.String() {
			return comparisons[i].Target.String() < comparisons[j].Target.String()
		}
		if comparisons[i].PromptVariant != comparisons[j].PromptVariant {
			return comparisons[i].PromptVariant < comparisons[j].PromptVariant
		}
		return displayMCPServerInstructions(comparisons[i].MCPServerInstructions, comparisons[i].LegacyMCPInstructions) <
			displayMCPServerInstructions(comparisons[j].MCPServerInstructions, comparisons[j].LegacyMCPInstructions)
	})

	return comparisons, nil
}

func filterPublishableBenchmarkComparison(comp *BenchComparisonSummary) *BenchComparisonSummary {
	if comp == nil {
		return nil
	}

	filtered := *comp
	filtered.SmallBaseline = publishableBenchmarkRun(comp.SmallBaseline)
	filtered.SmallSemedit = publishableBenchmarkRun(comp.SmallSemedit)
	filtered.SmallVerifiedBaseline = publishableBenchmarkRun(comp.SmallVerifiedBaseline)
	filtered.SmallVerifiedSemedit = publishableBenchmarkRun(comp.SmallVerifiedSemedit)
	filtered.LargeBaseline = publishableBenchmarkRun(comp.LargeBaseline)
	filtered.LargeSemedit = publishableBenchmarkRun(comp.LargeSemedit)
	filtered.LargeVerifiedBaseline = publishableBenchmarkRun(comp.LargeVerifiedBaseline)
	filtered.LargeVerifiedSemedit = publishableBenchmarkRun(comp.LargeVerifiedSemedit)

	if filtered.SmallBaseline == nil && filtered.SmallSemedit == nil &&
		filtered.SmallVerifiedBaseline == nil && filtered.SmallVerifiedSemedit == nil &&
		filtered.LargeBaseline == nil && filtered.LargeSemedit == nil &&
		filtered.LargeVerifiedBaseline == nil && filtered.LargeVerifiedSemedit == nil {
		return nil
	}
	return &filtered
}

func publishableBenchmarkRun(run *BenchRunResult) *BenchRunResult {
	if run == nil || run.Error != "" || run.Oracle == nil {
		return nil
	}
	return run
}

func resolveTxtarProvenance(repoRoot, txtarRelPath string) string {
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

func renderBenchmarksDoc(rootDir string) (string, error) {
	comparisons, err := loadAllBenchmarkComparisons(rootDir)
	if err != nil {
		return "", fmt.Errorf("load comparisons: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(`---
title: "Empirical Benchmarks"
description: "Empirical performance and token efficiency evaluation: Vanilla LLM vs Semedit MCP across models and context sizes."
icon: "speed"
draft: false
weight: 20
---

Empirical telemetry measuring wall-clock latency, token usage, tool turns, and compiler correctness when AI coding agents perform code modifications.

Every benchmark pairs **Vanilla LLM** (standard file editing tools) against **Semedit MCP** (deterministic AST compiler operations).

`)
	sb.WriteString(benchmarkToolCallCSS)
	sb.WriteString(`
## Benchmark Methodology & Transparency

* **Reproducibility & Provenance**: Test scenarios are defined in self-contained txtar archives. Links below resolve to commit-anchored GitHub source files for published commits, or cryptographic SHA-256 fingerprints for uncommitted local fixtures.
* **Multi-Level Correctness Oracle**: Each trial is graded across 4 validation levels:
  1. *Level 1 (Mutation Policy)*: Restricts file modifications strictly to authorized paths.
  2. *Level 2 (AST Invariants)*: Compiler AST verification of required symbols, imports, and relative declaration ordering.
  3. *Level 3 (Clean Build)*: Strict compilation validation with zero compiler or typecheck errors.
  4. *Level 4 (Test Suite)*: Unit and integration test suite execution.
* **Isolated Sandboxing**: Each trial executes in an ephemeral directory with pristine git and module state.

`)

	if len(comparisons) == 0 {
		sb.WriteString(`> [!NOTE]
> No benchmark evaluations are recorded yet in ` + "`data/benchmarks/results/`" + `. Run the benchmark harness using ` + "`go run ./tools/benchmark-harness --target ...`" + ` to record empirical results.
`)
		return sb.String(), nil
	}

	currentTaskID := ""
	currentTarget := ""
	for _, comp := range comparisons {
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
		fmt.Fprintf(&sb, "#### %s\n\n", formatDocConfiguration(comp))
		writeDocConfigurationDetails(&sb, comp)

		if comp.VanillaPrompt != "" && comp.MCPPrompt != "" && comp.VanillaPrompt == comp.MCPPrompt {
			writeDocPrompt(&sb, "LLM Prompt", comp.VanillaPrompt)
		} else {
			if comp.VanillaPrompt != "" {
				writeDocPrompt(&sb, "Vanilla LLM Prompt", comp.VanillaPrompt)
			}
			if comp.MCPPrompt != "" {
				writeDocPrompt(&sb, "Semedit MCP Prompt", comp.MCPPrompt)
			}
		}

		hasVerified := comp.SmallVerifiedBaseline != nil || comp.SmallVerifiedSemedit != nil ||
			comp.LargeVerifiedBaseline != nil || comp.LargeVerifiedSemedit != nil

		if hasVerified {
			if comp.VanillaVerifiedPrompt != "" && comp.MCPVerifiedPrompt != "" && comp.VanillaVerifiedPrompt == comp.MCPVerifiedPrompt {
				writeDocPrompt(&sb, "Verified Prompt", comp.VanillaVerifiedPrompt)
			} else {
				if comp.VanillaVerifiedPrompt != "" {
					writeDocPrompt(&sb, "Vanilla (Verified) Prompt", comp.VanillaVerifiedPrompt)
				}
				if comp.MCPVerifiedPrompt != "" {
					writeDocPrompt(&sb, "Semedit MCP (Verified) Prompt", comp.MCPVerifiedPrompt)
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
			renderDocComparisonTable(&sb, comp.SmallBaseline, comp.SmallSemedit, comp.LargeBaseline, comp.LargeSemedit)
			renderDocRunSummary(&sb, 5, "Standard vs Semedit in Small Context", comp.SmallBaseline, comp.SmallSemedit)
			renderDocSemanticToolDiagnostic(&sb, comp.SmallSemedit)
			renderDocRunSummary(&sb, 5, "Standard vs Semedit in Large Context", comp.LargeBaseline, comp.LargeSemedit)
			renderDocSemanticToolDiagnostic(&sb, comp.LargeSemedit)
		}

		// 2. Verified Directive Comparison Table
		if hasVerified {
			sb.WriteString("##### Verified Directive Comparison (+Self-Correction Loop)\n\n")
			renderDocComparisonTable(&sb, comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit, comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
			renderDocRunSummary(&sb, 6, "Verified vs Semedit in Small Context", comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit)
			renderDocSemanticToolDiagnostic(&sb, comp.SmallVerifiedSemedit)
			renderDocRunSummary(&sb, 6, "Verified vs Semedit in Large Context", comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
			renderDocSemanticToolDiagnostic(&sb, comp.LargeVerifiedSemedit)
		}
	}

	return sb.String(), nil
}

func writeDocPrompt(sb *strings.Builder, label, prompt string) {
	fmt.Fprintf(sb, "**%s**:\n%s\n\n", label, formatDocPromptBlockquote(prompt))
}

func formatDocPromptBlockquote(prompt string) string {
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

func formatDocConfiguration(comp *BenchComparisonSummary) string {
	promptVariant := strings.TrimSpace(comp.PromptVariant)
	if promptVariant == "" {
		promptVariant = "default"
	}
	return fmt.Sprintf("%s prompt · %s MCP instructions", promptVariant, displayMCPServerInstructions(comp.MCPServerInstructions, comp.LegacyMCPInstructions))
}

func writeDocConfigurationDetails(sb *strings.Builder, comp *BenchComparisonSummary) {
	promptVariant := strings.TrimSpace(comp.PromptVariant)
	if promptVariant == "" {
		promptVariant = "default"
	}

	sb.WriteString("| Setting | Value |\n")
	sb.WriteString("| :--- | :--- |\n")
	fmt.Fprintf(sb, "| Test case | `%s` |\n", comp.TaskID)
	fmt.Fprintf(sb, "| Target | `%s` |\n", comp.Target.String())
	fmt.Fprintf(sb, "| Prompt variant | `%s` |\n", promptVariant)
	fmt.Fprintf(sb, "| MCP server instructions | `%s` |\n", displayMCPServerInstructions(comp.MCPServerInstructions, comp.LegacyMCPInstructions))
	if provenance := displayProvenance(comp.Provenance, comp.LegacyClassifiers); provenance != "" {
		fmt.Fprintf(sb, "| Run provenance | `%s` |\n", provenance)
	}
	if fixture := formatDocFixture(comp); fixture != "" {
		fmt.Fprintf(sb, "| Fixture | %s |\n", fixture)
	}
	sb.WriteString("\n")
}

func formatDocFixture(comp *BenchComparisonSummary) string {
	switch {
	case comp.TxtarProvenance != "" && strings.HasPrefix(comp.TxtarProvenance, "http"):
		return fmt.Sprintf("[%s](%s)", comp.TxtarPath, comp.TxtarProvenance)
	case comp.TxtarProvenance != "":
		return fmt.Sprintf("`%s` (`%s`)", comp.TxtarPath, comp.TxtarProvenance)
	case comp.TxtarPath != "":
		return fmt.Sprintf("`%s`", comp.TxtarPath)
	default:
		return ""
	}
}

func displayMCPServerInstructions(mode, legacyMode string) string {
	if strings.TrimSpace(mode) == "" {
		mode = legacyMode
	}
	switch strings.TrimSpace(mode) {
	case "", "none":
		return "none"
	case "directive":
		return "prescriptive"
	default:
		return mode
	}
}

func displayProvenance(provenance, legacyClassifiers map[string]string) string {
	if len(provenance) == 0 {
		provenance = legacyClassifiers
	}
	if len(provenance) == 0 {
		return ""
	}
	keys := make([]string, 0, len(provenance))
	for key := range provenance {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+provenance[key])
	}
	return strings.Join(parts, ",")
}

func renderDocComparisonTable(sb *strings.Builder, sbRun, smRun, lbRun, lmRun *BenchRunResult) {
	fmt.Fprintf(sb, "| Metric | Vanilla (Small) | %s | Δ (Small) | Vanilla (Large) | %s | Δ (Large) |\n", docMCPColumnHeader("Small", smRun), docMCPColumnHeader("Large", lmRun))
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	sb.WriteString(formatDocMetricRowDuration("Wall-Clock Latency",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) time.Duration { return r.WallClock }))

	sb.WriteString(formatDocMetricRowOptionalDuration("Process Start → First Event",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) *time.Duration { return r.ProcessStartToFirstEvent }))

	sb.WriteString(formatDocMetricRowOptionalDuration("First Event → First Tool Call",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) *time.Duration { return r.FirstEventToFirstToolCall }))

	sb.WriteString(formatDocMetricRowOptionalDuration("MCP Initialize → First Semantic Call",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) *time.Duration { return r.MCPInitializeToFirstSemanticCall }))

	sb.WriteString(formatDocMetricRowOptionalDuration("MCP Server Start → Initialize",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) *time.Duration { return r.MCPServerStartToInitialize }))

	sb.WriteString(formatDocMetricRowInt("Top-Level User Turns",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.Turns }))

	sb.WriteString(formatDocMetricRowInt("Internal Tool Cycles",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.InternalTurns }))

	sb.WriteString(formatDocMetricRowInt("Initial Load / Discovery Turns",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.InitialLoadTurns }))

	sb.WriteString(formatDocMetricRowInt("MCP Discovery / Schema Turns",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.MCPLoadTurns }))

	sb.WriteString(formatDocMetricRowInt("Tool Invocations (per run)",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.ToolCount }))

	sb.WriteString(formatDocMetricRowInt("Output Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.OutputTokens }))

	sb.WriteString(formatDocMetricRowInt("Reasoning / Thinking Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.ReasoningTokens }))

	sb.WriteString(formatDocMetricRowInt("Total Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.PromptTokens }))

	sb.WriteString(formatDocMetricRowIntWithDirection("Cached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.CachedPromptTokens }, docDeltaHigherIsBetter))

	sb.WriteString(formatDocMetricRowInt("Uncached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.UncachedPromptTokens }))

	sb.WriteString(formatDocCachedToUncachedRatioRow(sbRun, smRun, lbRun, lmRun))

	sb.WriteString(formatDocOracleRow("Oracle L1: Mutation Policy",
		sbRun, smRun, lbRun, lmRun,
		func(o *BenchOracleResult) bool { return o.Level1Policy }))

	sb.WriteString(formatDocOracleRow("Oracle L2: AST Invariants",
		sbRun, smRun, lbRun, lmRun,
		func(o *BenchOracleResult) bool { return o.Level2AST }))

	sb.WriteString(formatDocOracleRow("Oracle L3: Clean Build",
		sbRun, smRun, lbRun, lmRun,
		func(o *BenchOracleResult) bool { return o.Level3Build }))

	sb.WriteString(formatDocOracleRow("Oracle L4: Verification Test",
		sbRun, smRun, lbRun, lmRun,
		func(o *BenchOracleResult) bool { return o.Level4Test }))

	sb.WriteString(formatDocMCPVerifiedRow("Semantic Tool Invocation Verified (per run)",
		sbRun, smRun, lbRun, lmRun))

	sb.WriteString("\n")
}

func docMCPColumnHeader(context string, run *BenchRunResult) string {
	label := fmt.Sprintf("MCP (%s)", context)
	if run == nil || run.MCPVerified {
		return label
	}
	return fmt.Sprintf("<span role=\"img\" aria-label=\"Semantic tool invocation not verified\" title=\"Semantic tool invocation not verified\">⚠</span> %s", label)
}

func renderDocRunSummary(sb *strings.Builder, headingLevel int, comparisonLabel string, base, mcp *BenchRunResult) {
	fmt.Fprintf(sb, "%s %s\n", strings.Repeat("#", headingLevel), comparisonLabel)
	renderDocEditSummary(sb, "Vanilla", comparisonLabel, base)
	renderDocEditSummary(sb, "Semedit MCP", comparisonLabel, mcp)
	sb.WriteString("\n")
	renderDocToolCallComparison(sb, base, mcp)
	sb.WriteString("\n")
}

func renderDocEditSummary(sb *strings.Builder, arm, contextLabel string, run *BenchRunResult) {
	if run != nil && run.Diff != "" {
		fmt.Fprintf(sb, "* **%s (%s) Edit**: %s\n", arm, contextLabel, run.Diff)
	}
}

func renderDocSemanticToolDiagnostic(sb *strings.Builder, run *BenchRunResult) {
	if run == nil || run.SemanticToolReflection == nil {
		return
	}
	reflection := run.SemanticToolReflection
	if reflection.Response != "" {
		renderDocSemanticToolCallout(sb, reflection.Response)
		return
	}
	if reflection.Error != "" {
		fmt.Fprintf(sb, "<p><strong>Semantic tool-use diagnostic unavailable:</strong> %s</p>\n\n", html.EscapeString(reflection.Error))
	}
}

func renderDocSemanticToolCallout(sb *strings.Builder, response string) {
	calloutClass, icon := "callout-warning", "⚠"
	if noSemanticToolsAvailablePattern.MatchString(response) {
		calloutClass, icon = "callout-error", "✕"
	}
	fmt.Fprintf(sb, "<div class=\"callout %s\"><div class=\"callout-title\"><span>%s</span> Why no semantic edit tool was used</div><div class=\"callout-desc\">%s</div></div>\n\n", calloutClass, icon, html.EscapeString(response))
}

func renderDocToolCallComparison(sb *strings.Builder, base, mcp *BenchRunResult) {
	if docRunNeedsHTMLTable(base) || docRunNeedsHTMLTable(mcp) {
		renderDocToolCallHTMLTable(sb, base, mcp)
		return
	}

	sb.WriteString("| # | Vanilla | Semedit MCP |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")
	rows := max(docToolCallCount(base), docToolCallCount(mcp), 1)
	for index := range rows {
		fmt.Fprintf(sb, "| %d | %s | %s |\n", index+1, docToolCallAt(base, index), docToolCallAt(mcp, index))
	}
}

func docRunNeedsHTMLTable(run *BenchRunResult) bool {
	if run == nil {
		return false
	}
	return slices.ContainsFunc(docReasoningToolCalls(run), func(call BenchToolCall) bool {
		return docToolCallIsShellCommand(call) || len(call.Arguments) > 0
	})
}

func renderDocToolCallHTMLTable(sb *strings.Builder, base, mcp *BenchRunResult) {
	sb.WriteString("<table class=\"benchmark-tool-calls\">\n<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>\n<tbody>\n")
	rows := max(docToolCallCount(base), docToolCallCount(mcp), 1)
	for index := range rows {
		fmt.Fprintf(sb, "<tr><td>%d</td><td>%s</td><td>%s</td></tr>\n", index+1, docToolCallHTMLCell(base, index), docToolCallHTMLCell(mcp, index))
	}
	sb.WriteString("</tbody>\n</table>\n")
}

func docToolCallCount(run *BenchRunResult) int {
	if run == nil {
		return 0
	}
	return max(len(docReasoningToolCalls(run)), len(run.ToolsUsed))
}

func docToolCallAt(run *BenchRunResult, index int) string {
	if run == nil {
		return "not published"
	}
	calls := docReasoningToolCalls(run)
	if index < len(calls) {
		call := calls[index]
		if docToolCallIsShellCommand(call) {
			return formatDocShellCommandCell(call)
		}
		server := ""
		if call.Server != "" {
			server = call.Server + "/"
		}
		entry := fmt.Sprintf("`%s%s` (%s)%s", server, call.Name, docToolCallStatusSummary(call), html.EscapeString(docToolCallDetails(call)))
		return escapeDocTableCell(entry)
	}
	if index < len(run.ToolsUsed) {
		return fmt.Sprintf("`%s` (outcome unavailable)", run.ToolsUsed[index])
	}
	if docToolCallCount(run) == 0 {
		return "none recorded"
	}
	return "—"
}

func docToolCallIsShellCommand(call BenchToolCall) bool {
	return call.Server == "" && (strings.HasPrefix(call.Name, "/bin/") || strings.HasPrefix(call.Name, "bash ") || strings.HasPrefix(call.Name, "zsh "))
}

func formatDocShellCommandCell(call BenchToolCall) string {
	command := formatDocCodeShortcode("shell", formatDocShellCommand(call.Name))
	return fmt.Sprintf("%s%s%s%s", docToolCallStatusSummary(call), html.EscapeString(docToolCallDetails(call)), command, formatDocToolArguments(call.Arguments))
}

func docToolCallHTMLCell(run *BenchRunResult, index int) string {
	if run == nil {
		return "not published"
	}
	calls := docReasoningToolCalls(run)
	if index < len(calls) {
		call := calls[index]
		if docToolCallIsShellCommand(call) {
			return formatDocShellCommandCell(call)
		}
		name := call.Name
		if call.Server != "" {
			name = call.Server + "/" + name
		}
		return fmt.Sprintf("<code>%s</code> (%s)%s%s", html.EscapeString(name), docToolCallStatusSummary(call), html.EscapeString(docToolCallDetails(call)), formatDocToolArguments(call.Arguments))
	}
	if index < len(run.ToolsUsed) {
		return fmt.Sprintf("<code>%s</code> (outcome unavailable)", html.EscapeString(run.ToolsUsed[index]))
	}
	if docToolCallCount(run) == 0 {
		return "none recorded"
	}
	return "—"
}

func formatDocToolArguments(arguments json.RawMessage) string {
	arguments = bytes.TrimSpace(arguments)
	if len(arguments) == 0 || bytes.Equal(arguments, []byte("null")) {
		return ""
	}

	var formatted bytes.Buffer
	if err := json.Indent(&formatted, arguments, "", "  "); err != nil {
		formatted.Write(arguments)
	}
	return formatDocCodeShortcode("json", formatted.String())
}

func formatDocCodeShortcode(language, content string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	return fmt.Sprintf(`{{< benchmark-code lang="%s" content="%s" >}}`, language, encoded)
}

func docReasoningToolCalls(run *BenchRunResult) []BenchToolCall {
	if run == nil {
		return nil
	}
	if len(run.InteractionSteps) == 0 {
		return run.ToolCalls
	}

	steps := slices.Clone(run.InteractionSteps)
	slices.SortStableFunc(steps, func(a, b BenchInteractionStep) int {
		switch {
		case a.Step < b.Step:
			return -1
		case a.Step > b.Step:
			return 1
		default:
			return 0
		}
	})

	var calls []BenchToolCall
	for _, step := range steps {
		calls = append(calls, step.ToolCalls...)
	}
	return calls
}

func docToolCallStatusSummary(call BenchToolCall) string {
	return fmt.Sprintf("%s %s", formatDocToolCallStatus("Transport", call.TransportStatus), formatDocToolCallStatus("Functional", call.FunctionalStatus))
}

func docToolCallDetails(call BenchToolCall) string {
	entry := ""
	if call.Failure != "" {
		entry += fmt.Sprintf(": %s", call.Failure)
	}
	return entry + formatDocMCPMetrics(call.MCPMetrics)
}

func formatDocToolCallStatus(kind, status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
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

func formatDocShellCommand(command string) string {
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

func formatDocMCPMetrics(metrics *BenchMCPMetrics) string {
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
		parts := make([]string, 0, 2)
		if verificationMS > 0 {
			parts = append(parts, fmt.Sprintf("verification: %s", time.Duration(verificationMS)*time.Millisecond))
		}
		if formattingMS > 0 {
			parts = append(parts, fmt.Sprintf("formatting: %s", time.Duration(formattingMS)*time.Millisecond))
		}
		entry += " (" + strings.Join(parts, "; ") + ")"
	}
	return entry
}

func escapeDocTableCell(value string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(value), " "), "|", "\\|")
}

func formatDocMetricRowInt(name string, sb, sm, lb, lm *BenchRunResult, get func(*BenchRunResult) int) string {
	return formatDocMetricRowIntWithDirection(name, sb, sm, lb, lm, get, docDeltaLowerIsBetter)
}

func formatDocMetricRowIntWithDirection(name string, sb, sm, lb, lm *BenchRunResult, get func(*BenchRunResult) int, direction docDeltaDirection) string {
	valSB, valSM, deltaS := computeDocIntDelta(sb, sm, get, direction)
	valLB, valLM, deltaL := computeDocIntDelta(lb, lm, get, direction)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatDocCachedToUncachedRatioRow(sb, sm, lb, lm *BenchRunResult) string {
	valSB, valSM, deltaS := computeDocCachedToUncachedRatioDelta(sb, sm)
	valLB, valLM, deltaL := computeDocCachedToUncachedRatioDelta(lb, lm)
	return fmt.Sprintf("| **Cached vs Uncached Token Ratio** | %s | %s | %s | %s | %s | %s |\n", valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatDocMetricRowDuration(name string, sb, sm, lb, lm *BenchRunResult, get func(*BenchRunResult) time.Duration) string {
	valSB, valSM, deltaS := computeDocDurationDelta(sb, sm, get, docDeltaLowerIsBetter)
	valLB, valLM, deltaL := computeDocDurationDelta(lb, lm, get, docDeltaLowerIsBetter)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatDocMetricRowOptionalDuration(name string, sb, sm, lb, lm *BenchRunResult, get func(*BenchRunResult) *time.Duration) string {
	format := func(run *BenchRunResult) string {
		if run == nil || get(run) == nil {
			return "—"
		}
		return fmt.Sprintf("%.2fs", get(run).Seconds())
	}
	return fmt.Sprintf("| **%s** | %s | %s | — | %s | %s | — |\n", name, format(sb), format(sm), format(lb), format(lm))
}

func formatDocOracleRow(name string, sb, sm, lb, lm *BenchRunResult, check func(*BenchOracleResult) bool) string {
	render := func(r *BenchRunResult) string {
		if r == nil {
			return "—"
		}
		if r.Oracle == nil {
			return "SKIP"
		}
		if check(r.Oracle) {
			return "PASS"
		}
		return "**FAIL**"
	}
	return fmt.Sprintf("| **%s** | %s | %s | — | %s | %s | — |\n",
		name, render(sb), render(sm), render(lb), render(lm))
}

func formatDocMCPVerifiedRow(name string, sb, sm, lb, lm *BenchRunResult) string {
	render := func(r *BenchRunResult) string {
		if r == nil {
			return "—"
		}
		if r.Arm == "baseline-diff" {
			return "N/A"
		}
		if r.MCPVerified {
			return "PASS"
		}
		return "**FAIL**"
	}
	return fmt.Sprintf("| **%s** | %s | %s | — | %s | %s | — |\n",
		name, render(sb), render(sm), render(lb), render(lm))
}

type docDeltaDirection bool

const (
	docDeltaLowerIsBetter  docDeltaDirection = false
	docDeltaHigherIsBetter docDeltaDirection = true
)

func computeDocIntDelta(base, mcp *BenchRunResult, get func(*BenchRunResult) int, direction docDeltaDirection) (string, string, string) {
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
		return bStr, mStr, "—"
	}

	diff := mVal - bVal
	pct := (float64(diff) / float64(bVal)) * 100.0
	return bStr, mStr, formatDocDelta(pct, float64(diff), direction, mcp.MCPVerified)
}

func computeDocCachedToUncachedRatioDelta(base, mcp *BenchRunResult) (string, string, string) {
	if base == nil && mcp == nil {
		return "—", "—", "—"
	}
	if base == nil {
		_, value, _ := docCachedToUncachedRatio(mcp)
		return "—", value, "—"
	}
	if mcp == nil {
		_, value, _ := docCachedToUncachedRatio(base)
		return value, "—", "—"
	}

	baseRatio, baseText, baseComparable := docCachedToUncachedRatio(base)
	mcpRatio, mcpText, mcpComparable := docCachedToUncachedRatio(mcp)
	if !baseComparable || !mcpComparable || baseRatio == 0 {
		return baseText, mcpText, "—"
	}
	diff := mcpRatio - baseRatio
	pct := diff / baseRatio * 100
	return baseText, mcpText, formatDocDelta(pct, diff, docDeltaHigherIsBetter, mcp.MCPVerified)
}

func docCachedToUncachedRatio(run *BenchRunResult) (float64, string, bool) {
	if run.UncachedPromptTokens <= 0 {
		if run.CachedPromptTokens > 0 {
			return 0, "∞:1", false
		}
		return 0, "—", false
	}
	ratio := float64(run.CachedPromptTokens) / float64(run.UncachedPromptTokens)
	return ratio, fmt.Sprintf("%.2f:1", ratio), true
}

func computeDocDurationDelta(base, mcp *BenchRunResult, get func(*BenchRunResult) time.Duration, direction docDeltaDirection) (string, string, string) {
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
	return bStr, mStr, formatDocDelta(pct, diff, direction, mcp.MCPVerified)
}

func formatDocDelta(pct float64, diff float64, direction docDeltaDirection, mcpVerified bool) string {
	if !mcpVerified {
		return "N/A"
	}
	if diff == 0 {
		return "0%"
	}
	delta := fmt.Sprintf("%+.1f%%", pct)
	mcpBenefits := (direction == docDeltaLowerIsBetter && diff < 0) || (direction == docDeltaHigherIsBetter && diff > 0)
	if mcpBenefits {
		return fmt.Sprintf("<span class=\"benchmark-delta-positive\">%s</span>", delta)
	}
	if !mcpBenefits {
		return fmt.Sprintf("<span class=\"benchmark-delta-negative\">%s</span>", delta)
	}
	return delta
}
