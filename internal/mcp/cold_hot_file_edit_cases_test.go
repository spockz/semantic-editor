// WHY: Declares the exhaustive Go file-edit benchmark cases separately from the cold/hot runner.

package mcp_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
)

type fileEditCostCase struct {
	name     string
	tool     string
	files    map[string]string
	args     map[string]any
	expected map[string][]string
	absent   map[string][]string
	prepare  func(testing.TB, managedMCPFixture, *warmMCPCostClient) map[string]any
	validate func(testing.TB, managedMCPFixture, int, []byte)
}

var fileEditCostCaseTable = []fileEditCostCase{
	{
		name: "rename", tool: "semantic_rename",
		files:    map[string]string{"cost_rename.go": "package sample\n\nfunc costRenameValue() int { return 1 }\nfunc costRenameCaller() int { return costRenameValue() }\n"},
		args:     map[string]any{"file": "cost_rename.go", "symbol": "costRenameValue", "to": "costRenamedValue"},
		expected: map[string][]string{"cost_rename.go": {"func costRenamedValue()", "return costRenamedValue()"}}, absent: map[string][]string{"cost_rename.go": {"costRenameValue"}},
	},
	{
		name: "replace_body", tool: "semantic_replace_body",
		files:    map[string]string{"cost_replace_body.go": "package sample\n\nfunc costBody() int { return 1 }\n"},
		args:     map[string]any{"file": "cost_replace_body.go", "symbol": "costBody", "body": "return 2"},
		expected: map[string][]string{"cost_replace_body.go": {"return 2"}}, absent: map[string][]string{"cost_replace_body.go": {"return 1"}},
	},
	{
		name: "organize_imports", tool: "semantic_organize_imports",
		files:    map[string]string{"cost_imports.go": "package sample\n\nimport \"fmt\"\n\nfunc costImports() int { return 1 }\n"},
		args:     map[string]any{"file": "cost_imports.go"},
		expected: map[string][]string{"cost_imports.go": {"func costImports() int"}}, absent: map[string][]string{"cost_imports.go": {"\"fmt\""}},
	},
	{
		name: "batch", tool: "semantic_batch",
		files: map[string]string{
			"cost_batch_one.go": "package sample\n\nfunc costBatchOne() int { return 1 }\n",
			"cost_batch_two.go": "package sample\n\nfunc costBatchTwo() int { return 10 }\n",
		},
		args: map[string]any{"edits": []any{
			map[string]any{"tool": "semantic_replace_body", "params": map[string]any{"file": "cost_batch_one.go", "symbol": "costBatchOne", "body": "return 2"}},
			map[string]any{"tool": "semantic_replace_body", "params": map[string]any{"file": "cost_batch_two.go", "symbol": "costBatchTwo", "body": "return 20"}},
		}},
		expected: map[string][]string{"cost_batch_one.go": {"return 2"}, "cost_batch_two.go": {"return 20"}},
	},
	{
		name: "assertion_mode", tool: "semantic_assertion_mode",
		files:    map[string]string{"cost_assert_test.go": "package sample\n\nimport \"testing\"\n\nfunc TestCost(t *testing.T) { t.Fatal(\"stop\") }\n"},
		args:     map[string]any{"file": "cost_assert_test.go", "mode": "relax", "trust_workspace": true},
		expected: map[string][]string{"cost_assert_test.go": {"t.Error(\"stop\")"}}, absent: map[string][]string{"cost_assert_test.go": {"t.Fatal("}},
	},
	{
		name: "insert_construct", tool: "semantic_insert_construct",
		files:    map[string]string{"cost_insert_construct.go": "package sample\n\nfunc costConstruct(v int) int {\n\treturn v\n}\n"},
		args:     map[string]any{"file": "cost_insert_construct.go", "kind": "statement/expression", "in_function": "costConstruct", "discriminator": "return v", "source": "v++", "anchor": "before"},
		expected: map[string][]string{"cost_insert_construct.go": {"v++", "return v"}},
	},
	{
		name: "insert_construct_element", tool: "semantic_insert_construct",
		files:    map[string]string{"cost_insert_element.go": "package sample\n\nvar costLabels = map[string]int{\"one\": 1}\n"},
		args:     map[string]any{"file": "cost_insert_element.go", "kind": "element", "in_declaration": "costLabels", "discriminator": "map[string]int{\"one\": 1}", "source": "\"two\": 2"},
		expected: map[string][]string{"cost_insert_element.go": {"\"two\": 2"}},
	},
	{
		name: "insert_declaration", tool: "semantic_insert_declaration",
		files:    map[string]string{"cost_insert_declaration.go": "package sample\n"},
		args:     map[string]any{"file": "cost_insert_declaration.go", "source": "func costGeneric() int { return 2 }"},
		expected: map[string][]string{"cost_insert_declaration.go": {"func costGeneric() int", "return 2"}},
	},
	{
		name: "insert_function", tool: "semantic_insert_function",
		files:    map[string]string{"cost_insert_function.go": "package sample\n"},
		args:     map[string]any{"file": "cost_insert_function.go", "source": "func costAdded() int { return 2 }"},
		expected: map[string][]string{"cost_insert_function.go": {"func costAdded() int", "return 2"}},
	},
	{
		name: "insert_type", tool: "semantic_insert_type",
		files:    map[string]string{"cost_insert_type.go": "package sample\n"},
		args:     map[string]any{"file": "cost_insert_type.go", "source": "type costAddedType struct{ value int }"},
		expected: map[string][]string{"cost_insert_type.go": {"type costAddedType struct", "value int"}},
	},
	{
		name: "insert_decl", tool: "semantic_insert_decl",
		files:    map[string]string{"cost_insert_decl.go": "package sample\n"},
		args:     map[string]any{"file": "cost_insert_decl.go", "source": "const costAddedValue = 2"},
		expected: map[string][]string{"cost_insert_decl.go": {"const costAddedValue = 2"}},
	},
	{
		name: "replace_decl", tool: "semantic_replace_decl",
		files:    map[string]string{"cost_replace_decl.go": "package sample\n\nconst costLimit = 1\n"},
		args:     map[string]any{"file": "cost_replace_decl.go", "symbol": "costLimit", "source": "const costLimit = 2"},
		expected: map[string][]string{"cost_replace_decl.go": {"const costLimit = 2"}}, absent: map[string][]string{"cost_replace_decl.go": {"const costLimit = 1"}},
	},
	{
		name: "remove_construct", tool: "semantic_remove_construct",
		files:    map[string]string{"cost_remove.go": "package sample\n\nfunc costObsolete() int { return 1 }\nfunc costKeeper() int { return 2 }\n"},
		args:     map[string]any{"file": "cost_remove.go", "kind": "function", "symbol": "costObsolete", "safe_delete": false},
		expected: map[string][]string{"cost_remove.go": {"func costKeeper() int"}}, absent: map[string][]string{"cost_remove.go": {"costObsolete"}},
	},
	{
		name: "replace_construct", tool: "semantic_replace_construct",
		files:    map[string]string{"cost_replace_construct.go": "package sample\n\nfunc costCheck(v int) int {\n\tif v > 0 { return v }\n\treturn 0\n}\n"},
		args:     map[string]any{"file": "cost_replace_construct.go", "in_function": "costCheck", "kind": "if", "discriminator": "v > 0", "source": "if v >= 0 { return v }"},
		expected: map[string][]string{"cost_replace_construct.go": {"if v >= 0"}}, absent: map[string][]string{"cost_replace_construct.go": {"if v > 0"}},
	},
	{
		name: "move_construct", tool: "semantic_move_construct",
		files: map[string]string{
			"cost_move_source.go":      "package sample\n\nimport \"strings\"\n\nfunc costNormalize(value string) string { return strings.TrimSpace(value) }\n",
			"cost_move_destination.go": "package sample\n\nfunc costDestination() {}\n",
		},
		args:     map[string]any{"file": "cost_move_source.go", "destination_file": "cost_move_destination.go", "symbol": "costNormalize", "placement": "file_end"},
		expected: map[string][]string{"cost_move_destination.go": {"func costNormalize", "\"strings\""}}, absent: map[string][]string{"cost_move_source.go": {"costNormalize", "\"strings\""}},
	},
	{
		name: "move_file", tool: "semantic_move_file",
		files: map[string]string{
			"cost_move_file.go":         "package sample\n\nfunc CostMovedValue() int { return 2 }\n",
			"cost_move_target/value.go": "package moved\n",
		},
		args:     map[string]any{"file": "cost_move_file.go", "destination_file": "cost_move_target/value.go", "move_matching_tests": false},
		expected: map[string][]string{"cost_move_target/value.go": {"func CostMovedValue() int", "return 2"}},
		absent:   map[string][]string{"cost_move_file.go": {"CostMovedValue"}},
	},
	{
		name: "extract_function", tool: "semantic_extract_function",
		files:    map[string]string{"cost_extract_function.go": "package sample\n\nimport \"fmt\"\n\nfunc costEmit() {\n\tfmt.Println(\"selected\")\n\tfmt.Println(\"after\")\n}\n"},
		args:     map[string]any{"file": "cost_extract_function.go", "symbol": "costSendSelected", "in_function": "costEmit", "kind": "function", "start_line": 6, "end_line": 6},
		expected: map[string][]string{"cost_extract_function.go": {"func costSendSelected()", "costSendSelected()"}},
	},
	{
		name: "extract_variable", tool: "semantic_extract_variable",
		files:    map[string]string{"cost_extract_variable.go": "package sample\n\nfunc costConsume(int) {}\nfunc costRender(distance int) { costConsume(distance) }\n"},
		args:     map[string]any{"file": "cost_extract_variable.go", "symbol": "costDistance", "in_function": "costRender", "discriminator": "distance"},
		expected: map[string][]string{"cost_extract_variable.go": {"costDistance := distance", "costConsume(costDistance)"}},
	},
	{
		name: "extract_constant", tool: "semantic_extract_constant",
		files:    map[string]string{"cost_extract_constant.go": "package sample\n\nfunc costRetry() int { return 3 * 60 }\n"},
		args:     map[string]any{"file": "cost_extract_constant.go", "symbol": "costRetryWindow", "in_function": "costRetry", "discriminator": "3 * 60"},
		expected: map[string][]string{"cost_extract_constant.go": {"const costRetryWindow", "return costRetryWindow"}},
	},
	{
		name: "inline_variable", tool: "semantic_inline_variable",
		files:    map[string]string{"cost_inline_variable.go": "package sample\n\nfunc costInlineValue() int {\n\tvalue := 2\n\treturn value\n}\n"},
		args:     map[string]any{"file": "cost_inline_variable.go", "symbol": "value", "in_function": "costInlineValue"},
		expected: map[string][]string{"cost_inline_variable.go": {"return int(2)"}}, absent: map[string][]string{"cost_inline_variable.go": {"value :="}},
	},
	{
		name: "inline_function", tool: "semantic_inline_function",
		files: map[string]string{
			"cost_inline_callee.go": "package sample\n\nfunc costAdjust() int { return 2 }\n",
			"cost_inline_caller.go": "package sample\n\nfunc costRun() int { return costAdjust() }\n",
		},
		args:     map[string]any{"file": "cost_inline_callee.go", "symbol": "costAdjust", "call_site": map[string]any{"file": "cost_inline_caller.go", "in_function": "costRun", "discriminator": "costAdjust()"}},
		expected: map[string][]string{"cost_inline_callee.go": {"func costAdjust() int"}, "cost_inline_caller.go": {"return int(2)"}}, absent: map[string][]string{"cost_inline_caller.go": {"costAdjust()"}},
	},
	{
		name: "migrate_type", tool: "semantic_migrate_type",
		files:    map[string]string{"cost_migrate_type.go": "package sample\n\ntype costDuration = int64\ntype costSettings struct { timeout int64 }\nfunc costTimeout(s costSettings) int64 { return s.timeout }\n"},
		args:     map[string]any{"file": "cost_migrate_type.go", "symbol": "costSettings.timeout", "type": "costDuration"},
		expected: map[string][]string{"cost_migrate_type.go": {"timeout costDuration"}}, absent: map[string][]string{"cost_migrate_type.go": {"timeout int64"}},
	},
	{
		name: "scaffold_file", tool: "semantic_scaffold_file",
		args:     map[string]any{"file": "cost_scaffolded.go", "package": "sample", "purpose_header": "WHY: Holds the generated benchmark declaration."},
		expected: map[string][]string{"cost_scaffolded.go": {"package sample", "WHY: Holds the generated benchmark declaration."}},
	},
	{
		name: "insert_case", tool: "semantic_insert_case",
		files:    map[string]string{"cost_insert_case.go": "package sample\n\nfunc costSwitch(v int) int {\n\tswitch v {\n\tcase 1:\n\t\treturn 1\n\tdefault:\n\t\treturn 0\n\t}\n}\n"},
		args:     map[string]any{"file": "cost_insert_case.go", "in_function": "costSwitch", "discriminator": "v", "case": "case 2:\n\treturn 2", "placement": "before_default"},
		expected: map[string][]string{"cost_insert_case.go": {"case 2:", "return 2"}},
	},
	{
		name: "change_signature", tool: "semantic_change_signature",
		files: map[string]string{"cost_change_signature.go": "package sample\n\nfunc costTransform(id int) int { return id + 1 }\nfunc costTransformCaller() int { return costTransform(1) }\n"},
		args: map[string]any{"file": "cost_change_signature.go", "symbol": "costTransform", "parameters": []any{
			map[string]any{"source_index": 0, "parameter": map[string]any{"name": "id", "type": "int"}},
			map[string]any{"parameter": map[string]any{"name": "label", "type": "string"}, "argument": "\"ready\""},
		}},
		expected: map[string][]string{"cost_change_signature.go": {"costTransform(id int, label string)", "costTransform(1, \"ready\")"}},
	},
	{
		name: "introduce_parameter", tool: "semantic_introduce_parameter",
		files: map[string]string{"cost_introduce_parameter.go": "package sample\n\nfunc costProcess(id int) int { return id + 3 }\nfunc costProcessCaller() int { return costProcess(1) }\n"},
		args: map[string]any{"file": "cost_introduce_parameter.go", "symbol": "costProcess", "parameters": []any{
			map[string]any{"source_index": 0, "parameter": map[string]any{"name": "id", "type": "int"}},
			map[string]any{"parameter": map[string]any{"name": "mode", "type": "int"}, "argument": "3"},
		}, "discriminator": "3"},
		expected: map[string][]string{"cost_introduce_parameter.go": {"costProcess(id int, mode int)", "return id + mode", "costProcess(1, 3)"}},
	},
	{
		name: "introduce_parameter_object", tool: "semantic_introduce_parameter_object",
		files: map[string]string{"cost_parameter_object.go": "package sample\n\nfunc costWrite(path string, mode uint32) string { return path }\nfunc costWriteCaller() string { return costWrite(\"file\", 7) }\n"},
		args: map[string]any{"file": "cost_parameter_object.go", "symbol": "costWrite", "destination_file": "cost_parameter_object.go", "parameter": map[string]any{"name": "options", "type": "costWriteOptions"}, "fields": []any{
			map[string]any{"source_index": 0, "field": map[string]any{"name": "path", "type": "string"}},
			map[string]any{"source_index": 1, "field": map[string]any{"name": "mode", "type": "uint32"}},
		}},
		expected: map[string][]string{"cost_parameter_object.go": {"type costWriteOptions struct", "options costWriteOptions", "options.path", "costWriteOptions{"}},
	},
	{
		name: "insert_field", tool: "semantic_insert_field",
		files:    map[string]string{"cost_insert_field.go": "package sample\n\ntype costConfig struct {\n\tvalue int\n}\n"},
		args:     map[string]any{"file": "cost_insert_field.go", "in_type": "costConfig", "symbol": "costConfig.timeout", "source": "timeout int", "placement": "type_end"},
		expected: map[string][]string{"cost_insert_field.go": {"timeout int"}},
	},
	{
		name: "extract_type", tool: "semantic_extract_type",
		files:    map[string]string{"cost_extract_type.go": "package sample\n\ntype costCache struct {\n\tentries map[string]int\n\tsize int\n}\nfunc costCacheSize(cache costCache) int { return cache.size }\n"},
		args:     map[string]any{"file": "cost_extract_type.go", "symbol": "costCacheState", "in_type": "costCache", "members": []any{map[string]any{"file": "cost_extract_type.go", "symbol": "costCache.size"}}, "field": "state"},
		expected: map[string][]string{"cost_extract_type.go": {"type costCacheState struct", "state costCacheState", "cache.state.size"}},
	},
	{
		name: "set_doc_comment", tool: "semantic_set_doc_comment",
		files:    map[string]string{"cost_doc.go": "package sample\n\nfunc costServe() {}\n"},
		args:     map[string]any{"file": "cost_doc.go", "scope": "declaration", "symbol": "costServe", "comment": "costServe handles benchmark requests."},
		expected: map[string][]string{"cost_doc.go": {"// costServe handles benchmark requests.", "func costServe()"}},
	},
	{
		name: "extract_interface", tool: "semantic_extract_interface",
		files:    map[string]string{"cost_extract_interface.go": "package sample\n\ntype costStore struct{}\nfunc (*costStore) Read() int { return 1 }\n"},
		args:     map[string]any{"file": "cost_extract_interface.go", "in_type": "costStore", "symbol": "costStoreAPI", "receiver_kind": "pointer", "members": []any{map[string]any{"file": "cost_extract_interface.go", "symbol": "(*costStore).Read"}}},
		expected: map[string][]string{"cost_extract_interface.go": {"type costStoreAPI interface", "Read() int"}},
	},
	{
		name: "implement_contract", tool: "semantic_implement_contract",
		files:    map[string]string{"cost_implement_contract.go": "package sample\n\ntype costRunner interface { Run() error }\ntype costWorker struct{}\n"},
		args:     map[string]any{"file": "cost_implement_contract.go", "symbol": "costWorker", "contract": "costRunner", "contract_file": "cost_implement_contract.go", "destination_file": "cost_implement_contract.go", "receiver_kind": "pointer", "body_policy": "panic_stub"},
		expected: map[string][]string{"cost_implement_contract.go": {"func (", "*costWorker", "Run() error", "panic("}},
	},
	{
		name: "undo", tool: "semantic_undo",
		files:    map[string]string{"value.go": "package sample\n\nfunc Value() int { return 1 }\n"},
		args:     map[string]any{"snapshot_id": "latest"},
		expected: map[string][]string{"value.go": {"return 1"}}, absent: map[string][]string{"value.go": {"return 9"}},
	},
}

