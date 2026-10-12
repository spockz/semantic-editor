// Package operation groups body replacement, file scaffolding, and switch case handlers.
package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/sourcebuf"
)

// ReplaceBodyReq replaces one function or method body with bare statements.
type ReplaceBodyReq struct {
	Project             backend.ProjectContext
	File                string
	Symbol              string
	ExpectedRevision    *string
	Body                string
	AutoOrganizeImports bool
}

// ReplaceConstructReq replaces one Go control-flow construct within a function or method.
type ReplaceConstructReq struct {
	Project                   backend.ProjectContext
	File                      string
	Symbol                    string
	InFunction                string
	Kind                      backend.ConstructKind
	Discriminator             string
	ConstructPath             string
	ExpectedSelectionRevision string
	Source                    string
	AutoOrganizeImports       bool
}

// MoveConstructReq relocates a declaration without requiring source text.
type MoveConstructReq struct {
	Project             backend.ProjectContext
	File                string
	DestinationFile     string
	Symbol              string
	Placement           astedit.Placement
	TargetSymbol        string
	AutoOrganizeImports bool
}

// GetProjectContext returns the project used for registry dispatch.
func (r MoveConstructReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the project during dispatch merge.
func (r *MoveConstructReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var moveConstructParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: "destination_file", CLIName: "destination-file", JSONName: "destination_file", Type: ParamString, Role: RoleTarget, Description: "Target: destination Go source file. Cross-package moves support exported free functions within one Go module.", Required: false},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Description: "Target: qualified declaration symbol to relocate", Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, semanticMoveConstructToolName + ".symbol"}},
	{Name: "placement", CLIName: "placement", JSONName: "placement", Type: ParamString, Description: "Destination placement: file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol", Enums: placementEnum},
	{Name: wireTargetSymbol, CLIName: "target-symbol", JSONName: wireTargetSymbol, Type: ParamString, Description: "Target symbol required for before_symbol or after_symbol placement; omit it for every other placement"},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Organize imports after relocation (default true)", Default: true},
}

func validateMoveTargetPlacement(placement astedit.Placement, targetSymbol string) error {
	if targetSymbol == "" || placement == astedit.PlacementBeforeSymbol || placement == astedit.PlacementAfterSymbol {
		return nil
	}
	return fmt.Errorf("target_symbol requires before_symbol or after_symbol placement: %w", ErrInvalidParams)
}

func parseMoveConstruct(raw map[string]any) (MoveConstructReq, error) {
	var req MoveConstructReq
	if err := CheckParams(raw, moveConstructParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.DestinationFile, err = ParseString(raw, "destination_file", "destination-file", false); err != nil {
		return req, err
	}
	if req.DestinationFile == "" {
		req.DestinationFile = req.File
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	placement, err := ParseString(raw, "placement", "placement", false)
	if err != nil {
		return req, err
	}
	req.Placement = astedit.Placement(placement)
	if req.TargetSymbol, err = ParseString(raw, wireTargetSymbol, "target-symbol", false); err != nil {
		return req, err
	}
	err = validateMoveTargetPlacement(req.Placement, req.TargetSymbol)
	if err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	return req, nil
}

func validateBatchMoveTargets(ctx context.Context, workDir, sourcePath, destinationPath string) error {
	rootPath, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolve batch workspace root: %w", err)
	}
	root, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return fmt.Errorf("resolve batch workspace root symlinks: %w", err)
	}
	for _, target := range []string{sourcePath, destinationPath} {
		if targetErr := validateBatchMovePath(ctx, rootPath, root, target); targetErr != nil {
			return targetErr
		}
	}
	return nil
}

func validateBatchMovePath(ctx context.Context, rootPath, root, target string) error {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve batch move path %q: %w", target, err)
	}
	lexicalRelative, err := filepath.Rel(rootPath, absolute)
	if err != nil || lexicalRelative == "." || lexicalRelative == ".." || strings.HasPrefix(lexicalRelative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("batch move path %q must remain inside the workspace", target)
	}
	resolved, err := sourcebuf.CanonicalPath(ctx, absolute)
	if err != nil {
		return fmt.Errorf("resolve batch move path %q: %w", target, err)
	}
	if symlinkErr := rejectBatchMoveSymlinkComponents(rootPath, lexicalRelative, target); symlinkErr != nil {
		return symlinkErr
	}
	if relativeErr := validateBatchMoveRelative(root, resolved, target); relativeErr != nil {
		return relativeErr
	}
	_, err = sourcebuf.ReadFile(ctx, resolved)
	if err != nil {
		return fmt.Errorf("inspect batch move file %q: %w", target, err)
	}
	if filepath.Ext(resolved) != ".go" {
		return fmt.Errorf("batch move path %q must be a regular Go source file", target)
	}
	return nil
}

