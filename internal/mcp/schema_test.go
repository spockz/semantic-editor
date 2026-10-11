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
	javabackend "semedit/internal/backend/java"
	"semedit/internal/mcp"
	"semedit/internal/mcptest"
	"semedit/internal/operation"
	"semedit/internal/sourcebuf"
	"slices"
)

const fieldInsertionToolName = "semantic_insert_field"

func TestMCPBatchSchemaDerivesFromRegistry(t *testing.T) {
	t.Parallel()
	for _, profile := range []string{"full", "mutations-only"} {
		t.Run(profile, func(t *testing.T) {
			assertBatchSchemaProfile(t, profile)
		})
	}
}

func TestMCPToolCatalogProfilesAndVariantFilters(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	base := toolsForOptions(t, root, mcp.WithToolProfile("base"))
	baseNames := toolNameSet(base)
	for _, name := range []string{"semantic_insert_construct", "semantic_replace_construct", "semantic_remove_construct", "semantic_move_construct", "semantic_rename", "semantic_lookup", "semantic_inspect_symbol", "semantic_outline", "semantic_find_references", "semantic_verify", "semantic_batch", "semantic_snapshot", "semantic_undo"} {
		if !baseNames[name] {
			t.Errorf("base tools/list omitted %s", name)
		}
	}
	for _, name := range []string{"semantic_insert_function", "semantic_insert_type", "semantic_insert_decl", "semantic_replace_body", "semantic_insert_field", "semantic_insert_case"} {
		if baseNames[name] {
			t.Errorf("base tools/list exposed %s", name)
		}
	}

	full := toolNameSet(toolsForOptions(t, root, mcp.WithToolProfile("full")))
	for _, name := range []string{"semantic_insert_function", "semantic_insert_type", "semantic_insert_decl", "semantic_replace_body"} {
		if !full[name] {
			t.Errorf("full tools/list omitted %s", name)
		}
	}

	for _, test := range []struct {
		patterns []string
		want     []string
	}{
		{patterns: []string{"insert:*"}, want: []string{"semantic_insert_function", "semantic_insert_type", "semantic_insert_decl", "semantic_insert_case", "semantic_insert_field"}},
		{patterns: []string{"*:function"}, want: []string{"semantic_insert_function", "semantic_replace_body"}},
		{patterns: []string{"insert:function,insert:type"}, want: []string{"semantic_insert_function", "semantic_insert_type"}},
	} {
		got := toolNameSet(toolsForOptions(t, root, mcp.WithToolProfile("base"), mcp.WithToolVariants(test.patterns)))
		for _, name := range test.want {
			if !got[name] {
				t.Errorf("patterns %v omitted %s", test.patterns, name)
			}
		}
		if len(got) != len(baseNames)+len(test.want) {
			t.Errorf("patterns %v exposed %d tools; base %d plus %d variants expected", test.patterns, len(got), len(baseNames), len(test.want))
		}
	}
}

func TestMCPInvalidToolVariantFailsBeforeServing(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	before, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = mcptest.NewServer(t, "full", root, &out, mcp.WithToolProfile("base"), mcp.WithToolVariants([]string{"remove:function"})).Serve(context.Background(), strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), "matched no registered operation") {
		t.Fatalf("invalid tool variant error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("server emitted output before rejecting variant: %s", out.String())
	}
	after, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("invalid tool variant changed workspace state")
	}
}

func TestMCPBaseCatalogStillDispatchesSpecializedBatchChild(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_insert_function","params":{"file":"main.go","source":"func Added() int { return 1 }"}}]}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &out, mcp.WithToolProfile("base")).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatalf("MCP serve failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("MCP responses = %d, want tools/list plus batch call: %s", len(lines), out.String())
	}
	var listed struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &listed); err != nil {
		t.Fatal(err)
	}
	if toolNameSet(listed.Result.Tools)["semantic_insert_function"] {
		t.Fatal("base tools/list exposed standalone semantic_insert_function")
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &response); err != nil {
		t.Fatal(err)
	}
	if response["error"] != nil {
		t.Fatalf("specialized batch child failed: %s", lines[1])
	}
	content, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "func Added() int") {
		t.Fatalf("specialized batch child did not mutate source: %s", content)
	}
}

