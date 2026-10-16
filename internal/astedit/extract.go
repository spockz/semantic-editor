// Package astedit extracts safe contiguous statement regions into local helpers.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
)

// ExtractKind selects the declaration form created by ExtractFunction.
type ExtractKind string

const (
	// ExtractKindFunction creates a package-level helper function.
	ExtractKindFunction ExtractKind = "function"
	// ExtractMethod creates a method with the owner receiver.
	ExtractMethod ExtractKind = "method"
)

// ExtractOptions describes a deliberately bounded extraction contract.
type ExtractOptions struct {
	Kind      ExtractKind
	StartLine int
	EndLine   int
}

type extractionOutput struct {
	name     string
	typeName string
}

type extractionInput struct {
	ctx         context.Context
	cleanPath   string
	ownedBuffer *sourcebuf.Buffer
	standalone  bool
	content     []byte
	fset        *token.FileSet
	file        *ast.File
	target      *ast.FuncDecl
	selected    []ast.Stmt
}

type extractionCandidate struct {
	source      []byte
	declaration string
}

// ExtractFunction moves consecutive top-level call statements, optionally
// preceded by one supported local declaration, into a helper.
func ExtractFunction(ctx context.Context, filePath, owner, helper string, opts ExtractOptions) (string, error) {
	input, candidate, err := prepareFunctionExtraction(ctx, filePath, owner, helper, opts)
	if err != nil {
		return "", err
	}
	return publishExtraction(input, candidate, owner)
}

func validateExtractRequest(helper string, opts ExtractOptions) error {
	if opts.Kind != ExtractKindFunction && opts.Kind != ExtractMethod {
		return fmt.Errorf("unsupported extraction kind %q", opts.Kind)
	}
	if opts.StartLine <= 0 || opts.EndLine < opts.StartLine {
		return fmt.Errorf("start_line and end_line must describe a positive ordered line range")
	}
	if !token.IsIdentifier(helper) || token.Lookup(helper).IsKeyword() || helper == "_" {
		return fmt.Errorf("symbol %q is not a valid helper identifier", helper)
	}
	if types.Universe.Lookup(helper) != nil {
		return fmt.Errorf("cannot extract: helper name %q collides with a predeclared Go identifier", helper)
	}
	return nil
}

func loadExtractionInput(ctx context.Context, filePath, owner string, opts ExtractOptions) (extractionInput, error) {
	input := extractionInput{cleanPath: filepath.Clean(filePath)}
	preparedCtx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, input.cleanPath)
	if err != nil {
		return extractionInput{}, err
	}
	input.ctx, input.ownedBuffer, input.standalone = preparedCtx, ownedBuffer, standalone
	input.content, err = sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return extractionInput{}, fmt.Errorf("read extraction target: %w", err)
	}
	recv, name, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return extractionInput{}, fmt.Errorf("parse in_function selector: %w", err)
	}
	input.fset = token.NewFileSet()
	input.file, err = parser.ParseFile(input.fset, input.cleanPath, input.content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return extractionInput{}, &SyntaxError{File: input.cleanPath, Pos: extractSyntaxPosition(input.fset, err), Cause: err, Err: ErrSyntax}
	}
	input.target = findExtractOwner(input.file, name, recv)
	if input.target == nil {
		return extractionInput{}, &symbol.SymbolError{Op: "extract_function", File: input.cleanPath, Symbol: owner, Err: symbol.ErrNotFound}
	}
	if ownerErr := validateExtractOwner(input.file, input.target, owner, opts.Kind); ownerErr != nil {
		return extractionInput{}, ownerErr
	}
	input.selected, err = selectExtractStatements(input.fset, input.target.Body, owner, opts)
	if err != nil {
		return extractionInput{}, err
	}
	return input, nil
}

