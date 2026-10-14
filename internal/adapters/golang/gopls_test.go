// Package golang tests cache selection at the subprocess boundary so rename tooling remains usable in restricted workspaces.
package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/atomicfile"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/lsp"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"slices"
)

func TestCommandEnvUsesOptInWorkspaceLocalGoState(t *testing.T) {
	root := t.TempDir()
	for key, value := range map[string]string{
		"GOENV":      filepath.Join(t.TempDir(), "env"),
		"GOCACHE":    t.TempDir(),
		"GOMODCACHE": t.TempDir(),
		"GOTMPDIR":   t.TempDir(),
		"GOBIN":      t.TempDir(),
		"GOPATH":     t.TempDir(),
		"GOFLAGS":    "-mod=readonly",
		"GOWORK":     filepath.Join(t.TempDir(), "go.work"),
	} {
		t.Setenv(key, value)
	}
	baseDir := filepath.Join(root, ".scratch", "go")
	env, err := gocache.Environment(gocache.WithBaseDir(t.Context(), baseDir), root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"GOENV":      filepath.Join(baseDir, "env"),
		"GOCACHE":    filepath.Join(baseDir, "build"),
		"GOMODCACHE": filepath.Join(baseDir, "mod"),
		"GOTMPDIR":   filepath.Join(baseDir, "tmp"),
		"GOBIN":      filepath.Join(baseDir, "bin"),
		"GOPATH":     baseDir,
		"GOFLAGS":    "-mod=readonly",
		"GOWORK":     "off",
	}
	for key, path := range want {
		if got := envValue(env, key); got != path {
			t.Errorf("%s = %q, want %q", key, got, path)
		}
	}
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOTMPDIR", "GOBIN", "GOPATH"} {
		if _, err := os.Stat(want[key]); err != nil {
			t.Errorf("%s directory was not created: %v", key, err)
		}
	}
}

func TestCommandEnvSupportsAmbientStateWithoutWorkspace(t *testing.T) {
	if _, err := gocache.Environment(t.Context(), ""); err != nil {
		t.Fatalf("Environment(\"\") failed in ambient mode: %v", err)
	}
}

func TestCommandEnvUsesConfiguredBaseDir(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(t.TempDir(), "go-state")
	env, err := gocache.Environment(gocache.WithBaseDir(t.Context(), baseDir), root)
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "GOCACHE"); got != filepath.Join(baseDir, "build") {
		t.Errorf("GOCACHE = %q, want %q", got, filepath.Join(baseDir, "build"))
	}
}

func TestGoRenameProtocolHelper(t *testing.T) {
	if os.Getenv("SEMEDIT_GO_RENAME_HELPER") != "1" {
		return
	}
	reader := lsp.NewFrameReader(os.Stdin, 0)
	writer := lsp.NewFrameWriter(os.Stdout, 0)
	methods := make([]string, 0)
	for {
		messageData, readErr := reader.ReadMessage()
		if readErr != nil {
			os.Exit(1)
		}
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if decodeErr := json.Unmarshal(messageData, &request); decodeErr != nil {
			os.Exit(1)
		}
		if len(request.ID) == 0 {
			methods = append(methods, request.Method)
			if path := os.Getenv("SEMEDIT_GO_RENAME_LOG"); path != "" {
				logData, marshalErr := json.Marshal(methods)
				if marshalErr != nil {
					os.Exit(1)
				}
				if writeErr := atomicfile.WriteAtomic(path, logData); writeErr != nil {
					os.Exit(1)
				}
			}
			if request.Method == "exit" {
				os.Exit(0)
			}
			continue
		}
		result := json.RawMessage("null")
		switch request.Method {
		case "initialize":
			result = json.RawMessage(`{"capabilities":{"diagnosticProvider":{},"renameProvider":{"prepareProvider":true},"positionEncoding":"utf-16"}}`)
		case "textDocument/prepareRename":
			result = json.RawMessage(`{"range":{"start":{"line":2,"character":5},"end":{"line":2,"character":10}},"placeholder":"Value"}`)
		case "textDocument/rename":
			result = json.RawMessage(os.Getenv("SEMEDIT_GO_RENAME_EDIT"))
		case "barrier", "shutdown":
		default:
			os.Exit(1)
		}
		messageData, encodeErr := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
		if encodeErr != nil {
			os.Exit(1)
		}
		if writeErr := writer.WriteMessage(messageData); writeErr != nil {
			os.Exit(1)
		}
	}
}

