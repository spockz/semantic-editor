// WHY: These tests keep the analysis export tied to explicit result inputs and preserve evidence used by benchmark analyses.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAnalysisExportPreservesRunAndInteractionEvidence(t *testing.T) {
	reflection := &SemanticToolReflection{Prompt: "Why did you avoid the semantic tool?", Response: "The operation was unsupported."}
	call := ToolCall{Name: "semantic_extract_function", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed, Failure: "unsupported selection"}
	run := &RunResult{
		TaskID: "extract-function", JobID: "job-1", ComparisonPairID: "pair-1", Target: Target{Harness: "codex", Model: "gpt-6-luna"}, Arm: ArmSemedit,
		Success: false, WallClock: 2 * time.Second, ToolCount: 1, FailedToolCallCount: 1, FailedSemanticToolCallCount: 1,
		PromptTokens: 12, OutputTokens: 5, EstimatedCostUSD: new(0.01), Prompt: "Extract this function", BeforeState: "before", Diff: "diff",
		Oracle: &OracleResult{Passed: false, FailureStage: "build", ErrorMessage: "compile failed", Duration: 1500 * time.Millisecond}, ToolCalls: []ToolCall{call}, ToolsUsed: []string{call.Name},
		SemanticToolReflection: reflection, SemanticIntegrationReflection: &SemanticToolReflection{Prompt: "What should improve?", Response: "Clearer parameter docs."},
		InteractionSteps: []InteractionStep{{Step: 1, Prompt: "Follow-up", WallClock: 500 * time.Millisecond, Turns: 1, ToolCalls: []ToolCall{call}}},
	}
	report := BenchmarkReport{Runs: []*RunResult{run}}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sources, err := loadAnalysisInput(path)
	if err != nil {
		t.Fatal(err)
	}
	events, err := buildAnalysisEvents(sources, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 5 {
		t.Fatalf("got %d events, want run + tool call + interaction + two reflections", len(events))
	}
	if got := events[0].Run.WallClockMS; got != 2000 {
		t.Errorf("run wall clock = %d ms, want 2000", got)
	}
	if got := events[0].Run.FailedSemanticToolCallCount; got != 1 {
		t.Errorf("failed semantic calls = %d, want 1", got)
	}
	if events[0].Run.Oracle == nil || events[0].Run.Oracle.FailureStage != "build" || events[0].Run.Oracle.DurationMS != 1500 || events[0].Run.Diff != "diff" {
		t.Errorf("run analysis evidence was dropped: %#v", events[0].Run)
	}
	if events[1].ToolCall == nil || events[1].ToolCall.Failure != "unsupported selection" {
		t.Errorf("tool call failure was dropped: %#v", events[1])
	}
	if events[1].Index == nil || *events[1].Index != 0 {
		t.Errorf("first tool call index is not linkable: %#v", events[1].Index)
	}
	if events[2].Interaction == nil || events[2].Interaction.Prompt != "Follow-up" || len(events[2].Interaction.ToolCallIndices) != 1 || events[2].Interaction.ToolCallIndices[0] != 0 {
		t.Errorf("interaction step was dropped: %#v", events[2])
	}
	if events[3].ReflectionPrompt != reflection.Prompt || events[3].Response != reflection.Response {
		t.Errorf("self-reflection evidence was dropped: %#v", events[3])
	}
	if events[4].Reflection != "semantic_integration" || events[4].Response != "Clearer parameter docs." {
		t.Errorf("tooling testimonial was dropped: %#v", events[4])
	}
}

