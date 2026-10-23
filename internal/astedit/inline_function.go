// This file resolves inline-function semantics against the current Go overlay and applies one validated edit.

package astedit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"

	"golang.org/x/tools/go/packages"
)

// InlineFunctionOptions identifies one caller-owned direct call and its captured revision.
type InlineFunctionOptions struct {
	CallerFile        string
	InFunction        string
	Discriminator     string
	OccurrenceIndices []int
	CallerRevision    string
	CalleeRevision    string
}

// SemanticEditPlan contains one validated source change and the bytes used to plan it.
type SemanticEditPlan struct {
	Change  sourcebuf.Change
	ReadSet map[string][]byte
	Diff    string
}

type inlineLiteralResult struct {
	typeName    string
	literalText string
}

type inlineFunctionPaths struct {
	moduleRoot string
	calleePath string
	callerPath string
}

// PlanInlineFunctionOverlay resolves and plans selected direct calls from the current Go overlay.
func PlanInlineFunctionOverlay(ctx context.Context, workspaceRoot, calleeFile, calleeSymbol string, opts InlineFunctionOptions) (SemanticEditPlan, error) {
	target, err := parseInlineCalleeTarget(calleeSymbol)
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("inline callable target: %w", err)
	}
	if opts.InFunction == "" || opts.Discriminator == "" {
		return SemanticEditPlan{}, fmt.Errorf("inline callable requires an unqualified function or receiver-qualified method, caller function, and exact call-site discriminator")
	}
	if len(opts.OccurrenceIndices) > 0 && opts.CallerRevision == "" {
		return SemanticEditPlan{}, fmt.Errorf("call_site.expected_revision is required with occurrence_indices")
	}
	paths, err := resolveInlineFunctionPaths(workspaceRoot, calleeFile, opts.CallerFile)
	if err != nil {
		return SemanticEditPlan{}, err
	}

	readSet, packagesInScope, pkg, err := captureInlinePackage(ctx, paths.moduleRoot, paths.calleePath, paths.callerPath, target.name, opts)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	callerSource, ok := readSet[paths.callerPath]
	if !ok {
		return SemanticEditPlan{}, fmt.Errorf("caller file is not an existing Go source file in the selected module")
	}
	calleeAST, callerAST := syntaxForMoveFile(pkg, paths.calleePath), syntaxForMoveFile(pkg, paths.callerPath)
	if calleeAST == nil || callerAST == nil {
		return SemanticEditPlan{}, fmt.Errorf("inline function requires active caller and callee declarations")
	}
	callee, calleeObject := findInlineCallee(pkg, calleeAST, paths.calleePath, target)
	if callee == nil || calleeObject == nil {
		return SemanticEditPlan{}, fmt.Errorf("inline callee %q is not one uniquely identified callable declared in %s", calleeSymbol, calleeFile)
	}
	owner, err := findInlineOwner(callerAST, opts.InFunction)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	candidates, err := findInlineCallCandidates(pkg, callerSource, owner, opts.Discriminator)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	selected, err := selectInlineFunctionCalls(candidates, opts.OccurrenceIndices, pkg, calleeObject)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	replacements, err := planInlineCallReplacements(pkg, owner, callee, calleeObject, calleeSymbol, selected)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	after, err := applyInlineCallReplacements(callerSource, replacements)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if _, parseErr := parser.ParseFile(token.NewFileSet(), paths.callerPath, after, parser.ParseComments|parser.AllErrors); parseErr != nil {
		return SemanticEditPlan{}, fmt.Errorf("parse inlined caller: %w", parseErr)
	}
	change := sourcebuf.Change{Path: paths.callerPath, Before: bytes.Clone(callerSource), After: after}
	if validationErr := validateInlineReplacements(ctx, paths.moduleRoot, paths.callerPath, paths.calleePath, calleeSymbol, opts.InFunction, replacements, after); validationErr != nil {
		return SemanticEditPlan{}, validationErr
	}
	readSet, err = completeRefactorReadSet(ctx, paths.moduleRoot, readSet, packagesInScope)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if err := assertRefactorReadSet(ctx, readSet); err != nil {
		return SemanticEditPlan{}, err
	}
	return SemanticEditPlan{Change: change, ReadSet: readSet, Diff: UnifiedDiff(paths.callerPath, string(callerSource), string(after))}, nil
}

type inlineCallContext uint8

