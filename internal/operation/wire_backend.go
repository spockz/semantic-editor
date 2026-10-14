// Package operation wires backend lookup, rename, and verify leaves into registry operations.
package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/capability"
	"github.com/spockz/semantic-editor/internal/navigation"
	"github.com/spockz/semantic-editor/internal/projectverify"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/verification"
)

// languageEnums accepts every backend language so support is decided by handler presence, not parsing.

const wireCLIExpectedRevision = "expected-revision"

var renameParams = []ParameterContract{
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: symbol identifier (e.g. 'NormalizeEvent', 'Server.Start', or '(*Client).Do'). Direct short names are resolved in-place; calling semantic_lookup first is not required.", Type: ParamString, Role: RoleTarget, Required: true, SourceFields: append(append(append(append([]string(nil), sourceSymbolContinuations...), sourceExtractFunctionReceiptSymbol, sourceExtractConstantReceiptSymbol, sourceExtractVariableReceiptSymbol, sourceInlineVariableReceiptSymbol), sourceContractExtractSymbol, sourceContractImplementSymbol), sourceConcreteTargetSymbol, sourceGeneratedMethodSymbol)},
	{Name: "to", CLIName: "to", JSONName: "to", Description: "New identifier name (e.g. Serve)", Type: ParamString, Required: true},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Target: source file containing the declaration (e.g. 'pipeline/workflow.go'). Optional for Go if the symbol is unique across the workspace, but recommended to disambiguate common names. Required for Rust or Java.", Type: ParamString, Role: RoleTarget, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceExtractFunctionReceiptFile, sourceExtractConstantReceiptFile, "semantic_extract_variable.result.continuations[].file", "semantic_inline_variable.result.continuations[].file", sourceContractExtractFile, sourceContractImplementFile, sourceConcreteTargetFile, sourceGeneratedMethodFile}},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto; Rust requires a selected .rs file and workspace trust; Java requires a selected .java file and trust_workspace=true)", Type: ParamString, Enums: languageEnums, LanguageSelector: true},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for this request; required for Rust or Java rename (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java rename", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Enable or disable trusted JDT LS Maven import; defaults on when a trusted workspace contains pom.xml and never runs Maven", Type: ParamBoolean},
	{Name: wireAutoOrganizeImports, CLIName: "", JSONName: wireAutoOrganizeImports, Description: "Automatically clean up and organize imports after rename (default true)", Type: ParamBoolean, Default: true},
}

var lookupParams = readRequestParams(
	ParameterContract{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: qualified symbol identifier (e.g. Server.ServeHTTP, (*Client).Do, or ValidateToken)", Type: ParamString, Role: RoleTarget, Required: true, SourceFields: append(sourceSymbolContinuations, sourceContractExtractSymbol, sourceContractImplementSymbol)},
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Description: "Target: optional source file that constrains symbol search scope", Type: ParamString, Role: RoleTarget, SourceFields: append(sourceFileContinuations, sourceContractExtractFile, sourceContractImplementFile)},
)

const (
	wireCLITrustWorkspace         = "trust-workspace"
	wireJDTLSHome                 = "jdtls_home"
	wireImportMaven               = "import_maven"
	wireCLIAutoOrganizeImports    = "auto-organize-imports"
	wireAccessModifier            = "access_modifier"
	wireNavigationHintDescription = "Navigation hint; not an operation parameter."
)

