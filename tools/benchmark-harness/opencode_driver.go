// opencode_driver.go configures OpenCode and converts its event stream into benchmark telemetry.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
	"slices"
	"strings"
	"time"
)

const (
	openCodeDefaultModel   = "amdbeast/qwen36-coder"
	openCodeQwenBaseURL    = "http://192.168.1.122:1234/v1"
	openCodeQwenBaseURLEnv = "SEMEDIT_OPENCODE_QWEN_BASE_URL"
	openRouterAPIKeyEnv    = "OPENROUTER_API_KEY" //nolint:gosec // environment variable name, never a credential value
	maxOpenCodeStderrBytes = 64 * 1024
)

func (r *Runner) runOpenCode(ctx context.Context, workDir string, target Target, prompt string, res *RunResult, resumeID string) (string, error) {
	configPath, env, err := r.openCodeEnvironment(ctx, workDir, res.Arm, target)
	if err != nil {
		return "", fmt.Errorf("prepare OpenCode environment: %w", err)
	}
	model := target.Model
	if model == "" {
		model = openCodeDefaultModel
	}
	if strings.HasPrefix(model, "openrouter/") && strings.TrimSpace(environmentLookup(env, openRouterAPIKeyEnv)) == "" {
		return "", fmt.Errorf("%s is not set; OpenCode benchmarks require OpenRouter authentication", openRouterAPIKeyEnv)
	}
	args := []string{"run", "--format", "json", "--auto", "--dir", workDir}
	if resumeID != "" {
		args = append(args, "--session", resumeID)
	}
	args = append(args, "--model", model)
	if target.Effort != "" {
		args = append(args, "--variant", target.Effort)
	}
	args = append(args, prompt)

	// #nosec G204 -- external driver invocation controlled by benchmark harness
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = workDir
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("open OpenCode stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	processStartedAt := time.Now()
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start OpenCode using %s: %w", configPath, err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	tracker := openCodeToolTracker{}
	var firstEventAt time.Time
	sessionID := resumeID
	for scanner.Scan() {
		var event map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		observedAt := time.Now()
		if firstEventAt.IsZero() {
			firstEventAt = observedAt
		}
		if candidate := openCodeEventSessionID(event); candidate != "" {
			sessionID = candidate
		}
		appendOpenCodeAgentResponse(res, event)
		tracker.observe(res, event, observedAt)
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		res.OpenCodeExitCode = &code
	}
	res.OpenCodeStderr = sanitizeOpenRouterDiagnostic(boundedDiagnostic(stderr.Bytes(), maxOpenCodeStderrBytes), env)
	res.Turns = 1
	res.InternalTurns = tracker.internalTurns
	res.InitialLoadTurns = tracker.initialLoadTurns
	res.MCPLoadTurns = tracker.mcpLoadTurns
	res.ToolCount = len(res.ToolCalls)
	if !firstEventAt.IsZero() {
		value := firstEventAt.Sub(processStartedAt)
		res.ProcessStartToFirstEvent = &value
	}
	if !firstEventAt.IsZero() && !tracker.firstToolCallAt.IsZero() {
		value := tracker.firstToolCallAt.Sub(firstEventAt)
		res.FirstEventToFirstToolCall = &value
	}
	if scanErr != nil {
		return sessionID, fmt.Errorf("scan OpenCode output: %w", scanErr)
	}
	if waitErr != nil {
		if diagnostic := strings.TrimSpace(res.OpenCodeStderr); diagnostic != "" {
			return sessionID, fmt.Errorf("run OpenCode: %w: %s", waitErr, diagnostic)
		}
		return sessionID, fmt.Errorf("run OpenCode: %w", waitErr)
	}
	if sessionID == "" {
		return "", fmt.Errorf("OpenCode did not emit a session ID")
	}
	return sessionID, nil
}

func (r *Runner) openCodeEnvironment(ctx context.Context, workDir string, arm ArmType, target Target) (string, []string, error) {
	goEnv, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return "", nil, err
	}
	configRoot := filepath.Join(workDir, ".scratch", "opencode")
	configPath := filepath.Join(configRoot, "opencode.json")
	if err := os.MkdirAll(configRoot, 0o750); err != nil {
		return "", nil, fmt.Errorf("create OpenCode fixture state: %w", err)
	}
	config, err := r.openCodeConfig(workDir, arm, target, goEnv)
	if err != nil {
		return "", nil, err
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", nil, fmt.Errorf("marshal OpenCode configuration: %w", err)
	}
	if err := pipeline.WriteAtomic(configPath, data); err != nil {
		return "", nil, fmt.Errorf("write OpenCode configuration: %w", err)
	}

	homeDir := filepath.Join(configRoot, "home")
	for _, dir := range []string{homeDir, filepath.Join(configRoot, "config"), filepath.Join(configRoot, "data"), filepath.Join(configRoot, "cache")} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return "", nil, fmt.Errorf("create OpenCode isolated directory: %w", err)
		}
	}
	env := append([]string(nil), goEnv...)
	env = replaceEnvironment(env, map[string]string{
		"HOME":                            homeDir,
		"XDG_CONFIG_HOME":                 filepath.Join(configRoot, "config"),
		"XDG_DATA_HOME":                   filepath.Join(configRoot, "data"),
		"XDG_CACHE_HOME":                  filepath.Join(configRoot, "cache"),
		"OPENCODE_CONFIG":                 configPath,
		"OPENCODE_DISABLE_PROJECT_CONFIG": "1",
	})
	return configPath, env, nil
}