const (
	inlineCallUnsupported inlineCallContext = iota
	inlineCallReturn
	inlineCallAssignment
)

type inlineCallCandidate struct {
	call    *ast.CallExpr
	start   int
	end     int
	context inlineCallContext
}

type inlineCallReplacement struct {
	start   int
	end     int
	text    string
	literal inlineLiteralResult
}

func resolveInlineFunctionPaths(workspaceRoot, calleeFile, callerFile string) (inlineFunctionPaths, error) {
	workspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return inlineFunctionPaths{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	calleePath, err := filepath.Abs(filepath.Join(workspaceRoot, calleeFile))
	if err != nil {
		return inlineFunctionPaths{}, fmt.Errorf("resolve callee file: %w", err)
	}
	callerPath, err := filepath.Abs(filepath.Join(workspaceRoot, callerFile))
	if err != nil {
		return inlineFunctionPaths{}, fmt.Errorf("resolve caller file: %w", err)
	}
	moduleRoot, err := moveModuleRoot(calleePath)
	if err != nil {
		return inlineFunctionPaths{}, err
	}
	if !moveWithin(workspaceRoot, moduleRoot) || !moveWithin(moduleRoot, calleePath) || !moveWithin(moduleRoot, callerPath) {
		return inlineFunctionPaths{}, fmt.Errorf("inline function requires existing caller and callee files within the selected workspace module")
	}
	if err := rejectMoveSymlinkPath(moduleRoot, calleePath); err != nil {
		return inlineFunctionPaths{}, err
	}
	if err := rejectMoveSymlinkPath(moduleRoot, callerPath); err != nil {
		return inlineFunctionPaths{}, err
	}
	return inlineFunctionPaths{moduleRoot: moduleRoot, calleePath: calleePath, callerPath: callerPath}, nil
}

func validateInlineLiteralCallee(pkg *packages.Package, owner *ast.FuncDecl, callee *ast.FuncDecl, calleeObject *types.Func, call *ast.CallExpr) (inlineLiteralResult, error) {
	if callee.Body == nil {
		return inlineLiteralResult{}, fmt.Errorf("inline callee declaration has no body")
	}
	if len(call.Args) != 0 {
		return inlineLiteralResult{}, fmt.Errorf("inline function currently supports only zero-argument calls")
	}
	signature, ok := calleeObject.Type().(*types.Signature)
	if !ok || signature.TypeParams() != nil && signature.TypeParams().Len() != 0 || signature.Variadic() || signature.Params().Len() != 0 || signature.Results().Len() != 1 {
		return inlineLiteralResult{}, fmt.Errorf("inline callee must be a non-generic function or method with no arguments and exactly one result")
	}
	if signature.Recv() == nil {
		if _, isIdentifier := call.Fun.(*ast.Ident); !isIdentifier {
			return inlineLiteralResult{}, fmt.Errorf("free-function inline calls must use a direct identifier")
		}
	} else {
		if signature.RecvTypeParams() != nil && signature.RecvTypeParams().Len() != 0 {
			return inlineLiteralResult{}, fmt.Errorf("inline methods on generic receiver types are unsupported")
		}
		if len(callee.Type.Results.List) != 1 || len(callee.Type.Results.List[0].Names) != 0 {
			return inlineLiteralResult{}, fmt.Errorf("inline methods require one unnamed result")
		}
		if !inlineCallTargets(pkg, owner, call, calleeObject) {
			return inlineLiteralResult{}, fmt.Errorf("inline method call requires a direct caller-parameter receiver, no receiver indirection, and a single-statement owner")
		}
	}
	if len(callee.Body.List) != 1 {
		return inlineLiteralResult{}, fmt.Errorf("inline callee body must contain one direct return statement")
	}
	returned, ok := callee.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return inlineLiteralResult{}, fmt.Errorf("inline callee must return one value directly")
	}
	literal, ok := returned.Results[0].(*ast.BasicLit)
	if !ok {
		return inlineLiteralResult{}, fmt.Errorf("inline callee return expression must be a literal")
	}
	resultType := types.Unalias(signature.Results().At(0).Type())
	typeName, ok := resultType.(*types.Basic)
	if !ok {
		return inlineLiteralResult{}, fmt.Errorf("inline callee result must be a predeclared basic type")
	}
	resultName := typeName.Name()
	predeclared, ok := types.Universe.Lookup(resultName).(*types.TypeName)
	if !ok || !types.Identical(resultType, types.Unalias(predeclared.Type())) {
		return inlineLiteralResult{}, fmt.Errorf("inline callee result type is not a predeclared Go type")
	}
	return inlineLiteralResult{typeName: resultName, literalText: literal.Value}, nil
}

