// Package mcp keeps public critical-text projections structurally faithful while proving bulk ordering.
package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/projectverify"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestCriticalTextPreservesCompanionsAndOrdersOutcomeBeforeBulk(t *testing.T) {
	t.Parallel()
	continuations := []any{map[string]any{
		"file": "main.go", "symbol": "Renamed", "expected_revision": "rev-2",
		"companions": map[string]any{"read_set": []any{"preserve-inside-record"}},
	}}
	changes := make([]any, 64)
	for i := range changes {
		changes[i] = map[string]any{"file": strings.Repeat("x", 32), "revision": i}
	}
	readSet := map[string]any{"analysis": strings.Repeat("bulk", 2048)}
	structured := map[string]any{
		"result": map[string]any{
			"status": "applied", "continuations": continuations,
			"changes": changes, "read_set": readSet,
			"counter": int64(9007199254740993),
			"source":  map[string]any{"file": "main.go", "symbol": "OldName"},
			"receipt": map[string]any{"continuations": continuations, "changes": changes, "read_set": readSet, "diff": "large"},
		},
	}
	body := strings.Repeat("large formatter body", 128)
	text, err := CriticalText(structured, body)
	if err != nil {
		t.Fatalf("criticalText() error = %v", err)
	}
	prefix, rest, ok := strings.Cut(text, "\n\n")
	if !ok || rest != body {
		t.Fatalf("critical text should contain one prefix followed by the formatter body: %q", text)
	}
	const label = "Critical result (JSON): "
	if !strings.HasPrefix(prefix, label) {
		t.Fatalf("critical prefix label missing: %q", prefix)
	}
	var projected map[string]any
	if decodeErr := json.Unmarshal([]byte(strings.TrimPrefix(prefix, label)), &projected); decodeErr != nil {
		t.Fatalf("decode critical projection: %v", decodeErr)
	}
	result := projected["result"].(map[string]any)
	if result["status"] != "applied" {
		t.Fatalf("status missing from prefix: %#v", result)
	}
	if !strings.Contains(prefix, `"counter":9007199254740993`) {
		t.Fatalf("large integer was rounded in the prefix: %s", prefix)
	}
	if _, exists := result["read_set"]; exists {
		t.Fatalf("bulk read_set should be omitted from prefix: %#v", result)
	}
	if !reflect.DeepEqual(result["source"], map[string]any{"file": "main.go", "symbol": "OldName"}) {
		t.Fatalf("historical identity metadata was stripped with bulk source text: %#v", result["source"])
	}
	gotContinuations, err := json.Marshal(result["continuations"])
	if err != nil {
		t.Fatalf("encode projected continuations: %v", err)
	}
	wantContinuations, err := json.Marshal(continuations)
	if err != nil {
		t.Fatalf("encode original continuations: %v", err)
	}
	if !reflect.DeepEqual(gotContinuations, wantContinuations) {
		t.Fatalf("continuations changed: got %s, want %s", gotContinuations, wantContinuations)
	}
	if strings.Index(prefix, `"continuations"`) > strings.Index(prefix, `"changes"`) {
		t.Fatalf("continuations must precede large changes: %q", prefix)
	}
	if _, exists := result["receipt"].(map[string]any)["read_set"]; exists {
		t.Fatalf("receipt analysis read_set should be omitted from prefix: %#v", result["receipt"])
	}
	if _, exists := result["receipt"].(map[string]any)["changes"]; !exists {
		t.Fatalf("known receipt changes were lost: %#v", result["receipt"])
	}
}

