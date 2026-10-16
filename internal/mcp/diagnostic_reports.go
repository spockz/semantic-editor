// Package mcp exposes registry-defined semantic operations over the MCP stdio transport.
// WHY: This file stores immutable diagnostic snapshots and serves bounded pages without rerunning analysis.
package mcp

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
)

const (
	diagnosticReportMaxEntries        = 16
	diagnosticReportMaxBytes          = 16 << 20
	diagnosticReportPageSize          = 50
	diagnosticReportMaxPage           = 100
	diagnosticWorkspaceRootField      = "workspace_root"
	diagnosticReportStatusUnavailable = "unavailable"
)

type diagnosticReport struct {
	WorkspaceRoot string
	Coverage      []operation.LanguageCoverage
	Status        string
	Active        []backend.Diagnostic
	Introduced    []backend.Diagnostic
	Resolved      []backend.Diagnostic
	Suggestions   []backend.Diagnostic
}

type diagnosticReportEntry struct {
	report diagnosticReport
	size   int
}

type diagnosticReportCache struct {
	mu      sync.Mutex
	entries map[string]diagnosticReportEntry
	order   []string
	bytes   int
}

func newDiagnosticReportCache() *diagnosticReportCache {
	return &diagnosticReportCache{entries: make(map[string]diagnosticReportEntry)}
}

func (c *diagnosticReportCache) store(report diagnosticReport) (string, error) {
	if c == nil {
		return "", errors.New("diagnostic report cache is unavailable")
	}
	snapshot := cloneDiagnosticReport(report)
	if (snapshot.Status == "available" || snapshot.Status == "partial") && snapshot.Active == nil {
		snapshot.Active = []backend.Diagnostic{}
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("measure diagnostic report: %w", err)
	}
	if len(encoded) > diagnosticReportMaxBytes {
		return "", fmt.Errorf("diagnostic report exceeds the %d byte cache limit", diagnosticReportMaxBytes)
	}
	id := rand.Text()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]diagnosticReportEntry)
	}
	for len(c.order) >= diagnosticReportMaxEntries || c.bytes+len(encoded) > diagnosticReportMaxBytes {
		if len(c.order) == 0 {
			return "", errors.New("diagnostic report cannot fit in the cache")
		}
		oldest := c.order[0]
		c.order = c.order[1:]
		c.bytes -= c.entries[oldest].size
		delete(c.entries, oldest)
	}
	c.entries[id] = diagnosticReportEntry{report: snapshot, size: len(encoded)}
	c.order = append(c.order, id)
	c.bytes += len(encoded)
	return id, nil
}

type diagnosticReportPage struct {
	WorkspaceRoot string
	Status        string
	Available     bool
	Total         int
	Items         []backend.Diagnostic
}

