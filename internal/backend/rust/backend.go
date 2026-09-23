// Package rust adapts Rust language-server operations behind the neutral backend contract.
package rust

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	neutralbackend "semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/lsp"
	"semedit/internal/pipeline"
)

var (
	// ErrRustFileRequired indicates that lookup did not receive a selected Rust source file.
	ErrRustFileRequired = errors.New("a selected Rust .rs file is required")
	// ErrRustWorkspaceRequired indicates that no Cargo root could be discovered.
	ErrRustWorkspaceRequired = errors.New("rust workspace root is required")
	// ErrRustWorkspaceAmbiguous indicates that ancestor Cargo manifests would require guessing.
	ErrRustWorkspaceAmbiguous = errors.New("rust workspace root is ambiguous")
	// ErrRustFileOutsideWorkspace indicates that the selected file is not within the trusted root.
	ErrRustFileOutsideWorkspace = errors.New("rust file is outside the workspace root")
	// ErrRustAnalyzerUnavailable indicates that rust-analyzer is not preinstalled.
	ErrRustAnalyzerUnavailable = errors.New("rust-analyzer is not installed")
	// ErrRustMalformedResponse indicates an invalid hierarchical DocumentSymbol response.
	ErrRustMalformedResponse = errors.New("malformed rust-analyzer document symbols response")
	// ErrRustUnsupportedResponse indicates a valid but unsupported LSP symbol response shape.
	ErrRustUnsupportedResponse = errors.New("unsupported rust-analyzer document symbols response")
	// ErrRustSessionConflict indicates that a second workspace was requested while a session is active.
	ErrRustSessionConflict = errors.New("another Rust language session is active")
	// ErrRustSymbolNotFound indicates that an exact hierarchical query had no candidate.
	ErrRustSymbolNotFound = errors.New("rust symbol not found")
	// ErrRustRenameInvalidEdit indicates an unsafe or malformed workspace edit.
	ErrRustRenameInvalidEdit = errors.New("invalid rust rename edit")
	// ErrRustRenameStaleEdit indicates that an edit preimage differs from source.
	ErrRustRenameStaleEdit = errors.New("stale rust rename edit")
)

// RustError identifies a failure in the read-only Rust lookup adapter.
type RustError struct {
	Op        string
	File      string
	Symbol    string
	Workspace string
	Err       error
}

