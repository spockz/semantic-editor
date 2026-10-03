// Package gobackend adapts the established Go resolver and diagnostics behind the neutral contract.
package gobackend

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"semedit/internal/adapters/golang"
	"semedit/internal/astedit"
	backend "semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/godiagnostics"
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

// SupportedConstructs reports executable Go construct kinds for one operation.
func (GoBackend) SupportedConstructs(operation backend.Operation) []backend.ConstructKind {
	var kinds []backend.ConstructKind
	switch operation {
	case backend.OperationInsertConstruct:
		kinds = astedit.GoInsertConstructKinds
	case backend.OperationReplaceConstruct:
		kinds = astedit.GoReplaceConstructKinds
	case backend.OperationRemoveConstruct:
		kinds = astedit.GoRemoveConstructKinds
	case backend.OperationMoveConstruct:
		kinds = astedit.GoMoveConstructKinds
	default:
		return nil
	}
	return append([]backend.ConstructKind(nil), kinds...)
}

// Capabilities returns operations supported by the Go adapter.
func (GoBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilities(backend.OperationLookup, backend.OperationRename, backend.OperationVerify, backend.OperationInspect, backend.OperationOutline, backend.OperationFindReferences)
}

// CapabilityMatrix returns the declarative documentation matrix for the Go backend.
func (GoBackend) CapabilityMatrix() backend.LanguageMatrix {
	return backend.LanguageMatrix{
		Language:           "go",
		DisplayName:        "Go (Golang)",
		Maturity:           "Production",
		SupportedModifiers: goSupportedModifierNames(),
		Operations:         goCapabilityOperations(),
		Limitations:        goCapabilityLimitations(),
	}
}

func goSupportedModifierNames() []string {
	raw := astedit.GolangBackend{}.SupportedAccessModifiers()
	mods := make([]string, len(raw))
	for i, modifier := range raw {
		mods[i] = string(modifier)
	}
	return mods
}

func goCapabilityOperations() map[string]backend.OpCapability {
	operations := make(map[string]backend.OpCapability)
	maps.Copy(operations, goExtractionCapabilities())
	maps.Copy(operations, goEditCapabilities())
	maps.Copy(operations, goReadCapabilities())
	return operations
}

func goExtractionCapabilities() map[string]backend.OpCapability {
	return map[string]backend.OpCapability{
		"extract_function": {
			Supported:   true,
			Description: "Extracts contiguous direct calls, optionally with one leading short declaration whose distinct live-out values have validated supported result types; rejects effectful or unsupported results, address-taken values, closures, recover, uncertain receiver dispatch, other local flow, control flow, composite literals, generic or variadic owners, and comments.",
			CLICommand:  "semedit extract-function --file <path> --symbol <helper> --in-function <owner> --kind <function|method> --start-line <line> --end-line <line>",
			MCPTool:     "semantic_extract_function",
		},
		"extract_constant": {
			Supported:   true,
			Description: "Extracts one exact resolved compile-time basic constant expression from a Go function to a package constant; refuses iota, local constant dependencies, unsupported types or syntax, comments, collisions, duplicate matches, and stale selected-file revisions.",
			CLICommand:  "semedit extract-constant --file <path> --symbol <name> --in-function <owner> --discriminator <expression> [--expected-revision <sha256>]",
			MCPTool:     "semantic_extract_constant",
		},
		"extract_variable": {
			Supported:   true,
			Description: "Extracts one exact side-effect-free typed first argument from a direct identifier call statement into a preceding local while preserving later argument order; a supplied construct_path requires the matching selected-file expected_revision. It refuses callable shadowing, unresolved imports, comments, constant-only or unsupported types, nested calls, and unsupported contexts.",
			CLICommand:  "semedit extract-variable --file <path> --symbol <name> --in-function <owner> --discriminator <expression> [--construct-path <path> --expected-revision <sha256>]",
			MCPTool:     "semantic_extract_variable",
		},
		"inline_variable": {
			Supported:   true,
			Description: "Inlines one local literal into all proven direct uses in the immediately following statement; refuses computed or explicitly typed initializers, intervening statements, shadowing, extra references, unsupported use shapes, and comments. Call inlining is unsupported.",
			CLICommand:  "semedit inline-variable --file <path> --symbol <name> --in-function <owner>",
			MCPTool:     "semantic_inline_variable",
		},
	}
}

func goEditCapabilities() map[string]backend.OpCapability {
	return map[string]backend.OpCapability{
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
	}
}

