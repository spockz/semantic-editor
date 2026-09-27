// This file verifies project verification discovery and strict configuration behavior.

package projectverify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverSignalsAndRejectsUnknownConfigFields(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("main.go", "package main\nfunc main() {}\n")
	write(".golangci.yml", "version: '2'\n")
	plan, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Sources) != 1 || plan.Sources[0] != (Source{Path: "main.go", Language: "go"}) {
		t.Fatalf("sources = %#v", plan.Sources)
	}
	if len(plan.Checks) != 1 || plan.Checks[0].Signal != "golangci-config" || plan.Checks[0].ConfigPath != ".golangci.yml" {
		t.Fatalf("checks = %#v", plan.Checks)
	}
	write(".semedit.yaml", "version: 1\nunknown: true\n")
	if _, err := Discover(root); err == nil {
		t.Fatal("Discover accepted an unknown project config field")
	}
}

func TestRunPhaseUsesDisjointBuildableGolintPackageScopes(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string, mode os.FileMode) {
		t.Helper()
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module fixture.test/root\n\ngo 1.27.1\n", 0o600)
	write("main.go", "package main\nfunc main() {}\n", 0o600)
	write(".golangci.yml", "version: '2'\n", 0o600)
	write("nested/go.mod", "module fixture.test/nested\n\ngo 1.27.1\n", 0o600)
	write("nested/nested.go", "package nested\n", 0o600)
	write("nested/.golangci.yml", "version: '2'\n", 0o600)
	bin := filepath.Join(root, "bin")
	record := filepath.Join(root, "calls")
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(bin, "golangci-lint")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf '%s|%s\\n' \"$PWD\" \"$*\" >> \"$RECORDING_FILE\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RECORDING_FILE", record)
	plan, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Checks) != 2 {
		t.Fatalf("discovered checks = %#v", plan.Checks)
	}
	report, err := RunPhase(context.Background(), plan, PhaseCheck, Scope{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 2 || report.Results[0].Status != HookSucceeded || report.Results[1].Status != HookSucceeded {
		t.Fatalf("report = %#v", report)
	}
	calls, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "|run --config "+filepath.Join(root, ".golangci.yml")+" ./") || !strings.Contains(lines[1], "|run --config "+filepath.Join(root, "nested/.golangci.yml")+" ./") {
		t.Fatalf("golangci calls = %q", calls)
	}
	if !strings.HasPrefix(lines[0], root+"|") || !strings.HasPrefix(lines[1], filepath.Join(root, "nested")+"|") {
		t.Fatalf("golangci working directories = %q", calls)
	}
}

func TestDiscoverExplicitDisableSuppressesDetectedLinter(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"main.go":       "package main\n",
		".golangci.yml": "version: '2'\n",
		".semedit.yaml": "version: 1\nverify:\n  check:\n    - id: golangci-lint\n      languages: [go]\n      disabled: true\n",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Checks) != 0 {
		t.Fatalf("disabled detected linter remains enabled: %#v", plan.Checks)
	}
}