func validateInlineReplacements(ctx context.Context, moduleRoot, callerPath, calleePath, calleeSymbol, ownerName string, replacements []inlineCallReplacement, after []byte) error {
	postPackages, err := typecheckMoveOverlay(ctx, moduleRoot, map[string][]byte{callerPath: after})
	if err != nil {
		return fmt.Errorf("inline function type preflight: %w", err)
	}
	postPkg := packageForMoveFile(postPackages, callerPath)
	if postPkg == nil {
		return fmt.Errorf("inline caller disappeared during type preflight")
	}
	target, err := parseInlineCalleeTarget(calleeSymbol)
	if err != nil {
		return err
	}
	postCallee, postCalleeObject := findInlineCallee(postPkg, syntaxForMoveFile(postPkg, calleePath), calleePath, target)
	postOwner, ownerErr := findInlineOwner(syntaxForMoveFile(postPkg, callerPath), ownerName)
	if postCallee == nil || postCalleeObject == nil || ownerErr != nil {
		return fmt.Errorf("inline target identity did not survive type preflight")
	}
	postSignature, postOK := postCalleeObject.Type().(*types.Signature)
	if !postOK || postSignature == nil || postSignature.Results() == nil || postSignature.Results().Len() != 1 || postPkg.TypesInfo == nil {
		return fmt.Errorf("inlined conversion does not preserve the callee result type")
	}
	resultType := types.Unalias(postSignature.Results().At(0).Type())
	slices.SortFunc(replacements, func(left, right inlineCallReplacement) int { return left.start - right.start })
	shift := 0
	for _, replacement := range replacements {
		start := replacement.start + shift
		end := start + len(replacement.text)
		postCall, findErr := findInlineCastAt(postPkg, postOwner, replacement.literal.typeName, replacement.literal.literalText, start, end)
		if findErr != nil {
			return fmt.Errorf("inlined conversion does not preserve the callee result type: %w", findErr)
		}
		postCallType := postPkg.TypesInfo.TypeOf(postCall)
		if postCallType == nil || !types.Identical(postCallType, resultType) {
			return fmt.Errorf("inlined conversion does not preserve the callee result type")
		}
		shift += len(replacement.text) - (replacement.end - replacement.start)
	}
	return nil
}

func captureInlinePackage(ctx context.Context, moduleRoot, calleePath, callerPath, calleeSymbol string, opts InlineFunctionOptions) (map[string][]byte, []*packages.Package, *packages.Package, error) {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if authorizationErr := buffer.AuthorizeWrites(ctx, moduleRoot); authorizationErr != nil {
			return nil, nil, nil, authorizationErr
		}
	}
	readSet, err := inspectMoveModuleFiles(ctx, moduleRoot, calleePath, callerPath, calleeSymbol)
	if err != nil {
		return nil, nil, nil, err
	}
	calleeSource, ok := readSet[calleePath]
	if !ok {
		return nil, nil, nil, fmt.Errorf("callee file is not an existing Go source file in the selected module")
	}
	callerSource, ok := readSet[callerPath]
	if !ok {
		return nil, nil, nil, fmt.Errorf("caller file is not an existing Go source file in the selected module")
	}
	if opts.CalleeRevision != "" && revisionOf(calleeSource) != opts.CalleeRevision {
		return nil, nil, nil, fmt.Errorf("expected_revision does not match captured callee source")
	}
	if opts.CallerRevision != "" && revisionOf(callerSource) != opts.CallerRevision {
		return nil, nil, nil, fmt.Errorf("call_site.expected_revision does not match captured caller source")
	}
	pkgs, err := loadMovePackages(ctx, moduleRoot, readSet)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load inline-function package types: %w", err)
	}
	if packageErr := validateRefactorPackages(pkgs, "inline function"); packageErr != nil {
		return nil, nil, nil, packageErr
	}
	readSet, err = completeRefactorReadSet(ctx, moduleRoot, readSet, pkgs)
	if err != nil {
		return nil, nil, nil, err
	}
	pkgs, err = loadMovePackages(ctx, moduleRoot, readSet)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reload inline-function package types from captured inputs: %w", err)
	}
	if err := validateRefactorPackages(pkgs, "inline function"); err != nil {
		return nil, nil, nil, err
	}
	pkg := packageForMoveFile(pkgs, calleePath)
	if pkg == nil || packageForMoveFile(pkgs, callerPath) != pkg {
		return nil, nil, nil, fmt.Errorf("inline function requires caller and callee in the same active Go package")
	}
	return readSet, pkgs, pkg, nil
}

