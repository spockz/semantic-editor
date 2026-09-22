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

	"semedit/internal/backend"
	"semedit/internal/mcp"
)

type mcpJavaSession struct{}

func (mcpJavaSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	switch method {
	case "initialize":
		return json.RawMessage(`{}`), nil
	case "textDocument/documentSymbol":
		return json.RawMessage(`[{"name":"Widget","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}}]`), nil
	case "textDocument/prepareRename":
		return json.RawMessage(`{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}`), nil
	case "textDocument/rename":
		p := params.(map[string]any)
		doc := p["textDocument"].(map[string]string)
		uri := doc["uri"]
		return json.RawMessage(fmt.Sprintf(`{"changes":{%q:[{"range":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}},"newText":"Gadget"}]}}`, uri)), nil
	default:
		return json.RawMessage(`{}`), nil
	}
}
func (mcpJavaSession) Notify(context.Context, string, any) error { return nil }
func (mcpJavaSession) WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error) {
	return nil, nil
}
func (mcpJavaSession) Close() error { return nil }

var _ backend.JavaSession = mcpJavaSession{}

type mcpVerifySession struct{ methods []string }

func (s *mcpVerifySession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	s.methods = append(s.methods, method)
	switch method {
	case "initialize":
		return json.RawMessage(`{}`), nil
	case "textDocument/formatting":
		return json.RawMessage(`[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":""}]`), nil
	case "textDocument/codeAction":
		return json.RawMessage(`[]`), nil
	default:
		return json.RawMessage(`{}`), nil
	}
}
func (*mcpVerifySession) Notify(context.Context, string, any) error { return nil }
func (*mcpVerifySession) WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error) {
	return nil, nil
}
func (*mcpVerifySession) Close() error { return nil }

var _ backend.JavaSession = (*mcpVerifySession)(nil)

func TestMCPServerLifecycle(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
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
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
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
	if initResp.Result.ProtocolVersion != "2025-06-18" {
		t.Errorf("expected protocol version 2025-06-18, got %q", initResp.Result.ProtocolVersion)
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
		"semantic_add_build_dependency",
		"semantic_verify",
		"semantic_snapshot",
		"semantic_undo",
		"semantic_lookup",
	} {
		if !toolNames[required] {
			t.Errorf("missing tool in full profile: %s", required)
		}
	}
}

func TestMCPServerReturnsConfiguredInstructions(t *testing.T) {
	t.Parallel()

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", ".", &out, mcp.WithInstructions("Prefer semantic operations.")).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Instructions string `json:"instructions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if got, want := response.Result.Instructions, "Prefer semantic operations."; got != want {
		t.Errorf("initialize instructions = %q, want %q", got, want)
	}
}

func TestMCPFirstSemanticCallIncludesSessionTiming(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/sessiontiming\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package sessiontiming\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_verify\",\"arguments\":{\"path\":\".\",\"language\":\"go\"}}}\n", root)
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("response lines = %d, want 2: %s", len(lines), out.String())
	}
	var response struct {
		Result struct {
			StructuredContent struct {
				Result         string              `json:"result"`
				Metrics        map[string]any      `json:"metrics"`
				SessionMetrics *mcp.StartupMetrics `json:"session_metrics"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.StructuredContent.SessionMetrics == nil {
		t.Fatalf("first semantic response missing session_metrics: %s", lines[1])
	}
	if response.Result.StructuredContent.Result == "" {
		t.Fatalf("first semantic response missing structured result: %s", lines[1])
	}
	if response.Result.StructuredContent.Metrics == nil {
		t.Fatalf("first semantic response missing structured metrics: %s", lines[1])
	}
	if got := response.Result.StructuredContent.SessionMetrics; got.ServerStartToInitializeMS < 0 || got.InitializeToFirstSemanticCallMS < 0 {
		t.Errorf("session metrics = %#v, want non-negative durations", got)
	}
}

func TestMCPServerRejectsUnusableGoBaseDir(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(baseDir, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := mcp.NewServer("full", t.TempDir(), &out, mcp.WithGoBaseDir(baseDir)).Serve(t.Context(), strings.NewReader(""))
	if err == nil {
		t.Fatal("Serve succeeded with an unusable Go base directory")
	}
	if !strings.Contains(err.Error(), "prepare MCP Go base directory") {
		t.Errorf("Serve error = %v, want Go base directory context", err)
	}
}

func TestSemanticRenameAdvertisesRust(t *testing.T) {
	var out bytes.Buffer
	srv := mcp.NewServer("full", ".", &out)
	if err := srv.Serve(context.Background(), bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"language"`) || !strings.Contains(out.String(), `"rust"`) || !strings.Contains(out.String(), `selected .rs`) {
		t.Fatalf("semantic_rename schema does not advertise Rust: %s", out.String())
	}
}

func TestJavaMavenImportIsAdvertisedByMCP(t *testing.T) {
	var out bytes.Buffer
	srv := mcp.NewServer("full", ".", &out)
	if err := srv.Serve(context.Background(), bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), `"import_maven"`) < 2 {
		t.Fatalf("MCP schemas must expose import_maven for lookup and rename: %s", out.String())
	}
}

