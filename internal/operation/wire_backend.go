// Package operation wires backend lookup, rename, and verify leaves into registry operations.
package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/capability"
)

// languageEnums accepts every backend language so support is decided by handler presence, not parsing.

const (
	wireCLITrustWorkspace      = "trust-workspace"
	wireJDTLSHome              = "jdtls_home"
	wireImportMaven            = "import_maven"
	wireCLIAutoOrganizeImports = "auto-organize-imports"
	wireAccessModifier         = "access_modifier"
)

const (
	wireTrustWorkspace      = "trust_workspace"
	wireAutoOrganizeImports = "auto_organize_imports"
	wireExampleFile         = "api/server.go"
	wireTargetSymbol        = "target_symbol"
)

var languageEnums = []string{"auto", "go", "rust", "java", "scala", "haskell"}

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
	Diagnostics []backend.Diagnostic
	Formatted   bool
}

var lookupParams = []ParameterContract{
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target symbol identifier (e.g. Server.Start or ValidateToken)", Type: ParamString, Required: true},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Optional file path to constrain search", Type: ParamString},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto; Haskell requires standalone_haskell=true and supports read-only lookup only)", Type: ParamString, Enums: languageEnums},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for future external-tool backends (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java lookup", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
	{Name: "metals_home", CLIName: "metals-home", JSONName: "metals_home", Description: "Explicit preinstalled pinned Metals distribution home required for Scala lookup", Type: ParamString},
	{Name: "metals_bin", CLIName: "metals-bin", JSONName: "metals_bin", Description: "Direct pinned Metals executable, alternative to metals_home", Type: ParamString},
	{Name: "java_version", CLIName: "java-version", JSONName: "java_version", Description: "Recorded Java major version required for Scala lookup", Type: ParamString},
	{Name: "standalone_haskell", CLIName: "haskell-standalone", JSONName: "standalone_haskell", Description: "Explicitly select standalone Haskell .hs lookup; rejects project markers", Type: ParamBoolean, Default: false},
	{Name: "ghc_bin", CLIName: "ghc-bin", JSONName: "ghc_bin", Description: "Preinstalled GHC executable for standalone Haskell lookup", Type: ParamString},
	{Name: "hls_bin", CLIName: "hls-bin", JSONName: "hls_bin", Description: "Preinstalled haskell-language-server-wrapper executable", Type: ParamString},
	{Name: "ghc_version", CLIName: "ghc-version", JSONName: "ghc_version", Description: "Recorded GHC version to require", Type: ParamString},
	{Name: "hls_version", CLIName: "hls-version", JSONName: "hls_version", Description: "Recorded HLS version to require", Type: ParamString},
}

var renameParams = []ParameterContract{
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target symbol identifier (e.g. Server.Start or ValidateToken)", Type: ParamString, Required: true},
	{Name: "to", CLIName: "to", JSONName: "to", Description: "New identifier name (e.g. Serve)", Type: ParamString, Required: true},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Optional file path containing the declaration to disambiguate scope", Type: ParamString},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto; Rust requires a selected .rs file and workspace trust; Java requires a selected .java file and workspace trust)", Type: ParamString, Enums: languageEnums},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for future external-tool backends (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java rename", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
	{Name: wireAutoOrganizeImports, CLIName: "", JSONName: wireAutoOrganizeImports, Description: "Automatically clean up and organize imports after rename (default true)", Type: ParamBoolean, Default: true},
}

