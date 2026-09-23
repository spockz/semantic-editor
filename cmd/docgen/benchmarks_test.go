// Tests ensure generated benchmark documentation excludes harness failures while retaining evaluated oracle failures.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUnmarshalBenchmarkReportNormalizesVersionedMillisecondsAndLegacyNanoseconds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data string
		want time.Duration
	}{
		{
			name: "versioned milliseconds",
			data: `{"format_version":2,"duration_unit":"milliseconds","runs":[{"wall_clock_ms":1500,"process_start_to_first_event_ms":250,"oracle":{"level_1_policy":false,"duration_ms":1750}}]}`,
			want: 1500 * time.Millisecond,
		},
		{
			name: "legacy nanoseconds",
			data: `{"runs":[{"wall_clock_ms":1500000000,"process_start_to_first_event_ms":250000000,"oracle":{"level_1_policy":false,"duration_ms":1750000000}}]}`,
			want: 1500 * time.Millisecond,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var report BenchReport
			if err := unmarshalBenchmarkReport([]byte(tc.data), &report); err != nil {
				t.Fatalf("unmarshal benchmark report: %v", err)
			}
			if len(report.Runs) != 1 {
				t.Fatalf("runs = %d, want 1", len(report.Runs))
			}
			run := report.Runs[0]
			if got := run.WallClock; got != tc.want {
				t.Errorf("wall clock = %v, want %v", got, tc.want)
			}
			if got, want := *run.ProcessStartToFirstEvent, 250*time.Millisecond; got != want {
				t.Errorf("process start to first event = %v, want %v", got, want)
			}
			if got, want := run.Oracle.Duration, 1750*time.Millisecond; got != want {
				t.Errorf("oracle duration = %v, want %v", got, want)
			}
		})
	}
}

func TestLoadAllBenchmarkComparisonsExcludesIncompleteRuns(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	resultsDir := filepath.Join(rootDir, "data", "benchmarks", "results", "task")
	if err := os.MkdirAll(resultsDir, 0o750); err != nil {
		t.Fatalf("create results directory: %v", err)
	}

	completed := &BenchRunResult{
		TaskID:  "task-mixed",
		Target:  BenchTarget{Harness: "codex", Model: "gpt-5.6-luna"},
		Arm:     "baseline-diff",
		Success: true,
		Oracle:  &BenchOracleResult{Passed: true},
	}
	oracleFailure := &BenchRunResult{
		TaskID:  "task-mixed",
		Target:  BenchTarget{Harness: "codex", Model: "gpt-5.6-luna"},
		Arm:     "baseline-diff",
		Success: false,
		Oracle:  &BenchOracleResult{Passed: false, FailureStage: "ast"},
	}
	executionFailure := &BenchRunResult{
		TaskID: "task-mixed",
		Target: BenchTarget{Harness: "agy", Model: "gemini"},
		Arm:    "semedit",
		Error:  "harness quota exhausted",
	}
	report := BenchReport{Comparisons: []*BenchComparisonSummary{
		{
			TaskID:        "task-mixed",
			Target:        BenchTarget{Harness: "codex", Model: "gpt-5.6-luna"},
			SmallBaseline: completed,
			SmallSemedit:  executionFailure,
			LargeBaseline: oracleFailure,
		},
		{
			TaskID:        "task-execution-failure",
			Target:        BenchTarget{Harness: "agy", Model: "gemini"},
			SmallBaseline: executionFailure,
		},
	}}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal benchmark report: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resultsDir, "report.json"), data, 0o600); err != nil {
		t.Fatalf("write benchmark report: %v", err)
	}

	comparisons, err := loadAllBenchmarkComparisons(rootDir)
	if err != nil {
		t.Fatalf("load benchmark comparisons: %v", err)
	}
	if len(comparisons) != 1 {
		t.Fatalf("expected one publishable comparison, got %d", len(comparisons))
	}

	comparison := comparisons[0]
	if comparison.SmallBaseline == nil || !comparison.SmallBaseline.Success {
		t.Fatal("expected completed run to remain publishable")
	}
	if comparison.SmallSemedit != nil {
		t.Fatal("expected execution failure to be excluded")
	}
	if comparison.LargeBaseline == nil || comparison.LargeBaseline.Success {
		t.Fatal("expected evaluated oracle failure to remain publishable")
	}
}

