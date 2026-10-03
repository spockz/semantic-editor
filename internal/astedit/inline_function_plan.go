// WHY: Plans safe inline-function transformations without publishing intermediate source.

package astedit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
)

// InlineFunctionPlan identifies the exact caller expression a safe inline can replace.
type InlineFunctionPlan struct {
	Start       int
	End         int
	Replacement string
}

type inlineFunctionSources struct {
	caller       *ast.File
	callerSet    *token.FileSet
	callee       *ast.File
	callerSource string
}

type inlineCallee struct {
	literal    *ast.BasicLit
	resultType string
}

// PlanInlineFunction plans a direct zero-argument return inline and preserves its declared built-in result type.
func PlanInlineFunction(callerSource, calleeSource, calleeName, ownerName, discriminator string) (InlineFunctionPlan, error) {
	sources, err := parseInlineFunctionSources(callerSource, calleeSource)
	if err != nil {
		return InlineFunctionPlan{}, err
	}
	callee, err := resolveInlineCallee(sources.callee, calleeName)
	if err != nil {
		return InlineFunctionPlan{}, err
	}
	owner, err := resolveInlineOwner(sources.caller, ownerName)
	if err != nil {
		return InlineFunctionPlan{}, err
	}
	if inlineDeclares(owner, calleeName) || inlineDeclares(owner, callee.resultType) || inlineImportShadows(sources.caller, calleeName) || inlineImportShadows(sources.caller, callee.resultType) {
		return InlineFunctionPlan{}, fmt.Errorf("callee or result type is shadowed at the call site")
	}
	call, err := resolveInlineCall(sources, owner, calleeName, discriminator)
	if err != nil {
		return InlineFunctionPlan{}, err
	}
	start := sources.callerSet.Position(call.Pos()).Offset
	end := sources.callerSet.Position(call.End()).Offset
	return InlineFunctionPlan{Start: start, End: end, Replacement: callee.resultType + "(" + callee.literal.Value + ")"}, nil
}

func parseInlineFunctionSources(callerSource, calleeSource string) (inlineFunctionSources, error) {
	callerSet := token.NewFileSet()
	caller, err := parser.ParseFile(callerSet, "caller.go", callerSource, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return inlineFunctionSources{}, fmt.Errorf("parse caller: %w", err)
	}
	callee, err := parser.ParseFile(token.NewFileSet(), "callee.go", calleeSource, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return inlineFunctionSources{}, fmt.Errorf("parse callee: %w", err)
	}
	if caller.Name.Name != callee.Name.Name {
		return inlineFunctionSources{}, fmt.Errorf("inline function requires caller and callee in the same package")
	}
	return inlineFunctionSources{caller: caller, callerSet: callerSet, callee: callee, callerSource: callerSource}, nil
}

func resolveInlineCallee(file *ast.File, calleeName string) (inlineCallee, error) {
	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != calleeName {
			continue
		}
		if target != nil {
			return inlineCallee{}, fmt.Errorf("callee %q is ambiguous", calleeName)
		}
		target = function
	}
	if target == nil || target.Body == nil {
		return inlineCallee{}, fmt.Errorf("callee %q is not a function with a body", calleeName)
	}
	if target.Recv != nil || target.Type.TypeParams != nil && len(target.Type.TypeParams.List) != 0 || target.Type.Params.NumFields() != 0 || target.Type.Results.NumFields() != 1 || len(target.Type.Results.List[0].Names) != 0 {
		return inlineCallee{}, fmt.Errorf("callee must be a non-generic, non-method, zero-argument function with one unnamed result")
	}
	if len(target.Body.List) != 1 {
		return inlineCallee{}, fmt.Errorf("callee body must contain exactly one return statement")
	}
	returned, ok := target.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return inlineCallee{}, fmt.Errorf("callee body must return one value directly")
	}
	literal, ok := returned.Results[0].(*ast.BasicLit)
	if !ok {
		return inlineCallee{}, fmt.Errorf("callee return expression must be a literal to preserve lexical binding and evaluation")
	}
	resultType, ok := target.Type.Results.List[0].Type.(*ast.Ident)
	if !ok || !inlineLiteralMatchesType(literal, resultType.Name) || inlineImportShadows(file, resultType.Name) {
		return inlineCallee{}, fmt.Errorf("callee result must be a supported built-in type compatible with its literal return")
	}
	return inlineCallee{literal: literal, resultType: resultType.Name}, nil
}

