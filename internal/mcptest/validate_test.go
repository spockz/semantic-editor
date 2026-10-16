// Package mcptest validates the test response validator against advertised schema fixtures.
package mcptest

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"context"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/operation"
)

func TestResponseValidatorChecksErrorResultsAndIntegerPrecision(t *testing.T) {
	schema, err := compileSchema(json.RawMessage(`{"type":"object","properties":{"result":{"type":"integer"}},"required":["result"],"additionalProperties":false}`))
	if err != nil {
		t.Fatalf("compile fixture schema: %v", err)
	}
	var failures []string
	validator := &responseValidator{
		testName:        t.Name(),
		report:          func(message string) { failures = append(failures, message) },
		advertised:      map[string]*jsonschema.Schema{"sample": schema},
		advertisedReady: true,
		full:            make(map[string]*jsonschema.Schema),
	}
	validator.validateToolResult("sample", json.RawMessage(`{"isError":true,"structuredContent":{"result":9007199254740993}}`))
	if len(failures) != 0 {
		t.Fatalf("large integer error result failed validation: %v", failures)
	}
	validator.validateToolResult("sample", json.RawMessage(`{"isError":true,"structuredContent":{"wrong":1}}`))
	if len(failures) != 1 || !strings.Contains(failures[0], "sample") || !strings.Contains(failures[0], `"keywordLocation":"/required"`) {
		t.Fatalf("invalid error result diagnostics = %v, want tool name and schema path", failures)
	}
}

func TestResponseValidatorCorrelatesIDsAndPreservesWireBytes(t *testing.T) {
	schema, err := compileSchema(json.RawMessage(`{"type":"object","properties":{"ok":{"const":true}},"required":["ok"],"additionalProperties":false}`))
	if err != nil {
		t.Fatalf("compile fixture schema: %v", err)
	}
	var output bytes.Buffer
	var failures []string
	validator := &responseValidator{
		testName:        t.Name(),
		report:          func(message string) { failures = append(failures, message) },
		dst:             &output,
		requests:        make(map[string]requestInfo),
		advertised:      map[string]*jsonschema.Schema{"sample": schema},
		advertisedReady: true,
		full:            make(map[string]*jsonschema.Schema),
	}
	for _, id := range []string{`"string-id"`, `7`} {
		validator.recordRequest([]byte(`{"jsonrpc":"2.0","id":` + id + `,"method":"tools/call","params":{"name":"sample"}}`))
		response := []byte(`{"jsonrpc":"2.0","id":` + id + `,"result":{"isError":true,"structuredContent":{"ok":true}}}` + "\n")
		if n, err := validator.Write(response); err != nil || n != len(response) {
			t.Fatalf("Write() = %d, %v; want %d, nil", n, err, len(response))
		}
	}
	if len(failures) != 0 {
		t.Fatalf("valid correlated responses failed: %v", failures)
	}
	if got, want := output.String(), "{\"jsonrpc\":\"2.0\",\"id\":\"string-id\",\"result\":{\"isError\":true,\"structuredContent\":{\"ok\":true}}}\n{\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"isError\":true,\"structuredContent\":{\"ok\":true}}}\n"; got != want {
		t.Fatalf("forwarded bytes = %q, want %q", got, want)
	}
}

