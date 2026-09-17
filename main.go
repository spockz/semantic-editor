// Package main serves as the entry point for semedit, coordinating LLM intent planning with deterministic AST transformations.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/mcp"
	"semedit/internal/pipeline"
	"semedit/internal/snapshot"
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
		newReplaceBodyCmd(workDir),
		newScaffoldFileCmd(workDir),
		newInsertCaseCmd(workDir),
		newSnapshotCmd(workDir),
		newUndoCmd(workDir),
	)

	return rootCmd
}

func newLookupCmd(workDir string) *cobra.Command {
	var file string
	var sym string
	var language string
	var trustWorkspace bool
	var jdtlsHome string
	var javaBin string
	var metalsHome string
	var metalsBin string
	var javaVersion string
	var haskellStandalone bool
	var ghcBin string
	var hlsBin string
	var ghcVersion string
	var hlsVersion string

	cmd := &cobra.Command{
		Use:           "lookup",
		Short:         "Resolve symbol location and AST coordinates",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if sym == "" {
				fmt.Fprintf(os.Stderr, "lookup requires --symbol\n")
				return errCommandFailed
			}

			service := backend.NewDefaultService()
			res, err := service.Lookup(cmd.Context(), backend.ProjectContext{
				RootDir:           workDir,
				File:              file,
				Language:          backend.LanguageID(language),
				WorkspaceTrust:    backend.NewWorkspaceTrust(workDir, trustWorkspace),
				Java:              backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin},
				Scala:             backend.ScalaConfig{MetalsHome: metalsHome, MetalsBin: metalsBin, JavaBin: javaBin, JavaVersion: javaVersion},
				Haskell:           backend.HaskellConfig{Standalone: haskellStandalone, GHCBin: ghcBin, HLSBin: hlsBin, GHCVersion: ghcVersion, HLSVersion: hlsVersion},
				HaskellStandalone: haskellStandalone,
			}, sym)
			if err != nil {
				if errors.Is(err, symbol.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
					return errCommandFailed
				}
				formatCLIError("lookup", err)
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
	cmd.Flags().StringVar(&language, "language", string(backend.LanguageAuto), "Language backend (auto, go, rust, java, scala, or explicit haskell standalone lookup)")
	cmd.Flags().BoolVar(&trustWorkspace, "trust-workspace", false, "Explicitly trust this workspace for future external-tool backends")
	cmd.Flags().StringVar(&jdtlsHome, "jdtls-home", "", "Preinstalled JDT LS distribution home (required for Java lookup)")
	cmd.Flags().StringVar(&javaBin, "java-bin", "", "Java 21+ executable (defaults to java on PATH)")
	cmd.Flags().StringVar(&metalsHome, "metals-home", "", "Preinstalled pinned Metals distribution home (required for Scala lookup)")
	cmd.Flags().StringVar(&metalsBin, "metals-bin", "", "Direct pinned Metals executable (alternative to --metals-home)")
	cmd.Flags().StringVar(&javaVersion, "java-version", "", "Recorded Java major version (required for Scala lookup)")
	cmd.Flags().BoolVar(&haskellStandalone, "haskell-standalone", false, "Explicitly select standalone Haskell .hs lookup (project markers are rejected)")
	cmd.Flags().StringVar(&ghcBin, "ghc-bin", "", "Preinstalled GHC executable for standalone Haskell lookup")
	cmd.Flags().StringVar(&hlsBin, "hls-bin", "", "Preinstalled haskell-language-server-wrapper executable")
	cmd.Flags().StringVar(&ghcVersion, "ghc-version", "", "Recorded GHC version to require")
	cmd.Flags().StringVar(&hlsVersion, "hls-version", "", "Recorded HLS version to require")
	return cmd
}

func newRenameCmd(workDir string) *cobra.Command {
	var file string
	var sym string
	var to string
	var language string
	var trustWorkspace bool

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

			ctx := cmd.Context()
			service := backend.NewDefaultService()
			result, err := service.Rename(ctx, backend.RenameRequest{
				Project: backend.ProjectContext{
					RootDir:        workDir,
					File:           file,
					Language:       backend.LanguageID(language),
					WorkspaceTrust: backend.NewWorkspaceTrust(workDir, trustWorkspace),
				},
				Symbol:          sym,
				To:              to,
				OrganizeImports: false,
			})
			if err != nil {
				switch {
				case errors.Is(err, symbol.ErrNotFound):
					fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
				case errors.Is(err, backend.ErrAmbiguous):
					fmt.Fprintf(os.Stderr, "ambiguous symbol %q, please qualify receiver\n", sym)
				default:
					if _, ok := errors.AsType[*symbol.SymbolError](err); ok {
						formatCLIError("rename resolution", err)
					} else {
						formatCLIError("rename execution", err)
					}
				}
				return errCommandFailed
			}

			delta := result.Diagnostics

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
	cmd.Flags().StringVar(&language, "language", string(backend.LanguageAuto), "Language backend (auto, go)")
	cmd.Flags().BoolVar(&trustWorkspace, "trust-workspace", false, "Explicitly trust this workspace for future external-tool backends")
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
				formatCLIError("insert", err)
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
				formatCLIError("insert-func", err)
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
				formatCLIError("insert-type", err)
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
				formatCLIError("insert-decl", err)
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
				formatCLIError("organize imports", err)
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
				formatCLIError("get dependency", err)
				return errCommandFailed
			}

			fmt.Printf("Successfully added dependency %s\n", pkg)
			return nil
		},
	}
}

