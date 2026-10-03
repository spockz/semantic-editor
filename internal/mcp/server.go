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
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/backends"
	"semedit/internal/gocache"
	"semedit/internal/godiagnostics"
	"semedit/internal/maven"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/projectverify"
	"semedit/internal/symbol"
	"semedit/internal/telemetry"
)

// Server handles MCP JSON-RPC requests over stdio streams.
type Server struct {
	profile               string
	toolProfile           string
	toolVariantKeys       map[string]struct{}
	toolVariantPatterns   []string
	toolVariantErr        error
	workDir               string
	expectedRoot          string
	service               *backend.Service
	diagnostics           *godiagnostics.SessionManager
	registry              *operation.Registry
	registryValidationErr error
	liveReload            bool
	instructions          string
	goBaseDir             string
	explicitGoBaseDir     bool
	workspaceTrust        backend.WorkspaceTrust
	initializationFailed  bool
	enabledLanguageText   string
	enabledLanguages      []backend.LanguageID
	enabledLanguageErr    error
	startedAt             time.Time
	sessionMu             sync.Mutex
	initializedAt         time.Time
	firstSemanticCallAt   time.Time
	metricsMu             sync.Mutex
	metricSequence        uint64
	recentMetrics         []toolMetricsRecord
	metricsFile           string
	metricsFileLoaded     bool
	fileMetrics           []toolMetricsRecord
	metricsPersistErr     error
	outMu                 sync.Mutex
	out                   io.Writer
	verificationPlan      projectverify.Plan
}

const structuredContentKey = "structuredContent"

const (
	// DescriptiveInstructions advertises semantic operations without requiring their use.
	DescriptiveInstructions = "Semedit semantic tools are available for supported source-code operations."
	// PrescriptiveInstructions is the controlled server-level policy used by benchmark instruction experiments.
	PrescriptiveInstructions    = "Inspect the complete tool inventory, including deferred or lazy tools provided by the current environment, before editing. Confirm that applicable semantic-editing tools are callable; do not infer that a tool is absent from an initially visible subset. Use a semantic-editing tool for applicable mutations. Use ordinary file editing only when no applicable callable semantic tool exists, or when it fails, is unsupported, or is ambiguous."
	toolFieldDescription        = "description"
	toolFieldInputSchema        = "inputSchema"
	toolFieldOutputSchema       = "outputSchema"
	semanticInsertConstructTool = "semantic_insert_construct"
	semanticExtractVariableTool = "semantic_extract_" + "variable"
	semanticMoveConstructTool   = "semantic_move_construct"
	additionalPropertiesKey     = "additionalProperties"
)

const semanticVerifyToolName = "semantic_verify"

const finalDiffToolName = "final_diff"

var errNoRegisteredParameterChoices = errors.New("no parameter choices for registered backends")

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

// WithMetricsFile preserves every timed call in a file for benchmark evidence.
func WithMetricsFile(path string) Option {
	return func(s *Server) { s.metricsFile = strings.TrimSpace(path) }
}

// WithToolProfile selects the MCP catalog profile: base exposes generic tools,
// while full exposes every registered operation.
func WithToolProfile(profile string) Option {
	return func(s *Server) { s.toolProfile = profile }
}

// WithToolVariants enables specialized verb-kind variants in the MCP catalog.
func WithToolVariants(values []string) Option {
	return func(s *Server) {
		s.toolVariantPatterns = append([]string(nil), values...)
	}
}

// WithGoBaseDir configures the directory used for Go subprocess state.
func WithGoBaseDir(baseDir string) Option {
	return func(s *Server) {
		if baseDir = strings.TrimSpace(baseDir); baseDir != "" {
			s.goBaseDir = baseDir
			s.explicitGoBaseDir = true
		}
	}
}

// WithExpectedRoot rejects client initialization that selects a different workspace.
func WithExpectedRoot(root string) Option {
	return func(s *Server) {
		if root = strings.TrimSpace(root); root != "" {
			s.expectedRoot = backend.CanonicalWorkspaceRoot(root)
		}
	}
}

func workspaceRootFromURI(raw string) (string, error) {
	root, err := pathutil.FilePathFromURI(raw)
	if err != nil {
		return "", fmt.Errorf("parse workspace root URI %q: %w", raw, err)
	}
	return root, nil
}

// WithEnabledLanguages constrains language-specific tools to an explicit allowlist.
func WithEnabledLanguages(value string) Option {
	return func(s *Server) {
		s.enabledLanguageText = value
		s.enabledLanguages, s.enabledLanguageErr = backend.ParseEnabledLanguages(value)
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
	s := &Server{
		profile:     profile,
		toolProfile: "base",
		workDir:     workDir,
		service:     backends.NewDefaultService(),
		diagnostics: godiagnostics.NewSessionManager(),
		registry:    operation.DefaultRegistry(),
		startedAt:   time.Now(),
		out:         out,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.toolProfile == "" {
		s.toolProfile = "base"
	}
	if s.toolVariantErr == nil {
		s.toolVariantKeys, s.toolVariantErr = operation.ParseToolVariants(s.toolVariantPatterns, s.registry.All())
	}
	if s.toolVariantErr == nil && s.toolProfile != "base" && s.toolProfile != "full" {
		s.toolVariantErr = fmt.Errorf("invalid MCP tool profile %q; expected base or full", s.toolProfile)
	}
	if s.toolVariantErr == nil && s.toolProfile == "full" {
		s.toolVariantKeys = map[string]struct{}{}
		for _, entry := range s.registry.All() {
			if entry.VariantVerb != "" {
				s.toolVariantKeys[entry.Key] = struct{}{}
			}
		}
	}
	s.registryValidationErr = ValidateSourceFields(s.registry.All())
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
const schemaDescriptionKey = "description"

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
// Cancellation closes in if it implements io.Closer, including caller-supplied files
// and pipes. Otherwise the caller must unblock any read before Serve can finish
// and release its diagnostic processes.
func (s *Server) Serve(ctx context.Context, in io.Reader) error {
	readClosed := make(chan error, 1)
	// Stdio and pipes must unblock an idle read before persistent LSP cleanup can run.
	stop := context.AfterFunc(ctx, func() {
		var closeErr error
		if closer, ok := in.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				closeErr = fmt.Errorf("close canceled MCP input: %w", err)
			}
		}
		readClosed <- closeErr
	})
	runErr := s.serveRequests(ctx, in)
	if !stop() {
		runErr = errors.Join(runErr, <-readClosed)
	}
	if err := ctx.Err(); err != nil {
		runErr = errors.Join(err, runErr)
	}
	return errors.Join(runErr, s.closeDiagnosticSessions())
}

// diagnosticSession shares one manager because it keys Go processes by enclosing module, not request root.
func (s *Server) diagnosticSession(_ string) *godiagnostics.SessionManager {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.diagnostics == nil {
		s.diagnostics = godiagnostics.NewSessionManager()
	}
	return s.diagnostics
}

func (s *Server) requestView(root string, trusted bool) *Server {
	return &Server{
		profile:             s.profile,
		workDir:             root,
		service:             s.service,
		registry:            s.registry,
		workspaceTrust:      canonicalTrust(root, trusted),
		goBaseDir:           s.goBaseDirForRoot(root),
		enabledLanguages:    s.enabledLanguages,
		enabledLanguageText: s.enabledLanguageText,
		enabledLanguageErr:  s.enabledLanguageErr,
	}
}

func (s *Server) closeDiagnosticSessions() error {
	s.sessionMu.Lock()
	manager := s.diagnostics
	s.diagnostics = godiagnostics.NewSessionManager()
	s.sessionMu.Unlock()
	if manager != nil {
		return manager.Close()
	}
	return nil
}

func (s *Server) goBaseDirForRoot(root string) string {
	if root == s.workDir && s.goBaseDir != "" {
		return s.goBaseDir
	}
	if relative, err := filepath.Rel(s.workDir, root); err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		worktreePrefix := filepath.Join(".scratch", "worktrees") + string(filepath.Separator)
		if relative != "." && !strings.HasPrefix(relative, worktreePrefix) && s.goBaseDir != "" {
			return s.goBaseDir
		}
	}
	return filepath.Join(root, ".scratch", "go")
}

func (s *Server) serveRequests(ctx context.Context, in io.Reader) error {
	if s.toolVariantErr != nil {
		return fmt.Errorf("configure MCP tool catalog: %w", s.toolVariantErr)
	}
	if s.enabledLanguageErr != nil {
		return fmt.Errorf("parse --enabled-languages: %w", s.enabledLanguageErr)
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
		s.metricsMu.Lock()
		metricsErr := s.metricsPersistErr
		s.metricsMu.Unlock()
		if metricsErr != nil {
			return metricsErr
		}
	}
}

