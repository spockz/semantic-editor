// Package operation_test checks the Java explicit-type operation's public continuation contract.
// WHY: A renamed file is a valid target while the method selector and required revision still come from fresh inspection.
package operation_test

import (
	"slices"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
)

func TestJavaMakeTypeExplicitContinuationSources(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("refactor_make_type_explicit")
	if !ok {
		t.Fatal("Java explicit-type operation is not registered")
	}
	for _, target := range []struct {
		name, source string
		role         operation.ParameterRole
	}{
		{"file", "refactor_rename.file", operation.RoleTarget},
		{"file", "semantic_inspect_symbol.result.continuations[].file", operation.RoleTarget},
		{"in_function", "semantic_inspect_symbol.result.continuations[].in_function", operation.RoleSelector},
		{"expected_revision", "semantic_inspect_symbol.result.continuations[].expected_revision", operation.RoleTarget},
	} {
		t.Run(target.source, func(t *testing.T) {
			index := slices.IndexFunc(entry.Params, func(parameter operation.ParameterContract) bool { return parameter.JSONName == target.name })
			if index < 0 {
				t.Fatalf("missing parameter %q", target.name)
			}
			parameter := entry.Params[index]
			if !parameter.Required || parameter.Role != target.role || !slices.Contains(parameter.SourceFields, target.source) {
				t.Fatalf("parameter %q must require a %s target accepting %s: %#v", target.name, target.role, target.source, parameter)
			}
		})
	}
}

func TestJavaMakeTypeExplicitReceivesProjectedMethodContinuation(t *testing.T) {
	want := "Service.count()"
	projection, err := operation.ProjectInspectResult(&backend.InspectResult{Matches: []backend.InspectedSymbol{
		{QualifiedName: want, Kind: "method", File: "src/Service.java", Revision: "captured-revision"},
	}})
	if err != nil {
		t.Fatalf("project Java method inspection: %v", err)
	}
	if len(projection.Continuations) != 1 {
		t.Fatalf("continuations = %#v, want one method continuation", projection.Continuations)
	}
	got := projection.Continuations[0]
	if got.File != "src/Service.java" || got.Symbol != want || got.ExpectedRevision != "captured-revision" || got.InFunction == nil || *got.InFunction != want {
		t.Fatalf("Java inspection continuation = %#v, want file, method identity, and revision for %q", got, want)
	}
}
