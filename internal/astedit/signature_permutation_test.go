// WHY: Exercises the pure permutation plan's rejection of incomplete or stale compiler evidence before any output is returned.

package astedit

import (
	"slices"
	"strings"
	"testing"
)

func TestPlanChangeSignaturePermutationEvidence(t *testing.T) {
	t.Parallel()
	source := []byte("package p\nfunc combine(left,right int)int{return left*100+right}\nfunc use()int{return combine(1+2,4)}\n")
	start := strings.Index(string(source), "combine(1+2,4)")
	argument := start + len("combine(")
	zero, one := 0, 1
	request := SignatureChangeRequest{File: "service.go", Symbol: "combine", Parameters: []SignatureParameterMapping{{SourceIndex: &one, Name: "right", Type: "int"}, {SourceIndex: &zero, Name: "left", Type: "int"}}}
	call := SignatureCallSite{File: "service.go", Start: start, End: start + len("combine(1+2,4)"), Revision: signatureRevision(source), Arguments: []SignatureArgumentEvidence{
		{Index: 0, Start: argument, End: argument + 3, Source: "1+2", Value: "Int:3", Type: "int"},
		{Index: 1, Start: argument + 4, End: argument + 5, Source: "4", Value: "Int:4", Type: "int"},
	}}
	proof := SignatureCallSetProof{Complete: true, Files: []SignatureSourceFile{{File: "service.go", Source: source}}, Calls: []SignatureCallSite{call}}
	plan, err := PlanChangeSignature(request, proof)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Files) != 1 || !strings.Contains(string(plan.Files[0].After), "func combine(right int, left int)int{return left*100+right}") || !strings.Contains(string(plan.Files[0].After), "combine(4,1+2)") {
		t.Errorf("permutation output = %+v", plan.Files)
	}
	for _, scenario := range []struct {
		name   string
		mutate func(*SignatureCallSetProof)
	}{
		{"missing arguments", func(p *SignatureCallSetProof) { p.Calls[0].Arguments = nil }},
		{"duplicate argument", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[1] = p.Calls[0].Arguments[0] }},
		{"stale caller", func(p *SignatureCallSetProof) { p.Calls[0].Revision = "stale" }},
		{"stale range", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[0].Start++ }},
		{"stale source", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[0].Source = "3" }},
		{"missing value", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[0].Value = "" }},
		{"missing type", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[0].Type = "" }},
		{"duplicate call", func(p *SignatureCallSetProof) { p.Calls = append(p.Calls, p.Calls[0]) }},
		{"incomplete set", func(p *SignatureCallSetProof) { p.Complete = false }},
		{"missing file", func(p *SignatureCallSetProof) { p.Files = nil }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			candidate := proof
			candidate.Calls = slices.Clone(proof.Calls)
			candidate.Calls[0].Arguments = slices.Clone(call.Arguments)
			scenario.mutate(&candidate)
			output, planErr := PlanChangeSignature(request, candidate)
			if planErr == nil || len(output.Files) != 0 {
				t.Errorf("invalid evidence produced output: %+v, %v", output, planErr)
			}
		})
	}
	t.Run("stale target", func(t *testing.T) {
		candidate := request
		candidate.ExpectedRevision = "stale"
		if output, planErr := PlanChangeSignature(candidate, proof); planErr == nil || len(output.Files) != 0 {
			t.Errorf("stale target output: %+v, %v", output, planErr)
		}
	})
}
