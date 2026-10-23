// Package gobackend adapts the established Go resolver and diagnostics behind the neutral contract.
package gobackend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/adapters/golang"
	"github.com/spockz/semantic-editor/internal/astedit"
	backend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
	"github.com/spockz/semantic-editor/internal/telemetry"
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
	case backend.OperationInspectConstruct:
		kinds = astedit.GoInspectConstructKinds
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
	return backend.NewCapabilities(backend.OperationLookup, backend.OperationRename, backend.OperationVerify, backend.OperationInspect, backend.OperationInspectConstruct, backend.OperationOutline, backend.OperationFindReferences, backend.OperationExtractVariable, backend.OperationExtractFunction)
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
		"refactor_extract_callable": {
			Supported:   true,
			Description: ("Extracts a supported contiguous region of direct calls, simple := or = assignments, or one complete typed scalar if/else into a helper. Typed read/write flow derives live-in parameters and live-out results, including pre-existing locals, mixed declarations and assignments, and nameable Go result types while preserving assignment evaluation and first-write result order. Callable range extraction also accepts direct local or parameter map/slice identifiers, two-index slice expressions, and three-index full-slice expressions rooted in local or parameter slices when explicit bounds are pure compiler-resolved integers; it preserves the original range expression and exact base value. In one expression-switch case/default body, it also accepts one direct simple = assignment to a pre-existing owner local or named parameter (never the receiver) when that value is read after the switch; the helper returns the assigned value and the call writes it back at the same clause location. Other case statements, nested cases, type-switch/select clauses, and cross-case ranges refuse. Ordinary comments move with the selected source only when every comment in the owner body lies wholly inside the selected statement range; directives and comments outside that range refuse. Address-taken or closure-captured values, recover, uncertain receiver dispatch, unsupported statements, other control flow, composite literals, and generic or variadic owners also refuse."),
			CLICommand:  "semedit refactor-extract-callable --file <path> --symbol <helper> --in-function <owner> --kind <function|method> --start-line <line> --end-line <line>",
			MCPTool:     "refactor_extract_callable",
		},
		"refactor_extract_value": {
			Supported:   true,
			Description: "kind=variable extracts exact side-effect-free typed arguments from a direct identifier call statement, or the RHS of one simple assignment to a named identifier in a direct or nested statement block, including a short declaration with one target and one value or a one-spec local var declaration with one named target and one initializer, or one whole expression in a direct or nested return with exactly one explicit owner result slot. The result slot may be unnamed or have one nonblank name; the original return still assigns named results before defers run. Multiple result slots, blank or multiply named result fields, naked returns, and untyped-only expressions refuse. A nested local stays immediately before its assignment, declaration, or return in that statement's existing block; function-literal bodies are not candidates. Grouped var declarations and multi-name or multi-value var specs refuse. When multiple assignment, declaration, or return statements match, select one with construct_path and its matching expected_revision. In one selected call, all repeated matches may share one local only when the expression uses stable scalar owner inputs and compile-time constants, every occurrence retains the same resolved type, and its inputs are not address-escaped explicitly or through an implicit pointer-receiver method selection, or closure-captured. Return extraction preserves the expression's inferred type and evaluates immediately before that return. Call siblings must satisfy the compiler-backed evaluation-order whitelist; assignment extraction rejects compound or tuple assignments, multi-name short declarations, and non-identifier targets. kind=constant extracts one exact resolved compile-time basic constant expression into a package constant. kind=field is unsupported. Each supported kind refuses unsupported syntax, collisions, comments, ambiguous selections, and stale selectors; variable extraction also refuses callable shadowing, unresolved imports, nested calls, and unsupported contexts.",
			CLICommand:  "semedit refactor-extract-value --file <path> --symbol <name> --in-function <owner> --discriminator <expression> [--construct-path <path> --expected-revision <sha256>]",
			MCPTool:     "refactor_extract_value",
		},
		"refactor_encapsulate_field": {
			Supported:   true,
			Description: "Generates private pointer-receiver accessors for one private non-generic struct field and rewrites every typed direct read and standalone assignment proved safe across active package and test variants. Refuses inactive or generated package sources, embedding, interface method-set collisions, positional literals, unsupported references, and evaluation-order-sensitive expressions.",
			CLICommand:  "semedit refactor-encapsulate-field --file <path> --symbol <Type.Field> --getter <name> --setter <name> [--destination-file <path>]",
			MCPTool:     "refactor_encapsulate_field",
		},
		"refactor_inline_value": {
			Supported:   true,
			Description: ("Inlines a basic-literal local at proven direct call-argument or return uses later in the same lexical block before control flow. For a pure scalar computed initializer, it also supports the complete expression at its sole immediate direct return, the sole argument of an immediate direct call to a declared function or builtin, the complete RHS of the immediately following one-target simple assignment to a distinct existing variable under ADR-0113, the complete RHS of the immediately following single-name short declaration under ADR-0114, or the complete RHS of the immediately following single-name, single-value local var declaration under ADR-0119. Candidate checks preserve resolved input bindings and types. Also inlines one keyed scalar-only same-package nongeneric named struct literal at its sole immediate direct return after compiler-resolved binding and type checks. Other computed shapes, explicit types, unsupported struct fields or values, extra references, and comments in the moved or removed region refuse. Callable inlining is exposed separately as refactor_inline_callable."),
			CLICommand:  "semedit refactor-inline-value --file <path> --symbol <name> --in-function <owner>",
			MCPTool:     "refactor_inline_value",
		},
	}
}

