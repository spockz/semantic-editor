// Package astedit resolves statement and expression slots within one function because source text alone cannot distinguish overlapping syntax spans.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

func isDirectOwnerStatementCandidate(candidate constructCandidate) bool {
	return candidate.class == "statement" && !strings.Contains(candidate.path, ".")
}

func replaceStatementOrExpression(ctx context.Context, filePath, ownerName, discriminator, source string, opts ConstructOptions) (string, error) {
	if strings.TrimSpace(discriminator) == "" {
		return "", fmt.Errorf("discriminator is required for statement/expression replacement")
	}
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	owner, err := findConstructOwner(file, ownerName)
	if err != nil {
		return "", err
	}
	candidates := collectStatementExpressionCandidates(fset, content, owner.Body)
	matches := make([]constructCandidate, 0, 1)
	for _, candidate := range candidates {
		if candidate.key == normalizeConstructSelector(discriminator) {
			matches = append(matches, candidate)
		}
	}
	if opts.ConstructPath != "" {
		selected := make([]constructCandidate, 0, 1)
		for _, candidate := range matches {
			if candidate.path == opts.ConstructPath {
				selected = append(selected, candidate)
			}
		}
		if len(selected) == 0 {
			return "", fmt.Errorf("construct_path %q does not select statement/expression %q in %s; candidates:\n%s", opts.ConstructPath, discriminator, ownerName, formatSpanCandidates(fset, matches))
		}
		matches = selected
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("statement/expression %q not found in %s; candidates:\n%s", discriminator, ownerName, formatSpanCandidates(fset, candidates))
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous statement/expression %q in %s; set construct_path to one of:\n%s", discriminator, ownerName, formatSpanCandidates(fset, matches))
	}
	selected := matches[0]
	start, end := fset.Position(selected.node.Pos()).Offset, fset.Position(selected.node.End()).Offset
	if start < 0 || end > len(content) || start >= end {
		return "", fmt.Errorf("selected source span is invalid")
	}
	replacement, err := parseSpanReplacement(selected.class, source)
	if err != nil {
		return "", err
	}
	updated := replaceBytes(content, start, end, []byte(replacement))
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	formatted, err = cleanupMoveImports(ctx, cleanPath, formatted)
	if err != nil {
		return "", fmt.Errorf("clean imports after construct replacement: %w", err)
	}
	formatted, err = format.Source(formatted)
	if err != nil {
		return "", fmt.Errorf("format import-cleaned source: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if err := pipeline.WriteAtomicContext(ctx, cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	}
	return UnifiedDiff(ownerName, strings.TrimSpace(string(content[start:end])), strings.TrimSpace(replacement)), nil
}

func findConstructOwner(file *ast.File, ownerName string) (*ast.FuncDecl, error) {
	receiver, name, err := symbol.ParseIdentifier(ownerName)
	if err != nil {
		return nil, &symbol.SymbolError{Op: replaceConstructOperation, Symbol: ownerName, Err: err}
	}
	var matches []*ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == receiver {
			matches = append(matches, fn)
		}
	}
	if len(matches) == 0 {
		return nil, &symbol.SymbolError{Op: replaceConstructOperation, Symbol: ownerName, Err: symbol.ErrNotFound}
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("function selector %q is ambiguous", ownerName)
	}
	if matches[0].Body == nil {
		return nil, &symbol.SymbolError{Op: replaceConstructOperation, Symbol: ownerName, Err: ErrNoBody}
	}
	return matches[0], nil
}

func collectStatementExpressionCandidates(fset *token.FileSet, source []byte, body *ast.BlockStmt) []constructCandidate {
	var candidates []constructCandidate
	var walkBlock func(*ast.BlockStmt, string)
	var walkStmt func(ast.Stmt, string)
	addExprRoot := makeSpanExpressionAdder(fset, &candidates)
	walkBlock = func(block *ast.BlockStmt, prefix string) {
		if block == nil {
			return
		}
		for index, stmt := range block.List {
			walkStmt(stmt, childLoopPath(prefix, index))
		}
	}
	walkStmt = func(stmt ast.Stmt, path string) {
		start, end := fset.Position(stmt.Pos()).Offset, fset.Position(stmt.End()).Offset
		if start >= 0 && end <= len(source) {
			candidates = append(candidates, constructCandidate{node: stmt, path: path, key: strings.TrimSpace(nodeText(fset, stmt)), class: "statement"})
		}
		addStatementExpressionRoots(stmt, path, addExprRoot)
		walkConstructStatementChildren(stmt, path, walkBlock, walkStmt, addExprRoot)
	}
	walkBlock(body, "")
	return candidates
}

func makeSpanExpressionAdder(fset *token.FileSet, candidates *[]constructCandidate) func(ast.Expr, string) {
	var add func(ast.Expr, string)
	add = func(root ast.Expr, path string) {
		if root == nil {
			return
		}
		if _, excluded := root.(*ast.KeyValueExpr); !excluded {
			if _, excluded := root.(*ast.Ellipsis); !excluded {
				*candidates = append(*candidates, constructCandidate{node: root, path: path, key: strings.TrimSpace(nodeText(fset, root)), class: "expression"})
			}
		}
		for index, child := range expressionChildren(root) {
			add(child, childLoopPath(path, index))
		}
	}
	return add
}

func addStatementExpressionRoots(stmt ast.Stmt, path string, addExprRoot func(ast.Expr, string)) {
	if stmt == nil {
		return
	}
	var roots []ast.Expr
	switch value := stmt.(type) {
	case *ast.ExprStmt:
		roots = append(roots, value.X)
	case *ast.AssignStmt:
		roots = append(roots, value.Rhs...)
		for index, lhs := range value.Lhs {
			if slot, ok := lhs.(*ast.IndexExpr); ok {
				addExprRoot(slot.Index, path+".l"+strconv.Itoa(index))
			}
		}
	case *ast.IncDecStmt:
		if slot, ok := value.X.(*ast.IndexExpr); ok {
			addExprRoot(slot.Index, path+".l0")
		}
	case *ast.ReturnStmt:
		roots = append(roots, value.Results...)
	case *ast.SendStmt:
		roots = append(roots, value.Chan, value.Value)
	case *ast.GoStmt:
		roots = append(roots, value.Call)
	case *ast.DeferStmt:
		roots = append(roots, value.Call)
	case *ast.IfStmt:
		roots = append(roots, value.Cond)
		addStatementExpressionRoots(value.Init, path+".init", addExprRoot)
	case *ast.ForStmt:
		roots = append(roots, value.Cond)
		addStatementExpressionRoots(value.Init, path+".init", addExprRoot)
		addStatementExpressionRoots(value.Post, path+".post", addExprRoot)
	case *ast.RangeStmt:
		roots = append(roots, value.X)
	case *ast.SwitchStmt:
		roots = append(roots, value.Tag)
		addStatementExpressionRoots(value.Init, path+".init", addExprRoot)
	case *ast.DeclStmt:
		if decl, ok := value.Decl.(*ast.GenDecl); ok {
			for _, spec := range decl.Specs {
				if values, ok := spec.(*ast.ValueSpec); ok {
					roots = append(roots, values.Values...)
				}
			}
		}
	}
	addExpressionRoots(roots, path, addExprRoot)
}

func addExpressionRoots(roots []ast.Expr, path string, addExprRoot func(ast.Expr, string)) {
	for index, root := range roots {
		addExprRoot(root, path+".e"+strconv.Itoa(index))
	}
}

func walkConstructStatementChildren(stmt ast.Stmt, path string, walkBlock func(*ast.BlockStmt, string), walkStmt func(ast.Stmt, string), addExprRoot func(ast.Expr, string)) {
	switch value := stmt.(type) {
	case *ast.BlockStmt:
		walkBlock(value, childLoopPath(path, 0))
	case *ast.IfStmt:
		walkBlock(value.Body, childLoopPath(path, 0))
		if value.Else != nil {
			walkStmt(value.Else, childLoopPath(path, 1))
		}
	case *ast.ForStmt:
		walkBlock(value.Body, childLoopPath(path, 0))
	case *ast.RangeStmt:
		walkBlock(value.Body, childLoopPath(path, 0))
	case *ast.SwitchStmt:
		for index, item := range value.Body.List {
			clause, ok := item.(*ast.CaseClause)
			if !ok {
				continue
			}
			clausePath := childLoopPath(path, index)
			addExpressionRoots(clause.List, clausePath, addExprRoot)
			walkBlock(&ast.BlockStmt{List: clause.Body}, clausePath)
		}
	case *ast.TypeSwitchStmt:
		for index, item := range value.Body.List {
			if clause, ok := item.(*ast.CaseClause); ok {
				walkBlock(&ast.BlockStmt{List: clause.Body}, childLoopPath(path, index))
			}
		}
	case *ast.SelectStmt:
		for index, item := range value.Body.List {
			if clause, ok := item.(*ast.CommClause); ok {
				walkBlock(&ast.BlockStmt{List: clause.Body}, childLoopPath(path, index))
			}
		}
	case *ast.LabeledStmt:
		walkStmt(value.Stmt, childLoopPath(path, 0))
	}
}

func expressionChildren(expr ast.Expr) []ast.Expr {
	switch value := expr.(type) {
	case *ast.BinaryExpr:
		return []ast.Expr{value.X, value.Y}
	case *ast.CallExpr:
		return value.Args
	case *ast.SelectorExpr:
		return []ast.Expr{value.X}
	case *ast.IndexExpr:
		return []ast.Expr{value.X}
	case *ast.IndexListExpr:
		return []ast.Expr{value.X}
	case *ast.SliceExpr:
		children := []ast.Expr{value.X}
		if value.Low != nil {
			children = append(children, value.Low)
		}
		if value.High != nil {
			children = append(children, value.High)
		}
		if value.Max != nil {
			children = append(children, value.Max)
		}
		return children
	case *ast.TypeAssertExpr:
		return []ast.Expr{value.X}
	case *ast.StarExpr:
		return []ast.Expr{value.X}
	case *ast.UnaryExpr:
		return []ast.Expr{value.X}
	case *ast.ParenExpr:
		return []ast.Expr{value.X}
	case *ast.KeyValueExpr:
		return []ast.Expr{value.Value}
	case *ast.CompositeLit:
		return value.Elts
	case *ast.FuncLit:
		return nil
	default:
		return nil
	}
}

func normalizeConstructSelector(source string) string {
	trimmed := strings.TrimSpace(source)
	if expr, err := parser.ParseExpr(trimmed); err == nil {
		var buf strings.Builder
		if format.Node(&buf, token.NewFileSet(), expr) == nil {
			return buf.String()
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), "selector.go", []byte("package p\nfunc _(){\n"+trimmed+"\n}"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return trimmed
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Body.List) != 1 {
		return trimmed
	}
	var buf strings.Builder
	if format.Node(&buf, token.NewFileSet(), fn.Body.List[0]) == nil {
		return buf.String()
	}
	return trimmed
}

func parseSpanReplacement(class, source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", ErrEmptySnippet
	}
	if class == "expression" {
		expr, err := parser.ParseExpr(trimmed)
		if err != nil {
			return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
		}
		if _, isCall := expr.(*ast.CallExpr); isCall {
			return trimmed, nil
		}
		return "(" + trimmed + ")", nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), "statement.go", []byte("package p\nfunc _(){\n"+trimmed+"\n}"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Body.List) != 1 {
		return "", fmt.Errorf("statement replacement must contain exactly one statement or block")
	}
	node := fn.Body.List[0]
	var buf strings.Builder
	if err := format.Node(&buf, token.NewFileSet(), node); err != nil {
		return "", fmt.Errorf("format replacement statement: %w", err)
	}
	return buf.String(), nil
}

func formatSpanCandidates(fset *token.FileSet, candidates []constructCandidate) string {
	lines := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.class == "" || candidate.key == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("construct_path %s (%s) at %s: %s", candidate.path, candidate.class, fset.Position(candidate.node.Pos()), strings.TrimSpace(candidate.key)))
	}
	if len(lines) == 0 {
		return "(no candidates found)"
	}
	return strings.Join(lines, "\n")
}
