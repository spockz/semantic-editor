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
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	backend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/backend/readlsp"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/lsp"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
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
	// ErrJavaSymbolAmbiguous indicates that a selector resolved to more than one Java symbol.
	ErrJavaSymbolAmbiguous = errors.New("java symbol selector is ambiguous")
	// ErrJavaRenameInvalidEdit indicates an unsafe or malformed workspace edit.
	ErrJavaRenameInvalidEdit = errors.New("invalid Java rename edit")
	// ErrJavaRenameStaleEdit indicates that an edit preimage differs from source.
	ErrJavaRenameStaleEdit = errors.New("stale Java rename edit")
	// ErrJavaBatchResourceRenameUnsupported indicates that resource moves cannot be staged in a Java batch.
	ErrJavaBatchResourceRenameUnsupported = errors.New("java resource renames are unsupported in buffered batches")
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

const lspDocumentSymbolKey = "documentSymbol"

const lspTextDocumentKey = "textDocument"

func effectiveJavaConfig(project backend.ProjectContext) backend.JavaConfig {
	config := project.Java
	if config.ImportMaven || config.ImportMavenSet {
		return config
	}
	root, err := javaWorkspaceRoot(project)
	if err != nil || !project.WorkspaceTrust.Allows(root) {
		return config
	}
	info, err := os.Lstat(filepath.Join(root, "pom.xml"))
	if err == nil && info.Mode().IsRegular() {
		config.ImportMaven = true
	}
	return config
}

type javaProcessSession struct {
	client           *lsp.Client
	mu               sync.Mutex
	diagnostics      map[string]javaDiagnosticReceipt
	documentVersions map[string]int
	settings         map[string]any
	wake             chan struct{}
	applyEditMu      sync.Mutex
	applyEditHandler func(json.RawMessage) (bool, error)
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
				latest := "no diagnostic receipt"
				if ok {
					latest = fmt.Sprintf("latest receipt version %d", receipt.version)
				}
				return nil, fmt.Errorf("%w: waiting for %s version %d; %s: %w", ErrJavaDiagnosticsTimeout, uri, version, latest, ctx.Err())
			}
			return nil, ctx.Err()
		case <-s.wake:
		}
	}
}

func (s *javaProcessSession) recordDiagnostics(uri string, version int, diagnostics []backend.Diagnostic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.diagnostics == nil {
		s.diagnostics = make(map[string]javaDiagnosticReceipt)
	}
	if version == 0 {
		version = s.documentVersions[uri]
	}
	if receipt, ok := s.diagnostics[uri]; ok && receipt.version > version {
		return
	}
	s.diagnostics[uri] = javaDiagnosticReceipt{version: version, diagnostics: diagnostics}
}

func (s *javaProcessSession) recordDocumentVersion(uri string, version int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.documentVersions == nil {
		s.documentVersions = make(map[string]int)
	}
	if s.documentVersions[uri] < version {
		s.documentVersions[uri] = version
	}
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
	documents   map[string]javaDocumentState
	sessions    map[string]javaSessionRecord
}

// TrustedWorkspaceRoot discovers the workspace used for trust comparison.
func (b *JavaBackend) TrustedWorkspaceRoot(project backend.ProjectContext) (string, error) {
	return javaWorkspaceRoot(project)
}

// NotifyProjectConfigurationUpdate informs an already-running trusted JDT LS session of POM edits.
func (b *JavaBackend) NotifyProjectConfigurationUpdate(ctx context.Context, project backend.ProjectContext, changedPOMs []string) error {
	root, err := javaWorkspaceRoot(project)
	if err != nil {
		return err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return &backend.WorkspaceTrustError{Operation: backend.OperationAddBuildDependency, Language: backend.LanguageJava, Workspace: root}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var session JavaSession
	if b.root == root {
		session = b.session
	} else if retained, ok := b.sessions[root]; ok {
		session = retained.session
	}
	if session == nil {
		return nil
	}
	changes := make([]map[string]any, 0, len(changedPOMs))
	for _, path := range changedPOMs {
		changes = append(changes, map[string]any{"uri": pathutil.FileURI(path), "type": 2})
	}
	if err := session.Notify(ctx, "workspace/didChangeWatchedFiles", map[string]any{"changes": changes}); err != nil {
		return fmt.Errorf("notify JDT LS of Maven POM changes: %w", err)
	}
	for _, path := range changedPOMs {
		if err := session.Notify(ctx, "java/projectConfigurationUpdate", map[string]string{"uri": pathutil.FileURI(path)}); err != nil {
			return fmt.Errorf("notify JDT LS to refresh Maven POM %q: %w", path, err)
		}
	}
	return nil
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
	return backend.NewCapabilitiesRequiringWorkspaceTrust(backend.OperationLookup, backend.OperationRename, backend.OperationVerify, backend.OperationOrganizeImports, backend.OperationAddBuildDependency, backend.OperationOutline, backend.OperationInspect, backend.OperationFindReferences, backend.OperationInspectConstruct, backend.OperationInsertConstruct, backend.OperationReplaceConstruct, backend.OperationReplaceBody, backend.OperationReplaceDecl, backend.OperationRemoveConstruct, backend.OperationExtractFunction, backend.OperationExtractVariable, backend.OperationExtractConstant, backend.OperationInlineVariable, backend.OperationInlineFunction, backend.OperationImplementContract, backend.OperationChangeSignature, backend.OperationIntroduceParameter, backend.OperationIntroduceParameterObject, backend.OperationMoveConstruct)
}

// SupportedConstructs reports construct kinds that Java supports for an operation.
func (*JavaBackend) SupportedConstructs(operation backend.Operation) []backend.ConstructKind {
	if operation == backend.OperationInsertConstruct {
		return []backend.ConstructKind{
			backend.ConstructLoop, backend.ConstructIf, javacst.ConstructSwitch,
			backend.ConstructTryWithResources, backend.ConstructSynchronized, backend.ConstructStatementOrExpr, javacst.ConstructElement,
		}
	}
	if operation == backend.OperationRemoveConstruct {
		return []backend.ConstructKind{
			backend.ConstructMethod, backend.ConstructConstructor, backend.ConstructField, backend.ConstructType,
			backend.ConstructClass, backend.ConstructInterface, backend.ConstructEnum,
			backend.ConstructLoop, backend.ConstructIf, backend.ConstructElse,
			backend.ConstructCase, javacst.ConstructSwitch, backend.ConstructTryCatch,
			backend.ConstructTryWithResources, backend.ConstructSynchronized, backend.ConstructStatementOrExpr,
		}
	}
	if operation != backend.OperationInspectConstruct && operation != backend.OperationReplaceConstruct {
		return nil
	}
	return []backend.ConstructKind{
		backend.ConstructLoop, backend.ConstructIf, backend.ConstructElse, backend.ConstructCase, javacst.ConstructSwitch,
		backend.ConstructTryCatch, backend.ConstructTryWithResources, backend.ConstructSynchronized,
		backend.ConstructStatementOrExpr, javacst.ConstructElement,
	}
}

// CapabilityMatrix returns the declarative documentation matrix for the Java backend.
func (*JavaBackend) CapabilityMatrix() backend.LanguageMatrix {
	operations := map[string]backend.OpCapability{
		"scaffold_file":              {Supported: true, Description: "Create a Java source file with a package inferred from sibling .java files or a Maven source-root path.", MCPTool: "semantic_scaffold_file", PlacementKey: false},
		"set_doc_comment":            {Supported: true, Description: "Set or remove the attached Javadoc for one uniquely selected Java declaration; JDT LS formats and verifies the selected file.", MCPTool: "semantic_set_doc_comment", PlacementKey: false},
		"insert_field":               {Supported: true, Description: "Insert one Java field into a uniquely selected class, enum, or record; member ordering follows static/instance tier and visibility, then JDT LS formats and verifies the file.", MCPTool: "semantic_insert_field", PlacementKey: true},
		"insert_function":            {Supported: true, Description: "Insert one Java method or constructor into a uniquely selected owner type, ordered by member tier and visibility, then format and verify through JDT LS.", MCPTool: "semantic_insert_function", PlacementKey: true},
		"insert_type":                {Supported: true, Description: "Insert one Java nested class, interface, enum, or record into a uniquely selected owner type, ordered by visibility, then format and verify through JDT LS.", MCPTool: "semantic_insert_type", PlacementKey: true},
		"extract_function":           {Supported: true, Description: "Delegate one contiguous direct-statement selection in a Java method to JDT LS refactor.extract.function, rename the generated helper to the requested method name, then format and verify.", MCPTool: "semantic_extract_function", PlacementKey: false},
		"implement_contract":         {Supported: true, Description: "Ensure a Java class implements the selected interface, then delegate missing method generation to JDT LS source.overrideMethods and verify.", MCPTool: "semantic_implement_contract", PlacementKey: false},
		"extract_interface":          {Supported: true, Description: "Extract selected public instance methods from a Java class into a package-private interface in the same source file, then format and verify through JDT LS.", MCPTool: "semantic_extract_interface", PlacementKey: false},
		"extract_variable":           {Supported: true, Description: "Delegate exact Java expression extraction to a JDT LS code action, rename its generated local to the requested symbol, then format and verify the source.", MCPTool: "semantic_extract_variable", PlacementKey: false},
		"extract_constant":           {Supported: true, Description: "Delegate exact Java compile-time constant extraction to a JDT LS code action, rename its generated constant, then format and verify the source.", MCPTool: "semantic_extract_constant", PlacementKey: false},
		"inline_variable":            {Supported: true, Description: "Delegate one selected Java local declaration to JDT LS refactor.inline, then format and verify the inlined result.", MCPTool: "semantic_inline_variable", PlacementKey: false},
		"inline_function":            {Supported: true, Description: "Delegate one selected Java method invocation to JDT LS refactor.inline after resolving the call-site expression in its owner method.", MCPTool: "semantic_inline_function", PlacementKey: false},
		"change_signature":           {Supported: true, Description: "Delegate a complete ordered Java method-parameter mapping to JDT LS refactor.change.signature, apply the returned workspace edit across callers, and verify every affected Java file.", MCPTool: "semantic_change_signature", PlacementKey: false},
		"introduce_parameter":        {Supported: true, Description: "Replace one exact expression in a Java method with a new parameter, delegate signature and caller updates to JDT LS, then format and verify every affected file.", MCPTool: "semantic_introduce_parameter", PlacementKey: false},
		"introduce_parameter_object": {Supported: true, Description: "Replace a Java method's complete parameter list with a package-private value object, rewrite bound body references, and update proven same-package direct callers through JDT LS.", MCPTool: "semantic_introduce_parameter_object", PlacementKey: false},
		"move_construct":             {Supported: true, Description: "Move one Java static method, field, or nested type to a selected destination type through JDT LS, updating imports and references across the workspace.", MCPTool: "semantic_move_construct", PlacementKey: false},
		"verify":                     {Supported: true, Description: "Trusted Java file or directory-scope formatting, source.organizeImports, and diagnostics through JDT LS; Maven/Gradle are never executed.", MCPTool: "semantic_verify"},
		"organize_imports":           {Supported: true, Description: "Organize imports in one trusted Java file through JDT LS; explicit additions and removals are applied in the verified source buffer.", CLICommand: "semedit imports --language java --trust-workspace --jdtls-home <path> --file <path.java>", MCPTool: "semantic_organize_imports", PlacementKey: false},
		"add_build_dependency":       {Supported: true, Description: "Update trusted Maven reactor POMs with dependencyManagement-aware edits; does not run Maven and refreshes an active JDT LS project.", CLICommand: "semedit add-build-dependency --language java --trust-workspace --dependency <group:artifact[:version][:scope][:classifier]> [--file <module-or-pom>]", MCPTool: "semantic_add_build_dependency", PlacementKey: false},
		"inspect_construct":          {Supported: true, Description: "Inspect Java method-local control-flow and expression candidates with a pure-Go Tree-sitter Java CST locator; this operation is read-only.", MCPTool: "semantic_inspect_construct", PlacementKey: false},
		"insert_construct":           {Supported: true, Description: "Insert one Java method-local construct at an explicit anchor or append an element to a selected Map.of, List.of, Set.of, or array initializer; then format and verify through JDT LS.", MCPTool: "semantic_insert_construct", PlacementKey: false},
		"insert_case":                {Supported: true, Description: "Insert one Java case clause into an exact switch in a selected method, with candidate-set revision-checked ambiguity selection and JDT LS formatting and diagnostics.", MCPTool: "semantic_insert_case", PlacementKey: false},
		"replace_body":               {Supported: true, Description: "Replace the statements in one Java method body while preserving the declaration signature; JDT LS formats, organizes imports, and verifies the result.", MCPTool: "semantic_replace_body", PlacementKey: false},
		"replace_decl":               {Supported: true, Description: "Replace one Java method, field, or nested type declaration while preserving its kind and name, then format and verify through JDT LS.", MCPTool: "semantic_replace_decl", PlacementKey: false},
		"replace_construct":          {Supported: true, Description: "Replace one selected Java method-local construct by its inspected Tree-sitter path, then format, organize imports, and verify through JDT LS.", MCPTool: "semantic_replace_construct", PlacementKey: false},
		"remove_construct":           {Supported: true, Description: "Remove one Java method, constructor, field, nested type, or method-local control-flow construct; safe_delete requires a private declaration and complete JDT LS workspace reference coverage with no usages.", MCPTool: "semantic_remove_construct", PlacementKey: false},
		"rename": {
			Supported:    true,
			Description:  "Trusted JDT LS semantic rename applies validated text edits across Java files and supports one standalone public-type RenameFile operation; buffered resource renames are refused.",
			CLICommand:   "semedit rename --language java --trust-workspace --jdtls-home <path> --java-bin <path> --file <path.java> --symbol <sym> --to <name>",
			MCPTool:      "semantic_rename",
			PlacementKey: false,
		},
		"lookup": {
			Supported:    true,
			Description:  "File- or directory-scoped hierarchical Java symbol lookup through a trusted, preinstalled JDT LS session using UTF-16 LSP positions; overloads return signature-qualified candidates.",
			CLICommand:   "semedit lookup --language java --file <path.java-or-directory> --symbol <sym> --jdtls-home <path>",
			MCPTool:      "resolve_symbol_location",
			PlacementKey: false,
		},
		"outline": {
			Supported:    true,
			Description:  "Trusted Java declaration outline for a selected file or directory, with optional public-only and test-source filtering, from validated hierarchical JDT LS document symbols.",
			CLICommand:   "semedit outline --language java --path <path.java-or-directory> --trust-workspace --jdtls-home <path>",
			MCPTool:      "semantic_outline",
			Level:        "symbol",
			ReadOnly:     true,
			PlacementKey: false,
		},
		"inspect_symbol": {
			Supported:    true,
			Description:  "Trusted file- or directory-scoped Java source inspection for all matching hierarchical declarations and overloads.",
			CLICommand:   "semedit inspect-symbol --language java --file <path.java> --symbol <sym> --trust-workspace --jdtls-home <path>",
			MCPTool:      "semantic_inspect_symbol",
			Level:        "symbol",
			ReadOnly:     true,
			PlacementKey: false,
		},
		"find_references": {
			Supported:    true,
			Description:  "Trusted JDT LS typed references for a uniquely selected declaration, validated to remain inside the workspace.",
			CLICommand:   "semedit find-references --language java --file <path.java> --symbol <sym> --trust-workspace --jdtls-home <path>",
			MCPTool:      "semantic_find_references",
			Level:        "symbol",
			ReadOnly:     true,
			PlacementKey: false,
		},
	}
	maps.Copy(operations, javaWorkspaceOperationCapabilities())
	return backend.LanguageMatrix{
		Language: "java", DisplayName: "Java", Maturity: "Selected-file refactoring preview",
		Operations: operations, Limitations: javaCapabilityLimitations(),
	}
}

func javaWorkspaceOperationCapabilities() map[string]backend.OpCapability {
	return map[string]backend.OpCapability{
		"workspace_outline": {Supported: true, Description: "Read-only Java package and Maven reactor outline with package-info or primary-type Javadocs as synopses.", MCPTool: "semantic_workspace_outline", Level: "workspace", ReadOnly: true, PlacementKey: false},
	}
}

func javaCapabilityLimitations() []backend.Constraint {
	return []backend.Constraint{
		{Title: "Selected-File Java Inspection", Description: "Java reference discovery requires one uniquely selected declaration in a .java file; outline, lookup, symbol inspection, and verification also support directory scopes.", Severity: "error"},
		{Title: "Java Resource Rename Limits", Description: "RenameFile supports standalone public top-level type moves within the same package directory; resource renames inside buffered batches are refused. Extraction, inline, move, signature, parameter, and contract refactorings are delegated to JDT LS and may be limited by the installed server version.", Severity: "error"},
		{Title: "Trusted Explicit Workspace", Description: "JDT LS lookup, reads, and edits require explicit trust and a preinstalled distribution; Maven process actions are registered separately and are not JDT LS capabilities. Gradle import remains disabled.", Severity: "error"},
		{Title: "Hierarchical Document Symbols", Description: "Only validated hierarchical package, type, field, method, and constructor document symbols are returned; local variables and generated symbols are not part of the selected-file declaration tree.", Severity: "info"},
	}
}

// Close terminates the managed JDT LS server, if one is active.
func (b *JavaBackend) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var closeErr error
	if b.session != nil {
		closeErr = errors.Join(closeErr, b.session.Close())
	}
	for root, retained := range b.sessions {
		if retained.session != nil {
			if err := retained.session.Close(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("close Java session for %s: %w", root, err))
			}
		}
	}
	b.session, b.root, b.fingerprint = nil, "", ""
	b.documents = nil
	clear(b.sessions)
	return closeErr
}

