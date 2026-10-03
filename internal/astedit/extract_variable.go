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

	"golang.org/x/tools/go/packages"
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

type variableExtractionInput struct {
	ctx         context.Context
	cleanPath   string
	ownedBuffer *sourcebuf.Buffer
	standalone  bool
	content     []byte
	file        *ast.File
	fset        *token.FileSet
	target      *ast.FuncDecl
	candidate   variableCandidate
}

type variableTypeContext struct {
	owner *ast.FuncDecl
	info  *types.Info
	fset  *token.FileSet
}

// ExtractVariable moves the first side-effect-free, typed argument of an unshadowed same-file package call into a local.
func ExtractVariable(ctx context.Context, filePath, owner, name string, opts ExtractVariableOptions) (string, error) {
	if err := validateVariableExtractionRequest(name, opts); err != nil {
		return "", err
	}
	input, err := loadVariableExtractionInput(ctx, filePath, owner, opts)
	if err != nil {
		return "", err
	}
	if targetErr := validateVariableExtractionTarget(input, owner, name, opts); targetErr != nil {
		return "", targetErr
	}
	updated, err := rewriteVariableArgument(input, name, opts.Discriminator)
	if err != nil {
		return "", err
	}
	return publishVariableExtraction(input, owner, name, opts.Discriminator, updated)
}

func validateVariableExtractionRequest(name string, opts ExtractVariableOptions) error {
	if !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() || name == "_" {
		return fmt.Errorf("symbol %q is not a valid local identifier", name)
	}
	if opts.Discriminator == "" {
		return fmt.Errorf("discriminator must contain the exact source text of the expression")
	}
	return nil
}

func loadVariableExtractionInput(ctx context.Context, filePath, owner string, opts ExtractVariableOptions) (variableExtractionInput, error) {
	input := variableExtractionInput{cleanPath: filepath.Clean(filePath)}
	preparedCtx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, input.cleanPath)
	if err != nil {
		return variableExtractionInput{}, err
	}
	input.ctx, input.ownedBuffer, input.standalone = preparedCtx, ownedBuffer, standalone
	input.content, err = sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return variableExtractionInput{}, fmt.Errorf("read extraction target: %w", err)
	}
	if opts.ConstructPath != "" && opts.ExpectedRevision == "" {
		return variableExtractionInput{}, fmt.Errorf("construct_path requires a non-empty expected_revision")
	}
	if opts.ExpectedRevision != "" && goReadRevision(input.content) != opts.ExpectedRevision {
		return variableExtractionInput{}, fmt.Errorf("selected source revision mismatch: expected %q, current %q", opts.ExpectedRevision, goReadRevision(input.content))
	}
	recv, ownerName, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return variableExtractionInput{}, fmt.Errorf("parse in_function selector: %w", err)
	}
	input.fset = token.NewFileSet()
	input.file, err = parser.ParseFile(input.fset, input.cleanPath, input.content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return variableExtractionInput{}, &SyntaxError{File: input.cleanPath, Pos: extractSyntaxPosition(input.fset, err), Cause: err, Err: ErrSyntax}
	}
	input.target = findExtractOwner(input.file, ownerName, recv)
	if input.target == nil {
		return variableExtractionInput{}, &symbol.SymbolError{Op: "extract_variable", File: input.cleanPath, Symbol: owner, Err: symbol.ErrNotFound}
	}
	if input.target.Body == nil {
		return variableExtractionInput{}, fmt.Errorf("cannot extract from %s: function has no body", owner)
	}
	for _, group := range input.file.Comments {
		if group.Pos() > input.target.Body.Lbrace && group.End() < input.target.Body.Rbrace {
			return variableExtractionInput{}, fmt.Errorf("cannot extract from %s: comments inside the owner function require manual comment ownership review", owner)
		}
	}
	candidates, err := findVariableCandidates(input.fset, input.content, input.target, opts.Discriminator)
	if err != nil {
		return variableExtractionInput{}, err
	}
	input.candidate, err = chooseVariableCandidate(candidates, opts.ConstructPath)
	if err != nil {
		return variableExtractionInput{}, err
	}
	return input, nil
}

