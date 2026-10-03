// Package mcp_test validates JSON-RPC protocol compliance and tool routing for the MCP server.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"semedit/internal/backend"
	javabackend "semedit/internal/backend/java"
	"semedit/internal/godiagnostics"
	"semedit/internal/mcp"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
)

type managedMCPFixture struct {
	root    string
	counter string
	pidFile string
}

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

var _ javabackend.JavaSession = mcpJavaSession{}

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

var _ javabackend.JavaSession = (*mcpVerifySession)(nil)

func newManagedMCPFixture(t *testing.T) managedMCPFixture {
	t.Helper()

	gopls, err := godiagnostics.FindGopls()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":   "module example.com/mcpreuse\n\ngo 1.24\n",
		"value.go": "package sample\n\nfunc Value() int { return 1 }\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, name), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, ".scratch", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(bin, "gopls")
	if err := pipeline.WriteAtomic(wrapper, []byte("#!/bin/sh\nprintf 'start\\n' >> \"$SEMEDIT_GOPLS_COUNT\"\nprintf '%s\\n' \"$$\" >> \"$SEMEDIT_GOPLS_PIDS\"\nexec \"$SEMEDIT_REAL_GOPLS\" \"$@\"\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(wrapper, 0o700); err != nil {
		t.Fatal(err)
	}
	counter := filepath.Join(root, ".scratch", "starts")
	pidFile := filepath.Join(root, ".scratch", "pids")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_REAL_GOPLS", gopls)
	t.Setenv("SEMEDIT_GOPLS_COUNT", counter)
	t.Setenv("SEMEDIT_GOPLS_PIDS", pidFile)
	return managedMCPFixture{root: root, counter: counter, pidFile: pidFile}
}

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
		"semantic_insert_structure",
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

func TestMCPStructuredContinuationFields(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/continuation\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := "package continuation\n\nfunc Hello() string { return \"before\" }\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	input := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_lookup\",\"arguments\":{\"file\":\"main.go\",\"symbol\":\"Hello\"}}}\n{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_replace_body\",\"arguments\":{\"file\":\"main.go\",\"symbol\":\"Hello\",\"body\":\"return \\\"after\\\"\"}}}\n", root)
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("responses = %d, want 3: %s", len(lines), out.String())
	}
	var lookup, edit struct {
		Result struct {
			IsError    bool           `json:"isError"`
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &lookup); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[2]), &edit); err != nil {
		t.Fatal(err)
	}
	if lookup.Result.IsError || edit.Result.IsError {
		t.Fatalf("tool calls failed: %s", out.String())
	}
	for label, content := range map[string]map[string]any{"lookup": lookup.Result.Structured, "replace_body": edit.Result.Structured} {
		if content["result"] == nil {
			t.Errorf("%s structuredContent omitted existing result: %#v", label, content)
		}
		if content["symbol"] != "Hello" {
			t.Errorf("%s structured symbol = %#v", label, content["symbol"])
		}
		if content["file"] != "main.go" {
			t.Errorf("%s structured file = %#v", label, content["file"])
		}
		for _, hidden := range []string{"offset", "receiver"} {
			if _, ok := content[hidden]; ok {
				t.Errorf("%s structuredContent contains %q: %#v", label, hidden, content)
			}
		}
	}
	if lookup.Result.Structured["line"] == nil || lookup.Result.Structured["column"] == nil || lookup.Result.Structured["kind"] != "function" {
		t.Errorf("lookup navigation fields = %#v", lookup.Result.Structured)
	}
}

func TestMCPAmbiguousLookupDoesNotPublishSelectedScopeAsContinuation(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":  "module example.com/ambiguouslookup\n\ngo 1.24\n",
		"main.go": "package ambiguouslookup\n\ntype Server struct{}\nfunc (*Server) Serve() {}\n\ntype Other struct{}\nfunc (*Other) Serve() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_lookup","arguments":{"symbol":"Serve","file":"main.go"}}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("decode ambiguous lookup: %v\n%s", err, out.String())
	}
	if response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("ambiguous lookup failed: %s", out.String())
	}
	text := response.Result.Content[0].Text
	for _, evidence := range []string{`"ambiguous": true`, "Server.Serve", "Other.Serve"} {
		if !strings.Contains(text, evidence) {
			t.Errorf("lookup text omitted ambiguity evidence %q: %s", evidence, text)
		}
	}
	for _, alias := range []string{"file", "symbol", "kind", "line", "column"} {
		if _, exists := response.Result.Structured[alias]; exists {
			t.Errorf("ambiguous lookup advertised unproven continuation alias %q: %#v", alias, response.Result.Structured)
		}
	}
}

