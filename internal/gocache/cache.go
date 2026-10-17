// Package gocache preserves caller Go state and creates isolated state only when explicitly selected.
package gocache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/spockz/semantic-editor/internal/telemetry"
)

const moduleCacheEnvironmentKey = "GOMODCACHE"

type baseDirContextKey struct{}

type moduleCacheContextKey struct{}

type environmentCacheContextKey struct{}

type environmentCache struct {
	mu      sync.Mutex
	entries map[string]map[string]string
}

// WithEnvironmentCache shares effective Go settings across one request tree.
func WithEnvironmentCache(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(environmentCacheContextKey{}).(*environmentCache); ok {
		return ctx
	}
	return context.WithValue(ctx, environmentCacheContextKey{}, &environmentCache{entries: make(map[string]map[string]string)})
}

// WithBaseDir selects the directory holding Go subprocess state for one request tree.
func WithBaseDir(ctx context.Context, baseDir string) context.Context {
	return context.WithValue(ctx, baseDirContextKey{}, strings.TrimSpace(baseDir))
}

// WithModuleCache selects an explicit module cache independently of other Go state.
func WithModuleCache(ctx context.Context, moduleCache string) context.Context {
	return context.WithValue(ctx, moduleCacheContextKey{}, strings.TrimSpace(moduleCache))
}

// HasModuleCache reports whether the request selected an explicit module cache.
func HasModuleCache(ctx context.Context) bool {
	moduleCache, _ := ctx.Value(moduleCacheContextKey{}).(string)
	return strings.TrimSpace(moduleCache) != ""
}

// HasBaseDir reports whether a nonblank isolated Go state base was selected.
func HasBaseDir(ctx context.Context) bool {
	baseDir, _ := ctx.Value(baseDirContextKey{}).(string)
	return strings.TrimSpace(baseDir) != ""
}

// CanUseProcessRuntimeDirectories reports whether process Go paths are complete and may bypass go env.
func CanUseProcessRuntimeDirectories(ctx context.Context) bool {
	if ctx == nil || HasBaseDir(ctx) {
		return false
	}
	seed, _ := ctx.Value(moduleCacheSeedKey{}).(string)
	if strings.TrimSpace(seed) != "" {
		return false
	}
	moduleCache, _ := ctx.Value(moduleCacheContextKey{}).(string)
	if strings.TrimSpace(moduleCache) != "" {
		return false
	}
	required := map[string]bool{
		"GOCACHE":                 false,
		moduleCacheEnvironmentKey: false,
		"GOTMPDIR":                false,
		"GOPATH":                  false,
	}
	values := make(map[string]string, len(required))
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, isRequired := required[key]; !isRequired {
			continue
		}
		if required[key] {
			return false
		}
		required[key] = true
		values[key] = value
	}
	for key, found := range required {
		value := values[key]
		if !found || value == "" || !filepath.IsAbs(value) {
			return false
		}
	}
	gopath := filepath.SplitList(values["GOPATH"])
	if len(gopath) == 0 {
		return false
	}
	for _, path := range gopath {
		if path == "" || !filepath.IsAbs(path) {
			return false
		}
	}
	return true
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
	defer telemetry.StartPhase(ctx, "go.environment")()
	baseDir, baseDirErr := resolveBaseDir(ctx)
	if baseDirErr != nil {
		return nil, baseDirErr
	}
	seed, _ := ctx.Value(moduleCacheSeedKey{}).(string)
	explicitModuleCache, _ := ctx.Value(moduleCacheContextKey{}).(string)
	if explicitModuleCache != "" && seed != "" {
		return nil, fmt.Errorf("explicit Go module cache conflicts with offline module cache seed")
	}
	if seed != "" && baseDir == "" {
		return nil, fmt.Errorf("offline Go module cache seed requires an explicit Go isolation base directory")
	}
	if baseDir == "" {
		return inheritedEnvironment(ctx, workDir, explicitModuleCache)
	}

	cacheDir := filepath.Join(baseDir, "build")
	moduleCacheDir, err := selectedModuleCacheDir(baseDir, explicitModuleCache)
	if err != nil {
		return nil, err
	}
	tempDir := filepath.Join(baseDir, "tmp")
	binDir := filepath.Join(baseDir, "bin")
	goplsCacheDir := filepath.Join(baseDir, "gopls")
	for _, dir := range []string{baseDir, cacheDir, moduleCacheDir, tempDir, binDir, goplsCacheDir} {
		if stateErr := verifyStateDirectory(dir); stateErr != nil {
			return nil, stateErr
		}
	}
	if seedErr := seedModuleCache(ctx, baseDir, moduleCacheDir); seedErr != nil {
		return nil, seedErr
	}
	goEnvIdentity, err := goEnvConfigIdentity([]string{"GOENV=" + filepath.Join(baseDir, "env")})
	if err != nil {
		return nil, err
	}

	managed := map[string]struct{}{
		"GOENV":                   {},
		"GOCACHE":                 {},
		moduleCacheEnvironmentKey: {},
		"GOTMPDIR":                {},
		"TMPDIR":                  {},
		"TEMP":                    {},
		"TMP":                     {},
		"GOBIN":                   {},
		"GOPATH":                  {},
		"GOPLSCACHE":              {},
		"GOFLAGS":                 {},
		"GOWORK":                  {},
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
		"SEMEDIT_GOENV_CONFIG_ID="+goEnvIdentity,
	), nil
}

