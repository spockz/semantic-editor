package main

import (
	"strings"
	"testing"
	"time"
)

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
	if !strings.Contains(md, "Task: `task-01-rename-local`") {
		t.Errorf("markdown missing task title")
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
	if !strings.Contains(mdVerified, "Verified Directive Comparison (+Self-Correction Loop)") {
		t.Errorf("markdown missing Verified Directive Comparison table")
	}
	if !strings.Contains(mdVerified, "Variant: Verified / Small Context") {
		t.Errorf("markdown missing verified small-context variant")
	}
	if !strings.Contains(mdVerified, "| 1 | `run_command` (outcome unavailable) | `semedit/semantic_rename` (transport: succeeded; functional: failed): symbol not found |") {
		t.Errorf("markdown missing tool call outcome: %s", mdVerified)
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
