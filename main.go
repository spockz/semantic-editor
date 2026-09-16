// Package main serves as the entry point for semedit, coordinating LLM intent planning with deterministic AST transformations.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

var errCommandFailed = errors.New("command execution failed")

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

	rootCmd := newRootCmd(workDir)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errCommandFailed) {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return 1
	}
	return 0
}

func newRootCmd(workDir string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "semedit",
		Short:         "Semantic Editor for LLM Intent-Driven Code Refactoring",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("semedit initialized")
			return nil
		},
	}

	rootCmd.AddCommand(
		newLookupCmd(workDir),
		newRenameCmd(workDir),
		newInsertCmd(workDir),
		newInsertFuncCmd(workDir),
		newInsertTypeCmd(workDir),
		newInsertDeclCmd(workDir),
		newImportsCmd(workDir),
		newGetCmd(workDir),
		newMCPCmd(workDir),
	)

	return rootCmd
}

func newLookupCmd(workDir string) *cobra.Command {
	var file string
	var sym string

	cmd := &cobra.Command{
		Use:           "lookup",
		Short:         "Resolve symbol location and AST coordinates",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if sym == "" {
				fmt.Fprintf(os.Stderr, "lookup requires --symbol\n")
				return errCommandFailed
			}

			res, err := symbol.Resolve(workDir, file, sym)
			if err != nil {
				if errors.Is(err, symbol.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
					return errCommandFailed
				}
				fmt.Fprintf(os.Stderr, "lookup error: %v\n", err)
				return errCommandFailed
			}

			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "json format error: %v\n", err)
				return errCommandFailed
			}

			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&sym, "symbol", "s", "", "Target symbol identifier")
	return cmd
}

