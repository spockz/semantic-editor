// This package is the shared source-file and ignored-directory registry used by language discovery and verification planning.

// Package sourcefiles shares source-language and ignored-directory classification.
package sourcefiles

import (
	"maps"
	"path/filepath"
	"strings"
)

// LanguageForPath returns the recognized source language for path, or an empty string.
func LanguageForPath(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if base == "makefile" || base == "gnumakefile" || strings.EqualFold(filepath.Ext(path), ".mk") {
		return "make"
	}
	return extensionLanguages[strings.ToLower(filepath.Ext(path))]
}

// IsIgnoredDirectory reports whether discovery skips a directory name.
func IsIgnoredDirectory(name string) bool {
	return ignoredDirectories[name]
}

// ExtensionMap returns the recognized source extensions mapped to language identifiers.
func ExtensionMap[T ~string]() map[string]T {
	result := make(map[string]T, len(extensionLanguages))
	for extension, language := range extensionLanguages {
		result[extension] = T(language)
	}
	return result
}

// IgnoredDirectories returns the directory names excluded during source discovery.
func IgnoredDirectories() map[string]bool {
	result := make(map[string]bool, len(ignoredDirectories))
	maps.Copy(result, ignoredDirectories)
	return result
}

var ignoredDirectories = map[string]bool{
	".git": true, ".scratch": true, ".cache": true, ".gradle": true, "vendor": true,
	"node_modules": true, "target": true, "build": true, "dist": true,
}

var extensionLanguages = map[string]string{
	".go": "go", ".rs": "rust", ".java": "java", ".scala": "scala", ".hs": "haskell",
	".kt": "kotlin", ".kts": "kotlin", ".sh": "bash", ".bash": "bash",
}