func TestRenderBenchmarkDocumentationSelectsBestPairsAndPreservesRunPages(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	writeBenchmarkDocumentationReport(t, rootDir, "run-a", &BenchComparisonSummary{
		TaskID: "task-speed", Target: BenchTarget{Harness: "codex", Model: "gpt-5.6-luna", Effort: "medium"},
		SmallBaseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
		SmallSemedit:  benchmarkDocumentationRunResult(true, 9*time.Second+800*time.Millisecond, 1000),
	})
	writeBenchmarkDocumentationReport(t, rootDir, "run-b", &BenchComparisonSummary{
		TaskID: "task-speed", Target: BenchTarget{Harness: "codex", Model: "gpt-5.6-luna", Effort: "medium"},
		SmallBaseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
		SmallSemedit:  benchmarkDocumentationRunResult(true, 9*time.Second+500*time.Millisecond, 300),
	})

	documentation, err := renderBenchmarkDocumentation(rootDir)
	if err != nil {
		t.Fatalf("render benchmark documentation: %v", err)
	}
	if len(documentation.Runs) != 2 {
		t.Fatalf("got %d run pages, want 2", len(documentation.Runs))
	}
	for _, want := range []string{
		"## Best measured improvements",
		"| Best speed increase | 5.0% faster |",
		"| Best token reduction | 70.0% fewer cache-adjusted token units |",
		"| MCP first-time-right edits | MCP 2/2 (100.0%) vs Vanilla 2/2 (100.0%), +0.0 pp |",
		"## Required corrective turns",
		"| 0 | 2 | 2 |",
		"best-case evidence, not an average",
		"MCP (Small) | Δ (Small)",
		"9.50s",
		"/docs/benchmarks/aggregates/",
		"/docs/benchmarks/runs/run-a/",
		"/docs/benchmarks/runs/run-b/",
	} {
		if !strings.Contains(documentation.Index, want) {
			t.Errorf("best-case index missing %q", want)
		}
	}
	if strings.Index(documentation.Index, "## Best measured improvements") > strings.Index(documentation.Index, "## Benchmark Methodology & Transparency") {
		t.Fatal("headline table must appear before the benchmark methodology")
	}
	if strings.Contains(documentation.Index, "9.80s") {
		t.Fatal("best-case index must use the lower-cost run when speeds are comparable")
	}
	if !strings.Contains(documentation.Aggregates, "| Wall-clock latency | 2 | 9.50s | 9.80s | 9.65s |") {
		t.Errorf("aggregate page does not contain run range: %s", documentation.Aggregates)
	}
	if !strings.Contains(documentation.Aggregates, "| Cost | 2 | 0.0015 | 0.0050 | 0.0033 |") {
		t.Errorf("aggregate page does not contain model cost range: %s", documentation.Aggregates)
	}
	if got := renderBenchmarkRunDoc(documentation.Runs[0]); !strings.Contains(got, "9.80s") {
		t.Errorf("run-a page must retain its unaggregated observation: %s", got)
	}
}

func TestBestBenchmarkPairPrefersMCPSuccessOverSpeed(t *testing.T) {
	t.Parallel()

	passingMCP := bestBenchmarkPair{
		runID: "slow-success", baseline: benchmarkDocumentationRunResult(false, time.Second, 100), semedit: benchmarkDocumentationRunResult(true, 20*time.Second, 100),
	}
	fastFailure := bestBenchmarkPair{
		runID: "fast-failure", baseline: benchmarkDocumentationRunResult(true, time.Second, 100), semedit: benchmarkDocumentationRunResult(false, time.Millisecond, 100),
	}
	if !passingMCP.preferredTo(fastFailure) {
		t.Fatal("MCP oracle success must outrank a faster MCP oracle failure")
	}
}

