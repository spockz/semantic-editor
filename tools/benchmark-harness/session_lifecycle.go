// This file keeps task sequencing, measurement boundaries, and diagnostic policy shared across provider adapters.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func (s *agentSession) runFollowups(taskCtx, oracleCtx context.Context) {
	result := s.result
	for index, followup := range s.execution.Followups {
		if result.Success {
			break
		}
		turn := &RunResult{}
		turnStarted := time.Now()
		runErr := s.runTurn(taskCtx, followup, sessionTaskTurn, turn)
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
			return
		}
		if err := s.evaluate(oracleCtx, turn); err != nil {
			turn.WallClock = turn.measuredWallClock(turnStarted)
			turn.Error = err.Error()
			result.Error = err.Error()
			result.WallClock = elapsedTaskTime(s.started)
			s.retainWorkDir = shouldRetainWorkDir(oracleCtx, err)
			mergeTurn(result, turn)
			result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
			return
		}
		turn.WallClock = turn.measuredWallClock(turnStarted)
		result.InteractionSteps = append(result.InteractionSteps, interactionStep(index+2, followup, turn))
		mergeTurn(result, turn)
	}
}

func (s *agentSession) runStagedFollowups(taskCtx, oracleCtx context.Context) {
	result := s.result
	allPassed := result.Success
	for index, followup := range s.execution.StagedFollowups {
		turn := &RunResult{}
		turnStarted := time.Now()
		runErr := s.runTurn(taskCtx, followup.Instruction, sessionTaskTurn, turn)
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
			return
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
			return
		}
		turn.WallClock = turn.measuredWallClock(turnStarted)
		allPassed = allPassed && turn.Success
		result.InteractionSteps = append(result.InteractionSteps, stagedInteractionStep(index+2, followup.TotalEdits, followup.Instruction, turn))
		mergeTurn(result, turn)
		result.Success = allPassed
	}
	result.WallClock = elapsedTaskTime(s.started)
}

func mergeTurn(total, turn *RunResult) {
	total.Turns += turn.Turns
	total.InitialLoadTurns += turn.InitialLoadTurns
	total.MCPLoadTurns += turn.MCPLoadTurns
	total.InternalTurns += turn.InternalTurns
	total.ToolCalls = append(total.ToolCalls, turn.ToolCalls...)
	total.ToolsUsed = append(total.ToolsUsed, turn.ToolsUsed...)
	total.ToolCount = len(total.ToolCalls)
	total.PromptTokens += turn.PromptTokens
	total.CachedPromptTokens += turn.CachedPromptTokens
	total.UncachedPromptTokens += turn.UncachedPromptTokens
	total.OutputTokens += turn.OutputTokens
	total.ReasoningTokens += turn.ReasoningTokens
	if total.ProcessStartToFirstEvent == nil {
		total.ProcessStartToFirstEvent = cloneDuration(turn.ProcessStartToFirstEvent)
	}
	if total.FirstEventToFirstToolCall == nil {
		total.FirstEventToFirstToolCall = cloneDuration(turn.FirstEventToFirstToolCall)
	}
	if total.MCPInitializeToFirstSemanticCall == nil {
		total.MCPInitializeToFirstSemanticCall = cloneDuration(turn.MCPInitializeToFirstSemanticCall)
	}
	if total.MCPServerStartToInitialize == nil {
		total.MCPServerStartToInitialize = cloneDuration(turn.MCPServerStartToInitialize)
	}
	total.MCPVerified = total.MCPVerified || turn.MCPVerified
	if turn.Oracle != nil {
		total.Oracle, total.Success = turn.Oracle, turn.Success
	}
	if turn.Error != "" {
		total.Error = appendResultError(total.Error, errors.New(turn.Error))
	}
	for key, value := range turn.Provenance {
		if value == "unavailable" {
			ensureProvenance(total)
			total.Provenance[key] = value
		}
	}
}

func interactionStep(step int, prompt string, res *RunResult) InteractionStep {
	return InteractionStep{
		Step:                             step,
		Prompt:                           prompt,
		WallClock:                        res.WallClock,
		Turns:                            res.Turns,
		PromptTokens:                     res.PromptTokens,
		CachedPromptTokens:               res.CachedPromptTokens,
		UncachedPromptTokens:             res.UncachedPromptTokens,
		OutputTokens:                     res.OutputTokens,
		ReasoningTokens:                  res.ReasoningTokens,
		InitialLoadTurns:                 res.InitialLoadTurns,
		MCPLoadTurns:                     res.MCPLoadTurns,
		InternalTurns:                    res.InternalTurns,
		ProcessStartToFirstEvent:         res.ProcessStartToFirstEvent,
		MCPInitializeToFirstSemanticCall: res.MCPInitializeToFirstSemanticCall,
		ToolCalls:                        res.ToolCalls,
		Oracle:                           res.Oracle,
		Error:                            res.Error,
	}
}

func stagedInteractionStep(step int, totalEdits int, prompt string, res *RunResult) InteractionStep {
	result := interactionStep(step, prompt, res)
	result.TotalEdits = totalEdits
	return result
}

