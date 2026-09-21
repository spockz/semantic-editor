// Package gocache selects a verified writable Go build cache for subprocesses.
package gocache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment returns an environment with a verified writable GOCACHE.
func Environment(workDir string) ([]string, error) {
	env := os.Environ()
	candidate := os.Getenv("GOCACHE")
	if !writable(candidate) {
		if workDir == "" {
			return nil, fmt.Errorf("GOCACHE is unusable and work directory is empty")
		}
		candidate = filepath.Join(workDir, ".scratch", "go-cache")
		if err := os.MkdirAll(candidate, 0o750); err != nil {
			return nil, fmt.Errorf("create fallback GOCACHE %q: %w", candidate, err)
		}
		if !writable(candidate) {
			return nil, fmt.Errorf("fallback GOCACHE %q is not writable", candidate)
		}
	}
	filtered := make([]string, 0, len(env)+1)
	for _, value := range env {
		if !strings.HasPrefix(value, "GOCACHE=") {
			filtered = append(filtered, value)
		}
	}
	return append(filtered, "GOCACHE="+candidate), nil
}

func writable(dir string) bool {
	// #nosec G703 -- dir is the caller-selected GOCACHE candidate; no path outside it is targeted.
	if dir == "" || os.MkdirAll(dir, 0o750) != nil {
		return false
	}
	probe, err := os.CreateTemp(dir, ".semedit-cache-")
	if err != nil {
		return false
	}
	name := probe.Name()
	_ = probe.Close()
	// #nosec G703 -- name is returned by CreateTemp in the verified cache directory.
	_ = os.Remove(name)
	return true
}
