// Package operation selects language-server verification from discovered sources and explicit launcher configuration.
package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/projectverify"
)

// LanguageCoverage reports the verification status and source receipt for one language.
type LanguageCoverage struct {
	Language        string   `json:"language"`
	Status          string   `json:"status"`
	Paths           []string `json:"paths,omitempty"`
	ModuleRoot      string   `json:"module_root,omitempty"`
	PackagePattern  string   `json:"package_pattern,omitempty"`
	BuildExclusions []string `json:"build_exclusions,omitempty"`
	Detail          string   `json:"detail,omitempty"`
}

var goBuildExclusions = []string{"Go build context and build tags", "testdata", "build", "dist", "target", "vendor", "node_modules", ".scratch", ".git", "bazel-*", "_*", ".*"}

func runPlanLSPVerification(ctx context.Context, service *backend.Service, project backend.ProjectContext, request VerifyReq, plan projectverify.Plan, scope projectverify.Scope) ([]backend.Diagnostic, []LanguageCoverage, error) {
	if service == nil {
		return nil, nil, fmt.Errorf("verification backend service is nil")
	}
	selectedLanguage := request.Project.Language
	selectedFile := request.Project.File
	fileSelected := isExistingSourceFile(project.RootDir, selectedFile)
	if !fileSelected && isExistingSourceFile(project.RootDir, request.Path) {
		selectedFile = request.Path
		fileSelected = true
	}
	if selectedLanguage == "" || selectedLanguage == backend.LanguageAuto {
		if fileSelected {
			selectedLanguage = backend.LanguageIDFromFile(selectedFile)
		}
	}
	explicit := (selectedLanguage != "" && selectedLanguage != backend.LanguageAuto) || fileSelected
	byLanguage := make(map[string][]string)
	for _, source := range plan.Sources {
		if sourceInScope(source.Path, scope) {
			byLanguage[source.Language] = append(byLanguage[source.Language], source.Path)
		}
	}
	languages := make([]string, 0, len(byLanguage))
	for language := range byLanguage {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	if explicit {
		languages = []string{string(selectedLanguage)}
	}
	var diagnostics []backend.Diagnostic
	coverage := make([]LanguageCoverage, 0, len(languages))
	verified := 0
	for _, language := range languages {
		paths := byLanguage[language]
		if !explicit && !autoLSPEnabled(project, plan, language) {
			coverage = append(coverage, LanguageCoverage{Language: language, Status: "not-configured", Paths: paths, Detail: "No configured language server is selected for this source language."})
			continue
		}
		if backend.LanguageID(language) == backend.LanguageMake {
			coverage = append(coverage, LanguageCoverage{Language: language, Status: "unsupported", Paths: paths, Detail: "The Make backend does not implement verification."})
			return diagnostics, coverage, fmt.Errorf("verification is unsupported for language %q", language)
		}
		if backend.LanguageID(language) == backend.LanguageRust || backend.LanguageID(language) == backend.LanguageScala || backend.LanguageID(language) == backend.LanguageHaskell {
			coverage = append(coverage, LanguageCoverage{Language: language, Status: "unsupported", Paths: paths, Detail: "The selected backend does not implement verification."})
			return diagnostics, coverage, fmt.Errorf("verification is unsupported for language %q", language)
		}
		if backend.LanguageID(language) == backend.LanguageGo {
			moduleRoots, err := SelectedGoModuleRoots(project.RootDir, paths)
			if err != nil {
				return diagnostics, coverage, err
			}
			if len(moduleRoots) == 0 {
				return diagnostics, coverage, fmt.Errorf("detected Go sources have no verifiable module scope")
			}
			for _, moduleRoot := range moduleRoots {
				selected := project
				selected.RootDir = moduleRoot
				selected.Language = backend.LanguageGo
				selected.File = "."
				result, err := service.Verify(ctx, backend.VerifyRequest{Project: selected, Path: ".", CheckOnly: true})
				if err != nil {
					coverage = append(coverage, LanguageCoverage{Language: language, Status: "failed", ModuleRoot: relativeModuleRoot(project.RootDir, moduleRoot), PackagePattern: "./...", BuildExclusions: goBuildExclusions, Detail: err.Error()})
					return diagnostics, coverage, fmt.Errorf("verify Go module %q: %w", moduleRoot, err)
				}
				for _, diagnostic := range result.Diagnostics {
					diagnostic.Message = "[" + relativeModuleRoot(project.RootDir, moduleRoot) + "] " + diagnostic.Message
					diagnostics = append(diagnostics, diagnostic)
				}
				coverage = append(coverage, LanguageCoverage{Language: language, Status: "verified", ModuleRoot: relativeModuleRoot(project.RootDir, moduleRoot), PackagePattern: "./...", BuildExclusions: goBuildExclusions, Detail: "gopls checked the effective module package pattern; active build tags, build constraints, and listed excluded directories govern selected files."})
			}
			verified++
			continue
		}
		if len(paths) == 0 {
			continue
		}
		for _, path := range paths {
			selected := project
			selected.Language = backend.LanguageID(language)
			selected.File = path
			result, err := service.Verify(ctx, backend.VerifyRequest{Project: selected, Path: path, CheckOnly: true})
			if err != nil {
				coverage = append(coverage, LanguageCoverage{Language: language, Status: "failed", Paths: paths, Detail: err.Error()})
				return diagnostics, coverage, fmt.Errorf("verify %s source %q: %w", language, path, err)
			}
			diagnostics = append(diagnostics, result.Diagnostics...)
		}
		verified++
		coverage = append(coverage, LanguageCoverage{Language: language, Status: "verified", Paths: paths, Detail: "Language-server diagnostics completed for the selected source scope."})
	}
	if verified == 0 {
		return diagnostics, coverage, fmt.Errorf("no selected source has a configured verification backend")
	}
	return diagnostics, coverage, nil
}

// SelectedGoModuleRoots returns the effective Go module roots for selected source paths.
func SelectedGoModuleRoots(projectRoot string, selectedPaths []string) ([]string, error) {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve Go source root %q: %w", projectRoot, err)
	}
	modules := make(map[string]struct{})
	for _, source := range selectedPaths {
		absolute := source
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(root, filepath.FromSlash(absolute))
		}
		absolute, err = filepath.Abs(absolute)
		if err != nil {
			return nil, fmt.Errorf("resolve selected Go source %q: %w", source, err)
		}
		dir := filepath.Dir(absolute)
		found := ""
		for {
			info, statErr := os.Stat(filepath.Join(dir, "go.mod"))
			if statErr == nil {
				if !info.Mode().IsRegular() {
					return nil, fmt.Errorf("go module manifest %q is not a regular file", filepath.Join(dir, "go.mod"))
				}
				found = dir
				break
			}
			if !errors.Is(statErr, os.ErrNotExist) {
				return nil, fmt.Errorf("inspect Go module manifest in %q: %w", dir, statErr)
			}
			if dir == root || filepath.Dir(dir) == dir {
				break
			}
			dir = filepath.Dir(dir)
		}
		if found == "" {
			found = root
		}
		modules[filepath.Clean(found)] = struct{}{}
	}
	roots := make([]string, 0, len(modules))
	for module := range modules {
		roots = append(roots, module)
	}
	slices.Sort(roots)
	return roots, nil
}

