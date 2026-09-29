// Package astedit tests move publication preconditions that are difficult to trigger through the CLI.
package astedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyMoveFileUnchangedRejectsConcurrentEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.go")
	expected := []byte("package api\n")
	if err := os.WriteFile(path, expected, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package api\n// concurrent edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := verifyMoveFileUnchanged(path, expected)
	if err == nil || !strings.Contains(err.Error(), "changed while move was being prepared") {
		t.Fatalf("verifyMoveFileUnchanged error = %v, want concurrent-edit refusal", err)
	}
}
