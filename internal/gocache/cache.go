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

// Environment returns an environment whose Go state remains below the workspace scratch directory.
func Environment(ctx context.Context, workDir string) ([]string, error) {
	baseDir, _ := ctx.Value(baseDirContextKey{}).(string)
	var err error
	if baseDir == "" {
		if workDir == "" {
			return nil, fmt.Errorf("go workspace directory is empty")
		}
		root, err := filepath.Abs(workDir)
		if err != nil {
			return nil, fmt.Errorf("resolve Go workspace directory: %w", err)
		}
		baseDir = filepath.Join(root, ".scratch", "go")
	} else {
		baseDir, err = filepath.Abs(baseDir)
		if err != nil {
			return nil, fmt.Errorf("resolve Go base directory: %w", err)
		}
	}

	cacheDir := filepath.Join(baseDir, "build")
	moduleCacheDir := filepath.Join(baseDir, "mod")
	tempDir := filepath.Join(baseDir, "tmp")
	binDir := filepath.Join(baseDir, "bin")
	for _, dir := range []string{baseDir, cacheDir, moduleCacheDir, tempDir, binDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create Go state directory %q: %w", dir, err)
		}
		if _, err := os.ReadDir(dir); err != nil {
			return nil, fmt.Errorf("read Go state directory %q: %w", dir, err)
		}
		probe, err := os.CreateTemp(dir, ".semedit-go-state-")
		if err != nil {
			return nil, fmt.Errorf("write Go state directory %q: %w", dir, err)
		}
		name := probe.Name()
		if err := probe.Close(); err != nil {
			_ = os.Remove(name)
			return nil, fmt.Errorf("close Go state probe %q: %w", name, err)
		}
		if err := os.Remove(name); err != nil {
			return nil, fmt.Errorf("remove Go state probe %q: %w", name, err)
		}
	}

	managed := map[string]struct{}{
		"GOENV":      {},
		"GOCACHE":    {},
		"GOMODCACHE": {},
		"GOTMPDIR":   {},
		"GOBIN":      {},
		"GOPATH":     {},
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
		"GOBIN="+binDir,
		"GOPATH="+baseDir,
		"GOFLAGS=",
		"GOWORK=off",
	), nil
}