func goReadCapabilities() map[string]backend.OpCapability {
	return map[string]backend.OpCapability{
		"lookup": {
			Supported:    true,
			Description:  "Fast symbol coordinate, byte offset, receiver, and AST range lookup without line counting.",
			CLICommand:   "semedit lookup --file <path> --symbol <sym>",
			MCPTool:      "resolve_symbol_location",
			PlacementKey: false,
		},
		"inspect_symbol": {
			Supported:    true,
			Description:  "Resolve a Go declaration and return its exact source snapshot, revision, range, signature, docs, imports, and type metadata.",
			CLICommand:   "semedit inspect-symbol --symbol <name> [--file <path>]",
			MCPTool:      "semantic_inspect_symbol",
			PlacementKey: false,
			Level:        "symbol",
			ReadOnly:     true,
		},
		"outline": {
			Supported:    true,
			Description:  "Return a file-centric Go declaration outline with optional kind, visibility, and test-file filters.",
			CLICommand:   "semedit outline --path <file-or-directory>",
			MCPTool:      "semantic_outline",
			PlacementKey: false,
			Level:        "symbol",
			ReadOnly:     true,
		},
		"verify": {
			Supported:    true,
			Description:  "Format source files and report compiler diagnostics for the project.",
			CLICommand:   "semedit verify --file <path>",
			MCPTool:      "semantic_verify",
			PlacementKey: false,
		},
		"find_references": {
			Supported:    true,
			Description:  "Find typed references in the selected Go module using active build configuration, including test variants. Cgo and transformed inputs are unsupported.",
			CLICommand:   "semedit find-references --symbol <sym> [--file <path>]",
			MCPTool:      "semantic_find_references",
			PlacementKey: false,
			Level:        "symbol",
			ReadOnly:     true,
		},
	}
}

func goCapabilityLimitations() []backend.Constraint {
	return []backend.Constraint{
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
	}
}

// Lookup resolves a Go symbol and converts its location to the neutral contract.
func (GoBackend) Lookup(_ context.Context, project backend.ProjectContext, query string) (*backend.LookupResult, error) {
	result, err := symbol.Resolve(project.RootDir, project.File, query)
	if err != nil {
		return nil, err
	}
	return convertLookup(project.RootDir, result), nil
}

// Rename resolves and executes a Go rename, retaining the established CLI/MCP pipeline.
func (GoBackend) Rename(ctx context.Context, request backend.RenameRequest) (*backend.RenameResult, error) {
	snapshot, err := symbol.ResolveSnapshotWithContext(ctx, request.Project.RootDir, request.Project.File, request.Symbol)
	if err != nil {
		return nil, err
	}
	lookup := convertLookup(request.Project.RootDir, snapshot.Result)
	if lookup.Ambiguous {
		return nil, &backend.Error{Operation: backend.OperationRename, Err: backend.ErrAmbiguous}
	}
	if lookup.Definition == nil {
		return nil, fmt.Errorf("rename lookup lacks a declaration identity")
	}
	sourcePath := lookup.Definition.File
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(request.Project.RootDir, sourcePath)
	}
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve rename lookup preimage: %w", err)
	}
	_, ok := snapshot.Files[filepath.Clean(sourcePath)]
	if !ok {
		return nil, errors.New("rename lookup lacks the source preimage")
	}
	var before []godiagnostics.Diagnostic
	if !request.DeferVerification {
		before, err = pipeline.CheckDiagnosticDetails(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), request.Project.RootDir)
		if err != nil {
			return nil, err
		}
	}
	changedPaths, err := golang.RenameSelectedFiles(ctx, request.Project.RootDir, *lookup.Definition, snapshot.Files, request.To)
	if err != nil {
		return nil, err
	}
	if !request.DeferFormatting && len(changedPaths) > 0 {
		if request.OrganizeImports {
			if organizeErr := pipeline.OrganizeImports(ctx, request.Project.RootDir, changedPaths...); organizeErr != nil {
				return nil, fmt.Errorf("organize imports after rename: %w", organizeErr)
			}
		} else {
			if formatErr := pipeline.Format(ctx, request.Project.RootDir, changedPaths...); formatErr != nil {
				return nil, fmt.Errorf("format after rename: %w", formatErr)
			}
		}
	}
	var after []godiagnostics.Diagnostic
	if !request.DeferVerification {
		after, err = pipeline.CheckDiagnosticDetails(telemetry.WithPhase(ctx, telemetry.PhaseVerificationAfter), request.Project.RootDir)
		if err != nil {
			return nil, err
		}
	}
	beforeMessages := godiagnostics.Messages(before)
	afterMessages := godiagnostics.Messages(after)
	delta := pipeline.ComputeDelta(beforeMessages, afterMessages)
	introducedErrors := introducedDiagnosticErrors(before, after)
	result := &backend.RenameResult{Lookup: lookup, Diagnostics: backend.DiagnosticDelta{
		Before: delta.Before, After: delta.After, NetDelta: delta.NetDelta,
		Introduced: delta.Introduced, Resolved: delta.Resolved, Suggestions: delta.Suggestions,
	}, Active: afterMessages}
	if !request.DeferVerification && len(introducedErrors) > 0 {
		return result, &backend.RenameDiagnosticsError{Result: result, IntroducedErrors: introducedErrors}
	}
	return result, nil
}

