// toolexec_test.go proves linker publication and fixture Go commands work on a real tmpfs.
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestFixtureToolexecRefreshesLinkerOutputOnTmpfs(t *testing.T) {
	root := os.Getenv("SEMEDIT_TEST_TMPFS_ROOT")
	if root == "" {
		t.Skip("set SEMEDIT_TEST_TMPFS_ROOT to a mounted tmpfs to run this integration test")
	}
	tmpfs, err := isTmpfsMount(root)
	if err != nil || !tmpfs {
		t.Fatalf("SEMEDIT_TEST_TMPFS_ROOT %q must be a mounted tmpfs: %v", root, err)
	}
	workDir, err := os.MkdirTemp(root, "semedit-toolexec-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workDir) })
	linker := filepath.Join(workDir, "link")
	output := filepath.Join(workDir, "program")
	writeErr := os.WriteFile(linker, []byte("#!/bin/sh\nprintf 'exact-linker-bytes' > \"$2\"\n"), 0o755)
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	chmodErr := os.Chmod(linker, 0o755)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	writeErr = os.WriteFile(output, []byte("old"), 0o755)
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	before, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if status := runFixtureToolexec([]string{linker, "-o", output}); status != 0 {
		t.Fatalf("fixture toolexec status = %d, want 0", status)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "exact-linker-bytes" {
		t.Fatalf("linker output = %q, want exact bytes", contents)
	}
	after, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("linker output inode was not refreshed")
	}
	if before.Mode().Perm() != after.Mode().Perm() || after.Mode().Perm() != 0o755 {
		t.Fatalf("linker output mode changed from %v to %v", before.Mode().Perm(), after.Mode().Perm())
	}
	if !after.ModTime().After(before.ModTime()) {
		t.Fatalf("linker output mtime did not advance: before %s, after %s", before.ModTime(), after.ModTime())
	}
}

func TestFixtureGoCommandsRunOnTmpfsWithCachedOutputsAndPathSpaces(t *testing.T) {
	root := os.Getenv("SEMEDIT_TEST_TMPFS_ROOT")
	if root == "" {
		t.Skip("set SEMEDIT_TEST_TMPFS_ROOT to a mounted tmpfs to run this integration test")
	}
	tmpfs, err := isTmpfsMount(root)
	if err != nil || !tmpfs {
		t.Fatalf("SEMEDIT_TEST_TMPFS_ROOT %q must be a mounted tmpfs: %v", root, err)
	}
	workDir, err := os.MkdirTemp(root, "semedit-go-smoke-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workDir) })
	writeFixtureGoModule(t, workDir)
	env, err := fixtureGoEnvironment(gocache.WithBaseDir(context.Background(), filepath.Join(workDir, ".scratch", "go")), workDir)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" && !strings.Contains(environmentValue(env, "GOFLAGS"), "-toolexec") {
		t.Fatalf("Darwin tmpfs GoFLAGS = %q, want the linker refresh wrapper", environmentValue(env, "GOFLAGS"))
	}
	// Use an executable path with spaces to verify both Go flag-parsing layers.
	helperExecutable := copyToolexecTestBinary(t)
	toolFlags, err := fixtureToolexecGOFlags(helperExecutable)
	if err != nil {
		t.Fatal(err)
	}
	env = replaceEnvironment(env, map[string]string{"GOFLAGS": "-mod=readonly -buildvcs=false " + toolFlags})
	runGo := func(args ...string) string {
		t.Helper()
		command := exec.CommandContext(t.Context(), "go", args...)
		command.Dir = workDir
		command.Env = env
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}
	for range 2 {
		runGo("test", "-count=1", "./...")
		if got := runGo("run", "."); got != "fixture-ok\n" {
			t.Fatalf("go run output = %q", got)
		}
	}
	binaryPath := filepath.Join(workDir, "fixture-app")
	for range 2 {
		runGo("build", "-o", binaryPath, ".")
		command := exec.CommandContext(t.Context(), binaryPath)
		command.Env = env
		output, err := command.CombinedOutput()
		if err != nil || string(output) != "fixture-ok\n" {
			t.Fatalf("built program output = %q, error = %v", output, err)
		}
		removeErr := os.Remove(binaryPath)
		if removeErr != nil {
			t.Fatal(removeErr)
		}
	}
}

func TestFixtureToolexecPreservesLinkerFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("failure passthrough fixture uses a POSIX shell")
	}
	workDir := t.TempDir()
	failureTool := filepath.Join(workDir, "link")
	writeErr := os.WriteFile(failureTool, []byte("#!/bin/sh\nexit 23\n"), 0o755)
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	chmodErr := os.Chmod(failureTool, 0o755)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	if status := runFixtureToolexec([]string{failureTool, "-o", filepath.Join(workDir, "output")}); status != 23 {
		t.Fatalf("fixture toolexec status = %d, want original tool status 23", status)
	}
}

func writeFixtureGoModule(t *testing.T, workDir string) {
	t.Helper()
	for path, contents := range map[string]string{
		"go.mod":       "module example.test/ram\n\ngo 1.23\n",
		"main.go":      "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"fixture-ok\") }\n",
		"main_test.go": "package main\nimport \"testing\"\nfunc TestFixture(t *testing.T) {}\n",
	} {
		writeErr := os.WriteFile(filepath.Join(workDir, path), []byte(contents), 0o600)
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}
}

func copyToolexecTestBinary(t *testing.T) string {
	t.Helper()
	helperDir, err := os.MkdirTemp("", "semedit toolexec helper ")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(helperDir) })
	currentExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	helperExecutable := filepath.Join(helperDir, "benchmark harness")
	binary, err := os.ReadFile(currentExecutable)
	if err != nil {
		t.Fatal(err)
	}
	writeErr := os.WriteFile(helperExecutable, binary, 0o755)
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	chmodErr := os.Chmod(helperExecutable, 0o755)
	if chmodErr != nil {
		t.Fatal(chmodErr)
	}
	return helperExecutable
}
