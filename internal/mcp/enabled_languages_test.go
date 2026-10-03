// Package mcp_test proves enabled-language policies across MCP requests and batches.

package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/backend"
	gobackend "semedit/internal/backend/golang"
	javabackend "semedit/internal/backend/java"
	"semedit/internal/mcp"
)

type enabledLanguageGoBackend struct {
	gobackend.GoBackend

	verifyCalls int
}

func decodeEnabledVerifyCoverage(t *testing.T, payload []byte) (bool, map[string]string, map[string]string) {
	t.Helper()
	var response struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Result string `json:"result"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatal(err)
	}
	text := response.Result.StructuredContent.Result
	const marker = "Verification report: "
	_, after, ok := strings.Cut(text, marker)
	if !ok {
		t.Fatalf("verification report is missing: %s", text)
	}
	var report struct {
		Coverage []struct {
			Language string `json:"language"`
			Status   string `json:"status"`
		} `json:"coverage"`
		Normalize struct {
			Results []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"results"`
		} `json:"normalize"`
	}
	if err := json.Unmarshal([]byte(after), &report); err != nil {
		t.Fatalf("decode verification report: %v\n%s", err, text)
	}
	coverage := make(map[string]string, len(report.Coverage))
	hooks := make(map[string]string, len(report.Normalize.Results))
	for _, item := range report.Coverage {
		coverage[item.Language] = item.Status
	}
	for _, item := range report.Normalize.Results {
		hooks[item.ID] = item.Status
	}
	return response.Result.IsError, coverage, hooks
}

func (b *enabledLanguageGoBackend) Verify(context.Context, backend.VerifyRequest) ([]backend.Diagnostic, error) {
	b.verifyCalls++
	return nil, nil
}

func TestMCPVerifyEnabledLanguagesRestrictsAutoCoverage(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":      "module example.com/enabled\n\ngo 1.24\n",
		"main.go":     "package main\nfunc main() {}\n",
		"Widget.java": "class Widget {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	goBackend := &enabledLanguageGoBackend{}
	javaStarts := 0
	javaBackend := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		javaStarts++
		return &mcpVerifySession{}, nil
	})
	registry, err := backend.NewRegistry(goBackend, javaBackend)
	if err != nil {
		t.Fatal(err)
	}
	service := backend.NewService(registry)
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"auto","check_only":true}}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out, mcp.WithService(service), mcp.WithEnabledLanguages("go")).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `"isError":true`) {
		t.Fatalf("auto verification failed: %s", out.String())
	}
	if goBackend.verifyCalls != 1 || javaStarts != 0 {
		t.Fatalf("backend calls: Go=%d Java=%d, want Go=1 Java=0", goBackend.verifyCalls, javaStarts)
	}
	isError, coverage, _ := decodeEnabledVerifyCoverage(t, out.Bytes())
	if isError || coverage["go"] != "verified" || coverage["java"] != "disabled" {
		t.Fatalf("auto coverage = %#v, isError=%t; want Go verified and Java disabled", coverage, isError)
	}
}