func TestCriticalTextKeepsFunctionTargetBeforeBulkAndOmitsBatchWorkingTarget(t *testing.T) {
	t.Parallel()
	target := map[string]any{"file": "main.go", "symbol": "update", "expected_revision": "rev-4"}
	structured := map[string]any{
		"result": map[string]any{
			"status": "applied", "function_target": target,
			"changes": []any{map[string]any{"file": "main.go", "revision": "rev-5"}},
		},
		"read_set": map[string]any{"analysis": strings.Repeat("bulk", 512)},
	}
	text, err := CriticalText(structured, "formatter output")
	if err != nil {
		t.Fatalf("criticalText() error = %v", err)
	}
	prefix, _, ok := strings.Cut(text, "\n\n")
	if !ok {
		t.Fatalf("critical text missing body separator: %q", text)
	}
	var projection map[string]any
	if decodeErr := json.Unmarshal([]byte(strings.TrimPrefix(prefix, "Critical result (JSON): ")), &projection); decodeErr != nil {
		t.Fatalf("decode critical projection: %v", decodeErr)
	}
	result := projection["result"].(map[string]any)
	for _, field := range []string{"function_target", "changes"} {
		if _, exists := result[field]; !exists {
			t.Fatalf("critical projection missing %s: %#v", field, result)
		}
	}
	if _, exists := projection["read_set"]; !exists {
		t.Fatalf("critical projection missing top-level read_set: %#v", projection)
	}
	for _, field := range []string{`"read_set"`, `"changes"`} {
		if strings.Index(prefix, `"function_target"`) > strings.Index(prefix, field) {
			t.Fatalf("function target must precede %s: %q", field, prefix)
		}
	}
	if !reflect.DeepEqual(result["function_target"], map[string]any{
		"file": "main.go", "symbol": "update", "expected_revision": "rev-4",
	}) {
		t.Fatalf("function target identity or companion revision changed: %#v", result["function_target"])
	}

	batchText, err := CriticalText(map[string]any{"result": &BatchResponse{
		Status: "ok",
		Results: []BatchResult{{
			Tool: "semantic_change_signature", Status: "ok",
			Result: json.RawMessage(`{"status":"applied_working","function_target":{"file":"main.go","symbol":"update","expected_revision":"working-rev"}}`),
		}},
		FinalDiff: "large final diff",
	}}, "")
	if err != nil {
		t.Fatalf("criticalText() batch error = %v", err)
	}
	var batchProjection map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(batchText, "Critical result (JSON): ")), &batchProjection); err != nil {
		t.Fatalf("decode batch critical projection: %v", err)
	}
	batchResult := batchProjection["result"].(map[string]any)
	child := batchResult["results"].([]any)[0].(map[string]any)["result"].(map[string]any)
	if _, exists := child["function_target"]; exists {
		t.Fatalf("buffered batch child exposed a live function target in critical text: %#v", child)
	}
}

func TestCriticalTextOrdersErrorAndReceiptContinuationsBeforeDiagnostics(t *testing.T) {
	t.Parallel()
	record := map[string]any{"file": "main.go", "symbol": "Value", "expected_revision": "rev-3"}
	changes := make([]any, 96)
	for index := range changes {
		changes[index] = map[string]any{"file": "main.go", "revision": index}
	}
	delta := map[string]any{"net_delta": 96, "introduced": changes, "resolved": []any{}}
	structured := map[string]any{
		"application_state": "partially_published",
		"error":             "verification failed after write",
		"result": map[string]any{
			"receipt": map[string]any{
				"continuations": []any{record},
				"changes":       changes,
				"read_set":      map[string]any{"analysis": strings.Repeat("bulk", 1024)},
				"diff":          strings.Repeat("diff", 1024),
			},
		},
		"diagnostic_delta": delta,
	}
	text, err := CriticalText(structured, strings.Repeat("formatter", 128))
	if err != nil {
		t.Fatalf("criticalText() error = %v", err)
	}
	prefix, _, ok := strings.Cut(text, "\n\n")
	if !ok {
		t.Fatalf("critical text missing body separator: %q", text)
	}
	positions := []int{
		strings.Index(prefix, `"error"`),
		strings.Index(prefix, `"result"`),
		strings.Index(prefix, `"receipt"`),
		strings.Index(prefix, `"continuations"`),
		strings.Index(prefix, `"changes"`),
		strings.Index(prefix, `"diagnostic_delta"`),
	}
	for index, position := range positions {
		if position < 0 {
			t.Fatalf("critical prefix omitted ordered field %d: %q", index, prefix)
		}
		if index > 0 && positions[index-1] >= position {
			t.Fatalf("critical fields were not ordered before bulk diagnostics: %v in %q", positions, prefix)
		}
	}
}