func TestBestBenchmarkPreambleOffsetsFirstTimeRightAgainstVanilla(t *testing.T) {
	t.Parallel()

	firstTurnBaseline := benchmarkDocumentationRunResult(true, 10*time.Second, 1000)
	firstTurnMCP := benchmarkDocumentationRunResult(true, 9*time.Second, 900)
	laterBaseline := benchmarkDocumentationRunResult(true, 10*time.Second, 1000)
	laterBaseline.Turns = 2
	laterBaseline.InteractionSteps = []BenchInteractionStep{{Step: 1, Oracle: &BenchOracleResult{Passed: false}}}
	laterMCP := benchmarkDocumentationRunResult(true, 9*time.Second, 900)
	laterMCP.InteractionSteps = []BenchInteractionStep{{Step: 1, Oracle: &BenchOracleResult{Passed: true}}}
	pairs := []bestBenchmarkPair{
		{runID: "run-a", comparison: &BenchComparisonSummary{TaskID: "task-a"}, context: benchmarkPairSmall, baseline: firstTurnBaseline, semedit: firstTurnMCP},
		{runID: "run-b", comparison: &BenchComparisonSummary{TaskID: "task-b"}, context: benchmarkPairSmall, baseline: laterBaseline, semedit: laterMCP},
	}
	runs := []benchmarkDocumentationRun{{ID: "run-a", Comparisons: []*BenchComparisonSummary{
		{
			SmallBaseline:         firstTurnBaseline,
			SmallSemedit:          firstTurnMCP,
			SmallVerifiedBaseline: benchmarkDocumentationRunResult(false, time.Second, 100),
			SmallVerifiedSemedit:  benchmarkDocumentationRunResult(true, time.Second, 100),
		},
		{SmallBaseline: laterBaseline, SmallSemedit: laterMCP},
	}}}

	preamble := renderBestBenchmarkPreamble(pairs, runs)
	if !strings.Contains(preamble, "MCP 2/2 (100.0%) vs Vanilla 1/2 (50.0%), +50.0 pp") {
		t.Errorf("first-time-right headline = %q", preamble)
	}
}

func TestCorrectiveTurnHistogramSeparatesArmsAndExcludesVerifiedContexts(t *testing.T) {
	t.Parallel()

	baseline := benchmarkDocumentationRunResult(true, 10*time.Second, 1000)
	baseline.InteractionSteps = []BenchInteractionStep{{Step: 1}, {Step: 2}, {Step: 3}}
	semedit := benchmarkDocumentationRunResult(true, 9*time.Second, 900)
	semedit.InteractionSteps = []BenchInteractionStep{{Step: 1}}
	runs := []benchmarkDocumentationRun{{Comparisons: []*BenchComparisonSummary{{
		SmallBaseline:         baseline,
		SmallSemedit:          semedit,
		SmallVerifiedBaseline: benchmarkDocumentationRunResult(true, time.Second, 100),
		SmallVerifiedSemedit:  benchmarkDocumentationRunResult(true, time.Second, 100),
	}}}}

	histogram := renderCorrectiveTurnHistogram(runs)
	for _, want := range []string{
		"| 0 | 0 | 1 |",
		"| 2 | 1 | 0 |",
		"| 5 | 0 | 0 |",
	} {
		if !strings.Contains(histogram, want) {
			t.Errorf("histogram missing %q: %s", want, histogram)
		}
	}
}

func TestBestBenchmarkPairPrefersOneShotWhenCostsAreComparable(t *testing.T) {
	t.Parallel()

	oneShot := bestBenchmarkPair{
		runID: "one-shot", baseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000), semedit: benchmarkDocumentationRunResult(true, 10*time.Second, 1020),
	}
	oneShot.semedit.Turns = 1
	multipleTurns := bestBenchmarkPair{
		runID: "multiple-turns", baseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000), semedit: benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
	}
	multipleTurns.semedit.Turns = 2
	if !oneShot.preferredTo(multipleTurns) {
		t.Fatal("one-shot MCP completion must win when model costs are within five percent")
	}
}

func TestBestBenchmarkPairAllowsTenfoldCostGainToOverrideSpeed(t *testing.T) {
	t.Parallel()

	slowerCheaper := bestBenchmarkPair{
		runID: "slower-cheaper", baseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000), semedit: benchmarkDocumentationRunResult(true, 20*time.Second, 100),
	}
	fasterExpensive := bestBenchmarkPair{
		runID: "faster-expensive", baseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000), semedit: benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
	}
	if !slowerCheaper.preferredTo(fasterExpensive) {
		t.Fatal("a tenfold model-cost reduction must override a non-comparable speed regression")
	}
}

func benchmarkDocumentationRunResult(passed bool, wallClock time.Duration, uncached int) *BenchRunResult {
	return &BenchRunResult{
		Target: BenchTarget{Harness: "codex", Model: "gpt-5.6-luna", Effort: "medium"}, Arm: "baseline-diff", Success: passed,
		Turns: 1, WallClock: wallClock, PromptTokens: uncached, UncachedPromptTokens: uncached, CachedPromptTokens: 0,
		Oracle: &BenchOracleResult{Passed: passed, Level1Policy: passed, Level2AST: passed, Level3Build: passed, Level4Test: passed},
	}
}

