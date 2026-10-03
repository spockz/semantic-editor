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

	"golang.org/x/tools/go/packages"
	"semedit/internal/sourcebuf"
)

// InlineFunctionOptions identifies one caller-owned direct call and its captured revision.
type InlineFunctionOptions struct {
	CallerFile     string
	InFunction     string
	Discriminator  string
	CallerRevision string
	CalleeRevision string
}

// SemanticEditPlan contains one validated source change and the bytes used to plan it.
type SemanticEditPlan struct {
	Change  sourcebuf.Change
	ReadSet map[string][]byte
	Diff    string
}

// PlanInlineFunctionOverlay resolves and plans one direct zero-argument literal-return call from the current Go overlay.
func PlanInlineFunctionOverlay(ctx context.Context, workspaceRoot, calleeFile, calleeSymbol string, opts InlineFunctionOptions) (SemanticEditPlan, error) {
	if !token.IsIdentifier(calleeSymbol) || opts.InFunction == "" || opts.Discriminator == "" {
		return SemanticEditPlan{}, fmt.Errorf("inline function requires a simple callee name, caller function, and exact call-site discriminator")
	}
	workspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	calleePath, err := filepath.Abs(filepath.Join(workspaceRoot, calleeFile))
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("resolve callee file: %w", err)
	}
	callerPath, err := filepath.Abs(filepath.Join(workspaceRoot, opts.CallerFile))
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("resolve caller file: %w", err)
	}
	moduleRoot, err := moveModuleRoot(calleePath)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if !moveWithin(workspaceRoot, moduleRoot) || !moveWithin(moduleRoot, calleePath) || !moveWithin(moduleRoot, callerPath) {
		return SemanticEditPlan{}, fmt.Errorf("inline function requires existing caller and callee files within the selected workspace module")
	}
	if calleePathErr := rejectMoveSymlinkPath(moduleRoot, calleePath); calleePathErr != nil {
		return SemanticEditPlan{}, calleePathErr
	}
	if callerPathErr := rejectMoveSymlinkPath(moduleRoot, callerPath); callerPathErr != nil {
		return SemanticEditPlan{}, callerPathErr
	}
	readSet, packagesInScope, pkg, err := captureInlinePackage(ctx, moduleRoot, calleePath, callerPath, calleeSymbol, opts)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	callerSource, ok := readSet[callerPath]
	if !ok {
		return SemanticEditPlan{}, fmt.Errorf("caller file is not an existing Go source file in the selected module")
	}
	calleeAST, callerAST := syntaxForMoveFile(pkg, calleePath), syntaxForMoveFile(pkg, callerPath)
	if calleeAST == nil || callerAST == nil {
		return SemanticEditPlan{}, fmt.Errorf("inline function requires active caller and callee declarations")
	}
	callee, calleeObject := findInlineCallee(pkg, calleeAST, calleePath, calleeSymbol)
	if callee == nil || calleeObject == nil {
		return SemanticEditPlan{}, fmt.Errorf("inline callee %q is not a free function declared in %s", calleeSymbol, calleeFile)
	}
	owner, err := findInlineOwner(callerAST, opts.InFunction)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	call, start, end, err := findInlineCall(pkg, callerSource, owner, calleeObject, opts.Discriminator)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if len(call.Args) != 0 {
		return SemanticEditPlan{}, fmt.Errorf("inline function currently supports only zero-argument calls")
	}
	signature, ok := calleeObject.Type().(*types.Signature)
	if !ok || signature.Recv() != nil || signature.TypeParams() != nil && signature.TypeParams().Len() != 0 || signature.Variadic() || signature.Params().Len() != 0 || signature.Results().Len() != 1 {
		return SemanticEditPlan{}, fmt.Errorf("inline callee must be a non-generic free function with no arguments and exactly one result")
	}
	if len(callee.Body.List) != 1 {
		return SemanticEditPlan{}, fmt.Errorf("inline callee body must contain one direct return statement")
	}
	returned, ok := callee.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return SemanticEditPlan{}, fmt.Errorf("inline callee must return one value directly")
	}
	literal, ok := returned.Results[0].(*ast.BasicLit)
	if !ok {
		return SemanticEditPlan{}, fmt.Errorf("inline callee return expression must be a literal")
	}
	resultType := types.Unalias(signature.Results().At(0).Type())
	typeName, ok := resultType.(*types.Basic)
	if !ok {
		return SemanticEditPlan{}, fmt.Errorf("inline callee result must be a predeclared basic type")
	}
	resultName := typeName.Name()
	predeclared, ok := types.Universe.Lookup(resultName).(*types.TypeName)
	if !ok || !types.Identical(resultType, types.Unalias(predeclared.Type())) {
		return SemanticEditPlan{}, fmt.Errorf("inline callee result type is not a predeclared Go type")
	}
	replacement := resultName + "(" + literal.Value + ")"
	after := make([]byte, 0, len(callerSource)+len(replacement)-(end-start))
	after = append(after, callerSource[:start]...)
	after = append(after, replacement...)
	after = append(after, callerSource[end:]...)
	if _, parseErr := parser.ParseFile(token.NewFileSet(), callerPath, after, parser.ParseComments|parser.AllErrors); parseErr != nil {
		return SemanticEditPlan{}, fmt.Errorf("parse inlined caller: %w", parseErr)
	}
	change := sourcebuf.Change{Path: callerPath, Before: bytes.Clone(callerSource), After: after}
	postPackages, err := typecheckMoveOverlay(ctx, moduleRoot, map[string][]byte{callerPath: after})
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("inline function type preflight: %w", err)
	}
	postPkg := packageForMoveFile(postPackages, callerPath)
	if postPkg == nil {
		return SemanticEditPlan{}, fmt.Errorf("inline caller disappeared during type preflight")
	}
	postCallee, postCalleeObject := findInlineCallee(postPkg, syntaxForMoveFile(postPkg, calleePath), calleePath, calleeSymbol)
	postCallerAST := syntaxForMoveFile(postPkg, callerPath)
	postOwner, ownerErr := findInlineOwner(postCallerAST, opts.InFunction)
	if postCallee == nil || postCalleeObject == nil || ownerErr != nil {
		return SemanticEditPlan{}, fmt.Errorf("inline target identity did not survive type preflight")
	}
	postCall, castErr := findInlineCast(postPkg, postOwner, resultName, literal.Value, start, start+len(replacement))
	if castErr != nil || !types.Identical(postPkg.TypesInfo.TypeOf(postCall), types.Unalias(postCalleeObject.Type().(*types.Signature).Results().At(0).Type())) {
		return SemanticEditPlan{}, fmt.Errorf("inlined conversion does not preserve the callee result type")
	}
	readSet, err = completeRefactorReadSet(ctx, moduleRoot, readSet, packagesInScope)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if err := assertRefactorReadSet(ctx, readSet); err != nil {
		return SemanticEditPlan{}, err
	}
	return SemanticEditPlan{Change: change, ReadSet: readSet, Diff: UnifiedDiff(callerPath, string(callerSource), string(after))}, nil
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

