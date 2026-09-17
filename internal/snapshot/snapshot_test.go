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
	if string(blobBytes) != string(content1) {
		t.Errorf("blob content mismatch: expected %q, got %q", string(content1), string(blobBytes))
	}
}

func TestUndo_CleanRestore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file1 := filepath.Join(dir, "calc.go")
	originalContent := []byte("package main\n\nfunc Calculate() int { return 1 }\n")
	if err := os.WriteFile(file1, originalContent, 0o600); err != nil {
		t.Fatalf("write original: %v", err)
	}

	ctx := context.Background()
	res, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{
		Label: "before-edit",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
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
	if string(restoredBytes) != string(originalContent) {
		t.Errorf("expected %q, got %q", string(originalContent), string(restoredBytes))
	}
}

func TestUndo_ConflictDetection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file1 := filepath.Join(dir, "calc.go")
	preimage := []byte("package main\n\nfunc V() int { return 10 }\n")
	if err := os.WriteFile(file1, preimage, 0o600); err != nil {
		t.Fatalf("write preimage: %v", err)
	}

	ctx := context.Background()
	res, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{
		Label: "pre-refactor",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
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
	if string(currentBytes) != string(conflictingEdit) {
		t.Errorf("file was mutated during conflicted undo! expected %q, got %q", string(conflictingEdit), string(currentBytes))
	}
}

func TestUndo_CorruptBlob(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file1 := filepath.Join(dir, "calc.go")
	preimage := []byte("package main\n\nfunc Original() {}\n")
	if err := os.WriteFile(file1, preimage, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx := context.Background()
	res, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{Label: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Mutate on-disk file
	if err := os.WriteFile(file1, []byte("package main\n\nfunc Edited() {}\n"), 0o600); err != nil {
		t.Fatalf("edit: %v", err)
	}

	manifest, err := snapshot.GetManifest(dir, res.SnapshotID)
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	blobPath := filepath.Join(dir, ".scratch", "snapshots", res.SnapshotID, "blobs", manifest.Files[0].PreimageSHA256)

	// Tamper with blob content
	if err := os.WriteFile(blobPath, []byte("tampered blob bytes"), 0o600); err != nil {
		t.Fatalf("tamper blob: %v", err)
	}

	_, err = snapshot.Undo(ctx, dir, res.SnapshotID)
	if err == nil {
		t.Fatalf("expected ErrCorruptBlob, got nil")
	}
	if !errors.Is(err, snapshot.ErrCorruptBlob) {
		t.Errorf("expected ErrCorruptBlob, got: %v", err)
	}
}

func TestUndo_LatestResolution(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file1 := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(file1, []byte("v1"), 0o600); err != nil {
		t.Fatalf("v1: %v", err)
	}

	ctx := context.Background()
	_, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{Label: "snap1"})
	if err != nil {
		t.Fatalf("snap1: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(file1, []byte("v2"), 0o600); err != nil {
		t.Fatalf("v2: %v", err)
	}
	snap2, err := snapshot.Create(ctx, dir, snapshot.CreateOptions{Label: "snap2"})
	if err != nil {
		t.Fatalf("snap2: %v", err)
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
