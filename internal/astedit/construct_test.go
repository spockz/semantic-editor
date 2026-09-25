// Package astedit_test verifies construct replacement behavior through visible source changes.
package astedit_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
	"semedit/internal/pipeline"
)

func writeGo(t *testing.T, path, content string) {
	t.Helper()
	if err := pipeline.WriteAtomic(path, []byte(content)); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readGo(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestReplaceLoop_TargetsNestedRangeAndPreservesOuterBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	initial := "package sample\nfunc Check(items []string) {\n before()\n for _, item := range items {\n  for _, want := range []string{\"a\"} {\n   check(item, want)\n  }\n }\n after()\n}\n"
	writeGo(t, path, initial)
	if _, err := astedit.ReplaceLoop(context.Background(), path, "Check", "want", "for _, want := range []string{\"b\", \"c\"} { check(want) }", astedit.LoopOptions{}); err != nil {
		t.Fatalf("ReplaceLoop: %v", err)
	}
	got := readGo(t, path)
	for _, preserved := range []string{"before()", "after()", "for _, item := range items", "[]string{\"b\", \"c\"}"} {
		if !strings.Contains(got, preserved) {
			t.Errorf("updated file missing %q:\n%s", preserved, got)
		}
	}
}

func TestReplaceLoop_AmbiguityListsPathsAndLocations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	writeGo(t, path, "package sample\nfunc Check(items []string) { for _, want := range items { use(want) }; for _, want := range items { use(want) } }\n")
	_, err := astedit.ReplaceLoop(context.Background(), path, "Check", "want", "for _, want := range items { use(1) }", astedit.LoopOptions{})
	if err == nil || !strings.Contains(err.Error(), "path 0") || !strings.Contains(err.Error(), "path 1") || !strings.Contains(err.Error(), ":2:") {
		t.Fatalf("expected path and source locations in ambiguity error, got %v", err)
	}
	if _, err := astedit.ReplaceLoop(context.Background(), path, "Check", "want", "for _, want := range items { use(1) }", astedit.LoopOptions{LoopPath: "1"}); err != nil {
		t.Fatalf("ReplaceLoop with reported path: %v", err)
	}
	got := readGo(t, path)
	if strings.Count(got, "use(want)") != 1 || strings.Count(got, "use(1)") != 1 {
		t.Fatalf("loop_path did not select only the second loop:\n%s", got)
	}
}

func TestReplaceLoop_RejectsBareStatementsWithoutMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	initial := "package sample\nfunc Check(items []string) { for _, want := range items { use(want) } }\n"
	writeGo(t, path, initial)
	_, err := astedit.ReplaceLoop(context.Background(), path, "Check", "want", "use(1)", astedit.LoopOptions{})
	if err == nil || !strings.Contains(err.Error(), "complete for or range loop") {
		t.Fatalf("expected complete loop validation, got %v", err)
	}
	if got := readGo(t, path); got != initial {
		t.Fatalf("file mutated after rejected source:\n%s", got)
	}
}

func TestReplaceLoop_DiscriminatorMatchesIdentifierExactly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	initial := "package sample\nfunc Check(want, unwanted []string) { for _, unwanted := range unwanted { use(unwanted) }; for _, want := range want { use(want) } }\n"
	writeGo(t, path, initial)
	if _, err := astedit.ReplaceLoop(context.Background(), path, "Check", "want", "for _, want := range want { use(\"changed\") }", astedit.LoopOptions{}); err != nil {
		t.Fatalf("ReplaceLoop: %v", err)
	}
	got := readGo(t, path)
	if !strings.Contains(got, "use(unwanted)") || !strings.Contains(got, "use(\"changed\")") {
		t.Fatalf("identifier discriminator matched the wrong loop:\n%s", got)
	}
}

func TestReplaceLoop_ReplacesClassicForHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	writeGo(t, path, "package sample\nfunc Check() { before(); for i := 0; i < 3; i++ { tick(i) }; after() }\n")
	if _, err := astedit.ReplaceLoop(context.Background(), path, "Check", "i := 0", "for i := 10; i < 20; i++ { tick(i * 2) }", astedit.LoopOptions{}); err != nil {
		t.Fatalf("ReplaceLoop: %v", err)
	}
	got := readGo(t, path)
	for _, want := range []string{"before()", "for i := 10; i < 20; i++", "tick(i * 2)", "after()"} {
		if !strings.Contains(got, want) {
			t.Errorf("updated file missing %q:\n%s", want, got)
		}
	}
}

func TestReplaceLoop_PathSelectsNestedMatchingLoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	writeGo(t, path, "package sample\nfunc Check(wants []string) { for _, want := range wants { for _, want := range wants { use(want) } } }\n")
	if _, err := astedit.ReplaceLoop(context.Background(), path, "Check", "want", "for _, want := range wants { use(\"inner\") }", astedit.LoopOptions{LoopPath: "0.0.0"}); err != nil {
		t.Fatalf("ReplaceLoop with nested loop_path: %v", err)
	}
	got := readGo(t, path)
	if strings.Count(got, "range wants") != 2 || !strings.Contains(got, "use(\"inner\")") || strings.Contains(got, "use(want)") {
		t.Fatalf("loop_path did not replace only the inner matching loop:\n%s", got)
	}
}

