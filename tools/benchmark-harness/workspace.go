// Shared fixture preparation lives here so every harness receives the same dependency and cache setup.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/gocache"
	"golang.org/x/mod/module"
)

func fixtureGoEnvironment(ctx context.Context, workDir string) ([]string, error) {
	cacheContext := gocache.WithBaseDir(ctx, filepath.Join(workDir, ".scratch", "go"))
	env, err := gocache.Environment(cacheContext, workDir)
	if err != nil {
		return nil, fmt.Errorf("prepare fixture Go environment: %w", err)
	}
	// Fixture builds measure code correctness, not Git revision stamping.
	overrides := map[string]string{"GOFLAGS": "-mod=readonly -buildvcs=false"}
	if gocache.HasModuleCache(ctx) {
		overrides["GOPROXY"] = "off"
		overrides["GONOPROXY"] = "none"
		overrides["GOSUMDB"] = "off"
		overrides["GOTOOLCHAIN"] = "local"
	}
	refreshLinkerOutput, err := fixtureNeedsLinkerOutputRefresh(workDir)
	if err != nil {
		return nil, fmt.Errorf("inspect fixture filesystem for linker workaround: %w", err)
	}
	if refreshLinkerOutput {
		executable, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("locate fixture harness for Go tool wrapper: %w", err)
		}
		toolFlags, err := fixtureToolexecGOFlags(executable)
		if err != nil {
			return nil, fmt.Errorf("configure fixture Go tool wrapper: %w", err)
		}
		overrides["GOFLAGS"] += " " + toolFlags
	}
	return replaceEnvironment(env, overrides), nil
}

