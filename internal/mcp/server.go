// Package mcp exposes registry-defined semantic operations over the MCP stdio transport.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/gocache"
	"semedit/internal/maven"
	"semedit/internal/operation"
	"semedit/internal/symbol"
	"semedit/internal/telemetry"
)

// Server handles MCP JSON-RPC requests over stdio streams.
type Server struct {
	profile             string
	workDir             string
	service             *backend.Service
	registry            *operation.Registry
	liveReload          bool
	instructions        string
	goBaseDir           string
	startedAt           time.Time
	sessionMu           sync.Mutex
	initializedAt       time.Time
	firstSemanticCallAt time.Time
	outMu               sync.Mutex
	out                 io.Writer
}

const (
	// DescriptiveInstructions advertises semantic operations without requiring their use.
	DescriptiveInstructions = "Semedit semantic tools are available for supported source-code operations."
	// PrescriptiveInstructions is the controlled server-level policy used by benchmark instruction experiments.
	PrescriptiveInstructions = "Inspect the complete tool inventory, including deferred or lazy tools provided by the current environment, before editing. Confirm that applicable semantic-editing tools are callable; do not infer that a tool is absent from an initially visible subset. Use a semantic-editing tool for applicable mutations. Use ordinary file editing only when no applicable callable semantic tool exists, or when it fails, is unsupported, or is ambiguous."
)

// Option configures a Server instance.
type Option func(*Server)

// WithLiveReload enables in-place re-exec and dynamic tool updates.
func WithLiveReload(enabled bool) Option {
	return func(s *Server) { s.liveReload = enabled }
}

// WithInstructions sets optional server-wide guidance returned during MCP initialization.
func WithInstructions(instructions string) Option {
	return func(s *Server) { s.instructions = strings.TrimSpace(instructions) }
}

// WithGoBaseDir configures the directory used for Go subprocess state.
func WithGoBaseDir(baseDir string) Option {
	return func(s *Server) {
		if baseDir = strings.TrimSpace(baseDir); baseDir != "" {
			s.goBaseDir = baseDir
		}
	}
}

// WithService injects the language service used by registry dispatch.
func WithService(service *backend.Service) Option {
	return func(s *Server) { s.service = service }
}

// WithRegistry injects operation definitions for transport tests and composition.
func WithRegistry(registry *operation.Registry) Option {
	return func(s *Server) { s.registry = registry }
}

// NewServer initializes an MCP server instance.
func NewServer(profile string, workDir string, out io.Writer, opts ...Option) *Server {
	if profile == "" {
		profile = "full"
	}
	if workDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workDir = wd
		} else {
			workDir = "."
		}
	}
	s := &Server{
		profile:   profile,
		workDir:   workDir,
		service:   backends.NewDefaultService(),
		registry:  operation.DefaultRegistry(),
		goBaseDir: filepath.Join(workDir, ".scratch", "go"),
		startedAt: time.Now(),
		out:       out,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type feedbackReport struct {
	Intent           string         `json:"intent"`
	Interface        string         `json:"interface"`
	Command          string         `json:"command"`
	Parameters       map[string]any `json:"parameters"`
	TargetContext    string         `json:"target_context,omitempty"`
	ObservedResult   string         `json:"observed_result"`
	UnexpectedReason string         `json:"unexpected_reason"`
	ManualTouchups   string         `json:"manual_touchups"`
	SuspectedCause   string         `json:"suspected_cause,omitempty"`
	SuggestedFix     string         `json:"suggested_fix,omitempty"`
}

const schemaPropertiesKey = "properties"

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve reads newline-delimited JSON-RPC requests from in and writes responses to out.
func (s *Server) Serve(ctx context.Context, in io.Reader) error {
	ctx = gocache.WithBaseDir(ctx, s.goBaseDir)
	if _, err := gocache.Environment(ctx, s.workDir); err != nil {
		return fmt.Errorf("prepare MCP Go base directory: %w", err)
	}

	reader := bufio.NewReader(in)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read jsonrpc input: %w", err)
		}
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(trimmed), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}
		s.handleRequest(ctx, &req)
	}
}

