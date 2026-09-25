// Package gobackend adapts the established Go resolver and diagnostics behind the neutral contract.
package gobackend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	backend "semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
	"semedit/internal/telemetry"
)

// GoBackend adapts the existing Go resolver, gopls rename, and diagnostic pipeline.
// The adapter deliberately preserves the established Go implementation in this slice.
type GoBackend struct{}

// NewGoBackend constructs the adapter for the existing Go implementation.
func NewGoBackend() *GoBackend { return &GoBackend{} }

// Language returns the adapter language ID.
func (GoBackend) Language() backend.LanguageID { return backend.LanguageGo }

// SupportedConstructs reports the Go construct handlers actually implemented by this adapter.
func (GoBackend) SupportedConstructs() []backend.ConstructKind {
	return append([]backend.ConstructKind(nil), astedit.GoConstructKinds...)
}

// SupportedStructures reports the Go structural insertion kinds implemented by this adapter.
func (GoBackend) SupportedStructures() []backend.StructureKind {
	return append([]backend.StructureKind(nil), astedit.GoStructureKinds...)
}

// Capabilities returns operations supported by the Go adapter.
func (GoBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilities(backend.OperationLookup, backend.OperationRename, backend.OperationVerify)
}

// CapabilityMatrix returns the declarative documentation matrix for the Go backend.
func (GoBackend) CapabilityMatrix() backend.LanguageMatrix {
	// Derive supported access modifiers from the astedit backend to keep them in sync.
	raw := astedit.GolangBackend{}.SupportedAccessModifiers()
	mods := make([]string, len(raw))
	for i, m := range raw {
		mods[i] = string(m)
	}

	return backend.LanguageMatrix{
		Language:           "go",
		DisplayName:        "Go (Golang)",
		Maturity:           "Production",
		SupportedModifiers: mods,
		Operations: map[string]backend.OpCapability{
			"rename": {
				Supported:    true,
				Description:  "Compiler-backed symbol renaming across identifiers, struct fields, methods, interfaces, and packages with automatic import tidying.",
				CLICommand:   "semedit rename --file <path> --symbol <sym> --to <name>",
				MCPTool:      "semantic_rename",
				PlacementKey: false,
			},
			"insert_func": {
				Supported:    true,
				Description:  "Function and method AST insertion with receiver clustering and public-precedes-private section partitioning.",
				CLICommand:   "semedit insert-func --file <path> --source <code snippet>",
				MCPTool:      "semantic_insert_function",
				PlacementKey: true,
			},
			"insert_type": {
				Supported:    true,
				Description:  "Struct, interface, and type alias AST insertion anchored in public/private type sections.",
				CLICommand:   "semedit insert-type --file <path> --source <type snippet>",
				MCPTool:      "semantic_insert_type",
				PlacementKey: true,
			},
			"insert_decl": {
				Supported:    true,
				Description:  "Constant and variable declaration insertion with automatic merging into existing const/var blocks.",
				CLICommand:   "semedit insert-decl --file <path> --source <decl snippet>",
				MCPTool:      "semantic_insert_decl",
				PlacementKey: true,
			},
			"imports": {
				Supported:    true,
				Description:  "Deterministic import management: resolve missing packages, remove unused imports, and add aliased imports.",
				CLICommand:   "semedit imports --file <path> [--add <pkg>] [--remove <pkg>]",
				MCPTool:      "semantic_organize_imports",
				PlacementKey: false,
			},
			"lookup": {
				Supported:    true,
				Description:  "Fast symbol coordinate, byte offset, receiver, and AST range lookup without line counting.",
				CLICommand:   "semedit lookup --file <path> --symbol <sym>",
				MCPTool:      "resolve_symbol_location",
				PlacementKey: false,
			},
			"verify": {
				Supported:    true,
				Description:  "Format source files and report compiler diagnostics for the project.",
				CLICommand:   "semedit verify --file <path>",
				MCPTool:      "semantic_verify",
				PlacementKey: false,
			},
		},
		Limitations: []backend.Constraint{
			{
				Title:       "Unsupported Modifiers Rejection",
				Description: "Go lacks 'protected' and 'package-private' scopes. The engine rejects these modifiers with ErrUnsupportedModifier.",
				Severity:    "error",
			},
			{
				Title:       "Casing & Visibility Invariant",
				Description: "Identifier capitalization governs visibility. Specifying 'public' for a lowercase symbol or 'private' for an uppercase symbol returns VisibilityMismatchError.",
				Severity:    "error",
			},
			{
				Title:       "Strict Public-Precedes-Private Ordering",
				Description: "All public declarations precede private declarations within generated or updated source files.",
				Severity:    "info",
			},
			{
				Title:       "Receiver Method Clustering",
				Description: "Methods sharing a common receiver type cluster near each other while maintaining public vs private partitioning.",
				Severity:    "info",
			},
			{
				Title:       "Declaration Block Merging",
				Description: "Constants and variables automatically merge into existing 'const (...)' or 'var (...)' blocks instead of creating duplicate blocks.",
				Severity:    "info",
			},
		},
	}
}

