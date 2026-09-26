// matrix_report.go keeps per-fixture benchmark output separate from matrix execution.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func saveMatrixReports(resultDir, benchDir string, allRuns []*RunResult) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("find repository root: %w", err)
	}
	type benchKey struct {
		taskBase              string
		target                string
		repeat                int
		mcpServerInstructions MCPServerInstructionMode
		semeditArmRestrict    SemeditArmRestriction
	}
	benchGroups := make(map[benchKey][]*RunResult)
	for _, r := range allRuns {
		base := comparisonTaskIdentity(r)
		k := benchKey{taskBase: base, target: r.Target.String(), repeat: r.Repeat, mcpServerInstructions: r.MCPServerInstructions, semeditArmRestrict: r.SemeditArmRestrict}
		benchGroups[k] = append(benchGroups[k], r)
	}

	for k, runs := range benchGroups {
		fixtureFile := resolveFixturePath(benchDir, k.taskBase, "small")
		relFixture, err := filepath.Rel(repoRoot, fixtureFile)
		if err != nil {
			relFixture = fixtureFile
		}
		provenance := ResolveTxtarProvenance(repoRoot, relFixture)

		targetSlug := strings.ReplaceAll(k.target, "/", "-")
		if k.repeat > 0 {
			targetSlug += fmt.Sprintf("-repeat-%d", k.repeat)
		}
		if instructionMode := normalizeMCPServerInstructions(k.mcpServerInstructions); instructionMode != MCPServerInstructionsNone {
			targetSlug += "-mcp-server-instructions-" + string(instructionMode)
		}
		if k.semeditArmRestrict != "" {
			targetSlug += "-semedit-arm-restrict-" + safePathFragment(string(k.semeditArmRestrict))
		}
		taskOutDir := filepath.Join(resultDir, k.taskBase)
		if err := os.MkdirAll(taskOutDir, 0o750); err != nil {
			return fmt.Errorf("create task output directory %s: %w", taskOutDir, err)
		}

		benchComparisons := BuildComparisons(runs)
		for _, comp := range benchComparisons {
			comp.TxtarPath = filepath.ToSlash(relFixture)
			comp.TxtarProvenance = provenance
		}

		singleReport := &BenchmarkReport{
			Timestamp:   time.Now(),
			Runs:        runs,
			Comparisons: benchComparisons,
		}

		jsonFile := filepath.Join(taskOutDir, targetSlug+".json")
		mdFile := filepath.Join(taskOutDir, targetSlug+".md")
		if err := SaveReport(singleReport, jsonFile, mdFile); err != nil {
			return fmt.Errorf("save benchmark %s (%s): %w", k.taskBase, k.target, err)
		}
		fmt.Printf(" Saved benchmark results: %s\n", jsonFile)
	}
	return nil
}
