// This file protects comment attachment across Go insertion and construct replacement operations.
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

	"semedit/internal/pipeline"
)

func TestNormalizeInsertionOffsetCommentBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		target      string
		want        string
		afterTarget bool
	}{
		{name: "leading standalone line comment", source: "package p\n// docs\nvar Value int\n", target: "var Value", want: "// docs"},
		{name: "leading same-line block comment", source: "package p\n/* docs */ var Value int\n", target: "var Value", want: "/* docs */"},
		{name: "blank-separated comment", source: "package p\n// docs\n\nvar Value int\n", target: "var Value", want: "var Value"},
		{name: "inline trailing line comment", source: "package p\nvar Value int // docs\nvar Next int\n", target: "var Value int", want: "var Next", afterTarget: true},
		{name: "inline trailing block comment", source: "package p\nvar Value int /* docs */\nvar Next int\n", target: "var Value int", want: "var Next", afterTarget: true},
		{name: "standalone comment after declaration", source: "package p\nvar Value int\n// next docs\nvar Next int\n", target: "var Next", want: "// next docs"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "anchor.go", test.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse source: %v", err)
			}
			targetOffset := strings.Index(test.source, test.target)
			if targetOffset < 0 {
				t.Fatalf("target %q not found", test.target)
			}
			if test.afterTarget {
				targetOffset += len(test.target)
			}
			got := normalizeInsertionOffset(fset, file, []byte(test.source), targetOffset)
			want := strings.Index(test.source, test.want)
			if got != want {
				t.Fatalf("normalized offset = %d (%q), want %d (%q)", got, test.source[got:], want, test.source[want:])
			}
		})
	}
}

func TestInsertDeclarationKeepsFunctionDocAttached(t *testing.T) {
	initial := "package p\n\n// Keep documentation.\nfunc Keep() {}\n"
	path := writeCommentFixture(t, initial)
	if err := InsertDeclaration(context.Background(), path, "func Added() {}", Options{Placement: PlacementBeforeSymbol, TargetSymbol: "Keep"}); err != nil {
		t.Fatalf("InsertDeclaration: %v", err)
	}
	got := readCommentFixture(t, path)
	assertCommentSource(t, got, "package p\n\nfunc Added() {}\n\n// Keep documentation.\nfunc Keep() {}\n")
	assertFunctionDoc(t, got, "Keep", "Keep documentation.\n")
}

func TestInsertFunctionKeepsFunctionDocAttached(t *testing.T) {
	initial := "package p\n\n// Keep documentation.\nfunc Keep() {}\n"
	path := writeCommentFixture(t, initial)
	if err := InsertFunction(context.Background(), path, "func Added() {}", FunctionOptions{Placement: PlacementBeforeSymbol, TargetSymbol: "Keep"}); err != nil {
		t.Fatalf("InsertFunction: %v", err)
	}
	got := readCommentFixture(t, path)
	assertCommentSource(t, got, "package p\n\nfunc Added() {}\n\n// Keep documentation.\nfunc Keep() {}\n")
	assertFunctionDoc(t, got, "Keep", "Keep documentation.\n")
}

func TestInsertTypeKeepsTypeDocAttached(t *testing.T) {
	initial := "package p\n\n// Keep documentation.\ntype Keep struct{}\n"
	path := writeCommentFixture(t, initial)
	if err := InsertType(context.Background(), path, "type Added struct{}", TypeOptions{}); err != nil {
		t.Fatalf("InsertType: %v", err)
	}
	got := readCommentFixture(t, path)
	assertCommentSource(t, got, "package p\n\ntype Added struct{}\n\n// Keep documentation.\ntype Keep struct{}\n")
	assertGenDeclDoc(t, got, "Keep", "Keep documentation.\n")
}

func TestInsertDeclStandaloneKeepsDeclarationDocAttached(t *testing.T) {
	initial := "package p\n\n// Keep documentation.\nvar Keep = 1\n"
	path := writeCommentFixture(t, initial)
	if err := InsertDecl(context.Background(), path, "var Added = 2", DeclOptions{Group: "standalone", Placement: PlacementBeforeSymbol, TargetSymbol: "Keep"}); err != nil {
		t.Fatalf("InsertDecl: %v", err)
	}
	got := readCommentFixture(t, path)
	assertCommentSource(t, got, "package p\n\nvar Added = 2\n\n// Keep documentation.\nvar Keep = 1\n")
	assertGenDeclDoc(t, got, "Keep", "Keep documentation.\n")
}

