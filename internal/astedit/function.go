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

// FunctionOptions configures function and method insertion behavior.
type FunctionOptions struct {
	AccessModifier      AccessModifier
	Placement           Placement
	TargetSymbol        string
	AutoOrganizeImports bool
}

// InsertFunction injects a function or method declaration into filePath.
func InsertFunction(ctx context.Context, filePath string, source string, opts FunctionOptions) error {
	_, err := InsertStructure(ctx, filePath, source, StructureOptions{
		Kind:                StructureKindFunction,
		AccessModifier:      opts.AccessModifier,
		Placement:           opts.Placement,
		TargetSymbol:        opts.TargetSymbol,
		AutoOrganizeImports: opts.AutoOrganizeImports,
	})
	return err
}

// InsertFunction injects a function or method declaration into filePath.
func insertFunctionImpl(ctx context.Context, filePath string, source string, opts FunctionOptions) error {
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

	fnDecl, err := parseFunctionSnippet(source)
	if err != nil {
		return fmt.Errorf("validate function snippet: %w", err)
	}

	fnName := fnDecl.Name.Name
	if err := ValidateAccess(DefaultBackend, opts.AccessModifier, fnName); err != nil {
		if visErr, ok := errors.AsType[*VisibilityMismatchError](err); ok {
			visErr.File = "snippet"
		}
		return fmt.Errorf("validate access modifier: %w", err)
	}

	effectiveAccess, err := ResolveAccess(DefaultBackend, opts.AccessModifier, fnName)
	if err != nil {
		return fmt.Errorf("resolve effective access: %w", err)
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil && fileNode == nil {
		return appendToEOF(ctx, cleanPath, content, source, opts.AutoOrganizeImports)
	}

	insertOffset, err := calculateFunctionOffset(fset, fileNode, content, fnDecl, effectiveAccess, opts)
	if err != nil {
		if pErr, ok := errors.AsType[*PlacementError](err); ok {
			if pErr.File == "" {
				pErr.File = cleanPath
			}
			if !pErr.Pos.IsValid() && fileNode != nil && fileNode.Package.IsValid() {
				pErr.Pos = fset.Position(fileNode.Package)
			}
		}
		return fmt.Errorf("calculate function offset: %w", err)
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

func parseFunctionSnippet(source string) (*ast.FuncDecl, error) {
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

	if len(node.Decls) > 1 {
		return nil, fmt.Errorf("%w: expected single function declaration, found %d declarations", ErrMultipleDeclarations, len(node.Decls))
	}

	fnDecl, ok := node.Decls[0].(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("%w: expected function declaration (*ast.FuncDecl), found %T", ErrUnexpectedDeclType, node.Decls[0])
	}

	return fnDecl, nil
}

func extractReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	if idx, ok := t.(*ast.IndexExpr); ok {
		if ident, ok := idx.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func calculateFunctionOffset(fset *token.FileSet, fileNode *ast.File, content []byte, fnDecl *ast.FuncDecl, effectiveAccess AccessModifier, opts FunctionOptions) (int, error) {
	if opts.Placement != "" {
		if effectiveAccess == AccessModifierPublic && (opts.Placement == PlacementPrivateStart || opts.Placement == PlacementPrivateEnd) {
			return 0, &PlacementError{
				Strategy:     opts.Placement,
				TargetSymbol: fnDecl.Name.Name,
				Err:          ErrSectionViolation,
			}
		}
		if effectiveAccess == AccessModifierPrivate && (opts.Placement == PlacementPublicStart || opts.Placement == PlacementPublicEnd) {
			return 0, &PlacementError{
				Strategy:     opts.Placement,
				TargetSymbol: fnDecl.Name.Name,
				Err:          ErrSectionViolation,
			}
		}

		insertOpts := Options{
			Placement:    opts.Placement,
			TargetSymbol: opts.TargetSymbol,
		}
		return calculateInsertionOffset(fset, fileNode, content, insertOpts)
	}

	recvType := extractReceiverTypeName(fnDecl.Recv)
	if recvType != "" {
		var publicReceiverMethods []*ast.FuncDecl
		var privateReceiverMethods []*ast.FuncDecl
		var targetTypeDecl ast.Decl

		for _, decl := range fileNode.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if extractReceiverTypeName(fn.Recv) == recvType {
					if isDeclExported(fn) {
						publicReceiverMethods = append(publicReceiverMethods, fn)
					} else {
						privateReceiverMethods = append(privateReceiverMethods, fn)
					}
				}
			} else if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
				for _, spec := range gen.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == recvType {
						targetTypeDecl = decl
						break
					}
				}
			}
		}

		if effectiveAccess == AccessModifierPublic {
			if len(publicReceiverMethods) > 0 {
				return fset.Position(publicReceiverMethods[len(publicReceiverMethods)-1].End()).Offset, nil
			}
			if targetTypeDecl != nil && isDeclExported(targetTypeDecl) {
				return fset.Position(targetTypeDecl.End()).Offset, nil
			}
			// Fall back to public section placement
			return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPublicEnd})
		}

		// Private method placement
		if len(privateReceiverMethods) > 0 {
			return fset.Position(privateReceiverMethods[len(privateReceiverMethods)-1].End()).Offset, nil
		}
		// Fall back to private section start
		return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPrivateStart})
	}

	// Regular function placement
	if effectiveAccess == AccessModifierPublic {
		return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPublicEnd})
	}
	return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPrivateEnd})
}