func (s *Server) handleRequest(ctx context.Context, req *jsonRPCRequest) {
	isNotification := len(req.ID) == 0
	switch req.Method {
	case "initialize":
		var initParams struct {
			RootURI          string `json:"rootUri"`
			RootPath         string `json:"rootPath"`
			WorkspaceFolders []struct {
				URI string `json:"uri"`
			} `json:"workspaceFolders"`
		}
		if err := json.Unmarshal(req.Params, &initParams); err == nil {
			switch {
			case initParams.RootURI != "":
				s.workDir = strings.TrimPrefix(initParams.RootURI, "file://")
			case initParams.RootPath != "":
				s.workDir = initParams.RootPath
			case len(initParams.WorkspaceFolders) > 0 && initParams.WorkspaceFolders[0].URI != "":
				s.workDir = strings.TrimPrefix(initParams.WorkspaceFolders[0].URI, "file://")
			}
		}
		s.markInitialized(time.Now())
		result := map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
			"serverInfo":      map[string]any{"name": "semedit", "version": "0.1.0"},
		}
		if s.instructions != "" {
			result["instructions"] = s.instructions
		}
		s.sendResult(req.ID, result)
	case "notifications/initialized":
		if s.liveReload {
			s.notifyToolsListChanged()
		}
	case "ping":
		if !isNotification {
			s.sendResult(req.ID, map[string]any{})
		}
	case "tools/list":
		s.sendResult(req.ID, map[string]any{"tools": s.listTools()})
	case "tools/call":
		s.handleToolCall(ctx, req.ID, req.Params)
	default:
		if !isNotification {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func (s *Server) listTools() []map[string]any {
	tools := make([]map[string]any, 0, len(s.registry.All())+3)
	batchable := make([]operation.Entry, 0)
	for _, entry := range s.registry.All() {
		if entry.MCPName == "" || (s.profile == "mutations-only" && entry.ReadOnly) {
			continue
		}
		tools = append(tools, toolSchema(entry))
		if entry.Batchable {
			batchable = append(batchable, entry)
		}
	}
	tools = append(tools, batchToolSchema(batchable))
	if s.liveReload {
		tools = append(tools, map[string]any{
			"name": "semantic_reload", "description": "Use this tool instead of restarting the client or MCP process after binary promotion when live reload is enabled; re-execute the server and announce updated tools.",
			"inputSchema":  map[string]any{"type": "object", schemaPropertiesKey: map[string]any{}, "additionalProperties": false},
			"outputSchema": reloadOutputSchema(),
		})
	}
	tools = append(tools, feedbackToolSchema())
	return tools
}

func toolSchema(entry operation.Entry) map[string]any {
	return map[string]any{
		"name":         entry.MCPName,
		"description":  entry.Summary,
		"inputSchema":  operationInputSchema(entry),
		"outputSchema": standardOutputSchema(map[string]any{"type": "string"}),
	}
}

func standardOutputSchema(resultSchema map[string]any) map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"result":          resultSchema,
			"metrics":         metricsOutputSchema(),
			"session_metrics": sessionMetricsOutputSchema(),
		},
		"required": []string{"result", "metrics"},
	}
}

func metricsOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"schema_version": map[string]any{"type": "integer"},
			"total_ms":       map[string]any{"type": "integer"},
			"phases": map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "object", schemaPropertiesKey: map[string]any{"count": map[string]any{"type": "integer"}, "duration_ms": map[string]any{"type": "integer"}}},
			},
		},
	}
}

func sessionMetricsOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"server_start_to_initialize_ms":        map[string]any{"type": "integer"},
			"initialize_to_first_semantic_call_ms": map[string]any{"type": "integer"},
		},
	}
}

func batchOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"status": map[string]any{"type": "string", "enum": []string{"ok", "error"}},
			"results": map[string]any{
				"type":  "array",
				"items": batchResultSchema(),
			},
			"diagnostic_delta": diagnosticDeltaSchema(),
			"final_diff":       map[string]any{"type": "string"},
		},
		"required": []string{"status", "results", "final_diff"},
	}
}

func batchResultSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"tool":   map[string]any{"type": "string"},
			"symbol": map[string]any{"type": "string"},
			"status": map[string]any{"type": "string", "enum": []string{"ok", "error"}},
			"diff":   map[string]any{"type": "string"},
			"error":  map[string]any{"type": "string"},
		},
		"required": []string{"tool", "status"},
	}
}

func diagnosticDeltaSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"before":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"after":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"net_delta":   map[string]any{"type": "integer"},
			"introduced":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"resolved":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"suggestions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"before", "after", "net_delta", "introduced", "resolved"},
	}
}

func reloadOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"result": map[string]any{
				"type":              "object",
				schemaPropertiesKey: map[string]any{"status": map[string]any{"const": "reloading"}},
				"required":          []string{"status"},
			},
		},
		"required": []string{"result"},
	}
}

