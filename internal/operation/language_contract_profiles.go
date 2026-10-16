// This file maps operation-specific scopes and fault masks into the typed capability contract model.

package operation

import (
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/capability"
)

func contractVariant(when capability.VariantSet, scope capability.AnalysisScope, requirements []capability.Requirement, features capability.FeatureSet, guarantees capability.GuaranteeSet, tolerance capability.BrokenCodeSupport) capability.ContractVariant {
	return capability.ContractVariant{
		When:           when,
		AnalysisScope:  scope,
		Preconditions:  requirements,
		Capabilities:   features,
		Guarantees:     guarantees,
		ErrorTolerance: tolerance,
	}
}

func classifyFaults(classify func(capability.CodeFaults) capability.FaultSupport, condition capability.RecoveryCondition) capability.BrokenCodeSupport {
	var support capability.BrokenCodeSupport
	for faults := capability.CodeFaults(0); faults <= capability.AllCodeFaults; faults++ {
		status := classify(faults)
		bit := capability.CombinationSet(uint64(1) << uint(faults))
		switch status {
		case capability.FaultSupported:
			support.Supported |= bit
		case capability.FaultRejected:
			support.Rejected |= bit
		case capability.FaultConditional:
			support.Conditional |= bit
		case capability.FaultUnknown:
		default:
			panic("unrecognized code-fault support status")
		}
	}
	if support.Conditional != 0 {
		support.Condition = condition
	}
	return support
}

func allFaultSupport() capability.BrokenCodeSupport {
	return classifyFaults(func(capability.CodeFaults) capability.FaultSupport {
		return capability.FaultSupported
	}, capability.RecoveryNone)
}

func withoutFaultSupport(rejected capability.CodeFaults) capability.BrokenCodeSupport {
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults&rejected != 0 {
			return capability.FaultRejected
		}
		return capability.FaultSupported
	}, capability.RecoveryNone)
}

func conditionalFaultSupport(condition capability.RecoveryCondition) capability.BrokenCodeSupport {
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults == 0 {
			return capability.FaultSupported
		}
		return capability.FaultConditional
	}, condition)
}

func contractRequirements(scope capability.AnalysisScope, kinds ...capability.RequirementKind) []capability.Requirement {
	requirements := make([]capability.Requirement, len(kinds))
	for i, kind := range kinds {
		requirements[i] = capability.Requirement{Kind: kind, Scope: scope}
	}
	return requirements
}

func typedContract(variants ...capability.ContractVariant) capability.OperationContract {
	return capability.OperationContract{Variants: variants}
}

func goFaultMask() capability.CodeFaults {
	return capability.AllCodeFaults
}

func singleContract(when capability.VariantSet, scope capability.AnalysisScope, requirements []capability.Requirement, features capability.FeatureSet, guarantees capability.GuaranteeSet, tolerance capability.BrokenCodeSupport) capability.OperationContract {
	return typedContract(contractVariant(when, scope, requirements, features, guarantees, tolerance))
}

func targetSyntaxRejectedSupport() capability.BrokenCodeSupport {
	return withoutFaultSupport(capability.FaultTargetSyntax)
}

func externalRequirements(language backend.LanguageID, scope capability.AnalysisScope) []capability.Requirement {
	kinds := []capability.RequirementKind{capability.RequireWorkspaceTrust, capability.RequireSelectedFile}
	switch language {
	case backend.LanguageRust:
		kinds = append(kinds, capability.RequireCargoRoot, capability.RequireRustAnalyzer)
	case backend.LanguageJava:
		kinds = append(kinds, capability.RequireJavaRoot, capability.RequireJDTLS, capability.RequireJava21)
	case backend.LanguageScala:
		kinds = append(kinds, capability.RequireScalaRoot, capability.RequireMetals, capability.RequireJava21)
	case backend.LanguageHaskell:
		kinds = append(kinds, capability.RequireStandaloneHaskell, capability.RequireNoHaskellProject, capability.RequireHLS)
	case backend.LanguageKotlin:
		kinds = append(kinds, capability.RequireKotlinServer)
	case backend.LanguageBash:
		kinds = append(kinds, capability.RequireBashServer)
	case backend.LanguageMake:
		kinds = append(kinds, capability.RequireMakeServer)
	case backend.LanguageGo, backend.LanguageAuto:
		panic(fmt.Sprintf("external language requirements requested for %q", language))
	default:
		panic(fmt.Sprintf("unsupported external language %q", language))
	}
	return contractRequirements(scope, kinds...)
}

func selectedScopeFaultSupport() capability.BrokenCodeSupport {
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults == 0 {
			return capability.FaultSupported
		}
		if faults&capability.FaultTargetSyntax != 0 {
			return capability.FaultRejected
		}
		if faults&(capability.FaultSamePackageSyntax|capability.FaultOtherPackageSyntax) != 0 {
			return capability.FaultUnknown
		}
		return capability.FaultSupported
	}, capability.RecoveryNone)
}

func targetUnknownFaultSupport() capability.BrokenCodeSupport {
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults&(capability.FaultTargetSyntax|capability.FaultTargetType) != 0 {
			return capability.FaultUnknown
		}
		return capability.FaultSupported
	}, capability.RecoveryNone)
}

func normalizationFaultSupport(rejected capability.CodeFaults, condition capability.RecoveryCondition) capability.BrokenCodeSupport {
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults&rejected != 0 {
			return capability.FaultRejected
		}
		if faults != 0 {
			return capability.FaultConditional
		}
		return capability.FaultSupported
	}, condition)
}

