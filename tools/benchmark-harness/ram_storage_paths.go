// ram_storage_paths.go prevents published benchmark evidence from being written to disposable RAM storage.

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func validateDurableBenchmarkStorage(options PlanOptions) error {
	if options.ModuleCache != "" {
		if err := validateDurableStoragePath("persistent module cache", options.ModuleCache, true); err != nil {
			return err
		}
	}
	if options.RAMFixtureRoot == "" {
		return nil
	}
	if options.ModuleCache == "" {
		return fmt.Errorf("--ram-fixture-root requires --module-cache on persistent storage")
	}
	for _, candidate := range []struct {
		name string
		path string
		dir  bool
	}{
		{"benchmark results", options.OutDir, true},
		{"benchmark JSON report", options.OutJSON, false},
		{"benchmark Markdown report", options.OutMD, false},
		{"persistent benchmark scratch", filepath.Join(".scratch", "benchmarks"), true},
	} {
		if candidate.path == "" {
			continue
		}
		if err := validateDurableStoragePath(candidate.name, candidate.path, candidate.dir); err != nil {
			return err
		}
	}
	return nil
}

func validateDurableStoragePath(name, path string, requireDirectory bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", name, err)
	}
	storagePath, err := nearestExistingStoragePath(absolute)
	if err != nil {
		return fmt.Errorf("inspect %s %q: %w", name, absolute, err)
	}
	if info, statErr := os.Stat(absolute); statErr == nil {
		if requireDirectory && !info.IsDir() {
			return fmt.Errorf("%s %q is not a directory", name, absolute)
		}
		if !requireDirectory && info.IsDir() {
			return fmt.Errorf("%s %q is a directory", name, absolute)
		}
	} else if !os.IsNotExist(statErr) || name == "persistent module cache" {
		return fmt.Errorf("inspect %s %q: %w", name, absolute, statErr)
	}
	tmpfs, err := isTmpfsMount(storagePath)
	if err != nil {
		return fmt.Errorf("inspect %s %q: %w", name, absolute, err)
	}
	if tmpfs {
		return fmt.Errorf("%s %q must use persistent storage outside tmpfs", name, absolute)
	}
	return nil
}

func nearestExistingStoragePath(path string) (string, error) {
	first := true
	for {
		if _, err := os.Lstat(path); err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(path)
			if resolveErr != nil {
				return "", fmt.Errorf("canonicalize %q: %w", path, resolveErr)
			}
			info, statErr := os.Stat(resolved)
			if statErr != nil {
				return "", fmt.Errorf("inspect %q: %w", resolved, statErr)
			}
			if !first && !info.IsDir() {
				return "", fmt.Errorf("%q is not a directory", path)
			}
			return resolved, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("no existing parent")
		}
		path = parent
		first = false
	}
}
