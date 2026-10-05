// Package mcp_test verifies that every advertised MCP tool reports schema-conformant failures without changing workspace files.

package mcp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"semedit/internal/mcp"
	"semedit/internal/mcptest"
	"semedit/internal/pipeline"
	"strings"
	"testing"
)

func TestAdvertisedMCPToolsReturnSchemaConformantFailures(t *testing.T) {
	root := failureWorkspace(t)
	options := []mcp.Option{mcp.WithToolProfile("full"), mcp.WithGoBaseDir(t.TempDir())}
	tools := advertisedFailureCatalog(t, root, options)
	assertFailureScenarioCompleteness(t, tools)
	if !t.Failed() {
		assertAdvertisedToolFailures(t, root, options, tools)
	}
}

func TestUnknownAndDisabledReloadToolCallsAreJSONRPCFailures(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		"go.mod":  "module example.com/mcpunknown\n\ngo 1.27.1\n",
		"main.go": "package sample\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, path), []byte(contents)); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	for _, test := range []struct {
		name        string
		request     string
		wantMessage string
	}{
		{name: "unknown tool", request: "{\"jsonrpc\":\"2.0\",\"id\":\"unknown\",\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_not_registered\",\"arguments\":{}}}", wantMessage: "Unknown tool"},
		{name: "disabled live reload", request: "{\"jsonrpc\":\"2.0\",\"id\":\"reload\",\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_reload\",\"arguments\":{}}}", wantMessage: "semantic_reload"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := []mcp.Option{mcp.WithToolProfile("full"), mcp.WithGoBaseDir(t.TempDir()), mcp.WithLiveReload(false)}
			var output bytes.Buffer
			server := mcptest.NewServer(t, "full", root, &output, options...)
			catalogRequest := "{\"jsonrpc\":\"2.0\",\"id\":\"list\",\"method\":\"tools/list\"}\n"
			if err := server.Serve(t.Context(), strings.NewReader(catalogRequest)); err != nil {
				t.Fatalf("list catalog: %v", err)
			}
			if strings.Contains(output.String(), "\"name\":\"semantic_reload\"") {
				t.Fatal("disabled semantic_reload was advertised")
			}
			if err := server.Serve(t.Context(), strings.NewReader(test.request+"\n")); err != nil {
				t.Fatalf("call %s: %v", test.name, err)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			lastLine := lines[len(lines)-1]
			var response struct {
				Error *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
				Result struct {
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(lastLine), &response); err != nil {
				t.Fatalf("decode %s response: %v", test.name, err)
			}
			if response.Error == nil || response.Error.Code != -32601 || !strings.Contains(response.Error.Message, test.wantMessage) {
				t.Fatalf("%s was not reported as JSON-RPC method-not-found: %s", test.name, lastLine)
			}
		})
	}
}

func TestMCPRejectedConstructsPreserveSource(t *testing.T) {
	cases := []rejectedConstructCase{
		{
			name:    "misplaced_import",
			source:  "package audit\n\nimport \"strings\"\n\nfunc Normalize(value string) string { return strings.TrimSpace(value) }\n",
			tool:    "semantic_insert_construct",
			params:  map[string]any{"file": "source.go", "kind": "declaration", "source": `import "strings"`},
			message: "imports must appear before other declarations",
		},
		{
			name:    "ignored_move_target",
			source:  "package audit\n\ntype Request struct { ID string }\n\nfunc Normalize(value string) string { return value }\n",
			tool:    "semantic_move_construct",
			params:  map[string]any{"file": "source.go", "symbol": "Request", "target_symbol": "Request.RequestID"},
			message: "target_symbol requires before_symbol or after_symbol",
		},
	}
	for _, testCase := range cases {
		for _, batch := range []bool{false, true} {
			name := testCase.name + "/standalone"
			if batch {
				name = testCase.name + "/batch"
			}
			t.Run(name, func(t *testing.T) {
				checkRejectedConstructSource(t, testCase, batch)
			})
		}
	}
}

type rejectedConstructCase struct {
	name    string
	source  string
	tool    string
	params  map[string]any
	message string
}

