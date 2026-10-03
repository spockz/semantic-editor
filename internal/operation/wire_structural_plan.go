// This file registers bounded structural edits that plan from current Go source and apply through the shared source buffer.

package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/astedit"
	"semedit/internal/backend"
)

type structuralPlanSpec struct {
	Key                  string
	CLIName              string
	MCPName              string
	Summary              string
	Params               []ParameterContract
	ExampleRaw           map[string]any
	Output               *OperationOutputContract
	Plan                 func(*structuralSnapshot, map[string]any) (astedit.SourcePlan, error)
	Result               func(CallContext, map[string]any, astedit.SourcePlan, string) FileEditRes
	Continuations        func(CallContext, *structuralSnapshot, map[string]any, astedit.SourcePlan) ([]SymbolTarget, error)
	ResolveContinuations ContinuationResolver
	ResolveWithRoot      func(context.Context, string, SymbolTarget) (SymbolTarget, bool, error)
}

type structuralPlanResult struct {
	Edit           FileEditRes
	Draft          EditReceiptDraft
	Resolve        ContinuationResolver
	Receipt        EditReceipt
	FunctionTarget *SymbolTarget
}

func structuralPlanDef(spec structuralPlanSpec) Def[map[string]any, structuralPlanResult] {
	run := func(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
		return runStructuralPlan(ctx, cc, raw, spec)
	}
	return Def[map[string]any, structuralPlanResult]{
		Key: spec.Key, Summary: spec.Summary, Params: spec.Params, Level: LevelFile,
		CLIName: spec.CLIName, MCPName: spec.MCPName, Batchable: true, BufferedBatch: true,
		ExampleRaw: spec.ExampleRaw,
		Parse:      func(raw map[string]any) (map[string]any, error) { return maps.Clone(raw), nil },
		Handlers:   map[backend.LanguageID]func(context.Context, CallContext, map[string]any) (structuralPlanResult, error){backend.LanguageGo: run},
		Format:     func(result structuralPlanResult) (string, error) { return formatFileEdit(result.Edit) },
		Output:     spec.Output,
		FinalizeOutput: func(cc CallContext, _ map[string]any, result structuralPlanResult) (structuralPlanResult, error) {
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			receipt, err := FinalizeEditReceipt(ctx, cc.WorkDir, cc.InBatch, result.Draft, result.Resolve)
			if err != nil {
				return structuralPlanResult{}, fmt.Errorf("finalize structural edit receipt: %w", err)
			}
			result.Receipt = receipt
			return result, nil
		},
		ProjectOutput: func(result structuralPlanResult) (any, error) {
			projected := map[string]any{"result": result.Edit.Detail, "edit_receipt": result.Receipt}
			if result.Receipt.Status != EditAppliedWorking {
				if result.Edit.Display != "" {
					projected["file"] = result.Edit.Display
				}
				if result.Edit.Symbol != "" {
					projected["symbol"] = result.Edit.Symbol
				}
				if result.Edit.InType != "" {
					projected["in_type"] = result.Edit.InType
				}
				if result.Edit.InFunction != "" {
					projected[wireInFunction] = result.Edit.InFunction
				}
				if result.Edit.Discriminator != "" {
					projected[wireDiscriminator] = result.Edit.Discriminator
				}
			}
			if result.Edit.Diff != "" {
				projected["diff"] = result.Edit.Diff
			}
			return projected, nil
		},
	}
}

