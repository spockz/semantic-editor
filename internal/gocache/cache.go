// Package gocache creates project-local Go tool state for semantic subprocesses.
package gocache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type baseDirContextKey struct{}

// WithBaseDir selects the directory holding Go subprocess state for one request tree.
func WithBaseDir(ctx context.Context, baseDir string) context.Context {
	return context.WithValue(ctx, baseDirContextKey{}, strings.TrimSpace(baseDir))
}

func resolveBaseDir(ctx context.Context, workDir string) (string, error) {
	if baseDir, _ := ctx.Value(baseDirContextKey{}).(string); baseDir != "" {
		resolved, err := filepath.Abs(baseDir)
		if err != nil {
			return "", fmt.Errorf("resolve Go base directory: %w", err)
		}
		return resolved, nil
	}
	if workDir == "" {
		return "", fmt.Errorf("go workspace directory is empty")
	}
	root, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve Go workspace directory: %w", err)
	}
	return filepath.Join(root, ".scratch", "go"), nil
}

func verifyStateDirectory(dir string) error {
	// #nosec G703 -- dir is the caller-selected or process-configured Go cache root.
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create Go state directory %q: %w", dir, err)
	}
	if _, err := os.ReadDir(dir); err != nil {
		return fmt.Errorf("read Go state directory %q: %w", dir, err)
	}
	probe, err := os.CreateTemp(dir, ".semedit-go-state-")
	if err != nil {
		return fmt.Errorf("write Go state directory %q: %w", dir, err)
	}
	name := probe.Name()
	_ = probe.Close()
	// #nosec G703 -- name comes from CreateTemp in dir with a fixed prefix.
	_ = os.Remove(name)
	return nil
}

// Environment returns an environment whose Go state remains below the workspace scratch directory.
func Environment(ctx context.Context, workDir string) ([]string, error) {
	explicitBaseDir := ctx.Value(baseDirContextKey{}) != nil
	baseDir, err := resolveBaseDir(ctx, workDir)
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(baseDir, "build")
	moduleCacheDir := filepath.Join(baseDir, "mod")
	tempDir := filepath.Join(baseDir, "tmp")
	binDir := filepath.Join(baseDir, "bin")
	goplsCacheDir := filepath.Join(baseDir, "gopls")
	if !explicitBaseDir {
		if shared := strings.TrimSpace(os.Getenv("SEMEDIT_SHARED_GOCACHE")); shared != "" {
			if absShared, resolveErr := filepath.Abs(shared); resolveErr == nil {
				cacheDir = absShared
			}
		}
		if shared := strings.TrimSpace(os.Getenv("SEMEDIT_SHARED_GOMODCACHE")); shared != "" {
			if absShared, resolveErr := filepath.Abs(shared); resolveErr == nil {
				moduleCacheDir = absShared
			}
		}
		if shared := strings.TrimSpace(os.Getenv("SEMEDIT_SHARED_GOPLSCACHE")); shared != "" {
			if absShared, resolveErr := filepath.Abs(shared); resolveErr == nil {
				goplsCacheDir = absShared
			}
		}
	}
	for _, dir := range []string{baseDir, cacheDir, moduleCacheDir, tempDir, binDir, goplsCacheDir} {
		if err := verifyStateDirectory(dir); err != nil {
			return nil, err
		}
	}

	managed := map[string]struct{}{
		"GOENV":      {},
		"GOCACHE":    {},
		"GOMODCACHE": {},
		"GOTMPDIR":   {},
		"TMPDIR":     {},
		"TEMP":       {},
		"TMP":        {},
		"GOBIN":      {},
		"GOPATH":     {},
		"GOPLSCACHE": {},
		"GOFLAGS":    {},
		"GOWORK":     {},
	}
	env := make([]string, 0, len(os.Environ())+len(managed))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if _, ok := managed[key]; !ok {
			env = append(env, value)
		}
	}
	return append(env,
		"GOENV="+filepath.Join(baseDir, "env"),
		"GOCACHE="+cacheDir,
		"GOMODCACHE="+moduleCacheDir,
		"GOTMPDIR="+tempDir,
		"TMPDIR="+tempDir,
		"TEMP="+tempDir,
		"TMP="+tempDir,
		"GOBIN="+binDir,
		"GOPATH="+baseDir,
		"GOPLSCACHE="+goplsCacheDir,
		"GOFLAGS=",
		"GOWORK=off",
	), nil
}
