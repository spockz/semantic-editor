// Language contracts keep operation-specific analysis requirements in the registry so every public surface describes the same implemented behavior.

package operation

import (
	"fmt"
	"semedit/internal/backend"
	"semedit/internal/capability"
	"strings"
)

// Description renders the registered language contracts for public help and tool discovery.
func (e Entry) Description(languages ...backend.LanguageID) string {
	selected := languages
	if len(selected) == 0 {
		selected = e.Languages
	}
	var description strings.Builder
	description.WriteString(e.Summary)
	for _, language := range selected {
		contract, ok := e.LanguageContracts[language]
		if !ok {
			continue
		}
		label := string(language)
		if language == backend.LanguageAuto {
			label = "language-independent / auto"
		}
		fmt.Fprintf(&description, "\n\n%s: %s", label, contract.Description())
	}
	return description.String()
}

func (r *Registry) installLanguageContracts() error {
	for _, entry := range r.All() {
		entry.LanguageContracts = make(map[backend.LanguageID]capability.OperationContract, len(entry.Languages))
		for _, language := range entry.Languages {
			contract, ok := registeredLanguageContract(entry.Key, language)
			if !ok {
				return fmt.Errorf("register %q language %q requires a language contract: %w", entry.Key, language, ErrInvalidDefinition)
			}
			if err := contract.Validate(); err != nil {
				return fmt.Errorf("register %q language %q has an invalid language contract: %w", entry.Key, language, err)
			}
			entry.LanguageContracts[language] = contract
		}
		publishRegisteredEntry(r, entry)
	}
	return nil
}

func registeredLanguageContract(key string, language backend.LanguageID) (capability.OperationContract, bool) {
	switch language {
	case backend.LanguageGo:
		return goOperationContract(key)
	case backend.LanguageAuto:
		return autoOperationContract(key)
	case backend.LanguageJava:
		if key == "maven_compile" || key == "maven_test" {
			return javaBuildContract(key)
		}
		return externalOperationContract(key, language)
	default:
		return externalOperationContract(key, language)
	}
}

func autoOperationContract(key string) (capability.OperationContract, bool) {
	switch key {
	case "snapshot":
		scope := capability.ScopeWorkspacePaths
		return singleContract(0, scope,
			contractRequirements(scope, capability.RequireSourceStable, capability.RequireWritableDestination, capability.RequireLabel),
			capability.FeatureSnapshot, capability.GuaranteeConflictChecked|capability.GuaranteeAtomicWrites, allFaultSupport()), true
	case "undo":
		scope := capability.ScopeWorkspacePaths
		return singleContract(0, scope,
			contractRequirements(scope, capability.RequireSnapshotState, capability.RequireWritableDestination),
			capability.FeatureRestore, capability.GuaranteeConflictChecked|capability.GuaranteeAtomicWrites, allFaultSupport()), true
	case capability.OpVerify:
		fileScope, scopeScope := capability.ScopeSelectedFile, capability.ScopeSelectedScope
		features := capability.FeatureDiagnostics | capability.FeatureProjectHooks
		guarantees := capability.GuaranteeDiagnostics
		tolerance := conditionalFaultSupport(capability.RecoveryDiagnostics)
		return typedContract(
			contractVariant(capability.VariantSelectedFile|capability.VariantNormalize, fileScope, contractRequirements(fileScope, capability.RequireSourceOrChecks, capability.RequireHookTrust), features|capability.FeatureFormat, guarantees, tolerance),
			contractVariant(capability.VariantSelectedScope|capability.VariantNormalize, scopeScope, contractRequirements(scopeScope, capability.RequireSourceOrChecks, capability.RequireHookTrust), features|capability.FeatureFormat, guarantees, tolerance),
			contractVariant(capability.VariantSelectedFile|capability.VariantCheckOnly, fileScope, contractRequirements(fileScope, capability.RequireSourceOrChecks, capability.RequireHookTrust), features, guarantees, tolerance),
			contractVariant(capability.VariantSelectedScope|capability.VariantCheckOnly, scopeScope, contractRequirements(scopeScope, capability.RequireSourceOrChecks, capability.RequireHookTrust), features, guarantees, tolerance),
		), true
	default:
		return capability.OperationContract{}, false
	}
}

func goOperationContract(key string) (capability.OperationContract, bool) {
	if contract, ok := goASTOperationContract(key); ok {
		return contract, true
	}
	if contract, ok := goServerOperationContract(key); ok {
		return contract, true
	}
	return goTypedOperationContract(key)
}

