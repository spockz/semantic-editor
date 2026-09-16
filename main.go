// Package main serves as the entry point for semedit, coordinating LLM intent planning with deterministic AST transformations.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"semedit/internal/adapters/golang"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Println("semedit initialized")
		return 0
	}

	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	switch args[0] {
	case "lookup":
		return runLookup(workDir, args[1:])
	case "rename":
		return runRename(workDir, args[1:])
	case "mcp":
		return runMCP(workDir, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		return 1
	}
}

func runMCP(workDir string, args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	var profile string
	fs.StringVar(&profile, "profile", "full", "MCP server profile (full, mutations-only)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "mcp flags error: %v\n", err)
		return 1
	}

	srv := mcp.NewServer(profile, workDir, os.Stdout)
	if err := srv.Serve(context.Background(), os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		return 1
	}
	return 0
}

func runLookup(workDir string, args []string) int {
	fs := flag.NewFlagSet("lookup", flag.ContinueOnError)
	var file string
	var sym string
	fs.StringVar(&file, "file", "", "Target file path")
	fs.StringVar(&sym, "symbol", "", "Target symbol identifier")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "lookup flags error: %v\n", err)
		return 1
	}

	if sym == "" {
		fmt.Fprintf(os.Stderr, "lookup requires --symbol\n")
		return 1
	}

	res, err := symbol.Resolve(workDir, file, sym)
	if err != nil {
		if errors.Is(err, symbol.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
			return 1
		}
		fmt.Fprintf(os.Stderr, "lookup error: %v\n", err)
		return 1
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json format error: %v\n", err)
		return 1
	}

	fmt.Println(string(data))
	return 0
}

func runRename(workDir string, args []string) int {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	var file string
	var sym string
	var to string
	fs.StringVar(&file, "file", "", "Target file path")
	fs.StringVar(&sym, "symbol", "", "Target symbol identifier")
	fs.StringVar(&to, "to", "", "New name for target symbol")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "rename flags error: %v\n", err)
		return 1
	}

	sym = strings.Trim(strings.TrimSpace(sym), `"'`)
	to = strings.Trim(strings.TrimSpace(to), `"'`)

	if sym == "" || to == "" {
		fmt.Fprintf(os.Stderr, "rename requires --symbol and --to\n")
		return 1
	}

	res, err := symbol.Resolve(workDir, file, sym)
	if err != nil {
		if errors.Is(err, symbol.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
			return 1
		}
		fmt.Fprintf(os.Stderr, "rename resolution error: %v\n", err)
		return 1
	}

	if res.Ambiguous {
		fmt.Fprintf(os.Stderr, "ambiguous symbol %q, please qualify receiver\n", sym)
		return 1
	}

	ctx := context.Background()
	if err := golang.Rename(ctx, workDir, res.File, res.Line, res.Column, to); err != nil {
		fmt.Fprintf(os.Stderr, "rename execution error: %v\n", err)
		return 1
	}

	_ = pipeline.Format(ctx, workDir, ".")
	_, _ = pipeline.CheckDiagnostics(ctx, workDir)

	return 0
}
