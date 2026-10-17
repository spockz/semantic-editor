// tmpfs_other.go keeps ordinary benchmark builds available where native tmpfs validation is unsupported.

//go:build !darwin && !linux

package main

import "fmt"

func validateTmpfsMount(path string) error {
	return fmt.Errorf("native tmpfs validation is unsupported on this platform for %q", path)
}

func isTmpfsMount(path string) (bool, error) {
	return false, fmt.Errorf("native tmpfs validation is unsupported on this platform for %q", path)
}

func fixtureNeedsLinkerOutputRefresh(string) (bool, error) {
	return false, nil
}
