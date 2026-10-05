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
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
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
	return insertFunctionImpl(ctx, filePath, source, opts)
}

// InsertFunction injects a function or method declaration into filePath.
func insertFunctionImpl(ctx context.Context, filePath string, source string, opts FunctionOptions) error {
	source = normalizeDeclarationSnippet(source)
	opts.TargetSymbol = strings.Trim(strings.TrimSpace(opts.TargetSymbol), `"'`)
	target, err := readDeclarationTarget(ctx, filePath)
	if err != nil {
		return err
	}

	fnDecl, err := parseFunctionSnippet(source)
	if err != nil {
		return fmt.Errorf("validate function snippet: %w", err)
	}

	fnName := fnDecl.Name.Name
	if accessErr := ValidateAccess(DefaultBackend, opts.AccessModifier, fnName); accessErr != nil {
		if visErr, ok := errors.AsType[*VisibilityMismatchError](accessErr); ok {
			visErr.File = "snippet"
		}
		return fmt.Errorf("validate access modifier: %w", accessErr)
	}

	effectiveAccess, err := resolveDeclarationAccess(opts.AccessModifier, fnName)
	if err != nil {
		return err
	}
	if collisionErr := checkConstructDeclarationCollision(ctx, target.path, target.file, fnDecl); collisionErr != nil {
		return fmt.Errorf("check declaration collision: %w", collisionErr)
	}

	insertOffset, err := calculateFunctionOffset(target.set, target.file, target.content, fnDecl, effectiveAccess, opts)
	if err != nil {
		return fmt.Errorf("calculate function offset: %w", annotatePlacementError(err, target.path, target.set, target.file))
	}
	insertOffset = normalizeInsertionOffset(target.set, target.file, target.content, insertOffset)
	updated := insertDeclarationText(target.content, source, insertOffset)
	return publishInsertedDeclaration(ctx, target.path, updated, opts.AutoOrganizeImports)
}

type declarationInsertionTarget struct {
	path    string
	content []byte
	set     *token.FileSet
	file    *ast.File
}

func normalizeDeclarationSnippet(source string) string {
	source = strings.TrimSpace(source)
	quoted := (strings.HasPrefix(source, "\"") && strings.HasSuffix(source, "\"")) ||
		(strings.HasPrefix(source, "'") && strings.HasSuffix(source, "'"))
	if quoted {
		return source[1 : len(source)-1]
	}
	return source
}

func readDeclarationTarget(ctx context.Context, filePath string) (declarationInsertionTarget, error) {
	path := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return declarationInsertionTarget{}, fmt.Errorf("read target file: %w", err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return declarationInsertionTarget{}, fmt.Errorf("parse target file before insertion: %w", err)
	}
	return declarationInsertionTarget{path: path, content: content, set: set, file: file}, nil
}

func resolveDeclarationAccess(access AccessModifier, name string) (AccessModifier, error) {
	effective, err := ResolveAccess(DefaultBackend, access, name)
	if err != nil {
		return "", fmt.Errorf("resolve effective access: %w", err)
	}
	return effective, nil
}

func annotatePlacementError(err error, path string, set *token.FileSet, file *ast.File) error {
	placementErr, ok := errors.AsType[*PlacementError](err)
	if !ok {
		return err
	}
	if placementErr.File == "" {
		placementErr.File = path
	}
	if !placementErr.Pos.IsValid() && file != nil && file.Package.IsValid() {
		placementErr.Pos = set.Position(file.Package)
	}
	return placementErr
}

func insertDeclarationText(content []byte, source string, offset int) []byte {
	var updated bytes.Buffer
	updated.Write(content[:offset])
	snippet := strings.TrimSpace(source)
	if strings.HasPrefix(snippet, "package ") {
		lines := strings.SplitN(snippet, "\n", 2)
		if len(lines) > 1 {
			snippet = strings.TrimSpace(lines[1])
		}
	}
	if offset > 0 && !bytes.HasSuffix(content[:offset], []byte("\n\n")) {
		if bytes.HasSuffix(content[:offset], []byte("\n")) {
			updated.WriteString("\n")
		} else {
			updated.WriteString("\n\n")
		}
	}
	updated.WriteString(snippet)
	updated.WriteString("\n\n")
	updated.Write(bytes.TrimLeft(content[offset:], "\r\n"))
	return updated.Bytes()
}

