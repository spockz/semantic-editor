// Package mcp tests cached diagnostic retrieval behavior at its public handler boundary.
// WHY: This verifies pagination, filtering, immutability, and explicit cache failures.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

func TestDiagnosticReportCacheSnapshotsAndEvictsOldest(t *testing.T) {
	cache := newDiagnosticReportCache()
	location := &backend.SourceLocation{URI: "file:///work/main.go"}
	report := diagnosticReport{WorkspaceRoot: "/work", Status: "available", Coverage: []operation.LanguageCoverage{{Language: "go", Status: "verified", Paths: []string{"main.go"}}}, Active: []backend.Diagnostic{{Message: "first", Location: location}}}
	firstID, err := cache.store(report)
	if err != nil {
		t.Fatal(err)
	}
	location.URI = "mutated"
	report.Active[0].Message = "mutated"
	report.Coverage[0].Paths[0] = "mutated"
	snapshot, ok := cache.page(firstID, "active", "", 0, 1)
	if !ok {
		t.Fatal("stored report was not retrievable")
	}
	if snapshot.Items[0].Message != "first" || snapshot.Items[0].Location.URI != "file:///work/main.go" || cache.entries[firstID].report.Coverage[0].Paths[0] != "main.go" {
		t.Fatalf("cache did not preserve an immutable snapshot: %#v", snapshot)
	}
	snapshot.Items[0].Message = "caller mutation"
	if got, _ := cache.page(firstID, "active", "", 0, 1); got.Items[0].Message != "first" {
		t.Fatal("retrieval returned cache-owned diagnostic memory")
	}
	for i := range diagnosticReportMaxEntries {
		if _, err := cache.store(diagnosticReport{WorkspaceRoot: "/work", Status: "available", Active: []backend.Diagnostic{{Message: fmt.Sprint(i)}}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := cache.page(firstID, "active", "", 0, 1); ok {
		t.Fatal("oldest report survived FIFO eviction")
	}
}

func TestDiagnosticReportPaginationFilteringAndErrors(t *testing.T) {
	var output bytes.Buffer
	server := &Server{out: &output, diagnosticReports: newDiagnosticReportCache()}
	report := diagnosticReport{WorkspaceRoot: "/work", Status: "partial", Coverage: []operation.LanguageCoverage{{Language: "go", Status: "verified"}}}
	for i := range 3 {
		report.Active = append(report.Active, backend.Diagnostic{Message: []string{"alpha", "beta", "alphabet"}[i]})
	}
	id, err := server.diagnosticReports.store(report)
	if err != nil {
		t.Fatal(err)
	}
	call := func(args string) map[string]any {
		t.Helper()
		output.Reset()
		server.handleDiagnosticsCall(json.RawMessage(`1`), json.RawMessage(args))
		var response map[string]any
		if err := json.Unmarshal(output.Bytes(), &response); err != nil {
			t.Fatalf("decode MCP response: %v (%s)", err, output.String())
		}
		result, ok := response["result"].(map[string]any)
		if !ok {
			t.Fatalf("MCP response omitted tool result: %#v", response)
		}
		return result
	}
	first := call(fmt.Sprintf(`{"report_id":%q,"category":"active","limit":2}`, id))
	if first["isError"] != false {
		t.Fatalf("first page failed: %s", output.String())
	}
	structured := first[structuredContentKey].(map[string]any)["result"].(map[string]any)
	if structured["total"] != float64(3) || structured["next_offset"] != float64(2) || len(structured["items"].([]any)) != 2 {
		t.Fatalf("unexpected first page: %#v", structured)
	}
	structured["items"].([]any)[0].(map[string]any)["message"] = "caller mutation"
	again := call(fmt.Sprintf(`{"report_id":%q,"category":"active","limit":2}`, id))
	againItems := again[structuredContentKey].(map[string]any)["result"].(map[string]any)["items"].([]any)
	if againItems[0].(map[string]any)["message"] != "alpha" {
		t.Fatalf("handler returned cache-owned diagnostic memory: %#v", againItems)
	}
	filtered := call(fmt.Sprintf(`{"report_id":%q,"category":"active","contains":"alpha"}`, id))
	filteredResult := filtered[structuredContentKey].(map[string]any)["result"].(map[string]any)
	if filteredResult["total"] != float64(2) || len(filteredResult["items"].([]any)) != 2 || filteredResult["next_offset"] != nil {
		t.Fatalf("unexpected filtered final page: %#v", filteredResult)
	}
	if got := call(fmt.Sprintf(`{"report_id":%q,"offset":9}`, id)); got["isError"] != true {
		t.Fatalf("out-of-range offset was accepted: %#v", got)
	}
	if got := call(`{"report_id":"missing"}`); got["isError"] != true || !strings.Contains(output.String(), "unknown or expired") {
		t.Fatalf("unknown report was not explicit: %#v", got)
	}
	if got := call(fmt.Sprintf(`{"report_id":%q,"unexpected":true}`, id)); got["isError"] != true {
		t.Fatalf("unknown argument was accepted: %#v", got)
	}
	if got := call(fmt.Sprintf(`{"report_id":%q,"category":"introduced"}`, id)); got["isError"] != true || !strings.Contains(output.String(), "was not captured") {
		t.Fatalf("uncaptured category appeared empty: %#v", got)
	}
	for _, args := range []string{fmt.Sprintf(`{"report_id":%q,"limit":0}`, id), fmt.Sprintf(`{"report_id":%q,"limit":null}`, id), fmt.Sprintf(`{"report_id":%q,"offset":null}`, id)} {
		if got := call(args); got["isError"] != true {
			t.Fatalf("invalid pagination accepted: %s => %#v", args, got)
		}
	}
}

func TestDiagnosticReportKnownZeroAndUnknownDelta(t *testing.T) {
	cache := newDiagnosticReportCache()
	report := diagnosticReport{WorkspaceRoot: "/work", Status: "available"}
	id, err := cache.store(report)
	if err != nil {
		t.Fatal(err)
	}
	active, ok := cache.page(id, "active", "", 0, diagnosticReportPageSize)
	if !ok || !active.Available || active.Total != 0 {
		t.Fatalf("healthy zero-active report unavailable: %#v", active)
	}
	introduced, ok := cache.page(id, "introduced", "", 0, diagnosticReportPageSize)
	if !ok || introduced.Available {
		t.Fatalf("unknown introduced category appeared available: %#v", introduced)
	}
	metadata := diagnosticReportMetadata(report)
	if metadata["remaining_count"] != 0 || metadata["introduced_count"] != nil || metadata["resolved_count"] != nil || metadata["suggestion_count"] != nil {
		t.Fatalf("unknown delta counts appeared as zero: %#v", metadata)
	}
	unknown := diagnosticReportMetadata(diagnosticReport{Status: "unavailable"})
	if unknown["remaining_count"] != nil {
		t.Fatalf("unavailable active count appeared as zero: %#v", unknown)
	}
	knownEmpty := diagnosticReport{WorkspaceRoot: "/work", Status: "available", Introduced: []backend.Diagnostic{}}
	emptyID, err := cache.store(knownEmpty)
	if err != nil {
		t.Fatal(err)
	}
	emptyPage, ok := cache.page(emptyID, "introduced", "", 0, diagnosticReportPageSize)
	if !ok || !emptyPage.Available || emptyPage.Total != 0 || diagnosticReportMetadata(knownEmpty)["introduced_count"] != 0 {
		t.Fatalf("captured zero introduced diagnostics appeared unavailable: %#v", emptyPage)
	}
}

func TestPublishDiagnosticReportRefusesOversizedSnapshot(t *testing.T) {
	server := &Server{diagnosticReports: newDiagnosticReportCache()}
	metadata := server.publishDiagnosticReport(diagnosticReport{WorkspaceRoot: "/work", Status: "available", Active: []backend.Diagnostic{{Message: strings.Repeat("x", diagnosticReportMaxBytes+1)}}})
	if metadata["status"] != "unavailable" || metadata["reason"] == nil || metadata["report_id"] != nil {
		t.Fatalf("oversized report was published: %#v", metadata)
	}
}

func TestMoveFileResponseTextKeepsDiagnosticDetailsPaged(t *testing.T) {
	result := operation.FileEditRes{Detail: "Moved Foo and helpers.", ApplicationState: "published", ChangedFiles: []string{"old/foo.go", "api/moved.go"}, Diff: "diff --git a/old/foo.go", Delta: pipeline.DiagnosticDelta{Introduced: []string{"new issue"}}}
	report := map[string]any{"status": "available", "report_id": "report-1", "remaining_count": 2, "introduced_count": 1, "resolved_count": 0, "suggestion_count": 0}
	text := moveFileResponseText(result, report)
	for _, want := range []string{"Successfully moved file declarations.", "published", "old/foo.go", "api/moved.go", "active 2", "semantic_diagnostics", "report-1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("response omitted %q: %s", want, text)
		}
	}
	for _, omit := range []string{"diff --git", "new issue", "diagnostic_delta"} {
		if strings.Contains(text, omit) {
			t.Fatalf("response included paged detail %q: %s", omit, text)
		}
	}
}

func TestMoveFilePresentationBoundsTextAndRetainsStructuredEvidence(t *testing.T) {
	changedFiles := make([]string, 30)
	changes := make([]operation.FileRevision, len(changedFiles))
	readSet := make([]operation.FileRevision, len(changedFiles))
	for i := range changedFiles {
		changedFiles[i] = fmt.Sprintf("pkg/file_%02d.go", i)
		changes[i] = operation.FileRevision{Path: changedFiles[i], Revision: fmt.Sprintf("revision-marker-%02d", i)}
		readSet[i] = operation.FileRevision{Path: changedFiles[i], Revision: fmt.Sprintf("read-revision-marker-%02d", i)}
	}
	diagnostics := make([]string, 200)
	for i := range diagnostics {
		diagnostics[i] = fmt.Sprintf("diagnostic-marker-%03d", i)
	}
	result := map[string]any{
		"file":          "api/moved.go",
		"changed_files": changedFiles,
		"diff":          strings.Repeat("diff-marker\n", 1000),
		"edit_receipt": map[string]any{
			"schema_version": "semedit.edit-receipt/v1",
			"status":         "persisted",
			"operation":      "move_file",
			"changes":        changes,
			"read_set":       readSet,
			"continuations":  []any{map[string]any{"file": "api/moved.go"}},
		},
	}
	fields := map[string]any{
		"file":   "api/moved.go",
		"result": result,
		"diagnostic_report": map[string]any{
			"status": "available", "report_id": "report-1", "remaining_count": 0,
			"introduced_count": len(diagnostics), "resolved_count": 0, "suggestion_count": 0,
		},
	}
	native := operation.FileEditRes{HasDelta: true, Delta: pipeline.DiagnosticDelta{Introduced: diagnostics, Resolved: []string{}}}
	var output bytes.Buffer
	server := &Server{out: &output}
	server.sendToolResultWithNativeAndPresentation(json.RawMessage(`1`), moveFileResponseText(operation.FileEditRes{ApplicationState: "persisted", ChangedFiles: changedFiles}, fields["diagnostic_report"]), newToolRequestTiming(context.Background(), nil), native, fields, false, true)
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode MCP response: %v (%s)", err, output.String())
	}
	resultEnvelope := response["result"].(map[string]any)
	structured := resultEnvelope[structuredContentKey].(map[string]any)
	structuredResult := structured["result"].(map[string]any)
	if len(structuredResult["changed_files"].([]any)) != len(changedFiles) || len(structuredResult["edit_receipt"].(map[string]any)["changes"].([]any)) != len(changes) {
		t.Fatalf("structured result lost complete changed-path evidence: %#v", structuredResult)
	}
	if len(structuredResult["edit_receipt"].(map[string]any)["read_set"].([]any)) != len(readSet) {
		t.Fatalf("structured receipt lost its read set: %#v", structuredResult["edit_receipt"])
	}
	if len(structured[diagnosticDeltaField].(map[string]any)["introduced"].([]any)) != len(diagnostics) {
		t.Fatalf("structured diagnostic delta lost entries: %#v", structured[diagnosticDeltaField])
	}
	text := resultEnvelope["content"].([]any)[0].(map[string]any)["text"].(string)
	if len(text) > 4000 || !strings.Contains(text, "Changed paths (30)") || !strings.Contains(text, "20 more in structuredContent.result.changed_files") || !strings.Contains(text, `"file":"api/moved.go"`) {
		t.Fatalf("move_file text was not compact or omitted its continuation: %d bytes: %s", len(text), text)
	}
	for _, marker := range []string{"revision-marker", "read-revision-marker", "diagnostic-marker", "diff-marker", "\"changed_files\":"} {
		if strings.Contains(text, marker) {
			t.Fatalf("move_file text contained bulk payload marker %q: %s", marker, text)
		}
	}
}