func rejectBatchMoveSymlinkComponents(rootPath, relative, target string) error {
	current := rootPath
	for component := range strings.SplitSeq(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("inspect batch move path %q: %w", target, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("batch move path %q cannot contain symlink components", target)
		}
	}
	return nil
}

func validateBatchMoveRelative(root, resolved, target string) error {
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("batch move path %q must resolve inside the workspace", target)
	}
	for component := range strings.SplitSeq(relative, string(filepath.Separator)) {
		switch component {
		case ".git", ".scratch", "vendor", "node_modules":
			return fmt.Errorf("batch move path %q is outside the captured workspace files", target)
		}
	}
	return nil
}

func runMoveConstruct(ctx context.Context, cc CallContext, req MoveConstructReq) (FileEditRes, error) {
	sourcePath := resolveWorkPath(cc.WorkDir, req.File)
	destinationPath := resolveWorkPath(cc.WorkDir, req.DestinationFile)
	if cc.InBatch {
		if err := validateBatchMoveTargets(ctx, cc.WorkDir, sourcePath, destinationPath); err != nil {
			return FileEditRes{}, err
		}

	}
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.MoveConstruct(ctx, sourcePath, req.Symbol, astedit.MoveConstructOptions{Placement: req.Placement, TargetSymbol: req.TargetSymbol, DestinationFile: destinationPath, AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports), DeferFormatting: cc.DeferImports})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, status := finishDelta()
	return FileEditRes{File: sourcePath, Display: displayWorkPath(cc.WorkDir, req.File), DestinationFile: destinationPath, DestinationDisplay: displayWorkPath(cc.WorkDir, req.DestinationFile), Symbol: req.Symbol, Diff: diff, Detail: "Successfully moved construct.", Delta: delta, DiagnosticStatus: status, HasDelta: true}, nil
}

