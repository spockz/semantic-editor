// WHY: Proves scalar initializer inputs retain their bindings when moved into an immediate return.

package astedit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
)

type inlineComputedOperand struct {
	name              string
	typeName          string
	packagePath       string
	declarationPath   string
	declarationOffset int
	constant          bool
	constantValue     string
}

func inlineComputedOperands(expression ast.Expr, owner *ast.FuncDecl, info *types.Info, fset *token.FileSet) ([]inlineComputedOperand, error) {
	if !inlineComputedScalar(info.TypeOf(expression)) {
		return nil, fmt.Errorf("computed initializer must have a resolved scalar type")
	}
	operands := make([]inlineComputedOperand, 0)
	var validationErr error
	ast.Inspect(expression, func(node ast.Node) bool {
		if node == nil || validationErr != nil {
			return false
		}
		switch expression := node.(type) {
		case *ast.ParenExpr, *ast.BasicLit:
		case *ast.BinaryExpr:
			if expression.Op.Precedence() == 0 {
				validationErr = fmt.Errorf("computed initializer has an unsupported binary operator")
			}
		case *ast.UnaryExpr:
			switch expression.Op {
			case token.ADD, token.SUB, token.XOR, token.NOT:
			default:
				validationErr = fmt.Errorf("computed initializer has an effectful or unsupported unary operator")
			}
		case *ast.Ident:
			operand, err := inlineComputedInput(info.Uses[expression], owner, fset)
			if err != nil {
				validationErr = err
			} else {
				operands = append(operands, operand)
			}
		default:
			validationErr = fmt.Errorf("computed initializer must contain only scalar literals, local values, constants, and pure unary/binary operators")
		}
		return validationErr == nil
	})
	if validationErr != nil {
		return nil, validationErr
	}
	return operands, nil
}

func inlineComputedInput(object types.Object, owner *ast.FuncDecl, fset *token.FileSet) (inlineComputedOperand, error) {
	if object == nil || !inlineComputedScalar(object.Type()) {
		return inlineComputedOperand{}, fmt.Errorf("computed initializer input is not a resolved scalar value")
	}
	input := inlineComputedOperand{name: object.Name(), typeName: extractionTypeIdentity(object.Type())}
	if object.Pkg() != nil {
		input.packagePath = object.Pkg().Path()
	}
	switch object := object.(type) {
	case *types.Var:
		if object.IsField() || object.Pos() < owner.Pos() || object.Pos() >= owner.End() {
			return inlineComputedOperand{}, fmt.Errorf("computed initializer variable inputs must be owner parameters or local bindings")
		}
	case *types.Const:
		input.constant = true
		input.constantValue = object.Val().ExactString()
		if object.Pkg() == nil || object.Parent() == object.Pkg().Scope() {
			return input, nil
		}
	default:
		return inlineComputedOperand{}, fmt.Errorf("computed initializer input must resolve to a local variable or constant")
	}
	position := fset.PositionFor(object.Pos(), false)
	if position.Filename == "" || position.Offset < 0 {
		return inlineComputedOperand{}, fmt.Errorf("computed initializer input declaration is unresolved")
	}
	input.declarationPath = filepath.Clean(position.Filename)
	input.declarationOffset = position.Offset
	return input, nil
}

func inlineComputedScalar(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := value.Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) != 0
}

func validateInlineComputedCandidate(original []inlineComputedOperand, expression ast.Expr, candidate variableTypeContext) error {
	operands, err := inlineComputedOperands(expression, candidate.owner, candidate.info, candidate.fset)
	if err != nil {
		return fmt.Errorf("resolve computed inline candidate inputs: %w", err)
	}
	if !slices.Equal(original, operands) {
		return fmt.Errorf("replacement changes a computed initializer input binding, type, or constant value")
	}
	return nil
}

func inlineReturnInitializerOperands(target inlineTarget, owner *ast.FuncDecl, info *types.Info, fset *token.FileSet) ([]inlineComputedOperand, error) {
	if _, composite := unparenExpr(target.value).(*ast.CompositeLit); composite {
		return nil, validateInlineStructLiteral(target, info)
	}
	return inlineComputedOperands(target.value, owner, info, fset)
}
