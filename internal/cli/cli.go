// Package cli builds a uniform Cobra surface directly from registered semantic operations.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"semedit/internal/backend"
	"semedit/internal/integration"
	"semedit/internal/maven"
	"semedit/internal/operation"
)

// ErrCommandFailed marks expected command failures without printing Cobra usage.
var ErrCommandFailed = errors.New("command execution failed")

// flagValues holds one variable per registered flag, keyed by CLI flag name.
type flagValues struct {
	strings map[string]*string
	bools   map[string]*bool
	slices  map[string]*[]string
	ints    map[string]*int
	json    map[string]*string
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
	values.ints = map[string]*int{}
	values.json = map[string]*string{}
	for _, param := range params {
		if param.CLIName == "" {
			continue
		}
		description := param.Description
		if isStdinPayloadParameter(param) {
			description += " Use - to read this payload from stdin."
		}
		switch param.Type {
		case operation.ParamObject, operation.ParamArray:
			values.json[param.CLIName] = cmd.Flags().String(param.CLIName, "", description+" (JSON value)")
		case operation.ParamBoolean:
			values.bools[param.CLIName] = cmd.Flags().Bool(param.CLIName, boolDefault(param.Default), description)
		case operation.ParamStringSlice:
			values.slices[param.CLIName] = cmd.Flags().StringArray(param.CLIName, nil, description)
		case operation.ParamInteger:
			values.ints[param.CLIName] = cmd.Flags().Int(param.CLIName, 0, description)
		default:
			values.strings[param.CLIName] = cmd.Flags().String(param.CLIName, stringDefault(param.Default), description)
		}
		for _, alias := range param.InputAliases {
			if alias == "" || alias == param.CLIName {
				continue
			}
			switch param.Type {
			case operation.ParamObject, operation.ParamArray:
				values.json[alias] = cmd.Flags().String(alias, "", "")
			case operation.ParamBoolean:
				values.bools[alias] = cmd.Flags().Bool(alias, false, "")
			case operation.ParamStringSlice:
				values.slices[alias] = cmd.Flags().StringArray(alias, nil, "")
			case operation.ParamInteger:
				values.ints[alias] = cmd.Flags().Int(alias, 0, "")
			default:
				values.strings[alias] = cmd.Flags().String(alias, "", "")
			}
			cmd.Flags().Lookup(alias).Hidden = true
		}
	}
}

// rawParams translates flag values to the JSONName-keyed raw map consumed by Parse.
func rawParams(entry operation.Entry, values *flagValues, flagChanged func(string) bool) (map[string]any, error) {
	raw := make(map[string]any, len(entry.Params))
	for _, param := range entry.Params {
		if param.CLIName != "" {
			value, include, err := rawParamValue(param, values, flagChanged)
			if err != nil {
				return nil, err
			}
			if include {
				raw[param.JSONName] = value
			}
		}
		for _, alias := range param.InputAliases {
			if alias == "" || alias == param.CLIName {
				continue
			}
			aliasParam := param
			aliasParam.CLIName = alias
			value, include, err := rawParamValue(aliasParam, values, flagChanged)
			if err != nil {
				return nil, err
			}
			if include {
				raw[alias] = value
			}
		}
	}
	return raw, nil
}

func rawParamValue(param operation.ParameterContract, values *flagValues, changed func(string) bool) (any, bool, error) {
	switch param.Type {
	case operation.ParamObject, operation.ParamArray:
		return rawJSONParam(param, values, changed)
	case operation.ParamBoolean:
		value, exists := values.bools[param.CLIName]
		if !exists {
			return nil, false, nil
		}
		return *value, true, nil
	case operation.ParamStringSlice:
		value, exists := values.slices[param.CLIName]
		if !exists || len(*value) == 0 {
			return nil, false, nil
		}
		return append([]string(nil), (*value)...), true, nil
	case operation.ParamInteger:
		value, exists := values.ints[param.CLIName]
		if !exists || *value == 0 {
			return nil, false, nil
		}
		return *value, true, nil
	default:
		value, exists := values.strings[param.CLIName]
		if !exists || *value == "" && !isChanged(changed, param.CLIName) {
			return nil, false, nil
		}
		return *value, true, nil
	}
}

