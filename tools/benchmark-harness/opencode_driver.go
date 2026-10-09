// opencode_driver.go configures OpenCode and converts its event stream into benchmark telemetry.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	openCodeDefaultModel   = "qwen36-iq4-xs"
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
	model := openCodeProviderModel(target.Model)
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
	openCodePath, err := exec.LookPath("opencode")
	if err != nil {
		return "", fmt.Errorf("locate OpenCode executable: %w", err)
	}
	sandboxPath, profilePath, err := r.openCodeSandboxProfile(workDir, openCodePath, openCodeModelBaseURL(target))
	if err != nil {
		return "", fmt.Errorf("prepare OpenCode filesystem containment: %w", err)
	}
	// #nosec G204 -- benchmark harness starts a fixed sandbox launcher and configured OpenCode executable.
	cmd := exec.CommandContext(ctx, sandboxPath, append([]string{"-f", profilePath, openCodePath}, args...)...)
	cmd.Dir = workDir
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("open OpenCode stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start sandboxed OpenCode using %s: %w", configPath, err)
	}
	processStartedAt := time.Now()
	res.processStartedAt = processStartedAt
	stream := readOpenCodeEventStream(stdout, res, resumeID)
	waitErr := cmd.Wait()
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		res.OpenCodeExitCode = &code
	}
	res.OpenCodeStderr = sanitizeOpenRouterDiagnostic(boundedDiagnostic(stderr.Bytes(), maxOpenCodeStderrBytes), env)
	applyOpenCodeStreamMetrics(res, stream, processStartedAt)
	providerErr := openCodeStreamFailure(stream, waitErr, res.OpenCodeStderr)
	if providerErr != nil {
		providerErr = &openCodeDiagnosticError{cause: providerErr, message: sanitizeOpenRouterDiagnostic(providerErr.Error(), env)}
	}
	setOpenCodeObservationState(res, providerErr, stream)
	if providerErr != nil {
		return stream.sessionID, providerErr
	}
	if stream.sessionID == "" {
		return "", fmt.Errorf("OpenCode did not emit a session ID")
	}
	if stream.validEvents == 0 {
		return stream.sessionID, fmt.Errorf("OpenCode emitted no valid task events")
	}
	return stream.sessionID, nil
}

type openCodeEventStream struct {
	tracker      openCodeToolTracker
	decodeErrors int
	validEvents  int
	firstEventAt time.Time
	sessionID    string
	identityErr  error
	scanErr      error
	usage        openCodeUsageTracker
}

func readOpenCodeEventStream(stdout io.Reader, result *RunResult, resumeID string) openCodeEventStream {
	stream := openCodeEventStream{sessionID: resumeID}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		consumeOpenCodeEvent(&stream, result, append([]byte(nil), scanner.Bytes()...))
	}
	stream.scanErr = scanner.Err()
	return stream
}

func consumeOpenCodeEvent(stream *openCodeEventStream, result *RunResult, raw []byte) {
	var event map[string]any
	if err := json.Unmarshal(raw, &event); err != nil || event == nil || len(event) == 0 || openCodeEventType(event) == "" {
		stream.decodeErrors++
		encoded, _ := json.Marshal(string(raw))
		result.rawEvents = append(result.rawEvents, json.RawMessage(encoded))
		return
	}
	stream.validEvents++
	result.rawEvents = append(result.rawEvents, json.RawMessage(raw))
	observedAt := time.Now()
	if stream.firstEventAt.IsZero() {
		stream.firstEventAt = observedAt
	}
	if candidate := openCodeEventSessionID(event); candidate != "" {
		if stream.sessionID == "" {
			stream.sessionID = candidate
		} else if stream.sessionID != candidate {
			stream.identityErr = errors.Join(stream.identityErr, fmt.Errorf("OpenCode event session %q does not match session %q", candidate, stream.sessionID))
		}
	}
	appendOpenCodeAgentResponse(result, event)
	stream.tracker.observe(result, event, observedAt)
	stream.usage.observe(event)
}

func applyOpenCodeStreamMetrics(result *RunResult, stream openCodeEventStream, processStartedAt time.Time) {
	stream.usage.apply(result)
	result.Turns = 0
	if stream.validEvents > 0 {
		result.Turns = 1
	}
	result.InternalTurns = stream.tracker.internalTurns
	result.InitialLoadTurns = stream.tracker.initialLoadTurns
	result.MCPLoadTurns = stream.tracker.mcpLoadTurns
	result.ToolCount = len(result.ToolCalls)
	if !processStartedAt.IsZero() && !stream.firstEventAt.IsZero() {
		value := stream.firstEventAt.Sub(processStartedAt)
		result.ProcessStartToFirstEvent = &value
	}
	if !stream.firstEventAt.IsZero() && !stream.tracker.firstToolCallAt.IsZero() {
		value := stream.tracker.firstToolCallAt.Sub(stream.firstEventAt)
		result.FirstEventToFirstToolCall = &value
	}
}

