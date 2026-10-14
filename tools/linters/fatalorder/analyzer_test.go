// Package fatalorder_test tests the fatalorder analyzer and nolint suppression behavior.
package fatalorder_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spockz/semantic-editor/tools/linters/fatalorder"
)

func runAnalyzer(t *testing.T, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample_test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse test source: %v", err) //nolint:fatalorder // setup precondition
	}

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	pkg, _ := conf.Check("sample", fset, []*ast.File{file}, nil)
	if pkg == nil {
		pkg = types.NewPackage("sample", "sample")
	}

	var diagnostics []string
	report := func(d analysis.Diagnostic) {
		diagnostics = append(diagnostics, d.Message)
	}

	inspectPass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}
	inspectRes, err := inspect.Analyzer.Run(inspectPass)
	if err != nil {
		t.Fatalf("run inspect: %v", err) //nolint:fatalorder // setup precondition
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
		Report:    report,
		ResultOf:  map[*analysis.Analyzer]any{inspect.Analyzer: inspectRes},
	}

	_, err = fatalorder.Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run fatalorder: %v", err)
	}

	return diagnostics
}

func TestFatalOrder_ReportsNonTerminalFatalf(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	if true {
		t.Fatalf("first check")
	}
	if true {
		t.Fatalf("second check")
	}
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 1 {
		t.Fatalf("diagnostics count = %d, want 1: %v", len(diags), diags)
	}
}

func TestFatalOrder_ReportsFatalfBeforeErrorf(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	if true {
		t.Fatalf("first check")
	}
	if true {
		t.Errorf("second check")
	}
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 1 {
		t.Fatalf("diagnostics count = %d, want 1: %v", len(diags), diags)
	}
}

func TestFatalOrder_AllowsTerminalFatalf(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	if true {
		t.Errorf("first check")
	}
	if true {
		t.Fatalf("final check")
	}
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFatalOrder_ReportsLoopFatalf(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	for range 3 {
		t.Fatalf("loop failure")
	}
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 1 {
		t.Fatalf("diagnostics count = %d, want 1: %v", len(diags), diags)
	}
}

func TestFatalOrder_RespectsNolintDirective(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	//nolint:fatalorder
	if true {
		t.Fatalf("precondition check")
	}
	if true {
		t.Fatalf("second check")
	}
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 0 {
		t.Fatalf("nolint was not respected, got diagnostics: %v", diags)
	}
}

func TestFatalOrder_RespectsInlineNolint(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	if true {
		t.Fatalf("precondition") //nolint:fatalorder // fatal setup
	}
	if true {
		t.Fatalf("second check")
	}
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 0 {
		t.Fatalf("inline nolint was not respected, got diagnostics: %v", diags)
	}
}

func TestFatalOrder_RespectsSubtestBoundaries(t *testing.T) {
	src := `package sample
import "testing"

func TestSample(t *testing.T) {
	t.Run("sub1", func(t *testing.T) {
		t.Fatalf("sub1 fatal")
	})
	t.Run("sub2", func(t *testing.T) {
		t.Fatalf("sub2 fatal")
	})
}
`
	diags := runAnalyzer(t, src)
	if len(diags) != 0 {
		t.Fatalf("subtests should not trigger cross-scope errors: %v", diags)
	}
}
