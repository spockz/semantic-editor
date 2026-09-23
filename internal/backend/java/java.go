// Package java keeps Java operations file-scoped behind the neutral backend contract.
package java

import (
	"bytes"
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
	"time"
	"unicode/utf8"

	backend "semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/lsp"
	"semedit/internal/pipeline"
)

// ErrJavaFileRequired indicates that lookup did not receive a selected Java source file.
var (
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
	// ErrJavaDiagnosticsTimeout indicates that JDT LS did not publish readiness diagnostics in time.
	ErrJavaDiagnosticsTimeout = errors.New("java diagnostics readiness timed out")
)

// JavaError identifies a failure in the Java language-server adapter.
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
	WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error)
	Close() error
}

// JavaSessionFactory allows hermetic tests to replace the process-backed session.
// Implementations may accept either (context.Context, string) or
// (context.Context, string, JavaConfig); the latter receives request settings.
type JavaSessionFactory any

func effectiveJavaConfig(project backend.ProjectContext) backend.JavaConfig {
	config := project.Java
	if config.JDTLSHome == "" {
		config.JDTLSHome = project.JDTLSHome
	}
	if config.JavaBin == "" {
		config.JavaBin = project.JavaBin
	}
	return config
}

type javaDiagnosticsSelector interface {
	SelectDiagnosticsURI(string)
}

type javaProcessSession struct {
	client      *lsp.Client
	mu          sync.Mutex
	diagnostics map[string]javaDiagnosticReceipt
	wake        chan struct{}
	selectedURI string
}
type javaDiagnosticReceipt struct {
	version     int
	diagnostics []backend.Diagnostic
}

func (s *javaProcessSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return s.client.Request(ctx, method, params)
}
func (s *javaProcessSession) Notify(ctx context.Context, method string, params any) error {
	return s.client.Notify(ctx, method, params)
}
func (s *javaProcessSession) Close() error { return s.client.Close() }
func (s *javaProcessSession) WaitDiagnostics(ctx context.Context, uri string, version int) ([]backend.Diagnostic, error) {
	s.SelectDiagnosticsURI(uri)
	for {
		s.mu.Lock()
		receipt, ok := s.diagnostics[uri]
		s.mu.Unlock()
		if ok && receipt.version >= version {
			return receipt.diagnostics, nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("%w: %w", ErrJavaDiagnosticsTimeout, ctx.Err())
			}
			return nil, ctx.Err()
		case <-s.wake:
		}
	}
}

func (s *javaProcessSession) SelectDiagnosticsURI(uri string) {
	s.mu.Lock()
	s.selectedURI = uri
	s.mu.Unlock()
}

func (s *javaProcessSession) recordDiagnostics(uri string, version int, diagnostics []backend.Diagnostic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if receipt, ok := s.diagnostics[uri]; ok && receipt.version > version {
		return
	}
	s.diagnostics[uri] = javaDiagnosticReceipt{version: version, diagnostics: diagnostics}
}

func newJavaProcessSession(client *lsp.Client) *javaProcessSession {
	s := &javaProcessSession{client: client, diagnostics: make(map[string]javaDiagnosticReceipt), wake: make(chan struct{}, 1)}
	return s
}

type javaSessionFactory func(context.Context, string, backend.JavaConfig) (JavaSession, error)

// JavaBackendOption configures a JavaBackend.
type JavaBackendOption func(*JavaBackend)

