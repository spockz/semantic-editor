// WHY: These tests prove planner outputs and refusals from go/types package objects without involving runtime publishing.

package contractplan

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestPlanExtractInterfacePreservesResolvedSignatureAndImportAliases(t *testing.T) {
	pkg := fixturePackage(t, `package fixture
import (
	"context"
	"io"
)
type Store struct{}
func (*Store) Fetch(ctx context.Context, values ...string) (reader io.Reader, err error) { return nil, nil }
`)
	store := fixtureNamed(t, pkg, "Store")
	plan, err := PlanExtractInterface(pkg, pkg, store, ReceiverPointer, []*types.Func{fixtureMethod(t, store, "Fetch")}, "StoreAPI", []ImportBinding{{Path: "context", Name: "ctx"}, {Path: "example.com/legacy", Name: "io"}})
	if err != nil {
		t.Fatalf("PlanExtractInterface returned error: %v", err)
	}
	want := "type StoreAPI interface {\n\tFetch(ctx ctx.Context, values ...string) (reader io2.Reader, err error)\n}"
	if plan.Declaration != want {
		t.Fatalf("declaration = %q, want %q", plan.Declaration, want)
	}
	if len(plan.Imports) != 1 || plan.Imports[0] != (ImportSpec{Path: "io", Name: "io2"}) {
		t.Fatalf("imports = %#v, want io as io2", plan.Imports)
	}
	if len(plan.Methods) != 1 || plan.Methods[0].Signature.Name != "Fetch" {
		t.Fatalf("typed methods = %#v, want Fetch signature", plan.Methods)
	}
	signature := plan.Methods[0].Signature
	if len(signature.Parameters) != 2 || signature.Parameters[1] != (ParameterSpec{Name: "values", Type: "string", Variadic: true}) {
		t.Fatalf("parameters = %#v, want named variadic values parameter", signature.Parameters)
	}
	if len(signature.Results) != 2 || signature.Results[0] != (ResultSpec{Name: "reader", Type: "io2.Reader"}) {
		t.Fatalf("results = %#v, want named reader and error results", signature.Results)
	}
}

func TestPlanExtractInterfaceRejectsPointerValueMismatchAndPromotedMethods(t *testing.T) {
	pkg := fixturePackage(t, `package fixture
type Embedded struct{}
func (*Embedded) Flush() error { return nil }
type Store struct { Embedded }
`)
	store := fixtureNamed(t, pkg, "Store")
	embeddedFlush := fixtureMethod(t, fixtureNamed(t, pkg, "Embedded"), "Flush")
	if _, err := PlanExtractInterface(pkg, pkg, store, ReceiverValue, []*types.Func{embeddedFlush}, "StoreAPI", nil); err == nil || !strings.Contains(err.Error(), "requested value receiver") {
		t.Fatalf("value receiver mismatch error = %v", err)
	}
	if _, err := PlanExtractInterface(pkg, pkg, store, ReceiverPointer, []*types.Func{embeddedFlush}, "StoreAPI", nil); err == nil || !strings.Contains(err.Error(), "promoted method") {
		t.Fatalf("promoted method error = %v", err)
	}
}

func TestPlanImplementContractGeneratesOnlyMissingPanicStub(t *testing.T) {
	source := `package fixture
import (
	"context"
	"io"
)
type Base interface { Close() error }
type Contract interface {
	Base
	Run(ctx context.Context, values ...string) (reader io.Reader, err error)
}
type Worker struct{}
func (Worker) Close() error { return nil }
`
	pkg := fixturePackage(t, source)
	plan, err := PlanImplementContract(pkg, pkg, fixtureNamed(t, pkg, "Worker"), fixtureNamed(t, pkg, "Contract"), ReceiverPointer, BodyPolicyPanicStub, []ImportBinding{{Path: "context", Name: "ctx"}})
	if err != nil {
		t.Fatalf("PlanImplementContract returned error: %v", err)
	}
	if len(plan.Methods) != 1 || plan.Methods[0].Signature.Name != "Run" {
		t.Fatalf("planned methods = %#v, want only Run", plan.Methods)
	}
	want := "func (*Worker) Run(ctx.Context, ...string) (io.Reader, error) { panic(`TODO: implement`) }"
	if plan.Methods[0].Declaration != want {
		t.Fatalf("method = %q, want %q", plan.Methods[0].Declaration, want)
	}
	if len(plan.Imports) != 1 || plan.Imports[0] != (ImportSpec{Path: "io", Name: "io"}) {
		t.Fatalf("imports = %#v, want io import", plan.Imports)
	}
}

