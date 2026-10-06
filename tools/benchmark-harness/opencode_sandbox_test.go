// These tests prove OpenCode's operating-system sandbox permits fixture work while denying parent-workspace reads.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func installPassthroughOpenCodeSandbox(t *testing.T, binDir string) {
	t.Helper()
	script := "#!/bin/sh\n[ \"$1\" = -f ] || exit 2\nshift 2\nexec \"$@\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "sandbox-exec"), []byte(script), 0o750); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeSeatbeltAllowsFixtureAndDeniesOutsideFiles(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS Seatbelt is required")
	}
	sandboxPath, lookupErr := exec.LookPath("sandbox-exec")
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	root := t.TempDir()
	fixtureRoot := filepath.Join(root, "fixture")
	if mkdirErr := os.MkdirAll(fixtureRoot, 0o750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	fixtureFile := filepath.Join(fixtureRoot, "allowed.txt")
	outsideFile := filepath.Join(root, "hidden.txt")
	if writeErr := os.WriteFile(fixtureFile, []byte("fixture-visible"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if writeErr := os.WriteFile(outsideFile, []byte("outside-secret"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if linkErr := os.Symlink(outsideFile, filepath.Join(fixtureRoot, "outside-link")); linkErr != nil {
		t.Fatal(linkErr)
	}
	profile, profileErr := renderOpenCodeSeatbeltProfile(fixtureRoot, "1234", "/bin/sh", "/bin/cat", "")
	if profileErr != nil {
		t.Fatal(profileErr)
	}
	profilePath := filepath.Join(root, "benchmark.sb")
	if writeErr := os.WriteFile(profilePath, []byte(profile), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	run := func(path string) ([]byte, error) {
		return exec.Command(sandboxPath, "-f", profilePath, "/bin/cat", path).CombinedOutput()
	}
	output, readErr := run(fixtureFile)
	if readErr != nil {
		if strings.Contains(string(output), "sandbox_apply: Operation not permitted") {
			t.Skipf("host does not permit nested macOS Seatbelt profiles: %s", strings.TrimSpace(string(output)))
		}
		t.Fatalf("fixture read failed: %v: %s", readErr, output)
	}
	if string(output) != "fixture-visible" {
		t.Fatalf("fixture read = %q", output)
	}
	for _, path := range []string{outsideFile, filepath.Join(fixtureRoot, "outside-link")} {
		output, outsideReadErr := run(path)
		if outsideReadErr == nil {
			t.Errorf("sandbox read outside path %q: %q", path, output)
		}
	}
}
