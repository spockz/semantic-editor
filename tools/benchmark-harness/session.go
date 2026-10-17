// This file owns per-run provider selection and the lifecycle of agent sessions so every task turn shares one adapter, resume state, and transcript.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const acceptanceOracleTimeout = 5 * time.Minute

func acceptanceOracleContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, acceptanceOracleTimeout)
}

func (s *agentSession) recordExecutionError(err error) {
	s.executionErr = joinDistinctErrors(s.executionErr, err)
}

func (s *agentSession) runTurn(ctx context.Context, target Target, prompt, classification string, result *RunResult) error {
	started := time.Now()
	var returnedID string
	var runErr error
	switch {
	case s.continuationErr != nil && (classification != "diagnostic_reflection" || strings.TrimSpace(s.resumeID) == ""):
		runErr = fmt.Errorf("cannot continue failed provider session: %w", s.continuationErr)
	case len(s.turns) > 0 && strings.TrimSpace(s.resumeID) == "":
		runErr = fmt.Errorf("cannot continue provider session: original session ID is missing")
	default:
		returnedID, runErr = s.adapter.run(ctx, s.workDir, target, prompt, result, s.resumeID)
		identityErr := providerSessionIdentityError(s.resumeID, returnedID, runErr)
		runErr = errors.Join(runErr, identityErr)
		if identityErr == nil && returnedID != "" && s.resumeID == "" {
			s.resumeID = returnedID
		}
	}
	if runErr != nil && s.continuationErr == nil {
		s.continuationErr = runErr
	}
	started = result.wallClockStart(started)
	if runErr != nil && result.toolObservationState != ToolObservationPartial {
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = runErr.Error()
	}
	if classification == "task" {
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
	s.turns = append(s.turns, sessionTurn{
		Number: len(s.turns) + 1, Classification: classification, Prompt: prompt,
		StartedAt: started, FinishedAt: finished, ResumeID: returnedID,
		ProviderError: errorString(runErr), ProviderExit: providerExitDescription(result),
		RawEvents: append([]json.RawMessage(nil), result.rawEvents...),
		ToolCalls: append([]ToolCall(nil), result.ToolCalls...), Response: result.agentResponse,
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
		s.oracleExecutionErr = fmt.Errorf("evaluate session result: result is nil")
		s.recordExecutionError(s.oracleExecutionErr)
		return s.oracleExecutionErr
	}
	oracleCtx, cancel := acceptanceOracleContext(ctx)
	defer cancel()
	s.oracleExecutionErr = evaluateAgentResult(oracleCtx, s.execution.Task, s.workDir, s.beforeFiles, s.initialPath, s.beforeContent, result)
	s.recordExecutionError(s.oracleExecutionErr)
	return s.oracleExecutionErr
}

func (s *agentSession) evaluateStage(ctx context.Context, result *RunResult, oracle OracleConfig) error {
	if result == nil {
		s.oracleExecutionErr = fmt.Errorf("evaluate session stage: result is nil")
		s.recordExecutionError(s.oracleExecutionErr)
		return s.oracleExecutionErr
	}
	task := *s.execution.Task
	task.Metadata.Oracle = oracle
	oracleCtx, cancel := acceptanceOracleContext(ctx)
	defer cancel()
	s.oracleExecutionErr = evaluateAgentResult(oracleCtx, &task, s.workDir, s.beforeFiles, s.initialPath, s.beforeContent, result)
	s.recordExecutionError(s.oracleExecutionErr)
	return s.oracleExecutionErr
}

func (s *agentSession) retainTaskMCPMetrics() error {
	if s.result == nil || !isSemeditArm(s.execution.Arm) {
		return nil
	}
	source := benchmarkMCPMetricsPath(s.workDir)
	// #nosec G304 -- source is derived from this benchmark's isolated workspace.
	data, readErr := os.ReadFile(source)
	if errors.Is(readErr, os.ErrNotExist) {
		return nil
	}
	if readErr != nil {
		return fmt.Errorf("read benchmark MCP metrics: %w", readErr)
	}
	var records []json.RawMessage
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("decode benchmark MCP metrics: %w", err)
	}
	var taskFinishedAt time.Time
	for _, turn := range s.turns {
		if turn.Classification == "task" {
			taskFinishedAt = turn.FinishedAt
		}
	}
	if taskFinishedAt.IsZero() {
		taskFinishedAt = time.Now()
	}
	taskRecords := make([]json.RawMessage, 0, len(records))
	for _, record := range records {
		var timing struct {
			CompletedAt time.Time `json:"completed_at"`
		}
		if err := json.Unmarshal(record, &timing); err != nil {
			return fmt.Errorf("decode benchmark MCP timing record: %w", err)
		}
		if timing.CompletedAt.IsZero() {
			return fmt.Errorf("benchmark MCP timing record has no completion time")
		}
		if !timing.CompletedAt.After(taskFinishedAt) {
			taskRecords = append(taskRecords, record)
		}
	}
	if len(taskRecords) == 0 {
		return nil
	}
	output, err := json.MarshalIndent(taskRecords, "", "  ")
	if err != nil {
		return fmt.Errorf("encode task MCP metrics: %w", err)
	}
	destination := strings.TrimSuffix(s.transcriptPath, ".json") + ".mcp-metrics.json"
	if err := pipeline.WriteAtomic(destination, output); err != nil {
		return fmt.Errorf("retain task MCP metrics: %w", err)
	}
	if s.result.Provenance == nil {
		s.result.Provenance = make(ProvenanceSet)
	}
	s.result.Provenance["mcp_metrics_file"] = destination
	return nil
}

func (s *agentSession) close() (retErr error) {
	defer func() { retErr = errors.Join(retErr, s.closeRoots()) }()
	if closer, ok := s.adapter.(interface{ close() error }); ok {
		if err := closer.close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close provider session: %w", err))
			s.retainWorkDir = true
		}
	}
	if err := s.retainTaskMCPMetrics(); err != nil {
		retErr = errors.Join(retErr, err)
		s.retainWorkDir = true
	}
	s.finalizeResult()
	if s.retainWorkDir {
		s.recordRetainedWorkDir()
		return retErr
	}
	return errors.Join(retErr, s.removeWorkspace())
}

