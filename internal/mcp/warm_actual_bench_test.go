// Package mcp_test measures completed warm MCP workflows so benchmark success requires observable work.
package mcp_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

type warmComponent struct {
	latencies []time.Duration
	wireBytes []int
}

type warmObservations struct {
	components map[string]*warmComponent
}

func newWarmObservations() *warmObservations {
	return &warmObservations{components: make(map[string]*warmComponent)}
}

func (observed *warmObservations) call(b *testing.B, client *warmMCPCostClient, component, tool string, args map[string]any) json.RawMessage {
	b.Helper()
	beforeBytes := client.lastResponseBytes
	started := time.Now()
	result := client.toolResult(b, tool, args)
	latency := time.Since(started)
	sample := observed.components[component]
	if sample == nil {
		sample = &warmComponent{}
		observed.components[component] = sample
	}
	sample.latencies = append(sample.latencies, latency)
	sample.wireBytes = append(sample.wireBytes, client.lastResponseBytes-beforeBytes)
	return result
}

func (observed *warmObservations) report(b *testing.B, budgets map[string]time.Duration, goplsStarts int) {
	b.Helper()
	type componentReport struct {
		Samples       int     `json:"samples"`
		P50MS         float64 `json:"p50_ms"`
		P99MS         float64 `json:"p99_ms"`
		MaxMS         float64 `json:"max_ms"`
		BudgetMS      float64 `json:"budget_ms"`
		WireBytesMean float64 `json:"wire_bytes_mean"`
		WireBytesMax  int     `json:"wire_bytes_max"`
	}
	reports := make(map[string]componentReport, len(budgets))
	names := make([]string, 0, len(budgets))
	for name := range budgets {
		names = append(names, name)
	}
	slices.Sort(names)
	var exceeded []string
	for _, name := range names {
		budget := budgets[name]
		component := observed.components[name]
		if component == nil {
			b.Fatalf("%s recorded no samples, want %d", name, b.N)
		}
		if len(component.latencies) != b.N {
			b.Fatalf("%s recorded %d samples, want %d", name, len(component.latencies), b.N)
		}
		ordered := slices.Clone(component.latencies)
		slices.Sort(ordered)
		p99 := ordered[(99*len(ordered)+99)/100-1]
		b.ReportMetric(float64(ordered[(len(ordered)-1)/2])/float64(time.Millisecond), name+"_p50_ms")
		b.ReportMetric(float64(p99)/float64(time.Millisecond), name+"_p99_ms")
		b.ReportMetric(float64(ordered[len(ordered)-1])/float64(time.Millisecond), name+"_max_ms")
		b.ReportMetric(float64(budget)/float64(time.Millisecond), name+"_budget_ms")
		var bytesTotal, bytesMax int
		for _, size := range component.wireBytes {
			bytesTotal += size
			bytesMax = max(bytesMax, size)
		}
		b.ReportMetric(float64(bytesTotal)/float64(len(component.wireBytes)), name+"_wire_bytes_mean")
		b.ReportMetric(float64(bytesMax), name+"_wire_bytes_max")
		reports[name] = componentReport{len(ordered), float64(ordered[(len(ordered)-1)/2]) / float64(time.Millisecond), float64(p99) / float64(time.Millisecond), float64(ordered[len(ordered)-1]) / float64(time.Millisecond), float64(budget) / float64(time.Millisecond), float64(bytesTotal) / float64(len(component.wireBytes)), bytesMax}
		if b.N > 1 && os.Getenv("SEMEDIT_BENCH_ENFORCE_BUDGETS") == "1" && (p99 >= time.Second || p99 > budget) {
			exceeded = append(exceeded, fmt.Sprintf("%s=%s (budget %s)", name, p99, budget))
		}
	}
	b.ReportMetric(float64(goplsStarts), "gopls_starts")
	encoded, err := json.Marshal(map[string]any{"components": reports, "gopls_starts": goplsStarts})
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("WARM_COST_RESULT %s", encoded)
	if len(exceeded) != 0 {
		b.Errorf("warm p99 exceeded target or one-second ceiling: %s", strings.Join(exceeded, ", "))
	}
}

func managedSessionStarts(tb testing.TB, fixture managedMCPFixture) int {
	tb.Helper()
	return managedSessionStartCount(fileEditSessionStarts(tb, fixture))
}

