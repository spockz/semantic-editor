// Package kotlin adapts trusted, selected-file Kotlin lookup and diagnostics through kotlin-language-server.
package kotlin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	neutralbackend "semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/lsp"
	"semedit/internal/pipeline"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	// ErrKotlinFileRequired indicates that lookup did not receive a selected Kotlin source file.
	ErrKotlinFileRequired = errors.New("a selected Kotlin .kt or .kts file is required")
	// ErrKotlinProjectMarkersRequireExplicitRoot prevents inferring a build workspace.
	ErrKotlinProjectMarkersRequireExplicitRoot = errors.New("kotlin project markers require an explicit workspace root")
	// ErrKotlinFileOutsideWorkspace indicates that the selected source is outside the root.
	ErrKotlinFileOutsideWorkspace = errors.New("kotlin file is outside the workspace root")
	// ErrKotlinLanguageServerUnavailable indicates that the configured server could not be used.
	ErrKotlinLanguageServerUnavailable = errors.New("kotlin-language-server is not installed")
	// ErrKotlinLanguageServerLauncherInvalid indicates that the configured executable is invalid.
	ErrKotlinLanguageServerLauncherInvalid = errors.New("kotlin-language-server distribution or executable is invalid")
	// ErrKotlinMalformedResponse indicates an invalid hierarchical response.
	ErrKotlinMalformedResponse = errors.New("malformed Kotlin document symbols response")
	// ErrKotlinUnsupportedResponse indicates a flat or otherwise unsupported response.
	ErrKotlinUnsupportedResponse = errors.New("unsupported Kotlin document symbols response")
	// ErrKotlinSessionConflict indicates that another root already owns the session.
	ErrKotlinSessionConflict = errors.New("another Kotlin language session is active")
	// ErrKotlinSymbolNotFound indicates that no exact hierarchical candidate matched.
	ErrKotlinSymbolNotFound = errors.New("kotlin symbol not found")
	// ErrKotlinDiagnosticsTimeout indicates that no matching report arrived within the bounded wait.
	ErrKotlinDiagnosticsTimeout = errors.New("kotlin diagnostics were not published before the bounded timeout")
	// ErrKotlinOperationTimeout indicates that a Kotlin operation exceeded its total deadline.
	ErrKotlinOperationTimeout = errors.New("kotlin language-server operation exceeded its bounded deadline")
)

// KotlinError identifies a failure in the read-only Kotlin lookup adapter.
type KotlinError struct {
	Op        string
	File      string
	Symbol    string
	Workspace string
	Err       error
}

func (e *KotlinError) Error() string {
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
func (e *KotlinError) Unwrap() error { return e.Err }

// KotlinSession is the bounded LSP subset needed by Kotlin lookup and verification.
type KotlinSession interface {
	Request(context.Context, string, any) (json.RawMessage, error)
	Notify(context.Context, string, any) error
	WaitDiagnostics(context.Context, string, int) ([]neutralbackend.Diagnostic, error)
	Close() error
}

// KotlinSessionFactory allows hermetic tests to replace the process-backed session.
// Implementations may accept either (context.Context, string) or
// (context.Context, string, neutralbackend.KotlinConfig); the latter receives request settings.
type KotlinSessionFactory any

type kotlinSessionFactory func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error)

const defaultKotlinOperationTimeout = 30 * time.Second

// KotlinBackendOption configures a KotlinBackend.
type KotlinBackendOption func(*KotlinBackend)

// WithKotlinSessionFactory injects a fake or managed LSP transport for tests.
func WithKotlinSessionFactory(factory KotlinSessionFactory) KotlinBackendOption {
	return func(backend *KotlinBackend) {
		switch typed := factory.(type) {
		case func(context.Context, string, neutralbackend.KotlinConfig) (KotlinSession, error):
			backend.factory = kotlinSessionFactory(typed)
		case func(context.Context, string) (KotlinSession, error):
			backend.factory = kotlinSessionFactory(func(ctx context.Context, root string, _ neutralbackend.KotlinConfig) (KotlinSession, error) {
				return typed(ctx, root)
			})
		default:
			backend.factory = nil
		}
	}
}