func inheritedEnvironment(ctx context.Context, workDir, explicitModuleCache string) ([]string, error) {
	settings, err := effectiveGoSettings(ctx, workDir)
	if err != nil {
		return nil, err
	}
	goFlags := settings["GOFLAGS"]
	if !moduleWritesAllowed(ctx) {
		goFlags = appendGoFlag(goFlags, "-mod=readonly")
	}
	env := setEnvironmentValue(os.Environ(), "GOFLAGS", goFlags)
	if explicitModuleCache != "" {
		moduleCache, resolveErr := resolveModuleCache(explicitModuleCache)
		if resolveErr != nil {
			return nil, resolveErr
		}
		env = setEnvironmentValue(env, moduleCacheEnvironmentKey, moduleCache)
	}
	for _, key := range []string{"GOCACHE", moduleCacheEnvironmentKey, "GOPATH", "GOTMPDIR"} {
		if value, exists := environmentValue(env, key); exists && value != "" {
			continue
		}
		if value := settings[key]; value != "" {
			env = setEnvironmentValue(env, key, value)
		}
	}
	goEnvIdentity, err := goEnvConfigIdentity(env)
	if err != nil {
		return nil, err
	}
	return setEnvironmentValue(env, "SEMEDIT_GOENV_CONFIG_ID", goEnvIdentity), nil
}

func resolveModuleCache(moduleCache string) (string, error) {
	resolved, err := filepath.Abs(moduleCache)
	if err != nil {
		return "", fmt.Errorf("resolve Go module cache directory: %w", err)
	}
	if err := verifyStateDirectory(resolved); err != nil {
		return "", fmt.Errorf("prepare Go module cache: %w", err)
	}
	return resolved, nil
}

func selectedModuleCacheDir(baseDir, explicitModuleCache string) (string, error) {
	if explicitModuleCache == "" {
		return filepath.Join(baseDir, "mod"), nil
	}
	return resolveModuleCache(explicitModuleCache)
}

func effectiveGoSettings(ctx context.Context, workDir string) (map[string]string, error) {
	defer telemetry.StartPhase(ctx, "go.environment.settings")()
	cache, _ := ctx.Value(environmentCacheContextKey{}).(*environmentCache)
	key, err := effectiveSettingsCacheKey(workDir)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.mu.Lock()
		settings, ok := cache.entries[key]
		if ok {
			cache.mu.Unlock()
			return cloneSettings(settings), nil
		}
	}
	settings, err := resolveEffectiveGoSettings(ctx, workDir)
	if err != nil {
		if cache != nil {
			cache.mu.Unlock()
		}
		return nil, err
	}
	if cache != nil {
		cache.entries[key] = cloneSettings(settings)
		cache.mu.Unlock()
	}
	return settings, nil
}

func resolveEffectiveGoSettings(ctx context.Context, workDir string) (map[string]string, error) {
	requested := []string{"GOFLAGS", "GOCACHE", moduleCacheEnvironmentKey, "GOPATH", "GOTMPDIR"}
	command := exec.CommandContext(ctx, "go", "env", "-json", "GOFLAGS", "GOCACHE", moduleCacheEnvironmentKey, "GOPATH", "GOTMPDIR")
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

func effectiveSettingsCacheKey(workDir string) (string, error) {
	directory, err := existingGoEnvDirectory(workDir)
	if err != nil {
		return "", err
	}
	environment := os.Environ()
	slices.Sort(environment)
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s\n", directory)
	for _, value := range environment {
		fmt.Fprintf(&builder, "%s\n", value)
	}
	identity, err := goEnvConfigIdentity(environment)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&builder, "GOENV_ID=%s", identity)
	return builder.String(), nil
}

func goEnvConfigIdentity(environment []string) (string, error) {
	goEnv, ok := environmentValue(environment, "GOENV")
	if !ok || goEnv == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve effective Go configuration directory: %w", err)
		}
		goEnv = filepath.Join(configDir, "go", "env")
	}
	if goEnv == "off" {
		return "off", nil
	}
	content, err := os.ReadFile(goEnv) // #nosec G304 -- GOENV is a process-selected Go configuration file.
	state := "present"
	if os.IsNotExist(err) {
		state = "missing"
		content = nil
	} else if err != nil {
		return "", fmt.Errorf("read effective Go configuration %q: %w", goEnv, err)
	}
	digest := sha256.Sum256(content)
	return state + ":" + hex.EncodeToString(digest[:]), nil
}

func cloneSettings(settings map[string]string) map[string]string {
	return maps.Clone(settings)
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
	case "GOENV", "GOCACHE", moduleCacheEnvironmentKey, "GOTMPDIR", "TMPDIR", "TEMP", "TMP", "GOBIN", "GOPATH", "GOPLSCACHE", "GOFLAGS", "GOWORK":
		return true
	default:
		return false
	}
}