type inlineCalleeTarget struct {
	receiver string
	name     string
}

func parseInlineCalleeTarget(symbol string) (inlineCalleeTarget, error) {
	if token.IsIdentifier(symbol) {
		return inlineCalleeTarget{name: symbol}, nil
	}
	parts := strings.Split(symbol, ".")
	if len(parts) != 2 || !token.IsIdentifier(parts[0]) || !token.IsIdentifier(parts[1]) {
		return inlineCalleeTarget{}, fmt.Errorf("callee symbol must be a function name or receiver-qualified method such as Type.Method")
	}
	return inlineCalleeTarget{receiver: parts[0], name: parts[1]}, nil
}

func findInlineCallee(pkg *packages.Package, file *ast.File, path string, target inlineCalleeTarget) (*ast.FuncDecl, *types.Func) {
	var foundDecl *ast.FuncDecl
	var foundObject *types.Func
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != target.name || (function.Recv == nil) != (target.receiver == "") {
			continue
		}
		if filepath.Clean(pkg.Fset.Position(function.Pos()).Filename) != filepath.Clean(path) {
			continue
		}
		object, ok := pkg.TypesInfo.Defs[function.Name].(*types.Func)
		if !ok {
			return nil, nil
		}
		if target.receiver != "" {
			if !inlineReceiverMatches(object, target.receiver) {
				continue
			}
		}
		if foundDecl != nil {
			return nil, nil
		}
		foundDecl, foundObject = function, object
	}
	return foundDecl, foundObject
}

func inlineReceiverMatches(object *types.Func, receiverName string) bool {
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiverType := types.Unalias(signature.Recv().Type())
	if pointer, isPointer := receiverType.(*types.Pointer); isPointer {
		receiverType = types.Unalias(pointer.Elem())
	}
	named, ok := receiverType.(*types.Named)
	return ok && named.Obj().Name() == receiverName
}

func findInlineOwner(file *ast.File, name string) (*ast.FuncDecl, error) {
	var found *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != name {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("inline owner %q is ambiguous", name)
		}
		found = function
	}
	if found == nil || found.Body == nil {
		return nil, fmt.Errorf("inline owner %q is not a free function with a body", name)
	}
	return found, nil
}

func findInlineCallCandidates(pkg *packages.Package, source []byte, owner *ast.FuncDecl, discriminator string) ([]inlineCallCandidate, error) {
	contexts := map[*ast.CallExpr]inlineCallContext{}
	for _, statement := range owner.Body.List {
		switch typed := statement.(type) {
		case *ast.ReturnStmt:
			if len(typed.Results) == 1 {
				if call, ok := typed.Results[0].(*ast.CallExpr); ok {
					contexts[call] = inlineCallReturn
				}
			}
		case *ast.AssignStmt:
			if (typed.Tok == token.DEFINE || typed.Tok == token.ASSIGN) && len(typed.Lhs) == 1 && len(typed.Rhs) == 1 {
				if call, ok := typed.Rhs[0].(*ast.CallExpr); ok {
					contexts[call] = inlineCallAssignment
				}
			}
		}
	}
	candidates := make([]inlineCallCandidate, 0)
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		start, end := pkg.Fset.Position(call.Pos()).Offset, pkg.Fset.Position(call.End()).Offset
		if start < 0 || end > len(source) || string(source[start:end]) != discriminator {
			return true
		}
		candidates = append(candidates, inlineCallCandidate{call: call, start: start, end: end, context: contexts[call]})
		return true
	})
	slices.SortFunc(candidates, func(left, right inlineCallCandidate) int { return left.start - right.start })
	if len(candidates) == 0 {
		return nil, fmt.Errorf("call_site.discriminator does not match a call expression in the selected owner")
	}
	return candidates, nil
}

