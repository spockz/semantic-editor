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
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// ConstructKind aliases the canonical cross-language syntax taxonomy.
type ConstructKind = backend.ConstructKind

// Go construct kind constants name syntax forms handled by ReplaceConstruct.
const (
	ConstructLoop   = backend.ConstructLoop
	ConstructIf     = backend.ConstructIf
	ConstructElse   = backend.ConstructElse
	ConstructCase   = backend.ConstructCase
	ConstructSelect = backend.ConstructSelect
	ConstructDefer  = backend.ConstructDefer
)

// GoConstructKinds lists the construct kinds implemented by the Go handler.
var GoConstructKinds = []ConstructKind{ConstructLoop, ConstructIf, ConstructElse, ConstructCase, ConstructSelect, ConstructDefer}

// GoConstructKindNames returns the kind identifiers executable by the Go handler.
func GoConstructKindNames() []string {
	names := make([]string, 0, len(GoConstructKinds))
	for _, kind := range GoConstructKinds {
		names = append(names, string(kind))
	}
	return names
}

// ConstructOptions configures construct-level replacement.
type ConstructOptions struct {
	AutoOrganizeImports bool
	ConstructPath       string
}

type constructCandidate struct {
	node ast.Node
	path string
	key  string
}

// ReplaceConstruct replaces one selected Go construct without rewriting sibling statements.
func ReplaceConstruct(ctx context.Context, filePath, funcName string, kind ConstructKind, discriminator, source string, opts ConstructOptions) (string, error) {
	if !validConstructKind(kind) {
		return "", fmt.Errorf("unsupported Go construct kind %q", kind)
	}
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	recv, name, err := symbol.ParseIdentifier(funcName)
	if err != nil {
		return "", &symbol.SymbolError{Op: "replace_construct", File: filePath, Symbol: funcName, Err: err}
	}
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	var target *ast.FuncDecl
	for _, decl := range fileNode.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == recv {
			target = fn
			break
		}
	}
	if target == nil {
		return "", &symbol.SymbolError{Op: "replace_construct", File: filePath, Symbol: funcName, Err: symbol.ErrNotFound}
	}
	if target.Body == nil {
		return "", &symbol.SymbolError{Op: "replace_construct", File: filePath, Symbol: funcName, Pos: fset.Position(target.Pos()), Err: ErrNoBody}
	}
	candidates := collectConstructCandidates(fset, target.Body, kind)
	var matches []constructCandidate
	for _, candidate := range candidates {
		if discriminator == "" || candidateMatches(fset, candidate, kind, discriminator) {
			matches = append(matches, candidate)
		}
	}
	if opts.ConstructPath != "" {
		var selected []constructCandidate
		for _, candidate := range matches {
			if candidate.path == opts.ConstructPath {
				selected = append(selected, candidate)
			}
		}
		if len(selected) == 0 {
			return "", fmt.Errorf("construct_path %q does not match %s %q in %s; candidates:\n%s", opts.ConstructPath, kind, discriminator, funcName, formatConstructCandidates(fset, matches, kind))
		}
		matches = selected
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("%s %q not found in %s", kind, discriminator, funcName)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous %s %q in %s; set construct_path to one of:\n%s", kind, discriminator, funcName, formatConstructCandidates(fset, matches, kind))
	}
	replacement, err := parseConstructReplacement(kind, source)
	if err != nil {
		return "", err
	}
	selected := matches[0].node
	start, end := fset.Position(selected.Pos()).Offset, fset.Position(selected.End()).Offset
	oldText := string(content[start:end])
	updated := replaceBytes(content, start, end, []byte(replacement))
	formatted, err := format.Source(updated)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	} else if err := pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
		return "", fmt.Errorf("format file: %w", err)
	}
	return UnifiedDiff(funcName, strings.TrimSpace(oldText), strings.TrimSpace(replacement)), nil
}

func validConstructKind(kind ConstructKind) bool {
	return slices.Contains(GoConstructKinds, kind)
}

