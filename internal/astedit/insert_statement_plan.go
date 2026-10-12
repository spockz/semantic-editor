// Package astedit plans direct statement insertions without claiming control-flow equivalence.
package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"semedit/internal/symbol"
	"slices"
	"strings"
)

type (
	// StatementPlacement selects the side of a direct anchor statement.
	StatementPlacement string

	// InsertStatementIntent supplies one direct statement and its exact owner anchor.
	InsertStatementIntent struct {
		Function         string
		Anchor           string
		Discriminator    string
		ConstructPath    OptionalString
		ExpectedRevision OptionalString
		Source           string
		Placement        StatementPlacement
	}
)

// Statement positions are relative to one live direct statement.
const (
	StatementBefore StatementPlacement = "before_construct"
	StatementAfter  StatementPlacement = "after_construct"
)

// PlanInsertStatement plans a direct call or assignment insertion without publication.
func PlanInsertStatement(source []byte, intent InsertStatementIntent) (SourcePlan, error) {
	anchorMode, err := validateInsertStatementIntent(source, intent)
	if err != nil {
		return SourcePlan{}, err
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	_, err = parseSingleDirectStatement(intent.Source)
	if err != nil {
		return SourcePlan{}, fmt.Errorf("validate inserted statement: %w", err)
	}
	owner, err := findInsertStatementOwner(file, intent.Function)
	if err != nil {
		return SourcePlan{}, err
	}
	selection, err := resolveInsertStatementAnchor(source, fset, file, owner, intent, anchorMode)
	if err != nil {
		return SourcePlan{}, err
	}
	return buildInsertStatementPlan(source, fset, owner, intent, selection)
}

func validateInsertStatementIntent(source []byte, intent InsertStatementIntent) (string, error) {
	if intent.Function == "" || intent.Source == "" {
		return "", fmt.Errorf("function and source are required")
	}
	anchorMode := resolveInsertStatementAnchorMode(intent.Anchor, intent.Placement)
	if !slices.Contains([]string{"start", "end", "before_return", "before", "after"}, anchorMode) {
		return "", fmt.Errorf("unsupported statement anchor %q", anchorMode)
	}
	if (anchorMode == "before" || anchorMode == "after") && intent.Discriminator == "" {
		return "", fmt.Errorf("discriminator is required for %s insertion", anchorMode)
	}
	if err := validateRevisionBoundSelector(source, intent.ConstructPath, intent.ExpectedRevision); err != nil {
		return "", err
	}
	return anchorMode, nil
}

func resolveInsertStatementAnchorMode(anchor string, placement StatementPlacement) string {
	if anchor != "" {
		return anchor
	}
	switch placement {
	case StatementBefore:
		return "before"
	case StatementAfter:
		return "after"
	case "":
		return "end"
	default:
		return string(placement)
	}
}

func findInsertStatementOwner(file *ast.File, selector string) (*ast.FuncDecl, error) {
	receiver, functionName, err := symbol.ParseIdentifier(selector)
	if err != nil {
		return nil, fmt.Errorf("function selector %q is invalid: %w", selector, err)
	}
	if !token.IsIdentifier(functionName) || token.Lookup(functionName).IsKeyword() {
		return nil, fmt.Errorf("function selector %q is invalid", selector)
	}
	var owner *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != functionName || !matchesFunctionOwner(function, receiver) {
			continue
		}
		if owner != nil {
			return nil, fmt.Errorf("function selector %q is ambiguous in this file", selector)
		}
		owner = function
	}
	if owner == nil || owner.Body == nil {
		return nil, fmt.Errorf("function %q not found", selector)
	}
	return owner, nil
}

type insertStatementAnchor struct {
	statement ast.Stmt
	mode      string
}

func resolveInsertStatementAnchor(source []byte, fset *token.FileSet, file *ast.File, owner *ast.FuncDecl, intent InsertStatementIntent, mode string) (insertStatementAnchor, error) {
	if mode == "before" || mode == "after" {
		return selectDirectStatementAnchor(source, fset, file, owner, intent, mode)
	}
	return selectConvenienceStatementAnchor(owner, intent.Function, mode)
}

