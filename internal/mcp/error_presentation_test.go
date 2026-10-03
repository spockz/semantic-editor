// Package mcp_test verifies public MCP error text keeps known operation state and uncertainty.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/mcp"
	"semedit/internal/operation"
)

func TestMCPKnownOperationErrorPreservesProjectionAndUnknownOnProjectionFailure(t *testing.T) {
	for _, test := range []struct {
		name        string
		known       bool
		projection  any
		wantState   string
		wantFailure bool
	}{
		{
			name: "known partial publication", known: true,
			projection: map[string]any{"application_state": "partially_published", "changed_files": []string{"main.go"}},
			wantState:  "partially_published",
		},
		{name: "unmatched error", wantState: "unknown"},
		{
			name: "invalid known projection", known: true,
			projection: map[string]any{"application_state": "partially_published", "changed_files": "main.go"},
			wantState:  "unknown", wantFailure: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/errorprojection\n\ngo 1.24\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package sample\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			registry := operation.NewRegistry()
			state := operation.ValueSchema{Kind: operation.SchemaString}
			files := operation.ValueSchema{Kind: operation.SchemaArray, Items: &operation.ValueSchema{Kind: operation.SchemaString}}
			if err := operation.Register(registry, operation.Def[struct{}, string]{
				Key: "probe_known_error", MCPName: "semantic_probe_known_error", CLIName: "probe-known-error",
				Batchable: true,
				Parse:     func(map[string]any) (struct{}, error) { return struct{}{}, nil },
				Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (string, error){
					backend.LanguageGo: func(context.Context, operation.CallContext, struct{}) (string, error) {
						return "", &astedit.SyntaxError{
							File: "main.go", Pos: token.Position{Filename: "main.go", Line: 1, Column: 2},
							Err: errors.New("handler reported a partial write"),
						}
					},
				},
				Format: func(string) (string, error) { return "unused", nil },
				Output: &operation.OperationOutputContract{ResultSchema: &operation.ValueSchema{
					Kind:       operation.SchemaObject,
					Properties: map[string]operation.ValueSchema{"application_state": state, "changed_files": files},
					Required:   []string{"application_state", "changed_files"},
				}},
				ProjectOutput: func(string) (any, error) {
					return map[string]any{"application_state": "persisted", "changed_files": []string{"main.go"}}, nil
				},
				ErrorOutput: &operation.OperationErrorOutputContract{ResultSchema: &operation.ValueSchema{
					Kind:       operation.SchemaObject,
					Properties: map[string]operation.ValueSchema{"application_state": state, "changed_files": files},
					Required:   []string{"application_state", "changed_files"},
				}},
				ProjectErrorOutput: func(error) (any, bool, error) {
					if !test.known {
						return nil, false, nil
					}
					return test.projection, true, nil
				},
			}); err != nil {
				t.Fatalf("register test operation: %v", err)
			}
			var output bytes.Buffer
			input := strings.Join([]string{
				`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_probe_known_error","arguments":{}}}`,
				`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_probe_known_error","params":{}}]}}}`,
			}, "\n") + "\n"
			if err := mcp.NewServer("full", root, &output, mcp.WithRegistry(registry)).Serve(t.Context(), strings.NewReader(input)); err != nil {
				t.Fatalf("serve error call: %v", err)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("response count = %d: %s", len(lines), output.String())
			}
			var response struct {
				Result struct {
					IsError bool `json:"isError"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					Location          map[string]any `json:"location"`
					StructuredContent map[string]any `json:"structuredContent"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(lines[0]), &response); err != nil {
				t.Fatalf("decode error response: %v\n%s", err, output.String())
			}
			if !response.Result.IsError || response.Result.StructuredContent["application_state"] != test.wantState {
				t.Fatalf("error response state = %#v, raw %s", response.Result.StructuredContent, output.String())
			}
			text := response.Result.Content[0].Text
			if !strings.HasPrefix(text, "Critical result (JSON): ") || !strings.Contains(text, "handler reported a partial write") {
				t.Fatalf("error text omitted critical outcome or original error: %q", text)
			}
			_, projectionFailed := response.Result.StructuredContent["error_projection_failure"]
			if projectionFailed != test.wantFailure {
				t.Fatalf("projection failure marker = %t, want %t: %#v", projectionFailed, test.wantFailure, response.Result.StructuredContent)
			}
			if response.Result.Location == nil || response.Result.StructuredContent["location"] == nil || !strings.Contains(text, `"location"`) {
				t.Fatalf("typed error lost source location from one of its response forms: %#v; %q", response.Result, text)
			}
			if test.known && !test.wantFailure {
				projected := response.Result.StructuredContent["result"].(map[string]any)
				if projected["changed_files"].([]any)[0] != "main.go" {
					t.Fatalf("known error result lost exact changed path: %#v", projected)
				}
			}
			var batchResponse struct {
				Result struct {
					IsError bool `json:"isError"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					StructuredContent struct {
						Result struct {
							Results []struct {
								Status           string         `json:"status"`
								ApplicationState string         `json:"application_state"`
								Error            string         `json:"error"`
								Result           map[string]any `json:"result"`
							} `json:"results"`
						} `json:"result"`
					} `json:"structuredContent"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(lines[1]), &batchResponse); err != nil {
				t.Fatalf("decode batch error response: %v\n%s", err, lines[1])
			}
			if !batchResponse.Result.IsError || len(batchResponse.Result.StructuredContent.Result.Results) != 1 {
				t.Fatalf("batch error response missing failed child: %s", lines[1])
			}
			child := batchResponse.Result.StructuredContent.Result.Results[0]
			if child.Status != "error" || child.ApplicationState != test.wantState || child.Error == "" {
				t.Fatalf("failed batch child state/error = %#v", child)
			}
			if test.known && !test.wantFailure && child.Result["changed_files"].([]any)[0] != "main.go" {
				t.Fatalf("failed batch child lost projected changed path: %#v", child.Result)
			}
		})
	}
}
