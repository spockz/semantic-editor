// Package astedit resolves statement and expression slots within one function because source text alone cannot distinguish overlapping syntax spans.
package astedit

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func isDirectOwnerStatementCandidate(candidate constructCandidate) bool {
	return candidate.class == "statement" && !strings.Contains(candidate.path, ".")
}

func replaceStatementOrExpression(ctx context.Context, filePath, ownerName, discriminator, source string, opts ConstructOptions) (string, error) {
	if strings.TrimSpace(discriminator) == "" {
		return "", fmt.Errorf("discriminator is required for statement/expression replacement")
	}
	target, err := readStatementExpressionTarget(ctx, filePath, ownerName)
	if err != nil {
		return "", err
	}
	var candidateSet ConstructCandidateSet
	candidateSetInspected := false
	if opts.ExpectedSelectionRevision != "" {
		candidateSet, err = inspectConstructCandidatesFromSource(target.cleanPath, target.content, ownerName, ConstructStatementOrExpr, discriminator)
		if err != nil {
			return "", err
		}
		candidateSetInspected = true
		if opts.ExpectedSelectionRevision != candidateSet.SelectionRevision {
			return "", selectionError("selection_revision_mismatch", candidateSet, nil, opts.ExpectedSelectionRevision)
		}
	}
	selected, selectionErr := selectStatementExpressionCandidate(target, discriminator, opts.ConstructPath)
	if selectionErr != nil {
		status := constructSelectionErrorStatus(selectionErr)
		if status == "" {
			return "", selectionErr
		}
		if !candidateSetInspected {
			var inspectErr error
			candidateSet, inspectErr = inspectConstructCandidatesFromSource(target.cleanPath, target.content, ownerName, ConstructStatementOrExpr, discriminator)
			if inspectErr != nil {
				return "", selectionErr
			}
		}
		return "", selectionError(status, candidateSet, selectionErr, "")
	}
	replacement, err := parseSpanReplacement(selected.class, source)
	if err != nil {
		return "", err
	}
	return publishStatementExpressionReplacement(ctx, target, ownerName, source, selected, replacement, opts.AutoOrganizeImports)
}

type statementExpressionTarget struct {
	cleanPath  string
	ownerName  string
	content    []byte
	fset       *token.FileSet
	candidates []constructCandidate
	typeFacts  constructTypeFacts
}

func readStatementExpressionTarget(ctx context.Context, filePath, ownerName string) (*statementExpressionTarget, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	owner, err := findConstructOwner(file, ownerName)
	if err != nil {
		return nil, err
	}
	candidates, typeFacts := collectStatementExpressionCandidates(fset, content, file, owner)
	return &statementExpressionTarget{cleanPath: cleanPath, ownerName: ownerName, content: content, fset: fset, candidates: candidates, typeFacts: typeFacts}, nil
}

func selectStatementExpressionCandidate(target *statementExpressionTarget, discriminator, constructPath string) (constructCandidate, error) {
	fset := target.fset
	ownerName := target.ownerName
	matches := make([]constructCandidate, 0, 1)
	unresolved := make([]constructCandidate, 0)
	for _, candidate := range target.candidates {
		if candidate.key == normalizeConstructSelector(discriminator) {
			if candidate.class == unresolvedExpressionClass {
				unresolved = append(unresolved, candidate)
			} else {
				matches = append(matches, candidate)
			}
		}
	}
	if constructPath != "" {
		selected := make([]constructCandidate, 0, 1)
		for _, candidate := range matches {
			if candidate.path == constructPath {
				selected = append(selected, candidate)
			}
		}
		if len(selected) == 0 {
			for _, candidate := range unresolved {
				if candidate.path == constructPath {
					return constructCandidate{}, unresolvedExpressionError(fset, discriminator, ownerName, target.typeFacts, []constructCandidate{candidate})
				}
			}
			return constructCandidate{}, selectionOutcome("invalid_path", fmt.Errorf("construct_path %q does not select statement/expression %q in %s; candidates:\n%s", constructPath, discriminator, ownerName, formatSpanCandidates(fset, matches)))
		}
		matches = selected
		unresolved = nil
	}
	if len(matches) == 0 {
		if len(unresolved) > 0 {
			return constructCandidate{}, unresolvedExpressionError(fset, discriminator, ownerName, target.typeFacts, unresolved)
		}
		return constructCandidate{}, selectionOutcome("not_found", fmt.Errorf("statement/expression %q not found in %s; candidates:\n%s", discriminator, ownerName, formatSpanCandidates(fset, target.candidates)))
	}
	if len(unresolved) > 0 {
		return constructCandidate{}, unresolvedExpressionError(fset, discriminator, ownerName, target.typeFacts, unresolved)
	}
	if len(matches) > 1 {
		return constructCandidate{}, selectionOutcome("ambiguous", fmt.Errorf("ambiguous statement/expression %q in %s; set construct_path to one of:\n%s", discriminator, ownerName, formatSpanCandidates(fset, matches)))
	}
	return matches[0], nil
}

