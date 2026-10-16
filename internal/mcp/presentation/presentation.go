// Package presentation keeps structured MCP results useful without forwarding bulk payloads.
package presentation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spockz/semantic-editor/internal/operation"
)

const applicationStateField = "application_state"

const (
	diagnosticStatusField = "diagnostic_status"
	diagnosticDeltaField  = "diagnostic_delta"
	introducedField       = "introduced"
	resolvedField         = "resolved"
	suggestionsField      = "suggestions"
)

// CriticalText renders outcome fields before potentially large formatter output.
func CriticalText(structured map[string]any, body string) (string, error) {
	if result, ok := structured["result"]; ok {
		var err error
		body, err = presentationBody(result, body)
		if err != nil {
			return "", err
		}
	}
	presentation := make(map[string]any, len(structured))
	for key, value := range structured {
		if key == "tool_request" {
			continue
		}
		if key == "result" {
			if _, legacyText := value.(string); legacyText {
				continue
			}
			essential, err := essentialPresentationValue(value)
			if err != nil {
				return "", fmt.Errorf("project critical text result: %w", err)
			}
			presentation[key] = essential
			if hasCompactedSource(essential) {
				body = ""
			}
			continue
		}
		presentation[key] = value
	}
	essential, err := essentialPresentationValue(presentation)
	if err != nil {
		return "", fmt.Errorf("compact content details: %w", err)
	}
	presentation = essential.(map[string]any)
	if _, typed := presentation["result"].(map[string]any); typed {
		presentation["details"] = "Full details are in structuredContent.result."
		if report, ok := presentation["diagnostic_report"].(map[string]any); ok && report["tool"] == "semantic_diagnostics" {
			presentation["details"] = "Full details are in structuredContent.result; use diagnostic_report.tool and arguments to page captured diagnostics."
		}
	}
	ordered := orderedPresentationValue(presentation, "")
	encoded, err := json.Marshal(ordered)
	if err != nil {
		return "", fmt.Errorf("encode critical text projection: %w", err)
	}
	if len(presentation) == 0 {
		return body, nil
	}
	text := "Critical result (JSON): " + string(encoded)
	if body != "" && body != structured["error"] {
		text += "\n\n" + body
	}
	return text, nil
}

func essentialPresentationValue(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode structured result: %w", err)
	}
	var projected any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&projected); err != nil {
		return nil, fmt.Errorf("decode structured result: %w", err)
	}
	result, ok := projected.(map[string]any)
	if !ok {
		return projected, nil
	}
	if err := sanitizeBatchPresentation(result); err != nil {
		return nil, err
	}
	compactPresentationDetails(result)
	sanitizePresentationMap(result)
	return result, nil
}

func sanitizeBatchPresentation(result map[string]any) error {
	if results, batch := result["results"].([]any); batch {
		if _, isBatchResponse := result["final_diff"]; isBatchResponse {
			delete(result, "final_diff")
			for _, item := range results {
				if err := sanitizeBatchItem(item); err != nil {
					return err
				}
			}
			if verification, ok := result["verification"].(map[string]any); ok {
				sanitizeVerificationPresentation(verification)
			}
		}
	}
	return nil
}

func sanitizeBatchItem(item any) error {
	child, ok := item.(map[string]any)
	if !ok {
		return nil
	}
	delete(child, "diff")
	working, ok := child["result"].(map[string]any)
	if !ok {
		return nil
	}
	deleteBatchWorkingContinuations(working)
	essential, err := essentialPresentationValue(working)
	if err != nil {
		return fmt.Errorf("project batch child result: %w", err)
	}
	child["result"] = essential
	return nil
}

func sanitizeVerificationPresentation(verification map[string]any) {
	delete(verification, "diagnostics")
	for _, reportName := range []string{"normalize", "checks"} {
		report, ok := verification[reportName].(map[string]any)
		if !ok {
			continue
		}
		hooks, ok := report["Results"].([]any)
		if !ok {
			continue
		}
		for _, hookValue := range hooks {
			hook, ok := hookValue.(map[string]any)
			if ok {
				delete(hook, "Stdout")
				delete(hook, "Stderr")
				delete(hook, "Diagnostics")
			}
		}
	}
}

