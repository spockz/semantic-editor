// Package mcp executes batches by dispatching only operations registered as batchable.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"semedit/internal/astedit"

	"semedit/internal/backend"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
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
	Status          string                    `json:"status"`
	Results         []BatchResult             `json:"results"`
	DiagnosticDelta *pipeline.DiagnosticDelta `json:"diagnostic_delta,omitempty"`
	FinalDiff       string                    `json:"final_diff"`
}

// ExecuteBatch runs an ordered sequence of registered semantic edits, fail-fast on disk.
func (s *Server) ExecuteBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	response := &BatchResponse{Status: "ok", Results: make([]BatchResult, 0, len(edits))}
	workspaceBefore, err := snapshotBatchWorkspace(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("snapshot workspace before batch: %w", err)
	}
	before, err := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), s.workDir)
	if err != nil {
		return nil, fmt.Errorf("batch diagnostics before: %w", err)
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
		} else {
			if filepath.Clean(file) == filepath.Clean(s.workDir) {
				err = formatWorkspace(ctx, s.workDir)
			} else {
				err = pipeline.Format(ctx, s.workDir, file)
			}
		}
		if err != nil {
			response.Status = "error"
			message := fmt.Errorf("post-process %s: %w", file, err)
			response.Results = append(response.Results, BatchResult{Tool: "post_process", Status: "error", Error: message.Error()})
			_ = captureBatchDiff(response, s.workDir, workspaceBefore)
			return response, nil
		}
	}
	after, err := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationAfter), s.workDir)
	if err != nil {
		response.Status = "error"
		message := fmt.Errorf("batch diagnostics after: %w", err)
		response.Results = append(response.Results, BatchResult{Tool: "post_process", Status: "error", Error: message.Error()})
		_ = captureBatchDiff(response, s.workDir, workspaceBefore)
		return response, nil
	}
	delta := pipeline.ComputeDelta(before, after)
	response.DiagnosticDelta = &delta
	if err := captureBatchDiff(response, s.workDir, workspaceBefore); err != nil {
		response.Status = "error"
		return response, nil //nolint:nilerr // Final diff errors are carried in the response.
	}
	return response, nil
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
	result, err := s.registry.Dispatch(operation.CallContext{
		Ctx:               ctx,
		WorkDir:           s.workDir,
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

func formatWorkspace(ctx context.Context, workDir string) error {
	paths := make([]string, 0)
	err := filepath.Walk(workDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == ".scratch" || info.Name() == "vendor" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk workspace for formatting: %w", err)
	}
	if len(paths) == 0 {
		return nil
	}
	return pipeline.Format(ctx, workDir, paths...)
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
	timing := newToolRequestTiming(ctx, s.firstSemanticCallMetrics(time.Now()))
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
		s.sendResult(id, map[string]any{
			"content":           []map[string]any{{"type": "text", "text": string(text)}},
			"isError":           true,
			"structuredContent": timing.structuredContentWithResult(response),
		})
		return
	}
	s.sendToolSuccessWithTimingAndResult(id, string(text), response, timing)
}
