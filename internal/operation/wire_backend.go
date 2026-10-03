// Package operation wires backend lookup, rename, and verify leaves into registry operations.
package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/backends"
	"semedit/internal/capability"
	"semedit/internal/pipeline"
	"semedit/internal/projectverify"
	"semedit/internal/sourcebuf"
)

// languageEnums accepts every backend language so support is decided by handler presence, not parsing.

const wireCLIExpectedRevision = "expected-revision"

var renameParams = []ParameterContract{
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: qualified symbol identifier (e.g. Server.ServeHTTP, (*Client).Do, or ValidateToken)", Type: ParamString, Required: true, SourceFields: append(append(append(append([]string(nil), sourceSymbolContinuations...), sourceExtractFunctionReceiptSymbol, sourceExtractConstantReceiptSymbol, sourceExtractVariableReceiptSymbol, sourceInlineVariableReceiptSymbol), sourceContractExtractSymbol, sourceContractImplementSymbol), sourceConcreteTargetSymbol, sourceGeneratedMethodSymbol)},
	{Name: "to", CLIName: "to", JSONName: "to", Description: "New identifier name (e.g. Serve)", Type: ParamString, Required: true},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Selector: source file containing the declaration; paths are relative to the workspace root. Required to select Rust or Java rename scope", Type: ParamString, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceExtractFunctionReceiptFile, sourceExtractConstantReceiptFile, "semantic_extract_variable.result.continuations[].file", "semantic_inline_variable.result.continuations[].file", sourceContractExtractFile, sourceContractImplementFile, sourceConcreteTargetFile, sourceGeneratedMethodFile}},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto; Rust requires a selected .rs file and workspace trust; Java requires a selected .java file and trust_workspace=true)", Type: ParamString, Enums: languageEnums, LanguageSelector: true},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for this request; required for Rust or Java rename (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java rename", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
	{Name: wireAutoOrganizeImports, CLIName: "", JSONName: wireAutoOrganizeImports, Description: "Automatically clean up and organize imports after rename (default true)", Type: ParamBoolean, Default: true},
}

var lookupParams = readRequestParams(
	ParameterContract{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: qualified symbol identifier (e.g. Server.ServeHTTP, (*Client).Do, or ValidateToken)", Type: ParamString, Required: true, SourceFields: append(sourceSymbolContinuations, sourceContractExtractSymbol, sourceContractImplementSymbol)},
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Description: "Selector: optional source file that constrains symbol search scope", Type: ParamString, SourceFields: append(sourceFileContinuations, sourceContractExtractFile, sourceContractImplementFile)},
)

const (
	wireCLITrustWorkspace      = "trust-workspace"
	wireJDTLSHome              = "jdtls_home"
	wireImportMaven            = "import_maven"
	wireCLIAutoOrganizeImports = "auto-organize-imports"
	wireAccessModifier         = "access_modifier"
)

const (
	wireTrustWorkspace          = "trust_workspace"
	wireAutoOrganizeImports     = "auto_organize_imports"
	wireExampleFile             = "api/server.go"
	wireTargetSymbol            = "target_symbol"
	wireDiscriminator           = "discriminator"
	wireInDeclaration           = "in_declaration"
	wireInFunction              = "in_function"
	wireCLIInFunction           = "in-function"
	wireConstructPath           = "construct_path"
	wireCLIConstructPath        = "construct" + "-path"
	wireTargetCase              = "target_case"
	wireTargetGoFileDescription = "Target: Go file"

	sourceLookupSymbol                 = "semantic_lookup.symbol"
	sourceRenameSymbol                 = "semantic_rename.symbol"
	sourceRenameExpectedRevision       = "semantic_rename.expected_revision"
	sourceReplaceBodySymbol            = "semantic_replace_body.symbol"
	sourceLookupFile                   = "semantic_lookup.file"
	sourceRenameFile                   = "semantic_rename.file"
	sourceReplaceBodyFile              = "semantic_replace_body.file"
	sourceReplaceConstructFile         = "semantic_replace_construct.file"
	sourceInsertCaseFile               = "semantic_insert_case.file"
	sourceInsertFunctionFile           = "semantic_insert_function.file"
	sourceInsertTypeFile               = "semantic_insert_type.file"
	sourceInsertDeclFile               = "semantic_insert_decl.file"
	sourceInsertDeclarationFile        = "semantic_insert_declaration.file"
	sourceInsertConstructFile          = "semantic_insert_construct.file"
	sourceReplaceDeclFile              = "semantic_replace_decl.file"
	sourceOrganizeImportsFile          = "semantic_organize_imports.file"
	sourceScaffoldFile                 = "semantic_scaffold_file.file"
	sourceScaffoldPackage              = "semantic_scaffold_file.package"
	sourceReplaceConstructInFunction   = "semantic_replace_construct.in_function"
	sourceInsertCaseInFunction         = "semantic_insert_case.in_function"
	sourceInsertCaseDiscriminator      = "semantic_insert_case.discriminator"
	sourceInsertFunctionSymbol         = "semantic_insert_function.symbol"
	sourceInsertTypeSymbol             = "semantic_insert_type.symbol"
	sourceInsertDeclSymbol             = "semantic_insert_decl.symbol"
	sourceInsertDeclarationSymbol      = "semantic_insert_declaration.symbol"
	sourceInsertConstructSymbol        = "semantic_insert_construct.symbol"
	sourceReplaceDeclSymbol            = "semantic_replace_decl.symbol"
	sourceInsertConstructInFunction    = "semantic_insert_construct.in_function"
	sourceInsertConstructDiscriminator = "semantic_insert_construct.discriminator"
)