func runStructuralPlan(ctx context.Context, cc CallContext, raw map[string]any, spec structuralPlanSpec) (structuralPlanResult, error) {
	snapshot, err := prepareStructuralSnapshot(ctx, cc, stringParam(raw, "file"))
	if err != nil {
		return structuralPlanResult{}, err
	}
	before := snapshot.Sources[snapshot.File]
	if expected, ok := raw["expected_revision"]; ok {
		revision, valid := expected.(string)
		if !valid || revision == "" {
			return structuralPlanResult{}, fmt.Errorf("expected_revision must be a non-empty string")
		}
		if revision != structuralRevision(before) {
			return structuralPlanResult{}, fmt.Errorf("expected_revision does not match the current selected-file revision")
		}
	}
	finishDelta := surroundingDelta(snapshot.Ctx, cc.WorkDir, cc.DeferVerification)
	plan, err := spec.Plan(snapshot, raw)
	if err != nil {
		return structuralPlanResult{}, err
	}
	if plan.Changed {
		if err := snapshot.validateOutput(plan.Source); err != nil {
			return structuralPlanResult{}, fmt.Errorf("validate structural plan output: %w", err)
		}
	}
	changed, err := snapshot.applyOutput(plan.Source)
	if err != nil {
		return structuralPlanResult{}, err
	}
	delta, diagnosticStatus := finishDelta()
	result := spec.Result(cc, raw, plan, diagnosticStatus)
	result.File = snapshot.File
	result.Display = displayWorkPath(cc.WorkDir, stringParam(raw, "file"))
	result.Diff = astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, stringParam(raw, "file")), string(before), string(plan.Source))
	result.Delta = delta
	result.DiagnosticStatus = diagnosticStatus
	result.HasDelta = true
	if !changed {
		result.Detail = "No source change was needed."
	}
	readSet, err := structuralReadSet(cc.WorkDir, snapshot)
	if err != nil {
		return structuralPlanResult{}, err
	}
	draft := EditReceiptDraft{Operation: spec.Key, Status: EditPersisted, ReadSet: readSet}
	if result.DiagnosticStatus == "changed" {
		draft.Status = EditPersistedWithDiagnostics
	}
	if changed {
		draft.PlannedChanges = []FileRevision{{Path: result.Display, Revision: structuralRevision(plan.Source)}}
	}
	if spec.Continuations != nil {
		draft.Continuations, err = spec.Continuations(cc, snapshot, raw, plan)
		if err != nil {
			return structuralPlanResult{}, err
		}
	}
	resolve := spec.ResolveContinuations
	if spec.ResolveWithRoot != nil {
		resolve = func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
			return spec.ResolveWithRoot(resolveCtx, cc.WorkDir, candidate)
		}
	}
	return structuralPlanResult{Edit: result, Draft: draft, Resolve: resolve}, nil
}

func structuralReadSet(workDir string, snapshot *structuralSnapshot) ([]FileRevision, error) {
	readSet := make([]FileRevision, 0, len(snapshot.Sources))
	for path, content := range snapshot.Sources {
		relative, err := filepath.Rel(workDir, path)
		if err != nil {
			return nil, fmt.Errorf("structural read-set path %q is outside workspace %q", path, workDir)
		}
		if !filepath.IsLocal(relative) {
			continue
		}
		readSet = append(readSet, FileRevision{Path: filepath.ToSlash(relative), Revision: structuralRevision(content)})
	}
	slices.SortFunc(readSet, func(left, right FileRevision) int { return strings.Compare(left.Path, right.Path) })
	return readSet, nil
}

func stringParam(raw map[string]any, name string) string {
	value, _ := raw[name].(string)
	return value
}

func structuralRevision(source []byte) string {
	digest := sha256.Sum256(source)
	return hex.EncodeToString(digest[:])
}

func structuralStringParam(name, description string, required, allowEmpty bool, role ParameterRole, sources ...string) ParameterContract {
	schema := ValueSchema{Kind: SchemaString}
	if !allowEmpty {
		minimum := 1
		schema.MinLength = &minimum
	}
	return ParameterContract{Name: name, JSONName: name, CLIName: strings.ReplaceAll(name, "_", "-"), Type: ParamString, Description: description, Required: required, SourceFields: sources, Schema: &schema, Role: role}
}

