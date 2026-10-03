// This file registers Go function inlining as a shared CLI, MCP, and buffered operation.

package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/pipeline"
)

const (
	semanticInlineFunctionToolName = "semantic_inline_function"
	semanticMigrateTypeToolName    = "semantic_migrate_type"
	semanticReplaceBodyToolName    = "semantic_replace_body"
	semanticInspectSymbolToolName  = "semantic_inspect_symbol"
)

// InlineFunctionCallSite selects one caller-side expression for function inlining.
type InlineFunctionCallSite struct {
	File             string
	InFunction       string
	Discriminator    string
	ExpectedRevision string
}

// InlineFunctionReq requests one bounded direct Go call expansion.
type InlineFunctionReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	ExpectedRevision string
	CallSite         InlineFunctionCallSite
}

// GetProjectContext returns the project selector used for registry dispatch.
func (r InlineFunctionReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the project selector after ambient context merging.
func (r *InlineFunctionReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// SemanticEditRes keeps the human-readable diff and typed receipt draft through finalization.
type SemanticEditRes struct {
	Diff             string
	Delta            pipeline.DiagnosticDelta
	DiagnosticStatus string
	Draft            EditReceiptDraft
	Receipt          EditReceipt
}

var inlineFunctionCallSiteSchema = ValueSchema{
	Kind: SchemaObject,
	Properties: map[string]ValueSchema{
		"file":              {Kind: SchemaString},
		"in_function":       {Kind: SchemaString},
		wireDiscriminator:   {Kind: SchemaString},
		"expected_revision": {Kind: SchemaString},
	},
	Required: []string{"file", "in_function", wireDiscriminator},
}

var inlineFunctionParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target: existing Go file that declares the free function being inlined", Required: true, Role: RoleTarget, SourceFields: []string{sourceLookupFile, sourceRenameFile, semanticInspectSymbolToolName + ".result.continuations[].file", semanticInlineFunctionToolName + ".result.continuations[].file"}},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: unqualified free-function name to inline; the declaration is preserved", Required: true, Role: RoleTarget, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, semanticInspectSymbolToolName + ".result.continuations[].symbol", semanticInlineFunctionToolName + ".result.continuations[].symbol"}},
	{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Description: "Guard: full-file revision copied from semantic_inspect_symbol, a resolved rename continuation, or an inline-function continuation for the callee file", Role: RoleTarget, SourceFields: []string{semanticInspectSymbolToolName + ".result.continuations[].expected_revision", semanticInlineFunctionToolName + ".result.continuations[].expected_revision", sourceRenameExpectedRevision}},
	{Name: "call_site", CLIName: "call-site", JSONName: "call_site", Type: ParamObject, Description: "Selector: closed caller-selected current call site with file, in_function, and exact discriminator. Copy file, in_function, and optional expected_revision from same-named semantic_inspect_symbol continuation fields. The discriminator is the exact direct call expression visible in the inspected owner body.", Required: true, Role: RoleSelector, Schema: &inlineFunctionCallSiteSchema},
}

