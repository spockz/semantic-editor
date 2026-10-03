// This file proves that construct edits select syntax slots precisely and reject unsafe or ambiguous spans before writing.
package astedit

import (
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/backend"
)

func TestReplaceConstructStatementExpressionAmbiguityAndPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() {\n\tf(1, 1)\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	selector := " f( 1,\n 1 ) "
	_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, selector, "g()", ConstructOptions{})
	if err == nil || !strings.Contains(err.Error(), "ambiguous statement/expression") || !strings.Contains(err.Error(), "0 (statement)") || !strings.Contains(err.Error(), "0.e0 (expression)") {
		t.Fatalf("same-span statement/expression was not reported as ambiguous: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("ambiguous replacement changed the file")
	}
	_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "1", "2", ConstructOptions{ConstructPath: "0.e0.1"})
	if err != nil {
		t.Fatalf("whitespace-normalized repeated argument selection failed: %v", err)
	}
	after, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertCallArgumentValues(t, after, "1", "2")
}

func assertCallArgumentValues(t *testing.T, source []byte, expected ...string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	stmt := file.Decls[0].(*ast.FuncDecl).Body.List[0].(*ast.ExprStmt)
	call := stmt.X.(*ast.CallExpr)
	if len(call.Args) != len(expected) {
		t.Fatalf("argument count %d, want %d", len(call.Args), len(expected))
	}
	for index, expr := range call.Args {
		for {
			paren, ok := expr.(*ast.ParenExpr)
			if !ok {
				break
			}
			expr = paren.X
		}
		literal, ok := expr.(*ast.BasicLit)
		if !ok || literal.Value != expected[index] {
			t.Fatalf("argument %d is %T(%v), want literal %s", index, expr, expr, expected[index])
		}
	}
}

func TestReplaceConstructPreservesReplacementComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() int {\n\treturn value\n}\n")
	if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "value", "next /* retained */ + 1", ConstructOptions{ConstructPath: "0.e0"}); err != nil {
		t.Fatalf("replace expression: %v", err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "/* retained */") {
		t.Fatalf("replacement comment was discarded: %s", updated)
	}
}

func TestReplaceConstructRejectsTrailingLineCommentWithoutWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() int {\n\treturn value + 1\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "value", "next() // retained", ConstructOptions{ConstructPath: "0.e0.0"})
	if err == nil || !strings.Contains(err.Error(), "trailing line comment") {
		t.Fatalf("unsafe trailing line comment was not refused: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("refused replacement changed the file")
	}
}

func TestReplaceConstructPreservesStatementCommentsAndRejectsUnsafeLineComment(t *testing.T) {
	t.Run("nested block comment survives", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "main.go")
		writeConstructFixture(t, path, "package p\nfunc run() {\n\told()\n\tother()\n}\n")
		if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "old()", "if true { /* keep */ next() }", ConstructOptions{ConstructPath: "0"}); err != nil {
			t.Fatalf("replace statement: %v", err)
		}
		updated, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(updated), "/* keep */") || !strings.Contains(string(updated), "other()") {
			t.Fatalf("replacement comment or sibling statement was lost: %s", updated)
		}
	})
	t.Run("trailing line comment is refused without write", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "main.go")
		writeConstructFixture(t, path, "package p\nfunc run() { old(); other() }\n")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "old()", "next() // keep", ConstructOptions{ConstructPath: "0"})
		if err == nil || !strings.Contains(err.Error(), "trailing line comment") {
			t.Fatalf("unsafe trailing statement comment was not refused: %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("refused replacement changed the file")
		}
	})
}

func TestReplaceConstructExpressionKeepsReplacementPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() int {\n\treturn 1 + value\n}\n")
	if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "value", "left + right", ConstructOptions{ConstructPath: "0.e0.1"}); err != nil {
		t.Fatalf("replace expression: %v", err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", updated, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	returned := file.Decls[0].(*ast.FuncDecl).Body.List[0].(*ast.ReturnStmt)
	outer := returned.Results[0].(*ast.BinaryExpr)
	if _, ok := outer.Y.(*ast.ParenExpr); !ok {
		t.Fatalf("replacement precedence was not preserved: %s", updated)
	}
}

func TestReplaceConstructPrunesOrphanImportWithoutOptIn(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/p\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "main.go")
	writeConstructFixture(t, path, "package p\nimport \"strings\"\nfunc run() string {\n\treturn strings.TrimSpace(\" x \")\n}\n")
	if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "strings.TrimSpace(\" x \")", "\"x\"", ConstructOptions{ConstructPath: "0.e0"}); err != nil {
		t.Fatalf("replace expression: %v", err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated), "strings") {
		t.Fatalf("orphan import was retained without auto_organize_imports: %s", updated)
	}
}

