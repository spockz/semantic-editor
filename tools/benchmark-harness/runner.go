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

	"github.com/spockz/semantic-editor/internal/adapters/golang"
	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/symbol"
)

// ArmType identifies the evaluation arm.
type ArmType string

const (
	// ArmBaseline represents text diff / direct file edit tools.
	ArmBaseline ArmType = "baseline-diff"
	// ArmSemedit represents semedit semantic AST tools.
	ArmSemedit ArmType = "semedit"
	// ArmSemeditAgentic identifies the autonomous semantic-tool arm for repository benchmarks.
	ArmSemeditAgentic ArmType = "semedit-agentic"
	// ArmControl represents direct semedit execution without an LLM.
	ArmControl ArmType = "control"
)

const ArmSemeditExplicit ArmType = "semedit-explicit"

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
	case "":
		return MCPServerInstructionsPrescriptive, nil
	case MCPServerInstructionsNone:
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
	Step                             int            `json:"step"`
	TotalEdits                       int            `json:"total_edits,omitempty"`
	Prompt                           string         `json:"prompt"`
	WallClock                        time.Duration  `json:"wall_clock_ms"`
	Turns                            int            `json:"turns"`
	PromptTokens                     int            `json:"prompt_tokens"`
	CachedPromptTokens               int            `json:"cached_prompt_tokens"`
	UncachedPromptTokens             int            `json:"uncached_prompt_tokens"`
	OutputTokens                     int            `json:"output_tokens"`
	ReasoningTokens                  int            `json:"reasoning_tokens"`
	InitialLoadTurns                 int            `json:"initial_load_turns"`
	MCPLoadTurns                     int            `json:"mcp_load_turns"`
	InternalTurns                    int            `json:"internal_turns"`
	ProcessStartToFirstEvent         *time.Duration `json:"process_start_to_first_event_ms,omitempty"`
	MCPInitializeToFirstSemanticCall *time.Duration `json:"mcp_initialize_to_first_semantic_call_ms,omitempty"`
	ToolCalls                        []ToolCall     `json:"tool_calls,omitempty"`
	Oracle                           *OracleResult  `json:"oracle,omitempty"`
	Error                            string         `json:"error,omitempty"`
}

type DiagnosticToolCoverage struct {
	Expected           []string             `json:"expected"`
	Observed           []string             `json:"observed"`
	Missing            []string             `json:"missing"`
	ObservationState   ToolObservationState `json:"observation_state"`
	Complete           bool                 `json:"complete"`
	ExpectedSatisfied  bool                 `json:"expected_satisfied"`
	CompletenessReason string               `json:"completeness_reason,omitempty"`
}

type ToolObservationState string

const (
	ToolObservationUnknown  ToolObservationState = "unknown"
	ToolObservationComplete ToolObservationState = "complete"
	ToolObservationPartial  ToolObservationState = "partial"
)

