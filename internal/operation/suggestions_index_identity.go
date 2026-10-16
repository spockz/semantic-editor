// Package operation rejects stale index identities after destructive batch steps
// instead of attaching an old indexed declaration to a new declaration with its name.
package operation

import (
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/suggestions"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func indexedCandidateInvalidated(root string, candidate suggestions.Candidate, invalidated []sourcebuf.SymbolAlias) bool {
	for _, alias := range invalidated {
		if filepath.Ext(candidate.File) != filepath.Ext(alias.File) {
			continue
		}
		inScope := sameSuggestionFile(root, candidate.File, alias.File) || sameSuggestionFile(root, candidate.File, alias.NewFile)
		if alias.Kind == "type" || alias.Kind == "class" || alias.Kind == "interface" || alias.Kind == "record" || alias.Kind == "enum" {
			inScope = inScope || filepath.Dir(resolveWorkPath(root, candidate.File)) == filepath.Dir(resolveWorkPath(root, alias.File))
		}
		if inScope && (indexedSelectorOverlaps(candidate.Selector, alias.OldSelector) || indexedSelectorOverlaps(candidate.Selector, alias.NewSelector)) {
			return true
		}
	}
	return false
}

func indexedSelectorOverlaps(candidate, invalidated string) bool {
	if candidate == "" || invalidated == "" {
		return false
	}
	candidate, invalidated = indexedSelectorIdentity(candidate), indexedSelectorIdentity(invalidated)
	if candidate == invalidated || strings.HasPrefix(candidate, invalidated+".") {
		return true
	}
	return !strings.Contains(candidate, ".") && strings.HasSuffix(invalidated, "."+candidate)
}

func indexedSelectorIdentity(selector string) string {
	if open := strings.IndexByte(selector, '('); open > 0 && strings.HasSuffix(selector, ")") {
		selector = selector[:open]
	}
	if receiver, name, err := symbol.ParseIdentifier(selector); err == nil && receiver != "" {
		return receiver + "." + name
	}
	return selector
}
