// Package cli_test verifies registry-derived command generation without invoking language backends.
package cli_test

import (
	"strings"
	"testing"

	"semedit/internal/cli"
	"semedit/internal/operation"

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