type failureCallResponse struct {
	ID     string `json:"id"`
	Result struct {
		IsError           bool           `json:"isError"`
		StructuredContent map[string]any `json:"structuredContent"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type advertisedFailureTool struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}

func snapshotWorkspaceFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if relative != "." && entry.Name() == ".scratch" {
				return filepath.SkipDir
			}
			if relative != "." {
				files[relative] = []byte("directory")
			}
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[relative] = contents
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot workspace files: %v", err)
	}
	return files
}

func failureArgumentValue(tool, key string, schema map[string]any) any {
	if value, ok := schema["const"]; ok {
		return value
	}
	if values, ok := schema["enum"].([]any); ok && len(values) > 0 {
		if key == "language" {
			for _, value := range values {
				if value == "go" {
					return value
				}
			}
		}
		return values[0]
	}
	switch schema["type"] {
	case "boolean":
		if key == "check_only" {
			return true
		}
		return false
	case "integer", "number":
		return 1
	case "array":
		if key == "paths" {
			return []string{"missing.go"}
		}
		return []any{}
	case "object":
		return map[string]any{}
	case "string":
		switch key {

		case "package":
			return "../invalid-module-reference"
		case "file", "path":
			return "missing.go"
		case "symbol":
			return "MissingSymbol"
		case "to":
			return "RenamedSymbol"
		case "language":
			return "go"
		case "in_function", "in_declaration", "target_symbol":
			return "MissingFunction"
		case "source":
			return "func Added() int { return 2 }"
		case "body":
			return "return 2"
		case "case":
			return "case \"missing\":\n\treturn"
		case "snapshot_id":
			return "nonexistent-snapshot"
		case "label":
			return "failure-contract"
		default:
			return "invalid"
		}
	default:
		_ = tool
		return nil
	}
}

func failureArguments(tool string, schema map[string]any, root string) map[string]any {
	_ = root
	args := requiredFailureArguments(tool, schema)
	applyFailureArgumentOverrides(args, tool)
	return args
}

func failureWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for path, contents := range map[string]string{
		"go.mod":  "module example.com/mcpfailures\n\ngo 1.27.1\n",
		"main.go": "package sample\n\nfunc Existing() int { return 1 }\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, path), []byte(contents)); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	return root
}

func advertisedFailureCatalog(t *testing.T, root string, options []mcp.Option) []advertisedFailureTool {
	t.Helper()
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, options...)
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Serve(t.Context(), strings.NewReader(string(append(request, '\n')))); err != nil {
		t.Fatalf("list advertised tools: %v", err)
	}
	var response struct {
		Result struct {
			Tools []advertisedFailureTool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode advertised tools: %v", err)
	}
	if len(response.Result.Tools) == 0 {
		t.Fatal("full MCP catalog is empty")
	}
	return response.Result.Tools
}

func failureScenarioDescriptions() map[string]string {
	return map[string]string{
		"semantic_add_build_dependency":       "invalid module path",
		"semantic_assertion_mode":             "missing Go test source",
		"semantic_change_signature":           "missing symbol selector",
		"semantic_extract_constant":           "missing source file",
		"semantic_extract_function":           "missing source file",
		"semantic_extract_interface":          "missing source file",
		"semantic_extract_type":               "missing source file",
		"semantic_extract_variable":           "missing source file",
		"semantic_find_references":            "unmatched symbol",
		"semantic_implement_contract":         "missing source file",
		"semantic_inline_function":            "missing source file",
		"semantic_inline_variable":            "missing source file",
		"semantic_insert_case":                "unmatched enclosing-function selector",
		"semantic_insert_composite_element":   "unmatched declaration selector",
		"semantic_insert_construct":           "missing source file",
		"semantic_insert_decl":                "missing source file",
		"semantic_insert_declaration":         "missing source file",
		"semantic_insert_field":               "missing source file",
		"semantic_insert_function":            "missing source file",
		"semantic_insert_type":                "missing source file",
		"semantic_inspect_symbol":             "unmatched symbol",
		"semantic_introduce_parameter":        "unmatched function selector",
		"semantic_introduce_parameter_object": "unmatched function selector",
		"semantic_lookup":                     "unmatched symbol",
		"semantic_migrate_type":               "missing source file",
		"semantic_move_construct":             "unmatched source selector",
		"semantic_organize_imports":           "missing source file",
		"semantic_outline":                    "missing source path",
		"semantic_remove_construct":           "unmatched enclosing-function selector",
		"semantic_rename":                     "unmatched symbol",
		"semantic_replace_body":               "missing source file",
		"semantic_replace_construct":          "unmatched enclosing-function selector",
		"semantic_replace_decl":               "missing source file",
		"semantic_scaffold_file":              "target already exists",
		"semantic_set_doc_comment":            "missing source file",
		"semantic_snapshot":                   "missing snapshot path",
		"semantic_undo":                       "nonexistent snapshot id",
		"semantic_verify":                     "missing verification path",
		"semantic_batch":                      "child operation failure",
		"semantic_metrics":                    "out-of-range limit",
		"report_feedback":                     "invalid interface value",
	}
}

func assertFailureScenarioCompleteness(t *testing.T, tools []advertisedFailureTool) {
	t.Helper()
	scenarios := failureScenarioDescriptions()
	for _, tool := range tools {
		if _, ok := scenarios[tool.Name]; !ok {
			t.Errorf("advertised tool %q has no deliberate failure scenario", tool.Name)
		}
		delete(scenarios, tool.Name)
	}
	for name := range scenarios {
		t.Errorf("failure scenario %q is not present in the advertised full catalog", name)
	}
}

func failureRequests(t *testing.T, tools []advertisedFailureTool, root string) ([]string, map[string]string) {
	t.Helper()
	lines := make([]string, 0, len(tools))
	toolByID := make(map[string]string, len(tools))
	for index, tool := range tools {
		id := fmt.Sprintf("tool-%d", index+1)
		toolByID[id] = tool.Name
		encoded, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": id, "method": "tools/call",
			"params": map[string]any{"name": tool.Name, "arguments": failureArguments(tool.Name, tool.InputSchema, root)},
		})
		if err != nil {
			t.Fatalf("encode %s request: %v", tool.Name, err)
		}
		lines = append(lines, string(encoded))
	}
	return lines, toolByID
}

func assertAdvertisedToolFailures(t *testing.T, root string, options []mcp.Option, tools []advertisedFailureTool) {
	t.Helper()
	requests, toolByID := failureRequests(t, tools, root)
	before := snapshotWorkspaceFiles(t, root)
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, options...)
	if err := server.Serve(t.Context(), strings.NewReader(strings.Join(requests, "\n")+"\n")); err != nil {
		t.Fatalf("call advertised tools: %v", err)
	}
	assertFailureResponses(t, output.String(), toolByID)
	after := snapshotWorkspaceFiles(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Errorf("failed tool calls changed workspace files (before %d entries, after %d)", len(before), len(after))
	}
}

func decodeFailureResponses(t *testing.T, output string) map[string]failureCallResponse {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	responses := make(map[string]failureCallResponse, len(lines))
	for _, line := range lines {
		var response failureCallResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Errorf("decode tool response %q: %v", line, err)
			continue
		}
		responses[response.ID] = response
	}
	return responses
}

func assertFailureResponses(t *testing.T, output string, toolByID map[string]string) {
	t.Helper()
	responses := decodeFailureResponses(t, output)
	if len(responses) != len(toolByID) {
		t.Fatalf("tool response count = %d, want %d", len(responses), len(toolByID))
	}
	for id, name := range toolByID {
		response, ok := responses[id]
		if !ok {
			t.Errorf("no response received for %s (%s)", name, id)
			continue
		}
		assertFailureResponse(t, name, response)
	}
}

func assertFailureResponse(t *testing.T, name string, response failureCallResponse) {
	t.Helper()
	if response.Error != nil || !response.Result.IsError {
		t.Errorf("%s did not return an MCP tool error: %#v", name, response)
		return
	}
	content := response.Result.StructuredContent
	if len(content) == 0 {
		t.Errorf("%s error omitted structuredContent", name)
		return
	}
	switch name {
	case "semantic_verify":
		assertVerificationFailureDetails(t, content)
	case "semantic_batch":
		assertBatchFailureDetails(t, content)
	default:
		assertGenericFailureDetails(t, name, content)
	}
	result, hasResult := content["result"]
	if hasResult && content["application_state"] == nil && name != "semantic_verify" && name != "semantic_batch" {
		t.Errorf("%s failure returned an unclassified result: %#v", name, result)
	}
}

func assertVerificationFailureDetails(t *testing.T, content map[string]any) {
	t.Helper()
	message, generic := content["error"].(string)
	result, native := content["result"].(map[string]any)
	failure, hasFailure := result["failure"].(string)
	if (!generic || message == "") && (!native || !hasFailure || failure == "") {
		t.Errorf("semantic_verify failure omitted verification details: %#v", content)
	}
}

func assertBatchFailureDetails(t *testing.T, content map[string]any) {
	t.Helper()
	result, ok := content["result"].(map[string]any)
	if !ok || result["status"] != "error" {
		t.Errorf("semantic_batch failure omitted error status: %#v", content)
	}
}

func assertGenericFailureDetails(t *testing.T, name string, content map[string]any) {
	t.Helper()
	message, ok := content["error"].(string)
	if !ok || message == "" {
		t.Errorf("%s failure presented a success-shaped structured result: %#v", name, content)
	}
}

func requiredFailureArguments(tool string, schema map[string]any) map[string]any {
	properties, _ := schema["properties"].(map[string]any)
	required, _ := schema["required"].([]any)
	args := make(map[string]any, len(required)+4)
	for _, item := range required {
		name, ok := item.(string)
		if !ok {
			continue
		}
		property, _ := properties[name].(map[string]any)
		args[name] = failureArgumentValue(tool, name, property)
	}
	return args
}

func applyFailureArgumentOverrides(args map[string]any, tool string) {
	switch tool {
	case "semantic_snapshot":
		args["label"] = ""
		args["paths"] = []string{"missing.go"}
	case "semantic_undo":
		args["snapshot_id"] = "latest"
	case "semantic_metrics":
		args["limit"] = 101
	case "semantic_verify":
		args["path"] = "missing.go"
		args["language"] = "go"
		args["check_only"] = true
	case "semantic_organize_imports":
		args["file"] = "missing.go"
	case "semantic_scaffold_file":
		args["file"] = "main.go"
		args["package"] = "infer"
	case "semantic_batch":
		args["edits"] = []any{map[string]any{
			"tool":   "semantic_rename",
			"params": map[string]any{"symbol": "MissingSymbol", "to": "RenamedSymbol", "language": "go"},
		}}
	case "report_feedback":
		args["intent"] = "Failure schema probe"
		args["interface"] = "unsupported-interface"
		args["command"] = "semantic_verify"
		args["parameters"] = map[string]any{}
		args["observed_result"] = "Tool returned an error"
		args["unexpected_reason"] = "Expected invalid feedback interface to be rejected"
		args["manual_touchups"] = ""
	case "semantic_rename":
		args["symbol"] = "MissingSymbol"
		args["to"] = "RenamedSymbol"
		args["language"] = "go"
	case "semantic_lookup", "semantic_inspect_symbol", "semantic_find_references":
		args["symbol"] = "MissingSymbol"
		args["language"] = "go"
	case "semantic_outline":
		args["path"] = "missing.go"
		args["language"] = "go"
	case "semantic_replace_construct", "semantic_remove_construct", "semantic_insert_case":
		args["file"] = "main.go"
		args["in_function"] = "MissingFunction"
		args["kind"] = "if"
		if tool == "semantic_insert_case" {
			args["case"] = "case \"missing\":\n\treturn"
		}
	case "semantic_move_construct", "semantic_insert_composite_element":
		args["file"] = "main.go"
		args["symbol"] = "MissingSymbol"
	case "semantic_assertion_mode":
		args["file"] = "missing_test.go"
		args["mode"] = "relax"
	}
}

func checkRejectedConstructSource(t *testing.T, testCase rejectedConstructCase, batch bool) {
	t.Helper()
	root := t.TempDir()
	for path, contents := range map[string]string{
		"go.mod":    "module example.com/constructpreflight\n\ngo 1.27.1\n",
		"source.go": testCase.source,
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, path), []byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(root, "source.go")
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	params := testCase.params
	tool := testCase.tool
	if batch {
		tool = "semantic_batch"
		params = map[string]any{"edits": []map[string]any{{"tool": testCase.tool, "params": params}}}
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": t.Name(), "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": params},
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, mcp.WithGoBaseDir(t.TempDir()))
	if serveErr := server.Serve(t.Context(), bytes.NewReader(append(request, '\n'))); serveErr != nil {
		t.Fatal(serveErr)
	}
	var response failureCallResponse
	if decodeErr := json.Unmarshal(output.Bytes(), &response); decodeErr != nil {
		t.Fatalf("decode tool response: %v\n%s", decodeErr, output.Bytes())
	}
	if response.Error != nil || !response.Result.IsError {
		t.Fatalf("wanted a schema-conformant tool error: %s", output.Bytes())
	}
	payload, err := json.Marshal(response.Result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), testCase.message) {
		t.Fatalf("missing rejection diagnostic: %s", payload)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != testCase.source {
		t.Fatalf("failed construct changed source:\n%s", got)
	}
	after, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("failed construct advanced the source revision")
	}
}
