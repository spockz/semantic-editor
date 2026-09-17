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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf16"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"semedit/internal/snapshot"
	"semedit/internal/symbol"
)

// Server handles MCP JSON-RPC protocol requests over stdio streams.
type Server struct {
	profile    string
	workDir    string
	service    *backend.Service
	liveReload bool
	outMu      sync.Mutex
	out        io.Writer
}

// Option configures a Server instance.
type Option func(*Server)

// WithLiveReload enables in-place re-exec and dynamic tool updates.
func WithLiveReload(enabled bool) Option {
	return func(s *Server) {
		s.liveReload = enabled
	}
}

// WithService injects the shared language service for ingress routing tests and composition.
func WithService(service *backend.Service) Option {
	return func(s *Server) {
		s.service = service
	}
}

// NewServer initializes an MCP server instance.
func NewServer(profile string, workDir string, out io.Writer, opts ...Option) *Server {
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
	s := &Server{
		profile: profile,
		workDir: workDir,
		service: backend.NewDefaultService(),
		out:     out,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
				"tools": map[string]any{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]any{
				"name":    "semedit",
				"version": "0.1.0",
			},
		}
		s.sendResult(req.ID, res)

	case "notifications/initialized":
		// Acknowledgement notification. If live-reload is enabled, emit
		// dynamic tools notification in case the server just reloaded.
		if s.liveReload {
			s.notifyToolsListChanged()
		}

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
					"language": map[string]any{
						"type":        "string",
						"enum":        []string{"auto", "go"},
						"description": "Language backend (default auto)",
					},
					"trust_workspace": map[string]any{
						"type":        "boolean",
						"description": "Explicitly trust this workspace for future external-tool backends (default false)",
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
					"language": map[string]any{
						"type":        "string",
						"enum":        []string{"auto", "go"},
						"description": "Language backend (default auto)",
					},
					"trust_workspace": map[string]any{
						"type":        "boolean",
						"description": "Explicitly trust this workspace for future external-tool backends (default false)",
					},
				},
			},
		},
		{
			"name":        "semantic_replace_body",
			"description": "Replace the body of an existing Go function or method by name. The new body is provided as bare statements (no surrounding braces). Validates and formats in memory before writing; leaves the file untouched on any syntax error.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "relative path to the Go source file",
					},
					"symbol": map[string]any{
						"type":        "string",
						"description": "function or method name, e.g. 'Foo' or '(*T).Foo'",
					},
					"body": map[string]any{
						"type":        "string",
						"description": "replacement body as bare Go statements, no braces",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "run goimports after replacement (default false)",
					},
				},
				"required": []string{"file", "symbol", "body"},
			},
		},
		{
			"name":        "semantic_scaffold_file",
			"description": "Create a new Go source file with the correct package declaration. Use 'infer' (default) for package to auto-detect from sibling non-test files. Fails if the file already exists unless overwrite is true. Does not seed declarations — use insert tools afterward.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "relative path for the new file",
					},
					"package": map[string]any{
						"type":        "string",
						"description": "package name or 'infer' (default 'infer')",
					},
					"overwrite": map[string]any{
						"type":        "boolean",
						"description": "replace existing file (default false)",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "no-op for new files; present for schema uniformity",
					},
				},
				"required": []string{"file"},
			},
		},
		{
			"name":        "semantic_insert_case",
			"description": "Insert a new case clause into an existing Go switch statement. Locates the switch by its containing function name and optional discriminant expression (omit switch_on to match a tagless switch). Validates the case source in memory before writing.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "relative path to the Go source file",
					},
					"func": map[string]any{
						"type":        "string",
						"description": "name of the function containing the switch",
					},
					"switch_on": map[string]any{
						"type":        "string",
						"description": "the switch discriminant expression, e.g. 'method'; omit for tagless switch",
					},
					"case": map[string]any{
						"type":        "string",
						"description": "full case clause source, e.g. 'case \"foo\":\\n\\treturn bar'",
					},
					"placement": map[string]any{
						"type":        "string",
						"description": "one of: first, last, before_default, before, after (default 'before_default')",
						"enum": []string{
							"first", "last", "before_default", "before", "after",
						},
					},
					"anchor": map[string]any{
						"type":        "string",
						"description": "case value to insert before/after when placement is 'before' or 'after'",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "run goimports after insertion (default false)",
					},
				},
				"required": []string{"file", "func", "case"},
			},
		},
		{
			"name":        "semantic_batch",
			"description": "Execute multiple semantic edits in sequence (fail-fast). Each edit is written to disk on success. If any edit fails, processing stops and subsequent edits are skipped. auto_organize_imports runs once per written file after all batch edits complete. Cross-file atomicity (all-or-nothing) is not supported — use semantic_verify afterward to confirm workspace state.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"edits": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"tool": map[string]any{
									"type":        "string",
									"description": "name of semantic tool to execute",
								},
								"params": map[string]any{
									"type":        "object",
									"description": "tool-specific parameter map",
								},
							},
							"required": []string{"tool", "params"},
						},
						"description": "ordered list of tool calls to execute",
					},
					"auto_organize_imports": map[string]any{
						"type":        "boolean",
						"description": "run goimports once per written file at the end (default false)",
					},
				},
				"required": []string{"edits"},
			},
		},
		{
			"name":        "semantic_snapshot",
			"description": "Capture a pre-edit transactional snapshot of specified files or the workspace root, creating an immutable journal under .scratch/snapshots/<id>/ for subsequent conflict-checked undo.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"label": map[string]any{
						"type":        "string",
						"description": "Human-readable label for the snapshot (e.g. 'before-rename-server')",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Optional description of the pending change or purpose",
					},
					"paths": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"description": "Optional list of file or directory paths to capture; if omitted, captures all workspace source files",
					},
				},
				"required": []string{"label"},
			},
		},
		{
			"name":            "semantic_undo",
			"description":     "Roll back workspace files to a previously captured snapshot state using conflict-checked atomic writes. Fails and performs zero writes if any touched file was modified after the snapshot's recorded post-edit state.",
			"destructiveHint": true,
			"annotations": map[string]any{
				"destructiveHint": true,
			},
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"snapshot_id": map[string]any{
						"type":        "string",
						"description": "The unique snapshot ID to restore (e.g. 'snap-20260917-123456-abcdef' or 'latest')",
					},
				},
				"required": []string{"snapshot_id"},
			},
		},
	}

	if s.profile != "mutations-only" {
		tools = append(tools, map[string]any{
			"name":        "resolve_symbol_location",
			"description": "Locate a Go, trusted Rust, Java, Scala, or explicitly standalone Haskell symbol in a selected source file without line counting. Rust, Java, Scala, and Haskell lookup are read-only and require explicit workspace trust; standalone Haskell rejects project markers and requires preinstalled GHC and matching HLS.",
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
					"language": map[string]any{
						"type":        "string",
						"enum":        []string{"auto", "go", "rust", "java", "scala", "haskell"},
						"description": "Language backend (default auto; Haskell requires standalone_haskell=true and supports read-only lookup only)",
					},
					"trust_workspace": map[string]any{
						"type":        "boolean",
						"description": "Explicitly trust this workspace for future external-tool backends (default false)",
					},
					"jdtls_home": map[string]any{
						"type":        "string",
						"description": "Explicit preinstalled JDT LS distribution home required for Java lookup",
					},
					"java_bin": map[string]any{
						"type":        "string",
						"description": "Optional Java 21+ executable; defaults to java on PATH",
					},
					"metals_home": map[string]any{
						"type": "string", "description": "Explicit preinstalled pinned Metals distribution home required for Scala lookup",
					},
					"metals_bin": map[string]any{
						"type": "string", "description": "Direct pinned Metals executable, alternative to metals_home",
					},
					"java_version": map[string]any{
						"type": "string", "description": "Recorded Java major version required for Scala lookup",
					},
					"standalone_haskell": map[string]any{
						"type": "boolean", "description": "Explicitly select standalone Haskell .hs lookup; rejects project markers",
					},
					"ghc_bin": map[string]any{
						"type": "string", "description": "Preinstalled GHC executable for standalone Haskell lookup",
					},
					"hls_bin": map[string]any{
						"type": "string", "description": "Preinstalled haskell-language-server-wrapper executable",
					},
					"ghc_version": map[string]any{
						"type": "string", "description": "Recorded GHC version to require",
					},
					"hls_version": map[string]any{
						"type": "string", "description": "Recorded HLS version to require",
					},
				},
				"required": []string{"symbol"},
			},
		})
	}

	if s.liveReload {
		tools = append(tools, map[string]any{
			"name":        "semantic_reload",
			"description": "Reloads the semedit MCP server in-place after recompilation (make promote) and emits notifications/tools/list_changed to discover newly added tools.",
			"inputSchema": map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
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
			Symbol            string `json:"symbol"`
			File              string `json:"file"`
			Language          string `json:"language"`
			TrustWorkspace    bool   `json:"trust_workspace"`
			JDTLSHome         string `json:"jdtls_home"`
			JavaBin           string `json:"java_bin"`
			MetalsHome        string `json:"metals_home"`
			MetalsBin         string `json:"metals_bin"`
			JavaVersion       string `json:"java_version"`
			StandaloneHaskell bool   `json:"standalone_haskell"`
			GHCBin            string `json:"ghc_bin"`
			HLSBin            string `json:"hls_bin"`
			GHCVersion        string `json:"ghc_version"`
			HLSVersion        string `json:"hls_version"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		res, err := s.service.Lookup(ctx, backend.ProjectContext{
			RootDir:           s.workDir,
			File:              args.File,
			Language:          backend.LanguageID(args.Language),
			WorkspaceTrust:    backend.NewWorkspaceTrust(s.workDir, args.TrustWorkspace),
			Java:              backend.JavaConfig{JDTLSHome: args.JDTLSHome, JavaBin: args.JavaBin},
			Scala:             backend.ScalaConfig{MetalsHome: args.MetalsHome, MetalsBin: args.MetalsBin, JavaBin: args.JavaBin, JavaVersion: args.JavaVersion},
			Haskell:           backend.HaskellConfig{Standalone: args.StandaloneHaskell, GHCBin: args.GHCBin, HLSBin: args.HLSBin, GHCVersion: args.GHCVersion, HLSVersion: args.HLSVersion},
			HaskellStandalone: args.StandaloneHaskell,
		}, args.Symbol)
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
			Language            string `json:"language"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
			TrustWorkspace      bool   `json:"trust_workspace"`
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

		autoOrg := true
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}
		result, err := s.service.Rename(ctx, backend.RenameRequest{
			Project: backend.ProjectContext{
				RootDir:        s.workDir,
				File:           args.File,
				Language:       backend.LanguageID(args.Language),
				WorkspaceTrust: backend.NewWorkspaceTrust(s.workDir, args.TrustWorkspace),
			},
			Symbol:          sym,
			To:              to,
			OrganizeImports: autoOrg,
		})
		if err != nil {
			switch {
			case errors.Is(err, symbol.ErrNotFound):
				s.sendToolError(id, fmt.Sprintf("symbol resolution error: %v", err), err)
			case errors.Is(err, backend.ErrAmbiguous):
				s.sendToolError(id, fmt.Sprintf("ambiguous symbol %q, please specify receiver or file", sym))
			default:
				s.sendToolError(id, fmt.Sprintf("rename error: %v", err), err)
			}
			return
		}

		delta := result.Diagnostics

		respText := fmt.Sprintf("Successfully renamed %s to %s.", sym, to)
		if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
			respText += fmt.Sprintf("\nDiagnostics delta (net %d):\n", delta.NetDelta)
			if len(delta.Resolved) > 0 {
				respText += fmt.Sprintf("Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
			}
			if len(delta.Introduced) > 0 {
				respText += fmt.Sprintf("Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
			}
		} else if len(result.Active) > 0 {
			respText += fmt.Sprintf("\nDiagnostics unchanged (%d active):\n%s", len(result.Active), strings.Join(result.Active, "\n"))
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
			Path           string `json:"path"`
			Language       string `json:"language"`
			TrustWorkspace bool   `json:"trust_workspace"`
		}
		_ = json.Unmarshal(params.Arguments, &args)

		targetPath := "."
		if args.Path != "" {
			targetPath = args.Path
		}

		result, err := s.service.Verify(ctx, backend.VerifyRequest{
			Project: backend.ProjectContext{
				RootDir:        s.workDir,
				File:           targetPath,
				Language:       backend.LanguageID(args.Language),
				WorkspaceTrust: backend.NewWorkspaceTrust(s.workDir, args.TrustWorkspace),
			},
			Path: targetPath,
		})
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("diagnostic check failed: %v", err))
			return
		}

		if len(result.Diagnostics) == 0 {
			s.sendToolSuccess(id, "Verification clean: 0 diagnostics.")
		} else {
			messages := make([]string, 0, len(result.Diagnostics))
			for _, diagnostic := range result.Diagnostics {
				messages = append(messages, diagnostic.Message)
			}
			s.sendToolSuccess(id, fmt.Sprintf("Diagnostics detected:\n%s", strings.Join(messages, "\n")))
		}

	case "semantic_replace_body":
		var args struct {
			File                string `json:"file"`
			Symbol              string `json:"symbol"`
			Body                string `json:"body"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" || args.Symbol == "" || args.Body == "" {
			s.sendToolError(id, "semantic_replace_body requires 'file', 'symbol', and 'body' arguments")
			return
		}

		targetPath := s.resolvePath(args.File)
		autoOrg := false
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		diff, err := astedit.ReplaceBody(ctx, targetPath, args.Symbol, args.Body, astedit.BodyOptions{
			AutoOrganizeImports: autoOrg,
		})
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("replace body error: %v", err), err)
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully replaced body of %s in %s.\n%s", args.Symbol, args.File, diff)
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_scaffold_file":
		var args struct {
			File                string `json:"file"`
			Package             string `json:"package"`
			Overwrite           bool   `json:"overwrite"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" {
			s.sendToolError(id, "semantic_scaffold_file requires 'file' argument")
			return
		}

		targetPath := s.resolvePath(args.File)
		autoOrg := false
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		pkgName, err := astedit.ScaffoldFile(ctx, targetPath, args.Package, astedit.ScaffoldOptions{
			Overwrite:           args.Overwrite,
			AutoOrganizeImports: autoOrg,
		})
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("scaffold error: %v", err), err)
			return
		}

		s.sendToolSuccess(id, fmt.Sprintf("Successfully scaffolded %s with package %s.", args.File, pkgName))

	case "semantic_insert_case":
		var args struct {
			File                string `json:"file"`
			Func                string `json:"func"`
			SwitchOn            string `json:"switch_on"`
			Case                string `json:"case"`
			Placement           string `json:"placement"`
			Anchor              string `json:"anchor"`
			AutoOrganizeImports *bool  `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if args.File == "" || args.Func == "" || args.Case == "" {
			s.sendToolError(id, "semantic_insert_case requires 'file', 'func', and 'case' arguments")
			return
		}

		targetPath := s.resolvePath(args.File)
		autoOrg := false
		if args.AutoOrganizeImports != nil {
			autoOrg = *args.AutoOrganizeImports
		}

		diagsBefore, _ := pipeline.CheckDiagnostics(ctx, s.workDir)

		diff, err := astedit.InsertCase(ctx, targetPath, args.Func, args.SwitchOn, args.Case, astedit.CaseOptions{
			Placement:           astedit.CasePlacement(args.Placement),
			AnchorCase:          args.Anchor,
			AutoOrganizeImports: autoOrg,
		})
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("insert case error: %v", err), err)
			return
		}

		diagsAfter, _ := pipeline.CheckDiagnostics(ctx, s.workDir)
		delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

		respText := fmt.Sprintf("Successfully inserted case into %s in %s.\n%s", args.Func, args.File, diff)
		respText = appendDiagnosticDelta(respText, delta)
		s.sendToolSuccess(id, respText)

	case "semantic_batch":
		var args struct {
			Edits               []BatchEntry `json:"edits"`
			AutoOrganizeImports bool         `json:"auto_organize_imports"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		if len(args.Edits) == 0 {
			s.sendToolError(id, "semantic_batch requires non-empty 'edits' list")
			return
		}

		res, err := s.ExecuteBatch(ctx, args.Edits, args.AutoOrganizeImports)
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("batch execution error: %v", err))
			return
		}

		outJSON, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("serialization error: %v", err))
			return
		}

		if res.Status == "error" {
			s.sendToolError(id, string(outJSON))
		} else {
			s.sendToolSuccess(id, string(outJSON))
		}

	case "semantic_reload":
		if !s.liveReload {
			s.sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
			return
		}

		// Reply with success before executing in-place re-exec.
		reloadResp := map[string]string{
			"status":  "ok",
			"message": "server reloading",
		}
		reloadJSON, _ := json.Marshal(reloadResp)
		s.sendToolSuccess(id, string(reloadJSON))

		if err := execReload(); err != nil {
			s.sendError(id, -32000, fmt.Sprintf("reload failed: %v", err))
		}

	case "semantic_snapshot":
		var args struct {
			Label       string   `json:"label"`
			Description string   `json:"description"`
			Paths       []string `json:"paths"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		label := strings.TrimSpace(args.Label)
		if label == "" {
			s.sendToolError(id, "semantic_snapshot requires 'label' argument")
			return
		}

		res, err := snapshot.Create(ctx, s.workDir, snapshot.CreateOptions{
			Label:       label,
			Description: args.Description,
			Paths:       args.Paths,
		})
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("snapshot error: %v", err))
			return
		}

		outJSON, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("serialization error: %v", err))
			return
		}
		s.sendToolSuccess(id, string(outJSON))

	case "semantic_undo":
		var args struct {
			SnapshotID string `json:"snapshot_id"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendToolError(id, fmt.Sprintf("invalid arguments: %v", err))
			return
		}

		snapID := strings.TrimSpace(args.SnapshotID)
		if snapID == "" {
			s.sendToolError(id, "semantic_undo requires 'snapshot_id' argument")
			return
		}

		res, err := snapshot.Undo(ctx, s.workDir, snapID)
		if err != nil {
			if errors.Is(err, snapshot.ErrConflict) {
				s.sendToolError(id, fmt.Sprintf("conflict error: %v", err))
				return
			}
			s.sendToolError(id, fmt.Sprintf("undo error: %v", err))
			return
		}

		outJSON, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			s.sendToolError(id, fmt.Sprintf("serialization error: %v", err))
			return
		}
		s.sendToolSuccess(id, string(outJSON))
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
	var visErr *astedit.VisibilityMismatchError
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
	case errors.As(err, &visErr) && visErr.Pos.IsValid():
		pos = visErr.Pos
		filePath = visErr.File
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

	var uri string
	switch {
	case filePath == "" || filePath == "snippet" || filePath == "snippet.go" || pos.Filename == "snippet" || pos.Filename == "snippet.go":
		uri = "snippet:///source"
	case strings.HasPrefix(filePath, "file://"):
		uri = filePath
	default:
		absPath := filePath
		if !filepath.IsAbs(absPath) && s.workDir != "" {
			absPath = filepath.Join(s.workDir, absPath)
		} else if !filepath.IsAbs(absPath) {
			if cwd, err := os.Getwd(); err == nil {
				absPath = filepath.Join(cwd, absPath)
			}
		}
		u := &url.URL{
			Scheme: "file",
			Path:   filepath.ToSlash(filepath.Clean(absPath)),
		}
		uri = u.String()
	}

	startLine := max(pos.Line-1, 0)
	startChar := s.resolveLSPCharacter(pos, filePath, synErr)

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

