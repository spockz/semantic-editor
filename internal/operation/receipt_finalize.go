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
func FinalizeEditReceipt(ctx context.Context, root string, inBatch bool, draft EditReceiptDraft, resolve ContinuationResolver) (EditReceipt, error) {
	if draft.Operation == "" {
		return EditReceipt{}, fmt.Errorf("finalize receipt: operation is empty")
	}
	receipt := EditReceipt{SchemaVersion: "semedit.edit-receipt/v1", Status: draft.Status, Operation: draft.Operation, ReadSet: append([]FileRevision(nil), draft.ReadSet...)}
	if inBatch && draft.Status != EditRefused && draft.Status != EditNoOp {
		receipt.Status = EditAppliedWorking
	}
	switch receipt.Status {
	case EditRefused, EditNoOp, EditAppliedWorking, EditPersisted, EditPersistedWithDiagnostics, EditPartiallyPublished:
	default:
		return EditReceipt{}, fmt.Errorf("finalize receipt: unsupported outcome %q", receipt.Status)
	}
	if err := validateFileRevisions(receipt.ReadSet, "read_set"); err != nil {
		return EditReceipt{}, fmt.Errorf("finalize receipt read set: %w", err)
	}
	if receipt.Status == EditRefused {
		if len(draft.PlannedChanges) != 0 || len(draft.Continuations) != 0 {
			return EditReceipt{}, fmt.Errorf("refused receipt cannot contain planned changes or continuations: %w", ErrInvalidDefinition)
		}
		if err := ValidateEditReceipt(receipt); err != nil {
			return EditReceipt{}, fmt.Errorf("validate finalized receipt: %w", err)
		}
		return receipt, nil
	}
	if err := validateFileRevisions(draft.PlannedChanges, "planned_changes"); err != nil {
		return EditReceipt{}, fmt.Errorf("finalize planned changes: %w", err)
	}
	if receipt.Status == EditNoOp && len(draft.PlannedChanges) != 0 {
		return EditReceipt{}, fmt.Errorf("no-op receipt cannot contain planned changes: %w", ErrInvalidDefinition)
	}
	readRevisions := make(map[string]string, len(receipt.ReadSet))
	for _, revision := range receipt.ReadSet {
		readRevisions[revision.Path] = revision.Revision
	}
	expectedChanges := 0
	confirmedChanges := 0
	publicationUncertain := false
	knownUnchanged := true
	for _, planned := range draft.PlannedChanges {
		if readRevisions[planned.Path] == planned.Revision {
			continue
		}
		expectedChanges++
		actual, exists, err := receiptFileRevision(ctx, root, planned.Path, inBatch)
		if err != nil {
			return EditReceipt{}, fmt.Errorf("finalize changed file %q: %w", planned.Path, err)
		}
		if exists && actual.Revision == planned.Revision {
			receipt.Changes = append(receipt.Changes, actual)
			confirmedChanges++
			continue
		}
		if exists && readRevisions[planned.Path] != "" && actual.Revision == readRevisions[planned.Path] {
			continue
		}
		knownUnchanged = false
		publicationUncertain = true
		if exists {
			receipt.Changes = append(receipt.Changes, actual)
		}
	}
	if !inBatch && expectedChanges > confirmedChanges {
		if publicationUncertain {
			receipt.Status = EditPartiallyPublished
		} else if knownUnchanged {
			receipt.Status = EditRefused
		}
	}
	if receipt.Status != EditRefused && receipt.Status != EditAppliedWorking && !inBatch && resolve != nil {
		for index, candidate := range draft.Continuations {
			if err := ValidateSymbolTarget(candidate); err != nil {
				return EditReceipt{}, fmt.Errorf("receipt continuation %d: %w", index, err)
			}
			if candidate.ExpectedRevision == nil {
				continue
			}
			candidateFile, exists, err := receiptFileRevision(ctx, root, candidate.File, false)
			if err != nil {
				return EditReceipt{}, fmt.Errorf("verify receipt candidate %d: %w", index, err)
			}
			if !exists || candidateFile.Revision != *candidate.ExpectedRevision {
				continue
			}
			resolved, live, err := resolve(ctx, candidate)
			if err != nil {
				return EditReceipt{}, fmt.Errorf("resolve receipt continuation %d: %w", index, err)
			}
			if !live {
				continue
			}
			if err := ValidateSymbolTarget(resolved); err != nil {
				return EditReceipt{}, fmt.Errorf("resolved receipt continuation %d: %w", index, err)
			}
			fileRevision, exists, err := receiptFileRevision(ctx, root, resolved.File, false)
			if err != nil {
				return EditReceipt{}, fmt.Errorf("verify receipt continuation %d: %w", index, err)
			}
			if !exists {
				continue
			}
			resolved.File = fileRevision.Path
			revision := fileRevision.Revision
			resolved.ExpectedRevision = &revision
			receipt.Continuations = append(receipt.Continuations, resolved)
		}
	}
	if err := ValidateEditReceipt(receipt); err != nil {
		return EditReceipt{}, fmt.Errorf("validate finalized receipt: %w", err)
	}
	return receipt, nil
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
