// This file defines the provider lifecycle boundary so shared benchmark policy can consume stage outcomes without knowing provider protocols.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"strings"
	"time"
)

type adapterStageKind string

const (
	adapterPreparationStage adapterStageKind = "preparation"
	adapterWarmupStage      adapterStageKind = "warmup"
	adapterTurnStage        adapterStageKind = "turn"
	adapterShutdownStage    adapterStageKind = "shutdown"
)

type adapterSessionConfig struct {
	WorkDir  string
	Target   Target
	Arm      ArmType
	External bool
}

type adapterPreparationInput struct {
	Session adapterSessionConfig
}

type adapterCommandKind string

type adapterInvocation struct {
	Prompt   string
	ResumeID string
	Kind     sessionTurnKind
	Command  adapterCommandKind
}

type adapterStageResult struct {
	Kind             adapterStageKind    `json:"kind"`
	StartedAt        time.Time           `json:"started_at"`
	FinishedAt       time.Time           `json:"finished_at"`
	ProcessStartedAt *time.Time          `json:"process_started_at,omitempty"`
	SessionID        string              `json:"session_id,omitempty"`
	Measurements     adapterMeasurements `json:"measurements"`
	Provenance       ProvenanceSet       `json:"provenance,omitempty"`
	Evidence         []json.RawMessage   `json:"evidence,omitempty"`
	Diagnostics      string              `json:"diagnostics,omitempty"`
	Error            string              `json:"error,omitempty"`
}

type adapterMeasurements struct {
	ProcessStartToFirstEvent         *time.Duration `json:"process_start_to_first_event_ns,omitempty"`
	FirstEventToFirstToolCall        *time.Duration `json:"first_event_to_first_tool_call_ns,omitempty"`
	MCPInitializeToFirstSemanticCall *time.Duration `json:"mcp_initialize_to_first_semantic_call_ns,omitempty"`
	MCPServerStartToInitialize       *time.Duration `json:"mcp_server_start_to_initialize_ns,omitempty"`
}

type adapterTurnObservation struct {
	Turns                 int                  `json:"turns,omitempty"`
	InitialLoadTurns      int                  `json:"initial_load_turns,omitempty"`
	MCPLoadTurns          int                  `json:"mcp_load_turns,omitempty"`
	InternalTurns         int                  `json:"internal_turns,omitempty"`
	PromptTokens          int                  `json:"prompt_tokens,omitempty"`
	CachedPromptTokens    int                  `json:"cached_prompt_tokens,omitempty"`
	UncachedPromptTokens  int                  `json:"uncached_prompt_tokens,omitempty"`
	OutputTokens          int                  `json:"output_tokens,omitempty"`
	ReasoningTokens       int                  `json:"reasoning_tokens,omitempty"`
	InitialContextTokens  int                  `json:"initial_context_tokens,omitempty"`
	ToolCalls             []ToolCall           `json:"tool_calls,omitempty"`
	ToolsUsed             []string             `json:"tools_used,omitempty"`
	ToolObservationState  ToolObservationState `json:"tool_observation_state,omitempty"`
	ToolObservationReason string               `json:"tool_observation_reason,omitempty"`
	Response              string               `json:"response,omitempty"`
	Measurements          adapterMeasurements  `json:"measurements"`
	Provenance            ProvenanceSet        `json:"provenance,omitempty"`
	RawEvents             []json.RawMessage    `json:"raw_events,omitempty"`
	ProcessStartedAt      *time.Time           `json:"process_started_at,omitempty"`
	CodexExitCode         *int                 `json:"codex_exit_code,omitempty"`
	CodexStderr           string               `json:"codex_stderr,omitempty"`
	OpenCodeExitCode      *int                 `json:"opencode_exit_code,omitempty"`
	OpenCodeStderr        string               `json:"opencode_stderr,omitempty"`
}

type adapterTurnResult struct {
	Stage       adapterStageResult     `json:"stage"`
	Observation adapterTurnObservation `json:"observation"`
}

type adapterShutdownResult struct {
	Stage        adapterStageResult `json:"stage"`
	ProcessExit  error              `json:"-"`
	CleanupError error              `json:"-"`
	Stderr       string             `json:"stderr,omitempty"`
}