func managedSessionStartCount(starts map[string]int) int {
	total := 0
	for _, count := range starts {
		total += count
	}
	return total
}

func warmResult[T any](b *testing.B, raw json.RawMessage) T {
	b.Helper()
	var envelope struct {
		Result T `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		b.Fatalf("decode structured result: %v: %s", err, raw)
	}
	return envelope.Result
}

func warmWorkspace(b *testing.B, packages int) (managedMCPFixture, *warmMCPCostClient) {
	return warmWorkspaceWithFiles(b, packages, nil)
}

func warmWorkspaceWithFiles(b *testing.B, packages int, files map[string]string) (managedMCPFixture, *warmMCPCostClient) {
	b.Helper()
	fixture := newWarmMCPCostFixture(b, packages)
	for name, source := range files {
		if err := pipeline.WriteAtomic(filepath.Join(fixture.root, name), []byte(source)); err != nil {
			b.Fatal(err)
		}
	}
	client := newWarmMCPCostClient(b, fixture.root)
	client.call(b, "initialize", map[string]any{})
	warmVerify(b, client)
	return fixture, client
}

func warmVerify(b *testing.B, client *warmMCPCostClient) {
	b.Helper()
	assertFileRenameVerification(b, client.toolResult(b, "semantic_verify", map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true}))
}

func warmEmbeddedVerify(b *testing.B, result *operation.VerifyRes) {
	b.Helper()
	if result == nil || result.Failure != "" || len(result.Diagnostics) != 0 {
		b.Fatalf("embedded verification is absent or unclean: %+v", result)
	}
	for _, coverage := range result.Coverage {
		if coverage.Language == "go" && coverage.Status == "verified" {
			return
		}
	}
	b.Fatalf("embedded verification omitted completed Go coverage: %+v", result.Coverage)
}

func warmDiagnosticReport(b *testing.B, raw json.RawMessage) {
	b.Helper()
	var response struct {
		Report struct {
			Status         string `json:"status"`
			RemainingCount *int   `json:"remaining_count"`
		} `json:"diagnostic_report"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		b.Fatal(err)
	}
	if response.Report.Status != "available" || response.Report.RemainingCount == nil || *response.Report.RemainingCount != 0 {
		b.Fatalf("mutation did not capture a clean diagnostic inventory: %s", raw)
	}
}

func warmPersistedEdit(b *testing.B, raw json.RawMessage) {
	b.Helper()
	value := warmResult[struct {
		EditReceipt struct {
			Status string `json:"status"`
		} `json:"edit_receipt"`
	}](b, raw)
	if value.EditReceipt.Status != "persisted" {
		b.Fatalf("edit did not persist: %s", raw)
	}
}

func warmContinuity(b *testing.B, fixture managedMCPFixture, client *warmMCPCostClient) {
	b.Helper()
	warmVerify(b, client)
	if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
		b.Fatalf("warm workflow restarted gopls %d times; want one process", starts)
	}
}

func warmSourceContains(b *testing.B, root, file, fragment string) {
	b.Helper()
	source, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		b.Fatal(err)
	}
	if !strings.Contains(string(source), fragment) {
		b.Fatalf("%s omitted %q: %s", file, fragment, source)
	}
}

func warmBody(index int, first, second string) string {
	if index%2 == 0 {
		return second
	}
	return first
}

func BenchmarkMCPWarmOperationActualWork(b *testing.B) {
	for _, packages := range []int{1, 20} {
		b.Run(fmt.Sprintf("packages_%d", packages), func(b *testing.B) {
			b.Run("body_unique_values_first_read_inspect", func(b *testing.B) { benchmarkWarmBodyRead(b, packages) })
			b.Run("metrics_100_records", func(b *testing.B) { benchmarkWarmMetrics(b, packages) })
			b.Run("selected_file_snapshot", func(b *testing.B) { benchmarkWarmSnapshot(b, packages) })
			b.Run("inspect_construct", func(b *testing.B) { benchmarkWarmConstruct(b, packages) })
			b.Run("clean_verify", func(b *testing.B) { benchmarkWarmVerify(b, packages) })
			b.Run("dirty_organize_imports", func(b *testing.B) { benchmarkWarmImports(b, packages) })
			b.Run("two_file_batch", func(b *testing.B) { benchmarkWarmBatch(b, packages) })
			b.Run("stage_edit_commit", func(b *testing.B) { benchmarkWarmStage(b, packages) })
			b.Run("stage_abort_continuity", func(b *testing.B) { benchmarkWarmAbortContinuity(b, packages) })
		})
	}
}

