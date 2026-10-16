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

func TestGenericTreeObservationDoesNotRequireGo(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Sample.java")
	writeFiles(t, []string{target}, []byte("class Sample {}\n"))
	t.Setenv("PATH", t.TempDir())
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := sourcebuf.ObserveTree(ctx, root, ".java"); err != nil {
		t.Fatal(err)
	}
	if err := sourcebuf.WriteFile(ctx, target, []byte("class Sample { int value; }\n")); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestObservedLanguageTreesKeepTheirOwnExclusions(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "local-cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	java := filepath.Join(cache, "Sample.java")
	writeFiles(t, []string{java}, []byte("class Sample {}\n"))
	writeFiles(t, []string{filepath.Join(root, "value.go")}, []byte("package sample\n"))
	goenv := filepath.Join(root, "goenv")
	writeFiles(t, []string{goenv}, []byte("GOCACHE="+cache+"\n"))
	t.Setenv("GOENV", goenv)
	t.Setenv("GOCACHE", "")
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := sourcebuf.ObserveTree(ctx, root, ".java"); err != nil {
		t.Fatal(err)
	}
	if err := sourcebuf.ObserveTree(ctx, root, ".go"); err != nil {
		t.Fatal(err)
	}
	if err := buffer.AssertDisk(); err != nil {
		t.Fatalf("Go filter changed Java observation: %v", err)
	}
	writeFiles(t, []string{java}, []byte("class Changed {}\n"))
	if err := buffer.AssertDisk(); !errors.Is(err, sourcebuf.ErrConflict) {
		t.Fatalf("observed Java mutation lost: %v", err)
	}
}

func TestLazyBufferCapturesOnlyAccessedFiles(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "main.rs")
	sibling := filepath.Join(root, "unrelated.go")
	original := []byte("fn main() {}\n")
	writeFiles(t, []string{target}, original)
	writeFiles(t, []string{sibling}, []byte("package old\n"))
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	if err := buffer.AuthorizeWrites(t.Context(), root); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, []string{sibling}, []byte("package changed\n"))
	writeFiles(t, []string{filepath.Join(root, "new.java")}, []byte("class New {}\n"))
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := sourcebuf.WriteFile(ctx, target, []byte("fn main() { println!(\"changed\"); }\n")); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatalf("unobserved files invalidated a target edit: %v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil || !bytes.Contains(data, []byte("changed")) {
		t.Fatalf("target=%s err=%v", data, readErr)
	}
}
func TestObservedTreeSelectionsRemainIndependent(t *testing.T) {
	for _, extension := range []string{".go", ".java", ".rs", ".kt", ".sh", ""} {
		t.Run(extension, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "target"+extension)
			original := []byte("original\n")
			writeFiles(t, []string{target}, original)
			buffer, err := sourcebuf.New(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			ctx := sourcebuf.WithBuffer(t.Context(), buffer)
			extensions := []string(nil)
			if extension != "" {
				extensions = []string{extension}
			}
			if err := sourcebuf.ObserveTree(ctx, root, extensions...); err != nil {
				t.Fatal(err)
			}
			if err := sourcebuf.WriteFile(ctx, target, []byte("updated\n")); err != nil {
				t.Fatal(err)
			}
			if _, err := buffer.PublishRetained(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

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
	if err := sourcebuf.ObserveTree(sourcebuf.WithBuffer(t.Context(), buffer), root, ".go"); err != nil {
		t.Fatal(err)
	}
	if err := buffer.AssertDisk(); err != nil {
		t.Fatalf("Go state changed source read set: %v", err)
	}
	writeFiles(t, []string{app}, []byte("package sample\nfunc Value() int { return 2 }\n"))
	if err := buffer.AssertDisk(); !errors.Is(err, sourcebuf.ErrConflict) {
		t.Fatalf("application source changed without conflict: %v", err)
	}
}

func TestDirectoryFilterUsesExplicitGoRuntimeRootsWithoutGoEnv(t *testing.T) {
	root := t.TempDir()
	buildCache := filepath.Join(root, "runtime-build")
	moduleCache := filepath.Join(root, "runtime-mod")
	goTemp := filepath.Join(root, "runtime-tmp")
	goPath := filepath.Join(root, "runtime-path")
	fallbackCache := filepath.Join(root, "goenv-build")
	goenv := filepath.Join(root, "goenv")
	if err := atomicfile.WriteAtomic(goenv, []byte("GOCACHE="+fallbackCache+"\n")); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	callLog := filepath.Join(t.TempDir(), "go-env-calls")
	goCommand := filepath.Join(bin, "go")
	script := "#!/bin/sh\nprintf 'called\\n' >> \"$GO_ENV_CALLS\"\nexit 23\n"
	if err := os.WriteFile(goCommand, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("GO_ENV_CALLS", callLog)
	t.Setenv("GOENV", goenv)
	t.Setenv("GOCACHE", buildCache)
	t.Setenv("GOMODCACHE", moduleCache)
	t.Setenv("GOTMPDIR", goTemp)
	t.Setenv("GOPATH", goPath)
	filter, err := sourcebuf.DirectoryFilterFor(t.Context(), root)
	if err != nil {
		t.Fatalf("DirectoryFilterFor() error = %v", err)
	}
	for _, path := range []string{buildCache, moduleCache, goTemp, filepath.Join(goPath, "pkg", "mod")} {
		if !filter.Skip(root, path) {
			t.Errorf("filter did not skip explicit runtime root %q", path)
		}
	}
	newBuildCache := filepath.Join(root, "runtime-build-updated")
	t.Setenv("GOCACHE", newBuildCache)
	updated, err := sourcebuf.DirectoryFilterFor(t.Context(), root)
	if err != nil {
		t.Fatalf("DirectoryFilterFor() after root change: %v", err)
	}
	if !updated.Skip(root, newBuildCache) || updated.Skip(root, buildCache) || updated.Skip(root, fallbackCache) {
		t.Fatalf("filter did not prefer changed explicit roots over GOENV: roots=%v", updated.GeneratedRoots())
	}
	if _, err := os.Stat(callLog); !os.IsNotExist(err) {
		t.Fatalf("go env was called; marker stat error = %v", err)
	}
}

func TestDirectoryFilterReadsUpdatedGoEnvRoots(t *testing.T) {
	root := t.TempDir()
	goenv := filepath.Join(root, "goenv")
	buildCache := filepath.Join(root, "goenv-build-a")
	updatedBuildCache := filepath.Join(root, "goenv-build-b")
	moduleCache := filepath.Join(root, "goenv-mod")
	goTemp := filepath.Join(root, "goenv-tmp")
	goPath := filepath.Join(root, "goenv-path")
	writeConfig := func(cache string) {
		t.Helper()
		content := fmt.Sprintf("GOCACHE=%s\nGOMODCACHE=%s\nGOPATH=%s\nGOTMPDIR=%s\n", cache, moduleCache, goPath, goTemp)
		if err := atomicfile.WriteAtomic(goenv, []byte(content)); err != nil {
			t.Fatalf("write GOENV configuration: %v", err)
		}
	}
	writeConfig(buildCache)
	t.Setenv("GOENV", goenv)
	t.Setenv("GOWORK", "off")
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR"} {
		t.Setenv(key, "")
	}
	first, err := sourcebuf.DirectoryFilterFor(t.Context(), root)
	if err != nil {
		t.Fatalf("DirectoryFilterFor() with GOENV: %v", err)
	}
	if !first.Skip(root, buildCache) {
		t.Fatalf("GOENV GOCACHE was not excluded: %v", first.GeneratedRoots())
	}
	writeConfig(updatedBuildCache)
	second, err := sourcebuf.DirectoryFilterFor(t.Context(), root)
	if err != nil {
		t.Fatalf("DirectoryFilterFor() after GOENV update: %v", err)
	}
	if !second.Skip(root, updatedBuildCache) || second.Skip(root, buildCache) {
		t.Fatalf("filter did not follow updated GOENV: roots=%v", second.GeneratedRoots())
	}
}

func TestDirectoryFilterFallsBackForRelativeGoRoots(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	callLog := filepath.Join(t.TempDir(), "go-env-calls")
	goCommand := filepath.Join(bin, "go")
	payload := fmt.Sprintf(`{"GOFLAGS":"","GOCACHE":%q,"GOMODCACHE":%q,"GOPATH":%q,"GOTMPDIR":"relative-go-tmp"}`,
		filepath.Join(root, "build-cache"), filepath.Join(root, "module-cache"), filepath.Join(root, "go-path"))
	script := "#!/bin/sh\nprintf 'called\\n' >> \"$GO_ENV_CALLS\"\nprintf '%s\\n' '" + payload + "'\n"
	if err := os.WriteFile(goCommand, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("GO_ENV_CALLS", callLog)
	t.Setenv("GOCACHE", filepath.Join(root, "build-cache"))
	t.Setenv("GOMODCACHE", filepath.Join(root, "module-cache"))
	t.Setenv("GOPATH", filepath.Join(root, "go-path"))
	t.Setenv("GOTMPDIR", "relative-go-tmp")
	filter, err := sourcebuf.DirectoryFilterFor(t.Context(), root)
	if err != nil {
		t.Fatalf("DirectoryFilterFor() with relative GOTMPDIR: %v", err)
	}
	if !filter.Skip(root, filepath.Join(root, "relative-go-tmp")) {
		t.Fatalf("relative GOTMPDIR did not resolve beneath workspace: roots=%v", filter.GeneratedRoots())
	}
	if _, err := os.Stat(callLog); err != nil {
		t.Fatalf("go env fallback was not called: %v", err)
	}
}

func TestDirectoryFilterKeepsSeedWithoutBaseRefusal(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	callLog := filepath.Join(t.TempDir(), "go-env-calls")
	goCommand := filepath.Join(bin, "go")
	script := "#!/bin/sh\nprintf 'called\\n' >> \"$GO_ENV_CALLS\"\nexit 23\n"
	if err := os.WriteFile(goCommand, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("GO_ENV_CALLS", callLog)
	t.Setenv("GOCACHE", filepath.Join(root, "build-cache"))
	t.Setenv("GOMODCACHE", filepath.Join(root, "module-cache"))
	t.Setenv("GOTMPDIR", filepath.Join(root, "go-tmp"))
	t.Setenv("GOPATH", filepath.Join(root, "go-path"))
	ctx := gocache.WithModuleCacheSeed(t.Context(), filepath.Join(root, "seed"))
	_, err := sourcebuf.DirectoryFilterFor(ctx, root)
	if err == nil || !strings.Contains(err.Error(), "explicit Go isolation base directory") {
		t.Fatalf("DirectoryFilterFor() error = %v, want seed-without-base refusal", err)
	}
	if _, err := os.Stat(callLog); !os.IsNotExist(err) {
		t.Fatalf("Go settings subprocess should not run after seed refusal; marker stat error = %v", err)
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
	if err := sourcebuf.ObserveTree(sourcebuf.WithBuffer(ctx, buffer), root, ".go"); err != nil {
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
	if err := sourcebuf.ObserveTree(ctx, root, ".go"); err != nil {
		t.Fatal(err)
	}
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
	if _, err := sourcebuf.ReadFile(ctx, second); err != nil {
		t.Fatal(err)
	}
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

func TestApplyCreateRejectsConflictsWithoutChangingOtherOutputs(t *testing.T) {
	for _, scenario := range []string{"existing-empty", "appeared", "invalid-later", "non-nil-before", "nil-after", "empty-after"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := prepareCreateConflict(t, scenario)
			if applyErr := fixture.buffer.Apply(fixture.ctx, fixture.plan); applyErr == nil {
				t.Fatal("conflicting Create plan succeeded")
			}
			working, readErr := sourcebuf.ReadFile(fixture.ctx, fixture.existing)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(working, fixture.prior) {
				t.Fatalf("failed plan changed prior revision: %q", working)
			}
			if !fixture.diskTarget {
				if _, missingErr := sourcebuf.ReadFile(fixture.ctx, fixture.created); !errors.Is(missingErr, os.ErrNotExist) {
					t.Fatalf("failed plan created working output: %v", missingErr)
				}
			}
		})
	}
}

func TestApplyCreatePublishesOneMixedWorkingRevision(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(root, "existing.go")
	created := filepath.Join(root, "nested", "created.go")
	before := []byte("package sample\n")
	after := []byte("package sample\nconst Value = 1\n")
	writeFiles(t, []string{existing}, before)
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	plan := []sourcebuf.Change{{Path: existing, Before: before, After: after}, {Path: created, Create: true, After: after}}
	if err := buffer.Apply(ctx, plan); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourcebuf.Overlay(ctx)[created], after) {
		t.Fatal("new source absent from working overlay")
	}
	if _, err := os.Stat(created); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new file published early: %v", err)
	}
	if err := buffer.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{existing, created} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, after) {
			t.Fatalf("published %s = %q", path, got)
		}
	}
}

type cancelAfterDiskRevision struct {
	context.Context

	path     string
	revision []byte
}

func (c cancelAfterDiskRevision) Err() error {
	if err := c.Context.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return fmt.Errorf("observe publication: %w", err)
	}
	if bytes.Equal(data, c.revision) {
		return context.Canceled
	}
	return nil
}
func TestFlushReportsActualPartialPublication(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, second := filepath.Join(root, "a.go"), filepath.Join(root, "b.go")
	before := []byte("package sample\n")
	after := []byte("package sample\nconst Value = 1\n")
	writeFiles(t, []string{first, second}, before)
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if applyErr := buffer.Apply(ctx, []sourcebuf.Change{{Path: first, Before: before, After: after}, {Path: second, Before: before, After: after}}); applyErr != nil {
		t.Fatal(applyErr)
	}
	err = buffer.Flush(cancelAfterDiskRevision{Context: t.Context(), path: first, revision: after})
	var publication *sourcebuf.PublicationError
	if !errors.As(err, &publication) || !errors.Is(err, context.Canceled) {
		t.Fatalf("Flush error = %v", err)
	}
	if len(publication.Published) != 1 || publication.Published[0] != first || len(publication.Planned) != 2 {
		t.Fatalf("publication = %#v", publication)
	}
	firstDisk, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondDisk, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstDisk, after) || !bytes.Equal(secondDisk, before) {
		t.Fatalf("partial outputs = %q, %q", firstDisk, secondDisk)
	}
}