func TestMCPVerifyRejectsDisabledLanguageSelectorsAndKeepsDirectoryScopes(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":               "module example.com/enabled\n\ngo 1.24\n",
		"main.go":              "package main\nfunc main() {}\n",
		"Widget.java":          "class Widget {}\n",
		"src.java/Widget.java": "class NestedWidget {}\n",
		"src.java/main.go":     "package main\nfunc nested() {}\n",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	goBackend := &enabledLanguageGoBackend{}
	javaStarts := 0
	javaBackend := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		javaStarts++
		return &mcpVerifySession{}, nil
	})
	registry, err := backend.NewRegistry(goBackend, javaBackend)
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"java","trust_workspace":true,"jdtls_home":"/jdtls","check_only":true}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","file":"Widget.java","trust_workspace":true,"jdtls_home":"/jdtls","check_only":true}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":"Widget.java","trust_workspace":true,"jdtls_home":"/jdtls","check_only":true}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":"src.java","language":"auto","check_only":true}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out, mcp.WithService(backend.NewService(registry)), mcp.WithEnabledLanguages("go")).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	responses := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(responses) != 4 {
		t.Fatalf("MCP responses = %s", out.String())
	}
	for index := range 3 {
		var response struct {
			ID     int `json:"id"`
			Result struct {
				IsError bool `json:"isError"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(responses[index]), &response); err != nil {
			t.Fatal(err)
		}
		if !response.Result.IsError || response.ID != index+1 || len(response.Result.Content) == 0 || !strings.Contains(response.Result.Content[0].Text, "disabled by --enabled-languages") {
			t.Fatalf("selector request %d response = %s, want its own disabled-language error", index+1, responses[index])
		}
	}
	isError, coverage, _ := decodeEnabledVerifyCoverage(t, []byte(responses[3]))
	if isError || coverage["go"] != "verified" || coverage["java"] != "disabled" {
		t.Fatalf("src.java directory response coverage = %#v, isError=%t; want Go verified and Java disabled", coverage, isError)
	}
	if javaStarts != 0 || goBackend.verifyCalls != 1 {
		t.Fatalf("backend calls: Go=%d Java=%d, want Go=1 Java=0 for src.java directory", goBackend.verifyCalls, javaStarts)
	}
}

func TestMCPBatchJavaOnlySkipsGoBaselineAndVerification(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":      "this is deliberately not a valid Go module\n",
		"main.go":     "package main\nfunc main() {}\n",
		"Widget.java": "class Widget {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	javaSessions := 0
	javaBackend := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		javaSessions++
		if javaSessions == 1 {
			return mcpJavaSession{}, nil
		}
		return &mcpVerifySession{}, nil
	})
	registry, err := backend.NewRegistry(javaBackend)
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","initializationOptions":{"trust_workspace":true}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"semantic_batch","arguments":{"trust_workspace":true,"edits":[{"tool":"semantic_rename","params":{"symbol":"Widget","to":"Gadget","file":"Widget.java","language":"java","trust_workspace":true,"jdtls_home":"/jdtls"}}]}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out, mcp.WithService(backend.NewService(registry)), mcp.WithEnabledLanguages("java")).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Result mcp.BatchResponse `json:"result"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	responses := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(responses) != 2 {
		t.Fatalf("MCP responses = %s", out.String())
	}
	if err := json.Unmarshal([]byte(responses[1]), &response); err != nil {
		t.Fatal(err)
	}
	batch := response.Result.StructuredContent.Result
	if response.Result.IsError || batch.Status != "ok" {
		t.Fatalf("Java-only batch failed (a Go baseline would fail on malformed go.mod): %s", out.String())
	}
	if batch.DiagnosticDelta != nil || batch.DiagnosticDeltaNote == "" {
		t.Fatalf("disabled Go baseline reported a normal delta: %+v", batch)
	}
	if batch.Verification == nil || batch.Verification.Failure != "" {
		t.Fatalf("final Java verification did not complete: %+v", batch.Verification)
	}
	coverage := map[string]string{}
	for _, item := range batch.Verification.Coverage {
		coverage[item.Language] = item.Status
	}
	if coverage["java"] != "verified" || coverage["go"] != "disabled" {
		t.Fatalf("final coverage = %#v, want Java verified and Go disabled", coverage)
	}
	updated, err := os.ReadFile(filepath.Join(root, "Widget.java"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "Gadget") {
		t.Fatalf("Java batch rename was not published: %s", updated)
	}
}

func TestMCPVerifyEnabledLanguagesFiltersLSPHooksAndKeepsShellHooks(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":      "module example.com/hooks\n\ngo 1.24\n",
		"main.go":     "package main\nfunc main() {}\n",
		"Widget.java": "class Widget {}\n",
		".semedit.yaml": `version: 1
verify:
  normalize:
    - id: java-organize
      languages: [java]
      lsp:
        language: java
        action_kind: source.organizeImports
    - id: generic-shell
      languages: [go, java]
      shell:
        executable: /bin/sh
        args: ["-c"]
        script: 'printf ran >> "$SEMEDIT_HOOK_MARKER"'
`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	marker := filepath.Join(root, "shell-hook-ran")
	t.Setenv("SEMEDIT_HOOK_MARKER", marker)
	goBackend := &enabledLanguageGoBackend{}
	javaStarts := 0
	javaBackend := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		javaStarts++
		return &mcpVerifySession{}, nil
	})
	registry, err := backend.NewRegistry(goBackend, javaBackend)
	if err != nil {
		t.Fatal(err)
	}
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_verify","arguments":{"path":".","language":"auto","trust_workspace":true,"jdtls_home":"/jdtls"}}}` + "\n"
	var out bytes.Buffer
	if err := mcp.NewServer("full", root, &out, mcp.WithService(backend.NewService(registry)), mcp.WithEnabledLanguages("go")).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `"isError":true`) {
		t.Fatalf("verification with an allowed generic shell hook failed: %s", out.String())
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("allowed shell hook did not run: %v", err)
	}
	if goBackend.verifyCalls != 1 || javaStarts != 0 {
		t.Fatalf("backend calls: Go=%d Java=%d, want Go=1 Java=0", goBackend.verifyCalls, javaStarts)
	}
	isError, _, hooks := decodeEnabledVerifyCoverage(t, out.Bytes())
	if isError || hooks["java-organize"] != "skipped" || hooks["generic-shell"] != "succeeded" {
		t.Fatalf("normalization hook statuses = %#v, isError=%t", hooks, isError)
	}
}