func operationInputSchema(entry operation.Entry) map[string]any {
	properties := make(map[string]any, len(entry.Params))
	required := make([]string, 0, len(entry.Params))
	for _, param := range entry.Params {
		property := map[string]any{"description": param.Description}
		switch param.Type {
		case operation.ParamBoolean:
			property["type"] = "boolean"
		case operation.ParamStringSlice:
			property["type"] = "array"
			property["items"] = map[string]any{"type": "string"}
		default:
			property["type"] = "string"
		}
		if len(param.Enums) > 0 {
			property["enum"] = param.Enums
		}
		if param.Default != nil {
			property["default"] = param.Default
		}
		properties[param.JSONName] = property
		if param.Required {
			required = append(required, param.JSONName)
		}
	}
	schema := map[string]any{"type": "object", schemaPropertiesKey: properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func batchToolSchema(entries []operation.Entry) map[string]any {
	branches := make([]any, 0, len(entries))
	for _, entry := range entries {
		branches = append(branches, map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"tool":   map[string]any{"const": entry.MCPName},
				"params": operationInputSchema(entry),
			},
			"required":             []string{"tool", "params"},
			"additionalProperties": false,
		})
	}
	return map[string]any{
		"name": "semantic_batch", "description": "Use this tool instead of a sequence of built-in text patches when several registered semantic edits must run in order. Stop at the first failure. Returns a final_diff covering semantic edits and deferred formatting/import changes; successful batches also return one final diagnostic_delta.",
		"inputSchema": map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"edits":                 map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"oneOf": branches}},
				"auto_organize_imports": map[string]any{"type": "boolean", "default": false},
			},
			"required": []string{"edits"},
		},
		"outputSchema": standardOutputSchema(batchOutputSchema()),
	}
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) handleToolCall(ctx context.Context, id json.RawMessage, rawParams json.RawMessage) {
	var params toolCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		s.sendError(id, -32602, "Invalid params")
		return
	}
	if params.Name == "semantic_reload" {
		s.handleReload(id)
		return
	}
	if params.Name == "report_feedback" {
		s.handleFeedbackReport(ctx, id, params.Arguments)
		return
	}
	if params.Name == "semantic_batch" {
		s.handleBatch(ctx, id, params.Arguments)
		return
	}
	entry, ok := s.registry.LookupMCP(params.Name)
	if !ok || (s.profile == "mutations-only" && entry.ReadOnly) {
		s.sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}
	timing := newToolRequestTiming(ctx, s.firstSemanticCallMetrics(time.Now()))
	ctx = timing.Context(ctx)
	raw := map[string]any{}
	finishArguments := telemetry.Start(ctx, telemetry.PhaseArgumentParsing)
	if len(params.Arguments) > 0 && string(params.Arguments) != "null" {
		if err := json.Unmarshal(params.Arguments, &raw); err != nil {
			finishArguments()
			s.sendToolErrorWithTiming(id, fmt.Sprintf("invalid arguments: %v", err), timing, err)
			return
		}
	}
	finishArguments()
	finishDispatch := telemetry.Start(ctx, telemetry.PhaseDispatch)
	result, err := s.registry.Dispatch(operation.CallContext{Ctx: ctx, WorkDir: s.workDir, Registry: s.registry, Service: s.service}, entry.Key, raw)
	finishDispatch()
	if err != nil {
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
	text, err := entry.Format(result)
	finishResponse()
	if err != nil {
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	s.sendToolSuccessWithTiming(id, text, timing)
}

func (s *Server) handleReload(id json.RawMessage) {
	if !s.liveReload {
		s.sendError(id, -32601, "Unknown tool: semantic_reload")
		return
	}
	s.sendToolSuccessWithStructuredContent(id, `{"status":"reloading"}`, map[string]any{"result": map[string]any{"status": "reloading"}})
	if err := execReload(); err != nil {
		s.sendToolError(id, fmt.Sprintf("reload error: %v", err), err)
	}
}

func (s *Server) sendToolSuccessWithStructuredContent(id json.RawMessage, text string, structuredContent map[string]any) {
	s.sendResult(id, map[string]any{
		"content":           []map[string]any{{"type": "text", "text": text}},
		"isError":           false,
		"structuredContent": structuredContent,
	})
}

func (s *Server) sendToolSuccessWithTiming(id json.RawMessage, text string, timing *toolRequestTiming) {
	s.sendToolSuccessWithTimingAndResult(id, text, text, timing)
}

func (s *Server) sendToolSuccessWithTimingAndResult(id json.RawMessage, text string, result any, timing *toolRequestTiming) {
	s.sendResult(id, map[string]any{
		"content":           []map[string]any{{"type": "text", "text": text}},
		"isError":           false,
		"structuredContent": timing.structuredContentWithResult(result),
	})
}

func (s *Server) sendToolError(id json.RawMessage, text string, errs ...error) {
	result := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "isError": true}
	if len(errs) > 0 && errs[0] != nil {
		var mavenErr *maven.Error
		if errors.As(errs[0], &mavenErr) && mavenErr.MavenResult() != nil {
			result["structuredContent"] = map[string]any{"error": text, "result": mavenErr.MavenResult()}
		}
		if loc := s.extractLocation(errs[0]); loc != nil {
			result["location"] = loc
		}
	}
	s.sendResult(id, result)
}

