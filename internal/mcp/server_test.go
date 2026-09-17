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
		"semantic_snapshot",
		"semantic_undo",
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

func TestMCPToolLocationError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "app.go")
	if err := os.WriteFile(filePath, []byte("package app\n"), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	callMsg := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_insert_function","arguments":{"file":%q,"source":"func broken( {"}}}`, filePath) + "\n"
	inBuf := bytes.NewBufferString(callMsg)
	var outBuf bytes.Buffer

	srv := mcp.NewServer("full", dir, &outBuf)
	ctx := context.Background()

	if err := srv.Serve(ctx, inBuf); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			IsError  bool `json:"isError"`
			Location *struct {
				URI   string `json:"uri"`
				Range struct {
					Start struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					} `json:"start"`
					End struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					} `json:"end"`
				} `json:"range"`
			} `json:"location"`
		} `json:"result"`
	}

	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v\nraw: %s", err, outBuf.String())
	}

	if !resp.Result.IsError {
		t.Fatalf("expected isError true, got false")
	}

	if resp.Result.Location == nil {
		t.Fatalf("expected location in tool error response, got nil:\n%s", outBuf.String())
	}

	expectedURI := "snippet:///source"
	if resp.Result.Location.URI != expectedURI {
		t.Errorf("expected snippet URI %q, got %q", expectedURI, resp.Result.Location.URI)
	}

	if resp.Result.Location.Range.Start.Line < 0 || resp.Result.Location.Range.Start.Character < 0 {
		t.Errorf("expected non-negative 0-indexed range, got %+v", resp.Result.Location.Range)
	}

	// Also verify disk file location on parse error
	brokenPath := filepath.Join(dir, "broken.go")
	if err := os.WriteFile(brokenPath, []byte("package app\nfunc bad( {\n"), 0o600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}

	callFileMsg := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"resolve_symbol_location","arguments":{"file":%q,"symbol":"bad"}}}`, brokenPath) + "\n"
	var outBuf2 bytes.Buffer
	srv2 := mcp.NewServer("full", dir, &outBuf2)
	if err := srv2.Serve(ctx, bytes.NewBufferString(callFileMsg)); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp2 struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			IsError  bool `json:"isError"`
			Location *struct {
				URI   string `json:"uri"`
				Range struct {
					Start struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					} `json:"start"`
				} `json:"range"`
			} `json:"location"`
		} `json:"result"`
	}
	if err := json.Unmarshal(outBuf2.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal resp2: %v", err)
	}
	if resp2.Result.Location == nil {
		t.Fatalf("expected location in file error response, got nil")
	}
	if !strings.HasPrefix(resp2.Result.Location.URI, "file://") {
		t.Errorf("expected file:// URI for disk file, got %q", resp2.Result.Location.URI)
	}
}

func TestMCPSnapshotAndUndo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	calcFile := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(calcFile, []byte("package calc\nfunc Val() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write calc.go: %v", err)
	}

	ctx := t.Context()

	// 1. Take snapshot
	snapReq := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_snapshot","arguments":{"label":"test-snap"}}}` + "\n"
	var snapOut bytes.Buffer
	srv := mcp.NewServer("full", dir, &snapOut)
	if err := srv.Serve(ctx, bytes.NewBufferString(snapReq)); err != nil {
		t.Fatalf("snapshot call: %v", err)
	}

	var snapResp struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(snapOut.Bytes(), &snapResp); err != nil {
		t.Fatalf("unmarshal snapResp: %v", err)
	}
	if snapResp.Result.IsError || len(snapResp.Result.Content) == 0 {
		t.Fatalf("snapshot failed: %s", snapOut.String())
	}

	var snapData struct {
		SnapshotID string `json:"snapshot_id"`
	}
	if err := json.Unmarshal([]byte(snapResp.Result.Content[0].Text), &snapData); err != nil {
		t.Fatalf("unmarshal snapData: %v", err)
	}
	if snapData.SnapshotID == "" {
		t.Fatalf("empty snapshot id")
	}

	// 2. Modify file on disk
	if err := os.WriteFile(calcFile, []byte("package calc\nfunc Val() int { return 99 }\n"), 0o600); err != nil {
		t.Fatalf("modify calc.go: %v", err)
	}

	// 3. Undo snapshot
	undoReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_undo","arguments":{"snapshot_id":%q}}}`, snapData.SnapshotID) + "\n"
	var undoOut bytes.Buffer
	srv2 := mcp.NewServer("full", dir, &undoOut)
	if err := srv2.Serve(ctx, bytes.NewBufferString(undoReq)); err != nil {
		t.Fatalf("undo call: %v", err)
	}

	var undoResp struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(undoOut.Bytes(), &undoResp); err != nil {
		t.Fatalf("unmarshal undoResp: %v", err)
	}
	if undoResp.Result.IsError || len(undoResp.Result.Content) == 0 {
		t.Fatalf("undo failed: %s", undoOut.String())
	}

	// 4. Verify restored content on disk
	// #nosec G304 -- test reads temporary file
	restored, err := os.ReadFile(calcFile)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if !strings.Contains(string(restored), "return 1") {
		t.Errorf("expected return 1, got %s", string(restored))
	}
}
