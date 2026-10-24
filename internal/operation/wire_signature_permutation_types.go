// WHY: Distinguishes same-named local constant types across compiler snapshots while parameter edits shift physical offsets.

package operation

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

type signatureTypeBindings struct {
	pkg   *packages.Package
	names []string
	err   error
}

func signatureArgumentType(pkg *packages.Package, value types.Type) (string, error) {
	bindings := signatureTypeBindings{pkg: pkg}
	bindings.visit(value)
	if bindings.err != nil {
		return "", bindings.err
	}
	evidence := struct {
		Printed  string
		Bindings []string
	}{Printed: signatureResolvedType(types.Unalias(value)), Bindings: bindings.names}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return "", fmt.Errorf("encode permutation type evidence: %w", err)
	}
	return string(encoded), nil
}

func (bindings *signatureTypeBindings) visit(value types.Type) {
	if bindings.err != nil {
		return
	}
	switch current := types.Unalias(value).(type) {
	case *types.Basic:
	case *types.Named:
		bindings.typeName(current.Obj())
		for t := range current.TypeArgs().Types() {
			bindings.visit(t)
		}
	case *types.TypeParam:
		bindings.typeName(current.Obj())
	case *types.Pointer:
		bindings.visit(current.Elem())
	case *types.Array:
		bindings.visit(current.Elem())
	case *types.Slice:
		bindings.visit(current.Elem())
	case *types.Chan:
		bindings.visit(current.Elem())
	case *types.Map:
		bindings.visit(current.Key())
		bindings.visit(current.Elem())
	case *types.Tuple:
		for v := range current.Variables() {
			bindings.visit(v.Type())
		}
	case *types.Signature:
		bindings.visit(current.Params())
		bindings.visit(current.Results())
	case *types.Struct:
		for field := range current.Fields() {
			bindings.member(field)
		}
	case *types.Interface:
		for method := range current.Methods() {
			bindings.member(method)
		}
		for etyp := range current.EmbeddedTypes() {
			bindings.visit(etyp)
		}
	case *types.Union:
		for term := range current.Terms() {
			bindings.visit(term.Type())
		}
	default:
		bindings.err = fmt.Errorf("permutation argument has an unsupported resolved type %T", value)
	}
}

func (bindings *signatureTypeBindings) member(object types.Object) {
	if !object.Exported() && object.Pkg() != nil {
		bindings.names = append(bindings.names, "private:"+object.Pkg().Path())
	}
	bindings.visit(object.Type())
}

func (bindings *signatureTypeBindings) typeName(object *types.TypeName) {
	if object.Pkg() == nil {
		if types.Universe.Lookup(object.Name()) == object {
			bindings.names = append(bindings.names, "builtin."+object.Name())
			return
		}
		bindings.err = fmt.Errorf("permutation named constant type has no package identity")
		return
	}
	identity := object.Pkg().Path() + "." + object.Name()
	if object.Pkg().Scope().Lookup(object.Name()) == object {
		bindings.names = append(bindings.names, identity)
		return
	}
	origin, err := signatureLocalTypeOrigin(bindings.pkg, object)
	if err != nil {
		bindings.err = err
		return
	}
	bindings.names = append(bindings.names, identity+"@"+origin)
}

func signatureLocalTypeOrigin(pkg *packages.Package, object *types.TypeName) (string, error) {
	position := pkg.Fset.PositionFor(object.Pos(), false)
	for _, file := range pkg.Syntax {
		if pkg.Fset.PositionFor(file.Pos(), false).Filename != position.Filename {
			continue
		}
		ordinal, selected := 0, -1
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			definition, isType := pkg.TypesInfo.Defs[identifier].(*types.TypeName)
			if isType {
				if definition == object {
					selected = ordinal
				}
				ordinal++
			}
			return true
		})
		if selected >= 0 {
			path, err := filepath.Abs(position.Filename)
			if err != nil {
				return "", fmt.Errorf("resolve local permutation type: %w", err)
			}
			return fmt.Sprintf("%s#%d", filepath.Clean(path), selected), nil
		}
	}
	return "", fmt.Errorf("permutation local constant type lacks a physical declaration")
}
