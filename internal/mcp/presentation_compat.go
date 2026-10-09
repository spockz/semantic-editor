// Package mcp retains transport-facing presentation entry points while the projection logic lives in its own cacheable package.
package mcp

import (
	"semedit/internal/mcp/presentation"
)

const (
	applicationStateField = "application_state"
	diagnosticStatusField = "diagnostic_status"
	diagnosticDeltaField  = "diagnostic_delta"
	introducedField       = "introduced"
	resolvedField         = "resolved"
	suggestionsField      = "suggestions"
)

// CriticalText delegates projection to the independent presentation package.
func CriticalText(structured map[string]any, body string) (string, error) {
	return presentation.CriticalText(structured, body)
}

// CriticalNativeFields delegates native metadata projection to the presentation package.
func CriticalNativeFields(value any) (map[string]any, error) {
	return presentation.CriticalNativeFields(value)
}

func criticalTextFallback(structured map[string]any, body string, cause error) string {
	return presentation.CriticalTextFallback(structured, body, cause)
}
