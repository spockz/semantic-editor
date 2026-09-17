// Package snapshot provides transactional pre-edit snapshots and conflict-checked rollback for semantic refactorings.
package snapshot

import (
	"errors"
	"fmt"
)

var (
	// ErrSnapshotNotFound indicates the requested snapshot ID does not exist in .scratch/snapshots/.
	ErrSnapshotNotFound = errors.New("snapshot not found")
	// ErrConflict indicates the on-disk state diverged from the recorded post-edit SHA256.
	ErrConflict = errors.New("snapshot conflict")
	// ErrCorruptBlob indicates a content blob is missing or failed SHA256 checksum verification.
	ErrCorruptBlob = errors.New("corrupt snapshot blob")
	// ErrInvalidSnapshotID indicates the snapshot ID contains invalid characters or directory traversal.
	ErrInvalidSnapshotID = errors.New("invalid snapshot id")
	// ErrPathOutsideWorkspace indicates a file path traverses outside the workspace root.
	ErrPathOutsideWorkspace = errors.New("path outside workspace")
)

// ConflictError represents a hash divergence or missing file during preflight conflict checking.
type ConflictError struct {
	Path           string
	ExpectedSHA256 string
	ActualSHA256   string
	Message        string
}

func (e *ConflictError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("conflict in %s: %s", e.Path, e.Message)
	}
	return fmt.Sprintf("conflict in %s: expected post-edit sha256 %s, actual on-disk sha256 %s", e.Path, e.ExpectedSHA256, e.ActualSHA256)
}

func (e *ConflictError) Unwrap() error {
	return ErrConflict
}
