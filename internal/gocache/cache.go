// Package gocache preserves caller Go state and creates isolated state only when explicitly selected.
package gocache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type baseDirContextKey struct{}

// WithBaseDir selects the directory holding Go subprocess state for one request tree.
func WithBaseDir(ctx context.Context, baseDir string) context.Context {
	return context.WithValue(ctx, baseDirContextKey{}, strings.TrimSpace(baseDir))
}

// HasBaseDir reports whether a nonblank isolated Go state base was selected.
func HasBaseDir(ctx context.Context) bool {
	baseDir, _ := ctx.Value(baseDirContextKey{}).(string)
	return strings.TrimSpace(baseDir) != ""
}

func resolveBaseDir(ctx context.Context) (string, error) {
	if baseDir, _ := ctx.Value(baseDirContextKey{}).(string); baseDir != "" {
		resolved, err := filepath.Abs(baseDir)
		if err != nil {
			return "", fmt.Errorf("resolve Go base directory: %w", err)
		}
		return resolved, nil
	}
	return "", nil
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

// Environment preserves user Go state unless a caller explicitly selects a managed base directory.
func Environment(ctx context.Context, workDir string) ([]string, error) {
	baseDir, err := resolveBaseDir(ctx)
	if err != nil {
		return nil, err
	}
	seed, _ := ctx.Value(moduleCacheSeedKey{}).(string)
	if seed != "" && baseDir == "" {
		return nil, fmt.Errorf("offline Go module cache seed requires an explicit Go isolation base directory")
	}
	if baseDir == "" {
		return inheritedEnvironment(ctx, workDir)
	}

	cacheDir := filepath.Join(baseDir, "build")
	moduleCacheDir := filepath.Join(baseDir, "mod")
	tempDir := filepath.Join(baseDir, "tmp")
	binDir := filepath.Join(baseDir, "bin")
	goplsCacheDir := filepath.Join(baseDir, "gopls")
	for _, dir := range []string{baseDir, cacheDir, moduleCacheDir, tempDir, binDir, goplsCacheDir} {
		if err := verifyStateDirectory(dir); err != nil {
			return nil, err
		}
	}
	if err := seedModuleCache(ctx, baseDir, moduleCacheDir); err != nil {
		return nil, err
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
	goFlags := "-mod=readonly"
	if moduleWritesAllowed(ctx) {
		goFlags = ""
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
		"GOFLAGS="+goFlags,
		"GOWORK=off",
	), nil
}

func inheritedEnvironment(ctx context.Context, workDir string) ([]string, error) {
	settings, err := effectiveGoSettings(ctx, workDir)
	if err != nil {
		return nil, err
	}
	goFlags := settings["GOFLAGS"]
	if !moduleWritesAllowed(ctx) {
		goFlags = appendGoFlag(goFlags, "-mod=readonly")
	}
	env := setEnvironmentValue(os.Environ(), "GOFLAGS", goFlags)
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR"} {
		if value, exists := environmentValue(env, key); exists && value != "" {
			continue
		}
		if value := settings[key]; value != "" {
			env = setEnvironmentValue(env, key, value)
		}
	}
	return env, nil
}

func effectiveGoSettings(ctx context.Context, workDir string) (map[string]string, error) {
	requested := []string{"GOFLAGS", "GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR"}
	command := exec.CommandContext(ctx, "go", "env", "-json", "GOFLAGS", "GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR")
	goEnvDir, err := existingGoEnvDirectory(workDir)
	if err != nil {
		return nil, err
	}
	if goEnvDir != "" {
		command.Dir = goEnvDir
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("resolve effective Go environment: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	values := make(map[string]*string)
	if err := json.Unmarshal(output, &values); err != nil {
		return nil, fmt.Errorf("decode effective Go environment: %w", err)
	}
	settings := make(map[string]string, len(requested))
	for _, key := range requested {
		value, ok := values[key]
		if !ok || value == nil {
			return nil, fmt.Errorf("decode effective Go environment: missing string value for %s", key)
		}
		settings[key] = *value
	}
	return settings, nil
}

func existingGoEnvDirectory(workDir string) (string, error) {
	if strings.TrimSpace(workDir) == "" {
		return "", nil
	}
	directory, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve effective Go working directory: %w", err)
	}
	for {
		info, statErr := os.Stat(directory)
		if statErr == nil && info.IsDir() {
			return directory, nil
		}
		if statErr != nil && !os.IsNotExist(statErr) {
			return "", fmt.Errorf("inspect effective Go working directory %q: %w", directory, statErr)
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("resolve effective Go working directory: no existing directory above %q", workDir)
		}
		directory = parent
	}
}

func appendGoFlag(flags, flag string) string {
	if strings.TrimSpace(flags) == "" {
		return flag
	}
	return strings.TrimSpace(flags) + " " + flag
}

func setEnvironmentValue(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		entryKey, _, _ := strings.Cut(entry, "=")
		if entryKey != key {
			result = append(result, entry)
		}
	}
	return append(result, key+"="+value)
}

func environmentValue(environment []string, key string) (string, bool) {
	for _, entry := range environment {
		entryKey, value, _ := strings.Cut(entry, "=")
		if entryKey == key {
			return value, true
		}
	}
	return "", false
}

// IsManagedEnvironmentKey reports whether key is controlled by the Go cache environment.
func IsManagedEnvironmentKey(key string) bool {
	switch key {
	case "GOENV", "GOCACHE", "GOMODCACHE", "GOTMPDIR", "TMPDIR", "TEMP", "TMP", "GOBIN", "GOPATH", "GOPLSCACHE", "GOFLAGS", "GOWORK":
		return true
	default:
		return false
	}
}
