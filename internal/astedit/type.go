// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// TypeOptions configures type declaration insertion behavior.
type TypeOptions struct {
	AccessModifier      AccessModifier
	Placement           Placement
	TargetSymbol        string
	AutoOrganizeImports bool
}

// InsertType injects a struct, interface, or type alias declaration into filePath.
func InsertType(ctx context.Context, filePath string, source string, opts TypeOptions) error {
	return insertTypeImpl(ctx, filePath, source, opts)
}

// InsertType injects a struct, interface, or type alias declaration into filePath.
func insertTypeImpl(ctx context.Context, filePath string, source string, opts TypeOptions) error {
	source = normalizeDeclarationSnippet(source)
	opts.TargetSymbol = strings.Trim(strings.TrimSpace(opts.TargetSymbol), `"'`)
	target, err := readDeclarationTarget(ctx, filePath)
	if err != nil {
		return err
	}

	genDecl, typeName, err := verifyTypeSnippet(source)
	if err != nil {
		return fmt.Errorf("validate type snippet: %w", err)
	}

	if accessErr := ValidateAccess(DefaultBackend, opts.AccessModifier, typeName); accessErr != nil {
		if visErr, ok := errors.AsType[*VisibilityMismatchError](accessErr); ok {
			visErr.File = "snippet"
		}
		return fmt.Errorf("validate access modifier: %w", accessErr)
	}

	effectiveAccess, err := resolveDeclarationAccess(opts.AccessModifier, typeName)
	if err != nil {
		return err
	}
	if collisionErr := checkConstructDeclarationCollision(ctx, target.path, target.file, genDecl); collisionErr != nil {
		return fmt.Errorf("check declaration collision: %w", collisionErr)
	}

	insertOffset, err := calculateTypeOffset(target.set, target.file, target.content, genDecl, typeName, effectiveAccess, opts)
	if err != nil {
		return fmt.Errorf("calculate type offset: %w", annotatePlacementError(err, target.path, target.set, target.file))
	}

	insertOffset = normalizeInsertionOffset(target.set, target.file, target.content, insertOffset)
	updated := insertDeclarationText(target.content, source, insertOffset)
	return publishInsertedDeclaration(ctx, target.path, updated, opts.AutoOrganizeImports)
}

func verifyTypeSnippet(source string) (*ast.GenDecl, string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil, "", ErrEmptySnippet
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
		pos := extractSyntaxPosition(fset, err)
		if prepended && pos.Line > 2 {
			pos.Line -= 2
			pos.Offset -= len("package dummy\n\n")
			if pos.Offset < 0 {
				pos.Offset = 0
			}
		}
		pos.Filename = "snippet"
		return nil, "", &SyntaxError{
			Snippet: source,
			Pos:     pos,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	if len(node.Decls) == 0 {
		return nil, "", ErrNoDeclarations
	}

	genDecl, ok := node.Decls[0].(*ast.GenDecl)
	if !ok || genDecl.Tok != token.TYPE {
		return nil, "", fmt.Errorf("%w: expected type declaration (*ast.GenDecl with token.TYPE), found %T", ErrUnexpectedDeclType, node.Decls[0])
	}

	if len(genDecl.Specs) == 0 {
		return nil, "", ErrNoTypeSpecs
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return nil, "", fmt.Errorf("%w: expected *ast.TypeSpec, found %T", ErrUnexpectedDeclType, genDecl.Specs[0])
	}

	return genDecl, typeSpec.Name.Name, nil
}

func calculateTypeOffset(fset *token.FileSet, fileNode *ast.File, content []byte, genDecl *ast.GenDecl, typeName string, effectiveAccess AccessModifier, opts TypeOptions) (int, error) {
	_ = genDecl
	if opts.Placement != "" {
		if effectiveAccess == AccessModifierPublic && (opts.Placement == PlacementPrivateStart || opts.Placement == PlacementPrivateEnd) {
			return 0, &PlacementError{
				Strategy:     opts.Placement,
				TargetSymbol: typeName,
				Err:          ErrSectionViolation,
			}
		}
		if effectiveAccess == AccessModifierPrivate && (opts.Placement == PlacementPublicStart || opts.Placement == PlacementPublicEnd) {
			return 0, &PlacementError{
				Strategy:     opts.Placement,
				TargetSymbol: typeName,
				Err:          ErrSectionViolation,
			}
		}

		insertOpts := Options{
			Placement:    opts.Placement,
			TargetSymbol: opts.TargetSymbol,
		}
		return calculateInsertionOffset(fset, fileNode, content, insertOpts)
	}

	if effectiveAccess == AccessModifierPublic {
		// Place at start of public code section
		return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPublicStart})
	}
	// Place at start of private code section
	return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPrivateStart})
}