func (c *diagnosticReportCache) page(id, category, contains string, offset, limit int) (diagnosticReportPage, bool) {
	if c == nil {
		return diagnosticReportPage{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[id]
	if !ok {
		return diagnosticReportPage{}, false
	}
	report := entry.report
	items := reportCategory(report, category)
	page := diagnosticReportPage{WorkspaceRoot: report.WorkspaceRoot, Status: report.Status}
	if items == nil {
		return page, true
	}
	page.Available = true
	if contains == "" {
		page.Total = len(items)
		if offset <= page.Total {
			count := min(limit, page.Total-offset)
			page.Items = cloneBackendDiagnostics(items[offset : offset+count])
		}
	} else {
		needle := strings.ToLower(contains)
		for _, item := range items {
			if !strings.Contains(strings.ToLower(item.Message), needle) && (item.Location == nil || !strings.Contains(strings.ToLower(item.Location.URI), needle)) {
				continue
			}
			if page.Total >= offset && len(page.Items) < limit {
				cloned := item
				if item.Location != nil {
					location := *item.Location
					cloned.Location = &location
				}
				page.Items = append(page.Items, cloned)
			}
			page.Total++
		}
	}
	if page.Items == nil {
		page.Items = []backend.Diagnostic{}
	}
	return page, true
}

func cloneDiagnosticReport(report diagnosticReport) diagnosticReport {
	report.Coverage = append([]operation.LanguageCoverage(nil), report.Coverage...)
	for i := range report.Coverage {
		report.Coverage[i].Paths = append([]string(nil), report.Coverage[i].Paths...)
		report.Coverage[i].BuildExclusions = append([]string(nil), report.Coverage[i].BuildExclusions...)
	}
	report.Active = cloneBackendDiagnostics(report.Active)
	report.Introduced = cloneBackendDiagnostics(report.Introduced)
	report.Resolved = cloneBackendDiagnostics(report.Resolved)
	report.Suggestions = cloneBackendDiagnostics(report.Suggestions)
	return report
}

func cloneBackendDiagnostics(diagnostics []backend.Diagnostic) []backend.Diagnostic {
	if diagnostics == nil {
		return nil
	}
	cloned := make([]backend.Diagnostic, len(diagnostics))
	copy(cloned, diagnostics)
	for i := range cloned {
		if cloned[i].Location != nil {
			location := *cloned[i].Location
			cloned[i].Location = &location
		}
	}
	return cloned
}

func (s *Server) publishDiagnosticReport(report diagnosticReport) map[string]any {
	metadata := diagnosticReportMetadata(report)
	if report.Status != "available" && report.Status != "partial" {
		return metadata
	}
	if s.diagnosticReports == nil {
		metadata["status"] = diagnosticReportStatusUnavailable
		metadata["reason"] = "diagnostic report cache is unavailable"
		return metadata
	}
	id, err := s.diagnosticReports.store(report)
	if err != nil {
		metadata["status"] = diagnosticReportStatusUnavailable
		metadata["reason"] = err.Error()
		return metadata
	}
	metadata["report_id"] = id
	metadata["tool"] = "semantic_diagnostics"
	metadata["arguments"] = map[string]any{"report_id": id, "category": "active", "offset": 0, "limit": diagnosticReportPageSize}
	return metadata
}

func diagnosticReportMetadata(report diagnosticReport) map[string]any {
	coverage := report.Coverage
	if coverage == nil {
		coverage = []operation.LanguageCoverage{}
	}
	metadata := map[string]any{"status": report.Status, diagnosticWorkspaceRootField: report.WorkspaceRoot, "coverage": coverage}
	if report.Status == "available" || report.Status == "partial" {
		metadata["remaining_count"] = len(report.Active)
		if report.Introduced != nil {
			metadata["introduced_count"] = len(report.Introduced)
		}
		if report.Resolved != nil {
			metadata["resolved_count"] = len(report.Resolved)
		}
		if report.Suggestions != nil {
			metadata["suggestion_count"] = len(report.Suggestions)
		}
	}
	return metadata
}

func diagnosticReportMetadataSchema() map[string]any {
	properties := map[string]any{
		"status":                     map[string]any{"type": "string", "enum": []string{"available", "partial", diagnosticReportStatusUnavailable, "deferred"}},
		diagnosticWorkspaceRootField: map[string]any{"type": "string"},
		"coverage":                   map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
		"remaining_count":            map[string]any{"type": "integer", "minimum": 0},
		"introduced_count":           map[string]any{"type": "integer", "minimum": 0},
		"resolved_count":             map[string]any{"type": "integer", "minimum": 0},
		"suggestion_count":           map[string]any{"type": "integer", "minimum": 0},
		"reason":                     map[string]any{"type": "string"},
		"report_id":                  map[string]any{"type": "string"},
		"tool":                       map[string]any{"type": "string", "const": "semantic_diagnostics"},
		"arguments": map[string]any{"type": "object", schemaPropertiesKey: map[string]any{
			"report_id": map[string]any{"type": "string"},
			"category":  map[string]any{"type": "string", "const": "active"},
			"offset":    map[string]any{"type": "integer", "const": 0},
			"limit":     map[string]any{"type": "integer", "const": diagnosticReportPageSize},
		}, "required": []string{"report_id", "category", "offset", "limit"}, additionalPropertiesKey: false},
	}
	complete := map[string]any{"type": "object", schemaPropertiesKey: properties, "required": []string{"status", diagnosticWorkspaceRootField, "coverage", "remaining_count", "report_id", "tool", "arguments"}, additionalPropertiesKey: false}
	partial := map[string]any{"type": "object", schemaPropertiesKey: properties, "required": []string{"status"}, additionalPropertiesKey: false}
	return map[string]any{"type": "object", "oneOf": []any{
		map[string]any{"allOf": []any{complete, map[string]any{schemaPropertiesKey: map[string]any{"status": map[string]any{"enum": []string{"available", "partial"}}}}}},
		map[string]any{"allOf": []any{partial, map[string]any{schemaPropertiesKey: map[string]any{"status": map[string]any{"enum": []string{diagnosticReportStatusUnavailable, "deferred"}}}}}},
	}}
}

func diagnosticsToolSchema() map[string]any {
	result := map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"report_id":                  map[string]any{"type": "string"},
			diagnosticWorkspaceRootField: map[string]any{"type": "string"},
			"status":                     map[string]any{"type": "string", "enum": []string{"available", "partial"}},
			"category":                   map[string]any{"type": "string", "enum": []string{"active", "introduced", "resolved", "suggestions"}},
			"total":                      map[string]any{"type": "integer", "minimum": 0},
			"offset":                     map[string]any{"type": "integer", "minimum": 0},
			"limit":                      map[string]any{"type": "integer", "minimum": 1, "maximum": diagnosticReportMaxPage, "default": diagnosticReportPageSize},
			"contains":                   map[string]any{"type": "string", schemaDescriptionKey: "Return items whose message or source URI contains this text, ignoring case."},
			"items":                      map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
			"next_offset":                map[string]any{"type": "integer", "minimum": 0},
		},
		"required": []string{"report_id", diagnosticWorkspaceRootField, "status", "category", "total", "offset", "limit", "items"},
	}
	return map[string]any{
		"name":               "semantic_diagnostics",
		toolFieldDescription: "Use this tool instead of rerunning verification when reading diagnostics from a completed operation. Page the captured snapshot by category, filter, offset, and limit without analysis or refresh. For an expired report, use a newer operation report or semantic_verify with check_only=true to capture fresh diagnostics.",
		toolFieldInputSchema: map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"report_id": map[string]any{"type": "string"},
				"category":  map[string]any{"type": "string", "enum": []string{"active", "introduced", "resolved", "suggestions"}, "default": "active"},
				"offset":    map[string]any{"type": "integer", "minimum": 0, "default": 0},
				"limit":     map[string]any{"type": "integer", "minimum": 1, "maximum": diagnosticReportMaxPage, "default": diagnosticReportPageSize},
				"contains":  map[string]any{"type": "string", schemaDescriptionKey: "Return items whose message or source URI contains this text, ignoring case."},
			},
			"required":              []string{"report_id"},
			additionalPropertiesKey: false,
		},
		toolFieldOutputSchema: standardOutputSchema(result, nil),
		"annotations":         map[string]any{"readOnlyHint": true},
	}
}

