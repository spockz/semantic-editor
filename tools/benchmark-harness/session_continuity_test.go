// This file proves that session identity and provider shutdown ordering preserve benchmark evidence across follow-ups.

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/pipeline"
)

func TestAgentSessionRejectsChangedOriginalThread(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	session, sessionErr := newAgentSession(&continuitySessionProvider{}, root, root, "identity", workDir)
	if sessionErr != nil {
		t.Fatal(sessionErr)
	}
	session.execution = AgentExecution{Task: &Task{}}
	session.result = &RunResult{}
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Error(err)
		}
	})
	if err := session.runTurn(t.Context(), "initial", sessionTaskTurn, &RunResult{}); err != nil {
		t.Fatal(err)
	}
	turn := &RunResult{}
	if err := session.runTurn(t.Context(), "hint", sessionTaskTurn, turn); err == nil || !strings.Contains(err.Error(), "changed the original session ID") {
		t.Fatalf("continuation error = %v, want thread identity failure", err)
	}
	if session.resumeID != "original-thread" || turn.toolObservationState != ToolObservationPartial {
		t.Fatalf("original thread/capture = %q/%s", session.resumeID, turn.toolObservationState)
	}
	data, readErr := os.ReadFile(session.transcriptPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var transcript []sessionTurn
	if err := json.Unmarshal(data, &transcript); err != nil {
		t.Fatal(err)
	}
	if len(transcript) != 2 || transcript[1].ProviderError == "" || transcript[1].ResumeID != "replacement-thread" {
		t.Fatalf("identity failure omitted from transcript: %+v", transcript)
	}
}

func TestAgentSessionClosesProviderBeforeRetainingMCPMetrics(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.MkdirAll(filepath.Join(workDir, ".scratch"), 0o750); err != nil {
		t.Fatal(err)
	}
	finished := time.Now()
	closed := false
	provider := &continuitySessionProvider{closeHook: func() error {
		if _, err := os.Stat(workDir); err != nil {
			return err
		}
		records := []map[string]any{{"sequence": 1, "tool": "semantic_lookup", "completed_at": finished.Add(-time.Millisecond), "metrics": map[string]any{"total_ms": 10}}}
		data, marshalErr := json.Marshal(records)
		if marshalErr != nil {
			return marshalErr
		}
		if err := pipeline.WriteAtomic(benchmarkMCPMetricsPath(workDir), data); err != nil {
			return err
		}
		closed = true
		return nil
	}}
	session, sessionErr := newAgentSession(provider, root, root, "close-order", workDir)
	if sessionErr != nil {
		t.Fatal(sessionErr)
	}
	session.execution = AgentExecution{Task: &Task{}, Arm: ArmSemedit}
	session.result = &RunResult{}
	session.turns = []sessionTurn{{Classification: "task", FinishedAt: finished}}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("provider was not closed")
	}
	path := session.result.Provenance["mcp_metrics_file"]
	if path == "" {
		t.Fatal("provider shutdown metrics were not retained")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(data), "semantic_lookup") {
		t.Fatalf("metrics = %s", data)
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("workspace not removed: %v", err)
	}
}

func TestCodingHarnessesRequireOriginalSessionAcrossAllTurns(t *testing.T) {
	for _, harness := range []string{string(HarnessCodex), string(HarnessAgy), string(HarnessOpenCode)} {
		for _, testCase := range sessionIdentityCases() {
			t.Run(harness+"/"+testCase.name, func(t *testing.T) {
				runSessionIdentityCase(t, harness, testCase)
			})
		}
	}
}

func TestCodingHarnessTransportProvenance(t *testing.T) {
	for harness, transport := range map[string]string{
		string(HarnessCodex):    "codex_app_server_v2",
		string(HarnessAgy):      "agy_stream_json_v1",
		string(HarnessOpenCode): "opencode_attached_server_v1",
	} {
		t.Run(harness, func(t *testing.T) {
			result := &RunResult{}
			execution := AgentExecution{Prompt: "task", Target: Target{Harness: harness}}
			_, err := prepareAgentRunMetadata(execution, &agentSession{transcriptPath: "transcript.json"}, result, false)
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Provenance["provider_transport"]; got != transport {
				t.Fatalf("provider transport = %q, want %q", got, transport)
			}
		})
	}
}

type sessionIdentityCase struct {
	name         string
	ids          []string
	failAt       int
	wantOriginal string
}

func (p *sessionIdentityProvider) run(_ context.Context, _ string, _ Target, _ string, result *RunResult, resumeID string) (string, error) {
	index := len(p.resumes)
	p.resumes = append(p.resumes, resumeID)
	result.toolObservationState = ToolObservationComplete
	return p.ids[index], nil
}

func (p *sessionIdentityProvider) prepare(ctx context.Context, input adapterPreparationInput) (adapterStageResult, error) {
	p.config = input.Session
	return testAdapterPrepare(ctx, input)
}