func TestResponseValidatorReportsMalformedOutputAndSkipsProtocolErrors(t *testing.T) {
	var failures []string
	validator := &responseValidator{
		testName:   t.Name(),
		report:     func(message string) { failures = append(failures, message) },
		requests:   map[string]requestInfo{"1": {method: "tools/call", tool: "sample"}, "2": {method: "tools/call", tool: "sample"}},
		advertised: make(map[string]*jsonschema.Schema),
		full:       make(map[string]*jsonschema.Schema),
	}
	validator.recordResponse([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"invalid params"}}`))
	if len(failures) != 0 {
		t.Fatalf("protocol error was treated as a tool result: %v", failures)
	}
	validator.recordResponse([]byte("not-json"))
	if len(failures) != 1 || !strings.Contains(failures[0], "malformed MCP response") {
		t.Fatalf("malformed response diagnostics = %v", failures)
	}
}

func TestResponseValidatorChecksNativeErrorStatusConsistency(t *testing.T) {
	cases := []struct {
		name         string
		tool         string
		isError      bool
		content      map[string]any
		wantFailures int
	}{
		{name: "verify failure missing outer error", tool: "semantic_verify", content: map[string]any{"result": map[string]any{"failure": "failed"}}, wantFailures: 1},
		{name: "verify failure with outer error", tool: "semantic_verify", isError: true, content: map[string]any{"result": map[string]any{"failure": "failed"}}},
		{name: "verify generic error", tool: "semantic_verify", isError: true, content: map[string]any{"error": "missing path"}},
		{name: "structured error missing outer error", tool: "sample", content: map[string]any{"error": "failed"}, wantFailures: 1},
		{name: "empty structured error missing outer error", tool: "sample", content: map[string]any{"error": ""}, wantFailures: 1},
		{name: "batch error has diagnostic delta", tool: "semantic_batch", isError: true, content: map[string]any{"result": map[string]any{"status": "error", "diagnostic_delta": map[string]any{}, "results": []any{map[string]any{"status": "ok", "error": "bad"}}}}, wantFailures: 2},
		{name: "batch error missing outer error", tool: "semantic_batch", content: map[string]any{"result": map[string]any{"status": "error", "results": []any{}}}, wantFailures: 1},
		{name: "batch diagnostics have outer error", tool: "semantic_batch", isError: true, content: map[string]any{"result": map[string]any{"status": "applied_with_diagnostics", "diagnostic_delta": map[string]any{}, "results": []any{}}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var failures []string
			validator := &responseValidator{testName: t.Name(), report: func(message string) { failures = append(failures, message) }}
			validator.validateStatusConsistency(test.tool, test.isError, test.content)
			if len(failures) != test.wantFailures {
				t.Fatalf("status consistency failures = %v, want %d", failures, test.wantFailures)
			}
		})
	}
}

func TestOutputSchemaCompilationPreservesLargeIntegerEnum(t *testing.T) {
	schema, err := CompileOutputSchema(json.RawMessage(`{"type":"integer","enum":[9007199254740993]}`))
	if err != nil {
		t.Fatalf("compile exact integer enum: %v", err)
	}
	if err := schema.Validate(json.Number("9007199254740993")); err != nil {
		t.Fatalf("exact schema integer rejected: %v", err)
	}
	if err := schema.Validate(json.Number("9007199254740992")); err == nil {
		t.Fatal("schema accepted rounded neighboring integer")
	}
}

func TestCompileOutputSchemaReusesExactSchemaContentConcurrently(t *testing.T) {
	raw, marshalErr := json.Marshal(map[string]any{
		"title": t.TempDir(), "type": "object",
		"properties": map[string]any{"value": map[string]any{"const": "cached"}},
		"required":   []string{"value"},
	})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	const workers = 16
	start := make(chan struct{})
	results := make(chan *jsonschema.Schema, workers)
	failures := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			<-start
			schema, err := CompileOutputSchema(bytes.Clone(raw))
			if err != nil {
				failures <- err
				return
			}
			if err := schema.Validate(map[string]any{"value": "cached"}); err != nil {
				failures <- err
				return
			}
			if err := schema.Validate(map[string]any{"value": "different"}); err == nil {
				failures <- fmt.Errorf("cached schema accepted a different value")
				return
			}
			results <- schema
		})
	}
	close(start)
	group.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Errorf("concurrent compilation or validation: %v", err)
	}
	var first *jsonschema.Schema
	for schema := range results {
		if first == nil {
			first = schema
		}
		if schema != first {
			t.Fatal("concurrent compilation did not reuse the same compiled schema")
		}
	}
	if first == nil {
		t.Fatal("no concurrent validation succeeded")
	}
	different, err := CompileOutputSchema(json.RawMessage(`{"type":"object","properties":{"value":{"const":"different"}},"required":["value"]}`))
	if err != nil {
		t.Fatalf("compile different schema: %v", err)
	}
	if err := different.Validate(map[string]any{"value": "different"}); err != nil {
		t.Fatal(err)
	}
	if err := different.Validate(map[string]any{"value": "cached"}); err == nil {
		t.Fatal("different schema reused the cached schema's constraints")
	}
}

func TestResponseValidatorRejectsInvalidHiddenToolResult(t *testing.T) {
	schema, err := CompileOutputSchema(json.RawMessage(`{"type":"object","properties":{"ok":{"const":true}},"required":["ok"],"additionalProperties":false}`))
	if err != nil {
		t.Fatalf("compile fallback schema: %v", err)
	}
	var failures []string
	validator := &responseValidator{
		testName:        t.Name(),
		report:          func(message string) { failures = append(failures, message) },
		advertised:      make(map[string]*jsonschema.Schema),
		advertisedReady: true,
		full:            map[string]*jsonschema.Schema{"hidden": schema},
		fullReady:       true,
	}
	validator.validateToolResult("hidden", json.RawMessage(`{"structuredContent":{"wrong":true}}`))
	if len(failures) != 1 || !strings.Contains(failures[0], "full-catalog fallback") {
		t.Fatalf("hidden tool validation failures = %v, want fallback schema rejection", failures)
	}
}

func TestFullCatalogFallbackPreservesMutationOnlyProfile(t *testing.T) {
	registry := operation.NewRegistry()
	for _, fixture := range []struct {
		name     string
		readOnly bool
	}{
		{name: "semantic_hidden_read", readOnly: true}, {name: "semantic_hidden_write"},
	} {
		if err := operation.Register(registry, operation.Def[struct{}, string]{
			Key: fixture.name, MCPName: fixture.name, ReadOnly: fixture.readOnly,
			Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil },
			Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (string, error){
				backend.LanguageAuto: func(context.Context, operation.CallContext, struct{}) (string, error) { return "ok", nil },
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	validator := &responseValidator{
		profile: "mutations-only", catalogRoot: t.TempDir(),
		options: []mcp.Option{mcp.WithRegistry(registry), mcp.WithToolProfile("base")},
	}
	validator.captureCatalogs()
	if validator.advertisedErr != nil || validator.fullErr != nil {
		t.Fatalf("capture catalogs: advertised=%v full=%v", validator.advertisedErr, validator.fullErr)
	}
	if validator.full["semantic_hidden_write"] == nil {
		t.Fatal("full fallback omitted mutation fixture")
	}
	if validator.full["semantic_hidden_read"] != nil {
		t.Fatal("full fallback ignored the mutation-only profile")
	}
}

func TestLiveCatalogRefreshPreservesFrozenFallback(t *testing.T) {
	oldSchema, err := CompileOutputSchema(json.RawMessage(`{"type":"object","properties":{"old":{"const":true}},"required":["old"]}`))
	if err != nil {
		t.Fatalf("compile old schema: %v", err)
	}
	validator := &responseValidator{
		testName: t.Name(), report: func(message string) { t.Error(message) },
		requests:        map[string]requestInfo{"1": {method: "tools/list"}},
		advertised:      map[string]*jsonschema.Schema{"old": oldSchema},
		full:            map[string]*jsonschema.Schema{"hidden": oldSchema},
		advertisedReady: true, fullReady: true,
	}
	validator.recordResponse([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"current","outputSchema":{"type":"object","properties":{"current":{"const":true}},"required":["current"]}}]}}`))
	if _, exists := validator.advertised["old"]; exists {
		t.Fatal("live tools/list did not replace the advertised catalog")
	}
	if validator.advertised["current"] == nil {
		t.Fatal("live tools/list did not install the current output schema")
	}
	if !validator.fullReady || validator.full["hidden"] != oldSchema {
		t.Fatal("live tools/list replaced the session's frozen full-catalog fallback")
	}
}

func TestCatalogCaptureIsReusedUntilInitialization(t *testing.T) {
	constructions := 0
	var output bytes.Buffer
	server := NewServer(t, "full", t.TempDir(), &output, func(*mcp.Server) { constructions++ })
	call := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_metrics\",\"arguments\":{}}}\n"
	for range 2 {
		if err := server.Serve(t.Context(), strings.NewReader(call)); err != nil {
			t.Fatal(err)
		}
	}
	if constructions != 3 {
		t.Fatalf("repeated serving created %d servers, want actual server and two frozen catalogs", constructions)
	}
	if _, err := server.Initialize(t.Context(), json.RawMessage("{}")); err != nil {
		t.Fatal(err)
	}
	if constructions != 5 {
		t.Fatalf("initialization did not refresh catalogs: got %d server constructions, want 5", constructions)
	}
}
