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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target: source file to modify; path is relative to the workspace root", Required: true, SourceFields: sourceFileContinuations},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Construct kind supported by the selected language backend", Required: true, DynamicEnums: executableInsertConstructKinds},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Source code for the new construct", Required: true},
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier for declarations, fields, or cases", Enums: constructPlacementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol or anchor case identifier required for relative placement"},
	{Name: "in_type", CLIName: "in-type", JSONName: "in_type", Type: ParamString, Description: "Selector: existing Go struct type that owns a field insertion", SourceFields: []string{"semantic_inspect_symbol.result.continuations[].in_type"}},
	{Name: "group", CLIName: "group", JSONName: "group", Type: ParamString, Description: "Group merging behavior for declarations: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')", Enums: groupEnum},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified enclosing function or method name for nested constructs such as a case.", SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Discriminant expression or switch condition (e.g. for case)", SourceFields: sourceDiscriminatorContinuations},
	{Name: "anchor", CLIName: "anchor", JSONName: "anchor", Type: ParamString, Description: "Statement insertion anchor: start, end, before_return, before, or after; before/after require discriminator", Enums: []string{"start", "end", "before_return", "before", "after"}},
	{Name: "construct_path", CLIName: wireCLIConstructPath, JSONName: "construct_path", Type: ParamString, Description: "Candidate path from an ambiguity diagnostic, such as 0 or 0.1; repeated statement anchors also require its expected_revision"},
	{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Description: "Required with construct_path: file revision copied from the ambiguity diagnostic"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

var constructPlacementEnum = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol", "type_end", "first", "last", "before_default", "before", "after"}

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
	return req, nil
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
		Key:     "insert_construct",
		Summary: "Use this tool instead of text replacement to add one backend-supported declaration, field, switch case, or anchored statement. Use semantic_replace_construct for existing targets; specialized insert tools are optional variants. Insertions fail closed on collisions." + automaticVerificationGuidance,
		Params:  insertConstructParams,
		Level:   LevelFile,
		CLIName: "insert-construct",
		MCPName: "semantic_insert_construct",
		Parse:   parseInsertConstruct,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertConstructReq) (FileEditRes, error){
			backend.LanguageGo: runInsertConstruct,
		},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "kind": "function", "source": "func Stop() {}"},
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}
