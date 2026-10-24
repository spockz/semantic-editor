// Package adr provides deterministic, paginated retrieval over validated decision records.
package adr

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode"
)

// SearchOptions selects applicable ADRs and a page of ranked results.
type SearchOptions struct {
	Query     string
	Role      string
	Scope     string
	Area      string
	Languages []string
	Offset    int
	Limit     int
}

// SearchPage reports the total filtered result count and one stable result page.
type SearchPage struct {
	Total  int         `json:"total"`
	Offset int         `json:"offset"`
	Limit  int         `json:"limit"`
	Items  []SearchHit `json:"items"`
}

// SearchHit returns a compact summary and an ADR path relative to the workspace root.
type SearchHit struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Role        string   `json:"role"`
	Scope       string   `json:"scope"`
	Area        string   `json:"area"`
	Languages   []string `json:"languages"`
	Path        string   `json:"path"`
}

type scoredRecord struct {
	record Record
	score  int
}

// Search ranks records by exact lexical matches and applies optional taxonomy filters.
func Search(records []Record, options SearchOptions) (SearchPage, error) {
	if options.Offset < 0 {
		return SearchPage{}, fmt.Errorf("offset must not be negative")
	}
	if options.Limit == 0 {
		options.Limit = 10
	}
	if options.Limit < 1 || options.Limit > 100 {
		return SearchPage{}, fmt.Errorf("limit must be between 1 and 100")
	}
	queryTerms := terms(options.Query)
	filtered := make([]scoredRecord, 0, len(records))
	for _, record := range records {
		if !matchesFilter(record, options) {
			continue
		}
		score := scoreRecord(record, queryTerms)
		if len(queryTerms) > 0 && score == 0 {
			continue
		}
		filtered = append(filtered, scoredRecord{record: record, score: score})
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].score != filtered[j].score {
			return filtered[i].score > filtered[j].score
		}
		return filtered[i].record.ID < filtered[j].record.ID
	})
	page := SearchPage{Total: len(filtered), Offset: options.Offset, Limit: options.Limit, Items: make([]SearchHit, 0)}
	if options.Offset >= len(filtered) {
		return page, nil
	}
	end := min(options.Offset+options.Limit, len(filtered))
	for _, item := range filtered[options.Offset:end] {
		record := item.record
		page.Items = append(page.Items, SearchHit{
			ID: "ADR-" + record.ID, Title: record.Title, Status: record.Status,
			Summary: record.Summary, Description: record.Description,
			Role: record.Role, Scope: record.Scope, Area: record.Area,
			Languages: slices.Clone(record.Languages), Path: filepathSlash(record.Path),
		})
	}
	return page, nil
}

func matchesFilter(record Record, options SearchOptions) bool {
	if options.Role != "" && !strings.EqualFold(record.Role, options.Role) {
		return false
	}
	if options.Scope != "" && !strings.EqualFold(record.Scope, options.Scope) {
		return false
	}
	if options.Area != "" && !strings.EqualFold(record.Area, options.Area) {
		return false
	}
	for _, requested := range options.Languages {
		if !slices.ContainsFunc(record.Languages, func(language string) bool { return strings.EqualFold(language, requested) }) {
			return false
		}
	}
	return true
}

func scoreRecord(record Record, queryTerms []string) int {
	score := 0
	title := words(record.Title)
	summary := words(record.Summary)
	description := words(record.Description)
	taxonomy := words(strings.Join([]string{record.Role, record.Scope, record.Area, strings.Join(record.Languages, " ")}, " "))
	body := words(strings.Join([]string{record.Context, record.Decision, record.Invariants, record.Consequences}, " "))
	for _, term := range queryTerms {
		if slices.Contains(title, term) {
			score += 8
		}
		if slices.Contains(summary, term) {
			score += 5
		}
		if slices.Contains(description, term) {
			score += 3
		}
		if slices.Contains(taxonomy, term) {
			score += 2
		}
		if slices.Contains(body, term) {
			score++
		}
	}
	return score
}

func terms(text string) []string {
	return words(text)
}

func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}

func filepathSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
