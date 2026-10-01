// Package contractplan plans method signatures and declarations from resolved Go type objects.
// WHY: This file plans Go interface declarations from exact selected method objects without reading or publishing source.
package contractplan

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// PlanExtractInterface plans a closed interface from exact direct methods; an exact existing interface returns an empty no-op plan.
func PlanExtractInterface(sourcePackage, destinationPackage *types.Package, concrete *types.Named, receiverKind ReceiverKind, selected []*types.Func, newName string, imports []ImportBinding) (Plan, error) {
	if sourcePackage == nil || destinationPackage == nil {
		return Plan{}, fmt.Errorf("source and destination packages are required")
	}
	if sourcePackage != destinationPackage {
		return Plan{}, fmt.Errorf("interface destination must be the exact source package")
	}
	if concrete == nil || concrete.Obj() == nil || concrete.Obj().Pkg() != sourcePackage || sourcePackage.Scope().Lookup(concrete.Obj().Name()) != concrete.Obj() {
		return Plan{}, fmt.Errorf("concrete type must be an exact named object from the source package")
	}
	if concrete.TypeParams() != nil && concrete.TypeParams().Len() != 0 {
		return Plan{}, fmt.Errorf("generic concrete type %q is unsupported", concrete.Obj().Name())
	}
	if _, ok := concrete.Underlying().(*types.Interface); ok {
		return Plan{}, fmt.Errorf("concrete type %q is an interface", concrete.Obj().Name())
	}
	if err := validateIdentifier(newName, "interface name"); err != nil {
		return Plan{}, err
	}
	var existingInterface *types.Interface
	if existing := destinationPackage.Scope().Lookup(newName); existing != nil {
		typeName, ok := existing.(*types.TypeName)
		if !ok {
			return Plan{}, fmt.Errorf("interface name %q already exists in the destination package", newName)
		}
		named, ok := typeName.Type().(*types.Named)
		if !ok {
			return Plan{}, fmt.Errorf("interface name %q already exists in the destination package", newName)
		}
		existingInterface, ok = named.Underlying().(*types.Interface)
		if !ok {
			return Plan{}, fmt.Errorf("interface name %q already exists as a non-interface type", newName)
		}
		existingInterface.Complete()
	}
	if len(selected) == 0 {
		return Plan{}, fmt.Errorf("at least one direct method must be selected")
	}
	receiver, err := concreteReceiverType(concrete, receiverKind)
	if err != nil {
		return Plan{}, err
	}
	renderer, err := newTypeRenderer(destinationPackage, imports)
	if err != nil {
		return Plan{}, fmt.Errorf("prepare destination imports: %w", err)
	}
	renderer.names[newName] = destinationPackage.Path()
	methodSet := types.NewMethodSet(receiver)
	seen := make(map[string]struct{}, len(selected))
	planned := make([]PlannedMethod, 0, len(selected))
	for _, method := range selected {
		if method == nil {
			return Plan{}, fmt.Errorf("selected method object is nil")
		}
		if _, exists := seen[method.Name()]; exists {
			return Plan{}, fmt.Errorf("method %q was selected more than once", method.Name())
		}
		seen[method.Name()] = struct{}{}
		if !method.Exported() && method.Pkg() != destinationPackage {
			return Plan{}, fmt.Errorf("unexported method %q is inaccessible from the destination package", method.Name())
		}
		if method.Origin() != method {
			return Plan{}, fmt.Errorf("generic or instantiated method %q is unsupported", method.Name())
		}
		selection := methodSet.Lookup(method.Pkg(), method.Name())
		if selection == nil {
			return Plan{}, fmt.Errorf("method %q is not in the requested %s receiver method set", method.Name(), receiverKind)
		}
		if selection.Obj() != method {
			return Plan{}, fmt.Errorf("selected method %q does not resolve to the exact concrete method object", method.Name())
		}
		if len(selection.Index()) != 1 {
			return Plan{}, fmt.Errorf("promoted method %q is unsupported", method.Name())
		}
		methodSignature, ok := method.Type().(*types.Signature)
		if !ok || methodSignature.Recv() == nil {
			return Plan{}, fmt.Errorf("selected object %q is not a receiver method", method.Name())
		}
		receiverBase := types.Unalias(methodSignature.Recv().Type())
		if pointer, ok := receiverBase.(*types.Pointer); ok {
			receiverBase = types.Unalias(pointer.Elem())
		}
		if !types.Identical(receiverBase, concrete) {
			return Plan{}, fmt.Errorf("method %q does not have the exact concrete receiver type", method.Name())
		}
		signature, err := renderer.signature(method)
		if err != nil {
			return Plan{}, fmt.Errorf("render method %q: %w", method.Name(), err)
		}
		planned = append(planned, PlannedMethod{Signature: signature, Declaration: signature.declaration()})
	}
	if existingInterface != nil {
		if !existingInterface.IsMethodSet() {
			return Plan{}, fmt.Errorf("existing interface %q is a constraint, not a runtime method set", newName)
		}
		if !types.Implements(receiver, existingInterface) {
			return Plan{}, fmt.Errorf("requested %s receiver does not implement existing interface %q", receiverKind, newName)
		}
		if !exactInterfaceMatch(existingInterface, selected) {
			return Plan{}, fmt.Errorf("existing interface %q does not match the selected method set", newName)
		}
		return Plan{}, nil
	}
	sort.Slice(planned, func(left, right int) bool { return planned[left].Signature.Name < planned[right].Signature.Name })
	lines := make([]string, 0, len(planned))
	for _, method := range planned {
		lines = append(lines, "\t"+method.Declaration)
	}
	declaration := "type " + newName + " interface {\n" + strings.Join(lines, "\n") + "\n}"
	return Plan{Declaration: declaration, Methods: planned, Imports: renderer.importSpecs()}, nil
}

func exactInterfaceMatch(existing *types.Interface, selected []*types.Func) bool {
	if existing == nil || len(selected) != existing.NumMethods() {
		return false
	}
	existing.Complete()
	methods := make(map[string]*types.Func, existing.NumMethods())
	for method := range existing.Methods() {
		methods[types.Id(method.Pkg(), method.Name())] = method
	}
	for _, selectedMethod := range selected {
		if selectedMethod == nil {
			return false
		}
		existingMethod := methods[types.Id(selectedMethod.Pkg(), selectedMethod.Name())]
		if existingMethod == nil || !types.Identical(existingMethod.Type(), selectedMethod.Type()) {
			return false
		}
	}
	return true
}
