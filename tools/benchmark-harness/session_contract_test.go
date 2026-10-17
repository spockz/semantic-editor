// This file proves provider-independent session guarantees using an adapter that deliberately supplies no guards.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testAdapterTurn(ctx context.Context, config adapterSessionConfig, input adapterInvocation, invoke func(context.Context, string, Target, string, *RunResult, string) (string, error)) (adapterTurnResult, error) {
	started := time.Now()
	result := &RunResult{Target: config.Target, Arm: config.Arm}
	sessionID, err := invoke(ctx, config.WorkDir, config.Target, input.Prompt, result, input.ResumeID)
	stage := completeAdapterStage(adapterStage(adapterTurnStage, started, time.Now()), adapterTurnStage, started, err)
	stage.SessionID = sessionID
	return adapterTurnResult{Stage: stage, Observation: observationFromRunResult(result)}, err
}

func testAdapterPrepare(_ context.Context, _ adapterPreparationInput) (adapterStageResult, error) {
	return adapterNoopStage(adapterPreparationStage), nil
}

func testAdapterWarmup(_ context.Context) (adapterStageResult, error) {
	return adapterNoopStage(adapterWarmupStage), nil
}

func testAdapterShutdown(closeFn func() error) adapterShutdownResult {
	started := time.Now()
	var err error
	if closeFn != nil {
		err = closeFn()
	}
	return adapterShutdownFromError(completeAdapterStage(adapterStage(adapterShutdownStage, started, time.Now()), adapterShutdownStage, started, err), err)
}

type sessionContractCall struct {
	Identity sessionIdentity
	Prompt   string
	ResumeID string
}

type sessionContractProvider struct {
	Calls    []sessionContractCall
	Commands []adapterInvocation
	config   adapterSessionConfig
}

func (p *sessionContractProvider) run(_ context.Context, workDir string, target Target, prompt string, result *RunResult, resumeID string) (string, error) {
	p.Calls = append(p.Calls, sessionContractCall{Identity: sessionIdentity{WorkDir: workDir, Target: target, Arm: result.Arm}, Prompt: prompt, ResumeID: resumeID})
	result.PromptTokens = 7
	result.toolObservationState = ToolObservationComplete
	return "original-session", nil
}

func (p *sessionContractProvider) prepare(ctx context.Context, input adapterPreparationInput) (adapterStageResult, error) {
	p.config = input.Session
	return testAdapterPrepare(ctx, input)
}

func (p *sessionContractProvider) warmup(ctx context.Context) (adapterStageResult, error) {
	return testAdapterWarmup(ctx)
}

func (p *sessionContractProvider) runTurn(ctx context.Context, input adapterInvocation) (adapterTurnResult, error) {
	p.Commands = append(p.Commands, input)
	return testAdapterTurn(ctx, p.config, input, p.run)
}

func (p *sessionContractProvider) shutdown() adapterShutdownResult { return testAdapterShutdown(nil) }

