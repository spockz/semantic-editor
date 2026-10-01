// Package astedit_test verifies extraction behavior and fail-closed safety boundaries.
package astedit_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
)

func TestPlanConstantExtractionPreservesContextualTypeAndExactRange(t *testing.T) {
	source := []byte("package p\nfunc F() int { return 3 * 60 }\n")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "constant.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Uses: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("type-check source: %v", err)
	}
	var selected ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		if ret, ok := node.(*ast.ReturnStmt); ok {
			selected = ret.Results[0]
		}
		return true
	})
	plan, err := astedit.PlanConstantExtraction(source, fset, file, info, pkg.Scope(), selected, "retryWindow")
	if err != nil {
		t.Fatalf("PlanConstantExtraction(): %v", err)
	}
	if plan.Declaration != "const retryWindow int = 3 * 60" || plan.Replacement != "retryWindow" || plan.Type != "int" || plan.Value != "180" {
		t.Fatalf("unexpected constant plan: %#v", plan)
	}
	if got := string(source[plan.Start:plan.End]); got != "3 * 60" {
		t.Fatalf("plan selected %q, want exact expression", got)
	}
}

func TestPlanConstantExtractionRefusesRuntimeAndIotaExpressions(t *testing.T) {
	t.Run("runtime expression has complete valid type facts", func(t *testing.T) {
		source := "package p\nfunc F(input int) int { return input + 1 }\n"
		fset, file, info, pkg, selected := checkedConstantSelection(t, source, func(node ast.Node) ast.Expr {
			if ret, ok := node.(*ast.ReturnStmt); ok {
				return ret.Results[0]
			}
			return nil
		})
		if info.Types[selected].Value != nil {
			t.Fatalf("runtime selection unexpectedly has constant value %v", info.Types[selected].Value)
		}
		if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), selected, "candidate"); err == nil {
			t.Fatal("PlanConstantExtraction() accepted runtime expression")
		}
	})
	t.Run("valid local iota constant is refused", func(t *testing.T) {
		source := "package p\nfunc F() int { const (value = iota + 1); return value }\n"
		fset, file, info, pkg, selected := checkedConstantSelection(t, source, func(node ast.Node) ast.Expr {
			if spec, ok := node.(*ast.ValueSpec); ok {
				return spec.Values[0]
			}
			return nil
		})
		if info.Types[selected].Value == nil {
			t.Fatal("valid local iota expression has no constant value")
		}
		if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), selected, "candidate"); err == nil || !strings.Contains(err.Error(), "context-dependent") {
			t.Fatalf("PlanConstantExtraction() error = %v, want context-dependent refusal", err)
		}
	})
	t.Run("local constant dependency is refused", func(t *testing.T) {
		source := "package p\nfunc F() int { const multiplier = 2; return multiplier * 3 }\n"
		fset, file, info, pkg, selected := checkedConstantSelection(t, source, func(node ast.Node) ast.Expr {
			if ret, ok := node.(*ast.ReturnStmt); ok {
				return ret.Results[0]
			}
			return nil
		})
		if info.Types[selected].Value == nil {
			t.Fatal("local constant expression has no constant value")
		}
		if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), selected, "candidate"); err == nil || !strings.Contains(err.Error(), "unsupported or context-dependent") {
			t.Fatalf("PlanConstantExtraction() error = %v, want local dependency refusal", err)
		}
	})
}

func checkedConstantSelection(t *testing.T, source string, selectExpression func(ast.Node) ast.Expr) (*token.FileSet, *ast.File, *types.Info, *types.Package, ast.Expr) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "constant.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Uses: make(map[*ast.Ident]types.Object), Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("type-check source: %v", err)
	}
	var selected ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		if selected == nil {
			selected = selectExpression(node)
		}
		return selected == nil
	})
	if selected == nil {
		t.Fatal("selected expression was not found")
	}
	return fset, file, info, pkg, selected
}

func TestExtractFunctionMovesCallsAndInfersParameters(t *testing.T) {
	file := setupTestFile(t, `package testpkg

import "fmt"

func Emit( prefix string,count int){
fmt.Println(prefix)
fmt.Printf("%d",count)
fmt.Println("after")
}

`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source before extraction: %v", err)
	}
	diff, err := astedit.ExtractFunction(context.Background(), file, "Emit", "writeSummary", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 7})
	if err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	got := string(data)
	if string(before) == got {
		t.Fatal("source was not normalized during extraction")
	}
	wantDiff := astedit.UnifiedDiff(filepath.Base(file), string(before), got)
	if diff != wantDiff {
		t.Fatalf("ExtractFunction() diff did not describe persisted source\nwant:\n%s\ngot:\n%s", wantDiff, diff)
	}
	for _, want := range []string{
		"func Emit(prefix string, count int) {\n\twriteSummary(prefix, count)",
		"func writeSummary(prefix string, count int) {\n\tfmt.Println(prefix)\n\tfmt.Printf(\"%d\", count)",
		"fmt.Println(\"after\")",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("extracted source missing %q:\n%s", want, got)
		}
	}
}

