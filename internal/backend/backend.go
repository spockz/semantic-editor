// Package backend owns the language-neutral service boundary used by CLI and MCP ingress.
package backend

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/capability"
	"github.com/spockz/semantic-editor/internal/sourcefiles"

	"golang.org/x/mod/modfile"
)

// LanguageID identifies a source language supported by a backend registry.
type LanguageID string

// ConstructKind names a syntax construct in the cross-language capability taxonomy.
// Membership here does not claim that any registered backend can mutate that kind.
type ConstructKind string

// Canonical construct kinds reserved by the cross-language taxonomy.
const (
	ConstructFunction         ConstructKind = "function"
	ConstructMethod           ConstructKind = "method"
	ConstructConstructor      ConstructKind = "constructor"
	ConstructType             ConstructKind = "type"
	ConstructField            ConstructKind = "field"
	ConstructConst            ConstructKind = "const"
	ConstructVar              ConstructKind = "var"
	ConstructDeclaration      ConstructKind = "declaration"
	ConstructDecl             ConstructKind = "decl"
	ConstructStatementOrExpr  ConstructKind = "statement/expression"
	ConstructClass            ConstructKind = "class"
	ConstructInterface        ConstructKind = "interface"
	ConstructEnum             ConstructKind = "enum"
	ConstructLoop             ConstructKind = "loop"
	ConstructIf               ConstructKind = "if"
	ConstructElse             ConstructKind = "else"
	ConstructCase             ConstructKind = "case"
	ConstructTernary          ConstructKind = "ternary"
	ConstructSelect           ConstructKind = "select"
	ConstructDefer            ConstructKind = "defer"
	ConstructMatch            ConstructKind = "match"
	ConstructWhen             ConstructKind = "when"
	ConstructForComprehension ConstructKind = "for_comprehension"
	ConstructTryCatch         ConstructKind = "try_catch"
	ConstructTryExcept        ConstructKind = "try_except"
	ConstructTryWithResources ConstructKind = "try_with_resources"
	ConstructSynchronized     ConstructKind = "synchronized"
	ConstructLock             ConstructKind = "lock"
	ConstructUsing            ConstructKind = "using"
	ConstructQuery            ConstructKind = "query"
	ConstructYield            ConstructKind = "yield"
	ConstructWith             ConstructKind = "with"
	ConstructRule             ConstructKind = "rule"
	ConstructVariable         ConstructKind = "variable"
	ConstructInclude          ConstructKind = "include"
)

// ConstructElement names one keyed map entry or struct literal field insertion.
const ConstructElement ConstructKind = "element"

const (
	// LanguageAuto detects the language from project context.
	LanguageAuto LanguageID = "auto"
	// LanguageGo selects the built-in Go backend.
	LanguageGo LanguageID = "go"
	// LanguageRust selects the read-only Rust lookup backend.
	LanguageRust LanguageID = "rust"
	// LanguageJava selects the read-only Java lookup backend.
	LanguageJava LanguageID = "java"
	// LanguageScala selects the read-only Scala lookup backend.
	LanguageScala LanguageID = "scala"
	// LanguageHaskell selects the trusted standalone Haskell lookup backend.
	LanguageHaskell LanguageID = "haskell"
	// LanguageKotlin selects the trusted read-only Kotlin backend.
	LanguageKotlin LanguageID = "kotlin"
	// LanguageBash identifies shell sources for detection; no backend is registered yet.
	LanguageBash LanguageID = "bash"
	// LanguageMake identifies Make sources for detection; no backend is registered yet.
	LanguageMake LanguageID = "make"
)

// Operation identifies a service operation for capability checks.
type Operation string

