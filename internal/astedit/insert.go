// Package astedit provides AST-level code transformation routines including declaration insertion and boundary calculation.
package astedit

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

// Placement specifies where in the file or relative to declarations an insertion occurs.
type Placement string

// Supported declaration placement qualifiers.
const (
	PlacementFileStart    Placement = "file_start"
	PlacementFileEnd      Placement = "file_end"
	PlacementPublicStart  Placement = "public_start"
	PlacementPublicEnd    Placement = "public_end"
	PlacementPrivateStart Placement = "private_start"
	PlacementPrivateEnd   Placement = "private_end"
	PlacementBeforeSymbol Placement = "before_symbol"
	PlacementAfterSymbol  Placement = "after_symbol"
)

// Options configures declaration insertion behavior.
type Options struct {
	Placement           Placement
	TargetSymbol        string
	Visibility          string // "public", "private", or empty
	AutoOrganizeImports bool
}

// InsertDeclaration parses and injects one or more top-level declarations into filePath.
func InsertDeclaration(ctx context.Context, filePath string, source string, opts Options) error {
	if sourcebuf.FromContext(ctx) == nil {
		return runBufferedFile(ctx, filePath, func(editCtx context.Context) error {
			return InsertDeclaration(editCtx, filePath, source, opts)
		})
	}
	_, err := InsertConstruct(ctx, filePath, source, InsertConstructOptions{
		Kind:                backend.ConstructDeclaration,
		Placement:           opts.Placement,
		TargetSymbol:        opts.TargetSymbol,
		Visibility:          opts.Visibility,
		AutoOrganizeImports: opts.AutoOrganizeImports,
	})
	return err
}

// InsertDeclaration parses and injects one or more top-level declarations into filePath.
func insertDeclarationImpl(ctx context.Context, filePath string, source string, opts Options) error {
	source = normalizeDeclarationSnippet(source)
	opts.TargetSymbol = strings.Trim(strings.TrimSpace(opts.TargetSymbol), `"'`)
	target, err := readDeclarationTarget(ctx, filePath)
	if err != nil {
		return err
	}
	if _, syntaxErr := verifySnippetSyntax(source, opts.Visibility); syntaxErr != nil {
		return fmt.Errorf("validate snippet: %w", syntaxErr)
	}
	insertOffset, err := calculateInsertionOffset(target.set, target.file, target.content, opts)
	if err != nil {
		return fmt.Errorf("calculate insertion offset: %w", annotatePlacementError(err, target.path, target.set, target.file))
	}
	insertOffset = normalizeInsertionOffset(target.set, target.file, target.content, insertOffset)
	updated := insertDeclarationText(target.content, source, insertOffset)
	return publishInsertedDeclaration(ctx, target.path, updated, opts.AutoOrganizeImports)
}

func verifySnippetSyntax(source string, expectedVisibility string) ([]ast.Decl, error) {
	snippet, err := parseDeclarationSnippet(source)
	if err != nil {
		return nil, err
	}
	if len(snippet.file.Decls) == 0 {
		return nil, ErrNoDeclarations
	}
	if err := validateSnippetVisibility(snippet, expectedVisibility); err != nil {
		return nil, err
	}
	return snippet.file.Decls, nil
}

type parsedDeclarationSnippet struct {
	fset      *token.FileSet
	file      *ast.File
	prepended bool
}

func parseDeclarationSnippet(source string) (parsedDeclarationSnippet, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return parsedDeclarationSnippet{}, ErrEmptySnippet
	}

	toParse := trimmed
	prepended := false
	if !strings.HasPrefix(trimmed, "package ") {
		toParse = "package dummy\n\n" + trimmed
		prepended = true
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "snippet", toParse, parser.ParseComments)
	if err != nil {
		return parsedDeclarationSnippet{}, declarationSnippetSyntaxError(source, fset, prepended, err)
	}
	return parsedDeclarationSnippet{fset: fset, file: node, prepended: prepended}, nil
}

