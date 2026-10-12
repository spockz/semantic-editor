// Package operation verifies native MCP result projections.
package operation

import (
	"errors"
	"fmt"
	"semedit/internal/astedit"
	"semedit/internal/maven"
	"semedit/internal/projectverify"
	"testing"
)

func TestVerifyNativeResultProjectionPreservesWireShape(t *testing.T) {
	entry, ok := DefaultRegistry().LookupMCP("semantic_verify")
	if !ok || entry.Output == nil || entry.Output.ResultSchema == nil || entry.ProjectOutput == nil {
		t.Fatal("semantic_verify output contract is incomplete")
	}
	result := VerifyRes{Failure: "native verification failed", Scope: projectverify.Scope{Paths: nil}, NormalizeReport: projectverify.Report{Root: ".", Scope: projectverify.Scope{Paths: nil}, Results: nil}, CheckReport: projectverify.Report{Phase: projectverify.PhaseCheck, Scope: projectverify.Scope{Paths: nil}, Results: nil}}
	projected, err := entry.ProjectOutput(result)
	if err != nil {
		t.Fatalf("project verify output: %v", err)
	}
	object, ok := projected.(map[string]any)
	if !ok {
		t.Fatalf("projected verify result type = %T", projected)
	}
	if object["failure"] != result.Failure {
		t.Fatalf("failure = %#v", object["failure"])
	}
	normalize, ok := object["normalize"].(map[string]any)
	if !ok || normalize["Phase"] != "" || normalize["Scope"] == nil || normalize["Results"] != nil {
		t.Fatalf("native normalize report shape = %#v", object["normalize"])
	}
}

func TestMavenNativeResultAndKnownErrorProjection(t *testing.T) {
	entry, ok := DefaultRegistry().LookupMCP("semantic_maven_test")
	if !ok || entry.Output == nil || entry.ErrorOutput == nil || entry.ProjectOutput == nil || entry.ProjectErrorOutput == nil {
		t.Fatal("semantic_maven_test output contract is incomplete")
	}
	result := MavenRes{Goal: "test", Tool: "mvn", Command: nil, ExitCode: 1, DurationMS: 12, Error: "process failed"}
	projected, err := entry.ProjectOutput(result)
	if err != nil {
		t.Fatalf("project Maven output: %v", err)
	}
	if _, projectedIsObject := projected.(map[string]any); !projectedIsObject {
		t.Fatalf("projected Maven result type = %T", projected)
	}
	knownError := fmt.Errorf("outer wrapper: %w", &maven.Error{Kind: "execution", Err: errors.New("process failed"), Result: &result})
	projectedError, known, err := entry.ProjectErrorOutput(knownError)
	if err != nil || !known {
		t.Fatalf("project Maven error = (%#v, %v, %v)", projectedError, known, err)
	}
	if _, projectedErrorIsObject := projectedError.(map[string]any); !projectedErrorIsObject {
		t.Fatalf("projected Maven error result type = %T", projectedError)
	}
	for _, projectInputErr := range []error{errors.New("unrelated"), &maven.Error{Kind: "no result", Err: errors.New("failed")}} {
		if value, known, projectErr := entry.ProjectErrorOutput(projectInputErr); projectErr != nil || known || value != nil {
			t.Fatalf("projected unrelated Maven error = (%#v, %v, %v)", value, known, projectErr)
		}
	}
}

func TestConstructInspectionAndSelectionErrorUseTypedOutputs(t *testing.T) {
	registry := DefaultRegistry()
	inspect, ok := registry.LookupMCP("semantic_inspect_construct")
	if !ok || inspect.Output == nil || inspect.Output.ResultSchema == nil || inspect.ProjectOutput == nil {
		t.Fatal("semantic_inspect_construct output contract is incomplete")
	}
	result := InspectConstructRes{File: "sample.go", InFunction: "Check", Kind: "if", Candidates: []astedit.ConstructCandidateInfo{{ConstructPath: "0", Syntax: "if ready {\n}"}}, SelectionRevision: "current"}
	projected, err := inspect.ProjectOutput(result)
	if err != nil {
		t.Fatalf("project inspect construct output: %v", err)
	}
	output, projectedOK := projected.(map[string]any)
	if !projectedOK {
		t.Fatalf("projected inspect construct = %#v", projected)
	}
	if err := ValidateValue(*inspect.Output.ResultSchema, output); err != nil {
		t.Fatalf("projected inspect construct violates result schema: %v", err)
	}
	if output["file"] != "sample.go" || output[wireInFunction] != "Check" || output[wireExpectedSelectionRevision] != "current" {
		t.Fatalf("projected inspect construct fields = %#v", output)
	}
	candidates, candidatesOK := output[wireCandidates].([]any)
	if !candidatesOK || len(candidates) != 1 {
		t.Fatalf("projected inspect construct candidates = %#v", output[wireCandidates])
	}
	candidate, candidateOK := candidates[0].(map[string]any)
	if !candidateOK || candidate[wireConstructPath] != "0" || candidate["syntax"] != "if ready {\n}" {
		t.Fatalf("projected inspect construct candidate = %#v", candidates[0])
	}
	replace, ok := registry.LookupMCP("semantic_replace_construct")
	if !ok || replace.ErrorOutput == nil || replace.ProjectErrorOutput == nil {
		t.Fatal("semantic_replace_construct selection error contract is incomplete")
	}
	known := &astedit.ConstructSelectionError{Status: "ambiguous", File: "sample.go", Candidates: astedit.ConstructCandidateSet{InFunction: "Check", Kind: "if", Candidates: []astedit.ConstructCandidateInfo{{ConstructPath: "0", Syntax: "if ready {\n}"}}, SelectionRevision: "current"}}
	value, recognized, projectErr := replace.ProjectErrorOutput(fmt.Errorf("wrapped: %w", known))
	if projectErr != nil || !recognized {
		t.Fatalf("project construct selection error = (%#v, %v, %v)", value, recognized, projectErr)
	}
	errorOutput, errorOutputOK := value.(map[string]any)
	if !errorOutputOK || errorOutput["application_state"] != "not_applied" || errorOutput["expected_selection_revision"] != "current" {
		t.Fatalf("projected construct selection error = %#v", value)
	}
}
