// Package main owns isolated cloud and local Copilot CLI configuration and containment.
// WHY: Copilot reads user state and MCP configuration outside the fixture by default, so benchmark setup pins both to one disposable runtime.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/spockz/semantic-editor/internal/mcp"
)

const (
	copilotBaseURLEnv             = "SEMEDIT_COPILOT_BASE_URL"
	copilotOfflineEnv             = "COPILOT_OFFLINE"
	copilotAutoUpdateEnv          = "COPILOT_AUTO_UPDATE"
	copilotHomeEnv                = "COPILOT_HOME"
	copilotModelEnv               = "COPILOT_MODEL"
	copilotProviderTypeEnv        = "COPILOT_PROVIDER_TYPE"
	copilotProviderBaseURLEnv     = "COPILOT_PROVIDER_BASE_URL"
	copilotProviderModelIDEnv     = "COPILOT_PROVIDER_MODEL_ID"
	copilotProviderWireModelEnv   = "COPILOT_PROVIDER_WIRE_MODEL"
	copilotProviderWireAPIEnv     = "COPILOT_PROVIDER_WIRE_API"
	copilotProviderAPIKeyEnv      = "COPILOT_PROVIDER_API_KEY"
	copilotProviderTransportEnv   = "COPILOT_PROVIDER_TRANSPORT"
	copilotMaxPromptTokensEnv     = "COPILOT_PROVIDER_MAX_PROMPT_TOKENS"
	copilotMaxOutputTokensEnv     = "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS"
	copilotCLIDistDirEnv          = "COPILOT_CLI_DIST_DIR"
	copilotDefaultInstructionsRel = ".github/copilot-instructions.md"
	copilotCloudAPIBaseURL        = "https://api.githubcopilot.com"
)

type copilotPreparedRuntime struct {
	workDir         string
	target          Target
	arm             ArmType
	executable      string
	runtimeDir      string
	modelURL        string
	cloud           bool
	homeDir         string
	copilotHome     string
	mcpConfigPath   string
	mcpConfigData   []byte
	instructionPath string
	instructionData []byte
	env             []string
}

type copilotMCPConfiguration struct {
	MCPServers map[string]copilotMCPServer `json:"mcpServers"`
}

type copilotMCPServer struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Tools   []string          `json:"tools"`
}

func copilotModelBaseURL() string {
	if value := strings.TrimSpace(os.Getenv(copilotBaseURLEnv)); value != "" {
		return strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(os.Getenv(openCodeQwenBaseURLEnv)); value != "" {
		return strings.TrimRight(value, "/")
	}
	return strings.TrimRight(openCodeQwenBaseURL, "/")
}

func (p *copilotProvider) prepareFixture(ctx context.Context, workDir string, target Target, arm ArmType, external bool) error {
	if !external {
		if err := initializeOpenCodeFixtureGit(ctx, workDir); err != nil {
			return err
		}
	}
	runtime, err := prepareCopilotRuntime(ctx, p.runner, workDir, target, arm)
	if err != nil {
		return err
	}
	p.runtime = &runtime
	return nil
}

func prepareCopilotRuntime(ctx context.Context, runner *Runner, workDir string, target Target, arm ArmType) (copilotPreparedRuntime, error) {
	executable, runtimeDir, err := resolveCopilotInstallation()
	if err != nil {
		return copilotPreparedRuntime{}, err
	}
	goEnv, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return copilotPreparedRuntime{}, err
	}

	_, local := copilotLocalModel(target)
	var limits *localModelLimits
	var cloudToken string
	modelURL := copilotCloudAPIBaseURL
	if local {
		limits = runner.localModelLimits
		if limits == nil || limits.Context <= 0 || limits.Output <= 0 || limits.Output >= limits.Context {
			return copilotPreparedRuntime{}, fmt.Errorf("copilot local model requires valid context and output token limits")
		}
		modelURL = strings.TrimSpace(runner.localModelRelayURL)
		if modelURL == "" {
			return copilotPreparedRuntime{}, fmt.Errorf("copilot local model requires a run-owned relay URL")
		}
	} else {
		var credentialErr error
		cloudToken, credentialErr = copilotCloudCredential(os.Environ())
		if credentialErr != nil {
			return copilotPreparedRuntime{}, credentialErr
		}
	}
	const runtimeRoot = ".scratch/copilot"
	paths := map[string]string{
		"home":        filepath.Join(runtimeRoot, "home"),
		"copilotHome": filepath.Join(runtimeRoot, "copilot-home"),
		"config":      filepath.Join(runtimeRoot, "config"),
		"data":        filepath.Join(runtimeRoot, "data"),
		"cache":       filepath.Join(runtimeRoot, "cache"),
		"tmp":         filepath.Join(runtimeRoot, "tmp"),
	}
	directories := make([]string, 0, len(paths))
	for _, relative := range paths {
		directories = append(directories, relative)
	}
	if layoutErr := prepareRuntimeLayout(workDir, directories, nil); layoutErr != nil {
		return copilotPreparedRuntime{}, fmt.Errorf("create isolated Copilot runtime: %w", layoutErr)
	}
	root, err := canonicalFixtureRoot(workDir)
	if err != nil {
		return copilotPreparedRuntime{}, err
	}
	absolute := func(relative string) string { return filepath.Join(root, relative) }
	homeDir, copilotHome := absolute(paths["home"]), absolute(paths["copilotHome"])
	if _, endpointErr := openCodeSandboxRelayPort(modelURL); endpointErr != nil {
		return copilotPreparedRuntime{}, fmt.Errorf("resolve Copilot model endpoint: %w", endpointErr)
	}
	env := sanitizedCopilotEnvironment(goEnv)
	values := copilotProviderEnvironment(target, local, modelURL, limits, cloudToken)
	maps.Copy(values, (map[string]string{"HOME": homeDir, "TMPDIR": absolute(paths["tmp"]), "XDG_CONFIG_HOME": absolute(paths["config"]), "XDG_DATA_HOME": absolute(paths["data"]), "XDG_CACHE_HOME": absolute(paths["cache"]), copilotHomeEnv: copilotHome, copilotAutoUpdateEnv: "false", copilotCLIDistDirEnv: runtimeDir}))
	env = replaceEnvironment(env, values)
	runtime := copilotPreparedRuntime{
		workDir: workDir, target: target, arm: arm, executable: executable, runtimeDir: runtimeDir,
		modelURL: modelURL, cloud: !local, homeDir: homeDir, copilotHome: copilotHome,
		mcpConfigPath: filepath.Join(copilotHome, "mcp-config.json"), env: append([]string(nil), env...),
	}
	if isSemeditArm(arm) {
		if configErr := configureCopilotSemeditRuntime(runner, workDir, root, &runtime); configErr != nil {
			return copilotPreparedRuntime{}, configErr
		}
	}
	return runtime, nil
}

