// Package backend keeps Java lookup read-only and file-scoped behind one trusted JDT LS session.
package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"semedit/internal/lsp"
	"semedit/internal/pipeline"
)

var (
	// ErrJavaFileRequired indicates that lookup did not receive a selected Java source file.
	ErrJavaFileRequired = errors.New("a selected Java .java file is required")
	// ErrJavaWorkspaceRequired indicates that no Maven or Gradle root could be discovered.
	ErrJavaWorkspaceRequired = errors.New("java workspace root is required")
	// ErrJavaWorkspaceAmbiguous indicates same-level Maven and Gradle markers.
	ErrJavaWorkspaceAmbiguous = errors.New("java workspace root is ambiguous")
	// ErrJavaFileOutsideWorkspace indicates that the selected source is outside the root.
	ErrJavaFileOutsideWorkspace = errors.New("java file is outside the workspace root")
	// ErrJDTLSHomeRequired indicates that no explicit JDT LS distribution was supplied.
	ErrJDTLSHomeRequired = errors.New("an explicitly configured JDT LS home is required")
	// ErrJDTLSUnavailable indicates that the configured server could not be used.
	ErrJDTLSUnavailable = errors.New("JDT LS is not installed")
	// ErrJDTLSLauncherInvalid indicates that a distribution launcher or config is invalid.
	ErrJDTLSLauncherInvalid = errors.New("JDT LS launcher or configuration is invalid")
	// ErrJavaRuntimeUnavailable indicates that Java could not be executed.
	ErrJavaRuntimeUnavailable = errors.New("java runtime is unavailable")
	// ErrJavaRuntimeTooOld indicates that the runtime is older than Java 21.
	ErrJavaRuntimeTooOld = errors.New("java runtime 21 or newer is required")
	// ErrJavaMalformedResponse indicates an invalid hierarchical response.
	ErrJavaMalformedResponse = errors.New("malformed JDT LS document symbols response")
	// ErrJavaUnsupportedResponse indicates a flat or otherwise unsupported response.
	ErrJavaUnsupportedResponse = errors.New("unsupported JDT LS document symbols response")
	// ErrJavaSessionConflict indicates that another root already owns the session.
	ErrJavaSessionConflict = errors.New("another Java language session is active")
	// ErrJavaSymbolNotFound indicates that no exact hierarchical candidate matched.
	ErrJavaSymbolNotFound = errors.New("java symbol not found")
	// ErrJavaRenameInvalidEdit indicates an unsafe or malformed workspace edit.
	ErrJavaRenameInvalidEdit = errors.New("invalid Java rename edit")
	// ErrJavaRenameStaleEdit indicates that an edit preimage differs from source.
	ErrJavaRenameStaleEdit = errors.New("stale Java rename edit")
)

// JavaConfig contains only explicit external-tool paths. An empty JavaBin uses
// the user's PATH; JDTLSHome is always required and is never auto-discovered.
type JavaConfig struct {
	JDTLSHome string `json:"jdtls_home,omitempty"`
	JavaBin   string `json:"java_bin,omitempty"`
	// ImportMaven enables JDT LS Maven project import after explicit trust.
	// Gradle import is never enabled by this option.
	ImportMaven bool `json:"import_maven,omitempty"`
}

func effectiveJavaConfig(project ProjectContext) JavaConfig {
	config := project.Java
	if config.JDTLSHome == "" {
		config.JDTLSHome = project.JDTLSHome
	}
	if config.JavaBin == "" {
		config.JavaBin = project.JavaBin
	}
	return config
}

// JavaError identifies a failure in the read-only Java lookup adapter.
type JavaError struct {
	Op        string
	File      string
	Symbol    string
	Workspace string
	Err       error
}