const wireExpectedRevision = "expected_revision"

const (
	wireApplicationState = "application_state"
	wireDiagnostics      = "diagnostics"
)

var sourceFileContinuations = []string{
	sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile,
	sourceReplaceConstructFile, sourceInsertCaseFile, sourceInsertFunctionFile,
	sourceInsertTypeFile, sourceInsertDeclFile, sourceInsertDeclarationFile,
	sourceInsertConstructFile, sourceReplaceDeclFile, sourceOrganizeImportsFile, sourceScaffoldFile,
	sourceExtractFunctionReceiptFile, sourceExtractConstantReceiptFile, sourceExtractVariableReceiptFile, sourceInlineVariableReceiptFile,
}

var sourceReplaceDeclFileContinuations = []string{sourceLookupFile, sourceRenameFile, sourceReplaceDeclFile, sourceInsertDeclFile, sourceExtractConstantReceiptFile}

var sourceReplaceBodyFileContinuations = []string{sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile, sourceInsertFunctionFile, sourceExtractFunctionReceiptFile}

var sourceSymbolContinuations = []string{sourceLookupSymbol, sourceRenameSymbol, sourceReplaceBodySymbol, sourceInsertFunctionSymbol, sourceInsertTypeSymbol, sourceInsertDeclSymbol, sourceInsertDeclarationSymbol, sourceInsertConstructSymbol, sourceReplaceDeclSymbol, "semantic_move_construct.symbol"}

var sourceInFunctionContinuations = []string{
	sourceReplaceConstructInFunction, sourceInsertCaseInFunction, sourceInsertConstructInFunction,
	sourceInspectInFunction,
}

const sourceInspectInFunction = "semantic_inspect_symbol.result.continuations[].in_function"
const sourceInspectExpectedRevision = "semantic_inspect_symbol.result.continuations[].expected_revision"

var sourceDiscriminatorContinuations = []string{
	sourceInsertCaseDiscriminator, sourceInsertConstructDiscriminator,
}

var languageEnums = []string{"auto", "go", "rust", "java", "scala", "haskell", "kotlin", "bash", "make"}

// LookupReq requests symbol coordinates through the selected language backend.
type LookupReq struct {
	Project backend.ProjectContext
	Symbol  string
}

