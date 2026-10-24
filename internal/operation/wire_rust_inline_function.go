// Package operation wires the bounded Rust inline-call assist into the shared refactoring operation.
// WHY: A typed literal substitution needs Rust symbol identity, exact call-site selection, and receipt revalidation.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/readlsp"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

var rustInlineFunctionDeclaration = regexp.MustCompile(`^fn ([A-Za-z_][A-Za-z0-9_]*)\(\)\s*->\s*(i8|i16|i32|i64|i128|isize|u8|u16|u32|u64|u128|usize)\s*\{\s*([0-9][0-9_]*_(?:i8|i16|i32|i64|i128|isize|u8|u16|u32|u64|u128|usize))\s*\}$`)

func runInlineFunctionRust(ctx context.Context, cc CallContext, req InlineFunctionReq) (out SemanticEditRes, handlerErr error) {
	if filepath.Clean(resolveWorkPath(cc.WorkDir, req.File)) != filepath.Clean(resolveWorkPath(cc.WorkDir, req.CallSite.File)) {
		return SemanticEditRes{}, fmt.Errorf("rust inline callable requires the callee and call site in the same selected file")
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
		return SemanticEditRes{}, err
	}
	if ownedBuffer != nil {
		defer func() { out, handlerErr = finishRustOperation(ctx, ownedBuffer, out, handlerErr, "inline-call") }()
	}
	callee, err := service.Lookup(ctx, project, req.Symbol)
	if err != nil {
		return SemanticEditRes{}, err
	}
	if callee == nil || callee.IsAmbiguous() || callee.Definition == nil || callee.Definition.Kind != "function" || callee.Definition.QualifiedName != req.Symbol || callee.Definition.DeclarationRange == nil {
		return SemanticEditRes{}, fmt.Errorf("rust inline callee %q must resolve to one top-level free function", req.Symbol)
	}
	caller, err := service.Lookup(ctx, project, req.CallSite.InFunction)
	if err != nil {
		return SemanticEditRes{}, err
	}
	if caller == nil || caller.IsAmbiguous() || caller.Definition == nil || caller.Definition.Kind != "function" || caller.Definition.QualifiedName != req.CallSite.InFunction || caller.Definition.DeclarationRange == nil {
		return SemanticEditRes{}, fmt.Errorf("rust inline call-site owner %q must resolve to one top-level free function", req.CallSite.InFunction)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return SemanticEditRes{}, fmt.Errorf("read Rust inline source: %w", err)
	}
	revision := structuralRevision(source)
	if req.ExpectedRevision != "" && req.ExpectedRevision != revision || req.CallSite.ExpectedRevision != "" && req.CallSite.ExpectedRevision != revision {
		return SemanticEditRes{}, fmt.Errorf("rust inline source revision changed")
	}
	selection, expression, expected, err := rustInlineCallableCandidate(source, req, *callee.Definition.DeclarationRange, *caller.Definition.DeclarationRange)
	if err != nil {
		return SemanticEditRes{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationInlineFunction, Project: project, File: file,
		Range: selection, Kind: backend.CodeActionInlineKind, Symbol: req.Symbol, Expression: expression,
		EnclosingSymbol: caller.Definition.QualifiedName, ExpectedRevision: revision,
	})
	if err != nil {
		return SemanticEditRes{}, err
	}
	if result == nil || result.File != file || !bytes.Equal(result.Before, source) || !bytes.Equal(result.After, expected) || len(result.Files) != 1 || result.Files[0].File != file || !bytes.Equal(result.Files[0].Before, source) || !bytes.Equal(result.Files[0].After, expected) {
		return SemanticEditRes{}, fmt.Errorf("rust inline backend returned an unexpected target, source preimage, or whole-file postimage")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return SemanticEditRes{}, err
	}
	relative, afterRevision, err := rustExtractionContinuation(cc.WorkDir, req.File, file, result.After)
	if err != nil {
		return SemanticEditRes{}, err
	}
	draft := EditReceiptDraft{
		Operation: string(backend.OperationInlineFunction), Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{
			{File: relative, Symbol: callee.Definition.QualifiedName, ExpectedRevision: &afterRevision},
			{File: relative, Symbol: caller.Definition.QualifiedName, ExpectedRevision: &afterRevision},
		},
	}
	return SemanticEditRes{
		Diff:             astedit.UnifiedDiff(req.CallSite.File, string(source), string(result.After)),
		DiagnosticStatus: diagnosticStatusUnavailable, Draft: draft,
	}, nil
}

