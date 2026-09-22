package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveReportSerializesDurationsAsMilliseconds(t *testing.T) {
	firstEvent := 250 * time.Millisecond
	firstTool := 1250 * time.Millisecond
	report := &BenchmarkReport{Runs: []*RunResult{{
		WallClock:                 1500 * time.Millisecond,
		ProcessStartToFirstEvent:  &firstEvent,
		FirstEventToFirstToolCall: &firstTool,
		InteractionSteps: []InteractionStep{{
			WallClock: 1750 * time.Millisecond,
		}},
		SemanticToolReflection: &SemanticToolReflection{WallClock: 2 * time.Second},
		Oracle: &OracleResult{
			Level1Policy: true,
			Duration:     1750 * time.Millisecond,
		},
		ToolCalls: []ToolCall{{
			MCPMetrics: &MCPMetrics{
				TotalMS: 1250,
				Phases:  map[string]MCPPhaseMetric{"formatting": {DurationMS: 700}},
			},
		}},
	}}}
	jsonPath := filepath.Join(t.TempDir(), "report.json")
	if err := SaveReport(report, jsonPath, ""); err != nil {
		t.Fatalf("save report: %v", err)
	}
	// #nosec G304 -- jsonPath is created beneath this test's temporary directory.
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if got, want := raw["format_version"], float64(benchmarkReportFormatVersion); got != want {
		t.Errorf("format_version = %v, want %v", got, want)
	}
	if got, want := raw["duration_unit"], benchmarkReportDurationUnit; got != want {
		t.Errorf("duration_unit = %v, want %q", got, want)
	}
	run := raw["runs"].([]any)[0].(map[string]any)
	for key, want := range map[string]float64{
		"wall_clock_ms":                     1500,
		"process_start_to_first_event_ms":   250,
		"first_event_to_first_tool_call_ms": 1250,
	} {
		if got := run[key]; got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}
	if got, want := run["interaction_steps"].([]any)[0].(map[string]any)["wall_clock_ms"], float64(1750); got != want {
		t.Errorf("interaction wall_clock_ms = %v, want %v", got, want)
	}
	if got, want := run["semantic_tool_reflection"].(map[string]any)["wall_clock_ms"], float64(2000); got != want {
		t.Errorf("reflection wall_clock_ms = %v, want %v", got, want)
	}
	if got, want := run["oracle"].(map[string]any)["duration_ms"], float64(1750); got != want {
		t.Errorf("oracle duration_ms = %v, want %v", got, want)
	}
	metrics := run["tool_calls"].([]any)[0].(map[string]any)["mcp_metrics"].(map[string]any)
	if got, want := metrics["total_ms"], float64(1250); got != want {
		t.Errorf("MCP total_ms = %v, want %v", got, want)
	}
	if got, want := metrics["phases"].(map[string]any)["formatting"].(map[string]any)["duration_ms"], float64(700); got != want {
		t.Errorf("MCP phase duration_ms = %v, want %v", got, want)
	}
}

