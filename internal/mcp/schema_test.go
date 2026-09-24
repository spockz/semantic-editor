// This file verifies that MCP batch discovery mirrors the operation registry contract.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"semedit/internal/mcp"
	"semedit/internal/operation"
)

func TestMCPBatchSchemaDerivesFromRegistry(t *testing.T) {
	for _, profile := range []string{"full", "mutations-only"} {
		t.Run(profile, func(t *testing.T) {
			assertBatchSchemaProfile(t, profile)
		})
	}
}

func TestMCPBatchSchemaUsesInjectedRegistry(t *testing.T) {
	registry := operation.NewRegistry()
	params := []operation.ParameterContract{{
		Name:        "file",
		JSONName:    "file",
		Type:        operation.ParamString,
		Description: "target file",
		Required:    true,
	}}
	parse := func(map[string]any) (struct{}, error) { return struct{}{}, nil }
	if err := operation.Register(registry, operation.Def[struct{}, struct{}]{
		Key:       "custom_batch",
		Summary:   "Custom batchable operation.",
		Params:    params,
		MCPName:   "semantic_custom_batch",
		Batchable: true,
		Parse:     parse,
	}); err != nil {
		t.Fatal(err)
	}
	if err := operation.Register(registry, operation.Def[struct{}, struct{}]{
		Key:     "custom_nonbatch",
		Summary: "Custom non-batchable operation.",
		Params:  params,
		MCPName: "semantic_custom_nonbatch",
		Parse:   parse,
	}); err != nil {
		t.Fatal(err)
	}

	tools := listToolsWithRegistry(t, "full", registry)
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	if _, ok := byName["semantic_insert_function"]; ok {
		t.Fatal("injected registry discovery inherited a default operation")
	}
	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch for injected registry")
	}
	customBatch, ok := byName["semantic_custom_batch"]
	if !ok {
		t.Fatal("tools/list omitted custom batchable operation")
	}
	assertStandardOutputSchema(t, customBatch["outputSchema"], "string")
	edits := batch["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)
	branches := edits["items"].(map[string]any)["oneOf"].([]any)
	if len(branches) != 1 {
		t.Fatalf("injected registry batch alternatives = %d, want 1", len(branches))
	}
	branch := branches[0].(map[string]any)
	properties := branch["properties"].(map[string]any)
	if got := properties["tool"].(map[string]any)["const"]; got != "semantic_custom_batch" {
		t.Fatalf("injected batch tool const = %#v, want semantic_custom_batch", got)
	}
	if !reflect.DeepEqual(properties["params"], customBatch["inputSchema"]) {
		t.Fatal("injected batch params schema does not match the custom operation schema")
	}
	if got := properties["params"].(map[string]any)["required"]; !reflect.DeepEqual(got, []any{"file"}) {
		t.Fatalf("injected batch params required = %#v, want file", got)
	}
	if _, ok := byName["semantic_custom_nonbatch"]; !ok {
		t.Fatal("tools/list omitted custom non-batchable operation")
	}
}

func TestMCPToolsAdvertiseStructuredOutputSchemas(t *testing.T) {
	for _, profile := range []string{"full", "mutations-only"} {
		t.Run(profile, func(t *testing.T) {
			for _, tool := range listTools(t, profile) {
				name, ok := tool["name"].(string)
				if !ok {
					t.Fatalf("tool has invalid name: %#v", tool["name"])
				}
				if tool["outputSchema"] == nil {
					t.Fatalf("%s omitted outputSchema", name)
				}
				switch name {
				case "semantic_batch":
					assertBatchOutputSchema(t, tool["outputSchema"])
				default:
					assertStandardOutputSchema(t, tool["outputSchema"], "string")
				}
			}
		})
	}
}

func TestMCPLiveReloadAdvertisesStructuredOutputSchema(t *testing.T) {
	tools := listToolsWithLiveReload(t)
	for _, tool := range tools {
		if tool["name"] != "semantic_reload" {
			continue
		}
		output, ok := tool["outputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("semantic_reload outputSchema = %#v, want object", tool["outputSchema"])
		}
		if got := output["required"]; !reflect.DeepEqual(got, []any{"result"}) {
			t.Fatalf("semantic_reload outputSchema required = %#v", got)
		}
		result := output["properties"].(map[string]any)["result"].(map[string]any)
		status := result["properties"].(map[string]any)["status"].(map[string]any)
		if status["const"] != "reloading" {
			t.Fatalf("semantic_reload status schema = %#v, want reloading", status)
		}
		return
	}
	t.Fatal("live-reload tools/list omitted semantic_reload")
}

func assertStandardOutputSchema(t *testing.T, raw any, resultType string) {
	t.Helper()
	schema, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("outputSchema = %#v, want object", raw)
	}
	if schema["type"] != "object" {
		t.Fatalf("outputSchema type = %#v, want object", schema["type"])
	}
	if got := schema["required"]; !reflect.DeepEqual(got, []any{"result", "metrics"}) {
		t.Fatalf("outputSchema required = %#v, want result and metrics", got)
	}
	properties := schema["properties"].(map[string]any)
	if got := properties["result"].(map[string]any)["type"]; got != resultType {
		t.Fatalf("outputSchema result type = %#v, want %s", got, resultType)
	}
	if properties["metrics"].(map[string]any)["type"] != "object" {
		t.Fatalf("outputSchema metrics = %#v, want object", properties["metrics"])
	}
}