// Lookup resolves one exact hierarchical symbol in the selected Java file.
func (b *JavaBackend) Lookup(ctx context.Context, project backend.ProjectContext, query string) (*backend.LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(project.File) != "" {
		root, target, directory, selectionErr := resolveJavaReadSelection(project, project.File)
		if selectionErr != nil {
			return nil, selectionErr
		}
		if directory {
			return b.lookupJavaDirectory(ctx, project, root, target, query)
		}
	}
	root, file, source, err := resolveJavaProject(ctx, project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationLookup, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: err}
	}
	if strings.TrimSpace(query) == "" {
		return nil, &JavaError{Op: "lookup", File: file, Symbol: query, Err: ErrJavaMalformedResponse}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); sessionErr != nil {
		return nil, sessionErr
	}
	symbols, err := b.readJavaSymbolsLocked(ctx, root, file, source)
	if err != nil {
		return nil, err
	}
	return selectJavaSymbol(query, file, symbols)
}

func (b *JavaBackend) lookupJavaDirectory(ctx context.Context, project backend.ProjectContext, root, target, query string) (*backend.LookupResult, error) {
	root = backend.CanonicalWorkspaceRoot(root)
	target = backend.CanonicalWorkspaceRoot(target)
	if !pathutil.PathWithin(root, target) {
		return nil, &JavaError{Op: "lookup", File: target, Workspace: root, Err: ErrJavaFileOutsideWorkspace}
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationLookup, Language: backend.LanguageJava, Workspace: root}
	}
	files, err := javaSourceFiles(target, root)
	if err != nil {
		return nil, err
	}
	config := effectiveJavaConfig(project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: err}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); err != nil {
		return nil, err
	}
	candidates := make([]*backend.NamedTarget, 0)
	for _, file := range files {
		fileProject := project
		fileProject.RootDir, fileProject.File = root, file
		_, selected, source, readErr := resolveJavaProject(ctx, fileProject)
		if readErr != nil {
			return nil, readErr
		}
		symbols, readErr := b.readJavaSymbolsLocked(ctx, root, selected, source)
		if readErr != nil {
			return nil, readErr
		}
		lookup, lookupErr := selectJavaSymbol(query, selected, symbols)
		if errors.Is(lookupErr, ErrJavaSymbolNotFound) {
			continue
		}
		if lookupErr != nil {
			return nil, lookupErr
		}
		if lookup.Definition != nil {
			candidates = append(candidates, lookup.Definition)
		} else {
			candidates = append(candidates, lookup.Candidates...)
		}
	}
	if len(candidates) == 0 {
		return nil, &JavaError{Op: "lookup", File: target, Symbol: query, Err: ErrJavaSymbolNotFound}
	}
	if len(candidates) == 1 {
		return &backend.LookupResult{Definition: candidates[0]}, nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Location.URI != candidates[j].Location.URI {
			return candidates[i].Location.URI < candidates[j].Location.URI
		}
		return candidates[i].QualifiedName < candidates[j].QualifiedName
	})
	return &backend.LookupResult{Candidates: candidates}, nil
}

type javaLocation struct {
	URI   string    `json:"uri"`
	Range javaRange `json:"range"`
}

const (
	javaOperationFingerprint = "fingerprint"
	javaOperationReferences  = "references"
)

// FindReferences resolves the selected declaration and asks JDT LS for its uses.
func (b *JavaBackend) FindReferences(ctx context.Context, request backend.ReferencesRequest) (*backend.ReferencesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, resolveErr := resolveJavaProject(ctx, request.Project)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationFindReferences, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, config)
	if fingerprintErr != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: fingerprintErr}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); err != nil {
		return nil, err
	}
	symbols, err := b.readJavaSymbolsLocked(ctx, root, file, source)
	if err != nil {
		return nil, err
	}
	lookup, err := selectJavaSymbol(request.Symbol, file, symbols)
	if err != nil {
		return nil, err
	}
	if lookup.IsAmbiguous() || lookup.Definition == nil {
		return nil, &JavaError{Op: javaOperationReferences, File: file, Symbol: request.Symbol, Err: ErrJavaSymbolAmbiguous}
	}
	params := map[string]any{
		lspTextDocumentKey: map[string]string{"uri": lookup.Definition.Location.URI},
		"position":         lookup.Definition.Location.Range.Start,
		"context":          map[string]bool{"includeDeclaration": false},
	}
	var locations []javaLocation
	err = b.withJavaDocumentLocked(ctx, file, source, func() error {
		raw, requestErr := b.session.Request(ctx, "textDocument/references", params)
		if requestErr != nil {
			return &JavaError{Op: javaOperationReferences, File: file, Err: requestErr}
		}
		if len(raw) == 0 || string(raw) == "null" {
			return nil
		}
		if decodeErr := json.Unmarshal(raw, &locations); decodeErr != nil {
			return &JavaError{Op: javaOperationReferences, File: file, Err: fmt.Errorf("%w: %w", ErrJavaMalformedResponse, decodeErr)}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := &backend.ReferencesResult{
		Scope: backend.ReadScope{
			Kind: "workspace", Path: root, Complete: false,
			Limitations: []string{"JDT LS does not expose whether its workspace reference index is complete."},
		},
		Symbol: backend.ReadSymbol{
			Name: lookup.Definition.Name, QualifiedName: lookup.Definition.QualifiedName,
			Kind: lookup.Definition.Kind, File: file, Range: lookup.Definition.Location.Range,
			SelectionRange: lookup.Definition.Location.Range,
		},
		References: make([]backend.Reference, 0, len(locations)),
	}
	files := map[string][32]byte{file: sha256.Sum256(source)}
	for _, location := range locations {
		reference, err := appendJavaReference(ctx, root, file, source, location, files)
		if err != nil {
			return nil, err
		}
		result.References = append(result.References, reference)
	}
	result.Files = javaReferenceFileRevisions(files)
	return result, nil
}

func javaReferenceFileRevisions(files map[string][32]byte) []backend.FileRevision {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	revisions := make([]backend.FileRevision, 0, len(paths))
	for _, path := range paths {
		digest := files[path]
		revisions = append(revisions, backend.FileRevision{File: path, Revision: hex.EncodeToString(digest[:])})
	}
	return revisions
}

func appendJavaReference(ctx context.Context, root, sourceFile string, source []byte, location javaLocation, files map[string][32]byte) (backend.Reference, error) {
	refFile, err := pathutil.FilePathFromURI(location.URI)
	if err != nil || !pathutil.PathWithin(root, refFile) {
		return backend.Reference{}, &JavaError{Op: javaOperationReferences, File: refFile, Err: ErrJavaMalformedResponse}
	}
	refSource := source
	if refFile != sourceFile {
		refSource, err = sourcebuf.ReadFile(ctx, refFile)
		if err != nil {
			return backend.Reference{}, &JavaError{Op: "references read", File: refFile, Err: err}
		}
		files[refFile] = sha256.Sum256(refSource)
	}
	if !validJavaRange(location.Range, refSource) {
		return backend.Reference{}, &JavaError{Op: javaOperationReferences, File: refFile, Err: ErrJavaMalformedResponse}
	}
	start, startErr := javaByteOffset(refSource, location.Range.Start)
	end, endErr := javaByteOffset(refSource, location.Range.End)
	if startErr != nil || endErr != nil || start > end {
		return backend.Reference{}, &JavaError{Op: javaOperationReferences, File: refFile, Err: ErrJavaMalformedResponse}
	}
	return backend.Reference{
		File: refFile, Range: backend.Range{Start: backend.Position(location.Range.Start), End: backend.Position(location.Range.End)},
		Snippet: string(refSource[start:end]),
	}, nil
}

// Rename executes one trusted JDT LS rename transaction across validated workspace files.
func (b *JavaBackend) Rename(ctx context.Context, request backend.RenameRequest) (result *backend.RenameResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, resolveErr := resolveJavaProject(ctx, request.Project)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationRename, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, config)
	if fingerprintErr != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: fingerprintErr}
	}
	if strings.TrimSpace(request.Symbol) == "" || strings.TrimSpace(request.To) == "" {
		return nil, &backend.Error{Operation: backend.OperationRename, Language: backend.LanguageJava, Err: ErrJavaRenameInvalidEdit}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); sessionErr != nil {
		return nil, sessionErr
	}
	session := b.session
	defer func() {
		if retErr == nil || b.session == nil {
			return
		}
		retErr = errors.Join(retErr, b.invalidateJavaSessionLocked(session))
	}()
	execution, err := b.prepareJavaRenameExecution(ctx, session, root, file, source, request)
	if err != nil {
		return nil, err
	}
	return b.applyJavaRenameExecution(ctx, session, root, file, request, execution)
}

