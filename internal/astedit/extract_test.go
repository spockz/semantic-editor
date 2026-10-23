// Package astedit_test verifies extraction behavior and fail-closed safety boundaries.
package astedit_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/astedit"
)

func TestPlanConstantExtractionPreservesContextualTypeAndExactRange(t *testing.T) {
	t.Parallel()
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
	plan, err := astedit.PlanConstantExtraction(source, fset, file, info, pkg.Scope(), []ast.Expr{selected}, "retryWindow")
	if err != nil {
		t.Fatalf("PlanConstantExtraction(): %v", err)
	}
	if plan.Declaration != "const retryWindow int = 3 * 60" || plan.Replacement != "retryWindow" || plan.Type != "int" || plan.Value != "180" {
		t.Fatalf("unexpected constant plan: %#v", plan)
	}
	if len(plan.Occurrences) != 1 {
		t.Fatalf("plan occurrences = %d, want 1", len(plan.Occurrences))
	}
	if got := string(source[plan.Occurrences[0].Start:plan.Occurrences[0].End]); got != "3 * 60" {
		t.Fatalf("plan selected %q, want exact expression", got)
	}
}

func TestPlanConstantExtractionTracksMultipleEqualOccurrencesInSourceOrder(t *testing.T) {
	t.Parallel()
	source := []byte("package p\nfunc F() int { return 3 * 60 + 3 * 60 }\n")
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
	selected := make([]ast.Expr, 0, 2)
	ast.Inspect(file, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		start, end := fset.Position(expression.Pos()).Offset, fset.Position(expression.End()).Offset
		if start >= 0 && end <= len(source) && string(source[start:end]) == "3 * 60" {
			selected = append(selected, expression)
		}
		return true
	})
	plan, err := astedit.PlanConstantExtraction(source, fset, file, info, pkg.Scope(), selected, "retryWindow")
	if err != nil {
		t.Fatalf("PlanConstantExtraction(): %v", err)
	}
	if len(plan.Occurrences) != 2 {
		t.Fatalf("plan occurrence count = %d, want 2", len(plan.Occurrences))
	}
	for _, occurrence := range plan.Occurrences {
		if got := string(source[occurrence.Start:occurrence.End]); got != "3 * 60" {
			t.Errorf("planned occurrence = %q, want exact expression", got)
		}
	}
	if plan.Value != "180" || plan.Type != "untyped int" {
		t.Fatalf("plan constant = %s %s, want untyped int 180", plan.Type, plan.Value)
	}
}

