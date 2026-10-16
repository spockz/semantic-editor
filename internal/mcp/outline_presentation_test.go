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

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/mcptest"
)

func TestMCPOutlineTextSummarizesBulkCollections(t *testing.T) {
	t.Parallel()
	root := largeOutlineWorkspace(t)
	responses := callOutlineMCP(t, root)
	for _, response := range responses {
		assertOutlineMCPResponse(t, root, response)
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
	Payload          []byte
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
			Text: response.Result.Content[0].Text, StructuredResult: response.Result.Structured["result"].(map[string]any), Payload: []byte(line),
		})
	}
	return responses
}

func assertOutlineMCPResponse(t *testing.T, root string, response outlineMCPResponse) {
	t.Helper()
	scope := response.StructuredResult["scope"].(map[string]any)
	switch scope["kind"] {
	case "selected_file":
		assertDeclarationOutlineMCPResponse(t, root, response)
	case "workspace":
		assertWorkspaceOutlineMCPResponse(t, response)
	default:
		t.Fatalf("unexpected outline scope kind: %#v", scope)
	}
}

func assertDeclarationOutlineMCPResponse(t *testing.T, root string, response outlineMCPResponse) {
	t.Helper()
	if response.StructuredResult["file_count"] != float64(1) || response.StructuredResult["symbol_count"] != float64(160) {
		t.Fatalf("structured outline counts = %#v", response.StructuredResult)
	}
	for _, bulkField := range []string{"files", "symbols"} {
		if _, exists := response.StructuredResult[bulkField]; exists {
			t.Fatalf("structured outline duplicated bulk field %q: %#v", bulkField, response.StructuredResult)
		}
	}
	summary := response.StructuredResult["summary"].(string)
	if !strings.Contains(summary, "160 declarations") || strings.Contains(summary, "BulkSymbol") {
		t.Fatalf("structured outline summary should contain counts only: %s", summary)
	}
	for index := range 160 {
		name := fmt.Sprintf("BulkSymbol%03d", index)
		if strings.Count(response.Text, name) != 1 {
			t.Fatalf("MCP text listing retained %s %d times", name, strings.Count(response.Text, name))
		}
		if bytes.Count(response.Payload, []byte(name)) != 1 {
			t.Fatalf("MCP payload contained %s %d times", name, bytes.Count(response.Payload, []byte(name)))
		}
	}
	if strings.Contains(response.Text, `"files"`) || strings.Contains(response.Text, `"symbols"`) {
		t.Fatalf("outline MCP text serialized bulk collections: %s", response.Text)
	}
	if !strings.Contains(response.Text, "160 declarations") || !strings.Contains(response.Text, "1 files") {
		t.Fatalf("outline MCP text omitted file and declaration counts: %s", response.Text)
	}
	nativeBytes, err := minimalNativeOutlineBytes(t, root)
	if err != nil {
		t.Fatalf("marshal equivalent native navigation records: %v", err)
	}
	t.Logf("160-symbol navigation payload: MCP=%d bytes, equivalent native records=%d bytes", len(response.Payload), len(nativeBytes))
	if len(response.Payload) >= len(nativeBytes) {
		t.Fatalf("compact MCP payload (%d bytes) was not smaller than equivalent native records (%d bytes)", len(response.Payload), len(nativeBytes))
	}
}

func minimalNativeOutlineBytes(t *testing.T, root string) ([]byte, error) {
	t.Helper()
	type navigationSymbol struct {
		Name           string        `json:"name"`
		QualifiedName  string        `json:"qualified_name"`
		Kind           string        `json:"kind"`
		File           string        `json:"file"`
		Range          backend.Range `json:"range"`
		SelectionRange backend.Range `json:"selection_range"`
		Signature      *string       `json:"signature,omitempty"`
	}
	type navigationFile struct {
		File     string             `json:"file"`
		Language backend.LanguageID `json:"language"`
		Symbols  []navigationSymbol `json:"symbols"`
	}
	type navigationOutline struct {
		Scope       backend.ReadScope `json:"scope"`
		FileCount   int               `json:"file_count"`
		SymbolCount int               `json:"symbol_count"`
		Files       []navigationFile  `json:"files"`
	}
	native, err := backends.NewDefaultService().Outline(context.Background(), backend.OutlineRequest{
		Project: backend.ProjectContext{RootDir: root, File: "main.go", Language: backend.LanguageGo},
		Path:    "main.go", IncludeUnexported: true,
	})
	if err != nil {
		return nil, fmt.Errorf("native outline: %w", err)
	}
	projection := navigationOutline{Scope: native.Scope, FileCount: len(native.Files), Files: make([]navigationFile, 0, len(native.Files))}
	for _, file := range native.Files {
		projectedFile := navigationFile{File: file.File, Language: file.Language, Symbols: make([]navigationSymbol, 0, len(file.Symbols))}
		for _, symbol := range file.Symbols {
			projectedFile.Symbols = append(projectedFile.Symbols, navigationSymbol{
				Name: symbol.Name, QualifiedName: symbol.QualifiedName, Kind: symbol.Kind, File: symbol.File,
				Range: symbol.Range, SelectionRange: symbol.SelectionRange, Signature: symbol.Signature,
			})
			projection.SymbolCount++
		}
		projection.Files = append(projection.Files, projectedFile)
	}
	return json.Marshal(projection)
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
