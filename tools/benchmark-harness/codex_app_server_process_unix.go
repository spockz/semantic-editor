//go:build unix

// Package main configures the Codex process group for bounded Unix cleanup.
package main

import (
	"os/exec"
	"syscall"
)

func configureCodexAppServerProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalCodexAppServerProcess(pid int, force bool) error {
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-pid, signal)
}