func collectConstructCandidates(fset *token.FileSet, body *ast.BlockStmt, kind ConstructKind) []constructCandidate {
	var out []constructCandidate
	var walkBlock func(*ast.BlockStmt, string)
	var walkStmt func(ast.Stmt, string)
	walkBlock = func(block *ast.BlockStmt, prefix string) {
		if block == nil {
			return
		}
		for i, stmt := range block.List {
			walkStmt(stmt, childLoopPath(prefix, i))
		}
	}
	walkStmt = func(stmt ast.Stmt, path string) {
		addAt := func(node ast.Node, key, candidatePath string) {
			out = append(out, constructCandidate{node: node, path: candidatePath, key: key})
		}
		add := func(node ast.Node, key string) {
			addAt(node, key, path)
		}
		switch n := stmt.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			if kind == ConstructLoop {
				add(stmt, loopHeader(fset, stmt))
			}
			var nested *ast.BlockStmt
			switch x := stmt.(type) {
			case *ast.ForStmt:
				nested = x.Body
			case *ast.RangeStmt:
				nested = x.Body
			}
			walkBlock(nested, childLoopPath(path, 0))
		case *ast.IfStmt:
			if kind == ConstructIf {
				add(n, nodeText(fset, n.Cond))
			}
			walkBlock(n.Body, childLoopPath(path, 0))
			if n.Else != nil {
				elsePath := childLoopPath(path, 1)
				if block, ok := n.Else.(*ast.BlockStmt); ok {
					if kind == ConstructElse {
						addAt(block, nodeText(fset, n.Cond), elsePath)
					}
					walkBlock(block, childLoopPath(elsePath, 0))
				} else {
					walkStmt(n.Else, elsePath)
				}
			}
		case *ast.BlockStmt:
			walkBlock(n, childLoopPath(path, 0))
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			var list []ast.Stmt
			switch x := stmt.(type) {
			case *ast.SwitchStmt:
				list = x.Body.List
			case *ast.TypeSwitchStmt:
				list = x.Body.List
			}
			for i, item := range list {
				if clause, ok := item.(*ast.CaseClause); ok {
					clausePath := childLoopPath(path, i)
					if kind == ConstructCase {
						addAt(clause, caseHeader(fset, clause), clausePath)
					}
					walkBlock(&ast.BlockStmt{List: clause.Body}, clausePath)
				}
			}
		case *ast.SelectStmt:
			for i, item := range n.Body.List {
				if clause, ok := item.(*ast.CommClause); ok {
					clausePath := childLoopPath(path, i)
					if kind == ConstructSelect {
						addAt(clause, commHeader(fset, clause), clausePath)
					}
					walkBlock(&ast.BlockStmt{List: clause.Body}, clausePath)
				}
			}
		case *ast.DeferStmt:
			if kind == ConstructDefer {
				add(n, nodeText(fset, n.Call))
			}
		}
	}
	walkBlock(body, "")
	return out
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
	if kind == ConstructIf {
		return c.key == needle
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
	// Identifiers are useful selectors for expressions such as defer calls and case values.
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
		lines = append(lines, fmt.Sprintf("construct_path %s at %s: %s %s", c.path, fset.Position(c.node.Pos()), kind, c.key))
	}
	return strings.Join(lines, "\n")
}

func parseConstructReplacement(kind ConstructKind, source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", ErrEmptySnippet
	}
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
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet", stub, parser.ParseComments)
	if err != nil {
		return "", &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	var node ast.Node
	switch kind {
	case ConstructLoop, ConstructIf, ConstructDefer:
		if len(fn.Body.List) == 1 {
			node = fn.Body.List[0]
		}
		if kind == ConstructLoop {
			if _, ok := node.(*ast.ForStmt); !ok {
				if _, ok := node.(*ast.RangeStmt); !ok {
					node = nil
				}
			}
		}
		if kind == ConstructIf {
			if _, ok := node.(*ast.IfStmt); !ok {
				node = nil
			}
		}
		if kind == ConstructDefer {
			if _, ok := node.(*ast.DeferStmt); !ok {
				node = nil
			}
		}
	case ConstructElse:
		if len(fn.Body.List) == 1 {
			if n, ok := fn.Body.List[0].(*ast.IfStmt); ok {
				if _, ok = n.Else.(*ast.BlockStmt); ok {
					node = n.Else
				}
			}
		}
	case ConstructCase:
		if n, ok := fn.Body.List[0].(*ast.SwitchStmt); ok && len(n.Body.List) == 1 {
			node, _ = n.Body.List[0].(*ast.CaseClause)
		}
	case ConstructSelect:
		if n, ok := fn.Body.List[0].(*ast.SelectStmt); ok && len(n.Body.List) == 1 {
			node, _ = n.Body.List[0].(*ast.CommClause)
		}
	}
	if node == nil {
		return "", fmt.Errorf("replacement must be one complete Go %s construct", want)
	}
	start, end := fset.Position(node.Pos()).Offset, fset.Position(node.End()).Offset
	return strings.TrimSpace(stub[start:end]), nil
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