// GetProjectContext returns the request project for registry dispatch.
func (r LookupReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project with the effective dispatch project.
func (r *LookupReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// RenameReq requests a semantic rename through the selected language backend.
type RenameReq struct {
	Project         backend.ProjectContext
	Symbol          string
	To              string
	OrganizeImports bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r RenameReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project with the effective dispatch project.
func (r *RenameReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// VerifyReq requests formatting and diagnostics through the selected language backend.
type VerifyReq struct {
	Project            backend.ProjectContext
	Path               string
	FormatSelectedFile bool
	OrganizeImports    bool
	CheckOnly          bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r VerifyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project with the effective dispatch project.
func (r *VerifyReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// LookupRes reports symbol coordinates in backend-neutral form.
type LookupRes = backend.LookupResult

// RenameRes reports the resolved target and diagnostic delta after a rename.
type RenameRes = backend.RenameResult

// VerifyRes reports compiler diagnostics and whether gofmt reformatted sources.
type VerifyRes struct {
	Diagnostics     []backend.Diagnostic `json:"diagnostics,omitempty"`
	Formatted       bool                 `json:"formatted,omitempty"`
	NormalizeReport projectverify.Report `json:"normalize"`
	CheckReport     projectverify.Report `json:"checks"`
	Scope           projectverify.Scope  `json:"scope"`
	Failure         string               `json:"failure,omitempty"`
	Coverage        []LanguageCoverage   `json:"coverage,omitempty"`
}

var verifyParams = []ParameterContract{
	{Name: "path", CLIName: "path", JSONName: "path", Description: "Optional Go file or directory to format and check; relative paths resolve from the active semedit workspace root. Go verification runs formatting before diagnostics and can write files", Type: ParamString, Default: "."},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Target: Java, Kotlin, or Bash source file to verify; path is relative to the workspace root", Type: ParamString, SourceFields: []string{sourceLookupFile, sourceRenameFile}},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto)", Type: ParamString, Enums: languageEnums, LanguageSelector: true},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for this request; required for external language verification (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java verification", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
	{Name: "format_selected_file", CLIName: "format-selected-file", JSONName: "format_selected_file", Description: "Apply JDT LS formatting to the selected Java file", Type: ParamBoolean, Default: false},
	{Name: "organize_imports", CLIName: "organize-imports", JSONName: "organize_imports", Description: "Apply bounded JDT LS source.organizeImports to the selected Java file", Type: ParamBoolean, Default: false},
	{Name: "kotlin_bin", CLIName: "kotlin-bin", JSONName: "kotlin_bin", Description: "Path to a preinstalled fwcd/kotlin-language-server executable", Type: ParamString},
	{Name: "bash_bin", CLIName: "bash-bin", JSONName: "bash_bin", Description: "Path to a preinstalled bash-language-server executable", Type: ParamString},
	{Name: "make_bin", CLIName: "make-bin", JSONName: "make_bin", Description: "Path to a preinstalled make-ls executable", Type: ParamString},
	{Name: "check_only", CLIName: "check-only", JSONName: "check_only", Description: "Skip all formatting and normalization hooks; verify the current files only", Type: ParamBoolean, Default: false},
}

func parseLookup(raw map[string]any) (LookupReq, error) {
	if err := CheckParams(raw, lookupParams); err != nil {
		return LookupReq{}, err
	}
	symbol, err := ParseString(raw, "symbol", "symbol", true)
	if err != nil {
		return LookupReq{}, err
	}
	file, err := ParseString(raw, "file", "file", false)
	if err != nil {
		return LookupReq{}, err
	}
	project, err := parseReadProject(raw, file)
	if err != nil {
		return LookupReq{}, err
	}
	return LookupReq{Project: project, Symbol: symbol}, nil
}

func parseRename(raw map[string]any) (RenameReq, error) {
	if err := CheckParams(raw, renameParams); err != nil {
		return RenameReq{}, err
	}
	symbol, err := ParseString(raw, "symbol", "symbol", true)
	if err != nil {
		return RenameReq{}, err
	}
	to, err := ParseString(raw, "to", "to", true)
	if err != nil {
		return RenameReq{}, err
	}
	file, err := ParseString(raw, "file", "file", false)
	if err != nil {
		return RenameReq{}, err
	}
	language, err := ParseEnum(raw, "language", "language", languageEnums, false, "")
	if err != nil {
		return RenameReq{}, err
	}
	trusted, err := ParseBool(raw, wireTrustWorkspace, wireCLITrustWorkspace, false)
	if err != nil {
		return RenameReq{}, err
	}
	jdtlsHome, err := ParseString(raw, wireJDTLSHome, "jdtls-home", false)
	if err != nil {
		return RenameReq{}, err
	}
	javaBin, err := ParseString(raw, "java_bin", "java-bin", false)
	if err != nil {
		return RenameReq{}, err
	}
	importMaven, err := ParseBool(raw, wireImportMaven, "import-maven", false)
	if err != nil {
		return RenameReq{}, err
	}
	organizeImports, err := ParseBool(raw, wireAutoOrganizeImports, "", true)
	if err != nil {
		return RenameReq{}, err
	}
	return RenameReq{
		Project: backend.ProjectContext{
			File:           file,
			Language:       backend.LanguageID(language),
			WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted},
			Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven},
		},
		Symbol:          backend.NormalizeRenameInput(symbol),
		To:              backend.NormalizeRenameInput(to),
		OrganizeImports: organizeImports,
	}, nil
}

func parseVerify(raw map[string]any) (VerifyReq, error) {
	if err := CheckParams(raw, verifyParams); err != nil {
		return VerifyReq{}, err
	}
	path, err := ParseStringDefault(raw, "path", "path", ".")
	if err != nil {
		return VerifyReq{}, err
	}
	file, err := ParseString(raw, "file", "file", false)
	if err != nil {
		return VerifyReq{}, err
	}
	if file != "" {
		path = file
	}
	language, err := ParseEnum(raw, "language", "language", languageEnums, false, "")
	if err != nil {
		return VerifyReq{}, err
	}
	trusted, err := ParseBool(raw, wireTrustWorkspace, wireCLITrustWorkspace, false)
	if err != nil {
		return VerifyReq{}, err
	}
	jdtlsHome, err := ParseString(raw, wireJDTLSHome, "jdtls-home", false)
	if err != nil {
		return VerifyReq{}, err
	}
	javaBin, err := ParseString(raw, "java_bin", "java-bin", false)
	if err != nil {
		return VerifyReq{}, err
	}
	importMaven, err := ParseBool(raw, wireImportMaven, "import-maven", false)
	if err != nil {
		return VerifyReq{}, err
	}
	formatSelected, err := ParseBool(raw, "format_selected_file", "format-selected-file", false)
	if err != nil {
		return VerifyReq{}, err
	}
	organizeImports, err := ParseBool(raw, "organize_imports", "organize-imports", false)
	if err != nil {
		return VerifyReq{}, err
	}
	kotlinBin, err := ParseString(raw, "kotlin_bin", "kotlin-bin", false)
	if err != nil {
		return VerifyReq{}, err
	}
	bashBin, err := ParseString(raw, "bash_bin", "bash-bin", false)
	if err != nil {
		return VerifyReq{}, err
	}
	makeBin, err := ParseString(raw, "make_bin", "make-bin", false)
	if err != nil {
		return VerifyReq{}, err
	}
	checkOnly, err := ParseBool(raw, "check_only", "check-only", false)
	if err != nil {
		return VerifyReq{}, err
	}
	return VerifyReq{
		Project: backend.ProjectContext{
			File:           path,
			Language:       backend.LanguageID(language),
			WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted},
			Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven},
			Kotlin:         backend.KotlinConfig{KotlinBin: kotlinBin},
			Bash:           backend.BashConfig{BashBin: bashBin},
			Make:           backend.MakeConfig{MakeBin: makeBin},
			KotlinBin:      kotlinBin,
		},
		Path:               path,
		FormatSelectedFile: formatSelected,
		OrganizeImports:    organizeImports, CheckOnly: checkOnly,
	}, nil
}

func lookupRun(ctx context.Context, cc CallContext, request LookupReq) (*backend.LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	project := effectiveProject(cc, request.Project)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	return service.Lookup(ctx, project, request.Symbol)
}

func renameRun(ctx context.Context, cc CallContext, request RenameReq) (*backend.RenameResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	project := effectiveProject(cc, request.Project)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	result, err := service.Rename(ctx, backend.RenameRequest{
		Project:           project,
		Symbol:            request.Symbol,
		To:                request.To,
		OrganizeImports:   effectiveAutoOrganize(cc, request.OrganizeImports),
		DeferFormatting:   cc.DeferVerification,
		DeferVerification: cc.DeferVerification,
	})
	if err != nil {
		var diagnosticsError *backend.RenameDiagnosticsError
		if errors.As(err, &diagnosticsError) && diagnosticsError.Result != nil {
			diagnosticsError.Result.Source = cloneRenameLookup(diagnosticsError.Result.Lookup)
			diagnosticsError.Result.Lookup = nil
			diagnosticsError.Result.ProjectRoot = project.RootDir
			diagnosticsError.Result.ApplicationState = string(EditPersisted)
			if sourcebuf.FromContext(ctx) != nil {
				diagnosticsError.Result.ApplicationState = string(EditAppliedWorking)
			}
		}
		return result, err
	}
	if result == nil {
		return nil, fmt.Errorf("rename backend returned no result")
	}
	result.ProjectRoot = project.RootDir
	result.ApplicationState = string(EditPersisted)
	if sourcebuf.FromContext(ctx) != nil {
		result.ApplicationState = string(EditAppliedWorking)
		result.Source = cloneRenameLookup(result.Lookup)
		result.Lookup = nil
		result.ContinuationUnavailable = "rename is part of an in-progress workspace batch"
		return result, nil
	}
	return resolvePostRenameIdentity(ctx, service, project, request, result), nil
}

func resolvePostRenameIdentity(ctx context.Context, service *backend.Service, project backend.ProjectContext, request RenameReq, result *backend.RenameResult) *backend.RenameResult {
	if result.Lookup == nil {
		result.ContinuationUnavailable = "backend returned no pre-rename identity"
		return result
	}
	result.Source = cloneRenameLookup(result.Lookup)
	result.Lookup = nil
	query, expectedSymbol := postRenameLookupName(result.Source, request.To)
	if query == "" {
		result.ContinuationUnavailable = "rename source did not contain a qualified identity"
		return result
	}
	resolved, err := service.Lookup(ctx, project, query)
	if err != nil {
		result.ContinuationUnavailable = "post-rename lookup failed: " + err.Error()
		return result
	}
	if resolved == nil || resolved.Ambiguous || resolved.Definition == nil {
		result.ContinuationUnavailable = "post-rename lookup did not resolve one declaration"
		return result
	}
	resolvedSymbol := resolved.Definition.QualifiedName
	if resolvedSymbol == "" || resolvedSymbol != expectedSymbol {
		result.ContinuationUnavailable = "post-rename lookup did not resolve the expected qualified identity"
		return result
	}
	if result.Source.Definition == nil || !sameWorkspacePath(project.RootDir, result.Source.Definition.File, resolved.Definition.File) {
		result.ContinuationUnavailable = "post-rename lookup resolved a different source file"
		return result
	}
	resolved.Symbol = resolvedSymbol
	result.Lookup = resolved
	inspectProject := project
	inspectProject.File = resolved.Definition.File
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: inspectProject, Symbol: resolvedSymbol})
	if err != nil || inspection == nil {
		if err != nil {
			result.ContinuationUnavailable = "post-rename inspection failed: " + err.Error()
		} else {
			result.ContinuationUnavailable = "post-rename inspection returned no result"
		}
		return result
	}
	matched := ""
	matchCount := 0
	for _, match := range inspection.Matches {
		if match.QualifiedName != resolvedSymbol || !sameWorkspacePath(project.RootDir, match.File, resolved.Definition.File) {
			continue
		}
		matchCount++
		if matchCount > 1 {
			result.ContinuationUnavailable = "post-rename inspection matched multiple declarations"
			return result
		}
		matched = match.Revision
	}
	result.Revision = matched
	if matchCount != 1 || matched == "" {
		result.ContinuationUnavailable = "post-rename inspection did not prove a matching file revision"
	}
	return result
}

