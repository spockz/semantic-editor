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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/pipeline"
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

	snippetDecls, err := verifySnippetSyntax(source, opts.Visibility)
	if err != nil {
		return fmt.Errorf("validate declaration snippet: %w", err)
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil && fileNode == nil {
		// Fall back to appending at EOF for unparseable files (ADR-0002)
		return appendToEOF(ctx, cleanPath, content, source, opts.AutoOrganizeImports)
	}

	insertOffset, err := calculateInsertionOffset(fset, fileNode, content, opts)
	if err != nil {
		return fmt.Errorf("calculate insertion offset: %w", err)
	}

	var newContent bytes.Buffer
	newContent.Write(content[:insertOffset])

	snippetText := strings.TrimSpace(source)
	// Strip package declaration from snippet if user provided it
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
	// Avoid excess leading newlines in trailing content
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

	_ = snippetDecls
	return nil
}

func verifySnippetSyntax(source string, expectedVisibility string) ([]ast.Decl, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil, errors.New("empty declaration snippet")
	}

	toParse := trimmed
	if !strings.HasPrefix(trimmed, "package ") {
		toParse = "package dummy\n\n" + trimmed
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "snippet.go", toParse, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	if len(node.Decls) == 0 {
		return nil, errors.New("no Go declarations found in snippet")
	}

	if expectedVisibility != "" {
		for _, decl := range node.Decls {
			names := extractDeclNames(decl)
			for _, name := range names {
				exported := ast.IsExported(name)
				if expectedVisibility == "public" && !exported {
					return nil, fmt.Errorf("visibility mismatch: expected public symbol, found private %q", name)
				}
				if expectedVisibility == "private" && exported {
					return nil, fmt.Errorf("visibility mismatch: expected private symbol, found public %q", name)
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
			return 0, errors.New("target symbol required for before_symbol and after_symbol placement")
		}
		targetDecl, err := findTargetDecl(fileNode, opts.TargetSymbol)
		if err != nil {
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
	parts := strings.Split(targetSymbol, ".")
	var targetReceiver, targetName string
	if len(parts) == 2 {
		targetReceiver = parts[0]
		targetName = parts[1]
	} else {
		targetName = parts[0]
	}

	for _, decl := range fileNode.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			var rec string
			if d.Recv != nil && len(d.Recv.List) > 0 {
				switch r := d.Recv.List[0].Type.(type) {
				case *ast.Ident:
					rec = r.Name
				case *ast.StarExpr:
					if id, ok := r.X.(*ast.Ident); ok {
						rec = id.Name
					}
				}
			}
			if targetReceiver != "" {
				if rec == targetReceiver && d.Name.Name == targetName {
					return decl, nil
				}
			} else if d.Name.Name == targetName {
				return decl, nil
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == targetName {
						return decl, nil
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.Name == targetName {
							return decl, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("symbol %q not found", targetSymbol)
}

func appendToEOF(ctx context.Context, filePath string, content []byte, source string, autoOrganize bool) error {
	var buf bytes.Buffer
	buf.Write(content)
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n\n")) {
		if bytes.HasSuffix(content, []byte("\n")) {
			buf.WriteString("\n")
		} else {
			buf.WriteString("\n\n")
		}
	}
	buf.WriteString(strings.TrimSpace(source))
	buf.WriteString("\n")

	if err := pipeline.WriteAtomic(filePath, buf.Bytes()); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}

	dir := filepath.Dir(filePath)
	if autoOrganize {
		return pipeline.OrganizeImports(ctx, dir, filePath)
	}
	return pipeline.Format(ctx, dir, filePath)
}
