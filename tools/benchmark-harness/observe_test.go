// WHY: These tests protect deterministic observations from falsely turning incomplete or heuristic evidence into conclusions.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectAnalysisObservationsLinksFailuresRetriesRecoveryAndCohorts(t *testing.T) {
	index0, index1, index2 := 0, 1, 2
	cost := 0.025
	input := filepath.Join(t.TempDir(), "analysis.jsonl")
	events := []analysisEvent{
		{SchemaVersion: analysisExtractSchemaVersion, Type: "run", Source: "report.json", RunID: "job-1", TaskID: "rename", Run: &analysisRunContext{TaskID: "rename", Arm: ArmSemedit, Target: Target{Harness: "codex", Model: "gpt-6-luna", Effort: "medium"}, Success: true, WallClockMS: 900, ToolCount: 3, FailedToolCallCount: 1, FailedSemanticToolCallCount: 1, PromptTokens: 100, OutputTokens: 20, EstimatedCostUSD: &cost}},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "tool_call", Source: "report.json", RunID: "job-1", TaskID: "rename", Index: &index0, ToolCall: &ToolCall{Name: "semantic_rename", Arguments: json.RawMessage(`{"symbol":"Old"}`), TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed, Failure: "missing required parameter new_name"}},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "tool_call", Source: "report.json", RunID: "job-1", TaskID: "rename", Index: &index1, ToolCall: &ToolCall{Name: "semantic_rename", Arguments: json.RawMessage(` { "symbol" : "Old" } `), TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed, Failure: "missing required parameter new_name"}},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "tool_call", Source: "report.json", RunID: "job-1", TaskID: "rename", Index: &index2, ToolCall: &ToolCall{Name: "semantic_lookup", Arguments: json.RawMessage(`{"symbol":"Old"}`), TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusSucceeded}},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "reflection", Source: "report.json", RunID: "job-1", TaskID: "rename", Reflection: "semantic_integration", ReflectionPrompt: "What should improve?", Response: "Clearer parameter guidance and faster responses."},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "page_info", Source: "analysis_export", Pagination: &analysisPagination{Limit: 2, Returned: 2, HasMore: true, NextCursor: "next"}},
	}
	writeAnalysisEvents(t, input, events)
	report, err := collectAnalysisObservations([]string{input})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Coverage.HasMorePages {
		t.Fatal("partial pagination page was reported as complete")
	}
	if len(report.Cohorts) != 1 || report.Cohorts[0].Runs != 1 || report.Cohorts[0].FailedToolCalls != 1 || report.Cohorts[0].WallClockP99MS != 900 || report.Cohorts[0].EstimatedCostUSD != cost {
		t.Fatalf("cohort summary lost run measurements: %+v", report.Cohorts)
	}
	if len(report.ToolStats) != 2 || report.ToolStats[0].Calls+report.ToolStats[1].Calls != 3 {
		t.Fatalf("per-tool call metrics are incomplete: %+v", report.ToolStats)
	}
	var missing, failureEvidence, repeated, recovery, parameterTheme, performanceTheme bool
	for _, observation := range report.Observations {
		switch {
		case observation.Kind == "tool_failure" && observation.Signal == "missing_required_parameter":
			missing = true
			if len(observation.Evidence) != 1 || observation.Evidence[0].Index == nil {
				t.Errorf("failure is not trace-linked: %+v", observation)
			} else if *observation.Evidence[0].Index == 0 {
				failureEvidence = true
			}
		case observation.Kind == "repeated_tool_call":
			repeated = true
			if got := observation.Facts["count"]; got != 2 {
				t.Errorf("canonical-equivalent arguments were not grouped: %+v", observation.Facts)
			}
		case observation.Kind == "post_failure_action" && observation.Signal == "same_tool_same_arguments":
			recovery = true
		case observation.Kind == "reflection_theme" && observation.Signal == "parameter_guidance":
			parameterTheme = true
		case observation.Kind == "reflection_theme" && observation.Signal == "performance":
			performanceTheme = true
		}
	}
	if !missing || !failureEvidence || !repeated || !recovery || !parameterTheme || !performanceTheme {
		t.Fatalf("missing derived observation: failures=%t failure-evidence=%t repeated=%t recovery=%t parameter=%t performance=%t; observations=%+v", missing, failureEvidence, repeated, recovery, parameterTheme, performanceTheme, report.Observations)
	}
}