// Initialize processes client initialization parameters, establishes workspace root and Go cache environments,
// and marks the server initialized.
func (s *Server) Initialize(ctx context.Context, rawParams json.RawMessage) (map[string]any, error) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.initializationFailed {
		return nil, errors.New("MCP initialization previously failed")
	}
	initParams, err := parseInitializeParams(rawParams)
	if err != nil {
		return nil, err
	}
	selectedRoot, err := initializeWorkspaceRoot(s.workDir, initParams)
	if err != nil {
		return nil, err
	}
	actualRoot := backend.CanonicalWorkspaceRoot(selectedRoot)
	if s.expectedRoot != "" && actualRoot != s.expectedRoot {
		s.initializationFailed = true
		return nil, fmt.Errorf("MCP workspace root mismatch: expected %q, actual %q", s.expectedRoot, actualRoot)
	}
	s.workDir = actualRoot
	if !s.explicitGoBaseDir || s.goBaseDir == "" {
		s.goBaseDir = filepath.Join(s.workDir, ".scratch", "go")
	}
	initCtx := gocache.WithBaseDir(ctx, s.goBaseDir)
	if _, cacheErr := gocache.Environment(initCtx, s.workDir); cacheErr != nil {
		return nil, fmt.Errorf("prepare MCP Go base directory: %w", cacheErr)
	}
	s.workspaceTrust = backend.NewWorkspaceTrust(s.workDir, initParams.InitializationOptions.TrustWorkspace)
	plan, err := projectverify.Discover(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("discover project verification plan: %w", err)
	}
	s.verificationPlan = plan
	if s.initializedAt.IsZero() {
		s.initializedAt = time.Now()
	}

	serverInfo := map[string]any{
		"name":        "semedit",
		"version":     "0.1.0",
		"profile":     s.profile,
		"toolCatalog": "operation-registry",
	}
	if binaryPath, err := os.Executable(); err == nil {
		if resolvedPath, err := filepath.EvalSymlinks(binaryPath); err == nil {
			binaryPath = resolvedPath
		}
		serverInfo["binaryPath"] = binaryPath
	}
	result := map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
		"serverInfo":      serverInfo,
		"workspaceRoot":   s.workDir,
	}
	if s.instructions != "" {
		result["instructions"] = s.instructions
	}
	return result, nil
}

type initializeParams struct {
	RootURI          string `json:"rootUri"`
	RootPath         string `json:"rootPath"`
	WorkspaceFolders []struct {
		URI string `json:"uri"`
	} `json:"workspaceFolders"`
	InitializationOptions struct {
		WorkspaceRoot  string `json:"workspace_root"`
		RootURI        string `json:"rootUri"`
		TrustWorkspace bool   `json:"trust_workspace"`
	} `json:"initializationOptions"`
}

func parseInitializeParams(raw json.RawMessage) (initializeParams, error) {
	var params initializeParams
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &params); err != nil {
			return initializeParams{}, fmt.Errorf("parse initialize params: %w", err)
		}
	}
	return params, nil
}

func initializeWorkspaceRoot(defaultRoot string, params initializeParams) (string, error) {
	selectedRoot := defaultRoot
	uri := ""
	switch {
	case params.RootURI != "":
		uri = params.RootURI
	case params.RootPath != "":
		selectedRoot = params.RootPath
	case len(params.WorkspaceFolders) > 0 && params.WorkspaceFolders[0].URI != "":
		uri = params.WorkspaceFolders[0].URI
	case params.InitializationOptions.RootURI != "":
		uri = params.InitializationOptions.RootURI
	case params.InitializationOptions.WorkspaceRoot != "":
		selectedRoot = params.InitializationOptions.WorkspaceRoot
	}
	if uri != "" {
		root, err := workspaceRootFromURI(uri)
		if err != nil {
			return "", err
		}
		selectedRoot = root
	}
	if selectedRoot == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve default MCP workspace root: %w", err)
		}
		return workingDirectory, nil
	}
	return selectedRoot, nil
}