func TestMCPExtractedContractContinuesIntoImplement(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod":           "module example.com/contractflow\n\ngo 1.27.1\n",
		"api/store.go":     "package api\n\ntype Store struct{}\n\nfunc (*Store) Read() (int, error) { return 0, nil }\n\ntype Consumer struct{}\n",
		"api/contracts.go": "package api\n",
		"api/methods.go":   "package api\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, name), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	call := func(tool string, arguments map[string]any) map[string]any {
		t.Helper()
		encoded, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": arguments}})
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := mcp.NewServer("full", root, &out).Serve(t.Context(), bytes.NewReader(append(encoded, '\n'))); err != nil {
			t.Fatal(err)
		}
		var response struct {
			Result struct {
				IsError    bool           `json:"isError"`
				Structured map[string]any `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatalf("decode %s response %q: %v", tool, out.String(), err)
		}
		if response.Result.IsError {
			t.Fatalf("%s failed: %s", tool, out.String())
		}
		return response.Result.Structured
	}
	extracted := call("semantic_extract_interface", map[string]any{
		"file": "api/store.go", "in_type": "Store", "symbol": "StoreAPI", "destination_file": "api/contracts.go",
		"receiver_kind": "pointer", "members": []map[string]string{{"file": "api/store.go", "symbol": "(*Store).Read"}},
	})
	contract, ok := extracted["contract"].(string)
	if !ok || contract != "StoreAPI" {
		t.Fatalf("extract contract selector = %#v, want StoreAPI", extracted["contract"])
	}
	contractFile, ok := extracted["contract_file"].(string)
	if !ok || contractFile != "api/contracts.go" {
		t.Fatalf("extract contract_file selector = %#v, want api/contracts.go", extracted["contract_file"])
	}
	consumer := call("semantic_implement_contract", map[string]any{
		"file": "api/store.go", "symbol": "Consumer", "contract": contract, "contract_file": contractFile,
		"destination_file": "api/methods.go", "receiver_kind": "pointer", "body_policy": "panic_stub",
	})
	result, ok := consumer["result"].(map[string]any)
	if !ok {
		t.Fatalf("implement omitted structured result: %#v", consumer)
	}
	if result["type_target"] == nil {
		t.Fatalf("implement omitted live type_target: %#v", result)
	}
	if result["generated_methods"] == nil {
		t.Fatalf("implement omitted generated method continuation: %#v", result)
	}
}

func TestMCPRenameContinuationUsesRenamedSymbol(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/renamecontinuation\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package renamecontinuation\n\nfunc Hello() string { return \"ok\" }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	renameInput := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_rename\",\"arguments\":{\"file\":\"main.go\",\"symbol\":\"Hello\",\"to\":\"Welcome\"}}}\n", root)
	var renameOut bytes.Buffer
	if err := mcp.NewServer("full", root, &renameOut).Serve(t.Context(), strings.NewReader(renameInput)); err != nil {
		t.Fatal(err)
	}
	responses := strings.Split(strings.TrimSpace(renameOut.String()), "\n")
	if len(responses) != 2 {
		t.Fatalf("rename responses = %d: %s", len(responses), renameOut.String())
	}
	var rename struct {
		Result struct {
			IsError    bool           `json:"isError"`
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(responses[1]), &rename); err != nil {
		t.Fatal(err)
	}
	if rename.Result.IsError {
		t.Fatalf("rename failed: %s", responses[1])
	}
	continuedSymbol, ok := rename.Result.Structured["symbol"].(string)
	if !ok || continuedSymbol != "Welcome" {
		t.Fatalf("rename continuation symbol = %#v, want Welcome", rename.Result.Structured["symbol"])
	}
	if rename.Result.Structured["file"] != "main.go" {
		t.Fatalf("rename continuation file = %#v, want main.go", rename.Result.Structured["file"])
	}
	for _, field := range []string{"kind", "line", "column"} {
		if _, exists := rename.Result.Structured[field]; exists {
			t.Errorf("rename structuredContent contains unadvertised field %q: %#v", field, rename.Result.Structured)
		}
	}

	lookupInput := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_lookup\",\"arguments\":{\"file\":\"main.go\",\"symbol\":%q}}}\n", root, continuedSymbol)
	var lookupOut bytes.Buffer
	if err := mcp.NewServer("full", root, &lookupOut).Serve(t.Context(), strings.NewReader(lookupInput)); err != nil {
		t.Fatal(err)
	}
	lookupResponses := strings.Split(strings.TrimSpace(lookupOut.String()), "\n")
	if len(lookupResponses) != 2 {
		t.Fatalf("lookup responses = %d: %s", len(lookupResponses), lookupOut.String())
	}
	var lookup struct {
		Result struct {
			IsError    bool           `json:"isError"`
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lookupResponses[1]), &lookup); err != nil {
		t.Fatal(err)
	}
	if lookup.Result.IsError || lookup.Result.Structured["symbol"] != continuedSymbol {
		t.Fatalf("follow-up lookup did not resolve rename continuation: %s", lookupResponses[1])
	}
}

func TestMCPInsertFunctionContinuationUsesInsertedTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/insertcontinuation\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package insertcontinuation\n\ntype Service struct{}\n\nfunc Anchor() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_insert_function\",\"arguments\":{\"file\":\"main.go\",\"placement\":\"before_symbol\",\"target_symbol\":\"Anchor\",\"source\":\"func (s *Service) Added() {}\"}}}\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError    bool           `json:"isError"`
			Structured map[string]any `json:"structuredContent"`
			Content    []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.IsError {
		t.Fatalf("insert failed: %s", out.String())
	}
	if got := response.Result.Structured["symbol"]; got != "Service.Added" {
		t.Fatalf("continued symbol = %#v, want inserted target Service.Added (not placement anchor Anchor)", got)
	}
	if response.Result.Structured["file"] != "main.go" {
		t.Fatalf("continued file = %#v", response.Result.Structured["file"])
	}
	if line, ok := response.Result.Structured["line"].(float64); !ok || line < 1 {
		t.Fatalf("continued line = %#v, want positive declaration line", response.Result.Structured["line"])
	}
	if len(response.Result.Content) != 1 || !strings.Contains(response.Result.Content[0].Text, "symbol: Service.Added") || !strings.Contains(response.Result.Content[0].Text, "line (navigation only):") || !strings.Contains(response.Result.Content[0].Text, "diagnostic_status:") {
		t.Fatalf("result omitted resolved target and declaration location: %s", out.String())
	}
	continuedFile := response.Result.Structured["file"].(string)
	continuedSymbol := response.Result.Structured["symbol"].(string)
	followup := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_lookup\",\"arguments\":{\"file\":%q,\"symbol\":%q}}}\n", continuedFile, continuedSymbol)
	var followupOut bytes.Buffer
	if err := mcp.NewServer("full", root, &followupOut).Serve(t.Context(), strings.NewReader(followup)); err != nil {
		t.Fatal(err)
	}
	var lookup struct {
		Result struct {
			IsError    bool           `json:"isError"`
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(followupOut.Bytes(), &lookup); err != nil {
		t.Fatal(err)
	}
	if lookup.Result.IsError || lookup.Result.Structured["symbol"] != continuedSymbol || lookup.Result.Structured["file"] != continuedFile {
		t.Fatalf("inserted target fields did not work as a follow-up lookup: %s", followupOut.String())
	}
}

func TestMCPContinuationOmitsAmbiguousSymbolAndStructureSelectorIsNotTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/continuationcases\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package continuationcases\n\nfunc Handle(v string) int { switch v { default: return 0 } }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_insert_structure\",\"arguments\":{\"file\":\"main.go\",\"kind\":\"case\",\"in_function\":\"Handle\",\"discriminator\":\"v\",\"source\":\"case \\\"x\\\": return 1\"}}}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_insert_declaration\",\"arguments\":{\"file\":\"main.go\",\"source\":\"const (Alpha = 1; Beta = 2)\"}}}\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("responses = %d: %s", len(lines), out.String())
	}
	var caseResult, multiResult struct {
		Result struct {
			IsError    bool           `json:"isError"`
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &caseResult); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &multiResult); err != nil {
		t.Fatal(err)
	}
	if caseResult.Result.IsError || caseResult.Result.Structured["in_function"] != "Handle" {
		t.Fatalf("case continuation omitted containing function: %s", lines[0])
	}
	if _, exists := caseResult.Result.Structured["symbol"]; exists {
		t.Fatalf("case selector leaked as changed symbol: %s", lines[0])
	}
	if multiResult.Result.IsError {
		t.Fatalf("multi-spec declaration insertion failed: %s", lines[1])
	}
	if _, exists := multiResult.Result.Structured["symbol"]; exists {
		t.Fatalf("ambiguous multi-spec declaration advertised one symbol: %s", lines[1])
	}
}

func TestMCPServerConstructionRejectsBrokenContinuationRegistry(t *testing.T) {
	registry := operation.NewRegistry()
	err := operation.Register(registry, operation.Def[struct{}, string]{
		Key: "broken", MCPName: "semantic_broken", CLIName: "broken",
		Params:   []operation.ParameterContract{{Name: "file", JSONName: "file", CLIName: "file", Type: operation.ParamString, SourceFields: []string{"semantic_missing.file"}}},
		Parse:    func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (string, error){backend.LanguageGo: func(context.Context, operation.CallContext, struct{}) (string, error) { return "ok", nil }},
		Format:   func(value string) (string, error) { return value, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_broken\",\"arguments\":{}}}\n"
	if err := mcp.NewServer("full", t.TempDir(), &out, mcp.WithRegistry(registry)).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	responses := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(responses) != 2 {
		t.Fatalf("responses = %d: %s", len(responses), out.String())
	}
	for _, raw := range responses {
		var response struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(response.Error.Message, "validate MCP continuation registry") || !strings.Contains(response.Error.Message, "unknown output tool") {
			t.Fatalf("MCP request was not rejected for broken SourceFields: %s", raw)
		}
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
	input := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_verify\",\"arguments\":{\"path\":\".\",\"language\":\"go\"}}}\n{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_metrics\",\"arguments\":{\"limit\":1}}}\n", root)
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("response lines = %d, want 3: %s", len(lines), out.String())
	}
	var toolResponse struct {
		Result struct {
			StructuredContent map[string]json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &toolResponse); err != nil {
		t.Fatal(err)
	}
	if _, ok := toolResponse.Result.StructuredContent["result"]; !ok {
		t.Fatalf("semantic result missing structured result: %s", lines[1])
	}
	if _, ok := toolResponse.Result.StructuredContent["metrics"]; ok {
		t.Fatalf("semantic response included inline metrics: %s", lines[1])
	}
	if _, ok := toolResponse.Result.StructuredContent["session_metrics"]; ok {
		t.Fatalf("semantic response included inline session metrics: %s", lines[1])
	}
	var metricsResponse struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent struct {
				Result json.RawMessage `json:"result"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &metricsResponse); err != nil {
		t.Fatal(err)
	}
	var metricsData struct {
		Records []struct {
			Tool           string              `json:"tool"`
			Metrics        map[string]any      `json:"metrics"`
			SessionMetrics *mcp.StartupMetrics `json:"session_metrics"`
		} `json:"records"`
	}
	if err := json.Unmarshal(metricsResponse.Result.StructuredContent.Result, &metricsData); err != nil {
		t.Fatalf("decode structured metrics records: %v", err)
	}
	records := metricsData.Records
	if len(records) != 1 || records[0].Tool != "semantic_verify" {
		t.Fatalf("metrics records = %#v, want latest semantic_verify call", records)
	}
	if records[0].Metrics["schema_version"] == nil {
		t.Fatalf("separate metrics record has no timings: %#v", records[0])
	}
	if got := records[0].SessionMetrics; got == nil || got.ServerStartToInitializeMS < 0 || got.InitializeToFirstSemanticCallMS < 0 {
		t.Errorf("session metrics = %#v, want non-negative durations", got)
	}
	const prefixLabel = "Critical result (JSON): "
	text := metricsResponse.Result.Content[0].Text
	prefix, _, found := strings.Cut(text, "\n\n")
	if !found || !strings.HasPrefix(prefix, prefixLabel) {
		t.Fatalf("metrics text omitted the critical JSON prefix: %q", text)
	}
	var presented map[string]any
	decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(prefix, prefixLabel)))
	decoder.UseNumber()
	if err := decoder.Decode(&presented); err != nil {
		t.Fatalf("decode metrics text prefix: %v", err)
	}
	var structured any
	decoder = json.NewDecoder(bytes.NewReader(metricsResponse.Result.StructuredContent.Result))
	decoder.UseNumber()
	if err := decoder.Decode(&structured); err != nil {
		t.Fatalf("decode structured metrics result: %v", err)
	}
	if !reflect.DeepEqual(presented["result"], structured) {
		t.Fatalf("metrics text records differ from structured result: text=%#v structured=%#v", presented["result"], structured)
	}
}

