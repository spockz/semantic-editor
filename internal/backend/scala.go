// Package backend keeps Scala lookup read-only and file-scoped behind one trusted Metals session.
package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"semedit/internal/backend/pathutil"
	"semedit/internal/lsp"
)

var (
	// ErrScalaFileRequired indicates that lookup did not receive a selected Scala source file.
	ErrScalaFileRequired = errors.New("a selected Scala .scala file is required")
	// ErrScalaWorkspaceRequired indicates that an explicit root is required for a project file.
	ErrScalaWorkspaceRequired = errors.New("scala workspace root is required")
	// ErrScalaProjectMarkersRequireExplicitRoot prevents inferring a build workspace.
	ErrScalaProjectMarkersRequireExplicitRoot = errors.New("scala project markers require an explicit workspace root")
	// ErrScalaFileOutsideWorkspace indicates that the selected source is outside the root.
	ErrScalaFileOutsideWorkspace = errors.New("scala file is outside the workspace root")
	// ErrMetalsHomeRequired indicates that no direct, pinned Metals distribution was supplied.
	ErrMetalsHomeRequired = errors.New("an explicitly configured Metals distribution is required")
	// ErrMetalsUnavailable indicates that the configured server could not be used.
	ErrMetalsUnavailable = errors.New("metals is not installed")
	// ErrMetalsLauncherInvalid indicates that a direct distribution or executable is invalid.
	ErrMetalsLauncherInvalid = errors.New("metals distribution or executable is invalid")
	// ErrScalaJavaRuntimeUnavailable indicates that Java could not be executed for Scala.
	ErrScalaJavaRuntimeUnavailable = errors.New("java runtime is unavailable for scala")
	// ErrScalaJavaRuntimeTooOld indicates that the runtime is older than Java 21.
	ErrScalaJavaRuntimeTooOld = errors.New("java runtime 21 or newer is required for scala")
	// ErrScalaJavaRuntimeVersionMismatch indicates a recorded version disagrees with the executable.
	ErrScalaJavaRuntimeVersionMismatch = errors.New("configured scala java runtime version does not match")
	// ErrScalaMalformedResponse indicates an invalid hierarchical response.
	ErrScalaMalformedResponse = errors.New("malformed Metals document symbols response")
	// ErrScalaUnsupportedResponse indicates a flat or otherwise unsupported response.
	ErrScalaUnsupportedResponse = errors.New("unsupported Metals document symbols response")
	// ErrScalaSessionConflict indicates that another root already owns the session.
	ErrScalaSessionConflict = errors.New("another Scala language session is active")
	// ErrScalaSymbolNotFound indicates that no exact hierarchical candidate matched.
	ErrScalaSymbolNotFound = errors.New("scala symbol not found")
)

// ScalaConfig contains explicit, preinstalled external-tool configuration.
type ScalaConfig struct {
	MetalsHome  string `json:"metals_home,omitempty"`
	MetalsBin   string `json:"metals_bin,omitempty"`
	JavaBin     string `json:"java_bin,omitempty"`
	JavaVersion string `json:"java_version,omitempty"`
}

// ScalaError identifies a failure in the read-only Scala lookup adapter.
type ScalaError struct {
	Op        string
	File      string
	Symbol    string
	Workspace string
	Err       error
}

