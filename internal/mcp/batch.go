// Package mcp executes batches by dispatching only operations registered as batchable.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/projectverify"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/telemetry"
)

// BatchEntry represents a single tool execution request within a batch.
type BatchEntry struct {
	Tool   string          `json:"tool"`
	Params json.RawMessage `json:"params"`
}

// BatchResult records the execution status and output of an individual batch entry.
type BatchResult struct {
	Tool             string                       `json:"tool"`
	File             string                       `json:"file,omitempty"`
	OriginFile       string                       `json:"origin_file,omitempty"`
	Symbol           string                       `json:"symbol,omitempty"`
	Package          string                       `json:"package,omitempty"`
	InFunction       string                       `json:"in_function,omitempty"`
	Discriminator    string                       `json:"discriminator,omitempty"`
	DiagnosticStatus string                       `json:"diagnostic_status,omitempty"`
	ApplicationState string                       `json:"application_state,omitempty"`
	ProjectionError  string                       `json:"projection_error,omitempty"`
	DestinationFile  string                       `json:"destination_file,omitempty"`
	Status           string                       `json:"status"`
	Diff             string                       `json:"diff,omitempty"`
	Error            string                       `json:"error,omitempty"`
	Suggestions      []operation.TargetSuggestion `json:"suggestions,omitempty"`
	Result           any                          `json:"result,omitempty"`
}

// BatchResponse aggregates the outcomes of a semantic batch execution.
type BatchResponse struct {
	Status              string                    `json:"status"`
	WorkspaceRoot       string                    `json:"workspace_root"`
	Results             []BatchResult             `json:"results"`
	DiagnosticDelta     *pipeline.DiagnosticDelta `json:"diagnostic_delta,omitempty"`
	DiagnosticDeltaNote string                    `json:"diagnostic_delta_note,omitempty"`
	Verification        *operation.VerifyRes      `json:"verification,omitempty"`
	FinalDiff           string                    `json:"final_diff"`
}

// ExecuteBatch runs an ordered sequence of registered semantic edits, fail-fast on disk.
func (s *Server) ExecuteBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	configuration := s.configurationSnapshot()
	trusted := configuration.workspaceTrust.Trusted && configuration.workspaceTrust.Allows(configuration.workDir)
	return s.executeBatchForConfiguration(ctx, configuration, configuration.workDir, trusted, edits, autoOrganizeImports)
}

// ExecuteBatchForWorkspace binds all child edits and final verification to one request root.
func (s *Server) ExecuteBatchForWorkspace(ctx context.Context, root string, trustWorkspace bool, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	configuration := s.configurationSnapshot()
	return s.executeBatchForConfiguration(ctx, configuration, root, trustWorkspace, edits, autoOrganizeImports)
}

func (s *Server) executeBatchWithContext(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	ctx, release := godiagnostics.EnsureSessionManager(ctx)
	observer := godiagnostics.SynchronizeWithContext
	if !s.goEnabled() {
		observer = func(context.Context) error { return nil }
	}
	ctx = operation.WithMutationObserver(ctx, observer)
	response, resultErr := s.executeBatch(ctx, edits, autoOrganizeImports)
	if s.goEnabled() {
		if err := godiagnostics.SynchronizeWithContext(ctx); err != nil {
			if response == nil {
				return nil, errors.Join(resultErr, err, release())
			}
			response.Status = "error"
			response.DiagnosticDelta = nil
			response.Results = append(response.Results, BatchResult{Tool: "session_sync", Status: "error", Error: err.Error()})
		}
	}
	return response, errors.Join(resultErr, release())
}

// UnmarshalJSON normalizes canonical, flat, and tool-keyed batch entries before validation.
func (entry *BatchEntry) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("batch edit must be a JSON object: %w", err)
	}
	if raw == nil {
		return fmt.Errorf("batch edit must be a JSON object")
	}
	if toolRaw, hasTool := raw["tool"]; hasTool {
		return entry.unmarshalNamedBatchEntry(raw, toolRaw)
	}
	return entry.unmarshalShorthandBatchEntry(raw)
}

