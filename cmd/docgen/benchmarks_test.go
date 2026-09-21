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

func TestRenderDocRunSummaryQualifiesToolUsage(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, "Standard / Small Context",
		&BenchRunResult{Diff: "File main.go modified", ToolCalls: []BenchToolCall{{Name: "view_file", TransportStatus: "succeeded", FunctionalStatus: "succeeded"}, {Name: "replace_file_content", TransportStatus: "succeeded", FunctionalStatus: "succeeded"}}},
		&BenchRunResult{Diff: "File main.go modified", ToolCalls: []BenchToolCall{{Name: "semantic_replace_body", Server: "semedit", TransportStatus: "succeeded", FunctionalStatus: "failed", Failure: "symbol not found", MCPMetrics: &BenchMCPMetrics{TotalMS: 1250, Phases: map[string]BenchMCPPhaseMetric{"verification.after_diagnostics": {DurationMS: 700}, "formatting.organize_imports": {DurationMS: 300}}}}}},
	)

	for _, want := range []string{
		"#### Variant: Standard / Small Context",
		"| # | Vanilla | Semedit MCP |",
		"| 1 | `view_file` (transport: succeeded; functional: succeeded) | `semedit/semantic_replace_body` (transport: succeeded; functional: failed): symbol not found; server: 1.25s (verification: 700ms; formatting: 300ms) |",
		"| 2 | `replace_file_content` (transport: succeeded; functional: succeeded) | — |",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered run summary missing %q", want)
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

func TestRenderDocRunSummaryShowsAbsentTelemetry(t *testing.T) {
	t.Parallel()

	var rendered strings.Builder
	renderDocRunSummary(&rendered, "Standard / Large Context", &BenchRunResult{}, nil)

	for _, want := range []string{
		"#### Variant: Standard / Large Context",
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
