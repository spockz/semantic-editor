// Package astedit extracts a selected Go call argument into a local variable.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	types "go/types"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

// ExtractVariableOptions selects one exact first-argument expression in the owner's direct body.
type ExtractVariableOptions struct {
	Discriminator    string
	ConstructPath    string
	ExpectedRevision string
}

type variableCandidate struct {
	path string
	stmt *ast.ExprStmt
	expr ast.Expr
}

// ExtractVariable moves the first side-effect-free, typed argument of an unshadowed same-file package call into a local.
func ExtractVariable(ctx context.Context, filePath, owner, name string, opts ExtractVariableOptions) (string, error) {
	if !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() || name == "_" {
		return "", fmt.Errorf("symbol %q is not a valid local identifier", name)
	}
	if opts.Discriminator == "" {
		return "", fmt.Errorf("discriminator must contain the exact source text of the expression")
	}
	cleanPath := filepath.Clean(filePath)
	ctx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, cleanPath)
	if err != nil {
		return "", err
	}
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read extraction target: %w", err)
	}
	if opts.ConstructPath != "" && opts.ExpectedRevision == "" {
		return "", fmt.Errorf("construct_path requires a non-empty expected_revision")
	}
	if opts.ExpectedRevision != "" && goReadRevision(content) != opts.ExpectedRevision {
		return "", fmt.Errorf("selected source revision mismatch: expected %q, current %q", opts.ExpectedRevision, goReadRevision(content))
	}
	recv, ownerName, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return "", fmt.Errorf("parse in_function selector: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	target := findExtractOwner(file, ownerName, recv)
	if target == nil {
		return "", &symbol.SymbolError{Op: "extract_variable", File: cleanPath, Symbol: owner, Err: symbol.ErrNotFound}
	}
	if target.Body == nil {
		return "", fmt.Errorf("cannot extract from %s: function has no body", owner)
	}
	for _, group := range file.Comments {
		if group.Pos() > target.Body.Lbrace && group.End() < target.Body.Rbrace {
			return "", fmt.Errorf("cannot extract from %s: comments inside the owner function require manual comment ownership review", owner)
		}
	}
	candidates, err := findVariableCandidates(fset, content, target, opts.Discriminator)
	if err != nil {
		return "", err
	}
	candidate, err := chooseVariableCandidate(candidates, opts.ConstructPath)
	if err != nil {
		return "", err
	}
	if !sideEffectFree(candidate.expr) {
		return "", fmt.Errorf("cannot extract expression %q: only side-effect-free expressions without calls, receives, address-taking, or composite literals are supported", opts.Discriminator)
	}
	if !hasTypedInput(candidate.expr, target) {
		return "", fmt.Errorf("cannot extract expression %q: its Go type may depend on call context or defaulting when assigned to a local", opts.Discriminator)
	}
	if variableNameVisible(file, target, name) {
		return "", fmt.Errorf("cannot extract: local name %q collides with a name visible in %s", name, owner)
	}
	if !resolvedStaticFunctionCallee(file, target, candidate.stmt) {
		return "", fmt.Errorf("cannot extract expression %q: only an unshadowed same-file package function call with no unresolved import binding is supported", opts.Discriminator)
	}

	exprStart := fset.Position(candidate.expr.Pos()).Offset
	exprEnd := fset.Position(candidate.expr.End()).Offset
	stmtStart := fset.Position(candidate.stmt.Pos()).Offset
	if exprStart < stmtStart || exprEnd > len(content) || stmtStart > exprStart {
		return "", fmt.Errorf("cannot extract expression: source positions are inconsistent")
	}
	updated := make([]byte, 0, len(content)+len(name)+len(opts.Discriminator)+16)
	updated = append(updated, content[:stmtStart]...)
	updated = append(updated, []byte(name+" := "+opts.Discriminator+"\n")...)
	updated = append(updated, content[stmtStart:exprStart]...)
	updated = append(updated, []byte(name)...)
	updated = append(updated, content[exprEnd:]...)
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: opts.Discriminator, Cause: err, Err: ErrSyntax}
	}
	formatted = normalizeExtractionNewlines(formatted)
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: opts.Discriminator, Cause: err, Err: ErrSyntax}
	}
	if err := validateVariableCandidate(ctx, cleanPath, owner, name, opts.Discriminator, candidate.path, content, formatted); err != nil {
		return "", err
	}
	buffer := sourcebuf.FromContext(ctx)
	if err := buffer.Apply(ctx, []sourcebuf.Change{{Path: cleanPath, Before: content, After: formatted}}); err != nil {
		return "", fmt.Errorf("apply extracted variable after compiler preflight: %w", err)
	}
	finalSource, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read formatted extraction result: %w", err)
	}
	if standalone {
		if err := ownedBuffer.Flush(ctx); err != nil {
			return "", fmt.Errorf("publish extracted variable: %w", err)
		}
	}
	return UnifiedDiff(filepath.Base(cleanPath), string(content), string(finalSource)), nil
}

