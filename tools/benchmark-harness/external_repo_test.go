// This test file exercises pinned repository hydration, hidden oracle patches, and the upper-bound replay arm.
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePatchReplayUsesPinnedRepositoryAndPrivateTestPatch(t *testing.T) {
	repository := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repository, 0o750); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, repository, "init", "-q")
	runGitTestCommand(t, repository, "config", "user.email", "benchmark@example.invalid")
	runGitTestCommand(t, repository, "config", "user.name", "Benchmark Test")
	writeTestFile(t, filepath.Join(repository, "go.mod"), "module example.com/replay\n\ngo 1.24\n")
	writeTestFile(t, filepath.Join(repository, "value.go"), "package replay\n\nfunc Value() int { return 1 }\n")
	runGitTestCommand(t, repository, "add", "go.mod", "value.go")
	runGitTestCommand(t, repository, "commit", "-qm", "base")
	base := strings.TrimSpace(string(runGitTestCommand(t, repository, "rev-parse", "HEAD")))
	if err := os.WriteFile(filepath.Join(repository, "value.go"), []byte("package replay\n\nfunc Value() int { return 2 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixPatch := runGitTestCommand(t, repository, "diff", "--binary")
	if err := os.WriteFile(filepath.Join(repository, "value.go"), []byte("package replay\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	testPatch := []byte("diff --git a/value_test.go b/value_test.go\nnew file mode 100644\nindex 0000000..a03a9b1\n--- /dev/null\n+++ b/value_test.go\n@@ -0,0 +1,7 @@\n+package replay\n+\n+import \"testing\"\n+\n+func TestValue(t *testing.T) {\n+\tif Value() != 2 { t.Fatal(Value()) }\n+}\n")
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:      "local-replay",
			Instruction: "Change Value to return 2.",
			Repository:  RepositoryMetadata{URL: repository, BaseCommit: base},
			Oracle:      OracleConfig{Test: TestConfig{PassTests: true, Command: []string{"./..."}}},
		},
		Archive:   ParseArchive([]byte("-- .benchmark/fix.patch --\nmetadata only\n")),
		FixPatch:  fixPatch,
		TestPatch: testPatch,
	}
	result, err := NewRunner(filepath.Join(t.TempDir(), "scratch")).ExecutePatchReplay(context.Background(), task, Target{Harness: "control"}, "small")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Oracle == nil || !result.Oracle.Passed {
		t.Fatalf("patch replay result = %+v", result)
	}
	if result.ExecutionMode != "ground_truth_fix_patch_replay" {
		t.Fatalf("execution mode = %q", result.ExecutionMode)
	}
	if !strings.Contains(result.Diff, "return 2") {
		t.Fatalf("replayed diff does not include fix: %s", result.Diff)
	}
}

func TestMultiSWETxtarKeepsProblemStatementPrivateAndUsesExactText(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "grpc__grpc-go-2951.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	task, err := ParseTask(data)
	if err != nil {
		t.Fatal(err)
	}
	statement, found := archiveFileContents(task.Archive, task.Metadata.ProblemStatementFile)
	if !found || string(statement) != task.Metadata.Instruction {
		t.Fatal("problem statement was not loaded verbatim from private benchmark metadata")
	}
	if len(task.TestPatch) == 0 || len(task.FixPatch) == 0 {
		t.Fatal("official test and fix patches were not loaded")
	}
	workspace := t.TempDir()
	if err := task.ExtractVariantTo(workspace, "small"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(task.Metadata.ProblemStatementFile))); !os.IsNotExist(err) {
		t.Fatalf("private problem statement was exposed to agent workspace: stat err = %v", err)
	}
}

func TestExternalRepositoryPlanUsesThreeArmsAndOrthogonalSkillCondition(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "testdata", "bench")
	options := PlanOptions{
		BenchDir:              fixtureDir,
		TaskIDs:               []string{"grpc__grpc-go-2951"},
		Targets:               []Target{{Harness: "codex", Model: "fixture"}},
		Variants:              []string{"small"},
		Repeats:               1,
		Concurrency:           1,
		SemeditArmRestriction: SemeditArmRestrictWrite,
		SemeditSkill:          true,
	}
	plan, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 3 {
		t.Fatalf("job count = %d, want baseline, agentic, and explicit: %+v", len(plan.Jobs), plan.Jobs)
	}
	byArm := make(map[ArmType]Job, len(plan.Jobs))
	for _, job := range plan.Jobs {
		byArm[job.Arm] = job
	}
	if len(byArm) != 3 || byArm[ArmBaseline].Task == nil || byArm[ArmSemeditAgentic].Task == nil || byArm[ArmSemeditExplicit].Task == nil {
		t.Fatalf("planned arms = %v", mapsKeys(byArm))
	}
	if byArm[ArmBaseline].Execution.Prompt != byArm[ArmSemeditAgentic].Execution.Prompt {
		t.Fatal("baseline and semedit-agentic do not share the issue prompt")
	}
	if byArm[ArmBaseline].SemeditSkill || !byArm[ArmSemeditAgentic].SemeditSkill || byArm[ArmSemeditExplicit].SemeditSkill {
		t.Fatalf("skill condition leaked across arms: baseline=%t agentic=%t explicit=%t", byArm[ArmBaseline].SemeditSkill, byArm[ArmSemeditAgentic].SemeditSkill, byArm[ArmSemeditExplicit].SemeditSkill)
	}
	if byArm[ArmSemeditExplicit].Execution.Prompt != "" {
		t.Fatal("patch replay arm unexpectedly planned an agent prompt")
	}

	options.SemeditSkill = false
	withoutSkill, err := BuildBenchmarkPlan(options)
	if err != nil {
		t.Fatal(err)
	}
	var skillJob Job
	for _, job := range withoutSkill.Jobs {
		if job.Arm == ArmSemeditAgentic {
			skillJob = job
		}
	}
	if skillJob.ID == byArm[ArmSemeditAgentic].ID || skillJob.PairID == byArm[ArmSemeditAgentic].PairID {
		t.Fatal("skill ablation did not produce distinct job and pair identities")
	}
}

func mapsKeys[V any](values map[ArmType]V) []ArmType {
	keys := make([]ArmType, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func runGitTestCommand(t *testing.T, directory string, args ...string) []byte {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

func writeTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
