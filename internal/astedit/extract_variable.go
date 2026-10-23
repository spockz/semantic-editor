// Package astedit extracts selected Go expressions into locals while preserving typed evaluation boundaries.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	types "go/types"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"

	"golang.org/x/tools/go/packages"
)

// ExtractVariableOptions selects an exact direct call argument, assignment RHS, return expression, or complete if condition in the owner's body.
type ExtractVariableOptions struct {
	Discriminator    string
	ConstructPath    string
	EndConstructPath string
	ExpectedRevision string
}

type variableCandidate struct {
	path                    string
	endPath                 string
	stmt                    *ast.ExprStmt
	assignment              *ast.AssignStmt
	declaration             *ast.DeclStmt
	returnStmt              *ast.ReturnStmt
	ifStatement             *ast.IfStmt
	expr                    ast.Expr
	refusal                 string
	matchingArgumentIndexes []int
	calls                   []variableCallSelection
	assignments             []variableAssignmentSelection
}

func selectedVariableCallSelections(candidate variableCandidate) []variableCallSelection {
	if len(candidate.calls) > 0 {
		return candidate.calls
	}
	if candidate.stmt == nil {
		return nil
	}
	return []variableCallSelection{{stmt: candidate.stmt, matchingArgumentIndexes: candidate.matchingArgumentIndexes}}
}

func selectedVariableExpressions(candidate variableCandidate) ([]ast.Expr, error) {
	calls := selectedVariableCallSelections(candidate)
	if len(calls) == 0 {
		if len(candidate.assignments) > 0 {
			expressions := make([]ast.Expr, 0, len(candidate.assignments))
			for _, selected := range candidate.assignments {
				expressions = append(expressions, selected.expression)
			}
			return expressions, nil
		}
		if candidate.expr == nil {
			return nil, fmt.Errorf("cannot extract expression: selected expression is no longer available")
		}
		return []ast.Expr{candidate.expr}, nil
	}
	var expressions []ast.Expr
	for _, selected := range calls {
		call, ok := selected.stmt.X.(*ast.CallExpr)
		if !ok {
			return nil, fmt.Errorf("cannot extract expression: selected call is no longer available")
		}
		for _, index := range selected.matchingArgumentIndexes {
			if index < 0 || index >= len(call.Args) {
				return nil, fmt.Errorf("cannot extract expression: selected call argument is no longer available")
			}
			expressions = append(expressions, call.Args[index])
		}
	}
	if len(expressions) == 0 {
		return nil, fmt.Errorf("cannot extract expression: selected call has no matching arguments")
	}
	return expressions, nil
}

type variableCallSelection struct {
	stmt                    *ast.ExprStmt
	matchingArgumentIndexes []int
}

type variableAssignmentSelection struct {
	assignment *ast.AssignStmt
	expression ast.Expr
}

type variableExtractionInput struct {
	ctx         context.Context
	cleanPath   string
	ownedBuffer *sourcebuf.Buffer
	standalone  bool
	content     []byte
	file        *ast.File
	fset        *token.FileSet
	target      *ast.FuncDecl
	candidate   variableCandidate
}

type variableTypeContext struct {
	owner *ast.FuncDecl
	file  *ast.File
	info  *types.Info
	fset  *token.FileSet
}

// ExtractVariable extracts one exact typed expression into a local at a supported statement boundary.
func ExtractVariable(ctx context.Context, filePath, owner, name string, opts ExtractVariableOptions) (string, error) {
	input, err := prepareVariableExtraction(ctx, filePath, owner, name, opts)
	if err != nil {
		return "", err
	}
	updated, err := rewriteVariableExpression(input, name, opts.Discriminator)
	if err != nil {
		return "", err
	}
	return publishVariableExtraction(input, owner, name, opts.Discriminator, updated)
}

func validateVariableExtractionRequest(name string, opts ExtractVariableOptions) error {
	if !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() || name == "_" {
		return fmt.Errorf("symbol %q is not a valid local identifier", name)
	}
	if opts.Discriminator == "" {
		return fmt.Errorf("discriminator must contain the exact source text of the expression")
	}
	if opts.EndConstructPath != "" && opts.ConstructPath == "" {
		return fmt.Errorf("end_construct_path requires construct_path")
	}
	return nil
}

func loadVariableExtractionInput(ctx context.Context, filePath, owner string, opts ExtractVariableOptions) (variableExtractionInput, error) {
	input := variableExtractionInput{cleanPath: filepath.Clean(filePath)}
	preparedCtx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, input.cleanPath)
	if err != nil {
		return variableExtractionInput{}, err
	}
	input.ctx, input.ownedBuffer, input.standalone = preparedCtx, ownedBuffer, standalone
	input.content, err = sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return variableExtractionInput{}, fmt.Errorf("read extraction target: %w", err)
	}
	if (opts.ConstructPath != "" || opts.EndConstructPath != "") && opts.ExpectedRevision == "" {
		return variableExtractionInput{}, fmt.Errorf("construct_path selection requires a non-empty expected_revision")
	}
	if opts.ExpectedRevision != "" && goReadRevision(input.content) != opts.ExpectedRevision {
		return variableExtractionInput{}, fmt.Errorf("selected source revision mismatch: expected %q, current %q", opts.ExpectedRevision, goReadRevision(input.content))
	}
	recv, ownerName, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return variableExtractionInput{}, fmt.Errorf("parse in_function selector: %w", err)
	}
	input.fset = token.NewFileSet()
	input.file, err = parser.ParseFile(input.fset, input.cleanPath, input.content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return variableExtractionInput{}, &SyntaxError{File: input.cleanPath, Pos: extractSyntaxPosition(input.fset, err), Cause: err, Err: ErrSyntax}
	}
	input.target = findExtractOwner(input.file, ownerName, recv)
	if input.target == nil {
		return variableExtractionInput{}, &symbol.SymbolError{Op: "extract_variable", File: input.cleanPath, Symbol: owner, Err: symbol.ErrNotFound}
	}
	if input.target.Body == nil {
		return variableExtractionInput{}, fmt.Errorf("cannot extract from %s: function has no body", owner)
	}
	for _, group := range input.file.Comments {
		if group.Pos() > input.target.Body.Lbrace && group.End() < input.target.Body.Rbrace {
			return variableExtractionInput{}, fmt.Errorf("cannot extract from %s: comments inside the owner function require manual comment ownership review", owner)
		}
	}
	candidates, err := findVariableCandidates(input.fset, input.content, input.file, input.target, opts.Discriminator)
	if err != nil {
		return variableExtractionInput{}, err
	}
	input.candidate, err = chooseVariableCandidate(candidates, input.target, opts.ConstructPath, opts.EndConstructPath)
	if err != nil {
		return variableExtractionInput{}, err
	}
	return input, nil
}