func (e *ScalaError) Error() string {
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
func (e *ScalaError) Unwrap() error { return e.Err }

// ScalaSession is the small part of a managed Metals session needed by lookup.
type ScalaSession interface {
	Request(context.Context, string, any) (json.RawMessage, error)
	Notify(context.Context, string, any) error
	Close() error
}

// ScalaSessionFactory allows hermetic tests to replace the process-backed session.
// Implementations may accept either (context.Context, string) or
// (context.Context, string, ScalaConfig); the latter receives request settings.
type ScalaSessionFactory any

type scalaSessionFactory func(context.Context, string, ScalaConfig) (ScalaSession, error)

// ScalaBackendOption configures a ScalaBackend.
type ScalaBackendOption func(*ScalaBackend)

// WithScalaSessionFactory injects a fake or managed LSP transport for tests.
func WithScalaSessionFactory(factory ScalaSessionFactory) ScalaBackendOption {
	return func(backend *ScalaBackend) {
		switch typed := factory.(type) {
		case func(context.Context, string, ScalaConfig) (ScalaSession, error):
			backend.factory = scalaSessionFactory(typed)
		case func(context.Context, string) (ScalaSession, error):
			backend.factory = scalaSessionFactory(func(ctx context.Context, root string, _ ScalaConfig) (ScalaSession, error) { return typed(ctx, root) })
		default:
			backend.factory = nil
		}
	}
}

// ScalaBackend provides trusted, file-scoped, read-only lookup through Metals.
type ScalaBackend struct {
	mu      sync.Mutex
	factory scalaSessionFactory
	root    string
	session ScalaSession
}

// TrustedWorkspaceRoot discovers the workspace used for trust comparison.
func (b *ScalaBackend) TrustedWorkspaceRoot(project ProjectContext) (string, error) {
	return scalaWorkspaceRoot(project)
}

// NewScalaBackend constructs the managed JDT LS lookup adapter.
func NewScalaBackend(options ...ScalaBackendOption) *ScalaBackend {
	backend := &ScalaBackend{factory: defaultScalaSessionFactory}
	for _, option := range options {
		if option != nil {
			option(backend)
		}
	}
	return backend
}

// NewScalaBackendWithFactory constructs a Scala backend with an injected session factory.
func NewScalaBackendWithFactory(factory ScalaSessionFactory) *ScalaBackend {
	return NewScalaBackend(WithScalaSessionFactory(factory))
}

// Language returns the Scala language identifier.
func (*ScalaBackend) Language() LanguageID { return LanguageScala }

// Capabilities declares Scala's trusted read-only lookup capability.
func (*ScalaBackend) Capabilities() Capabilities {
	return NewCapabilitiesRequiringWorkspaceTrust(OperationLookup)
}

// CapabilityMatrix returns the declarative documentation matrix for the Scala backend.
func (*ScalaBackend) CapabilityMatrix() LanguageMatrix {
	return LanguageMatrix{
		Language:    "scala",
		DisplayName: "Scala",
		Maturity:    "Read-only preview",
		Operations: map[string]OpCapability{
			"lookup": {
				Supported:    true,
				Description:  "File-scoped hierarchical Scala symbol lookup through a trusted, pinned Metals session using UTF-16 LSP positions.",
				CLICommand:   "semedit lookup --language scala --file <path.scala> --symbol <sym> --metals-bin <path> --java-bin <path> --java-version <major>",
				MCPTool:      "resolve_symbol_location",
				PlacementKey: false,
			},
		},
		Limitations: []Constraint{
			{
				Title:       "Lookup Only",
				Description: "Scala rename, formatting, imports, verification, build import, and structural edits are unavailable.",
				Severity:    "error",
			},
			{
				Title:       "Trusted Explicit Tools",
				Description: "Lookup requires a selected .scala file, an explicit workspace root for project markers, explicit workspace trust, a pinned preinstalled Metals distribution, and recorded Java 21 or newer; no build tool is invoked.",
				Severity:    "error",
			},
			{
				Title:       "Hierarchical Document Symbols",
				Description: "Only exact hierarchical classes, objects, traits, enums, methods, fields, and nested types are resolved; overload signatures, givens, extensions, package objects, generated symbols, and cross-file SemanticDB search are not promised.",
				Severity:    "info",
			},
		},
	}
}

// Close terminates the managed JDT LS server, if one is active.
func (b *ScalaBackend) Close() error {
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

// Lookup resolves one exact hierarchical symbol in the selected Scala file.
func (b *ScalaBackend) Lookup(ctx context.Context, project ProjectContext, query string) (*LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveScalaProject(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &WorkspaceTrustError{Operation: OperationLookup, Language: LanguageScala, Workspace: root}
	}
	if strings.TrimSpace(query) == "" {
		return nil, &ScalaError{Op: "lookup", File: file, Symbol: query, Err: ErrScalaMalformedResponse}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil && b.root != root {
		return nil, &ScalaError{Op: "lookup", Workspace: root, Err: ErrScalaSessionConflict}
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &ScalaError{Op: "start", Workspace: root, Err: ErrMetalsUnavailable}
		}
		scalaConfig := project.Scala
		if scalaConfig.MetalsHome == "" {
			scalaConfig.MetalsHome = project.MetalsHome
		}
		if scalaConfig.MetalsBin == "" {
			scalaConfig.MetalsBin = project.MetalsBin
		}
		if scalaConfig.JavaBin == "" {
			scalaConfig.JavaBin = project.JavaBin
		}
		if scalaConfig.JavaVersion == "" {
			scalaConfig.JavaVersion = project.JavaVersion
		}
		session, factoryErr := b.factory(ctx, root, scalaConfig)
		if factoryErr != nil {
			return nil, &ScalaError{Op: "start", Workspace: root, Err: factoryErr}
		}
		if session == nil {
			return nil, &ScalaError{Op: "start", Workspace: root, Err: ErrMetalsUnavailable}
		}
		b.session, b.root = session, root
		if err := initializeScalaSession(ctx, session, root); err != nil {
			_ = session.Close()
			b.session, b.root = nil, ""
			return nil, &ScalaError{Op: "initialize", Workspace: root, Err: err}
		}
	}
	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": pathutil.FileURI(file), "languageId": "scala", "version": 1, "text": string(source),
	}}); err != nil {
		return nil, &ScalaError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}})
	if err != nil {
		return nil, &ScalaError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	symbols, err := decodeScalaDocumentSymbols(raw, root, source)
	if err != nil {
		return nil, &ScalaError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	return selectScalaSymbol(query, root, file, source, symbols)
}

// Rename is intentionally unavailable for the Scala lookup-only slice.
func (*ScalaBackend) Rename(context.Context, RenameRequest) (*RenameResult, error) {
	return nil, &Error{Operation: OperationRename, Language: LanguageScala, Err: ErrUnsupportedOperation}
}

// Verify is intentionally unavailable for the Scala lookup-only slice.
func (*ScalaBackend) Verify(context.Context, VerifyRequest) ([]Diagnostic, error) {
	return nil, &Error{Operation: OperationVerify, Language: LanguageScala, Err: ErrUnsupportedOperation}
}

var scalaMetalsSettings = map[string]any{
	"metals.autoImportBuild": false,
	"metals": map[string]any{
		"autoImportBuild":   false,
		"bsp":               map[string]any{"enabled": false},
		"bspSwitch":         false,
		"bloop":             map[string]any{"enabled": false},
		"statusBarProvider": "off",
	},
}

func rejectScalaServerRequest(_ context.Context, request lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
	// Metals may ask the client to import a build or create configuration. This
	// slice is file-scoped and has no safe response for either operation.
	method := strings.ToLower(request.Method)
	params := strings.ToLower(string(request.Params))
	if strings.Contains(method, "build") || strings.Contains(method, "config") ||
		strings.Contains(params, "build-import") || strings.Contains(params, "buildimport") ||
		strings.Contains(params, "import") || strings.Contains(params, "create") {
		return nil, &lsp.ErrorObject{Code: -32001, Message: "Scala build import and configuration prompts are disabled"}
	}
	return nil, &lsp.ErrorObject{Code: -32601, Message: "server request is not supported by read-only Scala lookup"}
}

func initializeScalaSession(ctx context.Context, session ScalaSession, root string) error {
	params := map[string]any{
		"processId":        nil,
		"rootUri":          pathutil.FileURI(root),
		"workspaceFolders": []map[string]string{{"uri": pathutil.FileURI(root), "name": filepath.Base(root)}},
		"capabilities": map[string]any{
			"general":      map[string]any{"positionEncodings": []string{"utf-16"}},
			"textDocument": map[string]any{"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true}},
			"workspace":    map[string]any{"workspaceFolders": true, "configuration": true},
		},
		"initializationOptions": map[string]any{
			"statusBarProvider": "off",
			"isHttpServer":      false,
			"clientName":        "semedit",
		},
		"settings": scalaMetalsSettings,
	}
	if _, err := session.Request(ctx, "initialize", params); err != nil {
		return err
	}
	if err := session.Notify(ctx, "initialized", map[string]any{}); err != nil {
		return err
	}
	return session.Notify(ctx, "workspace/didChangeConfiguration", map[string]any{"settings": scalaMetalsSettings})
}

func defaultScalaSessionFactory(ctx context.Context, root string, config ScalaConfig) (ScalaSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if config.JavaBin == "" || config.JavaVersion == "" {
		return nil, fmt.Errorf("%w: explicit java_bin and java_version are required", ErrScalaJavaRuntimeUnavailable)
	}
	if config.MetalsBin == "" && config.MetalsHome == "" {
		return nil, ErrMetalsHomeRequired
	}
	javaBin, err := filepath.Abs(config.JavaBin)
	javaInfo, statErr := os.Stat(javaBin)
	if err != nil || statErr != nil || javaInfo.IsDir() || javaInfo.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("%w: invalid java_bin %q", ErrScalaJavaRuntimeUnavailable, config.JavaBin)
	}
	if err := validateScalaJavaRuntime(ctx, javaBin, config.JavaVersion); err != nil {
		return nil, err
	}
	metalsBin, err := resolveMetalsBinary(config)
	if err != nil {
		return nil, err
	}
	dataPath := filepath.Join(root, ".scratch", "metals", stableScalaRootHash(root))
	if err := os.MkdirAll(dataPath, 0o700); err != nil {
		return nil, fmt.Errorf("create Metals data directory: %w", err)
	}
	cmd := exec.CommandContext(ctx, metalsBin, "--stdio", "--no-http-server", "--data", dataPath) // #nosec G204 -- path and fixed arguments are explicitly configured.
	cmd.Dir = root
	client, err := lsp.NewProcessClient(cmd, lsp.WithRequestHandler(rejectScalaServerRequest))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func validateScalaJavaRuntime(ctx context.Context, javaBin, recorded string) error {
	output, err := exec.CommandContext(ctx, javaBin, "-version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScalaJavaRuntimeUnavailable, err)
	}
	major, ok := parseScalaJavaMajor(string(output))
	if !ok {
		return fmt.Errorf("%w: could not parse Java version", ErrScalaJavaRuntimeTooOld)
	}
	if major < 21 {
		return fmt.Errorf("%w: found Java %d", ErrScalaJavaRuntimeTooOld, major)
	}
	recordedMajor := strings.TrimSpace(recorded)
	if dot := strings.IndexByte(recordedMajor, '.'); dot >= 0 {
		recordedMajor = recordedMajor[:dot]
	}
	if recordedMajor != strconv.Itoa(major) {
		return fmt.Errorf("%w: recorded %s, found %d", ErrScalaJavaRuntimeVersionMismatch, recorded, major)
	}
	return nil
}

func parseScalaJavaMajor(output string) (int, bool) {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(strings.ToLower(line), "version") && !strings.Contains(strings.ToLower(line), "openjdk") {
			continue
		}
		for i := 0; i < len(line); i++ {
			if line[i] < '0' || line[i] > '9' {
				continue
			}
			j := i
			for j < len(line) && line[j] >= '0' && line[j] <= '9' {
				j++
			}
			major, err := strconv.Atoi(line[i:j])
			if err == nil {
				if major == 1 && j < len(line) && line[j] == '.' && j+1 < len(line) {
					continue
				}
				return major, true
			}
			i = j
		}
	}
	return 0, false
}

func resolveMetalsBinary(config ScalaConfig) (string, error) {
	if config.MetalsBin != "" {
		path, err := filepath.Abs(config.MetalsBin)
		info, statErr := os.Stat(path)
		if err != nil || statErr != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("%w: invalid metals_bin %q", ErrMetalsLauncherInvalid, config.MetalsBin)
		}
		return path, nil
	}
	info, err := os.Stat(config.MetalsHome)
	if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
		path, absErr := filepath.Abs(config.MetalsHome)
		if absErr == nil {
			return path, nil
		}
	}
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: invalid metals_home %q", ErrMetalsLauncherInvalid, config.MetalsHome)
	}
	for _, name := range []string{"metals", "metals-emacs", "metals-vscode"} {
		candidate := filepath.Join(config.MetalsHome, "bin", name)
		if pathutil.FileExists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%w: no direct Metals executable found", ErrMetalsLauncherInvalid)
}

