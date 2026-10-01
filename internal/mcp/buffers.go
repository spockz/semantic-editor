// Package mcp publishes buffered batch segments at disk-only operation and verification boundaries.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/godiagnostics"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"slices"
	"strings"
	"time"
)

type batchJournalKey struct{}

type batchJournal struct {
	before map[string]string
	roots  map[string]bool
	root   string
}

func (s *Server) executeBufferedBatchEdits(ctx context.Context, edits []BatchEntry, response *BatchResponse) (*sourcebuf.Buffer, error) {
	buffer, err := sourcebuf.New(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("create batch source buffers: %w", err)
	}
	for _, batchEntry := range edits {
		entry, ok := s.registry.LookupMCP(batchEntry.Tool)
		if !ok {
			return buffer, fmt.Errorf("unregistered batch operation %q", batchEntry.Tool)
		}
		raw := map[string]any{}
		if err := json.Unmarshal(batchEntry.Params, &raw); err != nil {
			return buffer, fmt.Errorf("parse buffered batch parameters: %w", err)
		}
		buffered, err := entry.SupportsBufferedBatch(operation.CallContext{Ctx: ctx, WorkDir: s.workDir, Project: backend.ProjectContext{RootDir: s.workDir, WorkspaceTrust: s.workspaceTrust}, Registry: s.registry, Service: s.service}, raw)
		if err != nil {
			response.Status = "error"
			response.Results = append(response.Results, batchEntryResult(batchEntry.Tool, BatchResult{}, err))
			return buffer, err
		}
		editCtx := sourcebuf.WithBuffer(ctx, buffer)
		if !buffered {
			if !s.finishBufferedBatch(ctx, buffer, response) {
				return nil, errors.New("publish batch before disk operation failed")
			}
			editCtx = sourcebuf.WithoutBuffer(ctx)
		}
		result, err := s.executeBatchEdit(editCtx, batchEntry)
		response.Results = append(response.Results, batchEntryResult(batchEntry.Tool, result, err))
		if err != nil {
			response.Status = "error"
			return buffer, err
		}
		if !buffered {
			buffer, err = sourcebuf.New(s.workDir)
			if err != nil {
				return nil, fmt.Errorf("restart batch source buffers: %w", err)
			}
		}
	}
	return buffer, nil
}

func (s *Server) finishBufferedBatch(ctx context.Context, buffer *sourcebuf.Buffer, response *BatchResponse) bool {
	if buffer == nil {
		return response.Status != "error"
	}
	if err := recordBufferedChanges(ctx, buffer); err != nil {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "buffer_publish", Status: "error", Error: err.Error()})
		return false
	}
	publicationCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := buffer.Flush(publicationCtx); err != nil {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "buffer_publish", Status: "error", Error: err.Error()})
		if syncErr := godiagnostics.InvalidateWithContext(publicationCtx); syncErr != nil {
			response.Results = append(response.Results, BatchResult{Tool: "session_sync", Status: "error", Error: syncErr.Error()})
		}
		return false
	}
	if err := godiagnostics.SynchronizeWithContext(sourcebuf.WithoutBuffer(publicationCtx)); err != nil {
		response.Status = "error"
		response.Results = append(response.Results, BatchResult{Tool: "session_sync", Status: "error", Error: err.Error()})
		return false
	}
	return true
}

func (s *Server) normalizeBufferedBatch(ctx context.Context, buffer *sourcebuf.Buffer, before map[string]string, organize bool) ([]string, error) {
	if err := recordBufferedChanges(ctx, buffer); err != nil {
		return nil, err
	}
	if journal, _ := ctx.Value(batchJournalKey{}).(*batchJournal); journal != nil {
		for root := range journal.roots {
			if err := buffer.AuthorizeWrites(ctx, root); err != nil {
				return nil, err
			}
		}
	}

	paths, err := changedBatchPaths(s.workDir, before)
	if err != nil {
		return nil, fmt.Errorf("collect changed batch paths: %w", err)
	}
	selected := make(map[string]bool, len(paths))
	for _, path := range paths {
		selected[path] = true
	}
	for _, path := range buffer.ChangedPaths() {
		relative, err := filepath.Rel(s.workDir, path)
		if err != nil {
			return nil, fmt.Errorf("relativize buffered source: %w", err)
		}
		relative = filepath.ToSlash(relative)
		if strings.HasPrefix(relative, "../") {
			content, err := sourcebuf.ReadFile(sourcebuf.WithBuffer(ctx, buffer), path)
			if err != nil {
				return nil, err
			}
			if original, exists := before[relative]; exists && original == string(content) {
				delete(selected, relative)
				continue
			}
		}
		selected[relative] = true
	}
	paths = paths[:0]
	goPaths := make([]string, 0)
	for path := range selected {
		paths = append(paths, path)
		if filepath.Ext(path) == ".go" {
			goPaths = append(goPaths, path)
		}
	}
	slices.Sort(paths)
	slices.Sort(goPaths)
	if len(goPaths) == 0 {
		return paths, nil
	}
	bufferedCtx := sourcebuf.WithBuffer(ctx, buffer)
	if organize {
		err = pipeline.OrganizeImports(bufferedCtx, s.workDir, goPaths...)
	} else {
		err = pipeline.Format(bufferedCtx, s.workDir, goPaths...)
	}
	if err != nil {
		return paths, fmt.Errorf("normalize buffered batch: %w", err)
	}
	return paths, nil
}

func recordBufferedChanges(ctx context.Context, buffer *sourcebuf.Buffer) error {
	journal, _ := ctx.Value(batchJournalKey{}).(*batchJournal)
	if journal == nil {
		return nil
	}
	for _, root := range buffer.AuthorizedRoots() {
		journal.roots[root] = true
	}
	for _, change := range buffer.Changes() {
		relative, err := filepath.Rel(journal.root, change.Path)
		if err != nil {
			return fmt.Errorf("record buffered output: %w", err)
		}
		relative = filepath.ToSlash(relative)
		if relative != ".." && !strings.HasPrefix(relative, "../") {
			continue
		}
		if _, exists := journal.before[relative]; !exists {
			journal.before[relative] = string(change.Before)
		}
	}
	return nil
}
