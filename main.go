// Package main is the executable boundary: semantic commands come from the shared operation registry.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"semedit/internal/cli"
	"semedit/internal/mcp"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	rootCmd := newRootCmd(workDir)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, cli.ErrCommandFailed) {
			fmt.Fprintln(os.Stderr, err)
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	rootCmd.AddCommand(cli.Commands(workDir)...)
	rootCmd.AddCommand(cli.IntegrationCommands(workDir)...)
	rootCmd.AddCommand(cli.IntegrationGroup(workDir))
	rootCmd.AddCommand(newMCPCmd(workDir))
	return rootCmd
}

func newMCPCmd(workDir string) *cobra.Command {
	var profile string
	var toolProfile string
	var exposeToolVariants []string
	var liveReload bool
	var instructions string
	var goBaseDir string
	var expectedRoot string
	var enabledLanguages string
	var metricsFile string
	cmd := &cobra.Command{
		Use:           "mcp",
		Short:         "Start Model Context Protocol (MCP) stdio server",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := mcp.NewServer(profile, workDir, os.Stdout, mcp.WithToolProfile(toolProfile), mcp.WithToolVariants(exposeToolVariants), mcp.WithLiveReload(liveReload), mcp.WithInstructions(instructions), mcp.WithGoBaseDir(goBaseDir), mcp.WithExpectedRoot(expectedRoot), mcp.WithEnabledLanguages(enabledLanguages), mcp.WithMetricsFile(metricsFile))
			if err := srv.Serve(cmd.Context(), os.Stdin); err != nil {
				fmt.Fprintf(os.Stderr, "mcp server error: %v\\n", err)
				return cli.ErrCommandFailed
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "full", "MCP server profile (full, mutations-only)")
	cmd.Flags().StringVar(&toolProfile, "tool-profile", "base", "MCP tool catalog profile (base or full)")
	cmd.Flags().StringArrayVar(&exposeToolVariants, "expose-tool-variant", nil, "Expose specialized MCP tool variants by verb:kind; repeat or use comma-separated patterns")
	cmd.Flags().BoolVar(&liveReload, "live-reload", false, "Enable in-place live-reload and dynamic tool schema discovery")
	cmd.Flags().StringVar(&instructions, "instructions", mcp.PrescriptiveInstructions, "Server-wide MCP instructions returned during initialization; empty disables guidance")
	cmd.Flags().StringVar(&goBaseDir, "go-base-dir", "", "Directory for Go subprocess state; defaults to .scratch/go below the server working directory")
	cmd.Flags().StringVar(&expectedRoot, "expected-root", "", "Require the client-selected workspace root to match this canonical root")
	cmd.Flags().StringVar(&enabledLanguages, "enabled-languages", "", "Comma-separated language allowlist for MCP tools (go,rust,java,scala,haskell,kotlin,bash,make)")
	cmd.Flags().StringVar(&metricsFile, "metrics-file", "", "Persist all MCP tool timings to this JSON file")
	return cmd
}