func stableScalaRootHash(root string) string {
	digest := sha256.Sum256([]byte(CanonicalWorkspaceRoot(root)))
	return hex.EncodeToString(digest[:])[:24]
}

func resolveScalaProject(project ProjectContext) (string, string, []byte, error) {
	if strings.TrimSpace(project.File) == "" || !strings.EqualFold(filepath.Ext(project.File), ".scala") {
		return "", "", nil, &ScalaError{Op: "project", File: project.File, Err: ErrScalaFileRequired}
	}
	base := project.RootDir
	file := project.File
	if base == "" {
		base = "."
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(base, file)
	}
	file = CanonicalWorkspaceRoot(file)
	if project.RootDir == "" {
		if hasScalaProjectMarker(file) {
			return "", "", nil, &ScalaError{Op: "project", File: file, Err: ErrScalaProjectMarkersRequireExplicitRoot}
		}
		base = filepath.Dir(file)
	}
	root := CanonicalWorkspaceRoot(base)
	if !pathutil.PathWithin(root, file) {
		return "", "", nil, &ScalaError{Op: "project", File: file, Workspace: root, Err: ErrScalaFileOutsideWorkspace}
	}
	source, err := os.ReadFile(file) // #nosec G304 -- file is explicitly selected by the caller.
	if err != nil {
		return "", "", nil, &ScalaError{Op: "read", File: file, Workspace: root, Err: err}
	}
	return root, file, source, nil
}