func resolveInlineOwner(file *ast.File, ownerName string) (*ast.FuncDecl, error) {
	var owner *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == ownerName && function.Recv == nil {
			if owner != nil {
				return nil, fmt.Errorf("caller %q is ambiguous", ownerName)
			}
			owner = function
		}
	}
	if owner == nil || owner.Body == nil {
		return nil, fmt.Errorf("caller %q is not a function with a body", ownerName)
	}
	return owner, nil
}

func resolveInlineCall(sources inlineFunctionSources, owner *ast.FuncDecl, calleeName, discriminator string) (*ast.CallExpr, error) {
	if inlineMatchingCallCount(owner, sources.callerSet, sources.callerSource, calleeName, discriminator) != 1 {
		return nil, fmt.Errorf("call-site discriminator must identify exactly one matching call in caller %q", owner.Name.Name)
	}
	var call *ast.CallExpr
	for _, statement := range owner.Body.List {
		returned, ok := statement.(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			continue
		}
		candidate, ok := returned.Results[0].(*ast.CallExpr)
		if !ok || len(candidate.Args) != 0 {
			continue
		}
		identifier, ok := candidate.Fun.(*ast.Ident)
		if !ok || identifier.Name != calleeName {
			continue
		}
		start := sources.callerSet.Position(candidate.Pos()).Offset
		end := sources.callerSet.Position(candidate.End()).Offset
		if sources.callerSource[start:end] != discriminator {
			continue
		}
		call = candidate
	}
	if call == nil {
		return nil, fmt.Errorf("the selected call must be the entire expression of a single-value return")
	}
	return call, nil
}

func inlineDeclares(function *ast.FuncDecl, name string) bool {
	if inlineFieldListsDeclare(function, name) {
		return true
	}
	shadowed := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if inlineNodeDeclares(node, name) {
			shadowed = true
			return false
		}
		return true
	})
	return shadowed
}

func inlineFieldListsDeclare(function *ast.FuncDecl, name string) bool {
	for _, list := range []*ast.FieldList{function.Recv, function.Type.Params, function.Type.Results} {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			if inlineIdentifiersDeclare(field.Names, name) {
				return true
			}
		}
	}
	return false
}

func inlineNodeDeclares(node ast.Node, name string) bool {
	switch declaration := node.(type) {
	case *ast.AssignStmt:
		return declaration.Tok == token.DEFINE && inlineExpressionNamesDeclare(declaration.Lhs, name)
	case *ast.ValueSpec:
		return inlineIdentifiersDeclare(declaration.Names, name)
	case *ast.TypeSpec:
		return declaration.Name.Name == name
	case *ast.RangeStmt:
		return declaration.Tok == token.DEFINE && inlineExpressionNamesDeclare([]ast.Expr{declaration.Key, declaration.Value}, name)
	default:
		return false
	}
}

func inlineExpressionNamesDeclare(expressions []ast.Expr, name string) bool {
	for _, expression := range expressions {
		identifier, ok := expression.(*ast.Ident)
		if ok && identifier.Name == name {
			return true
		}
	}
	return false
}

func inlineIdentifiersDeclare(identifiers []*ast.Ident, name string) bool {
	for _, identifier := range identifiers {
		if identifier.Name == name {
			return true
		}
	}
	return false
}

func inlineImportShadows(file *ast.File, name string) bool {
	for _, imported := range file.Imports {
		if imported.Name == nil {
			return true
		}
		if imported.Name.Name == name {
			return true
		}
	}
	return false
}

func inlineLiteralMatchesType(literal *ast.BasicLit, resultType string) bool {
	object := types.Universe.Lookup(resultType)
	typeName, ok := object.(*types.TypeName)
	if !ok {
		return false
	}
	basic, ok := typeName.Type().Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch literal.Kind {
	case token.INT:
		return basic.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0
	case token.FLOAT:
		return basic.Info()&(types.IsFloat|types.IsComplex) != 0
	case token.IMAG:
		return basic.Info()&types.IsComplex != 0
	case token.STRING:
		return basic.Info()&types.IsString != 0
	default:
		return false
	}
}

func inlineMatchingCallCount(owner *ast.FuncDecl, fset *token.FileSet, source, calleeName, discriminator string) int {
	count := 0
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return true
		}
		callee, ok := call.Fun.(*ast.Ident)
		if !ok || callee.Name != calleeName {
			return true
		}
		start := fset.Position(call.Pos()).Offset
		end := fset.Position(call.End()).Offset
		if source[start:end] == discriminator {
			count++
		}
		return true
	})
	return count
}
