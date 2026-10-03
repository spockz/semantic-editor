// Package operation groups Go declaration insertion handlers so parsing, mutation, and contracts stay together.
package operation

import (
	"context"
	"fmt"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
)

// InsertDeclarationReq inserts one top-level Go declaration.
type InsertDeclarationReq struct {
	Project             backend.ProjectContext
	File                string
	Source              string
	Placement           string
	TargetSymbol        string
	Visibility          string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertDeclarationReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertDeclarationReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var replaceDeclParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceReplaceDeclFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: existing constant, global variable, or type alias identifier", Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, sourceInsertTypeSymbol, sourceInsertDeclSymbol, sourceInsertDeclarationSymbol, sourceInsertConstructSymbol, sourceReplaceDeclSymbol, sourceExtractConstantReceiptSymbol}},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "New declaration source snippet", Required: true},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically clean up imports after replacement (default true)", Default: true},
}

type insertDeclarationFields struct {
	Project             backend.ProjectContext
	File                string
	Source              string
	AccessModifier      string
	Placement           string
	TargetSymbol        string
	AutoOrganizeImports bool
}

var insertDeclarationParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "One top-level Go declaration. Prefer semantic_insert_function, semantic_insert_type, or semantic_insert_decl for functions, types, constants, and variables", Required: true},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Placement qualifier: file_start, file_end (default), public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: "visibility", CLIName: "visibility", JSONName: "visibility", Type: ParamString, Description: "Optional validation constraint ensuring declaration matches exported scope ('public' or 'private')", Enums: visibilityEnum},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports required by the inserted declaration (default true)", Default: true},
}

