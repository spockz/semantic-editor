// Package astedit contains bounded source transformations that can prove local semantic safety.
// InlineVariable keeps evaluation unchanged by accepting literal initializers and immediate expression uses only.
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

	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

// InlineVariable inlines one local literal binding at its immediate use site.
func InlineVariable(ctx context.Context, filePath, function, name string) (string, error) {
	cleanPath := filepath.Clean(filePath)
	ctx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, cleanPath)
	if err != nil {
		return "", err
	}
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	if !token.IsIdentifier(name) || name == "_" {
		return "", fmt.Errorf("inline variable target %q is not a Go identifier", name)
	}
	recv, functionName, err := symbol.ParseIdentifier(function)
	if err != nil {
		return "", fmt.Errorf("parse containing function %q: %w", function, err)
	}
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	fn, err := findInlineFunction(fileNode, recv, functionName)
	if err != nil {
		return "", err
	}
	target, err := findInlineTarget(fn, name)
	if err != nil {
		return "", err
	}
	if target.explicitType {
		return "", fmt.Errorf("cannot inline %q: explicitly typed declarations require preserving their declared type", name)
	}
	if target.value == nil || !literalExpression(target.value) {
		return "", fmt.Errorf("cannot inline %q: initializer must be a literal expression", name)
	}
	if !target.constant {
		typeName, ok := defaultTypeName(target.value)
		if !ok {
			return "", fmt.Errorf("cannot inline %q: initializer literal has no supported default type", name)
		}
		shadowed, err := conversionNameShadowed(ctx, cleanPath, fileNode, fn, typeName)
		if err != nil {
			return "", fmt.Errorf("cannot inline %q: check default type name %s bindings: %w", name, typeName, err)
		}
		if shadowed {
			return "", fmt.Errorf("cannot inline %q: default type name %s is shadowed by a local or package binding or import", name, typeName)
		}
	}
	positions, err := inlineUsePositions(fn, target, name)
	if err != nil {
		return "", err
	}
	formatted, err := inlineSource(content, fset, fileNode, target, positions, name)
	if err != nil {
		return "", err
	}
	current, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("recheck inline source before publication: %w", err)
	}
	if !bytes.Equal(current, content) {
		return "", fmt.Errorf("cannot inline %q: source changed during planning", name)
	}
	if err := validateExtractionCandidate(ctx, cleanPath, formatted); err != nil {
		return "", fmt.Errorf("validate inlined package: %w", err)
	}
	buffer := sourcebuf.FromContext(ctx)
	if err := buffer.Apply(ctx, []sourcebuf.Change{{Path: cleanPath, Before: content, After: formatted}}); err != nil {
		return "", fmt.Errorf("apply inlined source after compiler preflight: %w", err)
	}
	current, err = sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read inlined file: %w", err)
	}
	if standalone {
		if err := ownedBuffer.Flush(ctx); err != nil {
			return "", fmt.Errorf("publish inlined source: %w", err)
		}
	}
	return UnifiedDiff(cleanPath, string(content), string(current)), nil
}

type inlineTarget struct {
	stmt         ast.Stmt
	block        *ast.BlockStmt
	index        int
	value        ast.Expr
	constant     bool
	explicitType bool
}

func inlineQualifiedFunction(receiver, name string) string {
	if receiver == "" {
		return name
	}
	return receiver + "." + name
}

func findInlineFunction(fileNode *ast.File, receiver, name string) (*ast.FuncDecl, error) {
	var found *ast.FuncDecl
	for _, decl := range fileNode.Decls {
		candidate, ok := decl.(*ast.FuncDecl)
		if !ok || candidate.Name.Name != name || extractReceiverName(candidate.Recv) != receiver {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("containing function %q is ambiguous", inlineQualifiedFunction(receiver, name))
		}
		found = candidate
	}
	if found == nil || found.Body == nil {
		return nil, fmt.Errorf("containing function %q not found or has no body", inlineQualifiedFunction(receiver, name))
	}
	return found, nil
}

