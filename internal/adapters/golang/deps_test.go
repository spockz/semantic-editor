// Package golang_test validates Go adapter dependency operations.
package golang_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/adapters/golang"
)

func TestAddDependency_Validation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := golang.AddDependency(ctx, t.TempDir(), "")
	if !errors.Is(err, golang.ErrEmptyPackage) {
		t.Fatalf("expected ErrEmptyPackage, got %v", err)
	}
}

func TestAddDependencyPermitsExplicitManifestChanges(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "go.mod")
	if err := os.WriteFile(manifest, []byte("module example.com/deps\n\ngo 1.27.2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(t.TempDir(), "trace")
	stub := filepath.Join(binDir, "go")
	script := "#!/bin/sh\nprintf '%s %s\\n' \"$GOFLAGS\" \"$*\" >> \"$SEMEDIT_DEP_TRACE\"\nif [ \"$1\" = get ]; then printf 'require example.com/added v1.0.0\\n' >> go.mod; fi\n"
	if err := os.WriteFile(stub, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_DEP_TRACE", trace)
	if err := golang.AddDependency(context.Background(), root, "example.com/added@v1.0.0"); err != nil {
		t.Fatalf("AddDependency() error: %v", err)
	}
	manifestData, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifestData), "require example.com/added v1.0.0") {
		t.Fatalf("go.mod was not changed by explicit dependency operation: %s", manifestData)
	}
	traceData, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(traceData), "-mod=readonly") {
		t.Fatalf("explicit dependency operation inherited read-only module flags: %s", traceData)
	}
}
