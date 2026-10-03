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
	for _, test := range knownOperationErrorCases() {
		t.Run(test.Name, func(t *testing.T) { runKnownOperationErrorCase(t, test) })
	}
}

type knownOperationErrorCase struct {
	Name        string
	Known       bool
	Projection  any
	WantState   string
	WantFailure bool
}

type knownOperationErrorResponse struct {
	Result struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Location          map[string]any `json:"location"`
		StructuredContent map[string]any `json:"structuredContent"`
	} `json:"result"`
}

type knownOperationBatchErrorResponse struct {
	Result struct {
		IsError           bool `json:"isError"`
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

func knownOperationErrorCases() []knownOperationErrorCase {
	return []knownOperationErrorCase{
		{
			Name: "known partial publication", Known: true,
			Projection: map[string]any{"application_state": "partially_published", "changed_files": []string{"main.go"}},
			WantState:  "partially_published",
		},
		{Name: "unmatched error", WantState: "unknown"},
		{
			Name: "invalid known projection", Known: true,
			Projection: map[string]any{"application_state": "partially_published", "changed_files": "main.go"},
			WantState:  "unknown", WantFailure: true,
		},
	}
}

func runKnownOperationErrorCase(t *testing.T, test knownOperationErrorCase) {
	t.Helper()
	root := prepareKnownOperationErrorRoot(t)
	registry := operation.NewRegistry()
	registerKnownOperationErrorProbe(t, registry, test)
	lines := serveKnownOperationErrorCalls(t, root, registry)
	assertKnownOperationCall(t, test, lines[0])
	assertKnownOperationBatchCall(t, test, lines[1])
}

func prepareKnownOperationErrorRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeMCPTestFile(t, filepath.Join(root, "go.mod"), "module example.com/errorprojection\n\ngo 1.24\n")
	writeMCPTestFile(t, filepath.Join(root, "main.go"), "package sample\n\nfunc Value() int { return 1 }\n")
	return root
}

func writeMCPTestFile(t *testing.T, path, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func registerKnownOperationErrorProbe(t *testing.T, registry *operation.Registry, test knownOperationErrorCase) {
	t.Helper()
	schemas := knownErrorOutputSchemas()
	definition := operation.Def[struct{}, string]{
		Key: "probe_known_error", MCPName: "semantic_probe_known_error", CLIName: "probe-known-error", Batchable: true,
		Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (string, error){
			backend.LanguageGo: func(context.Context, operation.CallContext, struct{}) (string, error) {
				return "", &astedit.SyntaxError{File: "main.go", Pos: token.Position{Filename: "main.go", Line: 1, Column: 2}, Err: errors.New("handler reported a partial write")}
			},
		},
		Format: func(string) (string, error) { return "unused", nil },
		Output: knownErrorOutputContract(schemas),
		ProjectOutput: func(string) (any, error) {
			return map[string]any{"application_state": "persisted", "changed_files": []string{"main.go"}}, nil
		},
		ErrorOutput: knownErrorErrorContract(schemas),
		ProjectErrorOutput: func(error) (any, bool, error) {
			if !test.Known {
				return nil, false, nil
			}
			return test.Projection, true, nil
		},
	}
	if err := operation.Register(registry, definition); err != nil {
		t.Fatalf("register test operation: %v", err)
	}
}

type knownOperationErrorSchemas struct {
	State operation.ValueSchema
	Files operation.ValueSchema
}

func knownErrorOutputSchemas() knownOperationErrorSchemas {
	return knownOperationErrorSchemas{
		State: operation.ValueSchema{Kind: operation.SchemaString},
		Files: operation.ValueSchema{Kind: operation.SchemaArray, Items: &operation.ValueSchema{Kind: operation.SchemaString}},
	}
}

func knownErrorOutputContract(schemas knownOperationErrorSchemas) *operation.OperationOutputContract {
	return &operation.OperationOutputContract{ResultSchema: &operation.ValueSchema{
		Kind: operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{
			"application_state": schemas.State,
			"changed_files":     schemas.Files,
		},
		Required: []string{"application_state", "changed_files"},
	}}
}

func knownErrorErrorContract(schemas knownOperationErrorSchemas) *operation.OperationErrorOutputContract {
	return &operation.OperationErrorOutputContract{ResultSchema: &operation.ValueSchema{
		Kind: operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{
			"application_state": schemas.State,
			"changed_files":     schemas.Files,
		},
		Required: []string{"application_state", "changed_files"},
	}}
}

func serveKnownOperationErrorCalls(t *testing.T, root string, registry *operation.Registry) []string {
	t.Helper()
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
	return lines
}

func assertKnownOperationCall(t *testing.T, test knownOperationErrorCase, line string) {
	t.Helper()
	var response knownOperationErrorResponse
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatalf("decode error response: %v\n%s", err, line)
	}
	assertKnownOperationErrorFields(t, test, response)
}

func assertKnownOperationErrorFields(t *testing.T, test knownOperationErrorCase, response knownOperationErrorResponse) {
	t.Helper()
	content := response.Result.StructuredContent
	if !response.Result.IsError || content["application_state"] != test.WantState {
		t.Fatalf("error response state = %#v", content)
	}
	text := response.Result.Content[0].Text
	if !strings.HasPrefix(text, "Critical result (JSON): ") || !strings.Contains(text, "handler reported a partial write") {
		t.Fatalf("error text omitted critical outcome or original error: %q", text)
	}
	_, projectionFailed := content["error_projection_failure"]
	if projectionFailed != test.WantFailure {
		t.Fatalf("projection failure marker = %t, want %t: %#v", projectionFailed, test.WantFailure, content)
	}
	if response.Result.Location == nil || content["location"] == nil || !strings.Contains(text, `"location"`) {
		t.Fatalf("typed error lost source location: %#v; %q", response.Result, text)
	}
	if test.Known && !test.WantFailure {
		assertKnownErrorChangedPath(t, content["result"])
	}
}

func assertKnownErrorChangedPath(t *testing.T, value any) {
	t.Helper()
	projected := value.(map[string]any)
	if projected["changed_files"].([]any)[0] != "main.go" {
		t.Fatalf("known error result lost exact changed path: %#v", projected)
	}
}

func assertKnownOperationBatchCall(t *testing.T, test knownOperationErrorCase, line string) {
	t.Helper()
	var response knownOperationBatchErrorResponse
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatalf("decode batch error response: %v\n%s", err, line)
	}
	children := response.Result.StructuredContent.Result.Results
	if !response.Result.IsError || len(children) != 1 {
		t.Fatalf("batch error response missing failed child: %s", line)
	}
	child := children[0]
	if child.Status != "error" || child.ApplicationState != test.WantState || child.Error == "" {
		t.Fatalf("failed batch child state/error = %#v", child)
	}
	if test.Known && !test.WantFailure {
		assertKnownErrorChangedPath(t, child.Result)
	}
}