type diagnosticReportRequest struct {
	ReportID string          `json:"report_id"`
	Category string          `json:"category"`
	Offset   json.RawMessage `json:"offset"`
	Limit    json.RawMessage `json:"limit"`
	Contains string          `json:"contains,omitempty"`
}

func (s *Server) handleDiagnosticsCall(id json.RawMessage, arguments json.RawMessage) {
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	var request diagnosticReportRequest
	if err := decoder.Decode(&request); err != nil {
		s.sendToolError(id, fmt.Sprintf("invalid semantic_diagnostics arguments: %v", err), err)
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		s.sendToolError(id, fmt.Sprintf("invalid semantic_diagnostics arguments: %v", err), err)
		return
	}
	if request.ReportID == "" {
		s.sendToolError(id, "report_id is required")
		return
	}
	if request.Category == "" {
		request.Category = "active"
	}
	offset, err := diagnosticPageInteger(request.Offset, 0)
	if err != nil || offset < 0 {
		s.sendToolError(id, "offset must be a nonnegative integer")
		return
	}
	limit, err := diagnosticPageInteger(request.Limit, diagnosticReportPageSize)
	if err != nil || limit < 1 || limit > diagnosticReportMaxPage {
		s.sendToolError(id, fmt.Sprintf("limit must be an integer between 1 and %d", diagnosticReportMaxPage))
		return
	}
	if request.Category != "active" && request.Category != "introduced" && request.Category != "resolved" && request.Category != "suggestions" {
		s.sendToolError(id, fmt.Sprintf("unknown diagnostic category %q", request.Category))
		return
	}
	page, ok := s.diagnosticReports.page(request.ReportID, request.Category, request.Contains, offset, limit)
	if !ok {
		s.sendToolError(id, "diagnostic report is unknown or expired; use a newer operation report or semantic_verify with check_only=true for fresh diagnostics")
		return
	}
	if !page.Available {
		s.sendToolError(id, fmt.Sprintf("diagnostic category %q was not captured in this report", request.Category))
		return
	}
	if offset > page.Total {
		s.sendToolError(id, fmt.Sprintf("offset %d exceeds diagnostic count %d", offset, page.Total))
		return
	}
	end := offset + len(page.Items)
	result := map[string]any{
		"report_id":                  request.ReportID,
		diagnosticWorkspaceRootField: page.WorkspaceRoot,
		"status":                     page.Status,
		"category":                   request.Category,
		"total":                      page.Total,
		"offset":                     offset,
		"limit":                      limit,
		"items":                      page.Items,
	}
	if end < page.Total {
		result["next_offset"] = end
	}
	if request.Contains != "" {
		result["contains"] = request.Contains
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		s.sendToolError(id, fmt.Sprintf("encode semantic_diagnostics result: %v", err), err)
		return
	}
	s.sendToolSuccessWithStructuredContent(id, string(encoded), map[string]any{"result": result})
}

func diagnosticPageInteger(raw json.RawMessage, defaultValue int) (int, error) {
	if raw == nil {
		return defaultValue, nil
	}
	if bytes.Equal(raw, []byte("null")) {
		return 0, errors.New("null is not an integer")
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func reportCategory(report diagnosticReport, category string) []backend.Diagnostic {
	switch category {
	case "active":
		return report.Active
	case "introduced":
		return report.Introduced
	case "resolved":
		return report.Resolved
	default:
		return report.Suggestions
	}
}