const (
	// OperationLookup resolves a symbol location.
	OperationLookup Operation = "lookup"
	// OperationRename performs a semantic rename.
	OperationRename Operation = "rename"
	// OperationVerify formats and checks project diagnostics.
	OperationVerify Operation = "verify"
	// OperationOrganizeImports updates and organizes source imports.
	OperationOrganizeImports Operation = "organize_imports"
	// OperationAddBuildDependency updates the selected language's build manifest.
	OperationAddBuildDependency Operation = "add_build_dependency"
	// OperationInsertConstruct inserts one syntax construct.
	OperationInsertConstruct Operation = "insert_construct"
	// OperationReplaceConstruct replaces one selected syntax construct.
	OperationReplaceConstruct Operation = "replace_construct"
	// OperationReplaceBody replaces the statements inside one function or method body.
	OperationReplaceBody Operation = "replace_body"
	// OperationReplaceDecl replaces one selected declaration.
	OperationReplaceDecl Operation = "replace_decl"
	// OperationInspectConstruct inspects candidates for one nested replacement selector.
	OperationInspectConstruct Operation = "inspect_construct"
	// OperationRemoveConstruct removes one selected syntax construct.
	OperationRemoveConstruct Operation = "remove_construct"
	// OperationExtractVariable extracts one selected expression into a local.
	OperationExtractVariable Operation = "extract_variable"
	// OperationInlineVariable inlines one selected local declaration.
	OperationInlineVariable Operation = "inline_variable"
	// OperationInlineFunction inlines one selected Java invocation.
	OperationInlineFunction Operation = "inline_function"
	// OperationImplementContract generates missing methods for an implemented Java interface.
	OperationImplementContract Operation = "implement_contract"
	// OperationExtractFunction extracts selected statements into a method.
	OperationExtractFunction Operation = "extract_function"
	// OperationExtractConstant extracts one selected compile-time expression into a constant.
	OperationExtractConstant Operation = "extract_constant"
	// OperationMoveConstruct moves one existing syntax construct.
	OperationMoveConstruct Operation = "move_construct"
	// OperationChangeSignature updates a Java method signature and resolved call sites.
	OperationChangeSignature Operation = "change_signature"
	// OperationIntroduceParameter updates one Java method expression and its resolved call sites.
	OperationIntroduceParameter Operation = "introduce_parameter"
	// OperationIntroduceParameterObject replaces a Java parameter list with a value object.
	OperationIntroduceParameterObject Operation = "introduce_parameter_object"
	// CapabilityLookup is the lookup capability key.
	CapabilityLookup = OperationLookup
	// CapabilityRename is the rename capability key.
	CapabilityRename = OperationRename
	// CapabilityVerify is the verify capability key.
	CapabilityVerify = OperationVerify
)

var (
	// ErrInvalidLanguage indicates a malformed language selection.
	ErrInvalidLanguage = errors.New("invalid language")
	// ErrLanguageUnavailable indicates no registered backend for a language.
	ErrLanguageUnavailable = errors.New("language backend unavailable")
	// ErrLanguageUndetected indicates auto-selection found no supported language.
	ErrLanguageUndetected = errors.New("language could not be detected")
	// ErrBackendAlreadyDefined indicates duplicate registration.
	ErrBackendAlreadyDefined = errors.New("language backend already registered")
	// ErrUnsupportedOperation indicates a capability is not available.
	ErrUnsupportedOperation = errors.New("operation is not supported by language backend")
	// ErrAmbiguous indicates a rename target has multiple matches.
	ErrAmbiguous = errors.New("ambiguous symbol")
	// ErrInvalidProject indicates missing project context.
	ErrInvalidProject = errors.New("invalid project context")
	// ErrWorkspaceTrustRequired indicates an external-tool operation needs explicit consent.
	ErrWorkspaceTrustRequired = errors.New("workspace trust required")
)

// WorkspaceTrust is an explicit, in-memory consent scoped to one canonical workspace root.
// The zero value is untrusted and no consent is persisted between requests.
type WorkspaceTrust struct {
	WorkspaceRoot string `json:"workspace_root,omitempty"`
	Trusted       bool   `json:"trusted,omitempty"`
}

