// Package bench implements the evaluation harness and multi-level correctness oracle for semedit benchmarks.
package bench

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ArmType identifies the evaluation arm.
type ArmType string

const (
	// ArmBaseline represents text diff / direct file edit tools.
	ArmBaseline ArmType = "baseline-diff"
	// ArmSemedit represents semedit semantic AST tools.
	ArmSemedit ArmType = "semedit"
	// ArmControl represents direct semedit execution without an LLM.
	ArmControl ArmType = "control"
)

// RunConfig configures a benchmark execution run.
type RunConfig struct {
	Arm        ArmType
	Task       *Task
	ScratchDir string
	Timeout    time.Duration
}

// RunResult aggregates telemetry, performance metrics, and oracle outcomes.
type RunResult struct {
	TaskID          string        `json:"task_id"`
	Arm             ArmType       `json:"arm"`
	Success         bool          `json:"success"`
	Turns           int           `json:"turns"`
	WallClock       time.Duration `json:"wall_clock_ms"`
	PromptTokens    int           `json:"prompt_tokens"`
	OutputTokens    int           `json:"output_tokens"`
	ReasoningTokens int           `json:"reasoning_tokens"`
	Oracle          *OracleResult `json:"oracle"`
	Error           string        `json:"error,omitempty"`
}

// Runner coordinates execution across evaluation arms and benchmarks.
type Runner struct {
	baseScratchDir string
}

// NewRunner initializes a benchmark runner.
func NewRunner(scratchDir string) *Runner {
	if scratchDir == "" {
		scratchDir = filepath.Join(".scratch", "benchmarks")
	}
	return &Runner{
		baseScratchDir: scratchDir,
	}
}

// ExecuteControl runs a benchmark task using deterministic semedit operations directly.
func (r *Runner) ExecuteControl(ctx context.Context, task *Task) (*RunResult, error) {
	start := time.Now()
	runID := fmt.Sprintf("run_control_%s_%d", task.Metadata.TaskID, time.Now().UnixNano())
	workDir := filepath.Join(r.baseScratchDir, runID)

	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(workDir)
	}()

	if err := task.ExtractTo(workDir); err != nil {
		return nil, fmt.Errorf("extract task fixture: %w", err)
	}

	res := &RunResult{
		TaskID:    task.Metadata.TaskID,
		Arm:       ArmControl,
		WallClock: time.Since(start),
		Turns:     1,
	}

	return res, nil
}
