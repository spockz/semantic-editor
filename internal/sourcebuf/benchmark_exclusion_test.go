// Buffered source observations must not read or conflict with unrelated benchmark artifacts.
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
