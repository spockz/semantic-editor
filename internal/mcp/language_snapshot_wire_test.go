// Package mcp_test verifies snapshot behavior through schema-validated MCP wire calls.

package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/mcptest"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLanguageSnapshotGuardsDirectAndBatchWireCalls(t *testing.T) {
	root := t.TempDir()
	writeWireLanguageSource(t, root, "main.go")
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, mcp.WithRegistry(languageWireProbeRegistry(t)), mcp.WithToolProfile("full"))
	if err := initializeWireRoot(t, server, root); err != nil {
		t.Fatalf("initial initialize: %v", err)
	}
	writeWireLanguageSource(t, root, "main.rs")
	arguments := map[string]any{"language": string(backend.LanguageRust)}

	if err := server.Serve(t.Context(), bytes.NewReader(languageWireRequest(t, "semantic_language_probe", arguments))); err != nil {
		t.Fatal(err)
	}
	var direct struct {
		Result struct {
			IsError           bool           `json:"isError"`
			StructuredContent map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &direct); err != nil {
		t.Fatal(err)
	}
	if !direct.Result.IsError || !strings.Contains(fmt.Sprint(direct.Result.StructuredContent["error"]), "not detected") {
		t.Fatalf("direct tools/call response = %#v, want frozen-language rejection", direct)
	}

	output.Reset()
	batchArguments := map[string]any{"edits": []any{
		map[string]any{"tool": "semantic_language_probe", "params": arguments},
	}}
	if err := server.Serve(t.Context(), bytes.NewReader(languageWireRequest(t, "semantic_batch", batchArguments))); err != nil {
		t.Fatal(err)
	}
	var batch struct {
		Result struct {
			IsError           bool           `json:"isError"`
			StructuredContent map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &batch); err != nil {
		t.Fatal(err)
	}
	if !batch.Result.IsError {
		t.Fatalf("semantic_batch wire response = %#v, want language rejection", batch)
	}
	content := fmt.Sprint(batch.Result.StructuredContent)
	if !strings.Contains(content, "not detected") {
		t.Fatalf("semantic_batch response = %#v, want frozen-language rejection", batch.Result.StructuredContent)
	}

	if err := initializeWireRoot(t, server, root); err != nil {
		t.Fatalf("reinitialize after adding Rust source: %v", err)
	}
	output.Reset()
	if err := server.Serve(t.Context(), bytes.NewReader(languageWireRequest(t, "semantic_language_probe", arguments))); err != nil {
		t.Fatal(err)
	}
	var accepted struct {
		Result struct {
			IsError           bool           `json:"isError"`
			StructuredContent map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Result.IsError || accepted.Result.StructuredContent["result"] != "handler reached" {
		t.Fatalf("direct call after reinitialize = %#v, want handler result", accepted)
	}
}

func (request languageWireProbeInput) GetProjectContext() backend.ProjectContext {
	return backend.ProjectContext{Language: request.Language}
}

type languageWireProbeInput struct {
	Language backend.LanguageID
}

func languageWireProbeRegistry(t *testing.T) *operation.Registry {
	t.Helper()
	registry := operation.NewRegistry()
	handler := func(_ context.Context, _ operation.CallContext, _ languageWireProbeInput) (string, error) {
		return "handler reached", nil
	}
	definition := operation.Def[languageWireProbeInput, string]{
		Key:       "language-probe",
		Summary:   "Probe language snapshot dispatch",
		MCPName:   "semantic_language_probe",
		Batchable: true,
		ReadOnly:  true,
		Params: []operation.ParameterContract{{
			Name: "language", CLIName: "language", JSONName: "language", Type: operation.ParamString,
			Description: "Language to dispatch", Required: true, Enums: []string{"go", "rust"}, LanguageSelector: true,
		}},
		Parse: func(raw map[string]any) (languageWireProbeInput, error) {
			language, _ := raw["language"].(string)
			return languageWireProbeInput{Language: backend.LanguageID(language)}, nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, languageWireProbeInput) (string, error){
			backend.LanguageGo:   handler,
			backend.LanguageRust: handler,
		},
		Format: func(result string) (string, error) { return result, nil },
	}
	if err := operation.Register(registry, definition); err != nil {
		t.Fatalf("register probe operation: %v", err)
	}
	return registry
}

func writeWireLanguageSource(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(path, nil); err != nil {
		t.Fatal(err)
	}
}

func initializeWireRoot(t *testing.T, server *mcptest.Server, root string) error {
	t.Helper()
	params, err := json.Marshal(map[string]any{"rootPath": root})
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.Initialize(t.Context(), params)
	return err
}

func languageWireRequest(t *testing.T, tool string, arguments map[string]any) []byte {
	t.Helper()
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": arguments},
	})
	if err != nil {
		t.Fatal(err)
	}
	return append(request, byte(10))
}