func TestPlanConstantExtractionRefusesRuntimeAndIotaExpressions(t *testing.T) {
	t.Parallel()
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
		if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), []ast.Expr{selected}, "candidate"); err == nil {
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
		if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), []ast.Expr{selected}, "candidate"); err == nil || !strings.Contains(err.Error(), "context-dependent") {
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
		if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), []ast.Expr{selected}, "candidate"); err == nil || !strings.Contains(err.Error(), "unsupported or context-dependent") {
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
	t.Parallel()
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

func TestExtractFunctionSelectsRevisionBoundStatementPaths(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

func Emit() {
	printOne()
	printTwo()
	printThree()
}

func printOne() {}
func printTwo() {}
func printThree() {}
`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	digest := sha256.Sum256(before)
	revision := hex.EncodeToString(digest[:])
	if _, extractErr := astedit.ExtractFunction(context.Background(), file, "Emit", "writeTail", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, ConstructPath: "1", EndConstructPath: "2", ExpectedRevision: revision,
	}); extractErr != nil {
		t.Fatalf("ExtractFunction() with statement paths: %v", extractErr)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted source: %v", err)
	}
	for _, want := range []string{
		"func Emit() {\n\tprintOne()\n\twriteTail()",
		"func writeTail() {\n\tprintTwo()\n\tprintThree()",
	} {
		if !strings.Contains(string(after), want) {
			t.Errorf("path-based extraction missing %q:\n%s", want, after)
		}
	}

	t.Run("switch-case assignment returns live-out", func(t *testing.T) {
		file := setupTestFile(t, `package testpkg

func Assign(enabled bool) int {
	count := 0
	switch enabled {
	case true:
		count = count + 1
	}
	return count
}
`)
		before, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read source: %v", err)
		}
		digest := sha256.Sum256(before)
		revision := hex.EncodeToString(digest[:])
		if _, extractErr := astedit.ExtractFunction(context.Background(), file, "Assign", "increment", astedit.ExtractOptions{
			Kind: astedit.ExtractKindFunction, ConstructPath: "1.0.0", ExpectedRevision: revision,
		}); extractErr != nil {
			t.Fatalf("ExtractFunction() switch-case assignment: %v", extractErr)
		}
		after, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read extracted source: %v", err)
		}
		for _, want := range []string{
			"count = increment(count)",
			"func increment(count int) int",
		} {
			if !strings.Contains(string(after), want) {
				t.Errorf("switch-case extraction missing %q:\n%s", want, after)
			}
		}
	})
	testExtractFunctionSwitchCaseNamedParameter(t)
}

func TestExtractFunctionSelectsRevisionBoundNestedBlockStatementPaths(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

func Emit(enabled bool) int {
	count := 0
	if enabled {
		count = count + 1
		count = count + 2
	}
	return count
}
`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	digest := sha256.Sum256(before)
	revision := hex.EncodeToString(digest[:])
	if _, extractErr := astedit.ExtractFunction(context.Background(), file, "Emit", "addCount", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, ConstructPath: "1.0.0", EndConstructPath: "1.0.1", ExpectedRevision: revision,
	}); extractErr != nil {
		t.Fatalf("ExtractFunction() with nested paths: %v", extractErr)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted source: %v", err)
	}
	for _, want := range []string{
		"if enabled {\n\t\tcount = addCount(count)",
		"func addCount(count int) int {\n\tcount = count + 1\n\tcount = count + 2\n\treturn count",
	} {
		if !strings.Contains(string(after), want) {
			t.Errorf("nested extraction missing %q:\n%s", want, after)
		}
	}
}

func TestExtractFunctionNestedBlocksPreserveLiveOuts(t *testing.T) {
	t.Parallel()

	t.Run("read later in the selected block", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, `package testpkg

func Emit(enabled bool) {
	count := 0
	if enabled {
		count = count + 1
		println(count)
	}
}
`)
		extractAndAssertLiveOut(t, file, "1.0.0")
	})

	t.Run("read by the enclosing loop condition", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, `package testpkg

func Emit(limit int) {
	count := 0
	for count < limit {
		count = count + 1
	}
}
`)
		extractAndAssertLiveOut(t, file, "1.0.0")
	})

	t.Run("read at the start of the next counted-loop iteration", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, `package testpkg

func Emit() {
	count := 0
	for i := 0; i < 3; i++ {
		println(count)
		count = count + 1
	}
}
`)
		extractAndAssertLiveOut(t, file, "1.0.1")
	})

	t.Run("read at the start of the next range-loop iteration", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, `package testpkg

func Emit() {
	count := 0
	for range []int{1, 2, 3} {
		println(count)
		count = count + 1
	}
}
`)
		extractAndAssertLiveOut(t, file, "1.0.1")
	})

}