// KotlinBackend provides trusted, file-scoped, read-only lookup through kotlin-language-server.
type KotlinBackend struct {
	mu               sync.Mutex
	factory          kotlinSessionFactory
	root             string
	tempRoot         string
	session          KotlinSession
	requestSeq       uint64
	operationTimeout time.Duration
}

// TrustedWorkspaceRoot discovers the workspace used for trust comparison.
func (b *KotlinBackend) TrustedWorkspaceRoot(project neutralbackend.ProjectContext) (string, error) {
	return kotlinWorkspaceRoot(project)
}

// NewKotlinBackend constructs the managed kotlin-language-server lookup adapter.
func NewKotlinBackend(options ...KotlinBackendOption) *KotlinBackend {
	backend := &KotlinBackend{factory: defaultKotlinSessionFactory, operationTimeout: defaultKotlinOperationTimeout}
	for _, option := range options {
		if option != nil {
			option(backend)
		}
	}
	return backend
}

// NewKotlinBackendWithFactory constructs a Kotlin backend with an injected session factory.
func NewKotlinBackendWithFactory(factory KotlinSessionFactory) *KotlinBackend {
	return NewKotlinBackend(WithKotlinSessionFactory(factory))
}

// Language returns the Kotlin language identifier.
func (*KotlinBackend) Language() neutralbackend.LanguageID { return neutralbackend.LanguageKotlin }

// Capabilities declares Kotlin's trusted read-only lookup capability.
func (*KotlinBackend) Capabilities() neutralbackend.Capabilities {
	return neutralbackend.NewCapabilitiesRequiringWorkspaceTrust(neutralbackend.OperationLookup, neutralbackend.OperationVerify)
}

// CapabilityMatrix returns the declarative documentation matrix for the Kotlin backend.
func (*KotlinBackend) CapabilityMatrix() neutralbackend.LanguageMatrix {
	return neutralbackend.LanguageMatrix{
		Language:    "kotlin",
		DisplayName: "Kotlin",
		Maturity:    "Read-only preview",
		Operations: map[string]neutralbackend.OpCapability{
			"lookup": {Supported: true, Description: "Trusted file-scoped hierarchical Kotlin symbol lookup through preinstalled fwcd/kotlin-language-server using UTF-16 positions.", CLICommand: "semedit lookup --language kotlin --file <path.kt> --symbol <sym> --kotlin-bin <path>", MCPTool: "semantic_lookup"},
			"verify": {Supported: true, Description: "Trusted selected-file diagnostics from a matching publishDiagnostics notification; formatting and imports are unsupported.", CLICommand: "semedit verify --language kotlin --file <path.kt> --kotlin-bin <path>", MCPTool: "semantic_verify"},
		},
		Limitations: []neutralbackend.Constraint{
			{
				Title:       "Read Only",
				Description: "Kotlin rename, formatting, imports, build import, and structural edits are unavailable.",
				Severity:    "error",
			},
			{
				Title:       "Trusted Explicit Tools",
				Description: "Lookup and verification require a selected .kt or .kts file, explicit workspace trust, and a preinstalled fwcd/kotlin-language-server executable; upstream may run project classpath scripts or build tools after trust is granted.",
				Severity:    "error",
			},
			{
				Title:       "Hierarchical Document Symbols",
				Description: "Only exact hierarchical classes, objects, interfaces, enums, functions, properties, and nested types are resolved; overload signatures, extension functions, generated symbols, and cross-file search are not promised.",
				Severity:    "info",
			},
		},
	}
}

// Close terminates the managed kotlin-language-server server, if one is active.
func (b *KotlinBackend) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closeSessionLocked()
}

