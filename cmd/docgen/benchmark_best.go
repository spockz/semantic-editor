// Package main renders selected benchmark outcomes and per-run summaries.
package main

import (
	"fmt"
	"slices"
	"strings"
)

type benchmarkPolicyReport struct {
	policy string
	best   []bestBenchmarkPair
	runs   []benchmarkDocumentationRun
}

func renderBestBenchmarksDoc(best []bestBenchmarkPair, runs []benchmarkDocumentationRun) string {
	var sb strings.Builder
	sb.WriteString("The benchmark runs below link to pages containing each run's detailed observations and a browser scoped to that run. Use the [cross-run browser](/docs/benchmarks/browser/) to explore all published runs, or see [aggregate metrics](/docs/benchmarks/aggregates/).\n\n")
	fmt.Fprintf(&sb, "Published runs: %d. Selected best-case pairs: %d.\n\n", len(runs), len(best))
	sb.WriteString("## Benchmark runs\n\n")
	for _, run := range runs {
		fmt.Fprintf(&sb, "- [%s](/docs/benchmarks/runs/%s/) (%d comparison records)\n", run.ID, run.ID, len(run.Comparisons))
	}
	if len(runs) == 0 {
		sb.WriteString("No publishable benchmark runs are available yet.\n")
	}
	return sb.String()
}

type benchmarkHeadlineMetric struct {
	reduction float64
	pair      bestBenchmarkPair
}

type benchmarkHeadlineSummary struct {
	bestSpeed                     *benchmarkHeadlineMetric
	bestToken                     *benchmarkHeadlineMetric
	initialBaselineFirstTimeRight int
	initialMCPFirstTimeRight      int
	initialPairCount              int
}

func summarizeBestBenchmarkPairs(best []bestBenchmarkPair, runs []benchmarkDocumentationRun) benchmarkHeadlineSummary {
	summary := benchmarkHeadlineSummary{}
	for _, pair := range best {
		if !pair.baseline.Oracle.Passed || !pair.semedit.Oracle.Passed {
			continue
		}
		if speedReduction, comparable := benchmarkReduction(pair.baseline.WallClock.Seconds(), pair.semedit.WallClock.Seconds()); comparable &&
			(summary.bestSpeed == nil || speedReduction > summary.bestSpeed.reduction) {
			summary.bestSpeed = &benchmarkHeadlineMetric{reduction: speedReduction, pair: pair}
		}
		if tokenReduction, comparable := benchmarkReduction(cacheAdjustedTokenUnits(pair.baseline), cacheAdjustedTokenUnits(pair.semedit)); comparable &&
			(summary.bestToken == nil || tokenReduction > summary.bestToken.reduction) {
			summary.bestToken = &benchmarkHeadlineMetric{reduction: tokenReduction, pair: pair}
		}
	}
	for _, run := range runs {
		for _, comparison := range run.Comparisons {
			for _, context := range []benchmarkPairContext{benchmarkPairSmall, benchmarkPairLarge} {
				baseline, semedit := benchmarkPairRuns(comparison, context)
				baselineOracle, baselineAvailable := initialBenchmarkOracle(baseline)
				semeditOracle, semeditAvailable := initialBenchmarkOracle(semedit)
				if !baselineAvailable || !semeditAvailable {
					continue
				}
				summary.initialPairCount++
				if baselineOracle.Passed {
					summary.initialBaselineFirstTimeRight++
				}
				if semeditOracle.Passed {
					summary.initialMCPFirstTimeRight++
				}
			}
		}
	}
	return summary
}

func initialBenchmarkOracle(run *BenchRunResult) (*BenchOracleResult, bool) {
	if run == nil || run.InteractiveMode == "staged" {
		return nil, false
	}
	for _, step := range run.InteractionSteps {
		if step.Step == 1 && step.Oracle != nil && step.Error == "" {
			return step.Oracle, true
		}
	}
	if len(run.InteractionSteps) == 0 && run.Turns == 1 && run.Oracle != nil {
		return run.Oracle, true
	}
	return nil, false
}

