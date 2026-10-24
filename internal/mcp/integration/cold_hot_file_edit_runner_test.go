// Package mcp_test measures source edits in disposable corpora.

package mcp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func BenchmarkMCPColdHotFileEdits(b *testing.B) {
	if os.Getenv("SEMEDIT_MCP_TOOL_COSTS") != "1" {
		b.Skip("set SEMEDIT_MCP_TOOL_COSTS=1")
	}
	samples := coldHotSampleCounts(b)
	kind := os.Getenv("SEMEDIT_MCP_COST_CORPUS")
	if kind == "" {
		kind = "20"
	}
	if kind != "20" && kind != "repository" {
		b.Fatalf("unknown corpus %q", kind)
	}
	var corpus *repositoryCostCorpus
	if kind == "repository" {
		value := newRepositoryCostCorpus(b)
		corpus = &value
		b.Logf("corpus=semantic-editor revision=%s target=internal/telemetry probe_file=true clone=platform-native", value.revision)
	}
	var rows []coldHotResult
	for _, scenario := range fileEditCostCases() {
		filter := os.Getenv("SEMEDIT_MCP_TOOL_FILTER")
		if filter != "" && !strings.Contains(scenario.tool, filter) && !strings.Contains(scenario.name, filter) {
			continue
		}
		var row coldHotResult
		ok := b.Run(scenario.name, func(child *testing.B) { row = runFileEditCosts(child, scenario, samples, corpus) })
		if !ok {
			row = coldHotResult{tool: scenario.tool, status: "FAILED"}
		}
		variant := strings.TrimPrefix(scenario.name, strings.TrimPrefix(scenario.tool, "semantic_")+"_")
		if variant != scenario.name {
			row.tool += "[" + variant + "]"
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		if kind == "20" {
			for _, baseline := range coldHotToolCases() {
				filter := os.Getenv("SEMEDIT_MCP_TOOL_FILTER")
				if strings.Contains(baseline.tool, filter) || strings.Contains(baseline.name, filter) {
					b.Skip("filter selects the baseline read/lifecycle matrix")
				}
			}
		}
		b.Fatalf("no source-edit cases match filter %q", os.Getenv("SEMEDIT_MCP_TOOL_FILTER"))
	}
	reportFileEditCosts(b, kind, rows)
}

func runFileEditCosts(b *testing.B, scenario fileEditCostCase, counts coldHotSamples, corpus *repositoryCostCorpus) coldHotResult {
	b.StopTimer()
	prior := saveColdHotEnvironment()
	defer prior.restore(b)
	row := coldHotResult{tool: scenario.tool, status: "PASS", budget: time.Second}
	cold, restarted := collectColdFileEditCosts(b, scenario, counts.cold, corpus, prior)
	if restarted {
		row.status = "RESTART"
		return row
	}
	if scenario.tool == "semantic_undo" {
		row.workflowP99, row.workflowSamples = coldHotP99(b, cold), len(cold)
	} else {
		row.coldP99, row.coldSamples = coldHotP99(b, cold), len(cold)
	}
	fixture, selected := newFileEditFixture(b, scenario, corpus)
	client := newWarmFileEditClient(b, fixture)
	baseStarts := fileEditSessionStarts(b, fixture)
	if len(baseStarts) == 0 || fileEditSessionRestarted(baseStarts) {
		b.Fatalf("hot setup did not retain one gopls per module: %v", baseStarts)
	}
	var hot []time.Duration
	for index := range counts.hot {
		if index > 0 {
			resetFileEditFixture(b, fixture, selected)
			client.toolResult(b, "semantic_verify", coldHotVerifyArgs())
		}
		args := fileEditArguments(b, fixture, selected, client)
		started := time.Now()
		raw := client.toolResult(b, selected.tool, args)
		elapsed := time.Since(started)
		validateFileEdit(b, fixture, selected, raw)
		verifyFileEditState(b, client)
		if change := fileEditSessionChange(baseStarts, fileEditSessionStarts(b, fixture)); change != "" {
			row.status = change
			row.hotSamples = len(hot)
			client.close(b)
			return row
		}
		hot = append(hot, elapsed)
	}
	client.close(b)
	row.hotP99, row.hotSamples = coldHotP99(b, hot), len(hot)
	if counts.cold < 100 || counts.hot < 100 {
		row.status = "SMOKE"
	} else if row.hotP99 >= time.Second {
		row.status = "BUDGET"
	}
	return row
}

func newFileEditFixture(b *testing.B, scenario fileEditCostCase, corpus *repositoryCostCorpus) (managedMCPFixture, fileEditCostCase) {
	var fixture managedMCPFixture
	if corpus == nil {
		fixture = newWarmMCPCostFixture(b, 20)
	} else {
		fixture = newManagedMCPFixture(b)
		fixture.root = corpus.clone(b)
		coldHotWrite(b, fixture.root, ".semedit.yaml", "version: 1\nverify:\n  check:\n    - id: golangci-lint\n      languages: [go]\n      disabled: true\n")
		scenario = repositoryFileEditCase(scenario)
	}
	resetFileEditFixture(b, fixture, scenario)
	if corpus == nil {
		count := countGoSourceFiles(b, fixture.root)
		for index := 19; count > 20 && index > 0; index-- {
			path := filepath.Join(fixture.root, fmt.Sprintf("consumer%d", index), "consumer.go")
			if err := os.Remove(path); err != nil {
				b.Fatal(err)
			}
			count--
		}
		coldHotExactFiles(b, fixture)
	}
	return fixture, scenario
}

func resetFileEditFixture(tb testing.TB, fixture managedMCPFixture, scenario fileEditCostCase) {
	for path, source := range scenario.files {
		absolute := filepath.Join(fixture.root, path)
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			tb.Fatal(err)
		}
		if err := pipeline.WriteAtomic(absolute, []byte(source)); err != nil {
			tb.Fatal(err)
		}
	}
	for path := range scenario.expected {
		if _, exists := scenario.files[path]; exists {
			continue
		}
		if err := os.Remove(filepath.Join(fixture.root, path)); err != nil && !os.IsNotExist(err) {
			tb.Fatal(err)
		}
	}
}

