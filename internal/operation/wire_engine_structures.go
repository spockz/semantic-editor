// Package operation groups Go structural construct insertion handlers.
package operation

import (
	"context"
	"fmt"
	"semedit/internal/astedit"
	"semedit/internal/backend"
)

// InsertStructureReq inserts a structural construct into an existing source file.
type InsertStructureReq struct {
	Project             backend.ProjectContext
	File                string
	Kind                backend.StructureKind
	Source              string
	AccessModifier      string
	Placement           string
	TargetSymbol        string
	Group               string
	Overwrite           bool
	Function            string
	Discriminator       string
	ConstructPath       string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertStructureReq) GetProjectContext() backend.ProjectContext {
	return r.Project
}

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertStructureReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var insertStructureParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Source file target; relative paths resolve from the active semedit workspace root", Required: true},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Structural construct kind supported by the selected language backend", Required: true, DynamicEnums: executableStructureKinds},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Structural construct source code snippet", Required: true},
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol, first, last, before, after, before_default", Enums: structurePlacementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol or anchor case identifier required for relative placement"},
	{Name: "group", CLIName: "group", JSONName: "group", Type: ParamString, Description: "Group merging behavior for declarations: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')", Enums: groupEnum},
	{Name: "overwrite", CLIName: "overwrite", JSONName: "overwrite", Type: ParamBoolean, Description: "Replace an existing declaration with the same name (default false)"},
	{Name: "function", CLIName: "function", JSONName: "function", Type: ParamString, Description: "Enclosing function or method name (e.g. for nested constructs like case)"},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Discriminant expression or switch condition (e.g. for case)"},
	{Name: "construct_path", CLIName: "construct-path", JSONName: "construct_path", Type: ParamString, Description: "Candidate path from an ambiguity diagnostic, such as 0 or 0.1"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

func executableStructureKinds(candidate backend.Backend) []string {
	provider, ok := candidate.(backend.StructureCapabilityProvider)
	if !ok {
		return nil
	}
	kinds := provider.SupportedStructures()
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}
	return names
}

func parseInsertStructure(raw map[string]any) (InsertStructureReq, error) {
	var req InsertStructureReq
	if err := CheckParams(raw, insertStructureParams); err != nil {
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
	req.Kind = backend.StructureKind(kindStr)
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
	if req.Group, err = ParseString(raw, "group", "group", false); err != nil {
		return req, err
	}
	if req.Overwrite, err = ParseBool(raw, "overwrite", "overwrite", false); err != nil {
		return req, err
	}
	if req.Function, err = ParseString(raw, "function", "function", false); err != nil {
		return req, err
	}
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.ConstructPath, err = ParseString(raw, "construct_path", "construct-path", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertStructure(ctx context.Context, cc CallContext, req InsertStructureReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	res, err := astedit.InsertStructure(ctx, targetPath, req.Source, astedit.StructureOptions{
		Kind:                req.Kind,
		AccessModifier:      astedit.AccessModifier(req.AccessModifier),
		Placement:           astedit.Placement(req.Placement),
		TargetSymbol:        req.TargetSymbol,
		Group:               req.Group,
		Overwrite:           req.Overwrite,
		Function:            req.Function,
		SwitchOn:            req.Discriminator,
		SwitchPath:          req.ConstructPath,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return FileEditRes{
		File:     targetPath,
		Display:  req.File,
		Symbol:   req.TargetSymbol,
		Diff:     res.Diff,
		Detail:   fmt.Sprintf("Successfully inserted %s structure into %%s.", req.Kind),
		Delta:    finishDelta(),
		HasDelta: true,
	}, nil
}

func insertStructureDef() Def[InsertStructureReq, FileEditRes] {
	return Def[InsertStructureReq, FileEditRes]{
		Key:     "insert_structure",
		Summary: "Use this tool instead of replace_file_content or write_to_file when adding one Go structural construct (function, method, type, constant, variable, or switch case) to an existing file. Supply the construct kind and source code; exported names require public access. Automatically resolves imports and places the construct according to language layout rules." + automaticVerificationGuidance,
		Params:  insertStructureParams,
		Level:   LevelFile,
		CLIName: "insert-structure",
		MCPName: "semantic_insert_structure",
		Parse:   parseInsertStructure,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertStructureReq) (FileEditRes, error){
			backend.LanguageGo: runInsertStructure,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": wireExampleFile, "kind": "function", "source": "func Stop() {}"},
		PlacementKey: true,
		Batchable:    true,
	}
}
