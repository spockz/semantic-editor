// Package mcp_test measures first-call and retained-session MCP tool costs on a fixed Go workspace.
package mcp_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

type coldHotSamples struct {
	cold int
	hot  int
}

type coldHotTool struct {
	name        string
	tool        string
	cold        bool
	args        func(int) map[string]any
	prepare     func(testing.TB, managedMCPFixture, *warmMCPCostClient, int)
	prepareOnce bool
	validate    func(testing.TB, managedMCPFixture, int, []byte)
	after       func(testing.TB, managedMCPFixture, *warmMCPCostClient, int)
	budget      time.Duration
	prereqNote  string
}

func BenchmarkMCPColdHotToolCosts(b *testing.B) {
	if os.Getenv("SEMEDIT_MCP_COST_CORPUS") == "repository" {
		b.Skip("repository corpus is measured by the source-edit registry")
	}
	if os.Getenv("SEMEDIT_MCP_TOOL_COSTS") != "1" {
		b.Skip("set SEMEDIT_MCP_TOOL_COSTS=1 to run the cold/hot operation matrix")
	}
	samples := coldHotSampleCounts(b)
	filter := os.Getenv("SEMEDIT_MCP_TOOL_FILTER")
	tools := coldHotToolCases()
	var rows []coldHotResult
	for _, scenario := range tools {
		if filter != "" && !strings.Contains(scenario.name, filter) && !strings.Contains(scenario.tool, filter) {
			continue
		}
		b.Logf("MCP_COST_PROGRESS tool=%s cold=%d hot=%d", scenario.tool, samples.cold, samples.hot)
		var row coldHotResult
		ok := b.Run(scenario.name, func(child *testing.B) {
			row = runColdHotTool(child, scenario, samples)
		})
		if !ok {
			row = coldHotResult{tool: scenario.tool, status: "FAILED", budget: scenario.budget, prereqNote: "native or lifecycle validation failed; p99 withheld"}
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		b.Skipf("no baseline tools match SEMEDIT_MCP_TOOL_FILTER=%q", filter)
	}
	printColdHotTable(rows, samples)
	for _, row := range rows {
		if row.status == "RESTART" || row.status == "COLD_RESTART" {
			b.Errorf("%s lost retained gopls continuity", row.tool)
		}
		if os.Getenv("SEMEDIT_MCP_COST_ENFORCE_BUDGETS") == "1" && row.status == "BUDGET" {
			b.Errorf("%s hot p99 %s exceeds target %s or one-second ceiling", row.tool, row.hotP99, row.budget)
		}
	}
}

func coldHotSampleCounts(b *testing.B) coldHotSamples {
	b.Helper()
	cold := coldHotCount(b, "SEMEDIT_MCP_COLD_SAMPLES", 100)
	hot := coldHotCount(b, "SEMEDIT_MCP_HOT_SAMPLES", 1000)
	if cold < 100 || hot < 100 {
		if os.Getenv("SEMEDIT_MCP_TOOL_SMOKE") != "1" || cold > 2 || hot > 2 {
			b.Fatalf("p99 runs require at least 100 samples; use SEMEDIT_MCP_TOOL_SMOKE=1 for contract smoke runs of at most 2")
		}
	}
	return coldHotSamples{cold: cold, hot: hot}
}

func coldHotCount(b *testing.B, name string, fallback int) int {
	b.Helper()
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		b.Fatalf("%s must be a positive integer, got %q", name, value)
	}
	return parsed
}

type coldHotResult struct {
	tool            string
	coldP99         time.Duration
	workflowP99     time.Duration
	hotP99          time.Duration
	coldSamples     int
	workflowSamples int
	workflowRestart bool
	hotSamples      int
	starts          int
	restarts        int
	status          string
	budget          time.Duration
	prereqNote      string
}