func copilotProviderEnvironment(target Target, local bool, modelURL string, limits *localModelLimits, cloudToken string) map[string]string {
	values := map[string]string{copilotModelEnv: copilotWireModel(target)}
	if !local {
		values["COPILOT_GITHUB_TOKEN"] = cloudToken
		return values
	}
	values[copilotOfflineEnv] = "true"
	values[copilotProviderTypeEnv] = "openai"
	values[copilotProviderBaseURLEnv] = strings.TrimRight(modelURL, "/")
	values[copilotProviderModelIDEnv] = copilotWireModel(target)
	values[copilotProviderWireModelEnv] = copilotWireModel(target)
	values[copilotProviderWireAPIEnv] = "completions"
	values[copilotProviderTransportEnv] = "http"
	if limits != nil {
		values[copilotMaxPromptTokensEnv] = strconv.Itoa(limits.Context - limits.Output)
		values[copilotMaxOutputTokensEnv] = strconv.Itoa(limits.Output)
	}
	values[copilotProviderAPIKeyEnv] = ""
	values["COPILOT_PROVIDER_BEARER_TOKEN"] = ""
	return values
}

func copilotCloudCredential(environment []string) (string, error) {
	for _, name := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		for _, entry := range environment {
			key, value, found := strings.Cut(entry, "=")
			if found && key == name && strings.TrimSpace(value) != "" {
				return value, nil
			}
		}
	}
	return "", fmt.Errorf("copilot cloud target requires COPILOT_GITHUB_TOKEN, GH_TOKEN, or GITHUB_TOKEN")
}

func configureCopilotSemeditRuntime(runner *Runner, workDir, root string, prepared *copilotPreparedRuntime) error {
	instruction, err := copilotInstructionConfiguration(AgentExecution{Arm: prepared.arm, Policy: runner.semeditArmRestriction})
	if err != nil {
		return err
	}
	prepared.instructionPath = filepath.Join(root, copilotDefaultInstructionsRel)
	if existing, readErr := os.ReadFile(prepared.instructionPath); readErr == nil && len(existing) != 0 {
		prepared.instructionData = append(append(append([]byte(nil), existing...), '\n', '\n'), []byte(instruction+"\n")...)
	} else if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read existing copilot instructions: %w", readErr)
	} else {
		prepared.instructionData = []byte(instruction + "\n")
	}
	if writeErr := prepareRuntimeLayout(workDir, nil, map[string][]byte{copilotDefaultInstructionsRel: prepared.instructionData}); writeErr != nil {
		return fmt.Errorf("write copilot semedit arm instructions: %w", writeErr)
	}
	config, err := copilotMCPConfig(runner, workDir, root)
	if err != nil {
		return err
	}
	prepared.mcpConfigData, err = json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal copilot MCP configuration: %w", err)
	}
	prepared.mcpConfigData = append(prepared.mcpConfigData, '\n')
	const configPath = ".scratch/copilot/copilot-home/mcp-config.json"
	if writeErr := prepareRuntimeLayout(workDir, nil, map[string][]byte{configPath: prepared.mcpConfigData}); writeErr != nil {
		return fmt.Errorf("write copilot MCP configuration: %w", writeErr)
	}
	return nil
}