func (e *RustError) Error() string {
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

func (e *RustError) Unwrap() error { return e.Err }

// RustSession is the small part of an LSP session needed by Rust lookup.
// Implementations own the process and must make Close terminate it.
type RustSession interface {
	Request(context.Context, string, any) (json.RawMessage, error)
	Notify(context.Context, string, any) error
	Close() error
}

// RustSessionFactory creates one managed rust-analyzer session for a canonical root.
type RustSessionFactory func(context.Context, string) (RustSession, error)

// RustBackendOption configures a RustBackend.
type RustBackendOption func(*RustBackend)

// WithRustSessionFactory injects a fake or managed LSP transport for hermetic tests.
func WithRustSessionFactory(factory RustSessionFactory) RustBackendOption {
	return func(backend *RustBackend) { backend.factory = factory }
}

// RustBackend provides file-scoped, read-only Rust symbol lookup through rust-analyzer.
type RustBackend struct {
	mu      sync.Mutex
	factory RustSessionFactory
	root    string
	session RustSession
}

// TrustedWorkspaceRoot discovers the workspace used for trust comparison.
func (b *RustBackend) TrustedWorkspaceRoot(project neutralbackend.ProjectContext) (string, error) {
	return rustWorkspaceRoot(project)
}

// NewRustBackend constructs the managed rust-analyzer lookup adapter.
func NewRustBackend(options ...RustBackendOption) *RustBackend {
	backend := &RustBackend{factory: defaultRustSessionFactory}
	for _, option := range options {
		if option != nil {
			option(backend)
		}
	}
	return backend
}

// NewRustBackendWithFactory is a convenience constructor for injected transports.
func NewRustBackendWithFactory(factory RustSessionFactory) *RustBackend {
	return NewRustBackend(WithRustSessionFactory(factory))
}

// Language returns the Rust language identifier.
func (*RustBackend) Language() neutralbackend.LanguageID { return neutralbackend.LanguageRust }

// Capabilities declares Rust's read-only lookup capability.
func (*RustBackend) Capabilities() neutralbackend.Capabilities {
	return neutralbackend.NewCapabilitiesRequiringWorkspaceTrust(neutralbackend.OperationLookup, neutralbackend.OperationRename)
}

// CapabilityMatrix returns the declarative documentation matrix for the Rust backend.
func (*RustBackend) CapabilityMatrix() neutralbackend.LanguageMatrix {
	return neutralbackend.LanguageMatrix{
		Language:    "rust",
		DisplayName: "Rust",
		Maturity:    "Selected-file refactoring preview",
		Operations: map[string]neutralbackend.OpCapability{
			"rename": {
				Supported:    true,
				Description:  "Trusted rust-analyzer semantic rename confined to one selected canonical Rust file; workspace-wide edits are rejected.",
				CLICommand:   "semedit rename --language rust --trust-workspace --file <path.rs> --symbol <sym> --to <name>",
				MCPTool:      "semantic_rename",
				PlacementKey: false,
			},
			"lookup": {
				Supported:    true,
				Description:  "File-scoped hierarchical Rust symbol lookup through a trusted, preinstalled rust-analyzer session using UTF-16 LSP positions.",
				CLICommand:   "semedit lookup --language rust --file <path.rs> --symbol <sym>",
				MCPTool:      "resolve_symbol_location",
				PlacementKey: false,
			},
		},
		Limitations: []neutralbackend.Constraint{
			{
				Title:       "Selected-File Rename Only",
				Description: "Rust supports selected-file rename only; formatting, imports, verification, extraction, inline, move, and other structural edit/refactoring capabilities are unavailable.",
				Severity:    "error",
			},
			{
				Title:       "Trusted Explicit Workspace",
				Description: "Lookup requires a selected .rs file, a deterministic Cargo root, explicit workspace trust, and a preinstalled rust-analyzer; Cargo is never invoked.",
				Severity:    "error",
			},
			{
				Title:       "Hierarchical Document Symbols",
				Description: "Only exact hierarchical document symbols and associated items are resolved; macros, generated symbols, locals, ambiguous trait methods, and multiple impl resolution are not promised.",
				Severity:    "info",
			},
		},
	}
}

// Close terminates the managed Rust server, if one is active.
func (b *RustBackend) Close() error {
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

// Lookup resolves one exact hierarchical symbol in the selected Rust file.
func (b *RustBackend) Lookup(ctx context.Context, project neutralbackend.ProjectContext, query string) (*neutralbackend.LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveRustProject(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &neutralbackend.WorkspaceTrustError{Operation: neutralbackend.OperationLookup, Language: neutralbackend.LanguageRust, Workspace: root}
	}
	if strings.TrimSpace(query) == "" {
		return nil, &RustError{Op: "lookup", File: file, Symbol: query, Err: ErrRustMalformedResponse}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil && b.root != root {
		return nil, &RustError{Op: "lookup", Workspace: root, Err: ErrRustSessionConflict}
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &RustError{Op: "start", Workspace: root, Err: ErrRustAnalyzerUnavailable}
		}
		session, factoryErr := b.factory(ctx, root)
		if factoryErr != nil {
			return nil, &RustError{Op: "start", Workspace: root, Err: factoryErr}
		}
		if session == nil {
			return nil, &RustError{Op: "start", Workspace: root, Err: ErrRustAnalyzerUnavailable}
		}
		b.session, b.root = session, root
		if err := initializeRustSession(ctx, session, root); err != nil {
			_ = session.Close()
			b.session, b.root = nil, ""
			return nil, &RustError{Op: "initialize", Workspace: root, Err: err}
		}
	}

	if err := b.session.Notify(ctx, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        pathutil.FileURI(file),
			"languageId": "rust",
			"version":    1,
			"text":       string(source),
		},
	}); err != nil {
		return nil, &RustError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]string{"uri": pathutil.FileURI(file)},
	})
	if err != nil {
		return nil, &RustError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	symbols, err := decodeRustDocumentSymbols(raw, root, source)
	if err != nil {
		return nil, &RustError{Op: "documentSymbol", File: file, Symbol: query, Err: err}
	}
	return selectRustSymbol(query, root, file, source, symbols)
}

