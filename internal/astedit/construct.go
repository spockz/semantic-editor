// Package astedit contains construct-bounded edits because callers should only
// have to provide the syntax they intend to change.
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
	"slices"
	"strconv"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

// ConstructKind aliases the canonical cross-language syntax taxonomy.
type ConstructKind = backend.ConstructKind

// Go construct kind constants name syntax forms handled by ReplaceConstruct.
const (
	ConstructFunction        = backend.ConstructFunction
	ConstructMethod          = backend.ConstructMethod
	ConstructType            = backend.ConstructType
	ConstructField           = backend.ConstructField
	ConstructConst           = backend.ConstructConst
	ConstructVar             = backend.ConstructVar
	ConstructDecl            = backend.ConstructDecl
	ConstructDeclaration     = backend.ConstructDeclaration
	ConstructStatementOrExpr = backend.ConstructStatementOrExpr
	ConstructLoop            = backend.ConstructLoop
	ConstructIf              = backend.ConstructIf
	ConstructElse            = backend.ConstructElse
	ConstructCase            = backend.ConstructCase
	ConstructSelect          = backend.ConstructSelect
	ConstructDefer           = backend.ConstructDefer
)

// GoConstructKinds lists the construct kinds implemented by the Go handler.
var GoConstructKinds = []ConstructKind{ConstructLoop, ConstructIf, ConstructElse, ConstructCase, ConstructSelect, ConstructDefer}

// GoReplaceConstructKinds lists kinds supported by the whole-construct replacement operation.
var GoReplaceConstructKinds = []ConstructKind{
	ConstructLoop, ConstructIf, ConstructElse, ConstructCase, ConstructSelect, ConstructDefer,
	ConstructFunction, ConstructMethod, ConstructType, ConstructConst, ConstructVar, ConstructStatementOrExpr,
}

// GoConstructKindNames returns the kind identifiers executable by the Go handler.
func GoConstructKindNames() []string {
	names := make([]string, 0, len(GoReplaceConstructKinds))
	for _, kind := range GoReplaceConstructKinds {
		names = append(names, string(kind))
	}
	return names
}

// ConstructOptions configures construct-level replacement.
type ConstructOptions struct {
	AutoOrganizeImports bool
	ConstructPath       string
	Symbol              string
	InFunction          string
	Discriminator       string
}

type constructCandidate struct {
	node  ast.Node
	path  string
	key   string
	class string
}

const replaceConstructOperation = "replace_construct"

// ReplaceConstruct replaces one selected Go construct without rewriting sibling statements.
func ReplaceConstruct(ctx context.Context, filePath, funcName string, kind ConstructKind, discriminator, source string, opts ConstructOptions) (string, error) {
	if selector := opts.Discriminator; selector != "" {
		discriminator = selector
	}
	if kind == ConstructStatementOrExpr {
		owner := opts.InFunction
		if owner == "" {
			owner = funcName
		}
		return replaceStatementOrExpression(ctx, filePath, owner, discriminator, source, opts)
	}
	if kind == ConstructFunction || kind == ConstructMethod || kind == ConstructType || kind == ConstructConst || kind == ConstructVar {
		symbolQuery := opts.Symbol
		if symbolQuery == "" {
			symbolQuery = funcName
		}
		return replaceDeclarationConstruct(ctx, filePath, symbolQuery, kind, source, opts)
	}
	if !validConstructKind(kind) {
		return "", fmt.Errorf("unsupported Go construct kind %q", kind)
	}
	return replaceControlConstruct(ctx, filePath, funcName, kind, discriminator, source, opts)
}

func replaceControlConstruct(ctx context.Context, filePath, funcName string, kind ConstructKind, discriminator, source string, opts ConstructOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	fileSet := token.NewFileSet()
	fileNode, err := parser.ParseFile(fileSet, cleanPath, content, parser.ParseComments)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fileSet, err), Cause: err, Err: ErrSyntax}
	}
	target, err := findControlConstructOwner(fileSet, fileNode, filePath, funcName)
	if err != nil {
		return "", err
	}
	selected, err := selectControlConstruct(fileSet, target.Body, funcName, kind, discriminator, opts.ConstructPath)
	if err != nil {
		return "", err
	}
	replacement, err := parseConstructReplacement(kind, source)
	if err != nil {
		return "", err
	}
	return publishControlConstruct(ctx, cleanPath, funcName, source, content, fileSet, selected.node, replacement, opts.AutoOrganizeImports)
}