func externalOperationContract(key string, language backend.LanguageID) (capability.OperationContract, bool) {
	scope := capability.ScopeSelectedFile
	requirements := externalRequirements(language, scope)
	switch key {
	case capability.OpLookup:
		lookup := append(append([]capability.Requirement{}, requirements...),
			capability.Requirement{Kind: capability.RequireDocumentSymbols, Scope: scope},
			capability.Requirement{Kind: capability.RequireExactSelector, Scope: scope})
		return singleContract(capability.VariantSelectedFile, scope, lookup, capability.FeatureLookup,
			capability.GuaranteeReadOnly, conditionalFaultSupport(capability.RecoveryServerAnalysis)), true
	case capability.OpOutline:
		outline := append(append([]capability.Requirement{}, requirements...),
			capability.Requirement{Kind: capability.RequireDocumentSymbols, Scope: scope},
			capability.Requirement{Kind: capability.RequireIncludeUnexported, Scope: scope})
		return singleContract(capability.VariantSelectedFile, scope, outline, capability.FeatureOutline,
			capability.GuaranteeReadOnly, conditionalFaultSupport(capability.RecoveryServerAnalysis)), true
	case capability.OpInspect:
		inspect := append(append([]capability.Requirement{}, requirements...),
			capability.Requirement{Kind: capability.RequireDocumentSymbols, Scope: scope},
			capability.Requirement{Kind: capability.RequireExactSelector, Scope: scope})
		return singleContract(capability.VariantSelectedFile, scope, inspect, capability.FeatureInspect,
			capability.GuaranteeReadOnly, conditionalFaultSupport(capability.RecoveryServerAnalysis)), true
	case capability.OpRename:
		if language != backend.LanguageJava && language != backend.LanguageRust {
			return capability.OperationContract{}, false
		}
		rename := append(append([]capability.Requirement{}, requirements...),
			capability.Requirement{Kind: capability.RequireDocumentSymbols, Scope: scope},
			capability.Requirement{Kind: capability.RequireExactSelector, Scope: scope},
			capability.Requirement{Kind: capability.RequireServerRename, Scope: scope},
			capability.Requirement{Kind: capability.RequireSourceStable, Scope: scope})
		return singleContract(capability.VariantSelectedFile, scope, rename, capability.FeatureRename,
			capability.GuaranteeServerEdit|capability.GuaranteeSelectedFileOnly|capability.GuaranteeAtomicWrites,
			conditionalFaultSupport(capability.RecoveryServerAnalysis)), true
	case capability.OpVerify:
		switch language {
		case backend.LanguageJava:
			normalized := append(append([]capability.Requirement{}, requirements...),
				capability.Requirement{Kind: capability.RequireDiagnosticsReceipt, Scope: scope},
				capability.Requirement{Kind: capability.RequireSourceStable, Scope: scope})
			checked := append(append([]capability.Requirement{}, requirements...),
				capability.Requirement{Kind: capability.RequireDiagnosticsReceipt, Scope: scope})
			tolerance := conditionalFaultSupport(capability.RecoveryDiagnostics)
			return typedContract(
				contractVariant(capability.VariantSelectedFile|capability.VariantNormalize, scope, normalized,
					capability.FeatureFormat|capability.FeatureOrganizeImports|capability.FeatureDiagnostics,
					capability.GuaranteeServerEdit|capability.GuaranteeSelectedFileOnly|capability.GuaranteeDiagnostics|capability.GuaranteeAtomicWrites,
					conditionalFaultSupport(capability.RecoveryNormalization)),
				contractVariant(capability.VariantSelectedFile, scope, checked,
					capability.FeatureDiagnostics, capability.GuaranteeReadOnly|capability.GuaranteeDiagnostics, tolerance),
				contractVariant(capability.VariantSelectedFile|capability.VariantCheckOnly, scope, checked,
					capability.FeatureDiagnostics, capability.GuaranteeReadOnly|capability.GuaranteeDiagnostics, tolerance),
			), true
		case backend.LanguageKotlin, backend.LanguageBash:
			isolated := capability.ScopeIsolatedFile
			isolatedRequirements := append(externalRequirements(language, isolated),
				capability.Requirement{Kind: capability.RequireDiagnosticsReceipt, Scope: isolated},
				capability.Requirement{Kind: capability.RequireDiagnosticsOnly, Scope: isolated})
			tolerance := conditionalFaultSupport(capability.RecoveryDiagnostics)
			features := capability.FeatureDiagnostics
			guarantees := capability.GuaranteeReadOnly | capability.GuaranteeDiagnostics
			return typedContract(
				contractVariant(capability.VariantSelectedFile, isolated, isolatedRequirements, features, guarantees, tolerance),
				contractVariant(capability.VariantSelectedFile|capability.VariantCheckOnly, isolated, isolatedRequirements, features, guarantees, tolerance),
			), true
		default:
			return capability.OperationContract{}, false
		}
	default:
		return capability.OperationContract{}, false
	}
}

