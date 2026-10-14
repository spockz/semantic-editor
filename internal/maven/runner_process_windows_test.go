//go:build windows

// Package maven_test keeps the Unix cancellation fixture buildable on Windows.
package maven_test

func mavenProcessAlive(int) (bool, error) {
	return false, nil
}
