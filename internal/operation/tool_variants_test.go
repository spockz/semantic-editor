// This file locks down strict verb-kind filter parsing so catalog flags cannot silently select nothing.
package operation_test

import (
	"testing"

	"semedit/internal/operation"
)

func TestParseToolVariants(t *testing.T) {
	entries := []operation.Entry{
		{Key: "insert_function", VariantVerb: "insert", VariantKind: "function"},
		{Key: "insert_type", VariantVerb: "insert", VariantKind: "type"},
		{Key: "replace_body", VariantVerb: "replace", VariantKind: "function"},
		{Key: "semantic_insert_construct"},
	}
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr bool
	}{
		{name: "insert wildcard", input: []string{"insert:*"}, want: []string{"insert_function", "insert_type"}},
		{name: "kind wildcard and comma", input: []string{"*:function,insert:type"}, want: []string{"insert_function", "insert_type", "replace_body"}},
		{name: "deduplicates patterns", input: []string{"insert:function", "insert:function"}, want: []string{"insert_function"}},
		{name: "all variants", input: []string{"*:*"}, want: []string{"insert_function", "insert_type", "replace_body"}},
		{name: "reject malformed", input: []string{"insert"}, wantErr: true},
		{name: "reject unknown pair", input: []string{"remove:function"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotMap, err := operation.ParseToolVariants(test.input, entries)
			if test.wantErr {
				if err == nil {
					t.Fatal("ParseToolVariants returned nil error")
				}
				if len(gotMap) != 0 {
					t.Fatalf("selected variants on failure: %v", gotMap)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseToolVariants returned error: %v", err)
			}
			got := make([]string, 0, len(gotMap))
			for key := range gotMap {
				got = append(got, key)
			}
			want := append([]string(nil), test.want...)
			// Selection is a set; assert stable registry-order results separately.
			if len(got) != len(want) {
				t.Fatalf("selected %v; want %v", got, want)
			}
			for _, key := range want {
				if _, ok := gotMap[key]; !ok {
					t.Errorf("selected %v; missing %q", gotMap, key)
				}
			}
		})
	}
}
