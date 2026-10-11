// Package operation verifies lookup paths remain safe workspace-relative output hints.
package operation

import (
	"path/filepath"
	"testing"

	"semedit/internal/backend"
)

func TestNormalizeLookupResultPathsOmitsOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	result := &backend.LookupResult{
		File: filepath.Join(root, "src", "main.go"),
		Candidates: []*backend.SymbolCandidate{
			{Name: "Inside", Kind: "function", File: filepath.Join(root, "src", "inside.go")},
			{Name: "Outside", Kind: "function", File: filepath.Join(filepath.Dir(root), "outside.go")},
		},
		Ambiguous: true,
	}

	normalizeLookupResultPaths(root, result)

	if result.File != "src/main.go" {
		t.Errorf("lookup file = %q, want src/main.go", result.File)
	}
	if result.Candidates[0].File != "src/inside.go" {
		t.Errorf("inside candidate file = %q, want src/inside.go", result.Candidates[0].File)
	}
	if result.Candidates[1].File != "" {
		t.Errorf("outside candidate file = %q, want omitted", result.Candidates[1].File)
	}
	projected, err := projectLookupResult(result)
	if err != nil {
		t.Fatal(err)
	}
	candidates := projected.(map[string]any)["candidates"].([]map[string]any)
	if _, exposed := candidates[1]["file"]; exposed {
		t.Fatalf("outside candidate projection exposed a file: %#v", candidates[1])
	}
}
