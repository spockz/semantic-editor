// This file exposes one conservative typed Go struct-field insertion through CLI, MCP, receipts, and buffered batches.

package operation

import (
	"context"
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const insertFieldToolName = "semantic_insert_field"

func insertFieldStructuralDef() Def[map[string]any, structuralPlanResult] {
	placement := structuralStringParam("placement", "Placement: type_start or type_end, or relative to an existing target_symbol; Java type_end places fields by static/instance tier and visibility.", false, false, RolePolicy)
	placement.Schema.Enum = []any{"type_start", "type_end", "before_symbol", "after_symbol"}
	placement.Default = "type_end"
	params := make([]ParameterContract, 0, 7+len(readProjectParams))
	params = append(params, []ParameterContract{
		structuralStringParam("file", "Target: existing Go or Java source file that declares the selected struct/class; copy the exact file from semantic_inspect_symbol.result.continuations[].file.", true, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].file",
			insertFieldToolName+".result.edit_receipt.continuations[].file"),
		structuralStringParam("in_type", "Selector: existing Go struct or Java class/record owner; Java may omit this only when exactly one top-level type is declared in the selected file.", false, true, RoleSelector,
			"semantic_inspect_symbol.result.continuations[].in_type"),
		structuralStringParam("symbol", "Target: caller-chosen new qualified field identity in Type.Field form; its type prefix must match in_type and its simple field name must match source.", true, false, RoleTarget),
		structuralStringParam("source", "Replacement: exactly one named Go struct field or Java field declaration; do not provide the enclosing type.", true, false, RoleValue),
		placement,
		structuralStringParam("target_symbol", "Target: an existing qualified Type.Field anchor, required only for before_symbol or after_symbol; it must name a field currently declared on in_type.", false, true, RoleTarget),
		structuralStringParam("expected_revision", "Precondition: optional opaque full-file revision; copy from the same semantic_inspect_symbol.result.continuations[] record as file and in_type. A stale revision is refused.", false, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].expected_revision",
			insertFieldToolName+".result.edit_receipt.continuations[].expected_revision"),
	}...)
	output := structuralOutputContract(false)
	targetProperties := SymbolTargetSchema().Properties
	output.Continuations = append(output.Continuations,
		OutputContinuation{Path: "result.edit_receipt.continuations[].file", Schema: targetProperties["file"], Role: RoleTarget, Consumers: []string{insertFieldToolName}},
		OutputContinuation{Path: "result.edit_receipt.continuations[].symbol", Schema: targetProperties["symbol"], Role: RoleTarget, Consumers: []string{semanticRenameToolName, semanticInspectSymbolToolName}},
		OutputContinuation{Path: "result.edit_receipt.continuations[].expected_revision", Schema: targetProperties["expected_revision"], Role: RoleTarget, Consumers: []string{insertFieldToolName}},
	)
	params = append(params, readProjectParams...)
	return structuralPlanDef(structuralPlanSpec{
		Key: "insert_field", VariantVerb: "insert", VariantKind: "field", CLIName: "insert-field", MCPName: insertFieldToolName,
		Summary: "Use this tool instead of editing a type declaration to add one named field to an existing private, non-generic Go struct or a Java class, enum, or record. Java type_end insertion respects member category and visibility ordering, then formats and verifies through JDT LS.",
		Params:  params, Output: &output,
		JavaHandler: runJavaInsertFieldStructural,
		ExampleRaw:  map[string]any{"file": "config.go", "in_type": "config", "symbol": "config.timeout", "source": "timeout int", "placement": "type_end"},
		Plan: func(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
			inType := stringParam(raw, "in_type")
			owner, named, typeName, err := structuralFieldOwner(snapshot, inType)
			if err != nil {
				return astedit.SourcePlan{}, err
			}
			fieldName, err := structuralMemberName(inType, stringParam(raw, "symbol"))
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

func runJavaInsertFieldStructural(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file := stringParam(raw, "file")
	project, err := parseReadProject(raw, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	if project.Language != backend.LanguageJava {
		return structuralPlanResult{}, fmt.Errorf("java field insertion requires language=java")
	}
	owner := stringParam(raw, "in_type")
	symbol := stringParam(raw, "symbol")
	if owner == "" {
		source, readErr := sourcebuf.ReadFile(ctx, resolveWorkPath(cc.WorkDir, file))
		if readErr != nil {
			return structuralPlanResult{}, fmt.Errorf("read Java field target: %w", readErr)
		}
		topLevel, inferErr := javacst.SoleJavaTopLevelType(source)
		if inferErr != nil {
			return structuralPlanResult{}, inferErr
		}
		separator := strings.LastIndex(symbol, ".")
		if separator < 1 || symbol[:separator] != topLevel {
			return structuralPlanResult{}, fmt.Errorf("java field symbol %q must be qualified by sole top-level type %q", symbol, topLevel)
		}
		owner = topLevel
	}
	name, err := javaStructuralMemberName(owner, symbol)
	if err != nil {
		return structuralPlanResult{}, err
	}
	sourceSnippet := stringParam(raw, "source")
	placement := stringParam(raw, "placement")
	targetField := ""
	if rawTarget, ok := raw["target_symbol"]; ok {
		text, valid := rawTarget.(string)
		if !valid {
			return structuralPlanResult{}, fmt.Errorf("target_symbol must be a string")
		}
		if text != "" {
			targetField, err = javaStructuralMemberName(owner, text)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("target_symbol: %w", err)
			}
		}
	}
	var expectedRevision *string
	if value, ok := raw["expected_revision"]; ok {
		revision, valid := value.(string)
		if !valid || revision == "" {
			return structuralPlanResult{}, fmt.Errorf("expected_revision must be a non-empty string")
		}
		expectedRevision = &revision
	}
	root, absoluteFile, before, after, err := runJavaInspectedSourceEdit(ctx, cc, project, file, owner, "field insertion", expectedRevision, true, func(ownerSource []byte) ([]byte, error) {
		return javacst.InsertJavaField(ownerSource, []byte(sourceSnippet), name, placement, targetField)
	})
	if err != nil {
		return structuralPlanResult{}, err
	}
	return javaInsertFieldResult(root, file, owner, symbol, absoluteFile, before, after)
}

func javaInsertFieldResult(root, file, owner, symbol, absoluteFile string, before, after []byte) (structuralPlanResult, error) {
	relative, err := filepath.Rel(root, absoluteFile)
	if err != nil || !filepath.IsLocal(relative) {
		return structuralPlanResult{}, fmt.Errorf("derive Java field receipt path for %q: %w", absoluteFile, err)
	}
	relative = filepath.ToSlash(relative)
	display := displayWorkPath(root, file)
	revision := structuralRevision(after)
	detail := "Inserted one Java field into the selected type."
	edit := FileEditRes{
		File: absoluteFile, Display: display, Symbol: symbol, InType: owner,
		Diff: astedit.UnifiedDiff(display, string(before), string(after)), Detail: detail,
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
	}
	draft := EditReceiptDraft{
		Operation: "insert_field", Status: EditPersisted,
		ReadSet:        []FileRevision{{Path: relative, Revision: structuralRevision(before)}},
		PlannedChanges: []FileRevision{{Path: relative, Revision: revision}},
		Continuations:  []SymbolTarget{{File: relative, Symbol: symbol, ExpectedRevision: &revision}},
	}
	return structuralPlanResult{Edit: edit, Draft: draft}, nil
}

func javaStructuralMemberName(owner, symbol string) (string, error) {
	prefix := owner + "."
	if !strings.HasPrefix(symbol, prefix) {
		return "", fmt.Errorf("qualified Java field target %q must be owned by in_type %q", symbol, owner)
	}
	name := strings.TrimPrefix(symbol, prefix)
	if name == "" || strings.Contains(name, ".") {
		return "", fmt.Errorf("qualified Java field target %q must end in one field name", symbol)
	}
	if err := javacst.ValidateJavaIdentifier(name); err != nil {
		return "", err
	}
	return name, nil
}
