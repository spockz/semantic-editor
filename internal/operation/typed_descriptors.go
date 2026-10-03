// This file provides named typed descriptors shared by semantic refactoring planners.
// Keeping them pure lets operation-specific planners validate intent without publishing source.

package operation

import (
	"fmt"
	"go/token"
	"path/filepath"
	"slices"
)

// CallSiteSelector addresses one call site in its owning function.

// ParameterRole classifies a parameter's effect on target selection or caller policy.

// OutputContinuation describes one complete live result field and its compatible consumers.

// OperationOutputContract declares a result value and typed live continuations.

// FileRevision binds one project-relative path to an opaque content revision.

// EditStatus records the actual working-buffer and disk publication outcome.

// EditPartiallyPublished means some publication completed before another part failed.
const EditPartiallyPublished EditStatus = "partially_published"

// EditPersistedWithDiagnostics means disk publication completed with diagnostics.
const EditPersistedWithDiagnostics EditStatus = "persisted_with_diagnostics"

// EditPersisted means the edit was persisted to disk.
const EditPersisted EditStatus = "persisted"

// EditAppliedWorking means the edit was applied to the active working buffer.
const EditAppliedWorking EditStatus = "applied_working"

// EditRefused means no edit was published.
const EditRefused EditStatus = "refused"

// RoleValue identifies a parameter carrying replacement data.
const RoleValue ParameterRole = "value"

// RoleSource identifies a parameter selecting source material.
const RoleSource ParameterRole = "source"

// RolePolicy identifies a parameter controlling planner behavior.
const RolePolicy ParameterRole = "policy"

// RoleSelector identifies a parameter selecting an existing identity.
const RoleSelector ParameterRole = "selector"

// RoleTarget identifies a parameter selecting an edit target.
const RoleTarget ParameterRole = "target"

// EditReceipt reports changed revisions, the complete analysis read set, and live targets.
type EditReceipt struct {
	SchemaVersion string         `json:"schema_version"`
	Status        EditStatus     `json:"status"`
	Operation     string         `json:"operation"`
	Changes       []FileRevision `json:"changes,omitempty"`
	ReadSet       []FileRevision `json:"read_set,omitempty"`
	Continuations []SymbolTarget `json:"continuations,omitempty"`
}

// EditStatus records the actual working-buffer and disk publication outcome.
type EditStatus string

// FileRevision binds one project-relative path to an opaque content revision.
type FileRevision struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

// OperationOutputContract declares a result value and typed live continuations.
type OperationOutputContract struct {
	ResultSchema  *ValueSchema         `json:"result_schema,omitempty"`
	Continuations []OutputContinuation `json:"continuations,omitempty"`
}

// OperationErrorOutputContract declares a private typed projection for known operation errors.
type OperationErrorOutputContract struct {
	ResultSchema *ValueSchema
}

// OutputContinuation describes one complete live result field and its compatible consumers.
type OutputContinuation struct {
	Field     string        `json:"field,omitempty"`
	Path      string        `json:"path,omitempty"`
	Schema    ValueSchema   `json:"schema"`
	Role      ParameterRole `json:"role"`
	Consumers []string      `json:"consumers"`
	Optional  bool          `json:"optional,omitempty"`
}

// ParameterRole classifies a parameter effect on target selection or caller policy.
type ParameterRole string

// CallSiteSelector addresses one call site in its owning function.
type CallSiteSelector struct {
	File             string  `json:"file"`
	InFunction       string  `json:"in_function"`
	Discriminator    string  `json:"discriminator"`
	ConstructPath    *string `json:"construct_path,omitempty"`
	ExpectedRevision *string `json:"expected_revision,omitempty"`
}

// ConstructSelector addresses one revision-bound construct in a declared owner.
type ConstructSelector struct {
	File             string  `json:"file"`
	InFunction       *string `json:"in_function,omitempty"`
	InDeclaration    *string `json:"in_declaration,omitempty"`
	Kind             string  `json:"kind"`
	Discriminator    *string `json:"discriminator,omitempty"`
	ConstructPath    *string `json:"construct_path,omitempty"`
	ExpectedRevision *string `json:"expected_revision,omitempty"`
}

// SymbolTarget identifies a live declaration at a project-relative path.
type SymbolTarget struct {
	File             string  `json:"file"`
	Symbol           string  `json:"symbol"`
	ExpectedRevision *string `json:"expected_revision,omitempty"`
}

