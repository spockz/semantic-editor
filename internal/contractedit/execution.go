// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file turns typed contract intents into preflighted changes and continuation evidence.
package contractedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/spockz/semantic-editor/internal/contractplan"
)

// ExtractInterface resolves exact direct methods and applies a type-checked interface candidate.
func ExtractInterface(ctx context.Context, request ExtractRequest) (Result, error) {
	paths, err := selectedContractPaths(request.WorkspaceRoot, request.File, request.DestinationFile)
	if err != nil {
		return Result{}, err
	}
	selectedPaths := []string{paths.selected, paths.destination}
	for _, member := range request.Members {
		selectedPaths = append(selectedPaths, member.File)
	}
	snapshot, err := LoadSnapshot(ctx, request.WorkspaceRoot, selectedPaths)
	if err != nil {
		return Result{}, err
	}
	pkg, concrete, err := findType(snapshot, paths.selected, request.InType)
	if err != nil {
		return Result{}, err
	}
	if snapshot.PackageForFile(paths.destination) != pkg {
		return Result{}, fmt.Errorf("interface destination must be an existing file in the concrete type package")
	}
	methods, err := resolveMethods(snapshot, pkg, concrete, request.Members)
	if err != nil {
		return Result{}, err
	}
	imports, err := ImportBindings(pkg, snapshot.Syntax[paths.destination])
	if err != nil {
		return Result{}, err
	}
	plan, err := contractplan.PlanExtractInterface(pkg.Types, pkg.Types, concrete, request.ReceiverKind, methods, request.Symbol, imports)
	if err != nil {
		return Result{}, err
	}
	result, err := finishPlan(ctx, snapshot, paths.destination, paths.selected, request.Symbol, request.ExpectedRevision, plan, func(finalPackage *packages.Package) error {
		generated, typeErr := typeInFile(finalPackage, paths.destination, request.Symbol)
		if typeErr != nil {
			return typeErr
		}
		interfaceType, ok := generated.Underlying().(*types.Interface)
		if !ok || !interfaceType.IsMethodSet() {
			return fmt.Errorf("generated type %q is not a runtime interface", request.Symbol)
		}
		finalConcrete, concreteErr := typeInFile(finalPackage, paths.selected, request.InType)
		if concreteErr != nil {
			return concreteErr
		}
		var receiver types.Type = finalConcrete
		if request.ReceiverKind == contractplan.ReceiverPointer {
			receiver = types.NewPointer(receiver)
		}
		if !types.Implements(receiver, interfaceType) {
			return fmt.Errorf("generated interface %q is not implemented by the requested receiver", request.Symbol)
		}
		return nil
	}, nil)
	return result, err
}