func validateVariableExtractionTarget(input variableExtractionInput, owner, name string, opts ExtractVariableOptions) error {
	if !sideEffectFree(input.candidate.expr) {
		return fmt.Errorf("cannot extract expression %q: only side-effect-free expressions without calls, receives, address-taking, or composite literals are supported", opts.Discriminator)
	}
	if !hasTypedInput(input.candidate.expr, input.target) {
		return fmt.Errorf("cannot extract expression %q: its Go type may depend on call context or defaulting when assigned to a local", opts.Discriminator)
	}
	if variableNameVisible(input.file, input.target, name) {
		return fmt.Errorf("cannot extract: local name %q collides with a name visible in %s", name, owner)
	}
	if !resolvedStaticFunctionCallee(input.file, input.target, input.candidate.stmt) {
		return fmt.Errorf("cannot extract expression %q: only an unshadowed same-file package function call with no unresolved import binding is supported", opts.Discriminator)
	}
	return nil
}

func rewriteVariableArgument(input variableExtractionInput, name, discriminator string) ([]byte, error) {
	exprStart := input.fset.Position(input.candidate.expr.Pos()).Offset
	exprEnd := input.fset.Position(input.candidate.expr.End()).Offset
	stmtStart := input.fset.Position(input.candidate.stmt.Pos()).Offset
	if exprStart < stmtStart || exprEnd > len(input.content) || stmtStart > exprStart {
		return nil, fmt.Errorf("cannot extract expression: source positions are inconsistent")
	}
	updated := make([]byte, 0, len(input.content)+len(name)+len(discriminator)+16)
	updated = append(updated, input.content[:stmtStart]...)
	updated = append(updated, []byte(name+" := "+discriminator+"\n")...)
	updated = append(updated, input.content[stmtStart:exprStart]...)
	updated = append(updated, []byte(name)...)
	updated = append(updated, input.content[exprEnd:]...)
	return updated, nil
}

func publishVariableExtraction(input variableExtractionInput, owner, name, discriminator string, updated []byte) (string, error) {
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: input.cleanPath, Snippet: discriminator, Cause: err, Err: ErrSyntax}
	}
	formatted = normalizeExtractionNewlines(formatted)
	if _, parseErr := parser.ParseFile(token.NewFileSet(), input.cleanPath, formatted, parser.ParseComments|parser.AllErrors); parseErr != nil {
		return "", &SyntaxError{File: input.cleanPath, Snippet: discriminator, Cause: parseErr, Err: ErrSyntax}
	}
	if validateErr := validateVariableCandidate(input.ctx, input.cleanPath, owner, name, discriminator, input.candidate.path, input.content, formatted); validateErr != nil {
		return "", validateErr
	}
	buffer := sourcebuf.FromContext(input.ctx)
	if applyErr := buffer.Apply(input.ctx, []sourcebuf.Change{{Path: input.cleanPath, Before: input.content, After: formatted}}); applyErr != nil {
		return "", fmt.Errorf("apply extracted variable after compiler preflight: %w", applyErr)
	}
	finalSource, err := sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return "", fmt.Errorf("read formatted extraction result: %w", err)
	}
	if input.standalone {
		if flushErr := input.ownedBuffer.Flush(input.ctx); flushErr != nil {
			return "", fmt.Errorf("publish extracted variable: %w", flushErr)
		}
	}
	return UnifiedDiff(filepath.Base(input.cleanPath), string(input.content), string(finalSource)), nil
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
	originalContext, err := loadVariableTypeContext(ctx, cleanPath, owner, original)
	if err != nil {
		return err
	}
	originalType, err := originalVariableExpressionType(originalContext, original, discriminator, constructPath)
	if err != nil {
		return err
	}
	candidateContext, err := loadVariableTypeContext(ctx, cleanPath, owner, candidate)
	if err != nil {
		return err
	}
	candidateType, err := generatedVariableExpressionType(candidateContext, name, discriminator)
	if err != nil {
		return err
	}
	if !sameResolvedVariableType(originalType, candidateType) {
		return fmt.Errorf("cannot extract expression %q: generated local changes Go type from %s to %s", discriminator, extractionTypeIdentity(originalType), extractionTypeIdentity(candidateType))
	}
	return nil
}

