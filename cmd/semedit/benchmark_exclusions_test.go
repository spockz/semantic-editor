// Native Go discovery and Make formatting must prune benchmark data before entering archived workspaces.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/atomicfile"
)

func TestGoAndFormattingPruneBenchmarkData(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	makefile, makefileErr := os.ReadFile(filepath.Join(repositoryRoot, "Makefile"))
	if makefileErr != nil {
		t.Fatal(makefileErr)
	}
	boundary, boundaryErr := os.ReadFile(filepath.Join(repositoryRoot, "data", "benchmarks", "go.mod"))
	if boundaryErr != nil {
		t.Fatal(boundaryErr)
	}
	files := map[string]string{
		"Makefile":                           string(makefile),
		"go.mod":                             "module example.com/exclusions\n\ngo 1.24\n",
		"main.go":                            "package main;func main(){}\n",
		"data/benchmarks/go.mod":             string(boundary),
		"data/benchmarks/archive/invalid.go": "invalid Go source\n",
		".scratch/formatters/gofumpt":        "#!/bin/sh\nset -eu\nexec \"$GOFMT\" \"$@\"\n",
		".scratch/formatters/goimports":      "#!/bin/sh\nset -eu\nshift 3\nexec \"$GOFMT\" -w \"$@\"\n",
	}
	for relative, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := atomicfile.WriteAtomic(path, []byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"gofumpt", "goimports"} {
		if err := os.Chmod(filepath.Join(root, ".scratch", "formatters", name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	archive := filepath.Join(root, "data", "benchmarks", "archive")
	if err := os.Chmod(archive, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(archive, 0o750); err != nil {
			t.Error(err)
		}
	})
	listed := exec.CommandContext(t.Context(), "go", "list", "./...")
	listed.Dir = root
	if output, err := listed.CombinedOutput(); err != nil || strings.TrimSpace(string(output)) != "example.com/exclusions" {
		t.Fatalf("Go discovery entered benchmark data: %s, %v", output, err)
	}
	gofmt, lookupErr := exec.LookPath("gofmt")
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	command := exec.CommandContext(t.Context(), "make", "fmt-go")
	command.Dir = root
	command.Env = append(os.Environ(), "GOFMT="+gofmt, "GOTOOLCHAIN=local", "MAKEFLAGS=", "PATH="+filepath.Join(root, ".scratch", "formatters")+string(os.PathListSeparator)+os.Getenv("PATH"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("formatting entered benchmark data: %s, %v", output, err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil || string(contents) != "package main\n\nfunc main() {}\n" {
		t.Fatalf("source was not formatted: %q, %v", contents, err)
	}
}