func TestExtractFunctionNestedElsePathMatchesInspection(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

func Emit(enabled bool) int {
	count := 0
	if enabled {
		count = count + 1
	} else {
		count = count + 2
	}
	return count
}
`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	digest := sha256.Sum256(before)
	revision := hex.EncodeToString(digest[:])
	if _, err := astedit.ExtractFunction(context.Background(), file, "Emit", "incrementElseCount", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, ConstructPath: "1.1.0.0", EndConstructPath: "1.1.0.0", ExpectedRevision: revision,
	}); err != nil {
		t.Fatalf("ExtractFunction() with inspected else path: %v", err)
	}
}

func TestExtractFunctionLoopLocalsStayInsideTheHelper(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, source string }{
		{"counted", "package testpkg\nfunc Emit() { for i := 0; i < 3; i++ { value := 1; println(value) } }"},
		{"range", "package testpkg\nfunc Emit() { for range []int{1, 2, 3} { value := 1; println(value) } }"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			file := setupTestFile(t, test.source)
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			digest := sha256.Sum256(before)
			revision := hex.EncodeToString(digest[:])
			if _, extractErr := astedit.ExtractFunction(context.Background(), file, "Emit", "printValue", astedit.ExtractOptions{
				Kind: astedit.ExtractKindFunction, ConstructPath: "0.0.0", EndConstructPath: "0.0.1", ExpectedRevision: revision,
			}); extractErr != nil {
				t.Fatalf("ExtractFunction() for loop-local value: %v", extractErr)
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read extracted source: %v", err)
			}
			if !strings.Contains(string(after), "printValue()") || !strings.Contains(string(after), "func printValue() {") {
				t.Errorf("loop-local extraction did not produce a void helper call:\n%s", after)
			}
		})
	}
}

func extractAndAssertLiveOut(t *testing.T, file, path string) {
	t.Helper()
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	digest := sha256.Sum256(before)
	revision := hex.EncodeToString(digest[:])
	if _, extractErr := astedit.ExtractFunction(context.Background(), file, "Emit", "incrementCount", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, ConstructPath: path, EndConstructPath: path, ExpectedRevision: revision,
	}); extractErr != nil {
		t.Fatalf("ExtractFunction() with nested live-out: %v", extractErr)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted source: %v", err)
	}
	want := "count = incrementCount(count)"
	if !strings.Contains(string(after), want) {
		t.Fatalf("nested extraction missing %q:\n%s", want, after)
	}
}

func TestExtractFunctionRejectsStaleOrCrossBlockStatementPathsWithoutWriting(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		start string
		end   string
	}{
		{name: "stale revision", start: "0"},
		{name: "different nested blocks", start: "0.0.0", end: "0.1.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := setupTestFile(t, `package testpkg

func Emit() {
	if true {
		printOne()
	} else {
		printTwo()
	}
}

func printOne() {}
func printTwo() {}
`)
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			revision := "stale"
			if test.name != "stale revision" {
				digest := sha256.Sum256(before)
				revision = hex.EncodeToString(digest[:])
			}
			_, err = astedit.ExtractFunction(context.Background(), file, "Emit", "writeTail", astedit.ExtractOptions{
				Kind: astedit.ExtractKindFunction, ConstructPath: test.start, EndConstructPath: test.end, ExpectedRevision: revision,
			})
			if err == nil {
				t.Fatal("ExtractFunction() unexpectedly accepted the invalid selector")
			}
			after, readErr := os.ReadFile(file)
			if readErr != nil {
				t.Fatalf("read source after refusal: %v", readErr)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("refused extraction changed source:\n%s", after)
			}
		})
	}
}

func TestExtractFunctionReturnsMultipleLiveOuts(t *testing.T) {
	t.Parallel()
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

func TestExtractFunctionUpdatesBasicOwnerParameter(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

func Update(count int) {
	count = count + 1
	println(count)
}
`)
	if _, err := astedit.ExtractFunction(context.Background(), file, "Update", "increment", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 4, EndLine: 4}); err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"count = increment(count)",
		"func increment(count int) int {\n\tcount = count + 1\n\treturn count",
		"println(count)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("extracted source missing %q:\n%s", want, got)
		}
	}
}

