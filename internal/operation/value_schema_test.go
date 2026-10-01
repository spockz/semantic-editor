// This file proves recursive schema presence, closure, nullability, and typed selector invariants.

package operation

import "testing"

func TestValueSchemaPreservesPresenceAndClosesNestedObjects(t *testing.T) {
	zero := 0
	floatPtr := floatPointer
	stringSchema := ValueSchema{Kind: SchemaString}
	parameterSchema := ValueSchema{
		Kind: SchemaObject,
		Properties: map[string]ValueSchema{
			"name":     {Kind: SchemaString, MinLength: &zero},
			"type":     stringSchema,
			"variadic": {Kind: SchemaBoolean},
		},
		Required: []string{"name", "type"},
	}
	mappingSchema := ValueSchema{
		Kind: SchemaObject,
		Properties: map[string]ValueSchema{
			"source_index": {Kind: SchemaInteger, Minimum: floatPtr(0)},
			"parameter":    parameterSchema,
			"argument":     {Kind: SchemaString, Nullable: true},
		},
		Required: []string{"source_index", "parameter"},
	}
	schema := ValueSchema{Kind: SchemaArray, Items: &mappingSchema}
	valid := []any{map[string]any{
		"source_index": 0,
		"parameter":    map[string]any{"name": "", "type": "Context", "variadic": false},
		"argument":     "",
	}}
	if err := ValidateValue(schema, valid); err != nil {
		t.Fatalf("valid zero/false/empty mapping rejected: %v", err)
	}
	if err := ValidateValue(schema, []any{}); err != nil {
		t.Fatalf("empty array rejected: %v", err)
	}
	if err := ValidateValue(schema, []any{map[string]any{
		"parameter": map[string]any{"name": "ctx", "type": "Context"},
	}}); err == nil {
		t.Error("missing required source_index was accepted")
	}
	if err := ValidateValue(schema, []any{map[string]any{
		"source_index": nil,
		"parameter":    map[string]any{"name": "ctx", "type": "Context"},
	}}); err == nil {
		t.Error("null non-nullable source_index was accepted")
	}
	if err := ValidateValue(schema, []any{map[string]any{
		"source_index": 0,
		"parameter":    map[string]any{"name": "ctx", "type": "Context"},
		"unexpected":   true,
	}}); err == nil {
		t.Error("unknown nested property was accepted")
	}
	if err := ValidateValue(schema, []any{map[string]any{
		"source_index": 0,
		"parameter":    map[string]any{"name": "ctx", "type": "Context"},
		"argument":     nil,
	}}); err != nil {
		t.Errorf("nullable explicit null was rejected: %v", err)
	}
	if err := ValidateValue(schema, []any{map[string]any{
		"source_index": 0,
		"parameter":    map[string]any{"type": "Context"},
	}}); err == nil {
		t.Error("missing required nested parameter name was accepted")
	}
}

func TestValueSchemaRejectsUnknownKindsAndMalformedDeclarations(t *testing.T) {
	intPtr := intPointer
	cases := []ValueSchema{
		{Kind: SchemaKind("future")},
		{Kind: SchemaArray},
		{Kind: SchemaObject, Required: []string{"missing"}},
		{Kind: SchemaString, MinLength: intPtr(3), MaxLength: intPtr(2)},
		{Kind: SchemaString, Properties: map[string]ValueSchema{}},
	}
	for index, schema := range cases {
		if err := ValidateSchema(schema); err == nil {
			t.Errorf("schema case %d was accepted: %#v", index, schema)
		}
	}
	if err := ValidateValue(ValueSchema{Kind: SchemaInteger}, 1.5); err == nil {
		t.Error("fractional integer was accepted")
	}
	if err := ValidateValue(ValueSchema{Kind: SchemaBoolean, Enum: []any{false}}, false); err != nil {
		t.Errorf("false enum value was rejected: %v", err)
	}
	if err := ValidateValue(ValueSchema{Kind: SchemaBoolean, Enum: []any{false}}, true); err == nil {
		t.Error("value outside enum was accepted")
	}
}

func TestTypedDescriptorsPreserveOptionalZeroAndRequireRevisionForPaths(t *testing.T) {
	zero := 0
	notVariadic := false
	mappings := []ParameterMapping{{SourceIndex: &zero, Parameter: ParameterSpec{Name: "ctx", Type: "Context", Variadic: &notVariadic}}}
	if err := ValidateParameterMappings(mappings); err != nil {
		t.Fatalf("zero source index or explicit false variadic was rejected: %v", err)
	}
	if err := ValidateParameterToFieldMappings([]ParameterToFieldMapping{{SourceIndex: 0, Field: FieldSpec{Name: "ctx", Type: "Context"}}}); err != nil {
		t.Fatalf("zero field source index was rejected: %v", err)
	}
	if err := ValidateParameterMappings([]ParameterMapping{mappings[0], mappings[0]}); err == nil {
		t.Error("duplicate source index was accepted")
	}
	if err := ValidateParameterToFieldMappings([]ParameterToFieldMapping{
		{SourceIndex: 0, Field: FieldSpec{Name: "first", Type: "Context"}},
		{SourceIndex: 0, Field: FieldSpec{Name: "second", Type: "Context"}},
	}); err == nil {
		t.Error("one parameter mapped to multiple fields")
	}
	path := "0"
	owner := "Server.Handle"
	selector := ConstructSelector{File: "server.go", InFunction: &owner, Kind: "case", ConstructPath: &path}
	if err := ValidateConstructSelector(selector); err == nil {
		t.Error("construct path without expected_revision was accepted")
	}
	revision := "sha256:opaque"
	selector.ExpectedRevision = &revision
	if err := ValidateConstructSelector(selector); err != nil {
		t.Errorf("revision-bound construct path rejected: %v", err)
	}
	call := CallSiteSelector{File: "server.go", InFunction: owner, Discriminator: "handle(req)", ConstructPath: &path}
	if err := ValidateCallSiteSelector(call); err == nil {
		t.Error("call-site path without expected_revision was accepted")
	}
	call.ExpectedRevision = &revision
	if err := ValidateCallSiteSelector(call); err != nil {
		t.Errorf("revision-bound call-site path rejected: %v", err)
	}
}

func TestIntegerSchemaAcceptsIntegralJSONNumbers(t *testing.T) {
	if err := ValidateValue(ValueSchema{Kind: SchemaInteger}, float64(0)); err != nil {
		t.Errorf("integral JSON number zero was rejected: %v", err)
	}
	if err := ValidateValue(ValueSchema{Kind: SchemaInteger}, float64(1.5)); err == nil {
		t.Error("fractional JSON number was accepted as an integer")
	}
}

//go:fix inline
func floatPointer(value float64) *float64 {
	return new(value)
}

//go:fix inline
func intPointer(value int) *int {
	return new(value)
}