// Rename executes one trusted, file-scoped rust-analyzer rename transaction.
func (b *RustBackend) Rename(ctx context.Context, request neutralbackend.RenameRequest) (*neutralbackend.RenameResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveRustProject(request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &neutralbackend.WorkspaceTrustError{Operation: neutralbackend.OperationRename, Language: neutralbackend.LanguageRust, Workspace: root}
	}
	if strings.TrimSpace(request.Symbol) == "" || strings.TrimSpace(request.To) == "" {
		return nil, &neutralbackend.Error{Operation: neutralbackend.OperationRename, Language: neutralbackend.LanguageRust, Err: ErrRustRenameInvalidEdit}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil && b.root != root {
		return nil, &RustError{Op: "rename", Workspace: root, Err: ErrRustSessionConflict}
	}
	if b.session == nil {
		if b.factory == nil {
			return nil, &RustError{Op: "start", Workspace: root, Err: ErrRustAnalyzerUnavailable}
		}
		s, factoryErr := b.factory(ctx, root)
		if factoryErr != nil {
			return nil, &RustError{Op: "start", Workspace: root, Err: factoryErr}
		}
		if s == nil {
			return nil, &RustError{Op: "start", Workspace: root, Err: ErrRustAnalyzerUnavailable}
		}
		b.session, b.root = s, root
		if initErr := initializeRustSession(ctx, s, root); initErr != nil {
			_ = s.Close()
			b.session, b.root = nil, ""
			return nil, &RustError{Op: "initialize", Workspace: root, Err: initErr}
		}
	}
	session := b.session
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(file), "languageId": "rust", "version": 1, "text": string(source)}}); err != nil {
		return nil, &RustError{Op: "didOpen", File: file, Err: err}
	}
	raw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}})
	if err != nil {
		return nil, &RustError{Op: "documentSymbol", File: file, Err: err}
	}
	symbols, err := decodeRustDocumentSymbols(raw, root, source)
	if err != nil {
		return nil, &RustError{Op: "documentSymbol", File: file, Err: err}
	}
	lookup, err := selectRustSymbol(request.Symbol, root, file, source, symbols)
	if err != nil {
		return nil, err
	}
	prepRaw, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}, "position": lookup.Location.Range.Start})
	if err != nil {
		return nil, &RustError{Op: "prepareRename", File: file, Err: err}
	}
	if !validPrepareRename(prepRaw) {
		return nil, &RustError{Op: "prepareRename", File: file, Err: ErrRustRenameInvalidEdit}
	}
	editRaw, err := session.Request(ctx, "textDocument/rename", map[string]any{"textDocument": map[string]string{"uri": pathutil.FileURI(file)}, "position": lookup.Location.Range.Start, "newName": request.To})
	if err != nil {
		return nil, &RustError{Op: "rename", File: file, Err: err}
	}
	oldName := request.Symbol
	if parts := strings.Split(oldName, "::"); len(parts) > 0 {
		oldName = strings.TrimSpace(parts[len(parts)-1])
	}
	updated, err := applyRustWorkspaceEdit(file, source, editRaw, oldName)
	if err != nil {
		return nil, &RustError{Op: "rename", File: file, Err: err}
	}
	if err := pipeline.WriteAtomic(file, updated); err != nil {
		return nil, &RustError{Op: "write", File: file, Err: err}
	}
	_ = session.Close()
	b.session, b.root = nil, ""
	return &neutralbackend.RenameResult{Lookup: lookup}, nil
}