func TestReplaceLoop_RejectsMalformedReplacementWithoutMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loops.go")
	initial := "package sample\nfunc Check() { for i := 0; i < 3; i++ { tick(i) } }\n"
	writeGo(t, path, initial)
	_, err := astedit.ReplaceLoop(context.Background(), path, "Check", "i := 0", "for i := ; i < 10; i++ { tick(i) }", astedit.LoopOptions{})
	if err == nil {
		t.Fatal("expected malformed loop source to fail")
	}
	if got := readGo(t, path); got != initial {
		t.Fatalf("file mutated after malformed source:\n%s", got)
	}
}

func TestReplaceDecl_GroupedVarPreservesSiblingSpecs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.go")
	writeGo(t, path, "package sample\nvar (\n // keep this comment\n target = 1\n sibling = 2\n)\n")
	if _, err := astedit.ReplaceDecl(context.Background(), path, "target", "var target = 3", astedit.ReplaceDeclOptions{}); err != nil {
		t.Fatalf("ReplaceDecl: %v", err)
	}
	got := readGo(t, path)
	if !strings.Contains(got, "sibling = 2") || !strings.Contains(got, "keep this comment") {
		t.Fatalf("updated file lost sibling spec or comment:\n%s", got)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, got, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse updated declarations: %v", err)
	}
	foundTarget := false
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				if name.Name == "target" && len(value.Values) == 1 {
					literal, ok := value.Values[0].(*ast.BasicLit)
					foundTarget = ok && literal.Value == "3"
				}
			}
		}
	}
	if !foundTarget {
		t.Fatalf("updated target declaration does not have value 3:\n%s", got)
	}
}

func TestReplaceDecl_RejectsImplicitConstSiblingMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.go")
	writeGo(t, path, "package sample\nconst (\n first = 1\n second\n)\n")
	_, err := astedit.ReplaceDecl(context.Background(), path, "first", "const first = 4", astedit.ReplaceDeclOptions{})
	if err == nil || !strings.Contains(err.Error(), "inherits its expression") {
		t.Fatalf("expected inherited const rejection, got %v", err)
	}
	if got := readGo(t, path); !strings.Contains(got, "first = 1") {
		t.Fatalf("file mutated after rejected replacement:\n%s", got)
	}
}

func TestReplaceDecl_RejectsSharedVarSpecWithoutMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.go")
	initial := "package sample\nvar first, second = 1, 2\n"
	writeGo(t, path, initial)
	_, err := astedit.ReplaceDecl(context.Background(), path, "first", "var first = 4", astedit.ReplaceDeclOptions{})
	if err == nil || !strings.Contains(err.Error(), "shared value spec") {
		t.Fatalf("expected shared variable spec rejection, got %v", err)
	}
	if got := readGo(t, path); got != initial {
		t.Fatalf("file mutated after rejected replacement:\n%s", got)
	}
}

func TestReplaceDecl_ReplacesTypeAlias(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.go")
	writeGo(t, path, "package sample\ntype Payload = string\ntype Other = int\n")
	if _, err := astedit.ReplaceDecl(context.Background(), path, "Payload", "type Payload = []byte", astedit.ReplaceDeclOptions{}); err != nil {
		t.Fatalf("ReplaceDecl type alias: %v", err)
	}
	got := readGo(t, path)
	if !strings.Contains(got, "type Payload = []byte") || !strings.Contains(got, "type Other = int") {
		t.Fatalf("alias update changed the wrong declaration:\n%s", got)
	}
}

func TestInsertDeclRejectsPackageCollisionAndOverwritesOnlyTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "one.go")
	writeGo(t, path, "package sample\nconst Existing = 1\n")
	err := astedit.InsertDecl(context.Background(), path, "const Existing = 2", astedit.DeclOptions{})
	if !errors.Is(err, astedit.ErrDeclCollision) {
		t.Fatalf("expected collision sentinel, got %v", err)
	}
	if err := astedit.InsertDecl(context.Background(), path, "const Existing = 2", astedit.DeclOptions{Overwrite: true}); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got := readGo(t, path)
	if !strings.Contains(got, "Existing = 2") || strings.Contains(got, "Existing = 1") {
		t.Fatalf("overwrite did not replace target:\n%s", got)
	}
}

func TestInsertDeclFindsSiblingPackageCollision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "one.go")
	writeGo(t, path, "package sample\nfunc Use() {}\n")
	writeGo(t, filepath.Join(dir, "two.go"), "package sample\nconst Existing = 1\n")
	err := astedit.InsertDecl(context.Background(), path, "const Existing = 2", astedit.DeclOptions{})
	if !errors.Is(err, astedit.ErrDeclCollision) || !strings.Contains(err.Error(), "two.go") {
		t.Fatalf("expected sibling collision, got %v", err)
	}
}
