// Package mcp_test checks that success and failure structured results are mutually constrained by advertised JSON Schemas.

package mcp_test

import (
	"bytes"
	"encoding/json"
	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/mcptest"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestAdvertisedOutputSchemasConstrainFailureShapes(t *testing.T) {
	schemas := advertisedOutputSchemas(t)
	assertRenameOutputSchema(t, schemas)
	assertVerifyOutputSchema(t, schemas)
	assertReloadOutputSchema(t, schemas)
	assertBatchOutputSchemaFailures(t, schemas)
}

func TestBatchUnknownAndNonBatchableChildrenAreSchemaValidErrors(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		"go.mod":  "module example.com/batchoutputcontracts\n\ngo 1.27.1\n",
		"main.go": "package sample\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, path), []byte(contents)); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	requests := []string{
		`{"jsonrpc":"2.0","id":"unknown","method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_not_registered","params":{}}]}}}`,
		`{"jsonrpc":"2.0","id":"nonbatchable","method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_snapshot","params":{}}]}}}`,
	}
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, mcp.WithToolProfile("full"), mcp.WithGoBaseDir(t.TempDir()))
	if err := server.Serve(t.Context(), strings.NewReader(strings.Join(requests, "\n")+"\n")); err != nil {
		t.Fatalf("call batch error cases: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	responses := make([]struct {
		ID     string `json:"id"`
		Result struct {
			IsError           bool           `json:"isError"`
			StructuredContent map[string]any `json:"structuredContent"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}, 0, len(lines))
	for _, line := range lines {
		var response struct {
			ID     string `json:"id"`
			Result struct {
				IsError           bool           `json:"isError"`
				StructuredContent map[string]any `json:"structuredContent"`
			} `json:"result"`
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("decode batch response: %v", err)
		}
		responses = append(responses, response)
	}
	if len(responses) != len(requests) {
		t.Fatalf("received %d batch responses, want %d: %s", len(responses), len(requests), output.String())
	}
	for _, response := range responses {
		if len(response.Error) != 0 || !response.Result.IsError {
			t.Errorf("batch child %s was not reported as an MCP tool error: %#v", response.ID, response)
			continue
		}
		batchResult, ok := response.Result.StructuredContent["result"].(map[string]any)
		if !ok || batchResult["status"] != "error" {
			t.Errorf("batch child %s omitted error status: %#v", response.ID, response.Result.StructuredContent)
		}
		rows, ok := batchResult["results"].([]any)
		if !ok || len(rows) != 1 {
			t.Errorf("batch child %s returned malformed result rows: %#v", response.ID, batchResult["results"])
			continue
		}
		row, ok := rows[0].(map[string]any)
		if !ok || row["status"] != "error" || row["error"] == "" {
			t.Errorf("batch child %s did not include an error row: %#v", response.ID, rows[0])
		}
	}
}

func advertisedOutputSchemas(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	root := t.TempDir()
	for path, contents := range map[string]string{
		"go.mod":  "module example.com/outputcontracts\n\ngo 1.27.1\n",
		"main.go": "package sample\n\nfunc Existing() int { return 1 }\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, path), []byte(contents)); err != nil {
			t.Fatalf("write schema fixture %s: %v", path, err)
		}
	}
	options := []mcp.Option{mcp.WithToolProfile("full"), mcp.WithGoBaseDir(t.TempDir()), mcp.WithLiveReload(true)}
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, options...)
	request := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}\n"
	if err := server.Serve(t.Context(), strings.NewReader(request)); err != nil {
		t.Fatalf("list advertised schemas: %v", err)
	}
	var response struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result struct {
			Tools []struct {
				Name         string          `json:"name"`
				OutputSchema json.RawMessage `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode advertised schemas: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("tools/list failed: %d %s", response.Error.Code, response.Error.Message)
	}
	schemas := make(map[string]json.RawMessage, len(response.Result.Tools))
	for _, tool := range response.Result.Tools {
		schemas[tool.Name] = bytes.Clone(tool.OutputSchema)
	}
	return schemas
}

func compileAdvertisedOutputSchema(t *testing.T, raw json.RawMessage) *jsonschema.Schema {
	t.Helper()
	schema, err := mcptest.CompileOutputSchema(raw)
	if err != nil {
		t.Fatalf("compile advertised output schema: %v", err)
	}
	return schema
}

func assertOutputSchemaValue(t *testing.T, schema *jsonschema.Schema, label string, value any, valid bool) {
	t.Helper()
	err := schema.Validate(value)
	if valid && err != nil {
		t.Errorf("%s rejected valid output: %v", label, err)
	}
	if !valid && err == nil {
		t.Errorf("%s accepted invalid output: %#v", label, value)
	}
}

func compileToolOutputSchema(t *testing.T, schemas map[string]json.RawMessage, name string) *jsonschema.Schema {
	t.Helper()
	raw, ok := schemas[name]
	if !ok {
		t.Fatalf("full tools/list omitted %s", name)
	}
	return compileAdvertisedOutputSchema(t, raw)
}

func assertRenameOutputSchema(t *testing.T, schemas map[string]json.RawMessage) {
	schema := compileToolOutputSchema(t, schemas, "semantic_rename")
	diagnostics := map[string]any{"net_delta": 0, "introduced": []any{}, "resolved": []any{}}
	success := map[string]any{"status": "renamed", "application_state": "persisted", "diagnostics": diagnostics, "active": []any{}}
	known := map[string]any{
		"status": "applied_with_diagnostics", "application_state": "persisted",
		"origin":      map[string]any{"file": "main.go", "symbol": "Existing"},
		"diagnostics": diagnostics, "active": []any{},
	}
	assertOutputSchemaValue(t, schema, "rename success", map[string]any{"result": success}, true)
	assertOutputSchemaValue(t, schema, "rename generic error", map[string]any{"error": "symbol not found", "application_state": "unknown"}, true)
	suggestion := map[string]any{"arguments": map[string]any{"file": "main.go", "symbol": "Existing"}, "file": "main.go", "line": 3, "column": 6, "kind": "function", "reason": "same package", "signature": "func Existing()"}
	assertOutputSchemaValue(t, schema, "rename error with target suggestions", map[string]any{"error": "symbol is ambiguous", "application_state": "unknown", "suggestions": []any{suggestion}}, true)
	assertOutputSchemaValue(t, schema, "rename error with malformed target suggestion", map[string]any{"error": "symbol is ambiguous", "application_state": "unknown", "suggestions": []any{map[string]any{"file": "main.go"}}}, false)
	assertOutputSchemaValue(t, schema, "rename error with too many target suggestions", map[string]any{"error": "symbol is ambiguous", "application_state": "unknown", "suggestions": []any{suggestion, suggestion, suggestion, suggestion}}, false)
	assertOutputSchemaValue(t, schema, "rename known projected error", map[string]any{
		"error": "rename applied with diagnostics", "application_state": "persisted", "result": known,
	}, true)
	assertOutputSchemaValue(t, schema, "malformed known typed result", map[string]any{
		"error": "rename applied with diagnostics", "application_state": "persisted",
		"result": map[string]any{"status": "applied_with_diagnostics"},
	}, false)
	assertOutputSchemaValue(t, schema, "generic error with success result", map[string]any{
		"error": "symbol not found", "application_state": "unknown", "result": success,
	}, false)
	assertOutputSchemaValue(t, schema, "success result plus top-level error", map[string]any{
		"error": "tool failed", "application_state": "unknown", "result": success,
	}, false)
}

func assertVerifyOutputSchema(t *testing.T, schemas map[string]json.RawMessage) {
	schema := compileToolOutputSchema(t, schemas, "semantic_verify")
	report := map[string]any{
		"Root": "/workspace", "ConfigPath": "", "Phase": "complete",
		"Scope": map[string]any{"Paths": nil}, "Results": nil, "Completed": true, "Partial": false,
	}
	result := map[string]any{
		"diagnostics": []any{}, "formatted": false, "normalize": report,
		"checks": report, "scope": map[string]any{"Paths": nil}, "failure": "verification failed",
	}
	assertOutputSchemaValue(t, schema, "native verification failure result", map[string]any{"result": result}, true)
	assertOutputSchemaValue(t, schema, "native verification failure missing required report", map[string]any{
		"result": map[string]any{"failure": "verification failed"},
	}, false)
	assertOutputSchemaValue(t, schema, "native result disguised as generic error", map[string]any{
		"error": "verification failed", "application_state": "unknown", "result": result,
	}, false)
}

func assertReloadOutputSchema(t *testing.T, schemas map[string]json.RawMessage) {
	schema := compileToolOutputSchema(t, schemas, "semantic_reload")
	assertOutputSchemaValue(t, schema, "reload success", map[string]any{
		"result": map[string]any{"status": "reloading"},
	}, true)
	assertOutputSchemaValue(t, schema, "reload generic error", map[string]any{
		"error": "reload is unavailable", "application_state": "unknown",
	}, true)
	assertOutputSchemaValue(t, schema, "reload success and error simultaneously", map[string]any{
		"error": "reload failed", "application_state": "unknown",
		"result": map[string]any{"status": "reloading"},
	}, false)
}

func assertBatchOutputSchemaFailures(t *testing.T, schemas map[string]json.RawMessage) {
	schema := compileToolOutputSchema(t, schemas, "semantic_batch")
	diagnostics := map[string]any{"net_delta": 0, "introduced": []any{}, "resolved": []any{}}
	removeSuccess := map[string]any{
		"tool": "semantic_remove_construct", "status": "ok",
		"result": map[string]any{
			"status": "removed", "safe_delete": false, "checks_run": []any{}, "diff": "",
			"diagnostic_status": "unchanged", "diagnostic_delta": diagnostics,
		},
	}
	genericError := map[string]any{
		"tool": "semantic_insert_function", "status": "error",
		"error": "source file not found", "application_state": "unknown",
	}
	knownError := map[string]any{
		"tool": "semantic_remove_construct", "status": "error",
		"error": "safe deletion refused", "application_state": "persisted",
		"result": map[string]any{
			"status": "refused", "safe_delete": true, "checks_run": []any{}, "refusal": "exported declaration",
		},
	}
	genericError["suggestions"] = []any{map[string]any{"arguments": map[string]any{"file": "main.go", "symbol": "Existing"}, "file": "main.go", "line": 3, "column": 6, "kind": "function", "reason": "same package"}}
	assertOutputSchemaValue(t, schema, "successful batch with typed result", batchOutput("ok", []any{removeSuccess}, nil), true)
	assertOutputSchemaValue(t, schema, "failed batch with generic error", batchOutput("error", []any{genericError}, nil), true)
	assertOutputSchemaValue(t, schema, "failed batch with typed error", batchOutput("error", []any{knownError}, nil), true)
	assertOutputSchemaValue(t, schema, "success batch containing error child", batchOutput("ok", []any{genericError}, nil), false)
	assertOutputSchemaValue(t, schema, "error batch containing only success children", batchOutput("error", []any{removeSuccess}, nil), false)
	assertOutputSchemaValue(t, schema, "failed batch retaining diagnostic delta", batchOutput("error", []any{genericError}, diagnostics), false)
	assertOutputSchemaValue(t, schema, "applied batch retaining diagnostic delta", batchOutput("applied_with_diagnostics", []any{removeSuccess}, diagnostics), true)
	badKnownError := map[string]any{
		"tool": "semantic_remove_construct", "status": "error",
		"error": "safe deletion refused", "application_state": "persisted",
		"result": map[string]any{"status": "refused"},
	}
	assertOutputSchemaValue(t, schema, "known batch error with malformed typed result", batchOutput("error", []any{badKnownError}, nil), false)
}

func batchOutput(status string, rows []any, delta any) map[string]any {
	result := map[string]any{
		"status": status, "results": rows, "final_diff": "unchanged", "workspace_root": "/workspace",
	}
	if delta != nil {
		result["diagnostic_delta"] = delta
	}
	return map[string]any{"result": result}
}
