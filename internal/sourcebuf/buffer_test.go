// Package sourcebuf_test proves buffered publication rejects stale semantic inputs without exposing intermediate revisions.
package sourcebuf_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/atomicfile"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBufferedEditsPublishFinalRevision(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a.go")
	second := filepath.Join(root, "b.go")
	original := []byte("package sample\nfunc Value()int{return 1}\n")
	before := createBufferedSources(t, first, second, original)
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	final := stageBufferedFinalRevision(ctx, t, first, second)
	assertBufferedRevision(ctx, t, first, final, original, before)
	if err := buffer.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertPublishedRevision(t, root, first, final, before)
}

func TestRunPublishesOwnedBufferOnlyAfterSuccessfulEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.go")
	original := []byte("package sample\nfunc Value() int { return 1 }\n")
	writeFiles(t, []string{path}, original)
	updated := []byte("package sample\nfunc Value() int { return 2 }\n")
	if err := sourcebuf.Run(t.Context(), root, func(ctx context.Context) error {
		if err := sourcebuf.WriteFile(ctx, path, updated); err != nil {
			return err
		}
		working, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return err
		}
		if !bytes.Equal(working, updated) {
			return fmt.Errorf("working revision=%q", working)
		}
		disk, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(disk, original) {
			return fmt.Errorf("edit reached disk before completion: %q", disk)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, updated) {
		t.Fatalf("published revision=%q", disk)
	}
}

func TestRunRollsBackOnlyFailedSharedEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.go")
	original := []byte("package sample\nfunc Value() int { return 1 }\n")
	prior := []byte("package sample\nfunc Value() int { return 2 }\n")
	writeFiles(t, []string{path}, original)
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, path, prior); writeErr != nil {
		t.Fatal(writeErr)
	}
	wantErr := errors.New("refused after staging")
	err = sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		if writeErr := sourcebuf.WriteFile(editCtx, path, []byte("package sample\nfunc Value() int { return 3 }\n")); writeErr != nil {
			return writeErr
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error=%v", err)
	}
	working, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(working, prior) {
		t.Fatalf("rolled back to %q; want prior batch revision %q", working, prior)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, original) {
		t.Fatalf("shared edit reached disk: %q", disk)
	}
}

func TestChangesSinceTracksChildRevertingPriorBatchEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.go")
	original := []byte("package sample\nfunc Value() int { return 1 }\n")
	prior := []byte("package sample\nfunc Value() int { return 2 }\n")
	writeFiles(t, []string{path}, original)
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, path, prior); writeErr != nil {
		t.Fatal(writeErr)
	}
	checkpoint := buffer.Checkpoint()
	if runErr := sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		return sourcebuf.WriteFile(editCtx, path, original)
	}); runErr != nil {
		t.Fatal(runErr)
	}
	changes := buffer.ChangesSince(checkpoint)
	if len(changes) != 1 || !bytes.Equal(changes[0].Before, prior) || !bytes.Equal(changes[0].After, original) {
		t.Fatalf("child changes=%+v; want prior-to-original revision", changes)
	}
	if got := buffer.ChangedPaths(); len(got) != 0 {
		t.Fatalf("net dirty paths=%v; want none", got)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, original) {
		t.Fatalf("child unexpectedly published disk bytes: %q", disk)
	}
}