func (s *Server) handleRequest(ctx context.Context, req *jsonRPCRequest) {
	isNotification := len(req.ID) == 0
	switch req.Method {
	case "initialize":
		result, err := s.Initialize(ctx, req.Params)
		if err != nil {
			s.sendError(req.ID, -32603, err.Error())
			return
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
		tools, err := s.listTools()
		if err != nil {
			s.sendError(req.ID, -32603, err.Error())
			return
		}
		s.sendResult(req.ID, map[string]any{"tools": tools})
	case "tools/call":
		s.handleToolCall(godiagnostics.WithSessionManager(ctx, s.diagnosticSession(s.workDir)), req.ID, req.Params)
	default:
		if !isNotification {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func (s *Server) listTools() ([]map[string]any, error) {
	if s.registryValidationErr != nil {
		return nil, fmt.Errorf("validate MCP continuation registry: %w", s.registryValidationErr)
	}
	tools := make([]map[string]any, 0, len(s.registry.All())+4)
	batchable := make([]operation.Entry, 0)
	languages, err := s.activeLanguages()
	if err != nil {
		return nil, err
	}
	activeBackends := s.backendsForLanguages(languages)
	activeLanguages := make([]backend.LanguageID, 0, len(activeBackends))
	for _, candidate := range activeBackends {
		activeLanguages = append(activeLanguages, candidate.Language())
	}
	for _, entry := range s.registry.All() {
		if entry.MCPName == "" || (s.profile == "mutations-only" && entry.ReadOnly) {
			continue
		}
		if !entrySupportsAny(entry, activeLanguages) {
			continue
		}
		tool, err := toolSchemaForLanguages(entry, activeBackends)
		if err != nil {
			if errors.Is(err, errNoRegisteredParameterChoices) {
				continue
			}
			return nil, fmt.Errorf("build schema for %q: %w", entry.MCPName, err)
		}
		addWorkspaceRootInput(tool[toolFieldInputSchema].(map[string]any))
		if entry.Batchable {
			batchable = append(batchable, entry)
		}
		if !s.exposesEntry(entry) {
			continue
		}
		tools = append(tools, tool)
	}
	if len(batchable) > 0 {
		batchTool, err := batchToolSchemaForLanguages(batchable, activeBackends)
		if err != nil {
			return nil, err
		}
		batchSchema := batchTool[toolFieldInputSchema].(map[string]any)
		properties := batchSchema[schemaPropertiesKey].(map[string]any)
		properties["workspace_root"] = workspaceRootParameterSchema()
		properties["trust_workspace"] = map[string]any{"type": "boolean", "default": false, "description": "Explicitly trust this selected workspace for configured verification hooks; applies only to this batch request."}
		tools = append(tools, batchTool)
	}
	if s.toolProfile == "full" {
		tools = append(tools, metricsToolSchema(), feedbackToolSchema())
	}
	if s.liveReload {
		tools = append(tools, map[string]any{
			"name": "semantic_reload", toolFieldDescription: "Use this tool instead of restarting the client or MCP process after binary promotion when live reload is enabled; re-execute the server and announce updated tools.",
			toolFieldInputSchema:  map[string]any{"type": "object", schemaPropertiesKey: map[string]any{}, additionalPropertiesKey: false},
			toolFieldOutputSchema: reloadOutputSchema(),
		})
	}
	return tools, nil
}

func (s *Server) exposesEntry(entry operation.Entry) bool {
	if s.toolProfile == "full" {
		return true
	}
	if entry.VariantVerb != "" || entry.VariantKind != "" {
		_, ok := s.toolVariantKeys[entry.Key]
		return ok
	}
	return baseToolKeys[entry.MCPName]
}

var baseToolKeys = map[string]bool{
	"semantic_insert_construct":  true,
	"semantic_replace_construct": true,
	"semantic_remove_construct":  true,
	semanticMoveConstructTool:    true,
	"semantic_rename":            true,
	"semantic_lookup":            true,
	"semantic_inspect_symbol":    true,
	"semantic_outline":           true,
	"semantic_find_references":   true,
	"semantic_verify":            true,
	"semantic_snapshot":          true,
	"semantic_undo":              true,
}

func (s *Server) backendsForLanguages(languages []backend.LanguageID) []backend.Backend {
	if s.service == nil || s.service.Registry == nil {
		return nil
	}
	registered := make([]backend.Backend, 0, len(languages))
	for _, language := range languages {
		if candidate, ok := s.service.Registry.Backend(language); ok {
			registered = append(registered, candidate)
		}
	}
	return registered
}

func (s *Server) activeLanguages() ([]backend.LanguageID, error) {
	detected, err := backend.DetectLanguages(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("detect workspace source languages: %w", err)
	}
	if len(detected) == 0 {
		if s.service != nil && s.service.Registry != nil {
			if selected, selectErr := s.service.Registry.Select(backend.ProjectContext{RootDir: s.workDir}); selectErr == nil {
				detected = []backend.LanguageID{selected.Language()}
			}
		}
	}
	if len(s.enabledLanguages) == 0 {
		return withoutImplicitHaskell(detected), nil
	}
	if len(detected) == 0 {
		return append([]backend.LanguageID(nil), s.enabledLanguages...), nil
	}
	allowed := make([]backend.LanguageID, 0, len(detected))
	for _, language := range detected {
		if slices.Contains(s.enabledLanguages, language) {
			allowed = append(allowed, language)
		}
	}
	return allowed, nil
}

func (s *Server) goEnabled() bool {
	return s.enabledLanguages == nil || slices.Contains(s.enabledLanguages, backend.LanguageGo)
}

func withoutImplicitHaskell(languages []backend.LanguageID) []backend.LanguageID {
	filtered := make([]backend.LanguageID, 0, len(languages))
	for _, language := range languages {
		if language != backend.LanguageHaskell {
			filtered = append(filtered, language)
		}
	}
	return filtered
}

func entrySupportsAny(entry operation.Entry, languages []backend.LanguageID) bool {
	if len(entry.Languages) == 0 || slices.Contains(entry.Languages, backend.LanguageAuto) {
		return true
	}
	for _, language := range languages {
		if slices.Contains(entry.Languages, language) {
			return true
		}
	}
	return false
}

func toolSchemaForLanguages(entry operation.Entry, backends []backend.Backend) (map[string]any, error) {
	tool, err := toolSchema(entry)
	if err != nil {
		return nil, err
	}
	tool["inputSchema"], err = operationInputSchemaForBackends(entry, backends)
	if err != nil {
		return nil, err
	}
	return tool, nil
}

func toolSchema(entry operation.Entry) (map[string]any, error) {
	toolOutput, err := toolOutputAndContinuations(entry)
	if err != nil {
		return nil, err
	}
	description := toolSchemaDescription(entry, toolOutput.continuations)
	inputSchema, err := operationInputSchema(entry)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name":                entry.MCPName,
		toolFieldDescription:  description,
		toolFieldInputSchema:  inputSchema,
		toolFieldOutputSchema: toolOutput.schema,
	}, nil
}

type toolOutputAndContinuationsResult struct {
	schema        map[string]any
	continuations map[string]any
}

func toolOutputAndContinuations(entry operation.Entry) (toolOutputAndContinuationsResult, error) {
	output := standardOutputSchema(map[string]any{"type": "string"})
	if entry.MCPName == "semantic_remove_construct" {
		output = standardOutputSchema(removeConstructResultSchema())
	}
	properties := output[schemaPropertiesKey].(map[string]any)
	continuations := continuationOutputFieldSchemas(entry.MCPName)
	if entry.Output != nil {
		if entry.Output.ResultSchema != nil {
			resultSchema, err := entry.Output.ResultSchema.JSONSchema()
			if err != nil {
				return toolOutputAndContinuationsResult{}, fmt.Errorf("operation %q result schema: %w", entry.MCPName, err)
			}
			output = standardOutputSchema(resultSchema)
			properties = output[schemaPropertiesKey].(map[string]any)
		}
		continuations = make(map[string]any, len(entry.Output.Continuations))
		for _, continuation := range entry.Output.Continuations {
			if continuation.Path != "" {
				continue
			}
			fieldSchema, err := continuation.Schema.JSONSchema()
			if err != nil {
				return toolOutputAndContinuationsResult{}, fmt.Errorf("operation %q continuation %q schema: %w", entry.MCPName, continuation.Field, err)
			}
			continuations[continuation.Field] = fieldSchema
		}
	}
	maps.Copy(properties, continuations)
	return toolOutputAndContinuationsResult{schema: output, continuations: continuations}, nil
}

func toolSchemaDescription(entry operation.Entry, continuations map[string]any) string {
	description := entry.Summary
	if entry.Output != nil && len(entry.Output.Continuations) > 0 {
		fieldNames := make([]string, 0, len(entry.Output.Continuations))
		for _, continuation := range entry.Output.Continuations {
			name := continuation.Field
			if continuation.Path != "" {
				name = continuation.Path
			}
			if name != "" {
				fieldNames = append(fieldNames, "`"+name+"` copied from the same result record to "+strings.Join(continuation.Consumers, ", "))
			}
		}
		slices.Sort(fieldNames)
		description += " Structured results may expose declared continuations when resolved: " + strings.Join(fieldNames, "; ") + ". An absent continuation means there is no resolved continuation target."
	} else if len(continuations) > 0 {
		fieldNames := make([]string, 0, len(continuations))
		for name := range continuations {
			switch name {
			case "origin_file":
				fieldNames = append(fieldNames, "`origin_file` (historical origin evidence; not a continuation target)")
			case "line", "column":
				fieldNames = append(fieldNames, "`"+name+"` (navigation only)")
			case "kind":
				fieldNames = append(fieldNames, "`kind` (descriptive only)")
			case "package":
				fieldNames = append(fieldNames, "`"+name+"` (copyable resolved package name)")
			default:
				fieldNames = append(fieldNames, "`"+name+"` (copyable)")
			}
		}
		slices.Sort(fieldNames)
		description += " Successful structured results may include: " + strings.Join(fieldNames, ", ") + "."
	}
	return description
}

func removeConstructResultSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"status":              map[string]any{"type": "string", "enum": []string{"removed", "refused"}},
			"safe_delete":         map[string]any{"type": "boolean"},
			"checks_run":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"refusal":             map[string]any{"type": "string"},
			"diff":                map[string]any{"type": "string"},
			diagnosticStatusField: map[string]any{"type": "string"},
			diagnosticDeltaField:  map[string]any{"type": "object", additionalPropertiesKey: true},
		},
		"required":              []string{"status", "safe_delete", "checks_run"},
		additionalPropertiesKey: false,
	}
}