func goEditCapabilities() map[string]backend.OpCapability {
	return map[string]backend.OpCapability{
		"refactor_rename": {
			Supported:    true,
			Description:  "Compiler-backed symbol renaming across identifiers, struct fields, methods, interfaces, and packages with automatic import tidying.",
			CLICommand:   "semedit refactor-rename --file <path> --symbol <sym> --to <name>",
			MCPTool:      "refactor_rename",
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
func (GoBackend) Lookup(ctx context.Context, project backend.ProjectContext, query string) (*backend.LookupResult, error) {
	snapshot, err := symbol.ResolveSnapshotWithContext(ctx, project.RootDir, project.File, query)
	if err != nil {
		return nil, fmt.Errorf("resolve Go lookup snapshot: %w", err)
	}
	result, err := convertLookup(project.RootDir, snapshot)
	if err != nil {
		return nil, err
	}
	result.SourceView = goSourceView(ctx)
	return result, nil
}

type renameIdentity struct {
	path          string
	qualifiedName string
	kind          string
}

type resolvedRename struct {
	snapshot *symbol.ResolutionSnapshot
	lookup   *backend.LookupResult
	identity renameIdentity
}

// Rename resolves and executes a Go rename, retaining the established CLI/MCP pipeline.
func (GoBackend) Rename(ctx context.Context, request backend.RenameRequest) (result *backend.RenameResult, resultErr error) {
	ctx = gocache.WithEnvironmentCache(ctx)
	ctx, release := godiagnostics.EnsureSessionManager(ctx)
	defer func() { resultErr = errors.Join(resultErr, release()) }()

	resolved, err := resolveRename(ctx, request)
	if err != nil {
		return nil, err
	}
	result, err = executeRename(ctx, request, resolved)
	if !isRetryableRenameConflict(err) {
		return result, err
	}

	fresh, err := resolveRename(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("re-resolve Go rename after source change: %w", err)
	}
	if fresh.identity != resolved.identity {
		return nil, fmt.Errorf("selected Go declaration changed before rename retry: was %s in %s, now %s in %s", resolved.identity.qualifiedName, resolved.identity.path, fresh.identity.qualifiedName, fresh.identity.path)
	}
	result, err = executeRename(ctx, request, fresh)
	if isRetryableRenameConflict(err) {
		return nil, fmt.Errorf("rename refused after its fresh retry also found changed source; no files were published: %w", err)
	}
	return result, err
}

func resolveRename(ctx context.Context, request backend.RenameRequest) (resolvedRename, error) {
	snapshot, lookup, err := resolveRenameTarget(ctx, request.Project, request.Symbol)
	if err != nil {
		return resolvedRename{}, err
	}
	sourcePath, err := pathutil.FilePathFromURI(lookup.Definition.Location.URI)
	if err != nil {
		return resolvedRename{}, fmt.Errorf("resolve rename lookup location: %w", err)
	}
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(request.Project.RootDir, sourcePath)
	}
	canonicalPath, err := sourcebuf.CanonicalPath(ctx, sourcePath)
	if err != nil {
		return resolvedRename{}, fmt.Errorf("canonicalize Go rename declaration path: %w", err)
	}
	qualifiedName := lookup.Definition.QualifiedName
	if qualifiedName == "" {
		qualifiedName = lookup.Definition.Name
	}
	if qualifiedName == "" || lookup.Definition.Kind == "" {
		return resolvedRename{}, errors.New("rename lookup lacks a stable declaration identity")
	}
	return resolvedRename{
		snapshot: snapshot,
		lookup:   lookup,
		identity: renameIdentity{path: filepath.Clean(canonicalPath), qualifiedName: qualifiedName, kind: lookup.Definition.Kind},
	}, nil
}

func hasRenameSourcePreimage(files map[string][]byte, selectedPath string) bool {
	absoluteSelected, err := filepath.Abs(selectedPath)
	if err != nil {
		return false
	}
	if _, ok := files[filepath.Clean(absoluteSelected)]; ok {
		return true
	}
	canonicalSelected, err := filepath.EvalSymlinks(selectedPath)
	if err != nil {
		return false
	}
	for path := range files {
		canonicalPath, err := filepath.EvalSymlinks(path)
		if err == nil && filepath.Clean(canonicalPath) == filepath.Clean(canonicalSelected) {
			return true
		}
	}
	return false
}

func executeRename(ctx context.Context, request backend.RenameRequest, resolved resolvedRename) (*backend.RenameResult, error) {
	var before []godiagnostics.Diagnostic
	if !request.DeferVerification {
		var err error
		before, err = pipeline.CheckDiagnosticDetails(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), request.Project.RootDir)
		if err != nil {
			return nil, err
		}
	}
	changedPaths, err := golang.RenameSelectedFiles(ctx, request.Project.RootDir, *resolved.lookup.Definition, resolved.snapshot.Files, request.To)
	if err != nil {
		var publicationErr *golang.RenamePublicationError
		if errors.As(err, &publicationErr) && len(publicationErr.PublishedPaths) > 0 {
			return &backend.RenameResult{Lookup: resolved.lookup}, err
		}
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
	result := &backend.RenameResult{Lookup: resolved.lookup, Diagnostics: backend.DiagnosticDelta{
		Before: delta.Before, After: delta.After, NetDelta: delta.NetDelta,
		Introduced: delta.Introduced, Resolved: delta.Resolved, Suggestions: delta.Suggestions,
	}, Active: afterMessages}
	if !request.DeferVerification && len(introducedErrors) > 0 {
		return result, &backend.RenameDiagnosticsError{Result: result, IntroducedErrors: introducedErrors}
	}
	return result, nil
}

func isRetryableRenameConflict(err error) bool {
	var publicationErr *golang.RenamePublicationError
	return errors.As(err, &publicationErr) && publicationErr.Retryable()
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
	diagnostics, err := pipeline.CheckDiagnosticDetailsForPatterns(ctx, project.RootDir, request.GoPackagePatterns...)
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
func (GoBackend) Outline(ctx context.Context, request backend.OutlineRequest) (*backend.OutlineResult, error) {
	return astedit.OutlineGo(ctx, request.Project.RootDir, request.Path, request.Kinds, request.IncludeUnexported, request.IncludeTests)
}

type goLookupMetadata struct {
	declarationRange *backend.Range
	signature        *string
}

func convertCandidate(root string, candidate *symbol.Symbol, sources map[string][]byte, metadataByPath map[string]map[int]goLookupMetadata) (*backend.NamedTarget, error) {
	location, err := sourceLocation(root, candidate.File, candidate.Offset, candidate.Name, sources)
	if err != nil {
		return nil, fmt.Errorf("capture Go lookup candidate: %w", err)
	}
	path := candidate.File
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path = filepath.Clean(path)
	source, ok := sources[path]
	if !ok {
		return nil, fmt.Errorf("lookup snapshot lacks source for %s", path)
	}
	metadata, ok := metadataByPath[path]
	if !ok {
		metadata, err = goLookupDeclarationMetadata(path, source)
		if err != nil {
			return nil, fmt.Errorf("capture Go lookup declaration metadata: %w", err)
		}
		metadataByPath[path] = metadata
	}
	value, ok := metadata[candidate.Offset]
	if !ok {
		return &backend.NamedTarget{Name: candidate.Name, QualifiedName: candidate.BuildQualifiedName(), Kind: candidate.Kind, Location: location}, nil
	}
	return &backend.NamedTarget{Name: candidate.Name, QualifiedName: candidate.BuildQualifiedName(), Kind: candidate.Kind, Location: location, DeclarationRange: value.declarationRange, Signature: value.signature}, nil
}

func goLookupDeclarationMetadata(path string, source []byte) (map[int]goLookupMetadata, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse captured source %s: %w", path, err)
	}
	metadata := make(map[int]goLookupMetadata)
	for _, declaration := range file.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			declarationRange, rangeErr := goLookupRange(source, fset, typed.Pos(), typed.End())
			if rangeErr != nil {
				return nil, rangeErr
			}
			signatureDecl := *typed
			signatureDecl.Doc = nil
			signatureDecl.Body = nil
			var rendered bytes.Buffer
			if formatErr := format.Node(&rendered, fset, &signatureDecl); formatErr != nil {
				return nil, fmt.Errorf("format declaration signature: %w", formatErr)
			}
			signature := rendered.String()
			metadata[fset.PositionFor(typed.Name.Pos(), false).Offset] = goLookupMetadata{declarationRange: &declarationRange, signature: &signature}
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				for _, name := range goLookupSpecNames(spec) {
					declarationRange, rangeErr := goLookupRange(source, fset, spec.Pos(), spec.End())
					if rangeErr != nil {
						return nil, rangeErr
					}
					metadata[fset.PositionFor(name.Pos(), false).Offset] = goLookupMetadata{declarationRange: &declarationRange}
				}
			}
		}
	}
	return metadata, nil
}