func newRenameCmd(workDir string) *cobra.Command {
	var file string
	var sym string
	var to string

	cmd := &cobra.Command{
		Use:           "rename",
		Short:         "Execute compiler-accurate semantic symbol rename across workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sym = strings.Trim(strings.TrimSpace(sym), `"'`)
			to = strings.Trim(strings.TrimSpace(to), `"'`)

			if sym == "" || to == "" {
				fmt.Fprintf(os.Stderr, "rename requires --symbol and --to\n")
				return errCommandFailed
			}

			res, err := symbol.Resolve(workDir, file, sym)
			if err != nil {
				if errors.Is(err, symbol.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
					return errCommandFailed
				}
				fmt.Fprintf(os.Stderr, "rename resolution error: %v\n", err)
				return errCommandFailed
			}

			if res.Ambiguous {
				fmt.Fprintf(os.Stderr, "ambiguous symbol %q, please qualify receiver\n", sym)
				return errCommandFailed
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			if err := golang.Rename(ctx, workDir, res.File, res.Line, res.Column, to); err != nil {
				fmt.Fprintf(os.Stderr, "rename execution error: %v\n", err)
				return errCommandFailed
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

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&sym, "symbol", "s", "", "Target symbol identifier")
	cmd.Flags().StringVarP(&to, "to", "t", "", "New name for target symbol")
	return cmd
}

func newInsertCmd(workDir string) *cobra.Command {
	var file string
	var placement string
	var target string
	var visibility string
	var source string
	var organizeImports bool

	cmd := &cobra.Command{
		Use:           "insert",
		Short:         "Insert top-level Go declaration into file",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			source = strings.TrimSpace(source)
			if (strings.HasPrefix(source, "\"") && strings.HasSuffix(source, "\"")) ||
				(strings.HasPrefix(source, "'") && strings.HasSuffix(source, "'")) {
				source = source[1 : len(source)-1]
			}
			target = strings.Trim(strings.TrimSpace(target), `"'`)

			if file == "" || source == "" {
				fmt.Fprintf(os.Stderr, "insert requires --file and --source\n")
				return errCommandFailed
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			opts := astedit.Options{
				Placement:           astedit.Placement(placement),
				TargetSymbol:        target,
				Visibility:          visibility,
				AutoOrganizeImports: organizeImports,
			}

			if err := astedit.InsertDeclaration(ctx, targetPath, source, opts); err != nil {
				fmt.Fprintf(os.Stderr, "insert error: %v\n", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Printf("Successfully inserted declaration into %s\n", file)
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&placement, "placement", "p", "file_end", "Placement boundary (file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target symbol for before_symbol / after_symbol")
	cmd.Flags().StringVarP(&visibility, "visibility", "v", "", "Optional visibility constraint (public, private)")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Go declaration code snippet")
	cmd.Flags().BoolVar(&organizeImports, "organize-imports", true, "Automatically organize imports after insertion")
	return cmd
}

func newInsertFuncCmd(workDir string) *cobra.Command {
	var file string
	var source string
	var access string
	var placement string
	var target string
	var noImports bool

	cmd := &cobra.Command{
		Use:           "insert-func",
		Short:         "Insert function or method with access modifier and section placement",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" || source == "" {
				fmt.Fprintf(os.Stderr, "insert-func requires --file and --source\n")
				return errCommandFailed
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			opts := astedit.FunctionOptions{
				AccessModifier:      astedit.AccessModifier(access),
				Placement:           astedit.Placement(placement),
				TargetSymbol:        target,
				AutoOrganizeImports: !noImports,
			}

			if err := astedit.InsertFunction(ctx, targetPath, source, opts); err != nil {
				fmt.Fprintf(os.Stderr, "insert-func error: %v\n", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Printf("Successfully inserted function into %s\n", file)
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Function or method source code snippet")
	cmd.Flags().StringVarP(&access, "access", "a", "infer", "Access modifier (infer, public, private, protected, package-private)")
	cmd.Flags().StringVarP(&placement, "placement", "p", "", "Placement (file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target symbol for before_symbol/after_symbol placement")
	cmd.Flags().BoolVar(&noImports, "no-imports", false, "Disable automatic import resolution")
	return cmd
}

func newInsertTypeCmd(workDir string) *cobra.Command {
	var file string
	var source string
	var access string
	var placement string
	var target string
	var noImports bool

	cmd := &cobra.Command{
		Use:           "insert-type",
		Short:         "Insert type declaration with access modifier and section placement",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" || source == "" {
				fmt.Fprintf(os.Stderr, "insert-type requires --file and --source\n")
				return errCommandFailed
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			opts := astedit.TypeOptions{
				AccessModifier:      astedit.AccessModifier(access),
				Placement:           astedit.Placement(placement),
				TargetSymbol:        target,
				AutoOrganizeImports: !noImports,
			}

			if err := astedit.InsertType(ctx, targetPath, source, opts); err != nil {
				fmt.Fprintf(os.Stderr, "insert-type error: %v\n", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Printf("Successfully inserted type into %s\n", file)
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Type declaration source code snippet")
	cmd.Flags().StringVarP(&access, "access", "a", "infer", "Access modifier (infer, public, private, protected, package-private)")
	cmd.Flags().StringVarP(&placement, "placement", "p", "", "Placement (file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target symbol for before_symbol/after_symbol placement")
	cmd.Flags().BoolVar(&noImports, "no-imports", false, "Disable automatic import resolution")
	return cmd
}

func newInsertDeclCmd(workDir string) *cobra.Command {
	var file string
	var source string
	var access string
	var group string
	var placement string
	var target string
	var noImports bool

	cmd := &cobra.Command{
		Use:           "insert-decl",
		Short:         "Insert general declaration (const, var) with grouping support",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" || source == "" {
				fmt.Fprintf(os.Stderr, "insert-decl requires --file and --source\n")
				return errCommandFailed
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			opts := astedit.DeclOptions{
				AccessModifier:      astedit.AccessModifier(access),
				Group:               group,
				Placement:           astedit.Placement(placement),
				TargetSymbol:        target,
				AutoOrganizeImports: !noImports,
			}

			if err := astedit.InsertDecl(ctx, targetPath, source, opts); err != nil {
				fmt.Fprintf(os.Stderr, "insert-decl error: %v\n", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Printf("Successfully inserted declaration into %s\n", file)
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Declaration source code snippet")
	cmd.Flags().StringVarP(&access, "access", "a", "infer", "Access modifier (infer, public, private, protected, package-private)")
	cmd.Flags().StringVarP(&group, "group", "g", "append", "Group merging behavior (append, standalone)")
	cmd.Flags().StringVarP(&placement, "placement", "p", "", "Placement (file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target symbol for before_symbol/after_symbol placement")
	cmd.Flags().BoolVar(&noImports, "no-imports", false, "Disable automatic import resolution")
	return cmd
}

func newImportsCmd(workDir string) *cobra.Command {
	var file string
	var addFlags []string
	var removeFlags []string

	cmd := &cobra.Command{
		Use:           "imports",
		Short:         "Organize, sort, and reconcile import declarations",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var paths []string
			if file != "" {
				paths = []string{file}
			} else {
				paths = []string{"."}
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			opts := pipeline.ImportOptions{
				Add:    addFlags,
				Remove: removeFlags,
			}

			if err := pipeline.OrganizeImportsWithOptions(ctx, workDir, opts, paths...); err != nil {
				fmt.Fprintf(os.Stderr, "organize imports error: %v\n", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Println("Successfully organized imports.")
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path or directory (defaults to entire workspace)")
	cmd.Flags().StringArrayVar(&addFlags, "add", nil, "Explicit import path or alias to add (repeatable)")
	cmd.Flags().StringArrayVar(&removeFlags, "remove", nil, "Explicit import path to remove (repeatable)")
	return cmd
}

func newGetCmd(workDir string) *cobra.Command {
	return &cobra.Command{
		Use:           "get <package>",
		Short:         "Add external Go dependency and tidy module",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				fmt.Fprintf(os.Stderr, "get requires package name (e.g. semedit get github.com/google/uuid)\n")
				return errCommandFailed
			}

			ctx := cmd.Context()
			pkg := args[0]
			if err := golang.AddDependency(ctx, workDir, pkg); err != nil {
				fmt.Fprintf(os.Stderr, "get dependency error: %v\n", err)
				return errCommandFailed
			}

			fmt.Printf("Successfully added dependency %s\n", pkg)
			return nil
		},
	}
}

func newMCPCmd(workDir string) *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:           "mcp",
		Short:         "Start Model Context Protocol (MCP) stdio server",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := mcp.NewServer(profile, workDir, os.Stdout)
			if err := srv.Serve(cmd.Context(), os.Stdin); err != nil {
				fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
				return errCommandFailed
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "full", "MCP server profile (full, mutations-only)")
	return cmd
}

func printDelta(delta pipeline.DiagnosticDelta) {
	if len(delta.Introduced) > 0 {
		fmt.Fprintf(os.Stderr, "diagnostics introduced:\n%s\n", strings.Join(delta.Introduced, "\n"))
	}
	if len(delta.Suggestions) > 0 {
		fmt.Printf("Actionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	if len(delta.Resolved) > 0 {
		fmt.Printf("diagnostics resolved:\n%s\n", strings.Join(delta.Resolved, "\n"))
	}
}
