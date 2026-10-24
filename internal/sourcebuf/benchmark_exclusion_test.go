// Buffered source observations must isolate benchmark artifacts and retain package-tree publication dependencies.
package sourcebuf_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestObservedTreeIgnoresBenchmarkData(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Sample.java")
	archive := filepath.Join(root, "data", "benchmarks", "archive", "Sample.java")
	if err := os.MkdirAll(filepath.Dir(archive), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, []string{target, archive}, []byte("class Sample {}\n"))
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := sourcebuf.ObserveTree(ctx, root, ".java"); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, []string{archive}, []byte("unrelated archived result changed\n"))
	if err := sourcebuf.WriteFile(ctx, target, []byte("class Sample { int value; }\n")); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatalf("archive mutation must not invalidate the observed source tree: %v", err)
	}
	contents, err := os.ReadFile(archive)
	if err != nil || string(contents) != "unrelated archived result changed\n" {
		t.Fatalf("archive changed during source publication: %q, %v", contents, err)
	}
}

func TestObservedSourceTreeAllowsPublishingNewOutputDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "lib.rs")
	output := filepath.Join(root, "generated", "Other.rs")
	writeFiles(t, []string{source}, []byte("pub fn source() {}\n"))
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := buffer.AuthorizeWrites(ctx, root); err != nil {
		t.Fatal(err)
	}
	if err := sourcebuf.ObserveSourceTree(ctx, root, ".rs"); err != nil {
		t.Fatal(err)
	}
	want := []byte("pub fn generated() {}\n")
	if err := sourcebuf.WriteFile(ctx, output, want); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatalf("Flush() with a buffered new directory output = %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != string(want) {
		t.Fatalf("published output = %q, %v; want %q", got, err, want)
	}
}

func TestObservedSourceTreeAllowsPublishingUnderExcludedDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "lib.rs")
	output := filepath.Join(root, ".scratch", "generated", "Other.rs")
	writeFiles(t, []string{source}, []byte("pub fn source() {}\n"))
	buffer, createErr := sourcebuf.New(t.Context(), root)
	if createErr != nil {
		t.Fatal(createErr)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if err := buffer.AuthorizeWrites(ctx, root); err != nil {
		t.Fatal(err)
	}
	if err := sourcebuf.ObserveSourceTree(ctx, root, ".rs"); err != nil {
		t.Fatal(err)
	}
	want := []byte("pub fn generated() {}\n")
	if err := sourcebuf.WriteFile(ctx, output, want); err != nil {
		t.Fatal(err)
	}
	if _, err := buffer.PublishRetained(ctx); err != nil {
		t.Fatalf("PublishRetained() with output under an excluded directory = %v", err)
	}
	if err := buffer.AssertDisk(); err != nil {
		t.Fatalf("AssertDisk() after excluded-directory publication = %v", err)
	}
	if _, err := buffer.PublishRetained(ctx); err != nil {
		t.Fatalf("second PublishRetained() after excluded-directory output = %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != string(want) {
		t.Fatalf("published output = %q, %v; want %q", got, err, want)
	}
}