func parseInlineFunction(raw map[string]any) (InlineFunctionReq, error) {
	var req InlineFunctionReq
	if err := CheckParams(raw, inlineFunctionParams); err != nil {
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
	if value, exists := raw[wireExpectedRevision]; exists {
		text, ok := value.(string)
		if !ok || text == "" {
			return req, fmt.Errorf("expected_revision must be a non-empty string")
		}
		req.ExpectedRevision = text
	}
	callSite, ok := raw["call_site"].(map[string]any)
	if !ok {
		return req, fmt.Errorf("call_site must be an object")
	}
	if req.CallSite.File, err = ParseString(callSite, "file", "file", true); err != nil {
		return req, fmt.Errorf("call_site: %w", err)
	}
	if req.CallSite.InFunction, err = ParseString(callSite, "in_function", "in_function", true); err != nil {
		return req, fmt.Errorf("call_site: %w", err)
	}
	if req.CallSite.Discriminator, err = ParseString(callSite, wireDiscriminator, wireDiscriminator, true); err != nil {
		return req, fmt.Errorf("call_site: %w", err)
	}
	if value, exists := callSite[wireExpectedRevision]; exists {
		revision, ok := value.(string)
		if !ok || revision == "" {
			return req, fmt.Errorf("call_site.expected_revision must be a non-empty string")
		}
		req.CallSite.ExpectedRevision = revision
	}
	if err := validateInlineCallSite(callSite); err != nil {
		return req, err
	}
	return req, nil
}

func validateInlineCallSite(raw map[string]any) error {
	for name := range raw {
		switch name {
		case "file", "in_function", wireDiscriminator, wireExpectedRevision:
		default:
			return fmt.Errorf("call_site has unknown field %q", name)
		}
	}
	if !token.IsIdentifier(raw["in_function"].(string)) {
		return fmt.Errorf("call_site.in_function must be an unqualified free-function name")
	}
	if strings.TrimSpace(raw[wireDiscriminator].(string)) != raw[wireDiscriminator].(string) || raw[wireDiscriminator].(string) == "" {
		return fmt.Errorf("call_site.discriminator must be an exact non-empty source expression")
	}
	return nil
}

func runInlineFunction(ctx context.Context, cc CallContext, req InlineFunctionReq) (SemanticEditRes, error) {
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	plan, err := astedit.PlanInlineFunctionOverlay(ctx, cc.WorkDir, req.File, req.Symbol, astedit.InlineFunctionOptions{
		CallerFile: req.CallSite.File, InFunction: req.CallSite.InFunction, Discriminator: req.CallSite.Discriminator,
		CallerRevision: req.CallSite.ExpectedRevision, CalleeRevision: req.ExpectedRevision,
	})
	if err != nil {
		return SemanticEditRes{}, err
	}
	calleePath, err := filepath.Abs(filepath.Join(cc.WorkDir, req.File))
	if err != nil {
		return SemanticEditRes{}, fmt.Errorf("resolve inline callee path: %w", err)
	}
	calleeRevision := revisionHex(plan.ReadSet[calleePath])
	if filepath.Clean(calleePath) == filepath.Clean(plan.Change.Path) {
		calleeRevision = revisionHex(plan.Change.After)
	}
	callerRevision := revisionHex(plan.Change.After)
	draft, err := semanticEditDraft(cc.WorkDir, "inline-function", plan, []SymbolTarget{
		{File: absoluteToWorkPath(cc.WorkDir, calleePath), Symbol: req.Symbol, ExpectedRevision: &calleeRevision},
		{File: absoluteToWorkPath(cc.WorkDir, plan.Change.Path), Symbol: req.CallSite.InFunction, ExpectedRevision: &callerRevision},
	})
	if err != nil {
		return SemanticEditRes{}, err
	}
	if err := astedit.ApplySemanticEdit(ctx, plan); err != nil {
		return SemanticEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return SemanticEditRes{Diff: plan.Diff, Delta: delta, DiagnosticStatus: diagnosticStatus, Draft: draft}, nil
}

func inlineFunctionDef() Def[InlineFunctionReq, SemanticEditRes] {
	receiptSchema := EditReceiptSchema()
	return Def[InlineFunctionReq, SemanticEditRes]{
		Key:     "inline_function",
		Summary: "Use this tool instead of text editing to inline one direct zero-argument call to a same-package Go free function whose body is a single literal return. Select the callee with file and symbol, then select the current caller expression with the closed call_site object. The callee declaration remains in place. Arguments, methods, generic functions, nonliteral bodies, and unsupported scopes are refused before the caller changes.",
		Params:  inlineFunctionParams, Level: LevelFile, CLIName: "inline-function", MCPName: semanticInlineFunctionToolName,
		Parse:    parseInlineFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InlineFunctionReq) (SemanticEditRes, error){backend.LanguageGo: runInlineFunction},
		Format: func(res SemanticEditRes) (string, error) {
			text := "Successfully inlined one call. diagnostic_status: " + res.DiagnosticStatus + "."
			if res.DiagnosticStatus == "changed" {
				text = AppendDiagnosticDelta(text, res.Delta)
			}
			return text + "\n" + res.Diff, nil
		},
		ExampleRaw: map[string]any{"file": "helper.go", "symbol": "adjust", "call_site": map[string]any{"file": "caller.go", "in_function": "Run", wireDiscriminator: "adjust()"}},
		Output: &OperationOutputContract{ResultSchema: &receiptSchema, Continuations: []OutputContinuation{
			{Path: "result.continuations[].file", Schema: SymbolTargetSchema().Properties["file"], Role: RoleTarget, Consumers: []string{semanticInlineFunctionToolName, semanticReplaceBodyToolName}},
			{Path: "result.continuations[].symbol", Schema: SymbolTargetSchema().Properties["symbol"], Role: RoleTarget, Consumers: []string{semanticInlineFunctionToolName, semanticReplaceBodyToolName}},
			{Path: "result.continuations[].expected_revision", Schema: SymbolTargetSchema().Properties[wireExpectedRevision], Role: RoleTarget, Consumers: []string{semanticInlineFunctionToolName, semanticReplaceBodyToolName}},
		}},
		FinalizeOutput: finalizeInlineFunctionOutput,
		ProjectOutput:  func(res SemanticEditRes) (any, error) { return res.Receipt, nil },
		BufferedBatch:  true, Batchable: true,
	}
}

func semanticEditDraft(root, operation string, plan astedit.SemanticEditPlan, targets []SymbolTarget) (EditReceiptDraft, error) {
	readSet := make([]FileRevision, 0, len(plan.ReadSet))
	for path, source := range plan.ReadSet {
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return EditReceiptDraft{}, fmt.Errorf("semantic read input is outside the selected workspace: %s", path)
		}
		readSet = append(readSet, FileRevision{Path: filepath.ToSlash(relative), Revision: revisionHex(source)})
	}
	sort.Slice(readSet, func(i, j int) bool { return readSet[i].Path < readSet[j].Path })
	outputPath, err := filepath.Rel(root, plan.Change.Path)
	if err != nil || outputPath == ".." || strings.HasPrefix(outputPath, ".."+string(filepath.Separator)) {
		return EditReceiptDraft{}, fmt.Errorf("semantic output is outside the selected workspace: %s", plan.Change.Path)
	}
	planned := []FileRevision{{Path: filepath.ToSlash(outputPath), Revision: revisionHex(plan.Change.After)}}
	return EditReceiptDraft{Operation: operation, Status: EditPersisted, PlannedChanges: planned, ReadSet: readSet, Continuations: targets}, nil
}

