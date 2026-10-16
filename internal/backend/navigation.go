// Package backend renders compact declaration rows shared by navigation results and ambiguity errors.
package backend

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

// NavigationDeclaration is the presentation-safe identity and inclusive line span of one declaration.
type NavigationDeclaration struct {
	File        string
	Name        string
	Selector    string
	Description string
	StartLine   int
	EndLine     int
	StartColumn int
	PointOnly   bool
}

// RenderNavigationDeclaration renders one declaration without duplicating serialized payloads.
func RenderNavigationDeclaration(summary NavigationDeclaration) string {
	value := fmt.Sprintf("%s:%d", summary.File, summary.StartLine)
	if summary.PointOnly && summary.StartColumn > 0 {
		value += fmt.Sprintf(":%d", summary.StartColumn)
	} else if summary.EndLine > summary.StartLine {
		value += fmt.Sprintf("-%d", summary.EndLine)
	}
	value += "  " + summary.Description
	if summary.Selector != "" && summary.Selector != summary.Name {
		value += "\n  symbol: " + summary.Selector
	}
	return value
}

// NavigationDescription prefers one-line signatures and falls back to the native kind and name.
func NavigationDescription(kind, name string, signature *string) string {
	if signature != nil {
		if value := strings.Join(strings.Fields(*signature), " "); value != "" {
			return value
		}
	}
	return strings.TrimSpace(kind + " " + name)
}

// RenderAmbiguousCandidates renders every candidate as a compact, line-oriented declaration row.
func RenderAmbiguousCandidates(root string, candidates []*NamedTarget) (string, error) {
	rows := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			return "", fmt.Errorf("render ambiguous candidate: nil target")
		}
		file, err := pathutil.FilePathFromURI(candidate.Location.URI)
		if err != nil {
			return "", fmt.Errorf("render ambiguous candidate location: %w", err)
		}
		relative, err := filepath.Rel(root, file)
		if err != nil {
			return "", fmt.Errorf("render ambiguous candidate path: %w", err)
		}
		start := candidate.Location.Range.Start
		endLine := start.Line + 1
		pointOnly := true
		if candidate.DeclarationRange != nil {
			start = candidate.DeclarationRange.Start
			endLine = candidate.DeclarationRange.End.Line + 1
			if candidate.DeclarationRange.End.Line > candidate.DeclarationRange.Start.Line && candidate.DeclarationRange.End.Character == 0 {
				endLine--
			}
			pointOnly = false
		}
		rows = append(rows, RenderNavigationDeclaration(NavigationDeclaration{File: filepath.ToSlash(relative), Name: candidate.Name, Selector: candidate.Identity(), Description: NavigationDescription(candidate.Kind, candidate.Name, candidate.Signature), StartLine: start.Line + 1, EndLine: endLine, StartColumn: start.Character + 1, PointOnly: pointOnly}))
	}
	return strings.Join(rows, "\n"), nil
}