func TestBuildComparisonsAndRenderMarkdown(t *testing.T) {
	target := Target{Harness: "agy", Model: "gemini-3.8-flash-low"}

	runs := []*RunResult{ //nolint:prealloc // test data slice
		{
			TaskID:               "task-01-rename-local",
			Variant:              "small",
			Target:               target,
			Arm:                  ArmBaseline,
			Success:              true,
			Turns:                3,
			WallClock:            12 * time.Second,
			PromptTokens:         15000,
			CachedPromptTokens:   10000,
			UncachedPromptTokens: 5000,
			OutputTokens:         800,
			ReasoningTokens:      200,
			Oracle: &OracleResult{
				Passed:       true,
				Level1Policy: true,
				Level2AST:    true,
				Level3Build:  true,
				Level4Test:   true,
			},
		},
		{
			TaskID:               "task-01-rename-local",
			Variant:              "small",
			Target:               target,
			Arm:                  ArmSemedit,
			Success:              true,
			Turns:                1,
			WallClock:            2 * time.Second,
			PromptTokens:         12000,
			CachedPromptTokens:   10000,
			UncachedPromptTokens: 2000,
			OutputTokens:         150,
			ReasoningTokens:      50,
			Oracle: &OracleResult{
				Passed:       true,
				Level1Policy: true,
				Level2AST:    true,
				Level3Build:  true,
				Level4Test:   true,
			},
			SemanticToolReflection: &SemanticToolReflection{
				Prompt:    "Why did you not use semantic_* tools?",
				Response:  "Ordinary editing seemed simpler.",
				WallClock: time.Second,
				Turns:     1,
			},
			SemanticBatchReflection: &SemanticToolReflection{
				Prompt:    "Why did you not use semantic_batch?",
				Response:  "The operations were planned separately.",
				WallClock: time.Second,
				Turns:     1,
			},
		},
		{
			TaskID:               "task-01-rename-local",
			Variant:              "large",
			Target:               target,
			Arm:                  ArmBaseline,
			Success:              true,
			Turns:                5,
			WallClock:            25 * time.Second,
			PromptTokens:         30000,
			CachedPromptTokens:   25000,
			UncachedPromptTokens: 5000,
			OutputTokens:         1200,
			ReasoningTokens:      300,
			Oracle: &OracleResult{
				Passed:       true,
				Level1Policy: true,
				Level2AST:    true,
				Level3Build:  true,
				Level4Test:   true,
			},
		},
		{
			TaskID:               "task-01-rename-local",
			Variant:              "large",
			Target:               target,
			Arm:                  ArmSemedit,
			Success:              true,
			Turns:                1,
			WallClock:            2 * time.Second,
			PromptTokens:         27000,
			CachedPromptTokens:   25000,
			UncachedPromptTokens: 2000,
			OutputTokens:         160,
			ReasoningTokens:      50,
			Oracle: &OracleResult{
				Passed:       true,
				Level1Policy: true,
				Level2AST:    true,
				Level3Build:  true,
				Level4Test:   true,
			},
		},
	}

	comps := BuildComparisons(runs)
	if len(comps) != 1 {
		t.Fatalf("expected 1 comparison set, got %d", len(comps))
	}

	comp := comps[0]
	if comp.SmallBaseline == nil || comp.SmallSemedit == nil || comp.LargeBaseline == nil || comp.LargeSemedit == nil {
		t.Fatalf("expected all 4 runs populated in comparison set")
	}

	report := &BenchmarkReport{
		Timestamp:   time.Now(),
		Runs:        runs,
		Comparisons: comps,
	}

	md := report.RenderMarkdown()
	if !strings.Contains(md, "## Test case: `task-01-rename-local`") {
		t.Errorf("markdown missing testcase heading")
	}
	if !strings.Contains(md, "### Target: `agy/gemini-3.8-flash-low`") {
		t.Errorf("markdown missing target heading")
	}
	if !strings.Contains(md, "#### Configuration: default prompt · none MCP instructions") {
		t.Errorf("markdown missing configuration heading")
	}
	if !strings.Contains(md, "Top-Level User Turns") {
		t.Errorf("markdown missing Top-Level User Turns row")
	}
	if !strings.Contains(md, "Internal Tool Cycles") {
		t.Errorf("markdown missing Internal Tool Cycles row")
	}
	if !strings.Contains(md, "Initial Load / Discovery Turns") {
		t.Errorf("markdown missing Initial Load / Discovery Turns row")
	}
	if !strings.Contains(md, "MCP Discovery / Schema Turns") {
		t.Errorf("markdown missing MCP Discovery / Schema Turns row")
	}
	if !strings.Contains(md, "Total Tool Invocations") {
		t.Errorf("markdown missing Total Tool Invocations row")
	}
	if !strings.Contains(md, "Oracle L1: Mutation Policy") {
		t.Errorf("markdown missing Oracle L1 row")
	}
	if !strings.Contains(md, "Oracle L4: Verification Test") {
		t.Errorf("markdown missing Oracle L4 row")
	}
	if !strings.Contains(md, "Semedit Tool-Use Reflection") || !strings.Contains(md, "Ordinary editing seemed simpler.") {
		t.Errorf("markdown missing semantic tool-use reflection: %s", md)
	}
	if !strings.Contains(md, "Semedit Batch-Use Reflection") || !strings.Contains(md, "The operations were planned separately.") {
		t.Errorf("markdown missing semantic batch-use reflection: %s", md)
	}

	// Add verified runs to assert verified rendering
	runs = append(runs,
		&RunResult{
			TaskID:    "task-01-rename-local",
			Variant:   "small+verified",
			Target:    target,
			Arm:       ArmBaseline,
			Success:   true,
			Turns:     4,
			WallClock: 18 * time.Second,
			Prompt:    "Rename Server.oldName to newName. Verify with go test.",
			Diff:      "api/server.go: rename Server.oldName to Server.newName",
			ToolsUsed: []string{"run_command", "replace_file_content"},
			Oracle:    &OracleResult{Passed: true, Level1Policy: true, Level2AST: true, Level3Build: true, Level4Test: true},
		},
		&RunResult{
			TaskID:    "task-01-rename-local",
			Variant:   "small+verified",
			Target:    target,
			Arm:       ArmSemedit,
			Success:   true,
			Turns:     1,
			WallClock: 2 * time.Second,
			Prompt:    "Rename Server.oldName to newName. Verify with go test.",
			Diff:      "api/server.go: semantic_rename Server.oldName -> newName",
			ToolsUsed: []string{"semantic_rename"},
			ToolCalls: []ToolCall{{
				Name:             "semantic_rename",
				Server:           "semedit",
				TransportStatus:  ToolCallStatusSucceeded,
				FunctionalStatus: ToolCallStatusFailed,
				Failure:          "symbol not found",
			}},
			Oracle: &OracleResult{Passed: true, Level1Policy: true, Level2AST: true, Level3Build: true, Level4Test: true},
		},
	)

	reportVerified := &BenchmarkReport{
		Timestamp:   time.Now(),
		Runs:        runs,
		Comparisons: BuildComparisons(runs),
	}
	mdVerified := reportVerified.RenderMarkdown()
	if !strings.Contains(mdVerified, "##### Verified Directive Comparison (+Self-Correction Loop)") {
		t.Errorf("markdown missing Verified Directive Comparison table")
	}
	if !strings.Contains(mdVerified, "###### Verified vs Semedit in Small Context") {
		t.Errorf("markdown missing verified small-context variant")
	}
	if !strings.Contains(mdVerified, "<div class=\"callout callout-warning\"><div class=\"callout-title\"><span>⚠</span> Why no semantic edit tool was used</div><div class=\"callout-desc\">Ordinary editing seemed simpler.</div></div>") {
		t.Errorf("markdown missing semantic tool-use diagnostic reason: %s", mdVerified)
	}
	if !strings.Contains(mdVerified, "| 1 | `run_command` (outcome unavailable) | `semedit/semantic_rename` (<span role=\"img\" aria-label=\"Transport succeeded\" title=\"Transport succeeded\">✓</span> <span role=\"img\" aria-label=\"Functional failed\" title=\"Functional failed\">✗</span>): symbol not found |") {
		t.Errorf("markdown missing tool call outcome: %s", mdVerified)
	}
}

