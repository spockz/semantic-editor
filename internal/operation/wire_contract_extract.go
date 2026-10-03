// WHY: This file keeps interface extraction's registration contract beside its type-checked execution path.

package operation

import (
	"context"
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"

	"semedit/internal/backend"
	"semedit/internal/contractplan"
)

func runExtractInterface(ctx context.Context, cc CallContext, request extractInterfaceRequest) (contractOutput, error) {
	workspace := cc.WorkDir
	if workspace == "" {
		workspace = cc.Project.RootDir
	}
	selected, err := canonicalContractPath(workspace, request.File)
	if err != nil {
		return contractOutput{}, err
	}
	destinationInput := request.DestinationFile
	if destinationInput == "" {
		destinationInput = request.File
	}
	destination, err := canonicalContractPath(workspace, destinationInput)
	if err != nil {
		return contractOutput{}, err
	}
	selectedPaths := []string{selected, destination}
	for _, member := range request.Members {
		selectedPaths = append(selectedPaths, member.File)
	}
	snapshot, err := loadContractSnapshot(ctx, workspace, selectedPaths)
	if err != nil {
		return contractOutput{}, err
	}
	pkg, concrete, err := findContractType(snapshot, selected, request.InType)
	if err != nil {
		return contractOutput{}, err
	}
	if snapshot.packageForFile(destination) != pkg {
		return contractOutput{}, fmt.Errorf("interface destination must be an existing file in the concrete type package")
	}
	methods, err := resolveContractMembers(snapshot, pkg, concrete, request.Members)
	if err != nil {
		return contractOutput{}, err
	}
	imports, err := contractImportBindings(pkg, snapshot.Syntax[destination])
	if err != nil {
		return contractOutput{}, err
	}
	plan, err := contractplan.PlanExtractInterface(pkg.Types, pkg.Types, concrete, request.ReceiverKind, methods, request.Symbol, imports)
	if err != nil {
		return contractOutput{}, err
	}
	return finishContractPlan(ctx, snapshot, destination, selected, request.Symbol, request.ExpectedRevision, plan, func(finalPackage *packages.Package) error {
		generated, err := contractTypeInFile(finalPackage, destination, request.Symbol)
		if err != nil {
			return err
		}
		interfaceType, ok := generated.Underlying().(*types.Interface)
		if !ok || !interfaceType.IsMethodSet() {
			return fmt.Errorf("generated type %q is not a runtime interface", request.Symbol)
		}
		finalConcrete, err := contractTypeInFile(finalPackage, selected, request.InType)
		if err != nil {
			return err
		}
		var receiver types.Type = finalConcrete
		if request.ReceiverKind == contractplan.ReceiverPointer {
			receiver = types.NewPointer(receiver)
		}
		if !types.Implements(receiver, interfaceType) {
			return fmt.Errorf("generated interface %q is not implemented by the requested receiver", request.Symbol)
		}
		return nil
	})
}

func extractInterfaceDef() Def[extractInterfaceRequest, contractOutput] {
	return Def[extractInterfaceRequest, contractOutput]{
		Key: "extract_interface", Summary: "Use this tool instead of reconstructing method signatures to create a Go interface from exact selected direct methods. Choose a new interface symbol; in_type and typed members identify its concrete source and methods. destination_file defaults to file and must already exist in the same package. Copy result.contract and result.contract_file to semantic_implement_contract.contract and contract_file when those aliases accompany the finalized live interface continuation. Buffered working results omit the finalized aliases; batch children use static parameters. Use live receipt continuations for later inspect or rename calls. Generated declarations are type-checked against the current source overlay before one apply; this operation does not implement methods or rewrite consumers.",
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