func (s *Server) sendToolErrorWithTiming(id json.RawMessage, text string, timing *toolRequestTiming, errs ...error) {
	result := map[string]any{
		"content":           []map[string]any{{"type": "text", "text": text}},
		"isError":           true,
		"structuredContent": timing.structuredContent(),
	}
	if len(errs) > 0 && errs[0] != nil {
		var mavenErr *maven.Error
		if errors.As(errs[0], &mavenErr) && mavenErr.MavenResult() != nil {
			result["structuredContent"].(map[string]any)["error"] = text
			result["structuredContent"].(map[string]any)["result"] = mavenErr.MavenResult()
		}
		if loc := s.extractLocation(errs[0]); loc != nil {
			result["location"] = loc
		}
	}
	s.sendResult(id, result)
}

type toolRequestTiming struct {
	metrics *telemetry.Metrics
	started time.Time
	session *StartupMetrics
}

func newToolRequestTiming(_ context.Context, session *StartupMetrics) *toolRequestTiming {
	return &toolRequestTiming{metrics: telemetry.NewMetrics(), started: time.Now(), session: session}
}

func (t *toolRequestTiming) Context(ctx context.Context) context.Context {
	return telemetry.WithMetrics(ctx, t.metrics)
}

func (t *toolRequestTiming) Snapshot() telemetry.Snapshot {
	return t.metrics.Snapshot(time.Since(t.started))
}

func (t *toolRequestTiming) structuredContent() map[string]any {
	content := map[string]any{"metrics": t.Snapshot()}
	if t.session != nil {
		content["session_metrics"] = t.session
	}
	return content
}

func (t *toolRequestTiming) structuredContentWithResult(result any) map[string]any {
	content := t.structuredContent()
	content["result"] = result
	return content
}

// StartupMetrics records one server session's initialization and first semantic request delays.
type StartupMetrics struct {
	ServerStartToInitializeMS       int64 `json:"server_start_to_initialize_ms"`
	InitializeToFirstSemanticCallMS int64 `json:"initialize_to_first_semantic_call_ms"`
}

func (s *Server) markInitialized(at time.Time) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.initializedAt.IsZero() {
		s.initializedAt = at
	}
}

func (s *Server) firstSemanticCallMetrics(at time.Time) *StartupMetrics {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.initializedAt.IsZero() || !s.firstSemanticCallAt.IsZero() {
		return nil
	}
	s.firstSemanticCallAt = at
	return &StartupMetrics{
		ServerStartToInitializeMS:       s.initializedAt.Sub(s.startedAt).Milliseconds(),
		InitializeToFirstSemanticCallMS: at.Sub(s.initializedAt).Milliseconds(),
	}
}

