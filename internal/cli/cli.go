// Package cli builds a uniform Cobra surface directly from registered semantic operations.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"semedit/internal/backend"
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