func validateVariableExtractionTarget(input variableExtractionInput, owner, name string, opts ExtractVariableOptions) error {
	if err := validateIfConditionCandidate(input, opts.Discriminator); err != nil {
		return err
	}
	if input.candidate.returnStmt != nil {
		if len(input.candidate.returnStmt.Results) != 1 {
			return fmt.Errorf("cannot extract expression %q: only one whole return expression is supported", opts.Discriminator)
		}
		if !hasSingleReturnResult(input.target) {
			return fmt.Errorf("cannot extract expression %q: return extraction requires exactly one named or unnamed owner result", opts.Discriminator)
		}
	}
	if input.candidate.stmt == nil && !sideEffectFree(input.candidate.expr) {
		return fmt.Errorf("cannot extract expression %q: only side-effect-free expressions without calls, receives, address-taking, or composite literals are supported", opts.Discriminator)
	}
	if variableNameVisible(input.file, input.target, name) {
		return fmt.Errorf("cannot extract: local name %q collides with a name visible in %s", name, owner)
	}
	return validateSelectedCallArguments(input, opts.Discriminator)
}

func validateIfConditionCandidate(input variableExtractionInput, discriminator string) error {
	if input.candidate.ifStatement == nil {
		return nil
	}
	if input.candidate.ifStatement.Init != nil {
		return fmt.Errorf("cannot extract expression %q: if conditions with an init clause are not supported", discriminator)
	}
	if !sideEffectFree(input.candidate.expr) {
		return fmt.Errorf("cannot extract expression %q: if condition must be side-effect-free", discriminator)
	}
	return nil
}

func validateSelectedCallArguments(input variableExtractionInput, discriminator string) error {
	for _, selected := range selectedVariableCallSelections(input.candidate) {
		call, ok := selected.stmt.X.(*ast.CallExpr)
		if !ok || len(selected.matchingArgumentIndexes) == 0 {
			return fmt.Errorf("cannot extract expression %q: selected call argument is no longer available", discriminator)
		}
		expression := call.Args[selected.matchingArgumentIndexes[0]]
		if !sideEffectFree(expression) {
			return fmt.Errorf("cannot extract expression %q: only side-effect-free expressions without calls, receives, address-taking, or composite literals are supported", discriminator)
		}
		if !hasTypedInput(expression, input.target) {
			return fmt.Errorf("cannot extract expression %q: its Go type may depend on call context or defaulting when assigned to a local", discriminator)
		}
		function, ok := resolvedStaticFunctionDeclaration(input.file, input.target, selected.stmt)
		if !ok {
			return fmt.Errorf("cannot extract expression %q: only an unshadowed same-file package function call with no unresolved import binding is supported", discriminator)
		}
		if function.Type.TypeParams != nil && len(function.Type.TypeParams.List) > 0 {
			return fmt.Errorf("cannot extract expression %q: generic callees are not supported", discriminator)
		}
		if function.Type.Params != nil && len(function.Type.Params.List) > 0 {
			if _, variadic := function.Type.Params.List[len(function.Type.Params.List)-1].Type.(*ast.Ellipsis); variadic {
				return fmt.Errorf("cannot extract expression %q: variadic callees are not supported", discriminator)
			}
		}
		if call.Ellipsis.IsValid() {
			return fmt.Errorf("cannot extract expression %q: variadic expansion is not supported", discriminator)
		}
	}
	return nil
}

func rewriteVariableExpression(input variableExtractionInput, name, discriminator string) ([]byte, error) {
	var stmtStart int
	switch {
	case input.candidate.stmt != nil:
		stmtStart = input.fset.Position(input.candidate.stmt.Pos()).Offset
	case input.candidate.assignment != nil:
		stmtStart = input.fset.Position(input.candidate.assignment.Pos()).Offset
	case input.candidate.declaration != nil:
		stmtStart = input.fset.Position(input.candidate.declaration.Pos()).Offset
	case input.candidate.returnStmt != nil:
		stmtStart = input.fset.Position(input.candidate.returnStmt.Pos()).Offset
	case input.candidate.ifStatement != nil:
		stmtStart = input.fset.Position(input.candidate.ifStatement.Pos()).Offset
	default:
		return nil, fmt.Errorf("cannot extract expression: selected statement is unsupported")
	}
	expressions, err := selectedVariableExpressions(input.candidate)
	if err != nil {
		return nil, err
	}
	updated := make([]byte, 0, len(input.content)+len(name)*len(expressions)+len(discriminator)+16)
	updated = append(updated, input.content[:stmtStart]...)
	updated = append(updated, []byte(name+" := "+discriminator+"\n")...)
	cursor := stmtStart
	for _, expr := range expressions {
		exprStart := input.fset.Position(expr.Pos()).Offset
		exprEnd := input.fset.Position(expr.End()).Offset
		if exprStart < cursor || exprEnd < exprStart || exprEnd > len(input.content) {
			return nil, fmt.Errorf("cannot extract expression: source positions are inconsistent")
		}
		updated = append(updated, input.content[cursor:exprStart]...)
		updated = append(updated, name...)
		cursor = exprEnd
	}
	if cursor < stmtStart || cursor > len(input.content) {
		return nil, fmt.Errorf("cannot extract expression: source positions are inconsistent")
	}
	updated = append(updated, input.content[cursor:]...)
	return updated, nil
}

func publishVariableExtraction(input variableExtractionInput, owner, name, discriminator string, updated []byte) (string, error) {
	formatted, err := prepareVariableExtractionOutput(input, owner, name, discriminator, updated)
	if err != nil {
		return "", err
	}
	return publishPreparedSource(input.ctx, sourcePublication{
		path: input.cleanPath, diffPath: filepath.Base(input.cleanPath), before: input.content, after: formatted,
		ownedBuffer: input.ownedBuffer, standalone: input.standalone, operation: "extract variable",
	})
}

func hasSingleReturnResult(target *ast.FuncDecl) bool {
	if target == nil || target.Type.Results == nil || len(target.Type.Results.List) != 1 {
		return false
	}
	names := target.Type.Results.List[0].Names
	return len(names) == 0 || len(names) == 1 && names[0].Name != "_"
}

