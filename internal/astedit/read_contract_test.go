// Package astedit tests observable Go read API results without inspecting projection internals.
package astedit

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"semedit/internal/backend"
	"strings"
	"testing"
)

func TestInspectGoPreservesGroupFunctionImportsRevisionAndUTF16(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := []byte("package demo\nimport (\n fmt \"fmt\"\n _ \"net/http/pprof\"\n . \"strings\"\n \"example.org/implicit\"\n)\n/*" + string([]byte{0xF0, 0x9F, 0x98, 0x80}) + "*/ const ( First = iota; Second; Third )\nfunc Convert(input string) (string, error) {\n\tif input == \"\" {\n\t\treturn \"\", fmt.Errorf(\"empty\")\n\t}\n\treturn input, nil\n}\n")
	path := filepath.Join(root, "source.go")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	group, err := InspectGo(root, "", "Second")
	if err != nil {
		t.Fatal(err)
	}
	if len(group.Matches) != 1 {
		t.Fatalf("group matches = %d, want 1", len(group.Matches))
	}
	constant := group.Matches[0]
	wantGroup := "const ( First = iota; Second; Third )"
	if constant.Source != wantGroup {
		t.Fatalf("constant source = %q, want %q", constant.Source, wantGroup)
	}
	if constant.Range.Start.Line != 7 || constant.Range.Start.Character != 7 || constant.Range.End.Line != 7 || constant.Range.End.Character != 7+len(wantGroup) {
		t.Fatalf("constant declaration range = %#v", constant.Range)
	}
	if constant.Name != "Second" || constant.SelectionRange.Start.Line != 7 || constant.SelectionRange.Start.Character != 29 || constant.SelectionRange.End.Character != 35 {
		t.Fatalf("constant selection = %#v", constant.SelectionRange)
	}
	sum := sha256.Sum256(source)
	if constant.Revision != fmt.Sprintf("%x", sum) {
		t.Fatalf("revision = %q, want SHA-256 %x", constant.Revision, sum)
	}
	wantImports := []struct {
		path string
		name *string
	}{
		{path: "fmt", name: new("fmt")}, {path: "net/http/pprof", name: new("_")},
		{path: "strings", name: new(".")}, {path: "example.org/implicit"},
	}
	if len(constant.Imports) != len(wantImports) {
		t.Fatalf("imports = %#v", constant.Imports)
	}
	for index, want := range wantImports {
		got := constant.Imports[index]
		if got.Path != want.path || !sameOptionalString(got.Name, want.name) {
			t.Errorf("import %d = %#v, want path %q name %v", index, got, want.path, want.name)
		}
	}
	function, err := InspectGo(root, "", "Convert")
	if err != nil {
		t.Fatal(err)
	}
	if len(function.Matches) != 1 {
		t.Fatalf("function matches = %d, want 1", len(function.Matches))
	}
	got := function.Matches[0]
	wantFunction := "func Convert(input string) (string, error) {\n\tif input == \"\" {\n\t\treturn \"\", fmt.Errorf(\"empty\")\n\t}\n\treturn input, nil\n}"
	if got.Source != wantFunction {
		t.Fatalf("function source = %q, want %q", got.Source, wantFunction)
	}
	if got.Range.Start.Line != 8 || got.Range.Start.Character != 0 || got.Range.End.Line != 13 || got.Range.End.Character != 1 {
		t.Fatalf("function declaration range = %#v", got.Range)
	}
	if got.Signature == nil || strings.Contains(*got.Signature, "fmt.Errorf") || !strings.Contains(*got.Signature, "func Convert(input string)") {
		t.Fatalf("function signature = %#v", got.Signature)
	}
}