// Lookup resolves a Go symbol and converts its location to the neutral contract.
func (GoBackend) Lookup(_ context.Context, project backend.ProjectContext, query string) (*backend.LookupResult, error) {
	result, err := symbol.Resolve(project.RootDir, project.File, query)
	if err != nil {
		return nil, err
	}
	converted := &backend.LookupResult{
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
		converted.Candidates = make([]*backend.SymbolCandidate, 0, len(result.Candidates))
		for _, candidate := range result.Candidates {
			converted.Candidates = append(converted.Candidates, convertCandidate(project.RootDir, candidate))
		}
		return converted, nil
	}
	converted.Location = sourceLocation(project.RootDir, result.File, result.Offset)
	return converted, nil
}

// Rename resolves and executes a Go rename, retaining the established CLI/MCP pipeline.
func (GoBackend) Rename(ctx context.Context, request backend.RenameRequest) (*backend.RenameResult, error) {
	lookup, err := (GoBackend{}).Lookup(ctx, request.Project, request.Symbol)
	if err != nil {
		return nil, err
	}
	if lookup.Ambiguous {
		return nil, &backend.Error{Operation: backend.OperationRename, Err: backend.ErrAmbiguous}
	}
	var before []string
	if !request.DeferVerification {
		before, err = pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), request.Project.RootDir)
		if err != nil {
			return nil, err
		}
	}
	if err := golang.Rename(ctx, request.Project.RootDir, lookup.File, lookup.Line, lookup.Column, request.To); err != nil {
		return nil, err
	}
	if !request.DeferFormatting {
		if request.OrganizeImports {
			if err := pipeline.OrganizeImports(ctx, request.Project.RootDir, "."); err != nil {
				return nil, fmt.Errorf("organize imports after rename: %w", err)
			}
		} else {
			if err := pipeline.Format(ctx, request.Project.RootDir, "."); err != nil {
				return nil, fmt.Errorf("format after rename: %w", err)
			}
		}
	}
	var after []string
	if !request.DeferVerification {
		after, err = pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationAfter), request.Project.RootDir)
		if err != nil {
			return nil, err
		}
	}
	delta := pipeline.ComputeDelta(before, after)
	result := &backend.RenameResult{Lookup: lookup, Diagnostics: backend.DiagnosticDelta{
		Before: delta.Before, After: delta.After, NetDelta: delta.NetDelta,
		Introduced: delta.Introduced, Resolved: delta.Resolved, Suggestions: delta.Suggestions,
	}, Active: after}
	if !request.DeferVerification && len(result.Diagnostics.Introduced) > 0 {
		return result, &backend.RenameDiagnosticsError{Result: result}
	}
	return result, nil
}

// Verify delegates formatting and diagnostics to the existing Go pipeline.
func (GoBackend) Verify(ctx context.Context, request backend.VerifyRequest) ([]backend.Diagnostic, error) {
	project, path := request.Project, request.Path
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
	result := make([]backend.Diagnostic, 0, len(diagnostics))
	for _, message := range diagnostics {
		result = append(result, backend.Diagnostic{Message: message, Severity: 1})
	}
	return result, nil
}

func convertCandidate(root string, candidate *symbol.Symbol) *backend.SymbolCandidate {
	converted := &backend.SymbolCandidate{
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

func sourceLocation(root, file string, offset int) backend.SourceLocation {
	path := file
	if !filepath.IsAbs(path) && root != "" {
		path = filepath.Join(root, path)
	}
	// #nosec G304 -- path is derived from the selected project and resolved symbol.
	source, err := os.ReadFile(path)
	if err != nil {
		return backend.SourceLocation{URI: pathutil.FileURI(path)}
	}
	position := backend.PositionFromByteOffset(source, offset)
	return backend.SourceLocation{
		URI:   pathutil.FileURI(path),
		Range: backend.Range{Start: position, End: position},
	}
}