func moveConstructDef() Def[MoveConstructReq, FileEditRes] {
	return Def[MoveConstructReq, FileEditRes]{Key: "move_construct", Summary: "Use this tool instead of text replacement to relocate one supported declaration to an explicit destination and placement. Use semantic_insert_construct to add a declaration or semantic_replace_construct to update one in place.", Params: moveConstructParams, Level: LevelWorkspace, CLIName: "move-construct", MCPName: semanticMoveConstructToolName, Parse: parseMoveConstruct, Handlers: map[backend.LanguageID]func(context.Context, CallContext, MoveConstructReq) (FileEditRes, error){backend.LanguageGo: bufferedFileEditHandler("move_construct", func(req MoveConstructReq) string { return req.File }, runMoveConstruct)}, Format: formatFileEdit, ExampleRaw: map[string]any{"file": "api/server.go", "destination_file": "api/server.go", "symbol": "Run", "placement": "before_symbol", "target_symbol": "Init", wireAutoOrganizeImports: true}, BufferedBatch: true, Batchable: true,
		Output: fileEditOutputContract(map[string]ValueSchema{"file": {Kind: SchemaString}, "symbol": {Kind: SchemaString}, "origin_file": {Kind: SchemaString}, "diff": {Kind: SchemaString}}, []OutputContinuation{outputContinuation("file", RoleTarget, fileOutputConsumers), outputContinuation("symbol", RoleTarget, symbolOutputConsumers)}),
		FinalizeOutput: func(cc CallContext, _ MoveConstructReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: func(result FileEditRes) (any, error) {
			return projectFileEditOutput(result, "file", "symbol", "origin_file", "diff")
		},
	}
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Go source file containing the declaration or construct; direct symbol resolution is scoped to this file.", Required: true, SourceFields: append(append([]string(nil), sourceFileContinuations...), semanticInspectConstructToolName+".result.file")},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: short declaration name or Receiver.Method resolved directly in the given file; declaration replacement does not require an enclosing function", Role: RoleTarget, SourceFields: []string{sourceLookupSymbol, sourceReplaceBodySymbol, sourceInsertConstructSymbol, sourceReplaceDeclSymbol}},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: enclosing function or method for nested control-flow and statement/expression constructs", Role: RoleSelector, SourceFields: append(append(append([]string(nil), sourceInFunctionContinuations...), "semantic_inspect_symbol.result.continuations[].in_function"), semanticInspectConstructToolName+".result.in_function")},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Construct kind supported by the selected language backend", Required: true, Role: RoleSelector, DynamicEnums: executableReplaceConstructKinds, SourceFields: []string{semanticInspectConstructToolName + ".result.kind"}},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: existing condition, statement, expression, or construct source; required to identify nested targets", Role: RoleSelector, SourceFields: append(append([]string(nil), sourceDiscriminatorContinuations...), semanticInspectConstructToolName+".result.discriminator")},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Description: "Selector: exact construct_path from semantic_inspect_construct candidate list.", Role: RoleSelector, SourceFields: []string{semanticInspectConstructToolName + ".result." + wireCandidates + "[]." + wireConstructPath}},
	{Name: wireExpectedSelectionRevision, CLIName: wireCLIExpectedSelectionRevision, JSONName: wireExpectedSelectionRevision, Type: ParamString, Description: "Optional guard from semantic_inspect_construct for nested path-addressable constructs. A mismatch refuses before writing and returns the current choices and token; it is not a whole-file revision.", Role: RoleTarget, SourceFields: []string{semanticInspectConstructToolName + ".result." + wireExpectedSelectionRevision}},
	{Name: "source", CLIName: "source", JSONName: "source", Type: ParamString, Description: "Complete replacement construct source; expression kinds require an expression and statement/declaration kinds require the matching syntax", Required: true},
	{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically clean up and resolve imports after mutation (default true)", Default: true},
}

var replaceBodyParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target: Go source file containing the function or method; the short name or receiver-qualified name is resolved directly in this file.", Required: true, Role: RoleTarget, SourceFields: append(append(append([]string(nil), sourceReplaceBodyFileContinuations...), "semantic_inspect_symbol.result.continuations[].file", sourceGeneratedMethodFile), signatureTargetSourceFields("file")...)},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: short function name or Receiver.Method (e.g. 'Foo' or '(*T).Foo'), resolved directly in the given file; copy signature-tool targets from its finalized function_target.", Required: true, Role: RoleTarget, SourceFields: append([]string{sourceLookupSymbol, sourceRenameSymbol, sourceReplaceBodySymbol, sourceInsertFunctionSymbol, sourceExtractFunctionReceiptSymbol, sourceExtractVariableReceiptSymbol, sourceInlineVariableReceiptSymbol, "semantic_inspect_symbol.result.continuations[].symbol", sourceGeneratedMethodSymbol}, signatureTargetSourceFields("symbol")...)},
	{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Description: "Guard: full-file revision from inspection, an extraction receipt, a resolved rename continuation, or the same finalized function_target record as file and symbol", Role: RoleTarget, SourceFields: append([]string{"semantic_inspect_symbol.result.continuations[].expected_revision", sourceRenameExpectedRevision, sourceExtractFunctionReceiptRevision, sourceExtractVariableReceiptRevision, sourceInlineVariableReceiptRevision}, signatureTargetSourceFields(wireExpectedRevision)...)},
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
	if value, exists := raw[wireExpectedRevision]; exists {
		expected, ok := value.(string)
		if !ok {
			return req, fmt.Errorf("expected_revision must be a string")
		}
		if expected == "" {
			return req, fmt.Errorf("expected_revision must not be empty")
		}
		req.ExpectedRevision = &expected
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
	if req.ExpectedRevision != nil {
		if err := validateExpectedRevision(ctx, cc, req.File, req.Symbol, *req.ExpectedRevision); err != nil {
			return FileEditRes{}, err
		}
	}
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ReplaceBody(ctx, targetPath, req.Symbol, req.Body, astedit.BodyOptions{
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol, Diff: diff, Detail: "Replaced body.", Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}, nil
}

func replaceBodyDef() Def[ReplaceBodyReq, FileEditRes] {
	return Def[ReplaceBodyReq, FileEditRes]{
		Key: "replace_body", VariantVerb: "replace", VariantKind: "function",
		Summary: "Use this tool instead of text editing or replacing the entire declaration when changing the body of an existing Go function or method by name. The new body is provided as bare statements (no surrounding braces). Use semantic_replace_construct for replacing a declaration or selecting a statement/expression slot. Validates and formats in memory before writing; leaves the file untouched on any syntax error." + automaticVerificationGuidance,
		Params:  replaceBodyParams,
		Level:   LevelFile,
		CLIName: "replace-body",
		MCPName: replaceBodyTool,
		Parse:   parseReplaceBody,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceBodyReq) (FileEditRes, error){
			backend.LanguageGo: bufferedFileEditHandler("replace_body", func(req ReplaceBodyReq) string { return req.File }, runReplaceBody),
		},
		Format: func(res FileEditRes) (string, error) {
			text := fmt.Sprintf("Successfully replaced body of %s in %s.", res.Symbol, res.Display)
			if res.HasDelta {
				text += " diagnostic_status: " + res.DiagnosticStatus + "."
				if res.DiagnosticStatus == "changed" {
					text = AppendDiagnosticDelta(text, res.Delta)
				}
			}
			if res.Diff != "" {
				text += "\n" + res.Diff
			}
			return text, nil
		},
		ExampleRaw: map[string]any{"file": wireExampleFile, "symbol": "Server.Start", "body": "return nil"},
		Output:     fileEditOutputContract(map[string]ValueSchema{"file": {Kind: SchemaString}, "symbol": {Kind: SchemaString}, "diff": {Kind: SchemaString}}, []OutputContinuation{outputContinuation("file", RoleTarget, fileOutputConsumers), outputContinuation("symbol", RoleTarget, symbolOutputConsumers)}),
		FinalizeOutput: func(cc CallContext, _ ReplaceBodyReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: func(result FileEditRes) (any, error) { return projectFileEditOutput(result, "file", "symbol", "diff") },
		BufferedBatch: true, Batchable: true,
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
	File                  string
	Display               string
	Package               string
	Receipt               EditReceipt
	SuppressContinuations bool
}

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
	return ScaffoldFileRes{File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Package: pkgName}, nil
}

func scaffoldFileDef() Def[ScaffoldFileReq, ScaffoldFileRes] {
	return Def[ScaffoldFileReq, ScaffoldFileRes]{
		Key:           "scaffold_file",
		Summary:       "Use this tool instead of a built-in file writer when creating a Go source file with the correct package declaration. Use 'infer' (default) for package to auto-detect from sibling non-test files. Fails if the file already exists unless overwrite is true. Does not seed declarations — use insert tools afterward.",
		Params:        scaffoldFileParams,
		Level:         LevelFile,
		CLIName:       "scaffold-file",
		MCPName:       "semantic_scaffold_file",
		BufferedBatch: true,
		Parse:         parseScaffoldFile,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ScaffoldFileReq) (ScaffoldFileRes, error){
			backend.LanguageGo: runScaffoldFile,
		},
		Format: func(res ScaffoldFileRes) (string, error) {
			return fmt.Sprintf("Successfully scaffolded %s with package %s.", res.Display, res.Package), nil
		},
		ExampleRaw: map[string]any{"file": wireExampleFile},
		Output:     fileEditOutputContract(map[string]ValueSchema{"file": {Kind: SchemaString}, "package": {Kind: SchemaString}}, []OutputContinuation{outputContinuation("file", RoleTarget, fileOutputConsumers)}),
		FinalizeOutput: func(cc CallContext, _ ScaffoldFileReq, result ScaffoldFileRes) (ScaffoldFileRes, error) {
			content, err := sourcebuf.ReadFile(cc.Ctx, result.File)
			if err != nil {
				return ScaffoldFileRes{}, fmt.Errorf("read scaffolded file for receipt: %w", err)
			}
			path := filepath.ToSlash(result.Display)
			if path == "" {
				return ScaffoldFileRes{}, fmt.Errorf("scaffolded file path is empty: %w", ErrInvalidDefinition)
			}
			revision := structuralRevision(content)
			receipt, err := finalizeCallReceipt(cc.Ctx, cc, cc.WorkDir, EditReceiptDraft{Operation: "scaffold_file", Status: EditPersisted, PlannedChanges: []FileRevision{{Path: path, Revision: revision}}}, nil)
			if err != nil {
				return ScaffoldFileRes{}, err
			}
			result.Receipt = receipt
			result.SuppressContinuations = cc.InBatch
			return result, nil
		},
		ProjectOutput: func(result ScaffoldFileRes) (any, error) {
			projected := map[string]any{"result": "Successfully scaffolded " + result.Display + " with package " + result.Package + ".", editReceiptField: result.Receipt, "package": result.Package}
			if !result.SuppressContinuations {
				projected["file"] = result.Display
			}
			return projected, nil
		},
		Batchable: true,
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
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: wireTargetGoFileDescription, Required: true, SourceFields: sourceFileContinuations},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: qualified name of the containing function (e.g. 'Server.ServeHTTP', '(*Client).Do'). Identifies the scope to search; not the construct being replaced.", Role: RoleSelector, Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: switch discriminant expression or switch condition (e.g. 'method'). Omit only for a tagless switch.", Role: RoleSelector, SourceFields: sourceDiscriminatorContinuations},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic (e.g. '0' or '0.1'). The error diagnostic carries this path as text."},
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
	if req.Func, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return req, err
	}
	if req.SwitchOn, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.SwitchPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
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
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.InsertCase(ctx, targetPath, req.Func, req.SwitchOn, req.Case, astedit.CaseOptions{
		Placement:           astedit.CasePlacement(req.Placement),
		AnchorCase:          req.Anchor,
		SwitchPath:          req.SwitchPath,
		AutoOrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), InFunction: req.Func, Discriminator: req.SwitchOn, Diff: diff, Detail: "Inserted case.", Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}, nil
}

