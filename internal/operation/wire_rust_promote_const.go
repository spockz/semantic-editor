// Package operation registers Rust promote-local-to-const as an independent user intent.
// WHY: Promotion changes a local binding into a constant and must keep its distinct proof and native assist contract.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/readlsp"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// PromoteLocalToConstReq carries the target and revision for local promotion.
type PromoteLocalToConstReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	ExpectedRevision string
}

// GetProjectContext returns the request's project context.
func (r PromoteLocalToConstReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext updates the request's project context.
func (r *PromoteLocalToConstReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var promoteLocalToConstParams = readRequestParams(
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Target: selected Rust source file", Required: true, Role: RoleTarget, SourceFields: []string{sourceInspectFile}},
	ParameterContract{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: exact immutable local binding to promote", Required: true, Role: RoleTarget, SourceFields: []string{sourceInspectSymbol}},
	ParameterContract{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Description: "Selector: exact top-level Rust function containing the binding", Required: true, Role: RoleSelector, SourceFields: []string{sourceInspectInFunction}},
	ParameterContract{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Description: "Guard: full selected-file SHA-256 revision copied from a matching source inspection", Role: RoleTarget, SourceFields: []string{sourceInspectExpectedRevision}},
)

func parsePromoteLocalToConst(raw map[string]any) (PromoteLocalToConstReq, error) {
	var req PromoteLocalToConstReq
	if err := CheckParams(raw, promoteLocalToConstParams); err != nil {
		return req, err
	}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return req, err
	}
	if req.ExpectedRevision, err = ParseString(raw, wireExpectedRevision, wireCLIExpectedRevision, false); err != nil {
		return req, err
	}
	if value, present := rawLookup(raw, wireExpectedRevision, wireCLIExpectedRevision); present && value != nil && req.ExpectedRevision == "" {
		return req, fmt.Errorf("expected_revision must not be empty when supplied: %w", ErrInvalidParams)
	}
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		req.Project.Language = backend.LanguageRust
	}
	if req.Project.Language != backend.LanguageRust {
		return req, fmt.Errorf("promote-local-to-const is supported only for Rust: %w", ErrInvalidParams)
	}
	return req, nil
}

func runPromoteLocalToConstRust(ctx context.Context, cc CallContext, req PromoteLocalToConstReq) (out ExtractionResult, handlerErr error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	ctx, ownedBuffer, err := rustOperationContext(ctx, project.RootDir)
	if err != nil {
		return ExtractionResult{}, err
	}
	if ownedBuffer != nil {
		defer func() {
			out, handlerErr = finishRustOperation(ctx, ownedBuffer, out, handlerErr, "promote-local-to-const")
		}()
	}
	owner, err := service.Lookup(ctx, project, req.InFunction)
	if err != nil {
		return ExtractionResult{}, err
	}
	if owner == nil || owner.Definition == nil || owner.Definition.DeclarationRange == nil || owner.Definition.Kind != "function" || strings.Contains(owner.Definition.QualifiedName, "::") {
		return ExtractionResult{}, fmt.Errorf("rust promote-local-to-const owner %q must resolve to one top-level function", req.InFunction)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Rust promote-local-to-const preimage: %w", err)
	}
	rangeValue, err := resolveRustPromoteBindingRange(source, *owner.Definition.DeclarationRange, req.Symbol)
	if err != nil {
		return ExtractionResult{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationPromoteLocalToConst, Project: project, File: file,
		Range: rangeValue, Kind: "promote_local_to_const", Symbol: req.Symbol,
		EnclosingSymbol: owner.Definition.QualifiedName, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || !filepath.IsAbs(result.File) || !bytes.Equal(result.Before, source) {
		return ExtractionResult{}, fmt.Errorf("rust promote-local-to-const backend returned an unexpected target or source preimage")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	relative, revision, err := rustExtractionContinuation(cc.WorkDir, req.File, file, result.After)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft := EditReceiptDraft{Operation: string(backend.OperationPromoteLocalToConst), Status: EditPersisted, ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{{File: relative, Symbol: owner.Definition.QualifiedName, ExpectedRevision: &revision}}}
	edit := FileEditRes{File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol, InFunction: owner.Definition.QualifiedName,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Promoted one bounded Rust local to a constant through a verified rust-analyzer assist.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: !bytes.Equal(result.Before, result.After)}
	return ExtractionResult{Edit: edit, Draft: draft}, nil
}

func resolveRustPromoteBindingRange(source []byte, owner backend.Range, name string) (backend.Range, error) {
	start, err := readlsp.ByteOffset(source, owner.Start)
	if err != nil {
		return backend.Range{}, fmt.Errorf("resolve Rust promote owner start: %w", err)
	}
	end, err := readlsp.ByteOffset(source, owner.End)
	if err != nil || end <= start {
		return backend.Range{}, fmt.Errorf("invalid Rust promote owner range")
	}
	ownerSource := source[start:end]
	needle := []byte("let " + name + ":")
	first := bytes.Index(ownerSource, needle)
	if first < 0 || bytes.Contains(ownerSource[first+len(needle):], needle) {
		return backend.Range{}, fmt.Errorf("rust promote binding %q must resolve to one explicitly typed local", name)
	}
	nameStart := start + first + len("let ")
	return byteOffsetsToBackendRange(source, nameStart, nameStart+len(name))
}

func promoteLocalToConstDef() Def[PromoteLocalToConstReq, ExtractionResult] {
	return Def[PromoteLocalToConstReq, ExtractionResult]{
		Key:     canonicalPromoteLocalToConstKey,
		Summary: "Use this operation to promote one immutable Rust local to a constant. It accepts a simple explicitly typed fixed-width integer initialized by a range-valid decimal literal when the binding is used only as a direct value argument of the immediately following call. Macros, attributes, imports, comments, custom types, type shadows, extra uses, mutation, destructuring, aggregates, and unexpected or stale rust-analyzer edits refuse. This is separate from extract-constant, which retains the original local binding.",
		Params:  promoteLocalToConstParams, Level: LevelFile, CLIName: "refactor-promote-local-to-const", MCPName: canonicalPromoteLocalToConstKey,
		Parse:    parsePromoteLocalToConst,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, PromoteLocalToConstReq) (ExtractionResult, error){backend.LanguageRust: runPromoteLocalToConstRust},
		Format:   formatExtractionResult,
		Output:   extractionReceiptOutput(semanticLookupToolName, semanticReplaceBodyToolName),
		FinalizeOutput: func(cc CallContext, req PromoteLocalToConstReq, result ExtractionResult) (ExtractionResult, error) {
			return finalizeRustExtractionOutput(cc, ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol}, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": "src/lib.rs", "symbol": "retry_limit", wireInFunction: "run", "language": "rust", wireTrustWorkspace: true},
		BufferedBatch: true, Batchable: true,
	}
}
