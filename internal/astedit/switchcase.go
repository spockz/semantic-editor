// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/symbol"
	"semedit/internal/telemetry"
)

// CasePlacement specifies relative placement for a newly inserted switch case.
type CasePlacement string

const (
	// CasePlacementFirst inserts as the first case in the switch.
	CasePlacementFirst CasePlacement = "first"
	// CasePlacementLast inserts as the last case in the switch.
	CasePlacementLast CasePlacement = "last"
	// CasePlacementBeforeDefault inserts immediately before the default clause, or as the last case if absent.
	CasePlacementBeforeDefault CasePlacement = "before_default"
	// CasePlacementBefore inserts before the anchor case.
	CasePlacementBefore CasePlacement = "before"
	// CasePlacementAfter inserts after the anchor case.
	CasePlacementAfter CasePlacement = "after"
)

// CaseOptions configures case insertion behavior.
type CaseOptions struct {
	Placement           CasePlacement
	AnchorCase          string
	SwitchPath          string
	AutoOrganizeImports bool
}

// InsertCase injects a case clause into an existing switch statement.
func InsertCase(ctx context.Context, filePath, funcName, switchOn, caseSource string, opts CaseOptions) (string, error) {
	res, err := InsertStructure(ctx, filePath, caseSource, StructureOptions{
		Kind:                StructureKindCase,
		Function:            funcName,
		SwitchOn:            switchOn,
		SwitchPath:          opts.SwitchPath,
		CasePlacement:       opts.Placement,
		AnchorCase:          opts.AnchorCase,
		AutoOrganizeImports: opts.AutoOrganizeImports,
	})
	return res.Diff, err
}

