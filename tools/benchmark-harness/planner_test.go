// Package main contains behavioral tests for resolved benchmark planning.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func writePlanningFixture(t *testing.T, dir, id, metadata string) {
	t.Helper()
	path := filepath.Join(dir, id+".txtar")
	content := fmt.Sprintf("task_id: %s\ncategory: test\ninstruction: Test planning.\n%s\noracle:\n  level_3_build:\n    clean_compile: false\n  level_4_test:\n    pass_tests: false\n-- go.mod --\nmodule example.com/%s\ngo 1.23\n", id, metadata, id)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildBenchmarkPlanDefaultsAndContextExclusions(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "small-only", "contexts:\n  - small")
	plan, err := BuildBenchmarkPlan(PlanOptions{BenchDir: dir, OutDir: "", Variants: []string{"small", "large"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 1 || plan.Jobs[0].Context != "small" || plan.Jobs[0].Arm != ArmControl {
		t.Fatalf("unexpected jobs: %+v", plan.Jobs)
	}
	if len(plan.Exclusions) != 1 || !slices.Contains(plan.Exclusions[0].Requested, "large") {
		t.Fatalf("missing large-context exclusion: %+v", plan.Exclusions)
	}
	if plan.Options.Repeats != 1 || plan.Options.Concurrency != 4 || plan.Options.Timeout != 5*time.Minute || plan.Options.SemeditArmRestriction != SemeditArmRestrictWrite {
		t.Fatalf("unexpected defaults: %+v", plan.Options)
	}
}

func TestBuildBenchmarkPlanMixedControlAndAgentMissingPrompt(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "no-prompts", "contexts:\n  - small")
	plan, err := BuildBenchmarkPlan(PlanOptions{BenchDir: dir, OutDir: "", Variants: []string{"small"}, Targets: []Target{{Harness: "control"}, {Harness: "codex"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 1 || plan.Jobs[0].Arm != ArmControl {
		t.Fatalf("control job was lost: %+v", plan.Jobs)
	}
	if len(plan.Exclusions) != 1 || !strings.Contains(plan.Exclusions[0].Reason, "prompt") {
		t.Fatalf("missing prompt exclusion: %+v", plan.Exclusions)
	}
}

func TestBuildBenchmarkPlanRejectsInvalidContexts(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "invalid", "contexts:\n  - medium")
	_, err := BuildBenchmarkPlan(PlanOptions{BenchDir: dir, OutDir: ""})
	if err == nil || !strings.Contains(err.Error(), "unsupported context") {
		t.Fatalf("expected invalid context error, got %v", err)
	}
}

func TestBuildBenchmarkPlanPreservesVerifiedAndUsesConditionIDs(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "agent", "contexts:\n  - small\n  - large\nprompt_variants:\n  default: concise\n")
	options := PlanOptions{BenchDir: dir, OutDir: "", Targets: []Target{{Harness: "codex"}}, Variants: []string{"small-verified:default"}, MCPServerInstructions: MCPServerInstructionsDescriptive}
	plan, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 2 || plan.Jobs[0].Context != "small+verified" || plan.Jobs[0].PairID != plan.Jobs[1].PairID {
		t.Fatalf("verified context or pair not preserved: %+v", plan.Jobs)
	}
	options.MCPServerInstructions = MCPServerInstructionsPrescriptive
	other, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Jobs[0].ID == other.Jobs[0].ID || plan.Jobs[0].PairID == other.Jobs[0].PairID {
		t.Fatal("MCP mode was omitted from planned identity")
	}
}

func TestRenderBenchmarkPlanPropagatesWriterError(t *testing.T) {
	plan := &BenchmarkPlan{Options: PlanOptions{Timeout: time.Second}}
	if err := RenderBenchmarkPlan(failingWriter{}, plan); err == nil {
		t.Fatal("expected writer error")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestBuildBenchmarkPlanRejectsOverlappingResolvedJobs(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "overlap", "contexts:\n  - small\nprompt_variants:\n  default: concise\n")
	_, err := BuildBenchmarkPlan(PlanOptions{BenchDir: dir, OutDir: "", Targets: []Target{{Harness: "codex"}}, Variants: []string{"small", "small:default"}})
	if err == nil || !strings.Contains(err.Error(), "duplicate resolved benchmark job") {
		t.Fatalf("expected duplicate resolved job error, got %v", err)
	}
}

func TestBenchmarkMakeListUsesFlagsBeforePositionalArguments(t *testing.T) {
	output, err := exec.Command("make", "-n", "-f", "Makefile", "list-benchmarks").CombinedOutput()
	if err != nil {
		t.Fatalf("make list-benchmarks: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "--list --dir") {
		t.Fatalf("list target uses unsupported positional flag order:\n%s", output)
	}
}