func runColdHotTool(b *testing.B, scenario coldHotTool, samples coldHotSamples) coldHotResult {
	b.Helper()
	b.StopTimer()
	priorEnv := saveColdHotEnvironment()
	defer priorEnv.restore(b)
	fixture := newWarmMCPCostFixture(b, 20)
	configureColdHotFixture(b, fixture, scenario.tool)
	if files := countGoSourceFiles(b, fixture.root); files != 20 {
		b.Fatalf("fixture for %s has %d Go source files, want exactly 20", scenario.tool, files)
	}
	result := coldHotResult{tool: scenario.tool, budget: scenario.budget, status: "PASS", prereqNote: scenario.prereqNote}
	if scenario.cold {
		result.coldP99, result.coldSamples = runColdSamples(b, fixture, scenario, samples.cold)
	} else {
		result.workflowP99, result.workflowSamples, result.workflowRestart = runColdWorkflowSamples(b, fixture, scenario, samples.cold)
	}
	coldHotResetSourceState(b, fixture, scenario.tool, 0)
	priorStarts := coldHotStarts(b, fixture.counter)
	client := newWarmMCPCostClient(b, fixture.root)
	client.call(b, "initialize", map[string]any{})
	if scenario.tool == "semantic_diagnostics" {
		pageArguments := coldHotCaptureDiagnostics(b, fixture, client, 0)
		scenario.args = coldHotFixed(pageArguments)
	} else {
		warmVerify(b, client)
		if scenario.prepareOnce {
			scenario.prepare(b, fixture, client, 0)
		}
	}
	baseStarts := coldHotStarts(b, fixture.counter)
	if baseStarts != priorStarts+1 {
		b.Fatalf("hot %s setup started %d gopls processes, want one", scenario.tool, baseStarts-priorStarts)
	}
	result = runHotSamples(b, fixture, scenario, samples.hot, client, baseStarts, result)
	if result.status != "RESTART" {
		coldHotFinalState(b, fixture, scenario, client, baseStarts)
		switch {
		case result.workflowRestart:
			result.status = "COLD_RESTART"
			result.prereqNote += "; cold workflow restart, p99 withheld"
		case samples.hot < 100 || samples.cold < 100:
			result.status = "SMOKE"
		case result.hotP99 >= time.Second || result.hotP99 > result.budget:
			result.status = "BUDGET"
		}
	}
	client.close(b)
	return result
}

func runColdSamples(b *testing.B, fixture managedMCPFixture, scenario coldHotTool, count int) (time.Duration, int) {
	latencies := make([]time.Duration, 0, count)
	startsTotal := 0
	for index := range count {
		coldHotResetSourceState(b, fixture, scenario.tool, index)
		client := newWarmMCPCostClient(b, fixture.root)
		client.call(b, "initialize", map[string]any{})
		startsBefore := coldHotStarts(b, fixture.counter)
		if startsBefore != startsTotal {
			b.Fatalf("cold session %s started gopls during initialize: before=%d expected=%d", scenario.tool, startsBefore, startsTotal)
		}
		if scenario.prepare != nil {
			scenario.prepare(b, fixture, client, index)
		}
		coldHotExactFiles(b, fixture)
		started := time.Now()
		raw := client.toolResult(b, scenario.tool, scenario.args(index))
		latencies = append(latencies, time.Since(started))
		if scenario.validate != nil {
			scenario.validate(b, fixture, index, raw)
		}
		startsAfterCall := coldHotStarts(b, fixture.counter)
		if startsAfterCall-startsBefore > 1 {
			b.Fatalf("cold %s request started gopls %d times", scenario.tool, startsAfterCall-startsBefore)
		}
		if scenario.after != nil {
			scenario.after(b, fixture, client, index)
		}
		startsTotal = coldHotStarts(b, fixture.counter)
		if startsTotal-startsBefore > 1 {
			b.Fatalf("cold %s full sample started gopls %d times", scenario.tool, startsTotal-startsBefore)
		}
		client.close(b)
		coldHotExactFiles(b, fixture)
	}
	return coldHotP99(b, latencies), len(latencies)
}

func runColdWorkflowSamples(b *testing.B, fixture managedMCPFixture, scenario coldHotTool, count int) (time.Duration, int, bool) {
	b.Helper()
	latencies := make([]time.Duration, 0, count)
	startsTotal := coldHotStarts(b, fixture.counter)
	for index := range count {
		client := newWarmMCPCostClient(b, fixture.root)
		client.call(b, "initialize", map[string]any{})
		startsBefore := coldHotStarts(b, fixture.counter)
		if startsBefore != startsTotal {
			b.Fatalf("cold workflow %s started gopls during initialize: before=%d expected=%d", scenario.tool, startsBefore, startsTotal)
		}
		var arguments map[string]any
		started := time.Now()
		if scenario.tool == "semantic_diagnostics" {
			arguments = coldHotCaptureDiagnostics(b, fixture, client, index)
		} else {
			arguments = scenario.args(index)
			if scenario.prepare != nil {
				scenario.prepare(b, fixture, client, index)
			}
		}
		prepareDuration := time.Since(started)
		coldHotExactFiles(b, fixture)
		started = time.Now()
		raw := client.toolResult(b, scenario.tool, arguments)
		elapsed := prepareDuration + time.Since(started)
		if scenario.validate != nil {
			scenario.validate(b, fixture, index, raw)
		}
		if scenario.tool == "semantic_diagnostics" {
			coldHotPageIdentity(b, raw, arguments)
		}
		if scenario.after != nil {
			scenario.after(b, fixture, client, index)
		}
		coldHotExactFiles(b, fixture)
		startsAfter := coldHotStarts(b, fixture.counter)
		client.close(b)
		if startsAfter-startsBefore > 1 {
			b.Logf("MCP_COST_REJECTED tool=%s cold_workflow_sample=%d starts=%d expected_at_most=1", scenario.tool, index+1, startsAfter-startsBefore)
			return 0, len(latencies), true
		}
		startsTotal = startsAfter
		latencies = append(latencies, elapsed)
	}
	return coldHotP99(b, latencies), len(latencies), false
}