// ValidateRenameTarget confirms that JDT LS can safely prepare a rename without publishing source changes.
func (b *JavaBackend) ValidateRenameTarget(ctx context.Context, request backend.RenameRequest) (retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request.Symbol = backend.NormalizeRenameInput(request.Symbol)
	request.To = backend.NormalizeRenameInput(request.To)
	root, file, source, resolveErr := resolveJavaProject(ctx, request.Project)
	if resolveErr != nil {
		return resolveErr
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return &backend.WorkspaceTrustError{Operation: backend.OperationRename, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, config)
	if fingerprintErr != nil {
		return &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: fingerprintErr}
	}
	if request.Symbol == "" || request.To == "" {
		return &backend.Error{Operation: backend.OperationRename, Language: backend.LanguageJava, Err: ErrJavaRenameInvalidEdit}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); sessionErr != nil {
		return sessionErr
	}
	session := b.session
	defer func() {
		if retErr == nil || b.session == nil {
			return
		}
		retErr = errors.Join(retErr, b.invalidateJavaSessionLocked(session))
	}()
	_, err := b.prepareJavaRenameTarget(ctx, ctx, session, root, file, source, request, sourcebuf.FromContext(ctx))
	return err
}

type javaRenameExecution struct {
	edit          javaRenameEdit
	buffer        *sourcebuf.Buffer
	workingBuffer *sourcebuf.Buffer
	workingCtx    context.Context
	beforeErrors  []string
}

func (b *JavaBackend) prepareJavaRenameExecution(ctx context.Context, session JavaSession, root, file string, source []byte, request backend.RenameRequest) (javaRenameExecution, error) {
	buffer := sourcebuf.FromContext(ctx)
	workingBuffer, workingCtx := buffer, ctx
	if workingBuffer == nil {
		createdBuffer, err := sourcebuf.New(ctx, root)
		if err != nil {
			return javaRenameExecution{}, &JavaError{Op: "rename buffer", Workspace: root, Err: err}
		}
		workingBuffer = createdBuffer
		if err := workingBuffer.AuthorizeWrites(ctx, root); err != nil {
			return javaRenameExecution{}, &JavaError{Op: "rename buffer", Workspace: root, Err: err}
		}
		workingCtx = sourcebuf.WithBuffer(ctx, workingBuffer)
	}
	edit, err := b.prepareJavaRenameTarget(ctx, workingCtx, session, root, file, source, request, buffer)
	if err != nil {
		return javaRenameExecution{}, err
	}
	before, err := b.javaRenameDiagnostics(ctx, session, root, edit, "rename diagnostics before", request.DeferVerification)
	if err != nil {
		return javaRenameExecution{}, err
	}
	return javaRenameExecution{edit: edit, buffer: buffer, workingBuffer: workingBuffer, workingCtx: workingCtx, beforeErrors: before}, nil
}

func (b *JavaBackend) prepareJavaRenameTarget(ctx, documentCtx context.Context, session JavaSession, root, file string, source []byte, request backend.RenameRequest, buffer *sourcebuf.Buffer) (javaRenameEdit, error) {
	if err := b.syncBufferedJavaDocuments(ctx, buffer); err != nil {
		return javaRenameEdit{}, err
	}
	if err := b.syncJavaDocumentLocked(documentCtx, file, source); err != nil {
		return javaRenameEdit{}, err
	}
	if err := b.saveCommittedJavaDocumentsLocked(ctx); err != nil {
		return javaRenameEdit{}, err
	}
	edit, err := prepareJavaRenameEdit(ctx, session, root, file, source, request.Symbol, request.To)
	if err != nil {
		return javaRenameEdit{}, err
	}
	if len(edit.renames) > 0 && buffer != nil {
		return javaRenameEdit{}, ErrJavaBatchResourceRenameUnsupported
	}
	for _, fileEdit := range edit.files {
		if err := b.syncJavaDocumentLocked(documentCtx, fileEdit.file, fileEdit.before); err != nil {
			return javaRenameEdit{}, err
		}
		if fileEdit.version != nil && b.documents[fileEdit.uri].version != *fileEdit.version {
			return javaRenameEdit{}, &JavaError{Op: "rename", File: fileEdit.file, Err: ErrJavaRenameStaleEdit}
		}
	}
	return edit, nil
}

func (b *JavaBackend) syncBufferedJavaDocuments(ctx context.Context, buffer *sourcebuf.Buffer) error {
	if buffer == nil {
		return nil
	}
	for _, change := range buffer.Changes() {
		if strings.EqualFold(filepath.Ext(change.Path), ".java") {
			if err := b.syncJavaDocumentLocked(ctx, change.Path, change.After); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *JavaBackend) javaRenameDiagnostics(ctx context.Context, session JavaSession, root string, edit javaRenameEdit, operation string, deferred bool) ([]string, error) {
	if deferred {
		return nil, nil
	}
	var messages []string
	for _, fileEdit := range edit.files {
		state := b.documents[fileEdit.uri]
		diagnostics, err := waitJavaVerifyDiagnostics(ctx, session, fileEdit.file, state.version)
		if err != nil {
			return nil, &JavaError{Op: operation, File: fileEdit.file, Err: err}
		}
		messages = append(messages, javaDiagnosticMessages(root, diagnostics)...)
	}
	sort.Strings(messages)
	return messages, nil
}

func (b *JavaBackend) applyJavaRenameExecution(ctx context.Context, session JavaSession, root, file string, request backend.RenameRequest, execution javaRenameExecution) (*backend.RenameResult, error) {
	result := &backend.RenameResult{Lookup: execution.edit.lookup}
	if err := b.applyJavaRenameBuffer(ctx, session, file, execution); err != nil {
		return result, err
	}
	if len(execution.edit.renames) > 0 {
		if execution.buffer != nil {
			return result, ErrJavaBatchResourceRenameUnsupported
		}
		for _, rename := range execution.edit.renames {
			if err := b.publishJavaResourceRename(ctx, session, rename, &execution.edit); err != nil {
				return result, &JavaError{Op: "rename Java resource", File: rename.oldPath, Err: err}
			}
			result.TargetFile = rename.newPath
		}
	}
	after, err := b.javaRenameDiagnostics(ctx, session, root, execution.edit, "rename diagnostics after", request.DeferVerification)
	if err != nil {
		return result, err
	}
	if !request.DeferVerification {
		delta := pipeline.ComputeDelta(execution.beforeErrors, after)
		result.Diagnostics = backend.DiagnosticDelta{Before: delta.Before, After: delta.After, NetDelta: delta.NetDelta, Introduced: delta.Introduced, Resolved: delta.Resolved, Suggestions: delta.Suggestions}
		result.Active = after
	}
	return result, nil
}

func (b *JavaBackend) applyJavaRenameBuffer(ctx context.Context, session JavaSession, file string, execution javaRenameExecution) error {
	changes := make([]sourcebuf.Change, 0, len(execution.edit.files))
	for _, fileEdit := range execution.edit.files {
		changes = append(changes, sourcebuf.Change{Path: fileEdit.file, Before: fileEdit.before, After: fileEdit.after})
	}
	if err := execution.workingBuffer.Apply(execution.workingCtx, changes); err != nil {
		return &JavaError{Op: "buffer rename", File: file, Err: err}
	}
	if execution.buffer == nil {
		if err := execution.workingBuffer.Flush(ctx); err != nil {
			return &JavaError{Op: "publish rename", File: file, Err: err}
		}
	}
	for _, fileEdit := range execution.edit.files {
		if err := b.syncJavaDocumentLocked(ctx, fileEdit.file, fileEdit.after); err != nil {
			if execution.buffer == nil {
				_ = b.invalidateJavaSessionLocked(session)
			}
			return err
		}
		if err := b.saveJavaRenameDocument(ctx, session, execution.buffer, fileEdit); err != nil {
			return err
		}
	}
	return nil
}

func (b *JavaBackend) saveJavaRenameDocument(ctx context.Context, session JavaSession, buffer *sourcebuf.Buffer, fileEdit javaWorkspaceFileEdit) error {
	uri := pathutil.FileURI(fileEdit.file)
	if buffer == nil {
		if err := session.Notify(ctx, "textDocument/didSave", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}}); err != nil {
			return &JavaError{Op: "didSave", File: fileEdit.file, Err: errors.Join(err, b.invalidateJavaSessionLocked(session))}
		}
		return nil
	}
	state := b.documents[uri]
	state.pendingSave = true
	b.documents[uri] = state
	return nil
}

func (b *JavaBackend) publishJavaResourceRename(ctx context.Context, session JavaSession, rename javaResourceRename, edit *javaRenameEdit) error {
	var selected *javaWorkspaceFileEdit
	for i := range edit.files {
		if edit.files[i].file == rename.oldPath {
			selected = &edit.files[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("RenameFile has no validated selected-file edit: %w", ErrJavaRenameInvalidEdit)
	}
	diskCtx := sourcebuf.WithoutBuffer(ctx)
	if err := pipeline.WriteAtomicContext(diskCtx, rename.newPath, selected.after); err != nil {
		return fmt.Errorf("write renamed Java file: %w", err)
	}
	if err := os.Remove(rename.oldPath); err != nil {
		removeErr := os.Remove(rename.newPath)
		return errors.Join(fmt.Errorf("remove old Java file: %w", err), removeErr)
	}
	if err := session.Notify(ctx, "workspace/didRenameFiles", map[string]any{"files": []map[string]string{{"oldUri": rename.oldURI, "newUri": rename.newURI}}}); err != nil {
		return fmt.Errorf("notify JDT LS of Java file rename: %w", err)
	}
	state, ok := b.documents[rename.oldURI]
	if !ok {
		return fmt.Errorf("renamed Java document is absent from session state: %w", ErrJavaRenameInvalidEdit)
	}
	delete(b.documents, rename.oldURI)
	if err := session.Notify(ctx, "textDocument/didClose", map[string]any{lspTextDocumentKey: map[string]string{"uri": rename.oldURI}}); err != nil {
		return fmt.Errorf("close old Java document after resource rename: %w", err)
	}
	state.version++
	state.digest = sha256.Sum256(selected.after)
	state.pendingSave = false
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{lspTextDocumentKey: map[string]any{
		"uri": rename.newURI, "languageId": "java", "version": state.version, "text": string(selected.after),
	}}); err != nil {
		return fmt.Errorf("open renamed Java document: %w", err)
	}
	if processSession, ok := session.(*javaProcessSession); ok {
		processSession.recordDocumentVersion(rename.newURI, state.version)
	}
	b.documents[rename.newURI] = state
	selected.file, selected.uri = rename.newPath, rename.newURI
	return nil
}

type javaRenameEdit struct {
	lookup  *backend.LookupResult
	files   []javaWorkspaceFileEdit
	renames []javaResourceRename
}

type javaResourceRename struct {
	oldURI, newURI   string
	oldPath, newPath string
}

type javaWorkspaceEdit struct {
	files   []javaWorkspaceFileEdit
	renames []javaResourceRename
}

type javaWorkspaceDocument struct {
	edits   []javaTextEdit
	version *int
}

type javaWorkspaceFileEdit struct {
	file, uri string
	version   *int
	before    []byte
	after     []byte
}

func prepareJavaRenameEdit(ctx context.Context, session JavaSession, root, file string, source []byte, symbol, newName string) (javaRenameEdit, error) {
	raw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{lspTextDocumentKey: map[string]string{"uri": pathutil.FileURI(file)}})
	if err != nil {
		return javaRenameEdit{}, &JavaError{Op: lspDocumentSymbolKey, File: file, Err: err}
	}
	symbols, err := decodeJavaDocumentSymbols(raw, root, source)
	if err != nil {
		return javaRenameEdit{}, &JavaError{Op: lspDocumentSymbolKey, File: file, Err: err}
	}
	if uriErr := validateJavaDocumentSymbolURI(symbols, pathutil.FileURI(file)); uriErr != nil {
		return javaRenameEdit{}, &JavaError{Op: lspDocumentSymbolKey, File: file, Err: uriErr}
	}
	lookup, err := selectJavaSymbol(symbol, file, symbols)
	if err != nil {
		return javaRenameEdit{}, err
	}
	if lookup.IsAmbiguous() || lookup.Definition == nil {
		return javaRenameEdit{}, &JavaError{Op: "rename", File: file, Symbol: symbol, Err: ErrJavaSymbolAmbiguous}
	}
	position := lookup.Definition.Location.Range.Start
	prepRaw, err := session.Request(ctx, "textDocument/prepareRename", map[string]any{lspTextDocumentKey: map[string]string{"uri": pathutil.FileURI(file)}, "position": position})
	if err != nil || !validJavaPrepareRename(prepRaw) {
		if err == nil {
			err = ErrJavaRenameInvalidEdit
		}
		return javaRenameEdit{}, &JavaError{Op: "prepareRename", File: file, Err: err}
	}
	editRaw, err := session.Request(ctx, "textDocument/rename", map[string]any{lspTextDocumentKey: map[string]string{"uri": pathutil.FileURI(file)}, "position": position, "newName": newName})
	if err != nil {
		return javaRenameEdit{}, &JavaError{Op: "rename", File: file, Err: err}
	}
	parts, _, _, parseErr := parseJavaSymbolQuery(symbol)
	if parseErr != nil {
		return javaRenameEdit{}, parseErr
	}
	oldName := parts[len(parts)-1]
	updated, err := prepareJavaWorkspaceEdits(ctx, root, file, source, editRaw, oldName, newName, sourcebuf.FromContext(ctx) == nil)
	if err != nil {
		return javaRenameEdit{}, &JavaError{Op: "rename", File: file, Err: err}
	}
	if len(updated.renames) == 0 && filepath.Base(file) == oldName+".java" {
		publicType, isPublic, typeErr := javacst.JavaPublicTopLevelTypeName(source)
		if typeErr != nil {
			return javaRenameEdit{}, fmt.Errorf("inspect Java public type for resource rename: %w", typeErr)
		}
		if isPublic && publicType == oldName {
			newPath := filepath.Join(filepath.Dir(file), newName+".java")
			if !pathutil.PathWithin(root, newPath) {
				return javaRenameEdit{}, fmt.Errorf("java public type rename target escapes workspace")
			}
			if _, statErr := os.Stat(newPath); statErr == nil {
				return javaRenameEdit{}, fmt.Errorf("java public type rename target %q already exists", newPath)
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return javaRenameEdit{}, fmt.Errorf("inspect Java public type rename target %q: %w", newPath, statErr)
			}
			updated.renames = append(updated.renames, javaResourceRename{
				oldURI: pathutil.FileURI(file), newURI: pathutil.FileURI(newPath), oldPath: file, newPath: newPath,
			})
		}
	}
	return javaRenameEdit{lookup: lookup, files: updated.files, renames: updated.renames}, nil
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

func prepareJavaWorkspaceEdits(ctx context.Context, root, selectedFile string, selectedSource []byte, raw json.RawMessage, oldName, newName string, allowResourceRenames bool) (javaWorkspaceEdit, error) {
	root = backend.CanonicalWorkspaceRoot(root)
	documents, renames, err := decodeJavaWorkspaceDocuments(root, raw, allowResourceRenames)
	if err != nil {
		return javaWorkspaceEdit{}, err
	}
	selectedFile = backend.CanonicalWorkspaceRoot(selectedFile)
	files, err := materializeJavaWorkspaceFiles(ctx, root, selectedFile, selectedSource, documents, oldName, newName)
	if err != nil {
		return javaWorkspaceEdit{}, err
	}
	if len(renames) > 1 {
		return javaWorkspaceEdit{}, fmt.Errorf("%w: multiple Java resource renames are unsupported", ErrJavaRenameInvalidEdit)
	}
	for _, rename := range renames {
		if err := validateJavaResourceRename(root, selectedFile, oldName, newName, files, rename); err != nil {
			return javaWorkspaceEdit{}, err
		}
	}
	return javaWorkspaceEdit{files: files, renames: renames}, nil
}

func decodeJavaWorkspaceDocuments(root string, raw json.RawMessage, allowResourceRenames bool) (map[string]javaWorkspaceDocument, []javaResourceRename, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, nil, fmt.Errorf("%w: workspace edit is not an object", ErrJavaRenameInvalidEdit)
	}
	_, hasChanges := object["changes"]
	_, hasDocuments := object["documentChanges"]
	if hasChanges == hasDocuments {
		return nil, nil, fmt.Errorf("%w: workspace edit must have exactly one document change form", ErrJavaRenameInvalidEdit)
	}
	for key := range object {
		if key != "changes" && key != "documentChanges" {
			return nil, nil, fmt.Errorf("%w: unsupported workspace edit field %q", ErrJavaRenameInvalidEdit, key)
		}
	}
	if hasChanges {
		documents, err := decodeJavaWorkspaceChanges(object["changes"])
		return documents, nil, err
	}
	return decodeJavaDocumentChanges(root, object["documentChanges"], allowResourceRenames)
}