func TestBuildComparisonsPreservesIndependentRepeats(t *testing.T) {
	target := Target{Harness: string(HarnessOpenCode), Model: "openrouter/cohere/north-mini-code:free"}
	runs := []*RunResult{
		{TaskID: "task-01-rename-local", Variant: "small", Target: target, Repeat: 1, Arm: ArmBaseline},
		{TaskID: "task-01-rename-local", Variant: "small", Target: target, Repeat: 1, Arm: ArmSemedit},
		{TaskID: "task-01-rename-local", Variant: "small", Target: target, Repeat: 2, Arm: ArmBaseline},
		{TaskID: "task-01-rename-local", Variant: "small", Target: target, Repeat: 2, Arm: ArmSemedit},
	}
	comparisons := BuildComparisons(runs)
	if got, want := len(comparisons), 2; got != want {
		t.Fatalf("comparison count = %d, want %d", got, want)
	}
	if comparisons[0].Repeat != 1 || comparisons[1].Repeat != 2 {
		t.Fatalf("repeat identities = %d, %d; want 1, 2", comparisons[0].Repeat, comparisons[1].Repeat)
	}
	markdown := (&BenchmarkReport{Comparisons: comparisons}).RenderMarkdown()
	if !strings.Contains(markdown, "repeat 1") || !strings.Contains(markdown, "repeat 2") {
		t.Fatalf("markdown does not identify repeats: %s", markdown)
	}
}

