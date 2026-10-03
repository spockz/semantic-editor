// WHY: This file keeps interface implementation's registry definition and type-checked execution path together.

package operation

import (
	"context"
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"

	"semedit/internal/backend"
	"semedit/internal/contractplan"
)

func runImplementContract(ctx context.Context, cc CallContext, request implementContractRequest) (contractOutput, error) {
	workspace := cc.WorkDir
	if workspace == "" {
		workspace = cc.Project.RootDir
	}
	selected, err := canonicalContractPath(workspace, request.File)
	if err != nil {
		return contractOutput{}, err
	}
	contractInput := request.ContractFile
	if contractInput == "" {
		contractInput = request.File
	}
	contractPath, err := canonicalContractPath(workspace, contractInput)
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
	snapshot, err := loadContractSnapshot(ctx, workspace, []string{selected, contractPath, destination})
	if err != nil {
		return contractOutput{}, err
	}
	concretePackage, concrete, err := findContractType(snapshot, selected, request.Symbol)
	if err != nil {
		return contractOutput{}, err
	}
	contractPackage, contract, err := findContractType(snapshot, contractPath, request.Contract)
	if err != nil {
		return contractOutput{}, err
	}
	if snapshot.packageForFile(destination) != concretePackage {
		return contractOutput{}, fmt.Errorf("method destination must be an existing file in the concrete type package")
	}
	imports, err := contractImportBindings(concretePackage, snapshot.Syntax[destination])
	if err != nil {
		return contractOutput{}, err
	}
	plan, err := contractplan.PlanImplementContract(concretePackage.Types, concretePackage.Types, concrete, contract, request.ReceiverKind, request.BodyPolicy, imports)
	if err != nil {
		return contractOutput{}, err
	}
	if plan.Declaration != "" {
		if err := requirePanicBuiltin(concretePackage, imports); err != nil {
			return contractOutput{}, err
		}
	}
	result, err := finishContractPlan(ctx, snapshot, destination, selected, request.Symbol, request.ExpectedRevision, plan, func(finalPackage *packages.Package) error {
		finalConcrete, err := contractTypeInFile(finalPackage, selected, request.Symbol)
		if err != nil {
			return fmt.Errorf("final concrete type identity: %w", err)
		}
		finalContractPackage := snapshot.ByPath[contractPackage.PkgPath]
		if finalContractPackage == nil || finalContractPackage.Types == nil {
			return fmt.Errorf("contract package %q disappeared from the final package load", contractPackage.PkgPath)
		}
		finalContract, err := contractTypeInFile(finalContractPackage, contractPath, request.Contract)
		if err != nil {
			return err
		}
		interfaceType, ok := finalContract.Underlying().(*types.Interface)
		if !ok || !interfaceType.IsMethodSet() {
			return fmt.Errorf("contract %q is not a runtime interface", request.Contract)
		}
		receiver := types.Type(finalConcrete)
		if request.ReceiverKind == contractplan.ReceiverPointer {
			receiver = types.NewPointer(receiver)
		}
		if !types.Implements(receiver, interfaceType) {
			return fmt.Errorf("generated methods do not make %s implement %q", request.ReceiverKind, request.Contract)
		}
		return nil
	})
	if err != nil {
		return contractOutput{}, err
	}
	selectedRevision := contractDigest(snapshot.Contents[selected])
	if selected == destination && len(result.Changes) > 0 {
		selectedRevision = changedContractRevision(result.Changes, displayWorkPath(snapshot.Root, selected))
		if selectedRevision == "" {
			return contractOutput{}, fmt.Errorf("changed concrete type file %q lacks final revision evidence", displayWorkPath(snapshot.Root, selected))
		}
	}
	result.ReceiptCandidates = []SymbolTarget{{File: displayWorkPath(snapshot.Root, selected), Symbol: request.Symbol, ExpectedRevision: &selectedRevision}}
	if len(plan.Methods) > 0 {
		destinationRevision := changedContractRevision(result.Changes, displayWorkPath(snapshot.Root, destination))
		if destinationRevision == "" {
			return contractOutput{}, fmt.Errorf("generated method destination lacks planned revision evidence")
		}
		result.ReceiptCandidates = appendGeneratedMethodContinuations(result, plan.Methods, request.Symbol, request.ReceiverKind, displayWorkPath(snapshot.Root, destination), destinationRevision)
	}
	return result, nil
}

func changedContractRevision(changes []contractRevision, path string) string {
	for _, change := range changes {
		if change.Path == path {
			return change.Revision
		}
	}
	return ""
}

func appendGeneratedMethodContinuations(result contractOutput, methods []contractplan.PlannedMethod, symbol string, receiver contractplan.ReceiverKind, destination, revision string) []SymbolTarget {
	for _, method := range methods {
		qualified := symbol + "." + method.Signature.Name
		if receiver == contractplan.ReceiverPointer {
			qualified = "(*" + symbol + ")." + method.Signature.Name
		}
		result.ReceiptCandidates = append(result.ReceiptCandidates, SymbolTarget{File: destination, Symbol: qualified, ExpectedRevision: &revision})
	}
	return result.ReceiptCandidates
}

func implementContractDef() Def[implementContractRequest, contractOutput] {
	return Def[implementContractRequest, contractOutput]{
		Key:     "implement_contract",
		Summary: "Use this tool instead of rebuilding interface method signatures to generate missing Go methods on an existing concrete type. contract_file identifies an ordinary interface, receiver_kind selects pointer or value semantics, and body_policy must explicitly select panic_stub. Generated methods panic when called; they are placeholders, not completed implementations. destination_file must already exist in the concrete type package. Copy type_target to inspect the concrete type and each generated_methods file/symbol pair to replace_body or rename. Existing incompatible methods, unsupported promoted methods, and failed required-package overlay type checks are refused before one apply.",
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
