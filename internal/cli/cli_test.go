// Package cli_test verifies registry-derived command generation without invoking language backends.
package cli_test

import (
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/cli"
	"github.com/spockz/semantic-editor/internal/operation"

	"github.com/spf13/cobra"
)

func TestCommandsCoverRegistryVerbs(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	commands := cli.Commands(t.TempDir())
	commandsByVerb := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		verb := cmd.Name()
		if commandsByVerb[verb] {
			t.Errorf("duplicate CLI verb %q", verb)
		}
		commandsByVerb[verb] = true
		entry, ok := registry.LookupCLI(verb)
		if !ok {
			t.Errorf("verb %q has no registry operation", verb)
			continue
		}
		if cmd.Use != entry.CLIName {
			t.Errorf("command %q Use = %q, want registry CLI name %q", verb, cmd.Use, entry.CLIName)
		}
		if cmd.Short != entry.Summary {
			t.Errorf("command %q Short does not use registry summary", verb)
		}
		if err := cmd.Args(cmd, []string{"unexpected"}); err == nil {
			t.Errorf("command %q accepts positional arguments", verb)
		}
		for _, param := range entry.Params {
			if param.CLIName != "" && cmd.Flags().Lookup(param.CLIName) == nil {
				t.Errorf("command %q is missing registry flag --%s", verb, param.CLIName)
			}
		}
	}
	for _, entry := range registry.All() {
		if entry.CLIName == "" {
			continue
		}
		if !commandsByVerb[entry.CLIName] {
			t.Errorf("registry CLI name %q has no generated command", entry.CLIName)
		}
	}
}

func TestRegisteredPayloadFlagsAdvertiseStdin(t *testing.T) {
	registry := operation.DefaultRegistry()
	commands := cli.Commands(t.TempDir())
	commandsByVerb := make(map[string]*cobra.Command, len(commands))
	for _, cmd := range commands {
		commandsByVerb[cmd.Name()] = cmd
	}
	found := 0
	for _, entry := range registry.All() {
		if entry.CLIName == "" {
			continue
		}
		cmd := commandsByVerb[entry.CLIName]
		if cmd == nil {
			t.Fatalf("registry CLI command %q is missing", entry.CLIName)
		}
		for _, param := range entry.Params {
			if param.Type != operation.ParamString || (param.JSONName != "source" && param.JSONName != "body" && param.JSONName != "case") {
				continue
			}
			found++
			flag := cmd.Flags().Lookup(param.CLIName)
			if flag == nil || !strings.Contains(flag.Usage, "stdin") {
				t.Errorf("%s --%s does not advertise stdin payload input", entry.CLIName, param.CLIName)
			}
		}
	}
	if found == 0 {
		t.Fatal("registry has no source, body, or case payload parameters")
	}
}

func TestCommandsIncludeLanguageContractsInLongHelp(t *testing.T) {
	registry := operation.DefaultRegistry()
	commands := cli.Commands(t.TempDir())
	for _, command := range commands {
		entry, ok := registry.LookupCLI(command.Name())
		if !ok {
			t.Fatalf("command %q missing registry entry", command.Name())
		}
		want := entry.Description()
		if command.Long != want {
			t.Errorf("command %q Long = %q, want registry description %q", command.Name(), command.Long, want)
		}
		if len(entry.LanguageContracts) > 0 && command.Long == command.Short {
			t.Errorf("command %q help omits language contract details", command.Name())
		}
	}
}

func TestOutlineFileAliasIsHiddenAndSchemaCanonical(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("outline")
	if !ok {
		t.Fatal("outline registry entry missing")
	}
	var command *cobra.Command
	for _, candidate := range cli.Commands(t.TempDir()) {
		if candidate.Name() == "outline" {
			command = candidate
			break
		}
	}
	if command == nil {
		t.Fatal("outline command missing")
	}
	fileAlias := command.Flags().Lookup("file")
	if fileAlias == nil {
		t.Fatal("outline command is missing the --file alias")
	}
	if !fileAlias.Hidden {
		t.Fatal("outline --file compatibility alias is visible in help")
	}
	var pathParameter *operation.ParameterContract
	for index := range entry.Params {
		param := &entry.Params[index]
		if param.JSONName == "path" {
			pathParameter = param
		}
		if param.JSONName == "file" {
			t.Fatal("outline advertises file as a canonical parameter")
		}
	}
	if pathParameter == nil {
		t.Fatal("outline canonical path parameter missing")
	}
	if len(pathParameter.InputAliases) != 1 || pathParameter.InputAliases[0] != "file" {
		t.Fatalf("outline path input aliases = %v, want [file]", pathParameter.InputAliases)
	}
}
