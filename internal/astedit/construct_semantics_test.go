// This file proves that construct edits select syntax slots precisely and reject unsafe or ambiguous spans before writing.
package astedit

import (
	"context"
	"go/ast"
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
			assertConstructTestLiteral(t, updated, test.name, test.replacement)
		})
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

func writeConstructFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