func TestExtractMethodPreservesReceiverAndParameters(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	cases := []struct {
		name, source, wantError string
		start, end              int
	}{
		{name: "compound assignment", source: "package p\nfunc F(value int) {\n value += 1\n}\n", start: 3, end: 3, wantError: "only short declarations and plain assignments are supported"},
		{name: "assignment parameter address taken", source: "package p\nfunc F(value int) {\n ptr := &value\n value = 1\n println(value)\n _ = ptr\n}\n", start: 4, end: 4, wantError: "address-taken values"},
		{name: "assignment parameter captured", source: "package p\nfunc F(value int) {\n use := func() { println(value) }\n value = 1\n println(value)\n _ = use\n}\n", start: 4, end: 4, wantError: "closure-captured values"},
		{name: "control flow", source: "package p\nfunc F() {\n return\n}\n", start: 3, end: 3, wantError: ("selected statement on line")},
		{name: "variadic owner", source: "package p\nfunc F(values ...int) {\n println(values[0])\n}\n", start: 3, end: 3, wantError: "variadic function"},
		{name: "comment ownership", source: "package p\nimport \"fmt\"\nfunc F() {\n // keep with this call\n fmt.Println(1)\n}\n", start: 5, end: 5, wantError: "comments outside the selected region"},
		{name: "ambiguous same-line selection", source: "package p\nimport \"fmt\"\nfunc F() { fmt.Println(1); fmt.Println(2) }\n", start: 3, end: 3, wantError: "matches multiple statements"},
		{name: "forwarded parameter address taken before selection", source: "package p\n\nfunc F(value int) {\n ptr := &value\n println(value)\n _ = ptr\n}\n", start: 5, end: 5, wantError: "address-taken values"},
		{name: "returned local address taken in selected region", source: "package p\n\nfunc F() {\n result := 42\n println(&result)\n println(result)\n}\n", start: 4, end: 5, wantError: ("address-taken values in the owner")},
		{name: "forwarded parameter captured by closure", source: "package p\n\nfunc F(value int) {\n use := func() { println(value) }\n println(value)\n _ = use\n}\n", start: 5, end: 5, wantError: "closure-captured values"},
		{name: "direct recover call", source: "package p\n\nfunc F() {\n recover()\n}\n", start: 4, end: 4, wantError: "direct recover() calls"},
		{name: "parenthesized recover in assignment", source: "package p\n\nfunc F(value int) {\n value = ((recover)()).(int)\n println(value)\n}\n", start: 4, end: 4, wantError: "direct recover() calls"},
		{name: "implicit pointer receiver on value parameter", source: "package p\n\ntype Box struct{}\nfunc (b *Box) Touch() {}\nfunc F(b Box) {\n b.Touch()\n}\n", start: 6, end: 6, wantError: "implicit pointer-receiver calls"},
		{name: "pointer receiver escape before selected assignment", source: "package p\n\ntype Box struct{}\nvar saved *Box\nfunc (b *Box) Expose() { saved = b }\nfunc F(b Box, next Box) {\n b.Expose()\n b = next\n println(b)\n}\n", start: 8, end: 8, wantError: "implicit pointer-receiver calls"},
		{name: "array slice aliases variable before selected assignment", source: "package p\n\ntype Box [1]int\nfunc F(next Box) int {\n var b Box\n p := b[:]\n b = next\n result := p[0]\n return result + b[0]\n}\n", start: 7, end: 8, wantError: "array slices that alias copied variables"},
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

func TestExtractFunctionRejectsPointerGenericReceiverWithoutWriting(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

type Box[T any] struct{}

func (b *Box[T]) F() {
	println(1)
}
`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source before: %v", err)
	}
	_, err = astedit.ExtractFunction(context.Background(), file, "(*Box).F", "helper", astedit.ExtractOptions{Kind: astedit.ExtractMethod, StartLine: 6, EndLine: 6})
	if err == nil || !strings.Contains(err.Error(), "generic receiver method") {
		t.Fatalf("ExtractFunction() error = %v, want generic receiver refusal", err)
	}
	after, readErr := os.ReadFile(file)
	if readErr != nil || string(after) != string(before) {
		t.Fatalf("rejected extraction changed source:\nbefore:\n%s\nafter:\n%s\nerror: %v", before, after, readErr)
	}
}

func TestExtractFunctionReturnsOneLiveLocal(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
		if err == nil || !strings.Contains(err.Error(), "composite literal") {
			t.Fatalf("ExtractFunction() error = %v, want composite literal refusal", err)
		}
	})
}

func TestExtractFunctionPreservesDefinedOutputType(t *testing.T) {
	t.Parallel()
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

func TestExtractFunctionRejectsTypeErrorsWithoutWriting(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

func consume(value int) {}

func Emit() {
	consume("invalid")
}
`)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source before extraction: %v", err)
	}
	_, extractErr := astedit.ExtractFunction(context.Background(), file, "Emit", "writeResult", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 6})
	if extractErr == nil || !strings.Contains(extractErr.Error(), "load typed extraction input") || !strings.Contains(extractErr.Error(), "cannot use") {
		t.Fatalf("ExtractFunction() error = %v, want original package compiler rejection", extractErr)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source after extraction: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("type-check failure changed source:\n%s", after)
	}
}

