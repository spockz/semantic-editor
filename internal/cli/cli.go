// Package cli builds a uniform Cobra surface directly from registered semantic operations.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"semedit/internal/backend"
	"semedit/internal/integration"
	"semedit/internal/operation"
)

// ErrCommandFailed marks expected command failures without printing Cobra usage.
var ErrCommandFailed = errors.New("command execution failed")

// flagValues holds one variable per registered flag, keyed by CLI flag name.
type flagValues struct {
	strings map[string]*string
	bools   map[string]*bool
	slices  map[string]*[]string
}

func stringDefault(value any) string {
	text, _ := value.(string)
	return text
}

func boolDefault(value any) bool {
	enabled, _ := value.(bool)
	return enabled
}

// registerFlags creates one long flag per parameter contract entry.
func registerFlags(cmd *cobra.Command, params []operation.ParameterContract, values *flagValues) {
	values.strings = map[string]*string{}
	values.bools = map[string]*bool{}
	values.slices = map[string]*[]string{}
	for _, param := range params {
		if param.CLIName == "" {
			continue
		}
		switch param.Type {
		case operation.ParamBoolean:
			values.bools[param.CLIName] = cmd.Flags().Bool(param.CLIName, boolDefault(param.Default), param.Description)
		case operation.ParamStringSlice:
			values.slices[param.CLIName] = cmd.Flags().StringArray(param.CLIName, nil, param.Description)
		default:
			values.strings[param.CLIName] = cmd.Flags().String(param.CLIName, stringDefault(param.Default), param.Description)
		}
	}
}

// rawParams translates flag values to the JSONName-keyed raw map consumed by Parse.
func rawParams(entry operation.Entry, values *flagValues) map[string]any {
	raw := make(map[string]any, len(entry.Params))
	for _, param := range entry.Params {
		if param.CLIName == "" {
			continue
		}
		switch param.Type {
		case operation.ParamBoolean:
			if variable, ok := values.bools[param.CLIName]; ok {
				raw[param.JSONName] = *variable
			}
		case operation.ParamStringSlice:
			if variable, ok := values.slices[param.CLIName]; ok && len(*variable) > 0 {
				raw[param.JSONName] = append([]string(nil), *variable...)
			}
		default:
			if variable, ok := values.strings[param.CLIName]; ok && *variable != "" {
				raw[param.JSONName] = *variable
			}
		}
	}
	return raw
}

// Commands builds one Cobra command for every registry operation with a CLI name.
func Commands(workDir string) []*cobra.Command {
	registry := operation.DefaultRegistry()
	entries := registry.All()
	commands := make([]*cobra.Command, 0, len(entries))
	for _, entry := range entries {
		if entry.CLIName == "" {
			continue
		}
		commands = append(commands, buildCommand(workDir, registry, entry))
	}
	return commands
}

// IntegrationCommands exposes target-native MCP installation separately from
// semantic operations because it changes harness configuration, not source code.
func IntegrationCommands(workDir string) []*cobra.Command {
	return []*cobra.Command{newInstallCommand(workDir), newIntegrationStatusCommand(workDir), newUninstallCommand(workDir)}
}

// IntegrationGroup provides the candidate `integration status TARGET` spelling
// while retaining the shorter top-level status command for shell ergonomics.
func IntegrationGroup(workDir string) *cobra.Command {
	group := &cobra.Command{Use: "integration", Short: "Inspect native semedit harness integrations", Args: cobra.NoArgs}
	group.AddCommand(newIntegrationStatusCommand(workDir))
	return group
}

func targetArgs(args []string) (integration.Target, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("target must be copilot or codex")
	}
	switch args[0] {
	case string(integration.TargetCopilot):
		return integration.TargetCopilot, nil
	case string(integration.TargetCodex):
		return integration.TargetCodex, nil
	default:
		return "", fmt.Errorf("unsupported integration target %q", args[0])
	}
}

