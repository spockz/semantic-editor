// Package capability models language operation contracts as validated, typed profiles.
package capability

import (
	"fmt"
	"slices"
	"strings"
)

// AnalysisScope bounds which source area an operation analyzes.
type AnalysisScope string

// The scope values distinguish parsing, typing, and project-level proof boundaries.
const (
	ScopeNone            AnalysisScope = "none"
	ScopeSelectedFile    AnalysisScope = "selected_file"
	ScopeSelectedScope   AnalysisScope = "selected_scope"
	ScopeWorkspaceScan   AnalysisScope = "workspace_scan"
	ScopeSelectedPackage AnalysisScope = "selected_package"
	ScopeActiveModule    AnalysisScope = "active_module"
	ScopeServerScope     AnalysisScope = "server_scope"
	ScopeIsolatedFile    AnalysisScope = "isolated_file"
	ScopeModuleManifests AnalysisScope = "module_manifests"
	ScopeProjectBuild    AnalysisScope = "project_build"
	ScopeWorkspacePaths  AnalysisScope = "workspace_paths"
	ScopeSiblingClauses  AnalysisScope = "sibling_package_clauses"
)

// Valid reports whether s is a declared analysis scope.
func (s AnalysisScope) Valid() bool {
	switch s {
	case ScopeNone, ScopeSelectedFile, ScopeSelectedScope, ScopeWorkspaceScan, ScopeSelectedPackage, ScopeActiveModule, ScopeServerScope, ScopeIsolatedFile, ScopeModuleManifests, ScopeProjectBuild, ScopeWorkspacePaths, ScopeSiblingClauses:
		return true
	default:
		return false
	}
}

// Description returns the human-readable analysis boundary for s.
func (s AnalysisScope) Description() string {
	switch s {
	case ScopeNone:
		return "no analysis"
	case ScopeSelectedFile:
		return "the selected file"
	case ScopeSelectedScope:
		return "the selected scope"
	case ScopeWorkspaceScan:
		return "a workspace scan"
	case ScopeSelectedPackage:
		return "the selected package"
	case ScopeActiveModule:
		return "the active module"
	case ScopeServerScope:
		return "the language server's scope"
	case ScopeIsolatedFile:
		return "the selected file in isolation"
	case ScopeModuleManifests:
		return "module manifests"
	case ScopeProjectBuild:
		return "the project build"
	case ScopeWorkspacePaths:
		return "workspace paths"
	case ScopeSiblingClauses:
		return "sibling package clauses"
	default:
		return "unknown scope (" + string(s) + ")"
	}
}

// RequirementKind identifies a closed, documented precondition category.
type RequirementKind string

// Requirement values are closed identifiers; arbitrary prose is not accepted.
const (
	RequireTargetParse            RequirementKind = "target_parse"
	RequireScopeParse             RequirementKind = "scope_parse"
	RequireSnippetParse           RequirementKind = "snippet_parse"
	RequireExactSelector          RequirementKind = "exact_selector"
	RequirePackageTypes           RequirementKind = "package_types"
	RequireModuleTypes            RequirementKind = "module_types"
	RequireModuleSnapshot         RequirementKind = "module_snapshot"
	RequireOverlayTypes           RequirementKind = "overlay_types"
	RequireTestVariants           RequirementKind = "test_variants"
	RequireDocumentSymbols        RequirementKind = "document_symbols"
	RequireServerRename           RequirementKind = "server_rename"
	RequireBaselineDiagnostics    RequirementKind = "baseline_diagnostics"
	RequireDiagnosticsReceipt     RequirementKind = "diagnostics_receipt"
	RequireImportResolution       RequirementKind = "import_resolution"
	RequireWorkspaceTrust         RequirementKind = "workspace_trust"
	RequireHookTrust              RequirementKind = "hook_trust"
	RequireSelectedFile           RequirementKind = "selected_file"
	RequireModuleRoot             RequirementKind = "module_root"
	RequireToolchain              RequirementKind = "toolchain"
	RequireGopls                  RequirementKind = "gopls"
	RequireRustAnalyzer           RequirementKind = "rust_analyzer"
	RequireJDTLS                  RequirementKind = "jdtls"
	RequireJava21                 RequirementKind = "java_21"
	RequireMetals                 RequirementKind = "metals"
	RequireHLS                    RequirementKind = "hls"
	RequireKotlinServer           RequirementKind = "kotlin_server"
	RequireBashServer             RequirementKind = "bash_server"
	RequireMakeServer             RequirementKind = "make_server"
	RequireMaven                  RequirementKind = "maven"
	RequireMavenRoot              RequirementKind = "maven_root"
	RequireCargoRoot              RequirementKind = "cargo_root"
	RequireJavaRoot               RequirementKind = "java_root"
	RequireScalaRoot              RequirementKind = "scala_root"
	RequireStandaloneHaskell      RequirementKind = "standalone_haskell"
	RequireNoHaskellProject       RequirementKind = "no_haskell_project"
	RequireSourceStable           RequirementKind = "source_stable"
	RequireWritableDestination    RequirementKind = "writable_destination"
	RequireSnapshot               RequirementKind = "snapshot"
	RequireSnapshotState          RequirementKind = "snapshot_state"
	RequireLabel                  RequirementKind = "label"
	RequireSiblingPackageClause   RequirementKind = "sibling_package_clause"
	RequireSourceOrChecks         RequirementKind = "source_or_checks"
	RequireDiagnosticsOnly        RequirementKind = "diagnostics_only"
	RequireIncludeUnexported      RequirementKind = "include_unexported"
	RequireBoundedSource          RequirementKind = "bounded_source"
	RequireResolvableDependencies RequirementKind = "resolvable_dependencies"
	RequireStageSession           RequirementKind = "stage_session"
)

