// Package backend provides neutral request and result contracts for optional semantic read operations.
package backend

import (
	"context"
	"errors"
	"fmt"
)

const (
	// OperationInspect returns source and structured declaration metadata.
	OperationInspect Operation = "inspect_symbol"
	// OperationOutline returns structured declarations for a file or directory.
	OperationOutline Operation = "outline"
	// OperationFindReferences returns typed source references.
	OperationFindReferences Operation = "find_references"

	// CapabilityInspect is the inspect capability key.
	CapabilityInspect = OperationInspect
	// CapabilityOutline is the outline capability key.
	CapabilityOutline = OperationOutline
	// CapabilityFindReferences is the find_references capability key.
	CapabilityFindReferences = OperationFindReferences
)

// ReadScope describes the source region covered by a read result.
type ReadScope struct {
	Kind        string   `json:"kind"`
	Path        string   `json:"path"`
	Complete    bool     `json:"complete"`
	Limitations []string `json:"limitations,omitempty"`
}

// ReadImport describes an import path and its optional explicit local name.
type ReadImport struct {
	Path string  `json:"path"`
	Name *string `json:"name,omitempty"`
}

// ReadSymbol is the shared language-neutral projection of a declaration.
type ReadSymbol struct {
	Name           string       `json:"name"`
	QualifiedName  string       `json:"qualified_name"`
	Kind           string       `json:"kind"`
	File           string       `json:"file"`
	Range          Range        `json:"range"`
	SelectionRange Range        `json:"selection_range"`
	Signature      *string      `json:"signature,omitempty"`
	Doc            *string      `json:"doc,omitempty"`
	Type           *string      `json:"type,omitempty"`
	Tag            *string      `json:"tag,omitempty"`
	Children       []ReadSymbol `json:"children,omitempty"`
	ServerDetail   *string      `json:"server_detail,omitempty"`
}

// InspectedSymbol adds an exact source snapshot and contextual metadata to a declaration.
type InspectedSymbol struct {
	ReadSymbol

	Source       string       `json:"source"`
	Revision     string       `json:"revision"`
	SourceExtent string       `json:"source_extent,omitempty"`
	Package      string       `json:"package,omitempty"`
	Imports      []ReadImport `json:"imports"`
}

// InspectResult contains every declaration match within its scope.
type InspectResult struct {
	Scope   ReadScope         `json:"scope"`
	Matches []InspectedSymbol `json:"matches"`
}

// OutlineFile contains the symbols and file metadata from one source snapshot.
type OutlineFile struct {
	File     string       `json:"file"`
	Language LanguageID   `json:"language"`
	Revision string       `json:"revision"`
	Package  string       `json:"package,omitempty"`
	Doc      *string      `json:"doc,omitempty"`
	Imports  []ReadImport `json:"imports"`
	Symbols  []ReadSymbol `json:"symbols"`
}

// OutlineResult contains per-file outline data for its declared scope.
type OutlineResult struct {
	Scope ReadScope     `json:"scope"`
	Files []OutlineFile `json:"files"`
}

// Reference identifies a source use of a resolved declaration.
type Reference struct {
	File            string `json:"file"`
	Range           Range  `json:"range"`
	EnclosingSymbol string `json:"enclosing_symbol,omitempty"`
	Snippet         string `json:"snippet"`
}

// FileRevision records the source snapshot used for reference analysis.
type FileRevision struct {
	File     string `json:"file"`
	Revision string `json:"revision"`
}

// ReferencesResult contains the resolved symbol, its uses, and analyzed source snapshots.
type ReferencesResult struct {
	Scope      ReadScope      `json:"scope"`
	Symbol     ReadSymbol     `json:"symbol"`
	References []Reference    `json:"references"`
	Files      []FileRevision `json:"files"`
}

// InspectRequest selects a symbol and optional file scope for inspection.
type InspectRequest struct {
	Project ProjectContext
	Symbol  string
}

// OutlineRequest selects a file or directory and filters its declaration outline.
type OutlineRequest struct {
	Project           ProjectContext
	Path              string
	Kinds             []string
	IncludeUnexported bool
	IncludeTests      bool
}

// ReferencesRequest selects the declaration whose typed references are requested.
type ReferencesRequest struct {
	Project ProjectContext
	Symbol  string
}

// InspectBackend is an optional backend extension for source inspection.
type InspectBackend interface {
	Inspect(context.Context, InspectRequest) (*InspectResult, error)
}

// OutlineBackend is an optional backend extension for declaration outlines.
type OutlineBackend interface {
	Outline(context.Context, OutlineRequest) (*OutlineResult, error)
}

// ReferencesBackend is an optional backend extension for typed reference search.
type ReferencesBackend interface {
	FindReferences(context.Context, ReferencesRequest) (*ReferencesResult, error)
}

// ErrBackendInterfaceMismatch means a backend advertises a read operation without implementing its optional interface.
var ErrBackendInterfaceMismatch = errors.New("backend capability has no corresponding optional interface")

func readBackendMismatch(b Backend, operation Operation, interfaceName string) error {
	return &Error{
		Operation: operation,
		Language:  b.Language(),
		Err:       fmt.Errorf("%w: backend does not implement %s", ErrBackendInterfaceMismatch, interfaceName),
	}
}

// Inspect dispatches source inspection after capability and workspace-trust checks.
func (s *Service) Inspect(ctx context.Context, request InspectRequest) (*InspectResult, error) {
	b, err := s.backendFor(request.Project, OperationInspect)
	if err != nil {
		return nil, err
	}
	reader, ok := b.(InspectBackend)
	if !ok {
		return nil, readBackendMismatch(b, OperationInspect, "InspectBackend")
	}
	return reader.Inspect(ctx, request)
}

// Outline dispatches a source outline after capability and workspace-trust checks.
func (s *Service) Outline(ctx context.Context, request OutlineRequest) (*OutlineResult, error) {
	b, err := s.backendFor(request.Project, OperationOutline)
	if err != nil {
		return nil, err
	}
	reader, ok := b.(OutlineBackend)
	if !ok {
		return nil, readBackendMismatch(b, OperationOutline, "OutlineBackend")
	}
	return reader.Outline(ctx, request)
}

// FindReferences dispatches typed reference search after capability and workspace-trust checks.
func (s *Service) FindReferences(ctx context.Context, request ReferencesRequest) (*ReferencesResult, error) {
	b, err := s.backendFor(request.Project, OperationFindReferences)
	if err != nil {
		return nil, err
	}
	reader, ok := b.(ReferencesBackend)
	if !ok {
		return nil, readBackendMismatch(b, OperationFindReferences, "ReferencesBackend")
	}
	return reader.FindReferences(ctx, request)
}

// GetProjectContext returns the project selected for inspection.
func (r InspectRequest) GetProjectContext() ProjectContext { return r.Project }

// SetProjectContext updates the inspection project after ingress context is merged.
func (r *InspectRequest) SetProjectContext(project ProjectContext) { r.Project = project }

// GetProjectContext returns the project selected for outlining.
func (r OutlineRequest) GetProjectContext() ProjectContext { return r.Project }

// SetProjectContext updates the outline project after ingress context is merged.
func (r *OutlineRequest) SetProjectContext(project ProjectContext) { r.Project = project }

// GetProjectContext returns the project selected for reference search.
func (r ReferencesRequest) GetProjectContext() ProjectContext { return r.Project }

// SetProjectContext updates the reference-search project after ingress context is merged.
func (r *ReferencesRequest) SetProjectContext(project ProjectContext) { r.Project = project }