type rustTextEdit struct {
	Range   rustRange `json:"range"`
	NewText string    `json:"newText"`
}
type rustEditSpan struct {
	start, end int
	text       string
}

func decodeRustTextEdits(raw json.RawMessage) ([]rustTextEdit, error) {
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil || len(values) == 0 {
		return nil, ErrRustRenameInvalidEdit
	}
	edits := make([]rustTextEdit, len(values))
	for i, value := range values {
		var object map[string]json.RawMessage
		if json.Unmarshal(value, &object) != nil || object == nil {
			return nil, ErrRustRenameInvalidEdit
		}
		if _, ok := object["annotationId"]; ok {
			return nil, ErrRustRenameInvalidEdit
		}
		if json.Unmarshal(value, &edits[i]) != nil {
			return nil, ErrRustRenameInvalidEdit
		}
	}
	return edits, nil
}

func validPrepareRename(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var direct rustRange
	if json.Unmarshal(raw, &direct) == nil && validPositionRangeShape(direct) {
		return true
	}
	var wrapped struct {
		Range rustRange `json:"range"`
	}
	return json.Unmarshal(raw, &wrapped) == nil && validPositionRangeShape(wrapped.Range)
}

func validPositionRangeShape(r rustRange) bool {
	return r.Start.Line >= 0 && r.Start.Character >= 0 && r.End.Line >= 0 && r.End.Character >= 0
}

func applyRustWorkspaceEdit(file string, source []byte, raw json.RawMessage, oldName string) ([]byte, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, ErrRustRenameInvalidEdit
	}
	_, hasChanges := object["changes"]
	_, hasDocuments := object["documentChanges"]
	if hasChanges == hasDocuments {
		return nil, ErrRustRenameInvalidEdit
	}
	if _, ok := object["changeAnnotations"]; ok {
		return nil, ErrRustRenameInvalidEdit
	}
	if _, ok := object["resourceOperations"]; ok {
		return nil, ErrRustRenameInvalidEdit
	}
	var edits []rustTextEdit
	if hasChanges {
		var changes map[string]json.RawMessage
		if json.Unmarshal(object["changes"], &changes) != nil || len(changes) != 1 {
			return nil, ErrRustRenameInvalidEdit
		}
		for uri, encoded := range changes {
			path, err := pathutil.FilePathFromURI(uri)
			if err != nil || path != neutralbackend.CanonicalWorkspaceRoot(file) {
				return nil, ErrRustRenameInvalidEdit
			}
			edits, err = decodeRustTextEdits(encoded)
			if err != nil {
				return nil, err
			}
		}
	} else {
		var docs []json.RawMessage
		if json.Unmarshal(object["documentChanges"], &docs) != nil || len(docs) != 1 {
			return nil, ErrRustRenameInvalidEdit
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
			return nil, ErrRustRenameInvalidEdit
		}
		path, err := pathutil.FilePathFromURI(doc.TextDocument.URI)
		if err != nil || path != neutralbackend.CanonicalWorkspaceRoot(file) {
			return nil, ErrRustRenameInvalidEdit
		}
		edits, err = decodeRustTextEdits(doc.Edits)
		if err != nil {
			return nil, err
		}
	}
	spans := make([]rustEditSpan, 0, len(edits))
	seen := map[string]bool{}
	for _, edit := range edits {
		start, err := rustByteOffset(source, edit.Range.Start)
		if err != nil {
			return nil, ErrRustRenameInvalidEdit
		}
		end, err := rustByteOffset(source, edit.Range.End)
		if err != nil || end <= start {
			return nil, ErrRustRenameInvalidEdit
		}
		if edit.NewText == "" {
			return nil, ErrRustRenameInvalidEdit
		}
		if string(source[start:end]) != oldName {
			return nil, ErrRustRenameStaleEdit
		}
		key := fmt.Sprintf("%d:%d", start, end)
		if seen[key] {
			return nil, ErrRustRenameInvalidEdit
		}
		seen[key] = true
		spans = append(spans, rustEditSpan{start: start, end: end, text: edit.NewText})
	}
	for i := 1; i < len(spans); i++ {
		for j := i; j > 0 && spans[j].start > spans[j-1].start; j-- {
			spans[j], spans[j-1] = spans[j-1], spans[j]
		}
	}
	for i := 1; i < len(spans); i++ {
		if spans[i].end > spans[i-1].start {
			return nil, ErrRustRenameInvalidEdit
		}
	}
	updated := append([]byte(nil), source...)
	for _, span := range spans {
		updated = append(updated[:span.start], append([]byte(span.text), updated[span.end:]...)...)
	}
	return updated, nil
}

