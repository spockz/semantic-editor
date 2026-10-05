// Package operation verifies native MCP result projections.
package operation

import (
	"errors"
	"fmt"
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
	if _, ok := projected.(map[string]any); !ok {
		t.Fatalf("projected Maven result type = %T", projected)
	}
	knownError := fmt.Errorf("outer wrapper: %w", &maven.Error{Kind: "execution", Err: errors.New("process failed"), Result: &result})
	projectedError, known, err := entry.ProjectErrorOutput(knownError)
	if err != nil || !known {
		t.Fatalf("project Maven error = (%#v, %v, %v)", projectedError, known, err)
	}
	if _, ok := projectedError.(map[string]any); !ok {
		t.Fatalf("projected Maven error result type = %T", projectedError)
	}
	for _, err := range []error{errors.New("unrelated"), &maven.Error{Kind: "no result", Err: errors.New("failed")}} {
		if value, known, projectErr := entry.ProjectErrorOutput(err); projectErr != nil || known || value != nil {
			t.Fatalf("projected unrelated Maven error = (%#v, %v, %v)", value, known, projectErr)
		}
	}
}