func (b *KotlinBackend) closeSessionLocked() error {
	var err error
	if b.session != nil {
		err = b.session.Close()
	}
	b.session, b.root = nil, ""
	if b.tempRoot != "" {
		if removeErr := os.RemoveAll(b.tempRoot); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove Kotlin request workspace: %w", removeErr))
		}
		b.tempRoot = ""
	}
	return err
}

// Lookup resolves one exact hierarchical symbol in the selected Kotlin file.
func (b *KotlinBackend) Lookup(ctx context.Context, project neutralbackend.ProjectContext, query string) (result *neutralbackend.LookupResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	callerCtx := ctx
	timeout := b.operationTimeout
	if timeout <= 0 {
		timeout = defaultKotlinOperationTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer func() {
		timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
		cancel()
		if retErr != nil && callerCtx.Err() == nil && timedOut {
			retErr = errors.Join(retErr, ErrKotlinOperationTimeout)
		}
	}()
	root, file, source, err := resolveKotlinProject(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &neutralbackend.WorkspaceTrustError{Operation: neutralbackend.OperationLookup, Language: neutralbackend.LanguageKotlin, Workspace: root}
	}
	if strings.TrimSpace(query) == "" {
		return nil, &KotlinError{Op: "lookup", File: file, Symbol: query, Err: ErrKotlinMalformedResponse}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() {
		if closeErr := b.closeSessionLocked(); closeErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close Kotlin request session: %w", closeErr))
		}
	}()
	if b.session != nil && b.root != root {
		return nil, &KotlinError{Op: "lookup", Workspace: root, Err: ErrKotlinSessionConflict}
	}
	if err := b.ensureSession(ctx, root, project.Kotlin); err != nil {
		return nil, err
	}
	scratchFile, err := b.copySelectedFile(root, file, source)
	if err != nil {
		return nil, err
	}
	uri := pathutil.FileURI(scratchFile)
	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "languageId": "kotlin", "version": 1, "text": string(source),
	}}); err != nil {
		return nil, &KotlinError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": uri}})
	if err != nil {
		return nil, &KotlinError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	symbols, err := decodeKotlinDocumentSymbols(raw, b.tempRoot, source)
	if err != nil {
		return nil, &KotlinError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	return selectKotlinSymbol(query, root, file, source, symbols)
}

// Rename is intentionally unavailable for the Kotlin lookup-only slice.
func (*KotlinBackend) Rename(context.Context, neutralbackend.RenameRequest) (*neutralbackend.RenameResult, error) {
	return nil, &neutralbackend.Error{Operation: neutralbackend.OperationRename, Language: neutralbackend.LanguageKotlin, Err: neutralbackend.ErrUnsupportedOperation}
}

// Verify returns only an explicit selected-file publishDiagnostics report.
func (b *KotlinBackend) Verify(ctx context.Context, request neutralbackend.VerifyRequest) (result []neutralbackend.Diagnostic, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	callerCtx := ctx
	timeout := b.operationTimeout
	if timeout <= 0 {
		timeout = defaultKotlinOperationTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer func() {
		timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
		cancel()
		if retErr != nil && callerCtx.Err() == nil && timedOut {
			retErr = errors.Join(retErr, ErrKotlinOperationTimeout)
		}
	}()
	if request.FormatSelectedFile || request.OrganizeImports {
		return nil, &neutralbackend.Error{Operation: neutralbackend.OperationVerify, Language: neutralbackend.LanguageKotlin, Err: neutralbackend.ErrUnsupportedOperation}
	}
	root, file, source, err := resolveKotlinProject(request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &neutralbackend.WorkspaceTrustError{Operation: neutralbackend.OperationVerify, Language: neutralbackend.LanguageKotlin, Workspace: root}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() {
		if closeErr := b.closeSessionLocked(); closeErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close Kotlin request session: %w", closeErr))
		}
	}()
	if b.session != nil && b.root != root {
		return nil, &KotlinError{Op: "verify", Workspace: root, Err: ErrKotlinSessionConflict}
	}
	if err := b.ensureSession(ctx, root, request.Project.Kotlin); err != nil {
		return nil, err
	}
	scratchFile, err := b.copySelectedFile(root, file, source)
	if err != nil {
		return nil, err
	}
	uri := pathutil.FileURI(scratchFile)
	if selector, ok := b.session.(interface{ SelectDiagnosticsURI(string) }); ok {
		selector.SelectDiagnosticsURI(uri)
	}
	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "kotlin", "version": 1, "text": string(source)}}); err != nil {
		return nil, err
	}
	diagnosticCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	diagnostics, err := b.session.WaitDiagnostics(diagnosticCtx, uri, 1)
	if errors.Is(err, context.DeadlineExceeded) {
		err = fmt.Errorf("%w: %w", ErrKotlinDiagnosticsTimeout, err)
	}
	if err != nil {
		return nil, &KotlinError{Op: "verify", File: file, Err: err}
	}
	for i := range diagnostics {
		if diagnostics[i].Location != nil {
			diagnostics[i].Location.URI = pathutil.FileURI(file)
		}
	}
	return diagnostics, nil
}