// MemberMapping connects resolved source and target members.
type MemberMapping struct {
	SourceMember MemberTarget `json:"source_member"`
	TargetMember MemberTarget `json:"target_member"`
}

// MemberTarget identifies one resolved member without rebuilding its identity.
type MemberTarget struct {
	File   string `json:"file"`
	Symbol string `json:"symbol"`
}

// ParameterToFieldMapping maps an existing parameter index to a field shape.
type ParameterToFieldMapping struct {
	SourceIndex int       `json:"source_index"`
	Field       FieldSpec `json:"field"`
}

// FieldSpec declares one field name, type, and optional source tag.
type FieldSpec struct {
	Name string  `json:"name"`
	Type string  `json:"type"`
	Tag  *string `json:"tag,omitempty"`
}

// ResultSpec declares one function result type and optional result name.
type ResultSpec struct {
	Type string  `json:"type"`
	Name *string `json:"name,omitempty"`
}

// ParameterMapping describes one final parameter and its optional source flow.
type ParameterMapping struct {
	SourceIndex *int          `json:"source_index,omitempty"`
	Parameter   ParameterSpec `json:"parameter"`
	Argument    *string       `json:"argument,omitempty"`
}

// ParameterSpec declares a parameter name, type, and optional variadic mode.
type ParameterSpec struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Variadic *bool  `json:"variadic,omitempty"`
}

// ValidateParameterMappings checks parameter shapes and unique source indices.
func ValidateParameterMappings(mappings []ParameterMapping) error {
	seen := make(map[int]struct{}, len(mappings))
	for index, mapping := range mappings {
		if err := ValidateParameterSpec(mapping.Parameter); err != nil {
			return fmt.Errorf("parameter mapping %d: %w", index, err)
		}
		if mapping.SourceIndex != nil {
			if *mapping.SourceIndex < 0 {
				return fmt.Errorf("parameter mapping %d has negative source_index %d", index, *mapping.SourceIndex)
			}
			if _, ok := seen[*mapping.SourceIndex]; ok {
				return fmt.Errorf("source_index %d is mapped more than once", *mapping.SourceIndex)
			}
			seen[*mapping.SourceIndex] = struct{}{}
		}
		if mapping.Argument != nil && *mapping.Argument == "" {
			return fmt.Errorf("parameter mapping %d has an empty argument expression", index)
		}
	}
	return nil
}

// ValidateParameterToFieldMappings checks fields and unique source indices.
func ValidateParameterToFieldMappings(mappings []ParameterToFieldMapping) error {
	seen := make(map[int]struct{}, len(mappings))
	for index, mapping := range mappings {
		if mapping.SourceIndex < 0 {
			return fmt.Errorf("field mapping %d has negative source_index %d", index, mapping.SourceIndex)
		}
		if _, ok := seen[mapping.SourceIndex]; ok {
			return fmt.Errorf("source_index %d is mapped to more than one field", mapping.SourceIndex)
		}
		seen[mapping.SourceIndex] = struct{}{}
		if err := validateFieldSpec(mapping.Field); err != nil {
			return fmt.Errorf("field mapping %d: %w", index, err)
		}
	}
	return nil
}

// ValidateMemberMapping requires resolved source and target member identities.
func ValidateMemberMapping(mapping MemberMapping) error {
	if err := ValidateMemberTarget(mapping.SourceMember); err != nil {
		return fmt.Errorf("source_member: %w", err)
	}
	if err := ValidateMemberTarget(mapping.TargetMember); err != nil {
		return fmt.Errorf("target_member: %w", err)
	}
	return nil
}

// ValidateMemberTarget rejects incomplete member identities.
func ValidateMemberTarget(target MemberTarget) error {
	if target.File == "" {
		return fmt.Errorf("member target has an empty file")
	}
	if target.Symbol == "" {
		return fmt.Errorf("member target has an empty symbol")
	}
	return nil
}

// ValidateSymbolTarget rejects incomplete or malformed live target data.
func ValidateSymbolTarget(target SymbolTarget) error {
	if target.File == "" {
		return fmt.Errorf("symbol target has an empty file")
	}
	if target.Symbol == "" {
		return fmt.Errorf("symbol target has an empty symbol")
	}
	if target.ExpectedRevision != nil && *target.ExpectedRevision == "" {
		return fmt.Errorf("symbol target expected_revision is empty")
	}
	return nil
}

