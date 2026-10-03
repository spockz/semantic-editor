// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file validates active test variants and symlink inputs before a contract candidate is published.
package contractedit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func validateTestVariants(loaded []*packages.Package, destinationDirectory string) error {
	var validationErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if validationErr != nil || pkg == nil {
			return validationErr == nil
		}
		for _, ignored := range pkg.IgnoredFiles {
			if !strings.HasSuffix(ignored, "_test.go") {
				continue
			}
			sameDirectory, err := sameDirectory(ignored, destinationDirectory)
			if err != nil {
				validationErr = fmt.Errorf("resolve ignored test source %q: %w", ignored, err)
				return false
			}
			if sameDirectory {
				validationErr = fmt.Errorf("contract preflight cannot prove inactive test source %q under the current build tags", ignored)
				return false
			}
		}
		return validateCompiledTestSources(pkg, destinationDirectory, &validationErr)
	}, nil)
	return validationErr
}

func validateCompiledTestSources(pkg *packages.Package, destinationDirectory string, validationErr *error) bool {
	hasTestSource := false
	for _, filename := range pkg.CompiledGoFiles {
		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}
		inDirectory, err := sameDirectory(filename, destinationDirectory)
		if err != nil {
			*validationErr = fmt.Errorf("resolve compiled test source %q: %w", filename, err)
			return false
		}
		hasTestSource = hasTestSource || inDirectory
	}
	if !hasTestSource {
		return true
	}
	if pkg.Types == nil {
		*validationErr = fmt.Errorf("contract test variant %q has incomplete type information", pkg.ID)
		return false
	}
	if len(pkg.Errors) > 0 {
		*validationErr = fmt.Errorf("contract test variant %q has type errors: %s", pkg.ID, pkg.Errors[0].Msg)
		return false
	}
	return true
}

func sameDirectory(path, directory string) (bool, error) {
	pathDirectory, err := canonicalDirectory(filepath.Dir(path))
	if err != nil {
		return false, err
	}
	canonicalDirectory, err := canonicalDirectory(directory)
	if err != nil {
		return false, err
	}
	return pathDirectory == canonicalDirectory, nil
}

func canonicalDirectory(directory string) (string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve directory %q: %w", directory, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize directory %q: %w", absolute, err)
	}
	return canonical, nil
}

func validateTestSymlinks(root, directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read destination package directory %q: %w", directory, err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		if err := testSymlinkError(root, filepath.Join(directory, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func testSymlinkError(root, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect test source %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve test source symlink %q: %w", path, err)
	}
	relative, err := filepath.Rel(root, canonical)
	if err != nil {
		return fmt.Errorf("check test source symlink %q against snapshot root: %w", path, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("test source symlink %q targets %q outside the snapshot root; refusing untracked source", path, canonical)
	}
	return fmt.Errorf("destination-local test symlink %q targets %q; contract preflight does not support test symlinks", path, canonical)
}
