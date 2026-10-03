// Package astedit replaces complete named declarations because callers need explicit identity checks before replacing their syntax.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

type parsedFunctionReplacement struct {
	receiver    string
	name        string
	fset        *token.FileSet
	source      string
	declaration *ast.FuncDecl
}

func replaceDeclarationConstruct(ctx context.Context, filePath, symbolQuery string, kind ConstructKind, source string, opts ConstructOptions) (string, error) {
	if strings.TrimSpace(symbolQuery) == "" {
		return "", fmt.Errorf("symbol is required for %s replacement", kind)
	}
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	decls, err := verifySnippetSyntax(source, "")
	if err != nil {
		return "", fmt.Errorf("validate replacement declaration: %w", err)
	}
	if len(decls) != 1 {
		return "", ErrMultipleDeclarations
	}
	replacementDecl := decls[0]
	if kind == ConstructFunction || kind == ConstructMethod {
		return replaceFunctionConstruct(ctx, cleanPath, content, fset, file, symbolQuery, kind, source, opts)
	}
	if kind == ConstructConst || kind == ConstructVar || kind == ConstructType {
		gen, ok := replacementDecl.(*ast.GenDecl)
		if !ok {
			return "", fmt.Errorf("replacement source must be one const, var, or type declaration")
		}
		if (kind == ConstructType && gen.Tok != token.TYPE) || (kind == ConstructConst && gen.Tok != token.CONST) || (kind == ConstructVar && gen.Tok != token.VAR) {
			return "", fmt.Errorf("replacement declaration does not match construct kind %q", kind)
		}
		targetKind, found, kindErr := namedDeclarationKind(file, symbolQuery)
		if kindErr != nil {
			return "", kindErr
		}
		if !found {
			return "", fmt.Errorf("declaration %q not found in %s", symbolQuery, cleanPath)
		}
		if targetKind != kind {
			return "", fmt.Errorf("selected declaration %q is %s, not %s", symbolQuery, targetKind, kind)
		}
		return ReplaceDecl(ctx, cleanPath, symbolQuery, source, ReplaceDeclOptions{AutoOrganizeImports: opts.AutoOrganizeImports})
	}
	return "", fmt.Errorf("unsupported Go declaration construct kind %q", kind)
}