func findControlConstructOwner(fset *token.FileSet, file *ast.File, filePath, query string) (*ast.FuncDecl, error) {
	receiver, name, err := symbol.ParseIdentifier(query)
	if err != nil {
		return nil, &symbol.SymbolError{Op: replaceConstructOperation, File: filePath, Symbol: query, Err: err}
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name || extractReceiverName(fn.Recv) != receiver {
			continue
		}
		if fn.Body == nil {
			return nil, &symbol.SymbolError{Op: replaceConstructOperation, File: filePath, Symbol: query, Pos: fset.Position(fn.Pos()), Err: ErrNoBody}
		}
		return fn, nil
	}
	return nil, &symbol.SymbolError{Op: replaceConstructOperation, File: filePath, Symbol: query, Err: symbol.ErrNotFound}
}

func selectControlConstruct(fset *token.FileSet, body *ast.BlockStmt, owner string, kind ConstructKind, discriminator, path string) (constructCandidate, error) {
	candidates := collectConstructCandidates(fset, body, kind)
	matches := matchingConstructs(fset, candidates, kind, discriminator)
	if path != "" {
		matches = candidatesAtPath(matches, path)
		if len(matches) == 0 {
			return constructCandidate{}, fmt.Errorf("construct_path %q does not select %s %q in %s; available candidates:\n%s", path, kind, discriminator, owner, formatConstructCandidates(fset, candidates, kind))
		}
	}
	if len(matches) == 0 {
		return constructCandidate{}, fmt.Errorf("%s %q not found in %s; available candidates (select an if by its condition or short initializer, or set construct_path):\n%s", kind, discriminator, owner, formatConstructCandidates(fset, candidates, kind))
	}
	if len(matches) > 1 {
		return constructCandidate{}, fmt.Errorf("ambiguous %s %q in %s; set construct_path to one of:\n%s", kind, discriminator, owner, formatConstructCandidates(fset, matches, kind))
	}
	return matches[0], nil
}

