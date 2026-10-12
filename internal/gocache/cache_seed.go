// Package gocache copies explicitly selected offline module caches into managed Go state.
package gocache

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type moduleCacheSeedKey struct{}
type moduleWritesContextKey struct{}

const (
	maxModuleCacheSeedBytes = int64(2 << 30)
	maxModuleCacheSeedFiles = 200_000
)

var (
	seededModuleCacheMu sync.Mutex
	seededModuleCaches  = make(map[string]struct{})
)

type moduleCacheSeedFile struct {
	source      string
	destination string
	size        int64
	mode        os.FileMode
	info        os.FileInfo
}

// WithModuleCacheSeed selects an offline module cache to copy into managed state.
func WithModuleCacheSeed(ctx context.Context, seedDir string) context.Context {
	return context.WithValue(ctx, moduleCacheSeedKey{}, strings.TrimSpace(seedDir))
}

// WithWritableModuleChanges permits explicit dependency operations to update module files.
func WithWritableModuleChanges(ctx context.Context) context.Context {
	return context.WithValue(ctx, moduleWritesContextKey{}, true)
}

func moduleWritesAllowed(ctx context.Context) bool {
	allowed, _ := ctx.Value(moduleWritesContextKey{}).(bool)
	return allowed
}

func seedModuleCache(ctx context.Context, baseDir, destination string) error {
	seed, _ := ctx.Value(moduleCacheSeedKey{}).(string)
	if seed == "" {
		return nil
	}
	if os.Getenv("GOPROXY") != "off" {
		return fmt.Errorf("offline Go module cache seed requires GOPROXY=off")
	}
	seedRoot, err := canonicalCacheDirectory(seed)
	if err != nil {
		return fmt.Errorf("resolve offline module cache seed: %w", err)
	}
	destinationRoot, err := canonicalCacheDirectory(destination)
	if err != nil {
		return fmt.Errorf("resolve managed module cache: %w", err)
	}
	baseRoot, err := canonicalCacheDirectory(baseDir)
	if err != nil {
		return fmt.Errorf("resolve managed Go state directory: %w", err)
	}
	if !cachePathWithin(baseRoot, destinationRoot) {
		return fmt.Errorf("managed module cache %q escapes Go state directory %q", destinationRoot, baseRoot)
	}
	if cachePathsOverlap(seedRoot, destinationRoot) {
		return fmt.Errorf("offline module cache seed and managed module cache must not contain one another")
	}
	key := seedRoot + string(os.PathSeparator) + destinationRoot
	seededModuleCacheMu.Lock()
	defer seededModuleCacheMu.Unlock()
	if _, ok := seededModuleCaches[key]; ok {
		return nil
	}
	if err := seedModuleCacheLocked(seedRoot, destinationRoot); err != nil {
		return err
	}
	seededModuleCaches[key] = struct{}{}
	return nil
}

func seedModuleCacheLocked(seedRoot, destinationRoot string) (result error) {
	sourceTree, err := os.OpenRoot(seedRoot)
	if err != nil {
		return fmt.Errorf("open offline module cache seed: %w", err)
	}
	defer closeSeedRoot(&result, sourceTree, "offline module cache seed")
	destinationTree, err := os.OpenRoot(destinationRoot)
	if err != nil {
		return fmt.Errorf("open managed module cache: %w", err)
	}
	defer closeSeedRoot(&result, destinationTree, "managed module cache")
	files, err := collectModuleCacheSeedFiles(seedRoot, sourceTree, destinationRoot, destinationTree)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := seedModuleCacheFile(sourceTree, destinationTree, file); err != nil {
			return fmt.Errorf("seed module cache file %q: %w", file.source, err)
		}
	}
	return nil
}

func canonicalCacheDirectory(path string) (result string, resultErr error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve directory %q: %w", path, err)
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return "", fmt.Errorf("open directory %q: %w", absolute, err)
	}
	defer closeSeedRoot(&resultErr, root, "module cache directory")
	info, err := root.Stat(".")
	if err != nil {
		return "", fmt.Errorf("inspect directory %q: %w", absolute, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", absolute)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve directory symlinks %q: %w", absolute, err)
	}
	return filepath.Clean(canonical), nil
}

