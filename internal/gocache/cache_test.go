// Temporary file coverage ensures retained language servers do not pollute workspace snapshots.

package gocache_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/gocache"
	"strings"
	"testing"
)

func TestGoSubprocessTemporaryFiles(t *testing.T) {
	if os.Getenv("SEMEDIT_TEMP_HELPER") == "1" {
		file, err := os.CreateTemp("", "semedit-child-")
		if err != nil {
			panic(err)
		}
		if err := file.Close(); err != nil {
			panic(err)
		}
		fmt.Print(file.Name())
		os.Exit(0)
	}
	workspace := t.TempDir()
	inherited := filepath.Join(workspace, ".tmp")
	if err := os.MkdirAll(inherited, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, inherited)
	}
	environment, err := gocache.Environment(t.Context(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestGoSubprocessTemporaryFiles$")
	environment = append(environment, "SEMEDIT_TEMP_HELPER=1")
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("child temporary file: %v: %s", err, output)
	}
	path := string(output)
	defer func() {
		if removeErr := os.Remove(path); removeErr != nil {
			t.Error(removeErr)
		}
	}()
	expected := filepath.Join(workspace, ".scratch", "go", "tmp")
	actual, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	expected, err = filepath.EvalSymlinks(expected)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("temporary file %q, want directory %q", path, expected)
	}
	entries, err := os.ReadDir(inherited)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("inherited temporary directory polluted: %v", entries)
	}
}

func TestEnvironmentKeepsGoplsCacheBelowSelectedBase(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "selected", ".scratch", "go")
	environment, err := gocache.Environment(gocache.WithBaseDir(t.Context(), baseDir), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := "GOPLSCACHE=" + filepath.Join(baseDir, "gopls")
	foundGoplsCache := false
	for _, variable := range environment {
		if strings.HasPrefix(variable, "GOPLSCACHE=") {
			if variable != want {
				t.Fatalf("GOPLSCACHE = %q, want %q", variable, want)
			}
			foundGoplsCache = true
		}
	}
	if !foundGoplsCache {
		t.Fatal("Go subprocess environment omitted GOPLSCACHE")
	}
}
