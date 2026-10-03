// Package snapshot_test verifies transactional snapshot persistence, conflict detection, and rollback.
package snapshot_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bytes"
	"semedit/internal/snapshot"
)

func TestCreateSnapshot_SpecificFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file1 := filepath.Join(dir, "pkg", "calc.go")
	if err := os.MkdirAll(filepath.Dir(file1), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content1 := []byte("package pkg\n\nfunc Add(a, b int) int { return a + b }\n")
	if err := os.WriteFile(file1, content1, 0o600); err != nil {
		t.Fatalf("write file1: %v", err)
	}

	ctx := context.Background()
	res, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{
		Label:       "test-label",
		Description: "test-description",
		Paths:       []string{"pkg/calc.go"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !strings.HasPrefix(res.SnapshotID, "snap-") {
		t.Errorf("expected snap- prefix, got %s", res.SnapshotID)
	}
	if len(res.CapturedFiles) != 1 || res.CapturedFiles[0] != "pkg/calc.go" {
		t.Errorf("unexpected captured files: %v", res.CapturedFiles)
	}

	manifest, err := snapshot.GetManifest(dir, res.SnapshotID)
	if err != nil {
		t.Fatalf("GetManifest failed: %v", err)
	}
	if manifest.Label != "test-label" || manifest.Description != "test-description" {
		t.Errorf("manifest metadata mismatch: %+v", manifest)
	}
	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 file in manifest, got %d", len(manifest.Files))
	}
	rec := manifest.Files[0]
	if rec.Path != "pkg/calc.go" {
		t.Errorf("expected path pkg/calc.go, got %s", rec.Path)
	}
	if rec.PreimageSHA256 == "" {
		t.Errorf("expected non-empty preimage sha256")
	}

	// Verify blob exists on disk
	blobPath := filepath.Join(dir, ".scratch", "snapshots", res.SnapshotID, "blobs", rec.PreimageSHA256)
	// #nosec G304 -- test reads created snapshot blob
	blobBytes, err := os.ReadFile(blobPath)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}
	if !bytes.Equal(blobBytes, content1) {
		t.Errorf("blob content mismatch: expected %q, got %q", string(content1), string(blobBytes))
	}
}

func TestUndo_CleanRestore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSnapshotTestModule(t, dir)

	file1 := filepath.Join(dir, "calc.go")
	originalContent := []byte("package main\n\nfunc Calculate() int { return 1 }\n")
	if err := os.WriteFile(file1, originalContent, 0o600); err != nil {
		t.Fatalf("write original: %v", err)
	}

	ctx := context.Background()
	res, createErr := snapshot.Create(ctx, dir, snapshot.CreateOptions{
		Label: "before-edit",
	})
	if createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}

	// Modify file on disk
	modifiedContent := []byte("package main\n\nfunc Calculate() int { return 2 }\n")
	if err := os.WriteFile(file1, modifiedContent, 0o600); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Execute undo
	undoRes, err := snapshot.Undo(ctx, dir, res.SnapshotID)
	if err != nil {
		t.Fatalf("Undo failed: %v", err)
	}

	if len(undoRes.RestoredFiles) != 1 || undoRes.RestoredFiles[0] != "calc.go" {
		t.Errorf("expected calc.go restored, got %v", undoRes.RestoredFiles)
	}
	if !strings.Contains(undoRes.RestoredDiff, "--- a/calc.go") {
		t.Errorf("expected diff to mention a/calc.go, got:\n%s", undoRes.RestoredDiff)
	}

	// #nosec G304 -- test reads temporary file
	restoredBytes, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if !bytes.Equal(restoredBytes, originalContent) {
		t.Errorf("expected %q, got %q", string(originalContent), string(restoredBytes))
	}
}