func (s *Server) extractLocation(err error) map[string]any {
	var pos token.Position
	var filePath string
	var synErr *astedit.SyntaxError
	var symErr *symbol.SymbolError
	var visErr *astedit.VisibilityMismatchError
	var placeErr *astedit.PlacementError
	switch {
	case errors.As(err, &synErr) && synErr.Pos.IsValid():
		pos, filePath = synErr.Pos, synErr.File
	case errors.As(err, &symErr) && symErr.Pos.IsValid():
		pos, filePath = symErr.Pos, symErr.File
	case errors.As(err, &visErr) && visErr.Pos.IsValid():
		pos, filePath = visErr.Pos, visErr.File
	case errors.As(err, &placeErr) && placeErr.Pos.IsValid():
		pos, filePath = placeErr.Pos, placeErr.File
	}
	if !pos.IsValid() {
		return nil
	}
	if filePath == "" {
		filePath = pos.Filename
	}
	uri := "snippet:///source"
	if filePath != "" && filePath != "snippet" && filePath != "snippet.go" {
		if !strings.HasPrefix(filePath, "file://") {
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(s.workDir, filePath)
			}
			filePath = (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(filePath))}).String()
		}
		uri = filePath
	}
	character := s.resolveLSPCharacter(pos, filePath, synErr)
	return map[string]any{"uri": uri, "range": map[string]any{"start": map[string]int{"line": max(pos.Line-1, 0), "character": character}, "end": map[string]int{"line": max(pos.Line-1, 0), "character": character}}}
}

func (s *Server) resolveLSPCharacter(pos token.Position, filePath string, synErr *astedit.SyntaxError) int {
	if synErr != nil && synErr.Snippet != "" {
		lines := strings.Split(synErr.Snippet, "\n")
		if line := pos.Line - 1; line >= 0 && line < len(lines) {
			return utf16Length(lines[line], pos.Column-1)
		}
	}
	if filePath != "" && !strings.HasPrefix(filePath, "file://") {
		if data, err := os.ReadFile(filepath.Clean(filePath)); err == nil {
			lines := strings.Split(string(data), "\n")
			if line := pos.Line - 1; line >= 0 && line < len(lines) {
				return utf16Length(lines[line], pos.Column-1)
			}
		}
	}
	return max(pos.Column-1, 0)
}

func utf16Length(line string, byteOffset int) int {
	byteOffset = min(max(byteOffset, 0), len(line))
	count := 0
	for _, r := range line[:byteOffset] {
		count += utf16.RuneLen(r)
	}
	return count
}

func (s *Server) sendResult(id json.RawMessage, result any) {
	s.writeJSON(&jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) sendError(id json.RawMessage, code int, message string) {
	s.writeJSON(&jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: code, Message: message}})
}

func (s *Server) writeJSON(resp *jsonRPCResponse) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if data, err := json.Marshal(resp); err == nil {
		_, _ = s.out.Write(append(data, '\n'))
	}
}

func (s *Server) notifyToolsListChanged() {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"}); err == nil {
		_, _ = s.out.Write(append(data, '\n'))
	}
}

