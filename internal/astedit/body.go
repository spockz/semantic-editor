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
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
	"semedit/internal/telemetry"
)

// BodyOptions configures function and method body replacement behavior.
type BodyOptions struct {
	AutoOrganizeImports bool
}

// ReplaceBody replaces the statements of an existing function or method body.
func ReplaceBody(ctx context.Context, filePath, symbolQuery, bodySource string, opts BodyOptions) (string, error) {
	if sourcebuf.FromContext(ctx) == nil {
		var result string
		err := runBufferedFile(ctx, filePath, func(editCtx context.Context) error {
			var err error
			result, err = ReplaceBody(editCtx, filePath, symbolQuery, bodySource, opts)
			return err
		})
		return result, err
	}
	cleanPath := filepath.Clean(filePath)
	target, err := readBodyTarget(ctx, cleanPath, filePath, symbolQuery)
	if err != nil {
		return "", err
	}
	bodyTrimmed, err := validateBodySnippet(bodySource)
	if err != nil {
		return "", err
	}
	formatted, err := formatBodyReplacement(ctx, cleanPath, target, bodySource, bodyTrimmed)
	if err != nil {
		return "", err
	}
	if err := publishBodyReplacement(ctx, cleanPath, formatted, filepath.Dir(cleanPath), opts.AutoOrganizeImports); err != nil {
		return "", err
	}
	return UnifiedDiff(symbolQuery, strings.TrimSpace(target.oldBody), bodyTrimmed), nil
}

type bodyTarget struct {
	content  []byte
	fset     *token.FileSet
	function *ast.FuncDecl
	oldBody  string
}

func readBodyTarget(ctx context.Context, cleanPath, filePath, symbolQuery string) (*bodyTarget, error) {
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read target file: %w", err)
	}
	receiver, name, err := symbol.ParseIdentifier(symbolQuery)
	if err != nil {
		return nil, &symbol.SymbolError{Op: "replace_body", File: filePath, Symbol: symbolQuery, Err: err}
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments)
	if err != nil {
		pos := extractSyntaxPosition(fset, err)
		return nil, &SyntaxError{
			File:  cleanPath,
			Pos:   pos,
			Cause: err,
			Err:   ErrSyntax,
		}
	}

	var target *ast.FuncDecl
	for _, decl := range fileNode.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		fnRecv := extractReceiverName(fn.Recv)
		if fnRecv == receiver && fn.Name.Name == name {
			target = fn
			break
		}
	}

	if target == nil {
		return nil, &symbol.SymbolError{
			Op:     "replace_body",
			File:   filePath,
			Symbol: symbolQuery,
			Err:    symbol.ErrNotFound,
		}
	}

	if target.Body == nil {
		pos := fset.Position(target.Pos())
		return nil, &symbol.SymbolError{
			Op:     "replace_body",
			File:   filePath,
			Symbol: symbolQuery,
			Pos:    pos,
			Err:    ErrNoBody,
		}
	}

	lbraceOffset := fset.Position(target.Body.Lbrace).Offset
	rbraceOffset := fset.Position(target.Body.Rbrace).Offset
	return &bodyTarget{
		content:  content,
		fset:     fset,
		function: target,
		oldBody:  string(content[lbraceOffset+1 : rbraceOffset]),
	}, nil
}

func validateBodySnippet(bodySource string) (string, error) {
	bodyTrimmed := strings.TrimSpace(bodySource)
	prefix := "package dummy\n\nfunc _() {\n"
	synthetic := prefix + bodyTrimmed + "\n}\n"
	synthFset := token.NewFileSet()
	file, err := parser.ParseFile(synthFset, "snippet", synthetic, parser.ParseComments)
	if err != nil {
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
	if len(file.Decls) != 1 {
		return "", fmt.Errorf("replacement body must remain inside its function wrapper")
	}
	function, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || function.Name.Name != "_" || function.Body == nil {
		return "", fmt.Errorf("replacement body wrapper is not one complete function")
	}
	if len(function.Body.List) == 1 {
		if block, ok := function.Body.List[0].(*ast.BlockStmt); ok {
			start := synthFset.Position(block.Lbrace).Offset - len(prefix)
			end := synthFset.Position(block.Rbrace).Offset - len(prefix)
			if start >= 0 && end >= start && end < len(bodyTrimmed) &&
				strings.TrimSpace(bodyTrimmed[:start]) == "" &&
				strings.TrimSpace(bodyTrimmed[end+1:]) == "" {
				bodyTrimmed = strings.TrimSpace(bodyTrimmed[start+1 : end])
			}
		}
	}
	return bodyTrimmed, nil
}

func formatBodyReplacement(ctx context.Context, cleanPath string, target *bodyTarget, bodySource, bodyTrimmed string) ([]byte, error) {
	lbraceOffset := target.fset.Position(target.function.Body.Lbrace).Offset
	rbraceOffset := target.fset.Position(target.function.Body.Rbrace).Offset
	var newContent bytes.Buffer
	newContent.Write(target.content[:lbraceOffset+1])
	newContent.WriteString("\n")
	newContent.WriteString(bodyTrimmed)
	newContent.WriteString("\n")
	newContent.Write(target.content[rbraceOffset:])

	finishFormatting := telemetry.Start(ctx, telemetry.PhaseFormattingAST)
	formatted, err := format.Source(newContent.Bytes())
	finishFormatting()
	if err != nil {
		return nil, &SyntaxError{
			File:    cleanPath,
			Snippet: bodySource,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}
	return formatted, nil
}

func publishBodyReplacement(ctx context.Context, cleanPath string, formatted []byte, directory string, organizeImports bool) error {
	formatted, err := cleanupMoveImports(ctx, cleanPath, formatted)
	if err != nil {
		return fmt.Errorf("prune orphan imports: %w", err)
	}
	if err := pipeline.WriteAtomicContext(ctx, cleanPath, formatted); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}

	if organizeImports {
		if err := pipeline.OrganizeImports(ctx, directory, cleanPath); err != nil {
			_ = pipeline.Format(ctx, directory, cleanPath)
			return fmt.Errorf("organize imports: %w", err)
		}
	} else {
		if err := pipeline.Format(ctx, directory, cleanPath); err != nil {
			return fmt.Errorf("format file: %w", err)
		}
	}
	return nil
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