func cachePathWithin(parent, path string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func cachePathsOverlap(first, second string) bool {
	return cachePathWithin(first, second) || cachePathWithin(second, first)
}

func collectModuleCacheSeedFiles(seedRoot string, sourceTree *os.Root, destinationRoot string, destinationTree *os.Root) ([]moduleCacheSeedFile, error) {
	var files []moduleCacheSeedFile
	var total int64
	err := filepath.WalkDir(seedRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk seed path %q: %w", path, walkErr)
		}
		if path == seedRoot {
			return nil
		}
		relative, err := filepath.Rel(seedRoot, path)
		if err != nil {
			return fmt.Errorf("make seed path relative %q: %w", path, err)
		}
		info, err := sourceTree.Lstat(relative)
		if err != nil {
			return fmt.Errorf("inspect seed path %q: %w", path, err)
		}
		if entry.Type()&os.ModeSymlink != 0 || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("offline module cache seed contains symlink %q", path)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("offline module cache seed contains non-regular file %q", path)
		}
		if len(files) >= maxModuleCacheSeedFiles || info.Size() > maxModuleCacheSeedBytes-total {
			return fmt.Errorf("offline module cache seed exceeds limits of %d files or %d bytes", maxModuleCacheSeedFiles, maxModuleCacheSeedBytes)
		}
		files = append(files, moduleCacheSeedFile{source: relative, destination: relative, size: info.Size(), mode: info.Mode().Perm(), info: info})
		total += info.Size()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := validateSeedDestinationParents(destinationRoot, destinationTree, files); err != nil {
		return nil, err
	}
	return files, nil
}

func validateSeedDestinationParents(destinationRoot string, destinationTree *os.Root, files []moduleCacheSeedFile) error {
	for _, file := range files {
		for parent := filepath.Dir(file.destination); parent != "."; parent = filepath.Dir(parent) {
			info, err := destinationTree.Lstat(parent)
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			if err != nil {
				return fmt.Errorf("inspect managed cache directory %q: %w", filepath.Join(destinationRoot, parent), err)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("managed cache parent %q is not a real directory", filepath.Join(destinationRoot, parent))
			}
		}
	}
	return nil
}

func seedModuleCacheFile(sourceTree, destinationTree *os.Root, file moduleCacheSeedFile) (result error) {
	exists, err := seedDestinationMatches(sourceTree, destinationTree, file)
	if err != nil || exists {
		return err
	}
	parent := filepath.Dir(file.destination)
	if mkdirErr := destinationTree.MkdirAll(parent, 0o750); mkdirErr != nil {
		return fmt.Errorf("create cache directory: %w", mkdirErr)
	}
	source, err := openSeedFile(sourceTree, file)
	if err != nil {
		return err
	}
	defer closeSeedFile(&result, source, "offline module cache seed file")
	temporary, temporaryPath, err := createSeedTemp(destinationTree, parent)
	if err != nil {
		return err
	}
	defer removeSeedTemp(&result, destinationTree, temporaryPath)
	if err := writeSeedTemp(temporary, source, file); err != nil {
		return err
	}
	if exists, err := seedDestinationMatches(sourceTree, destinationTree, file); err != nil || exists {
		return err
	}
	if err := destinationTree.Rename(temporaryPath, file.destination); err != nil {
		return fmt.Errorf("publish staged cache file with atomic rename: %w", err)
	}
	return syncSeedDirectory(destinationTree, parent)
}

func seedDestinationMatches(sourceTree, destinationTree *os.Root, file moduleCacheSeedFile) (bool, error) {
	info, err := destinationTree.Lstat(file.destination)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect managed cache destination %q: %w", file.destination, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("managed cache destination %q is not a regular file", file.destination)
	}
	same, err := sameSeedFileContents(sourceTree, destinationTree, file)
	if err != nil {
		return false, err
	}
	if !same {
		return false, fmt.Errorf("offline module cache seed conflicts with existing managed cache file %q", file.destination)
	}
	return true, nil
}