func goASTOperationContract(key string) (capability.OperationContract, bool) {
	if contract, ok := goASTSymbolOperationContract(key); ok {
		return contract, true
	}
	fileScope := capability.ScopeSelectedFile
	targetParse := contractRequirements(fileScope, capability.RequireTargetParse)
	localTolerance := targetSyntaxRejectedSupport()
	if key == "add_build_dependency" {
		scope := capability.ScopeModuleManifests
		return singleContract(0, scope, contractRequirements(scope, capability.RequireModuleRoot, capability.RequireToolchain),
			capability.FeatureAddDependency, capability.GuaranteeAtomicWrites,
			conditionalFaultSupport(capability.RecoveryGoCommand)), true
	}
	switch key {
	case "insert_declaration", "insert_function", "insert_type", "insert_decl", "insert_case",
		"replace_decl", "replace_body", "replace_construct", "insert_construct", "remove_construct":
		editRequirements := append([]capability.Requirement{}, targetParse...)
		editRequirements = append(editRequirements, capability.Requirement{Kind: capability.RequireExactSelector, Scope: fileScope})
		if key != "remove_construct" {
			editRequirements = append(editRequirements, capability.Requirement{Kind: capability.RequireSnippetParse, Scope: fileScope})
		}
		edit := contractVariant(capability.VariantSelectedFile, fileScope, editRequirements,
			capability.FeatureASTEdit|capability.FeatureFormat, capability.GuaranteeASTEdit|capability.GuaranteeAtomicWrites, localTolerance)
		if key != "remove_construct" {
			return typedContract(edit), true
		}
		safeDelete := contractVariant(capability.VariantSelectedFile|capability.VariantSafeDelete, capability.ScopeActiveModule,
			contractRequirements(capability.ScopeActiveModule, capability.RequireModuleSnapshot, capability.RequireModuleTypes, capability.RequireTestVariants, capability.RequireExactSelector),
			capability.FeatureASTEdit|capability.FeatureFormat, capability.GuaranteeASTEdit|capability.GuaranteeModuleProof|capability.GuaranteeAtomicWrites,
			moduleProofFaultSupport())
		return typedContract(edit, safeDelete), true
	case "organize_imports":
		file := contractVariant(capability.VariantSelectedFile, fileScope,
			append(targetParse, capability.Requirement{Kind: capability.RequireImportResolution, Scope: fileScope}),
			capability.FeatureOrganizeImports|capability.FeatureFormat|capability.FeatureRepairImports,
			capability.GuaranteeASTEdit|capability.GuaranteeAtomicWrites, localTolerance)
		scopeScope := capability.ScopeSelectedScope
		scope := contractVariant(capability.VariantSelectedScope, scopeScope,
			contractRequirements(scopeScope, capability.RequireScopeParse, capability.RequireImportResolution),
			capability.FeatureOrganizeImports|capability.FeatureFormat|capability.FeatureRepairImports,
			capability.GuaranteeASTEdit|capability.GuaranteeAtomicWrites, selectedScopeFaultSupport())
		return typedContract(file, scope), true
	case "scaffold_file":
		inferred, explicit := capability.VariantSelectedFile|capability.VariantInferPackage, capability.VariantSelectedFile|capability.VariantExplicitPackage
		scope := capability.ScopeSiblingClauses
		reqs := contractRequirements(scope, capability.RequireWritableDestination, capability.RequireSiblingPackageClause)
		return typedContract(
			contractVariant(inferred, scope, reqs, capability.FeatureScaffold, capability.GuaranteeAtomicWrites, targetUnknownFaultSupport()),
			contractVariant(explicit, capability.ScopeSelectedFile, contractRequirements(capability.ScopeSelectedFile, capability.RequireWritableDestination),
				capability.FeatureScaffold, capability.GuaranteeAtomicWrites, targetUnknownFaultSupport()),
		), true
	case "assertion_mode":
		writeReqs := append(append([]capability.Requirement{}, targetParse...),
			capability.Requirement{Kind: capability.RequireExactSelector, Scope: fileScope},
			capability.Requirement{Kind: capability.RequireWorkspaceTrust, Scope: fileScope})
		dryRunReqs := append(append([]capability.Requirement{}, targetParse...), capability.Requirement{Kind: capability.RequireExactSelector, Scope: fileScope})
		return typedContract(
			contractVariant(capability.VariantSelectedFile|capability.VariantDryRun, fileScope, dryRunReqs,
				capability.FeatureAssertions, capability.GuaranteeReadOnly, localTolerance),
			contractVariant(capability.VariantSelectedFile, fileScope, writeReqs,
				capability.FeatureAssertions|capability.FeatureFormat, capability.GuaranteeASTEdit|capability.GuaranteeAtomicWrites, localTolerance),
		), true
	case "move_construct":
		return goMoveOperationContract(), true
	default:
		return capability.OperationContract{}, false
	}
}

