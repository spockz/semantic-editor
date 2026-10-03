// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file resolves source and module paths before analysis so workspace containment stays feature-owned.
package contractedit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/sourcebuf"
)

// FindModuleRoot finds the module containing the first selected source file.
func FindModuleRoot(ctx context.Context, workspaceRoot string, selectedPaths []string) (string, error) {
	if len(selectedPaths) == 0 {
		return "", fmt.Errorf("at least one contract source file is required")
	}
	selected, err := CanonicalPath(workspaceRoot, selectedPaths[0])
	if err != nil {
		return "", err
	}
	current := filepath.Dir(selected)
	for {
		if _, readErr := sourcebuf.ReadFile(ctx, filepath.Join(current, "go.mod")); readErr == nil {
			return current, nil
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return "", fmt.Errorf("read contract module manifest: %w", readErr)
		}
		if current == workspaceRoot {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("selected contract source %q is not inside a Go module", selectedPaths[0])
}

// CanonicalPath resolves one existing Go file within the selected workspace root.
func CanonicalPath(root, selected string) (string, error) {
	if strings.TrimSpace(selected) == "" {
		return "", fmt.Errorf("contract file path is required")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve contract workspace root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("canonicalize contract workspace root: %w", err)
	}
	path := selected
	if !filepath.IsAbs(path) {
		path = filepath.Join(canonicalRoot, path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve contract file %q: %w", selected, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize contract file %q: %w", selected, err)
	}
	relative, err := filepath.Rel(canonicalRoot, canonical)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("contract file %q must remain inside the workspace", selected)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("inspect contract file %q: %w", selected, err)
	}
	if !info.Mode().IsRegular() || filepath.Ext(canonical) != ".go" {
		return "", fmt.Errorf("contract file %q must be an existing Go source file", selected)
	}
	return canonical, nil
}