func (r *Runner) openCodeConfig(workDir string, arm ArmType, target Target, env []string) (map[string]any, error) {
	model := target.Model
	if model == "" {
		model = openCodeDefaultModel
	}
	provider, providerModel, found := strings.Cut(model, "/")
	if !found || providerModel == "" {
		return nil, fmt.Errorf("OpenCode target %q must use provider/model form", model)
	}
	var providerConfig map[string]any
	switch provider {
	case "openrouter":
		providerConfig = map[string]any{
			"models": map[string]any{
				providerModel: map[string]any{"name": providerModel},
			},
		}
	case "amdbeast":
		baseURL := openCodeQwenBaseURL
		if override := strings.TrimSpace(os.Getenv(openCodeQwenBaseURLEnv)); override != "" {
			baseURL = override
		}
		providerConfig = map[string]any{
			"name": "AMD Beast",
			"npm":  "@ai-sdk/openai-compatible",
			"options": map[string]any{
				"baseURL": baseURL,
			},
			"models": map[string]any{
				providerModel: map[string]any{"name": "QWen 3.6 Coder"},
			},
		}
	default:
		return nil, fmt.Errorf("unsupported OpenCode provider %q", provider)
	}
	config := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]any{
			provider: providerConfig,
		},
	}
	if provider == "openrouter" {
		config["model"] = model
		config["small_model"] = model
	}
	if arm == ArmBaseline {
		return config, nil
	}
	if arm != ArmSemedit {
		return nil, fmt.Errorf("unsupported OpenCode benchmark arm %q", arm)
	}
	args, err := r.openCodeMCPArguments()
	if err != nil {
		return nil, err
	}
	repositoryRoot := filepath.Dir(filepath.Dir(r.baseScratchDir))
	config["mcp"] = map[string]any{
		"servers": map[string]any{
			"semedit": map[string]any{
				"type":        "local",
				"command":     append([]string{filepath.Join(repositoryRoot, "bin", "semedit-next")}, args...),
				"cwd":         workDir,
				"environment": environmentMap(env),
				"codemode":    false,
			},
		},
	}
	return config, nil
}

func (r *Runner) openCodeMCPArguments() ([]string, error) {
	switch r.mcpServerInstructionsMode {
	case MCPServerInstructionsNone:
		return []string{"mcp", "--profile", "full"}, nil
	case MCPServerInstructionsDescriptive:
		return []string{"mcp", "--profile", "full", "--instructions", mcp.DescriptiveInstructions}, nil
	case MCPServerInstructionsPrescriptive:
		return []string{"mcp", "--profile", "full", "--instructions", mcp.PrescriptiveInstructions}, nil
	default:
		return nil, fmt.Errorf("unsupported MCP server instruction mode %q", r.mcpServerInstructionsMode)
	}
}

