// This file verifies that MCP batch discovery mirrors the operation registry contract.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"semedit/internal/backend"
	gobackend "semedit/internal/backend/golang"
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
			required:         []string{"file", "function", "kind", "source"},
			stringProperties: []string{"file", "function", "kind", "discriminator", "construct_path", "source"},
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

	assertContains("semantic_verify description", toolDescription("semantic_verify"), "Go verification runs formatting before diagnostics and can write files", "There is no dry-run")
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
	assertContains("semantic_rename.file", paramDescription("semantic_rename", "file"), "Required to select scope for Rust or Java rename", "active semedit workspace root")
	assertContains("semantic_insert_declaration.source", paramDescription("semantic_insert_declaration", "source"), "Prefer semantic_insert_function", "semantic_insert_type", "semantic_insert_decl")
	assertContains("semantic_insert_function description", toolDescription("semantic_insert_function"), "one Go function or method", "instead of replace_file_content")
	assertContains("semantic_insert_type description", toolDescription("semantic_insert_type"), "one Go struct, interface, or type alias", "instead of replace_file_content")
	assertContains("semantic_insert_decl description", toolDescription("semantic_insert_decl"), "one Go constant or variable", "instead of replace_file_content")
	assertContains("semantic_insert_declaration description", toolDescription("semantic_insert_declaration"), "generic placement controls", "Prefer semantic_insert_function", "semantic_insert_type", "semantic_insert_decl")
	assertContains("semantic_insert_structure description", toolDescription("semantic_insert_structure"), "structural construct", "instead of replace_file_content")
}

func toAnySlice(values []string) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}
