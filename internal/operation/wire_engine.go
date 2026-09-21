// Package operation wires host-engine editing operations into the central registry.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/pipeline"
)

var (
	placementEnum  = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol"}
	accessEnum     = []string{"infer", "public", "private", "protected", "package-private"}
	visibilityEnum = []string{"public", "private"}
	groupEnum      = []string{"append", "standalone"}
	caseEnum       = []string{"first", "last", "before_default", "before", "after"}
)

// FileOutcome is implemented by results that record a written file for batch post-processing.
type FileOutcome interface {
	WrittenFile() string
}

// AppendDiagnosticDelta appends a compiler diagnostic delta to a response base string.
// It mirrors the MCP server helper retired in the ingress migration step.
func AppendDiagnosticDelta(base string, delta pipeline.DiagnosticDelta) string {
	if len(delta.Introduced) == 0 && len(delta.Resolved) == 0 && len(delta.Suggestions) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
		fmt.Fprintf(&sb, "\nDiagnostics delta (net %d):\n", delta.NetDelta)
		if len(delta.Resolved) > 0 {
			fmt.Fprintf(&sb, "Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
		}
		if len(delta.Introduced) > 0 {
			fmt.Fprintf(&sb, "Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
		}
	}
	if len(delta.Suggestions) > 0 {
		fmt.Fprintf(&sb, "\nActionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	return sb.String()
}

// resolveWorkPath mirrors the MCP server path resolution: relative paths join the workspace root.
func resolveWorkPath(workDir, file string) string {
	if file == "" {
		return workDir
	}
	if workDir != "" && !filepath.IsAbs(file) {
		return filepath.Join(workDir, file)
	}
	return file
}

// effectiveAutoOrganize applies the batch deferral flag over the parsed default.
func effectiveAutoOrganize(cc CallContext, parsed bool) bool {
	if cc.DeferImports {
		return false
	}
	return parsed
}

func surroundingDelta(ctx context.Context, workDir string) ([]string, func() pipeline.DiagnosticDelta) {
	before, _ := pipeline.CheckDiagnostics(ctx, workDir)
	return before, func() pipeline.DiagnosticDelta {
		after, _ := pipeline.CheckDiagnostics(ctx, workDir)
		return pipeline.ComputeDelta(before, after)
	}
}

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

// FileEditRes reports a single-file mutation for response rendering and batch post-processing.
type FileEditRes struct {
	File     string
	Display  string
	Symbol   string
	Diff     string
	Detail   string
	Delta    pipeline.DiagnosticDelta
	HasDelta bool
}

// WrittenFile returns the resolved mutated path for batch post-processing.
func (r FileEditRes) WrittenFile() string { return r.File }

func formatFileEdit(res FileEditRes) (string, error) {
	text := fmt.Sprintf(res.Detail, res.Display)
	if res.Diff != "" {
		text += "\n" + res.Diff
	}
	if res.HasDelta {
		text = AppendDiagnosticDelta(text, res.Delta)
	}
	return text, nil
}

var insertDeclarationParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target file path", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Go declaration code snippet to insert", Required: true},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Placement qualifier: file_start, file_end (default), public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: "target_symbol", CLIName: "target", JSONName: "target_symbol", Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: "visibility", CLIName: "visibility", JSONName: "visibility", Type: ParamString, Description: "Optional validation constraint ensuring declaration matches exported scope ('public' or 'private')", Enums: visibilityEnum},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "Automatically resolve and organize package imports required by the inserted declaration (default true)", Default: true},
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
	if req.TargetSymbol, err = ParseString(raw, "target_symbol", "target", false); err != nil {
		return req, err
	}
	if req.Visibility, err = ParseString(raw, "visibility", "visibility", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, "auto_organize_imports", "auto-organize-imports", true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertDeclaration(ctx context.Context, cc CallContext, req InsertDeclarationReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	_, finishDelta := surroundingDelta(ctx, cc.WorkDir)
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
		Summary: "Use this tool instead of replace_file_content or write_to_file whenever adding a new top-level function, method, type, or constant to an existing Go file. Accurately places declarations at file boundaries, public/private sections, or relative to existing symbols without coordinate hunting.",
		Params:  insertDeclarationParams,
		Level:   LevelFile,
		CLIName: "insert",
		MCPName: "semantic_insert_declaration",
		Parse:   parseInsertDeclaration,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclarationReq) (FileEditRes, error){
			backend.LanguageGo: runInsertDeclaration,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": "api/server.go", "source": "func InitServer() *Server { return &Server{} }"},
		PlacementKey: true,
		Batchable:    true,
	}
}