func writeBenchmarkDocumentationReport(t *testing.T, rootDir, runID string, comparison *BenchComparisonSummary) {
	t.Helper()
	directory := filepath.Join(rootDir, "data", "benchmarks", "results", runID)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatalf("create run result directory: %v", err)
	}
	data, err := json.Marshal(BenchReport{Comparisons: []*BenchComparisonSummary{comparison}})
	if err != nil {
		t.Fatalf("marshal run report: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "report.json"), data, 0o600); err != nil {
		t.Fatalf("write run report: %v", err)
	}
}

func TestRenderBenchmarksDocOmitsTheThemeProvidedPageTitle(t *testing.T) {
	t.Parallel()

	page, err := renderBenchmarksDoc(t.TempDir())
	if err != nil {
		t.Fatalf("render benchmarks document: %v", err)
	}
	if strings.Contains(page, "# Empirical Benchmarks\n") {
		t.Fatal("benchmarks document must not duplicate Hugo's page title")
	}
}

func TestRenderDocRunSummaryQualifiesToolUsage(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, 5, "Standard vs Semedit in Small Context",
		&BenchRunResult{Diff: "File main.go modified", ToolCalls: []BenchToolCall{{Name: "view_file", TransportStatus: "succeeded", FunctionalStatus: "succeeded"}, {Name: "replace_file_content", TransportStatus: "succeeded", FunctionalStatus: "succeeded"}}},
		&BenchRunResult{Diff: "File main.go modified", ToolCalls: []BenchToolCall{{Name: "semantic_replace_body", Server: "semedit", TransportStatus: "succeeded", FunctionalStatus: "failed", Failure: "symbol not found", MCPMetrics: &BenchMCPMetrics{TotalMS: 1250, Phases: map[string]BenchMCPPhaseMetric{"verification.after_diagnostics": {DurationMS: 700}, "formatting.organize_imports": {DurationMS: 300}}}}}},
	)

	for _, want := range []string{
		"##### Standard vs Semedit in Small Context",
		"| # | Vanilla | Semedit MCP |",
		"| 1 | `view_file` (<span role=\"img\" aria-label=\"Transport succeeded\" title=\"Transport succeeded\">✓</span> <span role=\"img\" aria-label=\"Functional succeeded\" title=\"Functional succeeded\">✓</span>) | `semedit/semantic_replace_body` (<span role=\"img\" aria-label=\"Transport succeeded\" title=\"Transport succeeded\">✓</span> <span role=\"img\" aria-label=\"Functional failed\" title=\"Functional failed\">✗</span>): symbol not found; server: 1.25s (verification: 700ms; formatting: 300ms) |",
		"| 2 | `replace_file_content` (<span role=\"img\" aria-label=\"Transport succeeded\" title=\"Transport succeeded\">✓</span> <span role=\"img\" aria-label=\"Functional succeeded\" title=\"Functional succeeded\">✓</span>) | — |",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered run summary missing %q", want)
		}
	}
}

func TestRenderDocRunSummaryRendersShellCommandsAsCopyableBlocks(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, 5, "Standard vs Semedit in Small Context",
		&BenchRunResult{ToolCalls: []BenchToolCall{{Name: "/bin/zsh -lc 'make check && go test ./... || true'", TransportStatus: "timeout", FunctionalStatus: "failed"}}},
		nil,
	)

	for _, want := range []string{
		"<table class=\"benchmark-tool-calls\">\n<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>",
		"<tr><td>1</td><td><span role=\"img\" aria-label=\"Transport timeout\" title=\"Transport timeout\">⏱</span> <span role=\"img\" aria-label=\"Functional failed\" title=\"Functional failed\">✗</span>{{< benchmark-code lang=\"shell\" content=\"L2Jpbi96c2ggLWxjICdtYWtlIGNoZWNrICYmIFwKZ28gdGVzdCAuLy4uLiB8fCBcCnRydWUn\" >}}</td><td>not published</td></tr>",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered shell command missing %q", want)
		}
	}
	if strings.Contains(rendered.String(), "```shell") {
		t.Error("shell command must remain inside its table cell")
	}
	if strings.Contains(rendered.String(), "benchmark-shell-command") {
		t.Error("shell command must not use the legacy manual pre class")
	}
}