func TestExtractFunctionSupportsEffectfulMultipleOutput(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, `package testpkg

func Emit() {
	left, right := compute(), "ready"
	println(left, right)
}

func compute() int { return 1 }
`)
	_, err := astedit.ExtractFunction(context.Background(), file, "Emit", "writeResult", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 4, EndLine: 4})
	if err != nil {
		t.Fatalf("ExtractFunction() error = %v", err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"left, right := writeResult()", "func writeResult() (int, string)", `left, right := compute(), "ready"`, "return left, right"} {
		if !strings.Contains(string(after), want) {
			t.Errorf("extracted source missing %q:\n%s", want, after)
		}
	}
}

func TestPlanConstantExtractionRefusesNamedConstantType(t *testing.T) {
	t.Parallel()
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
	if _, err := astedit.PlanConstantExtraction([]byte(source), fset, file, info, pkg.Scope(), []ast.Expr{selected}, "candidate"); err == nil || !strings.Contains(err.Error(), "named and non-basic") {
		t.Fatalf("PlanConstantExtraction() error = %v, want named-type refusal", err)
	}
}

func TestExtractConstantRejectsStaleRevisionWithoutWriting(t *testing.T) {
	t.Parallel()
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

func TestExtractConstantRefusesSymlinkedPackageSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/symlinkconstant\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	selected := []byte("package sample\nfunc Retry() int { return 3 * 60 }\n")
	selectedPath := filepath.Join(root, "retry.go")
	if err := os.WriteFile(selectedPath, selected, 0o600); err != nil {
		t.Fatalf("write selected source: %v", err)
	}
	external := filepath.Join(t.TempDir(), "external_test.go")
	if err := os.WriteFile(external, []byte("package sample\nconst retryWindow = 1\n"), 0o600); err != nil {
		t.Fatalf("write external target: %v", err)
	}
	if err := os.Symlink(external, filepath.Join(root, "retry_test.go")); err != nil {
		t.Fatalf("create test symlink: %v", err)
	}
	_, err := astedit.ExtractConstant(context.Background(), selectedPath, "Retry", "limit", astedit.ExtractConstantOptions{Discriminator: "3 * 60"})
	if err == nil || !strings.Contains(err.Error(), "symlinked Go package source") {
		t.Fatalf("ExtractConstant() error = %v, want symlink refusal", err)
	}
	after, err := os.ReadFile(selectedPath)
	if err != nil {
		t.Fatalf("read selected source: %v", err)
	}
	if string(after) != string(selected) {
		t.Fatalf("symlink refusal changed selected source: %s", after)
	}
}