var requirementDescriptions = map[RequirementKind]string{
	RequireTargetParse: "the target must parse", RequireScopeParse: "the selected scope must parse", RequireSnippetParse: "the supplied snippet must parse", RequireExactSelector: "the selector must identify exactly one target", RequirePackageTypes: "package type information must be available", RequireModuleTypes: "module type information must be available", RequireModuleSnapshot: "the module snapshot must be available", RequireOverlayTypes: "overlay type information must be available", RequireTestVariants: "test variants must be available", RequireDocumentSymbols: "hierarchical, valid document symbols must be available", RequireServerRename: "the language server must support rename", RequireBaselineDiagnostics: "baseline diagnostics must be available", RequireDiagnosticsReceipt: "a timely diagnostics receipt matching the current file must be available; silence is not a receipt", RequireImportResolution: "imports must resolve", RequireWorkspaceTrust: "the workspace must be trusted", RequireHookTrust: "project hooks must be trusted", RequireSelectedFile: "a file must be selected", RequireModuleRoot: "a module root must be discoverable", RequireToolchain: "the language toolchain must be available", RequireGopls: "gopls must be available", RequireRustAnalyzer: "rust-analyzer must be available", RequireJDTLS: "JDT LS must be available", RequireJava21: "Java 21 or newer must be available", RequireMetals: "Metals must be available", RequireHLS: "HLS matching the selected GHC toolchain must be available", RequireKotlinServer: "the Kotlin language server must be available", RequireBashServer: "the Bash language server must be available", RequireMakeServer: "the Make language server must be available", RequireMaven: "Maven must be available", RequireMavenRoot: "a Maven root must be discoverable", RequireCargoRoot: "a Cargo root must be discoverable", RequireJavaRoot: "an explicit, unambiguous Java project root must be selected", RequireScalaRoot: "a Scala project root must be discoverable", RequireStandaloneHaskell: "standalone Haskell mode with matching GHC and HLS must be selected", RequireNoHaskellProject: "no Haskell project markers may be present", RequireSourceStable: "the source must remain stable", RequireWritableDestination: "the destination must be writable", RequireSnapshot: "a snapshot must be available", RequireSnapshotState: "the snapshot state must be available", RequireLabel: "a label must be supplied", RequireSiblingPackageClause: "a valid non-test sibling package clause must be available; the first eligible sibling is selected", RequireSourceOrChecks: "source or project checks must be available", RequireDiagnosticsOnly: "the operation must be diagnostics-only", RequireIncludeUnexported: "unexported declarations must be included", RequireBoundedSource: "the source must fit configured bounds", RequireResolvableDependencies: "dependencies must be resolvable", RequireStageSession: "an active retained staging session must exist",
}

// Requirement combines a known precondition with its optional analysis scope.
type Requirement struct {
	Kind  RequirementKind
	Scope AnalysisScope
}

