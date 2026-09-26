// Package gobackend verifies public Go reference-analysis contracts at project boundaries.
package gobackend

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	backend "semedit/internal/backend"
	"strings"
	"testing"
	"time"
)

func TestFindReferencesHonorsReadOnlyModuleFlags(t *testing.T) {
	root := writeReferenceFixture(t, map[string]string{
		"go.mod":     "module example.com/read-only-refs\n\ngo 1.23\n\nrequire github.com/BurntSushi/toml v1.6.0\n",
		"go.sum":     "github.com/rogpeppe/go-internal v1.16.0 h1:O9DK+vNMDVGLr2BeZqmpLeMjiMNkuXfcqntWbZV6S5g=\n",
		"api/api.go": "package api\nimport \"github.com/BurntSushi/toml\"\ntype Config struct { Name string }\nfunc Decode(data []byte) (Config, error) { var config Config; err := toml.Unmarshal(data, &config); return config, err }\n",
	})
	cacheOutput, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("read Go module cache: %v", err)
	}
	cachedModule := filepath.Join(strings.TrimSpace(string(cacheOutput)), "github.com", "!burnt!sushi", "toml@v1.6.0")
	if _, err := os.Stat(cachedModule); os.IsNotExist(err) {
		t.Skipf("BurntSushi/toml v1.6.0 is not cached at %s", cachedModule)
	} else if err != nil {
		t.Fatalf("stat cached module: %v", err)
	}

	t.Setenv("GOFLAGS", "-mod=mod")
	modulePath := filepath.Join(root, "go.mod")
	sumPath := filepath.Join(root, "go.sum")
	oldTime := time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC)
	for _, path := range []string{modulePath, sumPath} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("set manifest timestamp: %v", err)
		}
	}
	type manifestState struct {
		path    string
		content []byte
		modTime time.Time
	}
	before := make([]manifestState, 0, 2)
	for _, path := range []string{modulePath, sumPath} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read manifest before request: %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat manifest before request: %v", err)
		}
		before = append(before, manifestState{path: path, content: content, modTime: info.ModTime()})
	}

	_, err = (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: root, Language: backend.LanguageGo},
		Symbol:  "Decode",
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "sum") {
		t.Fatalf("FindReferences() error = %v, want missing-sum failure under read-only module loading", err)
	}
	for _, expected := range before {
		content, err := os.ReadFile(expected.path)
		if err != nil {
			t.Errorf("read manifest after request: %v", err)
			continue
		}
		info, err := os.Stat(expected.path)
		if err != nil {
			t.Errorf("stat manifest after request: %v", err)
			continue
		}
		if !bytes.Equal(content, expected.content) || !info.ModTime().Equal(expected.modTime) {
			t.Errorf("FindReferences changed manifest %s: bytes_equal=%t mtime=%s want=%s", expected.path, bytes.Equal(content, expected.content), info.ModTime(), expected.modTime)
		}
	}
	for _, path := range []string{filepath.Join(root, "go.work"), filepath.Join(root, "go.work.sum")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("FindReferences created workspace manifest %s: stat error = %v", path, err)
		}
	}
}

func TestFindReferencesPreservesNestedModuleAndProjectRelativePaths(t *testing.T) {
	repositoryRoot := writeReferenceFixture(t, map[string]string{
		"nested/go.mod":      "module example.com/nestedrefs\n\ngo 1.23\n",
		"nested/api/a.go":    "package api\nfunc Read() {}\n",
		"nested/client/b.go": "package client\nimport \"example.com/nestedrefs/api\"\nfunc Use() { api.Read() }\n",
	})
	result, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: repositoryRoot, File: "nested/api/a.go", Language: backend.LanguageGo},
		Symbol:  "Read",
	})
	if err != nil {
		t.Fatalf("FindReferences from repository root: %v", err)
	}
	if result.Scope.Path != "nested" || result.Symbol.File != "nested/api/a.go" {
		t.Fatalf("nested module scope and symbol paths = %#v / %q", result.Scope, result.Symbol.File)
	}
	if len(result.References) != 1 || result.References[0].File != "nested/client/b.go" || result.References[0].EnclosingSymbol != "Use" || result.References[0].Snippet != "func Use() { api.Read() }" {
		t.Fatalf("nested module references = %#v", result.References)
	}
	nestedFiles := make(map[string]bool)
	for _, file := range result.Files {
		nestedFiles[file.File] = true
	}
	if !reflect.DeepEqual(nestedFiles, map[string]bool{"nested/api/a.go": true, "nested/client/b.go": true}) {
		t.Fatalf("nested module file revisions = %#v", nestedFiles)
	}

	moduleRoot := writeReferenceFixture(t, map[string]string{
		"go.mod":         "module example.com/projectrootrefs\n\ngo 1.23\n",
		"client/read.go": "package client\nfunc Read() {}\n",
		"api/use.go":     "package api\nimport \"example.com/projectrootrefs/client\"\nfunc Use() { client.Read() }\n",
	})
	projectRoot := filepath.Join(moduleRoot, "client")
	fromSubdirectory, err := (GoBackend{}).FindReferences(context.Background(), backend.ReferencesRequest{
		Project: backend.ProjectContext{RootDir: projectRoot, File: "read.go", Language: backend.LanguageGo},
		Symbol:  "Read",
	})
	if err != nil {
		t.Fatalf("FindReferences from module subdirectory: %v", err)
	}
	if fromSubdirectory.Scope.Path != moduleRoot || fromSubdirectory.Symbol.File != "read.go" {
		t.Fatalf("subdirectory scope and symbol paths = %#v / %q, want absolute module root and project-relative declaration", fromSubdirectory.Scope, fromSubdirectory.Symbol.File)
	}
	absoluteUse := filepath.Join(moduleRoot, "api", "use.go")
	if len(fromSubdirectory.References) != 1 || fromSubdirectory.References[0].File != absoluteUse || fromSubdirectory.References[0].EnclosingSymbol != "Use" || fromSubdirectory.References[0].Snippet != "func Use() { client.Read() }" {
		t.Fatalf("references outside project root = %#v, want absolute file %s", fromSubdirectory.References, absoluteUse)
	}
	subdirectoryFiles := make(map[string]bool)
	for _, file := range fromSubdirectory.Files {
		subdirectoryFiles[file.File] = true
	}
	if !reflect.DeepEqual(subdirectoryFiles, map[string]bool{"read.go": true, absoluteUse: true}) {
		t.Fatalf("subdirectory file revisions = %#v", subdirectoryFiles)
	}
}
