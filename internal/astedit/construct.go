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
	"strconv"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// LoopOptions configures construct-level loop replacement.
type LoopOptions struct {
	AutoOrganizeImports bool
	LoopPath            string
}

type loopCandidate struct {
	node   ast.Stmt
	header string
	path   string
}

// ReplaceLoop replaces one for or range loop in a function with another complete loop.
func ReplaceLoop(ctx context.Context, filePath, funcName, loopOn, replacementSource string, opts LoopOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}
	recv, name, err := symbol.ParseIdentifier(funcName)
	if err != nil {
		return "", &symbol.SymbolError{Op: "replace_loop", File: filePath, Symbol: funcName, Err: err}
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
		return "", &symbol.SymbolError{Op: "replace_loop", File: filePath, Symbol: funcName, Err: symbol.ErrNotFound}
	}
	if target.Body == nil {
		return "", &symbol.SymbolError{Op: "replace_loop", File: filePath, Symbol: funcName, Pos: fset.Position(target.Pos()), Err: ErrNoBody}
	}
	candidates := collectLoopCandidates(fset, target.Body)
	var matches []loopCandidate
	for _, candidate := range candidates {
		if loopOn == "" || loopMatches(candidate.node, fset, loopOn) {
			matches = append(matches, candidate)
		}
	}
	if opts.LoopPath != "" {
		var selected []loopCandidate
		for _, candidate := range matches {
			if candidate.path == opts.LoopPath {
				selected = append(selected, candidate)
			}
		}
		if len(selected) == 0 {
			return "", fmt.Errorf("loop path %q does not match loop %q in %s; candidates:\n%s", opts.LoopPath, loopOn, funcName, formatLoopCandidates(matches, fset))
		}
		matches = selected
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("loop %q not found in %s", loopOn, funcName)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous loop %q in %s; set loop_path to one of:\n%s", loopOn, funcName, formatLoopCandidates(matches, fset))
	}
	selected := matches[0]
	replacement, err := parseLoopReplacement(replacementSource)
	if err != nil {
		return "", err
	}
	start, end := fset.Position(selected.node.Pos()).Offset, fset.Position(selected.node.End()).Offset
	oldText := string(content[start:end])
	content = replaceBytes(content, start, end, []byte(replacement.text))
	formatted, err := format.Source(content)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: replacementSource, Cause: err, Err: ErrSyntax}
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
	return UnifiedDiff(funcName, strings.TrimSpace(oldText), strings.TrimSpace(replacement.text)), nil
}

type parsedLoopReplacement struct{ text string }

func parseLoopReplacement(source string) (parsedLoopReplacement, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return parsedLoopReplacement{}, ErrEmptySnippet
	}
	synth := "package dummy\nfunc _() {\n" + trimmed + "\n}"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet", synth, parser.ParseComments)
	if err != nil {
		return parsedLoopReplacement{}, &SyntaxError{File: "snippet", Snippet: source, Cause: err, Err: ErrSyntax}
	}
	if len(file.Decls) == 1 {
		if fn, ok := file.Decls[0].(*ast.FuncDecl); ok && len(fn.Body.List) == 1 {
			if _, ok := fn.Body.List[0].(*ast.ForStmt); ok {
				return parsedLoopReplacement{text: trimmed}, nil
			}
			if _, ok := fn.Body.List[0].(*ast.RangeStmt); ok {
				return parsedLoopReplacement{text: trimmed}, nil
			}
		}
	}
	return parsedLoopReplacement{}, fmt.Errorf("replacement must be one complete for or range loop")
}

func collectLoopCandidates(fset *token.FileSet, body *ast.BlockStmt) []loopCandidate {
	var out []loopCandidate
	var walkBlock func(*ast.BlockStmt, string)
	var walkStmt func(ast.Stmt, string)
	walkBlock = func(block *ast.BlockStmt, prefix string) {
		if block == nil {
			return
		}
		for index, stmt := range block.List {
			path := childLoopPath(prefix, index)
			walkStmt(stmt, path)
		}
	}
	walkStmt = func(stmt ast.Stmt, path string) {
		switch n := stmt.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			out = append(out, loopCandidate{node: stmt, header: loopHeader(fset, stmt), path: path})
			var nested *ast.BlockStmt
			switch x := stmt.(type) {
			case *ast.ForStmt:
				nested = x.Body
			case *ast.RangeStmt:
				nested = x.Body
			}
			walkBlock(nested, childLoopPath(path, 0))
		case *ast.IfStmt:
			walkBlock(n.Body, childLoopPath(path, 0))
			if n.Else != nil {
				walkStmt(n.Else, childLoopPath(path, 1))
			}
		case *ast.BlockStmt:
			walkBlock(n, childLoopPath(path, 0))
		case *ast.SwitchStmt:
			for i, stmt := range n.Body.List {
				if c, ok := stmt.(*ast.CaseClause); ok {
					walkBlock(&ast.BlockStmt{List: c.Body}, childLoopPath(path, i))
				}
			}
		case *ast.TypeSwitchStmt:
			for i, stmt := range n.Body.List {
				if c, ok := stmt.(*ast.CaseClause); ok {
					walkBlock(&ast.BlockStmt{List: c.Body}, childLoopPath(path, i))
				}
			}
		case *ast.SelectStmt:
			for i, stmt := range n.Body.List {
				if c, ok := stmt.(*ast.CommClause); ok {
					walkBlock(&ast.BlockStmt{List: c.Body}, childLoopPath(path, i))
				}
			}
		}
	}
	walkBlock(body, "")
	return out
}

func childLoopPath(parent string, index int) string {
	if parent == "" {
		return strconv.Itoa(index)
	}
	return parent + "." + strconv.Itoa(index)
}

func formatLoopCandidates(matches []loopCandidate, fset *token.FileSet) string {
	details := make([]string, 0, len(matches))
	for _, candidate := range matches {
		details = append(details, fmt.Sprintf("loop_path %s at %s: %s", candidate.path, fset.Position(candidate.node.Pos()), candidate.header))
	}
	return strings.Join(details, "\n")
}

func loopHeader(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	switch n := node.(type) {
	case *ast.ForStmt:
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
	switch loop := node.(type) {
	case *ast.ForStmt:
		parts = []ast.Node{loop.Init, loop.Cond, loop.Post}
	case *ast.RangeStmt:
		parts = []ast.Node{loop.Key, loop.Value, loop.X}
	}
	for _, part := range parts {
		if part == nil {
			continue
		}
		var formatted bytes.Buffer
		if format.Node(&formatted, fset, part) == nil && formatted.String() == needle {
			return true
		}
		found := false
		ast.Inspect(part, func(child ast.Node) bool {
			ident, ok := child.(*ast.Ident)
			if ok && ident.Name == needle {
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