func TestPlanImplementContractRefusesIncompatibleAndWrongReceiverMethods(t *testing.T) {
	incompatible := fixturePackage(t, `package fixture
type Contract interface { Run(int) error }
type Worker struct{}
func (*Worker) Run(string) error { return nil }
`)
	worker := fixtureNamed(t, incompatible, "Worker")
	contract := fixtureNamed(t, incompatible, "Contract")
	if _, err := PlanImplementContract(incompatible, incompatible, worker, contract, ReceiverPointer, BodyPolicyPanicStub, nil); err == nil || !strings.Contains(err.Error(), "incompatible signature") {
		t.Fatalf("incompatible method error = %v", err)
	}
	receiverMismatch := fixturePackage(t, `package fixture
type Contract interface { Run(int) error }
type Worker struct{}
func (*Worker) Run(int) error { return nil }
`)
	worker = fixtureNamed(t, receiverMismatch, "Worker")
	contract = fixtureNamed(t, receiverMismatch, "Contract")
	if plan, err := PlanImplementContract(receiverMismatch, receiverMismatch, worker, contract, ReceiverPointer, BodyPolicyPanicStub, nil); err != nil || len(plan.Methods) != 0 {
		t.Fatalf("pointer receiver no-op = %#v, %v", plan, err)
	}
	if _, err := PlanImplementContract(receiverMismatch, receiverMismatch, worker, contract, ReceiverValue, BodyPolicyPanicStub, nil); err == nil || !strings.Contains(err.Error(), "unavailable on the requested value receiver") {
		t.Fatalf("receiver mismatch error = %v", err)
	}
}

func TestPlanImplementContractRefusesAmbiguousPromotedMethods(t *testing.T) {
	pkg := fixturePackage(t, `package fixture
type Left struct{}
func (Left) Run() error { return nil }
type Right struct{}
func (Right) Run() error { return nil }
type Contract interface { Run() error }
type Worker struct { Left; Right }
`)
	_, err := PlanImplementContract(pkg, pkg, fixtureNamed(t, pkg, "Worker"), fixtureNamed(t, pkg, "Contract"), ReceiverValue, BodyPolicyPanicStub, nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous method error = %v", err)
	}
}

func TestPlanRefusesGenericTargetsAndUnknownBodyPolicy(t *testing.T) {
	generic := fixturePackage(t, `package fixture
type Contract interface { Run() }
type Worker[T any] struct{}
func (Worker[T]) Run() {}
`)
	if _, err := PlanImplementContract(generic, generic, fixtureNamed(t, generic, "Worker"), fixtureNamed(t, generic, "Contract"), ReceiverValue, BodyPolicyPanicStub, nil); err == nil || !strings.Contains(err.Error(), "generic concrete type") {
		t.Fatalf("generic target error = %v", err)
	}
	plain := fixturePackage(t, `package fixture
type Contract interface { Run() }
type Worker struct{}
`)
	if _, err := PlanImplementContract(plain, plain, fixtureNamed(t, plain, "Worker"), fixtureNamed(t, plain, "Contract"), ReceiverPointer, BodyPolicy("zero"), nil); err == nil || !strings.Contains(err.Error(), "body_policy") {
		t.Fatalf("unknown body policy error = %v", err)
	}
	otherPackage := types.NewPackage(plain.Path(), plain.Name())
	if _, err := PlanImplementContract(plain, otherPackage, fixtureNamed(t, plain, "Worker"), fixtureNamed(t, plain, "Contract"), ReceiverPointer, BodyPolicyPanicStub, nil); err == nil || !strings.Contains(err.Error(), "exact concrete package") {
		t.Fatalf("package identity error = %v", err)
	}
}

