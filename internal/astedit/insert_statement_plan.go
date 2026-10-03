// This file plans one direct call or assignment insertion without changing files or claiming control-flow equivalence.
package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
	if intent.Function == "" || intent.Source == "" {
		return SourcePlan{}, fmt.Errorf("function and source are required")
	}
	anchorMode := intent.Anchor
	if anchorMode == "" {
		switch intent.Placement {
		case StatementBefore:
			anchorMode = "before"
		case StatementAfter:
			anchorMode = "after"
		case "":
			anchorMode = "end"
		default:
			anchorMode = string(intent.Placement)
		}
	}
	if !slices.Contains([]string{"start", "end", "before_return", "before", "after"}, anchorMode) {
		return SourcePlan{}, fmt.Errorf("unsupported statement anchor %q", anchorMode)
	}
	if (anchorMode == "before" || anchorMode == "after") && intent.Discriminator == "" {
		return SourcePlan{}, fmt.Errorf("discriminator is required for %s insertion", anchorMode)
	}
	if err := validateRevisionBoundSelector(source, intent.ConstructPath, intent.ExpectedRevision); err != nil {
		return SourcePlan{}, err
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	functionName := intent.Function
	ownerSelector := ""
	if dot := strings.LastIndex(functionName, "."); dot >= 0 {
		ownerSelector, functionName = functionName[:dot], functionName[dot+1:]
	}
	if !token.IsIdentifier(functionName) || token.Lookup(functionName).IsKeyword() {
		return SourcePlan{}, fmt.Errorf("function selector %q is invalid", intent.Function)
	}
	var owner *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != functionName {
			continue
		}
		if !matchesFunctionOwner(fn, ownerSelector) {
			continue
		}
		if owner != nil {
			return SourcePlan{}, fmt.Errorf("function selector %q is ambiguous in this file", intent.Function)
		}
		owner = fn
	}
	if owner == nil || owner.Body == nil {
		return SourcePlan{}, fmt.Errorf("function %q not found", intent.Function)
	}
	_, err = parseSingleDirectStatement(intent.Source)
	if err != nil {
		return SourcePlan{}, fmt.Errorf("validate inserted statement: %w", err)
	}
	var anchor ast.Stmt
	switch anchorMode {
	case "start":
		if len(owner.Body.List) > 0 {
			anchor = owner.Body.List[0]
		}
	case "end":
		if len(owner.Body.List) > 0 {
			anchor = owner.Body.List[len(owner.Body.List)-1]
			if isInsertTerminal(anchor) {
				anchorMode = "before"
			}
		}
	case "before_return":
		for _, stmt := range slices.Backward(owner.Body.List) {
			if _, ok := stmt.(*ast.ReturnStmt); ok {
				anchor, anchorMode = stmt, "before"
				break
			}
		}
		if anchor == nil {
			return SourcePlan{}, fmt.Errorf("function %q has no direct return statement", intent.Function)
		}
	case "before", "after":
		candidates := collectStatementExpressionCandidates(fset, source, file, owner)
		matches := make([]constructCandidate, 0, 1)
		for _, candidate := range candidates {
			if !isDirectOwnerStatementCandidate(candidate) || candidate.key != normalizeConstructSelector(intent.Discriminator) {
				continue
			}
			matches = append(matches, candidate)
		}
		if intent.ConstructPath.Present {
			matches = slices.DeleteFunc(matches, func(candidate constructCandidate) bool { return candidate.path != intent.ConstructPath.Value })
		}
		if len(matches) == 0 {
			if intent.ConstructPath.Present {
				return SourcePlan{}, fmt.Errorf("construct_path %q does not select a direct statement anchor %q", intent.ConstructPath.Value, intent.Discriminator)
			}
			return SourcePlan{}, fmt.Errorf("statement anchor %q not found in %q", intent.Discriminator, intent.Function)
		}
		if len(matches) > 1 {
			return SourcePlan{}, fmt.Errorf("ambiguous statement anchor %q; select construct_path from:\n%s", intent.Discriminator, formatSpanCandidates(fset, matches))
		}
		selected := matches[0]
		if selected.class != "statement" {
			return SourcePlan{}, fmt.Errorf("construct_path %s selects an expression, not a direct statement anchor", selected.path)
		}
		anchor, _ = selected.node.(ast.Stmt)
		for _, stmt := range owner.Body.List {
			if stmt == anchor {
				break
			}
		}
	}
	if anchor == nil && (anchorMode == "start" || anchorMode == "end") {
		offset, indent := emptyBlockInsertionOffset(source, fset, owner.Body)
		newline := sourceNewline(source)
		updated := spliceSource(source, offset, offset, []byte(indent+strings.TrimSpace(intent.Source)+newline))
		return newSourcePlan(source, updated, PlanSelection{Kind: "statement", Owner: intent.Function}, ProofCompleteSourceSet), nil
	}
	if err := requireDedicatedStatementLine(source, fset, owner.Body, anchor); err != nil {
		return SourcePlan{}, err
	}
	var offset int
	var indent string
	if anchorMode == "start" || anchorMode == "before" {
		offset, indent = fieldLineStart(source, fset.Position(anchor.Pos()).Offset)
	} else {
		if isInsertTerminal(anchor) {
			return SourcePlan{}, fmt.Errorf("cannot insert after a terminal return, panic, or branch statement")
		}
		offset, indent = fieldLineEnd(source, fset.Position(anchor.End()).Offset)
	}
	newline := sourceNewline(source)
	updated := spliceSource(source, offset, offset, []byte(indent+strings.TrimSpace(intent.Source)+newline))
	return newSourcePlan(source, updated, PlanSelection{
		Kind: "statement", Owner: intent.Function, Target: intent.Discriminator,
		Start: fset.Position(anchor.Pos()).Offset, End: fset.Position(anchor.End()).Offset,
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
	switch recv := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return selector == recv.Name
	case *ast.StarExpr:
		ident, ok := recv.X.(*ast.Ident)
		return ok && selector == "(*"+ident.Name+")"
	default:
		return false
	}
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
