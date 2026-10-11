// Dependency tests prove prepared fixtures can build offline without changing their module manifests.

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

import "archive/zip"

func TestPreparedFixtureBuildsOfflineWithoutManifestChanges(t *testing.T) {
	root := t.TempDir()
	proxy := createFixtureDependencyProxy(t, root)
	workDir := filepath.Join(root, "fixture")
	if mkdirErr := os.Mkdir(workDir, 0o750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	for name, content := range map[string]string{
		"go.mod":       "module example.com/fixture\ngo 1.21\nrequire example.com/dependency v1.0.0\n",
		"main_test.go": "package fixture\nimport (\"testing\"; \"example.com/dependency\")\nfunc TestValue(t *testing.T) { if dependency.Value() != 42 { t.Fatal(\"wrong value\") } }\n",
	} {
		if writeErr := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	t.Setenv("GOPROXY", "file://"+proxy)
	t.Setenv("GOSUMDB", "off")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	env, err := fixtureGoEnvironment(ctx, workDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanFixtureModuleCache(t, workDir, env) })
	download := exec.CommandContext(ctx, "go", "mod", "download", "example.com/dependency")
	download.Dir, download.Env = workDir, env
	if output, downloadErr := download.CombinedOutput(); downloadErr != nil {
		t.Fatalf("seed checksums: %v: %s", downloadErr, output)
	}
	before, err := snapshotWorkspaceFiles(workDir)
	if err != nil {
		t.Fatal(err)
	}
	cleanFixtureModuleCache(t, workDir, env)
	if prepareErr := prepareFixtureGoDependencies(ctx, workDir); prepareErr != nil {
		t.Fatal(prepareErr)
	}
	after, err := snapshotWorkspaceFiles(workDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		if before[name] != after[name] {
			t.Fatalf("%s changed during preparation", name)
		}
	}
	offline := exec.CommandContext(ctx, "go", "test", "-mod=readonly", "./...")
	offline.Dir = workDir
	offline.Env = replaceEnvironment(env, map[string]string{"GOPROXY": "off", "GOSUMDB": "off"})
	if output, offlineErr := offline.CombinedOutput(); offlineErr != nil {
		t.Fatalf("offline fixture test: %v: %s", offlineErr, output)
	}
}

func createFixtureDependencyProxy(t *testing.T, root string) string {
	t.Helper()
	proxyRoot := filepath.Join(root, "proxy")
	versionDir := filepath.Join(proxyRoot, "example.com", "dependency", "@v")
	if mkdirErr := os.MkdirAll(versionDir, 0o750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	module := "module example.com/dependency\ngo 1.21\n"
	for name, content := range map[string]string{
		"v1.0.0.mod":  module,
		"v1.0.0.info": `{"Version":"v1.0.0","Time":"2020-01-01T00:00:00Z"}`,
	} {
		if writeErr := os.WriteFile(filepath.Join(versionDir, name), []byte(content), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	archive, err := os.Create(filepath.Join(versionDir, "v1.0.0.zip"))
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	for path, content := range map[string]string{"go.mod": module, "dependency.go": "package dependency\nfunc Value() int { return 42 }\n"} {
		entry, createErr := writer.Create("example.com/dependency@v1.0.0/" + path)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(content)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if closeErr := archive.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	return proxyRoot
}

func cleanFixtureModuleCache(t *testing.T, workDir string, env []string) {
	t.Helper()
	cmd := exec.Command("go", "clean", "-modcache")
	cmd.Dir, cmd.Env = workDir, env
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clean fixture module cache: %v: %s", err, output)
	}
}
