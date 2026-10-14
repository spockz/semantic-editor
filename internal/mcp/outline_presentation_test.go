// Package mcp_test verifies that outline collections appear once in MCP results.
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

	"semedit/internal/mcptest"
)

func TestMCPOutlineTextSummarizesBulkCollections(t *testing.T) {
	t.Parallel()
	root := largeOutlineWorkspace(t)
	responses := callOutlineMCP(t, root)
	for _, response := range responses {
		assertOutlineMCPResponse(t, response)
	}
}

func largeOutlineWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestGoModule(t, root)
	var source strings.Builder
	source.WriteString("package sample\n\n")
	for index := range 160 {
		fmt.Fprintf(&source, "// BulkSymbol%03d documents the declaration and carries enough detail to make the complete outline large.\nfunc BulkSymbol%03d() {}\n\n", index, index)
	}
	writeOutlineFixtureFile(t, filepath.Join(root, "main.go"), source.String())
	for index := range 48 {
		name := fmt.Sprintf("pkg%03d", index)
		directory := filepath.Join(root, name)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		packageSource := fmt.Sprintf("// Package %s documents a workspace package.\npackage %s\n\nconst Payload = %q\n", name, name, strings.Repeat("package-detail-", 32))
		writeOutlineFixtureFile(t, filepath.Join(directory, "package.go"), packageSource)
	}
	return root
}

func writeOutlineFixtureFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

type outlineMCPResponse struct {
	Text             string
	StructuredResult map[string]any
}

func callOutlineMCP(t *testing.T, root string) []outlineMCPResponse {
	t.Helper()
	requests := []map[string]any{
		{"name": "semantic_outline", "arguments": map[string]any{"path": "main.go", "language": "go"}},
		{"name": "semantic_workspace_outline", "arguments": map[string]any{"path": "."}},
	}
	var input strings.Builder
	for index, request := range requests {
		encoded, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": index + 1, "method": "tools/call", "params": request})
		if err != nil {
			t.Fatal(err)
		}
		input.Write(encoded)
		input.WriteByte('\n')
	}
	var output bytes.Buffer
	if err := mcptest.NewServer(t, "full", root, &output).Serve(context.Background(), strings.NewReader(input.String())); err != nil {
		t.Fatalf("serve outline calls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != len(requests) {
		t.Fatalf("MCP responses = %d, want %d", len(lines), len(requests))
	}
	responses := make([]outlineMCPResponse, 0, len(lines))
	for _, line := range lines {
		var response struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				Structured map[string]any `json:"structuredContent"`
				IsError    bool           `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("decode MCP response: %v", err)
		}
		if response.Result.IsError || len(response.Result.Content) != 1 {
			t.Fatalf("outline MCP call failed: %s", line)
		}
		responses = append(responses, outlineMCPResponse{
			Text: response.Result.Content[0].Text, StructuredResult: response.Result.Structured["result"].(map[string]any),
		})
	}
	return responses
}

func assertOutlineMCPResponse(t *testing.T, response outlineMCPResponse) {
	t.Helper()
	scope := response.StructuredResult["scope"].(map[string]any)
	switch scope["kind"] {
	case "selected_file":
		assertDeclarationOutlineMCPResponse(t, response)
	case "workspace":
		assertWorkspaceOutlineMCPResponse(t, response)
	default:
		t.Fatalf("unexpected outline scope kind: %#v", scope)
	}
}

func assertDeclarationOutlineMCPResponse(t *testing.T, response outlineMCPResponse) {
	t.Helper()
	files := response.StructuredResult["files"].([]any)
	file := files[0].(map[string]any)
	if len(file["symbols"].([]any)) != 160 {
		t.Fatalf("structured outline lost symbols: %d", len(file["symbols"].([]any)))
	}
	if !strings.Contains(response.Text, "1 file, 160 symbols") {
		t.Fatalf("outline summary omitted file and symbol counts: %s", response.Text)
	}
	if strings.Contains(response.Text, "BulkSymbol") || strings.Contains(response.Text, `"files"`) || strings.Contains(response.Text, `"symbols"`) {
		t.Fatalf("outline text duplicated declaration data: %s", response.Text)
	}
}

func assertWorkspaceOutlineMCPResponse(t *testing.T, response outlineMCPResponse) {
	t.Helper()
	packages := response.StructuredResult["packages"].([]any)
	if len(packages) != 49 {
		t.Fatalf("structured workspace outline packages = %d, want root plus 48 subpackages", len(packages))
	}
	if !strings.Contains(response.Text, "1 module, 49 packages") {
		t.Fatalf("workspace outline summary omitted counts: %s", response.Text)
	}
	if strings.Contains(response.Text, "pkg000") || strings.Contains(response.Text, `"packages"`) || strings.Contains(response.Text, `"modules"`) {
		t.Fatalf("workspace outline text duplicated package data: %s", response.Text)
	}
}