func rejectKotlinServerRequest(_ context.Context, request lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
	// kotlin-language-server may ask the client to import a build or create configuration. This
	// slice is file-scoped and has no safe response for either operation.
	method := strings.ToLower(request.Method)
	params := strings.ToLower(string(request.Params))
	if strings.Contains(method, "build") || strings.Contains(method, "config") ||
		strings.Contains(params, "build-import") || strings.Contains(params, "buildimport") ||
		strings.Contains(params, "import") || strings.Contains(params, "create") {
		return nil, &lsp.ErrorObject{Code: -32001, Message: "Kotlin build import and configuration prompts are disabled"}
	}
	return nil, &lsp.ErrorObject{Code: -32601, Message: "server request is not supported by read-only Kotlin lookup"}
}

func initializeKotlinSession(ctx context.Context, session KotlinSession, root string) error {
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
	}
	if _, err := session.Request(ctx, "initialize", params); err != nil {
		return err
	}
	return session.Notify(ctx, "initialized", map[string]any{})
}

func (b *KotlinBackend) ensureSession(ctx context.Context, root string, config neutralbackend.KotlinConfig) error {
	if b.session != nil {
		return nil
	}
	if b.factory == nil {
		return &KotlinError{Op: "start", Workspace: root, Err: ErrKotlinLanguageServerUnavailable}
	}
	scratchParent := filepath.Join(root, ".scratch")
	if err := os.MkdirAll(scratchParent, 0o700); err != nil {
		return fmt.Errorf("create Kotlin scratch directory: %w", err)
	}
	tempRoot, err := os.MkdirTemp(scratchParent, "kotlin-ls-")
	if err != nil {
		return fmt.Errorf("create isolated Kotlin workspace: %w", err)
	}
	session, err := b.factory(ctx, tempRoot, config)
	if err != nil {
		cleanupErr := os.RemoveAll(tempRoot)
		return &KotlinError{Op: "start", Workspace: root, Err: errors.Join(err, cleanupErr)}
	}
	if session == nil {
		cleanupErr := os.RemoveAll(tempRoot)
		return &KotlinError{Op: "start", Workspace: root, Err: errors.Join(ErrKotlinLanguageServerUnavailable, cleanupErr)}
	}
	if err := initializeKotlinSession(ctx, session, tempRoot); err != nil {
		closeErr := session.Close()
		removeErr := os.RemoveAll(tempRoot)
		return &KotlinError{Op: "initialize", Workspace: root, Err: errors.Join(err, closeErr, removeErr)}
	}
	b.session, b.root, b.tempRoot = session, root, tempRoot
	return nil
}
func (b *KotlinBackend) copySelectedFile(root, file string, source []byte) (string, error) {
	if !pathutil.PathWithin(root, file) {
		return "", ErrKotlinFileOutsideWorkspace
	}
	ext := strings.ToLower(filepath.Ext(file))
	if ext != ".kt" && ext != ".kts" {
		return "", ErrKotlinFileRequired
	}
	b.requestSeq++
	target := filepath.Join(b.tempRoot, fmt.Sprintf("request-%d%s", b.requestSeq, ext))
	if err := pipeline.WriteAtomic(target, source); err != nil {
		return "", fmt.Errorf("write isolated Kotlin source atomically: %w", err)
	}
	return target, nil
}