func createBufferedSources(t *testing.T, first, second string, original []byte) os.FileInfo {
	t.Helper()
	for _, path := range []string{first, second} {
		if err := atomicfile.WriteAtomic(path, original); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func stageBufferedFinalRevision(ctx context.Context, t *testing.T, first, second string) []byte {
	t.Helper()
	for _, path := range []string{first, second} {
		if err := sourcebuf.WriteFile(ctx, path, []byte("package sample\nfunc Value()int{return 2}\n")); err != nil {
			t.Fatal(err)
		}
	}
	final := []byte("package sample\nfunc Renamed()int{return 3}\n")
	if err := sourcebuf.WriteFile(ctx, first, final); err != nil {
		t.Fatal(err)
	}
	return final
}

func assertBufferedRevision(ctx context.Context, t *testing.T, first string, final, original []byte, before os.FileInfo) {
	t.Helper()
	visible, err := sourcebuf.ReadFile(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(visible, final) {
		t.Fatalf("buffer=%q", visible)
	}
	sourcebuf.Overlay(ctx)[first][0] = '!'
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
}

func assertPublishedRevision(t *testing.T, root, first string, final []byte, before os.FileInfo) {
	t.Helper()
	disk, err := os.ReadFile(first)
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
			testPublicationConflict(t, scenario)
		})
	}
}

func TestBufferedSourceSetIgnoresEffectiveGoState(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app.go")
	writeFiles(t, []string{app}, []byte("package sample\nfunc Value() int { return 1 }\n"))
	cacheRoots := []string{
		filepath.Join(root, "build-cache"),
		filepath.Join(root, "module-cache"),
		filepath.Join(root, "go-path", "pkg", "mod"),
		filepath.Join(root, "go-temp"),
		filepath.Join(root, "gopls-cache"),
		filepath.Join(root, "process-temp"),
	}
	for _, directory := range cacheRoots {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		writeFiles(t, []string{filepath.Join(directory, "existing.go")}, []byte("package generated\n"))
	}
	goenv := filepath.Join(root, "goenv")
	goenvData := fmt.Sprintf("GOCACHE=%s\nGOMODCACHE=%s\nGOPATH=%s\nGOTMPDIR=%s\n", cacheRoots[0], cacheRoots[1], filepath.Join(root, "go-path"), cacheRoots[3])
	if err := atomicfile.WriteAtomic(goenv, []byte(goenvData)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", goenv)
	t.Setenv("GOWORK", "off")
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR"} {
		t.Setenv(key, "")
	}
	t.Setenv("GOPLSCACHE", cacheRoots[4])
	t.Setenv("TMPDIR", cacheRoots[5])
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, directory := range cacheRoots {
		writeFiles(t, []string{filepath.Join(directory, "existing.go"), filepath.Join(directory, "new.go")}, []byte("package generated\nfunc Generated() {}\n"))
	}
	if err := buffer.AssertDisk(); err != nil {
		t.Fatalf("Go state changed source read set: %v", err)
	}
	writeFiles(t, []string{app}, []byte("package sample\nfunc Value() int { return 2 }\n"))
	if err := buffer.AssertDisk(); !errors.Is(err, sourcebuf.ErrConflict) {
		t.Fatalf("application source changed without conflict: %v", err)
	}
}

func TestBufferedSourceSetIgnoresExplicitGoBaseDir(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "selected-go-state")
	ctx := gocache.WithBaseDir(t.Context(), base)
	buffer, err := sourcebuf.New(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"build", "mod", "tmp", "gopls"} {
		path := filepath.Join(base, directory, "generated.go")
		writeFiles(t, []string{path}, []byte("package generated\n"))
	}
	if err := buffer.AssertDisk(); err != nil {
		t.Fatalf("explicit Go state changed source read set: %v", err)
	}
	writeFiles(t, []string{filepath.Join(root, "app.go")}, []byte("package sample\n"))
	if err := buffer.AssertDisk(); !errors.Is(err, sourcebuf.ErrConflict) {
		t.Fatalf("application source appeared without conflict: %v", err)
	}
}

func testPublicationConflict(t *testing.T, scenario string) {
	t.Helper()
	root := t.TempDir()
	target, sibling := filepath.Join(root, "a.go"), filepath.Join(root, "b.go")
	original := []byte("package sample\nfunc Value()int{return 1}\n")
	writeFiles(t, []string{target, sibling}, original)
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
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
	if err := alterPublicationInput(scenario, root, target, sibling, missing, original); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(t.Context()); !errors.Is(err, sourcebuf.ErrConflict) {
		t.Fatalf("flush=%v", err)
	}
	assertUnpublishedConflict(ctx, t, target, original)
}