func TestRenderSemanticToolCalloutReportsUnavailableToolsAsError(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderSemanticToolCallout(&rendered, "No Semedit MCP tools were available to the agent.")

	if !strings.Contains(rendered.String(), "callout callout-error") {
		t.Errorf("unavailable semantic tools must render an error callout: %s", rendered.String())
	}
}

func TestFormatDeltaColorsVerifiedMCPBenefits(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		pct       float64
		diff      float64
		direction deltaDirection
		verified  bool
		want      string
	}{
		{name: "verified lower metric", pct: -25, diff: -25, direction: deltaLowerIsBetter, verified: true, want: "benchmark-delta-positive"},
		{name: "unverified lower metric", pct: -25, diff: -25, direction: deltaLowerIsBetter, want: "N/A"},
		{name: "unverified baseline advantage", pct: 25, diff: 25, direction: deltaLowerIsBetter, want: "N/A"},
		{name: "baseline lower metric", pct: 25, diff: 25, direction: deltaLowerIsBetter, verified: true, want: "benchmark-delta-negative"},
		{name: "verified cached-token gain", pct: 25, diff: 25, direction: deltaHigherIsBetter, verified: true, want: "benchmark-delta-positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := formatDelta(test.pct, test.diff, test.direction, test.verified); !strings.Contains(got, test.want) {
				t.Errorf("delta = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMCPColumnHeaderQualifiesOnlyUnverifiedRuns(t *testing.T) {
	t.Parallel()

	if got, want := mcpColumnHeader("Small", &RunResult{}), `<span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small)`; got != want {
		t.Errorf("unverified MCP header = %q, want %q", got, want)
	}
	if got, want := mcpColumnHeader("Large", &RunResult{MCPVerified: true}), "MCP (Large)"; got != want {
		t.Errorf("verified MCP header = %q, want %q", got, want)
	}
	if got, want := mcpColumnHeader("Large", nil), "MCP (Large)"; got != want {
		t.Errorf("missing MCP header = %q, want %q", got, want)
	}
}

func TestCachedToUncachedTokenRatioFavorsHigherCachedShare(t *testing.T) {
	t.Parallel()

	base := &RunResult{CachedPromptTokens: 136192, UncachedPromptTokens: 31107}
	mcp := &RunResult{CachedPromptTokens: 161536, UncachedPromptTokens: 27570, MCPVerified: true}
	row := formatCachedToUncachedRatioRow(base, mcp, nil, nil)
	for _, want := range []string{"**Cached vs Uncached Token Ratio**", "4.38:1", "5.86:1", "benchmark-delta-positive"} {
		if !strings.Contains(row, want) {
			t.Errorf("ratio row = %q, want %q", row, want)
		}
	}
}

func TestRenderToolCallComparisonRendersShellCommandsAsCopyableBlocks(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderToolCallComparison(&rendered,
		&RunResult{ToolCalls: []ToolCall{{Name: "/bin/zsh -lc 'make check && go test ./... || true'", TransportStatus: ToolCallStatus("timeout"), FunctionalStatus: ToolCallStatusFailed}}},
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

func TestRenderToolCallComparisonRendersToolArguments(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderToolCallComparison(&rendered, nil, &RunResult{ToolCalls: []ToolCall{{
		Name:             "semantic_rename",
		Server:           "semedit",
		Arguments:        json.RawMessage(`{"symbol":"Old","to":"New"}`),
		TransportStatus:  ToolCallStatusSucceeded,
		FunctionalStatus: ToolCallStatusSucceeded,
	}}})

	for _, want := range []string{
		"<code>semedit/semantic_rename</code>",
		"<pre class=\"benchmark-tool-arguments\"><code class=\"language-json\">{\n  &#34;symbol&#34;: &#34;Old&#34;,\n  &#34;to&#34;: &#34;New&#34;\n}</code></pre>",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered tool arguments missing %q", want)
		}
	}
}

func TestWritePromptQuotesEveryLine(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	writePrompt(&rendered, "Vanilla LLM Prompt", "First paragraph.\n\nSecond paragraph.")

	const want = "**Vanilla LLM Prompt**:\n> First paragraph.\n>\n> Second paragraph.\n\n"
	if got := rendered.String(); got != want {
		t.Errorf("quoted prompt = %q, want %q", got, want)
	}
}

func TestBuildComparisonsSeparatesMCPServerInstructionModes(t *testing.T) {
	t.Parallel()

	target := Target{Harness: "codex", Model: "gpt-5.6-luna", Effort: "medium"}
	runs := []*RunResult{
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmBaseline, MCPServerInstructions: MCPServerInstructionsNone},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmSemedit, MCPServerInstructions: MCPServerInstructionsNone},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmBaseline, MCPServerInstructions: MCPServerInstructionsDescriptive},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmSemedit, MCPServerInstructions: MCPServerInstructionsDescriptive},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmBaseline, MCPServerInstructions: MCPServerInstructionsPrescriptive},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmSemedit, MCPServerInstructions: MCPServerInstructionsPrescriptive},
	}

	comparisons := BuildComparisons(runs)
	if len(comparisons) != 3 {
		t.Fatalf("comparison count = %d, want 3", len(comparisons))
	}
	for _, comparison := range comparisons {
		if comparison.SmallBaseline == nil || comparison.SmallSemedit == nil {
			t.Errorf("incomplete comparison for instruction mode %q", comparison.MCPServerInstructions)
		}
	}

	markdown := (&BenchmarkReport{Comparisons: comparisons}).RenderMarkdown()
	if got := strings.Count(markdown, "### Target:"); got != 1 {
		t.Errorf("target heading count = %d, want 1", got)
	}
	if got := strings.Count(markdown, "#### Configuration:"); got != 3 {
		t.Errorf("configuration heading count = %d, want 3", got)
	}
}

