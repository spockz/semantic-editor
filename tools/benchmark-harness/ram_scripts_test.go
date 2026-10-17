// This test runs the RAM script's unprivileged sizing and refusal checks from the package test suite.
package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBenchmarkRAMScripts(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate benchmark RAM script test")
	}
	script := filepath.Join(filepath.Dir(file), "scripts", "test-benchmark-ram.sh")
	command := exec.Command("bash", script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run benchmark RAM script checks: %v\n%s", err, output)
	}
}
