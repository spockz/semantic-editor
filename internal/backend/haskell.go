// Package backend keeps standalone Haskell lookup bounded to one trusted HLS document session.
package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"semedit/internal/lsp"
)

var (
	// ErrHaskellFileRequired indicates a selected source file is not a supported .hs file.
	ErrHaskellFileRequired = errors.New("a selected Haskell .hs file is required")
	// ErrHaskellStandaloneRequired indicates explicit standalone mode was omitted.
	ErrHaskellStandaloneRequired = errors.New("explicit standalone Haskell lookup is required")
	// ErrHaskellProjectUnsupported indicates standalone mode found a project marker.
	ErrHaskellProjectUnsupported = errors.New("haskell project markers are unsupported for standalone lookup")
	// ErrHaskellFileOutsideWorkspace indicates a selected source is outside the root.
	ErrHaskellFileOutsideWorkspace = errors.New("haskell file is outside the workspace root")
	// ErrHaskellToolchainUnavailable indicates required preinstalled binaries are unavailable.
	ErrHaskellToolchainUnavailable = errors.New("preinstalled GHC and Haskell Language Server are required")
	// ErrHaskellVersionMismatch indicates GHC and HLS probes disagree.
	ErrHaskellVersionMismatch = errors.New("haskell GHC and HLS versions do not match")
	// ErrHaskellVersionUnknown indicates version probes lacked an exact match.
	ErrHaskellVersionUnknown = errors.New("haskell GHC/HLS version could not be established")
	// ErrHaskellMalformedResponse indicates an invalid hierarchical response.
	ErrHaskellMalformedResponse = errors.New("malformed Haskell document symbols response")
	// ErrHaskellUnsupportedResponse indicates a flat or unsupported response shape.
	ErrHaskellUnsupportedResponse = errors.New("unsupported Haskell document symbols response")
	// ErrHaskellSessionConflict indicates a second root was requested for one session.
	ErrHaskellSessionConflict = errors.New("another Haskell language session is active")
	// ErrHaskellSymbolNotFound indicates no exact hierarchical candidate matched.
	ErrHaskellSymbolNotFound = errors.New("haskell symbol not found")
)

// HaskellError identifies a failure in the standalone Haskell lookup adapter.
type HaskellError struct {
	Op        string
	File      string
	Symbol    string
	Workspace string
	Err       error
}