func goServerOperationContract(key string) (capability.OperationContract, bool) {
	switch key {
	case capability.OpRename:
		selected := capability.ScopeSelectedFile
		scanned := capability.ScopeWorkspaceScan
		server := capability.ScopeServerScope
		module := capability.ScopeActiveModule
		common := append(contractRequirements(selected, capability.RequireTargetParse),
			capability.Requirement{Kind: capability.RequireGopls, Scope: capability.ScopeNone},
			capability.Requirement{Kind: capability.RequireServerRename, Scope: server})
		standalone := append(append([]capability.Requirement{}, common...),
			capability.Requirement{Kind: capability.RequireBaselineDiagnostics, Scope: module})
		scoped := append(contractRequirements(scanned, capability.RequireScopeParse),
			capability.Requirement{Kind: capability.RequireGopls, Scope: capability.ScopeNone},
			capability.Requirement{Kind: capability.RequireServerRename, Scope: server})
		scopedStandalone := append(append([]capability.Requirement{}, scoped...),
			capability.Requirement{Kind: capability.RequireBaselineDiagnostics, Scope: module})
		tolerance := normalizationFaultSupport(capability.FaultTargetSyntax, capability.RecoveryServerAnalysis)
		return typedContract(
			contractVariant(capability.VariantSelectedFile, server, standalone, capability.FeatureRename,
				capability.GuaranteeServerEdit|capability.GuaranteeDiagnosticDelta|capability.GuaranteeAtomicWrites, tolerance),
			contractVariant(capability.VariantSelectedFile|capability.VariantBatch, server, common, capability.FeatureRename,
				capability.GuaranteeServerEdit|capability.GuaranteeAtomicWrites, tolerance),
			contractVariant(capability.VariantSelectedScope, server, scopedStandalone, capability.FeatureRename,
				capability.GuaranteeServerEdit|capability.GuaranteeDiagnosticDelta|capability.GuaranteeAtomicWrites, tolerance),
			contractVariant(capability.VariantSelectedScope|capability.VariantBatch, server, scoped, capability.FeatureRename,
				capability.GuaranteeServerEdit|capability.GuaranteeAtomicWrites, tolerance),
		), true
	case capability.OpVerify:
		fileScope, scopeScope := capability.ScopeSelectedFile, capability.ScopeSelectedScope
		fileNormalization := normalizationFaultSupport(capability.FaultTargetSyntax, capability.RecoveryNormalization)
		scopeNormalization := normalizationFaultSupport(capability.FaultTargetSyntax|capability.FaultSamePackageSyntax, capability.RecoveryNormalization)
		diagnostics := conditionalFaultSupport(capability.RecoveryDiagnostics)
		normalizationFeatures := capability.FeatureFormat | capability.FeatureOrganizeImports | capability.FeatureDiagnostics | capability.FeatureProjectHooks
		return typedContract(
			contractVariant(capability.VariantSelectedFile|capability.VariantNormalize, fileScope,
				contractRequirements(fileScope, capability.RequireGopls, capability.RequireToolchain, capability.RequireHookTrust),
				normalizationFeatures, capability.GuaranteeDiagnostics, fileNormalization),
			contractVariant(capability.VariantSelectedScope|capability.VariantNormalize, scopeScope,
				contractRequirements(scopeScope, capability.RequireGopls, capability.RequireToolchain, capability.RequireHookTrust),
				normalizationFeatures, capability.GuaranteeDiagnostics, scopeNormalization),
			contractVariant(capability.VariantSelectedFile|capability.VariantCheckOnly, fileScope,
				contractRequirements(fileScope, capability.RequireGopls, capability.RequireToolchain, capability.RequireDiagnosticsOnly, capability.RequireHookTrust),
				capability.FeatureDiagnostics|capability.FeatureProjectHooks, capability.GuaranteeDiagnostics, diagnostics),
			contractVariant(capability.VariantSelectedScope|capability.VariantCheckOnly, scopeScope,
				contractRequirements(scopeScope, capability.RequireGopls, capability.RequireToolchain, capability.RequireDiagnosticsOnly, capability.RequireHookTrust),
				capability.FeatureDiagnostics|capability.FeatureProjectHooks, capability.GuaranteeDiagnostics, diagnostics),
		), true
	default:
		return capability.OperationContract{}, false
	}
}