func integrationOutput(cmd *cobra.Command, result integration.Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func newInstallCommand(workDir string) *cobra.Command {
	cmd := &cobra.Command{Use: "install", Short: "Install semedit into a native harness configuration", Args: cobra.NoArgs}
	for _, target := range []integration.Target{integration.TargetCopilot, integration.TargetCodex} {
		var scope, binary, profile, workspace string
		var dryRun, replace bool
		targetCmd := &cobra.Command{
			Use:   string(target),
			Short: fmt.Sprintf("Install semedit MCP for %s", target),
			Args:  cobra.NoArgs,
			RunE: func(target integration.Target, scope, binary, profile, workspace *string, dryRun, replace *bool) func(*cobra.Command, []string) error {
				return func(cmd *cobra.Command, _ []string) error {
					if strings.TrimSpace(*scope) == "" {
						return fmt.Errorf("--scope is required (user or workspace)")
					}
					if *workspace == "" {
						*workspace = workDir
					}
					result, err := integration.Install(integration.Request{Target: target, Scope: integration.Scope(*scope), Workspace: filepath.Clean(*workspace), Binary: *binary, Profile: *profile, DryRun: *dryRun, Replace: *replace})
					if err != nil {
						return err
					}
					return integrationOutput(cmd, result)
				}
			}(target, &scope, &binary, &profile, &workspace, &dryRun, &replace),
		}
		targetCmd.Flags().StringVar(&scope, "scope", "", "Registration scope: user or workspace (required)")
		targetCmd.Flags().StringVar(&workspace, "workspace", workDir, "Workspace root for workspace scope")
		targetCmd.Flags().StringVar(&binary, "binary", "", "Absolute path to an already installed semedit executable")
		targetCmd.Flags().StringVar(&profile, "profile", "full", "MCP profile: full or mutations-only")
		targetCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report the change without writing")
		targetCmd.Flags().BoolVar(&replace, "replace", false, "Explicitly replace a conflicting semedit registration")
		cmd.AddCommand(targetCmd)
	}
	return cmd
}

func newIntegrationStatusCommand(workDir string) *cobra.Command {
	var scope, workspace string
	cmd := &cobra.Command{Use: "status TARGET", Short: "Report native semedit harness integration status", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		target, err := targetArgs(args)
		if err != nil {
			return err
		}
		if workspace == "" {
			workspace = workDir
		}
		if scope != "" {
			result, err := integration.Status(integration.StatusRequest{Target: target, Scope: integration.Scope(scope), Workspace: workspace})
			if err != nil {
				return err
			}
			return integrationOutput(cmd, result)
		}
		results := make([]integration.Result, 0, 2)
		for _, selectedScope := range []integration.Scope{integration.ScopeWorkspace, integration.ScopeUser} {
			result, err := integration.Status(integration.StatusRequest{Target: target, Scope: selectedScope, Workspace: workspace})
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		data, err := json.MarshalIndent(map[string]any{"target": target, "statuses": results}, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return err
	}}
	cmd.Flags().StringVar(&scope, "scope", "", "Limit status to user or workspace")
	cmd.Flags().StringVar(&workspace, "workspace", workDir, "Workspace root for workspace status")
	return cmd
}

func newUninstallCommand(workDir string) *cobra.Command {
	var scope, workspace string
	cmd := &cobra.Command{Use: "uninstall TARGET", Short: "Remove only an owned native semedit registration", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		target, err := targetArgs(args)
		if err != nil {
			return err
		}
		if scope == "" {
			return fmt.Errorf("--scope is required (user or workspace)")
		}
		if workspace == "" {
			workspace = workDir
		}
		result, err := integration.Uninstall(integration.StatusRequest{Target: target, Scope: integration.Scope(scope), Workspace: workspace})
		if err != nil {
			return err
		}
		return integrationOutput(cmd, result)
	}}
	cmd.Flags().StringVar(&scope, "scope", "", "Registration scope: user or workspace (required)")
	cmd.Flags().StringVar(&workspace, "workspace", workDir, "Workspace root for workspace scope")
	return cmd
}

func buildCommand(workDir string, registry *operation.Registry, entry operation.Entry) *cobra.Command {
	values := &flagValues{}
	cmd := &cobra.Command{
		Use:           entry.CLIName,
		Short:         entry.Summary,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cc := operation.NewCallContext(workDir, backend.ProjectContext{})
			cc.Ctx = cmd.Context()
			result, err := registry.Dispatch(cc, entry.Key, rawParams(entry, values))
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", entry.CLIName, err)
				return ErrCommandFailed
			}
			text, err := entry.Format(result)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: format result: %v\n", entry.CLIName, err)
				return ErrCommandFailed
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), text); err != nil {
				return fmt.Errorf("write %s result: %w", entry.CLIName, err)
			}
			return nil
		},
	}
	registerFlags(cmd, entry.Params, values)
	return cmd
}