// InsertFunctionReq inserts one Go function or method.
type InsertFunctionReq struct {
	Project             backend.ProjectContext
	File                string
	Source              string
	AccessModifier      string
	Placement           string
	TargetSymbol        string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertFunctionReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertFunctionReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertFunctionParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target file path", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Function or method Go source code snippet", Required: true},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: "access_modifier", CLIName: "access", JSONName: "access_modifier", Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: "target_symbol", CLIName: "target", JSONName: "target_symbol", Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

func parseInsertFunction(raw map[string]any) (InsertFunctionReq, error) {
	var req InsertFunctionReq
	if err := CheckParams(raw, insertFunctionParams); err != nil {
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
	if req.AccessModifier, err = ParseString(raw, "access_modifier", "access", false); err != nil {
		return req, err
	}
	if req.Placement, err = ParseString(raw, "placement", "placement", false); err != nil {
		return req, err
	}
	if req.TargetSymbol, err = ParseString(raw, "target_symbol", "target", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, "auto_organize_imports", "auto-organize-imports", true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertFunction(ctx context.Context, cc CallContext, req InsertFunctionReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	_, finishDelta := surroundingDelta(ctx, cc.WorkDir)
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
		Summary: "Use this tool instead of replace_file_content whenever adding a new top-level function or method to an existing Go file. Automatically clusters methods near their receiver types and enforces public vs private section partitioning.",
		Params:  insertFunctionParams,
		Level:   LevelFile,
		CLIName: "insert-func",
		MCPName: "semantic_insert_function",
		Parse:   parseInsertFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertFunctionReq) (FileEditRes, error){
			backend.LanguageGo: runInsertFunction,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": "api/server.go", "source": "func (s *Server) Stop() {}"},
		PlacementKey: true,
		Batchable:    true,
	}
}

// InsertTypeReq inserts one Go struct, interface, or type alias.
type InsertTypeReq struct {
	Project             backend.ProjectContext
	File                string
	Source              string
	AccessModifier      string
	Placement           string
	TargetSymbol        string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertTypeReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertTypeReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertTypeParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target file path", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Type definition Go source code snippet", Required: true},
	// CLI today exposes --no-imports (negated); generation normalizes to --auto-organize-imports.
	{Name: "access_modifier", CLIName: "access", JSONName: "access_modifier", Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: "target_symbol", CLIName: "target", JSONName: "target_symbol", Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
}

func parseInsertType(raw map[string]any) (InsertTypeReq, error) {
	var req InsertTypeReq
	if err := CheckParams(raw, insertTypeParams); err != nil {
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
	if req.AccessModifier, err = ParseString(raw, "access_modifier", "access", false); err != nil {
		return req, err
	}
	if req.Placement, err = ParseString(raw, "placement", "placement", false); err != nil {
		return req, err
	}
	if req.TargetSymbol, err = ParseString(raw, "target_symbol", "target", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, "auto_organize_imports", "auto-organize-imports", true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertType(ctx context.Context, cc CallContext, req InsertTypeReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	_, finishDelta := surroundingDelta(ctx, cc.WorkDir)
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
		Summary: "Use this tool instead of replace_file_content whenever adding a new struct, interface, or type alias to an existing Go file. Automatically anchors types within the appropriate section and resolves package imports.",
		Params:  insertTypeParams,
		Level:   LevelFile,
		CLIName: "insert-type",
		MCPName: "semantic_insert_type",
		Parse:   parseInsertType,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertTypeReq) (FileEditRes, error){
			backend.LanguageGo: runInsertType,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": "api/server.go", "source": "type Config struct{}"},
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
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertDeclReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertDeclReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertDeclParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target file path", Required: true},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Declaration Go source code snippet", Required: true},
	{Name: "access_modifier", CLIName: "access", JSONName: "access_modifier", Type: ParamString, Description: "Access modifier (infer, public, private, protected, package-private)", Enums: accessEnum},
	{Name: "group", CLIName: "group", JSONName: "group", Type: ParamString, Description: "Group merging behavior for const/var: 'append' merges into existing block, 'standalone' inserts separate declaration (default 'append')", Enums: groupEnum},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Optional placement qualifier: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: "target_symbol", CLIName: "target", JSONName: "target_symbol", Type: ParamString, Description: "Target symbol identifier required when placement is before_symbol or after_symbol"},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "Automatically resolve and organize package imports (default true)", Default: true},
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
	if req.AccessModifier, err = ParseString(raw, "access_modifier", "access", false); err != nil {
		return req, err
	}
	if req.Group, err = ParseString(raw, "group", "group", false); err != nil {
		return req, err
	}
	if req.Placement, err = ParseString(raw, "placement", "placement", false); err != nil {
		return req, err
	}
	if req.TargetSymbol, err = ParseString(raw, "target_symbol", "target", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, "auto_organize_imports", "auto-organize-imports", true); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertDecl(ctx context.Context, cc CallContext, req InsertDeclReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	_, finishDelta := surroundingDelta(ctx, cc.WorkDir)
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
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.TargetSymbol, Detail: "Successfully inserted declaration into %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func insertDeclDef() Def[InsertDeclReq, FileEditRes] {
	return Def[InsertDeclReq, FileEditRes]{
		Key:     "insert_decl",
		Summary: "Use this tool instead of replace_file_content whenever adding constants, variables, or declarations to an existing Go file. A single declaration with default placement merges into the parenthesized block matching its kind and visibility. Sentinel-shaped vars (`Err*`) resolve to the block already holding sentinel errors, placed alphabetically; without such a block they land alphabetically at the canonical section location, so sentinel grouping emerges without managed state. Use `target_symbol` with before/after placement for explicit anchoring, or `group: standalone` to opt out of merging.",
		Params:  insertDeclParams,
		Level:   LevelFile,
		CLIName: "insert-decl",
		MCPName: "semantic_insert_decl",
		Parse:   parseInsertDecl,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertDeclReq) (FileEditRes, error){
			backend.LanguageGo: runInsertDecl,
		},
		Format:       formatFileEdit,
		ExampleRaw:   map[string]any{"file": "api/server.go", "source": "const DefaultPort = 8080"},
		PlacementKey: true,
		Batchable:    true,
	}
}

// OrganizeImportsReq manages import statements of a file or workspace scope.
type OrganizeImportsReq struct {
	Project backend.ProjectContext
	File    string
	Add     []string
	Remove  []string
}

// GetProjectContext returns the request project for registry dispatch.
func (r OrganizeImportsReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *OrganizeImportsReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var organizeImportsParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Optional file or directory path to process (defaults to entire workspace)"},
	{Name: "add", CLIName: "add", JSONName: "add", Type: ParamStringSlice, Description: "Optional list of import paths to explicitly add. Supports 'path', 'alias path', or '_ path'."},
	{Name: "remove", CLIName: "remove", JSONName: "remove", Type: ParamStringSlice, Description: "Optional list of import paths to explicitly remove."},
}

func parseOrganizeImports(raw map[string]any) (OrganizeImportsReq, error) {
	var req OrganizeImportsReq
	if err := CheckParams(raw, organizeImportsParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", false); err != nil {
		return req, err
	}
	if req.Add, err = ParseStringSlice(raw, "add", "add"); err != nil {
		return req, err
	}
	if req.Remove, err = ParseStringSlice(raw, "remove", "remove"); err != nil {
		return req, err
	}
	return req, nil
}

func runOrganizeImports(ctx context.Context, cc CallContext, req OrganizeImportsReq) (FileEditRes, error) {
	var paths []string
	if req.File != "" {
		paths = []string{resolveWorkPath(cc.WorkDir, req.File)}
	}
	_, finishDelta := surroundingDelta(ctx, cc.WorkDir)
	err := pipeline.OrganizeImportsWithOptions(ctx, cc.WorkDir, pipeline.ImportOptions{
		Add:    req.Add,
		Remove: req.Remove,
	}, paths...)
	if err != nil {
		return FileEditRes{}, err
	}
	written := cc.WorkDir
	if len(paths) > 0 {
		written = paths[0]
	}
	return FileEditRes{File: written, Display: req.File, Symbol: req.File, Detail: "Successfully organized imports.", Delta: finishDelta(), HasDelta: true}, nil
}

func organizeImportsDef() Def[OrganizeImportsReq, FileEditRes] {
	return Def[OrganizeImportsReq, FileEditRes]{
		Key:     "organize_imports",
		Summary: "Use this tool to format imports, resolve missing package imports, and strip unused imports across specified files or the workspace. Supports explicit package additions (including aliases and blank imports) and explicit removals.",
		Params:  organizeImportsParams,
		Level:   LevelFile,
		CLIName: "imports",
		MCPName: "semantic_organize_imports",
		Parse:   parseOrganizeImports,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, OrganizeImportsReq) (FileEditRes, error){
			backend.LanguageGo: runOrganizeImports,
		},
		Format: func(res FileEditRes) (string, error) {
			return AppendDiagnosticDelta("Successfully organized imports.", res.Delta), nil
		},
		ExampleRaw: map[string]any{"file": "api/server.go"},
		Batchable:  true,
	}
}

// AddBuildDependencyReq fetches one external module into the build (go get + go mod tidy).
type AddBuildDependencyReq struct {
	Project backend.ProjectContext
	Package string
}

// GetProjectContext returns the request project for registry dispatch.
func (r AddBuildDependencyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *AddBuildDependencyReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

// BuildDependencyRes reports a build-file mutation for response rendering and batch post-processing.
type BuildDependencyRes struct {
	Package string
	GoMod   string
}

// WrittenFile returns the mutated go.mod path for batch post-processing.
func (r BuildDependencyRes) WrittenFile() string { return r.GoMod }

var addBuildDependencyParams = []ParameterContract{
	{Name: "package", CLIName: "package", JSONName: "package", Type: ParamString, Description: "Module path to fetch into the build (e.g. github.com/google/uuid@latest); mutates go.mod, not source files", Required: true},
}

func parseAddBuildDependency(raw map[string]any) (AddBuildDependencyReq, error) {
	var req AddBuildDependencyReq
	if err := CheckParams(raw, addBuildDependencyParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.Package, err = ParseString(raw, "package", "package", true); err != nil {
		return req, err
	}
	return req, nil
}

func runAddBuildDependency(ctx context.Context, cc CallContext, req AddBuildDependencyReq) (BuildDependencyRes, error) {
	if err := golang.AddDependency(ctx, cc.WorkDir, req.Package); err != nil {
		return BuildDependencyRes{}, err
	}
	return BuildDependencyRes{Package: req.Package, GoMod: filepath.Join(cc.WorkDir, "go.mod")}, nil
}

func addBuildDependencyDef() Def[AddBuildDependencyReq, BuildDependencyRes] {
	return Def[AddBuildDependencyReq, BuildDependencyRes]{
		Key:     "add_build_dependency",
		Summary: "Add an external module to the build (runs 'go get <package>' and 'go mod tidy') without manual shell execution. Build scope (go.mod); contrast with organize_imports, which edits import statements within source files.",
		Params:  addBuildDependencyParams,
		Level:   LevelBuild,
		CLIName: "add-build-dependency",
		MCPName: "semantic_add_build_dependency",
		Parse:   parseAddBuildDependency,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, AddBuildDependencyReq) (BuildDependencyRes, error){
			backend.LanguageGo: runAddBuildDependency,
		},
		Format: func(res BuildDependencyRes) (string, error) {
			return fmt.Sprintf("Successfully added dependency %s and tidied go.mod.", res.Package), nil
		},
		ExampleRaw: map[string]any{"package": "github.com/google/uuid@latest"},
		Batchable:  true,
	}
}

// ReplaceBodyReq replaces one function or method body with bare statements.
type ReplaceBodyReq struct {
	Project             backend.ProjectContext
	File                string
	Symbol              string
	Body                string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r ReplaceBodyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ReplaceBodyReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var replaceBodyParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "relative path to the Go source file", Required: true},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "function or method name, e.g. 'Foo' or '(*T).Foo'", Required: true},
	{Name: "body", CLIName: "body", JSONName: "body", Type: ParamString, Description: "replacement body as bare Go statements, no braces", Required: true},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "run goimports after replacement (default false)"},
}

