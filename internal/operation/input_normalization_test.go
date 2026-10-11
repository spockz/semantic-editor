// Package operation verifies ingress aliases are canonical before contracts and handlers run.
package operation_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"testing"

	"semedit/internal/backend"
	gobackend "semedit/internal/backend/golang"
	"semedit/internal/operation"
)

func TestNormalizeConstructKindInputAliases(t *testing.T) {
	for _, alias := range []string{"composite_element", "literal_element", "entry", "item"} {
		t.Run(alias, func(t *testing.T) {
			input := map[string]any{"kind": alias, "file": "value.go"}
			got, err := operation.NormalizeConstructKindInput(input)
			if err != nil {
				t.Fatal(err)
			}
			if got["kind"] != "element" {
				t.Fatalf("normalized kind = %v, want element", got["kind"])
			}
			if input["kind"] != alias {
				t.Fatalf("normalizer mutated caller input: %v", input)
			}
		})
	}
	for _, kind := range []any{"element", "case", 1, nil} {
		input := map[string]any{"kind": kind}
		got, err := operation.NormalizeConstructKindInput(input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, input) {
			t.Fatalf("normalizing %v = %v, want unchanged", input, got)
		}
	}
}

func TestInsertConstructRegistersCanonicalElementVocabulary(t *testing.T) {
	registry := operation.DefaultRegistry()
	entry, ok := registry.LookupKey("insert_construct")
	if !ok {
		t.Fatal("insert_construct is not registered")
	}
	if entry.NormalizeInput == nil {
		t.Fatal("insert_construct has no ingress normalizer")
	}
	for _, alias := range []string{"composite_element", "literal_element", "entry", "item"} {
		parsed, err := entry.Parse(map[string]any{
			"file": "routes.go", "kind": alias, "source": `"ready": 2`,
			"in_declaration": "routes", "discriminator": `map[string]int{"health": 1}`,
		})
		if err != nil {
			t.Fatalf("Parse rejected %q: %v", alias, err)
		}
		request, ok := parsed.(operation.InsertConstructReq)
		if !ok || request.Kind != "element" {
			t.Fatalf("Parse(%q) = %#v, want InsertConstructReq{Kind: element}", alias, parsed)
		}
	}
	kindParam := parameterByJSONName(t, entry, "kind")
	for _, alias := range []string{"composite_element", "literal_element", "entry", "item"} {
		if slices.Contains(kindParam.Enums, alias) {
			t.Errorf("kind schema advertises compatibility alias %q", alias)
		}
	}
	if kindParam.Role != "" || slices.Contains(kindParam.SourceFields, "semantic_insert_construct.result.selector_continuations[].kind") {
		t.Errorf("informational selector kind must remain outside the consumer graph: %#v", kindParam)
	}
	parameters := operation.ParametersForBackends(entry, []backend.Backend{gobackend.NewGoBackend()})
	kindParam = parameterByJSONName(t, operation.Entry{Params: parameters}, "kind")
	if !slices.Contains(kindParam.Enums, "element") {
		t.Errorf("kind schema omits canonical element: %v", kindParam.Enums)
	}
	for _, alias := range []string{"composite_element", "literal_element", "entry", "item"} {
		if slices.Contains(kindParam.Enums, alias) {
			t.Errorf("kind schema advertises compatibility alias %q", alias)
		}
	}
	selectorSchema := entry.Output.ResultSchema.Properties["selector_continuations"]
	if selectorSchema.Items == nil {
		t.Fatal("selector_continuations has no typed item schema")
	}
	kindSchema := selectorSchema.Items.Properties["kind"]
	if kindSchema.Kind != operation.SchemaString || !reflect.DeepEqual(kindSchema.Enum, []any{"element"}) {
		t.Errorf("selector kind schema = %#v, want closed element enum", kindSchema)
	}
}

func TestInsertConstructElementParserContract(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupKey("insert_construct")
	if !ok {
		t.Fatal("insert_construct is not registered")
	}
	base := map[string]any{
		"file": "routes.go", "kind": "element", "source": `"ready": 2`,
		"in_declaration": "routes", "discriminator": `map[string]int{"health": 1}`,
	}
	assertInsertElementParserContracts(t, entry, base)
}

