// This file checks that declared output aliases are required only in the native-success schema branch.

package mcp

import (
	"semedit/internal/operation"
	"slices"
	"testing"
)

func TestDeclaredOutputFieldsAreRequiredOnlyForNativeSuccess(t *testing.T) {
	text := operation.ValueSchema{Kind: operation.SchemaString}
	cases := []struct {
		name     string
		contract *operation.OperationOutputContract
		required []string
		optional []string
	}{
		{
			name: "legacy envelope",
			contract: &operation.OperationOutputContract{
				Metadata: []operation.OutputMetadataField{
					{Field: "kind", Schema: text},
					{Field: "line", Schema: operation.ValueSchema{Kind: operation.SchemaInteger}, Optional: true},
				},
				EnvelopeContinuations: []operation.EnvelopeContinuationField{
					{Field: "file", Schema: text, Role: operation.RoleTarget, Consumers: []string{"semantic_consumer"}},
					{Field: "symbol", Schema: text, Role: operation.RoleTarget, Consumers: []string{"semantic_consumer"}, Optional: true},
				},
			},
			required: []string{"kind", "file"},
			optional: []string{"line", "symbol"},
		},
		{
			name: "projected alias",
			contract: &operation.OperationOutputContract{
				ResultSchema: &operation.ValueSchema{
					Kind: operation.SchemaObject,
					Properties: map[string]operation.ValueSchema{
						"symbol": text,
					},
					Required: []string{"symbol"},
				},
				Continuations: []operation.OutputContinuation{
					{Field: "symbol", Schema: text, Role: operation.RoleTarget, Consumers: []string{"semantic_consumer"}},
				},
			},
			required: []string{"symbol"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := operation.Entry{MCPName: "semantic_provider", Output: tc.contract}
			result, err := toolOutputAndContinuations(entry)
			if err != nil {
				t.Fatalf("build output schema: %v", err)
			}
			branches := result.schema["oneOf"].([]any)
			success := branches[0].(map[string]any)
			assertRequiredFields(t, success, append([]string{"result"}, tc.required...))
			for _, field := range tc.optional {
				assertFieldNotRequired(t, success, field)
			}
			genericError := branches[1].(map[string]any)
			assertRequiredFields(t, genericError, []string{"error", applicationStateField})
		})
	}
}

func assertRequiredFields(t *testing.T, branch map[string]any, fields []string) {
	t.Helper()
	required, ok := branch["required"].([]string)
	if !ok {
		t.Fatalf("branch required fields have type %T", branch["required"])
	}
	for _, field := range fields {
		found := slices.Contains(required, field)
		if !found {
			t.Errorf("branch required fields %v omitted %q", required, field)
		}
	}
}

func assertFieldNotRequired(t *testing.T, branch map[string]any, field string) {
	t.Helper()
	required, ok := branch["required"].([]string)
	if !ok {
		t.Fatalf("branch required fields have type %T", branch["required"])
	}
	for _, actual := range required {
		if actual == field {
			t.Errorf("optional field %q was required", field)
		}
	}
}
