// Diagnostic page benchmarks measure production MCP reads of populated captured reports with one retained gopls.

package mcp_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

func BenchmarkMCPDiagnosticPageCost(b *testing.B) {
	for _, packages := range []int{1, 20} {
		b.Run(fmt.Sprintf("packages_%d", packages), func(b *testing.B) { benchmarkMCPDiagnosticPageCost(b, packages) })
	}
}

type warmDiagnosticCapture struct {
	Report struct {
		Status         string         `json:"status"`
		ID             string         `json:"report_id"`
		RemainingCount int            `json:"remaining_count"`
		Arguments      map[string]any `json:"arguments"`
	} `json:"diagnostic_report"`
}

type warmDiagnosticPage struct {
	Result struct {
		ReportID string `json:"report_id"`
		Status   string `json:"status"`
		Category string `json:"category"`
		Total    int    `json:"total"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
		Items    []struct {
			Message string `json:"message"`
		} `json:"items"`
	} `json:"result"`
}

func decodeWarmDiagnosticCapture(b *testing.B, raw json.RawMessage) warmDiagnosticCapture {
	b.Helper()
	var captured warmDiagnosticCapture
	if err := json.Unmarshal(raw, &captured); err != nil {
		b.Fatal(err)
	}
	if captured.Report.Status != "available" || captured.Report.ID == "" || captured.Report.RemainingCount < 50 {
		b.Fatalf("benchmark requires a populated diagnostic report: %s", raw)
	}
	return captured
}

func decodeWarmDiagnosticPage(b *testing.B, raw json.RawMessage, captured warmDiagnosticCapture) warmDiagnosticPage {
	b.Helper()
	var page warmDiagnosticPage
	if err := json.Unmarshal(raw, &page); err != nil {
		b.Fatal(err)
	}
	result := page.Result
	if result.ReportID != captured.Report.ID || result.Status != "available" || result.Category != "active" || result.Offset != 0 || result.Limit != 50 || result.Total != captured.Report.RemainingCount || len(result.Items) != 50 || result.Items[0].Message == "" || result.Items[49].Message == "" {
		b.Fatalf("incomplete captured diagnostic page: %s", raw)
	}
	return page
}

func assertWarmDiagnosticPageContinuity(b *testing.B, client *warmMCPCostClient, fixture managedMCPFixture, captured warmDiagnosticCapture) {
	b.Helper()
	after := client.toolResult(b, "semantic_verify", map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true})
	verified := warmResult[operation.VerifyRes](b, after)
	if verified.Failure != "" || len(verified.Diagnostics) < 50 {
		b.Fatalf("post-page verification lost diagnostics: %s", after)
	}
	goVerified := false
	for _, coverage := range verified.Coverage {
		if coverage.Language == "go" && coverage.Status == "verified" {
			goVerified = true
		}
	}
	if !goVerified {
		b.Fatalf("post-page verification omitted completed Go coverage: %s", after)
	}
	var afterCapture warmDiagnosticCapture
	if err := json.Unmarshal(after, &afterCapture); err != nil {
		b.Fatal(err)
	}
	if afterCapture.Report.Status != "available" || afterCapture.Report.RemainingCount != captured.Report.RemainingCount {
		b.Fatalf("diagnostic capture changed after paging: %s", after)
	}
	if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
		b.Fatalf("post-page verification restarted gopls: %d", starts)
	}
}

func benchmarkMCPDiagnosticPageCost(b *testing.B, packages int) {
	fixture := newWarmMCPCostFixture(b, packages)
	var source strings.Builder
	source.WriteString("package sample\nfunc Value() int { return 1 }\n")
	for index := range 50 {
		fmt.Fprintf(&source, "var broken%03d = missing%03d\n", index, index)
	}
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "value.go"), []byte(source.String())); err != nil {
		b.Fatal(err)
	}
	client := newWarmMCPCostClient(b, fixture.root)
	client.call(b, "initialize", map[string]any{})
	verified := client.toolResult(b, "semantic_verify", map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true})
	captured := decodeWarmDiagnosticCapture(b, verified)
	arguments := captured.Report.Arguments
	arguments["limit"] = 50
	warm := client.toolResult(b, "semantic_diagnostics", arguments)
	page := decodeWarmDiagnosticPage(b, warm, captured)
	observed := newWarmObservations()
	b.ResetTimer()
	for range b.N {
		result := observed.call(b, client, "page", "semantic_diagnostics", arguments)
		b.StopTimer()
		current := decodeWarmDiagnosticPage(b, result, captured)
		if current.Result.Items[0].Message != page.Result.Items[0].Message || current.Result.Items[49].Message != page.Result.Items[49].Message {
			b.Fatalf("captured diagnostic page changed: %s", result)
		}
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("diagnostic paging restarted gopls: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	assertWarmDiagnosticPageContinuity(b, client, fixture, captured)
	observed.report(b, map[string]time.Duration{"page": 10 * time.Millisecond}, managedSessionStarts(b, fixture))
	b.ReportMetric(50, "diagnostics_per_page")
	b.ReportMetric(float64(captured.Report.RemainingCount), "captured_diagnostics")
}
