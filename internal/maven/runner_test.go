package maven_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/maven"
)

func fixture(t *testing.T) (string, backend.WorkspaceTrust) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, backend.NewWorkspaceTrust(root, true)
}

func script(t *testing.T, root string, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil { // #nosec G306 -- executable test fixture.
		t.Fatal(err)
	}
	return path
}

func TestRunWrapperFixedArgsAndScratchEnv(t *testing.T) {
	root, trust := fixture(t)
	log := filepath.Join(root, "args.log")
	script(t, root, "mvnw", "printf '%s\\n' \"$@\" > \""+log+"\"; printf '%s\\n' \"$MAVEN_USER_HOME\" >> \""+log+"\"; exit 0")
	result, err := maven.Run(context.Background(), maven.Request{Root: root, Trust: trust, Tool: maven.ToolWrapper, Goal: "test-compile"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log) // #nosec G304 -- test-controlled temporary path.
	text := string(data)
	for _, want := range []string{"--batch-mode", "--no-transfer-progress", "-f", filepath.Join(root, "pom.xml"), "-o", "test-compile", "maven-home-"} {
		if !strings.Contains(text, want) {
			t.Errorf("wrapper log missing %q: %s", want, text)
		}
	}
	if result.ExitCode != 0 || result.Tool != "wrapper" {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunRequiresTrustAndRejectsOutsideWrapper(t *testing.T) {
	root, _ := fixture(t)
	_, err := maven.Run(context.Background(), maven.Request{Root: root, Trust: backend.NewWorkspaceTrust(root, false), Tool: maven.ToolSystem, SystemBin: "/bin/true", Goal: "test"})
	if !errors.Is(err, maven.ErrTrustRequired) {
		t.Fatalf("err=%v", err)
	}
	outside := script(t, t.TempDir(), "mvnw", "exit 0")
	if symlinkErr := os.Symlink(outside, filepath.Join(root, "mvnw")); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	_, err = maven.Run(context.Background(), maven.Request{Root: root, Trust: backend.NewWorkspaceTrust(root, true), Tool: maven.ToolWrapper, Goal: "test"})
	if !errors.Is(err, maven.ErrUnsafeWrapper) {
		t.Fatalf("outside wrapper err=%v", err)
	}
}

func TestRunFailureCapturesTypedExit(t *testing.T) {
	root, trust := fixture(t)
	bin := script(t, root, "mvn", "echo failed >&2; exit 7")
	result, err := maven.Run(context.Background(), maven.Request{Root: root, Trust: trust, Tool: maven.ToolSystem, SystemBin: bin, Goal: "test"})
	if err == nil || result.ExitCode != 7 || !strings.Contains(result.Stderr, "failed") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRunCancellationStopsMavenAndRetainsResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture requires a Unix shell and process groups")
	}
	root, trust := fixture(t)
	bin := script(t, root, "mvn-slow", "sleep 30 & child=$!; printf 'started\\n'; printf '%s %s\\n' \"$$\" \"$child\" > maven-child.pid.tmp && mv maven-child.pid.tmp maven-child.pid || exit 1; exit 0")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	var result maven.Result
	var runErr error
	go func() {
		result, runErr = maven.Run(ctx, maven.Request{Root: root, Trust: trust, Tool: maven.ToolSystem, SystemBin: bin, Goal: "test"})
		close(done)
	}()
	waitForExitedMavenParent(t, root)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Maven cancellation did not stop the process")
	}
	if runErr == nil || !errors.Is(runErr, context.Canceled) {
		t.Fatalf("err=%v", runErr)
	}
	var typed *maven.Error
	if !errors.As(runErr, &typed) || typed.MavenResult() == nil || result.Error == "" {
		t.Fatalf("result=%+v err=%v", result, runErr)
	}
	if !strings.Contains(result.Stdout, "started") {
		t.Fatalf("result did not retain output written before cancellation: %+v", result)
	}
}

func waitForExitedMavenParent(t *testing.T, root string) {
	t.Helper()
	// Startup can be slow under full race/Hugo load; cancellation has its own short bound below.
	deadline := time.NewTimer(30 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	var parentPID, childPID int
	for {
		if data, err := os.ReadFile(filepath.Join(root, "maven-child.pid")); err == nil {
			pids := strings.Fields(string(data))
			if len(pids) != 2 {
				t.Fatalf("invalid Maven process marker %q", data)
			}
			parentPID, err = strconv.Atoi(pids[0])
			if err != nil {
				t.Fatalf("invalid Maven parent pid %q: %v", pids[0], err)
			}
			childPID, err = strconv.Atoi(pids[1])
			if err != nil {
				t.Fatalf("invalid Maven child pid %q: %v", pids[1], err)
			}
			break
		}
		select {
		case <-deadline.C:
			t.Fatal("Maven child did not start")
		case <-ticker.C:
		}
	}
	for {
		alive, err := mavenProcessAlive(parentPID)
		if err != nil {
			t.Fatalf("check Maven parent process: %v", err)
		}
		if !alive {
			break
		}
		select {
		case <-deadline.C:
			t.Fatal("Maven parent did not exit before cancellation")
		case <-ticker.C:
		}
	}
	childAlive, err := mavenProcessAlive(childPID)
	if err != nil || !childAlive {
		t.Fatalf("Maven child was not alive before cancellation: alive=%t err=%v", childAlive, err)
	}
}
