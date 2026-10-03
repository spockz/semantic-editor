// Package mcp resolves request-selected roots here to bind dispatch, trust, and path checks together.
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/backend"
)

var filesystemPathFields = map[string]struct{}{
	"file":             {},
	"path":             {},
	"paths":            {},
	"source_file":      {},
	"destination_file": {},
	"target_file":      {},
	"contract_file":    {},
}

func resolveRequestRoot(configuredRoot, requestedRoot string) (string, error) {
	base := configuredRoot
	if base == "" {
		base = "."
	}
	base, err := filepath.Abs(filepath.Clean(base))
	if err != nil {
		return "", fmt.Errorf("resolve configured workspace root %q: %w", configuredRoot, err)
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return "", fmt.Errorf("canonicalize configured workspace root %q: %w", configuredRoot, err)
	}
	if requestedRoot == "" {
		requestedRoot = base
	} else if !filepath.IsAbs(requestedRoot) {
		requestedRoot = filepath.Join(base, requestedRoot)
	}
	root, err := filepath.Abs(filepath.Clean(requestedRoot))
	if err != nil {
		return "", fmt.Errorf("resolve workspace root %q: %w", requestedRoot, err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("canonicalize workspace root %q: %w", requestedRoot, err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat workspace root %q: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace root %q is not a directory", root)
	}
	if !pathContained(base, root) {
		return "", fmt.Errorf("workspace root %q is outside configured root %q", root, base)
	}
	return root, nil
}

func pathContained(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func requestPathWithinRoot(root, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	clean := filepath.Clean(raw)
	if filepath.IsAbs(raw) {
		lexicalRelative, err := filepath.Rel(root, clean)
		if err == nil && lexicalRelative != ".." && !strings.HasPrefix(lexicalRelative, ".."+string(filepath.Separator)) {
			clean = lexicalRelative
		} else {
			canonical, err := canonicalizeExistingParent(clean)
			if err != nil {
				return fmt.Errorf("resolve absolute path %q against workspace root: %w", raw, err)
			}
			clean, err = filepath.Rel(root, canonical)
			if err != nil {
				return fmt.Errorf("resolve absolute path %q against workspace root: %w", raw, err)
			}
		}
	}
	if clean == "." {
		return nil
	}
	resolved := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside workspace root %q", raw, root)
	}
	current := root
	for part := range strings.SplitSeq(clean, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			current = filepath.Dir(current)
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect workspace path %q: %w", raw, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace path %q traverses symlink %q", raw, current)
		}
	}
	return nil
}

func canonicalizeExistingParent(path string) (string, error) {
	probe := filepath.Clean(path)
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			for _, m := range slices.Backward(missing) {
				resolved = filepath.Join(resolved, m)
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", err
		}
		missing = append(missing, filepath.Base(probe))
		probe = parent
	}
}

func canonicalTrust(root string, trusted bool) backend.WorkspaceTrust {
	return backend.NewWorkspaceTrust(root, trusted)
}

func validateRequestPaths(root string, raw map[string]any) error {
	_, err := validateRequestPathValue(root, raw, "")
	return err
}

func validateRequestPathValue(root string, value any, location string) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childLocation := key
			if location != "" {
				childLocation = location + "." + key
			}
			var err error
			if _, isPath := filesystemPathFields[key]; isPath {
				typed[key], err = validateFilesystemPathValue(root, child, childLocation)
			} else {
				typed[key], err = validateRequestPathValue(root, child, childLocation)
			}
			if err != nil {
				return nil, err
			}
		}
		return typed, nil
	case []any:
		for index, child := range typed {
			childLocation := fmt.Sprintf("%s[%d]", location, index)
			var err error
			typed[index], err = validateRequestPathValue(root, child, childLocation)
			if err != nil {
				return nil, err
			}
		}
		return typed, nil
	default:
		return value, nil
	}
}

func validateFilesystemPathValue(root string, value any, location string) (any, error) {
	validate := func(path, field string) (string, error) {
		if err := requestPathWithinRoot(root, path); err != nil {
			return "", fmt.Errorf("invalid %s: %w", field, err)
		}
		if filepath.IsAbs(path) {
			relative, err := canonicalRequestRelativePath(root, path)
			if err != nil {
				return "", fmt.Errorf("invalid %s: %w", field, err)
			}
			return relative, nil
		}
		return path, nil
	}
	switch typed := value.(type) {
	case string:
		return validate(typed, location)
	case []any:
		for index, item := range typed {
			path, ok := item.(string)
			if !ok {
				continue
			}
			checked, err := validate(path, fmt.Sprintf("%s[%d]", location, index))
			if err != nil {
				return nil, err
			}
			typed[index] = checked
		}
		return typed, nil
	default:
		return value, nil
	}
}

func canonicalRequestRelativePath(root, path string) (string, error) {
	clean := filepath.Clean(path)
	if relative, err := filepath.Rel(root, clean); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return relative, nil
	}
	canonical, err := canonicalizeExistingParent(clean)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, canonical)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside workspace root %q", path, root)
	}
	return relative, nil
}

func prepareWorkspaceArguments(configuredRoot string, raw map[string]any) (string, error) {
	rootValue, present := raw["workspace_root"]
	if present {
		if _, ok := rootValue.(string); !ok {
			return "", fmt.Errorf("workspace_root must be a string")
		}
	}
	if trustValue, present := raw["trust_workspace"]; present {
		if _, ok := trustValue.(bool); !ok {
			return "", fmt.Errorf("trust_workspace must be a boolean")
		}
	}
	rootText, _ := rootValue.(string)
	delete(raw, "workspace_root")
	root, err := resolveRequestRoot(configuredRoot, rootText)
	if err != nil {
		return "", err
	}
	if err := validateRequestPaths(root, raw); err != nil {
		return "", err
	}
	return root, nil
}

func requestTrustedRoot(root string, configured backend.WorkspaceTrust, raw map[string]any) bool {
	if trusted, ok := raw["trust_workspace"].(bool); ok {
		return trusted
	}
	return configured.Trusted && configured.Allows(root)
}
