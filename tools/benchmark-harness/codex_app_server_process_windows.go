//go:build windows

// Package main configures Codex process cleanup on Windows.
package main

import (
	"fmt"
	"os"
	"os/exec"
)

func configureCodexAppServerProcess(cmd *exec.Cmd) {}

func signalCodexAppServerProcess(pid int, force bool) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find Codex process %d: %w", pid, err)
	}
	if err := process.Kill(); err != nil {
		return fmt.Errorf("kill Codex process %d: %w", pid, err)
	}
	return nil
}
