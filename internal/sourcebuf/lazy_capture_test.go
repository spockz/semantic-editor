// Package sourcebuf_test proves source capture starts at explicit reads and observations.

package sourcebuf_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestLazyCaptureIgnoresUntouchedMixedTreeAndSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "main.rs")
	writeFiles(t, []string{target}, []byte("fn main() {}\n"))
	if mkdirErr := os.MkdirAll(filepath.Join(root, "nested"), 0o700); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	for _, name := range []string{
		"a.go", "A.java", "lib.rs", "A.kt", "script.sh", "Makefile",
		"nested/other.go", "nested/Other.java", "nested/tool.rs", "nested/build.sh",
	} {
		writeFiles(t, []string{filepath.Join(root, name)}, []byte("unrelated\n"))
	}
	if err := os.Symlink(filepath.Join(root, "missing-target"), filepath.Join(root, "broken.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing-directory"), filepath.Join(root, "broken-dir")); err != nil {
		t.Fatal(err)
	}

	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatalf("create buffer with unrelated broken symlinks: %v", createErr)
	}
	if authorizeErr := buffer.AuthorizeWrites(t.Context(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	latestBeforeRead := []byte("fn main() { println!(\"changed before capture\"); }\n")
	writeFiles(t, []string{target}, latestBeforeRead)
	got, firstReadErr := sourcebuf.ReadFile(ctx, target)
	if firstReadErr != nil || !bytes.Equal(got, latestBeforeRead) {
		t.Fatalf("first read=%q err=%v; want latest disk bytes %q", got, firstReadErr, latestBeforeRead)
	}

	writeFiles(t, []string{filepath.Join(root, "a.go")}, []byte("changed after buffer creation\n"))
	writeFiles(t, []string{filepath.Join(root, "new.java")}, []byte("class New {}\n"))
	updated := []byte("fn main() { println!(\"published\"); }\n")
	if targetWriteErr := sourcebuf.WriteFile(ctx, target, updated); targetWriteErr != nil {
		t.Fatal(targetWriteErr)
	}
	scriptPath := filepath.Join(root, "script.sh")
	script := []byte("external edit published through buffer\n")
	if scriptWriteErr := sourcebuf.WriteFile(ctx, scriptPath, script); scriptWriteErr != nil {
		t.Fatal(scriptWriteErr)
	}
	makefilePath := filepath.Join(root, "Makefile")
	makefile := []byte("all:\n\t@echo published\n")
	if makefileWriteErr := sourcebuf.WriteFile(ctx, makefilePath, makefile); makefileWriteErr != nil {
		t.Fatal(makefileWriteErr)
	}
	if _, publishErr := buffer.PublishRetained(ctx); publishErr != nil {
		t.Fatalf("publish buffered Rust, shell, and Makefile edits after unrelated tree changes: %v", publishErr)
	}
	disk, targetReadErr := os.ReadFile(target)
	if targetReadErr != nil || !bytes.Equal(disk, updated) {
		t.Fatalf("published target=%q err=%v; want %q", disk, targetReadErr, updated)
	}
	for path, want := range map[string][]byte{scriptPath: script, makefilePath: makefile} {
		got, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(got, want) {
			t.Errorf("published %s=%q err=%v; want %q", filepath.Base(path), got, readErr, want)
		}
	}
	for path, want := range map[string][]byte{
		filepath.Join(root, "a.go"):     []byte("changed after buffer creation\n"),
		filepath.Join(root, "new.java"): []byte("class New {}\n"),
	} {
		got, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(got, want) {
			t.Errorf("unrelated file %s=%q err=%v; want external revision %q", filepath.Base(path), got, readErr, want)
		}
	}
}

func TestLazyCaptureDetectsTargetChangesAfterFirstRead(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "main.rs")
	original := []byte("fn main() {}\n")
	writeFiles(t, []string{target}, original)
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if got, firstReadErr := sourcebuf.ReadFile(ctx, target); firstReadErr != nil || !bytes.Equal(got, original) {
		t.Fatalf("initial read=%q err=%v", got, firstReadErr)
	}
	if authorizeErr := buffer.AuthorizeWrites(t.Context(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	if writeErr := sourcebuf.WriteFile(ctx, target, []byte("fn main() { println!(\"buffered\"); }\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	external := []byte("fn main() { println!(\"external\"); }\n")
	writeFiles(t, []string{target}, external)
	if _, publishErr := buffer.PublishRetained(ctx); !errors.Is(publishErr, sourcebuf.ErrConflict) {
		t.Fatalf("publication after captured target changed=%v; want ErrConflict", publishErr)
	}
	disk, finalReadErr := os.ReadFile(target)
	if finalReadErr != nil || !bytes.Equal(disk, external) {
		t.Fatalf("conflicting disk target=%q err=%v; want external revision %q", disk, finalReadErr, external)
	}
}