func TestMCPMetricsFileSurvivesServerRestart(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/metricsfile\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package metricsfile\ntype Widget struct{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	metricsDir := filepath.Join(root, ".scratch")
	if err := os.MkdirAll(metricsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	metricsPath := filepath.Join(metricsDir, "mcp-metrics.json")
	input := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_lookup\",\"arguments\":{\"file\":\"main.go\",\"symbol\":\"Widget\"}}}\n", root)
	for range 2 {
		var output bytes.Buffer
		if err := mcp.NewServer("full", root, &output, mcp.WithMetricsFile(metricsPath)).Serve(t.Context(), strings.NewReader(input)); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) != 2 || !strings.Contains(lines[1], "\"symbol\":\"Widget\"") {
			t.Fatalf("lookup response = %s", output.String())
		}
	}
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatal(err)
	}
	var records []struct {
		Sequence       uint64              `json:"sequence"`
		Tool           string              `json:"tool"`
		CompletedAt    string              `json:"completed_at"`
		Metrics        map[string]any      `json:"metrics"`
		SessionMetrics *mcp.StartupMetrics `json:"session_metrics"`
	}
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("persisted records = %d, want 2: %s", len(records), data)
	}
	for index, record := range records {
		if record.Sequence != uint64(index+1) || record.Tool != "semantic_lookup" || record.CompletedAt == "" || record.Metrics["total_ms"] == nil || record.SessionMetrics == nil {
			t.Fatalf("persisted record %d = %+v", index, record)
		}
	}
}