func TestGoSourceEditBenchmarkCoverage(t *testing.T) {
	registry := operation.DefaultRegistry()
	cases := fileEditCostCases()
	classified := validateFileEditCostCaseSchemas(t, registry, cases)
	excluded := map[string]string{
		"semantic_add_build_dependency": "manifest and network operation outside source-edit costs",
		"semantic_snapshot":             "undo setup lifecycle operation",
		"semantic_stage":                "retained-session lifecycle operation",
		"semantic_stage_drop":           "retained-session lifecycle operation",
		"semantic_commit":               "retained-session lifecycle operation",
		"semantic_stage_abort":          "retained-session lifecycle operation",
		"semantic_verify":               "verification and normalization boundary",
	}
	for _, entry := range registry.All() {
		if entry.ReadOnly || !slices.Contains(entry.Languages, backend.LanguageGo) && !slices.Contains(entry.Languages, backend.LanguageAuto) {
			continue
		}
		if classified[entry.MCPName] {
			continue
		}
		if reason := excluded[entry.MCPName]; reason != "" {
			continue
		}
		t.Errorf("Go mutation %s (%s) lacks a file-edit benchmark case or explicit exclusion", entry.Key, entry.MCPName)
	}
}

func fileEditCostCases() []fileEditCostCase {
	cases := slices.Clone(fileEditCostCaseTable)
	for _, scenario := range fileEditCostCaseTable {
		if scenario.name != "remove_construct" {
			continue
		}
		safeDelete := scenario
		safeDelete.name = "remove_construct_safe"
		safeDelete.args = maps.Clone(scenario.args)
		safeDelete.args["safe_delete"] = true
		cases = append(cases, safeDelete)
		break
	}
	registry := operation.DefaultRegistry()
	for index := range cases {
		cases[index].args = explicitGoFileEditArgs(registry, cases[index].tool, cases[index].args)
	}
	return cases
}

