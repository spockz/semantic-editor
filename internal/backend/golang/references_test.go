// Package gobackend tests typed reference results across module, generic, and snapshot boundaries.
package gobackend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	backend "semedit/internal/backend"
	"semedit/internal/symbol"
	"strconv"
	"strings"
	"testing"
)

func TestFindReferencesSearchesModuleFromSelectedFile(t *testing.T) {
	root := writeReferenceFixture(t, map[string]string{
		"go.mod":           "module example.com/references\ngo 1.23\n",
		"api/store.go":     "package api\ntype Store struct{}\nfunc (*Store) Read() string { return \"\" }\n",
		"client/client.go": "package client\nimport \"example.com/references/api\"\nfunc Use(s *api.Store) { _ = s.Read() }\n",
	})
	result, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: root, File: "api/store.go", Language: backend.LanguageGo},
		Symbol:  "Store.Read",
	})
	if err != nil {
		t.Fatalf("FindReferences() error = %v", err)
	}
	if result.Symbol.File != "api/store.go" || result.Symbol.QualifiedName != "Store.Read" {
		t.Errorf("resolved symbol = %#v", result.Symbol)
	}
	if result.Scope.Kind != "workspace_active_build" || result.Scope.Path != "." || !result.Scope.Complete {
		t.Errorf("scope = %#v", result.Scope)
	}
	if len(result.References) != 1 {
		t.Fatalf("references = %#v, want one cross-package use", result.References)
	}
	if reference := result.References[0]; reference.File != "client/client.go" || reference.EnclosingSymbol != "Use" || reference.Snippet != "func Use(s *api.Store) { _ = s.Read() }" {
		t.Errorf("reference = %#v", reference)
	}
	foundSource, foundClient := false, false
	for _, file := range result.Files {
		foundSource = foundSource || file.File == "api/store.go"
		foundClient = foundClient || file.File == "client/client.go"
	}
	if !foundSource || !foundClient {
		t.Errorf("file revisions = %#v, want API and client source", result.Files)
	}
}

func TestFindReferencesNormalizesGenericOriginsAcrossTestVariants(t *testing.T) {
	root := writeReferenceFixture(t, map[string]string{
		"go.mod":                  "module example.com/genericrefs\ngo 1.23\n",
		"sample/types.go":         "package sample\ntype Box[T any] struct { Value T }\nfunc (b Box[T]) Echo(v T) T { return v }\ntype Other[T any] struct { Value T }\nfunc (o Other[T]) Echo(v T) T { return v }\nfunc Apply[T any](v T) T { return v }\nfunc Use() { b := Box[int]{}; o := Other[int]{}; _ = b.Value; _ = b.Echo(1); _ = o.Value; _ = o.Echo(1); _ = Apply(1) }\n",
		"sample/types_test.go":    "package sample\nimport \"testing\"\nfunc TestApplyInternal(t *testing.T) { _ = Apply(2) }\n",
		"sample/external_test.go": "package sample_test\nimport (\"testing\"; \"example.com/genericrefs/sample\")\nfunc TestApplyExternal(t *testing.T) { _ = sample.Apply(3) }\n",
		"other/other.go":          "package other\ntype Box[T any] struct { Value T }\nfunc (b Box[T]) Echo(v T) T { return v }\nfunc Apply[T any](v T) T { return v }\nfunc UseOther() { b := Box[int]{}; _ = b.Value; _ = b.Echo(1); _ = Apply(1) }\n",
	})
	want := map[string]map[string]int{
		"Box.Value": {"sample/types.go": 1},
		"Box.Echo":  {"sample/types.go": 1},
		"Apply":     {"sample/types.go": 1, "sample/types_test.go": 1, "sample/external_test.go": 1},
	}
	for query, wantFiles := range want {
		result, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
			Project: backend.ProjectContext{RootDir: root, File: "sample/types.go", Language: backend.LanguageGo},
			Symbol:  query,
		})
		if err != nil {
			t.Fatalf("FindReferences(%q) error = %v", query, err)
		}
		gotFiles := make(map[string]int)
		seen := make(map[string]bool)
		for _, reference := range result.References {
			key := reference.File + ":" + strconv.Itoa(reference.Range.Start.Line) + ":" + strconv.Itoa(reference.Range.Start.Character)
			if seen[key] {
				t.Errorf("FindReferences(%q) duplicated physical range %s", query, key)
			}
			seen[key] = true
			gotFiles[reference.File]++
		}
		if !reflect.DeepEqual(gotFiles, wantFiles) {
			t.Errorf("FindReferences(%q) reference sites = %#v, want %#v", query, gotFiles, wantFiles)
		}
	}
}

func writeReferenceFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, contents := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir fixture path: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	return root
}

func TestFindReferencesRespectsQualifiedMethodFileScope(t *testing.T) {
	root := writeReferenceFixture(t, map[string]string{
		"go.mod":           "module example.com/scopedrefs\ngo 1.23\n",
		"api/types.go":     "package api\ntype T struct{}\ntype Inner struct{}\ntype Promoted struct { Inner }\n",
		"api/methods.go":   "package api\nfunc (T) M() {}\nfunc (Inner) M() {}\n",
		"api/base.go":      "package api\ntype Base interface { Read() }\n",
		"api/outer.go":     "package api\ntype Outer interface { Base }\n",
		"client/client.go": "package client\nimport \"example.com/scopedrefs/api\"\nfunc Use(v api.T) { v.M() }\n",
	})
	backendValue := GoBackend{}
	accepted := []struct {
		query string
		file  string
	}{
		{query: "T.M", file: "api/methods.go"},
		{query: "Outer.Read", file: "api/outer.go"},
	}
	for _, test := range accepted {
		result, err := backendValue.FindReferences(context.Background(), backend.ReferencesRequest{
			Project: backend.ProjectContext{RootDir: root, File: test.file, Language: backend.LanguageGo},
			Symbol:  test.query,
		})
		if err != nil {
			t.Errorf("FindReferences(%q, %q) error = %v", test.query, test.file, err)
			continue
		}
		if result.Symbol.QualifiedName != test.query {
			t.Errorf("FindReferences(%q) symbol = %#v", test.query, result.Symbol)
		}
		if test.query == "Outer.Read" && (result.References == nil || len(result.References) != 0) {
			t.Errorf("unused inherited interface method references = %#v, want non-nil empty", result.References)
		}
	}
	for _, test := range []struct {
		query string
		file  string
	}{
		{query: "T.M", file: "api/types.go"},
		{query: "Outer.Read", file: "api/base.go"},
		{query: "Promoted.M", file: "api/methods.go"},
	} {
		_, err := backendValue.FindReferences(context.Background(), backend.ReferencesRequest{
			Project: backend.ProjectContext{RootDir: root, File: test.file, Language: backend.LanguageGo},
			Symbol:  test.query,
		})
		if !errors.Is(err, symbol.ErrNotFound) {
			t.Errorf("FindReferences(%q, %q) error = %v, want ErrNotFound", test.query, test.file, err)
		}
	}
}

func TestFindReferencesUsesOnlyOriginalActiveSources(t *testing.T) {
	root := writeReferenceFixture(t, map[string]string{
		"go.mod":               "module example.com/activesources\ngo 1.23\n",
		"api/api.go":           "package api\nfunc Read() {}\n",
		"api/active_use.go":    "//go:build active\n\npackage api\nfunc UseActive() { Read() }\n",
		"api/inactive.go":      "//go:build !active\n\npackage api\nfunc ThisIsNotValid( {}\n",
		"api/api_test.go":      "package api\nimport \"testing\"\nfunc TestInternal(t *testing.T) { Read() }\n",
		"api/external_test.go": "package api_test\nimport (\"testing\"; \"example.com/activesources/api\")\nfunc TestExternal(t *testing.T) { api.Read() }\n",
		"client/client.go":     "package client\nimport \"example.com/activesources/api\"\nfunc Use() { api.Read() }\n",
	})
	cache := filepath.Join(root, ".scratch", "go-cache")
	temp := filepath.Join(root, ".scratch", "go-tmp")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(temp, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOCACHE", cache)
	t.Setenv("GOTMPDIR", temp)
	t.Setenv("GOFLAGS", "-tags=active -mod=mod")
	moduleBefore, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: root, Language: backend.LanguageGo},
		Symbol:  "Read",
	})
	if err != nil {
		t.Fatalf("FindReferences() error = %v", err)
	}
	wantFiles := map[string]bool{
		"api/api.go": true, "api/active_use.go": true, "api/api_test.go": true,
		"api/external_test.go": true, "client/client.go": true,
	}
	gotFiles := make(map[string]bool)
	for _, file := range result.Files {
		gotFiles[file.File] = true
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Errorf("revision sources = %#v, want %#v", gotFiles, wantFiles)
	}
	for _, file := range result.Files {
		if strings.Contains(file.File, ".scratch") {
			t.Errorf("compiler cache file escaped into revisions: %q", file.File)
		}
	}
	gotReferences := make(map[string]bool)
	for _, reference := range result.References {
		gotReferences[reference.File] = true
	}
	wantReferences := map[string]bool{
		"api/active_use.go": true, "api/api_test.go": true,
		"api/external_test.go": true, "client/client.go": true,
	}
	if !reflect.DeepEqual(gotReferences, wantReferences) {
		t.Errorf("references = %#v, want %#v", gotReferences, wantReferences)
	}
	moduleAfter, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(moduleBefore, moduleAfter) {
		t.Error("FindReferences changed go.mod")
	}
	if _, err := os.Stat(filepath.Join(root, "go.work")); !os.IsNotExist(err) {
		t.Errorf("FindReferences created go.work: stat error = %v", err)
	}
}

