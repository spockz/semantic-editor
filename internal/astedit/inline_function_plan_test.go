// WHY: Exercises semantic edge cases that the inline-function planner must refuse or preserve.

package astedit

import "testing"

func TestPlanInlineFunctionPreservesDeclaredResultType(t *testing.T) {
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