func returnVariableCandidate(fset *token.FileSet, content []byte, path string, stmt *ast.ReturnStmt, owner *ast.FuncDecl, discriminator string) (variableCandidate, bool, error) {
	var matches []ast.Expr
	for _, expr := range stmt.Results {
		matchesSource, err := expressionMatchesSource(fset, content, expr, discriminator)
		if err != nil {
			return variableCandidate{}, false, err
		}
		if matchesSource {
			matches = append(matches, expr)
		}
	}
	if len(matches) == 0 {
		return variableCandidate{}, false, nil
	}
	candidate := variableCandidate{path: path, returnStmt: stmt, expr: matches[0]}
	switch {
	case len(matches) > 1:
		candidate.refusal = "multiple return values match the selected expression"
	case len(stmt.Results) != 1:
		candidate.refusal = "only one whole return expression is supported"
	case !hasSingleReturnResult(owner):
		candidate.refusal = ("return extraction requires exactly one named or unnamed owner result")
	}
	return candidate, true, nil
}

func findVariableCandidates(fset *token.FileSet, content []byte, file *ast.File, target *ast.FuncDecl, discriminator string) ([]variableCandidate, error) {
	var candidates []variableCandidate
	for index, stmt := range target.Body.List {
		expression, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		candidate, found, err := callVariableCandidate(fset, content, index, expression, discriminator)
		if err != nil {
			return nil, err
		}
		if found {
			candidates = append(candidates, candidate)
		}
	}
	statements, _ := collectStatementExpressionCandidates(fset, content, file, target)
	for _, statement := range statements {
		candidate, found, err := statementVariableCandidate(fset, content, target, discriminator, statement)
		if err != nil {
			return nil, err
		}
		if found {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

func statementVariableCandidate(fset *token.FileSet, content []byte, target *ast.FuncDecl, discriminator string, statement constructCandidate) (variableCandidate, bool, error) {
	switch node := statement.node.(type) {
	case *ast.AssignStmt:
		return assignmentVariableCandidate(fset, content, statement.path, node, discriminator)
	case *ast.DeclStmt:
		return variableDeclarationCandidate(fset, content, statement.path, node, discriminator)
	case *ast.ReturnStmt:
		return returnVariableCandidate(fset, content, statement.path, node, target, discriminator)
	case *ast.IfStmt:
		return ifConditionVariableCandidate(fset, content, statement.path, node, discriminator)
	default:
		return variableCandidate{}, false, nil
	}
}

func ifConditionVariableCandidate(fset *token.FileSet, content []byte, path string, statement *ast.IfStmt, discriminator string) (variableCandidate, bool, error) {
	if statement.Cond == nil {
		return variableCandidate{}, false, nil
	}
	matches, err := expressionMatchesSource(fset, content, statement.Cond, discriminator)
	if err != nil || !matches {
		return variableCandidate{}, false, err
	}
	candidate := variableCandidate{path: path, ifStatement: statement, expr: statement.Cond}
	if statement.Init != nil {
		candidate.refusal = "if conditions with an init clause are not supported"
	}
	return candidate, true, nil
}

func chooseVariableCandidate(candidates []variableCandidate, owner *ast.FuncDecl, path, endPath string) (variableCandidate, error) {
	if len(candidates) == 0 {
		return variableCandidate{}, fmt.Errorf("no supported call argument, assignment RHS, return expression, or complete if condition exactly matches the discriminator")
	}
	if endPath != "" {
		return chooseVariableRangeCandidate(candidates, owner, path, endPath)
	}
	if path != "" {
		for _, candidate := range candidates {
			if candidate.path == path {
				if candidate.refusal != "" {
					return variableCandidate{}, fmt.Errorf("cannot extract expression: %s", candidate.refusal)
				}
				return candidate, nil
			}
		}
		return variableCandidate{}, fmt.Errorf("construct_path %q does not select a matching expression; candidates: %s", path, variablePaths(candidates))
	}
	eligible := supportedVariableCandidates(candidates)
	switch len(eligible) {
	case 1:
		return eligible[0], nil
	case 0:
		if len(candidates) == 1 {
			return variableCandidate{}, fmt.Errorf("cannot extract expression: %s", candidates[0].refusal)
		}
		return variableCandidate{}, fmt.Errorf("no supported call argument, assignment RHS, return expression, or complete if condition exactly matches the discriminator")
	default:
		return variableCandidate{}, fmt.Errorf("expression selection is ambiguous; set construct_path to one of: %s", variablePaths(eligible))
	}
}

func chooseVariableRangeCandidate(candidates []variableCandidate, owner *ast.FuncDecl, path, endPath string) (variableCandidate, error) {
	if path == "" {
		return variableCandidate{}, fmt.Errorf("end_construct_path requires construct_path")
	}
	byPath := make(map[string]variableCandidate, len(candidates))
	for _, candidate := range candidates {
		byPath[candidate.path] = candidate
	}
	if startIndex, endIndex, ok := directCallRangeIndexes(path, endPath); ok {
		if candidate, found := byPath[path]; found && candidate.stmt != nil {
			if endIndex-startIndex >= len(candidates) {
				return variableCandidate{}, fmt.Errorf("construct_path range includes a statement that is not a matching direct call")
			}
			return chooseDirectCallRange(byPath, startIndex, endIndex, endPath)
		}
	}
	return chooseAssignmentRange(byPath, owner, path, endPath)
}

func directCallRangeIndexes(path, endPath string) (int, int, bool) {
	startIndex, startErr := strconv.Atoi(path)
	endIndex, endErr := strconv.Atoi(endPath)
	if startErr != nil || endErr != nil || startIndex < 0 || endIndex < startIndex || strconv.Itoa(startIndex) != path || strconv.Itoa(endIndex) != endPath {
		return 0, 0, false
	}
	return startIndex, endIndex, true
}

func chooseDirectCallRange(byPath map[string]variableCandidate, startIndex, endIndex int, endPath string) (variableCandidate, error) {
	var selected variableCandidate
	calls := make([]variableCallSelection, 0, endIndex-startIndex+1)
	for index := startIndex; index <= endIndex; index++ {
		candidate, ok := byPath[strconv.Itoa(index)]
		if !ok || candidate.stmt == nil {
			return variableCandidate{}, fmt.Errorf("construct_path range includes a statement that is not a matching direct call at path %d", index)
		}
		if candidate.refusal != "" {
			return variableCandidate{}, fmt.Errorf("cannot extract expression: %s", candidate.refusal)
		}
		if index == startIndex {
			selected = candidate
		}
		calls = append(calls, variableCallSelection{stmt: candidate.stmt, matchingArgumentIndexes: candidate.matchingArgumentIndexes})
	}
	selected.endPath = endPath
	selected.calls = calls
	return selected, nil
}

func chooseAssignmentRange(byPath map[string]variableCandidate, owner *ast.FuncDecl, path, endPath string) (variableCandidate, error) {
	startParent, startIndex, startOK := splitStatementPath(path)
	endParent, endIndex, endOK := splitStatementPath(endPath)
	if !startOK || !endOK || startParent != endParent || endIndex < startIndex {
		return variableCandidate{}, fmt.Errorf("construct_path range must select ordered direct statement paths in one block")
	}
	if startParent != "" {
		return variableCandidate{}, fmt.Errorf("construct_path range does not support nested or control-flow assignments")
	}
	if endIndex-startIndex >= len(byPath) {
		return variableCandidate{}, fmt.Errorf("construct_path range includes a statement that is not a matching direct simple assignment")
	}
	selected := make([]variableCandidate, 0, endIndex-startIndex+1)
	for index := startIndex; index <= endIndex; index++ {
		statementPath := childLoopPath(startParent, index)
		candidate, ok := byPath[statementPath]
		if !ok || candidate.assignment == nil {
			return variableCandidate{}, fmt.Errorf("construct_path range includes a statement that is not a matching direct simple assignment at path %s", statementPath)
		}
		if candidate.refusal != "" {
			return variableCandidate{}, fmt.Errorf("cannot extract expression: %s", candidate.refusal)
		}
		if candidate.assignment.Tok != token.ASSIGN || len(candidate.assignment.Lhs) != 1 || len(candidate.assignment.Rhs) != 1 || !namedAssignmentTarget(candidate.assignment.Lhs[0]) {
			return variableCandidate{}, fmt.Errorf("cannot extract expression: selected range requires direct one-target simple assignments")
		}
		selected = append(selected, candidate)
	}
	if !directAssignmentRange(owner, selected) {
		return variableCandidate{}, fmt.Errorf("construct_path range must select contiguous direct assignments in one statement block")
	}
	first := selected[0]
	first.endPath = endPath
	first.assignments = make([]variableAssignmentSelection, 0, len(selected))
	for _, candidate := range selected {
		first.assignments = append(first.assignments, variableAssignmentSelection{assignment: candidate.assignment, expression: candidate.expr})
	}
	return first, nil
}

func splitStatementPath(path string) (string, int, bool) {
	before, after, ok := strings.CutLast(path, ".")
	parent, last := "", path
	if ok {
		parent, last = before, after
	}
	index, err := strconv.Atoi(last)
	if err != nil || index < 0 || strconv.Itoa(index) != last {
		return "", 0, false
	}
	return parent, index, true
}

func directAssignmentRange(owner *ast.FuncDecl, selected []variableCandidate) bool {
	if owner == nil || owner.Body == nil || len(selected) == 0 {
		return false
	}
	for index, statement := range owner.Body.List {
		if statement != selected[0].assignment || index+len(selected) > len(owner.Body.List) {
			continue
		}
		for offset, candidate := range selected {
			if owner.Body.List[index+offset] != candidate.assignment {
				return false
			}
		}
		return true
	}
	return false
}

func variablePaths(candidates []variableCandidate) string {
	paths := make([]string, len(candidates))
	for i, candidate := range candidates {
		paths[i] = candidate.path
	}
	return strings.Join(paths, ", ")
}

func sideEffectFree(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return sideEffectFree(node.X)
	case *ast.UnaryExpr:
		return node.Op != token.AND && node.Op != token.ARROW && sideEffectFree(node.X)
	case *ast.BinaryExpr:
		return sideEffectFree(node.X) && sideEffectFree(node.Y)
	case *ast.SelectorExpr:
		return sideEffectFree(node.X)
	case *ast.IndexExpr:
		return sideEffectFree(node.X) && sideEffectFree(node.Index)
	case *ast.TypeAssertExpr:
		return sideEffectFree(node.X)
	default:
		return false
	}
}

func hasTypedInput(expr ast.Expr, target *ast.FuncDecl) bool {
	inputs := make(map[string]bool)
	for _, fields := range []*ast.FieldList{target.Type.Params, target.Recv} {
		if fields == nil {
			continue
		}
		for _, field := range fields.List {
			for _, name := range field.Names {
				inputs[name.Name] = true
			}
		}
	}
	return referencesTypedInput(expr, inputs) && preservesExtractedType(expr, inputs)
}

func preservesExtractedType(expr ast.Expr, inputs map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.ParenExpr:
		return preservesExtractedType(node.X, inputs)
	case *ast.UnaryExpr:
		return preservesExtractedType(node.X, inputs)
	case *ast.BinaryExpr:
		if node.Op == token.SHL || node.Op == token.SHR {
			return referencesTypedInput(node.X, inputs) && preservesExtractedType(node.X, inputs)
		}
	default:
		return true
	}
	return true
}

func referencesTypedInput(expr ast.Expr, inputs map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.Ident:
		return inputs[node.Name]
	case *ast.ParenExpr:
		return referencesTypedInput(node.X, inputs)
	case *ast.UnaryExpr:
		return referencesTypedInput(node.X, inputs)
	case *ast.BinaryExpr:
		return referencesTypedInput(node.X, inputs) || referencesTypedInput(node.Y, inputs)
	case *ast.SelectorExpr:
		return referencesTypedInput(node.X, inputs)
	case *ast.IndexExpr:
		return referencesTypedInput(node.X, inputs) || referencesTypedInput(node.Index, inputs)
	case *ast.TypeAssertExpr:
		return referencesTypedInput(node.X, inputs)
	default:
		return false
	}
}

