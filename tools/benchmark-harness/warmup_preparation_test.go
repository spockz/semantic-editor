// warmup_preparation_test.go verifies fixture warmup discovery and shared dependency setup.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/gocache"
)

func TestPrepareSharedGoDependenciesPopulatesColdCacheConcurrently(t *testing.T) {
	fixture := newSharedWarmupFixture(t)
	fixture.prepareConcurrently(t)
	t.Run("acceptance dependencies remain private", func(t *testing.T) {
		fixture.assertPrivateSetup(t)
	})
	t.Run("cached dependencies support offline fixture tests", func(t *testing.T) {
		fixture.assertOffline(t)
	})
}

func TestFixtureGoWarmupFileDiscoversBuildSelectedSourceAndRejectsEscape(t *testing.T) {
	t.Run("no module skips fallback discovery", func(t *testing.T) {
		workDir := t.TempDir()
		file, err := fixtureGoWarmupFile(context.Background(), &Task{}, workDir)
		if err != nil {
			t.Fatal(err)
		}
		if file != "" {
			t.Fatalf("warmup target = %q, want no target", file)
		}
	})

	t.Run("fallback selects active Go source", func(t *testing.T) {
		workDir := t.TempDir()
		writeTestFile(t, filepath.Join(workDir, "go.mod"), "module example.com/warmup\n\ngo 1.23\n")
		writeTestFile(t, filepath.Join(workDir, "main.go"), "package warmup\nfunc Value() int { return 1 }\n")
		writeTestFile(t, filepath.Join(workDir, "ignored.go"), "//go:build never\n\npackage ignored\nthis is not Go\n")
		writeTestFile(t, filepath.Join(workDir, "README.txt"), "not a Go source file\n")
		if err := os.MkdirAll(filepath.Join(workDir, "vendor", "example.com", "ignored"), 0o750); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(workDir, "vendor", "example.com", "ignored", "ignored.go"), "package ignored\nthis is not Go\n")
		file, err := fixtureGoWarmupFile(context.Background(), &Task{}, workDir)
		if err != nil {
			t.Fatal(err)
		}
		if file != "main.go" {
			t.Fatalf("warmup target = %q, want main.go", file)
		}
	})

	t.Run("AST path cannot escape through symlink", func(t *testing.T) {
		root := t.TempDir()
		workDir := filepath.Join(root, "fixture")
		if err := os.Mkdir(workDir, 0o750); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(root, "outside.go")
		writeTestFile(t, outside, "package outside\n")
		if err := os.Symlink(outside, filepath.Join(workDir, "escape.go")); err != nil {
			t.Fatal(err)
		}
		task := &Task{Metadata: TaskMetadata{Oracle: OracleConfig{AST: ASTConfig{File: "escape.go"}}}}
		if _, err := fixtureGoWarmupFile(context.Background(), task, workDir); err == nil || !strings.Contains(err.Error(), "escapes fixture") {
			t.Fatalf("warmup target error = %v, want fixture escape rejection", err)
		}
	})
}

