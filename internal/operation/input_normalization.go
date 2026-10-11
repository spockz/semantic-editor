// Package operation centralizes ingress normalization so aliases disappear before schemas and handlers observe requests.
package operation

import (
	"fmt"
	"maps"
	"strings"
)

func normalizedInput(normalize func(map[string]any) (map[string]any, error), raw map[string]any) (map[string]any, error) {
	input := cloneRaw(raw)
	if normalize == nil {
		return input, nil
	}
	normalized, err := normalize(input)
	if err != nil {
		return nil, err
	}
	if normalized == nil {
		return nil, fmt.Errorf("input normalizer returned nil: %w", ErrInvalidParams)
	}
	copy := make(map[string]any, len(normalized))
	maps.Copy(copy, normalized)
	return copy, nil
}

// NormalizeConstructKindInput maps element vocabulary aliases to the canonical construct kind.
func NormalizeConstructKindInput(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	normalized := cloneRaw(raw)
	kind, ok := normalized["kind"].(string)
	if !ok {
		return normalized, nil
	}
	switch strings.TrimSpace(kind) {
	case "composite_element", "literal_element", "entry", "item":
		normalized["kind"] = "element"
	}
	return normalized, nil
}
