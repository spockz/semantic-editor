// Package gobackend validates Go rename targets with the same resolver and gopls preflight as execution.
package gobackend

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/adapters/golang"
	backend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/symbol"
)

// ValidateRenameTarget proves that the selected Go symbol has a complete
// resolver preimage and that gopls accepts it for the requested rename. It
// never sends textDocument/rename or writes source files.
func (GoBackend) ValidateRenameTarget(ctx context.Context, request backend.RenameRequest) error {
	project := request.Project
	snapshot, lookup, err := resolveRenameTarget(ctx, project, backend.NormalizeRenameInput(request.Symbol))
	if err != nil {
		return err
	}
	return golang.ValidateRenameSelected(ctx, project.RootDir, *lookup.Definition, snapshot.Files, backend.NormalizeRenameInput(request.To))
}

func resolveRenameTarget(ctx context.Context, project backend.ProjectContext, selector string) (*symbol.ResolutionSnapshot, *backend.LookupResult, error) {
	snapshot, err := symbol.ResolveSnapshotWithContext(ctx, project.RootDir, project.File, selector)
	if err != nil {
		return nil, nil, err
	}
	lookup, err := convertLookup(project.RootDir, snapshot)
	if err != nil {
		return nil, nil, fmt.Errorf("capture Go rename lookup: %w", err)
	}
	if lookup.IsAmbiguous() {
		return nil, nil, renameAmbiguity(project.RootDir, selector, lookup.Candidates, goSourceView(ctx))
	}
	if lookup.Definition == nil {
		return nil, nil, fmt.Errorf("rename lookup lacks a declaration identity")
	}
	sourcePath, err := pathutil.FilePathFromURI(lookup.Definition.Location.URI)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve rename lookup location: %w", err)
	}
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(project.RootDir, sourcePath)
	}
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve rename lookup preimage: %w", err)
	}
	if _, ok := snapshot.Files[filepath.Clean(sourcePath)]; !ok {
		return nil, nil, errors.New("rename lookup lacks the source preimage")
	}
	return snapshot, lookup, nil
}
