// Package java implements Java language-server operations and source transformations.
// WHY: This file infers package names so later semantic edits use the right namespace.
package java

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

var javaPackageLine = regexp.MustCompile(`^\s*package\s+([^;]+)\s*;\s*$`)

var javaKeywords = map[string]struct{}{
	"abstract": {}, "assert": {}, "boolean": {}, "break": {}, "byte": {}, "case": {}, "catch": {}, "char": {},
	"class": {}, "const": {}, "continue": {}, "default": {}, "do": {}, "double": {}, "else": {}, "enum": {},
	"extends": {}, "final": {}, "finally": {}, "float": {}, "for": {}, "goto": {}, "if": {}, "implements": {},
	"import": {}, "instanceof": {}, "int": {}, "interface": {}, "long": {}, "native": {}, "new": {}, "package": {},
	"private": {}, "protected": {}, "public": {}, "return": {}, "short": {}, "static": {}, "strictfp": {},
	"super": {}, "switch": {}, "synchronized": {}, "this": {}, "throw": {}, "throws": {}, "transient": {},
	"try": {}, "void": {}, "volatile": {}, "while": {}, "_": {},
}

// ScaffoldFile writes a Java package declaration atomically, preserving a caller-owned source buffer when present.
func ScaffoldFile(ctx context.Context, root, file, requestedPackage string, overwrite bool) (string, error) {
	root = backend.CanonicalWorkspaceRoot(root)
	file = canonicalJavaFilePath(file)
	if !pathutil.PathWithin(root, file) || !strings.EqualFold(filepath.Ext(file), ".java") {
		return "", fmt.Errorf("java scaffold target must be a .java file inside the workspace: %s", file)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := sourcebuf.ReadFile(ctx, file); err == nil && !overwrite {
		return "", fmt.Errorf("java scaffold target already exists: %s", file)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect Java scaffold target: %w", err)
	}

	packageName := strings.TrimSpace(requestedPackage)
	if packageName == "" || packageName == "infer" {
		var err error
		packageName, err = inferJavaScaffoldPackage(file)
		if err != nil {
			return "", err
		}
	}
	if err := validateJavaPackageName(packageName); err != nil {
		return "", err
	}

	content := []byte("// WHY: Starts this Java source file in its package so declarations can be added with semantic operations.\n")
	if packageName != "" {
		content = append(content, []byte("package "+packageName+";\n\n")...)
	}
	if sourcebuf.FromContext(ctx) == nil {
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			return "", fmt.Errorf("create Java scaffold directory: %w", err)
		}
	}
	if err := pipeline.WriteAtomicContext(ctx, file, content); err != nil {
		return "", fmt.Errorf("write Java scaffold atomically: %w", err)
	}
	return packageName, nil
}

func inferJavaScaffoldPackage(file string) (string, error) {
	dir := filepath.Dir(file)
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read Java package siblings: %w", err)
	}
	packages := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".java") || entry.Name() == filepath.Base(file) {
			continue
		}
		name, declared, err := readJavaPackage(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", err
		}
		if declared {
			packages[name] = struct{}{}
		}
	}
	if len(packages) > 1 {
		return "", fmt.Errorf("java sibling files disagree on package in %s", dir)
	}
	for name := range packages {
		return name, nil
	}

	parts := strings.Split(filepath.ToSlash(dir), "/")
	for i, part := range slices.Backward(parts) {
		if part != "java" || i == 0 || (parts[i-1] != "main" && parts[i-1] != "test") || i < 2 || parts[i-2] != "src" {
			continue
		}
		packageParts := parts[i+1:]
		if len(packageParts) == 0 {
			return "", nil
		}
		name := strings.Join(packageParts, ".")
		if err := validateJavaPackageName(name); err != nil {
			return "", fmt.Errorf("infer Java package from source path: %w", err)
		}
		return name, nil
	}
	return "", fmt.Errorf("cannot infer Java package from siblings or Maven source path for %s", file)
}

func readJavaPackage(file string) (string, bool, error) {
	data, err := os.ReadFile(file) // #nosec G304 -- file is an enumerated Java sibling of the requested scaffold target.
	if err != nil {
		return "", false, fmt.Errorf("read Java package sibling %q: %w", file, err)
	}
	inBlockComment := false
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if inBlockComment {
			if end := strings.Index(trimmed, "*/"); end >= 0 {
				trimmed = strings.TrimSpace(trimmed[end+2:])
				inBlockComment = false
			} else {
				continue
			}
		}
		for strings.HasPrefix(trimmed, "/*") {
			trimmed = strings.TrimSpace(trimmed[2:])
			if end := strings.Index(trimmed, "*/"); end >= 0 {
				trimmed = strings.TrimSpace(trimmed[end+2:])
			} else {
				inBlockComment = true
				break
			}
		}
		if inBlockComment || trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		match := javaPackageLine.FindStringSubmatch(trimmed)
		if len(match) == 2 {
			name := strings.TrimSpace(match[1])
			if err := validateJavaPackageName(name); err != nil {
				return "", false, fmt.Errorf("invalid Java package in %s: %w", file, err)
			}
			return name, true, nil
		}
		if !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "@") {
			return "", false, nil
		}
	}
	return "", false, nil
}

func validateJavaPackageName(name string) error {
	if name == "" {
		return nil
	}
	for part := range strings.SplitSeq(name, ".") {
		if part == "" {
			return fmt.Errorf("invalid Java package name %q", name)
		}
		for i, r := range part {
			valid := r == '_' || r == '$' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r))
			if !valid {
				return fmt.Errorf("invalid Java package name %q", name)
			}
		}
		if _, keyword := javaKeywords[part]; keyword {
			return fmt.Errorf("invalid Java package name %q: keyword %q", name, part)
		}
	}
	return nil
}
