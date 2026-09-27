// This file collects and compares Go diagnostic receipts for semantic batches.

package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/projectverify"
	"semedit/internal/telemetry"
)

type batchVerificationBaseline struct {
	Messages []string
	Modules  map[string]struct{}
}

func collectBatchVerificationBaseline(ctx context.Context, workDir string, plan projectverify.Plan) (batchVerificationBaseline, error) {
	baseline := batchVerificationBaseline{Modules: make(map[string]struct{})}
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
	for _, moduleRoot := range moduleRoots {
		diagnostics, err := pipeline.CheckDiagnosticDetails(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), moduleRoot)
		if err != nil {
			return batchVerificationBaseline{}, fmt.Errorf("batch diagnostics before in %q: %w", moduleRoot, err)
		}
		rel, err := filepath.Rel(workDir, moduleRoot)
		if err != nil {
			return batchVerificationBaseline{}, fmt.Errorf("make Go baseline module root relative: %w", err)
		}
		moduleLabel := filepath.ToSlash(rel)
		for _, diagnostic := range diagnostics {
			baseline.Messages = append(baseline.Messages, "["+moduleLabel+"] "+diagnostic.String())
		}
		baseline.Modules[moduleLabel] = struct{}{}
	}
	return baseline, nil
}

func recordBatchDiagnosticDelta(response *BatchResponse, baseline batchVerificationBaseline, verification operation.VerifyRes) {
	verifiedModules := make(map[string]struct{})
	coverageComplete := len(baseline.Modules) > 0
	for _, coverage := range verification.Coverage {
		if coverage.Language != string(backend.LanguageGo) {
			continue
		}
		if coverage.Status != "verified" || coverage.PackagePattern != "./..." {
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
	case coverageComplete:
		delta := pipeline.ComputeDelta(baseline.Messages, after)
		response.DiagnosticDelta = &delta
	case len(baseline.Modules) == 0:
		response.DiagnosticDeltaNote = "diagnostic delta omitted because the batch has no Go module baseline"
	default:
		response.DiagnosticDeltaNote = "diagnostic delta omitted because post verification did not cover the same Go module roots"
	}
}
