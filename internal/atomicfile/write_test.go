// Package atomicfile_test verifies failed prepublication checks preserve concurrent disk changes and clean staged files.
package atomicfile_test

import (
	"errors"
	"github.com/spockz/semantic-editor/internal/atomicfile"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicCheckedRejectsStagedConflict(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "source.go")
	if err := atomicfile.WriteAtomic(target, []byte("original")); err != nil {
		t.Fatal(err)
	}
	conflict := errors.New("source changed")
	called := false
	err := atomicfile.WriteAtomicChecked(target, []byte("rename result"), func() error {
		called = true
		entries, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		if len(entries) != 2 {
			t.Fatalf("check must run after staging: %v", entries)
		}
		if writeErr := atomicfile.WriteAtomic(target, []byte("concurrent change")); writeErr != nil {
			return writeErr
		}
		current, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if string(current) != "original" {
			return conflict
		}
		return nil
	})
	if !called || !errors.Is(err, conflict) || !errors.Is(err, atomicfile.ErrAtomicWrite) {
		t.Fatalf("WriteAtomicChecked error = %v, check called = %v", err, called)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "concurrent change" {
		t.Fatalf("concurrent update overwritten: %s", content)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "source.go" {
		t.Fatalf("staging files leaked: %v", entries)
	}
}