func benchmarkReduction(baseline, semedit float64) (float64, bool) {
	if baseline <= 0 || semedit >= baseline {
		return 0, false
	}
	return (baseline - semedit) / baseline, true
}

func renderBestBenchmarkPreamble(best []bestBenchmarkPair, runs []benchmarkDocumentationRun) string {
	var sb strings.Builder
	sb.WriteString("## Best measured improvements\n\n")
	for _, group := range partitionBenchmarkPolicies(best, runs) {
		fmt.Fprintf(&sb, "### Semedit restriction: %c%s%c\n\n", 96, group.policy, 96)
		sb.WriteString(renderBenchmarkPolicyHeadline(group.best, group.runs))
	}

	sb.WriteString("\n\nSpeed and token figures include only selected pairs where both Vanilla and MCP passed their oracle. “First-time right” compares the initial oracle pass rate across all publishable standard-context paired observations, excluding verified/self-correction contexts.\n\n## Best-case outcomes measured so far\n\nThis page presents the most beneficial complete Vanilla/MCP pair measured so far for each testcase, target, prompt variant, MCP-instruction mode, semedit restriction policy, and context variant. It is **best-case evidence, not an average**. Selection favors an MCP oracle pass over a failure, then relative wall-clock improvement when both arms pass. A model-cost improvement of at least 10× can outweigh a non-comparable speed regression; otherwise, lower model cost resolves speed ties within five percentage points. When costs are also within five percentage points, a Semedit one-shot completion wins. Cost uses the target's declared per-million-token credits for uncached input, cached input, reasoning, and visible output, and counts thinking tokens at the output rate.\n\n[Open the interactive benchmark browser](/docs/benchmarks/browser/). [View median, IQR, p90, max, and success-rate metrics](/docs/benchmarks/aggregates/). The complete observations remain available on the individual run pages below.\n\n")
	return sb.String()
}

func writeBenchmarkHeadlineMetric(sb *strings.Builder, name string, metric *benchmarkHeadlineMetric, suffix string) {
	if metric == nil {
		fmt.Fprintf(sb, "| %s | — | No selected pair where both arms passed |\n", name)
		return
	}
	pair := metric.pair
	fmt.Fprintf(sb, "| %s | %.1f%% %s | `%s` · %s · [%s](/docs/benchmarks/runs/%s/) |\n", name, metric.reduction*100, suffix, pair.comparison.TaskID, pair.context.label(), pair.runID, pair.runID)
}

type benchmarkCorrectiveTurnHistogram struct {
	vanilla [6]int
	mcp     [6]int
}

const maximumCorrectiveInteractiveTurns = 5

func renderCorrectiveTurnHistogram(runs []benchmarkDocumentationRun) string {
	histogram := summarizeCorrectiveTurns(runs)
	var sb strings.Builder
	sb.WriteString("\n## Required corrective turns\n\n")
	sb.WriteString("Each row counts paired observations by the number of follow-up prompts attempted after the initial task prompt. The count excludes verified/self-correction contexts.\n\n")
	sb.WriteString("| Required corrective turns | Vanilla count | Semedit MCP count |\n| ---: | ---: | ---: |\n")
	for turn := 0; turn <= maximumCorrectiveInteractiveTurns; turn++ {
		fmt.Fprintf(&sb, "| %d | %d | %d |\n", turn, histogram.vanilla[turn], histogram.mcp[turn])
	}
	sb.WriteString("\n")
	return sb.String()
}

func summarizeCorrectiveTurns(runs []benchmarkDocumentationRun) benchmarkCorrectiveTurnHistogram {
	histogram := benchmarkCorrectiveTurnHistogram{}
	for _, run := range runs {
		for _, comparison := range run.Comparisons {
			for _, context := range []benchmarkPairContext{benchmarkPairSmall, benchmarkPairLarge} {
				baseline, semedit := benchmarkPairRuns(comparison, context)
				if baseline == nil || semedit == nil {
					continue
				}
				if turns, available := correctiveInteractiveTurns(baseline); available {
					histogram.vanilla[turns]++
				}
				if turns, available := correctiveInteractiveTurns(semedit); available {
					histogram.mcp[turns]++
				}
			}
		}
	}
	return histogram
}