// Verify delegates formatting and diagnostics to the existing Go pipeline.
func (GoBackend) Verify(ctx context.Context, request backend.VerifyRequest) ([]backend.Diagnostic, error) {
	project, path := request.Project, request.Path
	if path == "" {
		path = "."
	}
	if !request.CheckOnly {
		if err := pipeline.Format(ctx, project.RootDir, path); err != nil {
			return nil, err
		}
	}
	diagnostics, err := pipeline.CheckDiagnosticDetails(ctx, project.RootDir)
	if err != nil {
		return nil, err
	}
	result := make([]backend.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		var location *backend.SourceLocation
		if diagnostic.Source != "package" {
			file := diagnostic.Path
			if !filepath.IsAbs(file) {
				file = filepath.Join(project.RootDir, filepath.FromSlash(file))
			}
			position := backend.Position{}
			if diagnostic.Line > 0 {
				position.Line = diagnostic.Line - 1
			}
			if diagnostic.Column > 0 {
				position.Character = diagnostic.Column - 1
			}
			location = &backend.SourceLocation{URI: pathutil.FileURI(file), Range: backend.Range{Start: position, End: position}}
		}
		result = append(result, backend.Diagnostic{Message: diagnostic.String(), Severity: diagnostic.Severity, Location: location})
	}
	return result, nil
}

func introducedDiagnosticErrors(before, after []godiagnostics.Diagnostic) []string {
	counts := make(map[string]int)
	for _, diagnostic := range before {
		if diagnostic.Severity == 1 {
			counts[diagnosticIdentity(diagnostic)]++
		}
	}
	introduced := make([]string, 0)
	for _, diagnostic := range after {
		if diagnostic.Severity != 1 {
			continue
		}
		identity := diagnosticIdentity(diagnostic)
		if counts[identity] > 0 {
			counts[identity]--
			continue
		}
		introduced = append(introduced, diagnostic.String())
	}
	return introduced
}

func diagnosticIdentity(diagnostic godiagnostics.Diagnostic) string {
	return diagnostic.Path + "\x00" + diagnostic.Source + "\x00" + diagnostic.Message
}

// Inspect resolves a Go declaration and returns its exact source snapshot.
func (GoBackend) Inspect(ctx context.Context, request backend.InspectRequest) (*backend.InspectResult, error) {
	return astedit.InspectGoWithContext(ctx, request.Project.RootDir, request.Project.File, request.Symbol)
}

// Outline returns a file-centric declaration projection for a Go source selection.
func (GoBackend) Outline(_ context.Context, request backend.OutlineRequest) (*backend.OutlineResult, error) {
	return astedit.OutlineGo(request.Project.RootDir, request.Path, request.Kinds, request.IncludeUnexported, request.IncludeTests)
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

func convertLookup(root string, result *symbol.LookupResult) *backend.LookupResult {
	converted := &backend.LookupResult{
		Symbol: result.Symbol, File: result.File, Line: result.Line, Column: result.Column,
		Offset: result.Offset, Kind: result.Kind, Receiver: result.Receiver,
		Ambiguous: result.Ambiguous,
	}
	if result.Definition != nil {
		converted.Definition = convertCandidate(root, result.Definition)
		converted.Location = converted.Definition.Location
	}
	if result.Usages != nil {
		converted.Usages = make([]*backend.SymbolCandidate, 0, len(result.Usages))
		for _, usage := range result.Usages {
			converted.Usages = append(converted.Usages, convertCandidate(root, usage))
		}
	}
	if result.Candidates != nil {
		converted.Candidates = make([]*backend.SymbolCandidate, 0, len(result.Candidates))
		for _, candidate := range result.Candidates {
			converted.Candidates = append(converted.Candidates, convertCandidate(root, candidate))
		}
	}
	return converted
}