func runHotSamples(b *testing.B, fixture managedMCPFixture, scenario coldHotTool, count int, client *warmMCPCostClient, baseStarts int, result coldHotResult) coldHotResult {
	latencies := make([]time.Duration, 0, count)
	for index := range count {
		if scenario.prepare != nil && !scenario.prepareOnce {
			scenario.prepare(b, fixture, client, index)
		}
		coldHotExactFiles(b, fixture)
		started := time.Now()
		raw := client.toolResult(b, scenario.tool, scenario.args(index))
		latencies = append(latencies, time.Since(started))
		if scenario.validate != nil {
			scenario.validate(b, fixture, index, raw)
		}
		if scenario.tool == "semantic_diagnostics" {
			coldHotPageIdentity(b, raw, scenario.args(index))
		}
		if scenario.after != nil {
			scenario.after(b, fixture, client, index)
		}
		coldHotExactFiles(b, fixture)
		starts := coldHotStarts(b, fixture.counter)
		if starts != baseStarts {
			result.status = "RESTART"
			result.restarts = starts - baseStarts
			result.starts = starts
			result.hotSamples = len(latencies) - 1
			result.prereqNote = "post-call process continuity failed; hot p99 withheld"
			b.Logf("MCP_COST_REJECTED tool=%s samples=%d starts=%d expected=%d", scenario.tool, index+1, starts, baseStarts)
			return result
		}
	}
	result.hotP99 = coldHotP99(b, latencies)
	result.hotSamples = len(latencies)
	result.starts = coldHotStarts(b, fixture.counter)
	return result
}

func coldHotExactFiles(tb testing.TB, fixture managedMCPFixture) {
	tb.Helper()
	if count := countGoSourceFiles(tb, fixture.root); count != 20 {
		tb.Fatalf("fixture has %d Go source files, want exactly 20", count)
	}
}

func coldHotFinalState(b *testing.B, fixture managedMCPFixture, scenario coldHotTool, client *warmMCPCostClient, expectedStarts int) {
	b.Helper()
	coldHotExactFiles(b, fixture)
	if scenario.tool == "semantic_diagnostics" {
		coldHotVerifyDiagnostics(b, client.toolResult(b, "semantic_verify", coldHotVerifyArgs()), 50)
	} else {
		warmVerify(b, client)
	}
	if starts := coldHotStarts(b, fixture.counter); starts != expectedStarts {
		b.Fatalf("hot %s final verification restarted gopls: got %d, want %d", scenario.tool, starts, expectedStarts)
	}
}

func coldHotStarts(tb testing.TB, counter string) int {
	tb.Helper()
	content, err := os.ReadFile(counter)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		tb.Fatalf("read gopls start counter: %v", err)
	}
	return strings.Count(string(content), "start\n")
}

func coldHotVerifyArgs() map[string]any {
	return map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true}
}

type coldHotEnvironment struct {
	values map[string]coldHotEnvValue
}

type coldHotEnvValue struct {
	value string
	set   bool
}

func saveColdHotEnvironment() coldHotEnvironment {
	values := make(map[string]coldHotEnvValue)
	for _, name := range []string{"PATH", "SEMEDIT_REAL_GOPLS", "SEMEDIT_GOPLS_COUNT", "SEMEDIT_GOPLS_PIDS"} {
		value, set := os.LookupEnv(name)
		values[name] = coldHotEnvValue{value: value, set: set}
	}
	return coldHotEnvironment{values: values}
}