func (e *JavaError) Error() string {
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
func (e *JavaError) Unwrap() error { return e.Err }

// JavaSession is the small part of a managed JDT LS session needed by lookup.
type JavaSession interface {
	Request(context.Context, string, any) (json.RawMessage, error)
	Notify(context.Context, string, any) error
	Close() error
}

// JavaSessionFactory allows hermetic tests to replace the process-backed session.
// Implementations may accept either (context.Context, string) or
// (context.Context, string, JavaConfig); the latter receives request settings.
type JavaSessionFactory any

type javaSessionFactory func(context.Context, string, JavaConfig) (JavaSession, error)

// JavaBackendOption configures a JavaBackend.
type JavaBackendOption func(*JavaBackend)

// WithJavaSessionFactory injects a fake or managed LSP transport for tests.
func WithJavaSessionFactory(factory JavaSessionFactory) JavaBackendOption {
	return func(backend *JavaBackend) {
		switch typed := factory.(type) {
		case func(context.Context, string, JavaConfig) (JavaSession, error):
			backend.factory = javaSessionFactory(typed)
		case func(context.Context, string) (JavaSession, error):
			backend.factory = javaSessionFactory(func(ctx context.Context, root string, _ JavaConfig) (JavaSession, error) { return typed(ctx, root) })
		default:
			backend.factory = nil
		}
	}
}

// JavaBackend provides trusted, file-scoped lookup and rename through JDT LS.
type JavaBackend struct {
	mu          sync.Mutex
	factory     javaSessionFactory
	root        string
	fingerprint string
	session     JavaSession
}

// TrustedWorkspaceRoot discovers the workspace used for trust comparison.
func (b *JavaBackend) TrustedWorkspaceRoot(project ProjectContext) (string, error) {
	return javaWorkspaceRoot(project)
}

// NewJavaBackend constructs the managed JDT LS lookup adapter.
func NewJavaBackend(options ...JavaBackendOption) *JavaBackend {
	backend := &JavaBackend{factory: defaultJavaSessionFactory}
	for _, option := range options {
		if option != nil {
			option(backend)
		}
	}
	return backend
}

// NewJavaBackendWithFactory constructs a Java backend with an injected session factory.
func NewJavaBackendWithFactory(factory JavaSessionFactory) *JavaBackend {
	return NewJavaBackend(WithJavaSessionFactory(factory))
}

// Language returns the Java language identifier.
func (*JavaBackend) Language() LanguageID { return LanguageJava }

// Capabilities declares Java's trusted read-only lookup capability.
func (*JavaBackend) Capabilities() Capabilities {
	return NewCapabilitiesRequiringWorkspaceTrust(OperationLookup, OperationRename)
}

// CapabilityMatrix returns the declarative documentation matrix for the Java backend.
func (*JavaBackend) CapabilityMatrix() LanguageMatrix {
	return LanguageMatrix{
		Language:    "java",
		DisplayName: "Java",
		Maturity:    "Selected-file refactoring preview",
		Operations: map[string]OpCapability{
			"rename": {
				Supported:    true,
				Description:  "Trusted JDT LS semantic rename confined to one selected canonical Java file; workspace-wide edits are rejected.",
				CLICommand:   "semedit rename --language java --trust-workspace --jdtls-home <path> --java-bin <path> --file <path.java> --symbol <sym> --to <name>",
				MCPTool:      "semantic_rename",
				PlacementKey: false,
			},
			"lookup": {
				Supported:    true,
				Description:  "File-scoped hierarchical Java symbol lookup through a trusted, preinstalled JDT LS session using UTF-16 LSP positions.",
				CLICommand:   "semedit lookup --language java --file <path.java> --symbol <sym> --jdtls-home <path>",
				MCPTool:      "resolve_symbol_location",
				PlacementKey: false,
			},
		},
		Limitations: []Constraint{
			{
				Title:       "Selected-File Rename Only",
				Description: "Java supports selected-file rename only; formatting, imports, verification, extraction, inline, move, and hierarchy refactoring capabilities are unavailable.",
				Severity:    "error",
			},
			{
				Title:       "Trusted Explicit Workspace",
				Description: "Lookup requires a selected .java file, an explicit or Maven-reactor-aware workspace root, explicit workspace trust, a preinstalled JDT LS distribution, and Java 21 or newer; build tools are never invoked. Maven import is opt-in and Gradle import remains disabled.",
				Severity:    "error",
			},
			{
				Title:       "Hierarchical Document Symbols",
				Description: "Only exact hierarchical package, type, field, method, and constructor document symbols are resolved; overload signatures, locals, generated symbols, and malformed-source fallback are not promised.",
				Severity:    "info",
			},
		},
	}
}

// Close terminates the managed JDT LS server, if one is active.
func (b *JavaBackend) Close() error {
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
	b.fingerprint = ""
	return err
}

// Lookup resolves one exact hierarchical symbol in the selected Java file.
func (b *JavaBackend) Lookup(ctx context.Context, project ProjectContext, query string) (*LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveJavaProject(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &WorkspaceTrustError{Operation: OperationLookup, Language: LanguageJava, Workspace: root}
	}
	javaConfig := effectiveJavaConfig(project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, javaConfig)
	if fingerprintErr != nil {
		return nil, &JavaError{Op: "fingerprint", Workspace: root, Err: fingerprintErr}
	}
	if strings.TrimSpace(query) == "" {
		return nil, &JavaError{Op: "lookup", File: file, Symbol: query, Err: ErrJavaMalformedResponse}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil && b.root != root {
		return nil, &JavaError{Op: "lookup", Workspace: root, Err: ErrJavaSessionConflict}
	}
	if b.session != nil && b.fingerprint != fingerprint {
		if err := b.session.Close(); err != nil {
			return nil, &JavaError{Op: "restart", Workspace: root, Err: fmt.Errorf("close stale Java session: %w", err)}
		}
		b.session, b.root, b.fingerprint = nil, "", ""
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
		}
		session, factoryErr := b.factory(ctx, root, javaConfig)
		if factoryErr != nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: factoryErr}
		}
		if session == nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
		}
		b.session, b.root, b.fingerprint = session, root, fingerprint
		if err := initializeJavaSession(ctx, session, root, javaConfig); err != nil {
			_ = session.Close()
			b.session, b.root, b.fingerprint = nil, "", ""
			return nil, &JavaError{Op: "initialize", Workspace: root, Err: err}
		}
	}
	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": fileURI(file), "languageId": "java", "version": 1, "text": string(source),
	}}); err != nil {
		return nil, &JavaError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": fileURI(file)}})
	if err != nil {
		return nil, &JavaError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	symbols, err := decodeJavaDocumentSymbols(raw, root, source)
	if err != nil {
		return nil, &JavaError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	return selectJavaSymbol(query, root, file, source, symbols)
}

// Rename executes one trusted, file-scoped JDT LS rename transaction.
func (b *JavaBackend) Rename(ctx context.Context, request RenameRequest) (*RenameResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveJavaProject(request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &WorkspaceTrustError{Operation: OperationRename, Language: LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, config)
	if fingerprintErr != nil {
		return nil, &JavaError{Op: "fingerprint", Workspace: root, Err: fingerprintErr}
	}
	if strings.TrimSpace(request.Symbol) == "" || strings.TrimSpace(request.To) == "" {
		return nil, &Error{Operation: OperationRename, Language: LanguageJava, Err: ErrJavaRenameInvalidEdit}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil && b.root != root {
		return nil, &JavaError{Op: "rename", Workspace: root, Err: ErrJavaSessionConflict}
	}
	if b.session != nil && b.fingerprint != fingerprint {
		if err := b.session.Close(); err != nil {
			return nil, &JavaError{Op: "restart", Workspace: root, Err: fmt.Errorf("close stale Java session: %w", err)}
		}
		b.session, b.root, b.fingerprint = nil, "", ""
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
		}
		s, factoryErr := b.factory(ctx, root, config)
		if factoryErr != nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: factoryErr}
		}
		if s == nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
		}
		b.session, b.root, b.fingerprint = s, root, fingerprint
		if initErr := initializeJavaSession(ctx, s, root, config); initErr != nil {
			_ = s.Close()
			b.session, b.root, b.fingerprint = nil, "", ""
			return nil, &JavaError{Op: "initialize", Workspace: root, Err: initErr}
		}
	}
	session := b.session
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": fileURI(file), "languageId": "java", "version": 1, "text": string(source)}}); err != nil {
		return nil, &JavaError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": fileURI(file)}})
	if err != nil {
		return nil, &JavaError{Op: "documentSymbol", File: file, Err: err}
	}
	symbols, err := decodeJavaDocumentSymbols(raw, root, source)
	if err != nil {
		return nil, &JavaError{Op: "documentSymbol", File: file, Err: err}
	}
	lookup, err := selectJavaSymbol(request.Symbol, root, file, source, symbols)
	if err != nil {
		return nil, err
	}
	prepRaw, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{"textDocument": map[string]string{"uri": fileURI(file)}, "position": lookup.Location.Range.Start})
	if err != nil || !validJavaPrepareRename(prepRaw) {
		if err == nil {
			err = ErrJavaRenameInvalidEdit
		}
		return nil, &JavaError{Op: "prepareRename", File: file, Err: err}
	}
	editRaw, err := session.Request(ctx, "textDocument/rename", map[string]any{"textDocument": map[string]string{"uri": fileURI(file)}, "position": lookup.Location.Range.Start, "newName": request.To})
	if err != nil {
		return nil, &JavaError{Op: "rename", File: file, Err: err}
	}
	oldName := request.Symbol
	if parts := strings.Split(oldName, "."); len(parts) > 0 {
		oldName = strings.TrimSpace(parts[len(parts)-1])
	}
	updated, err := applyJavaWorkspaceEdit(file, source, editRaw, oldName)
	if err != nil {
		return nil, &JavaError{Op: "rename", File: file, Err: err}
	}
	if err := pipeline.WriteAtomic(file, updated); err != nil {
		return nil, &JavaError{Op: "write", File: file, Err: err}
	}
	_ = session.Close()
	b.session, b.root = nil, ""
	b.fingerprint = ""
	return &RenameResult{Lookup: lookup}, nil
}