// ValidateSourceFields verifies every parameter continuation references an advertised top-level output field.
func ValidateSourceFields(entries []operation.Entry) error {
	for _, entry := range entries {
		if err := operation.ValidateParameterContracts(entry.Params); err != nil {
			return fmt.Errorf("operation %q parameter contract: %w", entry.MCPName, err)
		}
	}
	for _, entry := range entries {
		for _, param := range entry.Params {
			for _, source := range param.SourceFields {
				if err := validateSourceField(entries, entry, param, source); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateSourceField(entries []operation.Entry, entry operation.Entry, param operation.ParameterContract, source string) error {
	tool, fieldPath, ok := strings.Cut(source, ".")
	if !ok || tool == "" || fieldPath == "" {
		return fmt.Errorf("operation %q parameter %q has invalid SourceFields entry %q; want tool.field", entry.MCPName, param.JSONName, source)
	}
	provider, exists := findEntryByMCP(entries, tool)
	if !exists {
		return fmt.Errorf("operation %q parameter %q SourceFields entry %q names unknown output tool", entry.MCPName, param.JSONName, source)
	}
	var continuation *operation.OutputContinuation
	if provider.Output != nil {
		for index := range provider.Output.Continuations {
			candidate := &provider.Output.Continuations[index]
			if candidate.Field == fieldPath || candidate.Path == fieldPath {
				continuation = candidate
				break
			}
		}
	}
	if param.Role == "" {
		if continuation != nil {
			return nil
		}
		legacy, err := hasLegacySourceField(provider, fieldPath)
		if err != nil {
			return fmt.Errorf("operation %q parameter %q legacy source %q: %w", entry.MCPName, param.JSONName, source, err)
		}
		if !legacy {
			return fmt.Errorf("operation %q parameter %q SourceFields entry %q names a field absent from %s outputSchema", entry.MCPName, param.JSONName, source, tool)
		}
		return nil
	}
	if param.Role != operation.RoleTarget && param.Role != operation.RoleSelector {
		return fmt.Errorf("operation %q parameter %q has invalid live continuation role %q", entry.MCPName, param.JSONName, param.Role)
	}
	if continuation == nil {
		legacy, err := hasLegacySourceField(provider, fieldPath)
		if err != nil {
			return fmt.Errorf("operation %q parameter %q legacy source %q: %w", entry.MCPName, param.JSONName, source, err)
		}
		if legacy {
			return nil
		}
	}
	if continuation == nil || continuation.Role != param.Role || !slices.Contains(continuation.Consumers, entry.MCPName) {
		return fmt.Errorf("operation %q parameter %q SourceFields entry %q has no compatible typed continuation", entry.MCPName, param.JSONName, source)
	}
	leaf := strings.TrimSuffix(fieldPath[strings.LastIndex(fieldPath, ".")+1:], "[]")
	if leaf != param.JSONName {
		return fmt.Errorf("operation %q parameter %q cannot copy continuation field %q; live field names must match", entry.MCPName, param.JSONName, leaf)
	}
	targetSchema, err := operation.ParameterValueSchema(param)
	if err != nil {
		return fmt.Errorf("operation %q parameter %q schema: %w", entry.MCPName, param.JSONName, err)
	}
	if !operation.SchemaAssignable(continuation.Schema, targetSchema) {
		return fmt.Errorf("operation %q parameter %q schema is incompatible with provider %q field %q", entry.MCPName, param.JSONName, tool, fieldPath)
	}
	return nil
}

func findEntryByMCP(entries []operation.Entry, name string) (operation.Entry, bool) {
	for _, entry := range entries {
		if entry.MCPName == name {
			return entry, true
		}
	}
	return operation.Entry{}, false
}

func hasLegacySourceField(provider operation.Entry, fieldPath string) (bool, error) {
	if strings.Contains(fieldPath, ".") {
		return false, nil
	}
	output, err := toolSchema(provider)
	if err != nil {
		return false, fmt.Errorf("build output schema for %q: %w", provider.MCPName, err)
	}
	outputSchema, ok := output[toolFieldOutputSchema].(map[string]any)
	if !ok {
		return false, fmt.Errorf("tool %q output schema is not an object", provider.MCPName)
	}
	properties, ok := outputSchema[schemaPropertiesKey].(map[string]any)
	if !ok {
		return false, fmt.Errorf("tool %q output schema has no object properties", provider.MCPName)
	}
	_, exists := properties[fieldPath]
	return exists, nil
}

func continuationOutputFieldSchemas(toolName string) map[string]any {
	fields := map[string]any{}
	switch toolName {
	case "semantic_lookup":
		fields["symbol"] = map[string]any{"type": "string"}
		fields["file"] = map[string]any{"type": "string"}
		fields["kind"] = map[string]any{"type": "string", schemaDescriptionKey: "Descriptive symbol category; its vocabulary differs from mutation kind inputs."}
		fields["line"] = map[string]any{"type": "integer"}
		fields["column"] = map[string]any{"type": "integer"}
	case "semantic_rename", "semantic_replace_body":
		fields["symbol"] = map[string]any{"type": "string"}
		fields["file"] = map[string]any{"type": "string"}
	case "semantic_extract_function", semanticExtractVariableTool:
		fields["symbol"] = map[string]any{"type": "string"}
		fields["file"] = map[string]any{"type": "string"}
		fields["in_function"] = map[string]any{"type": "string"}

	case "semantic_replace_construct", "semantic_insert_case", "semantic_inline_variable":
		fields["file"] = map[string]any{"type": "string"}
		fields["in_function"] = map[string]any{"type": "string"}

		if toolName == "semantic_insert_case" {
			fields["discriminator"] = map[string]any{"type": "string"}
		}
	case "semantic_remove_construct":
		fields["file"] = map[string]any{"type": "string"}
	case semanticMoveConstructTool:
		fields["file"] = map[string]any{"type": "string", schemaDescriptionKey: "Live destination file after a successful move; the source file on an error is only the attempted target."}
		fields["origin_file"] = map[string]any{"type": "string", schemaDescriptionKey: "Historical source location before the move; evidence only, not a continuation target."}
		fields["symbol"] = map[string]any{"type": "string"}
	case "semantic_insert_function", "semantic_insert_type", "semantic_insert_decl",
		"semantic_insert_declaration", semanticInsertConstructTool, "semantic_replace_decl":
		fields["symbol"] = map[string]any{"type": "string"}
		fields["file"] = map[string]any{"type": "string"}
		fields["line"] = map[string]any{"type": "integer", schemaDescriptionKey: "Navigation hint; not an operation parameter."}
		if toolName == semanticInsertConstructTool {
			fields["in_function"] = map[string]any{"type": "string"}
			fields["discriminator"] = map[string]any{"type": "string"}
		}
	case "semantic_organize_imports":
		fields["file"] = map[string]any{"type": "string"}
	case "semantic_scaffold_file":
		fields["file"] = map[string]any{"type": "string"}
		fields["package"] = map[string]any{"type": "string"}
	}
	return fields
}

func moveConstructContinuationFields(result any, root string) map[string]any {
	fields := map[string]any{}
	edit, ok := result.(operation.FileEditRes)
	if !ok {
		return fields
	}
	if file := continuationPath(root, edit.DestinationFile); file != "" {
		fields["file"] = file
	}
	if file := continuationPath(root, edit.File); file != "" {
		fields["origin_file"] = file
	}
	if edit.Symbol != "" {
		fields["symbol"] = edit.Symbol
	}
	return fields
}

func structuredContinuationFields(toolName string, result any, root string, arguments map[string]any) map[string]any {
	fields := map[string]any{}
	switch toolName {
	case "semantic_lookup":
		if lookup, ok := result.(*backend.LookupResult); ok {
			addLookupContinuation(fields, lookup, root)
		}
	case "semantic_rename":
		addRenameContinuation(fields, result, root, arguments)
	case "semantic_replace_body", "semantic_replace_decl", "semantic_insert_function",
		"semantic_insert_type", "semantic_insert_decl", "semantic_insert_declaration",
		"semantic_organize_imports", "semantic_extract_function", semanticExtractVariableTool:
		addFileEditContinuationFields(fields, toolName, result, root)
	case "semantic_remove_construct":
		addRemoveContinuationFields(fields, result, root)
	case "semantic_replace_construct", "semantic_insert_case", semanticInsertConstructTool, "semantic_inline_variable":
		addConstructContinuation(fields, toolName, result, root)
	case semanticMoveConstructTool:
		maps.Copy(fields, moveConstructContinuationFields(result, root))
	case "semantic_scaffold_file":
		addScaffoldContinuation(fields, result, root)
	}
	return fields
}

func addLookupContinuation(fields map[string]any, lookup *backend.LookupResult, root string) {
	if lookup == nil || lookup.Ambiguous {
		return
	}
	if lookup.Symbol != "" {
		fields["symbol"] = lookup.Symbol
	}
	if file := continuationPath(root, lookup.File); file != "" {
		fields["file"] = file
	}
	if lookup.Kind != "" {
		fields["kind"] = lookup.Kind
	}
	if lookup.Line > 0 {
		fields["line"] = lookup.Line
	}
	if lookup.Column > 0 {
		fields["column"] = lookup.Column
	}
}

func addRenameContinuation(fields map[string]any, result any, root string, arguments map[string]any) {
	rename, ok := result.(*backend.RenameResult)
	if !ok || rename == nil || rename.Lookup == nil {
		return
	}
	if symbol := renamedQualifiedSymbol(rename.Lookup.Symbol, rawString(arguments, "to")); symbol != "" {
		fields["symbol"] = symbol
	}
	if file := continuationPath(root, rename.Lookup.File); file != "" {
		fields["file"] = file
	}
}

func addConstructContinuation(fields map[string]any, toolName string, result any, root string) {
	edit, ok := result.(operation.FileEditRes)
	if !ok {
		return
	}
	if toolName == semanticInsertConstructTool && edit.Symbol != "" {
		fields["symbol"] = edit.Symbol
	}
	if file := continuationPath(root, edit.File); file != "" {
		fields["file"] = file
	}
	if toolName == semanticInsertConstructTool && edit.Line > 0 {
		fields["line"] = edit.Line
	}
	if edit.InFunction != "" {
		fields["in_function"] = edit.InFunction
	}
	if (toolName == "semantic_insert_case" || toolName == semanticInsertConstructTool) && edit.Discriminator != "" {
		fields["discriminator"] = edit.Discriminator
	}
}

func addScaffoldContinuation(fields map[string]any, result any, root string) {
	file, ok := result.(operation.ScaffoldFileRes)
	if !ok {
		return
	}
	if resolved := continuationPath(root, file.File); resolved != "" {
		fields["file"] = resolved
	}
	if file.Package != "" {
		fields["package"] = file.Package
	}
}

func addFileEditContinuationFields(fields map[string]any, toolName string, result any, root string) {
	edit, ok := result.(operation.FileEditRes)
	if !ok {
		return
	}
	if edit.Symbol != "" {
		fields["symbol"] = edit.Symbol
	}
	if file := continuationPath(root, edit.File); file != "" && (toolName != "semantic_organize_imports" || regularFile(edit.File)) {
		fields["file"] = file
	}
	if edit.Line > 0 {
		fields["line"] = edit.Line
	}
	if (toolName == "semantic_extract_function" || toolName == semanticExtractVariableTool) && edit.InFunction != "" {
		fields["in_function"] = edit.InFunction
	}

}

func addRemoveContinuationFields(fields map[string]any, result any, root string) {
	edit, ok := result.(operation.RemoveRes)
	if !ok {
		return
	}
	if file := continuationPath(root, edit.File); file != "" {
		fields["file"] = file
	}

}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func rawString(arguments map[string]any, key string) string {
	value, _ := arguments[key].(string)
	return backend.NormalizeRenameInput(value)
}

func renamedQualifiedSymbol(previous, next string) string {
	if previous == "" || next == "" {
		return ""
	}
	separator := strings.LastIndex(previous, "::")
	separatorWidth := 2
	if dot := strings.LastIndex(previous, "."); dot > separator {
		separator = dot
		separatorWidth = 1
	}
	if separator < 0 {
		return next
	}
	return previous[:separator+separatorWidth] + next
}

func continuationPath(root, source string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	root = pathutil.CanonicalWorkspaceRoot(root)
	var absolute string
	if filepath.IsAbs(source) {
		absolute = filepath.Clean(source)
	} else {
		absolute = filepath.Join(root, filepath.Clean(source))
	}
	if !pathutil.PathWithin(root, absolute) {
		return ""
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == "." {
		return ""
	}
	return filepath.ToSlash(relative)
}

func standardOutputSchema(resultSchema map[string]any) map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			applicationStateField:         map[string]any{"type": "string"},
			"result":                      resultSchema,
			"workspace_root":              map[string]any{"type": "string", schemaDescriptionKey: "Canonical root used for this request; pass it to a later semantic tool call to keep using the same workspace."},
			"presentation_error":          map[string]any{"type": "string"},
			"native_metadata_unavailable": map[string]any{"type": "boolean"},
			diagnosticStatusField:         map[string]any{"type": "string"},
			diagnosticDeltaField:          diagnosticDeltaSchema(),
		},
		"required": []string{"result"},
	}
}

func metricsOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"schema_version": map[string]any{"type": "integer"},
			"total_ms":       map[string]any{"type": "integer"},
			"phases": map[string]any{
				"type":                  "object",
				additionalPropertiesKey: map[string]any{"type": "object", schemaPropertiesKey: map[string]any{"count": map[string]any{"type": "integer"}, "duration_ms": map[string]any{"type": "integer"}}},
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

func batchOutputSchemaForEntries(entries []operation.Entry) (map[string]any, error) {
	items, err := batchResultSchemaForEntries(entries)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"status":             map[string]any{"type": "string", "enum": []string{"ok", "error"}},
			"workspace_root":     map[string]any{"type": "string", schemaDescriptionKey: "Canonical root used by every child in this batch."},
			"results":            map[string]any{"type": "array", "items": items},
			diagnosticDeltaField: diagnosticDeltaSchema(),
			finalDiffToolName:    map[string]any{"type": "string"},
		},
		"required": []string{"status", "results", finalDiffToolName},
	}, nil
}

func batchResultSchemaForEntries(entries []operation.Entry) (map[string]any, error) {
	branches := make([]any, 0, len(entries)+4)
	baseProperties := func(name string) map[string]any {
		return map[string]any{
			"tool":                map[string]any{"const": name},
			"file":                map[string]any{"type": "string", schemaDescriptionKey: "Requested source path for this operation; on errors it identifies the attempted target, not a successful write."},
			"destination_file":    map[string]any{"type": "string", schemaDescriptionKey: "Requested destination path for this operation; on errors it identifies the attempted target, not a successful write."},
			"symbol":              map[string]any{"type": "string"},
			"package":             map[string]any{"type": "string", schemaDescriptionKey: "Package result, not a symbol target."},
			"in_function":         map[string]any{"type": "string"},
			"discriminator":       map[string]any{"type": "string"},
			diagnosticStatusField: map[string]any{"type": "string"},
			"status":              map[string]any{"type": "string", "enum": []string{"ok", "error"}},
			applicationStateField: map[string]any{"type": "string"},
			"projection_error":    map[string]any{"type": "string"},
			"diff":                map[string]any{"type": "string"},
			"error":               map[string]any{"type": "string"},
		}
	}
	appendBranch := func(properties map[string]any) {
		branches = append(branches, map[string]any{
			"type":                  "object",
			schemaPropertiesKey:     properties,
			"required":              []string{"tool", "status"},
			additionalPropertiesKey: false,
		})
	}
	for _, entry := range entries {
		properties := baseProperties(entry.MCPName)
		if entry.MCPName == semanticMoveConstructTool {
			properties["file"] = map[string]any{"type": "string", schemaDescriptionKey: "Captured destination path on success; attempted source path on error. In a batch this may describe an intermediate path rather than the final workspace target."}
			properties["origin_file"] = map[string]any{"type": "string", schemaDescriptionKey: "Historical source location before a successful move; evidence only, not a continuation target."}
			properties["destination_file"] = map[string]any{"type": "string", schemaDescriptionKey: "Captured destination path for this move; in a batch it may describe an intermediate result rather than the final workspace target."}
		}
		resultSchemas := make([]any, 0, 2)
		if entry.ProjectOutput != nil {
			if entry.Output == nil || entry.Output.ResultSchema == nil {
				return nil, fmt.Errorf("batch result for %q has a projector without a result schema", entry.MCPName)
			}
			resultSchema, err := entry.Output.ResultSchema.JSONSchema()
			if err != nil {
				return nil, fmt.Errorf("batch result schema for %q: %w", entry.MCPName, err)
			}
			resultSchemas = append(resultSchemas, resultSchema)
		}
		if entry.ProjectErrorOutput != nil {
			if entry.ErrorOutput == nil || entry.ErrorOutput.ResultSchema == nil {
				return nil, fmt.Errorf("batch error result for %q has a projector without a result schema", entry.MCPName)
			}
			resultSchema, err := entry.ErrorOutput.ResultSchema.JSONSchema()
			if err != nil {
				return nil, fmt.Errorf("batch error result schema for %q: %w", entry.MCPName, err)
			}
			resultSchemas = append(resultSchemas, resultSchema)
		}
		if len(resultSchemas) == 1 {
			properties["result"] = resultSchemas[0]
		} else if len(resultSchemas) > 1 {
			properties["result"] = map[string]any{"anyOf": resultSchemas}
		}
		appendBranch(properties)
	}
	for _, name := range []string{semanticVerifyToolName, "post_process", "session_sync", finalDiffToolName} {
		properties := baseProperties(name)
		appendBranch(properties)
	}
	return map[string]any{"oneOf": branches}, nil
}

func diagnosticDeltaSchema() map[string]any {
	return map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"net_delta":      map[string]any{"type": "integer"},
			introducedField:  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			resolvedField:    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			suggestionsField: map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"net_delta", introducedField, resolvedField},
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

func operationInputSchema(entry operation.Entry) (map[string]any, error) {
	schema, err := operationInputSchemaForParams(entry.Params)
	if err != nil {
		return nil, err
	}
	addOperationExample(schema, entry.ExampleRaw, entry.Params)
	return schema, nil
}

func workspaceRootParameterSchema() map[string]any {
	return map[string]any{
		"type":        "string",
		"description": "Optional workspace root for this request. Relative roots resolve from the MCP server's configured root; relative file paths resolve from the selected root. Any trust_workspace approval applies only to this canonical root, which is returned for copy-forward.",
	}
}

func addWorkspaceRootInput(schema map[string]any) {
	properties, ok := schema[schemaPropertiesKey].(map[string]any)
	if !ok {
		return
	}
	properties["workspace_root"] = workspaceRootParameterSchema()
}

func operationInputSchemaForBackends(entry operation.Entry, backends []backend.Backend) (map[string]any, error) {
	params := operation.ParametersForBackends(entry, backends)
	for _, param := range params {
		if (param.DynamicEnums != nil || param.LanguageSelector) && len(param.Enums) == 0 {
			return nil, fmt.Errorf("%w: parameter %q", errNoRegisteredParameterChoices, param.JSONName)
		}
	}
	schema, err := operationInputSchemaForParams(params)
	if err != nil {
		return nil, err
	}
	addOperationExample(schema, entry.ExampleRaw, params)
	return schema, nil
}

func addOperationExample(schema map[string]any, example map[string]any, params []operation.ParameterContract) {
	if example == nil {
		return
	}

	supported := make(map[string]operation.ParameterContract, len(params))
	for _, param := range params {
		supported[param.JSONName] = param
	}

	filtered := make(map[string]any, len(example))
	for key, value := range example {
		param, ok := supported[key]
		if !ok {
			continue
		}
		if len(param.Enums) > 0 {
			choice, ok := value.(string)
			if !ok || !slices.Contains(param.Enums, choice) {
				return
			}
		}
		filtered[key] = value
	}
	schema["examples"] = []map[string]any{filtered}
}

func parameterDescription(param operation.ParameterContract) string {
	if len(param.SourceFields) == 0 {
		return param.Description
	}
	fieldTools := make(map[string][]string)
	fieldOrder := make([]string, 0)
	for _, source := range param.SourceFields {
		tool, field, ok := strings.Cut(source, ".")
		if !ok || tool == "" || field == "" {
			continue
		}
		if _, exists := fieldTools[field]; !exists {
			fieldOrder = append(fieldOrder, field)
		}
		if !slices.Contains(fieldTools[field], tool) {
			fieldTools[field] = append(fieldTools[field], tool)
		}
	}
	var description strings.Builder
	description.WriteString(param.Description)
	for _, field := range fieldOrder {
		tools := fieldTools[field]
		if len(tools) == 0 {
			continue
		}
		if after, ok := strings.CutPrefix(field, "result.continuations[]."); ok {
			name := after
			fmt.Fprintf(&description, " Receipt %s from %s.", name, joinToolNames(tools))
			continue
		}
		fmt.Fprintf(&description, " Copy from the %s field of a preceding %s result.", field, joinToolNames(tools))
	}
	return description.String()
}

func joinToolNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " or " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", or " + names[len(names)-1]
	}
}