// ValidateConstructSelector requires one owner and binds paths to a revision.
func ValidateConstructSelector(selector ConstructSelector) error {
	if selector.File == "" {
		return fmt.Errorf("construct selector has an empty file")
	}
	if selector.Kind == "" {
		return fmt.Errorf("construct selector has an empty kind")
	}
	owners := 0
	if selector.InFunction != nil {
		if *selector.InFunction == "" {
			return fmt.Errorf("construct selector in_function is empty")
		}
		owners++
	}
	if selector.InDeclaration != nil {
		if *selector.InDeclaration == "" {
			return fmt.Errorf("construct selector in_declaration is empty")
		}
		owners++
	}
	if owners != 1 {
		return fmt.Errorf("construct selector requires exactly one owner selector")
	}
	if selector.Discriminator != nil && *selector.Discriminator == "" {
		return fmt.Errorf("construct selector discriminator is empty")
	}
	if selector.ConstructPath != nil {
		if *selector.ConstructPath == "" {
			return fmt.Errorf("construct selector construct_path is empty")
		}
		if selector.ExpectedRevision == nil || *selector.ExpectedRevision == "" {
			return fmt.Errorf("construct_path requires a non-empty expected_revision")
		}
	}
	if selector.ExpectedRevision != nil && *selector.ExpectedRevision == "" {
		return fmt.Errorf("construct selector expected_revision is empty")
	}
	return nil
}

// ValidateCallSiteSelector requires an exact call owner and revision-bound path.
func ValidateCallSiteSelector(selector CallSiteSelector) error {
	if selector.File == "" || selector.InFunction == "" || selector.Discriminator == "" {
		return fmt.Errorf("call-site selector requires file, in_function, and discriminator")
	}
	if selector.ConstructPath != nil {
		if *selector.ConstructPath == "" {
			return fmt.Errorf("call-site selector construct_path is empty")
		}
		if selector.ExpectedRevision == nil || *selector.ExpectedRevision == "" {
			return fmt.Errorf("construct_path requires a non-empty expected_revision")
		}
	}
	if selector.ExpectedRevision != nil && *selector.ExpectedRevision == "" {
		return fmt.Errorf("call-site selector expected_revision is empty")
	}
	return nil
}

// ValidateParameterSpec checks the declared parameter name and type.
func ValidateParameterSpec(parameter ParameterSpec) error {
	if parameter.Name == "" || !token.IsIdentifier(parameter.Name) {
		return fmt.Errorf("invalid parameter name %q", parameter.Name)
	}
	if parameter.Type == "" {
		return fmt.Errorf("parameter %q has an empty type", parameter.Name)
	}
	return nil
}

// ValidateResultSpec checks the result type and any declared result name.
func ValidateResultSpec(result ResultSpec) error {
	if result.Type == "" {
		return fmt.Errorf("result has an empty type")
	}
	if result.Name != nil && (*result.Name == "" || !token.IsIdentifier(*result.Name)) {
		return fmt.Errorf("invalid result name %q", *result.Name)
	}
	return nil
}

// ValidateFieldSpec checks the declared field name and type.
func ValidateFieldSpec(field FieldSpec) error {
	return validateFieldSpec(field)
}

// ParameterSpecSchema declares the closed object shape of one parameter.
func ParameterSpecSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"name":     text,
		"type":     text,
		"variadic": {Kind: SchemaBoolean},
	}, Required: []string{"name", "type"}}
}

// ParameterMappingSchema declares one closed source-parameter mapping.
func ParameterMappingSchema() ValueSchema {
	minimum := float64(0)
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{wireSourceIndex: {Kind: SchemaInteger, Minimum: &minimum}, "parameter": ParameterSpecSchema(), "argument": {Kind: SchemaString}}, Required: []string{"parameter"}}
}

// ResultSpecSchema declares one closed function-result shape.
func ResultSpecSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"type": text,
		"name": text,
	}, Required: []string{"type"}}
}

// FieldSpecSchema declares one closed struct-field shape.
func FieldSpecSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"name": text,
		"type": text,
		"tag":  {Kind: SchemaString},
	}, Required: []string{"name", "type"}}
}