func declarationSnippetSyntaxError(source string, fset *token.FileSet, prepended bool, cause error) error {
	position := extractSyntaxPosition(fset, cause)
	position = snippetSourcePosition(position, prepended)
	return &SyntaxError{Snippet: source, Pos: position, Cause: cause, Err: ErrSyntax}
}

func snippetSourcePosition(position token.Position, prepended bool) token.Position {
	if prepended && position.Line > 2 {
		position.Line -= 2
		position.Offset -= len("package dummy\n\n")
		if position.Offset < 0 {
			position.Offset = 0
		}
	}
	position.Filename = "snippet"
	return position
}

func validateSnippetVisibility(snippet parsedDeclarationSnippet, expected string) error {
	if expected != "public" && expected != "private" {
		return nil
	}
	for _, declaration := range snippet.file.Decls {
		for _, name := range extractDeclNames(declaration) {
			mismatch := snippetVisibilityMismatch(snippet, declaration, name, expected)
			if mismatch != nil {
				return mismatch
			}
		}
	}
	return nil
}

func snippetVisibilityMismatch(snippet parsedDeclarationSnippet, declaration ast.Decl, name, expected string) error {
	exported := ast.IsExported(name)
	if (expected == "public" && exported) || (expected == "private" && !exported) {
		return nil
	}
	requested, effective := AccessModifierPublic, AccessModifierPrivate
	if expected == "private" {
		requested, effective = AccessModifierPrivate, AccessModifierPublic
	}
	position := snippetSourcePosition(snippet.fset.Position(declaration.Pos()), snippet.prepended)
	return &VisibilityMismatchError{
		File: "snippet", Pos: position, Identifier: name,
		Requested: requested, Effective: effective, Err: ErrVisibilityMismatch,
	}
}

func extractDeclNames(decl ast.Decl) []string {
	var names []string
	switch d := decl.(type) {
	case *ast.FuncDecl:
		names = append(names, d.Name.Name)
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				names = append(names, s.Name.Name)
			case *ast.ValueSpec:
				for _, name := range s.Names {
					names = append(names, name.Name)
				}
			}
		}
	}
	return names
}

func isDeclExported(decl ast.Decl) bool {
	names := extractDeclNames(decl)
	return slices.ContainsFunc(names, ast.IsExported)
}

func calculateInsertionOffset(fset *token.FileSet, fileNode *ast.File, content []byte, opts Options) (int, error) {
	switch opts.Placement {
	case PlacementFileStart:
		return offsetAfterImports(fset, fileNode), nil

	case PlacementFileEnd, "":
		return len(content), nil

	case PlacementBeforeSymbol, PlacementAfterSymbol:
		return symbolInsertionOffset(fset, fileNode, opts)
	case PlacementPublicStart, PlacementPublicEnd, PlacementPrivateStart, PlacementPrivateEnd:
		return groupedDeclarationOffset(fset, fileNode, content, opts.Placement)
	}

	return len(content), nil
}

func symbolInsertionOffset(fset *token.FileSet, file *ast.File, opts Options) (int, error) {
	if opts.TargetSymbol == "" {
		return 0, &PlacementError{Strategy: opts.Placement, Err: ErrSymbolNotFound}
	}
	declaration, err := findTargetDecl(file, opts.TargetSymbol)
	if err != nil {
		if placementErr, ok := errors.AsType[*PlacementError](err); ok {
			placementErr.Strategy = opts.Placement
		}
		return 0, err
	}
	if opts.Placement == PlacementBeforeSymbol {
		return fset.Position(declaration.Pos()).Offset, nil
	}
	return fset.Position(declaration.End()).Offset, nil
}