func TestReplaceConstructCallSlotsInGoAndDefer(t *testing.T) {
	for _, keyword := range []string{"go", "defer"} {
		t.Run(keyword, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.go")
			writeConstructFixture(t, path, "package p\nfunc run() {\n\t"+keyword+" old()\n}\n")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, invalidErr := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "old()", "next", ConstructOptions{ConstructPath: "0.e0"}); invalidErr == nil {
				t.Fatal("non-call replacement was accepted for a go/defer call slot")
			}
			afterFailure, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(afterFailure) != string(before) {
				t.Fatal("invalid call-slot replacement changed the file")
			}
			if _, replaceErr := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "old()", "next()", ConstructOptions{ConstructPath: "0.e0"}); replaceErr != nil {
				t.Fatalf("replace call operand: %v", replaceErr)
			}
			updated, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(updated), keyword+" next()") {
				t.Fatalf("call expression was not preserved in %s slot: %s", keyword, updated)
			}
		})
	}
}

func TestReplaceConstructDeclarationKindAndFunctionComments(t *testing.T) {
	t.Run("declaration kind mismatch is rejected unchanged", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "main.go")
		writeConstructFixture(t, path, "package p\nvar Value = 1\n")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ReplaceConstruct(context.Background(), path, "", backend.ConstructConst, "", "const Value = 2", ConstructOptions{Symbol: "Value"})
		if err == nil || !strings.Contains(err.Error(), "selected declaration") {
			t.Fatalf("const kind mismatch was not rejected: %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("kind mismatch changed the file")
		}
	})
	t.Run("function body comments survive re-emission", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "main.go")
		writeConstructFixture(t, path, "package p\nfunc run() { old() }\n")
		_, err := ReplaceConstruct(context.Background(), path, "", backend.ConstructFunction, "", "func run() {\n // body comment\n updated()\n}", ConstructOptions{Symbol: "run"})
		if err != nil {
			t.Fatalf("replace function: %v", err)
		}
		updated, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(updated), "// body comment") || !strings.Contains(string(updated), "updated()") {
			t.Fatalf("replacement comment or body was lost: %s", updated)
		}
	})
}

func TestReplaceConstructFindsHeaderInitializerAndAssignmentIndexExpressions(t *testing.T) {
	for _, test := range []struct {
		name, source, selector, replacement, path string
	}{
		{
			name: "return runtime index", source: "package p\nfunc run(items []int) int {\n\treturn items[17]\n}\n",
			selector: "17", replacement: "18", path: "0.e0.1",
		},
		{
			name: "return index variable", source: "package p\nfunc run(items []int, index int) int {\n\treturn items[index]\n}\n",
			selector: "index", replacement: "index + 1", path: "0.e0.1",
		},
		{
			name: "return call index", source: "package p\nfunc index() int { return 1 }\nfunc run(items []int) int {\n\treturn items[index()]\n}\n",
			selector: "index()", replacement: "len(items)", path: "0.e0.1",
		},
		{
			name: "if initializer RHS", source: "package p\nfunc run() {\n\tif x := compute(7); x > 0 { return }\n}\n",
			selector: "7", replacement: "8", path: "0.init.e0.0",
		},
		{
			name: "assignment index", source: "package p\nfunc run() {\n\tvalues[17] = 3\n}\n",
			selector: "17", replacement: "18", path: "0.l0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.go")
			writeConstructFixture(t, path, test.source)
			if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, test.selector, test.replacement, ConstructOptions{ConstructPath: test.path}); err != nil {
				t.Fatalf("replace expression: %v", err)
			}
			updated, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			switch test.name {
			case "return runtime index", "return index variable", "return call index":
				assertReturnIndexOperand(t, updated, test.replacement)
			default:
				assertConstructTestLiteral(t, updated, test.name, test.replacement)
			}
		})
	}
}