func decodeJavaWorkspaceChanges(encoded json.RawMessage) (map[string]javaWorkspaceDocument, error) {
	documents := make(map[string]javaWorkspaceDocument)
	var changes map[string]json.RawMessage
	if json.Unmarshal(encoded, &changes) != nil || len(changes) == 0 {
		return nil, fmt.Errorf("%w: workspace changes are empty or malformed", ErrJavaRenameInvalidEdit)
	}
	for uri, rawEdits := range changes {
		edits, err := decodeJavaTextEdits(rawEdits)
		if err != nil {
			return nil, err
		}
		documents[uri] = javaWorkspaceDocument{edits: edits}
	}
	return documents, nil
}

func decodeJavaDocumentChanges(root string, encoded json.RawMessage, allowResourceRenames bool) (map[string]javaWorkspaceDocument, []javaResourceRename, error) {
	documents := make(map[string]javaWorkspaceDocument)
	var renames []javaResourceRename
	var changes []json.RawMessage
	if json.Unmarshal(encoded, &changes) != nil || len(changes) == 0 {
		return nil, nil, ErrJavaRenameInvalidEdit
	}
	for _, encoded := range changes {
		operation, err := decodeJavaDocumentChange(encoded)
		if err != nil {
			return nil, nil, err
		}
		if operation.rename != nil {
			if !allowResourceRenames {
				return nil, nil, ErrJavaBatchResourceRenameUnsupported
			}
			rename, renameErr := parseJavaResourceRename(root, operation.resource)
			if renameErr != nil {
				return nil, nil, renameErr
			}
			renames = append(renames, rename)
			continue
		}
		if _, duplicate := documents[operation.document.URI]; duplicate {
			return nil, nil, ErrJavaRenameInvalidEdit
		}
		documents[operation.document.URI] = javaWorkspaceDocument{edits: operation.edits, version: operation.document.Version}
	}
	return documents, renames, nil
}

type decodedJavaDocumentChange struct {
	document struct {
		URI     string `json:"uri"`
		Version *int   `json:"version"`
	}
	resource map[string]json.RawMessage
	rename   *struct{}
	edits    []javaTextEdit
}

func decodeJavaDocumentChange(encoded json.RawMessage) (decodedJavaDocumentChange, error) {
	var operation map[string]json.RawMessage
	if json.Unmarshal(encoded, &operation) != nil || operation == nil {
		return decodedJavaDocumentChange{}, ErrJavaRenameInvalidEdit
	}
	if kind, ok := operation["kind"]; ok {
		var kindName string
		if json.Unmarshal(kind, &kindName) != nil || kindName != "rename" {
			return decodedJavaDocumentChange{}, ErrJavaRenameInvalidEdit
		}
		return decodedJavaDocumentChange{resource: operation, rename: &struct{}{}}, nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(encoded, &fields) != nil || len(fields) != 2 || fields[lspTextDocumentKey] == nil || fields["edits"] == nil {
		return decodedJavaDocumentChange{}, fmt.Errorf("%w: malformed TextDocumentEdit fields %s", ErrJavaRenameInvalidEdit, encoded)
	}
	var payload struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version *int   `json:"version"`
		} `json:"textDocument"`
		Edits json.RawMessage `json:"edits"`
	}
	if json.Unmarshal(encoded, &payload) != nil || payload.TextDocument.URI == "" || (payload.TextDocument.Version != nil && *payload.TextDocument.Version < 0) {
		return decodedJavaDocumentChange{}, ErrJavaRenameInvalidEdit
	}
	edits, err := decodeJavaTextEdits(payload.Edits)
	if err != nil {
		return decodedJavaDocumentChange{}, err
	}
	decoded := decodedJavaDocumentChange{edits: edits}
	decoded.document.URI, decoded.document.Version = payload.TextDocument.URI, payload.TextDocument.Version
	return decoded, nil
}

func materializeJavaWorkspaceFiles(ctx context.Context, root, selectedFile string, selectedSource []byte, documents map[string]javaWorkspaceDocument, oldName, newName string) ([]javaWorkspaceFileEdit, error) {
	files := make([]javaWorkspaceFileEdit, 0, len(documents))
	selectedIncluded := false
	for uri, document := range documents {
		file, err := pathutil.FilePathFromURI(uri)
		if err != nil {
			return nil, fmt.Errorf("%w: edited URI %q is invalid", ErrJavaRenameInvalidEdit, uri)
		}
		file = backend.CanonicalWorkspaceRoot(file)
		if !pathutil.PathWithin(root, file) || !strings.EqualFold(filepath.Ext(file), ".java") {
			return nil, fmt.Errorf("%w: edited file %q is outside the workspace or is not Java", ErrJavaRenameInvalidEdit, file)
		}
		if file == selectedFile {
			selectedIncluded = true
		}
		source := selectedSource
		if file != selectedFile {
			source, err = sourcebuf.ReadFile(ctx, file)
			if err != nil {
				return nil, fmt.Errorf("read Java workspace edit preimage %q: %w", file, err)
			}
		}
		spans, err := validateJavaWorkspaceDocumentSpans(source, document.edits, oldName, newName)
		if err != nil {
			return nil, fmt.Errorf("validate edits for %s: %w", file, err)
		}
		files = append(files, javaWorkspaceFileEdit{file: file, uri: pathutil.FileURI(file), version: document.version, before: bytes.Clone(source), after: applyJavaEditSpans(source, spans)})
	}
	if !selectedIncluded {
		return nil, fmt.Errorf("%w: selected Java file is absent from workspace edit", ErrJavaRenameInvalidEdit)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].file < files[j].file })
	return files, nil
}

func validateJavaWorkspaceDocumentSpans(source []byte, edits []javaTextEdit, oldName, newName string) ([]javaEditSpan, error) {
	if oldName == "" {
		return validateJavaWorkspaceEditSpans(source, edits)
	}
	return validateJavaRenameSpans(source, edits, oldName, newName)
}

func parseJavaResourceRename(root string, fields map[string]json.RawMessage) (javaResourceRename, error) {
	for key := range fields {
		if key != "kind" && key != "oldUri" && key != "newUri" && key != "options" {
			return javaResourceRename{}, fmt.Errorf("%w: unsupported RenameFile field %q", ErrJavaRenameInvalidEdit, key)
		}
	}
	var operation struct {
		Kind    string          `json:"kind"`
		OldURI  string          `json:"oldUri"`
		NewURI  string          `json:"newUri"`
		Options json.RawMessage `json:"options"`
	}
	encoded, err := json.Marshal(fields)
	if err != nil || json.Unmarshal(encoded, &operation) != nil || operation.Kind != "rename" || operation.OldURI == "" || operation.NewURI == "" {
		return javaResourceRename{}, ErrJavaRenameInvalidEdit
	}
	if len(operation.Options) > 0 && string(operation.Options) != "null" {
		var options map[string]json.RawMessage
		if json.Unmarshal(operation.Options, &options) != nil || options == nil {
			return javaResourceRename{}, ErrJavaRenameInvalidEdit
		}
		for key, value := range options {
			if key != "overwrite" && key != "ignoreIfExists" {
				return javaResourceRename{}, fmt.Errorf("%w: unsupported RenameFile option %q", ErrJavaRenameInvalidEdit, key)
			}
			var enabled bool
			if json.Unmarshal(value, &enabled) != nil || enabled {
				return javaResourceRename{}, fmt.Errorf("%w: RenameFile option %q must be false", ErrJavaRenameInvalidEdit, key)
			}
		}
	}
	oldPath, err := pathutil.FilePathFromURI(operation.OldURI)
	if err != nil {
		return javaResourceRename{}, fmt.Errorf("%w: invalid RenameFile oldUri", ErrJavaRenameInvalidEdit)
	}
	newPath, err := pathutil.FilePathFromURI(operation.NewURI)
	if err != nil {
		return javaResourceRename{}, fmt.Errorf("%w: invalid RenameFile newUri", ErrJavaRenameInvalidEdit)
	}
	oldPath = canonicalJavaFilePath(oldPath)
	newPath = canonicalJavaFilePath(newPath)
	if !pathutil.PathWithin(root, oldPath) || !pathutil.PathWithin(root, newPath) {
		return javaResourceRename{}, fmt.Errorf("%w: RenameFile paths must stay inside the trusted workspace", ErrJavaRenameInvalidEdit)
	}
	return javaResourceRename{oldURI: pathutil.FileURI(oldPath), newURI: pathutil.FileURI(newPath), oldPath: oldPath, newPath: newPath}, nil
}