// ImplementContract generates missing methods and publishes only after overlay and test validation.
func ImplementContract(ctx context.Context, request ImplementRequest) (Result, error) {
	paths, err := selectedContractPaths(request.WorkspaceRoot, request.File, request.DestinationFile)
	if err != nil {
		return Result{}, err
	}
	contractInput := request.ContractFile
	if contractInput == "" {
		contractInput = request.File
	}
	contractPath, err := CanonicalPath(request.WorkspaceRoot, contractInput)
	if err != nil {
		return Result{}, err
	}
	snapshot, err := LoadSnapshot(ctx, request.WorkspaceRoot, []string{paths.selected, contractPath, paths.destination})
	if err != nil {
		return Result{}, err
	}
	concretePackage, concrete, err := findType(snapshot, paths.selected, request.Symbol)
	if err != nil {
		return Result{}, err
	}
	contractPackage, contract, err := findType(snapshot, contractPath, request.Contract)
	if err != nil {
		return Result{}, err
	}
	if snapshot.PackageForFile(paths.destination) != concretePackage {
		return Result{}, fmt.Errorf("method destination must be an existing file in the concrete type package")
	}
	imports, err := ImportBindings(concretePackage, snapshot.Syntax[paths.destination])
	if err != nil {
		return Result{}, err
	}
	plan, err := contractplan.PlanImplementContract(concretePackage.Types, concretePackage.Types, concrete, contract, request.ReceiverKind, request.BodyPolicy, imports)
	if err != nil {
		return Result{}, err
	}
	if plan.Declaration != "" {
		if panicErr := requirePanicBuiltin(concretePackage, imports); panicErr != nil {
			return Result{}, panicErr
		}
	}
	result, err := finishPlan(ctx, snapshot, paths.destination, paths.selected, request.Symbol, request.ExpectedRevision, plan, func(finalPackage *packages.Package) error {
		finalConcrete, concreteErr := typeInFile(finalPackage, paths.selected, request.Symbol)
		if concreteErr != nil {
			return fmt.Errorf("final concrete type identity: %w", concreteErr)
		}
		finalContractPackage := snapshot.ByPath[contractPackage.PkgPath]
		if finalContractPackage == nil || finalContractPackage.Types == nil {
			return fmt.Errorf("contract package %q disappeared from the final package load", contractPackage.PkgPath)
		}
		finalContract, contractErr := typeInFile(finalContractPackage, contractPath, request.Contract)
		if contractErr != nil {
			return contractErr
		}
		interfaceType, ok := finalContract.Underlying().(*types.Interface)
		if !ok || !interfaceType.IsMethodSet() {
			return fmt.Errorf("contract %q is not a runtime interface", request.Contract)
		}
		var receiver types.Type = finalConcrete
		if request.ReceiverKind == contractplan.ReceiverPointer {
			receiver = types.NewPointer(receiver)
		}
		if !types.Implements(receiver, interfaceType) {
			return fmt.Errorf("generated methods do not make %s implement %q", request.ReceiverKind, request.Contract)
		}
		return nil
	}, func(result *Result) error {
		return addImplementCandidates(result, snapshot, request, paths, plan)
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

type contractPaths struct {
	selected    string
	destination string
}

func selectedContractPaths(root, selectedInput, destinationInput string) (contractPaths, error) {
	selected, err := CanonicalPath(root, selectedInput)
	if err != nil {
		return contractPaths{}, err
	}
	if destinationInput == "" {
		destinationInput = selectedInput
	}
	destination, err := CanonicalPath(root, destinationInput)
	if err != nil {
		return contractPaths{}, err
	}
	return contractPaths{selected: selected, destination: destination}, nil
}

func findType(snapshot *Snapshot, selected, name string) (*packages.Package, *types.Named, error) {
	pkg := snapshot.PackageForFile(selected)
	if pkg == nil {
		return nil, nil, fmt.Errorf("selected file %q has no package in the current overlay", selected)
	}
	if pkg.Types == nil || pkg.TypesInfo == nil {
		return nil, nil, fmt.Errorf("selected package %q has incomplete Go type information", pkg.PkgPath)
	}
	if len(pkg.Errors) > 0 {
		return nil, nil, fmt.Errorf("type-check selected package %q: %s", pkg.PkgPath, pkg.Errors[0].Msg)
	}
	named, err := typeInFile(pkg, selected, name)
	if err != nil {
		return nil, nil, err
	}
	return pkg, named, nil
}

func typeInFile(pkg *packages.Package, path, name string) (*types.Named, error) {
	if pkg == nil || pkg.Types == nil || pkg.TypesInfo == nil {
		return nil, fmt.Errorf("selected package has incomplete type information")
	}
	object, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("type %q is not declared in package %q", name, pkg.PkgPath)
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("type %q is not a named type", name)
	}
	file := findFile(pkg, path)
	if file == nil {
		return nil, fmt.Errorf("source file %q is not in the selected package", path)
	}
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, spec := range group.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name.Name == name && pkg.TypesInfo.Defs[typeSpec.Name] == object {
				return named, nil
			}
		}
	}
	return nil, fmt.Errorf("type %q is not declared by selected file %q", name, path)
}