type javaTextEdit struct {
	Range   javaRange `json:"range"`
	NewText string    `json:"newText"`
}
type javaEditSpan struct {
	start, end int
	text       string
}

func validJavaPrepareRename(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var direct javaRange
	if json.Unmarshal(raw, &direct) == nil && validJavaPositionShape(direct) {
		return true
	}
	var wrapped struct {
		Range javaRange `json:"range"`
	}
	return json.Unmarshal(raw, &wrapped) == nil && validJavaPositionShape(wrapped.Range)
}

func validJavaPositionShape(r javaRange) bool {
	return r.Start.Line >= 0 && r.Start.Character >= 0 && r.End.Line >= 0 && r.End.Character >= 0
}

func decodeJavaTextEdits(raw json.RawMessage) ([]javaTextEdit, error) {
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil || len(values) == 0 {
		return nil, ErrJavaRenameInvalidEdit
	}
	edits := make([]javaTextEdit, len(values))
	for i, value := range values {
		var object map[string]json.RawMessage
		if json.Unmarshal(value, &object) != nil || object == nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		if _, ok := object["annotationId"]; ok {
			return nil, ErrJavaRenameInvalidEdit
		}
		if json.Unmarshal(value, &edits[i]) != nil {
			return nil, ErrJavaRenameInvalidEdit
		}
	}
	return edits, nil
}