func openCodeStreamFailure(stream openCodeEventStream, waitErr error, stderr string) error {
	var providerErr error
	if stream.scanErr != nil {
		providerErr = fmt.Errorf("scan OpenCode output: %w", stream.scanErr)
	}
	if stream.decodeErrors > 0 {
		providerErr = errors.Join(providerErr, fmt.Errorf("decode %d malformed OpenCode event records", stream.decodeErrors))
	}
	if waitErr != nil {
		if diagnostic := strings.TrimSpace(stderr); diagnostic != "" {
			providerErr = errors.Join(providerErr, fmt.Errorf("run OpenCode: %w: %s", waitErr, diagnostic))
		} else {
			providerErr = errors.Join(providerErr, fmt.Errorf("run OpenCode: %w", waitErr))
		}
	}
	providerErr = errors.Join(providerErr, stream.identityErr, stream.usage.err)
	if stream.validEvents > 0 && (!stream.usage.finished || len(stream.usage.starts) > len(stream.usage.steps)) {
		providerErr = errors.Join(providerErr, fmt.Errorf("OpenCode stream lacks complete terminal token usage"))
	}
	return providerErr
}

func setOpenCodeObservationState(result *RunResult, providerErr error, stream openCodeEventStream) {
	switch {
	case providerErr != nil:
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = providerErr.Error()
	case stream.sessionID == "" || stream.validEvents == 0:
		result.toolObservationState = ToolObservationPartial
		result.toolObservationReason = "OpenCode stream lacks a session ID or valid events"
	default:
		result.toolObservationState = ToolObservationUnknown
		result.toolObservationReason = "OpenCode adapter has no validated terminal event marker"
	}
}

func openCodeEventType(event map[string]any) string {
	eventType, _ := event["type"].(string)
	return strings.TrimSpace(eventType)
}

func (r *Runner) openCodeEnvironment(ctx context.Context, workDir string, arm ArmType, target Target) (string, []string, error) {
	goEnv, envErr := fixtureGoEnvironment(ctx, workDir)
	if envErr != nil {
		return "", nil, envErr
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
	configDir := filepath.Join(configRoot, "config")
	dataDir := filepath.Join(configRoot, "data")
	cacheDir := filepath.Join(configRoot, "cache")
	tempDir := filepath.Join(configRoot, "tmp")
	for _, dir := range []string{homeDir, configDir, dataDir, cacheDir, tempDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return "", nil, fmt.Errorf("create OpenCode isolated directory: %w", err)
		}
	}
	env := sanitizedOpenCodeEnvironment(goEnv)
	env = replaceEnvironment(env, map[string]string{
		"HOME":                            homeDir,
		"XDG_CONFIG_HOME":                 configDir,
		"XDG_DATA_HOME":                   dataDir,
		"XDG_CACHE_HOME":                  cacheDir,
		"TMPDIR":                          tempDir,
		"OPENCODE_CONFIG":                 configPath,
		"OPENCODE_DISABLE_PROJECT_CONFIG": "1",
	})
	return configPath, env, nil
}

func (r *Runner) openCodeConfig(workDir string, arm ArmType, target Target, env []string) (map[string]any, error) {
	model := openCodeProviderModel(target.Model)
	provider, providerModel, found := strings.Cut(model, "/")
	if !found || providerModel == "" {
		return nil, fmt.Errorf("OpenCode target %q must use provider/model form", model)
	}
	var providerConfig map[string]any
	switch provider {
	case "openrouter":
		providerConfig = map[string]any{
			"models": map[string]any{providerModel: map[string]any{"name": providerModel}},
		}
	case "amdbeast":
		baseURL := openCodeQwenBaseURL
		if override := strings.TrimSpace(os.Getenv(openCodeQwenBaseURLEnv)); override != "" {
			baseURL = override
		}
		providerConfig = map[string]any{
			"name":    "AMD Beast",
			"npm":     "@ai-sdk/openai-compatible",
			"options": map[string]any{"baseURL": baseURL},
			"models":  map[string]any{providerModel: map[string]any{"name": providerModel, "id": providerModel}},
		}
	default:
		return nil, fmt.Errorf("unsupported OpenCode provider %q", provider)
	}
	config := map[string]any{
		"$schema":    "https://opencode.ai/config.json",
		"tools":      map[string]bool{"task": false},
		"permission": map[string]any{"external_directory": "deny"},
		"provider":   map[string]any{provider: providerConfig},
	}
	if provider == "openrouter" {
		config["model"] = model
		config["small_model"] = model
	}
	if arm == ArmBaseline {
		return config, nil
	}
	if !isSemeditArm(arm) {
		return nil, fmt.Errorf("unsupported OpenCode benchmark arm %q", arm)
	}
	args, err := r.openCodeMCPArguments(workDir)
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

func (r *Runner) openCodeMCPArguments(workDir string) ([]string, error) {
	root, err := canonicalFixtureRoot(workDir)
	if err != nil {
		return nil, err
	}
	args := []string{"mcp", "--profile", "full", "--expected-root", root, "--metrics-file", benchmarkMCPMetricsPath(workDir)}
	switch r.mcpServerInstructionsMode {
	case MCPServerInstructionsNone:
	case MCPServerInstructionsDescriptive:
		args = append(args, "--instructions", mcp.DescriptiveInstructions)
	case MCPServerInstructionsPrescriptive:
		args = append(args, "--instructions", mcp.PrescriptiveInstructions)
	default:
		return nil, fmt.Errorf("unsupported MCP server instruction mode %q", r.mcpServerInstructionsMode)
	}
	return args, nil
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

func openCodeProviderModel(model string) string {
	if model == "" {
		model = openCodeDefaultModel
	}
	if !strings.Contains(model, "/") {
		return "amdbeast/" + model
	}
	return model
}