func validateJavaResourceRename(root, selectedFile, oldName, newName string, files []javaWorkspaceFileEdit, rename javaResourceRename) error {
	if oldName == "" && newName == "" {
		return validateJavaResourceMove(root, selectedFile, files, rename)
	}
	if rename.oldPath != selectedFile || rename.oldPath == rename.newPath || filepath.Dir(rename.oldPath) != filepath.Dir(rename.newPath) ||
		!strings.EqualFold(filepath.Ext(rename.oldPath), ".java") || !strings.EqualFold(filepath.Ext(rename.newPath), ".java") ||
		strings.TrimSuffix(filepath.Base(rename.oldPath), filepath.Ext(rename.oldPath)) != oldName ||
		strings.TrimSuffix(filepath.Base(rename.newPath), filepath.Ext(rename.newPath)) != newName ||
		!pathutil.PathWithin(root, rename.oldPath) || !pathutil.PathWithin(root, rename.newPath) {
		return fmt.Errorf("%w: Java public-type rename must remain in the same package directory and match the new type name", ErrJavaRenameInvalidEdit)
	}
	if !hasJavaPublicType(selectedFile, oldName) {
		return fmt.Errorf("%w: RenameFile is only accepted for the selected public top-level type", ErrJavaRenameInvalidEdit)
	}
	if _, err := os.Lstat(rename.newPath); err == nil {
		return fmt.Errorf("%w: Java rename destination already exists: %s", ErrJavaRenameInvalidEdit, rename.newPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect Java rename destination %q: %w", rename.newPath, err)
	}
	for _, file := range files {
		if file.file != rename.oldPath {
			continue
		}
		if !bytes.Contains(file.after, []byte("public ")) || !hasJavaPublicTypeSource(file.after, newName) {
			return fmt.Errorf("%w: RenameFile text edits do not rename the selected public type", ErrJavaRenameInvalidEdit)
		}
		return nil
	}
	return fmt.Errorf("%w: RenameFile has no matching text document edit", ErrJavaRenameInvalidEdit)
}

func hasJavaPublicType(file, name string) bool {
	source, err := os.ReadFile(file) // #nosec G304 -- the caller validates the selected file remains under the trusted workspace root.
	return err == nil && hasJavaPublicTypeSource(source, name)
}

func hasJavaPublicTypeSource(source []byte, name string) bool {
	if name == "" || strings.Contains(name, ".") || validateJavaPackageName(name) != nil {
		return false
	}
	pattern := regexp.MustCompile(`\bpublic\s+(?:(?:abstract|final|sealed|non-sealed|static)\s+)*(?:class|interface|enum|record)\s+` + regexp.QuoteMeta(name) + `\b`)
	return pattern.Match(source)
}

func validateJavaRenameSpans(source []byte, edits []javaTextEdit, oldName, newName string) ([]javaEditSpan, error) {
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
		expected, renameErr := javacst.RenameJavaIdentifiersInRange(source, start, end, oldName, newName)
		if renameErr != nil || !bytes.Equal(expected, []byte(edit.NewText)) {
			return nil, fmt.Errorf("%w: expected %q renamed to %q at [%d:%d-%d:%d], found %q", ErrJavaRenameStaleEdit, oldName, newName, edit.Range.Start.Line, edit.Range.Start.Character, edit.Range.End.Line, edit.Range.End.Character, source[start:end])
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
	return spans, nil
}

func validateJavaWorkspaceEditSpans(source []byte, edits []javaTextEdit) ([]javaEditSpan, error) {
	if len(edits) == 0 {
		return nil, ErrJavaRenameInvalidEdit
	}
	spans := make([]javaEditSpan, 0, len(edits))
	for _, edit := range edits {
		start, startErr := javaByteOffset(source, edit.Range.Start)
		end, endErr := javaByteOffset(source, edit.Range.End)
		if startErr != nil || endErr != nil || end < start {
			return nil, ErrJavaRenameInvalidEdit
		}
		spans = append(spans, javaEditSpan{start: start, end: end, text: edit.NewText})
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start == spans[j].start {
			return spans[i].end > spans[j].end
		}
		return spans[i].start > spans[j].start
	})
	for index := 1; index < len(spans); index++ {
		if spans[index].end > spans[index-1].start || spans[index].start == spans[index-1].start {
			return nil, ErrJavaRenameInvalidEdit
		}
	}
	return spans, nil
}

func applyJavaEditSpans(source []byte, spans []javaEditSpan) []byte {
	updated := append([]byte(nil), source...)
	for _, span := range spans {
		updated = append(updated[:span.start], append([]byte(span.text), updated[span.end:]...)...)
	}
	return updated
}

const javaDiagnosticsTimeout = 5 * time.Second

// Verify applies only validated selected-file edits and collects bounded diagnostics.
func (b *JavaBackend) Verify(ctx context.Context, request backend.VerifyRequest) (diagnostics []backend.Diagnostic, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, target, directory, selectionErr := resolveJavaReadSelection(request.Project, request.Path)
	if selectionErr != nil {
		return nil, selectionErr
	}
	if directory {
		return b.verifyJavaScope(ctx, request, root, target)
	}
	root, file, source, resolveErr := resolveJavaProject(ctx, request.Project)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationVerify, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, fingerprintErr := javaImportFingerprint(root, config)
	if fingerprintErr != nil {
		return nil, fmt.Errorf("verify fingerprint: %w", fingerprintErr)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); err != nil {
		return nil, err
	}
	session := b.session
	if err := b.syncJavaDocumentLocked(ctx, file, source); err != nil {
		return nil, err
	}
	state := b.documents[pathutil.FileURI(file)]
	verified, err := b.applyJavaVerifyEdits(ctx, session, request, file, source, state.version)
	if err != nil {
		return nil, err
	}
	diagnosticsVersion, err := b.publishJavaVerifyEdits(ctx, session, file, source, verified, sourcebuf.FromContext(ctx))
	if err != nil {
		return nil, err
	}
	if diagnosticsVersion > state.version {
		b.documents[pathutil.FileURI(file)] = javaDocumentState{version: diagnosticsVersion, digest: sha256.Sum256(verified.content), pendingSave: sourcebuf.FromContext(ctx) != nil && !bytes.Equal(verified.content, source)}
	}
	diagnostics, waitErr := waitJavaVerifyDiagnostics(ctx, session, file, diagnosticsVersion)
	if waitErr != nil {
		return diagnostics, errors.Join(waitErr, b.invalidateJavaSessionLocked(session))
	}
	return diagnostics, nil
}

func (b *JavaBackend) verifyJavaScope(ctx context.Context, request backend.VerifyRequest, root, target string) ([]backend.Diagnostic, error) {
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationVerify, Language: backend.LanguageJava, Workspace: root}
	}
	files, err := javaSourceFiles(target, root)
	if err != nil {
		return nil, err
	}
	diagnostics := make([]backend.Diagnostic, 0)
	for _, file := range files {
		fileRequest := request
		fileRequest.Project = request.Project
		fileRequest.Project.RootDir = root
		fileRequest.Project.File = file
		fileRequest.Path = file
		fileDiagnostics, verifyErr := b.Verify(ctx, fileRequest)
		if verifyErr != nil {
			return diagnostics, fmt.Errorf("verify Java scope file %q: %w", file, verifyErr)
		}
		diagnostics = append(diagnostics, fileDiagnostics...)
	}
	return diagnostics, nil
}

func waitJavaVerifyDiagnostics(ctx context.Context, session JavaSession, file string, diagnosticsVersion int) ([]backend.Diagnostic, error) {
	diagnosticCtx, cancelDiagnostics := context.WithTimeout(ctx, javaDiagnosticsTimeout)
	defer cancelDiagnostics()
	diagnostics, err := session.WaitDiagnostics(diagnosticCtx, pathutil.FileURI(file), diagnosticsVersion)
	if errors.Is(err, context.DeadlineExceeded) {
		return diagnostics, fmt.Errorf("%w: %w", ErrJavaDiagnosticsTimeout, err)
	}
	return diagnostics, err
}

func javaDiagnosticMessages(root string, diagnostics []backend.Diagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		location := ""
		if diagnostic.Location != nil {
			file, err := pathutil.FilePathFromURI(diagnostic.Location.URI)
			if err == nil {
				relative, relErr := filepath.Rel(root, file)
				if relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
					location = filepath.ToSlash(relative)
				}
			}
			location = fmt.Sprintf("%s:%d:%d", location, diagnostic.Location.Range.Start.Line+1, diagnostic.Location.Range.Start.Character+1)
		}
		messages = append(messages, fmt.Sprintf("%s:%d:%s", location, diagnostic.Severity, diagnostic.Message))
	}
	return messages
}

type javaVerifiedSource struct {
	content []byte
	version int
}

func (b *JavaBackend) applyJavaVerifyEdits(ctx context.Context, session JavaSession, request backend.VerifyRequest, file string, source []byte, version int) (javaVerifiedSource, error) {
	verified := javaVerifiedSource{content: source, version: version}
	if len(request.AddImports)+len(request.RemoveImports) > 0 {
		content, err := applyJavaExplicitImports(verified.content, request.AddImports, request.RemoveImports)
		if err != nil {
			return javaVerifiedSource{}, err
		}
		verified.content = content
	}
	if request.FormatSelectedFile {
		if err := applyJavaFormattingRequest(ctx, session, file, verified.content, &verified); err != nil {
			return javaVerifiedSource{}, err
		}
	}
	if request.OrganizeImports {
		if err := b.applyJavaOrganizeRequest(ctx, session, file, source, &verified); err != nil {
			return javaVerifiedSource{}, err
		}
	}
	return verified, nil
}

func applyJavaFormattingRequest(ctx context.Context, session JavaSession, file string, source []byte, verified *javaVerifiedSource) error {
	raw, err := session.Request(ctx, "textDocument/formatting", map[string]any{lspTextDocumentKey: map[string]string{"uri": pathutil.FileURI(file)}, "options": map[string]any{"tabSize": 4, "insertSpaces": true}})
	if err != nil {
		return err
	}
	verified.content, err = applyJavaFormattingEdits(source, raw)
	return err
}

func (b *JavaBackend) applyJavaOrganizeRequest(ctx context.Context, session JavaSession, file string, source []byte, verified *javaVerifiedSource) error {
	if !bytes.Equal(verified.content, source) {
		if err := b.syncJavaDocumentLocked(ctx, file, verified.content); err != nil {
			return err
		}
		verified.version = b.documents[pathutil.FileURI(file)].version
	}
	raw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{lspTextDocumentKey: map[string]any{"uri": pathutil.FileURI(file), "version": verified.version}, "range": javaRange{}, "context": map[string]any{"only": []string{"source.organizeImports"}, "diagnostics": []any{}}})
	if err != nil {
		return err
	}
	verified.content, err = applyJavaOrganizeImportsEditVersion(file, verified.content, raw, verified.version)
	return err
}

func (b *JavaBackend) publishJavaVerifyEdits(ctx context.Context, session JavaSession, file string, source []byte, verified javaVerifiedSource, buffer *sourcebuf.Buffer) (int, error) {
	changed := !bytes.Equal(verified.content, source)
	if changed && buffer != nil {
		if err := buffer.Apply(ctx, []sourcebuf.Change{{Path: file, Before: source, After: verified.content}}); err != nil {
			return 0, fmt.Errorf("buffer Java verification edits: %w", err)
		}
	} else if changed {
		if err := pipeline.WriteAtomicContext(ctx, file, verified.content); err != nil {
			return 0, err
		}
	}
	if err := b.syncJavaDocumentLocked(ctx, file, verified.content); err != nil {
		return 0, err
	}
	uri := pathutil.FileURI(file)
	state := b.documents[uri]
	if buffer == nil && changed {
		if err := session.Notify(ctx, "textDocument/didSave", map[string]any{lspTextDocumentKey: map[string]string{"uri": pathutil.FileURI(file)}}); err != nil {
			return 0, err
		}
		state.pendingSave = false
	} else if buffer != nil && changed {
		state.pendingSave = true
	}
	b.documents[uri] = state
	return state.version, nil
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
	action, err := decodeJavaOrganizeAction(raw)
	if err != nil {
		return nil, err
	}
	if action.empty {
		return append([]byte(nil), source...), nil
	}
	textEdits, err := decodeJavaOrganizeTextEdits(file, action.edit, expectedVersion)
	if err != nil {
		return nil, err
	}
	spans, err := javaFormattingSpans(source, textEdits)
	if err != nil {
		return nil, err
	}
	return applyJavaEditSpans(source, spans), nil
}

type javaOrganizeAction struct {
	edit  map[string]json.RawMessage
	empty bool
}

