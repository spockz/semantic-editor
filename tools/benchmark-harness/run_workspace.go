// run_workspace.go creates exclusive per-execution fixture directories so concurrent jobs never share mutable state.

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func createBenchmarkWorkspace(root, runID string) (string, error) {
	if !filepath.IsLocal(runID) || filepath.Base(runID) != runID || runID == "." {
		return "", fmt.Errorf("benchmark run ID %q must be a local directory name", runID)
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", fmt.Errorf("create benchmark workspace root %q: %w", root, err)
	}
	workDir := filepath.Join(root, runID)
	// #nosec G301,G703 -- caller-generated run IDs stay under the validated fixture root.
	if err := os.Mkdir(workDir, 0o750); err != nil {
		return "", fmt.Errorf("create work dir %q: %w", workDir, err)
	}
	return workDir, nil
}