func TestAnalysisExportMeasuresProviderToolWaitAndOccupiedWallTime(t *testing.T) {
	index0, index1 := 0, 1
	call0 := ToolCall{Name: "semantic_rename", Server: "semedit", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusSucceeded}
	call1 := ToolCall{Name: "go test ./...", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusSucceeded}
	events := []analysisEvent{
		{Type: "run", RunID: "job-1", Run: &analysisRunContext{JobID: "job-1", WallClockMS: 1000, ToolCount: 2}},
		{Type: "tool_call", RunID: "job-1", Index: &index0, ToolCall: &call0},
		{Type: "tool_call", RunID: "job-1", Index: &index1, ToolCall: &call1},
		{Type: "transcript_turn", RunID: "job-1", Transcript: &analysisTranscriptTurn{
			ToolCalls: []ToolCall{call0, call1}, ToolCallIndices: []int{0, 1}, RawEvents: []json.RawMessage{
				json.RawMessage(`{"method":"item/started","params":{"item":{"type":"mcpToolCall","id":"mcp-1","tool":"semantic_rename"},"startedAtMs":1000},"emittedAtMs":1000}`),
				json.RawMessage(`{"method":"item/started","params":{"item":{"type":"commandExecution","id":"cmd-1","command":"go test ./..."},"startedAtMs":1200},"emittedAtMs":1200}`),
				json.RawMessage(`{"method":"item/completed","params":{"item":{"type":"mcpToolCall","id":"mcp-1","tool":"semantic_rename"},"completedAtMs":1400},"emittedAtMs":1400}`),
				json.RawMessage(`{"method":"item/completed","params":{"item":{"type":"commandExecution","id":"cmd-1","command":"go test ./..."},"completedAtMs":1500},"emittedAtMs":1500}`),
			},
		}},
	}
	enrichAnalysisToolTimings(events)
	if got := *events[1].ToolDurationMS; got != 400 {
		t.Fatalf("semantic tool duration = %d ms, want 400", got)
	}
	if got := *events[2].ToolDurationMS; got != 300 {
		t.Fatalf("command duration = %d ms, want 300", got)
	}
	run := events[0].Run
	if run.ToolWaitMS == nil || *run.ToolWaitMS != 700 || run.ToolOccupiedMS == nil || *run.ToolOccupiedMS != 500 || run.NonToolWallClockMS == nil || *run.NonToolWallClockMS != 500 {
		t.Fatalf("run timing = wait %v occupied %v non-tool %v, want 700/500/500 ms", run.ToolWaitMS, run.ToolOccupiedMS, run.NonToolWallClockMS)
	}
	if run.SemanticToolWaitMS == nil || *run.SemanticToolWaitMS != 400 || run.OtherToolWaitMS == nil || *run.OtherToolWaitMS != 300 {
		t.Fatalf("semantic/other tool wait = %v/%v, want 400/300 ms", run.SemanticToolWaitMS, run.OtherToolWaitMS)
	}
	if run.ToolTimingCoverage == nil || *run.ToolTimingCoverage != 1 || run.ToolCallsTimed != 2 {
		t.Fatalf("run timing coverage = %v timed calls %d, want 1 and 2", run.ToolTimingCoverage, run.ToolCallsTimed)
	}
}

func TestTranscriptToolLinkIgnoresMetricsAddedAfterObservation(t *testing.T) {
	arguments := json.RawMessage(`{"symbol":"Old","file":"value.go"}`)
	calls := []ToolCall{{Name: "semantic_rename", Server: "semedit", Arguments: arguments, MCPMetrics: &MCPMetrics{TotalMS: 12}}}
	turns := []sessionTurn{{Classification: "task", Observation: adapterTurnObservation{ToolCalls: []ToolCall{{Name: "semantic_rename", Arguments: json.RawMessage(` { "file" : "value.go", "symbol" : "Old" } `)}}}}}
	indices := indexTranscriptToolCalls(calls, turns)
	if len(indices) != 1 || len(indices[0]) != 1 || indices[0][0] != 0 {
		t.Fatalf("transcript call did not link despite post-observation MCP metrics: %v", indices)
	}
}