func (s *Server) resolveLSPCharacter(pos token.Position, filePath string, synErr *astedit.SyntaxError) int {
	startChar := max(pos.Column-1, 0)
	var lineContent string
	if synErr != nil && synErr.Snippet != "" {
		lines := strings.Split(synErr.Snippet, "\n")
		lineIdx := pos.Line - 1
		if lineIdx >= 0 && lineIdx < len(lines) {
			lineContent = lines[lineIdx]
		}
	} else if filePath != "" && filePath != "snippet" && filePath != "snippet.go" {
		targetPath := filePath
		if !filepath.IsAbs(targetPath) && s.workDir != "" {
			targetPath = filepath.Join(s.workDir, targetPath)
		}
		cleanTarget := filepath.Clean(targetPath)
		// #nosec G304 -- reading verified target source file for coordinate translation
		if data, err := os.ReadFile(cleanTarget); err == nil {
			lines := strings.Split(string(data), "\n")
			lineIdx := pos.Line - 1
			if lineIdx >= 0 && lineIdx < len(lines) {
				lineContent = lines[lineIdx]
			}
		}
	}

	if lineContent != "" {
		byteOffset := min(pos.Column-1, len(lineContent))
		if byteOffset > 0 {
			prefix := lineContent[:byteOffset]
			utf16Chars := 0
			for _, r := range prefix {
				utf16Chars += utf16.RuneLen(r)
			}
			return utf16Chars
		}
	}
	return startChar
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

func (s *Server) notifyToolsListChanged() {
	s.outMu.Lock()
	defer s.outMu.Unlock()

	notif := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/tools/list_changed",
	}
	data, err := json.Marshal(notif)
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
