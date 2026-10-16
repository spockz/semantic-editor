// Package astedit shares remove target preflight so suggestion validation proves the same acceptance as mutation.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type removePreflight struct {
	cleanPath string
	content   []byte
	fset      *token.FileSet
	file      *ast.File
	target    removeTarget
	checks    []string
	snapshot  *safeDeleteSnapshot
	refusal   string
}

// ValidateRemoveTarget proves that Remove would accept the selected target without
// formatting, publishing, or otherwise changing the current source view.
func ValidateRemoveTarget(ctx context.Context, file string, opts RemoveOptions) error {
	preflight, err := preflightRemoveTarget(ctx, file, opts)
	if err != nil {
		return err
	}
	if preflight.refusal != "" {
		return fmt.Errorf("safe_delete refused: %s", preflight.refusal)
	}
	return nil
}

func preflightRemoveTarget(ctx context.Context, filePath string, opts RemoveOptions) (removePreflight, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return removePreflight{}, fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return removePreflight{}, &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	target, err := resolveRemoveTarget(fset, fileNode, content, opts, cleanPath)
	if err != nil {
		return removePreflight{}, err
	}
	if opts.SafeDelete && target.class == "statement" {
		return removePreflight{}, fmt.Errorf("safe_delete is supported only for eligible top-level declarations")
	}
	preflight := removePreflight{
		cleanPath: cleanPath,
		content:   content,
		fset:      fset,
		file:      fileNode,
		target:    target,
		checks:    []string{"go_syntax_parse", "target_resolution"},
	}
	if opts.SafeDelete {
		preflight.snapshot, preflight.checks, preflight.refusal = safeDeleteDeclaration(ctx, fset, fileNode, target.node, cleanPath, content, preflight.checks)
	}
	return preflight, nil
}