func findVariableCandidates(fset *token.FileSet, content []byte, target *ast.FuncDecl, discriminator string) ([]variableCandidate, error) {
	var candidates []variableCandidate
	for index, stmt := range target.Body.List {
		exprStmt, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			continue
		}
		if _, ok := call.Fun.(*ast.Ident); !ok {
			continue
		}
		expr := call.Args[0]
		start := fset.Position(expr.Pos()).Offset
		end := fset.Position(expr.End()).Offset
		if start < 0 || end < start || end > len(content) {
			return nil, fmt.Errorf("cannot extract expression: source positions are inconsistent")
		}
		if string(content[start:end]) != discriminator {
			continue
		}
		candidates = append(candidates, variableCandidate{path: strconv.Itoa(index), stmt: exprStmt, expr: expr})
	}
	return candidates, nil
}

func chooseVariableCandidate(candidates []variableCandidate, path string) (variableCandidate, error) {
	if len(candidates) == 0 {
		return variableCandidate{}, fmt.Errorf("no supported first argument of a direct call exactly matches discriminator")
	}
	if path != "" {
		for _, candidate := range candidates {
			if candidate.path == path {
				return candidate, nil
			}
		}
		return variableCandidate{}, fmt.Errorf("construct_path %q does not select a matching expression; candidates: %s", path, variablePaths(candidates))
	}
	if len(candidates) != 1 {
		return variableCandidate{}, fmt.Errorf("expression selection is ambiguous; set construct_path to one of: %s", variablePaths(candidates))
	}
	return candidates[0], nil
}

func variablePaths(candidates []variableCandidate) string {
	paths := make([]string, len(candidates))
	for i, candidate := range candidates {
		paths[i] = candidate.path
	}
	return strings.Join(paths, ", ")
}

func sideEffectFree(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return sideEffectFree(node.X)
	case *ast.UnaryExpr:
		return node.Op != token.AND && node.Op != token.ARROW && sideEffectFree(node.X)
	case *ast.BinaryExpr:
		return sideEffectFree(node.X) && sideEffectFree(node.Y)
	case *ast.SelectorExpr:
		return sideEffectFree(node.X)
	case *ast.IndexExpr:
		return sideEffectFree(node.X) && sideEffectFree(node.Index)
	case *ast.TypeAssertExpr:
		return sideEffectFree(node.X)
	default:
		return false
	}
}

func hasTypedInput(expr ast.Expr, target *ast.FuncDecl) bool {
	inputs := make(map[string]bool)
	for _, fields := range []*ast.FieldList{target.Type.Params, target.Recv} {
		if fields == nil {
			continue
		}
		for _, field := range fields.List {
			for _, name := range field.Names {
				inputs[name.Name] = true
			}
		}
	}
	return referencesTypedInput(expr, inputs) && preservesExtractedType(expr, inputs)
}

func preservesExtractedType(expr ast.Expr, inputs map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.ParenExpr:
		return preservesExtractedType(node.X, inputs)
	case *ast.UnaryExpr:
		return preservesExtractedType(node.X, inputs)
	case *ast.BinaryExpr:
		if node.Op == token.SHL || node.Op == token.SHR {
			return referencesTypedInput(node.X, inputs) && preservesExtractedType(node.X, inputs)
		}
	default:
		return true
	}
	return true
}

func referencesTypedInput(expr ast.Expr, inputs map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.Ident:
		return inputs[node.Name]
	case *ast.ParenExpr:
		return referencesTypedInput(node.X, inputs)
	case *ast.UnaryExpr:
		return referencesTypedInput(node.X, inputs)
	case *ast.BinaryExpr:
		return referencesTypedInput(node.X, inputs) || referencesTypedInput(node.Y, inputs)
	case *ast.SelectorExpr:
		return referencesTypedInput(node.X, inputs)
	case *ast.IndexExpr:
		return referencesTypedInput(node.X, inputs) || referencesTypedInput(node.Index, inputs)
	case *ast.TypeAssertExpr:
		return referencesTypedInput(node.X, inputs)
	default:
		return false
	}
}

func variableNameVisible(file *ast.File, target *ast.FuncDecl, name string) bool {
	visible := false
	ast.Inspect(target, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && ident.Name == name {
			visible = true
		}
		return !visible
	})
	if visible {
		return true
	}
	for _, decl := range file.Decls {
		if nodeDeclaresName(decl, name) {
			return true
		}
	}

	return false
}