// CanonicalWorkspaceRoot returns the cleaned absolute path used for trust comparisons.
// Existing symlinks are resolved, while not-yet-created roots still receive a stable absolute path.
func CanonicalWorkspaceRoot(root string) string {
	return pathutil.CanonicalWorkspaceRoot(root)
}

// NewWorkspaceTrust creates request-scoped consent for the supplied workspace root.
func NewWorkspaceTrust(root string, trusted bool) WorkspaceTrust {
	return WorkspaceTrust{WorkspaceRoot: CanonicalWorkspaceRoot(root), Trusted: trusted}
}

// Allows reports whether this consent applies to the requested canonical workspace root.
func (t WorkspaceTrust) Allows(root string) bool {
	return t.Trusted && t.WorkspaceRoot != "" && CanonicalWorkspaceRoot(t.WorkspaceRoot) == CanonicalWorkspaceRoot(root)
}

// WorkspaceTrustError identifies an operation that was refused without explicit consent.
type WorkspaceTrustError struct {
	Operation Operation
	Language  LanguageID
	Workspace string
}

func (e *WorkspaceTrustError) Error() string {
	parts := []string{ErrWorkspaceTrustRequired.Error()}
	if e.Operation != "" {
		parts = append(parts, string(e.Operation))
	}
	if e.Language != "" {
		parts = append(parts, string(e.Language))
	}
	if e.Workspace != "" {
		parts = append(parts, e.Workspace)
	}
	return strings.Join(parts, ": ")
}

func (e *WorkspaceTrustError) Unwrap() error { return ErrWorkspaceTrustRequired }

// Error is a typed service boundary error. Callers can use errors.Is and errors.As.
type Error struct {
	Operation Operation
	Language  LanguageID
	Err       error
}

