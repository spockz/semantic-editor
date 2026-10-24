// Package operation keeps typed output continuation consumer sets next to output contract construction.
package operation

var discriminatorOutputConsumers = []string{
	semanticInsertCaseToolName, semanticInsertConstructToolName, semanticRemoveConstructToolName,
	semanticInspectConstructToolName, semanticReplaceConstructToolName,
}

var functionOutputConsumers = []string{
	canonicalMakeTypeExplicitKey,
	semanticExtractFunctionToolName, semanticExtractVariableToolName,
	semanticInlineVariableToolName, semanticInsertCaseToolName, semanticInsertConstructToolName,
	semanticInspectConstructToolName, semanticRemoveConstructToolName, semanticReplaceConstructToolName,
}

var symbolOutputConsumers = []string{
	canonicalMakeTypeExplicitKey,
	canonicalChangeSignatureKey, semanticFindReferencesToolName, semanticInlineFunctionToolName,
	semanticInlineVariableToolName, semanticInspectSymbolToolName, "refactor_introduce_parameter",
	"refactor_introduce_parameter_object", semanticLookupToolName, semanticMigrateTypeToolName,
	semanticMoveConstructToolName, semanticRemoveConstructToolName, semanticRenameToolName,
	semanticReplaceBodyToolName, semanticReplaceConstructToolName, semanticReplaceDeclToolName,
	canonicalMakeStaticKey,
}

var fileOutputConsumers = []string{
	canonicalMakeTypeExplicitKey,
	semanticAssertionModeToolName, canonicalChangeSignatureKey,
	semanticExtractFunctionToolName, semanticExtractVariableToolName, semanticFindReferencesToolName,
	semanticInlineFunctionToolName, semanticInlineVariableToolName, semanticInsertCaseToolName,
	semanticInsertConstructToolName, semanticInsertDeclToolName, semanticInsertDeclarationToolName,
	semanticInsertFunctionToolName, semanticInsertTypeToolName, semanticInspectSymbolToolName,
	semanticInspectConstructToolName,
	"refactor_introduce_parameter", "refactor_introduce_parameter_object", semanticLookupToolName,
	semanticMigrateTypeToolName, semanticMoveConstructToolName, semanticRemoveConstructToolName,
	semanticRenameToolName, semanticReplaceBodyToolName, semanticReplaceConstructToolName,
	"semantic_replace_decl", "semantic_verify", canonicalMakeStaticKey,
}

func outputContinuation(field string, role ParameterRole, consumers []string) OutputContinuation {
	return OutputContinuation{Field: field, Schema: ValueSchema{Kind: SchemaString}, Role: role, Consumers: consumers, Optional: true}
}
