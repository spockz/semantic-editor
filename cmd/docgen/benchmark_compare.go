// Package main selects comparable benchmark pairs and computes aggregate metrics.
package main

import (
	"crypto/sha256"
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
	benchmarkPairExplicitSmall
	benchmarkPairExplicitLarge
	benchmarkPairExplicitVerifiedSmall
	benchmarkPairExplicitVerifiedLarge
)

const benchmarkAggregateArmBaseline = "baseline"

const benchmarkAggregateArmSemeditImplicit = "semedit-implicit"

const benchmarkAggregateArmSemeditExplicit = "semedit-explicit"

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
	case benchmarkPairExplicitSmall:
		return "semedit explicit small context"
	case benchmarkPairExplicitLarge:
		return "semedit explicit large context"
	case benchmarkPairExplicitVerifiedSmall:
		return "verified semedit explicit small context"
	case benchmarkPairExplicitVerifiedLarge:
		return "verified semedit explicit large context"
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
	case benchmarkPairExplicitSmall:
		return comp.SmallBaseline, comp.SmallSemeditExplicit
	case benchmarkPairExplicitLarge:
		return comp.LargeBaseline, comp.LargeSemeditExplicit
	case benchmarkPairExplicitVerifiedSmall:
		return comp.SmallVerifiedBaseline, comp.SmallVerifiedExplicit
	case benchmarkPairExplicitVerifiedLarge:
		return comp.LargeVerifiedBaseline, comp.LargeVerifiedExplicit
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
	case benchmarkPairExplicitSmall:
		comp.SmallBaseline, comp.SmallSemeditExplicit = baseline, semedit
	case benchmarkPairExplicitLarge:
		comp.LargeBaseline, comp.LargeSemeditExplicit = baseline, semedit
	case benchmarkPairExplicitVerifiedSmall:
		comp.SmallVerifiedBaseline, comp.SmallVerifiedExplicit = baseline, semedit
	case benchmarkPairExplicitVerifiedLarge:
		comp.LargeVerifiedBaseline, comp.LargeVerifiedExplicit = baseline, semedit
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
	contexts := []benchmarkPairContext{benchmarkPairSmall, benchmarkPairLarge, benchmarkPairVerifiedSmall, benchmarkPairVerifiedLarge, benchmarkPairExplicitSmall, benchmarkPairExplicitLarge, benchmarkPairExplicitVerifiedSmall, benchmarkPairExplicitVerifiedLarge}
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
		displaySemeditArmRestriction(comp.SemeditArmRestrict),
		prompt,
		displayMCPServerInstructions(comp.MCPServerInstructions, comp.LegacyMCPInstructions),
		comp.VanillaPrompt,
		comp.MCPPrompt,
		comp.VanillaVerifiedPrompt,
		comp.MCPVerifiedPrompt,
	}, "\x00")
}

type benchmarkMetric struct {
	name  string
	unit  string
	value func(*BenchRunResult) (float64, bool)
}

type benchmarkMetricSummary struct {
	count  int
	values []float64
}

func (summary *benchmarkMetricSummary) add(value float64) {
	summary.count++
	summary.values = append(summary.values, value)
}

func (summary benchmarkMetricSummary) quantile(p float64) float64 {
	if len(summary.values) == 0 {
		return 0
	}
	p = max(0, min(1, p))
	values := append([]float64(nil), summary.values...)
	sort.Float64s(values)
	position := p * float64(len(values)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return values[lower]
	}
	fraction := position - float64(lower)
	return values[lower] + fraction*(values[upper]-values[lower])
}

func (summary benchmarkMetricSummary) median() float64 {
	return summary.quantile(0.5)
}

type benchmarkAggregateGroup struct {
	restriction          string
	task                 string
	target               string
	prompt               string
	instructions         string
	context              string
	arm                  string
	sessionCount         int
	totalInputTokens     int
	totalOutputTokens    int
	totalReasoningTokens int
	totalCost            float64
	costKnown            bool
	metrics              map[string]*benchmarkMetricSummary
}

type benchmarkArmKey struct {
	task                string
	target              BenchTarget
	restriction         string
	promptVariant       string
	promptConfiguration string
	context             benchmarkPairContext
	arm                 string
	variant             string
	runInstructions     string
	interactiveMode     string
	prompt              string
}