func (e *HaskellError) Error() string {
	parts := make([]string, 0, 4)
	if e.Op != "" {
		parts = append(parts, e.Op)
	}
	if e.Symbol != "" {
		parts = append(parts, e.Symbol)
	}
	if e.File != "" {
		parts = append(parts, e.File)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *HaskellError) Unwrap() error { return e.Err }

// HaskellConfig contains only explicit, preinstalled Haskell tool settings.
type HaskellConfig struct {
	Standalone bool   `json:"standalone,omitempty"`
	GHCBin     string `json:"ghc_bin,omitempty"`
	HLSBin     string `json:"hls_bin,omitempty"`
	GHCVersion string `json:"ghc_version,omitempty"`
	HLSVersion string `json:"hls_version,omitempty"`
	workingDir string
}

// HaskellSession is the small part of an HLS session needed by lookup.
type HaskellSession interface {
	Request(context.Context, string, any) (json.RawMessage, error)
	Notify(context.Context, string, any) error
	Close() error
}

// HaskellSessionFactory allows hermetic tests to replace the process-backed session.
type HaskellSessionFactory any

type haskellSessionFactory func(context.Context, string, HaskellConfig) (HaskellSession, error)

// HaskellBackendOption configures a HaskellBackend.
type HaskellBackendOption func(*HaskellBackend)

// WithHaskellSessionFactory injects a fake or managed HLS transport.
func WithHaskellSessionFactory(factory HaskellSessionFactory) HaskellBackendOption {
	return func(backend *HaskellBackend) {
		switch typed := factory.(type) {
		case func(context.Context, string, HaskellConfig) (HaskellSession, error):
			backend.factory = haskellSessionFactory(typed)
		case func(context.Context, string) (HaskellSession, error):
			backend.factory = haskellSessionFactory(func(ctx context.Context, root string, _ HaskellConfig) (HaskellSession, error) {
				return typed(ctx, root)
			})
		default:
			backend.factory = nil
		}
	}
}

// NewHaskellBackend constructs the trusted standalone HLS lookup adapter.
func NewHaskellBackend(options ...HaskellBackendOption) *HaskellBackend {
	backend := &HaskellBackend{factory: defaultHaskellSessionFactory}
	for _, option := range options {
		if option != nil {
			option(backend)
		}
	}
	return backend
}

// NewHaskellBackendWithFactory constructs a Haskell backend with an injected transport.
func NewHaskellBackendWithFactory(factory HaskellSessionFactory) *HaskellBackend {
	return NewHaskellBackend(WithHaskellSessionFactory(factory))
}

// HaskellBackend provides standalone, file-scoped, read-only symbol lookup through HLS.
type HaskellBackend struct {
	mu      sync.Mutex
	factory haskellSessionFactory
	root    string
	session HaskellSession
}

// TrustedWorkspaceRoot discovers the workspace used for trust comparison.
func (b *HaskellBackend) TrustedWorkspaceRoot(project ProjectContext) (string, error) {
	return haskellWorkspaceRoot(project)
}

// Language returns the Haskell language identifier.
func (*HaskellBackend) Language() LanguageID { return LanguageHaskell }

// Capabilities declares Haskell's trusted read-only lookup capability.
func (*HaskellBackend) Capabilities() Capabilities {
	return NewCapabilitiesRequiringWorkspaceTrust(OperationLookup)
}

// CapabilityMatrix returns the declarative documentation matrix for the Haskell backend.
func (*HaskellBackend) CapabilityMatrix() LanguageMatrix {
	return LanguageMatrix{
		Language:    "haskell",
		DisplayName: "Haskell",
		Maturity:    "Read-only preview",
		Operations: map[string]OpCapability{
			"lookup": {
				Supported:    true,
				Description:  "Explicit standalone .hs hierarchical symbol lookup through a trusted, preinstalled Haskell Language Server session using UTF-16 LSP positions.",
				CLICommand:   "semedit lookup --language haskell --haskell-standalone --file <path.hs> --symbol <sym> --ghc-bin <path> --hls-bin <path>",
				MCPTool:      "resolve_symbol_location",
				PlacementKey: false,
			},
		},
		Limitations: []Constraint{
			{
				Title:       "Lookup Only",
				Description: "Haskell rename, formatting, imports, verification, compilation, diagnostics, and structural edits are unavailable.",
				Severity:    "error",
			},
			{
				Title:       "Explicit Standalone Trust",
				Description: "Lookup requires --haskell-standalone (or standalone_haskell=true), a selected .hs file, explicit workspace trust, preinstalled GHC, and a matching preinstalled haskell-language-server-wrapper; hie.yaml, stack.yaml, cabal.project, *.cabal, and package.yaml project markers are rejected.",
				Severity:    "error",
			},
			{
				Title:       "Hierarchical Document Symbols",
				Description: "Only module, top-level values, types, classes, constructors, fields, and instances returned hierarchically by HLS are resolved; locals, pattern synonyms, duplicate record fields, reexports, generated or Template Haskell symbols, and malformed-source fallback are not promised.",
				Severity:    "info",
			},
		},
	}
}

// Close terminates the managed HLS server, if one is active.
func (b *HaskellBackend) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session == nil {
		return nil
	}
	err := b.session.Close()
	b.session = nil
	b.root = ""
	return err
}