func rawJSONParam(param operation.ParameterContract, values *flagValues, changed func(string) bool) (any, bool, error) {
	value, exists := values.json[param.CLIName]
	if !exists || *value == "" && !isChanged(changed, param.CLIName) {
		return nil, false, nil
	}
	decoder := json.NewDecoder(strings.NewReader(*value))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false, fmt.Errorf("decode --%s JSON value: %w", param.CLIName, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, false, fmt.Errorf("decode --%s JSON value: multiple values", param.CLIName)
		}
		return nil, false, fmt.Errorf("decode --%s JSON value trailing data: %w", param.CLIName, err)
	}
	return decoded, true, nil
}

func isChanged(changed func(string) bool, name string) bool {
	return changed != nil && changed(name)
}

func writeMavenFailure(stderr io.Writer, err error) (bool, error) {
	var mavenErr *maven.Error
	if !errors.As(err, &mavenErr) || mavenErr.MavenResult() == nil {
		return false, nil
	}
	payload := struct {
		Error  string        `json:"error"`
		Result *maven.Result `json:"result"`
	}{Error: err.Error(), Result: mavenErr.MavenResult()}
	data, marshalErr := json.MarshalIndent(payload, "", "  ")
	if marshalErr != nil {
		return false, fmt.Errorf("marshal Maven failure: %w", marshalErr)
	}
	if _, writeErr := fmt.Fprintln(stderr, string(data)); writeErr != nil {
		return true, fmt.Errorf("write Maven failure: %w", writeErr)
	}
	return true, nil
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

func integrationJSON(cmd *cobra.Command, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal integration result: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
		return fmt.Errorf("write integration JSON: %w", err)
	}
	return nil
}

func integrationHuman(cmd *cobra.Command, result integration.Result, includeBinaryAvailability bool) error {
	var message string
	switch result.Status {
	case "installed":
		message = fmt.Sprintf("Installed semedit for %s (%s). Config: %s", result.Target, result.Scope, result.Path)
	case "already-installed":
		message = fmt.Sprintf("semedit is already installed for %s (%s). Config: %s", result.Target, result.Scope, result.Path)
	case "dry-run":
		message = fmt.Sprintf("Preview only: semedit would be installed for %s (%s). Nothing was written to %s; rerun without --dry-run to install.", result.Target, result.Scope, result.Path)
	case "not-installed":
		message = fmt.Sprintf("semedit is not installed for %s (%s). Config: %s", result.Target, result.Scope, result.Path)
	case "uninstalled":
		message = fmt.Sprintf("Uninstalled semedit for %s (%s). Config: %s", result.Target, result.Scope, result.Path)
	case "not-owned":
		message = fmt.Sprintf("No owned semedit registration was removed for %s (%s). Config: %s", result.Target, result.Scope, result.Path)
	default:
		message = fmt.Sprintf("Integration status for %s (%s): %s. Config: %s", result.Target, result.Scope, result.Status, result.Path)
	}
	if result.Detail != "" {
		message += ". Detail: " + result.Detail
	}
	if includeBinaryAvailability && result.Status == "installed" && result.Binary != "" && !result.BinaryExists {
		message += ". Configured executable is unavailable: " + result.Binary
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), message); err != nil {
		return fmt.Errorf("write integration status: %w", err)
	}
	return nil
}

func integrationOutput(cmd *cobra.Command, result integration.Result, jsonOutput, install bool) error {
	if jsonOutput {
		return integrationJSON(cmd, result)
	}
	if err := integrationHuman(cmd, result, false); err != nil {
		return err
	}
	if install && (result.Status == "installed" || result.Status == "already-installed") {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Restart the %s coding harness or reconnect its MCP session to load semedit tools.\n", result.Target); err != nil {
			return fmt.Errorf("write install next steps: %w", err)
		}
	}
	return nil
}