func operationInputSchemaForParams(params []operation.ParameterContract) (map[string]any, error) {
	properties := make(map[string]any, len(params))
	required := make([]string, 0, len(params))
	for _, param := range params {
		property := map[string]any{toolFieldDescription: parameterDescription(param)}
		if param.Schema != nil {
			declared, err := param.Schema.JSONSchema()
			if err != nil {
				return nil, fmt.Errorf("parameter %q schema: %w", param.JSONName, err)
			}
			maps.Copy(property, declared)
		}
		if param.Schema == nil {
			switch param.Type {
			case operation.ParamBoolean:
				property["type"] = "boolean"
			case operation.ParamInteger:
				property["type"] = "integer"
			case operation.ParamStringSlice:
				property["type"] = "array"
				property["items"] = map[string]any{"type": "string"}
			default:
				property["type"] = "string"
			}
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
	schema := map[string]any{"type": "object", schemaPropertiesKey: properties, additionalPropertiesKey: false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema, nil
}

func batchToolSchemaForLanguages(entries []operation.Entry, backends []backend.Backend) (map[string]any, error) {
	const batchDescription = "Use this tool instead of sending several independent semantic edits when they must run in order. Each edit must have exactly {\"tool\":\"semantic_tool_name\",\"params\":{...}}; put that operation arguments inside params, never beside tool. Example: {\"tool\":\"semantic_replace_body\",\"params\":{\"file\":\"main.go\",\"symbol\":\"Run\",\"body\":\"return nil\"}}. Validation errors identify the zero-based edit index and params problem. On failure, earlier successful edits remain applied and are not rolled back. Child outcomes and receipts are captured intermediate states, not final continuation targets; inspect final targets after the batch to resolve continuation fields. Successful batches return final_diff and diagnostic_delta."
	branches := make([]any, 0, len(entries))
	for _, entry := range entries {
		paramsSchema, err := operationInputSchemaForBackends(entry, backends)
		if err != nil {
			return nil, fmt.Errorf("build batch params schema for %q: %w", entry.MCPName, err)
		}
		addWorkspaceRootInput(paramsSchema)
		branches = append(branches, map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"tool":   map[string]any{"const": entry.MCPName},
				"params": paramsSchema,
			},
			"required":              []string{"tool", "params"},
			additionalPropertiesKey: false,
		})
	}
	outputSchema, err := batchOutputSchemaForEntries(entries)
	if err != nil {
		return nil, fmt.Errorf("build batch output schema: %w", err)
	}
	return map[string]any{
		"name":               "semantic_batch",
		toolFieldDescription: batchDescription,
		toolFieldInputSchema: map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"edits":                 map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"oneOf": branches}},
				"auto_organize_imports": map[string]any{"type": "boolean", "default": false},
				"workspace_root":        workspaceRootParameterSchema(),
				"trust_workspace":       map[string]any{"type": "boolean", "description": "Trust configured verification hooks for this batch request and its selected workspace root."},
			},
			"required":              []string{"edits"},
			additionalPropertiesKey: false,
		},
		toolFieldOutputSchema: standardOutputSchema(outputSchema),
	}, nil
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
	if s.registryValidationErr != nil {
		s.sendError(id, -32603, fmt.Sprintf("validate MCP continuation registry: %v", s.registryValidationErr))
		return
	}
	if s.handleSpecialToolCall(ctx, id, params) {
		return
	}
	if !s.ensureInitialized(ctx, id) {
		return
	}
	entry, ok := s.registry.LookupMCP(params.Name)
	if !ok || (s.profile == "mutations-only" && entry.ReadOnly) {
		s.sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}
	timing := newToolRequestTiming(ctx, s.firstSemanticCallMetrics(time.Now()))
	timing.toolName = params.Name
	ctx = timing.Context(ctx)
	raw := map[string]any{}
	finishArguments := telemetry.Start(ctx, telemetry.PhaseArgumentParsing)
	if len(params.Arguments) > 0 && string(params.Arguments) != "null" {
		if err := decodeArgumentObject(params.Arguments, &raw); err != nil {
			finishArguments()
			s.sendToolErrorWithTiming(id, fmt.Sprintf("invalid arguments: %v", err), timing, err)
			return
		}
	}
	requestRoot, rootErr := prepareWorkspaceArguments(s.workDir, raw)
	if rootErr != nil {
		finishArguments()
		s.sendToolErrorWithTiming(id, rootErr.Error(), timing, rootErr)
		return
	}
	requestTrust := requestTrustedRoot(requestRoot, s.workspaceTrust, raw)
	projectTrust := canonicalTrust(requestRoot, requestTrust)
	requestServer := s.requestView(requestRoot, requestTrust)
	observer := requestMutationObserver(requestServer)
	ctx = operation.WithMutationObserver(ctx, observer)
	ctx = godiagnostics.WithSessionManager(ctx, s.diagnosticSession(requestRoot))
	ctx = gocache.WithBaseDir(ctx, s.goBaseDirForRoot(requestRoot))
	finishArguments()
	if err := requestServer.validateLanguageAllowed(entry, raw); err != nil {
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	finishDispatch := telemetry.Start(ctx, telemetry.PhaseDispatch)
	result, dispatchErr := s.registry.Dispatch(operation.CallContext{
		Ctx:              ctx,
		WorkDir:          requestRoot,
		Project:          backend.ProjectContext{RootDir: requestRoot, WorkspaceTrust: projectTrust},
		Registry:         s.registry,
		Service:          s.service,
		AllowedLanguages: s.enabledLanguages,
	}, entry.Key, raw)
	finishDispatch()
	if dispatchErr != nil {
		s.sendOperationToolErrorWithTiming(id, dispatchErr.Error(), timing, entry, dispatchErr)
		return
	}
	s.sendToolDispatchResponse(ctx, id, timing, params.Name, entry, result, requestRoot, raw)
}