func selectInlineCallCandidates(candidates []inlineCallCandidate, indices []int) ([]inlineCallCandidate, error) {
	if len(indices) == 0 {
		return nil, fmt.Errorf("occurrence_indices is required to select multiple exact call occurrences")
	}
	selected := make([]inlineCallCandidate, 0, len(indices))
	previous := -1
	for _, index := range indices {
		if index < 0 || index <= previous {
			return nil, fmt.Errorf("occurrence_indices must contain unique, strictly increasing nonnegative indices")
		}
		if index >= len(candidates) {
			return nil, fmt.Errorf("occurrence_indices contains out-of-range index %d for %d exact call occurrences", index, len(candidates))
		}
		selected = append(selected, candidates[index])
		previous = index
	}
	return selected, nil
}

func selectInlineReturnCandidate(candidates []inlineCallCandidate, pkg *packages.Package, callee *types.Func) ([]inlineCallCandidate, error) {
	var selected []inlineCallCandidate
	for _, candidate := range candidates {
		if candidate.context != inlineCallReturn || !inlineCallResolvesTo(pkg, candidate.call, callee) {
			continue
		}
		selected = append(selected, candidate)
	}
	if len(selected) > 1 {
		return nil, fmt.Errorf("call_site.discriminator selects multiple direct return calls; it must select exactly one direct return call")
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("call_site.discriminator must select exactly one direct return call resolved to %s", callee.Name())
	}
	return selected, nil
}

func applyInlineCallReplacements(source []byte, replacements []inlineCallReplacement) ([]byte, error) {
	ordered := slices.Clone(replacements)
	slices.SortFunc(ordered, func(left, right inlineCallReplacement) int { return left.start - right.start })
	for index, replacement := range ordered {
		if replacement.start < 0 || replacement.end > len(source) || replacement.start >= replacement.end {
			return nil, fmt.Errorf("selected call has an invalid source span")
		}
		if index > 0 && ordered[index-1].end > replacement.start {
			return nil, fmt.Errorf("selected call spans overlap")
		}
	}
	result := bytes.Clone(source)
	for _, replacement := range slices.Backward(ordered) {

		next := make([]byte, 0, len(result)+len(replacement.text)-(replacement.end-replacement.start))
		next = append(next, result[:replacement.start]...)
		next = append(next, replacement.text...)
		next = append(next, result[replacement.end:]...)
		result = next
	}
	return result, nil
}

func hasInlineCommentOverlap(pkg *packages.Package, call *ast.CallExpr) bool {
	start, end := pkg.Fset.Position(call.Pos()).Offset, pkg.Fset.Position(call.End()).Offset
	filename := filepath.Clean(pkg.Fset.PositionFor(call.Pos(), false).Filename)
	for _, group := range pkg.Syntax {
		if filepath.Clean(pkg.Fset.PositionFor(group.Pos(), false).Filename) != filename {
			continue
		}
		for _, comment := range group.Comments {
			commentStart, commentEnd := pkg.Fset.Position(comment.Pos()).Offset, pkg.Fset.Position(comment.End()).Offset
			if commentStart < end && start < commentEnd {
				return true
			}
		}
	}
	return false
}

