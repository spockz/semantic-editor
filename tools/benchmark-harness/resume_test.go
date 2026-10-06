// resume_test.go verifies run fingerprinting, per-job checkpoint persistence, and safe resume behavior.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResumeCheckpointRestoresJobResultAndSkipsCompletedJob(t *testing.T) {
	root := t.TempDir()
	fixturePath := filepath.Join(root, "task.txtar")
	if err := os.WriteFile(fixturePath, []byte("task fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	job := Job{
		ID:          "job-resume-1",
		PairID:      "pair-1",
		TaskID:      "task-1",
		FixturePath: fixturePath,
		Context:     "small",
		Target:      Target{Harness: "codex", Model: "test"},
		Arm:         ArmSemedit,
		Repeat:      1,
		Execution:   AgentExecution{Prompt: "analysis prompt"},
	}
	plan := &BenchmarkPlan{
		Jobs:    []Job{job, Job{ID: "job-resume-2", TaskID: "task-1", FixturePath: fixturePath}},
		Options: PlanOptions{OutDir: root, RunID: "resume-run", Timeout: time.Minute, Concurrency: 1},
	}
	runDir, err := prepareBenchmarkRunDir(plan)
	if err != nil {
		t.Fatal(err)
	}
	firstEvent := 1250 * time.Millisecond
	turnFirstEvent := 750 * time.Millisecond
	result := &RunResult{
		JobID:                    job.ID,
		TaskID:                   job.TaskID,
		Target:                   job.Target,
		Arm:                      job.Arm,
		Success:                  true,
		WallClock:                2750 * time.Millisecond,
		ProcessStartToFirstEvent: &firstEvent,
		Oracle:                   &OracleResult{Passed: true, Duration: 4250 * time.Millisecond},
		InteractionSteps: []InteractionStep{{
			WallClock:                950 * time.Millisecond,
			ProcessStartToFirstEvent: &turnFirstEvent,
		}},
	}
	path, err := saveJobOutcomeCheckpoint(runDir, JobOutcome{Job: job, Result: result})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{fixturePath, "analysis prompt", "wall_clock_ms", "oracle"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("checkpoint missing %q", want)
		}
	}
	plan.Options.Resume = true
	resumedDir, err := prepareBenchmarkRunDir(plan)
	if err != nil {
		t.Fatal(err)
	}
	checkpoints, err := loadJobCheckpoints(resumedDir, plan)
	if err != nil {
		t.Fatal(err)
	}
	restored := checkpoints[job.ID]
	if restored == nil {
		t.Fatal("completed job checkpoint was not loaded")
	}
	if restored.WallClock != result.WallClock || restored.ProcessStartToFirstEvent == nil || *restored.ProcessStartToFirstEvent != firstEvent || restored.Oracle == nil || restored.Oracle.Duration != result.Oracle.Duration || len(restored.InteractionSteps) != 1 || restored.InteractionSteps[0].WallClock != result.InteractionSteps[0].WallClock || restored.InteractionSteps[0].ProcessStartToFirstEvent == nil || *restored.InteractionSteps[0].ProcessStartToFirstEvent != turnFirstEvent {
		t.Fatalf("duration round-trip failed: got wall=%s oracle=%+v", restored.WallClock, restored.Oracle)
	}
	pending := pendingBenchmarkPlan(plan, checkpoints)
	if len(pending.Jobs) != 1 || pending.Jobs[0].ID != "job-resume-2" {
		t.Fatalf("pending jobs=%v, want only job-resume-2", pending.Jobs)
	}
}

func TestResumeRejectsChangedPlanFingerprint(t *testing.T) {
	root := t.TempDir()
	fixturePath := filepath.Join(root, "task.txtar")
	if err := os.WriteFile(fixturePath, []byte("task fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := &BenchmarkPlan{
		Jobs:    []Job{{ID: "job-resume-1", TaskID: "task-1", FixturePath: fixturePath, Execution: AgentExecution{Prompt: "original prompt"}}},
		Options: PlanOptions{OutDir: root, RunID: "resume-run", Timeout: time.Minute, Concurrency: 1},
	}
	if _, err := prepareBenchmarkRunDir(plan); err != nil {
		t.Fatal(err)
	}
	plan.Options.Resume = true
	plan.Jobs[0].Execution.Prompt = "changed prompt"
	if _, err := prepareBenchmarkRunDir(plan); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("resume error=%v, want plan fingerprint mismatch", err)
	}
}

func TestBenchmarkCLIParsesResumeFlag(t *testing.T) {
	cli, err := parseBenchmarkCLI([]string{"--resume", "--run-id", "resume-run", "--target", "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if !cli.Plan.Resume || cli.Plan.RunID != "resume-run" {
		t.Fatalf("resume options were not retained: %+v", cli.Plan)
	}
}
