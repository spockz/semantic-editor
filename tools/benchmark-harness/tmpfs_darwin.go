// tmpfs_darwin.go identifies macOS RAM filesystems through the native statfs type field.

//go:build darwin

package main

import (
	"fmt"
	"strings"
	"syscall"
)

func validateTmpfsMount(path string) error {
	tmpfs, err := isTmpfsMount(path)
	if err != nil {
		return err
	}
	if !tmpfs {
		return fmt.Errorf("filesystem is not tmpfs")
	}
	return nil
}

func isTmpfsMount(path string) (bool, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return false, fmt.Errorf("stat filesystem %q: %w", path, err)
	}
	var bytes strings.Builder
	for _, value := range stat.Fstypename {
		if value == 0 {
			break
		}
		bytes.WriteRune(rune(value))
	}
	return strings.EqualFold(bytes.String(), "tmpfs"), nil
}

func fixtureNeedsLinkerOutputRefresh(path string) (bool, error) {
	tmpfs, err := isTmpfsMount(path)
	if err != nil {
		return false, err
	}
	return tmpfs, nil
}