// Valid reports whether r uses a known requirement and optional known scope.
func (r Requirement) Valid() bool {
	_, ok := requirementDescriptions[r.Kind]
	return ok && (r.Scope == "" || r.Scope.Valid())
}

// Description returns the readable precondition represented by r.
func (r Requirement) Description() string {
	d, ok := requirementDescriptions[r.Kind]
	if !ok {
		return "unknown requirement (" + string(r.Kind) + ")"
	}
	if r.Scope != "" && r.Scope != ScopeNone {
		return d + " in " + r.Scope.Description()
	}
	return d
}

// VariantSet identifies one exact operation profile; bits never imply profiles by subset.
type VariantSet uint16

// Variant bits form exact profiles; the validator rejects contradictory pairs.
const (
	VariantSelectedFile VariantSet = 1 << iota
	VariantSelectedScope
	VariantSamePackage
	VariantCrossPackage
	VariantSafeDelete
	VariantInferPackage
	VariantExplicitPackage
	VariantCheckOnly
	VariantNormalize
	VariantDryRun
	VariantBatch
	VariantField
	VariantElement
)

const allVariants = VariantSelectedFile | VariantSelectedScope | VariantSamePackage | VariantCrossPackage | VariantSafeDelete | VariantInferPackage | VariantExplicitPackage | VariantCheckOnly | VariantNormalize | VariantDryRun | VariantBatch | VariantField | VariantElement

// Valid reports whether v contains only known, mutually compatible selector bits.
func (v VariantSet) Valid() bool {
	if v&^allVariants != 0 {
		return false
	}
	conflicts := []VariantSet{
		VariantSelectedFile | VariantSelectedScope,
		VariantSamePackage | VariantCrossPackage,
		VariantInferPackage | VariantExplicitPackage,
		VariantNormalize | VariantCheckOnly,
		VariantField | VariantElement,
	}
	return !slices.ContainsFunc(conflicts, v.Has)
}

// Has reports whether all bits in flag are present in v.
func (v VariantSet) Has(flag VariantSet) bool { return flag != 0 && v&flag == flag }

// String returns the stable selector label used by contract documentation.
func (v VariantSet) String() string {
	if v == 0 {
		return "default"
	}
	labels := []struct {
		flag VariantSet
		name string
	}{{VariantSelectedFile, "selected_file"}, {VariantSelectedScope, "selected_scope"}, {VariantSamePackage, "same_package"}, {VariantCrossPackage, "cross_package"}, {VariantSafeDelete, "safe_delete"}, {VariantInferPackage, "infer_package"}, {VariantExplicitPackage, "explicit_package"}, {VariantCheckOnly, "check_only"}, {VariantNormalize, "normalize"}, {VariantDryRun, "dry_run"}, {VariantBatch, "batch"}, {VariantField, "field"}, {VariantElement, "element"}}
	parts := make([]string, 0, len(labels))
	for _, x := range labels {
		if v.Has(x.flag) {
			parts = append(parts, x.name)
		}
	}
	if v & ^allVariants != 0 {
		parts = append(parts, fmt.Sprintf("unknown_%x", uint16(v & ^allVariants)))
	}
	return strings.Join(parts, ",")
}

// FeatureSet lists the operation features promised by a profile.
type FeatureSet uint32

// Feature bits name typed operations available in a profile.
const (
	FeatureLookup FeatureSet = 1 << iota
	FeatureOutline
	FeatureInspect
	FeatureASTEdit
	FeatureTypedRefactor
	FeatureRename
	FeatureFormat
	FeatureOrganizeImports
	FeatureDiagnostics
	FeatureProjectHooks
	FeatureCompile
	FeatureTest
	FeatureSnapshot
	FeatureRestore
	FeatureAddDependency
	FeatureScaffold
	FeatureAssertions
	FeatureRepairImports
	FeatureStaging        FeatureSet = 1 << 19
	FeaturePublication    FeatureSet = 1 << 20
	FeatureDecisionSearch FeatureSet = 1 << 21
	FeatureDocumentIndex  FeatureSet = 1 << 22
)
const allFeatures = FeatureLookup | FeatureOutline | FeatureInspect | FeatureASTEdit | FeatureTypedRefactor | FeatureRename | FeatureFormat | FeatureOrganizeImports | FeatureDiagnostics | FeatureProjectHooks | FeatureCompile | FeatureTest | FeatureSnapshot | FeatureRestore | FeatureAddDependency | FeatureScaffold | FeatureAssertions | FeatureRepairImports | FeatureReferences | FeatureStaging | FeaturePublication | FeatureDecisionSearch | FeatureDocumentIndex

