// Package mcp tests workspace-scoped language snapshots and immutable catalog caching.

package mcp

import (
	"bytes"
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spockz/semantic-editor/internal/pipeline"

	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLanguageCatalogSnapshotsRefreshOnlyAtInitialize(t *testing.T) {
	root := t.TempDir()
	writeCatalogSource(t, root, "main.go")
	server := NewServer("full", root, io.Discard, WithToolProfile("full"))

	before, err := server.listTools()
	if err != nil {
		t.Fatal(err)
	}
	languages, err := server.activeLanguages()
	if err != nil || !reflect.DeepEqual(languages, []backend.LanguageID{backend.LanguageGo}) {
		t.Fatalf("initial languages = %v, %v", languages, err)
	}

	writeCatalogSource(t, root, "main.rs")
	frozen, err := server.listTools()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, frozen) {
		t.Fatal("tools/list changed after files changed without reinitialization")
	}
	if got, snapshotErr := server.activeLanguages(); snapshotErr != nil || !reflect.DeepEqual(got, []backend.LanguageID{backend.LanguageGo}) {
		t.Fatalf("language snapshot after file change = %v, %v", got, snapshotErr)
	}

	childRoot := filepath.Join(root, ".scratch", "worktrees", "rust")
	if mkdirErr := os.MkdirAll(childRoot, 0o700); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	writeCatalogSource(t, childRoot, "lib.rs")
	child := server.requestView(childRoot, false)
	if got, childErr := child.activeLanguages(); childErr != nil || !reflect.DeepEqual(got, []backend.LanguageID{backend.LanguageRust}) {
		t.Fatalf("child-root language snapshot = %v, %v", got, childErr)
	}
	if got, parentErr := server.activeLanguages(); parentErr != nil || !reflect.DeepEqual(got, []backend.LanguageID{backend.LanguageGo}) {
		t.Fatalf("parent-root snapshot changed after child discovery = %v, %v", got, parentErr)
	}

	if initializeErr := initializeCatalogRoot(t, server, root); initializeErr != nil {
		t.Fatalf("reinitialize same root: %v", initializeErr)
	}
	refreshed, err := server.listTools()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, refreshed) {
		t.Fatal("tools/list catalog did not refresh after successful reinitialization")
	}
	if got, err := server.activeLanguages(); err != nil || !reflect.DeepEqual(got, []backend.LanguageID{backend.LanguageGo, backend.LanguageRust}) {
		t.Fatalf("refreshed language snapshot = %v, %v", got, err)
	}
}

func TestFailedInitializePreservesLanguageAndCatalogState(t *testing.T) {
	root := t.TempDir()
	writeCatalogSource(t, root, "main.go")
	wrongRoot := t.TempDir()
	writeCatalogSource(t, wrongRoot, "lib.rs")
	server := NewServer("full", root, io.Discard, WithExpectedRoot(root), WithToolProfile("full"))

	before, err := server.listTools()
	if err != nil {
		t.Fatal(err)
	}
	if initializeErr := initializeCatalogRoot(t, server, root); initializeErr != nil {
		t.Fatalf("initial initialize: %v", initializeErr)
	}
	if initializeErr := initializeCatalogRoot(t, server, wrongRoot); initializeErr == nil {
		t.Fatal("Initialize accepted a root that violates WithExpectedRoot")
	}
	if server.workDir != backend.CanonicalWorkspaceRoot(root) {
		t.Fatalf("workDir after rejected initialize = %q, want %q", server.workDir, root)
	}
	after, err := server.listTools()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed Initialize changed the previously usable catalog")
	}
	if got, err := server.activeLanguages(); err != nil || !reflect.DeepEqual(got, []backend.LanguageID{backend.LanguageGo}) {
		t.Fatalf("language snapshot after failed initialize = %v, %v", got, err)
	}
	if err := initializeCatalogRoot(t, server, root); err != nil {
		t.Fatalf("valid reinitialize after rejected re-handshake: %v", err)
	}
}

func TestInjectedNilRegistryDoesNotFallBackToDefaults(t *testing.T) {
	var output bytes.Buffer
	server := NewServer("full", t.TempDir(), &output, WithRegistry(nil), WithToolProfile("full"))
	tools, err := server.listTools()
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if tool["name"] == "refactor_rename" || tool["name"] == "semantic_verify" {
			t.Fatalf("explicit nil registry exposed default operation %q", tool["name"])
		}
	}
	if len(tools) != 3 {
		t.Fatalf("explicit nil registry tool count = %d, want diagnostics, metrics, and feedback utilities only", len(tools))
	}
}