// ParameterToFieldMappingSchema declares one closed parameter-to-field mapping.
func ParameterToFieldMappingSchema() ValueSchema {
	minimum := float64(0)
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{wireSourceIndex: {Kind: SchemaInteger, Minimum: &minimum}, "field": FieldSpecSchema()}, Required: []string{wireSourceIndex, "field"}}
}

// MemberTargetSchema declares one closed source or target member identity.
func MemberTargetSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file":   text,
		"symbol": text,
	}, Required: []string{"file", "symbol"}}
}

// MemberMappingSchema declares a paired source and target member mapping.
func MemberMappingSchema() ValueSchema {
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"source_member": MemberTargetSchema(),
		"target_member": MemberTargetSchema(),
	}, Required: []string{"source_member", "target_member"}}
}

// SymbolTargetSchema declares a live symbol identity.
func SymbolTargetSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file":               text,
		"symbol":             text,
		wireExpectedRevision: text,
	}, Required: []string{"file", "symbol"}}
}

// ConstructSelectorSchema declares a closed construct selector.
func ConstructSelectorSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file":               text,
		wireInFunction:       text,
		"in_declaration":     text,
		"kind":               text,
		wireDiscriminator:    text,
		"construct_path":     text,
		wireExpectedRevision: text,
	}, Required: []string{"file", "kind"}}
}

// CallSiteSelectorSchema declares a closed call-site selector.
func CallSiteSelectorSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"file": text, wireInFunction: text, wireDiscriminator: text, "construct_path": text, wireExpectedRevision: text}, Required: []string{"file", wireInFunction, wireDiscriminator}}
}

// FileRevisionSchema describes one project-relative path and its opaque revision.
func FileRevisionSchema() ValueSchema {
	text := nonEmptyStringSchema()
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"path":     text,
		"revision": text,
	}, Required: []string{"path", "revision"}}
}

// EditReceiptSchema describes an outcome without implying batch-wide atomicity.
func EditReceiptSchema() ValueSchema {
	version := ValueSchema{Kind: SchemaString, Enum: []any{"semedit.edit-receipt/v1"}}
	status := ValueSchema{Kind: SchemaString, Enum: []any{string(EditNoOp), string(EditRefused), string(EditAppliedWorking), string(EditPersisted), string(EditPersistedWithDiagnostics), string(EditPartiallyPublished)}}
	text := nonEmptyStringSchema()
	revisions := ValueSchema{Kind: SchemaArray, Items: new(FileRevisionSchema())}
	continuations := ValueSchema{Kind: SchemaArray, Items: new(SymbolTargetSchema())}
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"schema_version":       version,
		"status":               status,
		"operation":            text,
		"changes":              revisions,
		"read_set":             revisions,
		errorContinuationField: continuations,
	}, Required: []string{"schema_version", "status", "operation"}}
}

// ValidateOutputContract checks that declared continuations identify complete live projections.
func ValidateOutputContract(contract OperationOutputContract) error {
	if contract.ResultSchema != nil {
		if err := ValidateSchema(*contract.ResultSchema); err != nil {
			return fmt.Errorf("result schema: %w", err)
		}
	}
	seenFields := make(map[string]struct{}, len(contract.Continuations))
	for index, continuation := range contract.Continuations {
		if (continuation.Field == "") == (continuation.Path == "") {
			return fmt.Errorf("continuation %d must declare exactly one field or result path", index)
		}
		identity := continuation.Field
		if continuation.Path != "" {
			identity = continuation.Path
			if contract.ResultSchema == nil {
				return fmt.Errorf("continuation %q requires a result schema", continuation.Path)
			}
			fieldSchema, err := SchemaAtResultPath(*contract.ResultSchema, continuation.Path)
			if err != nil {
				return fmt.Errorf("continuation path %q: %w", continuation.Path, err)
			}
			if !SchemasEquivalent(fieldSchema, continuation.Schema) {
				return fmt.Errorf("continuation path %q schema disagrees with its result field", continuation.Path)
			}
		}
		if continuation.Field != "" {
			if contract.ResultSchema == nil {
				return fmt.Errorf("continuation %q requires a result schema", continuation.Field)
			}
			fieldSchema, err := SchemaAtResultPath(*contract.ResultSchema, "result."+continuation.Field)
			if err != nil {
				return fmt.Errorf("continuation field %q: %w", continuation.Field, err)
			}
			if !SchemasEquivalent(fieldSchema, continuation.Schema) {
				return fmt.Errorf("continuation field %q schema disagrees with its result field", continuation.Field)
			}
			if continuation.Optional && slices.Contains(contract.ResultSchema.Required, continuation.Field) {
				return fmt.Errorf("optional continuation field %q is required by its result schema", continuation.Field)
			}
		}
		if _, ok := seenFields[identity]; ok {
			return fmt.Errorf("continuation %q is repeated", identity)
		}
		seenFields[identity] = struct{}{}
		if err := ValidateSchema(continuation.Schema); err != nil {
			return fmt.Errorf("continuation %q schema: %w", continuation.Field, err)
		}
		if continuation.Role != RoleTarget && continuation.Role != RoleSelector {
			return fmt.Errorf("continuation %q has non-live role %q", continuation.Field, continuation.Role)
		}
		if len(continuation.Consumers) == 0 {
			return fmt.Errorf("continuation %q has no consumer tools", continuation.Field)
		}
		seenConsumers := make(map[string]struct{}, len(continuation.Consumers))
		for _, consumer := range continuation.Consumers {
			if consumer == "" {
				return fmt.Errorf("continuation %q has an empty consumer name", continuation.Field)
			}
			if _, ok := seenConsumers[consumer]; ok {
				return fmt.Errorf("continuation %q repeats consumer %q", continuation.Field, consumer)
			}
			seenConsumers[consumer] = struct{}{}
		}
	}
	return nil
}