func parseReplaceBody(raw map[string]any) (ReplaceBodyReq, error) {
	var req ReplaceBodyReq
	if err := CheckParams(raw, replaceBodyParams); err != nil {
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
	if req.Body, err = ParseString(raw, "body", "body", true); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, "auto_organize_imports", "auto-organize-imports", false); err != nil {
		return req, err
	}
	return req, nil
}

func runReplaceBody(ctx context.Context, cc CallContext, req ReplaceBodyReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	_, finishDelta := surroundingDelta(ctx, cc.WorkDir)
	diff, err := astedit.ReplaceBody(ctx, targetPath, req.Symbol, req.Body, astedit.BodyOptions{
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.Symbol, Diff: diff, Detail: "Successfully replaced body of " + req.Symbol + " in %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func replaceBodyDef() Def[ReplaceBodyReq, FileEditRes] {
	return Def[ReplaceBodyReq, FileEditRes]{
		Key:     "replace_body",
		Summary: "Replace the body of an existing Go function or method by name. The new body is provided as bare statements (no surrounding braces). Validates and formats in memory before writing; leaves the file untouched on any syntax error.",
		Params:  replaceBodyParams,
		Level:   LevelFile,
		CLIName: "replace-body",
		MCPName: "semantic_replace_body",
		Parse:   parseReplaceBody,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceBodyReq) (FileEditRes, error){
			backend.LanguageGo: runReplaceBody,
		},
		Format: func(res FileEditRes) (string, error) {
			text := fmt.Sprintf("Successfully replaced body of %s in %s.\n%s", res.Symbol, res.Display, res.Diff)
			return AppendDiagnosticDelta(text, res.Delta), nil
		},
		ExampleRaw: map[string]any{"file": "api/server.go", "symbol": "Server.Start", "body": "return nil"},
		Batchable:  true,
	}
}

// ScaffoldFileReq creates one new Go source file with its package clause.
type ScaffoldFileReq struct {
	Project   backend.ProjectContext
	File      string
	Package   string
	Overwrite bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r ScaffoldFileReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ScaffoldFileReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// ScaffoldFileRes reports a created file for response rendering and batch post-processing.
type ScaffoldFileRes struct {
	File    string
	Display string
	Package string
}

// WrittenFile returns the created path for batch post-processing.
func (r ScaffoldFileRes) WrittenFile() string { return r.File }

var scaffoldFileParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "relative path for the new file", Required: true},
	{Name: "package", CLIName: "package", JSONName: "package", Type: ParamString, Description: "package name or 'infer' (default 'infer')"},
	{Name: "overwrite", CLIName: "overwrite", JSONName: "overwrite", Type: ParamBoolean, Description: "replace existing file (default false)"},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "no-op for new files; present for schema uniformity"},
}