func (p *sessionIdentityProvider) warmup(ctx context.Context) (adapterStageResult, error) {
	return testAdapterWarmup(ctx)
}

func (p *sessionIdentityProvider) runTurn(ctx context.Context, input adapterInvocation) (adapterTurnResult, error) {
	return testAdapterTurn(ctx, p.config, input, p.run)
}

func (p *sessionIdentityProvider) shutdown() adapterShutdownResult { return testAdapterShutdown(nil) }

type sessionIdentityProvider struct {
	ids     []string
	resumes []string
	config  adapterSessionConfig
}

func (p *continuitySessionProvider) run(_ context.Context, _ string, _ Target, _ string, result *RunResult, _ string) (string, error) {
	p.calls++
	result.toolObservationState = ToolObservationComplete
	if p.calls == 1 {
		return "original-thread", nil
	}
	return "replacement-thread", nil
}

func (p *continuitySessionProvider) close() error {
	if p.closeHook != nil {
		return p.closeHook()
	}
	return nil
}

func (p *continuitySessionProvider) prepare(ctx context.Context, input adapterPreparationInput) (adapterStageResult, error) {
	p.config = input.Session
	return testAdapterPrepare(ctx, input)
}

func (p *continuitySessionProvider) warmup(ctx context.Context) (adapterStageResult, error) {
	return testAdapterWarmup(ctx)
}

func (p *continuitySessionProvider) runTurn(ctx context.Context, input adapterInvocation) (adapterTurnResult, error) {
	return testAdapterTurn(ctx, p.config, input, p.run)
}

func (p *continuitySessionProvider) shutdown() adapterShutdownResult {
	return testAdapterShutdown(p.close)
}

type continuitySessionProvider struct {
	calls     int
	closeHook func() error
	config    adapterSessionConfig
}

func sessionIdentityCases() []sessionIdentityCase {
	return []sessionIdentityCase{
		{name: "initial_missing", ids: []string{""}, failAt: 0},
		{name: "initial_blank", ids: []string{" "}, failAt: 0},
		{name: "hint_missing", ids: []string{"original", ""}, failAt: 1, wantOriginal: "original"},
		{name: "hint_changed", ids: []string{"original", "replacement"}, failAt: 1, wantOriginal: "original"},
		{name: "reflection_missing", ids: []string{"original", "original", ""}, failAt: 2, wantOriginal: "original"},
		{name: "reflection_changed", ids: []string{"original", "original", "replacement"}, failAt: 2, wantOriginal: "original"},
		{name: "same_session", ids: []string{"original", "original", "original"}, failAt: -1, wantOriginal: "original"},
	}
}

func runSessionIdentityCase(t *testing.T, harness string, testCase sessionIdentityCase) {
	t.Helper()
	root := t.TempDir()
	workDir := filepath.Join(root, "workspace")
	if err := os.Mkdir(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	provider := &sessionIdentityProvider{ids: testCase.ids}
	session, err := newAgentSession(provider, root, root, "identity", workDir)
	if err != nil {
		t.Fatal(err)
	}
	target := Target{Harness: harness}
	session.execution = AgentExecution{Task: &Task{}, Target: target}
	session.result = &RunResult{}
	t.Cleanup(func() {
		if closeErr := session.close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	assertSessionIdentityTurns(t, session, provider, target, testCase)
	if session.resumeID != testCase.wantOriginal || provider.resumes[0] != "" {
		t.Fatalf("original session=%q resumes=%v", session.resumeID, provider.resumes)
	}
	for _, id := range provider.resumes[1:] {
		if id != "original" {
			t.Fatalf("follow-up dispatched to %q", id)
		}
	}
	assertSessionIdentityTranscript(t, session.transcriptPath, testCase.failAt >= 0)
}

func assertSessionIdentityTurns(t *testing.T, session *agentSession, provider *sessionIdentityProvider, target Target, testCase sessionIdentityCase) {
	t.Helper()
	for index, classification := range []string{"task", "task", "diagnostic_reflection"} {
		turn := &RunResult{Target: target}
		runErr := session.runTurn(t.Context(), classification, sessionTurnKind(classification), turn)
		if index == testCase.failAt {
			if runErr == nil || turn.toolObservationState != ToolObservationPartial {
				t.Fatalf("identity failure was not reported: error=%v capture=%s", runErr, turn.toolObservationState)
			}
			calls := len(provider.resumes)
			if retryErr := session.runTurn(t.Context(), "retry", sessionTaskTurn, &RunResult{}); retryErr == nil || len(provider.resumes) != calls {
				t.Fatalf("failed session was dispatched again: error=%v calls=%d", retryErr, len(provider.resumes))
			}
			return
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
	}
}

func assertSessionIdentityTranscript(t *testing.T, path string, wantFailure bool) {
	t.Helper()
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var transcript []sessionTurn
	if decodeErr := json.Unmarshal(data, &transcript); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if wantFailure && transcript[len(transcript)-1].ProviderError == "" {
		t.Fatal("session failure was omitted from transcript")
	}
}
