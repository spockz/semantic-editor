// WHY: Share compiler evidence for signature projections while preserving the distinct permutation and unused-scalar removal proofs.

package operation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"golang.org/x/tools/go/packages"
)

type signatureProjectionPolicy struct {
	RequireConstants bool
	RemovedIndex     *int
}

func signatureProjectionSelection(snapshot *structuralSnapshot, symbol string, mappings []astedit.SignatureParameterMapping) (signatureProjectionPolicy, bool, error) {
	policy := signatureProjectionPolicy{RequireConstants: true}
	file, err := parser.ParseFile(token.NewFileSet(), snapshot.File, snapshot.Sources[snapshot.File], 0)
	if err != nil {
		return policy, false, fmt.Errorf("parse signature mapping target: %w", err)
	}
	count := -1
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == symbol {
			count = function.Type.Params.NumFields()
		}
	}
	if count < 0 {
		return policy, false, fmt.Errorf("signature mapping target was not found")
	}
	if len(mappings) < count {
		if len(mappings) != count-1 {
			return policy, false, fmt.Errorf("change signature: remove exactly one unused scalar parameter")
		}
		next, removed := 0, 0
		for _, mapping := range mappings {
			if mapping.SourceIndex == nil || *mapping.SourceIndex < next || *mapping.SourceIndex >= count || mapping.Argument != "" {
				return policy, false, fmt.Errorf("change signature: removal must preserve surviving parameter indices and order without additions")
			}
			if *mapping.SourceIndex == removed {
				removed++
			}
			next = *mapping.SourceIndex + 1
		}
		return signatureProjectionPolicy{RemovedIndex: &removed}, true, nil
	}
	for index, mapping := range mappings {
		if mapping.SourceIndex != nil && *mapping.SourceIndex != index {
			return policy, true, nil
		}
	}
	return policy, false, nil
}

func signatureResolvedType(value types.Type) string {
	return types.TypeString(value, func(pkg *types.Package) string { return pkg.Path() })
}

