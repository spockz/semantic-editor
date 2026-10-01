// This file proves recursive schema presence, closure, nullability, and typed selector invariants.

package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"semedit/internal/sourcebuf"
	"testing"
)

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
	for _, value := range []json.Number{"1.0", "1e0", "0e1000000000", "-0.0e-1000000000"} {
		if err := ValidateValue(ValueSchema{Kind: SchemaInteger}, value); err != nil {
			t.Errorf("integral JSON number %q was rejected: %v", value, err)
		}
	}
	for _, value := range []json.Number{"1.5", "1e1000000000", "1e-1000000000"} {
		if err := ValidateValue(ValueSchema{Kind: SchemaInteger}, value); err == nil {
			t.Errorf("non-integer or unbounded JSON number %q was accepted", value)
		}
	}
}

func TestValueSchemaRejectsInvalidEnumsAndNegativeBounds(t *testing.T) {
	validMinimum := 1
	validMaximum := 5
	valid := ValueSchema{Kind: SchemaString, MinLength: &validMinimum, MaxLength: &validMaximum, Enum: []any{"valid"}}
	if err := ValidateSchema(valid); err != nil {
		t.Fatalf("valid string enum rejected: %v", err)
	}
	if err := ValidateValue(valid, "valid"); err != nil {
		t.Errorf("valid string enum value rejected: %v", err)
	}
	invalidEnumSchemas := []ValueSchema{
		{Kind: SchemaString, Enum: []any{true}},
		{Kind: SchemaString, Enum: []any{nil}},
	}
	for index, schema := range invalidEnumSchemas {
		if err := ValidateSchema(schema); err == nil {
			t.Errorf("invalid string enum case %d was accepted", index)
		}
	}
	negativeMaximum := -1
	if err := ValidateSchema(ValueSchema{Kind: SchemaString, MaxLength: &negativeMaximum}); err == nil {
		t.Error("negative string maximum was accepted")
	}
	item := ValueSchema{Kind: SchemaString}
	if err := ValidateSchema(ValueSchema{Kind: SchemaArray, Items: &item, MaxItems: &negativeMaximum}); err == nil {
		t.Error("negative array maximum was accepted")
	}
}

func TestIntegerSchemaComparesLargeEnumsExactly(t *testing.T) {
	first := uint64(9007199254740992)
	second := uint64(9007199254740993)
	schema := ValueSchema{Kind: SchemaInteger, Enum: []any{first}}
	if err := ValidateValue(schema, second); err == nil {
		t.Error("adjacent large uint64 enum value was accepted")
	}
	if err := ValidateValue(schema, first); err != nil {
		t.Errorf("matching large uint64 enum value was rejected: %v", err)
	}
	decimalSchema := ValueSchema{Kind: SchemaInteger, Enum: []any{json.Number("9007199254740993")}}
	if err := ValidateValue(decimalSchema, json.Number("9007199254740992")); err == nil {
		t.Error("adjacent large json.Number enum value was accepted")
	}
	if err := ValidateValue(decimalSchema, json.Number("9007199254740993")); err != nil {
		t.Errorf("matching large json.Number enum value was rejected: %v", err)
	}
	if err := ValidateValue(ValueSchema{Kind: SchemaInteger}, float64(9007199254740992)); err == nil {
		t.Error("integer above the exact float64 input range was accepted")
	}
}

