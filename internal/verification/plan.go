// Package verification selects language-server verification from discovered sources and explicit launcher configuration.
// WHY: This file owns backend selection and module-scoped diagnostic coverage for verification plans.
package verification

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

func runPlanLSPVerificationEnabled(ctx context.Context, service *backend.Service, project backend.ProjectContext, request Request, plan projectverify.Plan, scope projectverify.Scope) ([]backend.Diagnostic, []LanguageCoverage, error) {
	if service == nil {
		return nil, nil, fmt.Errorf("verification backend service is nil")
	}
	selection := selectVerificationLanguages(request, project, plan, scope)
	result := languageVerificationResult{Coverage: make([]LanguageCoverage, 0, len(selection.Languages))}
	verified := 0
	for _, language := range selection.Languages {
		languageResult, err := verifyLanguage(ctx, service, project, request, plan, selection, language)
		result.append(languageResult)
		if err != nil {
			return result.Diagnostics, result.Coverage, err
		}
		if languageResult.Verified {
			verified++
		}
	}
	if verified == 0 {
		return result.Diagnostics, result.Coverage, fmt.Errorf("no selected source has a configured verification backend")
	}
	return result.Diagnostics, result.Coverage, nil
}

type languageSelection struct {
	Languages  []string
	ByLanguage map[string][]string
	Explicit   bool
}

type languageVerificationResult struct {
	Diagnostics []backend.Diagnostic
	Coverage    []LanguageCoverage
	Verified    bool
}

func (result *languageVerificationResult) append(next languageVerificationResult) {
	result.Diagnostics = append(result.Diagnostics, next.Diagnostics...)
	result.Coverage = append(result.Coverage, next.Coverage...)
}

func selectVerificationLanguages(request Request, project backend.ProjectContext, plan projectverify.Plan, scope projectverify.Scope) languageSelection {
	selectedLanguage := request.Project.Language
	selectedFile := request.Project.File
	fileSelected := isExistingSourceFile(project.RootDir, selectedFile)
	if !fileSelected && isExistingSourceFile(project.RootDir, request.Path) {
		selectedFile = request.Path
		fileSelected = true
	}
	if (selectedLanguage == "" || selectedLanguage == backend.LanguageAuto) && fileSelected {
		selectedLanguage = backend.LanguageIDFromFile(selectedFile)
	}
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
	explicit := (selectedLanguage != "" && selectedLanguage != backend.LanguageAuto) || fileSelected
	if explicit {
		languages = []string{string(selectedLanguage)}
	}
	return languageSelection{Languages: languages, ByLanguage: byLanguage, Explicit: explicit}
}

func verifyLanguage(ctx context.Context, service *backend.Service, project backend.ProjectContext, request Request, plan projectverify.Plan, selection languageSelection, language string) (languageVerificationResult, error) {
	paths := selection.ByLanguage[language]
	if !selection.Explicit && !autoLSPEnabled(project, plan, language) {
		return languageVerificationResult{Coverage: []LanguageCoverage{{Language: language, Status: "not-configured", Paths: paths, Detail: "No configured language server is selected for this source language."}}}, nil
	}
	if backend.LanguageID(language) == backend.LanguageMake || unsupportedVerificationLanguage(backend.LanguageID(language)) {
		return unsupportedLanguageResult(language, paths), fmt.Errorf("verification is unsupported for language %q", language)
	}
	if backend.LanguageID(language) == backend.LanguageGo {
		return verifyGoLanguage(ctx, service, project, request, language, paths)
	}
	return verifySelectedLanguageFiles(ctx, service, project, language, paths)
}

func unsupportedVerificationLanguage(language backend.LanguageID) bool {
	return language == backend.LanguageRust || language == backend.LanguageScala || language == backend.LanguageHaskell
}

func unsupportedLanguageResult(language string, paths []string) languageVerificationResult {
	detail := "The selected backend does not implement verification."
	if backend.LanguageID(language) == backend.LanguageMake {
		detail = "The Make backend does not implement verification."
	}
	return languageVerificationResult{Coverage: []LanguageCoverage{{Language: language, Status: "unsupported", Paths: paths, Detail: detail}}}
}