func benchmarkWarmBodyRead(b *testing.B, packages int) {
	fixture, client := warmWorkspace(b, packages)
	client.toolResult(b, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": "return 2"})
	client.toolResult(b, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": "return 1"})
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		body := fmt.Sprintf("return %d", index+2)
		edit := observed.call(b, client, "edit", "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": body})
		lookup := observed.call(b, client, "first_lookup", "semantic_lookup", map[string]any{"file": "value.go", "symbol": "Value"})
		inspection := observed.call(b, client, "following_inspect", "semantic_inspect_symbol", map[string]any{"file": "value.go", "symbol": "Value"})
		recordWarmTotal(observed, "edit", "first_lookup", "following_inspect")
		b.StopTimer()
		warmPersistedEdit(b, edit)
		warmDiagnosticReport(b, edit)
		if !strings.Contains(string(lookup), `"symbol":"Value"`) || !strings.Contains(string(inspection), body) {
			b.Fatalf("post-edit reads omitted Value or %q: lookup=%s inspect=%s", body, lookup, inspection)
		}
		warmSourceContains(b, fixture.root, "value.go", body)
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"edit": time.Second, "first_lookup": 10 * time.Millisecond, "following_inspect": 10 * time.Millisecond, "tool_calls_total": time.Second}, managedSessionStarts(b, fixture))
}

func recordWarmTotal(observed *warmObservations, parts ...string) {
	var bytes int
	var total time.Duration
	for _, name := range parts {
		component := observed.components[name]
		bytes += component.wireBytes[len(component.wireBytes)-1]
		total += component.latencies[len(component.latencies)-1]
	}
	sample := observed.components["tool_calls_total"]
	if sample == nil {
		sample = &warmComponent{}
		observed.components["tool_calls_total"] = sample
	}
	sample.latencies = append(sample.latencies, total)
	sample.wireBytes = append(sample.wireBytes, bytes)
}

func benchmarkWarmMetrics(b *testing.B, packages int) {
	fixture, client := warmWorkspace(b, packages)
	for range 100 {
		client.toolResult(b, "semantic_lookup", map[string]any{"file": "value.go", "symbol": "Value"})
	}
	client.toolResult(b, "semantic_metrics", map[string]any{"limit": 100})
	observed := newWarmObservations()
	b.ResetTimer()
	for range b.N {
		result := observed.call(b, client, "metrics", "semantic_metrics", map[string]any{"limit": 100})
		b.StopTimer()
		data := warmResult[struct {
			Records []struct {
				Tool     string `json:"tool"`
				Sequence uint64 `json:"sequence"`
			} `json:"records"`
		}](b, result)
		if len(data.Records) != 100 {
			b.Fatalf("metrics returned %d records, want 100: %s", len(data.Records), result)
		}
		for index, record := range data.Records {
			if record.Tool != "semantic_lookup" || record.Sequence == 0 || index > 0 && record.Sequence >= data.Records[index-1].Sequence {
				b.Fatalf("metrics record %d is not a seeded lookup in descending order: %s", index, result)
			}
		}
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"metrics": 10 * time.Millisecond}, managedSessionStarts(b, fixture))
}

func benchmarkWarmSnapshot(b *testing.B, packages int) {
	fixture, client := warmWorkspace(b, packages)
	client.toolResult(b, "semantic_snapshot", map[string]any{"label": "warm-snapshot-setup", "paths": []string{"value.go"}})
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		result := observed.call(b, client, "snapshot", "semantic_snapshot", map[string]any{"label": fmt.Sprintf("warm-snapshot-%d", index), "paths": []string{"value.go"}})
		b.StopTimer()
		data := warmResult[struct {
			Status string `json:"status"`
			ID     string `json:"snapshot_id"`
			Count  int    `json:"captured_file_count"`
		}](b, result)
		if data.Status != "created" || data.ID == "" || data.Count != 1 {
			b.Fatalf("selected snapshot invalid: %s", result)
		}
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"snapshot": time.Second}, managedSessionStarts(b, fixture))
}

