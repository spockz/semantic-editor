// Package operation centralizes ingress normalization so aliases disappear before schemas and handlers observe requests.
package operation

import (
	"fmt"
	symbolpkg "github.com/spockz/semantic-editor/internal/symbol"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"reflect"
	"strings"
)

func normalizedInput(normalize func(map[string]any) (map[string]any, error), raw map[string]any, params []ParameterContract) (map[string]any, error) {
	input := cloneRaw(raw)
	for _, param := range params {
		for _, alias := range param.InputAliases {
			aliasValue, aliasPresent := input[alias]
			if !aliasPresent {
				continue
			}
			canonicalValue, canonicalPresent := rawLookup(input, param.JSONName, param.CLIName)
			if canonicalPresent {
				canonicalText, canonicalIsText := canonicalValue.(string)
				aliasText, aliasIsText := aliasValue.(string)
				if canonicalIsText && strings.TrimSpace(canonicalText) == "" && aliasIsText && strings.TrimSpace(aliasText) != "" {
					input[param.JSONName] = aliasValue
					canonicalValue = aliasValue
				}
				if canonicalValue != nil && aliasValue != nil &&
					nonEmptyInputValue(canonicalValue) && nonEmptyInputValue(aliasValue) &&
					!reflect.DeepEqual(canonicalValue, aliasValue) {
					return nil, fmt.Errorf("conflicting values for param %q and alias %q: %w", param.JSONName, alias, ErrInvalidParams)
				}
			} else {
				input[param.JSONName] = aliasValue
			}
			delete(input, alias)
		}
	}
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

func nonEmptyInputValue(value any) bool {
	if value == nil {
		return false
	}
	text, isText := value.(string)
	return !isText || strings.TrimSpace(text) != ""
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
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "composite_element", "literal_element", "entry", "item":
		kind = "element"
	case "variable":
		kind = "var"
	case "constant":
		kind = "const"
	case "procedure":
		kind = "function"
	}
	normalized["kind"] = kind
	if kind != "function" {
		return normalized, nil
	}
	symbolQuery, ok := normalized["symbol"].(string)
	if !ok || strings.TrimSpace(symbolQuery) == "" {
		return normalized, nil
	}
	receiver, name, err := symbolpkg.ParseIdentifier(strings.TrimSpace(symbolQuery))
	if err != nil {
		return nil, fmt.Errorf("parse replacement symbol selector: %w: %w", err, ErrInvalidParams)
	}
	if receiver == "" {
		return normalized, nil
	}
	source, ok := normalized["source"].(string)
	if !ok {
		return normalized, nil
	}
	snippet := strings.TrimSpace(source)
	if !strings.HasPrefix(snippet, "package ") {
		snippet = "package replacement\n\n" + snippet
	}
	file, err := parser.ParseFile(token.NewFileSet(), "replacement.go", snippet, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse replacement declaration: %w: %w", err, ErrInvalidParams)
	}
	if len(file.Decls) != 1 {
		return normalized, nil
	}
	function, ok := file.Decls[0].(*ast.FuncDecl)
	if ok && function.Name.Name == name && function.Recv != nil {
		normalized["kind"] = "method"
	}
	return normalized, nil
}