var verifyParams = []ParameterContract{
	{Name: "path", CLIName: "path", JSONName: "path", Description: "Optional file or directory path to check and format", Type: ParamString, Default: "."},
	{Name: "file", CLIName: "file", JSONName: "file", Description: "Selected Java source file; required for Java verification", Type: ParamString},
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto)", Type: ParamString, Enums: languageEnums},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for future external-tool backends (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java verification", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
	{Name: "format_selected_file", CLIName: "format-selected-file", JSONName: "format_selected_file", Description: "Apply JDT LS formatting to the selected Java file", Type: ParamBoolean, Default: false},
	{Name: "organize_imports", CLIName: "organize-imports", JSONName: "organize_imports", Description: "Apply bounded JDT LS source.organizeImports to the selected Java file", Type: ParamBoolean, Default: false},
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
	language, err := ParseEnum(raw, "language", "language", languageEnums, false, "")
	if err != nil {
		return LookupReq{}, err
	}
	trusted, err := ParseBool(raw, wireTrustWorkspace, wireCLITrustWorkspace, false)
	if err != nil {
		return LookupReq{}, err
	}
	jdtlsHome, err := ParseString(raw, wireJDTLSHome, "jdtls-home", false)
	if err != nil {
		return LookupReq{}, err
	}
	javaBin, err := ParseString(raw, "java_bin", "java-bin", false)
	if err != nil {
		return LookupReq{}, err
	}
	importMaven, err := ParseBool(raw, wireImportMaven, "import-maven", false)
	if err != nil {
		return LookupReq{}, err
	}
	metalsHome, err := ParseString(raw, "metals_home", "metals-home", false)
	if err != nil {
		return LookupReq{}, err
	}
	metalsBin, err := ParseString(raw, "metals_bin", "metals-bin", false)
	if err != nil {
		return LookupReq{}, err
	}
	javaVersion, err := ParseString(raw, "java_version", "java-version", false)
	if err != nil {
		return LookupReq{}, err
	}
	standalone, err := ParseBool(raw, "standalone_haskell", "haskell-standalone", false)
	if err != nil {
		return LookupReq{}, err
	}
	ghcBin, err := ParseString(raw, "ghc_bin", "ghc-bin", false)
	if err != nil {
		return LookupReq{}, err
	}
	hlsBin, err := ParseString(raw, "hls_bin", "hls-bin", false)
	if err != nil {
		return LookupReq{}, err
	}
	ghcVersion, err := ParseString(raw, "ghc_version", "ghc-version", false)
	if err != nil {
		return LookupReq{}, err
	}
	hlsVersion, err := ParseString(raw, "hls_version", "hls-version", false)
	if err != nil {
		return LookupReq{}, err
	}
	return LookupReq{
		Project: backend.ProjectContext{
			File:           file,
			Language:       backend.LanguageID(language),
			WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted},
			Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven},
			Scala: backend.ScalaConfig{
				MetalsHome: metalsHome, MetalsBin: metalsBin, JavaBin: javaBin, JavaVersion: javaVersion,
			},
			Haskell: backend.HaskellConfig{
				Standalone: standalone, GHCBin: ghcBin, HLSBin: hlsBin, GHCVersion: ghcVersion, HLSVersion: hlsVersion,
			},
			HaskellStandalone: standalone,
		},
		Symbol: symbol,
	}, nil
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
	return VerifyReq{
		Project: backend.ProjectContext{
			File:           path,
			Language:       backend.LanguageID(language),
			WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted},
			Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven},
		},
		Path:               path,
		FormatSelectedFile: formatSelected,
		OrganizeImports:    organizeImports,
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
	return service.Rename(ctx, backend.RenameRequest{
		Project:           project,
		Symbol:            request.Symbol,
		To:                request.To,
		OrganizeImports:   effectiveAutoOrganize(cc, request.OrganizeImports),
		DeferFormatting:   cc.DeferVerification,
		DeferVerification: cc.DeferVerification,
	})
}

func verifyRun(ctx context.Context, cc CallContext, request VerifyReq) (VerifyRes, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	project := effectiveProject(cc, request.Project)
	path := request.Path
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	formatted := false
	if project.Language != backend.LanguageJava && (project.Language != backend.LanguageAuto || !strings.HasSuffix(strings.ToLower(path), ".java")) {
		var err error
		formatted, err = gofmtListsFiles(ctx, project.RootDir, path)
		if err != nil {
			return VerifyRes{}, err
		}
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	result, err := service.Verify(ctx, backend.VerifyRequest{Project: project, Path: path, FormatSelectedFile: request.FormatSelectedFile, OrganizeImports: request.OrganizeImports})
	if err != nil {
		return VerifyRes{}, err
	}
	return VerifyRes{Diagnostics: result.Diagnostics, Formatted: formatted}, nil
}

// gofmtListsFiles reports whether gofmt would reformat sources under path.
func gofmtListsFiles(ctx context.Context, rootDir, path string) (bool, error) {
	// #nosec G204 -- canonical formatter invocation with caller-selected path.
	cmd := exec.CommandContext(ctx, "gofmt", "-l", path)
	if rootDir != "" {
		cmd.Dir = rootDir
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("gofmt -l %s: %w", path, err)
	}
	return len(bytes.TrimSpace(output.Bytes())) > 0, nil
}

func formatLookup(result *backend.LookupResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format lookup result: %w", err)
	}
	return string(data), nil
}

