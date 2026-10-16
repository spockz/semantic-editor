// Package presentation tests result projection without compiling MCP dispatch and server integration tests.
package presentation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCriticalTextOmitsShortSourceAndBulkFallback(t *testing.T) {
	t.Parallel()
	source := "func Value() int { return 42 }"
	text, err := CriticalText(map[string]any{"result": map[string]any{"inspection": source}}, source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, source) || !strings.Contains(text, "inspection_bytes") {
		t.Fatalf("short source duplicated: %s", text)
	}
	err = json.Unmarshal([]byte("{"), new(any))
	bulk := strings.Repeat("bulk formatter\n", 100)
	fallback := CriticalTextFallback(map[string]any{"error": "partial write", "location": map[string]any{"file": "main.go"}}, bulk, err)
	if strings.Contains(fallback, "bulk formatter") || !strings.Contains(fallback, "partial write") || !strings.Contains(fallback, "main.go") {
		t.Fatalf("fallback lost failure or repeated bulk: %s", fallback)
	}
}

func TestCriticalTextDropsBulkPayloadAndPreservesOutcome(t *testing.T) {
	t.Parallel()
	text, err := CriticalText(map[string]any{
		"result": map[string]any{
			"status": "applied",
			"diff":   strings.Repeat("large diff", 512),
			"source": map[string]any{"file": "main.go", "symbol": "Rename"},
		},
	}, "formatter body")
	if err != nil {
		t.Fatalf("CriticalText() error = %v", err)
	}
	if !strings.Contains(text, `"status":"applied"`) || !strings.Contains(text, `"source":{"file":"main.go","symbol":"Rename"}`) {
		t.Fatalf("critical projection omitted the outcome or source identity: %s", text)
	}
	if strings.Contains(text, "large diff") || !strings.HasSuffix(text, "\n\nformatter body") {
		t.Fatalf("critical projection retained bulk content or dropped formatter output: %s", text)
	}
}

func TestCriticalTextSummarizesOutlinesAndCollectionCounts(t *testing.T) {
	t.Parallel()
	bulkValue := strings.Repeat("large package payload", 128)
	structured := map[string]any{"result": map[string]any{
		"scope":    map[string]any{"kind": "workspace", "path": ".", "complete": true},
		"modules":  []any{map[string]any{"path": ".", "module_path": "example.com/root"}},
		"packages": []any{map[string]any{"path": "pkg", "name": "pkg", "synopsis": bulkValue}},
	}}
	body := `{"packages":[{"synopsis":"` + bulkValue + `"}]}`
	text, err := CriticalText(structured, body)
	if err != nil {
		t.Fatalf("CriticalText() error = %v", err)
	}
	if !strings.Contains(text, `"scope":{"complete":true,"kind":"workspace","path":"."}`) {
		t.Fatalf("outline scope missing from critical projection: %s", text)
	}
	if !strings.Contains(text, "1 module, 1 package") || strings.Contains(text, bulkValue) || strings.Contains(text, `"modules":`) || strings.Contains(text, `"packages":`) {
		t.Fatalf("outline payload was duplicated into critical text: %s", text)
	}
	var full map[string]any
	encoded, err := json.Marshal(structured)
	if err != nil {
		t.Fatalf("marshal original structured result: %v", err)
	}
	if decodeErr := json.Unmarshal(encoded, &full); decodeErr != nil {
		t.Fatalf("decode original structured result: %v", decodeErr)
	}
	result := full["result"].(map[string]any)
	if len(result["modules"].([]any)) != 1 || result["packages"].([]any)[0].(map[string]any)["synopsis"] != bulkValue {
		t.Fatalf("original structured result was modified: %#v", result)
	}

	ordinaryText, err := CriticalText(map[string]any{"result": map[string]any{
		"scope": map[string]any{"kind": "workspace_active_build", "path": ".", "complete": true},
		"files": []any{"user-data.txt"}, "symbols": []any{"UserDefined"},
	}}, "")
	if err != nil {
		t.Fatalf("CriticalText() ordinary result error = %v", err)
	}
	if !strings.Contains(ordinaryText, `"files_count":1`) || !strings.Contains(ordinaryText, `"symbols_count":1`) {
		t.Fatalf("collection counts were omitted: %s", ordinaryText)
	}
}

