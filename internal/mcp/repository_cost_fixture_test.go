// This file exports one committed source tree and creates isolated benchmark fixtures from it so setup does not measure working-tree discovery or copying strategy.
package mcp_test

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
)

type repositoryCostCorpus struct {
	root     string
	revision string
}

func newRepositoryCostCorpus(tb testing.TB) repositoryCostCorpus {
	tb.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("get working directory for repository corpus: %v", err)
	}
	repositoryRootBytes, err := exec.Command("git", "-C", workingDir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		tb.Fatalf("resolve repository root for corpus: %v", err)
	}
	repositoryRoot := strings.TrimSpace(string(repositoryRootBytes))
	revisionBytes, err := exec.Command("git", "-C", repositoryRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		tb.Fatalf("resolve repository corpus revision: %v", err)
	}
	revision := strings.TrimSpace(string(revisionBytes))
	base := tb.TempDir()
	archivePath := filepath.Join(base, "repository.tar")
	archiveArgs := append([]string{"-C", repositoryRoot, "archive", "--format=tar", "--output", archivePath, revision, "--"}, repositoryCostArchivePaths(tb, repositoryRoot, revision)...)
	archiveCommand := exec.Command("git", archiveArgs...)
	if output, err := archiveCommand.CombinedOutput(); err != nil {
		tb.Fatalf("export repository corpus at %s: %v: %s", revision, err, strings.TrimSpace(string(output)))
	}
	staged := filepath.Join(base, "staged")
	if err := os.Mkdir(staged, 0o755); err != nil {
		tb.Fatalf("create staged repository corpus: %v", err)
	}
	command := exec.Command("tar", "-xf", archivePath, "-C", staged)
	if output, err := command.CombinedOutput(); err != nil {
		tb.Fatalf("extract repository corpus at %s: %v: %s", revision, err, strings.TrimSpace(string(output)))
	}
	root := filepath.Join(base, "repository")
	if err := os.Rename(staged, root); err != nil {
		tb.Fatalf("publish repository corpus at %s: %v", revision, err)
	}
	for _, relativePath := range []string{"go.mod", filepath.Join("internal", "telemetry", "timing.go")} {
		if _, err := os.Stat(filepath.Join(root, relativePath)); err != nil {
			tb.Fatalf("repository corpus at %s is missing %s: %v", revision, relativePath, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		tb.Fatalf("repository corpus at %s unexpectedly contains .git: %v", revision, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "data", "benchmarks")); !os.IsNotExist(err) {
		tb.Fatalf("repository corpus at %s unexpectedly contains benchmark artifacts: %v", revision, err)
	}
	return repositoryCostCorpus{root: root, revision: revision}
}

func repositoryCostArchivePaths(tb testing.TB, root, revision string) []string {
	tb.Helper()
	entries, err := exec.Command("git", "-C", root, "ls-tree", "-z", "--name-only", revision).Output()
	if err != nil {
		tb.Fatalf("list repository corpus paths: %v", err)
	}
	var paths []string
	for entry := range strings.SplitSeq(strings.TrimSuffix(string(entries), "\x00"), "\x00") {
		if entry != "data" {
			paths = append(paths, entry)
			continue
		}
		children, err := exec.Command("git", "-C", root, "ls-tree", "-z", "--name-only", revision+":data").Output()
		if err != nil {
			tb.Fatalf("list repository corpus data paths: %v", err)
		}
		for child := range strings.SplitSeq(strings.TrimSuffix(string(children), "\x00"), "\x00") {
			if child != "benchmarks" {
				paths = append(paths, "data/"+child)
			}
		}
	}
	return paths
}

func (c repositoryCostCorpus) clone(tb testing.TB) string {
	tb.Helper()
	destination := filepath.Join(tb.TempDir(), "repository")
	if runtime.GOOS == "darwin" {
		output, err := exec.Command("/bin/cp", "-cR", c.root, destination).CombinedOutput()
		if err != nil {
			tb.Fatalf("clone repository corpus with macOS copy-on-write: %v: %s", err, strings.TrimSpace(string(output)))
		}
		return destination
	}
	if err := copyRepositoryCostTree(c.root, destination); err != nil {
		tb.Fatalf("copy repository corpus: %v", err)
	}
	return destination
}

func copyRepositoryCostTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && path != source && sourcefiles.IsBenchmarkDataDirectory(path) {
			return filepath.SkipDir
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("relativize corpus path %q: %w", path, err)
		}
		target := filepath.Join(destination, relative)
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect corpus path %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return copyRepositoryCostSymlink(path, target)
		}
		if entry.IsDir() {
			if relative == "." {
				return os.MkdirAll(destination, info.Mode().Perm())
			}
			if mkdirErr := os.Mkdir(target, info.Mode().Perm()); mkdirErr != nil {
				return fmt.Errorf("copy corpus directory %q: %w", path, mkdirErr)
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read corpus file %q: %w", path, err)
		}
		if err := pipeline.WriteAtomic(target, data); err != nil {
			return fmt.Errorf("write copied corpus file %q: %w", target, err)
		}
		if err := os.Chmod(target, info.Mode().Perm()); err != nil {
			return fmt.Errorf("preserve corpus file mode %q: %w", target, err)
		}
		return nil
	})
}

func copyRepositoryCostSymlink(source, destination string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return fmt.Errorf("read corpus symlink %q: %w", source, err)
	}
	if err := os.Symlink(link, destination); err != nil {
		return fmt.Errorf("copy corpus symlink %q: %w", source, err)
	}
	return nil
}

func TestRepositoryCostCorpusCloneIsolation(t *testing.T) {
	corpus := newRepositoryCostCorpus(t)
	first := corpus.clone(t)
	second := corpus.clone(t)
	readme := filepath.Join(corpus.root, "README.md")
	original, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read corpus README.md: %v", err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(first, "README.md"), []byte("mutated fixture")); err != nil {
		t.Fatalf("mutate first fixture README.md: %v", err)
	}
	for label, path := range map[string]string{"source corpus": readme, "second fixture": filepath.Join(second, "README.md")} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s README.md: %v", label, err)
		}
		if string(contents) != string(original) {
			t.Fatalf("mutating the first fixture changed %s", label)
		}
	}
}
