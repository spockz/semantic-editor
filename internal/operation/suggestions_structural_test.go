// Package operation proves structural alternatives preserve requested payloads
// and pass native compiler and layout checks when explicitly retried.
package operation

import (
	"context"
	"errors"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func TestStructuralSuggestionsRetryThroughNativePlans(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	for _, test := range []struct {
		key, selectorKey, expected string
		raw                        map[string]any
	}{
		{"set_doc_comment", "symbol", "ParseConfig", map[string]any{"scope": "declaration", "symbol": "ParseConfg", "comment": "ParseConfig reads settings."}},
		{"insert_field", "in_type", "settings", map[string]any{"in_type": "settngs", "symbol": "settings.timeout", "source": "timeout int", "placement": "type_end"}},
		{"insert_construct", "in_declaration", "routes", map[string]any{"kind": "element", "in_declaration": "routez", "discriminator": "map[string]int{}", "source": "\"ready\": 1"}},
		{"refactor_extract_type", "in_type", "settings", map[string]any{"in_type": "settngs", "symbol": "measurements", "field": "metrics", "members": []any{map[string]any{"file": "source.go", "symbol": "settings.value"}}}},
	} {
		t.Run(test.key, func(t *testing.T) {
			cc, buffer := suggestionFixture(t, map[string]string{
				"go.mod":    "module example.com/structuralsuggestions\n\ngo 1.24\n",
				"source.go": "package config\nfunc ParseConfig() {}\ntype settings struct {\n value int\n}\nconst settengs = 1\nvar routes = map[string]int{}\n",
			})
			raw := cloneRaw(test.raw)
			raw["file"] = "source.go"
			control := cloneRaw(raw)
			control[test.selectorKey] = test.expected
			controlCall := cc
			controlCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
			if _, err := controlCall.Registry.Dispatch(controlCall, test.key, control); err != nil {
				t.Fatalf("valid target control: %v", err)
			}
			_, err := cc.Registry.Dispatch(cc, test.key, raw)
			alternatives := SuggestionsFromError(err)
			if !errors.Is(err, symbol.ErrNotFound) || len(alternatives) != 1 || alternatives[0].Arguments[test.selectorKey] != test.expected {
				t.Fatalf("structural alternatives: %#v (%v)", alternatives, err)
			}
			if len(buffer.ChangedPaths()) != 0 {
				t.Fatal("structural suggestions changed sources")
			}
			for _, alternative := range alternatives {
				retry := cloneRaw(raw)
				for key, value := range alternative.Arguments {
					retry[key] = value
				}
				retryCall := cc
				retryCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
				if _, retryErr := cc.Registry.Dispatch(retryCall, test.key, retry); retryErr != nil {
					t.Fatalf("structural explicit retry: %v", retryErr)
				}
			}
		})
	}
}