func planExtraction(input extractionInput, owner, helper string, opts ExtractOptions) (extractionCandidate, error) {
	collides, err := helperNameExists(input.ctx, input.cleanPath, input.file.Name.Name, helper, opts.Kind, input.target)
	if err != nil {
		return extractionCandidate{}, err
	}
	if collides {
		return extractionCandidate{}, fmt.Errorf("cannot extract: package declaration %q already exists", helper)
	}
	outputs, err := extractOutputs(input.fset, input.target, input.selected)
	if err != nil {
		return extractionCandidate{}, fmt.Errorf("cannot extract %s: %w", owner, err)
	}
	if aliasErr := validateOutputAliases(input.selected, outputs, owner); aliasErr != nil {
		return extractionCandidate{}, aliasErr
	}
	if typeNameErr := validateOutputTypeNames(input.ctx, input.cleanPath, input.file, outputs, owner, input.content); typeNameErr != nil {
		return extractionCandidate{}, typeNameErr
	}
	params, err := extractionParameters(input.fset, input.target, input.selected, opts.Kind, outputs)
	if err != nil {
		return extractionCandidate{}, fmt.Errorf("cannot extract %s: %w", owner, err)
	}
	if forwardedErr := validateForwardedNames(input.target, input.selected, params.names, owner); forwardedErr != nil {
		return extractionCandidate{}, forwardedErr
	}
	if scopeErr := validateExtractedScope(input.file, input.target, input.selected, params.names, outputs, owner); scopeErr != nil {
		return extractionCandidate{}, scopeErr
	}
	selected := input.selected
	selectedSource := strings.TrimSpace(string(input.content[input.fset.Position(selected[0].Pos()).Offset:input.fset.Position(selected[len(selected)-1].End()).Offset]))
	callSource := extractionCall(input.target, helper, params.names, outputs, opts.Kind)
	declaration, err := extractionDeclaration(input.fset, input.target, helper, opts.Kind, params.declarations, outputs, selectedSource)
	if err != nil {
		return extractionCandidate{}, err
	}
	source, err := rewriteExtractedStatements(input.content, input.fset, input.target, selected, callSource, declaration, owner)
	if err != nil {
		return extractionCandidate{}, err
	}
	return extractionCandidate{source: source, declaration: declaration}, nil
}

func publishExtraction(input extractionInput, candidate extractionCandidate, owner string) (string, error) {
	formatted, err := prepareFunctionExtractionOutput(input, candidate)
	if err != nil {
		return "", err
	}
	current, err := sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return "", fmt.Errorf("recheck extraction target before publication: %w", err)
	}
	if !bytes.Equal(current, input.content) {
		return "", fmt.Errorf("cannot extract %s: source changed during extraction", owner)
	}
	buffer := sourcebuf.FromContext(input.ctx)
	if applyErr := buffer.Apply(input.ctx, []sourcebuf.Change{{Path: input.cleanPath, Before: input.content, After: formatted}}); applyErr != nil {
		return "", fmt.Errorf("apply extracted source after compiler preflight: %w", applyErr)
	}
	finalSource, err := sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return "", fmt.Errorf("read formatted extraction result: %w", err)
	}
	if input.standalone {
		if err := input.ownedBuffer.Flush(input.ctx); err != nil {
			return "", fmt.Errorf("publish extracted source: %w", err)
		}
	}
	return UnifiedDiff(filepath.Base(input.cleanPath), string(input.content), string(finalSource)), nil
}

func validateOutputAliases(selected []ast.Stmt, outputs []extractionOutput, owner string) error {
	if len(outputs) == 0 {
		return nil
	}
	outputNames := make(map[string]bool, len(outputs))
	for _, output := range outputs {
		outputNames[output.name] = true
	}
	for _, stmt := range selected {
		var addressTaken string
		ast.Inspect(stmt, func(node ast.Node) bool {
			unary, ok := node.(*ast.UnaryExpr)
			if !ok || unary.Op != token.AND {
				return true
			}
			addressTaken = forwardedIdent(unary.X, outputNames)
			return addressTaken == ""
		})
		if addressTaken != "" {
			return fmt.Errorf("cannot extract %s: result local %q is address-taken in the selected region", owner, addressTaken)
		}
	}
	return nil
}

func normalizeExtractionNewlines(source []byte) []byte {
	return append(bytes.TrimRight(source, "\n"), '\n')
}

func findExtractOwner(file *ast.File, name, recv string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == recv {
			return fn
		}
	}
	return nil
}

func validateExtractOwner(file *ast.File, target *ast.FuncDecl, owner string, kind ExtractKind) error {
	if target.Body == nil {
		return fmt.Errorf("cannot extract from %s: function has no body", owner)
	}
	if target.Type.TypeParams != nil && len(target.Type.TypeParams.List) > 0 {
		return fmt.Errorf("cannot extract from generic function %s", owner)
	}
	if target.Type.Results != nil {
		for _, field := range target.Type.Results.List {
			if len(field.Names) > 0 {
				return fmt.Errorf("cannot extract from %s: named result variables may be affected by the moved statements", owner)
			}
		}
	}
	if target.Recv != nil {
		if len(target.Recv.List) != 1 || len(target.Recv.List[0].Names) != 1 {
			return fmt.Errorf("cannot extract from %s: receiver must have one named variable", owner)
		}
		switch target.Recv.List[0].Type.(type) {
		case *ast.IndexExpr, *ast.IndexListExpr:
			return fmt.Errorf("cannot extract from generic receiver method %s", owner)
		}
	}
	if kind == ExtractMethod && target.Recv == nil {
		return fmt.Errorf("cannot extract method from function %s without a receiver", owner)
	}
	for _, group := range file.Comments {
		if group.Pos() > target.Body.Lbrace && group.End() < target.Body.Rbrace {
			return fmt.Errorf("cannot extract from %s: comments inside the owner function require manual comment ownership review", owner)
		}
	}
	return nil
}

