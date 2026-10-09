//go:build !windows

// opencode_process_unix.go scopes serve shutdown to OpenCode and its descendants.

package main

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureOpenCodeServerProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalOpenCodeServerProcess(cmd *exec.Cmd, force bool) error {
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	err := syscall.Kill(-cmd.Process.Pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
