// Package atomicfile stages complete outputs before publication to minimize partial batch visibility.
package atomicfile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"
	"time"
)

// TemporaryPath identifies the owned staging file so membership validation can exclude it.
func (s *StagedFile) TemporaryPath() string { return s.temporary }

// StagedFile owns one fully synced temporary output awaiting publication.
type StagedFile struct{ target, temporary string }

// Discard removes an unpublished temporary output.
func (s *StagedFile) Discard() { _ = os.Remove(s.temporary) }

// Publish replaces the target with its already synced output.
func (s *StagedFile) Publish() error {
	if err := traceDiskOperation("rename", func() error { return os.Rename(s.temporary, s.target) }); err != nil {
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
	var temporary *os.File
	err := traceDiskOperation("create", func() error {
		var createErr error
		temporary, createErr = os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-*")
		return createErr
	})
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
	if err := traceDiskOperation("write", func() error { _, writeErr := temporary.Write(data); return writeErr }); err != nil {
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
	if err := traceDiskOperation("sync", temporary.Sync); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("%w: sync temp file: %w", ErrAtomicWrite, err)
	}
	if err := traceDiskOperation("close", temporary.Close); err != nil {
		return nil, fmt.Errorf("%w: close temp file: %w", ErrAtomicWrite, err)
	}
	succeeded = true
	return staged, nil
}

func traceDiskOperation(name string, operation func() error) error {
	if !trace.IsEnabled() {
		return operation()
	}
	region := trace.StartRegion(context.Background(), "disk."+name)
	defer region.End()
	return operation()
}
