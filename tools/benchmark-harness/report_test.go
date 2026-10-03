// These tests protect benchmark condition identity and published measurement semantics.
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
		SemanticToolReflection:        &SemanticToolReflection{WallClock: 2 * time.Second, Turns: 99, ToolCalls: []ToolCall{{Name: "diagnostic-only"}}, Response: "Reason recorded"},
		SemanticIntegrationReflection: &SemanticToolReflection{WallClock: 9 * time.Second, Turns: 999, ToolCalls: []ToolCall{{Name: "integration-only"}}, Response: "I would ask my owner to add semedit."},
		SemanticBatchReflection:       &SemanticToolReflection{WallClock: 2 * time.Second, Turns: 99, ToolCalls: []ToolCall{{Name: "diagnostic-only"}}, Response: "Batch reason recorded"},
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
	for _, key := range []string{"semantic_tool_reflection", "semantic_batch_reflection", "semantic_integration_reflection"} {
		reflection := run[key].(map[string]any)
		for _, metric := range []string{"wall_clock_ms", "turns", "tool_calls", "output_tokens", "prompt_tokens"} {
			if _, present := reflection[metric]; present {
				t.Errorf("%s contains diagnostic metric %s", key, metric)
			}
		}
		if reflection["response"] == "" {
			t.Errorf("%s lost explanation", key)
		}
	}
	if rendered := report.RenderMarkdown(); strings.Contains(rendered, "Reflection wall-clock:") {
		t.Error("Markdown publishes interrogation metrics")
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

type comparisonFixtureSpec struct {
	variant              string
	arm                  ArmType
	turns                int
	wallClock            time.Duration
	promptTokens         int
	cachedPromptTokens   int
	uncachedPromptTokens int
	outputTokens         int
	reasoningTokens      int
	reflections          bool
}

func comparisonFixtureRuns(target Target) []*RunResult {
	specs := []comparisonFixtureSpec{
		{
			variant:              "small",
			arm:                  ArmBaseline,
			turns:                3,
			wallClock:            12 * time.Second,
			promptTokens:         15000,
			cachedPromptTokens:   10000,
			uncachedPromptTokens: 5000,
			outputTokens:         800,
			reasoningTokens:      200,
		},
		{
			variant:              "small",
			arm:                  ArmSemedit,
			turns:                1,
			wallClock:            2 * time.Second,
			promptTokens:         12000,
			cachedPromptTokens:   10000,
			uncachedPromptTokens: 2000,
			outputTokens:         150,
			reasoningTokens:      50,
			reflections:          true,
		},
		{
			variant:              "large",
			arm:                  ArmBaseline,
			turns:                5,
			wallClock:            25 * time.Second,
			promptTokens:         30000,
			cachedPromptTokens:   25000,
			uncachedPromptTokens: 5000,
			outputTokens:         1200,
			reasoningTokens:      300,
		},
		{
			variant:              "large",
			arm:                  ArmSemedit,
			turns:                1,
			wallClock:            2 * time.Second,
			promptTokens:         27000,
			cachedPromptTokens:   25000,
			uncachedPromptTokens: 2000,
			outputTokens:         160,
			reasoningTokens:      50,
		},
	}
	runs := make([]*RunResult, 0, len(specs))
	for _, spec := range specs {
		runs = append(runs, comparisonFixtureRun(target, spec))
	}
	return runs
}

func comparisonFixtureRun(target Target, spec comparisonFixtureSpec) *RunResult {
	run := &RunResult{
		TaskID:               "task-01-rename-local",
		Variant:              spec.variant,
		Target:               target,
		Arm:                  spec.arm,
		Success:              true,
		Turns:                spec.turns,
		WallClock:            spec.wallClock,
		PromptTokens:         spec.promptTokens,
		CachedPromptTokens:   spec.cachedPromptTokens,
		UncachedPromptTokens: spec.uncachedPromptTokens,
		OutputTokens:         spec.outputTokens,
		ReasoningTokens:      spec.reasoningTokens,
		Oracle:               &OracleResult{Passed: true, Level1Policy: true, Level2AST: true, Level3Build: true, Level4Test: true},
	}
	if spec.reflections {
		run.SemanticToolReflection = &SemanticToolReflection{Prompt: "Why did you not use semantic_* tools?", Response: "Ordinary editing seemed simpler.", WallClock: time.Second, Turns: 1}
		run.SemanticBatchReflection = &SemanticToolReflection{Prompt: "Why did you not use semantic_batch?", Response: "The operations were planned separately.", WallClock: time.Second, Turns: 1}
	}
	return run
}

func verifiedComparisonRuns(target Target) []*RunResult {
	baseline := &RunResult{
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
	}
	semedit := &RunResult{
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
		ToolCalls: []ToolCall{{Name: "semantic_rename", Server: "semedit", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed, Failure: "symbol not found"}},
		Oracle:    &OracleResult{Passed: true, Level1Policy: true, Level2AST: true, Level3Build: true, Level4Test: true},
	}
	return []*RunResult{baseline, semedit}
}

func assertMarkdownContains(t *testing.T, markdown string, expected []string) {
	t.Helper()
	for _, text := range expected {
		if !strings.Contains(markdown, text) {
			t.Errorf("markdown missing %q", text)
		}
	}
}

func TestBuildComparisonsAndRenderMarkdown(t *testing.T) {
	target := Target{Harness: "agy", Model: "gemini-3.8-flash-low"}
	runs := comparisonFixtureRuns(target)
	comparisons := BuildComparisons(runs)
	if len(comparisons) != 1 {
		t.Fatalf("expected 1 comparison set, got %d", len(comparisons))
	}
	comparison := comparisons[0]
	if comparison.SmallBaseline == nil || comparison.SmallSemedit == nil || comparison.LargeBaseline == nil || comparison.LargeSemedit == nil {
		t.Fatal("expected all four runs in the comparison set")
	}
	markdown := (&BenchmarkReport{Timestamp: time.Now(), Runs: runs, Comparisons: comparisons}).RenderMarkdown()
	assertMarkdownContains(t, markdown, []string{
		"## Test case: `task-01-rename-local`",
		"### Target: `agy/gemini-3.8-flash-low`",
		"#### Configuration: default prompt · none MCP instructions",
		"Top-Level User Turns",
		"Internal Tool Cycles",
		"Initial Load / Discovery Turns",
		"MCP Discovery / Schema Turns",
		"Total Tool Invocations",
		"Oracle L1: Mutation Policy",
		"Oracle L4: Verification Test",
		"Semedit Tool-Use Reflection",
		"Ordinary editing seemed simpler.",
		"Semedit Batch-Use Reflection",
		"The operations were planned separately.",
	})
}

func TestRenderMarkdownIncludesVerifiedDirectiveAndToolOutcome(t *testing.T) {
	target := Target{Harness: "agy", Model: "gemini-3.8-flash-low"}
	runs := append(comparisonFixtureRuns(target), verifiedComparisonRuns(target)...)
	report := &BenchmarkReport{Timestamp: time.Now(), Runs: runs, Comparisons: BuildComparisons(runs)}
	markdown := report.RenderMarkdown()
	assertMarkdownContains(t, markdown, []string{
		"##### Verified Directive Comparison (+Self-Correction Loop)",
		"###### Verified vs Semedit in Small Context",
		"<div class=\"callout callout-warning\"><div class=\"callout-title\"><span>⚠</span> Why no semantic edit tool was used</div><div class=\"callout-desc\">Ordinary editing seemed simpler.</div></div>",
		"| 1 | `run_command` (outcome unavailable) | `semedit/semantic_rename` (<span role=\"img\" aria-label=\"Transport succeeded\" title=\"Transport succeeded\">✓</span> <span role=\"img\" aria-label=\"Functional failed\" title=\"Functional failed\">✗</span>): symbol not found |",
	})
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

func TestRestrictionPoliciesKeepReportsAndFilesDistinct(t *testing.T) {
	t.Parallel()
	policies := []SemeditArmRestriction{"", SemeditArmRestrictRead, SemeditArmRestrictWrite, SemeditArmRestrictReadWrite}
	runs := make([]*RunResult, 0, 2*len(policies))
	for _, policy := range policies {
		for _, arm := range []ArmType{ArmBaseline, ArmSemedit} {
			runs = append(runs, &RunResult{TaskID: "task-01-rename-local", Target: Target{Harness: "codex"}, Variant: "small:default", PromptVariant: "default", Repeat: 1, Arm: arm, SemeditArmRestrict: policy, SemeditArmRestrictionApplied: arm == ArmSemedit && policy != "", Success: true, Oracle: &OracleResult{Passed: true}, WallClock: 1500 * time.Millisecond})
		}
	}
	report := &BenchmarkReport{Runs: runs, Comparisons: BuildComparisons(runs)}
	if len(report.Comparisons) != 4 {
		t.Fatalf("comparison count = %d, want 4", len(report.Comparisons))
	}
	seen := make(map[SemeditArmRestriction]bool)
	for _, comp := range report.Comparisons {
		seen[comp.SemeditArmRestrict] = true
		if comp.SmallBaseline == nil || comp.SmallSemedit == nil {
			t.Errorf("policy %q lost its pair", comp.SemeditArmRestrict)
			continue
		}
		if comp.SmallBaseline.SemeditArmRestrict != comp.SemeditArmRestrict || comp.SmallSemedit.SemeditArmRestrict != comp.SemeditArmRestrict {
			t.Errorf("policy %q paired across conditions", comp.SemeditArmRestrict)
		}
		if comp.SmallBaseline.SemeditArmRestrictionApplied {
			t.Errorf("baseline policy applied for %q", comp.SemeditArmRestrict)
		}
	}
	for _, policy := range policies {
		if !seen[policy] {
			t.Errorf("missing policy %q", policy)
		}
	}
	data, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	var wire struct {
		FormatVersion int    `json:"format_version"`
		DurationUnit  string `json:"duration_unit"`
		Runs          []struct {
			Policy    string `json:"semedit_arm_restrict"`
			WallClock int    `json:"wall_clock_ms"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.FormatVersion != 2 || wire.DurationUnit != "milliseconds" {
		t.Errorf("duration contract changed: %+v", wire)
	}
	for i, run := range wire.Runs {
		if run.Policy != string(runs[i].SemeditArmRestrict) || run.WallClock != 1500 {
			t.Errorf("run %d lost condition or units: %+v", i, run)
		}
	}
	dir := t.TempDir()
	if err := saveMatrixReports(dir, "../../testdata/bench", runs); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "task-01-rename-local", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Errorf("result files = %d, want 4 policies", len(files))
	}
	if !strings.Contains(report.RenderMarkdown(), "unspecified semedit restriction") {
		t.Error("legacy restriction missing explicit unspecified label")
	}
}

func TestPlannedPhysicalAndGeneratedLargeFixturesRemainDistinct(t *testing.T) {
	t.Parallel()
	runs := []*RunResult{
		{JobID: "generated-small", TaskID: "task-01-rename-local", Variant: "small", Target: Target{Harness: "control"}, Arm: ArmControl},
		{JobID: "generated-large", TaskID: "task-01-rename-local", Variant: "large", Target: Target{Harness: "control"}, Arm: ArmControl},
		{JobID: "physical-large", TaskID: "task-01b-rename-local-large-context", Variant: "large", Target: Target{Harness: "control"}, Arm: ArmControl},
	}
	comparisons := BuildComparisons(runs)
	if len(comparisons) != 2 {
		t.Fatalf("comparisons=%d, want distinct fixtures", len(comparisons))
	}
	count := 0
	for _, comparison := range comparisons {
		if comparison.SmallSemedit != nil {
			count++
		}
		if comparison.LargeSemedit != nil {
			count++
		}
	}
	if count != 3 {
		t.Errorf("published %d of 3 planned observations", count)
	}
	dir := t.TempDir()
	if err := saveMatrixReports(dir, "../../testdata/bench", runs); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("distinct fixture reports=%d, want 2", len(files))
	}
	for _, run := range runs {
		run.JobID = ""
	}
	if len(BuildComparisons(runs)) != 1 {
		t.Error("historical normalization changed")
	}
}

func TestStagedInteractionMetricsSerializePerTurn(t *testing.T) {
	startup := 250 * time.Millisecond
	report := BenchmarkReport{Runs: []*RunResult{{InteractiveMode: "staged", InteractionSteps: []InteractionStep{{Step: 1, TotalEdits: 1, WallClock: 1500 * time.Millisecond, PromptTokens: 5, OutputTokens: 2, ProcessStartToFirstEvent: &startup}, {Step: 2, TotalEdits: 2, WallClock: 900 * time.Millisecond, PromptTokens: 7, OutputTokens: 3}}}}}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Runs []struct {
			InteractiveMode string `json:"interactive_mode"`
			Steps           []struct {
				TotalEdits   int    `json:"total_edits"`
				WallClockMS  int64  `json:"wall_clock_ms"`
				PromptTokens int    `json:"prompt_tokens"`
				OutputTokens int    `json:"output_tokens"`
				StartupMS    *int64 `json:"process_start_to_first_event_ms"`
			} `json:"interaction_steps"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Runs) != 1 || decoded.Runs[0].InteractiveMode != "staged" || len(decoded.Runs[0].Steps) != 2 {
		t.Fatalf("staged report = %+v", decoded)
	}
	first, second := decoded.Runs[0].Steps[0], decoded.Runs[0].Steps[1]
	if first.TotalEdits != 1 || first.WallClockMS != 1500 || first.PromptTokens != 5 || first.OutputTokens != 2 || first.StartupMS == nil || *first.StartupMS != 250 {
		t.Fatalf("first stage metrics = %+v", first)
	}
	if second.TotalEdits != 2 || second.WallClockMS != 900 || second.PromptTokens != 7 || second.OutputTokens != 3 || second.StartupMS != nil {
		t.Fatalf("follow-up stage metrics = %+v", second)
	}
}
