// Package astedit provides shared byte-snapshot projections for semantic read operations.
package astedit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"semedit/internal/backend"
	"strconv"
	"strings"
)

type goReadSource struct {
	path        string
	displayPath string
	source      []byte
	file        *ast.File
	fset        *token.FileSet
}

func parseGoReadSource(root, path string, source []byte) (*goReadSource, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve Go source path %q: %w", path, err)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Go workspace %q: %w", root, err)
	}
	displayPath := filepath.ToSlash(absolutePath)
	relativePath, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil {
		return nil, fmt.Errorf("make Go source path relative to workspace: %w", err)
	}
	if relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		displayPath = filepath.ToSlash(relativePath)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absolutePath, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		position := extractSyntaxPosition(fset, err)
		position.Filename = displayPath
		return nil, &SyntaxError{File: displayPath, Pos: position, Cause: err, Err: ErrSyntax}
	}
	return &goReadSource{path: absolutePath, displayPath: displayPath, source: source, file: file, fset: fset}, nil
}

func goReadNodeText(source []byte, fset *token.FileSet, node ast.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("read Go source node: nil node")
	}
	start := fset.PositionFor(node.Pos(), false).Offset
	end := fset.PositionFor(node.End(), false).Offset
	if start < 0 || end < start || end > len(source) {
		return "", fmt.Errorf("read Go source node: invalid byte span [%d,%d) for %d bytes", start, end, len(source))
	}
	return string(source[start:end]), nil
}

func goReadRange(source []byte, fset *token.FileSet, start, end token.Pos) (backend.Range, error) {
	startOffset := fset.PositionFor(start, false).Offset
	endOffset := fset.PositionFor(end, false).Offset
	if startOffset < 0 || endOffset < startOffset || endOffset > len(source) {
		return backend.Range{}, fmt.Errorf("convert Go source range: invalid byte span [%d,%d) for %d bytes", startOffset, endOffset, len(source))
	}
	return backend.Range{
		Start: backend.PositionFromByteOffset(source, startOffset),
		End:   backend.PositionFromByteOffset(source, endOffset),
	}, nil
}

func goReadFormattedNode(fset *token.FileSet, node any) (string, error) {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		return "", fmt.Errorf("format Go declaration: %w", err)
	}
	return buffer.String(), nil
}

func goReadRevision(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}

//nolint:nilnil // A missing comment is represented as an omitted optional value.
func goReadComment(source []byte, fset *token.FileSet, group *ast.CommentGroup) (*string, error) {
	if group == nil {
		return nil, nil
	}
	start := fset.PositionFor(group.Pos(), false).Offset
	last := group.List[len(group.List)-1]
	lastStart := fset.PositionFor(last.Slash, false).Offset
	if start < 0 || lastStart < start || lastStart >= len(source) {
		return nil, fmt.Errorf("read Go comment: invalid start offsets %d and %d for %d bytes", start, lastStart, len(source))
	}
	var end int
	if strings.HasPrefix(string(source[lastStart:]), "//") {
		if newline := bytes.IndexByte(source[lastStart:], '\n'); newline >= 0 {
			end = lastStart + newline
		} else {
			end = len(source)
		}
	} else if close := bytes.Index(source[lastStart:], []byte("*/")); close >= 0 {
		end = lastStart + close + 2
	} else {
		return nil, fmt.Errorf("read Go comment: unterminated block comment at byte %d", lastStart)
	}
	if start < 0 || end < start || end > len(source) {
		return nil, fmt.Errorf("read Go comment: invalid byte span [%d,%d) for %d bytes", start, end, len(source))
	}
	text := string(source[start:end])
	return &text, nil
}

func goReadImports(file *ast.File) ([]backend.ReadImport, error) {
	imports := make([]backend.ReadImport, 0, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("unquote Go import path %s: %w", spec.Path.Value, err)
		}
		readImport := backend.ReadImport{Path: path}
		if spec.Name != nil {
			name := spec.Name.Name
			readImport.Name = &name
		}
		imports = append(imports, readImport)
	}
	return imports, nil
}

func goReadSpan(source []byte, fset *token.FileSet, start, end token.Pos) (string, error) {
	startOffset := fset.PositionFor(start, false).Offset
	endOffset := fset.PositionFor(end, false).Offset
	if startOffset < 0 || endOffset < startOffset || endOffset > len(source) {
		return "", fmt.Errorf("read Go source span: invalid byte span [%d,%d) for %d bytes", startOffset, endOffset, len(source))
	}
	return string(source[startOffset:endOffset]), nil
}

func goEmbeddedBaseIdent(expression ast.Expr) *ast.Ident {
	for {
		switch typed := expression.(type) {
		case *ast.StarExpr:
			expression = typed.X
		case *ast.IndexExpr:
			expression = typed.X
		case *ast.IndexListExpr:
			expression = typed.X
		case *ast.Ident:
			return typed
		case *ast.SelectorExpr:
			return typed.Sel
		default:
			return nil
		}
	}
}