func correctiveInteractiveTurns(run *BenchRunResult) (int, bool) {
	if run == nil || run.InteractiveMode == "staged" {
		return 0, false
	}
	if len(run.InteractionSteps) > 0 {
		turns := 0
		for _, step := range run.InteractionSteps {
			if step.Step > 1 {
				turns++
			}
		}
		return min(turns, maximumCorrectiveInteractiveTurns), true
	}
	if run.Turns > 0 {
		return min(run.Turns-1, maximumCorrectiveInteractiveTurns), true
	}
	return 0, false
}

func renderBenchmarkRunDoc(run benchmarkDocumentationRun) string {
	preamble := fmt.Sprintf("This page contains exactly the publishable benchmark observations recorded in run `%s`; it does not select or aggregate them. [Return to the run index](/docs/benchmarks/) or [view aggregate metrics](/docs/benchmarks/aggregates/).\n\n## Interactive browser\n\n{{< benchmark-browser run=%q >}}\n\n## Detailed observations\n\n", run.ID, run.ID)
	return renderBenchmarkComparisonsDoc("Benchmark run "+run.ID, "Complete empirical benchmark observations for run "+run.ID+".", preamble, run.Comparisons)
}

func partitionBenchmarkPolicies(best []bestBenchmarkPair, runs []benchmarkDocumentationRun) []benchmarkPolicyReport {
	groups := make(map[string]*benchmarkPolicyReport)
	get := func(policy string) *benchmarkPolicyReport {
		name := displaySemeditArmRestriction(policy)
		if groups[name] == nil {
			groups[name] = &benchmarkPolicyReport{policy: name}
		}
		return groups[name]
	}
	for _, pair := range best {
		group := get(pair.comparison.SemeditArmRestrict)
		group.best = append(group.best, pair)
	}
	for _, run := range runs {
		partition := make(map[string][]*BenchComparisonSummary)
		for _, comparison := range run.Comparisons {
			group := get(comparison.SemeditArmRestrict)
			partition[group.policy] = append(partition[group.policy], comparison)
		}
		for policy, comparisons := range partition {
			groups[policy].runs = append(groups[policy].runs, benchmarkDocumentationRun{ID: run.ID, Comparisons: comparisons})
		}
	}
	if len(groups) == 0 {
		get("")
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	slices.Sort(names)
	result := make([]benchmarkPolicyReport, 0, len(names))
	for _, name := range names {
		result = append(result, *groups[name])
	}
	return result
}

func renderBenchmarkPolicyHeadline(best []bestBenchmarkPair, runs []benchmarkDocumentationRun) string {
	summary := summarizeBestBenchmarkPairs(best, runs)
	var sb strings.Builder
	sb.WriteString("| Measure | Result | Evidence |\n| :--- | :--- | :--- |\n")
	writeBenchmarkHeadlineMetric(&sb, "Best speed increase", summary.bestSpeed, "faster")
	writeBenchmarkHeadlineMetric(&sb, "Best token reduction", summary.bestToken, "fewer cache-adjusted token units")
	if summary.initialPairCount == 0 {
		sb.WriteString("| MCP first-time-right edits | — | No publishable standard-context first attempts recorded yet |\n")
	} else {
		baselineRate := float64(summary.initialBaselineFirstTimeRight) / float64(summary.initialPairCount) * 100
		mcpRate := float64(summary.initialMCPFirstTimeRight) / float64(summary.initialPairCount) * 100
		fmt.Fprintf(&sb, "| MCP first-time-right edits | MCP %d/%d (%.1f%%) vs Vanilla %d/%d (%.1f%%), %+.1f pp | Initial oracle pass across all publishable standard-context paired observations |\n", summary.initialMCPFirstTimeRight, summary.initialPairCount, mcpRate, summary.initialBaselineFirstTimeRight, summary.initialPairCount, baselineRate, mcpRate-baselineRate)
	}
	sb.WriteString(renderCorrectiveTurnHistogram(runs))

	return sb.String()
}