func initialInteractionStep(task *Task, prompt string, res *RunResult) InteractionStep {
	if task.Metadata.InteractiveMode == "staged" {
		return stagedInteractionStep(1, task.Metadata.StagedInitialTotalEdits, prompt, res)
	}
	return interactionStep(1, prompt, res)
}

func (s *agentSession) execute(ctx context.Context, prompt string) {
	taskCtx, cancelTask := agentTaskContext(ctx)
	defer cancelTask()
	stopEarly := s.runInitialTask(taskCtx, ctx, prompt)
	if shouldRequestTimeoutReflections(taskCtx, ctx, s, s.execution.Arm) {
		s.requestTimeoutReflections(ctx)
	}
	if stopEarly {
		return
	}
	if s.execution.Task.Metadata.InteractiveMode == "staged" {
		s.runStagedFollowups(taskCtx, ctx)
	} else {
		s.runFollowups(taskCtx, ctx)
	}
	s.result.WallClock = elapsedTaskTime(s.started)
	if shouldRequestTimeoutReflections(taskCtx, ctx, s, s.execution.Arm) {
		s.requestTimeoutReflections(ctx)
	} else {
		s.runDiagnostics(ctx)
	}
}

func (s *agentSession) runInitialTask(taskCtx, oracleCtx context.Context, prompt string) bool {
	result, task, target := s.result, s.execution.Task, s.execution.Target
	runErr := s.runTurn(taskCtx, prompt, sessionTaskTurn, result)
	s.started = result.processStartedAt
	result.WallClock = elapsedTaskTime(s.started)
	if runErr != nil {
		s.retainWorkDir = shouldRetainWorkDir(taskCtx, runErr)
		primaryErr := fmt.Errorf("%s execution: %w", target.Harness, runErr)
		s.recordExecutionError(primaryErr)
		result.Error = appendResultError(result.Error, primaryErr)
		result.InteractionSteps = append(result.InteractionSteps, initialInteractionStep(task, prompt, result))
		return true
	}
	if err := s.evaluate(oracleCtx, result); err != nil {
		result.WallClock = elapsedTaskTime(s.started)
		s.retainWorkDir = shouldRetainWorkDir(oracleCtx, err)
		result.Error = err.Error()
		result.InteractionSteps = append(result.InteractionSteps, initialInteractionStep(task, prompt, result))
		return true
	}
	if task.Metadata.InteractiveMode == "staged" {
		result.WallClock = elapsedTaskTime(s.started)
	}
	result.InteractionSteps = append(result.InteractionSteps, initialInteractionStep(task, prompt, result))
	return false
}

func (s *agentSession) runDiagnostics(ctx context.Context) {
	result, arm := s.result, s.execution.Arm
	if shouldRequestSemanticBatchReflection(arm, s.resumeID, result) {
		result.SemanticBatchReflection = s.requestSemanticToolReflection(ctx, semanticBatchReflectionPrompt)
	} else if shouldRequestSemanticToolReflection(arm, s.resumeID, result) {
		result.SemanticToolReflection = s.requestSemanticToolReflection(ctx, semanticToolReflectionPrompt)
	}
	if isSemeditArm(arm) && s.resumeID != "" && result.Success {
		result.SemanticIntegrationReflection = s.requestSemanticToolReflection(ctx, semanticIntegrationReflectionPrompt)
	}
}

func shouldRequestTimeoutReflections(taskCtx, parentCtx context.Context, session *agentSession, arm ArmType) bool {
	if !isSemeditArm(arm) || parentCtx.Err() != nil {
		return false
	}
	return errors.Is(taskCtx.Err(), context.DeadlineExceeded) || errors.Is(session.executionErr, context.DeadlineExceeded)
}

func (s *agentSession) requestTimeoutReflections(ctx context.Context) {
	if s.resumeID == "" {
		s.result.DiagnosticReflectionSkipped = "provider did not expose a resumable session ID after timeout"
		return
	}
	reflectionCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	s.result.DiagnosticTimeoutReflection = s.requestSemanticToolReflection(reflectionCtx, semanticTimeoutDiagnosticReflectionPrompt)
	s.result.SemanticIntegrationReflection = s.requestSemanticToolReflection(reflectionCtx, semanticTimeoutIntegrationReflectionPrompt)
}

func (s *agentSession) requestSemanticToolReflection(ctx context.Context, prompt string) *SemanticToolReflection {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	turn := (&RunResult{})
	started := time.Now()
	runErr := s.runTurn(ctx, prompt, sessionDiagnosticTurn, turn)
	reflection := &SemanticToolReflection{Prompt: prompt, Response: turn.agentResponse, WallClock: time.Since(started), Turns: turn.Turns, ToolCalls: turn.ToolCalls}
	if runErr != nil {
		reflection.Error = runErr.Error()
	} else if reflection.Response == "" {
		reflection.Error = "harness did not expose an assistant reflection response"
	}
	return reflection
}
