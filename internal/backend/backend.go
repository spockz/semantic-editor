// Package backend owns the language-neutral service boundary used by CLI and MCP ingress.
package backend

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"semedit/internal/pipeline"
)

// LanguageID identifies a source language supported by a backend registry.
type LanguageID string

const (
	// LanguageAuto detects the language from project context.
	LanguageAuto LanguageID = "auto"
	// LanguageGo selects the built-in Go backend.
	LanguageGo LanguageID = "go"
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
)

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
	Before      []string `json:"before"`
	After       []string `json:"after"`
	NetDelta    int      `json:"net_delta"`
	Introduced  []string `json:"introduced"`
	Resolved    []string `json:"resolved"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// Capabilities declares the operations a backend can execute.
type Capabilities struct {
	Operations map[Operation]bool `json:"operations"`
}

// NewCapabilities constructs a capability set from supported operations.
func NewCapabilities(operations ...Operation) Capabilities {
	set := make(map[Operation]bool, len(operations))
	for _, operation := range operations {
		set[operation] = true
	}
	return Capabilities{Operations: set}
}

// Supports reports whether operation is declared by the capability set.
func (c Capabilities) Supports(operation Operation) bool { return c.Operations[operation] }

// ProjectContext identifies the project and optional source file selected by an ingress.
type ProjectContext struct {
	RootDir  string
	File     string
	Language LanguageID
}

// SymbolCandidate is the neutral representation of an ambiguous symbol.
type SymbolCandidate struct {
	Name          string         `json:"name"`
	Receiver      string         `json:"receiver,omitempty"`
	QualifiedName string         `json:"qualified_name,omitempty"`
	Kind          string         `json:"kind"`
	File          string         `json:"file"`
	Line          int            `json:"line,omitempty"`
	Column        int            `json:"column,omitempty"`
	Offset        int            `json:"offset,omitempty"`
	Location      SourceLocation `json:"-"`
}

// LookupResult retains legacy fields for CLI/MCP compatibility and adds a neutral location.
type LookupResult struct {
	Symbol     string             `json:"symbol,omitempty"`
	File       string             `json:"file,omitempty"`
	Line       int                `json:"line,omitempty"`
	Column     int                `json:"column,omitempty"`
	Offset     int                `json:"offset,omitempty"`
	Kind       string             `json:"kind,omitempty"`
	Receiver   string             `json:"receiver,omitempty"`
	Ambiguous  bool               `json:"ambiguous,omitempty"`
	Candidates []*SymbolCandidate `json:"candidates,omitempty"`
	Location   SourceLocation     `json:"-"`
}

// RenameRequest describes a semantic rename without language-specific coordinates.
type RenameRequest struct {
	Project         ProjectContext
	Symbol          string
	To              string
	OrganizeImports bool
}

// RenameResult reports the resolved target and diagnostic delta after a rename.
type RenameResult struct {
	Lookup      *LookupResult
	Diagnostics DiagnosticDelta
	Active      []string
}

// VerifyRequest describes a verification operation.
type VerifyRequest struct {
	Project ProjectContext
	Path    string
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
	Rename(context.Context, ProjectContext, *LookupResult, string) error
	Verify(context.Context, ProjectContext, string) ([]Diagnostic, error)
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
		language = detectLanguage(project)
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

func detectLanguage(project ProjectContext) LanguageID {
	if strings.EqualFold(filepath.Ext(project.File), ".go") {
		return LanguageGo
	}
	root := project.RootDir
	if root == "" {
		root = "."
	}
	for _, name := range []string{"go.mod", "go.work"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return LanguageGo
		}
	}
	return ""
}

// Service is the ingress-facing orchestration boundary shared by CLI and MCP.
type Service struct {
	Registry *Registry
}

// NewService constructs the shared ingress-facing service.
func NewService(registry *Registry) *Service { return &Service{Registry: registry} }

// NewDefaultService constructs a service with the built-in Go backend.
func NewDefaultService() *Service {
	registry, err := NewRegistry(NewGoBackend())
	if err != nil {
		panic(fmt.Sprintf("register built-in backends: %v", err))
	}
	return NewService(registry)
}

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
	b, err := s.backendFor(request.Project, OperationRename)
	if err != nil {
		return nil, err
	}
	lookup, err := b.Lookup(ctx, request.Project, request.Symbol)
	if err != nil {
		return nil, err
	}
	if lookup.Ambiguous {
		return nil, &Error{Operation: OperationRename, Err: ErrAmbiguous}
	}
	before, _ := pipeline.CheckDiagnostics(ctx, request.Project.RootDir)
	if err := b.Rename(ctx, request.Project, lookup, request.To); err != nil {
		return nil, err
	}
	if request.OrganizeImports {
		_ = pipeline.OrganizeImports(ctx, request.Project.RootDir, ".")
	} else {
		_ = pipeline.Format(ctx, request.Project.RootDir, ".")
	}
	after, _ := pipeline.CheckDiagnostics(ctx, request.Project.RootDir)
	delta := pipeline.ComputeDelta(before, after)
	return &RenameResult{
		Lookup: lookup,
		Diagnostics: DiagnosticDelta{
			Before:      delta.Before,
			After:       delta.After,
			NetDelta:    delta.NetDelta,
			Introduced:  delta.Introduced,
			Resolved:    delta.Resolved,
			Suggestions: delta.Suggestions,
		},
		Active: after,
	}, nil
}

// Verify formats and checks diagnostics through the selected backend.
func (s *Service) Verify(ctx context.Context, request VerifyRequest) (*VerifyResult, error) {
	b, err := s.backendFor(request.Project, OperationVerify)
	if err != nil {
		return nil, err
	}
	returnResult, err := b.Verify(ctx, request.Project, request.Path)
	if err != nil {
		return nil, err
	}
	return &VerifyResult{Diagnostics: returnResult}, nil
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

func fileURI(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}