// Verify is intentionally unavailable for the Rust lookup-only slice.
func (b *RustBackend) Verify(context.Context, neutralbackend.VerifyRequest) ([]neutralbackend.Diagnostic, error) {
	return nil, &neutralbackend.Error{Operation: neutralbackend.OperationVerify, Language: neutralbackend.LanguageRust, Err: neutralbackend.ErrUnsupportedOperation}
}

const rustAnalyzerSettings = `{"cargo":{"buildScripts":{"enable":false}},"procMacro":{"enable":false},"checkOnSave":{"enable":false}}`

func initializeRustSession(ctx context.Context, session RustSession, root string) error {
	var settings map[string]any
	if err := json.Unmarshal([]byte(rustAnalyzerSettings), &settings); err != nil {
		return fmt.Errorf("decode rust-analyzer settings: %w", err)
	}
	params := map[string]any{
		"processId": nil,
		"rootUri":   pathutil.FileURI(root),
		"workspaceFolders": []map[string]string{{
			"uri":  pathutil.FileURI(root),
			"name": filepath.Base(root),
		}},
		"capabilities": map[string]any{
			"general":   map[string]any{"positionEncodings": []string{"utf-16"}},
			"workspace": map[string]any{"workspaceFolders": true, "configuration": true},
		},
		"initializationOptions": settings,
	}
	if _, err := session.Request(ctx, "initialize", params); err != nil {
		return err
	}
	if err := session.Notify(ctx, "initialized", map[string]any{}); err != nil {
		return err
	}
	return session.Notify(ctx, "workspace/didChangeConfiguration", map[string]any{"settings": settings})
}

func defaultRustSessionFactory(ctx context.Context, root string) (RustSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := exec.LookPath("rust-analyzer")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRustAnalyzerUnavailable, err)
	}
	cmd := exec.CommandContext(ctx, path) // #nosec G204 -- path is resolved by exec.LookPath and receives no project arguments.
	cmd.Dir = root
	client, err := lsp.NewProcessClient(cmd)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func resolveRustProject(project neutralbackend.ProjectContext) (string, string, []byte, error) {
	if strings.TrimSpace(project.File) == "" || !strings.EqualFold(filepath.Ext(project.File), ".rs") {
		return "", "", nil, &RustError{Op: "project", File: project.File, Err: ErrRustFileRequired}
	}
	base := project.RootDir
	if base == "" {
		base = "."
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(base, file)
	}
	file = neutralbackend.CanonicalWorkspaceRoot(file)
	if project.RootDir == "" {
		root, err := discoverCargoRoot(file)
		if err != nil {
			return "", "", nil, &RustError{Op: "project", File: file, Err: err}
		}
		base = root
	}
	root := neutralbackend.CanonicalWorkspaceRoot(base)
	if !pathutil.PathWithin(root, file) {
		return "", "", nil, &RustError{Op: "project", File: file, Workspace: root, Err: ErrRustFileOutsideWorkspace}
	}
	source, err := os.ReadFile(file) // #nosec G304 -- file is explicitly selected by the caller.
	if err != nil {
		return "", "", nil, &RustError{Op: "read", File: file, Workspace: root, Err: err}
	}
	return root, file, source, nil
}