func goTypedOperationContract(key string) (capability.OperationContract, bool) {
	packageScope := capability.ScopeSelectedPackage
	localRequirements := append(contractRequirements(packageScope, capability.RequireTargetParse, capability.RequirePackageTypes),
		capability.Requirement{Kind: capability.RequireExactSelector, Scope: packageScope},
		capability.Requirement{Kind: capability.RequireBoundedSource, Scope: packageScope},
		capability.Requirement{Kind: capability.RequireOverlayTypes, Scope: packageScope})
	local := contractVariant(capability.VariantSelectedFile, packageScope, localRequirements,
		capability.FeatureTypedRefactor|capability.FeatureFormat, capability.GuaranteePackageOverlay|capability.GuaranteeAtomicWrites,
		selectedPackageFaultSupport())
	moduleScope := capability.ScopeActiveModule
	moduleRequirements := append(contractRequirements(moduleScope, capability.RequireModuleRoot, capability.RequireModuleSnapshot, capability.RequireModuleTypes, capability.RequireTestVariants),
		capability.Requirement{Kind: capability.RequireExactSelector, Scope: moduleScope},
		capability.Requirement{Kind: capability.RequireBoundedSource, Scope: moduleScope},
		capability.Requirement{Kind: capability.RequireOverlayTypes, Scope: moduleScope})
	moduleMutation := contractVariant(capability.VariantSelectedFile, moduleScope, moduleRequirements,
		capability.FeatureTypedRefactor|capability.FeatureFormat,
		capability.GuaranteeModuleProof|capability.GuaranteePackageOverlay|capability.GuaranteeAtomicWrites,
		moduleProofFaultSupport())
	switch key {
	case "extract_constant":
		local.Preconditions = append(local.Preconditions, capability.Requirement{Kind: capability.RequireTestVariants, Scope: packageScope})
		return typedContract(local), true
	case "extract_function", "extract_variable", "inline_variable":
		return typedContract(local), true
	case capability.OpFindReferences:
		references := contractVariant(capability.VariantSelectedFile, moduleScope,
			append(contractRequirements(moduleScope, capability.RequireModuleRoot, capability.RequireModuleSnapshot, capability.RequireModuleTypes, capability.RequireTestVariants),
				capability.Requirement{Kind: capability.RequireExactSelector, Scope: moduleScope}),
			capability.FeatureReferences, capability.GuaranteeReadOnly|capability.GuaranteeModuleProof,
			moduleProofFaultSupport())
		return typedContract(references), true
	case "inline_function", "migrate_type", "change_signature", "introduce_parameter",
		"introduce_parameter_object", "extract_interface", "implement_contract":
		return typedContract(moduleMutation), true
	case "insert_field", "insert_composite_element", "extract_type", "set_doc_comment":
		structuralRequirements := append(contractRequirements(moduleScope, capability.RequireModuleSnapshot, capability.RequireTestVariants),
			capability.Requirement{Kind: capability.RequireExactSelector, Scope: packageScope},
			capability.Requirement{Kind: capability.RequireBoundedSource, Scope: packageScope},
			capability.Requirement{Kind: capability.RequirePackageTypes, Scope: packageScope},
			capability.Requirement{Kind: capability.RequireOverlayTypes, Scope: packageScope})
		structural := contractVariant(capability.VariantSelectedFile, moduleScope, structuralRequirements,
			capability.FeatureTypedRefactor|capability.FeatureFormat,
			capability.GuaranteePackageOverlay|capability.GuaranteeAtomicWrites,
			structuralSnapshotFaultSupport())
		return typedContract(structural), true
	default:
		return capability.OperationContract{}, false
	}
}