// InsertCase injects a case clause into an existing switch statement.
func insertCaseImpl(ctx context.Context, filePath, funcName, switchOn, caseSource string, opts CaseOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}

	recv, name, err := symbol.ParseIdentifier(funcName)
	if err != nil {
		return "", &symbol.SymbolError{
			Op:     "insert_case",
			File:   filePath,
			Symbol: funcName,
			Err:    err,
		}
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments)
	if err != nil {
		pos := extractSyntaxPosition(fset, err)
		return "", &SyntaxError{
			File:  cleanPath,
			Pos:   pos,
			Cause: err,
			Err:   ErrSyntax,
		}
	}

	var targetFunc *ast.FuncDecl
	for _, decl := range fileNode.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if extractReceiverName(fn.Recv) == recv && fn.Name.Name == name {
			targetFunc = fn
			break
		}
	}

	if targetFunc == nil {
		return "", &symbol.SymbolError{
			Op:     "insert_case",
			File:   filePath,
			Symbol: funcName,
			Err:    symbol.ErrNotFound,
		}
	}

	if targetFunc.Body == nil {
		return "", &symbol.SymbolError{
			Op:     "insert_case",
			File:   filePath,
			Symbol: funcName,
			Pos:    fset.Position(targetFunc.Pos()),
			Err:    ErrNoBody,
		}
	}

	switchBody, err := locateSwitch(fset, targetFunc.Body, strings.TrimSpace(switchOn), opts.SwitchPath)
	if err != nil {
		return "", fmt.Errorf("locate switch %q in %s: %w", switchOn, funcName, err)
	}

	caseTrimmed := strings.TrimSpace(caseSource)
	synthetic := "package dummy\n\nfunc _() {\nswitch {\n" + caseTrimmed + "\n}\n}\n"
	synthFset := token.NewFileSet()
	if _, err := parser.ParseFile(synthFset, "snippet", synthetic, parser.ParseComments); err != nil {
		pos := extractSyntaxPosition(synthFset, err)
		if pos.Line > 4 {
			pos.Line -= 4
		}
		return "", &SyntaxError{
			File:    "snippet",
			Snippet: caseSource,
			Pos:     pos,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	offset, err := calculateCaseOffset(fset, content, switchBody, opts)
	if err != nil {
		return "", err
	}

	offset = normalizeInsertionOffset(fset, fileNode, content, offset)

	var newContent bytes.Buffer
	newContent.Write(content[:offset])
	if offset == 0 || content[offset-1] != '\n' {
		newContent.WriteByte('\n')
	}
	newContent.WriteString(caseTrimmed)
	newContent.WriteString("\n")
	newContent.Write(content[offset:])

	finishFormatting := telemetry.Start(ctx, telemetry.PhaseFormattingAST)
	formatted, err := format.Source(newContent.Bytes())
	finishFormatting()
	if err != nil {
		return "", &SyntaxError{
			File:    cleanPath,
			Snippet: caseSource,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}

	dir := filepath.Dir(cleanPath)
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, dir, cleanPath); err != nil {
			_ = pipeline.Format(ctx, dir, cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	} else {
		if err := pipeline.Format(ctx, dir, cleanPath); err != nil {
			return "", fmt.Errorf("format file: %w", err)
		}
	}

	diff := UnifiedDiff(funcName, "", caseTrimmed)
	return diff, nil
}

func locateSwitch(fset *token.FileSet, body *ast.BlockStmt, targetSwitchOn, switchPath string) (*ast.BlockStmt, error) {
	var canonicalTarget string
	if targetSwitchOn != "" {
		if parsed, err := parser.ParseExpr(targetSwitchOn); err == nil {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, parsed); err == nil {
				canonicalTarget = buf.String()
			}
		}
		if canonicalTarget == "" {
			canonicalTarget = targetSwitchOn
		}
	}

	matches := collectSwitchMatches(fset, body, targetSwitchOn, canonicalTarget)
	if len(matches) == 0 {
		return nil, ErrSwitchNotFound
	}
	if switchPath != "" {
		parts, err := parseSwitchPath(switchPath)
		if err != nil {
			return nil, err
		}
		current := matches
		var selected switchMatch
		for depth, part := range parts {
			index := part
			if depth > 0 {
				index-- // The parent switch occupies index zero at each nested level.
			}
			if index < 0 || index >= len(current) {
				return nil, fmt.Errorf("switch path %q does not exist; matching paths: %s", switchPath, formatSwitchPaths(matches))
			}
			selected = current[index]
			current = selected.children
		}
		return selected.body, nil
	}
	if countSwitchMatches(matches) > 1 {
		return nil, fmt.Errorf("%w for %q; candidates:\n%s\nselect one with switch_path", ErrSwitchAmbiguous, targetSwitchOn, formatSwitchCandidates(matches, fset))
	}
	return matches[0].body, nil
}

func countSwitchMatches(matches []switchMatch) int {
	count := len(matches)
	for _, match := range matches {
		count += countSwitchMatches(match.children)
	}
	return count
}

type switchMatch struct {
	path     string
	body     *ast.BlockStmt
	children []switchMatch
}

func collectSwitchMatches(fset *token.FileSet, body *ast.BlockStmt, targetSwitchOn, canonicalTarget string) []switchMatch {
	var matches []switchMatch
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.SwitchStmt:
			matched := false
			if targetSwitchOn == "" {
				matched = s.Tag == nil
			} else if s.Tag != nil {
				var buf bytes.Buffer
				if err := format.Node(&buf, fset, s.Tag); err == nil {
					matched = buf.String() == canonicalTarget || strings.TrimSpace(buf.String()) == canonicalTarget
				}
			}
			if matched {
				matches = append(matches, switchMatch{body: s.Body})
				return false
			}
		case *ast.TypeSwitchStmt:
			if targetSwitchOn != "" && s.Assign != nil {
				var selector ast.Expr
				if assign, ok := s.Assign.(*ast.AssignStmt); ok && len(assign.Rhs) == 1 {
					if assertion, ok := assign.Rhs[0].(*ast.TypeAssertExpr); ok {
						selector = assertion.X
					}
				}
				var buf bytes.Buffer
				if selector != nil && format.Node(&buf, fset, selector) == nil && buf.String() == canonicalTarget {
					matches = append(matches, switchMatch{body: s.Body})
					return false
				}
			}
		}
		return true
	})
	for i := range matches {
		matches[i].children = collectSwitchMatches(fset, matches[i].body, targetSwitchOn, canonicalTarget)
	}
	assignSwitchPaths(matches, "")
	return matches
}

func assignSwitchPaths(matches []switchMatch, prefix string) {
	for i := range matches {
		if prefix == "" {
			matches[i].path = fmt.Sprint(i)
		} else {
			matches[i].path = prefix + fmt.Sprint(i+1)
		}
		assignSwitchPaths(matches[i].children, matches[i].path+".")
	}
}

func parseSwitchPath(path string) ([]int, error) {
	var parts []int
	for part := range strings.SplitSeq(path, ".") {
		var index int
		if part == "" {
			return nil, fmt.Errorf("invalid switch path %q: expected dot-separated non-negative indexes", path)
		}
		if _, err := fmt.Sscanf(part, "%d", &index); err != nil || index < 0 || fmt.Sprint(index) != part {
			return nil, fmt.Errorf("invalid switch path %q: expected dot-separated non-negative indexes", path)
		}
		parts = append(parts, index)
	}
	return parts, nil
}

