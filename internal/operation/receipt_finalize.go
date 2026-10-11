// This file finalizes receipt revisions and live continuation candidates against the actual standalone disk view or batch working view.

package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"semedit/internal/sourcebuf"
)

// EditReceiptDraft carries handler evidence for final receipt projection.
type EditReceiptDraft struct {
	Operation      string
	Status         EditStatus
	PlannedChanges []FileRevision
	ReadSet        []FileRevision
	Continuations  []SymbolTarget
}

// ContinuationResolver rechecks one candidate against the operation's final semantic identity.
type ContinuationResolver func(context.Context, SymbolTarget) (SymbolTarget, bool, error)

// FinalizeEditReceipt derives final revisions from the selected disk or batch working view.
func FinalizeEditReceipt(ctx context.Context, root string, workingView, suppressLiveContinuations bool, draft EditReceiptDraft, resolve ContinuationResolver) (EditReceipt, error) {
	receipt, complete, err := prepareReceipt(draft, workingView)
	if err != nil || complete {
		return receipt, err
	}
	changes, err := reconcileReceiptChanges(ctx, root, workingView, draft.PlannedChanges, &receipt)
	if err != nil {
		return EditReceipt{}, err
	}
	setPublicationStatus(&receipt, workingView, changes)
	if shouldResolveContinuations(receipt, suppressLiveContinuations, resolve) {
		continuations, err := resolveReceiptContinuations(ctx, root, draft.Continuations, resolve)
		if err != nil {
			return EditReceipt{}, err
		}
		receipt.Continuations = continuations
	}
	if err := ValidateEditReceipt(receipt); err != nil {
		return EditReceipt{}, fmt.Errorf("validate finalized receipt: %w", err)
	}
	return receipt, nil
}

func finalizeCallReceipt(ctx context.Context, cc CallContext, root string, draft EditReceiptDraft, resolve ContinuationResolver) (EditReceipt, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return FinalizeEditReceipt(ctx, root, sourcebuf.FromContext(ctx) != nil, cc.InBatch, draft, resolve)
}

func prepareReceipt(draft EditReceiptDraft, inBatch bool) (EditReceipt, bool, error) {
	if draft.Operation == "" {
		return EditReceipt{}, false, fmt.Errorf("finalize receipt: operation is empty")
	}
	receipt := EditReceipt{SchemaVersion: "semedit.edit-receipt/v1", Status: draft.Status, Operation: draft.Operation, ReadSet: append([]FileRevision(nil), draft.ReadSet...)}
	if inBatch && draft.Status != EditRefused && draft.Status != EditNoOp {
		receipt.Status = EditAppliedWorking
	}
	if err := validateReceiptStatus(receipt.Status); err != nil {
		return EditReceipt{}, false, err
	}
	if err := validateFileRevisions(receipt.ReadSet, "read_set"); err != nil {
		return EditReceipt{}, false, fmt.Errorf("finalize receipt read set: %w", err)
	}
	if receipt.Status == EditRefused {
		if len(draft.PlannedChanges) != 0 || len(draft.Continuations) != 0 {
			return EditReceipt{}, false, fmt.Errorf("refused receipt cannot contain planned changes or continuations: %w", ErrInvalidDefinition)
		}
		if err := ValidateEditReceipt(receipt); err != nil {
			return EditReceipt{}, false, fmt.Errorf("validate finalized receipt: %w", err)
		}
		return receipt, true, nil
	}
	if err := validateFileRevisions(draft.PlannedChanges, "planned_changes"); err != nil {
		return EditReceipt{}, false, fmt.Errorf("finalize planned changes: %w", err)
	}
	if receipt.Status == EditNoOp && len(draft.PlannedChanges) != 0 {
		return EditReceipt{}, false, fmt.Errorf("no-op receipt cannot contain planned changes: %w", ErrInvalidDefinition)
	}
	return receipt, false, nil
}

func validateReceiptStatus(status EditStatus) error {
	switch status {
	case EditRefused, EditNoOp, EditAppliedWorking, EditPersisted, EditPersistedWithDiagnostics, EditPartiallyPublished:
		return nil
	default:
		return fmt.Errorf("finalize receipt: unsupported outcome %q", status)
	}
}

type receiptChangeSummary struct {
	expectedChanges    int
	confirmedChanges   int
	publicationUnclear bool
	knownUnchanged     bool
}

