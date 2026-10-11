// This file proves composite selector continuations can be copied only after they describe final live source.

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
	"strings"
	"testing"

	"semedit/internal/mcptest"
	"semedit/internal/operation"
)

func TestLegacyCompositeElementOperationIsRejected(t *testing.T) {
	registry := operation.DefaultRegistry()
	for _, lookup := range []struct {
		name string
		find func(string) bool
	}{
		{name: "insert_composite_element", find: func(name string) bool { _, ok := registry.LookupKey(name); return ok }},
		{name: "semantic_insert_composite_element", find: func(name string) bool { _, ok := registry.LookupMCP(name); return ok }},
		{name: "insert-composite-element", find: func(name string) bool { _, ok := registry.LookupCLI(name); return ok }},
	} {
		if lookup.find(lookup.name) {
			t.Errorf("legacy operation lookup %q remains registered", lookup.name)
		}
	}
	if _, err := registry.Dispatch(operation.CallContext{}, "insert_composite_element", map[string]any{}); !errors.Is(err, operation.ErrUnknownOperation) {
		t.Errorf("legacy batch action dispatch error = %v, want ErrUnknownOperation", err)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": "semantic_insert_composite_element", "arguments": map[string]any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := mcptest.NewServer(t, "full", t.TempDir(), &output).Serve(context.Background(), bytes.NewReader(append(request, '\n'))); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode unknown legacy tool response: %v: %s", err, output.String())
	}
	if response["error"] == nil {
		t.Fatalf("legacy MCP tool call was not rejected as unknown: %#v", response)
	}
}

