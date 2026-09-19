// Package main implements the evaluation harness and multi-level correctness oracle for semedit benchmarks.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
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
	TaskID               string        `json:"task_id"`
	Variant              string        `json:"variant,omitempty"` // "small", "large"
	PromptVariant        string        `json:"prompt_variant,omitempty"`
	Target               Target        `json:"target"`
	Arm                  ArmType       `json:"arm"`
	Success              bool          `json:"success"`
	Turns                int           `json:"turns"`
	InitialLoadTurns     int           `json:"initial_load_turns"`
	MCPLoadTurns         int           `json:"mcp_load_turns"`
	InternalTurns        int           `json:"internal_turns"`
	ToolCount            int           `json:"tool_count"`
	WallClock            time.Duration `json:"wall_clock_ms"`
	InitialContextTokens int           `json:"initial_context_tokens"`
	PromptTokens         int           `json:"prompt_tokens"`
	CachedPromptTokens   int           `json:"cached_prompt_tokens"`
	UncachedPromptTokens int           `json:"uncached_prompt_tokens"`
	OutputTokens         int           `json:"output_tokens"`
	ReasoningTokens      int           `json:"reasoning_tokens"`
	Oracle               *OracleResult `json:"oracle"`
	Prompt               string        `json:"prompt,omitempty"`
	BeforeState          string        `json:"before_state,omitempty"`
	Diff                 string        `json:"diff,omitempty"`
	ToolsUsed            []string      `json:"tools_used,omitempty"`
	MCPVerified          bool          `json:"mcp_verified"`
	Error                string        `json:"error,omitempty"`
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
	if abs, err := filepath.Abs(scratchDir); err == nil {
		scratchDir = abs
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

	// #nosec G703,G301 -- creating isolated run scratch directory
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	defer func() {
		// #nosec G703 -- cleanup of isolated run scratch directory
		_ = os.RemoveAll(workDir)
	}()

	if err := task.ExtractTo(workDir); err != nil {
		return nil, fmt.Errorf("extract task fixture: %w", err)
	}

	var modifiedFiles []string
	var execErr error

	switch task.Metadata.TaskID {
	case "task-01-rename-local", "task-01b-rename-local-large-context":
		// Rename method Server.oldName to Server.newName in api/server.go
		res, err := symbol.Resolve(workDir, "api/server.go", "Server.oldName")
		if err != nil {
			execErr = fmt.Errorf("resolve Server.oldName: %w", err)
			break
		}
		if err := golang.Rename(ctx, workDir, res.File, res.Line, res.Column, "newName"); err != nil {
			execErr = fmt.Errorf("rename Server.oldName to newName: %w", err)
			break
		}
		_ = pipeline.Format(ctx, workDir, "api/server.go")
		modifiedFiles = append(modifiedFiles, "api/server.go")

	case "task-02-rename-cross-pkg":
		// Rename ValidateToken to VerifyToken across packages
		res, err := symbol.Resolve(workDir, "auth/token.go", "ValidateToken")
		if err != nil {
			execErr = fmt.Errorf("resolve ValidateToken: %w", err)
			break
		}
		if err := golang.Rename(ctx, workDir, res.File, res.Line, res.Column, "VerifyToken"); err != nil {
			execErr = fmt.Errorf("rename ValidateToken to VerifyToken: %w", err)
			break
		}
		_ = pipeline.Format(ctx, workDir, ".")
		modifiedFiles = append(modifiedFiles, "auth/token.go", "cmd/main.go")

	case "task-03-rename-interface":
		// Rename interface method Storage.Save to Storage.Persist
		res, err := symbol.Resolve(workDir, "store/store.go", "Storage.Save")
		if err != nil {
			execErr = fmt.Errorf("resolve Storage.Save: %w", err)
			break
		}
		if err := golang.Rename(ctx, workDir, res.File, res.Line, res.Column, "Persist"); err != nil {
			execErr = fmt.Errorf("rename Storage.Save to Persist: %w", err)
			break
		}
		_ = pipeline.Format(ctx, workDir, ".")
		modifiedFiles = append(modifiedFiles, "store/store.go")

	case "task-04-insert-public":
		// Insert public constructor InitServer() *Server before private helpers
		source := "func InitServer() *Server {\n\treturn &Server{Port: 8080}\n}"
		opts := astedit.Options{
			Placement:    astedit.PlacementBeforeSymbol,
			TargetSymbol: "internalRun",
			Visibility:   "public",
		}
		targetPath := filepath.Join(workDir, "api", "server.go")
		if err := astedit.InsertDeclaration(ctx, targetPath, source, opts); err != nil {
			execErr = fmt.Errorf("insert InitServer: %w", err)
			break
		}
		modifiedFiles = append(modifiedFiles, "api/server.go")

	case "task-05-insert-method":
		// Insert method func (s *Server) Stop() after Server.Start
		source := "func (s *Server) Stop() {\n\ts.running = false\n}"
		opts := astedit.Options{
			Placement:    astedit.PlacementAfterSymbol,
			TargetSymbol: "Server.Start",
		}
		targetPath := filepath.Join(workDir, "api", "server.go")
		if err := astedit.InsertDeclaration(ctx, targetPath, source, opts); err != nil {
			execErr = fmt.Errorf("insert Server.Stop: %w", err)
			break
		}
		modifiedFiles = append(modifiedFiles, "api/server.go")

	case "task-06-imports-cleanup":
		// Clean unused imports in api/server.go
		targetFile := filepath.Join(workDir, "api", "server.go")
		if err := pipeline.OrganizeImports(ctx, workDir, targetFile); err != nil {
			execErr = fmt.Errorf("organize imports: %w", err)
			break
		}
		modifiedFiles = append(modifiedFiles, "api/server.go")

	default:
		execErr = fmt.Errorf("task %s has no registered control transformation", task.Metadata.TaskID)
	}

	res := &RunResult{
		TaskID:    task.Metadata.TaskID,
		Arm:       ArmControl,
		WallClock: time.Since(start),
		Turns:     1,
	}

	if execErr != nil {
		res.Error = execErr.Error()
		return res, nil
	}

	oracleRes, err := Evaluate(ctx, task, workDir, modifiedFiles)
	if err != nil {
		res.Error = fmt.Sprintf("oracle evaluation: %v", err)
		return res, nil
	}

	res.Oracle = oracleRes
	res.Success = oracleRes.Passed
	return res, nil
}
