// Package operation verifies native MCP result projections.
package operation

import (
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/maven"
	"github.com/spockz/semantic-editor/internal/projectverify"
	"testing"
)

func TestWorkspaceOutlineNativeMixedMetadataMatchesSchema(t *testing.T) {
	entry, ok := DefaultRegistry().LookupMCP("semantic_workspace_outline")
	if !ok || entry.Output == nil || entry.Output.ResultSchema == nil || entry.ProjectOutput == nil {
		t.Fatal("semantic_workspace_outline output contract is incomplete")
	}
	doc := "An app module."
	result := WorkspaceOutlineRes{Result: &backend.WorkspaceOutlineResult{
		Scope: backend.ReadScope{Kind: "workspace", Path: ".", Complete: true},
		Modules: []backend.WorkspaceModule{
			{Path: "go", ModulePath: "example.test/go", IsRoot: false, Language: "go", PackagesCount: 0},
			{Path: "java", GroupID: "com.example", ArtifactID: "app", IsRoot: false, Language: "java", Synopsis: "Application module", Doc: &doc, PackagesCount: 1},
		},
		Packages: []backend.WorkspacePackage{
			{Path: "go/pkg", Name: "pkg", Language: "go", Module: "example.test/go"},
			{Path: "java/src", Name: "com.example", Language: "java", Module: "com.example:app"},
		},
	}}
	projected, err := entry.ProjectOutput(result)
	if err != nil {
		t.Fatalf("project workspace outline: %v", err)
	}
	output, ok := projected.(map[string]any)
	if !ok {
		t.Fatalf("projected workspace outline type = %T", projected)
	}
	if err := ValidateValue(*entry.Output.ResultSchema, output); err != nil {
		t.Fatalf("projected workspace outline violates result schema: %v", err)
	}
	modules, ok := output["modules"].([]any)
	if !ok || len(modules) != 2 {
		t.Fatalf("projected modules = %#v", output["modules"])
	}
	firstModule, ok := modules[0].(map[string]any)
	if !ok || firstModule["language"] != "go" || fmt.Sprint(firstModule["packages_count"]) != "0" {
		t.Fatalf("first module metadata = %#v", modules[0])
	}
	packages, ok := output["packages"].([]any)
	if !ok || len(packages) != 2 {
		t.Fatalf("projected packages = %#v", output["packages"])
	}
	javaPackage, ok := packages[1].(map[string]any)
	if !ok || javaPackage["language"] != "java" || javaPackage["module"] != "com.example:app" {
		t.Fatalf("Java package metadata = %#v", packages[1])
	}
}

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