func TestUndoReportsUnavailableDiagnosticsAfterRestore(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	goplsPath := filepath.Join(binDir, "gopls")
	markerPath := filepath.Join(t.TempDir(), "gopls-invoked")
	if err := os.WriteFile(goplsPath, []byte("#!/bin/sh\nprintf invoked > \"$SEMEDIT_GOPLS_MARKER\"\nexit 23\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_GOPLS_MARKER", markerPath)
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module snapshot.test/fixture\n\ngo 1.27\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	preimage := []byte("package main\n\nfunc Value() int { return 1 }\n")
	if err := os.WriteFile(file, preimage, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshotResult, createErr := snapshot.Create(context.Background(), dir, snapshot.CreateOptions{Label: "before-edit"})
	if createErr != nil {
		t.Fatal(createErr)
	}
	if writeErr := os.WriteFile(file, []byte("package main\n\nfunc Value() int { return 2 }\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	result, undoErr := snapshot.Undo(context.Background(), dir, snapshotResult.SnapshotID)
	if undoErr != nil {
		t.Fatalf("Undo failed: %v", undoErr)
	}
	if result.DiagnosticStatus != "unavailable" || result.DiagnosticError == "" || result.Delta != nil {
		t.Fatalf("diagnostics outcome = status %q, error %q, delta %#v", result.DiagnosticStatus, result.DiagnosticError, result.Delta)
	}
	if !strings.Contains(result.DiagnosticError, "gopls") {
		t.Fatalf("diagnostic error = %q, want gopls failure", result.DiagnosticError)
	}
	marker, markerErr := os.ReadFile(markerPath)
	if markerErr != nil || string(marker) != "invoked" {
		t.Fatalf("gopls failure marker = %q, error = %v", marker, markerErr)
	}
	if len(result.RestoredFiles) != 1 || result.RestoredFiles[0] != "main.go" {
		t.Fatalf("restored files = %v", result.RestoredFiles)
	}
	// #nosec G304 -- test reads temporary workspace file
	current, readErr := os.ReadFile(file)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(current, preimage) {
		t.Fatalf("restored bytes = %q, want %q", current, preimage)
	}
}

func TestUndo_ConflictDetection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSnapshotTestModule(t, dir)

	file1 := filepath.Join(dir, "calc.go")
	preimage := []byte("package main\n\nfunc V() int { return 10 }\n")
	if err := os.WriteFile(file1, preimage, 0o600); err != nil {
		t.Fatalf("write preimage: %v", err)
	}

	ctx := context.Background()
	res, createErr := snapshot.Create(ctx, dir, snapshot.CreateOptions{
		Label: "pre-refactor",
	})
	if createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}

	// Apply intentional staged edit
	postEdit := []byte("package main\n\nfunc V() int { return 20 }\n")
	if err := os.WriteFile(file1, postEdit, 0o600); err != nil {
		t.Fatalf("write postEdit: %v", err)
	}

	// Record post edit
	if _, err := snapshot.RecordPostEdit(ctx, dir, res.SnapshotID); err != nil {
		t.Fatalf("RecordPostEdit failed: %v", err)
	}

	// External modification causing conflict
	conflictingEdit := []byte("package main\n\nfunc V() int { return 999 }\n")
	if err := os.WriteFile(file1, conflictingEdit, 0o600); err != nil {
		t.Fatalf("write conflictingEdit: %v", err)
	}

	// Undo must abort with conflict and leave file untouched
	undoRes, err := snapshot.Undo(ctx, dir, res.SnapshotID)
	if err == nil {
		t.Fatalf("expected conflict error, got success: %+v", undoRes)
	}
	if !errors.Is(err, snapshot.ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}

	var conflictErr *snapshot.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *snapshot.ConflictError, got %T (%v)", err, err)
	}
	if conflictErr.Path != "calc.go" {
		t.Errorf("expected conflict in calc.go, got %s", conflictErr.Path)
	}

	// Assert zero writes occurred
	// #nosec G304 -- test reads temporary file
	currentBytes, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("read after failed undo: %v", err)
	}
	if !bytes.Equal(currentBytes, conflictingEdit) {
		t.Errorf("file was mutated during conflicted undo! expected %q, got %q", string(conflictingEdit), string(currentBytes))
	}
}

