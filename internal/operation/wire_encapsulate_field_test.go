// Package operation tests Go field encapsulation behavior through the registered public operation.
// WHY: Cancellation must stop analysis before a partially validated edit reaches the source buffer.
package operation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/pipeline"
)

func TestEncapsulateFieldCanceledAnalysisLeavesSourcesUnchanged(t *testing.T) {
	root := t.TempDir()
	original := []byte("package sample\n\ntype config struct { value int }\n\nfunc (c *config) read() int { return c.value }\n")
	if err := pipeline.WriteAtomic(filepath.Join(root, "go.mod"), []byte("module example.com/canceledencapsulate\n\ngo 1.24\n")); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(root, "config.go"), original); err != nil {
		t.Fatal(err)
	}
	registry := DefaultRegistry()
	entry, ok := registry.LookupMCP("refactor_encapsulate_field")
	if !ok {
		t.Fatal("encapsulate-field operation is not registered")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	project := backend.ProjectContext{RootDir: root, Language: backend.LanguageGo, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	call := NewCallContext(root, project)
	call.Ctx = ctx
	call.Project = project
	_, err := entry.Invoke(call, map[string]any{
		"file": "config.go", "symbol": "config.value", "getter": "readValue", "setter": "setValue",
	})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled encapsulation error = %v, want context cancellation", err)
	}
	current, readErr := os.ReadFile(filepath.Join(root, "config.go"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(current) != string(original) {
		t.Fatalf("canceled encapsulation changed source: %s", current)
	}
}
