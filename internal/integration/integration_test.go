// Package integration_test verifies target-isolated native harness registration and ownership semantics.
package integration_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bytes"
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
	configPath := filepath.Join(workspace, ".mcp.json")
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), `"mcpServers"`) || !strings.Contains(string(data), `"type": "local"`) || !strings.Contains(string(data), `"tools"`) || !strings.Contains(string(data), `"*"`) || !strings.Contains(string(data), `"mutations-only"`) {
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

func TestCopilotUserScopeUsesCopilotHome(t *testing.T) {
	workspace := t.TempDir()
	copilotHome := filepath.Join(workspace, "copilot-home")
	t.Setenv("COPILOT_HOME", copilotHome)
	binary := executable(t, workspace)
	result, err := integration.Install(integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeUser, Binary: binary})
	if err != nil {
		t.Fatalf("user install: %v", err)
	}
	expected := filepath.Join(copilotHome, "mcp-config.json")
	if result.Path != expected {
		t.Fatalf("user config path = %q, want %q", result.Path, expected)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("user config missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".mcp.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("user install touched workspace config: %v", err)
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
	original := "# keep this comment\n[model] # keep this header comment\nname = \"example\"\n\n"
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

func TestCodexRejectsAmbiguousTargetTables(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	configDir := filepath.Join(workspace, ".codex")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	duplicate := "[mcp_servers.semedit]\ncommand = \"/one\"\nargs = [\"mcp\", \"--profile\", \"full\"]\n\n[mcp_servers.semedit]\ncommand = \"/two\"\nargs = [\"mcp\", \"--profile\", \"full\"]\n"
	if err := os.WriteFile(configPath, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := integration.Install(integration.Request{Target: integration.TargetCodex, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary}); err == nil {
		t.Fatal("expected duplicate target table error")
	}
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil || string(data) != duplicate {
		t.Fatalf("duplicate config mutated: %q, %v", data, err)
	}
	array := "[[mcp_servers.semedit]]\ncommand = \"/one\"\nargs = [\"mcp\"]\n"
	if err := os.WriteFile(configPath, []byte(array), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := integration.Install(integration.Request{Target: integration.TargetCodex, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary}); err == nil {
		t.Fatal("expected array target table error")
	}
}

func TestCodexAcceptsQuotedTargetHeader(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	configDir := filepath.Join(workspace, ".codex")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	config := "[mcp_servers.'semedit']\ncommand = \"/old\"\nargs = [\"mcp\", \"--profile\", \"full\"]\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := integration.Install(integration.Request{Target: integration.TargetCodex, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary, Replace: true}); err != nil {
		t.Fatalf("quoted target install: %v", err)
	}
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil || !strings.Contains(string(data), binary) {
		t.Fatalf("quoted target was not replaced: %q, %v", data, err)
	}
}

func TestUninstallDoesNotRemoveUnownedRegistration(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	config := `{"mcpServers":{"semedit":{"type":"stdio","command":"/preexisting","args":["mcp","--profile","full"]},"other":{"type":"stdio","command":"other"}}}`
	configPath := filepath.Join(workspace, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := integration.Uninstall(integration.StatusRequest{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace})
	if !errors.Is(err, integration.ErrNotOwned) {
		t.Fatalf("uninstall error = %v", err)
	}
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
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
	if err := os.MkdirAll(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(workspace, ".mcp.json")
	original := []byte(`{"servers":`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := integration.Install(integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary})
	if err == nil {
		t.Fatal("expected malformed configuration error")
	}
	data, readErr := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if readErr != nil || !bytes.Equal(data, original) {
		t.Fatalf("malformed config mutated: %q, %v", data, readErr)
	}
}

func TestDryRunParsesExistingState(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	req := integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary, Profile: "full"}
	if _, err := integration.Install(req); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(workspace, ".mcp.json")) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	result, err := integration.Install(integration.Request{Target: req.Target, Scope: req.Scope, Workspace: workspace, Binary: binary, Profile: req.Profile, DryRun: true})
	if err != nil || result.Status != "already-installed" || result.Changed {
		t.Fatalf("matching dry-run = %+v, %v", result, err)
	}
	conflict := req
	conflict.Binary = filepath.Join(workspace, "other")
	if err := os.WriteFile(conflict.Binary, []byte("x"), 0o755); err != nil { // #nosec G306 -- executable fixture.
		t.Fatal(err)
	}
	conflict.DryRun = true
	if _, err := integration.Install(conflict); !errors.Is(err, integration.ErrConflict) {
		t.Fatalf("conflicting dry-run = %v", err)
	}
	after, err := os.ReadFile(filepath.Join(workspace, ".mcp.json")) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil || !bytes.Equal(after, before) {
		t.Fatalf("dry-run mutated config: %q, %v", after, err)
	}
}

func TestMalformedDryRunDoesNotWrite(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	configPath := filepath.Join(workspace, ".mcp.json")
	original := []byte(`{"mcpServers":`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := integration.Install(integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary, DryRun: true})
	if err == nil {
		t.Fatal("expected malformed dry-run error")
	}
	after, readErr := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if readErr != nil || !bytes.Equal(after, original) {
		t.Fatalf("malformed dry-run mutated config: %q, %v", after, readErr)
	}
}

func TestCopilotCommentsFailSafelyAndUnrelatedStatusIsNotInstalled(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(workspace, ".mcp.json")
	commented := []byte("{\n  // preserve this comment\n  \"mcpServers\": {\"other\": {\"command\": \"other\"}}\n}\n")
	if err := os.WriteFile(configPath, commented, 0o600); err != nil {
		t.Fatal(err)
	}
	binary := executable(t, workspace)
	_, err := integration.Install(integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary})
	if err == nil {
		t.Fatal("expected unsupported comments error")
	}
	after, readErr := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if readErr != nil || !bytes.Equal(after, commented) {
		t.Fatalf("commented config mutated: %q, %v", after, readErr)
	}
	status, err := integration.Status(integration.StatusRequest{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace})
	if err != nil || status.Status != "malformed" {
		t.Fatalf("commented status = %+v, %v", status, err)
	}
	if err := os.WriteFile(configPath, []byte(`{"mcpServers":{"other":{"command":"other"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err = integration.Status(integration.StatusRequest{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace})
	if err != nil || status.Status != "not-installed" {
		t.Fatalf("unrelated status = %+v, %v", status, err)
	}
}

func TestEditedOwnedRegistrationIsNotRemoved(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	req := integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary}
	if _, err := integration.Install(req); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(workspace, ".mcp.json")
	data, err := os.ReadFile(configPath) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"full"`, `"mutations-only"`, 1))
	if err := os.WriteFile(configPath, data, 0o600); err != nil { // #nosec G703 -- test path is created inside t.TempDir.
		t.Fatal(err)
	}
	_, err = integration.Uninstall(integration.StatusRequest{Target: req.Target, Scope: req.Scope, Workspace: workspace})
	if !errors.Is(err, integration.ErrNotOwned) {
		t.Fatalf("edited registration uninstall = %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestOwnershipFailureRollsBackConfiguration(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	if err := os.MkdirAll(filepath.Join(workspace, ".semedit"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, ".semedit", "copilot-workspace.json"), 0o750); err != nil {
		t.Fatal(err)
	}
	_, err := integration.Install(integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary})
	if err == nil {
		t.Fatal("expected ownership write failure")
	}
	if _, statErr := os.Stat(filepath.Join(workspace, ".mcp.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("config remained after ownership failure: %v", statErr)
	}
}

func TestOwnershipCleanupFailureRollsBackUninstall(t *testing.T) {
	workspace := t.TempDir()
	binary := executable(t, workspace)
	req := integration.Request{Target: integration.TargetCopilot, Scope: integration.ScopeWorkspace, Workspace: workspace, Binary: binary}
	if _, err := integration.Install(req); err != nil {
		t.Fatal(err)
	}
	ledgerDir := filepath.Join(workspace, ".semedit")
	if err := os.Chmod(ledgerDir, 0o500); err != nil { // #nosec G302 -- restrict fixture directory to force cleanup failure.
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(ledgerDir, 0o750) }() // #nosec G302 -- restore fixture permissions.
	if _, err := integration.Uninstall(integration.StatusRequest{Target: req.Target, Scope: req.Scope, Workspace: workspace}); err == nil {
		t.Fatal("expected ownership cleanup failure")
	}
	data, err := os.ReadFile(filepath.Join(workspace, ".mcp.json")) // #nosec G304 -- test path is created inside t.TempDir.
	if err != nil || !strings.Contains(string(data), "mcpServers") {
		t.Fatalf("uninstall cleanup failure lost registration: %q, %v", data, err)
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
