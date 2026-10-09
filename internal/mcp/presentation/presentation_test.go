// Package presentation tests result projection without compiling MCP dispatch and server integration tests.
package presentation

import (
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