func environmentMap(env []string) map[string]string {
	values := make(map[string]string, 8)
	allowed := map[string]struct{}{
		"GOENV": {}, "GOCACHE": {}, "GOMODCACHE": {}, "GOTMPDIR": {},
		"GOBIN": {}, "GOPATH": {}, "GOFLAGS": {}, "GOWORK": {},
	}
	for _, entry := range env {
		key, value, found := strings.Cut(entry, "=")
		if found {
			if _, ok := allowed[key]; !ok {
				continue
			}
			values[key] = value
		}
	}
	return values
}

func environmentLookup(env []string, key string) string {
	for _, entry := range env {
		entryKey, value, found := strings.Cut(entry, "=")
		if found && entryKey == key {
			return value
		}
	}
	return ""
}

func sanitizeOpenRouterDiagnostic(diagnostic string, env []string) string {
	for _, entry := range env {
		key, value, found := strings.Cut(entry, "=")
		if found && key == openRouterAPIKeyEnv && value != "" {
			diagnostic = strings.ReplaceAll(diagnostic, value, "[REDACTED]")
		}
	}
	return diagnostic
}

func replaceEnvironment(env []string, replacements map[string]string) []string {
	result := make([]string, 0, len(env)+len(replacements))
	for _, entry := range env {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, replaced := replacements[key]; !replaced {
			result = append(result, entry)
		}
	}
	keys := slices.Sorted(maps.Keys(replacements))
	for _, key := range keys {
		result = append(result, key+"="+replacements[key])
	}
	return result
}

type openCodeToolTracker struct {
	calls            map[string]int
	internalTurns    int
	initialLoadTurns int
	mcpLoadTurns     int
	mutatingSeen     bool
	firstToolCallAt  time.Time
}

func (t *openCodeToolTracker) observe(res *RunResult, event map[string]any, observedAt time.Time) {
	for _, tool := range openCodeToolEvents(event) {
		call := openCodeToolOutcome(tool)
		if call.Name == "" {
			continue
		}
		if callIndex, found := t.calls[tool.id]; found {
			if len(call.Arguments) == 0 {
				call.Arguments = res.ToolCalls[callIndex].Arguments
			}
			res.ToolCalls[callIndex] = call
			applyMCPStartupMetrics(res, call.MCPStartupMetrics)
			refreshMCPVerified(res)
			continue
		}
		if t.firstToolCallAt.IsZero() {
			t.firstToolCallAt = observedAt
		}
		if t.calls == nil {
			t.calls = make(map[string]int)
		}
		callIndex := appendToolCall(res, call)
		if tool.id != "" {
			t.calls[tool.id] = callIndex
		}
		applyMCPStartupMetrics(res, call.MCPStartupMetrics)
		t.internalTurns++
		if isSemanticTool(call.Name, call.Server) || isMutatingTool(call.Name) {
			t.mutatingSeen = true
			continue
		}
		if !t.mutatingSeen {
			t.initialLoadTurns++
			if call.Server != "" || isMCPDiscoveryTool(call.Name) {
				t.mcpLoadTurns++
			}
		}
	}
}

type openCodeToolEvent struct {
	id        string
	name      string
	server    string
	arguments any
	status    string
	result    any
	failure   any
}

func openCodeToolEvents(event map[string]any) []openCodeToolEvent {
	var events []openCodeToolEvent
	var visit func(map[string]any)
	visit = func(value map[string]any) {
		if tool, found := openCodeToolEventFromMap(value); found {
			events = append(events, tool)
			return
		}
		for _, child := range value {
			switch typed := child.(type) {
			case map[string]any:
				visit(typed)
			case []any:
				for _, entry := range typed {
					if nested, ok := entry.(map[string]any); ok {
						visit(nested)
					}
				}
			}
		}
	}
	visit(event)
	return events
}