func scalaWorkspaceRoot(project ProjectContext) (string, error) {
	if project.RootDir != "" {
		return CanonicalWorkspaceRoot(project.RootDir), nil
	}
	if project.File == "" {
		return "", ErrScalaFileRequired
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(".", file)
	}
	file = CanonicalWorkspaceRoot(file)
	if hasScalaProjectMarker(file) {
		return "", ErrScalaProjectMarkersRequireExplicitRoot
	}
	return CanonicalWorkspaceRoot(filepath.Dir(file)), nil
}

func hasScalaProjectMarker(file string) bool {
	dir := filepath.Dir(file)
	for {
		for _, marker := range []string{"build.sbt", "build.sc", "pom.xml", "build.gradle", "build.gradle.kts", ".mill-version"} {
			if pathutil.FileExists(filepath.Join(dir, marker)) {
				return true
			}
		}
		if pathutil.FileExists(filepath.Join(dir, "project")) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

type scalaDocumentSymbol struct {
	Name           string
	Kind           int
	Range          scalaRange
	SelectionRange scalaRange
	Children       []scalaDocumentSymbol
	URI            string
}
type scalaRange struct {
	Start scalaPosition
	End   scalaPosition
}
type scalaPosition struct {
	Line      int
	Character int
}

func decodeScalaDocumentSymbols(raw json.RawMessage, root string, source []byte) ([]scalaDocumentSymbol, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScalaMalformedResponse, err)
	}
	result := make([]scalaDocumentSymbol, 0, len(values))
	for _, value := range values {
		symbol, err := decodeScalaDocumentSymbol(value, root, source)
		if err != nil {
			return nil, err
		}
		result = append(result, symbol)
	}
	return result, nil
}