func repositoryFileEditCase(scenario fileEditCostCase) fileEditCostCase {
	mapPath := func(path string) string {
		return filepath.ToSlash(filepath.Join("internal", "telemetry", "cost_"+path))
	}
	files := make(map[string]string, len(scenario.files))
	for path, source := range scenario.files {
		files[mapPath(path)] = strings.Replace(source, "package sample", "package telemetry", 1)
	}
	scenario.files = files
	for _, target := range []*map[string][]string{&scenario.expected, &scenario.absent} {
		mapped := make(map[string][]string, len(*target))
		for path, parts := range *target {
			mappedParts := make([]string, len(parts))
			for index, part := range parts {
				mappedParts[index] = strings.ReplaceAll(part, "package sample", "package telemetry")
			}
			mapped[mapPath(path)] = mappedParts
		}
		*target = mapped
	}
	scenario.args = mapFileEditArguments(scenario.args, mapPath)
	if scenario.args["package"] == "sample" {
		scenario.args["package"] = "telemetry"
	}
	return scenario
}

func mapFileEditArguments(args map[string]any, mapPath func(string) string) map[string]any {
	result := make(map[string]any, len(args))
	for key, value := range args {
		switch typed := value.(type) {
		case string:
			if key == "file" || key == "destination_file" || key == "contract_file" {
				result[key] = mapPath(typed)
			} else {
				result[key] = typed
			}
		case map[string]any:
			result[key] = mapFileEditArguments(typed, mapPath)
		case []map[string]any:
			items := make([]map[string]any, len(typed))
			for index, item := range typed {
				items[index] = mapFileEditArguments(item, mapPath)
			}
			result[key] = items
		case []any:
			items := make([]any, len(typed))
			for index, item := range typed {
				if object, ok := item.(map[string]any); ok {
					items[index] = mapFileEditArguments(object, mapPath)
				} else {
					items[index] = item
				}
			}
			result[key] = items
		default:
			result[key] = value
		}
	}
	return result
}

