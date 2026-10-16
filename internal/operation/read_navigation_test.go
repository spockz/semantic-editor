// Package operation verifies human-readable navigation output keeps complete, valid selector identities.
package operation

import (
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
)

func TestRenderLookupNavigationUsesInclusiveDeclarationLinesAndWorkingView(t *testing.T) {
	signature := "func (Server) Serve() error"
	output := renderLookupNavigation(&LookupRes{
		Symbol: "Server.Serve", SourceView: "working",
		Definition: &LookupTarget{Name: "Serve", QualifiedName: "Server.Serve", Kind: "method", File: "server.go", Line: 5, Column: 2,
			DeclarationRange: &backend.Range{Start: backend.Position{Line: 4}, End: backend.Position{Line: 7}}, Signature: &signature},
	})
	for _, want := range []string{"Resolved lookup in working source", "server.go:5-7  func (Server) Serve() error", "symbol: Server.Serve"} {
		if !strings.Contains(output, want) {
			t.Errorf("lookup output missing %q: %s", want, output)
		}
	}
}

func TestRenderLookupNavigationAllowsNativeStartOnlyTargets(t *testing.T) {
	output := renderLookupNavigation(&LookupRes{Symbol: "Run", Definition: &LookupTarget{Name: "Run", Kind: "function", File: "main.rs", Line: 9, Column: 3}})
	if !strings.Contains(output, "main.rs:9:3  function Run") || strings.Contains(output, "symbol: Run") {
		t.Fatalf("start-only native lookup output = %s", output)
	}
}

func TestRenderLookupNavigationRetainsLocalBindingsWithoutSelectors(t *testing.T) {
	output := renderLookupNavigation(&LookupRes{Symbol: "Run", Definition: &LookupTarget{Name: "Run", QualifiedName: "pkg.Run", Kind: "function", File: "global.go", Line: 1, Column: 6, DeclarationRange: &backend.Range{Start: backend.Position{}, End: backend.Position{Line: 1}}}, Usages: []LookupTarget{
		{Name: "Run", QualifiedName: "pkg.Run", Kind: "function", File: "global.go", Line: 1, Column: 6},
		{Name: "Run", QualifiedName: "pkg.Run", Kind: "parameter", File: "local.go", Line: 8, Column: 9},
	}})
	for _, want := range []string{"Local bindings (navigation only):", "local.go:8:9  parameter Run"} {
		if !strings.Contains(output, want) {
			t.Errorf("lookup output missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "global.go:1:6  function Run") || strings.Contains(output, "symbol: pkg.Run\n  local.go") {
		t.Fatalf("lookup output duplicated the promoted definition or gave a local binding a selector: %s", output)
	}
}

func TestRenderReferencesNavigationGroupsOnlyKnownEnclosingSymbolsPerFile(t *testing.T) {
	signature := "func Serve()"
	result := &backend.ReferencesResult{Scope: backend.ReadScope{Kind: "workspace", Path: ".", Complete: true}, SourceView: "working", Symbol: backend.ReadSymbol{Name: "Run"}, References: []backend.Reference{
		{File: "one.go", EnclosingSymbol: "Serve", EnclosingDeclaration: &backend.ReadSymbol{Name: "Serve", Kind: "function", File: "one.go", Range: backend.Range{Start: backend.Position{Line: 0}, End: backend.Position{Line: 1}}, Signature: &signature}, Range: backend.Range{Start: backend.Position{Line: 2}}},
		{File: "one.go", EnclosingSymbol: "Serve", EnclosingDeclaration: &backend.ReadSymbol{Name: "Serve", Kind: "function", File: "one.go", Range: backend.Range{Start: backend.Position{Line: 0}, End: backend.Position{Line: 1}}, Signature: &signature}, Range: backend.Range{Start: backend.Position{Line: 3, Character: 4}}},
		{File: "two.go", EnclosingSymbol: "Serve", Range: backend.Range{Start: backend.Position{Line: 4}}},
		{File: "free.go", Range: backend.Range{Start: backend.Position{Line: 6}}, Snippet: "line one\n" + strings.Repeat("very-long-context ", 12)},
	}}
	output := renderReferencesNavigation(result)
	for _, want := range []string{"source: working", "one.go:1  func Serve()", "references at L3:C1, L4:C5", "two.go | symbol: Serve", "free.go:L7:C1"} {
		if !strings.Contains(output, want) {
			t.Errorf("references output missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "package scope") {
		t.Fatalf("references output invented an enclosing scope: %s", output)
	}
	if strings.Count(output, "func Serve()") != 1 {
		t.Fatalf("references output repeated its enclosing declaration: %s", output)
	}
	if !strings.Contains(output, "free.go:L7:C1 line one very-long-context") || !strings.Contains(output, "…") || strings.Contains(output, "line one\n") {
		t.Fatalf("references output did not compact unknown-owner context: %s", output)
	}
}

func TestRenderOutlineNavigationElidesParentBodiesAndIndentsSelectors(t *testing.T) {
	typeSignature := "type Server struct { Field string }"
	result := &backend.OutlineResult{Scope: backend.ReadScope{Kind: "file", Path: "server.go", Complete: true}, Files: []backend.OutlineFile{{
		File: "server.go", Language: backend.LanguageGo, Revision: "sha", SourceView: "working", Symbols: []backend.ReadSymbol{{
			Name: "Server", QualifiedName: "pkg.Server", Kind: "type", File: "server.go", Range: backend.Range{Start: backend.Position{Line: 0}, End: backend.Position{Line: 2}}, Signature: &typeSignature,
			Children: []backend.ReadSymbol{{Name: "Field", QualifiedName: "Server.Field", Kind: "field", File: "server.go", Range: backend.Range{Start: backend.Position{Line: 1}, End: backend.Position{Line: 1}}}},
		}},
	}}}
	output := renderOutlineNavigation(result)
	for _, want := range []string{"source: working", "  server.go:1-2  type Server", "  symbol: pkg.Server", "    server.go:2  field Field", "    symbol: Server.Field"} {
		if !strings.Contains(output, want) {
			t.Errorf("outline output missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "struct { Field string }") || strings.Count(output, "Field") != 2 {
		t.Fatalf("outline output duplicated parent body or child identity: %s", output)
	}
}
