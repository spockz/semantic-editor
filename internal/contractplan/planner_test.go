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
	pkg := fixturePackage(t, `package fixture
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
`)
	plan, err := PlanImplementContract(pkg, pkg, fixtureNamed(t, pkg, "Worker"), fixtureNamed(t, pkg, "Contract"), ReceiverPointer, BodyPolicyPanicStub, []ImportBinding{{Path: "context", Name: "ctx"}})
	if err != nil {
		t.Fatalf("PlanImplementContract returned error: %v", err)
	}
	if len(plan.Methods) != 1 || plan.Methods[0].Signature.Name != "Run" {
		t.Fatalf("planned methods = %#v, want only Run", plan.Methods)
	}
	want := "func (contractReceiver *Worker) Run(ctx ctx.Context, values ...string) (reader io.Reader, err error) { panic(`TODO: implement`) }"
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
