// This test file exercises pinned repository hydration, hidden oracle patches, and the upper-bound replay arm.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
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
	writeTestFile(t, filepath.Join(repository, "value_test.go"), "package replay\n\nimport \"testing\"\n\nfunc TestVisible(t *testing.T) {\n\tif Value() != 1 { t.Fatal(Value()) }\n}\n")
	runGitTestCommand(t, repository, "add", "go.mod", "value.go", "value_test.go")
	runGitTestCommand(t, repository, "commit", "-qm", "base")
	base := strings.TrimSpace(string(runGitTestCommand(t, repository, "rev-parse", "HEAD")))
	if err := os.WriteFile(filepath.Join(repository, "value.go"), []byte("package replay\n\nfunc Value() int { return 2 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "value_test.go"), []byte("package replay\n\nimport \"testing\"\n\nfunc TestVisible(t *testing.T) {\n\tif Value() != 2 { t.Fatal(Value()) }\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixPatch := runGitTestCommand(t, repository, "diff", "--binary")
	if err := os.WriteFile(filepath.Join(repository, "value.go"), []byte("package replay\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "value_test.go"), []byte("package replay\n\nimport \"testing\"\n\nfunc TestVisible(t *testing.T) {\n\tif Value() != 1 { t.Fatal(Value()) }\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	testPatch := []byte("diff --git a/acceptance_test.go b/acceptance_test.go\nnew file mode 100644\nindex 0000000..a03a9b1\n--- /dev/null\n+++ b/acceptance_test.go\n@@ -0,0 +1,7 @@\n+package replay\n+\n+import \"testing\"\n+\n+func TestValue(t *testing.T) {\n+\tif Value() != 2 { t.Fatal(Value()) }\n+}\n")
	task := &Task{
		Metadata: TaskMetadata{
			TaskID:      "local-replay",
			Instruction: "Change Value to return 2.",
			Repository:  RepositoryMetadata{URL: repository, BaseCommit: base},
			Oracle: OracleConfig{
				MutationPolicy: MutationPolicyConfig{ForbidTestEdits: true},
				Test:           TestConfig{PassTests: true, Command: []string{"./..."}},
			},
		},
		Archive:   ParseArchive([]byte("-- .benchmark/fix.patch --\nmetadata only\n")),
		FixPatch:  fixPatch,
		TestPatch: testPatch,
	}
	scratch := filepath.Join(t.TempDir(), "scratch")
	cacheDir := filepath.Join(scratch, "cache", "repos", "repo", base)
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0o750); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, repository, "clone", "--no-hardlinks", repository, cacheDir)
	result, err := NewRunner(scratch).ExecutePatchReplay(context.Background(), task, Target{Harness: "control"}, "small")
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

	task.TestPatch = []byte("invalid test patch\n")
	result, err = NewRunner(scratch).ExecutePatchReplay(t.Context(), task, Target{Harness: "control"}, "small")
	if err == nil || result == nil || result.Success || result.Oracle != nil || !strings.Contains(result.Error, "prepare oracle tests") {
		t.Fatalf("patch setup failure was not retained as a harness error: result=%+v err=%v", result, err)
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
	task, specification, options := externalRepositoryPlanningFixture()
	jobs, err := makeExternalRepositoryJobs(specification, options)
	if err != nil {
		t.Fatal(err)
	}
	byArm := externalRepositoryJobsByArm(t, jobs)
	assertExternalRepositoryPrompts(t, task, byArm)
	assertExternalRepositoryJobSettings(t, options, byArm)
	assertExternalRepositoryPairing(t, options, byArm, jobs)
	assertExternalRepositoryArmRotation(t, specification, options)
	assertExternalRepositoryRequiresRecipe(t, task, specification, options)
	assertExternalRepositorySkillAblation(t, specification, options, byArm)
}

func TestEvaluateInstallsMultiSWETestsInsideNestedCheckout(t *testing.T) {
	patch := []byte("diff --git a/acceptance_test.go b/acceptance_test.go\nnew file mode 100644\n--- /dev/null\n+++ b/acceptance_test.go\n@@ -0,0 +1,7 @@\n+package replay\n+\n+import \"testing\"\n+\n+func TestAcceptance(t *testing.T) {\n+ if Value() != 2 { t.Fatal(\"acceptance failure\") }\n+}\n")
	for _, testCase := range []struct {
		name       string
		value      string
		conflict   bool
		wantPassed bool
	}{
		{name: "unsolved", value: "1"},
		{name: "solved", value: "2", wantPassed: true},
		{name: "test installation fails", value: "2", conflict: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			workspace := t.TempDir()
			runGitTestCommand(t, workspace, "init", "-q")
			writeTestFile(t, filepath.Join(workspace, "go.mod"), "module example.com/replay\n\ngo 1.24\n")
			writeTestFile(t, filepath.Join(workspace, "value.go"), "package replay\n\nfunc Value() int { return "+testCase.value+" }\n")
			if testCase.conflict {
				writeTestFile(t, filepath.Join(workspace, "acceptance_test.go"), "package replay\n")
			}
			task := &Task{
				Metadata:  TaskMetadata{Oracle: OracleConfig{Test: TestConfig{PassTests: true, Command: []string{"./..."}}}},
				TestPatch: patch,
			}
			result, err := Evaluate(t.Context(), task, workspace, nil)
			if testCase.conflict {
				if err == nil || result != nil || !strings.Contains(err.Error(), "prepare oracle tests") {
					t.Fatalf("test setup produced a task verdict: result=%+v err=%v", result, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Passed != testCase.wantPassed {
				t.Fatalf("acceptance verdict=%+v, want passed=%t", result, testCase.wantPassed)
			}
			if !testCase.wantPassed && (result.FailureStage != "level_4_test" || !strings.Contains(result.ErrorMessage, "acceptance failure")) {
				t.Fatalf("unsolved task did not execute acceptance test: %+v", result)
			}
			if _, err := os.Stat(filepath.Join(workspace, "acceptance_test.go")); !os.IsNotExist(err) {
				t.Fatalf("private acceptance test reached agent workspace: %v", err)
			}
		})
	}
}

func TestEvaluateExternalRepositoryAppliesTestsBeforeCandidatePatch(t *testing.T) {
	repository, task := externalOracleFixture(t)
	writeTestFile(t, filepath.Join(repository, "value.go"), "package replay\n\nfunc Value() int { return helper() }\n")
	writeTestFile(t, filepath.Join(repository, "helper.go"), "package replay\n\nfunc helper() int { return 2 }\n")
	task.TestPatch = []byte("diff --git a/value_test.go b/value_test.go\nindex 885f375..46d3a13 100644\n--- a/value_test.go\n+++ b/value_test.go\n@@ -5,5 +5,5 @@ import \"testing\"\n func TestValue(t *testing.T) {\n-\tif Value() != 1 { t.Fatal(Value()) }\n+\tif Value() != 2 { t.Fatal(Value()) }\n }\n")

	originalTest, err := os.ReadFile(filepath.Join(repository, "value_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(repository, "value_test.go"), strings.Replace(string(originalTest), "Value() != 1", "Value() != 2", 1))
	task.TestPatch = runGitTestCommand(t, repository, "diff", "--", "value_test.go")
	writeTestFile(t, filepath.Join(repository, "value_test.go"), string(originalTest))

	if mkdirErr := os.MkdirAll(filepath.Join(repository, ".git", "refs", "tags"), 0o750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	writeTestFile(t, filepath.Join(repository, ".git", "refs", "tags", "unrelated-missing-object"), "71adf79396437ec5d5d35a8dc197ef75728e4b66\n")

	result, err := Evaluate(t.Context(), task, repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || !result.Level4Test {
		t.Fatalf("oracle did not apply test patch before candidate patch: %+v", result)
	}
	visibleTest, err := os.ReadFile(filepath.Join(repository, "value_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(visibleTest), "Value() != 1") {
		t.Fatalf("private test patch changed visible workspace: %s", visibleTest)
	}
}

func TestEvaluateExternalRepositoryRejectsCandidateTestEdits(t *testing.T) {
	repository, task := externalOracleFixture(t)
	writeTestFile(t, filepath.Join(repository, "value_test.go"), "package replay\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {\n\tif Value() != 3 { t.Fatal(Value()) }\n}\n")

	result, err := Evaluate(t.Context(), task, repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || result.FailureStage != "level_1_mutation_policy" || !strings.Contains(result.ErrorMessage, "value_test.go") {
		t.Fatalf("candidate test edit was not rejected by the external task policy: %+v", result)
	}
}

func externalOracleFixture(t *testing.T) (string, *Task) {
	t.Helper()
	repository := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repository, 0o750); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, repository, "init", "-q")
	runGitTestCommand(t, repository, "config", "user.email", "benchmark@example.invalid")
	runGitTestCommand(t, repository, "config", "user.name", "Benchmark Test")
	writeTestFile(t, filepath.Join(repository, "go.mod"), "module example.com/replay\n\ngo 1.24\n")
	writeTestFile(t, filepath.Join(repository, "value.go"), "package replay\n\nfunc Value() int { return 1 }\n")
	writeTestFile(t, filepath.Join(repository, "value_test.go"), "package replay\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {\n\tif Value() != 1 { t.Fatal(Value()) }\n}\n")
	runGitTestCommand(t, repository, "add", "go.mod", "value.go", "value_test.go")
	runGitTestCommand(t, repository, "commit", "-qm", "base")
	base := strings.TrimSpace(string(runGitTestCommand(t, repository, "rev-parse", "HEAD")))
	task := &Task{Metadata: TaskMetadata{
		TaskID:     "external-oracle",
		Repository: RepositoryMetadata{URL: repository, BaseCommit: base},
		Oracle: OracleConfig{
			MutationPolicy: MutationPolicyConfig{ForbidTestEdits: true},
			Test:           TestConfig{PassTests: true, Command: []string{"./..."}},
		},
	}}
	return repository, task
}

func TestMultiSWEFixturesDeclareProgressiveHints(t *testing.T) {
	tasks := []string{"grpc__grpc-go-2735", "grpc__grpc-go-2951", "grpc__grpc-go-3361", "zeromicro__go-zero-2463", "zeromicro__go-zero-964"}
	for _, taskID := range tasks {
		t.Run(taskID, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", taskID+".txtar"))
			if err != nil {
				t.Fatal(err)
			}
			task, err := ParseTask(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(task.Metadata.InteractiveFollowups) != 3 {
				t.Fatalf("hints = %d, want 3", len(task.Metadata.InteractiveFollowups))
			}
			assertExternalTestEditPolicy(t, task)
			for index, hint := range task.Metadata.InteractiveFollowups {
				if strings.TrimSpace(hint) == "" {
					t.Errorf("hint %d is blank", index+1)
				}
				if strings.Contains(hint, "diff --git") || strings.Contains(hint, ".benchmark/") {
					t.Errorf("hint %d exposes private patch material: %q", index+1, hint)
				}
			}
		})
	}
}

func assertExternalTestEditPolicy(t *testing.T, task *Task) {
	t.Helper()
	if !task.Metadata.Oracle.MutationPolicy.ForbidTestEdits {
		t.Fatal("external Multi-SWE task does not forbid candidate test edits")
	}
	prompt, err := prepareAgentPromptWithPolicy(task, ArmBaseline, "small", &RunResult{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Do not edit tests.") {
		t.Fatalf("external baseline prompt omits the test-edit constraint: %q", prompt)
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

func externalRepositoryPlanningFixture() (*Task, Job, PlanOptions) {
	task := &Task{Metadata: TaskMetadata{
		TaskID:      "external-task",
		Instruction: "Fix the reported issue.",
		Repository:  RepositoryMetadata{URL: "https://example.invalid/repo"},
		PromptVariants: map[string]string{
			"implicit": "Use an unrelated prompt variant.",
			"explicit": "Run semantic_lookup for the declaration, then use one ordered semantic_batch for the required edits.",
		},
	}}
	specification := Job{
		TaskID:        task.Metadata.TaskID,
		FixturePath:   "external-task.txtar",
		Context:       "small",
		PromptVariant: "implicit",
		Target:        Target{Harness: "codex", Model: "fixture"},
		Repeat:        1,
		Task:          task,
	}
	options := PlanOptions{
		SemeditArmRestriction: SemeditArmRestrictWrite,
		SemeditSkill:          true,
	}
	return task, specification, options
}

func externalRepositoryJobsByArm(t *testing.T, jobs []Job) map[ArmType]Job {
	t.Helper()
	if len(jobs) != 3 {
		t.Fatalf("job count = %d, want three external arms: %+v", len(jobs), jobs)
	}
	byArm := make(map[ArmType]Job, len(jobs))
	for _, job := range jobs {
		byArm[job.Arm] = job
	}
	if len(byArm) != 3 {
		t.Fatalf("planned arms = %v", mapsKeys(byArm))
	}
	return byArm
}

func assertExternalRepositoryPrompts(t *testing.T, task *Task, byArm map[ArmType]Job) {
	t.Helper()
	baseline, agentic, explicit := byArm[ArmBaseline], byArm[ArmSemeditAgentic], byArm[ArmSemeditExplicit]
	if baseline.Execution.Prompt != task.Metadata.Instruction {
		t.Error("baseline must retain the original issue prompt")
	}
	if !strings.HasPrefix(agentic.Execution.Prompt, task.Metadata.Instruction) || strings.Contains(agentic.Execution.Prompt, "unrelated prompt variant") {
		t.Error("semedit-agentic must retain the original issue prompt before policy steering")
	}
	explicitRecipe := task.Metadata.PromptVariants["explicit"]
	if strings.TrimSpace(explicitRecipe) == "" || !strings.HasPrefix(explicit.Execution.Prompt, explicitRecipe) {
		t.Errorf("explicit prompt does not start with its required recipe: %q", explicit.Execution.Prompt)
	}
	if agentic.PromptVariant != "implicit" || agentic.Execution.Variant != "small:implicit" || explicit.PromptVariant != "explicit" || explicit.Execution.Variant != "small:explicit" {
		t.Errorf("prompt variants = agentic job %q / execution %q, explicit job %q / execution %q", agentic.PromptVariant, agentic.Execution.Variant, explicit.PromptVariant, explicit.Execution.Variant)
	}
}

func assertExternalRepositoryJobSettings(t *testing.T, options PlanOptions, byArm map[ArmType]Job) {
	t.Helper()
	baseline, agentic, explicit := byArm[ArmBaseline], byArm[ArmSemeditAgentic], byArm[ArmSemeditExplicit]
	if baseline.SemeditSkill || !agentic.SemeditSkill || !explicit.SemeditSkill {
		t.Errorf("skill conditions = baseline %t, agentic %t, explicit %t", baseline.SemeditSkill, agentic.SemeditSkill, explicit.SemeditSkill)
	}
	if baseline.Policy != "" || agentic.Policy != options.SemeditArmRestriction || explicit.Policy != options.SemeditArmRestriction {
		t.Errorf("job policies = baseline %q, agentic %q, explicit %q", baseline.Policy, agentic.Policy, explicit.Policy)
	}
	if agentic.Execution.Policy != options.SemeditArmRestriction || explicit.Execution.Policy != options.SemeditArmRestriction {
		t.Errorf("execution policies = agentic %q, explicit %q", agentic.Execution.Policy, explicit.Execution.Policy)
	}
	wantMode := MCPServerInstructionsPrescriptive
	if len(options.MCPServerInstructions) > 0 {
		wantMode = options.MCPServerInstructions[0]
	}
	if baseline.MCPServerInstructions != wantMode || agentic.MCPServerInstructions != wantMode || explicit.MCPServerInstructions != wantMode {
		t.Errorf("MCP instruction modes = baseline %q, agentic %q, explicit %q; want %q", baseline.MCPServerInstructions, agentic.MCPServerInstructions, explicit.MCPServerInstructions, wantMode)
	}
	if baseline.PairID != agentic.PairID || baseline.PairID != explicit.PairID {
		t.Errorf("external arm pair IDs differ: %q, %q, %q", baseline.PairID, agentic.PairID, explicit.PairID)
	}
	if baseline.ID == agentic.ID || baseline.ID == explicit.ID || agentic.ID == explicit.ID {
		t.Errorf("external job IDs are not unique: %q, %q, %q", baseline.ID, agentic.ID, explicit.ID)
	}
	legacyReplayID := stablePlanID("job", explicit.TaskID, explicit.Target.String(), string(ArmSemeditExplicit), "small:implicit", fmt.Sprint(explicit.Repeat), string(MCPServerInstructionsNone), "semedit-skill=false", string(options.SemeditArmRestriction))
	if explicit.ID == legacyReplayID {
		t.Error("explicit agent job identity collides with the legacy patch-replay job")
	}
	if baseline.Execution.Prompt == explicit.Execution.Prompt {
		t.Error("explicit arm unexpectedly uses the issue prompt")
	}
}

func assertExternalRepositoryPairing(t *testing.T, options PlanOptions, byArm map[ArmType]Job, jobs []Job) {
	t.Helper()
	baseline, agentic, explicit := byArm[ArmBaseline], byArm[ArmSemeditAgentic], byArm[ArmSemeditExplicit]
	results := make([]*RunResult, 0, len(jobs))
	for _, job := range jobs {
		result := &RunResult{}
		populatePlanResult(&BenchmarkPlan{Options: options}, job, result, nil)
		results = append(results, result)
	}
	comparisons := BuildComparisons(results)
	if len(comparisons) != 2 {
		t.Fatalf("external comparison count = %d, want implicit and explicit cells", len(comparisons))
	}
	for _, comparison := range comparisons {
		if comparison.SmallBaseline == nil || comparison.SmallBaseline.JobID != baseline.ID {
			t.Errorf("prompt variant %q is missing its shared baseline: %+v", comparison.PromptVariant, comparison)
		}
		switch comparison.PromptVariant {
		case "implicit":
			if comparison.SmallSemedit == nil || comparison.SmallSemedit.JobID != agentic.ID {
				t.Errorf("implicit comparison has the wrong agentic result: %+v", comparison)
			}
		case "explicit":
			if comparison.SmallSemeditExplicit == nil || comparison.SmallSemeditExplicit.JobID != explicit.ID {
				t.Errorf("explicit comparison has the wrong explicit agent result: %+v", comparison)
			}
		default:
			t.Errorf("unexpected external comparison prompt variant %q", comparison.PromptVariant)
		}
	}
}

func assertExternalRepositoryArmRotation(t *testing.T, specification Job, options PlanOptions) {
	t.Helper()
	for repeat, wantFirst := range []ArmType{ArmBaseline, ArmSemeditAgentic, ArmSemeditExplicit} {
		repeatedSpec := specification
		repeatedSpec.Repeat = repeat + 1
		jobs, err := makeExternalRepositoryJobs(repeatedSpec, options)
		if err != nil {
			t.Fatal(err)
		}
		if got := jobs[0].Arm; got != wantFirst {
			t.Errorf("repeat %d first arm = %s, want rotated arm %s", repeatedSpec.Repeat, got, wantFirst)
		}
	}
}

func assertExternalRepositoryRequiresRecipe(t *testing.T, task *Task, specification Job, options PlanOptions) {
	t.Helper()
	for _, promptVariants := range []map[string]string{
		{"implicit": "Issue prompt"},
		{"explicit": "  "},
	} {
		missingRecipeTask := *task
		missingRecipeTask.Metadata.PromptVariants = promptVariants
		missingRecipeSpec := specification
		missingRecipeSpec.Task = &missingRecipeTask
		if _, err := makeExternalRepositoryJobs(missingRecipeSpec, options); err == nil {
			t.Error("external explicit planning accepted a missing or blank explicit recipe")
		}
	}
}

func assertExternalRepositorySkillAblation(t *testing.T, specification Job, options PlanOptions, withSkill map[ArmType]Job) {
	t.Helper()
	options.SemeditSkill = false
	jobs, err := makeExternalRepositoryJobs(specification, options)
	if err != nil {
		t.Fatal(err)
	}
	withoutSkill := externalRepositoryJobsByArm(t, jobs)
	for _, arm := range []ArmType{ArmSemeditAgentic, ArmSemeditExplicit} {
		if withoutSkill[arm].ID == withSkill[arm].ID || withoutSkill[arm].PairID == withSkill[arm].PairID {
			t.Errorf("skill ablation did not produce distinct %s job and pair identities", arm)
		}
	}
}

func TestMultiSWEExplicitRecipesArePortableSingleBatchesAndPlanCorrectly(t *testing.T) {
	taskIDs := []string{
		"grpc__grpc-go-2735",
		"grpc__grpc-go-2951",
		"grpc__grpc-go-3361",
		"zeromicro__go-zero-2463",
		"zeromicro__go-zero-964",
	}
	tasks := make(map[string]*Task, len(taskIDs))
	for _, taskID := range taskIDs {
		t.Run(taskID+"/recipe", func(t *testing.T) {
			tasks[taskID] = loadAndCheckMultiSWERecipe(t, taskID)
		})
	}
	plan, err := BuildBenchmarkPlan(PlanOptions{
		BenchDir:              filepath.Join("..", "..", "testdata", "bench"),
		TaskIDs:               taskIDs,
		Targets:               []Target{{Harness: string(HarnessCodex), Model: "fixture"}},
		Variants:              []string{"small"},
		Repeats:               1,
		SemeditArmRestriction: SemeditArmRestrictWrite,
	})
	if err != nil {
		t.Fatal(err)
	}
	jobsByTask := make(map[string][]Job, len(taskIDs))
	for _, job := range plan.Jobs {
		jobsByTask[job.TaskID] = append(jobsByTask[job.TaskID], job)
	}
	for _, taskID := range taskIDs {
		t.Run(taskID+"/plan", func(t *testing.T) {
			assertMultiSWEPlanPrompts(t, taskID, tasks[taskID], jobsByTask[taskID])
		})
	}
}

func TestExternalRepositoryJobsExpandSelectedInstructionModes(t *testing.T) {
	_, specification, options := externalRepositoryPlanningFixture()
	options.MCPServerInstructions = []MCPServerInstructionMode{MCPServerInstructionsNone, MCPServerInstructionsPrescriptive}
	pairByMode := make(map[MCPServerInstructionMode]string)
	seenIDs := make(map[string]bool)
	for _, repeat := range []int{1, 2} {
		assertExternalRepositoryModeRepeat(t, specification, options, repeat, pairByMode, seenIDs)
	}
	if pairByMode[MCPServerInstructionsNone] == pairByMode[MCPServerInstructionsPrescriptive] {
		t.Error("none and prescriptive external mode cells share a pair ID")
	}
}

func extractMultiSWERecipePayload(prompt string) (string, string, bool) {
	if strings.Count(prompt, "```") != 2 || strings.Count(prompt, "```json") != 1 {
		return "", "", false
	}
	const opening = "```json\n"
	openingIndex := strings.Index(prompt, opening)
	if openingIndex < 0 {
		return "", "", false
	}
	payloadStart := openingIndex + len(opening)
	payloadEnd := strings.Index(prompt[payloadStart:], "\n```")
	if payloadEnd < 0 {
		return "", "", false
	}
	closingEnd := payloadStart + payloadEnd + len("\n```")
	payload := strings.TrimSpace(prompt[payloadStart : payloadStart+payloadEnd])
	outside := prompt[:openingIndex] + prompt[closingEnd:]
	return payload, outside, true
}

func loadAndCheckMultiSWERecipe(t *testing.T, taskID string) *Task {
	t.Helper()
	fixturePath := filepath.Join("..", "..", "testdata", "bench", taskID+".txtar")
	fixture, readErr := os.ReadFile(fixturePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	task, parseErr := ParseTask(fixture)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	prompt := task.Metadata.PromptVariants["explicit"]
	if strings.TrimSpace(prompt) == "" {
		t.Fatal("explicit prompt is empty")
	}
	payload, outside, ok := extractMultiSWERecipePayload(prompt)
	if !ok || !strings.Contains(outside, "semantic_batch") {
		t.Fatal("explicit prompt must request one semantic_batch JSON payload")
	}
	for _, tool := range []string{"semantic_lookup", "semantic_inspect_symbol", "refactor_rename", "semantic_replace_body", "semantic_replace_construct", "semantic_insert_construct", "semantic_insert_function", "semantic_insert_decl", "semantic_replace_decl", "semantic_remove_construct", "semantic_move_construct", "semantic_organize_imports"} {
		if strings.Contains(outside, tool) {
			t.Fatalf("explicit prompt requests separate %s outside its batch", tool)
		}
	}
	evidence := readMultiSWERecipeEvidence(t, taskID)
	if taskID == "grpc__grpc-go-2735" {
		evidence = canonicalizeHistoricalRenameRecipe(t, evidence)
	}
	if canonicalJSON(payload) != canonicalJSON(string(evidence)) {
		t.Fatal("explicit prompt JSON payload differs from durable batch_call evidence")
	}
	for _, forbidden := range []string{".benchmark/fix.patch", ".benchmark/test.patch", ".scratch/", "/Users/", "/private/tmp/"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("explicit prompt contains private artifact path %q", forbidden)
		}
	}
	assertPortableMultiSWERecipe(t, payload)
	return task
}

// This archived recipe predates the public rename name migration; keep its captured evidence intact while comparing it with today's fixture.
func canonicalizeHistoricalRenameRecipe(t *testing.T, evidence []byte) []byte {
	t.Helper()
	var recipe map[string]json.RawMessage
	if err := json.Unmarshal(evidence, &recipe); err != nil {
		t.Fatalf("decode historical batch recipe: %v", err)
	}
	var edits []map[string]json.RawMessage
	if err := json.Unmarshal(recipe["edits"], &edits); err != nil {
		t.Fatalf("decode historical batch edits: %v", err)
	}
	for _, edit := range edits {
		var tool string
		if err := json.Unmarshal(edit["tool"], &tool); err != nil {
			t.Fatalf("decode historical batch tool: %v", err)
		}
		if tool == "semantic_rename" {
			canonicalTool, err := json.Marshal("refactor_rename")
			if err != nil {
				t.Fatalf("encode canonical historical rename tool: %v", err)
			}
			edit["tool"] = canonicalTool
		}
	}
	encodedEdits, err := json.Marshal(edits)
	if err != nil {
		t.Fatalf("encode canonical historical batch edits: %v", err)
	}
	recipe["edits"] = encodedEdits
	canonical, err := json.Marshal(recipe)
	if err != nil {
		t.Fatalf("encode canonical historical batch recipe: %v", err)
	}
	return canonical
}

func readMultiSWERecipeEvidence(t *testing.T, taskID string) []byte {
	t.Helper()
	evidenceDir := filepath.Join("..", "..", "data", "benchmarks", "fixture-validation", taskID)
	receiptPath := filepath.Join(evidenceDir, "integrated-validation-receipt.json")
	evidencePath := ""
	receiptData, receiptErr := os.ReadFile(receiptPath)
	if receiptErr == nil {
		var receipt struct {
			Recipe string `json:"recipe"`
		}
		if decodeErr := json.Unmarshal(receiptData, &receipt); decodeErr != nil {
			t.Fatalf("decode validation receipt: %v", decodeErr)
		}
		if receipt.Recipe != "" {
			if !filepath.IsLocal(receipt.Recipe) {
				t.Fatalf("validation receipt recipe path is not local: %q", receipt.Recipe)
			}
			evidencePath = filepath.Join(evidenceDir, filepath.FromSlash(receipt.Recipe))
		}
	} else if !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	if evidencePath == "" {
		for _, name := range []string{"mcp-recipe-single-call.json", "mcp-recipe.json"} {
			candidate := filepath.Join(evidenceDir, name)
			if _, statErr := os.Stat(candidate); statErr == nil {
				evidencePath = candidate
				break
			} else if !os.IsNotExist(statErr) {
				t.Fatal(statErr)
			}
		}
	}
	if evidencePath == "" {
		t.Fatal("durable semantic_batch recipe evidence is missing")
	}
	evidence, readErr := os.ReadFile(evidencePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var wrapper map[string]json.RawMessage
	if decodeErr := json.Unmarshal(evidence, &wrapper); decodeErr != nil {
		t.Fatalf("decode durable recipe JSON: %v", decodeErr)
	}
	if batchCall, wrapped := wrapper["batch_call"]; wrapped {
		var args map[string]json.RawMessage
		if decodeErr := json.Unmarshal(batchCall, &args); decodeErr != nil {
			t.Fatalf("decode durable batch_call: %v", decodeErr)
		}
		if _, hasEdits := args["edits"]; !hasEdits {
			t.Fatal("durable batch_call has no edits")
		}
		return batchCall
	}
	return evidence
}

func canonicalJSON(source string) string {
	var value any
	if err := json.Unmarshal([]byte(source), &value); err != nil {
		return ""
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(canonical)
}

func assertPortableMultiSWERecipe(t *testing.T, payload string) {
	t.Helper()
	var recipe struct {
		Edits []struct {
			Tool   string                     `json:"tool"`
			Params map[string]json.RawMessage `json:"params"`
		} `json:"edits"`
	}
	if err := json.Unmarshal([]byte(payload), &recipe); err != nil {
		t.Fatalf("decode semantic_batch arguments: %v", err)
	}
	if len(recipe.Edits) == 0 {
		t.Fatal("semantic_batch has no ordered edits")
	}
	for index, edit := range recipe.Edits {
		if strings.TrimSpace(edit.Tool) == "" || len(edit.Params) == 0 {
			t.Fatalf("edit %d has no exact tool and parameters", index+1)
		}
		rawFile, ok := edit.Params["file"]
		if !ok {
			t.Fatalf("edit %d has no file parameter", index+1)
		}
		var file string
		if err := json.Unmarshal(rawFile, &file); err != nil || file == "" {
			t.Fatalf("edit %d has invalid file parameter %s: %v", index+1, rawFile, err)
		}
		if path.IsAbs(file) || path.Clean(file) != file || strings.Contains(file, "\\") || strings.Contains(file, ":") || strings.HasPrefix(file, "../") || strings.HasPrefix(file, ".benchmark/") || strings.HasPrefix(file, ".scratch/") {
			t.Fatalf("edit %d has nonportable or private file path %q", index+1, file)
		}
	}
}

func assertMultiSWEPlanPrompts(t *testing.T, taskID string, task *Task, jobs []Job) {
	t.Helper()
	if len(jobs) != 3 {
		t.Fatalf("%s planned jobs = %d, want baseline, implicit, and explicit", taskID, len(jobs))
	}
	byArm := make(map[ArmType]Job, len(jobs))
	for _, job := range jobs {
		byArm[job.Arm] = job
	}
	baseline, hasBaseline := byArm[ArmBaseline]
	implicit, hasImplicit := byArm[ArmSemeditAgentic]
	explicit, hasExplicit := byArm[ArmSemeditExplicit]
	if !hasBaseline || !hasImplicit || !hasExplicit {
		t.Fatalf("%s planned arms = %v", taskID, mapsKeys(byArm))
	}
	if !strings.HasPrefix(baseline.Execution.Prompt, task.Metadata.Instruction) {
		t.Error("baseline prompt must begin with the original issue")
	}
	if task.Metadata.Oracle.MutationPolicy.ForbidTestEdits && !strings.Contains(baseline.Execution.Prompt, "Do not edit tests.") {
		t.Error("baseline prompt omits the candidate test-edit restriction")
	}
	issue := strings.TrimSpace(task.Metadata.Instruction)
	if !strings.HasPrefix(strings.TrimSpace(implicit.Execution.Prompt), issue) {
		t.Error("implicit prompt must begin with the original issue")
	}
	explicitRecipe := strings.TrimSpace(task.Metadata.PromptVariants["explicit"])
	if !strings.HasPrefix(explicit.Execution.Prompt, explicitRecipe) {
		t.Error("explicit prompt must begin with the exact recipe")
	}
	if len(explicit.Execution.Followups) != 0 || len(explicit.Execution.StagedFollowups) != 0 {
		t.Error("explicit execution must not receive corrective followups")
	}
}

func assertExternalRepositoryModeRepeat(t *testing.T, specification Job, options PlanOptions, repeat int, pairByMode map[MCPServerInstructionMode]string, seenIDs map[string]bool) {
	t.Helper()
	repeatedSpec := specification
	repeatedSpec.Repeat = repeat
	jobs, err := makeExternalRepositoryJobs(repeatedSpec, options)
	if err != nil {
		t.Fatal(err)
	}
	wantArms := []ArmType{ArmBaseline, ArmSemeditAgentic, ArmSemeditExplicit}
	wantJobCount := len(options.MCPServerInstructions) * len(wantArms)
	if len(jobs) != wantJobCount {
		t.Fatalf("repeat %d jobs = %d, want %d", repeat, len(jobs), wantJobCount)
	}
	for modeIndex, mode := range options.MCPServerInstructions {
		start := modeIndex * len(wantArms)
		modeJobs := jobs[start : start+len(wantArms)]
		assertExternalRepositoryModeArms(t, modeJobs, mode, repeat, wantArms, pairByMode, seenIDs)
	}
}

func assertExternalRepositoryModeArms(t *testing.T, modeJobs []Job, mode MCPServerInstructionMode, repeat int, wantArms []ArmType, pairByMode map[MCPServerInstructionMode]string, seenIDs map[string]bool) {
	t.Helper()
	wantFirst := wantArms[(repeat-1)%len(wantArms)]
	if got := modeJobs[0].Arm; got != wantFirst {
		t.Errorf("repeat %d mode %q first arm = %q, want rotated arm %q", repeat, mode, got, wantFirst)
	}
	for index, job := range modeJobs {
		if job.MCPServerInstructions != mode {
			t.Errorf("repeat %d job mode = %q, want %q", repeat, job.MCPServerInstructions, mode)
		}
		wantArm := wantArms[(index+repeat-1)%len(wantArms)]
		if job.Arm != wantArm {
			t.Errorf("repeat %d mode %q job %d arm = %q, want %q", repeat, mode, index, job.Arm, wantArm)
		}
		if job.ID == "" || seenIDs[job.ID] {
			t.Errorf("repeat %d mode %q has missing or duplicate job ID %q", repeat, mode, job.ID)
		}
		seenIDs[job.ID] = true
		if index == 0 {
			if previous := pairByMode[mode]; previous != "" && previous == job.PairID {
				t.Errorf("repeat %d mode %q reused pair ID %q", repeat, mode, job.PairID)
			}
			pairByMode[mode] = job.PairID
		} else if job.PairID != modeJobs[0].PairID {
			t.Errorf("repeat %d mode %q jobs have different pair IDs", repeat, mode)
		}
	}
}
