// This file proves recursive schema presence, closure, nullability, and typed selector invariants.

package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"os"
	"path/filepath"
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

func TestRegistryProjectsKnownErrorsAgainstPrivateErrorSchema(t *testing.T) {
	resultSchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"status": {Kind: SchemaString, Enum: []any{"done"}},
	}, Required: []string{"status"}}
	errorSchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"application_state": {Kind: SchemaString, Enum: []any{"partially_published", "unknown"}},
	}, Required: []string{"application_state"}}
	definition := Def[string, string]{
		Key:    "typed-error-output",
		Parse:  func(map[string]any) (string, error) { return "", nil },
		Output: &OperationOutputContract{ResultSchema: &resultSchema},
		ProjectOutput: func(string) (any, error) {
			return map[string]any{"status": "done"}, nil
		},
		ErrorOutput: &OperationErrorOutputContract{ResultSchema: &errorSchema},
		ProjectErrorOutput: func(error) (any, bool, error) {
			return map[string]any{"application_state": "partially_published"}, true, nil
		},
	}
	registry := NewRegistry()
	if err := Register(registry, definition); err != nil {
		t.Fatalf("register typed error output: %v", err)
	}
	entry, ok := registry.LookupKey(definition.Key)
	if !ok {
		t.Fatal("registered operation was not found")
	}
	projected, known, err := entry.ProjectErrorOutput(errors.New("write failed"))
	if err != nil {
		t.Fatalf("project known error: %v", err)
	}
	if !known {
		t.Fatal("typed error state was not marked known")
	}
	if got := projected.(map[string]any)["application_state"]; got != "partially_published" {
		t.Fatalf("error state = %v", got)
	}
	result, err := entry.ProjectOutput("success")
	if err != nil || result.(map[string]any)["status"] != "done" {
		t.Fatalf("success projection = %v, %v", result, err)
	}
	definition.ProjectErrorOutput = func(error) (any, bool, error) {
		return map[string]any{"status": "done"}, true, nil
	}
	registry = NewRegistry()
	if err := Register(registry, definition); err != nil {
		t.Fatalf("register mismatched projection test: %v", err)
	}
	entry, _ = registry.LookupKey(definition.Key)
	if _, wasKnown, projectionErr := entry.ProjectErrorOutput(errors.New("write failed")); projectionErr == nil || wasKnown {
		t.Fatalf("error output was not checked against its paired private schema: known=%v err=%v", wasKnown, projectionErr)
	}
}

func TestRegistryRejectsUnpairedErrorOutputContract(t *testing.T) {
	schema := ValueSchema{Kind: SchemaObject}
	definition := Def[string, string]{
		Key:         "orphan-error-output",
		Parse:       func(map[string]any) (string, error) { return "", nil },
		ErrorOutput: &OperationErrorOutputContract{ResultSchema: &schema},
	}
	if err := Register(NewRegistry(), definition); !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("orphan error contract registration error = %v", err)
	}
}

func TestErrorProjectionRejectsLiveFieldsAndUnknownState(t *testing.T) {
	text := ValueSchema{Kind: SchemaString}
	errorSchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"state": text,
	}, Required: []string{"state"}}
	definition := Def[string, string]{
		Key:         "guard-error-output",
		Parse:       func(map[string]any) (string, error) { return "", nil },
		ErrorOutput: &OperationErrorOutputContract{ResultSchema: &errorSchema},
		ProjectErrorOutput: func(error) (any, bool, error) {
			return nil, false, nil
		},
	}
	registry := NewRegistry()
	if err := Register(registry, definition); err != nil {
		t.Fatalf("register unknown error output: %v", err)
	}
	entry, _ := registry.LookupKey(definition.Key)
	if projected, known, err := entry.ProjectErrorOutput(errors.New("untyped failure")); err != nil || known || projected != nil {
		t.Fatalf("unknown error projection = %#v, %v, %v", projected, known, err)
	}
	definition.ProjectErrorOutput = func(error) (any, bool, error) {
		return map[string]any{"state": "unknown", "nested": map[string]any{"continuations": []any{}}}, true, nil
	}
	registry = NewRegistry()
	if err := Register(registry, definition); err != nil {
		t.Fatalf("register runtime guard test: %v", err)
	}
	entry, _ = registry.LookupKey(definition.Key)
	if _, known, err := entry.ProjectErrorOutput(errors.New("typed failure")); err == nil || known {
		t.Fatalf("nested additional live field passed output validation: known=%v err=%v", known, err)
	}

	definition.ErrorOutput.ResultSchema = &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"details": {Kind: SchemaObject, Properties: map[string]ValueSchema{"expected_revision": text}},
	}}
	if err := Register(NewRegistry(), definition); err == nil {
		t.Fatalf("nested reserved field schema error = %v", err)
	}
}