func formatRename(result *backend.RenameResult) (string, error) {
	if result == nil || result.Lookup == nil {
		return "Successfully renamed symbol.", nil
	}
	return fmt.Sprintf("Successfully renamed %s.", result.Lookup.Symbol), nil
}

func formatVerify(result VerifyRes) (string, error) {
	if len(result.Diagnostics) == 0 {
		return "Verification clean: 0 diagnostics.", nil
	}
	messages := make([]string, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	return fmt.Sprintf("Verification: %d diagnostics:\n%s", len(messages), strings.Join(messages, "\n")), nil
}

func lookupDef() Def[LookupReq, *backend.LookupResult] {
	run := lookupRun
	return Def[LookupReq, *backend.LookupResult]{
		Key:     capability.OpLookup,
		Summary: "Use this tool instead of grep, text search, or line counting when locating a named symbol in Go or a selected trusted Rust, Java, Scala, or explicitly standalone Haskell source file. Omit file for a Go workspace-wide symbol search when its owning file is unknown. Rust, Java, Scala, and Haskell lookup are read-only and require explicit workspace trust; standalone Haskell rejects project markers and requires preinstalled GHC and matching HLS.",
		Params:  lookupParams,
		Level:   LevelSymbol,
		// The only read-only operation: lookup never mutates the workspace.
		// Every other operation (including verify, which reformats) is refactoring.
		ReadOnly: true,
		CLIName:  "lookup",
		MCPName:  "semantic_lookup",
		Parse:    parseLookup,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, LookupReq) (*backend.LookupResult, error){
			backend.LanguageGo:      run,
			backend.LanguageRust:    run,
			backend.LanguageJava:    run,
			backend.LanguageScala:   run,
			backend.LanguageHaskell: run,
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
	return Def[RenameReq, *backend.RenameResult]{
		Key:       capability.OpRename,
		Summary:   "Use this tool instead of text search-and-replace when renaming a symbol and its references through the selected language backend. Go supports workspace rename; trusted Rust and Java support selected-file language-server rename. Other languages may be lookup-only." + automaticVerificationGuidance,
		Params:    renameParams,
		Level:     LevelSymbol,
		CLIName:   "rename",
		MCPName:   "semantic_rename",
		Batchable: true,
		Parse:     parseRename,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, RenameReq) (*backend.RenameResult, error){
			backend.LanguageGo:   run,
			backend.LanguageRust: run,
			backend.LanguageJava: run,
		},
		Format: formatRename,
		ExampleRaw: map[string]any{
			"symbol":                "Server.Start",
			"to":                    "Serve",
			"file":                  wireExampleFile,
			wireAutoOrganizeImports: true,
		},
	}
}

func verifyDef() Def[VerifyReq, VerifyRes] {
	return Def[VerifyReq, VerifyRes]{
		Key:     capability.OpVerify,
		Summary: "Use this tool instead of shelling out to formatters or ad hoc diagnostic commands when explicitly checking or formatting supported Go sources or bounded trusted Java files. It may write formatting/import changes; standalone edits already return diagnostics, so avoid redundant verification. Does not run Maven or Gradle.",
		Params:  verifyParams,
		Level:   LevelBuild,
		CLIName: "verify",
		MCPName: "semantic_verify",
		Parse:   parseVerify,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, VerifyReq) (VerifyRes, error){
			backend.LanguageGo:   verifyRun,
			backend.LanguageJava: verifyRun,
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
	if err := Register(registry, lookupDef()); err != nil {
		panic(err)
	}
	if err := Register(registry, renameDef()); err != nil {
		panic(err)
	}
	if err := Register(registry, verifyDef()); err != nil {
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
	return registry
}