func benchmarkWarmConstruct(b *testing.B, packages int) {
	const source = "package sample\n\nfunc Check(v int) int { if v > 0 { return v }; return 0 }\n"
	fixture, client := warmWorkspaceWithFiles(b, packages, map[string]string{"inspect.go": source})
	args := map[string]any{"file": "inspect.go", "in_function": "Check", "kind": "if", "discriminator": "v > 0"}
	client.toolResult(b, "semantic_inspect_construct", args)
	observed := newWarmObservations()
	b.ResetTimer()
	for range b.N {
		result := observed.call(b, client, "inspect_construct", "semantic_inspect_construct", args)
		b.StopTimer()
		data := warmResult[struct {
			Candidates []struct {
				Path   string `json:"construct_path"`
				Syntax string `json:"syntax"`
			} `json:"candidates"`
			Revision string `json:"expected_selection_revision"`
		}](b, result)
		if len(data.Candidates) != 1 || data.Candidates[0].Path == "" || !strings.Contains(data.Candidates[0].Syntax, "v > 0") || data.Revision == "" {
			b.Fatalf("construct selection invalid: %s", result)
		}
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"inspect_construct": 10 * time.Millisecond}, managedSessionStarts(b, fixture))
}

func benchmarkWarmVerify(b *testing.B, packages int) {
	fixture, client := warmWorkspace(b, packages)
	observed := newWarmObservations()
	args := map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true}
	b.ResetTimer()
	for range b.N {
		result := observed.call(b, client, "verify", "semantic_verify", args)
		b.StopTimer()
		assertFileRenameVerification(b, result)
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"verify": time.Second}, managedSessionStarts(b, fixture))
}

func benchmarkWarmImports(b *testing.B, packages int) {
	fixture, client := warmWorkspaceWithFiles(b, packages, map[string]string{"imports.go": "package sample\n\nvar ImportValue = 1\n"})
	file := filepath.Join(fixture.root, "imports.go")
	const dirty = "package sample\n\nimport \"fmt\"\n\nvar ImportValue = 1\n"
	if err := pipeline.WriteAtomic(file, []byte(dirty)); err != nil {
		b.Fatal(err)
	}
	client.toolResult(b, "semantic_organize_imports", map[string]any{"file": "imports.go"})
	observed := newWarmObservations()
	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		if err := pipeline.WriteAtomic(file, []byte(dirty)); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		result := observed.call(b, client, "organize_imports", "semantic_organize_imports", map[string]any{"file": "imports.go"})
		b.StopTimer()
		if !strings.Contains(string(result), `"application_state":"persisted"`) {
			b.Fatalf("imports were not organized: %s", result)
		}
		warmDiagnosticReport(b, result)
		content, err := os.ReadFile(file)
		if err != nil {
			b.Fatal(err)
		}
		if strings.Contains(string(content), `"fmt"`) || !strings.Contains(string(content), "ImportValue") {
			b.Fatalf("imports result incorrect: %s", content)
		}
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
	}
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"organize_imports": time.Second}, managedSessionStarts(b, fixture))
}

func benchmarkWarmBatch(b *testing.B, packages int) {
	fixture, client := warmWorkspaceWithFiles(b, packages, map[string]string{"other.go": "package sample\n\nfunc Other() int { return 10 }\n"})
	client.toolResult(b, "semantic_batch", map[string]any{"edits": warmBatchEdits("return 2", "return 20")})
	client.toolResult(b, "semantic_batch", map[string]any{"edits": warmBatchEdits("return 1", "return 10")})
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		first, second := warmBody(index, "return 1", "return 2"), warmBody(index, "return 10", "return 20")
		result := observed.call(b, client, "batch", "semantic_batch", map[string]any{"edits": warmBatchEdits(first, second)})
		b.StopTimer()
		data := warmResult[mcp.BatchResponse](b, result)
		if data.Status != "ok" || len(data.Results) < 2 || data.FinalDiff == "" || data.Verification == nil {
			b.Fatalf("batch omitted completed work: %s", result)
		}
		warmEmbeddedVerify(b, data.Verification)
		for _, row := range data.Results[:2] {
			if row.Status != "ok" {
				b.Fatalf("batch child failed: %s", result)
			}
		}
		warmSourceContains(b, fixture.root, "value.go", first)
		warmSourceContains(b, fixture.root, "other.go", second)
		warmContinuity(b, fixture, client)
		b.StartTimer()
	}
	b.StopTimer()
	observed.report(b, map[string]time.Duration{"batch": time.Second}, managedSessionStarts(b, fixture))
}

