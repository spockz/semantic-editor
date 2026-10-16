// Package symbol exposes bounded native Go declaration enumeration for
// suggestion providers while reusing its overlay-aware source snapshot reader.
package symbol

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
)

const (
	maxEnumeratedSymbols     = 4096
	maxEnumeratedFiles       = 256
	maxEnumeratedSourceBytes = 2 << 20
	maxEnumerationTotalBytes = 64 << 20
)

// Enumerate returns bounded declarations from the selected file or workspace.
// It reads the current source snapshot, including overlays, and caps work at
// 256 files, 64 MiB total source, 2 MiB per file, and 4096 declarations.
func Enumerate(ctx context.Context, rootDir, filePath string) ([]*Symbol, error) {
	files, err := resolveTargetFiles(ctx, rootDir, filePath)
	if err != nil {
		return nil, fmt.Errorf("resolve symbol enumeration files: %w", err)
	}
	reader := newSourceSnapshotReader()
	reader.ctx = ctx
	var result []*Symbol
	totalBytes := 0
	for index, filePath := range files {
		if index >= maxEnumeratedFiles || totalBytes >= maxEnumerationTotalBytes {
			break
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, statErr := os.Stat(filePath)
		if statErr == nil && info.Size() > maxEnumeratedSourceBytes {
			continue
		}
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return nil, fmt.Errorf("inspect symbol enumeration file %s: %w", filePath, statErr)
		}
		source, err := reader.read(filePath)
		if err != nil {
			return nil, fmt.Errorf("read symbol enumeration file %s: %w", filePath, err)
		}
		if len(source) > maxEnumeratedSourceBytes || len(source) > maxEnumerationTotalBytes-totalBytes {
			continue
		}
		totalBytes += len(source)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, source, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse symbol enumeration file %s: %w", filePath, err)
		}
		result = appendEnumeratedFile(ctx, result, fset, file, filePath)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(result) >= maxEnumeratedSymbols {
			return result[:maxEnumeratedSymbols], nil
		}
	}
	return result, nil
}

func appendEnumeratedFile(ctx context.Context, result []*Symbol, fset *token.FileSet, file *ast.File, filePath string) []*Symbol {
	appendSymbol := func(name, receiver, kind string, position token.Pos) {
		if name == "" || len(result) >= maxEnumeratedSymbols {
			return
		}
		location := fset.Position(position)
		symbol := &Symbol{Name: name, Receiver: receiver, Kind: kind, File: filePath, Line: location.Line, Column: location.Column, Offset: location.Offset}
		symbol.QualifiedName = symbol.BuildQualifiedName()
		result = append(result, symbol)
	}
	for _, declaration := range file.Decls {
		if ctx.Err() != nil || len(result) >= maxEnumeratedSymbols {
			return result
		}
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			receiver := extractReceiver(declaration.Recv)
			kind := "function"
			if receiver != "" {
				kind = "method"
			}
			appendSymbol(declaration.Name.Name, receiver, kind, declaration.Name.Pos())
		case *ast.GenDecl:
			appendEnumeratedGeneralDeclaration(ctx, declaration, appendSymbol)
		}
	}
	return result
}

func appendEnumeratedGeneralDeclaration(ctx context.Context, declaration *ast.GenDecl, appendSymbol func(string, string, string, token.Pos)) {
	for _, raw := range declaration.Specs {
		if ctx.Err() != nil {
			return
		}
		switch spec := raw.(type) {
		case *ast.TypeSpec:
			appendSymbol(spec.Name.Name, "", "type", spec.Name.Pos())
			enumerateTypeMembers(ctx, spec, appendSymbol)
		case *ast.ValueSpec:
			appendEnumeratedValueSpec(ctx, declaration.Tok, spec, appendSymbol)
		}
	}
}

func appendEnumeratedValueSpec(ctx context.Context, tok token.Token, spec *ast.ValueSpec, appendSymbol func(string, string, string, token.Pos)) {
	var kind string
	switch tok {
	case token.CONST:
		kind = "constant"
	case token.VAR:
		kind = "variable"
	default:
		return
	}
	for _, name := range spec.Names {
		if ctx.Err() != nil {
			return
		}
		appendSymbol(name.Name, "", kind, name.Pos())
	}
}

func enumerateTypeMembers(ctx context.Context, spec *ast.TypeSpec, appendSymbol func(string, string, string, token.Pos)) {
	var fields *ast.FieldList
	switch typeNode := spec.Type.(type) {
	case *ast.StructType:
		fields = typeNode.Fields
	case *ast.InterfaceType:
		fields = typeNode.Methods
	}
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if ctx.Err() != nil {
			return
		}
		kind := "field"
		if _, ok := spec.Type.(*ast.InterfaceType); ok {
			kind = "method"
		}
		for _, name := range field.Names {
			appendSymbol(name.Name, spec.Name.Name, kind, name.Pos())
		}
	}
}