func batchParameterError(entry operation.Entry, raw map[string]any) string {
	supportedSet := make(map[string]struct{}, len(entry.Params)*2)
	required := make([]string, 0, len(entry.Params))
	for _, param := range entry.Params {
		if param.JSONName != "" {
			supportedSet[param.JSONName] = struct{}{}
			if param.Required {
				required = append(required, param.JSONName)
			}
		}
		if param.CLIName != "" {
			supportedSet[param.CLIName] = struct{}{}
		}
	}
	supported := make([]string, 0, len(supportedSet))
	for key := range supportedSet {
		supported = append(supported, key)
	}
	slices.Sort(supported)
	slices.Sort(required)

	unknown := make([]string, 0)
	for key := range raw {
		if _, ok := supportedSet[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return ""
	}
	slices.Sort(unknown)
	message := fmt.Sprintf("unknown parameter %q for %q; supported keys are [%s]", unknown[0], entry.MCPName, strings.Join(supported, ", "))
	if suggestion := closestBatchParameter(unknown[0], supported); suggestion != "" {
		message += fmt.Sprintf(". Did you mean %q?", suggestion)
	}
	if len(required) > 0 {
		message += fmt.Sprintf(" Required parameters: [%s]", strings.Join(required, ", "))
	}
	return message
}

func closestBatchParameter(unknown string, supported []string) string {
	best, bestDistance := "", len([]rune(unknown))+1
	for _, candidate := range supported {
		distance := batchEditDistance(unknown, candidate)
		threshold := max(2, len([]rune(unknown))/3)
		if distance <= threshold && distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}

func batchEditDistance(left, right string) int {
	leftRunes, rightRunes := []rune(left), []rune(right)
	previous := make([]int, len(rightRunes)+1)
	for index := range previous {
		previous[index] = index
	}
	for row, leftRune := range leftRunes {
		current := make([]int, len(rightRunes)+1)
		current[0] = row + 1
		for column, rightRune := range rightRunes {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[column+1] = min(current[column]+1, previous[column+1]+1, previous[column]+cost)
		}
		previous = current
	}
	return previous[len(rightRunes)]
}

func (entry *BatchEntry) unmarshalNamedBatchEntry(raw map[string]json.RawMessage, toolRaw json.RawMessage) error {
	var tool string
	if err := json.Unmarshal(toolRaw, &tool); err != nil {
		return fmt.Errorf("batch edit tool must be a string: %w", err)
	}
	if strings.TrimSpace(tool) == "" {
		return fmt.Errorf("batch edit tool must not be empty")
	}
	entry.Tool = tool
	lifted := make(map[string]json.RawMessage, len(raw))
	if paramsRaw, hasParams := raw["params"]; hasParams {
		if len(paramsRaw) == 0 || string(paramsRaw) == "null" {
			return fmt.Errorf("batch edit params must be a JSON object when provided")
		}
		if err := json.Unmarshal(paramsRaw, &lifted); err != nil || lifted == nil {
			return fmt.Errorf("batch edit params must be a JSON object")
		}
	}
	for key, value := range raw {
		if key == "tool" || key == "params" {
			continue
		}
		if _, exists := lifted[key]; exists {
			return fmt.Errorf("batch edit parameter %q appears in both params and sibling fields", key)
		}
		lifted[key] = value
	}
	params, err := json.Marshal(lifted)
	if err != nil {
		return fmt.Errorf("encode batch edit params: %w", err)
	}
	entry.Params = params
	return nil
}

func (entry *BatchEntry) unmarshalShorthandBatchEntry(raw map[string]json.RawMessage) error {
	if len(raw) != 1 {
		return fmt.Errorf("invalid batch edit: specify a 'tool' key or exactly one 'semantic_*' tool key")
	}
	for tool, params := range raw {
		if !strings.HasPrefix(tool, "semantic_") {
			return fmt.Errorf("invalid batch edit: specify a 'tool' key or exactly one 'semantic_*' tool key")
		}
		if len(params) == 0 || string(params) == "null" {
			return fmt.Errorf("batch edit shorthand for %q requires a JSON argument object", tool)
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(params, &object); err != nil || object == nil {
			return fmt.Errorf("batch edit shorthand for %q requires a JSON argument object", tool)
		}
		entry.Tool = tool
		entry.Params = params
	}
	return nil
}

func validateBatchEntry(entry operation.Entry, params map[string]any) error {
	if _, err := entry.Parse(params); err != nil {
		if message := batchParameterError(entry, params); message != "" {
			return errors.New(message)
		}
		return err
	}
	return nil
}

func batchEntryResult(tool string, result BatchResult, err error) BatchResult {
	result.Tool = tool
	result.Status = "ok"
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Suggestions = operation.SuggestionsFromError(err)
		if enriched, ok := errors.AsType[*operation.SuggestedTargetError](err); ok {
			result.Error = enriched.Cause.Error()
		}
	}
	return result
}

type batchSkipHooksKey struct{}

const batchStatusAppliedWithDiagnostics = "applied_with_diagnostics"

var batchVerificationOptionKeys = []string{
	trustWorkspaceField, "skip_hooks", "jdtls_home", "java_bin", "import_maven", "metals_home", "metals_bin",
	"java_version", "standalone_haskell", "ghc_bin", "hls_bin", "ghc_version", "hls_version",
	"kotlin_bin", "bash_bin", "make_bin",
}

func isGoDiagnostic(diagnostic backend.Diagnostic) bool {
	if diagnostic.Location == nil {
		return strings.Contains(diagnostic.Message, ": [package] ")
	}
	path, err := pathutil.FilePathFromURI(diagnostic.Location.URI)
	return err == nil && filepath.Ext(path) == ".go"
}

func snapshotBatchWorkspace(ctx context.Context, root string) (map[string]string, error) {
	files := make(map[string]string)
	filter, err := sourcebuf.DirectoryFilterFor(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("resolve batch snapshot directories: %w", err)
	}
	workspace, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open workspace root: %w", err)
	}
	defer func() { _ = workspace.Close() }()
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if filter.Skip(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, relativeErr := filepath.Rel(root, path)
		if relativeErr != nil {
			return fmt.Errorf("relativize %s: %w", path, relativeErr)
		}
		data, readErr := workspace.ReadFile(rel)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rel, readErr)
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	return files, err
}

func captureBatchDiff(ctx context.Context, response *BatchResponse, root string, before map[string]string) error {
	diff, err := finalBatchDiff(ctx, root, before)
	if err != nil {
		message := fmt.Errorf("capture final batch diff: %w", err)
		response.Status = "error"
		response.DiagnosticDelta = nil
		response.Results = append(response.Results, BatchResult{Tool: finalDiffToolName, Status: "error", Error: message.Error()})
		return message
	}
	response.FinalDiff = diff
	return nil
}

func finalBatchDiff(ctx context.Context, root string, before map[string]string) (string, error) {
	after, err := snapshotBatchWorkspace(ctx, root)
	if err != nil {
		return "", err
	}
	if err := includeExternalBatchSources(root, before, after); err != nil {
		return "", err
	}
	paths := make(map[string]struct{}, len(before)+len(after))
	for path, content := range before {
		if next, ok := after[path]; !ok || next != content {
			paths[path] = struct{}{}
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			paths[path] = struct{}{}
		}
	}
	ordered := slices.Sorted(maps.Keys(paths))
	var diff strings.Builder
	for _, path := range ordered {
		diff.WriteString(astedit.UnifiedDiff(path, before[path], after[path]))
	}
	return diff.String(), nil
}

func changedBatchPaths(ctx context.Context, root string, before map[string]string) ([]string, error) {
	after, err := snapshotBatchWorkspace(ctx, root)
	if err != nil {
		return nil, err
	}
	if err := includeExternalBatchSources(root, before, after); err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for path, content := range after {
		if original, exists := before[path]; !exists || original != content {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths, nil
}

// decodeArgumentObject preserves exact JSON integer tokens for recursive schema validation.
func decodeArgumentObject(data []byte, target *map[string]any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return fmt.Errorf("read trailing JSON value: %w", err)
	}
	if *target == nil {
		*target = map[string]any{}
	}
	return nil
}

func (s *Server) executeBatchEdit(ctx context.Context, batchEntry BatchEntry) (BatchResult, error) {
	entry, ok := s.registry.LookupMCP(batchEntry.Tool)
	if !ok || !entry.Batchable {
		return BatchResult{}, fmt.Errorf("tool is not batchable: %s", batchEntry.Tool)
	}
	raw := map[string]any{}
	if err := decodeArgumentObject(batchEntry.Params, &raw); err != nil {
		return BatchResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if err := s.validateLanguageAllowed(entry, raw); err != nil {
		return batchMoveReceipt(batchEntry.Tool, raw, s.workDir), err
	}
	result, err := s.registry.Dispatch(operation.CallContext{
		Ctx:               ctx,
		WorkDir:           s.workDir,
		Project:           backend.ProjectContext{RootDir: s.workDir, WorkspaceTrust: s.workspaceTrust},
		Registry:          s.registry,
		Service:           s.service,
		AllowedLanguages:  s.enabledLanguages,
		InBatch:           true,
		DeferImports:      true,
		DeferVerification: true,
	}, entry.Key, raw)
	if err != nil {
		return s.projectBatchDispatchError(batchEntry, entry, raw, err), err
	}
	return projectBatchDispatchSuccess(batchEntry, entry, result, s.workDir)
}

func (s *Server) projectBatchDispatchError(batchEntry BatchEntry, entry operation.Entry, raw map[string]any, dispatchErr error) BatchResult {
	result := batchMoveReceipt(batchEntry.Tool, raw, s.workDir)
	result.ApplicationState = "unknown"
	if entry.ProjectErrorOutput == nil {
		return result
	}
	projected, known, err := entry.ProjectErrorOutput(dispatchErr)
	if err != nil {
		result.ProjectionError = err.Error()
		return result
	}
	if !known {
		return result
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		result.ProjectionError = fmt.Sprintf("encode known error projection: %v", err)
		return result
	}
	result.Result = json.RawMessage(encoded)
	if projection, ok := projected.(map[string]any); ok {
		if state, ok := projection[applicationStateField].(string); ok {
			result.ApplicationState = state
		} else if projection["status"] == "refused" && projection["safe_delete"] == true {
			result.ApplicationState = "not_applied"
		}
	}
	return result
}

func projectBatchDispatchSuccess(batchEntry BatchEntry, entry operation.Entry, dispatched any, workDir string) (BatchResult, error) {
	result := resultForBatch(dispatched, workDir)
	if entry.ProjectOutput == nil {
		return result, nil
	}
	projected, err := entry.ProjectOutput(dispatched)
	if err != nil {
		return BatchResult{}, fmt.Errorf("project batch result for %q: %w", batchEntry.Tool, err)
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		return BatchResult{}, fmt.Errorf("encode batch result for %q: %w", batchEntry.Tool, err)
	}
	result.Result = json.RawMessage(encoded)
	return result, nil
}

func batchMoveReceipt(tool string, raw map[string]any, workDir string) BatchResult {
	if tool != "semantic_move_construct" {
		return BatchResult{}
	}
	file, _ := raw["file"].(string)
	destination, _ := raw["destination_file"].(string)
	if destination == "" {
		destination = file
	}
	symbol, _ := raw["symbol"].(string)
	return BatchResult{
		File:            continuationPath(workDir, file),
		DestinationFile: continuationPath(workDir, destination),
		Symbol:          symbol,
	}
}

func resultForBatch(result any, workDir string) BatchResult {
	switch typed := result.(type) {
	case operation.FileEditRes:
		file := continuationPath(workDir, typed.File)
		if file == "" {
			file = continuationPath(workDir, typed.Display)
		}
		destination := continuationPath(workDir, typed.DestinationFile)
		if destination == "" {
			destination = continuationPath(workDir, typed.DestinationDisplay)
		}
		if destination != "" {
			return BatchResult{File: destination, OriginFile: file, DestinationFile: destination, Symbol: typed.Symbol, InFunction: typed.InFunction, Discriminator: typed.Discriminator, DiagnosticStatus: typed.DiagnosticStatus, Diff: typed.Diff}
		}
		return BatchResult{File: file, Symbol: typed.Symbol, InFunction: typed.InFunction, Discriminator: typed.Discriminator, DiagnosticStatus: typed.DiagnosticStatus, Diff: typed.Diff}
	case operation.ScaffoldFileRes:
		file := continuationPath(workDir, typed.File)
		if file == "" {
			file = continuationPath(workDir, typed.Display)
		}
		return BatchResult{File: file, Package: typed.Package}
	case operation.RemoveRes:
		return BatchResult{File: continuationPath(workDir, typed.File)}
	case operation.BuildDependencyRes:
		return BatchResult{File: continuationPath(workDir, typed.GoMod)}
	default:
		return BatchResult{}
	}
}

func parseBatchWorkspaceRequest(configuredRoot string, rootJSON, trustJSON json.RawMessage) (string, bool, error) {
	var requestedRoot string
	if len(rootJSON) > 0 {
		if strings.TrimSpace(string(rootJSON)) == "null" {
			return "", false, fmt.Errorf("workspace_root must be a string")
		}
		if err := json.Unmarshal(rootJSON, &requestedRoot); err != nil {
			return "", false, fmt.Errorf("workspace_root must be a string: %w", err)
		}
	}
	trusted := false
	if len(trustJSON) > 0 {
		if strings.TrimSpace(string(trustJSON)) == "null" {
			return "", false, fmt.Errorf("trust_workspace must be a boolean")
		}
		if err := json.Unmarshal(trustJSON, &trusted); err != nil {
			return "", false, fmt.Errorf("trust_workspace must be a boolean: %w", err)
		}
	}
	root, err := ResolveRequestRoot(configuredRoot, requestedRoot)
	return root, trusted, err
}

func prepareBatchEntries(configuredRoot, batchRoot string, edits []BatchEntry) ([]BatchEntry, error) {
	prepared := append([]BatchEntry(nil), edits...)
	for index := range prepared {
		entry := &prepared[index]
		if len(entry.Params) == 0 || strings.TrimSpace(string(entry.Params)) == "null" {
			continue
		}
		var child map[string]any
		if err := decodeArgumentObject(entry.Params, &child); err != nil {
			continue
		}
		if child == nil {
			continue
		}
		if childRootValue, present := child["workspace_root"]; present {
			childRoot, ok := childRootValue.(string)
			if !ok {
				return nil, fmt.Errorf("edit %d workspace_root must be a string", index)
			}
			resolvedChildRoot, err := ResolveRequestRoot(configuredRoot, childRoot)
			if err != nil {
				return nil, fmt.Errorf("edit %d: %w", index, err)
			}
			if resolvedChildRoot != batchRoot {
				return nil, fmt.Errorf("edit %d workspace_root %q resolves to %q, differing from batch root %q", index, childRoot, resolvedChildRoot, batchRoot)
			}
			delete(child, "workspace_root")
		}
		if err := ValidateRequestPaths(batchRoot, child); err != nil {
			return nil, fmt.Errorf("edit %d: %w", index, err)
		}
		encoded, err := json.Marshal(child)
		if err != nil {
			return nil, fmt.Errorf("edit %d encode normalized arguments: %w", index, err)
		}
		entry.Params = encoded
	}
	return prepared, nil
}

func (s *Server) handleBatch(ctx context.Context, id json.RawMessage, raw json.RawMessage) {
	s.sessionMu.Lock()
	uninitialized := s.initializedAt.IsZero()
	s.sessionMu.Unlock()
	if uninitialized {
		if _, err := s.Initialize(ctx, nil); err != nil {
			s.sendError(id, -32603, fmt.Sprintf("initialize server: %v", err))
			return
		}
	}
	configuration := s.configurationSnapshot()
	timing := newToolRequestTiming(ctx, s.firstSemanticCallMetrics(time.Now()))
	timing.toolName = "semantic_batch"
	ctx = timing.Context(ctx)
	finishArguments := telemetry.Start(ctx, telemetry.PhaseArgumentParsing)
	request, err := parseBatchToolRequest(configuration.workDir, raw)
	finishArguments()
	if err != nil {
		s.sendToolErrorWithTiming(id, err.Error(), timing, err)
		return
	}
	ctx, requestServer, prepareErr := s.prepareBatchStageContext(ctx, configuration, &request)
	if prepareErr != nil {
		s.sendToolErrorWithTiming(id, prepareErr.Error(), timing, prepareErr)
		return
	}
	finishDispatch := telemetry.Start(ctx, telemetry.PhaseDispatch)
	response, err := requestServer.executeBatchWithContext(ctx, request.request.Edits, request.request.AutoOrganizeImports)
	finishDispatch()
	if err != nil {
		s.sendToolErrorWithTiming(id, fmt.Sprintf("batch execution error: %v", err), timing, err)
		return
	}
	prepareBatchOutput(response)
	finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
	bodyBytes, err := json.MarshalIndent(response, "", "  ")
	finishResponse()
	if err != nil {
		s.sendToolErrorWithTiming(id, fmt.Sprintf("serialization error: %v", err), timing, err)
		return
	}
	structured := timing.structuredContentWithResult(response)
	critical, err := CriticalText(structured, string(bodyBytes))
	if err != nil {
		structured["presentation_error"] = err.Error()
		critical = criticalTextFallback(structured, string(bodyBytes), err)
	} else if response.Status == "error" || len(response.Results) > 0 {
		note := "Batch child results are intermediate and do not provide final continuation targets."
		if response.Verification != nil {
			if response.Verification.Failure != "" {
				note += " Verification failed: " + response.Verification.Failure
			} else {
				note += fmt.Sprintf(" Verification returned %d diagnostics.", len(response.Verification.Diagnostics))
			}
		}
		prefix, remainder, found := strings.Cut(critical, "\n\n")
		if found {
			critical = note + "\n\n" + prefix + "\n\n" + remainder
		}
	}
	s.recordToolMetrics(timing)
	if response.Status != "ok" {
		s.sendResult(id, map[string]any{
			"content": []map[string]any{{"type": "text", "text": critical}},
			"isError": true, structuredContentKey: structured,
		})
		return
	}
	s.sendResult(id, map[string]any{
		"content": []map[string]any{{"type": "text", "text": critical}},
		"isError": false, structuredContentKey: structured,
	})
}

func (s *Server) prepareBatchStageContext(ctx context.Context, configuration serverConfiguration, request *parsedBatchToolRequest) (context.Context, *Server, error) {
	ctx = gocache.WithModuleCacheSeed(
		gocache.WithBaseDir(ctx, selectedGoBaseDir(configuration.workDir, configuration.goBaseDir, request.root)),
		configuration.goModuleCacheSeed,
	)
	if request.request.SkipHooks {
		ctx = context.WithValue(ctx, batchSkipHooksKey{}, true)
	}
	ctx = godiagnostics.WithSessionManager(ctx, s.diagnosticSession(request.root))
	requestServer := requestViewFromConfiguration(configuration, request.root, request.trustWorkspace)
	edits, prepareErr := prepareBatchEntries(configuration.workDir, request.root, request.request.Edits)
	if prepareErr != nil {
		return nil, nil, prepareErr
	}
	request.request.Edits = edits
	if stageErr := s.stages.BlockBatch(ctx, request.root); stageErr != nil {
		return nil, nil, stageErr
	}
	return ctx, requestServer, nil
}

type batchToolPayload struct {
	Edits               []BatchEntry    `json:"edits"`
	AutoOrganizeImports bool            `json:"auto_organize_imports"`
	SkipHooks           bool            `json:"skip_hooks"`
	WorkspaceRoot       json.RawMessage `json:"workspace_root"`
	TrustWorkspace      json.RawMessage `json:"trust_workspace"`
}

type parsedBatchToolRequest struct {
	request        batchToolPayload
	root           string
	trustWorkspace bool
}

func parseBatchToolRequest(configuredRoot string, raw []byte) (parsedBatchToolRequest, error) {
	var payload batchToolPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return parsedBatchToolRequest{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if len(payload.Edits) == 0 {
		return parsedBatchToolRequest{}, fmt.Errorf("semantic_batch requires non-empty 'edits' list")
	}
	root, trustWorkspace, err := parseBatchWorkspaceRequest(configuredRoot, payload.WorkspaceRoot, payload.TrustWorkspace)
	if err != nil {
		return parsedBatchToolRequest{}, err
	}
	return parsedBatchToolRequest{request: payload, root: root, trustWorkspace: trustWorkspace}, nil
}

func (s *Server) executeBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	response := &BatchResponse{Status: "ok", WorkspaceRoot: s.workDir, Results: make([]BatchResult, 0, len(edits))}
	verifyOptions := make(map[string]any)
	if batchSkipsHooks(ctx) {
		verifyOptions["skip_hooks"] = true
	}
	for index, batchEntry := range edits {
		entry, ok := s.registry.LookupMCP(batchEntry.Tool)
		if !ok || !entry.Batchable {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d: tool is not batchable: %s", index, batchEntry.Tool)})
			return response, nil
		}
		raw := map[string]any{}
		if len(batchEntry.Params) == 0 || string(batchEntry.Params) == "null" {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d params for %s must be a JSON object; put operation arguments under params, for example {\"tool\":\"semantic_rename\",\"params\":{\"symbol\":\"Old\",\"to\":\"New\"}}", index, batchEntry.Tool)})
			return response, nil
		}
		if err := decodeArgumentObject(batchEntry.Params, &raw); err != nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Errorf("edit %d params: invalid JSON object: %w", index, err).Error()})
			return response, nil
		}
		if raw == nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d params must be a JSON object; put operation arguments under params, for example {\"tool\":\"semantic_rename\",\"params\":{\"symbol\":\"Old\",\"to\":\"New\"}}", index)})
			return response, nil
		}
		if err := validateBatchEntry(entry, raw); err != nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d params: %v", index, err)})
			return response, nil
		}
		if err := mergeBatchVerificationOptions(verifyOptions, raw); err != nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d params: %v", index, err)})
			return response, nil
		}
		if err := s.validateLanguageAllowed(entry, raw); err != nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d params: %v", index, err)})
			return response, nil
		}
	}
	snapshot, err := s.captureBatchExecutionSnapshot(ctx, edits)
	if err != nil {
		return nil, err
	}
	workspaceBefore, baseline := snapshot.workspace, snapshot.baseline
	ctx = context.WithValue(ctx, batchJournalKey{}, &batchJournal{before: workspaceBefore, root: s.workDir, roots: make(map[string]bool)})
	buffer, err := s.executeBufferedBatchEdits(ctx, edits, response)
	if err != nil {
		return s.finishFailedBatchEdits(ctx, buffer, response, workspaceBefore, err)
	}
	normalizationPaths, err := s.normalizeBufferedBatch(ctx, buffer, workspaceBefore, autoOrganizeImports)
	if err != nil {
		return s.finishFailedBatchNormalization(ctx, buffer, response, workspaceBefore, err)
	}
	if !s.finishBufferedBatch(ctx, buffer, response) {
		_ = captureBatchDiff(ctx, response, s.workDir, workspaceBefore)
		return response, nil
	}
	return s.verifyFinishedBatch(ctx, response, workspaceBefore, baseline, verifyOptions, normalizationPaths)
}