func sanitizePresentationMap(result map[string]any) {
	if isOutlinePresentation(result) {
		for _, key := range []string{"files", "packages", "modules", "symbols"} {
			delete(result, key)
		}
	}
	if _, bulkDiff := result["diff"].(string); bulkDiff {
		delete(result, "diff")
	}
	_, hasState := result[applicationStateField]
	_, hasReceipt := result["edit_receipt"]
	if hasState || hasReceipt {
		for _, key := range []string{"inspection", "source", "body", "log", "logs"} {
			if value, ok := result[key].(string); ok && isLargeTextPayload(value) {
				delete(result, key)
			}
		}
	}
	delete(result, "read_set")
	for _, key := range []string{"edit_receipt", "receipt"} {
		if receipt, ok := result[key].(map[string]any); ok {
			delete(receipt, "read_set")
			if _, bulkText := receipt["diff"].(string); bulkText {
				delete(receipt, "diff")
			}
		}
	}
}

func outlineSummary(value any) (string, bool, error) {
	if result, ok := value.(map[string]any); ok {
		if _, hasScope := result["scope"]; !hasScope {
			return "", false, nil
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", false, fmt.Errorf("encode outline result: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return "", false, fmt.Errorf("decode outline result: %w", err)
	}
	result, ok := decoded.(map[string]any)
	if !ok {
		return "", false, nil
	}
	if !isOutlinePresentation(result) {
		return "", false, nil
	}
	scope := result["scope"].(map[string]any)
	path, _ := scope["path"].(string)
	if modules, ok := result["modules"].([]any); ok {
		packages := result["packages"].([]any)
		return fmt.Sprintf("Workspace outline for %q: %d %s, %d %s.", path, len(modules), plural(len(modules), "module", "modules"), len(packages), plural(len(packages), "package", "packages")), true, nil
	}
	if files, ok := result["files"].([]any); ok {
		symbolCount := 0
		for _, value := range files {
			if file, ok := value.(map[string]any); ok {
				if symbols, ok := file["symbols"].([]any); ok {
					symbolCount += len(symbols)
				}
			}
		}
		return fmt.Sprintf("Declaration outline for %q: %d %s, %d %s.", path, len(files), plural(len(files), "file", "files"), symbolCount, plural(symbolCount, "symbol", "symbols")), true, nil
	}
	symbols := result["symbols"].([]any)
	return fmt.Sprintf("Declaration outline for %q: %d %s.", path, len(symbols), plural(len(symbols), "symbol", "symbols")), true, nil
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func isOutlinePresentation(result map[string]any) bool {
	scope, ok := result["scope"].(map[string]any)
	if !ok {
		return false
	}
	kind, _ := scope["kind"].(string)
	switch kind {
	case "workspace":
		_, modules := result["modules"].([]any)
		_, packages := result["packages"].([]any)
		return modules && packages
	case "directory", "selected_file":
		if _, files := result["files"].([]any); files {
			return true
		}
		_, symbols := result["symbols"].([]any)
		return symbols
	default:
		return false
	}
}

func isLargeTextPayload(value string) bool {
	return len(value) >= 256 || strings.ContainsAny(value, "\r\n")
}

type orderedJSONField struct {
	name  string
	value any
}

type orderedJSONObject []orderedJSONField

func orderedPresentationValue(value any, path string) any {
	switch current := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			left, leftOK := presentationFieldPriority(keys[i])
			right, rightOK := presentationFieldPriority(keys[j])
			if leftOK != rightOK {
				return leftOK
			}
			if leftOK && left != right {
				return left < right
			}
			return keys[i] < keys[j]
		})
		fields := make(orderedJSONObject, 0, len(keys))
		for _, key := range keys {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			fields = append(fields, orderedJSONField{name: key, value: orderedPresentationValue(current[key], childPath)})
		}
		return fields
	case []any:
		items := make([]any, len(current))
		for i := range current {
			items[i] = orderedPresentationValue(current[i], path)
		}
		return items
	default:
		return value
	}
}