type createConflictFixture struct {
	buffer     *sourcebuf.Buffer
	ctx        context.Context
	existing   string
	created    string
	prior      []byte
	plan       []sourcebuf.Change
	diskTarget bool
}

func prepareCreateConflict(t *testing.T, scenario string) createConflictFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing, created := filepath.Join(root, "existing.go"), filepath.Join(root, "created.go")
	original := []byte("package sample\n")
	prior := []byte("package sample\nconst Prior = 1\n")
	writeFiles(t, []string{existing}, original)
	if scenario == "existing-empty" {
		writeFiles(t, []string{created}, []byte{})
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, existing, prior); writeErr != nil {
		t.Fatal(writeErr)
	}
	if scenario == "appeared" {
		if _, bindErr := sourcebuf.ReadFile(ctx, created); !errors.Is(bindErr, os.ErrNotExist) {
			t.Fatalf("bind absence: %v", bindErr)
		}
		writeFiles(t, []string{created}, original)
	}
	creation := sourcebuf.Change{Path: created, Create: true, After: []byte("package sample\nconst Created = 1\n")}
	switch scenario {
	case "non-nil-before":
		creation.Before = []byte{}
	case "nil-after":
		creation.After = nil
	case "empty-after":
		creation.After = []byte{}
	}
	plan := []sourcebuf.Change{{Path: existing, Before: prior, After: []byte("package sample\nconst Changed = 1\n")}, creation}
	if scenario == "invalid-later" {
		plan = append(plan, sourcebuf.Change{Path: existing, Before: prior, After: original})
	}
	return createConflictFixture{buffer: buffer, ctx: ctx, existing: existing, created: created, prior: prior, plan: plan, diskTarget: scenario == "existing-empty" || scenario == "appeared"}
}