func variableNameVisible(file *ast.File, target *ast.FuncDecl, name string) bool {
	visible := false
	ast.Inspect(target, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && ident.Name == name {
			visible = true
		}
		return !visible
	})
	if visible {
		return true
	}
	for _, decl := range file.Decls {
		if nodeDeclaresName(decl, name) {
			return true
		}
	}

	return false
}

func resolvedStaticFunctionDeclaration(file *ast.File, target *ast.FuncDecl, stmt *ast.ExprStmt) (*ast.FuncDecl, bool) {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	callee, ok := call.Fun.(*ast.Ident)
	if !ok || parameterHasName(target.Type.Params, callee.Name) || fieldListHasName(target.Recv, callee.Name) || localBindings(target.Body)[callee.Name] {
		return nil, false
	}
	for _, imported := range file.Imports {
		if imported.Name == nil || imported.Name.Name == "." {
			return nil, false
		}
		if imported.Name.Name == callee.Name {
			return nil, false
		}
	}
	var match *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == callee.Name {
			if match != nil {
				return nil, false
			}
			match = function
		}
	}
	return match, match != nil
}

func validateVariableCandidate(ctx context.Context, filePath, owner, name, discriminator, constructPath, endConstructPath string, original, candidate []byte) error {
	cleanPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("resolve variable extraction validation path: %w", err)
	}
	originalContext, err := loadVariableTypeContext(ctx, cleanPath, owner, original)
	if err != nil {
		return err
	}
	originalType, err := originalVariableExpressionType(originalContext, original, discriminator, constructPath, endConstructPath)
	if err != nil {
		return err
	}
	candidateContext, err := loadVariableTypeContext(ctx, cleanPath, owner, candidate)
	if err != nil {
		return err
	}
	candidateType, err := generatedVariableExpressionType(candidateContext, name, discriminator)
	if err != nil {
		return err
	}
	originalCandidates, err := findVariableCandidates(originalContext.fset, original, originalContext.file, originalContext.owner, discriminator)
	if err != nil {
		return err
	}
	selection, err := chooseVariableCandidate(originalCandidates, originalContext.owner, constructPath, endConstructPath)
	if err != nil {
		return err
	}
	if !sameResolvedVariableType(originalType, candidateType) &&
		(selection.ifStatement == nil || !types.Identical(types.Default(originalType), candidateType)) {
		return fmt.Errorf("cannot extract expression %q: generated local changes Go type from %s to %s", discriminator, extractionTypeIdentity(originalType), extractionTypeIdentity(candidateType))
	}
	if len(selection.assignments) > 0 {
		if err := validateAssignmentRangeBindings(candidateContext, selection, name, discriminator, candidate); err != nil {
			return err
		}
	}
	return nil
}