func requestMutationObserver(requestServer *Server) func(context.Context) error {
	if requestServer.goEnabled() {
		return godiagnostics.SynchronizeWithContext
	}
	return func(context.Context) error { return nil }
}

func (s *Server) sendToolDispatchResponse(ctx context.Context, id json.RawMessage, timing *toolRequestTiming, toolName string, entry operation.Entry, result any, requestRoot string, raw map[string]any) {
	if verify, ok := result.(operation.VerifyRes); ok && verify.Failure != "" {
		finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
		body, formatErr := entry.Format(result)
		finishResponse()
		if formatErr != nil {
			s.sendToolErrorWithTiming(id, formatErr.Error(), timing, formatErr)
			return
		}
		structured := timing.structuredContentWithResult(verify)
		presented, presentationErr := criticalTextWithNative(structured, body, verify)
		if presentationErr != nil {
			structured["presentation_error"] = presentationErr.Error()
			presented = fmt.Sprintf("Critical result projection unavailable: %v\n\n%s", presentationErr, body)
		}
		s.recordToolMetrics(timing)
		s.sendResult(id, map[string]any{
			"content":            []map[string]any{{"type": "text", "text": presented}},
			"isError":            true,
			structuredContentKey: structured,
		})
		return
	}
	fields := structuredContinuationFields(toolName, result, requestRoot, raw)
	if entry.ProjectOutput != nil {
		projected, projectErr := entry.ProjectOutput(result)
		if projectErr != nil {
			s.sendToolErrorWithTiming(id, projectErr.Error(), timing, projectErr)
			return
		}
		fields = map[string]any{"result": projected}
		continuationFields, continuationErr := operation.ProjectedContinuationFields(projected, entry.Output)
		if continuationErr != nil {
			s.sendToolErrorWithTiming(id, continuationErr.Error(), timing, continuationErr)
			return
		}
		maps.Copy(fields, continuationFields)
	}
	fields["workspace_root"] = requestRoot
	finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
	text, formatErr := entry.Format(result)
	finishResponse()
	if formatErr != nil {
		s.sendToolErrorWithTiming(id, formatErr.Error(), timing, formatErr)
		return
	}
	s.sendToolSuccessWithNative(id, text, timing, result, fields)
}