func warmBatchEdits(first, second string) []map[string]any {
	return []map[string]any{
		{"tool": "semantic_replace_body", "params": map[string]any{"file": "value.go", "symbol": "Value", "body": first}},
		{"tool": "semantic_replace_body", "params": map[string]any{"file": "other.go", "symbol": "Other", "body": second}},
	}
}

func benchmarkWarmStage(b *testing.B, packages int) {
	fixture, client := warmWorkspace(b, packages)
	client.toolResult(b, "semantic_stage", map[string]any{})
	client.toolResult(b, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": "return 2"})
	client.toolResult(b, "semantic_commit", map[string]any{"skip_hooks": true})
	client.toolResult(b, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": "return 1"})
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		body := warmBody(index, "return 1", "return 2")
		start := observed.call(b, client, "stage", "semantic_stage", map[string]any{})
		edit := observed.call(b, client, "edit", "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": body})
		b.StopTimer()
		if !strings.Contains(string(start), `"stage_id"`) || !strings.Contains(string(edit), `"step_count":1`) {
			b.Fatalf("stage did not retain one edit: stage=%s edit=%s", start, edit)
		}
		before := "return 1"
		if index > 0 {
			before = warmBody(index-1, "return 1", "return 2")
		}
		warmSourceContains(b, fixture.root, "value.go", before)
		b.StartTimer()
		commit := observed.call(b, client, "commit", "semantic_commit", map[string]any{"skip_hooks": true})
		recordWarmTotal(observed, "stage", "edit", "commit")
		b.StopTimer()
		data := warmResult[operation.StageCommitResult](b, commit)
		if data.Status != "ok" || data.PublishedFiles != 1 || data.Verification == nil || data.FinalDiff == "" {
			b.Fatalf("stage commit incomplete: %s", commit)
		}
		warmEmbeddedVerify(b, data.Verification)
		warmSourceContains(b, fixture.root, "value.go", body)
		warmContinuity(b, fixture, client)
		b.StartTimer()
	}
	b.StopTimer()
	observed.report(b, map[string]time.Duration{"stage": time.Second, "edit": time.Second, "commit": time.Second, "tool_calls_total": time.Second}, managedSessionStarts(b, fixture))
}

func benchmarkWarmAbortContinuity(b *testing.B, packages int) {
	fixture, client := warmWorkspace(b, packages)
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		start := observed.call(b, client, "stage", "semantic_stage", map[string]any{})
		edit := observed.call(b, client, "edit", "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": "return 2"})
		abort := observed.call(b, client, "abort", "semantic_stage_abort", map[string]any{})
		b.StopTimer()
		if !strings.Contains(string(start), `"stage_id"`) || !strings.Contains(string(edit), `"step_count":1`) || !strings.Contains(string(abort), `"aborted"`) {
			b.Fatalf("stage abort lifecycle invalid: stage=%s edit=%s abort=%s", start, edit, abort)
		}
		warmSourceContains(b, fixture.root, "value.go", "return 1")
		startsAfterAbort := fileRenameGoplsStarts(b, fixture.counter)
		verified := observed.call(b, client, "followup_verify", "semantic_verify", map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true})
		assertFileRenameVerification(b, verified)
		starts := fileRenameGoplsStarts(b, fixture.counter)
		if starts != 1 {
			latest := make(map[string]map[string]any, len(observed.components))
			for name, sample := range observed.components {
				last := len(sample.latencies) - 1
				latest[name] = map[string]any{"latency_ms": float64(sample.latencies[last]) / float64(time.Millisecond), "wire_bytes": sample.wireBytes[last]}
			}
			encoded, err := json.Marshal(map[string]any{"reason": "gopls_restart", "samples_collected": index + 1, "starts_after_abort": startsAfterAbort, "starts_after_verify": starts, "latest_components": latest})
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("WARM_COST_REJECTED %s", encoded)
			b.Fatalf("stage abort broke retained gopls continuity after %d samples: starts=%d", index+1, starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	observed.report(b, map[string]time.Duration{"stage": time.Second, "edit": time.Second, "abort": time.Second, "followup_verify": time.Second}, managedSessionStarts(b, fixture))
}
