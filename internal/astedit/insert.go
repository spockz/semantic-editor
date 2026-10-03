// Package astedit provides AST-level code transformation routines including declaration insertion and boundary calculation.
package astedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/pipeline"
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
	source = strings.TrimSpace(source)
	if (strings.HasPrefix(source, "\"") && strings.HasSuffix(source, "\"")) ||
		(strings.HasPrefix(source, "'") && strings.HasSuffix(source, "'")) {
		source = source[1 : len(source)-1]
	}
	opts.TargetSymbol = strings.Trim(strings.TrimSpace(opts.TargetSymbol), `"'`)

	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}

	snippetDecls, err := verifySnippetSyntax(source, opts.Visibility)
	if err != nil {
		return fmt.Errorf("validate snippet: %w", err)
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse target file before insertion: %w", err)
	}

	insertOffset, err := calculateInsertionOffset(fset, fileNode, content, opts)
	if err != nil {
		if pErr, ok := errors.AsType[*PlacementError](err); ok {
			if pErr.File == "" {
				pErr.File = cleanPath
			}
			if !pErr.Pos.IsValid() && fileNode != nil && fileNode.Package.IsValid() {
				pErr.Pos = fset.Position(fileNode.Package)
			}
		}
		return fmt.Errorf("calculate insertion offset: %w", err)
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

	if err := pipeline.WriteAtomicContext(ctx, cleanPath, newContent.Bytes()); err != nil {
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

	_ = snippetDecls
	return nil
}

func verifySnippetSyntax(source string, expectedVisibility string) ([]ast.Decl, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil, ErrEmptySnippet
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
		return nil, &SyntaxError{
			Snippet: source,
			Pos:     pos,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	if len(node.Decls) == 0 {
		return nil, ErrNoDeclarations
	}

	if expectedVisibility != "" {
		for _, decl := range node.Decls {
			names := extractDeclNames(decl)
			for _, name := range names {
				exported := ast.IsExported(name)
				pos := fset.Position(decl.Pos())
				if prepended && pos.Line > 2 {
					pos.Line -= 2
					pos.Offset -= len("package dummy\n\n")
					if pos.Offset < 0 {
						pos.Offset = 0
					}
				}
				pos.Filename = "snippet"
				if expectedVisibility == "public" && !exported {
					return nil, &VisibilityMismatchError{
						File:       "snippet",
						Pos:        pos,
						Identifier: name,
						Requested:  AccessModifierPublic,
						Effective:  AccessModifierPrivate,
						Err:        ErrVisibilityMismatch,
					}
				}
				if expectedVisibility == "private" && exported {
					return nil, &VisibilityMismatchError{
						File:       "snippet",
						Pos:        pos,
						Identifier: name,
						Requested:  AccessModifierPrivate,
						Effective:  AccessModifierPublic,
						Err:        ErrVisibilityMismatch,
					}
				}
			}
		}
	}

	return node.Decls, nil
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
		if opts.TargetSymbol == "" {
			return 0, &PlacementError{
				Strategy: opts.Placement,
				Err:      ErrSymbolNotFound,
			}
		}
		targetDecl, err := findTargetDecl(fileNode, opts.TargetSymbol)
		if err != nil {
			if pErr, ok := errors.AsType[*PlacementError](err); ok {
				pErr.Strategy = opts.Placement
			}
			return 0, err
		}
		if opts.Placement == PlacementBeforeSymbol {
			return fset.Position(targetDecl.Pos()).Offset, nil
		}
		return fset.Position(targetDecl.End()).Offset, nil

	case PlacementPublicStart, PlacementPublicEnd, PlacementPrivateStart, PlacementPrivateEnd:
		var codeDecls []ast.Decl
		for _, decl := range fileNode.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
				continue
			}
			codeDecls = append(codeDecls, decl)
		}

		var publicDecls []ast.Decl
		var privateDecls []ast.Decl
		for _, decl := range codeDecls {
			if isDeclExported(decl) {
				publicDecls = append(publicDecls, decl)
			} else {
				privateDecls = append(privateDecls, decl)
			}
		}

		switch opts.Placement {
		case PlacementPublicStart:
			if len(publicDecls) > 0 {
				return fset.Position(publicDecls[0].Pos()).Offset, nil
			}
			if len(privateDecls) > 0 {
				// Public precedes private invariant: land before first private
				return fset.Position(privateDecls[0].Pos()).Offset, nil
			}
			return offsetAfterImports(fset, fileNode), nil

		case PlacementPublicEnd:
			if len(publicDecls) > 0 {
				return fset.Position(publicDecls[len(publicDecls)-1].End()).Offset, nil
			}
			if len(privateDecls) > 0 {
				// Public precedes private invariant: land before first private (identical to PublicStart)
				return fset.Position(privateDecls[0].Pos()).Offset, nil
			}
			return offsetAfterImports(fset, fileNode), nil

		case PlacementPrivateStart:
			if len(privateDecls) > 0 {
				return fset.Position(privateDecls[0].Pos()).Offset, nil
			}
			return len(content), nil

		case PlacementPrivateEnd:
			if len(privateDecls) > 0 {
				return fset.Position(privateDecls[len(privateDecls)-1].End()).Offset, nil
			}
			return len(content), nil
		default:
			return 0, fmt.Errorf("invalid grouped placement %q", opts.Placement)
		}
	}

	return len(content), nil
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
