// Package astedit verifies safe deletion rechecks captured inputs without rescanning unrelated source additions.
package astedit

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeDeleteAllowsUnobservedSourceAddition(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	selected := filepath.Join(root, "selected.go")
	selectedSource := []byte("package sample\nfunc unused() {}\n")
	if err := os.WriteFile(selected, selectedSource, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := &safeDeleteSnapshot{
		files: map[string][32]byte{
			selected: sha256.Sum256(selectedSource),
		},
	}
	newCaller := filepath.Join(root, "caller.go")
	if err := os.WriteFile(newCaller, []byte("package sample\nfunc call() { unused() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateSafeDeleteSnapshot(t.Context(), snapshot); err != nil {
		t.Fatalf("validateSafeDeleteSnapshot() after unrelated caller addition = %v", err)
	}
}