func TestCollectAnalysisObservationsSummarizesToolWaitAndNonToolTime(t *testing.T) {
	index, index1 := 0, 1
	wait, occupied, residual, coverage := int64(700), int64(500), int64(500), 1.0
	input := filepath.Join(t.TempDir(), "analysis.jsonl")
	semanticWait, otherWait := int64(400), int64(300)
	events := []analysisEvent{
		{Type: "run", Source: "report.json", RunID: "job-timing", Run: &analysisRunContext{TaskID: "rename", Arm: ArmSemedit, Target: Target{Harness: "codex", Model: "gpt-6-luna"}, WallClockMS: 1000, ToolCount: 2, ToolCallsTimed: 2, ToolTimingCoverage: &coverage, ToolWaitMS: &wait, SemanticToolWaitMS: &semanticWait, OtherToolWaitMS: &otherWait, ToolOccupiedMS: &occupied, NonToolWallClockMS: &residual}},
		{Type: "tool_call", Source: "report.json", RunID: "job-timing", Index: &index, ToolCall: &ToolCall{Name: "semantic_rename", Server: "semedit"}, ToolDurationMS: &semanticWait},
		{Type: "tool_call", Source: "report.json", RunID: "job-timing", Index: &index1, ToolCall: &ToolCall{Name: "go test ./..."}, ToolDurationMS: &otherWait},
	}
	writeAnalysisEvents(t, input, events)
	report, err := collectAnalysisObservations([]string{input})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cohorts) != 1 {
		t.Fatalf("got %d cohorts, want 1", len(report.Cohorts))
	}
	cohort := report.Cohorts[0]
	if cohort.ToolTimingRuns != 1 || cohort.ToolCallsTimed != 2 || cohort.ToolCallsTotal != 2 || cohort.ToolTimingCoverage != 1 || cohort.ToolWaitP50MS != wait || cohort.SemanticToolWaitP50MS != semanticWait || cohort.OtherToolWaitP50MS != otherWait || cohort.ToolOccupiedP50MS != occupied || cohort.NonToolWallP50MS != residual {
		t.Errorf("cohort timing summary = %+v", cohort)
	}
	if len(report.ToolStats) != 2 {
		t.Fatalf("tool timing summaries = %+v, want semantic and other tool rows", report.ToolStats)
	}
	var semantic, other bool
	for _, summary := range report.ToolStats {
		if summary.Semantic && summary.ToolWaitMS == semanticWait && summary.MeanToolWaitMS == semanticWait && summary.ToolWaitP50MS == semanticWait && summary.ToolWaitP95MS == semanticWait {
			semantic = true
		}
		if !summary.Semantic && summary.ToolWaitMS == otherWait && summary.MeanToolWaitMS == otherWait && summary.ToolWaitP50MS == otherWait && summary.ToolWaitP95MS == otherWait {
			other = true
		}
	}
	if !semantic || !other {
		t.Errorf("tool timing summary = %+v", report.ToolStats)
	}
}

func TestCollectAnalysisObservationsRecognizesCodexFailureAndTacticChange(t *testing.T) {
	input := filepath.Join(t.TempDir(), "codex.jsonl")
	callOrdinal, resultOrdinal, nextOrdinal := int64(1), int64(2), int64(3)
	events := []analysisEvent{
		{SchemaVersion: analysisExtractSchemaVersion, Type: "codex_chat", Source: "codex.jsonl", CodexChat: &analysisCodexChatRecord{ChatID: "chat-1", File: "session.jsonl", Ordinal: &callOrdinal, Kind: "tool_call", ToolName: "semantic_rename", CallID: "call-1", Arguments: json.RawMessage(`{"symbol":"Old"}`)}},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "codex_chat", Source: "codex.jsonl", CodexChat: &analysisCodexChatRecord{ChatID: "chat-1", File: "session.jsonl", Ordinal: &resultOrdinal, Kind: "tool_result", CallID: "call-1", Output: json.RawMessage(`"missing required parameter new_name"`)}},
		{SchemaVersion: analysisExtractSchemaVersion, Type: "codex_chat", Source: "codex.jsonl", CodexChat: &analysisCodexChatRecord{ChatID: "chat-1", File: "session.jsonl", Ordinal: &nextOrdinal, Kind: "tool_call", ToolName: "semantic_lookup", CallID: "call-2", Arguments: json.RawMessage(`{"symbol":"Old"}`)}},
	}
	writeAnalysisEvents(t, input, events)
	report, err := collectAnalysisObservations([]string{input})
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.CodexRecordsRead != 3 {
		t.Fatalf("read %d Codex records, want 3", report.Coverage.CodexRecordsRead)
	}
	var failure, tactic bool
	for _, observation := range report.Observations {
		if observation.Kind == "tool_failure" && observation.Signal == "missing_required_parameter" && len(observation.Evidence) == 2 {
			failure = true
		}
		if observation.Kind == "post_failure_action" && observation.Signal == "different_tool" && len(observation.Evidence) == 2 {
			tactic = true
		}
	}
	if !failure || !tactic {
		t.Fatalf("Codex tool failure/recovery was not derived: %+v", report.Observations)
	}
}

func TestObservationInputRejectsWrongSchema(t *testing.T) {
	input := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(input, []byte(`{"schema_version":"old","type":"run"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := collectAnalysisObservations([]string{input}); err == nil || !strings.Contains(err.Error(), "unsupported schema") {
		t.Fatalf("collectAnalysisObservations() error = %v, want unsupported schema", err)
	}
}

func TestObservationCLIHelpExitsSuccessfully(t *testing.T) {
	options, status := parseObservationCLIOptions([]string{"--help"})
	if status != 0 || !options.help {
		t.Fatalf("parseObservationCLIOptions(--help) = (%+v, %d), want help and success", options, status)
	}
}

func writeAnalysisEvents(t *testing.T, path string, events []analysisEvent) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encoder := json.NewEncoder(file)
	for _, event := range events {
		if event.SchemaVersion == "" {
			event.SchemaVersion = analysisExtractSchemaVersion
		}
		if err := encoder.Encode(event); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