func TestReplaceConstructClassifiesIndexSlotsWithPartialTypeInfo(t *testing.T) {
	t.Run("unresolved import does not hide known literal index", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "main.go")
		writeConstructFixture(t, path, "package p\nimport _ \"example.com/unavailable\"\nfunc run(items []int) int { return items[17] }\n")
		if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "17", "18", ConstructOptions{ConstructPath: "0.e0.1"}); err != nil {
			t.Fatalf("replace known runtime index with partial type info: %v", err)
		}
		updated, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		assertReturnIndexOperand(t, updated, "18")
	})
	t.Run("unresolved index operand reports why selection is refused", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "main.go")
		writeConstructFixture(t, path, "package p\nimport _ \"example.com/unavailable\"\nfunc run(items []int) int { return items[Missing] }\n")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "Missing", "17", ConstructOptions{ConstructPath: "0.e0.1"})
		if err == nil || !strings.Contains(err.Error(), "type analysis is incomplete") {
			t.Fatalf("unresolved index operand was not refused with a reason: %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("unresolved index refusal changed the file")
		}
	})
}

func assertReturnIndexOperand(t *testing.T, source []byte, want string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		candidate, ok := decl.(*ast.FuncDecl)
		if ok && candidate.Name.Name == "run" {
			fn = candidate
			break
		}
	}
	if fn == nil {
		t.Fatal("run function is missing")
	}
	returned := fn.Body.List[0].(*ast.ReturnStmt)
	index := returned.Results[0].(*ast.IndexExpr)
	operand := index.Index
	for {
		parenthesized, ok := operand.(*ast.ParenExpr)
		if !ok {
			break
		}
		operand = parenthesized.X
	}
	var actual strings.Builder
	if err := format.Node(&actual, fset, operand); err != nil {
		t.Fatal(err)
	}
	if actual.String() != want {
		t.Fatalf("index operand=%q, want %q; source=%s", actual.String(), want, source)
	}
}

func TestReplaceConstructIncludesSelectAndTypeSwitchRuntimeHeaders(t *testing.T) {
	for _, test := range []struct {
		name, source, selector, replacement, path, want string
	}{
		{
			name: "select send value", source: "package p\nfunc run(ch chan int) {\n\tselect { case ch <- 17: }\n}\n",
			selector: "17", replacement: "18", path: "0.0.comm.e1", want: "ch <- (18)",
		},
		{
			name: "type switch guard", source: "package p\nfunc run(value any, other any) {\n\tswitch x := value.(type) { case int: _ = x }\n}\n",
			selector: "value", replacement: "other", path: "0.assign.e0.0", want: "(other).(type)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.go")
			writeConstructFixture(t, path, test.source)
			if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, test.selector, test.replacement, ConstructOptions{ConstructPath: test.path}); err != nil {
				t.Fatalf("replace header expression: %v", err)
			}
			updated, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(updated), test.want) {
				t.Fatalf("replacement missing %q: %s", test.want, updated)
			}
		})
	}
}

func TestReplaceConstructDoesNotExposeTypeArgumentsAsExpressions(t *testing.T) {
	for _, test := range []struct {
		name, source, selector, path string
	}{
		{
			name: "make type argument", source: "package p\nfunc run() { _ = make([]int, 0) }\n", selector: "[]int", path: "0.e0.0",
		},
		{
			name: "generic type argument", source: "package p\nfunc identity[T any](value T) T { return value }\nfunc run() { _ = identity[int](0) }\n", selector: "int", path: "0.e0.0.1",
		},
		{
			name: "make named type argument", source: "package p\ntype Values []int\nfunc run() { _ = make(Values, 0) }\n", selector: "Values", path: "0.e0.0",
		},
		{
			name: "new named type argument", source: "package p\ntype Value struct{}\nfunc run() { _ = new(Value) }\n", selector: "Value", path: "0.e0.0",
		},
		{
			name: "parenthesized make named type argument", source: "package p\ntype Values []int\nfunc run() { _ = (make)(Values, 0) }\n", selector: "Values", path: "0.e0.0",
		},
		{
			name: "parenthesized new type argument", source: "package p\nfunc run() { _ = (new)(int) }\n", selector: "int", path: "0.e0.0",
		},
		{
			name: "unresolved imported make type argument", source: "package p\nimport \"example.com/types\"\nfunc run() { _ = make(types.Slice, 0) }\n", selector: "types.Slice", path: "0.e0.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.go")
			writeConstructFixture(t, path, test.source)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, test.selector, "replacement", ConstructOptions{ConstructPath: test.path})
			if err == nil {
				t.Fatal("type syntax was accepted as an expression slot")
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("rejected type-slot replacement changed the file")
			}
		})
	}
}

