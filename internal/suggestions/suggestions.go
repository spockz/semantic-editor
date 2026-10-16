// Package suggestions retrieves a small, deterministic set of plausible
// declarations for a host semantic operation to resolve authoritatively.
package suggestions

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	defaultLimit = 5
	maximumLimit = 50
)

// Candidate is a declaration found by a language-native enumeration source.
type Candidate struct {
	Selector  string
	File      string
	Name      string
	Kind      string
	Signature string
	Identity  string
	Language  string
}

// Query supplies a selector and the native enumeration to rank.
type Query struct {
	Selector        string
	Operation       string
	TargetParameter string
	Candidates      []Candidate
	Limit           int
	Language        string
	Scope           string
	Kinds           []string
	Receiver        string
	Signature       string
}

// Provider retrieves bounded candidate declarations for a selector query.
type Provider interface {
	Retrieve(context.Context, Query) ([]Candidate, error)
}

// Matcher ranks a caller-supplied native candidate enumeration.
type Matcher struct{}

// Retrieve returns plausible candidates in deterministic order. It does not
// establish that a candidate is valid for a semantic operation.
func (Matcher) Retrieve(ctx context.Context, query Query) ([]Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	selector := strings.ToLower(query.Selector)
	selectorLength := utf8.RuneCountInString(selector)
	if selector == "" || selectorLength > 256 {
		return []Candidate{}, nil
	}

	type ranked struct {
		candidate Candidate
		distance  int
		exact     bool
	}
	items := make([]ranked, 0, len(query.Candidates))
	for _, candidate := range query.Candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := candidate.Selector
		if name == "" {
			name = candidate.Name
		}
		name = matcherCandidateName(query, name)
		separator := "."
		if strings.EqualFold(query.Language, "rust") {
			separator = "::"
		}
		if !strings.Contains(query.Selector, separator) {
			if index := strings.LastIndex(name, separator); index >= 0 {
				name = name[index+len(separator):]
			}
		}
		name = strings.ToLower(name)
		if utf8.RuneCountInString(name) > 256 {
			continue
		}
		threshold := maxDistance(selectorLength)
		distance := damerauLevenshteinWithin(selector, name, threshold)
		if distance > threshold {
			continue
		}
		items = append(items, ranked{candidate: candidate, distance: distance, exact: selector == name})
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.exact != b.exact {
			return a.exact
		}
		if a.distance != b.distance {
			return a.distance < b.distance
		}
		return candidateLess(a.candidate, b.candidate)
	})

	limit := query.Limit
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maximumLimit {
		limit = maximumLimit
	}
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]Candidate, len(items))
	for i := range items {
		result[i] = items[i].candidate
	}
	return result, nil
}

func matcherCandidateName(query Query, candidate string) string {
	if !strings.EqualFold(query.Language, "java") || strings.Contains(query.Selector, "(") {
		return candidate
	}
	if open := strings.IndexByte(candidate, '('); open >= 0 {
		candidate = candidate[:open]
	}
	if strings.Contains(query.Selector, ".") {
		parts := strings.Split(candidate, ".")
		queryParts := strings.Count(query.Selector, ".") + 1
		if len(parts) > queryParts {
			candidate = strings.Join(parts[len(parts)-queryParts:], ".")
		}
	}
	return candidate
}

func candidateLess(a, b Candidate) bool {
	for _, pair := range [][2]string{
		{a.Selector, b.Selector}, {a.File, b.File}, {a.Name, b.Name},
		{a.Kind, b.Kind}, {a.Signature, b.Signature}, {a.Identity, b.Identity},
		{a.Language, b.Language},
	} {
		if pair[0] != pair[1] {
			return pair[0] < pair[1]
		}
	}
	return false
}

func maxDistance(length int) int {
	switch {
	case length <= 3:
		return 0
	case length <= 7:
		return 1
	case length <= 16:
		return 2
	default:
		return 3
	}
}

// damerauLevenshteinWithin uses the optimal-string-alignment variant and only
// computes the band that can satisfy the conservative threshold.
func damerauLevenshteinWithin(a, b string, threshold int) int {
	left, right := []rune(a), []rune(b)
	if len(left)-len(right) > threshold || len(right)-len(left) > threshold {
		return threshold + 1
	}
	infinity := threshold + 1
	previousPrevious := make([]int, len(right)+1)
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = infinity
	}
	for j := 0; j <= min(len(right), threshold); j++ {
		previous[j] = j
	}
	for i := 1; i <= len(left); i++ {
		for j := range current {
			current[j] = infinity
		}
		if i <= threshold {
			current[0] = i
		}
		low := max(1, i-threshold)
		high := min(len(right), i+threshold)
		rowMinimum := infinity
		for j := low; j <= high; j++ {
			cost := 0
			if left[i-1] != right[j-1] {
				cost = 1
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
			if i > 1 && j > 1 && left[i-1] == right[j-2] && left[i-2] == right[j-1] {
				current[j] = min(current[j], previousPrevious[j-2]+1)
			}
			rowMinimum = min(rowMinimum, current[j])
		}
		if rowMinimum > threshold {
			return infinity
		}
		previousPrevious, previous, current = previous, current, previousPrevious
	}
	return previous[len(right)]
}