func findInlineCallee(pkg *packages.Package, file *ast.File, path, name string) (*ast.FuncDecl, *types.Func) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != name {
			continue
		}
		if filepath.Clean(pkg.Fset.Position(function.Pos()).Filename) != filepath.Clean(path) {
			continue
		}
		object, ok := pkg.TypesInfo.Defs[function.Name].(*types.Func)
		if !ok {
			return nil, nil
		}
		return function, object
	}
	return nil, nil
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

func findInlineCall(pkg *packages.Package, source []byte, owner *ast.FuncDecl, callee *types.Func, discriminator string) (*ast.CallExpr, int, int, error) {
	var selected *ast.CallExpr
	var start, end, exactCount int
	for _, statement := range owner.Body.List {
		returned, ok := statement.(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			continue
		}
		call, ok := returned.Results[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		if pkg.TypesInfo.Uses[identifier] != callee {
			continue
		}
		callStart := pkg.Fset.Position(call.Pos()).Offset
		callEnd := pkg.Fset.Position(call.End()).Offset
		if callStart < 0 || callEnd > len(source) || string(source[callStart:callEnd]) != discriminator {
			continue
		}
		exactCount++
		selected, start, end = call, callStart, callEnd
	}
	if exactCount != 1 {
		return nil, 0, 0, fmt.Errorf("call_site.discriminator must select exactly one direct return call resolved to %s", callee.Name())
	}
	return selected, start, end, nil
}

func findInlineCast(pkg *packages.Package, owner *ast.FuncDecl, typeName, literal string, start, end int) (*ast.CallExpr, error) {
	if owner == nil {
		return nil, fmt.Errorf("inline owner is missing after preflight")
	}
	var found *ast.CallExpr
	for _, statement := range owner.Body.List {
		returned, ok := statement.(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			continue
		}
		call, ok := returned.Results[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || !pkg.TypesInfo.Types[call.Fun].IsType() {
			continue
		}
		if pkg.Fset.Position(call.Pos()).Offset != start || pkg.Fset.Position(call.End()).Offset != end {
			continue
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if !ok || identifier.Name != typeName {
			continue
		}
		argument, ok := call.Args[0].(*ast.BasicLit)
		if !ok || argument.Value != literal {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("inlined result cast is ambiguous")
		}
		found = call
	}
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
	readSet := make(map[string][]byte, len(original))
	for path, content := range original {
		readSet[path] = bytes.Clone(content)
	}
	if overlay := sourcebuf.Overlay(ctx); overlay != nil {
		for path, content := range overlay {
			if moveWithin(root, path) {
				readSet[filepath.Clean(path)] = bytes.Clone(content)
			}
		}
	}
	moduleRoots := map[string]bool{filepath.Clean(root): true}
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
					readSetErr = fmt.Errorf("capture local Go package input %s: %w", clean, err)
					return false
				}
				readSet[clean] = content
			}
		}
		return true
	}, nil)
	if readSetErr != nil {
		return nil, readSetErr
	}
	for _, moduleRoot := range slices.Sorted(maps.Keys(moduleRoots)) {
		for _, name := range []string{"go.mod", "go.sum"} {
			path := filepath.Join(moduleRoot, name)
			content, err := sourcebuf.ReadFile(ctx, path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) && name == "go.sum" {
					continue
				}
				return nil, fmt.Errorf("capture refactor manifest %s: %w", path, err)
			}
			clean := filepath.Clean(path)
			if captured, exists := readSet[clean]; exists && !bytes.Equal(captured, content) {
				return nil, fmt.Errorf("semantic input changed while capturing refactor manifest %s", clean)
			}
			readSet[clean] = bytes.Clone(content)
		}
	}
	return readSet, nil
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