func newSessionContractFixture(t *testing.T, arm ArmType) (*agentSession, *sessionContractProvider) {
	t.Helper()
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	provider := &sessionContractProvider{}
	session, err := newAgentSession(provider, root, root, "contract", workDir)
	if err != nil {
		t.Fatal(err)
	}
	session.execution = AgentExecution{Task: &Task{}, Target: Target{Harness: "future-adapter", Model: "pinned-model", Effort: "high"}, Arm: arm}
	t.Cleanup(func() {
		if closeErr := session.close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	return session, provider
}

func TestAdapterLifecycleStagesAreRequiredAndRecorded(t *testing.T) {
	session, _ := newSessionContractFixture(t, ArmBaseline)
	session.result = &RunResult{}
	if err := session.prepareAdapter(t.Context(), session.execution.Target, session.execution.Arm, false); err != nil {
		t.Fatal(err)
	}
	if err := session.warmupAdapter(t.Context()); err != nil {
		t.Fatal(err)
	}
	turn := &RunResult{}
	if err := session.runTurn(t.Context(), "task", sessionTaskTurn, turn); err != nil {
		t.Fatal(err)
	}
	if turn.PromptTokens != 7 || turn.toolObservationState != ToolObservationComplete {
		t.Fatalf("provider observation was not applied by shared lifecycle: %+v", turn)
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(session.stageTranscriptPath)
	if err != nil {
		t.Fatal(err)
	}
	var stages []adapterStageResult
	if err := json.Unmarshal(data, &stages); err != nil {
		t.Fatal(err)
	}
	want := []adapterStageKind{adapterPreparationStage, adapterWarmupStage, adapterTurnStage, adapterShutdownStage}
	if len(stages) != len(want) {
		t.Fatalf("stage transcript has %d stages, want %d: %+v", len(stages), len(want), stages)
	}
	for index, stage := range stages {
		if stage.Kind != want[index] || stage.StartedAt.IsZero() || stage.FinishedAt.IsZero() {
			t.Fatalf("stage %d is incomplete: %+v", index, stage)
		}
	}
}

func TestSessionWrapperSuppliesIdentityForEveryArm(t *testing.T) {
	for _, arm := range []ArmType{ArmBaseline, ArmSemedit, ArmSemeditAgentic, ArmSemeditExplicit} {
		t.Run(string(arm), func(t *testing.T) {
			session, provider := newSessionContractFixture(t, arm)
			expected := sessionIdentity{WorkDir: session.workDir, Target: session.execution.Target, Arm: arm}
			for index, kind := range []sessionTurnKind{sessionTaskTurn, sessionTaskTurn, sessionDiagnosticTurn} {
				result := &RunResult{Target: Target{Harness: "wrong", Model: "wrong"}, Arm: ArmSemedit, Prompt: "wrong"}
				if err := session.runTurn(t.Context(), "prompt", kind, result); err != nil {
					t.Fatal(err)
				}
				call := provider.Calls[index]
				if call.Identity != expected || result.Target != expected.Target || result.Arm != arm || result.Prompt != "prompt" {
					t.Fatalf("turn %d changed identity: call=%+v result=%+v", index+1, call, result)
				}
				expectedResume := "original-session"
				if index == 0 {
					expectedResume = ""
				}
				if call.ResumeID != expectedResume {
					t.Fatalf("turn %d resume=%q want %q", index+1, call.ResumeID, expectedResume)
				}
				command := provider.Commands[index]
				wantCommand := adapterFollowUpCommand
				if index == 0 {
					wantCommand = adapterInitialPromptCommand
				}
				if command.Command != wantCommand || command.Kind != kind || command.ResumeID != expectedResume {
					t.Fatalf("turn %d command = %+v, want command=%q kind=%q resume=%q", index+1, command, wantCommand, kind, expectedResume)
				}
			}
		})
	}
}

func TestSessionWrapperRejectsIdentityMutationBeforeDispatch(t *testing.T) {
	for _, field := range []string{"workspace", "harness", "model", "effort", "arm"} {
		t.Run(field, func(t *testing.T) {
			session, provider := newSessionContractFixture(t, ArmSemeditExplicit)
			if err := session.runTurn(t.Context(), "initial", sessionTaskTurn, &RunResult{}); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "workspace":
				session.workDir = filepath.Join(session.workDir, "other")
			case "harness":
				session.execution.Target.Harness = "other"
			case "model":
				session.execution.Target.Model = "other"
			case "effort":
				session.execution.Target.Effort = "low"
			case "arm":
				session.execution.Arm = ArmSemedit
			}
			runErr := session.runTurn(t.Context(), "followup", sessionDiagnosticTurn, &RunResult{})
			if runErr == nil || !strings.Contains(runErr.Error(), "workspace, target, or arm changed") || len(provider.Calls) != 1 {
				t.Fatalf("mutation reached provider: error=%v calls=%+v", runErr, provider.Calls)
			}
			if retryErr := session.runTurn(t.Context(), "retry", sessionTaskTurn, &RunResult{}); retryErr == nil || len(provider.Calls) != 1 {
				t.Fatalf("invalid session dispatched again: error=%v calls=%+v", retryErr, provider.Calls)
			}
		})
	}
}

func TestSessionWrapperRejectsWorkAfterDiagnostics(t *testing.T) {
	session, provider := newSessionContractFixture(t, ArmSemeditExplicit)
	for _, kind := range []sessionTurnKind{sessionTaskTurn, sessionDiagnosticTurn} {
		if err := session.runTurn(t.Context(), "prompt", kind, &RunResult{}); err != nil {
			t.Fatal(err)
		}
	}
	runErr := session.runTurn(t.Context(), "late work", sessionTaskTurn, &RunResult{})
	if runErr == nil || !strings.Contains(runErr.Error(), "after diagnostics started") || len(provider.Calls) != 2 {
		t.Fatalf("late task dispatched: error=%v calls=%+v", runErr, provider.Calls)
	}
}

func TestSessionWrapperRejectsAggregateReuseWithoutMutation(t *testing.T) {
	session, provider := newSessionContractFixture(t, ArmSemeditExplicit)
	session.result = &RunResult{}
	if err := session.runTurn(t.Context(), "initial", sessionTaskTurn, session.result); err != nil {
		t.Fatal(err)
	}
	frozen, err := json.Marshal(session.result)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []sessionTurnKind{sessionTaskTurn, sessionDiagnosticTurn} {
		if runErr := session.runTurn(t.Context(), "followup", kind, session.result); runErr == nil || !strings.Contains(runErr.Error(), "separate turn result") {
			t.Fatalf("aggregate reuse error=%v", runErr)
		}
	}
	after, err := json.Marshal(session.result)
	if err != nil {
		t.Fatal(err)
	}
	if string(frozen) != string(after) || len(provider.Calls) != 1 {
		t.Fatal("aggregate reuse changed work evidence or dispatched a turn")
	}
}

func TestSessionWrapperRejectsInvalidTurnKind(t *testing.T) {
	session, provider := newSessionContractFixture(t, ArmBaseline)
	if err := session.runTurn(t.Context(), "prompt", "diagnostic_typo", &RunResult{}); err == nil || !strings.Contains(err.Error(), "unsupported session turn kind") || len(provider.Calls) != 0 {
		t.Fatalf("invalid turn kind reached provider: error=%v calls=%+v", err, provider.Calls)
	}
}

func TestAdapterInvocationCommandsSeparateInitialAndFollowUp(t *testing.T) {
	initial := newInitialPromptCommand("rename the symbol", sessionTaskTurn)
	if initial.Command != adapterInitialPromptCommand || initial.Kind != sessionTaskTurn || initial.ResumeID != "" {
		t.Fatalf("initial command = %+v", initial)
	}
	if err := validateAdapterInvocation(initial, true, ""); err != nil {
		t.Fatalf("validate initial command: %v", err)
	}
	if err := validateAdapterInvocation(initial, true, "unexpected-session"); err == nil {
		t.Fatal("initial command accepted a preexisting session ID")
	}

	invalidInitial := newInitialPromptCommand("diagnose first", sessionDiagnosticTurn)
	if err := validateAdapterInvocation(invalidInitial, true, ""); err == nil {
		t.Fatal("initial diagnostic command succeeded")
	}

	followUp, err := newFollowUpCommand("fix the missed call site", sessionTaskTurn, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if followUp.Command != adapterFollowUpCommand || followUp.ResumeID != "session-1" {
		t.Fatalf("follow-up command = %+v", followUp)
	}
	err = validateAdapterInvocation(followUp, false, "session-1")
	if err != nil {
		t.Fatalf("validate follow-up command: %v", err)
	}

	diagnostic, err := newFollowUpCommand("explain the remaining diagnostics", sessionDiagnosticTurn, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	err = validateAdapterInvocation(diagnostic, false, "session-1")
	if err != nil {
		t.Fatalf("validate diagnostic follow-up command: %v", err)
	}

	if _, commandErr := newFollowUpCommand("follow up", sessionTaskTurn, " "); commandErr == nil {
		t.Fatal("follow-up command with a blank session ID succeeded")
	}
	if err := validateAdapterInvocation(followUp, true, ""); err == nil {
		t.Fatal("follow-up command was accepted as the initial command")
	}
	followUp.ResumeID = "replacement-session"
	if err := validateAdapterInvocation(followUp, false, "session-1"); err == nil {
		t.Fatal("follow-up command with a changed session ID succeeded")
	}
}