func selectExtractStatements(fset *token.FileSet, body *ast.BlockStmt, owner string, opts ExtractOptions) ([]ast.Stmt, error) {
	selected, err := findExtractStatementRange(fset, body, owner, opts)
	if err != nil {
		return nil, err
	}
	for i, stmt := range selected {
		if err := validateExtractStatement(fset, stmt, i, len(selected), owner, opts); err != nil {
			return nil, err
		}
	}
	return selected, nil
}

func findExtractStatementRange(fset *token.FileSet, body *ast.BlockStmt, owner string, opts ExtractOptions) ([]ast.Stmt, error) {
	startIdx, endIdx := -1, -1
	for i, stmt := range body.List {
		pos, end := fset.Position(stmt.Pos()), fset.Position(stmt.End())
		if pos.Line == opts.StartLine {
			if startIdx >= 0 {
				return nil, fmt.Errorf("start_line %d matches multiple statements in %s", opts.StartLine, owner)
			}
			startIdx = i
		}
		if end.Line == opts.EndLine {
			if endIdx >= 0 {
				return nil, fmt.Errorf("end_line %d matches multiple statements in %s", opts.EndLine, owner)
			}
			endIdx = i
		}
	}
	if startIdx < 0 || endIdx < 0 || endIdx < startIdx {
		return nil, fmt.Errorf("line range %d-%d does not select a contiguous top-level statement region in %s", opts.StartLine, opts.EndLine, owner)
	}
	return body.List[startIdx : endIdx+1], nil
}

func validateExtractStatement(fset *token.FileSet, stmt ast.Stmt, index, count int, owner string, opts ExtractOptions) error {
	switch item := stmt.(type) {
	case *ast.ExprStmt:
		if _, ok := item.X.(*ast.CallExpr); !ok {
			return fmt.Errorf("cannot extract %s: selected statement on line %d is not a call expression", owner, fset.Position(stmt.Pos()).Line)
		}
	case *ast.AssignStmt:
		if index != 0 || item.Tok != token.DEFINE || len(item.Lhs) == 0 || len(item.Lhs) != len(item.Rhs) {
			return fmt.Errorf("cannot extract %s: only one leading := declaration with matching named results is supported", owner)
		}
		if err := validateExtractionResultNames(item.Lhs, owner); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot extract %s: selected statement on line %d is not a call or supported local declaration", owner, fset.Position(stmt.Pos()).Line)
	}
	if index == 0 && fset.Position(stmt.Pos()).Line != opts.StartLine || index == count-1 && fset.Position(stmt.End()).Line != opts.EndLine {
		return fmt.Errorf("line range %d-%d must match statement boundaries exactly", opts.StartLine, opts.EndLine)
	}
	return nil
}

func validateExtractionResultNames(lhs []ast.Expr, owner string) error {
	seen := make(map[string]bool, len(lhs))
	for _, expression := range lhs {
		ident, ok := expression.(*ast.Ident)
		if !ok || ident.Name == "_" || seen[ident.Name] {
			return fmt.Errorf("cannot extract %s: extraction results must be distinct named local variables", owner)
		}
		seen[ident.Name] = true
	}
	return nil
}

func validateExtractedScope(file *ast.File, target *ast.FuncDecl, selected []ast.Stmt, params []string, outputs []extractionOutput, owner string) error {
	allowed := extractionScope(file, params)
	locals := localBindings(target.Body)
	for _, output := range outputs {
		delete(locals, output.name)
		allowed[output.name] = true
	}
	for _, stmt := range selected {
		var literal bool
		ast.Inspect(stmt, func(node ast.Node) bool {
			if _, ok := node.(*ast.FuncLit); ok {
				literal = true
				return false
			}
			if _, ok := node.(*ast.CompositeLit); ok {
				literal = true
				return false
			}
			return !literal
		})
		if literal {
			return fmt.Errorf("cannot extract %s: function literals and composite literals in the selected region are unsupported because their nested scope and keyed identifiers cannot be proven safely", owner)
		}
		if err := rejectShadowedNames(stmt, allowed, locals); err != nil {
			return fmt.Errorf("cannot extract %s: %w", owner, err)
		}
	}
	return nil
}