func newInstallCommand(workDir string) *cobra.Command {
	const installNextSteps = `On success, the default output shows the saved config path. With --json, the path appears in the JSON; you do not need to copy or edit it. Restart the target coding harness or reconnect its MCP session to load semedit tools. A --dry-run is only a preview and writes nothing; rerun without --dry-run to install. The skill field reports optional steering-skill availability; "not-installed" does not mean MCP installation failed.`
	cmd := &cobra.Command{Use: "install", Short: "Install semedit into a native harness configuration", Args: cobra.NoArgs}
	cmd.Long = `Install semedit into a native harness configuration. Human-readable output is the default; use --json for machine-readable output on stdout.

Choose --scope user or --scope workspace. Pass --binary as the absolute path to an already installed semedit executable.

Examples:
  semedit install copilot --scope user --binary "$(command -v semedit)"
  semedit install copilot --scope workspace --binary "$(command -v semedit)"
  semedit install codex --scope user --binary "$(command -v semedit)"
  semedit install codex --scope workspace --binary "$(command -v semedit)"

Run "semedit install TARGET --help" for target-specific flags.` + "\n\n" + installNextSteps
	for _, target := range []integration.Target{integration.TargetCopilot, integration.TargetCodex} {
		var scope, binary, profile, workspace string
		var dryRun, replace, jsonOutput bool
		targetCmd := &cobra.Command{
			Use:   string(target),
			Short: fmt.Sprintf("Install semedit MCP for %s", target),
			Long:  installNextSteps,
			Args:  cobra.NoArgs,
			RunE: func(target integration.Target, scope, binary, profile, workspace *string, dryRun, replace, jsonOutput *bool) func(*cobra.Command, []string) error {
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
					return integrationOutput(cmd, result, *jsonOutput, true)
				}
			}(target, &scope, &binary, &profile, &workspace, &dryRun, &replace, &jsonOutput),
		}
		targetCmd.Flags().StringVar(&scope, "scope", "", "Registration scope: user or workspace (required)")
		targetCmd.Flags().StringVar(&workspace, "workspace", workDir, "Workspace root for workspace scope")
		targetCmd.Flags().StringVar(&binary, "binary", "", "Required: absolute path to an already installed semedit executable")
		targetCmd.Flags().StringVar(&profile, "profile", "full", "MCP profile: full or mutations-only")
		targetCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report the change without writing")
		targetCmd.Flags().BoolVar(&replace, "replace", false, "Explicitly replace a conflicting semedit registration")
		targetCmd.Flags().BoolVar(&jsonOutput, "json", false, "Write structured JSON to stdout")
		cmd.AddCommand(targetCmd)
	}
	return cmd
}

func newIntegrationStatusCommand(workDir string) *cobra.Command {
	var scope, workspace string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "status TARGET", Short: "Report native semedit harness integration status", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		target, targetErr := targetArgs(args)
		if targetErr != nil {
			return targetErr
		}
		if workspace == "" {
			workspace = workDir
		}
		if scope != "" {
			result, err := integration.Status(integration.StatusRequest{Target: target, Scope: integration.Scope(scope), Workspace: workspace})
			if err != nil {
				return err
			}
			if jsonOutput {
				return integrationJSON(cmd, result)
			}
			return integrationHuman(cmd, result, true)
		}
		results := make([]integration.Result, 0, 2)
		for _, selectedScope := range []integration.Scope{integration.ScopeWorkspace, integration.ScopeUser} {
			result, err := integration.Status(integration.StatusRequest{Target: target, Scope: selectedScope, Workspace: workspace})
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		if jsonOutput {
			return integrationJSON(cmd, map[string]any{"target": target, "statuses": results})
		}
		for _, result := range results {
			if err := integrationHuman(cmd, result, true); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&scope, "scope", "", "Limit status to user or workspace")
	cmd.Flags().StringVar(&workspace, "workspace", workDir, "Workspace root for workspace status")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write structured JSON to stdout")
	return cmd
}

func newUninstallCommand(workDir string) *cobra.Command {
	var scope, workspace string
	var jsonOutput bool
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
		return integrationOutput(cmd, result, jsonOutput, false)
	}}
	cmd.Flags().StringVar(&scope, "scope", "", "Registration scope: user or workspace (required)")
	cmd.Flags().StringVar(&workspace, "workspace", workDir, "Workspace root for workspace scope")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write structured JSON to stdout")
	return cmd
}

