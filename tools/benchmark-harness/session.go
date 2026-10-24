// This file owns per-run provider selection and the lifecycle of agent sessions so every task turn shares one adapter, resume state, and transcript.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/pipeline"
)

const acceptanceOracleTimeout = 5 * time.Minute

func acceptanceOracleContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, acceptanceOracleTimeout)
}

func (s *agentSession) recordExecutionError(err error) {
	s.executionErr = joinDistinctErrors(s.executionErr, err)
}

func (s *agentSession) runTurn(ctx context.Context, prompt string, kind sessionTurnKind, result *RunResult) error {
	if s.closed {
		return fmt.Errorf("cannot run provider turn after session shutdown")
	}
	if result != nil && result == s.result && (len(s.turns) > 0 || kind == sessionDiagnosticTurn) {
		return fmt.Errorf("continuation requires a separate turn result")
	}
	if result == nil {
		return fmt.Errorf("session turn result is nil")
	}
	started := time.Now()
	var returnedID string
	runErr := s.prepareTurn(prompt, kind, result)
	if runErr == nil {
		runErr = s.ensureAdapterReady(ctx)
	}
	if runErr == nil && kind == sessionDiagnosticTurn {
		if err := s.captureTaskMCPMetrics(); err != nil {
			runErr = fmt.Errorf("capture task MCP metrics before diagnostic turn: %w", err)
		}
	}
	switch {
	case runErr != nil:
	case s.continuationErr != nil && (string(kind) != "diagnostic_reflection" || strings.TrimSpace(s.resumeID) == ""):
		runErr = fmt.Errorf("cannot continue failed provider session: %w", s.continuationErr)
	case len(s.turns) > 0 && strings.TrimSpace(s.resumeID) == "":
		runErr = fmt.Errorf("cannot continue provider session: original session ID is missing")
	default:
		returnedID, runErr = s.dispatchAdapterInvocation(ctx, prompt, kind, result, started)
	}
	if runErr != nil && s.continuationErr == nil {
		s.continuationErr = runErr
	}
	started = result.wallClockStart(started)
	if runErr != nil && result.toolObservationState != ToolObservationPartial {
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = runErr.Error()
	}
	if kind == sessionTaskTurn {
		s.recordTaskObservation(result)
	}
	refreshMCPVerified(result)
	observation := observationFromRunResult(result)
	finished := time.Now()
	s.turns = append(s.turns, sessionTurn{
		Number: len(s.turns) + 1, Classification: string(kind), Prompt: prompt,
		StartedAt: started, FinishedAt: finished, ResumeID: returnedID,
		ProviderError: errorString(runErr), ProviderExit: providerExitDescription(result),
		WallClock: finished.Sub(started), Observation: observation,
	})
	if err := s.writeTranscript(); err != nil {
		return errors.Join(runErr, err)
	}
	return runErr
}