func selectConvenienceStatementAnchor(owner *ast.FuncDecl, function, mode string) (insertStatementAnchor, error) {
	selection := insertStatementAnchor{mode: mode}
	switch mode {
	case "start":
		if len(owner.Body.List) > 0 {
			selection.statement = owner.Body.List[0]
		}
	case "end":
		if len(owner.Body.List) > 0 {
			selection.statement = owner.Body.List[len(owner.Body.List)-1]
			if isInsertTerminal(selection.statement) {
				selection.mode = "before"
			}
		}
	case "before_return":
		for _, statement := range slices.Backward(owner.Body.List) {
			if _, ok := statement.(*ast.ReturnStmt); ok {
				selection.statement, selection.mode = statement, "before"
				break
			}
		}
		if selection.statement == nil {
			return insertStatementAnchor{}, fmt.Errorf("function %q has no direct return statement", function)
		}
	}
	return selection, nil
}

func selectDirectStatementAnchor(source []byte, fset *token.FileSet, file *ast.File, owner *ast.FuncDecl, intent InsertStatementIntent, mode string) (insertStatementAnchor, error) {
	candidates, _ := collectStatementExpressionCandidates(fset, source, file, owner)
	matches := matchingDirectStatementCandidates(candidates, intent)
	if len(matches) == 0 {
		return insertStatementAnchor{}, missingDirectStatementAnchorError(intent)
	}
	if len(matches) > 1 {
		return insertStatementAnchor{}, fmt.Errorf("ambiguous statement anchor %q; select construct_path with expected_revision %q from:\n%s", intent.Discriminator, goReadRevision(source), formatSpanCandidates(fset, matches))
	}
	selected := matches[0]
	if selected.class != "statement" {
		return insertStatementAnchor{}, fmt.Errorf("construct_path %s selects an expression, not a direct statement anchor", selected.path)
	}
	statement, _ := selected.node.(ast.Stmt)
	return insertStatementAnchor{statement: statement, mode: mode}, nil
}

func matchingDirectStatementCandidates(candidates []constructCandidate, intent InsertStatementIntent) []constructCandidate {
	matches := make([]constructCandidate, 0, 1)
	for _, candidate := range candidates {
		if candidate.class == "statement" && candidate.key == normalizeConstructSelector(intent.Discriminator) {
			matches = append(matches, candidate)
		}
	}
	if intent.ConstructPath.Present {
		matches = slices.DeleteFunc(matches, func(candidate constructCandidate) bool { return candidate.path != intent.ConstructPath.Value })
	}
	return matches
}

func missingDirectStatementAnchorError(intent InsertStatementIntent) error {
	if intent.ConstructPath.Present {
		return fmt.Errorf("construct_path %q does not select a direct statement anchor %q", intent.ConstructPath.Value, intent.Discriminator)
	}
	return fmt.Errorf("statement anchor %q not found in %q", intent.Discriminator, intent.Function)
}

