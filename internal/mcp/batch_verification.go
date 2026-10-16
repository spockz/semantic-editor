// This file collects and compares Go diagnostic receipts for semantic batches.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/projectverify"
	"github.com/spockz/semantic-editor/internal/telemetry"

	"go/build/constraint"
)

type batchVerificationBaseline struct {
	Messages          []string
	GoErrorCount      int
	Modules           map[string]struct{}
	ModuleRoots       []string
	GoPatterns        map[string][]string
	PatternsByLabel   map[string][]string
	ScaffoldGoTargets map[string]string
	OmittedReason     string
	ScopeReason       string
}

func collectBatchVerificationBaseline(ctx context.Context, workDir string, plan projectverify.Plan, allowedLanguages []backend.LanguageID, edits []BatchEntry) (batchVerificationBaseline, error) {
	baseline := batchVerificationBaseline{Modules: make(map[string]struct{}), GoPatterns: make(map[string][]string), PatternsByLabel: make(map[string][]string)}
	if allowedLanguages != nil && !slices.Contains(allowedLanguages, backend.LanguageGo) {
		baseline.OmittedReason = "diagnostic delta omitted because Go verification is disabled by --enabled-languages"
		return baseline, nil
	}
	var goSources []string
	for _, source := range plan.Sources {
		if source.Language == string(backend.LanguageGo) {
			goSources = append(goSources, source.Path)
		}
	}
	if len(goSources) == 0 {
		return baseline, nil
	}
	moduleRoots, err := operation.SelectedGoModuleRoots(workDir, goSources)
	if err != nil {
		return batchVerificationBaseline{}, fmt.Errorf("select batch Go module baselines: %w", err)
	}
	baseline.ModuleRoots = slices.Clone(moduleRoots)
	baseline.ScaffoldGoTargets = scaffoldGoTargets(workDir, edits, moduleRoots)
	baseline.GoPatterns, baseline.ScopeReason = selectBatchGoPatterns(ctx, workDir, edits, moduleRoots)
	if hasExternalNormalizer(plan) && !batchSkipsHooks(ctx) {
		baseline.GoPatterns = workspaceGoPatterns(moduleRoots)
		baseline.ScopeReason = "configured external normalization hooks may change packages; verification used workspace-wide scope"
	}
	for _, moduleRoot := range moduleRoots {
		patterns := baseline.GoPatterns[moduleRoot]
		if patterns == nil {
			continue
		}
		diagnosticCtx := godiagnostics.WithIncrementalDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore))
		diagnostics, err := pipeline.CheckDiagnosticDetailsForPatterns(diagnosticCtx, moduleRoot, patterns...)
		if err != nil {
			return batchVerificationBaseline{}, fmt.Errorf("batch diagnostics before in %q: %w", moduleRoot, err)
		}
		rel, err := filepath.Rel(workDir, moduleRoot)
		if err != nil {
			return batchVerificationBaseline{}, fmt.Errorf("make Go baseline module root relative: %w", err)
		}
		moduleLabel := filepath.ToSlash(rel)
		baseline.PatternsByLabel[moduleLabel] = slices.Clone(patterns)
		for _, diagnostic := range diagnostics {
			baseline.Messages = append(baseline.Messages, "["+moduleLabel+"] "+diagnostic.String())
			if diagnostic.Severity == 1 {
				baseline.GoErrorCount++
			}
		}
		baseline.Modules[moduleLabel] = struct{}{}
	}
	return baseline, nil
}

