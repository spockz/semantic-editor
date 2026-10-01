// Package sourcebuf_test proves buffered publication rejects stale semantic inputs without exposing intermediate revisions.
package sourcebuf_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"semedit/internal/atomicfile"
	"semedit/internal/sourcebuf"
	"strings"
	"testing"
)

func TestBufferedEditsPublishFinalRevision(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a.go")
	second := filepath.Join(root, "b.go")
	original := []byte("package sample\nfunc Value()int{return 1}\n")
	for _, path := range []string{first, second} {
		if err := atomicfile.WriteAtomic(path, original); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	for _, path := range []string{first, second} {
		if err := sourcebuf.WriteFile(ctx, path, []byte("package sample\nfunc Value()int{return 2}\n")); err != nil {
			t.Fatal(err)
		}
	}
	final := []byte("package sample\nfunc Renamed()int{return 3}\n")
	if err := sourcebuf.WriteFile(ctx, first, final); err != nil {
		t.Fatal(err)
	}
	visible, err := sourcebuf.ReadFile(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(visible, final) {
		t.Fatalf("buffer=%q", visible)
	}
	snapshot := sourcebuf.Overlay(ctx)
	snapshot[first][0] = '!'
	visible, err = sourcebuf.ReadFile(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(visible, final) {
		t.Fatal("caller mutated retained bytes")
	}
	disk, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, original) {
		t.Fatalf("intermediate edit reached disk: %q", disk)
	}
	during, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, during) {
		t.Fatal("buffered edit replaced source inode")
	}
	if err := buffer.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	disk, err = os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, final) {
		t.Fatalf("published=%q", disk)
	}
	after, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != 0o640 || !after.ModTime().After(before.ModTime()) {
		t.Fatalf("publication metadata=%v", after)
	}
	matches, err := filepath.Glob(filepath.Join(root, ".*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("staged files remain: %v", matches)
	}
}

func TestBufferedPublicationRejectsConflicts(t *testing.T) {
	for _, scenario := range []string{"changed_sibling", "added_source", "replaced_identity", "changed_mode", "missing_input_appeared", "symlink_target"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "a.go")
			sibling := filepath.Join(root, "b.go")
			original := []byte("package sample\nfunc Value()int{return 1}\n")
			for _, path := range []string{target, sibling} {
				if err := atomicfile.WriteAtomic(path, original); err != nil {
					t.Fatal(err)
				}
			}
			buffer, err := sourcebuf.New(root)
			if err != nil {
				t.Fatal(err)
			}
			ctx := sourcebuf.WithBuffer(t.Context(), buffer)
			if _, err := sourcebuf.ReadFile(ctx, sibling); err != nil {
				t.Fatal(err)
			}
			missing := filepath.Join(root, "go.sum")
			if _, err := sourcebuf.ReadFile(ctx, missing); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing=%v", err)
			}
			if err := sourcebuf.WriteFile(ctx, target, []byte("package sample\nfunc Value()int{return 2}\n")); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "changed_sibling":
				err = atomicfile.WriteAtomic(sibling, []byte("package sample\nfunc Other(){}\n"))
			case "added_source":
				err = atomicfile.WriteAtomic(filepath.Join(root, "c.go"), []byte("package sample\n"))
			case "replaced_identity":
				err = atomicfile.WriteAtomic(sibling, original)
			case "changed_mode":
				err = os.Chmod(sibling, 0o640)
			case "missing_input_appeared":
				err = atomicfile.WriteAtomic(missing, []byte("new input\n"))
			case "symlink_target":
				if err = os.Remove(sibling); err == nil {
					err = os.Symlink(target, sibling)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := buffer.Flush(t.Context()); !errors.Is(err, sourcebuf.ErrConflict) {
				t.Fatalf("flush=%v", err)
			}
			disk, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(disk, original) {
				t.Fatalf("conflict published source: %q", disk)
			}
			if len(sourcebuf.Overlay(ctx)) != 0 {
				t.Fatal("failed publication retained unpublished bytes")
			}
		})
	}
}

func TestBufferedStagingFailurePublishesNoFiles(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a.go")
	second := filepath.Join(root, strings.Repeat("z", 250)+".go")
	original := []byte("package sample\n")
	for _, path := range []string{first, second} {
		initial := filepath.Join(root, "initial.go")
		if err := atomicfile.WriteAtomic(initial, original); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(initial, path); err != nil {
			t.Fatal(err)
		}
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	for _, path := range []string{first, second} {
		if err := sourcebuf.WriteFile(ctx, path, []byte("package sample\nfunc Added(){}\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := buffer.Flush(t.Context()); !errors.Is(err, atomicfile.ErrAtomicWrite) {
		t.Fatalf("flush=%v", err)
	}
	for _, path := range []string{first, second} {
		disk, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(disk, original) {
			t.Fatalf("staging failure published %s", path)
		}
	}
	matches, err := filepath.Glob(filepath.Join(root, ".*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("staged files remain: %v", matches)
	}
}

func TestBufferedApplyRejectsEntireOperation(t *testing.T) {
	for _, scenario := range []string{"stale", "duplicate", "outside", "cancelled", "disk_changed"} {
		t.Run(scenario, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			first, second := filepath.Join(root, "a.go"), filepath.Join(root, "b.go")
			original := []byte("package sample\n")
			for _, path := range []string{first, second} {
				if err := atomicfile.WriteAtomic(path, original); err != nil {
					t.Fatal(err)
				}
			}
			buffer, err := sourcebuf.New(root)
			if err != nil {
				t.Fatal(err)
			}
			ctx := sourcebuf.WithBuffer(t.Context(), buffer)
			prior := []byte("package sample\nfunc Prior(){}\n")
			if err := sourcebuf.WriteFile(ctx, first, prior); err != nil {
				t.Fatal(err)
			}
			changes := []sourcebuf.Change{{Path: first, Before: prior, After: []byte("package sample\nfunc Changed(){}\n")}, {Path: second, Before: original, After: []byte("package sample\nfunc Added(){}\n")}}
			switch scenario {
			case "stale":
				changes[1].Before = []byte("stale")
			case "duplicate":
				changes[1] = changes[0]
			case "outside":
				changes[1].Path = filepath.Join(t.TempDir(), "outside.go")
				if err := atomicfile.WriteAtomic(changes[1].Path, original); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "disk_changed":
				if err := atomicfile.WriteAtomic(second, original); err != nil {
					t.Fatal(err)
				}
			}
			if err := buffer.Apply(ctx, changes); err == nil {
				t.Fatal("invalid operation succeeded")
			}
			for path, want := range map[string][]byte{first: prior, second: original} {
				got, err := sourcebuf.ReadFile(ctx, path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("failed operation changed %s: %s", path, got)
				}
			}
		})
	}
}

func TestObservedReplacementRemainsReadOnly(t *testing.T) {
	root := t.TempDir()
	replacement, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(replacement, "external.go")
	original := []byte("package sample\n")
	if err := atomicfile.WriteAtomic(file, original); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := sourcebuf.ObserveTree(ctx, replacement); err != nil {
		t.Fatal(err)
	}
	changes := []sourcebuf.Change{{Path: file, Before: original, After: []byte("package sample\nfunc Changed(){}\n")}}
	if err := buffer.Apply(ctx, changes); err == nil {
		t.Fatal("observed replacement became writable")
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("rejected operation retained dirty bytes")
	}
	if err := buffer.AuthorizeWrites(ctx, replacement); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Apply(ctx, changes); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, changes[0].After) {
		t.Fatalf("authorized output=%s", got)
	}
}