func postRenameLookupName(source *backend.LookupResult, requestedName string) (string, string) {
	if source == nil {
		return "", ""
	}
	oldIdentity := source.Symbol
	if source.Definition != nil && source.Definition.QualifiedName != "" {
		oldIdentity = source.Definition.QualifiedName
	}
	newLeaf := backend.NormalizeRenameInput(requestedName)
	if oldIdentity == "" || newLeaf == "" {
		return "", ""
	}
	dot := strings.LastIndex(oldIdentity, ".")
	colon := strings.LastIndex(oldIdentity, "::")
	separator, width := dot, 1
	if colon > separator {
		separator, width = colon, 2
	}
	if separator < 0 {
		return newLeaf, newLeaf
	}
	qualified := oldIdentity[:separator+width] + newLeaf
	return qualified, qualified
}

func cloneRenameLookup(source *backend.LookupResult) *backend.LookupResult {
	if source == nil {
		return nil
	}
	clone := *source
	if source.Definition != nil {
		definition := *source.Definition
		clone.Definition = &definition
	}
	return &clone
}

func sameWorkspacePath(root, left, right string) bool {
	canonical := func(path string) (string, error) {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return filepath.Clean(absolute), nil
	}
	leftPath, leftErr := canonical(left)
	rightPath, rightErr := canonical(right)
	return leftErr == nil && rightErr == nil && leftPath == rightPath
}

