// Package atomicfile stages complete outputs before publication to minimize partial batch visibility.
package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StagedFile owns one fully synced temporary output awaiting publication.
type StagedFile struct{ target, temporary string }

// Discard removes an unpublished temporary output.
func (s *StagedFile) Discard() { _ = os.Remove(s.temporary) }

// Publish replaces the target with its already synced output.
func (s *StagedFile) Publish() error {
	if err := os.Rename(s.temporary, s.target); err != nil {
		return fmt.Errorf("%w: rename to target: %w", ErrAtomicWrite, err)
	}
	return nil
}

// Stage prepares output bytes, permissions, and an advancing modification time.
func Stage(targetPath string, data []byte) (*StagedFile, error) {
	var oldInfo os.FileInfo
	if info, err := os.Stat(targetPath); err == nil {
		oldInfo = info
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: stat target: %w", ErrAtomicWrite, err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return nil, fmt.Errorf("%w: create temp file: %w", ErrAtomicWrite, err)
	}
	staged := &StagedFile{target: targetPath, temporary: temporary.Name()}
	succeeded := false
	defer func() {
		if !succeeded {
			staged.Discard()
		}
	}()
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("%w: write temp file: %w", ErrAtomicWrite, err)
	}
	if oldInfo != nil {
		if err := temporary.Chmod(oldInfo.Mode().Perm()); err != nil {
			_ = temporary.Close()
			return nil, fmt.Errorf("%w: preserve target permissions: %w", ErrAtomicWrite, err)
		}
		newTime := time.Now()
		if minimum := oldInfo.ModTime().Add(time.Millisecond); newTime.Before(minimum) {
			newTime = minimum
		}
		if err := os.Chtimes(staged.temporary, newTime, newTime); err != nil {
			_ = temporary.Close()
			return nil, fmt.Errorf("%w: set target mtime: %w", ErrAtomicWrite, err)
		}
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("%w: sync temp file: %w", ErrAtomicWrite, err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("%w: close temp file: %w", ErrAtomicWrite, err)
	}
	succeeded = true
	return staged, nil
}