func validateAssignmentRangeBindings(candidateContext variableTypeContext, selection variableCandidate, name, discriminator string, candidateSource []byte) error {
	local, localIdent, err := generatedRangeLocal(candidateContext, name)
	if err != nil || len(selection.assignments) < 2 {
		return fmt.Errorf("cannot extract repeated expression %q: generated local or selected range is ambiguous", discriminator)
	}
	if expressionErr := validateRangeLocalExpression(candidateContext, local, candidateSource, discriminator); expressionErr != nil {
		return fmt.Errorf("cannot extract repeated expression %q: generated local does not preserve the selected expression", discriminator)
	}
	localObject, ok := candidateContext.info.Defs[localIdent].(*types.Var)
	if !ok || localObject == nil {
		return fmt.Errorf("cannot extract repeated expression %q: generated local binding is unresolved", discriminator)
	}
	wantTargets, err := assignmentRangeTargetNames(selection)
	if err != nil {
		return fmt.Errorf("cannot extract repeated expression %q: original assignment target is unresolved", discriminator)
	}
	if !candidateAssignmentRangeUsesLocal(candidateContext, local, wantTargets, name, localObject) {
		return fmt.Errorf("cannot extract repeated expression %q: every selected assignment must use the generated local binding", discriminator)
	}
	return nil
}

func generatedRangeLocal(context variableTypeContext, name string) (*ast.AssignStmt, *ast.Ident, error) {
	var local *ast.AssignStmt
	var localIdent *ast.Ident
	count := 0
	ast.Inspect(context.owner.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return true
		}
		ident, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || ident.Name != name {
			return true
		}
		count++
		local, localIdent = assignment, ident
		return true
	})
	if count != 1 || local == nil {
		return nil, nil, fmt.Errorf("generated local count is %d", count)
	}
	return local, localIdent, nil
}

func validateRangeLocalExpression(context variableTypeContext, local *ast.AssignStmt, candidateSource []byte, discriminator string) error {
	start := context.fset.Position(local.Rhs[0].Pos()).Offset
	end := context.fset.Position(local.Rhs[0].End()).Offset
	if start < 0 || end < start || end > len(candidateSource) || string(candidateSource[start:end]) != discriminator {
		return fmt.Errorf("generated local expression differs from discriminator")
	}
	return nil
}

func assignmentRangeTargetNames(selection variableCandidate) ([]string, error) {
	names := make([]string, len(selection.assignments))
	for index, selected := range selection.assignments {
		if selected.assignment == nil || len(selected.assignment.Lhs) != 1 {
			return nil, fmt.Errorf("assignment target is not a single identifier")
		}
		ident, ok := selected.assignment.Lhs[0].(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("assignment target is not an identifier")
		}
		names[index] = ident.Name
	}
	return names, nil
}

func candidateAssignmentRangeUsesLocal(context variableTypeContext, local *ast.AssignStmt, targetNames []string, name string, localObject *types.Var) bool {
	found := false
	ast.Inspect(context.owner.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for index, statement := range block.List {
			if statement != local || index+1+len(targetNames) > len(block.List) {
				continue
			}
			found = true
			for offset, targetName := range targetNames {
				if !candidateAssignmentUsesLocal(context, block.List[index+1+offset], targetName, name, localObject) {
					found = false
					break
				}
			}
			if found {
				return false
			}
		}
		return true
	})
	return found
}

func candidateAssignmentUsesLocal(context variableTypeContext, statement ast.Stmt, targetName, name string, localObject *types.Var) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	target, targetOK := assignment.Lhs[0].(*ast.Ident)
	value, valueOK := assignment.Rhs[0].(*ast.Ident)
	if !targetOK || !valueOK || target.Name != targetName || value.Name != name || context.info.Uses[value] != localObject {
		return false
	}
	targetObject, targetOK := context.info.Uses[target].(*types.Var)
	return targetOK && ownerLocalVariable(context, targetObject) && targetObject != localObject
}

func loadVariableTypeContext(ctx context.Context, cleanPath, owner string, source []byte) (variableTypeContext, error) {
	overlay := sourcebuf.Overlay(ctx, ".go")
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[cleanPath] = bytes.Clone(source)
	loaded, err := loadExtractionPackages(ctx, filepath.Dir(cleanPath), overlay, extractionLoadOptions{typeInfo: true})
	if err != nil {
		return variableTypeContext{}, fmt.Errorf("type-check variable extraction candidate: %w", err)
	}
	parsed, err := selectVariableTypeSyntax(loaded, cleanPath)
	if err != nil {
		return variableTypeContext{}, err
	}
	receiver, function, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return variableTypeContext{}, fmt.Errorf("parse in_function selector: %w", err)
	}
	target := findExtractOwner(parsed.syntax, function, receiver)
	if target == nil || target.Body == nil {
		return variableTypeContext{}, fmt.Errorf("type-check variable extraction candidate: owner %q was not found", owner)
	}
	return variableTypeContext{owner: target, file: parsed.syntax, info: parsed.info, fset: parsed.fset}, nil
}

type variableTypeSyntax struct {
	syntax *ast.File
	info   *types.Info
	fset   *token.FileSet
}