func validateForwardedNames(owner *ast.FuncDecl, selected []ast.Stmt, forwarded []string, ownerName string) error {
	if err := rejectSelectedRecoverCalls(selected, ownerName); err != nil {
		return err
	}
	info := forwardedParameterInfo(owner, forwarded)
	if len(info.names) == 0 {
		return nil
	}
	if err := rejectAddressedOrCapturedForwarded(owner.Body, info.names, ownerName); err != nil {
		return err
	}
	return rejectImplicitPointerReceivers(selected, info.names, info.types, ownerName)
}

func rejectSelectedRecoverCalls(selected []ast.Stmt, ownerName string) error {
	for _, stmt := range selected {
		var recoveryCall bool
		ast.Inspect(stmt, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "recover" {
					recoveryCall = true
					return false
				}
			}
			return !recoveryCall
		})
		if recoveryCall {
			return fmt.Errorf("cannot extract %s: selected direct recover() calls change recovery semantics when moved", ownerName)
		}
	}
	return nil
}

type forwardedNameInfo struct {
	names map[string]bool
	types map[string]ast.Expr
}

func forwardedParameterInfo(owner *ast.FuncDecl, forwarded []string) forwardedNameInfo {
	info := forwardedNameInfo{names: make(map[string]bool, len(forwarded)), types: make(map[string]ast.Expr, len(forwarded))}
	for _, name := range forwarded {
		info.names[name] = true
		if owner.Recv != nil && owner.Recv.List[0].Names[0].Name == name {
			info.types[name] = owner.Recv.List[0].Type
			continue
		}
		if owner.Type.Params != nil {
			for _, field := range owner.Type.Params.List {
				for _, ident := range field.Names {
					if ident.Name == name {
						info.types[name] = field.Type
					}
				}
			}
		}
	}
	return info
}

func rejectAddressedOrCapturedForwarded(body *ast.BlockStmt, forwardedSet map[string]bool, ownerName string) error {
	var unsafe error
	ast.Inspect(body, func(node ast.Node) bool {
		if unsafe != nil || node == nil {
			return false
		}
		switch item := node.(type) {
		case *ast.UnaryExpr:
			if item.Op == token.AND {
				if name := forwardedIdent(item.X, forwardedSet); name != "" {
					unsafe = fmt.Errorf("cannot extract %s: forwarded parameter or receiver %q is address-taken in the owner body", ownerName, name)
					return false
				}
			}
		case *ast.FuncLit:
			var captured string
			ast.Inspect(item.Body, func(nested ast.Node) bool {
				ident, ok := nested.(*ast.Ident)
				if ok && forwardedSet[ident.Name] && !parentSelectorIdent(item.Body, ident) {
					captured = ident.Name
					return false
				}
				return captured == ""
			})
			if captured != "" {
				unsafe = fmt.Errorf("cannot extract %s: forwarded parameter or receiver %q may be captured by a closure in the owner body", ownerName, captured)
				return false
			}
		}
		return true
	})
	if unsafe != nil {
		return unsafe
	}
	return nil
}