func (s *Server) handleFeedbackReport(ctx context.Context, id json.RawMessage, arguments json.RawMessage) {
	timing := newToolRequestTiming(ctx, nil)
	ctx = timing.Context(ctx)
	finishArguments := telemetry.Start(ctx, telemetry.PhaseArgumentParsing)
	decoder := json.NewDecoder(strings.NewReader(string(arguments)))
	decoder.DisallowUnknownFields()
	var report feedbackReport
	if err := decoder.Decode(&report); err != nil {
		finishArguments()
		err = fmt.Errorf("invalid feedback arguments: %w", err)
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values are not allowed")
		}
		finishArguments()
		err = fmt.Errorf("invalid feedback arguments: %w", err)
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	if err := validateFeedbackReport(report); err != nil {
		finishArguments()
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	finishArguments()

	finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
	date := time.Now().Format("2006-01-02")
	parameters, err := json.MarshalIndent(report.Parameters, "", "  ")
	if err != nil {
		finishResponse()
		err = fmt.Errorf("format feedback parameters: %w", err)
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	var draft strings.Builder
	fmt.Fprintf(&draft, "# Feedback report draft (%s)\n\nReview this draft for accuracy and sensitive content before manually posting it as a GitHub issue. It has not been saved or posted.\n\n", date)
	fmt.Fprintf(&draft, "## Intended task\n\n%s\n\n", report.Intent)
	fmt.Fprintf(&draft, "## Command\n\nInterface: %s\n\nCommand: %s\n\nParameters:\n\n```json\n%s\n```\n\n", report.Interface, report.Command, parameters)
	if report.TargetContext != "" {
		fmt.Fprintf(&draft, "## Target context\n\n%s\n\n", report.TargetContext)
	}
	fmt.Fprintf(&draft, "## Observed result\n\n%s\n\n", report.ObservedResult)
	fmt.Fprintf(&draft, "## Why it was unexpected\n\n%s\n\n", report.UnexpectedReason)
	fmt.Fprintf(&draft, "## Manual touch-ups\n\n%s\n\n", report.ManualTouchups)
	if report.SuspectedCause != "" {
		fmt.Fprintf(&draft, "## Suspected cause\n\n%s\n\n", report.SuspectedCause)
	}
	if report.SuggestedFix != "" {
		fmt.Fprintf(&draft, "## Suggested fix\n\n%s\n\n", report.SuggestedFix)
	}
	finishResponse()

	result := map[string]any{
		"status":   "draft_only",
		"date":     date,
		"report":   report,
		"markdown": draft.String(),
	}
	s.sendToolSuccessWithTimingAndResult(id, draft.String(), result, timing)
}

func feedbackToolSchema() map[string]any {
	const (
		descriptionKey          = "description"
		additionalPropertiesKey = "additionalProperties"
	)
	stringProperty := func(description string) map[string]any {
		return map[string]any{"type": "string", "minLength": 1, descriptionKey: description}
	}
	properties := map[string]any{
		"intent":            stringProperty("In a few sentences, describe what the user was trying to accomplish. Keep project details suitable for sharing."),
		"interface":         map[string]any{"type": "string", "enum": []string{"mcp", "cli", "other"}, descriptionKey: "Where the command was run."},
		"command":           stringProperty("Exact semedit MCP tool name or CLI command that was executed."),
		"parameters":        map[string]any{"type": "object", additionalPropertiesKey: true, descriptionKey: "The command parameters exactly as supplied. Omit proprietary values and source code."},
		"target_context":    map[string]any{"type": "string", descriptionKey: "Optional concise target context, such as a public project-relative file path or symbol. Do not paste source text or private paths."},
		"observed_result":   stringProperty("Describe the result or error that actually occurred. Do not include source code, secrets, or private data."),
		"unexpected_reason": stringProperty("State what was expected instead and why the observed result fell short."),
		"manual_touchups":   stringProperty("Describe manual edits or extra commands needed afterward. Write None if no follow-up was needed."),
		"suspected_cause":   map[string]any{"type": "string", descriptionKey: "Optional suspected cause or missing invariant, if known."},
		"suggested_fix":     map[string]any{"type": "string", descriptionKey: "Optional candidate fix or follow-up, if known."},
	}
	reportSchema := map[string]any{
		"type":                  "object",
		schemaPropertiesKey:     properties,
		"required":              []string{"intent", "interface", "command", "parameters", "observed_result", "unexpected_reason", "manual_touchups"},
		additionalPropertiesKey: false,
	}
	return map[string]any{
		"name":         "report_feedback",
		descriptionKey: "Use this tool instead of manually drafting a tool issue when a semedit command or MCP tool fails, surprises you, or needs a manual touch-up. Prepare a structured report about that friction. Use after an unexpected result, an error, or a required manual touch-up. The report is a draft for the user to review for accuracy and sensitive content, then post manually as a GitHub issue if appropriate; this tool does not save or post feedback. Privacy: do not include source code, credentials, personal or customer data, private paths, proprietary business details, or intellectual property. For an open-source project, public project and tool details are generally okay to include, but still omit secrets and private information. If unsure whether a detail is safe to share or whether the project is open source, ask the user first and leave uncertain details out until approved.",
		"inputSchema": map[string]any{
			"type":                  "object",
			schemaPropertiesKey:     properties,
			"required":              []string{"intent", "interface", "command", "parameters", "observed_result", "unexpected_reason", "manual_touchups"},
			additionalPropertiesKey: false,
		},
		"outputSchema": standardOutputSchema(map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"status":   map[string]any{"const": "draft_only"},
				"date":     map[string]any{"type": "string"},
				"report":   reportSchema,
				"markdown": map[string]any{"type": "string"},
			},
			"required":              []string{"status", "date", "report", "markdown"},
			additionalPropertiesKey: false,
		}),
	}
}

func validateFeedbackReport(report feedbackReport) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "intent", value: report.Intent},
		{name: "interface", value: report.Interface},
		{name: "command", value: report.Command},
		{name: "observed_result", value: report.ObservedResult},
		{name: "unexpected_reason", value: report.UnexpectedReason},
		{name: "manual_touchups", value: report.ManualTouchups},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("feedback field %q is required", field.name)
		}
	}
	if report.Parameters == nil {
		return errors.New("feedback field \"parameters\" must be an object; use an empty object when there are no parameters")
	}
	switch report.Interface {
	case "mcp", "cli", "other":
	default:
		return errors.New("feedback field \"interface\" must be mcp, cli, or other")
	}
	return nil
}
