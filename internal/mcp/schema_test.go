// This file verifies that MCP batch discovery mirrors the operation registry contract.
package mcp_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"semedit/internal/backend"
	gobackend "semedit/internal/backend/golang"
	"semedit/internal/mcp"
	"semedit/internal/operation"
	"semedit/internal/sourcebuf"
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

func TestMCPContinuationDescriptionsNameSourcesWithinBudget(t *testing.T) {
	var tool map[string]any
	for _, candidate := range listTools(t, "full") {
		if candidate["name"] == "semantic_insert_function" {
			tool = candidate
			break
		}
	}
	if tool == nil {
		t.Fatal("tools/list omitted semantic_insert_function")
	}
	properties := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
	description := properties["file"].(map[string]any)["description"].(string)
	if len(description) > 500 {
		t.Fatalf("file parameter description is %d bytes, exceeds 500-byte budget: %s", len(description), description)
	}
	for _, source := range []string{
		"semantic_lookup", "semantic_rename", "semantic_replace_body", "semantic_replace_construct",
		"semantic_insert_case", "semantic_insert_function", "semantic_insert_type", "semantic_insert_decl",
		"semantic_insert_declaration", "semantic_insert_structure", "semantic_replace_decl",
		"semantic_extract_function", "semantic_extract_variable", "semantic_inline_variable", "semantic_organize_imports", "semantic_scaffold_file",
	} {
		if !strings.Contains(description, source) {
			t.Errorf("file description %q omitted relevant source tool %s", description, source)
		}
	}
	t.Logf("semantic_insert_function.file continuation description: %d bytes", len(description))
	for _, candidate := range listTools(t, "full") {
		if candidate["name"] != "semantic_replace_body" {
			continue
		}
		props := candidate["inputSchema"].(map[string]any)["properties"].(map[string]any)
		symbolDescription := props["symbol"].(map[string]any)["description"].(string)
		if !strings.Contains(symbolDescription, "semantic_insert_function") {
			t.Fatalf("replace_body.symbol description omitted insert_function continuation: %s", symbolDescription)
		}
		return
	}
	t.Fatal("tools/list omitted semantic_replace_body")
}

func TestInlineVariableToolSchemaAndContinuations(t *testing.T) {
	var tool map[string]any
	for _, candidate := range listTools(t, "full") {
		if candidate["name"] == "semantic_inline_variable" {
			tool = candidate
			break
		}
	}
	if tool == nil {
		t.Fatal("tools/list omitted semantic_inline_variable")
	}
	input := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
	for key, role := range map[string]string{"symbol": "Target:", "file": "Selector:", "in_function": "Selector:"} {
		field, ok := input[key].(map[string]any)
		if !ok {
			t.Fatalf("semantic_inline_variable input omitted %s", key)
		}
		if !strings.Contains(field["description"].(string), role) {
			t.Errorf("semantic_inline_variable.%s description omitted role %q", key, role)
		}
	}
	output := tool["outputSchema"].(map[string]any)["properties"].(map[string]any)
	for _, key := range []string{"file", "in_function"} {
		if _, ok := output[key]; !ok {
			t.Errorf("semantic_inline_variable output omitted continuation field %s", key)
		}
	}
}

func TestSourceFieldsResolveToAdvertisedOutputSchema(t *testing.T) {
	registry := operation.DefaultRegistry()
	if err := mcp.ValidateSourceFields(registry.All()); err != nil {
		t.Fatal(err)
	}
	broken := append(registry.All(), operation.Entry{
		MCPName: "semantic_broken_source",
		Params:  []operation.ParameterContract{{Name: "symbol", JSONName: "symbol", SourceFields: []string{"semantic_lookup.offset"}}},
	})
	if err := mcp.ValidateSourceFields(broken); err == nil || !strings.Contains(err.Error(), "semantic_lookup.offset") {
		t.Fatalf("invalid SourceFields reference error = %v", err)
	}
}

