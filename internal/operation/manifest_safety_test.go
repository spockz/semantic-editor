// WHY: This test enforces ADR-0005 workspace manifest safety across all registered operations, ensuring go.work is never created, mutated, or deleted.

package operation_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/operation"
)

func TestAllOperationsPreserveWorkspaceManifest(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	allOps := registry.All()
	if len(allOps) == 0 {
		t.Fatal("expected registry to contain operations, found none")
	}

	for _, entry := range allOps {
		t.Run(entry.Key, func(t *testing.T) {
			t.Parallel()

			t.Run("seeded_manifest", func(t *testing.T) {
				t.Parallel()
				workDir := t.TempDir()
				setupTestWorkspace(t, workDir, true)

				manifestPath := filepath.Join(workDir, "go.work")
				initialHash := fileHash(t, manifestPath)

				raw := cloneRawMap(entry.ExampleRaw)
				_, _ = entry.Invoke(operation.CallContext{
					Ctx:     context.Background(),
					WorkDir: workDir,
				}, raw)

				if _, err := os.Stat(manifestPath); err != nil {
					t.Fatalf("ADR-0005 invariant violation: operation %q deleted or moved go.work: %v", entry.Key, err)
				}
				afterHash := fileHash(t, manifestPath)
				if initialHash != afterHash {
					t.Fatalf("ADR-0005 invariant violation: operation %q mutated go.work (hash %s != %s)", entry.Key, initialHash, afterHash)
				}
			})

			t.Run("unmanifested", func(t *testing.T) {
				t.Parallel()
				workDir := t.TempDir()
				setupTestWorkspace(t, workDir, false)

				manifestPath := filepath.Join(workDir, "go.work")

				raw := cloneRawMap(entry.ExampleRaw)
				_, _ = entry.Invoke(operation.CallContext{
					Ctx:     context.Background(),
					WorkDir: workDir,
				}, raw)

				if _, err := os.Stat(manifestPath); err == nil {
					t.Fatalf("ADR-0005 invariant violation: operation %q created go.work in an unmanifested workspace", entry.Key)
				}
			})
		})
	}
}

func setupTestWorkspace(t *testing.T, workDir string, includeManifest bool) {
	t.Helper()
	goModContent := "module example.com/safety\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goModContent), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	mainGoContent := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte(mainGoContent), 0o600); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	if includeManifest {
		goWorkContent := "go 1.24\n\nuse .\n"
		if err := os.WriteFile(filepath.Join(workDir, "go.work"), []byte(goWorkContent), 0o600); err != nil {
			t.Fatalf("write go.work: %v", err)
		}
	}
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func cloneRawMap(raw map[string]any) map[string]any {
	if raw == nil {
		return make(map[string]any)
	}
	cloned := make(map[string]any, len(raw))
	maps.Copy(cloned, raw)
	return cloned
}
