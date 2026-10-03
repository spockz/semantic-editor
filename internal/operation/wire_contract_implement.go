// Package operation keeps the contract implementation registration beside its typed request adapter.
// WHY: The registered definition is the public edge; contractedit owns package analysis and publication.
package operation

import (
	"context"

	"semedit/internal/backend"
	"semedit/internal/contractedit"
)

func runImplementContract(ctx context.Context, cc CallContext, request implementContractRequest) (contractOutput, error) {
	workspace := cc.WorkDir
	if workspace == "" {
		workspace = cc.Project.RootDir
	}
	result, err := contractedit.ImplementContract(ctx, contractedit.ImplementRequest{
		WorkspaceRoot:    workspace,
		File:             request.File,
		Symbol:           request.Symbol,
		Contract:         request.Contract,
		ContractFile:     request.ContractFile,
		DestinationFile:  request.DestinationFile,
		ReceiverKind:     request.ReceiverKind,
		BodyPolicy:       request.BodyPolicy,
		ExpectedRevision: request.ExpectedRevision,
	})
	if err != nil {
		return contractOutput{}, err
	}
	return adaptContractResult(result), nil
}

func implementContractDef() Def[implementContractRequest, contractOutput] {
	return Def[implementContractRequest, contractOutput]{
		Key:     "implement_contract",
		Summary: "Use this tool instead of rebuilding interface method signatures to generate missing Go methods on an existing concrete type. contract_file identifies an ordinary interface, receiver_kind selects pointer or value semantics, and body_policy must explicitly select panic_stub. Generated methods panic when called; they are placeholders, not completed implementations. destination_file must already exist in the concrete type package. Copy type_target to inspect the concrete type and each generated_methods file/symbol pair to replace_body or rename. Existing incompatible methods, unsupported promoted methods, failed required-package overlay type checks, and mutable local replacement modules are refused before one apply; versioned replacements remain supported.",
		Params:  implementContractParams, Level: LevelSymbol, CLIName: "implement-contract", MCPName: contractImplementTool,
		Parse: parseImplementContractRequest,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, implementContractRequest) (contractOutput, error){
			backend.LanguageGo: runImplementContract,
		},
		Format: formatContractOutput, Output: implementContractOutputContract(),
		FinalizeOutput: func(cc CallContext, _ implementContractRequest, result contractOutput) (contractOutput, error) {
			result.Operation = "semantic_implement_contract"
			return finalizeContractOutput(cc, nil, result)
		},
		ProjectOutput: projectContractOutput, Batchable: true, BufferedBatch: true,
		ExampleRaw: map[string]any{
			"file": wireExampleFile, "symbol": "Worker", "contract": "Runner", "contract_file": "runner.go",
			"receiver_kind": "pointer", "body_policy": "panic_stub",
		},
	}
}