func toolsForOptions(t *testing.T, root string, options ...mcp.Option) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
	if err := mcptest.NewServer(t, "full", root, &out, options...).Serve(context.Background(), strings.NewReader(request)); err != nil {
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

func toolNameSet(tools []map[string]any) map[string]bool {
	result := make(map[string]bool, len(tools))
	for _, tool := range tools {
		result[tool["name"].(string)] = true
	}
	return result
}

func TestMCPBatchSchemaUsesInjectedRegistry(t *testing.T) {
	t.Parallel()
	registry := operation.NewRegistry()
	params := []operation.ParameterContract{{Name: "file", JSONName: "file", Type: operation.ParamString, Required: true}}
	parse := func(map[string]any) (struct{}, error) { return struct{}{}, nil }
	if err := operation.Register(registry, operation.Def[struct{}, string]{
		Key: "custom-batch", Summary: "Custom batchable operation.", Params: params, MCPName: "semantic_custom_batch", Batchable: true, Parse: parse,
	}); err != nil {
		t.Fatal(err)
	}
	if err := operation.Register(registry, operation.Def[struct{}, string]{
		Key: "custom-nonbatch", Summary: "Custom non-batchable operation.", Params: params, MCPName: "semantic_custom_nonbatch", Parse: parse,
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
	customBatch, ok := byName["semantic_custom_batch"]
	if !ok {
		t.Fatal("tools/list omitted custom batchable operation")
	}
	assertStandardOutputSchema(t, customBatch["outputSchema"], "string")
	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch for injected registry")
	}
	items := batch["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	names := items["properties"].(map[string]any)["tool"].(map[string]any)["enum"].([]any)
	if !reflect.DeepEqual(names, []any{"semantic_custom_batch"}) {
		t.Fatalf("injected registry batch enum = %#v, want custom batchable tool only", names)
	}
	if items["additionalProperties"] != true || items["properties"].(map[string]any)["params"].(map[string]any)["type"] != "object" {
		t.Fatalf("injected batch schema is not compact: %#v", items)
	}
	if _, ok := byName["semantic_custom_nonbatch"]; !ok {
		t.Fatal("tools/list omitted custom non-batchable operation")
	}
}

func TestMCPContinuationDescriptionsNameSourcesWithinBudget(t *testing.T) {
	t.Parallel()
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
		"semantic_insert_declaration", "semantic_insert_construct", "semantic_replace_decl",
		"semantic_extract_function", "semantic_extract_constant", "semantic_extract_variable", "semantic_inline_variable", "semantic_organize_imports", "semantic_scaffold_file",
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
	t.Parallel()
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
	for key, role := range map[string]string{"symbol": "Target:", "file": "Target:", "in_function": "Selector:"} {
		field, ok := input[key].(map[string]any)
		if !ok {
			t.Fatalf("semantic_inline_variable input omitted %s", key)
		}
		if !strings.Contains(field["description"].(string), role) {
			t.Errorf("semantic_inline_variable.%s description omitted role %q", key, role)
		}
	}
	assertSemanticEditReceiptOutputSchema(t, "semantic_inline_variable", tool["outputSchema"])
	output := tool["outputSchema"].(map[string]any)["properties"].(map[string]any)
	if _, ok := output["in_function"]; ok {
		t.Error("semantic_inline_variable output advertises an in_function alias absent from its receipt")
	}
}

func TestSourceFieldsResolveToAdvertisedOutputSchema(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	for _, profile := range []string{"full", "mutations-only"} {
		t.Run(profile, func(t *testing.T) {
			for _, tool := range listTools(t, profile) {
				assertListedToolOutputSchema(t, tool)
			}
		})
	}
}

func assertListedToolOutputSchema(t *testing.T, tool map[string]any) {
	t.Helper()
	name, ok := tool["name"].(string)
	if !ok {
		t.Fatalf("tool has invalid name: %#v", tool["name"])
	}
	output := tool["outputSchema"]
	if output == nil {
		t.Fatalf("%s omitted outputSchema", name)
	}
	switch name {
	case "semantic_set_doc_comment", "semantic_extract_type", "semantic_change_signature", "semantic_introduce_parameter", "semantic_introduce_parameter_object":
		assertStructuralOutputSchema(t, output, name)
	case fieldInsertionToolName:
		assertInsertFieldOutputSchema(t, output)
	case "semantic_insert_composite_element":
		assertCompositeElementOutputSchema(t, output)
	case "semantic_remove_construct":
		assertRemoveConstructOutputSchema(t, output)
	case "semantic_extract_interface", "semantic_implement_contract":
		assertContractOutputSchema(t, name, output)
	case "semantic_inline_function", "semantic_migrate_type", "semantic_extract_constant", "semantic_extract_function", "semantic_extract_variable", "semantic_inline_variable":
		assertSemanticEditReceiptOutputSchema(t, name, output)
	case "semantic_rename", "semantic_organize_imports", "semantic_maven_compile", "semantic_maven_test", "semantic_verify":
		assertTypedOutcomeOrLegacyStringSchema(t, name, output)
	case "semantic_lookup":
		assertRegisteredTypedOutputSchema(t, name, output)
	case "semantic_insert_construct", "semantic_insert_declaration", "semantic_insert_function", "semantic_insert_type", "semantic_insert_decl",
		"semantic_replace_decl", "semantic_replace_body", "semantic_replace_construct", "semantic_insert_case", "semantic_move_construct", "semantic_scaffold_file":
		assertFileEditOutputSchema(t, name, output)
	case "semantic_batch":
		assertBatchOutputSchema(t, output)
	case "semantic_metrics":
		assertMetricsOutputSchema(t, output)
	case "report_feedback":
		assertFeedbackOutputSchema(t, output)
	case "semantic_inspect_symbol":
		assertStandardOutputSchema(t, output, "object")
		assertInspectOutputSchema(t, output)
	default:
		assertStandardOutputSchema(t, output, "string")
	}
}

func assertRegisteredTypedOutputSchema(t *testing.T, name string, output any) map[string]any {
	t.Helper()
	assertStandardOutputSchema(t, output, "object")
	entry, ok := operation.DefaultRegistry().LookupMCP(name)
	if !ok || entry.Output == nil || entry.Output.ResultSchema == nil || entry.ProjectOutput == nil {
		t.Fatalf("%s omitted its typed output contract or projector", name)
	}
	result := nativeResultSchema(t, output)
	assertResultSchemaMatchesRegistration(t, name, result, entry)
	return result
}

func assertFileEditOutputSchema(t *testing.T, name string, output any) {
	t.Helper()
	result := assertRegisteredTypedOutputSchema(t, name, output)
	properties := result["properties"].(map[string]any)
	for _, field := range []string{"result", "edit_receipt"} {
		if properties[field] == nil {
			t.Errorf("%s typed result omitted %q", name, field)
		}
	}
	if got := result["required"]; !reflect.DeepEqual(got, []any{"result", "edit_receipt"}) {
		t.Errorf("%s typed result required = %#v", name, got)
	}
	fieldsByTool := map[string][]string{
		"semantic_insert_construct":   {"file", "symbol", "in_type", "in_function", "discriminator", "line", "diff"},
		"semantic_insert_declaration": {"file", "symbol", "line"},
		"semantic_insert_function":    {"file", "symbol", "line"},
		"semantic_insert_type":        {"file", "symbol", "line"},
		"semantic_insert_decl":        {"file", "symbol", "line"},
		"semantic_replace_decl":       {"file", "symbol", "line", "diff"},
		"semantic_replace_body":       {"file", "symbol", "diff"},
		"semantic_replace_construct":  {"file", "in_function", "diff"},
		"semantic_insert_case":        {"file", "in_function", "discriminator", "diff"},
		"semantic_move_construct":     {"file", "symbol", "origin_file", "diff"},
		"semantic_scaffold_file":      {"file", "package"},
	}
	for _, field := range fieldsByTool[name] {
		if properties[field] == nil {
			t.Errorf("%s typed result omitted %q", name, field)
		}
	}
}

func assertMetricsOutputSchema(t *testing.T, output any) {
	t.Helper()
	assertStandardOutputSchema(t, output, "object")
	result := output.(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	records := result["properties"].(map[string]any)["records"].(map[string]any)
	item := records["items"].(map[string]any)
	if item["properties"].(map[string]any)["metrics"] == nil {
		t.Fatal("semantic_metrics record schema omitted metrics")
	}
	if !reflect.DeepEqual(result["required"], []any{"records"}) {
		t.Fatalf("semantic_metrics result required = %#v", result["required"])
	}
}

func assertFeedbackOutputSchema(t *testing.T, output any) {
	t.Helper()
	assertStandardOutputSchema(t, output, "object")
	result := output.(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	if got := result["required"]; !reflect.DeepEqual(got, []any{"status", "date", "report", "markdown"}) {
		t.Fatalf("feedback result required = %#v", got)
	}
	status := result["properties"].(map[string]any)["status"].(map[string]any)
	if status["const"] != "draft_only" {
		t.Fatalf("feedback status schema = %#v, want draft_only", status)
	}
}

func assertCompositeElementOutputSchema(t *testing.T, raw any) {
	t.Helper()
	assertStructuralOutputSchema(t, raw, "semantic_insert_composite_element")
	result := raw.(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	properties := result["properties"].(map[string]any)
	selectors, ok := properties["selector_continuations"].(map[string]any)
	if !ok || selectors["type"] != "array" {
		t.Fatalf("composite output omitted typed selector_continuations: %#v", properties)
	}
	item, ok := selectors["items"].(map[string]any)
	if !ok || item["type"] != "object" || item["additionalProperties"] != false {
		t.Fatalf("composite selector item is not a closed object: %#v", selectors["items"])
	}
	itemProperties := item["properties"].(map[string]any)
	for _, field := range []string{"file", "in_declaration", "discriminator", "expected_revision"} {
		if itemProperties[field] == nil {
			t.Errorf("composite selector omitted required field %q", field)
		}
	}
	if got := item["required"]; !reflect.DeepEqual(got, []any{"file", "in_declaration", "discriminator", "expected_revision"}) {
		t.Errorf("composite selector required = %#v", got)
	}
}

func assertRemoveConstructOutputSchema(t *testing.T, raw any) {
	t.Helper()
	assertStandardOutputSchema(t, raw, "object")
	output := raw.(map[string]any)["properties"].(map[string]any)
	result := nativeResultSchema(t, raw)
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_remove_construct")
	if !ok || entry.Output == nil || entry.Output.ResultSchema == nil {
		t.Fatalf("remove operation omitted its registered result schema")
	}
	assertResultSchemaMatchesRegistration(t, "semantic_remove_construct", result, entry)
	for _, field := range []string{"status", "safe_delete", "checks_run"} {
		if _, ok := result["properties"].(map[string]any)[field]; !ok {
			t.Errorf("remove result schema omitted %q", field)
		}
	}
	if len(entry.Output.Continuations) == 0 {
		for _, field := range []string{"file", "symbol", "expected_revision"} {
			if _, ok := output[field]; ok {
				t.Errorf("remove outputSchema advertised unresolved continuation alias %q", field)
			}
		}
	}
}

func assertStructuralOutputSchema(t *testing.T, raw any, name string) {
	t.Helper()
	assertStandardOutputSchema(t, raw, "object")
	result := raw.(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	properties := result["properties"].(map[string]any)
	if properties["result"].(map[string]any)["type"] != "string" {
		t.Errorf("%s result projection must be a string", name)
	}
	receipt, ok := properties["edit_receipt"].(map[string]any)
	if !ok || receipt["type"] != "object" {
		t.Fatalf("%s omitted its native edit_receipt object", name)
	}
	if got := receipt["required"]; !reflect.DeepEqual(got, []any{"schema_version", "status", "operation"}) {
		t.Errorf("%s edit_receipt required = %#v", name, got)
	}
}

func assertSemanticEditReceiptOutputSchema(t *testing.T, name string, schema any) {
	t.Helper()
	assertStandardOutputSchema(t, schema, "object")
	result := schema.(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	for _, field := range []string{"schema_version", "status", "operation", "changes", "read_set", "continuations"} {
		if _, ok := result["properties"].(map[string]any)[field]; !ok {
			t.Errorf("%s receipt schema omitted %q", name, field)
		}
	}
	continuations := result["properties"].(map[string]any)["continuations"].(map[string]any)
	item := continuations["items"].(map[string]any)
	for _, field := range []string{"file", "symbol", "expected_revision"} {
		if _, ok := item["properties"].(map[string]any)[field]; !ok {
			t.Errorf("%s continuation schema omitted %q", name, field)
		}
	}
}

func assertInsertFieldOutputSchema(t *testing.T, raw any) {
	t.Helper()
	assertStructuralOutputSchema(t, raw, fieldInsertionToolName)
	properties := raw.(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"in_type", "symbol"} {
		if _, ok := properties[field]; !ok {
			t.Errorf("%s output schema omitted %q", fieldInsertionToolName, field)
		}
	}
}

func TestMCPLiveReloadAdvertisesStructuredOutputSchema(t *testing.T) {
	t.Parallel()
	tools := listToolsWithLiveReload(t)
	for _, tool := range tools {
		if tool["name"] != "semantic_reload" {
			continue
		}
		output, ok := tool["outputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("semantic_reload outputSchema = %#v, want object", tool["outputSchema"])
		}
		assertStandardOutputSchema(t, output, "object")
		result := nativeResultSchema(t, output)
		status := result["properties"].(map[string]any)["status"].(map[string]any)
		if status["const"] != "reloading" {
			t.Fatalf("semantic_reload status schema = %#v, want reloading", status)
		}
		return
	}
	t.Fatal("live-reload tools/list omitted semantic_reload")
}

func nativeResultSchema(t *testing.T, raw any) map[string]any {
	t.Helper()
	schema := raw.(map[string]any)
	for _, rawBranch := range schema["oneOf"].([]any) {
		branch := rawBranch.(map[string]any)
		required, _ := branch["required"].([]any)
		not, _ := branch["not"].(map[string]any)
		forbidden, _ := not["required"].([]any)
		hasResult := false
		for _, field := range required {
			hasResult = hasResult || field == "result"
		}
		forbidsError := false
		for _, field := range forbidden {
			forbidsError = forbidsError || field == "error"
		}
		if hasResult && forbidsError {
			return branch["properties"].(map[string]any)["result"].(map[string]any)
		}
	}
	t.Fatalf("outputSchema omitted native result branch: %#v", schema["oneOf"])
	return nil
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
	properties := schema["properties"].(map[string]any)
	assertStandardOutputProperties(t, properties)
	if resultType != "" && nativeResultSchema(t, raw)["type"] != resultType {
		t.Fatalf("native outputSchema result = %#v, want type %s", nativeResultSchema(t, raw), resultType)
	}
	assertDisjointStandardOutputBranches(t, schema["oneOf"])
}

func assertStandardOutputProperties(t *testing.T, properties map[string]any) {
	t.Helper()
	if got := properties["application_state"].(map[string]any)["type"]; got != "string" {
		t.Fatalf("application_state schema = %#v, want string", properties["application_state"])
	}
	if got := properties["error"].(map[string]any)["type"]; got != "string" {
		t.Fatalf("error schema = %#v, want string", properties["error"])
	}
	if _, ok := properties["metrics"]; ok {
		t.Fatalf("standard outputSchema must not include per-call metrics: %#v", properties["metrics"])
	}
	if _, ok := properties["session_metrics"]; ok {
		t.Fatalf("standard outputSchema must not include session metrics: %#v", properties["session_metrics"])
	}
}

func assertDisjointStandardOutputBranches(t *testing.T, raw any) {
	t.Helper()
	branches, ok := raw.([]any)
	if !ok || len(branches) < 2 {
		t.Fatalf("outputSchema oneOf = %#v, want native and generic error branches", raw)
	}
	foundNative := false
	foundGeneric := false
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		required, _ := branch["required"].([]any)
		not, _ := branch["not"].(map[string]any)
		forbidden, _ := not["required"].([]any)
		if schemaIncludes(required, "result") && schemaIncludes(forbidden, "error") {
			foundNative = true
		}
		if schemaIncludes(required, "error") && schemaIncludes(required, "application_state") && schemaIncludes(forbidden, "result") {
			foundGeneric = true
		}
	}
	if !foundNative || !foundGeneric {
		t.Fatalf("outputSchema oneOf omitted disjoint native or generic error branches: %#v", branches)
	}
}

func schemaIncludes(fields []any, field string) bool {
	for _, value := range fields {
		if value == field {
			return true
		}
	}
	return false
}

func assertTypedOutcomeOrLegacyStringSchema(t *testing.T, name string, raw any) {
	t.Helper()
	schema, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("%s outputSchema = %#v, want object", name, raw)
	}
	result := nativeResultSchema(t, schema)
	entry, ok := operation.DefaultRegistry().LookupMCP(name)
	if !ok || entry.Output == nil || entry.Output.ResultSchema == nil {
		assertStandardOutputSchema(t, raw, "string")
		return
	}
	assertResultSchemaMatchesRegistration(t, name, result, entry)
	want, err := entry.Output.ResultSchema.JSONSchema()
	if err != nil {
		t.Fatalf("%s registered result schema: %v", name, err)
	}
	wantType, _ := want["type"].(string)
	assertStandardOutputSchema(t, raw, wantType)
}

func assertResultSchemaMatchesRegistration(t *testing.T, name string, result map[string]any, entry operation.Entry) {
	t.Helper()
	want, err := entry.Output.ResultSchema.JSONSchema()
	if err != nil {
		t.Fatalf("%s registered result schema: %v", name, err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("%s encode registered result schema: %v", name, err)
	}
	var normalizedWant map[string]any
	if err := json.Unmarshal(wantJSON, &normalizedWant); err != nil {
		t.Fatalf("%s normalize registered result schema: %v", name, err)
	}
	if !reflect.DeepEqual(result, normalizedWant) {
		t.Fatalf("%s advertised native result schema = %#v, want registered schema %#v", name, result, normalizedWant)
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
	assertBatchItemOutcomeBranches(t, properties["results"].(map[string]any)["items"].(map[string]any))
	assertBatchAggregateOutcomeBranches(t, result)
	if diff, diffOK := properties["final_diff"].(map[string]any); !diffOK || diff["type"] != "string" {
		t.Fatalf("batch final_diff schema = %#v", properties["final_diff"])
	}
	delta, ok := properties["diagnostic_delta"].(map[string]any)
	if !ok {
		t.Fatal("batch result omitted diagnostic_delta schema")
	}
	if got := delta["required"]; !reflect.DeepEqual(got, []any{"net_delta", "introduced", "resolved"}) {
		t.Fatalf("diagnostic_delta required = %#v", got)
	}
	if note, ok := properties["diagnostic_delta_note"].(map[string]any); !ok || note["type"] != "string" {
		t.Fatalf("diagnostic_delta_note schema = %#v, want string", properties["diagnostic_delta_note"])
	}
}

func assertBatchItemOutcomeBranches(t *testing.T, items map[string]any) {
	t.Helper()
	branches, ok := items["oneOf"].([]any)
	if !ok || len(branches) == 0 {
		t.Fatalf("batch result item schema = %#v, want registered tool branches", items)
	}
	for _, rawBranch := range branches {
		assertBatchItemOutcomeBranch(t, rawBranch.(map[string]any))
	}
}

func assertBatchItemOutcomeBranch(t *testing.T, branch map[string]any) {
	t.Helper()
	if branch["additionalProperties"] != false {
		t.Fatalf("batch result branch is not closed: %#v", branch)
	}
	properties := branch["properties"].(map[string]any)
	status := properties["status"].(map[string]any)["const"]
	required := branch["required"].([]any)
	not, _ := branch["not"].(map[string]any)
	forbidden, _ := not["required"].([]any)
	if !schemaIncludes(required, "tool") || !schemaIncludes(required, "status") {
		t.Fatalf("batch result branch required = %#v, want tool and status", required)
	}
	if status == "ok" && !schemaIncludes(forbidden, "error") {
		t.Fatalf("successful batch branch permits error: %#v", branch)
	}
	if status == "error" && (!schemaIncludes(required, "error") || !schemaIncludes(required, "application_state")) {
		t.Fatalf("error batch branch omits error envelope: %#v", branch)
	}
	if status != "ok" && status != "error" {
		t.Fatalf("batch result branch status = %#v, want ok or error", status)
	}
	if _, hasResult := properties["result"]; status == "error" && !hasResult && !schemaIncludes(forbidden, "result") {
		t.Fatalf("generic error batch branch permits undeclared result: %#v", branch)
	}
}

func assertBatchAggregateOutcomeBranches(t *testing.T, result map[string]any) {
	t.Helper()
	branches, ok := result["oneOf"].([]any)
	if !ok || len(branches) != 2 {
		t.Fatalf("batch aggregate outcomes = %#v, want ok and error branches", result["oneOf"])
	}
	foundSuccess, foundError := false, false
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		switch properties["status"].(map[string]any)["const"] {
		case "ok":
			foundSuccess = true
			status := properties["results"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["status"].(map[string]any)
			if status["const"] != "ok" {
				t.Fatalf("successful aggregate branch permits an error row: %#v", branch)
			}
		case "error":
			foundError = true
			if _, exists := properties["results"].(map[string]any)["contains"]; !exists {
				t.Fatalf("error aggregate branch does not require an error row: %#v", branch)
			}
			not := branch["not"].(map[string]any)
			if !reflect.DeepEqual(not["required"], []any{"diagnostic_delta"}) {
				t.Fatalf("error aggregate branch permits diagnostic_delta: %#v", branch)
			}
		}
	}
	if !foundSuccess || !foundError {
		t.Fatalf("batch aggregate schema omitted successful or error branch: %#v", branches)
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
	opts = append(opts, mcp.WithToolProfile("full"))
	if registry != nil {
		opts = append(opts, mcp.WithRegistry(registry))
	}
	if err := mcptest.NewServer(t, profile, root, &out, opts...).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
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
	writeTestGoModule(t, root)
	for name, source := range map[string]string{"main.go": "package sample\n", "Main.java": "class Main {}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func writeTestGoModule(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/mcpfixture\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("write fixture module: %v", err)
	}
}

func listToolsWithLiveReload(t *testing.T) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := mcptest.NewServer(t, "full", schemaWorkspace(t), &out, mcp.WithLiveReload(true)).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
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

func schemaEnumContains(values []any, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func assertBatchSchemaProfile(t *testing.T, profile string) {
	tools := listTools(t, profile)
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	assertBatchOutputSchema(t, batch["outputSchema"])
	description := batch["description"].(string)
	for _, phrase := range []string{"sequentially against the working state", "earlier edits", "diagnostic_delta", "final_diff"} {
		if !strings.Contains(description, phrase) {
			t.Errorf("batch description %q omitted %q", description, phrase)
		}
	}
	input := batch["inputSchema"].(map[string]any)
	edits := input["properties"].(map[string]any)["edits"].(map[string]any)
	if edits["minItems"] != float64(1) {
		t.Fatalf("edits.minItems = %#v, want 1", edits["minItems"])
	}
	item := edits["items"].(map[string]any)
	properties := item["properties"].(map[string]any)
	toolSchema := properties["tool"].(map[string]any)
	expected := expectedBatchSchemaEntries(profile)
	expectedNames := make([]string, 0, len(expected.ByName))
	for name := range expected.ByName {
		expectedNames = append(expectedNames, name)
	}
	slices.Sort(expectedNames)
	if !reflect.DeepEqual(toolSchema["enum"], toAnySlice(expectedNames)) {
		t.Fatalf("batch tool enum = %#v, want registry batchable names %#v", toolSchema["enum"], expectedNames)
	}
	if item["required"] == nil || !reflect.DeepEqual(item["required"], []any{"tool"}) {
		t.Fatalf("batch item required = %#v, want [tool]", item["required"])
	}
	if item["additionalProperties"] != true || properties["params"].(map[string]any)["type"] != "object" {
		t.Fatalf("batch item schema is not compact and tolerant: %#v", item)
	}
	if _, exists := item["oneOf"]; exists {
		t.Fatalf("batch item schema retains duplicated oneOf schemas: %#v", item)
	}
}

type batchSchemaExpectations struct {
	ByName map[string]operation.Entry
	Names  []string
}

func expectedBatchSchemaEntries(profile string) batchSchemaExpectations {
	expected := batchSchemaExpectations{ByName: make(map[string]operation.Entry)}
	for _, entry := range operation.DefaultRegistry().All() {
		if entry.MCPName == "" || !entry.Batchable || (profile == "mutations-only" && entry.ReadOnly) {
			continue
		}
		expected.ByName[entry.MCPName] = entry
		expected.Names = append(expected.Names, entry.MCPName)
	}
	return expected
}

func TestConstructReplacementSchemasAreExposedAndBatchable(t *testing.T) {
	t.Parallel()
	tools := listTools(t, "full")
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	assertConstructReplacementSchema(t, byName, "semantic_replace_construct", []string{"file", "kind", "source"}, []string{"file", "in_function", "kind", "discriminator", "construct_path", "source"})
	assertConstructReplacementSchema(t, byName, "semantic_replace_decl", []string{"file", "symbol", "source"}, []string{"file", "symbol", "source"})
	assertConstructKindSchema(t, byName)
	assertConstructInsertionSchema(t, byName)
	assertConstructBatchSchema(t, byName)
}

func assertConstructReplacementSchema(t *testing.T, byName map[string]map[string]any, name string, required, stringProperties []string) {
	t.Helper()
	tool, ok := byName[name]
	if !ok {
		t.Fatalf("tools/list omitted %s", name)
	}
	input := tool["inputSchema"].(map[string]any)
	properties := input["properties"].(map[string]any)
	for _, propertyName := range stringProperties {
		property, ok := properties[propertyName].(map[string]any)
		if !ok || property["type"] != "string" {
			t.Errorf("%s property schema = %#v, want string", propertyName, properties[propertyName])
		}
	}
	if got := input["required"]; !reflect.DeepEqual(got, toAnySlice(required)) {
		t.Errorf("required = %#v, want %v", got, required)
	}
	description, _ := tool["description"].(string)
	if !strings.Contains(description, "Use this tool instead of") ||
		(!strings.Contains(description, "semantic_replace_body") && !strings.Contains(description, "replace_file_content")) {
		t.Errorf("description does not advertise the semantic-edit trigger: %q", description)
	}
}

func assertConstructKindSchema(t *testing.T, byName map[string]map[string]any) {
	t.Helper()
	construct := byName["semantic_replace_construct"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["kind"].(map[string]any)
	if got, want := construct["enum"], toAnySlice([]string{"loop", "if", "else", "case", "select", "defer", "function", "method", "type", "const", "var", "statement/expression"}); !reflect.DeepEqual(got, want) {
		t.Errorf("Go construct enum = %#v, want %#v", got, want)
	}
}

func assertConstructInsertionSchema(t *testing.T, byName map[string]map[string]any) {
	t.Helper()
	insertConstruct := byName["semantic_insert_construct"]
	if insertConstruct == nil {
		t.Fatal("tools/list omitted semantic_insert_construct")
	}
	insertProperties := insertConstruct["inputSchema"].(map[string]any)["properties"].(map[string]any)
	if _, exists := insertProperties["overwrite"]; exists {
		t.Fatal("semantic_insert_construct retained retired overwrite parameter")
	}
	if _, exists := byName["semantic_insert_decl"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["overwrite"]; exists {
		t.Fatal("semantic_insert_decl retained retired overwrite parameter")
	}
}

func assertConstructBatchSchema(t *testing.T, byName map[string]map[string]any) {
	t.Helper()
	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	description, ok := batch["description"].(string)
	if !ok || !strings.Contains(description, "sequentially against the working state") || !strings.Contains(description, "inspect final targets after the batch") {
		t.Fatalf("batch description omitted sequential and final-state semantics: %#v", batch["description"])
	}
	items := batch["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	toolSchema := items["properties"].(map[string]any)["tool"].(map[string]any)
	names, ok := toolSchema["enum"].([]any)
	if !ok {
		t.Fatalf("batch tool schema enum = %#v", toolSchema["enum"])
	}
	for _, name := range []string{"semantic_replace_construct", "semantic_replace_decl", "semantic_insert_construct"} {
		if !schemaEnumContains(names, name) {
			t.Errorf("semantic_batch schema omitted %s", name)
		}
	}
}

func TestMCPEnabledLanguagesFilterMixedAndSourceLessWorkspaces(t *testing.T) {
	t.Parallel()
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
	if err := mcptest.NewServer(t, "full", empty, &bytes.Buffer{}, mcp.WithEnabledLanguages("go,unknown")).Serve(context.Background(), strings.NewReader("")); err == nil {
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

func (b customConstructBackend) SupportedConstructs(operation backend.Operation) []backend.ConstructKind {
	if operation != backend.OperationReplaceConstruct {
		return nil
	}
	return append([]backend.ConstructKind(nil), b.kinds...)
}

func TestConstructSchemaUsesRegisteredBackendCapabilities(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/customconstruct\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	custom := customConstructBackend{Backend: gobackend.NewGoBackend(), kinds: []backend.ConstructKind{"custom_kind", backend.ConstructLoop}}
	registry, err := backend.NewRegistry(custom)
	if err != nil {
		t.Fatal(err)
	}
	tools := listToolsAt(t, root, mcp.WithService(backend.NewService(registry)), mcp.WithEnabledLanguages("go"))
	byName := namedMCPTools(tools)
	replace := byName["semantic_replace_construct"]
	if replace == nil {
		t.Fatal("tools/list omitted semantic_replace_construct")
	}
	kind := replace["inputSchema"].(map[string]any)["properties"].(map[string]any)["kind"].(map[string]any)
	if got, want := kind["enum"], toAnySlice([]string{"custom_kind", "loop"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("construct kind enum = %#v, want registered backend values %#v", got, want)
	}
	if byName["semantic_insert_construct"] != nil {
		t.Fatal("tools/list advertised insertion without a structure capability provider")
	}
	batchItems := byName["semantic_batch"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	batchNames := batchItems["properties"].(map[string]any)["tool"].(map[string]any)["enum"].([]any)
	if !schemaEnumContains(batchNames, "semantic_replace_construct") || schemaEnumContains(batchNames, "semantic_insert_construct") {
		t.Fatalf("batch tool enum = %#v, want replace only", batchNames)
	}
	access := byName["semantic_insert_function"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["access_modifier"].(map[string]any)["enum"]
	if got, want := access, toAnySlice([]string{"infer", "public", "private"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Go insertion access enum = %#v, want %#v", got, want)
	}
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
	allOptions := append([]mcp.Option{mcp.WithToolProfile("full")}, options...)
	if err := mcptest.NewServer(t, "full", root, &out, allOptions...).Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
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
	t.Parallel()
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
	t.Parallel()
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
	t.Run("direct symbol targeting", func(t *testing.T) {
		assertDirectSymbolTargeting(t, byName)
	})
	assertContains("semantic_insert_declaration.source", paramDescription("semantic_insert_declaration", "source"), "Prefer semantic_insert_function", "semantic_insert_type", "semantic_insert_decl")
	assertContains("semantic_insert_function description", toolDescription("semantic_insert_function"), "one complete Go function or method", "parameters, results, and body")
	assertContains("semantic_insert_type description", toolDescription("semantic_insert_type"), "one Go struct, interface, or type alias", "instead of replace_file_content")
	assertContains("semantic_insert_decl description", toolDescription("semantic_insert_decl"), "one Go constant or variable", "instead of replace_file_content")
	assertContains("semantic_insert_declaration description", toolDescription("semantic_insert_declaration"), "generic placement controls", "Prefer semantic_insert_function", "semantic_insert_type", "semantic_insert_decl")
	assertContains("semantic_insert_construct description", toolDescription("semantic_insert_construct"), "construct", "semantic_replace_construct")
	assertContains("semantic_batch description", toolDescription("semantic_batch"), `{"tool":"semantic_tool_name","params":{...}}`, "flat form", "single-key", "sequentially against the working state", "zero-based edit index")
	assertContains("semantic_lookup description", toolDescription("semantic_lookup"), "Returns symbol, file, kind, line, and column", "file-navigation reads", "not accepted as semedit tool parameters")
	assertContains("semantic_insert_case description", toolDescription("semantic_insert_case"), "exact discriminator expression", "kind=case alone does not identify", "construct_path")
	assertContains("semantic_insert_case.discriminator", paramDescription("semantic_insert_case", "discriminator"), "Selector: switch discriminant expression", "tagless switch", "Copy discriminator from")
}

func TestGeneratedSemanticInputSchemasRejectUnknownProperties(t *testing.T) {
	t.Parallel()
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
	if verifySchema["additionalProperties"] != false || batchSchema["additionalProperties"] != false {
		t.Fatal("standalone and batch request schemas must reject unknown top-level properties")
	}
	items := batchSchema["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	itemProperties := items["properties"].(map[string]any)
	if items["additionalProperties"] != true || !reflect.DeepEqual(items["required"], []any{"tool"}) {
		t.Fatalf("batch item schema should tolerate flat params and require only tool: %#v", items)
	}
	if itemProperties["params"].(map[string]any)["type"] != "object" {
		t.Fatalf("batch params schema = %#v, want generic object", itemProperties["params"])
	}
	if _, exists := items["oneOf"]; exists {
		t.Fatal("batch item schema contains duplicated per-tool schemas")
	}
	names := itemProperties["tool"].(map[string]any)["enum"].([]any)
	if !schemaEnumContains(names, "semantic_replace_body") {
		t.Fatal("batch tool enum omitted semantic_replace_body")
	}
	access := insertFunctionSchema["properties"].(map[string]any)["access_modifier"].(map[string]any)
	if !reflect.DeepEqual(access["enum"], []any{"infer", "public", "private"}) {
		t.Fatalf("Go insert-function access enum = %#v, want only infer/public/private", access["enum"])
	}
}

func TestScaffoldPurposeHeaderSchemaIsOptional(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	tools := make(map[string]map[string]any)
	for _, tool := range listTools(t, "full") {
		tools[tool["name"].(string)] = tool
	}
	fileSources := []string{
		"semantic_lookup", "semantic_rename", "semantic_replace_body", "semantic_replace_construct",
		"semantic_insert_case", "semantic_insert_function", "semantic_insert_type", "semantic_insert_decl",
		"semantic_insert_declaration", "semantic_insert_construct", "semantic_replace_decl",
		"semantic_extract_function", "semantic_extract_constant", "semantic_extract_variable", "semantic_inline_variable", "semantic_organize_imports", "semantic_scaffold_file",
	}
	inFunctionSources := []string{
		"semantic_replace_construct", "semantic_insert_case", "semantic_insert_construct",
	}
	discriminatorSources := []string{"semantic_insert_case", "semantic_insert_construct"}
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
	assertSources("semantic_insert_decl", "file", []string{"Receipt file from semantic_extract_function"})
	for _, toolName := range []string{"semantic_extract_function", "semantic_extract_constant", "semantic_extract_variable"} {
		entry, ok := operation.DefaultRegistry().LookupMCP(toolName)
		if !ok {
			t.Fatalf("default registry omitted %s", toolName)
		}
		for _, param := range entry.Params {
			if param.JSONName == "symbol" && len(param.SourceFields) != 0 {
				t.Errorf("%s.symbol creation name declares source fields: %v", toolName, param.SourceFields)
			}
		}
		assertSources(toolName, "in_function", inFunctionSources)
	}
	for _, tool := range inFunctionSources {
		assertSources(tool, "in_function", inFunctionSources)
		assertSources(tool, "discriminator", discriminatorSources)
	}
}

func TestMoveConstructIsAdvertisedInBatchWithDestinationReceipt(t *testing.T) {
	t.Parallel()
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
	moveOutput := move["outputSchema"].(map[string]any)
	envelope := moveOutput["properties"].(map[string]any)
	for _, name := range []string{"file", "symbol"} {
		if property, ok := envelope[name].(map[string]any); !ok || property["type"] != "string" {
			t.Errorf("standalone move continuation %s = %#v, want string", name, envelope[name])
		}
	}
	if _, exists := envelope["origin_file"]; exists {
		t.Error("standalone move exposed historical origin_file as a live continuation")
	}
	resultProperties := nativeResultSchema(t, moveOutput)["properties"].(map[string]any)
	for _, name := range []string{"file", "symbol", "origin_file"} {
		if property, ok := resultProperties[name].(map[string]any); !ok || property["type"] != "string" {
			t.Errorf("standalone move typed result property %s = %#v, want string", name, resultProperties[name])
		}
	}
	batch := byName["semantic_batch"]
	if batch == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	assertMoveBatchSchema(t, batch)
}

func assertMoveBatchSchema(t *testing.T, batch map[string]any) {
	t.Helper()
	items := batch["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	names := items["properties"].(map[string]any)["tool"].(map[string]any)["enum"].([]any)
	if !schemaEnumContains(names, "semantic_move_construct") {
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
		file := properties["file"].(map[string]any)
		origin := properties["origin_file"].(map[string]any)
		destination := properties["destination_file"].(map[string]any)
		if file["type"] != "string" || !strings.Contains(file["description"].(string), "Captured destination path on success") || !strings.Contains(file["description"].(string), "attempted source path on error") || !strings.Contains(file["description"].(string), "intermediate path rather than the final workspace target") {
			t.Fatalf("file result schema = %#v, want captured intermediate evidence", file)
		}
		if origin["type"] != "string" || !strings.Contains(origin["description"].(string), "evidence only") {
			t.Fatalf("origin_file result schema = %#v, want historical evidence", origin)
		}
		if destination["type"] != "string" || !strings.Contains(destination["description"].(string), "intermediate result rather than the final workspace target") {
			t.Fatalf("destination_file result schema = %#v, want destination evidence", destination)
		}
		return
	}
	t.Fatal("batch output schema omitted semantic_move_construct result branch")
}

func TestMCPStaleContinuationsAreNotAdvertised(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	root := schemaWorkspace(t)
	registry := operation.NewRegistry()
	operations := registerTypedSchemaOperations(t, registry)
	tools := listToolsAtWithRegistry(t, "full", root, registry)
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	assertTypedAdvertisedSchemas(t, byName, operations)
	var projectedOutput bytes.Buffer
	projectedRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_typed_schema_provider","arguments":{}}}`
	if err := mcptest.NewServer(t, "full", root, &projectedOutput, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(projectedRequest+"\n")); err != nil {
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
	consumerRequest, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": operations.Consumer.MCPName, "arguments": map[string]any{"payload": map[string]any{"text": "copied", "rows": []any{0}}, "file": projectedTarget["file"], "symbol": projectedTarget["symbol"]}}})
	if err != nil {
		t.Fatalf("encode nested continuation consumer call: %v", err)
	}
	var consumerOutput bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &consumerOutput, mcp.WithRegistry(registry)).Serve(context.Background(), bytes.NewReader(append(consumerRequest, '\n'))); err != nil {
		t.Fatalf("serve nested continuation consumer: %v", err)
	}
	if strings.Contains(consumerOutput.String(), "isError\":true") || !strings.Contains(consumerOutput.String(), "accepted") {
		t.Fatalf("copied nested continuation was rejected: %s", consumerOutput.String())
	}
	var malformedOutput bytes.Buffer
	malformedRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_typed_schema_provider","arguments":{"invalid_output":true}}}`
	if err := mcptest.NewServer(t, "full", root, &malformedOutput, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(malformedRequest+"\n")); err != nil {
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
		if err := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(request+"\n")); err != nil {
			t.Fatalf("serve typed request: %v", err)
		}
		if !strings.Contains(output.String(), "unknown property \\\"unexpected\\\"") {
			t.Errorf("typed ingress did not reject unknown nested field: %s", output.String())
		}
	}
}

func assertTypedAdvertisedSchemas(t *testing.T, byName map[string]map[string]any, operations typedSchemaOperations) {
	t.Helper()
	providerOutput := byName[operations.Provider.MCPName]["outputSchema"].(map[string]any)
	providerProperties := providerOutput["properties"].(map[string]any)
	resultSchema := providerProperties["result"].(map[string]any)
	targetShape := resultSchema["properties"].(map[string]any)["target"].(map[string]any)
	if targetShape["additionalProperties"] != false || targetShape["type"] != "object" {
		t.Fatalf("typed nested target schema = %#v", targetShape)
	}
	consumerInput := byName[operations.Consumer.MCPName]["inputSchema"].(map[string]any)
	payload := consumerInput["properties"].(map[string]any)["payload"].(map[string]any)
	if payload["additionalProperties"] != false {
		t.Fatalf("consumer payload schema = %#v", payload)
	}
	assertTypedBatchPayloadSchema(t, byName, operations.Consumer.MCPName)
}

func assertTypedBatchPayloadSchema(t *testing.T, byName map[string]map[string]any, consumerName string) {
	t.Helper()
	batchInput := byName["semantic_batch"]["inputSchema"].(map[string]any)
	items := batchInput["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	properties := items["properties"].(map[string]any)
	names := properties["tool"].(map[string]any)["enum"].([]any)
	if !schemaEnumContains(names, consumerName) {
		t.Fatalf("compact batch schema omitted typed consumer %s", consumerName)
	}
	params := properties["params"].(map[string]any)
	if params["type"] != "object" || items["additionalProperties"] != true {
		t.Fatalf("batch item schema does not accept typed args and flat siblings: %#v", items)
	}
	if _, exists := params["properties"]; exists {
		t.Fatalf("batch params schema duplicated operation property schemas: %#v", params)
	}
}

type typedSchemaOperations struct {
	Provider operation.Def[map[string]any, string]
	Consumer operation.Def[map[string]any, string]
}

func registerTypedSchemaOperations(t *testing.T, registry *operation.Registry) typedSchemaOperations {
	t.Helper()
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
	return typedSchemaOperations{Provider: provider, Consumer: consumer}
}

func TestBatchToolAdvertisesAndEmitsRegisteredResultShapes(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	registry := operation.NewRegistry()
	registerBatchResultShapeEntries(t, registry)
	assertBatchResultShapeSchemas(t, root, registry)
	assertBatchResultShapeExecution(t, root, registry)
}

func registerBatchResultShapeEntries(t *testing.T, registry *operation.Registry) {
	t.Helper()
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
}

func assertBatchResultShapeSchemas(t *testing.T, root string, registry *operation.Registry) {
	t.Helper()
	tools := listToolsAtWithRegistry(t, "full", root, registry)
	batchTool := namedMCPTools(tools)["semantic_batch"]
	if batchTool == nil {
		t.Fatal("tools/list omitted semantic_batch")
	}
	resultsSchema := batchTool["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)["properties"].(map[string]any)["results"].(map[string]any)["items"].(map[string]any)
	branches := resultsSchema["oneOf"].([]any)
	for name, wantKind := range batchResultShapeKinds() {
		assertBatchResultBranch(t, branches, name, wantKind)
	}
	assertLegacyBatchBranchOmitsResult(t, branches)
}

func batchResultShapeKinds() map[string]string {
	return map[string]string{
		"semantic_typed_object": "object",
		"semantic_typed_scalar": "string",
		"semantic_typed_array":  "array",
		"semantic_typed_null":   "nullable",
	}
}

func assertBatchResultBranch(t *testing.T, branches []any, name, wantKind string) {
	t.Helper()
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		tool, ok := properties["tool"].(map[string]any)["const"].(string)
		if !ok || tool != name || properties["status"].(map[string]any)["const"] != "ok" {
			continue
		}
		resultSchema, ok := properties["result"].(map[string]any)
		if !ok {
			t.Fatalf("batch schema omitted result for %s: %#v", name, branch)
		}
		assertBatchResultSchemaKind(t, name, resultSchema, wantKind)
		return
	}
	t.Fatalf("batch output schema omitted registered successful result branch %s", name)
}

func assertBatchResultSchemaKind(t *testing.T, name string, resultSchema map[string]any, wantKind string) {
	t.Helper()
	if wantKind == "nullable" {
		types, ok := resultSchema["type"].([]any)
		if !ok || !reflect.DeepEqual(types, []any{"string", "null"}) {
			t.Fatalf("nullable result schema for %s = %#v", name, resultSchema)
		}
		return
	}
	if resultSchema["type"] != wantKind {
		t.Fatalf("batch result schema for %s = %#v, want type %s", name, resultSchema, wantKind)
	}
}

func assertLegacyBatchBranchOmitsResult(t *testing.T, branches []any) {
	t.Helper()
	for _, rawBranch := range branches {
		branch := rawBranch.(map[string]any)
		properties := branch["properties"].(map[string]any)
		tool, ok := properties["tool"].(map[string]any)["const"].(string)
		if !ok || tool != "semantic_legacy_output" || properties["status"].(map[string]any)["const"] != "ok" {
			continue
		}
		if _, hasResult := properties["result"]; hasResult {
			t.Fatalf("legacy batch branch advertises a result: %#v", branch)
		}
		return
	}
	t.Fatal("legacy successful batch schema branch is missing")
}

func assertBatchResultShapeExecution(t *testing.T, root string, registry *operation.Registry) {
	t.Helper()
	var edits = make([]map[string]any, 0, 5)
	for _, name := range []string{"semantic_typed_object", "semantic_typed_scalar", "semantic_typed_array", "semantic_typed_null", "semantic_legacy_output"} {
		edits = append(edits, map[string]any{"tool": name, "params": map[string]any{}})
	}
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "semantic_batch", "arguments": map[string]any{"edits": edits}}})
	if err != nil {
		t.Fatalf("encode batch request: %v", err)
	}
	var output bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), bytes.NewReader(append(request, '\n'))); err != nil {
		t.Fatalf("serve batch typed results: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode batch results: %v", err)
	}
	assertTypedBatchResults(t, response)
}

func assertTypedBatchResults(t *testing.T, response map[string]any) {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("batch call returned no result object: %#v", response)
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("batch call returned no structured content: %#v", response)
	}
	batchResult, ok := structured["result"].(map[string]any)
	if !ok {
		t.Fatalf("batch call returned no structured batch result: %#v", response)
	}
	results, ok := batchResult["results"].([]any)
	if !ok {
		t.Fatalf("batch call returned no results array: %#v", response)
	}
	assertOrderedTypedBatchRows(t, results)
}

func assertOrderedTypedBatchRows(t *testing.T, results []any) {
	t.Helper()
	operationNames := []string{
		"semantic_typed_object",
		"semantic_typed_scalar",
		"semantic_typed_array",
		"semantic_typed_null",
		"semantic_legacy_output",
	}
	if len(results) != len(operationNames)+1 {
		t.Fatalf("batch results contain %d rows, want %d requested operations plus verification: %#v", len(results), len(operationNames)+1, results)
	}
	expected := map[string]any{
		"semantic_typed_object": map[string]any{"value": float64(7)},
		"semantic_typed_scalar": "scalar",
		"semantic_typed_array":  []any{"array"},
		"semantic_typed_null":   nil,
	}
	for index, name := range operationNames {
		result := typedBatchResultRow(t, results[index], index)
		if result["tool"] != name {
			t.Fatalf("batch row %d names tool %#v, want %q", index, result["tool"], name)
		}
		if result["status"] != "ok" {
			t.Fatalf("batch row %q status = %#v, want ok", name, result["status"])
		}
		if name == "semantic_legacy_output" {
			assertLegacyTypedBatchResult(t, result)
			continue
		}
		assertTypedBatchResult(t, result, name, expected[name])
	}
	assertMissingVerificationBatchResult(t, typedBatchResultRow(t, results[len(operationNames)], len(operationNames)))
}

func typedBatchResultRow(t *testing.T, value any, index int) map[string]any {
	t.Helper()
	row, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("batch result row %d has type %T, want object", index, value)
	}
	return row
}

func assertTypedBatchResult(t *testing.T, result map[string]any, name string, want any) {
	t.Helper()
	got, present := result["result"]
	if !present || !reflect.DeepEqual(got, want) {
		t.Fatalf("batch result %s = %#v present=%t, want %#v", name, got, present, want)
	}
}

func assertLegacyTypedBatchResult(t *testing.T, result map[string]any) {
	t.Helper()
	if _, present := result["result"]; present {
		t.Fatalf("legacy batch result unexpectedly has result: %#v", result)
	}
}

func assertMissingVerificationBatchResult(t *testing.T, result map[string]any) {
	t.Helper()
	if result["tool"] != "semantic_verify" || result["status"] != "error" || result["error"] != "verification operation is not registered" {
		t.Fatalf("synthetic verification row = %#v, want explicit missing-registration error", result)
	}
}

func TestInspectProjectionFeedsGuardedReplaceBody(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	module := []byte("module example.com/inspectprojection\n\ngo 1.24\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), module, 0o600); err != nil {
		t.Fatalf("write inspection fixture module: %v", err)
	}
	path := filepath.Join(root, "main.go")
	initial := []byte("package sample\nfunc Value() int { return 1 }\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write inspection fixture: %v", err)
	}
	var inspectOutput bytes.Buffer
	inspectServer := mcptest.NewServer(t, "full", root, &inspectOutput)
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
	if got := result["content"].([]any)[0].(map[string]any)["text"].(string); !strings.HasPrefix(got, "Critical result (JSON): ") || !strings.HasSuffix(got, inspection) {
		t.Fatalf("inspection text must prefix critical result and retain formatter body: %#v", got)
	}
	record := assertInspectionContinuation(t, structured)
	arguments := guardedReplaceArguments(record)
	guardedOutput := executeGuardedReplace(t, root, arguments, 2)
	if strings.Contains(guardedOutput, "isError\":true") {
		t.Fatalf("guarded edit rejected matching captured evidence: %s", guardedOutput)
	}
	staleOutput := executeGuardedReplace(t, root, arguments, 3)
	if !strings.Contains(staleOutput, "expected_revision does not match captured source") || !strings.Contains(staleOutput, "isError\":true") {
		t.Fatalf("stale guarded edit did not refuse explicitly: %s", staleOutput)
	}
	assertGuardedReplaceSource(t, path)
}

func assertInspectionContinuation(t *testing.T, structured map[string]any) map[string]any {
	t.Helper()
	records := structured["continuations"].([]any)
	if len(records) != 1 {
		t.Fatalf("inspection continuations = %#v", records)
	}
	record := records[0].(map[string]any)
	if record["file"] != "main.go" || record["symbol"] != "Value" || record["expected_revision"] == "" || record["in_function"] != "Value" {
		t.Fatalf("inspection target projection = %#v", record)
	}
	return record
}

func guardedReplaceArguments(record map[string]any) map[string]any {
	return map[string]any{
		"file":              record["file"],
		"symbol":            record["symbol"],
		"expected_revision": record["expected_revision"],
		"body":              "return 2",
	}
}

func executeGuardedReplace(t *testing.T, root string, arguments map[string]any, id int) string {
	t.Helper()
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": "semantic_replace_body", "arguments": arguments}})
	if err != nil {
		t.Fatalf("encode guarded edit: %v", err)
	}
	var output bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &output).Serve(context.Background(), bytes.NewReader(append(request, '\n'))); err != nil {
		t.Fatalf("serve guarded edit: %v", err)
	}
	return output.String()
}

func assertGuardedReplaceSource(t *testing.T, path string) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read guarded edit result: %v", err)
	}
	if !strings.Contains(string(actual), "return 2") || strings.Contains(string(actual), "return 1") {
		t.Fatalf("stale request changed the successful edit: %s", actual)
	}
}

func TestInspectProjectionFeedsInsertField(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	path := filepath.Join(root, "config.go")
	initial := []byte("package sample\ntype options struct{}\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write field inspection fixture: %v", err)
	}
	inspectRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_inspect_symbol","arguments":{"symbol":"options","file":"config.go","language":"go","trust_workspace":true}}}` + "\n"
	var inspectOutput bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &inspectOutput).Serve(context.Background(), strings.NewReader(inspectRequest)); err != nil {
		t.Fatalf("serve field inspection: %v", err)
	}
	var inspected map[string]any
	if err := json.Unmarshal(inspectOutput.Bytes(), &inspected); err != nil {
		t.Fatalf("decode field inspection: %v", err)
	}
	structured := inspected["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	records := structured["continuations"].([]any)
	if len(records) != 1 {
		t.Fatalf("field inspection continuations = %#v", records)
	}
	record := records[0].(map[string]any)
	if record["file"] != "config.go" || record["in_type"] != "options" || record["expected_revision"] == "" {
		t.Fatalf("inspection did not project same-record field evidence: %#v", record)
	}
	arguments := map[string]any{
		"file": record["file"], "in_type": record["in_type"], "expected_revision": record["expected_revision"],
		"symbol": "options.timeout", "source": "timeout int", "placement": "type_end",
	}
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "semantic_insert_field", "arguments": arguments}})
	if err != nil {
		t.Fatalf("encode field insertion: %v", err)
	}
	var output bytes.Buffer
	if serveErr := mcptest.NewServer(t, "full", root, &output).Serve(context.Background(), bytes.NewReader(append(request, '\n'))); serveErr != nil {
		t.Fatalf("serve field insertion: %v", serveErr)
	}
	if strings.Contains(output.String(), `"isError":true`) {
		t.Fatalf("field insertion rejected matching inspect record: %s", output.String())
	}
	var inserted map[string]any
	if decodeErr := json.Unmarshal(output.Bytes(), &inserted); decodeErr != nil {
		t.Fatalf("decode field insertion: %v", decodeErr)
	}
	result := inserted["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	receipt := result["edit_receipt"].(map[string]any)
	continuations := receipt["continuations"].([]any)
	if len(continuations) != 1 {
		t.Fatalf("field receipt continuations = %#v", continuations)
	}
	target := continuations[0].(map[string]any)
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read inserted field source: %v", err)
	}
	digest := sha256.Sum256(updated)
	if target["file"] != "config.go" || target["symbol"] != "options.timeout" || target["expected_revision"] != hex.EncodeToString(digest[:]) {
		t.Fatalf("field receipt does not identify the live target revision: %#v", target)
	}
	if !strings.Contains(string(updated), "timeout int") {
		t.Fatalf("field insertion missing from source: %s", updated)
	}
}

func TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision(t *testing.T) {
	t.Parallel()
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
	provider := receiptMutationProvider(path, root, initialDigest, consumerName)
	consumer := receiptConsumer(consumerName, providerName)
	registerReceiptOperations(t, registry, provider, consumer)

	responses := serveReceiptCalls(t, root, registry)
	assertPersistedReceiptIdentity(t, responses[0])
	assertRemovedReceiptProjection(t, responses[1])
	assertReplacementReceiptProjection(t, responses[2])
}

func receiptMutationProvider(path, root string, initialDigest [32]byte, consumerName string) operation.Def[map[string]any, operation.EditReceipt] {
	providerName := "semantic_receipt_mutation"
	fileSchema := operation.SymbolTargetSchema().Properties["file"]
	symbolSchema := operation.SymbolTargetSchema().Properties["symbol"]
	return operation.Def[map[string]any, operation.EditReceipt]{
		Key: "receipt-mutation", MCPName: providerName,
		Params: []operation.ParameterContract{{Name: "identity", JSONName: "identity", Type: operation.ParamString, Required: true}, {Name: "remove", JSONName: "remove", Type: operation.ParamBoolean}, {Name: "replacement", JSONName: "replacement", Type: operation.ParamBoolean}},
		Output: &operation.OperationOutputContract{ResultSchema: func() *operation.ValueSchema { schema := operation.EditReceiptSchema(); return &schema }(), Continuations: []operation.OutputContinuation{
			{Path: "result.continuations[].file", Schema: fileSchema, Role: operation.RoleTarget, Consumers: []string{consumerName}},
			{Path: "result.continuations[].symbol", Schema: symbolSchema, Role: operation.RoleTarget, Consumers: []string{consumerName}},
		}},
		Parse:          func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers:       receiptMutationHandlers(path, initialDigest),
		FinalizeOutput: receiptFinalizer(root),
		ProjectOutput:  func(receipt operation.EditReceipt) (any, error) { return receipt, nil },
	}
}

func receiptMutationHandlers(path string, initialDigest [32]byte) map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (operation.EditReceipt, error) {
	return map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (operation.EditReceipt, error){
		backend.LanguageAuto: func(ctx context.Context, _ operation.CallContext, raw map[string]any) (operation.EditReceipt, error) {
			identity, _ := raw["identity"].(string)
			source := receiptMutationSource(identity, raw)
			if err := sourcebuf.WriteFile(ctx, path, source); err != nil {
				return operation.EditReceipt{}, err
			}
			return operation.EditReceipt{
				SchemaVersion: "semedit.edit-receipt/v1",
				Status:        operation.EditPersisted,
				Operation:     "receipt-mutation",
				ReadSet:       []operation.FileRevision{{Path: "main.go", Revision: hex.EncodeToString(initialDigest[:])}},
				Continuations: []operation.SymbolTarget{{File: "main.go", Symbol: "Widget"}},
			}, nil
		},
	}
}

func receiptMutationSource(identity string, raw map[string]any) []byte {
	if remove, _ := raw["remove"].(bool); remove {
		return []byte("package sample\nvar Other = 2\n")
	}
	if replacement, _ := raw["replacement"].(bool); replacement {
		identity = "replacement"
	}
	return []byte("package sample\n// identity:" + identity + "\nvar Widget = 2\n")
}

func receiptFinalizer(root string) func(operation.CallContext, map[string]any, operation.EditReceipt) (operation.EditReceipt, error) {
	return func(cc operation.CallContext, request map[string]any, draft operation.EditReceipt) (operation.EditReceipt, error) {
		identity, _ := request["identity"].(string)
		plannedBytes := receiptMutationSource(identity, request)
		plannedDigest := sha256.Sum256(plannedBytes)
		expectedRevision := hex.EncodeToString(plannedDigest[:])
		continuations := append([]operation.SymbolTarget(nil), draft.Continuations...)
		for index := range continuations {
			continuations[index].ExpectedRevision = &expectedRevision
		}
		draftValue := operation.EditReceiptDraft{
			Operation:      draft.Operation,
			Status:         draft.Status,
			PlannedChanges: []operation.FileRevision{{Path: "main.go", Revision: expectedRevision}},
			ReadSet:        draft.ReadSet,
			Continuations:  continuations,
		}
		return operation.FinalizeEditReceipt(cc.Ctx, cc.WorkDir, cc.InBatch, cc.InBatch, draftValue, receiptIdentityResolver(root, identity))
	}
}

func receiptIdentityResolver(root, identity string) func(context.Context, operation.SymbolTarget) (operation.SymbolTarget, bool, error) {
	return func(_ context.Context, target operation.SymbolTarget) (operation.SymbolTarget, bool, error) {
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
}

func receiptConsumer(consumerName, providerName string) operation.Def[map[string]any, string] {
	return operation.Def[map[string]any, string]{Key: "receipt-consumer", MCPName: consumerName, Batchable: true, Params: []operation.ParameterContract{
		{Name: "file", JSONName: "file", Type: operation.ParamString, Required: true, Role: operation.RoleTarget, SourceFields: []string{providerName + ".result.continuations[].file"}},
		{Name: "symbol", JSONName: "symbol", Type: operation.ParamString, Required: true, Role: operation.RoleTarget, SourceFields: []string{providerName + ".result.continuations[].symbol"}},
	}, Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil }, Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){backend.LanguageAuto: func(context.Context, operation.CallContext, map[string]any) (string, error) { return "consumed", nil }}}
}

func registerReceiptOperations(t *testing.T, registry *operation.Registry, provider operation.Def[map[string]any, operation.EditReceipt], consumer operation.Def[map[string]any, string]) {
	t.Helper()
	if err := operation.Register(registry, provider); err != nil {
		t.Fatalf("register receipt provider: %v", err)
	}
	if err := operation.Register(registry, consumer); err != nil {
		t.Fatalf("register receipt consumer: %v", err)
	}
}

func serveReceiptCalls(t *testing.T, root string, registry *operation.Registry) []map[string]any {
	t.Helper()
	requests := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_receipt_mutation","arguments":{"identity":"after"}}}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_receipt_mutation","arguments":{"identity":"removed","remove":true}}}` + "\n" + `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_receipt_mutation","arguments":{"identity":"original","replacement":true}}}` + "\n"
	var output bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(requests)); err != nil {
		t.Fatalf("serve receipt calls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("MCP responses = %d: %s", len(lines), output.String())
	}
	responses := make([]map[string]any, len(lines))
	for index, line := range lines {
		if err := json.Unmarshal([]byte(line), &responses[index]); err != nil {
			t.Fatalf("decode response %d: %v", index+1, err)
		}
	}
	return responses
}

func assertPersistedReceiptIdentity(t *testing.T, response map[string]any) {
	t.Helper()
	first := response
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
}

func assertRemovedReceiptProjection(t *testing.T, response map[string]any) {
	t.Helper()
	second := response
	removedReceipt := second["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	if removed, exists := removedReceipt["continuations"]; exists && len(removed.([]any)) != 0 {
		t.Fatalf("removed identity remained live: %#v", removed)
	}
}

func assertReplacementReceiptProjection(t *testing.T, response map[string]any) {
	t.Helper()
	replacementResponse := response
	replacementReceipt := replacementResponse["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	if continuations, exists := replacementReceipt["continuations"]; exists && len(continuations.([]any)) != 0 {
		t.Fatalf("same-name replacement inherited a stale identity: %#v", continuations)
	}
}

func TestMCPIntegerNotationAndRangeValidation(t *testing.T) {
	t.Parallel()
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
				if err := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(registry)).Serve(context.Background(), strings.NewReader(request+"\n")); err != nil {
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
	t.Parallel()
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

func TestToolsListScopesLanguageAndAccessEnums(t *testing.T) {
	t.Parallel()
	javaOnly := t.TempDir()
	if err := os.WriteFile(filepath.Join(javaOnly, "Main.java"), []byte("class Main {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		root      string
		options   []mcp.Option
		languages []string
	}{
		{name: "go only", root: func() string {
			root := t.TempDir()
			writeTestGoModule(t, root)
			if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package sample\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			return root
		}(), languages: []string{"auto", "go"}},
		{name: "mixed", root: schemaWorkspace(t), languages: []string{"auto", "go", "java"}},
		{name: "java only", root: javaOnly, languages: []string{"auto", "java"}},
		{name: "mixed with Java enabled", root: schemaWorkspace(t), options: []mcp.Option{mcp.WithEnabledLanguages("java")}, languages: []string{"auto", "java"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertToolsListLanguageCase(t, tc)
		})
	}
}

func assertToolsListLanguageCase(t *testing.T, test struct {
	name      string
	root      string
	options   []mcp.Option
	languages []string
}) {
	t.Helper()
	tools := listToolsAt(t, test.root, test.options...)
	byName := namedMCPTools(tools)
	assertLanguageSelectorEnums(t, byName, test.languages)
	assertLanguageSpecificTools(t, byName, test.name)
	assertBatchRenameLanguageEnum(t, byName, test.languages)
}

func namedMCPTools(tools []map[string]any) map[string]map[string]any {
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		byName[tool["name"].(string)] = tool
	}
	return byName
}

func assertLanguageSelectorEnums(t *testing.T, byName map[string]map[string]any, languages []string) {
	t.Helper()
	for _, name := range []string{"semantic_lookup", "semantic_inspect_symbol"} {
		properties := byName[name]["inputSchema"].(map[string]any)["properties"].(map[string]any)
		got := properties["language"].(map[string]any)["enum"].([]any)
		if !reflect.DeepEqual(got, toAnySlice(languages)) {
			t.Errorf("%s language enum = %#v, want %#v", name, got, languages)
		}
	}
}

func assertLanguageSpecificTools(t *testing.T, byName map[string]map[string]any, caseName string) {
	t.Helper()
	if caseName == "java only" {
		if _, ok := byName["semantic_insert_construct"]; ok {
			t.Error("Java-only tools/list included the Go-only insertion tool")
		}
		if _, ok := byName["semantic_verify"]["inputSchema"].(map[string]any)["examples"]; ok {
			t.Error("Java-only verify schema retained an example with an unsupported language")
		}
		renameDescription := byName["semantic_rename"]["description"].(string)
		if !strings.Contains(renameDescription, "java:") || strings.Contains(renameDescription, "go:") {
			t.Errorf("Java-only rename description has unfiltered language contracts: %s", renameDescription)
		}
		verifyDescription := byName["semantic_verify"]["description"].(string)
		if !strings.Contains(verifyDescription, "language-independent / auto:") {
			t.Errorf("Java-only verify description omitted its auto routing contract: %s", verifyDescription)
		}
	}
	if caseName == "go only" || caseName == "mixed" {
		access := byName["semantic_insert_construct"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["access_modifier"].(map[string]any)["enum"]
		if got, want := access, toAnySlice([]string{"infer", "public", "private"}); !reflect.DeepEqual(got, want) {
			t.Errorf("Go access enum = %#v, want %#v", got, want)
		}
	}
}

func assertBatchRenameLanguageEnum(t *testing.T, byName map[string]map[string]any, languages []string) {
	t.Helper()
	items := byName["semantic_batch"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["edits"].(map[string]any)["items"].(map[string]any)
	toolSchema := items["properties"].(map[string]any)["tool"].(map[string]any)
	names := toolSchema["enum"].([]any)
	if !schemaEnumContains(names, "semantic_rename") {
		t.Fatal("batch schema omitted semantic_rename")
	}
	rename := byName["semantic_rename"]["inputSchema"].(map[string]any)["properties"].(map[string]any)["language"].(map[string]any)
	if got := rename["enum"]; !reflect.DeepEqual(got, toAnySlice(languages)) {
		t.Errorf("standalone rename language enum = %#v, want %#v", got, languages)
	}
	params := items["properties"].(map[string]any)["params"].(map[string]any)
	if params["type"] != "object" {
		t.Fatalf("batch params schema = %#v, want generic object schema", params)
	}
}

func TestToolsListScopesVerifyAndOmitsBatchForMakeOnlyWorkspace(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n\t@true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tools := listToolsAt(t, root)
	if !hasTool(tools, "semantic_verify") {
		t.Fatal("Auto verify tool was omitted from Make-only tools/list")
	}
	for _, tool := range tools {
		if tool["name"] != "semantic_verify" {
			continue
		}
		language := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)["language"].(map[string]any)
		if got, want := language["enum"], toAnySlice([]string{"auto"}); !reflect.DeepEqual(got, want) {
			t.Fatalf("Make-only verify language enum = %#v, want %#v", got, want)
		}
	}
	if hasTool(tools, "semantic_batch") {
		t.Fatal("tools/list included semantic_batch without any eligible batch children")
	}
	for _, tool := range tools {
		input, ok := tool["inputSchema"].(map[string]any)
		if !ok {
			continue
		}
		properties, ok := input["properties"].(map[string]any)
		if !ok {
			continue
		}
		for name, raw := range properties {
			property, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if enum, hasEnum := property["enum"]; hasEnum && len(enum.([]any)) == 0 {
				t.Errorf("%s.%s exposes an empty enum", tool["name"], name)
			}
		}
	}
}

func TestToolsListUsesRegisteredBackendsForOperationInventory(t *testing.T) {
	t.Parallel()
	root := schemaWorkspace(t)
	registry, err := backend.NewRegistry(javabackend.NewJavaBackend())
	if err != nil {
		t.Fatal(err)
	}
	service := backend.NewService(registry)
	tools := listToolsAt(t, root, mcp.WithService(service))
	if hasTool(tools, "semantic_insert_construct") {
		t.Fatal("Go-only insertion tool was listed without a registered Go backend")
	}
	if !hasTool(tools, "semantic_lookup") || !hasTool(tools, "semantic_metrics") {
		t.Fatal("tools/list omitted language-compatible lookup or backend-independent metrics")
	}
	for _, tool := range tools {
		if tool["name"] != "semantic_lookup" {
			continue
		}
		language := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)["language"].(map[string]any)
		if got, want := language["enum"], toAnySlice([]string{"auto", "java"}); !reflect.DeepEqual(got, want) {
			t.Fatalf("lookup language enum = %#v, want %#v", got, want)
		}
	}
}

func TestBatchVerificationSchemaUsesNativeVerifyResultAcrossCustomProjection(t *testing.T) {
	root := t.TempDir()
	registry := operation.NewRegistry()
	registerNativeVerifyProjectionFixture(t, registry)
	assertNativeVerifyProjectionSchemas(t, root, registry)
	assertNativeVerifyProjectionWireResults(t, root, registry)
}

func registerNativeVerifyProjectionFixture(t *testing.T, registry *operation.Registry) {
	t.Helper()
	verifySchema := operation.ValueSchema{Kind: operation.SchemaString}
	verifyDefinition := operation.Def[struct{}, operation.VerifyRes]{
		Key: "verify", Summary: "Verify custom test fixture.", MCPName: "semantic_verify",
		Params: []operation.ParameterContract{
			{Name: "path", JSONName: "path", Type: operation.ParamString},
			{Name: "language", JSONName: "language", Type: operation.ParamString},
			{Name: "trust_workspace", JSONName: "trust_workspace", Type: operation.ParamBoolean},
		},
		Output: &operation.OperationOutputContract{ResultSchema: &verifySchema},
		Parse:  func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (operation.VerifyRes, error){
			backend.LanguageAuto: func(context.Context, operation.CallContext, struct{}) (operation.VerifyRes, error) {
				return operation.VerifyRes{}, nil
			},
		},
		ProjectOutput: func(operation.VerifyRes) (any, error) { return "standalone projection", nil },
	}
	if err := operation.Register(registry, verifyDefinition); err != nil {
		t.Fatalf("register custom verify: %v", err)
	}
	probe := operation.Def[struct{}, string]{
		Key: "batch-probe", Summary: "No-op batch fixture.", MCPName: "semantic_batch_probe", Batchable: true,
		Params: []operation.ParameterContract{{Name: "path", JSONName: "path", Type: operation.ParamString}},
		Parse:  func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (string, error){
			backend.LanguageAuto: func(context.Context, operation.CallContext, struct{}) (string, error) { return "done", nil },
		},
	}
	if err := operation.Register(registry, probe); err != nil {
		t.Fatalf("register no-op batch fixture: %v", err)
	}
}

func assertNativeVerifyProjectionSchemas(t *testing.T, root string, registry *operation.Registry) {
	t.Helper()
	byName := namedMCPTools(listToolsAtWithRegistry(t, "full", root, registry))
	verifyResultSchema := byName["semantic_verify"]["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	if verifyResultSchema["type"] != "string" {
		t.Fatalf("standalone verify output schema = %#v, want custom string projection", verifyResultSchema)
	}
	batchSchema := byName["semantic_batch"]["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)
	batchProperties := batchSchema["properties"].(map[string]any)
	verificationSchema, err := operation.NativeVerifyOutputSchema().JSONSchema()
	if err != nil {
		t.Fatalf("build native verify schema: %v", err)
	}
	actual, err := json.Marshal(batchProperties["verification"])
	if err != nil {
		t.Fatalf("encode batch verification schema: %v", err)
	}
	want, err := json.Marshal(verificationSchema)
	if err != nil {
		t.Fatalf("encode native verification schema: %v", err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("batch verification schema = %s, want native verify schema %s", actual, want)
	}
}

func assertNativeVerifyProjectionWireResults(t *testing.T, root string, registry *operation.Registry) {
	t.Helper()
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":"."}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_batch_probe","params":{"path":"."}}]}}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(registry))
	if err := server.Serve(context.Background(), strings.NewReader(requests)); err != nil {
		t.Fatalf("serve custom verify calls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("MCP responses = %d, want 2: %s", len(lines), output.String())
	}
	var standaloneResponse, batchResponse map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &standaloneResponse); err != nil {
		t.Fatalf("decode standalone response: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &batchResponse); err != nil {
		t.Fatalf("decode batch response: %v", err)
	}
	standalone := standaloneResponse["result"].(map[string]any)["structuredContent"].(map[string]any)["result"]
	if standalone != "standalone projection" {
		t.Fatalf("standalone projected verify result = %#v", standalone)
	}
	batch := batchResponse["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	verification, ok := batch["verification"].(map[string]any)
	if batch["status"] != "ok" || !ok || verification["normalize"] == nil || verification["checks"] == nil {
		t.Fatalf("batch native verification result = %#v", batch)
	}
}

func assertDirectSymbolTargeting(t *testing.T, byName map[string]map[string]any) {
	t.Helper()
	paramDescription := func(toolName, paramName string) string {
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
	assertContains("semantic_rename.file", paramDescription("semantic_rename", "file"), "Optional for Go if the symbol is unique across the workspace", "recommended to disambiguate common names", "Required for Rust or Java")
	assertContains("semantic_rename.symbol", paramDescription("semantic_rename", "symbol"), "Direct short names are resolved in-place", "calling semantic_lookup first is not required")
	assertContains("semantic_replace_body.symbol", paramDescription("semantic_replace_body", "symbol"), "short function name or Receiver.Method", "resolved directly in the given file")
	assertContains("semantic_replace_construct.symbol", paramDescription("semantic_replace_construct", "symbol"), "short declaration name or Receiver.Method", "resolved directly in the given file")
}
