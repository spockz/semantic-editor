// This file replaces complete named declarations because callers need explicit identity checks before replacing their syntax.
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
	"slices"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

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
		if _, ok := replacementDecl.(*ast.GenDecl); !ok {
			return "", fmt.Errorf("replacement source must be one const, var, or type declaration")
		}
		return ReplaceDecl(ctx, cleanPath, symbolQuery, source, ReplaceDeclOptions{AutoOrganizeImports: opts.AutoOrganizeImports})
	}
	return "", fmt.Errorf("unsupported Go declaration construct kind %q", kind)
}

func replaceFunctionConstruct(ctx context.Context, cleanPath string, content []byte, fset *token.FileSet, file *ast.File, symbolQuery string, kind ConstructKind, source string, opts ConstructOptions) (string, error) {
	receiver, name, err := symbol.ParseIdentifier(symbolQuery)
	if err != nil {
		return "", fmt.Errorf("parse symbol %q: %w", symbolQuery, err)
	}
	replacementFset := token.NewFileSet()
	replacementSource := strings.TrimSpace(source)
	if !strings.HasPrefix(replacementSource, "package ") {
		replacementSource = "package replacement\n\n" + replacementSource
	}
	replacementFile, err := parser.ParseFile(replacementFset, "replacement.go", []byte(replacementSource), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if len(replacementFile.Decls) != 1 {
		return "", ErrMultipleDeclarations
	}
	replacement, ok := replacementFile.Decls[0].(*ast.FuncDecl)
	if !ok {
		return "", fmt.Errorf("replacement source must be one function or method declaration")
	}
	if (kind == ConstructMethod) != (replacement.Recv != nil) || replacement.Name.Name != name {
		return "", fmt.Errorf("replacement must preserve the selected %s symbol %q", kind, symbolQuery)
	}
	if kind == ConstructMethod && extractReceiverName(replacement.Recv) != receiver {
		return "", fmt.Errorf("replacement method receiver must preserve %q", receiver)
	}
	var target *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name || extractReceiverName(fn.Recv) != receiver || (kind == ConstructMethod) != (fn.Recv != nil) {
			continue
		}
		if target != nil {
			return "", fmt.Errorf("symbol %q is ambiguous in %s", symbolQuery, cleanPath)
		}
		target = fn
	}
	if target == nil {
		return "", fmt.Errorf("%s %q not found in %s", kind, symbolQuery, cleanPath)
	}
	if kind == ConstructMethod && receiverTypeKey(target.Recv) != receiverTypeKey(replacement.Recv) {
		return "", fmt.Errorf("replacement method receiver must preserve the selected receiver syntax")
	}
	filtered := *file
	filtered.Decls = slices.DeleteFunc(slices.Clone(file.Decls), func(decl ast.Decl) bool { return decl == target })
	if err := checkConstructDeclarationCollision(ctx, cleanPath, &filtered, replacement); err != nil {
		return "", fmt.Errorf("check replacement declaration collision: %w", err)
	}
	startPos := target.Pos()
	if target.Doc != nil && replacement.Doc != nil {
		startPos = target.Doc.Pos()
	}
	start, end := fset.Position(startPos).Offset, fset.Position(target.End()).Offset
	var replacementText bytes.Buffer
	if err := format.Node(&replacementText, replacementFset, replacement); err != nil {
		return "", fmt.Errorf("format replacement function: %w", err)
	}
	updated := replaceBytes(content, start, end, replacementText.Bytes())
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if opts.AutoOrganizeImports {
		formatted, err = cleanupMoveImports(ctx, cleanPath, formatted)
		if err != nil {
			return "", fmt.Errorf("clean imports after construct replacement: %w", err)
		}
		formatted, err = format.Source(formatted)
		if err != nil {
			return "", fmt.Errorf("format import-cleaned source: %w", err)
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if err := pipeline.WriteAtomicContext(ctx, cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if !opts.AutoOrganizeImports {
		if err := pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			return "", fmt.Errorf("format file: %w", err)
		}
	}
	return UnifiedDiff(symbolQuery, strings.TrimSpace(string(content[start:end])), strings.TrimSpace(replacementText.String())), nil
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