func selectVariableTypeSyntax(loaded []*packages.Package, cleanPath string) (variableTypeSyntax, error) {
	matches := 0
	var selected variableTypeSyntax
	for _, pkg := range loaded {
		if pkg.TypesInfo == nil {
			continue
		}
		for index, compiled := range pkg.CompiledGoFiles {
			path, err := filepath.Abs(compiled)
			if err != nil || filepath.Clean(path) != cleanPath {
				continue
			}
			if index >= len(pkg.Syntax) {
				return variableTypeSyntax{}, fmt.Errorf("type-check variable extraction candidate: package syntax is incomplete")
			}
			matches++
			selected = variableTypeSyntax{syntax: pkg.Syntax[index], info: pkg.TypesInfo, fset: pkg.Fset}
		}
	}
	if matches != 1 || selected.syntax == nil || selected.info == nil || selected.fset == nil {
		return variableTypeSyntax{}, fmt.Errorf("type-check variable extraction candidate: selected file resolves in %d package variants", matches)
	}
	return selected, nil
}

func originalVariableExpressionType(context variableTypeContext, source []byte, discriminator, constructPath, endConstructPath string) (types.Type, error) {
	candidates, err := findVariableCandidates(context.fset, source, context.file, context.owner, discriminator)
	if err != nil {
		return nil, fmt.Errorf("resolve original variable extraction type: %w", err)
	}
	selection, err := chooseVariableCandidate(candidates, context.owner, constructPath, endConstructPath)
	if err != nil {
		return nil, fmt.Errorf("resolve original variable extraction type: %w", err)
	}
	var expressionType types.Type
	switch {
	case len(selection.assignments) > 0:
		expressionType, err = validateRepeatedAssignmentRange(context, selection, discriminator)
		if err != nil {
			return nil, err
		}
	case selection.stmt != nil:
		expressionType, err = originalCallArgumentExpressionType(context, selection, discriminator)
		if err != nil {
			return nil, err
		}
	default:
		expressionType = context.info.Types[selection.expr].Type
	}
	if expressionType == nil {
		return nil, fmt.Errorf("cannot extract expression %q: Go type information is unavailable", discriminator)
	}
	if selection.ifStatement != nil && !hasTypedExpressionInput(selection.expr, context.info) {
		return nil, fmt.Errorf("cannot extract expression %q: if condition must contain a resolved typed Go input", discriminator)
	}
	if (selection.assignment != nil || selection.declaration != nil) && !hasTypedExpressionInput(selection.expr, context.info) {
		return nil, fmt.Errorf("cannot extract expression %q: assignment or variable initializer RHS must contain a resolved typed Go input", discriminator)
	}
	if selection.returnStmt != nil {
		if !hasSingleReturnResult(context.owner) {
			return nil, fmt.Errorf("cannot extract expression %q: return extraction requires exactly one named or unnamed owner result", discriminator)
		}
		if _, isTuple := expressionType.(*types.Tuple); isTuple {
			return nil, fmt.Errorf("cannot extract expression %q: multi-result return expressions are not supported", discriminator)
		}
		function, ok := context.info.Defs[context.owner.Name].(*types.Func)
		if !ok {
			return nil, fmt.Errorf("cannot extract expression %q: owner result type is unavailable", discriminator)
		}
		signature, ok := function.Type().(*types.Signature)
		if !ok || signature.Results().Len() != 1 {
			return nil, fmt.Errorf("cannot extract expression %q: return extraction requires exactly one named or unnamed owner result", discriminator)
		}
		if !hasTypedExpressionInput(selection.expr, context.info) {
			return nil, fmt.Errorf("cannot extract expression %q: return expression must contain a resolved typed Go input", discriminator)
		}
	}
	return expressionType, nil
}

func originalCallArgumentExpressionType(context variableTypeContext, selection variableCandidate, discriminator string) (types.Type, error) {
	callSelections := selectedVariableCallSelections(selection)
	ownerInputs := ownerInputObjects(context)
	var selectedType types.Type
	for _, callSelection := range callSelections {
		call, ok := callSelection.stmt.X.(*ast.CallExpr)
		if !ok || len(callSelection.matchingArgumentIndexes) == 0 {
			return nil, fmt.Errorf("cannot extract expression %q: selected call argument is no longer available", discriminator)
		}
		candidate := variableCandidate{stmt: callSelection.stmt, matchingArgumentIndexes: callSelection.matchingArgumentIndexes}
		if err := validateCallArgumentSiblings(context, candidate, ownerInputs, discriminator); err != nil {
			return nil, err
		}
		if len(callSelections) > 1 || len(callSelection.matchingArgumentIndexes) > 1 {
			if err := validateRepeatedCallArguments(context, call, callSelection.matchingArgumentIndexes, ownerInputs, discriminator); err != nil {
				return nil, err
			}
		}
		for _, index := range callSelection.matchingArgumentIndexes {
			expressionType := context.info.Types[call.Args[index]].Type
			if expressionType == nil {
				return nil, fmt.Errorf("cannot extract expression %q: Go type information is unavailable at every selected call", discriminator)
			}
			if selectedType != nil && !types.Identical(selectedType, expressionType) {
				return nil, fmt.Errorf("cannot extract expression %q: selected call arguments have different resolved Go types", discriminator)
			}
			selectedType = expressionType
		}
	}
	return selectedType, nil
}

func validateRepeatedAssignmentRange(context variableTypeContext, selection variableCandidate, discriminator string) (types.Type, error) {
	if len(selection.assignments) < 2 {
		return nil, fmt.Errorf("cannot extract repeated expression %q: assignment range must contain at least two statements", discriminator)
	}
	ownerInputs := ownerInputObjects(context)
	usedInputs := make(map[*types.Var]struct{})
	var selectedType types.Type
	for _, selected := range selection.assignments {
		assignment := selected.assignment
		if assignment == nil || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || assignment.Rhs[0] != selected.expression {
			return nil, fmt.Errorf("cannot extract repeated expression %q: selected range is not a direct one-target simple assignment range", discriminator)
		}
		lhs, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || lhs.Name == "_" {
			return nil, fmt.Errorf("cannot extract repeated expression %q: assignment target must be an existing owner-local variable", discriminator)
		}
		lhsObject, ok := context.info.Uses[lhs].(*types.Var)
		if !ok || !ownerLocalVariable(context, lhsObject) {
			return nil, fmt.Errorf("cannot extract repeated expression %q: assignment target must be an existing owner-local variable", discriminator)
		}
		if _, isInput := ownerInputs[lhsObject]; isInput {
			return nil, fmt.Errorf("cannot extract repeated expression %q: assignment target must be distinct from expression inputs", discriminator)
		}
		expressionType := context.info.Types[selected.expression].Type
		if expressionType == nil || !stableScalarArgumentExpression(selected.expression, context.info, ownerInputs, usedInputs) {
			return nil, fmt.Errorf("cannot extract repeated expression %q: every RHS must be a stable scalar expression over typed owner inputs and compiler constants", discriminator)
		}
		if selectedType != nil && !types.Identical(selectedType, expressionType) {
			return nil, fmt.Errorf("cannot extract repeated expression %q: assignment RHS expressions have different resolved Go types", discriminator)
		}
		selectedType = expressionType
	}
	for _, selected := range selection.assignments {
		lhs := selected.assignment.Lhs[0].(*ast.Ident)
		lhsObject := context.info.Uses[lhs].(*types.Var)
		if _, used := usedInputs[lhsObject]; used {
			return nil, fmt.Errorf("cannot extract repeated expression %q: an RHS input is written within the selected range", discriminator)
		}
	}
	if len(usedInputs) == 0 {
		return nil, fmt.Errorf("cannot extract repeated expression %q: every RHS must use at least one typed scalar owner input", discriminator)
	}
	if ownerInputEscapes(context.owner, context.info, usedInputs) {
		return nil, fmt.Errorf("cannot extract repeated expression %q: an RHS input escapes through its address or a nested function", discriminator)
	}
	return selectedType, nil
}

