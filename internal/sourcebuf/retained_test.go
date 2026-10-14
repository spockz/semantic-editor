// This file verifies retained buffers isolate candidate revisions without weakening disk evidence.
package sourcebuf_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestRetainedForkKeepsParentRevision(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	candidate := buffer.Fork()
	ctx := sourcebuf.WithBuffer(context.Background(), candidate)
	before, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourcebuf.WriteFile(ctx, path, append(before[:len(before)-2], []byte("2 }\n")...)); err != nil {
		t.Fatal(err)
	}
	if buffer.DirtySourceBytes() != 0 {
		t.Fatalf("parent dirty bytes = %d", buffer.DirtySourceBytes())
	}
	if candidate.DirtySourceBytes() == 0 {
		t.Fatal("candidate did not retain dirty bytes")
	}
}

func TestPublishRetainedReportsZeroPublicationWithoutClearing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, path, []byte("package sample\n\nfunc Value() {}\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if writeErr := os.WriteFile(path, []byte("package changed\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	published, err := buffer.PublishRetained(context.Background())
	if err == nil || published != 0 {
		t.Fatalf("PublishRetained = (%d, %v), want zero conflict", published, err)
	}
	if buffer.DirtySourceBytes() == 0 {
		t.Fatal("zero-publication failure cleared retained bytes")
	}
}

func TestFlushPublishesNewGoFileOnlyAtPublication(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/buffer\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "generated", "model.go")
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, path, []byte("package generated\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if _, statErr := os.Stat(filepath.Dir(path)); !os.IsNotExist(statErr) {
		t.Fatalf("buffered output directory exists before Flush: %v", statErr)
	}
	if got, readErr := sourcebuf.ReadFile(ctx, path); readErr != nil || string(got) != "package generated\n" {
		t.Fatalf("ReadFile buffered new file = %q, %v", got, readErr)
	}
	if _, ok := sourcebuf.Overlay(ctx)[path]; !ok {
		t.Fatal("overlay omitted buffered new Go file")
	}
	if flushErr := buffer.Flush(ctx); flushErr != nil {
		t.Fatal(flushErr)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "package generated\n" {
		t.Fatalf("published new file = %q, %v", got, err)
	}
}

func TestWriteFileRejectsNewFileBelowSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if symlinkErr := os.Symlink(outside, filepath.Join(root, "outside")); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	path := filepath.Join(root, "outside", "model.go")
	if err := sourcebuf.WriteFile(ctx, path, []byte("package outside\n")); err == nil {
		t.Fatal("buffered write through symlinked parent succeeded")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "model.go")); !os.IsNotExist(statErr) {
		t.Fatalf("write through symlink created outside file: %v", statErr)
	}
}

func TestPublishRetainedRejectsNewParentReplacedBySymlink(t *testing.T) {
	root := t.TempDir()
	sibling := filepath.Join(root, "sibling")
	if err := os.Mkdir(sibling, 0o700); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	path := filepath.Join(root, "generated", "model.go")
	if writeErr := sourcebuf.WriteFile(ctx, path, []byte("package generated\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if symlinkErr := os.Symlink(sibling, filepath.Dir(path)); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	published, err := buffer.PublishRetained(context.Background())
	if err == nil || published != 0 {
		t.Fatalf("PublishRetained = (%d, %v), want zero rejected publication", published, err)
	}
	if _, statErr := os.Stat(filepath.Join(sibling, "model.go")); !os.IsNotExist(statErr) {
		t.Fatalf("aliased publication created sibling source: %v", statErr)
	}
}