func explicitGoFileEditArgs(registry *operation.Registry, tool string, args map[string]any) map[string]any {
	result := maps.Clone(args)
	if tool == "semantic_batch" {
		result["edits"] = explicitGoBatchEdits(registry, args["edits"])
		return result
	}
	entry, ok := registry.LookupMCP(tool)
	if !ok {
		return result
	}
	for _, parameter := range entry.Params {
		if parameter.JSONName == "language" {
			result["language"] = "go"
			break
		}
	}
	return result
}

func explicitGoBatchEdits(registry *operation.Registry, value any) any {
	edits, ok := value.([]any)
	if !ok {
		return value
	}
	result := slices.Clone(edits)
	for index, raw := range edits {
		edit, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cloned := maps.Clone(edit)
		tool, _ := edit["tool"].(string)
		params, _ := edit["params"].(map[string]any)
		cloned["params"] = explicitGoFileEditArgs(registry, tool, params)
		result[index] = cloned
	}
	return result
}

func validateFileEditCostCaseSchemas(t *testing.T, registry *operation.Registry, cases []fileEditCostCase) map[string]bool {
	t.Helper()
	classified := make(map[string]bool, len(cases))
	names := make(map[string]bool, len(cases))
	for _, scenario := range cases {
		if names[scenario.name] {
			t.Errorf("duplicate file-edit benchmark case name %q", scenario.name)
		}
		names[scenario.name] = true
		if scenario.tool == "semantic_batch" {
			edits, ok := scenario.args["edits"].([]any)
			if !ok {
				t.Errorf("%s edits have type %T, want []any", scenario.name, scenario.args["edits"])
				continue
			}
			for index, raw := range edits {
				edit, ok := raw.(map[string]any)
				if !ok {
					t.Errorf("%s edit %d has type %T", scenario.name, index, raw)
					continue
				}
				tool, _ := edit["tool"].(string)
				params, _ := edit["params"].(map[string]any)
				entry, ok := registry.LookupMCP(tool)
				if !ok {
					t.Errorf("%s edit %d names unregistered tool %q", scenario.name, index, tool)
					continue
				}
				if _, err := entry.Parse(params); err != nil {
					t.Errorf("%s edit %d arguments: %v", scenario.name, index, err)
				}
			}
			continue
		}
		entry, ok := registry.LookupMCP(scenario.tool)
		if !ok {
			t.Errorf("%s names unregistered tool %q", scenario.name, scenario.tool)
			continue
		}
		if _, err := entry.Parse(scenario.args); err != nil {
			t.Errorf("%s arguments: %v", scenario.name, err)
		}
		classified[scenario.tool] = true
	}
	return classified
}
