// shared_module_cache_test.go proves cached module archives cannot expand into the persistent cache during benchmark preflight.

package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestSharedModuleCacheRequiresExtractedSources(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "fixture")
	cacheDir := filepath.Join(root, "modules")
	proxyDir := filepath.Join(root, "proxy", "example.com", "localdep", "@v")
	hiddenDir := filepath.Join(root, "hidden")
	for _, dir := range []string{workDir, cacheDir, proxyDir, hiddenDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(workDir, "go.mod"), "module fixture\n\ngo 1.23\n\nrequire example.com/localdep v1.0.0\n")
	writeTestFile(t, filepath.Join(workDir, "main.go"), "package fixture\nimport _ \"example.com/localdep\"\n")
	writeTestFile(t, filepath.Join(hiddenDir, "acceptance_test.go"), "package fixture\nimport _ \"example.com/localdep\"\n")
	writeLocalModuleArchive(t, proxyDir)
	proxyRoot := filepath.Dir(filepath.Dir(filepath.Dir(proxyDir)))
	proxyPath := filepath.ToSlash(proxyRoot)
	if filepath.VolumeName(proxyRoot) != "" {
		proxyPath = "/" + proxyPath
	}
	prime := func() {
		command := exec.Command("go", "mod", "download", "all")
		command.Dir = workDir
		command.Env = replaceEnvironment(os.Environ(), map[string]string{
			"GOMODCACHE":  cacheDir,
			"GOCACHE":     filepath.Join(root, "build"),
			"GOPROXY":     "file://" + proxyPath,
			"GOSUMDB":     "off",
			"GOTOOLCHAIN": "local",
			"GOWORK":      "off",
		})
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("prepare local module archive: %v: %s", err, output)
		}
	}
	prime()
	moduleDir := filepath.Join(cacheDir, "example.com", "localdep@v1.0.0")
	t.Cleanup(func() {
		if err := os.Chmod(moduleDir, 0o750); err != nil && !os.IsNotExist(err) {
			t.Errorf("restore module directory permissions: %v", err)
		}
	})
	if err := os.Chmod(moduleDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(moduleDir); err != nil {
		t.Fatal(err)
	}
	ctx := gocache.WithModuleCache(context.Background(), cacheDir)
	preflightErr := prepareFixtureGoDependencies(ctx, workDir)
	if preflightErr == nil || !strings.Contains(preflightErr.Error(), "not fully extracted") {
		t.Fatalf("zip-only cache error = %v, want refusal before extraction", preflightErr)
	}
	assertSharedPreflightBuildCacheClean(t, workDir)
	if _, statErr := os.Stat(moduleDir); !os.IsNotExist(statErr) {
		t.Fatalf("zip-only module was extracted: %v", statErr)
	}
	task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{Test: TestConfig{PassTests: true, HiddenTestDir: hiddenDir}}}}
	preflightErr = prepareSharedGoDependencies(ctx, task, workDir)
	if preflightErr != nil {
		t.Fatalf("populate private zip-only cache: %v", preflightErr)
	}
	if info, statErr := os.Stat(moduleDir); statErr != nil || !info.IsDir() {
		t.Fatalf("setup did not extract cached module: %v", statErr)
	}
	prime()
	if err := prepareFixtureGoDependencies(ctx, workDir); err != nil {
		t.Fatalf("pre-extracted visible module: %v", err)
	}
	assertWarmupAfterSharedPreflight(ctx, t, workDir, root)
	if err := prepareSharedGoDependencies(ctx, task, workDir); err != nil {
		t.Fatalf("pre-extracted private module: %v", err)
	}
}

func assertWarmupAfterSharedPreflight(ctx context.Context, t *testing.T, workDir, root string) {
	t.Helper()
	assertSharedPreflightBuildCacheClean(t, workDir)
	verify := filepath.Join(root, "fake-semedit-verify")
	writeTestFile(t, verify, "#!/bin/sh\nset -eu\ntest \"$GOCACHE\" = \"$SEMEDIT_EXPECTED_BUILD_CACHE\"\ntest \"$GOPROXY\" = off\n")
	if err := os.Chmod(verify, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEMEDIT_EXPECTED_BUILD_CACHE", filepath.Join(workDir, ".scratch", "go", "build"))
	if _, err := runSemeditVerifyWarmup(ctx, workDir, "main.go", verify); err != nil {
		t.Fatalf("prewarm after real dependency setup: %v", err)
	}
}

func TestSharedDependencyPreflightCleansBuildCacheOnCancellation(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "fixture")
	binDir := filepath.Join(root, "bin")
	for _, dir := range []string{workDir, binDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(workDir, "go.mod"), "module fixture\n\ngo 1.23\n")
	started := filepath.Join(root, "started")
	fakeGo := filepath.Join(binDir, "go")
	writeTestFile(t, fakeGo, "#!/bin/sh\nset -eu\n: > \"$SEMEDIT_SHARED_PREFLIGHT_STARTED\"\nexec sleep 30\n")
	if err := os.Chmod(fakeGo, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_SHARED_PREFLIGHT_STARTED", started)
	ctx, cancel := context.WithCancel(gocache.WithModuleCache(context.Background(), filepath.Join(root, "modules")))
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- prepareFixtureGoDependencies(ctx, workDir) }()
	waitForOraclePreflightFile(t, started)
	cancel()
	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("canceled preflight error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preflight did not stop after cancellation")
	}
	assertSharedPreflightBuildCacheClean(t, workDir)
}