func TestCriticalTextFallbackKeepsOperationErrorBeforeBody(t *testing.T) {
	t.Parallel()
	structured := map[string]any{
		"error":    "operation failed after a partial write",
		"location": map[string]any{"file": "main.go", "line": 4},
	}
	text := criticalTextFallback(structured, "formatter body", fmt.Errorf("cannot encode result"))
	prefix, body, ok := strings.Cut(text, "\n\n")
	if !ok || body != "formatter body" {
		t.Fatalf("fallback did not preserve body once: %q", text)
	}
	var got map[string]any
	decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(prefix, "Critical result (JSON): ")))
	decoder.UseNumber()
	if err := decoder.Decode(&got); err != nil {
		t.Fatalf("decode fallback: %v", err)
	}
	if got["application_state"] != "unknown" || got["error"] != "operation failed after a partial write" || got["location"] == nil {
		t.Fatalf("fallback lost uncertainty, original error, or location: %#v", got)
	}
}

func TestCriticalTextKeepsTypedEssentialsWhenNativeMetadataIsUnavailable(t *testing.T) {
	t.Parallel()
	structured := map[string]any{
		"result": map[string]any{
			"status": "applied", "application_state": "persisted",
			"continuations": []any{map[string]any{"file": "main.go", "symbol": "Value", "expected_revision": "rev-2"}},
		},
		"native_metadata_unavailable": true,
		"presentation_error":          "invalid native diagnostic metadata",
	}
	text, err := CriticalText(structured, "formatted body")
	if err != nil {
		t.Fatalf("criticalText() error = %v", err)
	}
	prefix, body, ok := strings.Cut(text, "\n\n")
	if !ok || body != "formatted body" {
		t.Fatalf("native metadata fallback duplicated or lost body: %q", text)
	}
	for _, field := range []string{`"application_state":"persisted"`, `"expected_revision":"rev-2"`, `"native_metadata_unavailable":true`, `"presentation_error"`} {
		if !strings.Contains(prefix, field) {
			t.Fatalf("native metadata failure prefix omitted %s: %q", field, prefix)
		}
	}
}

func TestCriticalNativeFieldsExposeExactDiagnosticResult(t *testing.T) {
	t.Parallel()
	native := operation.FileEditRes{
		DiagnosticStatus: "changed",
		Delta:            pipeline.DiagnosticDelta{NetDelta: 1, Introduced: []string{"new diagnostic"}, Resolved: []string{}},
		HasDelta:         true,
	}
	fields, err := CriticalNativeFields(native)
	if err != nil {
		t.Fatalf("criticalNativeFields() error = %v", err)
	}
	if fields["diagnostic_status"] != "changed" {
		t.Fatalf("diagnostic status = %#v", fields["diagnostic_status"])
	}
	encoded, err := json.Marshal(fields["diagnostic_delta"])
	if err != nil {
		t.Fatalf("encode diagnostic delta: %v", err)
	}
	var gotDelta map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	if decodeErr := decoder.Decode(&gotDelta); decodeErr != nil {
		t.Fatalf("decode diagnostic delta: %v", decodeErr)
	}
	wantDelta := map[string]any{"net_delta": json.Number("1"), "introduced": []any{"new diagnostic"}, "resolved": []any{}}
	if !reflect.DeepEqual(gotDelta, wantDelta) {
		t.Fatalf("diagnostic delta = %#v, want %#v", gotDelta, wantDelta)
	}
	deferred, err := CriticalNativeFields(operation.FileEditRes{DiagnosticStatus: "deferred", HasDelta: true})
	if err != nil {
		t.Fatalf("deferred diagnostic fields rejected nil lists: %v", err)
	}
	deferredDelta := deferred["diagnostic_delta"].(map[string]any)
	wantDeferred := map[string]any{
		"net_delta": json.Number("0"), "introduced": []any{}, "resolved": []any{},
	}
	if !reflect.DeepEqual(deferredDelta, wantDeferred) {
		t.Fatalf("deferred diagnostic delta = %#v, want %#v", deferredDelta, wantDeferred)
	}
}