func formatSwitchPaths(matches []switchMatch) string {
	var paths []string
	var visit func([]switchMatch)
	visit = func(current []switchMatch) {
		for _, match := range current {
			paths = append(paths, match.path)
			visit(match.children)
		}
	}
	visit(matches)
	return strings.Join(paths, ", ")
}

func formatSwitchCandidates(matches []switchMatch, fset *token.FileSet) string {
	var lines []string
	var visit func([]switchMatch, int)
	visit = func(current []switchMatch, depth int) {
		for _, match := range current {
			var labels []string
			for _, stmt := range match.body.List {
				clause, ok := stmt.(*ast.CaseClause)
				if !ok {
					continue
				}
				if clause.List == nil {
					labels = append(labels, "default")
					continue
				}
				for _, expr := range clause.List {
					var buf bytes.Buffer
					if format.Node(&buf, fset, expr) == nil {
						labels = append(labels, buf.String())
					}
				}
			}
			lines = append(lines, fmt.Sprintf("  %s%s  cases: %s", strings.Repeat("  ", depth), match.path, strings.Join(labels, ", ")))
			visit(match.children, depth+1)
		}
	}
	visit(matches, 0)
	return strings.Join(lines, "\n")
}

func calculateCaseOffset(fset *token.FileSet, content []byte, switchBody *ast.BlockStmt, opts CaseOptions) (int, error) {
	lbraceOffset := fset.Position(switchBody.Lbrace).Offset
	rbraceOffset := fset.Position(switchBody.Rbrace).Offset

	var clauses []*ast.CaseClause
	for _, stmt := range switchBody.List {
		if cc, ok := stmt.(*ast.CaseClause); ok {
			clauses = append(clauses, cc)
		}
	}

	placement := opts.Placement
	if placement == "" {
		placement = CasePlacementBeforeDefault
	}

	switch placement {
	case CasePlacementFirst:
		if len(clauses) > 0 {
			return fset.Position(clauses[0].Pos()).Offset, nil
		}
		return lbraceOffset + 1, nil

	case CasePlacementLast:
		return rbraceOffset, nil

	case CasePlacementBeforeDefault:
		for _, cc := range clauses {
			if cc.List == nil {
				return fset.Position(cc.Pos()).Offset, nil
			}
		}
		return rbraceOffset, nil

	case CasePlacementBefore:
		cc, err := findAnchorCase(fset, content, clauses, opts.AnchorCase)
		if err != nil {
			return 0, err
		}
		return fset.Position(cc.Pos()).Offset, nil

	case CasePlacementAfter:
		cc, err := findAnchorCase(fset, content, clauses, opts.AnchorCase)
		if err != nil {
			return 0, err
		}
		return fset.Position(cc.End()).Offset, nil

	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedPlacement, placement)
	}
}

func findAnchorCase(fset *token.FileSet, content []byte, clauses []*ast.CaseClause, rawAnchor string) (*ast.CaseClause, error) {
	cleanAnchor := strings.TrimSpace(rawAnchor)
	cleanAnchor = strings.TrimPrefix(cleanAnchor, "case ")
	cleanAnchor = strings.TrimSuffix(cleanAnchor, ":")
	cleanAnchor = strings.TrimSpace(cleanAnchor)

	if cleanAnchor == "" {
		return nil, fmt.Errorf("%w: empty anchor case", ErrAnchorNotFound)
	}

	var canonicalAnchor string
	if expr, err := parser.ParseExpr(cleanAnchor); err == nil {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, expr); err == nil {
			canonicalAnchor = buf.String()
		}
	}
	if canonicalAnchor == "" {
		canonicalAnchor = cleanAnchor
	}

	for _, cc := range clauses {
		if cleanAnchor == "default" && cc.List == nil {
			return cc, nil
		}
		for _, expr := range cc.List {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, expr); err == nil {
				if buf.String() == canonicalAnchor || buf.String() == cleanAnchor {
					return cc, nil
				}
			}
			rawExpr := strings.TrimSpace(string(content[fset.Position(expr.Pos()).Offset:fset.Position(expr.End()).Offset]))
			if rawExpr == cleanAnchor || rawExpr == canonicalAnchor {
				return cc, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrAnchorNotFound, rawAnchor)
}