func TestProjectedContinuationFieldsAllowsOnlyDeclaredOptionalOmissions(t *testing.T) {
	text := nonEmptyStringSchema()
	resultSchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"file": text}}
	contract := &OperationOutputContract{ResultSchema: &resultSchema, Continuations: []OutputContinuation{
		{Field: "file", Schema: text, Role: RoleSelector, Consumers: []string{"semantic_organize_imports"}, Optional: true},
	}}
	fields, err := ProjectedContinuationFields(map[string]any{"status": "organized"}, contract)
	if err != nil || len(fields) != 0 {
		t.Fatalf("omitted optional continuation = %#v, %v", fields, err)
	}
	fields, err = ProjectedContinuationFields(map[string]any{"file": "selected.go"}, contract)
	if err != nil || fields["file"] != "selected.go" {
		t.Fatalf("present optional continuation = %#v, %v", fields, err)
	}
	if _, err := ProjectedContinuationFields(map[string]any{"file": ""}, contract); err == nil {
		t.Fatal("empty optional file continuation was accepted")
	}
	if err := ValidateOutputContract(OperationOutputContract{
		ResultSchema:  &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"file": text}, Required: []string{"file"}},
		Continuations: contract.Continuations,
	}); err == nil {
		t.Fatal("required result field accepted an optional continuation")
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
	receipt, err := FinalizeEditReceipt(context.Background(), root, false, false, draft, nil)
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
	receipt, err = FinalizeEditReceipt(context.Background(), root, false, false, noOp, nil)
	if err != nil {
		t.Fatalf("finalize no-op receipt: %v", err)
	}
	if len(receipt.Changes) != 0 {
		t.Fatalf("no-op receipt reports changes: %#v", receipt.Changes)
	}
	refused, err := FinalizeEditReceipt(context.Background(), root, false, false, EditReceiptDraft{Operation: "replace-value", Status: EditRefused}, nil)
	if err != nil {
		t.Fatalf("finalize refused receipt: %v", err)
	}
	if len(refused.Changes) != 0 || len(refused.Continuations) != 0 {
		t.Fatalf("refused receipt reports published values: %#v", refused)
	}
}

