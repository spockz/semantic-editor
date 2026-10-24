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
	source := []byte("package p\n\nvar (\n\tthing = map[string]int{\"key\": 1}\n\tthing = map[string]int{\"other\": 2}\n)\n")
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatalf("write duplicate declarations: %v", err)
	}
	hash := sha256.Sum256(source)
	entry, ok := DefaultRegistry().LookupMCP("semantic_insert_construct")
	if !ok {
		t.Fatal("generic construct insert operation is not registered")
	}
	draft := EditReceiptDraft{
		Operation: "insert_construct",
		Status:    EditPersisted,
		PlannedChanges: []FileRevision{{
			Path: "value.go", Revision: hex.EncodeToString(hash[:]),
		}},
	}
	result := FileEditRes{
		Detail: "Appended one keyed composite element.", ReceiptDraft: draft,
		ElementCandidate: &compositeSelectorCandidate{File: "value.go", Declaration: "thing", ElementSource: `"key": 1`},
	}
	finalized, err := entry.FinalizeOutput(CallContext{WorkDir: root}, map[string]any{
		"file": "value.go", "kind": "element", "source": `"key": 3`,
		"in_declaration": "thing", "discriminator": `map[string]int{"key": 1}`,
	}, result)
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
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("canonicalize structural field workspace: %v", err)
	}
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

func TestStructuralSnapshotRechecksLocalReplacementInputs(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	appRoot := filepath.Join(workspace, "app")
	depRoot := filepath.Join(workspace, "dep")
	for _, dir := range []string{appRoot, depRoot} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("create module directory %q: %v", dir, err)
		}
	}
	writeFixtureFile(t, filepath.Join(appRoot, "go.mod"), "module example.com/app\n\ngo 1.24\n\nrequire example.com/dep v0.0.0\nreplace example.com/dep => ../dep\n")
	writeFixtureFile(t, filepath.Join(appRoot, "main.go"), "package app\n\nimport \"example.com/dep\"\n\nfunc Value() int { return dep.Value() }\n")
	depManifest := filepath.Join(depRoot, "go.mod")
	beforeDepManifest := "module example.com/dep\n\ngo 1.24\n"
	writeFixtureFile(t, depManifest, beforeDepManifest)
	depSource := filepath.Join(depRoot, "dep.go")
	beforeDepSource := "package dep\n\nimport _ \"embed\"\n\n//go:embed message.txt\nvar message string\n\nfunc Value() int { return len(message) }\n"
	writeFixtureFile(t, depSource, beforeDepSource)
	depAsset := filepath.Join(depRoot, "message.txt")
	beforeDepAsset := "inventory\n"
	writeFixtureFile(t, depAsset, beforeDepAsset)
	appRoot, err := filepath.Abs(appRoot)
	if err != nil {
		t.Fatalf("resolve app module path: %v", err)
	}
	snapshot, err := prepareStructuralSnapshot(context.Background(), CallContext{WorkDir: appRoot}, "main.go")
	if err != nil {
		t.Fatalf("prepare structural snapshot: %v", err)
	}
	for _, path := range []string{depSource, depManifest, depAsset} {
		if _, captured := snapshot.Sources[path]; !captured {
			t.Fatalf("local replacement input %q was not recorded in the structural snapshot", path)
		}
	}
	output := snapshot.Sources[snapshot.File]
	writeFixtureFile(t, depSource, "package dep\n\nimport _ \"embed\"\n\n//go:embed message.txt\nvar message string\n\nfunc Value() int { return len(message) + 1 }\n")
	assertStructuralOutputRefused(t, snapshot, output, "local replacement source drift")
	writeFixtureFile(t, depSource, beforeDepSource)
	if err := snapshot.Buffer.AssertDisk(); err != nil {
		t.Fatalf("restore local replacement source preimage: %v", err)
	}
	writeFixtureFile(t, depManifest, "module example.com/dep\n\ngo 1.25\n")
	assertStructuralOutputRefused(t, snapshot, output, "local replacement manifest drift")
	writeFixtureFile(t, depManifest, beforeDepManifest)
	if err := snapshot.Buffer.AssertDisk(); err != nil {
		t.Fatalf("restore local replacement manifest preimage: %v", err)
	}
	writeFixtureFile(t, depAsset, "inventory replenished\n")
	assertStructuralOutputRefused(t, snapshot, output, "local replacement embedded asset drift")
}

func assertStructuralOutputRefused(t *testing.T, snapshot *structuralSnapshot, output []byte, scenario string) {
	t.Helper()
	if _, err := snapshot.applyOutput(output); err == nil || !strings.Contains(err.Error(), "revalidate structural read set") {
		t.Fatalf("applyOutput after %s = %v, want captured-input refusal", scenario, err)
	}
	after, err := os.ReadFile(snapshot.File)
	if err != nil {
		t.Fatalf("read target after %s: %v", scenario, err)
	}
	if string(after) != string(output) {
		t.Fatalf("target changed after %s: %q", scenario, after)
	}
}