func goLookupSpecNames(spec ast.Spec) []*ast.Ident {
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		return []*ast.Ident{typed.Name}
	case *ast.ValueSpec:
		return typed.Names
	default:
		return nil
	}
}

func goLookupRange(source []byte, fset *token.FileSet, start, end token.Pos) (backend.Range, error) {
	startOffset := fset.PositionFor(start, false).Offset
	endOffset := fset.PositionFor(end, false).Offset
	if startOffset < 0 || endOffset < startOffset || endOffset > len(source) {
		return backend.Range{}, fmt.Errorf("invalid declaration byte span [%d,%d) for %d bytes", startOffset, endOffset, len(source))
	}
	return backend.Range{Start: backend.PositionFromByteOffset(source, startOffset), End: backend.PositionFromByteOffset(source, endOffset)}, nil
}

func sourceLocation(root, file string, offset int, name string, sources map[string][]byte) (backend.SourceLocation, error) {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path = filepath.Clean(path)
	source, ok := sources[path]
	if !ok {
		return backend.SourceLocation{}, fmt.Errorf("lookup snapshot lacks source for %s", path)
	}
	if offset < 0 || offset > len(source) || name == "" || !strings.HasPrefix(string(source[offset:]), name) {
		return backend.SourceLocation{}, fmt.Errorf("lookup snapshot has invalid identifier offset for %s in %s", name, path)
	}
	return backend.SourceLocation{
		URI:   pathutil.FileURI(path),
		Range: backend.Range{Start: backend.PositionFromByteOffset(source, offset), End: backend.PositionFromByteOffset(source, offset+len(name))},
	}, nil
}