func TestBuildComparisonsDoesNotGroupByProvenance(t *testing.T) {
	t.Parallel()

	target := Target{Harness: "codex"}
	runs := []*RunResult{
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmBaseline, Provenance: ProvenanceSet{"parser-limit": "16MiB"}},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmSemedit, Provenance: ProvenanceSet{"parser-limit": "16MiB"}},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmBaseline, Provenance: ProvenanceSet{"parser-limit": "32MiB"}},
		{TaskID: "task-11-mixed-sink-api-migration", Variant: "small", Target: target, Arm: ArmSemedit, Provenance: ProvenanceSet{"parser-limit": "32MiB"}},
	}

	comparisons := BuildComparisons(runs)
	if got := len(comparisons); got != 1 {
		t.Fatalf("comparison count = %d, want one result cell", got)
	}
	if got := comparisons[0].Provenance["parser-limit"]; got != "" {
		t.Errorf("comparison retained differing parser limit %q", got)
	}
}

func TestEscapeToolCallTableCell(t *testing.T) {
	t.Parallel()

	if got, want := escapeToolCallTableCell("`rg one |\nrg two`"), "`rg one \\| rg two`"; got != want {
		t.Errorf("escaped table cell = %q, want %q", got, want)
	}
}

func TestFormatMCPMetrics(t *testing.T) {
	t.Parallel()

	got := formatMCPMetrics(&MCPMetrics{TotalMS: 1250, Phases: map[string]MCPPhaseMetric{
		"verification.before_diagnostics": {DurationMS: 700},
		"formatting.gofmt":                {DurationMS: 300},
	}})
	if want := "; server: 1.25s (verification: 700ms; formatting: 300ms)"; got != want {
		t.Errorf("MCP timing summary = %q, want %q", got, want)
	}
}