func TestFinalizeEditReceiptDistinguishesPartialPublicationFromRefusal(t *testing.T) {
	root := t.TempDir()
	beforeA := []byte("package sample\nvar First = 1\n")
	afterA := []byte("package sample\nvar First = 2\n")
	beforeB := []byte("package sample\nvar Second = 1\n")
	afterB := []byte("package sample\nvar Second = 2\n")
	if err := os.WriteFile(filepath.Join(root, "first.go"), afterA, 0600); err != nil {
		t.Fatalf("write published output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "second.go"), beforeB, 0600); err != nil {
		t.Fatalf("write unchanged preimage: %v", err)
	}
	digest := func(source []byte) string {
		sum := sha256.Sum256(source)
		return hex.EncodeToString(sum[:])
	}
	draft := EditReceiptDraft{
		Operation: "multi-file-edit",
		Status:    EditPersisted,
		PlannedChanges: []FileRevision{
			{Path: "first.go", Revision: digest(afterA)},
			{Path: "second.go", Revision: digest(afterB)},
		},
		ReadSet: []FileRevision{
			{Path: "first.go", Revision: digest(beforeA)},
			{Path: "second.go", Revision: digest(beforeB)},
		},
	}
	partial, err := FinalizeEditReceipt(context.Background(), root, false, false, draft, nil)
	if err != nil {
		t.Fatalf("finalize partial publication: %v", err)
	}
	if partial.Status != EditPartiallyPublished || len(partial.Changes) != 1 || partial.Changes[0].Path != "first.go" {
		t.Fatalf("mixed published and unchanged files = %#v", partial)
	}
	if writeErr := os.WriteFile(filepath.Join(root, "first.go"), beforeA, 0600); writeErr != nil {
		t.Fatalf("restore unchanged preimage: %v", writeErr)
	}
	refused, err := FinalizeEditReceipt(context.Background(), root, false, false, draft, nil)
	if err != nil {
		t.Fatalf("finalize fully unchanged publication: %v", err)
	}
	if refused.Status != EditRefused || len(refused.Changes) != 0 {
		t.Fatalf("fully unchanged planned files = %#v, want refused with no changes", refused)
	}
}

func TestFinalizeEditReceiptRequiresObservedCandidateDigest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	before := []byte("package sample\nvar Widget = 1\n")
	planned := []byte("package sample\nvar Widget = 2\n")
	observed := []byte("package sample\nvar Other = 3\n")
	if err := os.WriteFile(path, observed, 0600); err != nil {
		t.Fatalf("write observed source: %v", err)
	}
	beforeDigest := sha256.Sum256(before)
	plannedDigest := sha256.Sum256(planned)
	observedDigest := sha256.Sum256(observed)
	expected := hex.EncodeToString(plannedDigest[:])
	resolverCalls := 0
	receipt, err := FinalizeEditReceipt(context.Background(), root, false, false, EditReceiptDraft{
		Operation:      "replace-value",
		Status:         EditPersisted,
		PlannedChanges: []FileRevision{{Path: "main.go", Revision: expected}},
		ReadSet:        []FileRevision{{Path: "main.go", Revision: hex.EncodeToString(beforeDigest[:])}},
		Continuations:  []SymbolTarget{{File: "main.go", Symbol: "Widget", ExpectedRevision: &expected}},
	}, func(_ context.Context, target SymbolTarget) (SymbolTarget, bool, error) {
		resolverCalls++
		return target, true, nil
	})
	if err != nil {
		t.Fatalf("finalize normalized postimage: %v", err)
	}
	observedRevision := hex.EncodeToString(observedDigest[:])
	if receipt.Status != EditPartiallyPublished || len(receipt.Changes) != 1 || receipt.Changes[0].Revision != observedRevision {
		t.Fatalf("mismatched postimage evidence = %#v", receipt)
	}
	if resolverCalls != 0 || len(receipt.Continuations) != 0 {
		t.Fatalf("stale candidate reached weak resolver or was emitted: calls=%d continuations=%#v", resolverCalls, receipt.Continuations)
	}

	liveExpected := observedRevision
	receipt, err = FinalizeEditReceipt(context.Background(), root, false, false, EditReceiptDraft{
		Operation:      "replace-value",
		Status:         EditPersisted,
		PlannedChanges: []FileRevision{{Path: "main.go", Revision: observedRevision}},
		ReadSet:        []FileRevision{{Path: "main.go", Revision: hex.EncodeToString(beforeDigest[:])}},
		Continuations:  []SymbolTarget{{File: "main.go", Symbol: "Other", ExpectedRevision: &liveExpected}},
	}, func(_ context.Context, target SymbolTarget) (SymbolTarget, bool, error) {
		return target, true, nil
	})
	if err != nil {
		t.Fatalf("finalize matching candidate: %v", err)
	}
	if len(receipt.Continuations) != 1 || receipt.Continuations[0].ExpectedRevision == nil || *receipt.Continuations[0].ExpectedRevision != observedRevision {
		t.Fatalf("matching final candidate was not emitted with observed digest: %#v", receipt.Continuations)
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
	standalone, err := FinalizeEditReceipt(context.Background(), root, false, false, draft, func(_ context.Context, target SymbolTarget) (SymbolTarget, bool, error) { return target, true, nil })
	if err != nil {
		t.Fatalf("finalize standalone no-op: %v", err)
	}
	if standalone.Status != EditNoOp || len(standalone.Changes) != 0 || len(standalone.Continuations) != 1 {
		t.Fatalf("standalone no-op receipt = %#v", standalone)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if authorizeErr := buffer.AuthorizeWrites(ctx, root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	working, err := FinalizeEditReceipt(ctx, root, true, true, draft, nil)
	if err != nil {
		t.Fatalf("finalize buffered no-op: %v", err)
	}
	if working.Status != EditNoOp || len(working.Changes) != 0 || len(working.Continuations) != 0 {
		t.Fatalf("buffered no-op receipt = %#v", working)
	}
}

func TestProjectInspectResultKeepsLanguageNeutralAmbiguousMatches(t *testing.T) {
	result := &backend.InspectResult{Matches: []backend.InspectedSymbol{
		{QualifiedName: "Widget.run", Kind: "method", File: "src/Widget.rs", Revision: "captured-one"},
		{QualifiedName: "Widget.run", Kind: "method", File: "src/Other.rs", Revision: "captured-two"},
	}}
	projection, err := ProjectInspectResult(result)
	if err != nil {
		t.Fatalf("project language-neutral matches: %v", err)
	}
	if len(projection.Continuations) != 2 {
		t.Fatalf("continuations = %#v, want both ambiguous matches", projection.Continuations)
	}
	for index, want := range []InspectContinuation{
		{File: "src/Widget.rs", Symbol: "Widget.run", ExpectedRevision: "captured-one"},
		{File: "src/Other.rs", Symbol: "Widget.run", ExpectedRevision: "captured-two"},
	} {
		got := projection.Continuations[index]
		if got.File != want.File || got.Symbol != want.Symbol || got.ExpectedRevision != want.ExpectedRevision || got.InFunction != nil || got.InType != nil || got.InDeclaration != nil {
			t.Errorf("continuation %d = %#v, want language-neutral captured identity %#v", index, got, want)
		}
	}
}

func TestProjectInspectResultAddsGoDeclarationOwnerAliases(t *testing.T) {
	kinds := []string{"function", "method", "type", "interface", "constant", "variable", "field"}
	matches := make([]backend.InspectedSymbol, 0, len(kinds))
	for _, kind := range kinds {
		matches = append(matches, backend.InspectedSymbol{
			QualifiedName: "pkg." + kind, Kind: kind, File: "src/sample.go",
			Revision: "captured-" + kind,
		})
	}
	projection, err := ProjectInspectResult(&backend.InspectResult{Matches: matches})
	if err != nil {
		t.Fatalf("project Go declarations: %v", err)
	}
	if len(projection.Continuations) != len(kinds) {
		t.Fatalf("continuations = %d, want %d", len(projection.Continuations), len(kinds))
	}
	for index, kind := range kinds {
		got := projection.Continuations[index]
		wantFunction := kind == "function" || kind == "method"
		wantType := kind == "type" || kind == "interface"
		wantDeclaration := kind != "field"
		if (got.InFunction != nil) != wantFunction || (got.InType != nil) != wantType || (got.InDeclaration != nil) != wantDeclaration {
			t.Errorf("%s aliases = function:%v type:%v declaration:%v", kind, got.InFunction != nil, got.InType != nil, got.InDeclaration != nil)
		}
		for _, alias := range []*string{got.InFunction, got.InType, got.InDeclaration} {
			if alias != nil && *alias != got.Symbol {
				t.Errorf("%s alias = %q, want exact qualified identity %q", kind, *alias, got.Symbol)
			}
		}
	}
}

func TestOutputModeValidationRequiresTypedProjectedContracts(t *testing.T) {
	text := ValueSchema{Kind: SchemaString}
	object := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"value": text}}
	projected := &OperationOutputContract{ResultSchema: &object}
	cases := []struct {
		name        string
		contract    *OperationOutputContract
		projector   bool
		wantInvalid bool
	}{
		{name: "no output", wantInvalid: false},
		{name: "typed result needs projector", contract: projected, wantInvalid: true},
		{name: "projector needs typed result", projector: true, wantInvalid: true},
		{name: "typed result with projector", contract: projected, projector: true},
		{name: "typed continuation needs result", contract: &OperationOutputContract{Continuations: []OutputContinuation{{Field: "value", Schema: text, Role: RoleTarget, Consumers: []string{"semantic_consumer"}}}}, wantInvalid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOutputMode(tc.contract, tc.projector)
			if tc.wantInvalid && err == nil {
				t.Fatal("ValidateOutputMode() accepted an invalid output mode")
			}
			if !tc.wantInvalid && err != nil {
				t.Fatalf("ValidateOutputMode() error = %v", err)
			}
		})
	}
}