func parseScaffoldFile(raw map[string]any) (ScaffoldFileReq, error) {
	var req ScaffoldFileReq
	if err := CheckParams(raw, scaffoldFileParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Package, err = ParseStringDefault(raw, "package", "package", "infer"); err != nil {
		return req, err
	}
	if req.Overwrite, err = ParseBool(raw, "overwrite", "overwrite", false); err != nil {
		return req, err
	}
	return req, nil
}

func runScaffoldFile(ctx context.Context, cc CallContext, req ScaffoldFileReq) (ScaffoldFileRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	pkgName, err := astedit.ScaffoldFile(ctx, targetPath, req.Package, astedit.ScaffoldOptions{
		Overwrite:           req.Overwrite,
		AutoOrganizeImports: effectiveAutoOrganize(cc, false),
	})
	if err != nil {
		return ScaffoldFileRes{}, err
	}
	return ScaffoldFileRes{File: targetPath, Display: req.File, Package: pkgName}, nil
}

func scaffoldFileDef() Def[ScaffoldFileReq, ScaffoldFileRes] {
	return Def[ScaffoldFileReq, ScaffoldFileRes]{
		Key:     "scaffold_file",
		Summary: "Create a new Go source file with the correct package declaration. Use 'infer' (default) for package to auto-detect from sibling non-test files. Fails if the file already exists unless overwrite is true. Does not seed declarations — use insert tools afterward.",
		Params:  scaffoldFileParams,
		Level:   LevelFile,
		CLIName: "scaffold-file",
		MCPName: "semantic_scaffold_file",
		Parse:   parseScaffoldFile,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ScaffoldFileReq) (ScaffoldFileRes, error){
			backend.LanguageGo: runScaffoldFile,
		},
		Format: func(res ScaffoldFileRes) (string, error) {
			return fmt.Sprintf("Successfully scaffolded %s with package %s.", res.Display, res.Package), nil
		},
		ExampleRaw: map[string]any{"file": "api/server.go"},
		Batchable:  true,
	}
}