func buildInsertStatementPlan(source []byte, fset *token.FileSet, owner *ast.FuncDecl, intent InsertStatementIntent, selection insertStatementAnchor) (SourcePlan, error) {
	if selection.statement == nil && (selection.mode == "start" || selection.mode == "end") {
		offset, indent := emptyBlockInsertionOffset(source, fset, owner.Body)
		newline := sourceNewline(source)
		updated := spliceSource(source, offset, offset, []byte(indent+strings.TrimSpace(intent.Source)+newline))
		return newSourcePlan(source, updated, PlanSelection{Kind: "statement", Owner: intent.Function}, ProofCompleteSourceSet), nil
	}
	block := statementContainingBlock(owner.Body, selection.statement)
	if block == nil {
		return SourcePlan{}, fmt.Errorf("selected statement does not belong to a block in %q", intent.Function)
	}
	if err := requireDedicatedStatementLine(source, fset, block, selection.statement); err != nil {
		return SourcePlan{}, err
	}
	var offset int
	var indent string
	if selection.mode == "start" || selection.mode == "before" {
		offset, indent = fieldLineStart(source, fset.Position(selection.statement.Pos()).Offset)
	} else {
		if isInsertTerminal(selection.statement) {
			return SourcePlan{}, fmt.Errorf("cannot insert after a terminal return, panic, or branch statement")
		}
		offset, indent = fieldLineEnd(source, fset.Position(selection.statement.End()).Offset)
	}
	newline := sourceNewline(source)
	updated := spliceSource(source, offset, offset, []byte(indent+strings.TrimSpace(intent.Source)+newline))
	return newSourcePlan(source, updated, PlanSelection{
		Kind: "statement", Owner: intent.Function, Target: intent.Discriminator,
		Start: fset.Position(selection.statement.Pos()).Offset, End: fset.Position(selection.statement.End()).Offset,
	}, ProofCompleteSourceSet, ProofBlockBinding, ProofControlFlowSet), nil
}

func isInsertTerminal(stmt ast.Stmt) bool {
	switch value := stmt.(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	case *ast.ExprStmt:
		call, ok := value.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		ident, ok := call.Fun.(*ast.Ident)
		return ok && ident.Name == "panic"
	default:
		return false
	}
}

func sourceNewline(source []byte) string {
	if bytes.Contains(source, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func emptyBlockInsertionOffset(source []byte, fset *token.FileSet, body *ast.BlockStmt) (int, string) {
	open := fset.Position(body.Lbrace).Offset
	_, ownerIndent := fieldLineStart(source, open)
	return open + 1, ownerIndent + "\t"
}

func matchesFunctionOwner(fn *ast.FuncDecl, selector string) bool {
	if selector == "" {
		return fn.Recv == nil
	}
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	return extractReceiverTypeName(fn.Recv) == selector
}

func requireDedicatedStatementLine(source []byte, fset *token.FileSet, block *ast.BlockStmt, anchor ast.Stmt) error {
	start := fset.Position(anchor.Pos()).Offset
	end := fset.Position(anchor.End()).Offset
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := bytes.IndexByte(source[end:], '\n')
	if lineEnd < 0 {
		lineEnd = len(source)
	} else {
		lineEnd += end
	}
	if strings.TrimSpace(string(source[lineStart:start])) != "" {
		return fmt.Errorf("selected statement shares a source line with another token")
	}
	suffix := strings.TrimSpace(string(source[end:lineEnd]))
	if suffix != "" && !strings.HasPrefix(suffix, "//") {
		return fmt.Errorf("selected statement shares a source line with another token")
	}
	for _, stmt := range block.List {
		if stmt == anchor {
			continue
		}
		otherStart := fset.Position(stmt.Pos()).Offset
		otherEnd := fset.Position(stmt.End()).Offset
		if otherStart >= lineStart && otherStart <= lineEnd || otherEnd >= lineStart && otherEnd <= lineEnd {
			return fmt.Errorf("selected statement shares a source line with another statement")
		}
	}
	return nil
}

func parseSingleDirectStatement(source string) (int, error) {
	const wrapperPrefix = "package p\nfunc _() {\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "statement.go", []byte(wrapperPrefix+source+"\n}"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return 0, err
	}
	if len(file.Decls) != 1 {
		return 0, fmt.Errorf("expected exactly one statement and no declarations outside the wrapper function")
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || fn.Name.Name != "_" || fn.Recv != nil || len(fn.Body.List) != 1 {
		return 0, fmt.Errorf("expected exactly one statement")
	}
	endOffset := fset.Position(fn.Body.List[0].End()).Offset - len(wrapperPrefix)
	return endOffset, nil
}

func statementContainingBlock(root *ast.BlockStmt, target ast.Stmt) *ast.BlockStmt {
	var selected *ast.BlockStmt
	ast.Inspect(root, func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		if slices.Contains(block.List, target) {
			selected = block
			return false
		}
		return selected == nil
	})
	return selected
}