func inlineCallResolvesTo(pkg *packages.Package, call *ast.CallExpr, callee *types.Func) bool {
	if identifier, ok := call.Fun.(*ast.Ident); ok {
		return pkg.TypesInfo.Uses[identifier] == callee
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := pkg.TypesInfo.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal && selection.Obj() == callee
}

func inlineCallTargets(pkg *packages.Package, owner *ast.FuncDecl, call *ast.CallExpr, callee *types.Func) bool {
	if !inlineCallResolvesTo(pkg, call, callee) {
		return false
	}
	signature, ok := callee.Type().(*types.Signature)
	if !ok {
		return false
	}
	if signature.Recv() == nil {
		_, isIdentifier := call.Fun.(*ast.Ident)
		return isIdentifier
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := pkg.TypesInfo.Selections[selector]
	if selection == nil || len(selection.Index()) != 1 {
		return false
	}
	// Selection.Indirect is unreliable for some MethodVal selections, so compare the actual receiver types.
	if !types.Identical(types.Unalias(selection.Recv()), types.Unalias(signature.Recv().Type())) {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	variable, ok := pkg.TypesInfo.Uses[receiver].(*types.Var)
	if !ok || !inlineOwnerParameter(pkg, owner, variable) || len(owner.Body.List) != 1 {
		return false
	}
	return true
}

func inlineOwnerParameter(pkg *packages.Package, owner *ast.FuncDecl, target *types.Var) bool {
	if owner == nil || owner.Type == nil || owner.Type.Params == nil {
		return false
	}
	for _, field := range owner.Type.Params.List {
		for _, name := range field.Names {
			if pkg.TypesInfo.Defs[name] == target {
				return true
			}
		}
	}
	return false
}

func findInlineCastAt(pkg *packages.Package, owner *ast.FuncDecl, typeName, literal string, start, end int) (*ast.CallExpr, error) {
	if owner == nil {
		return nil, fmt.Errorf("inline owner is missing after preflight")
	}
	var found *ast.CallExpr
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || !pkg.TypesInfo.Types[call.Fun].IsType() {
			return true
		}
		if pkg.Fset.Position(call.Pos()).Offset != start || pkg.Fset.Position(call.End()).Offset != end {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok || identifier.Name != typeName {
			return true
		}
		argument, ok := call.Args[0].(*ast.BasicLit)
		if !ok || argument.Value != literal {
			return true
		}
		if found != nil {
			return false
		}
		found = call
		return true
	})
	if found == nil {
		return nil, fmt.Errorf("inlined result cast is missing")
	}
	return found, nil
}

func validateRefactorPackages(pkgs []*packages.Package, operation string) error {
	if len(pkgs) == 0 {
		return fmt.Errorf("%s package analysis returned no packages", operation)
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("%s cannot establish complete types for %s: %s", operation, pkg.PkgPath, pkg.Errors[0].Msg)
		}
	}
	return nil
}

func completeRefactorReadSet(ctx context.Context, root string, original map[string][]byte, pkgs []*packages.Package) (map[string][]byte, error) {
	readSet := cloneRefactorSources(original)
	applyRefactorOverlay(ctx, root, readSet)
	moduleRoots := map[string]bool{filepath.Clean(root): true}
	if err := collectRefactorPackageInputs(ctx, readSet, moduleRoots, pkgs); err != nil {
		return nil, err
	}
	if err := captureRefactorManifests(ctx, readSet, moduleRoots); err != nil {
		return nil, err
	}
	return readSet, nil
}

func cloneRefactorSources(original map[string][]byte) map[string][]byte {
	readSet := make(map[string][]byte, len(original))
	for path, content := range original {
		readSet[path] = bytes.Clone(content)
	}
	return readSet
}

func applyRefactorOverlay(ctx context.Context, root string, readSet map[string][]byte) {
	for path, content := range sourcebuf.Overlay(ctx, ".go") {
		if moveWithin(root, path) {
			readSet[filepath.Clean(path)] = bytes.Clone(content)
		}
	}
}

func collectRefactorPackageInputs(ctx context.Context, readSet map[string][]byte, moduleRoots map[string]bool, pkgs []*packages.Package) error {
	var readSetErr error
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		module := pkg.Module
		if module == nil {
			return true
		}
		if module.Replace != nil {
			module = module.Replace
		}
		if module.Dir != "" && module.Version == "" {
			moduleRoots[filepath.Clean(module.Dir)] = true
		}
		localModule := module.Main || module.Version == ""
		if localModule {
			for _, path := range slices.Concat(pkg.GoFiles, pkg.CompiledGoFiles, pkg.OtherFiles, pkg.EmbedFiles) {
				clean := filepath.Clean(path)
				if _, exists := readSet[clean]; exists {
					continue
				}
				content, err := sourcebuf.ReadFile(ctx, clean)
				if err != nil {
					if readSetErr == nil {
						readSetErr = fmt.Errorf("capture local Go package input %s: %w", clean, err)
					}
					return false
				}
				readSet[clean] = content
			}
		}
		return true
	}, nil)
	return readSetErr
}

func captureRefactorManifests(ctx context.Context, readSet map[string][]byte, moduleRoots map[string]bool) error {
	for _, moduleRoot := range slices.Sorted(maps.Keys(moduleRoots)) {
		for _, name := range []string{"go.mod", "go.sum"} {
			path := filepath.Join(moduleRoot, name)
			content, err := sourcebuf.ReadFile(ctx, path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) && name == "go.sum" {
					continue
				}
				return fmt.Errorf("capture refactor manifest %s: %w", path, err)
			}
			clean := filepath.Clean(path)
			if captured, exists := readSet[clean]; exists && !bytes.Equal(captured, content) {
				return fmt.Errorf("semantic input changed while capturing refactor manifest %s", clean)
			}
			readSet[clean] = bytes.Clone(content)
		}
	}
	return nil
}