func matchingConstructs(fset *token.FileSet, candidates []constructCandidate, kind ConstructKind, discriminator string) []constructCandidate {
	var matches []constructCandidate
	for _, candidate := range candidates {
		if discriminator == "" || candidateMatches(fset, candidate, kind, discriminator) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func candidatesAtPath(candidates []constructCandidate, path string) []constructCandidate {
	var selected []constructCandidate
	for _, candidate := range candidates {
		if candidate.path == path {
			selected = append(selected, candidate)
		}
	}
	return selected
}

func publishControlConstruct(ctx context.Context, cleanPath, owner, source string, content []byte, fset *token.FileSet, node ast.Node, replacement string, organizeImports bool) (string, error) {
	start, end := fset.Position(node.Pos()).Offset, fset.Position(node.End()).Offset
	updated := replaceBytes(content, start, end, []byte(replacement))
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	formatted, err = cleanupMoveImports(ctx, cleanPath, formatted)
	if err != nil {
		return "", fmt.Errorf("clean imports after construct replacement: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments); err != nil {
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
	return UnifiedDiff(owner, strings.TrimSpace(string(content[start:end])), strings.TrimSpace(replacement)), nil
}

func validConstructKind(kind ConstructKind) bool {
	return slices.Contains(GoReplaceConstructKinds, kind)
}

type constructCandidateWalker struct {
	fset *token.FileSet
	kind ConstructKind
	list []constructCandidate
}

func collectConstructCandidates(fset *token.FileSet, body *ast.BlockStmt, kind ConstructKind) []constructCandidate {
	walker := constructCandidateWalker{fset: fset, kind: kind}
	walker.walkBlock(body, "")
	return walker.list
}

func (walker *constructCandidateWalker) walkBlock(block *ast.BlockStmt, prefix string) {
	if block == nil {
		return
	}
	for index, stmt := range block.List {
		walker.walkStatement(stmt, childLoopPath(prefix, index))
	}
}

func (walker *constructCandidateWalker) walkStatement(stmt ast.Stmt, path string) {
	switch node := stmt.(type) {
	case *ast.ForStmt:
		walker.walkLoop(node, path)
	case *ast.RangeStmt:
		walker.walkLoop(node, path)
	case *ast.IfStmt:
		walker.walkIf(node, path)
	case *ast.BlockStmt:
		walker.walkBlock(node, childLoopPath(path, 0))
	case *ast.SwitchStmt:
		walker.walkSwitch(node.Body.List, path)
	case *ast.TypeSwitchStmt:
		walker.walkSwitch(node.Body.List, path)
	case *ast.SelectStmt:
		walker.walkSelect(node, path)
	case *ast.DeferStmt:
		if walker.kind == ConstructDefer {
			walker.add(node, nodeText(walker.fset, node.Call), path)
		}
	}
}

func (walker *constructCandidateWalker) walkLoop(stmt ast.Stmt, path string) {
	if walker.kind == ConstructLoop {
		walker.add(stmt, loopHeader(walker.fset, stmt), path)
	}
	var body *ast.BlockStmt
	switch loop := stmt.(type) {
	case *ast.ForStmt:
		body = loop.Body
	case *ast.RangeStmt:
		body = loop.Body
	}
	walker.walkBlock(body, childLoopPath(path, 0))
}

func (walker *constructCandidateWalker) walkIf(node *ast.IfStmt, path string) {
	if walker.kind == ConstructIf {
		walker.add(node, nodeText(walker.fset, node.Cond), path)
	}
	walker.walkBlock(node.Body, childLoopPath(path, 0))
	if node.Else == nil {
		return
	}
	elsePath := childLoopPath(path, 1)
	if block, ok := node.Else.(*ast.BlockStmt); ok {
		if walker.kind == ConstructElse {
			walker.add(block, nodeText(walker.fset, node.Cond), elsePath)
		}
		walker.walkBlock(block, childLoopPath(elsePath, 0))
		return
	}
	walker.walkStatement(node.Else, elsePath)
}

func (walker *constructCandidateWalker) walkSwitch(clauses []ast.Stmt, path string) {
	for index, stmt := range clauses {
		clause, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		clausePath := childLoopPath(path, index)
		if walker.kind == ConstructCase {
			walker.add(clause, caseHeader(walker.fset, clause), clausePath)
		}
		walker.walkBlock(&ast.BlockStmt{List: clause.Body}, clausePath)
	}
}

func (walker *constructCandidateWalker) walkSelect(node *ast.SelectStmt, path string) {
	for index, stmt := range node.Body.List {
		clause, ok := stmt.(*ast.CommClause)
		if !ok {
			continue
		}
		clausePath := childLoopPath(path, index)
		if walker.kind == ConstructSelect {
			walker.add(clause, commHeader(walker.fset, clause), clausePath)
		}
		walker.walkBlock(&ast.BlockStmt{List: clause.Body}, clausePath)
	}
}

func (walker *constructCandidateWalker) add(node ast.Node, key, path string) {
	walker.list = append(walker.list, constructCandidate{node: node, path: path, key: key, class: "construct"})
}

func candidateMatches(fset *token.FileSet, c constructCandidate, kind ConstructKind, selector string) bool {
	needle := strings.TrimSpace(selector)
	if c.key == needle {
		return true
	}
	if kind == ConstructLoop {
		return loopMatches(c.node, fset, needle)
	}
	if kind == ConstructElse {
		return c.key == needle
	}
	if conditional, ok := c.node.(*ast.IfStmt); ok {
		return conditional.Init != nil && nodeText(fset, conditional.Init) == needle
	}
	if kind == ConstructIf {
		return false
	}
	if clause, ok := c.node.(*ast.CaseClause); ok {
		for _, expression := range clause.List {
			if nodeText(fset, expression) == needle {
				return true
			}
		}
		return false
	}
	if clause, ok := c.node.(*ast.CommClause); ok {
		return clause.Comm != nil && nodeText(fset, clause.Comm) == needle
	}
	var found bool
	ast.Inspect(c.node, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name == needle {
			found = true
			return false
		}
		return !found
	})
	return found
}

func formatConstructCandidates(fset *token.FileSet, matches []constructCandidate, kind ConstructKind) string {
	lines := make([]string, 0, len(matches))
	for _, c := range matches {
		label := fmt.Sprintf("%s %s", kind, c.key)
		if conditional, ok := c.node.(*ast.IfStmt); ok && conditional.Init != nil {
			label = fmt.Sprintf("%s init %q; condition %q", kind, nodeText(fset, conditional.Init), nodeText(fset, conditional.Cond))
		}
		lines = append(lines, fmt.Sprintf("construct_path %s at %s: %s", c.path, fset.Position(c.node.Pos()), label))
	}
	if len(lines) == 0 {
		return "(no matching constructs found)"
	}
	return strings.Join(lines, "\n")
}

func parseConstructReplacement(kind ConstructKind, source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", ErrEmptySnippet
	}
	stub, want := constructReplacementWrapper(kind, trimmed)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet", stub, parser.ParseComments)
	if err != nil {
		return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	node := constructReplacementNode(kind, fn)
	if node == nil {
		return "", fmt.Errorf("replacement must be one complete Go %s construct", want)
	}
	start, end := fset.Position(node.Pos()).Offset, fset.Position(node.End()).Offset
	return strings.TrimSpace(stub[start:end]), nil
}

func constructReplacementWrapper(kind ConstructKind, trimmed string) (string, string) {
	var stub, want string
	switch kind {
	case ConstructLoop:
		stub = "package p\nfunc _(){\n" + trimmed + "\n}"
		want = "loop"
	case ConstructIf:
		stub = "package p\nfunc _(){\n" + trimmed + "\n}"
		want = "if"
	case ConstructElse:
		stub = "package p\nfunc _(){\nif true {} else " + trimmed + "\n}"
		want = "else"
	case ConstructCase:
		stub = "package p\nfunc _(){\nswitch {\n" + trimmed + "\n}\n}"
		want = "case"
	case ConstructSelect:
		stub = "package p\nfunc _(){\nselect {\n" + trimmed + "\n}\n}"
		want = "select"
	case ConstructDefer:
		stub = "package p\nfunc _(){\n" + trimmed + "\n}"
		want = "defer"
	}
	return stub, want
}

func constructReplacementNode(kind ConstructKind, fn *ast.FuncDecl) ast.Node {
	switch kind {
	case ConstructLoop:
		return singleLoopStatement(fn)
	case ConstructIf:
		return singleIfStatement(fn)
	case ConstructDefer:
		return singleDeferStatement(fn)
	case ConstructElse:
		return singleElseBlock(fn)
	case ConstructCase:
		return singleCaseClause(fn)
	case ConstructSelect:
		return singleCommClause(fn)
	}
	return nil
}

func singleLoopStatement(fn *ast.FuncDecl) ast.Node {
	stmt := singleBodyStatement(fn)
	if _, ok := stmt.(*ast.ForStmt); ok {
		return stmt
	}
	if _, ok := stmt.(*ast.RangeStmt); ok {
		return stmt
	}
	return nil
}

func singleIfStatement(fn *ast.FuncDecl) ast.Node {
	stmt := singleBodyStatement(fn)
	if _, ok := stmt.(*ast.IfStmt); ok {
		return stmt
	}
	return nil
}

func singleDeferStatement(fn *ast.FuncDecl) ast.Node {
	stmt := singleBodyStatement(fn)
	if _, ok := stmt.(*ast.DeferStmt); ok {
		return stmt
	}
	return nil
}

func singleElseBlock(fn *ast.FuncDecl) ast.Node {
	conditional, ok := singleBodyStatement(fn).(*ast.IfStmt)
	if !ok {
		return nil
	}
	block, _ := conditional.Else.(*ast.BlockStmt)
	return block
}

func singleCaseClause(fn *ast.FuncDecl) ast.Node {
	switchNode, ok := singleBodyStatement(fn).(*ast.SwitchStmt)
	if !ok || len(switchNode.Body.List) != 1 {
		return nil
	}
	clause, _ := switchNode.Body.List[0].(*ast.CaseClause)
	return clause
}

func singleCommClause(fn *ast.FuncDecl) ast.Node {
	selectNode, ok := singleBodyStatement(fn).(*ast.SelectStmt)
	if !ok || len(selectNode.Body.List) != 1 {
		return nil
	}
	clause, _ := selectNode.Body.List[0].(*ast.CommClause)
	return clause
}

func singleBodyStatement(fn *ast.FuncDecl) ast.Stmt {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return nil
	}
	return fn.Body.List[0]
}

func nodeText(fset *token.FileSet, node ast.Node) string {
	if node == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}

func caseHeader(fset *token.FileSet, n *ast.CaseClause) string {
	if len(n.List) == 0 {
		return "default:"
	}
	parts := make([]string, 0, len(n.List))
	for _, item := range n.List {
		parts = append(parts, nodeText(fset, item))
	}
	return "case " + strings.Join(parts, ", ") + ":"
}

func commHeader(fset *token.FileSet, n *ast.CommClause) string {
	if n.Comm == nil {
		return "default:"
	}
	return "case " + nodeText(fset, n.Comm) + ":"
}

func childLoopPath(parent string, index int) string {
	if parent == "" {
		return strconv.Itoa(index)
	}
	return parent + "." + strconv.Itoa(index)
}

func loopHeader(fset *token.FileSet, node ast.Node) string {
	switch n := node.(type) {
	case *ast.ForStmt:
		var buf bytes.Buffer
		buf.WriteString("for ")
		if n.Init != nil {
			_ = format.Node(&buf, fset, n.Init)
		}
		if n.Init != nil || n.Post != nil {
			buf.WriteString("; ")
			if n.Cond != nil {
				_ = format.Node(&buf, fset, n.Cond)
			}
			buf.WriteString("; ")
			if n.Post != nil {
				_ = format.Node(&buf, fset, n.Post)
			}
		} else if n.Cond != nil {
			_ = format.Node(&buf, fset, n.Cond)
		}
		return strings.TrimSpace(buf.String())
	case *ast.RangeStmt:
		var buf bytes.Buffer
		buf.WriteString("for ")
		if n.Key != nil {
			_ = format.Node(&buf, fset, n.Key)
			if n.Value != nil {
				buf.WriteString(", ")
				_ = format.Node(&buf, fset, n.Value)
			}
			buf.WriteByte(' ')
			buf.WriteString(n.Tok.String())
			buf.WriteByte(' ')
		}
		buf.WriteString("range ")
		_ = format.Node(&buf, fset, n.X)
		return buf.String()
	}
	return ""
}

func loopMatches(node ast.Node, fset *token.FileSet, discriminator string) bool {
	needle := strings.TrimSpace(discriminator)
	if needle == loopHeader(fset, node) {
		return true
	}
	var parts []ast.Node
	switch n := node.(type) {
	case *ast.ForStmt:
		parts = []ast.Node{n.Init, n.Cond, n.Post}
	case *ast.RangeStmt:
		parts = []ast.Node{n.Key, n.Value, n.X}
	}
	for _, part := range parts {
		if part == nil {
			continue
		}
		if nodeText(fset, part) == needle {
			return true
		}
		found := false
		ast.Inspect(part, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && ident.Name == needle {
				found = true
				return false
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

func replaceBytes(src []byte, start, end int, replacement []byte) []byte {
	out := make([]byte, 0, len(src)-(end-start)+len(replacement))
	out = append(out, src[:start]...)
	out = append(out, replacement...)
	out = append(out, src[end:]...)
	return out
}