func renderBenchmarkAggregatesDoc(comparisons []*BenchComparisonSummary) string {
	groups := aggregateBenchmarkMetrics(comparisons)
	var sb strings.Builder
	sb.WriteString("---\ntitle: \"Benchmark aggregates\"\ndescription: \"A compact overview of pooled benchmark results with links to per-benchmark distributions.\"\nicon: \"chart-bar\"\ndraft: false\nweight: 21\n---\n\n")
	sb.WriteString("Results pool session observations across all included runs. Each benchmark page contains per-arm Median, IQR (Q3 − Q1), p90, and Max, along with session and token usage and cost availability.\n\n")
	sb.WriteString("[Return to best-case outcomes](/docs/benchmarks/).\n\n")
	if len(groups) == 0 {
		sb.WriteString("> [!NOTE]\n> No publishable benchmark observations are available to aggregate.\n")
		return sb.String()
	}
	totals := benchmarkAggregateUsageTotals(groups)
	fmt.Fprintf(&sb, "**Overall:** %d sessions · %d input tokens · %d output tokens · %d reasoning tokens · %d total tokens · %s known cost (credits) across %d priced arm groups · %d unpriced arm groups.\n\n",
		totals.sessions, totals.inputTokens, totals.outputTokens, totals.reasoningTokens, totals.inputTokens+totals.outputTokens+totals.reasoningTokens,
		formatBenchmarkMetric("cost", totals.cost), totals.pricedGroups, totals.unpricedGroups)
	sb.WriteString("| Benchmark | Sessions | Arm/context groups | Input tokens | Output tokens | Total tokens | Known cost (credits) | Unpriced groups |\n| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	byTask := make(map[string][]benchmarkAggregateGroup)
	tasks := make([]string, 0)
	for _, group := range groups {
		if _, exists := byTask[group.task]; !exists {
			tasks = append(tasks, group.task)
		}
		byTask[group.task] = append(byTask[group.task], group)
	}
	sort.Strings(tasks)
	for _, task := range tasks {
		taskGroups := byTask[task]
		taskTotals := benchmarkAggregateUsageTotals(taskGroups)
		id := benchmarkAggregatePageID(task)
		fmt.Fprintf(&sb, "| [%s](/docs/benchmarks/aggregates/tasks/%s/) | %d | %d | %d | %d | %d | %s | %d |\n",
			task, id, taskTotals.sessions, len(taskGroups), taskTotals.inputTokens, taskTotals.outputTokens,
			taskTotals.inputTokens+taskTotals.outputTokens+taskTotals.reasoningTokens, formatBenchmarkMetric("cost", taskTotals.cost), taskTotals.unpricedGroups)
	}
	return sb.String()
}

func benchmarkAggregatePageID(task string) string {
	sum := sha256.Sum256([]byte(task))
	var slug strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(task) {
		if slug.Len() >= 48 {
			break
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			slug.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen && slug.Len() > 0 {
			slug.WriteByte('-')
			lastHyphen = true
		}
	}
	name := strings.Trim(slug.String(), "-")
	if name == "" {
		name = "benchmark"
	}
	return fmt.Sprintf("%s-%x", name, sum[:8])
}

func renderBenchmarkAggregateDetails(comparisons []*BenchComparisonSummary) []benchmarkAggregatePage {
	groups := aggregateBenchmarkMetrics(comparisons)
	byTask := make(map[string][]benchmarkAggregateGroup)
	tasks := make([]string, 0)
	for _, group := range groups {
		if _, exists := byTask[group.task]; !exists {
			tasks = append(tasks, group.task)
		}
		byTask[group.task] = append(byTask[group.task], group)
	}
	sort.Strings(tasks)
	pages := make([]benchmarkAggregatePage, 0, len(tasks))
	for _, task := range tasks {
		pages = append(pages, benchmarkAggregatePage{
			ID:      benchmarkAggregatePageID(task),
			Task:    task,
			Content: renderBenchmarkAggregateDetailDoc(task, byTask[task]),
		})
	}
	return pages
}

func renderBenchmarkAggregateDetailDoc(task string, groups []benchmarkAggregateGroup) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "---\ntitle: \"%s benchmark results\"\ndescription: \"Pooled session-level metric distributions and usage for %s.\"\nicon: \"chart-bar\"\ndraft: false\n---\n\n", task, task)
	sb.WriteString("[Back to all benchmark aggregates](/docs/benchmarks/aggregates/).\n\n")
	if len(groups) == 0 {
		sb.WriteString("> [!NOTE]\n> No publishable benchmark observations are available to aggregate.\n")
		return sb.String()
	}
	fmt.Fprintf(&sb, "Results pool underlying session observations across all included runs. Arm identity is based on experimental configuration; run provenance and observed metrics do not define an arm. Legacy `semedit` observations are labeled `%s` (default); `%s` appears only when a run records that arm. Numeric summaries use linearly interpolated quantiles over sorted observations: Median (p50), IQR (Q3 − Q1), p90, and Max. Model-specific cost uses declared per-million-token rates. Unknown cost formulas are marked unavailable and excluded from known-cost totals.\n\n", benchmarkAggregateArmSemeditImplicit, benchmarkAggregateArmSemeditExplicit)
	totals := benchmarkAggregateUsageTotals(groups)
	fmt.Fprintf(&sb, "**Benchmark total:** %d sessions · %d input tokens · %d output tokens · %d reasoning tokens · %d total tokens · %s known cost (credits) across %d priced arm groups · %d unpriced arm groups.\n\n",
		totals.sessions, totals.inputTokens, totals.outputTokens, totals.reasoningTokens, totals.inputTokens+totals.outputTokens+totals.reasoningTokens,
		formatBenchmarkMetric("cost", totals.cost), totals.pricedGroups, totals.unpricedGroups)
	for _, group := range groups {
		fmt.Fprintf(&sb, "## %s · %s · %s · Arm: %s\n\n", group.target, group.context, group.prompt, benchmarkAggregateArmLabel(group.arm))
		fmt.Fprintf(&sb, "MCP server instructions: %s · Semedit restriction: %s\n\n", group.instructions, group.restriction)
		sb.WriteString("| Measure | N | Median / pass rate | IQR | p90 | Max | Total usage / cost |\n| :--- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		fmt.Fprintf(&sb, "| Sessions | %d | — | — | — | — | %d |\n", group.sessionCount, group.sessionCount)
		fmt.Fprintf(&sb, "| Input tokens | %d | — | — | — | — | %d |\n", group.sessionCount, group.totalInputTokens)
		fmt.Fprintf(&sb, "| Output tokens | %d | — | — | — | — | %d |\n", group.sessionCount, group.totalOutputTokens)
		fmt.Fprintf(&sb, "| Reasoning tokens | %d | — | — | — | — | %d |\n", group.sessionCount, group.totalReasoningTokens)
		fmt.Fprintf(&sb, "| Total tokens | %d | — | — | — | — | %d |\n", group.sessionCount, group.totalInputTokens+group.totalOutputTokens+group.totalReasoningTokens)
		if group.costKnown {
			fmt.Fprintf(&sb, "| Total model cost (credits) | %d | — | — | — | — | %s |\n", group.sessionCount, formatBenchmarkMetric("cost", group.totalCost))
		} else {
			fmt.Fprintf(&sb, "| Total model cost (credits) | %d | — | — | — | — | Unavailable: no declared model cost formula |\n", group.sessionCount)
		}
		for _, metric := range benchmarkMetrics() {
			summary := group.metrics[metric.name]
			if summary == nil || metric.unit == "binary" {
				continue
			}
			fmt.Fprintf(&sb, "| %s | %d | %s | %s | %s | %s | — |\n", metric.name, summary.count,
				formatBenchmarkMetric(metric.unit, summary.median()),
				formatBenchmarkMetric(metric.unit, summary.quantile(0.75)-summary.quantile(0.25)),
				formatBenchmarkMetric(metric.unit, summary.quantile(0.9)),
				formatBenchmarkMetric(metric.unit, summary.quantile(1)))
		}
		for _, metric := range benchmarkMetrics() {
			summary := group.metrics[metric.name]
			if summary == nil || metric.unit != "binary" || summary.count == 0 {
				continue
			}
			passed := 0
			for _, value := range summary.values {
				if value >= 0.5 {
					passed++
				}
			}
			fmt.Fprintf(&sb, "| %s | %d | %d/%d (%.1f%%) | — | — | — | — |\n", metric.name, summary.count, passed, summary.count, float64(passed)/float64(summary.count)*100)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

type benchmarkAggregatePage struct {
	ID      string
	Task    string
	Content string
}

type benchmarkAggregateUsage struct {
	sessions        int
	inputTokens     int
	outputTokens    int
	reasoningTokens int
	cost            float64
	pricedGroups    int
	unpricedGroups  int
}

func benchmarkAggregateUsageTotals(groups []benchmarkAggregateGroup) benchmarkAggregateUsage {
	var totals benchmarkAggregateUsage
	for _, group := range groups {
		totals.sessions += group.sessionCount
		totals.inputTokens += group.totalInputTokens
		totals.outputTokens += group.totalOutputTokens
		totals.reasoningTokens += group.totalReasoningTokens
		if group.costKnown {
			totals.cost += group.totalCost
			totals.pricedGroups++
		} else {
			totals.unpricedGroups++
		}
	}
	return totals
}

func aggregateBenchmarkMetrics(comparisons []*BenchComparisonSummary) []benchmarkAggregateGroup {
	groups := make(map[benchmarkArmKey]*benchmarkAggregateGroup)
	for _, comparison := range comparisons {
		for _, context := range []benchmarkPairContext{benchmarkPairSmall, benchmarkPairLarge, benchmarkPairVerifiedSmall, benchmarkPairVerifiedLarge, benchmarkPairExplicitSmall, benchmarkPairExplicitLarge, benchmarkPairExplicitVerifiedSmall, benchmarkPairExplicitVerifiedLarge} {
			addBenchmarkComparisonMetrics(groups, comparison, context)
		}
	}

	result := make([]benchmarkAggregateGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		return strings.Join([]string{left.task, left.target, left.prompt, left.instructions, left.restriction, left.context, left.arm}, "\x00") <
			strings.Join([]string{right.task, right.target, right.prompt, right.instructions, right.restriction, right.context, right.arm}, "\x00")
	})
	return result
}

func addBenchmarkComparisonMetrics(groups map[benchmarkArmKey]*benchmarkAggregateGroup, comparison *BenchComparisonSummary, context benchmarkPairContext) {
	baseline, semedit := benchmarkPairRuns(comparison, context)
	if isExplicitBenchmarkPairContext(context) {
		addBenchmarkMetricRun(groups, comparison, context, benchmarkAggregateArmName(benchmarkAggregateArmSemeditExplicit, semedit), semedit)
		return
	}
	addBenchmarkMetricRun(groups, comparison, context, benchmarkAggregateArmName(benchmarkAggregateArmBaseline, baseline), baseline)
	addBenchmarkMetricRun(groups, comparison, context, benchmarkAggregateArmName(benchmarkAggregateArmSemeditImplicit, semedit), semedit)
}

func isExplicitBenchmarkPairContext(context benchmarkPairContext) bool {
	switch context {
	case benchmarkPairExplicitSmall, benchmarkPairExplicitLarge, benchmarkPairExplicitVerifiedSmall, benchmarkPairExplicitVerifiedLarge:
		return true
	default:
		return false
	}
}

func benchmarkAggregateArmName(defaultName string, run *BenchRunResult) string {
	if run == nil {
		return defaultName
	}
	switch strings.ToLower(strings.TrimSpace(run.Arm)) {
	case benchmarkAggregateArmSemeditExplicit:
		return benchmarkAggregateArmSemeditExplicit
	case benchmarkAggregateArmSemeditImplicit:
		return benchmarkAggregateArmSemeditImplicit
	}
	return defaultName
}

func benchmarkAggregateArmLabel(arm string) string {
	switch arm {
	case benchmarkAggregateArmBaseline:
		return benchmarkAggregateArmBaseline + " (vanilla)"
	case benchmarkAggregateArmSemeditImplicit:
		return benchmarkAggregateArmSemeditImplicit + " (default)"
	case benchmarkAggregateArmSemeditExplicit:
		return benchmarkAggregateArmSemeditExplicit
	default:
		return arm
	}
}

func addBenchmarkMetricRun(groups map[benchmarkArmKey]*benchmarkAggregateGroup, comparison *BenchComparisonSummary, context benchmarkPairContext, arm string, run *BenchRunResult) {
	if run == nil {
		return
	}
	config := resolveBenchmarkAggregateRunConfig(comparison, context, arm, run)
	costRun := *run
	costRun.Target = config.target
	group := groups[config.key]
	if group == nil {
		group = &benchmarkAggregateGroup{
			task: comparison.TaskID, target: config.target.String(), prompt: config.prompt,
			instructions: config.instructions, context: context.label(), arm: arm,
			metrics: make(map[string]*benchmarkMetricSummary), restriction: config.restriction,
			costKnown: true,
		}
		groups[config.key] = group
	}
	group.sessionCount++
	group.totalInputTokens += run.PromptTokens
	group.totalOutputTokens += run.OutputTokens
	group.totalReasoningTokens += run.ReasoningTokens
	if cost, available := benchmarkCost(&costRun); available {
		group.totalCost += cost
	} else {
		group.costKnown = false
	}
	for _, metric := range benchmarkMetrics() {
		value, present := metric.value(&costRun)
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

func benchmarkArmPromptConfiguration(comparison *BenchComparisonSummary, context benchmarkPairContext, arm string) string {
	verified := context == benchmarkPairVerifiedSmall || context == benchmarkPairVerifiedLarge
	switch arm {
	case benchmarkAggregateArmBaseline:
		if verified {
			return comparison.VanillaVerifiedPrompt
		}
		return comparison.VanillaPrompt
	case benchmarkAggregateArmSemeditImplicit, benchmarkAggregateArmSemeditExplicit:
		if verified {
			return comparison.MCPVerifiedPrompt
		}
		return comparison.MCPPrompt
	default:
		return ""
	}
}

type benchmarkAggregateRunConfig struct {
	key          benchmarkArmKey
	target       BenchTarget
	prompt       string
	instructions string
	restriction  string
}

func resolveBenchmarkAggregateRunConfig(comparison *BenchComparisonSummary, context benchmarkPairContext, arm string, run *BenchRunResult) benchmarkAggregateRunConfig {
	prompt := strings.TrimSpace(comparison.PromptVariant)
	if prompt == "" {
		prompt = "default"
	}
	promptVariant := strings.TrimSpace(run.PromptVariant)
	if promptVariant == "" {
		promptVariant = prompt
	}
	instructions := run.MCPServerInstructions
	legacyInstructions := run.LegacyMCPInstructions
	if instructions == "" && legacyInstructions == "" {
		instructions = comparison.MCPServerInstructions
		legacyInstructions = comparison.LegacyMCPInstructions
	}
	restriction := run.SemeditArmRestrict
	if restriction == "" {
		restriction = comparison.SemeditArmRestrict
	}
	target := run.Target
	if target.Harness == "" {
		target.Harness = comparison.Target.Harness
	}
	if target.Model == "" {
		target.Model = comparison.Target.Model
	}
	if target.Effort == "" {
		target.Effort = comparison.Target.Effort
	}
	promptConfiguration := benchmarkArmPromptConfiguration(comparison, context, arm)
	if promptConfiguration == "" {
		promptConfiguration = run.Prompt
	}
	effectivePrompt := run.Prompt
	if effectivePrompt == "" {
		effectivePrompt = promptConfiguration
	}
	return benchmarkAggregateRunConfig{
		key: benchmarkArmKey{
			task: comparison.TaskID, target: target,
			restriction: displaySemeditArmRestriction(restriction), promptVariant: promptVariant,
			promptConfiguration: promptConfiguration, context: context, arm: arm, variant: run.Variant,
			runInstructions: displayMCPServerInstructions(instructions, legacyInstructions),
			interactiveMode: run.InteractiveMode, prompt: effectivePrompt,
		},
		target: target, prompt: promptVariant,
		instructions: displayMCPServerInstructions(instructions, legacyInstructions),
		restriction:  displaySemeditArmRestriction(restriction),
	}
}
