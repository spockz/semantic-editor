package operation

import "testing"

func TestDefaultRegistryPublishesOneCanonicalRefactoringIdentity(t *testing.T) {
	registry := DefaultRegistry()
	cases := []struct {
		key, cli, mcp string
	}{
		{"refactor_extract_callable", "refactor-extract-callable", "refactor_extract_callable"},
		{"refactor_extract_value", "refactor-extract-value", "refactor_extract_value"},
		{"refactor_inline_callable", "refactor-inline-callable", "refactor_inline_callable"},
		{"refactor_inline_value", "refactor-inline-value", "refactor_inline_value"},
		{"refactor_change_signature", "refactor-change-signature", "refactor_change_signature"},
		{"refactor_rename", "refactor-rename", "refactor_rename"},
		{"refactor_move_declaration", "refactor-move-declaration", "refactor_move_declaration"},
		{"refactor_move_file", "refactor-move-file", "refactor_move_file"},
		{"refactor_extract_interface", "refactor-extract-interface", "refactor_extract_interface"},
		{"refactor_implement_contract", "refactor-implement-contract", "refactor_implement_contract"},
		{"refactor_extract_type", "refactor-extract-type", "refactor_extract_type"},
		{"refactor_extract_delegate", "refactor-extract-delegate", "refactor_extract_delegate"},
		{"refactor_migrate_type", "refactor-migrate-type", "refactor_migrate_type"},
		{"refactor_introduce_parameter", "refactor-introduce-parameter", "refactor_introduce_parameter"},
		{"refactor_introduce_parameter_object", "refactor-introduce-parameter-object", "refactor_introduce_parameter_object"},
		{"refactor_encapsulate_field", "refactor-encapsulate-field", "refactor_encapsulate_field"},
		{"refactor_promote_local_to_const", "refactor-promote-local-to-const", "refactor_promote_local_to_const"},
	}
	for _, test := range cases {
		t.Run(test.key, func(t *testing.T) {
			byKey, ok := registry.LookupKey(test.key)
			if !ok {
				t.Fatalf("canonical key %q is not registered", test.key)
			}
			byCLI, ok := registry.LookupCLI(test.cli)
			if !ok || byCLI.Key != byKey.Key {
				t.Fatalf("canonical CLI command %q resolved to %+v, found=%t", test.cli, byCLI, ok)
			}
			byMCP, ok := registry.LookupMCP(test.mcp)
			if !ok || byMCP.Key != byKey.Key {
				t.Fatalf("canonical MCP tool %q resolved to %+v, found=%t", test.mcp, byMCP, ok)
			}
		})
	}
}

func TestRefactoringAliasesAreNotRegistered(t *testing.T) {
	registry := DefaultRegistry()
	for _, name := range []string{
		"semantic_extract_function", "extract-function", "extract_function",
		"semantic_extract_variable", "extract-variable", "extract_variable",
		"semantic_extract_constant", "extract-constant", "extract_constant", "refactor_extract_constant",
		"semantic_inline_function", "inline-function", "inline_function",
		"semantic_inline_variable", "inline-variable", "inline_variable",
		"semantic_change_signature", "change-signature", "change_signature",
		"semantic_rename", "rename",
		"semantic_move_construct", "move-construct", "move_construct",
		"semantic_move_file", "move-file", "move_file",
		"semantic_extract_interface", "extract-interface", "extract_interface",
		"semantic_implement_contract", "implement-contract", "implement_contract",
		"semantic_extract_type", "extract-type", "extract_type",
		"semantic_extract_delegate", "extract-delegate", "extract_delegate",
		"semantic_migrate_type", "migrate-type", "migrate_type",
		"semantic_introduce_parameter", "introduce-parameter", "introduce_parameter",
		"semantic_introduce_parameter_object", "introduce-parameter-object", "introduce_parameter_object",
		"semantic_encapsulate_field", "encapsulate-field", "encapsulate_field",
		"refactor_extract_function", "refactor_extract_method", "refactor_extract_variable",
		"refactor_introduce_variable", "refactor_inline",
	} {
		if _, ok := registry.LookupMCP(name); ok {
			t.Errorf("MCP alias %q is registered", name)
		}
		if _, ok := registry.LookupCLI(name); ok {
			t.Errorf("CLI alias %q is registered", name)
		}
		if _, ok := registry.LookupKey(name); ok {
			t.Errorf("operation key alias %q is registered", name)
		}
	}
}

func TestExtractValueSelectsConstantThroughKindParameter(t *testing.T) {
	entry, ok := DefaultRegistry().LookupMCP(canonicalExtractValueKey)
	if !ok {
		t.Fatal("canonical extract-value operation is not registered")
	}
	parsed, err := entry.Parse(map[string]any{
		"file": "service.go", "symbol": "retryWindow", "in_function": "Retry",
		"discriminator": "3 * 60", "kind": "constant",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	request, ok := parsed.(ExtractVariableReq)
	if !ok || request.Kind != "constant" {
		t.Fatalf("parsed request = %#v, want constant extraction", parsed)
	}
}
