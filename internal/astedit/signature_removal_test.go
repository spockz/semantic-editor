// WHY: Keep parameter removal plans bound to typed unused-formal and caller evidence before any output exists.
package astedit

import (
	"slices"
	"strings"
	"testing"
)

func TestPlanChangeSignatureRemovalEvidence(t *testing.T) {
	t.Parallel()
	source := []byte("package p\nfunc combine(left int,legacy string,right int)int{return left*100+right}\nfunc next()int{return 3}\nfunc use()int{return combine(next(),\"old\",8)}\n")
	start := strings.Index(string(source), "combine(next(),")
	call := SignatureCallSite{File: "service.go", Start: start, End: start + len("combine(next(),\"old\",8)"), Revision: signatureRevision(source)}
	position := start + len("combine(")
	for index, argument := range []struct{ source, value, kind string }{{"next()", "", "int"}, {`"old"`, "String:old", "string"}, {"8", "Int:8", "int"}} {
		call.Arguments = append(call.Arguments, SignatureArgumentEvidence{Index: index, Start: position, End: position + len(argument.source), Source: argument.source, Value: argument.value, Type: argument.kind})
		position += len(argument.source) + 1
	}
	zero, two := 0, 2
	request := SignatureChangeRequest{File: "service.go", Symbol: "combine", Parameters: []SignatureParameterMapping{{SourceIndex: &zero, Name: "left", Type: "int"}, {SourceIndex: &two, Name: "right", Type: "int"}}}
	proof := SignatureCallSetProof{Complete: true, Files: []SignatureSourceFile{{File: "service.go", Source: source}}, Calls: []SignatureCallSite{call}, Removal: &SignatureParameterRemovalEvidence{File: "service.go", Revision: signatureRevision(source), Index: 1, Name: "legacy", Scalar: true, Unused: true}}
	plan, err := PlanChangeSignature(request, proof)
	if err != nil {
		t.Fatalf("plan removal: %v", err)
	}
	if len(plan.Files) != 1 || !strings.Contains(string(plan.Files[0].After), "func combine(left int, right int)int{return left*100+right}") || !strings.Contains(string(plan.Files[0].After), "combine(next(), 8)") {
		t.Errorf("removal output = %+v", plan.Files)
	}
	for _, scenario := range []struct {
		name   string
		mutate func(*SignatureCallSetProof)
	}{
		{"missing formal", func(p *SignatureCallSetProof) { p.Removal = nil }},
		{"used formal", func(p *SignatureCallSetProof) { p.Removal.Unused = false }},
		{"nonscalar formal", func(p *SignatureCallSetProof) { p.Removal.Scalar = false }},
		{"stale formal", func(p *SignatureCallSetProof) { p.Removal.Revision = "stale" }},
		{"wrong file", func(p *SignatureCallSetProof) { p.Removal.File = "other.go" }},
		{"wrong formal index", func(p *SignatureCallSetProof) { p.Removal.Index = 0 }},
		{"wrong formal name", func(p *SignatureCallSetProof) { p.Removal.Name = "left" }},
		{"nonconstant discarded argument", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[1].Value = "" }},
		{"untyped survivor", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[0].Type = "" }},
		{"stale argument", func(p *SignatureCallSetProof) { p.Calls[0].Arguments[0].Start++ }},
		{"missing caller", func(p *SignatureCallSetProof) { p.Calls[0].Arguments = nil }},
		{"stale caller", func(p *SignatureCallSetProof) { p.Calls[0].Revision = "stale" }},
		{"duplicate caller", func(p *SignatureCallSetProof) { p.Calls = append(p.Calls, p.Calls[0]) }},
		{"incomplete set", func(p *SignatureCallSetProof) { p.Complete = false }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			candidate := proof
			formal := *proof.Removal
			candidate.Removal = &formal
			candidate.Calls = slices.Clone(proof.Calls)
			candidate.Calls[0].Arguments = slices.Clone(call.Arguments)
			scenario.mutate(&candidate)
			if output, planErr := PlanChangeSignature(request, candidate); planErr == nil || len(output.Files) != 0 {
				t.Errorf("invalid evidence produced output: %+v, %v", output, planErr)
			}
		})
	}
}
