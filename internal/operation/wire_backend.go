// Package operation wires backend lookup, rename, and verify leaves into registry operations.
package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/capability"
	"semedit/internal/navigation"
	"semedit/internal/projectverify"
	"semedit/internal/sourcebuf"
	"semedit/internal/verification"
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
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
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
type VerifyRes = verification.Result

// LanguageCoverage reports one language's verification scope and outcome.
type LanguageCoverage = verification.LanguageCoverage

var verifyParams = []ParameterContract{
	{Name: "path", CLIName: "path", JSONName: "path", Description: "Optional Go file or directory to format and check; relative paths resolve from the active semedit workspace root. Go verification runs formatting before diagnostics and can write files", Type: ParamString, Default: "."},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Target: Java, Kotlin, or Bash source file to verify; path is relative to the workspace root", Type: ParamString, Role: RoleTarget, SourceFields: []string{sourceLookupFile, sourceRenameFile}},
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
	project := effectiveProject(cc, request.Project)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	return navigation.Lookup(ctx, service, project, request.Symbol)
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
		Project:            project,
		Path:               request.Path,
		FormatSelectedFile: request.FormatSelectedFile,
		OrganizeImports:    request.OrganizeImports,
		CheckOnly:          request.CheckOnly,
		AllowedLanguages:   cc.AllowedLanguages,
		GoFormattingPaths:  cc.GoFormattingPaths,
		Service:            cc.Service,
	})
}

// SelectedGoModuleRoots retains the batch verifier's path helper at the operation boundary.
func SelectedGoModuleRoots(projectRoot string, selectedPaths []string) ([]string, error) {
	return verification.SelectedGoModuleRoots(projectRoot, selectedPaths)
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
		Output: legacyEnvelopeOutput([]EnvelopeContinuationField{
			legacyEnvelopeField("symbol", RoleTarget, envelopeSymbolConsumers),
			legacyEnvelopeField("file", RoleTarget, envelopeFileConsumers),
		},
			OutputMetadataField{Field: "kind", Schema: ValueSchema{Kind: SchemaString}, Description: "Descriptive symbol category; its vocabulary differs from mutation kind inputs.", Optional: true},
			OutputMetadataField{Field: "line", Schema: ValueSchema{Kind: SchemaInteger}, Description: wireNavigationHintDescription, Optional: true},
			OutputMetadataField{Field: "column", Schema: ValueSchema{Kind: SchemaInteger}, Description: wireNavigationHintDescription, Optional: true},
		),
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
	semanticAssertionModeToolName, semanticExtractConstantToolName,
	semanticMoveConstructToolName, semanticRemoveConstructToolName, semanticReplaceConstructToolName, semanticInsertCaseToolName,
	semanticInsertFunctionToolName, semanticInsertTypeToolName, semanticInsertDeclToolName,
	semanticInsertDeclarationToolName, semanticInsertConstructToolName, semanticReplaceDeclToolName,
	semanticExtractFunctionToolName, semanticExtractVariableToolName,
	semanticInlineFunctionToolName, semanticInlineVariableToolName, semanticMigrateTypeToolName,
	semanticFindReferencesToolName, semanticInspectSymbolToolName,
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
	if err := registry.installLanguageContracts(); err != nil {
		panic(err)
	}
	return registry
}