func rejectImplicitPointerReceivers(selected []ast.Stmt, forwardedSet map[string]bool, typesByName map[string]ast.Expr, ownerName string) error {
	var unsafe error
	for _, stmt := range selected {
		ast.Inspect(stmt, func(node ast.Node) bool {
			if unsafe != nil || node == nil {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			root := forwardedIdent(selector.X, forwardedSet)
			if root == "" || isPointerType(typesByName[root]) {
				return true
			}
			unsafe = fmt.Errorf("cannot extract %s: call %s may use an implicit pointer receiver on forwarded value %q", ownerName, selector.Sel.Name, root)
			return false
		})
		if unsafe != nil {
			return unsafe
		}
	}
	return nil
}

func forwardedIdent(expr ast.Expr, forwarded map[string]bool) string {
	switch node := expr.(type) {
	case *ast.Ident:
		if forwarded[node.Name] {
			return node.Name
		}
	case *ast.SelectorExpr:
		return forwardedIdent(node.X, forwarded)
	case *ast.IndexExpr:
		return forwardedIdent(node.X, forwarded)
	case *ast.IndexListExpr:
		return forwardedIdent(node.X, forwarded)
	case *ast.ParenExpr:
		return forwardedIdent(node.X, forwarded)
	}
	return ""
}

func isPointerType(expr ast.Expr) bool {
	for {
		switch node := expr.(type) {
		case *ast.StarExpr:
			return true
		case *ast.ParenExpr:
			expr = node.X
		default:
			return false
		}
	}
}

func extractionCall(target *ast.FuncDecl, helper string, params []string, outputs []extractionOutput, kind ExtractKind) string {
	args := append([]string(nil), params...)
	callName := helper
	if kind == ExtractMethod {
		receiver := target.Recv.List[0]
		callName = receiver.Names[0].Name + "." + helper
		args = removeString(args, receiver.Names[0].Name)
	}
	call := callName + "(" + strings.Join(args, ", ") + ")"
	if len(outputs) > 0 {
		names := make([]string, len(outputs))
		for index, output := range outputs {
			names[index] = output.name
		}
		call = strings.Join(names, ", ") + " := " + call
	}
	return call
}

func rewriteExtractedStatements(content []byte, fset *token.FileSet, target *ast.FuncDecl, selected []ast.Stmt, callSource, declSource, owner string) ([]byte, error) {
	start := fset.Position(selected[0].Pos()).Offset
	end := fset.Position(selected[len(selected)-1].End()).Offset
	insert := fset.Position(target.End()).Offset
	if insert < end {
		return nil, fmt.Errorf("cannot extract %s: selected region overlaps the declaration boundary", owner)
	}
	updated := make([]byte, 0, len(content)+len(declSource)+len(callSource))
	updated = append(updated, content[:start]...)
	updated = append(updated, callSource...)
	updated = append(updated, content[end:insert]...)
	updated = append(updated, '\n', '\n')
	updated = append(updated, declSource...)
	updated = append(updated, content[insert:]...)
	return updated, nil
}
func extractOutputs(fset *token.FileSet, owner *ast.FuncDecl, selected []ast.Stmt) ([]extractionOutput, error) {
	assignment, ok := selected[0].(*ast.AssignStmt)
	if !ok {
		return nil, nil
	}
	if len(assignment.Lhs) == 0 || len(assignment.Lhs) != len(assignment.Rhs) {
		return nil, fmt.Errorf("cannot extract declaration with mismatched result expressions")
	}
	outputs := make([]extractionOutput, 0, len(assignment.Lhs))
	for index, lhs := range assignment.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			return nil, fmt.Errorf("cannot extract: declaration results must be named local variables")
		}
		if localDeclarationCount(owner.Body, ident.Name) != 1 {
			return nil, fmt.Errorf("local %q has multiple owner-scope bindings and cannot be moved safely", ident.Name)
		}
		if parameterHasName(owner.Type.Params, ident.Name) || owner.Recv != nil && fieldListHasName(owner.Recv, ident.Name) {
			return nil, fmt.Errorf("local %q overlaps an owner parameter or receiver", ident.Name)
		}
		usedAfter := false
		for _, later := range owner.Body.List {
			if later.Pos() > selected[len(selected)-1].End() && statementUsesName(later, ident.Name) {
				usedAfter = true
				break
			}
		}
		if !usedAfter {
			return nil, fmt.Errorf("local %q is not used after the selected region", ident.Name)
		}
		resultType, err := inferExtractType(fset, owner, assignment.Rhs[index])
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, extractionOutput{name: ident.Name, typeName: resultType})
	}
	return outputs, nil
}

func localDeclarationCount(body *ast.BlockStmt, name string) int {
	count := 0
	ast.Inspect(body, func(node ast.Node) bool {
		count += localDeclarationCountAt(node, name)
		return true
	})
	return count
}

func localDeclarationCountAt(node ast.Node, name string) int {
	switch item := node.(type) {
	case *ast.AssignStmt:
		if item.Tok == token.DEFINE {
			return matchingIdentifierCount(item.Lhs, name)
		}
	case *ast.ValueSpec:
		return matchingNameCount(item.Names, name)
	case *ast.RangeStmt:
		if item.Tok == token.DEFINE {
			return matchingIdentifierCount([]ast.Expr{item.Key, item.Value}, name)
		}
	}
	return 0
}

func matchingIdentifierCount(expressions []ast.Expr, name string) int {
	count := 0
	for _, expr := range expressions {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
			count++
		}
	}
	return count
}

func matchingNameCount(identifiers []*ast.Ident, name string) int {
	count := 0
	for _, ident := range identifiers {
		if ident.Name == name {
			count++
		}
	}
	return count
}

