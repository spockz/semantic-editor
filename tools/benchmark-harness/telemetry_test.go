// This file verifies shared telemetry derivation because every provider adapter publishes the same run schema.

package main

import (
	"encoding/json"
	"testing"
)

func TestRefreshToolCallCounts(t *testing.T) {
	result := &RunResult{ToolCalls: []ToolCall{
		{Name: "semantic_rename", Server: "semedit", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusFailed},
		{Name: "semantic_lookup", TransportStatus: ToolCallStatusFailed, FunctionalStatus: ToolCallStatusUnknown},
		{Name: "shell", TransportStatus: ToolCallStatusFailed, FunctionalStatus: ToolCallStatusUnknown},
		{Name: "semantic_verify", TransportStatus: ToolCallStatusSucceeded, FunctionalStatus: ToolCallStatusUnknown},
	}}

	refreshToolCallCounts(result)

	if result.ToolCount != 4 || result.FailedToolCallCount != 3 || result.FailedSemanticToolCallCount != 2 {
		t.Fatalf("tool call counts = total %d, failed %d, failed semantic %d; want 4, 3, 2", result.ToolCount, result.FailedToolCallCount, result.FailedSemanticToolCallCount)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal run result: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode run result: %v", err)
	}
	if fields["failed_tool_call_count"] != float64(3) || fields["failed_semantic_tool_call_count"] != float64(2) {
		t.Fatalf("serialized failure counts = %v and %v, want 3 and 2", fields["failed_tool_call_count"], fields["failed_semantic_tool_call_count"])
	}
}