func (s *Server) finishFailedBatchEdits(ctx context.Context, buffer *sourcebuf.Buffer, response *BatchResponse, workspaceBefore map[string]string, cause error) (*BatchResponse, error) {
	response.Status = "error"
	if len(response.Results) == 0 || response.Results[len(response.Results)-1].Status != "error" {
		response.Results = append(response.Results, BatchResult{Tool: "batch_buffers", Status: "error", Error: cause.Error()})
	}
	s.finishBufferedBatch(ctx, buffer, response)
	_ = captureBatchDiff(ctx, response, s.workDir, workspaceBefore)
	return response, nil
}

func (s *Server) finishFailedBatchNormalization(ctx context.Context, buffer *sourcebuf.Buffer, response *BatchResponse, workspaceBefore map[string]string, cause error) (*BatchResponse, error) {
	response.Status = "error"
	response.Results = append(response.Results, BatchResult{Tool: "post_process", Status: "error", Error: cause.Error()})
	s.finishBufferedBatch(ctx, buffer, response)
	_ = captureBatchDiff(ctx, response, s.workDir, workspaceBefore)
	return response, nil
}

type batchExecutionSnapshot struct {
	workspace map[string]string
	baseline  batchVerificationBaseline
}

func (s *Server) captureBatchExecutionSnapshot(ctx context.Context, edits []BatchEntry) (batchExecutionSnapshot, error) {
	workspace, err := snapshotBatchWorkspace(ctx, s.workDir)
	if err != nil {
		return batchExecutionSnapshot{}, fmt.Errorf("snapshot workspace before batch: %w", err)
	}
	plan, err := projectverify.Discover(s.workDir)
	if err != nil {
		return batchExecutionSnapshot{}, fmt.Errorf("discover batch verification plan: %w", err)
	}
	baseline, err := collectBatchVerificationBaseline(ctx, s.workDir, plan, s.enabledLanguages, edits)
	if err != nil {
		return batchExecutionSnapshot{}, err
	}
	return batchExecutionSnapshot{workspace: workspace, baseline: baseline}, nil
}

