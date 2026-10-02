// This file plans declaration doc-comment edits while preserving source bytes and refusing directive ambiguity.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// SetDocCommentIntent distinguishes a missing comment from an explicit empty removal.
type SetDocCommentIntent struct {
	Symbol  string
	Comment OptionalString
}

// PlanSetDocComment edits only a uniquely owned top-level declaration doc comment.
func PlanSetDocComment(source []byte, intent SetDocCommentIntent) (SourcePlan, error) {
	if !intent.Comment.Present {
		return SourcePlan{}, fmt.Errorf("comment must be present; use an explicit empty string to remove it")
	}
	if intent.Comment.Null {
		return SourcePlan{}, fmt.Errorf("comment cannot be null; use an explicit empty string to remove it")
	}
	if !token.IsIdentifier(intent.Symbol) || token.Lookup(intent.Symbol).IsKeyword() {
		return SourcePlan{}, fmt.Errorf("declaration symbol %q must be a simple Go identifier", intent.Symbol)
	}
	if err := validateDocText(intent.Comment.Value); err != nil {
		return SourcePlan{}, err
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	target, doc, err := findDocCommentTarget(file, intent.Symbol)
	if err != nil {
		return SourcePlan{}, err
	}
	if err := validateDocCommentOwnership(source, fset, file, target, doc, intent.Symbol); err != nil {
		return SourcePlan{}, err
	}
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	replacement := []byte(nil)
	if intent.Comment.Value != "" {
		var lines []string
		for line := range strings.SplitSeq(strings.ReplaceAll(intent.Comment.Value, "\r\n", "\n"), "\n") {
			lines = append(lines, "// "+line)
		}
		replacement = []byte(strings.Join(lines, newline) + newline)
	}
	start := fset.Position(target.Pos()).Offset
	end := start
	if doc != nil {
		start = fset.Position(doc.Pos()).Offset
		end, err = rawCommentGroupEnd(source, fset, doc)
		if err != nil {
			return SourcePlan{}, err
		}
		for end < len(source) && (source[end] == ' ' || source[end] == '\t') {
			end++
		}
		if end+1 < len(source) && source[end] == '\r' && source[end+1] == '\n' {
			end += 2
		} else if end < len(source) && source[end] == '\n' {
			end++
		}
	}
	updated := spliceSource(source, start, end, replacement)
	return newSourcePlan(source, updated, PlanSelection{
		Kind: "declaration_comment", Target: intent.Symbol,
		Start: fset.Position(target.Pos()).Offset, End: fset.Position(target.End()).Offset,
	}), nil
}

func rawCommentGroupEnd(source []byte, fset *token.FileSet, group *ast.CommentGroup) (int, error) {
	if group == nil || len(group.List) == 0 {
		return 0, fmt.Errorf("comment range is empty")
	}
	last := group.List[len(group.List)-1]
	start := fset.Position(last.Slash).Offset
	if start < 0 || start >= len(source) {
		return 0, fmt.Errorf("comment start %d is outside source of %d bytes", start, len(source))
	}
	comment := source[start:]
	if bytes.HasPrefix(comment, []byte("//")) {
		if newline := bytes.IndexByte(comment, '\n'); newline >= 0 {
			return start + newline, nil
		}
		return len(source), nil
	}
	if bytes.HasPrefix(comment, []byte("/*")) {
		if close := bytes.Index(comment[2:], []byte("*/")); close >= 0 {
			return start + 2 + close + 2, nil
		}
	}
	return 0, fmt.Errorf("comment at byte %d has no supported raw terminator", start)
}

func findDocCommentTarget(file *ast.File, symbol string) (ast.Node, *ast.CommentGroup, error) {
	var target ast.Node
	var doc *ast.CommentGroup
	for _, decl := range file.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if value.Recv == nil && value.Name.Name == symbol {
				if target != nil {
					return nil, nil, fmt.Errorf("declaration %q is ambiguous", symbol)
				}
				target, doc = value, value.Doc
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				name, specDoc, err := commentSpecName(spec, symbol)
				if err != nil {
					return nil, nil, err
				}
				if name != symbol {
					continue
				}
				if len(value.Specs) != 1 {
					return nil, nil, fmt.Errorf("declaration %q is grouped; comment ownership is unsupported", symbol)
				}
				if target != nil {
					return nil, nil, fmt.Errorf("declaration %q is ambiguous", symbol)
				}
				target, doc = value, value.Doc
				if doc == nil {
					doc = specDoc
				}
			}
		}
	}
	if target == nil {
		return nil, nil, fmt.Errorf("top-level declaration %q not found", symbol)
	}
	return target, doc, nil
}

func commentSpecName(spec ast.Spec, symbol string) (string, *ast.CommentGroup, error) {
	switch item := spec.(type) {
	case *ast.TypeSpec:
		return item.Name.Name, item.Doc, nil
	case *ast.ValueSpec:
		if len(item.Names) > 1 {
			for _, ident := range item.Names {
				if ident.Name == symbol {
					return "", nil, fmt.Errorf("declaration %q shares a ValueSpec; comment ownership is unsupported", symbol)
				}
			}
		}
		for _, ident := range item.Names {
			if ident.Name == symbol {
				return ident.Name, item.Doc, nil
			}
		}
	}
	return "", nil, nil
}

func validateDocCommentOwnership(source []byte, fset *token.FileSet, file *ast.File, target ast.Node, doc *ast.CommentGroup, symbol string) error {
	for _, comment := range file.Comments {
		if doc != nil && comment == doc || comment.End() >= target.Pos() {
			continue
		}
		gap := source[fset.Position(comment.End()).Offset:fset.Position(target.Pos()).Offset]
		if strings.TrimSpace(string(gap)) == "" && strings.Count(string(gap), "\n") <= 1 {
			return fmt.Errorf("detached comment ownership before %q is ambiguous", symbol)
		}
	}
	if containsDirectiveGroup(doc) {
		return fmt.Errorf("existing documentation comment for %q contains a directive", symbol)
	}
	return nil
}

func validateDocText(comment string) error {
	if strings.ContainsAny(comment, "\r") || strings.Contains(comment, "/*") || strings.Contains(comment, "*/") || strings.Contains(comment, "//") {
		return fmt.Errorf("comment must be plain text without Go comment delimiters or carriage returns")
	}
	if containsDirective(comment) {
		return fmt.Errorf("comment text contains a compiler or build directive")
	}
	return nil
}

func containsDirective(text string) bool {
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range []string{"go:", "+build", "line ", "export ", "#cgo"} {
			if strings.HasPrefix(trimmed, prefix) {
				return true
			}
		}
	}
	return false
}

func containsDirectiveGroup(group *ast.CommentGroup) bool {
	if group == nil {
		return false
	}
	for _, comment := range group.List {
		text := strings.TrimSpace(comment.Text)
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "//"), "/*"))
		for _, prefix := range []string{"go:", "+build", "line ", "export ", "#cgo"} {
			if strings.HasPrefix(text, prefix) {
				return true
			}
		}
	}
	return false
}