func TestDownloadSharedGoModulesPreservesProxySettingsAndUsesLocalToolchain(t *testing.T) {
	tests := []struct {
		name  string
		proxy string
		sumdb string
	}{
		{name: "explicit offline setting", proxy: "off", sumdb: "off"},
		{name: "configured defaults", proxy: "https://proxy.golang.org,direct", sumdb: "sum.golang.org"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			workDir := filepath.Join(root, "fixture")
			binDir := filepath.Join(root, "bin")
			if err := os.MkdirAll(binDir, 0o750); err != nil {
				t.Fatal(err)
			}
			program := fmt.Sprintf("#!/bin/sh\nset -eu\ntest \"$*\" = 'mod download all'\ntest \"$GOPROXY\" = %q\ntest \"$GOSUMDB\" = %q\ntest \"$GOTOOLCHAIN\" = local\ntest -n \"$GOMODCACHE\"\n", test.proxy, test.sumdb)
			writeTestFile(t, filepath.Join(binDir, "go"), program)
			if err := os.Chmod(filepath.Join(binDir, "go"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(workDir, 0o750); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("GOPROXY", test.proxy)
			t.Setenv("GOSUMDB", test.sumdb)
			if err := downloadSharedGoModules(context.Background(), workDir); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type sharedWarmupFixture struct {
	root      string
	proxy     string
	cacheDir  string
	ctx       context.Context
	workDirs  []string
	manifests []map[string][]byte
	tasks     []*Task
}

func newSharedWarmupFixture(t *testing.T) *sharedWarmupFixture {
	root := t.TempDir()
	proxy := createFixtureDependencyProxy(t, root)
	cacheDir := filepath.Join(root, "shared-modules")
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	fixture := &sharedWarmupFixture{
		root:      root,
		proxy:     proxy,
		cacheDir:  cacheDir,
		ctx:       gocache.WithModuleCache(context.Background(), cacheDir),
		workDirs:  make([]string, 2),
		manifests: make([]map[string][]byte, 2),
		tasks:     make([]*Task, 2),
	}
	checksums := seedSharedWarmupChecksums(t, root, proxy)
	fixture.createJobs(t, checksums)
	return fixture
}

func seedSharedWarmupChecksums(t *testing.T, root, proxy string) []byte {
	seedDir := filepath.Join(root, "seed")
	seedCache := filepath.Join(root, "seed-modules")
	if err := os.MkdirAll(seedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	module := "module example.com/seed\n\ngo 1.23\n\nrequire example.com/dependency v1.0.0\n"
	writeTestFile(t, filepath.Join(seedDir, "go.mod"), module)
	seedEnv := replaceEnvironment(os.Environ(), map[string]string{
		"GOMODCACHE":  seedCache,
		"GOPROXY":     "file://" + filepath.ToSlash(proxy),
		"GOSUMDB":     "off",
		"GOTOOLCHAIN": "local",
		"GOWORK":      "off",
		"GOENV":       "off",
	})
	seed := exec.Command("go", "mod", "download", "all")
	seed.Dir, seed.Env = seedDir, seedEnv
	if output, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("seed module checksums: %v: %s", err, output)
	}
	seedModuleDir := filepath.Join(seedCache, "example.com", "dependency@v1.0.0")
	t.Cleanup(func() {
		if err := os.Chmod(seedModuleDir, 0o750); err != nil {
			t.Error(err)
		}
	})
	sums, err := os.ReadFile(filepath.Join(seedDir, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	return sums
}

func (f *sharedWarmupFixture) createJobs(t *testing.T, checksums []byte) {
	module := "module example.com/seed\n\ngo 1.23\n\nrequire example.com/dependency v1.0.0\n"
	for i := range f.workDirs {
		workDir := filepath.Join(f.root, fmt.Sprintf("fixture-%d", i))
		hiddenDir := filepath.Join(f.root, fmt.Sprintf("hidden-%d", i))
		if err := os.MkdirAll(workDir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(hiddenDir, 0o750); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(workDir, "go.mod"), strings.Replace(module, "example.com/seed", fmt.Sprintf("example.com/fixture%d", i), 1))
		writeTestFile(t, filepath.Join(workDir, "go.sum"), string(checksums))
		writeTestFile(t, filepath.Join(workDir, "main.go"), "package fixture\nfunc Value() int { return 42 }\n")
		writeTestFile(t, filepath.Join(hiddenDir, "acceptance_test.go"), "package fixture\nimport (\"testing\"; dependency \"example.com/dependency\")\nfunc TestAcceptanceDependency(t *testing.T) { if dependency.Value() != 42 { t.Fatal(\"wrong value\") } }\n")
		f.workDirs[i] = workDir
		f.manifests[i] = map[string][]byte{}
		for _, name := range []string{"go.mod", "go.sum"} {
			data, err := os.ReadFile(filepath.Join(workDir, name))
			if err != nil {
				t.Fatal(err)
			}
			f.manifests[i][name] = data
		}
		f.tasks[i] = &Task{Metadata: TaskMetadata{Oracle: OracleConfig{Test: TestConfig{PassTests: true, HiddenTestDir: hiddenDir}}}}
	}
}

func (f *sharedWarmupFixture) prepareConcurrently(t *testing.T) {
	errs := make(chan error, len(f.workDirs))
	for i, workDir := range f.workDirs {
		go func(task *Task, workDir string) {
			errs <- prepareSharedGoDependencies(f.ctx, task, workDir)
		}(f.tasks[i], workDir)
	}
	for range f.workDirs {
		if err := <-errs; err != nil {
			t.Fatalf("prepare concurrent private acceptance dependencies: %v", err)
		}
	}
}

func (f *sharedWarmupFixture) assertPrivateSetup(t *testing.T) {
	moduleDir := filepath.Join(f.cacheDir, "example.com", "dependency@v1.0.0")
	if _, err := os.Stat(filepath.Join(moduleDir, "dependency.go")); err != nil {
		t.Fatalf("shared module cache lacks extracted acceptance dependency: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(moduleDir, 0o750); err != nil {
			t.Error(err)
		}
	})
	for i, workDir := range f.workDirs {
		for name, want := range f.manifests[i] {
			got, err := os.ReadFile(filepath.Join(workDir, name))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("fixture %d %s changed during private setup", i, name)
			}
		}
		assertNoPrivateFixtureWrites(t, i, workDir)
	}
}

func assertNoPrivateFixtureWrites(t *testing.T, index int, workDir string) {
	if _, err := os.Stat(filepath.Join(workDir, "acceptance_test.go")); !os.IsNotExist(err) {
		t.Fatalf("fixture %d exposed private acceptance tests: %v", index, err)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".scratch", "go")); !os.IsNotExist(err) {
		t.Fatalf("fixture %d received a private module cache: %v", index, err)
	}
}

func (f *sharedWarmupFixture) assertOffline(t *testing.T) {
	if err := os.RemoveAll(f.proxy); err != nil {
		t.Fatal(err)
	}
	for i, workDir := range f.workDirs {
		if err := prepareFixtureGoDependencies(f.ctx, workDir); err != nil {
			t.Fatalf("fixture %d offline dependency preparation: %v", i, err)
		}
		assertSharedPreflightBuildCacheClean(t, workDir)
		runOfflineFixtureTest(f.ctx, t, f.root, i, workDir)
	}
}

func runOfflineFixtureTest(ctx context.Context, t *testing.T, root string, index int, workDir string) {
	goEnv, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		t.Fatal(err)
	}
	testCache := filepath.Join(root, fmt.Sprintf("job-build-%d", index))
	if err := os.MkdirAll(testCache, 0o750); err != nil {
		t.Fatal(err)
	}
	goEnv = replaceEnvironment(goEnv, map[string]string{"GOCACHE": testCache})
	cmd := exec.Command("go", "test", "-mod=readonly", "./...")
	cmd.Dir, cmd.Env = workDir, goEnv
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture %d offline test after proxy removal: %v: %s", index, err, output)
	}
}

func TestFixtureOfflinePolicyDisablesPrivateModuleBypass(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "git-called")
	fakeGit := filepath.Join(bin, "git")
	writeTestFile(t, fakeGit, "#!/bin/sh\nprintf called > \"$SEMEDIT_TEST_GIT_MARKER\"\nexit 91\n")
	if err := os.Chmod(fakeGit, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SEMEDIT_TEST_GIT_MARKER", marker)
	t.Setenv("GOPRIVATE", "github.com/semedit-benchmark-private")
	t.Setenv("GONOPROXY", "github.com/semedit-benchmark-private")
	workDir := filepath.Join(root, "fixture")
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(workDir, "go.mod"), "module fixture\n\ngo 1.23\n")
	ctx := gocache.WithModuleCache(t.Context(), filepath.Join(root, "modules"))
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "go", "mod", "download", "github.com/semedit-benchmark-private/missing@v1.0.0")
	command.Dir, command.Env = workDir, env
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "module lookup disabled by GOPROXY=off") {
		t.Fatalf("offline module lookup = %v: %s", err, output)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("offline policy invoked VCS: %v", err)
	}
}