func verifyRun(ctx context.Context, cc CallContext, request VerifyReq) (VerifyRes, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	project := effectiveProject(cc, request.Project)
	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = "."
	}
	path, err := canonicalVerifyPath(project.RootDir, path)
	if err != nil {
		return VerifyRes{}, err
	}
	if project.File != "" {
		project.File, err = canonicalVerifyPath(project.RootDir, project.File)
		if err != nil {
			return VerifyRes{}, err
		}
	}
	selectedLanguage, explicitSelection := verificationSelectionLanguage(project, path)
	if explicitSelection && !cc.AllowsLanguage(selectedLanguage) {
		return VerifyRes{}, fmt.Errorf("verification language %q is disabled by --enabled-languages", selectedLanguage)
	}
	if !request.CheckOnly && cc.AllowedLanguages != nil && (request.FormatSelectedFile || request.OrganizeImports) && !explicitSelection {
		return VerifyRes{Failure: "select an enabled source language or file before applying selected-file normalization"}, nil
	}
	if path != "." {
		if _, statErr := os.Stat(filepath.Join(project.RootDir, filepath.FromSlash(path))); statErr != nil {
			return VerifyRes{}, fmt.Errorf("verification path %q: %w", path, statErr)
		}
	}
	plan, err := projectverify.Discover(project.RootDir)
	if err != nil {
		return VerifyRes{}, fmt.Errorf("discover project verification plan: %w", err)
	}
	requestedScope := projectverify.Scope{}
	if path != "." {
		requestedScope.Paths = []string{path}
	}
	actualPaths := scopedSourcePaths(plan, requestedScope)
	allowedPlan := planForAllowedLanguages(plan, cc.AllowedLanguages)
	if len(actualPaths) == 0 && path != "." {
		actualPaths = []string{path}
	}
	scope := requestedScope
	service := serviceForContext(cc)
	response := VerifyRes{Scope: projectverify.Scope{Paths: actualPaths}}
	formatPaths := []string{path}
	if cc.GoFormattingPaths != nil {
		formatPaths = cc.GoFormattingPaths
	}
	goFormatPaths := make([]string, 0, len(formatPaths))
	for _, formatPath := range formatPaths {
		if cc.AllowsLanguage(backend.LanguageGo) && shouldRunGofmt(project, formatPath) {
			goFormatPaths = append(goFormatPaths, formatPath)
		}
	}
	if !request.CheckOnly && len(goFormatPaths) > 0 {
		needsFormatting, err := gofmtListsFiles(ctx, project.RootDir, goFormatPaths...)
		if err != nil {
			return response, fmt.Errorf("check Go formatting before verification: %w", err)
		}
		if needsFormatting {
			if err := pipeline.Format(ctx, project.RootDir, goFormatPaths...); err != nil {
				return response, fmt.Errorf("format Go files before verification: %w", err)
			}
			response.Formatted = true
		}
	}
	if !request.CheckOnly {
		if hookID, trustErr := workspaceTrustRequiredHook(allowedPlan, allowedPlan.Normalize, scope); trustErr != nil {
			return response, trustErr
		} else if hookID != "" && !project.WorkspaceTrust.Allows(project.RootDir) {
			response.Failure = fmt.Sprintf("workspace trust is required to run verification hook %q", hookID)
			return response, nil
		}
		response.NormalizeReport, err = projectverify.RunPhase(ctx, allowedPlan, projectverify.PhaseNormalize, scope, verificationLSPRunner(service, project, projectverify.PhaseNormalize, cc.AllowedLanguages))
		if err != nil {
			response.Failure = err.Error()
			// Preserve hook results while ingress maps Failure to a failed operation.
			return response, nil //nolint:nilerr // Preserve the partial public report; ingress maps Failure to an operation error.
		}
		if request.FormatSelectedFile || request.OrganizeImports {
			_, err = service.Verify(ctx, backend.VerifyRequest{Project: project, Path: path, FormatSelectedFile: request.FormatSelectedFile, OrganizeImports: request.OrganizeImports})
			if err != nil {
				response.Failure = fmt.Errorf("apply selected backend normalization: %w", err).Error()
				// Preserve completed normalization details in the public result.
				return response, nil
			}
		}
		plan, err = projectverify.Discover(project.RootDir)
		if err != nil {
			response.Failure = fmt.Errorf("rediscover project verification plan: %w", err).Error()
			// Preserve completed normalization details in the public result.
			return response, nil
		}
		allowedPlan = planForAllowedLanguages(plan, cc.AllowedLanguages)
		response.Scope = projectverify.Scope{Paths: scopedSourcePaths(plan, scope)}
		if len(response.Scope.Paths) == 0 && len(scope.Paths) > 0 {
			response.Scope = scope
		}
	}
	response.Diagnostics, response.Coverage, err = runPlanLSPVerification(ctx, service, project, request, plan, scope, cc.AllowedLanguages)
	if err != nil {
		response.Failure = err.Error()
		// Preserve diagnostics gathered before the failed hook.
		return response, nil //nolint:nilerr // Preserve the partial public report; ingress maps Failure to an operation error.
	}
	if hookID, trustErr := workspaceTrustRequiredHook(allowedPlan, allowedPlan.Checks, scope); trustErr != nil {
		return response, trustErr
	} else if hookID != "" && !project.WorkspaceTrust.Allows(project.RootDir) {
		response.Failure = fmt.Sprintf("workspace trust is required to run verification hook %q", hookID)
		return response, nil
	}
	response.CheckReport, err = projectverify.RunPhase(ctx, allowedPlan, projectverify.PhaseCheck, scope, verificationLSPRunner(service, project, projectverify.PhaseCheck, cc.AllowedLanguages))
	if err != nil {
		response.Failure = err.Error()
	}
	return response, nil
}