func presentationFieldPriority(key string) (int, bool) {
	value, ok := presentationFieldPriorities[key]
	return value, ok
}

func deleteBatchWorkingContinuations(result map[string]any) {
	delete(result, "continuations")
	delete(result, "function_target")
	for _, key := range []string{"receipt", "edit_receipt"} {
		if receipt, ok := result[key].(map[string]any); ok {
			delete(receipt, "continuations")
		}
	}
}

// CriticalTextFallback preserves operation errors when structured projection fails.
func CriticalTextFallback(structured map[string]any, body string, cause error) string {
	if isLargeTextPayload(body) || json.Valid([]byte(body)) {
		body = ""
	}
	fallback := map[string]any{
		applicationStateField: "unknown",
		"presentation_error":  cause.Error(),
	}
	if operationErr, ok := structured["error"]; ok {
		fallback["error"] = operationErr
	}
	if location, ok := structured["location"]; ok {
		fallback["location"] = location
	}
	if text, err := CriticalText(fallback, body); err == nil {
		return text
	}
	return fmt.Sprintf("Critical result projection unavailable: %v\n\n%s", cause, body)
}

func (fields orderedJSONObject) MarshalJSON() ([]byte, error) {
	var result bytes.Buffer
	result.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			result.WriteByte(',')
		}
		name, err := json.Marshal(field.name)
		if err != nil {
			return nil, fmt.Errorf("encode critical field name: %w", err)
		}
		value, err := json.Marshal(field.value)
		if err != nil {
			return nil, fmt.Errorf("encode critical field %q: %w", field.name, err)
		}
		result.Write(name)
		result.WriteByte(':')
		result.Write(value)
	}
	result.WriteByte('}')
	return result.Bytes(), nil
}

// CriticalNativeFields returns schema-validated diagnostic metadata from native results.
func CriticalNativeFields(value any) (map[string]any, error) {
	if edit, ok := operation.DiagnosticFileEditResult(value); ok {
		value = edit
	}
	switch result := value.(type) {
	case operation.FileEditRes:
		fields := map[string]any{}
		if result.DiagnosticStatus != "" {
			fields[diagnosticStatusField] = result.DiagnosticStatus
		}
		if result.HasDelta {
			fields[diagnosticDeltaField] = result.Delta
		}
		if len(fields) == 0 {
			return fields, nil
		}
		encoded, err := json.Marshal(fields)
		if err != nil {
			return nil, fmt.Errorf("encode native diagnostic metadata: %w", err)
		}
		var normalized map[string]any
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		if err := decoder.Decode(&normalized); err != nil {
			return nil, fmt.Errorf("decode native diagnostic metadata: %w", err)
		}
		if delta, ok := normalized[diagnosticDeltaField].(map[string]any); ok {
			for _, key := range []string{introducedField, resolvedField, suggestionsField} {
				if value, exists := delta[key]; exists && value == nil {
					delta[key] = []any{}
				}
			}
		}
		if err := operation.ValidateObject(nativeDiagnosticFieldsSchema(), normalized); err != nil {
			return nil, fmt.Errorf("validate native diagnostic metadata: %w", err)
		}
		return normalized, nil
	default:
		return map[string]any{}, nil
	}
}