func publishStatementExpressionReplacement(ctx context.Context, target *statementExpressionTarget, ownerName, source string, selected constructCandidate, replacement string, organizeImports bool) (string, error) {
	cleanPath := target.cleanPath
	content := target.content
	fset := target.fset
	start, end := fset.Position(selected.node.Pos()).Offset, fset.Position(selected.node.End()).Offset
	if start < 0 || end > len(content) || start >= end {
		return "", fmt.Errorf("selected source span is invalid")
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
	if organizeImports {
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

func collectStatementExpressionCandidates(fset *token.FileSet, source []byte, file *ast.File, owner *ast.FuncDecl) ([]constructCandidate, constructTypeFacts) {
	var candidates []constructCandidate
	info := collectExpressionTypes(fset, file, owner)
	var walkBlock func(*ast.BlockStmt, string)
	var walkStmt func(ast.Stmt, string)
	addExprRoot := makeSpanExpressionAdder(fset, &candidates, info)
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
			candidates = append(candidates, constructCandidate{node: stmt, path: path, key: normalizeConstructSelector(nodeText(fset, stmt)), class: "statement"})
		}
		addStatementExpressionRoots(stmt, path, addExprRoot)
		walkConstructStatementChildren(stmt, path, walkBlock, walkStmt, addExprRoot)
	}
	if owner != nil {
		walkBlock(owner.Body, "")
	}
	return candidates, info
}

// Partial type information remains useful, while a missing fact stays ambiguous.
type constructTypeFacts struct {
	info *types.Info
	// checkErr records whether this snapshot is partial.
	checkErr    error
	diagnostics []error
}

func collectExpressionTypes(fset *token.FileSet, file *ast.File, owner *ast.FuncDecl) constructTypeFacts {
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	if file == nil || owner == nil || !needsExpressionTypeInfo(owner.Body) {
		return constructTypeFacts{info: info}
	}
	var diagnostics []error
	config := types.Config{Error: func(err error) { diagnostics = append(diagnostics, err) }}
	_, checkErr := config.Check(file.Name.Name, fset, []*ast.File{file}, info)
	return constructTypeFacts{info: info, checkErr: checkErr, diagnostics: diagnostics}
}

func needsExpressionTypeInfo(body *ast.BlockStmt) bool {
	needsInfo := false
	ast.Inspect(body, func(node ast.Node) bool {
		if needsInfo {
			return false
		}
		switch value := node.(type) {
		case *ast.IndexExpr:
			needsInfo = true
		case *ast.CallExpr:
			if ident, ok := typeConstructorIdent(value); ok && (ident.Name == "make" || ident.Name == "new") {
				needsInfo = true
			}
		}
		return true
	})
	return needsInfo
}

func isKnownType(facts constructTypeFacts, expr ast.Expr) bool {
	if facts.info == nil {
		return false
	}
	value, ok := facts.info.Types[expr]
	return ok && value.IsType()
}

func isKnownValue(facts constructTypeFacts, expr ast.Expr) bool {
	if facts.info == nil {
		return false
	}
	value, ok := facts.info.Types[expr]
	return ok && value.IsValue()
}

func isSyntacticType(expr ast.Expr) bool {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	switch expr.(type) {
	case *ast.ArrayType, *ast.MapType, *ast.FuncType, *ast.ChanType, *ast.StructType, *ast.InterfaceType:
		return true
	default:
		return false
	}
}

func makeSpanExpressionAdder(fset *token.FileSet, candidates *[]constructCandidate, info constructTypeFacts) func(ast.Expr, string) {
	var add func(ast.Expr, string)
	add = func(root ast.Expr, path string) {
		if root == nil {
			return
		}
		if !isSyntacticType(root) && !isKnownType(info, root) {
			if _, excluded := root.(*ast.KeyValueExpr); !excluded {
				if _, excluded := root.(*ast.Ellipsis); !excluded {
					*candidates = append(*candidates, constructCandidate{node: root, path: path, key: normalizeConstructSelector(nodeText(fset, root)), class: "expression"})
				}
			}
		}
		if indexed, ok := root.(*ast.IndexExpr); ok && !isKnownType(info, indexed.Index) && !isKnownValue(info, indexed.Index) {
			indexPath := childLoopPath(path, 1)
			*candidates = append(*candidates, constructCandidate{
				node: indexed.Index, path: indexPath, key: normalizeConstructSelector(nodeText(fset, indexed.Index)), class: unresolvedExpressionClass,
			})
		}
		for index, child := range expressionChildren(root, info) {
			add(child, childLoopPath(path, index))
		}
	}
	return add
}

const unresolvedExpressionClass = "unresolved expression"

func unresolvedExpressionError(fset *token.FileSet, selector, ownerName string, facts constructTypeFacts, candidates []constructCandidate) error {
	reason := "type analysis did not prove the selected index operand is a runtime value"
	if len(facts.diagnostics) > 0 {
		reason = errors.Join(facts.diagnostics...).Error()
	} else if facts.checkErr != nil {
		reason = facts.checkErr.Error()
	}
	return fmt.Errorf("cannot classify %q in %s as a runtime expression because type analysis is incomplete: %s; candidates:\n%s", selector, ownerName, reason, formatSpanCandidates(fset, candidates))
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
	case *ast.TypeSwitchStmt:
		addStatementExpressionRoots(value.Init, path+".init", addExprRoot)
		addStatementExpressionRoots(value.Assign, path+".assign", addExprRoot)
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
				clausePath := childLoopPath(path, index)
				addStatementExpressionRoots(clause.Comm, clausePath+".comm", addExprRoot)
				walkBlock(&ast.BlockStmt{List: clause.Body}, clausePath)
			}
		}
	case *ast.LabeledStmt:
		walkStmt(value.Stmt, childLoopPath(path, 0))
	}
}