func decodeJavaOrganizeAction(raw json.RawMessage) (javaOrganizeAction, error) {
	var actions []json.RawMessage
	if json.Unmarshal(raw, &actions) != nil || len(actions) > 1 {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	if len(actions) == 0 {
		return javaOrganizeAction{empty: true}, nil
	}
	var action map[string]json.RawMessage
	if json.Unmarshal(actions[0], &action) != nil || action == nil {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	var kind string
	if json.Unmarshal(action["kind"], &kind) != nil || kind != "source.organizeImports" {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	if _, ok := action["command"]; ok {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	if disabled, ok := action["disabled"]; ok && string(disabled) != "null" {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	editRaw, ok := action["edit"]
	if !ok {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	var edit map[string]json.RawMessage
	if json.Unmarshal(editRaw, &edit) != nil || edit == nil {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	if _, ok := edit["changeAnnotations"]; ok {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	if _, ok := edit["resourceOperations"]; ok {
		return javaOrganizeAction{}, ErrJavaRenameInvalidEdit
	}
	return javaOrganizeAction{edit: edit}, nil
}

func decodeJavaOrganizeTextEdits(file string, edit map[string]json.RawMessage, expectedVersion int) ([]javaTextEdit, error) {
	_, hasChanges := edit["changes"]
	_, hasDocuments := edit["documentChanges"]
	if hasChanges == hasDocuments {
		return nil, ErrJavaRenameInvalidEdit
	}
	if raw, ok := edit["changes"]; ok {
		return decodeJavaOrganizeChanges(file, raw)
	}
	return decodeJavaOrganizeDocumentChanges(file, edit["documentChanges"], expectedVersion)
}

func decodeJavaOrganizeChanges(file string, raw json.RawMessage) ([]javaTextEdit, error) {
	var files map[string]json.RawMessage
	if json.Unmarshal(raw, &files) != nil || len(files) != 1 {
		return nil, ErrJavaRenameInvalidEdit
	}
	for uri, encoded := range files {
		path, err := pathutil.FilePathFromURI(uri)
		if err != nil || canonicalJavaFilePath(path) != canonicalJavaFilePath(file) {
			return nil, ErrJavaRenameInvalidEdit
		}
		return decodeJavaTextEdits(encoded)
	}
	return nil, ErrJavaRenameInvalidEdit
}

func decodeJavaOrganizeDocumentChanges(file string, raw json.RawMessage, expectedVersion int) ([]javaTextEdit, error) {
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil || len(values) != 1 {
		return nil, ErrJavaRenameInvalidEdit
	}
	var valueObject map[string]json.RawMessage
	if json.Unmarshal(values[0], &valueObject) != nil || valueObject == nil {
		return nil, ErrJavaRenameInvalidEdit
	}
	for key := range valueObject {
		if key != lspTextDocumentKey && key != "edits" {
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
	return decodeJavaTextEdits(doc.Edits)
}

func javaFormattingSpans(source []byte, textEdits []javaTextEdit) ([]javaEditSpan, error) {
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
	return spans, nil
}

func canonicalJavaFilePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	path = filepath.Clean(path)
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return evaluated
	}
	suffix := make([]string, 0, 4)
	for current := path; ; current = filepath.Dir(current) {
		parent := filepath.Dir(current)
		suffix = append(suffix, filepath.Base(current))
		if evaluated, err := filepath.EvalSymlinks(parent); err == nil {
			for _, s := range slices.Backward(suffix) {
				evaluated = filepath.Join(evaluated, s)
			}
			return filepath.Clean(evaluated)
		}
		if parent == current {
			return path
		}
	}
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
			lspTextDocumentKey: map[string]any{
				lspDocumentSymbolKey: map[string]any{"hierarchicalDocumentSymbolSupport": true},
				"formatting":         map[string]any{"dynamicRegistration": false},
				"codeAction": map[string]any{"dynamicRegistration": false, "codeActionLiteralSupport": map[string]any{"codeActionKind": map[string]any{"valueSet": []string{
					"quickfix", "refactor", "refactor.extract", "refactor.extract.function", "refactor.extract.variable", "refactor.extract.constant",
					"refactor.inline", "refactor.inline.variable", "refactor.inline.method", "refactor.rewrite", "refactor.move", "refactor.changeSignature",
					"source", "source.organizeImports", "source.overrideMethods",
				}}}},
			},
			"workspace": map[string]any{
				"workspaceFolders": true,
				"configuration":    true,
				"fileOperations":   map[string]any{"didRename": true},
				"workspaceEdit": map[string]any{
					"documentChanges":    true,
					"resourceOperations": []string{"rename"},
				},
			},
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
	if runtimeErr := validateJavaRuntime(ctx, resolvedJava); runtimeErr != nil {
		return nil, runtimeErr
	}
	launcher, configuration, err := findJDTLSDistribution(config.JDTLSHome)
	if err != nil {
		return nil, err
	}
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, fmt.Errorf("fingerprint JDT LS workspace: %w", err)
	}
	dataPath := javaWorkspaceDataPath(root, fingerprint)
	if pathutil.PathWithin(root, dataPath) {
		return nil, fmt.Errorf("JDT LS data directory %q must be outside workspace root %q", dataPath, root)
	}
	if directoryErr := os.MkdirAll(dataPath, 0o700); directoryErr != nil {
		return nil, fmt.Errorf("create JDT LS data directory: %w", directoryErr)
	}
	// The managed LSP session outlives the request context that starts it.
	cmd := exec.CommandContext(context.Background(), resolvedJava, // #nosec G204 -- executable is resolved from an explicit path or PATH and arguments are fixed.
		"-Declipse.application=org.eclipse.jdt.ls.core.id1",
		"-Dosgi.bundles.defaultStartLevel=4",
		"-Declipse.product=org.eclipse.jdt.ls.core.product",
		"-Dlog.protocol=true", "-Dlog.level=ALL", "-Xms1g", "-Xmx2G",
		"-jar", launcher, "-configuration", configuration, "-data", dataPath,
	)
	cmd.Dir = root
	processSession := newJavaProcessSession(nil)
	processSession.settings = javaJDTLSSettings(config)
	client, err := lsp.NewProcessClient(cmd, lsp.WithRequestHandler(processSession.handleServerRequest), lsp.WithNotificationHandler(func(notification lsp.Notification) {
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
		uriPath, pathErr := pathutil.FilePathFromURI(params.URI)
		if pathErr != nil {
			return
		}
		uri := pathutil.FileURI(uriPath)
		if uri != pathutil.FileURI(root) && !pathutil.PathWithin(root, uriPath) {
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

func javaWorkspaceDataPath(root, fingerprint string) string {
	return filepath.Join(filepath.Dir(root), ".scratch", "jdtls-data", stableJavaRootHash(root+"\x00"+fingerprint))
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

func resolveJavaProject(ctx context.Context, project backend.ProjectContext) (string, string, []byte, error) {
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
	source, err := sourcebuf.ReadFile(ctx, file)
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
	Detail         string
}
type javaRange struct {
	Start javaPosition `json:"start"`
	End   javaPosition `json:"end"`
}
type javaPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
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
	if err := readlsp.ValidateRangeShape(object["range"]); err != nil {
		return javaDocumentSymbol{}, fmt.Errorf("%w: invalid range: %w", ErrJavaMalformedResponse, err)
	}
	if err := readlsp.ValidateRangeShape(object["selectionRange"]); err != nil {
		return javaDocumentSymbol{}, fmt.Errorf("%w: invalid selection range: %w", ErrJavaMalformedResponse, err)
	}
	var symbol javaDocumentSymbol
	if err := decodeJavaDocumentSymbolFields(object, source, &symbol); err != nil {
		return javaDocumentSymbol{}, err
	}
	if err := decodeJavaDocumentSymbolMetadata(object, root, &symbol); err != nil {
		return javaDocumentSymbol{}, err
	}
	if err := decodeJavaDocumentSymbolChildren(object, root, source, &symbol); err != nil {
		return javaDocumentSymbol{}, err
	}
	return symbol, nil
}

func decodeJavaDocumentSymbolFields(object map[string]json.RawMessage, source []byte, symbol *javaDocumentSymbol) error {
	if err := json.Unmarshal(object["name"], &symbol.Name); err != nil || symbol.Name == "" {
		return fmt.Errorf("%w: missing name", ErrJavaMalformedResponse)
	}
	if err := json.Unmarshal(object["kind"], &symbol.Kind); err != nil {
		return fmt.Errorf("%w: invalid kind", ErrJavaMalformedResponse)
	}
	if err := json.Unmarshal(object["range"], &symbol.Range); err != nil || !validJavaRange(symbol.Range, source) {
		return fmt.Errorf("%w: invalid range", ErrJavaMalformedResponse)
	}
	if err := json.Unmarshal(object["selectionRange"], &symbol.SelectionRange); err != nil || !validJavaRange(symbol.SelectionRange, source) {
		return fmt.Errorf("%w: invalid selection range", ErrJavaMalformedResponse)
	}
	return nil
}

func decodeJavaDocumentSymbolMetadata(object map[string]json.RawMessage, root string, symbol *javaDocumentSymbol) error {
	if detailRaw, ok := object["detail"]; ok && string(detailRaw) != "null" {
		if err := json.Unmarshal(detailRaw, &symbol.Detail); err != nil {
			return fmt.Errorf("%w: invalid detail", ErrJavaMalformedResponse)
		}
	}
	if uriRaw, ok := object["uri"]; ok {
		if err := json.Unmarshal(uriRaw, &symbol.URI); err != nil {
			return fmt.Errorf("%w: invalid uri", ErrJavaMalformedResponse)
		}
		if symbol.URI != "" {
			uriPath, err := pathutil.FilePathFromURI(symbol.URI)
			if err != nil || !pathutil.PathWithin(root, uriPath) {
				return fmt.Errorf("%w: symbol uri is outside workspace", ErrJavaMalformedResponse)
			}
		}
	}
	return nil
}

func decodeJavaDocumentSymbolChildren(object map[string]json.RawMessage, root string, source []byte, symbol *javaDocumentSymbol) error {
	if childrenRaw, ok := object["children"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return fmt.Errorf("%w: invalid children", ErrJavaMalformedResponse)
		}
		for _, childRaw := range children {
			child, err := decodeJavaDocumentSymbol(childRaw, root, source)
			if err != nil {
				return err
			}
			symbol.Children = append(symbol.Children, child)
		}
	}
	return nil
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
	case 14, 22:
		return "field"
	case 23:
		return "record"
	default:
		return ""
	}
}
func javaKindSupported(kind int) bool { return javaKindName(kind) != "" }

func selectJavaSymbol(query, file string, symbols []javaDocumentSymbol) (*backend.LookupResult, error) {
	parts, signature, signatureSpecified, err := parseJavaSymbolQuery(query)
	if err != nil {
		return nil, &JavaError{Op: "lookup", Symbol: query, Err: err}
	}
	qualifiedPrefix := strings.Join(parts, ".")
	baseName := parts[len(parts)-1]
	parentQuery := strings.Join(parts[:len(parts)-1], ".")
	type match struct {
		symbol javaDocumentSymbol
		path   []string
	}
	matches := make([]match, 0)
	var visit func([]string, []javaDocumentSymbol)
	visit = func(parent []string, items []javaDocumentSymbol) {
		for _, item := range items {
			itemBase, itemSignature, itemHasSignature := javaSymbolNames(item)
			path := append(append([]string(nil), parent...), itemBase)
			qualified := strings.Join(path, ".")
			parentMatches := parentQuery == "" || qualified == qualifiedPrefix || strings.HasSuffix(qualified, "."+parentQuery+"."+baseName)
			nameMatches := itemBase == baseName
			if signatureSpecified {
				nameMatches = nameMatches && itemHasSignature && normalizeJavaSignature(itemSignature) == signature
			}
			if javaKindSupported(item.Kind) && parentMatches && nameMatches {
				candidatePath := path
				if itemHasSignature {
					candidatePath = append(append([]string(nil), parent...), itemBase+"("+itemSignature+")")
				}
				matches = append(matches, match{item, candidatePath})
			}
			visit(path, item.Children)
		}
	}
	visit(nil, symbols)
	if len(matches) == 0 {
		return nil, &JavaError{Op: "lookup", File: file, Symbol: query, Err: ErrJavaSymbolNotFound}
	}
	converted := make([]*backend.NamedTarget, 0, len(matches))
	for _, item := range matches {
		converted = append(converted, javaCandidate(file, item.symbol, item.path))
	}
	if len(converted) > 1 {
		return &backend.LookupResult{Candidates: converted}, nil
	}
	candidate := converted[0]
	return &backend.LookupResult{Definition: candidate}, nil
}

func parseJavaSymbolQuery(query string) ([]string, string, bool, error) {
	query = strings.Trim(strings.TrimSpace(query), `"'`)
	signature := ""
	signatureSpecified := false
	if open := strings.IndexByte(query, '('); open >= 0 {
		if !strings.HasSuffix(query, ")") || strings.Count(query[open:], "(") != 1 || strings.Count(query[open:], ")") != 1 {
			return nil, "", false, ErrJavaMalformedResponse
		}
		signatureSpecified = true
		signature = normalizeJavaSignature(query[open+1 : len(query)-1])
		query = strings.TrimSpace(query[:open])
	}
	parts := strings.Split(query, ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, "", false, ErrJavaMalformedResponse
		}
	}
	return parts, signature, signatureSpecified, nil
}

func normalizeJavaSignature(signature string) string {
	parameters := strings.Split(signature, ",")
	for i := range parameters {
		parameters[i] = strings.Join(strings.Fields(parameters[i]), " ")
	}
	return strings.Join(parameters, ", ")
}

func javaSymbolNames(symbol javaDocumentSymbol) (string, string, bool) {
	name := strings.TrimSpace(symbol.Name)
	if open := strings.IndexByte(name, '('); open >= 0 && strings.HasSuffix(name, ")") {
		return strings.TrimSpace(name[:open]), normalizeJavaSignature(name[open+1 : len(name)-1]), true
	}
	if kind := javaKindName(symbol.Kind); kind == "method" || kind == "constructor" {
		if open := strings.IndexByte(symbol.Detail, '('); open >= 0 {
			close := strings.IndexByte(symbol.Detail[open+1:], ')')
			if close >= 0 {
				params := strings.Split(symbol.Detail[open+1:open+1+close], ",")
				for i := range params {
					params[i] = javaSimpleParameterType(params[i])
				}
				return name, normalizeJavaSignature(strings.Join(params, ",")), true
			}
		}
	}
	return name, "", false
}

func javaSimpleParameterType(parameter string) string {
	parameter = strings.TrimSpace(parameter)
	parameter = strings.TrimPrefix(parameter, "final ")
	if fields := strings.Fields(parameter); len(fields) > 1 {
		parameter = strings.Join(fields[:len(fields)-1], " ")
	}
	parameter = strings.ReplaceAll(parameter, "...", "[]")
	if generic := strings.IndexByte(parameter, '<'); generic >= 0 {
		parameter = parameter[:generic] + strings.TrimSpace(parameter[strings.LastIndex(parameter, ">")+1:])
	}
	if index := strings.LastIndex(parameter, "."); index >= 0 {
		arraySuffix := strings.Repeat("[]", strings.Count(parameter, "[]"))
		parameter = strings.TrimSuffix(parameter, arraySuffix)
		parameter = parameter[index+1:] + arraySuffix
	}
	return strings.TrimSpace(parameter)
}

func javaCandidate(file string, symbol javaDocumentSymbol, path []string) *backend.NamedTarget {
	start := symbol.SelectionRange.Start
	location := backend.SourceLocation{URI: pathutil.FileURI(file), Range: backend.Range{Start: backend.Position(start), End: backend.Position(symbol.SelectionRange.End)}}
	return &backend.NamedTarget{Name: symbol.Name, QualifiedName: strings.Join(path, "."), Kind: javaKindName(symbol.Kind), Location: location}
}

func javaByteOffset(source []byte, position javaPosition) (int, error) {
	return readlsp.ByteOffset(source, backend.Position{Line: position.Line, Character: position.Character})
}

func (b *JavaBackend) withJavaDocumentLocked(ctx context.Context, file string, source []byte, action func() error) error {
	if err := b.syncJavaDocumentLocked(ctx, file, source); err != nil {
		return err
	}
	return action()
}

func (b *JavaBackend) invalidateJavaSessionLocked(session JavaSession) error {
	if b.session == nil {
		return nil
	}
	delete(b.sessions, b.root)
	closeErr := session.Close()
	b.session, b.root, b.fingerprint = nil, "", ""
	b.documents = nil
	if closeErr != nil {
		return fmt.Errorf("close invalid Java session: %w", closeErr)
	}
	return nil
}

func (b *JavaBackend) readJavaSymbolsLocked(ctx context.Context, root, file string, source []byte) ([]javaDocumentSymbol, error) {
	var symbols []javaDocumentSymbol
	err := b.withJavaDocumentLocked(ctx, file, source, func() error {
		raw, err := b.session.Request(ctx, "textDocument/documentSymbol", map[string]any{lspTextDocumentKey: map[string]string{"uri": pathutil.FileURI(file)}})
		if err != nil {
			return &JavaError{Op: lspDocumentSymbolKey, File: file, Err: err}
		}
		symbols, err = decodeJavaDocumentSymbols(raw, root, source)
		if err != nil {
			return &JavaError{Op: lspDocumentSymbolKey, File: file, Err: err}
		}
		if err := validateJavaDocumentSymbolURI(symbols, pathutil.FileURI(file)); err != nil {
			return &JavaError{Op: lspDocumentSymbolKey, File: file, Err: err}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return symbols, nil
}

func validateJavaDocumentSymbolURI(symbols []javaDocumentSymbol, expected string) error {
	for _, symbol := range symbols {
		if symbol.URI != "" && symbol.URI != expected {
			return fmt.Errorf("%w: symbol uri does not match selected document", ErrJavaMalformedResponse)
		}
		if err := validateJavaDocumentSymbolURI(symbol.Children, expected); err != nil {
			return err
		}
	}
	return nil
}

func (b *JavaBackend) ensureJavaSessionLocked(ctx context.Context, root string, config backend.JavaConfig, fingerprint string) error {
	if b.sessions == nil {
		b.sessions = make(map[string]javaSessionRecord)
	}
	if b.session != nil && b.root != root {
		b.sessions[b.root] = javaSessionRecord{session: b.session, fingerprint: b.fingerprint, documents: b.documents}
		b.session, b.root, b.fingerprint, b.documents = nil, "", "", nil
	}
	if b.session == nil {
		if retained, ok := b.sessions[root]; ok {
			delete(b.sessions, root)
			if retained.fingerprint == fingerprint {
				b.session, b.root, b.fingerprint, b.documents = retained.session, root, fingerprint, retained.documents
				return nil
			}
			if retained.session != nil {
				if err := retained.session.Close(); err != nil {
					return &JavaError{Op: "restart", Workspace: root, Err: fmt.Errorf("close stale Java session: %w", err)}
				}
			}
		}
	}
	if b.session != nil && b.fingerprint != fingerprint {
		session := b.session
		closeErr := session.Close()
		b.session, b.root, b.fingerprint, b.documents = nil, "", "", nil
		if closeErr != nil {
			return &JavaError{Op: "restart", Workspace: root, Err: fmt.Errorf("close stale Java session: %w", closeErr)}
		}
	}
	if b.session != nil {
		return nil
	}
	if b.factory == nil {
		return &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
	}
	session, err := b.factory(ctx, root, config)
	if err != nil {
		return &JavaError{Op: "start", Workspace: root, Err: err}
	}
	if session == nil {
		return &JavaError{Op: "start", Workspace: root, Err: ErrJDTLSUnavailable}
	}
	b.session, b.root, b.fingerprint = session, root, fingerprint
	if err := initializeJavaSession(ctx, session, root, config); err != nil {
		closeErr := session.Close()
		b.session, b.root, b.fingerprint, b.documents = nil, "", "", nil
		return &JavaError{Op: "initialize", Workspace: root, Err: errors.Join(err, closeErr)}
	}
	return nil
}

func mapJavaReadSymbols(items []javaDocumentSymbol, parent []string, file string) ([]backend.ReadSymbol, error) {
	result := make([]backend.ReadSymbol, 0, len(items))
	for _, item := range items {
		kind := javaKindName(item.Kind)
		if kind == "" {
			return nil, fmt.Errorf("%w: unsupported Java symbol kind %d", ErrJavaUnsupportedResponse, item.Kind)
		}
		baseName, signatureName, hasSignature := javaSymbolNames(item)
		nameParts := strings.Split(baseName, ".")
		pathParts := append(append([]string(nil), parent...), nameParts...)
		qualifiedParts := append([]string(nil), pathParts...)
		var signature *string
		if hasSignature {
			qualifiedParts[len(qualifiedParts)-1] += "(" + signatureName + ")"
			value := signatureName
			signature = &value
		}
		qualified := strings.Join(qualifiedParts, ".")
		symbol := backend.ReadSymbol{
			Name:           baseName,
			QualifiedName:  qualified,
			Kind:           kind,
			File:           file,
			Range:          javaReadRange(item.Range),
			SelectionRange: javaReadRange(item.SelectionRange),
			Signature:      signature,
		}
		if item.Detail != "" {
			detail := item.Detail
			symbol.ServerDetail = &detail
		}
		children, err := mapJavaReadSymbols(item.Children, pathParts, file)
		if err != nil {
			return nil, err
		}
		symbol.Children = children
		result = append(result, symbol)
	}
	return result, nil
}

func javaReadRange(value javaRange) backend.Range {
	return backend.Range{
		Start: backend.Position{Line: value.Start.Line, Character: value.Start.Character},
		End:   backend.Position{Line: value.End.Line, Character: value.End.Character},
	}
}

func resolveJavaReadSelection(project backend.ProjectContext, requested string) (string, string, bool, error) {
	target := requested
	if strings.TrimSpace(target) == "" {
		target = project.File
	}
	if strings.TrimSpace(target) == "" {
		return "", "", false, ErrJavaFileRequired
	}
	base := project.RootDir
	if base == "" {
		base = "."
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, target)
	}
	target = backend.CanonicalWorkspaceRoot(target)
	info, statErr := os.Stat(target)
	directory := statErr == nil && info.IsDir()
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", "", false, &JavaError{Op: "project", File: target, Err: statErr}
	}
	var root string
	if project.RootDir != "" {
		root = backend.CanonicalWorkspaceRoot(project.RootDir)
	} else {
		marker := target
		if directory {
			marker = filepath.Join(target, "__semedit_outline__.java")
		}
		var err error
		root, err = discoverJavaWorkspaceRoot(marker)
		if err != nil {
			return "", "", directory, &JavaError{Op: "project", File: target, Err: err}
		}
	}
	if !pathutil.PathWithin(root, target) {
		return "", "", directory, &JavaError{Op: "project", File: target, Workspace: root, Err: ErrJavaFileOutsideWorkspace}
	}
	return root, target, directory, nil
}

// Outline returns the selected Java file's validated hierarchical declarations.
func (b *JavaBackend) Outline(ctx context.Context, request backend.OutlineRequest) (*backend.OutlineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, directory, err := resolveJavaReadSelection(request.Project, request.Path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Path) == "" {
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: backend.LanguageJava, Err: errors.New("outline path is required")}
	}
	if directory {
		return b.outlineJavaDirectory(ctx, request, root)
	}
	outlineRequest := request
	outlineRequest.IncludeUnexported = true
	if requestErr := readlsp.ValidateOutlineRequest(outlineRequest, false); requestErr != nil {
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: backend.LanguageJava, Err: requestErr}
	}
	if !strings.EqualFold(filepath.Ext(file), ".java") {
		return nil, &JavaError{Op: "outline", File: file, Err: ErrJavaFileRequired}
	}
	project := request.Project
	project.RootDir, project.File = root, file
	root, file, source, err := resolveJavaProject(ctx, project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationOutline, Language: backend.LanguageJava, Workspace: root}
	}
	config := effectiveJavaConfig(project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: err}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); sessionErr != nil {
		return nil, sessionErr
	}
	symbols, err := b.readJavaSymbolsLocked(ctx, root, file, source)
	if err != nil {
		return nil, err
	}
	mapped, err := mapJavaReadSymbols(symbols, nil, file)
	if err != nil {
		return nil, &JavaError{Op: "outline", File: file, Err: err}
	}
	result, err := readlsp.Outline(readlsp.Document{Root: root, File: file, Language: backend.LanguageJava, Source: source, Symbols: mapped, Complete: true}, outlineRequest)
	if err == nil && !request.IncludeUnexported {
		filterJavaPublicSymbols(source, &result.Files[0].Symbols)
	}
	return result, err
}

func (b *JavaBackend) outlineJavaDirectory(ctx context.Context, request backend.OutlineRequest, root string) (*backend.OutlineResult, error) {
	if err := validateJavaOutlineKinds(request.Kinds); err != nil {
		return nil, err
	}
	target := request.Path
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target = backend.CanonicalWorkspaceRoot(target)
	if !pathutil.PathWithin(root, target) {
		return nil, &JavaError{Op: "outline", File: target, Workspace: root, Err: ErrJavaFileOutsideWorkspace}
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationOutline, Language: backend.LanguageJava, Workspace: root}
	}
	project := request.Project
	project.RootDir = root
	config := effectiveJavaConfig(project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: err}
	}
	files, err := javaOutlineFiles(target, root, request.IncludeTests)
	if err != nil {
		return nil, fmt.Errorf("scan Java outline directory %q: %w", target, err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); err != nil {
		return nil, err
	}
	result := &backend.OutlineResult{Scope: backend.ReadScope{Kind: "directory", Path: filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(target, root), string(filepath.Separator))), Complete: true}, Files: make([]backend.OutlineFile, 0, len(files))}
	if target == root {
		result.Scope.Path = "."
	}
	for _, file := range files {
		fileOutline, outlineErr := b.outlineJavaFile(ctx, request, root, project, file)
		if outlineErr != nil {
			return nil, outlineErr
		}
		result.Files = append(result.Files, fileOutline)
	}
	return result, nil
}

func (b *JavaBackend) outlineJavaFile(ctx context.Context, request backend.OutlineRequest, root string, project backend.ProjectContext, file string) (backend.OutlineFile, error) {
	fileProject := project
	fileProject.File = file
	_, canonicalFile, source, err := resolveJavaProject(ctx, fileProject)
	if err != nil {
		return backend.OutlineFile{}, err
	}
	symbols, err := b.readJavaSymbolsLocked(ctx, root, canonicalFile, source)
	if err != nil {
		return backend.OutlineFile{}, err
	}
	mapped, err := mapJavaReadSymbols(symbols, nil, canonicalFile)
	if err != nil {
		return backend.OutlineFile{}, &JavaError{Op: "outline", File: canonicalFile, Err: err}
	}
	fileRequest := request
	fileRequest.Path, fileRequest.IncludeUnexported = canonicalFile, true
	outlined, err := readlsp.Outline(readlsp.Document{Root: root, File: canonicalFile, Language: backend.LanguageJava, Source: source, Symbols: mapped, Complete: true}, fileRequest)
	if err != nil {
		return backend.OutlineFile{}, err
	}
	fileOutline := outlined.Files[0]
	if !request.IncludeUnexported {
		filterJavaPublicSymbols(source, &fileOutline.Symbols)
	}
	return fileOutline, nil
}

func validateJavaOutlineKinds(kinds []string) error {
	for _, kind := range kinds {
		switch kind {
		case "package", "class", "method", "field", "constructor", "enum", "interface", "record":
		default:
			return &backend.Error{Operation: backend.OperationOutline, Language: backend.LanguageJava, Err: fmt.Errorf("unknown outline kind %q", kind)}
		}
	}
	return nil
}

func javaOutlineFiles(target, root string, includeTests bool) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != target && sourcefiles.IsIgnoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".java") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		isTest := strings.Contains(rel, "/src/test/java/") || strings.HasPrefix(rel, "src/test/java/")
		if isTest && !includeTests {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

var javaPublicModifier = regexp.MustCompile(`(^|\s)public(\s|$)`)

func filterJavaPublicSymbols(source []byte, symbols *[]backend.ReadSymbol) {
	filtered := (*symbols)[:0]
	for _, symbol := range *symbols {
		start, startErr := javaByteOffset(source, javaPosition{Line: symbol.Range.Start.Line, Character: symbol.Range.Start.Character})
		end, endErr := javaByteOffset(source, javaPosition{Line: symbol.Range.End.Line, Character: symbol.Range.End.Character})
		if startErr != nil || endErr != nil || end < start || !javaPublicModifier.Match(source[start:end]) {
			continue
		}
		filterJavaPublicSymbols(source, &symbol.Children)
		filtered = append(filtered, symbol)
	}
	*symbols = filtered
}

// Inspect returns all Java declarations whose hierarchical names match the query.
func (b *JavaBackend) Inspect(ctx context.Context, request backend.InspectRequest) (*backend.InspectResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(request.Project.File) != "" {
		root, target, directory, selectionErr := resolveJavaReadSelection(request.Project, request.Project.File)
		if selectionErr != nil {
			return nil, selectionErr
		}
		if directory {
			return b.inspectJavaDirectory(ctx, request, root, target)
		}
	}
	root, file, source, err := resolveJavaProject(ctx, request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationInspect, Language: backend.LanguageJava, Workspace: root}
	}
	if strings.TrimSpace(request.Symbol) == "" {
		return nil, &JavaError{Op: "inspect", File: file, Symbol: request.Symbol, Err: ErrJavaMalformedResponse}
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: err}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); sessionErr != nil {
		return nil, sessionErr
	}
	rawSymbols, err := b.readJavaSymbolsLocked(ctx, root, file, source)
	if err != nil {
		return nil, err
	}
	symbols, err := mapJavaReadSymbols(rawSymbols, nil, file)
	if err != nil {
		return nil, &JavaError{Op: "inspect", File: file, Symbol: request.Symbol, Err: err}
	}
	matches, err := matchJavaReadSymbols(request.Symbol, symbols)
	if err != nil {
		return nil, &JavaError{Op: "inspect", File: file, Symbol: request.Symbol, Err: err}
	}
	if len(matches) == 0 {
		return nil, &JavaError{Op: "inspect", File: file, Symbol: request.Symbol, Err: ErrJavaSymbolNotFound}
	}
	inspectMatches := make([]readlsp.InspectMatch, 0, len(matches))
	for _, symbol := range matches {
		inspectMatches = append(inspectMatches, readlsp.InspectMatch{Symbol: symbol, SourceExtent: "declaration"})
	}
	return readlsp.Inspect(readlsp.Document{Root: root, File: file, Language: backend.LanguageJava, Source: source, Symbols: symbols, Complete: true}, inspectMatches)
}

func (b *JavaBackend) inspectJavaDirectory(ctx context.Context, request backend.InspectRequest, root, target string) (*backend.InspectResult, error) {
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationInspect, Language: backend.LanguageJava, Workspace: root}
	}
	if strings.TrimSpace(request.Symbol) == "" {
		return nil, &JavaError{Op: "inspect", File: target, Symbol: request.Symbol, Err: ErrJavaMalformedResponse}
	}
	files, err := javaSourceFiles(target, root)
	if err != nil {
		return nil, err
	}
	config := effectiveJavaConfig(request.Project)
	fingerprint, err := javaImportFingerprint(root, config)
	if err != nil {
		return nil, &JavaError{Op: javaOperationFingerprint, Workspace: root, Err: err}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureJavaSessionLocked(ctx, root, config, fingerprint); err != nil {
		return nil, err
	}
	result := &backend.InspectResult{Scope: backend.ReadScope{Kind: "directory", Path: javaScopeDisplay(root, target), Complete: true}, Matches: make([]backend.InspectedSymbol, 0)}
	for _, file := range files {
		fileProject := request.Project
		fileProject.RootDir, fileProject.File = root, file
		_, selected, source, readErr := resolveJavaProject(ctx, fileProject)
		if readErr != nil {
			return nil, readErr
		}
		rawSymbols, readErr := b.readJavaSymbolsLocked(ctx, root, selected, source)
		if readErr != nil {
			return nil, readErr
		}
		symbols, mapErr := mapJavaReadSymbols(rawSymbols, nil, selected)
		if mapErr != nil {
			return nil, &JavaError{Op: "inspect", File: selected, Symbol: request.Symbol, Err: mapErr}
		}
		matches, matchErr := matchJavaReadSymbols(request.Symbol, symbols)
		if matchErr != nil {
			return nil, &JavaError{Op: "inspect", File: selected, Symbol: request.Symbol, Err: matchErr}
		}
		if len(matches) == 0 {
			continue
		}
		inspectMatches := make([]readlsp.InspectMatch, 0, len(matches))
		for _, symbol := range matches {
			inspectMatches = append(inspectMatches, readlsp.InspectMatch{Symbol: symbol, SourceExtent: "declaration"})
		}
		fileResult, inspectErr := readlsp.Inspect(readlsp.Document{Root: root, File: selected, Language: backend.LanguageJava, Source: source, Symbols: symbols, Complete: true}, inspectMatches)
		if inspectErr != nil {
			return nil, inspectErr
		}
		result.Matches = append(result.Matches, fileResult.Matches...)
	}
	if len(result.Matches) == 0 {
		return nil, &JavaError{Op: "inspect", File: target, Symbol: request.Symbol, Err: ErrJavaSymbolNotFound}
	}
	return result, nil
}

