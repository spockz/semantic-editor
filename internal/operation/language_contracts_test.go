// These tests protect the published operation-language requirements and capability matrix from incomplete or misleading registry metadata.

package operation_test

import (
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/capability"
	"github.com/spockz/semantic-editor/internal/operation"
	"reflect"
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

func TestGoStageContractsSeparateVolatilePhasesFromPublication(t *testing.T) {
	registry := operation.DefaultRegistry()
	module := capability.ScopeActiveModule
	phaseRequirements := []capability.Requirement{
		{Kind: capability.RequireModuleRoot, Scope: module},
		{Kind: capability.RequireModuleSnapshot, Scope: module},
	}
	sessionRequirements := append(append([]capability.Requirement{}, phaseRequirements...), capability.Requirement{Kind: capability.RequireStageSession, Scope: capability.ScopeServerScope})
	for _, key := range []string{"stage_start", "stage_drop", "stage_abort"} {
		entry, ok := registry.LookupKey(key)
		if !ok {
			t.Fatalf("%s is not registered", key)
		}
		profile, ok := entry.LanguageContracts[backend.LanguageGo].Variant(0)
		if !ok {
			t.Fatalf("%s is missing its default lifecycle profile", key)
		}
		wantRequirements := phaseRequirements
		if key != "stage_start" {
			wantRequirements = sessionRequirements
		}
		if profile.AnalysisScope != module || !reflect.DeepEqual(profile.Preconditions, wantRequirements) {
			t.Errorf("%s lifecycle profile = %+v, want module workflow requirements %+v", key, profile, wantRequirements)
		}
		if profile.Capabilities != capability.FeatureStaging || profile.Guarantees != capability.GuaranteeVolatileState {
			t.Errorf("%s lifecycle profile = %+v, want volatile staging without disk publication", key, profile)
		}
	}
	entry, ok := registry.LookupKey("stage_commit")
	if !ok {
		t.Fatal("stage_commit is not registered")
	}
	profile, ok := entry.LanguageContracts[backend.LanguageGo].Variant(0)
	if !ok {
		t.Fatal("stage_commit is missing its default publication profile")
	}
	wantFeatures := capability.FeaturePublication | capability.FeatureFormat | capability.FeatureOrganizeImports | capability.FeatureDiagnostics | capability.FeatureProjectHooks
	wantGuarantees := capability.GuaranteeConflictChecked | capability.GuaranteeAtomicWrites | capability.GuaranteeDiagnostics | capability.GuaranteePublicationOutcome
	if profile.AnalysisScope != module || !reflect.DeepEqual(profile.Preconditions, sessionRequirements) || profile.Capabilities != wantFeatures || profile.Guarantees != wantGuarantees {
		t.Errorf("stage_commit profile = %+v, want explicit publication with final verification", profile)
	}
	if profile.Capabilities.Has(capability.FeatureASTEdit) || profile.Guarantees.Has(capability.GuaranteeASTEdit) {
		t.Errorf("stage_commit must not advertise AST mutation: %+v", profile)
	}
}

func TestGoInspectConstructReadOnlyProfile(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupKey("inspect_construct")
	if !ok {
		t.Fatal("inspect_construct is not registered")
	}
	contract := entry.LanguageContracts[backend.LanguageGo]
	profile, ok := contract.Variant(capability.VariantSelectedFile)
	if !ok {
		t.Fatal("inspect_construct is missing its selected-file profile")
	}
	if profile.AnalysisScope != capability.ScopeSelectedFile {
		t.Errorf("analysis scope = %q, want %q", profile.AnalysisScope, capability.ScopeSelectedFile)
	}
	wantRequirements := []capability.Requirement{
		{Kind: capability.RequireTargetParse, Scope: capability.ScopeSelectedFile},
		{Kind: capability.RequireExactSelector, Scope: capability.ScopeSelectedFile},
	}
	if !reflect.DeepEqual(profile.Preconditions, wantRequirements) {
		t.Errorf("preconditions = %+v, want %+v", profile.Preconditions, wantRequirements)
	}
	if profile.Capabilities != capability.FeatureInspect || profile.Guarantees != capability.GuaranteeReadOnly {
		t.Errorf("profile = %+v, want read-only inspect", profile)
	}
	if support, err := profile.ErrorTolerance.Classify(capability.FaultTargetSyntax); err != nil || support != capability.FaultRejected {
		t.Errorf("target syntax support = (%v, %v), want rejected", support, err)
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

func TestGoInsertConstructStructuralCapabilityProfiles(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupKey("insert_construct")
	if !ok {
		t.Fatal("insert_construct is not registered")
	}
	contract := entry.LanguageContracts[backend.LanguageGo]
	if len(contract.Variants) != 3 {
		t.Fatalf("insert_construct variants = %d, want 3", len(contract.Variants))
	}
	if _, baseExists := contract.Variant(capability.VariantSelectedFile); !baseExists {
		t.Fatal("insert_construct is missing its base selected-file profile")
	}

	selector := capability.VariantSelectedFile | capability.VariantField
	field, fieldExists := contract.Variant(selector)
	if !fieldExists {
		t.Fatalf("insert_construct is missing %q profile", selector)
	}
	if field.AnalysisScope != capability.ScopeActiveModule {
		t.Errorf("field analysis scope = %q, want %q", field.AnalysisScope, capability.ScopeActiveModule)
	}
	wantRequirements := []capability.Requirement{
		{Kind: capability.RequireModuleSnapshot, Scope: capability.ScopeActiveModule},
		{Kind: capability.RequireTestVariants, Scope: capability.ScopeActiveModule},
		{Kind: capability.RequireExactSelector, Scope: capability.ScopeSelectedPackage},
		{Kind: capability.RequireBoundedSource, Scope: capability.ScopeSelectedPackage},
		{Kind: capability.RequirePackageTypes, Scope: capability.ScopeSelectedPackage},
		{Kind: capability.RequireOverlayTypes, Scope: capability.ScopeSelectedPackage},
		{Kind: capability.RequireSnippetParse, Scope: capability.ScopeSelectedFile},
	}
	if !reflect.DeepEqual(field.Preconditions, wantRequirements) {
		t.Errorf("field preconditions = %+v, want %+v", field.Preconditions, wantRequirements)
	}
	wantFeatures := capability.FeatureASTEdit | capability.FeatureTypedRefactor | capability.FeatureFormat
	if field.Capabilities != wantFeatures {
		t.Errorf("field capabilities = %v, want %v", field.Capabilities, wantFeatures)
	}
	wantGuarantees := capability.GuaranteePackageOverlay | capability.GuaranteeAtomicWrites
	if field.Guarantees != wantGuarantees {
		t.Errorf("field guarantees = %v, want %v", field.Guarantees, wantGuarantees)
	}
	if support, err := field.ErrorTolerance.Classify(capability.FaultTargetSyntax); err != nil || support != capability.FaultRejected {
		t.Errorf("field target syntax support = (%v, %v), want rejected", support, err)
	}
	if support, err := field.ErrorTolerance.Classify(capability.FaultOtherPackageType); err != nil || support != capability.FaultUnknown {
		t.Errorf("field other-package type support = (%v, %v), want unknown", support, err)
	}
	elementSelector := capability.VariantSelectedFile | capability.VariantElement
	element, elementExists := contract.Variant(elementSelector)
	if !elementExists {
		t.Fatalf("insert_construct is missing %q profile", elementSelector)
	}
	wantElement := field
	wantElement.When = elementSelector
	wantElement.Capabilities = capability.FeatureASTEdit | capability.FeatureTypedRefactor
	if !reflect.DeepEqual(element, wantElement) {
		t.Errorf("element profile = %+v, want structural bounds %+v", element, wantElement)
	}
}