func defaultKotlinSessionFactory(ctx context.Context, root string, config neutralbackend.KotlinConfig) (KotlinSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	binary := config.KotlinBin
	if binary == "" {
		var err error
		binary, err = exec.LookPath("kotlin-language-server")
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrKotlinLanguageServerUnavailable, err)
		}
	}
	path, err := filepath.Abs(binary)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("%w: invalid executable %q", ErrKotlinLanguageServerLauncherInvalid, binary)
	}
	cmd := exec.CommandContext(ctx, path) // #nosec G204 -- path is an explicit or PATH-resolved preinstalled server executable.
	cmd.Dir = root
	home := filepath.Join(root, ".home")
	configHome := filepath.Join(root, ".config")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(configHome, 0o700); err != nil {
		return nil, err
	}
	cmd.Env = filteredKotlinEnvironment(os.Environ(), home, configHome)
	processSession := &kotlinProcessSession{root: root, reports: make(map[string]kotlinDiagnosticReceipt), wake: make(chan struct{}, 1)}
	client, err := lsp.NewProcessClient(cmd, lsp.WithRequestHandler(rejectKotlinServerRequest), lsp.WithNotificationHandler(processSession.record))
	if err != nil {
		return nil, err
	}
	processSession.client = client
	return processSession, nil
}

func filteredKotlinEnvironment(env []string, home, configHome string) []string {
	out := make([]string, 0, len(env)+2)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if key != "HOME" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" {
			out = append(out, entry)
		}
	}
	return append(out, "HOME="+home, "XDG_CONFIG_HOME="+configHome, "XDG_CACHE_HOME="+filepath.Join(home, ".cache"))
}

type kotlinProcessSession struct {
	client      *lsp.Client
	root        string
	mu          sync.Mutex
	reports     map[string]kotlinDiagnosticReceipt
	wake        chan struct{}
	selectedURI string
}
type kotlinDiagnosticReceipt struct {
	version        int
	versionPresent bool
	diagnostics    []neutralbackend.Diagnostic
}

