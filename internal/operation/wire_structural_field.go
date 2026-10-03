// This file exposes one conservative typed Go struct-field insertion through CLI, MCP, receipts, and buffered batches.

package operation

import (
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"semedit/internal/astedit"
)

const insertFieldToolName = "semantic_insert_field"

func insertFieldStructuralDef() Def[map[string]any, structuralPlanResult] {
	placement := structuralStringParam("placement", "Placement: type_start or type_end, or relative to an existing target_symbol.", false, false, RolePolicy)
	placement.Schema.Enum = []any{"type_start", "type_end", "before_symbol", "after_symbol"}
	placement.Default = "type_end"
	params := []ParameterContract{
		structuralStringParam("file", "Target: existing Go file that declares the selected non-generic struct; copy the exact file from semantic_inspect_symbol.result.continuations[].file.", true, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].file",
			insertFieldToolName+".result.edit_receipt.continuations[].file"),
		structuralStringParam("in_type", "Selector: locally declared, private, non-generic struct owner; copy its exact in_type value from the same semantic_inspect_symbol.result.continuations[] record as file and expected_revision.", true, true, RoleSelector,
			"semantic_inspect_symbol.result.continuations[].in_type"),
		structuralStringParam("symbol", "Target: caller-chosen new qualified field identity in Type.Field form; its type prefix must match in_type and its simple field name must match source.", true, false, RoleTarget),
		structuralStringParam("source", "Replacement: exactly one named Go field declaration, optionally including its tag and field documentation comment; do not provide the enclosing struct.", true, false, RoleValue),
		placement,
		structuralStringParam("target_symbol", "Target: an existing qualified Type.Field anchor, required only for before_symbol or after_symbol; it must name a field currently declared on in_type.", false, true, RoleTarget),
		structuralStringParam("expected_revision", "Precondition: optional opaque full-file revision; copy from the same semantic_inspect_symbol.result.continuations[] record as file and in_type. A stale revision is refused.", false, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].expected_revision",
			insertFieldToolName+".result.edit_receipt.continuations[].expected_revision"),
	}
	output := structuralOutputContract(false)
	targetProperties := SymbolTargetSchema().Properties
	output.Continuations = append(output.Continuations,
		OutputContinuation{Path: "result.edit_receipt.continuations[].file", Schema: targetProperties["file"], Role: RoleTarget, Consumers: []string{insertFieldToolName}},
		OutputContinuation{Path: "result.edit_receipt.continuations[].symbol", Schema: targetProperties["symbol"], Role: RoleTarget, Consumers: []string{semanticRenameToolName, semanticInspectSymbolToolName}},
		OutputContinuation{Path: "result.edit_receipt.continuations[].expected_revision", Schema: targetProperties["expected_revision"], Role: RoleTarget, Consumers: []string{insertFieldToolName}},
	)
	return structuralPlanDef(structuralPlanSpec{
		Key: "insert_field", VariantVerb: "insert", VariantKind: "field", CLIName: "insert-field", MCPName: insertFieldToolName,
		Summary: "Use this tool instead of editing a struct declaration to add one named field to an existing private, non-generic Go struct. Use semantic_insert_construct with kind=field for the generic interface; in_type selects the owner.",
		Params:  params, Output: &output,
		ExampleRaw: map[string]any{"file": "config.go", "in_type": "config", "symbol": "config.timeout", "source": "timeout int", "placement": "type_end"},
		Plan: func(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
			inType := stringParam(raw, "in_type")
			fieldName, err := structuralMemberName(inType, stringParam(raw, "symbol"))
			if err != nil {
				return astedit.SourcePlan{}, err
			}
			owner, named, typeName, err := structuralFieldOwner(snapshot, inType)
			if err != nil {
				return astedit.SourcePlan{}, err
			}
			structure := named.Underlying().(*types.Struct)
			for field := range structure.Fields() {
				if field.Embedded() {
					return astedit.SourcePlan{}, fmt.Errorf("in_type %q has embedded fields outside this insertion's proof scope", inType)
				}
			}
			if sourceErr := structuralFieldSourceSet(snapshot, owner, named, typeName, fieldName); sourceErr != nil {
				return astedit.SourcePlan{}, sourceErr
			}
			targetField := ""
			if rawTarget, ok := raw["target_symbol"]; ok {
				text, valid := rawTarget.(string)
				if !valid {
					return astedit.SourcePlan{}, fmt.Errorf("target_symbol must be a string")
				}
				targetField, err = structuralMemberName(inType, text)
				if err != nil {
					return astedit.SourcePlan{}, fmt.Errorf("target_symbol: %w", err)
				}
			}
			return astedit.PlanInsertField(snapshot.Sources[snapshot.File], astedit.InsertFieldIntent{
				TypeName: typeName, FieldName: fieldName, FieldSource: stringParam(raw, "source"),
				Placement: astedit.FieldPlacement(stringParam(raw, "placement")), TargetField: targetField,
			})
		},
		Result: func(_ CallContext, raw map[string]any, _ astedit.SourcePlan, _ string) FileEditRes {
			return FileEditRes{Symbol: stringParam(raw, "symbol"), InType: stringParam(raw, "in_type"), Detail: "Inserted one field into the selected Go struct."}
		},
		Continuations: func(cc CallContext, snapshot *structuralSnapshot, raw map[string]any, plan astedit.SourcePlan) ([]SymbolTarget, error) {
			relative, err := filepath.Rel(cc.WorkDir, snapshot.File)
			if err != nil || !filepath.IsLocal(relative) {
				return nil, fmt.Errorf("derive field continuation path for %q: %w", snapshot.File, err)
			}
			revision := structuralRevision(plan.Source)
			return []SymbolTarget{{File: filepath.ToSlash(relative), Symbol: stringParam(raw, "symbol"), ExpectedRevision: &revision}}, nil
		},
		ResolveWithRoot: resolveStructuralFieldTarget,
	})
}

func structuralMemberName(owner, symbol string) (string, error) {
	prefix := owner + "."
	if !strings.HasPrefix(symbol, prefix) {
		return "", fmt.Errorf("qualified field target %q must be owned by in_type %q", symbol, owner)
	}
	name := strings.TrimPrefix(symbol, prefix)
	if name == "" || strings.Contains(name, ".") || !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() {
		return "", fmt.Errorf("qualified field target %q must end in one simple Go field name", symbol)
	}
	return name, nil
}
