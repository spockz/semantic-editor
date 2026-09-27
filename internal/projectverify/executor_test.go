// This file tests hook ordering, partial failures, and staged normalization publication.

package projectverify

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunPhasePreservesHookOrderAndPartialFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orderPath := filepath.Join(root, "order")
	plan := Plan{
		Root:    root,
		Sources: []Source{{Path: "main.go", Language: "go"}},
		Checks: []Hook{
			{ID: "first", Languages: []string{"go"}, Shell: &ShellSpec{Executable: "/bin/sh", Args: []string{"-c"}, Script: "printf first >> order"}},
			{ID: "second", Languages: []string{"go"}, Shell: &ShellSpec{Executable: "/bin/sh", Args: []string{"-c"}, Script: "printf second >> order; exit 9"}},
		},
	}
	report, err := RunPhase(context.Background(), plan, PhaseCheck, Scope{}, nil)
	if err == nil {
		t.Fatal("RunPhase succeeded after a hook failed")
	}
	if !report.Partial || report.Completed || len(report.Results) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.Results[0].Status != HookSucceeded || report.Results[1].Status != HookFailed {
		t.Fatalf("hook statuses = %q, %q", report.Results[0].Status, report.Results[1].Status)
	}
	order, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(order) != "firstsecond" {
		t.Fatalf("hook order output = %q", order)
	}
}

func TestRunPhasePublishesScopedNormalizationAtomically(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".scratch"), 0o700); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORIGINAL_SOURCE", mainPath)
	plan := Plan{
		Root:    root,
		Sources: []Source{{Path: "main.go", Language: "go"}},
		Normalize: []Hook{{
			ID:        "format",
			Languages: []string{"go"},
			Shell:     &ShellSpec{Executable: "/bin/sh", Args: []string{"-c"}, Script: "test \"$(cat \"$ORIGINAL_SOURCE\")\" = \"package main\" && printf 'package main\\n// formatted\\n' > main.go"},
		}},
	}
	report, err := RunPhase(context.Background(), plan, PhaseNormalize, Scope{Paths: []string{mainPath}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Completed || len(report.Results) != 1 || len(report.Results[0].Changed) != 1 {
		t.Fatalf("report = %#v", report)
	}
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package main\n// formatted\n" {
		t.Fatalf("published content = %q", content)
	}
}

func TestRunPhaseRejectsOutOfScopeNormalizationWithoutPublishing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".scratch"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"main.go": "package main\n", "other.go": "package other\n", "go.mod": "module fixture.test\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan := Plan{
		Root:    root,
		Sources: []Source{{Path: "main.go", Language: "go"}},
		Normalize: []Hook{{
			ID:        "bad",
			Languages: []string{"go"},
			Shell:     &ShellSpec{Executable: "/bin/sh", Args: []string{"-c"}, Script: "printf changed > main.go; printf changed > other.go; printf changed > go.mod"},
		}},
	}
	report, err := RunPhase(context.Background(), plan, PhaseNormalize, Scope{Paths: []string{"main.go"}}, nil)
	if err == nil || report.Completed || len(report.Results) != 1 || report.Results[0].Status != HookFailed {
		t.Fatalf("report = %#v, error = %v", report, err)
	}
	for name, want := range map[string]string{"main.go": "package main\n", "other.go": "package other\n", "go.mod": "module fixture.test\n"} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != want {
			t.Fatalf("%s was published as %q", name, content)
		}
	}
}

func TestRunPhaseFailsWhenExplicitExecutableIsMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := Plan{
		Root:    root,
		Sources: []Source{{Path: "main.go", Language: "go"}},
		Checks:  []Hook{{ID: "required", Languages: []string{"go"}, Exec: []string{filepath.Join(root, "missing-tool")}}},
	}
	report, err := RunPhase(context.Background(), plan, PhaseCheck, Scope{}, nil)
	if err == nil || report.Completed || len(report.Results) != 1 || report.Results[0].Status != HookFailed {
		t.Fatalf("report = %#v, error = %v", report, err)
	}
}