func (s *kotlinProcessSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return s.client.Request(ctx, method, params)
}
func (s *kotlinProcessSession) Notify(ctx context.Context, method string, params any) error {
	return s.client.Notify(ctx, method, params)
}
func (s *kotlinProcessSession) Close() error { return s.client.Close() }
func (s *kotlinProcessSession) SelectDiagnosticsURI(uri string) {
	s.mu.Lock()
	s.selectedURI = uri
	s.mu.Unlock()
}
func (s *kotlinProcessSession) record(notification lsp.Notification) {
	if notification.Method != "textDocument/publishDiagnostics" {
		return
	}
	var raw struct {
		URI         string          `json:"uri"`
		Version     *int            `json:"version"`
		Diagnostics json.RawMessage `json:"diagnostics"`
	}
	if json.Unmarshal(notification.Params, &raw) != nil || raw.URI == "" || len(raw.Diagnostics) == 0 || raw.Diagnostics[0] != '[' {
		return
	}
	var rawDiagnostics []struct {
		Message  string      `json:"message"`
		Severity int         `json:"severity"`
		Range    kotlinRange `json:"range"`
	}
	if err := json.Unmarshal(raw.Diagnostics, &rawDiagnostics); err != nil {
		return
	}
	path, err := pathutil.FilePathFromURI(raw.URI)
	if err != nil || !pathutil.PathWithin(s.root, path) {
		return
	}
	diagnostics := make([]neutralbackend.Diagnostic, 0, len(rawDiagnostics))
	for _, d := range rawDiagnostics {
		diagnostics = append(diagnostics, neutralbackend.Diagnostic{Message: d.Message, Severity: d.Severity, Location: &neutralbackend.SourceLocation{URI: raw.URI, Range: neutralbackend.Range{Start: neutralbackend.Position(d.Range.Start), End: neutralbackend.Position(d.Range.End)}}})
	}
	s.mu.Lock()
	if s.selectedURI != "" && s.selectedURI != raw.URI {
		s.mu.Unlock()
		return
	}
	version := 0
	versionPresent := raw.Version != nil
	if versionPresent {
		version = *raw.Version
		if version < 1 {
			s.mu.Unlock()
			return
		}
	}
	old, ok := s.reports[raw.URI]
	if ok && old.versionPresent && versionPresent && old.version > version {
		s.mu.Unlock()
		return
	}
	s.reports[raw.URI] = kotlinDiagnosticReceipt{version: version, versionPresent: versionPresent, diagnostics: diagnostics}
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *kotlinProcessSession) WaitDiagnostics(ctx context.Context, uri string, version int) ([]neutralbackend.Diagnostic, error) {
	for {
		s.mu.Lock()
		receipt, ok := s.reports[uri]
		s.mu.Unlock()
		if ok && (!receipt.versionPresent || receipt.version >= version) {
			return receipt.diagnostics, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.wake:
		}
	}
}

func resolveKotlinProject(project neutralbackend.ProjectContext) (string, string, []byte, error) {
	if strings.TrimSpace(project.File) == "" || (!strings.EqualFold(filepath.Ext(project.File), ".kt") && !strings.EqualFold(filepath.Ext(project.File), ".kts")) {
		return "", "", nil, &KotlinError{Op: "project", File: project.File, Err: ErrKotlinFileRequired}
	}
	base := project.RootDir
	file := project.File
	if base == "" {
		base = "."
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(base, file)
	}
	file = neutralbackend.CanonicalWorkspaceRoot(file)
	if project.RootDir == "" {
		if hasKotlinProjectMarker(file) {
			return "", "", nil, &KotlinError{Op: "project", File: file, Err: ErrKotlinProjectMarkersRequireExplicitRoot}
		}
		base = filepath.Dir(file)
	}
	root := neutralbackend.CanonicalWorkspaceRoot(base)
	if !pathutil.PathWithin(root, file) {
		return "", "", nil, &KotlinError{Op: "project", File: file, Workspace: root, Err: ErrKotlinFileOutsideWorkspace}
	}
	source, err := os.ReadFile(file) // #nosec G304 -- file is explicitly selected by the caller.
	if err != nil {
		return "", "", nil, &KotlinError{Op: "read", File: file, Workspace: root, Err: err}
	}
	return root, file, source, nil
}

func kotlinWorkspaceRoot(project neutralbackend.ProjectContext) (string, error) {
	if project.RootDir != "" {
		return neutralbackend.CanonicalWorkspaceRoot(project.RootDir), nil
	}
	if project.File == "" {
		return "", ErrKotlinFileRequired
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(".", file)
	}
	file = neutralbackend.CanonicalWorkspaceRoot(file)
	if hasKotlinProjectMarker(file) {
		return "", ErrKotlinProjectMarkersRequireExplicitRoot
	}
	return neutralbackend.CanonicalWorkspaceRoot(filepath.Dir(file)), nil
}