func shouldRunGofmt(project backend.ProjectContext, path string) bool {
	language := project.Language
	if language != "" && language != backend.LanguageAuto {
		return language == backend.LanguageGo
	}
	language = backend.LanguageIDFromFile(path)
	if language != "" && language != backend.LanguageAuto {
		return language == backend.LanguageGo
	}
	info, err := os.Stat(filepath.Join(project.RootDir, filepath.FromSlash(path)))
	return err == nil && info.IsDir()
}

func canonicalVerifyPath(root, requested string) (string, error) {
	if !filepath.IsAbs(requested) {
		clean := filepath.Clean(requested)
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("verification path %q is outside workspace %q", requested, pathutil.CanonicalWorkspaceRoot(root))
		}
		requested = filepath.Join(root, clean)
	}
	canonicalRoot := pathutil.CanonicalWorkspaceRoot(root)
	canonicalPath := pathutil.CanonicalWorkspaceRoot(requested)
	if !pathutil.PathWithin(canonicalRoot, canonicalPath) {
		return "", fmt.Errorf("verification path %q is outside workspace %q", requested, canonicalRoot)
	}
	rel, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil {
		return "", fmt.Errorf("resolve verification path %q relative to workspace: %w", requested, err)
	}
	return filepath.ToSlash(rel), nil
}

func scopedSourcePaths(plan projectverify.Plan, scope projectverify.Scope) []string {
	paths := make([]string, 0)
	for _, source := range plan.Sources {
		if sourceInScope(source.Path, scope) {
			paths = append(paths, source.Path)
		}
	}
	return paths
}

func hasMutatingLSPHooks(hooks []projectverify.Hook) bool {
	for _, hook := range hooks {
		if hook.LSP != nil && (hook.LSP.ActionKind != "" || hook.LSP.Command != "") {
			return true
		}
	}
	return false
}

// gofmtListsFiles reports whether gofmt would reformat sources under path.
func gofmtListsFiles(ctx context.Context, rootDir string, paths ...string) (bool, error) {
	formatted, err := pipeline.NeedsFormatting(ctx, rootDir, paths...)
	if err != nil {
		return false, fmt.Errorf("gofmt -l %s: %w", strings.Join(paths, ", "), err)
	}
	return formatted, nil
}

func formatLookup(result *backend.LookupResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format lookup result: %w", err)
	}
	return string(data), nil
}

func formatRename(result *backend.RenameResult) (string, error) {
	text := "Successfully renamed symbol."
	if result != nil && result.Lookup != nil {
		symbol := result.Lookup.Symbol
		if result.Lookup.Definition != nil && result.Lookup.Definition.QualifiedName != "" {
			symbol = result.Lookup.Definition.QualifiedName
		}
		text = fmt.Sprintf("Successfully renamed %s.", symbol)
	}
	if result != nil && result.ContinuationUnavailable != "" {
		text += " Final continuation unavailable: " + result.ContinuationUnavailable + "."
	}
	return text, nil
}

func formatVerify(result VerifyRes) (string, error) {
	messages := make([]string, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	text := "Verification clean: 0 diagnostics."
	if len(messages) > 0 {
		text = fmt.Sprintf("Verification: %d diagnostics:\n%s", len(messages), strings.Join(messages, "\n"))
	}
	if result.Failure != "" {
		text = "Verification failed: " + result.Failure
	}
	if result.Formatted {
		text += "\nGo formatting changes were applied."
	}
	report := struct {
		Scope       projectverify.Scope  `json:"scope"`
		Normalize   projectverify.Report `json:"normalize"`
		Checks      projectverify.Report `json:"checks"`
		Diagnostics []backend.Diagnostic `json:"diagnostics,omitempty"`
		Coverage    []LanguageCoverage   `json:"coverage"`
		Failure     string               `json:"failure,omitempty"`
	}{
		Scope: result.Scope, Normalize: result.NormalizeReport, Checks: result.CheckReport,
		Diagnostics: result.Diagnostics, Coverage: result.Coverage, Failure: result.Failure,
	}
	data, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("marshal verification report: %w", err)
	}
	return text + "\nVerification report: " + string(data), nil
}