func (s *agentSession) dispatchAdapterInvocation(ctx context.Context, prompt string, kind sessionTurnKind, result *RunResult, started time.Time) (string, error) {
	firstTurn := len(s.turns) == 0
	var command adapterInvocation
	var err error
	if firstTurn {
		command = newInitialPromptCommand(prompt, kind)
	} else {
		command, err = newFollowUpCommand(prompt, kind, s.resumeID)
	}
	if err != nil {
		return "", err
	}
	if err := validateAdapterInvocation(command, firstTurn, s.resumeID); err != nil {
		return "", err
	}

	output, runErr := s.adapter.runTurn(ctx, command)
	returnedID := output.Stage.SessionID
	applyAdapterObservation(result, output.Observation)
	identityErr := providerSessionIdentityError(s.resumeID, returnedID, runErr)
	runErr = errors.Join(runErr, identityErr)
	output.Stage = completeAdapterStage(output.Stage, adapterTurnStage, started, runErr)
	if output.Stage.ProcessStartedAt == nil {
		output.Stage.ProcessStartedAt = cloneTimePtr(output.Observation.ProcessStartedAt)
	}
	if stageErr := s.recordAdapterStage(output.Stage); stageErr != nil {
		runErr = errors.Join(runErr, stageErr)
	}
	if identityErr == nil && returnedID != "" && s.resumeID == "" {
		s.resumeID = returnedID
	}
	return returnedID, runErr
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
	if !s.taskMCPMetricsCaptured {
		if err := s.captureTaskMCPMetrics(); err != nil {
			return err
		}
	}
	if len(s.taskMCPMetrics) == 0 {
		return nil
	}
	output, err := json.MarshalIndent(s.taskMCPMetrics, "", "  ")
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

func (s *agentSession) captureTaskMCPMetrics() error {
	if s.taskMCPMetricsCaptured {
		return nil
	}
	if s.result == nil || !isSemeditArm(s.execution.Arm) {
		s.taskMCPMetricsCaptured = true
		return nil
	}
	source := benchmarkMCPMetricsPath(s.workDir)
	// #nosec G304 -- source is derived from this benchmark's isolated workspace.
	data, readErr := os.ReadFile(source)
	if errors.Is(readErr, os.ErrNotExist) {
		s.taskMCPMetricsCaptured = true
		return nil
	}
	if readErr != nil {
		return fmt.Errorf("read benchmark MCP metrics: %w", readErr)
	}
	var records []json.RawMessage
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("decode benchmark MCP metrics: %w", err)
	}
	if records == nil {
		return fmt.Errorf("benchmark MCP metrics must be a JSON array")
	}
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
	}
	s.taskMCPMetrics = records
	s.taskMCPMetricsCaptured = true
	return nil
}

func (s *agentSession) writeTranscript() error {
	data, err := json.MarshalIndent(s.turns, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session transcript: %w", err)
	}
	if err := pipeline.WriteAtomic(s.transcriptPath, data); err != nil {
		return fmt.Errorf("record session transcript: %w", err)
	}
	return nil
}

func (s *agentSession) recordAdapterStage(stage adapterStageResult) error {
	s.adapterStages = append(s.adapterStages, stage)
	if s.stageTranscriptPath == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.adapterStages, "", "  ")
	if err != nil {
		s.retainWorkDir = true
		return fmt.Errorf("encode adapter stage transcript: %w", err)
	}
	if err := pipeline.WriteAtomic(s.stageTranscriptPath, data); err != nil {
		s.retainWorkDir = true
		return fmt.Errorf("record adapter stage transcript: %w", err)
	}
	return nil
}

func (s *agentSession) lastDiagnosticProviderFailure() bool {
	if len(s.turns) == 0 {
		return false
	}
	last := s.turns[len(s.turns)-1]
	return last.Classification == "diagnostic_reflection" && last.ProviderError != ""
}

func (s *agentSession) close() (retErr error) {
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	defer func() {
		retErr = errors.Join(retErr, s.closeRoots())
		s.closeErr = retErr
	}()
	shutdown := s.adapter.shutdown()
	shutdown.Stage = completeAdapterStage(shutdown.Stage, adapterShutdownStage, time.Now(), errors.Join(shutdown.ProcessExit, shutdown.CleanupError))
	shutdown.Stage.Diagnostics = shutdown.Stderr
	if err := s.recordAdapterStage(shutdown.Stage); err != nil {
		retErr = errors.Join(retErr, err)
	}
	diagnosticExit := shutdown.ProcessExit != nil && s.lastDiagnosticProviderFailure()
	if diagnosticExit {
		last := &s.turns[len(s.turns)-1]
		last.ProviderExit = appendResultError(last.ProviderExit, shutdown.ProcessExit)
		s.retainWorkDir = true
		retErr = errors.Join(retErr, s.writeTranscript())
	} else if shutdown.ProcessExit != nil {
		retErr = errors.Join(retErr, fmt.Errorf("provider process exit: %w", shutdown.ProcessExit))
		s.retainWorkDir = true
	}
	if shutdown.CleanupError != nil {
		retErr = errors.Join(retErr, fmt.Errorf("close provider session: %w", shutdown.CleanupError))
		s.retainWorkDir = true
	}
	if err := s.retainTaskMCPMetrics(); err != nil {
		retErr = errors.Join(retErr, err)
		s.retainWorkDir = true
	}
	s.finalizeResult()
	if s.retainWorkDir {
		for _, cachePath := range []string{
			".scratch/go/build",
			".scratch/go/gopls",
			".scratch/go/tmp",
			".scratch/copilot/cache",
			".scratch/copilot/tmp",
			".scratch/copilot/home/Library/Caches",
			".scratch/opencode/cache",
			".scratch/opencode/tmp",
		} {
			if err := removeRetainedWorkspaceCache(s.workspaceRoot, cachePath); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("prune retained workspace cache %q: %w", cachePath, err))
			}
		}
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
	shouldSkipReflection := isSemeditArm(s.execution.Arm) && s.resumeID != "" && !s.result.Success && s.result.DiagnosticTimeoutReflection == nil && s.result.SemanticIntegrationReflection == nil
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

