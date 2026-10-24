// Package operation proves extraction suggestions replace only an existing owner
// and retain caller-chosen helper names and native selection constraints.
package operation

import (
	"context"
	"errors"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func TestExtractionSuggestionsValidateOwnerRetries(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	for _, test := range []struct {
		key string
		raw map[string]any
	}{
		{canonicalExtractCallableKey, map[string]any{"kind": "function", "start_line": 4, "end_line": 4}},
		{canonicalExtractValueKey, map[string]any{"discriminator": "x"}},
	} {
		t.Run(test.key, func(t *testing.T) {
			cc, buffer := suggestionFixture(t, map[string]string{
				"go.mod":    "module example.com/extractionsuggestions\n\ngo 1.23\n",
				"source.go": "package config\nfunc consume(x int) {}\nfunc ParseConfig(x int) {\n consume(x)\n}\n",
			})
			raw := cloneRaw(test.raw)
			raw["file"], raw["in_function"], raw["symbol"] = "source.go", "ParseConfg", "ParseConff"
			_, err := cc.Registry.Dispatch(cc, test.key, raw)
			alternatives := SuggestionsFromError(err)
			if !errors.Is(err, symbol.ErrNotFound) || len(alternatives) != 1 || alternatives[0].Arguments["in_function"] != "ParseConfig" {
				t.Fatalf("owner alternatives: %#v (%v)", alternatives, err)
			}
			if len(buffer.ChangedPaths()) != 0 {
				t.Fatal("extraction suggestions changed sources")
			}
			for _, alternative := range alternatives {
				if _, replaced := alternative.Arguments["symbol"]; replaced {
					t.Fatal("suggestion replaced a name being introduced")
				}
				retry := cloneRaw(raw)
				for key, value := range alternative.Arguments {
					retry[key] = value
				}
				retryCall := cc
				retryCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
				if _, retryErr := cc.Registry.Dispatch(retryCall, test.key, retry); retryErr != nil {
					t.Fatalf("extraction retry: %v", retryErr)
				}
			}
		})
	}
}
