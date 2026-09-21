// Package golang provides language tooling adapters for Go refactoring and module management.
package golang

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"semedit/internal/gocache"
	"semedit/internal/pipeline"
)

var (
	// ErrEmptyPackage indicates an empty package was supplied to dependency tooling.
	ErrEmptyPackage = errors.New("empty package name")
	// ErrNoModuleRoot indicates a go.mod file could not be located.
	ErrNoModuleRoot = errors.New("cannot determine module root")
	// ErrDependencyFailed indicates go get failed.
	ErrDependencyFailed = errors.New("dependency acquisition failed")
	// ErrModTidyFailed indicates go mod tidy failed.
	ErrModTidyFailed = errors.New("go mod tidy failed")
)

// AddDependency executes 'go get <pkg>' and 'go mod tidy' in the enclosing module root.
func AddDependency(ctx context.Context, workDir string, pkg string) error {
	trimmedPkg := strings.TrimSpace(pkg)
	if trimmedPkg == "" {
		return ErrEmptyPackage
	}

	modRoot := pipeline.FindModuleRoot(workDir)
	if modRoot == "" {
		return ErrNoModuleRoot
	}
	env, err := gocache.Environment(ctx, modRoot)
	if err != nil {
		return fmt.Errorf("prepare Go dependency environment: %w", err)
	}

	// #nosec G204 -- trimmedPkg is an explicit package identifier
	getCmd := exec.CommandContext(ctx, "go", "get", trimmedPkg)
	getCmd.Dir = modRoot
	getCmd.Env = env

	var getErr bytes.Buffer
	getCmd.Stderr = &getErr

	if err := getCmd.Run(); err != nil {
		return fmt.Errorf("%w: go get %s: %s: %w", ErrDependencyFailed, trimmedPkg, getErr.String(), err)
	}

	tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidyCmd.Dir = modRoot
	tidyCmd.Env = env

	var tidyErr bytes.Buffer
	tidyCmd.Stderr = &tidyErr

	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrModTidyFailed, tidyErr.String(), err)
	}

	return nil
}
