// Package mcp executes batches by dispatching only operations registered as batchable.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

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
	Status  string        `json:"status"`
	Results []BatchResult `json:"results"`
}

// ExecuteBatch runs an ordered sequence of registered semantic edits, fail-fast on disk.
func (s *Server) ExecuteBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	response := &BatchResponse{Status: "ok", Results: make([]BatchResult, 0, len(edits))}
	writtenFiles := make(map[string]struct{})
	for _, batchEntry := range edits {
		result, writtenFile, err := s.executeBatchEdit(ctx, batchEntry)
		if err != nil {
			response.Status = "error"
			response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Symbol: result.Symbol, Status: "error", Error: err.Error()})
			return response, nil
		}
		if writtenFile != "" {
			writtenFiles[writtenFile] = struct{}{}
		}
		response.Results = append(response.Results, BatchResult{Tool: batchEntry.Tool, Symbol: result.Symbol, Status: "ok", Diff: result.Diff})
	}
	for file := range writtenFiles {
		var err error
		if autoOrganizeImports {
			err = pipeline.OrganizeImports(ctx, s.workDir, file)
		} else {
			err = pipeline.Format(ctx, s.workDir, file)
		}
		if err != nil {
			return response, fmt.Errorf("post-process %s: %w", file, err)
		}
	}
	return response, nil
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
		Ctx:          ctx,
		WorkDir:      s.workDir,
		Registry:     s.registry,
		Service:      s.service,
		InBatch:      true,
		DeferImports: true,
	}, entry.Key, raw)
	if err != nil {
		return BatchResult{}, "", err
	}
	batchResult := resultForBatch(result)
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
	timing := newToolRequestTiming(ctx)
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
		s.sendToolErrorWithTiming(id, string(text), timing)
		return
	}
	s.sendToolSuccessWithTiming(id, string(text), timing)
}
