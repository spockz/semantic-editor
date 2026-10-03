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

	"semedit/internal/astedit"
	"semedit/internal/godiagnostics"

	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/projectverify"
	"semedit/internal/telemetry"
)

// BatchEntry represents a single tool execution request within a batch.
type BatchEntry struct {
	Tool   string          `json:"tool"`
	Params json.RawMessage `json:"params"`
}

// BatchResult records the execution status and output of an individual batch entry.
type BatchResult struct {
	Tool             string `json:"tool"`
	File             string `json:"file,omitempty"`
	OriginFile       string `json:"origin_file,omitempty"`
	Symbol           string `json:"symbol,omitempty"`
	Package          string `json:"package,omitempty"`
	InFunction       string `json:"in_function,omitempty"`
	Discriminator    string `json:"discriminator,omitempty"`
	DiagnosticStatus string `json:"diagnostic_status,omitempty"`
	ApplicationState string `json:"application_state,omitempty"`
	ProjectionError  string `json:"projection_error,omitempty"`
	DestinationFile  string `json:"destination_file,omitempty"`
	Status           string `json:"status"`
	Diff             string `json:"diff,omitempty"`
	Error            string `json:"error,omitempty"`
	Result           any    `json:"result,omitempty"`
}

// BatchResponse aggregates the outcomes of a semantic batch execution.
type BatchResponse struct {
	Status              string                    `json:"status"`
	Results             []BatchResult             `json:"results"`
	DiagnosticDelta     *pipeline.DiagnosticDelta `json:"diagnostic_delta,omitempty"`
	DiagnosticDeltaNote string                    `json:"diagnostic_delta_note,omitempty"`
	Verification        *operation.VerifyRes      `json:"verification,omitempty"`
	FinalDiff           string                    `json:"final_diff"`
}

// ExecuteBatch runs an ordered sequence of registered semantic edits, fail-fast on disk.
func (s *Server) ExecuteBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
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
			response.Results = append(response.Results, BatchResult{Tool: "session_sync", Status: "error", Error: err.Error()})
		}
	}
	return response, errors.Join(resultErr, release())
}

func batchEntryResult(tool string, result BatchResult, err error) BatchResult {
	result.Tool = tool
	result.Status = "ok"
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
	}
	return result
}

var batchVerificationOptionKeys = []string{
	"trust_workspace", "jdtls_home", "java_bin", "import_maven", "metals_home", "metals_bin",
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

func snapshotBatchWorkspace(root string) (map[string]string, error) {
	files := make(map[string]string)
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
			if path != root && (entry.Name() == ".git" || entry.Name() == ".scratch" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relativize %s: %w", path, err)
		}
		data, err := workspace.ReadFile(rel)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	return files, err
}

func captureBatchDiff(response *BatchResponse, root string, before map[string]string) error {
	diff, err := finalBatchDiff(root, before)
	if err != nil {
		message := fmt.Errorf("capture final batch diff: %w", err)
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: finalDiffToolName, Status: "error", Error: message.Error()})
		return message
	}
	response.FinalDiff = diff
	return nil
}

func finalBatchDiff(root string, before map[string]string) (string, error) {
	after, err := snapshotBatchWorkspace(root)
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

func changedBatchPaths(root string, before map[string]string) ([]string, error) {
	after, err := snapshotBatchWorkspace(root)
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
		batchResult := batchMoveReceipt(batchEntry.Tool, raw, s.workDir)
		batchResult.ApplicationState = "unknown"
		if entry.ProjectErrorOutput != nil {
			projected, known, projectErr := entry.ProjectErrorOutput(err)
			if projectErr != nil {
				batchResult.ProjectionError = projectErr.Error()
			} else if known {
				encoded, encodeErr := json.Marshal(projected)
				if encodeErr != nil {
					batchResult.ProjectionError = fmt.Sprintf("encode known error projection: %v", encodeErr)
				} else {
					batchResult.Result = json.RawMessage(encoded)
					if projection, ok := projected.(map[string]any); ok {
						if state, ok := projection[applicationStateField].(string); ok {
							batchResult.ApplicationState = state
						} else if projection["status"] == "refused" && projection["safe_delete"] == true {
							batchResult.ApplicationState = "not_applied"
						}
					}
				}
			}
		}
		return batchResult, err
	}
	batchResult := resultForBatch(result, s.workDir)
	if entry.ProjectOutput != nil {
		projected, err := entry.ProjectOutput(result)
		if err != nil {
			return BatchResult{}, fmt.Errorf("project batch result for %q: %w", batchEntry.Tool, err)
		}
		encoded, err := json.Marshal(projected)
		if err != nil {
			return BatchResult{}, fmt.Errorf("encode batch result for %q: %w", batchEntry.Tool, err)
		}
		batchResult.Result = json.RawMessage(encoded)
	}
	return batchResult, nil
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
	timing := newToolRequestTiming(ctx, s.firstSemanticCallMetrics(time.Now()))
	timing.toolName = "semantic_batch"
	ctx = timing.Context(ctx)
	var request struct {
		Edits               []BatchEntry `json:"edits"`
		AutoOrganizeImports bool         `json:"auto_organize_imports"`
	}
	finishArguments := telemetry.Start(ctx, telemetry.PhaseArgumentParsing)
	if err := json.Unmarshal(raw, &request); err != nil {
		finishArguments()
		s.sendToolErrorWithTiming(id, fmt.Sprintf("invalid arguments: %v", err), timing, err)
		return
	}
	finishArguments()
	if len(request.Edits) == 0 {
		s.sendToolErrorWithTiming(id, "semantic_batch requires non-empty 'edits' list", timing)
		return
	}
	finishDispatch := telemetry.Start(ctx, telemetry.PhaseDispatch)
	response, err := s.ExecuteBatch(ctx, request.Edits, request.AutoOrganizeImports)
	finishDispatch()
	if err != nil {
		s.sendToolErrorWithTiming(id, fmt.Sprintf("batch execution error: %v", err), timing, err)
		return
	}
	for index := range response.Results {
		if response.Results[index].Status == "error" && response.Results[index].ApplicationState == "" {
			response.Results[index].ApplicationState = "unknown"
		}
	}
	finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
	bodyBytes, err := json.MarshalIndent(response, "", "  ")
	finishResponse()
	if err != nil {
		s.sendToolErrorWithTiming(id, fmt.Sprintf("serialization error: %v", err), timing, err)
		return
	}
	structured := timing.structuredContentWithResult(response)
	critical, err := criticalText(structured, string(bodyBytes))
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
	if response.Status == "error" {
		s.sendResult(id, map[string]any{
			"content":            []map[string]any{{"type": "text", "text": critical}},
			"isError":            true,
			structuredContentKey: structured,
		})
		return
	}
	s.sendResult(id, map[string]any{
		"content":            []map[string]any{{"type": "text", "text": critical}},
		"isError":            false,
		structuredContentKey: structured,
	})
}