func verifyGoLanguage(ctx context.Context, service *backend.Service, project backend.ProjectContext, request Request, language string, paths []string) (languageVerificationResult, error) {
	moduleRoots, err := SelectedGoModuleRoots(project.RootDir, paths)
	if err != nil {
		return languageVerificationResult{}, err
	}
	if len(moduleRoots) == 0 {
		return languageVerificationResult{}, fmt.Errorf("detected Go sources have no verifiable module scope")
	}
	result := languageVerificationResult{Coverage: make([]LanguageCoverage, 0, len(moduleRoots)), Verified: true}
	for _, moduleRoot := range moduleRoots {
		if request.GoPackagePatterns != nil {
			if _, selected := request.GoPackagePatterns[moduleRoot]; !selected {
				continue
			}
		}
		patterns := request.GoPackagePatterns[moduleRoot]
		moduleResult, err := verifyGoModule(ctx, service, project, language, moduleRoot, patterns, request.GoPackageScopeDetail)
		result.append(moduleResult)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func verifyGoModule(ctx context.Context, service *backend.Service, project backend.ProjectContext, language, moduleRoot string, patterns []string, scopeDetail string) (languageVerificationResult, error) {
	selected := project
	selected.RootDir = moduleRoot
	selected.Language = backend.LanguageGo
	selected.File = "."
	patterns = slices.Clone(patterns)
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	checked, err := service.Verify(ctx, backend.VerifyRequest{Project: selected, Path: ".", CheckOnly: true, GoPackagePatterns: patterns})
	if err != nil {
		coverage := LanguageCoverage{Language: language, Status: "failed", ModuleRoot: relativeModuleRoot(project.RootDir, moduleRoot), PackagePattern: strings.Join(patterns, " "), BuildExclusions: goBuildExclusions, Detail: err.Error()}
		result := languageVerificationResult{Coverage: []LanguageCoverage{coverage}}
		return result, fmt.Errorf("verify Go module %q: %w", moduleRoot, err)
	}
	return successfulGoModuleResult(project.RootDir, language, moduleRoot, checked.Diagnostics, patterns, scopeDetail), nil
}

func successfulGoModuleResult(projectRoot, language, moduleRoot string, diagnostics []backend.Diagnostic, patterns []string, scopeDetail string) languageVerificationResult {
	relative := relativeModuleRoot(projectRoot, moduleRoot)
	prefixedDiagnostics := make([]backend.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagnostic.Message = "[" + relative + "] " + diagnostic.Message
		prefixedDiagnostics = append(prefixedDiagnostics, diagnostic)
	}
	pattern := strings.Join(patterns, " ")
	detail := fmt.Sprintf("gopls checked the frozen Go package patterns %s at this module root; build constraints and module boundaries govern coverage. Module-root discovery follows project source-discovery rules.", pattern)
	if scopeDetail != "" {
		detail += " " + scopeDetail
	}
	coverage := LanguageCoverage{
		Language: language, Status: "verified", ModuleRoot: relative, PackagePattern: pattern,
		BuildExclusions: goBuildExclusions,
		Detail:          detail,
	}
	return languageVerificationResult{Diagnostics: prefixedDiagnostics, Coverage: []LanguageCoverage{coverage}, Verified: true}
}

func verifySelectedLanguageFiles(ctx context.Context, service *backend.Service, project backend.ProjectContext, language string, paths []string) (languageVerificationResult, error) {
	if len(paths) == 0 {
		return languageVerificationResult{}, nil
	}
	result := languageVerificationResult{Verified: true}
	for _, path := range paths {
		selected := project
		selected.Language = backend.LanguageID(language)
		selected.File = path
		checked, err := service.Verify(ctx, backend.VerifyRequest{Project: selected, Path: path, CheckOnly: true})
		if err != nil {
			coverage := LanguageCoverage{Language: language, Status: "failed", Paths: paths, Detail: err.Error()}
			result.Coverage = append(result.Coverage, coverage)
			return result, fmt.Errorf("verify %s source %q: %w", language, path, err)
		}
		result.Diagnostics = append(result.Diagnostics, checked.Diagnostics...)
	}
	result.Coverage = append(result.Coverage, LanguageCoverage{Language: language, Status: "verified", Paths: paths, Detail: "Language-server diagnostics completed for the selected source scope."})
	return result, nil
}

func runPlanLSPVerification(ctx context.Context, service *backend.Service, project backend.ProjectContext, request Request, plan projectverify.Plan, scope projectverify.Scope, allowed []backend.LanguageID) ([]backend.Diagnostic, []LanguageCoverage, error) {
	if allowed == nil {
		return runPlanLSPVerificationEnabled(ctx, service, project, request, plan, scope)
	}
	filtered := planForAllowedLanguages(plan, allowed)
	diagnostics, coverage, err := runPlanLSPVerificationEnabled(ctx, service, project, request, filtered, scope)
	seen := make(map[string]bool, len(coverage))
	for _, item := range coverage {
		seen[item.Language] = true
	}
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
	for _, language := range languages {
		if !seen[language] && !slices.Contains(allowed, backend.LanguageID(language)) {
			coverage = append(coverage, LanguageCoverage{Language: language, Status: "disabled", Paths: byLanguage[language], Detail: "Verification is disabled by --enabled-languages."})
		}
	}
	return diagnostics, coverage, err
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
		return project.Java.JDTLSHome != ""
	case backend.LanguageScala:
		return project.Scala.MetalsHome != "" || project.Scala.MetalsBin != ""
	case backend.LanguageKotlin:
		return project.Kotlin.KotlinBin != ""
	case backend.LanguageHaskell:
		return project.Haskell.Standalone && project.Haskell.HLSBin != ""
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

func planForAllowedLanguages(plan projectverify.Plan, allowed []backend.LanguageID) projectverify.Plan {
	if allowed == nil {
		return plan
	}
	filtered := plan
	filtered.Sources = make([]projectverify.Source, 0, len(plan.Sources))
	for _, source := range plan.Sources {
		if slices.Contains(allowed, backend.LanguageID(source.Language)) {
			filtered.Sources = append(filtered.Sources, source)
		}
	}
	filterHooks := func(hooks []projectverify.Hook) []projectverify.Hook {
		result := append([]projectverify.Hook(nil), hooks...)
		for i := range result {
			if result[i].LSP != nil && !slices.Contains(allowed, backend.LanguageID(result[i].LSP.Language)) {
				result[i].Disabled = true
			}
			if len(result[i].Languages) == 0 {
				continue
			}
			languages := make([]string, 0, len(result[i].Languages))
			for _, language := range result[i].Languages {
				if slices.Contains(allowed, backend.LanguageID(language)) {
					languages = append(languages, language)
				}
			}
			if len(languages) == 0 {
				result[i].Disabled = true
			}
			result[i].Languages = languages
		}
		return result
	}
	filtered.Normalize = filterHooks(plan.Normalize)
	filtered.Checks = filterHooks(plan.Checks)
	return filtered
}

func verificationSelectionLanguage(project backend.ProjectContext, path string) (backend.LanguageID, bool) {
	if project.Language != "" && project.Language != backend.LanguageAuto {
		return project.Language, true
	}
	if project.File != "" {
		language := backend.LanguageIDFromFile(project.File)
		info, err := os.Stat(filepath.Join(project.RootDir, filepath.FromSlash(project.File)))
		if language != "" && language != backend.LanguageAuto && (errors.Is(err, os.ErrNotExist) || err == nil && info.Mode().IsRegular()) {
			return language, true
		}
	}
	if isExistingSourceFile(project.RootDir, path) {
		language := backend.LanguageIDFromFile(path)
		if language != "" && language != backend.LanguageAuto {
			return language, true
		}
	}
	return "", false
}