func sameSeedFileContents(sourceTree, destinationTree *os.Root, file moduleCacheSeedFile) (same bool, result error) {
	left, err := openSeedFile(sourceTree, file)
	if err != nil {
		return false, err
	}
	defer closeSeedFile(&result, left, "offline module cache seed file")
	right, err := destinationTree.Open(file.destination)
	if err != nil {
		return false, fmt.Errorf("open managed module cache file: %w", err)
	}
	defer closeSeedFile(&result, right, "managed module cache file")
	rightInfo, err := right.Stat()
	if err != nil {
		return false, fmt.Errorf("stat managed module cache file: %w", err)
	}
	if !rightInfo.Mode().IsRegular() {
		return false, fmt.Errorf("managed cache destination %q is not a regular file", file.destination)
	}
	if rightInfo.Size() != file.size {
		return false, nil
	}
	leftHash, rightHash := sha256.New(), sha256.New()
	leftBytes, err := io.Copy(leftHash, io.LimitReader(left, file.size+1))
	if err != nil || leftBytes != file.size {
		if err != nil {
			return false, fmt.Errorf("read seed file %q: %w", file.source, err)
		}
		return false, fmt.Errorf("seed file %q changed while comparing", file.source)
	}
	rightBytes, err := io.Copy(rightHash, io.LimitReader(right, file.size+1))
	if err != nil {
		return false, fmt.Errorf("read managed cache file %q: %w", file.destination, err)
	}
	return rightBytes == file.size && bytes.Equal(leftHash.Sum(nil), rightHash.Sum(nil)), nil
}

func openSeedFile(sourceTree *os.Root, file moduleCacheSeedFile) (*os.File, error) {
	info, err := sourceTree.Lstat(file.source)
	if err != nil {
		return nil, fmt.Errorf("inspect seed file: %w", err)
	}
	if !info.Mode().IsRegular() || !os.SameFile(file.info, info) || info.Size() != file.size {
		return nil, fmt.Errorf("seed file changed after enumeration")
	}
	source, err := sourceTree.Open(file.source)
	if err != nil {
		return nil, fmt.Errorf("open seed file: %w", err)
	}
	opened, statErr := source.Stat()
	if statErr != nil || !os.SameFile(info, opened) || opened.Size() != file.size {
		closeErr := source.Close()
		if statErr != nil {
			return nil, errors.Join(fmt.Errorf("stat opened seed file: %w", statErr), closeErr)
		}
		return nil, errors.Join(fmt.Errorf("seed file changed while opening"), closeErr)
	}
	return source, nil
}

func createSeedTemp(destinationTree *os.Root, parent string) (*os.File, string, error) {
	for range 5 {
		random := make([]byte, 12)
		if _, err := rand.Read(random); err != nil {
			return nil, "", fmt.Errorf("generate staged cache name: %w", err)
		}
		path := filepath.Join(parent, ".semedit-cache-seed-"+hex.EncodeToString(random))
		file, err := destinationTree.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("create staged cache file: %w", err)
		}
		return file, path, nil
	}
	return nil, "", fmt.Errorf("could not allocate a unique staged cache file after five attempts")
}

func writeSeedTemp(temporary, source *os.File, file moduleCacheSeedFile) (result error) {
	defer closeSeedFile(&result, temporary, "staged module cache file")
	written, err := io.Copy(temporary, io.LimitReader(source, file.size+1))
	if err != nil {
		return fmt.Errorf("copy cache contents: %w", err)
	}
	if written != file.size {
		return fmt.Errorf("seed file changed while copying: copied %d bytes, expected %d", written, file.size)
	}
	if err := temporary.Chmod(file.mode); err != nil {
		return fmt.Errorf("set cache file mode: %w", err)
	}
	return temporary.Sync()
}

func syncSeedDirectory(root *os.Root, parent string) (result error) {
	directory, err := root.Open(parent)
	if err != nil {
		return fmt.Errorf("open cache directory for sync: %w", err)
	}
	defer closeSeedFile(&result, directory, "managed module cache directory")
	return directory.Sync()
}

func closeSeedRoot(result *error, root *os.Root, description string) {
	if err := root.Close(); err != nil {
		*result = errors.Join(*result, fmt.Errorf("close %s: %w", description, err))
	}
}

func closeSeedFile(result *error, file *os.File, description string) {
	if err := file.Close(); err != nil {
		*result = errors.Join(*result, fmt.Errorf("close %s: %w", description, err))
	}
}

func removeSeedTemp(result *error, root *os.Root, path string) {
	if err := root.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		*result = errors.Join(*result, fmt.Errorf("remove staged cache file %q: %w", path, err))
	}
}