func (environment coldHotEnvironment) restore(tb testing.TB) {
	for name, value := range environment.values {
		if value.set {
			if err := os.Setenv(name, value.value); err != nil {
				tb.Errorf("restore %s: %v", name, err)
			}
		} else {
			if err := os.Unsetenv(name); err != nil {
				tb.Errorf("restore %s: %v", name, err)
			}
		}
	}
}

func coldHotResetSourceState(tb testing.TB, fixture managedMCPFixture, tool string, index int) {
	switch tool {
	case "refactor_rename":
		from, _ := coldHotName(index)
		source := "package sample\n\nfunc " + from + "() int { return 1 }\n"
		coldHotWrite(tb, fixture.root, "value.go", source)
		entries, err := os.ReadDir(fixture.root)
		if err != nil {
			tb.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "consumer") {
				continue
			}
			path := filepath.Join(fixture.root, entry.Name(), "consumer.go")
			content, err := os.ReadFile(path)
			if err != nil {
				tb.Fatal(err)
			}
			updated := strings.ReplaceAll(string(content), "sample.Value()", "sample."+from+"()")
			updated = strings.ReplaceAll(updated, "sample.Renamed()", "sample."+from+"()")
			coldHotWrite(tb, fixture.root, filepath.Join(entry.Name(), "consumer.go"), updated)
		}
	case "semantic_replace_body", "semantic_stage", "semantic_commit", "semantic_stage_abort":
		coldHotWrite(tb, fixture.root, "value.go", "package sample\n\nfunc Value() int { return 1 }\n")
	case "semantic_batch":
		coldHotWrite(tb, fixture.root, "value.go", "package sample\n\nfunc Value() int { return 1 }\n")
		coldHotWrite(tb, fixture.root, "consumer1/consumer.go", "package consumer\n\nimport sample \"example.com/mcpreuse\"\n\nfunc Read() int { return sample.Value() }\nfunc Other() int { return 10 }\n")
	}
}

func coldHotToolCases() []coldHotTool {
	return []coldHotTool{
		{name: "lookup", tool: "semantic_lookup", cold: true, args: coldHotFixed(map[string]any{"file": "value.go", "symbol": "Value"}), validate: coldHotBaseline("semantic_lookup"), budget: 10 * time.Millisecond},
		{name: "inspect_symbol", tool: "semantic_inspect_symbol", cold: true, args: coldHotFixed(map[string]any{"file": "value.go", "symbol": "Value"}), validate: coldHotBaseline("semantic_inspect_symbol"), budget: 10 * time.Millisecond},
		{name: "file_outline", tool: "semantic_outline", cold: true, args: coldHotFixed(map[string]any{"path": "value.go"}), validate: coldHotBaseline("semantic_outline"), budget: 10 * time.Millisecond},
		{name: "workspace_outline", tool: "semantic_workspace_outline", cold: true, args: coldHotFixed(map[string]any{"path": "."}), validate: coldHotBaseline("semantic_workspace_outline"), budget: 10 * time.Millisecond},
		{name: "references", tool: "semantic_find_references", cold: true, args: coldHotFixed(map[string]any{"file": "value.go", "symbol": "Value"}), validate: coldHotBaseline("semantic_find_references"), budget: 10 * time.Millisecond},
		{name: "rename", tool: "refactor_rename", cold: true, args: func(index int) map[string]any {
			from, to := coldHotName(index)
			return map[string]any{"file": "value.go", "symbol": from, "to": to}
		}, validate: coldHotRename, budget: time.Second},
		{name: "replace_body", tool: "semantic_replace_body", cold: true, args: func(index int) map[string]any {
			return map[string]any{"file": "value.go", "symbol": "Value", "body": fmt.Sprintf("return %d", index+2)}
		}, validate: coldHotBody, budget: time.Second},
		{name: "inspect_construct", tool: "semantic_inspect_construct", cold: true, args: coldHotFixed(map[string]any{"file": "value.go", "in_function": "Check", "kind": "if", "discriminator": "v > 0"}), validate: coldHotConstruct, budget: 10 * time.Millisecond},
		{name: "metrics", tool: "semantic_metrics", args: coldHotFixed(map[string]any{"limit": 100}), prepare: coldHotSeedMetrics, prepareOnce: true, validate: coldHotMetrics, budget: 10 * time.Millisecond, prereqNote: "cold N/A; seeded 100 records before timed hot call"},
		{name: "snapshot", tool: "semantic_snapshot", cold: true, args: func(index int) map[string]any {
			return map[string]any{"label": fmt.Sprintf("cold-hot-%d", index), "paths": []string{"value.go"}}
		}, validate: coldHotSnapshot, budget: time.Second},
		{name: "verify", tool: "semantic_verify", cold: true, args: coldHotFixed(map[string]any{"path": ".", "language": "go", "check_only": true, "skip_hooks": true}), validate: coldHotVerify, budget: time.Second},
		{name: "organize_imports", tool: "semantic_organize_imports", cold: true, args: coldHotFixed(map[string]any{"file": "value.go"}), prepare: coldHotDirtyImports, validate: coldHotImports, budget: time.Second},
		{name: "batch", tool: "semantic_batch", cold: true, args: func(index int) map[string]any {
			first, second := coldHotBodyPair(index)
			return map[string]any{"edits": []map[string]any{{"tool": "semantic_replace_body", "params": map[string]any{"file": "value.go", "symbol": "Value", "body": first}}, {"tool": "semantic_replace_body", "params": map[string]any{"file": "consumer1/consumer.go", "symbol": "Other", "body": second}}}}
		}, validate: coldHotBatch, budget: time.Second},
		{name: "stage", tool: "semantic_stage", cold: true, args: coldHotFixed(map[string]any{}), validate: coldHotStage, after: coldHotCompleteStage, budget: time.Second},
		{name: "commit", tool: "semantic_commit", args: coldHotFixed(map[string]any{"skip_hooks": true}), prepare: coldHotPrepareStage, validate: coldHotCommit, budget: time.Second, prereqNote: "cold N/A; staged actual body edit before timed commit"},
		{name: "abort", tool: "semantic_stage_abort", args: coldHotFixed(map[string]any{}), prepare: coldHotPrepareStage, validate: coldHotAbort, after: coldHotAbortSync, budget: time.Second, prereqNote: "cold N/A; staged actual body edit before timed abort"},
		{name: "diagnostics_page", tool: "semantic_diagnostics", args: nil, validate: coldHotDiagnosticPage, budget: 10 * time.Millisecond, prereqNote: "cold N/A; 50-item report capture is a prerequisite"},
	}
}

