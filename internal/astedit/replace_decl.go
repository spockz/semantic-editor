// Package astedit implements declaration edits because package-level symbols
// need syntax-aware replacement and collision checks before writing source.
package astedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// ReplaceDeclOptions configures package declaration replacement.
type ReplaceDeclOptions struct {
	AutoOrganizeImports bool
}

// ReplaceDecl replaces one existing package constant, variable, or type alias.
func ReplaceDecl(ctx context.Context, filePath, symbolQuery, source string, opts ReplaceDeclOptions) (string, error) {
	if sourcebuf.FromContext(ctx) == nil {
		var result string
		err := runBufferedFile(ctx, filePath, func(editCtx context.Context) error {
			var err error
			result, err = ReplaceDecl(editCtx, filePath, symbolQuery, source, opts)
			return err
		})
		return result, err
	}
	target, err := loadDeclarationReplacementTarget(ctx, filePath)
	if err != nil {
		return "", err
	}
	replacement, err := parseSingleReplacementDecl(source, symbolQuery)
	if err != nil {
		return "", err
	}
	match, err := findDeclarationReplacementMatch(target.file, symbolQuery, target.path)
	if err != nil {
		return "", err
	}
	if err := validateDeclarationReplacement(match, replacement, symbolQuery); err != nil {
		return "", err
	}
	span := declarationReplacementSpan(target, match, replacement)
	if err := publishDeclarationReplacement(ctx, target.path, target.content, source, span, opts.AutoOrganizeImports); err != nil {
		return "", err
	}
	previous := strings.TrimSpace(string(target.content[span.start:span.end]))
	return UnifiedDiff(symbolQuery, previous, strings.TrimSpace(span.text)), nil
}

type declarationReplacementTarget struct {
	path    string
	content []byte
	fset    *token.FileSet
	file    *ast.File
}

type declarationReplacementMatch struct {
	declaration ast.Decl
	group       *ast.GenDecl
	spec        ast.Spec
}

type declarationReplacementSource struct {
	start int
	end   int
	text  string
}

func loadDeclarationReplacementTarget(ctx context.Context, filePath string) (declarationReplacementTarget, error) {
	path := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return declarationReplacementTarget{}, fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return declarationReplacementTarget{}, &SyntaxError{File: path, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	return declarationReplacementTarget{path: path, content: content, fset: fset, file: file}, nil
}

func findDeclarationReplacementMatch(file *ast.File, symbol, path string) (declarationReplacementMatch, error) {
	matches := collectDeclarationReplacementMatches(file, symbol)
	if len(matches) == 0 {
		return declarationReplacementMatch{}, fmt.Errorf("declaration %q not found in %s: %w", symbol, path, ErrSymbolNotFound)
	}
	if len(matches) > 1 {
		return declarationReplacementMatch{}, fmt.Errorf("declaration %q is ambiguous in %s", symbol, path)
	}
	return matches[0], nil
}

func collectDeclarationReplacementMatches(file *ast.File, symbol string) []declarationReplacementMatch {
	var matches []declarationReplacementMatch
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range group.Specs {
			if declarationSpecHasName(group, spec, symbol) {
				matches = append(matches, declarationReplacementMatch{declaration: declaration, group: group, spec: spec})
			}
		}
	}
	return matches
}

func declarationSpecHasName(group *ast.GenDecl, spec ast.Spec, name string) bool {
	switch candidate := spec.(type) {
	case *ast.ValueSpec:
		if group.Tok != token.CONST && group.Tok != token.VAR {
			return false
		}
		for _, identifier := range candidate.Names {
			if identifier.Name == name {
				return true
			}
		}
	case *ast.TypeSpec:
		return group.Tok == token.TYPE && candidate.Name.Name == name
	}
	return false
}

func validateDeclarationReplacement(match declarationReplacementMatch, replacement replacementDecl, symbol string) error {
	if match.group.Tok != replacement.kind {
		return fmt.Errorf("replacement declaration %q has kind %s; existing declaration uses %s", symbol, replacement.kind, match.group.Tok)
	}
	if value, ok := match.spec.(*ast.ValueSpec); ok && match.group.Tok == token.CONST {
		if len(value.Names) != 1 {
			return fmt.Errorf("cannot replace %q inside a shared const spec; split the spec first", symbol)
		}
		if constSpecInheritsExpression(match.group, value) {
			return fmt.Errorf("cannot replace %q: later const spec inherits its expression and would change", symbol)
		}
	}
	if value, ok := match.spec.(*ast.ValueSpec); ok && len(value.Names) != 1 {
		return fmt.Errorf("cannot replace %q inside a shared value spec; split the spec first", symbol)
	}
	return nil
}