type providerAdapter interface {
	prepare(context.Context, adapterPreparationInput) (adapterStageResult, error)
	warmup(context.Context) (adapterStageResult, error)
	runTurn(context.Context, adapterInvocation) (adapterTurnResult, error)
	shutdown() adapterShutdownResult
}

func adapterStage(kind adapterStageKind, started time.Time, finished time.Time) adapterStageResult {
	return adapterStageResult{Kind: kind, StartedAt: started, FinishedAt: finished}
}

func adapterNoopStage(kind adapterStageKind) adapterStageResult {
	now := time.Now()
	return adapterStage(kind, now, now)
}

func completeAdapterStage(stage adapterStageResult, kind adapterStageKind, started time.Time, err error) adapterStageResult {
	if stage.Kind == "" {
		stage.Kind = kind
	}
	if stage.StartedAt.IsZero() {
		stage.StartedAt = started
	}
	if stage.FinishedAt.IsZero() {
		stage.FinishedAt = time.Now()
	}
	if err != nil && stage.Error == "" {
		stage.Error = err.Error()
	}
	return stage
}

func adapterShutdownFromError(stage adapterStageResult, err error) adapterShutdownResult {
	out := adapterShutdownResult{Stage: stage}
	var cleanupErrors []error
	for _, component := range errorComponents(err) {
		if _, ok := errors.AsType[*exec.ExitError](component); ok {
			out.ProcessExit = errors.Join(out.ProcessExit, component)
			continue
		}
		cleanupErrors = append(cleanupErrors, component)
	}
	out.CleanupError = errors.Join(cleanupErrors...)
	if err != nil {
		out.Stage.Error = err.Error()
	}
	return out
}

func observationFromRunResult(result *RunResult) adapterTurnObservation {
	return adapterTurnObservation{
		Turns: result.Turns, InitialLoadTurns: result.InitialLoadTurns,
		MCPLoadTurns: result.MCPLoadTurns, InternalTurns: result.InternalTurns,
		PromptTokens: result.PromptTokens, CachedPromptTokens: result.CachedPromptTokens,
		UncachedPromptTokens: result.UncachedPromptTokens, OutputTokens: result.OutputTokens,
		ReasoningTokens: result.ReasoningTokens, InitialContextTokens: result.InitialContextTokens,
		ToolCalls: append([]ToolCall(nil), result.ToolCalls...),
		ToolsUsed: append([]string(nil), result.ToolsUsed...), ToolObservationState: result.toolObservationState,
		ToolObservationReason: result.toolObservationReason, Response: result.agentResponse,
		Measurements: adapterMeasurements{
			ProcessStartToFirstEvent:         cloneDuration(result.ProcessStartToFirstEvent),
			FirstEventToFirstToolCall:        cloneDuration(result.FirstEventToFirstToolCall),
			MCPInitializeToFirstSemanticCall: cloneDuration(result.MCPInitializeToFirstSemanticCall),
			MCPServerStartToInitialize:       cloneDuration(result.MCPServerStartToInitialize),
		},
		Provenance: cloneProvenance(result.Provenance), RawEvents: cloneRawEvents(result.rawEvents),
		ProcessStartedAt: cloneTimePtr(nonzeroTime(result.processStartedAt)),
		CodexExitCode:    cloneInt(result.CodexExitCode), CodexStderr: result.CodexStderr,
		OpenCodeExitCode: cloneInt(result.OpenCodeExitCode), OpenCodeStderr: result.OpenCodeStderr,
	}
}

