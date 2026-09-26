// This file owns per-run provider selection and the lifecycle of agent sessions so every task turn shares one adapter, resume state, and transcript.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"semedit/internal/pipeline"
	"strings"
	"time"
)

func (s *agentSession) runTurn(ctx context.Context, target Target, prompt, classification string, result *RunResult) error {
	started := time.Now()
	resumeID, runErr := s.adapter.run(ctx, s.workDir, target, prompt, result, s.resumeID)
	if classification == "task" {
		if runErr != nil && result.toolObservationState != ToolObservationPartial {
			result.toolObservationState = ToolObservationPartial
			result.toolObservationReason = runErr.Error()
		}
		if result.toolObservationState == "" {
			result.toolObservationState = ToolObservationUnknown
			result.toolObservationReason = "provider did not confirm complete tool observation capture"
		}
		if s.taskObservationTurns == 0 {
			s.toolObservationState = result.toolObservationState
			if result.toolObservationReason != "" {
				s.toolObservationReasons = append(s.toolObservationReasons, result.toolObservationReason)
			}
		} else if result.toolObservationState != ToolObservationComplete {
			if result.toolObservationState == ToolObservationPartial || s.toolObservationState == ToolObservationComplete {
				s.toolObservationState = result.toolObservationState
			}
			if result.toolObservationReason != "" {
				s.toolObservationReasons = append(s.toolObservationReasons, result.toolObservationReason)
			}
		}
		s.taskObservationTurns++
	}
	refreshMCPVerified(result)
	finished := time.Now()
	if resumeID != "" {
		s.resumeID = resumeID
	}
	s.turns = append(s.turns, sessionTurn{
		Number:         len(s.turns) + 1,
		Classification: classification,
		Prompt:         prompt,
		StartedAt:      started,
		FinishedAt:     finished,
		ResumeID:       resumeID,
		ProviderError:  errorString(runErr),
		ProviderExit:   providerExitDescription(result),
		RawEvents:      append([]json.RawMessage(nil), result.rawEvents...),
		ToolCalls:      append([]ToolCall(nil), result.ToolCalls...),
		Response:       result.agentResponse,
	})
	data, err := json.MarshalIndent(s.turns, "", "  ")
	if err != nil {
		return errors.Join(runErr, fmt.Errorf("encode session transcript: %w", err))
	}
	if err := pipeline.WriteAtomic(s.transcriptPath, data); err != nil {
		return errors.Join(runErr, fmt.Errorf("record session transcript: %w", err))
	}
	return runErr
}

func (s *agentSession) evaluate(ctx context.Context, result *RunResult) error {
	if result == nil {
		return fmt.Errorf("evaluate session result: result is nil")
	}
	return evaluateAgentResult(ctx, s.execution.Task, s.workDir, s.beforeFiles, s.initialPath, s.beforeContent, result)
}

func (s *agentSession) close() error {
	if s.result != nil {
		state := s.toolObservationState
		reason := strings.Join(s.toolObservationReasons, "; ")
		if s.taskObservationTurns == 0 {
			state = ToolObservationUnknown
			reason = "no task turn produced tool observation evidence"
		}
		s.result.DiagnosticToolCoverage = diagnosticToolCoverage(s.execution.Task.Metadata.Oracle.DiagnosticExpectedTools, s.result.ToolCalls, state, reason)
		if s.execution.Arm == ArmSemedit && s.resumeID != "" && !s.result.Success && (!s.result.MCPVerified || hasConsecutiveUnbatchedSemanticToolCalls(s.result.ToolCalls)) {
			if s.result.Error != "" {
				s.result.DiagnosticReflectionSkipped = "task execution or oracle evaluation failed"
			} else {
				s.result.DiagnosticReflectionSkipped = "final task oracle did not pass"
			}
		}
	}
	if s.retainWorkDir {
		if s.result != nil {
			if s.result.Provenance == nil {
				s.result.Provenance = make(ProvenanceSet)
			}
			s.result.Provenance["retained_working_directory"] = s.workDir
		}
		return nil
	}
	if err := os.RemoveAll(s.workDir); err != nil {
		return fmt.Errorf("remove session workspace: %w", err)
	}
	return nil
}

func (s *agentSession) runFollowups(ctx context.Context) (string, bool) {
	result := s.result
	target := s.execution.Target
	arm := s.execution.Arm
	for index, followup := range s.execution.Followups {
		if result.Success {
			break
		}
		turn := &RunResult{Target: target, Arm: arm, Prompt: followup}
		turnStarted := time.Now()
		runErr := s.runTurn(ctx, target, followup, "task", turn)
		if runErr != nil {
			turn.WallClock = time.Since(turnStarted)
			turn.Error = runErr.Error()
			result.Error = fmt.Sprintf("interactive step %d: %v", index+2, runErr)
			result.WallClock = time.Since(s.started)
			s.retainWorkDir = shouldRetainWorkDir(ctx, runErr)
			mergeTurn(result, turn)
			result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
			return s.resumeID, s.retainWorkDir
		}
		turn.WallClock = time.Since(turnStarted)
		if err := s.evaluate(ctx, turn); err != nil {
			turn.Error = err.Error()
			result.Error = err.Error()
			result.WallClock = time.Since(s.started)
			s.retainWorkDir = shouldRetainWorkDir(ctx, err)
			mergeTurn(result, turn)
			result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
			return s.resumeID, s.retainWorkDir
		}
		result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
		mergeTurn(result, turn)
	}
	return s.resumeID, s.retainWorkDir
}