func coldHotFixed(args map[string]any) func(int) map[string]any {
	return func(int) map[string]any { return args }
}

func coldHotName(index int) (string, string) {
	if index%2 == 0 {
		return "Value", "Renamed"
	}
	return "Renamed", "Value"
}

func coldHotBodyPair(index int) (string, string) {
	if index%2 == 0 {
		return "return 2", "return 20"
	}
	return "return 1", "return 10"
}

func configureColdHotFixture(b testing.TB, fixture managedMCPFixture, tool string) {
	b.Helper()
	if tool == "semantic_inspect_construct" {
		coldHotWrite(b, fixture.root, "value.go", "package sample\n\nfunc Value() int { return 1 }\nfunc Check(v int) int { if v > 0 { return v }; return 0 }\n")
	}
	if tool == "semantic_batch" {
		coldHotEnsureBatch(b, fixture, nil, 0)
	}
	if tool == "semantic_organize_imports" {
		coldHotWrite(b, fixture.root, "value.go", "package sample\n\nfunc Value() int { return 1 }\n")
	}
	if tool == "semantic_diagnostics" {
		var source strings.Builder
		source.WriteString("package sample\nfunc Value() int { return 1 }\n")
		for index := range 50 {
			fmt.Fprintf(&source, "var broken%03d = missing%03d\n", index, index)
		}
		coldHotWrite(b, fixture.root, "value.go", source.String())
	}
}

func countGoSourceFiles(b testing.TB, root string) int {
	b.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && entry.IsDir() && entry.Name() == ".scratch" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			count++
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	return count
}

func coldHotWrite(b testing.TB, root, name, source string) {
	b.Helper()
	if err := pipeline.WriteAtomic(filepath.Join(root, name), []byte(source)); err != nil {
		b.Fatal(err)
	}
}

func coldHotBaseline(tool string) func(testing.TB, managedMCPFixture, int, []byte) {
	return func(tb testing.TB, fixture managedMCPFixture, index int, raw []byte) {
		var b *testing.B
		if value, ok := tb.(*testing.B); ok {
			b = value
		} else {
			tb.Fatalf("unexpected test type %T", tb)
		}
		warmValidateBaseline(b, fixture.root, 20, tool, index, raw)
	}
}

