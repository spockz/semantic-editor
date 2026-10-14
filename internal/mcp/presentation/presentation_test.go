// Package presentation tests result projection without compiling MCP dispatch and server integration tests.
package presentation

import (
	"encoding/json"
	"strings"
	"testing"
)

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

func TestCriticalTextSummarizesOutlinesWithoutPruningUserCollections(t *testing.T) {
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
	if !strings.Contains(ordinaryText, `"files":["user-data.txt"]`) || !strings.Contains(ordinaryText, `"symbols":["UserDefined"]`) {
		t.Fatalf("ordinary user collections were pruned: %s", ordinaryText)
	}
}
