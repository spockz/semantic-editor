// Shared diagnostics retain Go subprocess errors so sandbox runtime failures remain actionable.

package godiagnostics

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedDiagnosticsReportsGoRuntimeFailure(t *testing.T) {
	gopls, err := FindGopls()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOROOT", filepath.Join(t.TempDir(), "missing-go-runtime"))
	manager := NewSessionManager()
	defer func() {
		if closeErr := manager.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	_, err = manager.CheckDetailed(t.Context(), t.TempDir(), gopls, os.Environ())
	if err == nil || !strings.Contains(err.Error(), "go: cannot find GOROOT directory:") {
		t.Fatalf("diagnostics error lacks Go stderr: %v", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("diagnostics error lost subprocess cause: %v", err)
	}
}
