// Package main selects comparable benchmark pairs and computes aggregate metrics.
package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type benchmarkPairContext uint8

const (
	benchmarkPairSmall benchmarkPairContext = iota
	benchmarkPairLarge
	benchmarkPairVerifiedSmall
	benchmarkPairVerifiedLarge
)

func (c benchmarkPairContext) label() string {
	switch c {
	case benchmarkPairSmall:
		return "standard small context"
	case benchmarkPairLarge:
		return "standard large context"
	case benchmarkPairVerifiedSmall:
		return "verified small context"
	case benchmarkPairVerifiedLarge:
		return "verified large context"
	default:
		return "unknown context"
	}
}

func benchmarkPairRuns(comp *BenchComparisonSummary, context benchmarkPairContext) (*BenchRunResult, *BenchRunResult) {
	switch context {
	case benchmarkPairSmall:
		return comp.SmallBaseline, comp.SmallSemedit
	case benchmarkPairLarge:
		return comp.LargeBaseline, comp.LargeSemedit
	case benchmarkPairVerifiedSmall:
		return comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit
	case benchmarkPairVerifiedLarge:
		return comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit
	default:
		return nil, nil
	}
}

func setBenchmarkPairRuns(comp *BenchComparisonSummary, context benchmarkPairContext, baseline, semedit *BenchRunResult) {
	switch context {
	case benchmarkPairSmall:
		comp.SmallBaseline, comp.SmallSemedit = baseline, semedit
	case benchmarkPairLarge:
		comp.LargeBaseline, comp.LargeSemedit = baseline, semedit
	case benchmarkPairVerifiedSmall:
		comp.SmallVerifiedBaseline, comp.SmallVerifiedSemedit = baseline, semedit
	case benchmarkPairVerifiedLarge:
		comp.LargeVerifiedBaseline, comp.LargeVerifiedSemedit = baseline, semedit
	}
}

type bestBenchmarkPair struct {
	runID      string
	comparison *BenchComparisonSummary
	context    benchmarkPairContext
	baseline   *BenchRunResult
	semedit    *BenchRunResult
}

const benchmarkRelativeSimilarityThreshold = 0.05

func selectBestBenchmarkComparisons(runs []benchmarkDocumentationRun) []bestBenchmarkPair {
	selected := make(map[string]bestBenchmarkPair)
	contexts := []benchmarkPairContext{benchmarkPairSmall, benchmarkPairLarge, benchmarkPairVerifiedSmall, benchmarkPairVerifiedLarge}
	for _, run := range runs {
		for _, comparison := range run.Comparisons {
			for _, context := range contexts {
				baseline, semedit := benchmarkPairRuns(comparison, context)
				if baseline == nil || semedit == nil {
					continue
				}
				candidate := bestBenchmarkPair{runID: run.ID, comparison: comparison, context: context, baseline: baseline, semedit: semedit}
				key := benchmarkAggregateIdentity(comparison) + "\x00" + context.label()
				current, exists := selected[key]
				if !exists || candidate.preferredTo(current) {
					selected[key] = candidate
				}
			}
		}
	}

	best := make([]bestBenchmarkPair, 0, len(selected))
	for _, pair := range selected {
		best = append(best, pair)
	}
	sort.Slice(best, func(i, j int) bool {
		left, right := best[i], best[j]
		if left.comparison.TaskID != right.comparison.TaskID {
			return left.comparison.TaskID < right.comparison.TaskID
		}
		if left.comparison.Target.String() != right.comparison.Target.String() {
			return left.comparison.Target.String() < right.comparison.Target.String()
		}
		if benchmarkAggregateIdentity(left.comparison) != benchmarkAggregateIdentity(right.comparison) {
			return benchmarkAggregateIdentity(left.comparison) < benchmarkAggregateIdentity(right.comparison)
		}
		return left.context < right.context
	})
	return best
}

