// Temporary file coverage ensures retained language servers do not pollute workspace snapshots.

package gocache_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestGoSubprocessTemporaryFiles(t *testing.T) {
	if os.Getenv("SEMEDIT_TEMP_HELPER") == "1" {
		file, err := os.CreateTemp("", "semedit-child-")
		if err != nil {
			panic(err)
		}
		if err := file.Close(); err != nil {
			panic(err)
		}
		fmt.Print(file.Name())
		os.Exit(0)
	}
	workspace := t.TempDir()
	inherited := filepath.Join(workspace, ".tmp")
	if err := os.MkdirAll(inherited, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, inherited)
	}
	baseDir := filepath.Join(workspace, ".scratch", "go")
	environment, err := gocache.Environment(gocache.WithBaseDir(t.Context(), baseDir), workspace)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestGoSubprocessTemporaryFiles$")
	environment = append(environment, "SEMEDIT_TEMP_HELPER=1")
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("child temporary file: %v: %s", err, output)
	}
	path := string(output)
	defer func() {
		if removeErr := os.Remove(path); removeErr != nil {
			t.Error(removeErr)
		}
	}()
	expected := filepath.Join(baseDir, "tmp")
	actual, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	expected, err = filepath.EvalSymlinks(expected)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("temporary file %q, want directory %q", path, expected)
	}
	entries, err := os.ReadDir(inherited)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("inherited temporary directory polluted: %v", entries)
	}
}

func TestEnvironmentCacheReusesEffectiveGoSettingsWithinRequest(t *testing.T) {
	bin := t.TempDir()
	workDir := t.TempDir()
	callLog := filepath.Join(t.TempDir(), "go-env-calls")
	goCommand := filepath.Join(bin, "go")
	script := "#!/bin/sh\nprintf 'call\\n' >> \"$GO_ENV_CALLS\"\nprintf '{\\\"GOFLAGS\\\":\\\"\\\",\\\"GOCACHE\\\":\\\"cache\\\",\\\"GOMODCACHE\\\":\\\"modules\\\",\\\"GOPATH\\\":\\\"path\\\",\\\"GOTMPDIR\\\":\\\"tmp\\\"}\\n'\n"
	if err := os.WriteFile(goCommand, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("GO_ENV_CALLS", callLog)
	ctx := gocache.WithEnvironmentCache(t.Context())
	if _, err := gocache.Environment(ctx, workDir); err != nil {
		t.Fatal(err)
	}
	if _, err := gocache.Environment(gocache.WithEnvironmentCache(ctx), workDir); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(calls), "call\n"); got != 1 {
		t.Fatalf("effective Go settings subprocess calls = %d, want 1", got)
	}
}

func TestEnvironmentKeepsGoplsCacheBelowSelectedBase(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "selected", ".scratch", "go")
	environment, err := gocache.Environment(gocache.WithBaseDir(t.Context(), baseDir), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := "GOPLSCACHE=" + filepath.Join(baseDir, "gopls")
	foundGoplsCache := false
	for _, variable := range environment {
		if strings.HasPrefix(variable, "GOPLSCACHE=") {
			if variable != want {
				t.Fatalf("GOPLSCACHE = %q, want %q", variable, want)
			}
			foundGoplsCache = true
		}
	}
	if !foundGoplsCache {
		t.Fatal("Go subprocess environment omitted GOPLSCACHE")
	}
}

func TestEnvironmentUsesExplicitModuleCacheIndependentlyOfManagedBase(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "managed")
	moduleCache := filepath.Join(t.TempDir(), "persistent-modules")
	ctx := gocache.WithModuleCache(gocache.WithBaseDir(t.Context(), baseDir), moduleCache)
	environment, err := gocache.Environment(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	if values["GOMODCACHE"] != moduleCache {
		t.Errorf("GOMODCACHE = %q, want explicit module cache %q", values["GOMODCACHE"], moduleCache)
	}
	for key, want := range map[string]string{
		"GOCACHE":    filepath.Join(baseDir, "build"),
		"GOTMPDIR":   filepath.Join(baseDir, "tmp"),
		"GOPLSCACHE": filepath.Join(baseDir, "gopls"),
		"GOBIN":      filepath.Join(baseDir, "bin"),
	} {
		if values[key] != want {
			t.Errorf("%s = %q, want %q", key, values[key], want)
		}
	}
	if _, err := os.Stat(moduleCache); err != nil {
		t.Fatalf("explicit module cache was not prepared: %v", err)
	}
}

