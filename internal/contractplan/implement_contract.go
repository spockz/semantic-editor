// WHY: This file plans explicit panic-stub methods from a resolved Go interface without applying source edits.

package contractplan

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// PlanImplementContract plans missing interface methods using an explicit receiver and panic-stub policy.
func PlanImplementContract(concretePackage, destinationPackage *types.Package, concrete, contract *types.Named, receiverKind ReceiverKind, bodyPolicy BodyPolicy, imports []ImportBinding) (Plan, error) {
	contractInterface, err := validateImplementationTypes(concretePackage, destinationPackage, concrete, contract, bodyPolicy)
	if err != nil {
		return Plan{}, err
	}
	receiver, err := concreteReceiverType(concrete, receiverKind)
	if err != nil {
		return Plan{}, err
	}
	renderer, err := newTypeRenderer(destinationPackage, imports)
	if err != nil {
		return Plan{}, fmt.Errorf("prepare destination imports: %w", err)
	}
	requiredMethods := sortedContractMethods(contractInterface)
	if err = validateExistingContractMethods(receiver, concretePackage, concrete, receiverKind, requiredMethods); err != nil {
		return Plan{}, err
	}
	planned, err := planMissingContractMethods(receiver, concrete, receiverKind, requiredMethods, renderer)
	if err != nil {
		return Plan{}, err
	}
	if len(planned) > 0 {
		if err := requirePanicStubBuiltin(destinationPackage, imports); err != nil {
			return Plan{}, fmt.Errorf("prepare panic stub body: %w", err)
		}
	}
	declarations := make([]string, 0, len(planned))
	for _, method := range planned {
		declarations = append(declarations, method.Declaration)
	}
	return Plan{Declaration: strings.Join(declarations, "\n"), Methods: planned, Imports: renderer.importSpecs()}, nil
}

func validateImplementationTypes(concretePackage, destinationPackage *types.Package, concrete, contract *types.Named, bodyPolicy BodyPolicy) (*types.Interface, error) {
	if concretePackage == nil || destinationPackage == nil {
		return nil, fmt.Errorf("concrete and destination packages are required")
	}
	if concretePackage != destinationPackage {
		return nil, fmt.Errorf("method destination must be the exact concrete package")
	}
	if concrete == nil || concrete.Obj() == nil || concrete.Obj().Pkg() != concretePackage || concretePackage.Scope().Lookup(concrete.Obj().Name()) != concrete.Obj() {
		return nil, fmt.Errorf("concrete type must be an exact named object from the concrete package")
	}
	if concrete.TypeParams() != nil && concrete.TypeParams().Len() != 0 {
		return nil, fmt.Errorf("generic concrete type %q is unsupported", concrete.Obj().Name())
	}
	if contract == nil || contract.Obj() == nil || contract.Obj().Pkg() == nil || contract.Obj().Pkg().Scope().Lookup(contract.Obj().Name()) != contract.Obj() {
		return nil, fmt.Errorf("contract must be an exact named interface object")
	}
	if contract.TypeParams() != nil && contract.TypeParams().Len() != 0 {
		return nil, fmt.Errorf("generic interface %q is unsupported", contract.Obj().Name())
	}
	contractInterface, ok := contract.Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("contract %q is not an interface", contract.Obj().Name())
	}
	contractInterface.Complete()
	if !contractInterface.IsMethodSet() {
		return nil, fmt.Errorf("constraint interface %q is unsupported", contract.Obj().Name())
	}
	if bodyPolicy != BodyPolicyPanicStub {
		return nil, fmt.Errorf("body_policy %q is unsupported; select panic_stub", bodyPolicy)
	}
	return contractInterface, nil
}

func sortedContractMethods(contractInterface *types.Interface) []*types.Func {
	requiredMethods := make([]*types.Func, 0, contractInterface.NumMethods())
	for method := range contractInterface.Methods() {
		requiredMethods = append(requiredMethods, method)
	}
	sort.Slice(requiredMethods, func(left, right int) bool {
		return requiredMethods[left].Name() < requiredMethods[right].Name()
	})
	return requiredMethods
}

func validateExistingContractMethods(receiver types.Type, concretePackage *types.Package, concrete *types.Named, receiverKind ReceiverKind, requiredMethods []*types.Func) error {
	for _, required := range requiredMethods {
		if err := validateExistingContractMethod(receiver, concretePackage, concrete, receiverKind, required); err != nil {
			return err
		}
	}
	return nil
}