func TestUndo_CorruptBlob(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSnapshotTestModule(t, dir)

	file1 := filepath.Join(dir, "calc.go")
	preimage := []byte("package main\n\nfunc Original() {}\n")
	if err := os.WriteFile(file1, preimage, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx := context.Background()
	res, createErr := snapshot.Create(ctx, dir, snapshot.CreateOptions{Label: "test"})
	if createErr != nil {
		t.Fatalf("Create: %v", createErr)
	}

	// Mutate on-disk file
	if err := os.WriteFile(file1, []byte("package main\n\nfunc Edited() {}\n"), 0o600); err != nil {
		t.Fatalf("edit: %v", err)
	}

	manifest, manifestErr := snapshot.GetManifest(dir, res.SnapshotID)
	if manifestErr != nil {
		t.Fatalf("GetManifest: %v", manifestErr)
	}
	blobPath := filepath.Join(dir, ".scratch", "snapshots", res.SnapshotID, "blobs", manifest.Files[0].PreimageSHA256)

	// Tamper with blob content
	if err := os.WriteFile(blobPath, []byte("tampered blob bytes"), 0o600); err != nil {
		t.Fatalf("tamper blob: %v", err)
	}

	_, undoErr := snapshot.Undo(ctx, dir, res.SnapshotID)
	if undoErr == nil {
		t.Fatalf("expected ErrCorruptBlob, got nil")
	}
	if !errors.Is(undoErr, snapshot.ErrCorruptBlob) {
		t.Errorf("expected ErrCorruptBlob, got: %v", undoErr)
	}
}

func TestUndo_LatestResolution(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSnapshotTestModule(t, dir)

	file1 := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(file1, []byte("v1"), 0o600); err != nil {
		t.Fatalf("v1: %v", err)
	}

	ctx := context.Background()
	_, firstCreateErr := snapshot.Create(ctx, dir, snapshot.CreateOptions{Label: "snap1"})
	if firstCreateErr != nil {
		t.Fatalf("snap1: %v", firstCreateErr)
	}

	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(file1, []byte("v2"), 0o600); err != nil {
		t.Fatalf("v2: %v", err)
	}
	snap2, secondCreateErr := snapshot.Create(ctx, dir, snapshot.CreateOptions{Label: "snap2"})
	if secondCreateErr != nil {
		t.Fatalf("snap2: %v", secondCreateErr)
	}

	// Now modify to v3
	if err := os.WriteFile(file1, []byte("v3"), 0o600); err != nil {
		t.Fatalf("v3: %v", err)
	}

	manifests, err := snapshot.ListSnapshots(dir)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(manifests) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(manifests))
	}
	if manifests[0].ID != snap2.SnapshotID {
		t.Errorf("expected latest snapshot %s first, got %s", snap2.SnapshotID, manifests[0].ID)
	}

	// Undo "latest" should restore v2
	undoRes, err := snapshot.Undo(ctx, dir, "latest")
	if err != nil {
		t.Fatalf("Undo latest: %v", err)
	}
	if len(undoRes.RestoredFiles) != 1 {
		t.Fatalf("expected 1 file restored, got %d", len(undoRes.RestoredFiles))
	}

	// #nosec G304 -- test reads temporary file
	data, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "v2" {
		t.Errorf("expected v2, got %q", string(data))
	}
}

func writeSnapshotTestModule(t *testing.T, dir string) {
	t.Helper()
	module := []byte("module snapshot.test/fixture\n\ngo 1.27.1\n")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), module, 0o600); err != nil {
		t.Fatalf("write test go.mod: %v", err)
	}
}

func TestUndo_PathOutsideWorkspace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	ctx := context.Background()
	_, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{
		Paths: []string{"../escape.go"},
	})
	if err == nil {
		t.Fatalf("expected error for path traversal, got nil")
	}
	if !errors.Is(err, snapshot.ErrPathOutsideWorkspace) {
		t.Errorf("expected ErrPathOutsideWorkspace, got: %v", err)
	}
}