func relativeModuleRoot(projectRoot, moduleRoot string) string {
	rel, err := filepath.Rel(projectRoot, moduleRoot)
	if err != nil || rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func isExistingSourceFile(root, path string) bool {
	if path == "" {
		return false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	base := filepath.Base(path)
	return filepath.Ext(path) != "" || strings.EqualFold(base, "Makefile") || strings.EqualFold(base, "GNUmakefile") || strings.EqualFold(base, "makefile")
}

func sourceInScope(path string, scope projectverify.Scope) bool {
	if len(scope.Paths) == 0 {
		return true
	}
	path = filepath.ToSlash(filepath.Clean(path))
	for _, candidate := range scope.Paths {
		candidate = filepath.ToSlash(filepath.Clean(candidate))
		if candidate == "." || path == candidate || strings.HasPrefix(path, strings.TrimSuffix(candidate, "/")+"/") {
			return true
		}
	}
	return false
}

func autoLSPEnabled(project backend.ProjectContext, plan projectverify.Plan, language string) bool {
	if language == string(backend.LanguageGo) {
		return true
	}
	for _, hooks := range [][]projectverify.Hook{plan.Normalize, plan.Checks} {
		for _, hook := range hooks {
			if !hook.Disabled && hook.LSP != nil && hook.LSP.Language == language {
				return true
			}
		}
	}
	switch backend.LanguageID(language) {
	case backend.LanguageJava:
		return project.Java.JDTLSHome != "" || project.JDTLSHome != ""
	case backend.LanguageScala:
		return project.Scala.MetalsHome != "" || project.Scala.MetalsBin != "" || project.MetalsHome != "" || project.MetalsBin != ""
	case backend.LanguageKotlin:
		return project.Kotlin.KotlinBin != "" || project.KotlinBin != ""
	case backend.LanguageHaskell:
		return project.Haskell.Standalone && (project.Haskell.HLSBin != "" || project.HaskellStandalone)
	case backend.LanguageBash:
		if strings.TrimSpace(project.Bash.BashBin) != "" {
			return true
		}
		return executableAvailable("", "bash-language-server")
	case backend.LanguageMake:
		if strings.TrimSpace(project.Make.MakeBin) != "" {
			return true
		}
		return executableAvailable("", "make-ls")
	default:
		return false
	}
}

func executableAvailable(configured, fallback string) bool {
	if strings.TrimSpace(configured) != "" {
		_, err := exec.LookPath(configured)
		return err == nil
	}
	_, err := exec.LookPath(fallback)
	return err == nil
}
