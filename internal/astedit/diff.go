// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"fmt"
	"strings"
)

// UnifiedDiff generates a standard unified diff representation between oldText and newText.
func UnifiedDiff(name string, oldText, newText string) string {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	m, n := len(oldLines), len(newLines)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := range m {
		for j := range n {
			switch {
			case oldLines[i] == newLines[j]:
				dp[i+1][j+1] = dp[i][j] + 1
			case dp[i+1][j] > dp[i][j+1]:
				dp[i+1][j+1] = dp[i][j]
			default:
				dp[i+1][j+1] = dp[i][j+1]
			}
		}
	}

	type editLine struct {
		op   byte
		text string
	}
	var edits []editLine
	i, j := m, n
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && oldLines[i-1] == newLines[j-1]:
			edits = append(edits, editLine{' ', oldLines[i-1]})
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			edits = append(edits, editLine{'+', newLines[j-1]})
			j--
		case i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]):
			edits = append(edits, editLine{'-', oldLines[i-1]})
			i--
		}
	}

	for l, r := 0, len(edits)-1; l < r; l, r = l+1, r-1 {
		edits[l], edits[r] = edits[r], edits[l]
	}

	var sb strings.Builder
	oldName := "a/" + name
	newName := "b/" + name
	if name == "" {
		oldName = "old"
		newName = "new"
	}
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", oldName, newName)
	for _, e := range edits {
		sb.WriteByte(e.op)
		sb.WriteString(e.text)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}