func (s *Server) executeBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	response := &BatchResponse{Status: "ok", Results: make([]BatchResult, 0, len(edits))}
	verifyOptions := make(map[string]any)
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
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Status: "error", Error: fmt.Sprintf("edit %d params must be a JSON object; put operation arguments under params, for example {\"tool\":\"semantic_rename\",\"params\":{\"symbol\":\"Old\",\"to\":\"New\"}}", index)})
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
		if _, err := entry.Parse(raw); err != nil {
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
	workspaceBefore, err := snapshotBatchWorkspace(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("snapshot workspace before batch: %w", err)
	}
	plan, err := projectverify.Discover(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("discover batch verification plan: %w", err)
	}
	baseline, err := collectBatchVerificationBaseline(ctx, s.workDir, plan, s.enabledLanguages)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, batchJournalKey{}, &batchJournal{before: workspaceBefore, root: s.workDir, roots: make(map[string]bool)})
	buffer, err := s.executeBufferedBatchEdits(ctx, edits, response)
	if err != nil {
		response.Status = "error"
		if len(response.Results) == 0 || response.Results[len(response.Results)-1].Status != "error" {
			response.Results = append(response.Results, BatchResult{Tool: "batch_buffers", Status: "error", Error: err.Error()})
		}
		s.finishBufferedBatch(ctx, buffer, response)
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	normalizationPaths, err := s.normalizeBufferedBatch(ctx, buffer, workspaceBefore, autoOrganizeImports)
	if err != nil {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "post_process", Status: "error", Error: err.Error()})
		s.finishBufferedBatch(ctx, buffer, response)
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil //nolint:nilerr // Batch results carry the failure and disk diff.
	}
	if !s.finishBufferedBatch(ctx, buffer, response) {
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}

	verifyEntry, ok := s.registry.LookupMCP(semanticVerifyToolName)
	if !ok {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: "verification operation is not registered"})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	verifyRaw := map[string]any{"path": ".", "language": "auto", "trust_workspace": s.workspaceTrust.Trusted}
	maps.Copy(verifyRaw, verifyOptions)
	goFormattingPaths := make([]string, 0, len(normalizationPaths))
	if s.goEnabled() {
		for _, path := range normalizationPaths {
			if filepath.Ext(path) == ".go" {
				goFormattingPaths = append(goFormattingPaths, path)
			}
		}
	}
	verified, err := s.registry.Dispatch(operation.CallContext{
		Ctx: ctx, WorkDir: s.workDir,
		Project:  backend.ProjectContext{RootDir: s.workDir, WorkspaceTrust: s.workspaceTrust},
		Registry: s.registry, Service: s.service, GoFormattingPaths: goFormattingPaths, AllowedLanguages: s.enabledLanguages,
	}, verifyEntry.Key, verifyRaw)
	if err != nil {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: err.Error()})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil //nolint:nilerr // Batch results carry the failure and final diff.
	}
	verifyResult, ok := verified.(operation.VerifyRes)
	if !ok {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: fmt.Sprintf("unexpected verification result type %T", verified)})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	response.Verification = &verifyResult
	if verifyResult.Failure != "" {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: verifyResult.Failure})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	recordBatchDiagnosticDelta(response, baseline, verifyResult)
	if err := captureBatchDiff(response, s.workDir, workspaceBefore); err != nil {
		response.Status = "error"
		return response, nil //nolint:nilerr // Final diff errors are carried in the response.
	}
	return response, nil
}

func mergeBatchVerificationOptions(options, raw map[string]any) error {
	for _, key := range batchVerificationOptionKeys {
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
