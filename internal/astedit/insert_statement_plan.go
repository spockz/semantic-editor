// This file plans one direct call or assignment insertion without changing files or claiming control-flow equivalence.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type (
	// StatementPlacement selects the side of a direct anchor statement.
	// StatementPlacement selects the side of a direct anchor statement.
	StatementPlacement string

	// InsertStatementIntent supplies one direct statement and its exact owner anchor.
	// InsertStatementIntent supplies one direct statement and its exact owner anchor.
	InsertStatementIntent struct {
		Function         string
		Discriminator    string
		ConstructPath    OptionalString
		ExpectedRevision OptionalString
		Source           string
		Placement        StatementPlacement
	}
)

// Statement positions are relative to one live direct statement.
// Statement positions are relative to one live direct statement.
const (
	StatementBefore StatementPlacement = "before_construct"
	StatementAfter  StatementPlacement = "after_construct"
)

// PlanInsertStatement plans a direct call or assignment insertion without publication.
// PlanInsertStatement plans a direct call or assignment insertion without publication.
func PlanInsertStatement(source []byte, intent InsertStatementIntent) (SourcePlan, error) {
	if intent.Function == "" || intent.Discriminator == "" || intent.Source == "" {
		return SourcePlan{}, fmt.Errorf("function, discriminator, and source are required")
	}
	if intent.Placement != StatementBefore && intent.Placement != StatementAfter {
		return SourcePlan{}, fmt.Errorf("unsupported statement placement %q", intent.Placement)
	}
	if err := validateRevisionBoundSelector(source, intent.ConstructPath, intent.ExpectedRevision); err != nil {
		return SourcePlan{}, err
	}
	if strings.ContainsAny(intent.Source, "\r\n") {
		return SourcePlan{}, fmt.Errorf("inserted statement must be one single-line statement")
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	if intent.ConstructPath.Present {
		return SourcePlan{}, fmt.Errorf("construct_path has no compatible live producer for direct call or assignment statements")
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
	var unsafeControlFlow bool
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.BranchStmt, *ast.LabeledStmt, *ast.ReturnStmt, *ast.IfStmt,
			*ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt,
			*ast.SelectStmt, *ast.GoStmt, *ast.DeferStmt:
			unsafeControlFlow = true
		}
		return true
	})
	if unsafeControlFlow {
		return SourcePlan{}, fmt.Errorf("function %q contains control-flow or label constructs outside this planner's supported scope", intent.Function)
	}
	newStmt, err := parseSingleDirectStatement(intent.Source)
	if err != nil {
		return SourcePlan{}, fmt.Errorf("validate inserted statement: %w", err)
	}
	if !supportedDirectStatement(newStmt) {
		return SourcePlan{}, fmt.Errorf("inserted statement must be one direct call or simple assignment")
	}
	matches := make([]ast.Stmt, 0, 1)
	for _, stmt := range owner.Body.List {
		start := fset.Position(stmt.Pos()).Offset
		end := fset.Position(stmt.End()).Offset
		if start >= 0 && end <= len(source) && string(source[start:end]) == intent.Discriminator && supportedDirectStatement(stmt) {
			matches = append(matches, stmt)
		}
	}
	if len(matches) == 0 {
		return SourcePlan{}, fmt.Errorf("direct call or assignment discriminator not found in %q", intent.Function)
	}
	if len(matches) != 1 {
		return SourcePlan{}, fmt.Errorf("discriminator matches %d direct statements; no compatible construct_path producer exists", len(matches))
	}
	anchor := matches[0]
	if err := requireDedicatedStatementLine(source, fset, owner.Body, anchor); err != nil {
		return SourcePlan{}, err
	}
	var offset int
	var indent string
	if intent.Placement == StatementBefore {
		offset, indent = fieldLineStart(source, fset.Position(anchor.Pos()).Offset)
	} else {
		offset, indent = fieldLineEnd(source, fset.Position(anchor.End()).Offset)
	}
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	updated := spliceSource(source, offset, offset, []byte(indent+strings.TrimSpace(intent.Source)+newline))
	return newSourcePlan(source, updated, PlanSelection{
		Kind: "statement", Owner: intent.Function, Target: intent.Discriminator,
		Start: fset.Position(anchor.Pos()).Offset, End: fset.Position(anchor.End()).Offset,
	}, ProofCompleteSourceSet, ProofBlockBinding, ProofControlFlowSet), nil
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

func parseSingleDirectStatement(source string) (ast.Stmt, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "statement.go", []byte("package p\nfunc _() {\n"+source+"\n}"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Body.List) != 1 {
		return nil, fmt.Errorf("expected exactly one statement")
	}
	return fn.Body.List[0], nil
}

func supportedDirectStatement(stmt ast.Stmt) bool {
	switch value := stmt.(type) {
	case *ast.ExprStmt:
		_, ok := value.X.(*ast.CallExpr)
		return ok
	case *ast.AssignStmt:
		return value.Tok == token.ASSIGN && len(value.Lhs) == 1 && len(value.Rhs) == 1
	default:
		return false
	}
}
