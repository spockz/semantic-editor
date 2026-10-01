// This file provides named typed descriptors shared by semantic refactoring planners.
// Keeping them pure lets operation-specific planners validate intent without publishing source.

package operation

import (
	"fmt"
	"go/token"
)

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

func validateFieldSpec(field FieldSpec) error {
	if field.Name == "" || !token.IsIdentifier(field.Name) {
		return fmt.Errorf("invalid field name %q", field.Name)
	}
	if field.Type == "" {
		return fmt.Errorf("field %q has an empty type", field.Name)
	}
	return nil
}
