// Package operation verifies ingress aliases are canonical before contracts and handlers run.
package operation_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	gobackend "github.com/spockz/semantic-editor/internal/backend/golang"
	"github.com/spockz/semantic-editor/internal/operation"
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

func TestNativeRefactorBatchEligibility(t *testing.T) {
	registry := operation.DefaultRegistry()
	cases := []struct {
		language backend.LanguageID
		key      string
		file     string
	}{
		{backend.LanguageRust, "refactor_rename", "src/lib.rs"},
		{backend.LanguageRust, "refactor_extract_value", "src/lib.rs"},
		{backend.LanguageRust, "refactor_extract_callable", "src/lib.rs"},
		{backend.LanguageRust, "refactor_inline_callable", "src/lib.rs"},
		{backend.LanguageRust, "refactor_inline_value", "src/lib.rs"},
		{backend.LanguageRust, "refactor_promote_local_to_const", "src/lib.rs"},
		{backend.LanguageScala, "refactor_extract_value", "src/Thing.scala"},
		{backend.LanguageScala, "refactor_extract_callable", "src/Thing.scala"},
	}
	for _, test := range cases {
		t.Run(string(test.language)+"/"+test.key, func(t *testing.T) {
			entry, ok := registry.LookupKey(test.key)
			if !ok {
				t.Fatalf("operation %q is not registered", test.key)
			}
			raw := make(map[string]any, len(entry.ExampleRaw)+2)
			maps.Copy(raw, entry.ExampleRaw)
			raw["file"] = test.file
			raw["language"] = string(test.language)
			switch test.key {
			case "refactor_extract_value":
				raw["in_function"] = "run"
				if test.language == backend.LanguageRust {
					raw["discriminator"] = "1_i32 + 2_i32"
				} else {
					raw["discriminator"] = "1 + 2"
				}
			case "refactor_extract_callable":
				raw["symbol"] = "extracted"
				raw["in_function"] = "run"
				raw["kind"] = "function"
				raw["start_line"], raw["end_line"] = 2, 2
			case "refactor_inline_callable":
				callSiteInput, ok := raw["call_site"].(map[string]any)
				if !ok {
					t.Fatal("inline-callable example has no call_site object")
				}
				callSite := maps.Clone(callSiteInput)
				callSite["file"] = test.file
				callSite["in_function"] = "run"
				callSite["discriminator"] = raw["symbol"].(string) + "()"
				raw["call_site"] = callSite
			case "refactor_inline_value":
				raw["in_function"] = "run"
			case "refactor_promote_local_to_const":
				raw["symbol"] = "retry_limit"
				raw["in_function"] = "run"
			}
			eligible, err := entry.SupportsBufferedBatch(operation.CallContext{
				Project: backend.ProjectContext{Language: test.language},
			}, raw)
			if err != nil {
				t.Fatalf("SupportsBufferedBatch() error = %v", err)
			}
			if !eligible {
				t.Fatal("SupportsBufferedBatch() = false, want shared-overlay eligibility")
			}
		})
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

func TestDirectOperationUnknownParameterErrorIsActionable(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_replace_body")
	if !ok {
		t.Fatal("semantic_replace_body is not registered")
	}
	_, err := entry.Parse(map[string]any{"flie": "value.go", "symbol": "Value", "body": "return 2"})
	if err == nil {
		t.Fatal("unknown parameter was accepted")
	}
	for _, part := range []string{`unknown parameter "flie"`, "supported keys", "Required parameters"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("parse error %q does not include %q", err, part)
		}
	}
}

func TestRenameUnknownNewNameParameterDoesNotSuggestTo(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("refactor_rename")
	if !ok {
		t.Fatal("refactor_rename is not registered")
	}
	_, err := entry.Parse(map[string]any{"symbol": "Value", "new_name": "RenamedValue"})
	if err == nil {
		t.Fatal("unknown new_name parameter was accepted")
	}
	if !strings.Contains(err.Error(), `unknown parameter "new_name"`) {
		t.Fatalf("parse error = %q, want unknown new_name parameter", err)
	}
	if !strings.Contains(err.Error(), `Required parameters: [to]`) {
		t.Fatalf("parse error = %q, want the missing canonical parameter to be reported", err)
	}
}

func TestJavaInsertConstructElementParserContract(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupKey("insert_construct")
	if !ok {
		t.Fatal("insert_construct is not registered")
	}
	parsed, err := entry.Parse(map[string]any{
		"file": "LedgerService.java", "language": "java", "trust_workspace": true,
		"jdtls_home": "/opt/jdtls", "kind": "element", "source": `"MONTHLY", 5000`,
		"in_declaration": "LedgerService.limits", "discriminator": `Map.of("DAILY", 1000)`,
	})
	if err != nil {
		t.Fatalf("Java collection element request rejected: %v", err)
	}
	request := parsed.(operation.InsertConstructReq)
	if request.Project.Language != backend.LanguageJava || request.Placement != "append" || request.InDeclaration != "LedgerService.limits" {
		t.Fatalf("Java element request = %#v", request)
	}
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
	type request struct {
		Kind string
		Path any
	}
	registry := operation.NewRegistry()
	var parsedKinds []string
	var parsedPaths []any
	var finalizedKinds []string
	var finalizedPaths []any
	def := operation.Def[request, string]{
		Key:           "normalize_input_probe",
		BufferedBatch: true,
		Params: []operation.ParameterContract{
			{Name: "kind", JSONName: "kind", Type: operation.ParamString, Required: true, Enums: []string{"element"}},
			{Name: "path", CLIName: "path", JSONName: "path", InputAliases: []string{"file"}, Type: operation.ParamString},
		},
		NormalizeInput: operation.NormalizeConstructKindInput,
		Parse: func(raw map[string]any) (request, error) {
			kind, _ := raw["kind"].(string)
			parsedKinds = append(parsedKinds, kind)
			parsedPaths = append(parsedPaths, raw["path"])
			return request{Kind: kind, Path: raw["path"]}, nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, request) (string, error){
			backend.LanguageGo: func(_ context.Context, _ operation.CallContext, req request) (string, error) {
				return req.Kind, nil
			},
		},
		FinalizeOutput: func(_ operation.CallContext, req request, result string) (string, error) {
			finalizedKinds = append(finalizedKinds, req.Kind)
			finalizedPaths = append(finalizedPaths, req.Path)
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
	if _, err := entry.Parse(map[string]any{"kind": "item", "file": "input.go"}); err != nil {
		t.Fatalf("Parse rejected aliases: %v", err)
	}
	if eligible, err := entry.SupportsBufferedBatch(operation.CallContext{Project: backend.ProjectContext{Language: backend.LanguageGo}}, map[string]any{"kind": "entry", "file": "batch.go"}); err != nil || !eligible {
		t.Fatalf("SupportsBufferedBatch = %v, %v; want true, nil", eligible, err)
	}
	result, err := registry.Dispatch(operation.CallContext{Ctx: context.Background(), Project: backend.ProjectContext{Language: backend.LanguageGo}}, def.Key,
		map[string]any{"kind": "literal_element", "file": "run.go"})
	if err != nil {
		t.Fatalf("Dispatch rejected aliases: %v", err)
	}
	if result != "element" {
		t.Fatalf("Dispatch result = %v, want canonical kind element", result)
	}
	if !reflect.DeepEqual(parsedKinds, []string{"element", "element", "element", "element"}) {
		t.Fatalf("parsed kinds = %v, want Parse, batch eligibility, invocation, and finalization all canonical", parsedKinds)
	}
	if !reflect.DeepEqual(parsedPaths, []any{"input.go", "batch.go", "run.go", "run.go"}) {
		t.Fatalf("parsed paths = %#v, want aliases folded before every registered ingress path", parsedPaths)
	}
	if !reflect.DeepEqual(finalizedKinds, []string{"element"}) || !reflect.DeepEqual(finalizedPaths, []any{"run.go"}) {
		t.Fatalf("finalized values = (%v, %#v), want ([element], [run.go])", finalizedKinds, finalizedPaths)
	}
	parse := func(raw map[string]any) error {
		_, parseErr := entry.Parse(raw)
		return parseErr
	}
	assertRegisteredPathAliases(t, parse, &parsedPaths)
}

func TestNormalizeConstructKindVocabularyAndMethods(t *testing.T) {
	for alias, want := range map[string]string{
		" Variable ":        "var",
		"CONSTANT":          "const",
		"procedure":         "function",
		" Element ":         "element",
		"composite_element": "element",
		"literal_element":   "element",
		"entry":             "element",
		"item":              "element",
	} {
		got, err := operation.NormalizeConstructKindInput(map[string]any{"kind": alias})
		if err != nil {
			t.Fatalf("NormalizeConstructKindInput(%q): %v", alias, err)
		}
		if got["kind"] != want {
			t.Errorf("NormalizeConstructKindInput(%q).kind = %v, want %q", alias, got["kind"], want)
		}
	}
	for _, test := range []struct {
		name   string
		symbol string
		source string
		want   string
	}{
		{name: "qualified pointer method", symbol: "(*Server).Start", source: "func (s *Server) Start() {}", want: "method"},
		{name: "qualified value method", symbol: "worker.Flush", source: "func (w worker) Flush() {}", want: "method"},
		{name: "ordinary function", symbol: "Normalize", source: "func Normalize() {}", want: "function"},
		{name: "receiverless source", symbol: "(*Server).Start", source: "func Start() {}", want: "function"},
		{name: "different declaration name", symbol: "(*Server).Start", source: "func (s *Server) Stop() {}", want: "function"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := operation.NormalizeConstructKindInput(map[string]any{
				"kind": " FUNCTION ", "symbol": test.symbol, "source": test.source,
			})
			if err != nil {
				t.Fatalf("NormalizeConstructKindInput: %v", err)
			}
			if got["kind"] != test.want {
				t.Errorf("normalized kind = %v, want %q", got["kind"], test.want)
			}
		})
	}
	if _, err := operation.NormalizeConstructKindInput(map[string]any{
		"kind": "function", "symbol": "(*Server).Start", "source": "func (",
	}); !errors.Is(err, operation.ErrInvalidParams) {
		t.Fatalf("malformed replacement source error = %v, want ErrInvalidParams", err)
	}
}

func TestNormalizeConstructKindRejectsMalformedSymbolSelector(t *testing.T) {
	for _, selector := range []string{"(*Server).", ".Start", "pkg..Start", "(*).Start"} {
		if _, err := operation.NormalizeConstructKindInput(map[string]any{
			"kind": "function", "symbol": selector, "source": "func Start() {}",
		}); !errors.Is(err, operation.ErrInvalidParams) {
			t.Fatalf("malformed symbol selector %q error = %v, want ErrInvalidParams", selector, err)
		}
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

func assertRegisteredPathAliases(t *testing.T, parse func(map[string]any) error, parsedPaths *[]any) {
	t.Helper()
	if err := parse(map[string]any{"kind": "element", "path": "canonical.go", "file": "alias.go"}); !errors.Is(err, operation.ErrInvalidParams) {
		t.Fatalf("conflicting canonical and alias values error = %v, want ErrInvalidParams", err)
	}
	if err := parse(map[string]any{"kind": "element", "path": "same.go", "file": "same.go"}); err != nil {
		t.Fatalf("equal canonical and alias values rejected: %v", err)
	}
	if err := parse(map[string]any{"kind": "element", "path": nil, "file": "ignored.go"}); err != nil {
		t.Fatalf("explicit null canonical value rejected: %v", err)
	}
	if (*parsedPaths)[len(*parsedPaths)-1] != nil {
		t.Fatalf("explicit null canonical value fell through to alias: %#v", (*parsedPaths)[len(*parsedPaths)-1])
	}
	if err := parse(map[string]any{"kind": "element", "file": nil}); err != nil {
		t.Fatalf("null alias rejected before validation: %v", err)
	}
	if (*parsedPaths)[len(*parsedPaths)-1] != nil {
		t.Fatalf("null alias did not reach canonical parser value: %#v", (*parsedPaths)[len(*parsedPaths)-1])
	}
	if err := parse(map[string]any{"kind": "element", "path": " ", "file": "pkg"}); err != nil {
		t.Fatalf("blank canonical value with nonblank alias rejected: %v", err)
	}
	if (*parsedPaths)[len(*parsedPaths)-1] != "pkg" {
		t.Fatalf("blank canonical value did not adopt nonblank alias: %#v", (*parsedPaths)[len(*parsedPaths)-1])
	}
}