func applyJavaWorkspaceEdit(file string, source []byte, raw json.RawMessage, oldName string) ([]byte, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, ErrJavaRenameInvalidEdit
	}
	_, hasChanges := object["changes"]
	_, hasDocuments := object["documentChanges"]
	if hasChanges == hasDocuments {
		return nil, ErrJavaRenameInvalidEdit
	}
	if _, ok := object["changeAnnotations"]; ok {
		return nil, ErrJavaRenameInvalidEdit
	}
	if _, ok := object["resourceOperations"]; ok {
		return nil, ErrJavaRenameInvalidEdit
	}
	var edits []javaTextEdit
	if hasChanges {
		var changes map[string]json.RawMessage
		if json.Unmarshal(object["changes"], &changes) != nil || len(changes) != 1 {
			return nil, ErrJavaRenameInvalidEdit
		}
		for uri, encoded := range changes {
			path, err := filePathFromURI(uri)
			if err != nil || path != CanonicalWorkspaceRoot(file) {
				return nil, ErrJavaRenameInvalidEdit
			}
			edits, err = decodeJavaTextEdits(encoded)
			if err != nil {
				return nil, err
			}
		}
	} else {
		var docs []json.RawMessage
		if json.Unmarshal(object["documentChanges"], &docs) != nil || len(docs) != 1 {
			return nil, ErrJavaRenameInvalidEdit
		}
		var doc struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version *int   `json:"version"`
			} `json:"textDocument"`
			Edits              json.RawMessage `json:"edits"`
			ResourceOperations json.RawMessage `json:"resourceOperations"`
			AnnotationID       json.RawMessage `json:"annotationId"`
		}
		if json.Unmarshal(docs[0], &doc) != nil || doc.TextDocument.URI == "" || doc.TextDocument.Version == nil || *doc.TextDocument.Version != 1 || doc.ResourceOperations != nil || doc.AnnotationID != nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		path, err := filePathFromURI(doc.TextDocument.URI)
		if err != nil || path != CanonicalWorkspaceRoot(file) {
			return nil, ErrJavaRenameInvalidEdit
		}
		var decodeErr error
		edits, decodeErr = decodeJavaTextEdits(doc.Edits)
		if decodeErr != nil {
			return nil, decodeErr
		}
	}
	spans := make([]javaEditSpan, 0, len(edits))
	seen := map[string]bool{}
	for _, edit := range edits {
		start, err := javaByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		end, err := javaByteOffset(source, edit.Range.End)
		if err != nil || end <= start || edit.NewText == "" {
			return nil, ErrJavaRenameInvalidEdit
		}
		if string(source[start:end]) != oldName {
			return nil, ErrJavaRenameStaleEdit
		}
		key := fmt.Sprintf("%d:%d", start, end)
		if seen[key] {
			return nil, ErrJavaRenameInvalidEdit
		}
		seen[key] = true
		spans = append(spans, javaEditSpan{start: start, end: end, text: edit.NewText})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })
	for i := 1; i < len(spans); i++ {
		if spans[i].end > spans[i-1].start {
			return nil, ErrJavaRenameInvalidEdit
		}
	}
	updated := append([]byte(nil), source...)
	for _, span := range spans {
		updated = append(updated[:span.start], append([]byte(span.text), updated[span.end:]...)...)
	}
	return updated, nil
}