func TestCompositeSelectorContinuationFeedsNextInsertion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestGoModule(t, root)
	path := filepath.Join(root, "routes.go")
	initial := []byte("package routes\nvar routes = map[string]int{\"health\": 1}\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write composite fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "routes_test.go"), []byte("package routes\nimport \"testing\"\nfunc TestRoutes(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatalf("write composite test package file: %v", err)
	}
	inspected := callCompositeTool(t, root, 1, "semantic_inspect_symbol", map[string]any{
		"symbol": "routes", "file": "routes.go", "language": "go", "trust_workspace": true,
	})
	inspectResult := inspected["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	inspectRecords := inspectResult["continuations"].([]any)
	if len(inspectRecords) != 1 {
		t.Fatalf("inspection continuations = %#v", inspectRecords)
	}
	inspectRecord := inspectRecords[0].(map[string]any)
	if inspectRecord["file"] != "routes.go" || inspectRecord["in_declaration"] != "routes" || inspectRecord["expected_revision"] == "" {
		t.Fatalf("inspect did not provide same-record declaration evidence: %#v", inspectRecord)
	}
	first := callCompositeTool(t, root, 2, "semantic_insert_construct", map[string]any{
		"file": inspectRecord["file"], "kind": "element", "in_declaration": inspectRecord["in_declaration"],
		"expected_revision": inspectRecord["expected_revision"], "discriminator": `map[string]int{"health": 1}`,
		"source": `"metrics": 2`, "placement": "append",
	})
	firstResult := first["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	firstRecords, ok := firstResult["selector_continuations"].([]any)
	if !ok || len(firstRecords) != 1 {
		t.Fatalf("first insertion omitted live selector record: %#v", firstResult)
	}
	record := firstRecords[0].(map[string]any)
	if record["file"] != "routes.go" || record["kind"] != "element" || record["in_declaration"] != "routes" || !strings.Contains(record["discriminator"].(string), `"metrics": 2`) || record["expected_revision"] == "" {
		t.Fatalf("first selector continuation is not refreshed: %#v", record)
	}
	second := callCompositeTool(t, root, 3, "semantic_insert_construct", map[string]any{
		"file": record["file"], "kind": record["kind"], "in_declaration": record["in_declaration"], "discriminator": record["discriminator"],
		"expected_revision": record["expected_revision"], "source": `"ready": 3`,
	})
	secondResult := second["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	secondRecords, ok := secondResult["selector_continuations"].([]any)
	if !ok || len(secondRecords) != 1 {
		t.Fatalf("follow-up insertion omitted refreshed selector: %#v", secondResult)
	}
	secondRecord := secondRecords[0].(map[string]any)
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read final composite source: %v", err)
	}
	digest := sha256.Sum256(updated)
	if secondRecord["expected_revision"] != hex.EncodeToString(digest[:]) || !strings.Contains(secondRecord["discriminator"].(string), `"ready": 3`) {
		t.Fatalf("follow-up selector does not match final source: %#v", secondRecord)
	}
	staleSource := append(bytes.Clone(updated), []byte("// external drift\n")...)
	if writeErr := os.WriteFile(path, staleSource, 0o600); writeErr != nil {
		t.Fatalf("introduce stale selector revision: %v", writeErr)
	}
	beforeStale := bytes.Clone(staleSource)
	stale := callCompositeTool(t, root, 4, "semantic_insert_construct", map[string]any{
		"file": secondRecord["file"], "kind": secondRecord["kind"], "in_declaration": secondRecord["in_declaration"],
		"discriminator": secondRecord["discriminator"], "expected_revision": secondRecord["expected_revision"],
		"source": `"stale": 4`,
	}, true)
	if !strings.Contains(stale["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string), "expected_revision does not match") {
		t.Fatalf("stale composite revision did not refuse: %#v", stale)
	}
	afterStale, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read source after stale refusal: %v", err)
	}
	if !bytes.Equal(afterStale, beforeStale) {
		t.Fatalf("stale composite request changed source:\n%s", afterStale)
	}
}

func TestCompositeContinuationSelectsOuterLiteralWithNestedValue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestGoModule(t, root)
	path := filepath.Join(root, "routes.go")
	initial := []byte("package routes\nvar routes = map[string]map[string]int{\"health\": map[string]int{\"ready\": 1}}\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write nested composite fixture: %v", err)
	}
	first := callCompositeTool(t, root, 1, "semantic_insert_construct", map[string]any{
		"file": "routes.go", "kind": "element", "in_declaration": "routes", "discriminator": `map[string]map[string]int{"health": map[string]int{"ready": 1}}`,
		"source": `"metrics": map[string]int{"count": 2}`, "placement": "append",
	})
	firstResult := first["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	records, ok := firstResult["selector_continuations"].([]any)
	if !ok || len(records) != 1 {
		t.Fatalf("nested insertion omitted selector continuation: %#v", firstResult)
	}
	record := records[0].(map[string]any)
	discriminator := record["discriminator"].(string)
	if !strings.HasPrefix(discriminator, "map[string]map[string]int{") || !strings.Contains(discriminator, `"metrics": map[string]int{"count": 2}`) {
		t.Fatalf("continuation selected nested literal instead of outer literal: %q", discriminator)
	}
	second := callCompositeTool(t, root, 2, "semantic_insert_construct", map[string]any{
		"file": record["file"], "kind": record["kind"], "in_declaration": record["in_declaration"], "discriminator": discriminator,
		"expected_revision": record["expected_revision"], "source": `"dashboard": map[string]int{"count": 3}`,
	})
	secondResult := second["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	if records, ok := secondResult["selector_continuations"].([]any); !ok || len(records) != 1 {
		t.Fatalf("nested follow-up insertion omitted selector continuation: %#v", secondResult)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read final nested composite source: %v", err)
	}
	if !bytes.Contains(updated, []byte(`"metrics": map[string]int{"count": 2}`)) || !bytes.Contains(updated, []byte(`"dashboard": map[string]int{"count": 3}`)) {
		t.Fatalf("nested outer appends are missing:\n%s", updated)
	}
}