func TestGoRenameRejectsUnsafeEditsBeforeWriting(t *testing.T) {
	root, rootErr := filepath.EvalSymlinks(t.TempDir())
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	source := []byte("package sample\n\nfunc Value() int { return 1 }\n")
	for name, content := range map[string][]byte{"go.mod": []byte("module example.com/renamepolicy\n\ngo 1.24\n"), "value.go": source, "other.go": []byte("package sample\n\nfunc Other() int { return 2 }\n")} {
		if err := atomicfile.WriteAtomic(filepath.Join(root, name), content); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, ".scratch", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(bin, "gopls")
	if err := atomicfile.WriteAtomic(wrapper, []byte("#!/bin/sh\nexec \"$SEMEDIT_GO_RENAME_TEST_BIN\" -test.run=^TestGoRenameProtocolHelper$\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(wrapper, 0o700); err != nil {
		t.Fatal(err)
	}
	executable, executableErr := os.Executable()
	if executableErr != nil {
		t.Fatal(executableErr)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_GO_RENAME_TEST_BIN", executable)
	t.Setenv("SEMEDIT_GO_RENAME_HELPER", "1")
	uri := pathutil.FileURI(filepath.Join(root, "value.go"))
	validText := map[string]any{"range": map[string]any{"start": map[string]any{"line": 2, "character": 5}, "end": map[string]any{"line": 2, "character": 10}}, "newText": "Renamed"}
	validDocument := func(uri string, version any, edits any) map[string]any {
		return map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "edits": edits}
	}
	valid := validDocument(uri, 1, []any{validText})
	cases := map[string]any{
		"stale version":       map[string]any{"documentChanges": []any{validDocument(uri, 2, []any{validText})}},
		"null version":        map[string]any{"documentChanges": []any{validDocument(uri, nil, []any{validText})}},
		"foreign second file": map[string]any{"documentChanges": []any{valid, validDocument(pathutil.FileURI(filepath.Join(t.TempDir(), "foreign.go")), 1, []any{validText})}},
		"duplicate document":  map[string]any{"documentChanges": []any{valid, valid}},
		"overlap":             map[string]any{"documentChanges": []any{validDocument(uri, 1, []any{validText, validText})}},
		"resource operation":  map[string]any{"documentChanges": []any{valid, map[string]any{"kind": "delete", "uri": uri}}},
		"unversioned changes": map[string]any{"changes": map[string]any{uri: []any{validText}}},
		"annotations":         map[string]any{"documentChanges": []any{valid}, "changeAnnotations": map[string]any{"confirmation": map[string]any{"label": "rename"}}},
		"invalid syntax":      map[string]any{"documentChanges": []any{validDocument(uri, 1, []any{map[string]any{"range": validText["range"], "newText": "("}})}},
		"missing text":        map[string]any{"documentChanges": []any{validDocument(uri, 1, []any{map[string]any{"range": validText["range"]}})}},
		"wrong replacement":   map[string]any{"documentChanges": []any{validDocument(uri, 1, []any{map[string]any{"range": validText["range"], "newText": "Other"}})}},
		"empty":               map[string]any{},
		"outside range":       map[string]any{"documentChanges": []any{validDocument(uri, 1, []any{map[string]any{"range": map[string]any{"start": map[string]any{"line": 100, "character": 0}, "end": map[string]any{"line": 100, "character": 1}}, "newText": "Renamed"}})}},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) { assertRenameEditRejected(t, root, source, edit) })
	}

	t.Run("changed lookup input", func(t *testing.T) {
		assertChangedLookupSnapshotRejected(t, root, source)
	})
}

func assertRenameEditRejected(t *testing.T, root string, source []byte, edit any) {
	t.Helper()
	data, err := json.Marshal(edit)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEMEDIT_GO_RENAME_EDIT", string(data))
	file := filepath.Join(root, "value.go")
	if renameErr := Rename(t.Context(), root, file, 3, 6, "Renamed"); !errors.Is(renameErr, ErrRenameFailed) {
		t.Fatalf("Rename error = %v", renameErr)
	}
	actual, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, source) {
		t.Fatalf("rejected edit changed source: %s", actual)
	}
}

