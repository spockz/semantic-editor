// Package mcp_test measures warm MCP operation costs with real retained gopls and explicit latency budgets.

package mcp_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

type warmMCPCostClient struct {
	encoder           *json.Encoder
	reader            *bufio.Reader
	requests          *io.PipeWriter
	responses         *io.PipeReader
	cancel            context.CancelFunc
	done              chan error
	closeOnce         sync.Once
	nextID            int
	lastResponseBytes int
}

const warmMCPRequestTimeout = 30 * time.Second

type warmMCPCostCase struct {
	name      string
	tool      string
	arguments map[string]any
	budget    time.Duration
}

func BenchmarkMCPWarmOperationCost(b *testing.B) {
	for _, packages := range []int{1, 20} {
		b.Run(fmt.Sprintf("packages_%d", packages), func(b *testing.B) {
			cases := []warmMCPCostCase{
				{"lookup", "semantic_lookup", map[string]any{"file": "value.go", "symbol": "Value"}, 10 * time.Millisecond},
				{"inspect", "semantic_inspect_symbol", map[string]any{"file": "value.go", "symbol": "Value"}, 10 * time.Millisecond},
				{"file_outline", "semantic_outline", map[string]any{"path": "value.go"}, 10 * time.Millisecond},
				{"workspace_outline", "semantic_workspace_outline", map[string]any{"path": "."}, 10 * time.Millisecond},
				{"references", "semantic_find_references", map[string]any{"file": "value.go", "symbol": "Value"}, 10 * time.Millisecond},
				{"symbol_rename", "semantic_rename", nil, time.Second},
			}
			for _, scenario := range cases {
				b.Run(scenario.name, func(b *testing.B) { benchmarkWarmMCPCost(b, packages, scenario) })
			}
		})
	}
}

func benchmarkWarmMCPCost(b *testing.B, packages int, scenario warmMCPCostCase) {
	fixture := newWarmMCPCostFixture(b, packages)
	client := newWarmMCPCostClient(b, fixture.root)
	client.call(b, "initialize", map[string]any{})
	warmVerify(b, client)
	warmArguments := scenario.arguments
	if scenario.tool == "semantic_rename" {
		warmArguments = map[string]any{"file": "value.go", "symbol": "Value", "to": "Renamed"}
	}
	client.toolResult(b, scenario.tool, warmArguments)
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		arguments := warmBaselineArguments(scenario, index)
		result := observed.call(b, client, "operation", scenario.tool, arguments)
		b.StopTimer()
		warmValidateBaseline(b, fixture.root, packages, scenario.tool, index, result)
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("gopls restarted: %d", starts)
		}
		b.StartTimer()
	}
	b.StopTimer()
	warmContinuity(b, fixture, client)
	observed.report(b, map[string]time.Duration{"operation": scenario.budget}, managedSessionStarts(b, fixture))
}

func warmBaselineArguments(scenario warmMCPCostCase, index int) map[string]any {
	if scenario.tool != "semantic_rename" {
		return scenario.arguments
	}
	from, to := "Renamed", "Value"
	if index%2 != 0 {
		from, to = to, from
	}
	return map[string]any{"file": "value.go", "symbol": from, "to": to}
}