func buildCommand(workDir string, registry *operation.Registry, entry operation.Entry) *cobra.Command {
	values := &flagValues{}
	cmd := &cobra.Command{
		Use:           entry.CLIName,
		Short:         entry.Summary,
		Long:          entry.Description(),
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommand(cmd, workDir, registry, entry, values)
		},
	}
	registerFlags(cmd, entry.Params, values)
	return cmd
}

func executeCommand(cmd *cobra.Command, workDir string, registry *operation.Registry, entry operation.Entry, values *flagValues) error {
	cc := operation.NewCallContext(workDir, backend.ProjectContext{})
	cc.Ctx = cmd.Context()
	raw, err := rawParams(entry, values, func(name string) bool { return cmd.Flags().Changed(name) })
	if err != nil {
		return err
	}
	err = expandStdinPayloads(entry, raw, cmd.InOrStdin())
	if err != nil {
		return err
	}
	result, err := registry.Dispatch(cc, entry.Key, raw)
	if err != nil {
		return reportCommandFailure(cmd, entry, err)
	}
	return writeCommandResult(cmd, entry, result)
}

func reportCommandFailure(cmd *cobra.Command, entry operation.Entry, operationErr error) error {
	written, err := writeMavenFailure(cmd.ErrOrStderr(), operationErr)
	if err != nil {
		return err
	}
	if written {
		return ErrCommandFailed
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", entry.CLIName, operationErr); err != nil {
		return fmt.Errorf("write %s error: %w", entry.CLIName, err)
	}
	return ErrCommandFailed
}

func writeCommandResult(cmd *cobra.Command, entry operation.Entry, result any) error {
	text, err := entry.Format(result)
	if err != nil {
		if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "%s: format result: %v\n", entry.CLIName, err); writeErr != nil {
			return fmt.Errorf("write %s format error: %w", entry.CLIName, writeErr)
		}
		return ErrCommandFailed
	}
	if verify, ok := result.(operation.VerifyRes); ok && verify.Failure != "" {
		return writeVerificationFailure(cmd, entry, text, verify.Failure)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), text); err != nil {
		return fmt.Errorf("write %s result: %w", entry.CLIName, err)
	}
	return nil
}

func writeVerificationFailure(cmd *cobra.Command, entry operation.Entry, result, failure string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), result); err != nil {
		return fmt.Errorf("write %s result: %w", entry.CLIName, err)
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", entry.CLIName, failure); err != nil {
		return fmt.Errorf("write %s verification error: %w", entry.CLIName, err)
	}
	return ErrCommandFailed
}

func isStdinPayloadParameter(param operation.ParameterContract) bool {
	return param.Type == operation.ParamString && (param.JSONName == "source" || param.JSONName == "body" || param.JSONName == "case")
}

func expandStdinPayloads(entry operation.Entry, raw map[string]any, input io.Reader) error {
	flagNames := make([]string, 0, 2)
	var consumer operation.ParameterContract
	for _, param := range entry.Params {
		if !isStdinPayloadParameter(param) {
			continue
		}
		value, ok := raw[param.JSONName].(string)
		if ok && value == "-" {
			flagNames = append(flagNames, "--"+param.CLIName)
			consumer = param
		}
	}
	if len(flagNames) > 1 {
		return fmt.Errorf("multiple stdin payload flags %s are ambiguous", strings.Join(flagNames, " and "))
	}
	if len(flagNames) == 0 {
		return nil
	}
	payload, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("read --%s from stdin: %w", consumer.CLIName, err)
	}
	raw[consumer.JSONName] = string(payload)
	return nil
}
