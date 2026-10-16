// Package mcp_test verifies retained Go sessions across MCP publication boundaries.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/mcptest"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestMCPNonGoPublicationSkipsRetainedGoSessionSync(t *testing.T) {
	for _, mode := range []string{"standalone", "batch", "stage_commit"} {
		t.Run(mode, func(t *testing.T) {
			fixture := newManagedMCPFixture(t)
			registry := operation.DefaultRegistry()
			registerNonGoPublicationProbe(t, registry, fixture.root)
			calls := []string{
				`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
				`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"go","check_only":true}}}`,
			}
			wantResponses := 3
			switch mode {
			case "standalone":
				calls = append(calls, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_test_publish_non_go","arguments":{"file":"notes.json"}}}`)
			case "batch":
				calls = append(calls, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"edits":[{"tool":"semantic_test_publish_non_go","params":{"file":"notes.json"}}]}}}`)
			case "stage_commit":
				calls = append(calls,
					`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_stage","arguments":{}}}`,
					`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"semantic_test_publish_non_go","arguments":{"file":"notes.json"}}}`,
					`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"semantic_commit","arguments":{"skip_hooks":true}}}`)
				wantResponses = 5
			}
			input := strings.Join(calls, "\n") + "\n"
			var output bytes.Buffer
			err := mcptest.NewServer(t, "full", fixture.root, &output, mcp.WithRegistry(registry)).Serve(t.Context(), strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			assertPublicationResponsesSucceed(t, output.String(), wantResponses)
			if _, err := os.Stat(filepath.Join(fixture.root, "notes.json")); err != nil {
				t.Fatalf("non-Go file was not published: %v", err)
			}
		})
	}
}

func registerNonGoPublicationProbe(t *testing.T, registry *operation.Registry, root string) {
	t.Helper()
	params := []operation.ParameterContract{{Name: "file", JSONName: "file", CLIName: "file", Type: operation.ParamString, Required: true}}
	err := operation.Register(registry, operation.Def[string, operation.FileEditRes]{
		Key: "test_publish_non_go", MCPName: "semantic_test_publish_non_go", Params: params, Batchable: true, BufferedBatch: true,
		Parse: func(raw map[string]any) (string, error) {
			if err := operation.CheckParams(raw, params); err != nil {
				return "", err
			}
			return operation.ParseString(raw, "file", "file", true)
		},
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, string) (operation.FileEditRes, error){
			backend.LanguageGo: func(ctx context.Context, call operation.CallContext, file string) (operation.FileEditRes, error) {
				if err := os.Remove(filepath.Join(root, ".scratch", "bin", "gopls")); err != nil {
					return operation.FileEditRes{}, fmt.Errorf("disable future session validation: %w", err)
				}
				if err := sourcebuf.WriteFile(ctx, filepath.Join(call.WorkDir, file), []byte("{\"ok\":true}\n")); err != nil {
					return operation.FileEditRes{}, err
				}
				return operation.FileEditRes{File: file}, nil
			},
		},
		Format: func(operation.FileEditRes) (string, error) { return "published", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMCPGoStageCommitKeepsPublicationRepair(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"go","check_only":true}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_stage","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"semantic_insert_function","arguments":{"file":"value.go","source":"func Added() int { return 2 }"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"semantic_commit","arguments":{"skip_hooks":true}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"semantic_rename","arguments":{"file":"value.go","symbol":"Added","to":"Renamed"}}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	err := mcptest.NewServer(t, "full", fixture.root, &output).Serve(t.Context(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	assertPublicationResponsesSucceed(t, output.String(), 6)
	source, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "func Renamed() int") {
		t.Fatalf("post-commit Go refactoring did not observe published source: %s", source)
	}
}

func assertPublicationResponsesSucceed(t *testing.T, output string, want int) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != want {
		t.Fatalf("got %d responses, want %d: %s", len(lines), want, output)
	}
	for _, line := range lines {
		var response struct {
			Error  json.RawMessage `json:"error"`
			Result struct {
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error != nil || response.Result.IsError {
			t.Fatalf("MCP request failed: %s", line)
		}
	}
}