func TestPlanExtractInterfaceReturnsEmptyPlanForExactExistingInterface(t *testing.T) {
	pkg := fixturePackage(t, `package fixture
type Store struct{}
func (*Store) Read() error { return nil }
type StoreAPI interface { Read() error }
`)
	store := fixtureNamed(t, pkg, "Store")
	plan, err := PlanExtractInterface(pkg, pkg, store, ReceiverPointer, []*types.Func{fixtureMethod(t, store, "Read")}, "StoreAPI", nil)
	if err != nil {
		t.Fatalf("exact existing interface was not accepted as a no-op: %v", err)
	}
	if plan.Declaration != "" || len(plan.Methods) != 0 || len(plan.Imports) != 0 {
		t.Fatalf("no-op plan = %#v, want empty write plan", plan)
	}
}

func TestGeneratedSingleNamedResultAndShadowSafeStubTypeCheck(t *testing.T) {
	source := `package fixture
type Contract interface {
	Read() (value int)
	Run(panic int) (len string)
}
type Worker struct{}
`
	pkg := fixturePackage(t, source)
	worker := fixtureNamed(t, pkg, "Worker")
	contract := fixtureNamed(t, pkg, "Contract")
	stub, err := PlanImplementContract(pkg, pkg, worker, contract, ReceiverPointer, BodyPolicyPanicStub, nil)
	if err != nil {
		t.Fatalf("PlanImplementContract returned error: %v", err)
	}
	typeCheckGenerated(t, source, "", stub.Declaration)
	if strings.Contains(stub.Declaration, "panic int") || strings.Contains(stub.Declaration, "len string") {
		t.Fatalf("stub copied shadowing argument/result names: %s", stub.Declaration)
	}
	storeSource := `package fixture
type Store struct{}
func (Store) Read() (value int) { return 1 }
`
	storePackage := fixturePackage(t, storeSource)
	store := fixtureNamed(t, storePackage, "Store")
	extracted, err := PlanExtractInterface(storePackage, storePackage, store, ReceiverValue, []*types.Func{fixtureMethod(t, store, "Read")}, "StoreReader", nil)
	if err != nil {
		t.Fatalf("PlanExtractInterface returned error: %v", err)
	}
	typeCheckGenerated(t, storeSource, "", extracted.Declaration)
	if !strings.Contains(extracted.Declaration, "Read() (value int)") {
		t.Fatalf("single named result lost valid parenthesized form: %s", extracted.Declaration)
	}
}

func TestExtractInterfaceReservesPackageAndNewTypeNamesForImportAliases(t *testing.T) {
	source := `package fixture
import (
	ctx "context"
	encjson "encoding/json"
)
var json int
type Store struct{}
func (Store) Read(ctx.Context, encjson.RawMessage) {}
`
	pkg := fixturePackage(t, source)
	store := fixtureNamed(t, pkg, "Store")
	plan, err := PlanExtractInterface(pkg, pkg, store, ReceiverValue, []*types.Func{fixtureMethod(t, store, "Read")}, "context", nil)
	if err != nil {
		t.Fatalf("PlanExtractInterface returned error: %v", err)
	}
	if len(plan.Imports) != 2 || plan.Imports[0] != (ImportSpec{Path: "context", Name: "context2"}) || plan.Imports[1] != (ImportSpec{Path: "encoding/json", Name: "json2"}) {
		t.Fatalf("imports = %#v, want collision-free context2 and json2 aliases", plan.Imports)
	}
	generatedImports := "import (\n\tcontext2 \"context\"\n\tjson2 \"encoding/json\"\n)\n"
	typeCheckGenerated(t, source, generatedImports, plan.Declaration)
}

