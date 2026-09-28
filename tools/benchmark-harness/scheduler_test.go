// Package main tests provider-neutral benchmark scheduling behavior.
package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestExecutePlanReturnsPlanOrderAndCompleteConditionMetadata(t *testing.T) {
	releaseFirst := make(chan struct{})
	writer := &gatedProgressWriter{firstLine: make(chan string, 1)}
	plan := &BenchmarkPlan{Options: PlanOptions{Concurrency: 2, Timeout: time.Second, SemeditArmRestriction: SemeditArmRestrictRead, MCPServerInstructions: []MCPServerInstructionMode{MCPServerInstructionsPrescriptive}, Provenance: ProvenanceSet{"owner": "plan"}}, Jobs: []Job{
		{ID: "first", PairID: "pair", TaskID: "task", MCPServerInstructions: MCPServerInstructionsPrescriptive, Context: "small", PromptVariant: "Default", Target: Target{Harness: "codex"}, Arm: ArmBaseline, Repeat: 1},
		{ID: "second", PairID: "pair", TaskID: "task", MCPServerInstructions: MCPServerInstructionsPrescriptive, Context: "large+verified", PromptVariant: "Default", Target: Target{Harness: "codex"}, Arm: ArmSemedit, Repeat: 2},
	}}
	type resultSet struct {
		outcomes []JobOutcome
		err      error
	}
	done := make(chan resultSet, 1)
	go func() {
		outcomes, err := ExecutePlan(context.Background(), plan, writer, func(ctx context.Context, job Job) (*RunResult, error) {
			if job.ID == "first" {
				<-releaseFirst
			}
			return &RunResult{Success: true, Provenance: ProvenanceSet{"provider": "retained"}}, nil
		})
		done <- resultSet{outcomes: outcomes, err: err}
	}()
	firstLine := <-writer.firstLine
	if !strings.Contains(firstLine, "job=second") || !strings.Contains(firstLine, "[1/2 completed · 1 remaining]") {
		t.Fatalf("progress did not reflect completion order: %q", firstLine)
	}
	close(releaseFirst)
	executed := <-done
	if executed.err != nil {
		t.Fatal(executed.err)
	}
	outcomes := executed.outcomes
	if len(outcomes) != 2 || outcomes[0].Job.ID != "first" || outcomes[1].Job.ID != "second" {
		t.Fatalf("results not in plan order: %+v", outcomes)
	}
	for i, outcome := range outcomes {
		result := outcome.Result
		if result.JobID != outcome.Job.ID || result.ComparisonPairID != "pair" || result.TaskID != "task" || result.PromptVariant != "Default" {
			t.Fatalf("incomplete result identity: %+v", result)
		}
		if result.SemeditArmRestrict != SemeditArmRestrictRead || result.SemeditArmRestrictionApplied != (i == 1) {
			t.Fatalf("policy condition incorrect: %+v", result)
		}
		if result.MCPServerInstructions != MCPServerInstructionsPrescriptive || result.Provenance["owner"] != "plan" || result.Provenance["provider"] != "retained" {
			t.Fatalf("condition metadata lost: %+v", result)
		}
		if !result.Success || outcome.Err != nil {
			t.Fatalf("unexpected failure: %+v", outcome)
		}
	}
	if !strings.Contains(writer.String(), "[2/2 completed · 0 remaining]") {
		t.Fatalf("progress missing: %q", writer.String())
	}
}

func TestExecutePlanRecordsQueuedCancellationAndNilResultFailure(t *testing.T) {
	plan := &BenchmarkPlan{Options: PlanOptions{Concurrency: 1, Timeout: time.Second, SemeditArmRestriction: SemeditArmRestrictWrite}, Jobs: []Job{
		{ID: "a", PairID: "p", TaskID: "t", Context: "small", Target: Target{Harness: "control"}, Arm: ArmControl, Repeat: 1},
		{ID: "b", PairID: "p", TaskID: "t", Context: "large", Target: Target{Harness: "codex"}, Arm: ArmBaseline, Repeat: 1},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcomes, err := ExecutePlan(ctx, plan, io.Discard, func(context.Context, Job) (*RunResult, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("lost queued outcomes: %d", len(outcomes))
	}
	for _, outcome := range outcomes {
		if !errors.Is(outcome.Err, context.Canceled) || outcome.Result.Success || outcome.Result.JobID != outcome.Job.ID || outcome.Result.Error == "" {
			t.Fatalf("cancelled outcome incomplete: %+v", outcome)
		}
	}
	ctx = context.Background()
	outcomes, err = ExecutePlan(ctx, &BenchmarkPlan{Options: plan.Options, Jobs: plan.Jobs[:1]}, io.Discard, func(context.Context, Job) (*RunResult, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Err == nil || outcomes[0].Result.Success || outcomes[0].Result.Error == "" {
		t.Fatalf("nil,nil executor result was not an explicit failure: %+v", outcomes[0])
	}
}

func TestExecutePlanPropagatesProgressWriterFailureAfterDraining(t *testing.T) {
	plan := &BenchmarkPlan{Options: PlanOptions{Concurrency: 2, Timeout: time.Second}, Jobs: []Job{
		{ID: "a", TaskID: "t", Context: "small", Target: Target{Harness: "control"}, Arm: ArmControl},
		{ID: "b", TaskID: "t", Context: "large", Target: Target{Harness: "control"}, Arm: ArmControl},
	}}
	outcomes, err := ExecutePlan(context.Background(), plan, failingProgressWriter{}, func(context.Context, Job) (*RunResult, error) { return &RunResult{Success: true}, nil })
	if err == nil {
		t.Fatal("expected progress writer failure")
	}
	if len(outcomes) != 2 || outcomes[0].Result.JobID == "" || outcomes[1].Result.JobID == "" {
		t.Fatalf("failed to drain outcomes: %+v", outcomes)
	}
}

type failingProgressWriter struct{}

func (failingProgressWriter) Write([]byte) (int, error) {
	return 0, errors.New("progress stream closed")
}

type gatedProgressWriter struct {
	mu        sync.Mutex
	buf       strings.Builder
	firstLine chan string
}

func (w *gatedProgressWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	_, err := w.buf.Write(data)
	w.mu.Unlock()
	select {
	case w.firstLine <- string(data):
	default:
	}
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *gatedProgressWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func TestExecutePlanLogsRunAndJobForResultErrors(t *testing.T) {
	plan := &BenchmarkPlan{
		Options: PlanOptions{Concurrency: 1, Timeout: time.Second, RunID: "benchmark-run-17"},
		Jobs: []Job{{
			ID:     "job-killed",
			TaskID: "task",
			Target: Target{Harness: "codex"},
			Arm:    ArmSemedit,
		}},
	}
	var output strings.Builder
	outcomes, err := ExecutePlan(context.Background(), plan, &output, func(context.Context, Job) (*RunResult, error) {
		return &RunResult{Error: "codex execution: run codex: signal: killed"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].Result.Success {
		t.Fatalf("unexpected outcome: %+v", outcomes)
	}
	for _, want := range []string{
		"FAIL run_id=benchmark-run-17 job=job-killed",
		"error=codex execution: run codex: signal: killed",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("progress log missing %q: %s", want, output.String())
		}
	}
}
