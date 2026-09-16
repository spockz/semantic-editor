// Package golang provides language tooling adapters for Go refactoring and module management.
package golang

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"semedit/internal/pipeline"
)

// AddDependency executes 'go get <pkg>' and 'go mod tidy' in the enclosing module root.
func AddDependency(ctx context.Context, workDir string, pkg string) error {
	trimmedPkg := strings.TrimSpace(pkg)
	if trimmedPkg == "" {
		return errors.New("empty package name")
	}

	modRoot := pipeline.FindModuleRoot(workDir)
	if modRoot == "" {
		return errors.New("cannot determine module root")
	}

	// #nosec G204 -- trimmedPkg is an explicit package identifier
	getCmd := exec.CommandContext(ctx, "go", "get", trimmedPkg)
	getCmd.Dir = modRoot
	getCmd.Env = os.Environ()

	var getErr bytes.Buffer
	getCmd.Stderr = &getErr

	if err := getCmd.Run(); err != nil {
		return fmt.Errorf("go get %s: %w: %s", trimmedPkg, err, getErr.String())
	}

	tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidyCmd.Dir = modRoot
	tidyCmd.Env = os.Environ()

	var tidyErr bytes.Buffer
	tidyCmd.Stderr = &tidyErr

	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w: %s", err, tidyErr.String())
	}

	return nil
}