func javaScopeDisplay(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." {
		return "."
	}
	return filepath.ToSlash(relative)
}

func matchJavaReadSymbols(query string, symbols []backend.ReadSymbol) ([]backend.ReadSymbol, error) {
	parts, signature, signatureSpecified, err := parseJavaSymbolQuery(query)
	if err != nil {
		return nil, err
	}
	baseQuery := strings.Join(parts, ".")
	matches := make([]backend.ReadSymbol, 0)
	var visit func([]backend.ReadSymbol)
	visit = func(items []backend.ReadSymbol) {
		for _, item := range items {
			qualifiedBase := item.QualifiedName
			if open := strings.IndexByte(qualifiedBase, '('); open >= 0 {
				qualifiedBase = qualifiedBase[:open]
			}
			nameMatches := qualifiedBase == baseQuery || strings.HasSuffix(qualifiedBase, "."+baseQuery)
			if signatureSpecified {
				nameMatches = nameMatches && item.Signature != nil && normalizeJavaSignature(*item.Signature) == signature
			}
			if nameMatches {
				matches = append(matches, item)
			}
			visit(item.Children)
		}
	}
	visit(symbols)
	return matches, nil
}

type javaDocumentState struct {
	version     int
	digest      [32]byte
	pendingSave bool
}

type javaSessionRecord struct {
	session     JavaSession
	fingerprint string
	documents   map[string]javaDocumentState
}