func findInlineTarget(fn *ast.FuncDecl, name string) (inlineTarget, error) {
	var targets []inlineTarget
	var walkBlock func(*ast.BlockStmt)
	walkBlock = func(block *ast.BlockStmt) {
		if block == nil {
			return
		}
		for i, stmt := range block.List {
			var value ast.Expr
			var names []*ast.Ident
			var constant, explicitType bool
			switch node := stmt.(type) {
			case *ast.AssignStmt:
				if node.Tok == token.DEFINE && len(node.Lhs) == 1 && len(node.Rhs) == 1 {
					if id, ok := node.Lhs[0].(*ast.Ident); ok {
						names, value = []*ast.Ident{id}, node.Rhs[0]
					}
				}
			case *ast.DeclStmt:
				gen, ok := node.Decl.(*ast.GenDecl)
				if ok && (gen.Tok == token.VAR || gen.Tok == token.CONST) && len(gen.Specs) == 1 {
					if spec, ok := gen.Specs[0].(*ast.ValueSpec); ok && len(spec.Names) == 1 && len(spec.Values) == 1 {
						names, value = spec.Names, spec.Values[0]
						constant, explicitType = gen.Tok == token.CONST, spec.Type != nil
					}
				}
			}
			for _, id := range names {
				if id.Name == name {
					targets = append(targets, inlineTarget{stmt: stmt, block: block, index: i, value: value, constant: constant, explicitType: explicitType})
				}
			}
			ast.Inspect(stmt, func(node ast.Node) bool {
				if _, ok := node.(*ast.FuncLit); ok {
					return false
				}
				if nested, ok := node.(*ast.BlockStmt); ok && nested != block {
					walkBlock(nested)
					return false
				}
				return true
			})
		}
	}
	walkBlock(fn.Body)
	switch len(targets) {
	case 0:
		return inlineTarget{}, fmt.Errorf("local variable %q not found in %s", name, fn.Name.Name)
	case 1:
		return targets[0], nil
	default:
		return inlineTarget{}, fmt.Errorf("local variable %q is ambiguous in %s", name, fn.Name.Name)
	}
}

func inlineUsePositions(fn *ast.FuncDecl, target inlineTarget, name string) ([]token.Pos, error) {
	if target.index+1 >= len(target.block.List) {
		return nil, fmt.Errorf("cannot inline %q: declaration has no immediate following statement", name)
	}
	positions, direct := directExpressionUses(target.block.List[target.index+1], name)
	if !direct || len(positions) == 0 {
		return nil, fmt.Errorf("cannot inline %q: uses must be direct arguments of one call or direct return values in the next statement", name)
	}
	identifiers := 0
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == name {
			identifiers++
		}
		return true
	})
	if hasFunctionBinding(fn, name) || identifiers != len(positions)+1 {
		return nil, fmt.Errorf("cannot inline %q: additional references, assignments, or shadowing make the binding ambiguous", name)
	}
	return positions, nil
}

func hasFunctionBinding(fn *ast.FuncDecl, name string) bool {
	for _, list := range []*ast.FieldList{fn.Recv, fn.Type.Params, fn.Type.Results} {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, ident := range field.Names {
				if ident.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func functionDeclaresName(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.AssignStmt:
			if item.Tok == token.DEFINE {
				for _, expr := range item.Lhs {
					if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
						found = true
					}
				}
			}
		case *ast.ValueSpec:
			for _, ident := range item.Names {
				if ident.Name == name {
					found = true
				}
			}
		case *ast.TypeSpec:
			if item.Name.Name == name {
				found = true
			}
		case *ast.RangeStmt:
			for _, expr := range []ast.Expr{item.Key, item.Value} {
				if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
					found = true
				}
			}
		case *ast.Field:
			for _, ident := range item.Names {
				if ident.Name == name {
					found = true
				}
			}
		}
		return !found
	})
	return found
}

func inlineSource(content []byte, fset *token.FileSet, fileNode *ast.File, target inlineTarget, positions []token.Pos, name string) ([]byte, error) {
	start, end, err := standaloneStatementRange(content, fset, target.stmt, fileNode.Comments)
	if err != nil {
		return nil, fmt.Errorf("cannot inline %q: %w", name, err)
	}
	valueText, err := inlineValueText(content, fset, target.value, target.constant)
	if err != nil {
		return nil, fmt.Errorf("cannot inline %q: %w", name, err)
	}
	type edit struct {
		start int
		end   int
		text  []byte
	}
	edits := make([]edit, 0, len(positions))
	for _, pos := range positions {
		offset := fset.Position(pos).Offset
		edits = append(edits, edit{start: offset, end: offset + len(name), text: []byte(valueText)})
	}
	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			if edits[j].start > edits[i].start {
				edits[i], edits[j] = edits[j], edits[i]
			}
		}
	}
	updated := append([]byte(nil), content...)
	for _, item := range edits {
		updated = append(updated[:item.start], append(item.text, updated[item.end:]...)...)
	}
	updated = append(updated[:start], updated[end:]...)
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, &SyntaxError{File: fset.Position(target.stmt.Pos()).Filename, Cause: err, Err: ErrSyntax}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), fset.Position(target.stmt.Pos()).Filename, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, &SyntaxError{File: fset.Position(target.stmt.Pos()).Filename, Cause: err, Err: ErrSyntax}
	}
	return formatted, nil
}

func literalExpression(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return literalExpression(node.X)
	default:
		return false
	}
}

