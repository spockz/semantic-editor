//go:build !windows

package mcp

import (
	"fmt"
	"os"
	"syscall"
)

// execReload executes the current process executable in-place, preserving stdio file descriptors.
func execReload() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	// Stdio descriptors (0, 1, 2) are preserved across execve.
	return syscall.Exec(execPath, os.Args, os.Environ()) //nolint:gosec // Re-executing self executable with identical arguments is intentional for live-reload.
}
