// WHY: This file keeps interface extraction's registration contract beside its type-checked execution path.

package operation

import (
	"context"

	"semedit/internal/backend"
	"semedit/internal/contractedit"
)

func runExtractInterface(ctx context.Context, cc CallContext, request extractInterfaceRequest) (contractOutput, error) {
	workspace := cc.WorkDir
	if workspace == "" {
		workspace = cc.Project.RootDir
	}
	result, err := contractedit.ExtractInterface(ctx, contractedit.ExtractRequest{
		WorkspaceRoot:    workspace,
		File:             request.File,
		InType:           request.InType,
		Symbol:           request.Symbol,
		DestinationFile:  request.DestinationFile,
		ReceiverKind:     request.ReceiverKind,
		Members:          request.Members,
		ExpectedRevision: request.ExpectedRevision,
	})
	if err != nil {
		return contractOutput{}, err
	}
	return adaptContractResult(result), nil
}

func extractInterfaceDef() Def[extractInterfaceRequest, contractOutput] {
	return Def[extractInterfaceRequest, contractOutput]{
		Key: "extract_interface", Summary: "Use this tool instead of reconstructing method signatures to create a Go interface from exact selected direct methods. Choose a new interface symbol; in_type and typed members identify its concrete source and methods. destination_file defaults to file and must already exist in the same package. Copy result.contract and result.contract_file to semantic_implement_contract.contract and contract_file when those aliases accompany the finalized live interface continuation. Buffered working results omit the finalized aliases; batch children use static parameters. Use live receipt continuations for later inspect or rename calls. Generated declarations are type-checked against the current source overlay before one apply; mutable local replacement modules are unsupported, while versioned replacements remain supported.",
		Params: extractInterfaceParams, Level: LevelSymbol, CLIName: "extract-interface", MCPName: "semantic_extract_interface",
		Parse: parseExtractInterfaceRequest, Handlers: map[backend.LanguageID]func(context.Context, CallContext, extractInterfaceRequest) (contractOutput, error){backend.LanguageGo: runExtractInterface},
		Format: formatContractOutput, Output: extractInterfaceOutputContract(),
		FinalizeOutput: func(cc CallContext, _ extractInterfaceRequest, result contractOutput) (contractOutput, error) {
			result.Operation = "semantic_extract_interface"
			finalized, err := finalizeContractOutput(cc, nil, result)
			if err != nil {
				return contractOutput{}, err
			}
			for _, continuation := range finalized.Receipt.Continuations {
				if continuation.Symbol == finalized.Symbol {
					finalized.Contract = continuation.Symbol
					finalized.ContractFile = continuation.File
					break
				}
			}
			return finalized, nil
		},
		ProjectOutput: projectContractOutput, Batchable: true, BufferedBatch: true,
		ExampleRaw: map[string]any{"file": wireExampleFile, "in_type": "Store", "symbol": "StoreAPI", "receiver_kind": "pointer", "members": []any{map[string]any{"file": wireExampleFile, "symbol": "(*Store).Read"}}},
	}
}