func TestExtractFunctionReturnsMultipleLiveOuts(t *testing.T) {
	file := setupTestFile(t, `package testpkg

func Emit() {
	left, right := 1, "summary"
	println("inside")
	println(left, right)
}
`)
	if _, err := astedit.ExtractFunction(context.Background(), file, "Emit", "writeSummary", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 4, EndLine: 5}); err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"left, right := writeSummary()",
		"func writeSummary() (int, string)",
		"left, right := 1, \"summary\"",
		"return left, right",
		"println(left, right)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("extracted source missing %q:\n%s", want, got)
		}
	}
}

func TestExtractMethodPreservesReceiverAndParameters(t *testing.T) {
	file := setupTestFile(t, `package testpkg

import "fmt"

type Logger struct { prefix string }

func (l *Logger) Emit(message string) {
	fmt.Println(l.prefix, message)
	fmt.Println("after")
}
`)
	if _, err := astedit.ExtractFunction(context.Background(), file, "(*Logger).Emit", "writeMessage", astedit.ExtractOptions{Kind: astedit.ExtractMethod, StartLine: 8, EndLine: 8}); err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"l.writeMessage(message)",
		"func (l *Logger) writeMessage(message string) {\n\tfmt.Println(l.prefix, message)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("extracted source missing %q:\n%s", want, got)
		}
	}
}

func TestExtractFunctionRejectsUnsafeSelectionsWithoutWriting(t *testing.T) {
	cases := []struct {
		name, source, wantError string
		start, end              int
	}{
		{name: "assignment data flow", source: "package p\nfunc F(value int) {\n value = value + 1\n}\n", start: 3, end: 3, wantError: "only one leading := declaration"},
		{name: "local variable", source: "package p\nimport \"fmt\"\nfunc F() {\n local := 1\n fmt.Println(local)\n}\n", start: 5, end: 5, wantError: `identifier "local"`},
		{name: "control flow", source: "package p\nfunc F() {\n return\n}\n", start: 3, end: 3, wantError: "not a call or supported local declaration"},
		{name: "comment ownership", source: "package p\nimport \"fmt\"\nfunc F() {\n // keep with this call\n fmt.Println(1)\n}\n", start: 5, end: 5, wantError: "comments inside the owner"},
		{name: "ambiguous same-line selection", source: "package p\nimport \"fmt\"\nfunc F() { fmt.Println(1); fmt.Println(2) }\n", start: 3, end: 3, wantError: "matches multiple statements"},
		{name: "forwarded parameter address taken before selection", source: "package p\n\nfunc F(value int) {\n ptr := &value\n println(value)\n _ = ptr\n}\n", start: 5, end: 5, wantError: "address-taken in the owner body"},
		{name: "returned local address taken in selected region", source: "package p\n\nfunc F() {\n result := 42\n println(&result)\n println(result)\n}\n", start: 4, end: 5, wantError: "result local \"result\" is address-taken"},
		{name: "forwarded parameter captured by closure", source: "package p\n\nfunc F(value int) {\n use := func() { println(value) }\n println(value)\n _ = use\n}\n", start: 5, end: 5, wantError: "captured by a closure"},
		{name: "direct recover call", source: "package p\n\nfunc F() {\n recover()\n}\n", start: 4, end: 4, wantError: "direct recover() calls"},
		{name: "implicit pointer receiver on value parameter", source: "package p\n\ntype Box struct{}\nfunc (b *Box) Touch() {}\nfunc F(b Box) {\n b.Touch()\n}\n", start: 6, end: 6, wantError: "implicit pointer receiver"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := setupTestFile(t, tc.source)
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read source before: %v", err)
			}
			_, err = astedit.ExtractFunction(context.Background(), file, "F", "helper", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: tc.start, EndLine: tc.end})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ExtractFunction() error = %v, want substring %q", err, tc.wantError)
			}
			after, readErr := os.ReadFile(filepath.Clean(file))
			if readErr != nil {
				t.Fatalf("read source after: %v", readErr)
			}
			if string(after) != string(before) {
				t.Fatalf("rejected extraction changed source:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestExtractFunctionReturnsOneLiveLocal(t *testing.T) {
	file := setupTestFile(t, `package testpkg

import "fmt"

func BuildValue() {
	result := 42
	fmt.Println("during", result)
	fmt.Println("after", result)
}
`)
	if _, err := astedit.ExtractFunction(context.Background(), file, "BuildValue", "buildValue", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 7}); err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"result := buildValue()",
		"fmt.Println(\"after\", result)",
		"func buildValue() int {\n\tresult := 42\n\tfmt.Println(\"during\", result)\n\treturn result",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("extracted source missing %q:\n%s", want, got)
		}
	}
}

