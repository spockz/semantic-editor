// Package main_test verifies entry point functionality for semedit.
package main

import (
	"bytes"
	"os"
	"os/exec"
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
		"rust-analyzer":                   runFakeRustAnalyzer,
		"java":                            runFakeJava,
		"metals":                          runFakeMetals,
		"ghc":                             runFakeGHC,
		"haskell-language-server-wrapper": runFakeHLS,
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

			for variable, command := range map[string]string{
				"SEMEDIT_TEST_JAVA":   "java",
				"SEMEDIT_TEST_GHC":    "ghc",
				"SEMEDIT_TEST_HLS":    "haskell-language-server-wrapper",
				"SEMEDIT_TEST_METALS": "metals",
			} {
				path, err := exec.LookPath(command)
				if err != nil {
					return err
				}
				env.Vars = append(env.Vars, variable+"="+path)
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

func TestRootCommandWithoutArgumentsRendersHelp(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd(t.TempDir())
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root command without arguments failed: %v", err)
	}
	if !strings.Contains(output.String(), "Semantic Editor for LLM Intent-Driven Code Refactoring") {
		t.Errorf("root help = %q, want command description", output.String())
	}
}