func TestExtractFunctionRejectsShadowedHelperNameWithoutWriting(t *testing.T) {
	t.Parallel()
	file := setupTestFile(t, "package testpkg\nfunc Update(helper func(int) int, value int) int {\n\tvalue = value + 1\n\treturn helper(value)\n}\n")
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source before extraction: %v", err)
	}
	_, err = astedit.ExtractFunction(context.Background(), file, "Update", "helper", astedit.ExtractOptions{Kind: astedit.ExtractKindFunction, StartLine: 3, EndLine: 3})
	if err == nil || !strings.Contains(err.Error(), "shadowed at the replacement site") {
		t.Fatalf("ExtractFunction() error = %v, want shadowed helper refusal", err)
	}
	after, readErr := os.ReadFile(file)
	if readErr != nil {
		t.Fatalf("read source after extraction: %v", readErr)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("shadowed helper refusal changed source:\n%s", after)
	}
}

func TestExtractFunctionMapRangeTypeBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("package named map type", func(t *testing.T) {
		t.Parallel()
		assertPackageNamedRangeExtraction(t, "type EntryMap map[int]string", "entries EntryMap", "for key, value := range entries { record(key, value) }", "func visitEntries(entries EntryMap)", "for key, value := range entries")
	})

	t.Run("function-local named map type", func(t *testing.T) {
		t.Parallel()
		assertLocalNamedRangeRefusal(t, "EntryMap map[int]int", "EntryMap{1: 2}", "for key, value := range entries { record(key, value) }")
	})
}

func TestExtractFunctionSliceRangeTypeBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("package named slice type", func(t *testing.T) {
		t.Parallel()
		assertPackageNamedRangeExtraction(t, "type Entries []int", "entries Entries", "for index, value := range entries { record(index, value) }", "func visitEntries(entries Entries)", "for index, value := range entries")
	})

	t.Run("function-local named slice type", func(t *testing.T) {
		t.Parallel()
		assertLocalNamedRangeRefusal(t, "Entries []int", "Entries{1}", "for index, value := range entries { record(index, value) }")
	})
}

func TestExtractFunctionSliceExpressionRangeTypeBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("package named slice with explicit bounds", func(t *testing.T) {
		t.Parallel()
		assertPackageNamedRangeExtraction(t, "type Entries []int; const low, high = 0, 1", "entries Entries", "for index, value := range entries[low:high] { record(index, value) }", "func visitEntries(entries Entries)", "for index, value := range entries[low:high]")
	})

	t.Run("function-local named slice", func(t *testing.T) {
		t.Parallel()
		assertLocalNamedRangeRefusal(t, "Entries []int", "Entries{1}", "for index, value := range entries[:] { record(index, value) }")
	})
}

func TestExtractFunctionRejectsUnsupportedSliceExpressionRangesWithoutWriting(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, source, wantError string
	}{
		{
			name:      "effectful lower bound",
			source:    "package testpkg\n\nfunc Emit(entries []int) {\n\tfor index := range entries[nextBound():] { println(index) }\n}\n\nfunc nextBound() int { return 0 }\n",
			wantError: "slice bounds must be pure compiler-resolved integer expressions",
		},
		{
			name:      "effectful maximum bound",
			source:    "package testpkg\n\nfunc Emit(entries []int) {\n\tfor index := range entries[:1:nextBound()] { println(index) }\n}\n\nfunc nextBound() int { return 1 }\n",
			wantError: "slice bounds must be pure compiler-resolved integer expressions",
		},
		{
			name:      "non-integer lower bound",
			source:    "package testpkg\n\nfunc Emit(entries []int, bound float64) {\n\tfor index := range entries[bound:] { println(index) }\n}\n",
			wantError: "must be integer",
		},
		{
			name:      "non-integer maximum bound",
			source:    "package testpkg\n\nfunc Emit(entries []int, bound float64) {\n\tfor index := range entries[:1:bound] { println(index) }\n}\n",
			wantError: "must be integer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			file := setupTestFile(t, tc.source)
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read source before extraction: %v", err)
			}
			_, err = astedit.ExtractFunction(context.Background(), file, "Emit", "visitEntries", astedit.ExtractOptions{
				Kind: astedit.ExtractKindFunction, StartLine: 4, EndLine: 4,
			})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ExtractFunction() error = %v, want %q", err, tc.wantError)
			}
			after, readErr := os.ReadFile(file)
			if readErr != nil || !bytes.Equal(after, before) {
				t.Fatalf("rejected extraction changed source:\nbefore:\n%s\nafter:\n%s\nerror: %v", before, after, readErr)
			}
		})
	}
}

