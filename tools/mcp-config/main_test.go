// These tests exercise the installation command because it is the user-facing contract for generated MCP configs.
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestInstallCommandRendersRepositoryTemplatesForWorkspacePath(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), `workspace with spaces and "quotes"`)
	prepareWorkspace(t, workspace)
	runInstallCommand(t, workspace, true)

	semeditPath := filepath.Join(workspace, "bin", "semedit")
	semeditNextPath := filepath.Join(workspace, "bin", "semedit-next")
	tomlData, err := os.ReadFile(filepath.Join(workspace, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var tomlConfig struct {
		MCPServers map[string]struct {
			Command string
			Args    []string
		} `toml:"mcp_servers"`
	}
	if decodeErr := toml.Unmarshal(tomlData, &tomlConfig); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if got := tomlConfig.MCPServers["semedit"].Command; got != semeditPath {
		t.Fatalf("TOML semedit command = %q, want %q", got, semeditPath)
	}
	if got := strings.Join(tomlConfig.MCPServers["semedit"].Args, " "); got != "mcp --profile full --live-reload" {
		t.Fatalf("TOML args changed: %q", got)
	}

	jsonData, err := os.ReadFile(filepath.Join(workspace, ".agents", "plugins", "workspace-servers", "mcp_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var jsonConfig struct {
		Servers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(jsonData, &jsonConfig); err != nil {
		t.Fatal(err)
	}
	if got := jsonConfig.Servers["semedit"].Command; got != semeditPath {
		t.Fatalf("JSON semedit command = %q, want %q", got, semeditPath)
	}
	if got := jsonConfig.Servers["semedit-next"].Command; got != semeditNextPath {
		t.Fatalf("JSON semedit-next command = %q, want %q", got, semeditNextPath)
	}
	if got := strings.Join(jsonConfig.Servers["gopls"].Args, " "); got != "mcp" {
		t.Fatalf("gopls args changed: %q", got)
	}
	if strings.Contains(string(tomlData), "{{") || strings.Contains(string(jsonData), "{{") {
		t.Fatal("generated config contains an unresolved placeholder")
	}
}

func TestInstallCommandRejectsMissingBinaryWithoutOutputs(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")
	prepareWorkspace(t, workspace)
	if err := os.Remove(filepath.Join(workspace, "bin", "semedit-next")); err != nil {
		t.Fatal(err)
	}
	runInstallCommand(t, workspace, false)
	assertNoGeneratedConfigs(t, workspace)
}

func TestInstallCommandRejectsInvalidTemplateWithoutOutputs(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")
	prepareWorkspace(t, workspace)
	previous := map[string][]byte{}
	for _, file := range configFiles {
		path := filepath.Join(workspace, filepath.FromSlash(file.output))
		previous[file.output] = []byte("existing config: " + file.output)
		if err := os.WriteFile(path, previous[file.output], 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, ".agents", "plugins", "workspace-servers", "mcp_config.json.example"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	runInstallCommand(t, workspace, false)
	for _, file := range configFiles {
		path := filepath.Join(workspace, filepath.FromSlash(file.output))
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(previous[file.output]) {
			t.Fatalf("failed install changed existing output %s: %q", file.output, got)
		}
	}
}

func runInstallCommand(t *testing.T, workspace string, wantSuccess bool) {
	t.Helper()
	command := exec.Command("go", "run", ".", "--workspace", workspace)
	command.Dir = filepath.Dir(sourceFile(t))
	output, err := command.CombinedOutput()
	if wantSuccess && err != nil {
		t.Fatalf("install command failed: %v\n%s", err, output)
	}
	if !wantSuccess && err == nil {
		t.Fatalf("install command unexpectedly succeeded:\n%s", output)
	}
}

func prepareWorkspace(t *testing.T, workspace string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(workspace, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	jsonDir := filepath.Join(workspace, ".agents", "plugins", "workspace-servers")
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"semedit", "semedit-next"} {
		if err := os.WriteFile(filepath.Join(workspace, "bin", name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	repositoryRoot := filepath.Dir(filepath.Dir(filepath.Dir(sourceFile(t))))
	for _, file := range configFiles {
		source := filepath.Join(repositoryRoot, filepath.FromSlash(file.template))
		destination := filepath.Join(workspace, filepath.FromSlash(file.template))
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func assertNoGeneratedConfigs(t *testing.T, workspace string) {
	t.Helper()
	for _, file := range configFiles {
		if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(file.output))); !os.IsNotExist(err) {
			t.Fatalf("generated output %s exists after failure (stat error: %v)", file.output, err)
		}
	}
}

func sourceFile(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return source
}