// Verify is intentionally unavailable for the Java lookup-only slice.
func (*JavaBackend) Verify(context.Context, ProjectContext, string) ([]Diagnostic, error) {
	return nil, &Error{Operation: OperationVerify, Language: LanguageJava, Err: ErrUnsupportedOperation}
}

func javaJDTLSSettings(config JavaConfig) map[string]any {
	mavenImport := config.ImportMaven
	return map[string]any{
		"java.autobuild.enabled":                          false,
		"java.import.maven.enabled":                       mavenImport,
		"java.import.gradle.enabled":                      false,
		"java.import.generatesMetadataFilesAtProjectRoot": false,
		"java.configuration.updateBuildConfiguration":     "disabled",
		"java": map[string]any{
			"autobuild": map[string]any{"enabled": false},
			"import": map[string]any{
				"maven":                               map[string]any{"enabled": mavenImport},
				"gradle":                              map[string]any{"enabled": false},
				"generatesMetadataFilesAtProjectRoot": false,
			},
			"configuration": map[string]any{"updateBuildConfiguration": "disabled"},
		},
	}
}

func initializeJavaSession(ctx context.Context, session JavaSession, root string, config JavaConfig) error {
	settings := javaJDTLSSettings(config)
	params := map[string]any{
		"processId":        nil,
		"rootUri":          fileURI(root),
		"workspaceFolders": []map[string]string{{"uri": fileURI(root), "name": filepath.Base(root)}},
		"capabilities": map[string]any{
			"general":      map[string]any{"positionEncodings": []string{"utf-16"}},
			"textDocument": map[string]any{"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true}},
			"workspace":    map[string]any{"workspaceFolders": true, "configuration": true},
		},
		"initializationOptions": map[string]any{"bundles": []string{}},
		"settings":              settings,
	}
	if _, err := session.Request(ctx, "initialize", params); err != nil {
		return err
	}
	if err := session.Notify(ctx, "initialized", map[string]any{}); err != nil {
		return err
	}
	return session.Notify(ctx, "workspace/didChangeConfiguration", map[string]any{"settings": settings})
}

func defaultJavaSessionFactory(ctx context.Context, root string, config JavaConfig) (JavaSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.JDTLSHome) == "" {
		return nil, ErrJDTLSHomeRequired
	}
	javaBin := config.JavaBin
	if javaBin == "" {
		javaBin = "java"
	}
	resolvedJava, err := exec.LookPath(javaBin)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrJavaRuntimeUnavailable, err)
	}
	if err := validateJavaRuntime(ctx, resolvedJava); err != nil {
		return nil, err
	}
	launcher, configuration, err := findJDTLSDistribution(config.JDTLSHome)
	if err != nil {
		return nil, err
	}
	dataPath := filepath.Join(root, ".scratch", "jdtls", stableJavaRootHash(root))
	if err := os.MkdirAll(dataPath, 0o700); err != nil {
		return nil, fmt.Errorf("create JDT LS data directory: %w", err)
	}
	cmd := exec.CommandContext(ctx, resolvedJava, // #nosec G204 -- executable is resolved from an explicit path or PATH and arguments are fixed.
		"-Declipse.application=org.eclipse.jdt.ls.core.id1",
		"-Dosgi.bundles.defaultStartLevel=4",
		"-Declipse.product=org.eclipse.jdt.ls.core.product",
		"-Dlog.protocol=true", "-Dlog.level=ALL", "-Xms1g", "-Xmx2G",
		"-jar", launcher, "-configuration", configuration, "-data", dataPath,
	)
	cmd.Dir = root
	client, err := lsp.NewProcessClient(cmd)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func validateJavaRuntime(ctx context.Context, javaBin string) error {
	output, err := exec.CommandContext(ctx, javaBin, "-version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrJavaRuntimeUnavailable, err)
	}
	major, ok := parseJavaMajor(string(output))
	if !ok {
		return fmt.Errorf("%w: could not parse java version", ErrJavaRuntimeTooOld)
	}
	if major < 21 {
		return fmt.Errorf("%w: found Java %d", ErrJavaRuntimeTooOld, major)
	}
	return nil
}