func coldHotPrepareStage(tb testing.TB, fixture managedMCPFixture, client *warmMCPCostClient, index int) {
	target := fmt.Sprintf("return %d", index%2+2)
	current, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		tb.Fatal(err)
	}
	if strings.Contains(string(current), target) {
		tb.Fatalf("stage prerequisite is a no-op: %s", target)
	}
	coldHotStage(tb, fixture, index, client.toolResult(tb, "semantic_stage", map[string]any{}))
	coldHotStagedEdit(tb, client.toolResult(tb, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": target}))
}

func coldHotCompleteStage(tb testing.TB, fixture managedMCPFixture, client *warmMCPCostClient, index int) {
	target := fmt.Sprintf("return %d", index%2+2)
	coldHotStagedEdit(tb, client.toolResult(tb, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": target}))
	coldHotCommit(tb, fixture, index, client.toolResult(tb, "semantic_commit", map[string]any{"skip_hooks": true}))
}

func coldHotStagedEdit(tb testing.TB, raw []byte) {
	tb.Helper()
	var receipt struct {
		StageReceipt operation.StageReceipt `json:"stage_receipt"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		tb.Fatal(err)
	}
	if receipt.StageReceipt.Status != "staged" || receipt.StageReceipt.StageID == "" || receipt.StageReceipt.StepCount != 1 {
		tb.Fatalf("staged edit did not retain one changed step: %s", raw)
	}
}

func coldHotEnsureBatch(tb testing.TB, fixture managedMCPFixture, _ *warmMCPCostClient, _ int) {
	path := filepath.Join(fixture.root, "consumer1", "consumer.go")
	source, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	if !strings.Contains(string(source), "func Other()") {
		coldHotWrite(tb, fixture.root, "consumer1/consumer.go", string(source)+"\nfunc Other() int { return 10 }\n")
	}
}

func coldHotSeedMetrics(tb testing.TB, _ managedMCPFixture, client *warmMCPCostClient, _ int) {
	for range 100 {
		client.toolResult(tb, "semantic_lookup", map[string]any{"file": "value.go", "symbol": "Value"})
	}
}

func coldHotDirtyImports(tb testing.TB, fixture managedMCPFixture, _ *warmMCPCostClient, _ int) {
	coldHotWrite(tb, fixture.root, "value.go", "package sample\n\nimport \"fmt\"\n\nfunc Value() int { return 1 }\n")
}

func coldHotCaptureDiagnostics(tb testing.TB, _ managedMCPFixture, client *warmMCPCostClient, _ int) map[string]any {
	raw := client.toolResult(tb, "semantic_verify", coldHotVerifyArgs())
	coldHotVerifyDiagnostics(tb, raw, 50)
	var capture struct {
		Report struct {
			Status         string         `json:"status"`
			ID             string         `json:"report_id"`
			RemainingCount int            `json:"remaining_count"`
			Arguments      map[string]any `json:"arguments"`
		} `json:"diagnostic_report"`
	}
	if err := json.Unmarshal(raw, &capture); err != nil {
		tb.Fatal(err)
	}
	if capture.Report.Status != "available" || capture.Report.ID == "" || capture.Report.RemainingCount < 50 || len(capture.Report.Arguments) == 0 {
		tb.Fatalf("diagnostic capture did not retain 50 items: %s", raw)
	}
	capture.Report.Arguments["limit"] = 50
	return capture.Report.Arguments
}

func coldHotVerifyDiagnostics(tb testing.TB, raw []byte, minimum int) {
	tb.Helper()
	verified := warmResult[operation.VerifyRes](coldHotBenchmark(tb), raw)
	if verified.Failure != "" || len(verified.Diagnostics) < minimum || (minimum == 0 && len(verified.Diagnostics) != 0) {
		tb.Fatalf("verification has failure or unexpected diagnostics: %s", raw)
	}
	goVerified := false
	for _, coverage := range verified.Coverage {
		if coverage.Language == "go" && coverage.Status == "verified" {
			goVerified = true
		}
	}
	if !goVerified {
		tb.Fatalf("verification omitted completed Go coverage: %s", raw)
	}
}

func coldHotBenchmark(tb testing.TB) *testing.B {
	tb.Helper()
	benchmark, ok := tb.(*testing.B)
	if !ok {
		tb.Fatalf("expected benchmark context, got %T", tb)
	}
	return benchmark
}

func coldHotBody(tb testing.TB, fixture managedMCPFixture, index int, raw []byte) {
	warmPersistedEdit(coldHotBenchmark(tb), raw)
	warmDiagnosticReport(coldHotBenchmark(tb), raw)
	warmSourceContains(coldHotBenchmark(tb), fixture.root, "value.go", fmt.Sprintf("return %d", index+2))
}

func coldHotRename(tb testing.TB, fixture managedMCPFixture, index int, raw []byte) {
	_, to := coldHotName(index)
	result := warmResult[struct {
		Status           string `json:"status"`
		ApplicationState string `json:"application_state"`
		Symbol           string `json:"symbol"`
		ActiveCount      int    `json:"active_count"`
	}](coldHotBenchmark(tb), raw)
	if result.Status != "renamed" || result.ApplicationState != "persisted" || result.Symbol != to || result.ActiveCount != 0 {
		tb.Fatalf("rename did not publish clean target %s: %s", to, raw)
	}
	warmDiagnosticReport(coldHotBenchmark(tb), raw)
	warmSourceContains(coldHotBenchmark(tb), fixture.root, "value.go", "func "+to+"()")
	for consumer := 1; consumer < 20; consumer++ {
		warmSourceContains(coldHotBenchmark(tb), fixture.root, fmt.Sprintf("consumer%d/consumer.go", consumer), "sample."+to+"()")
	}
}

func coldHotConstruct(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	result := warmResult[operation.InspectConstructRes](coldHotBenchmark(tb), raw)
	if result.File != "value.go" || result.InFunction != "Check" || result.SelectionRevision == "" || len(result.Candidates) != 1 {
		tb.Fatalf("construct inspection omitted exact candidate: %s", raw)
	}
}

func coldHotMetrics(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	result := warmResult[struct {
		Records []struct {
			Tool     string `json:"tool"`
			Sequence uint64 `json:"sequence"`
		} `json:"records"`
	}](coldHotBenchmark(tb), raw)
	if len(result.Records) != 100 {
		tb.Fatalf("metrics returned %d records, want 100", len(result.Records))
	}
	for index, record := range result.Records {
		if record.Tool != "semantic_lookup" || record.Sequence == 0 || (index > 0 && record.Sequence >= result.Records[index-1].Sequence) {
			tb.Fatalf("metrics record %d is not a seeded lookup in descending order: %s", index, raw)
		}
	}
}

func coldHotSnapshot(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	result := warmResult[struct {
		Status string `json:"status"`
		ID     string `json:"snapshot_id"`
		Count  int    `json:"captured_file_count"`
	}](coldHotBenchmark(tb), raw)
	if result.Status != "created" || result.ID == "" || result.Count != 1 {
		tb.Fatalf("invalid snapshot result: %+v", result)
	}
}

func coldHotVerify(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	coldHotVerifyDiagnostics(tb, raw, 0)
}

func coldHotImports(tb testing.TB, fixture managedMCPFixture, _ int, raw []byte) {
	result := warmResult[struct {
		Status           string   `json:"status"`
		ApplicationState string   `json:"application_state"`
		ChangedFiles     []string `json:"changed_files"`
	}](coldHotBenchmark(tb), raw)
	if result.Status != "organized" || result.ApplicationState != "persisted" || len(result.ChangedFiles) != 1 || result.ChangedFiles[0] != "value.go" {
		tb.Fatalf("imports were not persisted in value.go: %s", raw)
	}
	warmDiagnosticReport(coldHotBenchmark(tb), raw)
	source, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		tb.Fatal(err)
	}
	if strings.Contains(string(source), "fmt") {
		tb.Fatalf("organize imports did not remove unused import: %s", source)
	}
}

func coldHotBatch(tb testing.TB, fixture managedMCPFixture, index int, raw []byte) {
	result := warmResult[mcp.BatchResponse](coldHotBenchmark(tb), raw)
	if result.Status != "ok" || len(result.Results) != 2 || result.FinalDiff == "" || result.Verification == nil || result.Results[0].Status != "ok" || result.Results[1].Status != "ok" {
		tb.Fatalf("batch did not complete both edits and verification: %s", raw)
	}
	warmEmbeddedVerify(coldHotBenchmark(tb), result.Verification)
	warmDiagnosticReport(coldHotBenchmark(tb), raw)
	first, second := coldHotBodyPair(index)
	warmSourceContains(tb.(*testing.B), fixture.root, "value.go", first)
	warmSourceContains(tb.(*testing.B), fixture.root, "consumer1/consumer.go", second)
}

func coldHotStage(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	result := warmResult[operation.StageReceipt](coldHotBenchmark(tb), raw)
	if result.Status != "staged" || result.StageID == "" || result.StepCount != 0 {
		tb.Fatalf("stage response omitted stage_id: %s", raw)
	}
}
func coldHotCommit(tb testing.TB, fixture managedMCPFixture, index int, raw []byte) {
	result := warmResult[operation.StageCommitResult](coldHotBenchmark(tb), raw)
	if result.Status != "ok" || result.PublishedFiles != 1 || result.Verification == nil || result.FinalDiff == "" || len(result.Results) != 1 || result.Results[0].Status != "ok" {
		tb.Fatalf("commit did not publish one verified file: %+v", result)
	}
	warmEmbeddedVerify(coldHotBenchmark(tb), result.Verification)
	warmDiagnosticReport(coldHotBenchmark(tb), raw)
	warmSourceContains(coldHotBenchmark(tb), fixture.root, "value.go", fmt.Sprintf("return %d", index%2+2))
}
func coldHotAbort(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	result := warmResult[operation.StageReceipt](coldHotBenchmark(tb), raw)
	if result.Status != "aborted" || result.StageID == "" {
		tb.Fatalf("stage abort did not abort: %s", raw)
	}
}
func coldHotAbortSync(tb testing.TB, fixture managedMCPFixture, client *warmMCPCostClient, _ int) {
	warmSourceContains(coldHotBenchmark(tb), fixture.root, "value.go", "return 1")
	coldHotVerifyDiagnostics(tb, client.toolResult(tb, "semantic_verify", coldHotVerifyArgs()), 0)
}
func coldHotDiagnosticPage(tb testing.TB, _ managedMCPFixture, _ int, raw []byte) {
	result := warmResult[struct {
		ReportID string `json:"report_id"`
		Status   string `json:"status"`
		Category string `json:"category"`
		Total    int    `json:"total"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
		Items    []struct {
			Message string `json:"message"`
		} `json:"items"`
	}](coldHotBenchmark(tb), raw)
	if result.ReportID == "" || result.Status != "available" || result.Category != "active" || result.Total < 50 || result.Offset != 0 || result.Limit != 50 || len(result.Items) != 50 || result.Items[0].Message == "" || result.Items[49].Message == "" {
		tb.Fatalf("diagnostic page was incomplete: %s", raw)
	}
}

func coldHotPageIdentity(tb testing.TB, raw []byte, arguments map[string]any) {
	tb.Helper()
	page := warmResult[struct {
		ReportID string `json:"report_id"`
	}](coldHotBenchmark(tb), raw)
	id, ok := arguments["report_id"].(string)
	if !ok || id == "" || page.ReportID != id {
		tb.Fatalf("diagnostic page returned a different captured report: %s", raw)
	}
}

func coldHotP99(tb testing.TB, samples []time.Duration) time.Duration {
	if len(samples) == 0 {
		tb.Fatal("cannot report p99 from an empty sample set")
	}
	ordered := append([]time.Duration(nil), samples...)
	slices.Sort(ordered)
	return ordered[(99*len(ordered)+99)/100-1]
}

func printColdHotTable(rows []coldHotResult, samples coldHotSamples) {
	fmt.Printf("MCP cold/hot costs: 20 packages, exactly 20 Go source files; initialize excluded; cold starts fresh MCP/gopls when needed; shared disk caches retained; cold=%d hot=%d\n", samples.cold, samples.hot)
	fmt.Printf("%-30s %12s %15s %12s %17s %10s %s\n", "tool", "cold p99", "workflow p99", "hot p99", "samples c/w/h", "status", "cold prerequisite")
	for _, row := range rows {
		cold := "N/A"
		workflow := "N/A"
		if row.workflowRestart {
			workflow = "WITHHELD"
		}
		if samples.cold >= 100 && row.coldSamples >= 100 {
			cold = row.coldP99.Round(time.Microsecond).String()
		}
		if samples.cold >= 100 && row.workflowSamples >= 100 && !row.workflowRestart {
			workflow = row.workflowP99.Round(time.Microsecond).String()
		}
		hot := "WITHHELD"
		if row.status == "SMOKE" {
			if row.coldSamples > 0 {
				cold = "SMOKE"
			}
			if row.workflowSamples > 0 {
				workflow = "SMOKE"
			}
			hot = "SMOKE"
		} else if row.hotSamples >= 100 && row.status != "RESTART" && row.status != "FAILED" {
			hot = row.hotP99.Round(time.Microsecond).String()
		}
		fmt.Printf("%-30s %12s %15s %12s %4d/%-4d/%-4d %10s %s\n", row.tool, cold, workflow, hot, row.coldSamples, row.workflowSamples, row.hotSamples, row.status, row.prereqNote)
	}
}
