// Package mcp executes batches by dispatching only operations registered as batchable.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"semedit/internal/astedit"

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
	Tool   string `json:"tool"`
	Symbol string `json:"symbol,omitempty"`
	Status string `json:"status"`
	Diff   string `json:"diff,omitempty"`
	Error  string `json:"error,omitempty"`
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
		if err := json.Unmarshal(batchEntry.Params, &raw); err != nil {
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
	baseline, err := collectBatchVerificationBaseline(ctx, s.workDir, plan)
	if err != nil {
		return nil, err
	}
	writtenFiles := make(map[string]struct{})
	workspaceScope := false
	for _, batchEntry := range edits {
		result, writtenFile, err := s.executeBatchEdit(ctx, batchEntry)
		if err != nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Symbol: result.Symbol, Status: "error", Error: err.Error()})
			_ = captureBatchDiff(response, s.workDir, workspaceBefore)
			return response, nil //nolint:nilerr // Batch execution errors are carried in the response status and entry.
		}
		if writtenFile != "" {
			if filepath.Clean(writtenFile) == filepath.Clean(s.workDir) {
				workspaceScope = true
			} else {
				writtenFiles[writtenFile] = struct{}{}
			}
		}
		response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Symbol: result.Symbol, Status: "ok", Diff: result.Diff})
	}
	if workspaceScope {
		writtenFiles = map[string]struct{}{s.workDir: {}}
	}
	for file := range writtenFiles {
		var err error
		if autoOrganizeImports {
			err = pipeline.OrganizeImports(ctx, s.workDir, file)
		}
		if err != nil {
			response.Status = "error"
			message := fmt.Errorf("post-process %s: %w", file, err)
			response.Results = append(response.Results, BatchResult{Tool: "post_process", Status: "error", Error: message.Error()})
			_ = captureBatchDiff(response, s.workDir, workspaceBefore)
			return response, nil
		}
	}
	verifyEntry, ok := s.registry.LookupMCP("semantic_verify")
	if !ok {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "semantic_verify", Status: "error", Error: "verification operation is not registered"})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	verifyRaw := map[string]any{"path": ".", "language": "auto", "trust_workspace": s.workspaceTrust.Trusted}
	maps.Copy(verifyRaw, verifyOptions)
	verified, err := s.registry.Dispatch(operation.CallContext{Ctx: ctx, WorkDir: s.workDir, Project: backend.ProjectContext{RootDir: s.workDir, WorkspaceTrust: s.workspaceTrust}, Registry: s.registry, Service: s.service}, verifyEntry.Key, verifyRaw)
	if err != nil {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "semantic_verify", Status: "error", Error: err.Error()})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil //nolint:nilerr // Batch results carry the failure and final diff.
	}
	verifyResult, ok := verified.(operation.VerifyRes)
	if !ok {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "semantic_verify", Status: "error", Error: fmt.Sprintf("unexpected verification result type %T", verified)})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	response.Verification = &verifyResult
	if verifyResult.Failure != "" {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "semantic_verify", Status: "error", Error: verifyResult.Failure})
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
		response.Results = append(response.Results, BatchResult{Tool: "final_diff", Status: "error", Error: message.Error()})
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

func (s *Server) executeBatchEdit(ctx context.Context, batchEntry BatchEntry) (BatchResult, string, error) {
	entry, ok := s.registry.LookupMCP(batchEntry.Tool)
	if !ok || !entry.Batchable {
		return BatchResult{}, "", fmt.Errorf("tool is not batchable: %s", batchEntry.Tool)
	}
	raw := map[string]any{}
	if err := json.Unmarshal(batchEntry.Params, &raw); err != nil {
		return BatchResult{}, "", fmt.Errorf("invalid arguments: %w", err)
	}
	if err := s.validateLanguageAllowed(entry, raw); err != nil {
		return BatchResult{}, "", err
	}
	result, err := s.registry.Dispatch(operation.CallContext{
		Ctx:               ctx,
		WorkDir:           s.workDir,
		Project:           backend.ProjectContext{RootDir: s.workDir, WorkspaceTrust: s.workspaceTrust},
		Registry:          s.registry,
		Service:           s.service,
		InBatch:           true,
		DeferImports:      true,
		DeferVerification: true,
	}, entry.Key, raw)
	if err != nil {
		return BatchResult{}, "", err
	}
	batchResult := resultForBatch(result)
	if _, ok := result.(*backend.RenameResult); ok {
		return batchResult, s.workDir, nil
	}
	if outcome, ok := result.(operation.FileOutcome); ok {
		return batchResult, filepath.Clean(outcome.WrittenFile()), nil
	}
	return batchResult, "", nil
}

func resultForBatch(result any) BatchResult {
	switch typed := result.(type) {
	case operation.FileEditRes:
		return BatchResult{Symbol: typed.Symbol, Diff: typed.Diff}
	case operation.ScaffoldFileRes:
		return BatchResult{Symbol: typed.Package}
	case operation.BuildDependencyRes:
		return BatchResult{Symbol: typed.Package}
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
	finishResponse := telemetry.Start(ctx, telemetry.PhaseResponseFormatting)
	text, err := json.MarshalIndent(response, "", "  ")
	finishResponse()
	if err != nil {
		s.sendToolErrorWithTiming(id, fmt.Sprintf("serialization error: %v", err), timing, err)
		return
	}
	if response.Status == "error" {
		s.recordToolMetrics(timing)
		s.sendResult(id, map[string]any{
			"content":            []map[string]any{{"type": "text", "text": string(text)}},
			"isError":            true,
			structuredContentKey: timing.structuredContentWithResult(response),
		})
		return
	}
	s.sendToolSuccessWithTimingAndResult(id, string(text), response, timing)
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