func TestRenderDocRunSummaryRendersToolArguments(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, 5, "Standard vs Semedit in Small Context", nil,
		&BenchRunResult{ToolCalls: []BenchToolCall{{
			Name:             "semantic_rename",
			Server:           "semedit",
			Arguments:        json.RawMessage(`{"symbol":"Old","to":"New"}`),
			TransportStatus:  "succeeded",
			FunctionalStatus: "succeeded",
		}}},
	)

	for _, want := range []string{
		"<code>semedit/semantic_rename</code>",
		"{{< benchmark-code lang=\"json\" content=\"ewogICJzeW1ib2wiOiAiT2xkIiwKICAidG8iOiAiTmV3Igp9\" >}}",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered tool arguments missing %q", want)
		}
	}
	if strings.Contains(rendered.String(), "benchmark-tool-arguments") {
		t.Error("tool arguments must not use the legacy manual pre class")
	}
}

func TestRenderDocSemanticToolDiagnosticPrintsReflectionReason(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocSemanticToolDiagnostic(&rendered, &BenchRunResult{SemanticToolReflection: &BenchSemanticToolReflection{
		Response: "I did not discover any callable semantic tools.",
	}})

	const want = "<div class=\"callout callout-warning\"><div class=\"callout-title\"><span>⚠</span> Why no semantic edit tool was used</div><div class=\"callout-desc\">I did not discover any callable semantic tools.</div></div>"
	if !strings.Contains(rendered.String(), want) {
		t.Errorf("rendered diagnostic missing %q", want)
	}
}

func TestRenderDocSemanticToolDiagnosticReportsUnavailableToolsAsError(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocSemanticToolDiagnostic(&rendered, &BenchRunResult{SemanticToolReflection: &BenchSemanticToolReflection{
		Response: "I did not call a semantic_* tool because no Semedit MCP tools were available in the exposed tool set.",
	}})

	if !strings.Contains(rendered.String(), "callout callout-error") {
		t.Errorf("unavailable semantic tools must render an error callout: %s", rendered.String())
	}
}

func TestRenderDocRunSummaryUsesTheOrderedReasoningChain(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, 5, "Standard vs Semedit in Small Context", nil, &BenchRunResult{
		ToolCalls: []BenchToolCall{{Name: "/bin/zsh -lc 'obsolete flattened call'"}},
		InteractionSteps: []BenchInteractionStep{
			{
				Step: 2,
				ToolCalls: []BenchToolCall{
					{Name: "/bin/zsh -lc 'make check'", TransportStatus: "succeeded", FunctionalStatus: "succeeded"},
					{Name: "semantic_verify", Server: "semedit", TransportStatus: "succeeded", FunctionalStatus: "succeeded"},
				},
			},
			{
				Step:      1,
				ToolCalls: []BenchToolCall{{Name: "semantic_rename", Server: "semedit", TransportStatus: "succeeded", FunctionalStatus: "succeeded"}},
			},
		},
	})

	page := rendered.String()
	if strings.Contains(page, "obsolete flattened call") {
		t.Fatal("rendered tool chain must not use the flattened aggregate when turn telemetry is available")
	}

	previous := -1
	for _, want := range []string{"semedit/semantic_rename", "L2Jpbi96c2ggLWxjICdtYWtlIGNoZWNrJw==", "semedit/semantic_verify"} {
		position := strings.Index(page, want)
		if position < 0 {
			t.Errorf("rendered tool chain missing %q", want)
			continue
		}
		if position <= previous {
			t.Errorf("tool call %q is out of chronological order", want)
		}
		previous = position
	}
}

func TestWriteDocPromptQuotesEveryLine(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	writeDocPrompt(&rendered, "Vanilla LLM Prompt", "First paragraph.\n\nSecond paragraph.")

	const want = "**Vanilla LLM Prompt**:\n> First paragraph.\n>\n> Second paragraph.\n\n"
	if got := rendered.String(); got != want {
		t.Errorf("quoted prompt = %q, want %q", got, want)
	}
}