// Lookup resolves one exact hierarchical symbol in the selected standalone Haskell file.
func (b *HaskellBackend) Lookup(ctx context.Context, project ProjectContext, query string) (*LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveHaskellProject(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &WorkspaceTrustError{Operation: OperationLookup, Language: LanguageHaskell, Workspace: root}
	}
	if strings.TrimSpace(query) == "" {
		return nil, &HaskellError{Op: "lookup", File: file, Symbol: query, Err: ErrHaskellMalformedResponse}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil && b.root != root {
		return nil, &HaskellError{Op: "lookup", Workspace: root, Err: ErrHaskellSessionConflict}
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &HaskellError{Op: "start", Workspace: root, Err: ErrHaskellToolchainUnavailable}
		}
		config := project.Haskell
		if project.HaskellStandalone {
			config.Standalone = true
		}
		config.workingDir = filepath.Dir(file)
		session, factoryErr := b.factory(ctx, root, config)
		if factoryErr != nil {
			return nil, &HaskellError{Op: "start", Workspace: root, Err: factoryErr}
		}
		if session == nil {
			return nil, &HaskellError{Op: "start", Workspace: root, Err: ErrHaskellToolchainUnavailable}
		}
		b.session, b.root = session, root
		if err := initializeHaskellSession(ctx, session, root); err != nil {
			_ = session.Close()
			b.session, b.root = nil, ""
			return nil, &HaskellError{Op: "initialize", Workspace: root, Err: err}
		}
	}
	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": fileURI(file), "languageId": "haskell", "version": 1, "text": string(source),
	}}); err != nil {
		return nil, &HaskellError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": fileURI(file)}})
	if err != nil {
		return nil, &HaskellError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	symbols, err := decodeHaskellDocumentSymbols(raw, root, file, source)
	if err != nil {
		return nil, &HaskellError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	return selectHaskellSymbol(query, root, file, source, symbols)
}

// Rename is intentionally unavailable for the Haskell lookup-only slice.
func (*HaskellBackend) Rename(context.Context, RenameRequest) (*RenameResult, error) {
	return nil, &Error{Operation: OperationRename, Language: LanguageHaskell, Err: ErrUnsupportedOperation}
}

// Verify is intentionally unavailable for the Haskell lookup-only slice.
func (*HaskellBackend) Verify(context.Context, ProjectContext, string) ([]Diagnostic, error) {
	return nil, &Error{Operation: OperationVerify, Language: LanguageHaskell, Err: ErrUnsupportedOperation}
}

var haskellSettings = map[string]any{
	"haskell": map[string]any{
		"checkProject":      false,
		"checkParents":      "NeverCheck",
		"componentsLoading": "single",
		"plugin": map[string]any{
			"ghcide-code-actions-fill-holes":      map[string]any{"globalOn": false},
			"ghcide-code-actions-type-signatures": map[string]any{"globalOn": false},
			"ghcide-code-actions-bindings":        map[string]any{"globalOn": false},
			"ghcide-code-actions-imports-exports": map[string]any{"globalOn": false},
			"ghcide-type-lenses":                  map[string]any{"globalOn": false},
			"eval":                                map[string]any{"globalOn": false},
			"hlint":                               map[string]any{"globalOn": false},
			"rename":                              map[string]any{"globalOn": false},
			"retrie":                              map[string]any{"globalOn": false},
			"tactics":                             map[string]any{"globalOn": false},
			"formatting":                          map[string]any{"globalOn": false},
		},
	},
}

func initializeHaskellSession(ctx context.Context, session HaskellSession, root string) error {
	params := map[string]any{
		"processId":        nil,
		"rootUri":          fileURI(root),
		"workspaceFolders": []map[string]string{{"uri": fileURI(root), "name": filepath.Base(root)}},
		"capabilities": map[string]any{
			"general":      map[string]any{"positionEncodings": []string{"utf-16"}},
			"textDocument": map[string]any{"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true}},
			"workspace":    map[string]any{"workspaceFolders": true, "configuration": true},
		},
		"initializationOptions": haskellSettings,
		"settings":              haskellSettings,
	}
	if _, err := session.Request(ctx, "initialize", params); err != nil {
		return err
	}
	if err := session.Notify(ctx, "initialized", map[string]any{}); err != nil {
		return err
	}
	return session.Notify(ctx, "workspace/didChangeConfiguration", map[string]any{"settings": haskellSettings})
}

func rejectHaskellServerRequest(_ context.Context, _ lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
	return nil, &lsp.ErrorObject{Code: -32601, Message: "server request is not supported by standalone Haskell lookup"}
}

type haskellToolchain struct {
	ghc string
	hls string
}

func defaultHaskellSessionFactory(ctx context.Context, root string, config HaskellConfig) (HaskellSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tools, err := probeHaskellToolchain(ctx, root, config)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, tools.hls, "--lsp") // #nosec G204 -- path is preinstalled and explicitly resolved.
	cmd.Dir = root
	if config.workingDir != "" {
		cmd.Dir = config.workingDir
	}
	client, err := lsp.NewProcessClient(cmd, lsp.WithRequestHandler(rejectHaskellServerRequest))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func probeHaskellToolchain(ctx context.Context, root string, config HaskellConfig) (haskellToolchain, error) {
	ghc, err := resolveHaskellExecutable(config.GHCBin, "ghc")
	if err != nil {
		return haskellToolchain{}, err
	}
	hls, err := resolveHaskellExecutable(config.HLSBin, "haskell-language-server-wrapper")
	if err != nil {
		return haskellToolchain{}, err
	}
	workingDir := root
	if config.workingDir != "" {
		workingDir = config.workingDir
	}
	ghcOutput, err := runHaskellProbe(ctx, workingDir, ghc, "--numeric-version")
	if err != nil {
		return haskellToolchain{}, fmt.Errorf("%w: ghc --numeric-version: %w", ErrHaskellToolchainUnavailable, err)
	}
	ghcVersion := parseGHCVersion(ghcOutput)
	if ghcVersion == "" {
		return haskellToolchain{}, fmt.Errorf("%w: GHC version", ErrHaskellVersionUnknown)
	}
	if config.GHCVersion != "" && normalizeVersion(config.GHCVersion) != ghcVersion {
		return haskellToolchain{}, fmt.Errorf("%w: configured GHC %s, found %s", ErrHaskellVersionMismatch, config.GHCVersion, ghcVersion)
	}
	hlsOutput, err := runHaskellProbe(ctx, workingDir, hls, "--probe-tools")
	if err != nil {
		return haskellToolchain{}, fmt.Errorf("%w: HLS --probe-tools: %w", ErrHaskellToolchainUnavailable, err)
	}
	hlsVersion, reportedGHC := parseHLSProbe(hlsOutput)
	if hlsVersion == "" || reportedGHC == "" {
		return haskellToolchain{}, fmt.Errorf("%w: HLS probe output did not identify HLS and GHC versions", ErrHaskellVersionUnknown)
	}
	if config.HLSVersion != "" && normalizeVersion(config.HLSVersion) != hlsVersion {
		return haskellToolchain{}, fmt.Errorf("%w: configured HLS %s, found %s", ErrHaskellVersionMismatch, config.HLSVersion, hlsVersion)
	}
	if reportedGHC != ghcVersion {
		return haskellToolchain{}, fmt.Errorf("%w: GHC %s, HLS supports GHC %s", ErrHaskellVersionMismatch, ghcVersion, reportedGHC)
	}
	return haskellToolchain{ghc: ghc, hls: hls}, nil
}

func resolveHaskellExecutable(configured, name string) (string, error) {
	if configured == "" {
		path, err := exec.LookPath(name)
		if err != nil {
			return "", fmt.Errorf("%w: %s: %w", ErrHaskellToolchainUnavailable, name, err)
		}
		return path, nil
	}
	path, err := filepath.Abs(configured)
	if err != nil {
		return "", fmt.Errorf("%w: invalid %s path: %w", ErrHaskellToolchainUnavailable, name, err)
	}
	info, statErr := os.Stat(path)
	if statErr != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%w: invalid %s path %q", ErrHaskellToolchainUnavailable, name, configured)
	}
	return path, nil
}

func runHaskellProbe(ctx context.Context, root, executable string, arg string) (string, error) {
	cmd := exec.CommandContext(ctx, executable, arg) // #nosec G204 -- executable is resolved from preinstalled tooling.
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func normalizeVersion(version string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(version), "v"))
}

func parseGHCVersion(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+$`).MatchString(line) {
			return line
		}
	}
	return ""
}

var hlsProbePattern = regexp.MustCompile(`(?i)haskell-language-server\s+version:\s*([0-9]+(?:\.[0-9]+)+).*?\(GHC:\s*([0-9]+(?:\.[0-9]+)+)\)`)

func parseHLSProbe(output string) (string, string) {
	match := hlsProbePattern.FindStringSubmatch(output)
	if len(match) != 3 {
		return "", ""
	}
	return normalizeVersion(match[1]), normalizeVersion(match[2])
}

func resolveHaskellProject(project ProjectContext) (string, string, []byte, error) {
	if strings.TrimSpace(project.File) == "" || !strings.EqualFold(filepath.Ext(project.File), ".hs") || strings.HasSuffix(strings.ToLower(project.File), ".hs-boot") {
		return "", "", nil, &HaskellError{Op: "project", File: project.File, Err: ErrHaskellFileRequired}
	}
	if !project.Haskell.Standalone && !project.HaskellStandalone {
		return "", "", nil, &HaskellError{Op: "project", File: project.File, Err: ErrHaskellStandaloneRequired}
	}
	base := project.RootDir
	if base == "" {
		base = filepath.Dir(project.File)
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(base, file)
	}
	file = CanonicalWorkspaceRoot(file)
	root := CanonicalWorkspaceRoot(base)
	if !pathWithin(root, file) {
		return "", "", nil, &HaskellError{Op: "project", File: file, Workspace: root, Err: ErrHaskellFileOutsideWorkspace}
	}
	if marker := hasHaskellProjectMarker(root, filepath.Dir(file)); marker != "" {
		return "", "", nil, &HaskellError{Op: "project", File: file, Workspace: root, Err: fmt.Errorf("%w: %s", ErrHaskellProjectUnsupported, marker)}
	}
	source, err := os.ReadFile(file) // #nosec G304 -- file is explicitly selected by the caller.
	if err != nil {
		return "", "", nil, &HaskellError{Op: "read", File: file, Workspace: root, Err: err}
	}
	return root, file, source, nil
}

func haskellWorkspaceRoot(project ProjectContext) (string, error) {
	if project.RootDir != "" {
		return CanonicalWorkspaceRoot(project.RootDir), nil
	}
	if strings.TrimSpace(project.File) == "" {
		return "", ErrHaskellFileRequired
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(".", file)
	}
	return CanonicalWorkspaceRoot(filepath.Dir(CanonicalWorkspaceRoot(file))), nil
}

func hasHaskellProjectMarker(root, start string) string {
	dir := CanonicalWorkspaceRoot(start)
	root = CanonicalWorkspaceRoot(root)
	for pathWithin(root, dir) {
		for _, name := range []string{"hie.yaml", "stack.yaml", "cabal.project", "package.yaml"} {
			if fileExists(filepath.Join(dir, name)) {
				return filepath.Join(dir, name)
			}
		}
		matches, _ := filepath.Glob(filepath.Join(dir, "*.cabal"))
		if len(matches) != 0 {
			return matches[0]
		}
		if dir == root {
			break
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

type haskellDocumentSymbol struct {
	Name           string
	Kind           int
	Range          haskellRange
	SelectionRange haskellRange
	Children       []haskellDocumentSymbol
	URI            string
}
type haskellRange struct{ Start, End haskellPosition }
type haskellPosition struct{ Line, Character int }

func decodeHaskellDocumentSymbols(raw json.RawMessage, root, file string, source []byte) ([]haskellDocumentSymbol, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHaskellMalformedResponse, err)
	}
	result := make([]haskellDocumentSymbol, 0, len(values))
	for _, value := range values {
		symbol, err := decodeHaskellDocumentSymbol(value, root, file, source)
		if err != nil {
			return nil, err
		}
		result = append(result, symbol)
	}
	return result, nil
}

func decodeHaskellDocumentSymbol(raw json.RawMessage, root, file string, source []byte) (haskellDocumentSymbol, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return haskellDocumentSymbol{}, fmt.Errorf("%w: symbol is not an object", ErrHaskellMalformedResponse)
	}
	if _, flat := object["location"]; flat {
		return haskellDocumentSymbol{}, ErrHaskellUnsupportedResponse
	}
	var symbol haskellDocumentSymbol
	if err := json.Unmarshal(object["name"], &symbol.Name); err != nil || symbol.Name == "" {
		return haskellDocumentSymbol{}, fmt.Errorf("%w: missing name", ErrHaskellMalformedResponse)
	}
	if err := json.Unmarshal(object["kind"], &symbol.Kind); err != nil {
		return haskellDocumentSymbol{}, fmt.Errorf("%w: invalid kind", ErrHaskellMalformedResponse)
	}
	if err := json.Unmarshal(object["range"], &symbol.Range); err != nil || !validHaskellRange(symbol.Range, source) {
		return haskellDocumentSymbol{}, fmt.Errorf("%w: invalid range", ErrHaskellMalformedResponse)
	}
	if err := json.Unmarshal(object["selectionRange"], &symbol.SelectionRange); err != nil || !validHaskellRange(symbol.SelectionRange, source) {
		return haskellDocumentSymbol{}, fmt.Errorf("%w: invalid selection range", ErrHaskellMalformedResponse)
	}
	if uriRaw, ok := object["uri"]; ok {
		if err := json.Unmarshal(uriRaw, &symbol.URI); err != nil {
			return haskellDocumentSymbol{}, fmt.Errorf("%w: invalid uri", ErrHaskellMalformedResponse)
		}
		if symbol.URI != "" {
			uriPath, err := filePathFromURI(symbol.URI)
			if err != nil || !pathWithin(root, uriPath) || CanonicalWorkspaceRoot(uriPath) != CanonicalWorkspaceRoot(file) {
				return haskellDocumentSymbol{}, fmt.Errorf("%w: symbol uri is outside selected file", ErrHaskellMalformedResponse)
			}
		}
	}
	if childrenRaw, ok := object["children"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return haskellDocumentSymbol{}, fmt.Errorf("%w: invalid children", ErrHaskellMalformedResponse)
		}
		for _, childRaw := range children {
			child, err := decodeHaskellDocumentSymbol(childRaw, root, file, source)
			if err != nil {
				return haskellDocumentSymbol{}, err
			}
			symbol.Children = append(symbol.Children, child)
		}
	}
	return symbol, nil
}

func validHaskellRange(r haskellRange, source []byte) bool {
	if r.Start.Line < 0 || r.Start.Character < 0 || r.End.Line < 0 || r.End.Character < 0 {
		return false
	}
	start, startErr := haskellByteOffset(source, r.Start)
	end, endErr := haskellByteOffset(source, r.End)
	return startErr == nil && endErr == nil && start <= end
}

func haskellKindName(kind int, name string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "instance ") {
		return "instance"
	}
	switch kind {
	case 2:
		return "module"
	case 5, 10, 26:
		return "type"
	case 11:
		return "class"
	case 6:
		return "value"
	case 7, 8:
		return "field"
	case 9:
		return "constructor"
	case 12, 13, 14:
		return "value"
	case 19, 23:
		return "instance"
	default:
		return ""
	}
}

func selectHaskellSymbol(query, root, file string, source []byte, symbols []haskellDocumentSymbol) (*LookupResult, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(query), `"'`), ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, &HaskellError{Op: "lookup", Symbol: query, Err: ErrHaskellMalformedResponse}
		}
	}
	type match struct {
		symbol haskellDocumentSymbol
		path   []string
	}
	matches := make([]match, 0)
	var visit func([]string, string, []haskellDocumentSymbol)
	visit = func(parent []string, parentKind string, items []haskellDocumentSymbol) {
		for _, item := range items {
			path := append(append([]string(nil), parent...), strings.Split(item.Name, ".")...)
			if haskellKindName(item.Kind, item.Name) != "" && len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], ".") == strings.Join(parts, ".") {
				matches = append(matches, match{item, path})
			}
			if parentKind != "value" && parentKind != "field" {
				visit(path, haskellKindName(item.Kind, item.Name), item.Children)
			}
		}
	}
	visit(nil, "", symbols)
	if len(matches) == 0 {
		return nil, &HaskellError{Op: "lookup", File: file, Symbol: query, Err: ErrHaskellSymbolNotFound}
	}
	converted := make([]*SymbolCandidate, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, haskellCandidate(root, file, source, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &LookupResult{Symbol: query, File: filepath.Clean(file), Ambiguous: true, Candidates: converted}, nil
	}
	candidate := converted[0]
	return &LookupResult{Symbol: candidate.QualifiedName, File: candidate.File, Line: candidate.Line, Column: candidate.Column, Offset: candidate.Offset, Kind: candidate.Kind, Receiver: candidate.Receiver, Location: candidate.Location}, nil
}

