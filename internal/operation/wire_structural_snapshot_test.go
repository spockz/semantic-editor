// This file verifies structural no-op plans still reject analysis read-set drift.

package operation

import (
	"context"
	"go/parser"
	"go/token"
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
	if writeErr := os.WriteFile(manifest, []byte("module example.com/structuraldrift\n\ngo 1.25\n"), 0o600); writeErr != nil {
		t.Fatalf("change module manifest: %v", writeErr)
	}
	changed, err := snapshot.applyOutput(snapshot.Sources[snapshot.File])
	if err == nil || !strings.Contains(err.Error(), "revalidate structural read set") {
		t.Fatalf("no-op output with manifest drift = (%t, %v), want read-set refusal", changed, err)
	}
	if changed {
		t.Fatal("no-op output reported a source change")
	}
}

func TestCompositeContinuationRejectsDuplicateOwnerSpecs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := []byte("package p\n\nvar (\n\tthing = []int{1}\n\tthing = []int{2}\n)\n")
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatalf("write duplicate declarations: %v", err)
	}
	parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments|parser.AllErrors)
	if parseErr != nil {
		t.Fatalf("parse duplicate declarations: %v", parseErr)
	}
	if _, ownerErr := structuralDeclarationOwner(parsed, "thing"); ownerErr == nil || !strings.Contains(ownerErr.Error(), "ambiguous") {
		t.Fatalf("duplicate declaration owner error = %v, want ambiguity refusal", ownerErr)
	}
	receipt := EditReceipt{Changes: []FileRevision{{Path: "value.go", Revision: structuralRevision(source)}}}
	candidate := compositeSelectorCandidate{File: "value.go", Declaration: "thing", ElementSource: "1"}
	_, live, err := resolveCompositeSelectorContinuation(context.Background(), root, candidate, receipt)
	if err != nil {
		t.Fatalf("resolve duplicate declaration continuation: %v", err)
	}
	if live {
		t.Fatal("duplicate declaration produced a live selector continuation")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read duplicate declarations after refusal: %v", readErr)
	}
	if string(after) != string(source) {
		t.Fatal("continuation refusal changed the source")
	}
}
