//go:build unix

// Package maven scopes cancellation to Maven and its descendants.
package maven

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureMavenProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func stopMavenProcess(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