func finalizeInlineFunctionOutput(cc CallContext, _ InlineFunctionReq, res SemanticEditRes) (SemanticEditRes, error) {
	return finalizeSemanticEditResult(cc, res)
}

func finalizeSemanticEditResult(cc CallContext, res SemanticEditRes) (SemanticEditRes, error) {
	receipt, err := FinalizeEditReceipt(cc.Ctx, cc.WorkDir, cc.InBatch, res.Draft, func(ctx context.Context, target SymbolTarget) (SymbolTarget, bool, error) {
		return resolveSemanticTarget(ctx, cc, target)
	})
	if err != nil {
		return SemanticEditRes{}, err
	}
	res.Receipt = receipt
	return res, nil
}

func revisionHex(source []byte) string {
	digest := sha256.Sum256(source)
	return hex.EncodeToString(digest[:])
}

func absoluteToWorkPath(root, path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	relative, err := filepath.Rel(root, abs)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

func resolveSemanticTarget(ctx context.Context, cc CallContext, target SymbolTarget) (SymbolTarget, bool, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, cc.Project)
	project.File = filepath.Join(cc.WorkDir, filepath.FromSlash(target.File))
	project.Language = backend.LanguageGo
	result, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: target.Symbol})
	if err != nil {
		return target, false, err
	}
	if target.ExpectedRevision == nil {
		return target, false, nil
	}
	for _, match := range result.Matches {
		if match.QualifiedName != target.Symbol || match.Revision != *target.ExpectedRevision {
			continue
		}
		matchedPath := match.File
		if !filepath.IsAbs(matchedPath) {
			matchedPath = filepath.Join(cc.WorkDir, matchedPath)
		}
		if filepath.Clean(matchedPath) == filepath.Clean(project.File) {
			return target, true, nil
		}
	}
	return target, false, nil
}
