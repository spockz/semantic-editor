package maven_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"semedit/internal/backend"
	"semedit/internal/maven"
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
	if err := os.Symlink(outside, filepath.Join(root, "mvnw")); err != nil {
		t.Fatal(err)
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
	root, trust := fixture(t)
	bin := script(t, root, "mvn-slow", "sleep 30")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var result maven.Result
	var runErr error
	go func() {
		result, runErr = maven.Run(ctx, maven.Request{Root: root, Trust: trust, Tool: maven.ToolSystem, SystemBin: bin, Goal: "test"})
		close(done)
	}()
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
}
