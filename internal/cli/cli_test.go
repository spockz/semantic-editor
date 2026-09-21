// Package cli_test verifies registry-derived command generation without invoking language backends.
package cli_test

import (
	"testing"

	"semedit/internal/cli"
	"semedit/internal/operation"
)

func TestCommandsCoverRegistryVerbs(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	commands := cli.Commands(t.TempDir())
	verbs := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		verb := cmd.Name()
		if verbs[verb] {
			t.Errorf("duplicate CLI verb %q", verb)
		}
		verbs[verb] = true
		if _, ok := registry.LookupCLI(verb); !ok {
			t.Errorf("verb %q has no registry operation", verb)
		}
	}
	for _, entry := range registry.All() {
		if entry.CLIName == "" {
			continue
		}
		if !verbs[entry.CLIName] {
			t.Errorf("registry CLI name %q has no generated command", entry.CLIName)
		}
	}
}