func selectedPackageFaultSupport() capability.BrokenCodeSupport {
	rejected := capability.FaultTargetSyntax | capability.FaultTargetType | capability.FaultSamePackageSyntax | capability.FaultSamePackageType
	uncertain := capability.FaultOtherPackageSyntax | capability.FaultOtherPackageType
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults&rejected != 0 {
			return capability.FaultRejected
		}
		if faults&uncertain != 0 {
			return capability.FaultUnknown
		}
		return capability.FaultSupported
	}, capability.RecoveryNone)
}

func structuralSnapshotFaultSupport() capability.BrokenCodeSupport {
	rejected := capability.FaultTargetSyntax | capability.FaultSamePackageSyntax | capability.FaultOtherPackageSyntax | capability.FaultTargetType | capability.FaultSamePackageType
	return classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
		if faults&rejected != 0 {
			return capability.FaultRejected
		}
		if faults&capability.FaultOtherPackageType != 0 {
			return capability.FaultUnknown
		}
		return capability.FaultSupported
	}, capability.RecoveryNone)
}

func moduleProofFaultSupport() capability.BrokenCodeSupport {
	return withoutFaultSupport(goFaultMask())
}

func javaBuildContract(key string) (capability.OperationContract, bool) {
	scope := capability.ScopeProjectBuild
	requirements := contractRequirements(scope, capability.RequireWorkspaceTrust, capability.RequireMavenRoot, capability.RequireMaven)
	features := capability.FeatureCompile
	if key == "maven_test" {
		features = capability.FeatureTest
	} else if key != "maven_compile" {
		return capability.OperationContract{}, false
	}
	return singleContract(0, scope, requirements, features, capability.GuaranteeDiagnostics,
		conditionalFaultSupport(capability.RecoveryBuildCommand)), true
}

func goASTSymbolOperationContract(key string) (capability.OperationContract, bool) {
	scope := capability.ScopeSelectedFile
	feature := capability.FeatureLookup
	switch key {
	case capability.OpLookup:
	case capability.OpInspect:
		feature = capability.FeatureInspect
	case capability.OpOutline:
		feature = capability.FeatureOutline
	default:
		return capability.OperationContract{}, false
	}
	fileRequirements := contractRequirements(scope, capability.RequireTargetParse)
	if key != capability.OpOutline {
		fileRequirements = append(fileRequirements, capability.Requirement{Kind: capability.RequireExactSelector, Scope: scope})
	}
	selectedScope := capability.ScopeSelectedScope
	scopeRequirements := contractRequirements(selectedScope, capability.RequireScopeParse)
	if key != capability.OpOutline {
		scopeRequirements = append(scopeRequirements, capability.Requirement{Kind: capability.RequireExactSelector, Scope: selectedScope})
	}
	return typedContract(
		contractVariant(capability.VariantSelectedFile, scope, fileRequirements, feature, capability.GuaranteeReadOnly, targetSyntaxRejectedSupport()),
		contractVariant(capability.VariantSelectedScope, selectedScope, scopeRequirements, feature, capability.GuaranteeReadOnly, selectedScopeFaultSupport()),
	), true
}

func goMoveOperationContract() capability.OperationContract {
	fileScope := capability.ScopeSelectedFile
	requirements := append(contractRequirements(fileScope, capability.RequireTargetParse, capability.RequireExactSelector),
		capability.Requirement{Kind: capability.RequireScopeParse, Scope: capability.ScopeSelectedScope})
	same := contractVariant(capability.VariantSelectedFile|capability.VariantSamePackage, capability.ScopeSelectedScope, requirements,
		capability.FeatureASTEdit|capability.FeatureFormat, capability.GuaranteeASTEdit|capability.GuaranteeAtomicWrites,
		classifyFaults(func(faults capability.CodeFaults) capability.FaultSupport {
			if faults&capability.FaultTargetSyntax != 0 {
				return capability.FaultRejected
			}
			if faults&capability.FaultSamePackageSyntax != 0 {
				return capability.FaultUnknown
			}
			return capability.FaultSupported
		}, capability.RecoveryNone))
	crossScope := capability.ScopeActiveModule
	cross := contractVariant(capability.VariantSelectedFile|capability.VariantCrossPackage, crossScope,
		contractRequirements(crossScope, capability.RequireModuleSnapshot, capability.RequireModuleTypes, capability.RequireTestVariants, capability.RequireExactSelector, capability.RequireBoundedSource, capability.RequireOverlayTypes),
		capability.FeatureTypedRefactor|capability.FeatureFormat, capability.GuaranteeModuleProof|capability.GuaranteePackageOverlay|capability.GuaranteeAtomicWrites,
		moduleProofFaultSupport())
	return typedContract(same, cross)
}

func goMoveFileOperationContract() capability.OperationContract {
	scope := capability.ScopeActiveModule
	return typedContract(contractVariant(capability.VariantSelectedFile|capability.VariantCrossPackage, scope,
		contractRequirements(scope, capability.RequireModuleSnapshot, capability.RequireModuleTypes, capability.RequireTestVariants, capability.RequireExactSelector, capability.RequireBoundedSource, capability.RequireOverlayTypes),
		capability.FeatureTypedRefactor|capability.FeatureFormat, capability.GuaranteeModuleProof|capability.GuaranteePackageOverlay|capability.GuaranteeAtomicWrites, moduleProofFaultSupport()))
}
