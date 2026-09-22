// Package main implements the evaluation harness and multi-level correctness oracle for semedit benchmarks.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// MCPServerInstructionMode identifies the server-wide instruction policy for a benchmark cell.
type MCPServerInstructionMode string

const (
	// MCPServerInstructionsNone keeps the MCP server neutral so prompt wording is the only intervention.
	MCPServerInstructionsNone MCPServerInstructionMode = "none"
	// MCPServerInstructionsDescriptive advertises semantic operations without directing their use.
	MCPServerInstructionsDescriptive MCPServerInstructionMode = "descriptive"
	// MCPServerInstructionsPrescriptive directs the model to prefer semantic source mutations.
	MCPServerInstructionsPrescriptive MCPServerInstructionMode = "prescriptive"
)

// ParseMCPServerInstructionMode validates a CLI-supplied server instruction experiment mode.
func ParseMCPServerInstructionMode(raw string) (MCPServerInstructionMode, error) {
	mode := MCPServerInstructionMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case "", MCPServerInstructionsNone:
		return MCPServerInstructionsNone, nil
	case MCPServerInstructionsDescriptive, MCPServerInstructionsPrescriptive:
		return mode, nil
	case "directive":
		return MCPServerInstructionsPrescriptive, nil
	default:
		return "", fmt.Errorf("unsupported MCP instruction mode %q (want none, descriptive, or prescriptive)", raw)
	}
}

// ProvenanceSet records technical execution context without defining a benchmark comparison cell.
type ProvenanceSet map[string]string

// Set parses a repeatable key=value provenance entry while preventing ambiguous duplicates.
func (c *ProvenanceSet) Set(raw string) error {
	key, value, ok := strings.Cut(raw, "=")
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if !ok || key == "" || value == "" {
		return fmt.Errorf("provenance must use key=value form, got %q", raw)
	}
	if strings.ContainsAny(key, "=,\t\n") || strings.ContainsAny(value, "\t\n") {
		return fmt.Errorf("invalid provenance %q", raw)
	}
	if *c == nil {
		*c = make(ProvenanceSet)
	}
	if existing, exists := (*c)[key]; exists && existing != value {
		return fmt.Errorf("provenance %q already has value %q", key, existing)
	}
	(*c)[key] = value
	return nil
}

// String returns a stable representation suitable for logs and presentation.
func (c ProvenanceSet) String() string {
	if len(c) == 0 {
		return ""
	}
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+c[key])
	}
	return strings.Join(parts, ",")
}

// Clone returns an independent provenance set so every result snapshots its execution context.
func (c ProvenanceSet) Clone() ProvenanceSet {
	if len(c) == 0 {
		return nil
	}
	clone := make(ProvenanceSet, len(c))
	maps.Copy(clone, c)
	return clone
}

// ToolCallStatus distinguishes confirmed outcomes from harnesses that expose only a planned tool call.
type ToolCallStatus string

const (
	ToolCallStatusSucceeded ToolCallStatus = "succeeded"
	ToolCallStatusFailed    ToolCallStatus = "failed"
	ToolCallStatusUnknown   ToolCallStatus = "unknown"
)

// ToolCall captures the transport and functional outcome of one tool invocation.
type ToolCall struct {
	Name              string             `json:"name"`
	Server            string             `json:"server,omitempty"`
	Arguments         json.RawMessage    `json:"arguments,omitempty"`
	TransportStatus   ToolCallStatus     `json:"transport_status"`
	FunctionalStatus  ToolCallStatus     `json:"functional_status"`
	Failure           string             `json:"failure,omitempty"`
	MCPMetrics        *MCPMetrics        `json:"mcp_metrics,omitempty"`
	MCPStartupMetrics *MCPStartupMetrics `json:"mcp_startup_metrics,omitempty"`
}