func openCodeToolEventFromMap(value map[string]any) (openCodeToolEvent, bool) {
	kind := strings.ToLower(stringValue(value["type"]))
	state, _ := value["state"].(map[string]any)
	name := firstString(value["tool"], value["tool_name"], value["name"])
	if name == "" || (kind != "tool" && !strings.Contains(kind, "tool") && state == nil) {
		return openCodeToolEvent{}, false
	}
	if state == nil {
		state = value
	}
	arguments := state["input"]
	if arguments == nil {
		arguments = value["input"]
	}
	status := firstString(state["status"], value["status"])
	return openCodeToolEvent{
		id:        firstString(value["id"], state["id"]),
		name:      name,
		server:    firstString(value["server"], state["server"]),
		arguments: arguments,
		status:    status,
		result:    firstValue(state["output"], state["result"], value["output"], value["result"]),
		failure:   firstValue(state["error"], value["error"]),
	}, true
}

func openCodeToolOutcome(event openCodeToolEvent) ToolCall {
	arguments, _ := json.Marshal(event.arguments)
	result, _ := json.Marshal(event.result)
	call := ToolCall{
		Name:              event.name,
		Server:            event.server,
		Arguments:         arguments,
		TransportStatus:   ToolCallStatusUnknown,
		FunctionalStatus:  ToolCallStatusUnknown,
		MCPMetrics:        codexMCPMetrics(result),
		MCPStartupMetrics: codexMCPStartupMetrics(result),
	}
	status := strings.ToLower(strings.TrimSpace(event.status))
	failure := compactToolFailure(toolFailureText(event.failure))
	switch status {
	case "completed", "success", "succeeded":
		call.TransportStatus = ToolCallStatusSucceeded
		call.FunctionalStatus = ToolCallStatusSucceeded
		if failure != "" {
			call.FunctionalStatus = ToolCallStatusFailed
			call.Failure = failure
		}
	case "failed", "error", "cancelled", "canceled":
		call.FunctionalStatus = ToolCallStatusFailed
		call.Failure = failure
		if openCodeTransportFailure(failure) {
			call.TransportStatus = ToolCallStatusFailed
		} else {
			call.TransportStatus = ToolCallStatusSucceeded
		}
	}
	return call
}

func openCodeTransportFailure(failure string) bool {
	message := strings.ToLower(failure)
	return strings.Contains(message, "mcp") && (strings.Contains(message, "connect") || strings.Contains(message, "unavailable") || strings.Contains(message, "transport") || strings.Contains(message, "initialize"))
}

func openCodeEventSessionID(event map[string]any) string {
	if value := firstString(event["sessionID"], event["session_id"]); value != "" {
		return value
	}
	if !strings.Contains(strings.ToLower(stringValue(event["type"])), "session") {
		return ""
	}
	for _, candidate := range []any{event["properties"], event["session"], event["info"]} {
		if value, ok := candidate.(map[string]any); ok {
			if id := firstString(value["id"], value["sessionID"], value["session_id"]); id != "" {
				return id
			}
			if info, ok := value["info"].(map[string]any); ok {
				if id := firstString(info["id"], info["sessionID"], info["session_id"]); id != "" {
					return id
				}
			}
		}
	}
	return ""
}

func appendOpenCodeAgentResponse(res *RunResult, event map[string]any) {
	if res == nil {
		return
	}
	var parts []string
	var visit func(map[string]any)
	visit = func(value map[string]any) {
		if strings.EqualFold(stringValue(value["type"]), "text") {
			if text := strings.TrimSpace(firstString(value["text"], value["content"])); text != "" {
				parts = append(parts, text)
			}
		}
		for _, child := range value {
			switch typed := child.(type) {
			case map[string]any:
				visit(typed)
			case []any:
				for _, entry := range typed {
					if nested, ok := entry.(map[string]any); ok {
						visit(nested)
					}
				}
			}
		}
	}
	visit(event)
	for _, text := range parts {
		if res.agentResponse != "" {
			res.agentResponse += "\n"
		}
		res.agentResponse += text
	}
}

func firstString(values ...any) string {
	for _, value := range values {
		if text := stringValue(value); text != "" {
			return text
		}
	}
	return ""
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func firstValue(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
