// This file verifies structural snapshot, field-selector, and continuation behavior through operation boundaries.

package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	hash := sha256.Sum256(source)
	entry, ok := DefaultRegistry().LookupMCP(insertCompositeElementToolName)
	if !ok {
		t.Fatal("composite insert operation is not registered")
	}
	draft := EditReceiptDraft{
		Operation: "insert_composite_element",
		Status:    EditPersisted,
		PlannedChanges: []FileRevision{{
			Path: "value.go", Revision: hex.EncodeToString(hash[:]),
		}},
	}
	result := compositePlanResult{
		Structural: structuralPlanResult{Edit: FileEditRes{Detail: "Appended one keyed composite element."}, Draft: draft},
		Candidate:  compositeSelectorCandidate{File: "value.go", Declaration: "thing", ElementSource: "1"},
	}
	finalized, err := entry.FinalizeOutput(CallContext{WorkDir: root}, map[string]any{}, result)
	if err != nil {
		t.Fatalf("finalize duplicate declaration result: %v", err)
	}
	projected, err := entry.ProjectOutput(finalized)
	if err != nil {
		t.Fatalf("project duplicate declaration result: %v", err)
	}
	output, ok := projected.(map[string]any)
	if !ok {
		t.Fatalf("projected composite result = %#v, want map", projected)
	}
	if _, exists := output["selector_continuations"]; exists {
		t.Fatalf("duplicate declaration produced a live selector continuation: %#v", output)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read duplicate declarations after refusal: %v", readErr)
	}
	if string(after) != string(source) {
		t.Fatalf("continuation projection changed source bytes: %q", after)
	}
}

func TestInsertFieldRegistryRejectsLeadingDotQualifierWithoutWriting(t *testing.T) {
	t.Parallel()
	invalidFixture := newStructuralFieldFixture(t)
	err := invokeStructuralFieldInsert(invalidFixture.Root, ".t", ".t.extra")
	if err == nil || !strings.Contains(err.Error(), "invalid package qualifier") {
		t.Fatalf("leading-dot in_type error = %v, want invalid package qualifier refusal", err)
	}
	assertSourceBytes(t, invalidFixture.Path, invalidFixture.Before, "source changed after invalid in_type")

	validFixture := newStructuralFieldFixture(t)
	if err := invokeStructuralFieldInsert(validFixture.Root, "t", "t.extra"); err != nil {
		t.Fatalf("valid field insertion rejected: %v", err)
	}
	updated, readErr := os.ReadFile(validFixture.Path)
	if readErr != nil {
		t.Fatalf("read valid field insertion result: %v", readErr)
	}
	if !strings.Contains(string(updated), "extra int") {
		t.Fatalf("valid field insertion result missing extra field: %q", updated)
	}
}

type structuralFieldFixture struct {
	Root   string
	Path   string
	Before []byte
}

func newStructuralFieldFixture(t *testing.T) structuralFieldFixture {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/fieldselector\n\ngo 1.24\n")
	path := filepath.Join(root, "types.go")
	before := []byte("package fieldselector\n\ntype t struct {\n\texisting int\n}\n")
	writeFixtureFile(t, path, string(before))
	return structuralFieldFixture{Root: root, Path: path, Before: before}
}

func invokeStructuralFieldInsert(root, inType, symbol string) error {
	entry, ok := DefaultRegistry().LookupMCP("semantic_insert_field")
	if !ok {
		return fmt.Errorf("struct field operation is not registered")
	}
	_, err := entry.Invoke(CallContext{Ctx: context.Background(), WorkDir: root}, map[string]any{
		"file": "types.go", "in_type": inType, "symbol": symbol,
		"source": "extra int", "placement": "type_end",
	})
	return err
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertSourceBytes(t *testing.T, path string, want []byte, detail string) {
	t.Helper()
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read source after operation: %v", readErr)
	}
	if string(after) != string(want) {
		t.Fatalf("%s: %q", detail, after)
	}
}