func newMCPCmd(workDir string) *cobra.Command {
	var profile string
	var liveReload bool

	cmd := &cobra.Command{
		Use:           "mcp",
		Short:         "Start Model Context Protocol (MCP) stdio server",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := mcp.NewServer(profile, workDir, os.Stdout, mcp.WithLiveReload(liveReload))
			if err := srv.Serve(cmd.Context(), os.Stdin); err != nil {
				formatCLIError("mcp server", err)
				return errCommandFailed
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "full", "MCP server profile (full, mutations-only)")
	cmd.Flags().BoolVar(&liveReload, "live-reload", false, "Enable in-place live-reload and dynamic tool schema discovery")
	return cmd
}

func newReplaceBodyCmd(workDir string) *cobra.Command {
	var file string
	var sym string
	var bodyFlag string
	var autoImports bool

	cmd := &cobra.Command{
		Use:           "replace-body",
		Short:         "Replace the body of an existing Go function or method by name",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" || sym == "" {
				fmt.Fprintf(os.Stderr, "replace-body requires --file and --symbol\n")
				return errCommandFailed
			}

			body := bodyFlag
			if body == "" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
					return errCommandFailed
				}
				body = string(data)
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			diff, err := astedit.ReplaceBody(ctx, targetPath, sym, body, astedit.BodyOptions{
				AutoOrganizeImports: autoImports,
			})
			if err != nil {
				formatCLIError("replace-body", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Printf("Successfully replaced body of %s in %s\n", sym, file)
			if diff != "" {
				fmt.Print(diff)
			}
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&sym, "symbol", "s", "", "Target symbol identifier (e.g. 'Foo' or '(*Server).Start')")
	cmd.Flags().StringVarP(&bodyFlag, "body", "b", "", "Replacement body as bare Go statements")
	cmd.Flags().BoolVar(&autoImports, "auto-imports", false, "Automatically organize imports after replacement")
	return cmd
}

func newScaffoldFileCmd(workDir string) *cobra.Command {
	var file string
	var pkg string
	var overwrite bool
	var autoImports bool

	cmd := &cobra.Command{
		Use:           "scaffold-file",
		Short:         "Scaffold a new Go source file with package declaration",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" {
				fmt.Fprintf(os.Stderr, "scaffold-file requires --file\n")
				return errCommandFailed
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			resolvedPkg, err := astedit.ScaffoldFile(cmd.Context(), targetPath, pkg, astedit.ScaffoldOptions{
				Overwrite:           overwrite,
				AutoOrganizeImports: autoImports,
			})
			if err != nil {
				formatCLIError("scaffold-file", err)
				return errCommandFailed
			}

			fmt.Printf("Successfully scaffolded %s with package %s\n", file, resolvedPkg)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVarP(&pkg, "package", "p", "infer", "Package name or 'infer' (default 'infer')")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite file if it already exists")
	cmd.Flags().BoolVar(&autoImports, "auto-imports", false, "Accepted for schema uniformity")
	return cmd
}

func newInsertCaseCmd(workDir string) *cobra.Command {
	var file string
	var fn string
	var switchOn string
	var caseFlag string
	var placement string
	var anchor string
	var autoImports bool

	cmd := &cobra.Command{
		Use:           "insert-case",
		Short:         "Insert a case clause into an existing switch statement",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" || fn == "" {
				fmt.Fprintf(os.Stderr, "insert-case requires --file and --func\n")
				return errCommandFailed
			}

			caseSrc := caseFlag
			if caseSrc == "" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
					return errCommandFailed
				}
				caseSrc = string(data)
			}

			if strings.TrimSpace(caseSrc) == "" {
				fmt.Fprintf(os.Stderr, "insert-case requires case source via --case or stdin\n")
				return errCommandFailed
			}

			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(workDir, targetPath)
			}

			ctx := cmd.Context()
			diagsBefore, _ := pipeline.CheckDiagnostics(ctx, workDir)

			diff, err := astedit.InsertCase(ctx, targetPath, fn, switchOn, caseSrc, astedit.CaseOptions{
				Placement:           astedit.CasePlacement(placement),
				AnchorCase:          anchor,
				AutoOrganizeImports: autoImports,
			})
			if err != nil {
				formatCLIError("insert-case", err)
				return errCommandFailed
			}

			diagsAfter, _ := pipeline.CheckDiagnostics(ctx, workDir)
			delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

			fmt.Printf("Successfully inserted case into %s in %s\n", fn, file)
			if diff != "" {
				fmt.Print(diff)
			}
			printDelta(delta)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Target file path")
	cmd.Flags().StringVar(&fn, "func", "", "Name of function containing the switch")
	cmd.Flags().StringVar(&switchOn, "switch-on", "", "Discriminant expression (omit for tagless switch)")
	cmd.Flags().StringVar(&caseFlag, "case", "", "Case clause Go source code")
	cmd.Flags().StringVarP(&placement, "placement", "p", "before_default", "Placement (first, last, before_default, before, after)")
	cmd.Flags().StringVar(&anchor, "anchor", "", "Anchor case value for before/after placement")
	cmd.Flags().BoolVar(&autoImports, "auto-imports", false, "Automatically organize imports after insertion")
	return cmd
}

