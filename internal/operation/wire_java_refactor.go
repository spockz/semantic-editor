// Package operation registers and executes semantic editing operations.
// WHY: This file routes Java extraction through JDT LS and records operation receipts.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func codeActionFileRevisions(cc CallContext, result *backend.CodeActionResult) ([]FileRevision, []FileRevision, error) {
	if result == nil || len(result.Files) == 0 {
		return nil, nil, fmt.Errorf("java code action returned no workspace file edits")
	}
	readSet := make([]FileRevision, 0, len(result.Files))
	planned := make([]FileRevision, 0, len(result.Files))
	for _, file := range result.Files {
		relative, err := relativeWorkPath(cc.WorkDir, file.File)
		if err != nil {
			return nil, nil, fmt.Errorf("derive code action receipt path for %q: %w", file.File, err)
		}
		readSet = append(readSet, FileRevision{Path: relative, Revision: structuralRevision(file.Before)})
		if !bytes.Equal(file.Before, file.After) {
			planned = append(planned, FileRevision{Path: relative, Revision: structuralRevision(file.After)})
		}
	}
	return readSet, planned, nil
}

func runExtractVariableJavaWithReceipt(ctx context.Context, cc CallContext, req ExtractVariableReq) (ExtractionResult, error) {
	return runJavaExpressionExtraction(ctx, cc, req, backend.OperationExtractVariable, "refactor.extract.variable", "Extracted the Java expression into a local variable through JDT LS.")
}

func runExtractValueJavaWithReceipt(ctx context.Context, cc CallContext, req ExtractVariableReq) (ExtractionResult, error) {
	if req.Kind == "constant" {
		return runExtractConstantJavaWithReceipt(ctx, cc, extractConstantRequest(req))
	}
	if req.Kind == "field" {
		return ExtractionResult{}, fmt.Errorf("java field extraction is unsupported because JDT LS produces mutable instance fields with method-local assignments")
	}
	return runExtractVariableJavaWithReceipt(ctx, cc, req)
}

func runExtractConstantJavaWithReceipt(ctx context.Context, cc CallContext, req ExtractConstantReq) (ExtractionResult, error) {
	variableReq := ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol, Kind: "constant", InFunction: req.InFunction, Discriminator: req.Discriminator, ConstructPath: req.ConstructPath, ExpectedRevision: req.ExpectedRevision}
	return runJavaExpressionExtraction(ctx, cc, variableReq, backend.OperationExtractConstant, "refactor.extract.constant", "Extracted the Java expression into a constant through JDT LS.")
}

