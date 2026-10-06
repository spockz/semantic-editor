// These tests protect the published operation-language requirements and capability matrix from incomplete or misleading registry metadata.

package operation_test

import (
	"reflect"
	"semedit/internal/backend"
	"semedit/internal/capability"
	"semedit/internal/operation"
	"slices"
	"strings"
	"testing"
)

func TestLanguageContractsCoverPublishedOperations(t *testing.T) {
	registry := operation.DefaultRegistry()
	for _, entry := range registry.All() {
		if len(entry.LanguageContracts) != len(entry.Languages) {
			t.Errorf("%s language contracts do not match its handler languages", entry.Key)
		}
		for _, language := range entry.Languages {
			contract, ok := entry.LanguageContracts[language]
			if !ok || contract.Validate() != nil {
				t.Errorf("%s/%s has an incomplete contract: %+v", entry.Key, language, contract)
				continue
			}
			description := entry.Description(language)
			for _, variant := range contract.Variants {
				for _, text := range []string{variant.When.String(), variant.AnalysisScope.Description(), variant.PreconditionsDescription(), variant.CapabilitiesDescription(), variant.GuaranteesDescription(), variant.ErrorTolerance.Description()} {
					if !strings.Contains(description, text) {
						t.Errorf("%s/%s description omits %q", entry.Key, language, text)
					}
				}
			}
		}
	}
}

func TestLanguageContractsMatchMatrixSupport(t *testing.T) {
	registry := operation.DefaultRegistry()
	languages := []backend.LanguageID{backend.LanguageGo, backend.LanguageRust, backend.LanguageJava, backend.LanguageScala, backend.LanguageHaskell, backend.LanguageKotlin, backend.LanguageBash, backend.LanguageMake}
	for _, language := range languages {
		matrix, err := registry.Matrix(language)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range registry.All() {
			item, ok := matrix.Operations[entry.Key]
			supported := slices.Contains(entry.Languages, language) || len(entry.Languages) == 1 && entry.Languages[0] == backend.LanguageAuto
			if !ok || item.Supported != supported || (item.LanguageContract != nil) != supported {
				t.Errorf("%s/%s support/contract mismatch: %+v", entry.Key, language, item)
			}
			if supported {
				contract, exists := entry.LanguageContracts[language]
				if !exists {
					contract = entry.LanguageContracts[backend.LanguageAuto]
				}
				if !reflect.DeepEqual(item.LanguageContract, &contract) {
					t.Errorf("%s/%s matrix contract differs from registry", entry.Key, language)
				}
			}
		}
	}

}

func TestTypedContractLabelsAndFaultProfilesRender(t *testing.T) {
	variant := capability.ContractVariant{
		When:          capability.VariantSelectedFile | capability.VariantCheckOnly,
		AnalysisScope: capability.ScopeSelectedFile,
		Preconditions: []capability.Requirement{{Kind: capability.RequireTargetParse}},
		Capabilities:  capability.FeatureDiagnostics,
		Guarantees:    capability.GuaranteeDiagnostics,
		ErrorTolerance: capability.BrokenCodeSupport{
			Supported: capability.CombinationSet(1 | (uint64(1) << capability.FaultTargetType)),
			Rejected:  capability.CombinationSet(uint64(1) << capability.FaultTargetSyntax),
		},
	}
	contract := capability.OperationContract{Variants: []capability.ContractVariant{variant}}
	if err := contract.Validate(); err != nil {
		t.Fatalf("validate typed contract: %v", err)
	}
	got, ok := contract.Variant(variant.When)
	if !ok || got.When.String() != "selected_file,check_only" {
		t.Fatalf("variant lookup = (%+v, %t)", got, ok)
	}
	if got.AnalysisScope.Description() != "the selected file" || got.Preconditions[0].Description() != "the target must parse" || got.GuaranteesDescription() != "diagnostics" {
		t.Fatalf("typed variant descriptions do not match enum labels: %+v", got)
	}
	if status, err := got.ErrorTolerance.Classify(capability.FaultTargetType); err != nil || status != capability.FaultSupported {
		t.Errorf("target type fault status = (%v, %v), want supported", status, err)
	}
	if status, err := got.ErrorTolerance.Classify(capability.FaultTargetSyntax); err != nil || status != capability.FaultRejected {
		t.Errorf("target syntax fault status = (%v, %v), want rejected", status, err)
	}
	for _, want := range []string{"the selected file", "the target must parse", "diagnostics", "target_type=supported", "target_syntax=rejected"} {
		if !strings.Contains(contract.Description(), want) {
			t.Errorf("typed contract description omits %q: %s", want, contract.Description())
		}
	}
}
