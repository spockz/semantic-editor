// This file verifies structural no-op plans still reject analysis read-set drift.

package operation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuralNoopRejectsManifestDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":   "module example.com/structuraldrift\n\ngo 1.24\n",
		"value.go": "package structuraldrift\n\nfunc Value() int { return 1 }\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	snapshot, err := prepareStructuralSnapshot(context.Background(), CallContext{WorkDir: root}, "value.go")
	if err != nil {
		t.Fatalf("prepare structural snapshot: %v", err)
	}
	manifest := filepath.Join(root, "go.mod")
	if err := os.WriteFile(manifest, []byte("module example.com/structuraldrift\n\ngo 1.25\n"), 0o600); err != nil {
		t.Fatalf("change module manifest: %v", err)
	}
	changed, err := snapshot.applyOutput(snapshot.Sources[snapshot.File])
	if err == nil || !strings.Contains(err.Error(), "revalidate structural read set") {
		t.Fatalf("no-op output with manifest drift = (%t, %v), want read-set refusal", changed, err)
	}
	if changed {
		t.Fatal("no-op output reported a source change")
	}
}