func rustWorkspaceRoot(project neutralbackend.ProjectContext) (string, error) {
	if project.RootDir != "" {
		return neutralbackend.CanonicalWorkspaceRoot(project.RootDir), nil
	}
	if project.File == "" {
		return "", ErrRustFileRequired
	}
	file := project.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(".", file)
	}
	return discoverCargoRoot(neutralbackend.CanonicalWorkspaceRoot(file))
}

func discoverCargoRoot(file string) (string, error) {
	dir := filepath.Dir(file)
	var roots []string
	for {
		if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
			roots = append(roots, neutralbackend.CanonicalWorkspaceRoot(dir))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	switch len(roots) {
	case 0:
		return "", ErrRustWorkspaceRequired
	case 1:
		return roots[0], nil
	default:
		return "", fmt.Errorf("%w: %s", ErrRustWorkspaceAmbiguous, strings.Join(roots, ", "))
	}
}

type rustDocumentSymbol struct {
	Name           string
	Kind           int
	Range          rustRange
	SelectionRange rustRange
	Children       []rustDocumentSymbol
	URI            string
}

type rustRange struct {
	Start rustPosition
	End   rustPosition
}

type rustPosition struct {
	Line      int
	Character int
}

func decodeRustDocumentSymbols(raw json.RawMessage, root string, source []byte) ([]rustDocumentSymbol, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRustMalformedResponse, err)
	}
	result := make([]rustDocumentSymbol, 0, len(values))
	for _, value := range values {
		symbol, err := decodeRustDocumentSymbol(value, root, source)
		if err != nil {
			return nil, err
		}
		result = append(result, symbol)
	}
	return result, nil
}

func decodeRustDocumentSymbol(raw json.RawMessage, root string, source []byte) (rustDocumentSymbol, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return rustDocumentSymbol{}, fmt.Errorf("%w: symbol is not an object", ErrRustMalformedResponse)
	}
	if _, flat := object["location"]; flat {
		return rustDocumentSymbol{}, fmt.Errorf("%w: flat SymbolInformation is not supported", ErrRustUnsupportedResponse)
	}
	var symbol rustDocumentSymbol
	if err := json.Unmarshal(object["name"], &symbol.Name); err != nil || symbol.Name == "" {
		return rustDocumentSymbol{}, fmt.Errorf("%w: missing name", ErrRustMalformedResponse)
	}
	if err := json.Unmarshal(object["kind"], &symbol.Kind); err != nil {
		return rustDocumentSymbol{}, fmt.Errorf("%w: invalid kind", ErrRustMalformedResponse)
	}
	if err := json.Unmarshal(object["range"], &symbol.Range); err != nil || !validRustRange(symbol.Range, source) {
		return rustDocumentSymbol{}, fmt.Errorf("%w: invalid range", ErrRustMalformedResponse)
	}
	if err := json.Unmarshal(object["selectionRange"], &symbol.SelectionRange); err != nil || !validRustRange(symbol.SelectionRange, source) {
		return rustDocumentSymbol{}, fmt.Errorf("%w: invalid selection range", ErrRustMalformedResponse)
	}
	if uriRaw, ok := object["uri"]; ok {
		if err := json.Unmarshal(uriRaw, &symbol.URI); err != nil {
			return rustDocumentSymbol{}, fmt.Errorf("%w: invalid uri", ErrRustMalformedResponse)
		}
		if symbol.URI != "" {
			uriPath, err := pathutil.FilePathFromURI(symbol.URI)
			if err != nil || !pathutil.PathWithin(root, uriPath) {
				return rustDocumentSymbol{}, fmt.Errorf("%w: symbol uri is outside workspace", ErrRustMalformedResponse)
			}
		}
	}
	if childrenRaw, ok := object["children"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return rustDocumentSymbol{}, fmt.Errorf("%w: invalid children", ErrRustMalformedResponse)
		}
		symbol.Children = make([]rustDocumentSymbol, 0, len(children))
		for _, childRaw := range children {
			child, err := decodeRustDocumentSymbol(childRaw, root, source)
			if err != nil {
				return rustDocumentSymbol{}, err
			}
			symbol.Children = append(symbol.Children, child)
		}
	}
	return symbol, nil
}

