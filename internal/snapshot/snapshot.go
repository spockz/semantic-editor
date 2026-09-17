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
	"sort"
	"strings"
	"time"

	"semedit/internal/astedit"
	"semedit/internal/pipeline"
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
	RestoredFiles []string                  `json:"restored_files"`
	RestoredDiff  string                    `json:"restored_diff"`
	Diagnostics   []string                  `json:"diagnostics"`
	Delta         *pipeline.DiagnosticDelta `json:"delta,omitempty"`
}

// Create captures pre-edit state of specified files or the workspace root into .scratch/snapshots/<id>/.
func Create(ctx context.Context, workDir string, opts CreateOptions) (*CreateResult, error) {
	absWorkDir, err := resolveWorkDir(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}

	snapshotID, err := generateSnapshotID()
	if err != nil {
		return nil, fmt.Errorf("generate snapshot id: %w", err)
	}

	snapDir := filepath.Join(absWorkDir, ".scratch", "snapshots", snapshotID)
	blobsDir := filepath.Join(snapDir, "blobs")
	if err := os.MkdirAll(blobsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create snapshot directory: %w", err)
	}

	filePaths, err := collectFilesToCapture(absWorkDir, opts.Paths)
	if err != nil {
		return nil, fmt.Errorf("collect files to capture: %w", err)
	}

	records := make([]FileRecord, 0, len(filePaths))
	captured := make([]string, 0, len(filePaths))

	for _, relPath := range filePaths {
		absPath := filepath.Join(absWorkDir, filepath.FromSlash(relPath))
		fi, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("stat target file %s: %w", relPath, err)
		}

		// #nosec G304 -- reading workspace file to capture snapshot preimage
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read target file %s: %w", relPath, err)
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

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
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
	absWorkDir, err := resolveWorkDir(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}

	resolvedID, err := resolveSnapshotID(absWorkDir, snapshotID)
	if err != nil {
		return nil, err
	}

	manifest, manifestPath, err := loadManifest(absWorkDir, resolvedID)
	if err != nil {
		return nil, err
	}

	for i := range manifest.Files {
		rec := &manifest.Files[i]
		absPath := filepath.Join(absWorkDir, filepath.FromSlash(rec.Path))
		// #nosec G304 -- reading workspace file to record post-edit sha256
		data, err := os.ReadFile(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				rec.PostEditSHA256 = "deleted"
				continue
			}
			return nil, fmt.Errorf("read file %s: %w", rec.Path, err)
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

	snapDir := filepath.Join(absWorkDir, ".scratch", "snapshots", resolvedID)
	blobsDir := filepath.Join(snapDir, "blobs")

	diagsBefore, _ := pipeline.CheckDiagnostics(ctx, absWorkDir)

	plans := make([]fileRestorePlan, 0, len(manifest.Files))

	// Preflight validation pass: perform zero writes if any conflict or corrupt blob is detected
	for _, rec := range manifest.Files {
		absPath := filepath.Join(absWorkDir, filepath.FromSlash(rec.Path))

		relToRoot, err := filepath.Rel(absWorkDir, absPath)
		if err != nil || strings.HasPrefix(relToRoot, "..") || filepath.IsAbs(relToRoot) {
			return nil, fmt.Errorf("%w: %s", ErrPathOutsideWorkspace, rec.Path)
		}

		blobPath := filepath.Join(blobsDir, rec.PreimageSHA256)
		// #nosec G304 -- reading snapshot preimage blob
		blobData, err := os.ReadFile(blobPath)
		if err != nil {
			return nil, fmt.Errorf("%w: missing blob %s for %s: %w", ErrCorruptBlob, rec.PreimageSHA256, rec.Path, err)
		}
		if actualBlobSHA := computeSHA256(blobData); actualBlobSHA != rec.PreimageSHA256 {
			return nil, fmt.Errorf("%w: blob checksum mismatch for %s: expected %s, got %s",
				ErrCorruptBlob, rec.Path, rec.PreimageSHA256, actualBlobSHA)
		}

		var currentData []byte
		var currentSHA string
		fileExists := true

		// #nosec G304 -- reading workspace file to preflight conflict check
		currentData, err = os.ReadFile(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fileExists = false
			} else {
				return nil, fmt.Errorf("read on-disk file %s: %w", rec.Path, err)
			}
		} else {
			currentSHA = computeSHA256(currentData)
		}

		// Conflict check: if post-edit SHA is recorded, disk must match post-edit SHA or already be restored
		if rec.PostEditSHA256 != "" {
			if rec.PostEditSHA256 == "deleted" {
				if fileExists && currentSHA != rec.PreimageSHA256 {
					return nil, &ConflictError{
						Path:           rec.Path,
						ExpectedSHA256: "deleted",
						ActualSHA256:   currentSHA,
						Message:        "file exists on disk but was marked deleted in post-edit manifest",
					}
				}
			} else {
				if !fileExists {
					return nil, &ConflictError{
						Path:           rec.Path,
						ExpectedSHA256: rec.PostEditSHA256,
						ActualSHA256:   "missing",
						Message:        "file missing on disk but recorded in post-edit manifest",
					}
				}
				if currentSHA != rec.PostEditSHA256 && currentSHA != rec.PreimageSHA256 {
					return nil, &ConflictError{
						Path:           rec.Path,
						ExpectedSHA256: rec.PostEditSHA256,
						ActualSHA256:   currentSHA,
					}
				}
			}
		}

		needsWrite := !fileExists || currentSHA != rec.PreimageSHA256

		plans = append(plans, fileRestorePlan{
			record:       rec,
			absPath:      absPath,
			currentBytes: currentData,
			blobBytes:    blobData,
			needsWrite:   needsWrite,
		})
	}

	// Execution pass: restore blobs atomically (ADR-0010)
	var restoredFiles []string
	var diffBuilder strings.Builder

	for _, plan := range plans {
		if !plan.needsWrite {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(plan.absPath), 0o750); err != nil {
			return nil, fmt.Errorf("create parent directory for %s: %w", plan.record.Path, err)
		}

		if err := pipeline.WriteAtomic(plan.absPath, plan.blobBytes); err != nil {
			return nil, fmt.Errorf("atomic write restore for %s: %w", plan.record.Path, err)
		}

		if plan.record.Mode != 0 {
			_ = os.Chmod(plan.absPath, plan.record.Mode)
		}

		restoredFiles = append(restoredFiles, plan.record.Path)

		diff := astedit.UnifiedDiff(plan.record.Path, string(plan.currentBytes), string(plan.blobBytes))
		if diff != "" {
			if diffBuilder.Len() > 0 {
				diffBuilder.WriteString("\n")
			}
			diffBuilder.WriteString(diff)
		}
	}

	diagsAfter, _ := pipeline.CheckDiagnostics(ctx, absWorkDir)
	delta := pipeline.ComputeDelta(diagsBefore, diagsAfter)

	return &UndoResult{
		RestoredFiles: restoredFiles,
		RestoredDiff:  diffBuilder.String(),
		Diagnostics:   diagsAfter,
		Delta:         &delta,
	}, nil
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

	sort.Strings(results)
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
			if name == ".git" || name == ".scratch" || name == "vendor" || name == "node_modules" || name == "dist" || name == "bin" {
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

	sort.Strings(files)
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