// ValidateEditReceipt rejects outcomes that cannot truthfully expose live targets.
func ValidateEditReceipt(receipt EditReceipt) error {
	if receipt.SchemaVersion != "semedit.edit-receipt/v1" {
		return fmt.Errorf("unsupported edit receipt schema version %q", receipt.SchemaVersion)
	}
	switch receipt.Status {
	case EditNoOp, EditRefused, EditAppliedWorking, EditPersisted, EditPersistedWithDiagnostics, EditPartiallyPublished:
	default:
		return fmt.Errorf("unsupported edit receipt status %q", receipt.Status)
	}
	if receipt.Operation == "" {
		return fmt.Errorf("edit receipt operation is empty")
	}
	if receipt.Status == EditRefused && (len(receipt.Changes) > 0 || len(receipt.Continuations) > 0) {
		return fmt.Errorf("refused receipt cannot expose changes or live continuations")
	}
	if receipt.Status == EditNoOp && len(receipt.Changes) > 0 {
		return fmt.Errorf("no-op receipt cannot expose changes")
	}
	if err := validateFileRevisions(receipt.Changes, "changes"); err != nil {
		return err
	}
	if err := validateFileRevisions(receipt.ReadSet, "read_set"); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(receipt.Continuations))
	for index, continuation := range receipt.Continuations {
		if err := ValidateSymbolTarget(continuation); err != nil {
			return fmt.Errorf("continuation %d: %w", index, err)
		}
		identity := continuation.File + "\x00" + continuation.Symbol
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("live continuation %q is repeated", continuation.Symbol)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validateFieldSpec(field FieldSpec) error {
	if field.Name == "" || !token.IsIdentifier(field.Name) {
		return fmt.Errorf("invalid field name %q", field.Name)
	}
	if field.Type == "" {
		return fmt.Errorf("field %q has an empty type", field.Name)
	}
	return nil
}

func nonEmptyStringSchema() ValueSchema {
	minimum := 1
	return ValueSchema{Kind: SchemaString, MinLength: &minimum}
}

func validateFileRevisions(revisions []FileRevision, label string) error {
	seen := make(map[string]struct{}, len(revisions))
	for index, revision := range revisions {
		if revision.Path == "" || !filepath.IsLocal(filepath.FromSlash(revision.Path)) {
			return fmt.Errorf("%s entry %d has invalid project-relative path %q", label, index, revision.Path)
		}
		if revision.Revision == "" {
			return fmt.Errorf("%s entry %d has an empty revision", label, index)
		}
		if _, ok := seen[revision.Path]; ok {
			return fmt.Errorf("%s repeats path %q", label, revision.Path)
		}
		seen[revision.Path] = struct{}{}
	}
	return nil
}

// EditNoOp means the requested semantic state already held and no edit was published.
const EditNoOp EditStatus = "no_op"
