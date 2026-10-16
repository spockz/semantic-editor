// Package operation verifies that operation ingress enforces the selector byte cap before provider work.
package operation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/suggestions"
	"github.com/spockz/semantic-editor/internal/symbol"
)

type selectorBoundaryProvider struct {
	candidate suggestions.Candidate
	calls     int
}

func (p *selectorBoundaryProvider) Retrieve(context.Context, suggestions.Query) ([]suggestions.Candidate, error) {
	p.calls++
	return []suggestions.Candidate{p.candidate}, nil
}

func TestSuggestionOperationByteLimitIsInclusive(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	candidateName := "Config" + strings.Repeat("A", 249) + "Z"
	if len(candidateName) != 256 {
		t.Fatalf("fixture selector bytes = %d, want 256", len(candidateName))
	}
	cc, _ := suggestionFixture(t, map[string]string{
		"go.mod":    "module example.com/selectorbounds\n\ngo 1.23\n",
		"source.go": fmt.Sprintf("package config\nfunc %s() {}\n", candidateName),
	})
	provider := &selectorBoundaryProvider{candidate: suggestions.Candidate{Selector: candidateName, File: "source.go", Kind: "function"}}
	cc.SuggestionProvider = provider
	query := candidateName[:len(candidateName)-1] + "Y"
	_, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": query})
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("256-byte selector failure identity = %v", err)
	}
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != candidateName {
		t.Fatalf("256-byte selector suggestions = %#v (%v)", alternatives, err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls at inclusive limit = %d, want 1", provider.calls)
	}
	over := query + "A"
	if len(over) != 257 {
		t.Fatalf("over-limit selector bytes = %d, want 257", len(over))
	}
	_, err = cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": over})
	if !errors.Is(err, symbol.ErrNotFound) || len(SuggestionsFromError(err)) != 0 {
		t.Fatalf("257-byte selector was not refused without suggestions: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider ran for over-limit selector: calls = %d", provider.calls)
	}
}