func TestMCPMetricsFileKeepsMoreThanRecentHistory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/metricsfile\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package metricsfile\ntype Widget struct{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	metricsPath := filepath.Join(root, "mcp-metrics.json")
	var input strings.Builder
	fmt.Fprintf(&input, "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n", root)
	for id := 2; id <= 102; id++ {
		fmt.Fprintf(&input, "{\"jsonrpc\":\"2.0\",\"id\":%d,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_lookup\",\"arguments\":{\"file\":\"main.go\",\"symbol\":\"Widget\"}}}\n", id)
	}
	var output bytes.Buffer
	if err := mcp.NewServer("full", root, &output, mcp.WithMetricsFile(metricsPath)).Serve(t.Context(), strings.NewReader(input.String())); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatal(err)
	}
	var records []json.RawMessage
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 101 {
		t.Fatalf("persisted records = %d, want 101", len(records))
	}
}

func TestMCPMetricsFileWriteFailureIsExplicit(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/metricsfile\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package metricsfile\ntype Widget struct{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	metricsPath := filepath.Join(root, "missing", "mcp-metrics.json")
	input := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootPath\":%q}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_lookup\",\"arguments\":{\"file\":\"main.go\",\"symbol\":\"Widget\"}}}\n", root)
	var output bytes.Buffer
	err := mcp.NewServer("full", root, &output, mcp.WithMetricsFile(metricsPath)).Serve(t.Context(), strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "write MCP metrics file") {
		t.Fatalf("metrics persistence failure = %v, want explicit write error", err)
	}
}