func warmValidateBaseline(b *testing.B, root string, packages int, tool string, index int, raw json.RawMessage) {
	b.Helper()
	switch tool {
	case "semantic_rename":
		expected := "Value"
		if index%2 != 0 {
			expected = "Renamed"
		}
		warmSourceContains(b, root, "value.go", "func "+expected+"(")
		if packages > 1 {
			warmSourceContains(b, root, "consumer1/consumer.go", "sample."+expected+"()")
		}
		warmDiagnosticReport(b, raw)
	case "semantic_lookup":
		value := warmResult[operation.LookupRes](b, raw)
		if value.Ambiguous || value.Symbol != "Value" || value.File != "value.go" {
			b.Fatalf("lookup did not resolve Value: %s", raw)
		}
	case "semantic_inspect_symbol":
		projection := warmResult[operation.InspectProjection](b, raw)
		var value backend.InspectResult
		if err := json.Unmarshal([]byte(projection.Inspection), &value); err != nil {
			b.Fatalf("decode captured inspection: %v: %s", err, raw)
		}
		if !value.Scope.Complete || len(value.Matches) != 1 || value.Matches[0].Name != "Value" || !strings.Contains(value.Matches[0].Source, "return 1") || value.Matches[0].Revision == "" {
			b.Fatalf("inspect omitted exact Value source: %s", raw)
		}
	case "semantic_outline":
		value := warmResult[backend.OutlineResult](b, raw)
		if !value.Scope.Complete || len(value.Files) != 1 || value.Files[0].File != "value.go" || len(value.Files[0].Symbols) != 1 || value.Files[0].Symbols[0].Name != "Value" {
			b.Fatalf("file outline omitted Value: %s", raw)
		}
	case "semantic_workspace_outline":
		value := warmResult[backend.WorkspaceOutlineResult](b, raw)
		if !value.Scope.Complete || len(value.Modules) != 1 || len(value.Packages) != packages {
			b.Fatalf("workspace outline omitted fixture packages: %s", raw)
		}
	case "semantic_find_references":
		value := warmResult[backend.ReferencesResult](b, raw)
		if !value.Scope.Complete || value.Symbol.Name != "Value" || len(value.References) != packages-1 {
			b.Fatalf("references omitted consumer uses: %s", raw)
		}
	default:
		b.Fatalf("unvalidated baseline tool %s", tool)
	}
}

func newWarmMCPCostClient(b testing.TB, root string) *warmMCPCostClient {
	b.Helper()
	input, requests := io.Pipe()
	responses, output := io.Pipe()
	ctx, cancel := context.WithTimeout(b.Context(), 2*time.Hour)
	server := mcp.NewServer("full", root, output, mcp.WithEnabledLanguages("go"))
	done := make(chan error, 1)
	go func() {
		err := server.Serve(ctx, input)
		done <- errors.Join(err, output.Close())
	}()
	client := &warmMCPCostClient{encoder: json.NewEncoder(requests), reader: bufio.NewReader(responses), requests: requests, responses: responses, cancel: cancel, done: done}
	b.Cleanup(func() { client.close(b) })
	return client
}

func (client *warmMCPCostClient) close(t testing.TB) {
	t.Helper()
	client.closeOnce.Do(func() {
		if err := client.requests.Close(); err != nil {
			t.Error(err)
		}
		select {
		case err := <-client.done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
		case <-time.After(10 * time.Second):
			client.cancel()
			t.Error("MCP shutdown exceeded 10 seconds")
		}
		client.cancel()
		if err := client.responses.Close(); err != nil {
			t.Error(err)
		}
	})
}

func readWarmMCPFrame(reader *bufio.Reader) ([]byte, error) {
	return reader.ReadBytes('\n')
}

func TestWarmMCPFrameDrainsDelimiter(t *testing.T) {
	input, output := io.Pipe()
	defer func() {
		if err := input.Close(); err != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if err := output.Close(); err != nil {
			t.Error(err)
		}
	}()
	const objectBytes = 512
	payload := []byte(`{"padding":"` + strings.Repeat("x", objectBytes-len(`{"padding":""}`)) + `"}`)
	payload = append(payload, '\n')
	frame := payload
	written := make(chan error, 1)
	go func() { _, err := output.Write(frame); written <- err }()
	got, err := readWarmMCPFrame(bufio.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, frame) {
		t.Fatalf("frame length %d, want %d", len(got), len(frame))
	}
	select {
	case err := <-written:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("response writer remained blocked on the frame delimiter")
	}
}