func setDocCommentStructuralDef() Def[map[string]any, structuralPlanResult] {
	params := []ParameterContract{
		structuralStringParam("file", wireTargetGoFileDescription, true, false, RoleSelector,
			"semantic_set_doc_comment.result.edit_receipt.continuations[].file"),
		structuralStringParam("scope", "Selector: required target mode; this operation supports declaration scope only.", true, false, RoleSelector),
		structuralStringParam("symbol", "Target: live top-level Go declaration whose documentation comment will change.", true, false, RoleTarget,
			"semantic_set_doc_comment.result.edit_receipt.continuations[].symbol"),
		structuralStringParam("comment", "Replacement: plain documentation text without Go comment delimiters. An explicit empty string removes the doc comment.", true, true, RoleValue),
		structuralStringParam("expected_revision", "Precondition: opaque full-file revision copied from this tool's standalone live continuation; stale values are refused.", false, false, RoleSelector,
			"semantic_set_doc_comment.result.edit_receipt.continuations[].expected_revision"),
	}
	params[1].Schema.Enum = []any{"declaration"}
	output := structuralOutputContract(true)
	return structuralPlanDef(structuralPlanSpec{
		Key: "set_doc_comment", CLIName: "set-doc-comment", MCPName: "semantic_set_doc_comment",
		Summary:    "Use this tool instead of editing source text to set or remove one owned top-level Go declaration documentation comment. Scope is declaration only. Missing and null comment values are invalid; an explicit empty string removes the comment. Existing directives, grouped declarations, detached ownership, and ambiguous targets are refused. Standalone results include a live declaration continuation; batch receipts report applied_working without live continuations.",
		Params:     params,
		Output:     &output,
		ExampleRaw: map[string]any{"file": "doc.go", "scope": "declaration", "symbol": "Serve", "comment": "Serve handles requests."},
		Plan: func(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
			comment, present := raw["comment"]
			text, valid := comment.(string)
			if !present || !valid {
				return astedit.SourcePlan{}, fmt.Errorf("comment must be a present string; null is invalid")
			}
			return astedit.PlanSetDocComment(snapshot.Sources[snapshot.File], astedit.SetDocCommentIntent{Symbol: stringParam(raw, "symbol"), Comment: astedit.OptionalString{Present: true, Value: text}})
		},
		Result: func(cc CallContext, raw map[string]any, plan astedit.SourcePlan, status string) FileEditRes {
			return FileEditRes{Symbol: stringParam(raw, "symbol"), Detail: "Updated declaration documentation comment."}
		},
		Continuations: func(cc CallContext, snapshot *structuralSnapshot, raw map[string]any, plan astedit.SourcePlan) ([]SymbolTarget, error) {
			revision := structuralRevision(plan.Source)
			file, err := filepath.Rel(cc.WorkDir, snapshot.File)
			if err != nil {
				return nil, fmt.Errorf("derive declaration continuation path: %w", err)
			}
			if !filepath.IsLocal(file) {
				return nil, fmt.Errorf("declaration continuation path %q is outside the workspace", file)
			}
			return []SymbolTarget{{File: filepath.ToSlash(file), Symbol: stringParam(raw, "symbol"), ExpectedRevision: &revision}}, nil
		},
		ResolveContinuations: func(_ context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
			return candidate, true, nil
		},
	})
}

func rawOptionalString(raw map[string]any, name string) astedit.OptionalString {
	value, present := raw[name]
	result := astedit.OptionalString{Present: present, Null: present && value == nil}
	if text, ok := value.(string); ok {
		result.Value = text
	}
	return result
}