func (s *agentSession) closeRoots() error {
	var closeErr error
	if s.workspaceRoot != nil {
		if err := s.workspaceRoot.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close session workspace root: %w", err))
		}
		s.workspaceRoot = nil
	}
	if s.scratchRoot != nil {
		if err := s.scratchRoot.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close benchmark scratch root: %w", err))
		}
		s.scratchRoot = nil
	}
	return closeErr
}

func (s *agentSession) finalizeResult() {
	if s.result == nil {
		return
	}
	cost := estimateCloudCost(
		s.result.Target,
		s.result.PromptTokens,
		s.result.CachedPromptTokens,
		s.result.UncachedPromptTokens,
		s.result.OutputTokens,
		s.result.ReasoningTokens,
	)
	if cost != nil {
		s.result.EstimatedCostUSD = &cost.TotalUSD
		s.result.EstimatedCostPricingID = cost.PricingID
		s.result.EstimatedCostRateCardAsOf = cost.RateCardAsOf
		s.result.EstimatedCostBasis = cost.Basis
	}
	state := s.toolObservationState
	reason := strings.Join(s.toolObservationReasons, "; ")
	if s.taskObservationTurns == 0 {
		state = ToolObservationUnknown
		reason = "no task turn produced tool observation evidence"
	}
	s.result.DiagnosticToolCoverage = diagnosticToolCoverage(s.execution.Task.Metadata.Oracle.DiagnosticExpectedTools, s.result.ToolCalls, state, reason)
	shouldSkipReflection := isSemeditArm(s.execution.Arm) && s.resumeID != "" && !s.result.Success && s.result.DiagnosticTimeoutReflection == nil
	if shouldSkipReflection && s.result.DiagnosticReflectionSkipped == "" {
		s.result.DiagnosticReflectionSkipped = diagnosticReflectionSkipReason(s.result.Error)
	}
}

func diagnosticReflectionSkipReason(resultError string) string {
	if resultError != "" {
		return "task execution or oracle evaluation failed"
	}
	return "final task oracle did not pass"
}

func (s *agentSession) recordRetainedWorkDir() {
	if s.result == nil {
		return
	}
	if s.result.Provenance == nil {
		s.result.Provenance = make(ProvenanceSet)
	}
	s.result.Provenance["retained_working_directory"] = s.workDir
}

func (s *agentSession) removeWorkspace() error {
	if err := makeSessionWorkspaceRemovable(s.workspaceRoot); err != nil {
		return fmt.Errorf("make session workspace removable: %w", err)
	}
	entries, err := fs.ReadDir(s.workspaceRoot.FS(), ".")
	if err != nil {
		return fmt.Errorf("read session workspace for cleanup: %w", err)
	}
	for _, entry := range entries {
		if err := s.workspaceRoot.RemoveAll(entry.Name()); err != nil {
			return fmt.Errorf("remove captured session workspace entry %q: %w", entry.Name(), err)
		}
	}
	if err := s.workspaceRoot.Close(); err != nil {
		s.workspaceRoot = nil
		return fmt.Errorf("close session workspace root before removal: %w", err)
	}
	s.workspaceRoot = nil
	if err := s.scratchRoot.RemoveAll(s.workspaceRelativePath); err != nil {
		return fmt.Errorf("remove session workspace: %w", err)
	}
	return nil
}