func assertChangedLookupSnapshotRejected(t *testing.T, root string, source []byte) {
	t.Helper()
	otherPath := filepath.Join(root, "other.go")
	other, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]byte{filepath.Join(root, "value.go"): source, otherPath: other}
	if writeErr := atomicfile.WriteAtomic(otherPath, []byte("package sample\n\nfunc Other() int {return 3}\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	selected := backend.NamedTarget{Name: "Value", Location: backend.SourceLocation{URI: pathutil.FileURI(filepath.Join(root, "value.go")), Range: backend.Range{Start: backend.PositionFromByteOffset(source, bytes.Index(source, []byte("Value")))}}}
	err = RenameSelected(t.Context(), root, selected, expected, "Renamed")
	if !errors.Is(err, ErrRenameFailed) || !strings.Contains(err.Error(), "changed since symbol lookup") {
		t.Fatalf("RenameSelected error = %v", err)
	}
	actual, err := os.ReadFile(filepath.Join(root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, source) {
		t.Fatalf("stale lookup changed the selected source: %s", actual)
	}
}

func TestBufferedRenameSaveNotifications(t *testing.T) {
	fixture := createBufferedRenameFixture(t)
	assertBufferedRenameUnpublished(t, fixture)
	if err := fixture.buffer.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := godiagnostics.SynchronizeWithContext(fixture.diskCtx); err != nil {
		t.Fatal(err)
	}
	fixture.barrier(t)
	assertBufferedRenamePublished(t, fixture)
}

type bufferedRenameFixture struct {
	root, target, log, wrapper string
	source                     []byte
	ctx, diskCtx               context.Context
	buffer                     *sourcebuf.Buffer
	environment                []string
}

func createBufferedRenameFixture(t *testing.T) bufferedRenameFixture {
	t.Helper()
	root, rootErr := filepath.EvalSymlinks(t.TempDir())
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	source := []byte("package sample\n\nfunc Value() int { return 1 }\n")
	target := filepath.Join(root, "value.go")
	for path, data := range map[string][]byte{target: source, filepath.Join(root, "go.mod"): []byte("module example.com/renamebuffer\n\ngo 1.24\n")} {
		if err := atomicfile.WriteAtomic(path, data); err != nil {
			t.Fatal(err)
		}
	}
	binary, executableErr := os.Executable()
	if executableErr != nil {
		t.Fatal(executableErr)
	}
	wrapper := filepath.Join(root, "gopls")
	if err := atomicfile.WriteAtomic(wrapper, []byte("#!/bin/sh\nexec \"$SEMEDIT_GO_RENAME_TEST_BIN\" -test.run=^TestGoRenameProtocolHelper$\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(wrapper, 0o700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(root, ".scratch", "notifications.json")
	if err := os.MkdirAll(filepath.Dir(log), 0o700); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"SEMEDIT_GO_RENAME_HELPER": "1", "SEMEDIT_GO_RENAME_TEST_BIN": binary, "SEMEDIT_GO_RENAME_LOG": log} {
		t.Setenv(key, value)
	}
	t.Setenv("SEMEDIT_GO_RENAME_EDIT", `{"documentChanges":[{"textDocument":{"uri":"`+pathutil.FileURI(target)+`","version":1},"edits":[{"range":{"start":{"line":2,"character":5},"end":{"line":2,"character":10}},"newText":"Renamed"}]}]}`)
	cacheCtx := gocache.WithBaseDir(t.Context(), filepath.Join(root, ".scratch", "go"))
	environment, environmentErr := gocache.Environment(cacheCtx, root)
	if environmentErr != nil {
		t.Fatal(environmentErr)
	}
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	diskCtx := godiagnostics.WithSessionManager(cacheCtx, manager)
	buffer, bufferErr := sourcebuf.New(diskCtx, root)
	if bufferErr != nil {
		t.Fatal(bufferErr)
	}
	ctx := sourcebuf.WithBuffer(diskCtx, buffer)
	if err := godiagnostics.UseWithContext(ctx, root, wrapper, environment, func(ctx context.Context, session *godiagnostics.Session) error {
		return renameInSession(ctx, session, target, 3, 6, "Renamed")
	}); err != nil {
		t.Fatal(err)
	}
	fixture := bufferedRenameFixture{root: root, target: target, log: log, wrapper: wrapper, source: source, ctx: ctx, diskCtx: diskCtx, buffer: buffer, environment: environment}
	fixture.barrier(t)
	return fixture
}

func (fixture bufferedRenameFixture) barrier(t *testing.T) {
	t.Helper()
	if err := godiagnostics.UseWithContext(fixture.ctx, fixture.root, fixture.wrapper, fixture.environment, func(ctx context.Context, session *godiagnostics.Session) error {
		_, err := session.Request(ctx, "barrier", map[string]any{})
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func (fixture bufferedRenameFixture) methods(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(fixture.log)
	if err != nil {
		t.Fatal(err)
	}
	var methods []string
	if err := json.Unmarshal(data, &methods); err != nil {
		t.Fatal(err)
	}
	return methods
}

func assertBufferedRenameUnpublished(t *testing.T, fixture bufferedRenameFixture) {
	t.Helper()
	unsaved := fixture.methods(t)
	if !slices.Contains(unsaved, "textDocument/didChange") || slices.Contains(unsaved, "textDocument/didSave") || slices.Contains(unsaved, "workspace/didChangeWatchedFiles") {
		t.Fatalf("unsaved notifications=%v", unsaved)
	}
	disk, readErr := os.ReadFile(fixture.target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(disk, fixture.source) {
		t.Fatalf("rename published early: %s", disk)
	}
}

func assertBufferedRenamePublished(t *testing.T, fixture bufferedRenameFixture) {
	t.Helper()
	saved := fixture.methods(t)
	if !slices.Contains(saved, "textDocument/didSave") || !slices.Contains(saved, "workspace/didChangeWatchedFiles") {
		t.Fatalf("published notifications=%v", saved)
	}
	disk, readErr := os.ReadFile(fixture.target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(disk), "func Renamed()") {
		t.Fatalf("published source=%s", disk)
	}
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		candidate, value, found := strings.Cut(entry, "=")
		if found && candidate == key {
			return value
		}
	}
	return ""
}