func publishInsertedDeclaration(ctx context.Context, path string, content []byte, organizeImports bool) error {
	if _, err := parser.ParseFile(token.NewFileSet(), path, content, parser.AllErrors); err != nil {
		return fmt.Errorf("validate inserted file: %w", err)
	}
	if err := pipeline.WriteAtomicContext(ctx, path, content); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}
	directory := filepath.Dir(path)
	if organizeImports {
		if err := pipeline.OrganizeImports(ctx, directory, path); err != nil {
			organizeErr := fmt.Errorf("organize imports: %w", err)
			if formatErr := pipeline.Format(ctx, directory, path); formatErr != nil {
				return errors.Join(organizeErr, fmt.Errorf("format file after import organization failure: %w", formatErr))
			}
			return organizeErr
		}
		return nil
	}
	if err := pipeline.Format(ctx, directory, path); err != nil {
		return fmt.Errorf("format file: %w", err)
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
	receiver := recv.List[0].Type
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = pointer.X
	}
	switch typed := receiver.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.IndexExpr:
		if ident, ok := typed.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.IndexListExpr:
		if ident, ok := typed.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func calculateFunctionOffset(fset *token.FileSet, fileNode *ast.File, content []byte, fnDecl *ast.FuncDecl, effectiveAccess AccessModifier, opts FunctionOptions) (int, error) {
	if opts.Placement != "" {
		return calculateExplicitFunctionOffset(fset, fileNode, content, fnDecl, effectiveAccess, opts)
	}
	recvType := extractReceiverTypeName(fnDecl.Recv)
	if recvType == "" {
		return calculateTopLevelFunctionOffset(fset, fileNode, content, effectiveAccess)
	}
	return calculateReceiverFunctionOffset(fset, fileNode, content, recvType, effectiveAccess)
}

func calculateExplicitFunctionOffset(fset *token.FileSet, fileNode *ast.File, content []byte, function *ast.FuncDecl, access AccessModifier, opts FunctionOptions) (int, error) {
	public := access == AccessModifierPublic
	privateSection := opts.Placement == PlacementPrivateStart || opts.Placement == PlacementPrivateEnd
	publicSection := opts.Placement == PlacementPublicStart || opts.Placement == PlacementPublicEnd
	if (public && privateSection) || (access == AccessModifierPrivate && publicSection) {
		return 0, &PlacementError{Strategy: opts.Placement, TargetSymbol: function.Name.Name, Err: ErrSectionViolation}
	}
	return calculateInsertionOffset(fset, fileNode, content, Options{Placement: opts.Placement, TargetSymbol: opts.TargetSymbol})
}

func calculateTopLevelFunctionOffset(fset *token.FileSet, fileNode *ast.File, content []byte, access AccessModifier) (int, error) {
	placement := PlacementPrivateEnd
	if access == AccessModifierPublic {
		placement = PlacementPublicEnd
	}
	return calculateInsertionOffset(fset, fileNode, content, Options{Placement: placement})
}

func calculateReceiverFunctionOffset(fset *token.FileSet, fileNode *ast.File, content []byte, receiver string, access AccessModifier) (int, error) {
	targets := receiverFunctionTargets(fileNode, receiver)
	wantedAccess := access
	if wantedAccess == AccessModifierPublic && len(targets.methods.public) > 0 {
		return fset.Position(targets.methods.public[len(targets.methods.public)-1].End()).Offset, nil
	}
	if wantedAccess == AccessModifierPublic && targets.typeDeclaration != nil && isDeclExported(targets.typeDeclaration) {
		return fset.Position(targets.typeDeclaration.End()).Offset, nil
	}
	if wantedAccess == AccessModifierPrivate && len(targets.methods.private) > 0 {
		return fset.Position(targets.methods.private[len(targets.methods.private)-1].End()).Offset, nil
	}
	placement := PlacementPrivateStart
	if wantedAccess == AccessModifierPublic {
		placement = PlacementPublicEnd
	}
	return calculateInsertionOffset(fset, fileNode, content, Options{Placement: placement})
}

type receiverFunctionMethods struct {
	public  []*ast.FuncDecl
	private []*ast.FuncDecl
}

type receiverFunctionTargetSet struct {
	methods         receiverFunctionMethods
	typeDeclaration ast.Decl
}

func receiverFunctionTargets(file *ast.File, receiver string) receiverFunctionTargetSet {
	var targets receiverFunctionTargetSet
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			if extractReceiverTypeName(function.Recv) == receiver {
				targets.methods.add(function)
			}
			continue
		}
		if declarationTypeMatches(declaration, receiver) {
			targets.typeDeclaration = declaration
		}
	}
	return targets
}

func (methods *receiverFunctionMethods) add(function *ast.FuncDecl) {
	if isDeclExported(function) {
		methods.public = append(methods.public, function)
		return
	}
	methods.private = append(methods.private, function)
}

func declarationTypeMatches(declaration ast.Decl, name string) bool {
	group, ok := declaration.(*ast.GenDecl)
	if !ok || group.Tok != token.TYPE {
		return false
	}
	for _, spec := range group.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == name {
			return true
		}
	}
	return false
}
