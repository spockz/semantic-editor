// Package backend exposes optional, read-only native symbol candidates without expanding outline capabilities.
package backend

import (
	"context"
	"fmt"
)

// SymbolCandidateBackend enumerates native candidates for a selected file.
type SymbolCandidateBackend interface {
	SymbolCandidates(context.Context, ProjectContext) ([]ReadSymbol, error)
}

// SymbolCandidates retrieves candidates from a backend that exposes native enumeration.
func (s *Service) SymbolCandidates(ctx context.Context, project ProjectContext) ([]ReadSymbol, error) {
	b, err := s.backendFor(project, OperationLookup)
	if err != nil {
		return nil, err
	}
	provider, ok := b.(SymbolCandidateBackend)
	if !ok {
		return nil, fmt.Errorf("backend %s does not expose native symbol candidates", b.Language())
	}
	return provider.SymbolCandidates(ctx, project)
}