func ownerLocalVariable(context variableTypeContext, target *types.Var) bool {
	if target == nil {
		return false
	}
	found := false
	ast.Inspect(context.owner.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if ok && context.info.Defs[ident] == target {
			found = true
			return false
		}
		return !found
	})
	return found
}

func generatedVariableExpressionType(context variableTypeContext, name, discriminator string) (types.Type, error) {
	var candidateType types.Type
	matches := 0
	ast.Inspect(context.owner.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 {
			return true
		}
		ident, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || ident.Name != name {
			return true
		}
		matches++
		if object := context.info.Defs[ident]; object != nil {
			candidateType = object.Type()
		}
		return true
	})
	if matches != 1 || candidateType == nil {
		return nil, fmt.Errorf("cannot extract expression %q: generated local has no unique resolved type (matches: %d, typed: %t)", discriminator, matches, candidateType != nil)
	}
	return candidateType, nil
}

func sameResolvedVariableType(original, candidate types.Type) bool {
	return extractionTypeIdentity(original) == extractionTypeIdentity(candidate)
}

func extractionTypeIdentity(value types.Type) string {
	return types.TypeString(value, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
}

func hasTypedExpressionInput(expr ast.Expr, info *types.Info) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if found {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		object := info.Uses[ident]
		if object == nil {
			object = info.Defs[ident]
		}
		if object == nil {
			return true
		}
		switch object.(type) {
		case *types.Var, *types.Const:
		default:
			return true
		}
		objectType := object.Type()
		if objectType == nil {
			return true
		}
		basic, ok := objectType.Underlying().(*types.Basic)
		found = !ok || basic.Info()&types.IsUntyped == 0
		return !found
	})
	return found
}

func validateCallArgumentSiblings(context variableTypeContext, selection variableCandidate, ownerInputs map[*types.Var]struct{}, discriminator string) error {
	call, ok := selection.stmt.X.(*ast.CallExpr)
	if !ok || len(selection.matchingArgumentIndexes) == 0 {
		return fmt.Errorf("cannot extract expression %q: selected call argument is no longer available", discriminator)
	}
	selected := make(map[int]struct{}, len(selection.matchingArgumentIndexes))
	for _, index := range selection.matchingArgumentIndexes {
		if index < 0 || index >= len(call.Args) {
			return fmt.Errorf("cannot extract expression %q: selected call argument is no longer available", discriminator)
		}
		selected[index] = struct{}{}
	}
	if len(selected) == 0 {
		return fmt.Errorf("cannot extract expression %q: selected call argument is no longer available", discriminator)
	}
	for index, sibling := range call.Args {
		if _, isSelected := selected[index]; isSelected || context.info.Types[sibling].Value != nil || isPredeclaredNil(sibling, context.info) || isBareScalarOwnerInput(sibling, context.info, ownerInputs) {
			continue
		}
		return fmt.Errorf("cannot extract expression %q: sibling call arguments must be compiler-proven constants, predeclared nil, or bare scalar owner inputs", discriminator)
	}
	return nil
}

func ownerInputObjects(context variableTypeContext) map[*types.Var]struct{} {
	inputs := make(map[*types.Var]struct{})
	for _, fields := range []*ast.FieldList{context.owner.Type.Params, context.owner.Recv} {
		if fields == nil {
			continue
		}
		for _, field := range fields.List {
			for _, name := range field.Names {
				if object, ok := context.info.Defs[name].(*types.Var); ok {
					inputs[object] = struct{}{}
				}
			}
		}
	}
	return inputs
}

func validateRepeatedCallArguments(context variableTypeContext, call *ast.CallExpr, selected []int, ownerInputs map[*types.Var]struct{}, discriminator string) error {
	usedInputs := make(map[*types.Var]struct{})
	var selectedType types.Type
	for _, index := range selected {
		expression := call.Args[index]
		expressionType := context.info.Types[expression].Type
		if expressionType == nil || !stableScalarArgumentExpression(expression, context.info, ownerInputs, usedInputs) {
			return fmt.Errorf("cannot extract repeated expression %q: every matching argument must be a stable scalar expression over owner inputs and compile-time constants", discriminator)
		}
		if selectedType != nil && !types.Identical(selectedType, expressionType) {
			return fmt.Errorf("cannot extract repeated expression %q: matching arguments have different resolved Go types", discriminator)
		}
		selectedType = expressionType
	}
	if ownerInputEscapes(context.owner, context.info, usedInputs) {
		return fmt.Errorf("cannot extract repeated expression %q: an input read by the expression escapes through its address or a nested function", discriminator)
	}
	return nil
}

func stableScalarArgumentExpression(expression ast.Expr, info *types.Info, ownerInputs map[*types.Var]struct{}, usedInputs map[*types.Var]struct{}) bool {
	switch node := expression.(type) {
	case *ast.BasicLit:
		return scalarBasicType(info.Types[node].Type)
	case *ast.Ident:
		switch object := info.Uses[node].(type) {
		case *types.Const:
			return scalarBasicType(object.Type())
		case *types.Var:
			if _, ok := ownerInputs[object]; !ok || !scalarBasicType(object.Type()) {
				return false
			}
			usedInputs[object] = struct{}{}
			return true
		default:
			return false
		}
	case *ast.ParenExpr:
		return scalarBasicType(info.Types[node].Type) && stableScalarArgumentExpression(node.X, info, ownerInputs, usedInputs)
	case *ast.UnaryExpr:
		if node.Op != token.ADD && node.Op != token.SUB && node.Op != token.XOR && node.Op != token.NOT {
			return false
		}
		return scalarBasicType(info.Types[node].Type) && stableScalarArgumentExpression(node.X, info, ownerInputs, usedInputs)
	case *ast.BinaryExpr:
		if !stableScalarBinaryOperator(node.Op) || !scalarBasicType(info.Types[node].Type) {
			return false
		}
		return stableScalarArgumentExpression(node.X, info, ownerInputs, usedInputs) && stableScalarArgumentExpression(node.Y, info, ownerInputs, usedInputs)
	default:
		return false
	}
}