func TestEnvironmentUsesExplicitModuleCacheWithInheritedGoState(t *testing.T) {
	moduleCache := filepath.Join(t.TempDir(), "persistent-modules")
	bin := t.TempDir()
	goCommand := filepath.Join(bin, "go")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"GOFLAGS\":\"\",\"GOCACHE\":\"cache\",\"GOMODCACHE\":\"modules\",\"GOPATH\":\"path\",\"GOTMPDIR\":\"tmp\"}'\n"
	if err := os.WriteFile(goCommand, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("GOENV", "off")
	t.Setenv("GOMODCACHE", filepath.Join(t.TempDir(), "ambient-modules"))
	ctx := gocache.WithModuleCache(t.Context(), moduleCache)
	environment, err := gocache.Environment(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range environment {
		if value, ok := strings.CutPrefix(entry, "GOMODCACHE="); ok {
			if value != moduleCache {
				t.Fatalf("GOMODCACHE = %q, want explicit module cache %q", value, moduleCache)
			}
			return
		}
	}
	t.Fatal("Go subprocess environment omitted GOMODCACHE")
}

func TestEnvironmentRejectsModuleCacheSeedWithExplicitModuleCache(t *testing.T) {
	ctx := gocache.WithModuleCacheSeed(
		gocache.WithModuleCache(gocache.WithBaseDir(t.Context(), t.TempDir()), t.TempDir()),
		t.TempDir(),
	)
	if _, err := gocache.Environment(ctx, t.TempDir()); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("Environment error = %v, want explicit-cache/seed conflict", err)
	}
}

func TestEnvironmentSeedsOfflineModuleCacheIntoManagedState(t *testing.T) {
	seed := filepath.Join(t.TempDir(), "seed")
	source := filepath.Join(seed, "cache", "download", "fixture.test", "dependency", "@v", "v1.0.0.mod")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	sourceData := []byte("module fixture.test/dependency\n\ngo 1.27.1\n")
	if err := os.WriteFile(source, sourceData, 0o400); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(source, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPROXY", "off")
	base := filepath.Join(t.TempDir(), "managed")
	ctx := gocache.WithModuleCacheSeed(gocache.WithBaseDir(t.Context(), base), seed)
	environment, err := gocache.Environment(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("prepare offline module cache: %v", err)
	}
	var moduleCache string
	for _, entry := range environment {
		if after, ok := strings.CutPrefix(entry, "GOMODCACHE="); ok {
			moduleCache = after
		}
	}
	if moduleCache != filepath.Join(base, "mod") {
		t.Fatalf("GOMODCACHE = %q, want managed cache %q", moduleCache, filepath.Join(base, "mod"))
	}
	copied := filepath.Join(moduleCache, "cache", "download", "fixture.test", "dependency", "@v", "v1.0.0.mod")
	got, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("read copied module metadata: %v", err)
	}
	if !bytes.Equal(got, sourceData) {
		t.Fatalf("copied module metadata = %q, want %q", got, sourceData)
	}
	sourceInfo, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	destinationInfo, err := os.Stat(copied)
	if err != nil {
		t.Fatal(err)
	}
	if !sourceInfo.ModTime().Equal(oldTime) || !destinationInfo.ModTime().After(oldTime) {
		t.Fatalf("source/destination timestamps = %v / %v, want preserved source and fresh destination", sourceInfo.ModTime(), destinationInfo.ModTime())
	}
}

func TestEnvironmentControlsModuleWriteFlags(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod -v")
	readValue := func(environment []string) string {
		for _, entry := range environment {
			if value, ok := strings.CutPrefix(entry, "GOFLAGS="); ok {
				return value
			}
		}
		return ""
	}
	ordinary, err := gocache.Environment(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := readValue(ordinary); got != "-mod=mod -v -mod=readonly" {
		t.Fatalf("ordinary GOFLAGS = %q, want inherited flags followed by -mod=readonly", got)
	}
	writable, err := gocache.Environment(gocache.WithWritableModuleChanges(t.Context()), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := readValue(writable); got != "-mod=mod -v" {
		t.Fatalf("explicit module-change GOFLAGS = %q, want inherited flags", got)
	}
	managed, err := gocache.Environment(gocache.WithBaseDir(t.Context(), filepath.Join(t.TempDir(), "managed")), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := readValue(managed); got != "-mod=readonly" {
		t.Fatalf("isolated GOFLAGS = %q, want managed read-only flags", got)
	}
}

func TestEnvironmentUsesAmbientGoEnvironmentWithoutCreatingWorkspaceState(t *testing.T) {
	workspace := t.TempDir()
	configDir := t.TempDir()
	goenvFile := filepath.Join(configDir, "go.env")
	settings := map[string]string{
		"GOCACHE":    filepath.Join(configDir, "build cache"),
		"GOMODCACHE": filepath.Join(configDir, "module cache"),
		"GOPATH":     filepath.Join(configDir, "go path"),
		"GOTMPDIR":   filepath.Join(configDir, "go temp"),
		"GOFLAGS":    "-tags=from-goenv",
	}
	var goenv strings.Builder
	for key, value := range settings {
		fmt.Fprintf(&goenv, "%s=%s\n", key, value)
	}
	if err := os.WriteFile(goenvFile, []byte(goenv.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", goenvFile)
	for _, key := range []string{"GOFLAGS", "GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR"} {
		t.Setenv(key, "")
	}
	inherited := map[string]string{
		"GOBIN":      filepath.Join(configDir, "user-bin"),
		"GOPLSCACHE": filepath.Join(configDir, "gopls-cache"),
		"GOWORK":     "off",
		"TMPDIR":     filepath.Join(configDir, "temp-dir"),
		"TEMP":       filepath.Join(configDir, "temp-env"),
		"TMP":        filepath.Join(configDir, "temp-tmp"),
	}
	for key, value := range inherited {
		t.Setenv(key, value)
	}
	environment, err := gocache.Environment(t.Context(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range settings {
		want := value
		if key == "GOFLAGS" {
			want += " -mod=readonly"
		}
		if got := readEnvValue(environment, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for key, value := range inherited {
		if got := readEnvValue(environment, key); got != value {
			t.Errorf("inherited %s = %q, want %q", key, got, value)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace, ".scratch")); !os.IsNotExist(err) {
		t.Errorf("ambient environment created workspace scratch state, stat error = %v", err)
	}
}

func TestEnvironmentRejectsModuleCacheSeedWithoutExplicitBase(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	seed := t.TempDir()
	workspace := t.TempDir()
	ctx := gocache.WithModuleCacheSeed(t.Context(), seed)
	if _, err := gocache.Environment(ctx, workspace); err == nil || !strings.Contains(err.Error(), "explicit Go isolation base") {
		t.Fatalf("Environment() error = %v, want explicit-base requirement", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".scratch")); !os.IsNotExist(err) {
		t.Fatalf("rejected seed created workspace scratch state, stat error = %v", err)
	}
}

func TestEnvironmentRejectsOfflineModuleCacheSeedConflicts(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	seed := filepath.Join(t.TempDir(), "seed")
	source := filepath.Join(seed, "cache", "download", "fixture.test", "dependency", "@v", "v1.0.0.mod")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("module fixture.test/dependency\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(t.TempDir(), "managed")
	destination := filepath.Join(base, "mod", "cache", "download", "fixture.test", "dependency", "@v", "v1.0.0.mod")
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("preexisting cache data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := gocache.WithModuleCacheSeed(gocache.WithBaseDir(t.Context(), base), seed)
	if _, err := gocache.Environment(ctx, t.TempDir()); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("Environment() error = %v, want cache conflict", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "preexisting cache data\n" {
		t.Fatalf("existing cache contents changed to %q", got)
	}
}

func TestEnvironmentRejectsOfflineModuleCacheSeedSymlinks(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	seed := filepath.Join(t.TempDir(), "seed")
	if err := os.MkdirAll(seed, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(seed, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	ctx := gocache.WithModuleCacheSeed(gocache.WithBaseDir(t.Context(), filepath.Join(t.TempDir(), "managed")), seed)
	if _, err := gocache.Environment(ctx, t.TempDir()); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Environment() error = %v, want seed symlink refusal", err)
	}
}

func TestEnvironmentRequiresOfflineModeForModuleCacheSeed(t *testing.T) {
	t.Setenv("GOPROXY", "https://proxy.golang.org")
	ctx := gocache.WithModuleCacheSeed(gocache.WithBaseDir(t.Context(), filepath.Join(t.TempDir(), "managed")), t.TempDir())
	if _, err := gocache.Environment(ctx, t.TempDir()); err == nil || !strings.Contains(err.Error(), "GOPROXY=off") {
		t.Fatalf("Environment() error = %v, want explicit offline-mode requirement", err)
	}
}

func TestEnvironmentSeedsOfflineModuleCacheConcurrently(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	seed := filepath.Join(t.TempDir(), "seed")
	source := filepath.Join(seed, "cache", "download", "fixture.test", "dependency", "@v", "v1.0.0.mod")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	contents := []byte("module fixture.test/dependency\n")
	if err := os.WriteFile(source, contents, 0o400); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(t.TempDir(), "managed")
	ctx := gocache.WithModuleCacheSeed(gocache.WithBaseDir(t.Context(), base), seed)
	var group sync.WaitGroup
	errors := make(chan error, 8)
	for range cap(errors) {
		group.Go(func() {
			_, err := gocache.Environment(ctx, t.TempDir())
			errors <- err
		})
	}
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent Environment() error: %v", err)
		}
	}
	copied := filepath.Join(base, "mod", "cache", "download", "fixture.test", "dependency", "@v", "v1.0.0.mod")
	got, err := os.ReadFile(copied)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, contents) {
		t.Fatalf("concurrent seed contents = %q, want %q", got, contents)
	}
}

func readEnvValue(environment []string, key string) string {
	for _, entry := range environment {
		if value, ok := strings.CutPrefix(entry, key+"="); ok {
			return value
		}
	}
	return ""
}

func TestEnvironmentIdentityTracksGoEnvChangesWithinRequestCache(t *testing.T) {
	workspace := t.TempDir()
	goEnv := filepath.Join(t.TempDir(), "go.env")
	t.Setenv("GOENV", goEnv)
	ctx := gocache.WithEnvironmentCache(t.Context())
	readIdentity := func() string {
		t.Helper()
		environment, err := gocache.Environment(ctx, workspace)
		if err != nil {
			t.Fatal(err)
		}
		return readEnvValue(environment, "SEMEDIT_GOENV_CONFIG_ID")
	}
	if err := os.WriteFile(goEnv, []byte("GOOS=linux\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	linuxIdentity := readIdentity()
	if linuxIdentity == "" {
		t.Fatal("first environment has no GOENV identity")
	}
	if err := os.WriteFile(goEnv, []byte("GOOS=windows\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	windowsIdentity := readIdentity()
	if windowsIdentity == "" || windowsIdentity == linuxIdentity {
		t.Fatalf("changed GOENV identity = %q, previous identity %q", windowsIdentity, linuxIdentity)
	}
}