func assertRefactorReadSet(ctx context.Context, readSet map[string][]byte) error {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		return buffer.AssertDisk()
	}
	for path, expected := range readSet {
		// #nosec G304 -- paths are exact source and manifest inputs returned by Go package analysis.
		actual, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("recheck semantic input %s: %w", path, err)
		}
		if !bytes.Equal(actual, expected) {
			return fmt.Errorf("semantic input changed during preflight: %s", path)
		}
	}
	return nil
}

func applySemanticEdit(ctx context.Context, plan SemanticEditPlan) error {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if err := assertRefactorReadSet(ctx, plan.ReadSet); err != nil {
			return fmt.Errorf("recheck semantic inputs before buffered application: %w", err)
		}
		if err := buffer.Apply(ctx, []sourcebuf.Change{plan.Change}); err != nil {
			return fmt.Errorf("apply buffered semantic edit: %w", err)
		}
		return nil
	}
	check := func() error {
		for path, expected := range plan.ReadSet {
			// #nosec G304 -- paths are source and manifest inputs captured inside the selected module.
			actual, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("recheck semantic input %s: %w", path, err)
			}
			if !bytes.Equal(actual, expected) {
				return fmt.Errorf("semantic input changed during preflight: %s", path)
			}
		}
		actual, err := os.ReadFile(plan.Change.Path)
		if err != nil {
			return fmt.Errorf("recheck semantic output: %w", err)
		}
		if !bytes.Equal(actual, plan.Change.Before) {
			return fmt.Errorf("semantic output changed during preflight: %s", plan.Change.Path)
		}
		return nil
	}
	if err := sourcebuf.WriteFileChecked(ctx, plan.Change.Path, plan.Change.After, check); err != nil {
		return fmt.Errorf("publish semantic edit: %w", err)
	}
	return nil
}

// ApplySemanticEdit publishes one complete preflighted output through the active buffer or atomic disk writer.
func ApplySemanticEdit(ctx context.Context, plan SemanticEditPlan) error {
	return applySemanticEdit(ctx, plan)
}

func revisionOf(source []byte) string {
	digest := sha256.Sum256(source)
	return hex.EncodeToString(digest[:])
}

func selectInlineFunctionCalls(candidates []inlineCallCandidate, indices []int, pkg *packages.Package, callee *types.Func) ([]inlineCallCandidate, error) {
	if len(indices) == 0 {
		return selectInlineReturnCandidate(candidates, pkg, callee)
	}
	selected, err := selectInlineCallCandidates(candidates, indices)
	if err != nil {
		return nil, err
	}
	signature, ok := callee.Type().(*types.Signature)
	if !ok || signature.Recv() != nil {
		return nil, fmt.Errorf("occurrence_indices supports free functions only")
	}
	if len(selected) < 2 {
		return nil, fmt.Errorf("occurrence_indices must select at least two call sites")
	}
	return selected, nil
}

func planInlineCallReplacements(pkg *packages.Package, owner, callee *ast.FuncDecl, calleeObject *types.Func, calleeSymbol string, selected []inlineCallCandidate) ([]inlineCallReplacement, error) {
	replacements := make([]inlineCallReplacement, 0, len(selected))
	for _, candidate := range selected {
		if candidate.context == inlineCallUnsupported {
			return nil, fmt.Errorf("selected call is not a supported top-level return or single-target assignment RHS")
		}
		if !inlineCallResolvesTo(pkg, candidate.call, calleeObject) {
			return nil, fmt.Errorf("selected call does not resolve to callee %q", calleeSymbol)
		}
		if calleeObject.Type().(*types.Signature).Recv() != nil && candidate.context != inlineCallReturn {
			return nil, fmt.Errorf("inline method calls support direct return expressions only")
		}
		literalResult, err := validateInlineLiteralCallee(pkg, owner, callee, calleeObject, candidate.call)
		if err != nil {
			return nil, err
		}
		if hasInlineCommentOverlap(pkg, candidate.call) {
			return nil, fmt.Errorf("selected call overlaps a comment or directive")
		}
		replacements = append(replacements, inlineCallReplacement{start: candidate.start, end: candidate.end, text: literalResult.typeName + "(" + literalResult.literalText + ")", literal: literalResult})
	}
	return replacements, nil
}