func applyAdapterObservation(result *RunResult, observation adapterTurnObservation) {
	result.Turns = observation.Turns
	result.InitialLoadTurns = observation.InitialLoadTurns
	result.MCPLoadTurns = observation.MCPLoadTurns
	result.InternalTurns = observation.InternalTurns
	result.PromptTokens = observation.PromptTokens
	result.CachedPromptTokens = observation.CachedPromptTokens
	result.UncachedPromptTokens = observation.UncachedPromptTokens
	result.OutputTokens = observation.OutputTokens
	result.ReasoningTokens = observation.ReasoningTokens
	result.InitialContextTokens = observation.InitialContextTokens
	result.ToolCalls = append([]ToolCall(nil), observation.ToolCalls...)
	result.ToolsUsed = append([]string(nil), observation.ToolsUsed...)
	result.ToolCount = len(result.ToolCalls)
	result.toolObservationState = observation.ToolObservationState
	result.toolObservationReason = observation.ToolObservationReason
	result.agentResponse = observation.Response
	result.ProcessStartToFirstEvent = cloneDuration(observation.Measurements.ProcessStartToFirstEvent)
	result.FirstEventToFirstToolCall = cloneDuration(observation.Measurements.FirstEventToFirstToolCall)
	result.MCPInitializeToFirstSemanticCall = cloneDuration(observation.Measurements.MCPInitializeToFirstSemanticCall)
	result.MCPServerStartToInitialize = cloneDuration(observation.Measurements.MCPServerStartToInitialize)
	if result.Provenance == nil {
		result.Provenance = cloneProvenance(observation.Provenance)
	} else {
		maps.Copy(result.Provenance, observation.Provenance)
	}
	result.rawEvents = cloneRawEvents(observation.RawEvents)
	if observation.ProcessStartedAt != nil {
		result.processStartedAt = *observation.ProcessStartedAt
	}
	result.CodexExitCode = cloneInt(observation.CodexExitCode)
	result.CodexStderr = observation.CodexStderr
	result.OpenCodeExitCode = cloneInt(observation.OpenCodeExitCode)
	result.OpenCodeStderr = observation.OpenCodeStderr
}

func cloneRawEvents(events []json.RawMessage) []json.RawMessage {
	cloned := make([]json.RawMessage, len(events))
	for i := range events {
		cloned[i] = append(json.RawMessage(nil), events[i]...)
	}
	return cloned
}

func nonzeroTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneDuration(value *time.Duration) *time.Duration {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneProvenance(value ProvenanceSet) ProvenanceSet {
	if value == nil {
		return nil
	}
	return maps.Clone(value)
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

const (
	adapterInitialPromptCommand adapterCommandKind = "initial_prompt"
	adapterFollowUpCommand      adapterCommandKind = "follow_up"
)

func newInitialPromptCommand(prompt string, kind sessionTurnKind) adapterInvocation {
	return adapterInvocation{
		Prompt:  prompt,
		Kind:    kind,
		Command: adapterInitialPromptCommand,
	}
}

func newFollowUpCommand(prompt string, kind sessionTurnKind, resumeID string) (adapterInvocation, error) {
	if strings.TrimSpace(resumeID) == "" {
		return adapterInvocation{}, fmt.Errorf("follow-up command has no provider session ID")
	}
	if kind != sessionTaskTurn && kind != sessionDiagnosticTurn {
		return adapterInvocation{}, fmt.Errorf("unsupported follow-up turn kind %q", kind)
	}
	return adapterInvocation{
		Prompt:   prompt,
		ResumeID: resumeID,
		Kind:     kind,
		Command:  adapterFollowUpCommand,
	}, nil
}

func validateAdapterInvocation(input adapterInvocation, firstTurn bool, sessionID string) error {
	expectedCommand := adapterFollowUpCommand
	if firstTurn {
		expectedCommand = adapterInitialPromptCommand
	}
	if input.Command != expectedCommand {
		return fmt.Errorf("adapter invocation command %q does not match first-turn state", input.Command)
	}
	if input.Kind != sessionTaskTurn && input.Kind != sessionDiagnosticTurn {
		return fmt.Errorf("unsupported adapter invocation turn kind %q", input.Kind)
	}
	if input.Command == adapterInitialPromptCommand {
		if input.ResumeID != "" || sessionID != "" {
			return fmt.Errorf("initial prompt command carries a prior provider session ID")
		}
		if input.Kind != sessionTaskTurn {
			return fmt.Errorf("initial prompt command cannot be diagnostic")
		}
		return nil
	}
	if strings.TrimSpace(sessionID) == "" || input.ResumeID != sessionID {
		return fmt.Errorf("follow-up command does not preserve the original provider session ID")
	}
	return nil
}