// FeatureReferences reports references using the active module build scope.
const FeatureReferences FeatureSet = 1 << 18

// Has reports whether all bits in flag are present in f.
func (f FeatureSet) Has(flag FeatureSet) bool { return flag != 0 && f&flag == flag }

// Valid reports whether f contains at least one known feature and no unknown bits.
func (f FeatureSet) Valid() bool { return f != 0 && f & ^allFeatures == 0 }

// Descriptions returns the human-readable feature names in f.
func (f FeatureSet) Descriptions() []string {
	return describeBits(uint64(f), []bitDescription{
		{uint64(FeatureLookup), "lookup"},
		{uint64(FeatureOutline), "outline"},
		{uint64(FeatureInspect), "inspect"},
		{uint64(FeatureASTEdit), "AST edits"},
		{uint64(FeatureTypedRefactor), "typed refactoring"},
		{uint64(FeatureRename), "rename"},
		{uint64(FeatureFormat), "formatting"},
		{uint64(FeatureOrganizeImports), "organize imports"},
		{uint64(FeatureDiagnostics), "diagnostics"},
		{uint64(FeatureProjectHooks), "project hooks"},
		{uint64(FeatureCompile), "compile"},
		{uint64(FeatureTest), "test"},
		{uint64(FeatureSnapshot), "snapshot"},
		{uint64(FeatureRestore), "restore"},
		{uint64(FeatureAddDependency), "add dependency"},
		{uint64(FeatureScaffold), "scaffold"},
		{uint64(FeatureAssertions), "assertions"},
		{uint64(FeatureRepairImports), "repair imports"},
		{uint64(FeatureReferences), "references"},
		{uint64(FeatureStaging), "volatile staging"},
		{uint64(FeaturePublication), "publication"},
		{uint64(FeatureDecisionSearch), "decision search"},
		{uint64(FeatureDocumentIndex), "document index"},
	})
}

// GuaranteeSet lists the output and verification guarantees promised by a profile.
type GuaranteeSet uint16

// Guarantee bits name observable outcomes promised by a profile.
const (
	GuaranteeReadOnly GuaranteeSet = 1 << iota
	GuaranteeASTEdit
	GuaranteePackageOverlay
	GuaranteeModuleProof
	GuaranteeServerEdit
	GuaranteeDiagnostics
	GuaranteeDiagnosticDelta
	GuaranteeConflictChecked
	GuaranteeAtomicWrites
	GuaranteeVolatileState      GuaranteeSet = 1 << 10
	GuaranteePublicationOutcome GuaranteeSet = 1 << 11
)
const allGuarantees = GuaranteeReadOnly | GuaranteeASTEdit | GuaranteePackageOverlay | GuaranteeModuleProof | GuaranteeServerEdit | GuaranteeDiagnostics | GuaranteeDiagnosticDelta | GuaranteeConflictChecked | GuaranteeAtomicWrites | GuaranteeSelectedFileOnly | GuaranteeVolatileState | GuaranteePublicationOutcome

// GuaranteeSelectedFileOnly limits edits to the selected file.
const GuaranteeSelectedFileOnly GuaranteeSet = 1 << 9

// Has reports whether all bits in flag are present in g.
func (g GuaranteeSet) Has(flag GuaranteeSet) bool { return flag != 0 && g&flag == flag }

// Valid reports whether g contains at least one known guarantee and no unknown bits.
func (g GuaranteeSet) Valid() bool { return g != 0 && g & ^allGuarantees == 0 }