func rustInlineCallableCandidate(source []byte, req InlineFunctionReq, calleeRange, callerRange backend.Range) (backend.Range, string, []byte, error) {
	calleeStart, err := readlsp.ByteOffset(source, calleeRange.Start)
	if err != nil {
		return backend.Range{}, "", nil, fmt.Errorf("resolve Rust inline callee start: %w", err)
	}
	calleeEnd, err := readlsp.ByteOffset(source, calleeRange.End)
	if err != nil || calleeEnd <= calleeStart {
		return backend.Range{}, "", nil, fmt.Errorf("invalid Rust inline callee range")
	}
	callerStart, err := readlsp.ByteOffset(source, callerRange.Start)
	if err != nil {
		return backend.Range{}, "", nil, fmt.Errorf("resolve Rust inline caller start: %w", err)
	}
	callerEnd, err := readlsp.ByteOffset(source, callerRange.End)
	if err != nil || callerEnd <= callerStart || callerStart == calleeStart {
		return backend.Range{}, "", nil, fmt.Errorf("invalid or recursive Rust inline caller range")
	}
	callee := rustInlineFunctionDeclaration.FindStringSubmatch(strings.TrimSpace(string(source[calleeStart:calleeEnd])))
	if len(callee) != 4 || callee[1] != req.Symbol || callee[2] != rustIntegerSuffixForInline(callee[3]) {
		return backend.Range{}, "", nil, fmt.Errorf("rust inline supports a zero-argument free function whose body is one explicitly typed decimal integer literal")
	}
	if rustInlineAttachedAttribute(source, calleeStart) || rustInlineAttachedAttribute(source, callerStart) {
		return backend.Range{}, "", nil, fmt.Errorf("rust inline refuses functions with attached attributes or documentation comments")
	}
	callerText := strings.TrimSpace(string(source[callerStart:callerEnd]))
	callerPattern := regexp.MustCompile(`^fn ` + regexp.QuoteMeta(req.CallSite.InFunction) + `\(\)\s*->\s*` + regexp.QuoteMeta(callee[2]) + `\s*\{\s*` + regexp.QuoteMeta(req.Symbol) + `\(\)\s*\}$`)
	if !callerPattern.MatchString(callerText) || req.CallSite.Discriminator != req.Symbol+"()" {
		return backend.Range{}, "", nil, fmt.Errorf("rust inline call site must be the sole zero-argument tail call in a same-file free function with the matching explicit return type")
	}
	trimmedStart := len(source[callerStart:callerEnd]) - len(bytes.TrimLeft(source[callerStart:callerEnd], " \t\r\n"))
	callOffset := callerStart + trimmedStart + strings.LastIndex(callerText, req.CallSite.Discriminator)
	selection, err := byteOffsetsToBackendRange(source, callOffset, callOffset+len(req.Symbol))
	if err != nil {
		return backend.Range{}, "", nil, fmt.Errorf("resolve Rust inline call name: %w", err)
	}
	callEnd := callOffset + len(req.CallSite.Discriminator)
	expected := make([]byte, 0, len(source)-callEnd+callOffset+len(callee[3]))
	expected = append(expected, source[:callOffset]...)
	expected = append(expected, callee[3]...)
	expected = append(expected, source[callEnd:]...)
	return selection, callee[3], expected, nil
}

func rustInlineAttachedAttribute(source []byte, declarationStart int) bool {
	if declarationStart <= 0 || declarationStart > len(source) {
		return false
	}
	lineStart := bytes.LastIndexByte(source[:declarationStart], '\n') + 1
	if lineStart == 0 {
		return false
	}
	previousEnd := lineStart - 1
	previousStart := bytes.LastIndexByte(source[:previousEnd], '\n') + 1
	previous := strings.TrimSpace(string(source[previousStart:previousEnd]))
	return strings.HasPrefix(previous, "#[") || strings.HasPrefix(previous, "///") || strings.HasPrefix(previous, "//!")
}

func rustIntegerSuffixForInline(literal string) string {
	_, after, ok := strings.CutLast(literal, "_")
	if !ok {
		return ""
	}
	return after
}

func finalizeRustInlineFunctionOutput(cc CallContext, req InlineFunctionReq, result SemanticEditRes) (SemanticEditRes, error) {
	project := effectiveProject(cc, req.Project)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	root := cc.WorkDir
	if root == "" {
		root = project.RootDir
	}
	receipt, err := finalizeCallReceipt(cc.Ctx, cc, root, result.Draft, func(ctx context.Context, target SymbolTarget) (SymbolTarget, bool, error) {
		file := resolveWorkPath(root, target.File)
		workingSource, readErr := sourcebuf.ReadFile(ctx, file)
		if readErr != nil {
			return SymbolTarget{}, false, readErr
		}
		if target.ExpectedRevision == nil || structuralRevision(workingSource) != *target.ExpectedRevision {
			return target, false, nil
		}
		selectedProject := project
		selectedProject.File = file
		lookup, lookupErr := service.Lookup(ctx, selectedProject, target.Symbol)
		if lookupErr != nil {
			return SymbolTarget{}, false, lookupErr
		}
		if lookup == nil || lookup.Definition == nil || lookup.IsAmbiguous() || lookup.Definition.Kind != "function" || lookup.Definition.QualifiedName != target.Symbol {
			return target, false, nil
		}
		return target, true, nil
	})
	if err != nil {
		return SemanticEditRes{}, fmt.Errorf("finalize Rust inline receipt: %w", err)
	}
	result.Receipt = receipt
	return result, nil
}
