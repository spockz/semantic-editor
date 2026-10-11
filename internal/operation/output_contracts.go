// Package operation keeps typed output continuation consumer sets next to output contract construction.
package operation

var discriminatorOutputConsumers = []string{
	semanticInsertCaseToolName, semanticInsertConstructToolName, semanticRemoveConstructToolName,
	semanticReplaceConstructToolName,
}

var functionOutputConsumers = []string{
	"semantic_extract_constant", "semantic_extract_function", "semantic_extract_variable",
	semanticInlineVariableToolName, semanticInsertCaseToolName, semanticInsertConstructToolName,
	semanticRemoveConstructToolName, semanticReplaceConstructToolName,
}

var symbolOutputConsumers = []string{
	"semantic_change_signature", semanticFindReferencesToolName, semanticInlineFunctionToolName,
	"semantic_inline_variable", "semantic_inspect_symbol", "semantic_introduce_parameter",
	"semantic_introduce_parameter_object", "semantic_lookup", "semantic_migrate_type",
	semanticMoveConstructToolName, semanticRemoveConstructToolName, semanticRenameToolName,
	semanticReplaceBodyToolName, semanticReplaceConstructToolName, semanticReplaceDeclToolName,
}

var fileOutputConsumers = []string{
	semanticAssertionModeToolName, "semantic_change_signature", semanticExtractConstantToolName,
	semanticExtractFunctionToolName, semanticExtractVariableToolName, semanticFindReferencesToolName,
	semanticInlineFunctionToolName, semanticInlineVariableToolName, semanticInsertCaseToolName,
	semanticInsertConstructToolName, semanticInsertDeclToolName, semanticInsertDeclarationToolName,
	semanticInsertFunctionToolName, semanticInsertTypeToolName, semanticInspectSymbolToolName,
	"semantic_introduce_parameter", "semantic_introduce_parameter_object", "semantic_lookup",
	semanticMigrateTypeToolName, semanticMoveConstructToolName, semanticRemoveConstructToolName,
	semanticRenameToolName, semanticReplaceBodyToolName, semanticReplaceConstructToolName,
	"semantic_replace_decl", "semantic_verify",
}

func outputContinuation(field string, role ParameterRole, consumers []string) OutputContinuation {
	return OutputContinuation{Field: field, Schema: ValueSchema{Kind: SchemaString}, Role: role, Consumers: consumers, Optional: true}
}