func parseJavaMajor(output string) (int, bool) {
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

func findJDTLSDistribution(home string) (string, string, error) {
	info, err := os.Stat(home)
	if err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("%w: invalid home %q", ErrJDTLSLauncherInvalid, home)
	}
	plugins := filepath.Join(home, "plugins")
	entries, err := os.ReadDir(plugins)
	if err != nil {
		return "", "", fmt.Errorf("%w: read plugins: %w", ErrJDTLSLauncherInvalid, err)
	}
	var launchers []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "org.eclipse.equinox.launcher_") && strings.HasSuffix(entry.Name(), ".jar") {
			launchers = append(launchers, filepath.Join(plugins, entry.Name()))
		}
	}
	sort.Strings(launchers)
	if len(launchers) == 0 {
		return "", "", fmt.Errorf("%w: launcher jar not found", ErrJDTLSLauncherInvalid)
	}
	configNames := map[string]string{"darwin": "config_mac", "linux": "config_linux", "windows": "config_win"}
	configName := configNames[runtime.GOOS]
	if configName == "" {
		return "", "", fmt.Errorf("%w: unsupported host OS", ErrJDTLSLauncherInvalid)
	}
	configuration := filepath.Join(home, configName)
	if info, err := os.Stat(configuration); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("%w: configuration directory not found", ErrJDTLSLauncherInvalid)
	}
	return launchers[0], configuration, nil
}

func stableJavaRootHash(root string) string {
	digest := sha256.Sum256([]byte(CanonicalWorkspaceRoot(root)))
	return hex.EncodeToString(digest[:])[:24]
}

func resolveJavaProject(project ProjectContext) (string, string, []byte, error) {
	if strings.TrimSpace(project.File) == "" || !strings.EqualFold(filepath.Ext(project.File), ".java") {
		return "", "", nil, &JavaError{Op: "project", File: project.File, Err: ErrJavaFileRequired}
	}
	base := project.RootDir
	if base == "" {
		base = "."
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(base, file)
	}
	file = CanonicalWorkspaceRoot(file)
	if project.RootDir == "" {
		root, err := discoverJavaWorkspaceRoot(file)
		if err != nil {
			return "", "", nil, &JavaError{Op: "project", File: file, Err: err}
		}
		base = root
	}
	root := CanonicalWorkspaceRoot(base)
	if !pathWithin(root, file) {
		return "", "", nil, &JavaError{Op: "project", File: file, Workspace: root, Err: ErrJavaFileOutsideWorkspace}
	}
	source, err := os.ReadFile(file) // #nosec G304 -- file is explicitly selected by the caller.
	if err != nil {
		return "", "", nil, &JavaError{Op: "read", File: file, Workspace: root, Err: err}
	}
	return root, file, source, nil
}

func javaWorkspaceRoot(project ProjectContext) (string, error) {
	if project.RootDir != "" {
		return CanonicalWorkspaceRoot(project.RootDir), nil
	}
	if project.File == "" {
		return "", ErrJavaFileRequired
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(".", file)
	}
	return discoverJavaWorkspaceRoot(CanonicalWorkspaceRoot(file))
}