func (s *agentSession) prepareAdapter(ctx context.Context, target Target, arm ArmType, external bool) error {
	input := adapterPreparationInput{Session: adapterSessionConfig{
		WorkDir: s.workDir, Target: target, Arm: arm, External: external,
	}}
	started := time.Now()
	stage, err := s.adapter.prepare(ctx, input)
	stage = completeAdapterStage(stage, adapterPreparationStage, started, err)
	if recordErr := s.recordAdapterStage(stage); recordErr != nil {
		err = errors.Join(err, recordErr)
	}
	if err != nil {
		return fmt.Errorf("prepare provider fixture: %w", err)
	}
	s.adapterPrepared = true
	return nil
}

func (s *agentSession) warmupAdapter(ctx context.Context) error {
	if !s.adapterPrepared {
		return fmt.Errorf("provider warmup requires adapter preparation")
	}
	started := time.Now()
	stage, err := s.adapter.warmup(ctx)
	stage = completeAdapterStage(stage, adapterWarmupStage, started, err)
	if recordErr := s.recordAdapterStage(stage); recordErr != nil {
		err = errors.Join(err, recordErr)
	}
	if err != nil {
		return fmt.Errorf("warm up provider adapter: %w", err)
	}
	s.adapterWarmed = true
	return nil
}

func (s *agentSession) ensureAdapterReady(ctx context.Context) error {
	if !s.adapterPrepared {
		external := s.execution.Task != nil && s.execution.Task.Metadata.Repository.URL != ""
		if err := s.prepareAdapter(ctx, s.identity.Target, s.identity.Arm, external); err != nil {
			return err
		}
	}
	if !s.adapterWarmed {
		if err := s.warmupAdapter(ctx); err != nil {
			return err
		}
	}
	return nil
}

type agentSession struct {
	adapter                providerAdapter
	runner                 *Runner
	execution              AgentExecution
	identity               *sessionIdentity
	diagnosticsStarted     bool
	workDir                string
	scratchRoot            *os.Root
	workspaceRoot          *os.Root
	workspaceRelativePath  string
	transcriptPath         string
	stageTranscriptPath    string
	adapterStages          []adapterStageResult
	closed                 bool
	closeErr               error
	adapterPrepared        bool
	adapterWarmed          bool
	resumeID               string
	continuationErr        error
	turns                  []sessionTurn
	taskMCPMetricsCaptured bool
	taskMCPMetrics         []json.RawMessage
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
	Number         int                    `json:"number"`
	Classification string                 `json:"classification"`
	Prompt         string                 `json:"prompt"`
	StartedAt      time.Time              `json:"started_at"`
	FinishedAt     time.Time              `json:"finished_at"`
	WallClock      time.Duration          `json:"wall_clock_ns"`
	ResumeID       string                 `json:"resume_id,omitempty"`
	ProviderError  string                 `json:"provider_error,omitempty"`
	ProviderExit   string                 `json:"provider_exit,omitempty"`
	Observation    adapterTurnObservation `json:"observation"`
}

