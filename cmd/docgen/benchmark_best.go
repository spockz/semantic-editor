// Package main renders selected benchmark outcomes and per-run summaries.
package main

import (
	"fmt"
	"strings"
)

func renderBestBenchmarksDoc(best []bestBenchmarkPair, runs []benchmarkDocumentationRun) string {
	comparisons := make([]*BenchComparisonSummary, 0, len(best))
	for _, pair := range best {
		comparison := *pair.comparison
		comparison.SelectedRunID = pair.runID
		comparison.SmallBaseline, comparison.SmallSemedit = nil, nil
		comparison.LargeBaseline, comparison.LargeSemedit = nil, nil
		comparison.SmallVerifiedBaseline, comparison.SmallVerifiedSemedit = nil, nil
		comparison.LargeVerifiedBaseline, comparison.LargeVerifiedSemedit = nil, nil
		setBenchmarkPairRuns(&comparison, pair.context, pair.baseline, pair.semedit)
		comparisons = append(comparisons, &comparison)
	}

	preamble := renderBestBenchmarkPreamble(best, runs)
	page := renderBenchmarkComparisonsDoc("Empirical Benchmarks", "Best-case measured Vanilla LLM versus Semedit MCP outcomes, with complete per-run evidence and aggregate statistics.", preamble, comparisons)
	if len(runs) == 0 {
		return page
	}

	var links strings.Builder
	links.WriteString("\n## Individual benchmark runs\n\n")
	for _, run := range runs {
		fmt.Fprintf(&links, "- [%s](/docs/benchmarks/runs/%s/)\n", run.ID, run.ID)
	}
	return page + links.String()
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
	if run == nil {
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
	summary := summarizeBestBenchmarkPairs(best, runs)
	var sb strings.Builder
	sb.WriteString("## Best measured improvements\n\n")
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

	sb.WriteString(`

Speed and token figures include only selected pairs where both Vanilla and MCP passed their oracle. “First-time right” compares the initial oracle pass rate across all publishable standard-context paired observations, excluding verified/self-correction contexts.

## Best-case outcomes measured so far

This page presents the most beneficial complete Vanilla/MCP pair measured so far for each testcase, target, prompt variant, MCP-instruction mode, and context variant. It is **best-case evidence, not an average**. Selection favors an MCP oracle pass over a failure, then relative wall-clock improvement when both arms pass. A model-cost improvement of at least 10× can outweigh a non-comparable speed regression; otherwise, lower model cost resolves speed ties within five percentage points. When costs are also within five percentage points, a Semedit one-shot completion wins. Cost uses the target's declared per-million-token credits for uncached input, cached input, reasoning, and visible output, and counts thinking tokens at the output rate.

[Open the interactive benchmark browser](/docs/benchmarks/browser/). [View min, max, and average metrics](/docs/benchmarks/aggregates/). The complete observations remain available on the individual run pages below.

`)
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
	if run == nil {
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
	preamble := fmt.Sprintf("This page contains exactly the publishable benchmark observations recorded in run `%s`; it does not select or aggregate them. [Return to best-case outcomes](/docs/benchmarks/) or [view aggregate metrics](/docs/benchmarks/aggregates/).\n\n", run.ID)
	return renderBenchmarkComparisonsDoc("Benchmark run "+run.ID, "Complete empirical benchmark observations for run "+run.ID+".", preamble, run.Comparisons)
}
