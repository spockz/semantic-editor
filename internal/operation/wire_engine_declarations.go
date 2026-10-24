// Package operation groups Go declaration insertion handlers so parsing, mutation, and contracts stay together.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
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

var replaceDeclParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target source file containing the declaration", Required: true, SourceFields: sourceReplaceDeclFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Description: "Target: existing Go declaration or uniquely selected Java declaration", Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, sourceInsertTypeSymbol, sourceInsertDeclSymbol, sourceInsertDeclarationSymbol, sourceInsertConstructSymbol, sourceReplaceDeclSymbol, sourceExtractValueReceiptSymbol}},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Complete replacement declaration; Java replacements preserve the selected declaration kind and name", Required: true},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically clean up imports after replacement (default true)", Default: true},
}, readProjectParams...)

type insertDeclarationFields struct {
	Project             backend.ProjectContext
	File                string
	InType              string
	Source              string
	AccessModifier      string
	Placement           string
	TargetSymbol        string
	AutoOrganizeImports bool
}

var insertDeclarationParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
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

func runReplaceDeclJava(ctx context.Context, cc CallContext, req ReplaceDeclReq) (FileEditRes, error) {
	root, file, before, after, err := runJavaInspectedSourceEdit(ctx, cc, req.Project, req.File, req.Symbol, "declaration replacement", nil, effectiveAutoOrganize(cc, req.AutoOrganizeImports), func(existing []byte) ([]byte, error) {
		if err := javacst.ValidateDeclarationReplacement(existing, []byte(req.Source)); err != nil {
			return nil, err
		}
		return []byte(req.Source), nil
	})
	if err != nil {
		return FileEditRes{}, err
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("derive Java declaration receipt path: %w", err)
	}
	if !filepath.IsLocal(relative) {
		return FileEditRes{}, fmt.Errorf("java declaration target %q escapes workspace %q", file, root)
	}
	relative = filepath.ToSlash(relative)
	display := displayWorkPath(root, req.File)
	draft := EditReceiptDraft{
		Operation: "replace_decl", Status: EditPersisted,
		ReadSet:        []FileRevision{{Path: relative, Revision: structuralRevision(before)}},
		PlannedChanges: []FileRevision{{Path: relative, Revision: structuralRevision(after)}},
	}
	return FileEditRes{
		File: file, Display: display, Symbol: req.Symbol,
		Diff:   astedit.UnifiedDiff(display, string(before), string(after)),
		Detail: "Successfully replaced Java declaration.", DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true, ReceiptDraft: draft,
	}, nil
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
		MCPName: semanticInsertDeclarationToolName,
		Parse:   parseInsertDeclaration,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclarationReq) (FileEditRes, error){
			backend.LanguageGo: bufferedFileEditHandler("insert_declaration", func(req InsertDeclarationReq) string { return req.File }, runInsertDeclaration),
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "source": "func InitServer() *Server { return &Server{} }"},
		Output:     declarationOutputContract(false),
		FinalizeOutput: func(cc CallContext, _ InsertDeclarationReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: projectDeclarationOutput,
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target source file for the declaration", Required: true, SourceFields: sourceFileContinuations},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "One complete Go function or Java method or constructor declaration", Required: true},
	{Name: "in_type", CLIName: "in-type", JSONName: "in_type", Type: ParamString, Role: RoleSelector, Description: "Java owner type; omit only when the file has exactly one top-level type", SourceFields: []string{"semantic_insert_function.in_type"}},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Go placement qualifier or Java type_start, type_end, before_symbol, after_symbol", Enums: append(slices.Clone(placementEnum), "type_start", "type_end")},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

var insertFunctionReadParams = append(append([]ParameterContract{}, insertFunctionParams...), readProjectParams...)

func parseInsertFunction(raw map[string]any) (InsertFunctionReq, error) {
	fields, err := parseInsertDeclarationFields(raw, insertFunctionReadParams)
	if err != nil {
		return InsertFunctionReq{}, err
	}
	fields.InType, err = ParseString(raw, "in_type", "in-type", false)
	if err != nil {
		return InsertFunctionReq{}, err
	}
	fields.Project, err = parseReadProject(raw, fields.File)
	if err != nil {
		return InsertFunctionReq{}, err
	}
	if fields.Project.Language == "" {
		fields.Project.Language = backend.LanguageGo
	}
	return InsertFunctionReq(fields), nil
}

func runInsertFunction(ctx context.Context, cc CallContext, req InsertFunctionReq) (FileEditRes, error) {
	if req.Project.Language == backend.LanguageJava {
		return runInsertFunctionJava(ctx, cc, req)
	}
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

//nolint:dupl // These public operation descriptors share structure but retain distinct typed schemas and semantics.
func insertFunctionDef() Def[InsertFunctionReq, FileEditRes] {
	return Def[InsertFunctionReq, FileEditRes]{
		Key: "insert_function", VariantVerb: "insert", VariantKind: "function",
		Summary: "Use this tool instead of text replacement to add one complete Go function or method, or one Java method or constructor. The complete parameters, results, and body are supplied as a declaration. Java inserts into one selected owner type and delegates formatting and verification to JDT LS." + automaticVerificationGuidance,
		Params:  insertFunctionReadParams,
		Level:   LevelFile,
		CLIName: "insert-func",
		MCPName: semanticInsertFunctionToolName,
		Parse:   parseInsertFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertFunctionReq) (FileEditRes, error){
			backend.LanguageGo:   bufferedFileEditHandler("insert_function", func(req InsertFunctionReq) string { return req.File }, runInsertFunction),
			backend.LanguageJava: runInsertFunctionJava,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "source": "func (s *Server) Stop() {}"},
		Output:     insertFunctionOutputContract(),
		FinalizeOutput: func(cc CallContext, _ InsertFunctionReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: projectInsertFunctionOutput,
		PlacementKey:  true,
		BufferedBatch: true, Batchable: true,
	}
}

func runInsertFunctionJava(ctx context.Context, cc CallContext, req InsertFunctionReq) (FileEditRes, error) {
	return runInsertJavaDeclaration(ctx, cc, req.Project, req.File, req.InType, req.Source, req.Placement, req.TargetSymbol, "insert_function", "method", "method insertion", "Inserted Java method or constructor.", req.AutoOrganizeImports)
}

func runInsertTypeJava(ctx context.Context, cc CallContext, req InsertTypeReq) (FileEditRes, error) {
	return runInsertJavaDeclaration(ctx, cc, req.Project, req.File, req.InType, req.Source, req.Placement, req.TargetSymbol, "insert_type", "nested type", "nested type insertion", "Inserted Java nested type.", req.AutoOrganizeImports)
}

func runInsertJavaDeclaration(ctx context.Context, cc CallContext, project backend.ProjectContext, file, owner, source, placement, targetSymbol, operation, declarationKind, actionLabel, detail string, organizeImports bool) (FileEditRes, error) {
	if owner == "" {
		target, err := sourcebuf.ReadFile(ctx, resolveWorkPath(cc.WorkDir, file))
		if err != nil {
			return FileEditRes{}, fmt.Errorf("read Java %s insertion target: %w", declarationKind, err)
		}
		owner, err = javacst.SoleJavaTopLevelType(target)
		if err != nil {
			return FileEditRes{}, err
		}
	}
	name, err := javacst.JavaDeclarationName([]byte(source))
	if err != nil {
		return FileEditRes{}, err
	}
	target := ""
	if targetSymbol != "" {
		target, err = javaStructuralMemberName(owner, targetSymbol)
		if err != nil {
			return FileEditRes{}, fmt.Errorf("target: %w", err)
		}
	}
	return runInsertJavaMember(ctx, cc, project, file, owner, name, source, placement, target, operation, actionLabel, detail, organizeImports)
}

func runInsertJavaMember(ctx context.Context, cc CallContext, project backend.ProjectContext, filePath, owner, name, source, placement, target, operation, action, detail string, organizeImports bool) (FileEditRes, error) {
	root, file, before, after, err := runJavaInspectedSourceEdit(ctx, cc, project, filePath, owner, action, nil, effectiveAutoOrganize(cc, organizeImports), func(existing []byte) ([]byte, error) {
		return javacst.InsertJavaMember(existing, []byte(source), name, placement, target)
	})
	if err != nil {
		return FileEditRes{}, err
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("derive Java member receipt path for %q: %w", file, err)
	}
	if !filepath.IsLocal(relative) {
		return FileEditRes{}, fmt.Errorf("java member target %q escapes workspace %q", file, root)
	}
	relative = filepath.ToSlash(relative)
	display := displayWorkPath(root, filePath)
	revision := structuralRevision(after)
	symbol := owner + "." + name
	draft := EditReceiptDraft{
		Operation: operation, Status: EditPersisted,
		ReadSet:        []FileRevision{{Path: relative, Revision: structuralRevision(before)}},
		PlannedChanges: []FileRevision{{Path: relative, Revision: revision}},
		Continuations:  []SymbolTarget{{File: relative, Symbol: symbol, ExpectedRevision: &revision}},
	}
	return FileEditRes{
		File: file, Display: display, Symbol: symbol, InType: owner,
		Diff: astedit.UnifiedDiff(display, string(before), string(after)), Detail: detail,
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true, ReceiptDraft: draft,
	}, nil
}

// InsertTypeReq inserts one Go struct, interface, or type alias.
type InsertTypeReq insertDeclarationFields

// GetProjectContext returns the request project for registry dispatch.
func (r InsertTypeReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertTypeReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertTypeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Go source file or Java source file receiving the type", Required: true, SourceFields: sourceFileContinuations},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "One complete Go type declaration or Java nested type declaration", Required: true},
	{Name: "in_type", CLIName: "in-type", JSONName: "in_type", Type: ParamString, Role: RoleSelector, Description: "Java owner type for nested type insertion; omit only when the file has exactly one top-level type", SourceFields: []string{"semantic_insert_type.in_type"}},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: wireAccessModifier, CLIName: "access", JSONName: wireAccessModifier, Type: ParamString, Description: "Go access modifier: infer, public, or private", Enums: accessEnum, DynamicEnums: supportedAccessModifiers},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Go placement qualifier or Java type_start, type_end, before_symbol, after_symbol", Enums: append(slices.Clone(placementEnum), "type_start", "type_end")},
	{Name: wireTargetSymbol, CLIName: "target", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

var insertTypeReadParams = append(append([]ParameterContract{}, insertTypeParams...), readProjectParams...)

func parseInsertType(raw map[string]any) (InsertTypeReq, error) {
	fields, err := parseInsertDeclarationFields(raw, insertTypeReadParams)
	if err != nil {
		return InsertTypeReq{}, err
	}
	fields.InType, err = ParseString(raw, "in_type", "in-type", false)
	if err != nil {
		return InsertTypeReq{}, err
	}
	fields.Project, err = parseReadProject(raw, fields.File)
	if err != nil {
		return InsertTypeReq{}, err
	}
	if fields.Project.Language == "" {
		fields.Project.Language = backend.LanguageGo
	}
	return InsertTypeReq(fields), nil
}

func runInsertType(ctx context.Context, cc CallContext, req InsertTypeReq) (FileEditRes, error) {
	if req.Project.Language == backend.LanguageJava {
		return runInsertTypeJava(ctx, cc, req)
	}
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

//nolint:dupl // These public operation descriptors share structure but retain distinct typed schemas and semantics.
func insertTypeDef() Def[InsertTypeReq, FileEditRes] {
	return Def[InsertTypeReq, FileEditRes]{
		Key: "insert_type", VariantVerb: "insert", VariantKind: "type",
		Summary: "Use this tool instead of replace_file_content or text insertion to add one Go struct, interface, or type alias, or one Java nested type to a uniquely selected owner type. Java member ordering follows type visibility and JDT LS formats and verifies the result." + automaticVerificationGuidance,
		Params:  insertTypeReadParams,
		Level:   LevelFile,
		CLIName: "insert-type",
		MCPName: semanticInsertTypeToolName,
		Parse:   parseInsertType,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertTypeReq) (FileEditRes, error){
			backend.LanguageGo:   bufferedFileEditHandler("insert_type", func(req InsertTypeReq) string { return req.File }, runInsertType),
			backend.LanguageJava: runInsertTypeJava,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "source": "type Config struct{}"},
		Output:     insertTypeOutputContract(),
		FinalizeOutput: func(cc CallContext, _ InsertTypeReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: projectInsertTypeOutput,
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
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
		Key: "insert_decl", VariantVerb: "insert", VariantKind: wireDeclaration,
		Summary: "Use this tool instead of replace_file_content to add one Go constant or variable declaration. Use semantic_insert_construct with kind=const or var for the generic interface. Existing package symbols fail closed." + automaticVerificationGuidance,
		Params:  insertDeclParams,
		Level:   LevelFile,
		CLIName: "insert-decl",
		MCPName: semanticInsertDeclToolName,
		Parse:   parseInsertDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclReq) (FileEditRes, error){
			backend.LanguageGo: bufferedFileEditHandler("insert_decl", func(req InsertDeclReq) string { return req.File }, runInsertDecl),
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "source": "const DefaultPort = 8080"},
		Output:     declarationOutputContract(false),
		FinalizeOutput: func(cc CallContext, _ InsertDeclReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: projectDeclarationOutput,
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
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	} else if req.Project.Language == "" {
		req.Project.Language = backend.LanguageGo
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
		Key: "replace_decl", VariantVerb: "replace", VariantKind: wireDeclaration,
		Summary: "Use this tool instead of replace_file_content to replace one Go package-level constant, variable, or type alias, or one uniquely selected Java method, field, or nested type declaration. Java replacements preserve declaration kind and name and run JDT LS formatting and diagnostics.",
		Params:  replaceDeclParams,
		Level:   LevelFile,
		CLIName: "replace-decl",
		MCPName: semanticReplaceDeclToolName,
		Parse:   parseReplaceDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceDeclReq) (FileEditRes, error){
			backend.LanguageGo:   bufferedFileEditHandler("replace_decl", func(req ReplaceDeclReq) string { return req.File }, runReplaceDecl),
			backend.LanguageJava: runReplaceDeclJava,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "symbol": "DefaultPort", "source": "const DefaultPort = 8443"},
		Output:     declarationOutputContract(true),
		FinalizeOutput: func(cc CallContext, _ ReplaceDeclReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: projectReplaceDeclarationOutput,
		BufferedBatch: true, Batchable: true,
	}
}

func declarationOutputContract(includeDiff bool) *OperationOutputContract {
	properties := map[string]ValueSchema{
		"file":   nonEmptyStringSchema(),
		"symbol": nonEmptyStringSchema(),
		"line":   {Kind: SchemaInteger},
	}
	if includeDiff {
		properties["diff"] = ValueSchema{Kind: SchemaString}
	}
	return fileEditOutputContract(properties, []OutputContinuation{
		{Field: "file", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: fileOutputConsumers, Optional: true},
		{Field: "symbol", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: symbolOutputConsumers, Optional: true},
	})
}

func insertFunctionOutputContract() *OperationOutputContract {
	properties := map[string]ValueSchema{
		"file":    nonEmptyStringSchema(),
		"symbol":  nonEmptyStringSchema(),
		"in_type": nonEmptyStringSchema(),
		"line":    {Kind: SchemaInteger},
	}
	return fileEditOutputContract(properties, []OutputContinuation{
		{Field: "file", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: fileOutputConsumers, Optional: true},
		{Field: "symbol", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: symbolOutputConsumers, Optional: true},
		{Field: "in_type", Schema: nonEmptyStringSchema(), Role: RoleSelector, Consumers: []string{semanticInsertFunctionToolName}, Optional: true},
	})
}

func insertTypeOutputContract() *OperationOutputContract {
	properties := map[string]ValueSchema{
		"file":    nonEmptyStringSchema(),
		"symbol":  nonEmptyStringSchema(),
		"in_type": nonEmptyStringSchema(),
		"line":    {Kind: SchemaInteger},
	}
	return fileEditOutputContract(properties, []OutputContinuation{
		{Field: "file", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: fileOutputConsumers, Optional: true},
		{Field: "symbol", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: symbolOutputConsumers, Optional: true},
		{Field: "in_type", Schema: nonEmptyStringSchema(), Role: RoleSelector, Consumers: []string{semanticInsertTypeToolName}, Optional: true},
	})
}

func projectDeclarationOutput(result FileEditRes) (any, error) {
	return projectFileEditOutput(result, "file", "symbol", "line")
}

func projectInsertFunctionOutput(result FileEditRes) (any, error) {
	return projectFileEditOutput(result, "file", "symbol", "in_type", "line")
}

func projectInsertTypeOutput(result FileEditRes) (any, error) {
	return projectFileEditOutput(result, "file", "symbol", "in_type", "line")
}

func projectReplaceDeclarationOutput(result FileEditRes) (any, error) {
	return projectFileEditOutput(result, "file", "symbol", "line", "diff")
}
