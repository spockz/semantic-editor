// Package operation groups registry parsing and dispatch for generic construct insertion.
package operation

import (
	"context"
	"fmt"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"strings"
)

// InsertConstructReq inserts one construct into an existing source file.
type InsertConstructReq struct {
	Project             backend.ProjectContext
	File                string
	Kind                backend.ConstructKind
	Source              string
	AccessModifier      string
	Placement           string
	TargetSymbol        string
	Group               string
	Function            string
	TypeName            string
	Discriminator       string
	InDeclaration       string
	Anchor              string
	ConstructPath       string
	ExpectedRevision    string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertConstructReq) GetProjectContext() backend.ProjectContext {
	return r.Project
}

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertConstructReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var insertConstructParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: source file to modify; path is relative to the workspace root", Required: true, SourceFields: append(append([]string(nil), sourceFileContinuations...), "semantic_insert_construct.result.selector_continuations[].file")},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Construct kind supported by the selected language backend; use element for one keyed map/dictionary entry or struct/object initializer property", Required: true, DynamicEnums: executableInsertConstructKinds},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Source code for the new construct", Required: true},
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier for declarations, fields, cases, or element append", Enums: append(append([]string(nil), constructPlacementEnum...), "append")},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol or anchor case identifier required for relative placement"},
	{Name: "in_type", CLIName: "in-type", JSONName: "in_type", Type: ParamString, Role: RoleSelector, Description: "Selector: existing locally declared, private, non-generic Go struct type that owns a field insertion", SourceFields: []string{"semantic_inspect_symbol.result.continuations[].in_type"}},
	{Name: "in_declaration", CLIName: "in-declaration", JSONName: "in_declaration", Type: ParamString, Role: RoleSelector, Description: "Selector for kind=element: exact owning top-level declaration, function name, or qualified receiver method such as (*Server).Serve", SourceFields: []string{"semantic_inspect_symbol.result.continuations[].in_declaration", "semantic_insert_construct.result.selector_continuations[].in_declaration"}},
	{Name: "group", CLIName: "group", JSONName: "group", Type: ParamString, Description: "Group merging behavior for declarations: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')", Enums: groupEnum},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified enclosing function or method name for nested constructs such as a case.", SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Role: RoleSelector, Description: "Discriminant expression, switch condition, or exact composite literal source selector", SourceFields: append(append([]string(nil), sourceDiscriminatorContinuations...), "semantic_insert_construct.result.selector_continuations[].discriminator")},
	{Name: "anchor", CLIName: "anchor", JSONName: "anchor", Type: ParamString, Description: "Statement insertion anchor: start, end, before_return, before, or after; before/after require discriminator", Enums: []string{"start", "end", "before_return", "before", "after"}},
	{Name: "construct_path", CLIName: wireCLIConstructPath, JSONName: "construct_path", Type: ParamString, Description: "Candidate path from an ambiguity diagnostic for statement/expression insertion; requires the matching expected_revision"},
	{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Role: RoleTarget, Description: "Opaque full-file revision required with construct_path for statement/expression insertion and optionally used as a full-file guard for statement, field, or element insertion", SourceFields: []string{"semantic_inspect_symbol.result.continuations[].expected_revision", "semantic_insert_construct.result.selector_continuations[].expected_revision"}},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

var constructPlacementEnum = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol", "type_start", "type_end", "first", "last", "before_default", "before", "after"}

func executableInsertConstructKinds(candidate backend.Backend) []string {
	return executableKinds(candidate, backend.OperationInsertConstruct)
}

func executableReplaceConstructKinds(candidate backend.Backend) []string {
	return executableKinds(candidate, backend.OperationReplaceConstruct)
}

func executableRemoveConstructKinds(candidate backend.Backend) []string {
	return executableKinds(candidate, backend.OperationRemoveConstruct)
}

func executableKinds(candidate backend.Backend, operation backend.Operation) []string {
	provider, ok := candidate.(backend.ConstructCapabilityProvider)
	if !ok {
		return nil
	}
	kinds := provider.SupportedConstructs(operation)
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}
	return names
}

