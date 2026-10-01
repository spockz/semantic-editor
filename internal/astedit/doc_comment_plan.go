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
	var target ast.Node
	var doc *ast.CommentGroup
	for _, decl := range file.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if value.Recv == nil && value.Name.Name == intent.Symbol {
				if target != nil {
					return SourcePlan{}, fmt.Errorf("declaration %q is ambiguous", intent.Symbol)
				}
				target, doc = value, value.Doc
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				var name string
				var specDoc *ast.CommentGroup
				switch item := spec.(type) {
				case *ast.TypeSpec:
					name, specDoc = item.Name.Name, item.Doc
				case *ast.ValueSpec:
					specDoc = item.Doc
					for _, ident := range item.Names {
						if ident.Name == intent.Symbol {
							name = ident.Name
							break
						}
					}
				}
				if name != intent.Symbol {
					continue
				}
				if len(value.Specs) != 1 {
					return SourcePlan{}, fmt.Errorf("declaration %q is grouped; comment ownership is unsupported", intent.Symbol)
				}
				if target != nil {
					return SourcePlan{}, fmt.Errorf("declaration %q is ambiguous", intent.Symbol)
				}
				target = value
				doc = value.Doc
				if doc == nil {
					doc = specDoc
				}
			}
		}
	}
	if target == nil {
		return SourcePlan{}, fmt.Errorf("top-level declaration %q not found", intent.Symbol)
	}
	for _, comment := range file.Comments {
		if doc != nil && comment == doc {
			continue
		}
		if comment.End() >= target.Pos() {
			continue
		}
		gap := source[fset.Position(comment.End()).Offset:fset.Position(target.Pos()).Offset]
		if strings.TrimSpace(string(gap)) == "" && strings.Count(string(gap), "\n") <= 1 {
			return SourcePlan{}, fmt.Errorf("detached comment ownership before %q is ambiguous", intent.Symbol)
		}
	}
	if containsDirectiveGroup(doc) {
		return SourcePlan{}, fmt.Errorf("existing documentation comment for %q contains a directive", intent.Symbol)
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
		end = fset.Position(doc.End()).Offset
	}
	updated := spliceSource(source, start, end, replacement)
	return newSourcePlan(source, updated, PlanSelection{
		Kind: "declaration_comment", Target: intent.Symbol,
		Start: fset.Position(target.Pos()).Offset, End: fset.Position(target.End()).Offset,
	}), nil
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
