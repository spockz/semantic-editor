// WHY: Proves the multi-site selector parses for Go and is rejected for Java and Rust.
package operation

import (
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
)

func TestParseInlineFunctionOccurrenceIndicesAreGoOnly(t *testing.T) {
	t.Parallel()
	base := map[string]any{
		"file":   "helper.go",
		"symbol": "adjust",
		"call_site": map[string]any{
			"file": "caller.go", "in_function": "Run", wireDiscriminator: "adjust()",
			wireExpectedRevision: strings.Repeat("a", 64), "occurrence_indices": []any{0, 1},
		},
	}
	request := cloneRaw(base)
	parsed, err := parseInlineFunction(request)
	if err != nil {
		t.Fatalf("parse Go multi-site request: %v", err)
	}
	if len(parsed.CallSite.OccurrenceIndices) != 2 || parsed.CallSite.OccurrenceIndices[0] != 0 || parsed.CallSite.OccurrenceIndices[1] != 1 {
		t.Fatalf("parsed occurrence indices = %v, want [0 1]", parsed.CallSite.OccurrenceIndices)
	}

	for _, language := range []backend.LanguageID{backend.LanguageJava, backend.LanguageRust} {
		t.Run(string(language), func(t *testing.T) {
			t.Parallel()
			raw := map[string]any{
				"file": "helper.java", "symbol": "adjust", "language": string(language),
				"call_site": map[string]any{
					"file": "Caller.java", "in_function": "Caller.run()", wireDiscriminator: "adjust()",
					wireExpectedRevision: strings.Repeat("a", 64), "occurrence_indices": []any{0, 1},
				},
			}
			if language == backend.LanguageRust {
				raw["file"] = "helper.rs"
				raw["call_site"].(map[string]any)["file"] = "main.rs"
				raw["call_site"].(map[string]any)["in_function"] = "run"
			}
			_, err := parseInlineFunction(raw)
			if err == nil || !strings.Contains(err.Error(), "supported only for Go") {
				t.Fatalf("parse non-Go multi-site request error = %v, want Go-only refusal", err)
			}
		})
	}
}
