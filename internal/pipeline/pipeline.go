// Package pipeline orchestrates atomic writes, code formatting, and compiler diagnostic reporting.
package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// WriteAtomic writes data to a file atomically via sibling temp file, fsync, and rename.
func WriteAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Ensure advancing mtime before renaming (ADR-0010)
	if fi, err := os.Stat(targetPath); err == nil {
		oldTime := fi.ModTime()
		now := time.Now()
		if !now.After(oldTime) {
			newTime := oldTime.Add(2 * time.Millisecond)
			_ = os.Chtimes(tmpName, newTime, newTime)
		}
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("rename to target: %w", err)
	}

	return nil
}

// Format runs gofmt on the specified paths or directories.
func Format(ctx context.Context, workDir string, paths ...string) error {
	args := append([]string{"-w"}, paths...)
	// #nosec G204 -- canonical formatter invocation
	cmd := exec.CommandContext(ctx, "gofmt", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = os.Environ()

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofmt failed: %w: %s", err, errBuf.String())
	}
	return nil
}

// CheckDiagnostics collects compiler/linter diagnostics without rolling back intermediate states (ADR-0004).
func CheckDiagnostics(ctx context.Context, workDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = os.Environ()

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	_ = cmd.Run() // Exit status may be non-zero on compiler errors, which is normal for diagnostic reporting.

	output := out.String()
	if output == "" {
		return nil, nil
	}

	var diagnostics []string
	for line := range bytes.SplitSeq([]byte(output), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			diagnostics = append(diagnostics, string(trimmed))
		}
	}
	return diagnostics, nil
}