func scaffoldGoTargets(workDir string, edits []BatchEntry, moduleRoots []string) map[string]string {
	targets := make(map[string]string)
	for _, edit := range edits {
		if edit.Tool != "semantic_scaffold_file" {
			continue
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(edit.Params, &values); err != nil {
			continue
		}
		var file string
		if raw, ok := values["file"]; !ok || json.Unmarshal(raw, &file) != nil || filepath.Ext(file) != ".go" {
			continue
		}
		target, err := canonicalBatchTarget(workDir, file)
		if err != nil {
			continue
		}
		if moduleRoot := assignedBatchModule(target, moduleRoots); moduleRoot != "" {
			targets[target] = moduleRoot
		}
	}
	return targets
}

func hasNetNewGoErrors(baseline batchVerificationBaseline, verification operation.VerifyRes) bool {
	finalErrorCount := 0
	for _, diagnostic := range verification.Diagnostics {
		if isGoDiagnostic(diagnostic) && diagnostic.Severity == 1 {
			finalErrorCount++
		}
	}
	return finalErrorCount > baseline.GoErrorCount
}

func recordBatchDiagnosticDelta(response *BatchResponse, baseline batchVerificationBaseline, verification operation.VerifyRes) {
	verifiedModules := make(map[string]struct{})
	coverageComplete := len(baseline.Modules) > 0
	for _, coverage := range verification.Coverage {
		if coverage.Language != string(backend.LanguageGo) {
			continue
		}
		if coverage.Status != "verified" {
			coverageComplete = false
		}
		if expected := baseline.PatternsByLabel[coverage.ModuleRoot]; expected != nil && coverage.PackagePattern != strings.Join(expected, " ") {
			coverageComplete = false
		}
		verifiedModules[coverage.ModuleRoot] = struct{}{}
	}
	if len(verifiedModules) != len(baseline.Modules) {
		coverageComplete = false
	}
	for module := range baseline.Modules {
		if _, ok := verifiedModules[module]; !ok {
			coverageComplete = false
		}
	}
	var after []string
	if coverageComplete {
		after = make([]string, 0, len(verification.Diagnostics))
		for _, diagnostic := range verification.Diagnostics {
			if isGoDiagnostic(diagnostic) {
				after = append(after, diagnostic.Message)
			}
		}
	}
	switch {
	case baseline.OmittedReason != "":
		response.DiagnosticDeltaNote = baseline.OmittedReason
	case coverageComplete:
		delta := pipeline.ComputeDelta(baseline.Messages, after)
		response.DiagnosticDelta = &delta
	case len(baseline.Modules) == 0:
		response.DiagnosticDeltaNote = "diagnostic delta omitted because the batch has no Go module baseline"
	default:
		response.DiagnosticDeltaNote = "diagnostic delta omitted because post verification did not cover the same Go module roots"
	}
}

func hasExternalNormalizer(plan projectverify.Plan) bool {
	for _, hook := range plan.Normalize {
		if !hook.Disabled && (hook.Shell != nil || len(hook.Exec) > 0) {
			return true
		}
	}
	return false
}

func batchSkipsHooks(ctx context.Context) bool {
	skipHooks, _ := ctx.Value(batchSkipHooksKey{}).(bool)
	return skipHooks
}

func selectBatchGoPatterns(ctx context.Context, workDir string, edits []BatchEntry, moduleRoots []string) (map[string][]string, string) {
	targetsByModule, reason := collectBatchGoTargets(workDir, edits, moduleRoots)
	if reason != "" {
		return workspaceGoPatterns(moduleRoots), reason
	}
	patternsByModule := make(map[string][]string, len(targetsByModule))
	moduleLabels := make([]string, 0, len(targetsByModule))
	for _, moduleRoot := range moduleRoots {
		targets := targetsByModule[moduleRoot]
		if len(targets) == 0 {
			continue
		}
		patterns, err := godiagnostics.AffectedPackagePatternsWithContext(ctx, moduleRoot, targets)
		if err != nil {
			return workspaceGoPatterns(moduleRoots), "Go package graph selection failed; verification used workspace-wide scope"
		}
		patternsByModule[moduleRoot] = patterns
		relative, relErr := filepath.Rel(workDir, moduleRoot)
		if relErr != nil {
			return workspaceGoPatterns(moduleRoots), "an edited Go module root could not be made relative; verification used workspace-wide scope"
		}
		moduleLabels = append(moduleLabels, filepath.ToSlash(relative))
	}
	if len(patternsByModule) == 0 {
		return workspaceGoPatterns(moduleRoots), "batch Go targets could not be assigned to an edited module; verification used workspace-wide scope"
	}
	slices.Sort(moduleLabels)
	detail := "verification used frozen Go package closures for edited module roots: " + strings.Join(moduleLabels, ", ")
	for _, patterns := range patternsByModule {
		if slices.Equal(patterns, []string{"./..."}) {
			return patternsByModule, "cached Go package scope was unavailable; verification used workspace-wide scope for edited module roots: " + strings.Join(moduleLabels, ", ")
		}
	}
	return patternsByModule, detail
}

func workspaceGoPatterns(moduleRoots []string) map[string][]string {
	patterns := make(map[string][]string, len(moduleRoots))
	for _, root := range moduleRoots {
		patterns[root] = []string{"./..."}
	}
	return patterns
}

func collectBatchGoTargets(workDir string, edits []BatchEntry, moduleRoots []string) (map[string][]string, string) {
	targetsByModule := make(map[string][]string)
	for _, edit := range edits {
		entryTargets, err := batchGoTargetsForEntry(workDir, edit)
		if err != nil || len(entryTargets) == 0 {
			return nil, "batch targets could not be mapped to Go packages"
		}
		for _, target := range entryTargets {
			moduleRoot := assignedBatchModule(target, moduleRoots)
			if moduleRoot == "" {
				return nil, "a Go target could not be assigned to a discovered module; verification fell back to workspace-wide scope"
			}
			targetsByModule[moduleRoot] = append(targetsByModule[moduleRoot], target)
		}
	}
	return targetsByModule, ""
}

func batchGoTargetsForEntry(workDir string, edit BatchEntry) ([]string, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(edit.Params, &values); err != nil {
		return nil, err
	}
	var targets []string
	for _, key := range []string{"file", "path", "origin_file", "destination_file", "source_file"} {
		var value string
		if raw, ok := values[key]; !ok || json.Unmarshal(raw, &value) != nil || value == "" || filepath.Ext(value) != ".go" {
			continue
		}
		target, err := canonicalBatchTarget(workDir, value)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func canonicalBatchTarget(workDir, target string) (string, error) {
	if !filepath.IsAbs(target) {
		target = filepath.Join(workDir, filepath.FromSlash(target))
	}
	absolute, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", err
	}
	probe := absolute
	var missing []string
	for {
		canonical, linkErr := filepath.EvalSymlinks(probe)
		if linkErr == nil {
			for _, m := range slices.Backward(missing) {
				canonical = filepath.Join(canonical, m)
			}
			return canonical, nil
		}
		if !os.IsNotExist(linkErr) {
			return "", linkErr
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", linkErr
		}
		missing = append(missing, filepath.Base(probe))
		probe = parent
	}
}

func pathWithinDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func reconcilePublishedGoScope(ctx context.Context, workDir string, before map[string]string, baseline *batchVerificationBaseline) {
	after, err := snapshotBatchWorkspace(ctx, workDir)
	if err != nil {
		fallbackBatchVerificationScope(baseline, "published workspace scope could not be reconciled")
		return
	}
	paths := make(map[string]struct{}, len(before)+len(after))
	for path := range before {
		paths[path] = struct{}{}
	}
	for path := range after {
		paths[path] = struct{}{}
	}
	for path := range paths {
		previous, current := before[path], after[path]
		if previous == current {
			continue
		}
		if strings.HasSuffix(path, "go.mod") || strings.HasSuffix(path, "go.sum") || filepath.Base(path) == "go.work" || filepath.Base(path) == "go.work.sum" {
			fallbackBatchVerificationScope(baseline, "Go build inputs changed after the baseline")
			return
		}
		if filepath.Ext(path) != ".go" {
			continue
		}
		moduleRoot := assignedBatchModule(filepath.Join(workDir, filepath.FromSlash(path)), baseline.ModuleRoots)
		if moduleRoot == "" {
			fallbackBatchVerificationScope(baseline, "a changed Go file was outside the frozen module set")
			return
		}
		if previous == "" && current != "" && baseline.isExpectedScaffoldGoTarget(workDir, path, moduleRoot) {
			continue
		}
		if hasChangedGoHeader(previous, current) {
			fallbackBatchVerificationScope(baseline, "a Go package, import, or build-constraint input changed after the baseline")
			return
		}
		rel, relErr := filepath.Rel(moduleRoot, filepath.Join(workDir, filepath.FromSlash(path)))
		if relErr != nil || !containsBatchPattern(baseline.GoPatterns[moduleRoot], filepath.ToSlash(filepath.Dir(rel))) {
			fallbackBatchVerificationScope(baseline, "a changed Go package was outside the frozen package scope")
			return
		}
	}
}

func (baseline batchVerificationBaseline) isExpectedScaffoldGoTarget(workDir, path, moduleRoot string) bool {
	target, err := canonicalBatchTarget(workDir, filepath.FromSlash(path))
	if err != nil || baseline.ScaffoldGoTargets[target] != moduleRoot {
		return false
	}
	return slices.Equal(baseline.GoPatterns[moduleRoot], []string{"./..."})
}

func fallbackBatchVerificationScope(baseline *batchVerificationBaseline, reason string) {
	baseline.GoPatterns = workspaceGoPatterns(baseline.ModuleRoots)
	baseline.OmittedReason = "diagnostic delta omitted because " + reason
	baseline.ScopeReason = baseline.OmittedReason + "; verification used workspace-wide scope"
}

func assignedBatchModule(path string, roots []string) string {
	ordered := slices.Clone(roots)
	slices.SortFunc(ordered, func(left, right string) int { return len(right) - len(left) })
	for _, root := range ordered {
		if pathWithinDir(root, path) {
			return root
		}
	}
	return ""
}

func containsBatchPattern(patterns []string, relDir string) bool {
	for _, pattern := range patterns {
		if pattern == "./..." || pattern == "." && relDir == "." || pattern == "./"+relDir {
			return true
		}
	}
	return false
}

func hasChangedGoHeader(before, after string) bool {
	if before == "" || after == "" {
		return true
	}
	beforeMetadata, beforeErr := goHeaderMetadata(before)
	afterMetadata, afterErr := goHeaderMetadata(after)
	return beforeErr != nil || afterErr != nil || beforeMetadata != afterMetadata
}

func goHeaderMetadata(source string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "batch-scope.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", err
	}
	metadata := []string{file.Name.Name}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if constraint.IsGoBuild(comment.Text) || constraint.IsPlusBuild(comment.Text) {
				metadata = append(metadata, strings.TrimSpace(comment.Text))
			}
		}
	}
	imports := make([]string, 0, len(file.Imports))
	for _, item := range file.Imports {
		path, unquoteErr := strconv.Unquote(item.Path.Value)
		if unquoteErr != nil {
			return "", unquoteErr
		}
		imports = append(imports, path)
	}
	slices.Sort(imports)
	metadata = append(metadata, imports...)
	return strings.Join(metadata, "\n"), nil
}