func validRustRange(r rustRange, source []byte) bool {
	if r.Start.Line < 0 || r.Start.Character < 0 || r.End.Line < 0 || r.End.Character < 0 {
		return false
	}
	start, startErr := rustByteOffset(source, r.Start)
	end, endErr := rustByteOffset(source, r.End)
	return startErr == nil && endErr == nil && start <= end
}

func rustKindName(kind int) string {
	switch kind {
	case 2:
		return "module"
	case 10:
		return "enum"
	case 11:
		return "trait"
	case 12:
		return "function"
	case 13:
		return "static"
	case 14:
		return "const"
	case 23:
		return "struct"
	case 26, 5:
		return "type"
	default:
		return ""
	}
}

func selectRustSymbol(query, root, file string, source []byte, symbols []rustDocumentSymbol) (*neutralbackend.LookupResult, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(query), `"'`), "::")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, &RustError{Op: "lookup", Symbol: query, Err: ErrRustMalformedResponse}
		}
	}
	type match struct {
		symbol rustDocumentSymbol
		path   []string
	}
	matches := make([]match, 0)
	var visit func([]string, []rustDocumentSymbol)
	visit = func(parent []string, items []rustDocumentSymbol) {
		for _, item := range items {
			path := append(append([]string(nil), parent...), item.Name)
			if len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], "::") == strings.Join(parts, "::") {
				matches = append(matches, match{symbol: item, path: path})
			}
			visit(path, item.Children)
		}
	}
	visit(nil, symbols)
	if len(matches) == 0 {
		return nil, &RustError{Op: "lookup", File: file, Symbol: query, Err: ErrRustSymbolNotFound}
	}
	converted := make([]*neutralbackend.SymbolCandidate, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, rustCandidate(root, file, source, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &neutralbackend.LookupResult{Symbol: query, File: filepath.Clean(file), Ambiguous: true, Candidates: converted}, nil
	}
	candidate := converted[0]
	return &neutralbackend.LookupResult{
		Symbol:   candidate.QualifiedName,
		File:     candidate.File,
		Line:     candidate.Line,
		Column:   candidate.Column,
		Offset:   candidate.Offset,
		Kind:     candidate.Kind,
		Receiver: candidate.Receiver,
		Location: candidate.Location,
	}, nil
}

func rustCandidate(root, file string, source []byte, symbol rustDocumentSymbol, path []string) *neutralbackend.SymbolCandidate {
	start := symbol.SelectionRange.Start
	offset, _ := rustByteOffset(source, start)
	receiver := ""
	if len(path) > 1 {
		receiver = strings.Join(path[:len(path)-1], "::")
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		relative = file
	}
	location := neutralbackend.SourceLocation{
		URI: pathutil.FileURI(file),
		Range: neutralbackend.Range{
			Start: neutralbackend.Position{Line: symbol.SelectionRange.Start.Line, Character: symbol.SelectionRange.Start.Character},
			End:   neutralbackend.Position{Line: symbol.SelectionRange.End.Line, Character: symbol.SelectionRange.End.Character},
		},
	}
	return &neutralbackend.SymbolCandidate{
		Name:          symbol.Name,
		Receiver:      receiver,
		QualifiedName: strings.Join(path, "::"),
		Kind:          rustKindName(symbol.Kind),
		File:          relative,
		Line:          start.Line + 1,
		Column:        start.Character + 1,
		Offset:        offset,
		Location:      location,
	}
}

func rustByteOffset(source []byte, position rustPosition) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, errors.New("negative position")
	}
	line := 0
	start := 0
	for i, b := range source {
		if b == '\n' {
			if line == position.Line {
				offset, err := rustCharacterOffset(source[start:i], position.Character)
				return start + offset, err
			}
			line++
			start = i + 1
		}
	}
	if line != position.Line {
		return 0, errors.New("line outside source")
	}
	offset, err := rustCharacterOffset(source[start:], position.Character)
	return start + offset, err
}

func rustCharacterOffset(line []byte, character int) (int, error) {
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