func (candidate bestBenchmarkPair) preferredTo(current bestBenchmarkPair) bool {
	candidateOutcome := benchmarkOutcomeBenefit(candidate.baseline, candidate.semedit)
	currentOutcome := benchmarkOutcomeBenefit(current.baseline, current.semedit)
	if candidateOutcome != currentOutcome {
		return candidateOutcome > currentOutcome
	}

	candidateSpeed := benchmarkRelativeMetric(candidate.baseline.WallClock.Seconds(), candidate.semedit.WallClock.Seconds())
	currentSpeed := benchmarkRelativeMetric(current.baseline.WallClock.Seconds(), current.semedit.WallClock.Seconds())
	candidateCost, candidateCostAvailable := benchmarkRelativeCost(candidate.baseline, candidate.semedit)
	currentCost, currentCostAvailable := benchmarkRelativeCost(current.baseline, current.semedit)
	if math.Abs(candidateSpeed-currentSpeed) > benchmarkRelativeSimilarityThreshold {
		if candidateCostAvailable && currentCostAvailable && candidateCost <= currentCost/10 {
			return true
		}
		if candidateCostAvailable && currentCostAvailable && currentCost <= candidateCost/10 {
			return false
		}
		return candidateSpeed < currentSpeed
	}
	if candidateCostAvailable && currentCostAvailable && math.Abs(candidateCost-currentCost) > benchmarkRelativeSimilarityThreshold {
		return candidateCost < currentCost
	}
	if candidateCostAvailable && currentCostAvailable && candidate.semedit.Turns != current.semedit.Turns {
		return candidate.semedit.Turns == 1
	}
	return candidate.runID < current.runID
}

func benchmarkOutcomeBenefit(baseline, semedit *BenchRunResult) int {
	baselinePassed := baseline.Oracle.Passed
	semeditPassed := semedit.Oracle.Passed
	switch {
	case semeditPassed && !baselinePassed:
		return 3
	case semeditPassed && baselinePassed:
		return 2
	case !semeditPassed && !baselinePassed:
		return 1
	default:
		return 0
	}
}

func benchmarkRelativeMetric(baseline, semedit float64) float64 {
	if baseline == 0 {
		if semedit == 0 {
			return 1
		}
		return math.Inf(1)
	}
	return semedit / baseline
}

func cacheAdjustedTokenUnits(run *BenchRunResult) float64 {
	uncachedInput := benchmarkUncachedInputTokens(run)
	return float64(uncachedInput+run.OutputTokens+run.ReasoningTokens) + float64(run.CachedPromptTokens)/10
}

func benchmarkUncachedInputTokens(run *BenchRunResult) int {
	if run.UncachedPromptTokens == 0 && run.CachedPromptTokens == 0 {
		return run.PromptTokens
	}
	return run.UncachedPromptTokens
}

type benchmarkCostRates struct {
	input  float64
	cached float64
	output float64
}

func benchmarkCost(run *BenchRunResult) (float64, bool) {
	rates, ok := benchmarkCostRatesForModel(run.Target.Model)
	if !ok {
		return 0, false
	}
	return (float64(benchmarkUncachedInputTokens(run))*rates.input +
		float64(run.CachedPromptTokens)*rates.cached +
		float64(run.ReasoningTokens+run.OutputTokens)*rates.output) / 1_000_000, true
}

func benchmarkRelativeCost(baseline, semedit *BenchRunResult) (float64, bool) {
	baselineCost, baselineAvailable := benchmarkCost(baseline)
	semeditCost, semeditAvailable := benchmarkCost(semedit)
	if !baselineAvailable || !semeditAvailable {
		return 0, false
	}
	return benchmarkRelativeMetric(baselineCost, semeditCost), true
}

func benchmarkCostRatesForModel(model string) (benchmarkCostRates, bool) {
	normalized := strings.NewReplacer("_", "-", " ", "-", "/", "-").Replace(strings.ToLower(strings.TrimSpace(model)))
	switch normalized {
	case "gpt-6-astra":
		return benchmarkCostRates{input: 250, cached: 25, output: 1250}, true
	case "gpt-6-sol":
		return benchmarkCostRates{input: 50, cached: 5, output: 250}, true
	case "gpt-5.6-sol":
		return benchmarkCostRates{input: 100, cached: 10, output: 500}, true
	case "gpt-5.6-terra":
		return benchmarkCostRates{input: 50, cached: 5, output: 300}, true
	case "gpt-6-luna":
		return benchmarkCostRates{input: 2.5, cached: 0.25, output: 12.5}, true
	case "gpt-5.6-luna":
		return benchmarkCostRates{input: 5, cached: 0.5, output: 30}, true
	default:
		return benchmarkCostRates{}, false
	}
}

func benchmarkAggregateIdentity(comp *BenchComparisonSummary) string {
	prompt := strings.TrimSpace(comp.PromptVariant)
	if prompt == "" {
		prompt = "default"
	}
	return strings.Join([]string{
		comp.TaskID,
		comp.Target.String(),
		prompt,
		displayMCPServerInstructions(comp.MCPServerInstructions, comp.LegacyMCPInstructions),
	}, "\x00")
}

type benchmarkMetric struct {
	name  string
	unit  string
	value func(*BenchRunResult) (float64, bool)
}

type benchmarkMetricSummary struct {
	count int
	min   float64
	max   float64
	total float64
}

