// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/symbol"
	"semedit/internal/telemetry"
)

// BodyOptions configures function and method body replacement behavior.
type BodyOptions struct {
	AutoOrganizeImports bool
}

// ReplaceBody replaces the statements of an existing function or method body.
func ReplaceBody(ctx context.Context, filePath, symbolQuery, bodySource string, opts BodyOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}

	recv, name, err := symbol.ParseIdentifier(symbolQuery)
	if err != nil {
		return "", &symbol.SymbolError{
			Op:     "replace_body",
			File:   filePath,
			Symbol: symbolQuery,
			Err:    err,
		}
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments)
	if err != nil {
		pos := extractSyntaxPosition(fset, err)
		return "", &SyntaxError{
			File:  cleanPath,
			Pos:   pos,
			Cause: err,
			Err:   ErrSyntax,
		}
	}

	var targetFunc *ast.FuncDecl
	for _, decl := range fileNode.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		fnRecv := extractReceiverName(fn.Recv)
		if fnRecv == recv && fn.Name.Name == name {
			targetFunc = fn
			break
		}
	}

	if targetFunc == nil {
		return "", &symbol.SymbolError{
			Op:     "replace_body",
			File:   filePath,
			Symbol: symbolQuery,
			Err:    symbol.ErrNotFound,
		}
	}

	if targetFunc.Body == nil {
		pos := fset.Position(targetFunc.Pos())
		return "", &symbol.SymbolError{
			Op:     "replace_body",
			File:   filePath,
			Symbol: symbolQuery,
			Pos:    pos,
			Err:    ErrNoBody,
		}
	}

	bodyTrimmed := strings.TrimSpace(bodySource)
	synthetic := "package dummy\n\nfunc _() {\n" + bodyTrimmed + "\n}\n"
	synthFset := token.NewFileSet()
	if _, err := parser.ParseFile(synthFset, "snippet", synthetic, parser.ParseComments); err != nil {
		pos := extractSyntaxPosition(synthFset, err)
		if pos.Line > 3 {
			pos.Line -= 3
		}
		return "", &SyntaxError{
			File:    "snippet",
			Snippet: bodySource,
			Pos:     pos,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	lbraceOffset := fset.Position(targetFunc.Body.Lbrace).Offset
	rbraceOffset := fset.Position(targetFunc.Body.Rbrace).Offset
	oldBody := string(content[lbraceOffset+1 : rbraceOffset])

	var newContent bytes.Buffer
	newContent.Write(content[:lbraceOffset+1])
	newContent.WriteString("\n")
	newContent.WriteString(bodyTrimmed)
	newContent.WriteString("\n")
	newContent.Write(content[rbraceOffset:])

	finishFormatting := telemetry.Start(ctx, telemetry.PhaseFormattingAST)
	formatted, err := format.Source(newContent.Bytes())
	finishFormatting()
	if err != nil {
		return "", &SyntaxError{
			File:    cleanPath,
			Snippet: bodySource,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}

	dir := filepath.Dir(cleanPath)
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, dir, cleanPath); err != nil {
			_ = pipeline.Format(ctx, dir, cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	} else {
		if err := pipeline.Format(ctx, dir, cleanPath); err != nil {
			return "", fmt.Errorf("format file: %w", err)
		}
	}

	diff := UnifiedDiff(symbolQuery, strings.TrimSpace(oldBody), bodyTrimmed)
	return diff, nil
}

func extractReceiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	for {
		switch node := t.(type) {
		case *ast.StarExpr:
			t = node.X
		case *ast.IndexExpr:
			t = node.X
		case *ast.IndexListExpr:
			t = node.X
		case *ast.Ident:
			return node.Name
		default:
			return ""
		}
	}
}