func TestInspectGoReturnsAllAmbiguitiesAndInheritedDeclaration(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	first := filepath.Join(root, "first", "shared.go")
	second := filepath.Join(root, "second", "shared.go")
	for path, source := range map[string]string{
		first:  "package first\nfunc Shared() {}\n",
		second: "package second\nfunc Shared() {}\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ambiguous, err := InspectGo(root, "", "Shared")
	if err != nil {
		t.Fatal(err)
	}
	if len(ambiguous.Matches) != 2 {
		t.Fatalf("ambiguous matches = %#v", ambiguous.Matches)
	}
	if ambiguous.Matches[0].File != filepath.ToSlash(filepath.Join("first", "shared.go")) || ambiguous.Matches[1].File != filepath.ToSlash(filepath.Join("second", "shared.go")) {
		t.Fatalf("ambiguous files are not deterministic: %q, %q", ambiguous.Matches[0].File, ambiguous.Matches[1].File)
	}

	basePath := filepath.Join(root, "base.go")
	if writeErr := os.WriteFile(basePath, []byte("package demo\ntype Base interface { Run() error }\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(root, "derived.go"), []byte("package demo\ntype Derived interface { Base }\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	inherited, err := InspectGo(root, "", "Derived.Run")
	if err != nil {
		t.Fatal(err)
	}
	if len(inherited.Matches) != 1 {
		t.Fatalf("inherited matches = %#v", inherited.Matches)
	}
	method := inherited.Matches[0]
	if method.File != "base.go" || method.Name != "Run" || method.QualifiedName != "Derived.Run" || method.Source != "Run() error" {
		t.Fatalf("inherited method = %#v", method)
	}

	testPath := filepath.Join(root, "selected_test.go")
	if writeErr := os.WriteFile(testPath, []byte("package demo\nimport \"testing\"\nfunc TestOnly(t *testing.T) {}\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	selected, err := InspectGo(root, testPath, "TestOnly")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Matches) != 1 || selected.Matches[0].File != "selected_test.go" || selected.Matches[0].Name != "TestOnly" {
		t.Fatalf("explicit test-file inspection = %#v", selected)
	}
}

func TestOutlineGoFiltersStructurallyAndPreservesEmptyAndTestFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	emptySource := []byte("// Package empty preserves file documentation.\npackage empty\n")
	if err := os.WriteFile(filepath.Join(root, "empty.go"), emptySource, 0o600); err != nil {
		t.Fatal(err)
	}
	mainSource := "package demo\ntype Box[T any] struct {\n *pkg.Box[T]\n Value int\n hidden string\n}\ntype Mixed interface {\n io.Reader\n Run(context.Context, ...string) error\n}\nfunc Visible() {}\nfunc hidden() {}\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(mainSource), 0o600); err != nil {
		t.Fatal(err)
	}

	outline, err := OutlineGo(root, root, []string{"field", "function"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	assertOutlinePreservesEmptyFile(t, outline)
	assertOutlineContainsStructuralBox(t, outline)

	methodOutline, err := OutlineGo(root, root, []string{"method"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	assertOutlineFiltersInterfaceMethods(t, methodOutline)

	testPath := filepath.Join(root, "selected_test.go")
	if writeErr := os.WriteFile(testPath, []byte("package demo\nfunc TestOnly() {}\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	selected, err := OutlineGo(root, testPath, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	assertOutlineIncludesExplicitTestFile(t, selected)
}

func assertOutlinePreservesEmptyFile(t *testing.T, outline *backend.OutlineResult) {
	t.Helper()
	for _, file := range outline.Files {
		if file.File == "empty.go" && file.Doc != nil && *file.Doc == "// Package empty preserves file documentation." && len(file.Symbols) == 0 {
			return
		}
	}
	t.Fatalf("empty file metadata is missing from outline: %#v", outline.Files)
}

func assertOutlineContainsStructuralBox(t *testing.T, outline *backend.OutlineResult) {
	t.Helper()
	main := outlineFileByName(outline, "main.go")
	if main == nil {
		t.Fatal("main.go is missing from outline")
	}
	var box *backend.ReadSymbol
	for index := range main.Symbols {
		symbol := &main.Symbols[index]
		if symbol.Name == "Box" {
			box = symbol
		}
		if symbol.Name == "hidden" {
			t.Fatalf("unexported free function survived visibility filter: %#v", symbol)
		}
	}
	if box == nil || len(box.Children) != 2 {
		t.Fatalf("field filter omitted expected Box structure: %#v", main.Symbols)
	}
	assertEmbeddedBoxField(t, box.Children[0])
}

func assertEmbeddedBoxField(t *testing.T, embedded backend.ReadSymbol) {
	t.Helper()
	if embedded.Name != "Box" || embedded.QualifiedName != "Box.Box" || embedded.Type == nil || *embedded.Type != "*pkg.Box[T]" {
		t.Fatalf("embedded field projection = %#v", embedded)
	}
	if embedded.Range.Start.Line != 2 || embedded.Range.Start.Character != 1 || embedded.Range.End.Character != 12 {
		t.Fatalf("embedded field full type range = %#v", embedded.Range)
	}
	if embedded.SelectionRange.Start.Line != 2 || embedded.SelectionRange.Start.Character != 6 || embedded.SelectionRange.End.Character != 9 {
		t.Fatalf("embedded field base identifier selection = %#v", embedded.SelectionRange)
	}
}

func assertOutlineFiltersInterfaceMethods(t *testing.T, outline *backend.OutlineResult) {
	t.Helper()
	file := outlineFileByName(outline, "main.go")
	if file == nil {
		t.Fatal("main.go is missing from method outline")
	}
	var mixed *backend.ReadSymbol
	for index := range file.Symbols {
		if file.Symbols[index].Name == "Mixed" {
			mixed = &file.Symbols[index]
			break
		}
	}
	if mixed == nil || len(mixed.Children) != 1 || mixed.Children[0].Name != "Run" {
		t.Fatalf("method filter did not retain only its structural interface context: %#v", mixed)
	}
	if mixed.Signature == nil || strings.Contains(*mixed.Signature, "io.Reader") {
		t.Fatalf("filtered interface signature retained unmatched embedding: %#v", mixed.Signature)
	}
}

func assertOutlineIncludesExplicitTestFile(t *testing.T, outline *backend.OutlineResult) {
	t.Helper()
	if len(outline.Files) != 1 || outline.Files[0].File != "selected_test.go" || len(outline.Files[0].Symbols) != 1 {
		t.Fatalf("explicit test-file outline = %#v", outline)
	}
}

func outlineFileByName(outline *backend.OutlineResult, name string) *backend.OutlineFile {
	for index := range outline.Files {
		if outline.Files[index].File == name {
			return &outline.Files[index]
		}
	}
	return nil
}

func TestOutlineGoRetainsExportedFreeFunctionWhenFilteringUnexported(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "functions.go")
	if err := os.WriteFile(path, []byte("package demo\nfunc Visible() {}\nfunc hidden() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := OutlineGo(root, path, []string{"function"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || len(result.Files[0].Symbols) != 1 || result.Files[0].Symbols[0].Name != "Visible" {
		t.Fatalf("visible function outline = %#v", result)
	}
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
