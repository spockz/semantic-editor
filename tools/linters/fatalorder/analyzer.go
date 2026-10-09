// Package fatalorder defines an analysis.Analyzer verifying that t.Fatalf is never followed by sibling test checks.
package fatalorder

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer checks that t.Fatalf / t.Fatal / t.FailNow is called as the terminal test failure function.
var Analyzer = &analysis.Analyzer{
	Name:     "fatalorder",
	Doc:      "reports t.Fatalf, t.Fatal, and t.FailNow calls that are not the last test failure function in a test case or appear in loops",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type testCallInfo struct {
	call           *ast.CallExpr
	name           string
	isFatal        bool
	inLoop         bool
	enclosingStmts []ast.Stmt
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	// Index comments by file for nolint directive lookups.
	fileComments := make(map[*ast.File][]*ast.CommentGroup)
	for _, file := range pass.Files {
		fileComments[file] = file.Comments
	}

	findFileForNode := func(node ast.Node) *ast.File {
		pos := node.Pos()
		for _, file := range pass.Files {
			if file.Pos() <= pos && pos <= file.End() {
				return file
			}
		}
		return nil
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		switch fn := n.(type) {
		case *ast.FuncDecl:
			body = fn.Body
		case *ast.FuncLit:
			body = fn.Body
		}
		if body == nil {
			return
		}

		file := findFileForNode(n)
		var comments []*ast.CommentGroup
		if file != nil {
			comments = fileComments[file]
		}

		calls := collectTestCalls(pass, body)

		for i, c := range calls {
			if !c.isFatal {
				continue
			}

			if isSuppressed(pass.Fset, comments, c.call, c.enclosingStmts) {
				continue
			}

			if c.inLoop {
				pass.Reportf(c.call.Pos(), "t.%s must not be called inside a loop; use t.Errorf to preserve row iteration (or suppress with //nolint:fatalorder)", c.name)
				continue
			}

			if i < len(calls)-1 {
				nextCall := calls[i+1]
				nextPos := pass.Fset.Position(nextCall.call.Pos())
				pass.Reportf(c.call.Pos(), "t.%s is not called as the last check in this test case; subsequent test assertion at line %d is masked (use t.Errorf or suppress with //nolint:fatalorder)", c.name, nextPos.Line)
			}
		}
	})

	return struct{}{}, nil
}

func collectTestCalls(pass *analysis.Pass, body *ast.BlockStmt) []testCallInfo {
	var calls []testCallInfo
	v := &scopeVisitor{
		pass:  pass,
		calls: &calls,
	}
	ast.Walk(v, body)
	return calls
}

type scopeVisitor struct {
	pass           *analysis.Pass
	loopDepth      int
	enclosingStmts []ast.Stmt
	calls          *[]testCallInfo
}

func (v *scopeVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch node := n.(type) {
	case *ast.FuncLit:
		// Do not descend into child closures; each closure is analyzed as its own scope.
		return nil

	case *ast.ForStmt:
		child := *v
		child.loopDepth++
		child.enclosingStmts = append(slices.Clip(child.enclosingStmts), node)
		if node.Init != nil {
			ast.Walk(&child, node.Init)
		}
		if node.Cond != nil {
			ast.Walk(&child, node.Cond)
		}
		if node.Post != nil {
			ast.Walk(&child, node.Post)
		}
		if node.Body != nil {
			ast.Walk(&child, node.Body)
		}
		return nil

	case *ast.RangeStmt:
		child := *v
		child.loopDepth++
		child.enclosingStmts = append(slices.Clip(child.enclosingStmts), node)
		if node.Key != nil {
			ast.Walk(&child, node.Key)
		}
		if node.Value != nil {
			ast.Walk(&child, node.Value)
		}
		if node.X != nil {
			ast.Walk(&child, node.X)
		}
		if node.Body != nil {
			ast.Walk(&child, node.Body)
		}
		return nil

	case *ast.CallExpr:
		if name, isFatal, isTest := isTestingTCall(v.pass, node); isTest {
			*v.calls = append(*v.calls, testCallInfo{
				call:           node,
				name:           name,
				isFatal:        isFatal,
				inLoop:         v.loopDepth > 0,
				enclosingStmts: v.enclosingStmts,
			})
		}
	}

	if stmt, ok := n.(ast.Stmt); ok {
		if _, isBlock := stmt.(*ast.BlockStmt); !isBlock {
			child := *v
			child.enclosingStmts = append(slices.Clip(child.enclosingStmts), stmt)
			return &child
		}
	}

	return v
}

func isTestingTCall(pass *analysis.Pass, call *ast.CallExpr) (name string, isFatal bool, isTestCall bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false, false
	}

	funcName := sel.Sel.Name
	switch funcName {
	case "Fatalf", "Fatal", "FailNow":
		isFatal = true
	case "Errorf", "Error", "Fail":
		isFatal = false
	default:
		return "", false, false
	}

	if tv, exists := pass.TypesInfo.Types[sel.X]; exists && tv.Type != nil {
		typ := tv.Type
		if ptr, isPtr := typ.(*types.Pointer); isPtr {
			typ = ptr.Elem()
		}
		if named, isNamed := typ.(*types.Named); isNamed {
			obj := named.Obj()
			if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "testing" && (obj.Name() == "T" || obj.Name() == "B") {
				return funcName, isFatal, true
			}
		}
	}

	// Syntactic fallback for standard testing variables
	if ident, isIdent := sel.X.(*ast.Ident); isIdent {
		switch ident.Name {
		case "t", "b", "tb":
			return funcName, isFatal, true
		}
	}

	return "", false, false
}

func isSuppressed(fset *token.FileSet, comments []*ast.CommentGroup, call *ast.CallExpr, enclosingStmts []ast.Stmt) bool {
	if len(comments) == 0 {
		return false
	}

	callLine := fset.Position(call.Pos()).Line

	linesToCheck := make(map[int]bool)
	linesToCheck[callLine] = true
	linesToCheck[callLine-1] = true

	for _, stmt := range enclosingStmts {
		stmtLine := fset.Position(stmt.Pos()).Line
		linesToCheck[stmtLine] = true
		linesToCheck[stmtLine-1] = true
	}

	for _, cg := range comments {
		for _, c := range cg.List {
			cLine := fset.Position(c.Pos()).Line
			if linesToCheck[cLine] && matchesNolint(c.Text, "fatalorder") {
				return true
			}
		}
	}

	return false
}

func matchesNolint(commentText, linterName string) bool {
	text := strings.TrimSpace(commentText)
	if after, ok := strings.CutPrefix(text, "//"); ok {
		text = strings.TrimSpace(after)
	} else if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/"))
	}

	if !strings.HasPrefix(text, "nolint") {
		return false
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "nolint"))

	// Bare "//nolint" matches all linters.
	if text == "" || strings.HasPrefix(text, "//") || strings.HasPrefix(text, "/*") {
		return true
	}

	if !strings.HasPrefix(text, ":") {
		return false
	}
	text = strings.TrimPrefix(text, ":")

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}

	linters := strings.SplitSeq(fields[0], ",")
	for l := range linters {
		l = strings.TrimSpace(l)
		if l == linterName || l == "*" {
			return true
		}
	}

	return false
}
