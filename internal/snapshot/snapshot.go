// Package snapshot provides transactional pre-edit snapshots and conflict-checked rollback for semantic refactorings.
package snapshot

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
)

// Manifest stores metadata and content-addressed file records for a pre-edit snapshot.
type Manifest struct {
	ID          string       `json:"id"`
	CreatedAt   time.Time    `json:"created_at"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	BaseCommit  string       `json:"base_commit,omitempty"`
	Files       []FileRecord `json:"files"`
}

// FileRecord represents a snapshot entry for a single workspace file.
type FileRecord struct {
	Path           string      `json:"path"`
	Mode           os.FileMode `json:"mode"`
	PreimageSHA256 string      `json:"preimage_sha256"`
	PostEditSHA256 string      `json:"post_edit_sha256,omitempty"`
}

// CreateOptions specifies parameters for capturing a snapshot.
type CreateOptions struct {
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Paths       []string `json:"paths,omitempty"`
}

// CreateResult returns summary metadata of a newly created snapshot.
type CreateResult struct {
	SnapshotID    string    `json:"snapshot_id"`
	CreatedAt     time.Time `json:"created_at"`
	CapturedFiles []string  `json:"captured_files"`
}

// UndoResult returns details of a restored snapshot state.
type UndoResult struct {
	RestoredFiles    []string                  `json:"restored_files"`
	RestoredDiff     string                    `json:"restored_diff"`
	Diagnostics      []string                  `json:"diagnostics"`
	DiagnosticStatus string                    `json:"diagnostic_status"`
	DiagnosticError  string                    `json:"diagnostic_error,omitempty"`
	Delta            *pipeline.DiagnosticDelta `json:"delta,omitempty"`
}

// Create captures pre-edit state of specified files or the workspace root into .scratch/snapshots/<id>/.
func Create(ctx context.Context, workDir string, opts CreateOptions) (*CreateResult, error) {
	absWorkDir, resolveErr := resolveWorkDir(workDir)
	if resolveErr != nil {
		return nil, fmt.Errorf("resolve workdir: %w", resolveErr)
	}

	snapshotID, idErr := generateSnapshotID()
	if idErr != nil {
		return nil, fmt.Errorf("generate snapshot id: %w", idErr)
	}

	snapDir := filepath.Join(absWorkDir, ".scratch", "snapshots", snapshotID)
	blobsDir := filepath.Join(snapDir, "blobs")
	if err := os.MkdirAll(blobsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create snapshot directory: %w", err)
	}

	filePaths, collectErr := collectFilesToCapture(absWorkDir, opts.Paths)
	if collectErr != nil {
		return nil, fmt.Errorf("collect files to capture: %w", collectErr)
	}

	records := make([]FileRecord, 0, len(filePaths))
	captured := make([]string, 0, len(filePaths))

	for _, relPath := range filePaths {
		absPath := filepath.Join(absWorkDir, filepath.FromSlash(relPath))
		fi, statErr := os.Stat(absPath)
		if statErr != nil {
			return nil, fmt.Errorf("stat target file %s: %w", relPath, statErr)
		}

		// #nosec G304 -- reading workspace file to capture snapshot preimage
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return nil, fmt.Errorf("read target file %s: %w", relPath, readErr)
		}

		sha := computeSHA256(data)
		blobPath := filepath.Join(blobsDir, sha)

		// #nosec G703 -- blobPath is constructed from sha256 hex string
		if _, err := os.Stat(blobPath); errors.Is(err, os.ErrNotExist) {
			if err := pipeline.WriteAtomic(blobPath, data); err != nil {
				return nil, fmt.Errorf("write blob for %s: %w", relPath, err)
			}
		}

		records = append(records, FileRecord{
			Path:           relPath,
			Mode:           fi.Mode(),
			PreimageSHA256: sha,
		})
		captured = append(captured, relPath)
	}

	baseCommit := getBaseGitCommit(ctx, absWorkDir)

	manifest := Manifest{
		ID:          snapshotID,
		CreatedAt:   time.Now().UTC(),
		Label:       opts.Label,
		Description: opts.Description,
		BaseCommit:  baseCommit,
		Files:       records,
	}

	manifestBytes, marshalErr := json.MarshalIndent(manifest, "", "  ")
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal manifest: %w", marshalErr)
	}

	manifestPath := filepath.Join(snapDir, "manifest.json")
	if err := pipeline.WriteAtomic(manifestPath, manifestBytes); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return &CreateResult{
		SnapshotID:    snapshotID,
		CreatedAt:     manifest.CreatedAt,
		CapturedFiles: captured,
	}, nil
}

// RecordPostEdit scans workspace files recorded in the snapshot manifest and updates their PostEditSHA256.
func RecordPostEdit(_ context.Context, workDir string, snapshotID string) (*Manifest, error) {
	absWorkDir, resolveErr := resolveWorkDir(workDir)
	if resolveErr != nil {
		return nil, fmt.Errorf("resolve workdir: %w", resolveErr)
	}

	resolvedID, idErr := resolveSnapshotID(absWorkDir, snapshotID)
	if idErr != nil {
		return nil, idErr
	}

	manifest, manifestPath, loadErr := loadManifest(absWorkDir, resolvedID)
	if loadErr != nil {
		return nil, loadErr
	}

	for i := range manifest.Files {
		rec := &manifest.Files[i]
		absPath := filepath.Join(absWorkDir, filepath.FromSlash(rec.Path))
		// #nosec G304 -- reading workspace file to record post-edit sha256
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				rec.PostEditSHA256 = "deleted"
				continue
			}
			return nil, fmt.Errorf("read file %s: %w", rec.Path, readErr)
		}
		rec.PostEditSHA256 = computeSHA256(data)
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}

	if err := pipeline.WriteAtomic(manifestPath, manifestBytes); err != nil {
		return nil, fmt.Errorf("write updated manifest: %w", err)
	}

	return manifest, nil
}

type fileRestorePlan struct {
	record       FileRecord
	absPath      string
	currentBytes []byte
	blobBytes    []byte
	needsWrite   bool
}

type restoreFileState struct {
	bytes  []byte
	sha256 string
	exists bool
}

type restoreResult struct {
	files []string
	diff  string
}

// Undo verifies preimage/post-edit integrity and atomically restores files from the specified snapshot.
func Undo(ctx context.Context, workDir string, snapshotID string) (*UndoResult, error) {
	absWorkDir, err := resolveWorkDir(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}

	resolvedID, err := resolveSnapshotID(absWorkDir, snapshotID)
	if err != nil {
		return nil, err
	}

	manifest, _, err := loadManifest(absWorkDir, resolvedID)
	if err != nil {
		return nil, err
	}

	diagsBefore, beforeErr := pipeline.CheckDiagnostics(ctx, absWorkDir)
	plans, err := preflightRestorePlans(absWorkDir, resolvedID, manifest)
	if err != nil {
		return nil, err
	}
	restored, err := restorePlans(plans)
	if err != nil {
		return nil, err
	}

	return undoResultWithDiagnostics(ctx, absWorkDir, diagsBefore, beforeErr, restored)
}

func undoResultWithDiagnostics(ctx context.Context, workDir string, before []string, beforeErr error, restored restoreResult) (*UndoResult, error) {
	after, afterErr := pipeline.CheckDiagnostics(ctx, workDir)
	result := &UndoResult{
		RestoredFiles:    restored.files,
		RestoredDiff:     restored.diff,
		Diagnostics:      after,
		DiagnosticStatus: "unavailable",
	}
	if beforeErr != nil || afterErr != nil {
		result.DiagnosticError = errors.Join(beforeErr, afterErr).Error()
		return result, nil
	}
	delta := pipeline.ComputeDelta(before, after)
	result.DiagnosticStatus = "available"
	result.Delta = &delta
	return result, nil
}

func preflightRestorePlans(root, snapshotID string, manifest *Manifest) ([]fileRestorePlan, error) {
	blobsDir := filepath.Join(root, ".scratch", "snapshots", snapshotID, "blobs")
	plans := make([]fileRestorePlan, 0, len(manifest.Files))
	for _, record := range manifest.Files {
		plan, err := prepareRestorePlan(root, blobsDir, record)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func prepareRestorePlan(root, blobsDir string, record FileRecord) (fileRestorePlan, error) {
	absPath := filepath.Join(root, filepath.FromSlash(record.Path))
	relToRoot, err := filepath.Rel(root, absPath)
	if err != nil || strings.HasPrefix(relToRoot, "..") || filepath.IsAbs(relToRoot) {
		return fileRestorePlan{}, fmt.Errorf("%w: %s", ErrPathOutsideWorkspace, record.Path)
	}
	blobData, err := readVerifiedBlob(blobsDir, record)
	if err != nil {
		return fileRestorePlan{}, err
	}
	current, err := readCurrentRestoreFile(absPath, record.Path)
	if err != nil {
		return fileRestorePlan{}, err
	}
	if err := validateRestoreConflict(record, current.sha256, current.exists); err != nil {
		return fileRestorePlan{}, err
	}
	return fileRestorePlan{record: record, absPath: absPath, currentBytes: current.bytes, blobBytes: blobData, needsWrite: !current.exists || current.sha256 != record.PreimageSHA256}, nil
}

func readVerifiedBlob(blobsDir string, record FileRecord) ([]byte, error) {
	blobPath := filepath.Join(blobsDir, record.PreimageSHA256)
	// #nosec G304 -- reading snapshot preimage blob
	data, err := os.ReadFile(blobPath)
	if err != nil {
		return nil, fmt.Errorf("%w: missing blob %s for %s: %w", ErrCorruptBlob, record.PreimageSHA256, record.Path, err)
	}
	if actualSHA := computeSHA256(data); actualSHA != record.PreimageSHA256 {
		return nil, fmt.Errorf("%w: blob checksum mismatch for %s: expected %s, got %s", ErrCorruptBlob, record.Path, record.PreimageSHA256, actualSHA)
	}
	return data, nil
}

func readCurrentRestoreFile(absPath, relativePath string) (restoreFileState, error) {
	// #nosec G304 -- reading workspace file to preflight conflict check
	data, err := os.ReadFile(absPath)
	if errors.Is(err, os.ErrNotExist) {
		return restoreFileState{}, nil
	}
	if err != nil {
		return restoreFileState{}, fmt.Errorf("read on-disk file %s: %w", relativePath, err)
	}
	return restoreFileState{bytes: data, sha256: computeSHA256(data), exists: true}, nil
}

func validateRestoreConflict(record FileRecord, currentSHA string, fileExists bool) error {
	if record.PostEditSHA256 == "" {
		return nil
	}
	if record.PostEditSHA256 == "deleted" {
		if fileExists && currentSHA != record.PreimageSHA256 {
			return &ConflictError{Path: record.Path, ExpectedSHA256: "deleted", ActualSHA256: currentSHA, Message: "file exists on disk but was marked deleted in post-edit manifest"}
		}
		return nil
	}
	if !fileExists {
		return &ConflictError{Path: record.Path, ExpectedSHA256: record.PostEditSHA256, ActualSHA256: "missing", Message: "file missing on disk but recorded in post-edit manifest"}
	}
	if currentSHA != record.PostEditSHA256 && currentSHA != record.PreimageSHA256 {
		return &ConflictError{Path: record.Path, ExpectedSHA256: record.PostEditSHA256, ActualSHA256: currentSHA}
	}
	return nil
}

func restorePlans(plans []fileRestorePlan) (restoreResult, error) {
	result := restoreResult{}
	var diff strings.Builder
	for _, plan := range plans {
		if !plan.needsWrite {
			continue
		}
		if err := restorePlan(plan); err != nil {
			return restoreResult{}, err
		}
		result.files = append(result.files, plan.record.Path)
		appendRestoreDiff(&diff, plan)
	}
	result.diff = diff.String()
	return result, nil
}

func restorePlan(plan fileRestorePlan) error {
	if err := os.MkdirAll(filepath.Dir(plan.absPath), 0o750); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", plan.record.Path, err)
	}
	if err := pipeline.WriteAtomic(plan.absPath, plan.blobBytes); err != nil {
		return fmt.Errorf("atomic write restore for %s: %w", plan.record.Path, err)
	}
	if plan.record.Mode != 0 {
		if err := os.Chmod(plan.absPath, plan.record.Mode); err != nil {
			return fmt.Errorf("restore mode for %s after content was restored: %w", plan.record.Path, err)
		}
	}
	return nil
}

func appendRestoreDiff(builder *strings.Builder, plan fileRestorePlan) {
	diff := astedit.UnifiedDiff(plan.record.Path, string(plan.currentBytes), string(plan.blobBytes))
	if diff == "" {
		return
	}
	if builder.Len() > 0 {
		builder.WriteString("\n")
	}
	builder.WriteString(diff)
}

// GetManifest retrieves the parsed manifest for a given snapshot ID or "latest".
func GetManifest(workDir string, snapshotID string) (*Manifest, error) {
	absWorkDir, err := resolveWorkDir(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}

	resolvedID, err := resolveSnapshotID(absWorkDir, snapshotID)
	if err != nil {
		return nil, err
	}

	manifest, _, err := loadManifest(absWorkDir, resolvedID)
	return manifest, err
}

// ListSnapshots returns all stored snapshot manifests in chronological order (newest first).
func ListSnapshots(workDir string) ([]*Manifest, error) {
	absWorkDir, err := resolveWorkDir(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}

	snapsDir := filepath.Join(absWorkDir, ".scratch", "snapshots")
	entries, err := os.ReadDir(snapsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}

	var manifests []*Manifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(snapsDir, entry.Name(), "manifest.json")
		// #nosec G304 -- reading snapshot manifest
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err == nil {
			manifests = append(manifests, &m)
		}
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].CreatedAt.After(manifests[j].CreatedAt)
	})

	return manifests, nil
}

func resolveWorkDir(workDir string) (string, error) {
	if workDir == "" {
		workDir = "."
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func resolveSnapshotID(absWorkDir string, id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", ErrInvalidSnapshotID
	}
	if trimmed == "latest" {
		manifests, err := ListSnapshots(absWorkDir)
		if err != nil {
			return "", err
		}
		if len(manifests) == 0 {
			return "", ErrSnapshotNotFound
		}
		return manifests[0].ID, nil
	}

	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") || strings.Contains(trimmed, "..") {
		return "", ErrInvalidSnapshotID
	}

	return trimmed, nil
}

func loadManifest(absWorkDir string, snapshotID string) (*Manifest, string, error) {
	manifestPath := filepath.Join(absWorkDir, ".scratch", "snapshots", snapshotID, "manifest.json")
	// #nosec G304 -- reading snapshot manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrSnapshotNotFound
		}
		return nil, "", fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", fmt.Errorf("unmarshal manifest: %w", err)
	}

	return &manifest, manifestPath, nil
}

func collectFilesToCapture(absWorkDir string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return scanWorkspaceFiles(absWorkDir)
	}

	var results []string
	seen := make(map[string]bool)

	for _, p := range paths {
		cleanPath := filepath.Clean(p)
		var absTarget string
		if filepath.IsAbs(cleanPath) {
			absTarget = cleanPath
		} else {
			absTarget = filepath.Join(absWorkDir, cleanPath)
		}

		rel, err := filepath.Rel(absWorkDir, absTarget)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("%w: %s", ErrPathOutsideWorkspace, p)
		}

		fi, err := os.Stat(absTarget)
		if err != nil {
			return nil, fmt.Errorf("stat target path %s: %w", p, err)
		}

		if fi.IsDir() {
			subFiles, err := scanDirectory(absWorkDir, absTarget)
			if err != nil {
				return nil, err
			}
			for _, sf := range subFiles {
				if !seen[sf] {
					seen[sf] = true
					results = append(results, sf)
				}
			}
		} else {
			slashRel := filepath.ToSlash(rel)
			if !seen[slashRel] {
				seen[slashRel] = true
				results = append(results, slashRel)
			}
		}
	}

	slices.Sort(results)
	return results, nil
}

func scanWorkspaceFiles(absWorkDir string) ([]string, error) {
	return scanDirectory(absWorkDir, absWorkDir)
}

func scanDirectory(absWorkDir string, rootDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".scratch" || name == "vendor" || name == "node_modules" || name == "dist" || name == "bin" || sourcefiles.IsBenchmarkDataDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(absWorkDir, path)
		if err != nil {
			return err
		}

		slashRel := filepath.ToSlash(rel)
		files = append(files, slashRel)
		return nil
	})

	if err != nil {
		return nil, err
	}

	slices.Sort(files)
	return files, nil
}

func computeSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func generateSnapshotID() (string, error) {
	b := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return fmt.Sprintf("snap-%s-%x", time.Now().UTC().Format("20060102-150405"), b), nil
}

func getBaseGitCommit(ctx context.Context, workDir string) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