func assertInsertElementParserContracts(t *testing.T, entry operation.Entry, base map[string]any) {
	t.Helper()
	parsed, err := entry.Parse(base)
	if err != nil {
		t.Fatalf("valid element request rejected: %v", err)
	}
	if got := parsed.(operation.InsertConstructReq).Placement; got != "append" {
		t.Errorf("default element placement = %q, want append", got)
	}
	assertElementForbiddenFields(t, entry, base)
	assertElementRequiredFields(t, entry, base)
	assertElementPlacementNormalization(t, entry, base)
	assertNonElementGuards(t, entry)
	assertRevisionAndPathContracts(t, entry)
}

func assertElementForbiddenFields(t *testing.T, entry operation.Entry, base map[string]any) {
	fields := map[string]any{
		"in_type": "Config", "in_function": "Handle", "target_symbol": "Anchor",
		"anchor": "end", "access_modifier": "infer", "group": "append", "construct_path": "0",
	}
	for field, value := range fields {
		t.Run("element rejects "+field, func(t *testing.T) {
			raw := cloneInput(base)
			raw[field] = value
			assertInvalidInput(t, entry, raw)
		})
		t.Run("dispatch rejects element policy "+field, func(t *testing.T) {
			raw := cloneInput(base)
			raw[field] = value
			_, err := operation.DefaultRegistry().Dispatch(operation.CallContext{Project: backend.ProjectContext{Language: backend.LanguageGo}}, "insert_construct", raw)
			if !errors.Is(err, operation.ErrInvalidParams) {
				t.Fatalf("Dispatch with %s returned %v, want ErrInvalidParams", field, err)
			}
		})
	}
}

func assertElementRequiredFields(t *testing.T, entry operation.Entry, base map[string]any) {
	for field, value := range map[string]any{"in_declaration": " \t", "discriminator": " \n", "source": " \t", "expected_revision": " \t"} {
		t.Run("element rejects blank "+field, func(t *testing.T) {
			raw := cloneInput(base)
			raw[field] = value
			assertInvalidInput(t, entry, raw)
		})
	}
}

func assertElementPlacementNormalization(t *testing.T, entry operation.Entry, base map[string]any) {
	raw := cloneInput(base)
	raw["placement"] = " \t"
	parsed, err := entry.Parse(raw)
	if err != nil || parsed.(operation.InsertConstructReq).Placement != "append" {
		t.Fatalf("blank placement = (%#v, %v), want canonical append", parsed, err)
	}
}

func assertNonElementGuards(t *testing.T, entry operation.Entry) {
	for field, value := range map[string]any{"in_declaration": "routes", "placement": "append", "construct_path": "0"} {
		t.Run("non-element rejects "+field, func(t *testing.T) {
			raw := map[string]any{"file": "routes.go", "kind": "function", "source": "func NewRoute() {}", field: value}
			assertInvalidInput(t, entry, raw)
		})
	}
}

func assertRevisionAndPathContracts(t *testing.T, entry operation.Entry) {
	for _, tc := range []struct {
		kind string
		want error
	}{
		{kind: "statement/expression"},
		{kind: "function", want: operation.ErrInvalidParams},
	} {
		t.Run("revision for "+tc.kind, func(t *testing.T) {
			raw := map[string]any{"file": "routes.go", "kind": tc.kind, "source": "return nil", "expected_revision": "rev"}
			_, err := entry.Parse(raw)
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("Parse returned %v, want %v", err, tc.want)
			}
			if tc.want == nil && err != nil {
				t.Fatalf("Parse rejected revision alone: %v", err)
			}
		})
	}
	for _, path := range []string{"0", "1.2"} {
		t.Run("statement path "+path, func(t *testing.T) {
			raw := map[string]any{"file": "routes.go", "kind": "statement/expression", "source": "return nil", "construct_path": path}
			assertInvalidInput(t, entry, raw)
			raw["expected_revision"] = "rev"
			if _, err := entry.Parse(raw); err != nil {
				t.Fatalf("statement path with revision rejected: %v", err)
			}
		})
	}
	for _, kind := range []string{"case", "field", "element"} {
		t.Run("unsupported path for "+kind, func(t *testing.T) {
			raw := minimalInsertInput(kind)
			if kind == "element" {
				raw["in_declaration"] = "routes"
				raw["discriminator"] = `map[string]int{"health": 1}`
			}
			if kind == "field" {
				raw["in_type"] = "Config"
			}
			raw["expected_revision"] = "rev"
			if kind == "case" {
				assertInvalidInput(t, entry, raw)
				return
			}
			if _, err := entry.Parse(raw); err != nil {
				t.Fatalf("revision rejected for %s: %v", kind, err)
			}
			raw["construct_path"] = "0"
			assertInvalidInput(t, entry, raw)
		})
	}
}

