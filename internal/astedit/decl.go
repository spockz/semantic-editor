// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
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

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
)

// DeclOptions configures general top-level declaration insertion behavior.
type DeclOptions struct {
	AccessModifier      AccessModifier
	Group               string // "append" (default for const/var) or "standalone"
	Placement           Placement
	TargetSymbol        string
	AutoOrganizeImports bool
}

// InsertDecl injects constants, variables, or general top-level declarations into filePath.
func InsertDecl(ctx context.Context, filePath string, source string, opts DeclOptions) error {
	return insertDeclImpl(ctx, filePath, source, opts)
}

// InsertDecl injects constants, variables, or general top-level declarations into filePath.
func insertDeclImpl(ctx context.Context, filePath string, source string, opts DeclOptions) error {
	source = normalizeDeclSource(source)
	opts.TargetSymbol = strings.Trim(strings.TrimSpace(opts.TargetSymbol), `"'`)
	target, err := prepareDeclInsertion(ctx, filePath, source, opts)
	if err != nil {
		return err
	}
	handled, err := insertDeclIntoGroup(ctx, target, source, opts)
	if err != nil || handled {
		return err
	}
	return insertDeclAtOffset(ctx, target, source, opts)
}

type declInsertionTarget struct {
	cleanPath       string
	content         []byte
	fset            *token.FileSet
	file            *ast.File
	snippetDecls    []ast.Decl
	effectiveAccess AccessModifier
}

func normalizeDeclSource(source string) string {
	source = strings.TrimSpace(source)
	if (strings.HasPrefix(source, "\"") && strings.HasSuffix(source, "\"")) ||
		(strings.HasPrefix(source, "'") && strings.HasSuffix(source, "'")) {
		return source[1 : len(source)-1]
	}
	return source
}

func prepareDeclInsertion(ctx context.Context, filePath, source string, opts DeclOptions) (*declInsertionTarget, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read target file: %w", err)
	}
	snippetDecls, err := verifySnippetSyntax(source, "")
	if err != nil {
		return nil, fmt.Errorf("validate declaration snippet: %w", err)
	}
	accessErr := validateDeclAccess(opts.AccessModifier, snippetDecls)
	if accessErr != nil {
		return nil, accessErr
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("cannot check package declaration collisions in unparseable target file: %w", err)
	}
	if err := checkInsertDeclCollisions(ctx, cleanPath, snippetDecls, file); err != nil {
		return nil, err
	}
	return &declInsertionTarget{
		cleanPath:       cleanPath,
		content:         content,
		fset:            fset,
		file:            file,
		snippetDecls:    snippetDecls,
		effectiveAccess: declEffectiveAccess(snippetDecls, opts.AccessModifier),
	}, nil
}

func validateDeclAccess(access AccessModifier, decls []ast.Decl) error {
	for _, decl := range decls {
		for _, name := range extractDeclNames(decl) {
			if err := ValidateAccess(DefaultBackend, access, name); err != nil {
				if mismatch, ok := errors.AsType[*VisibilityMismatchError](err); ok {
					mismatch.File = "snippet"
				}
				return fmt.Errorf("validate access modifier for %q: %w", name, err)
			}
		}
	}
	return nil
}

func declEffectiveAccess(decls []ast.Decl, requested AccessModifier) AccessModifier {
	if len(decls) == 0 {
		return AccessModifierPublic
	}
	names := extractDeclNames(decls[0])
	if len(names) == 0 {
		return AccessModifierPublic
	}
	access, err := ResolveAccess(DefaultBackend, requested, names[0])
	if err != nil {
		return AccessModifierPublic
	}
	return access
}

func insertDeclIntoGroup(ctx context.Context, target *declInsertionTarget, source string, opts DeclOptions) (bool, error) {
	if opts.Group != "" && opts.Group != "append" || opts.Placement != "" || len(target.snippetDecls) != 1 {
		return false, nil
	}
	incoming, ok := target.snippetDecls[0].(*ast.GenDecl)
	if !ok || (incoming.Tok != token.CONST && incoming.Tok != token.VAR) {
		return false, nil
	}
	group := findMatchingGroup(target.file, incoming.Tok, target.effectiveAccess)
	if group == nil || !group.Rparen.IsValid() {
		return false, nil
	}
	return true, writeGroupedDecl(ctx, target, group, incoming, source, opts.AutoOrganizeImports)
}

func writeGroupedDecl(ctx context.Context, target *declInsertionTarget, group, incoming *ast.GenDecl, source string, organizeImports bool) error {
	specSource := extractSpecSource(source, incoming.Tok)
	insertOffset := target.fset.Position(group.Rparen).Offset
	if incoming.Tok == token.VAR {
		if sentinelOffset, ok := sentinelVarInsertionOffset(target.fset, group, incoming); ok {
			insertOffset = sentinelOffset
		}
	}
	insertOffset = normalizeInsertionOffset(target.fset, target.file, target.content, insertOffset)
	var buffer bytes.Buffer
	buffer.Write(target.content[:insertOffset])
	groupEnd := target.fset.Position(group.Rparen).Offset
	writeGroupedDeclSeparator(&buffer, target.content, insertOffset, groupEnd, specSource)
	buffer.Write(target.content[insertOffset:])
	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("preflight format grouped declaration insertion: %w", err)
	}
	return writeAndOrganize(ctx, target.cleanPath, formatted, organizeImports)
}