// Descriptions returns the human-readable guarantees in g.
func (g GuaranteeSet) Descriptions() []string {
	return describeBits(uint64(g), []bitDescription{
		{uint64(GuaranteeReadOnly), "read-only"},
		{uint64(GuaranteeASTEdit), "AST edit"},
		{uint64(GuaranteePackageOverlay), "package overlay"},
		{uint64(GuaranteeModuleProof), "module proof"},
		{uint64(GuaranteeServerEdit), "validated server edit without resource operations"},
		{uint64(GuaranteeDiagnostics), "diagnostics"},
		{uint64(GuaranteeDiagnosticDelta), "diagnostic delta"},
		{uint64(GuaranteeConflictChecked), "conflict checked"},
		{uint64(GuaranteeAtomicWrites), "atomic writes"},
		{uint64(GuaranteeSelectedFileOnly), "edits confined to the selected file"},
		{uint64(GuaranteeVolatileState), "volatile state only"},
		{uint64(GuaranteePublicationOutcome), "explicit publication outcome"},
	})
}

// CodeFaults is a bitmask describing syntax or type faults by source region.
type CodeFaults uint8

// Fault bits identify syntax and type failures by target and package region.
const (
	FaultTargetSyntax CodeFaults = 1 << iota
	FaultTargetType
	FaultSamePackageSyntax
	FaultSamePackageType
	FaultOtherPackageSyntax
	FaultOtherPackageType
)

// AllCodeFaults is the mask of every supported source-fault bit.
const AllCodeFaults CodeFaults = FaultTargetSyntax | FaultTargetType | FaultSamePackageSyntax | FaultSamePackageType | FaultOtherPackageSyntax | FaultOtherPackageType

// String returns the readable names of the fault bits in f.
func (f CodeFaults) String() string {
	if f == 0 {
		return "healthy"
	}
	parts := describeBits(uint64(f), []bitDescription{{uint64(FaultTargetSyntax), "target_syntax"}, {uint64(FaultTargetType), "target_type"}, {uint64(FaultSamePackageSyntax), "same_package_syntax"}, {uint64(FaultSamePackageType), "same_package_type"}, {uint64(FaultOtherPackageSyntax), "other_package_syntax"}, {uint64(FaultOtherPackageType), "other_package_type"}})
	if uint8(f)&^uint8(AllCodeFaults) != 0 {
		parts = append(parts, fmt.Sprintf("unknown_0x%02x", uint8(f)&^uint8(AllCodeFaults)))
	}
	return strings.Join(parts, "+")
}

// CombinationSet marks exact CodeFaults combinations by bit position; combinations are not closed under subsets or supersets.
type CombinationSet uint64

// Contains reports exact membership; it does not infer support from related masks.
func (s CombinationSet) Contains(f CodeFaults) bool {
	return f <= AllCodeFaults && (uint8(f)&^uint8(AllCodeFaults)) == 0 && uint64(s)&(uint64(1)<<f) != 0
}

// FaultSupport classifies one exact source-fault combination.
type FaultSupport uint8

// Fault status values classify one exact fault combination.
const (
	FaultUnknown FaultSupport = iota
	FaultSupported
	FaultRejected
	FaultConditional
)

// String returns the stable label for s.
func (s FaultSupport) String() string {
	switch s {
	case FaultUnknown:
		return "unknown"
	case FaultSupported:
		return "supported"
	case FaultRejected:
		return "rejected"
	case FaultConditional:
		return "conditional"
	default:
		return "invalid"
	}
}

// RecoveryCondition names the mechanism that can recover a conditional fault case; its zero value means no condition.
type RecoveryCondition string

// Recovery values describe allowed conditions for conditional fault support.
const (
	RecoveryNone           RecoveryCondition = ""
	RecoveryServerAnalysis RecoveryCondition = "server_analysis"
	RecoveryDiagnostics    RecoveryCondition = "diagnostics"
	RecoveryNormalization  RecoveryCondition = "normalization"
	RecoveryGoCommand      RecoveryCondition = "go_command"
	RecoveryBuildCommand   RecoveryCondition = "build_command"
	RecoveryProjectChecks  RecoveryCondition = "project_checks"
)

// Valid reports whether c is a declared recovery condition; zero means none.
func (c RecoveryCondition) Valid() bool {
	switch c {
	case RecoveryNone, RecoveryServerAnalysis, RecoveryDiagnostics, RecoveryNormalization, RecoveryGoCommand, RecoveryBuildCommand, RecoveryProjectChecks:
		return true
	default:
		return false
	}
}

