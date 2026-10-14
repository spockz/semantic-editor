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
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
	"github.com/spockz/semantic-editor/internal/telemetry"
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
	if sourcebuf.FromContext(ctx) == nil {
		var result string
		err := runBufferedFile(ctx, filePath, func(editCtx context.Context) error {
			var err error
			result, err = InsertCase(editCtx, filePath, funcName, switchOn, caseSource, opts)
			return err
		})
		return result, err
	}
	res, err := InsertConstruct(ctx, filePath, caseSource, InsertConstructOptions{
		Kind:                backend.ConstructCase,
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
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
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

	caseTrimmed, err := validateCaseSource(caseSource)
	if err != nil {
		return "", err
	}
	formatted, err := planCaseInsertion(ctx, cleanPath, caseSource, caseTrimmed, content, fset, fileNode, switchBody, opts)
	if err != nil {
		return "", err
	}
	if err := publishCaseInsertion(ctx, cleanPath, formatted, opts.AutoOrganizeImports); err != nil {
		return "", err
	}

	diff := UnifiedDiff(funcName, "", caseTrimmed)
	return diff, nil
}

func validateCaseSource(source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	synthetic := "package dummy\n\nfunc _() {\nswitch {\n" + trimmed + "\n}\n}\n"
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "snippet", synthetic, parser.ParseComments); err != nil {
		position := extractSyntaxPosition(fset, err)
		if position.Line > 4 {
			position.Line -= 4
		}
		return "", &SyntaxError{File: "snippet", Snippet: source, Pos: position, Cause: err, Err: ErrSyntax}
	}
	return trimmed, nil
}

func planCaseInsertion(ctx context.Context, path, source, caseText string, content []byte, fset *token.FileSet, file *ast.File, switchBody *ast.BlockStmt, opts CaseOptions) ([]byte, error) {
	offset, err := calculateCaseOffset(fset, content, switchBody, opts)
	if err != nil {
		return nil, err
	}
	offset = normalizeInsertionOffset(fset, file, content, offset)
	var updated bytes.Buffer
	updated.Write(content[:offset])
	if offset == 0 || content[offset-1] != '\n' {
		updated.WriteByte('\n')
	}
	updated.WriteString(caseText)
	updated.WriteString("\n")
	updated.Write(content[offset:])
	finishFormatting := telemetry.Start(ctx, telemetry.PhaseFormattingAST)
	formatted, err := format.Source(updated.Bytes())
	finishFormatting()
	if err != nil {
		return nil, &SyntaxError{File: path, Snippet: source, Cause: err, Err: ErrSyntax}
	}
	return formatted, nil
}

func publishCaseInsertion(ctx context.Context, path string, content []byte, organizeImports bool) error {
	return publishInsertedDeclaration(ctx, path, content, organizeImports)
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

	matches, err := collectSwitchMatches(fset, body, targetSwitchOn, canonicalTarget)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("%w for %q; candidates:\n%s\nselect one with construct_path", ErrSwitchAmbiguous, targetSwitchOn, formatSwitchCandidates(matches, fset))
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

func collectSwitchMatches(fset *token.FileSet, body *ast.BlockStmt, targetSwitchOn, canonicalTarget string) ([]switchMatch, error) {
	var matches []switchMatch
	var visitErr error
	ast.Inspect(body, func(n ast.Node) bool {
		result, err := matchingSwitchBody(fset, n, targetSwitchOn, canonicalTarget)
		if err != nil {
			visitErr = err
			return false
		}
		if result.matched {
			matches = append(matches, switchMatch{body: result.body})
			return false
		}
		return true
	})
	if visitErr != nil {
		return nil, visitErr
	}
	for i := range matches {
		children, err := collectSwitchMatches(fset, matches[i].body, targetSwitchOn, canonicalTarget)
		if err != nil {
			return nil, err
		}
		matches[i].children = children
	}
	assignSwitchPaths(matches, "")
	return matches, nil
}

type switchMatchResult struct {
	body    *ast.BlockStmt
	matched bool
}

func matchingSwitchBody(fset *token.FileSet, node ast.Node, targetSwitchOn, canonicalTarget string) (switchMatchResult, error) {
	switch statement := node.(type) {
	case *ast.SwitchStmt:
		return matchingExpressionSwitchBody(fset, statement, targetSwitchOn, canonicalTarget)
	case *ast.TypeSwitchStmt:
		return matchingTypeSwitchBody(fset, statement, targetSwitchOn, canonicalTarget)
	default:
		return switchMatchResult{}, nil
	}
}

func matchingExpressionSwitchBody(fset *token.FileSet, statement *ast.SwitchStmt, targetSwitchOn, canonicalTarget string) (switchMatchResult, error) {
	if targetSwitchOn == "" {
		if statement.Tag == nil {
			return switchMatchResult{body: statement.Body, matched: true}, nil
		}
		return switchMatchResult{}, nil
	}
	if statement.Tag == nil {
		return switchMatchResult{}, nil
	}
	var formatted bytes.Buffer
	if err := format.Node(&formatted, fset, statement.Tag); err != nil {
		return switchMatchResult{}, fmt.Errorf("format expression switch selector: %w", err)
	}
	text := formatted.String()
	if text == canonicalTarget || strings.TrimSpace(text) == canonicalTarget {
		return switchMatchResult{body: statement.Body, matched: true}, nil
	}
	return switchMatchResult{}, nil
}

func matchingTypeSwitchBody(fset *token.FileSet, statement *ast.TypeSwitchStmt, targetSwitchOn, canonicalTarget string) (switchMatchResult, error) {
	if targetSwitchOn == "" || statement.Assign == nil {
		return switchMatchResult{}, nil
	}
	selector := typeSwitchSelector(statement.Assign)
	if selector == nil {
		return switchMatchResult{}, nil
	}
	var formatted bytes.Buffer
	if err := format.Node(&formatted, fset, selector); err != nil {
		return switchMatchResult{}, fmt.Errorf("format type switch selector: %w", err)
	}
	if formatted.String() != canonicalTarget {
		return switchMatchResult{}, nil
	}
	return switchMatchResult{body: statement.Body, matched: true}, nil
}

func typeSwitchSelector(assign ast.Stmt) ast.Expr {
	assignment, ok := assign.(*ast.AssignStmt)
	if !ok || len(assignment.Rhs) != 1 {
		return nil
	}
	assertion, ok := assignment.Rhs[0].(*ast.TypeAssertExpr)
	if !ok {
		return nil
	}
	return assertion.X
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