// InsertCaseReq inserts one case clause into a Go switch statement.
type InsertCaseReq struct {
	Project             backend.ProjectContext
	File                string
	Func                string
	SwitchOn            string
	Case                string
	Placement           string
	Anchor              string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r InsertCaseReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *InsertCaseReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var insertCaseParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "relative path to the Go source file", Required: true},
	{Name: "func", CLIName: "func", JSONName: "func", Type: ParamString, Description: "name of the function containing the switch", Required: true},
	{Name: "switch_on", CLIName: "switch-on", JSONName: "switch_on", Type: ParamString, Description: "the switch discriminant expression, e.g. 'method'; omit for tagless switch"},
	{Name: "case", CLIName: "case", JSONName: "case", Type: ParamString, Description: "full case clause source, e.g. 'case \"foo\":\\n\\treturn bar'", Required: true},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "one of: first, last, before_default, before, after (default 'before_default')", Enums: caseEnum},
	{Name: "anchor", CLIName: "anchor", JSONName: "anchor", Type: ParamString, Description: "case value to insert before/after when placement is 'before' or 'after'"},
	{Name: "auto_organize_imports", CLIName: "auto-organize-imports", JSONName: "auto_organize_imports", Type: ParamBoolean, Description: "run goimports after insertion (default false)"},
}

