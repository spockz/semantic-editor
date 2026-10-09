//go:build windows

// opencode_process_windows.go provides the OpenCode server process lifecycle on Windows.

package main

import (
	"os"
	"os/exec"
)

func configureOpenCodeServerProcess(cmd *exec.Cmd) {}

func signalOpenCodeServerProcess(cmd *exec.Cmd, force bool) error {
	if force {
		return cmd.Process.Kill()
	}
	return cmd.Process.Signal(os.Interrupt)
}
