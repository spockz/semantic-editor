// Package main verifies Copilot's fixture-local runtime, MCP configuration, and sandbox boundary.
// WHY: The runtime redirects account state and model access, so regressions must remain visible without launching the live CLI.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopilotModelBaseURL(t *testing.T) {
	t.Setenv(copilotBaseURLEnv, "http://copilot.local/v1/")
	t.Setenv(openCodeQwenBaseURLEnv, "http://opencode.local/v1")
	if got := copilotModelBaseURL(); got != "http://copilot.local/v1" {
		t.Fatalf("copilotModelBaseURL() = %q, want Copilot override", got)
	}
	t.Setenv(copilotBaseURLEnv, "")
	if got := copilotModelBaseURL(); got != "http://opencode.local/v1" {
		t.Fatalf("copilotModelBaseURL() = %q, want OpenCode fallback", got)
	}
	t.Setenv(openCodeQwenBaseURLEnv, "")
	if got := copilotModelBaseURL(); got != strings.TrimRight(openCodeQwenBaseURL, "/") {
		t.Fatalf("copilotModelBaseURL() = %q, want default %q", got, openCodeQwenBaseURL)
	}
}

func TestSanitizedCopilotEnvironment(t *testing.T) {
	source := []string{"PATH=/bin", "COPILOT_GITHUB_TOKEN=secret", "GITHUB_TOKEN=secret", "GH_TOKEN=secret", "OPENAI_API_KEY=secret", "ANTHROPIC_API_KEY=secret", "AZURE_OPENAI_API_KEY=secret", "MALFORMED", "LANG=en_US.UTF-8"}
	got := make(map[string]string)
	for _, entry := range sanitizedCopilotEnvironment(source) {
		key, value, found := strings.Cut(entry, "=")
		if found {
			got[key] = value
		}
	}
	if got["PATH"] != "/bin" || got["LANG"] != "en_US.UTF-8" {
		t.Fatalf("sanitized environment lost ordinary process settings: %#v", got)
	}
	for _, key := range []string{"COPILOT_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "AZURE_OPENAI_API_KEY"} {
		if _, found := got[key]; found {
			t.Errorf("sanitized environment retained credential variable %s", key)
		}
	}
}

