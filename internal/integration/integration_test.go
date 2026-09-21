// Package integration_test verifies target-isolated native harness registration and ownership semantics.
package integration_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/integration"
)

func TestCopilotWorkspaceInstallConflictAndOwnedUninstall(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	req := integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary, Profile: "mutations-only"}
	result, err := integration.Install(req)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if result.Status != "installed" || !result.Changed {
		t.Fatalf("install result = %+v", result)
	}
	configPath := filepath.Join(workspace, ".vscode", "mcp.json")
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), `"semedit"`) || !strings.Contains(string(data), `"mutations-only"`) {
		t.Fatalf("config = %s", data)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".codex", "config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("copilot installation touched Codex configuration: %v", err)
	}
	result, err = integration.Install(req)
	if err != nil || result.Status != "already-installed" {
		t.Fatalf("idempotent install = %+v, %v", result, err)
	}
	conflict := req
	conflict.Binary = filepath.Join(workspace, "other-semedit")
	if err := os.WriteFile(conflict.Binary, []byte("binary"), 0o755); err != nil { // #nosec G306 -- executable fixture.
		t.Fatal(err)
	}
	if _, err := integration.Install(conflict); !errors.Is(err, integration.ErrConflict) {
		t.Fatalf("conflict error = %v", err)
	}
	result, err = integration.Uninstall(integration.StatusRequest{Target: req.Target, Scope: req.Scope, Workspace: workspace})
	if err != nil || result.Status != "uninstalled" {
		t.Fatalf("uninstall = %+v, %v", result, err)
	}
	data, err = os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"semedit"`) {
		t.Fatalf("expected owned registration to be removed: %s", data)
	}
}

func TestCodexPreservesUnrelatedConfiguration(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	configDir := filepath.Join(workspace, ".codex")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	original := "# keep this comment\n[model]\nname = \"example\"\n\n"
	if err := os.WriteFile(configPath, []byte(original), 0o640); err != nil { // #nosec G306 -- permission preservation fixture.
		t.Fatal(err)
	}
	_, err := integration.Install(integration.Request{Target: integration.TargetCodex, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary, Profile: "full"})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), original) || !strings.Contains(string(data), "[mcp_servers.semedit]") {
		t.Fatalf("configuration did not preserve unrelated data: %s", data)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
	_, err = integration.Uninstall(integration.StatusRequest{Target: integration.TargetCodex, Scope: integration.ScopeWorkspace, Workspace: workspace})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	data, err = os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("uninstall changed unrelated config: %s", data)
	}
}

func TestUninstallDoesNotRemoveUnownedRegistration(t *testing.T) {
	workspace := t.TempDir()
	configDir := filepath.Join(workspace, ".vscode")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	config := `{"servers":{"semedit":{"type":"stdio","command":"/preexisting","args":["mcp","--profile","full"]},"other":{"type":"stdio","command":"other"}}}`
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := integration.Uninstall(integration.StatusRequest{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace})
	if !errors.Is(err, integration.ErrNotOwned) {
		t.Fatalf("uninstall error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(configDir, "mcp.json")) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != config {
		t.Fatalf("unowned config changed: %s", data)
	}
}

func TestMalformedConfigurationFailsWithoutMutation(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	configDir := filepath.Join(workspace, ".vscode")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "mcp.json")
	original := []byte(`{"servers":`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := integration.Install(integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary})
	if err == nil {
		t.Fatal("expected malformed configuration error")
	}
	data, readErr := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if readErr != nil || string(data) != string(original) {
		t.Fatalf("malformed config mutated: %q, %v", data, readErr)
	}
}

func executable(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "semedit")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil { // #nosec G306 -- executable fixture.
		t.Fatal(err)
	}
	return path
}
