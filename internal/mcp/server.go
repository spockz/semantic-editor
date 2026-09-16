// Package mcp implements a Model Context Protocol (MCP) server over stdio for AI agent harnesses.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"semedit/internal/adapters/golang"
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
				},
				"required": []string{"symbol", "to"},
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
			s.sendToolError(id, fmt.Sprintf("resolution error: %v", err))
			return
		}

		outJSON, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("serialization error: %v", err))
			return
		}
		s.sendToolSuccess(id, string(outJSON))

	case "semantic_rename":
		var args struct {
			Symbol string `json:"symbol"`
			To     string `json:"to"`
			File   string `json:"file"`
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
			s.sendToolError(id, fmt.Sprintf("symbol resolution error: %v", err))
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

		_ = pipeline.Format(ctx, s.workDir, ".")
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
		s.sendToolSuccess(id, respText)

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

func (s *Server) sendToolError(id json.RawMessage, text string) {
	s.sendResult(id, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": true,
	})
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