func discoverJavaWorkspaceRoot(file string) (string, error) {
	dir := filepath.Dir(file)
	for {
		hasMaven := fileExists(filepath.Join(dir, "pom.xml"))
		hasGradle := fileExists(filepath.Join(dir, "build.gradle")) || fileExists(filepath.Join(dir, "build.gradle.kts"))
		if hasMaven || hasGradle {
			if hasMaven && hasGradle {
				return "", ErrJavaWorkspaceAmbiguous
			}
			if hasMaven {
				return discoverMavenReactorRoot(dir)
			}
			return CanonicalWorkspaceRoot(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ErrJavaWorkspaceRequired
}

// discoverMavenReactorRoot follows only explicit POM parent/module relationships.
// It never invokes Maven or reads user settings, so discovery remains side-effect free.
func discoverMavenReactorRoot(module string) (string, error) {
	current := CanonicalWorkspaceRoot(module)
	ancestor := filepath.Dir(current)
	for ancestor != filepath.Dir(ancestor) {
		pom := filepath.Join(ancestor, "pom.xml")
		if fileExists(pom) {
			if mavenPOMListsModule(pom, current) {
				if fileExists(filepath.Join(ancestor, "build.gradle")) || fileExists(filepath.Join(ancestor, "build.gradle.kts")) {
					return "", ErrJavaWorkspaceAmbiguous
				}
				current = ancestor
			}
		}
		ancestor = filepath.Dir(ancestor)
	}
	return current, nil
}

type mavenPOM struct {
	Parent struct {
		RelativePath string `xml:"relativePath"`
	} `xml:"parent"`
	Modules struct {
		Module []string `xml:"module"`
	} `xml:"modules"`
}

func mavenPOMListsModule(parentPom, childDir string) bool {
	data, err := os.ReadFile(parentPom) // #nosec G304 -- parentPom is derived from canonical workspace traversal.
	if err != nil {
		return false
	}
	var pom mavenPOM
	if xml.Unmarshal(data, &pom) != nil {
		return false
	}
	parentDir := filepath.Dir(parentPom)
	for _, module := range pom.Modules.Module {
		if CanonicalWorkspaceRoot(filepath.Join(parentDir, filepath.Clean(module))) == CanonicalWorkspaceRoot(childDir) {
			return true
		}
	}
	return false
}

// javaImportFingerprint identifies the selected Maven reactor descriptors and
// import-related runtime configuration without invoking any build tool.
func javaImportFingerprint(root string, config JavaConfig) (string, error) {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "jdtls=%s\njava=%s\nimport_maven=%t\n", config.JDTLSHome, config.JavaBin, config.ImportMaven)
	paths := []string{filepath.Join(CanonicalWorkspaceRoot(root), "pom.xml")}
	seen := make(map[string]bool)
	for len(paths) > 0 {
		path := paths[0]
		paths = paths[1:]
		path = CanonicalWorkspaceRoot(path)
		if seen[path] {
			continue
		}
		seen[path] = true
		if !pathWithin(CanonicalWorkspaceRoot(root), path) {
			return "", fmt.Errorf("Java project descriptor %s is outside workspace root %s: %w", path, root, ErrJavaFileOutsideWorkspace)
		}
		data, err := os.ReadFile(path) // #nosec G304 -- path is derived from the trusted workspace and POM modules.
		if os.IsNotExist(err) && path == CanonicalWorkspaceRoot(filepath.Join(root, "pom.xml")) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read Java project descriptor %s: %w", path, err)
		}
		_, _ = fmt.Fprintf(hash, "%s\x00", path)
		_, _ = hash.Write(data)
		if len(strings.TrimSpace(string(data))) == 0 {
			continue
		}
		var pom mavenPOM
		if err := xml.Unmarshal(data, &pom); err != nil {
			return "", fmt.Errorf("parse Java project descriptor %s: %w", path, err)
		}
		base := filepath.Dir(path)
		for _, module := range pom.Modules.Module {
			paths = append(paths, filepath.Join(base, filepath.Clean(module), "pom.xml"))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func fileExists(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() }

type javaDocumentSymbol struct {
	Name           string
	Kind           int
	Range          javaRange
	SelectionRange javaRange
	Children       []javaDocumentSymbol
	URI            string
}
type javaRange struct {
	Start javaPosition
	End   javaPosition
}
type javaPosition struct {
	Line      int
	Character int
}

func decodeJavaDocumentSymbols(raw json.RawMessage, root string, source []byte) ([]javaDocumentSymbol, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrJavaMalformedResponse, err)
	}
	result := make([]javaDocumentSymbol, 0, len(values))
	for _, value := range values {
		symbol, err := decodeJavaDocumentSymbol(value, root, source)
		if err != nil {
			return nil, err
		}
		result = append(result, symbol)
	}
	return result, nil
}

func decodeJavaDocumentSymbol(raw json.RawMessage, root string, source []byte) (javaDocumentSymbol, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return javaDocumentSymbol{}, fmt.Errorf("%w: symbol is not an object", ErrJavaMalformedResponse)
	}
	if _, flat := object["location"]; flat {
		return javaDocumentSymbol{}, fmt.Errorf("%w: flat SymbolInformation is not supported", ErrJavaUnsupportedResponse)
	}
	var symbol javaDocumentSymbol
	if err := json.Unmarshal(object["name"], &symbol.Name); err != nil || symbol.Name == "" {
		return javaDocumentSymbol{}, fmt.Errorf("%w: missing name", ErrJavaMalformedResponse)
	}
	if err := json.Unmarshal(object["kind"], &symbol.Kind); err != nil {
		return javaDocumentSymbol{}, fmt.Errorf("%w: invalid kind", ErrJavaMalformedResponse)
	}
	if err := json.Unmarshal(object["range"], &symbol.Range); err != nil || !validJavaRange(symbol.Range, source) {
		return javaDocumentSymbol{}, fmt.Errorf("%w: invalid range", ErrJavaMalformedResponse)
	}
	if err := json.Unmarshal(object["selectionRange"], &symbol.SelectionRange); err != nil || !validJavaRange(symbol.SelectionRange, source) {
		return javaDocumentSymbol{}, fmt.Errorf("%w: invalid selection range", ErrJavaMalformedResponse)
	}
	if uriRaw, ok := object["uri"]; ok {
		if err := json.Unmarshal(uriRaw, &symbol.URI); err != nil {
			return javaDocumentSymbol{}, fmt.Errorf("%w: invalid uri", ErrJavaMalformedResponse)
		}
		if symbol.URI != "" {
			uriPath, err := filePathFromURI(symbol.URI)
			if err != nil || !pathWithin(root, uriPath) {
				return javaDocumentSymbol{}, fmt.Errorf("%w: symbol uri is outside workspace", ErrJavaMalformedResponse)
			}
		}
	}
	if childrenRaw, ok := object["children"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return javaDocumentSymbol{}, fmt.Errorf("%w: invalid children", ErrJavaMalformedResponse)
		}
		for _, childRaw := range children {
			child, err := decodeJavaDocumentSymbol(childRaw, root, source)
			if err != nil {
				return javaDocumentSymbol{}, err
			}
			symbol.Children = append(symbol.Children, child)
		}
	}
	return symbol, nil
}