func (s *agentSession) runFollowups(taskCtx, oracleCtx context.Context) (string, bool) {
	result := s.result
	target := s.execution.Target
	arm := s.execution.Arm
	for index, followup := range s.execution.Followups {
		if result.Success {
			break
		}
		turn := &RunResult{Target: target, Arm: arm, Prompt: followup}
		turnStarted := time.Now()
		runErr := s.runTurn(taskCtx, target, followup, "task", turn)
		if runErr != nil {
			turn.WallClock = turn.measuredWallClock(turnStarted)
			turn.Error = runErr.Error()
			primaryErr := fmt.Errorf("interactive step %d: %w", index+2, runErr)
			s.recordExecutionError(primaryErr)
			result.Error = appendResultError(result.Error, primaryErr)
			result.WallClock = elapsedTaskTime(s.started)
			s.retainWorkDir = shouldRetainWorkDir(taskCtx, runErr)
			mergeTurn(result, turn)
			result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
			return s.resumeID, s.retainWorkDir
		}
		if err := s.evaluate(oracleCtx, turn); err != nil {
			turn.WallClock = turn.measuredWallClock(turnStarted)
			turn.Error = err.Error()
			result.Error = err.Error()
			result.WallClock = elapsedTaskTime(s.started)
			s.retainWorkDir = shouldRetainWorkDir(oracleCtx, err)
			mergeTurn(result, turn)
			result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
			return s.resumeID, s.retainWorkDir
		}
		turn.WallClock = turn.measuredWallClock(turnStarted)
		result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
		mergeTurn(result, turn)
	}
	return s.resumeID, s.retainWorkDir
}

func (s *agentSession) runStagedFollowups(taskCtx, oracleCtx context.Context) (string, bool) {
	result := s.result
	target := s.execution.Target
	arm := s.execution.Arm
	allPassed := result.Success
	for index, followup := range s.execution.StagedFollowups {
		turn := &RunResult{Target: target, Arm: arm, Prompt: followup.Instruction}
		turnStarted := time.Now()
		runErr := s.runTurn(taskCtx, target, followup.Instruction, "task", turn)
		if runErr != nil {
			turn.WallClock = turn.measuredWallClock(turnStarted)
			turn.Error = runErr.Error()
			primaryErr := fmt.Errorf("staged step %d: %w", index+2, runErr)
			s.recordExecutionError(primaryErr)
			result.Error = appendResultError(result.Error, primaryErr)
			s.retainWorkDir = shouldRetainWorkDir(taskCtx, runErr)
			mergeTurn(result, turn)
			result.Success = false
			result.InteractionSteps = append(result.InteractionSteps, stagedInteractionStep(index+2, followup.TotalEdits, followup.Instruction, turn))
			result.WallClock = elapsedTaskTime(s.started)
			return s.resumeID, s.retainWorkDir
		}
		if err := s.evaluateStage(oracleCtx, turn, followup.Oracle); err != nil {
			turn.WallClock = turn.measuredWallClock(turnStarted)
			turn.Error = err.Error()
			result.Error = err.Error()
			s.retainWorkDir = shouldRetainWorkDir(oracleCtx, err)
			mergeTurn(result, turn)
			result.Success = false
			result.InteractionSteps = append(result.InteractionSteps, stagedInteractionStep(index+2, followup.TotalEdits, followup.Instruction, turn))
			result.WallClock = elapsedTaskTime(s.started)
			return s.resumeID, s.retainWorkDir
		}
		turn.WallClock = turn.measuredWallClock(turnStarted)
		allPassed = allPassed && turn.Success
		result.InteractionSteps = append(result.InteractionSteps, stagedInteractionStep(index+2, followup.TotalEdits, followup.Instruction, turn))
		mergeTurn(result, turn)
		result.Success = allPassed
	}
	result.WallClock = elapsedTaskTime(s.started)
	return s.resumeID, s.retainWorkDir
}

func (s *agentSession) prepareFixture(ctx context.Context, target Target, arm ArmType, external bool) error {
	if preparer, ok := s.adapter.(interface {
		prepareFixture(context.Context, string, Target, ArmType, bool) error
	}); ok {
		if err := preparer.prepareFixture(ctx, s.workDir, target, arm, external); err != nil {
			return fmt.Errorf("prepare provider fixture: %w", err)
		}
	}
	return nil
}