func directExpressionUses(stmt ast.Stmt, name string) ([]token.Pos, bool) {
	var positions []token.Pos
	switch node := stmt.(type) {
	case *ast.ExprStmt:
		call, ok := node.X.(*ast.CallExpr)
		if !ok {
			return nil, false
		}
		for _, arg := range call.Args {
			if id, ok := arg.(*ast.Ident); ok && id.Name == name {
				positions = append(positions, id.Pos())
			}
		}
	case *ast.ReturnStmt:
		for _, result := range node.Results {
			if id, ok := result.(*ast.Ident); ok && id.Name == name {
				positions = append(positions, id.Pos())
			}
		}
	default:
		return nil, false
	}
	if len(positions) == 0 {
		return nil, false
	}
	return positions, true
}

func inlineValueText(content []byte, fset *token.FileSet, expr ast.Expr, constant bool) (string, error) {
	start, end := fset.Position(expr.Pos()).Offset, fset.Position(expr.End()).Offset
	text := strings.TrimSpace(string(content[start:end]))
	if constant {
		return text, nil
	}
	typeName, ok := defaultTypeName(expr)
	if !ok {
		return "", fmt.Errorf("initializer literal has no supported default type")
	}
	return typeName + "(" + text + ")", nil
}

func defaultTypeName(expr ast.Expr) (string, bool) {
	literal, ok := unparenExpr(expr).(*ast.BasicLit)
	if !ok {
		return "", false
	}
	types := map[token.Token]string{token.INT: "int", token.FLOAT: "float64", token.IMAG: "complex128", token.CHAR: "rune", token.STRING: "string"}
	typeName, ok := types[literal.Kind]
	return typeName, ok
}

func packageDeclaresName(decls []ast.Decl, name string) bool {
	for _, decl := range decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			if node.Recv == nil && node.Name.Name == name {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch item := spec.(type) {
				case *ast.ValueSpec:
					for _, ident := range item.Names {
						if ident.Name == name {
							return true
						}
					}
				case *ast.TypeSpec:
					if item.Name.Name == name {
						return true
					}
				}
			}
		}
	}
	return false
}

func importsShadowConversion(imports []*ast.ImportSpec, name string) bool {
	for _, item := range imports {
		if item.Name == nil || item.Name.Name == "." || item.Name.Name == name {
			return true
		}
	}
	return false
}

func unparenExpr(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

func standaloneStatementRange(content []byte, fset *token.FileSet, stmt ast.Stmt, comments []*ast.CommentGroup) (int, int, error) {
	start := fset.Position(stmt.Pos()).Offset
	end := fset.Position(stmt.End()).Offset
	lineStart := bytes.LastIndexByte(content[:start], '\n') + 1
	lineEnd := len(content)
	if next := bytes.IndexByte(content[end:], '\n'); next >= 0 {
		lineEnd = end + next + 1
	}
	if strings.TrimSpace(string(content[lineStart:start])) != "" || strings.TrimSpace(string(content[end:lineEnd])) != "" {
		return 0, 0, fmt.Errorf("declaration must occupy its own source line")
	}
	for _, group := range comments {
		for _, comment := range group.List {
			commentStart := fset.Position(comment.Pos()).Offset
			commentEnd := fset.Position(comment.End()).Offset
			previousText := content[commentEnd:start]
			previousLineCount := strings.Count(string(previousText), "\n")
			attachedBefore := commentEnd < start && strings.TrimSpace(string(previousText)) == "" && previousLineCount <= 1
			if attachedBefore || (commentStart < start && commentEnd > lineStart) || (commentStart >= end && commentStart < lineEnd) {
				return 0, 0, fmt.Errorf("declaration has an adjacent comment")
			}
		}
	}
	if lineEnd == end && lineEnd < len(content) {
		lineEnd++
	}
	return lineStart, lineEnd, nil
}

func conversionNameShadowed(ctx context.Context, filePath string, fileNode *ast.File, fn *ast.FuncDecl, name string) (bool, error) {
	if hasFunctionBinding(fn, name) || functionDeclaresName(fn, name) || packageDeclaresName(fileNode.Decls, name) || importsShadowConversion(fileNode.Imports, name) {
		return true, nil
	}
	return siblingPackageDeclaresName(ctx, filePath, fileNode.Name.Name, name)
}

func siblingPackageDeclaresName(ctx context.Context, filePath, packageName, name string) (bool, error) {
	entries, err := os.ReadDir(filepath.Dir(filePath))
	if err != nil {
		return false, fmt.Errorf("read package directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || entry.Name() == filepath.Base(filePath) {
			continue
		}
		siblingPath := filepath.Join(filepath.Dir(filePath), entry.Name())
		sibling, parseErr := sourcebuf.ParseFile(ctx, token.NewFileSet(), siblingPath, nil, parser.ParseComments|parser.AllErrors)
		if parseErr != nil {
			return false, fmt.Errorf("parse sibling source %s: %w", entry.Name(), parseErr)
		}
		if sibling == nil || sibling.Name.Name != packageName {
			continue
		}
		if packageDeclaresName(sibling.Decls, name) {
			return true, nil
		}
	}
	return false, nil
}
