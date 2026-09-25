// Package operation groups body replacement, file scaffolding, and switch case handlers.
package operation

import (
	"context"
	"fmt"
	"semedit/internal/astedit"
	"semedit/internal/backend"
)

// ReplaceBodyReq replaces one function or method body with bare statements.
type ReplaceBodyReq struct {
	Project             backend.ProjectContext
	File                string
	Symbol              string
	Body                string
	AutoOrganizeImports bool
}

// ReplaceLoopReq replaces one for or range loop within a function or method.
type ReplaceLoopReq struct {
	Project             backend.ProjectContext
	File                string
	Function            string
	LoopOn              string
	LoopPath            string
	Source              string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r ReplaceLoopReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ReplaceLoopReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// GetProjectContext returns the request project for registry dispatch.
func (r ReplaceBodyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ReplaceBodyReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var replaceLoopParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Path to the Go source file", Required: true},
	{Name: "function", CLIName: "function", JSONName: "function", Type: ParamString, Description: "Name of the containing function or method (e.g. 'TestWriteAssets' or '(*Server).Serve')", Required: true},
	{Name: "loop_on", CLIName: "loop-on", JSONName: "loop_on", Type: ParamString, Description: "Optional expression or variable that identifies the loop (e.g. 'want', 'items', 'i := 0')"},
	{Name: "loop_path", CLIName: "loop-path", JSONName: "loop_path", Type: ParamString, Description: "Candidate path from an ambiguity diagnostic, such as 0 or 0.1"},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Replacement Go loop code, e.g. 'for _, want := range [...] { ... }'", Required: true},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically clean up and resolve imports after mutation (default true)", Default: true},
}

var replaceBodyParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "relative path to the Go source file", Required: true},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "function or method name, e.g. 'Foo' or '(*T).Foo'", Required: true},
	{Name: "body", CLIName: "body", JSONName: "body", Type: ParamString, Description: "replacement body as bare Go statements, no braces", Required: true},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "run goimports after replacement (default false)"},
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
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, false); err != nil {
		return req, err
	}
	return req, nil
}

func runReplaceBody(ctx context.Context, cc CallContext, req ReplaceBodyReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
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
		Summary: "Use this tool instead of text editing or replacing the entire declaration when changing the body of an existing Go function or method by name. The new body is provided as bare statements (no surrounding braces). Validates and formats in memory before writing; leaves the file untouched on any syntax error." + automaticVerificationGuidance,
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
		ExampleRaw: map[string]any{"file": wireExampleFile, "symbol": "Server.Start", "body": "return nil"},
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
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "no-op for new files; present for schema uniformity"},
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
		Summary: "Use this tool instead of a built-in file writer when creating a Go source file with the correct package declaration. Use 'infer' (default) for package to auto-detect from sibling non-test files. Fails if the file already exists unless overwrite is true. Does not seed declarations — use insert tools afterward.",
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
		ExampleRaw: map[string]any{"file": wireExampleFile},
		Batchable:  true,
	}
}

// InsertCaseReq inserts one case clause into a Go switch statement.
type InsertCaseReq struct {
	Project             backend.ProjectContext
	File                string
	Func                string
	SwitchOn            string
	SwitchPath          string
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
	{Name: "switch_path", CLIName: "switch-path", JSONName: "switch_path", Type: ParamString, Description: "matching-switch path such as 0 or 0.1; use a path reported by an ambiguity error"},
	{Name: "case", CLIName: "case", JSONName: "case", Type: ParamString, Description: "full case clause source, e.g. 'case \"foo\":\\n\\treturn bar'", Required: true},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "one of: first, last, before_default, before, after (default 'before_default')", Enums: caseEnum},
	{Name: "anchor", CLIName: "anchor", JSONName: "anchor", Type: ParamString, Description: "case value to insert before/after when placement is 'before' or 'after'"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "run goimports after insertion (default false)"},
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
	if req.SwitchPath, err = ParseString(raw, "switch_path", "switch-path", false); err != nil {
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
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, false); err != nil {
		return req, err
	}
	return req, nil
}

func runInsertCase(ctx context.Context, cc CallContext, req InsertCaseReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	diff, err := astedit.InsertCase(ctx, targetPath, req.Func, req.SwitchOn, req.Case, astedit.CaseOptions{
		Placement:           astedit.CasePlacement(req.Placement),
		AnchorCase:          req.Anchor,
		SwitchPath:          req.SwitchPath,
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
		Summary: "Use this tool instead of text insertion or replacing the enclosing function when adding a case clause to an existing Go switch statement. Locates the switch by its containing function name and optional discriminant expression (omit switch_on to match a tagless switch). If multiple switches match, use switch_path from the ambiguity diagnostic. Validates the case source in memory before writing.",
		Params:  insertCaseParams,
		Level:   LevelFile,
		CLIName: "insert-case",
		MCPName: "semantic_insert_case",
		Parse:   parseInsertCase,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertCaseReq) (FileEditRes, error){
			backend.LanguageGo: runInsertCase,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "func": "Serve", "case": "case \"stop\":\n\treturn nil"},
		Batchable:  true,
	}
}

func parseReplaceLoop(raw map[string]any) (ReplaceLoopReq, error) {
	var req ReplaceLoopReq
	if err := CheckParams(raw, replaceLoopParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Function, err = ParseString(raw, "function", "function", true); err != nil {
		return req, err
	}
	if req.LoopOn, err = ParseString(raw, "loop_on", "loop-on", false); err != nil {
		return req, err
	}
	if req.LoopPath, err = ParseString(raw, "loop_path", "loop-path", false); err != nil {
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

func runReplaceLoop(ctx context.Context, cc CallContext, req ReplaceLoopReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ReplaceLoop(ctx, targetPath, req.Function, req.LoopOn, req.Source, astedit.LoopOptions{
		LoopPath:            req.LoopPath,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.Function, Diff: diff, Detail: "Successfully replaced loop in " + req.Function + " in %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func replaceLoopDef() Def[ReplaceLoopReq, FileEditRes] {
	return Def[ReplaceLoopReq, FileEditRes]{
		Key:     "replace_loop",
		Summary: "Use this tool instead of replace_body or text editing when replacing one existing Go for or range loop inside a function or method. Supply the complete replacement loop in source; select by loop_on and, for ambiguity, loop_path. It changes the selected loop without regenerating the rest of the function body.",
		Params:  replaceLoopParams,
		Level:   LevelFile,
		CLIName: "replace-loop",
		MCPName: "semantic_replace_loop",
		Parse:   parseReplaceLoop,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceLoopReq) (FileEditRes, error){
			backend.LanguageGo: runReplaceLoop,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "function": "TestWriteAssets", "loop_on": "want", "source": "for _, want := range []string{\"a\", \"b\"} { t.Run(want, func(t *testing.T) {}) }"},
		Batchable:  true,
	}
}