func lookupDef() Def[LookupReq, *backend.LookupResult] {
	run := lookupRun
	return Def[LookupReq, *backend.LookupResult]{
		Key:     capability.OpLookup,
		Summary: "Use this tool instead of grep or text search when locating a named symbol in Go or a selected trusted source file. Omit file for Go workspace lookup. Rust, Java, Scala, Kotlin, Bash, Makefile, and Haskell lookup are read-only and require explicit workspace trust; external tools must already be installed. Returns symbol, file, kind, line, and column. Pass symbol and file to follow-up semedit calls. Use line and column only for file-navigation reads; they are not accepted as semedit tool parameters.",
		Params:  lookupParams,
		Level:   LevelSymbol,
		// Lookup is read-only; edit and verification operations can change source.
		ReadOnly: true,
		CLIName:  "lookup",
		MCPName:  semanticLookupToolName,
		Parse:    parseLookup,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, LookupReq) (*backend.LookupResult, error){
			backend.LanguageGo:      run,
			backend.LanguageRust:    run,
			backend.LanguageJava:    run,
			backend.LanguageScala:   run,
			backend.LanguageHaskell: run,
			backend.LanguageKotlin:  run,
			backend.LanguageBash:    run,
			backend.LanguageMake:    run,
		},
		Format: formatLookup,
		ExampleRaw: map[string]any{
			"symbol":           "Server.Start",
			"file":             wireExampleFile,
			"language":         "auto",
			wireTrustWorkspace: false,
		},
	}
}

func renameDef() Def[RenameReq, *backend.RenameResult] {
	run := renameRun
	output := renameOutputContract()
	return Def[RenameReq, *backend.RenameResult]{
		Key:           capability.OpRename,
		Summary:       "Use this tool instead of text search-and-replace when renaming a symbol and its references through the selected language backend. Go supports workspace rename; trusted Rust and Java support selected-file language-server rename. Other languages may be lookup-only." + automaticVerificationGuidance,
		Params:        renameParams,
		Level:         LevelSymbol,
		CLIName:       "rename",
		MCPName:       contractRenameTool,
		BufferedBatch: true, Batchable: true,
		Parse: parseRename,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, RenameReq) (*backend.RenameResult, error){
			backend.LanguageGo:   run,
			backend.LanguageRust: run,
			backend.LanguageJava: run,
		},
		Format:             formatRename,
		Output:             output,
		ErrorOutput:        &OperationErrorOutputContract{ResultSchema: renameErrorOutputSchema()},
		ProjectOutput:      projectRenameOutput,
		ProjectErrorOutput: projectRenameErrorOutput,
		ExampleRaw: map[string]any{
			"symbol":                "Server.Start",
			"to":                    "Serve",
			"file":                  wireExampleFile,
			wireAutoOrganizeImports: true,
		},
	}
}

func renameOutputContract() *OperationOutputContract {
	text := nonEmptyStringSchema()
	identitySchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file": text, "symbol": text,
	}, Required: []string{"file", "symbol"}}
	properties := map[string]ValueSchema{
		"status":                   {Kind: SchemaString, Enum: []any{"renamed"}},
		wireApplicationState:       {Kind: SchemaString, Enum: []any{string(EditPersisted), string(EditAppliedWorking)}},
		"origin":                   identitySchema,
		"file":                     text,
		"symbol":                   text,
		wireExpectedRevision:       text,
		wireDiagnostics:            diagnosticDeltaSchema(),
		"active":                   {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
		"continuation_unavailable": {Kind: SchemaString},
	}
	return &OperationOutputContract{
		ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: properties, Required: []string{"status", wireApplicationState, wireDiagnostics, "active"}},
		Continuations: []OutputContinuation{
			{Field: "file", Schema: text, Role: RoleTarget, Consumers: renameFileConsumers, Optional: true},
			{Field: "symbol", Schema: text, Role: RoleTarget, Consumers: renameSymbolConsumers, Optional: true},
			{Field: wireExpectedRevision, Schema: text, Role: RoleTarget, Consumers: renameRevisionConsumers, Optional: true},
		},
	}
}

func renameErrorOutputSchema() *ValueSchema {
	text := nonEmptyStringSchema()
	identitySchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"file": text, "symbol": text}, Required: []string{"file", "symbol"}}
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"status":             {Kind: SchemaString, Enum: []any{"applied_with_diagnostics"}},
		wireApplicationState: {Kind: SchemaString, Enum: []any{string(EditPersisted), string(EditAppliedWorking)}},
		"origin":             identitySchema,
		wireDiagnostics:      diagnosticDeltaSchema(),
		"active":             {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
	}, Required: []string{"status", wireApplicationState, "origin", wireDiagnostics, "active"}}
}

func projectRenameOutput(result *backend.RenameResult) (any, error) {
	if result == nil {
		return nil, fmt.Errorf("rename result is nil")
	}
	if result.ApplicationState == "" {
		return nil, fmt.Errorf("rename application state is unknown")
	}
	active := result.Active
	if active == nil {
		active = []string{}
	}
	projection := map[string]any{
		"status": "renamed", wireApplicationState: result.ApplicationState,
		wireDiagnostics: projectRenameDiagnostics(result.Diagnostics), "active": active,
	}
	if origin := projectRenameIdentity(result.ProjectRoot, result.Source); origin != nil {
		projection["origin"] = origin
	}
	if result.ContinuationUnavailable != "" {
		projection["continuation_unavailable"] = result.ContinuationUnavailable
	}
	if result.Lookup != nil && result.Revision != "" && result.ApplicationState == string(EditPersisted) {
		if identity := projectRenameIdentity(result.ProjectRoot, result.Lookup); identity != nil {
			projection["file"] = identity["file"]
			projection["symbol"] = identity["symbol"]
			projection[wireExpectedRevision] = result.Revision
		}
	}
	return projection, nil
}

