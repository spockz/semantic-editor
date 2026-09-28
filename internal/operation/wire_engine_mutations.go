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

// ReplaceConstructReq replaces one Go control-flow construct within a function or method.
type ReplaceConstructReq struct {
	Project             backend.ProjectContext
	File                string
	Function            string
	Kind                astedit.ConstructKind
	Discriminator       string
	ConstructPath       string
	Source              string
	AutoOrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r ReplaceConstructReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ReplaceConstructReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// GetProjectContext returns the request project for registry dispatch.
func (r ReplaceBodyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ReplaceBodyReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var replaceConstructParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile, sourceReplaceConstructFile}},
	{Name: wireInFunction, CLIName: "in-function", JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified name of the containing function (e.g. 'Server.ServeHTTP', '(*Client).Do'). Identifies the scope to search; not the construct being replaced.", Required: true, SourceFields: []string{sourceReplaceConstructInFunction, sourceInsertCaseInFunction}},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Construct kind supported by the selected language backend", Required: true, DynamicEnums: executableConstructKinds},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: optional condition or short initializer for if constructs, or expression, case label, or defer call identifying the construct."},
	{Name: wireConstructPath, CLIName: "construct-path", JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic (e.g. '0' or '0.1'). The error diagnostic carries this path as text."},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Complete replacement Go construct source", Required: true},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically clean up and resolve imports after mutation (default true)", Default: true},
}

func executableConstructKinds(candidate backend.Backend) []string {
	provider, ok := candidate.(backend.ConstructCapabilityProvider)
	if !ok {
		return nil
	}
	kinds := provider.SupportedConstructs()
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}
	return names
}

var replaceBodyParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile, sourceReplaceConstructFile}},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: qualified function or method name (e.g. 'Foo' or '(*T).Foo')", Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, sourceReplaceBodySymbol}},
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
	Project       backend.ProjectContext
	File          string
	Package       string
	PurposeHeader string
	Overwrite     bool
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target: relative path for the new file", Required: true},
	{Name: "package", CLIName: "package", JSONName: "package", Type: ParamString, Description: "package name or 'infer' (default 'infer')"},
	{Name: "purpose_header", CLIName: "purpose-header", JSONName: "purpose_header", Type: ParamString, Description: "optional Go comment header explaining why the file exists"},
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
	if req.PurposeHeader, err = ParseStringDefault(raw, "purpose_header", "purpose-header", ""); err != nil {
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
		PurposeHeader:       req.PurposeHeader,
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: wireTargetGoFileDescription, Required: true, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile, sourceReplaceConstructFile}},
	{Name: wireInFunction, CLIName: "in-function", JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified name of the containing function (e.g. 'Server.ServeHTTP', '(*Client).Do'). Identifies the scope to search; not the construct being replaced.", Required: true, SourceFields: []string{sourceReplaceConstructInFunction, sourceInsertCaseInFunction}},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: switch discriminant expression or switch condition (e.g. 'method'). Omit only for a tagless switch.", SourceFields: []string{sourceInsertCaseDiscriminator}},
	{Name: wireConstructPath, CLIName: "construct-path", JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic (e.g. '0' or '0.1'). The error diagnostic carries this path as text."},
	{Name: "case", CLIName: "case", JSONName: "case", Type: ParamString, Description: "Complete case clause, including colon and statements, e.g. case \"foo\":\\n\\treturn bar", Required: true},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "one of: first, last, before_default, before, after (default 'before_default')", Enums: caseEnum},
	{Name: wireTargetCase, CLIName: "target-case", JSONName: wireTargetCase, Type: ParamString, Description: "Selector: case value to insert before or after when placement is 'before' or 'after' (e.g. 'nil', '\"foo\"'). Not a symbol address."},
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
	if req.Func, err = ParseString(raw, wireInFunction, "in-function", true); err != nil {
		return req, err
	}
	if req.SwitchOn, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.SwitchPath, err = ParseString(raw, wireConstructPath, "construct-path", false); err != nil {
		return req, err
	}
	if req.Case, err = ParseString(raw, "case", "case", true); err != nil {
		return req, err
	}
	if req.Placement, err = ParseStringDefault(raw, "placement", "placement", "before_default"); err != nil {
		return req, err
	}
	if req.Anchor, err = ParseString(raw, wireTargetCase, "target-case", false); err != nil {
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
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.Func, InFunction: req.Func, Discriminator: req.SwitchOn, Diff: diff, Detail: "Successfully inserted case into " + req.Func + " in %s."}, nil
}

func insertCaseDef() Def[InsertCaseReq, FileEditRes] {
	return Def[InsertCaseReq, FileEditRes]{
		Key:     "insert_case",
		Summary: "Use this tool instead of text editing when adding one case clause to an existing Go switch. Supply the existing in_function selector and, for tagged switches, the exact discriminator expression (for example request.Method); kind=case alone does not identify the target switch. If multiple switches match, copy construct_path from the ambiguity diagnostic. The case value is a complete clause including its colon and statements.",
		Params:  insertCaseParams,
		Level:   LevelFile,
		CLIName: "insert-case",
		MCPName: "semantic_insert_case",
		Parse:   parseInsertCase,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertCaseReq) (FileEditRes, error){
			backend.LanguageGo: runInsertCase,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, wireInFunction: "Serve", wireDiscriminator: "request.Method", "case": "case \"stop\":\n\treturn nil"},
		Batchable:  true,
	}
}

func parseReplaceConstruct(raw map[string]any) (ReplaceConstructReq, error) {
	var req ReplaceConstructReq
	if err := CheckParams(raw, replaceConstructParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Function, err = ParseString(raw, wireInFunction, "in-function", true); err != nil {
		return req, err
	}
	kind, err := ParseString(raw, "kind", "kind", true)
	if err != nil {
		return req, err
	}
	req.Kind = astedit.ConstructKind(kind)
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.ConstructPath, err = ParseString(raw, wireConstructPath, "construct-path", false); err != nil {
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

func runReplaceConstruct(ctx context.Context, cc CallContext, req ReplaceConstructReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ReplaceConstruct(ctx, targetPath, req.Function, req.Kind, req.Discriminator, req.Source, astedit.ConstructOptions{
		ConstructPath:       req.ConstructPath,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	return FileEditRes{File: targetPath, Display: req.File, Symbol: req.Function, InFunction: req.Function, Diff: diff, Detail: "Successfully replaced " + string(req.Kind) + " construct in " + req.Function + " in %s.", Delta: finishDelta(), HasDelta: true}, nil
}

func replaceConstructDef() Def[ReplaceConstructReq, FileEditRes] {
	return Def[ReplaceConstructReq, FileEditRes]{
		Key:     "replace_construct",
		Summary: "Use this tool instead of replace_body or text editing when replacing one existing Go loop, conditional, else block, switch or select branch, or defer statement. Select with kind and an optional discriminator; for if statements the discriminator may be the condition or short initializer. Errors list candidate paths and selectors; use construct_path to choose one. The in_function selector accepts a function name or receiver-qualified method (for example, (*Server).Serve).",
		Params:  replaceConstructParams,
		Level:   LevelFile,
		CLIName: "replace-construct",
		MCPName: "semantic_replace_construct",
		Parse:   parseReplaceConstruct,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceConstructReq) (FileEditRes, error){
			backend.LanguageGo: runReplaceConstruct,
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, "in_function": "TestWriteAssets", "kind": "loop", "discriminator": "want", "source": "for _, want := range []string{\"a\", \"b\"} { t.Run(want, func(t *testing.T) {}) }"},
		Batchable:  true,
	}
}
