// Package astedit exposes read-only construct target validation so planners
// can resolve native target errors before producing a replacement body.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
)

// ValidateConstructReplacementTarget validates the native target using the
// same replacement identity constraints as ReplaceConstruct, without writing.
func ValidateConstructReplacementTarget(ctx context.Context, filePath, funcName string, kind ConstructKind, discriminator, source string, opts ConstructOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Discriminator != "" {
		discriminator = opts.Discriminator
	}
	if opts.ExpectedSelectionRevision != "" && !containsConstructKind(GoInspectConstructKinds, kind) {
		return fmt.Errorf("expected_selection_revision is supported only for nested construct kinds")
	}
	if kind == ConstructStatementOrExpr {
		owner := opts.InFunction
		if owner == "" {
			owner = funcName
		}
		return validateStatementExpressionTarget(ctx, filePath, owner, discriminator, opts)
	}
	if kind == ConstructFunction || kind == ConstructMethod || kind == ConstructType || kind == ConstructConst || kind == ConstructVar {
		symbolQuery := opts.Symbol
		if symbolQuery == "" {
			symbolQuery = funcName
		}
		return validateDeclarationReplacementTarget(ctx, filePath, symbolQuery, kind, source)
	}
	if !validConstructKind(kind) {
		return fmt.Errorf("unsupported Go construct kind %q", kind)
	}
	return validateControlConstructTarget(ctx, filePath, funcName, kind, discriminator, opts)
}

func validateControlConstructTarget(ctx context.Context, filePath, funcName string, kind ConstructKind, discriminator string, opts ConstructOptions) error {
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments)
	if err != nil {
		return &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	target, err := findControlConstructOwner(fset, file, filePath, funcName)
	if err != nil {
		return err
	}
	candidateSet, err := inspectConstructCandidatesFromSource(cleanPath, content, funcName, kind, discriminator)
	if err != nil {
		return err
	}
	if opts.ExpectedSelectionRevision != "" && opts.ExpectedSelectionRevision != candidateSet.SelectionRevision {
		return selectionError("selection_revision_mismatch", candidateSet, nil, opts.ExpectedSelectionRevision)
	}
	_, err = selectControlConstruct(fset, target.Body, funcName, kind, discriminator, opts.ConstructPath)
	if err != nil {
		return err
	}
	return ctx.Err()
}

func validateStatementExpressionTarget(ctx context.Context, filePath, ownerName, discriminator string, opts ConstructOptions) error {
	if strings.TrimSpace(discriminator) == "" {
		return fmt.Errorf("discriminator is required for statement/expression replacement")
	}
	target, err := readStatementExpressionTarget(ctx, filePath, ownerName)
	if err != nil {
		return err
	}
	if opts.ExpectedSelectionRevision != "" {
		candidateSet, inspectErr := inspectConstructCandidatesFromSource(target.cleanPath, target.content, ownerName, ConstructStatementOrExpr, discriminator)
		if inspectErr != nil {
			return inspectErr
		}
		if opts.ExpectedSelectionRevision != candidateSet.SelectionRevision {
			return selectionError("selection_revision_mismatch", candidateSet, nil, opts.ExpectedSelectionRevision)
		}
	}
	_, err = selectStatementExpressionCandidate(target, discriminator, opts.ConstructPath)
	if err != nil {
		return err
	}
	return ctx.Err()
}

func validateDeclarationReplacementTarget(ctx context.Context, filePath, symbolQuery string, kind ConstructKind, source string) error {
	if strings.TrimSpace(symbolQuery) == "" {
		return fmt.Errorf("symbol is required for %s replacement", kind)
	}
	if kind == ConstructFunction || kind == ConstructMethod {
		replacement, err := parseFunctionReplacement(symbolQuery, kind, source)
		if err != nil {
			return err
		}
		target, err := readDeclarationTarget(ctx, filePath)
		if err != nil {
			return err
		}
		selected, err := findFunctionConstructTarget(target.file, replacement, symbolQuery, target.path)
		if err != nil {
			return err
		}
		if err := validateFunctionReplacementIdentity(replacement, symbolQuery, kind); err != nil {
			return err
		}
		if selected.Recv != nil && receiverTypeKey(selected.Recv) != receiverTypeKey(replacement.declaration.Recv) {
			return fmt.Errorf("replacement method receiver must preserve the selected receiver syntax")
		}
		return ctx.Err()
	}
	decls, err := verifySnippetSyntax(source, "")
	if err != nil {
		return err
	}
	if len(decls) != 1 {
		return ErrMultipleDeclarations
	}
	replacementSyntax, ok := decls[0].(*ast.GenDecl)
	if !ok {
		return fmt.Errorf("replacement source must be one const, var, or type declaration")
	}
	if (kind == ConstructType && replacementSyntax.Tok != token.TYPE) || (kind == ConstructConst && replacementSyntax.Tok != token.CONST) || (kind == ConstructVar && replacementSyntax.Tok != token.VAR) {
		return fmt.Errorf("replacement declaration does not match construct kind %q", kind)
	}
	target, err := loadDeclarationReplacementTarget(ctx, filePath)
	if err != nil {
		return err
	}
	targetKind, found, err := namedDeclarationKind(target.file, symbolQuery)
	if err != nil {
		return err
	}
	if !found {
		return &symbol.SymbolError{Op: replaceConstructOperation, File: target.path, Symbol: symbolQuery, Err: symbol.ErrNotFound}
	}
	if targetKind != kind {
		return fmt.Errorf("selected declaration %q is %s, not %s", symbolQuery, targetKind, kind)
	}
	replacement, err := parseSingleReplacementDecl(source, symbolQuery)
	if err != nil {
		return err
	}
	match, err := findDeclarationReplacementMatch(target.file, symbolQuery, target.path)
	if err != nil {
		return err
	}
	if err := validateDeclarationReplacement(match, replacement, symbolQuery); err != nil {
		return err
	}
	return ctx.Err()
}