func resolveCopilotInstallation() (string, string, error) {
	executable, err := exec.LookPath("copilot")
	if err != nil {
		return "", "", fmt.Errorf("locate Copilot CLI: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", "", fmt.Errorf("resolve Copilot CLI executable: %w", err)
	}
	version := filepath.Base(filepath.Dir(executable))
	if version == "." || version == string(filepath.Separator) || version == "" {
		return "", "", fmt.Errorf("derive Copilot runtime bundle version from executable path %q", executable)
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", "", fmt.Errorf("locate Copilot runtime cache root: %w", err)
	}
	runtimeDir := filepath.Join(cacheRoot, "copilot", "pkg", runtime.GOOS+"-"+runtime.GOARCH, version)
	runtimeDir, err = filepath.EvalSymlinks(runtimeDir)
	if err != nil {
		return "", "", fmt.Errorf("resolve Copilot runtime bundle %q: %w", runtimeDir, err)
	}
	info, err := os.Stat(runtimeDir)
	if err != nil {
		return "", "", fmt.Errorf("inspect Copilot runtime bundle %q: %w", runtimeDir, err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("copilot runtime bundle %q is not a directory", runtimeDir)
	}
	return executable, runtimeDir, nil
}

func sanitizedCopilotEnvironment(source []string) []string {
	openCodeClean := sanitizedOpenCodeEnvironment(source)
	clean := make([]string, 0, len(openCodeClean))
	for _, entry := range openCodeClean {
		key, _, found := strings.Cut(entry, "=")
		if !found || strings.HasPrefix(key, "COPILOT_") || strings.HasPrefix(key, "GITHUB_") || strings.HasPrefix(key, "GH_") || strings.HasPrefix(key, "OPENAI_") || strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "AZURE_") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func copilotMCPConfig(runner *Runner, workDir, root string) (copilotMCPConfiguration, error) {
	repositoryRoot := filepath.Dir(filepath.Dir(runner.baseScratchDir))
	semeditPath := filepath.Join(repositoryRoot, "bin", "semedit-next")
	if resolvedPath, err := filepath.EvalSymlinks(semeditPath); err == nil {
		semeditPath = resolvedPath
	}
	args := []string{"mcp", "--profile", "full", "--expected-root", root, "--go-base-dir", filepath.Join(root, ".scratch", "go"), "--metrics-file", benchmarkMCPMetricsPath(workDir)}
	if runner.semeditBatching {
		args = append(args, "--enable-batching")
	}
	switch runner.mcpServerInstructionsMode {
	case MCPServerInstructionsNone:
		args = append(args, "--instructions", "")
	case MCPServerInstructionsDescriptive:
		args = append(args, "--instructions", mcp.DescriptiveInstructions)
	case MCPServerInstructionsPrescriptive:
		args = append(args, "--instructions", mcp.PrescriptiveInstructions)
	default:
		return copilotMCPConfiguration{}, fmt.Errorf("unsupported MCP server instruction mode %q", runner.mcpServerInstructionsMode)
	}
	return copilotMCPConfiguration{MCPServers: map[string]copilotMCPServer{
		"semedit": {Type: "local", Command: semeditPath, Args: args, Env: map[string]string{}, Tools: []string{"*"}},
	}}, nil
}

func (r *Runner) newCopilotCommand(ctx context.Context, workDir string, target Target, arm ArmType, prepared *copilotPreparedRuntime) (*exec.Cmd, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("start copilot process with canceled context: %w", err)
	}
	if prepared == nil || prepared.workDir != workDir || prepared.target != target || prepared.arm != arm {
		return nil, fmt.Errorf("prepared Copilot runtime does not match session workspace, target, or arm")
	}
	if snapshotErr := verifyCopilotRuntimeSnapshot(prepared); snapshotErr != nil {
		return nil, snapshotErr
	}
	sandboxPath, profilePath, err := r.localProviderSandboxProfile(workDir, prepared.executable, prepared.modelURL)
	if err != nil {
		return nil, fmt.Errorf("prepare Copilot filesystem containment: %w", err)
	}
	// #nosec G304 -- profilePath is created under the runner's isolated scratch root.
	profile, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("read Copilot sandbox profile: %w", err)
	}
	profile, err = appendCopilotRuntimeRule(profile, prepared.runtimeDir)
	if err != nil {
		return nil, err
	}
	if prepared.cloud {
		profile, err = appendCopilotCloudNetworkRules(profile)
		if err != nil {
			return nil, err
		}
	}
	if writeErr := prepareRuntimeLayout(r.baseScratchDir, nil, map[string][]byte{filepath.Join(".opencode-sandboxes", filepath.Base(profilePath)): profile}); writeErr != nil {
		return nil, fmt.Errorf("allow Copilot runtime bundle in sandbox: %w", writeErr)
	}
	args := ([]string{"-f", profilePath, prepared.executable, "--model", copilotWireModel(target), "--acp", "--no-auto-update", "--disable-builtin-mcps", "--no-ask-user", "--no-remote", "--no-remote-export", "--allow-all-mcp-server-instructions", "--log-level", "warning", "--log-dir", filepath.Join(prepared.copilotHome, "logs")})
	if prepared.cloud {
		args = append(args, "--auth-token-env=COPILOT_GITHUB_TOKEN", "--no-auto-login", "--secret-env-vars=COPILOT_GITHUB_TOKEN")
	}
	// #nosec G204 -- all arguments are fixed and the only external process path is resolved from PATH.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), sandboxPath, args...)
	cmd.Cancel = nil
	cmd.Dir = workDir
	cmd.Env = append([]string(nil), prepared.env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

func appendCopilotCloudNetworkRules(profile []byte) ([]byte, error) {
	if len(profile) == 0 {
		return nil, fmt.Errorf("copilot cloud sandbox requires a base Seatbelt profile")
	}
	// macOS DNS needs the /var symlink metadata and its resolver's UNIX socket.
	const rules = "(allow network-outbound (remote tcp \"*:443\"))\n" +
		"(allow network-outbound (remote udp \"*:53\"))\n" +
		"(allow mach-lookup (global-name \"com.apple.system.opendirectoryd.libinfo\"))\n" +
		"(allow mach-lookup (global-name \"com.apple.trustd\"))\n" +
		"(allow mach-lookup (global-name \"com.apple.trustd.agent\"))\n" +
		("(allow mach-lookup (global-name \"com.apple.SystemConfiguration.configd\"))\n(allow file-read-metadata (literal \"/var\"))\n(allow network-outbound (literal \"/private/var/run/mDNSResponder\"))\n")
	return append(profile, []byte(rules)...), nil
}

func appendCopilotRuntimeRule(profile []byte, runtimeDir string) ([]byte, error) {
	if filepath.Clean(runtimeDir) == string(filepath.Separator) || !filepath.IsAbs(runtimeDir) {
		return nil, fmt.Errorf("copilot runtime path %q must be an absolute non-root path", runtimeDir)
	}
	runtimeDir = filepath.Clean(runtimeDir)
	var rules strings.Builder
	fmt.Fprintf(&rules, "(allow file-read* (subpath %s))\n", strconv.Quote(runtimeDir))
	for parent := filepath.Dir(runtimeDir); parent != string(filepath.Separator); parent = filepath.Dir(parent) {
		fmt.Fprintf(&rules, "(allow file-read-metadata (literal %s))\n", strconv.Quote(parent))
	}
	return append(profile, []byte(rules.String())...), nil
}

func verifyCopilotRuntimeSnapshot(prepared *copilotPreparedRuntime) error {
	if len(prepared.mcpConfigData) != 0 {
		// #nosec G304 -- path is generated inside the isolated Copilot home.
		data, err := os.ReadFile(prepared.mcpConfigPath)
		if err != nil {
			return fmt.Errorf("read prepared Copilot MCP configuration: %w", err)
		}
		if string(data) != string(prepared.mcpConfigData) {
			return fmt.Errorf("copilot MCP configuration changed before process startup")
		}
	}
	if len(prepared.instructionData) != 0 {
		// #nosec G304 -- path is generated inside the fixture workspace.
		data, err := os.ReadFile(prepared.instructionPath)
		if err != nil {
			return fmt.Errorf("read prepared Copilot instructions: %w", err)
		}
		if string(data) != string(prepared.instructionData) {
			return fmt.Errorf("copilot instructions changed before process startup")
		}
	}
	return nil
}