// Description returns the readable recovery condition represented by c.
func (c RecoveryCondition) Description() string {
	switch c {
	case RecoveryServerAnalysis:
		return "the server supplies sufficient usable analysis"
	case RecoveryDiagnostics:
		return "current diagnostic coverage arrives successfully"
	case RecoveryNormalization:
		return "the requested normalizers accept the source or repair the relevant fault"
	case RecoveryGoCommand:
		return "the Go command succeeds with the available dependency graph"
	case RecoveryBuildCommand:
		return "the build tool can execute and report its outcome"
	case RecoveryProjectChecks:
		return "configured project checks can produce a complete report"
	case RecoveryNone:
		return ""
	default:
		return "unknown recovery condition"
	}
}

// BrokenCodeSupport classifies all 64 exact fault combinations with disjoint supported, rejected, and conditional masks.
type BrokenCodeSupport struct {
	Supported, Rejected, Conditional CombinationSet
	Condition                        RecoveryCondition
}

// FaultCase pairs one exact fault combination with its classification.
type FaultCase struct {
	Faults  CodeFaults
	Support FaultSupport
}

// Classify returns the explicit status of f without subset or superset inference.
func (b BrokenCodeSupport) Classify(f CodeFaults) (FaultSupport, error) {
	if err := b.Validate(); err != nil {
		return FaultUnknown, fmt.Errorf("invalid broken-code support: %w", err)
	}
	if uint8(f)&^uint8(AllCodeFaults) != 0 {
		return FaultUnknown, fmt.Errorf("invalid code fault mask 0x%02x", uint8(f))
	}
	bit := CombinationSet(uint64(1) << f)
	if b.Supported&bit != 0 {
		return FaultSupported, nil
	}
	if b.Rejected&bit != 0 {
		return FaultRejected, nil
	}
	if b.Conditional&bit != 0 {
		return FaultConditional, nil
	}
	return FaultUnknown, nil
}

// Cases enumerates all 64 exact fault combinations after validating the masks.
func (b BrokenCodeSupport) Cases() ([]FaultCase, error) {
	if err := b.Validate(); err != nil {
		return nil, fmt.Errorf("invalid broken-code support: %w", err)
	}
	out := make([]FaultCase, 64)
	for i := range out {
		f := CodeFaults(i)
		s, err := b.Classify(f)
		if err != nil {
			return nil, err
		}
		out[i] = FaultCase{Faults: f, Support: s}
	}
	return out, nil
}

// Validate rejects overlapping classifications and inconsistent recovery conditions.
func (b BrokenCodeSupport) Validate() error {
	if (b.Supported&b.Rejected)|(b.Supported&b.Conditional)|(b.Rejected&b.Conditional) != 0 {
		return fmt.Errorf("code fault classifications overlap")
	}
	if !b.Condition.Valid() {
		return fmt.Errorf("invalid recovery condition %q", b.Condition)
	}
	if b.Conditional != 0 && b.Condition == RecoveryNone {
		return fmt.Errorf("conditional code fault support requires a recovery condition")
	}
	if b.Conditional == 0 && b.Condition != RecoveryNone {
		return fmt.Errorf("recovery condition requires conditional code fault support")
	}
	return nil
}

// Description summarizes single-fault behavior and the mixed-fault rule.
func (b BrokenCodeSupport) Description() string {
	if err := b.Validate(); err != nil {
		return "invalid broken-code support: " + err.Error()
	}
	singles := []CodeFaults{0, FaultTargetSyntax, FaultTargetType, FaultSamePackageSyntax, FaultSamePackageType, FaultOtherPackageSyntax, FaultOtherPackageType}
	statuses := make([]string, 0, len(singles))
	for _, faults := range singles {
		support, err := b.Classify(faults)
		if err != nil {
			return "invalid broken-code support: " + err.Error()
		}
		statuses = append(statuses, faults.String()+"="+support.String())
	}
	description := strings.Join(statuses, ", ") + "; mixed faults follow the declared fault matrix"
	if b.Condition != RecoveryNone {
		description += "; conditional when " + b.Condition.Description()
	}
	return description
}