func groupedDeclarationOffset(fset *token.FileSet, file *ast.File, content []byte, placement Placement) (int, error) {
	sections := partitionDeclarationSections(file.Decls)
	switch placement {
	case PlacementPublicStart:
		return publicSectionStart(fset, file, sections), nil
	case PlacementPublicEnd:
		return publicSectionEnd(fset, file, sections), nil
	case PlacementPrivateStart:
		if len(sections.private) > 0 {
			return fset.Position(sections.private[0].Pos()).Offset, nil
		}
	case PlacementPrivateEnd:
		if len(sections.private) > 0 {
			return fset.Position(sections.private[len(sections.private)-1].End()).Offset, nil
		}
	default:
		return 0, fmt.Errorf("invalid grouped placement %q", placement)
	}
	return len(content), nil
}

type declarationSections struct {
	public  []ast.Decl
	private []ast.Decl
}

func partitionDeclarationSections(declarations []ast.Decl) declarationSections {
	sections := declarationSections{}
	for _, declaration := range declarations {
		if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.IMPORT {
			continue
		}
		if isDeclExported(declaration) {
			sections.public = append(sections.public, declaration)
		} else {
			sections.private = append(sections.private, declaration)
		}
	}
	return sections
}

func publicSectionStart(fset *token.FileSet, file *ast.File, sections declarationSections) int {
	if len(sections.public) > 0 {
		return fset.Position(sections.public[0].Pos()).Offset
	}
	if len(sections.private) > 0 {
		return fset.Position(sections.private[0].Pos()).Offset
	}
	return offsetAfterImports(fset, file)
}

func publicSectionEnd(fset *token.FileSet, file *ast.File, sections declarationSections) int {
	if len(sections.public) > 0 {
		return fset.Position(sections.public[len(sections.public)-1].End()).Offset
	}
	if len(sections.private) > 0 {
		return fset.Position(sections.private[0].Pos()).Offset
	}
	return offsetAfterImports(fset, file)
}

func offsetAfterImports(fset *token.FileSet, fileNode *ast.File) int {
	var lastImportEnd token.Pos
	for _, decl := range fileNode.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			if gen.End() > lastImportEnd {
				lastImportEnd = gen.End()
			}
		}
	}

	if lastImportEnd.IsValid() {
		return fset.Position(lastImportEnd).Offset
	}

	if fileNode.Name != nil {
		return fset.Position(fileNode.Name.End()).Offset
	}

	return 0
}

func findTargetDecl(fileNode *ast.File, targetSymbol string) (ast.Decl, error) {
	targetReceiver, targetName, err := symbol.ParseIdentifier(targetSymbol)
	if err != nil {
		return nil, &PlacementError{
			TargetSymbol: targetSymbol,
			Err:          err,
		}
	}

	for _, declaration := range fileNode.Decls {
		if declarationMatchesTarget(declaration, targetReceiver, targetName) {
			return declaration, nil
		}
	}

	return nil, &PlacementError{
		TargetSymbol: targetSymbol,
		Err:          ErrSymbolNotFound,
	}
}

func declarationMatchesTarget(declaration ast.Decl, targetReceiver, targetName string) bool {
	switch candidate := declaration.(type) {
	case *ast.FuncDecl:
		return candidate.Name.Name == targetName &&
			(targetReceiver == "" || functionReceiverName(candidate) == targetReceiver)
	case *ast.GenDecl:
		return genericDeclarationHasName(candidate, targetName)
	default:
		return false
	}
}

func functionReceiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	switch receiver := function.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return receiver.Name
	case *ast.StarExpr:
		if name, ok := receiver.X.(*ast.Ident); ok {
			return name.Name
		}
	}
	return ""
}

func genericDeclarationHasName(declaration *ast.GenDecl, targetName string) bool {
	for _, spec := range declaration.Specs {
		if specificationHasName(spec, targetName) {
			return true
		}
	}
	return false
}

func specificationHasName(spec ast.Spec, targetName string) bool {
	switch candidate := spec.(type) {
	case *ast.TypeSpec:
		return candidate.Name.Name == targetName
	case *ast.ValueSpec:
		for _, name := range candidate.Names {
			if name.Name == targetName {
				return true
			}
		}
	}
	return false
}
