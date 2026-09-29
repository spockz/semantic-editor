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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// ExtractVariableOptions selects one exact argument expression in the owner's direct body.
type ExtractVariableOptions struct {
	Discriminator string
	ConstructPath string
}

type variableCandidate struct {
	path string
	stmt *ast.ExprStmt
	call *ast.CallExpr
	expr ast.Expr
}

// ExtractVariable moves one side-effect-free, typed argument expression immediately before its call.
func ExtractVariable(ctx context.Context, filePath, owner, name string, opts ExtractVariableOptions) (string, error) {
	if !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() || name == "_" {
		return "", fmt.Errorf("symbol %q is not a valid local identifier", name)
	}
	if opts.Discriminator == "" {
		return "", fmt.Errorf("discriminator must contain the exact source text of the expression")
	}
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read extraction target: %w", err)
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
	formatted = preserveTrailingNewlines(content, formatted)
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: opts.Discriminator, Cause: err, Err: ErrSyntax}
	}
	current, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("recheck extraction target before publication: %w", err)
	}
	if !bytes.Equal(current, content) {
		return "", fmt.Errorf("cannot extract from %s: source changed during extraction", owner)
	}
	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write extracted variable atomically: %w", err)
	}
	if err := pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
		return "", fmt.Errorf("format extracted variable: %w", err)
	}
	finalSource, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read formatted extraction result: %w", err)
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
		if !ok || len(call.Args) != 1 {
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
		candidates = append(candidates, variableCandidate{path: strconv.Itoa(index), stmt: exprStmt, call: call, expr: expr})
	}
	return candidates, nil
}

func chooseVariableCandidate(candidates []variableCandidate, path string) (variableCandidate, error) {
	if len(candidates) == 0 {
		return variableCandidate{}, fmt.Errorf("no supported direct call argument exactly matches discriminator")
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
		switch declaration := decl.(type) {
		case *ast.FuncDecl:
			if declaration.Name.Name == name {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch item := spec.(type) {
				case *ast.ValueSpec:
					for _, ident := range item.Names {
						if ident.Name == name {
							return true
						}
					}
				case *ast.TypeSpec:
					if item.Name.Name == name {
						return true
					}
				}
			}
		}
	}

	return false
}