const (
	wireTrustWorkspace          = "trust_workspace"
	wireSkipHooks               = "skip_hooks"
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

const (
	operationKeyOrganizeImports = "organize_imports"
	operationKeyInsertConstruct = "insert_construct"
	operationKeyRemoveConstruct = "remove_construct"
	operationKeyDependency      = "dependency"
	wireDeclaration             = "declaration"
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
	semanticMoveConstructToolName + ".file",
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
	Project  backend.ProjectContext
	Symbol   string
	WorkRoot string
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
	SkipHooks          bool
}

// GetProjectContext returns the request project for registry dispatch.
func (r VerifyReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project with the effective dispatch project.
func (r *VerifyReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// LookupTarget describes one public lookup target without exposing source URIs.
type LookupTarget struct {
	Name          string `json:"name,omitempty"`
	QualifiedName string `json:"qualified_name,omitempty"`
	Kind          string `json:"kind,omitempty"`
	File          string `json:"file,omitempty"`
	Line          int    `json:"line,omitempty"`
	Column        int    `json:"column,omitempty"`
}

// LookupRes contains the public lookup projection.
type LookupRes struct {
	Symbol     string         `json:"symbol,omitempty"`
	File       string         `json:"file,omitempty"`
	Line       int            `json:"line,omitempty"`
	Column     int            `json:"column,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	Definition *LookupTarget  `json:"definition,omitempty"`
	Usages     []LookupTarget `json:"usages,omitempty"`
	Ambiguous  bool           `json:"ambiguous,omitempty"`
	Candidates []LookupTarget `json:"candidates,omitempty"`
	WorkRoot   string         `json:"-"`
}

// RenameRes reports the resolved target and diagnostic delta after a rename.
type RenameRes = backend.RenameResult

// VerifyRes reports compiler diagnostics and whether gofmt reformatted sources.
type VerifyRes = verification.Result

// LanguageCoverage reports one language's verification scope and outcome.
type LanguageCoverage = verification.LanguageCoverage

var verifyParams = []ParameterContract{
	{Name: "path", CLIName: "path", JSONName: "path", Description: "Target Go file/directory or Java directory scope to format and check; relative paths resolve from the active semedit workspace root. Go verification runs formatting before diagnostics and can write files during normalization", Type: ParamString, Default: "."},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Target: Java, Kotlin, or Bash source file to verify; path is relative to the workspace root", Type: ParamString, Role: RoleTarget, SourceFields: []string{sourceLookupFile, sourceRenameFile}},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto)", Type: ParamString, Enums: languageEnums, LanguageSelector: true},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for this request; required for external language verification (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java verification", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Enable or disable trusted JDT LS Maven import; defaults on when a trusted workspace contains pom.xml and never runs Maven", Type: ParamBoolean},
	{Name: "format_selected_file", CLIName: "format-selected-file", JSONName: "format_selected_file", Description: "Apply JDT LS formatting to the selected Java file or every Java file beneath a selected directory", Type: ParamBoolean, Default: false},
	{Name: operationKeyOrganizeImports, CLIName: "organize-imports", JSONName: operationKeyOrganizeImports, Description: "Apply JDT LS source.organizeImports to the selected Java file or every Java file beneath a selected directory", Type: ParamBoolean, Default: false},
	{Name: "kotlin_bin", CLIName: "kotlin-bin", JSONName: "kotlin_bin", Description: "Path to a preinstalled fwcd/kotlin-language-server executable", Type: ParamString},
	{Name: "bash_bin", CLIName: "bash-bin", JSONName: "bash_bin", Description: "Path to a preinstalled bash-language-server executable", Type: ParamString},
	{Name: "make_bin", CLIName: "make-bin", JSONName: "make_bin", Description: "Path to a preinstalled make-ls executable", Type: ParamString},
	{Name: "check_only", CLIName: "check-only", JSONName: "check_only", Description: "Skip all formatting and normalization hooks; verify the current files only", Type: ParamBoolean, Default: false},
	{Name: wireSkipHooks, CLIName: "skip-hooks", JSONName: wireSkipHooks, Description: "Skip configured project hooks while preserving language-server diagnostics and built-in formatting", Type: ParamBoolean, Default: false},
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
	return LookupReq{Project: project, Symbol: symbol, WorkRoot: project.RootDir}, nil
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
	_, importMavenSet := rawLookup(raw, wireImportMaven, "import-maven")
	organizeImports, err := ParseBool(raw, wireAutoOrganizeImports, "", true)
	if err != nil {
		return RenameReq{}, err
	}
	return RenameReq{
		Project: backend.ProjectContext{
			File:           file,
			Language:       backend.LanguageID(language),
			WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted},
			Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven, ImportMavenSet: importMavenSet},
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
	_, importMavenSet := rawLookup(raw, wireImportMaven, "import-maven")
	formatSelected, err := ParseBool(raw, "format_selected_file", "format-selected-file", false)
	if err != nil {
		return VerifyReq{}, err
	}
	organizeImports, err := ParseBool(raw, operationKeyOrganizeImports, "organize-imports", false)
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
	skipHooks, err := ParseBool(raw, wireSkipHooks, "skip-hooks", false)
	if err != nil {
		return VerifyReq{}, err
	}
	return VerifyReq{
		Project: backend.ProjectContext{
			File:           path,
			Language:       backend.LanguageID(language),
			WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted},
			Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven, ImportMavenSet: importMavenSet},
			Kotlin:         backend.KotlinConfig{KotlinBin: kotlinBin},
			Bash:           backend.BashConfig{BashBin: bashBin},
			Make:           backend.MakeConfig{MakeBin: makeBin},
		},
		Path:               path,
		FormatSelectedFile: formatSelected,
		OrganizeImports:    organizeImports, CheckOnly: checkOnly, SkipHooks: skipHooks,
	}, nil
}

func lookupRun(ctx context.Context, cc CallContext, request LookupReq) (*LookupRes, error) {
	project := effectiveProject(cc, request.Project)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	result, err := navigation.Lookup(ctx, service, project, request.Symbol)
	if err != nil {
		return nil, err
	}
	workRoot := cc.WorkDir
	if workRoot == "" {
		workRoot = project.RootDir
	}
	if workRoot == "" {
		workRoot = request.WorkRoot
	}
	return lookupResponse(workRoot, request.Symbol, result)
}

func lookupResponse(root, query string, result *backend.LookupResult) (*LookupRes, error) {
	if result == nil {
		return nil, fmt.Errorf("lookup backend returned no result")
	}
	ambiguous := result.IsAmbiguous()
	if result.Definition != nil && len(result.Candidates) != 0 {
		return nil, fmt.Errorf("lookup backend returned both a definition and candidates")
	}
	if result.Definition == nil && len(result.Candidates) < 2 {
		return nil, fmt.Errorf("lookup backend returned neither a unique definition nor multiple candidates")
	}
	response := &LookupRes{Symbol: query, WorkRoot: pathutil.CanonicalWorkspaceRoot(root), Ambiguous: ambiguous}
	if result.Definition != nil {
		definition, err := lookupTarget(root, result.Definition)
		if err != nil {
			return nil, fmt.Errorf("project lookup definition: %w", err)
		}
		response.Definition = &definition
		response.Symbol = result.Definition.Identity()
		response.File, response.Line, response.Column, response.Kind = definition.File, definition.Line, definition.Column, definition.Kind
	}
	for _, usage := range result.Usages {
		target, err := lookupTarget(root, usage)
		if err != nil {
			return nil, fmt.Errorf("project lookup usage: %w", err)
		}
		response.Usages = append(response.Usages, target)
	}
	for _, candidate := range result.Candidates {
		target, err := lookupTarget(root, candidate)
		if err != nil {
			return nil, fmt.Errorf("project lookup candidate: %w", err)
		}
		response.Candidates = append(response.Candidates, target)
	}
	return response, nil
}

func lookupTarget(root string, target *backend.NamedTarget) (LookupTarget, error) {
	if target == nil {
		return LookupTarget{}, fmt.Errorf("nil lookup target")
	}
	file, err := pathutil.FilePathFromURI(target.Location.URI)
	if err != nil {
		return LookupTarget{}, fmt.Errorf("parse target URI: %w", err)
	}
	if strings.TrimSpace(target.Name) == "" || strings.TrimSpace(target.Kind) == "" {
		return LookupTarget{}, fmt.Errorf("lookup target has no name or kind")
	}
	rangeValue := target.Location.Range
	start, end := rangeValue.Start, rangeValue.End
	if start.Line < 0 || start.Character < 0 || end.Line < 0 || end.Character < 0 ||
		end.Line < start.Line || (end.Line == start.Line && end.Character <= start.Character) {
		return LookupTarget{}, fmt.Errorf("lookup target has invalid identifier range")
	}
	position := start
	qualifiedName := target.Identity()
	return LookupTarget{
		Name: target.Name, QualifiedName: qualifiedName, Kind: target.Kind,
		File: lookupWorkspacePath(root, file), Line: position.Line + 1, Column: position.Character + 1,
	}, nil
}

func lookupWorkspacePath(root, source string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	root = pathutil.CanonicalWorkspaceRoot(root)
	absolute := source
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, absolute)
	}
	absolute = filepath.Clean(absolute)
	if !pathutil.PathWithin(root, absolute) {
		return ""
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == "." {
		return ""
	}
	return filepath.ToSlash(relative)
}

func renameRun(ctx context.Context, cc CallContext, request RenameReq) (*backend.RenameResult, error) {
	project := effectiveProject(cc, request.Project)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	buffered := ctx != nil && sourcebuf.FromContext(ctx) != nil
	return navigation.Rename(ctx, service, navigation.RenameRequest{
		Project:           project,
		Symbol:            request.Symbol,
		To:                request.To,
		OrganizeImports:   effectiveAutoOrganize(cc, request.OrganizeImports),
		DeferVerification: cc.DeferVerification,
		Buffered:          buffered,
	})
}

func verifyRun(ctx context.Context, cc CallContext, request VerifyReq) (VerifyRes, error) {
	project := effectiveProject(cc, request.Project)
	return verification.Run(ctx, verification.Request{
		Project:              project,
		Path:                 request.Path,
		FormatSelectedFile:   request.FormatSelectedFile,
		OrganizeImports:      request.OrganizeImports,
		CheckOnly:            request.CheckOnly,
		SkipHooks:            request.SkipHooks,
		AllowedLanguages:     cc.AllowedLanguages,
		GoFormattingPaths:    cc.GoFormattingPaths,
		GoPackagePatterns:    cc.GoPackagePatterns,
		GoPackageScopeDetail: cc.GoPackageScopeDetail,
		Service:              cc.Service,
	})
}

// SelectedGoModuleRoots retains the batch verifier's path helper at the operation boundary.
func SelectedGoModuleRoots(projectRoot string, selectedPaths []string) ([]string, error) {
	return verification.SelectedGoModuleRoots(projectRoot, selectedPaths)
}

func formatLookup(result *LookupRes) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format lookup result: %w", err)
	}
	return string(data), nil
}

func formatRename(result *backend.RenameResult) (string, error) {
	text := "Successfully renamed symbol."
	if result != nil && result.Lookup != nil {
		symbol := ""
		if result.Lookup.Definition != nil {
			symbol = result.Lookup.Definition.Identity()
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

func lookupDef() Def[LookupReq, *LookupRes] {
	text := ValueSchema{Kind: SchemaString}
	candidate := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"name": text, outputQualifiedNameField: text, "kind": text, "file": text,
		"line": {Kind: SchemaInteger}, "column": {Kind: SchemaInteger},
	}, Required: []string{"name", "kind"}}
	schema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"symbol": text, "file": text, "kind": text, "line": {Kind: SchemaInteger}, "column": {Kind: SchemaInteger},
		"ambiguous": {Kind: SchemaBoolean}, "candidates": {Kind: SchemaArray, Items: &candidate},
	}, Required: []string{"ambiguous"}}
	return Def[LookupReq, *LookupRes]{
		Key:     capability.OpLookup,
		Summary: "Use this tool instead of grep or text search when locating a named symbol in Go or a selected trusted source file. Omit file for Go workspace lookup. Rust, Java, Scala, Kotlin, Bash, Makefile, and Haskell lookup are read-only and require explicit workspace trust; external tools must already be installed. Returns symbol, file, kind, line, and column. Pass symbol and file to follow-up semedit calls. Use line and column only for file-navigation reads; they are not accepted as semedit tool parameters.",
		Params:  lookupParams, Level: LevelSymbol, ReadOnly: true,
		CLIName: "lookup", MCPName: semanticLookupToolName, Parse: parseLookup,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, LookupReq) (*LookupRes, error){
			backend.LanguageGo: lookupRun, backend.LanguageRust: lookupRun, backend.LanguageJava: lookupRun,
			backend.LanguageScala: lookupRun, backend.LanguageHaskell: lookupRun, backend.LanguageKotlin: lookupRun,
			backend.LanguageBash: lookupRun, backend.LanguageMake: lookupRun,
		},
		Format: formatLookup,
		Output: &OperationOutputContract{ResultSchema: &schema, Continuations: []OutputContinuation{
			outputContinuation("symbol", RoleTarget, symbolOutputConsumers), outputContinuation("file", RoleTarget, fileOutputConsumers),
		}},
		ProjectOutput: projectLookupResult,
		ExampleRaw:    map[string]any{"symbol": "Server.Start", "file": wireExampleFile, "language": "auto", wireTrustWorkspace: false},
	}
}

func projectLookupResult(result *LookupRes) (any, error) {
	if result == nil {
		return nil, fmt.Errorf("project lookup: missing result: %w", ErrInvalidDefinition)
	}
	projected := map[string]any{"ambiguous": result.Ambiguous}
	if !result.Ambiguous {
		projectResolvedLookup(result, projected)
		return projected, nil
	}
	candidates := make([]LookupTarget, 0, len(result.Candidates))
	candidates = append(candidates, result.Candidates...)
	projected["candidates"] = candidates
	return projected, nil
}

func projectResolvedLookup(result *LookupRes, projected map[string]any) {
	if result.Symbol != "" {
		projected["symbol"] = result.Symbol
	}
	if result.File != "" {
		projected["file"] = result.File
	}
	if result.Kind != "" {
		projected["kind"] = result.Kind
	}
	if result.Line > 0 {
		projected["line"] = result.Line
	}
	if result.Column > 0 {
		projected["column"] = result.Column
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
		"target":                   {Kind: SchemaObject, Properties: map[string]ValueSchema{"file": text}, Required: []string{"file"}},
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
	origin, found, err := projectRenameIdentity(result.ProjectRoot, result.Source)
	if err != nil {
		return nil, err
	}
	if found {
		projection["origin"] = origin.wireValue()
	}
	if result.TargetFile != "" {
		targetFile := renameProjectRelativeFile(result.ProjectRoot, result.TargetFile)
		if targetFile == "" {
			return nil, fmt.Errorf("java rename target file is outside the workspace")
		}
		projection["target"] = map[string]any{"file": targetFile}
	}
	if result.ContinuationUnavailable != "" {
		projection["continuation_unavailable"] = result.ContinuationUnavailable
	}
	if result.Lookup != nil && result.Revision != "" && result.ApplicationState == string(EditPersisted) {
		identity, identityFound, identityErr := projectRenameIdentity(result.ProjectRoot, result.Lookup)
		if identityErr != nil {
			return nil, identityErr
		}
		if identityFound {
			projection["file"] = identity.File
			projection["symbol"] = identity.Symbol
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
	origin, found, err := projectRenameIdentity(result.ProjectRoot, result.Source)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return map[string]any{
		"status": "applied_with_diagnostics", wireApplicationState: result.ApplicationState,
		"origin": origin.wireValue(), wireDiagnostics: projectRenameDiagnostics(result.Diagnostics), "active": active,
	}, true, nil
}

type renameIdentity struct {
	File   string
	Symbol string
}

func (identity renameIdentity) wireValue() map[string]any {
	return map[string]any{"file": identity.File, "symbol": identity.Symbol}
}

func projectRenameIdentity(root string, lookup *backend.LookupResult) (renameIdentity, bool, error) {
	if lookup == nil || lookup.Definition == nil {
		return renameIdentity{}, false, nil
	}
	file, err := pathutil.FilePathFromURI(lookup.Definition.Location.URI)
	if err != nil {
		return renameIdentity{}, false, fmt.Errorf("project rename identity location: %w", err)
	}
	symbol := lookup.Definition.Identity()
	file = renameProjectRelativeFile(root, file)
	if file == "" || symbol == "" {
		return renameIdentity{}, false, nil
	}
	return renameIdentity{File: file, Symbol: symbol}, true, nil
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
	return map[string]any{"net_delta": delta.NetDelta, introducedField: introduced, "resolved": resolved, "suggestions": suggestions}
}

var renameFileConsumers = []string{
	semanticLookupToolName, semanticRenameToolName, "semantic_verify", semanticReplaceBodyToolName,
	semanticAssertionModeToolName, semanticExtractConstantToolName,
	semanticMoveConstructToolName, semanticRemoveConstructToolName, semanticReplaceConstructToolName, semanticInsertCaseToolName,
	semanticInsertFunctionToolName, semanticInsertTypeToolName, semanticInsertDeclToolName,
	semanticInsertDeclarationToolName, semanticInsertConstructToolName, semanticReplaceDeclToolName,
	semanticExtractFunctionToolName, semanticExtractVariableToolName,
	semanticInlineFunctionToolName, semanticInlineVariableToolName, semanticMigrateTypeToolName,
	semanticFindReferencesToolName, semanticInspectSymbolToolName, semanticInspectConstructToolName,
}

var renameSymbolConsumers = []string{
	semanticLookupToolName, semanticRenameToolName, semanticReplaceBodyToolName, semanticMoveConstructToolName,
	semanticReplaceDeclToolName, semanticInlineFunctionToolName, semanticMigrateTypeToolName,
	semanticFindReferencesToolName, semanticInspectSymbolToolName, semanticRemoveConstructToolName,
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
		Format:        formatVerify,
		Output:        &OperationOutputContract{ResultSchema: NativeVerifyOutputSchema()},
		ProjectOutput: func(result VerifyRes) (any, error) { return result, nil },
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
	if err := registerInspectConstructOps(registry); err != nil {
		panic(err)
	}
	if err := registerEngineOps(registry); err != nil {
		panic(err)
	}
	if err := registerWorkspaceOps(registry); err != nil {
		panic(err)
	}
	if err := registerWorkspaceOutlineOps(registry); err != nil {
		panic(err)
	}
	if err := registerMavenOps(registry); err != nil {
		panic(err)
	}
	if err := registerContractOps(registry); err != nil {
		panic(err)
	}
	if err := registerStageOps(registry); err != nil {
		panic(err)
	}
	if err := registry.installLanguageContracts(); err != nil {
		panic(err)
	}
	return registry
}
