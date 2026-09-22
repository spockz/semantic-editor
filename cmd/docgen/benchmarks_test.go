// Tests ensure generated benchmark documentation excludes harness failures while retaining evaluated oracle failures.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		"<tr><td>1</td><td><span role=\"img\" aria-label=\"Transport timeout\" title=\"Transport timeout\">⏱</span> <span role=\"img\" aria-label=\"Functional failed\" title=\"Functional failed\">✗</span><pre class=\"benchmark-shell-command\"><code class=\"language-shell\">/bin/zsh -lc &#39;make check &amp;&amp; \\\ngo test ./... || \\\ntrue&#39;</code></pre></td><td>not published</td></tr>",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered shell command missing %q", want)
		}
	}
	if strings.Contains(rendered.String(), "```shell") {
		t.Error("shell command must remain inside its table cell")
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
		"<pre class=\"benchmark-tool-arguments\"><code class=\"language-json\">{\n  &#34;symbol&#34;: &#34;Old&#34;,\n  &#34;to&#34;: &#34;New&#34;\n}</code></pre>",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered tool arguments missing %q", want)
		}
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
	for _, want := range []string{"semedit/semantic_rename", "make check", "semedit/semantic_verify"} {
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

	for _, want := range []string{"table-layout: fixed", "benchmark-tool-arguments", "max-width: 32rem", "white-space: pre-wrap", "overflow-wrap: anywhere", "word-break: break-word", ".benchmark-delta-positive", ".benchmark-delta-negative"} {
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
