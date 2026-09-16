// Package mcp implements a Model Context Protocol (MCP) server over stdio for AI agent harnesses.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// Server handles MCP JSON-RPC protocol requests over stdio streams.
type Server struct {
	profile string
	workDir string
	outMu   sync.Mutex
	out     io.Writer
}

// NewServer initializes an MCP server instance.
func NewServer(profile string, workDir string, out io.Writer) *Server {
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
	return &Server{
		profile: profile,
		workDir: workDir,
		out:     out,
	}
}

// JSON-RPC 2.0 structures
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
	// Notifications (empty ID) don't receive responses
	isNotification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		var initParams struct {
			RootURI          string `json:"rootUri"`
			RootPath         string `json:"rootPath"`
			WorkspaceFolders []struct {
				URI  string `json:"uri"`
				Name string `json:"name"`
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

		res := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "semedit",
				"version": "0.1.0",
			},
		}
		s.sendResult(req.ID, res)

	case "notifications/initialized":
		// Acknowledgement notification, no response required.

	case "ping":
		if !isNotification {
			s.sendResult(req.ID, map[string]any{})
		}

	case "tools/list":
		tools := s.listTools()
		s.sendResult(req.ID, map[string]any{
			"tools": tools,
		})

	case "tools/call":
		s.handleToolCall(ctx, req.ID, req.Params)

	default:
		if !isNotification {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func (s *Server) listTools() []map[string]any {
	tools := []map[string]any{
		{
			"name":        "semantic_rename",
			"description": "Use this tool instead of replace_file_content whenever renaming an identifier, type, or function across one or more files. Executes deterministically via the host compiler/LSP without coordinate hunting.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol": map[string]any{
						"type":        "string",
						"description": "Target symbol identifier (e.g. Server.Start or ValidateToken)",
					},
					"to": map[string]any{
						"type":        "string",
						"description": "New identifier name (e.g. Serve)",
					},
					"file": map[string]any{
						"type":        "string",
						"description": "Optional file path containing the declaration to disambiguate scope",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "Automatically clean up and organize imports after rename (default true)",
					},
				},
				"required": []string{"symbol", "to"},
			},
		},
		{
			"name":        "semantic_insert_declaration",
			"description": "Use this tool instead of replace_file_content or write_to_file whenever adding a new top-level function, method, type, or constant to an existing Go file. Accurately places declarations at file boundaries, public/private sections, or relative to existing symbols without coordinate hunting.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "Target file path",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Go declaration code snippet to insert",
					},
					"placement": map[string]any{
						"type":        "string",
						"description": "Placement qualifier: file_start, file_end (default), public_start, public_end, private_start, private_end, before_symbol, after_symbol",
						"enum": []string{
							"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol",
						},
					},
					"target_symbol": map[string]any{
						"type":        "string",
						"description": "Target symbol identifier required when placement is before_symbol or after_symbol",
					},
					"visibility": map[string]any{
						"type":        "string",
						"description": "Optional validation constraint ensuring declaration matches exported scope ('public' or 'private')",
						"enum":        []string{"public", "private"},
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "Automatically resolve and organize package imports required by the inserted declaration (default true)",
					},
				},
				"required": []string{"file", "source"},
			},
		},
		{
			"name":        "semantic_insert_function",
			"description": "Use this tool instead of replace_file_content whenever adding a new top-level function or method to an existing Go file. Automatically clusters methods near their receiver types and enforces public vs private section partitioning.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "Target file path",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Function or method Go source code snippet",
					},
					"access_modifier": map[string]any{
						"type":        "string",
						"description": "Access modifier (infer, public, private, protected, package-private)",
						"enum": []string{
							"infer", "public", "private", "protected", "package-private",
						},
					},
					"placement": map[string]any{
						"type":        "string",
						"description": "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol",
						"enum": []string{
							"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol",
						},
					},
					"target_symbol": map[string]any{
						"type":        "string",
						"description": "Target symbol identifier required when placement is before_symbol or after_symbol",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "Automatically resolve and organize package imports (default true)",
					},
				},
				"required": []string{"file", "source"},
			},
		},
		{
			"name":        "semantic_insert_type",
			"description": "Use this tool instead of replace_file_content whenever adding a new struct, interface, or type alias to an existing Go file. Automatically anchors types within the appropriate section and resolves package imports.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "Target file path",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Type definition Go source code snippet",
					},
					"access_modifier": map[string]any{
						"type":        "string",
						"description": "Access modifier (infer, public, private, protected, package-private)",
						"enum": []string{
							"infer", "public", "private", "protected", "package-private",
						},
					},
					"placement": map[string]any{
						"type":        "string",
						"description": "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol",
						"enum": []string{
							"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol",
						},
					},
					"target_symbol": map[string]any{
						"type":        "string",
						"description": "Target symbol identifier required when placement is before_symbol or after_symbol",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "Automatically resolve and organize package imports (default true)",
					},
				},
				"required": []string{"file", "source"},
			},
		},
		{
			"name":        "semantic_insert_decl",
			"description": "Use this tool instead of replace_file_content whenever adding constants, variables, or declarations to an existing Go file. Intelligently merges constants and variables into existing parenthesized const (...) or var (...) blocks.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "Target file path",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Declaration Go source code snippet",
					},
					"access_modifier": map[string]any{
						"type":        "string",
						"description": "Access modifier (infer, public, private, protected, package-private)",
						"enum": []string{
							"infer", "public", "private", "protected", "package-private",
						},
					},
					"group": map[string]any{
						"type":        "string",
						"description": "Group merging behavior for const/var: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')",
						"enum":        []string{"append", "standalone"},
					},
					"placement": map[string]any{
						"type":        "string",
						"description": "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol",
						"enum": []string{
							"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol",
						},
					},
					"target_symbol": map[string]any{
						"type":        "string",
						"description": "Target symbol identifier required when placement is before_symbol or after_symbol",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "Automatically resolve and organize package imports (default true)",
					},
				},
				"required": []string{"file", "source"},
			},
		},
		{
			"name":        "semantic_organize_imports",
			"description": "Use this tool to format imports, resolve missing package imports, and strip unused imports across specified files or the workspace. Supports explicit package additions (including aliases and blank imports) and explicit removals.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "Optional file or directory path to process (defaults to entire workspace)",
					},
					"add": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"description": "Optional list of import paths to explicitly add. Supports 'path', 'alias path', or '_ path'.",
					},
					"remove": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"description": "Optional list of import paths to explicitly remove.",
					},
				},
			},
		},
		{
			"name":        "semantic_add_dependency",
			"description": "Use this tool to add an external Go dependency module (running 'go get <pkg>' and 'go mod tidy') without manual shell execution.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"package": map[string]any{
						"type":        "string",
						"description": "Package path to fetch (e.g. github.com/google/uuid@latest)",
					},
				},
				"required": []string{"package"},
			},
		},
		{
			"name":        "semantic_verify",
			"description": "Format source code and return compiler diagnostics across the workspace without rolling back.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Optional file or directory path to check and format",
					},
				},
			},
		},
	}

	if s.profile != "mutations-only" {
		tools = append(tools, map[string]any{
			"name":        "resolve_symbol_location",
			"description": "Locate the file, line, column, byte offset, and receiver for a Go symbol (e.g. 'Server.Start') without line counting.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol": map[string]any{
						"type":        "string",
						"description": "Target symbol identifier (e.g. Server.Start or ValidateToken)",
					},
					"file": map[string]any{
						"type":        "string",
						"description": "Optional file path to constrain search",
					},
				},
				"required": []string{"symbol"},
			},
		})
	}

	return tools
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

	switch params.Name {
	case "resolve_symbol_location":
		var args struct {
			Symbol string `json:"symbol"`
			File   string `json:"file"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		res, err := symbol.Resolve(s.workDir, args.File, args.Symbol)
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("resolution error: %v", err), err)
			return
		}

		outJSON, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("serialization error: %v", err), err)
			return
		}
		s.sendToolSuccess(id, string(outJSON))

	case "semantic_rename":
		var args struct {
			Symbol              string `json:"symbol"`
			To                  string `json:"to"`
			File                string `json:"file"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		sym := strings.Trim(strings.TrimSpace(args.Symbol), `"'`)
		to := strings.Trim(strings.TrimSpace(args.To), `"'`)

		if sym == "" || to == "" {
			s.sendToolError(id, "semantic_rename requires 'symbol' and 'to' arguments")
			return
		}

		res, err := symbol.Resolve(s.workDir, args.File, sym)
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("symbol resolution error: %v", err), err)
			return
		}

		if res.Ambiguous {
			s.sendToolError(id, fmt.Sprintf("ambiguous symbol %q, please specify receiver or file", sym))
			return
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		if err := golang.Rename(ctx, s.workDir, res.File, res.Line, res.Column, to); err != nil {
			s.sendToolError(id, fmt.Sprintf("rename error: %v", err))
			return
		}

		autoOrg := true
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}
		if autoOrg {
			_ = pipeline.OrganizeImports(ctx, s.workDir, ".")
		} else {
			_ = pipeline.Format(ctx, s.workDir, ".")
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully renamed %s to %s.", sym, to)
		if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
			respText += fmt.Sprintf("\nDiagnostics delta (net %d):\n", delta.NetDelta)
			if len(delta.Resolved) > 0 {
				respText += fmt.Sprintf("Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
			}
			if len(delta.Introduced) > 0 {
				respText += fmt.Sprintf("Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
			}
		} else if len(diagsAfter) > 0 {
			respText += fmt.Sprintf("\nDiagnostics unchanged (%d active):\n%s", len(diagsAfter), strings.Join(diagsAfter, "\n"))
		}
		if len(delta.Suggestions) > 0 {
			respText += fmt.Sprintf("\nActionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
		}
		s.sendToolSuccess(id, respText)

	case "semantic_insert_declaration":
		var args struct {
			File                string `json:"file"`
			Source              string `json:"source"`
			Placement           string `json:"placement"`
			TargetSymbol        string `json:"target_symbol"`
			Visibility          string `json:"visibility"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" || args.Source == "" {
			s.sendToolError(id, "semantic_insert_declaration requires 'file' and 'source' arguments")
			return
		}

		targetPath := args.File
		if s.workDir != "" && !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(s.workDir, targetPath)
		}

		autoOrg := true
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		placement := astedit.Placement(args.Placement)
		if placement == "" {
			placement = astedit.PlacementFileEnd
		}

		opts := astedit.Options{
			Placement:           placement,
			TargetSymbol:        args.TargetSymbol,
			Visibility:          args.Visibility,
			AutoOrganizeImports: autoOrg,
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		if err := astedit.InsertDeclaration(ctx, targetPath, args.Source, opts); err != nil {
			s.sendToolError(id, fmt.Sprintf("insert error: %v", err), err)
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully inserted declaration into %s.", args.File)
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_insert_function":
		var args struct {
			File                string `json:"file"`
			Source              string `json:"source"`
			AccessModifier      string `json:"access_modifier"`
			Placement           string `json:"placement"`
			TargetSymbol        string `json:"target_symbol"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" || args.Source == "" {
			s.sendToolError(id, "semantic_insert_function requires 'file' and 'source' arguments")
			return
		}

		targetPath := args.File
		if s.workDir != "" && !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(s.workDir, targetPath)
		}

		autoOrg := true
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		opts := astedit.FunctionOptions{
			AccessModifier:      astedit.AccessModifier(args.AccessModifier),
			Placement:           astedit.Placement(args.Placement),
			TargetSymbol:        args.TargetSymbol,
			AutoOrganizeImports: autoOrg,
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		if err := astedit.InsertFunction(ctx, targetPath, args.Source, opts); err != nil {
			s.sendToolError(id, fmt.Sprintf("insert function error: %v", err), err)
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully inserted function into %s.", args.File)
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_insert_type":
		var args struct {
			File                string `json:"file"`
			Source              string `json:"source"`
			AccessModifier      string `json:"access_modifier"`
			Placement           string `json:"placement"`
			TargetSymbol        string `json:"target_symbol"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" || args.Source == "" {
			s.sendToolError(id, "semantic_insert_type requires 'file' and 'source' arguments")
			return
		}

		targetPath := args.File
		if s.workDir != "" && !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(s.workDir, targetPath)
		}

		autoOrg := true
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		opts := astedit.TypeOptions{
			AccessModifier:      astedit.AccessModifier(args.AccessModifier),
			Placement:           astedit.Placement(args.Placement),
			TargetSymbol:        args.TargetSymbol,
			AutoOrganizeImports: autoOrg,
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		if err := astedit.InsertType(ctx, targetPath, args.Source, opts); err != nil {
			s.sendToolError(id, fmt.Sprintf("insert type error: %v", err), err)
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully inserted type into %s.", args.File)
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_insert_decl":
		var args struct {
			File                string `json:"file"`
			Source              string `json:"source"`
			AccessModifier      string `json:"access_modifier"`
			Group               string `json:"group"`
			Placement           string `json:"placement"`
			TargetSymbol        string `json:"target_symbol"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" || args.Source == "" {
			s.sendToolError(id, "semantic_insert_decl requires 'file' and 'source' arguments")
			return
		}

		targetPath := args.File
		if s.workDir != "" && !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(s.workDir, targetPath)
		}

		autoOrg := true
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		opts := astedit.DeclOptions{
			AccessModifier:      astedit.AccessModifier(args.AccessModifier),
			Group:               args.Group,
			Placement:           astedit.Placement(args.Placement),
			TargetSymbol:        args.TargetSymbol,
			AutoOrganizeImports: autoOrg,
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		if err := astedit.InsertDecl(ctx, targetPath, args.Source, opts); err != nil {
			s.sendToolError(id, fmt.Sprintf("insert decl error: %v", err), err)
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully inserted declaration into %s.", args.File)
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_organize_imports":
		var args struct {
			File   string   `json:"file"`
			Add    []string `json:"add"`
			Remove []string `json:"remove"`
		}
		_ = json.Unmarshal(params.Arguments, &args)

		paths := []string{"."}
		if args.File != "" {
			paths = []string{args.File}
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		opts := pipeline.ImportOptions{
			Add:    args.Add,
			Remove: args.Remove,
		}
		if err := pipeline.OrganizeImportsWithOptions(ctx, s.workDir, opts, paths...); err != nil {
			s.sendToolError(id, fmt.Sprintf("organize imports error: %v", err))
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := "Successfully organized imports."
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_add_dependency":
		var args struct {
			Package string `json:"package"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.Package == "" {
			s.sendToolError(id, "semantic_add_dependency requires 'package' argument")
			return
		}

		if err := golang.AddDependency(ctx, s.workDir, args.Package); err != nil {
			s.sendToolError(id, fmt.Sprintf("add dependency error: %v", err))
			return
		}

		s.sendToolSuccess(id, fmt.Sprintf("Successfully added dependency %s and tidied go.mod.", args.Package))

	case "semantic_verify":
		var args struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(params.Arguments, &args)

		targetPath := "."
		if args.Path != "" {
			targetPath = args.Path
		}

		_ = pipeline.Format(ctx, s.workDir, targetPath)
		diags, err := pipeline.CheckDiagnostics(ctx, s.workDir)
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("diagnostic check failed: %v", err))
			return
		}

		if len(diags) == 0 {
			s.sendToolSuccess(id, "Verification clean: 0 diagnostics.")
		} else {
			s.sendToolSuccess(id, fmt.Sprintf("Diagnostics detected:\n%s", strings.Join(diags, "\n")))
		}

	default:
		s.sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

func (s *Server) sendToolSuccess(id json.RawMessage, text string) {
	s.sendResult(id, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": false,
	})
}

func (s *Server) sendToolError(id json.RawMessage, text string, errs ...error) {
	result := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": true,
	}

	if len(errs) > 0 && errs[0] != nil {
		if loc := s.extractLocation(errs[0]); loc != nil {
			result["location"] = loc
		}
	}

	s.sendResult(id, result)
}

func (s *Server) extractLocation(err error) map[string]any {
	var pos token.Position
	var filePath string

	var synErr *astedit.SyntaxError
	var symErr *symbol.SymbolError
	var placeErr *astedit.PlacementError

	switch {
	case errors.As(err, &synErr) && synErr.Pos.IsValid():
		pos = synErr.Pos
		filePath = synErr.File
		if filePath == "" {
			filePath = pos.Filename
		}
	case errors.As(err, &symErr) && symErr.Pos.IsValid():
		pos = symErr.Pos
		filePath = symErr.File
		if filePath == "" {
			filePath = pos.Filename
		}
	case errors.As(err, &placeErr) && placeErr.Pos.IsValid():
		pos = placeErr.Pos
		filePath = placeErr.File
		if filePath == "" {
			filePath = pos.Filename
		}
	}

	if !pos.IsValid() {
		return nil
	}

	uri := filePath
	if !strings.HasPrefix(uri, "file://") {
		if !filepath.IsAbs(uri) && s.workDir != "" {
			uri = filepath.Join(s.workDir, uri)
		}
		uri = "file://" + filepath.ToSlash(filepath.Clean(uri))
	}

	startLine := max(pos.Line-1, 0)
	startChar := max(pos.Column-1, 0)

	return map[string]any{
		"uri": uri,
		"range": map[string]any{
			"start": map[string]int{
				"line":      startLine,
				"character": startChar,
			},
			"end": map[string]int{
				"line":      startLine,
				"character": startChar,
			},
		},
	}
}

func (s *Server) sendResult(id json.RawMessage, result any) {
	s.writeJSON(&jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) sendError(id json.RawMessage, code int, message string) {
	s.writeJSON(&jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	})
}

func (s *Server) writeJSON(resp *jsonRPCResponse) {
	s.outMu.Lock()
	defer s.outMu.Unlock()

	data, err := json.Marshal(resp)
	if err == nil {
		_, _ = s.out.Write(append(data, '\n'))
	}
}

func appendDiagnosticDelta(base string, delta pipeline.DiagnosticDelta) string {
	if len(delta.Introduced) == 0 && len(delta.Resolved) == 0 && len(delta.Suggestions) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
		fmt.Fprintf(&sb, "\nDiagnostics delta (net %d):\n", delta.NetDelta)
		if len(delta.Resolved) > 0 {
			fmt.Fprintf(&sb, "Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
		}
		if len(delta.Introduced) > 0 {
			fmt.Fprintf(&sb, "Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
		}
	}
	if len(delta.Suggestions) > 0 {
		fmt.Fprintf(&sb, "\nActionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	return sb.String()
}