func (s *Server) verifyFinishedBatch(ctx context.Context, response *BatchResponse, workspaceBefore map[string]string, baseline batchVerificationBaseline, verifyOptions map[string]any, normalizationPaths []string) (*BatchResponse, error) {
	reconcilePublishedGoScope(ctx, s.workDir, workspaceBefore, &baseline)
	verifyEntry, ok := s.registry.LookupMCP(semanticVerifyToolName)
	if !ok {
		response.Status = batchStatusAppliedWithDiagnostics
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: "verification operation is not registered"})
		captureAppliedBatchDiff(ctx, response, s.workDir, workspaceBefore)
		return response, nil
	}
	verifyRaw := map[string]any{"path": ".", "language": "auto", trustWorkspaceField: s.workspaceTrust.Trusted}
	maps.Copy(verifyRaw, verifyOptions)
	verifyRaw[trustWorkspaceField] = s.workspaceTrust.Trusted
	goFormattingPaths := batchGoFormattingPaths(normalizationPaths, s.goEnabled())
	goPackagePatterns := cloneGoPatterns(baseline.GoPatterns)
	verified, err := s.registry.Dispatch(operation.CallContext{
		Ctx: ctx, WorkDir: s.workDir,
		Project:  backend.ProjectContext{RootDir: s.workDir, WorkspaceTrust: s.workspaceTrust},
		Registry: s.registry, Service: s.service, GoFormattingPaths: goFormattingPaths,
		AllowedLanguages: s.enabledLanguages, GoPackagePatterns: goPackagePatterns,
		GoPackageScopeDetail: baseline.ScopeReason,
	}, verifyEntry.Key, verifyRaw)
	if err != nil {
		response.Status = batchStatusAppliedWithDiagnostics
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: err.Error()})
		captureAppliedBatchDiff(ctx, response, s.workDir, workspaceBefore)
		return response, nil //nolint:nilerr // Batch results carry the failure and final diff.
	}
	verifyResult, ok := verified.(operation.VerifyRes)
	if !ok {
		response.Status = batchStatusAppliedWithDiagnostics
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: fmt.Sprintf("unexpected verification result type %T", verified)})
		captureAppliedBatchDiff(ctx, response, s.workDir, workspaceBefore)
		return response, nil
	}
	response.Verification = &verifyResult
	if verifyResult.Failure != "" {
		response.Status = batchStatusAppliedWithDiagnostics
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: verifyResult.Failure})
		captureAppliedBatchDiff(ctx, response, s.workDir, workspaceBefore)
		return response, nil
	}
	recordBatchDiagnosticDelta(response, baseline, verifyResult)
	if response.DiagnosticDelta != nil && hasNetNewGoErrors(baseline, verifyResult) {
		response.Status = batchStatusAppliedWithDiagnostics
	}
	markUnbaselinedGoErrors(response, baseline, verifyResult)
	captureAppliedBatchDiff(ctx, response, s.workDir, workspaceBefore)
	return response, nil
}

