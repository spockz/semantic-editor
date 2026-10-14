// Package verification executes configured project hooks and selected language-server diagnostics.
// WHY: This file keeps verification phase ordering, partial reports, and workspace trust at one feature boundary.
package verification

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/projectverify"
)

type execution struct {
	finished    bool
	project     backend.ProjectContext
	path        string
	service     *backend.Service
	request     Request
	plan        projectverify.Plan
	allowedPlan projectverify.Plan
	scope       projectverify.Scope
	response    Result
}

type hookOutcome struct {
	Finished bool
	Failure  string
	Err      error
}

type checkOutcome struct {
	Err error
}

// Run discovers, executes, and reports project verification for one typed request.
func Run(ctx context.Context, request Request) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := prepareExecution(request)
	if err != nil || state.finished {
		return state.response, err
	}
	finished, err := state.normalize(ctx)
	if err != nil || finished {
		return state.response, err
	}
	state.collectDiagnostics(ctx)
	if state.response.Failure != "" {
		return state.response, nil
	}
	check := state.runChecks(ctx)
	return state.response, check.Err
}

func prepareExecution(request Request) (execution, error) {
	state := execution{request: request, project: request.Project, service: serviceForContext(request.Service)}
	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = "."
	}
	canonical, err := canonicalVerifyPath(state.project.RootDir, path)
	if err != nil {
		return state, err
	}
	state.path = canonical
	if state.project.File != "" {
		state.project.File, err = canonicalVerifyPath(state.project.RootDir, state.project.File)
		if err != nil {
			return state, err
		}
	}
	selectedLanguage, explicit := verificationSelectionLanguage(state.project, state.path)
	if explicit && !allowsLanguage(request.AllowedLanguages, selectedLanguage) {
		return state, fmt.Errorf("verification language %q is disabled by --enabled-languages", selectedLanguage)
	}
	if !request.CheckOnly && request.AllowedLanguages != nil && (request.FormatSelectedFile || request.OrganizeImports) && !explicit {
		state.response.Failure = "select an enabled source language or file before applying selected-file normalization"
		state.finished = true
		return state, nil
	}
	if state.path != "." {
		if _, statErr := os.Stat(filepath.Join(state.project.RootDir, filepath.FromSlash(state.path))); statErr != nil {
			return state, fmt.Errorf("verification path %q: %w", state.path, statErr)
		}
	}
	plan, err := projectverify.Discover(state.project.RootDir)
	if err != nil {
		return state, fmt.Errorf("discover project verification plan: %w", err)
	}
	state.plan = plan
	state.allowedPlan = planForAllowedLanguages(plan, request.AllowedLanguages)
	if state.path != "." {
		state.scope.Paths = []string{state.path}
	}
	actualPaths := scopedSourcePaths(plan, state.scope)
	if len(actualPaths) == 0 && state.path != "." {
		actualPaths = []string{state.path}
	}
	state.response.Scope = projectverify.Scope{Paths: actualPaths}
	return state, nil
}

func (state *execution) normalize(ctx context.Context) (bool, error) {
	if state.request.CheckOnly {
		return false, nil
	}
	if err := state.formatGoSources(ctx); err != nil {
		return false, err
	}
	hooks := state.runNormalizeHooks(ctx)
	if hooks.Err != nil {
		return false, hooks.Err
	}
	if hooks.Finished {
		state.response.Failure = hooks.Failure
		return true, nil
	}
	if state.request.FormatSelectedFile || state.request.OrganizeImports {
		if state.normalizeSelectedFile(ctx) {
			return true, nil
		}
	}
	if state.rediscoverAfterNormalization() {
		return true, nil
	}
	return false, nil
}

func (state *execution) formatGoSources(ctx context.Context) error {
	paths := state.request.GoFormattingPaths
	if paths == nil {
		paths = []string{state.path}
	}
	goPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if allowsLanguage(state.request.AllowedLanguages, backend.LanguageGo) && shouldRunGofmt(state.project, path) {
			goPaths = append(goPaths, path)
		}
	}
	if len(goPaths) == 0 {
		return nil
	}
	needsFormatting, err := gofmtListsFiles(ctx, state.project.RootDir, goPaths...)
	if err != nil {
		return fmt.Errorf("check Go formatting before verification: %w", err)
	}
	if !needsFormatting {
		return nil
	}
	if err := pipeline.Format(ctx, state.project.RootDir, goPaths...); err != nil {
		return fmt.Errorf("format Go files before verification: %w", err)
	}
	state.response.Formatted = true
	return nil
}

func (state *execution) runNormalizeHooks(ctx context.Context) hookOutcome {
	hookID, err := workspaceTrustRequiredHook(state.allowedPlan, state.allowedPlan.Normalize, state.scope)
	if err != nil {
		return hookOutcome{Err: err}
	}
	if hookID != "" && !state.project.WorkspaceTrust.Allows(state.project.RootDir) {
		return hookOutcome{Finished: true, Failure: fmt.Sprintf("workspace trust is required to run verification hook %q", hookID)}
	}
	plan := state.allowedPlan
	if state.request.SkipHooks {
		plan = planWithoutExternalCommands(plan)
	}
	report, err := projectverify.RunPhase(ctx, plan, projectverify.PhaseNormalize, state.scope,
		verificationLSPRunner(state.service, state.project, projectverify.PhaseNormalize, state.request.AllowedLanguages))
	state.response.NormalizeReport = report
	if err != nil {
		return hookOutcome{Finished: true, Failure: err.Error()}
	}
	return hookOutcome{}
}