func TestBatchCriticalOutcomeKeepsChildrenIntermediateAndAheadOfDiffs(t *testing.T) {
	t.Parallel()
	response := &BatchResponse{
		Status: "error",
		Results: []BatchResult{
			{
				Tool: "semantic_replace_body", File: "main.go", Status: "ok", Diff: strings.Repeat("large diff", 256),
				Result: json.RawMessage(`{"status":"applied","continuations":[{"file":"main.go","symbol":"Renamed","expected_revision":"working-rev"}],"diff":"large"}`),
			},
			{Tool: "semantic_remove_construct", Status: "error", Error: "refused"},
		},
		FinalDiff: strings.Repeat("final diff", 256),
	}
	body := strings.Repeat("bulk result", 128)
	text, err := CriticalText(map[string]any{"result": response}, body)
	if err != nil {
		t.Fatalf("criticalText() error = %v", err)
	}
	prefix, rest, ok := strings.Cut(text, "\n\n")
	if !ok || rest != body {
		t.Fatalf("batch critical text should retain one full body: %q", text)
	}
	var projected map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(prefix, "Critical result (JSON): ")), &projected); err != nil {
		t.Fatalf("decode batch prefix: %v", err)
	}
	result := projected["result"].(map[string]any)
	if result["status"] != "error" {
		t.Fatalf("final batch status missing: %#v", result)
	}
	children := result["results"].([]any)
	first := children[0].(map[string]any)
	if first["tool"] != "semantic_replace_body" || first["status"] != "ok" {
		t.Fatalf("successful child summary missing at results[]: %#v", first)
	}
	workingResult := first["result"].(map[string]any)
	if _, exists := workingResult["continuations"]; exists {
		t.Fatalf("batch child exposed final continuation: %#v", workingResult)
	}
	if _, exists := first["diff"]; exists {
		t.Fatalf("batch prefix retained child diff: %#v", first)
	}
	second := children[1].(map[string]any)
	if second["status"] != "error" || second["error"] != "refused" {
		t.Fatalf("failed child summary missing: %#v", second)
	}
}

func TestBatchCriticalVerificationKeepsPartialAndSkippedState(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	response := &BatchResponse{
		Status:  "error",
		Results: []BatchResult{{Tool: "semantic_replace_body", Status: "error", Error: "batch stopped"}},
		Verification: &operation.VerifyRes{
			NormalizeReport: projectverify.Report{
				Results:   []projectverify.HookResult{{ID: "format", Status: projectverify.HookSkipped, Error: "scope excluded"}},
				Completed: true,
			},
			CheckReport: projectverify.Report{
				Results: []projectverify.HookResult{{
					ID: "check", Status: projectverify.HookFailed, ExitCode: 2, Error: "hook failed",
					Changed: []string{"main.go"}, Truncated: true, Signal: "compiler",
					Stdout: strings.Repeat("stdout payload", 100), Stderr: strings.Repeat("stderr payload", 100),
					Diagnostics: []string{"full diagnostic body"}, StartedAt: started, FinishedAt: started.Add(time.Second),
				}},
				Partial: true,
			},
			Coverage: []operation.LanguageCoverage{{Language: "go", Status: "failed", Paths: []string{"main.go"}, Detail: "coverage note"}},
		},
	}
	text, err := CriticalText(map[string]any{"result": response}, "body")
	if err != nil {
		t.Fatalf("criticalText() error = %v", err)
	}
	prefix, _, _ := strings.Cut(text, "\n\n")
	var projected map[string]any
	decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(prefix, "Critical result (JSON): ")))
	decoder.UseNumber()
	if err := decoder.Decode(&projected); err != nil {
		t.Fatalf("decode batch prefix: %v", err)
	}
	result := projected["result"].(map[string]any)
	verification := result["verification"].(map[string]any)
	normalize := verification["normalize"].(map[string]any)
	if normalize["Completed"] != true || normalize["Partial"] != false {
		t.Fatalf("skipped normalize state lost: %#v", normalize)
	}
	normalizeHook := normalize["Results"].([]any)[0].(map[string]any)
	if normalizeHook["Status"] != string(projectverify.HookSkipped) || normalizeHook["Error"] != "scope excluded" {
		t.Fatalf("skipped hook evidence lost: %#v", normalizeHook)
	}
	checks := verification["checks"].(map[string]any)
	if checks["Completed"] != false || checks["Partial"] != true {
		t.Fatalf("partial checks state lost: %#v", checks)
	}
	checkHook := checks["Results"].([]any)[0].(map[string]any)
	wantCheckHook := map[string]any{
		"ID": "check", "Kind": "", "Status": string(projectverify.HookFailed),
		"ExitCode": json.Number("2"), "Truncated": true, "Changed": []any{"main.go"},
		"StartedAt": started.Format(time.RFC3339Nano), "FinishedAt": started.Add(time.Second).Format(time.RFC3339Nano),
		"Error": "hook failed", "Signal": "compiler", "ConfigPath": "", "ScopePaths": nil,
	}
	if !reflect.DeepEqual(checkHook, wantCheckHook) {
		t.Fatalf("failed hook summary = %#v, want %#v", checkHook, wantCheckHook)
	}
	wantCoverage := []any{map[string]any{
		"language": "go", "status": "failed", "paths": []any{"main.go"}, "detail": "coverage note",
	}}
	if !reflect.DeepEqual(verification["coverage"], wantCoverage) {
		t.Fatalf("coverage state = %#v, want %#v", verification["coverage"], wantCoverage)
	}
	for _, bulk := range []string{"Stdout", "Stderr", "Diagnostics"} {
		if _, exists := checkHook[bulk]; exists {
			t.Fatalf("bulk hook field %s retained in prefix: %#v", bulk, checkHook)
		}
	}
}