func cloneGoPatterns(patterns map[string][]string) map[string][]string {
	if patterns == nil {
		return nil
	}
	cloned := make(map[string][]string, len(patterns))
	for root, selected := range patterns {
		cloned[root] = slices.Clone(selected)
	}
	return cloned
}

func markUnbaselinedGoErrors(response *BatchResponse, baseline batchVerificationBaseline, verification operation.VerifyRes) {
	if baseline.OmittedReason == "" {
		return
	}
	for _, diagnostic := range verification.Diagnostics {
		if isGoDiagnostic(diagnostic) && diagnostic.Severity == 1 {
			response.Status = batchStatusAppliedWithDiagnostics
			response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: "final Go compiler errors are unbaselined because diagnostic delta comparison was omitted"})
			return
		}
	}
}

func captureAppliedBatchDiff(ctx context.Context, response *BatchResponse, root string, before map[string]string) {
	if err := captureBatchDiff(ctx, response, root, before); err != nil {
		response.Status = batchStatusAppliedWithDiagnostics
	}
}

func (s *Server) executeBatchForConfiguration(ctx context.Context, configuration serverConfiguration, root string, trustWorkspace bool, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	canonicalRoot, err := ResolveRequestRoot(configuration.workDir, root)
	if err != nil {
		return nil, err
	}
	preparedEdits, err := prepareBatchEntries(configuration.workDir, canonicalRoot, edits)
	if err != nil {
		return nil, err
	}
	requestServer := requestViewFromConfiguration(configuration, canonicalRoot, trustWorkspace)
	ctx = gocache.WithModuleCacheSeed(
		gocache.WithBaseDir(ctx, selectedGoBaseDir(configuration.workDir, configuration.goBaseDir, canonicalRoot)),
		configuration.goModuleCacheSeed,
	)
	return requestServer.executeBatchWithContext(ctx, preparedEdits, autoOrganizeImports)
}