func TestFindReferencesRejectsActiveWorkspace(t *testing.T) {
	root := writeReferenceFixture(t, map[string]string{
		"go.mod": "module example.com/workspacerefs\ngo 1.23\n",
		"api.go": "package workspacerefs\nfunc Read() {}\n",
	})
	workspace := filepath.Join(root, "go.work")
	t.Setenv("GOWORK", workspace)
	_, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: root, Language: backend.LanguageGo},
		Symbol:  "Read",
	})
	if err == nil || !strings.Contains(err.Error(), "active go.work is unsupported") {
		t.Errorf("FindReferences with active workspace error = %v", err)
	}
	if _, statErr := os.Stat(workspace); !os.IsNotExist(statErr) {
		t.Errorf("FindReferences created go.work: stat error = %v", statErr)
	}
}

func TestFindReferencesRejectsCgoTransformedSources(t *testing.T) {
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("C compiler is unavailable")
	}
	root := writeReferenceFixture(t, map[string]string{
		"go.mod":        "module example.com/cgorefs\ngo 1.23\n",
		"api/cgo.go":    "package api\nimport \"C\"\nfunc Read() {}\n",
		"client/use.go": "package client\nimport \"example.com/cgorefs/api\"\nfunc Use() { api.Read() }\n",
	})
	t.Setenv("CGO_ENABLED", "1")
	cache := filepath.Join(root, ".scratch", "go-cache")
	temp := filepath.Join(root, ".scratch", "go-tmp")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(temp, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOCACHE", cache)
	t.Setenv("GOTMPDIR", temp)
	_, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: root, Language: backend.LanguageGo},
		Symbol:  "Read",
	})
	if err == nil || !strings.Contains(err.Error(), "transformed or generated compiler input") {
		t.Errorf("FindReferences() error = %v, want explicit transformed-source limitation", err)
	}
}

