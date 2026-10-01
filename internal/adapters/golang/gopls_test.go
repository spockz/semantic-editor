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

	"semedit/internal/atomicfile"
	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/gocache"
	"semedit/internal/godiagnostics"
	"semedit/internal/lsp"
	"semedit/internal/sourcebuf"
	"slices"
)

func TestCommandEnvUsesWorkspaceLocalGoState(t *testing.T) {
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
	env, err := gocache.Environment(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"GOENV":      filepath.Join(root, ".scratch", "go", "env"),
		"GOCACHE":    filepath.Join(root, ".scratch", "go", "build"),
		"GOMODCACHE": filepath.Join(root, ".scratch", "go", "mod"),
		"GOTMPDIR":   filepath.Join(root, ".scratch", "go", "tmp"),
		"GOBIN":      filepath.Join(root, ".scratch", "go", "bin"),
		"GOPATH":     filepath.Join(root, ".scratch", "go"),
		"GOFLAGS":    "",
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

func TestCommandEnvRequiresWorkspace(t *testing.T) {
	if _, err := gocache.Environment(t.Context(), ""); err == nil {
		t.Fatal("Environment(\"\") succeeded, want error")
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
		data, err := reader.ReadMessage()
		if err != nil {
			os.Exit(1)
		}
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.Unmarshal(data, &request); err != nil {
			os.Exit(1)
		}
		if len(request.ID) == 0 {
			methods = append(methods, request.Method)
			if path := os.Getenv("SEMEDIT_GO_RENAME_LOG"); path != "" {
				data, err := json.Marshal(methods)
				if err != nil {
					os.Exit(1)
				}
				if err := atomicfile.WriteAtomic(path, data); err != nil {
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
		data, err = json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
		if err != nil {
			os.Exit(1)
		}
		if err := writer.WriteMessage(data); err != nil {
			os.Exit(1)
		}
	}
}

func TestGoRenameRejectsUnsafeEditsBeforeWriting(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
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
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
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
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(edit)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("SEMEDIT_GO_RENAME_EDIT", string(data))
			if err := Rename(t.Context(), root, filepath.Join(root, "value.go"), 3, 6, "Renamed"); !errors.Is(err, ErrRenameFailed) {
				t.Fatalf("Rename error = %v", err)
			}
			actual, err := os.ReadFile(filepath.Join(root, "value.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, source) {
				t.Fatalf("rejected edit changed source: %s", actual)
			}
		})
	}

	t.Run("changed lookup input", func(t *testing.T) {
		otherPath := filepath.Join(root, "other.go")
		other, err := os.ReadFile(otherPath)
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string][]byte{filepath.Join(root, "value.go"): source, otherPath: other}
		if err := atomicfile.WriteAtomic(otherPath, []byte("package sample\n\nfunc Other() int {return 3}\n")); err != nil {
			t.Fatal(err)
		}
		selected := backend.SymbolCandidate{Name: "Value", File: "value.go", Offset: bytes.Index(source, []byte("Value"))}
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
	})
}

func TestBufferedRenameSaveNotifications(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := []byte("package sample\n\nfunc Value() int { return 1 }\n")
	target := filepath.Join(root, "value.go")
	for path, data := range map[string][]byte{target: source, filepath.Join(root, "go.mod"): []byte("module example.com/renamebuffer\n\ngo 1.24\n")} {
		if err := atomicfile.WriteAtomic(path, data); err != nil {
			t.Fatal(err)
		}
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
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
	environment, err := gocache.Environment(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	manager := godiagnostics.NewSessionManager()
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	diskCtx := godiagnostics.WithSessionManager(t.Context(), manager)
	ctx := sourcebuf.WithBuffer(diskCtx, buffer)
	if err := godiagnostics.UseWithContext(ctx, root, wrapper, environment, func(ctx context.Context, session *godiagnostics.Session) error {
		return renameInSession(ctx, session, target, 3, 6, "Renamed")
	}); err != nil {
		t.Fatal(err)
	}
	barrier := func() {
		t.Helper()
		if err := godiagnostics.UseWithContext(ctx, root, wrapper, environment, func(ctx context.Context, session *godiagnostics.Session) error {
			_, err := session.Request(ctx, "barrier", map[string]any{})
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	barrier()
	methods := func() []string {
		t.Helper()
		data, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		var methods []string
		if err := json.Unmarshal(data, &methods); err != nil {
			t.Fatal(err)
		}
		return methods
	}
	unsaved := methods()
	if !slices.Contains(unsaved, "textDocument/didChange") || slices.Contains(unsaved, "textDocument/didSave") || slices.Contains(unsaved, "workspace/didChangeWatchedFiles") {
		t.Fatalf("unsaved notifications=%v", unsaved)
	}
	disk, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, source) {
		t.Fatalf("rename published early: %s", disk)
	}
	if err := buffer.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := godiagnostics.SynchronizeWithContext(diskCtx); err != nil {
		t.Fatal(err)
	}
	barrier()
	saved := methods()
	if !slices.Contains(saved, "textDocument/didSave") || !slices.Contains(saved, "workspace/didChangeWatchedFiles") {
		t.Fatalf("published notifications=%v", saved)
	}
	disk, err = os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(disk), "func Renamed()") {
		t.Fatalf("published source=%s", disk)
	}
}

func envValue(env []string, key string) string {
	for _, value := range env {
		candidate, value, found := strings.Cut(value, "=")
		if found && candidate == key {
			return value
		}
	}
	return ""
}
