// Package mcp_test verifies symbol suggestions track retained Go staging overlays.
package mcp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/mcp"
	"github.com/spockz/semantic-editor/internal/mcptest"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

func TestStagedRenameSuggestionsFollowDropAndAbort(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		"go.mod":  "module example.com/stagesuggestions\n\ngo 1.27.1\n",
		"main.go": "package sample\n\nfunc ParseConfig() {}\n",
	} {
		if err := pipeline.WriteAtomic(filepath.Join(root, path), []byte(contents)); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	var output bytes.Buffer
	server := mcptest.NewServer(t, "full", root, &output, mcp.WithToolProfile("full"), mcp.WithGoBaseDir(t.TempDir()))
	call := func(id int, name string, args map[string]any) map[string]any {
		t.Helper()
		request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		if err != nil {
			t.Fatalf("encode %s request: %v", name, err)
		}
		output.Reset()
		if err := server.Serve(t.Context(), strings.NewReader(string(request)+"\n")); err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		var response map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
			t.Fatalf("decode %s response: %v (%s)", name, err, output.String())
		}
		return response
	}
	content := func(response map[string]any) map[string]any {
		t.Helper()
		result, _ := response["result"].(map[string]any)
		structured, _ := result["structuredContent"].(map[string]any)
		return structured
	}
	call(1, "semantic_stage", map[string]any{})
	renamed := call(2, "refactor_rename", map[string]any{"symbol": "ParseConfig", "to": "ReadConfig", "file": "main.go"})
	stageReceipt, _ := content(renamed)["stage_receipt"].(map[string]any)
	stepID, _ := stageReceipt["step_id"].(string)
	if stepID == "" {
		t.Fatalf("rename omitted stage receipt step id: %#v", renamed)
	}
	assertSuggestion := func(id int, want string) {
		t.Helper()
		response := call(id, "refactor_rename", map[string]any{"symbol": "ParseConfg", "to": fmt.Sprintf("Retry%d", id), "file": "main.go"})
		candidates, _ := content(response)["suggestions"].([]any)
		if len(candidates) == 0 {
			t.Fatalf("missing suggestion for overlay state %q: %#v", want, response)
		}
		first, _ := candidates[0].(map[string]any)
		args, _ := first["arguments"].(map[string]any)
		if args["symbol"] != want {
			t.Fatalf("suggested symbol = %#v, want %q: %#v", args["symbol"], want, response)
		}
	}
	assertSuggestion(3, "ReadConfig")
	dropped := call(4, "semantic_stage_drop", map[string]any{"step_id": stepID})
	if dropped["error"] != nil {
		t.Fatalf("drop staged rename: %#v", dropped)
	}
	assertSuggestion(5, "ParseConfig")
	aborted := call(6, "semantic_stage_abort", map[string]any{})
	if aborted["error"] != nil {
		t.Fatalf("abort stage: %#v", aborted)
	}
	assertSuggestion(7, "ParseConfig")
	assertDiscardedReplaySuggestions(t, call, content, assertSuggestion)
}

func assertDiscardedReplaySuggestions(t *testing.T, call func(int, string, map[string]any) map[string]any, content func(map[string]any) map[string]any, assertSuggestion func(int, string)) {
	t.Helper()
	call(8, "semantic_stage", map[string]any{})
	first := call(9, "refactor_rename", map[string]any{"symbol": "ParseConfig", "to": "ParseConfg", "file": "main.go"})
	firstReceipt, _ := content(first)["stage_receipt"].(map[string]any)
	firstStep, _ := firstReceipt["step_id"].(string)
	if firstStep == "" {
		t.Fatalf("first chain rename omitted receipt: %#v", first)
	}
	call(10, "refactor_rename", map[string]any{"symbol": "ParseConfg", "to": "ReadConfig", "file": "main.go"})
	failedDrop := content(call(11, "semantic_stage_drop", map[string]any{"step_id": firstStep}))
	if failedDrop["error"] == nil || failedDrop["suggestions"] != nil {
		t.Fatalf("discarded replay alternatives escaped: %#v", failedDrop)
	}
	if strings.Contains(fmt.Sprint(failedDrop["error"]), "Suggested target:") {
		t.Fatalf("discarded replay alternatives escaped in error text: %#v", failedDrop)
	}
	assertSuggestion(12, "ReadConfig")
	call(13, "semantic_stage_abort", map[string]any{})
}
