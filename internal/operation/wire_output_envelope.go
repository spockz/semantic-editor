// Package operation declares MCP output envelope fields for legacy top-level continuations so consumers can validate them without synthesizing transport schemas.
package operation

var envelopeDiscriminatorConsumers = []string{
	semanticInsertCaseToolName, semanticInsertConstructToolName, semanticRemoveConstructToolName,
	semanticReplaceConstructToolName,
}

var envelopeFunctionConsumers = []string{
	"semantic_extract_constant", "semantic_extract_function", "semantic_extract_variable",
	semanticInlineVariableToolName, semanticInsertCaseToolName, semanticInsertConstructToolName,
	semanticRemoveConstructToolName, semanticReplaceConstructToolName,
}

var envelopeSymbolConsumers = []string{
	"semantic_change_signature", semanticFindReferencesToolName, semanticInlineFunctionToolName,
	"semantic_inline_variable", "semantic_inspect_symbol", "semantic_introduce_parameter",
	"semantic_introduce_parameter_object", "semantic_lookup", "semantic_migrate_type",
	semanticMoveConstructToolName, semanticRemoveConstructToolName, semanticRenameToolName,
	semanticReplaceBodyToolName, semanticReplaceConstructToolName, semanticReplaceDeclToolName,
}

var envelopeFileConsumers = []string{
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

func legacyEnvelopeOutput(continuations []EnvelopeContinuationField, metadata ...OutputMetadataField) *OperationOutputContract {
	return &OperationOutputContract{Metadata: metadata, EnvelopeContinuations: continuations}
}

func legacyEnvelopeField(field string, role ParameterRole, consumers []string) EnvelopeContinuationField {
	return EnvelopeContinuationField{Field: field, Schema: ValueSchema{Kind: SchemaString}, Role: role, Consumers: consumers, Optional: true}
}

func legacyEnvelopeFieldWithDescription(field string, role ParameterRole, consumers []string, description string) EnvelopeContinuationField {
	continuation := legacyEnvelopeField(field, role, consumers)
	continuation.Description = description
	return continuation
}
