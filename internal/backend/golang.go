package backend

import (
	"context"
	"os"
	"path/filepath"

	"semedit/internal/adapters/golang"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// GoBackend adapts the existing Go resolver, gopls rename, and diagnostic pipeline.
// The adapter deliberately preserves the established Go implementation in this slice.
type GoBackend struct{}

// NewGoBackend constructs the adapter for the existing Go implementation.
func NewGoBackend() *GoBackend { return &GoBackend{} }

// Language returns the adapter language ID.
func (GoBackend) Language() LanguageID { return LanguageGo }

// Capabilities returns operations supported by the Go adapter.
func (GoBackend) Capabilities() Capabilities {
	return NewCapabilities(OperationLookup, OperationRename, OperationVerify)
}

// Lookup resolves a Go symbol and converts its location to the neutral contract.
func (GoBackend) Lookup(_ context.Context, project ProjectContext, query string) (*LookupResult, error) {
	result, err := symbol.Resolve(project.RootDir, project.File, query)
	if err != nil {
		return nil, err
	}
	converted := &LookupResult{
		Symbol:    result.Symbol,
		File:      result.File,
		Line:      result.Line,
		Column:    result.Column,
		Offset:    result.Offset,
		Kind:      result.Kind,
		Receiver:  result.Receiver,
		Ambiguous: result.Ambiguous,
	}
	if result.Ambiguous {
		converted.Candidates = make([]*SymbolCandidate, 0, len(result.Candidates))
		for _, candidate := range result.Candidates {
			converted.Candidates = append(converted.Candidates, convertCandidate(project.RootDir, candidate))
		}
		return converted, nil
	}
	converted.Location = sourceLocation(project.RootDir, result.File, result.Offset)
	return converted, nil
}

// Rename delegates execution to the existing gopls adapter.
func (GoBackend) Rename(ctx context.Context, project ProjectContext, lookup *LookupResult, newName string) error {
	return golang.Rename(ctx, project.RootDir, lookup.File, lookup.Line, lookup.Column, newName)
}

// Verify delegates formatting and diagnostics to the existing Go pipeline.
func (GoBackend) Verify(ctx context.Context, project ProjectContext, path string) ([]Diagnostic, error) {
	if path == "" {
		path = "."
	}
	if err := pipeline.Format(ctx, project.RootDir, path); err != nil {
		return nil, err
	}
	diagnostics, err := pipeline.CheckDiagnostics(ctx, project.RootDir)
	if err != nil {
		return nil, err
	}
	result := make([]Diagnostic, 0, len(diagnostics))
	for _, message := range diagnostics {
		result = append(result, Diagnostic{Message: message, Severity: 1})
	}
	return result, nil
}

func convertCandidate(root string, candidate *symbol.Symbol) *SymbolCandidate {
	converted := &SymbolCandidate{
		Name:          candidate.Name,
		Receiver:      candidate.Receiver,
		QualifiedName: candidate.QualifiedName,
		Kind:          candidate.Kind,
		File:          candidate.File,
		Line:          candidate.Line,
		Column:        candidate.Column,
		Offset:        candidate.Offset,
	}
	converted.Location = sourceLocation(root, candidate.File, candidate.Offset)
	return converted
}

func sourceLocation(root, file string, offset int) SourceLocation {
	path := file
	if !filepath.IsAbs(path) && root != "" {
		path = filepath.Join(root, path)
	}
	// #nosec G304 -- path is derived from the selected project and resolved symbol.
	source, err := os.ReadFile(path)
	if err != nil {
		return SourceLocation{URI: fileURI(path)}
	}
	position := PositionFromByteOffset(source, offset)
	return SourceLocation{
		URI:   fileURI(path),
		Range: Range{Start: position, End: position},
	}
}