func TestCriticalTextFamiliesRemainActionableWithoutFormatterJSON(t *testing.T) {
	t.Parallel()
	target := map[string]any{"file": "main.go", "symbol": "Value", "expected_revision": "rev-1"}
	cases := []struct {
		name     string
		result   map[string]any
		required []string
	}{
		{"lookup", map[string]any{"ambiguous": true, "candidates": []any{target}}, []string{"candidates", "rev-1"}},
		{"inspect", map[string]any{"inspection": "func Value() {\n println(1)\n}", "continuations": []any{target}}, []string{"inspection_bytes", "rev-1"}},
		{"references", map[string]any{"usages": []any{map[string]any{"file": "caller.go", "line": 19}}, "complete": false}, []string{"usages_count", "false"}},
		{"edit", map[string]any{"status": "applied", "application_state": "persisted", "edit_receipt": map[string]any{"continuations": []any{target}}, "diff": strings.Repeat("bulk-diff", 100)}, []string{"persisted", "rev-1"}},
		{"rename", map[string]any{"status": "renamed", "application_state": "persisted", "active": []any{"one"}, "active_count": 500, "diagnostics": map[string]any{"introduced": []any{"main.go:2: new error"}, "resolved": []any{"old error"}, "net_delta": 0}}, []string{"active_count", "introduced_count", "resolved_count", "0"}},
		{"stage", map[string]any{"stage_id": "stage-opaque", "steps": []any{map[string]any{"step_id": "step-opaque", "status": "ok"}}}, []string{"stage-opaque", "step-opaque", "ok"}},
		{"commit", map[string]any{"stage_id": "stage-opaque", "publication": map[string]any{"status": "partial", "error": "publish failed"}}, []string{"stage-opaque", "partial", "publish failed"}},
		{"verify", map[string]any{"failure": "compiler failed", "diagnostics": []any{"main.go:5: undefined: Missing"}, "coverage": []any{map[string]any{"status": "failed"}}}, []string{"compiler failed", "diagnostics_count", "failed"}},
		{"top-level diagnostics", map[string]any{"diagnostic_delta": map[string]any{"net_delta": 1, "introduced": []any{"main.go:5: undefined: Missing"}, "resolved": []any{}}, "diagnostic_report": map[string]any{"report_id": "report-1", "remaining_count": 4}}, []string{"net_delta", "introduced_count", "resolved_count", "report-1", "remaining_count"}},
		{"staged diagnostic candidates", map[string]any{"status": "applied_with_diagnostics", "diagnostic_delta": map[string]any{"net_delta": 1, "candidates": []any{map[string]any{"diagnostic": "full message", "candidate_step_ids": []any{"step-1"}}}}}, []string{"applied_with_diagnostics", "net_delta", "candidates_count"}},
		{"feedback", map[string]any{"status": "draft_only", "date": "2026-10-15", "report": map[string]any{"intent": "large"}, "markdown": strings.Repeat("private draft", 64)}, []string{"draft_only", "markdown_bytes"}},
		{"maven logs", map[string]any{"log_tail": strings.Repeat("large log", 64), "stdout": strings.Repeat("large stdout", 64), "stderr": strings.Repeat("large stderr", 64)}, []string{"log_tail_bytes", "stdout_bytes", "stderr_bytes"}},
		{"workspace receipts", map[string]any{"captured_files": []any{"a.go", "b.go"}, "undo": map[string]any{"restored_files": []any{"a.go"}, "restored_file_count": 1}}, []string{"captured_files_count", "restored_files_count", "restored_file_count"}},
		{"control", map[string]any{"status": "reloading"}, []string{"reloading"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, err := json.Marshal(tc.result)
			if err != nil {
				t.Fatal(err)
			}
			text, err := CriticalText(map[string]any{"result": tc.result}, "{\"formatter-only\":true}")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(text, "formatter-only") || strings.Contains(text, "bulk-diff") || strings.Contains(text, "undefined: Missing") || strings.Contains(text, "private draft") || strings.Contains(text, "large log") || strings.Contains(text, "caller.go") || strings.Contains(text, "one\"") {
				t.Fatalf("duplicate or bulk formatter output: %s", text)
			}
			for _, required := range tc.required {
				if !strings.Contains(text, required) {
					t.Errorf("missing actionable %q in %s", required, text)
				}
			}
			if len(text) > 2048 {
				t.Errorf("small outcome exceeded 2 KiB text budget: %d", len(text))
			}
			after, err := json.Marshal(tc.result)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("presentation changed canonical data")
			}
		})
	}
}