func batchGoFormattingPaths(paths []string, goEnabled bool) []string {
	formatted := make([]string, 0, len(paths))
	if !goEnabled {
		return formatted
	}
	for _, path := range paths {
		if filepath.Ext(path) == ".go" {
			formatted = append(formatted, path)
		}
	}
	return formatted
}

func mergeBatchVerificationOptions(options, raw map[string]any) error {
	for _, key := range batchVerificationOptionKeys {
		if key == trustWorkspaceField {
			continue
		}
		value, present := raw[key]
		if !present {
			value, present = raw[strings.ReplaceAll(key, "_", "-")]
		}
		if !present {
			continue
		}
		if previous, exists := options[key]; exists && !reflect.DeepEqual(previous, value) {
			return fmt.Errorf("conflicting verification option %q across batch", key)
		}
		options[key] = value
	}
	return nil
}

func includeExternalBatchSources(root string, before, after map[string]string) error {
	for path := range before {
		if path != ".." && !strings.HasPrefix(path, "../") {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(path))
		// #nosec G304 -- only canonical Go outputs recorded by the validated module rename are read.
		data, err := os.ReadFile(target)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read external batch output: %w", err)
		}
		after[path] = string(data)
	}
	return nil
}

func prepareBatchOutput(response *BatchResponse) {
	for index := range response.Results {
		if response.Results[index].Status == "error" && response.Results[index].ApplicationState == "" {
			response.Results[index].ApplicationState = "unknown"
		}
	}
	if response.DiagnosticDelta != nil {
		delta := *response.DiagnosticDelta
		if delta.Introduced == nil {
			delta.Introduced = []string{}
		}
		if delta.Resolved == nil {
			delta.Resolved = []string{}
		}
		response.DiagnosticDelta = &delta
	}
}
