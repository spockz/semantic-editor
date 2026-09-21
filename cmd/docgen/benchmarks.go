// Package main synthesizes code-derived capability documentation and empirical benchmark results into a Hugo source tree.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	TaskID                string             `json:"task_id"`
	Variant               string             `json:"variant,omitempty"`
	PromptVariant         string             `json:"prompt_variant,omitempty"`
	MCPServerInstructions string             `json:"mcp_server_instructions,omitempty"`
	LegacyMCPInstructions string             `json:"mcp_instructions,omitempty"`
	Provenance            map[string]string  `json:"provenance,omitempty"`
	LegacyClassifiers     map[string]string  `json:"classifiers,omitempty"`
	Target                BenchTarget        `json:"target"`
	Arm                   string             `json:"arm"`
	Success               bool               `json:"success"`
	Turns                 int                `json:"turns"`
	InitialLoadTurns      int                `json:"initial_load_turns"`
	MCPLoadTurns          int                `json:"mcp_load_turns"`
	InternalTurns         int                `json:"internal_turns"`
	ToolCount             int                `json:"tool_count"`
	WallClock             time.Duration      `json:"wall_clock_ms"`
	InitialContextTokens  int                `json:"initial_context_tokens"`
	PromptTokens          int                `json:"prompt_tokens"`
	CachedPromptTokens    int                `json:"cached_prompt_tokens"`
	UncachedPromptTokens  int                `json:"uncached_prompt_tokens"`
	OutputTokens          int                `json:"output_tokens"`
	ReasoningTokens       int                `json:"reasoning_tokens"`
	Oracle                *BenchOracleResult `json:"oracle"`
	Prompt                string             `json:"prompt,omitempty"`
	BeforeState           string             `json:"before_state,omitempty"`
	Diff                  string             `json:"diff,omitempty"`
	ToolsUsed             []string           `json:"tools_used,omitempty"`
	ToolCalls             []BenchToolCall    `json:"tool_calls,omitempty"`
	MCPVerified           bool               `json:"mcp_verified"`
	Error                 string             `json:"error,omitempty"`
}

// BenchToolCall captures a tool outcome from a benchmark report without importing the harness package.
type BenchToolCall struct {
	Name             string           `json:"name"`
	Server           string           `json:"server,omitempty"`
	TransportStatus  string           `json:"transport_status"`
	FunctionalStatus string           `json:"functional_status"`
	Failure          string           `json:"failure,omitempty"`
	MCPMetrics       *BenchMCPMetrics `json:"mcp_metrics,omitempty"`
}

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
		if comparisons[i].PromptVariant != comparisons[j].PromptVariant {
			return comparisons[i].PromptVariant < comparisons[j].PromptVariant
		}
		return comparisons[i].Target.String() < comparisons[j].Target.String()
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

# Empirical Benchmarks

Empirical telemetry measuring wall-clock latency, token usage, tool turns, and compiler correctness when AI coding agents perform code modifications.

