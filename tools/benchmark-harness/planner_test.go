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
	options := PlanOptions{BenchDir: dir, OutDir: "", Targets: []Target{{Harness: "codex"}}, Variants: []string{"small-verified:default"}, MCPServerInstructions: []MCPServerInstructionMode{MCPServerInstructionsDescriptive}}
	plan, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 2 || plan.Jobs[0].Context != "small+verified" || plan.Jobs[0].PairID != plan.Jobs[1].PairID {
		t.Fatalf("verified context or pair not preserved: %+v", plan.Jobs)
	}
	options.MCPServerInstructions = []MCPServerInstructionMode{MCPServerInstructionsPrescriptive}
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

func TestBuildBenchmarkPlanExpandsInstructionModesAndKeepsControlSingle(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "mode-matrix", "contexts:\n  - small\nprompt_variants:\n  default: concise\n")
	plan, err := BuildBenchmarkPlan(PlanOptions{
		BenchDir:              dir,
		OutDir:                "",
		Targets:               []Target{{Harness: "codex"}, {Harness: "control"}},
		Variants:              []string{"small"},
		MCPServerInstructions: []MCPServerInstructionMode{MCPServerInstructionsNone, MCPServerInstructionsDescriptive, MCPServerInstructionsPrescriptive},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 7 {
		t.Fatalf("jobs = %d, want six agent jobs and one control: %+v", len(plan.Jobs), plan.Jobs)
	}
	controls := 0
	armsByMode := make(map[MCPServerInstructionMode]map[ArmType]string)
	for _, job := range plan.Jobs {
		if job.Arm == ArmControl {
			controls++
			if job.PairID != "" || job.MCPServerInstructions != MCPServerInstructionsNone {
				t.Errorf("control inherited a mode cell: %+v", job)
			}
			continue
		}
		if armsByMode[job.MCPServerInstructions] == nil {
			armsByMode[job.MCPServerInstructions] = make(map[ArmType]string)
		}
		armsByMode[job.MCPServerInstructions][job.Arm] = job.PairID
	}
	if controls != 1 {
		t.Errorf("control jobs = %d, want 1", controls)
	}
	for _, mode := range []MCPServerInstructionMode{MCPServerInstructionsNone, MCPServerInstructionsDescriptive, MCPServerInstructionsPrescriptive} {
		arms := armsByMode[mode]
		if arms[ArmBaseline] == "" || arms[ArmBaseline] != arms[ArmSemedit] {
			t.Errorf("mode %q is not an agent pair: %v", mode, arms)
		}
	}
}

func TestBuildBenchmarkPlanCreatesThreeArmExplicitPromptCell(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "explicit-cell", "contexts:\n  - small\nprompt_variants:\n  default: concise\n  explicit: call semantic tools in a fixed order\n")
	plan, err := BuildBenchmarkPlan(PlanOptions{BenchDir: dir, Targets: []Target{{Harness: "codex"}}, Variants: []string{"small:explicit"}, Repeats: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(plan.Jobs), 6; got != want {
		t.Fatalf("jobs = %d, want %d: %+v", got, want, plan.Jobs)
	}
	byPair := make(map[string]map[ArmType]Job)
	for _, job := range plan.Jobs {
		if byPair[job.PairID] == nil {
			byPair[job.PairID] = make(map[ArmType]Job)
		}
		byPair[job.PairID][job.Arm] = job
	}
	if len(byPair) != 2 {
		t.Fatalf("pair count = %d, want two repeats", len(byPair))
	}
	for pairID, jobs := range byPair {
		if len(jobs) != 3 {
			t.Errorf("pair %s has %d arms, want three", pairID, len(jobs))
		}
		if jobs[ArmBaseline].PairID != pairID || jobs[ArmSemedit].PairID != pairID || jobs[ArmSemeditExplicit].PairID != pairID {
			t.Errorf("pair %s does not contain baseline, implicit, and explicit arms: %+v", pairID, jobs)
		}
		if jobs[ArmBaseline].PromptVariant != "" || jobs[ArmSemedit].PromptVariant != "implicit" || jobs[ArmSemeditExplicit].PromptVariant != "explicit" {
			t.Errorf("pair %s prompt variants = %q, %q, %q; want shared baseline, implicit, explicit", pairID, jobs[ArmBaseline].PromptVariant, jobs[ArmSemedit].PromptVariant, jobs[ArmSemeditExplicit].PromptVariant)
		}
	}
}

func TestBuildBenchmarkPlanDefaultsMCPInstructionsToPrescriptive(t *testing.T) {
	dir := t.TempDir()
	writePlanningFixture(t, dir, "default-instructions", "contexts:\n  - small")
	plan, err := BuildBenchmarkPlan(PlanOptions{
		BenchDir: dir,
		OutDir:   "",
		Targets:  []Target{{Harness: "codex"}},
		Variants: []string{"small"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(plan.Options.MCPServerInstructions, []MCPServerInstructionMode{MCPServerInstructionsPrescriptive}) {
		t.Fatalf("plan instruction modes = %v, want [prescriptive]", plan.Options.MCPServerInstructions)
	}
	for _, job := range plan.Jobs {
		if job.MCPServerInstructions != MCPServerInstructionsPrescriptive {
			t.Errorf("job instruction mode = %q, want prescriptive", job.MCPServerInstructions)
		}
	}
}

func TestBuildBenchmarkPlanInjectsSkillOnlyIntoSemeditArms(t *testing.T) {
	for _, promptVariant := range []string{"default", "explicit"} {
		t.Run(promptVariant, func(t *testing.T) {
			assertPlannedSkillCondition(t, promptVariant)
		})
	}
}

func assertPlannedSkillCondition(t *testing.T, promptVariant string) {
	t.Helper()
	dir := t.TempDir()
	writePlanningFixture(t, dir, "skill-"+promptVariant, "contexts:\n  - small\nprompt_variants:\n  default: concise\n  explicit: use the prescribed semantic operations\n")
	options := PlanOptions{
		BenchDir:     dir,
		OutDir:       "",
		Targets:      []Target{{Harness: "codex", Model: "gpt-6-luna", Effort: "medium"}},
		Variants:     []string{"small:" + promptVariant},
		SemeditSkill: true,
		Repeats:      1,
	}
	withSkill, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	options.SemeditSkill = false
	withoutSkill, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	expectedJobs := 2
	if promptVariant == "explicit" {
		expectedJobs = 3
	}
	if len(withSkill.Jobs) != expectedJobs || len(withoutSkill.Jobs) != expectedJobs {
		t.Fatalf("job counts with and without skill = %d and %d, want %d each", len(withSkill.Jobs), len(withoutSkill.Jobs), expectedJobs)
	}
	withByArm := plannedJobsByArm(withSkill.Jobs)
	withoutByArm := plannedJobsByArm(withoutSkill.Jobs)
	assertPlannedSkillArm(t, withByArm[ArmBaseline], withoutByArm[ArmBaseline], false)
	assertPlannedSkillArm(t, withByArm[ArmSemedit], withoutByArm[ArmSemedit], true)
	if promptVariant == "explicit" {
		assertPlannedSkillArm(t, withByArm[ArmSemeditExplicit], withoutByArm[ArmSemeditExplicit], true)
	}
}

func plannedJobsByArm(jobs []Job) map[ArmType]Job {
	byArm := make(map[ArmType]Job, len(jobs))
	for _, job := range jobs {
		byArm[job.Arm] = job
	}
	return byArm
}

func assertPlannedSkillArm(t *testing.T, withSkill, withoutSkill Job, wantSkill bool) {
	t.Helper()
	if withSkill.SemeditSkill != wantSkill || withSkill.Execution.SemeditSkill != wantSkill {
		t.Errorf("%s skill settings = job %t / execution %t, want %t", withSkill.Arm, withSkill.SemeditSkill, withSkill.Execution.SemeditSkill, wantSkill)
	}
	if withSkill.PairID == withoutSkill.PairID {
		t.Errorf("%s pair identity did not record the skill condition", withSkill.Arm)
	}
	if withSkill.Arm == ArmBaseline {
		if withSkill.ID != withoutSkill.ID {
			t.Errorf("baseline identity changed with skill option: %q != %q", withSkill.ID, withoutSkill.ID)
		}
		return
	}
	if withSkill.ID == withoutSkill.ID {
		t.Errorf("%s identity did not record skill injection", withSkill.Arm)
	}
}