func TestExtractInterfaceRejectsBlankTargetAndConstraintNoOp(t *testing.T) {
	source := `package fixture
type Store int
func (Store) Run() {}
type StoreAPI interface { ~int; Run() }
`
	pkg := fixturePackage(t, source)
	store := fixtureNamed(t, pkg, "Store")
	method := fixtureMethod(t, store, "Run")
	if plan, err := PlanExtractInterface(pkg, pkg, store, ReceiverValue, []*types.Func{method}, "_", nil); err == nil || plan.Declaration != "" {
		t.Fatalf("blank target yielded plan=%#v err=%v", plan, err)
	}
	if plan, err := PlanExtractInterface(pkg, pkg, store, ReceiverValue, []*types.Func{method}, "StoreAPI", nil); err == nil || plan.Declaration != "" {
		t.Fatalf("constraint no-op yielded plan=%#v err=%v", plan, err)
	}
}

func TestImplementContractRejectsInvalidReceiverBasesWithoutPlan(t *testing.T) {
	for _, source := range []string{
		`package fixture
type Contract interface { Run() }
type PointerBase *int
`,
		`package fixture
type Contract interface { Run() }
type InterfaceBase interface { Other() }
`,
	} {
		pkg := fixturePackage(t, source)
		for _, name := range []string{"PointerBase", "InterfaceBase"} {
			object := pkg.Scope().Lookup(name)
			if object == nil {
				continue
			}
			concrete, ok := object.Type().(*types.Named)
			if !ok {
				t.Fatalf("receiver base %q has type %T", name, object.Type())
			}
			plan, err := PlanImplementContract(pkg, pkg, concrete, fixtureNamed(t, pkg, "Contract"), ReceiverPointer, BodyPolicyPanicStub, nil)
			if err == nil || plan.Declaration != "" || len(plan.Methods) != 0 {
				t.Fatalf("receiver base %q yielded plan=%#v err=%v", name, plan, err)
			}
		}
	}
}

func TestPlanImplementContractRefusesShadowedBuiltinPanic(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		imports []ImportBinding
	}{
		{
			name: "package declaration",
			source: `package fixture
type Contract interface { Run() }
type Worker struct{}
func panic(any) {}
`,
		},
		{
			name: "file import",
			source: `package fixture
import panic "context"
var _ = panic.Background
type Contract interface { Run() }
type Worker struct{}
`,
			imports: []ImportBinding{{Path: "context", Name: "panic"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := fixturePackage(t, test.source)
			plan, err := PlanImplementContract(pkg, pkg, fixtureNamed(t, pkg, "Worker"), fixtureNamed(t, pkg, "Contract"), ReceiverPointer, BodyPolicyPanicStub, test.imports)
			if err == nil || plan.Declaration != "" || len(plan.Methods) != 0 {
				t.Fatalf("shadowed builtin panic yielded plan=%#v err=%v", plan, err)
			}
			if !strings.Contains(err.Error(), "panic_stub would resolve panic") {
				t.Fatalf("shadow refusal = %v", err)
			}
		})
	}
}

func (fixture contractPlannerFixtureImporter) Import(path string) (*types.Package, error) {
	if pkg := fixture.packages[path]; pkg != nil {
		return pkg, nil
	}
	return fixture.fallback.Import(path)
}