func (s *Server) ensureInitialized(ctx context.Context, id json.RawMessage) bool {
	s.sessionMu.Lock()
	uninitialized := s.initializedAt.IsZero()
	s.sessionMu.Unlock()
	if !uninitialized {
		return true
	}
	if _, err := s.Initialize(ctx, nil); err != nil {
		s.sendError(id, -32603, fmt.Sprintf("initialize server: %v", err))
		return false
	}
	return true
}

func (s *Server) handleSpecialToolCall(ctx context.Context, id json.RawMessage, params toolCallParams) bool {
	switch params.Name {
	case "semantic_reload":
		s.handleReload(id)
	case "semantic_metrics":
		s.handleMetricsCall(id, params.Arguments)
	case "report_feedback":
		s.handleFeedbackReport(ctx, id, params.Arguments)
	case "semantic_batch":
		s.handleBatch(ctx, id, params.Arguments)
	default:
		return false
	}
	return true
}

func (s *Server) validateLanguageAllowed(entry operation.Entry, raw map[string]any) error {
	if len(entry.Languages) == 1 && slices.Contains(entry.Languages, backend.LanguageAuto) {
		return nil
	}
	active, err := s.activeLanguages()
	if err != nil {
		return err
	}
	selected := backend.LanguageID("")
	if value, ok := raw["language"].(string); ok && value != "" && value != string(backend.LanguageAuto) {
		selected = backend.LanguageID(value)
	}
	if selected == "" {
		if file, ok := raw["file"].(string); ok && file != "" {
			selected = backend.LanguageIDFromFile(file)
		}
	}
	if selected == "" {
		if path, ok := raw["path"].(string); ok && path != "" {
			selected, _ = existingSourceLanguage(s.workDir, path)
		}
	}
	if selected != "" {
		if s.enabledLanguages != nil && !slices.Contains(s.enabledLanguages, selected) {
			return fmt.Errorf("language %q is disabled by --enabled-languages", selected)
		}
		if selected == backend.LanguageHaskell {
			if standalone, _ := raw["standalone_haskell"].(bool); !standalone {
				return fmt.Errorf("haskell requires standalone_haskell=true")
			}
		} else if len(active) > 0 && !slices.Contains(active, selected) {
			return fmt.Errorf("language %q is not detected in the active workspace", selected)
		}
		if !slices.Contains(entry.Languages, selected) {
			return fmt.Errorf("operation %q has no handler for language %q", entry.Key, selected)
		}
		return nil
	}
	if entrySupportsAny(entry, active) || len(active) == 0 && len(s.enabledLanguages) == 0 {
		return nil
	}
	return fmt.Errorf("operation %q is unavailable for the active workspace languages", entry.Key)
}

func existingSourceLanguage(root, path string) (backend.LanguageID, bool) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	language := backend.LanguageIDFromFile(path)
	return language, language != ""
}

func (s *Server) handleReload(id json.RawMessage) {
	if !s.liveReload {
		s.sendError(id, -32601, "Unknown tool: semantic_reload")
		return
	}
	if err := s.closeDiagnosticSessions(); err != nil {
		s.sendToolError(id, fmt.Sprintf("close diagnostics before reload: %v", err), err)
		return
	}
	s.sendToolSuccessWithStructuredContent(id, `{"status":"reloading"}`, map[string]any{"result": map[string]any{"status": "reloading"}})
	if err := execReload(); err != nil {
		s.sendToolError(id, fmt.Sprintf("reload error: %v", err), err)
	}
}

func (s *Server) sendToolSuccessWithStructuredContent(id json.RawMessage, text string, structuredContent map[string]any) {
	presented, err := criticalText(structuredContent, text)
	if err != nil {
		structuredContent["presentation_error"] = err.Error()
		presented = criticalTextFallback(structuredContent, text, err)
	}
	s.sendResult(id, map[string]any{
		"content":            []map[string]any{{"type": "text", "text": presented}},
		"isError":            false,
		structuredContentKey: structuredContent,
	})
}

func (s *Server) sendToolSuccessWithTimingAndResult(id json.RawMessage, text string, result any, timing *toolRequestTiming) {
	s.recordToolMetrics(timing)
	structured := timing.structuredContentWithResult(result)
	s.sendResult(id, map[string]any{
		"content":            []map[string]any{{"type": "text", "text": text}},
		"isError":            false,
		structuredContentKey: structured,
	})
}

func (s *Server) sendToolError(id json.RawMessage, text string, errs ...error) {
	structured := map[string]any{"error": text, applicationStateField: "unknown"}
	if len(errs) > 0 && errs[0] != nil {
		if refusal, ok := errors.AsType[*operation.SafeDeleteRefusalError](errs[0]); ok {
			structured = safeDeleteRefusalContent(refusal)
			structured[applicationStateField] = "not_applied"
		}
		var mavenErr *maven.Error
		if errors.As(errs[0], &mavenErr) && mavenErr.MavenResult() != nil {
			structured = map[string]any{"error": text, "result": mavenErr.MavenResult(), applicationStateField: "unknown"}
		}
		if loc := s.extractLocation(errs[0]); loc != nil {
			structured["location"] = loc
		}
	}
	presented, err := criticalText(structured, text)
	if err != nil {
		structured["presentation_error"] = err.Error()
		presented = criticalTextFallback(structured, text, err)
	}
	response := map[string]any{
		"content": []map[string]any{{"type": "text", "text": presented}},
		"isError": true, structuredContentKey: structured,
	}
	if location, ok := structured["location"]; ok {
		response["location"] = location
	}
	s.sendResult(id, response)
}

func safeDeleteRefusalContent(refusal *operation.SafeDeleteRefusalError) map[string]any {
	result := map[string]any{
		"result": map[string]any{
			"status": "refused", "safe_delete": true,
			"checks_run": refusal.ChecksRun, "refusal": refusal.Reason,
		},
	}
	for key, value := range map[string]string{
		"file": refusal.File, "symbol": refusal.Symbol, "in_function": refusal.InFunction,
		"discriminator": refusal.Discriminator, "target_case": refusal.TargetCase,
		"construct_path": refusal.ConstructPath,
	} {
		if value != "" {
			result[key] = value
		}
	}
	return result
}

func (s *Server) sendToolErrorWithTiming(id json.RawMessage, text string, timing *toolRequestTiming, errs ...error) {
	s.recordToolMetrics(timing)
	structured := map[string]any{"error": text, applicationStateField: "unknown"}
	if len(errs) > 0 && errs[0] != nil {
		if refusal, ok := errors.AsType[*operation.SafeDeleteRefusalError](errs[0]); ok {
			structured = safeDeleteRefusalContent(refusal)
			structured[applicationStateField] = "not_applied"
		}
		var mavenErr *maven.Error
		if errors.As(errs[0], &mavenErr) && mavenErr.MavenResult() != nil {
			structured = map[string]any{
				"error": text, "result": mavenErr.MavenResult(),
				applicationStateField: "unknown",
			}
		}
		if loc := s.extractLocation(errs[0]); loc != nil {
			structured["location"] = loc
		}
	}
	presented, err := criticalText(structured, text)
	if err != nil {
		structured["presentation_error"] = err.Error()
		presented = criticalTextFallback(structured, text, err)
	}
	result := map[string]any{
		"content": []map[string]any{{"type": "text", "text": presented}},
		"isError": true, structuredContentKey: structured,
	}
	if location, ok := structured["location"]; ok {
		result["location"] = location
	}
	s.sendResult(id, result)
}