// WithJavaSessionFactory injects a fake or managed LSP transport for tests.
func WithJavaSessionFactory(factory JavaSessionFactory) JavaBackendOption {
	return func(javaBackend *JavaBackend) {
		switch typed := factory.(type) {
		case func(context.Context, string, backend.JavaConfig) (JavaSession, error):
			javaBackend.factory = javaSessionFactory(typed)
		case func(context.Context, string) (JavaSession, error):
			javaBackend.factory = javaSessionFactory(func(ctx context.Context, root string, _ backend.JavaConfig) (JavaSession, error) {
				return typed(ctx, root)
			})
		default:
			javaBackend.factory = nil
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
func (b *JavaBackend) TrustedWorkspaceRoot(project backend.ProjectContext) (string, error) {
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
func (*JavaBackend) Language() backend.LanguageID { return backend.LanguageJava }

// Capabilities declares Java's trusted, file-scoped capabilities.
func (*JavaBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilitiesRequiringWorkspaceTrust(backend.OperationLookup, backend.OperationRename, backend.OperationVerify)
}

// CapabilityMatrix returns the declarative documentation matrix for the Java backend.
func (*JavaBackend) CapabilityMatrix() backend.LanguageMatrix {
	return backend.LanguageMatrix{
		Language:    "java",
		DisplayName: "Java",
		Maturity:    "Selected-file refactoring preview",
		Operations: map[string]backend.OpCapability{
			"verify": {Supported: true, Description: "Trusted selected-file Java formatting and source.organizeImports through JDT LS; Maven/Gradle are never executed.", MCPTool: "semantic_verify"},
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
		Limitations: []backend.Constraint{
			{
				Title:       "Selected-File Rename Only",
				Description: "Java supports selected-file rename, formatting, source.organizeImports, and diagnostics; extraction, inline, move, and hierarchy refactoring remain unavailable.",
				Severity:    "error",
			},
			{
				Title:       "Trusted Explicit Workspace",
				Description: "JDT LS lookup and edits remain selected-file operations with explicit trust and a preinstalled distribution; Maven process actions are registered separately and are not JDT LS capabilities. Gradle import remains disabled.",
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
func (b *JavaBackend) Lookup(ctx context.Context, project backend.ProjectContext, query string) (*backend.LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveJavaProject(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationLookup, Language: backend.LanguageJava, Workspace: root}
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
		"uri": pathutil.FileURI(file), "languageId": "java", "version": 1, "text": string(source),
	}}); err != nil {
		return nil, &JavaError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}})
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
func (b *JavaBackend) Rename(ctx context.Context, request backend.RenameRequest) (*backend.RenameResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveJavaProject(request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationRename, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, config)
	if fingerprintErr != nil {
		return nil, &JavaError{Op: "fingerprint", Workspace: root, Err: fingerprintErr}
	}
	if strings.TrimSpace(request.Symbol) == "" || strings.TrimSpace(request.To) == "" {
		return nil, &backend.Error{Operation: backend.OperationRename, Language: backend.LanguageJava, Err: ErrJavaRenameInvalidEdit}
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
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(file), "languageId": "java", "version": 1, "text": string(source)}}); err != nil {
		return nil, &JavaError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}})
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
	prepRaw, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}, "position": lookup.Location.Range.Start})
	if err != nil || !validJavaPrepareRename(prepRaw) {
		if err == nil {
			err = ErrJavaRenameInvalidEdit
		}
		return nil, &JavaError{Op: "prepareRename", File: file, Err: err}
	}
	editRaw, err := session.Request(ctx, "textDocument/rename", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}, "position": lookup.Location.Range.Start, "newName": request.To})
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
	return &backend.RenameResult{Lookup: lookup}, nil
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
		for key := range object {
			if key != "range" && key != "newText" {
				return nil, ErrJavaRenameInvalidEdit
			}
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
			path, err := pathutil.FilePathFromURI(uri)
			if err != nil || path != backend.CanonicalWorkspaceRoot(file) {
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
		var docObject map[string]json.RawMessage
		if json.Unmarshal(docs[0], &docObject) != nil || docObject == nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		for key := range docObject {
			if key != "textDocument" && key != "edits" {
				return nil, ErrJavaRenameInvalidEdit
			}
		}
		if json.Unmarshal(docs[0], &doc) != nil || doc.TextDocument.URI == "" || doc.TextDocument.Version == nil || *doc.TextDocument.Version != 1 || doc.ResourceOperations != nil || doc.AnnotationID != nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		path, err := pathutil.FilePathFromURI(doc.TextDocument.URI)
		if err != nil || path != backend.CanonicalWorkspaceRoot(file) {
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

const javaDiagnosticsTimeout = 5 * time.Second

// Verify applies only validated selected-file edits and collects bounded diagnostics.
func (b *JavaBackend) Verify(ctx context.Context, request backend.VerifyRequest) (diagnostics []backend.Diagnostic, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveJavaProject(request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationVerify, Language: backend.LanguageJava, Workspace: root}
	}
	if !request.FormatSelectedFile && !request.OrganizeImports {
		return nil, &backend.Error{Operation: backend.OperationVerify, Language: backend.LanguageJava, Err: backend.ErrUnsupportedOperation}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, fmt.Errorf("verify fingerprint: %w", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	cleanupOnError := false
	defer func() {
		if !cleanupOnError || retErr == nil || b.session == nil {
			return
		}
		closeErr := b.session.Close()
		b.session, b.root, b.fingerprint = nil, "", ""
		if closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("close Java session after failure: %w", closeErr)
		}
	}()
	if b.session != nil && b.root != root {
		return nil, &JavaError{Op: "verify", Workspace: root, Err: ErrJavaSessionConflict}
	}
	if b.session != nil && b.fingerprint != fingerprint {
		if err := b.session.Close(); err != nil {
			return nil, fmt.Errorf("close stale Java session: %w", err)
		}
		b.session, b.root, b.fingerprint = nil, "", ""
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
		}
		session, err := b.factory(ctx, root, config)
		if err != nil {
			return nil, err
		}
		b.session, b.root, b.fingerprint = session, root, fingerprint
		cleanupOnError = true
		if err := initializeJavaSession(ctx, session, root, config); err != nil {
			return nil, err
		}
	}
	cleanupOnError = true
	if selector, ok := b.session.(javaDiagnosticsSelector); ok {
		selector.SelectDiagnosticsURI(pathutil.FileURI(file))
	}
	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(file), "languageId": "java", "version": 1, "text": string(source)}}); err != nil {
		return nil, err
	}
	updated := source
	version := 1
	formattingChanged := false
	if request.FormatSelectedFile {
		raw, err := b.session.Request(ctx, "textDocument/formatting", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}, "options": map[string]any{"tabSize": 4, "insertSpaces": true}})
		if err != nil {
			return nil, err
		}
		updated, err = applyJavaFormattingEdits(updated, raw)
		if err != nil {
			return nil, err
		}
		formattingChanged = !bytes.Equal(updated, source)
	}
	if request.OrganizeImports {
		if formattingChanged {
			version = 2
			if err := b.session.Notify(ctx, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(file), "version": version}, "contentChanges": []map[string]string{{"text": string(updated)}}}); err != nil {
				return nil, err
			}
		}
		raw, err := b.session.Request(ctx, "textDocument/codeAction", map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(file), "version": version}, "range": javaRange{}, "context": map[string]any{"only": []string{"source.organizeImports"}, "diagnostics": []any{}}})
		if err != nil {
			return nil, err
		}
		updated, err = applyJavaOrganizeImportsEditVersion(file, updated, raw, version)
		if err != nil {
			return nil, err
		}
	}
	diagnosticsVersion := 1
	if !bytes.Equal(updated, source) {
		if err := pipeline.WriteAtomic(file, updated); err != nil {
			return nil, err
		}
		version++
		if err := b.session.Notify(ctx, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(file), "version": version}, "contentChanges": []map[string]string{{"text": string(updated)}}}); err != nil {
			return nil, err
		}
		if err := b.session.Notify(ctx, "textDocument/didSave", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}}); err != nil {
			return nil, err
		}
		diagnosticsVersion = version
	}
	diagnosticCtx, cancelDiagnostics := context.WithTimeout(ctx, javaDiagnosticsTimeout)
	diagnostics, err = b.session.WaitDiagnostics(diagnosticCtx, pathutil.FileURI(file), diagnosticsVersion)
	cancelDiagnostics()
	if errors.Is(err, context.DeadlineExceeded) {
		err = fmt.Errorf("%w: %w", ErrJavaDiagnosticsTimeout, err)
	}
	closeErr := b.session.Close()
	b.session, b.root, b.fingerprint = nil, "", ""
	if err == nil && closeErr != nil {
		return diagnostics, fmt.Errorf("close Java session: %w", closeErr)
	}
	return diagnostics, err
}