// SemanticToolReflection records a diagnostic-only follow-up after semantic tool-use behavior needs explanation.
type SemanticToolReflection struct {
	Prompt    string        `json:"prompt"`
	Response  string        `json:"response,omitempty"`
	WallClock time.Duration `json:"-"`
	Turns     int           `json:"-"`
	ToolCalls []ToolCall    `json:"-"`
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
	Complexity                       *ComplexityProfile       `json:"complexity,omitempty"`
	ExecutionMode                    string                   `json:"execution_mode,omitempty"`
	SemeditSkill                     bool                     `json:"semedit_skill"`
	JobID                            string                   `json:"job_id,omitempty"`
	ComparisonPairID                 string                   `json:"comparison_pair_id,omitempty"`
	Variant                          string                   `json:"variant,omitempty"`
	Repeat                           int                      `json:"repeat,omitempty"`
	PromptVariant                    string                   `json:"prompt_variant,omitempty"`
	SemeditArmRestrict               SemeditArmRestriction    `json:"semedit_arm_restrict,omitempty"`
	SemeditArmRestrictionApplied     bool                     `json:"semedit_arm_restriction_applied"`
	SemeditPrewarmVerify             bool                     `json:"semedit_prewarm_verify"`
	SemeditPrewarmDurationMS         int64                    `json:"semedit_prewarm_duration_ms,omitempty"`
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
	EstimatedCostUSD                 *float64                 `json:"estimated_cost_usd,omitempty"`
	EstimatedCostPricingID           string                   `json:"estimated_cost_pricing_id,omitempty"`
	EstimatedCostRateCardAsOf        string                   `json:"estimated_cost_rate_card_as_of,omitempty"`
	EstimatedCostBasis               string                   `json:"estimated_cost_basis,omitempty"`
	Oracle                           *OracleResult            `json:"oracle"`
	Prompt                           string                   `json:"prompt,omitempty"`
	BeforeState                      string                   `json:"before_state,omitempty"`
	Diff                             string                   `json:"diff,omitempty"`
	ToolsUsed                        []string                 `json:"tools_used,omitempty"`
	ToolCalls                        []ToolCall               `json:"tool_calls,omitempty"`
	InteractiveMode                  string                   `json:"interactive_mode,omitempty"`
	InteractionSteps                 []InteractionStep        `json:"interaction_steps,omitempty"`
	DiagnosticToolCoverage           *DiagnosticToolCoverage  `json:"diagnostic_tool_coverage,omitempty"`
	DiagnosticReflectionSkipped      string                   `json:"diagnostic_reflection_skipped,omitempty"`
	toolObservationState             ToolObservationState     `json:"-"`
	toolObservationReason            string                   `json:"-"`
	SemanticToolReflection           *SemanticToolReflection  `json:"semantic_tool_reflection,omitempty"`
	SemanticIntegrationReflection    *SemanticToolReflection  `json:"semantic_integration_reflection,omitempty"`
	SemanticBatchReflection          *SemanticToolReflection  `json:"semantic_batch_reflection,omitempty"`
	MCPVerified                      bool                     `json:"mcp_verified"`
	Error                            string                   `json:"error,omitempty"`
	CodexExitCode                    *int                     `json:"codex_exit_code,omitempty"`
	CodexStderr                      string                   `json:"codex_stderr,omitempty"`
	OpenCodeExitCode                 *int                     `json:"opencode_exit_code,omitempty"`
	OpenCodeStderr                   string                   `json:"opencode_stderr,omitempty"`
	agentResponse                    string
	processStartedAt                 time.Time
	rawEvents                        []json.RawMessage
	DiagnosticTimeoutReflection      *SemanticToolReflection `json:"diagnostic_timeout_reflection,omitempty"`
}

func (r *RunResult) wallClockStart(fallback time.Time) time.Time {
	if r != nil && !r.processStartedAt.IsZero() {
		return r.processStartedAt
	}
	return fallback
}

func (r *RunResult) measuredWallClock(fallback time.Time) time.Duration {
	if r != nil && r.processStartedAt.IsZero() {
		return 0
	}
	return elapsedTaskTime(r.wallClockStart(fallback))
}

func elapsedTaskTime(start time.Time) time.Duration {
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}

// Runner coordinates execution across evaluation arms and benchmarks.
type Runner struct {
	baseScratchDir                string
	fixtureRoot                   string
	ramFixtureRootConfigured      bool
	moduleCache                   string
	configurationErr              error
	mcpServerInstructionsMode     MCPServerInstructionMode
	semeditArmRestriction         SemeditArmRestriction
	semeditPrewarmVerify          bool
	provenance                    ProvenanceSet
	agyExecutable                 string
	openCodeRelayURL              string
	openCodeModelLimits           *openCodeModelLimits
	openCodeSandboxProfileFactory func(workDir, openCodePath, modelURL string) (string, string, error)
}

// RunnerOption configures one benchmark runner without adding global process state.
type RunnerOption func(*Runner)

// WithMCPServerInstructions selects the server instruction experiment mode for all runs made by a runner.
func WithMCPServerInstructions(mode MCPServerInstructionMode) RunnerOption {
	if mode == "" {
		mode = MCPServerInstructionsPrescriptive
	}
	return func(r *Runner) { r.mcpServerInstructionsMode = mode }
}

func WithSemeditPrewarmVerify(enabled bool) RunnerOption {
	return func(r *Runner) { r.semeditPrewarmVerify = enabled }
}

// WithFixtureRoot puts disposable fixture workspaces on a caller-selected filesystem.
func WithFixtureRoot(root string) RunnerOption {
	return func(r *Runner) {
		r.fixtureRoot = root
		r.ramFixtureRootConfigured = root != ""
	}
}

// WithModuleCache selects the persistent, read-only-offline module cache shared by fixtures.
func WithModuleCache(path string) RunnerOption {
	return func(r *Runner) { r.moduleCache = path }
}

