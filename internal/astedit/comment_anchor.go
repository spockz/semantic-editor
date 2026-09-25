// Package astedit contains insertion-offset helpers that preserve comments attached to Go syntax.
package astedit

import (
	"bytes"
	"go/ast"
	"go/token"
	"strings"
)

func normalizeInsertionOffset(fset *token.FileSet, file *ast.File, content []byte, offset int) int {
	if fset == nil || file == nil || offset < 0 || offset > len(content) {
		return offset
	}

	for _, group := range file.Comments {
		commentStart := fset.Position(group.Pos()).Offset
		commentEnd := fset.Position(group.End()).Offset
		if commentStart < 0 || commentEnd > offset || commentEnd > len(content) {
			continue
		}
		lineStart := bytes.LastIndexByte(content[:commentStart], '\n') + 1
		if len(bytes.TrimSpace(content[lineStart:commentStart])) != 0 {
			continue
		}
		gap := content[commentEnd:offset]
		lineBreaks := bytes.Count(gap, []byte("\n"))
		sameLineBlockComment := lineBreaks == 0 && bytes.HasPrefix(content[commentStart:], []byte("/*"))
		if (lineBreaks != 1 && !sameLineBlockComment) || len(bytes.Trim(gap, " \t\r\n")) != 0 {
			continue
		}
		if offset < len(content) && !strings.ContainsRune(" \t\r\n", rune(content[offset])) {
			return lineStart
		}
	}

	lineStart := bytes.LastIndexByte(content[:offset], '\n') + 1
	if len(bytes.TrimSpace(content[lineStart:offset])) == 0 {
		return offset
	}
	for _, group := range file.Comments {
		commentStart := fset.Position(group.Pos()).Offset
		commentEnd := fset.Position(group.End()).Offset
		if commentStart < offset || commentEnd > len(content) {
			continue
		}
		gap := content[offset:commentStart]
		if bytes.ContainsAny(gap, "\r\n") || len(bytes.Trim(gap, " \t")) != 0 {
			continue
		}
		lineEnd := bytes.IndexByte(content[commentEnd:], '\n')
		if lineEnd < 0 {
			return len(content)
		}
		return commentEnd + lineEnd + 1
	}
	return offset
}
