// This file defines source-only plan results so AST editors can validate transformations without publishing them.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// OptionalString preserves missing, null, empty, and literal string values for schema inputs.
type OptionalString struct {
	Present bool
	Null    bool
	Value   string
}

type (
	// PlanSelection records the source node selected before a proposed structural edit.
	PlanSelection struct {
		Kind   string
		Owner  string
		Target string
		Start  int
		End    int
	}

	// ProofRequirement names semantic evidence that a coordinator must complete.
	ProofRequirement string

	// SourcePlan returns candidate source and explicit analysis gates without publishing it.
	SourcePlan struct {
		Source         []byte
		Selection      PlanSelection
		RequiredProofs []ProofRequirement
		Changed        bool
	}
)

// Proof requirements distinguish source coverage from type and control-flow analysis.
const (
	ProofCompleteSourceSet ProofRequirement = "complete_source_set"
	ProofTypedMemberSet    ProofRequirement = "typed_member_set"
	ProofUnkeyedLiteralSet ProofRequirement = "unkeyed_literal_scan"
	ProofTypedLiteral      ProofRequirement = "typed_literal_context"
	ProofBlockBinding      ProofRequirement = "block_binding_analysis"
	ProofControlFlowSet    ProofRequirement = "control_flow_analysis"
)

func newSourcePlan(before, after []byte, selection PlanSelection, proofs ...ProofRequirement) SourcePlan {
	return SourcePlan{
		Source:         bytes.Clone(after),
		Selection:      selection,
		RequiredProofs: append([]ProofRequirement(nil), proofs...),
		Changed:        !bytes.Equal(before, after),
	}
}

func parsePlanSource(source []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, nil, fmt.Errorf("parse source for structural plan: %w", err)
	}
	if file == nil {
		return nil, nil, fmt.Errorf("parse source for structural plan returned no file")
	}
	return fset, file, nil
}

type (
	// SymbolTarget names one exact live declaration target.
	SymbolTarget struct {
		File             string         `json:"file"`
		Symbol           string         `json:"symbol"`
		ExpectedRevision OptionalString `json:"expected_revision,omitzero"`
	}

	// ConstructSelector binds a node selector to its owner and file revision.
	ConstructSelector struct {
		File             string         `json:"file"`
		InFunction       OptionalString `json:"in_function,omitzero"`
		InDeclaration    OptionalString `json:"in_declaration,omitzero"`
		Kind             string         `json:"kind"`
		Discriminator    OptionalString `json:"discriminator,omitzero"`
		ConstructPath    OptionalString `json:"construct_path,omitzero"`
		ExpectedRevision *string        `json:"expected_revision,omitzero"`
	}

	// FieldSpec represents one declared struct field and optional tag.
	FieldSpec struct {
		Name string         `json:"name"`
		Type string         `json:"type"`
		Tag  OptionalString `json:"tag,omitzero"`
	}

	// MemberTarget names one resolved member without spelling reconstruction.
	MemberTarget struct {
		File   string `json:"file"`
		Symbol string `json:"symbol"`
	}

	// CallSiteSelector selects a source call independently of its callee target.
	CallSiteSelector struct {
		File             string  `json:"file"`
		InFunction       string  `json:"in_function"`
		Discriminator    string  `json:"discriminator"`
		ConstructPath    *string `json:"construct_path,omitzero"`
		ExpectedRevision *string `json:"expected_revision,omitzero"`
	}
)

func validateRevisionBoundSelector(source []byte, path, revision OptionalString) error {
	if path.Null && !path.Present || revision.Null && !revision.Present {
		return fmt.Errorf("null selector values must be marked present")
	}
	if path.Present && path.Null || revision.Present && revision.Null {
		return fmt.Errorf("selector path and revision cannot be null")
	}
	if path.Present && path.Value == "" {
		return fmt.Errorf("construct_path cannot be empty")
	}
	if revision.Present && revision.Value == "" {
		return fmt.Errorf("expected_revision cannot be empty")
	}
	if revision.Present && revision.Value != goReadRevision(source) {
		return fmt.Errorf("expected_revision does not match the selected source revision")
	}
	if path.Present && !revision.Present {
		return fmt.Errorf("construct_path requires its matching expected_revision")
	}
	return nil
}