func haskellCandidate(root, file string, source []byte, symbol haskellDocumentSymbol, path []string) *SymbolCandidate {
	start := symbol.SelectionRange.Start
	offset, _ := haskellByteOffset(source, start)
	receiver := ""
	if len(path) > 1 {
		receiver = strings.Join(path[:len(path)-1], ".")
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		relative = file
	}
	location := SourceLocation{URI: fileURI(file), Range: Range{Start: Position(start), End: Position(symbol.SelectionRange.End)}}
	selection := Range{Start: Position(symbol.SelectionRange.Start), End: Position(symbol.SelectionRange.End)}
	return &SymbolCandidate{Name: symbol.Name, Receiver: receiver, QualifiedName: strings.Join(path, "."), Kind: haskellKindName(symbol.Kind, symbol.Name), File: relative, Line: start.Line + 1, Column: start.Character + 1, Offset: offset, SelectionRange: &selection, Location: location}
}

func haskellByteOffset(source []byte, position haskellPosition) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, errors.New("negative position")
	}
	line, start := 0, 0
	for i, b := range source {
		if b == '\n' {
			if line == position.Line {
				offset, err := haskellCharacterOffset(source[start:i], position.Character)
				return start + offset, err
			}
			line++
			start = i + 1
		}
	}
	if line != position.Line {
		return 0, errors.New("line outside source")
	}
	offset, err := haskellCharacterOffset(source[start:], position.Character)
	return start + offset, err
}

func haskellCharacterOffset(line []byte, character int) (int, error) {
	units := 0
	for offset := 0; offset < len(line); {
		runeValue, size := utf8.DecodeRune(line[offset:])
		if runeValue == utf8.RuneError && size == 1 {
			return 0, errors.New("invalid UTF-8 source")
		}
		runeUnits := 1
		if runeValue > 0xffff {
			runeUnits = 2
		}
		if units == character {
			return offset, nil
		}
		if units < character && character < units+runeUnits {
			return 0, errors.New("UTF-16 position splits a code point")
		}
		units += runeUnits
		offset += size
	}
	if units == character {
		return len(line), nil
	}
	return 0, errors.New("character outside source")
}
