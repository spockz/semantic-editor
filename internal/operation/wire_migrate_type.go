// This file registers bounded Go alias-preserving type migration for the shared interfaces.

package operation

import (
	"context"
	"fmt"
	"path/filepath"

	"semedit/internal/astedit"
	"semedit/internal/backend"
)

// MigrateTypeReq requests a type-identity-preserving field spelling migration.
type MigrateTypeReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	Type             string
	ExpectedRevision string
}

// GetProjectContext returns the project selector used for registry dispatch.
func (r MigrateTypeReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the project selector after ambient context merging.
func (r *MigrateTypeReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var migrateTypeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target: existing Go file declaring the selected private struct field", Required: true, Role: RoleTarget, SourceFields: []string{sourceLookupFile, sourceRenameFile, semanticInspectSymbolToolName + ".result.continuations[].file", semanticMigrateTypeToolName + ".result.continuations[].file"}},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: qualified field name in Owner.field form; only unexported struct fields are supported", Required: true, Role: RoleTarget, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, semanticInspectSymbolToolName + ".result.continuations[].symbol", semanticMigrateTypeToolName + ".result.continuations[].symbol"}},
	{Name: "type", CLIName: "type", JSONName: "type", Type: ParamString, Description: "Value: unqualified Go type name in the same package or predeclared scope whose exact identity after alias resolution matches the field's current type", Required: true, Role: RoleValue},
	{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Description: "Guard: full-file revision copied from semantic_inspect_symbol, a resolved rename continuation, or a type-migration continuation for this target file", Role: RoleTarget, SourceFields: []string{semanticInspectSymbolToolName + ".result.continuations[].expected_revision", semanticMigrateTypeToolName + ".result.continuations[].expected_revision", sourceRenameExpectedRevision}},
}

func parseMigrateType(raw map[string]any) (MigrateTypeReq, error) {
	var req MigrateTypeReq
	if err := CheckParams(raw, migrateTypeParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	if req.Type, err = ParseString(raw, "type", "type", true); err != nil {
		return req, err
	}
	if value, exists := raw[wireExpectedRevision]; exists {
		text, ok := value.(string)
		if !ok || text == "" {
			return req, fmt.Errorf("expected_revision must be a non-empty string")
		}
		req.ExpectedRevision = text
	}
	return req, nil
}

func runMigrateType(ctx context.Context, cc CallContext, req MigrateTypeReq) (SemanticEditRes, error) {
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	plan, err := astedit.PlanAliasTypeMigrationOverlay(ctx, cc.WorkDir, req.File, req.Symbol, req.Type, req.ExpectedRevision)
	if err != nil {
		return SemanticEditRes{}, err
	}
	targetPath, err := filepath.Abs(filepath.Join(cc.WorkDir, req.File))
	if err != nil {
		return SemanticEditRes{}, fmt.Errorf("resolve type target path: %w", err)
	}
	expectedRevision := revisionHex(plan.Change.After)
	draft, err := semanticEditDraft(cc.WorkDir, "migrate-type", plan, []SymbolTarget{{File: absoluteToWorkPath(cc.WorkDir, targetPath), Symbol: req.Symbol, ExpectedRevision: &expectedRevision}})
	if err != nil {
		return SemanticEditRes{}, err
	}
	if err := astedit.ApplySemanticEdit(ctx, plan); err != nil {
		return SemanticEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return SemanticEditRes{Diff: plan.Diff, Delta: delta, DiagnosticStatus: diagnosticStatus, Draft: draft}, nil
}

func migrateTypeDef() Def[MigrateTypeReq, SemanticEditRes] {
	receiptSchema := EditReceiptSchema()
	return Def[MigrateTypeReq, SemanticEditRes]{
		Key:     "migrate_type",
		Summary: "Use this tool instead of text editing to migrate one unexported Go struct field between type names with identical type identity after alias resolution. Only one private field and one unqualified same-package or predeclared destination type are supported. Distinct named types, exported fields, conversions, imported selectors, and local variables are refused before the file changes.",
		Params:  migrateTypeParams, Level: LevelFile, CLIName: "migrate-type", MCPName: semanticMigrateTypeToolName,
		Parse:    parseMigrateType,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, MigrateTypeReq) (SemanticEditRes, error){backend.LanguageGo: runMigrateType},
		Format: func(res SemanticEditRes) (string, error) {
			text := "Successfully migrated the field type spelling. diagnostic_status: " + res.DiagnosticStatus + "."
			if res.DiagnosticStatus == "changed" {
				text = AppendDiagnosticDelta(text, res.Delta)
			}
			return text + "\n" + res.Diff, nil
		},
		ExampleRaw: map[string]any{"file": "config.go", "symbol": "settings.timeout", "type": "int64"},
		Output: &OperationOutputContract{ResultSchema: &receiptSchema, Continuations: []OutputContinuation{
			{Path: "result.continuations[].file", Schema: SymbolTargetSchema().Properties["file"], Role: RoleTarget, Consumers: []string{semanticMigrateTypeToolName}},
			{Path: "result.continuations[].symbol", Schema: SymbolTargetSchema().Properties["symbol"], Role: RoleTarget, Consumers: []string{semanticMigrateTypeToolName}},
			{Path: "result.continuations[].expected_revision", Schema: SymbolTargetSchema().Properties[wireExpectedRevision], Role: RoleTarget, Consumers: []string{semanticMigrateTypeToolName}},
		}},
		FinalizeOutput: finalizeMigrateTypeOutput,
		ProjectOutput:  func(res SemanticEditRes) (any, error) { return res.Receipt, nil },
		BufferedBatch:  true, Batchable: true,
	}
}

func finalizeMigrateTypeOutput(cc CallContext, _ MigrateTypeReq, res SemanticEditRes) (SemanticEditRes, error) {
	return finalizeSemanticEditResult(cc, res)
}
