// Package readlsp tests the shared source-snapshot projection contract used by LSP backends.
package readlsp

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"semedit/internal/backend"
	"testing"
)

func TestOutlineAndInspectUseExactUTF16Snapshot(t *testing.T) {
	root := "/workspace"
	file := filepath.Join(root, "main.sh")
	source := []byte("// 😀fn\r\n")
	symbol := makeReadSymbol(file, "fn", "function", backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 5}, backend.Position{Line: 0, Character: 7})
	document := makeDocument(root, file, source, []backend.ReadSymbol{symbol}, []backend.ReadImport{})
	doc := "file docs"
	document.Doc = &doc
	document.Complete = false
	document.Limitations = []string{"bounded server range"}
	outline, err := Outline(document, backend.OutlineRequest{Path: "main.sh", IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if outline.Scope.Kind != "selected_file" || outline.Scope.Path != "main.sh" || outline.Scope.Complete || len(outline.Scope.Limitations) != 1 {
		t.Fatalf("scope = %#v", outline.Scope)
	}
	if len(outline.Files) != 1 || outline.Files[0].Revision != fmt.Sprintf("%x", sha256.Sum256(source)) || outline.Files[0].Doc == nil || *outline.Files[0].Doc != doc || outline.Files[0].Imports == nil || len(outline.Files[0].Symbols) != 1 {
		t.Fatalf("outline file = %#v", outline.Files)
	}
	inspected, err := Inspect(document, []InspectMatch{{Symbol: symbol, SourceExtent: "server_range"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Matches) != 1 || inspected.Matches[0].Source != "// 😀fn" || inspected.Matches[0].SourceExtent != "server_range" || inspected.Matches[0].Revision != outline.Files[0].Revision {
		t.Fatalf("inspect = %#v", inspected)
	}
}

func TestProjectionRejectsInvalidUTF16AndRangesBeforeFiltering(t *testing.T) {
	root := "/workspace"
	file := filepath.Join(root, "main.sh")
	source := []byte("// 😀fn\r\n")
	cases := []struct {
		name                                               string
		rangeStart, rangeEnd, selectionStart, selectionEnd backend.Position
	}{{"surrogate split", backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 4}, backend.Position{Line: 0, Character: 7}}, {"past line end", backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 5}, backend.Position{Line: 0, Character: 8}}, {"reversed range", backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 5}, backend.Position{Line: 0, Character: 7}}, {"selection outside range", backend.Position{Line: 0, Character: 5}, backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 3}, backend.Position{Line: 0, Character: 5}}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			symbol := makeReadSymbol(file, "fn", "function", tc.rangeStart, tc.rangeEnd, tc.selectionStart, tc.selectionEnd)
			document := makeDocument(root, file, source, []backend.ReadSymbol{symbol}, nil)
			if _, err := Outline(document, backend.OutlineRequest{Path: "main.sh", Kinds: []string{"type"}, IncludeUnexported: true}); err == nil {
				t.Fatal("Outline accepted an invalid hidden symbol")
			}
		})
	}
}