func (state *execution) normalizeSelectedFile(ctx context.Context) bool {
	_, err := state.service.Verify(ctx, backend.VerifyRequest{
		Project:            state.project,
		Path:               state.path,
		FormatSelectedFile: state.request.FormatSelectedFile,
		OrganizeImports:    state.request.OrganizeImports,
	})
	if err != nil {
		state.response.Failure = fmt.Errorf("apply selected backend normalization: %w", err).Error()
		return true
	}
	return false
}

func (state *execution) rediscoverAfterNormalization() bool {
	plan, err := projectverify.Discover(state.project.RootDir)
	if err != nil {
		state.response.Failure = fmt.Errorf("rediscover project verification plan: %w", err).Error()
		return true
	}
	state.plan = plan
	state.allowedPlan = planForAllowedLanguages(plan, state.request.AllowedLanguages)
	state.response.Scope.Paths = scopedSourcePaths(plan, state.scope)
	if len(state.response.Scope.Paths) == 0 && len(state.scope.Paths) > 0 {
		state.response.Scope = state.scope
	}
	return false
}

func (state *execution) collectDiagnostics(ctx context.Context) {
	diagnostics, coverage, err := runPlanLSPVerification(ctx, state.service, state.project, state.request, state.plan, state.scope, state.request.AllowedLanguages)
	state.response.Diagnostics = diagnostics
	state.response.Coverage = coverage
	if err != nil {
		state.response.Failure = err.Error()
	}
}

func (state *execution) runChecks(ctx context.Context) checkOutcome {
	hookID, err := workspaceTrustRequiredHook(state.allowedPlan, state.allowedPlan.Checks, state.scope)
	if err != nil {
		return checkOutcome{Err: err}
	}
	if hookID != "" && !state.project.WorkspaceTrust.Allows(state.project.RootDir) {
		state.response.Failure = fmt.Sprintf("workspace trust is required to run verification hook %q", hookID)
		return checkOutcome{}
	}
	plan := state.allowedPlan
	if state.request.SkipHooks {
		plan = planWithoutExternalCommands(plan)
	}
	state.response.CheckReport, err = projectverify.RunPhase(ctx, plan, projectverify.PhaseCheck, state.scope,
		verificationLSPRunner(state.service, state.project, projectverify.PhaseCheck, state.request.AllowedLanguages))
	if err != nil {
		state.response.Failure = err.Error()
	}
	return checkOutcome{}
}

func allowsLanguage(allowed []backend.LanguageID, language backend.LanguageID) bool {
	return allowed == nil || slices.Contains(allowed, language)
}

func planWithoutExternalCommands(plan projectverify.Plan) projectverify.Plan {
	plan.Normalize = slices.Clone(plan.Normalize)
	plan.Checks = slices.Clone(plan.Checks)
	for index := range plan.Normalize {
		hook := &plan.Normalize[index]
		if hook.LSP == nil && (hook.Shell != nil || len(hook.Exec) > 0) {
			hook.Disabled = true
		}
	}
	for index := range plan.Checks {
		hook := &plan.Checks[index]
		if hook.LSP == nil && (hook.Shell != nil || len(hook.Exec) > 0) {
			hook.Disabled = true
		}
	}
	return plan
}

func shouldRunGofmt(project backend.ProjectContext, path string) bool {
	language := project.Language
	if language != "" && language != backend.LanguageAuto {
		return language == backend.LanguageGo
	}
	language = backend.LanguageIDFromFile(path)
	if language != "" && language != backend.LanguageAuto {
		return language == backend.LanguageGo
	}
	info, err := os.Stat(filepath.Join(project.RootDir, filepath.FromSlash(path)))
	return err == nil && info.IsDir()
}

func canonicalVerifyPath(root, requested string) (string, error) {
	if !filepath.IsAbs(requested) {
		clean := filepath.Clean(requested)
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("verification path %q is outside workspace %q", requested, pathutil.CanonicalWorkspaceRoot(root))
		}
		requested = filepath.Join(root, clean)
	}
	canonicalRoot := pathutil.CanonicalWorkspaceRoot(root)
	canonicalPath := pathutil.CanonicalWorkspaceRoot(requested)
	if !pathutil.PathWithin(canonicalRoot, canonicalPath) {
		return "", fmt.Errorf("verification path %q is outside workspace %q", requested, canonicalRoot)
	}
	rel, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil {
		return "", fmt.Errorf("resolve verification path %q relative to workspace: %w", requested, err)
	}
	return filepath.ToSlash(rel), nil
}

func scopedSourcePaths(plan projectverify.Plan, scope projectverify.Scope) []string {
	paths := make([]string, 0)
	for _, source := range plan.Sources {
		if sourceInScope(source.Path, scope) {
			paths = append(paths, source.Path)
		}
	}
	return paths
}

func gofmtListsFiles(ctx context.Context, root string, paths ...string) (bool, error) {
	formatted, err := pipeline.NeedsFormatting(ctx, root, paths...)
	if err != nil {
		return false, fmt.Errorf("gofmt -l %s: %w", strings.Join(paths, ", "), err)
	}
	return formatted, nil
}