func projectRenameErrorOutput(err error) (any, bool, error) {
	var diagnosticsError *backend.RenameDiagnosticsError
	if !errors.As(err, &diagnosticsError) || diagnosticsError.Result == nil || diagnosticsError.Result.Source == nil {
		return nil, false, nil
	}
	result := diagnosticsError.Result
	active := result.Active
	if active == nil {
		active = []string{}
	}
	if result.ApplicationState != string(EditPersisted) && result.ApplicationState != string(EditAppliedWorking) {
		return nil, false, nil
	}
	origin := projectRenameIdentity(result.ProjectRoot, result.Source)
	if origin == nil {
		return nil, false, nil
	}
	return map[string]any{
		"status": "applied_with_diagnostics", wireApplicationState: result.ApplicationState,
		"origin": origin, wireDiagnostics: projectRenameDiagnostics(result.Diagnostics), "active": active,
	}, true, nil
}

func projectRenameIdentity(root string, lookup *backend.LookupResult) map[string]any {
	if lookup == nil {
		return nil
	}
	file, symbol := lookup.File, lookup.Symbol
	if lookup.Definition != nil {
		file = lookup.Definition.File
		if lookup.Definition.QualifiedName != "" {
			symbol = lookup.Definition.QualifiedName
		}
	}
	file = renameProjectRelativeFile(root, file)
	if file == "" || symbol == "" {
		return nil
	}
	return map[string]any{"file": file, "symbol": symbol}
}

func renameProjectRelativeFile(root, file string) string {
	if file == "" {
		return ""
	}
	if root == "" {
		if filepath.IsAbs(file) {
			return ""
		}
		clean := filepath.Clean(file)
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return ""
		}
		return filepath.ToSlash(clean)
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	filePath := file
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(rootPath, filePath)
	}
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(rootPath, filePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(relative)
}

func projectRenameDiagnostics(delta backend.DiagnosticDelta) map[string]any {
	introduced := delta.Introduced
	if introduced == nil {
		introduced = []string{}
	}
	resolved := delta.Resolved
	if resolved == nil {
		resolved = []string{}
	}
	suggestions := delta.Suggestions
	if suggestions == nil {
		suggestions = []string{}
	}
	return map[string]any{"net_delta": delta.NetDelta, "introduced": introduced, "resolved": resolved, "suggestions": suggestions}
}

var renameFileConsumers = []string{
	semanticLookupToolName, semanticRenameToolName, "semantic_verify", semanticReplaceBodyToolName,
	semanticMoveConstructToolName, "semantic_remove_construct", "semantic_insert_case",
	"semantic_insert_function", "semantic_insert_type", "semantic_insert_decl",
	"semantic_insert_declaration", "semantic_insert_construct", semanticReplaceDeclToolName,
	semanticExtractFunctionToolName, semanticExtractVariableToolName,
	semanticInlineFunctionToolName, semanticInlineVariableToolName, semanticMigrateTypeToolName,
}

var renameSymbolConsumers = []string{
	semanticLookupToolName, semanticRenameToolName, semanticReplaceBodyToolName, semanticMoveConstructToolName,
	semanticReplaceDeclToolName, semanticInlineFunctionToolName, semanticMigrateTypeToolName,
}

var renameRevisionConsumers = []string{
	"semantic_replace_body", semanticInlineFunctionToolName, semanticMigrateTypeToolName,
}

func verifyDef() Def[VerifyReq, VerifyRes] {
	return Def[VerifyReq, VerifyRes]{
		Key:     capability.OpVerify,
		Summary: "Use this tool instead of ad hoc diagnostic commands to verify source files with detected or explicitly selected language servers. Go verification formats files before diagnostics; check-only skips all normalization.",
		Params:  verifyParams,
		Level:   LevelBuild,
		CLIName: "verify",
		MCPName: "semantic_verify",
		Parse:   parseVerify,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, VerifyReq) (VerifyRes, error){
			backend.LanguageGo:     verifyRun,
			backend.LanguageJava:   verifyRun,
			backend.LanguageKotlin: verifyRun,
			backend.LanguageBash:   verifyRun,
			backend.LanguageAuto:   verifyRun,
		},
		Format: formatVerify,
		ExampleRaw: map[string]any{
			"path":             ".",
			"language":         "go",
			wireTrustWorkspace: false,
		},
	}
}

// DefaultRegistry constructs the registry with lookup, rename, and verify operations wired to backend leaves.
func DefaultRegistry() *Registry {
	registry := NewRegistry()
	for _, register := range []func(*Registry) error{
		registerNavigationOps,
		registerRenameOps,
		registerVerificationOps,
	} {
		if err := register(registry); err != nil {
			panic(err)
		}
	}
	if err := registerReadInspectOps(registry); err != nil {
		panic(err)
	}
	if err := registerEngineOps(registry); err != nil {
		panic(err)
	}
	if err := registerWorkspaceOps(registry); err != nil {
		panic(err)
	}
	if err := registerMavenOps(registry); err != nil {
		panic(err)
	}
	if err := registerContractOps(registry); err != nil {
		panic(err)
	}
	return registry
}