func TestReplaceConstructKeepsShadowedMakeArgumentsAsExpressions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc make(value int, _ int) []int { return []int{value} }\nfunc run() { _ = (make)(17, 0) }\n")
	if _, err := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "17", "18", ConstructOptions{ConstructPath: "0.e0.0"}); err != nil {
		t.Fatalf("replace shadowed make argument: %v", err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", updated, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[1].(*ast.FuncDecl)
	call := fn.Body.List[0].(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)
	argument := call.Args[0]
	for {
		parenthesized, ok := argument.(*ast.ParenExpr)
		if !ok {
			break
		}
		argument = parenthesized.X
	}
	literal, ok := argument.(*ast.BasicLit)
	if !ok || literal.Value != "18" {
		t.Fatalf("shadowed make argument was not treated as a runtime value: %s", updated)
	}
}

func assertConstructTestLiteral(t *testing.T, source []byte, testName, want string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	var selected ast.Expr
	switch testName {
	case "if initializer RHS":
		statement := fn.Body.List[0].(*ast.IfStmt)
		assignment := statement.Init.(*ast.AssignStmt)
		call := assignment.Rhs[0].(*ast.CallExpr)
		selected = call.Args[0]
	case "assignment index":
		assignment := fn.Body.List[0].(*ast.AssignStmt)
		selected = assignment.Lhs[0].(*ast.IndexExpr).Index
	}
	for {
		parenthesized, ok := selected.(*ast.ParenExpr)
		if !ok {
			break
		}
		selected = parenthesized.X
	}
	literal, ok := selected.(*ast.BasicLit)
	if !ok || literal.Value != want {
		t.Fatalf("selected expression value=%T(%v), want literal %q; source=%s", selected, selected, want, source)
	}
}

func TestReplaceConstructRejectsStatementInjectionIntoExpressionWithoutWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() int {\n\treturn value\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "value", "if true { return 1 }", ConstructOptions{ConstructPath: "0.e0.0"})
	if err == nil {
		t.Fatal("statement snippet was accepted for an expression slot")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatal("invalid replacement changed the file")
	}
}

func TestReplaceConstructRejectsTypeSyntaxInExpressionSlot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() any {\n\treturn value\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "value", "[]int", ConstructOptions{ConstructPath: "0.e0"})
	if err == nil || !strings.Contains(err.Error(), "type expression") {
		t.Fatalf("type syntax was accepted as an expression replacement: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("rejected type syntax changed the file")
	}
}

func TestConstructCandidatesDoNotCrossFunctionLiteralOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() {\n\touter(func() { hidden() })\n\tkeep()\n}\nfunc hiddenOwner() { hidden() }\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "hidden()", "visible()", ConstructOptions{})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("nested function literal body leaked into outer owner: %v", err)
	}
	afterFailure, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterFailure) != string(before) {
		t.Fatal("rejected nested-owner replacement changed the file")
	}
	if _, replaceErr := ReplaceConstruct(context.Background(), path, "run", backend.ConstructStatementOrExpr, "keep()", "updated()", ConstructOptions{ConstructPath: "1"}); replaceErr != nil {
		t.Fatalf("replace outer statement: %v", replaceErr)
	}
	afterSuccess, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(afterSuccess), "outer(func() { hidden() })") || !strings.Contains(string(afterSuccess), "updated()") {
		t.Fatalf("outer edit changed nested closure or missed its target: %s", afterSuccess)
	}
}

