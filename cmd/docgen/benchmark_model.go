// Package main keeps benchmark report models and serialized fields together.
package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// BenchTarget identifies the execution harness and model for a benchmark.
type BenchTarget struct {
	Harness string `json:"harness"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
}

func (t BenchTarget) String() string {
	parts := []string{t.Harness}
	if t.Model != "" {
		parts = append(parts, t.Model)
	}
	if t.Effort != "" {
		parts = append(parts, t.Effort)
	}
	return strings.Join(parts, "/")
}

// BenchRunResult captures telemetry for a single trial.
type BenchRunResult struct {
	TaskID                           string                       `json:"task_id"`
	Variant                          string                       `json:"variant,omitempty"`
	PromptVariant                    string                       `json:"prompt_variant,omitempty"`
	MCPServerInstructions            string                       `json:"mcp_server_instructions,omitempty"`
	LegacyMCPInstructions            string                       `json:"mcp_instructions,omitempty"`
	Provenance                       map[string]string            `json:"provenance,omitempty"`
	LegacyClassifiers                map[string]string            `json:"classifiers,omitempty"`
	Target                           BenchTarget                  `json:"target"`
	Arm                              string                       `json:"arm"`
	Success                          bool                         `json:"success"`
	Turns                            int                          `json:"turns"`
	InitialLoadTurns                 int                          `json:"initial_load_turns"`
	MCPLoadTurns                     int                          `json:"mcp_load_turns"`
	InternalTurns                    int                          `json:"internal_turns"`
	ToolCount                        int                          `json:"tool_count"`
	WallClock                        time.Duration                `json:"wall_clock_ms"`
	ProcessStartToFirstEvent         *time.Duration               `json:"process_start_to_first_event_ms,omitempty"`
	FirstEventToFirstToolCall        *time.Duration               `json:"first_event_to_first_tool_call_ms,omitempty"`
	MCPInitializeToFirstSemanticCall *time.Duration               `json:"mcp_initialize_to_first_semantic_call_ms,omitempty"`
	MCPServerStartToInitialize       *time.Duration               `json:"mcp_server_start_to_initialize_ms,omitempty"`
	InitialContextTokens             int                          `json:"initial_context_tokens"`
	PromptTokens                     int                          `json:"prompt_tokens"`
	CachedPromptTokens               int                          `json:"cached_prompt_tokens"`
	UncachedPromptTokens             int                          `json:"uncached_prompt_tokens"`
	OutputTokens                     int                          `json:"output_tokens"`
	ReasoningTokens                  int                          `json:"reasoning_tokens"`
	Oracle                           *BenchOracleResult           `json:"oracle"`
	Prompt                           string                       `json:"prompt,omitempty"`
	BeforeState                      string                       `json:"before_state,omitempty"`
	Diff                             string                       `json:"diff,omitempty"`
	ToolsUsed                        []string                     `json:"tools_used,omitempty"`
	ToolCalls                        []BenchToolCall              `json:"tool_calls,omitempty"`
	InteractionSteps                 []BenchInteractionStep       `json:"interaction_steps,omitempty"`
	SemanticToolReflection           *BenchSemanticToolReflection `json:"semantic_tool_reflection,omitempty"`
	MCPVerified                      bool                         `json:"mcp_verified"`
	Error                            string                       `json:"error,omitempty"`
}

// BenchToolCall captures a tool outcome from a benchmark report without importing the harness package.
type BenchToolCall struct {
	Name             string           `json:"name"`
	Server           string           `json:"server,omitempty"`
	Arguments        json.RawMessage  `json:"arguments,omitempty"`
	TransportStatus  string           `json:"transport_status"`
	FunctionalStatus string           `json:"functional_status"`
	Failure          string           `json:"failure,omitempty"`
	MCPMetrics       *BenchMCPMetrics `json:"mcp_metrics,omitempty"`
}

// BenchInteractionStep preserves the independent outcome and tool-call order from one continued reasoning turn.
type BenchInteractionStep struct {
	Step      int                `json:"step"`
	Turns     int                `json:"turns"`
	ToolCalls []BenchToolCall    `json:"tool_calls,omitempty"`
	Oracle    *BenchOracleResult `json:"oracle,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// BenchSemanticToolReflection records the diagnostic reason an unverified semantic run did not use a semantic edit tool.
type BenchSemanticToolReflection struct {
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

var noSemanticToolsAvailablePattern = regexp.MustCompile(`(?i)\bno\b(?:\W+\w+)*\W+\btools?\b(?:\W+\w+)*\W+\bavailable\b`)

// BenchMCPMetrics records MCP server latency without importing the benchmark harness package.
type BenchMCPMetrics struct {
	SchemaVersion int                            `json:"schema_version"`
	TotalMS       int64                          `json:"total_ms"`
	Phases        map[string]BenchMCPPhaseMetric `json:"phases,omitempty"`
}

// BenchMCPPhaseMetric aggregates an instrumented MCP server phase.
type BenchMCPPhaseMetric struct {
	Count      int   `json:"count"`
	DurationMS int64 `json:"duration_ms"`
}

const benchmarkToolCallCSS = `<style>
table.benchmark-tool-calls {
  width: 100%;
  table-layout: fixed;
}

table.benchmark-tool-calls th:first-child,
table.benchmark-tool-calls td:first-child {
  width: 3rem;
}

table.benchmark-tool-calls td {
  min-width: 0;
}

table.benchmark-tool-calls .benchmark-code-block {
	width: 100%;
	max-width: 32rem;
	margin: 0.5rem 0 0;
	min-width: 0;
	overflow: hidden;
}

table.benchmark-tool-calls .benchmark-code-block pre {
	white-space: pre-wrap;
	overflow-wrap: anywhere;
	word-break: break-word;
	overflow-x: auto;
}

table.benchmark-tool-calls .benchmark-code-block code {
	white-space: inherit;
}

.benchmark-delta-positive {
  color: var(--bs-success, #198754);
  font-weight: 700;
}

.benchmark-delta-negative {
  color: var(--bs-danger, #dc3545);
  font-weight: 700;
}
</style>

`

const benchmarkCodeShortcode = `{{- $lang := .Get "lang" | default "text" -}}
{{- $encoded := .Get "content" -}}
{{- $content := $encoded | base64Decode -}}
<div class="benchmark-code-block">
  <div class="hextra-code-block hx:relative hx:mt-6 hx:first:mt-0 hx:group/code">
    {{- partial "components/codeblock" (dict "lang" $lang "content" $content "options" (dict)) -}}
    {{- if or (eq site.Params.highlight.copy.enable nil) (site.Params.highlight.copy.enable) -}}
      {{- partialCached "components/codeblock-copy-button" (dict "filename" "") "" -}}
    {{- end -}}
  </div>
</div>
`

// BenchOracleResult records oracle outcomes across evaluation levels.
type BenchOracleResult struct {
	Passed       bool          `json:"passed"`
	Level1Policy bool          `json:"level_1_policy"`
	Level2AST    bool          `json:"level_2_ast"`
	Level3Build  bool          `json:"level_3_build"`
	Level4Test   bool          `json:"level_4_test"`
	FailureStage string        `json:"failure_stage,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
}

// BenchComparisonSummary bundles Baseline vs MCP runs for a task and target.
type BenchComparisonSummary struct {
	SelectedRunID         string            `json:"-"`
	TaskID                string            `json:"task_id"`
	PromptVariant         string            `json:"prompt_variant,omitempty"`
	MCPServerInstructions string            `json:"mcp_server_instructions,omitempty"`
	LegacyMCPInstructions string            `json:"mcp_instructions,omitempty"`
	Provenance            map[string]string `json:"provenance,omitempty"`
	LegacyClassifiers     map[string]string `json:"classifiers,omitempty"`
	TxtarPath             string            `json:"txtar_path,omitempty"`
	TxtarProvenance       string            `json:"txtar_provenance,omitempty"`
	Target                BenchTarget       `json:"target"`
	VanillaPrompt         string            `json:"vanilla_prompt,omitempty"`
	MCPPrompt             string            `json:"mcp_prompt,omitempty"`
	VanillaVerifiedPrompt string            `json:"vanilla_verified_prompt,omitempty"`
	MCPVerifiedPrompt     string            `json:"mcp_verified_prompt,omitempty"`
	BeforeState           string            `json:"before_state,omitempty"`
	SmallBaseline         *BenchRunResult   `json:"small_baseline,omitempty"`
	SmallSemedit          *BenchRunResult   `json:"small_semedit,omitempty"`
	SmallVerifiedBaseline *BenchRunResult   `json:"small_verified_baseline,omitempty"`
	SmallVerifiedSemedit  *BenchRunResult   `json:"small_verified_semedit,omitempty"`
	LargeBaseline         *BenchRunResult   `json:"large_baseline,omitempty"`
	LargeSemedit          *BenchRunResult   `json:"large_semedit,omitempty"`
	LargeVerifiedBaseline *BenchRunResult   `json:"large_verified_baseline,omitempty"`
	LargeVerifiedSemedit  *BenchRunResult   `json:"large_verified_semedit,omitempty"`
}

// BenchReport captures an empirical report file.
type BenchReport struct {
	FormatVersion int                       `json:"format_version"`
	DurationUnit  string                    `json:"duration_unit"`
	Timestamp     time.Time                 `json:"timestamp"`
	Runs          []*BenchRunResult         `json:"runs"`
	Comparisons   []*BenchComparisonSummary `json:"comparisons,omitempty"`
}

// benchmarkDocumentationRun keeps one result directory intact so generated pages can expose auditable source observations.
type benchmarkDocumentationRun struct {
	ID          string
	Comparisons []*BenchComparisonSummary
}

// benchmarkDocumentation groups all generated benchmark pages from one immutable result snapshot.
type benchmarkDocumentation struct {
	Index      string
	Aggregates string
	Runs       []benchmarkDocumentationRun
}