func assertBatchOutputSchema(t *testing.T, raw any) {
	t.Helper()
	schema := raw.(map[string]any)
	assertStandardOutputSchema(t, schema, "object")
	result := schema["properties"].(map[string]any)["result"].(map[string]any)
	if got := result["required"]; !reflect.DeepEqual(got, []any{"status", "results", "final_diff"}) {
		t.Fatalf("batch result required = %#v, want status, results, and final_diff", got)
	}
	properties := result["properties"].(map[string]any)
	items := properties["results"].(map[string]any)["items"].(map[string]any)
	if got := items["required"]; !reflect.DeepEqual(got, []any{"tool", "status"}) {
		t.Fatalf("batch result item required = %#v", got)
	}
	if diff, ok := properties["final_diff"].(map[string]any); !ok || diff["type"] != "string" {
		t.Fatalf("batch final_diff schema = %#v", properties["final_diff"])
	}
	delta, ok := properties["diagnostic_delta"].(map[string]any)
	if !ok {
		t.Fatal("batch result omitted diagnostic_delta schema")
	}
	if got := delta["required"]; !reflect.DeepEqual(got, []any{"before", "after", "net_delta", "introduced", "resolved"}) {
		t.Fatalf("diagnostic_delta required = %#v", got)
	}
}

func listTools(t *testing.T, profile string) []map[string]any {
	t.Helper()
	return listToolsWithRegistry(t, profile, nil)
}

func listToolsWithRegistry(t *testing.T, profile string, registry *operation.Registry) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	var opts []mcp.Option
	if registry != nil {
		opts = append(opts, mcp.WithRegistry(registry))
	}
	if err := mcp.NewServer(profile, ".", &out, opts...).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	var response struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("decode tools/list response: %v", err)
	}
	return response.Result.Tools
}

func listToolsWithLiveReload(t *testing.T) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := mcp.NewServer("full", ".", &out, mcp.WithLiveReload(true)).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	var response struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("decode tools/list response: %v", err)
	}
	return response.Result.Tools
}

func assertBatchSchemaProfile(t *testing.T, profile string) {
	tools := listTools(t, profile)
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}

	batch, ok := byName["semantic_batch"]
	if !ok {
		t.Fatal("tools/list omitted semantic_batch")
	}
	assertBatchOutputSchema(t, batch["outputSchema"])
	description := batch["description"].(string)
	if !strings.Contains(description, "diagnostic_delta") || !strings.Contains(description, "final_diff") {
		t.Fatalf("batch description = %q, want final_diff and diagnostic_delta guidance", description)
	}

	inputSchema := batch["inputSchema"].(map[string]any)
	edits := inputSchema["properties"].(map[string]any)["edits"].(map[string]any)
	if got := edits["minItems"]; got != float64(1) {
		t.Fatalf("edits.minItems = %#v, want 1", got)
	}
	branches := edits["items"].(map[string]any)["oneOf"].([]any)

	expected := make(map[string]operation.Entry)
	expectedNames := make([]string, 0)
	for _, entry := range operation.DefaultRegistry().All() {
		if entry.MCPName == "" || !entry.Batchable || (profile == "mutations-only" && entry.ReadOnly) {
			continue
		}
		expected[entry.MCPName] = entry
		expectedNames = append(expectedNames, entry.MCPName)
	}
	if len(branches) != len(expected) {
		t.Fatalf("batch alternatives = %d, want %d", len(branches), len(expected))
	}

	seen := make(map[string]bool, len(branches))
	for index, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		name := properties["tool"].(map[string]any)["const"].(string)
		if name != expectedNames[index] {
			t.Fatalf("batch alternative %d = %q, want registry order %q", index, name, expectedNames[index])
		}
		entry, ok := expected[name]
		if !ok {
			t.Fatalf("batch schema advertised unexpected tool %q", name)
		}
		seen[name] = true
		if got := branch["required"].([]any); !reflect.DeepEqual(got, []any{"tool", "params"}) {
			t.Fatalf("%s required = %#v, want tool and params", name, got)
		}
		if got := branch["additionalProperties"]; got != false {
			t.Fatalf("%s additionalProperties = %#v, want false", name, got)
		}
		params := properties["params"].(map[string]any)
		direct := byName[name]["inputSchema"]
		if !reflect.DeepEqual(params, direct) {
			t.Fatalf("%s batch params schema drifted from direct schema:\nparams=%#v\ndirect=%#v", name, params, direct)
		}
		expectedRequired := make([]any, 0)
		for _, param := range entry.Params {
			if param.Required {
				expectedRequired = append(expectedRequired, param.JSONName)
			}
		}
		if got := params["required"]; len(expectedRequired) == 0 {
			if got != nil {
				t.Fatalf("%s batch params required = %#v, want omitted", name, got)
			}
		} else if !reflect.DeepEqual(got, expectedRequired) {
			t.Fatalf("%s batch params required = %#v, want %#v", name, got, expectedRequired)
		}
	}
	for name := range expected {
		if !seen[name] {
			t.Errorf("batch schema omitted registry batchable tool %q", name)
		}
	}
	for _, entry := range operation.DefaultRegistry().All() {
		if entry.MCPName != "" && !entry.Batchable && (profile != "mutations-only" || !entry.ReadOnly) && seen[entry.MCPName] {
			t.Errorf("batch schema advertised non-batchable tool %q", entry.MCPName)
		}
	}
}