func parseInsertCase(raw map[string]any) (InsertCaseReq, error) {
	var req InsertCaseReq
	if err := CheckParams(raw, insertCaseParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Func, err = ParseString(raw, "func", "func", true); err != nil {
		return req, err
	}
	if req.SwitchOn, err = ParseString(raw, "switch_on", "switch-on", false); err != nil {
		return req, err
	}
	if req.Case, err = ParseString(raw, "case", "case", true); err != nil {
		return req, err
	}
	if req.Placement, err = ParseStringDefault(raw, "placement", "placement", "before_default"); err != nil {
		return req, err
	}
	if req.Anchor, err = ParseString(raw, "anchor", "anchor", false); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, "auto_organize_imports", "auto-organize-imports", false); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertCase(ctx context.Context, cc CallContext, req InsertCaseReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	diff, err := astedit.InsertCase(ctx, targetPath, req.Func, req.SwitchOn, req.Case, astedit.CaseOptions{
		Placement:           astedit.CasePlacement(req.Placement),
		AnchorCase:          req.Anchor,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.Func, Diff: diff, Detail: "Successfully inserted case into " + req.Func + " in %s."}, nil
}

func insertCaseDef() Def[InsertCaseReq, FileEditRes] {
	return Def[InsertCaseReq, FileEditRes]{
		Key:     "insert_case",
		Summary: "Insert a new case clause into an existing Go switch statement. Locates the switch by its containing function name and optional discriminant expression (omit switch_on to match a tagless switch). Validates the case source in memory before writing.",
		Params:  insertCaseParams,
		Level:   LevelFile,
		CLIName: "insert-case",
		MCPName: "semantic_insert_case",
		Parse:   parseInsertCase,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertCaseReq) (FileEditRes, error){
			backend.LanguageGo: runInsertCase,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": "api/server.go", "func": "Serve", "case": "case \"stop\":\n\treturn nil"},
		Batchable:  true,
	}
}

// registerEngineOps adds the host-engine file and build operations to the registry.
func registerEngineOps(registry *Registry) error {
	if err := Register(registry, assertionModeDef()); err != nil {
		return err
	}
	if err := Register(registry, insertDeclarationDef()); err != nil {
		return err
	}
	if err := Register(registry, insertFunctionDef()); err != nil {
		return err
	}
	if err := Register(registry, insertTypeDef()); err != nil {
		return err
	}
	if err := Register(registry, insertDeclDef()); err != nil {
		return err
	}
	if err := Register(registry, organizeImportsDef()); err != nil {
		return err
	}
	if err := Register(registry, addBuildDependencyDef()); err != nil {
		return err
	}
	if err := Register(registry, replaceBodyDef()); err != nil {
		return err
	}
	if err := Register(registry, scaffoldFileDef()); err != nil {
		return err
	}
	if err := Register(registry, insertCaseDef()); err != nil {
		return err
	}
	return nil
}
