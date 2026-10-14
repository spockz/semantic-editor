// This file checks that typed result paths remain nested under the native-success result object.

package mcp

import (
	"github.com/spockz/semantic-editor/internal/operation"
	"slices"
	"testing"
)

func TestTypedOutputFieldsUseDeclaredResultPaths(t *testing.T) {
	text := operation.ValueSchema{Kind: operation.SchemaString}
	resultSchema := operation.ValueSchema{
		Kind: operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{
			"symbol": text,
			"line":   {Kind: operation.SchemaInteger},
		},
		Required: []string{"symbol"},
	}
	entry := operation.Entry{
		MCPName: "semantic_provider",
		Output: &operation.OperationOutputContract{
			ResultSchema: &resultSchema,
			Continuations: []operation.OutputContinuation{{
				Path: "result.symbol", Schema: text, Role: operation.RoleTarget, Consumers: []string{"semantic_consumer"},
			}},
		},
		ProjectOutput: func(any) (any, error) { return map[string]any{"symbol": "Widget"}, nil },
	}
	result, err := toolOutputAndContinuations(entry)
	if err != nil {
		t.Fatalf("build output schema: %v", err)
	}
	branches := result.schema["oneOf"].([]any)
	success := branches[0].(map[string]any)
	assertRequiredFields(t, success, []string{"result"})
	properties := success["properties"].(map[string]any)
	for _, field := range []string{"symbol", "line"} {
		if _, exists := properties[field]; exists {
			t.Errorf("native envelope exposed typed result field %q as a sibling", field)
		}
	}
	typedResult := properties["result"].(map[string]any)
	assertRequiredFields(t, typedResult, []string{"symbol"})
	typedProperties := typedResult["properties"].(map[string]any)
	for _, field := range []string{"symbol", "line"} {
		if typedProperties[field] == nil {
			t.Errorf("typed result omitted %q", field)
		}
	}
	genericError := branches[1].(map[string]any)
	assertRequiredFields(t, genericError, []string{"error", applicationStateField})
}

func assertRequiredFields(t *testing.T, branch map[string]any, fields []string) {
	t.Helper()
	required, ok := branch["required"].([]string)
	if !ok {
		t.Fatalf("branch required fields have type %T", branch["required"])
	}
	for _, field := range fields {
		if !slices.Contains(required, field) {
			t.Errorf("branch required fields %v omitted %q", required, field)
		}
	}
}
