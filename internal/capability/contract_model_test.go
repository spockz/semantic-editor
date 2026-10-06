// Package capability tests the closed contract model and exact fault classification.
package capability

import "testing"

func TestContractEnumsRejectUnknownValues(t *testing.T) {
	if !VariantSet(0).Valid() || VariantSet(0).String() != "default" {
		t.Fatal("zero variant selector must represent the default profile")
	}
	for _, profile := range []VariantSet{
		VariantSelectedFile | VariantNormalize,
		VariantSelectedScope | VariantCheckOnly,
		VariantSelectedFile | VariantSafeDelete,
		VariantSelectedFile | VariantBatch,
	} {
		if !profile.Valid() {
			t.Errorf("valid composite selector %q was rejected", profile)
		}
	}
	for _, profile := range []VariantSet{
		VariantSelectedFile | VariantSelectedScope,
		VariantSamePackage | VariantCrossPackage,
		VariantInferPackage | VariantExplicitPackage,
		VariantNormalize | VariantCheckOnly,
	} {
		if profile.Valid() {
			t.Errorf("contradictory selector %q was accepted", profile)
		}
	}
	if (AnalysisScope("invented")).Valid() {
		t.Fatal("unknown analysis scope was accepted")
	}
	if (Requirement{Kind: RequirementKind("free prose")}).Valid() {
		t.Fatal("unknown requirement kind was accepted")
	}
	if (VariantSet(1 << 15)).Valid() {
		t.Fatal("unknown variant bit was accepted")
	}
	if (FeatureSet(1 << 31)).Valid() {
		t.Fatal("unknown feature bit was accepted")
	}
	if (GuaranteeSet(1 << 15)).Valid() {
		t.Fatal("unknown guarantee bit was accepted")
	}
}

func TestContractRequiresHealthySourceSupport(t *testing.T) {
	contract := OperationContract{Variants: []ContractVariant{{
		AnalysisScope: ScopeSelectedFile,
		Capabilities:  FeatureLookup,
		Guarantees:    GuaranteeReadOnly,
	}}}
	if err := contract.Validate(); err == nil {
		t.Fatal("contract without explicit healthy-source support was accepted")
	}
	contract.Variants[0].ErrorTolerance.Supported = 1
	if err := contract.Validate(); err != nil {
		t.Fatalf("contract with healthy-source support is invalid: %v", err)
	}
}

func TestFaultCombinationClassificationIsExact(t *testing.T) {
	support := BrokenCodeSupport{Supported: CombinationSet(uint64(1)<<FaultTargetSyntax | uint64(1)<<FaultTargetType | uint64(1)<<AllCodeFaults)}
	for _, faults := range []CodeFaults{FaultTargetSyntax, FaultTargetType, AllCodeFaults} {
		got, err := support.Classify(faults)
		if err != nil || got != FaultSupported {
			t.Fatalf("classification for %s = %v, %v; want supported", faults, got, err)
		}
		if !support.Supported.Contains(faults) {
			t.Fatalf("exact bitmap is missing %s", faults)
		}
	}
	got, err := support.Classify(FaultTargetSyntax | FaultTargetType)
	if err != nil || got != FaultUnknown {
		t.Fatalf("combined fault classification = %v, %v; want unknown despite supported individual faults", got, err)
	}
	if support.Supported.Contains(FaultTargetSyntax | FaultTargetType) {
		t.Fatal("bitmap inferred combined support")
	}
	if _, err := support.Classify(1 << 6); err == nil {
		t.Fatal("out-of-range code fault mask was accepted")
	}
	if support.Supported.Contains(1 << 6) {
		t.Fatal("bitmap accepted an out-of-range fault mask")
	}
}

func TestBrokenCodeSupportRejectsOverlappingMasks(t *testing.T) {
	bit := CombinationSet(uint64(1) << FaultTargetSyntax)
	if err := (BrokenCodeSupport{Supported: bit, Rejected: bit}).Validate(); err == nil {
		t.Fatal("overlapping classifications were accepted")
	}
	if err := (BrokenCodeSupport{Conditional: bit}).Validate(); err == nil {
		t.Fatal("conditional classification without a condition was accepted")
	}
}

func TestVariantLookupUsesExactSelectorProfile(t *testing.T) {
	want := ContractVariant{When: VariantSelectedFile | VariantSamePackage}
	c := OperationContract{Variants: []ContractVariant{want}}
	if got, ok := c.Variant(want.When); !ok || got.When != want.When {
		t.Fatalf("exact profile lookup = %#v, %v", got, ok)
	}
	if _, ok := c.Variant(VariantSelectedFile); ok {
		t.Fatal("partial selector matched a different profile")
	}
}

func TestFaultCasesEnumerateExactMatrix(t *testing.T) {
	b := BrokenCodeSupport{
		Supported:   CombinationSet(1),
		Rejected:    CombinationSet(uint64(1) << AllCodeFaults),
		Conditional: CombinationSet(uint64(1) << FaultTargetType),
		Condition:   RecoveryDiagnostics,
	}
	cases, err := b.Cases()
	if err != nil || len(cases) != 64 {
		t.Fatalf("Cases = %d cases, %v; want 64", len(cases), err)
	}
	expected := map[CodeFaults]FaultSupport{0: FaultSupported, AllCodeFaults: FaultRejected, FaultTargetType: FaultConditional}
	for i, faultCase := range cases {
		if faultCase.Faults != CodeFaults(i) {
			t.Fatalf("case %d has faults %s", i, faultCase.Faults)
		}
		want, known := expected[faultCase.Faults]
		if !known {
			want = FaultUnknown
		}
		if faultCase.Support != want {
			t.Fatalf("case %s = %s; want %s", faultCase.Faults, faultCase.Support, want)
		}
	}
	invalid := BrokenCodeSupport{Supported: 1, Rejected: 1}
	if _, err := invalid.Cases(); err == nil {
		t.Fatal("invalid matrix supplied test cases")
	}
	if _, err := invalid.Classify(0); err == nil {
		t.Fatal("invalid matrix returned an apparent success")
	}
}
