// WHY: Proves type migration accepts alias identity and refuses distinct named types.

package astedit

import (
	"go/token"
	"go/types"
	"testing"
)

func TestPlanAliasTypeMigrationUsesIdentity(t *testing.T) {
	pkg := types.NewPackage("example.test/p", "p")
	aliasObject := types.NewTypeName(token.NoPos, pkg, "Count", nil)
	alias := types.NewAlias(aliasObject, types.Typ[types.Int])
	proof, err := PlanAliasTypeMigration("int", "Count", types.Typ[types.Int], alias)
	if err != nil {
		t.Fatalf("PlanAliasTypeMigration() error = %v", err)
	}
	if proof.Current != "int" || proof.Desired != "Count" {
		t.Fatalf("proof = %#v", proof)
	}
	if _, err := PlanAliasTypeMigration("int", "Count", types.Typ[types.Int], nil); err == nil {
		t.Fatal("nil type proof was accepted")
	}
}

func TestPlanAliasTypeMigrationRefusesDistinctNamedType(t *testing.T) {
	pkg := types.NewPackage("example.test/p", "p")
	namedObject := types.NewTypeName(token.NoPos, pkg, "Count", nil)
	named := types.NewNamed(namedObject, types.Typ[types.Int], nil)
	if _, err := PlanAliasTypeMigration("int", "Count", types.Typ[types.Int], named); err == nil {
		t.Fatal("distinct named type was accepted")
	}
}