func insertCaseDef() Def[InsertCaseReq, FileEditRes] {
	return Def[InsertCaseReq, FileEditRes]{
		Key: "insert_case", VariantVerb: "insert", VariantKind: "case",
		Summary: "Use this tool instead of text editing when adding one case clause to an existing Go switch. Use semantic_insert_construct for the generic insertion interface. Supply the existing in_function selector and, for tagged switches, the exact discriminator expression (for example request.Method); kind=case alone does not identify the target switch. If multiple switches match, copy construct_path from the ambiguity diagnostic. The case value is a complete clause including its colon and statements.",
		Params:  insertCaseParams,
		Level:   LevelFile,
		CLIName: "insert-case",
		MCPName: semanticInsertCaseToolName,
		Parse:   parseInsertCase,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InsertCaseReq) (FileEditRes, error){
			backend.LanguageGo: bufferedFileEditHandler("insert_case", func(req InsertCaseReq) string { return req.File }, runInsertCase),
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, wireInFunction: "Serve", wireDiscriminator: "request.Method", "case": "case \"stop\":\n\treturn nil"},
		Output:     fileEditOutputContract(map[string]ValueSchema{"file": {Kind: SchemaString}, wireInFunction: {Kind: SchemaString}, wireDiscriminator: {Kind: SchemaString}, "diff": {Kind: SchemaString}}, []OutputContinuation{outputContinuation("file", RoleTarget, fileOutputConsumers), outputContinuation(wireInFunction, RoleSelector, functionOutputConsumers), outputContinuation(wireDiscriminator, RoleSelector, discriminatorOutputConsumers)}),
		FinalizeOutput: func(cc CallContext, _ InsertCaseReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: func(result FileEditRes) (any, error) {
			return projectFileEditOutput(result, "file", wireInFunction, wireDiscriminator, "diff")
		},
		BufferedBatch: true, Batchable: true,
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
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", false); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, false); err != nil {
		return req, err
	}
	kind, err := ParseString(raw, "kind", "kind", true)
	if err != nil {
		return req, err
	}
	req.Kind = backend.ConstructKind(kind)
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return req, err
	}
	if req.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return req, err
	}
	if req.ExpectedSelectionRevision, err = parseNonEmptyOptionalString(raw, wireExpectedSelectionRevision, wireCLIExpectedSelectionRevision); err != nil {
		return req, err
	}
	if req.Source, err = ParseString(raw, "source", "source", true); err != nil {
		return req, err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return req, err
	}
	switch req.Kind {
	case backend.ConstructFunction, backend.ConstructMethod, backend.ConstructType, backend.ConstructConst, backend.ConstructVar:
		if req.Symbol == "" {
			return req, fmt.Errorf("symbol is required for kind=%s", req.Kind)
		}
		if req.InFunction != "" {
			return req, fmt.Errorf("in_function cannot be combined with declaration kind=%s", req.Kind)
		}
	default:
		if req.InFunction == "" {
			return req, fmt.Errorf("in_function is required for nested kind=%s", req.Kind)
		}
		if req.Symbol != "" {
			return req, fmt.Errorf("symbol cannot be combined with nested kind=%s", req.Kind)
		}
	}
	if req.ExpectedSelectionRevision != "" && !supportsInspectableConstructKind(req.Kind) {
		return req, fmt.Errorf("expected_selection_revision is supported only for nested candidate kinds: %w", ErrInvalidParams)
	}
	return req, nil
}

