// Package mcp implements a Model Context Protocol (MCP) server over stdio for AI agent harnesses.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
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
	Status string `json:"status"` // "ok" | "error"
	Diff   string `json:"diff,omitempty"`
	Error  string `json:"error,omitempty"`
}

// BatchResponse aggregates the outcomes of a semantic batch execution.
type BatchResponse struct {
	Status  string        `json:"status"` // "ok" | "error"
	Results []BatchResult `json:"results"`
}

// ExecuteBatch runs an ordered sequence of semantic edits fail-fast on disk.
func (s *Server) ExecuteBatch(ctx context.Context, edits []BatchEntry, autoOrganizeImports bool) (*BatchResponse, error) {
	resp := &BatchResponse{
		Status:  "ok",
		Results: make([]BatchResult, 0, len(edits)),
	}

	writtenFiles := make(map[string]bool)

	for _, entry := range edits {
		result, writtenFile, err := s.executeBatchEdit(ctx, entry)
		if err != nil {
			resp.Status = "error"
			resp.Results = append(resp.Results, BatchResult{
				Tool:   entry.Tool,
				Symbol: result.Symbol,
				Status: "error",
				Error:  err.Error(),
			})
			break // Fail-fast on first error
		}

		if writtenFile != "" {
			writtenFiles[writtenFile] = true
		}

		resp.Results = append(resp.Results, BatchResult{
			Tool:   entry.Tool,
			Symbol: result.Symbol,
			Status: "ok",
			Diff:   result.Diff,
		})
	}

	// Post-process all files written during the batch
	for file := range writtenFiles {
		if autoOrganizeImports {
			_ = pipeline.OrganizeImports(ctx, s.workDir, file)
		} else {
			_ = pipeline.Format(ctx, s.workDir, file)
		}
	}

	return resp, nil
}