func constSpecInheritsExpression(group *ast.GenDecl, target *ast.ValueSpec) bool {
	for _, later := range group.Specs {
		if later.Pos() <= target.Pos() {
			continue
		}
		if next, ok := later.(*ast.ValueSpec); ok && len(next.Values) == 0 {
			return true
		}
	}
	return false
}

func declarationReplacementSpan(target declarationReplacementTarget, match declarationReplacementMatch, replacement replacementDecl) declarationReplacementSource {
	start, end := match.spec.Pos(), match.spec.End()
	if replacement.hasLeadingComment && specDoc(match.spec) != nil {
		start = specDoc(match.spec).Pos()
	}
	if replacement.hasTrailingComment && specTrailingComment(match.spec) != nil {
		end = specTrailingComment(match.spec).End()
	}
	text := replacement.specText
	standalone := len(match.group.Specs) == 1 && !match.group.Lparen.IsValid()
	if standalone {
		if match.declaration.Pos() < start {
			start = match.declaration.Pos()
		}
		if replacement.hasLeadingComment && match.group.Doc != nil && match.group.Doc.Pos() < start {
			start = match.group.Doc.Pos()
		}
		if match.declaration.End() > end {
			end = match.declaration.End()
		}
		text = replacement.text
	}
	return declarationReplacementSource{start: target.fset.Position(start).Offset, end: target.fset.Position(end).Offset, text: text}
}

func specDoc(spec ast.Spec) *ast.CommentGroup {
	switch candidate := spec.(type) {
	case *ast.ValueSpec:
		return candidate.Doc
	case *ast.TypeSpec:
		return candidate.Doc
	default:
		return nil
	}
}

func specTrailingComment(spec ast.Spec) *ast.CommentGroup {
	switch candidate := spec.(type) {
	case *ast.ValueSpec:
		return candidate.Comment
	case *ast.TypeSpec:
		return candidate.Comment
	default:
		return nil
	}
}

func publishDeclarationReplacement(ctx context.Context, path string, original []byte, snippet string, span declarationReplacementSource, organizeImports bool) error {
	updated := replaceBytes(original, span.start, span.end, []byte(span.text))
	formatted, err := format.Source(updated)
	if err != nil {
		return &SyntaxError{File: path, Snippet: snippet, Cause: err, Err: ErrSyntax}
	}
	formatted, err = cleanupMoveImports(ctx, path, formatted)
	if err != nil {
		return fmt.Errorf("clean imports after declaration replacement: %w", err)
	}
	if err := pipeline.WriteAtomicContext(ctx, path, formatted); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}
	if organizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(path), path); err != nil {
			organizeErr := fmt.Errorf("organize imports: %w", err)
			if formatErr := pipeline.Format(ctx, filepath.Dir(path), path); formatErr != nil {
				return errors.Join(organizeErr, fmt.Errorf("format file after import organization failure: %w", formatErr))
			}
			return organizeErr
		}
	}
	return nil
}

type replacementDecl struct {
	kind               token.Token
	text, specText     string
	hasLeadingComment  bool
	hasTrailingComment bool
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
	declared := extractDeclNames(gen)
	if len(declared) != 1 || declared[0] != symbol {
		return replacementDecl{}, fmt.Errorf("replacement must declare only requested symbol %q", symbol)
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
		return replacementDecl{}, fmt.Errorf("format replacement node: %w", err)
	}
	formatted := formattedBuf.String()
	specText := extractSpecSource(string(formattedSource), formattedGen.Tok)
	result := replacementDecl{kind: formattedGen.Tok, text: formatted, specText: specText}
	result.hasLeadingComment = formattedGen.Doc != nil
	switch value := formattedGen.Specs[0].(type) {
	case *ast.ValueSpec:
		result.hasLeadingComment = result.hasLeadingComment || value.Doc != nil
		result.hasTrailingComment = value.Comment != nil
	case *ast.TypeSpec:
		result.hasLeadingComment = result.hasLeadingComment || value.Doc != nil
		result.hasTrailingComment = value.Comment != nil
	}
	return result, nil
}