func (summary *benchmarkMetricSummary) add(value float64) {
	if summary.count == 0 {
		summary.min, summary.max = value, value
	} else {
		summary.min = min(summary.min, value)
		summary.max = max(summary.max, value)
	}
	summary.count++
	summary.total += value
}

func (summary benchmarkMetricSummary) average() float64 {
	return summary.total / float64(summary.count)
}

type benchmarkAggregateGroup struct {
	task         string
	target       string
	prompt       string
	instructions string
	context      string
	arm          string
	metrics      map[string]*benchmarkMetricSummary
}

func renderBenchmarkAggregatesDoc(comparisons []*BenchComparisonSummary) string {
	var sb strings.Builder
	sb.WriteString(`---
title: "Benchmark metric aggregates"
description: "Minimum, maximum, and average empirical benchmark telemetry for each experimental condition and arm."
icon: "chart-bar"
draft: false
weight: 21
---

## Metric ranges across benchmark runs

Every table holds one experimental condition, context variant, and arm. **N** is the count of publishable observations with that metric. Technical provenance remains audit metadata and does not split aggregation cells. Cached input is shown raw and also as **cache-adjusted token units**: uncached input + output + reasoning + cached input / 10. **Cost** is model-specific credits calculated from the per-million-token rates in ADR-0043: (uncached input × input credits + cached input × cached credits + (reasoning + visible output) × output credits) / 1,000,000. It is omitted when the target model has no declared rate schedule.

[Return to best-case outcomes](/docs/benchmarks/).

`)

	groups := aggregateBenchmarkMetrics(comparisons)
	if len(groups) == 0 {
		sb.WriteString("> [!NOTE]\n> No publishable benchmark observations are available to aggregate.\n")
		return sb.String()
	}

	metrics := benchmarkMetrics()
	for _, group := range groups {
		fmt.Fprintf(&sb, "### `%s` · `%s` · %s · `%s` arm\n\n", group.task, group.target, group.context, group.arm)
		fmt.Fprintf(&sb, "Prompt variant: `%s` · MCP server instructions: `%s`\n\n", group.prompt, group.instructions)
		sb.WriteString("| Metric | N | Min | Max | Average |\n| :--- | ---: | ---: | ---: | ---: |\n")
		for _, metric := range metrics {
			summary := group.metrics[metric.name]
			if summary == nil {
				continue
			}
			fmt.Fprintf(&sb, "| %s | %d | %s | %s | %s |\n", metric.name, summary.count, formatBenchmarkMetric(metric.unit, summary.min), formatBenchmarkMetric(metric.unit, summary.max), formatBenchmarkMetric(metric.unit, summary.average()))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func aggregateBenchmarkMetrics(comparisons []*BenchComparisonSummary) []benchmarkAggregateGroup {
	groups := make(map[string]*benchmarkAggregateGroup)
	for _, comparison := range comparisons {
		for _, context := range []benchmarkPairContext{benchmarkPairSmall, benchmarkPairLarge, benchmarkPairVerifiedSmall, benchmarkPairVerifiedLarge} {
			baseline, semedit := benchmarkPairRuns(comparison, context)
			for _, observation := range []struct {
				arm string
				run *BenchRunResult
			}{{arm: "Vanilla", run: baseline}, {arm: "Semedit MCP", run: semedit}} {
				if observation.run == nil {
					continue
				}
				prompt := strings.TrimSpace(comparison.PromptVariant)
				if prompt == "" {
					prompt = "default"
				}
				instructions := displayMCPServerInstructions(comparison.MCPServerInstructions, comparison.LegacyMCPInstructions)
				key := strings.Join([]string{benchmarkAggregateIdentity(comparison), context.label(), observation.arm}, "\x00")
				group := groups[key]
				if group == nil {
					group = &benchmarkAggregateGroup{
						task: comparison.TaskID, target: comparison.Target.String(), prompt: prompt, instructions: instructions,
						context: context.label(), arm: observation.arm, metrics: make(map[string]*benchmarkMetricSummary),
					}
					groups[key] = group
				}
				for _, metric := range benchmarkMetrics() {
					value, present := metric.value(observation.run)
					if !present {
						continue
					}
					summary := group.metrics[metric.name]
					if summary == nil {
						summary = &benchmarkMetricSummary{}
						group.metrics[metric.name] = summary
					}
					summary.add(value)
				}
			}
		}
	}

	result := make([]benchmarkAggregateGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		return strings.Join([]string{left.task, left.target, left.prompt, left.instructions, left.context, left.arm}, "\x00") <
			strings.Join([]string{right.task, right.target, right.prompt, right.instructions, right.context, right.arm}, "\x00")
	})
	return result
}

func benchmarkMetrics() []benchmarkMetric {
	integer := func(name string, value func(*BenchRunResult) int) benchmarkMetric {
		return benchmarkMetric{name: name, unit: "count", value: func(run *BenchRunResult) (float64, bool) { return float64(value(run)), true }}
	}
	duration := func(name string, value func(*BenchRunResult) time.Duration) benchmarkMetric {
		return benchmarkMetric{name: name, unit: "seconds", value: func(run *BenchRunResult) (float64, bool) { return value(run).Seconds(), true }}
	}
	optionalDuration := func(name string, value func(*BenchRunResult) *time.Duration) benchmarkMetric {
		return benchmarkMetric{name: name, unit: "seconds", value: func(run *BenchRunResult) (float64, bool) {
			if duration := value(run); duration != nil {
				return duration.Seconds(), true
			}
			return 0, false
		}}
	}
	boolean := func(name string, value func(*BenchRunResult) bool) benchmarkMetric {
		return benchmarkMetric{name: name, unit: "binary", value: func(run *BenchRunResult) (float64, bool) {
			if value(run) {
				return 1, true
			}
			return 0, true
		}}
	}

	return []benchmarkMetric{
		duration("Wall-clock latency", func(run *BenchRunResult) time.Duration { return run.WallClock }),
		optionalDuration("Process start to first event", func(run *BenchRunResult) *time.Duration { return run.ProcessStartToFirstEvent }),
		optionalDuration("First event to first tool call", func(run *BenchRunResult) *time.Duration { return run.FirstEventToFirstToolCall }),
		optionalDuration("MCP initialize to first semantic call", func(run *BenchRunResult) *time.Duration { return run.MCPInitializeToFirstSemanticCall }),
		optionalDuration("MCP server start to initialize", func(run *BenchRunResult) *time.Duration { return run.MCPServerStartToInitialize }),
		integer("Top-level user turns", func(run *BenchRunResult) int { return run.Turns }),
		integer("Initial load and discovery turns", func(run *BenchRunResult) int { return run.InitialLoadTurns }),
		integer("MCP discovery and schema turns", func(run *BenchRunResult) int { return run.MCPLoadTurns }),
		integer("Internal tool cycles", func(run *BenchRunResult) int { return run.InternalTurns }),
		integer("Tool invocations", func(run *BenchRunResult) int { return run.ToolCount }),
		integer("Initial context tokens", func(run *BenchRunResult) int { return run.InitialContextTokens }),
		integer("Total input tokens", func(run *BenchRunResult) int { return run.PromptTokens }),
		integer("Cached input tokens", func(run *BenchRunResult) int { return run.CachedPromptTokens }),
		integer("Uncached input tokens", func(run *BenchRunResult) int { return run.UncachedPromptTokens }),
		integer("Output tokens", func(run *BenchRunResult) int { return run.OutputTokens }),
		integer("Reasoning tokens", func(run *BenchRunResult) int { return run.ReasoningTokens }),
		{name: "Cache-adjusted token units", unit: "tokens", value: func(run *BenchRunResult) (float64, bool) { return cacheAdjustedTokenUnits(run), true }},
		{name: "Cost", unit: "cost", value: benchmarkCost},
		optionalDuration("Oracle duration", func(run *BenchRunResult) *time.Duration {
			if run.Oracle == nil {
				return nil
			}
			return &run.Oracle.Duration
		}),
		boolean("Agent success (1=true)", func(run *BenchRunResult) bool { return run.Success }),
		boolean("Oracle passed (1=true)", func(run *BenchRunResult) bool { return run.Oracle != nil && run.Oracle.Passed }),
		boolean("Oracle L1 mutation policy (1=true)", func(run *BenchRunResult) bool { return run.Oracle != nil && run.Oracle.Level1Policy }),
		boolean("Oracle L2 AST invariants (1=true)", func(run *BenchRunResult) bool { return run.Oracle != nil && run.Oracle.Level2AST }),
		boolean("Oracle L3 clean build (1=true)", func(run *BenchRunResult) bool { return run.Oracle != nil && run.Oracle.Level3Build }),
		boolean("Oracle L4 verification test (1=true)", func(run *BenchRunResult) bool { return run.Oracle != nil && run.Oracle.Level4Test }),
	}
}

func formatBenchmarkMetric(unit string, value float64) string {
	switch unit {
	case "seconds":
		return fmt.Sprintf("%.2fs", value)
	case "count":
		return fmt.Sprintf("%.2f", value)
	case "tokens":
		return fmt.Sprintf("%.1f", value)
	case "binary":
		return fmt.Sprintf("%.2f", value)
	case "cost":
		return fmt.Sprintf("%.4f", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}