func parseInsertDeclaration(raw map[string]any) (InsertDeclarationReq, error) {
	var req InsertDeclarationReq
	if err := CheckParams(raw, insertDeclarationParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Source, err = ParseString(raw, "source", "source", true); err != nil {
		return req, err
	}
	if req.Placement, err = ParseStringDefault(raw, "placement", "placement", string(astedit.PlacementFileEnd)); err != nil {
		return req, err
	}
	if req.TargetSymbol, err = ParseString(raw, wireTargetSymbol, "target", false); err != nil {
		return req, err
	}
	if req.Visibility, err = ParseString(raw, "visibility", "visibility", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertDeclaration(ctx context.Context, cc CallContext, req InsertDeclarationReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	opts := astedit.Options{
		Placement:           astedit.Placement(req.Placement),
		TargetSymbol:        req.TargetSymbol,
		Visibility:          req.Visibility,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	}
	if err := astedit.InsertDeclaration(ctx, targetPath, req.Source, opts); err != nil {
		return FileEditRes{}, err
	}
	return finishInsertedGoDeclaration(cc, req.File, targetPath, req.Source, "Successfully inserted declaration.", finishDelta)
}

func finishInsertedGoDeclaration(cc CallContext, file, targetPath, source, detail string, finishDelta func() (pipeline.DiagnosticDelta, string)) (FileEditRes, error) {
	symbol, line, err := goTargetAfterEdit(cc.Ctx, targetPath, source)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("edit applied to %s, but could not inspect result: %w", displayWorkPath(cc.WorkDir, file), err)
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{File: targetPath, Display: displayWorkPath(cc.WorkDir, file), Symbol: symbol, Line: line, Detail: detail, Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}, nil
}

func insertDeclarationDef() Def[InsertDeclarationReq, FileEditRes] {
	return Def[InsertDeclarationReq, FileEditRes]{
		Key:     "insert_declaration",
		Summary: "Use this tool instead of text replacement when role-based placement and generic placement controls for one top-level Go declaration fit the intent. Prefer semantic_insert_function, semantic_insert_type, semantic_insert_decl, or semantic_insert_construct for their matching kinds." + automaticVerificationGuidance,
		Params:  insertDeclarationParams,
		Level:   LevelFile,
		CLIName: "insert",
		MCPName: "semantic_insert_declaration",
		Parse:   parseInsertDeclaration,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclarationReq) (FileEditRes, error){
			backend.LanguageGo: runInsertDeclaration,
		},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "source": "func InitServer() *Server { return &Server{} }"},
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}

// InsertFunctionReq inserts one Go function or method.
type InsertFunctionReq insertDeclarationFields

// GetProjectContext returns the request project for registry dispatch.
func (r InsertFunctionReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertFunctionReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertFunctionParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "One complete Go function or method declaration, including parameters, results, and body", Required: true},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

func parseInsertFunction(raw map[string]any) (InsertFunctionReq, error) {
	fields, err := parseInsertDeclarationFields(raw, insertFunctionParams)
	return InsertFunctionReq(fields), err
}

func runInsertFunction(ctx context.Context, cc CallContext, req InsertFunctionReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	err := astedit.InsertFunction(ctx, targetPath, req.Source, astedit.FunctionOptions{
		AccessModifier:      astedit.AccessModifier(req.AccessModifier),
		Placement:           astedit.Placement(req.Placement),
		TargetSymbol:        req.TargetSymbol,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return finishInsertedGoDeclaration(cc, req.File, targetPath, req.Source, "Successfully inserted function.", finishDelta)
}

func insertFunctionDef() Def[InsertFunctionReq, FileEditRes] {
	return Def[InsertFunctionReq, FileEditRes]{
		Key: "insert_function", VariantVerb: "insert", VariantKind: "function",
		Summary: "Use this tool instead of text replacement to add one complete Go function or method. Provide parameters, results, and body. Use semantic_insert_construct with kind=function or method for the generic interface; exported names require public access." + automaticVerificationGuidance,
		Params:  insertFunctionParams,
		Level:   LevelFile,
		CLIName: "insert-func",
		MCPName: "semantic_insert_function",
		Parse:   parseInsertFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertFunctionReq) (FileEditRes, error){
			backend.LanguageGo: runInsertFunction,
		},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "source": "func (s *Server) Stop() {}"},
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}

// InsertTypeReq inserts one Go struct, interface, or type alias.
type InsertTypeReq insertDeclarationFields

// GetProjectContext returns the request project for registry dispatch.
func (r InsertTypeReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertTypeReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertTypeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Type definition Go source code snippet", Required: true},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

func parseInsertType(raw map[string]any) (InsertTypeReq, error) {
	fields, err := parseInsertDeclarationFields(raw, insertTypeParams)
	return InsertTypeReq(fields), err
}

func runInsertType(ctx context.Context, cc CallContext, req InsertTypeReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	err := astedit.InsertType(ctx, targetPath, req.Source, astedit.TypeOptions{
		AccessModifier:      astedit.AccessModifier(req.AccessModifier),
		Placement:           astedit.Placement(req.Placement),
		TargetSymbol:        req.TargetSymbol,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return finishInsertedGoDeclaration(cc, req.File, targetPath, req.Source, "Successfully inserted type.", finishDelta)
}

func insertTypeDef() Def[InsertTypeReq, FileEditRes] {
	return Def[InsertTypeReq, FileEditRes]{
		Key: "insert_type", VariantVerb: "insert", VariantKind: "type",
		Summary: "Use this tool instead of replace_file_content to add one Go struct, interface, or type alias. Use semantic_insert_construct with kind=type for the generic interface; exported names require public access." + automaticVerificationGuidance,
		Params:  insertTypeParams,
		Level:   LevelFile,
		CLIName: "insert-type",
		MCPName: "semantic_insert_type",
		Parse:   parseInsertType,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertTypeReq) (FileEditRes, error){
			backend.LanguageGo: runInsertType,
		},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "source": "type Config struct{}"},
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}

// InsertDeclReq inserts Go constants or variables with block merging.
type InsertDeclReq struct {
	Project             backend.ProjectContext
	File                string
	Source              string
	AccessModifier      string
	Group               string
	Placement           string
	TargetSymbol        string
	AutoOrganizeImports bool
}

// ReplaceDeclReq replaces one package-level Go constant, variable, or type alias.
type ReplaceDeclReq struct {
	Project             backend.ProjectContext
	File                string
	Symbol              string
	Source              string
	AutoOrganizeImports bool
}

// SetProjectContext replaces the request project during dispatch merge.
func (r *ReplaceDeclReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// GetProjectContext returns the request project for registry dispatch.
func (r ReplaceDeclReq) GetProjectContext() backend.ProjectContext { return r.Project }

// GetProjectContext returns the request project for registry dispatch.
func (r InsertDeclReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertDeclReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertDeclParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Declaration Go source code snippet", Required: true},
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "group", CLIName: "group", JSONName: "group", Type: ParamString, Description: "Group merging behavior for const/var: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')", Enums: groupEnum},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

func parseInsertDecl(raw map[string]any) (InsertDeclReq, error) {
	var req InsertDeclReq
	if err := CheckParams(raw, insertDeclParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Source, err = ParseString(raw, "source", "source", true); err != nil {
		return req, err
	}
	if req.AccessModifier, err = ParseString(raw, wireAccessModifier, "access", false); err != nil {
		return req, err
	}
	if req.Group, err = ParseString(raw, "group", "group", false); err != nil {
		return req, err
	}
	if req.Placement, err = ParseString(raw, "placement", "placement", false); err != nil {
		return req, err
	}
	if req.TargetSymbol, err = ParseString(raw, wireTargetSymbol, "target", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertDecl(ctx context.Context, cc CallContext, req InsertDeclReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	err := astedit.InsertDecl(ctx, targetPath, req.Source, astedit.DeclOptions{
		AccessModifier:      astedit.AccessModifier(req.AccessModifier),
		Group:               req.Group,
		Placement:           astedit.Placement(req.Placement),
		TargetSymbol:        req.TargetSymbol,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return finishInsertedGoDeclaration(cc, req.File, targetPath, req.Source, "Successfully inserted declaration.", finishDelta)
}

func insertDeclDef() Def[InsertDeclReq, FileEditRes] {
	return Def[InsertDeclReq, FileEditRes]{
		Key: "insert_decl", VariantVerb: "insert", VariantKind: "declaration",
		Summary: "Use this tool instead of replace_file_content to add one Go constant or variable declaration. Use semantic_insert_construct with kind=const or var for the generic interface. Existing package symbols fail closed." + automaticVerificationGuidance,
		Params:  insertDeclParams,
		Level:   LevelFile,
		CLIName: "insert-decl",
		MCPName: "semantic_insert_decl",
		Parse:   parseInsertDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclReq) (FileEditRes, error){
			backend.LanguageGo: runInsertDecl,
		},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "source": "const DefaultPort = 8080"},
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}

func parseInsertDeclarationFields(raw map[string]any, params []ParameterContract) (insertDeclarationFields, error) {
	var req insertDeclarationFields
	if err := CheckParams(raw, params); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
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
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	return req, nil
}

func parseReplaceDecl(raw map[string]any) (ReplaceDeclReq, error) {
	var req ReplaceDeclReq
	if err := CheckParams(raw, replaceDeclParams); err != nil {
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
	if req.Source, err = ParseString(raw, "source", "source", true); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	return req, nil
}

func runReplaceDecl(ctx context.Context, cc CallContext, req ReplaceDeclReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ReplaceDecl(ctx, targetPath, req.Symbol, req.Source, astedit.ReplaceDeclOptions{
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	symbol, line, err := goTargetAfterEdit(ctx, targetPath, req.Source)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("edit applied to %s, but could not inspect result: %w", displayWorkPath(cc.WorkDir, req.File), err)
	}
	if symbol == "" {
		symbol, line, err = goExistingTargetAfterEdit(ctx, targetPath, req.Symbol)
		if err != nil {
			return FileEditRes{}, fmt.Errorf("edit applied to %s, but could not inspect result: %w", displayWorkPath(cc.WorkDir, req.File), err)
		}
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: symbol, Line: line, Diff: diff, Detail: "Successfully replaced declaration.", Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}, nil
}

func replaceDeclDef() Def[ReplaceDeclReq, FileEditRes] {
	return Def[ReplaceDeclReq, FileEditRes]{
		Key: "replace_decl", VariantVerb: "replace", VariantKind: "declaration",
		Summary: "Use this tool instead of replace_file_content to update one package-level Go constant, variable, or type alias. Use semantic_replace_construct with symbol and kind=const, var, or type for the generic interface; the selected name must match.",
		Params:  replaceDeclParams,
		Level:   LevelFile,
		CLIName: "replace-decl",
		MCPName: semanticReplaceDeclToolName,
		Parse:   parseReplaceDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceDeclReq) (FileEditRes, error){
			backend.LanguageGo: runReplaceDecl,
		},
		Format:        formatFileEdit,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "DefaultPort", "source": "const DefaultPort = 8443"},
		BufferedBatch: true, Batchable: true,
	}
}
