// Package astedit tests move publication preconditions that are difficult to trigger through the CLI.
package astedit

import (
	"context"
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
	err := verifyMoveFileUnchanged(context.Background(), path, expected)
	if err == nil || !strings.Contains(err.Error(), "changed while move was being prepared") {
		t.Fatalf("verifyMoveFileUnchanged error = %v, want concurrent-edit refusal", err)
	}
}

func TestMoveConstructRejectsTargetWithoutRelativePlacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.go")
	before := "package api\n\nfunc Keep() {}\n\nfunc MoveMe() {}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, moveErr := MoveConstruct(context.Background(), path, "MoveMe", MoveConstructOptions{TargetSymbol: "Keep"}); moveErr == nil || !strings.Contains(moveErr.Error(), "target_symbol requires") {
		t.Fatalf("MoveConstruct() error = %v, want target placement refusal", moveErr)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("source changed after rejected move: %s", after)
	}
	updated, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.ModTime().Equal(info.ModTime()) {
		t.Fatalf("rejected move changed mtime: before=%s after=%s", info.ModTime(), updated.ModTime())
	}
}