func convertLookup(root string, snapshot *symbol.ResolutionSnapshot) (*backend.LookupResult, error) {
	result := snapshot.Result
	converted := &backend.LookupResult{}
	metadataByPath := make(map[string]map[int]goLookupMetadata)
	var err error
	if result.Definition != nil {
		converted.Definition, err = convertCandidate(root, result.Definition, snapshot.Files, metadataByPath)
		if err != nil {
			return nil, err
		}
	}
	for _, usage := range result.Usages {
		candidate, convertErr := convertCandidate(root, usage, snapshot.Files, metadataByPath)
		if convertErr != nil {
			return nil, convertErr
		}
		converted.Usages = append(converted.Usages, candidate)
	}
	for _, choice := range result.Candidates {
		candidate, convertErr := convertCandidate(root, choice, snapshot.Files, metadataByPath)
		if convertErr != nil {
			return nil, convertErr
		}
		converted.Candidates = append(converted.Candidates, candidate)
	}
	return converted, nil
}

func goSourceView(ctx context.Context) string {
	if sourcebuf.FromContext(ctx) != nil {
		return "working"
	}
	return "disk"
}

func renameAmbiguity(root, query string, candidates []*backend.NamedTarget, sourceView string) error {
	locations, err := backend.RenderAmbiguousCandidates(root, candidates)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: symbol %q in %s source matches %d candidates:\n%s\nSpecify 'file' or a receiver qualification (e.g. 'TypeName.%s') to disambiguate directly", backend.ErrAmbiguous, query, sourceView, len(candidates), locations, query)
}