func runReplaceConstruct(ctx context.Context, cc CallContext, req ReplaceConstructReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ReplaceConstruct(ctx, targetPath, req.InFunction, req.Kind, req.Discriminator, req.Source, astedit.ConstructOptions{
		Symbol:                    req.Symbol,
		InFunction:                req.InFunction,
		Discriminator:             req.Discriminator,
		ConstructPath:             req.ConstructPath,
		ExpectedSelectionRevision: req.ExpectedSelectionRevision,
		AutoOrganizeImports:       effectiveAutoOrganize(cc, req.AutoOrganizeImports),
	})
	if err != nil {
		annotateConstructSelectionError(err, displayWorkPath(cc.WorkDir, req.File))
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol, InFunction: req.InFunction, Discriminator: req.Discriminator, Diff: diff, Detail: "Replaced " + string(req.Kind) + " construct.", Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}, nil
}

func replaceConstructDef() Def[ReplaceConstructReq, FileEditRes] {
	return Def[ReplaceConstructReq, FileEditRes]{
		Key:            "replace_construct",
		Summary:        "Use this tool instead of replacing a whole function body to update one selected construct by kind. For kind=statement/expression, the discriminator selects a source span and the tool chooses the whole statement or an inner expression slot, such as an assignment, call argument, literal, or condition, from the source shape. Use symbol for named declarations or in_function plus discriminator for nested targets. semantic_replace_body preserves a function signature when only its body changes.",
		Params:         replaceConstructParams,
		Level:          LevelFile,
		CLIName:        "replace-construct",
		MCPName:        semanticReplaceConstructToolName,
		Parse:          parseReplaceConstruct,
		NormalizeInput: NormalizeConstructKindInput,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReplaceConstructReq) (FileEditRes, error){
			backend.LanguageGo: bufferedFileEditHandler("replace_construct", func(req ReplaceConstructReq) string { return req.File }, runReplaceConstruct),
		},
		Format:     formatFileEdit,
		ExampleRaw: map[string]any{"file": wireExampleFile, wireInFunction: "TestWriteAssets", "kind": "loop", wireDiscriminator: "want", "source": "for _, want := range []string{\"a\", \"b\"} { t.Run(want, func(t *testing.T) {}) }"},
		Output: fileEditOutputContract(map[string]ValueSchema{"file": {Kind: SchemaString}, wireInFunction: {Kind: SchemaString}, "diff": {Kind: SchemaString}},
			[]OutputContinuation{outputContinuation("file", RoleTarget, fileOutputConsumers), outputContinuation(wireInFunction, RoleSelector, functionOutputConsumers)}),
		ErrorOutput: constructSelectionErrorOutput(),
		FinalizeOutput: func(cc CallContext, _ ReplaceConstructReq, result FileEditRes) (FileEditRes, error) {
			return finalizeFileEditOutput(cc, result)
		},
		ProjectOutput: func(result FileEditRes) (any, error) {
			return projectFileEditOutput(result, "file", wireInFunction, "diff")
		},
		ProjectErrorOutput: projectConstructSelectionError,
		BufferedBatch:      true, Batchable: true,
	}
}