// WithProvenance attaches immutable execution provenance to every benchmark result.
func WithProvenance(provenance ProvenanceSet) RunnerOption {
	return func(r *Runner) { r.provenance = provenance.Clone() }
}

func NewRunner(scratchDir string, options ...RunnerOption) *Runner {
	if scratchDir == "" {
		scratchDir = filepath.Join(".scratch", "benchmarks")
	}
	abs, err := filepath.Abs(scratchDir)
	if err != nil {
		return &Runner{configurationErr: fmt.Errorf("resolve persistent scratch directory: %w", err)}
	}
	scratchDir = abs
	runner := &Runner{
		baseScratchDir:            scratchDir,
		mcpServerInstructionsMode: MCPServerInstructionsPrescriptive,
		semeditArmRestriction:     SemeditArmRestrictWrite,
	}
	for _, option := range options {
		option(runner)
	}
	if runner.fixtureRoot == "" {
		runner.fixtureRoot = runner.baseScratchDir
	} else if absoluteRoot, err := filepath.Abs(runner.fixtureRoot); err != nil {
		runner.configurationErr = fmt.Errorf("resolve RAM fixture root: %w", err)
	} else {
		runner.fixtureRoot = absoluteRoot
	}
	if runner.moduleCache != "" {
		if absoluteCache, err := filepath.Abs(runner.moduleCache); err != nil {
			runner.configurationErr = fmt.Errorf("resolve persistent module cache: %w", err)
		} else {
			canonicalCache, resolveErr := filepath.EvalSymlinks(absoluteCache)
			if resolveErr != nil {
				runner.configurationErr = fmt.Errorf("canonicalize persistent module cache: %w", resolveErr)
			} else {
				runner.moduleCache = canonicalCache
			}
		}
	}
	return runner
}

func (r *Runner) mcpGoArguments(workDir string) ([]string, error) {
	root, err := canonicalFixtureRoot(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve MCP fixture workspace: %w", err)
	}
	args := []string{"mcp", "--profile", "full", "--expected-root", root, "--go-base-dir", filepath.Join(root, ".scratch", "go"), "--metrics-file", benchmarkMCPMetricsPath(workDir)}
	if r.moduleCache != "" {
		args = append(args, "--go-module-cache", r.moduleCache)
	}
	return args, nil
}