func TestImplementContractReservesUniverseNamesForGeneratedImports(t *testing.T) {
	source := `package fixture
import (
	panicpkg "example.com/panicpkg"
	intpkg "example.com/intpkg"
)
type Contract interface { Run(panicpkg.Value, intpkg.Value, int, any) }
type Worker struct{}
`
	newPackage := func(path, name string) *types.Package {
		pkg := types.NewPackage(path, name)
		object := types.NewTypeName(token.NoPos, pkg, "Value", nil)
		types.NewNamed(object, types.NewStruct(nil, nil), nil)
		pkg.Scope().Insert(object)
		pkg.MarkComplete()
		return pkg
	}
	fixtureImporter := contractPlannerFixtureImporter{
		packages: map[string]*types.Package{
			"example.com/panicpkg": newPackage("example.com/panicpkg", "panic"),
			"example.com/intpkg":   newPackage("example.com/intpkg", "int"),
		},
		fallback: importer.Default(),
	}
	pkg := fixturePackageWithImporter(t, source, fixtureImporter)
	plan, err := PlanImplementContract(pkg, pkg, fixtureNamed(t, pkg, "Worker"), fixtureNamed(t, pkg, "Contract"), ReceiverPointer, BodyPolicyPanicStub, nil)
	if err != nil {
		t.Fatalf("PlanImplementContract returned error: %v", err)
	}
	if len(plan.Imports) != 2 || plan.Imports[0] != (ImportSpec{Path: "example.com/intpkg", Name: "int2"}) || plan.Imports[1] != (ImportSpec{Path: "example.com/panicpkg", Name: "panic2"}) {
		t.Fatalf("generated imports = %#v, want int2 and panic2", plan.Imports)
	}
	if !strings.Contains(plan.Declaration, "panic2.Value, int2.Value, int, any") || !strings.Contains(plan.Declaration, "panic(`TODO: implement`)") {
		t.Fatalf("generated declaration changed foreign or universe bindings: %s", plan.Declaration)
	}
	generatedImports := "import (\n\tint2 \"example.com/intpkg\"\n\tpanic2 \"example.com/panicpkg\"\n)\n"
	typeCheckGeneratedWithImporter(t, source, generatedImports, plan.Declaration, fixtureImporter)
}

type contractPlannerFixtureImporter struct {
	packages map[string]*types.Package
	fallback types.Importer
}

func fixturePackage(t *testing.T, source string) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	configuration := types.Config{Importer: importer.Default()}
	pkg, err := configuration.Check("example.com/fixture", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("type-check fixture: %v", err)
	}
	return pkg
}

func fixtureNamed(t *testing.T, pkg *types.Package, name string) *types.Named {
	t.Helper()
	object := pkg.Scope().Lookup(name)
	if object == nil {
		t.Fatalf("fixture has no object %q", name)
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		t.Fatalf("fixture object %q has type %T", name, object.Type())
	}
	return named
}

func fixtureMethod(t *testing.T, named *types.Named, name string) *types.Func {
	t.Helper()
	for method := range named.Methods() {
		if method.Name() == name {
			return method
		}
	}
	t.Fatalf("fixture type %q has no direct method %q", named.Obj().Name(), name)
	return nil
}

func typeCheckGenerated(t *testing.T, source, imports, declaration string) {
	typeCheckGeneratedWithImporter(t, source, imports, declaration, importer.Default())
}

func typeCheckGeneratedWithImporter(t *testing.T, source, imports, declaration string, sourceImporter types.Importer) {
	t.Helper()
	fset := token.NewFileSet()
	original, err := parser.ParseFile(fset, "fixture.go", source, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse original fixture: %v", err)
	}
	generatedSource := "package fixture\n"
	if imports != "" {
		generatedSource += imports
	}
	generatedSource += declaration
	generated, err := parser.ParseFile(fset, "generated.go", generatedSource, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse generated source: %v\n%s", err, generatedSource)
	}
	configuration := types.Config{Importer: sourceImporter}
	if _, err := configuration.Check("example.com/fixture", fset, []*ast.File{original, generated}, nil); err != nil {
		t.Fatalf("type-check generated source: %v\n%s", err, generatedSource)
	}
}

func fixturePackageWithImporter(t *testing.T, source string, sourceImporter types.Importer) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	configuration := types.Config{Importer: sourceImporter}
	pkg, err := configuration.Check("example.com/fixture", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("type-check fixture: %v", err)
	}
	return pkg
}
