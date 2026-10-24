// Package operation adapts public refactoring names to backend operation identifiers.
// WHY: The registry owns canonical user-facing identities while language services keep their operation vocabulary.
package operation

const (
	backendExtractFunctionOperation = "extract_function"
	backendExtractValueOperation    = "extract_variable"

	backendInlineFunctionOperation     = "inline_function"
	backendInlineValueOperation        = "inline_variable"
	backendChangeSignatureOperation    = "change_signature"
	backendRenameOperation             = "rename"
	backendMoveDeclarationOperation    = "move_construct"
	backendMoveFileOperation           = "move_file"
	backendExtractInterfaceOperation   = "extract_interface"
	backendImplementContractOperation  = "implement_contract"
	backendExtractTypeOperation        = "extract_type"
	backendExtractDelegateOperation    = "extract_delegate"
	backendMigrateTypeOperation        = "migrate_type"
	backendIntroduceParameterOperation = "introduce_parameter"
	backendParameterObjectOperation    = "introduce_parameter_object"
	backendEncapsulateFieldOperation   = "encapsulate_field"
	canonicalImplementContractName     = "refactor_implement_contract"
	canonicalEncapsulateFieldKey       = "refactor_encapsulate_field"
	canonicalExtractDelegateKey        = "refactor_extract_delegate"
	canonicalInlineValueKey            = "refactor_inline_value"
	canonicalRenameKey                 = "refactor_rename"
	canonicalExtractCallableKey        = "refactor_extract_callable"
	canonicalExtractValueKey           = "refactor_extract_value"
	canonicalInlineCallableKey         = "refactor_inline_callable"
	canonicalExtractInterfaceKey       = "refactor_extract_interface"
	canonicalMigrateTypeKey            = "refactor_migrate_type"
	canonicalChangeSignatureKey        = "refactor_change_signature"
	canonicalMakeStaticKey             = "refactor_make_static"
	canonicalPromoteLocalToConstKey    = "refactor_promote_local_to_const"
)

const backendExtractConstantOperation = "extract_constant"

var backendOperationByPublicKey = map[string]string{
	canonicalExtractCallableKey:           backendExtractFunctionOperation,
	canonicalExtractValueKey:              backendExtractValueOperation,
	canonicalInlineCallableKey:            backendInlineFunctionOperation,
	canonicalInlineValueKey:               backendInlineValueOperation,
	canonicalChangeSignatureKey:           backendChangeSignatureOperation,
	canonicalRenameKey:                    backendRenameOperation,
	"refactor_move_declaration":           backendMoveDeclarationOperation,
	"refactor_move_file":                  backendMoveFileOperation,
	canonicalExtractInterfaceKey:          backendExtractInterfaceOperation,
	canonicalImplementContractName:        backendImplementContractOperation,
	"refactor_extract_type":               backendExtractTypeOperation,
	canonicalExtractDelegateKey:           backendExtractDelegateOperation,
	canonicalMigrateTypeKey:               backendMigrateTypeOperation,
	"refactor_introduce_parameter":        backendIntroduceParameterOperation,
	"refactor_introduce_parameter_object": backendParameterObjectOperation,
	canonicalEncapsulateFieldKey:          backendEncapsulateFieldOperation,
	canonicalMakeStaticKey:                "make_static",
	canonicalMakeTypeExplicitKey:          "make_type_explicit",
	canonicalPromoteLocalToConstKey:       "promote_local_to_const",
}

func cloneOperationContracts[Req any, Res any](def *Def[Req, Res]) {
	if def == nil {
		return
	}
	def.Params = append([]ParameterContract(nil), def.Params...)
	for index := range def.Params {
		def.Params[index].SourceFields = append([]string(nil), def.Params[index].SourceFields...)
	}
	if def.Output == nil {
		return
	}
	output := *def.Output
	output.Continuations = append([]OutputContinuation(nil), def.Output.Continuations...)
	for index := range output.Continuations {
		output.Continuations[index].Consumers = append([]string(nil), output.Continuations[index].Consumers...)
	}
	def.Output = &output
}

func backendOperationKey(publicKey string) string {
	if backendKey, ok := backendOperationByPublicKey[publicKey]; ok {
		return backendKey
	}
	return publicKey
}

func publicOperationKeyFromHandler(operation string) string {
	if operation == backendExtractConstantOperation {
		return canonicalExtractValueKey
	}
	for publicKey, backendKey := range backendOperationByPublicKey {
		if operation == backendKey {
			return publicKey
		}
	}
	return operation
}