func (b *JavaBackend) syncJavaDocumentLocked(ctx context.Context, file string, content []byte) error {
	session := b.session
	if session == nil {
		return ErrJDTLSUnavailable
	}
	uri := pathutil.FileURI(file)
	digest := sha256.Sum256(content)
	prior, opened := b.documents[uri]
	if opened && prior.digest == digest {
		shouldSave := prior.pendingSave && sourcebuf.FromContext(ctx) == nil
		if prior.pendingSave && !shouldSave {
			diskContent, readErr := sourcebuf.ReadFile(sourcebuf.WithoutBuffer(ctx), file)
			if readErr != nil {
				if !errors.Is(readErr, os.ErrNotExist) {
					return fmt.Errorf("read Java document before pending save: %w", readErr)
				}
			} else {
				shouldSave = bytes.Equal(diskContent, content)
			}
		}
		if shouldSave {
			if err := session.Notify(ctx, "textDocument/didSave", map[string]any{
				lspTextDocumentKey: map[string]string{"uri": uri}, "text": string(content),
			}); err != nil {
				return errors.Join(fmt.Errorf("didSave Java document: %w", err), b.invalidateJavaSessionLocked(session))
			}
			prior.pendingSave = false
			b.documents[uri] = prior
		}
		return nil
	}
	version := 1
	method := "textDocument/didOpen"
	params := map[string]any{lspTextDocumentKey: map[string]any{
		"uri": uri, "languageId": "java", "version": version, "text": string(content),
	}}
	if opened {
		version = prior.version + 1
		method = "textDocument/didChange"
		params = map[string]any{
			lspTextDocumentKey: map[string]any{"uri": uri, "version": version},
			"contentChanges":   []map[string]string{{"text": string(content)}},
		}
	}
	if err := session.Notify(ctx, method, params); err != nil {
		return errors.Join(fmt.Errorf("%s Java document: %w", method, err), b.invalidateJavaSessionLocked(session))
	}
	if processSession, ok := session.(*javaProcessSession); ok {
		processSession.recordDocumentVersion(uri, version)
	}
	if b.documents == nil {
		b.documents = make(map[string]javaDocumentState)
	}
	b.documents[uri] = javaDocumentState{version: version, digest: digest}
	return nil
}

func (b *JavaBackend) saveCommittedJavaDocumentsLocked(ctx context.Context) error {
	for uri, state := range b.documents {
		if !state.pendingSave {
			continue
		}
		file, err := pathutil.FilePathFromURI(uri)
		if err != nil {
			return fmt.Errorf("resolve pending Java document URI %q: %w", uri, err)
		}
		content, err := sourcebuf.ReadFile(sourcebuf.WithoutBuffer(ctx), file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("read pending Java document %q: %w", file, err)
		}
		if sha256.Sum256(content) != state.digest {
			continue
		}
		if err := b.session.Notify(ctx, "textDocument/didSave", map[string]any{
			lspTextDocumentKey: map[string]string{"uri": uri}, "text": string(content),
		}); err != nil {
			return errors.Join(fmt.Errorf("save committed Java document %q: %w", file, err), b.invalidateJavaSessionLocked(b.session))
		}
		state.pendingSave = false
		b.documents[uri] = state
	}
	return nil
}

func (s *javaProcessSession) RequestWithApplyEdit(ctx context.Context, method string, params any, apply func(json.RawMessage) (bool, error)) (json.RawMessage, error) {
	s.applyEditMu.Lock()
	defer s.applyEditMu.Unlock()
	s.mu.Lock()
	s.applyEditHandler = apply
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.applyEditHandler = nil
		s.mu.Unlock()
	}()
	return s.Request(ctx, method, params)
}

func (s *javaProcessSession) handleServerRequest(_ context.Context, request lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
	switch request.Method {
	case "workspace/configuration":
		var params struct {
			Items []struct {
				Section string `json:"section"`
			} `json:"items"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, &lsp.ErrorObject{Code: -32602, Message: "malformed workspace/configuration request"}
		}
		settings := make([]any, len(params.Items))
		for index, item := range params.Items {
			settings[index] = javaSettingValue(s.settings, item.Section)
		}
		result, err := json.Marshal(settings)
		if err != nil {
			return nil, &lsp.ErrorObject{Code: -32603, Message: "could not encode workspace/configuration response"}
		}
		return result, nil
	case "window/showMessageRequest":
		return json.RawMessage("null"), nil
	case "workspace/applyEdit":
		var params struct {
			Edit json.RawMessage `json:"edit"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil || len(params.Edit) == 0 || string(params.Edit) == "null" {
			result, _ := json.Marshal(map[string]any{"applied": false, "failureReason": "malformed workspace/applyEdit request"})
			return result, nil
		}
		s.mu.Lock()
		apply := s.applyEditHandler
		s.mu.Unlock()
		if apply == nil {
			result, _ := json.Marshal(map[string]any{"applied": false, "failureReason": "no active workspace edit handler"})
			return result, nil
		}
		applied, err := apply(params.Edit)
		response := map[string]any{"applied": applied}
		if err != nil {
			response["applied"] = false
			response["failureReason"] = err.Error()
		}
		result, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			return nil, &lsp.ErrorObject{Code: -32603, Message: "could not encode workspace/applyEdit response"}
		}
		return result, nil
	default:
		return nil, &lsp.ErrorObject{Code: -32601, Message: fmt.Sprintf("unsupported Java language-server request %q", request.Method)}
	}
}

func javaSettingValue(settings map[string]any, section string) any {
	if section == "" {
		return settings
	}
	if value, ok := settings[section]; ok {
		return value
	}
	var value any = settings
	for part := range strings.SplitSeq(section, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value, ok = object[part]
		if !ok {
			return nil
		}
	}
	return value
}

func validateJavaResourceMove(root, selectedFile string, files []javaWorkspaceFileEdit, rename javaResourceRename) error {
	if rename.oldPath != selectedFile || rename.oldPath == rename.newPath ||
		!strings.EqualFold(filepath.Ext(rename.oldPath), ".java") || !strings.EqualFold(filepath.Ext(rename.newPath), ".java") ||
		filepath.Base(rename.oldPath) != filepath.Base(rename.newPath) ||
		!pathutil.PathWithin(root, rename.oldPath) || !pathutil.PathWithin(root, rename.newPath) {
		return fmt.Errorf("%w: Java resource move must move the selected source file within the trusted workspace and preserve its filename", ErrJavaRenameInvalidEdit)
	}
	destinationInfo, err := os.Stat(filepath.Dir(rename.newPath))
	if err != nil {
		return fmt.Errorf("inspect Java resource move destination directory: %w", err)
	}
	if !destinationInfo.IsDir() {
		return fmt.Errorf("%w: Java resource move destination is not a directory", ErrJavaRenameInvalidEdit)
	}
	if _, err := os.Lstat(rename.newPath); err == nil {
		return fmt.Errorf("%w: Java resource move destination already exists: %s", ErrJavaRenameInvalidEdit, rename.newPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect Java resource move destination %q: %w", rename.newPath, err)
	}
	typeName := strings.TrimSuffix(filepath.Base(rename.oldPath), filepath.Ext(rename.oldPath))
	for _, file := range files {
		if file.file != rename.oldPath {
			continue
		}
		publicTopLevel, err := javacst.IsPublicTopLevelType(file.before, typeName)
		if err != nil {
			return fmt.Errorf("inspect Java resource move source: %w", err)
		}
		if !publicTopLevel || !hasJavaPublicTypeSource(file.after, typeName) {
			return fmt.Errorf("%w: Java resource moves require the selected public top-level type to remain in its matching source file", ErrJavaRenameInvalidEdit)
		}
		return nil
	}
	return fmt.Errorf("%w: Java resource move has no selected-file text edit", ErrJavaRenameInvalidEdit)
}
