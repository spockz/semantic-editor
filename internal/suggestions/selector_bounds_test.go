// Package suggestions verifies the exact Unicode-aware selector length boundary.
package suggestions

import (
	"strings"
	"testing"
)

func TestMatcherSelectorRuneLimitIsInclusive(t *testing.T) {
	within := strings.Repeat("界", 256)
	got, err := (Matcher{}).Retrieve(t.Context(), Query{Selector: within, Candidates: []Candidate{{Name: within}}, Limit: 3})
	if err != nil || len(got) != 1 || got[0].Name != within {
		t.Fatalf("256-rune selector result = %#v, %v", got, err)
	}
	over := within + "界"
	got, err = (Matcher{}).Retrieve(t.Context(), Query{Selector: over, Candidates: []Candidate{{Name: over}}, Limit: 3})
	if err != nil || len(got) != 0 {
		t.Fatalf("257-rune selector result = %#v, %v; want none", got, err)
	}
}