func TestMCPForwardsJavaMavenImportToLookupAndRename(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.java")
	if err := os.WriteFile(file, []byte("class Widget {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"lookup", "rename"} {
		t.Run(operation, func(t *testing.T) {
			var got backend.JavaConfig
			java := backend.NewJavaBackendWithFactory(func(_ context.Context, _ string, config backend.JavaConfig) (backend.JavaSession, error) {
				got = config
				return mcpJavaSession{}, nil
			})
			registry, err := backend.NewRegistry(java)
			if err != nil {
				t.Fatal(err)
			}
			service := backend.NewService(registry)
			var out bytes.Buffer
			args := fmt.Sprintf(`{"file":%q,"symbol":"Widget","language":"java","trust_workspace":true,"import_maven":true`, file)
			method := "semantic_lookup"
			if operation == "rename" {
				args += `,"to":"Gadget"`
				method = "semantic_rename"
			}
			args += `}`
			input := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`+"\n", method, args)
			if err := mcp.NewServer("full", root, &out, mcp.WithService(service)).Serve(context.Background(), bytes.NewBufferString(input)); err != nil {
				t.Fatal(err)
			}
			if !got.ImportMaven {
				t.Fatalf("ImportMaven was not forwarded for %s", operation)
			}
		})
	}
}

func TestMCPRegistrySemanticVerifyForwardsJavaContract(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Widget.java")
	if err := os.WriteFile(file, []byte("class Widget {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var config backend.JavaConfig
	var session *mcpVerifySession
	java := backend.NewJavaBackendWithFactory(func(_ context.Context, _ string, got backend.JavaConfig) (backend.JavaSession, error) {
		config = got
		session = &mcpVerifySession{}
		return session, nil
	})
	registry, err := backend.NewRegistry(java)
	if err != nil {
		t.Fatal(err)
	}
	service := backend.NewService(registry)
	args := fmt.Sprintf(`{"file":%q,"language":"java","trust_workspace":true,"jdtls_home":"/jdtls","java_bin":"/java","import_maven":true,"format_selected_file":true,"organize_imports":true}`, file)
	input := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_verify","arguments":%s}}`+"\n", args)
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out, mcp.WithService(service)).Serve(context.Background(), bytes.NewBufferString(input)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `"isError":true`) {
		t.Fatalf("semantic_verify failed: %s", out.String())
	}
	if !config.ImportMaven || config.JDTLSHome != "/jdtls" || config.JavaBin != "/java" {
		t.Fatalf("Java config not forwarded: %#v", config)
	}
	for _, want := range []string{"textDocument/formatting", "textDocument/codeAction"} {
		found := false
		for _, got := range session.methods {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s request: %v", want, session.methods)
		}
	}
}

func TestMCPMavenTestFailureIncludesStructuredResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	mavenBin := filepath.Join(root, "mvn-fail")
	if err := os.WriteFile(mavenBin, []byte("#!/bin/sh\necho failure-log >&2\nexit 7\n"), 0o700); err != nil { // #nosec G306 -- test-controlled executable fixture.
		t.Fatal(err)
	}
	args := fmt.Sprintf(`{"language":"java","root":%q,"trust_workspace":true,"maven_tool":"system","maven_bin":%q}`, root, mavenBin)
	input := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_maven_test","arguments":%s}}`+"\n", args)
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(context.Background(), bytes.NewBufferString(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Result  map[string]any `json:"result"`
				Metrics struct {
					SchemaVersion int `json:"schema_version"`
					TotalMS       int `json:"total_ms"`
				} `json:"metrics"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Result.IsError {
		t.Fatal("MCP Maven failure must set isError")
	}
	if response.Result.StructuredContent.Result == nil {
		t.Fatal("MCP Maven failure must include structuredContent.result")
	}
	if response.Result.StructuredContent.Metrics.SchemaVersion != 1 {
		t.Fatalf("MCP Maven failure metrics schema = %d, want 1", response.Result.StructuredContent.Metrics.SchemaVersion)
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
		if tool.Name == "semantic_lookup" {
			t.Errorf("mutations-only profile should not expose semantic_lookup")
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
	type timingResponse struct {
		Result struct {
			StructuredContent struct {
				Metrics struct {
					SchemaVersion int `json:"schema_version"`
					Phases        map[string]struct {
						Count int `json:"count"`
					} `json:"phases"`
				} `json:"metrics"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	responses := make([]timingResponse, 0, len(messages))
	for line := range strings.SplitSeq(strings.TrimSpace(outBuf.String()), "\n") {
		var response timingResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("unmarshal MCP response: %v", err)
		}
		responses = append(responses, response)
	}
	if len(responses) != len(messages) {
		t.Fatalf("MCP response count = %d, want %d", len(responses), len(messages))
	}
	if got := responses[0].Result.StructuredContent.Metrics; got.SchemaVersion != 1 || got.Phases["verification.before_diagnostics"].Count != 1 || got.Phases["verification.after_diagnostics"].Count != 1 {
		t.Errorf("first mutation metrics = %#v, want request metrics with before/after diagnostics", got)
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

	callFileMsg := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_lookup","arguments":{"file":%q,"symbol":"bad"}}}`, brokenPath) + "\n"
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
