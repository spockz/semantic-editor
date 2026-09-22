package main

import (
	"strings"
	"testing"
)

func TestOpenRouterFreeCatalogIsTopTenAndToolCapable(t *testing.T) {
	if got, want := len(openRouterFreeCatalog), 10; got != want {
		t.Fatalf("catalog length = %d, want %d", got, want)
	}
	for index, model := range openRouterFreeCatalog {
		if model.Rank != index+1 || !strings.HasSuffix(model.ID, ":free") || !model.SupportsToolCall {
			t.Errorf("catalog entry %d is not a free tool-capable model: %+v", index+1, model)
		}
	}
	targets := openRouterTopTargets()
	if got, want := len(targets), len(openRouterFreeCatalog); got != want {
		t.Fatalf("target count = %d, want %d", got, want)
	}
	if !strings.HasPrefix(targets[0].Model, "openrouter/") {
		t.Fatalf("target model = %q, want OpenRouter provider prefix", targets[0].Model)
	}
}