func TestExtractFunctionRejectsPackageCollisionAndCompositeLiterals(t *testing.T) {
	t.Run("package-wide helper collision", func(t *testing.T) {
		file := setupTestFile(t, "package testpkg\nfunc F() {\n println(1)\n}\n")
		other := filepath.Join(filepath.Dir(file), "other.go")
		if err := os.WriteFile(other, []byte("package testpkg\nfunc helper() {}\n"), 0o600); err != nil {
			t.Fatalf("write sibling source: %v", err)
		}
		before, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		_, err = astedit.ExtractFunction(context.Background(), file, "F", "helper", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 3, EndLine: 3})
		if err == nil || !strings.Contains(err.Error(), `package declaration "helper" already exists`) {
			t.Fatalf("ExtractFunction() error = %v, want package-wide collision", err)
		}
		after, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("package collision refusal changed target source")
		}
	})

	t.Run("composite literal key", func(t *testing.T) {
		file := setupTestFile(t, `package testpkg

import "fmt"

func F(key string) {
	fmt.Println(map[string]int{key: 1})
}
`)
		_, err := astedit.ExtractFunction(context.Background(), file, "F", "helper", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 6})
		if err == nil || !strings.Contains(err.Error(), "composite literals") {
			t.Fatalf("ExtractFunction() error = %v, want composite literal refusal", err)
		}
	})
}

func TestExtractFunctionPreservesDefinedOutputType(t *testing.T) {
	file := setupTestFile(t, `package testpkg

type Distance int

func Emit(distance Distance) {
	result := distance
	println(result)
}
`)
	if _, err := astedit.ExtractFunction(context.Background(), file, "Emit", "writeDistance", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 6}); err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"result := writeDistance(distance)", "func writeDistance(distance Distance) Distance", "return result"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("extracted source missing %q:\n%s", want, data)
		}
	}
}

func TestExtractFunctionRejectsEffectfulMultipleOutputWithoutWriting(t *testing.T) {
	file := setupTestFile(t, `package testpkg

func Emit() {
	left, right := compute(), "ready"
	println(left, right)
}

func compute() int { return 1 }
`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = astedit.ExtractFunction(context.Background(), file, "Emit", "writeResult", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 4, EndLine: 4})
	if err == nil || !strings.Contains(err.Error(), "cannot infer a safe return type") {
		t.Fatalf("ExtractFunction() error = %v, want unsupported result-type refusal", err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("rejected extraction changed source:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestPlanConstantExtractionRefusesNamedConstantType(t *testing.T) {
	source := "package p\ntype Duration int\nconst interval Duration = 1\nfunc F() Duration { return interval }\n"
	fset, file, info, pkg, selected := checkedConstantSelection(t, source, func(node ast.Node) ast.Expr {
		if ret, ok := node.(*ast.ReturnStmt); ok {
			return ret.Results[0]
		}
		return nil
	})
	if info.Types[selected].Value == nil {
		t.Fatal("named constant expression has no resolved value")
	}
	if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), selected, "candidate"); err == nil || !strings.Contains(err.Error(), "named and non-basic") {
		t.Fatalf("PlanConstantExtraction() error = %v, want named-type refusal", err)
	}
}

func TestExtractConstantRejectsStaleRevisionWithoutWriting(t *testing.T) {
	before := "package sample\nfunc Retry() int { return 3 * 60 }\n"
	path := filepath.Join(t.TempDir(), "retry.go")
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := astedit.ExtractConstant(context.Background(), path, "Retry", "retryWindow", astedit.ExtractConstantOptions{
		Discriminator: "3 * 60", ExpectedRevision: "stale",
	})
	if err == nil || !strings.Contains(err.Error(), "revision mismatch") {
		t.Fatalf("ExtractConstant() error = %v, want stale-revision refusal", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("stale revision changed source:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