func parameterHasName(fields *ast.FieldList, name string) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		for _, ident := range field.Names {
			if ident.Name == name {
				return true
			}
		}
	}
	return false
}

func fieldListHasName(fields *ast.FieldList, name string) bool {
	return parameterHasName(fields, name)
}

func statementUsesName(stmt ast.Stmt, name string) bool {
	used := false
	ast.Inspect(stmt, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && ident.Name == name && !parentSelectorIdent(stmt, ident) {
			used = true
			return false
		}
		return !used
	})
	return used
}

func inferExtractType(fset *token.FileSet, owner *ast.FuncDecl, expr ast.Expr) (string, error) {
	switch node := expr.(type) {
	case *ast.BasicLit:
		switch node.Kind {
		case token.INT:
			return "int", nil
		case token.FLOAT:
			return "float64", nil
		case token.CHAR:
			return "rune", nil
		case token.STRING:
			return "string", nil
		default:
			return "", fmt.Errorf("literal kind %s has no supported inferred extraction type", node.Kind)
		}
	case *ast.Ident:
		if node.Name == "true" || node.Name == "false" {
			return "bool", nil
		}
		if owner.Type.Params != nil {
			for _, field := range owner.Type.Params.List {
				for _, name := range field.Names {
					if name.Name == node.Name {
						return nodeSource(fset, field.Type)
					}
				}
			}
		}
	case *ast.UnaryExpr:
		if node.Op == token.ADD || node.Op == token.SUB || node.Op == token.NOT {
			return inferExtractType(fset, owner, node.X)
		}
	case *ast.ParenExpr:
		return inferExtractType(fset, owner, node.X)
	}
	return "", fmt.Errorf("cannot infer a safe return type for local declaration expression %q", formatExpr(fset, expr))
}

func formatExpr(fset *token.FileSet, expr ast.Expr) string {
	text, err := nodeSource(fset, expr)
	if err != nil {
		return "<unprintable>"
	}
	return text
}

type extractionParameterSet struct {
	names        []string
	declarations []string
}

func extractionParameters(fset *token.FileSet, owner *ast.FuncDecl, selected []ast.Stmt, kind ExtractKind, outputs []extractionOutput) (extractionParameterSet, error) {
	used := extractionUsedNames(selected, outputs)
	params := extractionParameterSet{}
	if err := appendReceiverParameter(fset, owner, kind, used, &params); err != nil {
		return extractionParameterSet{}, err
	}
	if err := appendFunctionParameters(fset, owner, used, &params); err != nil {
		return extractionParameterSet{}, err
	}
	if kind == ExtractMethod && owner.Recv == nil {
		return extractionParameterSet{}, fmt.Errorf("method extraction requires an owner receiver")
	}
	return params, nil
}

func extractionUsedNames(selected []ast.Stmt, outputs []extractionOutput) map[string]bool {
	used := make(map[string]bool)
	for _, stmt := range selected {
		ast.Inspect(stmt, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok || ident.Name == "_" || parentSelectorIdent(stmt, ident) || slices.ContainsFunc(outputs, func(output extractionOutput) bool { return output.name == ident.Name }) {
				return true
			}
			used[ident.Name] = true
			return true
		})
	}
	return used
}

func appendReceiverParameter(fset *token.FileSet, owner *ast.FuncDecl, kind ExtractKind, used map[string]bool, params *extractionParameterSet) error {
	if owner.Recv == nil {
		return nil
	}
	field := owner.Recv.List[0]
	name := field.Names[0].Name
	if !used[name] {
		return nil
	}
	params.names = append(params.names, name)
	if kind != ExtractKindFunction {
		return nil
	}
	text, err := nodeSource(fset, field.Type)
	if err != nil {
		return err
	}
	params.declarations = append(params.declarations, name+" "+text)
	return nil
}

func appendFunctionParameters(fset *token.FileSet, owner *ast.FuncDecl, used map[string]bool, params *extractionParameterSet) error {
	if owner.Type.Params != nil {
		for _, field := range owner.Type.Params.List {
			if _, ok := field.Type.(*ast.Ellipsis); ok {
				return fmt.Errorf("variadic owner parameters are unsupported")
			}
			if len(field.Names) == 0 {
				continue
			}
			typeText, err := nodeSource(fset, field.Type)
			if err != nil {
				return err
			}
			for _, ident := range field.Names {
				if used[ident.Name] && !slices.Contains(params.names, ident.Name) {
					params.declarations = append(params.declarations, ident.Name+" "+typeText)
					params.names = append(params.names, ident.Name)
				}
			}
		}
	}
	return nil
}

