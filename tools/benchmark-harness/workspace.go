// Shared fixture preparation lives here so every harness receives the same dependency and cache setup.

package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/gocache"
	"slices"
	"strings"
	"time"
)

func fixtureGoEnvironment(ctx context.Context, workDir string) ([]string, error) {
	env, err := gocache.Environment(gocache.WithBaseDir(ctx, filepath.Join(workDir, ".scratch", "go")), workDir)
	if err != nil {
		return nil, fmt.Errorf("prepare fixture Go environment: %w", err)
	}
	// Fixture builds measure code correctness, not Git revision stamping.
	return replaceEnvironment(env, map[string]string{"GOFLAGS": ("-mod=readonly -buildvcs=false")}), nil
}

func runSemeditVerifyWarmup(ctx context.Context, workDir, file, binaryPath string) (time.Duration, error) {
	if !filepath.IsLocal(filepath.FromSlash(file)) {
		return 0, fmt.Errorf("prewarm verify file %q must be fixture-local", file)
	}
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return 0, fmt.Errorf("prepare prewarm Go environment: %w", err)
	}
	for _, subdir := range []string{"build", "mod"} {
		cacheDir := filepath.Join(workDir, ".scratch", "go", subdir)
		entries, readErr := os.ReadDir(cacheDir)
		if readErr != nil {
			return 0, fmt.Errorf("read prewarm %s cache: %w", subdir, readErr)
		}
		if len(entries) != 0 {
			return 0, fmt.Errorf("prewarm %s cache is not empty in %s", subdir, cacheDir)
		}
	}
	// #nosec G204 -- binaryPath is the repository-owned semedit-next binary selected by the runner.
	cmd := exec.CommandContext(ctx, binaryPath, "verify", "--file", filepath.FromSlash(file), "--language", "go", "--check-only")
	cmd.Dir = workDir
	cmd.Env = env
	started := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(started)
	if err != nil {
		return duration, fmt.Errorf("prewarm semedit verify: %w: %s", err, boundedDiagnostic(output, 2048))
	}
	return duration, nil
}

func prepareFixtureGoDependencies(ctx context.Context, workDir string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if _, err := os.Stat(filepath.Join(workDir, "go.mod")); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect fixture module: %w", err)
	}
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return fmt.Errorf("prepare dependency Go environment: %w", err)
	}
	cmd := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-deps", "-test", "./...")
	cmd.Dir = workDir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("prepare offline fixture dependencies: %w: %s", err, boundedDiagnostic(output, 2048))
	}
	return nil
}

func canonicalFixtureRoot(workDir string) (string, error) {
	absoluteRoot, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve fixture root %q: %w", workDir, err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("canonicalize fixture root %q: %w", absoluteRoot, err)
	}
	return filepath.Clean(canonicalRoot), nil
}

func prepareRuntimeLayout(workDir string, directories []string, files map[string][]byte) (retErr error) {
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return fmt.Errorf("create runtime workspace: %w", err)
	}
	root, err := os.OpenRoot(workDir)
	if err != nil {
		return fmt.Errorf("open runtime workspace: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	for _, directory := range directories {
		if !filepath.IsLocal(directory) {
			return fmt.Errorf("runtime directory %q must be fixture-local", directory)
		}
		if err := root.MkdirAll(directory, 0o750); err != nil {
			return fmt.Errorf("create runtime directory %q: %w", directory, err)
		}
	}
	for path, data := range files {
		if !filepath.IsLocal(path) {
			return fmt.Errorf("runtime file %q must be fixture-local", path)
		}
		if err := root.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return fmt.Errorf("create runtime file directory: %w", err)
		}
		if err := writeRuntimeFile(root, path, data); err != nil {
			return err
		}
	}
	return nil
}

func writeRuntimeFile(root *os.Root, path string, data []byte) (retErr error) {
	temporaryPath := path + fmt.Sprintf(".tmp-%d", time.Now().UnixNano())
	file, err := root.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create runtime temporary file: %w", err)
	}
	defer func() {
		if err := root.Remove(temporaryPath); err != nil && !os.IsNotExist(err) {
			retErr = errors.Join(retErr, fmt.Errorf("remove runtime temporary file: %w", err))
		}
	}()
	if _, err := file.Write(data); err != nil {
		return errors.Join(fmt.Errorf("write runtime file: %w", err), file.Close())
	}
	if oldInfo, err := root.Stat(path); err == nil {
		nextTime := time.Now()
		if minimum := oldInfo.ModTime().Add(time.Millisecond); nextTime.Before(minimum) {
			nextTime = minimum
		}
		if timestampErr := root.Chtimes(temporaryPath, nextTime, nextTime); timestampErr != nil {
			return errors.Join(fmt.Errorf("advance runtime file mtime: %w", timestampErr), file.Close())
		}
	} else if !os.IsNotExist(err) {
		return errors.Join(fmt.Errorf("inspect runtime file: %w", err), file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync runtime file: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime file: %w", err)
	}
	if err := root.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish runtime file %q: %w", path, err)
	}
	return nil
}

func environmentMap(env []string) map[string]string {
	values := make(map[string]string, 8)
	allowed := map[string]struct{}{
		"GOENV": {}, "GOCACHE": {}, "GOMODCACHE": {}, "GOTMPDIR": {},
		"GOBIN": {}, "GOPATH": {}, "GOFLAGS": {}, "GOWORK": {},
	}
	for _, entry := range env {
		key, value, found := strings.Cut(entry, "=")
		if found {
			if _, ok := allowed[key]; !ok {
				continue
			}
			values[key] = value
		}
	}
	return values
}

func environmentLookup(env []string, key string) string {
	for _, entry := range env {
		entryKey, value, found := strings.Cut(entry, "=")
		if found && entryKey == key {
			return value
		}
	}
	return ""
}

func replaceEnvironment(env []string, replacements map[string]string) []string {
	result := make([]string, 0, len(env)+len(replacements))
	for _, entry := range env {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, replaced := replacements[key]; !replaced {
			result = append(result, entry)
		}
	}
	keys := slices.Sorted(maps.Keys(replacements))
	for _, key := range keys {
		result = append(result, key+"="+replacements[key])
	}
	return result
}
