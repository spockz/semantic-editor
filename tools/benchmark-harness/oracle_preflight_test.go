// oracle_preflight_test.go verifies hidden acceptance dependencies before timed provider execution.

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestPreflightOracleDependencies(t *testing.T) {
	t.Run("download failure removes private copy", func(t *testing.T) {
		fixture := newOraclePreflightFixture(t, "test -f go.mod\ntest -f acceptance_test.go\nprintf '%s' \"$PWD\" > \"$SEMEDIT_PREFLIGHT_DIR\"\nexit 7\n")
		err := prepareSharedGoDependencies(context.Background(), fixture.task, fixture.workDir)
		if err == nil || !strings.Contains(err.Error(), "download shared Go modules") {
			t.Fatalf("setup error = %v, want download failure", err)
		}
		assertOraclePreflightCleanup(t, fixture)
	})
	t.Run("storage exhaustion removes private copy", func(t *testing.T) {
		fixture := newOraclePreflightFixture(t, "test -f acceptance_test.go\nprintf '%s' \"$PWD\" > \"$SEMEDIT_PREFLIGHT_DIR\"\nprintf 'go: writing module cache: no space left on device\\n' >&2\nexit 1\n")
		err := prepareSharedGoDependencies(context.Background(), fixture.task, fixture.workDir)
		if err == nil || !strings.Contains(err.Error(), "no space left on device") {
			t.Fatalf("setup error = %v, want storage exhaustion", err)
		}
		assertOraclePreflightCleanup(t, fixture)
	})
	t.Run("download then offline validation uses shared cache", func(t *testing.T) {
		fixture := newOraclePreflightFixture(t, `
test -f go.mod
test -f acceptance_test.go
test "$GOMODCACHE" = "$SEMEDIT_PREFLIGHT_MODULE_CACHE"
test "$GOTOOLCHAIN" = local
test "$GOSUMDB" = off
printf '%s' "$PWD" > "$SEMEDIT_PREFLIGHT_DIR"
if [ "$1" = mod ]; then
  test "$*" = 'mod download all'
  test "$GOPROXY" = file:///setup-proxy
  printf downloaded > download-marker
  exit 0
fi
test -f download-marker
test "$GOPROXY" = off
test "$CGO_ENABLED" = 0
case "$*" in *' -m -json all') printf '{"Path":"preflight","Main":true,"Dir":"%s"}\n' "$PWD";; esac
`)
		moduleCache := filepath.Join(fixture.root, "persistent-modules")
		if err := os.MkdirAll(moduleCache, 0o750); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SEMEDIT_PREFLIGHT_MODULE_CACHE", moduleCache)
		t.Setenv("GOPROXY", "file:///setup-proxy")
		t.Setenv("GOSUMDB", "off")
		ctx := gocache.WithModuleCache(context.Background(), moduleCache)
		if err := prepareSharedGoDependencies(ctx, fixture.task, fixture.workDir); err != nil {
			t.Fatal(err)
		}
		assertOraclePreflightCleanup(t, fixture)
	})
	t.Run("cancellation reaches download and removes copy", func(t *testing.T) {
		fixture := newOraclePreflightFixture(t, "printf '%s' \"$PWD\" > \"$SEMEDIT_PREFLIGHT_DIR\"\nexec sleep 30\n")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result := make(chan error, 1)
		go func() { result <- prepareSharedGoDependencies(ctx, fixture.task, fixture.workDir) }()
		waitForOraclePreflightFile(t, fixture.dirLog)
		cancel()
		select {
		case err := <-result:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("setup error = %v, want cancellation", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("download did not stop after cancellation")
		}
		assertOraclePreflightCleanup(t, fixture)
	})
}

type oraclePreflightFixture struct {
	root    string
	workDir string
	hidden  string
	dirLog  string
	task    *Task
}

func newOraclePreflightFixture(t *testing.T, body string) oraclePreflightFixture {
	t.Helper()
	root := t.TempDir()
	workDir := filepath.Join(root, "fixture")
	hidden := filepath.Join(root, "hidden")
	binDir := filepath.Join(root, "bin")
	for _, dir := range []string{workDir, hidden, binDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module preflight\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hidden, "acceptance_test.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirLog := filepath.Join(root, "private-dir")
	program := "#!/bin/sh\nset -eu\n" + body
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(program), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_PREFLIGHT_DIR", dirLog)
	task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{Test: TestConfig{PassTests: true, HiddenTestDir: hidden}}}}
	return oraclePreflightFixture{root: root, workDir: workDir, hidden: hidden, dirLog: dirLog, task: task}
}

func assertOraclePreflightCleanup(t *testing.T, fixture oraclePreflightFixture) {
	t.Helper()
	data, err := os.ReadFile(fixture.dirLog)
	if err != nil {
		t.Fatalf("read private workspace path: %v", err)
	}
	privateDir := string(data)
	if privateDir == "" || privateDir == fixture.workDir {
		t.Fatalf("preflight used workspace %q, want private copy", privateDir)
	}
	if _, err := os.Stat(privateDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private oracle copy remains at %s: %v", privateDir, err)
	}
	if _, err := os.Stat(filepath.Join(fixture.workDir, "acceptance_test.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("hidden test reached original workspace: %v", err)
	}
}

func waitForOraclePreflightFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("fake Go command did not start")
}