func (client *warmMCPCostClient) call(b testing.TB, method string, params any) json.RawMessage {
	b.Helper()
	var requestState atomic.Uint32
	watchdog := time.AfterFunc(warmMCPRequestTimeout, func() {
		if !requestState.CompareAndSwap(0, 2) {
			return
		}
		client.cancel()
		// io.Pipe CloseWithError always returns nil; closing both ends releases blocked I/O.
		_ = client.requests.CloseWithError(context.DeadlineExceeded)
		_ = client.responses.CloseWithError(context.DeadlineExceeded)
	})
	defer watchdog.Stop()
	client.nextID++
	request := map[string]any{"jsonrpc": "2.0", "id": client.nextID, "method": method, "params": params}
	if err := client.encoder.Encode(request); err != nil {
		if requestState.Load() == 2 {
			b.Fatalf("MCP %s exceeded %s while writing request: %v", method, warmMCPRequestTimeout, err)
		}
		b.Fatal(err)
	}
	for {
		frame, err := readWarmMCPFrame(client.reader)
		if err != nil {
			if requestState.Load() == 2 {
				b.Fatalf("MCP %s exceeded %s awaiting response frame: %v", method, warmMCPRequestTimeout, err)
			}
			b.Fatalf("read MCP %s response frame: %v", method, err)
		}
		var response struct {
			ID     int             `json:"id"`
			Error  json.RawMessage `json:"error"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(frame, &response); err != nil {
			b.Fatalf("decode MCP %s response frame: %v", method, err)
		}
		client.lastResponseBytes += len(frame)
		if response.ID == 0 {
			continue
		}
		if response.ID != client.nextID {
			b.Fatalf("response id %d, want %d", response.ID, client.nextID)
		}
		if len(response.Error) > 0 {
			b.Fatalf("MCP %s failed: %s", method, response.Error)
		}
		if len(response.Result) == 0 {
			b.Fatalf("MCP %s omitted result", method)
		}
		if !requestState.CompareAndSwap(0, 1) {
			b.Fatalf("MCP %s exceeded %s before completing response", method, warmMCPRequestTimeout)
		}
		return response.Result
	}
}

func (client *warmMCPCostClient) toolResult(b testing.TB, name string, arguments map[string]any) json.RawMessage {
	b.Helper()
	raw := client.call(b, "tools/call", map[string]any{"name": name, "arguments": arguments})
	var result struct {
		IsError           bool            `json:"isError"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		b.Fatal(err)
	}
	if result.IsError {
		b.Fatalf("%s failed: %s", name, raw)
	}
	if len(result.StructuredContent) == 0 {
		b.Fatalf("%s omitted structured result: %s", name, raw)
	}
	return result.StructuredContent
}

func reportWarmMCPCost(b *testing.B, samples []time.Duration, budget time.Duration) {
	if len(samples) == 0 {
		b.Fatal("warm benchmark has no measured samples")
	}
	ordered := slices.Clone(samples)
	slices.Sort(ordered)
	p99 := ordered[(99*len(ordered)+99)/100-1]
	b.ReportMetric(float64(ordered[(len(ordered)-1)/2])/float64(time.Millisecond), "p50_ms")
	b.ReportMetric(float64(p99)/float64(time.Millisecond), "p99_ms")
	b.ReportMetric(float64(ordered[len(ordered)-1])/float64(time.Millisecond), "max_ms")
	b.ReportMetric(float64(p99)/float64(budget), "p99/budget")
	b.ReportMetric(float64(p99)/float64(time.Second), "p99/ceiling")
	b.ReportMetric(float64(len(samples)), "samples")
	if os.Getenv("SEMEDIT_BENCH_ENFORCE_BUDGETS") == "1" && (p99 >= time.Second || p99 > budget) {
		b.Errorf("warm p99 %s exceeds target %s or one-second ceiling", p99, budget)
	}
}

func newWarmMCPCostFixture(t testing.TB, packages int) managedMCPFixture {
	t.Helper()
	fixture := newManagedMCPFixture(t)
	for index := 1; index < packages; index++ {
		dir := filepath.Join(fixture.root, fmt.Sprintf("consumer%d", index))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		source := "package consumer\n\nimport sample \"example.com/mcpreuse\"\n\nfunc Read() int { return sample.Value() }\n"
		if err := pipeline.WriteAtomic(filepath.Join(dir, "consumer.go"), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}
