// Package mcptest validates the test response validator against advertised schema fixtures.
package mcptest

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestResponseValidatorChecksErrorResultsAndIntegerPrecision(t *testing.T) {
	schema, err := compileSchema(json.RawMessage(`{"type":"object","properties":{"result":{"type":"integer"}},"required":["result"],"additionalProperties":false}`))
	if err != nil {
		t.Fatalf("compile fixture schema: %v", err)
	}
	var failures []string
	validator := &responseValidator{
		testName:   t.Name(),
		report:     func(message string) { failures = append(failures, message) },
		advertised: map[string]*jsonschema.Schema{"sample": schema},
		full:       make(map[string]*jsonschema.Schema),
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
		testName:   t.Name(),
		report:     func(message string) { failures = append(failures, message) },
		dst:        &output,
		requests:   make(map[string]requestInfo),
		advertised: map[string]*jsonschema.Schema{"sample": schema},
		full:       make(map[string]*jsonschema.Schema),
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