func minimalInsertInput(kind string) map[string]any {
	return map[string]any{"file": "routes.go", "kind": kind, "source": "x"}
}

func assertInvalidInput(t *testing.T, entry operation.Entry, raw map[string]any) {
	t.Helper()
	if _, err := entry.Parse(raw); !errors.Is(err, operation.ErrInvalidParams) {
		t.Fatalf("Parse returned %v, want ErrInvalidParams", err)
	}
}

func TestRegisteredIngressPathsNormalizeBeforeValidationAndParsing(t *testing.T) {
	type request struct{ Kind string }
	registry := operation.NewRegistry()
	var parsedKinds []string
	var finalizedKinds []string
	def := operation.Def[request, string]{
		Key:           "normalize_input_probe",
		BufferedBatch: true,
		Params: []operation.ParameterContract{{
			Name: "kind", JSONName: "kind", Type: operation.ParamString,
			Required: true, Enums: []string{"element"},
		}},
		NormalizeInput: operation.NormalizeConstructKindInput,
		Parse: func(raw map[string]any) (request, error) {
			kind, _ := raw["kind"].(string)
			parsedKinds = append(parsedKinds, kind)
			return request{Kind: kind}, nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, request) (string, error){
			backend.LanguageGo: func(_ context.Context, _ operation.CallContext, req request) (string, error) {
				return req.Kind, nil
			},
		},
		FinalizeOutput: func(_ operation.CallContext, req request, result string) (string, error) {
			finalizedKinds = append(finalizedKinds, req.Kind)
			return result, nil
		},
	}
	if err := operation.Register(registry, def); err != nil {
		t.Fatal(err)
	}
	entry, ok := registry.LookupKey(def.Key)
	if !ok {
		t.Fatal("registered entry missing")
	}

	if _, err := entry.Parse(map[string]any{"kind": "item"}); err != nil {
		t.Fatalf("Parse rejected alias before canonicalizing it: %v", err)
	}
	if eligible, err := entry.SupportsBufferedBatch(operation.CallContext{Project: backend.ProjectContext{Language: backend.LanguageGo}}, map[string]any{"kind": "entry"}); err != nil || !eligible {
		t.Fatalf("SupportsBufferedBatch = %v, %v; want true, nil", eligible, err)
	}
	result, err := registry.Dispatch(operation.CallContext{Ctx: context.Background(), Project: backend.ProjectContext{Language: backend.LanguageGo}}, def.Key, map[string]any{"kind": "literal_element"})
	if err != nil {
		t.Fatalf("Dispatch rejected alias: %v", err)
	}
	if result != "element" {
		t.Fatalf("Dispatch result = %v, want canonical kind element", result)
	}
	if !reflect.DeepEqual(parsedKinds, []string{"element", "element", "element", "element"}) {
		t.Fatalf("parsed kinds = %v, want Parse, batch eligibility, invocation, and finalization all canonical", parsedKinds)
	}
	if !reflect.DeepEqual(finalizedKinds, []string{"element"}) {
		t.Fatalf("finalized kinds = %v, want [element]", finalizedKinds)
	}
}

func parameterByJSONName(t *testing.T, entry operation.Entry, name string) operation.ParameterContract {
	t.Helper()
	for _, parameter := range entry.Params {
		if parameter.JSONName == name {
			return parameter
		}
	}
	t.Fatalf("%s has no %q parameter", entry.Key, name)
	return operation.ParameterContract{}
}

func cloneInput(input map[string]any) map[string]any {
	cloned := make(map[string]any, len(input))
	maps.Copy(cloned, input)
	return cloned
}