func (r *Runner) validateFixtureRoot() error {
	if r.configurationErr != nil {
		return r.configurationErr
	}
	if r.moduleCache != "" {
		if err := validateDurableStoragePath("persistent module cache", r.moduleCache, true); err != nil {
			return err
		}
	}
	if !r.ramFixtureRootConfigured {
		return nil
	}
	if r.moduleCache == "" {
		return fmt.Errorf("RAM fixture root requires a persistent module cache")
	}
	if err := validateDurableStoragePath("persistent benchmark scratch", r.baseScratchDir, true); err != nil {
		return err
	}
	canonicalRoot, err := filepath.EvalSymlinks(r.fixtureRoot)
	if err != nil {
		return fmt.Errorf("canonicalize RAM fixture root: %w", err)
	}
	info, err := os.Stat(canonicalRoot)
	if err != nil {
		return fmt.Errorf("inspect RAM fixture root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("RAM fixture root %q is not a directory", r.fixtureRoot)
	}
	if r.moduleCache != "" {
		canonicalCache, cacheErr := filepath.EvalSymlinks(r.moduleCache)
		if cacheErr != nil {
			return fmt.Errorf("canonicalize persistent module cache: %w", cacheErr)
		}
		relativeCache, relErr := filepath.Rel(canonicalRoot, canonicalCache)
		if relErr != nil {
			return fmt.Errorf("compare RAM fixture root with module cache: %w", relErr)
		}
		if relativeCache == "." || (relativeCache != ".." && !strings.HasPrefix(relativeCache, ".."+string(filepath.Separator))) {
			return fmt.Errorf("persistent module cache %q is inside RAM fixture root", canonicalCache)
		}
	}
	if err := validateTmpfsMount(canonicalRoot); err != nil {
		return fmt.Errorf("RAM fixture root %q must be mounted tmpfs: %w", canonicalRoot, err)
	}
	return nil
}

func (r *Runner) provenanceFor() ProvenanceSet {
	return r.provenance.Clone()
}

func (r *Runner) ExecuteControlVariant(ctx context.Context, task *Task, variant string) (*RunResult, error) {
	if task == nil {
		return nil, fmt.Errorf("execute control variant: task is nil")
	}
	copy := *task
	copy.contextVariant = variant
	return r.ExecuteControl(ctx, &copy)
}

// ExecuteControl runs a benchmark task using deterministic semedit operations directly.
func (r *Runner) ExecuteControl(ctx context.Context, task *Task) (*RunResult, error) {
	if err := r.validateFixtureRoot(); err != nil {
		return nil, err
	}
	if r.moduleCache != "" {
		ctx = gocache.WithModuleCache(ctx, r.moduleCache)
	}
	start := time.Now()
	runID := fmt.Sprintf("run_control_%s_%d", task.Metadata.TaskID, time.Now().UnixNano())
	workDir, workspaceErr := createBenchmarkWorkspace(r.fixtureRoot, runID)
	if workspaceErr != nil {
		return nil, workspaceErr
	}
	defer func() {
		// #nosec G703 -- cleanup of isolated run scratch directory
		_ = os.RemoveAll(workDir)
	}()

	if err := task.ExtractTo(workDir); err != nil {
		return nil, fmt.Errorf("extract task fixture: %w", err)
	}

	modifiedFiles, execErr := executeControlTransformation(ctx, task, workDir)

	res := &RunResult{
		TaskID:                task.Metadata.TaskID,
		ExecutionMode:         "deterministic_ast_control",
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
		return res, fmt.Errorf("oracle evaluation: %w", err)
	}

	res.Oracle = oracleRes
	res.Success = oracleRes.Passed
	return res, nil
}

func executeControlTransformation(ctx context.Context, task *Task, workDir string) ([]string, error) {
	switch task.Metadata.TaskID {
	case "task-01-rename-local", "task-01b-rename-local-large-context":
		return renameControlSymbol(ctx, workDir, "api/server.go", "Server.oldName", "newName", "api/server.go", "api/server.go")
	case "task-02-rename-cross-pkg":
		return renameControlSymbol(ctx, workDir, "auth/token.go", "ValidateToken", "VerifyToken", ".", "auth/token.go", "cmd/main.go")
	case "task-03-rename-interface":
		return renameControlSymbol(ctx, workDir, "store/store.go", "Storage.Save", "Persist", ".", "store/store.go")
	case "task-04-insert-public":
		return insertControlDeclaration(ctx, workDir, "func InitServer() *Server {\n\treturn &Server{Port: 8080}\n}", astedit.Options{Placement: astedit.PlacementBeforeSymbol, TargetSymbol: "internalRun", Visibility: "public"}, "insert InitServer: %w")
	case "task-05-insert-method":
		return insertControlDeclaration(ctx, workDir, "func (s *Server) Stop() {\n\ts.running = false\n}", astedit.Options{Placement: astedit.PlacementAfterSymbol, TargetSymbol: "Server.Start"}, "insert Server.Stop: %w")
	case "task-06-imports-cleanup":
		return organizeControlImports(ctx, workDir)
	case "task-11-mixed-sink-api-migration":
		return applyMixedSinkControlMigration(ctx, workDir)
	default:
		return nil, fmt.Errorf("task %s has no registered control transformation", task.Metadata.TaskID)
	}
}

func renameControlSymbol(ctx context.Context, workDir, file, symbolName, replacement, formatPath string, changedFiles ...string) ([]string, error) {
	resolved, err := symbol.Resolve(workDir, file, symbolName)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", symbolName, err)
	}
	if resolved.Definition == nil || resolved.IsAmbiguous() {
		return nil, fmt.Errorf("resolve %s: expected a unique definition", symbolName)
	}
	definition := resolved.Definition
	if err := golang.Rename(ctx, workDir, definition.File, definition.Line, definition.Column, replacement); err != nil {
		return nil, fmt.Errorf("rename %s: %w", symbolName, err)
	}
	if err := pipeline.Format(ctx, workDir, formatPath); err != nil {
		return changedFiles, fmt.Errorf("format renamed files: %w", err)
	}
	return changedFiles, nil
}

func insertControlDeclaration(ctx context.Context, workDir, source string, options astedit.Options, errorFormat string) ([]string, error) {
	targetPath := filepath.Join(workDir, "api", "server.go")
	if err := astedit.InsertDeclaration(ctx, targetPath, source, options); err != nil {
		return nil, fmt.Errorf(errorFormat, err)
	}
	return []string{"api/server.go"}, nil
}