func resolvedStaticFunctionCallee(file *ast.File, target *ast.FuncDecl, stmt *ast.ExprStmt) bool {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	callee, ok := call.Fun.(*ast.Ident)
	if !ok || parameterHasName(target.Type.Params, callee.Name) || fieldListHasName(target.Recv, callee.Name) || localBindings(target.Body)[callee.Name] {
		return false
	}
	for _, imported := range file.Imports {
		if imported.Name == nil || imported.Name.Name == "." {
			return false
		}
		if imported.Name.Name == callee.Name {
			return false
		}
	}
	matches := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == callee.Name {
			matches++
		}
	}
	return matches == 1
}

func validateVariableCandidate(ctx context.Context, filePath, owner, name, discriminator, constructPath string, original, candidate []byte) error {
	cleanPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("resolve variable extraction validation path: %w", err)
	}
	loadTarget := func(source []byte) (*ast.FuncDecl, *types.Info, *token.FileSet, error) {
		overlay := sourcebuf.Overlay(ctx)
		if overlay == nil {
			overlay = make(map[string][]byte)
		}
		overlay[cleanPath] = bytes.Clone(source)
		loaded, loadErr := loadExtractionPackages(ctx, filepath.Dir(cleanPath), overlay, false)
		if loadErr != nil {
			return nil, nil, nil, fmt.Errorf("type-check variable extraction candidate: %w", loadErr)
		}
		matches := 0
		var syntax *ast.File
		var info *types.Info
		var fset *token.FileSet
		for _, pkg := range loaded {
			if pkg.TypesInfo == nil {
				continue
			}
			for index, compiled := range pkg.CompiledGoFiles {
				path, pathErr := filepath.Abs(compiled)
				if pathErr == nil && filepath.Clean(path) == cleanPath {
					if index >= len(pkg.Syntax) {
						return nil, nil, nil, fmt.Errorf("type-check variable extraction candidate: package syntax is incomplete")
					}
					matches++
					syntax, info, fset = pkg.Syntax[index], pkg.TypesInfo, pkg.Fset
				}
			}
		}
		if matches != 1 || syntax == nil || info == nil || fset == nil {
			return nil, nil, nil, fmt.Errorf("type-check variable extraction candidate: selected file resolves in %d package variants", matches)
		}
		receiver, function, parseErr := symbol.ParseIdentifier(owner)
		if parseErr != nil {
			return nil, nil, nil, fmt.Errorf("parse in_function selector: %w", parseErr)
		}
		target := findExtractOwner(syntax, function, receiver)
		if target == nil || target.Body == nil {
			return nil, nil, nil, fmt.Errorf("type-check variable extraction candidate: owner %q was not found", owner)
		}
		return target, info, fset, nil
	}
	originalOwner, originalInfo, originalFset, err := loadTarget(original)
	if err != nil {
		return err
	}
	originalCandidates, err := findVariableCandidates(originalFset, original, originalOwner, discriminator)
	if err != nil {
		return fmt.Errorf("resolve original variable extraction type: %w", err)
	}
	originalSelection, err := chooseVariableCandidate(originalCandidates, constructPath)
	if err != nil {
		return fmt.Errorf("resolve original variable extraction type: %w", err)
	}
	originalType := originalInfo.Types[originalSelection.expr].Type
	if originalType == nil {
		return fmt.Errorf("cannot extract expression %q: Go type information is unavailable", discriminator)
	}
	candidateOwner, candidateInfo, _, err := loadTarget(candidate)
	if err != nil {
		return err
	}
	var candidateType types.Type
	matches := 0
	for _, statement := range candidateOwner.Body.List {
		expression, ok := statement.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expression.X.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			continue
		}
		ident, ok := call.Args[0].(*ast.Ident)
		if !ok || ident.Name != name {
			continue
		}
		matches++
		candidateType = candidateInfo.Types[ident].Type
	}
	if matches != 1 || candidateType == nil {
		return fmt.Errorf("cannot extract expression %q: generated local has no unique resolved type", discriminator)
	}
	typeIdentity := func(value types.Type) string {
		return types.TypeString(value, func(pkg *types.Package) string {
			if pkg == nil {
				return ""
			}
			return pkg.Path()
		})
	}
	originalIdentity := typeIdentity(originalType)
	candidateIdentity := typeIdentity(candidateType)
	if originalIdentity != candidateIdentity {
		return fmt.Errorf("cannot extract expression %q: generated local changes Go type from %s to %s", discriminator, originalIdentity, candidateIdentity)
	}
	return nil
}
