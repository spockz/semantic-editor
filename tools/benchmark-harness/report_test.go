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
			Oracle:    &OracleResult{Passed: true, Level1Policy: true, Level2AST: true, Level3Build: true, Level4Test: true},
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
	if !strings.Contains(mdVerified, "Small (+Verified) Context Edit Summary") {
		t.Errorf("markdown missing Small (+Verified) Context Edit Summary")
	}
}