func expressionChildren(expr ast.Expr, info constructTypeFacts) []ast.Expr {
	switch value := expr.(type) {
	case *ast.BinaryExpr:
		return []ast.Expr{value.X, value.Y}
	case *ast.CallExpr:
		if isBuiltinTypeConstructor(info, value) {
			ident, _ := typeConstructorIdent(value)
			switch ident.Name {
			case "make":
				if len(value.Args) > 0 {
					return value.Args[1:]
				}
			case "new":
				if len(value.Args) > 0 && !isKnownValue(info, value.Args[0]) {
					return value.Args[1:]
				}
			}
		}
		return value.Args
	case *ast.SelectorExpr:
		return []ast.Expr{value.X}
	case *ast.IndexExpr:
		children := []ast.Expr{value.X}
		if isKnownValue(info, value.Index) {
			children = append(children, value.Index)
		}
		return children
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

func isBuiltinTypeConstructor(info constructTypeFacts, call *ast.CallExpr) bool {
	ident, ok := typeConstructorIdent(call)
	if !ok || (ident.Name != "make" && ident.Name != "new") {
		return false
	}
	if info.info != nil {
		if _, ok := info.info.Uses[ident].(*types.Builtin); ok {
			return true
		}
		if info.info.Uses[ident] != nil {
			return false
		}
	}
	return true
}

func typeConstructorIdent(call *ast.CallExpr) (*ast.Ident, bool) {
	callee := call.Fun
	for {
		parenthesized, ok := callee.(*ast.ParenExpr)
		if !ok {
			break
		}
		callee = parenthesized.X
	}
	ident, ok := callee.(*ast.Ident)
	return ident, ok
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
		fset := token.NewFileSet()
		expr, err := parser.ParseExprFrom(fset, "snippet.go", trimmed, parser.ParseComments)
		if err != nil {
			return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
		}
		if isSyntacticType(expr) {
			return "", fmt.Errorf("expression replacement cannot be a type expression")
		}
		if _, isCall := expr.(*ast.CallExpr); isCall {
			if hasTrailingLineComment(trimmed, fset.Position(expr.End()).Offset) {
				return "", fmt.Errorf("call expression replacement cannot safely preserve a trailing line comment")
			}
			return trimmed, nil
		}
		return "(" + trimmed + ")", nil
	}
	endOffset, err := parseSingleDirectStatement(trimmed)
	if err != nil {
		return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if hasTrailingLineComment(trimmed, endOffset) {
		return "", fmt.Errorf("statement replacement cannot safely preserve a trailing line comment")
	}
	return trimmed, nil
}

func hasTrailingLineComment(source string, endOffset int) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("comment-scan.go", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, []byte(source), nil, scanner.ScanComments)
	for {
		pos, tok, lit := scan.Scan()
		if tok == token.EOF {
			return false
		}
		if tok == token.COMMENT && strings.HasPrefix(lit, "//") && file.Offset(pos) >= endOffset {
			return true
		}
	}
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
