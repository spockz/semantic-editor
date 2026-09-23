// Package pathutil centralizes filesystem path and file-URI rules shared by backends.
package pathutil

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalWorkspaceRoot returns a stable absolute path and resolves existing symlinks.
func CanonicalWorkspaceRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	absolute, err := filepath.Abs(root)
	if err == nil {
		root = absolute
	}
	root = filepath.Clean(root)
	if evaluated, err := filepath.EvalSymlinks(root); err == nil {
		root = filepath.Clean(evaluated)
	}
	return root
}

// FileURI converts a filesystem path into a file URI.
func FileURI(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

// PathWithin reports whether path resolves inside root without traversing above it.
func PathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// FilePathFromURI parses a local file URI and canonicalizes its path.
func FilePathFromURI(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "file" || parsed.Host != "" {
		return "", errors.New("invalid file URI")
	}
	if parsed.Path == "" {
		return "", errors.New("empty file URI")
	}
	return CanonicalWorkspaceRoot(parsed.Path), nil
}

// FileExists reports whether path names a non-directory filesystem entry.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