func TestRegistryEntryParseValidatesTypedRecursiveObjectInput(t *testing.T) {
	nameSchema := ValueSchema{Kind: SchemaString}
	rowSchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"name": nameSchema}, Required: []string{"name"}}
	rowsSchema := ValueSchema{Kind: SchemaArray, Items: &rowSchema}
	nullableText := ValueSchema{Kind: SchemaString, Nullable: true}
	schema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"text": nameSchema, "enabled": {Kind: SchemaBoolean}, "index": {Kind: SchemaInteger},
		"rows": rowsSchema, "optional": nullableText,
	}, Required: []string{"text", "enabled", "index", "rows"}}
	registry := NewRegistry()
	definition := Def[map[string]any, string]{
		Key:    "typed-object-test",
		Params: []ParameterContract{{Name: "payload", JSONName: "payload", CLIName: "payload", Type: ParamObject, Required: true, Schema: &schema}},
		Parse:  func(raw map[string]any) (map[string]any, error) { return raw, nil },
	}
	if err := Register(registry, definition); err != nil {
		t.Fatalf("register typed operation: %v", err)
	}
	entry, ok := registry.LookupKey(definition.Key)
	if !ok {
		t.Fatal("registered typed operation was not found")
	}
	payload := map[string]any{"text": "", "enabled": false, "index": int64(0), "rows": []any{}, "optional": nil}
	parsed, err := entry.Parse(map[string]any{"payload": payload})
	if err != nil {
		t.Fatalf("valid zero, false, empty, and nullable values rejected: %v", err)
	}
	if parsed.(map[string]any)["payload"].(map[string]any)["enabled"] != false {
		t.Fatal("false value was not preserved")
	}
	invalid := []map[string]any{
		{"text": "", "enabled": false, "index": 0, "rows": []any{}, "unexpected": true},
		{"text": "", "enabled": false, "index": 0, "rows": []any{map[string]any{"unexpected": "x"}}},
		{"text": nil, "enabled": false, "index": 0, "rows": []any{}},
		{"text": "", "enabled": false, "index": 0, "rows": []any{map[string]any{}}},
	}
	for index, value := range invalid {
		if _, err := entry.Parse(map[string]any{"payload": value}); err == nil {
			t.Errorf("invalid recursive payload %d was accepted", index)
		}
	}
	if _, err := entry.Parse(map[string]any{}); err == nil {
		t.Error("omitted required object parameter was accepted")
	}
	if err := Register(NewRegistry(), Def[map[string]any, string]{Key: "missing-schema", Params: []ParameterContract{{Name: "payload", JSONName: "payload", Type: ParamObject}}, Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil }}); err == nil {
		t.Error("structured parameter without a declared schema was registered")
	}
}

