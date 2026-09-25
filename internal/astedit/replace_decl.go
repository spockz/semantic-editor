// Package astedit implements declaration edits because package-level symbols
// need syntax-aware replacement and collision checks before writing source.
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
	"strings"

	"semedit/internal/pipeline"
)

// ReplaceDeclOptions configures package declaration replacement.
type ReplaceDeclOptions struct {
	AutoOrganizeImports bool
}

// ReplaceDecl replaces one existing package constant, variable, or type alias.
func ReplaceDecl(ctx context.Context, filePath, symbolQuery, source string, opts ReplaceDeclOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	replacement, err := parseSingleReplacementDecl(source, symbolQuery)
	if err != nil {
		return "", err
	}
	type declMatch struct {
		decl ast.Decl
		gen  *ast.GenDecl
		spec ast.Spec
		name string
	}
	var matches []declMatch
	for _, decl := range fileNode.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				if gen.Tok != token.CONST && gen.Tok != token.VAR {
					continue
				}
				for _, ident := range s.Names {
					if ident.Name == symbolQuery {
						matches = append(matches, declMatch{decl: decl, gen: gen, spec: spec, name: ident.Name})
					}
				}
			case *ast.TypeSpec:
				if gen.Tok == token.TYPE && s.Assign.IsValid() && s.Name.Name == symbolQuery {
					matches = append(matches, declMatch{decl: decl, gen: gen, spec: spec, name: s.Name.Name})
				}
			}
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("declaration %q not found in %s: %w", symbolQuery, cleanPath, ErrSymbolNotFound)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("declaration %q is ambiguous in %s", symbolQuery, cleanPath)
	}
	match := matches[0]
	if match.gen.Tok != replacement.kind {
		return "", fmt.Errorf("replacement declaration %q has kind %s; existing declaration uses %s", symbolQuery, replacement.kind, match.gen.Tok)
	}
	if value, ok := match.spec.(*ast.ValueSpec); ok && match.gen.Tok == token.CONST {
		if len(value.Names) != 1 {
			return "", fmt.Errorf("cannot replace %q inside a shared const spec; split the spec first", symbolQuery)
		}
		for _, later := range match.gen.Specs {
			if later.Pos() <= value.Pos() {
				continue
			}
			if next, ok := later.(*ast.ValueSpec); ok && len(next.Values) == 0 {
				return "", fmt.Errorf("cannot replace %q: later const spec inherits its expression and would change", symbolQuery)
			}
		}
	}
	if value, ok := match.spec.(*ast.ValueSpec); ok && len(value.Names) != 1 {
		return "", fmt.Errorf("cannot replace %q inside a shared value spec; split the spec first", symbolQuery)
	}
	var newContent []byte
	if len(match.gen.Specs) == 1 && !match.gen.Lparen.IsValid() {
		start, end := fset.Position(match.decl.Pos()).Offset, fset.Position(match.decl.End()).Offset
		newContent = replaceBytes(content, start, end, []byte(replacement.text))
	} else {
		start := fset.Position(match.spec.Pos()).Offset
		end := fset.Position(match.spec.End()).Offset
		newContent = replaceBytes(content, start, end, []byte(replacement.specText))
	}
	formatted, err := format.Source(newContent)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	} else if err := pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
		return "", fmt.Errorf("format file: %w", err)
	}
	oldStart, oldEnd := fset.Position(match.spec.Pos()).Offset, fset.Position(match.spec.End()).Offset
	return UnifiedDiff(symbolQuery, strings.TrimSpace(string(content[oldStart:oldEnd])), strings.TrimSpace(replacement.specText)), nil
}

type replacementDecl struct {
	kind           token.Token
	text, specText string
}

func parseSingleReplacementDecl(source, symbol string) (replacementDecl, error) {
	decls, err := verifySnippetSyntax(source, "")
	if err != nil {
		return replacementDecl{}, fmt.Errorf("validate replacement declaration: %w", err)
	}
	if len(decls) != 1 {
		return replacementDecl{}, ErrMultipleDeclarations
	}
	gen, ok := decls[0].(*ast.GenDecl)
	if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR && gen.Tok != token.TYPE) {
		return replacementDecl{}, ErrUnexpectedDeclType
	}
	if len(gen.Specs) != 1 {
		return replacementDecl{}, ErrMultipleDeclarations
	}
	spec := gen.Specs[0]
	declared := extractDeclNames(gen)
	if len(declared) != 1 || declared[0] != symbol {
		return replacementDecl{}, fmt.Errorf("replacement must declare only requested symbol %q", symbol)
	}
	if typeSpec, ok := spec.(*ast.TypeSpec); ok && !typeSpec.Assign.IsValid() {
		return replacementDecl{}, fmt.Errorf("replacement of %q requires a type alias", symbol)
	}
	parseSource := strings.TrimSpace(source)
	if !strings.HasPrefix(parseSource, "package ") {
		parseSource = "package dummy\n\n" + parseSource
	}
	formattedSource, err := format.Source([]byte(parseSource))
	if err != nil {
		return replacementDecl{}, fmt.Errorf("format replacement declaration: %w", err)
	}
	fset := token.NewFileSet()
	formattedFile, err := parser.ParseFile(fset, "snippet", formattedSource, parser.ParseComments)
	if err != nil {
		return replacementDecl{}, fmt.Errorf("parse formatted replacement declaration: %w", err)
	}
	if len(formattedFile.Decls) != 1 {
		return replacementDecl{}, ErrMultipleDeclarations
	}
	formattedGen, ok := formattedFile.Decls[0].(*ast.GenDecl)
	if !ok {
		return replacementDecl{}, ErrUnexpectedDeclType
	}
	var formattedBuf bytes.Buffer
	if err := format.Node(&formattedBuf, fset, formattedGen); err != nil {
		return replacementDecl{}, err
	}
	formatted := formattedBuf.String()
	specText := formatted
	if after, ok := strings.CutPrefix(specText, formattedGen.Tok.String()); ok {
		specText = strings.TrimSpace(after)
	}
	if strings.HasPrefix(specText, "(") && strings.HasSuffix(specText, ")") {
		specText = strings.TrimSpace(specText[1 : len(specText)-1])
	}
	return replacementDecl{kind: formattedGen.Tok, text: formatted, specText: specText}, nil
}