func assertPackageNamedRangeExtraction(t *testing.T, typeDeclaration, ownerParameter, loop, expectedParameter, expectedLoop string) {
	t.Helper()
	source := "package testpkg\n\n" + typeDeclaration + "\n\nfunc Emit(" + ownerParameter + ") {\n\t" + loop + "\n}\n\nfunc record(any, any) {}\n"
	file := setupTestFile(t, source)
	if _, err := astedit.ExtractFunction(context.Background(), file, "Emit", "visitEntries", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 6,
	}); err != nil {
		t.Fatalf("ExtractFunction() for package named range type: %v", err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted source: %v", err)
	}
	for _, want := range []string{"visitEntries(entries)", expectedParameter, expectedLoop} {
		if !strings.Contains(string(after), want) {
			t.Errorf("extracted source missing %q:\n%s", want, after)
		}
	}
}

func assertLocalNamedRangeRefusal(t *testing.T, localTypeDeclaration, initializer, loop string) {
	t.Helper()
	source := "package testpkg\n\nfunc Emit() {\n\ttype " + localTypeDeclaration + "\n\tentries := " + initializer + "\n\t" + loop + "\n}\n\nfunc record(any, any) {}\n"
	file := setupTestFile(t, source)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read original source: %v", err)
	}
	_, extractErr := astedit.ExtractFunction(context.Background(), file, "Emit", "visitEntries", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, StartLine: 6, EndLine: 6,
	})
	if extractErr == nil || !strings.Contains(extractErr.Error(), "nameable from a package helper") {
		t.Fatalf("ExtractFunction() error = %v, want local range type refusal", extractErr)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source after refusal: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("local range type refusal changed source:\n%s", after)
	}
}

func TestExtractFunctionSliceExpressionRangeThreeIndexTypeBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("package named slice with three explicit bounds", func(t *testing.T) {
		t.Parallel()
		assertPackageNamedRangeExtraction(
			t,
			"type Entries []int; const low, high, max = 0, 1, 2",
			"entries Entries",
			"for index, value := range entries[low:high:max] { record(index, value) }",
			"func visitEntries(entries Entries)",
			"for index, value := range entries[low:high:max]",
		)
	})

	t.Run("omitted low bound with named slice", func(t *testing.T) {
		t.Parallel()
		assertPackageNamedRangeExtraction(
			t,
			"type Entries []int; const high, max = 1, 2",
			"entries Entries",
			"for index, value := range entries[:high:max] { record(index, value) }",
			"func visitEntries(entries Entries)",
			"for index, value := range entries[:high:max]",
		)
	})
}

func testExtractFunctionSwitchCaseNamedParameter(t *testing.T) {
	t.Helper()
	file := setupTestFile(t, (`package testpkg

func Assign(enabled bool, count int) int {
	switch enabled {
	case true:
		count = count + 1
	default:
		count = count - 1
	}
	return count
}
`))
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	digest := sha256.Sum256(before)
	revision := hex.EncodeToString(digest[:])
	if _, extractErr := astedit.ExtractFunction(context.Background(), file, "Assign", "increment", astedit.ExtractOptions{
		Kind: astedit.ExtractKindFunction, ConstructPath: "0.0.0", ExpectedRevision: revision,
	}); extractErr != nil {
		t.Fatalf("ExtractFunction() switch-case parameter assignment: %v", extractErr)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read extracted source: %v", err)
	}
	for _, want := range []string{
		"count = increment(count)",
		"func increment(count int) int",
		"default:\n\t\tcount = count - 1",
		"return count",
	} {
		if !strings.Contains(string(after), want) {
			t.Errorf("switch-case parameter extraction missing %q:\n%s", want, after)
		}
	}
}