func TestInsertDeclGroupedKeepsSpecDocAttached(t *testing.T) {
	initial := "package p\n\nimport \"errors\"\n\nvar (\n\t// Keep documentation.\n\tErrKeep = errors.New(\"keep\")\n)\n"
	path := writeCommentFixture(t, initial)
	if err := InsertDecl(context.Background(), path, "var ErrAdded = errors.New(\"added\")", DeclOptions{}); err != nil {
		t.Fatalf("InsertDecl grouped: %v", err)
	}
	got := readCommentFixture(t, path)
	want := "package p\n\nimport \"errors\"\n\nvar (\n\tErrAdded = errors.New(\"added\")\n\t// Keep documentation.\n\tErrKeep = errors.New(\"keep\")\n)\n"
	assertCommentSource(t, got, want)
	assertValueSpecDoc(t, got, "ErrKeep", "Keep documentation.\n")
}

func TestInsertCaseKeepsDefaultCommentBeforeDefault(t *testing.T) {
	initial := "package p\n\nfunc Handle(value string) {\n\tswitch value {\n\tcase \"keep\":\n\t\treturn\n\t// Keep documentation.\n\tdefault:\n\t\treturn\n\t}\n}\n"
	path := writeCommentFixture(t, initial)
	if _, err := InsertCase(context.Background(), path, "Handle", "value", "case \"added\":\n\treturn", CaseOptions{Placement: CasePlacementBeforeDefault}); err != nil {
		t.Fatalf("InsertCase: %v", err)
	}
	got := readCommentFixture(t, path)
	want := "package p\n\nfunc Handle(value string) {\n\tswitch value {\n\tcase \"keep\":\n\t\treturn\n\tcase \"added\":\n\t\treturn\n\t// Keep documentation.\n\tdefault:\n\t\treturn\n\t}\n}\n"
	assertCommentSource(t, got, want)
	if strings.Index(got, "// Keep documentation.") > strings.Index(got, "default:") {
		t.Fatalf("default comment moved below its case:\n%s", got)
	}
}

func TestInsertDeclarationAfterSymbolFollowsInlineTrailingComment(t *testing.T) {
	initial := "package p\n\nfunc Keep() {} // Keep this implementation.\n\nfunc Last() {}\n"
	path := writeCommentFixture(t, initial)
	if err := InsertDeclaration(context.Background(), path, "func Added() {}", Options{Placement: PlacementAfterSymbol, TargetSymbol: "Keep"}); err != nil {
		t.Fatalf("InsertDeclaration after symbol: %v", err)
	}
	got := readCommentFixture(t, path)
	want := "package p\n\nfunc Keep() {} // Keep this implementation.\n\nfunc Added() {}\n\nfunc Last() {}\n"
	assertCommentSource(t, got, want)
}

func TestReplaceLoopKeepsLeadingCommentWithLoop(t *testing.T) {
	initial := "package p\n\nfunc Count(values []int) {\n\t// Keep documentation.\n\tfor _, value := range values {\n\t\t_ = value\n\t}\n}\n"
	path := writeCommentFixture(t, initial)
	if _, err := ReplaceLoop(context.Background(), path, "Count", "values", "for range values {}", LoopOptions{}); err != nil {
		t.Fatalf("ReplaceLoop: %v", err)
	}
	got := readCommentFixture(t, path)
	want := "package p\n\nfunc Count(values []int) {\n\t// Keep documentation.\n\tfor range values {\n\t}\n}\n"
	assertCommentSource(t, got, want)
}

func writeCommentFixture(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target.go")
	if err := pipeline.WriteAtomic(path, []byte(source)); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return path
}

func readCommentFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	return string(data)
}

func assertCommentSource(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("source mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

func assertFunctionDoc(t *testing.T, source, name, want string) {
	t.Helper()
	file := parseCommentFixture(t, source)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name {
			if fn.Doc == nil || fn.Doc.Text() != want {
				t.Fatalf("%s doc = %v, want %q", name, fn.Doc, want)
			}
			return
		}
	}
	t.Fatalf("function %s not found", name)
}

func assertGenDeclDoc(t *testing.T, source, name, want string) {
	t.Helper()
	file := parseCommentFixture(t, source)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Doc == nil || gen.Doc.Text() != want {
			continue
		}
		for _, spec := range gen.Specs {
			switch value := spec.(type) {
			case *ast.TypeSpec:
				if value.Name.Name == name {
					return
				}
			case *ast.ValueSpec:
				for _, ident := range value.Names {
					if ident.Name == name {
						return
					}
				}
			}
		}
	}
	t.Fatalf("declaration %s with doc %q not found", name, want)
}

func assertValueSpecDoc(t *testing.T, source, name, want string) {
	t.Helper()
	file := parseCommentFixture(t, source)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || value.Doc == nil || value.Doc.Text() != want {
				continue
			}
			for _, ident := range value.Names {
				if ident.Name == name {
					return
				}
			}
		}
	}
	t.Fatalf("value spec %s with doc %q not found", name, want)
}

func parseCommentFixture(t *testing.T, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "result.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	return file
}
