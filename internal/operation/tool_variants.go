// Package operation owns explicit verb-kind metadata for optional tool variants.
package operation

import (
	"fmt"
	"strings"
)

// ToolVariant selects specialized operations by their semantic verb and construct kind.
type ToolVariant struct {
	Verb string
	Kind string
}

// ParseToolVariants parses repeatable and comma-separated verb-kind patterns and
// rejects patterns that do not select a registered specialized operation.
func ParseToolVariants(values []string, entries []Entry) (map[string]struct{}, error) {
	patterns := make([]ToolVariant, 0, len(values))
	seenPatterns := map[ToolVariant]struct{}{}
	for _, value := range values {
		for raw := range strings.SplitSeq(value, ",") {
			parts := strings.Split(strings.TrimSpace(raw), ":")
			if len(parts) != 2 || !validVariantPart(parts[0]) || !validVariantPart(parts[1]) {
				return nil, fmt.Errorf("invalid tool variant %q; expected verb:kind, such as insert:function", strings.TrimSpace(raw))
			}
			pattern := ToolVariant{Verb: parts[0], Kind: parts[1]}
			if _, ok := seenPatterns[pattern]; !ok {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = struct{}{}
			}
		}
	}

	selected := map[string]struct{}{}
	for _, pattern := range patterns {
		matched := false
		for _, entry := range entries {
			if entry.VariantVerb == "" || entry.VariantKind == "" || !variantMatches(pattern, entry.VariantVerb, entry.VariantKind) {
				continue
			}
			selected[entry.Key] = struct{}{}
			matched = true
		}
		if !matched {
			return nil, fmt.Errorf("tool variant %q matched no registered operation", pattern.String())
		}
	}
	return selected, nil
}

func validVariantPart(value string) bool {
	if value == "*" || value == "" {
		return value == "*"
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func variantMatches(pattern ToolVariant, verb, kind string) bool {
	return (pattern.Verb == "*" || pattern.Verb == verb) && (pattern.Kind == "*" || pattern.Kind == kind)
}

func (variant ToolVariant) String() string { return variant.Verb + ":" + variant.Kind }