func newSnapshotCmd(workDir string) *cobra.Command {
	var label string
	var desc string
	var paths []string
	var recordPost string

	cmd := &cobra.Command{
		Use:           "snapshot [paths...]",
		Short:         "Capture pre-edit state into transactional content-addressed snapshot journal",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if recordPost != "" {
				manifest, err := snapshot.RecordPostEdit(cmd.Context(), workDir, recordPost)
				if err != nil {
					fmt.Fprintf(os.Stderr, "record-post error: %v\n", err)
					return errCommandFailed
				}
				data, err := json.MarshalIndent(map[string]any{
					"status":      "ok",
					"snapshot_id": manifest.ID,
					"files_count": len(manifest.Files),
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			allPaths := append([]string(nil), paths...)
			allPaths = append(allPaths, args...)
			res, err := snapshot.Create(cmd.Context(), workDir, snapshot.CreateOptions{
				Label:       label,
				Description: desc,
				Paths:       allPaths,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "snapshot error: %v\n", err)
				return errCommandFailed
			}

			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&label, "label", "l", "snapshot", "Human-readable label for the snapshot")
	cmd.Flags().StringVarP(&desc, "desc", "d", "", "Description of pending change or purpose")
	cmd.Flags().StringSliceVarP(&paths, "paths", "p", nil, "Paths to include in snapshot")
	cmd.Flags().StringVar(&recordPost, "record-post", "", "Snapshot ID to record post-edit hashes for")
	return cmd
}

func newUndoCmd(workDir string) *cobra.Command {
	var idFlag string

	cmd := &cobra.Command{
		Use:           "undo [snapshot-id]",
		Short:         "Roll back workspace files to a snapshot state with conflict checking",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetID := idFlag
			if len(args) > 0 && args[0] != "" {
				targetID = args[0]
			}
			if targetID == "" {
				fmt.Fprintf(os.Stderr, "undo requires snapshot ID (as argument or --id)\n")
				return errCommandFailed
			}

			res, err := snapshot.Undo(cmd.Context(), workDir, targetID)
			if err != nil {
				if errors.Is(err, snapshot.ErrConflict) {
					fmt.Fprintf(os.Stderr, "conflict error: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "undo error: %v\n", err)
				}
				return errCommandFailed
			}

			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&idFlag, "id", "", "Snapshot ID to restore")
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

func formatCLIError(op string, err error) {
	var pos token.Position
	var synErr *astedit.SyntaxError
	var symErr *symbol.SymbolError
	var visErr *astedit.VisibilityMismatchError
	var placeErr *astedit.PlacementError

	switch {
	case errors.As(err, &synErr) && synErr.Pos.IsValid():
		pos = synErr.Pos
	case errors.As(err, &symErr) && symErr.Pos.IsValid():
		pos = symErr.Pos
	case errors.As(err, &visErr) && visErr.Pos.IsValid():
		pos = visErr.Pos
	case errors.As(err, &placeErr) && placeErr.Pos.IsValid():
		pos = placeErr.Pos
	}

	if pos.IsValid() {
		msg := err.Error()
		prefix := pos.String() + ": "
		if after, ok := strings.CutPrefix(msg, prefix); ok {
			msg = after
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", pos.String(), msg)
	} else {
		fmt.Fprintf(os.Stderr, "%s error: %v\n", op, err)
	}
}
