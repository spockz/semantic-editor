// Package mcp_test verifies raw MCP arguments are normalized before operation schemas validate them.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/mcptest"
	"github.com/spockz/semantic-editor/internal/operation"
)

func TestMCPIngressNormalizesConstructKindAliasesBeforeSchemaValidation(t *testing.T) {
	type request struct{ Kind string }
	registry := operation.NewRegistry()
	def := operation.Def[request, string]{
		Key:     "construct_kind_normalization_probe",
		MCPName: "semantic_construct_kind_normalization_probe",
		Params: []operation.ParameterContract{{
			Name: "kind", JSONName: "kind", Type: operation.ParamString,
			Required: true, Enums: []string{"element"},
		}},
		NormalizeInput: operation.NormalizeConstructKindInput,
		Parse: func(raw map[string]any) (request, error) {
			kind, _ := raw["kind"].(string)
			return request{Kind: kind}, nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, request) (string, error){
			backend.LanguageAuto: func(_ context.Context, _ operation.CallContext, req request) (string, error) {
				return req.Kind, nil
			},
		},
		Format: func(result string) (string, error) { return "kind=" + result, nil },
	}
	if err := operation.Register(registry, def); err != nil {
		t.Fatal(err)
	}
	var input strings.Builder
	for id, alias := range []string{"composite_element", "literal_element", "entry", "item"} {
		request := map[string]any{
			"jsonrpc": "2.0", "id": id + 1, "method": "tools/call",
			"params": map[string]any{"name": def.MCPName, "arguments": map[string]any{"kind": alias}},
		}
		encoded, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		input.Write(encoded)
		input.WriteByte('\n')
	}
	var output bytes.Buffer
	root := t.TempDir()
	server := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(registry))
	if err := server.Serve(t.Context(), strings.NewReader(input.String())); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("responses = %d, want 4: %s", len(lines), output.String())
	}
	for i, line := range lines {
		var response struct {
			Result struct {
				IsError bool `json:"isError"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("response %d JSON: %v\n%s", i, err, line)
		}
		if response.Result.IsError || len(response.Result.Content) == 0 || !strings.Contains(response.Result.Content[0].Text, "kind=element") {
			t.Fatalf("response %d did not dispatch canonical element kind: %s", i, line)
		}
	}
}
