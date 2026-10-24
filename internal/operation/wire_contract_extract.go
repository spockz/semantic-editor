// WHY: This file keeps interface extraction's registration contract beside its type-checked execution path.

package operation

import (
	"context"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/contractedit"
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
		Key: canonicalExtractInterfaceKey, Summary: "Use this tool instead of reconstructing method signatures to create an interface from exact selected direct methods. Go type-checks the generated declaration and supports current-overlay proof. Java selects public instance methods, appends a package-private interface to the concrete type file, adds implements, then formats and verifies through JDT LS. Copy result.contract and result.contract_file to refactor_implement_contract.contract and contract_file. Go mutable local replacement modules are unsupported; versioned replacements remain supported.",
		Params: extractInterfaceParams, Level: LevelSymbol, CLIName: "refactor-extract-interface", MCPName: canonicalExtractInterfaceKey,
		Parse: parseExtractInterfaceRequest, Handlers: map[backend.LanguageID]func(context.Context, CallContext, extractInterfaceRequest) (contractOutput, error){backend.LanguageGo: runExtractInterface, backend.LanguageJava: runExtractInterfaceJava},
		Format: formatContractOutput, Output: extractInterfaceOutputContract(),
		FinalizeOutput: func(cc CallContext, req extractInterfaceRequest, result contractOutput) (contractOutput, error) {
			result.Operation = canonicalExtractInterfaceKey
			cc.Project = effectiveProject(cc, req.Project)
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractInterfaceOutput(cc, result)
			}
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
