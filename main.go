// Package main serves as the entry point for semedit, coordinating LLM intent planning with deterministic AST transformations.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
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
	case "insert":
		return runInsert(workDir, args[1:])
	case "imports":
		return runImports(workDir, args[1:])
	case "get":
		return runGet(workDir, args[1:])
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
	diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

	if err := golang.Rename(ctx, workDir, res.File, res.Line, res.Column, to); err != nil {
		fmt.Fprintf(os.Stderr, "rename execution error: %v\n", err)
		return 1
	}

	_ = pipeline.Format(ctx, workDir, ".")
	diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
	delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

	if len(delta.Introduced) > 0 {
		fmt.Fprintf(os.Stderr, "diagnostics introduced:\n%s\n", strings.Join(delta.Introduced, "\n"))
	}
	if len(delta.Resolved) > 0 {
		fmt.Printf("diagnostics resolved:\n%s\n", strings.Join(delta.Resolved, "\n"))
	}

	return 0
}

func runInsert(workDir string, args []string) int {
	fs := flag.NewFlagSet("insert", flag.ContinueOnError)
	var file string
	var placement string
	var target string
	var visibility string
	var source string
	var organizeImports bool
	fs.StringVar(&file, "file", "", "Target file path")
	fs.StringVar(&placement, "placement", "file_end", "Placement boundary (file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol)")
	fs.StringVar(&target, "target", "", "Target symbol for before_symbol / after_symbol")
	fs.StringVar(&visibility, "visibility", "", "Optional visibility constraint (public, private)")
	fs.StringVar(&source, "source", "", "Go declaration code snippet")
	fs.BoolVar(&organizeImports, "organize-imports", true, "Automatically organize imports after insertion")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "insert flags error: %v\n", err)
		return 1
	}

	source = strings.TrimSpace(source)
	if (strings.HasPrefix(source, "\"") && strings.HasSuffix(source, "\"")) ||
		(strings.HasPrefix(source, "'") && strings.HasSuffix(source, "'")) {
		source = source[1 : len(source)-1]
	}
	target = strings.Trim(strings.TrimSpace(target), `"'`)

	if file == "" || source == "" {
		fmt.Fprintf(os.Stderr, "insert requires --file and --source\n")
		return 1
	}

	targetPath := file
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(workDir, targetPath)
	}

	ctx := context.Background()
	diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

	opts := astedit.Options{
		Placement:           astedit.Placement(placement),
		TargetSymbol:        target,
		Visibility:          visibility,
		AutoOrganizeImports: organizeImports,
	}

	if err := astedit.InsertDeclaration(ctx, targetPath, source, opts); err != nil {
		fmt.Fprintf(os.Stderr, "insert error: %v\n", err)
		return 1
	}

	diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
	delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

	fmt.Printf("Successfully inserted declaration into %s\n", file)
	if len(delta.Introduced) > 0 {
		fmt.Fprintf(os.Stderr, "diagnostics introduced:\n%s\n", strings.Join(delta.Introduced, "\n"))
	}
	if len(delta.Suggestions) > 0 {
		fmt.Printf("Actionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	if len(delta.Resolved) > 0 {
		fmt.Printf("diagnostics resolved:\n%s\n", strings.Join(delta.Resolved, "\n"))
	}

	return 0
}

func runImports(workDir string, args []string) int {
	fs := flag.NewFlagSet("imports", flag.ContinueOnError)
	var file string
	fs.StringVar(&file, "file", "", "Target file path or directory (defaults to entire workspace)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "imports flags error: %v\n", err)
		return 1
	}

	var paths []string
	if file != "" {
		paths = []string{file}
	} else {
		paths = []string{"."}
	}

	ctx := context.Background()
	diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

	if err := pipeline.OrganizeImports(ctx, workDir, paths...); err != nil {
		fmt.Fprintf(os.Stderr, "organize imports error: %v\n", err)
		return 1
	}

	diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
	delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

	fmt.Println("Successfully organized imports.")
	if len(delta.Introduced) > 0 {
		fmt.Fprintf(os.Stderr, "diagnostics introduced:\n%s\n", strings.Join(delta.Introduced, "\n"))
	}
	if len(delta.Suggestions) > 0 {
		fmt.Printf("Actionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	if len(delta.Resolved) > 0 {
		fmt.Printf("diagnostics resolved:\n%s\n", strings.Join(delta.Resolved, "\n"))
	}

	return 0
}

func runGet(workDir string, args []string) int {
	if len(args) == 0 || args[0] == "" {
		fmt.Fprintf(os.Stderr, "get requires package name (e.g. semedit get github.com/google/uuid)\n")
		return 1
	}

	ctx := context.Background()
	pkg := args[0]
	if err := golang.AddDependency(ctx, workDir, pkg); err != nil {
		fmt.Fprintf(os.Stderr, "get dependency error: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully added dependency %s\n", pkg)
	return 0
}