func (s *Server) sendOperationToolErrorWithTiming(id json.RawMessage, text string, timing *toolRequestTiming, entry operation.Entry, operationErr error) {
	if entry.ProjectErrorOutput == nil {
		s.sendToolErrorWithTiming(id, text, timing, operationErr)
		return
	}
	projected, known, projectErr := entry.ProjectErrorOutput(operationErr)
	if projectErr != nil {
		s.recordToolMetrics(timing)
		structured := map[string]any{
			"error":                    text,
			applicationStateField:      "unknown",
			"error_projection_failure": projectErr.Error(),
		}
		location := s.extractLocation(operationErr)
		if location != nil {
			structured["location"] = location
		}
		presented, presentationErr := criticalText(structured, text)
		if presentationErr != nil {
			structured["presentation_error"] = presentationErr.Error()
			presented = criticalTextFallback(structured, text, presentationErr)
		}
		response := map[string]any{
			"content": []map[string]any{{"type": "text", "text": presented}},
			"isError": true, structuredContentKey: structured,
		}
		if location != nil {
			response["location"] = location
		}
		s.sendResult(id, response)
		return
	}
	if !known {
		s.sendToolErrorWithTiming(id, text, timing, operationErr)
		return
	}
	s.recordToolMetrics(timing)
	structured := map[string]any{
		"error":               text,
		"result":              projected,
		applicationStateField: "unknown",
	}
	if _, refusal := errors.AsType[*operation.SafeDeleteRefusalError](operationErr); refusal {
		structured[applicationStateField] = "not_applied"
	} else if result, ok := projected.(map[string]any); ok {
		if state, ok := result[applicationStateField]; ok {
			structured[applicationStateField] = state
		}
	}
	location := s.extractLocation(operationErr)
	if location != nil {
		structured["location"] = location
	}
	presented, presentationErr := criticalText(structured, text)
	if presentationErr != nil {
		structured["presentation_error"] = presentationErr.Error()
		presented = criticalTextFallback(structured, text, presentationErr)
	}
	response := map[string]any{
		"content": []map[string]any{{"type": "text", "text": presented}},
		"isError": true, structuredContentKey: structured,
	}
	if location != nil {
		response["location"] = location
	}
	s.sendResult(id, response)
}

type toolRequestTiming struct {
	metrics  *telemetry.Metrics
	started  time.Time
	session  *StartupMetrics
	toolName string
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
	return map[string]any{}
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
	timing.toolName = "report_feedback"
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
		descriptionKey = "description"
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
		toolFieldInputSchema: map[string]any{
			"type":                  "object",
			schemaPropertiesKey:     properties,
			"required":              []string{"intent", "interface", "command", "parameters", "observed_result", "unexpected_reason", "manual_touchups"},
			additionalPropertiesKey: false,
		},
		toolFieldOutputSchema: standardOutputSchema(map[string]any{
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

type toolMetricsRecord struct {
	Sequence       uint64             `json:"sequence"`
	Tool           string             `json:"tool"`
	CompletedAt    time.Time          `json:"completed_at"`
	Metrics        telemetry.Snapshot `json:"metrics"`
	SessionMetrics *StartupMetrics    `json:"session_metrics,omitempty"`
}

func (s *Server) recordToolMetrics(timing *toolRequestTiming) {
	if timing == nil || timing.toolName == "" || timing.toolName == "semantic_metrics" {
		return
	}
	record := toolMetricsRecord{
		Tool:           timing.toolName,
		CompletedAt:    time.Now(),
		Metrics:        timing.Snapshot(),
		SessionMetrics: timing.session,
	}
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	if s.metricsFile != "" && !s.metricsFileLoaded {
		data, err := os.ReadFile(s.metricsFile)
		switch {
		case err == nil:
			if decodeErr := json.Unmarshal(data, &s.fileMetrics); decodeErr != nil {
				s.metricsPersistErr = fmt.Errorf("decode MCP metrics file %s: %w", s.metricsFile, decodeErr)
				return
			}
			if len(s.fileMetrics) > 0 {
				s.metricSequence = s.fileMetrics[len(s.fileMetrics)-1].Sequence
			}
		case errors.Is(err, os.ErrNotExist):
		default:
			s.metricsPersistErr = fmt.Errorf("read MCP metrics file %s: %w", s.metricsFile, err)
			return
		}
		s.metricsFileLoaded = true
	}
	s.metricSequence++
	record.Sequence = s.metricSequence
	const maxRetainedToolMetrics = 100
	if len(s.recentMetrics) == maxRetainedToolMetrics {
		copy(s.recentMetrics, s.recentMetrics[1:])
		s.recentMetrics = s.recentMetrics[:maxRetainedToolMetrics-1]
	}
	s.recentMetrics = append(s.recentMetrics, record)
	if s.metricsFile == "" {
		return
	}
	records := slices.Clone(s.fileMetrics)
	records = append(records, record)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		s.metricsPersistErr = fmt.Errorf("encode MCP metrics file %s: %w", s.metricsFile, err)
		return
	}
	if err := pipeline.WriteAtomic(s.metricsFile, data); err != nil {
		s.metricsPersistErr = fmt.Errorf("write MCP metrics file %s: %w", s.metricsFile, err)
		return
	}
	s.fileMetrics = records
}

func metricsToolSchema() map[string]any {
	recordSchema := map[string]any{
		"type": "object",
		schemaPropertiesKey: map[string]any{
			"sequence":        map[string]any{"type": "integer"},
			"tool":            map[string]any{"type": "string"},
			"completed_at":    map[string]any{"type": "string", "format": "date-time"},
			"metrics":         metricsOutputSchema(),
			"session_metrics": sessionMetricsOutputSchema(),
		},
		"required": []string{"sequence", "tool", "completed_at", "metrics"},
	}
	return map[string]any{
		"name":               "semantic_metrics",
		toolFieldDescription: "Use this tool instead of expecting timing data in each semantic response; retrieve measurements from recent calls in this server session. Returns the most recent records first, up to the requested limit. This query is not itself recorded.",
		toolFieldInputSchema: map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 10},
			},
			additionalPropertiesKey: false,
		},
		toolFieldOutputSchema: standardOutputSchema(map[string]any{
			"type": "object",
			schemaPropertiesKey: map[string]any{
				"records": map[string]any{"type": "array", "items": recordSchema},
			},
			"required": []string{"records"},
		}),
	}
}

func (s *Server) recentToolMetrics(limit int) []toolMetricsRecord {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	count := min(limit, len(s.recentMetrics))
	records := make([]toolMetricsRecord, 0, count)
	for index := len(s.recentMetrics) - 1; index >= 0 && len(records) < limit; index-- {
		records = append(records, s.recentMetrics[index])
	}
	return records
}

func (s *Server) handleMetricsCall(id json.RawMessage, arguments json.RawMessage) {
	var request struct {
		Limit int `json:"limit"`
	}
	if len(arguments) > 0 && string(arguments) != "null" {
		if err := json.Unmarshal(arguments, &request); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid semantic_metrics arguments: %v", err))
			return
		}
	}
	if request.Limit == 0 {
		request.Limit = 10
	}
	if request.Limit < 1 || request.Limit > 100 {
		s.sendToolError(id, "semantic_metrics limit must be between 1 and 100")
		return
	}
	records := s.recentToolMetrics(request.Limit)
	structured := map[string]any{"result": map[string]any{"records": records}}
	body := fmt.Sprintf("Returned %d timing records, newest first.", len(records))
	text, err := criticalText(structured, body)
	if err != nil {
		structured["presentation_error"] = err.Error()
		text = criticalTextFallback(structured, body, err)
	}
	s.sendResult(id, map[string]any{
		"content":           []map[string]any{{"type": "text", "text": text}},
		"isError":           false,
		"structuredContent": structured,
	})
}

func (s *Server) sendToolSuccessWithNative(id json.RawMessage, text string, timing *toolRequestTiming, native any, fields map[string]any) {
	s.recordToolMetrics(timing)
	structured := timing.structuredContentWithResult(text)
	maps.Copy(structured, fields)
	nativeFields, nativeErr := criticalNativeFields(native)
	if nativeErr == nil {
		for key, value := range nativeFields {
			if _, exists := structured[key]; !exists {
				structured[key] = value
			}
		}
	} else {
		structured["presentation_error"] = nativeErr.Error()
		structured["native_metadata_unavailable"] = true
	}
	presented, err := criticalText(structured, text)
	if err != nil {
		structured["presentation_error"] = err.Error()
		presented = criticalTextFallback(structured, text, err)
	}
	s.sendResult(id, map[string]any{
		"content":            []map[string]any{{"type": "text", "text": presented}},
		"isError":            false,
		structuredContentKey: structured,
	})
}
