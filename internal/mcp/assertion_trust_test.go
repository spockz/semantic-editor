// Package mcp_test verifies assertion consent follows the selected request root without persisting to later calls.
package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/mcptest"
)

func TestAssertionModeTrustIsScopedToRequestRoot(t *testing.T) {
	t.Parallel()
	configured, original := assertionTrustWorkspace(t, t.TempDir())
	selected, _ := assertionTrustWorkspace(t, filepath.Join(configured, "selected"))
	var input, output bytes.Buffer
	for index, trusted := range []bool{true, false} {
		arguments := map[string]any{"workspace_root": selected, "file": "main_test.go", "mode": "relax", "trust_workspace": trusted}
		if !trusted {
			arguments["mode"] = "restrict"
		}
		request := map[string]any{"jsonrpc": "2.0", "id": index + 1, "method": "tools/call", "params": map[string]any{"name": "semantic_assertion_mode", "arguments": arguments}}
		if err := json.NewEncoder(&input).Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := mcptest.NewServer(t, "full", configured, &output).Serve(t.Context(), &input); err != nil {
		t.Fatal(err)
	}
	assertAssertionTrustResponses(t, output.Bytes())
	configuredSource, err := os.ReadFile(filepath.Join(configured, "main_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(configuredSource) != original {
		t.Fatalf("configured workspace changed: %s", configuredSource)
	}
	selectedSource, err := os.ReadFile(filepath.Join(selected, "main_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(selectedSource), `t.Error("x")`) || strings.Contains(string(selectedSource), `t.Fatal("x")`) {
		t.Fatalf("selected source did not retain the authorized change after refusal: %s", selectedSource)
	}
}

func assertionTrustWorkspace(t *testing.T, root string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	source := "package sample\n\nimport \"testing\"\n\nfunc TestCheck(t *testing.T) { t.Fatal(\"x\") }\n"
	for name, contents := range map[string]string{"go.mod": "module example.com/assertiontrust\n\ngo 1.24\n", "main_test.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root, source
}

func assertAssertionTrustResponses(t *testing.T, output []byte) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(output))
	for index := range 2 {
		var response struct {
			Error  json.RawMessage `json:"error"`
			Result struct {
				IsError           bool `json:"isError"`
				StructuredContent struct {
					Error string `json:"error"`
				} `json:"structuredContent"`
			} `json:"result"`
		}
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if len(response.Error) != 0 || response.Result.IsError != (index == 1) {
			t.Fatalf("assertion request %d has unexpected outcome: %s", index+1, output)
		}
		if index == 1 && !strings.Contains(response.Result.StructuredContent.Error, "workspace trust required") {
			t.Fatalf("untrusted request did not report trust refusal: %s", output)
		}
	}
}