func TestCodexTranscriptExportStreamsOnlyInteractiveRecordsFromTranscriptRoots(t *testing.T) {
	home := t.TempDir()
	sessions := filepath.Join(home, "sessions", "2026", "10", "19")
	archived := filepath.Join(home, "archived_sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archived, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript := strings.Join([]string{
		`{"type":"session_meta","ordinal":0,"payload":{"id":"chat-123","base_instructions":{"text":"private instructions"}}}`,
		`{"type":"response_item","ordinal":1,"timestamp":"2026-10-19T00:00:00Z","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"private instructions"}]}}`,
		`{"type":"response_item","ordinal":2,"payload":{"type":"message","role":"assistant","phase":"analysis","content":[{"type":"output_text","text":"private reasoning"}]}}`,
		`{"type":"response_item","ordinal":3,"timestamp":"2026-10-19T00:00:01Z","payload":{"type":"message","id":"user-1","role":"user","content":[{"type":"input_text","text":"Why did this tool fail?"}]}}`,
		`{"type":"response_item","ordinal":4,"payload":{"type":"message","id":"assistant-1","role":"assistant","phase":"final_answer","content":[{"type":"output_text","text":"The server rejected the argument."}]}}`,
		`{"type":"response_item","ordinal":5,"payload":{"type":"function_call","id":"call-1","call_id":"call-1","name":"semantic_rename","arguments":"{\"symbol\":\"Old\"}"}}`,
		`{"type":"response_item","ordinal":6,"payload":{"type":"function_call_output","call_id":"call-1","output":"missing required parameter new_name"}}`,
		`{"type":"response_item","ordinal":7,"payload":{"type":"reasoning","raw_content":"private reasoning"}}`,
		`{"type":"token_usage_record","ordinal":8,"payload":{"turn_token_usage":{"input_tokens":10,"cached_input_tokens":3,"output_tokens":5,"reasoning_output_tokens":2,"total_tokens":15}}}`,
		"",
	}, "\n")
	path := filepath.Join(sessions, "rollout-2026-10-19T00-00-00-chat-123.jsonl")
	if err := os.WriteFile(path, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "unrelated.jsonl"), []byte("not a Codex transcript"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archived, "old.jsonl.zst"), []byte("compressed placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := listCodexTranscriptFiles(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("found %d transcripts, want active and archived transcript files", len(files))
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	emit := func(event analysisEvent, _ analysisPageCursor) error { return encoder.Encode(event) }
	if err := streamCodexTranscripts(context.Background(), home, []string{path}, "zstd", 0, nil, emit); err != nil {
		t.Fatal(err)
	}
	var events []analysisEvent
	decoder := json.NewDecoder(&output)
	for decoder.More() {
		var event analysisEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if len(events) != 5 {
		t.Fatalf("emitted %d events, want user, assistant, tool call, tool result, and token usage", len(events))
	}
	if events[0].CodexChat.ChatID != "chat-123" || events[0].CodexChat.Text != "Why did this tool fail?" || events[0].CodexChat.Role != "user" {
		t.Errorf("user message was not normalized: %#v", events[0].CodexChat)
	}
	if events[1].CodexChat.Text != "The server rejected the argument." || events[1].CodexChat.Phase != "final_answer" {
		t.Errorf("assistant message was not retained: %#v", events[1].CodexChat)
	}
	if events[2].CodexChat.Kind != "tool_call" || events[2].CodexChat.ToolName != "semantic_rename" || string(events[2].CodexChat.Arguments) != `"{\"symbol\":\"Old\"}"` {
		t.Errorf("tool call was not retained: %#v", events[2].CodexChat)
	}
	if events[3].CodexChat.Kind != "tool_result" || !strings.Contains(string(events[3].CodexChat.Output), "new_name") {
		t.Errorf("tool result was not retained: %#v", events[3].CodexChat)
	}
	if events[4].CodexChat.Kind != "token_usage" || events[4].CodexChat.Usage == nil || events[4].CodexChat.Usage.TotalTokens != 15 {
		t.Errorf("token usage was not retained: %#v", events[4].CodexChat)
	}
	if strings.Contains(output.String(), "private instructions") || strings.Contains(output.String(), "private reasoning") {
		t.Fatal("developer instructions or private reasoning leaked into the export")
	}
}

func TestCompressedCodexTranscriptExport(t *testing.T) {
	zstd, lookupErr := exec.LookPath("zstd")
	if lookupErr != nil {
		t.Skip("zstd executable is unavailable")
	}
	home := t.TempDir()
	root := filepath.Join(home, "sessions")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(t.TempDir(), "chat.jsonl")
	if err := os.WriteFile(plain, []byte("{\"type\":\"response_item\",\"ordinal\":1,\"payload\":{\"type\":\"message\",\"role\":\"user\",\"content\":[{\"type\":\"input_text\",\"text\":\"hello\"}]}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	compressed, err := exec.Command(zstd, "-q", "-c", "--", plain).Output()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "rollout-chat-456.jsonl.zst")
	if writeErr := os.WriteFile(path, compressed, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	files, err := listCodexTranscriptFiles(home)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	emit := func(event analysisEvent, _ analysisPageCursor) error { return encoder.Encode(event) }
	if err := streamCodexTranscripts(context.Background(), home, files, zstd, 0, nil, emit); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"text":"hello"`) {
		t.Fatalf("compressed transcript output omitted user message: %s", output.String())
	}
}

func TestAnalysisCLIExportsCodexHomeWithoutBenchmarkInputs(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sessions", "2026", "10", "19")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "rollout-chat-789.jsonl")
	transcript := `{"type":"session_meta","payload":{"id":"chat-789"}}` + "\n" +
		`{"type":"response_item","ordinal":1,"payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"inspect this chat"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "analysis.jsonl")
	if code := runAnalysisCLI([]string{"--codex-home", home, "--out", output}); code != 0 {
		t.Fatalf("runAnalysisCLI returned %d", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var event analysisEvent
	if err := json.Unmarshal(bytes.TrimSpace(data), &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "codex_chat" || event.CodexChat == nil || event.CodexChat.ChatID != "chat-789" || event.CodexChat.Text != "inspect this chat" {
		t.Fatalf("unexpected Codex chat event: %#v", event)
	}
}

func TestAnalysisCLIPaginatesCodexTranscriptsAndResumesAfterCompletedFiles(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sessions")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(root, "a.jsonl")
	second := filepath.Join(root, "b.jsonl")
	userLine := func(ordinal int, text string) string {
		return `{"type":"response_item","ordinal":` + strconv.Itoa(ordinal) + `,"payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + text + `"}]}}` + "\n"
	}
	if err := os.WriteFile(first, []byte(userLine(1, "first chat")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(userLine(1, "second chat")), 0o600); err != nil {
		t.Fatal(err)
	}
	pageOne := filepath.Join(t.TempDir(), "page-one.jsonl")
	if code := runAnalysisCLI([]string{"--codex-home", home, "--limit", "1", "--out", pageOne}); code != 0 {
		t.Fatalf("first page returned status %d", code)
	}
	firstData, err := os.ReadFile(pageOne)
	if err != nil {
		t.Fatal(err)
	}
	firstEvents := decodeAnalysisTestEvents(t, firstData)
	if len(firstEvents) != 2 || firstEvents[0].CodexChat.Text != "first chat" || firstEvents[1].Pagination == nil || !firstEvents[1].Pagination.HasMore || firstEvents[1].Pagination.Returned != 1 {
		t.Fatalf("unexpected first page: %#v", firstEvents)
	}
	// A continuation must skip completed files instead of reopening the archive prefix.
	if writeErr := os.WriteFile(first, []byte("deliberately invalid JSON in an already consumed file\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	pageTwo := filepath.Join(t.TempDir(), "page-two.jsonl")
	if code := runAnalysisCLI([]string{"--codex-home", home, "--limit", "1", "--cursor", firstEvents[1].Pagination.NextCursor, "--out", pageTwo}); code != 0 {
		t.Fatalf("second page returned status %d", code)
	}
	secondData, err := os.ReadFile(pageTwo)
	if err != nil {
		t.Fatal(err)
	}
	secondEvents := decodeAnalysisTestEvents(t, secondData)
	if len(secondEvents) != 2 || secondEvents[0].CodexChat == nil || secondEvents[0].CodexChat.Text != "second chat" || secondEvents[1].Pagination == nil || secondEvents[1].Pagination.HasMore {
		t.Fatalf("unexpected second page: %#v", secondEvents)
	}
}

func decodeAnalysisTestEvents(t *testing.T, data []byte) []analysisEvent {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var events []analysisEvent
	for decoder.More() {
		var event analysisEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func TestAnalysisCLISinceFiltersRunsAndCodexRecords(t *testing.T) {
	root := t.TempDir()
	cutoff := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		name      string
		timestamp time.Time
		task      string
	}{{"old", cutoff.Add(-time.Minute), "old-task"}, {"new", cutoff.Add(time.Minute), "new-task"}} {
		report := BenchmarkReport{Timestamp: item.timestamp, Runs: []*RunResult{{TaskID: item.task}}}
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, item.name+".json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	home := filepath.Join(root, "codex")
	transcripts := filepath.Join(home, "sessions")
	if err := os.MkdirAll(transcripts, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript := strings.Join([]string{
		`{"type":"response_item","ordinal":1,"timestamp":"2026-10-19T11:59:00Z","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"old chat"}]}}`,
		`{"type":"response_item","ordinal":2,"timestamp":"2026-10-19T12:01:00Z","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"new chat"}]}}`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(transcripts, "chat.jsonl"), []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "filtered.jsonl")
	if code := runAnalysisCLI([]string{"--input", filepath.Join(root, "old.json"), "--input", filepath.Join(root, "new.json"), "--codex-home", home, "--since", cutoff.Format(time.RFC3339), "--out", output}); code != 0 {
		t.Fatalf("runAnalysisCLI returned %d", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	events := decodeAnalysisTestEvents(t, data)
	if len(events) != 2 {
		t.Fatalf("got %d records after cutoff, want new run and new chat: %#v", len(events), events)
	}
	if events[0].Type != "run" || events[0].TaskID != "new-task" || events[0].OccurredAt != "2026-10-19T12:01:00Z" {
		t.Errorf("old benchmark run was not filtered or timestamp missing: %#v", events[0])
	}
	if events[1].Type != "codex_chat" || events[1].CodexChat == nil || events[1].CodexChat.Text != "new chat" {
		t.Errorf("old Codex transcript message was not filtered: %#v", events[1])
	}
}

func TestAnalysisExportReadsOnlyRunJobFilesAndOptInSidecars(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"format_version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	jobDir := filepath.Join(root, "jobs")
	if err := os.Mkdir(jobDir, 0o700); err != nil {
		t.Fatal(err)
	}
	transcriptPath := filepath.Join(root, "transcript.json")
	rawToolResult := json.RawMessage(`{"type":"tool_result","tool":"semantic_replace_body","status":"failed","message":"required parameter body is missing"}`)
	call := ToolCall{Name: "semantic_replace_body", Arguments: json.RawMessage(`{"file":"main.go"}`), TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed}
	transcript := []sessionTurn{{Number: 1, Classification: "task", Prompt: "solve task", StartedAt: time.Unix(1, 0), Observation: adapterTurnObservation{Response: "I will try the required body parameter.", ToolCalls: []ToolCall{call}, RawEvents: []json.RawMessage{rawToolResult}}}}
	transcriptData, err := json.Marshal(transcript)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(transcriptPath, transcriptData, 0o600); err != nil {
		t.Fatal(err)
	}
	metricsPath := filepath.Join(root, "metrics.json")
	if err = os.WriteFile(metricsPath, []byte(`[{"phase":"diagnostics"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	run := &RunResult{TaskID: "t", ToolCalls: []ToolCall{call}, Provenance: ProvenanceSet{"session_transcript": transcriptPath, "mcp_metrics_file": metricsPath}}
	checkpointData, err := json.Marshal(jobCheckpoint{Job: jobCheckpointMetadata{ID: "j", TaskID: "t"}, Report: &BenchmarkReport{Runs: []*RunResult{run}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(jobDir, "j.json"), checkpointData, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(root, "nested-archive"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "nested-archive", "unrelated.json"), []byte("not JSON"), 0o600); err != nil {
		t.Fatal(err)
	}
	sources, err := loadAnalysisInput(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("loaded %d sources, want only the direct jobs checkpoint", len(sources))
	}
	events, err := buildAnalysisEvents(sources, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 || events[2].Type != "transcript_turn" || events[2].Transcript.Prompt != "solve task" || events[3].Type != "mcp_metric" {
		t.Fatalf("unexpected sidecar events: %#v", events)
	}
	if len(events[2].Transcript.RawEvents) != 1 || string(events[2].Transcript.RawEvents[0]) != string(rawToolResult) || events[2].Transcript.Response != "I will try the required body parameter." {
		t.Fatalf("transcript omitted response-guidance evidence: %#v", events[2].Transcript)
	}
	if len(events[2].Transcript.ToolCallIndices) != 1 || events[2].Transcript.ToolCallIndices[0] != 0 {
		t.Fatalf("transcript does not link its call to the run call index: %#v", events[2].Transcript.ToolCallIndices)
	}
}