func assertContractOutputSchema(t *testing.T, name string, schema any) {
	t.Helper()
	assertStandardOutputSchema(t, schema, "object")
	output := schema.(map[string]any)["properties"].(map[string]any)
	result := output["result"].(map[string]any)
	properties := result["properties"].(map[string]any)
	for _, field := range []string{"receipt", "signatures"} {
		if _, ok := properties[field]; !ok {
			t.Errorf("%s result schema omitted %q", name, field)
		}
	}
	fields := []string{"contract", "contract_file"}
	if name == "semantic_implement_contract" {
		fields = []string{"type_target", "generated_methods"}
	}
	for _, field := range fields {
		if _, ok := properties[field]; !ok {
			t.Errorf("%s result schema omitted %q", name, field)
		}
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
				case "semantic_remove_construct":
					assertStandardOutputSchema(t, tool["outputSchema"], "object")
					output := tool["outputSchema"].(map[string]any)["properties"].(map[string]any)
					for _, field := range []string{"file"} {
						if _, ok := output[field]; !ok {
							t.Errorf("remove outputSchema omitted continuation field %q", field)
						}
					}
					result := output["result"].(map[string]any)
					for _, field := range []string{"status", "safe_delete", "checks_run"} {
						if _, ok := result["properties"].(map[string]any)[field]; !ok {
							t.Errorf("remove result schema omitted %q", field)
						}
					}
				case "semantic_extract_interface", "semantic_implement_contract":
					assertContractOutputSchema(t, name, tool["outputSchema"])
				case "semantic_lookup", "semantic_rename", "semantic_replace_body", "semantic_replace_construct", "semantic_insert_case", "semantic_extract_function", "semantic_extract_variable", "semantic_inline_variable":
					assertStandardOutputSchema(t, tool["outputSchema"], "string")
					properties := tool["outputSchema"].(map[string]any)["properties"].(map[string]any)
					for _, field := range map[string][]string{
						"semantic_lookup":            {"symbol", "file", "kind", "line", "column"},
						"semantic_rename":            {"symbol", "file"},
						"semantic_replace_body":      {"symbol", "file"},
						"semantic_extract_function":  {"symbol", "file", "in_function"},
						"semantic_extract_variable":  {"symbol", "file", "in_function"},
						"semantic_replace_construct": {"file", "in_function"},
						"semantic_inline_variable":   {"file", "in_function"},
						"semantic_insert_case":       {"file", "in_function", "discriminator"},
					}[name] {
						if _, ok := properties[field]; !ok {
							t.Errorf("%s outputSchema omitted continuation field %q", name, field)
						}
					}
				case "semantic_batch":
					assertBatchOutputSchema(t, tool["outputSchema"])
				case "semantic_metrics":
					assertStandardOutputSchema(t, tool["outputSchema"], "object")
					result := tool["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
					records := result["properties"].(map[string]any)["records"].(map[string]any)
					item := records["items"].(map[string]any)
					if item["properties"].(map[string]any)["metrics"] == nil {
						t.Fatal("semantic_metrics record schema omitted metrics")
					}
					if !reflect.DeepEqual(result["required"], []any{"records"}) {
						t.Fatalf("semantic_metrics result required = %#v", result["required"])
					}
				case "report_feedback":
					assertStandardOutputSchema(t, tool["outputSchema"], "object")
					output := tool["outputSchema"].(map[string]any)
					result := output["properties"].(map[string]any)["result"].(map[string]any)
					if got := result["required"]; !reflect.DeepEqual(got, []any{"status", "date", "report", "markdown"}) {
						t.Fatalf("feedback result required = %#v", got)
					}
					status := result["properties"].(map[string]any)["status"].(map[string]any)
					if status["const"] != "draft_only" {
						t.Fatalf("feedback status schema = %#v, want draft_only", status)
					}

				case "semantic_inspect_symbol":
					assertStandardOutputSchema(t, tool["outputSchema"], "object")
					assertInspectOutputSchema(t, tool["outputSchema"])
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
	if got := schema["required"]; !reflect.DeepEqual(got, []any{"result"}) {
		t.Fatalf("outputSchema required = %#v, want result only", got)
	}
	properties := schema["properties"].(map[string]any)
	if got := properties["result"].(map[string]any)["type"]; got != resultType {
		t.Fatalf("outputSchema result type = %#v, want %s", got, resultType)
	}
	if _, ok := properties["metrics"]; ok {
		t.Fatalf("standard outputSchema must not include per-call metrics: %#v", properties["metrics"])
	}
	if _, ok := properties["session_metrics"]; ok {
		t.Fatalf("standard outputSchema must not include session metrics: %#v", properties["session_metrics"])
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
	branches, ok := items["oneOf"].([]any)
	if !ok || len(branches) == 0 {
		t.Fatalf("batch result item schema = %#v, want registered tool branches", items)
	}
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		if got := branch["required"]; !reflect.DeepEqual(got, []any{"tool", "status"}) {
			t.Fatalf("batch result branch required = %#v", got)
		}
		if branch["additionalProperties"] != false {
			t.Fatalf("batch result branch is not closed: %#v", branch)
		}
	}
	if diff, ok := properties["final_diff"].(map[string]any); !ok || diff["type"] != "string" {
		t.Fatalf("batch final_diff schema = %#v", properties["final_diff"])
	}
	delta, ok := properties["diagnostic_delta"].(map[string]any)
	if !ok {
		t.Fatal("batch result omitted diagnostic_delta schema")
	}
	if got := delta["required"]; !reflect.DeepEqual(got, []any{"net_delta", "introduced", "resolved"}) {
		t.Fatalf("diagnostic_delta required = %#v", got)
	}
}

func listTools(t *testing.T, profile string) []map[string]any {
	t.Helper()
	return listToolsAtWithRegistry(t, profile, schemaWorkspace(t), nil)
}

func listToolsWithRegistry(t *testing.T, profile string, registry *operation.Registry) []map[string]any {
	t.Helper()
	return listToolsAtWithRegistry(t, profile, schemaWorkspace(t), registry)
}

func listToolsAtWithRegistry(t *testing.T, profile, root string, registry *operation.Registry) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	var opts []mcp.Option
	if registry != nil {
		opts = append(opts, mcp.WithRegistry(registry))
	}
	if err := mcp.NewServer(profile, root, &out, opts...).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
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

func schemaWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, source := range map[string]string{"main.go": "package sample\n", "Main.java": "class Main {}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func listToolsWithLiveReload(t *testing.T) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := mcp.NewServer("full", schemaWorkspace(t), &out, mcp.WithLiveReload(true)).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
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

func TestConstructReplacementSchemasAreExposedAndBatchable(t *testing.T) {
	tools := listTools(t, "full")
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}

	for _, test := range []struct {
		name             string
		required         []string
		stringProperties []string
	}{
		{
			name:             "semantic_replace_construct",
			required:         []string{"file", "in_function", "kind", "source"},
			stringProperties: []string{"file", "in_function", "kind", "discriminator", "construct_path", "source"},
		},
		{
			name:             "semantic_replace_decl",
			required:         []string{"file", "symbol", "source"},
			stringProperties: []string{"file", "symbol", "source"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tool, ok := byName[test.name]
			if !ok {
				t.Fatalf("tools/list omitted %s", test.name)
			}
			input := tool["inputSchema"].(map[string]any)
			properties := input["properties"].(map[string]any)
			for _, name := range test.stringProperties {
				property, ok := properties[name].(map[string]any)
				if !ok || property["type"] != "string" {
					t.Errorf("%s property schema = %#v, want string", name, properties[name])
				}
			}
			if got := input["required"]; !reflect.DeepEqual(got, toAnySlice(test.required)) {
				t.Errorf("required = %#v, want %v", got, test.required)
			}
			description, _ := tool["description"].(string)
			if !strings.Contains(description, "Use this tool instead of") ||
				(!strings.Contains(description, "replace_body") && !strings.Contains(description, "replace_file_content")) {
				t.Errorf("description does not advertise the semantic-edit trigger: %q", description)
			}
		})
	}
	construct := byName["semantic_replace_construct"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["kind"].(map[string]any)
	if got, want := construct["enum"], toAnySlice([]string{"loop", "if", "else", "case", "select", "defer"}); !reflect.DeepEqual(got, want) {
		t.Errorf("Go construct enum = %#v, want %#v", got, want)
	}

	insertStructure := byName["semantic_insert_structure"]
	if insertStructure == nil {
		t.Fatal("tools/list omitted semantic_insert_structure")
	}
	overwrite, ok := insertStructure["inputSchema"].(map[string]any)["properties"].(map[string]any)["overwrite"].(map[string]any)
	if !ok || overwrite["type"] != "boolean" {
		t.Fatalf("semantic_insert_structure overwrite schema = %#v, want optional boolean", overwrite)
	}
	if defaultValue, hasDefault := overwrite["default"]; hasDefault && defaultValue != false {
		t.Fatalf("semantic_insert_structure overwrite default = %#v, want false when specified", defaultValue)
	}

	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	edits := batch["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)
	branches := edits["items"].(map[string]any)["oneOf"].([]any)
	batchTools := make(map[string]map[string]any, len(branches))
	for _, raw := range branches {
		branch := raw.(map[string]any)
		properties := branch["properties"].(map[string]any)
		toolName, _ := properties["tool"].(map[string]any)["const"].(string)
		batchTools[toolName] = properties["params"].(map[string]any)
	}
	for _, name := range []string{"semantic_replace_construct", "semantic_replace_decl", "semantic_insert_structure"} {
		params, ok := batchTools[name]
		if !ok {
			t.Errorf("semantic_batch schema omitted %s", name)
			continue
		}
		if !reflect.DeepEqual(params, byName[name]["inputSchema"]) {
			t.Errorf("semantic_batch params for %s differ from tools/list schema", name)
		}
	}
}

func TestMCPEnabledLanguagesFilterMixedAndSourceLessWorkspaces(t *testing.T) {
	root := t.TempDir()
	write := func(name, source string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("main.go", "package sample\nfunc F() {}\n")
	write("rust/lib.rs", "pub fn f() {}\n")
	tools := listToolsAt(t, root, mcp.WithEnabledLanguages("go"))
	byName := make(map[string]bool, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = true
	}
	if !byName["semantic_replace_construct"] {
		t.Fatal("Go construct tool missing from enabled mixed workspace")
	}
	if byName["semantic_maven_compile"] {
		t.Fatal("Java-only Maven tool exposed when only Go is enabled")
	}

	empty := t.TempDir()
	for _, tool := range listToolsAt(t, empty) {
		if tool["name"] == "semantic_replace_construct" {
			t.Fatal("source-less workspace inferred a Go construct backend")
		}
	}
	if err := mcp.NewServer("full", empty, &bytes.Buffer{}, mcp.WithEnabledLanguages("go,unknown")).Serve(context.Background(), strings.NewReader("")); err == nil {
		t.Fatal("unknown enabled language was accepted")
	}

	haskell := t.TempDir()
	if err := os.WriteFile(filepath.Join(haskell, "Main.hs"), []byte("module Main where\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if hasTool(listToolsAt(t, haskell), "semantic_lookup") {
		t.Fatal("implicit Haskell source advertised standalone lookup")
	}
	if !hasTool(listToolsAt(t, haskell, mcp.WithEnabledLanguages("haskell")), "semantic_lookup") {
		t.Fatal("explicitly enabled Haskell omitted standalone lookup")
	}
}

type customConstructBackend struct {
	backend.Backend

	kinds []backend.ConstructKind
}

func (b customConstructBackend) SupportedConstructs() []backend.ConstructKind {
	return append([]backend.ConstructKind(nil), b.kinds...)
}

func TestConstructSchemaUsesRegisteredBackendCapabilities(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	custom := customConstructBackend{
		Backend: gobackend.NewGoBackend(),
		kinds:   []backend.ConstructKind{"custom_kind", backend.ConstructLoop},
	}
	registry, err := backend.NewRegistry(custom)
	if err != nil {
		t.Fatal(err)
	}
	tools := listToolsAt(t, root, mcp.WithService(backend.NewService(registry)))
	for _, tool := range tools {
		if tool["name"] != "semantic_replace_construct" {
			continue
		}
		properties := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
		kind := properties["kind"].(map[string]any)
		if got, want := kind["enum"], toAnySlice([]string{"custom_kind", "loop"}); !reflect.DeepEqual(got, want) {
			t.Fatalf("construct kind enum = %#v, want registered backend values %#v", got, want)
		}
		return
	}
	t.Fatal("tools/list omitted semantic_replace_construct")
}

func hasTool(tools []map[string]any, name string) bool {
	for _, tool := range tools {
		if tool["name"] == name {
			return true
		}
	}
	return false
}

func listToolsAt(t *testing.T, root string, options ...mcp.Option) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out, options...).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
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

func TestDefaultMCPDescriptionsRouteBeforeGenericTools(t *testing.T) {
	for _, catalog := range [][]map[string]any{
		listToolsWithRegistry(t, "full", operation.DefaultRegistry()),
		listToolsWithLiveReload(t),
	} {
		for _, tool := range catalog {
			name := tool["name"].(string)
			description := tool["description"].(string)
			if !strings.HasPrefix(description, "Use this tool instead of") {
				t.Errorf("%s description does not start with an explicit routing trigger: %q", name, description)
			}
		}
	}
}

func TestDefaultMCPCatalogDocumentsHighRiskToolBehavior(t *testing.T) {
	tools := listToolsWithRegistry(t, "full", operation.DefaultRegistry())
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	toolDescription := func(name string) string {
		t.Helper()
		tool, ok := byName[name]
		if !ok {
			t.Fatalf("tools/list omitted %s", name)
		}
		return tool["description"].(string)
	}
	paramDescription := func(toolName, paramName string) string {
		t.Helper()
		tool, ok := byName[toolName]
		if !ok {
			t.Fatalf("tools/list omitted %s", toolName)
		}
		properties := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
		property, ok := properties[paramName].(map[string]any)
		if !ok {
			t.Fatalf("%s schema omitted parameter %s", toolName, paramName)
		}
		return property["description"].(string)
	}
	assertContains := func(label, value string, want ...string) {
		t.Helper()
		for _, phrase := range want {
			if !strings.Contains(value, phrase) {
				t.Errorf("%s = %q, want wording containing %q", label, value, phrase)
			}
		}
	}

	assertContains("semantic_verify description", toolDescription("semantic_verify"), "Go verification formats files before diagnostics", "check-only skips all normalization")
	assertContains("semantic_verify.path", paramDescription("semantic_verify", "path"), "relative paths resolve from the active semedit workspace root", "runs formatting before diagnostics and can write files")
	assertContains("semantic_batch description", toolDescription("semantic_batch"), "earlier successful edits remain applied", "are not rolled back")
	for _, name := range []string{"semantic_maven_compile", "semantic_maven_test"} {
		assertContains(name+" description", toolDescription(name), "trust_workspace=true", "offline unless allow_network=true", "under .scratch", "build outputs")
	}
	assertContains("semantic_maven_compile.root", paramDescription("semantic_maven_compile", "root"), "absolute", "isolated worktree")
	assertContains("semantic_organize_imports.file", paramDescription("semantic_organize_imports", "file"), "Omit it to organize imports across the entire workspace", "may write multiple files")
	assertContains("semantic_add_build_dependency description", toolDescription("semantic_add_build_dependency"), "go.mod/go.sum", "may access the network")
	assertContains("semantic_add_build_dependency.package", paramDescription("semantic_add_build_dependency", "package"), "go.mod", "go.sum", "network")
	assertContains("semantic_assertion_mode description", toolDescription("semantic_assertion_mode"), "trust_workspace=true", "dry_run=true", "without trust or file writes")
	assertContains("semantic_assertion_mode.dry_run", paramDescription("semantic_assertion_mode", "dry_run"), "without writing", "trust_workspace is not required")
	assertContains("semantic_assertion_mode.trust_workspace", paramDescription("semantic_assertion_mode", "trust_workspace"), "Must be true", "non-dry-run")
	assertContains("semantic_rename.file", paramDescription("semantic_rename", "file"), "Required to select Rust or Java rename scope", "paths are relative to the workspace root", "Copy from the file field of a preceding semantic_lookup")
	assertContains("semantic_rename.symbol", paramDescription("semantic_rename", "symbol"), "Target: qualified symbol identifier", "Copy from the symbol field of a preceding semantic_lookup")
	assertContains("semantic_insert_declaration.source", paramDescription("semantic_insert_declaration", "source"), "Prefer semantic_insert_function", "semantic_insert_type", "semantic_insert_decl")
	assertContains("semantic_insert_function description", toolDescription("semantic_insert_function"), "one complete Go function or method", "parameters, results, and body")
	assertContains("semantic_insert_type description", toolDescription("semantic_insert_type"), "one Go struct, interface, or type alias", "instead of replace_file_content")
	assertContains("semantic_insert_decl description", toolDescription("semantic_insert_decl"), "one Go constant or variable", "instead of replace_file_content")
	assertContains("semantic_insert_declaration description", toolDescription("semantic_insert_declaration"), "generic placement controls", "Prefer semantic_insert_function", "semantic_insert_type", "semantic_insert_decl")
	assertContains("semantic_insert_structure description", toolDescription("semantic_insert_structure"), "structural construct", "instead of replace_file_content")
	assertContains("semantic_batch description", toolDescription("semantic_batch"), `{"tool":"semantic_tool_name","params":{...}}`, "never beside tool", "zero-based edit index", `"symbol":"Run"`)
	assertContains("semantic_lookup description", toolDescription("semantic_lookup"), "Returns symbol, file, kind, line, and column", "file-navigation reads", "not accepted as semedit tool parameters")
	assertContains("semantic_insert_case description", toolDescription("semantic_insert_case"), "exact discriminator expression", "kind=case alone does not identify", "construct_path")
	assertContains("semantic_insert_case.discriminator", paramDescription("semantic_insert_case", "discriminator"), "Selector: switch discriminant expression", "tagless switch", "Copy from the discriminator field")
}

func TestGeneratedSemanticInputSchemasRejectUnknownProperties(t *testing.T) {
	tools := listTools(t, "full")
	var verifySchema, batchSchema, insertFunctionSchema map[string]any
	for _, tool := range tools {
		switch tool["name"] {
		case "semantic_verify":
			verifySchema = tool["inputSchema"].(map[string]any)
		case "semantic_batch":
			batchSchema = tool["inputSchema"].(map[string]any)
		case "semantic_insert_function":
			insertFunctionSchema = tool["inputSchema"].(map[string]any)
		}
	}
	if verifySchema == nil || batchSchema == nil || insertFunctionSchema == nil {
		t.Fatal("tools/list omitted semantic_verify, semantic_batch, or semantic_insert_function")
	}
	for name, schema := range map[string]map[string]any{"semantic_verify": verifySchema, "semantic_batch": batchSchema} {
		if schema["additionalProperties"] != false {
			t.Errorf("%s additionalProperties = %#v, want false", name, schema["additionalProperties"])
		}
	}
	access := insertFunctionSchema["properties"].(map[string]any)["access_modifier"].(map[string]any)
	if !reflect.DeepEqual(access["enum"], []any{"infer", "public", "private"}) {
		t.Fatalf("Go insert-function access enum = %#v, want only infer/public/private", access["enum"])
	}
	edits := batchSchema["properties"].(map[string]any)["edits"].(map[string]any)
	branches := edits["items"].(map[string]any)["oneOf"].([]any)
	foundReplaceBody := false
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		if properties["tool"].(map[string]any)["const"] != "semantic_replace_body" {
			continue
		}
		foundReplaceBody = true
		params := properties["params"].(map[string]any)
		if params["additionalProperties"] != false {
			t.Errorf("semantic_batch semantic_replace_body params additionalProperties = %#v, want false", params["additionalProperties"])
		}
	}
	if !foundReplaceBody {
		t.Fatal("semantic_batch omitted semantic_replace_body")
	}
}

func TestScaffoldPurposeHeaderSchemaIsOptional(t *testing.T) {
	tools := listTools(t, "full")
	for _, tool := range tools {
		if tool["name"] != "semantic_scaffold_file" {
			continue
		}
		input := tool["inputSchema"].(map[string]any)
		properties := input["properties"].(map[string]any)
		header, ok := properties["purpose_header"].(map[string]any)
		if !ok || header["type"] != "string" {
			t.Fatalf("purpose_header schema = %#v, want string", properties["purpose_header"])
		}
		if got := input["required"]; got != nil {
			for _, name := range got.([]any) {
				if name == "purpose_header" {
					t.Fatal("purpose_header must remain optional")
				}
			}
		}
		if description, _ := header["description"].(string); !strings.Contains(description, "why the file exists") {
			t.Errorf("purpose_header description = %q, want file-purpose guidance", description)
		}
		return
	}
	t.Fatal("tools/list omitted semantic_scaffold_file")
}

func toAnySlice(values []string) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func assertInspectOutputSchema(t *testing.T, outputSchema any) {
	t.Helper()
	output, ok := outputSchema.(map[string]any)
	if !ok {
		t.Fatalf("inspection output schema = %#v", outputSchema)
	}
	properties, ok := output["properties"].(map[string]any)
	if !ok {
		t.Fatalf("inspection output properties = %#v", output["properties"])
	}
	result, ok := properties["result"].(map[string]any)
	if !ok {
		t.Fatalf("inspection result schema = %#v", properties["result"])
	}
	resultProperties, ok := result["properties"].(map[string]any)
	if !ok || resultProperties["inspection"] == nil {
		t.Fatalf("inspection result schema omitted legacy text: %#v", result)
	}
	continuations, ok := resultProperties["continuations"].(map[string]any)
	if !ok {
		t.Fatalf("inspection result schema omitted continuations: %#v", resultProperties)
	}
	items, ok := continuations["items"].(map[string]any)
	if !ok {
		t.Fatalf("inspection continuation item schema = %#v", continuations["items"])
	}
	itemProperties, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("inspection continuation properties = %#v", items["properties"])
	}
	for _, field := range []string{"file", "symbol", "expected_revision", "in_function", "in_type", "in_declaration"} {
		if _, ok := itemProperties[field]; !ok {
			t.Errorf("inspection continuation schema omitted %q", field)
		}
	}
	if !reflect.DeepEqual(items["required"], []any{"file", "symbol", "expected_revision"}) {
		t.Errorf("inspection continuation required = %#v", items["required"])
	}
}

func TestMCPContinuationDescriptionsCoverConstructSources(t *testing.T) {
	tools := make(map[string]map[string]any)
	for _, tool := range listTools(t, "full") {
		tools[tool["name"].(string)] = tool
	}
	fileSources := []string{
		"semantic_lookup", "semantic_rename", "semantic_replace_body", "semantic_replace_construct",
		"semantic_insert_case", "semantic_insert_function", "semantic_insert_type", "semantic_insert_decl",
		"semantic_insert_declaration", "semantic_insert_structure", "semantic_replace_decl",
		"semantic_extract_function", "semantic_extract_variable", "semantic_inline_variable", "semantic_organize_imports", "semantic_scaffold_file",
	}
	inFunctionSources := []string{
		"semantic_replace_construct", "semantic_insert_case", "semantic_insert_structure",
	}
	discriminatorSources := []string{"semantic_insert_case", "semantic_insert_structure"}
	assertSources := func(toolName, parameter string, sources []string) {
		t.Helper()
		tool, ok := tools[toolName]
		if !ok {
			t.Fatalf("tools/list omitted %s", toolName)
		}
		properties := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
		field, ok := properties[parameter].(map[string]any)
		if !ok {
			t.Fatalf("%s inputSchema omitted %s", toolName, parameter)
		}
		description, ok := field["description"].(string)
		if !ok {
			t.Fatalf("%s.%s has no description", toolName, parameter)
		}
		for _, source := range sources {
			if !strings.Contains(description, source) {
				t.Errorf("%s.%s description omitted %s: %s", toolName, parameter, source, description)
			}
		}
	}
	assertSources("semantic_insert_decl", "file", fileSources)
	assertSources("semantic_extract_function", "symbol", []string{"semantic_extract_function"})
	assertSources("semantic_extract_function", "in_function", []string{"semantic_extract_function"})
	assertSources("semantic_extract_variable", "symbol", []string{"semantic_extract_variable"})
	assertSources("semantic_extract_variable", "in_function", []string{"semantic_extract_variable"})
	for _, tool := range inFunctionSources {
		assertSources(tool, "in_function", inFunctionSources)
		assertSources(tool, "discriminator", discriminatorSources)
	}
}

func TestMoveConstructIsAdvertisedInBatchWithDestinationReceipt(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_move_construct")
	if !ok || !entry.Batchable {
		t.Fatalf("semantic_move_construct registry entry = %+v, present=%v; want batchable", entry, ok)
	}
	tools := listTools(t, "full")
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	move := byName["semantic_move_construct"]
	if move == nil {
		t.Fatal("tools/list omitted semantic_move_construct")
	}
	moveInput := move["inputSchema"].(map[string]any)
	moveProperties := moveInput["properties"].(map[string]any)
	for _, name := range []string{"file", "destination_file", "symbol"} {
		if property, ok := moveProperties[name].(map[string]any); !ok || property["type"] != "string" {
			t.Errorf("move input property %s = %#v, want string", name, moveProperties[name])
		}
	}
	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	edits := batch["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)
	branches := edits["items"].(map[string]any)["oneOf"].([]any)
	found := false
	for _, raw := range branches {
		properties := raw.(map[string]any)["properties"].(map[string]any)
		if properties["tool"].(map[string]any)["const"] == "semantic_move_construct" {
			found = true
			if !reflect.DeepEqual(properties["params"], moveInput) {
				t.Fatal("batch move params schema does not match the public operation schema")
			}
		}
	}
	if !found {
		t.Fatal("semantic_batch schema omitted semantic_move_construct")
	}
	output := batch["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)["properties"].(map[string]any)
	results := output["results"].(map[string]any)["items"].(map[string]any)["oneOf"].([]any)
	for _, rawBranch := range results {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		if properties["tool"].(map[string]any)["const"] != "semantic_move_construct" {
			continue
		}
		destination := properties["destination_file"].(map[string]any)
		if destination["type"] != "string" || !strings.Contains(destination["description"].(string), "not a successful write") {
			t.Fatalf("destination_file result schema = %#v, want requested-target semantics", destination)
		}
		return
	}
	t.Fatal("batch output schema omitted semantic_move_construct result branch")
}

func TestMCPStaleContinuationsAreNotAdvertised(t *testing.T) {
	tools := make(map[string]map[string]any)
	for _, tool := range listTools(t, "full") {
		tools[tool["name"].(string)] = tool
	}
	for _, check := range []struct {
		tool   string
		fields []string
	}{
		{tool: "semantic_inline_variable", fields: []string{"symbol"}},
		{tool: "semantic_extract_variable", fields: []string{"discriminator"}},
		{tool: "semantic_remove_construct", fields: []string{"symbol", "in_function", "discriminator", "target_case", "construct_path"}},
		{tool: "semantic_move_construct", fields: []string{"destination_file"}},
	} {
		output := tools[check.tool]["outputSchema"].(map[string]any)
		properties := output["properties"].(map[string]any)
		for _, field := range check.fields {
			if _, ok := properties[field]; ok {
				t.Errorf("%s output advertises stale continuation %q", check.tool, field)
			}
		}
	}
}

func TestTypedRecursiveSchemaFlowsThroughToolAndBatchIngress(t *testing.T) {
	root := schemaWorkspace(t)
	registry := operation.NewRegistry()
	payloadSchema := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{
		"text": {Kind: operation.SchemaString},
		"rows": {Kind: operation.SchemaArray, Items: &operation.ValueSchema{Kind: operation.SchemaInteger}},
	}, Required: []string{"text", "rows"}}
	handler := func(_ context.Context, _ operation.CallContext, raw map[string]any) (string, error) {
		if invalid, _ := raw["invalid_output"].(bool); invalid {
			return "malformed", nil
		}
		return "accepted", nil
	}
	targetSchema := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{
		"file": {Kind: operation.SchemaString}, "symbol": {Kind: operation.SchemaString},
	}, Required: []string{"file", "symbol"}}
	provider := operation.Def[map[string]any, string]{
		Key: "typed-schema-provider", MCPName: "semantic_typed_schema_provider",
		Params: []operation.ParameterContract{{Name: "invalid output", JSONName: "invalid_output", Type: operation.ParamBoolean}},
		Output: &operation.OperationOutputContract{
			ResultSchema: &operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{
				"message": {Kind: operation.SchemaString}, "target": targetSchema,
			}, Required: []string{"message", "target"}},
			Continuations: []operation.OutputContinuation{
				{Path: "result.target.file", Schema: operation.ValueSchema{Kind: operation.SchemaString}, Role: operation.RoleTarget, Consumers: []string{"semantic_typed_schema_consumer"}},
				{Path: "result.target.symbol", Schema: operation.ValueSchema{Kind: operation.SchemaString}, Role: operation.RoleTarget, Consumers: []string{"semantic_typed_schema_consumer"}},
			},
		},
		Parse:    func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){backend.LanguageAuto: handler},
		ProjectOutput: func(value string) (any, error) {
			result := map[string]any{"message": value}
			if value != "malformed" {
				result["target"] = map[string]any{"file": "main.go", "symbol": "Widget"}
			}
			return result, nil
		},
	}
	consumer := operation.Def[map[string]any, string]{
		Key: "typed-schema-consumer", MCPName: "semantic_typed_schema_consumer", Batchable: true,
		Params: []operation.ParameterContract{
			{Name: "payload", JSONName: "payload", Type: operation.ParamObject, Required: true, Schema: &payloadSchema},
			{Name: "file", JSONName: "file", Type: operation.ParamString, Required: true, Role: operation.RoleTarget, SourceFields: []string{"semantic_typed_schema_provider.result.target.file"}},
			{Name: "symbol", JSONName: "symbol", Type: operation.ParamString, Required: true, Role: operation.RoleTarget, SourceFields: []string{"semantic_typed_schema_provider.result.target.symbol"}},
		},
		Parse:    func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){backend.LanguageAuto: handler},
	}
	if err := operation.Register(registry, provider); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	if err := operation.Register(registry, consumer); err != nil {
		t.Fatalf("register consumer: %v", err)
	}
	tools := listToolsAtWithRegistry(t, "full", root, registry)
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	providerOutput := byName[provider.MCPName]["outputSchema"].(map[string]any)
	providerProperties := providerOutput["properties"].(map[string]any)
	resultSchema := providerProperties["result"].(map[string]any)
	targetShape := resultSchema["properties"].(map[string]any)["target"].(map[string]any)
	if targetShape["additionalProperties"] != false || targetShape["type"] != "object" {
		t.Fatalf("typed nested target schema = %#v", targetShape)
	}
	consumerInput := byName[consumer.MCPName]["inputSchema"].(map[string]any)
	payload := consumerInput["properties"].(map[string]any)["payload"].(map[string]any)
	if payload["additionalProperties"] != false {
		t.Fatalf("consumer payload schema = %#v", payload)
	}
	batchInput := byName["semantic_batch"]["inputSchema"].(map[string]any)
	items := batchInput["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	branches := items["oneOf"].([]any)
	var batchPayload map[string]any
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		if properties["tool"].(map[string]any)["const"] == consumer.MCPName {
			batchPayload = properties["params"].(map[string]any)["properties"].(map[string]any)["payload"].(map[string]any)
		}
	}
	if batchPayload == nil || batchPayload["additionalProperties"] != false {
		t.Fatalf("batch payload schema = %#v", batchPayload)
	}
	var projectedOutput bytes.Buffer
	projectedRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_typed_schema_provider","arguments":{}}}`
	if err := mcp.NewServer("full", root, &projectedOutput, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(projectedRequest+"\n")); err != nil {
		t.Fatalf("serve projected provider: %v", err)
	}
	var projectedResponse map[string]any
	if err := json.Unmarshal(projectedOutput.Bytes(), &projectedResponse); err != nil {
		t.Fatalf("decode projected provider response: %v", err)
	}
	projectedResult := projectedResponse["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	if projectedResult["target"].(map[string]any)["symbol"] != "Widget" {
		t.Fatalf("projected continuation missing: %#v", projectedResult)
	}
	projectedTarget := projectedResult["target"].(map[string]any)
	consumerRequest, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": consumer.MCPName, "arguments": map[string]any{"payload": map[string]any{"text": "copied", "rows": []any{0}}, "file": projectedTarget["file"], "symbol": projectedTarget["symbol"]}}})
	if err != nil {
		t.Fatalf("encode nested continuation consumer call: %v", err)
	}
	var consumerOutput bytes.Buffer
	if err := mcp.NewServer("full", root, &consumerOutput, mcp.WithRegistry(registry)).Serve(context.Background(), bytes.NewReader(append(consumerRequest, '\n'))); err != nil {
		t.Fatalf("serve nested continuation consumer: %v", err)
	}
	if strings.Contains(consumerOutput.String(), "isError\":true") || !strings.Contains(consumerOutput.String(), "accepted") {
		t.Fatalf("copied nested continuation was rejected: %s", consumerOutput.String())
	}
	var malformedOutput bytes.Buffer
	malformedRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_typed_schema_provider","arguments":{"invalid_output":true}}}`
	if err := mcp.NewServer("full", root, &malformedOutput, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(malformedRequest+"\n")); err != nil {
		t.Fatalf("serve malformed provider: %v", err)
	}
	if !strings.Contains(malformedOutput.String(), "violates declared schema") || strings.Contains(malformedOutput.String(), `"structuredContent":{"result":{}}`) {
		t.Fatalf("malformed projected result was not rejected: %s", malformedOutput.String())
	}
	for _, request := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_typed_schema_consumer","arguments":{"payload":{"text":"","rows":[0],"unexpected":true},"file":"main.go","symbol":"Widget"}}}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_typed_schema_consumer","params":{"payload":{"text":"","rows":[0],"unexpected":true},"file":"main.go","symbol":"Widget"}}]}}}`,
	} {
		var output bytes.Buffer
		if err := mcp.NewServer("full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(request+"\n")); err != nil {
			t.Fatalf("serve typed request: %v", err)
		}
		if !strings.Contains(output.String(), "unknown property \\\"unexpected\\\"") {
			t.Errorf("typed ingress did not reject unknown nested field: %s", output.String())
		}
	}
}

func TestBatchToolAdvertisesAndEmitsRegisteredResultShapes(t *testing.T) {
	root := schemaWorkspace(t)
	registry := operation.NewRegistry()
	objectSchema := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{
		"value": {Kind: operation.SchemaInteger},
	}, Required: []string{"value"}}
	arraySchema := operation.ValueSchema{Kind: operation.SchemaArray, Items: &operation.ValueSchema{Kind: operation.SchemaString}}
	nullSchema := operation.ValueSchema{Kind: operation.SchemaString, Nullable: true}
	makeEntry := func(key, name string, schema operation.ValueSchema, project func(string) any) error {
		return operation.Register(registry, operation.Def[map[string]any, string]{
			Key: key, MCPName: name, Batchable: true, Output: &operation.OperationOutputContract{ResultSchema: &schema},
			Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil },
			Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){
				backend.LanguageAuto: func(context.Context, operation.CallContext, map[string]any) (string, error) { return "ok", nil },
			},
			ProjectOutput: func(value string) (any, error) { return project(value), nil },
		})
	}
	if err := makeEntry("typed-object", "semantic_typed_object", objectSchema, func(string) any { return map[string]any{"value": 7} }); err != nil {
		t.Fatalf("register object provider: %v", err)
	}
	if err := makeEntry("typed-scalar", "semantic_typed_scalar", operation.ValueSchema{Kind: operation.SchemaString}, func(string) any { return "scalar" }); err != nil {
		t.Fatalf("register scalar provider: %v", err)
	}
	if err := makeEntry("typed-array", "semantic_typed_array", arraySchema, func(string) any { return []string{"array"} }); err != nil {
		t.Fatalf("register array provider: %v", err)
	}
	if err := makeEntry("typed-null", "semantic_typed_null", nullSchema, func(string) any { return nil }); err != nil {
		t.Fatalf("register null provider: %v", err)
	}
	if err := operation.Register(registry, operation.Def[map[string]any, string]{
		Key: "legacy-output", MCPName: "semantic_legacy_output", Batchable: true,
		Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){
			backend.LanguageAuto: func(context.Context, operation.CallContext, map[string]any) (string, error) { return "legacy", nil },
		},
	}); err != nil {
		t.Fatalf("register legacy operation: %v", err)
	}

	tools := listToolsAtWithRegistry(t, "full", root, registry)
	var batchTool map[string]any
	for _, tool := range tools {
		if tool["name"] == "semantic_batch" {
			batchTool = tool
			break
		}
	}
	if batchTool == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	resultsSchema := batchTool["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)["properties"].(map[string]any)["results"].(map[string]any)["items"].(map[string]any)
	branches := resultsSchema["oneOf"].([]any)
	for name, wantKind := range map[string]string{
		"semantic_typed_object": "object", "semantic_typed_scalar": "string", "semantic_typed_array": "array", "semantic_typed_null": "nullable",
	} {
		var found bool
		for _, rawBranch := range branches {
			branch := rawBranch.(map[string]any)
			properties := branch["properties"].(map[string]any)
			if properties["tool"].(map[string]any)["const"] == name {
				resultSchema, ok := properties["result"].(map[string]any)
				if !ok {
					t.Fatalf("batch schema omitted result for %s: %#v", name, branch)
				}
				if wantKind == "nullable" {
					types, ok := resultSchema["type"].([]any)
					if !ok || !reflect.DeepEqual(types, []any{"string", "null"}) {
						t.Fatalf("nullable result schema for %s = %#v", name, resultSchema)
					}
				} else if resultSchema["type"] != wantKind {
					t.Fatalf("batch result schema for %s = %#v, want type %s", name, resultSchema, wantKind)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("batch output schema omitted registered result branch %s", name)
		}
	}
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		if properties["tool"].(map[string]any)["const"] == "semantic_legacy_output" {
			if _, hasResult := properties["result"]; hasResult {
				t.Fatalf("legacy batch branch advertises a result: %#v", branch)
			}
		}
	}

	var edits = make([]map[string]any, 0, 5)
	for _, name := range []string{"semantic_typed_object", "semantic_typed_scalar", "semantic_typed_array", "semantic_typed_null", "semantic_legacy_output"} {
		edits = append(edits, map[string]any{"tool": name, "params": map[string]any{}})
	}
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "semantic_batch", "arguments": map[string]any{"edits": edits}}})
	if err != nil {
		t.Fatalf("encode batch request: %v", err)
	}
	var output bytes.Buffer
	if err := mcp.NewServer("full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), bytes.NewReader(append(request, '\n'))); err != nil {
		t.Fatalf("serve batch typed results: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode batch results: %v", err)
	}
	structured := response["result"].(map[string]any)["structuredContent"].(map[string]any)
	batchResult := structured["result"].(map[string]any)
	results := batchResult["results"].([]any)
	expected := map[string]any{
		"semantic_typed_object": map[string]any{"value": float64(7)},
		"semantic_typed_scalar": "scalar",
		"semantic_typed_array":  []any{"array"},
		"semantic_typed_null":   nil,
	}
	for index, rawResult := range results {
		result := rawResult.(map[string]any)
		name := result["tool"].(string)
		if name == "semantic_verify" {
			if result["status"] != "error" || result["error"] == nil {
				t.Errorf("synthetic verification row = %#v, want explicit missing-registration error", result)
			}
			continue
		}
		if want, typed := expected[name]; typed {
			got, present := result["result"]
			if !present || !reflect.DeepEqual(got, want) {
				t.Errorf("batch result %s = %#v present=%t, want %#v", name, got, present, want)
			}
			continue
		}
		if name == "semantic_legacy_output" {
			if _, present := result["result"]; present {
				t.Errorf("legacy batch result unexpectedly has result: %#v", result)
			}
			continue
		}
		t.Errorf("unexpected batch result at index %d: %#v", index, result)
	}
}

func TestInspectProjectionFeedsGuardedReplaceBody(t *testing.T) {
	root := schemaWorkspace(t)
	path := filepath.Join(root, "main.go")
	initial := []byte("package sample\nfunc Value() int { return 1 }\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write inspection fixture: %v", err)
	}
	var inspectOutput bytes.Buffer
	inspectServer := mcp.NewServer("full", root, &inspectOutput)
	inspectRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_inspect_symbol","arguments":{"symbol":"Value","file":"main.go","language":"go","trust_workspace":true}}}` + "\n"
	if err := inspectServer.Serve(context.Background(), strings.NewReader(inspectRequest)); err != nil {
		t.Fatalf("serve inspect call: %v", err)
	}
	var inspected map[string]any
	if err := json.Unmarshal(inspectOutput.Bytes(), &inspected); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	result := inspected["result"].(map[string]any)
	structured := result["structuredContent"].(map[string]any)["result"].(map[string]any)
	inspection := structured["inspection"].(string)
	if got := result["content"].([]any)[0].(map[string]any)["text"]; got != inspection {
		t.Fatalf("CLI-compatible inspection text = %#v, projected inspection = %#v", got, inspection)
	}
	records := structured["continuations"].([]any)
	if len(records) != 1 {
		t.Fatalf("inspection continuations = %#v", records)
	}
	record := records[0].(map[string]any)
	if record["file"] != "main.go" || record["symbol"] != "Value" || record["expected_revision"] == "" || record["in_function"] != "Value" {
		t.Fatalf("inspection target projection = %#v", record)
	}
	arguments := map[string]any{
		"file":              record["file"],
		"symbol":            record["symbol"],
		"expected_revision": record["expected_revision"],
		"body":              "return 2",
	}
	guardedRequest, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "semantic_replace_body", "arguments": arguments}})
	if err != nil {
		t.Fatalf("encode guarded edit: %v", err)
	}
	var guardedOutput bytes.Buffer
	guardedServer := mcp.NewServer("full", root, &guardedOutput)
	if err := guardedServer.Serve(context.Background(), bytes.NewReader(append(guardedRequest, '\n'))); err != nil {
		t.Fatalf("serve guarded edit: %v", err)
	}
	if strings.Contains(guardedOutput.String(), "isError\":true") {
		t.Fatalf("guarded edit rejected matching captured evidence: %s", guardedOutput.String())
	}
	staleRequest, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "semantic_replace_body", "arguments": arguments}})
	if err != nil {
		t.Fatalf("encode stale guarded edit: %v", err)
	}
	var staleOutput bytes.Buffer
	staleServer := mcp.NewServer("full", root, &staleOutput)
	if err := staleServer.Serve(context.Background(), bytes.NewReader(append(staleRequest, '\n'))); err != nil {
		t.Fatalf("serve stale guarded edit: %v", err)
	}
	if !strings.Contains(staleOutput.String(), "expected_revision does not match captured source") || !strings.Contains(staleOutput.String(), "isError\":true") {
		t.Fatalf("stale guarded edit did not refuse explicitly: %s", staleOutput.String())
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read guarded edit result: %v", err)
	}
	if !strings.Contains(string(actual), "return 2") || strings.Contains(string(actual), "return 1") {
		t.Fatalf("stale request changed the successful edit: %s", actual)
	}
}

func TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision(t *testing.T) {
	root := schemaWorkspace(t)
	path := filepath.Join(root, "main.go")
	initial := []byte("package sample\n// identity:before\nvar Widget = 1\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	initialDigest := sha256.Sum256(initial)
	registry := operation.NewRegistry()
	providerName := "semantic_receipt_mutation"
	consumerName := "semantic_receipt_consumer"
	fileSchema := operation.SymbolTargetSchema().Properties["file"]
	symbolSchema := operation.SymbolTargetSchema().Properties["symbol"]
	provider := operation.Def[map[string]any, operation.EditReceipt]{
		Key: "receipt-mutation", MCPName: providerName,
		Params: []operation.ParameterContract{{Name: "identity", JSONName: "identity", Type: operation.ParamString, Required: true}, {Name: "remove", JSONName: "remove", Type: operation.ParamBoolean}, {Name: "replacement", JSONName: "replacement", Type: operation.ParamBoolean}},
		Output: &operation.OperationOutputContract{ResultSchema: func() *operation.ValueSchema { schema := operation.EditReceiptSchema(); return &schema }(), Continuations: []operation.OutputContinuation{
			{Path: "result.continuations[].file", Schema: fileSchema, Role: operation.RoleTarget, Consumers: []string{consumerName}},
			{Path: "result.continuations[].symbol", Schema: symbolSchema, Role: operation.RoleTarget, Consumers: []string{consumerName}},
		}},
		Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (operation.EditReceipt, error){backend.LanguageAuto: func(ctx context.Context, cc operation.CallContext, raw map[string]any) (operation.EditReceipt, error) {
			identity, _ := raw["identity"].(string)
			remove, _ := raw["remove"].(bool)
			replacement, _ := raw["replacement"].(bool)
			source := []byte("package sample\n// identity:" + identity + "\nvar Widget = 2\n")
			if replacement {
				source = []byte("package sample\n// identity:replacement\nvar Widget = 2\n")
			}
			if remove {
				source = []byte("package sample\nvar Other = 2\n")
			}
			if err := sourcebuf.WriteFile(ctx, path, source); err != nil {
				return operation.EditReceipt{}, err
			}
			return operation.EditReceipt{SchemaVersion: "semedit.edit-receipt/v1", Status: operation.EditPersisted, Operation: "receipt-mutation", ReadSet: []operation.FileRevision{{Path: "main.go", Revision: hex.EncodeToString(initialDigest[:])}}, Continuations: []operation.SymbolTarget{{File: "main.go", Symbol: "Widget"}}}, nil
		}},
		FinalizeOutput: func(cc operation.CallContext, request map[string]any, draft operation.EditReceipt) (operation.EditReceipt, error) {
			identity, _ := request["identity"].(string)
			remove, _ := request["remove"].(bool)
			replacement, _ := request["replacement"].(bool)
			resolve := func(_ context.Context, target operation.SymbolTarget) (operation.SymbolTarget, bool, error) {
				content, err := os.ReadFile(filepath.Join(root, target.File))
				if errors.Is(err, os.ErrNotExist) {
					return target, false, nil
				}
				if err != nil {
					return target, false, err
				}
				live := bytes.Contains(content, []byte("// identity:"+identity+"\n")) && bytes.Contains(content, []byte("var "+target.Symbol+" ="))
				return target, live, nil
			}
			plannedIdentity := identity
			if replacement {
				plannedIdentity = "replacement"
			}
			plannedBytes := []byte("package sample\n// identity:" + plannedIdentity + "\nvar Widget = 2\n")
			if remove {
				plannedBytes = []byte("package sample\nvar Other = 2\n")
			}
			plannedDigest := sha256.Sum256(plannedBytes)
			expectedRevision := hex.EncodeToString(plannedDigest[:])
			continuations := append([]operation.SymbolTarget(nil), draft.Continuations...)
			for index := range continuations {
				continuations[index].ExpectedRevision = &expectedRevision
			}
			draftValue := operation.EditReceiptDraft{Operation: draft.Operation, Status: draft.Status, PlannedChanges: []operation.FileRevision{{Path: "main.go", Revision: expectedRevision}}, ReadSet: draft.ReadSet, Continuations: continuations}
			return operation.FinalizeEditReceipt(cc.Ctx, cc.WorkDir, cc.InBatch, draftValue, resolve)
		},
		ProjectOutput: func(receipt operation.EditReceipt) (any, error) { return receipt, nil },
	}
	consumer := operation.Def[map[string]any, string]{Key: "receipt-consumer", MCPName: consumerName, Batchable: true, Params: []operation.ParameterContract{
		{Name: "file", JSONName: "file", Type: operation.ParamString, Required: true, Role: operation.RoleTarget, SourceFields: []string{providerName + ".result.continuations[].file"}},
		{Name: "symbol", JSONName: "symbol", Type: operation.ParamString, Required: true, Role: operation.RoleTarget, SourceFields: []string{providerName + ".result.continuations[].symbol"}},
	}, Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil }, Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){backend.LanguageAuto: func(context.Context, operation.CallContext, map[string]any) (string, error) { return "consumed", nil }}}
	if err := operation.Register(registry, provider); err != nil {
		t.Fatalf("register receipt provider: %v", err)
	}
	if err := operation.Register(registry, consumer); err != nil {
		t.Fatalf("register receipt consumer: %v", err)
	}
	requests := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_receipt_mutation","arguments":{"identity":"after"}}}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_receipt_mutation","arguments":{"identity":"removed","remove":true}}}` + "\n" + `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_receipt_mutation","arguments":{"identity":"original","replacement":true}}}` + "\n"
	var output bytes.Buffer
	if err := mcp.NewServer("full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(requests)); err != nil {
		t.Fatalf("serve receipt calls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("MCP responses = %d: %s", len(lines), output.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	structured := first["result"].(map[string]any)["structuredContent"].(map[string]any)
	receipt := structured["result"].(map[string]any)
	if receipt["status"] != string(operation.EditPersisted) {
		t.Fatalf("receipt status = %#v", receipt["status"])
	}
	changes := receipt["changes"].([]any)
	readSet := receipt["read_set"].([]any)
	if changes[0].(map[string]any)["revision"] == readSet[0].(map[string]any)["revision"] {
		t.Fatalf("receipt did not separate final change revision from analysis read set: %#v", receipt)
	}
	continuations := receipt["continuations"].([]any)
	if len(continuations) != 1 {
		t.Fatalf("persisted continuation = %#v", continuations)
	}
	firstMutationBytes := []byte("package sample\n// identity:after\nvar Widget = 2\n")
	finalDigest := sha256.Sum256(firstMutationBytes)
	if continuations[0].(map[string]any)["expected_revision"] != hex.EncodeToString(finalDigest[:]) {
		t.Fatalf("continuation revision does not match final disk: %#v", continuations[0])
	}
	var second map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatal(err)
	}
	removedReceipt := second["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	if removed, exists := removedReceipt["continuations"]; exists && len(removed.([]any)) != 0 {
		t.Fatalf("removed identity remained live: %#v", removed)
	}
	var replacementResponse map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &replacementResponse); err != nil {
		t.Fatal(err)
	}
	replacementReceipt := replacementResponse["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	if continuations, exists := replacementReceipt["continuations"]; exists && len(continuations.([]any)) != 0 {
		t.Fatalf("same-name replacement inherited a stale identity: %#v", continuations)
	}
}

