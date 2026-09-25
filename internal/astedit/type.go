// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
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
	_, err := InsertStructure(ctx, filePath, source, StructureOptions{
		Kind:                StructureKindType,
		AccessModifier:      opts.AccessModifier,
		Placement:           opts.Placement,
		TargetSymbol:        opts.TargetSymbol,
		AutoOrganizeImports: opts.AutoOrganizeImports,
	})
	return err
}

// InsertType injects a struct, interface, or type alias declaration into filePath.
func insertTypeImpl(ctx context.Context, filePath string, source string, opts TypeOptions) error {
	source = strings.TrimSpace(source)
	if (strings.HasPrefix(source, "\"") && strings.HasSuffix(source, "\"")) ||
		(strings.HasPrefix(source, "'") && strings.HasSuffix(source, "'")) {
		source = source[1 : len(source)-1]
	}
	opts.TargetSymbol = strings.Trim(strings.TrimSpace(opts.TargetSymbol), `"'`)

	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}

	genDecl, typeName, err := verifyTypeSnippet(source)
	if err != nil {
		return fmt.Errorf("validate type snippet: %w", err)
	}

	if err := ValidateAccess(DefaultBackend, opts.AccessModifier, typeName); err != nil {
		if visErr, ok := errors.AsType[*VisibilityMismatchError](err); ok {
			visErr.File = "snippet"
		}
		return fmt.Errorf("validate access modifier: %w", err)
	}

	effectiveAccess, err := ResolveAccess(DefaultBackend, opts.AccessModifier, typeName)
	if err != nil {
		return fmt.Errorf("resolve effective access: %w", err)
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil && fileNode == nil {
		return appendToEOF(ctx, cleanPath, content, source, opts.AutoOrganizeImports)
	}

	insertOffset, err := calculateTypeOffset(fset, fileNode, content, genDecl, typeName, effectiveAccess, opts)
	if err != nil {
		if pErr, ok := errors.AsType[*PlacementError](err); ok {
			if pErr.File == "" {
				pErr.File = cleanPath
			}
			if !pErr.Pos.IsValid() && fileNode != nil && fileNode.Package.IsValid() {
				pErr.Pos = fset.Position(fileNode.Package)
			}
		}
		return fmt.Errorf("calculate type offset: %w", err)
	}

	insertOffset = normalizeInsertionOffset(fset, fileNode, content, insertOffset)

	var newContent bytes.Buffer
	newContent.Write(content[:insertOffset])

	snippetText := strings.TrimSpace(source)
	if strings.HasPrefix(snippetText, "package ") {
		lines := strings.SplitN(snippetText, "\n", 2)
		if len(lines) > 1 {
			snippetText = strings.TrimSpace(lines[1])
		}
	}

	if insertOffset > 0 {
		prev := content[:insertOffset]
		if !bytes.HasSuffix(prev, []byte("\n\n")) {
			if bytes.HasSuffix(prev, []byte("\n")) {
				newContent.WriteString("\n")
			} else {
				newContent.WriteString("\n\n")
			}
		}
	}

	newContent.WriteString(snippetText)
	newContent.WriteString("\n\n")

	rest := content[insertOffset:]
	restTrimmed := bytes.TrimLeft(rest, "\r\n")
	newContent.Write(restTrimmed)

	if err := pipeline.WriteAtomic(cleanPath, newContent.Bytes()); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}

	dir := filepath.Dir(cleanPath)
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, dir, cleanPath); err != nil {
			_ = pipeline.Format(ctx, dir, cleanPath)
			return fmt.Errorf("organize imports: %w", err)
		}
	} else {
		if err := pipeline.Format(ctx, dir, cleanPath); err != nil {
			return fmt.Errorf("format file: %w", err)
		}
	}

	return nil
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
