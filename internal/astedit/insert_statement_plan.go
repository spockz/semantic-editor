// This file plans one direct call or assignment insertion without changing files or claiming control-flow equivalence.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
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
	if err := validateRevisionBoundSelector(intent.ConstructPath, intent.ExpectedRevision); err != nil {
		return SourcePlan{}, err
	}
	if strings.ContainsAny(intent.Source, "\r\n") {
		return SourcePlan{}, fmt.Errorf("inserted statement must be one single-line statement")
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	functionName := intent.Function
	if dot := strings.LastIndex(functionName, "."); dot >= 0 {
		functionName = functionName[dot+1:]
	}
	var owner *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != functionName {
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
	var anchor ast.Stmt
	if !intent.ConstructPath.Present {
		if len(matches) != 1 {
			return SourcePlan{}, fmt.Errorf("discriminator matches %d direct statements; construct_path is required", len(matches))
		}
		anchor = matches[0]
	} else {
		index, err := strconv.Atoi(intent.ConstructPath.Value)
		if err != nil || index < 0 || index >= len(owner.Body.List) {
			return SourcePlan{}, fmt.Errorf("construct_path %q is not a valid zero-based direct statement index", intent.ConstructPath.Value)
		}
		candidate := owner.Body.List[index]
		if !supportedDirectStatement(candidate) {
			return SourcePlan{}, fmt.Errorf("construct_path %q does not select a direct call or simple assignment", intent.ConstructPath.Value)
		}
		start := fset.Position(candidate.Pos()).Offset
		end := fset.Position(candidate.End()).Offset
		if string(source[start:end]) != intent.Discriminator {
			return SourcePlan{}, fmt.Errorf("construct_path %q does not match discriminator", intent.ConstructPath.Value)
		}
		anchor = candidate
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