func assertSharedPreflightBuildCacheClean(t *testing.T, workDir string) {
	t.Helper()
	buildDir := filepath.Join(workDir, ".scratch", "go", "build")
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("job build cache has %d entries before warmup", len(entries))
	}
	temporaryDir := filepath.Join(workDir, ".scratch", "go", "tmp")
	entries, err = os.ReadDir(temporaryDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "dependency-preflight-build-") {
			t.Fatalf("dependency preflight cache remains: %s", entry.Name())
		}
	}
}

func writeLocalModuleArchive(t *testing.T, proxyDir string) {
	t.Helper()
	writeTestFile(t, filepath.Join(proxyDir, "v1.0.0.mod"), "module example.com/localdep\n\ngo 1.23\n")
	writeTestFile(t, filepath.Join(proxyDir, "v1.0.0.info"), `{"Version":"v1.0.0","Time":"2024-01-01T00:00:00Z"}`)
	archive, createErr := os.Create(filepath.Join(proxyDir, "v1.0.0.zip"))
	if createErr != nil {
		t.Fatal(createErr)
	}
	zipWriter := zip.NewWriter(archive)
	for name, data := range map[string]string{
		"go.mod": "module example.com/localdep\n\ngo 1.23\n",
		"dep.go": "package localdep\n",
	} {
		entry, createErr := zipWriter.Create("example.com/localdep@v1.0.0/" + name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(data)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedModuleCacheAcceptsLocalReplacementAndStandardLibrary(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "fixture")
	depDir := filepath.Join(root, "dep")
	for _, dir := range []string{workDir, depDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(workDir, "go.mod"), "module fixture\n\ngo 1.23\n\nrequire example.com/dep v1.0.0\nreplace example.com/dep => ../dep\n")
	writeTestFile(t, filepath.Join(workDir, "main.go"), "package fixture\nimport (\"fmt\"; _ \"example.com/dep\")\nvar _ = fmt.Sprint\n")
	writeTestFile(t, filepath.Join(depDir, "go.mod"), "module example.com/dep\n\ngo 1.23\n")
	writeTestFile(t, filepath.Join(depDir, "dep.go"), "package dep\n")
	ctx := gocache.WithModuleCache(context.Background(), filepath.Join(root, "modules"))
	if err := prepareFixtureGoDependencies(ctx, workDir); err != nil {
		t.Fatalf("local replacement and standard library: %v", err)
	}
}

func TestSharedModuleCacheWithoutContentChecksum(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "fixture")
	cacheDir := filepath.Join(root, "modules")
	proxyDir := filepath.Join(root, "proxy", "example.com", "localdep", "@v")
	for _, dir := range []string{workDir, proxyDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(workDir, "go.mod"), "module fixture\n\ngo 1.23\n\nrequire example.com/localdep v1.0.0\n")
	writeLocalModuleArchive(t, proxyDir)
	env := replaceEnvironment(os.Environ(), map[string]string{
		"GOMODCACHE": cacheDir, "GOPROXY": "file://" + filepath.ToSlash(filepath.Join(root, "proxy")),
		"GOSUMDB": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOENV": "off",
	})
	command := exec.Command("go", "mod", "download", "all")
	command.Dir, command.Env = workDir, env
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("prime module: %v: %s", err, output)
	}
	t.Cleanup(func() {
		if err := os.Chmod(filepath.Join(cacheDir, "example.com", "localdep@v1.0.0"), 0o750); err != nil {
			t.Error(err)
		}
	})
	sumPath := filepath.Join(workDir, "go.sum")
	sums, err := os.ReadFile(sumPath)
	if err != nil {
		t.Fatal(err)
	}
	var retained []string
	for line := range strings.SplitSeq(string(sums), "\n") {
		if strings.Contains(line, "/go.mod ") {
			retained = append(retained, line)
		}
	}
	writeTestFile(t, sumPath, strings.Join(retained, "\n")+"\n")
	ctx := gocache.WithModuleCache(context.Background(), cacheDir)
	env = replaceEnvironment(env, map[string]string{"GOPROXY": "off"})
	command = exec.Command("go", "list", "-mod=readonly", "-m", "-json", "example.com/localdep")
	command.Dir, command.Env = workDir, env
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct{ Dir string }
	if err := json.Unmarshal(output, &inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.Dir != "" {
		t.Fatalf("fixture did not reproduce omitted Dir: %q", inventory.Dir)
	}
	t.Run("complete without checksum", func(t *testing.T) {
		if err := preflightSharedModuleCache(ctx, workDir, env); err != nil {
			t.Fatal(err)
		}
	})
	archive := filepath.Join(cacheDir, "cache", "download", "example.com", "localdep", "@v", "v1.0.0")
	t.Run("partial extraction", func(t *testing.T) {
		writeTestFile(t, archive+".partial", "")
		defer func() {
			if err := os.Remove(archive + ".partial"); err != nil {
				t.Error(err)
			}
		}()
		if err := preflightSharedModuleCache(ctx, workDir, env); err == nil {
			t.Fatal("accepted incomplete extraction")
		}
	})
	t.Run("missing hash", func(t *testing.T) {
		if err := os.Remove(archive + ".ziphash"); err != nil {
			t.Fatal(err)
		}
		if err := preflightSharedModuleCache(ctx, workDir, env); err == nil {
			t.Fatal("accepted missing archive hash")
		}
		if _, err := os.Stat(archive + ".ziphash"); !os.IsNotExist(err) {
			t.Fatalf("preflight regenerated hash: %v", err)
		}
	})
}
