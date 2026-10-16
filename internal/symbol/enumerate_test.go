// These tests cover the native enumeration contract used to seed bounded
// suggestions, including declaration kinds and the active source revision.
package symbol

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestEnumerateIncludesDeclarationsAndMembers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	source := []byte(`package sample
type Store struct { Value string }
type Reader interface { Read() error }
func Open() {}
func (s *Store) Save() {}
const Version = 1
var DefaultStore Store
`)
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Enumerate(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"Store": "type", "Store.Value": "field", "Reader": "type",
		"Reader.Read": "method", "Open": "function", "Store.Save": "method",
		"Version": "constant", "DefaultStore": "variable",
	}
	for _, symbol := range got {
		key := symbol.BuildQualifiedName()
		kind, ok := want[key]
		if !ok || kind != symbol.Kind {
			t.Errorf("unexpected symbol %#v", symbol)
			continue
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Errorf("missing symbols: %#v", want)
	}
}

func TestEnumerateReadsCurrentBufferedRevision(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(canonicalRoot, "sample.go")
	if err = os.WriteFile(path, []byte("package sample\nfunc Old() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, _, _, err := sourcebuf.Prepare(context.Background(), canonicalRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err = sourcebuf.WriteFile(ctx, path, []byte("package sample\nfunc New() {}\n")); err != nil {
		t.Fatal(err)
	}
	got, err := Enumerate(ctx, root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "New" {
		t.Fatalf("Enumerate() = %#v, want buffered New declaration", got)
	}
}

func TestEnumerateSkipsOversizedSource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.go")
	if err := os.WriteFile(path, make([]byte, maxEnumeratedSourceBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Enumerate(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("Enumerate() returned %d symbols from oversized source", len(got))
	}
}
