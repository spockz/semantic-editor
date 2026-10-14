//go:build windows

// Package maven scopes cancellation to the Maven process on Windows.
package maven

import (
	"errors"
	"os"
	"os/exec"
)

func configureMavenProcess(_ *exec.Cmd) {}

func stopMavenProcess(cmd *exec.Cmd) error {
	err := cmd.Process.Kill()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}