func TestParameterSchemaContractsRejectConflictsAndKeepLegacyPrimitives(t *testing.T) {
	textSchema := ValueSchema{Kind: SchemaString}
	invalid := [][]ParameterContract{
		{{Name: "payload", JSONName: "payload", Type: ParamObject, Schema: &textSchema}},
		{{Name: "payload", JSONName: "payload", Type: ParamString, Schema: &textSchema, Enums: []string{"ok"}}},
		{{Name: "payload", JSONName: "payload", Type: ParamArray}},
	}
	for index, params := range invalid {
		if err := ValidateParameterContracts(params); err == nil {
			t.Errorf("conflicting parameter schema contract %d was accepted", index)
		}
	}
	legacy := []ParameterContract{{Name: "mode", JSONName: "mode", Type: ParamString, Enums: []string{"fast", "safe"}}}
	if err := ValidateParameterContracts(legacy); err != nil {
		t.Fatalf("legacy primitive contract was rejected: %v", err)
	}
	if err := ValidateParameterValues(map[string]any{"mode": "fast"}, legacy); err != nil {
		t.Errorf("legacy enum value was rejected: %v", err)
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

func TestRegistryRejectsOutputProjectorWithoutResultSchema(t *testing.T) {
	registry := NewRegistry()
	definition := Def[string, string]{
		Key:           "missing-result-schema",
		Parse:         func(map[string]any) (string, error) { return "", nil },
		Output:        &OperationOutputContract{},
		ProjectOutput: func(result string) (any, error) { return result, nil },
	}
	if err := Register(registry, definition); err == nil {
		t.Fatal("registered an output projector without a result schema")
	}
}

func TestSchemaAtResultPathRejectsMalformedSegmentsAndSelectsArrayLeaves(t *testing.T) {
	schema := EditReceiptSchema()
	leaf, err := SchemaAtResultPath(schema, "result.continuations[].symbol")
	if err != nil {
		t.Fatalf("resolve nested array leaf: %v", err)
	}
	if leaf.Kind != SchemaString {
		t.Fatalf("nested symbol schema kind = %q", leaf.Kind)
	}
	for _, path := range []string{"result.", "result..status", "result.continuations[].", "result.continuations[][]"} {
		if _, err := SchemaAtResultPath(schema, path); err == nil {
			t.Errorf("malformed result path %q was accepted", path)
		}
	}
}

func TestFinalizeEditReceiptIncludesOnlyObservedPostimages(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	before := []byte("package sample\nvar Value = 1\n")
	after := []byte("package sample\nvar Value = 2\n")
	if err := os.WriteFile(path, after, 0600); err != nil {
		t.Fatalf("write final source: %v", err)
	}
	beforeDigest := sha256.Sum256(before)
	afterDigest := sha256.Sum256(after)
	draft := EditReceiptDraft{
		Operation: "replace-value",
		Status:    EditPersisted,
		PlannedChanges: []FileRevision{
			{Path: "main.go", Revision: hex.EncodeToString(afterDigest[:])},
			{Path: "missing.go", Revision: hex.EncodeToString(afterDigest[:])},
		},
		ReadSet: []FileRevision{{Path: "main.go", Revision: hex.EncodeToString(beforeDigest[:])}},
	}
	receipt, err := FinalizeEditReceipt(context.Background(), root, false, draft, nil)
	if err != nil {
		t.Fatalf("finalize partial receipt: %v", err)
	}
	if receipt.Status != EditPartiallyPublished || len(receipt.Changes) != 1 || receipt.Changes[0].Path != "main.go" || receipt.Changes[0].Revision != hex.EncodeToString(afterDigest[:]) {
		t.Fatalf("partial publication evidence = %#v", receipt)
	}
	noOp := EditReceiptDraft{
		Operation:      "replace-value",
		Status:         EditPersisted,
		PlannedChanges: []FileRevision{{Path: "main.go", Revision: hex.EncodeToString(afterDigest[:])}},
		ReadSet:        []FileRevision{{Path: "main.go", Revision: hex.EncodeToString(afterDigest[:])}},
	}
	receipt, err = FinalizeEditReceipt(context.Background(), root, false, noOp, nil)
	if err != nil {
		t.Fatalf("finalize no-op receipt: %v", err)
	}
	if len(receipt.Changes) != 0 {
		t.Fatalf("no-op receipt reports changes: %#v", receipt.Changes)
	}
	refused, err := FinalizeEditReceipt(context.Background(), root, false, EditReceiptDraft{Operation: "replace-value", Status: EditRefused}, nil)
	if err != nil {
		t.Fatalf("finalize refused receipt: %v", err)
	}
	if len(refused.Changes) != 0 || len(refused.Continuations) != 0 {
		t.Fatalf("refused receipt reports published values: %#v", refused)
	}
}

func TestFinalizeEditReceiptPreservesNoOpOutcomes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	content := []byte("package sample\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	revision := hex.EncodeToString(digest[:])
	candidate := SymbolTarget{File: "main.go", Symbol: "Value", ExpectedRevision: &revision}
	draft := EditReceiptDraft{Operation: "contract", Status: EditNoOp, ReadSet: []FileRevision{{Path: "main.go", Revision: revision}}, Continuations: []SymbolTarget{candidate}}
	standalone, err := FinalizeEditReceipt(context.Background(), root, false, draft, func(_ context.Context, target SymbolTarget) (SymbolTarget, bool, error) { return target, true, nil })
	if err != nil {
		t.Fatalf("finalize standalone no-op: %v", err)
	}
	if standalone.Status != EditNoOp || len(standalone.Changes) != 0 || len(standalone.Continuations) != 1 {
		t.Fatalf("standalone no-op receipt = %#v", standalone)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if err := buffer.AuthorizeWrites(ctx, root); err != nil {
		t.Fatal(err)
	}
	working, err := FinalizeEditReceipt(ctx, root, true, draft, nil)
	if err != nil {
		t.Fatalf("finalize buffered no-op: %v", err)
	}
	if working.Status != EditNoOp || len(working.Changes) != 0 || len(working.Continuations) != 0 {
		t.Fatalf("buffered no-op receipt = %#v", working)
	}
}