func nativeDiagnosticFieldsSchema() operation.ValueSchema {
	stringArray := operation.ValueSchema{Kind: operation.SchemaArray, Items: &operation.ValueSchema{Kind: operation.SchemaString}}
	delta := operation.ValueSchema{
		Kind: operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{
			"net_delta":      {Kind: operation.SchemaInteger},
			introducedField:  stringArray,
			resolvedField:    stringArray,
			suggestionsField: stringArray,
		},
		Required: []string{"net_delta", introducedField, resolvedField},
	}
	return operation.ValueSchema{
		Kind: operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{
			diagnosticStatusField: {Kind: operation.SchemaString},
			diagnosticDeltaField:  delta,
		},
	}
}

var presentationFieldPriorities = map[string]int{
	"status": 0, applicationStateField: 1, "error": 2, "error_projection_failure": 3, "presentation_error": 4, "native_metadata_unavailable": 5,
	"safe_delete": 6, "refusal": 7, "limitation": 8, "continuation_unavailable": 9, "diagnostic_report": 9, diagnosticStatusField: 10,
	"failure": 11, "formatted": 12, "scope": 13, "report_id": 13, "category": 13, "total": 13, "offset": 13, "limit": 13, "next_offset": 13, "contains": 13,
	"function_target": 14, "result": 14, "receipt": 15, "edit_receipt": 16, "continuations": 17, "file": 18, "symbol": 19, "in_function": 20,
	"verification": 21, "Completed": 22, "Partial": 23, "Failure": 24, "Results": 25, "Status": 26, "Error": 27, "results": 28, "checks_run": 29,
	"changes": 30, "changed_files": 31, diagnosticDeltaField: 32,
}

func compactPresentationDetails(result map[string]any) {
	_, hasMarkdown := result["markdown"]
	switch result[diagnosticStatusField] {
	case "unavailable", "deferred", "failed", "unknown":
		delete(result, diagnosticDeltaField)
	}
	for key, value := range result {
		switch key {
		case "continuations", "function_target", "type_target", "candidates", "refusal", "safe_delete":
			continue
		}
		if compactPresentationField(result, key, value, hasMarkdown) {
			continue
		}
		switch nested := value.(type) {
		case map[string]any:
			if key == diagnosticDeltaField {
				if candidates, ok := nested["candidates"].([]any); ok {
					nested["candidates_count"] = len(candidates)
					delete(nested, "candidates")
				}
			}
			compactPresentationDetails(nested)
		case []any:
			for _, item := range nested {
				if record, ok := item.(map[string]any); ok {
					compactPresentationDetails(record)
				}
			}
		}
	}
}

func compactPresentationField(result map[string]any, key string, value any, hasMarkdown bool) bool {
	bulk := false
	switch key {
	case "inspection", "body", "log", "logs", "markdown", "diff", "final_diff", "read_set", "Stdout", "Stderr", "stdout", "stderr", "log_tail":
		bulk = true
	case "source":
		_, bulk = value.(string)
	case "references", "usages", "locations", "records", "items", "active", "introduced", "resolved", "suggestions", "changes", "changed_files", "files", "symbols", "modules", "packages", "Diagnostics", "diagnostics", "captured_files", "restored_files", "paths", "Paths", "ScopePaths", "Changed":
		_, bulk = value.([]any)
	case "report":
		bulk = hasMarkdown
	}
	if !bulk {
		return false
	}
	switch details := value.(type) {
	case []any:
		if _, known := result[key+"_count"]; !known {
			result[key+"_count"] = len(details)
		}
	case string:
		result[key+"_bytes"] = len(details)
	}
	delete(result, key)
	return true
}

func presentationBody(result any, body string) (string, error) {
	if summary, isOutline, err := outlineSummary(result); err != nil {
		return "", fmt.Errorf("summarize outline result: %w", err)
	} else if isOutline {
		return summary, nil
	}
	if _, legacy := result.(string); !legacy && (isLargeTextPayload(body) || json.Valid([]byte(body))) {
		return "", nil
	}
	return body, nil
}

func hasCompactedSource(value any) bool {
	result, ok := value.(map[string]any)
	if !ok {
		return false
	}
	_, inspection := result["inspection_bytes"]
	_, source := result["source_bytes"]
	return inspection || source
}