func TestBenchmarkToolCallCSSWrapsShellCommandsWithinTableCells(t *testing.T) {
	t.Parallel()

	for _, want := range []string{"table-layout: fixed", "benchmark-code-block", "max-width: 32rem", "white-space: pre-wrap", "overflow-wrap: anywhere", "word-break: break-word", ".benchmark-delta-positive", ".benchmark-delta-negative"} {
		if !strings.Contains(benchmarkToolCallCSS, want) {
			t.Errorf("benchmark tool-call CSS missing %q", want)
		}
	}
}

func TestFormatDocDeltaColorsVerifiedMCPBenefits(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		pct       float64
		diff      float64
		direction docDeltaDirection
		verified  bool
		want      string
	}{
		{name: "verified lower metric", pct: -25, diff: -25, direction: docDeltaLowerIsBetter, verified: true, want: "benchmark-delta-positive"},
		{name: "unverified lower metric", pct: -25, diff: -25, direction: docDeltaLowerIsBetter, want: "N/A"},
		{name: "unverified baseline advantage", pct: 25, diff: 25, direction: docDeltaLowerIsBetter, want: "N/A"},
		{name: "baseline lower metric", pct: 25, diff: 25, direction: docDeltaLowerIsBetter, verified: true, want: "benchmark-delta-negative"},
		{name: "verified cached-token gain", pct: 25, diff: 25, direction: docDeltaHigherIsBetter, verified: true, want: "benchmark-delta-positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := formatDocDelta(test.pct, test.diff, test.direction, test.verified); !strings.Contains(got, test.want) {
				t.Errorf("delta = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDocMCPColumnHeaderQualifiesOnlyUnverifiedRuns(t *testing.T) {
	t.Parallel()

	if got, want := docMCPColumnHeader("Small", &BenchRunResult{}), `<span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small)`; got != want {
		t.Errorf("unverified MCP header = %q, want %q", got, want)
	}
	if got, want := docMCPColumnHeader("Large", &BenchRunResult{MCPVerified: true}), "MCP (Large)"; got != want {
		t.Errorf("verified MCP header = %q, want %q", got, want)
	}
	if got, want := docMCPColumnHeader("Large", nil), "MCP (Large)"; got != want {
		t.Errorf("missing MCP header = %q, want %q", got, want)
	}
}

func TestDocCachedToUncachedTokenRatioFavorsHigherCachedShare(t *testing.T) {
	t.Parallel()

	base := &BenchRunResult{CachedPromptTokens: 136192, UncachedPromptTokens: 31107}
	mcp := &BenchRunResult{CachedPromptTokens: 161536, UncachedPromptTokens: 27570, MCPVerified: true}
	row := formatDocCachedToUncachedRatioRow(base, mcp, nil, nil)
	for _, want := range []string{"**Cached vs Uncached Token Ratio**", "4.38:1", "5.86:1", "benchmark-delta-positive"} {
		if !strings.Contains(row, want) {
			t.Errorf("ratio row = %q, want %q", row, want)
		}
	}
}

func TestBenchmarkCostUsesDeclaredModelRatesWithoutUnit(t *testing.T) {
	t.Parallel()

	run := &BenchRunResult{
		Target:               BenchTarget{Model: "gpt-5.6-luna"},
		UncachedPromptTokens: 100,
		CachedPromptTokens:   20,
		OutputTokens:         3,
		ReasoningTokens:      2,
		MCPVerified:          true,
	}
	cost, available := benchmarkCost(run)
	if !available {
		t.Fatal("Luna cost must be available")
	}
	if want := 0.00066; cost != want {
		t.Errorf("Luna cost = %v, want %v", cost, want)
	}

	if _, available := benchmarkCost(&BenchRunResult{Target: BenchTarget{Model: "unpriced-model"}}); available {
		t.Fatal("unknown model cost must remain unavailable")
	}

	row := formatDocCostRow(run, run, nil, nil)
	if !strings.Contains(row, "| **Cost** | 0.0007 | 0.0007 | 0% |") {
		t.Errorf("cost row = %q, want unitless cost values", row)
	}
}

func TestBestBenchmarkPairUsesModelCostInsteadOfCacheAdjustedTokenUnits(t *testing.T) {
	t.Parallel()

	lowerModelCost := bestBenchmarkPair{
		runID:    "lower-model-cost",
		baseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
		semedit:  benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
	}
	lowerCacheAdjustedUnits := bestBenchmarkPair{
		runID:    "lower-cache-adjusted-units",
		baseline: benchmarkDocumentationRunResult(true, 10*time.Second, 1000),
		semedit:  benchmarkDocumentationRunResult(true, 10*time.Second, 500),
	}
	lowerCacheAdjustedUnits.semedit.OutputTokens = 400

	if !lowerModelCost.preferredTo(lowerCacheAdjustedUnits) {
		t.Fatal("lower model cost must outrank lower cache-adjusted token units when speed is comparable")
	}
}

func TestBenchmarkCostRatesMatchDeclaredModelSchedule(t *testing.T) {
	t.Parallel()

	tests := map[string]benchmarkCostRates{
		"Luna":                  {input: 5, cached: 0.5, output: 30},
		"Gemini 3.5 Flash-Lite": {input: 7.5, cached: 0.75, output: 62.5},
		"Gemini 3.6 Flash":      {input: 18.75, cached: 1.875, output: 93.75},
		"Gemini 3.7 Flash":      {input: 18.75, cached: 1.875, output: 93.75},
		"Gemini 3.8 Flash":      {input: 18.75, cached: 1.875, output: 93.75},
		"Terra":                 {input: 50, cached: 5, output: 300},
		"Sol":                   {input: 100, cached: 10, output: 500},
	}
	for model, want := range tests {
		got, available := benchmarkCostRatesForModel(model)
		if !available {
			t.Errorf("rates for %q are unavailable", model)
			continue
		}
		if got != want {
			t.Errorf("rates for %q = %#v, want %#v", model, got, want)
		}
	}
}

func TestDisplayMCPServerInstructions(t *testing.T) {
	t.Parallel()

	if got, want := displayMCPServerInstructions("", ""), "none"; got != want {
		t.Errorf("empty instructions = %q, want %q", got, want)
	}
	if got, want := displayMCPServerInstructions("descriptive", ""), "descriptive"; got != want {
		t.Errorf("descriptive instructions = %q, want %q", got, want)
	}
	if got, want := displayMCPServerInstructions("", "directive"), "prescriptive"; got != want {
		t.Errorf("legacy directive instructions = %q, want %q", got, want)
	}
}

func TestDisplayProvenance(t *testing.T) {
	t.Parallel()

	if got, want := displayProvenance(map[string]string{"parser-limit": "16MiB"}, nil), "parser-limit=16MiB"; got != want {
		t.Errorf("provenance display = %q, want %q", got, want)
	}
	if got, want := displayProvenance(nil, map[string]string{"parser-limit": "32MiB"}), "parser-limit=32MiB"; got != want {
		t.Errorf("legacy classifier display = %q, want %q", got, want)
	}
}

func TestWriteDocConfigurationDetailsRestatesComparisonContext(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	writeDocConfigurationDetails(&rendered, &BenchComparisonSummary{
		TaskID:                "task-04-insert-public",
		Target:                BenchTarget{Harness: "codex", Model: "gpt-5.6-luna", Effort: "medium"},
		MCPServerInstructions: "descriptive",
		Provenance:            map[string]string{"run": "42"},
		TxtarPath:             "testdata/bench/task_04_insert_public.txtar",
		TxtarProvenance:       "sha256:abc",
	})

	const want = "| Setting | Value |\n" +
		"| :--- | :--- |\n" +
		"| Test case | `task-04-insert-public` |\n" +
		"| Target | `codex/gpt-5.6-luna/medium` |\n" +
		"| Prompt variant | `default` |\n" +
		"| MCP server instructions | `descriptive` |\n" +
		"| Run provenance | `run=42` |\n" +
		"| Fixture | `testdata/bench/task_04_insert_public.txtar` (`sha256:abc`) |\n\n"
	if got := rendered.String(); got != want {
		t.Errorf("configuration details = %q, want %q", got, want)
	}
}

func TestRenderDocRunSummaryShowsAbsentTelemetry(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, 5, "Standard vs Semedit in Large Context", &BenchRunResult{}, nil)

	for _, want := range []string{
		"##### Standard vs Semedit in Large Context",
		"| 1 | none recorded | not published |",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered run summary missing %q", want)
		}
	}
}

func TestEscapeDocToolCallTableCell(t *testing.T) {
	t.Parallel()

	if got, want := escapeDocTableCell("`rg one |\nrg two`"), "`rg one \\| rg two`"; got != want {
		t.Errorf("escaped table cell = %q, want %q", got, want)
	}
}
