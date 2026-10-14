//go:build unix

// Package maven_test observes process readiness before testing Unix cancellation.
package maven_test

import (
	"errors"
	"syscall"
)

func mavenProcessAlive(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, err
}