type agentSession struct {
	adapter                providerAdapter
	execution              AgentExecution
	workDir                string
	scratchRoot            *os.Root
	workspaceRoot          *os.Root
	workspaceRelativePath  string
	transcriptPath         string
	resumeID               string
	continuationErr        error
	turns                  []sessionTurn
	started                time.Time
	beforeFiles            workspaceSnapshot
	initialPath            string
	beforeContent          string
	result                 *RunResult
	oracleExecutionErr     error
	retainWorkDir          bool
	toolObservationState   ToolObservationState
	taskObservationTurns   int
	toolObservationReasons []string
	executionErr           error
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

type providerAdapter interface {
	run(context.Context, string, Target, string, *RunResult, string) (string, error)
}

func selectProvider(runner *Runner, target Target) (providerAdapter, error) {
	switch target.Harness {
	case string(HarnessCodex):
		return &codexProvider{runner: runner}, nil
	case string(HarnessAgy):
		return &agyProvider{runner: runner}, nil
	case string(HarnessOpenCode):
		return &openCodeProvider{runner: runner}, nil
	default:
		return nil, fmt.Errorf("unsupported harness: %s", target.Harness)
	}
}

func benchmarkMCPMetricsPath(workDir string) string {
	return filepath.Join(workDir, ".scratch", "mcp-metrics.json")
}

func newAgentSession(adapter providerAdapter, baseDir, runID, workDir string) (*agentSession, error) {
	transcriptDir := filepath.Join(baseDir, "transcripts")
	if err := os.MkdirAll(transcriptDir, 0o750); err != nil {
		return nil, fmt.Errorf("create session transcript directory: %w", err)
	}
	transcriptPath := filepath.Join(transcriptDir, runID+".json")
	workspaceRelativePath, err := filepath.Rel(baseDir, workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve session workspace relative path: %w", err)
	}
	if workspaceRelativePath == "." || workspaceRelativePath == ".." || strings.HasPrefix(workspaceRelativePath, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("session workspace %q is outside scratch directory %q", workDir, baseDir)
	}
	scratchRoot, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, fmt.Errorf("open benchmark scratch root: %w", err)
	}
	workspaceRoot, err := scratchRoot.OpenRoot(filepath.ToSlash(workspaceRelativePath))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open session workspace root: %w", err), scratchRoot.Close())
	}
	return &agentSession{
		adapter:               adapter,
		workDir:               workDir,
		scratchRoot:           scratchRoot,
		workspaceRoot:         workspaceRoot,
		workspaceRelativePath: filepath.ToSlash(workspaceRelativePath),
		transcriptPath:        transcriptPath,
		toolObservationState:  ToolObservationUnknown,
	}, nil
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

func makeSessionDirectoryWritable(workspaceRoot *os.Root, path string, walkInfo fs.FileInfo) error {
	file, err := workspaceRoot.Open(path)
	if err != nil {
		return fmt.Errorf("open workspace directory %q: %w", path, err)
	}
	openedInfo, statErr := file.Stat()
	if statErr != nil {
		closeErr := file.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close workspace directory %q: %w", path, closeErr)
		}
		return errors.Join(fmt.Errorf("stat opened workspace directory %q: %w", path, statErr), closeErr)
	}
	if !openedInfo.IsDir() || !os.SameFile(walkInfo, openedInfo) {
		closeErr := file.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close workspace directory %q: %w", path, closeErr)
		}
		return errors.Join(fmt.Errorf("workspace directory changed during cleanup: %q", path), closeErr)
	}
	mode := openedInfo.Mode().Perm() | 0o700
	if mode == openedInfo.Mode().Perm() {
		if err := file.Close(); err != nil {
			return fmt.Errorf("close workspace directory %q: %w", path, err)
		}
		return nil
	}
	chmodErr := file.Chmod(mode)
	closeErr := file.Close()
	if chmodErr != nil {
		chmodErr = fmt.Errorf("make workspace directory writable %q: %w", path, chmodErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("close workspace directory %q: %w", path, closeErr)
	}
	return errors.Join(chmodErr, closeErr)
}

func makeSessionWorkspaceRemovable(workspaceRoot *os.Root) error {
	return fs.WalkDir(workspaceRoot.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("inspect workspace path %q: %w", path, walkErr)
		}
		if !entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read workspace directory metadata %q: %w", path, err)
		}
		return makeSessionDirectoryWritable(workspaceRoot, path, info)
	})
}

func providerContinuationTransport(target Target) string {
	switch target.Harness {
	case string(HarnessCodex):
		return "codex_app_server_v2"
	case string(HarnessAgy):
		return "agy_stream_json_v1"
	case string(HarnessOpenCode):
		return "opencode_attached_server_v1"
	default:
		return ""
	}
}

func providerSessionIdentityError(originalID, returnedID string, runErr error) error {
	if strings.TrimSpace(returnedID) == "" {
		if runErr == nil {
			return fmt.Errorf("provider omitted the original session ID")
		}
		return nil
	}
	if originalID != "" && returnedID != originalID {
		return fmt.Errorf("provider changed the original session ID from %q to %q", originalID, returnedID)
	}
	return nil
}