func TestCompositeInsertionUsesLiteralLexicalScopeForMapKeys(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "local variable shadows package constant",
			source: "package routes\nconst elementKey = \"metrics\"\nfunc routes() {\n\telementKey := \"other\"\n\t_ = elementKey\n\t_ = map[string]int{\"health\": 1}\n}\n",
		},
		{
			name:   "local constant duplicate",
			source: "package routes\nconst elementKey = \"metrics\"\nfunc routes() {\n\tconst elementKey = \"health\"\n\t_ = map[string]int{\"health\": 1}\n}\n",
		},
	}
	for index, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestGoModule(t, root)
			path := filepath.Join(root, "routes.go")
			if err := os.WriteFile(path, []byte(test.source), 0o600); err != nil {
				t.Fatalf("write lexical-scope fixture: %v", err)
			}
			before := []byte(test.source)
			response := callCompositeTool(t, root, index+1, "semantic_insert_construct", map[string]any{
				"file": "routes.go", "kind": "element", "in_declaration": "routes", "discriminator": `map[string]int{"health": 1}`,
				"source": `elementKey: 2`, "placement": "append",
			}, true)
			result := response["result"].(map[string]any)
			if result["isError"] != true || !strings.Contains(result["content"].([]any)[0].(map[string]any)["text"].(string), "key") {
				t.Fatalf("lexically invalid key was not refused: %#v", result)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read source after lexical key refusal: %v", err)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("lexically invalid key changed source:\n%s", after)
			}
		})
	}
}

func TestPointerMethodInspectionContinuationFeedsStatementInsertion(t *testing.T) {
	root := t.TempDir()
	writeTestGoModule(t, root)
	path := filepath.Join(root, "dispatch.go")
	initial := []byte("package service\ntype Dispatcher struct{}\nfunc (d *Dispatcher) Record() {\n\tflush()\n}\nfunc flush() {}\nfunc record() {}\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write dispatcher fixture: %v", err)
	}
	inspected := callCompositeTool(t, root, 1, "semantic_inspect_symbol", map[string]any{
		"symbol": "Dispatcher.Record", "file": "dispatch.go", "language": "go", "trust_workspace": true,
	})
	inspectResult := inspected["result"].(map[string]any)["structuredContent"].(map[string]any)["result"].(map[string]any)
	records := inspectResult["continuations"].([]any)
	if len(records) != 1 {
		t.Fatalf("inspection continuations = %#v", records)
	}
	record := records[0].(map[string]any)
	if record["in_function"] != "Dispatcher.Record" {
		t.Fatalf("pointer method continuation = %#v", record)
	}
	callCompositeTool(t, root, 2, "semantic_insert_construct", map[string]any{
		"file": record["file"], "kind": "statement/expression", "in_function": record["in_function"],
		"discriminator": "flush()", "source": "record()", "anchor": "before",
	})
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read inserted dispatcher: %v", err)
	}
	if !strings.Contains(string(updated), "\trecord()\n\tflush()") {
		t.Fatalf("statement insertion did not use inspected pointer-method selector:\n%s", updated)
	}
}

func callCompositeTool(t *testing.T, root string, id int, name string, arguments map[string]any, allowRefusal ...bool) map[string]any {
	t.Helper()
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": id, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": arguments},
	})
	if err != nil {
		t.Fatalf("encode %s request: %v", name, err)
	}
	var output bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &output).Serve(context.Background(), bytes.NewReader(append(request, '\n'))); err != nil {
		t.Fatalf("serve %s request: %v", name, err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode %s response: %v: %s", name, err, output.String())
	}
	if response["error"] != nil {
		t.Fatalf("%s JSON-RPC error: %#v", name, response["error"])
	}
	if result, ok := response["result"].(map[string]any); ok && result["isError"] == true && (len(allowRefusal) == 0 || !allowRefusal[0]) {
		t.Fatalf("%s tool error: %s", name, output.String())
	}
	return response
}