func runInlineVariableJavaWithReceipt(ctx context.Context, cc CallContext, req InlineVariableReq) (ExtractionResult, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.InFunction})
	if err != nil {
		return ExtractionResult{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return ExtractionResult{}, fmt.Errorf("java inline owner %q must resolve to one method", req.InFunction)
	}
	method := inspection.Matches[0]
	candidate, err := javacst.JavaLocalDeclaration([]byte(method.Source), req.Symbol)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("resolve Java local declaration %q: %w", req.Symbol, err)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Java inline preimage: %w", err)
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationInlineVariable, Project: project, File: file,
		Range: offsetJavaRange(method.Range, candidate.Range), Kind: backend.CodeActionInlineKind,
		ExpectedRevision: structuralRevision(source),
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || !filepath.IsAbs(result.File) || result.File != file {
		return ExtractionResult{}, fmt.Errorf("java inline backend returned an unexpected target file")
	}
	relative, err := filepath.Rel(cc.WorkDir, file)
	if err != nil || !filepath.IsLocal(relative) {
		return ExtractionResult{}, fmt.Errorf("derive Java inline receipt path for %q: %w", file, err)
	}
	relative = filepath.ToSlash(relative)
	revision := structuralRevision(result.After)
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft := EditReceiptDraft{
		Operation: string(backend.OperationInlineVariable), Status: EditPersisted,
		ReadSet:        readSet,
		PlannedChanges: planned,
		Continuations:  []SymbolTarget{{File: relative, Symbol: method.QualifiedName, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: method.QualifiedName, Diff: astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Inlined the Java local through the JDT LS refactor.inline action.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
	}, Draft: draft}, nil
}

func runExtractFunctionJavaWithReceipt(ctx context.Context, cc CallContext, req ExtractFunctionReq) (ExtractionResult, error) {
	if req.Kind != astedit.ExtractMethod {
		return ExtractionResult{}, fmt.Errorf("java extract_function supports kind %q only", astedit.ExtractMethod)
	}
	if err := javacst.ValidateJavaIdentifier(req.Symbol); err != nil {
		return ExtractionResult{}, fmt.Errorf("invalid extracted method name: %w", err)
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.InFunction})
	if err != nil {
		return ExtractionResult{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return ExtractionResult{}, fmt.Errorf("java extraction owner %q must resolve to one method", req.InFunction)
	}
	method := inspection.Matches[0]
	if req.StartLine < 1 || req.EndLine < req.StartLine {
		return ExtractionResult{}, fmt.Errorf("java extract_function requires a valid 1-based statement line range")
	}
	candidate, err := javacst.JavaStatementRange([]byte(method.Source), req.StartLine-1-method.Range.Start.Line, req.EndLine-1-method.Range.Start.Line)
	if err != nil {
		return ExtractionResult{}, err
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Java extraction preimage: %w", err)
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationExtractFunction, Project: project, File: file,
		Range: offsetJavaRange(method.Range, candidate.Range), Kind: "refactor.extract.function",
		NewName: req.Symbol, ExpectedRevision: structuralRevision(source),
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || !filepath.IsAbs(result.File) || result.File != file {
		return ExtractionResult{}, fmt.Errorf("java extraction backend returned an unexpected target file")
	}
	relative, err := filepath.Rel(cc.WorkDir, file)
	if err != nil || !filepath.IsLocal(relative) {
		return ExtractionResult{}, fmt.Errorf("derive Java extraction receipt path for %q: %w", file, err)
	}
	relative = filepath.ToSlash(relative)
	revision := structuralRevision(result.After)
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft := EditReceiptDraft{
		Operation: string(backend.OperationExtractFunction), Status: EditPersisted,
		ReadSet:        readSet,
		PlannedChanges: planned,
		Continuations:  []SymbolTarget{{File: relative, Symbol: method.QualifiedName, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction:       method.QualifiedName,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Extracted the Java statements into a method through the JDT LS refactor.extract.function action.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
	}, Draft: draft}, nil
}

func runJavaExpressionExtraction(ctx context.Context, cc CallContext, req ExtractVariableReq, operation backend.Operation, kind, detail string) (ExtractionResult, error) {
	if err := javacst.ValidateJavaIdentifier(req.Symbol); err != nil {
		return ExtractionResult{}, fmt.Errorf("invalid extracted value name: %w", err)
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.InFunction})
	if err != nil {
		return ExtractionResult{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return ExtractionResult{}, fmt.Errorf("java extraction owner %q must resolve to one method", req.InFunction)
	}
	method := inspection.Matches[0]
	selected, err := selectJavaExtractionCandidate(method.Source, req)
	if err != nil {
		return ExtractionResult{}, err
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Java extraction preimage: %w", err)
	}
	if req.ConstructPath != "" && structuralRevision(source) != req.ExpectedRevision {
		return ExtractionResult{}, fmt.Errorf("java extraction source revision changed; inspect the current source and reselect the expression")
	}
	selection := offsetJavaRange(method.Range, selected.Range)
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: operation, Project: project, File: file,
		Range: selection, Kind: kind, NewName: req.Symbol,
		ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || !filepath.IsAbs(result.File) || result.File != file {
		return ExtractionResult{}, fmt.Errorf("java extraction backend returned an unexpected target file")
	}
	relative, err := filepath.Rel(cc.WorkDir, file)
	if err != nil || !filepath.IsLocal(relative) {
		return ExtractionResult{}, fmt.Errorf("derive Java extraction receipt path for %q: %w", file, err)
	}
	relative = filepath.ToSlash(relative)
	revision := structuralRevision(result.After)
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft := EditReceiptDraft{
		Operation: string(operation), Status: EditPersisted,
		ReadSet:        readSet,
		PlannedChanges: planned,
		Continuations:  []SymbolTarget{{File: relative, Symbol: method.QualifiedName, ExpectedRevision: &revision}},
	}
	return ExtractionResult{
		Edit: FileEditRes{
			File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
			InFunction: method.QualifiedName, Discriminator: req.Discriminator,
			Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
			Detail:           detail,
			DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
		},
		Draft: draft,
	}, nil
}

func selectJavaExtractionCandidate(methodSource string, req ExtractVariableReq) (*javacst.Candidate, error) {
	candidates, err := javacst.Candidates([]byte(methodSource), backend.ConstructStatementOrExpr, req.Discriminator)
	if err != nil {
		return nil, err
	}
	exact := candidates[:0]
	for _, candidate := range candidates {
		if compactJavaSelector(candidate.Syntax) == compactJavaSelector(req.Discriminator) {
			exact = append(exact, candidate)
		}
	}
	candidates = exact
	switch {
	case req.ConstructPath != "":
		for index := range candidates {
			if candidates[index].ConstructPath == req.ConstructPath {
				return &candidates[index], nil
			}
		}
	case len(candidates) == 1:
		return &candidates[0], nil
	case len(candidates) > 1:
		paths := make([]string, len(candidates))
		for index := range candidates {
			paths[index] = candidates[index].ConstructPath
		}
		return nil, fmt.Errorf("java extraction selector %q is ambiguous; choose construct_path from %s", req.Discriminator, strings.Join(paths, ", "))
	}
	return nil, fmt.Errorf("java extraction construct_path %q does not select a current expression candidate", req.ConstructPath)
}

func compactJavaSelector(source string) string { return strings.Join(strings.Fields(source), "") }

func finalizeJavaExtractionOutput(cc CallContext, req ExtractVariableReq, result ExtractionResult) (ExtractionResult, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	resolver := func(ctx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		project := effectiveProject(cc, req.Project)
		project.File = filepath.Join(cc.WorkDir, filepath.FromSlash(candidate.File))
		inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: candidate.Symbol})
		if err != nil {
			return SymbolTarget{}, false, err
		}
		if inspection == nil || len(inspection.Matches) != 1 {
			return SymbolTarget{}, false, nil
		}
		return candidate, true, nil
	}
	receipt, err := finalizeCallReceipt(cc.Ctx, cc, cc.WorkDir, result.Draft, resolver)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("finalize Java extraction receipt: %w", err)
	}
	result.Receipt = receipt
	return result, nil
}

func offsetJavaRange(base, relative backend.Range) backend.Range {
	add := func(start, offset backend.Position) backend.Position {
		position := backend.Position{Line: start.Line + offset.Line, Character: offset.Character}
		if offset.Line == 0 {
			position.Character += start.Character
		}
		return position
	}
	return backend.Range{Start: add(base.Start, relative.Start), End: add(base.Start, relative.End)}
}

func finalizeJavaMethodActionOutput[Req any](cc CallContext, _ Req, result ExtractionResult) (ExtractionResult, error) {
	if len(result.Draft.Continuations) != 1 || result.Draft.Continuations[0].Symbol == "" {
		return ExtractionResult{}, fmt.Errorf("java method action requires one resolved post-edit method continuation")
	}
	receipt, err := finalizeCallReceipt(cc.Ctx, cc, cc.WorkDir, result.Draft, func(_ context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		return candidate, true, nil
	})
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("finalize Java method action receipt: %w", err)
	}
	result.Receipt = receipt
	return result, nil
}