func hasKotlinProjectMarker(file string) bool {
	dir := filepath.Dir(file)
	for {
		for _, marker := range []string{"settings.gradle", "settings.gradle.kts", "build.gradle", "build.gradle.kts", "pom.xml", "kls-classpath", "kls-classpath.sh", "kls-classpath.bat", "kls-classpath.cmd"} {
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

type kotlinDocumentSymbol struct {
	Name           string
	Kind           int
	Range          kotlinRange
	SelectionRange kotlinRange
	Children       []kotlinDocumentSymbol
	URI            string
}
type kotlinRange struct {
	Start kotlinPosition
	End   kotlinPosition
}
type kotlinPosition struct {
	Line      int
	Character int
}

func decodeKotlinDocumentSymbols(raw json.RawMessage, root string, source []byte) ([]kotlinDocumentSymbol, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKotlinMalformedResponse, err)
	}
	result := make([]kotlinDocumentSymbol, 0, len(values))
	for _, value := range values {
		symbol, err := decodeKotlinDocumentSymbol(value, root, source)
		if err != nil {
			return nil, err
		}
		result = append(result, symbol)
	}
	return result, nil
}

func decodeKotlinDocumentSymbol(raw json.RawMessage, root string, source []byte) (kotlinDocumentSymbol, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return kotlinDocumentSymbol{}, fmt.Errorf("%w: symbol is not an object", ErrKotlinMalformedResponse)
	}
	if _, flat := object["location"]; flat {
		return kotlinDocumentSymbol{}, fmt.Errorf("%w: flat SymbolInformation is not supported", ErrKotlinUnsupportedResponse)
	}
	var symbol kotlinDocumentSymbol
	if err := json.Unmarshal(object["name"], &symbol.Name); err != nil || symbol.Name == "" {
		return kotlinDocumentSymbol{}, fmt.Errorf("%w: missing name", ErrKotlinMalformedResponse)
	}
	if err := json.Unmarshal(object["kind"], &symbol.Kind); err != nil {
		return kotlinDocumentSymbol{}, fmt.Errorf("%w: invalid kind", ErrKotlinMalformedResponse)
	}
	if err := json.Unmarshal(object["range"], &symbol.Range); err != nil || !validKotlinRange(symbol.Range, source) {
		return kotlinDocumentSymbol{}, fmt.Errorf("%w: invalid range", ErrKotlinMalformedResponse)
	}
	if err := json.Unmarshal(object["selectionRange"], &symbol.SelectionRange); err != nil || !validKotlinRange(symbol.SelectionRange, source) {
		return kotlinDocumentSymbol{}, fmt.Errorf("%w: invalid selection range", ErrKotlinMalformedResponse)
	}
	if uriRaw, ok := object["uri"]; ok {
		if err := json.Unmarshal(uriRaw, &symbol.URI); err != nil {
			return kotlinDocumentSymbol{}, fmt.Errorf("%w: invalid uri", ErrKotlinMalformedResponse)
		}
		if symbol.URI != "" {
			uriPath, err := pathutil.FilePathFromURI(symbol.URI)
			if err != nil || !pathutil.PathWithin(root, uriPath) {
				return kotlinDocumentSymbol{}, fmt.Errorf("%w: symbol uri is outside workspace", ErrKotlinMalformedResponse)
			}
		}
	}
	if childrenRaw, ok := object["children"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return kotlinDocumentSymbol{}, fmt.Errorf("%w: invalid children", ErrKotlinMalformedResponse)
		}
		for _, childRaw := range children {
			child, err := decodeKotlinDocumentSymbol(childRaw, root, source)
			if err != nil {
				return kotlinDocumentSymbol{}, err
			}
			symbol.Children = append(symbol.Children, child)
		}
	}
	return symbol, nil
}

func validKotlinRange(r kotlinRange, source []byte) bool {
	if r.Start.Line < 0 || r.Start.Character < 0 || r.End.Line < 0 || r.End.Character < 0 {
		return false
	}
	start, startErr := kotlinByteOffset(source, r.Start)
	end, endErr := kotlinByteOffset(source, r.End)
	return startErr == nil && endErr == nil && start <= end
}

