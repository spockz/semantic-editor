// WHY: This test guarantees every backend-supported language has an accompanying reference guide in skills/semedit/references/ without documentation drift.

package operation_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
)

func TestSupportedLanguagesHaveSkillReferences(t *testing.T) {
	t.Parallel()

	rootDir := findRepoRoot(t)
	referencesDir := filepath.Join(rootDir, "skills", "semedit", "references")
	skillMainPath := filepath.Join(rootDir, "skills", "semedit", "SKILL.md")

	skillMainContent, err := os.ReadFile(skillMainPath)
	if err != nil {
		t.Fatalf("read main skill file %s: %v", skillMainPath, err)
	}
	skillMainText := string(skillMainContent)

	registry := operation.DefaultRegistry()
	supportedLanguages := collectSupportedLanguages(registry)
	if len(supportedLanguages) == 0 {
		t.Fatal("expected registry to advertise supported languages, found none")
	}

	for _, lang := range supportedLanguages {
		t.Run(string(lang), func(t *testing.T) {
			t.Parallel()
			assertLanguageReferenceFile(t, lang, referencesDir, skillMainText, registry)
		})
	}
}

func collectSupportedLanguages(registry *operation.Registry) []backend.LanguageID {
	supported := make([]backend.LanguageID, 0)
	for _, entry := range registry.All() {
		for _, lang := range entry.Languages {
			if lang != "" && lang != backend.LanguageAuto && !slices.Contains(supported, lang) {
				supported = append(supported, lang)
			}
		}
	}
	slices.Sort(supported)
	return supported
}

func assertLanguageReferenceFile(t *testing.T, lang backend.LanguageID, referencesDir, skillMainText string, registry *operation.Registry) {
	refFileName := referenceFileNameForLanguage(lang)
	refPath := filepath.Join(referencesDir, refFileName)

	content, readErr := os.ReadFile(refPath)
	if readErr != nil {
		t.Fatalf("language %q missing skill reference file %s: %v", lang, refFileName, readErr)
	}
	if len(content) < 50 {
		t.Errorf("language reference file %s is suspiciously short (%d bytes)", refFileName, len(content))
	}

	text := string(content)
	if !strings.HasPrefix(text, "# ") {
		t.Errorf("language reference file %s must start with top-level heading '# '", refFileName)
	}

	expectedLink := "references/" + refFileName
	if !strings.Contains(skillMainText, expectedLink) {
		t.Errorf("main SKILL.md does not contain link to %s", expectedLink)
	}

	detailsDir := filepath.Join(referencesDir, strings.TrimSuffix(refFileName, filepath.Ext(refFileName)))
	if _, statErr := os.Stat(detailsDir); statErr == nil {
		err := filepath.WalkDir(detailsDir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			additional, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text += "\n" + string(additional)
			return nil
		})
		if err != nil {
			t.Fatalf("read language reference details for %s: %v", lang, err)
		}
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("inspect language reference details for %s: %v", lang, statErr)
	}

	assertLanguageToolsMentioned(t, lang, refFileName, text, registry)
}

func assertLanguageToolsMentioned(t *testing.T, lang backend.LanguageID, refFileName, text string, registry *operation.Registry) {
	for _, entry := range registry.All() {
		if entry.MCPName == "" || !slices.Contains(entry.Languages, lang) {
			continue
		}
		if !strings.Contains(text, entry.MCPName) {
			t.Errorf("language %q reference file %s does not mention supported tool %q; each supported tool must be demonstrated in at least one real-world refactoring pattern or workflow in this reference guide", lang, refFileName, entry.MCPName)
		}
	}
}

func referenceFileNameForLanguage(lang backend.LanguageID) string {
	switch lang {
	case backend.LanguageGo:
		return "golang.md"
	case backend.LanguageMake:
		return "make.md"
	default:
		return string(lang) + ".md"
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root containing go.mod")
		}
		dir = parent
	}
}