func findFile(pkg *packages.Package, path string) *ast.File {
	if pkg == nil {
		return nil
	}
	for _, file := range pkg.Syntax {
		position := pkg.Fset.Position(file.Pos())
		absolute, err := filepath.Abs(position.Filename)
		if err != nil {
			continue
		}
		canonical, err := filepath.EvalSymlinks(absolute)
		if err == nil && canonical == path {
			return file
		}
	}
	return nil
}

func resolveMethods(snapshot *Snapshot, pkg *packages.Package, owner *types.Named, members []MemberTarget) ([]*types.Func, error) {
	if len(members) == 0 {
		return nil, fmt.Errorf("members must select at least one direct method")
	}
	resolved := make([]*types.Func, 0, len(members))
	seen := make(map[string]bool, len(members))
	for _, member := range members {
		key := member.File + "\x00" + member.Symbol
		if seen[key] {
			return nil, fmt.Errorf("member %q in %q was selected more than once", member.Symbol, member.File)
		}
		seen[key] = true
		method, err := methodInFile(snapshot, pkg, owner, member)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, method)
	}
	return resolved, nil
}

func methodInFile(snapshot *Snapshot, pkg *packages.Package, owner *types.Named, member MemberTarget) (*types.Func, error) {
	path, err := CanonicalPath(snapshot.Root, member.File)
	if err != nil {
		return nil, err
	}
	if snapshot.PackageForFile(path) != pkg {
		return nil, fmt.Errorf("member file %q does not resolve to the exact owner package", member.File)
	}
	file := findFile(pkg, path)
	if file == nil || pkg.TypesInfo == nil {
		return nil, fmt.Errorf("member file %q has no resolved syntax", member.File)
	}
	separator := strings.LastIndex(member.Symbol, ".")
	if separator <= 0 || separator == len(member.Symbol)-1 {
		return nil, fmt.Errorf("member symbol %q must be a receiver-qualified method", member.Symbol)
	}
	wantReceiver := member.Symbol[:separator]
	wantName := member.Symbol[separator+1:]
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil || method.Name.Name != wantName || len(method.Recv.List) != 1 {
			continue
		}
		var rendered strings.Builder
		if err := format.Node(&rendered, pkg.Fset, method.Recv.List[0].Type); err != nil {
			return nil, fmt.Errorf("render method receiver %q: %w", member.Symbol, err)
		}
		receiver := rendered.String()
		if strings.HasPrefix(receiver, "*") {
			receiver = "(" + receiver + ")"
		}
		if receiver+"."+wantName != wantReceiver+"."+wantName {
			continue
		}
		object, ok := pkg.TypesInfo.Defs[method.Name].(*types.Func)
		if !ok || object.Pkg() != pkg.Types || object.Origin() != object {
			return nil, fmt.Errorf("method %q has no exact direct object identity", member.Symbol)
		}
		signature, ok := object.Type().(*types.Signature)
		if !ok || signature.Recv() == nil {
			return nil, fmt.Errorf("member %q is not a receiver method", member.Symbol)
		}
		base := types.Unalias(signature.Recv().Type())
		if pointer, ok := base.(*types.Pointer); ok {
			base = types.Unalias(pointer.Elem())
		}
		if !types.Identical(base, owner) {
			return nil, fmt.Errorf("member %q does not belong to the exact owner type", member.Symbol)
		}
		return object, nil
	}
	return nil, fmt.Errorf("direct method %q was not found in %q", member.Symbol, member.File)
}

func requirePanicBuiltin(pkg *packages.Package, imports []contractplan.ImportBinding) error {
	if pkg == nil || pkg.Types == nil {
		return fmt.Errorf("concrete package type information is required to resolve builtin panic")
	}
	if _, ok := types.Universe.Lookup("panic").(*types.Builtin); !ok {
		return fmt.Errorf("go universe does not define the panic builtin")
	}
	if shadow := pkg.Types.Scope().Lookup("panic"); shadow != nil {
		return fmt.Errorf("panic_stub would resolve panic to package declaration %q", shadow.Name())
	}
	for _, imported := range imports {
		if imported.Name == "panic" {
			return fmt.Errorf("panic_stub would resolve panic to file import %q", imported.Path)
		}
	}
	return nil
}
