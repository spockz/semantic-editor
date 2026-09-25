// Package operation groups Go declaration insertion handlers so parsing, mutation, and contracts stay together.
package operation

import (
	"context"
	"semedit/internal/astedit"
	"semedit/internal/backend"
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target Go source file", Required: true},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Existing constant, global variable, or type alias identifier", Required: true},
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Go source target; relative paths resolve from the active semedit workspace root", Required: true},
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
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.TargetSymbol, Detail: "Successfully inserted declaration into %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func insertDeclarationDef() Def[InsertDeclarationReq, FileEditRes] {
	return Def[InsertDeclarationReq, FileEditRes]{
		Key:     "insert_declaration",
		Summary: "Use this tool instead of replace_file_content or write_to_file when adding one top-level Go declaration and its generic placement controls fit the task. Prefer semantic_insert_function, semantic_insert_type, or semantic_insert_decl for functions, methods, types, constants, or variables so the operation matches the declaration. Accurately places one declaration at file boundaries, public/private sections, or relative to an existing symbol." + automaticVerificationGuidance,
		Params:  insertDeclarationParams,
		Level:   LevelFile,
		CLIName: "insert",
		MCPName: "semantic_insert_declaration",
		Parse:   parseInsertDeclaration,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclarationReq) (FileEditRes, error){
			backend.LanguageGo: runInsertDeclaration,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": wireExampleFile, "source": "func InitServer() *Server { return &Server{} }"},
		PlacementKey: true,
		Batchable:    true,
	}
}

// InsertFunctionReq inserts one Go function or method.
type InsertFunctionReq insertDeclarationFields

// GetProjectContext returns the request project for registry dispatch.
func (r InsertFunctionReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertFunctionReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertFunctionParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Go source target; relative paths resolve from the active semedit workspace root", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Function or method Go source code snippet", Required: true},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
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
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.TargetSymbol, Detail: "Successfully inserted function into %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func insertFunctionDef() Def[InsertFunctionReq, FileEditRes] {
	return Def[InsertFunctionReq, FileEditRes]{
		Key:     "insert_function",
		Summary: "Use this tool instead of replace_file_content or write_to_file when adding one Go function or method to an existing file. Supply one function or method declaration; exported names require public access. Automatically clusters methods near their receiver types and enforces public vs private section partitioning." + automaticVerificationGuidance,
		Params:  insertFunctionParams,
		Level:   LevelFile,
		CLIName: "insert-func",
		MCPName: "semantic_insert_function",
		Parse:   parseInsertFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertFunctionReq) (FileEditRes, error){
			backend.LanguageGo: runInsertFunction,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": wireExampleFile, "source": "func (s *Server) Stop() {}"},
		PlacementKey: true,
		Batchable:    true,
	}
}

// InsertTypeReq inserts one Go struct, interface, or type alias.
type InsertTypeReq insertDeclarationFields

// GetProjectContext returns the request project for registry dispatch.
func (r InsertTypeReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertTypeReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertTypeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Go source target; relative paths resolve from the active semedit workspace root", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Type definition Go source code snippet", Required: true},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
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
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.TargetSymbol, Detail: "Successfully inserted type into %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func insertTypeDef() Def[InsertTypeReq, FileEditRes] {
	return Def[InsertTypeReq, FileEditRes]{
		Key:     "insert_type",
		Summary: "Use this tool instead of replace_file_content or write_to_file when adding one Go struct, interface, or type alias to an existing file. Supply one type declaration; exported names require public access. Automatically anchors types within the appropriate section and resolves package imports." + automaticVerificationGuidance,
		Params:  insertTypeParams,
		Level:   LevelFile,
		CLIName: "insert-type",
		MCPName: "semantic_insert_type",
		Parse:   parseInsertType,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertTypeReq) (FileEditRes, error){
			backend.LanguageGo: runInsertType,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": wireExampleFile, "source": "type Config struct{}"},
		PlacementKey: true,
		Batchable:    true,
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
	Overwrite           bool
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Go source target; relative paths resolve from the active semedit workspace root", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Declaration Go source code snippet", Required: true},
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
	{Name: "group", CLIName: "group", JSONName: "group", Type: ParamString, Description: "Group merging behavior for const/var: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')", Enums: groupEnum},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
	{Name: "overwrite", CLIName: "overwrite", JSONName: "overwrite", Type: ParamBoolean, Description: "Replace an existing declaration with the same name (default false)"},
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
	if req.Overwrite, err = ParseBool(raw, "overwrite", "overwrite", false); err != nil {
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
		Overwrite:           req.Overwrite,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.TargetSymbol, Detail: "Successfully inserted declaration into %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func insertDeclDef() Def[InsertDeclReq, FileEditRes] {
	return Def[InsertDeclReq, FileEditRes]{
		Key:     "insert_decl",
		Summary: "Use this tool instead of replace_file_content or write_to_file when adding one Go constant or variable to an existing file. A single declaration with default placement merges into the parenthesized block matching its kind and visibility. Sentinel-shaped vars (`Err*`) resolve to the block already holding sentinel errors, placed alphabetically; without such a block they land alphabetically at the canonical section location, so sentinel grouping emerges without managed state. Existing package symbols are rejected by default; use semantic_replace_decl for updates or set overwrite=true intentionally. Use `target_symbol` with before/after placement for explicit anchoring, or `group: standalone` to opt out of merging." + automaticVerificationGuidance,
		Params:  insertDeclParams,
		Level:   LevelFile,
		CLIName: "insert-decl",
		MCPName: "semantic_insert_decl",
		Parse:   parseInsertDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclReq) (FileEditRes, error){
			backend.LanguageGo: runInsertDecl,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": wireExampleFile, "source": "const DefaultPort = 8080", "overwrite": false},
		PlacementKey: true,
		Batchable:    true,
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
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.Symbol, Diff: diff, Detail: "Successfully replaced declaration " + req.Symbol + " in %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func replaceDeclDef() Def[ReplaceDeclReq, FileEditRes] {
	return Def[ReplaceDeclReq, FileEditRes]{
		Key:     "replace_decl",
		Summary: "Use this tool instead of replace_file_content or inserting a duplicate declaration when updating an existing package-level Go constant, variable, or type alias. The selected name must match the replacement declaration; grouped specs and collisions are checked before writing.",
		Params:  replaceDeclParams,
		Level:   LevelFile,
		CLIName: "replace-decl",
		MCPName: "semantic_replace_decl",
		Parse:   parseReplaceDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceDeclReq) (FileEditRes, error){
			backend.LanguageGo: runReplaceDecl,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "symbol": "DefaultPort", "source": "const DefaultPort = 8443"},
		Batchable:  true,
	}
}