func runSemeditVerifyWarmup(ctx context.Context, workDir, file, binaryPath string) (time.Duration, error) {
	if !filepath.IsLocal(filepath.FromSlash(file)) {
		return 0, fmt.Errorf("prewarm verify file %q must be fixture-local", file)
	}
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return 0, fmt.Errorf("prepare prewarm Go environment: %w", err)
	}
	subdirs := []string{"build"}
	if !gocache.HasModuleCache(ctx) {
		subdirs = append(subdirs, "mod")
	}
	for _, subdir := range subdirs {
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

func prepareFixtureGoDependencies(ctx context.Context, workDir string) (retErr error) {
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
	if gocache.HasModuleCache(ctx) {
		temporaryCache, createErr := os.MkdirTemp(environmentLookup(env, "GOTMPDIR"), "dependency-preflight-build-")
		if createErr != nil {
			return fmt.Errorf("create isolated dependency preflight build cache: %w", createErr)
		}
		defer func() {
			if removeErr := os.RemoveAll(temporaryCache); removeErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove dependency preflight build cache: %w", removeErr))
			}
		}()
		env = replaceEnvironment(env, map[string]string{"GOCACHE": temporaryCache})
	}
	if preflightErr := preflightSharedModuleCache(ctx, workDir, env); preflightErr != nil {
		return fmt.Errorf("prepare offline fixture dependencies: %w", preflightErr)
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

// Inventory the module graph without extracting cached archives into the shared cache.
func preflightSharedModuleCache(ctx context.Context, workDir string, env []string) error {
	if !gocache.HasModuleCache(ctx) {
		return nil
	}
	moduleCache := environmentLookup(env, "GOMODCACHE")
	if moduleCache == "" {
		return fmt.Errorf("shared module cache path is missing from Go environment")
	}
	cmd := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-m", "-json", "all")
	cmd.Dir = workDir
	cmd.Env = env
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("inspect shared module cache: %w", ctx.Err())
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return fmt.Errorf("inspect shared module cache: %w: %s", err, boundedDiagnostic(exitErr.Stderr, 2048))
		}
		return fmt.Errorf("inspect shared module cache: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	count := 0
	for decoder.More() {
		var mod struct {
			Path    string
			Version string
			Dir     string
			Main    bool
			Replace *struct {
				Path    string
				Version string
			}
			Error *struct{ Err string }
		}
		if err := decoder.Decode(&mod); err != nil {
			return fmt.Errorf("decode shared module cache inventory: %w", err)
		}
		count++
		if mod.Error != nil {
			return fmt.Errorf("module %q is unavailable in shared cache: %s", mod.Path, mod.Error.Err)
		}
		if mod.Main {
			continue
		}
		path, version := mod.Path, mod.Version
		if mod.Replace != nil {
			if mod.Replace.Version == "" {
				if info, err := os.Stat(mod.Dir); err != nil || !info.IsDir() {
					return fmt.Errorf("local replacement for %s is unavailable at %q", mod.Path, mod.Dir)
				}
				continue
			}
			path, version = mod.Replace.Path, mod.Replace.Version
		}
		if err := requireExtractedModule(moduleCache, path, version); err != nil {
			return fmt.Errorf("module %s@%s is not fully extracted in shared cache %q: %w", path, version, moduleCache, err)
		}
	}
	if count == 0 {
		return fmt.Errorf("shared module cache inventory is empty")
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

// Require Go's extraction completion markers without relying on checksum-dependent go list Dir output.
func requireExtractedModule(cache, path, version string) error {
	escapedPath, err := module.EscapePath(path)
	if err != nil {
		return fmt.Errorf("escape module path: %w", err)
	}
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return fmt.Errorf("escape module version: %w", err)
	}
	dir := filepath.Join(cache, escapedPath+"@"+escapedVersion)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("inspect extracted module: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("extracted module %q is not a directory", dir)
	}
	archive := filepath.Join(cache, "cache", "download", escapedPath, "@v", escapedVersion)
	if _, statErr := os.Stat(archive + ".partial"); statErr == nil {
		return fmt.Errorf("module extraction is incomplete")
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect extraction marker: %w", statErr)
	}
	info, err = os.Stat(archive + ".ziphash")
	if err != nil {
		return fmt.Errorf("inspect module archive hash: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("module archive hash is not a regular file")
	}
	return nil
}

// Populate only the selected SSD module cache; callers supply a disposable manifest copy.
func downloadSharedGoModules(ctx context.Context, workDir string) error {
	env, err := gocache.Environment(gocache.WithBaseDir(ctx, filepath.Join(workDir, ".scratch", "go")), workDir)
	if err != nil {
		return fmt.Errorf("prepare module download environment: %w", err)
	}
	cmd := exec.CommandContext(ctx, "go", "mod", "download", "all")
	cmd.Dir = workDir
	cmd.Env = sharedModuleDownloadEnvironment(env)
	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("download shared Go modules: %w", ctx.Err())
		}
		return fmt.Errorf("download shared Go modules: %w: %s", err, boundedDiagnostic(output, 2048))
	}
	return nil
}

// Discover a build-selected visible Go file without coupling warmup to private AST assertions.
func fixtureGoWarmupFile(ctx context.Context, task *Task, workDir string) (file string, retErr error) {
	if strings.HasSuffix(task.Metadata.Oracle.AST.File, ".go") {
		if !filepath.IsLocal(filepath.FromSlash(task.Metadata.Oracle.AST.File)) {
			return "", fmt.Errorf("warmup target must be fixture-local")
		}
		return fixtureRelativeGoFile(workDir, filepath.Join(workDir, filepath.FromSlash(task.Metadata.Oracle.AST.File)))
	}
	if _, err := os.Stat(filepath.Join(workDir, "go.mod")); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("inspect warmup module: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		return "", err
	}
	temporaryCache, err := os.MkdirTemp(environmentLookup(env, "GOTMPDIR"), "warmup-discovery-build-")
	if err != nil {
		return "", fmt.Errorf("create warmup discovery cache: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(temporaryCache); cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove warmup discovery cache: %w", cleanupErr))
		}
	}()
	cmd := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-find", "-json", "./...")
	cmd.Dir = workDir
	cmd.Env = replaceEnvironment(env, map[string]string{"GOCACHE": temporaryCache, "GOPROXY": "off", "GONOPROXY": "none", "GOSUMDB": "off", "GOTOOLCHAIN": "local"})
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("discover Go warmup target: %w", ctx.Err())
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("discover Go warmup target: %w: %s", err, boundedDiagnostic(exitErr.Stderr, 2048))
		}
		return "", fmt.Errorf("discover Go warmup target: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	for decoder.More() {
		var pkg struct {
			Dir                                          string
			GoFiles, CgoFiles, TestGoFiles, XTestGoFiles []string
		}
		if err := decoder.Decode(&pkg); err != nil {
			return "", fmt.Errorf("decode Go warmup packages: %w", err)
		}
		for _, name := range slices.Concat(pkg.GoFiles, pkg.CgoFiles, pkg.TestGoFiles, pkg.XTestGoFiles) {
			return fixtureRelativeGoFile(workDir, filepath.Join(pkg.Dir, name))
		}
	}
	return "", nil
}

func fixtureRelativeGoFile(workDir, path string) (string, error) {
	root, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve warmup fixture: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve Go warmup file: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect Go warmup file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("warmup target %q is not a regular file", path)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve Go warmup containment: %w", err)
	}
	if !filepath.IsLocal(relative) {
		return "", fmt.Errorf("go warmup target %q escapes fixture", path)
	}
	return filepath.ToSlash(relative), nil
}

// Private-module proxy bypass must not defeat an explicitly offline setup request.
func sharedModuleDownloadEnvironment(env []string) []string {
	overrides := map[string]string{"GOTOOLCHAIN": "local"}
	if environmentLookup(env, "GOPROXY") == "off" {
		overrides["GONOPROXY"] = "none"
	}
	return replaceEnvironment(env, overrides)
}