func decodeScalaDocumentSymbol(raw json.RawMessage, root string, source []byte) (scalaDocumentSymbol, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return scalaDocumentSymbol{}, fmt.Errorf("%w: symbol is not an object", ErrScalaMalformedResponse)
	}
	if _, flat := object["location"]; flat {
		return scalaDocumentSymbol{}, fmt.Errorf("%w: flat SymbolInformation is not supported", ErrScalaUnsupportedResponse)
	}
	var symbol scalaDocumentSymbol
	if err := json.Unmarshal(object["name"], &symbol.Name); err != nil || symbol.Name == "" {
		return scalaDocumentSymbol{}, fmt.Errorf("%w: missing name", ErrScalaMalformedResponse)
	}
	if err := json.Unmarshal(object["kind"], &symbol.Kind); err != nil {
		return scalaDocumentSymbol{}, fmt.Errorf("%w: invalid kind", ErrScalaMalformedResponse)
	}
	if err := json.Unmarshal(object["range"], &symbol.Range); err != nil || !validScalaRange(symbol.Range, source) {
		return scalaDocumentSymbol{}, fmt.Errorf("%w: invalid range", ErrScalaMalformedResponse)
	}
	if err := json.Unmarshal(object["selectionRange"], &symbol.SelectionRange); err != nil || !validScalaRange(symbol.SelectionRange, source) {
		return scalaDocumentSymbol{}, fmt.Errorf("%w: invalid selection range", ErrScalaMalformedResponse)
	}
	if uriRaw, ok := object["uri"]; ok {
		if err := json.Unmarshal(uriRaw, &symbol.URI); err != nil {
			return scalaDocumentSymbol{}, fmt.Errorf("%w: invalid uri", ErrScalaMalformedResponse)
		}
		if symbol.URI != "" {
			uriPath, err := pathutil.FilePathFromURI(symbol.URI)
			if err != nil || !pathutil.PathWithin(root, uriPath) {
				return scalaDocumentSymbol{}, fmt.Errorf("%w: symbol uri is outside workspace", ErrScalaMalformedResponse)
			}
		}
	}
	if childrenRaw, ok := object["children"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return scalaDocumentSymbol{}, fmt.Errorf("%w: invalid children", ErrScalaMalformedResponse)
		}
		for _, childRaw := range children {
			child, err := decodeScalaDocumentSymbol(childRaw, root, source)
			if err != nil {
				return scalaDocumentSymbol{}, err
			}
			symbol.Children = append(symbol.Children, child)
		}
	}
	return symbol, nil
}