func TestFindReferencesResolvesTypedBindingsAndPhysicalRanges(t *testing.T) {
	source := "package localrefs\nvar Canonical int\nfunc Uses() {\n\tCanonical++\n\tCanonical, introduced := 2, 3; _ = Canonical\n\t_ = introduced\n\titem := 1\n\titem, added := item+1, 3\n\t_ = added\n\t_ = item\n\tconst Label = \"ok\"\n\t_ = Label\n\tshadow := 4\n\t{\n\t\tshadow := \"nested\"\n\t\t_ = shadow\n\t}\n\t_ = shadow\n}\nfunc Physical() {\n\t_ = \"é\"; Canonical = 8\n//line virtual.go:200\n\tCanonical = 9\n}\nfunc Unused() {}\n"
	root := writeReferenceFixture(t, map[string]string{
		"go.mod": "module example.com/localrefs\ngo 1.23\n",
		"api.go": source,
	})
	backendValue := GoBackend{}
	request := func(name string) backend.ReferencesRequest {
		return backend.ReferencesRequest{
			Project: backend.ProjectContext{RootDir: root, File: "api.go", Language: backend.LanguageGo},
			Symbol:  name,
		}
	}
	canonical, err := backendValue.FindReferences(context.Background(), request("Canonical"))
	if err != nil {
		t.Fatalf("FindReferences(Canonical) error = %v", err)
	}
	if canonical.Symbol.Kind != "variable" || canonical.Symbol.File != "api.go" {
		t.Errorf("canonical symbol = %#v", canonical.Symbol)
	}
	wantRanges := []backend.Range{
		{Start: backend.Position{Line: 3, Character: 1}, End: backend.Position{Line: 3, Character: 10}},
		{Start: backend.Position{Line: 20, Character: 10}, End: backend.Position{Line: 20, Character: 19}},
		{Start: backend.Position{Line: 22, Character: 1}, End: backend.Position{Line: 22, Character: 10}},
	}
	if len(canonical.References) != len(wantRanges) {
		t.Fatalf("Canonical references = %#v, want %d physical uses", canonical.References, len(wantRanges))
	}
	for index, reference := range canonical.References {
		if reference.File != "api.go" || reference.Range != wantRanges[index] {
			t.Errorf("Canonical reference[%d] = %#v, want range %#v", index, reference, wantRanges[index])
		}
	}
	digest := sha256.Sum256([]byte(source))
	wantRevision := hex.EncodeToString(digest[:])
	if len(canonical.Files) != 1 || canonical.Files[0].Revision != wantRevision {
		t.Errorf("snapshot revisions = %#v, want api.go SHA-256 %s", canonical.Files, wantRevision)
	}

	reused, err := backendValue.FindReferences(context.Background(), request("item"))
	if err != nil {
		t.Fatalf("FindReferences(reused := binding) error = %v", err)
	}
	if reused.Symbol.Kind != "variable" || len(reused.References) != 3 {
		t.Errorf("reused binding result = symbol %#v, references %#v", reused.Symbol, reused.References)
	}

	localConstant, err := backendValue.FindReferences(context.Background(), request("Label"))
	if err != nil {
		t.Fatalf("FindReferences(local const) error = %v", err)
	}
	if localConstant.Symbol.Kind != "constant" || len(localConstant.References) != 1 {
		t.Errorf("local const result = symbol %#v, references %#v", localConstant.Symbol, localConstant.References)
	}

	unused, err := backendValue.FindReferences(context.Background(), request("Unused"))
	if err != nil {
		t.Fatalf("FindReferences(unused declaration) error = %v", err)
	}
	if unused.Symbol.Name != "Unused" || unused.References == nil || len(unused.References) != 0 {
		t.Errorf("unused declaration result = symbol %#v, references %#v", unused.Symbol, unused.References)
	}
	if _, err := backendValue.FindReferences(context.Background(), request("shadow")); !errors.Is(err, backend.ErrAmbiguous) {
		t.Errorf("FindReferences(shadowed local) error = %v, want ambiguity", err)
	}
	if _, err := backendValue.FindReferences(context.Background(), request("Missing")); !errors.Is(err, symbol.ErrNotFound) {
		t.Errorf("FindReferences(missing) error = %v, want not-found", err)
	}
}

func TestReferenceSourceSnapshotDetectsConcurrentChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source.go")
	original := []byte("package snapshot\nfunc A() {}\n")
	changed := []byte("package snapshot\nfunc B() {}\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := &referenceSourceSnapshot{files: make(map[string][]byte)}
	for range 2 {
		if _, err := snapshot.parseFile(token.NewFileSet(), path, original); err != nil {
			t.Fatalf("parse same captured bytes: %v", err)
		}
	}
	if err := snapshot.verifyUnchanged(); err != nil {
		t.Fatalf("verify stable source: %v", err)
	}
	tempPath := filepath.Join(root, "replacement.tmp")
	if err := os.WriteFile(tempPath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.verifyUnchanged(); err == nil || !strings.Contains(err.Error(), "source changed during reference analysis") {
		t.Errorf("verify changed source error = %v", err)
	}

	overlap := &referenceSourceSnapshot{files: make(map[string][]byte)}
	if _, err := overlap.parseFile(token.NewFileSet(), path, original); err != nil {
		t.Fatalf("parse initial overlapping source: %v", err)
	}
	if _, err := overlap.parseFile(token.NewFileSet(), path, changed); err == nil || !strings.Contains(err.Error(), "source changed while packages were loading") {
		t.Errorf("parse changed bytes for one path error = %v", err)
	}
}
