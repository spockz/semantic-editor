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
	if concretePackage == nil || destinationPackage == nil {
		return Plan{}, fmt.Errorf("concrete and destination packages are required")
	}
	if concretePackage != destinationPackage {
		return Plan{}, fmt.Errorf("method destination must be the exact concrete package")
	}
	if concrete == nil || concrete.Obj() == nil || concrete.Obj().Pkg() != concretePackage || concretePackage.Scope().Lookup(concrete.Obj().Name()) != concrete.Obj() {
		return Plan{}, fmt.Errorf("concrete type must be an exact named object from the concrete package")
	}
	if concrete.TypeParams() != nil && concrete.TypeParams().Len() != 0 {
		return Plan{}, fmt.Errorf("generic concrete type %q is unsupported", concrete.Obj().Name())
	}
	if contract == nil || contract.Obj() == nil || contract.Obj().Pkg() == nil || contract.Obj().Pkg().Scope().Lookup(contract.Obj().Name()) != contract.Obj() {
		return Plan{}, fmt.Errorf("contract must be an exact named interface object")
	}
	if contract.TypeParams() != nil && contract.TypeParams().Len() != 0 {
		return Plan{}, fmt.Errorf("generic interface %q is unsupported", contract.Obj().Name())
	}
	contractInterface, ok := contract.Underlying().(*types.Interface)
	if !ok {
		return Plan{}, fmt.Errorf("contract %q is not an interface", contract.Obj().Name())
	}
	contractInterface.Complete()
	if !contractInterface.IsMethodSet() {
		return Plan{}, fmt.Errorf("constraint interface %q is unsupported", contract.Obj().Name())
	}
	if bodyPolicy != BodyPolicyPanicStub {
		return Plan{}, fmt.Errorf("body_policy %q is unsupported; select panic_stub", bodyPolicy)
	}
	receiver, err := concreteReceiverType(concrete, receiverKind)
	if err != nil {
		return Plan{}, err
	}
	renderer, err := newTypeRenderer(destinationPackage, imports)
	if err != nil {
		return Plan{}, fmt.Errorf("prepare destination imports: %w", err)
	}
	requiredMethods := make([]*types.Func, 0, contractInterface.NumMethods())
	for method := range contractInterface.Methods() {
		requiredMethods = append(requiredMethods, method)
	}
	sort.Slice(requiredMethods, func(left, right int) bool {
		return requiredMethods[left].Name() < requiredMethods[right].Name()
	})
	planned := make([]PlannedMethod, 0, len(requiredMethods))
	for _, required := range requiredMethods {
		if !required.Exported() && required.Pkg() != concretePackage {
			return Plan{}, fmt.Errorf("unexported contract method %q is inaccessible from the concrete package", required.Name())
		}
		requiredSignature, ok := required.Type().(*types.Signature)
		if !ok {
			return Plan{}, fmt.Errorf("contract method %q has no function signature", required.Name())
		}
		if requiredSignature.TypeParams() != nil && requiredSignature.TypeParams().Len() != 0 {
			return Plan{}, fmt.Errorf("generic contract method %q is unsupported", required.Name())
		}
		existing, index, _ := types.LookupFieldOrMethod(receiver, false, concretePackage, required.Name())
		if existing == nil {
			if index != nil {
				return Plan{}, fmt.Errorf("promoted method %q is ambiguous for receiver kind %q", required.Name(), receiverKind)
			}
			direct := directMethod(concrete, required)
			if direct != nil {
				directSignature, ok := direct.Type().(*types.Signature)
				if !ok || !types.Identical(directSignature, requiredSignature) {
					return Plan{}, fmt.Errorf("existing method %q has an incompatible signature", required.Name())
				}
				return Plan{}, fmt.Errorf("existing method %q is unavailable on the requested %s receiver", required.Name(), receiverKind)
			}
			continue
		}
		existingMethod, ok := existing.(*types.Func)
		if !ok {
			return Plan{}, fmt.Errorf("field %q conflicts with required contract method", required.Name())
		}
		existingSignature, ok := existingMethod.Type().(*types.Signature)
		if !ok || !types.Identical(existingSignature, requiredSignature) {
			return Plan{}, fmt.Errorf("existing method %q has an incompatible signature", required.Name())
		}
	}
	for _, required := range requiredMethods {
		existing, _, _ := types.LookupFieldOrMethod(receiver, false, concretePackage, required.Name())
		if existing != nil || directMethod(concrete, required) != nil {
			continue
		}
		signature, err := renderer.signature(required)
		if err != nil {
			return Plan{}, fmt.Errorf("render contract method %q: %w", required.Name(), err)
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
		planned = append(planned, PlannedMethod{Signature: signature, Declaration: declaration})
	}
	declarations := make([]string, 0, len(planned))
	for _, method := range planned {
		declarations = append(declarations, method.Declaration)
	}
	return Plan{Declaration: strings.Join(declarations, "\n"), Methods: planned, Imports: renderer.importSpecs()}, nil
}

func directMethod(concrete *types.Named, required *types.Func) *types.Func {
	for method := range concrete.Methods() {
		if method.Name() == required.Name() && (method.Exported() || method.Pkg() == required.Pkg()) {
			return method
		}
	}
	return nil
}