func fileEditArguments(tb testing.TB, fixture managedMCPFixture, scenario fileEditCostCase, client *warmMCPCostClient) map[string]any {
	if scenario.prepare != nil {
		return scenario.prepare(tb, fixture, client)
	}
	if scenario.tool != "semantic_undo" {
		return scenario.args
	}
	file := "value.go"
	if _, exists := scenario.files[file]; !exists {
		file = "internal/telemetry/cost_value.go"
	}
	raw := client.toolResult(tb, "semantic_snapshot", map[string]any{"paths": []string{file}, "label": "benchmark undo"})
	var response struct {
		Result struct {
			ID string `json:"snapshot_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		tb.Fatal(err)
	}
	if response.Result.ID == "" {
		tb.Fatalf("snapshot returned no ID: %s", raw)
	}
	client.toolResult(tb, "semantic_replace_body", map[string]any{"file": file, "symbol": "Value", "body": "return 99"})
	return map[string]any{"snapshot_id": response.Result.ID}
}

func validateFileEdit(tb testing.TB, fixture managedMCPFixture, scenario fileEditCostCase, raw []byte) {
	if scenario.validate != nil {
		scenario.validate(tb, fixture, 0, raw)
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		tb.Fatal(err)
	}
	if len(envelope.Result) == 0 || bytes.Equal(envelope.Result, []byte("null")) {
		tb.Fatalf("missing native result: %s", raw)
	}
	changed := scenario.tool == "semantic_undo"
	for path, fragments := range scenario.expected {
		content, err := os.ReadFile(filepath.Join(fixture.root, path))
		if err != nil {
			tb.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(content), fragment) {
				tb.Fatalf("%s missing %q: %s", path, fragment, content)
			}
		}
		if original, exists := scenario.files[path]; !exists || string(content) != original {
			changed = true
		}
	}
	for path, fragments := range scenario.absent {
		content, err := os.ReadFile(filepath.Join(fixture.root, path))
		if err != nil {
			tb.Fatal(err)
		}
		for _, fragment := range fragments {
			if strings.Contains(string(content), fragment) {
				tb.Fatalf("%s still contains %q: %s", path, fragment, content)
			}
		}
		if string(content) != scenario.files[path] {
			changed = true
		}
	}
	if !changed {
		tb.Fatalf("%s did not change any asserted file", scenario.tool)
	}
}

func newWarmFileEditClient(tb testing.TB, fixture managedMCPFixture) *warmMCPCostClient {
	client := newWarmMCPCostClient(tb, fixture.root)
	client.call(tb, "initialize", map[string]any{})
	client.toolResult(tb, "semantic_verify", coldHotVerifyArgs())
	return client
}

func collectColdFileEditCosts(b *testing.B, scenario fileEditCostCase, count int, corpus *repositoryCostCorpus, prior coldHotEnvironment) ([]time.Duration, bool) {
	var latencies []time.Duration
	for range count {
		fixture, selected := newFileEditFixture(b, scenario, corpus)
		client := newWarmMCPCostClient(b, fixture.root)
		client.call(b, "initialize", map[string]any{})
		started := time.Now()
		args := fileEditArguments(b, fixture, selected, client)
		if selected.tool != "semantic_undo" {
			started = time.Now()
		}
		raw := client.toolResult(b, selected.tool, args)
		elapsed := time.Since(started)
		validateFileEdit(b, fixture, selected, raw)
		verifyFileEditState(b, client)
		starts := fileEditSessionStarts(b, fixture)
		client.close(b)
		prior.restore(b)
		if fileEditSessionRestarted(starts) {
			return nil, true
		}
		latencies = append(latencies, elapsed)
	}
	return latencies, false
}

func verifyFileEditState(tb testing.TB, client *warmMCPCostClient) {
	raw := client.toolResult(tb, "semantic_verify", coldHotVerifyArgs())
	var response struct {
		Result struct {
			Checks struct {
				Completed bool
				Partial   bool
			} `json:"checks"`
			Diagnostics []struct {
				Severity int `json:"severity"`
			} `json:"diagnostics"`
			Coverage []struct {
				Language string `json:"language"`
				Status   string `json:"status"`
			} `json:"coverage"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		tb.Fatal(err)
	}
	if !response.Result.Checks.Completed || response.Result.Checks.Partial {
		tb.Fatalf("incomplete source-edit verification: %s", raw)
	}
	verified := false
	for _, coverage := range response.Result.Coverage {
		if coverage.Language == "go" {
			if coverage.Status != "verified" {
				tb.Fatalf("incomplete Go coverage: %s", raw)
			}
			verified = true
		}
	}
	if !verified {
		tb.Fatalf("source-edit verification omitted Go coverage: %s", raw)
	}
	for _, diagnostic := range response.Result.Diagnostics {
		if diagnostic.Severity == 1 {
			tb.Fatalf("source edit left compiler errors: %s", raw)
		}
	}
}

func fileEditCostRowFailed(row coldHotResult) bool {
	switch row.status {
	case "PASS", "SMOKE":
		return false
	case "BUDGET":
		return os.Getenv("SEMEDIT_MCP_COST_ENFORCE_BUDGETS") == "1"
	default:
		return true
	}
}

func reportFileEditCosts(b *testing.B, kind string, rows []coldHotResult) {
	fmt.Printf("MCP source edits corpus=%s; setup/reset/clone excluded; hot retains one gopls per module; undo cold includes prerequisites\n", kind)
	fmt.Printf("%-38s %14s %14s %14s %15s %s\n", "tool", "cold p99", "workflow p99", "hot p99", "samples c/w/h", "status")
	for _, row := range rows {
		cold, workflow, hot := "N/A", "N/A", "WITHHELD"
		if row.coldSamples >= 100 {
			cold = row.coldP99.Round(time.Microsecond).String()
		}
		if row.workflowSamples >= 100 {
			workflow = row.workflowP99.Round(time.Microsecond).String()
		}
		if row.hotSamples >= 100 && (row.status == "PASS" || row.status == "BUDGET") {
			hot = row.hotP99.Round(time.Microsecond).String()
		}
		if row.status == "SMOKE" {
			cold, hot = "SMOKE", "SMOKE"
			if row.workflowSamples > 0 {
				cold, workflow = "N/A", "SMOKE"
			}
		}
		fmt.Printf("%-38s %14s %14s %14s %4d/%-4d/%-4d %s\n", row.tool, cold, workflow, hot, row.coldSamples, row.workflowSamples, row.hotSamples, row.status)
		if fileEditCostRowFailed(row) {
			b.Errorf("%s: %s", row.tool, row.status)
		}
	}
}