func reconcileReceiptChanges(ctx context.Context, root string, inBatch bool, planned []FileRevision, receipt *EditReceipt) (receiptChangeSummary, error) {
	readRevisions := make(map[string]string, len(receipt.ReadSet))
	for _, revision := range receipt.ReadSet {
		readRevisions[revision.Path] = revision.Revision
	}
	summary := receiptChangeSummary{knownUnchanged: true}
	for _, change := range planned {
		if readRevisions[change.Path] == change.Revision {
			continue
		}
		summary.expectedChanges++
		actual, exists, err := receiptFileRevision(ctx, root, change.Path, inBatch)
		if err != nil {
			return receiptChangeSummary{}, fmt.Errorf("finalize changed file %q: %w", change.Path, err)
		}
		if exists && actual.Revision == change.Revision {
			receipt.Changes = append(receipt.Changes, actual)
			summary.confirmedChanges++
			continue
		}
		if exists && readRevisions[change.Path] != "" && actual.Revision == readRevisions[change.Path] {
			continue
		}
		summary.knownUnchanged = false
		summary.publicationUnclear = true
		if exists {
			receipt.Changes = append(receipt.Changes, actual)
		}
	}
	return summary, nil
}

func setPublicationStatus(receipt *EditReceipt, workingView bool, changes receiptChangeSummary) {
	if workingView || changes.expectedChanges <= changes.confirmedChanges {
		return
	}
	if changes.confirmedChanges > 0 || changes.publicationUnclear {
		receipt.Status = EditPartiallyPublished
	} else if changes.knownUnchanged {
		receipt.Status = EditRefused
	}
}

func shouldResolveContinuations(receipt EditReceipt, suppressLiveContinuations bool, resolve ContinuationResolver) bool {
	return receipt.Status != EditRefused && receipt.Status != EditAppliedWorking && !suppressLiveContinuations && resolve != nil
}

func resolveReceiptContinuations(ctx context.Context, root string, candidates []SymbolTarget, resolve ContinuationResolver) ([]SymbolTarget, error) {
	var continuations []SymbolTarget
	for index, candidate := range candidates {
		resolved, live, err := resolveReceiptCandidate(ctx, root, index, candidate, resolve)
		if err != nil {
			return nil, err
		}
		if live {
			continuations = append(continuations, resolved)
		}
	}
	return continuations, nil
}

func resolveReceiptCandidate(ctx context.Context, root string, index int, candidate SymbolTarget, resolve ContinuationResolver) (SymbolTarget, bool, error) {
	if err := ValidateSymbolTarget(candidate); err != nil {
		return SymbolTarget{}, false, fmt.Errorf("receipt continuation %d: %w", index, err)
	}
	if candidate.ExpectedRevision == nil {
		return SymbolTarget{}, false, nil
	}
	candidateFile, exists, err := receiptFileRevision(ctx, root, candidate.File, false)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("verify receipt candidate %d: %w", index, err)
	}
	if !exists || candidateFile.Revision != *candidate.ExpectedRevision {
		return SymbolTarget{}, false, nil
	}
	resolved, live, err := resolve(ctx, candidate)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("resolve receipt continuation %d: %w", index, err)
	}
	if !live {
		return SymbolTarget{}, false, nil
	}
	return refreshReceiptContinuation(ctx, root, index, resolved)
}

func refreshReceiptContinuation(ctx context.Context, root string, index int, resolved SymbolTarget) (SymbolTarget, bool, error) {
	if err := ValidateSymbolTarget(resolved); err != nil {
		return SymbolTarget{}, false, fmt.Errorf("resolved receipt continuation %d: %w", index, err)
	}
	fileRevision, exists, err := receiptFileRevision(ctx, root, resolved.File, false)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("verify receipt continuation %d: %w", index, err)
	}
	if !exists {
		return SymbolTarget{}, false, nil
	}
	resolved.File = fileRevision.Path
	revision := fileRevision.Revision
	resolved.ExpectedRevision = &revision
	return resolved, true, nil
}

func receiptFileRevision(ctx context.Context, root, relative string, inBatch bool) (FileRevision, bool, error) {
	if !filepath.IsLocal(filepath.FromSlash(relative)) {
		return FileRevision{}, false, fmt.Errorf("path %q is not workspace-relative", relative)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return FileRevision{}, false, fmt.Errorf("resolve receipt root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return FileRevision{}, false, fmt.Errorf("canonicalize receipt root: %w", err)
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil {
		path = resolved
	} else if !errors.Is(resolveErr, os.ErrNotExist) {
		return FileRevision{}, false, fmt.Errorf("canonicalize receipt file: %w", resolveErr)
	}
	relativePath, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relativePath) {
		return FileRevision{}, false, fmt.Errorf("receipt path %q escapes workspace root", relative)
	}
	var content []byte
	if inBatch {
		content, err = sourcebuf.ReadFile(ctx, path)
	} else {
		// #nosec G304 -- path is canonicalized and bounded beneath the workspace root above.
		content, err = os.ReadFile(path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return FileRevision{}, false, nil
	}
	if err != nil {
		return FileRevision{}, false, fmt.Errorf("read receipt source: %w", err)
	}
	digest := sha256.Sum256(content)
	return FileRevision{Path: filepath.ToSlash(relativePath), Revision: hex.EncodeToString(digest[:])}, true, nil
}