func TestMCPIntegerNotationAndRangeValidation(t *testing.T) {
	root := schemaWorkspace(t)
	registry := operation.DefaultRegistry()
	for _, item := range []struct {
		label, start, end string
		invalid           bool
	}{
		{label: "decimal integral", start: "1.0", end: "1e0"},
		{label: "large adjacent integers", start: "9007199254740992", end: "9007199254740993"},
		{label: "fractional", start: "1.5", end: "2", invalid: true},
		{label: "native overflow", start: "1", end: "9223372036854775808", invalid: true},
		{label: "huge positive exponent", start: "1", end: "1e1000000000", invalid: true},
		{label: "huge negative exponent", start: "1", end: "1e-1000000000", invalid: true},
		{label: "zero huge exponent", start: "0e1000000000", end: "0e-1000000000"},
	} {
		for _, batch := range []bool{false, true} {
			t.Run(item.label+map[bool]string{false: "/direct", true: "/batch"}[batch], func(t *testing.T) {
				edit := `{"tool":"semantic_extract_function","params":{"file":"main.go","symbol":"Helper","in_function":"Example","kind":"function","start_line":` + item.start + `,"end_line":` + item.end + `}}`
				request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_extract_function","arguments":{"file":"main.go","symbol":"Helper","in_function":"Example","kind":"function","start_line":` + item.start + `,"end_line":` + item.end + `}}}`
				if batch {
					request = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[` + edit + `]}}}`
				}
				var output bytes.Buffer
				if err := mcp.NewServer("full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(request+"\n")); err != nil {
					t.Fatalf("serve numeric request: %v", err)
				}
				gotIntegerError := strings.Contains(output.String(), "must be an integer")
				if item.invalid != gotIntegerError {
					t.Fatalf("integer validation mismatch for %s: %s", item.label, output.String())
				}
			})
		}
	}
}

func TestTypedContinuationRegistrationRejectsIncompatibleConsumers(t *testing.T) {
	stringSchema := operation.ValueSchema{Kind: operation.SchemaString}
	integerSchema := operation.ValueSchema{Kind: operation.SchemaInteger}
	producerObject := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"file": stringSchema, "symbol": stringSchema, "extra": stringSchema}, Required: []string{"file", "symbol"}}
	consumerObject := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"file": stringSchema, "symbol": stringSchema}, Required: []string{"file", "symbol"}}
	producerOptional := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"symbol": stringSchema}}
	consumerRequired := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"symbol": stringSchema}, Required: []string{"symbol"}}
	nullableString := operation.ValueSchema{Kind: operation.SchemaString, Nullable: true}
	enumWide := operation.ValueSchema{Kind: operation.SchemaString, Enum: []any{"old", "new"}}
	enumNarrow := operation.ValueSchema{Kind: operation.SchemaString, Enum: []any{"new"}}
	minOne, minTwo := 1, 2
	arrayWide := operation.ValueSchema{Kind: operation.SchemaArray, Items: &producerObject}
	arrayNarrow := operation.ValueSchema{Kind: operation.SchemaArray, Items: &consumerObject}
	for _, tc := range []struct {
		name           string
		source, target operation.ValueSchema
	}{
		{"scalar kind", stringSchema, integerSchema},
		{"closed object keys", producerObject, consumerObject},
		{"required field", producerOptional, consumerRequired},
		{"nullable", nullableString, stringSchema},
		{"enum", enumWide, enumNarrow},
		{"nested array object key", arrayWide, arrayNarrow},
		{"bounds", operation.ValueSchema{Kind: operation.SchemaArray, Items: &stringSchema, MinItems: &minOne}, operation.ValueSchema{Kind: operation.SchemaArray, Items: &stringSchema, MinItems: &minTwo}},
	} {
		if operation.SchemaAssignable(tc.source, tc.target) {
			t.Errorf("%s schema unexpectedly assignable", tc.name)
		}
	}
	if !operation.SchemaAssignable(operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"file": stringSchema}, Required: []string{"file"}}, operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"file": stringSchema, "optional": stringSchema}, Required: []string{"file"}}) {
		t.Fatal("safe closed object subset was rejected")
	}
	for _, tc := range []struct {
		name                           string
		providerSchema, consumerSchema operation.ValueSchema
		consumerName                   string
	}{
		{"type mismatch", stringSchema, integerSchema, "symbol"},
		{"renamed field", stringSchema, stringSchema, "renamed"},
		{"extra closed key", producerObject, consumerObject, "payload"},
		{"missing required key", producerOptional, consumerRequired, "payload"},
		{"nullable source", nullableString, stringSchema, "symbol"},
		{"enum narrowing", enumWide, enumNarrow, "symbol"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := operation.NewRegistry()
			providerName, consumerName := "semantic_contract_provider", "semantic_contract_consumer"
			provider := operation.Def[map[string]any, string]{Key: "contract-provider", MCPName: providerName, Output: &operation.OperationOutputContract{ResultSchema: &operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{"payload": tc.providerSchema}, Required: []string{"payload"}}, Continuations: []operation.OutputContinuation{{Field: "payload", Schema: tc.providerSchema, Role: operation.RoleTarget, Consumers: []string{consumerName}}}}, Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil }, ProjectOutput: func(string) (any, error) { return map[string]any{"payload": map[string]any{}}, nil }, Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){backend.LanguageAuto: func(context.Context, operation.CallContext, map[string]any) (string, error) { return "ok", nil }}}
			consumer := operation.Def[map[string]any, string]{Key: "contract-consumer", MCPName: consumerName, Params: []operation.ParameterContract{{Name: "payload", JSONName: tc.consumerName, Type: string(tc.consumerSchema.Kind), Required: true, Schema: &tc.consumerSchema, Role: operation.RoleTarget, SourceFields: []string{providerName + ".payload"}}}, Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil }, Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){backend.LanguageAuto: func(context.Context, operation.CallContext, map[string]any) (string, error) { return "ok", nil }}}
			if err := operation.Register(registry, provider); err != nil {
				t.Fatalf("register provider: %v", err)
			}
			if err := operation.Register(registry, consumer); err != nil {
				t.Fatalf("register consumer: %v", err)
			}
			if err := mcp.ValidateSourceFields(registry.All()); err == nil {
				t.Fatal("incompatible typed consumer registration was accepted")
			}
		})
	}
}
