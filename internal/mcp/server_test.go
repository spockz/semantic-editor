// Package mcp_test validates JSON-RPC protocol compliance and tool routing for the MCP server.
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

	"semedit/internal/mcp"
)

func TestMCPServerLifecycle(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}, "\n") + "\n"

	inBuf := bytes.NewBufferString(input)
	var outBuf bytes.Buffer

	srv := mcp.NewServer("full", ".", &outBuf)
	ctx := t.Context()

	if err := srv.Serve(ctx, inBuf); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 responses, got %d:\n%s", len(lines), outBuf.String())
	}

	// Verify initialize response
	var initResp struct {
		ID     int `json:"id"`
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp.Result.ServerInfo.Name != "semedit" {
		t.Errorf("expected server name 'semedit', got %q", initResp.Result.ServerInfo.Name)
	}

	// Verify tools/list response
	var toolsResp struct {
		ID     int `json:"id"`
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools response: %v", err)
	}
	toolNames := make(map[string]bool)
	for _, tool := range toolsResp.Result.Tools {
		toolNames[tool.Name] = true
	}
	for _, required := range []string{
		"semantic_rename",
		"semantic_insert_declaration",
		"semantic_insert_function",
		"semantic_insert_type",
		"semantic_insert_decl",
		"semantic_organize_imports",
		"semantic_add_dependency",
		"semantic_verify",
		"resolve_symbol_location",
	} {
		if !toolNames[required] {
			t.Errorf("missing tool in full profile: %s", required)
		}
	}
}

func TestMCPMutationsOnlyProfile(t *testing.T) {
	t.Parallel()

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
	inBuf := bytes.NewBufferString(input)
	var outBuf bytes.Buffer

	srv := mcp.NewServer("mutations-only", ".", &outBuf)
	ctx := context.Background()

	if err := srv.Serve(ctx, inBuf); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var toolsResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &toolsResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, tool := range toolsResp.Result.Tools {
		if tool.Name == "resolve_symbol_location" {
			t.Errorf("mutations-only profile should not expose resolve_symbol_location")
		}
	}
}

func TestMCPSpecializedInsertAndImports(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "app.go")
	initial := `package app

func Existing() {}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	messages := []string{
		fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_insert_type","arguments":{"file":%q,"source":"type Item struct { ID string }"}}}`, filePath),
		fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_insert_function","arguments":{"file":%q,"source":"func ProcessItem(i Item) error { return nil }"}}}`, filePath),
		fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_insert_decl","arguments":{"file":%q,"source":"const DefaultLimit = 50"}}}`, filePath),
		fmt.Sprintf(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"semantic_organize_imports","arguments":{"file":%q,"add":["crand crypto/rand"]}}}`, filePath),
	}

	input := strings.Join(messages, "\n") + "\n"
	inBuf := bytes.NewBufferString(input)
	var outBuf bytes.Buffer

	srv := mcp.NewServer("full", dir, &outBuf)
	ctx := context.Background()

	if err := srv.Serve(ctx, inBuf); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "type Item struct") {
		t.Errorf("expected Item struct in file, got:\n%s", content)
	}
	if !strings.Contains(content, "func ProcessItem") {
		t.Errorf("expected ProcessItem func in file, got:\n%s", content)
	}
	if !strings.Contains(content, "const DefaultLimit = 50") {
		t.Errorf("expected DefaultLimit const in file, got:\n%s", content)
	}
}