func loadVariableTypeContext(ctx context.Context, cleanPath, owner string, source []byte) (variableTypeContext, error) {
	overlay := sourcebuf.Overlay(ctx)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[cleanPath] = bytes.Clone(source)
	loaded, err := loadExtractionPackages(ctx, filepath.Dir(cleanPath), overlay, false)
	if err != nil {
		return variableTypeContext{}, fmt.Errorf("type-check variable extraction candidate: %w", err)
	}
	parsed, err := selectVariableTypeSyntax(loaded, cleanPath)
	if err != nil {
		return variableTypeContext{}, err
	}
	receiver, function, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return variableTypeContext{}, fmt.Errorf("parse in_function selector: %w", err)
	}
	target := findExtractOwner(parsed.syntax, function, receiver)
	if target == nil || target.Body == nil {
		return variableTypeContext{}, fmt.Errorf("type-check variable extraction candidate: owner %q was not found", owner)
	}
	return variableTypeContext{owner: target, info: parsed.info, fset: parsed.fset}, nil
}

type variableTypeSyntax struct {
	syntax *ast.File
	info   *types.Info
	fset   *token.FileSet
}

func selectVariableTypeSyntax(loaded []*packages.Package, cleanPath string) (variableTypeSyntax, error) {
	matches := 0
	var selected variableTypeSyntax
	for _, pkg := range loaded {
		if pkg.TypesInfo == nil {
			continue
		}
		for index, compiled := range pkg.CompiledGoFiles {
			path, err := filepath.Abs(compiled)
			if err != nil || filepath.Clean(path) != cleanPath {
				continue
			}
			if index >= len(pkg.Syntax) {
				return variableTypeSyntax{}, fmt.Errorf("type-check variable extraction candidate: package syntax is incomplete")
			}
			matches++
			selected = variableTypeSyntax{syntax: pkg.Syntax[index], info: pkg.TypesInfo, fset: pkg.Fset}
		}
	}
	if matches != 1 || selected.syntax == nil || selected.info == nil || selected.fset == nil {
		return variableTypeSyntax{}, fmt.Errorf("type-check variable extraction candidate: selected file resolves in %d package variants", matches)
	}
	return selected, nil
}

func originalVariableExpressionType(context variableTypeContext, source []byte, discriminator, constructPath string) (types.Type, error) {
	candidates, err := findVariableCandidates(context.fset, source, context.owner, discriminator)
	if err != nil {
		return nil, fmt.Errorf("resolve original variable extraction type: %w", err)
	}
	selection, err := chooseVariableCandidate(candidates, constructPath)
	if err != nil {
		return nil, fmt.Errorf("resolve original variable extraction type: %w", err)
	}
	typeOf := context.info.Types[selection.expr].Type
	if typeOf == nil {
		return nil, fmt.Errorf("cannot extract expression %q: Go type information is unavailable", discriminator)
	}
	return typeOf, nil
}

func generatedVariableExpressionType(context variableTypeContext, name, discriminator string) (types.Type, error) {
	var candidateType types.Type
	matches := 0
	for _, statement := range context.owner.Body.List {
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
		candidateType = context.info.Types[ident].Type
	}
	if matches != 1 || candidateType == nil {
		return nil, fmt.Errorf("cannot extract expression %q: generated local has no unique resolved type", discriminator)
	}
	return candidateType, nil
}

func sameResolvedVariableType(original, candidate types.Type) bool {
	return extractionTypeIdentity(original) == extractionTypeIdentity(candidate)
}

func extractionTypeIdentity(value types.Type) string {
	return types.TypeString(value, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
}