Every benchmark pairs **Vanilla LLM** (standard file editing tools) against **Semedit MCP** (deterministic AST compiler operations).

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

	for _, comp := range comparisons {
		targetStr := comp.Target.String()
		if comp.PromptVariant != "" {
			fmt.Fprintf(&sb, "## Task: `%s` (Prompt: `%s`) | Target: `%s`\n\n", comp.TaskID, comp.PromptVariant, targetStr)
		} else {
			fmt.Fprintf(&sb, "## Task: `%s` | Target: `%s`\n\n", comp.TaskID, targetStr)
		}
		fmt.Fprintf(&sb, "* **MCP Server Instructions**: `%s`\n\n", displayMCPServerInstructions(comp.MCPServerInstructions, comp.LegacyMCPInstructions))
		if provenance := displayProvenance(comp.Provenance, comp.LegacyClassifiers); provenance != "" {
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
			fmt.Fprintf(&sb, "**LLM Prompt**:\n> %s\n\n", comp.VanillaPrompt)
		} else {
			if comp.VanillaPrompt != "" {
				fmt.Fprintf(&sb, "**Vanilla LLM Prompt**:\n> %s\n\n", comp.VanillaPrompt)
			}
			if comp.MCPPrompt != "" {
				fmt.Fprintf(&sb, "**Semedit MCP Prompt**:\n> %s\n\n", comp.MCPPrompt)
			}
		}

		hasVerified := comp.SmallVerifiedBaseline != nil || comp.SmallVerifiedSemedit != nil ||
			comp.LargeVerifiedBaseline != nil || comp.LargeVerifiedSemedit != nil

		if hasVerified {
			if comp.VanillaVerifiedPrompt != "" && comp.MCPVerifiedPrompt != "" && comp.VanillaVerifiedPrompt == comp.MCPVerifiedPrompt {
				fmt.Fprintf(&sb, "**Verified Prompt**:\n> %s\n\n", comp.VanillaVerifiedPrompt)
			} else {
				if comp.VanillaVerifiedPrompt != "" {
					fmt.Fprintf(&sb, "**Vanilla (Verified) Prompt**:\n> %s\n\n", comp.VanillaVerifiedPrompt)
				}
				if comp.MCPVerifiedPrompt != "" {
					fmt.Fprintf(&sb, "**Semedit MCP (Verified) Prompt**:\n> %s\n\n", comp.MCPVerifiedPrompt)
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
			if hasVerified {
				sb.WriteString("### Standard Directive Comparison\n\n")
			}
			renderDocComparisonTable(&sb, comp.SmallBaseline, comp.SmallSemedit, comp.LargeBaseline, comp.LargeSemedit)
			renderDocRunSummary(&sb, "Standard / Small Context", comp.SmallBaseline, comp.SmallSemedit)
			renderDocRunSummary(&sb, "Standard / Large Context", comp.LargeBaseline, comp.LargeSemedit)
		}

		// 2. Verified Directive Comparison Table
		if hasVerified {
			sb.WriteString("### Verified Directive Comparison (+Self-Correction Loop)\n\n")
			renderDocComparisonTable(&sb, comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit, comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
			renderDocRunSummary(&sb, "Verified / Small Context", comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit)
			renderDocRunSummary(&sb, "Verified / Large Context", comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
		}

		sb.WriteString("\n---\n\n")
	}

	return sb.String(), nil
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
	sb.WriteString("| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	sb.WriteString(formatDocMetricRowDuration("Wall-Clock Latency",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) time.Duration { return r.WallClock }))

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

	sb.WriteString(formatDocMetricRowInt("Cached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.CachedPromptTokens }))

	sb.WriteString(formatDocMetricRowInt("Uncached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *BenchRunResult) int { return r.UncachedPromptTokens }))

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

func renderDocRunSummary(sb *strings.Builder, variantLabel string, base, mcp *BenchRunResult) {
	fmt.Fprintf(sb, "#### Variant: %s\n", variantLabel)
	renderDocEditSummary(sb, "Vanilla", variantLabel, base)
	renderDocEditSummary(sb, "Semedit MCP", variantLabel, mcp)
	sb.WriteString("\n")
	renderDocToolCallComparison(sb, base, mcp)
	sb.WriteString("\n")
}

func renderDocEditSummary(sb *strings.Builder, arm, contextLabel string, run *BenchRunResult) {
	if run != nil && run.Diff != "" {
		fmt.Fprintf(sb, "* **%s (%s) Edit**: %s\n", arm, contextLabel, run.Diff)
	}
}

func renderDocToolCallComparison(sb *strings.Builder, base, mcp *BenchRunResult) {
	sb.WriteString("| # | Vanilla | Semedit MCP |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")
	rows := max(docToolCallCount(base), docToolCallCount(mcp), 1)
	for index := range rows {
		fmt.Fprintf(sb, "| %d | %s | %s |\n", index+1, docToolCallAt(base, index), docToolCallAt(mcp, index))
	}
}

func docToolCallCount(run *BenchRunResult) int {
	if run == nil {
		return 0
	}
	return max(len(run.ToolCalls), len(run.ToolsUsed))
}

func docToolCallAt(run *BenchRunResult, index int) string {
	if run == nil {
		return "not published"
	}
	if index < len(run.ToolCalls) {
		call := run.ToolCalls[index]
		server := ""
		if call.Server != "" {
			server = call.Server + "/"
		}
		entry := fmt.Sprintf("`%s%s` (transport: %s; functional: %s)", server, call.Name, call.TransportStatus, call.FunctionalStatus)
		if call.Failure != "" {
			entry += fmt.Sprintf(": %s", call.Failure)
		}
		entry += formatDocMCPMetrics(call.MCPMetrics)
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
	valSB, valSM, deltaS := computeDocIntDelta(sb, sm, get)
	valLB, valLM, deltaL := computeDocIntDelta(lb, lm, get)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatDocMetricRowDuration(name string, sb, sm, lb, lm *BenchRunResult, get func(*BenchRunResult) time.Duration) string {
	valSB, valSM, deltaS := computeDocDurationDelta(sb, sm, get)
	valLB, valLM, deltaL := computeDocDurationDelta(lb, lm, get)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
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

func computeDocIntDelta(base, mcp *BenchRunResult, get func(*BenchRunResult) int) (string, string, string) {
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
	var delta string
	switch {
	case diff < 0:
		delta = fmt.Sprintf("**%.1f%%**", pct)
	case diff > 0:
		delta = fmt.Sprintf("+%.1f%%", pct)
	default:
		delta = "0%"
	}
	return bStr, mStr, delta
}

func computeDocDurationDelta(base, mcp *BenchRunResult, get func(*BenchRunResult) time.Duration) (string, string, string) {
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
	var delta string
	switch {
	case diff < 0:
		delta = fmt.Sprintf("**%.1f%%**", pct)
	case diff > 0:
		delta = fmt.Sprintf("+%.1f%%", pct)
	default:
		delta = "0%"
	}
	return bStr, mStr, delta
}