func TestCopilotCloudCredentialPrecedence(t *testing.T) {
	for _, test := range []struct {
		name        string
		environment []string
		want        string
	}{
		{name: "copilot token first", environment: []string{"GITHUB_TOKEN=github", "GH_TOKEN=gh", "COPILOT_GITHUB_TOKEN=copilot"}, want: "copilot"},
		{name: "gh token second", environment: []string{"GITHUB_TOKEN=github", "GH_TOKEN=gh"}, want: "gh"},
		{name: "github token fallback", environment: []string{"GITHUB_TOKEN=github"}, want: "github"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := copilotCloudCredential(test.environment)
			if err != nil || got != test.want {
				t.Fatalf("copilotCloudCredential() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
	if _, err := copilotCloudCredential([]string{"GITHUB_TOKEN=", "GH_TOKEN=  "}); err == nil || !strings.Contains(err.Error(), "COPILOT_GITHUB_TOKEN") {
		t.Fatalf("missing cloud credential error = %v, want actionable environment variable names", err)
	}
}

func TestCopilotProviderEnvironmentSeparatesLocalAndCloud(t *testing.T) {
	localTarget := Target{Harness: string(HarnessCopilot), Model: "local/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest"}
	local := copilotProviderEnvironment(localTarget, true, "http://127.0.0.1:43001/v1", &localModelLimits{Context: 32768, Output: 4096}, "ignored")
	if local[copilotModelEnv] != "Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest" || local[copilotProviderWireModelEnv] != local[copilotModelEnv] {
		t.Fatalf("local environment does not select its wire model: %#v", local)
	}
	if local[copilotOfflineEnv] != "true" || local[copilotProviderBaseURLEnv] != "http://127.0.0.1:43001/v1" || local[copilotProviderAPIKeyEnv] != "" {
		t.Fatalf("local environment missing offline BYOK settings: %#v", local)
	}
	if _, found := local["COPILOT_GITHUB_TOKEN"]; found {
		t.Fatalf("local environment retained cloud credential: %#v", local)
	}

	cloudTarget := Target{Harness: string(HarnessCopilot), Model: "gpt-5.4"}
	cloud := copilotProviderEnvironment(cloudTarget, false, copilotCloudAPIBaseURL, nil, "fixture-token")
	if cloud[copilotModelEnv] != cloudTarget.Model || cloud["COPILOT_GITHUB_TOKEN"] != "fixture-token" {
		t.Fatalf("cloud environment missing model or selected credential: %#v", cloud)
	}
	for _, localOnly := range []string{copilotOfflineEnv, copilotProviderTypeEnv, copilotProviderBaseURLEnv, copilotProviderAPIKeyEnv, copilotMaxPromptTokensEnv} {
		if _, found := cloud[localOnly]; found {
			t.Errorf("cloud environment retained local-only setting %s", localOnly)
		}
	}
}

func TestCopilotMCPConfig(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fixture")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{baseScratchDir: filepath.Join(base, ".scratch", "benchmark"), mcpServerInstructionsMode: MCPServerInstructionsNone}
	config, err := copilotMCPConfig(runner, root, root)
	if err != nil {
		t.Fatal(err)
	}
	server, found := config.MCPServers["semedit"]
	if !found {
		t.Fatalf("Copilot MCP config has no semedit server: %#v", config)
	}
	if server.Type != "local" || !strings.HasSuffix(server.Command, filepath.Join("bin", "semedit-next")) {
		t.Fatalf("unexpected semedit command configuration: %#v", server)
	}
	joined := strings.Join(server.Args, " ")
	for _, want := range []string{"--expected-root " + root, "--go-base-dir " + filepath.Join(root, ".scratch", "go"), "--metrics-file " + benchmarkMCPMetricsPath(root), "--instructions "} {
		if !strings.Contains(joined, want) {
			t.Errorf("semedit MCP arguments %q do not contain %q", joined, want)
		}
	}
	if len(server.Tools) != 1 || server.Tools[0] != "*" {
		t.Fatalf("semedit MCP server should expose all registered tools: %#v", server.Tools)
	}
}

func TestAppendCopilotRuntimeRuleIsNarrow(t *testing.T) {
	runtimeDir := filepath.Join(string(filepath.Separator), "Users", "worker", "Library", "Caches", "copilot", "pkg", "darwin-arm64", "1.0.88")
	profile, err := appendCopilotRuntimeRule([]byte("(deny default)\n"), runtimeDir)
	if err != nil {
		t.Fatal(err)
	}
	text := string(profile)
	if !strings.Contains(text, "(allow file-read* (subpath \""+runtimeDir+"\"))") {
		t.Fatalf("profile does not permit the exact runtime bundle: %s", text)
	}
	for _, broad := range []string{"(subpath \"/Users/worker\")", "(subpath \"/Users/worker/Library\")", "(allow network-outbound"} {
		if strings.Contains(text, broad) {
			t.Errorf("profile unexpectedly contains broad permission %q: %s", broad, text)
		}
	}
	if _, err := appendCopilotRuntimeRule(nil, string(filepath.Separator)); err == nil {
		t.Fatal("appendCopilotRuntimeRule accepted filesystem root")
	}
}

func TestAppendCopilotCloudNetworkRules(t *testing.T) {
	profile, err := appendCopilotCloudNetworkRules([]byte("(version 1)\n(deny default)\n"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(profile)
	for _, rule := range []string{`(allow network-outbound (remote tcp "*:443"))`, `(allow network-outbound (remote udp "*:53"))`, `(allow mach-lookup (global-name "com.apple.trustd"))`, `(allow mach-lookup (global-name "com.apple.SystemConfiguration.configd"))`, `(allow file-read-metadata (literal "/var"))`, `(allow network-outbound (literal "/private/var/run/mDNSResponder"))`} {
		if !strings.Contains(text, rule) {
			t.Errorf("cloud sandbox lacks required egress rule %q: %s", rule, text)
		}
	}
	for _, broad := range []string{"(allow network-outbound)\n", "(allow file-read* (subpath \"/Users\"))", "(allow file-write*"} {
		if strings.Contains(text, broad) {
			t.Errorf("cloud egress unexpectedly broadens local filesystem permissions %q: %s", broad, text)
		}
	}
	if _, err := appendCopilotCloudNetworkRules(nil); err == nil {
		t.Fatal("cloud network rules accepted an empty base profile")
	}
}

func TestNewCopilotCommandUsesACPAndSandbox(t *testing.T) {
	base := t.TempDir()
	profileDir := filepath.Join(base, ".opencode-sandboxes")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(profileDir, "copilot.sb")
	if err := os.WriteFile(profilePath, []byte("(version 1)\n(deny default)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{baseScratchDir: base, localProviderSandboxProfileFactory: func(_, _, _ string) (string, string, error) {
		return "/usr/bin/sandbox-exec", profilePath, nil
	}}
	workDir := filepath.Join(base, "fixture")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := Target{Harness: string(HarnessCopilot), Model: ("local/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest")}
	prepared := &copilotPreparedRuntime{
		workDir: workDir, target: target, arm: ArmBaseline,
		executable: "/opt/homebrew/Caskroom/copilot-cli/1.0.88/copilot",
		runtimeDir: "/Users/worker/Library/Caches/copilot/pkg/darwin-arm64/1.0.88",
		modelURL:   "http://127.0.0.1:43001/v1", env: []string{"HOME=" + filepath.Join(workDir, ".scratch", "copilot", "home")},
	}
	cmd, err := runner.newCopilotCommand(context.Background(), workDir, target, ArmBaseline, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Cancel != nil {
		t.Fatal("Copilot process lifetime should not inherit the task context")
	}
	if cmd.Path != "/usr/bin/sandbox-exec" || cmd.Dir != workDir {
		t.Fatalf("Copilot command is not sandboxed in the fixture: path=%q dir=%q", cmd.Path, cmd.Dir)
	}
	args := strings.Join(cmd.Args, " ")
	for _, want := range []string{"-f " + profilePath, prepared.executable, "--acp", "--no-auto-update", "--disable-builtin-mcps", "--no-ask-user", "--no-remote", "--no-remote-export", "--allow-all-mcp-server-instructions", "--log-level warning", "--log-dir " + filepath.Join(prepared.copilotHome, "logs")} {
		if !strings.Contains(args, want) {
			t.Errorf("Copilot command args %q do not contain %q", args, want)
		}
	}
	if strings.Contains(args, "--stdio") {
		t.Fatalf("Copilot ACP command should use its default stdio transport: %s", args)
	}
}

func TestNewCopilotCloudCommandProtectsCredentialAndAddsEgress(t *testing.T) {
	base := t.TempDir()
	profileDir := filepath.Join(base, ".opencode-sandboxes")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(profileDir, "copilot-cloud.sb")
	if err := os.WriteFile(profilePath, []byte("(version 1)\n(deny default)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{baseScratchDir: base, localProviderSandboxProfileFactory: func(_, _, endpoint string) (string, string, error) {
		if endpoint != copilotCloudAPIBaseURL {
			t.Fatalf("cloud sandbox endpoint = %q, want %q", endpoint, copilotCloudAPIBaseURL)
		}
		return "/usr/bin/sandbox-exec", profilePath, nil
	}}
	workDir := filepath.Join(base, "fixture")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := Target{Harness: string(HarnessCopilot), Model: "gpt-5.4"}
	prepared := &copilotPreparedRuntime{
		workDir: workDir, target: target, arm: ArmBaseline, cloud: true,
		executable: "/opt/homebrew/Caskroom/copilot-cli/1.0.88/copilot",
		runtimeDir: "/Users/worker/Library/Caches/copilot/pkg/darwin-arm64/1.0.88",
		modelURL:   copilotCloudAPIBaseURL, copilotHome: filepath.Join(workDir, ".scratch", "copilot", "copilot-home"),
		env: []string{"COPILOT_GITHUB_TOKEN=fixture-token"},
	}
	cmd, err := runner.newCopilotCommand(context.Background(), workDir, target, ArmBaseline, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(cmd.Args, " "), "--secret-env-vars=COPILOT_GITHUB_TOKEN") || !strings.Contains(strings.Join(cmd.Args, " "), "--auth-token-env=COPILOT_GITHUB_TOKEN") || !strings.Contains(strings.Join(cmd.Args, " "), "--no-auto-login") {
		t.Fatalf("cloud command does not mark credential secret: %#v", cmd.Args)
	}
	profile, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(profile), `remote tcp "*:443"`) || !strings.Contains(string(profile), `remote udp "*:53"`) {
		t.Fatalf("cloud command profile lacks bounded network rules: %s", profile)
	}
	if strings.Contains(string(profile), `file-write* (subpath "/Users")`) {
		t.Fatalf("cloud command widened filesystem writes: %s", profile)
	}
}