func TestBatchResultSchemaAllowsOverlappingSuccessAndErrorProjection(t *testing.T) {
	t.Parallel()
	schema := operation.ValueSchema{
		Kind:       operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{"status": {Kind: operation.SchemaString}},
		Required:   []string{"status"},
	}
	entry := operation.Entry{
		MCPName:       "semantic_probe_projection",
		Output:        &operation.OperationOutputContract{ResultSchema: &schema},
		ProjectOutput: func(any) (any, error) { return map[string]any{"status": "known"}, nil },
		ErrorOutput:   &operation.OperationErrorOutputContract{ResultSchema: &schema},
		ProjectErrorOutput: func(error) (any, bool, error) {
			return map[string]any{"status": "known"}, true, nil
		},
	}
	rowSchema, err := batchResultSchemaForEntries([]operation.Entry{entry})
	if err != nil {
		t.Fatalf("batchResultSchemaForEntries() error = %v", err)
	}
	encoded, err := json.Marshal(rowSchema)
	if err != nil {
		t.Fatalf("encode row schema: %v", err)
	}
	var document any
	if decodeErr := json.Unmarshal(encoded, &document); decodeErr != nil {
		t.Fatalf("decode row schema: %v", decodeErr)
	}
	compiler := jsonschema.NewCompiler()
	if resourceErr := compiler.AddResource("batch-row.json", document); resourceErr != nil {
		t.Fatalf("add row schema: %v", resourceErr)
	}
	compiled, err := compiler.Compile("batch-row.json")
	if err != nil {
		t.Fatalf("compile row schema: %v", err)
	}
	for label, row := range map[string]any{
		"success":     map[string]any{"tool": entry.MCPName, "status": "ok", "result": map[string]any{"status": "known"}},
		"known error": map[string]any{"tool": entry.MCPName, "status": "error", "error": "known failure", applicationStateField: "unknown", "result": map[string]any{"status": "known"}},
	} {
		if err := compiled.Validate(row); err != nil {
			t.Errorf("overlapping %s projection rejected: %v", label, err)
		}
	}
	if err := compiled.Validate(map[string]any{"tool": entry.MCPName, "status": "error", "error": "known failure", applicationStateField: "unknown"}); err != nil {
		t.Errorf("generic error projection rejected: %v", err)
	}
}