func (e *Error) Error() string {
	parts := make([]string, 0, 3)
	if e.Operation != "" {
		parts = append(parts, string(e.Operation))
	}
	if e.Language != "" {
		parts = append(parts, string(e.Language))
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *Error) Unwrap() error { return e.Err }

// Position is a protocol position using zero-based lines and UTF-16 character units.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a protocol source range.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// SourceLocation identifies a source range without exposing language-specific coordinates.
type SourceLocation struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic is a language-neutral verification diagnostic.
type Diagnostic struct {
	Message  string          `json:"message"`
	Severity int             `json:"severity,omitempty"`
	Location *SourceLocation `json:"location,omitempty"`
}

// DiagnosticDelta reports verification changes without exposing pipeline internals.
type DiagnosticDelta struct {
	Before      []string `json:"-"`
	After       []string `json:"-"`
	NetDelta    int      `json:"net_delta"`
	Introduced  []string `json:"introduced"`
	Resolved    []string `json:"resolved"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// Capabilities declares the operations a backend can execute.
type Capabilities struct {
	Operations             map[Operation]bool `json:"operations"`
	RequiresWorkspaceTrust map[Operation]bool `json:"requires_workspace_trust,omitempty"`
}

// NewCapabilities constructs a capability set from supported operations.
func NewCapabilities(operations ...Operation) Capabilities {
	set := make(map[Operation]bool, len(operations))
	for _, operation := range operations {
		set[operation] = true
	}
	return Capabilities{Operations: set}
}

// NewCapabilitiesRequiringWorkspaceTrust declares external-tool operations that need explicit consent.
func NewCapabilitiesRequiringWorkspaceTrust(operations ...Operation) Capabilities {
	capabilities := NewCapabilities(operations...)
	capabilities.RequiresWorkspaceTrust = make(map[Operation]bool, len(operations))
	for _, operation := range operations {
		capabilities.RequiresWorkspaceTrust[operation] = true
	}
	return capabilities
}

// Supports reports whether operation is declared by the capability set.
func (c Capabilities) Supports(operation Operation) bool { return c.Operations[operation] }

// RequiresTrust reports whether operation may invoke an external project tool.
func (c Capabilities) RequiresTrust(operation Operation) bool {
	return c.RequiresWorkspaceTrust[operation]
}

// OpCapability is an alias for capability.OpCapability.
// Use semedit/internal/capability directly when building LanguageMatrix values.
type OpCapability = capability.OpCapability

// Constraint is an alias for capability.Constraint.
type Constraint = capability.Constraint

// LanguageMatrix is an alias for capability.LanguageMatrix.
type LanguageMatrix = capability.LanguageMatrix

// MatrixProvider is an optional extension of Backend. Backends that implement it
// expose their declarative LanguageMatrix so that cmd/docgen can build capability
// documentation from the registry rather than from hardcoded tables.
type MatrixProvider interface {
	CapabilityMatrix() capability.LanguageMatrix
}

// ConstructCapabilityProvider reports executable construct kinds per operation.
type ConstructCapabilityProvider interface {
	SupportedConstructs(Operation) []ConstructKind
}

// ProjectContext identifies the project and optional source file selected by an ingress.
type ProjectContext struct {
	RootDir        string
	File           string
	Language       LanguageID
	WorkspaceTrust WorkspaceTrust
	Java           JavaConfig
	Scala          ScalaConfig
	Haskell        HaskellConfig
	Kotlin         KotlinConfig
	Bash           BashConfig
	Make           MakeConfig
}

// BashConfig identifies an explicitly preinstalled Bash language server.
type BashConfig struct {
	BashBin string `json:"bash_bin,omitempty"`
}

// MakeConfig identifies an explicitly preinstalled Make language server.
type MakeConfig struct {
	MakeBin string `json:"make_bin,omitempty"`
}

// NamedTarget identifies a named source target by its exact identifier range.
type NamedTarget struct {
	Name             string         `json:"-"`
	QualifiedName    string         `json:"-"`
	Kind             string         `json:"-"`
	Location         SourceLocation `json:"-"`
	DeclarationRange *Range         `json:"-"`
	Signature        *string        `json:"-"`
}

// Identity returns the qualified identity when present, falling back to the simple name.
func (target *NamedTarget) Identity() string {
	if target == nil {
		return ""
	}
	if target.QualifiedName != "" {
		return target.QualifiedName
	}
	return target.Name
}

// LookupResult represents a resolved target or the available ambiguous targets.
type LookupResult struct {
	Definition *NamedTarget   `json:"-"`
	Usages     []*NamedTarget `json:"-"`
	Candidates []*NamedTarget `json:"-"`
	SourceView string         `json:"-"`
}

// IsAmbiguous reports whether lookup found multiple candidate targets.
func (result *LookupResult) IsAmbiguous() bool {
	return result != nil && len(result.Candidates) >= 2
}

// RenameRequest describes a semantic rename without language-specific coordinates.
type RenameRequest struct {
	Project           ProjectContext
	Symbol            string
	To                string
	OrganizeImports   bool
	DeferFormatting   bool
	DeferVerification bool
}

// NormalizeRenameInput trims whitespace and removes one matching outer quote pair.
// A lone trailing apostrophe is valid in languages such as Haskell and is preserved.
func NormalizeRenameInput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) >= 2 {
		first, last := trimmed[0], trimmed[len(trimmed)-1]
		if (first == '\'' || first == '"') && first == last {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}

// RenameResult reports the resolved target and diagnostic delta after a rename.
type RenameResult struct {
	Lookup                  *LookupResult
	Diagnostics             DiagnosticDelta
	Active                  []string
	Source                  *LookupResult `json:"source,omitempty"`
	Revision                string        `json:"revision,omitempty"`
	ApplicationState        string        `json:"application_state,omitempty"`
	ContinuationUnavailable string        `json:"continuation_unavailable,omitempty"`
	TargetFile              string        `json:"-"`
	ProjectRoot             string        `json:"-"`
}

// RenameDiagnosticsError reports diagnostics introduced by an applied rename.
// The edit remains on disk so callers can continue a staged refactoring.
type RenameDiagnosticsError struct {
	Result           *RenameResult
	IntroducedErrors []string
}

func (e *RenameDiagnosticsError) Error() string {
	if e == nil || e.Result == nil {
		return "rename applied with compilation diagnostics"
	}
	introduced := e.IntroducedErrors
	if introduced == nil {
		introduced = e.Result.Diagnostics.Introduced
	}
	return fmt.Sprintf("rename applied with %d introduced error diagnostics: %s", len(introduced), strings.Join(introduced, "; "))
}

// VerifyRequest describes a verification operation.
type VerifyRequest struct {
	Project            ProjectContext
	Path               string
	FormatSelectedFile bool
	OrganizeImports    bool
	AddImports         []string
	RemoveImports      []string
	CheckOnly          bool
	GoPackagePatterns  []string
}

// VerifyResult contains diagnostics in their protocol-neutral form.
type VerifyResult struct {
	Diagnostics []Diagnostic
}

// Backend is the language adapter contract. Coordinates never cross this boundary.
type Backend interface {
	Language() LanguageID
	Capabilities() Capabilities
	Lookup(context.Context, ProjectContext, string) (*LookupResult, error)
	Rename(context.Context, RenameRequest) (*RenameResult, error)
	Verify(context.Context, VerifyRequest) ([]Diagnostic, error)
}

type trustedWorkspaceRootProvider interface {
	TrustedWorkspaceRoot(ProjectContext) (string, error)
}

// Registry stores one backend per language and selects the requested or detected backend.
type Registry struct {
	backends map[LanguageID]Backend
}

// NewRegistry constructs a registry and registers the supplied backends.
func NewRegistry(backends ...Backend) (*Registry, error) {
	r := &Registry{backends: make(map[LanguageID]Backend, len(backends))}
	for _, b := range backends {
		if err := r.Register(b); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Register adds a backend, rejecting duplicate language IDs.
func (r *Registry) Register(b Backend) error {
	if b == nil || b.Language() == "" || b.Language() == LanguageAuto {
		return &Error{Err: ErrInvalidLanguage}
	}
	if _, exists := r.backends[b.Language()]; exists {
		return &Error{Language: b.Language(), Err: ErrBackendAlreadyDefined}
	}
	r.backends[b.Language()] = b
	return nil
}

// Backend returns the backend registered for language, if any.
func (r *Registry) Backend(language LanguageID) (Backend, bool) {
	b, ok := r.backends[language]
	return b, ok
}

// Select chooses an explicitly requested backend or detects one from project context.
func (r *Registry) Select(project ProjectContext) (Backend, error) {
	language := project.Language
	if language == "" {
		language = LanguageAuto
	}
	if language == LanguageAuto {
		var err error
		language, err = r.detectLanguage(project)
		if err != nil {
			return nil, &Error{Operation: OperationLookup, Err: err}
		}
		if language == "" {
			return nil, &Error{Operation: OperationLookup, Err: ErrLanguageUndetected}
		}
	}
	b, ok := r.backends[language]
	if !ok {
		return nil, &Error{Language: language, Err: ErrLanguageUnavailable}
	}
	return b, nil
}

func (r *Registry) detectLanguage(project ProjectContext) (LanguageID, error) {
	root, target, sourceLanguage, err := resolveDetectionTarget(project)
	if err != nil {
		return "", err
	}
	if r.hasBackend(sourceLanguage) {
		return sourceLanguage, nil
	}
	// An unsupported source-file selection must not fall back to an unrelated
	// workspace manifest (for example, a Haskell file inside a Go repository).
	if sourceLanguage != "" {
		return "", nil
	}
	direct, err := r.directLanguages(target)
	if err != nil {
		return "", err
	}
	if len(direct) == 1 {
		return direct[0], nil
	}
	languages, err := r.detectedLanguages(target)
	if err != nil {
		return "", err
	}
	return r.selectDetectedLanguage(root, target, languages)
}

func resolveDetectionTarget(project ProjectContext) (string, string, LanguageID, error) {
	root := project.RootDir
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve language detection root %q: %w", root, err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve language detection root symlinks %q: %w", absRoot, err)
	}
	absRoot = canonicalRoot
	target := absRoot
	if project.File != "" {
		target = project.File
		if !filepath.IsAbs(target) {
			target = filepath.Join(absRoot, target)
		}
	}
	target = filepath.Clean(target)
	sourceLanguage := LanguageIDFromFile(target)
	canonicalTarget, err := filepath.EvalSymlinks(target)
	if err != nil && !os.IsNotExist(err) {
		return "", "", "", fmt.Errorf("resolve language detection target symlinks %q: %w", target, err)
	}
	if err == nil {
		target = canonicalTarget
	}
	if !pathWithin(absRoot, target) {
		return "", "", "", fmt.Errorf("language detection target %q is outside workspace root %q", target, absRoot)
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) && sourceLanguage != "" {
			return absRoot, filepath.Dir(target), sourceLanguage, nil
		}
		return "", "", "", fmt.Errorf("inspect language detection target %q: %w", target, err)
	}
	if !targetInfo.IsDir() {
		if sourceLanguage == "" {
			sourceLanguage = LanguageIDFromFile(target)
		}
		target = filepath.Dir(target)
	}
	return absRoot, target, sourceLanguage, nil
}

func (r *Registry) hasBackend(language LanguageID) bool {
	if language == "" || language == LanguageHaskell {
		return false
	}
	_, ok := r.backends[language]
	return ok
}

func (r *Registry) directLanguages(target string) ([]LanguageID, error) {
	entries, err := os.ReadDir(target)
	if err != nil {
		return nil, fmt.Errorf("read language detection target %q: %w", target, err)
	}
	var result []LanguageID
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		language := LanguageIDFromFile(entry.Name())
		if r.hasBackend(language) && !slices.Contains(result, language) {
			result = append(result, language)
		}
	}
	return result, nil
}

func (r *Registry) detectedLanguages(target string) ([]LanguageID, error) {
	languages, err := DetectLanguages(target)
	if err != nil {
		return nil, fmt.Errorf("detect languages under target %q: %w", target, err)
	}
	var result []LanguageID
	for _, language := range languages {
		if r.hasBackend(language) {
			result = append(result, language)
		}
	}
	return result, nil
}

func (r *Registry) selectDetectedLanguage(root, target string, languages []LanguageID) (LanguageID, error) {
	goSource := slices.Contains(languages, LanguageGo)
	javaSource := slices.Contains(languages, LanguageJava)
	if goSource && !javaSource {
		return LanguageGo, nil
	}
	if javaSource && !goSource {
		return LanguageJava, nil
	}
	boundary, err := discoveryBoundary(target)
	if err != nil {
		return "", err
	}
	goContext, err := goWorkspaceContext(boundary, target, root)
	if err != nil {
		return "", err
	}
	if goSource || goContext {
		return LanguageGo, nil
	}
	javaContext, err := javaWorkspaceContext(boundary, target)
	if err != nil {
		return "", err
	}
	if javaSource || javaContext {
		return LanguageJava, nil
	}
	if len(languages) == 1 {
		return languages[0], nil
	}
	return "", nil
}

// Service is the ingress-facing orchestration boundary shared by CLI and MCP.
type Service struct {
	Registry *Registry
}

// NewService constructs the shared ingress-facing service.
func NewService(registry *Registry) *Service { return &Service{Registry: registry} }

func (s *Service) backendFor(project ProjectContext, operation Operation) (Backend, error) {
	if s == nil || s.Registry == nil {
		return nil, &Error{Operation: operation, Err: ErrLanguageUnavailable}
	}
	b, err := s.Registry.Select(project)
	if err != nil {
		if boundaryErr, ok := errors.AsType[*Error](err); ok {
			boundaryErr.Operation = operation
			return nil, err
		}
		return nil, &Error{Operation: operation, Err: err}
	}
	if !b.Capabilities().Supports(operation) {
		return nil, &Error{Operation: operation, Language: b.Language(), Err: ErrUnsupportedOperation}
	}
	if b.Capabilities().RequiresTrust(operation) && !project.WorkspaceTrust.Allows(project.RootDir) {
		trustRoot := project.RootDir
		if provider, ok := b.(trustedWorkspaceRootProvider); ok && trustRoot == "" {
			if discovered, discoverErr := provider.TrustedWorkspaceRoot(project); discoverErr == nil {
				trustRoot = discovered
			}
		}
		if project.WorkspaceTrust.Allows(trustRoot) {
			return b, nil
		}
		return nil, &WorkspaceTrustError{
			Operation: operation,
			Language:  b.Language(),
			Workspace: CanonicalWorkspaceRoot(trustRoot),
		}
	}
	return b, nil
}

// Lookup resolves a symbol through the selected backend.
func (s *Service) Lookup(ctx context.Context, project ProjectContext, symbol string) (*LookupResult, error) {
	b, err := s.backendFor(project, OperationLookup)
	if err != nil {
		return nil, err
	}
	return b.Lookup(ctx, project, symbol)
}

// Rename resolves and executes a rename through the selected backend.
func (s *Service) Rename(ctx context.Context, request RenameRequest) (*RenameResult, error) {
	request.Symbol = NormalizeRenameInput(request.Symbol)
	request.To = NormalizeRenameInput(request.To)
	b, err := s.backendFor(request.Project, OperationRename)
	if err != nil {
		return nil, err
	}
	return b.Rename(ctx, request)
}

// Verify formats and checks diagnostics through the selected backend.
func (s *Service) Verify(ctx context.Context, request VerifyRequest) (*VerifyResult, error) {
	if request.CheckOnly {
		request.FormatSelectedFile = false
		request.OrganizeImports = false
	}
	b, err := s.backendFor(request.Project, OperationVerify)
	if err != nil {
		return nil, err
	}
	returnResult, err := b.Verify(ctx, request)
	if err != nil {
		return nil, err
	}
	return &VerifyResult{Diagnostics: returnResult}, nil
}

// NotifyProjectConfigurationUpdate refreshes an already-running Java language-server project after manifest edits.
func (s *Service) NotifyProjectConfigurationUpdate(ctx context.Context, project ProjectContext, changedPOMs []string) error {
	b, err := s.backendFor(project, OperationAddBuildDependency)
	if err != nil {
		return err
	}
	notifier, ok := b.(interface {
		NotifyProjectConfigurationUpdate(context.Context, ProjectContext, []string) error
	})
	if !ok {
		return nil
	}
	return notifier.NotifyProjectConfigurationUpdate(ctx, project, changedPOMs)
}

// UTF16Length returns the number of UTF-16 code units in text.
func UTF16Length(text string) int { return len(utf16.Encode([]rune(text))) }

// PositionFromByteOffset converts a UTF-8 byte offset to a zero-based UTF-16 position.
func PositionFromByteOffset(source []byte, offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	lineStart := 0
	line := 0
	for i, b := range source[:offset] {
		if b == '\n' {
			line++
			lineStart = i + 1
		}
	}
	return Position{Line: line, Character: UTF16Length(string(source[lineStart:offset]))}
}

func pathWithin(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func discoveryBoundary(target string) (string, error) {
	target, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve discovery target %q: %w", target, err)
	}
	for dir := target; ; dir = filepath.Dir(dir) {
		marker := filepath.Join(dir, ".git")
		if _, err := os.Stat(marker); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect repository boundary %q: %w", marker, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, nil
		}
	}
}

func goWorkspaceContext(root, target, scopeRoot string) (bool, error) {
	stopAt := root
	if ignoredBoundary := ignoredScopeBoundary(root, scopeRoot); ignoredBoundary != "" {
		stopAt = ignoredBoundary
	}
	for dir := target; ; dir = filepath.Dir(dir) {
		modulePath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modulePath); err == nil {
			if dir == scopeRoot && target == scopeRoot {
				return false, nil
			}
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("inspect Go module manifest %q: %w", modulePath, err)
		}
		workPath := filepath.Join(dir, "go.work")
		data, err := readFileWithin(root, workPath)
		if err == nil {
			work, parseErr := modfile.ParseWork(workPath, data, nil)
			if parseErr != nil {
				return false, fmt.Errorf("parse Go workspace manifest %q: %w", workPath, parseErr)
			}
			for _, use := range work.Use {
				moduleDir := use.Path
				if !filepath.IsAbs(moduleDir) {
					moduleDir = filepath.Join(dir, moduleDir)
				}
				moduleDir = filepath.Clean(moduleDir)
				if pathWithin(moduleDir, target) || pathWithin(target, moduleDir) || target == dir {
					return true, nil
				}
			}
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("read Go workspace manifest %q: %w", workPath, err)
		}
		if dir == stopAt {
			break
		}
	}
	return false, nil
}

func javaWorkspaceContext(root, target string) (bool, error) {
	stopAt := root
	if ignoredBoundary := ignoredScopeBoundary(root, target); ignoredBoundary != "" {
		stopAt = ignoredBoundary
	}
	for dir := target; ; dir = filepath.Dir(dir) {
		for _, name := range []string{"build.gradle", "build.gradle.kts"} {
			marker := filepath.Join(dir, name)
			if _, err := os.Stat(marker); err == nil {
				return true, nil
			} else if !os.IsNotExist(err) {
				return false, fmt.Errorf("inspect Java build marker %q: %w", marker, err)
			}
		}
		pomPath := filepath.Join(dir, "pom.xml")
		data, err := readFileWithin(root, pomPath)
		if err == nil {
			var pom struct {
				Modules []string `xml:"modules>module"`
			}
			if decodeErr := xml.Unmarshal(data, &pom); decodeErr != nil {
				return false, fmt.Errorf("parse Maven manifest %q: %w", pomPath, decodeErr)
			}
			if target == dir {
				return true, nil
			}
			for _, module := range pom.Modules {
				moduleDir := filepath.Clean(filepath.Join(dir, module))
				if pathWithin(moduleDir, target) || pathWithin(target, moduleDir) {
					return true, nil
				}
			}
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("read Maven manifest %q: %w", pomPath, err)
		}
		if dir == stopAt {
			break
		}
	}
	return false, nil
}

func ignoredScopeBoundary(root, scope string) string {
	rel, err := filepath.Rel(root, scope)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	dir := root
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		next := filepath.Join(dir, part)
		if sourcefiles.IsIgnoredDirectoryPath(next) {
			return next
		}
		dir = next
	}
	return ""
}

func readFileWithin(root, path string) ([]byte, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path %q is outside discovery root %q", path, root)
	}
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open discovery root %q: %w", root, err)
	}
	file, openErr := rootFS.Open(rel)
	if openErr != nil {
		closeErr := rootFS.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("close discovery root %q: %w", root, closeErr)
		}
		return nil, openErr
	}
	data, readErr := io.ReadAll(file)
	fileCloseErr := file.Close()
	rootCloseErr := rootFS.Close()
	if readErr != nil {
		return nil, readErr
	}
	if fileCloseErr != nil {
		return nil, fileCloseErr
	}
	if rootCloseErr != nil {
		return nil, rootCloseErr
	}
	return data, nil
}
