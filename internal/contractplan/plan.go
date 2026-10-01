// WHY: This file defines pure typed method-generation plans so runtime adapters can render and apply them against the current overlay.

package contractplan

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"
)

// BodyPolicyPanicStub marks generated methods as deliberate panic stubs.
const BodyPolicyPanicStub BodyPolicy = "panic_stub"

// BodyPolicy selects the explicit generated method body behavior.
type BodyPolicy string

// ReceiverPointer selects the pointer receiver method set.
const ReceiverPointer ReceiverKind = "pointer"

// ReceiverValue selects the value receiver method set.
const ReceiverValue ReceiverKind = "value"

// Plan contains rendered declarations and typed descriptors. An empty plan represents a verified no-op.
type Plan struct {
	Declaration string
	Methods     []PlannedMethod
	Imports     []ImportSpec
}

// PlannedMethod pairs one typed signature with its declaration source.
type PlannedMethod struct {
	Signature   MethodSignature
	Declaration string
}

// MethodSignature preserves a resolved method name, parameters, and results.
type MethodSignature struct {
	Name       string
	Parameters []ParameterSpec
	Results    []ResultSpec
}

// ResultSpec preserves one method result type and optional name.
type ResultSpec struct {
	Name string
	Type string
}

// ParameterSpec preserves one parameter type, optional name, and variadic status.
type ParameterSpec struct {
	Name     string
	Type     string
	Variadic bool
}

// ImportSpec names one package import required by generated declarations.
type ImportSpec struct {
	Path string
	Name string
}

// ImportBinding records an existing destination-file import path and declared name.
type ImportBinding struct {
	Path string
	Name string
}

// ReceiverKind selects the concrete type method set used by a plan.
type ReceiverKind string

func (signature MethodSignature) declaration() string {
	parameters := make([]string, 0, len(signature.Parameters))
	for _, parameter := range signature.Parameters {
		value := parameter.Type
		if parameter.Variadic {
			value = "..." + value
		}
		if parameter.Name != "" {
			value = parameter.Name + " " + value
		}
		parameters = append(parameters, value)
	}
	results := make([]string, 0, len(signature.Results))
	for _, result := range signature.Results {
		value := result.Type
		if result.Name != "" {
			value = result.Name + " " + value
		}
		results = append(results, value)
	}
	declaration := signature.Name + "(" + strings.Join(parameters, ", ") + ")"
	if len(results) == 1 {
		declaration += " " + results[0]
	}
	if len(results) > 1 {
		declaration += " (" + strings.Join(results, ", ") + ")"
	}
	return declaration
}

func (renderer *typeRenderer) qualifier(pkg *types.Package) string {
	if pkg == nil || pkg.Path() == renderer.current.Path() {
		return ""
	}
	if name, exists := renderer.aliases[pkg.Path()]; exists {
		return name
	}
	base := pkg.Name()
	if !token.IsIdentifier(base) || token.Lookup(base).IsKeyword() {
		base = "pkg"
	}
	name := base
	for suffix := 2; ; suffix++ {
		prior, exists := renderer.names[name]
		if !exists || prior == pkg.Path() {
			break
		}
		name = base + strconv.Itoa(suffix)
	}
	renderer.aliases[pkg.Path()] = name
	renderer.names[name] = pkg.Path()
	renderer.imports[pkg.Path()] = name
	return name
}

func (renderer *typeRenderer) renderType(value types.Type) string {
	return types.TypeString(value, renderer.qualifier)
}

func (renderer *typeRenderer) importSpecs() []ImportSpec {
	paths := make([]string, 0, len(renderer.imports))
	for importPath := range renderer.imports {
		paths = append(paths, importPath)
	}
	sort.Strings(paths)
	specs := make([]ImportSpec, 0, len(paths))
	for _, importPath := range paths {
		specs = append(specs, ImportSpec{Path: importPath, Name: renderer.imports[importPath]})
	}
	return specs
}

func (renderer *typeRenderer) signature(method *types.Func) (MethodSignature, error) {
	if method == nil {
		return MethodSignature{}, fmt.Errorf("method object is required")
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return MethodSignature{}, fmt.Errorf("method %q does not have a function signature", method.Name())
	}
	if signature.TypeParams() != nil && signature.TypeParams().Len() != 0 {
		return MethodSignature{}, fmt.Errorf("generic method %q is unsupported", method.Name())
	}
	result := MethodSignature{Name: method.Name(), Parameters: make([]ParameterSpec, 0, signature.Params().Len()), Results: make([]ResultSpec, 0, signature.Results().Len())}
	for index := range signature.Params().Len() {
		variable := signature.Params().At(index)
		parameter := ParameterSpec{Name: variable.Name(), Type: renderer.renderType(variable.Type())}
		if signature.Variadic() && index == signature.Params().Len()-1 {
			slice, ok := types.Unalias(variable.Type()).(*types.Slice)
			if !ok {
				return MethodSignature{}, fmt.Errorf("variadic parameter %q on method %q is not a slice", variable.Name(), method.Name())
			}
			parameter.Type = renderer.renderType(slice.Elem())
			parameter.Variadic = true
		}
		result.Parameters = append(result.Parameters, parameter)
	}
	for variable := range signature.Results().Variables() {
		result.Results = append(result.Results, ResultSpec{Name: variable.Name(), Type: renderer.renderType(variable.Type())})
	}
	return result, nil
}

type typeRenderer struct {
	current *types.Package
	aliases map[string]string
	names   map[string]string
	imports map[string]string
}

func newTypeRenderer(current *types.Package, bindings []ImportBinding) (*typeRenderer, error) {
	if current == nil {
		return nil, fmt.Errorf("current package is required")
	}
	renderer := &typeRenderer{
		current: current,
		aliases: make(map[string]string),
		names:   make(map[string]string),
		imports: make(map[string]string),
	}
	renderer.names[current.Name()] = current.Path()
	for _, binding := range bindings {
		if binding.Path == "" {
			return nil, fmt.Errorf("import path is empty")
		}
		if binding.Name == "" {
			return nil, fmt.Errorf("import name is required for path %q", binding.Path)
		}
		name := binding.Name
		if !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() || name == "_" || name == "." {
			return nil, fmt.Errorf("invalid import name %q for %q", name, binding.Path)
		}
		if prior, exists := renderer.names[name]; exists && prior != binding.Path {
			return nil, fmt.Errorf("import name %q collides between %q and %q", name, prior, binding.Path)
		}
		if prior, exists := renderer.aliases[binding.Path]; exists && prior != name {
			return nil, fmt.Errorf("import path %q has conflicting names %q and %q", binding.Path, prior, name)
		}
		renderer.aliases[binding.Path] = name
		renderer.names[name] = binding.Path
	}
	return renderer, nil
}

func validateIdentifier(value string, role string) error {
	if !token.IsIdentifier(value) || token.Lookup(value).IsKeyword() {
		return fmt.Errorf("%s %q is not a Go identifier", role, value)
	}
	return nil
}

func concreteReceiverType(concrete *types.Named, kind ReceiverKind) (types.Type, error) {
	if concrete == nil {
		return nil, fmt.Errorf("concrete named type is required")
	}
	if kind == ReceiverValue {
		return concrete, nil
	}
	if kind == ReceiverPointer {
		return types.NewPointer(concrete), nil
	}
	return nil, fmt.Errorf("receiver_kind %q must be value or pointer", kind)
}