func TestRemoveConstructOnlyDeletesDirectStatements(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() int {\n\treturn value + 1\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _, refusal, err := Remove(context.Background(), path, RemoveOptions{Kind: string(backend.ConstructStatementOrExpr), InFunction: "run", Discriminator: "value"}, false)
	if err == nil || !strings.Contains(err.Error(), "cannot remove expression slot") || refusal != "" {
		t.Fatalf("expression deletion should be refused: refusal=%q err=%v", refusal, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("expression deletion changed the file")
	}
}

func TestRemoveConstructUniqueCallUsesDirectStatementTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() {\n\tbefore()\n\trelease()\n\tafter()\n}\n")
	_, _, refusal, err := Remove(context.Background(), path, RemoveOptions{Kind: string(backend.ConstructStatementOrExpr), InFunction: "run", Discriminator: "release()"}, false)
	if err != nil || refusal != "" {
		t.Fatalf("remove unique direct call: refusal=%q err=%v", refusal, err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated), "release()") || !strings.Contains(string(updated), "before()") || !strings.Contains(string(updated), "after()") {
		t.Fatalf("direct statement removal changed wrong syntax: %s", updated)
	}
}

func TestInsertConstructRejectsStatementsEscapingWrapper(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() {\n\tkeep()\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = InsertConstruct(context.Background(), path, "next()\n}\nfunc injected() {}\nfunc _() {", InsertConstructOptions{
		Kind: backend.ConstructStatementOrExpr, Function: "run", Anchor: "end",
	})
	if err == nil {
		t.Fatalf("escaping declaration was not rejected: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("rejected snippet changed target file: %s", after)
	}
}

func TestInsertConstructAmbiguityRevisionBoundRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	writeConstructFixture(t, path, "package p\nfunc run() {\n\tflush()\n\tflush()\n}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	base := InsertConstructOptions{Kind: backend.ConstructStatementOrExpr, Function: "run", Anchor: "before", Discriminator: "flush()"}
	_, err = InsertConstruct(context.Background(), path, "audit()", base)
	revision := goReadRevision(before)
	if err == nil || !strings.Contains(err.Error(), "ambiguous statement anchor") || !strings.Contains(err.Error(), "construct_path") || !strings.Contains(err.Error(), revision) {
		t.Fatalf("ambiguity did not provide revision-bound candidates: %v", err)
	}
	selected := base
	selected.ConstructPath = "0"
	_, err = InsertConstruct(context.Background(), path, "audit()", selected)
	if err == nil || !strings.Contains(err.Error(), "construct_path requires its matching expected_revision") {
		t.Fatalf("path without expected_revision was not refused: %v", err)
	}
	selected.ExpectedRevision = "stale"
	_, err = InsertConstruct(context.Background(), path, "audit()", selected)
	if err == nil || !strings.Contains(err.Error(), "expected_revision does not match") {
		t.Fatalf("stale expected_revision was not refused: %v", err)
	}
	afterRefusals, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterRefusals) != string(before) {
		t.Fatal("refused selector retry changed the target file")
	}
	selected.ExpectedRevision = revision
	_, err = InsertConstruct(context.Background(), path, "audit()", selected)
	if err != nil {
		t.Fatalf("revision-bound retry failed: %v", err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", updated, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	first := fn.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
	name, ok := first.Fun.(*ast.Ident)
	if !ok || name.Name != "audit" {
		t.Fatalf("revision-bound retry inserted at the wrong target: %s", updated)
	}
}

func TestStatementInsertionAnchorsPreserveExistingCalls(t *testing.T) {
	source := []byte("package p\nfunc run() {\n\tflush()\n\treturn\n}\n")
	for _, intent := range []InsertStatementIntent{
		{Function: "run", Anchor: "before", Discriminator: "flush()", Source: "record()"},
		{Function: "run", Anchor: "start", Source: "firstRecord()"},
		{Function: "run", Anchor: "end", Source: "lastRecord()"},
		{Function: "run", Anchor: "before_return", Source: "returnRecord()"},
		{Function: "run", Anchor: "after", Discriminator: "flush()", Source: "afterFlush()"},
	} {
		plan, err := PlanInsertStatement(source, intent)
		if err != nil {
			t.Fatalf("anchor %q: %v", intent.Anchor, err)
		}
		source = plan.Source
	}
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Body.List) != 7 {
		t.Fatalf("got %d direct statements, want 7: %s", len(fn.Body.List), source)
	}
	for _, name := range []string{"flush", "record", "firstRecord", "lastRecord", "returnRecord", "afterFlush"} {
		if !strings.Contains(string(source), name+"()") {
			t.Fatalf("existing or inserted call %q missing: %s", name, source)
		}
	}
}

func writeConstructFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