func TestOutlineRetainsStructuralAncestorsAndDoesNotMutateTree(t *testing.T) {
	root := "/workspace"
	file := filepath.Join(root, "main.sh")
	source := []byte("Box F g\n")
	field := makeReadSymbol(file, "F", "field", backend.Position{Line: 0, Character: 4}, backend.Position{Line: 0, Character: 5}, backend.Position{Line: 0, Character: 4}, backend.Position{Line: 0, Character: 5})
	variable := makeReadSymbol(file, "g", "variable", backend.Position{Line: 0, Character: 6}, backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 6}, backend.Position{Line: 0, Character: 7})
	parent := makeReadSymbol(file, "Box", "type", backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 7}, backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 3})
	parent.Children = []backend.ReadSymbol{variable, field}
	document := makeDocument(root, file, source, []backend.ReadSymbol{parent}, nil)
	all, err := Outline(document, backend.OutlineRequest{Path: "main.sh", IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	children := all.Files[0].Symbols[0].Children
	if len(children) != 2 || children[0].Name != "F" || children[1].Name != "g" {
		t.Fatalf("sorted children = %#v", children)
	}
	if parent.Children[0].Name != "g" || parent.Children[1].Name != "F" {
		t.Fatalf("caller tree mutated: %#v", parent.Children)
	}
	fields, err := Outline(document, backend.OutlineRequest{Path: "main.sh", Kinds: []string{"field"}, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	projected := fields.Files[0].Symbols
	if len(projected) != 1 || projected[0].Kind != "type" || len(projected[0].Children) != 1 || projected[0].Children[0].Name != "F" {
		t.Fatalf("ancestor projection = %#v", projected)
	}
	types, err := Outline(document, backend.OutlineRequest{Path: "main.sh", Kinds: []string{"type"}, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(types.Files[0].Symbols) != 1 || len(types.Files[0].Symbols[0].Children) != 0 {
		t.Fatalf("matching parent retained filtered children: %#v", types.Files[0].Symbols)
	}
}

func TestInspectPreservesAllMatchesAndValidatesUnmatchedMappedSymbols(t *testing.T) {
	root := "/workspace"
	file := filepath.Join(root, "main.mk")
	source := []byte("foo()\nfoo()\n")
	first := makeReadSymbol(file, "foo", "target", backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 5}, backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 3})
	second := makeReadSymbol(file, "foo", "target", backend.Position{Line: 1, Character: 0}, backend.Position{Line: 1, Character: 5}, backend.Position{Line: 1, Character: 0}, backend.Position{Line: 1, Character: 3})
	document := makeDocument(root, file, source, []backend.ReadSymbol{second, first}, nil)
	result, err := Inspect(document, []InspectMatch{{Symbol: second, SourceExtent: "server_range"}, {Symbol: first, SourceExtent: "server_range"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 2 || result.Matches[0].Range.Start.Line != 0 || result.Matches[1].Range.Start.Line != 1 || result.Matches[0].Source != "foo()" || result.Matches[1].Source != "foo()" {
		t.Fatalf("matches = %#v", result.Matches)
	}
	if result.Matches[0].Imports != nil {
		t.Fatal("unavailable imports should remain nil")
	}
	bad := makeReadSymbol(file, "bad", "variable", backend.Position{Line: 1, Character: 0}, backend.Position{Line: 1, Character: 5}, backend.Position{Line: 1, Character: 4}, backend.Position{Line: 1, Character: 6})
	document.Symbols = append(document.Symbols, bad)
	if _, err := Inspect(document, []InspectMatch{{Symbol: first, SourceExtent: "declaration"}}); err == nil {
		t.Fatal("Inspect ignored malformed unmatched mapped symbol")
	}
	document.Symbols = []backend.ReadSymbol{first, second}
	document.Imports = []backend.ReadImport{}
	knownEmpty, err := Inspect(document, []InspectMatch{{Symbol: first, SourceExtent: "server_range"}})
	if err != nil {
		t.Fatal(err)
	}
	if knownEmpty.Matches[0].Imports == nil || len(knownEmpty.Matches[0].Imports) != 0 {
		t.Fatalf("known empty imports = %#v", knownEmpty.Matches[0].Imports)
	}
}

func TestEmptyFilteredOutlineRetainsFileMetadataAndCRLines(t *testing.T) {
	root := "/workspace"
	file := filepath.Join(root, "main.sh")
	source := []byte("header\rfn\n")
	symbol := makeReadSymbol(file, "fn", "function", backend.Position{Line: 1, Character: 0}, backend.Position{Line: 1, Character: 2}, backend.Position{Line: 1, Character: 0}, backend.Position{Line: 1, Character: 2})
	document := makeDocument(root, file, source, []backend.ReadSymbol{symbol}, []backend.ReadImport{})
	doc := "retained"
	document.Doc = &doc
	result, err := Outline(document, backend.OutlineRequest{Path: "main.sh", Kinds: []string{"type"}, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	fileResult := result.Files[0]
	if fileResult.Symbols == nil || len(fileResult.Symbols) != 0 || fileResult.Doc == nil || *fileResult.Doc != doc || fileResult.Imports == nil || fileResult.Package != "p" {
		t.Fatalf("filtered file metadata lost: %#v", fileResult)
	}
}

func TestOutlineRequestRejectsUnsupportedScopesAndKinds(t *testing.T) {
	request := backend.OutlineRequest{Path: "file.mk", IncludeUnexported: true}
	if err := ValidateOutlineRequest(request, true); err == nil {
		t.Fatal("directory request was accepted")
	}
	request.IncludeUnexported = false
	if err := ValidateOutlineRequest(request, false); err == nil {
		t.Fatal("visibility filter was accepted")
	}
	request.IncludeUnexported = true
	request.Kinds = []string{"unknown"}
	if err := ValidateOutlineRequest(request, false); err == nil {
		t.Fatal("unknown kind was accepted")
	}
}

func TestProjectionPreservesExternalPathsAndRejectsForeignMappedNodes(t *testing.T) {
	root := "/workspace"
	file := "/outside/nested/main.mk"
	source := []byte("foo\n")
	selected := makeReadSymbol(file, "foo", "target", backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 3}, backend.Position{Line: 0, Character: 0}, backend.Position{Line: 0, Character: 3})
	document := makeDocument(root, file, source, []backend.ReadSymbol{selected}, nil)
	outline, err := Outline(document, backend.OutlineRequest{Path: file, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if outline.Scope.Path != file || outline.Files[0].File != file || outline.Files[0].Symbols[0].File != file {
		t.Fatalf("external paths were rewritten: %#v", outline)
	}
	foreign := selected
	foreign.File = filepath.Join(root, "other.mk")
	document.Symbols = []backend.ReadSymbol{selected, foreign}
	if _, err := Outline(document, backend.OutlineRequest{Path: file, Kinds: []string{"function"}, IncludeUnexported: true}); err == nil {
		t.Fatal("filtered foreign mapped symbol was accepted")
	}
	parent := selected
	parent.Kind = "type"
	child := foreign
	child.Kind = "variable"
	parent.Children = []backend.ReadSymbol{child}
	document.Symbols = []backend.ReadSymbol{parent}
	if _, err := Outline(document, backend.OutlineRequest{Path: file, Kinds: []string{"type"}, IncludeUnexported: true}); err == nil {
		t.Fatal("foreign nonmatching child was accepted")
	}
}

func TestByteOffsetSupportsLineTerminatorsAndRejectsSurrogateSplits(t *testing.T) {
	source := []byte("😀x\rnext\r\nend\n")
	cases := []struct {
		position backend.Position
		want     int
	}{{backend.Position{Line: 0, Character: 2}, 4}, {backend.Position{Line: 0, Character: 3}, 5}, {backend.Position{Line: 1, Character: 4}, 10}, {backend.Position{Line: 2, Character: 3}, 15}}
	for _, tc := range cases {
		got, err := ByteOffset(source, tc.position)
		if err != nil {
			t.Fatalf("ByteOffset(%#v): %v", tc.position, err)
		}
		if got != tc.want {
			t.Errorf("ByteOffset(%#v) = %d, want %d", tc.position, got, tc.want)
		}
	}
	if _, err := ByteOffset(source, backend.Position{Line: 0, Character: 1}); err == nil {
		t.Fatal("ByteOffset accepted a split surrogate position")
	}
	if _, err := ByteOffset(source, backend.Position{Line: 0, Character: 4}); err == nil {
		t.Fatal("ByteOffset accepted a position beyond line end")
	}
}

func makeReadSymbol(file, name, kind string, start, end, selectionStart, selectionEnd backend.Position) backend.ReadSymbol {
	return backend.ReadSymbol{Name: name, QualifiedName: name, Kind: kind, File: file, Range: backend.Range{Start: start, End: end}, SelectionRange: backend.Range{Start: selectionStart, End: selectionEnd}}
}

func makeDocument(root, file string, source []byte, symbols []backend.ReadSymbol, imports []backend.ReadImport) Document {
	return Document{Root: root, File: file, Language: backend.LanguageBash, Source: source, Symbols: symbols, Package: "p", Imports: imports, Complete: true}
}
