// WHY: This file derives reproducible, evidence-linked observations from selected analysis exports without making causal or capability claims.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

const observationSchemaVersion = "semedit.benchmark-observations/v1"

type analysisEvidenceRef struct {
	Source  string `json:"source,omitempty"`
	RunID   string `json:"run_id,omitempty"`
	TaskID  string `json:"task_id,omitempty"`
	Index   *int   `json:"tool_call_index,omitempty"`
	Step    int    `json:"step,omitempty"`
	ChatID  string `json:"chat_id,omitempty"`
	File    string `json:"transcript_file,omitempty"`
	Ordinal *int64 `json:"ordinal,omitempty"`
	CallID  string `json:"call_id,omitempty"`
}

type analysisObservation struct {
	Kind     string                `json:"kind"`
	Signal   string                `json:"signal,omitempty"`
	Evidence []analysisEvidenceRef `json:"evidence"`
	Facts    map[string]any        `json:"facts,omitempty"`
}

type analysisCohort struct {
	Key                   string  `json:"key"`
	TaskID                string  `json:"task_id"`
	Arm                   ArmType `json:"arm"`
	Harness               string  `json:"harness"`
	Model                 string  `json:"model"`
	Effort                string  `json:"reasoning_effort,omitempty"`
	Variant               string  `json:"variant,omitempty"`
	Skill                 bool    `json:"semedit_skill"`
	Batching              bool    `json:"semedit_batching"`
	MCPServerInstructions string  `json:"mcp_server_instructions,omitempty"`
	ArmRestriction        string  `json:"semedit_arm_restriction,omitempty"`
	ArmRestrictionApplied bool    `json:"semedit_arm_restriction_applied"`
	ExecutionMode         string  `json:"execution_mode,omitempty"`
	Runs                  int     `json:"runs"`
	Successes             int     `json:"successes"`
	FailedToolCalls       int     `json:"failed_tool_calls"`
	FailedSemanticCalls   int     `json:"failed_semantic_tool_calls"`
	WallClockP50MS        int64   `json:"wall_clock_p50_ms"`
	WallClockP95MS        int64   `json:"wall_clock_p95_ms"`
	WallClockP99MS        int64   `json:"wall_clock_p99_ms"`
	ToolTimingRuns        int     `json:"tool_timing_runs"`
	ToolCallsTimed        int     `json:"tool_calls_timed"`
	ToolCallsTotal        int     `json:"tool_calls_total"`
	ToolTimingCoverage    float64 `json:"tool_timing_coverage"`
	ToolWaitP50MS         int64   `json:"tool_wait_p50_ms"`
	ToolWaitP95MS         int64   `json:"tool_wait_p95_ms"`
	SemanticToolWaitP50MS int64   `json:"semantic_tool_wait_p50_ms"`
	SemanticToolWaitP95MS int64   `json:"semantic_tool_wait_p95_ms"`
	OtherToolWaitP50MS    int64   `json:"other_tool_wait_p50_ms"`
	OtherToolWaitP95MS    int64   `json:"other_tool_wait_p95_ms"`
	ToolOccupiedP50MS     int64   `json:"tool_occupied_p50_ms"`
	ToolOccupiedP95MS     int64   `json:"tool_occupied_p95_ms"`
	NonToolWallP50MS      int64   `json:"non_tool_wall_clock_p50_ms"`
	NonToolWallP95MS      int64   `json:"non_tool_wall_clock_p95_ms"`
	PromptTokens          int64   `json:"prompt_tokens"`
	OutputTokens          int64   `json:"output_tokens"`
	ReasoningTokens       int64   `json:"reasoning_tokens"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
	CostEstimateCount     int     `json:"cost_estimate_count"`
}

type analysisToolSummary struct {
	TaskID             string  `json:"task_id,omitempty"`
	Arm                ArmType `json:"arm,omitempty"`
	Harness            string  `json:"harness,omitempty"`
	Model              string  `json:"model,omitempty"`
	Effort             string  `json:"reasoning_effort,omitempty"`
	Skill              bool    `json:"semedit_skill"`
	Batching           bool    `json:"semedit_batching"`
	Tool               string  `json:"tool"`
	Server             string  `json:"server,omitempty"`
	Semantic           bool    `json:"semantic_tool"`
	Calls              int     `json:"calls"`
	TransportFailures  int     `json:"transport_failures"`
	FunctionalFailures int     `json:"functional_failures"`
	UnknownOutcomes    int     `json:"unknown_outcomes"`
	TimedCalls         int     `json:"timed_calls"`
	ToolWaitMS         int64   `json:"tool_wait_ms"`
	MeanToolWaitMS     int64   `json:"mean_tool_wait_ms"`
	ToolWaitP50MS      int64   `json:"tool_wait_p50_ms"`
	ToolWaitP95MS      int64   `json:"tool_wait_p95_ms"`
}

type analysisObservationReport struct {
	SchemaVersion string                `json:"schema_version"`
	Inputs        []string              `json:"inputs"`
	Observations  []analysisObservation `json:"observations"`
	Cohorts       []analysisCohort      `json:"cohorts"`
	ToolStats     []analysisToolSummary `json:"tool_stats"`
	Coverage      analysisCoverage      `json:"coverage"`
}

type analysisCoverage struct {
	EventsRead       int  `json:"events_read"`
	HasMorePages     bool `json:"has_more_pages"`
	UnknownOutcomes  int  `json:"unknown_tool_outcomes"`
	CodexRecordsRead int  `json:"codex_records_read"`
}

type observationCall struct {
	ref     analysisEvidenceRef
	name    string
	args    string
	failed  bool
	signal  string
	status  string
	message string
}

type observationRun struct {
	run    *analysisRunContext
	source string
	calls  []observationCall
}

type observationCohortState struct {
	cohort       analysisCohort
	wall         []int64
	toolWait     []int64
	semanticWait []int64
	otherWait    []int64
	toolOccupied []int64
	nonToolWall  []int64
}

type observationAccumulator struct {
	report              *analysisObservationReport
	runs                map[string]*observationRun
	cohorts             map[string]*observationCohortState
	codexCalls          map[string][]observationCall
	codexByID           map[string]observationCall
	codexPendingFailure map[string]observationCall
	toolStats           map[string]*analysisToolSummary
	toolWaitSamples     map[string][]int64
	codexStatsByID      map[string]*analysisToolSummary
}

func runObservationCLI(args []string) int {
	options, status := parseObservationCLIOptions(args)
	if status != 0 {
		return status
	}
	if options.help {
		return 0
	}
	report, err := collectAnalysisObservations(options.inputs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "observe input: %v\n", err)
		return 1
	}
	report.Inputs = slices.Clone(options.inputs)
	if err := writeObservationReport(options.output, report); err != nil {
		fmt.Fprintf(os.Stderr, "write observations: %v\n", err)
		return 1
	}
	return 0
}

type observationCLIOptions struct {
	inputs analysisInputFlags
	output string
	help   bool
}

func parseObservationCLIOptions(args []string) (observationCLIOptions, int) {
	fs := flag.NewFlagSet("observe", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var options observationCLIOptions
	fs.Var(&options.inputs, "input", "Analysis JSONL file from analyze; repeat to combine selected pages")
	fs.StringVar(&options.output, "out", "", "Output JSON report (default: stdout)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			options.help = true
			return options, 0
		}
		return options, 2
	}
	if len(options.inputs) == 0 || fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: benchmark-harness observe --input analysis.jsonl [--input next-page.jsonl] [--out observations.json]")
		return options, 2
	}
	return options, 0
}

func collectAnalysisObservations(inputs []string) (analysisObservationReport, error) {
	report := analysisObservationReport{SchemaVersion: observationSchemaVersion, Observations: []analysisObservation{}, Cohorts: []analysisCohort{}, ToolStats: []analysisToolSummary{}}
	accumulator := observationAccumulator{
		report: &report, runs: make(map[string]*observationRun), cohorts: make(map[string]*observationCohortState),
		codexCalls: make(map[string][]observationCall), codexByID: make(map[string]observationCall), codexPendingFailure: make(map[string]observationCall),
		toolStats: make(map[string]*analysisToolSummary), toolWaitSamples: make(map[string][]int64), codexStatsByID: make(map[string]*analysisToolSummary),
	}
	for _, input := range inputs {
		lastPageMore, err := accumulator.readInput(input)
		if err != nil {
			return report, err
		}
		if lastPageMore {
			report.Coverage.HasMorePages = true
		}
	}
	accumulator.finalize()
	return report, nil
}

func (accumulator *observationAccumulator) readInput(input string) (bool, error) {
	// The caller explicitly selects analysis exports; this command does not discover files from this path.
	// #nosec G304 -- explicit CLI input, never a discovered or archived-result path.
	file, err := os.Open(input)
	if err != nil {
		return false, fmt.Errorf("open %q: %w", input, err)
	}
	lastPageMore := false
	err = decodeAnalysisJSONL(file, func(event analysisEvent) error {
		if event.Type == "page_info" && event.Pagination != nil {
			accumulator.report.Coverage.EventsRead++
			lastPageMore = event.Pagination.HasMore
			return nil
		}
		return accumulator.observe(event)
	})
	closeErr := file.Close()
	if err != nil {
		return false, fmt.Errorf("read %q: %w", input, err)
	}
	if closeErr != nil {
		return false, fmt.Errorf("close %q: %w", input, closeErr)
	}
	return lastPageMore, nil
}

func (accumulator *observationAccumulator) finalize() {
	report := accumulator.report
	for _, run := range accumulator.runs {
		appendRunSequenceObservations(report, run)
	}
	for _, calls := range accumulator.codexCalls {
		appendRepeatedCallObservations(report, calls)
	}
	for _, state := range accumulator.cohorts {
		slices.Sort(state.wall)
		state.cohort.WallClockP50MS = percentileObservation(state.wall, 0.50)
		state.cohort.WallClockP95MS = percentileObservation(state.wall, 0.95)
		state.cohort.WallClockP99MS = percentileObservation(state.wall, 0.99)
		slices.Sort(state.toolWait)
		slices.Sort(state.semanticWait)
		slices.Sort(state.otherWait)
		slices.Sort(state.toolOccupied)
		slices.Sort(state.nonToolWall)
		state.cohort.ToolWaitP50MS = percentileObservation(state.toolWait, 0.50)
		state.cohort.ToolWaitP95MS = percentileObservation(state.toolWait, 0.95)
		state.cohort.SemanticToolWaitP50MS = percentileObservation(state.semanticWait, 0.50)
		state.cohort.SemanticToolWaitP95MS = percentileObservation(state.semanticWait, 0.95)
		state.cohort.OtherToolWaitP50MS = percentileObservation(state.otherWait, 0.50)
		state.cohort.OtherToolWaitP95MS = percentileObservation(state.otherWait, 0.95)
		state.cohort.ToolOccupiedP50MS = percentileObservation(state.toolOccupied, 0.50)
		state.cohort.ToolOccupiedP95MS = percentileObservation(state.toolOccupied, 0.95)
		state.cohort.NonToolWallP50MS = percentileObservation(state.nonToolWall, 0.50)
		state.cohort.NonToolWallP95MS = percentileObservation(state.nonToolWall, 0.95)
		if state.cohort.ToolCallsTotal > 0 {
			state.cohort.ToolTimingCoverage = float64(state.cohort.ToolCallsTimed) / float64(state.cohort.ToolCallsTotal)
		}
		report.Cohorts = append(report.Cohorts, state.cohort)
	}
	for key, summary := range accumulator.toolStats {
		samples := accumulator.toolWaitSamples[key]
		slices.Sort(samples)
		summary.ToolWaitP50MS = percentileObservation(samples, 0.50)
		summary.ToolWaitP95MS = percentileObservation(samples, 0.95)
		if summary.TimedCalls > 0 {
			summary.MeanToolWaitMS = summary.ToolWaitMS / int64(summary.TimedCalls)
		}
		report.ToolStats = append(report.ToolStats, *summary)
	}
	sort.Slice(report.Cohorts, func(i, j int) bool { return report.Cohorts[i].Key < report.Cohorts[j].Key })
	sort.Slice(report.ToolStats, func(i, j int) bool {
		a, b := report.ToolStats[i], report.ToolStats[j]
		if a.TaskID != b.TaskID {
			return a.TaskID < b.TaskID
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		if a.Tool != b.Tool {
			return a.Tool < b.Tool
		}
		return a.Server < b.Server
	})
	sort.SliceStable(report.Observations, func(i, j int) bool {
		a, b := report.Observations[i], report.Observations[j]
		if a.Evidence[0].Source != b.Evidence[0].Source {
			return a.Evidence[0].Source < b.Evidence[0].Source
		}
		if a.Evidence[0].RunID != b.Evidence[0].RunID {
			return a.Evidence[0].RunID < b.Evidence[0].RunID
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Signal != b.Signal {
			return a.Signal < b.Signal
		}
		if a.Evidence[0].Index != nil && b.Evidence[0].Index != nil && *a.Evidence[0].Index != *b.Evidence[0].Index {
			return *a.Evidence[0].Index < *b.Evidence[0].Index
		}
		if a.Evidence[0].Ordinal != nil && b.Evidence[0].Ordinal != nil && *a.Evidence[0].Ordinal != *b.Evidence[0].Ordinal {
			return *a.Evidence[0].Ordinal < *b.Evidence[0].Ordinal
		}
		return fmt.Sprint(a.Facts["tool"]) < fmt.Sprint(b.Facts["tool"])
	})
}

func (accumulator *observationAccumulator) observe(event analysisEvent) error {
	report := accumulator.report
	report.Coverage.EventsRead++
	if event.CodexChat != nil {
		report.Coverage.CodexRecordsRead++
		collectCodexObservation(report, event, accumulator.codexCalls, accumulator.codexByID, accumulator.codexPendingFailure, accumulator.toolStats, accumulator.codexStatsByID)
	}
	if event.Type == "run" && event.Run != nil {
		key := observationRunKey(event)
		accumulator.runs[key] = &observationRun{run: event.Run, source: event.Source}
		addObservationCohort(accumulator.cohorts, *event.Run)
	}
	if event.Type == "tool_call" && event.ToolCall != nil {
		accumulator.observeToolCall(event)
	}
	if event.Type == "reflection" {
		accumulator.observeReflection(event)
	}
	return nil
}

func (accumulator *observationAccumulator) observeToolCall(event analysisEvent) {
	callRecord := event.ToolCall
	if callRecord.TransportStatus == ToolCallStatusUnknown || callRecord.FunctionalStatus == ToolCallStatusUnknown {
		accumulator.report.Coverage.UnknownOutcomes++
	}
	accumulator.addToolStats(event, callRecord.Name, callRecord.Server, callRecord.TransportStatus, callRecord.FunctionalStatus, event.ToolDurationMS)
	ref := eventEvidence(event)
	call := observationCall{name: callRecord.Name, args: canonicalObservationJSON(callRecord.Arguments), status: string(callRecord.FunctionalStatus), message: callRecord.Failure, ref: ref}
	call.signal, call.failed = classifyToolFailure(callRecord.TransportStatus, callRecord.FunctionalStatus, callRecord.Failure)
	if call.failed {
		accumulator.report.Observations = append(accumulator.report.Observations, analysisObservation{Kind: "tool_failure", Signal: call.signal, Evidence: []analysisEvidenceRef{ref}, Facts: map[string]any{"tool": call.name, "transport_status": string(callRecord.TransportStatus), "functional_status": string(callRecord.FunctionalStatus), "message": call.message}})
	}
	key := observationRunKey(event)
	accumulator.runs[key] = ensureObservationRun(accumulator.runs[key], event)
	accumulator.runs[key].calls = append(accumulator.runs[key].calls, call)
}

func (accumulator *observationAccumulator) observeReflection(event analysisEvent) {
	text := event.ReflectionPrompt + "\n" + event.Response + "\n" + event.Failure
	for _, theme := range reflectionObservationThemes(text) {
		accumulator.report.Observations = append(accumulator.report.Observations, analysisObservation{Kind: "reflection_theme", Signal: theme, Evidence: []analysisEvidenceRef{eventEvidence(event)}})
	}
}

func decodeAnalysisJSONL(reader io.Reader, visit func(analysisEvent) error) error {
	decoder := json.NewDecoder(bufio.NewReader(reader))
	for ordinal := 1; ; ordinal++ {
		var event analysisEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("JSONL record %d: %w", ordinal, err)
		}
		if event.SchemaVersion != analysisExtractSchemaVersion {
			return fmt.Errorf("JSONL record %d has unsupported schema %q", ordinal, event.SchemaVersion)
		}
		if err := visit(event); err != nil {
			return fmt.Errorf("JSONL record %d: %w", ordinal, err)
		}
	}
}

func observationRunKey(event analysisEvent) string {
	return event.Source + "\x00" + event.RunID
}

func ensureObservationRun(current *observationRun, event analysisEvent) *observationRun {
	if current != nil {
		return current
	}
	return &observationRun{run: event.Run, source: event.Source}
}

func eventEvidence(event analysisEvent) analysisEvidenceRef {
	ref := analysisEvidenceRef{Source: event.Source, RunID: event.RunID, TaskID: event.TaskID, Index: event.Index, Step: event.Step}
	if event.CodexChat != nil {
		ref.ChatID, ref.File, ref.Ordinal, ref.CallID = event.CodexChat.ChatID, event.CodexChat.File, event.CodexChat.Ordinal, event.CodexChat.CallID
	}
	return ref
}

func classifyToolFailure(transport, functional ToolCallStatus, message string) (string, bool) {
	if transport == ToolCallStatusFailed {
		return "transport_failure", true
	}
	if functional != ToolCallStatusFailed {
		return "", false
	}
	return classifyFailureText(message), true
}

func classifyFailureText(message string) string {
	text := strings.ToLower(message)
	switch {
	case strings.Contains(text, "missing required parameter"), strings.Contains(text, "required parameter") && strings.Contains(text, "missing"):
		return "missing_required_parameter"
	case strings.Contains(text, "unsupported"), strings.Contains(text, "not supported"), strings.Contains(text, "unavailable"):
		return "unsupported_or_unavailable_signal"
	case strings.Contains(text, "timeout"), strings.Contains(text, "timed out"):
		return "timeout_signal"
	case strings.Contains(text, "diagnostic"), strings.Contains(text, "compile"), strings.Contains(text, "syntax"):
		return "verification_or_syntax_signal"
	default:
		return "functional_rejection"
	}
}

func canonicalObservationJSON(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return string(data)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return string(data)
	}
	return string(canonical)
}

func appendRunSequenceObservations(report *analysisObservationReport, run *observationRun) {
	if run == nil || run.run == nil || len(run.calls) == 0 {
		return
	}
	appendRepeatedCallObservations(report, run.calls)
	for index, failed := range run.calls {
		if !failed.failed {
			continue
		}
		observation := analysisObservation{Kind: "post_failure_action", Signal: "no_later_tool_call_observed", Evidence: []analysisEvidenceRef{failed.ref}, Facts: map[string]any{"failed_tool": failed.name, "failure_signal": failed.signal}}
		if index+1 < len(run.calls) {
			next := run.calls[index+1]
			signal := tacticChangeSignal(failed, next)
			observation.Signal = signal
			observation.Evidence = append(observation.Evidence, next.ref)
			observation.Facts["next_tool"] = next.name
			observation.Facts["next_arguments"] = next.args
			observation.Facts["next_outcome"] = next.status
		}
		report.Observations = append(report.Observations, observation)
	}
}

func tacticChangeSignal(previous, next observationCall) string {
	if previous.name != next.name {
		return "different_tool"
	}
	if previous.args != next.args {
		return "same_tool_changed_arguments"
	}
	return "same_tool_same_arguments"
}

func appendRepeatedCallObservations(report *analysisObservationReport, calls []observationCall) {
	seen := make(map[string][]observationCall)
	for _, call := range calls {
		key := call.name + "\x00" + call.args
		seen[key] = append(seen[key], call)
	}
	for _, calls := range seen {
		if len(calls) < 2 {
			continue
		}
		refs := make([]analysisEvidenceRef, len(calls))
		for i := range calls {
			refs[i] = calls[i].ref
		}
		report.Observations = append(report.Observations, analysisObservation{Kind: "repeated_tool_call", Signal: "exact_tool_and_arguments_repeated", Evidence: refs, Facts: map[string]any{"tool": calls[0].name, "count": len(calls), "outcomes": callOutcomes(calls)}})
	}
}

func callOutcomes(calls []observationCall) []string {
	outcomes := make([]string, len(calls))
	for i := range calls {
		outcomes[i] = calls[i].status
	}
	return outcomes
}

func reflectionObservationThemes(text string) []string {
	text = strings.ToLower(text)
	themes := []struct {
		name  string
		terms []string
	}{
		{"parameter_guidance", []string{"parameter", "argument", "schema"}},
		{"missing_capability", []string{"unsupported", "missing capability", "does not support", "cannot"}},
		{"response_actionability", []string{"response", "diagnostic", "error message", "actionable", "guidance"}},
		{"performance", []string{"slow", "faster", "latency", "wait", "time", "performance"}},
		{"batch_workflow", []string{"batch", "staging"}},
		{"tool_discovery", []string{"tool selection", "which tool", "discover", "description"}},
	}
	var matches []string
	for _, theme := range themes {
		for _, term := range theme.terms {
			if strings.Contains(text, term) {
				matches = append(matches, theme.name)
				break
			}
		}
	}
	return matches
}

func collectCodexObservation(report *analysisObservationReport, event analysisEvent, calls map[string][]observationCall, callsByID map[string]observationCall, pending map[string]observationCall, toolStats map[string]*analysisToolSummary, codexStats map[string]*analysisToolSummary) {
	record := event.CodexChat
	if record == nil {
		return
	}
	ref := eventEvidence(event)
	chatID := record.ChatID
	switch record.Kind {
	case "tool_result":
		text := rawObservationText(record.Output)
		call := callsByID[chatID+"\x00"+record.CallID]
		if signal := classifyCodexResult(text, record.Status); signal != "" {
			call.failed, call.signal, call.message = true, signal, text
			call.ref.CallID = record.CallID
			tool := call.name
			if tool == "" {
				tool = record.ToolName
			}
			report.Observations = append(report.Observations, analysisObservation{Kind: "tool_failure", Signal: signal, Evidence: []analysisEvidenceRef{call.ref, ref}, Facts: map[string]any{"tool": tool, "call_id": record.CallID, "message": text}})
			if summary := codexStats[chatID+"\x00"+record.CallID]; summary != nil {
				summary.FunctionalFailures++
				summary.UnknownOutcomes--
			}
			pending[chatID] = call
		}
	case "tool_call":
		if failure, ok := pending[chatID]; ok {
			next := observationCall{name: record.ToolName, args: canonicalObservationJSON(record.Arguments)}
			report.Observations = append(report.Observations, analysisObservation{Kind: "post_failure_action", Signal: tacticChangeSignal(failure, next), Evidence: []analysisEvidenceRef{failure.ref, ref}, Facts: map[string]any{"failed_tool": failure.name, "next_tool": record.ToolName, "failure_signal": failure.signal, "next_arguments": next.args}})
			delete(pending, chatID)
		}
		ref.CallID = record.CallID
		call := observationCall{ref: ref, name: record.ToolName, args: canonicalObservationJSON(record.Arguments)}
		calls[chatID] = append(calls[chatID], call)
		summary := addTranscriptToolStats(toolStats, record)
		if record.CallID != "" {
			callsByID[chatID+"\x00"+record.CallID] = call
			codexStats[chatID+"\x00"+record.CallID] = summary
		}
	}
}

func classifyCodexResult(message, status string) string {
	text := strings.ToLower(strings.TrimSpace(message))
	if strings.EqualFold(status, "failed") || strings.EqualFold(status, "error") || strings.Contains(text, "missing required parameter") || strings.Contains(text, "iserror\":true") || strings.HasPrefix(text, "error:") || strings.Contains(text, "request failed") || strings.Contains(text, "tool call failed") || strings.Contains(text, "timed out") {
		return classifyFailureText(message)
	}
	return ""
}

func rawObservationText(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(data, &text) == nil {
		return text
	}
	return string(data)
}

func cohortKey(run analysisRunContext) string {
	identity := struct {
		TaskID                string
		Arm                   ArmType
		Harness               string
		Model                 string
		Effort                string
		Variant               string
		PromptVariant         string
		Skill                 bool
		Batching              bool
		Instructions          MCPServerInstructionMode
		ArmRestriction        SemeditArmRestriction
		ArmRestrictionApplied bool
		ExecutionMode         string
	}{run.TaskID, run.Arm, run.Target.Harness, run.Target.Model, run.Target.Effort, run.Variant, run.PromptVariant, run.Skill, run.Batching, run.MCPServerInstructions, run.ArmRestriction, run.ArmRestrictionApplied, run.ExecutionMode}
	encoded, _ := json.Marshal(identity)
	return string(encoded)
}

func addObservationCohort(cohorts map[string]*observationCohortState, run analysisRunContext) {
	key := cohortKey(run)
	state := cohorts[key]
	if state == nil {
		state = &observationCohortState{cohort: analysisCohort{Key: key, TaskID: run.TaskID, Arm: run.Arm, Harness: run.Target.Harness, Model: run.Target.Model, Effort: run.Target.Effort, Variant: run.Variant, Skill: run.Skill, Batching: run.Batching, MCPServerInstructions: string(run.MCPServerInstructions), ArmRestriction: string(run.ArmRestriction), ArmRestrictionApplied: run.ArmRestrictionApplied, ExecutionMode: run.ExecutionMode}}
		cohorts[key] = state
	}
	c := &state.cohort
	c.Runs++
	if run.Success {
		c.Successes++
	}
	c.FailedToolCalls += run.FailedToolCallCount
	c.FailedSemanticCalls += run.FailedSemanticToolCallCount
	c.ToolCallsTotal += run.ToolCount
	c.ToolCallsTimed += run.ToolCallsTimed
	c.PromptTokens += int64(run.PromptTokens)
	c.OutputTokens += int64(run.OutputTokens)
	c.ReasoningTokens += int64(run.ReasoningTokens)
	state.wall = append(state.wall, run.WallClockMS)
	if run.ToolWaitMS != nil && run.ToolOccupiedMS != nil && run.NonToolWallClockMS != nil {
		state.cohort.ToolTimingRuns++
		state.toolWait = append(state.toolWait, *run.ToolWaitMS)
		if run.SemanticToolWaitMS != nil {
			state.semanticWait = append(state.semanticWait, *run.SemanticToolWaitMS)
		}
		if run.OtherToolWaitMS != nil {
			state.otherWait = append(state.otherWait, *run.OtherToolWaitMS)
		}
		state.toolOccupied = append(state.toolOccupied, *run.ToolOccupiedMS)
		state.nonToolWall = append(state.nonToolWall, *run.NonToolWallClockMS)
	}
	if run.EstimatedCostUSD != nil {
		c.EstimatedCostUSD += *run.EstimatedCostUSD
		c.CostEstimateCount++
	}
}

func (accumulator *observationAccumulator) addToolStats(event analysisEvent, tool, server string, transport, functional ToolCallStatus, durationMS *int64) {
	run := accumulator.runs[observationRunKey(event)]
	if run == nil || run.run == nil {
		return
	}
	context := run.run
	key := cohortKey(*context) + "\x00" + tool + "\x00" + server
	summary := accumulator.toolStats[key]
	if summary == nil {
		summary = &analysisToolSummary{TaskID: context.TaskID, Arm: context.Arm, Harness: context.Target.Harness, Model: context.Target.Model, Effort: context.Target.Effort, Skill: context.Skill, Batching: context.Batching, Tool: tool, Server: server, Semantic: isSemanticTool(tool, server)}
		accumulator.toolStats[key] = summary
	}
	summary.Calls++
	if durationMS != nil {
		summary.TimedCalls++
		summary.ToolWaitMS += *durationMS
		accumulator.toolWaitSamples[key] = append(accumulator.toolWaitSamples[key], *durationMS)
	}
	if transport == ToolCallStatusFailed {
		summary.TransportFailures++
	}
	if functional == ToolCallStatusFailed {
		summary.FunctionalFailures++
	}
	if transport == ToolCallStatusUnknown || functional == ToolCallStatusUnknown {
		summary.UnknownOutcomes++
	}
}

func addTranscriptToolStats(stats map[string]*analysisToolSummary, record *analysisCodexChatRecord) *analysisToolSummary {
	key := "codex-chat\x00" + record.ToolName + "\x00" + record.Namespace
	summary := stats[key]
	if summary == nil {
		summary = &analysisToolSummary{Harness: "codex-chat", Tool: record.ToolName, Server: record.Namespace}
		stats[key] = summary
	}
	summary.Calls++
	summary.UnknownOutcomes++
	return summary
}

func percentileObservation(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	index := max(int(float64(len(values))*percentile+0.999999)-1, 0)
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func writeObservationReport(output string, report analysisObservationReport) error {
	var writer io.Writer = os.Stdout
	var temp *os.File
	if output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		var err error
		temp, err = os.CreateTemp(filepath.Dir(output), ".observations-*.tmp")
		if err != nil {
			return fmt.Errorf("create temporary output: %w", err)
		}
		tempPath := temp.Name()
		defer func() {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}()
		writer = temp
	}
	if err := json.NewEncoder(writer).Encode(report); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	if temp == nil {
		return nil
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync output: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	if err := os.Rename(temp.Name(), output); err != nil {
		return fmt.Errorf("publish output: %w", err)
	}
	return nil
}