func applyJavaFormattingEdits(source []byte, raw json.RawMessage) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("[]")) {
		return append([]byte(nil), source...), nil
	}
	edits, err := decodeJavaTextEdits(raw)
	if err != nil {
		return nil, err
	}
	spans := make([]javaEditSpan, 0, len(edits))
	for _, edit := range edits {
		start, err := javaByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		end, err := javaByteOffset(source, edit.Range.End)
		if err != nil || end < start {
			return nil, ErrJavaRenameInvalidEdit
		}
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

// applyJavaOrganizeImportsEdit validates one bounded JDT LS source action and
// applies its selected-file edits. Callers own the atomic write.
func applyJavaOrganizeImportsEdit(file string, source []byte, raw json.RawMessage) ([]byte, error) {
	return applyJavaOrganizeImportsEditVersion(file, source, raw, 1)
}

func applyJavaOrganizeImportsEditVersion(file string, source []byte, raw json.RawMessage, expectedVersion int) ([]byte, error) {
	var actions []json.RawMessage
	if json.Unmarshal(raw, &actions) != nil || len(actions) > 1 {
		return nil, ErrJavaRenameInvalidEdit
	}
	if len(actions) == 0 {
		return append([]byte(nil), source...), nil
	}
	var action map[string]json.RawMessage
	if json.Unmarshal(actions[0], &action) != nil || action == nil {
		return nil, ErrJavaRenameInvalidEdit
	}
	var kind string
	if json.Unmarshal(action["kind"], &kind) != nil || kind != "source.organizeImports" {
		return nil, ErrJavaRenameInvalidEdit
	}
	if _, ok := action["command"]; ok {
		return nil, ErrJavaRenameInvalidEdit
	}
	if disabled, ok := action["disabled"]; ok && string(disabled) != "null" {
		return nil, ErrJavaRenameInvalidEdit
	}
	editRaw, ok := action["edit"]
	if !ok {
		return nil, ErrJavaRenameInvalidEdit
	}
	var edit map[string]json.RawMessage
	if json.Unmarshal(editRaw, &edit) != nil || edit == nil {
		return nil, ErrJavaRenameInvalidEdit
	}
	if _, ok := edit["changeAnnotations"]; ok {
		return nil, ErrJavaRenameInvalidEdit
	}
	if _, ok := edit["resourceOperations"]; ok {
		return nil, ErrJavaRenameInvalidEdit
	}
	_, hasChanges := edit["changes"]
	_, hasDocuments := edit["documentChanges"]
	if hasChanges == hasDocuments {
		return nil, ErrJavaRenameInvalidEdit
	}
	var textEdits []javaTextEdit
	if changes, ok := edit["changes"]; ok {
		var files map[string]json.RawMessage
		if json.Unmarshal(changes, &files) != nil || len(files) != 1 {
			return nil, ErrJavaRenameInvalidEdit
		}
		for uri, encoded := range files {
			path, err := pathutil.FilePathFromURI(uri)
			if err != nil || canonicalJavaFilePath(path) != canonicalJavaFilePath(file) {
				return nil, ErrJavaRenameInvalidEdit
			}
			textEdits, err = decodeJavaTextEdits(encoded)
			if err != nil {
				return nil, err
			}
		}
	} else if docs, ok := edit["documentChanges"]; ok {
		var values []json.RawMessage
		if json.Unmarshal(docs, &values) != nil || len(values) != 1 {
			return nil, ErrJavaRenameInvalidEdit
		}
		var valueObject map[string]json.RawMessage
		if json.Unmarshal(values[0], &valueObject) != nil || valueObject == nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		for key := range valueObject {
			if key != "textDocument" && key != "edits" {
				return nil, ErrJavaRenameInvalidEdit
			}
		}
		var doc struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version *int   `json:"version"`
			} `json:"textDocument"`
			Edits json.RawMessage `json:"edits"`
		}
		if json.Unmarshal(values[0], &doc) != nil || doc.TextDocument.Version == nil || *doc.TextDocument.Version != expectedVersion {
			return nil, ErrJavaRenameInvalidEdit
		}
		path, err := pathutil.FilePathFromURI(doc.TextDocument.URI)
		if err != nil || canonicalJavaFilePath(path) != canonicalJavaFilePath(file) {
			return nil, ErrJavaRenameInvalidEdit
		}
		textEdits, err = decodeJavaTextEdits(doc.Edits)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrJavaRenameInvalidEdit
	}
	spans := make([]javaEditSpan, 0, len(textEdits))
	for _, item := range textEdits {
		start, err := javaByteOffset(source, item.Range.Start)
		if err != nil {
			return nil, ErrJavaRenameInvalidEdit
		}
		end, err := javaByteOffset(source, item.Range.End)
		if err != nil || end < start {
			return nil, ErrJavaRenameInvalidEdit
		}
		spans = append(spans, javaEditSpan{start: start, end: end, text: item.NewText})
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

func canonicalJavaFilePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	path = filepath.Clean(path)
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		path = evaluated
	}
	return path
}

