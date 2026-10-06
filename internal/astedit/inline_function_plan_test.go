// WHY: Exercises semantic edge cases that the inline-function planner must refuse or preserve.

package astedit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/sourcebuf"
)

func TestPlanInlineFunctionPreservesDeclaredResultType(t *testing.T) {
	t.Parallel()
	caller := "package sample\nfunc Run() any { return adjust() }\n"
	callee := "package sample\nfunc adjust() float64 { return 1 }\n"
	plan, err := PlanInlineFunction(caller, callee, "adjust", "Run", "adjust()")
	if err != nil {
		t.Fatalf("PlanInlineFunction() error = %v", err)
	}
	got := caller[:plan.Start] + plan.Replacement + caller[plan.End:]
	want := "package sample\nfunc Run() any { return float64(1) }\n"
	if got != want {
		t.Fatalf("inlined source = %q, want %q", got, want)
	}
}

func TestPlanInlineFunctionRefusesUnprovenBindingsAndEvaluation(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, caller, callee string }{
		{name: "parameter evaluation", caller: "package p\nfunc Run() int { return adjust(next()) }\nfunc next() int { return 1 }\n", callee: "package p\nfunc adjust(v int) int { return v }\n"},
		{name: "callee shadow", caller: "package p\nfunc Run(adjust func() int) int { return adjust() }\n", callee: "package p\nfunc adjust() int { return 1 }\n"},
		{name: "result type shadow", caller: "package p\nfunc Run(float64 int) any { return adjust() }\n", callee: "package p\nfunc adjust() float64 { return 1 }\n"},
		{name: "non-return call", caller: "package p\nfunc Run() { adjust() }\n", callee: "package p\nfunc adjust() int { return 1 }\n"},
		{name: "incompatible literal", caller: "package p\nfunc Run() string { return adjust() }\n", callee: "package p\nfunc adjust() string { return 1 }\n"},
		{name: "duplicate call site", caller: "package p\nfunc Run(flag bool) int { if flag { return adjust() }; return adjust() }\n", callee: "package p\nfunc adjust() int { return 1 }\n"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PlanInlineFunction(test.caller, test.callee, "adjust", "Run", "adjust()"); err == nil {
				t.Fatal("PlanInlineFunction() error = nil, want refusal")
			}
		})
	}
}

func TestApplySemanticEditRechecksCapturedInputsAtPublication(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	targetPath := filepath.Join(root, "main.go")
	manifestPath := filepath.Join(root, "go.mod")
	before := []byte("package sample\nfunc Value() int { return 1 }\n")
	after := []byte("package sample\nfunc Value() int { return 2 }\n")
	manifest := []byte("module example.com/sample\n\ngo 1.24\n")
	for path, content := range map[string][]byte{targetPath: before, manifestPath: manifest} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan := SemanticEditPlan{
		Change:  sourcebuf.Change{Path: targetPath, Before: bytes.Clone(before), After: after},
		ReadSet: map[string][]byte{targetPath: bytes.Clone(before), manifestPath: bytes.Clone(manifest)},
	}
	if err := os.WriteFile(manifestPath, []byte("module example.com/changed\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ApplySemanticEdit(context.Background(), plan); err == nil {
		t.Fatal("ApplySemanticEdit() error = nil, want captured manifest drift refusal")
	}
	actual, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, before) {
		t.Fatalf("target changed after manifest drift: %q", actual)
	}
}

func TestCompleteRefactorReadSetRefusesManifestDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifestPath := filepath.Join(root, "go.mod")
	if err := os.WriteFile(manifestPath, []byte("module example.com/current\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := map[string][]byte{manifestPath: []byte("module example.com/captured\n\ngo 1.24\n")}
	if _, err := completeRefactorReadSet(context.Background(), root, original, nil); err == nil {
		t.Fatal("completeRefactorReadSet() error = nil, want manifest drift refusal")
	}
}
