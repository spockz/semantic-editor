// ram_storage_test.go protects explicit RAM fixture storage from silently falling back to ordinary disk.

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestExplicitFixtureRootRejectsOrdinaryDiskEvenWhenItMatchesScratch(t *testing.T) {
	root := t.TempDir()
	runner := NewRunner(root, WithFixtureRoot(root), WithModuleCache(t.TempDir()))
	err := runner.validateFixtureRoot()
	if err == nil || !strings.Contains(err.Error(), "mounted tmpfs") {
		t.Fatalf("explicit ordinary fixture root error = %v, want tmpfs refusal", err)
	}
}

func TestFixtureGoEnvironmentUsesOfflinePolicyForContextModuleCache(t *testing.T) {
	workDir := t.TempDir()
	moduleCache := t.TempDir()
	env, err := fixtureGoEnvironment(gocache.WithModuleCache(context.Background(), moduleCache), workDir)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"GOPROXY": "off", "GOSUMDB": "off", "GOTOOLCHAIN": "local", "GOMODCACHE": moduleCache} {
		if got := environmentValue(env, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestRAMStorageRequiresPersistentModuleCache(t *testing.T) {
	err := validateDurableBenchmarkStorage(PlanOptions{RAMFixtureRoot: t.TempDir(), OutDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "--module-cache") {
		t.Fatalf("missing module cache error = %v", err)
	}
}

func TestPersistentModuleCacheMustExistAndBeDirectory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{"missing", filepath.Join(root, "missing"), "no such file"},
		{"file", file, "not a directory"},
		{"file ancestor", filepath.Join(file, "cache"), "not a directory"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateDurableBenchmarkStorage(PlanOptions{ModuleCache: test.path})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("module cache error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRAMStorageRejectsOutputThroughFileAncestor(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := validateDurableBenchmarkStorage(PlanOptions{
		RAMFixtureRoot: root,
		ModuleCache:    t.TempDir(),
		OutJSON:        filepath.Join(file, "report.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("output error = %v, want non-directory refusal", err)
	}
}

func TestExplicitModuleCacheRejectsTmpfs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux tmpfs fixture")
	}
	info, err := os.Stat("/dev/shm")
	if err != nil || !info.IsDir() {
		t.Skip("/dev/shm unavailable")
	}
	tmpfs, err := isTmpfsMount("/dev/shm")
	if err != nil || !tmpfs {
		t.Skip("/dev/shm is not tmpfs")
	}
	err = validateDurableBenchmarkStorage(PlanOptions{ModuleCache: "/dev/shm"})
	if err == nil || !strings.Contains(err.Error(), "outside tmpfs") {
		t.Fatalf("module cache error = %v, want tmpfs refusal", err)
	}
	err = validateDurableBenchmarkStorage(PlanOptions{
		RAMFixtureRoot: t.TempDir(),
		ModuleCache:    t.TempDir(),
		OutJSON:        filepath.Join("/dev/shm", "future", "report.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "outside tmpfs") {
		t.Fatalf("future JSON report error = %v, want tmpfs refusal", err)
	}
}