func validScalaRange(r scalaRange, source []byte) bool {
	if r.Start.Line < 0 || r.Start.Character < 0 || r.End.Line < 0 || r.End.Character < 0 {
		return false
	}
	start, startErr := scalaByteOffset(source, r.Start)
	end, endErr := scalaByteOffset(source, r.End)
	return startErr == nil && endErr == nil && start <= end
}

func scalaKindName(kind int) string {
	switch kind {
	case 2:
		return "object"
	case 5:
		return "class"
	case 6:
		return "method"
	case 7:
		return "field"
	case 8:
		return "field"
	case 10:
		return "enum"
	case 11:
		return "trait"
	case 23, 26:
		return "type"
	default:
		return ""
	}
}
func scalaKindSupported(kind int) bool { return scalaKindName(kind) != "" }

func selectScalaSymbol(query, root, file string, source []byte, symbols []scalaDocumentSymbol) (*LookupResult, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(query), `"'`), ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, &ScalaError{Op: "lookup", Symbol: query, Err: ErrScalaMalformedResponse}
		}
	}
	type match struct {
		symbol scalaDocumentSymbol
		path   []string
	}
	matches := make([]match, 0)
	var visit func([]string, []scalaDocumentSymbol)
	visit = func(parent []string, items []scalaDocumentSymbol) {
		for _, item := range items {
			path := append(append([]string(nil), parent...), strings.Split(item.Name, ".")...)
			if scalaKindSupported(item.Kind) && len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], ".") == strings.Join(parts, ".") {
				matches = append(matches, match{item, path})
			}
			visit(path, item.Children)
		}
	}
	visit(nil, symbols)
	if len(matches) == 0 {
		return nil, &ScalaError{Op: "lookup", File: file, Symbol: query, Err: ErrScalaSymbolNotFound}
	}
	converted := make([]*SymbolCandidate, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, scalaCandidate(root, file, source, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &LookupResult{Symbol: query, File: filepath.Clean(file), Ambiguous: true, Candidates: converted}, nil
	}
	candidate := converted[0]
	return &LookupResult{Symbol: candidate.QualifiedName, File: candidate.File, Line: candidate.Line, Column: candidate.Column, Offset: candidate.Offset, Kind: candidate.Kind, Receiver: candidate.Receiver, Location: candidate.Location}, nil
}

func scalaCandidate(root, file string, source []byte, symbol scalaDocumentSymbol, path []string) *SymbolCandidate {
	start := symbol.SelectionRange.Start
	offset, _ := scalaByteOffset(source, start)
	receiver := ""
	if len(path) > 1 {
		receiver = strings.Join(path[:len(path)-1], ".")
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		relative = file
	}
	location := SourceLocation{URI: pathutil.FileURI(file), Range: Range{Start: Position(start), End: Position(symbol.SelectionRange.End)}}
	return &SymbolCandidate{Name: symbol.Name, Receiver: receiver, QualifiedName: strings.Join(path, "."), Kind: scalaKindName(symbol.Kind), File: relative, Line: start.Line + 1, Column: start.Character + 1, Offset: offset, Location: location}
}

func scalaByteOffset(source []byte, position scalaPosition) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, errors.New("negative position")
	}
	line, start := 0, 0
	for i, b := range source {
		if b == '\n' {
			if line == position.Line {
				offset, err := scalaCharacterOffset(source[start:i], position.Character)
				return start + offset, err
			}
			line++
			start = i + 1
		}
	}
	if line != position.Line {
		return 0, errors.New("line outside source")
	}
	offset, err := scalaCharacterOffset(source[start:], position.Character)
	return start + offset, err
}

func scalaCharacterOffset(line []byte, character int) (int, error) {
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