func collectSignatureProjectionEvidence(snapshot *structuralSnapshot, symbol string, proof *astedit.SignatureCallSetProof, policy signatureProjectionPolicy) ([]string, error) {
	target, err := resolveSignatureCallTarget(snapshot, symbol, "")
	if err != nil {
		return nil, err
	}
	collector := signatureProjectionCollector{snapshot: snapshot, target: target, symbol: symbol, policy: policy, expected: make(map[string]bool, len(proof.Calls)), seen: make(map[string]astedit.SignatureCallSite, len(proof.Calls))}
	for _, call := range proof.Calls {
		collector.expected[signatureCallKey(call)] = true
	}
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.TypesInfo.Defs == nil || pkg.TypesInfo.Uses == nil || pkg.TypesInfo.Types == nil || pkg.Fset == nil || len(pkg.Errors) != 0 {
			return nil, fmt.Errorf("signature projection requires complete typed active package variants")
		}
		for _, file := range pkg.Syntax {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if err := collector.collectParameters(pkg, function); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, pkg := range snapshot.Packages {
		for _, file := range pkg.Syntax {
			if err := collector.collectFile(pkg, file); err != nil {
				return nil, err
			}
		}
	}
	if collector.parameterTypes == nil || len(collector.seen) != len(collector.expected) {
		return nil, fmt.Errorf("signature projection target or caller evidence is incomplete")
	}
	for index, call := range proof.Calls {
		evidence, exists := collector.seen[signatureCallKey(call)]
		if !exists {
			return nil, fmt.Errorf("signature projection caller evidence is missing")
		}
		proof.Calls[index] = evidence
	}
	proof.Removal = collector.removal
	return collector.parameterTypes, nil
}

func signatureCallKey(call astedit.SignatureCallSite) string {
	return fmt.Sprintf("%s:%d:%d", call.File, call.Start, call.End)
}

func signatureCallArgumentEvidence(snapshot *structuralSnapshot, pkg *packages.Package, call *ast.CallExpr, site astedit.SignatureCallSite, policy signatureProjectionPolicy) (astedit.SignatureCallSite, error) {
	source, exists := snapshot.Sources[site.File]
	if !exists || site.Start < 0 || site.End > len(source) || site.Start >= site.End || call.Ellipsis.IsValid() {
		return astedit.SignatureCallSite{}, fmt.Errorf("signature projection caller is outside the source snapshot or uses ellipsis")
	}
	site.Revision = structuralRevision(source)
	for index, argument := range call.Args {
		value, present := pkg.TypesInfo.Types[argument]
		if !present || value.Type == nil || ((policy.RequireConstants || policy.RemovedIndex != nil && index == *policy.RemovedIndex) && value.Value == nil) {
			if policy.RemovedIndex != nil {
				return astedit.SignatureCallSite{}, fmt.Errorf("parameter removal requires a compiler-proven constant for the discarded argument and complete types for survivors")
			}
			return astedit.SignatureCallSite{}, fmt.Errorf("permutation requires a compiler-proven constant for every argument")
		}
		start, end := pkg.Fset.PositionFor(argument.Pos(), false), pkg.Fset.PositionFor(argument.End(), false)
		if start.Filename != site.File || end.Filename != site.File || start.Offset < site.Start || end.Offset > site.End || start.Offset >= end.Offset {
			return astedit.SignatureCallSite{}, fmt.Errorf("signature projection argument has inconsistent physical positions")
		}
		argumentType, typeErr := signatureArgumentType(pkg, value.Type)
		if typeErr != nil {
			return astedit.SignatureCallSite{}, typeErr
		}
		constantValue := ""
		if value.Value != nil {
			constantValue = value.Value.Kind().String() + ":" + value.Value.ExactString()
		}
		site.Arguments = append(site.Arguments, astedit.SignatureArgumentEvidence{Index: index, Start: start.Offset, End: end.Offset, Source: string(source[start.Offset:end.Offset]), Value: constantValue, Type: argumentType})
	}
	return site, nil
}

func validateSignatureProjectionCandidate(snapshot *structuralSnapshot, plan astedit.SignaturePlan, loaded []*packages.Package) error {
	before := plan.Projection
	if before == nil {
		return nil
	}
	sources := make(map[string][]byte, len(snapshot.Sources))
	maps.Copy(sources, snapshot.Sources)
	for _, file := range plan.Files {
		sources[file.File] = file.After
	}
	candidate := *snapshot
	candidate.Sources = sources
	candidate.Packages = loaded
	proof, err := collectSignatureCallSetWithName(&candidate, before.Symbol, "")
	if err != nil {
		return fmt.Errorf("resolve signature projection candidate calls: %w", err)
	}
	parameterTypes, err := collectSignatureProjectionEvidence(&candidate, before.Symbol, &proof, signatureProjectionPolicy{RequireConstants: before.RemovedIndex == nil})
	if err != nil {
		return fmt.Errorf("prove signature projection candidate arguments: %w", err)
	}
	if len(parameterTypes) != len(before.Order) || len(before.ParameterTypes) != before.OriginalCount {
		return fmt.Errorf("signature projection candidate parameter evidence is incomplete")
	}
	for index, source := range before.Order {
		if parameterTypes[index] != before.ParameterTypes[source] {
			return fmt.Errorf("signature projection changed a resolved parameter type")
		}
	}
	original := slices.Clone(before.Calls)
	sortSignaturePermutationCalls(original)
	sortSignaturePermutationCalls(proof.Calls)
	if len(original) != len(proof.Calls) {
		return fmt.Errorf("signature projection candidate changed the complete call set")
	}
	for index, call := range proof.Calls {
		old := original[index]
		if call.File != old.File || len(call.Arguments) != len(before.Order) || len(old.Arguments) != before.OriginalCount {
			return fmt.Errorf("signature projection candidate call correspondence is incomplete")
		}
		for argument, source := range before.Order {
			previous, current := old.Arguments[source], call.Arguments[argument]
			if previous.Value != current.Value || previous.Type != current.Type || (previous.Source != current.Source) {
				return fmt.Errorf("signature projection changed an argument's constant value, effective type, or source")
			}
		}
	}
	return nil
}

type signatureProjectionCollector struct {
	snapshot       *structuralSnapshot
	target         signatureCallTarget
	symbol         string
	expected       map[string]bool
	seen           map[string]astedit.SignatureCallSite
	parameterTypes []string
	policy         signatureProjectionPolicy
	removal        *astedit.SignatureParameterRemovalEvidence
}

func (collector *signatureProjectionCollector) collectParameters(pkg *packages.Package, function *ast.FuncDecl) error {
	if !signatureObjectMatches(pkg, pkg.TypesInfo.Defs[function.Name], collector.target, collector.symbol) {
		return nil
	}
	signature, ok := pkg.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	if !ok || signature.Variadic() || function.Body == nil {
		return fmt.Errorf("signature projection requires a nonvariadic source-owned free function")
	}
	current := make([]string, signature.Params().Len())
	for index := range current {
		current[index] = signatureResolvedType(signature.Params().At(index).Type())
	}
	if collector.parameterTypes != nil && !slices.Equal(collector.parameterTypes, current) {
		return fmt.Errorf("signature projection target types disagree across active variants")
	}
	collector.parameterTypes = current
	if collector.policy.RemovedIndex != nil {
		index := *collector.policy.RemovedIndex
		if index < 0 || index >= signature.Params().Len() {
			return fmt.Errorf("removed parameter index is outside the target signature")
		}
		parameter := signature.Params().At(index)
		basic, scalar := types.Unalias(parameter.Type()).Underlying().(*types.Basic)
		if !scalar || basic.Info()&(types.IsBoolean|types.IsNumeric|types.IsString) == 0 {
			return fmt.Errorf("removed parameter must have a basic scalar type")
		}
		if astDeclarationUsesObject(function.Body, pkg.TypesInfo, parameter) {
			return fmt.Errorf("removed parameter is used by the function body")
		}
		evidence := astedit.SignatureParameterRemovalEvidence{File: collector.snapshot.File, Revision: structuralRevision(collector.snapshot.Sources[collector.snapshot.File]), Index: index, Name: parameter.Name(), Scalar: true, Unused: true}
		if collector.removal != nil && *collector.removal != evidence {
			return fmt.Errorf("unused scalar parameter evidence disagrees across active variants")
		}
		collector.removal = &evidence
	}
	return nil
}

func (collector *signatureProjectionCollector) collectCall(pkg *packages.Package, call *ast.CallExpr) error {
	position, end := pkg.Fset.PositionFor(call.Pos(), false), pkg.Fset.PositionFor(call.End(), false)
	path, err := filepath.Abs(position.Filename)
	if err != nil {
		return fmt.Errorf("resolve signature projection caller: %w", err)
	}
	site := astedit.SignatureCallSite{File: filepath.Clean(path), Start: position.Offset, End: end.Offset}
	key := signatureCallKey(site)
	identifier, direct := call.Fun.(*ast.Ident)
	matches := direct && signatureObjectMatches(pkg, pkg.TypesInfo.Uses[identifier], collector.target, collector.symbol)
	if !matches && !collector.expected[key] {
		return nil
	}
	if !matches || !collector.expected[key] {
		return fmt.Errorf("signature projection call binding disagrees across active variants")
	}
	evidence, err := signatureCallArgumentEvidence(collector.snapshot, pkg, call, site, collector.policy)
	if err != nil {
		return err
	}
	if previous, exists := collector.seen[key]; exists && (previous.Revision != evidence.Revision || !slices.Equal(previous.Arguments, evidence.Arguments)) {
		return fmt.Errorf("signature projection argument evidence disagrees across active variants")
	}
	collector.seen[key] = evidence
	return nil
}

func (collector *signatureProjectionCollector) collectFile(pkg *packages.Package, file *ast.File) error {
	var visitErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if visitErr != nil {
			return false
		}
		if call, ok := node.(*ast.CallExpr); ok {
			visitErr = collector.collectCall(pkg, call)
		}
		return visitErr == nil
	})
	return visitErr
}

func sortSignaturePermutationCalls(calls []astedit.SignatureCallSite) {
	slices.SortFunc(calls, func(left, right astedit.SignatureCallSite) int {
		if result := strings.Compare(left.File, right.File); result != 0 {
			return result
		}
		if left.Start < right.Start {
			return -1
		}
		if left.Start > right.Start {
			return 1
		}
		return 0
	})
}
