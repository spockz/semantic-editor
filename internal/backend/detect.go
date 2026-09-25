// Package backend detects source languages because workspace manifests can omit or misrepresent source files.
package backend

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

var sourceExtensions = map[string]LanguageID{
	".go": LanguageGo, ".rs": LanguageRust, ".java": LanguageJava,
	".scala": LanguageScala, ".hs": LanguageHaskell,
	".kt": LanguageKotlin, ".kts": LanguageKotlin, ".sh": LanguageBash, ".bash": LanguageBash,
}

var ignoredSourceDirectories = map[string]bool{
	".git": true, ".scratch": true, ".cache": true, ".gradle": true,
	"vendor": true, "node_modules": true, "target": true, "build": true, "dist": true,
}

// DetectLanguages returns source languages present under root, ordered by language ID.
// It skips generated, dependency, and worktree directories so repository tooling does
// not make a source-less project appear polyglot.
func DetectLanguages(root string) ([]LanguageID, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	found := map[LanguageID]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan source languages at %s: %w", path, walkErr)
		}
		if entry.IsDir() && path != root && ignoredSourceDirectories[entry.Name()] {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(entry.Name(), "Makefile") || strings.EqualFold(entry.Name(), "GNUmakefile") || strings.EqualFold(entry.Name(), "makefile") || strings.EqualFold(filepath.Ext(entry.Name()), ".mk") {
			found[LanguageMake] = true
		}
		if language, ok := sourceExtensions[strings.ToLower(filepath.Ext(entry.Name()))]; ok {
			found[language] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	languages := make([]LanguageID, 0, len(found))
	for language := range found {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	return languages, nil
}

// LanguageIDFromFile maps a recognized source-file path to its language ID.
func LanguageIDFromFile(path string) LanguageID {
	if strings.EqualFold(filepath.Base(path), "makefile") || strings.EqualFold(filepath.Base(path), "gnumakefile") || strings.EqualFold(filepath.Ext(path), ".mk") {
		return LanguageMake
	}
	return sourceExtensions[strings.ToLower(filepath.Ext(path))]
}

// ParseEnabledLanguages parses a comma-separated list of concrete language IDs.
func ParseEnabledLanguages(value string) ([]LanguageID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	seen := map[LanguageID]bool{}
	var languages []LanguageID
	for part := range strings.SplitSeq(value, ",") {
		language := LanguageID(strings.TrimSpace(part))
		switch language {
		case LanguageGo, LanguageRust, LanguageJava, LanguageScala, LanguageHaskell, LanguageKotlin, LanguageBash, LanguageMake:
		default:
			return nil, fmt.Errorf("unknown enabled language %q: %w", part, ErrInvalidLanguage)
		}
		if !seen[language] {
			seen[language] = true
			languages = append(languages, language)
		}
	}
	slices.Sort(languages)
	return languages, nil
}
