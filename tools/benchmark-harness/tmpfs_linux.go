// tmpfs_linux.go identifies Linux tmpfs mounts without parsing user-visible command output.

//go:build linux

package main

import (
	"fmt"
	"syscall"
)

const linuxTmpfsMagic = 0x01021994

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
	return stat.Type == linuxTmpfsMagic, nil
}

func fixtureNeedsLinkerOutputRefresh(string) (bool, error) {
	return false, nil
}
