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

// BenchmarkReport captures empirical comparison results across targets, tasks, variants, and arms.
type BenchmarkReport struct {
	Timestamp   time.Time            `json:"timestamp"`
	Runs        []*RunResult         `json:"runs"`
	Comparisons []*ComparisonSummary `json:"comparisons,omitempty"`
}

// ComparisonSummary bundles Baseline vs MCP runs for a specific Task and Target across variants.
type ComparisonSummary struct {
	TaskID                string     `json:"task_id"`
	PromptVariant         string     `json:"prompt_variant,omitempty"`
	TxtarPath             string     `json:"txtar_path,omitempty"`
	TxtarProvenance       string     `json:"txtar_provenance,omitempty"`
	Target                Target     `json:"target"`
	VanillaPrompt         string     `json:"vanilla_prompt,omitempty"`
	MCPPrompt             string     `json:"mcp_prompt,omitempty"`
	VanillaVerifiedPrompt string     `json:"vanilla_verified_prompt,omitempty"`
	MCPVerifiedPrompt     string     `json:"mcp_verified_prompt,omitempty"`
	BeforeState           string     `json:"before_state,omitempty"`
	SmallBaseline         *RunResult `json:"small_baseline,omitempty"`
	SmallSemedit          *RunResult `json:"small_semedit,omitempty"`
	SmallVerifiedBaseline *RunResult `json:"small_verified_baseline,omitempty"`
	SmallVerifiedSemedit  *RunResult `json:"small_verified_semedit,omitempty"`
	LargeBaseline         *RunResult `json:"large_baseline,omitempty"`
	LargeSemedit          *RunResult `json:"large_semedit,omitempty"`
	LargeVerifiedBaseline *RunResult `json:"large_verified_baseline,omitempty"`
	LargeVerifiedSemedit  *RunResult `json:"large_verified_semedit,omitempty"`
}

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
		baseTask      string
		target        string
		promptVariant string
	}

	grouped := make(map[key]*ComparisonSummary)

	for _, r := range runs {
		baseTask := normalizeTaskBase(r.TaskID)
		k := key{baseTask: baseTask, target: r.Target.String(), promptVariant: r.PromptVariant}
		comp, exists := grouped[k]
		if !exists {
			comp = &ComparisonSummary{
				TaskID:        baseTask,
				PromptVariant: r.PromptVariant,
				Target:        r.Target,
				BeforeState:   r.BeforeState,
			}
			grouped[k] = comp
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
		if result[i].PromptVariant != result[j].PromptVariant {
			return result[i].PromptVariant < result[j].PromptVariant
		}
		return result[i].Target.String() < result[j].Target.String()
	})

	return result
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
	fmt.Fprintf(&sb, "* **Date**: %s\n\n", rep.Timestamp.Format("2006-01-02 15:04:05 MST"))

	for _, comp := range rep.Comparisons {
		targetStr := comp.Target.String()
		if comp.PromptVariant != "" {
			fmt.Fprintf(&sb, "## Task: `%s` (Prompt: `%s`) | Target: `%s`\n\n", comp.TaskID, comp.PromptVariant, targetStr)
		} else {
			fmt.Fprintf(&sb, "## Task: `%s` | Target: `%s`\n\n", comp.TaskID, targetStr)
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
			renderComparisonTable(&sb, comp.SmallBaseline, comp.SmallSemedit, comp.LargeBaseline, comp.LargeSemedit)
			renderDiffSection(&sb, "Small Context Edit Summary", comp.SmallBaseline, comp.SmallSemedit)
			renderDiffSection(&sb, "Large Context Edit Summary", comp.LargeBaseline, comp.LargeSemedit)
		}

		// 2. Verified Directive Comparison Table
		if hasVerified {
			sb.WriteString("### Verified Directive Comparison (+Self-Correction Loop)\n\n")
			renderComparisonTable(&sb, comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit, comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
			renderDiffSection(&sb, "Small (+Verified) Context Edit Summary", comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit)
			renderDiffSection(&sb, "Large (+Verified) Context Edit Summary", comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit)
		}

		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

func renderComparisonTable(sb *strings.Builder, sbRun, smRun, lbRun, lmRun *RunResult) {
	sb.WriteString("| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	// 1. Wall-Clock Latency
	sb.WriteString(formatMetricRowDuration("Wall-Clock Latency",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) time.Duration { return r.WallClock }))

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
	sb.WriteString(formatMetricRowInt("Cached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.CachedPromptTokens }))

	// 7. Uncached Input Tokens
	sb.WriteString(formatMetricRowInt("Uncached Input Tokens",
		sbRun, smRun, lbRun, lmRun,
		func(r *RunResult) int { return r.UncachedPromptTokens }))

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

func renderDiffSection(sb *strings.Builder, title string, base, mcp *RunResult) {
	if (base == nil || (base.Diff == "" && len(base.ToolsUsed) == 0)) && (mcp == nil || (mcp.Diff == "" && len(mcp.ToolsUsed) == 0)) {
		return
	}
	fmt.Fprintf(sb, "#### %s\n", title)
	if base != nil {
		if base.Diff != "" {
			fmt.Fprintf(sb, "* **Vanilla Edit**: %s\n", base.Diff)
		}
		if len(base.ToolsUsed) > 0 {
			fmt.Fprintf(sb, "* **Vanilla Tools**: `%s`\n", strings.Join(base.ToolsUsed, "`, `"))
		}
	}
	if mcp != nil {
		if mcp.Diff != "" {
			fmt.Fprintf(sb, "* **MCP Edit**: %s\n", mcp.Diff)
		}
		if len(mcp.ToolsUsed) > 0 {
			fmt.Fprintf(sb, "* **MCP Tools**: `%s`\n", strings.Join(mcp.ToolsUsed, "`, `"))
		}
	}
	sb.WriteString("\n")
}

func formatMetricRowInt(name string, sb, sm, lb, lm *RunResult, get func(*RunResult) int) string {
	valSB, valSM, deltaS := computeIntDelta(sb, sm, get)
	valLB, valLM, deltaL := computeIntDelta(lb, lm, get)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
}

func formatMetricRowDuration(name string, sb, sm, lb, lm *RunResult, get func(*RunResult) time.Duration) string {
	valSB, valSM, deltaS := computeDurationDelta(sb, sm, get)
	valLB, valLM, deltaL := computeDurationDelta(lb, lm, get)

	return fmt.Sprintf("| **%s** | %s | %s | %s | %s | %s | %s |\n",
		name, valSB, valSM, deltaS, valLB, valLM, deltaL)
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

func computeIntDelta(base, mcp *RunResult, get func(*RunResult) int) (string, string, string) {
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
		return bStr, mStr, "+100%"
	}

	diff := mVal - bVal
	pct := float64(diff) / float64(bVal) * 100.0
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

func computeDurationDelta(base, mcp *RunResult, get func(*RunResult) time.Duration) (string, string, string) {
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