// DetailedDescription retains the complete fault matrix for reference documentation.
func (b BrokenCodeSupport) DetailedDescription() string {
	if err := b.Validate(); err != nil {
		return "invalid broken-code support: " + err.Error()
	}
	cases, err := b.Cases()
	if err != nil {
		return "invalid broken-code support: " + err.Error()
	}
	counts := map[FaultSupport]int{}
	for _, faultCase := range cases {
		counts[faultCase.Support]++
	}
	return fmt.Sprintf("%s (exact cases: supported=%d, rejected=%d, conditional=%d, unknown=%d; masks: supported=0x%016x, rejected=0x%016x, conditional=0x%016x; mixed-fault cases use the exact matrix with no subset or superset closure)", b.Description(), counts[FaultSupported], counts[FaultRejected], counts[FaultConditional], counts[FaultUnknown], uint64(b.Supported), uint64(b.Rejected), uint64(b.Conditional))
}

// ContractVariant describes one exact selector profile and its typed contract.
type ContractVariant struct {
	When           VariantSet
	AnalysisScope  AnalysisScope
	Preconditions  []Requirement
	Capabilities   FeatureSet
	Guarantees     GuaranteeSet
	ErrorTolerance BrokenCodeSupport
}

// PreconditionsDescription renders the typed preconditions for v.
func (v ContractVariant) PreconditionsDescription() string {
	parts := make([]string, len(v.Preconditions))
	for i, r := range v.Preconditions {
		parts[i] = r.Description()
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "; ")
}

// CapabilitiesDescription renders the features promised by v.
func (v ContractVariant) CapabilitiesDescription() string {
	return strings.Join(v.Capabilities.Descriptions(), ", ")
}

// GuaranteesDescription renders the outcomes promised by v.
func (v ContractVariant) GuaranteesDescription() string {
	return strings.Join(v.Guarantees.Descriptions(), ", ")
}

// Variant finds only the exact selector profile equal to when.
func (c OperationContract) Variant(when VariantSet) (ContractVariant, bool) {
	for _, v := range c.Variants {
		if v.When == when {
			return v, true
		}
	}
	return ContractVariant{}, false
}

// Validate checks that every selector has one complete, healthy-source contract.
func (c OperationContract) Validate() error {
	if len(c.Variants) == 0 {
		return fmt.Errorf("operation contract has no variants")
	}
	seen := make(map[VariantSet]bool, len(c.Variants))
	for i, v := range c.Variants {
		if !v.When.Valid() {
			return fmt.Errorf("variant %d has invalid selector %q", i, v.When)
		}
		if seen[v.When] {
			return fmt.Errorf("duplicate variant selector %q", v.When)
		}
		seen[v.When] = true
		if !v.AnalysisScope.Valid() {
			return fmt.Errorf("variant %q has invalid analysis scope %q", v.When, v.AnalysisScope)
		}
		if !v.Capabilities.Valid() {
			return fmt.Errorf("variant %q has empty or invalid capabilities", v.When)
		}
		if !v.Guarantees.Valid() {
			return fmt.Errorf("variant %q has empty or invalid guarantees", v.When)
		}
		for _, r := range v.Preconditions {
			if !r.Valid() {
				return fmt.Errorf("variant %q has invalid requirement %q", v.When, r.Kind)
			}
		}
		if err := v.ErrorTolerance.Validate(); err != nil {
			return fmt.Errorf("variant %q error tolerance: %w", v.When, err)
		}
		healthy, err := v.ErrorTolerance.Classify(0)
		if err != nil {
			return fmt.Errorf("variant %q error tolerance: %w", v.When, err)
		}
		if healthy != FaultSupported {
			return fmt.Errorf("variant %q must support healthy source", v.When)
		}
	}
	return nil
}

// Description renders each typed variant as a compact documentation line.
func (c OperationContract) Description() string {
	lines := make([]string, 0, len(c.Variants))
	for _, v := range c.Variants {
		lines = append(lines, fmt.Sprintf("Variant: %s. Analysis scope: %s. Capabilities: %s. Preconditions: %s. Guarantees: %s. Error tolerance: %s.", v.When, v.AnalysisScope.Description(), v.CapabilitiesDescription(), v.PreconditionsDescription(), v.GuaranteesDescription(), v.ErrorTolerance.Description()))
	}
	return strings.Join(lines, "\n")
}

type bitDescription struct {
	bit  uint64
	text string
}

func describeBits(value uint64, known []bitDescription) []string {
	out := make([]string, 0, len(known))
	for _, x := range known {
		if value&x.bit != 0 {
			out = append(out, x.text)
		}
	}
	return out
}