func validateExistingContractMethod(receiver types.Type, concretePackage *types.Package, concrete *types.Named, receiverKind ReceiverKind, required *types.Func) error {
	if !required.Exported() && required.Pkg() != concretePackage {
		return fmt.Errorf("unexported contract method %q is inaccessible from the concrete package", required.Name())
	}
	requiredSignature, isSignature := required.Type().(*types.Signature)
	if !isSignature {
		return fmt.Errorf("contract method %q has no function signature", required.Name())
	}
	if requiredSignature.TypeParams() != nil && requiredSignature.TypeParams().Len() != 0 {
		return fmt.Errorf("generic contract method %q is unsupported", required.Name())
	}
	existing, index, _ := types.LookupFieldOrMethod(receiver, false, concretePackage, required.Name())
	if existing == nil {
		return validateDirectContractMethod(concrete, required, requiredSignature, index, receiverKind)
	}
	existingMethod, isMethod := existing.(*types.Func)
	if !isMethod {
		return fmt.Errorf("field %q conflicts with required contract method", required.Name())
	}
	existingSignature, isSignature := existingMethod.Type().(*types.Signature)
	if !isSignature || !types.Identical(existingSignature, requiredSignature) {
		return fmt.Errorf("existing method %q has an incompatible signature", required.Name())
	}
	return nil
}

func validateDirectContractMethod(concrete *types.Named, required *types.Func, requiredSignature *types.Signature, index []int, receiverKind ReceiverKind) error {
	if index != nil {
		return fmt.Errorf("promoted method %q is ambiguous for receiver kind %q", required.Name(), receiverKind)
	}
	direct := directMethod(concrete, required)
	if direct == nil {
		return nil
	}
	directSignature, isSignature := direct.Type().(*types.Signature)
	if !isSignature || !types.Identical(directSignature, requiredSignature) {
		return fmt.Errorf("existing method %q has an incompatible signature", required.Name())
	}
	return fmt.Errorf("existing method %q is unavailable on the requested %s receiver", required.Name(), receiverKind)
}

func planMissingContractMethods(receiver types.Type, concrete *types.Named, receiverKind ReceiverKind, requiredMethods []*types.Func, renderer *typeRenderer) ([]PlannedMethod, error) {
	planned := make([]PlannedMethod, 0, len(requiredMethods))
	for _, required := range requiredMethods {
		method, missing, err := planMissingContractMethod(receiver, concrete, receiverKind, required, renderer)
		if err != nil {
			return nil, err
		}
		if missing {
			planned = append(planned, method)
		}
	}
	return planned, nil
}

func planMissingContractMethod(receiver types.Type, concrete *types.Named, receiverKind ReceiverKind, required *types.Func, renderer *typeRenderer) (PlannedMethod, bool, error) {
	existing, _, _ := types.LookupFieldOrMethod(receiver, false, concrete.Obj().Pkg(), required.Name())
	if existing != nil || directMethod(concrete, required) != nil {
		return PlannedMethod{}, false, nil
	}
	signature, err := renderer.signature(required)
	if err != nil {
		return PlannedMethod{}, false, fmt.Errorf("render contract method %q: %w", required.Name(), err)
	}
	stubSignature := signature
	stubSignature.Parameters = append([]ParameterSpec(nil), signature.Parameters...)
	for index := range stubSignature.Parameters {
		stubSignature.Parameters[index].Name = ""
	}
	stubSignature.Results = append([]ResultSpec(nil), signature.Results...)
	for index := range stubSignature.Results {
		stubSignature.Results[index].Name = ""
	}
	receiverTypeName := concrete.Obj().Name()
	if receiverKind == ReceiverPointer {
		receiverTypeName = "*" + receiverTypeName
	}
	declaration := "func (" + receiverTypeName + ") " + stubSignature.declaration() + " { panic(`TODO: implement`) }"
	return PlannedMethod{Signature: signature, Declaration: declaration}, true, nil
}

func directMethod(concrete *types.Named, required *types.Func) *types.Func {
	for method := range concrete.Methods() {
		if method.Name() == required.Name() && (method.Exported() || method.Pkg() == required.Pkg()) {
			return method
		}
	}
	return nil
}

func requirePanicStubBuiltin(destination *types.Package, imports []ImportBinding) error {
	if destination == nil {
		return fmt.Errorf("destination package is required")
	}
	if _, ok := types.Universe.Lookup("panic").(*types.Builtin); !ok {
		return fmt.Errorf("go universe does not define the panic builtin")
	}
	if shadow := destination.Scope().Lookup("panic"); shadow != nil {
		return fmt.Errorf("panic_stub would resolve panic to package declaration %q", shadow.Name())
	}
	for _, imported := range imports {
		if imported.Name == "panic" {
			return fmt.Errorf("panic_stub would resolve panic to file import %q", imported.Path)
		}
	}
	return nil
}
