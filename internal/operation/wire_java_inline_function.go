// Package operation registers and executes semantic editing operations.
// WHY: This file routes Java inlining through JDT LS and finalizes caller and callee revisions.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
)

func runInlineFunctionJava(ctx context.Context, cc CallContext, req InlineFunctionReq) (SemanticEditRes, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	calleeFile := resolveWorkPath(cc.WorkDir, req.File)
	calleeProject := project
	calleeProject.File = calleeFile
	calleeResult, err := service.Inspect(ctx, backend.InspectRequest{Project: calleeProject, Symbol: req.Symbol})
	if err != nil {
		return SemanticEditRes{}, err
	}
	if calleeResult == nil || len(calleeResult.Matches) != 1 || calleeResult.Matches[0].Kind != "method" {
		return SemanticEditRes{}, fmt.Errorf("java inline callee %q must resolve to one method", req.Symbol)
	}
	callee := calleeResult.Matches[0]
	if req.ExpectedRevision != "" && req.ExpectedRevision != callee.Revision {
		return SemanticEditRes{}, fmt.Errorf("java inline callee revision changed")
	}
	callerFile := resolveWorkPath(cc.WorkDir, req.CallSite.File)
	callerProject := project
	callerProject.File = callerFile
	callerResult, err := service.Inspect(ctx, backend.InspectRequest{Project: callerProject, Symbol: req.CallSite.InFunction})
	if err != nil {
		return SemanticEditRes{}, err
	}
	if callerResult == nil || len(callerResult.Matches) != 1 || callerResult.Matches[0].Kind != "method" {
		return SemanticEditRes{}, fmt.Errorf("java inline call-site owner %q must resolve to one method", req.CallSite.InFunction)
	}
	caller := callerResult.Matches[0]
	if req.CallSite.ExpectedRevision != "" && req.CallSite.ExpectedRevision != caller.Revision {
		return SemanticEditRes{}, fmt.Errorf("java inline call-site revision changed")
	}
	candidate, err := selectJavaInlineCallSite(caller.Source, req.CallSite.Discriminator)
	if err != nil {
		return SemanticEditRes{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationInlineFunction, Project: callerProject, File: callerFile,
		Range: offsetJavaRange(caller.Range, candidate.Range), Kind: backend.CodeActionInlineKind,
		ExpectedRevision: caller.Revision,
	})
	if err != nil {
		return SemanticEditRes{}, err
	}
	if result == nil || result.File != callerFile {
		return SemanticEditRes{}, fmt.Errorf("java inline backend returned an unexpected target file")
	}
	draft, err := javaInlineFunctionReceipt(cc, calleeFile, callerFile, callee, caller, result)
	if err != nil {
		return SemanticEditRes{}, err
	}
	return SemanticEditRes{
		Diff:             astedit.UnifiedDiff(req.CallSite.File, string(result.Before), string(result.After)),
		DiagnosticStatus: diagnosticStatusUnavailable, Draft: draft, javaService: service,
	}, nil
}

func javaInlineFunctionReceipt(cc CallContext, calleeFile, callerFile string, callee, caller backend.InspectedSymbol, result *backend.CodeActionResult) (EditReceiptDraft, error) {
	callerRelative, err := relativeWorkPath(cc.WorkDir, callerFile)
	if err != nil {
		return EditReceiptDraft{}, err
	}
	calleeRelative, err := relativeWorkPath(cc.WorkDir, calleeFile)
	if err != nil {
		return EditReceiptDraft{}, err
	}
	afterRevision := structuralRevision(result.After)
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return EditReceiptDraft{}, err
	}
	for _, revision := range readSet {
		if revision.Path == calleeRelative {
			return inlineFunctionReceipt(callerRelative, calleeRelative, caller, callee, afterRevision, readSet, planned), nil
		}
	}
	readSet = append(readSet, FileRevision{Path: calleeRelative, Revision: callee.Revision})
	return inlineFunctionReceipt(callerRelative, calleeRelative, caller, callee, afterRevision, readSet, planned), nil
}

func inlineFunctionReceipt(callerRelative, calleeRelative string, caller, callee backend.InspectedSymbol, afterRevision string, readSet, planned []FileRevision) EditReceiptDraft {
	calleeRevision := callee.Revision
	if calleeRelative == callerRelative {
		calleeRevision = afterRevision
	}
	return EditReceiptDraft{
		Operation: backendInlineFunctionOperation, Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{
			{File: calleeRelative, Symbol: callee.QualifiedName, ExpectedRevision: &calleeRevision},
			{File: callerRelative, Symbol: caller.QualifiedName, ExpectedRevision: &afterRevision},
		},
	}
}

func selectJavaInlineCallSite(source, discriminator string) (javacst.Candidate, error) {
	candidates, err := javacst.Candidates([]byte(source), backend.ConstructStatementOrExpr, discriminator)
	if err != nil {
		return javacst.Candidate{}, fmt.Errorf("resolve Java inline call site: %w", err)
	}
	if len(candidates) > 1 {
		needle := strings.Join(strings.Fields(discriminator), "")
		exact := make([]javacst.Candidate, 0, len(candidates))
		for _, candidate := range candidates {
			if strings.Join(strings.Fields(candidate.Syntax), "") == needle {
				exact = append(exact, candidate)
			}
		}
		if len(exact) == 1 {
			candidates = exact
		}
	}
	if len(candidates) != 1 {
		return javacst.Candidate{}, fmt.Errorf("java inline call-site expression %q resolves to %d candidates; select a unique direct expression", discriminator, len(candidates))
	}
	return candidates[0], nil
}

func relativeWorkPath(root, path string) (string, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return "", fmt.Errorf("derive workspace-relative Java path for %q: %w", path, err)
	}
	return filepath.ToSlash(relative), nil
}

func finalizeJavaInlineFunctionOutput(cc CallContext, result SemanticEditRes) (SemanticEditRes, error) {
	service := result.javaService
	if service == nil {
		service = cc.Service
	}
	if service == nil {
		service = backends.NewDefaultService()
	}
	root := cc.WorkDir
	if root == "" {
		root = cc.Project.RootDir
	}
	receipt, err := finalizeCallReceipt(cc.Ctx, cc, root, result.Draft, func(ctx context.Context, target SymbolTarget) (SymbolTarget, bool, error) {
		if target.ExpectedRevision == nil {
			return target, false, nil
		}
		project := effectiveProject(cc, cc.Project)
		project.File = resolveWorkPath(root, target.File)
		inspection, inspectErr := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: target.Symbol})
		if inspectErr != nil {
			return SymbolTarget{}, false, inspectErr
		}
		if inspection == nil {
			return SymbolTarget{}, false, nil
		}
		for _, match := range inspection.Matches {
			if match.Revision == *target.ExpectedRevision && (match.QualifiedName == target.Symbol || strings.TrimSuffix(match.QualifiedName, "()") == strings.TrimSuffix(target.Symbol, "()")) {
				return target, true, nil
			}
		}
		return target, false, nil
	})
	if err != nil {
		return SemanticEditRes{}, fmt.Errorf("finalize Java inline receipt: %w", err)
	}
	result.Receipt = receipt
	return result, nil
}