func replaceFunctionConstruct(ctx context.Context, cleanPath string, content []byte, fset *token.FileSet, file *ast.File, symbolQuery string, kind ConstructKind, source string, opts ConstructOptions) (string, error) {
	replacement, err := parseFunctionReplacement(symbolQuery, kind, source)
	if err != nil {
		return "", err
	}
	target, err := findFunctionConstructTarget(file, replacement, symbolQuery, kind, cleanPath)
	if err != nil {
		return "", err
	}
	if kind == ConstructMethod && receiverTypeKey(target.Recv) != receiverTypeKey(replacement.declaration.Recv) {
		return "", fmt.Errorf("replacement method receiver must preserve the selected receiver syntax")
	}
	filtered := *file
	filtered.Decls = slices.DeleteFunc(slices.Clone(file.Decls), func(decl ast.Decl) bool { return decl == target })
	if collisionErr := checkConstructDeclarationCollision(ctx, cleanPath, &filtered, replacement.declaration); collisionErr != nil {
		return "", fmt.Errorf("check replacement declaration collision: %w", collisionErr)
	}
	startPos := target.Pos()
	if target.Doc != nil && replacement.declaration.Doc != nil {
		startPos = target.Doc.Pos()
	}
	start, end := fset.Position(startPos).Offset, fset.Position(target.End()).Offset
	replacementBytes, err := functionReplacementBytes(replacement)
	if err != nil {
		return "", err
	}
	updated := replaceBytes(content, start, end, replacementBytes)
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	formatted, err = cleanupMoveImports(ctx, cleanPath, formatted)
	if err != nil {
		return "", fmt.Errorf("clean imports after construct replacement: %w", err)
	}
	formatted, err = format.Source(formatted)
	if err != nil {
		return "", fmt.Errorf("format import-cleaned source: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if err := pipeline.WriteAtomicContext(ctx, cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	}
	return UnifiedDiff(symbolQuery, strings.TrimSpace(string(content[start:end])), strings.TrimSpace(string(replacementBytes))), nil
}

func parseFunctionReplacement(symbolQuery string, kind ConstructKind, source string) (parsedFunctionReplacement, error) {
	receiver, name, err := symbol.ParseIdentifier(symbolQuery)
	if err != nil {
		return parsedFunctionReplacement{}, fmt.Errorf("parse symbol %q: %w", symbolQuery, err)
	}
	replacementSource := strings.TrimSpace(source)
	if !strings.HasPrefix(replacementSource, "package ") {
		replacementSource = "package replacement\n\n" + replacementSource
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "replacement.go", []byte(replacementSource), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return parsedFunctionReplacement{}, &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if len(file.Decls) != 1 {
		return parsedFunctionReplacement{}, ErrMultipleDeclarations
	}
	declaration, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		return parsedFunctionReplacement{}, fmt.Errorf("replacement source must be one function or method declaration")
	}
	if (kind == ConstructMethod) != (declaration.Recv != nil) || declaration.Name.Name != name {
		return parsedFunctionReplacement{}, fmt.Errorf("replacement must preserve the selected %s symbol %q", kind, symbolQuery)
	}
	if kind == ConstructMethod && extractReceiverName(declaration.Recv) != receiver {
		return parsedFunctionReplacement{}, fmt.Errorf("replacement method receiver must preserve %q", receiver)
	}
	return parsedFunctionReplacement{receiver: receiver, name: name, fset: fset, source: replacementSource, declaration: declaration}, nil
}

func findFunctionConstructTarget(file *ast.File, replacement parsedFunctionReplacement, symbolQuery string, kind ConstructKind, cleanPath string) (*ast.FuncDecl, error) {
	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != replacement.name || extractReceiverName(function.Recv) != replacement.receiver || (kind == ConstructMethod) != (function.Recv != nil) {
			continue
		}
		if target != nil {
			return nil, fmt.Errorf("symbol %q is ambiguous in %s", symbolQuery, cleanPath)
		}
		target = function
	}
	if target == nil {
		return nil, fmt.Errorf("%s %q not found in %s", kind, symbolQuery, cleanPath)
	}
	return target, nil
}

func functionReplacementBytes(replacement parsedFunctionReplacement) ([]byte, error) {
	start := replacement.fset.Position(replacement.declaration.Pos()).Offset
	end := replacement.fset.Position(replacement.declaration.End()).Offset
	if replacement.declaration.Doc != nil {
		start = replacement.fset.Position(replacement.declaration.Doc.Pos()).Offset
	}
	source := []byte(replacement.source)
	if start < 0 || end > len(source) || start >= end {
		return nil, fmt.Errorf("replacement function source span is invalid")
	}
	formatted, err := format.Source(source[start:end])
	if err != nil {
		return nil, fmt.Errorf("format replacement function: %w", err)
	}
	return formatted, nil
}

func namedDeclarationKind(file *ast.File, name string) (ConstructKind, bool, error) {
	var found ConstructKind
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			var candidate string
			switch value := spec.(type) {
			case *ast.TypeSpec:
				candidate = value.Name.Name
			case *ast.ValueSpec:
				for _, ident := range value.Names {
					if ident.Name == name {
						candidate = name
						break
					}
				}
			}
			if candidate != name {
				continue
			}
			kind := ConstructType
			switch gen.Tok {
			case token.CONST:
				kind = ConstructConst
			case token.VAR:
				kind = ConstructVar
			case token.TYPE:
			default:
				continue
			}
			if found != "" {
				return "", false, fmt.Errorf("declaration %q is ambiguous", name)
			}
			found = kind
		}
	}
	return found, found != "", nil
}

func receiverTypeKey(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) != 1 {
		return ""
	}
	return receiverExprKey(fields.List[0].Type)
}

func receiverExprKey(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return "*" + receiverExprKey(value.X)
	case *ast.ParenExpr:
		return "(" + receiverExprKey(value.X) + ")"
	case *ast.IndexExpr:
		return receiverExprKey(value.X) + "[" + receiverExprKey(value.Index) + "]"
	case *ast.IndexListExpr:
		items := make([]string, len(value.Indices))
		for i, item := range value.Indices {
			items[i] = receiverExprKey(item)
		}
		return receiverExprKey(value.X) + "[" + strings.Join(items, ",") + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}