func writeGroupedDeclSeparator(buffer *bytes.Buffer, content []byte, insertOffset, groupEnd int, source string) {
	switch {
	case insertOffset != groupEnd:
		buffer.WriteString(source)
		buffer.WriteString("\n\t")
	case !bytes.HasSuffix(content[:insertOffset], []byte("\n")):
		buffer.WriteString("\n\t")
	default:
		buffer.WriteString("\t")
	}
	if insertOffset == groupEnd {
		buffer.WriteString(source)
		buffer.WriteString("\n")
	}
}

func insertDeclAtOffset(ctx context.Context, target *declInsertionTarget, source string, opts DeclOptions) error {
	insertOffset, err := calculateDeclOffset(target.fset, target.file, target.content, target.effectiveAccess, opts)
	if err != nil {
		return enrichDeclPlacementError(err, target)
	}
	insertOffset = normalizeInsertionOffset(target.fset, target.file, target.content, insertOffset)
	newContent := buildDeclInsertionContent(target.content, insertOffset, source)
	return writeAndOrganize(ctx, target.cleanPath, newContent, opts.AutoOrganizeImports)
}

func enrichDeclPlacementError(err error, target *declInsertionTarget) error {
	if placementErr, ok := errors.AsType[*PlacementError](err); ok {
		if placementErr.File == "" {
			placementErr.File = target.cleanPath
		}
		if !placementErr.Pos.IsValid() && target.file.Package.IsValid() {
			placementErr.Pos = target.fset.Position(target.file.Package)
		}
	}
	return fmt.Errorf("calculate declaration offset: %w", err)
}

func buildDeclInsertionContent(content []byte, insertOffset int, source string) []byte {
	var newContent bytes.Buffer
	newContent.Write(content[:insertOffset])
	snippetText := normalizePackageDeclSnippet(source)
	if insertOffset > 0 {
		writeDeclLeadingSeparator(&newContent, content[:insertOffset])
	}
	newContent.WriteString(snippetText)
	newContent.WriteString("\n\n")
	newContent.Write(bytes.TrimLeft(content[insertOffset:], "\r\n"))
	return newContent.Bytes()
}

func normalizePackageDeclSnippet(source string) string {
	snippet := strings.TrimSpace(source)
	if !strings.HasPrefix(snippet, "package ") {
		return snippet
	}
	lines := strings.SplitN(snippet, "\n", 2)
	if len(lines) == 1 {
		return snippet
	}
	return strings.TrimSpace(lines[1])
}

func writeDeclLeadingSeparator(buffer *bytes.Buffer, previous []byte) {
	if bytes.HasSuffix(previous, []byte("\n\n")) {
		return
	}
	if bytes.HasSuffix(previous, []byte("\n")) {
		buffer.WriteString("\n")
		return
	}
	buffer.WriteString("\n\n")
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
			position := valueSpec.Pos()
			if valueSpec.Doc != nil {
				position = valueSpec.Doc.Pos()
			}
			return fset.Position(position).Offset, true
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
	prefix := "package semedit\n"
	if strings.HasPrefix(strings.TrimSpace(raw), "package ") {
		prefix = ""
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", prefix+raw, parser.ParseComments|parser.AllErrors)
	if err != nil || len(file.Decls) != 1 {
		return strings.TrimSpace(raw)
	}
	gen, ok := file.Decls[0].(*ast.GenDecl)
	if !ok || gen.Tok != tok {
		return strings.TrimSpace(raw)
	}

	fileToken := fset.File(gen.Pos())
	if fileToken == nil {
		return strings.TrimSpace(raw)
	}
	sourceOffset := func(position token.Pos) int {
		return fileToken.Offset(position) - len(prefix)
	}
	leadingDoc := gen.Doc
	if leadingDoc == nil {
		leadingDoc = nearestLeadingDeclComment(fset, file, gen)
	}

	parts := make([]string, 0, len(gen.Specs))
	for index, spec := range gen.Specs {
		start, end := specSourcePositions(spec)
		startOffset := sourceOffset(start)
		endOffset := sourceOffset(end)
		if index == 0 && leadingDoc != nil {
			docStart := sourceOffset(leadingDoc.Pos())
			tokStart := sourceOffset(gen.TokPos)
			specStart := sourceOffset(spec.Pos())
			if docStart >= 0 && docStart < tokStart && tokStart <= specStart {
				parts = append(parts, strings.TrimSpace(raw[docStart:tokStart]+raw[specStart:endOffset]))
				continue
			}
		}
		if startOffset < 0 || endOffset > len(raw) || startOffset >= endOffset {
			return strings.TrimSpace(raw)
		}
		parts = append(parts, strings.TrimSpace(raw[startOffset:endOffset]))
	}
	return strings.Join(parts, "\n")
}

func nearestLeadingDeclComment(fset *token.FileSet, file *ast.File, decl *ast.GenDecl) *ast.CommentGroup {
	var leading *ast.CommentGroup
	for _, comment := range file.Comments {
		if comment.End() <= decl.TokPos && fset.Position(comment.End()).Line+1 >= fset.Position(decl.TokPos).Line {
			leading = comment
		}
	}
	return leading
}

func specSourcePositions(spec ast.Spec) (token.Pos, token.Pos) {
	start, end := spec.Pos(), spec.End()
	valueSpec, ok := spec.(*ast.ValueSpec)
	if !ok {
		return start, end
	}
	if valueSpec.Doc != nil {
		start = valueSpec.Doc.Pos()
	}
	if valueSpec.Comment != nil {
		end = valueSpec.Comment.End()
	}
	return start, end
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
	if err := pipeline.WriteAtomicContext(ctx, cleanPath, data); err != nil {
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
