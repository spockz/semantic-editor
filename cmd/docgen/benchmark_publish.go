// Package main applies publication filters and deterministic report ordering.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func sortBenchmarkComparisons(comparisons []*BenchComparisonSummary) {
	sort.Slice(comparisons, func(i, j int) bool {
		if comparisons[i].TaskID != comparisons[j].TaskID {
			return comparisons[i].TaskID < comparisons[j].TaskID
		}
		if comparisons[i].Target.String() != comparisons[j].Target.String() {
			return comparisons[i].Target.String() < comparisons[j].Target.String()
		}
		if comparisons[i].PromptVariant != comparisons[j].PromptVariant {
			return comparisons[i].PromptVariant < comparisons[j].PromptVariant
		}
		return displayMCPServerInstructions(comparisons[i].MCPServerInstructions, comparisons[i].LegacyMCPInstructions) <
			displayMCPServerInstructions(comparisons[j].MCPServerInstructions, comparisons[j].LegacyMCPInstructions)
	})
}

func filterPublishableBenchmarkComparison(comp *BenchComparisonSummary) *BenchComparisonSummary {
	if comp == nil {
		return nil
	}

	filtered := *comp
	filtered.SmallBaseline = publishableBenchmarkRun(comp.SmallBaseline)
	filtered.SmallSemedit = publishableBenchmarkRun(comp.SmallSemedit)
	filtered.SmallVerifiedBaseline = publishableBenchmarkRun(comp.SmallVerifiedBaseline)
	filtered.SmallVerifiedSemedit = publishableBenchmarkRun(comp.SmallVerifiedSemedit)
	filtered.LargeBaseline = publishableBenchmarkRun(comp.LargeBaseline)
	filtered.LargeSemedit = publishableBenchmarkRun(comp.LargeSemedit)
	filtered.LargeVerifiedBaseline = publishableBenchmarkRun(comp.LargeVerifiedBaseline)
	filtered.LargeVerifiedSemedit = publishableBenchmarkRun(comp.LargeVerifiedSemedit)

	if filtered.SmallBaseline == nil && filtered.SmallSemedit == nil &&
		filtered.SmallVerifiedBaseline == nil && filtered.SmallVerifiedSemedit == nil &&
		filtered.LargeBaseline == nil && filtered.LargeSemedit == nil &&
		filtered.LargeVerifiedBaseline == nil && filtered.LargeVerifiedSemedit == nil {
		return nil
	}
	return &filtered
}

func publishableBenchmarkRun(run *BenchRunResult) *BenchRunResult {
	if run == nil || run.Error != "" || run.Oracle == nil {
		return nil
	}
	return run
}

func resolveTxtarProvenance(repoRoot, txtarRelPath string) string {
	fullPath := filepath.Join(repoRoot, filepath.FromSlash(txtarRelPath))
	// #nosec G304 -- reading benchmark fixture for provenance checksum
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	hashShort := fmt.Sprintf("sha256:%.12x (uncommitted)", hash)

	ctx := context.Background()
	// #nosec G204 -- inspecting git status of txtar fixture
	cmdStatus := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", txtarRelPath)
	cmdStatus.Dir = repoRoot
	outStatus, err := cmdStatus.Output()
	if err == nil && len(bytes.TrimSpace(outStatus)) == 0 {
		// #nosec G204 -- retrieving git commit hash of txtar fixture
		cmdLog := exec.CommandContext(ctx, "git", "log", "-n", "1", "--pretty=format:%H", "--", txtarRelPath)
		cmdLog.Dir = repoRoot
		outLog, err := cmdLog.Output()
		sha := strings.TrimSpace(string(outLog))
		if err == nil && len(sha) == 40 {
			return fmt.Sprintf("https://github.com/spockz/semantic-editor/blob/%s/%s", sha, filepath.ToSlash(txtarRelPath))
		}
	}

	return hashShort
}