func javaJDTLSSettings(config backend.JavaConfig) map[string]any {
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

func initializeJavaSession(ctx context.Context, session JavaSession, root string, config backend.JavaConfig) error {
	settings := javaJDTLSSettings(config)
	params := map[string]any{
		"processId":        nil,
		"rootUri":          pathutil.FileURI(root),
		"workspaceFolders": []map[string]string{{"uri": pathutil.FileURI(root), "name": filepath.Base(root)}},
		"capabilities": map[string]any{
			"general": map[string]any{"positionEncodings": []string{"utf-16"}},
			"textDocument": map[string]any{
				"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true},
				"formatting":     map[string]any{"dynamicRegistration": false},
				"codeAction":     map[string]any{"dynamicRegistration": false, "codeActionLiteralSupport": map[string]any{"codeActionKind": map[string]any{"valueSet": []string{"source.organizeImports"}}}},
			},
			"workspace": map[string]any{"workspaceFolders": true, "configuration": true},
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

func defaultJavaSessionFactory(ctx context.Context, root string, config backend.JavaConfig) (JavaSession, error) {
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
	processSession := newJavaProcessSession(nil)
	client, err := lsp.NewProcessClient(cmd, lsp.WithNotificationHandler(func(notification lsp.Notification) {
		if notification.Method != "textDocument/publishDiagnostics" {
			return
		}
		var params struct {
			URI         string `json:"uri"`
			Version     int    `json:"version"`
			Diagnostics []struct {
				Message  string    `json:"message"`
				Severity int       `json:"severity"`
				Range    javaRange `json:"range"`
			} `json:"diagnostics"`
		}
		if json.Unmarshal(notification.Params, &params) != nil {
			return
		}
		uriPath, err := pathutil.FilePathFromURI(params.URI)
		if err != nil {
			return
		}
		uri := pathutil.FileURI(uriPath)
		if uri != pathutil.FileURI(root) && !pathutil.PathWithin(root, uriPath) {
			return
		}
		processSession.mu.Lock()
		selectedURI := processSession.selectedURI
		processSession.mu.Unlock()
		if selectedURI != "" && selectedURI != uri {
			return
		}
		diagnostics := make([]backend.Diagnostic, 0, len(params.Diagnostics))
		for _, diagnostic := range params.Diagnostics {
			diagnostics = append(diagnostics, backend.Diagnostic{Message: diagnostic.Message, Severity: diagnostic.Severity, Location: &backend.SourceLocation{URI: uri, Range: backend.Range{Start: backend.Position(diagnostic.Range.Start), End: backend.Position(diagnostic.Range.End)}}})
		}
		processSession.recordDiagnostics(uri, params.Version, diagnostics)
		select {
		case processSession.wake <- struct{}{}:
		default:
		}
	}))
	if err != nil {
		return nil, err
	}
	processSession.client = client
	return processSession, nil
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
	digest := sha256.Sum256([]byte(backend.CanonicalWorkspaceRoot(root)))
	return hex.EncodeToString(digest[:])[:24]
}

func resolveJavaProject(project backend.ProjectContext) (string, string, []byte, error) {
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
	file = backend.CanonicalWorkspaceRoot(file)
	if project.RootDir == "" {
		root, err := discoverJavaWorkspaceRoot(file)
		if err != nil {
			return "", "", nil, &JavaError{Op: "project", File: file, Err: err}
		}
		base = root
	}
	root := backend.CanonicalWorkspaceRoot(base)
	if !pathutil.PathWithin(root, file) {
		return "", "", nil, &JavaError{Op: "project", File: file, Workspace: root, Err: ErrJavaFileOutsideWorkspace}
	}
	source, err := os.ReadFile(file) // #nosec G304 -- file is explicitly selected by the caller.
	if err != nil {
		return "", "", nil, &JavaError{Op: "read", File: file, Workspace: root, Err: err}
	}
	return root, file, source, nil
}

func javaWorkspaceRoot(project backend.ProjectContext) (string, error) {
	if project.RootDir != "" {
		return backend.CanonicalWorkspaceRoot(project.RootDir), nil
	}
	if project.File == "" {
		return "", ErrJavaFileRequired
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(".", file)
	}
	return discoverJavaWorkspaceRoot(backend.CanonicalWorkspaceRoot(file))
}

func discoverJavaWorkspaceRoot(file string) (string, error) {
	dir := filepath.Dir(file)
	for {
		hasMaven := pathutil.FileExists(filepath.Join(dir, "pom.xml"))
		hasGradle := pathutil.FileExists(filepath.Join(dir, "build.gradle")) || pathutil.FileExists(filepath.Join(dir, "build.gradle.kts"))
		if hasMaven || hasGradle {
			if hasMaven && hasGradle {
				return "", ErrJavaWorkspaceAmbiguous
			}
			if hasMaven {
				return discoverMavenReactorRoot(dir)
			}
			return backend.CanonicalWorkspaceRoot(dir), nil
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
	current := backend.CanonicalWorkspaceRoot(module)
	ancestor := filepath.Dir(current)
	for ancestor != filepath.Dir(ancestor) {
		pom := filepath.Join(ancestor, "pom.xml")
		if pathutil.FileExists(pom) {
			if mavenPOMListsModule(pom, current) {
				if pathutil.FileExists(filepath.Join(ancestor, "build.gradle")) || pathutil.FileExists(filepath.Join(ancestor, "build.gradle.kts")) {
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
		if backend.CanonicalWorkspaceRoot(filepath.Join(parentDir, filepath.Clean(module))) == backend.CanonicalWorkspaceRoot(childDir) {
			return true
		}
	}
	return false
}

// javaImportFingerprint identifies the selected Maven reactor descriptors and
// import-related runtime configuration without invoking any build tool.
func javaImportFingerprint(root string, config backend.JavaConfig) (string, error) {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "jdtls=%s\njava=%s\nimport_maven=%t\n", config.JDTLSHome, config.JavaBin, config.ImportMaven)
	paths := []string{filepath.Join(backend.CanonicalWorkspaceRoot(root), "pom.xml")}
	seen := make(map[string]bool)
	for len(paths) > 0 {
		path := paths[0]
		paths = paths[1:]
		path = backend.CanonicalWorkspaceRoot(path)
		if seen[path] {
			continue
		}
		seen[path] = true
		if !pathutil.PathWithin(backend.CanonicalWorkspaceRoot(root), path) {
			return "", fmt.Errorf("java project descriptor %s is outside workspace root %s: %w", path, root, ErrJavaFileOutsideWorkspace)
		}
		data, err := os.ReadFile(path) // #nosec G304 -- path is derived from the trusted workspace and POM modules.
		if os.IsNotExist(err) && path == backend.CanonicalWorkspaceRoot(filepath.Join(root, "pom.xml")) {
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
			uriPath, err := pathutil.FilePathFromURI(symbol.URI)
			if err != nil || !pathutil.PathWithin(root, uriPath) {
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

func selectJavaSymbol(query, root, file string, source []byte, symbols []javaDocumentSymbol) (*backend.LookupResult, error) {
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
	converted := make([]*backend.SymbolCandidate, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, javaCandidate(root, file, source, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &backend.LookupResult{Symbol: query, File: filepath.Clean(file), Ambiguous: true, Candidates: converted}, nil
	}
	candidate := converted[0]
	return &backend.LookupResult{Symbol: candidate.QualifiedName, File: candidate.File, Line: candidate.Line, Column: candidate.Column, Offset: candidate.Offset, Kind: candidate.Kind, Receiver: candidate.Receiver, Location: candidate.Location}, nil
}

func javaCandidate(root, file string, source []byte, symbol javaDocumentSymbol, path []string) *backend.SymbolCandidate {
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
	location := backend.SourceLocation{URI: pathutil.FileURI(file), Range: backend.Range{Start: backend.Position(start), End: backend.Position(symbol.SelectionRange.End)}}
	return &backend.SymbolCandidate{Name: symbol.Name, Receiver: receiver, QualifiedName: strings.Join(path, "."), Kind: javaKindName(symbol.Kind), File: relative, Line: start.Line + 1, Column: start.Character + 1, Offset: offset, Location: location}
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