func parseInsertConstruct(raw map[string]any) (InsertConstructReq, error) {
	var req InsertConstructReq
	if err := CheckParams(raw, insertConstructParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	kindStr, err := ParseString(raw, "kind", "kind", true)
	if err != nil {
		return req, err
	}
	req.Kind = backend.ConstructKind(kindStr)
	if req.Source, err = ParseString(raw, "source", "source", true); err != nil {
		return req, err
	}
	if req.AccessModifier, err = ParseString(raw, wireAccessModifier, "access", false); err != nil {
		return req, err
	}
	if req.Placement, err = ParseString(raw, "placement", "placement", false); err != nil {
		return req, err
	}
	if req.TargetSymbol, err = ParseString(raw, wireTargetSymbol, "target", false); err != nil {
		return req, err
	}
	if req.TypeName, err = ParseString(raw, "in_type", "in-type", false); err != nil {
		return req, err
	}
	if req.InDeclaration, err = ParseString(raw, "in_declaration", "in-declaration", false); err != nil {
		return req, err
	}
	if req.Group, err = ParseString(raw, "group", "group", false); err != nil {
		return req, err
	}
	if req.Function, err = ParseString(raw, wireInFunction, wireCLIInFunction, false); err != nil {
		return req, err
	}
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.Anchor, err = ParseString(raw, "anchor", "anchor", false); err != nil {
		return req, err
	}
	if req.ConstructPath, err = parseNonEmptyOptionalString(raw, "construct_path", wireCLIConstructPath); err != nil {
		return req, err
	}
	if req.ExpectedRevision, err = parseNonEmptyOptionalString(raw, wireExpectedRevision, wireCLIExpectedRevision); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	if req.Kind == backend.ConstructElement && strings.TrimSpace(req.Placement) == "" {
		req.Placement = "append"
	}
	if err := validateInsertConstructRequest(req); err != nil {
		return req, err
	}
	return req, nil
}

func validateInsertConstructRequest(req InsertConstructReq) error {
	if req.Kind == backend.ConstructElement {
		if strings.TrimSpace(req.InDeclaration) == "" {
			return fmt.Errorf("in_declaration is required for element insertion: %w", ErrInvalidParams)
		}
		if strings.TrimSpace(req.Discriminator) == "" {
			return fmt.Errorf("discriminator is required for element insertion: %w", ErrInvalidParams)
		}
		if strings.TrimSpace(req.Source) == "" {
			return fmt.Errorf("source is required for element insertion: %w", ErrInvalidParams)
		}
		if placement := strings.TrimSpace(req.Placement); placement != "" && placement != "append" {
			return fmt.Errorf("element placement must be append: %w", ErrInvalidParams)
		}
		if nonEmpty(req.TypeName, req.Function, req.TargetSymbol, req.Anchor, req.AccessModifier, req.Group, req.ConstructPath) {
			return fmt.Errorf("element insertion does not accept in_type, in_function, target_symbol, anchor, access_modifier, group, or construct_path: %w", ErrInvalidParams)
		}
		return nil
	}
	if strings.TrimSpace(req.InDeclaration) != "" {
		return fmt.Errorf("in_declaration is only valid for element insertion: %w", ErrInvalidParams)
	}
	if strings.TrimSpace(req.Placement) == "append" {
		return fmt.Errorf("append placement is only valid for element insertion: %w", ErrInvalidParams)
	}
	if req.ConstructPath != "" && req.Kind != backend.ConstructStatementOrExpr {
		return fmt.Errorf("construct_path is only valid for statement/expression insertion: %w", ErrInvalidParams)
	}
	if req.ConstructPath != "" && req.ExpectedRevision == "" {
		return fmt.Errorf("construct_path requires expected_revision: %w", ErrInvalidParams)
	}
	if req.ExpectedRevision != "" && req.Kind != backend.ConstructElement && req.Kind != backend.ConstructField && req.Kind != backend.ConstructStatementOrExpr {
		return fmt.Errorf("expected_revision is only valid for field, element, and statement/expression insertion: %w", ErrInvalidParams)
	}
	return nil
}

func nonEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func parseNonEmptyOptionalString(raw map[string]any, jsonName, cliName string) (string, error) {
	value, present := rawLookup(raw, jsonName, cliName)
	if !present {
		return "", nil
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("param %q must be a non-empty string when provided: %w", jsonName, ErrInvalidParams)
	}
	return text, nil
}

func runInsertConstruct(ctx context.Context, cc CallContext, req InsertConstructReq) (FileEditRes, error) {
	if req.Kind == backend.ConstructElement {
		return runInsertElementConstruct(ctx, cc, req)
	}
	if req.Kind == backend.ConstructField {
		return runInsertFieldConstruct(ctx, cc, req)
	}
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	res, err := astedit.InsertConstruct(ctx, targetPath, req.Source, astedit.InsertConstructOptions{
		Kind:                req.Kind,
		AccessModifier:      astedit.AccessModifier(req.AccessModifier),
		Placement:           astedit.Placement(req.Placement),
		TargetSymbol:        req.TargetSymbol,
		TypeName:            req.TypeName,
		Group:               req.Group,
		Function:            req.Function,
		Discriminator:       req.Discriminator,
		Anchor:              req.Anchor,
		ConstructPath:       req.ConstructPath,
		ExpectedRevision:    req.ExpectedRevision,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	var symbol string
	var line int
	switch req.Kind {
	case backend.ConstructFunction, backend.ConstructMethod, backend.ConstructType,
		backend.ConstructConst, backend.ConstructVar, backend.ConstructDecl, backend.ConstructDeclaration:
		symbol, line, err = goTargetAfterEdit(ctx, targetPath, req.Source)
		if err != nil {
			return FileEditRes{}, fmt.Errorf("edit applied to %s, but could not inspect result: %w", displayWorkPath(cc.WorkDir, req.File), err)
		}
	default:
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File:             targetPath,
		Display:          displayWorkPath(cc.WorkDir, req.File),
		Symbol:           symbol,
		InType:           req.TypeName,
		Line:             line,
		InFunction:       constructContinuationFunction(req),
		Discriminator:    constructContinuationDiscriminator(req),
		Diff:             res.Diff,
		Detail:           fmt.Sprintf("Successfully inserted %s construct.", req.Kind),
		Delta:            delta,
		DiagnosticStatus: diagnosticStatus,
		HasDelta:         true,
	}, nil
}

func runInsertFieldConstruct(ctx context.Context, cc CallContext, req InsertConstructReq) (FileEditRes, error) {
	fieldSymbol, err := constructFieldSymbol(req.TypeName, req.Source)
	if err != nil {
		return FileEditRes{}, err
	}
	fieldName := strings.TrimPrefix(fieldSymbol, req.TypeName+".")
	if validateErr := astedit.ValidateAccess(astedit.GolangBackend{}, astedit.AccessModifier(req.AccessModifier), fieldName); validateErr != nil {
		return FileEditRes{}, fmt.Errorf("validate field access modifier: %w: %w", validateErr, ErrInvalidParams)
	}
	raw := map[string]any{
		"file": req.File, "in_type": req.TypeName, "symbol": fieldSymbol,
		"source": req.Source, "placement": req.Placement,
	}
	if req.TargetSymbol != "" {
		raw["target_symbol"] = req.TargetSymbol
	}
	if req.ExpectedRevision != "" {
		raw[wireExpectedRevision] = req.ExpectedRevision
	}
	planned, err := insertFieldStructuralDef().Handlers[backend.LanguageGo](ctx, cc, raw)
	if err != nil {
		return FileEditRes{}, err
	}
	planned.Draft.Operation = "insert_construct"
	planned.Edit.Detail = "Successfully inserted field construct."
	return FileEditRes{
		File: planned.Edit.File, Display: planned.Edit.Display, Symbol: planned.Edit.Symbol,
		InType: planned.Edit.InType, Diff: planned.Edit.Diff, Detail: planned.Edit.Detail,
		Delta: planned.Edit.Delta, DiagnosticStatus: planned.Edit.DiagnosticStatus, HasDelta: planned.Edit.HasDelta,
		ReceiptDraft: planned.Draft, Resolve: planned.Resolve,
	}, nil
}

func constructFieldSymbol(typeName, source string) (string, error) {
	if typeName == "" {
		return "", fmt.Errorf("in_type is required for field insertion: %w", ErrInvalidParams)
	}
	fieldName, err := astedit.InferInsertedFieldName(source)
	if err != nil {
		return "", fmt.Errorf("parse inserted field source: %w: %w", err, ErrInvalidParams)
	}
	return typeName + "." + fieldName, nil
}

func constructContinuationFunction(req InsertConstructReq) string {
	if req.Kind == backend.ConstructCase || req.Kind == backend.ConstructStatementOrExpr {
		return req.Function
	}
	return ""
}

func constructContinuationDiscriminator(req InsertConstructReq) string {
	if req.Kind == backend.ConstructCase || req.Kind == backend.ConstructStatementOrExpr {
		return req.Discriminator
	}
	return ""
}

func insertConstructDef() Def[InsertConstructReq, FileEditRes] {
	return Def[InsertConstructReq, FileEditRes]{
		Key:            "insert_construct",
		Summary:        ("Use this tool instead of text replacement to add one backend-supported declaration, field, keyed composite element, switch case, or anchored statement. For kind=element, select a typed map/dictionary literal and append one keyed `key: value` pair, or select a struct/object initializer and append one property; slices/arrays and positional elements are unsupported. Select the exact literal with `discriminator` and its owning top-level declaration or function/method with `in_declaration`. Element insertion preserves the exact planned bytes and ignores auto_organize_imports. Field insertion requires an existing locally declared, private, non-generic Go struct. Use semantic_replace_construct for existing targets; insertions fail closed on collisions." + automaticVerificationGuidance),
		Params:         insertConstructParams,
		Level:          LevelFile,
		CLIName:        "insert-construct",
		MCPName:        semanticInsertConstructToolName,
		Parse:          parseInsertConstruct,
		NormalizeInput: NormalizeConstructKindInput,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertConstructReq) (FileEditRes, error){
			backend.LanguageGo: bufferedFileEditHandler("insert_construct", func(req InsertConstructReq) string { return req.File }, runInsertConstruct),
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "kind": "function", "source": "func Stop() {}"},
		Output:     insertConstructOutputContract(),
		FinalizeOutput: func(cc CallContext, _ InsertConstructReq, result FileEditRes) (FileEditRes, error) {
			return finalizeInsertConstructOutput(cc, result)
		},
		ProjectOutput: projectInsertConstructOutput,
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}