func extractionScope(file *ast.File, params []string) map[string]bool {
	allowed := make(map[string]bool)
	for _, name := range params {
		allowed[name] = true
	}
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			allowed[node.Name.Name] = true
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					allowed[item.Name.Name] = true
				case *ast.ValueSpec:
					for _, ident := range item.Names {
						allowed[ident.Name] = true
					}
				}
			}
		}
	}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if spec.Name != nil {
			if spec.Name.Name != "_" && spec.Name.Name != "." {
				allowed[spec.Name.Name] = true
			}
		} else {
			alias := filepath.Base(path)
			if token.IsIdentifier(alias) {
				allowed[alias] = true
			}
		}
	}
	for _, name := range types.Universe.Names() {
		allowed[name] = true
	}
	return allowed
}

func localBindings(body *ast.BlockStmt) map[string]bool {
	locals := make(map[string]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		addLocalBindingsAt(node, locals)
		return true
	})
	return locals
}

func addLocalBindingsAt(node ast.Node, locals map[string]bool) {
	switch item := node.(type) {
	case *ast.AssignStmt:
		if item.Tok == token.DEFINE {
			addIdentifierBindings(item.Lhs, locals)
		}
	case *ast.ValueSpec:
		for _, ident := range item.Names {
			locals[ident.Name] = true
		}
	case *ast.RangeStmt:
		if item.Tok == token.DEFINE {
			addIdentifierBindings([]ast.Expr{item.Key, item.Value}, locals)
		}
	}
}

func addIdentifierBindings(expressions []ast.Expr, locals map[string]bool) {
	for _, expr := range expressions {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name != "_" {
			locals[ident.Name] = true
		}
	}
}

func rejectShadowedNames(stmt ast.Stmt, allowed, locals map[string]bool) error {
	var result error
	ast.Inspect(stmt, func(node ast.Node) bool {
		if result != nil || node == nil {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok || parentSelectorIdent(stmt, ident) {
			return true
		}
		if locals[ident.Name] {
			result = fmt.Errorf("identifier %q may be shadowed by an owner-local declaration", ident.Name)
			return false
		}
		if ident.Name == "_" || !allowed[ident.Name] {
			result = fmt.Errorf("identifier %q is outside the extraction scope", ident.Name)
			return false
		}
		return true
	})
	return result
}

func parentSelectorIdent(root ast.Node, target *ast.Ident) bool {
	matched := false
	ast.Inspect(root, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel == target {
			matched = true
			return false
		}
		return !matched
	})
	return matched
}

func extractionDeclaration(fset *token.FileSet, owner *ast.FuncDecl, helper string, kind ExtractKind, params []string, outputs []extractionOutput, body string) (string, error) {
	var prefix string
	if kind == ExtractMethod {
		receiver := owner.Recv.List[0]
		receiverType, err := nodeSource(fset, receiver.Type)
		if err != nil {
			return "", err
		}
		prefix = "func (" + receiver.Names[0].Name + " " + receiverType + ") " + helper
	} else {
		prefix = "func " + helper
	}
	result := ""
	bodySource := body
	if len(outputs) == 1 {
		result = " " + outputs[0].typeName
	} else if len(outputs) > 1 {
		resultTypes := make([]string, len(outputs))
		for index, output := range outputs {
			resultTypes[index] = output.typeName
		}
		result = " (" + strings.Join(resultTypes, ", ") + ")"
	}
	if len(outputs) > 0 {
		resultNames := make([]string, len(outputs))
		for index, output := range outputs {
			resultNames[index] = output.name
		}
		bodySource += "\nreturn " + strings.Join(resultNames, ", ")
	}
	decl := prefix + "(" + strings.Join(params, ", ") + ")" + result + " {\n" + bodySource + "\n}"
	formatted, err := format.Source([]byte(decl))
	if err != nil {
		return "", fmt.Errorf("format extracted declaration: %w", err)
	}
	return strings.TrimSpace(string(formatted)), nil
}

func nodeSource(fset *token.FileSet, node ast.Node) (string, error) {
	var out bytes.Buffer
	if err := format.Node(&out, fset, node); err != nil {
		return "", fmt.Errorf("format extraction syntax: %w", err)
	}
	return out.String(), nil
}

func removeString(values []string, value string) []string {
	for i, item := range values {
		if item == value {
			return append(values[:i], values[i+1:]...)
		}
	}
	return values
}