func TestMCPServerRejectsUnusableGoBaseDir(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(baseDir, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	srv := mcp.NewServer("full", t.TempDir(), &out, mcp.WithGoBaseDir(baseDir))
	_, err := srv.Initialize(t.Context(), nil)
	if err == nil {
		t.Fatal("Initialize succeeded with an unusable Go base directory")
	}
	if !strings.Contains(err.Error(), "prepare MCP Go base directory") {
		t.Errorf("Initialize error = %v, want Go base directory context", err)
	}
}

func TestMCPServerInitializeDeferred(t *testing.T) {
	temp := t.TempDir()
	var out bytes.Buffer
	srv := mcp.NewServer("full", "", &out)

	// Ensure .scratch/go is not eagerly created before Initialize
	if _, err := os.Stat(filepath.Join(temp, ".scratch", "go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected .scratch/go to not exist before Initialize, got %v", err)
	}

	rawParams, err := json.Marshal(map[string]any{
		"rootUri": "file://" + temp,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := srv.Initialize(t.Context(), rawParams)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if result["protocolVersion"] != "2025-06-18" {
		t.Errorf("unexpected protocolVersion: %v", result["protocolVersion"])
	}
	if _, err := os.Stat(filepath.Join(temp, ".scratch", "go")); err != nil {
		t.Errorf("expected .scratch/go to be created on Initialize: %v", err)
	}
}

func TestSemanticRenameAdvertisesRust(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod":     "module example.com/mixed\n\ngo 1.24\n",
		"main.go":    "package main\n",
		"Cargo.toml": "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\nedition = \"2021\"\n",
		"src/lib.rs": "pub fn run() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tools := listToolsAt(t, root)
	for _, tool := range tools {
		if tool["name"] != "semantic_rename" {
			continue
		}
		input := tool["inputSchema"].(map[string]any)
		language := input["properties"].(map[string]any)["language"].(map[string]any)
		want := []any{"auto", "go", "rust"}
		got := language["enum"].([]any)
		if len(got) != len(want) {
			t.Fatalf("semantic_rename language enum = %#v, want %#v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("semantic_rename language enum = %#v, want %#v", got, want)
			}
		}
		description, _ := language["description"].(string)
		if !strings.Contains(description, "Rust requires a selected .rs file") {
			t.Fatalf("semantic_rename.language description does not explain Rust file selection: %q", description)
		}
		return
	}
	t.Fatal("tools/list omitted semantic_rename for a Go+Rust workspace")
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
			java := javabackend.NewJavaBackendWithFactory(func(_ context.Context, _ string, config backend.JavaConfig) (javabackend.JavaSession, error) {
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
	var sessions []*mcpVerifySession
	java := javabackend.NewJavaBackendWithFactory(func(_ context.Context, _ string, got backend.JavaConfig) (javabackend.JavaSession, error) {
		config = got
		session := &mcpVerifySession{}
		sessions = append(sessions, session)
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
		for _, session := range sessions {
			for _, got := range session.methods {
				if got == want {
					found = true
				}
			}
		}
		if !found {
			methods := make([]string, 0)
			for _, session := range sessions {
				methods = append(methods, session.methods...)
			}
			t.Fatalf("missing %s request: %v", want, methods)
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
	if response.Result.StructuredContent.Metrics.SchemaVersion != 0 {
		t.Fatalf("MCP Maven failure unexpectedly includes inline metrics: %#v", response.Result.StructuredContent.Metrics)
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
		fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_insert_structure","arguments":{"file":%q,"kind":"type","source":"type Item struct { ID string }"}}}`, filePath),
		fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_insert_structure","arguments":{"file":%q,"kind":"function","source":"func ProcessItem(i Item) error { return nil }"}}}`, filePath),
		fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_insert_structure","arguments":{"file":%q,"kind":"decl","source":"const DefaultLimit = 50"}}}`, filePath),
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
	for i, response := range responses {
		if got := response.Result.StructuredContent.Metrics; got.SchemaVersion != 0 || got.Phases != nil {
			t.Errorf("mutation %d unexpectedly includes inline metrics: %#v", i+1, got)
		}
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

	callMsg := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_insert_structure","arguments":{"file":%q,"kind":"function","source":"func broken( {"}}}`, filePath) + "\n"
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
	writeTestGoModule(t, dir)

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

type mcpExampleProperty struct {
	Type       string                        `json:"type"`
	Enum       []json.RawMessage             `json:"enum"`
	Items      *mcpExampleProperty           `json:"items"`
	Properties map[string]mcpExampleProperty `json:"properties"`
	Required   []string                      `json:"required"`
}

func mcpExampleValueMatches(value any, property mcpExampleProperty) bool {
	switch property.Type {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int(number))
	case "array":
		items, ok := value.([]any)
		if !ok || property.Items == nil {
			return false
		}
		for _, item := range items {
			if !mcpExampleValueMatches(item, *property.Items) {
				return false
			}
		}
		return true
	case "object":
		return mcpExampleObjectMatches(value, property)
	default:
		return false
	}
}

func mcpExampleObjectMatches(value any, property mcpExampleProperty) bool {
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for _, field := range property.Required {
		if _, ok := object[field]; !ok {
			return false
		}
	}
	for field, fieldValue := range object {
		fieldSchema, ok := property.Properties[field]
		if !ok || !mcpExampleValueMatches(fieldValue, fieldSchema) {
			return false
		}
	}
	return true
}

func TestMCPOperationExamplesMatchInputSchemas(t *testing.T) {
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-06-18\"}}\n" +
		"{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n"
	var output bytes.Buffer
	if err := mcp.NewServer("full", ".", &output).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]mcpExampleProperty `json:"properties"`
					Required   []string                      `json:"required"`
					Examples   []map[string]json.RawMessage  `json:"examples"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	responses := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(responses) == 0 {
		t.Fatal("tools/list produced no response")
	}
	if err := json.Unmarshal([]byte(responses[len(responses)-1]), &response); err != nil {
		t.Fatal(err)
	}

	exampleCount := 0
	for _, tool := range response.Result.Tools {
		for _, example := range tool.InputSchema.Examples {
			exampleCount++
			for _, required := range tool.InputSchema.Required {
				if _, ok := example[required]; !ok {
					t.Errorf("%s example is missing required parameter %q", tool.Name, required)
				}
			}
			for key, raw := range example {
				property, ok := tool.InputSchema.Properties[key]
				if !ok {
					t.Errorf("%s example includes unsupported parameter %q", tool.Name, key)
					continue
				}
				var value any
				if err := json.Unmarshal(raw, &value); err != nil {
					t.Errorf("%s example parameter %q is invalid JSON: %v", tool.Name, key, err)
					continue
				}
				validType := mcpExampleValueMatches(value, property)
				if !validType {
					t.Errorf("%s example parameter %q does not satisfy type %q", tool.Name, key, property.Type)
				}
				if len(property.Enum) > 0 {
					matched := false
					for _, allowed := range property.Enum {
						matched = matched || string(allowed) == string(raw)
					}
					if !matched {
						t.Errorf("%s example parameter %q is outside its enum", tool.Name, key)
					}
				}
			}
		}
	}
	if exampleCount == 0 {
		t.Fatal("tools/list did not advertise any operation examples")
	}
}

func TestFeedbackToolAdvertisesStructuredSchemaAndPrivacyGuidance(t *testing.T) {
	t.Parallel()
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("mutations-only", t.TempDir(), &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				InputSchema struct {
					Properties map[string]json.RawMessage `json:"properties"`
					Required   []string                   `json:"required"`
				} `json:"inputSchema"`
				OutputSchema json.RawMessage `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, feedback := range response.Result.Tools {
		if feedback.Name != "report_feedback" {
			continue
		}
		found = true
		for _, field := range []string{"intent", "interface", "command", "parameters", "observed_result", "unexpected_reason", "manual_touchups"} {
			if _, ok := feedback.InputSchema.Properties[field]; !ok {
				t.Errorf("feedback input schema is missing %q", field)
			}
			required := false
			for _, candidate := range feedback.InputSchema.Required {
				required = required || candidate == field
			}
			if !required {
				t.Errorf("feedback input schema does not require %q", field)
			}
		}
		for _, phrase := range []string{"open-source", "ask the user first", "proprietary", "intellectual property", "review for accuracy and sensitive content", "post manually as a GitHub issue", "does not save or post feedback"} {
			if !strings.Contains(feedback.Description, phrase) {
				t.Errorf("feedback tool description does not contain %q", phrase)
			}
		}
		if len(feedback.OutputSchema) == 0 {
			t.Error("report_feedback is missing its output schema")
		}
		break
	}
	if !found {
		t.Fatal("report_feedback is missing from tools/list")
	}
}

func TestReportFeedbackReturnsStructuredLocalDraft(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/feedback\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "docs", "SUBOPTIMAL_TOOLS.md")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		t.Fatal(err)
	}
	const originalLog = "existing local log\n"
	if err := os.WriteFile(logPath, []byte(originalLog), 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := `{"intent":"Rename a public function","interface":"mcp","command":"semantic_rename","parameters":{"symbol":"OldName","to":"NewName"},"observed_result":"The declaration changed but a reference did not.","unexpected_reason":"The rename should update all supported references.","manual_touchups":"None"}`
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"report_feedback","arguments":` + arguments + `}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", dir, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent struct {
				Result struct {
					Status   string `json:"status"`
					Date     string `json:"date"`
					Markdown string `json:"markdown"`
					Report   struct {
						Command    string         `json:"command"`
						Parameters map[string]any `json:"parameters"`
					} `json:"report"`
				} `json:"result"`
				Metrics map[string]any `json:"metrics"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.IsError {
		t.Fatalf("report_feedback returned an error: %s", out.String())
	}
	result := response.Result.StructuredContent.Result
	if result.Status != "draft_only" || result.Date == "" {
		t.Fatalf("feedback result status/date = %q/%q", result.Status, result.Date)
	}
	if result.Report.Command != "semantic_rename" || result.Report.Parameters["symbol"] != "OldName" {
		t.Errorf("feedback command details = %#v", result.Report)
	}
	for _, phrase := range []string{"Rename a public function", "semantic_rename", "The declaration changed", "The rename should update", "Manual touch-ups", "Review this draft for accuracy and sensitive content", "manually posting it as a GitHub issue", "not been saved or posted"} {
		if !strings.Contains(result.Markdown, phrase) {
			t.Errorf("feedback draft is missing %q", phrase)
		}
	}
	if response.Result.StructuredContent.Metrics != nil || len(response.Result.Content) != 1 || response.Result.Content[0].Text != result.Markdown {
		t.Fatalf("feedback response is missing its standard MCP envelope: %s", out.String())
	}
	gotLog, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotLog) != originalLog {
		t.Fatalf("report_feedback modified the workspace log: %q", gotLog)
	}
}

func TestReportFeedbackRejectsMissingCoreFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/feedbackvalidation\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := `{"intent":"Fix a tool issue","interface":"mcp","command":"semantic_verify","parameters":{},"observed_result":"Unexpected error","unexpected_reason":"The file is valid.","manual_touchups":""}`
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"report_feedback","arguments":` + arguments + `}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", dir, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Result.IsError || len(response.Result.Content) == 0 || !strings.Contains(response.Result.Content[0].Text, "manual_touchups") {
		t.Fatalf("report_feedback should reject missing manual touch-ups: %s", out.String())
	}
}

func TestMCPFullProfileListsFindReferences(t *testing.T) {
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-06-18\"}}\n" +
		"{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n"
	var output bytes.Buffer
	if err := mcp.NewServer("full", ".", &output).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "\"semantic_find_references\"") {
		t.Fatalf("tools/list response lacks semantic_find_references: %s", output.String())
	}
}

func TestMCPRejectsUnknownVerifyParamsBeforeFormatting(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	original := []byte("package main\nfunc main(){ println(\"ok\") }\n")
	if err := os.WriteFile(file, original, 0o600); err != nil {
		t.Fatal(err)
	}
	args := `{"files":["main.go"],"language":"go"}`
	input := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_verify","arguments":%s}}`+"\n", args)
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out).Serve(context.Background(), bytes.NewBufferString(input)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "supported keys") || !strings.Contains(out.String(), "path") {
		t.Fatalf("semantic_verify response = %s, want unknown-parameter error with supported keys", out.String())
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("semantic_verify formatted file before rejecting invalid params:\n%s", got)
	}
}

func TestMCPScaffoldFilePreservesPurposeHeader(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/scaffold\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "existing.go"), []byte("package service\n"), 0o600); err != nil {
		t.Fatalf("write sibling: %v", err)
	}

	const header = "// WHY: This file groups service adapters."
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_scaffold_file","arguments":{"file":"generated.go","package":"infer","purpose_header":"// WHY: This file groups service adapters."}}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", dir, &out).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}
	var response struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal MCP response: %v", err)
	}
	if response.Result.IsError {
		t.Fatalf("MCP scaffold returned an error: %s", out.String())
	}
	content, err := os.ReadFile(filepath.Join(dir, "generated.go"))
	if err != nil {
		t.Fatalf("read scaffolded file: %v", err)
	}
	want := header + "\n\npackage service\n"
	if string(content) != want {
		t.Fatalf("scaffolded content = %q, want %q", content, want)
	}
}

func TestMCPExpectedRootGuard(t *testing.T) {
	expectedRoot := t.TempDir()
	actualRoot := t.TempDir()
	marker := filepath.Join(actualRoot, "preserve.txt")
	if err := os.WriteFile(marker, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	rawParams, err := json.Marshal(map[string]any{"rootPath": actualRoot})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	srv := mcp.NewServer("full", expectedRoot, &out, mcp.WithExpectedRoot(expectedRoot))
	_, err = srv.Initialize(t.Context(), rawParams)
	if err == nil {
		t.Fatal("Initialize succeeded with a mismatched workspace root")
	}
	for _, want := range []string{"workspace root mismatch", backend.CanonicalWorkspaceRoot(expectedRoot), backend.CanonicalWorkspaceRoot(actualRoot)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Initialize error %q does not include %q", err, want)
		}
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "unchanged" {
		t.Errorf("mismatch changed fixture marker: contents=%q error=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(actualRoot, ".scratch")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("mismatch initialized fixture Go state: %v", err)
	}

	matchingRoot := t.TempDir()
	matchingServer := mcp.NewServer("full", matchingRoot, &out, mcp.WithExpectedRoot(filepath.Join(matchingRoot, ".")))
	result, err := matchingServer.Initialize(t.Context(), nil)
	if err != nil {
		t.Fatalf("Initialize rejected matching expected root: %v", err)
	}
	if got, want := result["workspaceRoot"], backend.CanonicalWorkspaceRoot(matchingRoot); got != want {
		t.Errorf("Initialize workspaceRoot = %v, want %q", got, want)
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("Initialize serverInfo = %#v, want object", result["serverInfo"])
	}
	for _, key := range []string{"binaryPath", "profile", "toolCatalog"} {
		if serverInfo[key] == "" {
			t.Errorf("Initialize serverInfo missing %q: %#v", key, serverInfo)
		}
	}
}

func TestMCPExpectedRootGuardDecodesWorkspaceFileURI(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fixture with spaces")
	if err := os.MkdirAll(root, 0o750); err != nil {
		t.Fatal(err)
	}
	rootURI := (&url.URL{Scheme: "file", Path: filepath.ToSlash(root)}).String()
	rawParams, err := json.Marshal(map[string]any{"rootUri": rootURI})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	srv := mcp.NewServer("full", root, &out, mcp.WithExpectedRoot(root))
	result, err := srv.Initialize(t.Context(), rawParams)
	if err != nil {
		t.Fatalf("Initialize rejected URI-escaped matching root %q: %v", rootURI, err)
	}
	if got, want := result["workspaceRoot"], backend.CanonicalWorkspaceRoot(root); got != want {
		t.Errorf("Initialize workspaceRoot = %v, want %q", got, want)
	}
}

func TestMCPExpectedRootMismatchCannotBeRetriedWithoutClientRoot(t *testing.T) {
	expectedRoot := t.TempDir()
	actualRoot := t.TempDir()
	rawParams, err := json.Marshal(map[string]any{"rootPath": actualRoot})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	srv := mcp.NewServer("full", expectedRoot, &out, mcp.WithExpectedRoot(expectedRoot))
	if _, err := srv.Initialize(t.Context(), rawParams); err == nil {
		t.Fatal("Initialize succeeded with a mismatched root")
	}
	if _, err := srv.Initialize(t.Context(), nil); err == nil {
		t.Fatal("Initialize retried without client root after a root mismatch")
	}
	if _, err := os.Stat(filepath.Join(expectedRoot, ".scratch")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("failed initialization created expected-root state: %v", err)
	}
}

func TestSafeDeleteRefusalReturnsMCPErrorAndStructuredChecks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/safedelete\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := "package safedelete\nfunc obsolete() {}\n"
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_remove_construct\",\"arguments\":{\"file\":\"main.go\",\"kind\":\"function\",\"symbol\":\"obsolete\",\"safe_delete\":true}}}\n"
	var output bytes.Buffer
	if err := mcp.NewServer("full", root, &output).Serve(t.Context(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Structured struct {
				ApplicationState string `json:"application_state"`
				Result           struct {
					Status  string   `json:"status"`
					Checks  []string `json:"checks_run"`
					Refusal string   `json:"refusal"`
				} `json:"result"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode MCP response: %v\n%s", err, output.String())
	}
	if !response.Result.IsError || response.Result.Structured.Result.Status != "refused" || response.Result.Structured.Result.Refusal == "" {
		t.Fatalf("safe delete response did not expose an explicit refusal: %+v", response.Result)
	}
	if response.Result.Structured.ApplicationState != "not_applied" {
		t.Fatalf("safe delete application_state = %q, want not_applied", response.Result.Structured.ApplicationState)
	}
	text := response.Result.Content[0].Text
	if !strings.HasPrefix(text, "Critical result (JSON): ") || !strings.Contains(text, response.Result.Structured.Result.Refusal) {
		t.Fatalf("safe delete text omitted refusal details: %q", text)
	}
	if len(response.Result.Structured.Result.Checks) != 3 || response.Result.Structured.Result.Checks[0] != "go_syntax_parse" || response.Result.Structured.Result.Checks[1] != "target_resolution" || response.Result.Structured.Result.Checks[2] != "safe_delete_scope_check" {
		t.Fatalf("checks_run = %v", response.Result.Structured.Result.Checks)
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != source {
		t.Fatalf("refused operation changed source: %q, %v", got, err)
	}
}

func TestMCPReusesGoDiagnosticsWithinInstance(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	root, counter := fixture.root, fixture.counter

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"go","check_only":true}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"go","check_only":true}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"semantic_replace_body","arguments":{"file":"value.go","symbol":"Value","body":"return 2"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"go","check_only":true}}}`,
	}, "\n") + "\n"
	for instance := 1; instance <= 2; instance++ {
		var output bytes.Buffer
		if err := mcp.NewServer("full", root, &output).Serve(t.Context(), strings.NewReader(input)); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) != 5 {
			t.Fatalf("responses = %s", output.String())
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
				t.Fatalf("failed response: %s", line)
			}
		}
		starts, err := os.ReadFile(counter)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(starts), "start\n"); got != instance {
			t.Fatalf("instance %d starts = %d", instance, got)
		}
	}
	content, err := os.ReadFile(filepath.Join(root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "return 2") {
		t.Fatalf("edit not published: %s", content)
	}
}

func TestMCPCancellationClosesIdleDiagnostics(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(func() {
		cancel()
		_ = inputWriter.Close()
		_ = outputReader.Close()
	})
	done := make(chan error, 1)
	go func() {
		err := mcp.NewServer("full", fixture.root, outputWriter).Serve(ctx, inputReader)
		_ = outputWriter.Close()
		done <- err
	}()
	decoder := json.NewDecoder(outputReader)
	for _, request := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"go","check_only":true}}}`,
	} {
		if _, err := io.WriteString(inputWriter, request+"\n"); err != nil {
			t.Fatal(err)
		}
		responseDone := make(chan error, 1)
		go func() {
			var response struct {
				Error  json.RawMessage `json:"error"`
				Result struct {
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			err := decoder.Decode(&response)
			if err == nil && (response.Error != nil || response.Result.IsError) {
				err = fmt.Errorf("failed MCP response: %+v", response)
			}
			responseDone <- err
		}()
		select {
		case err := <-responseDone:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("MCP response timed out")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Serve error = %v, want cancellation", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("idle Serve did not close diagnostics after cancellation")
	}
	pidBytes, err := os.ReadFile(fixture.pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		t.Fatal(err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = process.Release() }()
	if err := process.Signal(syscall.Signal(0)); !errors.Is(err, syscall.ESRCH) && !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("gopls process %d did not exit after canceled Serve: %v", pid, err)
	}
}

func TestMCPReusesGoSessionAcrossRefactorings(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	source := "package sample\n\nfunc Print(value int) { _ = value }\n\nfunc Emit() {\n\tPrint(1)\n}\n\nfunc Render(count int) {\n\tPrint(count + 1)\n}\n\nfunc Value() int {\n\tlocal := 2\n\treturn local\n}\n"
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "value.go"), []byte(source)); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "moved.go"), []byte("package sample\n")); err != nil {
		t.Fatal(err)
	}
	calls := []struct {
		name      string
		arguments map[string]any
	}{
		{"semantic_verify", map[string]any{"path": ".", "language": "go", "check_only": true}},
		{"semantic_rename", map[string]any{"file": "value.go", "symbol": "Value", "to": "Renamed"}},
		{"semantic_extract_function", map[string]any{"file": "value.go", "symbol": "emitOne", "in_function": "Emit", "kind": "function", "start_line": 6, "end_line": 6}},
		{"semantic_extract_variable", map[string]any{"file": "value.go", "symbol": "total", "in_function": "Render", "discriminator": "count + 1"}},
		{"semantic_inline_variable", map[string]any{"file": "value.go", "symbol": "local", "in_function": "Renamed"}},
		{"semantic_move_construct", map[string]any{"file": "value.go", "destination_file": "moved.go", "symbol": "emitOne"}},
		{"semantic_batch", map[string]any{"auto_organize_imports": true, "edits": []map[string]any{
			{"tool": "semantic_replace_body", "params": map[string]any{"file": "moved.go", "symbol": "emitOne", "body": "Print(3)"}},
			{"tool": "semantic_rename", "params": map[string]any{"file": "moved.go", "symbol": "emitOne", "to": "emitThree"}},
			{"tool": "semantic_rename", "params": map[string]any{"file": "value.go", "symbol": "Renamed", "to": "Value"}},
		}}},
		{"semantic_verify", map[string]any{"path": ".", "language": "go", "check_only": true}},
	}
	var input bytes.Buffer
	encoder := json.NewEncoder(&input)
	if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	for index, call := range calls {
		if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": index + 2, "method": "tools/call", "params": map[string]any{"name": call.name, "arguments": call.arguments}}); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	if err := mcp.NewServer("full", fixture.root, &output).Serve(t.Context(), &input); err != nil {
		t.Fatal(err)
	}
	responses := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(responses) != len(calls)+1 {
		t.Fatalf("responses = %s", output.String())
	}
	for _, line := range responses {
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
			t.Fatalf("failed response: %s", line)
		}
	}
	var moved struct {
		Result struct {
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(responses[6]), &moved); err != nil {
		t.Fatalf("decode move response: %v", err)
	}
	if moved.Result.Structured["file"] != "moved.go" || moved.Result.Structured["origin_file"] != "value.go" || moved.Result.Structured["symbol"] != "emitOne" {
		t.Fatalf("move continuation fields = %#v, want destination file, historical source, and live symbol", moved.Result.Structured)
	}
	starts, err := os.ReadFile(fixture.counter)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(starts), "start\n"); got != 1 {
		t.Fatalf("gopls starts = %d, want 1", got)
	}
	for file, expected := range map[string][]string{"value.go": {"emitThree()", "total := count + 1", "return int(2)", "func Value()"}, "moved.go": {"func emitThree()", "Print(3)"}} {
		content, err := os.ReadFile(filepath.Join(fixture.root, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range expected {
			if !strings.Contains(string(content), fragment) {
				t.Fatalf("%s missing %q: %s", file, fragment, content)
			}
		}
	}
}
