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

// DeclOptions configures general top-level declaration insertion behavior.
type DeclOptions struct {
	AccessModifier      AccessModifier
	Group               string // "append" (default for const/var) or "standalone"
	Placement           Placement
	TargetSymbol        string
	Overwrite           bool
	AutoOrganizeImports bool
}

// InsertDecl injects constants, variables, or general top-level declarations into filePath.
func InsertDecl(ctx context.Context, filePath string, source string, opts DeclOptions) error {
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

	snippetDecls, err := verifySnippetSyntax(source, "")
	if err != nil {
		return fmt.Errorf("validate declaration snippet: %w", err)
	}

	for _, decl := range snippetDecls {
		for _, name := range extractDeclNames(decl) {
			if err := ValidateAccess(DefaultBackend, opts.AccessModifier, name); err != nil {
				if visErr, ok := errors.AsType[*VisibilityMismatchError](err); ok {
					visErr.File = "snippet"
				}
				return fmt.Errorf("validate access modifier for %q: %w", name, err)
			}
		}
	}

	effectiveAccess := AccessModifierPublic
	if len(snippetDecls) > 0 {
		names := extractDeclNames(snippetDecls[0])
		if len(names) > 0 {
			eff, err := ResolveAccess(DefaultBackend, opts.AccessModifier, names[0])
			if err == nil {
				effectiveAccess = eff
			}
		}
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return fmt.Errorf("cannot check package declaration collisions in unparseable target file: %w", err)
	}
	handled, err := handleInsertDeclCollisions(ctx, cleanPath, source, snippetDecls, fileNode, opts)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// 1. Attempt group appending for const/var if opts.Group is "append" (or default empty)
	if (opts.Group == "" || opts.Group == "append") && opts.Placement == "" && len(snippetDecls) == 1 {
		if gen, ok := snippetDecls[0].(*ast.GenDecl); ok && (gen.Tok == token.CONST || gen.Tok == token.VAR) {
			targetGroup := findMatchingGroup(fileNode, gen.Tok, effectiveAccess)
			if targetGroup != nil && targetGroup.Rparen.IsValid() {
				specSource := extractSpecSource(source, gen.Tok)
				insertOffset := fset.Position(targetGroup.Rparen).Offset
				if gen.Tok == token.VAR {
					if sentinelOffset, ok := sentinelVarInsertionOffset(fset, targetGroup, gen); ok {
						insertOffset = sentinelOffset
					}
				}

				var buf bytes.Buffer
				buf.Write(content[:insertOffset])
				groupEnd := fset.Position(targetGroup.Rparen).Offset
				switch {
				case insertOffset != groupEnd:
					buf.WriteString(specSource)
					buf.WriteString("\n\t")
				case !bytes.HasSuffix(content[:insertOffset], []byte("\n")):
					buf.WriteString("\n")
					buf.WriteString("\t")
				default:
					buf.WriteString("\t")
				}
				if insertOffset == groupEnd {
					buf.WriteString(specSource)
					buf.WriteString("\n")
				}
				buf.Write(content[insertOffset:])

				return writeAndOrganize(ctx, cleanPath, buf.Bytes(), opts.AutoOrganizeImports)
			}
		}
	}

	// 2. Standalone insertion
	insertOffset, err := calculateDeclOffset(fset, fileNode, content, effectiveAccess, opts)
	if err != nil {
		if pErr, ok := errors.AsType[*PlacementError](err); ok {
			if pErr.File == "" {
				pErr.File = cleanPath
			}
			if !pErr.Pos.IsValid() && fileNode != nil && fileNode.Package.IsValid() {
				pErr.Pos = fset.Position(fileNode.Package)
			}
		}
		return fmt.Errorf("calculate declaration offset: %w", err)
	}

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

	return writeAndOrganize(ctx, cleanPath, newContent.Bytes(), opts.AutoOrganizeImports)
}

func sentinelVarInsertionOffset(fset *token.FileSet, group *ast.GenDecl, incoming *ast.GenDecl) (int, bool) {
	if len(incoming.Specs) != 1 {
		return 0, false
	}
	incomingSpec, ok := incoming.Specs[0].(*ast.ValueSpec)
	if !ok || len(incomingSpec.Names) != 1 || !strings.HasPrefix(incomingSpec.Names[0].Name, "Err") {
		return 0, false
	}
	incomingName := incomingSpec.Names[0].Name
	for _, spec := range group.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || len(valueSpec.Names) != 1 || !strings.HasPrefix(valueSpec.Names[0].Name, "Err") {
			continue
		}
		if valueSpec.Names[0].Name > incomingName {
			return fset.Position(valueSpec.Pos()).Offset, true
		}
	}
	return 0, false
}

func findMatchingGroup(fileNode *ast.File, tok token.Token, targetAccess AccessModifier) *ast.GenDecl {
	var candidate *ast.GenDecl
	for _, decl := range fileNode.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != tok || !gen.Lparen.IsValid() {
			continue
		}
		isExp := isDeclExported(gen)
		if (targetAccess == AccessModifierPublic && isExp) || (targetAccess == AccessModifierPrivate && !isExp) {
			return gen
		}
		if candidate == nil {
			candidate = gen
		}
	}
	return candidate
}

func extractSpecSource(raw string, tok token.Token) string {
	trimmed := strings.TrimSpace(raw)
	tokPrefix := tok.String() + " "
	if strings.HasPrefix(trimmed, tokPrefix) {
		trimmed = strings.TrimSpace(trimmed[len(tokPrefix):])
	}
	if strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	return trimmed
}

func calculateDeclOffset(fset *token.FileSet, fileNode *ast.File, content []byte, effectiveAccess AccessModifier, opts DeclOptions) (int, error) {
	if opts.Placement != "" {
		if effectiveAccess == AccessModifierPublic && (opts.Placement == PlacementPrivateStart || opts.Placement == PlacementPrivateEnd) {
			return 0, &PlacementError{
				Strategy: opts.Placement,
				Err:      ErrSectionViolation,
			}
		}
		if effectiveAccess == AccessModifierPrivate && (opts.Placement == PlacementPublicStart || opts.Placement == PlacementPublicEnd) {
			return 0, &PlacementError{
				Strategy: opts.Placement,
				Err:      ErrSectionViolation,
			}
		}

		insertOpts := Options{
			Placement:    opts.Placement,
			TargetSymbol: opts.TargetSymbol,
		}
		return calculateInsertionOffset(fset, fileNode, content, insertOpts)
	}

	if effectiveAccess == AccessModifierPublic {
		return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPublicStart})
	}
	return calculateInsertionOffset(fset, fileNode, content, Options{Placement: PlacementPrivateStart})
}

func writeAndOrganize(ctx context.Context, cleanPath string, data []byte, autoOrganize bool) error {
	if err := pipeline.WriteAtomic(cleanPath, data); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}

	dir := filepath.Dir(cleanPath)
	if autoOrganize {
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
