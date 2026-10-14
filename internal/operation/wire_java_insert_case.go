// Package operation registers and executes semantic editing operations.
// WHY: This file stages revision-checked Java switch cases before JDT LS verification.
package operation

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func runInsertCaseJava(ctx context.Context, cc CallContext, req InsertCaseReq) (FileEditRes, error) {
	root := cc.WorkDir
	if root == "" {
		root = cc.Project.RootDir
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("resolve Java case insertion workspace: %w", err)
	}
	file := resolveWorkPath(root, req.File)
	project := effectiveProject(cc, req.Project)
	project.RootDir, project.File = root, file
	if req.SwitchPath != "" && req.SelectionRevision == "" {
		return FileEditRes{}, fmt.Errorf("expected_selection_revision is required with construct_path for Java switch insertion")
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	var before, after []byte
	err = sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		var readErr error
		before, readErr = sourcebuf.ReadFile(editCtx, file)
		if readErr != nil {
			return fmt.Errorf("read Java switch insertion target: %w", readErr)
		}
		if req.ExpectedRevision != "" && structuralRevision(before) != req.ExpectedRevision {
			return fmt.Errorf("java switch insertion source revision changed")
		}
		updated, editErr := prepareJavaSwitchCaseEdit(editCtx, service, project, before, req)
		if editErr != nil {
			return editErr
		}
		if writeErr := pipeline.WriteAtomicContext(editCtx, file, updated); writeErr != nil {
			return fmt.Errorf("stage Java switch case: %w", writeErr)
		}
		if _, verifyErr := service.Verify(editCtx, backend.VerifyRequest{Project: project, Path: file, FormatSelectedFile: true, OrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports)}); verifyErr != nil {
			return fmt.Errorf("format and verify Java switch insertion: %w", verifyErr)
		}
		after, readErr = sourcebuf.ReadFile(editCtx, file)
		if readErr != nil {
			return fmt.Errorf("read verified Java switch insertion: %w", readErr)
		}
		return nil
	})
	if err != nil {
		return FileEditRes{}, err
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return FileEditRes{}, fmt.Errorf("derive Java switch insertion receipt path: %w", err)
	}
	if !filepath.IsLocal(relative) {
		return FileEditRes{}, fmt.Errorf("java switch insertion target %q escapes workspace %q", file, root)
	}
	relative = filepath.ToSlash(relative)
	draft := EditReceiptDraft{
		Operation: "insert_case", Status: EditPersisted,
		ReadSet:        []FileRevision{{Path: relative, Revision: structuralRevision(before)}},
		PlannedChanges: []FileRevision{{Path: relative, Revision: structuralRevision(after)}},
	}
	return FileEditRes{
		File: file, Display: displayWorkPath(root, req.File), InFunction: req.Func, Discriminator: req.SwitchOn,
		Diff:   astedit.UnifiedDiff(displayWorkPath(root, req.File), string(before), string(after)),
		Detail: "Inserted Java switch case.", DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true, ReceiptDraft: draft,
	}, nil
}

func prepareJavaSwitchCaseEdit(ctx context.Context, service *backend.Service, project backend.ProjectContext, before []byte, req InsertCaseReq) ([]byte, error) {
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.Func})
	if err != nil {
		return nil, err
	}
	if inspection == nil || len(inspection.Matches) != 1 || inspection.Matches[0].Kind != "method" {
		return nil, fmt.Errorf("java switch owner %q must resolve to one method", req.Func)
	}
	method := inspection.Matches[0]
	methodSource := []byte(method.Source)
	start, err := javacst.ByteOffsetFromPosition(before, method.Range.Start)
	if err != nil {
		return nil, fmt.Errorf("resolve Java switch owner source offset: %w", err)
	}
	end := start + len(methodSource)
	if end > len(before) || string(before[start:end]) != method.Source {
		return nil, fmt.Errorf("java switch owner source changed during insertion selection")
	}
	err = validateJavaSwitchSelection(method, methodSource, req)
	if err != nil {
		return nil, err
	}
	updatedMethod, err := javacst.InsertJavaSwitchCase(methodSource, req.SwitchOn, req.SwitchPath, []byte(req.Case), req.Placement, req.Anchor)
	if err != nil {
		return nil, err
	}
	updated := make([]byte, 0, len(before)-len(methodSource)+len(updatedMethod))
	updated = append(updated, before[:start]...)
	updated = append(updated, updatedMethod...)
	updated = append(updated, before[end:]...)
	return updated, nil
}

func validateJavaSwitchSelection(method backend.InspectedSymbol, source []byte, req InsertCaseReq) error {
	if req.SwitchPath == "" {
		return nil
	}
	candidates, err := javacst.Candidates(source, javacst.ConstructSwitch, req.SwitchOn)
	if err != nil {
		return err
	}
	revision, err := javacst.SelectionRevision(method.QualifiedName, javacst.ConstructSwitch, req.SwitchOn, candidates)
	if err != nil {
		return err
	}
	if revision != req.SelectionRevision {
		return fmt.Errorf("java switch candidate set changed; inspect the current method and reselect the switch")
	}
	return nil
}
