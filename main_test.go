// Package main_test verifies entry point functionality for semedit.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"semedit": func() {
			os.Exit(run(os.Args[1:]))
		},
	})
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("testdata", "scripts"),
		Setup: func(env *testscript.Env) error {
			for _, key := range []string{"GOPATH", "GOCACHE", "GOROOT", "HOME"} {
				if val := os.Getenv(key); val != "" {
					env.Vars = append(env.Vars, key+"="+val)
				}
			}

			// Prepend ~/go/bin to PATH for gopls discovery inside testscript sandboxes
			goBin := filepath.Join(os.Getenv("HOME"), "go", "bin")
			pathFound := false
			for i, v := range env.Vars {
				if after, ok := strings.CutPrefix(v, "PATH="); ok {
					env.Vars[i] = "PATH=" + goBin + ":" + after
					pathFound = true
					break
				}
			}
			if !pathFound {
				env.Vars = append(env.Vars, "PATH="+goBin+":"+os.Getenv("PATH"))
			}

			return nil
		},
	})
}

func TestMCPLiveReloadFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd(".")
	mcpCmd, _, err := cmd.Find([]string{"mcp"})
	if err != nil {
		t.Fatalf("failed to find mcp command: %v", err)
	}

	flag := mcpCmd.Flags().Lookup("live-reload")
	if flag == nil {
		t.Fatalf("expected --live-reload flag on mcp command")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --live-reload default value 'false', got %q", flag.DefValue)
	}
}
