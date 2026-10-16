// Package operation proves stale indexed identities cannot attach to declarations
// newly inserted under a deleted symbol's name in a retained batch view.
package operation

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/suggestions"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func TestIndexedSuggestionRejectsDeletedIdentityWithReusedName(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	cc, buffer := suggestionFixture(t, map[string]string{"source.go": "package config\nfunc ParseConfig() int { return 1 }\n"})
	cc.SuggestionProvider = fixedSuggestionProvider{candidates: []suggestions.Candidate{
		{Selector: "ParseConfig", File: "source.go", Kind: "function", Identity: "old-index-identity"},
	}}
	if _, err := cc.Registry.Dispatch(cc, "remove_construct", map[string]any{"file": "source.go", "kind": "function", "symbol": "ParseConfig"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cc.Registry.Dispatch(cc, "insert_construct", map[string]any{"file": "source.go", "kind": "function", "source": "func ParseConfig() int { return 2 }"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cc.WorkDir, "source.go")
	before, err := sourcebuf.ReadFile(cc.Ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cc.Registry.Dispatch(cc, "lookup", map[string]any{"file": "source.go", "symbol": "ParseConfg"})
	if !errors.Is(err, symbol.ErrNotFound) || len(SuggestionsFromError(err)) != 0 {
		t.Fatalf("stale index identity was redirected: %v", err)
	}
	after, err := sourcebuf.ReadFile(cc.Ctx, path)
	if err != nil || !bytes.Equal(before, after) || len(buffer.InvalidatedSymbolAliases()) == 0 {
		t.Fatalf("suggestions changed the retained state or lost invalidation: %v", err)
	}
}