// InteractionStep preserves the independent outcome of one turn in a continued agent session.
type InteractionStep struct {
	Step      int           `json:"step"`
	Prompt    string        `json:"prompt"`
	WallClock time.Duration `json:"wall_clock_ms"`
	Turns     int           `json:"turns"`
	ToolCalls []ToolCall    `json:"tool_calls,omitempty"`
	Oracle    *OracleResult `json:"oracle,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// SemanticToolReflection records a diagnostic-only follow-up after semantic tool-use behavior needs explanation.
type SemanticToolReflection struct {
	Prompt    string        `json:"prompt"`
	Response  string        `json:"response,omitempty"`
	WallClock time.Duration `json:"wall_clock_ms"`
	Turns     int           `json:"turns"`
	ToolCalls []ToolCall    `json:"tool_calls,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// MCPMetrics records server-observed latency returned in an MCP tool response.
type MCPMetrics struct {
	SchemaVersion int                       `json:"schema_version"`
	TotalMS       int64                     `json:"total_ms"`
	Phases        map[string]MCPPhaseMetric `json:"phases,omitempty"`
}

// MCPPhaseMetric aggregates one instrumented server phase.
type MCPPhaseMetric struct {
	Count      int   `json:"count"`
	DurationMS int64 `json:"duration_ms"`
}

// MCPStartupMetrics records server-session timing reported with the first semantic call.
type MCPStartupMetrics struct {
	ServerStartToInitializeMS       int64 `json:"server_start_to_initialize_ms"`
	InitializeToFirstSemanticCallMS int64 `json:"initialize_to_first_semantic_call_ms"`
}

// RunConfig configures a benchmark execution run.
type RunConfig struct {
	Arm        ArmType
	Task       *Task
	ScratchDir string
	Timeout    time.Duration
}

// RunResult aggregates telemetry, performance metrics, and oracle outcomes.
type RunResult struct {
	TaskID                           string                   `json:"task_id"`
	Variant                          string                   `json:"variant,omitempty"` // "small", "large"
	Repeat                           int                      `json:"repeat,omitempty"`
	PromptVariant                    string                   `json:"prompt_variant,omitempty"`
	MCPServerInstructions            MCPServerInstructionMode `json:"mcp_server_instructions,omitempty"`
	Provenance                       ProvenanceSet            `json:"provenance,omitempty"`
	Target                           Target                   `json:"target"`
	Arm                              ArmType                  `json:"arm"`
	Success                          bool                     `json:"success"`
	Turns                            int                      `json:"turns"`
	InitialLoadTurns                 int                      `json:"initial_load_turns"`
	MCPLoadTurns                     int                      `json:"mcp_load_turns"`
	InternalTurns                    int                      `json:"internal_turns"`
	ToolCount                        int                      `json:"tool_count"`
	WallClock                        time.Duration            `json:"wall_clock_ms"`
	ProcessStartToFirstEvent         *time.Duration           `json:"process_start_to_first_event_ms,omitempty"`
	FirstEventToFirstToolCall        *time.Duration           `json:"first_event_to_first_tool_call_ms,omitempty"`
	MCPInitializeToFirstSemanticCall *time.Duration           `json:"mcp_initialize_to_first_semantic_call_ms,omitempty"`
	MCPServerStartToInitialize       *time.Duration           `json:"mcp_server_start_to_initialize_ms,omitempty"`
	InitialContextTokens             int                      `json:"initial_context_tokens"`
	PromptTokens                     int                      `json:"prompt_tokens"`
	CachedPromptTokens               int                      `json:"cached_prompt_tokens"`
	UncachedPromptTokens             int                      `json:"uncached_prompt_tokens"`
	OutputTokens                     int                      `json:"output_tokens"`
	ReasoningTokens                  int                      `json:"reasoning_tokens"`
	Oracle                           *OracleResult            `json:"oracle"`
	Prompt                           string                   `json:"prompt,omitempty"`
	BeforeState                      string                   `json:"before_state,omitempty"`
	Diff                             string                   `json:"diff,omitempty"`
	ToolsUsed                        []string                 `json:"tools_used,omitempty"`
	ToolCalls                        []ToolCall               `json:"tool_calls,omitempty"`
	InteractionSteps                 []InteractionStep        `json:"interaction_steps,omitempty"`
	SemanticToolReflection           *SemanticToolReflection  `json:"semantic_tool_reflection,omitempty"`
	SemanticBatchReflection          *SemanticToolReflection  `json:"semantic_batch_reflection,omitempty"`
	MCPVerified                      bool                     `json:"mcp_verified"`
	Error                            string                   `json:"error,omitempty"`
	CodexExitCode                    *int                     `json:"codex_exit_code,omitempty"`
	CodexStderr                      string                   `json:"codex_stderr,omitempty"`
	OpenCodeExitCode                 *int                     `json:"opencode_exit_code,omitempty"`
	OpenCodeStderr                   string                   `json:"opencode_stderr,omitempty"`
	agentResponse                    string
}

// Runner coordinates execution across evaluation arms and benchmarks.
type Runner struct {
	baseScratchDir            string
	mcpServerInstructionsMode MCPServerInstructionMode
	provenance                ProvenanceSet
}

// RunnerOption configures one benchmark runner without adding global process state.
type RunnerOption func(*Runner)

// WithMCPServerInstructions selects the server instruction experiment mode for all runs made by a runner.
func WithMCPServerInstructions(mode MCPServerInstructionMode) RunnerOption {
	return func(r *Runner) { r.mcpServerInstructionsMode = mode }
}

// WithProvenance attaches immutable execution provenance to every benchmark result.
func WithProvenance(provenance ProvenanceSet) RunnerOption {
	return func(r *Runner) { r.provenance = provenance.Clone() }
}

// NewRunner initializes a benchmark runner.
func NewRunner(scratchDir string, options ...RunnerOption) *Runner {
	if scratchDir == "" {
		scratchDir = filepath.Join(".scratch", "benchmarks")
	}
	if abs, err := filepath.Abs(scratchDir); err == nil {
		scratchDir = abs
	}
	runner := &Runner{
		baseScratchDir:            scratchDir,
		mcpServerInstructionsMode: MCPServerInstructionsNone,
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

func (r *Runner) provenanceFor() ProvenanceSet {
	return r.provenance.Clone()
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

	case "task-11-mixed-sink-api-migration":
		normalizePath := filepath.Join(workDir, "audit", "normalize.go")
		if err := astedit.InsertFunction(ctx, normalizePath, "func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}", astedit.FunctionOptions{
			Placement:           astedit.PlacementFileEnd,
			AutoOrganizeImports: true,
		}); err != nil {
			execErr = fmt.Errorf("insert audit NormalizeKind: %w", err)
			break
		}
		if _, err := astedit.ReplaceBody(ctx, filepath.Join(workDir, "audit", "summary.go"), "Classify", "switch NormalizeKind(kind) {\ncase \"health\", \"probe\":\n\treturn \"control\"\ndefault:\n\treturn \"data\"\n}", astedit.BodyOptions{}); err != nil {
			execErr = fmt.Errorf("replace audit Classify body: %w", err)
			break
		}
		if _, err := astedit.ReplaceBody(ctx, filepath.Join(workDir, "service", "dispatch.go"), "(*Dispatcher).Record", "event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)", astedit.BodyOptions{}); err != nil {
			execErr = fmt.Errorf("replace Dispatcher.Record body: %w", err)
			break
		}
		modifiedFiles = append(modifiedFiles, "audit/normalize.go", "audit/summary.go", "service/dispatch.go")

	default:
		execErr = fmt.Errorf("task %s has no registered control transformation", task.Metadata.TaskID)
	}

	res := &RunResult{
		TaskID:                task.Metadata.TaskID,
		MCPServerInstructions: MCPServerInstructionsNone,
		Provenance:            r.provenanceFor(),
		Arm:                   ArmControl,
		WallClock:             time.Since(start),
		Turns:                 1,
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