func kotlinKindName(kind int) string {
	switch kind {
	case 2:
		return "module"
	case 5:
		return "class"
	case 6:
		return "method"
	case 7:
		return "property"
	case 8:
		return "field"
	case 9:
		return "constructor"
	case 10:
		return "enum"
	case 11:
		return "interface"
	case 12:
		return "function"
	case 13:
		return "variable"
	case 14:
		return "constant"
	case 19:
		return "object"
	case 22:
		return "enum member"
	case 23, 26:
		return "type"
	default:
		return ""
	}
}
func kotlinKindSupported(kind int) bool { return kotlinKindName(kind) != "" }

func selectKotlinSymbol(query, root, file string, source []byte, symbols []kotlinDocumentSymbol) (*neutralbackend.LookupResult, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(query), `"'`), ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, &KotlinError{Op: "lookup", Symbol: query, Err: ErrKotlinMalformedResponse}
		}
	}
	type match struct {
		symbol kotlinDocumentSymbol
		path   []string
	}
	matches := make([]match, 0)
	var visit func([]string, []kotlinDocumentSymbol)
	visit = func(parent []string, items []kotlinDocumentSymbol) {
		for _, item := range items {
			path := append(append([]string(nil), parent...), strings.Split(item.Name, ".")...)
			if kotlinKindSupported(item.Kind) && len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], ".") == strings.Join(parts, ".") {
				matches = append(matches, match{item, path})
			}
			visit(path, item.Children)
		}
	}
	visit(nil, symbols)
	if len(matches) == 0 {
		return nil, &KotlinError{Op: "lookup", File: file, Symbol: query, Err: ErrKotlinSymbolNotFound}
	}
	converted := make([]*neutralbackend.SymbolCandidate, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, kotlinCandidate(root, file, source, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &neutralbackend.LookupResult{Symbol: query, File: filepath.Clean(file), Ambiguous: true, Candidates: converted}, nil
	}
	candidate := converted[0]
	return &neutralbackend.LookupResult{Symbol: candidate.QualifiedName, File: candidate.File, Line: candidate.Line, Column: candidate.Column, Offset: candidate.Offset, Kind: candidate.Kind, Receiver: candidate.Receiver, Location: candidate.Location}, nil
}

func kotlinCandidate(root, file string, source []byte, symbol kotlinDocumentSymbol, path []string) *neutralbackend.SymbolCandidate {
	start := symbol.SelectionRange.Start
	offset, _ := kotlinByteOffset(source, start)
	receiver := ""
	if len(path) > 1 {
		receiver = strings.Join(path[:len(path)-1], ".")
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		relative = file
	}
	location := neutralbackend.SourceLocation{URI: pathutil.FileURI(file), Range: neutralbackend.Range{Start: neutralbackend.Position(start), End: neutralbackend.Position(symbol.SelectionRange.End)}}
	return &neutralbackend.SymbolCandidate{Name: symbol.Name, Receiver: receiver, QualifiedName: strings.Join(path, "."), Kind: kotlinKindName(symbol.Kind), File: relative, Line: start.Line + 1, Column: start.Character + 1, Offset: offset, Location: location}
}

func kotlinByteOffset(source []byte, position kotlinPosition) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, errors.New("negative position")
	}
	line, start := 0, 0
	for i, b := range source {
		if b == '\n' {
			if line == position.Line {
				offset, err := kotlinCharacterOffset(source[start:i], position.Character)
				return start + offset, err
			}
			line++
			start = i + 1
		}
	}
	if line != position.Line {
		return 0, errors.New("line outside source")
	}
	offset, err := kotlinCharacterOffset(source[start:], position.Character)
	return start + offset, err
}

func kotlinCharacterOffset(line []byte, character int) (int, error) {
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