func selectProvider(runner *Runner, target Target) (providerAdapter, error) {
	switch target.Harness {
	case string(HarnessCodex):
		return &codexProvider{runner: runner}, nil
	case string(HarnessAgy):
		return &agyProvider{runner: runner}, nil
	case string(HarnessOpenCode):
		return &openCodeProvider{runner: runner}, nil

	case string(HarnessCopilot):
		return &copilotProvider{runner: runner}, nil
	default:
		return nil, fmt.Errorf("unsupported harness: %s", target.Harness)
	}
}

func benchmarkMCPMetricsPath(workDir string) string {
	return filepath.Join(workDir, ".scratch", "mcp-metrics.json")
}

func newAgentSession(adapter providerAdapter, persistentDir, fixtureDir, runID, workDir string) (*agentSession, error) {
	transcriptDir := filepath.Join(persistentDir, "transcripts")
	if err := os.MkdirAll(transcriptDir, 0o750); err != nil {
		return nil, fmt.Errorf("create session transcript directory: %w", err)
	}
	transcriptPath := filepath.Join(transcriptDir, runID+".json")
	stageTranscriptPath := filepath.Join(transcriptDir, runID+".stages.json")
	workspaceRelativePath, err := filepath.Rel(fixtureDir, workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve session workspace relative path: %w", err)
	}
	if workspaceRelativePath == "." || workspaceRelativePath == ".." || strings.HasPrefix(workspaceRelativePath, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("session workspace %q is outside fixture root %q", workDir, fixtureDir)
	}
	scratchRoot, err := os.OpenRoot(fixtureDir)
	if err != nil {
		return nil, fmt.Errorf("open fixture root: %w", err)
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
		stageTranscriptPath:   stageTranscriptPath,
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

	case string(HarnessCopilot):
		return "copilot_acp_v1"
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

type sessionTurnKind string

const (
	sessionTaskTurn       sessionTurnKind = "task"
	sessionDiagnosticTurn sessionTurnKind = "diagnostic_reflection"
)

type sessionIdentity struct {
	WorkDir string
	Target  Target
	Arm     ArmType
}

func (s *agentSession) prepareTurn(prompt string, kind sessionTurnKind, result *RunResult) error {
	if kind != sessionTaskTurn && kind != sessionDiagnosticTurn {
		return fmt.Errorf("unsupported session turn kind %q", kind)
	}
	identity := sessionIdentity{WorkDir: s.workDir, Target: s.execution.Target, Arm: s.execution.Arm}
	if s.identity == nil {
		s.identity = &identity
	} else if *s.identity != identity {
		return fmt.Errorf("session workspace, target, or arm changed")
	}
	result.Target, result.Arm, result.Prompt = s.identity.Target, s.identity.Arm, prompt
	if kind == sessionTaskTurn && s.diagnosticsStarted {
		return fmt.Errorf("cannot run task work after diagnostics started")
	}
	if kind == sessionDiagnosticTurn {
		s.diagnosticsStarted = true
	}
	return nil
}

func (s *agentSession) recordTaskObservation(result *RunResult) {
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

func removeRetainedWorkspaceCache(workspaceRoot *os.Root, relative string) error {
	clean := filepath.Clean(relative)
	if !filepath.IsLocal(clean) || clean == "." {
		return fmt.Errorf("retained cache path %q is not workspace-local", relative)
	}
	for parent := filepath.Dir(clean); parent != "."; parent = filepath.Dir(parent) {
		info, err := fs.Stat(workspaceRoot.FS(), parent)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect retained cache parent %q: %w", parent, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("retained cache parent %q is not a directory", parent)
		}
		if err := makeSessionDirectoryWritable(workspaceRoot, parent, info); err != nil {
			return err
		}
	}
	walkErr := fs.WalkDir(workspaceRoot.FS(), clean, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("inspect retained cache path %q: %w", path, walkErr)
		}
		if !entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read retained cache directory metadata %q: %w", path, err)
		}
		return makeSessionDirectoryWritable(workspaceRoot, path, info)
	})
	if errors.Is(walkErr, fs.ErrNotExist) {
		return nil
	}
	if walkErr != nil {
		return walkErr
	}
	if err := workspaceRoot.RemoveAll(clean); err != nil {
		return fmt.Errorf("remove retained cache %q: %w", clean, err)
	}
	return nil
}
