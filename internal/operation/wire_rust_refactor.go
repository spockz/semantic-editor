// Package operation wires the bounded Rust extraction assist into the shared registry.
// WHY: Rust selectors are resolved against the active source snapshot before requesting an exact LSP range.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/readlsp"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func rustOperationContext(ctx context.Context, root string) (context.Context, *sourcebuf.Buffer, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bufferCtx, buffer, owned, err := sourcebuf.Prepare(ctx, root)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare Rust operation snapshot: %w", err)
	}
	if !owned {
		return bufferCtx, nil, nil
	}
	return bufferCtx, buffer, nil
}

func finishRustOperation[T any](ctx context.Context, buffer *sourcebuf.Buffer, result T, handlerErr error, operation string) (T, error) {
	if handlerErr != nil || buffer == nil {
		return result, handlerErr
	}
	if err := buffer.Flush(ctx); err != nil {
		var zero T
		return zero, fmt.Errorf("publish Rust %s result: %w", operation, err)
	}
	return result, nil
}

func runInlineVariableRustWithReceipt(ctx context.Context, cc CallContext, req InlineVariableReq) (out ExtractionResult, handlerErr error) {
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
		defer func() { out, handlerErr = finishRustOperation(ctx, ownedBuffer, out, handlerErr, "inline-variable") }()
	}
	owner, err := service.Lookup(ctx, project, req.InFunction)
	if err != nil {
		return ExtractionResult{}, err
	}
	if owner == nil || owner.Definition == nil || owner.Definition.DeclarationRange == nil || owner.Definition.Kind != "function" || strings.Contains(owner.Definition.QualifiedName, "::") {
		return ExtractionResult{}, fmt.Errorf("rust inline owner %q must resolve to one top-level function", req.InFunction)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read rust inline preimage: %w", err)
	}
	selection, err := resolveRustInlineBindingRange(source, *owner.Definition.DeclarationRange, req.Symbol)
	if err != nil {
		return ExtractionResult{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationInlineVariable, Project: project, File: file,
		Range: selection, Kind: backend.CodeActionInlineKind, Symbol: req.Symbol,
		EnclosingSymbol: owner.Definition.QualifiedName, ExpectedRevision: structuralRevision(source),
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || !filepath.IsAbs(result.File) || !bytes.Equal(result.Before, source) {
		return ExtractionResult{}, fmt.Errorf("rust inline backend returned an unexpected target or source preimage")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	relative, revision, err := rustExtractionContinuation(cc.WorkDir, req.File, file, result.After)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft := EditReceiptDraft{
		Operation: string(backend.OperationInlineVariable), Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{{File: relative, Symbol: owner.Definition.QualifiedName, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction:       owner.Definition.QualifiedName,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Inlined one suffixed-integer Rust local through a verified rust-analyzer assist.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: !bytes.Equal(result.Before, result.After),
	}, Draft: draft}, nil
}

func resolveRustInlineBindingRange(source []byte, owner backend.Range, name string) (backend.Range, error) {
	start, err := readlsp.ByteOffset(source, owner.Start)
	if err != nil {
		return backend.Range{}, fmt.Errorf("resolve rust inline owner start: %w", err)
	}
	end, err := readlsp.ByteOffset(source, owner.End)
	if err != nil || end <= start {
		return backend.Range{}, fmt.Errorf("invalid rust inline owner range")
	}
	declaration := []byte("let " + name + " = ")
	ownerSource := source[start:end]
	first := bytes.Index(ownerSource, declaration)
	if first < 0 || bytes.Contains(ownerSource[first+len(declaration):], declaration) {
		return backend.Range{}, fmt.Errorf("rust inline binding %q must resolve to one untyped local declaration", name)
	}
	nameStart := start + first + len("let ")
	return byteOffsetsToBackendRange(source, nameStart, nameStart+len(name))
}

func runExtractVariableRustWithReceipt(ctx context.Context, cc CallContext, req ExtractVariableReq) (out ExtractionResult, handlerErr error) {
	if req.Kind != "variable" && req.Kind != "constant" {
		return ExtractionResult{}, fmt.Errorf("rust extract_value currently supports kind=variable or kind=constant")
	}
	if req.ConstructPath != "" {
		return ExtractionResult{}, fmt.Errorf("rust extraction does not support construct_path selection")
	}
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
		defer func() { out, handlerErr = finishRustOperation(ctx, ownedBuffer, out, handlerErr, "extract-variable") }()
	}
	owner, err := service.Lookup(ctx, project, req.InFunction)
	if err != nil {
		return ExtractionResult{}, err
	}
	if owner == nil || owner.Definition == nil || owner.Definition.DeclarationRange == nil || owner.Definition.Kind != "function" && owner.Definition.Kind != "method" {
		return ExtractionResult{}, fmt.Errorf("rust extraction owner %q must resolve to one function", req.InFunction)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read rust extraction preimage: %w", err)
	}
	rangeValue, err := resolveRustExtractionRange(source, *owner.Definition.DeclarationRange, req.Discriminator, req.InFunction)
	if err != nil {
		return ExtractionResult{}, err
	}
	backendKind := "refactor.extract"
	if req.Kind == "constant" {
		backendKind = "constant"
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationExtractVariable, Project: project, File: file,
		Range: rangeValue, Kind: backendKind, NewName: req.Symbol,
		Expression: req.Discriminator, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || !filepath.IsAbs(result.File) || !bytes.Equal(result.Before, source) {
		return ExtractionResult{}, fmt.Errorf("rust extraction backend returned an unexpected target or source preimage")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	relative, revision, err := rustExtractionContinuation(cc.WorkDir, req.File, file, result.After)
	if err != nil {
		return ExtractionResult{}, err
	}
	edit := FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: owner.Definition.QualifiedName, Discriminator: req.Discriminator,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Extracted the Rust expression with a rust-analyzer assist and renamed its new local.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: !bytes.Equal(result.Before, result.After),
	}
	draft := EditReceiptDraft{
		Operation: string(backend.OperationExtractVariable), Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{{File: relative, Symbol: owner.Definition.QualifiedName, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: edit, Draft: draft}, nil
}

func runExtractFunctionRustWithReceipt(ctx context.Context, cc CallContext, req ExtractFunctionReq) (out ExtractionResult, handlerErr error) {
	if req.Kind != astedit.ExtractKindFunction {
		return ExtractionResult{}, fmt.Errorf("rust extract_callable supports kind=function only")
	}
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
		defer func() { out, handlerErr = finishRustOperation(ctx, ownedBuffer, out, handlerErr, "extract-callable") }()
	}
	owner, err := service.Lookup(ctx, project, req.InFunction)
	if err != nil {
		return ExtractionResult{}, err
	}
	if owner == nil || owner.Definition == nil || owner.Definition.DeclarationRange == nil || owner.Definition.Kind != "function" || strings.Contains(owner.Definition.QualifiedName, "::") {
		return ExtractionResult{}, fmt.Errorf("rust extraction owner %q must resolve to one top-level function", req.InFunction)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read rust extraction preimage: %w", err)
	}
	selection, selected, err := rustCallableStatementRange(source, *owner.Definition.DeclarationRange, req.StartLine, req.EndLine)
	if err != nil {
		return ExtractionResult{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationExtractFunction, Project: project, File: file,
		Range: selection, Kind: "refactor.extract", NewName: req.Symbol,
		Expression: selected, EnclosingSymbol: owner.Definition.QualifiedName,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || !filepath.IsAbs(result.File) || !bytes.Equal(result.Before, source) {
		return ExtractionResult{}, fmt.Errorf("rust callable extraction backend returned an unexpected target or source preimage")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	relative, revision, err := rustExtractionContinuation(cc.WorkDir, req.File, file, result.After)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft := EditReceiptDraft{
		Operation: string(backend.OperationExtractFunction), Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{{File: relative, Symbol: req.Symbol, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction:       owner.Definition.QualifiedName,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Extracted the Rust statement into a function through a rust-analyzer assist.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: !bytes.Equal(result.Before, result.After),
	}, Draft: draft}, nil
}

func rustCallableStatementRange(source []byte, ownerRange backend.Range, startLine, endLine int) (backend.Range, string, error) {
	if startLine < 1 || endLine < startLine {
		return backend.Range{}, "", fmt.Errorf("rust callable extraction requires an ordered contiguous line selection")
	}
	lineOffset := func(target int) (int, error) {
		offset := 0
		for range target {
			next := bytes.IndexByte(source[offset:], '\n')
			if next < 0 {
				return 0, fmt.Errorf("rust extraction line is outside the selected file")
			}
			offset += next + 1
		}
		return offset, nil
	}
	lineStart, err := lineOffset(startLine - 1)
	if err != nil {
		return backend.Range{}, "", err
	}
	lastStart, err := lineOffset(endLine - 1)
	if err != nil {
		return backend.Range{}, "", err
	}
	lastEnd := len(source)
	if next := bytes.IndexByte(source[lastStart:], '\n'); next >= 0 {
		lastEnd = lastStart + next
	}
	firstLine := source[lineStart:func() int {
		if next := bytes.IndexByte(source[lineStart:], '\n'); next >= 0 {
			return lineStart + next
		}
		return len(source)
	}()]
	lastLine := source[lastStart:lastEnd]
	start := lineStart + len(firstLine) - len(bytes.TrimLeft(firstLine, " \t"))
	end := lastStart + len(bytes.TrimRight(lastLine, " \t\r"))
	if end <= start {
		return backend.Range{}, "", fmt.Errorf("rust callable selection has no complete statement")
	}
	ownerStart, startErr := readlsp.ByteOffset(source, ownerRange.Start)
	ownerEnd, endErr := readlsp.ByteOffset(source, ownerRange.End)
	if startErr != nil || endErr != nil || ownerEnd <= ownerStart || start <= ownerStart || end >= ownerEnd {
		return backend.Range{}, "", fmt.Errorf("rust extraction selection must lie inside its owner function")
	}
	statement := string(source[start:end])
	selectedCallee := ""
	for selectedLine := range strings.SplitSeq(statement, "\n") {
		match := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\(`).FindStringSubmatch(strings.TrimSpace(selectedLine))
		if len(match) != 2 || !strings.HasSuffix(strings.TrimSpace(selectedLine), ";") || selectedCallee != "" && selectedCallee != match[1] {
			return backend.Range{}, "", fmt.Errorf("rust callable selection requires contiguous direct calls to the same callee")
		}
		selectedCallee = match[1]
	}
	selection, err := byteOffsetsToBackendRange(source, start, end)
	if err != nil {
		return backend.Range{}, "", err
	}
	return selection, statement, nil
}

func resolveRustExtractionRange(source []byte, ownerRange backend.Range, discriminator, owner string) (backend.Range, error) {
	ownerStart, err := readlsp.ByteOffset(source, ownerRange.Start)
	if err != nil {
		return backend.Range{}, fmt.Errorf("resolve rust extraction owner start: %w", err)
	}
	ownerEnd, err := readlsp.ByteOffset(source, ownerRange.End)
	if err != nil || ownerEnd <= ownerStart {
		return backend.Range{}, fmt.Errorf("invalid rust extraction owner range")
	}
	if strings.TrimSpace(discriminator) == "" {
		return backend.Range{}, fmt.Errorf("rust extraction discriminator must be exact source text")
	}
	ownerSource := source[ownerStart:ownerEnd]
	first := bytes.Index(ownerSource, []byte(discriminator))
	if first < 0 {
		return backend.Range{}, fmt.Errorf("rust extraction expression %q was not found in owner %q", discriminator, owner)
	}
	if bytes.Contains(ownerSource[first+len(discriminator):], []byte(discriminator)) {
		return backend.Range{}, fmt.Errorf("rust extraction expression %q is ambiguous; construct_path selection is unsupported", discriminator)
	}
	startOffset := ownerStart + first
	return byteOffsetsToBackendRange(source, startOffset, startOffset+len(discriminator))
}

func rustExtractionContinuation(workDir, requestedFile, absoluteFile string, postimage []byte) (string, string, error) {
	relative, err := relativeWorkPath(workDir, absoluteFile)
	if err != nil {
		return "", "", fmt.Errorf("derive Rust extraction continuation path for %q: %w", requestedFile, err)
	}
	return relative, structuralRevision(postimage), nil
}

func byteOffsetsToBackendRange(source []byte, start, end int) (backend.Range, error) {
	startPosition, err := backendPositionAtOffset(source, start)
	if err != nil {
		return backend.Range{}, err
	}
	endPosition, err := backendPositionAtOffset(source, end)
	if err != nil {
		return backend.Range{}, err
	}
	return backend.Range{Start: startPosition, End: endPosition}, nil
}

func backendPositionAtOffset(source []byte, offset int) (backend.Position, error) {
	if offset < 0 || offset > len(source) {
		return backend.Position{}, fmt.Errorf("source offset %d is outside the selected file", offset)
	}
	if !utf8.Valid(source[:offset]) {
		return backend.Position{}, fmt.Errorf("source offset %d splits a UTF-8 sequence", offset)
	}
	line, character := 0, 0
	for _, value := range string(source[:offset]) {
		switch {
		case value == '\n':
			line++
			character = 0
		case value > 0xffff:
			character += 2
		default:
			character++
		}
	}
	return backend.Position{Line: line, Character: character}, nil
}

func finalizeRustExtractionOutput(cc CallContext, req ExtractVariableReq, result ExtractionResult) (ExtractionResult, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	resolver := func(ctx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		project := effectiveProject(cc, req.Project)
		project.File = filepath.Join(cc.WorkDir, filepath.FromSlash(candidate.File))
		lookup, err := service.Lookup(ctx, project, candidate.Symbol)
		if err != nil {
			return SymbolTarget{}, false, err
		}
		if lookup == nil || lookup.Definition == nil {
			return SymbolTarget{}, false, nil
		}
		return candidate, true, nil
	}
	receipt, err := finalizeCallReceipt(cc.Ctx, cc, cc.WorkDir, result.Draft, resolver)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("finalize rust extraction receipt: %w", err)
	}
	result.Receipt = receipt
	return result, nil
}