func writeFiles(t *testing.T, paths []string, data []byte) {
	t.Helper()
	for _, path := range paths {
		if err := atomicfile.WriteAtomic(path, data); err != nil {
			t.Fatal(err)
		}
	}
}

func alterPublicationInput(scenario, root, target, sibling, missing string, original []byte) error {
	switch scenario {
	case "changed_sibling":
		return atomicfile.WriteAtomic(sibling, []byte("package sample\nfunc Other(){}\n"))
	case "added_source":
		return atomicfile.WriteAtomic(filepath.Join(root, "c.go"), []byte("package sample\n"))
	case "replaced_identity":
		return atomicfile.WriteAtomic(sibling, original)
	case "changed_mode":
		return os.Chmod(sibling, 0o640)
	case "missing_input_appeared":
		return atomicfile.WriteAtomic(missing, []byte("new input\n"))
	case "symlink_target":
		if err := os.Remove(sibling); err != nil {
			return err
		}
		return os.Symlink(target, sibling)
	default:
		return fmt.Errorf("unknown conflict scenario %q", scenario)
	}
}

func assertUnpublishedConflict(ctx context.Context, t *testing.T, target string, original []byte) {
	t.Helper()
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
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
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
			testBufferedApplyRejectsScenario(t, scenario)
		})
	}
}

func testBufferedApplyRejectsScenario(t *testing.T, scenario string) {
	t.Helper()
	root, resolveErr := filepath.EvalSymlinks(t.TempDir())
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	first, second := filepath.Join(root, "a.go"), filepath.Join(root, "b.go")
	original := []byte("package sample\n")
	writeFiles(t, []string{first, second}, original)
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	prior := []byte("package sample\nfunc Prior(){}\n")
	if err := sourcebuf.WriteFile(ctx, first, prior); err != nil {
		t.Fatal(err)
	}
	changes := []sourcebuf.Change{{Path: first, Before: prior, After: []byte("package sample\nfunc Changed(){}\n")}, {Path: second, Before: original, After: []byte("package sample\nfunc Added(){}\n")}}
	ctx, err := alterApplyScenario(ctx, t, scenario, changes, second, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := buffer.Apply(ctx, changes); err == nil {
		t.Fatal("invalid operation succeeded")
	}
	assertApplyUnchanged(ctx, t, first, second, prior, original)
}

func alterApplyScenario(ctx context.Context, t *testing.T, scenario string, changes []sourcebuf.Change, second string, original []byte) (context.Context, error) {
	switch scenario {
	case "stale":
		changes[1].Before = []byte("stale")
	case "duplicate":
		changes[1] = changes[0]
	case "outside":
		changes[1].Path = filepath.Join(t.TempDir(), "outside.go")
		return ctx, atomicfile.WriteAtomic(changes[1].Path, original)
	case "cancelled":
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		return cancelled, nil
	case "disk_changed":
		return ctx, atomicfile.WriteAtomic(second, original)
	default:
		return ctx, fmt.Errorf("unknown apply scenario %q", scenario)
	}
	return ctx, nil
}

func assertApplyUnchanged(ctx context.Context, t *testing.T, first, second string, prior, original []byte) {
	t.Helper()
	for path, want := range map[string][]byte{first: prior, second: original} {
		got, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("failed operation changed %s: %s", path, got)
		}
	}
}

func TestObservedReplacementRemainsReadOnly(t *testing.T) {
	root := t.TempDir()
	replacement, resolveErr := filepath.EvalSymlinks(t.TempDir())
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	file := filepath.Join(replacement, "external.go")
	original := []byte("package sample\n")
	if err := atomicfile.WriteAtomic(file, original); err != nil {
		t.Fatal(err)
	}
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
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