func helperNameExists(ctx context.Context, filePath, packageName, helper string, kind ExtractKind, owner *ast.FuncDecl) (bool, error) {
	entries, err := os.ReadDir(filepath.Dir(filePath))
	if err != nil {
		return false, fmt.Errorf("read Go package directory for helper collision check: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		candidate := filepath.Join(filepath.Dir(filePath), entry.Name())
		content, err := sourcebuf.ReadFile(ctx, candidate)
		if err != nil {
			return false, fmt.Errorf("read Go package file %q for helper collision check: %w", candidate, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), candidate, content, parser.ParseComments)
		if err != nil {
			return false, fmt.Errorf("parse Go package file %q for helper collision check: %w", candidate, err)
		}
		if parsed.Name.Name != packageName {
			continue
		}
		if packageDeclaresExtractionHelper(parsed, helper, kind, extractReceiverName(owner.Recv)) {
			return true, nil
		}
	}
	return false, nil
}

func packageDeclaresExtractionHelper(file *ast.File, helper string, kind ExtractKind, receiver string) bool {
	for _, decl := range file.Decls {
		if declarationMatchesExtractionHelper(decl, helper, kind, receiver) {
			return true
		}
	}
	return false
}

func declarationMatchesExtractionHelper(decl ast.Decl, helper string, kind ExtractKind, receiver string) bool {
	switch node := decl.(type) {
	case *ast.FuncDecl:
		if node.Name.Name != helper {
			return false
		}
		if kind == ExtractKindFunction {
			return node.Recv == nil
		}
		return node.Recv != nil && extractReceiverName(node.Recv) == receiver
	case *ast.GenDecl:
		if kind == ExtractMethod {
			return false
		}
		return generalDeclarationMatchesHelper(node, helper)
	default:
		return false
	}
}

func generalDeclarationMatchesHelper(decl *ast.GenDecl, helper string) bool {
	for _, spec := range decl.Specs {
		switch item := spec.(type) {
		case *ast.TypeSpec:
			if item.Name.Name == helper {
				return true
			}
		case *ast.ValueSpec:
			if slices.ContainsFunc(item.Names, func(name *ast.Ident) bool { return name.Name == helper }) {
				return true
			}
		}
	}
	return false
}

func validateExtractionCandidate(ctx context.Context, filePath string, source []byte) error {
	overlay := sourcebuf.Overlay(ctx)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	cleanPath := filepath.Clean(filePath)
	overlay[cleanPath] = bytes.Clone(source)
	if _, err := loadExtractionPackages(ctx, filepath.Dir(cleanPath), overlay, false); err != nil {
		return fmt.Errorf("validate complete extracted package: %w", err)
	}
	return nil
}

func validateOutputTypeNames(ctx context.Context, filePath string, file *ast.File, outputs []extractionOutput, owner string, source []byte) error {
	overlay := sourcebuf.Overlay(ctx)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[filepath.Clean(filePath)] = bytes.Clone(source)
	loaded, err := loadExtractionPackages(ctx, filepath.Dir(filePath), overlay, false)
	if err != nil {
		return fmt.Errorf("resolve extracted result types: %w", err)
	}
	if len(loaded) != 1 || loaded[0].Types == nil {
		return fmt.Errorf("resolve extracted result types: package scope is ambiguous")
	}
	for _, output := range outputs {
		universeType := types.Universe.Lookup(output.typeName)
		if universeType == nil {
			continue
		}
		if packageType := loaded[0].Types.Scope().Lookup(output.typeName); packageType != nil {
			return fmt.Errorf("cannot extract %s: generated result type %q is shadowed by a package declaration", owner, output.typeName)
		}
		for _, imported := range file.Imports {
			var alias string
			if imported.Name != nil {
				alias = imported.Name.Name
			} else {
				importPath, unquoteErr := strconv.Unquote(imported.Path.Value)
				if unquoteErr != nil {
					return fmt.Errorf("read import path while checking extracted result type: %w", unquoteErr)
				}
				importedPackage := loaded[0].Imports[importPath]
				if importedPackage == nil || importedPackage.Name == "" {
					return fmt.Errorf("cannot extract %s: imported package %q has no resolved package name", owner, importPath)
				}
				alias = importedPackage.Name
			}
			if alias == "." {
				return fmt.Errorf("cannot extract %s: dot imports prevent proving generated result type bindings", owner)
			}
			if alias != "_" && alias == output.typeName {
				return fmt.Errorf("cannot extract %s: generated result type %q is shadowed by a file import", owner, output.typeName)
			}
		}
	}
	return nil
}
