// WHY: Defines pure constant extraction planning from resolved Go syntax and type facts.

package astedit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"strconv"
)

// ConstantExtractionPlan carries one declaration and one exact expression replacement.
type ConstantExtractionPlan struct {
	Declaration string
	Replacement string
	Start       int
	End         int
	Type        string
	Value       string
}

// PlanConstantExtraction classifies resolved constant syntax without mutating source.
func PlanConstantExtraction(source []byte, fset *token.FileSet, file *ast.File, info *types.Info, scope *types.Scope, expr ast.Expr, name string) (ConstantExtractionPlan, error) {
	if err := validateConstantExtractionFacts(fset, file, info, scope, expr, name); err != nil {
		return ConstantExtractionPlan{}, err
	}
	owner, err := constantExpressionOwner(file, expr)
	if err != nil {
		return ConstantExtractionPlan{}, err
	}
	if scopeErr := validateConstantNameScope(file, info, scope, owner, name); scopeErr != nil {
		return ConstantExtractionPlan{}, scopeErr
	}
	details, err := inspectConstantExpression(source, fset, file, info, expr)
	if err != nil {
		return ConstantExtractionPlan{}, err
	}
	declaration := constantDeclaration(name, details)
	return ConstantExtractionPlan{
		Declaration: declaration, Replacement: name, Start: details.start, End: details.end,
		Type: details.basicType, Value: details.value,
	}, nil
}

func validateConstantExtractionFacts(fset *token.FileSet, file *ast.File, info *types.Info, scope *types.Scope, expr ast.Expr, name string) error {
	if !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() || name == "_" {
		return fmt.Errorf("symbol %q is not a valid constant identifier", name)
	}
	if fset == nil || file == nil || info == nil || scope == nil || expr == nil {
		return fmt.Errorf("constant extraction requires syntax, type, and package scope facts")
	}
	return nil
}

func constantExpressionOwner(file *ast.File, expr ast.Expr) (*ast.FuncDecl, error) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Body != nil && function.Body.Pos() <= expr.Pos() && expr.End() <= function.Body.End() {
			return function, nil
		}
	}
	return nil, fmt.Errorf("selected constant expression is not inside a function or method")
}

func validateConstantNameScope(file *ast.File, info *types.Info, scope *types.Scope, owner *ast.FuncDecl, name string) error {
	if scope.Lookup(name) != nil || types.Universe.Lookup(name) != nil {
		return fmt.Errorf("constant name %q collides with a package or universe binding", name)
	}
	for _, fields := range []*ast.FieldList{owner.Recv, owner.Type.Params, owner.Type.Results} {
		if fields == nil {
			continue
		}
		for _, field := range fields.List {
			for _, declared := range field.Names {
				if declared.Name == name {
					return fmt.Errorf("constant name %q is shadowed at the selected expression", name)
				}
			}
		}
	}
	for identifier, object := range info.Defs {
		if object != nil && identifier.Name == name && owner.Body.Pos() <= object.Pos() && object.Pos() <= owner.Body.End() {
			return fmt.Errorf("constant name %q is declared in the selected function or method", name)
		}
	}
	for _, imported := range file.Imports {
		if imported.Name != nil && imported.Name.Name == name {
			return fmt.Errorf("constant name %q collides with a file import", name)
		}
		if imported.Name == nil {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("read import path while checking constant scope: %w", err)
			}
			if path.Base(importPath) == name {
				return fmt.Errorf("constant name %q collides with an import name", name)
			}
		}
	}
	return nil
}

type constantExpressionDetails struct {
	start      int
	end        int
	expression string
	basicType  string
	typeName   string
	value      string
}

func inspectConstantExpression(source []byte, fset *token.FileSet, file *ast.File, info *types.Info, expr ast.Expr) (constantExpressionDetails, error) {
	start, end := fset.Position(expr.Pos()).Offset, fset.Position(expr.End()).Offset
	if start < 0 || end < start || end > len(source) {
		return constantExpressionDetails{}, fmt.Errorf("constant expression source range is inconsistent")
	}
	typed, ok := info.Types[expr]
	if !ok || typed.Value == nil {
		return constantExpressionDetails{}, fmt.Errorf("selected expression is not a resolved Go constant expression")
	}
	if !constantExtractionSyntax(expr, info) {
		return constantExpressionDetails{}, fmt.Errorf("selected expression contains unsupported or context-dependent constant syntax")
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			cs, ce := fset.Position(comment.Pos()).Offset, fset.Position(comment.End()).Offset
			if cs < end && ce > start {
				return constantExpressionDetails{}, fmt.Errorf("selected constant expression contains a comment or directive")
			}
		}
	}
	basic, isBasic := typed.Type.(*types.Basic)
	if !isBasic {
		return constantExpressionDetails{}, fmt.Errorf("named and non-basic constant types are not supported")
	}
	typeName := ""
	if basic.Info()&types.IsUntyped == 0 {
		typeName = basic.Name()
	}
	return constantExpressionDetails{
		start: start, end: end, expression: string(source[start:end]),
		basicType: basic.Name(), typeName: typeName, value: typed.Value.ExactString(),
	}, nil
}

func constantDeclaration(name string, details constantExpressionDetails) string {
	declaration := "const " + name
	if details.typeName != "" {
		declaration += " " + details.typeName
	}
	return declaration + " = " + details.expression
}

func constantExtractionSyntax(expr ast.Expr, info *types.Info) bool {
	switch node := expr.(type) {
	case *ast.BasicLit:
		switch node.Kind {
		case token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING:
			return true
		default:
			return false
		}
	case *ast.Ident:
		if node.Name == "iota" {
			return false
		}
		object, ok := info.Uses[node].(*types.Const)
		if !ok {
			return false
		}
		if object.Parent() == types.Universe {
			return true
		}
		return object.Pkg() != nil && object.Parent() == object.Pkg().Scope()
	case *ast.ParenExpr:
		return constantExtractionSyntax(node.X, info)
	case *ast.UnaryExpr:
		switch node.Op {
		case token.ADD, token.SUB, token.XOR, token.NOT:
			return constantExtractionSyntax(node.X, info)
		default:
			return false
		}
	case *ast.BinaryExpr:
		switch node.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.AND, token.OR, token.XOR, token.AND_NOT, token.SHL, token.SHR:
			return constantExtractionSyntax(node.X, info) && constantExtractionSyntax(node.Y, info)
		default:
			return false
		}
	case *ast.SelectorExpr:
		object, ok := info.Uses[node.Sel].(*types.Const)
		return ok && object.Pkg() != nil
	default:
		return false
	}
}