func organizeControlImports(ctx context.Context, workDir string) ([]string, error) {
	targetFile := filepath.Join(workDir, "api", "server.go")
	if err := pipeline.OrganizeImports(ctx, workDir, targetFile); err != nil {
		return nil, fmt.Errorf("organize imports: %w", err)
	}
	return []string{"api/server.go"}, nil
}

func applyMixedSinkControlMigration(ctx context.Context, workDir string) ([]string, error) {
	normalizePath := filepath.Join(workDir, "audit", "normalize.go")
	if err := astedit.InsertFunction(ctx, normalizePath, "func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}", astedit.FunctionOptions{Placement: astedit.PlacementFileEnd, AutoOrganizeImports: true}); err != nil {
		return nil, fmt.Errorf("insert audit NormalizeKind: %w", err)
	}
	if _, err := astedit.ReplaceBody(ctx, filepath.Join(workDir, "audit", "summary.go"), "Classify", "switch NormalizeKind(kind) {\ncase \"health\", \"probe\":\n\treturn \"control\"\ndefault:\n\treturn \"data\"\n}", astedit.BodyOptions{}); err != nil {
		return nil, fmt.Errorf("replace audit Classify body: %w", err)
	}
	if _, err := astedit.ReplaceBody(ctx, filepath.Join(workDir, "service", "dispatch.go"), "(*Dispatcher).Record", "event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)", astedit.BodyOptions{}); err != nil {
		return nil, fmt.Errorf("replace Dispatcher.Record body: %w", err)
	}
	return []string{"audit/normalize.go", "audit/summary.go", "service/dispatch.go"}, nil
}

// ParseMCPServerInstructionModes parses and canonicalizes a comma-separated mode list.
func ParseMCPServerInstructionModes(raw string) ([]MCPServerInstructionMode, error) {
	parts := strings.Split(raw, ",")
	modes := make([]MCPServerInstructionMode, 0, len(parts))
	seen := make(map[MCPServerInstructionMode]bool, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("MCP instruction mode list contains an empty entry")
		}
		mode, err := ParseMCPServerInstructionMode(part)
		if err != nil {
			return nil, err
		}
		if seen[mode] {
			continue
		}
		seen[mode] = true
		modes = append(modes, mode)
	}
	return modes, nil
}

func diagnosticToolCoverage(expected []string, calls []ToolCall, state ToolObservationState, reason string) *DiagnosticToolCoverage {
	expectedSet := make(map[string]struct{}, len(expected))
	for _, name := range expected {
		if normalized := strings.TrimSpace(semanticToolBaseName(name)); normalized != "" {
			expectedSet[normalized] = struct{}{}
		}
	}
	if len(expectedSet) == 0 {
		return nil
	}

	observedSet := make(map[string]struct{}, len(calls))
	for _, call := range calls {
		if normalized := strings.TrimSpace(semanticToolBaseName(call.Name)); normalized != "" {
			observedSet[normalized] = struct{}{}
		}
	}

	coverage := &DiagnosticToolCoverage{
		ObservationState:   state,
		Complete:           state == ToolObservationComplete,
		CompletenessReason: reason,
		ExpectedSatisfied:  true,
	}
	if state == ToolObservationComplete {
		coverage.Missing = make([]string, 0)
	}
	for name := range expectedSet {
		coverage.Expected = append(coverage.Expected, name)
		if _, found := observedSet[name]; !found {
			coverage.ExpectedSatisfied = false
			if coverage.Complete {
				coverage.Missing = append(coverage.Missing, name)
			}
		}
	}
	for name := range observedSet {
		coverage.Observed = append(coverage.Observed, name)
	}
	sort.Strings(coverage.Expected)
	sort.Strings(coverage.Observed)
	sort.Strings(coverage.Missing)
	return coverage
}

func isSemeditArm(arm ArmType) bool {
	return arm == ArmSemedit || arm == ArmSemeditAgentic || arm == ArmSemeditExplicit
}

func WithOpenCodeRelayURL(url string) RunnerOption {
	return func(r *Runner) { r.openCodeRelayURL = url }
}

// WithOpenCodeModelLimits supplies limits discovered once by the run-owned relay.
func WithOpenCodeModelLimits(limits openCodeModelLimits) RunnerOption {
	return func(r *Runner) { r.openCodeModelLimits = &limits }
}