func insertStatementStructuralDef() Def[map[string]any, structuralPlanResult] {
	placement := structuralStringParam("placement", "Placement: insert immediately before or after the selected direct statement.", true, false, RolePolicy)
	placement.Schema.Enum = []any{"before_construct", "after_construct"}
	params := []ParameterContract{
		structuralStringParam("file", wireTargetGoFileDescription, true, false, RoleSelector),
		structuralStringParam(wireInFunction, "Selector: qualified function or receiver-qualified method owning the direct statement.", true, false, RoleSelector),
		structuralStringParam(wireDiscriminator, "Selector: exact source text of one direct call or simple assignment statement.", true, false, RoleSelector),
		structuralStringParam("source", "Value: exactly one direct call or simple assignment statement on one line.", true, false, RoleValue),
		placement,
		structuralStringParam("expected_revision", "Precondition: optional opaque full-file revision; stale values are refused.", false, false, RoleSelector),
	}
	output := structuralOutputContract(false)
	return structuralPlanDef(structuralPlanSpec{
		Key: "insert_statement", CLIName: "insert-statement", MCPName: "semantic_insert_statement",
		Summary:    "Use this tool instead of replacing a function body to insert one direct call or simple assignment immediately before or after a uniquely selected direct statement. The qualified owner preserves its receiver; the discriminator is exact anchor source. This initial version only accepts straight-line owners without control-flow constructs or labels; shared-line statements and ambiguous anchors are refused. The operation typechecks the affected package before one buffered apply; batch receipts report applied_working without live selectors.",
		Params:     params,
		Output:     &output,
		ExampleRaw: map[string]any{"file": "server.go", wireInFunction: "(*Server).Run", wireDiscriminator: "flush()", "source": "record()", "placement": "before_construct"},
		Plan: func(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
			return astedit.PlanInsertStatement(snapshot.Sources[snapshot.File], astedit.InsertStatementIntent{
				Function: stringParam(raw, wireInFunction), Discriminator: stringParam(raw, wireDiscriminator),
				ExpectedRevision: rawOptionalString(raw, "expected_revision"),
				Source:           stringParam(raw, "source"), Placement: astedit.StatementPlacement(stringParam(raw, "placement")),
			})
		},
		Result: func(cc CallContext, raw map[string]any, plan astedit.SourcePlan, status string) FileEditRes {
			return FileEditRes{InFunction: stringParam(raw, wireInFunction), Discriminator: stringParam(raw, wireDiscriminator), Detail: "Inserted statement adjacent to the selected statement."}
		},
	})
}

func structuralOutputContract(includeContinuations bool) OperationOutputContract {
	properties := map[string]ValueSchema{
		"result":          {Kind: SchemaString},
		"file":            {Kind: SchemaString},
		"symbol":          {Kind: SchemaString},
		"in_type":         {Kind: SchemaString},
		wireInFunction:    {Kind: SchemaString},
		wireDiscriminator: {Kind: SchemaString},
		"diff":            {Kind: SchemaString},
		"edit_receipt":    EditReceiptSchema(),
	}
	schema := ValueSchema{Kind: SchemaObject, Properties: properties, Required: []string{"result", "edit_receipt"}}
	contract := OperationOutputContract{ResultSchema: &schema}
	if includeContinuations {
		targetProperties := SymbolTargetSchema().Properties
		contract.Continuations = append(contract.Continuations,
			OutputContinuation{Path: "result.edit_receipt.continuations[].file", Schema: targetProperties["file"], Role: RoleSelector, Consumers: []string{"semantic_set_doc_comment"}},
			OutputContinuation{Path: "result.edit_receipt.continuations[].symbol", Schema: targetProperties["symbol"], Role: RoleTarget, Consumers: []string{"semantic_set_doc_comment"}},
			OutputContinuation{Path: "result.edit_receipt.continuations[].expected_revision", Schema: targetProperties["expected_revision"], Role: RoleSelector, Consumers: []string{"semantic_set_doc_comment"}},
		)
	}
	return contract
}

func registerStructuralPlanOps(registry *Registry) error {
	for _, definition := range []Def[map[string]any, structuralPlanResult]{setDocCommentStructuralDef(), insertStatementStructuralDef(), insertFieldStructuralDef(), extractTypeStructuralDef()} {
		if err := Register(registry, definition); err != nil {
			return fmt.Errorf("register structural plan %q: %w", definition.Key, err)
		}
	}
	if err := Register(registry, insertCompositeElementStructuralDef()); err != nil {
		return fmt.Errorf("register structural plan %q: %w", "insert_composite_element", err)
	}
	return nil
}
