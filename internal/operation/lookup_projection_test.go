// Package operation tests workspace-safe wire projections for semantic lookup.
package operation

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

func TestLookupProjectionOmitsOutsideWorkspacePaths(t *testing.T) {
	root := t.TempDir()
	result := &backend.LookupResult{Candidates: []*backend.NamedTarget{
		{Name: "Inside", QualifiedName: "Inside", Kind: "function", Location: backend.SourceLocation{URI: pathutil.FileURI(filepath.Join(root, "src", "inside.go")), Range: backend.Range{Start: backend.Position{Line: 2, Character: 23}, End: backend.Position{Line: 2, Character: 28}}}},
		{Name: "Outside", QualifiedName: "Outside", Kind: "function", Location: backend.SourceLocation{URI: pathutil.FileURI(filepath.Join(filepath.Dir(root), "outside.go")), Range: backend.Range{Start: backend.Position{Line: 0, Character: 0}, End: backend.Position{Line: 0, Character: 7}}}},
	}}

	response, err := lookupResponse(root, "Run", result)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Ambiguous || len(response.Candidates) != 2 {
		t.Fatalf("lookup response = %#v", response)
	}
	if response.Candidates[0].File != "src/inside.go" || response.Candidates[0].Line != 3 || response.Candidates[0].Column != 24 {
		t.Errorf("inside candidate = %#v, want src/inside.go:3:24", response.Candidates[0])
	}
	if response.Candidates[1].File != "" {
		t.Errorf("outside candidate file = %q, want omitted", response.Candidates[1].File)
	}
	projected, err := projectLookupResult(response)
	if err != nil {
		t.Fatal(err)
	}
	candidates := projected.(map[string]any)["candidates"].([]LookupTarget)
	if candidates[1].File != "" {
		t.Fatalf("outside candidate projection exposed a file: %#v", candidates[1])
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "file://") || strings.Contains(string(encoded), filepath.Dir(root)) {
		t.Fatalf("public lookup projection exposed a URI or outside path: %s", encoded)
	}
}

func TestLookupProjectionRejectsMalformedBackendStates(t *testing.T) {
	root := t.TempDir()
	target := &backend.NamedTarget{Name: "Run", Kind: "function", Location: backend.SourceLocation{URI: pathutil.FileURI(filepath.Join(root, "run.go")), Range: backend.Range{Start: backend.Position{Line: 0, Character: 0}, End: backend.Position{Line: 0, Character: 3}}}}
	definitionAndCandidate := &backend.LookupResult{Definition: target, Candidates: []*backend.NamedTarget{target}}
	for name, result := range map[string]*backend.LookupResult{
		"empty":                      {},
		"one candidate":              {Candidates: []*backend.NamedTarget{target}},
		"definition with candidates": definitionAndCandidate,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := lookupResponse(root, "Run", result); err == nil {
				t.Fatal("lookupResponse succeeded for malformed backend state")
			}
		})
	}
	response, err := lookupResponse(root, "Run", &backend.LookupResult{Definition: target})
	if err != nil {
		t.Fatal(err)
	}
	if response.Symbol != "Run" {
		t.Fatalf("fallback identity = %q, want Run", response.Symbol)
	}
	invalid := *target
	invalid.Location.Range.End = invalid.Location.Range.Start
	if _, err := lookupResponse(root, "Run", &backend.LookupResult{Definition: &invalid}); err == nil {
		t.Fatal("lookupResponse accepted an empty target range")
	}
}