func TestCriticalTextOmitsUnavailableDiagnosticDelta(t *testing.T) {
	t.Parallel()
	text, err := CriticalText(map[string]any{
		"diagnostic_status": "unavailable",
		"diagnostic_delta":  map[string]any{"net_delta": 0, "introduced": []any{}, "resolved": []any{}},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `"diagnostic_status":"unavailable"`) || strings.Contains(text, "diagnostic_delta") || strings.Contains(text, "introduced_count") || strings.Contains(text, "resolved_count") {
		t.Fatalf("unavailable diagnostic delta was presented as zero: %s", text)
	}
}

func TestCriticalTextDropsReadPayloadButKeepsActionableSelectors(t *testing.T) {
	t.Parallel()
	source := strings.Repeat("requested source\n", 100)
	structured := map[string]any{"result": map[string]any{"inspection": source, "continuations": []any{map[string]any{"file": "main.go", "symbol": "Value", "expected_revision": "rev-2"}}}}
	text, err := CriticalText(structured, strings.Repeat("formatter duplicate", 100))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "requested source") || strings.Contains(text, "formatter duplicate") || !strings.Contains(text, "rev-2") || !strings.Contains(text, "inspection_bytes") {
		t.Fatalf("read payload was not compacted or selector was lost: %s", text)
	}
	if structured["result"].(map[string]any)["inspection"] != source {
		t.Fatal("canonical structured inspection was modified")
	}
}

func TestCriticalTextKeepsReadNavigationListingInTextOnly(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		result    map[string]any
		body      string
		listing   string
		critical  string
		preserved []string
	}{
		{
			name: "outline",
			result: map[string]any{
				"summary":    "Declaration outline for \"main.go\": 1 file, 1 symbol.",
				"scope":      map[string]any{"kind": "selected_file", "path": "main.go", "complete": true, "limitations": []any{"limited fixture scope"}},
				"file_count": 1, "symbol_count": 1,
			},
			body:    "Declaration outline of main.go (1 file).\nfunction sample.Keep in main.go:L3-L3; selector L3:C6-L3:C10\nsignature: func Keep()",
			listing: "function sample.Keep", critical: "1 file, 1 symbol", preserved: []string{`"symbol_count":1`, `"limitations":["limited fixture scope"]`},
		},
		{
			name: "references",
			result: map[string]any{
				"summary":         "References to sample.Keep: 2 locations in 1 file.",
				"scope":           map[string]any{"kind": "module", "path": ".", "complete": true},
				"symbol":          map[string]any{"name": "Keep", "qualified_name": "sample.Keep", "kind": "function", "file": "main.go"},
				"reference_count": 2, "file_count": 1,
			},
			body:    "References to sample.Keep (2 locations in 1 file).\nmain.go:L5:C20 in Caller",
			listing: "in Caller", critical: "2 locations in 1 file", preserved: []string{`"reference_count":2`},
		},
		{
			name: "ambiguous lookup",
			result: map[string]any{
				"summary":   "Ambiguous lookup for \"Keep\": 2 candidates.",
				"ambiguous": true, "candidate_count": 2,
			},
			body:    "Ambiguous lookup for \"Keep\": 2 candidates.\nfunction A.Keep at a.go:3:6\nfunction B.Keep at b.go:3:6",
			listing: "A.Keep", critical: "2 candidates", preserved: []string{`"candidate_count":2`},
		},
		{
			name: "empty references",
			result: map[string]any{
				"summary":         "References to sample.Keep: 0 locations in 0 files.",
				"reference_count": 0, "file_count": 0,
			},
			body:    "References to sample.Keep: 0 locations in 0 files.",
			listing: "References to sample.Keep", critical: "0 locations in 0 files", preserved: []string{`"reference_count":0`},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			structured := map[string]any{"result": test.result}
			structuredSummary := test.result["summary"]
			text, err := CriticalText(structured, test.body)
			if err != nil {
				t.Fatalf("CriticalText() error = %v", err)
			}
			if !strings.Contains(text, test.critical) || strings.Count(text, test.listing) != 1 {
				t.Fatalf("critical text omitted counts or duplicated listing: %s", text)
			}
			for _, field := range test.preserved {
				if !strings.Contains(text, field) {
					t.Errorf("critical text dropped preserved field %s: %s", field, text)
				}
			}
			if structured["result"].(map[string]any)["summary"] != structuredSummary {
				t.Fatalf("CriticalText mutated the structured navigation listing: %#v", structured)
			}
		})
	}
}
