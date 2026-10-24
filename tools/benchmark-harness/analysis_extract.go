// WHY: This file coordinates explicit benchmark inputs and opt-in Codex transcript exports without traversing archived benchmark results.

package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const analysisExtractSchemaVersion = "semedit.benchmark-analysis/v1"

type analysisOracle struct {
	Passed       bool   `json:"passed"`
	Level1Policy bool   `json:"level_1_policy"`
	Level2AST    bool   `json:"level_2_ast"`
	Level3Build  bool   `json:"level_3_build"`
	Level4Test   bool   `json:"level_4_test"`
	FailureStage string `json:"failure_stage,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	DurationMS   int64  `json:"duration_ms"`
}

func analysisOracleFrom(result *OracleResult) *analysisOracle {
	if result == nil {
		return nil
	}
	return &analysisOracle{
		Passed: result.Passed, Level1Policy: result.Level1Policy, Level2AST: result.Level2AST,
		Level3Build: result.Level3Build, Level4Test: result.Level4Test, FailureStage: result.FailureStage,
		ErrorMessage: result.ErrorMessage, DurationMS: result.Duration.Milliseconds(),
	}
}

type analysisInputFlags []string

func (f *analysisInputFlags) String() string { return strings.Join(*f, ",") }

func (f *analysisInputFlags) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("input path cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

type analysisRunContext struct {
	JobID                       string                   `json:"job_id,omitempty"`
	PairID                      string                   `json:"comparison_pair_id,omitempty"`
	TaskID                      string                   `json:"task_id"`
	Target                      Target                   `json:"target"`
	Arm                         ArmType                  `json:"arm"`
	Variant                     string                   `json:"variant,omitempty"`
	PromptVariant               string                   `json:"prompt_variant,omitempty"`
	Repeat                      int                      `json:"repeat,omitempty"`
	Skill                       bool                     `json:"semedit_skill"`
	Batching                    bool                     `json:"semedit_batching"`
	MCPServerInstructions       MCPServerInstructionMode `json:"mcp_server_instructions,omitempty"`
	ArmRestriction              SemeditArmRestriction    `json:"semedit_arm_restrict,omitempty"`
	ArmRestrictionApplied       bool                     `json:"semedit_arm_restriction_applied"`
	Success                     bool                     `json:"success"`
	MCPVerified                 bool                     `json:"mcp_verified"`
	ExecutionMode               string                   `json:"execution_mode,omitempty"`
	Complexity                  *ComplexityProfile       `json:"complexity,omitempty"`
	SemeditPrewarmVerify        bool                     `json:"semedit_prewarm_verify"`
	SemeditPrewarmDurationMS    int64                    `json:"semedit_prewarm_duration_ms,omitempty"`
	WallClockMS                 int64                    `json:"wall_clock_ms"`
	ToolWaitMS                  *int64                   `json:"tool_wait_ms,omitempty"`
	SemanticToolWaitMS          *int64                   `json:"semantic_tool_wait_ms,omitempty"`
	OtherToolWaitMS             *int64                   `json:"other_tool_wait_ms,omitempty"`
	ToolOccupiedMS              *int64                   `json:"tool_occupied_ms,omitempty"`
	NonToolWallClockMS          *int64                   `json:"non_tool_wall_clock_ms,omitempty"`
	ToolCallsTimed              int                      `json:"tool_calls_timed,omitempty"`
	ToolTimingCoverage          *float64                 `json:"tool_timing_coverage,omitempty"`
	Turns                       int                      `json:"turns"`
	InitialLoadTurns            int                      `json:"initial_load_turns"`
	MCPLoadTurns                int                      `json:"mcp_load_turns"`
	InternalTurns               int                      `json:"internal_turns"`
	ToolCount                   int                      `json:"tool_count"`
	FailedToolCallCount         int                      `json:"failed_tool_call_count"`
	FailedSemanticToolCallCount int                      `json:"failed_semantic_tool_call_count"`
	InitialContextTokens        int                      `json:"initial_context_tokens"`
	PromptTokens                int                      `json:"prompt_tokens"`
	CachedPromptTokens          int                      `json:"cached_prompt_tokens"`
	UncachedPromptTokens        int                      `json:"uncached_prompt_tokens"`
	OutputTokens                int                      `json:"output_tokens"`
	ReasoningTokens             int                      `json:"reasoning_tokens"`
	EstimatedCostUSD            *float64                 `json:"estimated_cost_usd,omitempty"`
	EstimatedCostPricingID      string                   `json:"estimated_cost_pricing_id,omitempty"`
	EstimatedCostRateCardAsOf   string                   `json:"estimated_cost_rate_card_as_of,omitempty"`
	EstimatedCostBasis          string                   `json:"estimated_cost_basis,omitempty"`
	ProcessStartToFirstEventMS  *int64                   `json:"process_start_to_first_event_ms,omitempty"`
	FirstEventToFirstToolCallMS *int64                   `json:"first_event_to_first_tool_call_ms,omitempty"`
	MCPServerStartToInitMS      *int64                   `json:"mcp_server_start_to_initialize_ms,omitempty"`
	MCPInitToFirstSemanticMS    *int64                   `json:"mcp_initialize_to_first_semantic_call_ms,omitempty"`
	Oracle                      *analysisOracle          `json:"oracle,omitempty"`
	DiagnosticToolCoverage      *DiagnosticToolCoverage  `json:"diagnostic_tool_coverage,omitempty"`
	DiagnosticReflectionSkipped string                   `json:"diagnostic_reflection_skipped,omitempty"`
	Prompt                      string                   `json:"prompt,omitempty"`
	BeforeState                 string                   `json:"before_state,omitempty"`
	Diff                        string                   `json:"diff,omitempty"`
	ToolsUsed                   []string                 `json:"tools_used,omitempty"`
	InteractiveMode             string                   `json:"interactive_mode,omitempty"`
	CodexExitCode               *int                     `json:"codex_exit_code,omitempty"`
	CodexStderr                 string                   `json:"codex_stderr,omitempty"`
	OpenCodeExitCode            *int                     `json:"opencode_exit_code,omitempty"`
	OpenCodeStderr              string                   `json:"opencode_stderr,omitempty"`
	Error                       string                   `json:"error,omitempty"`
	Provenance                  ProvenanceSet            `json:"provenance,omitempty"`
}

type analysisEvent struct {
	SchemaVersion    string                   `json:"schema_version"`
	Type             string                   `json:"type"`
	Source           string                   `json:"source"`
	OccurredAt       string                   `json:"occurred_at,omitempty"`
	RunID            string                   `json:"run_id"`
	TaskID           string                   `json:"task_id"`
	JobID            string                   `json:"job_id,omitempty"`
	Arm              ArmType                  `json:"arm,omitempty"`
	Job              *analysisJobContext      `json:"job,omitempty"`
	Run              *analysisRunContext      `json:"run,omitempty"`
	Index            *int                     `json:"index,omitempty"`
	Step             int                      `json:"step,omitempty"`
	Reflection       string                   `json:"reflection,omitempty"`
	ReflectionPrompt string                   `json:"reflection_prompt,omitempty"`
	ToolCall         *ToolCall                `json:"tool_call,omitempty"`
	ToolDurationMS   *int64                   `json:"tool_duration_ms,omitempty"`
	ToolTimingSource string                   `json:"tool_timing_source,omitempty"`
	toolStartMS      int64                    `json:"-"`
	toolFinishMS     int64                    `json:"-"`
	Response         string                   `json:"response,omitempty"`
	Failure          string                   `json:"failure,omitempty"`
	Interaction      *analysisInteraction     `json:"interaction,omitempty"`
	Transcript       *analysisTranscriptTurn  `json:"transcript,omitempty"`
	AdapterStage     *adapterStageResult      `json:"adapter_stage,omitempty"`
	MCPMetric        json.RawMessage          `json:"mcp_metric,omitempty"`
	SidecarStatus    string                   `json:"sidecar_status,omitempty"`
	SidecarError     string                   `json:"sidecar_error,omitempty"`
	CodexChat        *analysisCodexChatRecord `json:"codex_chat,omitempty"`
	Pagination       *analysisPagination      `json:"pagination,omitempty"`
}

type analysisJobContext struct {
	ID                    string                   `json:"id"`
	PairID                string                   `json:"pair_id,omitempty"`
	TaskID                string                   `json:"task_id"`
	FixturePath           string                   `json:"fixture_path"`
	Context               string                   `json:"context"`
	PromptVariant         string                   `json:"prompt_variant,omitempty"`
	Target                Target                   `json:"target"`
	Arm                   ArmType                  `json:"arm"`
	Repeat                int                      `json:"repeat"`
	Policy                SemeditArmRestriction    `json:"policy,omitempty"`
	Followups             []string                 `json:"followups,omitempty"`
	StagedFollowups       []StagedFollowup         `json:"staged_followups,omitempty"`
	MCPServerInstructions MCPServerInstructionMode `json:"mcp_server_instructions,omitempty"`
}

func analysisJobFrom(job *jobCheckpointMetadata) *analysisJobContext {
	if job == nil {
		return nil
	}
	return &analysisJobContext{
		ID: job.ID, PairID: job.PairID, TaskID: job.TaskID, FixturePath: job.FixturePath, Context: job.Context,
		PromptVariant: job.PromptVariant, Target: job.Target, Arm: job.Arm, Repeat: job.Repeat, Policy: job.Policy,
		Followups: job.Followups, StagedFollowups: job.StagedFollowups, MCPServerInstructions: job.MCPServerInstructions,
	}
}

type analysisInteraction struct {
	TotalEdits           int             `json:"total_edits,omitempty"`
	Prompt               string          `json:"prompt,omitempty"`
	WallClockMS          int64           `json:"wall_clock_ms"`
	Turns                int             `json:"turns"`
	PromptTokens         int             `json:"prompt_tokens"`
	CachedPromptTokens   int             `json:"cached_prompt_tokens"`
	UncachedPromptTokens int             `json:"uncached_prompt_tokens"`
	OutputTokens         int             `json:"output_tokens"`
	ReasoningTokens      int             `json:"reasoning_tokens"`
	Oracle               *analysisOracle `json:"oracle,omitempty"`
	Error                string          `json:"error,omitempty"`
	ToolCallIndices      []int           `json:"tool_call_indices,omitempty"`
}

type analysisTranscriptTurn struct {
	Number          int                  `json:"number"`
	Classification  string               `json:"classification"`
	Prompt          string               `json:"prompt,omitempty"`
	StartedAt       string               `json:"started_at,omitempty"`
	FinishedAt      string               `json:"finished_at,omitempty"`
	WallClockNS     int64                `json:"wall_clock_ns"`
	ProviderError   string               `json:"provider_error,omitempty"`
	ProviderExit    string               `json:"provider_exit,omitempty"`
	Response        string               `json:"response,omitempty"`
	ToolCalls       []ToolCall           `json:"tool_calls,omitempty"`
	ToolState       ToolObservationState `json:"tool_observation_state,omitempty"`
	ToolReason      string               `json:"tool_observation_reason,omitempty"`
	PromptTokens    int                  `json:"prompt_tokens,omitempty"`
	OutputTokens    int                  `json:"output_tokens,omitempty"`
	ReasoningTokens int                  `json:"reasoning_tokens,omitempty"`
	RawEvents       []json.RawMessage    `json:"raw_events,omitempty"`
	ToolCallIndices []int                `json:"tool_call_indices,omitempty"`
}

type analysisRunSource struct {
	source    string
	timestamp time.Time
	job       *jobCheckpointMetadata
	run       *RunResult
}

func runAnalysisCLI(args []string) int {
	options, status := parseAnalysisCLIOptions(args)
	if status != 0 {
		return status
	}
	var sources []analysisRunSource
	for _, input := range options.inputs {
		loaded, loadErr := loadAnalysisInput(input)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "analyze input %q: %v\n", input, loadErr)
			return 1
		}
		sources = append(sources, loaded...)
	}
	events, err := buildAnalysisEvents(sources, options.includeSidecars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build analysis events: %v\n", err)
		return 1
	}
	codexFiles, err := prepareCodexTranscriptInputs(options.codexHome, options.zstdBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare Codex transcripts: %v\n", err)
		return 1
	}
	if err := writeAnalysisOutput(options.output, events, options.codexHome, codexFiles, options.zstdBin, options.pageLimit, options.cursor, options.since); err != nil {
		fmt.Fprintf(os.Stderr, "write analysis output: %v\n", err)
		return 1
	}
	return 0
}

type analysisCLIOptions struct {
	inputs          analysisInputFlags
	output          string
	includeSidecars bool
	codexHome       string
	zstdBin         string
	pageLimit       int
	since           *time.Time
	cursor          *analysisPageCursor
}

func parseAnalysisCLIOptions(args []string) (analysisCLIOptions, int) {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var options analysisCLIOptions
	var cursorToken string
	var sinceText string
	fs.Var(&options.inputs, "input", "Report/checkpoint JSON file or run directory; repeat to combine inputs")
	fs.StringVar(&options.output, "out", "", "Output JSONL file (default: stdout)")
	fs.BoolVar(&options.includeSidecars, "include-sidecars", false, "Include transcript, adapter-stage, and MCP-metrics sidecars referenced by each result")
	fs.StringVar(&options.codexHome, "codex-home", "", "Also stream interactive chats from this Codex home (sessions/ and archived_sessions/ only)")
	fs.StringVar(&options.zstdBin, "zstd-bin", "zstd", "zstd executable used for compressed Codex transcripts")
	fs.IntVar(&options.pageLimit, "limit", 0, "Maximum analysis records per page; continue with --cursor from page_info")
	fs.StringVar(&cursorToken, "cursor", "", "Opaque continuation token from a previous paginated export")
	fs.StringVar(&sinceText, "since", "", "Include only data timestamped at or after this RFC3339 instant")
	if err := fs.Parse(args); err != nil {
		return options, 2
	}
	if (len(options.inputs) == 0 && options.codexHome == "") || fs.NArg() != 0 || options.pageLimit < 0 || (cursorToken != "" && options.pageLimit == 0) {
		fmt.Fprintln(os.Stderr, "usage: benchmark-harness analyze [--input report.json] [--input run-directory] [--codex-home ~/.codex] [--since RFC3339] [--include-sidecars] [--limit N [--cursor TOKEN]] [--out analysis.jsonl]")
		return options, 2
	}
	cutoff, err := parseAnalysisCutoff(sinceText)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return options, 2
	}
	var since *time.Time
	if sinceText != "" {
		since = &cutoff
	}
	parsedCursor, cursorErr := parseAnalysisCursor(cursorToken, options.codexHome)
	if cursorErr != nil {
		fmt.Fprintln(os.Stderr, cursorErr)
		return options, 2
	}
	var cursor *analysisPageCursor
	if cursorToken != "" {
		cursor = &parsedCursor
	}
	options.since, options.cursor = since, cursor
	return options, 0
}

func parseAnalysisCutoff(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since timestamp %q: use RFC3339, for example 2026-10-19T12:00:00Z", value)
	}
	return parsed, nil
}

func parseAnalysisCursor(token, codexHome string) (analysisPageCursor, error) {
	if token == "" {
		return analysisPageCursor{}, nil
	}
	cursor, err := decodeAnalysisPageCursor(token)
	if err != nil {
		return analysisPageCursor{}, fmt.Errorf("decode page cursor: %w", err)
	}
	if cursor.TranscriptFile != "" && codexHome == "" {
		return analysisPageCursor{}, errors.New("page cursor points into Codex transcripts; pass the same --codex-home")
	}
	return *cursor, nil
}

func loadAnalysisInput(path string) ([]analysisRunSource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect input: %w", err)
	}
	if !info.IsDir() {
		return loadAnalysisFile(path)
	}
	manifest, err := os.Stat(filepath.Join(path, "manifest.json"))
	if err != nil || !manifest.Mode().IsRegular() {
		if err == nil {
			err = errors.New("manifest is not a regular file")
		}
		return nil, fmt.Errorf("input directory must be a harness run directory with manifest.json: %w", err)
	}
	jobDir := filepath.Join(path, "jobs")
	entries, err := os.ReadDir(jobDir)
	if err != nil {
		return nil, fmt.Errorf("read run job checkpoints: %w", err)
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			paths = append(paths, filepath.Join(jobDir, entry.Name()))
		}
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		return nil, errors.New("run directory contains no job checkpoint JSON files")
	}
	var sources []analysisRunSource
	for _, checkpoint := range paths {
		loaded, err := loadAnalysisFile(checkpoint)
		if err != nil {
			return nil, err
		}
		sources = append(sources, loaded...)
	}
	return sources, nil
}

func loadAnalysisFile(path string) ([]analysisRunSource, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- inputs are explicit CLI paths or direct children of an explicitly selected run directory.
	if err != nil {
		return nil, fmt.Errorf("read JSON: %w", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	if rawReport, ok := envelope["report"]; ok {
		var checkpoint jobCheckpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			return nil, fmt.Errorf("decode job checkpoint: %w", err)
		}
		if checkpoint.Report == nil || len(checkpoint.Report.Runs) == 0 {
			return nil, errors.New("job checkpoint has no report runs")
		}
		if len(rawReport) == 0 {
			return nil, errors.New("job checkpoint contains an empty report")
		}
		var sources []analysisRunSource
		for _, run := range checkpoint.Report.Runs {
			if run == nil {
				return nil, errors.New("job checkpoint contains a nil run")
			}
			job := checkpoint.Job
			sources = append(sources, analysisRunSource{source: path, timestamp: checkpoint.Report.Timestamp, job: &job, run: run})
		}
		return sources, nil
	}
	var report BenchmarkReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("decode benchmark report: %w", err)
	}
	if len(report.Runs) == 0 {
		return nil, errors.New("benchmark report has no runs")
	}
	sources := make([]analysisRunSource, 0, len(report.Runs))
	for _, run := range report.Runs {
		if run == nil {
			return nil, errors.New("benchmark report contains a nil run")
		}
		sources = append(sources, analysisRunSource{source: path, timestamp: report.Timestamp, run: run})
	}
	return sources, nil
}

func buildAnalysisEvents(sources []analysisRunSource, includeSidecars bool) ([]analysisEvent, error) {
	var events []analysisEvent
	for sourceIndex, source := range sources {
		runEvents, err := buildAnalysisRunEvents(source, sourceIndex, includeSidecars)
		if err != nil {
			return nil, err
		}
		events = append(events, runEvents...)
	}
	return events, nil
}

func buildAnalysisRunEvents(source analysisRunSource, sourceIndex int, includeSidecars bool) ([]analysisEvent, error) {
	if source.run == nil {
		return nil, fmt.Errorf("analysis source %q contains a nil run", source.source)
	}
	run := source.run
	context := analysisContext(run)
	identity := analysisEventIdentity(source, context, sourceIndex)
	runEvent := identity
	runEvent.Type, runEvent.Job, runEvent.Run = "run", analysisJobFrom(source.job), &context
	events := []analysisEvent{runEvent}
	events = append(events, buildAnalysisCallAndStepEvents(run, identity)...)
	events = append(events, buildAnalysisReflectionEvents(run, identity)...)
	if includeSidecars {
		sidecarEvents, err := extractAnalysisSidecars(source, identity)
		if err != nil {
			return nil, err
		}
		events = append(events, sidecarEvents...)
		enrichAnalysisToolTimings(events)
	}
	return events, nil
}

type providerToolTiming struct {
	ID         string
	Tool       string
	Command    string
	StartMS    int64
	FinishMS   int64
	DurationMS int64
}

func enrichAnalysisToolTimings(events []analysisEvent) {
	callEvents := make(map[int]*analysisEvent)
	var runEvent *analysisEvent
	for index := range events {
		event := &events[index]
		if event.Type == "run" && event.Run != nil {
			runEvent = event
		}
		if event.Type == "tool_call" && event.Index != nil {
			callEvents[*event.Index] = event
		}
	}
	for index := range events {
		event := &events[index]
		if event.Type != "transcript_turn" || event.Transcript == nil {
			continue
		}
		timings := providerToolTimings(event.Transcript.RawEvents)
		assignTranscriptToolTimings(event.Transcript, timings, callEvents)
	}
	if runEvent != nil && runEvent.Run != nil {
		applyRunToolTimings(runEvent.Run, callEvents)
	}
}

func providerToolTimings(rawEvents []json.RawMessage) []providerToolTiming {
	type rawEvent struct {
		Method string `json:"method"`
		Params struct {
			Item struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Tool       string `json:"tool"`
				Command    string `json:"command"`
				DurationMS *int64 `json:"durationMs"`
			} `json:"item"`
			StartedAtMS   int64 `json:"startedAtMs"`
			CompletedAtMS int64 `json:"completedAtMs"`
		} `json:"params"`
		EmittedAtMS int64 `json:"emittedAtMs"`
	}
	type startedTool struct {
		Tool providerToolTiming
		Type string
	}
	starts := make(map[string]startedTool)
	var timings []providerToolTiming
	for _, raw := range rawEvents {
		var event rawEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			continue
		}
		item := event.Params.Item
		if item.ID == "" || (item.Type != "commandExecution" && item.Type != "mcpToolCall") {
			continue
		}
		switch event.Method {
		case "item/started":
			started := event.Params.StartedAtMS
			if started == 0 {
				started = event.EmittedAtMS
			}
			starts[item.ID] = startedTool{Type: item.Type, Tool: providerToolTiming{ID: item.ID, Tool: item.Tool, Command: item.Command, StartMS: started}}
		case "item/completed":
			start, ok := starts[item.ID]
			if !ok || start.Type != item.Type {
				continue
			}
			finished := event.Params.CompletedAtMS
			if finished == 0 {
				finished = event.EmittedAtMS
			}
			if finished < start.Tool.StartMS {
				continue
			}
			timing := start.Tool
			timing.FinishMS = finished
			timing.DurationMS = finished - timing.StartMS
			if timing.DurationMS == 0 && item.DurationMS != nil && *item.DurationMS > 0 {
				timing.DurationMS = *item.DurationMS
				timing.FinishMS = timing.StartMS + timing.DurationMS
			}
			timings = append(timings, timing)
			delete(starts, item.ID)
		}
	}
	slices.SortFunc(timings, func(a, b providerToolTiming) int { return cmp.Compare(a.StartMS, b.StartMS) })
	return timings
}

func assignTranscriptToolTimings(transcript *analysisTranscriptTurn, timings []providerToolTiming, callEvents map[int]*analysisEvent) {
	if len(transcript.ToolCalls) == 0 || len(transcript.ToolCalls) != len(transcript.ToolCallIndices) || len(timings) == 0 {
		return
	}
	used := make([]bool, len(transcript.ToolCalls))
	matched := 0
	assign := func(timing providerToolTiming, callIndex int) {
		if callIndex < 0 || callIndex >= len(transcript.ToolCalls) || used[callIndex] {
			return
		}
		used[callIndex] = true
		if event := callEvents[transcript.ToolCallIndices[callIndex]]; event != nil {
			matched++
			duration := timing.DurationMS
			event.ToolDurationMS = &duration
			event.ToolTimingSource = "provider_item_timestamps"
			event.toolStartMS = timing.StartMS
			event.toolFinishMS = timing.FinishMS
		}
	}
	for _, timing := range timings {
		name := timing.Tool
		if name == "" {
			name = timing.Command
		}
		for callIndex, call := range transcript.ToolCalls {
			if !used[callIndex] && name != "" && call.Name == name {
				assign(timing, callIndex)
				break
			}
		}
	}
	if matched == 0 && len(timings) == len(transcript.ToolCalls) {
		for timingIndex, timing := range timings {
			assign(timing, timingIndex)
		}
	}
}

func applyRunToolTimings(run *analysisRunContext, callEvents map[int]*analysisEvent) {
	var intervals [][2]int64
	var totalWait int64
	var semanticWait, otherWait int64
	var timed, total int
	for _, event := range callEvents {
		total++
		if event.ToolDurationMS == nil {
			continue
		}
		timed++
		totalWait += *event.ToolDurationMS
		if event.ToolCall != nil && isSemanticTool(event.ToolCall.Name, event.ToolCall.Server) {
			semanticWait += *event.ToolDurationMS
		} else {
			otherWait += *event.ToolDurationMS
		}
	}
	if total == 0 {
		return
	}
	run.ToolCallsTimed = timed
	coverage := float64(timed) / float64(total)
	run.ToolTimingCoverage = &coverage
	if timed != total {
		return
	}
	for _, event := range callEvents {
		if event.ToolDurationMS != nil {
			intervals = append(intervals, [2]int64{event.toolStartMS, event.toolFinishMS})
		}
	}
	occupied := unionDuration(intervals)
	nonTool := max(int64(0), run.WallClockMS-occupied)
	run.ToolWaitMS = &totalWait
	run.SemanticToolWaitMS = &semanticWait
	run.OtherToolWaitMS = &otherWait
	run.ToolOccupiedMS = &occupied
	run.NonToolWallClockMS = &nonTool
}

func unionDuration(intervals [][2]int64) int64 {
	if len(intervals) == 0 {
		return 0
	}
	slices.SortFunc(intervals, func(a, b [2]int64) int { return cmp.Compare(a[0], b[0]) })
	start, end := intervals[0][0], intervals[0][1]
	var total int64
	for _, interval := range intervals[1:] {
		if interval[0] <= end {
			end = max(end, interval[1])
			continue
		}
		total += end - start
		start, end = interval[0], interval[1]
	}
	return total + end - start
}

func analysisEventIdentity(source analysisRunSource, run analysisRunContext, sourceIndex int) analysisEvent {
	runID := run.JobID
	if runID == "" && source.job != nil {
		runID = source.job.ID
	}
	if runID == "" {
		runID = fmt.Sprintf("%s#%d", source.source, sourceIndex)
	}
	return analysisEvent{SchemaVersion: analysisExtractSchemaVersion, Source: source.source, OccurredAt: formatAnalysisTime(source.timestamp), RunID: runID, TaskID: run.TaskID, JobID: run.JobID, Arm: run.Arm}
}

func buildAnalysisCallAndStepEvents(run *RunResult, identity analysisEvent) []analysisEvent {
	stepForCall, stepCallIndices := indexInteractionToolCalls(run.ToolCalls, run.InteractionSteps)
	events := make([]analysisEvent, 0, len(run.ToolCalls)+len(run.InteractionSteps))
	for index := range run.ToolCalls {
		call := run.ToolCalls[index]
		event := identity
		event.Type, event.Index, event.Step, event.ToolCall = "tool_call", new(index), stepForCall[index], &call
		events = append(events, event)
	}
	for stepIndex, step := range run.InteractionSteps {
		interaction := &analysisInteraction{
			TotalEdits: step.TotalEdits, Prompt: step.Prompt, WallClockMS: step.WallClock.Milliseconds(), Turns: step.Turns,
			PromptTokens: step.PromptTokens, CachedPromptTokens: step.CachedPromptTokens, UncachedPromptTokens: step.UncachedPromptTokens,
			OutputTokens: step.OutputTokens, ReasoningTokens: step.ReasoningTokens, Oracle: analysisOracleFrom(step.Oracle), Error: step.Error,
			ToolCallIndices: stepCallIndices[stepIndex],
		}
		event := identity
		event.Type, event.Step, event.Interaction = "interaction_step", step.Step, interaction
		events = append(events, event)
	}
	return events
}

func indexInteractionToolCalls(calls []ToolCall, steps []InteractionStep) ([]int, [][]int) {
	stepForCall := make([]int, len(calls))
	stepCallIndices := make([][]int, len(steps))
	callOffset := 0
	for stepIndex, step := range steps {
		for _, stepCall := range step.ToolCalls {
			for callIndex := callOffset; callIndex < len(calls); callIndex++ {
				if !equivalentToolCall(calls[callIndex], stepCall) {
					continue
				}
				stepForCall[callIndex] = step.Step
				stepCallIndices[stepIndex] = append(stepCallIndices[stepIndex], callIndex)
				callOffset = callIndex + 1
				break
			}
		}
	}
	return stepForCall, stepCallIndices
}

func buildAnalysisReflectionEvents(run *RunResult, identity analysisEvent) []analysisEvent {
	var events []analysisEvent
	for _, item := range []struct {
		kind       string
		reflection *SemanticToolReflection
	}{
		{"semantic_tool", run.SemanticToolReflection},
		{"semantic_batch", run.SemanticBatchReflection},
		{"semantic_integration", run.SemanticIntegrationReflection},
		{"diagnostic_timeout", run.DiagnosticTimeoutReflection},
	} {
		if item.reflection == nil {
			continue
		}
		event := identity
		event.Type, event.Reflection, event.ReflectionPrompt, event.Response, event.Failure = "reflection", item.kind, item.reflection.Prompt, item.reflection.Response, item.reflection.Error
		events = append(events, event)
	}
	return events
}

func analysisContext(run *RunResult) analysisRunContext {
	return analysisRunContext{
		JobID: run.JobID, PairID: run.ComparisonPairID, TaskID: run.TaskID, Target: run.Target, Arm: run.Arm,
		Variant: run.Variant, PromptVariant: run.PromptVariant, Repeat: run.Repeat, Skill: run.SemeditSkill, Batching: run.SemeditBatching,
		MCPServerInstructions: run.MCPServerInstructions, ArmRestriction: run.SemeditArmRestrict, ArmRestrictionApplied: run.SemeditArmRestrictionApplied,
		Success: run.Success, MCPVerified: run.MCPVerified, ExecutionMode: run.ExecutionMode, Complexity: run.Complexity,
		SemeditPrewarmVerify: run.SemeditPrewarmVerify, SemeditPrewarmDurationMS: run.SemeditPrewarmDurationMS,
		WallClockMS: run.WallClock.Milliseconds(), Turns: run.Turns,
		InitialLoadTurns: run.InitialLoadTurns, MCPLoadTurns: run.MCPLoadTurns, InternalTurns: run.InternalTurns,
		ToolCount: run.ToolCount, FailedToolCallCount: run.FailedToolCallCount, FailedSemanticToolCallCount: run.FailedSemanticToolCallCount,
		InitialContextTokens: run.InitialContextTokens, PromptTokens: run.PromptTokens, CachedPromptTokens: run.CachedPromptTokens,
		UncachedPromptTokens: run.UncachedPromptTokens, OutputTokens: run.OutputTokens, ReasoningTokens: run.ReasoningTokens,
		EstimatedCostUSD: run.EstimatedCostUSD, EstimatedCostPricingID: run.EstimatedCostPricingID,
		EstimatedCostRateCardAsOf: run.EstimatedCostRateCardAsOf, EstimatedCostBasis: run.EstimatedCostBasis,
		ProcessStartToFirstEventMS: durationMillis(run.ProcessStartToFirstEvent), FirstEventToFirstToolCallMS: durationMillis(run.FirstEventToFirstToolCall),
		MCPServerStartToInitMS: durationMillis(run.MCPServerStartToInitialize), MCPInitToFirstSemanticMS: durationMillis(run.MCPInitializeToFirstSemanticCall),
		Oracle: analysisOracleFrom(run.Oracle), DiagnosticToolCoverage: run.DiagnosticToolCoverage, DiagnosticReflectionSkipped: run.DiagnosticReflectionSkipped,
		Prompt: run.Prompt, BeforeState: run.BeforeState, Diff: run.Diff, ToolsUsed: run.ToolsUsed, InteractiveMode: run.InteractiveMode,
		CodexExitCode: run.CodexExitCode, CodexStderr: run.CodexStderr, OpenCodeExitCode: run.OpenCodeExitCode, OpenCodeStderr: run.OpenCodeStderr,
		Error: run.Error, Provenance: run.Provenance,
	}
}

func durationMillis(duration *time.Duration) *int64 {
	if duration == nil {
		return nil
	}
	value := duration.Milliseconds()
	return &value
}

func extractAnalysisSidecars(source analysisRunSource, identity analysisEvent) ([]analysisEvent, error) {
	if source.run.Provenance == nil {
		return nil, nil
	}
	var events []analysisEvent
	if path := source.run.Provenance["session_transcript"]; path != "" {
		data, err := os.ReadFile(path) // #nosec G304 -- explicit opt-in to sidecars referenced by the selected result.
		if err != nil {
			events = append(events, unavailableSidecar(identity, "session_transcript", err))
		} else {
			var turns []sessionTurn
			if err := json.Unmarshal(data, &turns); err != nil {
				return nil, fmt.Errorf("decode session transcript %q: %w", path, err)
			}
			turnCallIndices := indexTranscriptToolCalls(source.run.ToolCalls, turns)
			for turnIndex, turn := range turns {
				observation := turn.Observation
				transcript := &analysisTranscriptTurn{
					Number: turn.Number, Classification: turn.Classification, Prompt: turn.Prompt, StartedAt: formatAnalysisTime(turn.StartedAt),
					FinishedAt: formatAnalysisTime(turn.FinishedAt), WallClockNS: int64(turn.WallClock), ProviderError: turn.ProviderError,
					ProviderExit: turn.ProviderExit, Response: observation.Response, ToolCalls: observation.ToolCalls,
					ToolState: observation.ToolObservationState, ToolReason: observation.ToolObservationReason,
					PromptTokens: observation.PromptTokens, OutputTokens: observation.OutputTokens, ReasoningTokens: observation.ReasoningTokens,
					RawEvents: observation.RawEvents, ToolCallIndices: turnCallIndices[turnIndex],
				}
				event := identity
				event.Type, event.Step, event.Transcript = "transcript_turn", turn.Number, transcript
				events = append(events, event)
			}
		}
	}
	if path := source.run.Provenance["adapter_stage_transcript"]; path != "" {
		data, err := os.ReadFile(path) // #nosec G304 -- explicit opt-in to sidecars referenced by the selected result.
		if err != nil {
			events = append(events, unavailableSidecar(identity, "adapter_stage_transcript", err))
		} else {
			var stages []adapterStageResult
			if err := json.Unmarshal(data, &stages); err != nil {
				return nil, fmt.Errorf("decode adapter-stage transcript %q: %w", path, err)
			}
			for index := range stages {
				stage := stages[index]
				event := identity
				event.Type, event.Index, event.AdapterStage = "adapter_stage", new(index), &stage
				events = append(events, event)
			}
		}
	}
	if path := source.run.Provenance["mcp_metrics_file"]; path != "" {
		data, err := os.ReadFile(path) // #nosec G304 -- explicit opt-in to sidecars referenced by the selected result.
		if err != nil {
			events = append(events, unavailableSidecar(identity, "mcp_metrics_file", err))
		} else {
			var records []json.RawMessage
			if err := json.Unmarshal(data, &records); err != nil {
				return nil, fmt.Errorf("decode MCP metrics sidecar %q: %w", path, err)
			}
			for index, record := range records {
				event := identity
				event.Type, event.Index, event.MCPMetric = "mcp_metric", new(index), record
				events = append(events, event)
			}
		}
	}
	return events, nil
}

func indexTranscriptToolCalls(calls []ToolCall, turns []sessionTurn) [][]int {
	indices := make([][]int, len(turns))
	callOffset := 0
	for turnIndex, turn := range turns {
		if strings.Contains(strings.ToLower(turn.Classification), "diagnostic") {
			continue
		}
		for _, turnCall := range turn.Observation.ToolCalls {
			for callIndex := callOffset; callIndex < len(calls); callIndex++ {
				if !equivalentToolCall(calls[callIndex], turnCall) {
					continue
				}
				indices[turnIndex] = append(indices[turnIndex], callIndex)
				callOffset = callIndex + 1
				break
			}
		}
	}
	return indices
}

func equivalentToolCall(left, right ToolCall) bool {
	if left.Name != right.Name || (left.Server != "" && right.Server != "" && left.Server != right.Server) {
		return false
	}
	return canonicalToolArguments(left.Arguments) == canonicalToolArguments(right.Arguments)
}

func canonicalToolArguments(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return string(raw)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return string(raw)
	}
	return string(encoded)
}

func unavailableSidecar(identity analysisEvent, name string, err error) analysisEvent {
	identity.Type, identity.SidecarStatus, identity.SidecarError = "sidecar_unavailable", name, err.Error()
	return identity
}

func formatAnalysisTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

type analysisCodexChatRecord struct {
	ChatID       string              `json:"chat_id"`
	File         string              `json:"file"`
	Ordinal      *int64              `json:"ordinal,omitempty"`
	Timestamp    string              `json:"timestamp,omitempty"`
	TurnID       string              `json:"turn_id,omitempty"`
	ItemID       string              `json:"item_id,omitempty"`
	Kind         string              `json:"kind"`
	Role         string              `json:"role,omitempty"`
	Phase        string              `json:"phase,omitempty"`
	Status       string              `json:"status,omitempty"`
	Text         string              `json:"text,omitempty"`
	ToolName     string              `json:"tool_name,omitempty"`
	Namespace    string              `json:"namespace,omitempty"`
	CallID       string              `json:"call_id,omitempty"`
	Arguments    json.RawMessage     `json:"arguments,omitempty"`
	Input        json.RawMessage     `json:"input,omitempty"`
	Output       json.RawMessage     `json:"output,omitempty"`
	ContentTypes []string            `json:"content_types,omitempty"`
	Usage        *analysisCodexUsage `json:"usage,omitempty"`
}

type analysisPagination struct {
	Limit      int    `json:"limit"`
	Returned   int    `json:"returned"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}
