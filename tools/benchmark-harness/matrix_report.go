// matrix_report.go keeps per-fixture benchmark output separate from matrix execution.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func saveMatrixReports(resultDir, benchDir string, allRuns []*RunResult) {
	repoRoot, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Failed to find repository root: %v\n", err)
		return
	}
	type benchKey struct {
		taskBase              string
		target                string
		repeat                int
		mcpServerInstructions MCPServerInstructionMode
	}
	benchGroups := make(map[benchKey][]*RunResult)
	for _, r := range allRuns {
		base := normalizeTaskBase(r.TaskID)
		k := benchKey{taskBase: base, target: r.Target.String(), repeat: r.Repeat, mcpServerInstructions: r.MCPServerInstructions}
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
		taskOutDir := filepath.Join(resultDir, k.taskBase)
		if err := os.MkdirAll(taskOutDir, 0o750); err != nil {
			fmt.Printf("❌ Failed to create task out dir %s: %v\n", taskOutDir, err)
			continue
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
			fmt.Printf("❌ Failed to save benchmark %s (%s): %v\n", k.taskBase, k.target, err)
		} else {
			fmt.Printf(" Saved benchmark results: %s\n", jsonFile)
		}
	}
}