func TestCloneToolCatalogCopiesTypedNestedValues(t *testing.T) {
	original := []map[string]any{{
		"outputSchema": map[string]any{
			"properties": map[string]any{
				"result": map[string]any{
					"enum": []map[string]string{{"nested": "original"}},
				},
			},
		},
	}}
	cloned := cloneToolCatalog(original)
	clonedEnum := cloned[0]["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)["enum"].([]map[string]string)
	clonedEnum[0]["nested"] = "mutated"
	originalEnum := original[0]["outputSchema"].(map[string]any)["properties"].(map[string]any)["result"].(map[string]any)["enum"].([]map[string]string)
	if originalEnum[0]["nested"] != "original" {
		t.Fatalf("cloning catalog changed original nested enum value: %v", originalEnum)
	}
}

func TestCatalogCacheRetriesDetectionAndSharesConcurrentFills(t *testing.T) {
	cache := newCatalogCache()
	root := t.TempDir()
	var detectionCalls atomic.Int32
	if _, err := cache.languageSnapshot(root, func() ([]backend.LanguageID, error) {
		detectionCalls.Add(1)
		return nil, errors.New("temporary scan failure")
	}); err == nil {
		t.Fatal("language detection failure was cached as success")
	}

	const callers = 12
	var wait sync.WaitGroup
	failures := make(chan error, callers)
	for range callers {
		wait.Go(func() {
			languages, err := cache.languageSnapshot(root, func() ([]backend.LanguageID, error) {
				detectionCalls.Add(1)
				time.Sleep(time.Millisecond)
				return []backend.LanguageID{backend.LanguageGo}, nil
			})
			if err != nil {
				failures <- err
				return
			}
			if !reflect.DeepEqual(languages, []backend.LanguageID{backend.LanguageGo}) {
				failures <- fmt.Errorf("languages = %v", languages)
			}
		})
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if got := detectionCalls.Load(); got != 2 {
		t.Fatalf("detection attempts = %d, want one failure and one shared successful fill", got)
	}

	var builds atomic.Int32
	buildFailures := make(chan error, callers)
	for range callers {
		wait.Go(func() {
			catalog, err := cache.toolCatalog(root, func() ([]map[string]any, error) {
				builds.Add(1)
				time.Sleep(time.Millisecond)
				return []map[string]any{{"name": "one"}}, nil
			})
			if err != nil || len(catalog) != 1 || catalog[0]["name"] != "one" {
				buildFailures <- fmt.Errorf("catalog = %#v, error = %w", catalog, err)
			}
		})
	}
	wait.Wait()
	close(buildFailures)
	for err := range buildFailures {
		t.Error(err)
	}
	if got := builds.Load(); got != 1 {
		t.Fatalf("catalog builds = %d, want one shared build", got)
	}
}

func TestLazyLanguageSnapshotMovesToInitializedRoot(t *testing.T) {
	initialRoot := t.TempDir()
	selectedRoot := t.TempDir()
	writeCatalogSource(t, initialRoot, "main.go")
	writeCatalogSource(t, selectedRoot, "lib.rs")
	server := NewServer("full", initialRoot, io.Discard)

	if languages, err := server.activeLanguages(); err != nil || !reflect.DeepEqual(languages, []backend.LanguageID{backend.LanguageGo}) {
		t.Fatalf("pre-initialize language snapshot = %v, %v", languages, err)
	}
	if err := initializeCatalogRoot(t, server, selectedRoot); err != nil {
		t.Fatalf("Initialize selected root: %v", err)
	}
	if server.workDir != backend.CanonicalWorkspaceRoot(selectedRoot) {
		t.Fatalf("initialized root = %q, want %q", server.workDir, selectedRoot)
	}
	if languages, err := server.activeLanguages(); err != nil || !reflect.DeepEqual(languages, []backend.LanguageID{backend.LanguageRust}) {
		t.Fatalf("post-initialize language snapshot = %v, %v", languages, err)
	}
}

func TestLanguageDetectionRetriesAfterFilesystemRecovery(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "workspace")
	server := NewServer("full", missingRoot, io.Discard)
	if _, err := server.activeLanguages(); err == nil {
		t.Fatal("language detection succeeded for a missing workspace")
	}
	writeCatalogSource(t, missingRoot, "main.go")
	if languages, err := server.activeLanguages(); err != nil || !reflect.DeepEqual(languages, []backend.LanguageID{backend.LanguageGo}) {
		t.Fatalf("language detection after workspace recovery = %v, %v", languages, err)
	}
}

func TestCatalogResetInvalidatesToolsAndSeedsNewRootLanguages(t *testing.T) {
	cache := newCatalogCache()
	root := t.TempDir()
	cache.reset(root, []backend.LanguageID{backend.LanguageGo})
	first, err := cache.toolCatalog(root, func() ([]map[string]any, error) {
		return []map[string]any{{"name": "go"}}, nil
	})
	if err != nil || first[0]["name"] != "go" {
		t.Fatalf("initial catalog = %#v, %v", first, err)
	}
	cache.reset(root, []backend.LanguageID{backend.LanguageRust})
	second, err := cache.toolCatalog(root, func() ([]map[string]any, error) {
		return []map[string]any{{"name": "rust"}}, nil
	})
	if err != nil || second[0]["name"] != "rust" {
		t.Fatalf("catalog after reset = %#v, %v", second, err)
	}
	languages, err := cache.languageSnapshot(root, func() ([]backend.LanguageID, error) {
		return nil, errors.New("seeded snapshot should not detect")
	})
	if err != nil || !reflect.DeepEqual(languages, []backend.LanguageID{backend.LanguageRust}) {
		t.Fatalf("language snapshot after reset = %v, %v", languages, err)
	}
}

func TestCatalogBuildRejectsRootFromSupersededGeneration(t *testing.T) {
	cache := newCatalogCache()
	oldRoot, newRoot := t.TempDir(), t.TempDir()
	generation := cache.currentGeneration()
	started, resume := make(chan struct{}), make(chan struct{})
	type buildResult struct {
		tools []map[string]any
		err   error
	}
	result := make(chan buildResult, 1)
	go func() {
		tools, err := cache.toolCatalogForGeneration(oldRoot, generation, func() ([]map[string]any, error) {
			close(started)
			<-resume
			return []map[string]any{{"name": "old-root"}}, nil
		})
		result <- buildResult{tools: tools, err: err}
	}()
	<-started
	cache.reset(newRoot, []backend.LanguageID{backend.LanguageRust})
	close(resume)
	stale := <-result
	if !errors.Is(stale.err, errCatalogGenerationChanged) || len(stale.tools) != 0 {
		t.Fatalf("superseded build = %#v, want generation-changed error", stale)
	}
	current, err := cache.toolCatalogForGeneration(newRoot, cache.currentGeneration(), func() ([]map[string]any, error) {
		return []map[string]any{{"name": "new-root"}}, nil
	})
	if err != nil || current[0]["name"] != "new-root" {
		t.Fatalf("catalog after root reset = %#v, %v", current, err)
	}
}

func TestCatalogToolsRetriesWhenInitializeChangesRootDuringBuild(t *testing.T) {
	oldRoot, newRoot := t.TempDir(), t.TempDir()
	writeCatalogSource(t, oldRoot, "main.go")
	writeCatalogSource(t, newRoot, "lib.rs")
	started, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	registry := operation.NewRegistry()
	handler := func(_ context.Context, _ operation.CallContext, _ map[string]any) (string, error) {
		return "called", nil
	}
	definition := operation.Def[map[string]any, string]{
		Key: "catalog-probe", Summary: "Probe catalog generation", MCPName: "semantic_catalog_probe", ReadOnly: true,
		Params: []operation.ParameterContract{{
			Name: "mode", CLIName: "mode", JSONName: "mode", Type: operation.ParamString,
			DynamicEnums: func(backend.Backend) []string {
				once.Do(func() {
					close(started)
					<-resume
				})
				return []string{"ready"}
			},
		}},
		Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){
			backend.LanguageGo: handler,
		},
		Format: func(result string) (string, error) { return result, nil },
	}
	if err := operation.Register(registry, definition); err != nil {
		t.Fatal(err)
	}
	server := NewServer("full", oldRoot, io.Discard, WithRegistry(registry), WithToolProfile("full"))
	if err := initializeCatalogRoot(t, server, oldRoot); err != nil {
		t.Fatal(err)
	}

	type catalogResult struct {
		tools []map[string]any
		err   error
	}
	result := make(chan catalogResult, 1)
	go func() {
		tools, err := server.catalogTools()
		result <- catalogResult{tools: tools, err: err}
	}()
	<-started
	if err := initializeCatalogRoot(t, server, newRoot); err != nil {
		close(resume)
		t.Fatalf("initialize new root during catalog build: %v", err)
	}
	close(resume)
	built := <-result
	if built.err != nil {
		t.Fatalf("tools/list after reinitialization: %v", built.err)
	}
	for _, tool := range built.tools {
		if tool["name"] == "semantic_catalog_probe" {
			t.Fatal("tools/list returned old-root Go catalog after reinitialization to Rust")
		}
	}
}

func writeCatalogSource(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(path, nil); err != nil {
		t.Fatal(err)
	}
}

func initializeCatalogRoot(t *testing.T, server *Server, root string) error {
	t.Helper()
	params, err := json.Marshal(map[string]any{"rootPath": root})
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.Initialize(t.Context(), params)
	return err
}