func stableScalarBinaryOperator(operator token.Token) bool {
	switch operator {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
		token.AND, token.OR, token.XOR, token.AND_NOT, token.SHL, token.SHR,
		token.LAND, token.LOR, token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func scalarBasicType(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := value.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch basic.Kind() {
	case types.Bool, types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
		types.Float32, types.Float64, types.Complex64, types.Complex128, types.String,
		types.UntypedBool, types.UntypedInt, types.UntypedRune, types.UntypedFloat, types.UntypedComplex, types.UntypedString:
		return true
	default:
		return false
	}
}

func ownerInputEscapes(owner *ast.FuncDecl, info *types.Info, inputs map[*types.Var]struct{}) bool {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	unsafeUse := false
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return !unsafeUse
		}
		object, ok := info.Uses[ident].(*types.Var)
		if !ok {
			return true
		}
		if _, selected := inputs[object]; !selected {
			return true
		}
		for parent := parents[ident]; parent != nil; parent = parents[parent] {
			if _, captured := parent.(*ast.FuncLit); captured {
				unsafeUse = true
				return false
			}
		}
		parent := parents[ident]
		for {
			paren, ok := parent.(*ast.ParenExpr)
			if !ok {
				break
			}
			parent = parents[paren]
		}
		if unary, ok := parent.(*ast.UnaryExpr); ok && unary.Op == token.AND {
			unsafeUse = true
			return false
		}
		if selector, ok := parent.(*ast.SelectorExpr); ok {
			if extractionUsesImplicitPointerReceiver(info, selector) {
				unsafeUse = true
				return false
			}
		}
		return true
	})
	return unsafeUse
}

func isPredeclaredNil(expr ast.Expr, info *types.Info) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && info.Uses[ident] == types.Universe.Lookup("nil")
}

func isBareScalarOwnerInput(expr ast.Expr, info *types.Info, ownerInputs map[*types.Var]struct{}) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	object, ok := info.Uses[ident].(*types.Var)
	if !ok {
		return false
	}
	if _, isOwnerInput := ownerInputs[object]; !isOwnerInput || object.Type() == nil {
		return false
	}
	basic, ok := object.Type().Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) != 0
}

func callVariableCandidate(fset *token.FileSet, content []byte, index int, stmt *ast.ExprStmt, discriminator string) (variableCandidate, bool, error) {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return variableCandidate{}, false, nil
	}
	if _, ok := call.Fun.(*ast.Ident); !ok {
		return variableCandidate{}, false, nil
	}
	var matches []int
	for argumentIndex, expr := range call.Args {
		matchesSource, err := expressionMatchesSource(fset, content, expr, discriminator)
		if err != nil {
			return variableCandidate{}, false, err
		}
		if matchesSource {
			matches = append(matches, argumentIndex)
		}
	}
	if len(matches) == 0 {
		return variableCandidate{}, false, nil
	}
	candidate := variableCandidate{path: strconv.Itoa(index), stmt: stmt, expr: call.Args[matches[0]], matchingArgumentIndexes: matches}
	if call.Ellipsis.IsValid() {
		candidate.refusal = "variadic expansion is not supported"
	}
	return candidate, true, nil
}

func assignmentVariableCandidate(fset *token.FileSet, content []byte, path string, assignment *ast.AssignStmt, discriminator string) (variableCandidate, bool, error) {
	var matches []ast.Expr
	for _, expr := range assignment.Rhs {
		matchesSource, err := expressionMatchesSource(fset, content, expr, discriminator)
		if err != nil {
			return variableCandidate{}, false, err
		}
		if matchesSource {
			matches = append(matches, expr)
		}
	}
	if len(matches) == 0 {
		return variableCandidate{}, false, nil
	}
	candidate := variableCandidate{path: path, assignment: assignment, expr: matches[0]}
	switch {
	case len(matches) != 1:
		candidate.refusal = "multiple assignment values match the selected expression"
	case assignment.Tok != token.ASSIGN && assignment.Tok != token.DEFINE:
		candidate.refusal = "only simple assignments and short declarations are supported"
	case len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1:
		candidate.refusal = "only assignments with one target and one value are supported"
	case !namedAssignmentTarget(assignment.Lhs[0]):
		candidate.refusal = "assignment target must be a named identifier"
	}
	return candidate, true, nil
}

func variableDeclarationCandidate(fset *token.FileSet, content []byte, path string, declaration *ast.DeclStmt, discriminator string) (variableCandidate, bool, error) {
	genDecl, ok := declaration.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return variableCandidate{}, false, nil
	}
	var valueSpecs []*ast.ValueSpec
	var matches []ast.Expr
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		valueSpecs = append(valueSpecs, valueSpec)
		for _, expression := range valueSpec.Values {
			matchesSource, err := expressionMatchesSource(fset, content, expression, discriminator)
			if err != nil {
				return variableCandidate{}, false, err
			}
			if matchesSource {
				matches = append(matches, expression)
			}
		}
	}
	if len(matches) == 0 {
		return variableCandidate{}, false, nil
	}
	candidate := variableCandidate{path: path, declaration: declaration, expr: matches[0]}
	switch {
	case len(matches) != 1:
		candidate.refusal = "multiple variable initializer expressions match the selected expression"
	case genDecl.Lparen.IsValid():
		candidate.refusal = "grouped var declarations are not supported"
	case len(genDecl.Specs) != 1 || len(valueSpecs) != 1:
		candidate.refusal = "only one local var declaration spec is supported"
	case len(valueSpecs[0].Names) != 1 || len(valueSpecs[0].Values) != 1:
		candidate.refusal = "only var declarations with one name and one initializer are supported"
	case valueSpecs[0].Names[0].Name == "_":
		candidate.refusal = "var declaration target must be a named identifier"
	}
	return candidate, true, nil
}

func namedAssignmentTarget(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name != "_"
}

func expressionMatchesSource(fset *token.FileSet, content []byte, expr ast.Expr, discriminator string) (bool, error) {
	start := fset.Position(expr.Pos()).Offset
	end := fset.Position(expr.End()).Offset
	if start < 0 || end < start || end > len(content) {
		return false, fmt.Errorf("cannot extract expression: source positions are inconsistent")
	}
	return string(content[start:end]) == discriminator, nil
}

func supportedVariableCandidates(candidates []variableCandidate) []variableCandidate {
	eligible := make([]variableCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.refusal == "" {
			eligible = append(eligible, candidate)
		}
	}
	return eligible
}