func validJavaRange(r javaRange, source []byte) bool {
	if r.Start.Line < 0 || r.Start.Character < 0 || r.End.Line < 0 || r.End.Character < 0 {
		return false
	}
	start, startErr := javaByteOffset(source, r.Start)
	end, endErr := javaByteOffset(source, r.End)
	return startErr == nil && endErr == nil && start <= end
}

func javaKindName(kind int) string {
	switch kind {
	case 4:
		return "package"
	case 5:
		return "class"
	case 6:
		return "method"
	case 8:
		return "field"
	case 9:
		return "constructor"
	case 10:
		return "enum"
	case 11:
		return "interface"
	case 23:
		return "record"
	default:
		return ""
	}
}
func javaKindSupported(kind int) bool { return javaKindName(kind) != "" }

func selectJavaSymbol(query, root, file string, source []byte, symbols []javaDocumentSymbol) (*LookupResult, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(query), `"'`), ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, &JavaError{Op: "lookup", Symbol: query, Err: ErrJavaMalformedResponse}
		}
	}
	type match struct {
		symbol javaDocumentSymbol
		path   []string
	}
	matches := make([]match, 0)
	var visit func([]string, []javaDocumentSymbol)
	visit = func(parent []string, items []javaDocumentSymbol) {
		for _, item := range items {
			path := append(append([]string(nil), parent...), strings.Split(item.Name, ".")...)
			if javaKindSupported(item.Kind) && len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], ".") == strings.Join(parts, ".") {
				matches = append(matches, match{item, path})
			}
			visit(path, item.Children)
		}
	}
	visit(nil, symbols)
	if len(matches) == 0 {
		return nil, &JavaError{Op: "lookup", File: file, Symbol: query, Err: ErrJavaSymbolNotFound}
	}
	converted := make([]*SymbolCandidate, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, javaCandidate(root, file, source, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &LookupResult{Symbol: query, File: filepath.Clean(file), Ambiguous: true, Candidates: converted}, nil
	}
	candidate := converted[0]
	return &LookupResult{Symbol: candidate.QualifiedName, File: candidate.File, Line: candidate.Line, Column: candidate.Column, Offset: candidate.Offset, Kind: candidate.Kind, Receiver: candidate.Receiver, Location: candidate.Location}, nil
}

func javaCandidate(root, file string, source []byte, symbol javaDocumentSymbol, path []string) *SymbolCandidate {
	start := symbol.SelectionRange.Start
	offset, _ := javaByteOffset(source, start)
	receiver := ""
	if len(path) > 1 {
		receiver = strings.Join(path[:len(path)-1], ".")
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		relative = file
	}
	location := SourceLocation{URI: fileURI(file), Range: Range{Start: Position(start), End: Position(symbol.SelectionRange.End)}}
	return &SymbolCandidate{Name: symbol.Name, Receiver: receiver, QualifiedName: strings.Join(path, "."), Kind: javaKindName(symbol.Kind), File: relative, Line: start.Line + 1, Column: start.Character + 1, Offset: offset, Location: location}
}

func javaByteOffset(source []byte, position javaPosition) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, errors.New("negative position")
	}
	line, start := 0, 0
	for i, b := range source {
		if b == '\n' {
			if line == position.Line {
				offset, err := javaCharacterOffset(source[start:i], position.Character)
				return start + offset, err
			}
			line++
			start = i + 1
		}
	}
	if line != position.Line {
		return 0, errors.New("line outside source")
	}
	offset, err := javaCharacterOffset(source[start:], position.Character)
	return start + offset, err
}

func javaCharacterOffset(line []byte, character int) (int, error) {
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