func (s *Server) executeBatchEdit(ctx context.Context, entry BatchEntry) (BatchResult, string, error) {
	var empty BatchResult

	switch entry.Tool {
	case "semantic_replace_body":
		var args struct {
			File string `json:"file"`
			Sym  string `json:"symbol"`
			Body string `json:"body"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		diff, err := astedit.ReplaceBody(ctx, targetPath, args.Sym, args.Body, astedit.BodyOptions{
			AutoOrganizeImports: false,
		})
		if err != nil {
			return BatchResult{Symbol: args.Sym}, "", err
		}
		return BatchResult{Symbol: args.Sym, Diff: diff}, targetPath, nil

	case "semantic_scaffold_file":
		var args struct {
			File      string `json:"file"`
			Package   string `json:"package"`
			Overwrite bool   `json:"overwrite"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		pkg, err := astedit.ScaffoldFile(ctx, targetPath, args.Package, astedit.ScaffoldOptions{
			Overwrite: args.Overwrite,
		})
		if err != nil {
			return BatchResult{Symbol: args.Package}, "", err
		}
		return BatchResult{Symbol: pkg}, targetPath, nil

	case "semantic_insert_case":
		var args struct {
			File      string `json:"file"`
			Func      string `json:"func"`
			SwitchOn  string `json:"switch_on"`
			Case      string `json:"case"`
			Placement string `json:"placement"`
			Anchor    string `json:"anchor"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		diff, err := astedit.InsertCase(ctx, targetPath, args.Func, args.SwitchOn, args.Case, astedit.CaseOptions{
			Placement:  astedit.CasePlacement(args.Placement),
			AnchorCase: args.Anchor,
		})
		if err != nil {
			return BatchResult{Symbol: args.Func}, "", err
		}
		return BatchResult{Symbol: args.Func, Diff: diff}, targetPath, nil

	case "semantic_insert_declaration":
		var args struct {
			File         string `json:"file"`
			Source       string `json:"source"`
			Placement    string `json:"placement"`
			TargetSymbol string `json:"target_symbol"`
			Visibility   string `json:"visibility"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		placement := astedit.Placement(args.Placement)
		if placement == "" {
			placement = astedit.PlacementFileEnd
		}
		err := astedit.InsertDeclaration(ctx, targetPath, args.Source, astedit.Options{
			Placement:    placement,
			TargetSymbol: args.TargetSymbol,
			Visibility:   args.Visibility,
		})
		if err != nil {
			return BatchResult{Symbol: args.TargetSymbol}, "", err
		}
		return BatchResult{Symbol: args.TargetSymbol}, targetPath, nil

	case "semantic_insert_function":
		var args struct {
			File           string `json:"file"`
			Source         string `json:"source"`
			AccessModifier string `json:"access_modifier"`
			Placement      string `json:"placement"`
			TargetSymbol   string `json:"target_symbol"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		err := astedit.InsertFunction(ctx, targetPath, args.Source, astedit.FunctionOptions{
			AccessModifier: astedit.AccessModifier(args.AccessModifier),
			Placement:      astedit.Placement(args.Placement),
			TargetSymbol:   args.TargetSymbol,
		})
		if err != nil {
			return BatchResult{Symbol: args.TargetSymbol}, "", err
		}
		return BatchResult{Symbol: args.TargetSymbol}, targetPath, nil

	case "semantic_insert_type":
		var args struct {
			File           string `json:"file"`
			Source         string `json:"source"`
			AccessModifier string `json:"access_modifier"`
			Placement      string `json:"placement"`
			TargetSymbol   string `json:"target_symbol"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		err := astedit.InsertType(ctx, targetPath, args.Source, astedit.TypeOptions{
			AccessModifier: astedit.AccessModifier(args.AccessModifier),
			Placement:      astedit.Placement(args.Placement),
			TargetSymbol:   args.TargetSymbol,
		})
		if err != nil {
			return BatchResult{Symbol: args.TargetSymbol}, "", err
		}
		return BatchResult{Symbol: args.TargetSymbol}, targetPath, nil

	case "semantic_insert_decl":
		var args struct {
			File           string `json:"file"`
			Source         string `json:"source"`
			AccessModifier string `json:"access_modifier"`
			Group          string `json:"group"`
			Placement      string `json:"placement"`
			TargetSymbol   string `json:"target_symbol"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		err := astedit.InsertDecl(ctx, targetPath, args.Source, astedit.DeclOptions{
			AccessModifier: astedit.AccessModifier(args.AccessModifier),
			Group:          args.Group,
			Placement:      astedit.Placement(args.Placement),
			TargetSymbol:   args.TargetSymbol,
		})
		if err != nil {
			return BatchResult{Symbol: args.TargetSymbol}, "", err
		}
		return BatchResult{Symbol: args.TargetSymbol}, targetPath, nil

	case "semantic_rename":
		var args struct {
			Symbol string `json:"symbol"`
			To     string `json:"to"`
			File   string `json:"file"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		res, err := symbol.Resolve(s.workDir, args.File, args.Symbol)
		if err != nil {
			return BatchResult{Symbol: args.Symbol}, "", err
		}
		if res.Ambiguous {
			return BatchResult{Symbol: args.Symbol}, "", fmt.Errorf("ambiguous symbol %q", args.Symbol)
		}
		if err := golang.Rename(ctx, s.workDir, res.File, res.Line, res.Column, args.To); err != nil {
			return BatchResult{Symbol: args.Symbol}, "", err
		}
		return BatchResult{Symbol: args.Symbol}, s.resolvePath(res.File), nil

	case "semantic_organize_imports":
		var args struct {
			File   string   `json:"file"`
			Add    []string `json:"add"`
			Remove []string `json:"remove"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		targetPath := s.resolvePath(args.File)
		err := pipeline.OrganizeImportsWithOptions(ctx, s.workDir, pipeline.ImportOptions{
			Add:    args.Add,
			Remove: args.Remove,
		}, targetPath)
		if err != nil {
			return BatchResult{Symbol: args.File}, "", err
		}
		return BatchResult{Symbol: args.File}, targetPath, nil

	case "semantic_add_dependency":
		var args struct {
			Package string `json:"package"`
		}
		if err := json.Unmarshal(entry.Params, &args); err != nil {
			return empty, "", fmt.Errorf("invalid arguments: %w", err)
		}
		if err := golang.AddDependency(ctx, s.workDir, args.Package); err != nil {
			return BatchResult{Symbol: args.Package}, "", err
		}
		return BatchResult{Symbol: args.Package}, filepath.Join(s.workDir, "go.mod"), nil

	case "semantic_batch":
		return empty, "", fmt.Errorf("nested semantic_batch is not supported")

	default:
		return empty, "", fmt.Errorf("unknown tool for batch execution: %s", entry.Tool)
	}
}

func (s *Server) resolvePath(path string) string {
	if path == "" {
		return s.workDir
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.workDir, path)
}
