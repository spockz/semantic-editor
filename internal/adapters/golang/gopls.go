// Package golang provides the language adapter driving gopls for Go refactorings.
package golang

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"semedit/internal/gocache"
)

var (
	// ErrGoplsNotFound indicates the gopls executable could not be located.
	ErrGoplsNotFound = errors.New("gopls executable not found")
	// ErrRenameFailed indicates gopls rename failed to execute.
	ErrRenameFailed = errors.New("gopls rename failed")
)

// FindGopls locates the gopls binary across system PATH and Go environment directories.
func FindGopls() (string, error) {
	if p, err := exec.LookPath("gopls"); err == nil {
		return p, nil
	}

	candidates := []string{
		filepath.Join(os.Getenv("GOBIN"), "gopls"),
		filepath.Join(os.Getenv("GOPATH"), "bin", "gopls"),
		filepath.Join(build.Default.GOPATH, "bin", "gopls"),
		filepath.Join(os.Getenv("HOME"), "go", "bin", "gopls"),
		"/opt/homebrew/bin/gopls",
		"/usr/local/bin/gopls",
	}

	for _, cand := range candidates {
		if cand == "gopls" || cand == "" {
			continue
		}
		cleanCand := filepath.Clean(cand)
		// #nosec G703 -- verifying known compiler candidate binary paths
		if info, err := os.Stat(cleanCand); err == nil && !info.IsDir() {
			return cleanCand, nil
		}
	}

	return "", ErrGoplsNotFound
}

// Rename executes gopls rename against an exact file coordinate.
func Rename(ctx context.Context, workDir string, file string, line int, col int, newName string) error {
	goplsPath, err := FindGopls()
	if err != nil {
		return err
	}

	target := fmt.Sprintf("%s:%d:%d", file, line, col)
	// #nosec G204 -- goplsPath is resolved from validated tool locations
	cmd := exec.CommandContext(ctx, goplsPath, "rename", "-w", target, newName)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env, err = gocache.Environment(workDir)
	if err != nil {
		return fmt.Errorf("prepare gopls environment: %w", err)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %w: %s", ErrRenameFailed, err, strings.TrimSpace(out.String()))
	}

	return nil
}