type agentSession struct {
	adapter                providerAdapter
	execution              AgentExecution
	workDir                string
	transcriptPath         string
	resumeID               string
	turns                  []sessionTurn
	started                time.Time
	beforeFiles            workspaceSnapshot
	initialPath            string
	beforeContent          string
	result                 *RunResult
	retainWorkDir          bool
	toolObservationState   ToolObservationState
	taskObservationTurns   int
	toolObservationReasons []string
}

type sessionTurn struct {
	Number         int               `json:"number"`
	Classification string            `json:"classification"`
	Prompt         string            `json:"prompt"`
	StartedAt      time.Time         `json:"started_at"`
	FinishedAt     time.Time         `json:"finished_at"`
	ResumeID       string            `json:"resume_id,omitempty"`
	ProviderError  string            `json:"provider_error,omitempty"`
	ProviderExit   string            `json:"provider_exit,omitempty"`
	RawEvents      []json.RawMessage `json:"raw_events,omitempty"`
	ToolCalls      []ToolCall        `json:"tool_calls,omitempty"`
	Response       string            `json:"response,omitempty"`
}

type openCodeProvider struct{ runner *Runner }

func (p openCodeProvider) run(ctx context.Context, workDir string, target Target, prompt string, result *RunResult, resumeID string) (string, error) {
	return p.runner.runOpenCode(ctx, workDir, target, prompt, result, resumeID)
}

type agyProvider struct {
	runner           *Runner
	calls            int
	rawEvents        int
	toolsUsed        int
	internalTurns    int
	initialLoadTurns int
	mcpLoadTurns     int
}

func (p *agyProvider) run(ctx context.Context, workDir string, target Target, prompt string, result *RunResult, resumeID string) (string, error) {
	conversationID, runErr := p.runner.runAgy(ctx, workDir, target, prompt, result, resumeID)
	callCount := len(result.ToolCalls)
	rawEventCount := len(result.rawEvents)
	normalizationErr := error(nil)
	if callCount < p.calls || rawEventCount < p.rawEvents || len(result.ToolsUsed) < p.toolsUsed {
		normalizationErr = fmt.Errorf("agy transcript cumulative observations regressed")
	} else if result.InternalTurns < p.internalTurns || result.InitialLoadTurns < p.initialLoadTurns || result.MCPLoadTurns < p.mcpLoadTurns {
		normalizationErr = fmt.Errorf("agy transcript turn counters regressed")
	}
	if normalizationErr != nil {
		result.ToolCalls = nil
		result.ToolsUsed = nil
		result.ToolCount = 0
		result.InternalTurns = 0
		result.InitialLoadTurns = 0
		result.MCPLoadTurns = 0
		result.MCPVerified = false
		return conversationID, errors.Join(runErr, normalizationErr)
	}
	result.ToolCalls = append([]ToolCall(nil), result.ToolCalls[p.calls:]...)
	result.rawEvents = append([]json.RawMessage(nil), result.rawEvents[p.rawEvents:]...)
	result.ToolsUsed = append([]string(nil), result.ToolsUsed[p.toolsUsed:]...)
	result.InternalTurns -= p.internalTurns
	result.InitialLoadTurns -= p.initialLoadTurns
	result.MCPLoadTurns -= p.mcpLoadTurns
	p.calls = callCount
	p.rawEvents = rawEventCount
	p.toolsUsed += len(result.ToolsUsed)
	p.internalTurns += result.InternalTurns
	p.initialLoadTurns += result.InitialLoadTurns
	p.mcpLoadTurns += result.MCPLoadTurns
	result.ToolCount = len(result.ToolCalls)
	refreshMCPVerified(result)
	return conversationID, runErr
}

type codexProvider struct{ runner *Runner }

func (p codexProvider) run(ctx context.Context, workDir string, target Target, prompt string, result *RunResult, resumeID string) (string, error) {
	return p.runner.runCodex(ctx, workDir, target, prompt, result, resumeID)
}

type providerAdapter interface {
	run(context.Context, string, Target, string, *RunResult, string) (string, error)
}

func selectProvider(runner *Runner, target Target) (providerAdapter, error) {
	switch target.Harness {
	case string(HarnessCodex):
		return codexProvider{runner: runner}, nil
	case string(HarnessAgy):
		return &agyProvider{runner: runner}, nil
	case string(HarnessOpenCode):
		return openCodeProvider{runner: runner}, nil
	default:
		return nil, fmt.Errorf("unsupported harness: %s", target.Harness)
	}
}

func newAgentSession(adapter providerAdapter, baseDir, runID, workDir string) (*agentSession, error) {
	transcriptDir := filepath.Join(baseDir, "transcripts")
	if err := os.MkdirAll(transcriptDir, 0o750); err != nil {
		return nil, fmt.Errorf("create session transcript directory: %w", err)
	}
	transcriptPath := filepath.Join(transcriptDir, runID+".json")
	return &agentSession{adapter: adapter, workDir: workDir, transcriptPath: transcriptPath, toolObservationState: ToolObservationUnknown}, nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func providerExitDescription(result *RunResult) string {
	if result == nil {
		return ""
	}
	if result.CodexExitCode != nil {
		return fmt.Sprintf("codex exit code %d", *result.CodexExitCode)
	}
	if result.OpenCodeExitCode != nil {
		return fmt.Sprintf("opencode exit code %d", *result.OpenCodeExitCode)
	}
	return ""
}
