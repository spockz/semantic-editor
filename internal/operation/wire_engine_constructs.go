// Package operation groups registry parsing and dispatch for generic construct insertion.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/javacst"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
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

var insertConstructParams = append([]ParameterContract{
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
}, readProjectParams...)

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
	} else if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	} else if req.Project.Language == "" {
		req.Project.Language = backend.LanguageGo
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
	if err := validateJavaInsertConstructRequest(req); err != nil {
		return req, err
	}
	if err := validateInsertConstructRequest(req); err != nil {
		return req, err
	}
	return req, nil
}

func validateJavaInsertConstructRequest(req InsertConstructReq) error {
	if req.Project.Language != backend.LanguageJava {
		return nil
	}
	if req.Kind == backend.ConstructElement {
		if strings.TrimSpace(req.InDeclaration) == "" {
			return fmt.Errorf("java element insertion requires in_declaration: %w", ErrInvalidParams)
		}
		if nonEmpty(req.Function, req.Anchor, req.TypeName, req.TargetSymbol, req.AccessModifier, req.Group) {
			return fmt.Errorf("java element insertion does not accept method-local or declaration placement selectors: %w", ErrInvalidParams)
		}
		return nil
	}
	switch req.Kind {
	case backend.ConstructLoop, backend.ConstructIf, javacst.ConstructSwitch, backend.ConstructTryWithResources,
		backend.ConstructSynchronized, backend.ConstructStatementOrExpr:
	default:
		return fmt.Errorf("java insert_construct does not support kind=%s: %w", req.Kind, ErrInvalidParams)
	}
	if strings.TrimSpace(req.Function) == "" || strings.TrimSpace(req.Anchor) == "" {
		return fmt.Errorf("java method-local insertion requires in_function and anchor: %w", ErrInvalidParams)
	}
	if nonEmpty(req.TypeName, req.InDeclaration, req.TargetSymbol, req.AccessModifier, req.Group, req.Placement) {
		return fmt.Errorf("java method-local insertion does not accept declaration or placement selectors: %w", ErrInvalidParams)
	}
	return nil
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
		if req.Project.Language == backend.LanguageJava {
			return runInsertElementConstructJava(ctx, cc, req)
		}
		return runInsertElementConstruct(ctx, cc, req)
	}
	if req.Kind == backend.ConstructField {
		return runInsertFieldConstruct(ctx, cc, req)
	}
	if req.Project.Language == backend.LanguageJava {
		return runInsertConstructJava(ctx, cc, req)
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
	planned.Draft.Operation = operationKeyInsertConstruct
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
		Key:            operationKeyInsertConstruct,
		Summary:        ("Use this tool instead of text replacement to add one backend-supported declaration, field, keyed composite element, switch case, or anchored statement. For kind=element, select a typed map/dictionary literal and append one keyed `key: value` pair, or select a struct/object initializer and append one property; slices/arrays and positional elements are unsupported. Select the exact literal with `discriminator` and its owning top-level declaration or function/method with `in_declaration`. Element insertion preserves the exact planned bytes and ignores auto_organize_imports. Field insertion requires an existing locally declared, private, non-generic Go struct. Use semantic_replace_construct for existing targets; insertions fail closed on collisions." + automaticVerificationGuidance),
		Params:         insertConstructParams,
		Level:          LevelFile,
		CLIName:        "insert-construct",
		MCPName:        semanticInsertConstructToolName,
		Parse:          parseInsertConstruct,
		NormalizeInput: NormalizeConstructKindInput,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertConstructReq) (FileEditRes, error){
			backend.LanguageGo:   bufferedFileEditHandler(operationKeyInsertConstruct, func(req InsertConstructReq) string { return req.File }, runInsertConstruct),
			backend.LanguageJava: runInsertConstructJava,
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

func runInsertConstructJava(ctx context.Context, cc CallContext, req InsertConstructReq) (result FileEditRes, retErr error) {
	if req.Kind == backend.ConstructElement {
		return runInsertElementConstructJava(ctx, cc, req)
	}
	if err := javacst.ValidateMethodConstructSnippet(req.Kind, []byte(req.Source)); err != nil {
		return result, fmt.Errorf("validate Java insertion source: %w", err)
	}
	if strings.TrimSpace(req.Function) == "" {
		return result, fmt.Errorf("in_function is required for Java method-local insertion")
	}
	root, err := filepath.Abs(cc.WorkDir)
	if err != nil {
		return result, fmt.Errorf("resolve Java insertion workspace: %w", err)
	}
	file := resolveWorkPath(root, req.File)
	project := effectiveProject(cc, req.Project)
	project.RootDir, project.File = root, file
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	display := displayWorkPath(root, req.File)
	before, after, err := applyJavaConstructInsertion(ctx, root, file, project, display, service, cc, req)
	if err != nil {
		return result, err
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return result, fmt.Errorf("derive Java insertion receipt path: %w", err)
	}
	if !filepath.IsLocal(relative) {
		return result, fmt.Errorf("java insertion target %q escapes workspace %q", file, root)
	}
	relative = filepath.ToSlash(relative)
	draft := EditReceiptDraft{
		Operation: operationKeyInsertConstruct, Status: EditPersisted,
		ReadSet:        []FileRevision{{Path: relative, Revision: structuralRevision(before)}},
		PlannedChanges: []FileRevision{{Path: relative, Revision: structuralRevision(after)}},
	}
	return FileEditRes{
		File: file, Display: display, InFunction: req.Function, Discriminator: req.Discriminator,
		Diff: astedit.UnifiedDiff(display, string(before), string(after)), Detail: "Successfully inserted Java method construct.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true, ReceiptDraft: draft,
	}, nil
}

func applyJavaConstructInsertion(ctx context.Context, root, file string, project backend.ProjectContext, display string, service *backend.Service, cc CallContext, req InsertConstructReq) ([]byte, []byte, error) {
	var before, after []byte
	err := sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		var readErr error
		before, readErr = sourcebuf.ReadFile(editCtx, file)
		if readErr != nil {
			return fmt.Errorf("read Java insertion target: %w", readErr)
		}
		if req.ExpectedRevision != "" && structuralRevision(before) != req.ExpectedRevision {
			return fmt.Errorf("java insertion target revision changed")
		}
		inspection, inspectErr := service.Inspect(editCtx, backend.InspectRequest{Project: project, Symbol: req.Function})
		if inspectErr != nil {
			return inspectErr
		}
		if inspection == nil || len(inspection.Matches) != 1 {
			return fmt.Errorf("java insertion owner %q must resolve to one method in %s", req.Function, display)
		}
		method := inspection.Matches[0]
		methodSource := []byte(method.Source)
		start, offsetErr := javacst.ByteOffsetFromPosition(before, method.Range.Start)
		if offsetErr != nil {
			return fmt.Errorf("resolve Java method source offset: %w", offsetErr)
		}
		end := start + len(methodSource)
		if end > len(before) || !bytes.Equal(before[start:end], methodSource) {
			return fmt.Errorf("java method source changed during insertion selection")
		}
		updatedMethod, insertErr := javacst.InsertIntoMethod(methodSource, req.Anchor, req.Discriminator, req.ConstructPath, []byte(req.Source))
		if insertErr != nil {
			return insertErr
		}
		updated := make([]byte, 0, len(before)-len(methodSource)+len(updatedMethod))
		updated = append(updated, before[:start]...)
		updated = append(updated, updatedMethod...)
		updated = append(updated, before[end:]...)
		if writeErr := pipeline.WriteAtomicContext(editCtx, file, updated); writeErr != nil {
			return fmt.Errorf("stage Java construct insertion: %w", writeErr)
		}
		if _, verifyErr := service.Verify(editCtx, backend.VerifyRequest{Project: project, Path: file, FormatSelectedFile: true, OrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports)}); verifyErr != nil {
			return fmt.Errorf("verify Java construct insertion: %w", verifyErr)
		}
		after, readErr = sourcebuf.ReadFile(editCtx, file)
		if readErr != nil {
			return fmt.Errorf("read verified Java construct result: %w", readErr)
		}
		return nil
	})
	return before, after, err
}
