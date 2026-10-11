// OpenCode usage accounting preserves provider token buckets and rejects incomplete benchmark measurements.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

func (e *openCodeDiagnosticError) Error() string { return e.message }

func (e *openCodeDiagnosticError) Unwrap() error { return e.cause }

type openCodeDiagnosticError struct {
	cause   error
	message string
}

func (t *openCodeUsageTracker) observe(event map[string]any) {
	if openCodeEventType(event) == "error" {
		t.err = errors.Join(t.err, fmt.Errorf("OpenCode provider error: %s", openCodeEventDiagnostic(event)))
		return
	}
	part, _ := event["part"].(map[string]any)
	if properties, ok := event["properties"].(map[string]any); ok {
		part, _ = properties["part"].(map[string]any)
	}
	kind := stringValue(part["type"])
	if kind != "step-start" && kind != "step-finish" {
		return
	}
	id := stringValue(part["id"])
	if id == "" {
		t.err = errors.Join(t.err, fmt.Errorf("OpenCode %s lacks a part ID", kind))
		return
	}
	if kind == "step-start" {
		if t.starts == nil {
			t.starts = make(map[string]bool)
		}
		if !t.starts[id] {
			t.starts[id] = true
			t.finished = false
		}
		return
	}
	usage, err := parseOpenCodeStepUsage(part["tokens"])
	if err != nil {
		t.err = errors.Join(t.err, fmt.Errorf("OpenCode step %s: %w", id, err))
		return
	}
	usage.Reason = stringValue(part["reason"])
	if usage.Reason == "" {
		t.err = errors.Join(t.err, fmt.Errorf("OpenCode step %s lacks a finish reason", id))
		return
	}
	if previous, exists := t.steps[id]; exists {
		if previous != usage {
			t.err = errors.Join(t.err, fmt.Errorf("OpenCode step %s has conflicting duplicate usage", id))
		}
		return
	}
	if t.steps == nil {
		t.steps = make(map[string]openCodeStepUsage)
	}
	if t.firstStepID == "" {
		t.firstStepID = id
	}
	t.steps[id] = usage
	if usage.Reason == "length" {
		t.err = errors.Join(t.err, fmt.Errorf("OpenCode step %s exhausted the model output token limit (finish reason length)", id))
	}
	t.finished = usage.Reason == "stop"
}

func (t *openCodeUsageTracker) apply(result *RunResult) {
	for _, usage := range t.steps {
		result.PromptTokens += usage.Input + usage.CacheRead + usage.CacheWrite
		result.CachedPromptTokens += usage.CacheRead
		result.UncachedPromptTokens += usage.Input + usage.CacheWrite
		result.OutputTokens += usage.Output
		result.ReasoningTokens += usage.Reasoning
	}
	if usage, ok := t.steps[t.firstStepID]; ok {
		result.InitialContextTokens = usage.Input + usage.CacheRead + usage.CacheWrite
	}
}

type openCodeUsageTracker struct {
	steps       map[string]openCodeStepUsage
	starts      map[string]bool
	firstStepID string
	finished    bool
	err         error
}

type openCodeStepUsage struct {
	Input      int
	Output     int
	Reasoning  int
	CacheRead  int
	CacheWrite int
	Reason     string
}

func parseOpenCodeStepUsage(value any) (openCodeStepUsage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return openCodeStepUsage{}, fmt.Errorf("encode OpenCode token usage: %w", err)
	}
	var wire struct {
		Input     *int `json:"input"`
		Output    *int `json:"output"`
		Reasoning *int `json:"reasoning"`
		Cache     *struct {
			Read  *int `json:"read"`
			Write *int `json:"write"`
		} `json:"cache"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return openCodeStepUsage{}, fmt.Errorf("decode OpenCode token usage: %w", err)
	}
	if wire.Input == nil || wire.Output == nil || wire.Reasoning == nil || wire.Cache == nil || wire.Cache.Read == nil || wire.Cache.Write == nil {
		return openCodeStepUsage{}, fmt.Errorf("OpenCode token usage lacks required input, output, reasoning, or cache counters")
	}
	usage := openCodeStepUsage{Input: *wire.Input, Output: *wire.Output, Reasoning: *wire.Reasoning, CacheRead: *wire.Cache.Read, CacheWrite: *wire.Cache.Write}
	for _, count := range []int{usage.Input, usage.Output, usage.Reasoning, usage.CacheRead, usage.CacheWrite} {
		if count < 0 {
			return openCodeStepUsage{}, fmt.Errorf("OpenCode token usage contains a negative counter")
		}
	}
	return usage, nil
}

func openCodeEventDiagnostic(event map[string]any) string {
	detail, _ := event["error"].(map[string]any)
	data, _ := detail["data"].(map[string]any)
	message := firstString(data["message"], detail["message"], detail["name"], event["error"])
	if message == "" {
		message = "unspecified provider error"
	}
	return boundedDiagnostic([]byte(message), maxOpenCodeStderrBytes)
}
