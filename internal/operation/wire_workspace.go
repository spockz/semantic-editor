// Package operation wires workspace-lifetime operations into the central registry.
package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/snapshot"
)

// SnapshotReq captures one transactional pre-edit snapshot.
type SnapshotReq struct {
	Project     backend.ProjectContext
	Label       string
	Description string
	Paths       []string
}

// GetProjectContext returns the request project for registry dispatch.
func (r SnapshotReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *SnapshotReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// SnapshotRes carries the created snapshot for JSON rendering.
type SnapshotRes struct {
	Result *snapshot.CreateResult
}

var snapshotParams = []ParameterContract{
	{Name: "label", CLIName: "label", JSONName: "label", Type: ParamString, Description: "Human-readable label for the snapshot (e.g. 'before-rename-server')", Required: true, Default: "snapshot"},
	{Name: "description", CLIName: "desc", JSONName: "description", Type: ParamString, Description: "Optional description of the pending change or purpose"},
	{Name: "paths", CLIName: "paths", JSONName: "paths", Type: ParamStringSlice, Description: "Optional list of file or directory paths to capture; if omitted, captures all workspace source files"},
}

func parseSnapshot(raw map[string]any) (SnapshotReq, error) {
	var req SnapshotReq
	if err := CheckParams(raw, snapshotParams); err != nil {
		return req, err
	}
	var err error
	if req.Label, err = ParseString(raw, "label", "", true); err != nil {
		return req, err
	}
	req.Label = strings.TrimSpace(req.Label)
	if req.Label == "" {
		return req, fmt.Errorf("param %q is required: %w", "label", ErrInvalidParams)
	}
	if req.Description, err = ParseString(raw, "description", "", false); err != nil {
		return req, err
	}
	if req.Paths, err = ParseStringSlice(raw, "paths", ""); err != nil {
		return req, err
	}
	return req, nil
}

func runSnapshot(ctx context.Context, cc CallContext, req SnapshotReq) (SnapshotRes, error) {
	res, err := snapshot.Create(ctx, cc.WorkDir, snapshot.CreateOptions{
		Label:       req.Label,
		Description: req.Description,
		Paths:       req.Paths,
	})
	if err != nil {
		return SnapshotRes{}, err
	}
	return SnapshotRes{Result: res}, nil
}

func snapshotDef() Def[SnapshotReq, SnapshotRes] {
	return Def[SnapshotReq, SnapshotRes]{
		Key:     "snapshot",
		Summary: "Capture a pre-edit transactional snapshot only when a later semantic_undo may be needed, creating an immutable journal under .scratch/snapshots/<id>/ for conflict-checked rollback. Do not use it for routine one-way edits, batches, or verification.",
		Params:  snapshotParams,
		Level:   LevelWorkspace,
		CLIName: "snapshot",
		MCPName: "semantic_snapshot",
		Parse:   parseSnapshot,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, SnapshotReq) (SnapshotRes, error){
			backend.LanguageAuto: runSnapshot,
		},
		Format: func(res SnapshotRes) (string, error) {
			data, err := json.MarshalIndent(res.Result, "", "  ")
			if err != nil {
				return "", fmt.Errorf("serialization error: %w", err)
			}
			return string(data), nil
		},
		ExampleRaw: map[string]any{"label": "before-rename-server"},
		Batchable:  false,
	}
}

// UndoReq restores one captured snapshot with conflict checking.
type UndoReq struct {
	Project    backend.ProjectContext
	SnapshotID string
}

// GetProjectContext returns the request project for registry dispatch.
func (r UndoReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *UndoReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// UndoRes carries the restore outcome for JSON rendering.
type UndoRes struct {
	Result *snapshot.UndoResult
}

var undoParams = []ParameterContract{
	{Name: "snapshot_id", CLIName: "id", JSONName: "snapshot_id", Type: ParamString, Description: "The unique snapshot ID to restore (e.g. 'snap-20260917-123456-abcdef' or 'latest')", Required: true},
}

func parseUndo(raw map[string]any) (UndoReq, error) {
	var req UndoReq
	if err := CheckParams(raw, undoParams); err != nil {
		return req, err
	}
	var err error
	if req.SnapshotID, err = ParseString(raw, "snapshot_id", "", true); err != nil {
		return req, err
	}
	req.SnapshotID = strings.TrimSpace(req.SnapshotID)
	if req.SnapshotID == "" {
		return req, fmt.Errorf("param %q is required: %w", "snapshot_id", ErrInvalidParams)
	}
	return req, nil
}

func runUndo(ctx context.Context, cc CallContext, req UndoReq) (UndoRes, error) {
	res, err := snapshot.Undo(ctx, cc.WorkDir, req.SnapshotID)
	if err != nil {
		return UndoRes{}, err
	}
	return UndoRes{Result: res}, nil
}

func undoDef() Def[UndoReq, UndoRes] {
	return Def[UndoReq, UndoRes]{
		Key:     "undo",
		Summary: "Roll back workspace files to a previously captured snapshot state using conflict-checked atomic writes. Fails and performs zero writes if any touched file was modified after the snapshot's recorded post-edit state.",
		Params:  undoParams,
		Level:   LevelWorkspace,
		CLIName: "undo",
		MCPName: "semantic_undo",
		Parse:   parseUndo,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, UndoReq) (UndoRes, error){
			backend.LanguageAuto: runUndo,
		},
		Format: func(res UndoRes) (string, error) {
			data, err := json.MarshalIndent(res.Result, "", "  ")
			if err != nil {
				return "", fmt.Errorf("serialization error: %w", err)
			}
			return string(data), nil
		},
		ExampleRaw: map[string]any{"snapshot_id": "latest"},
		Batchable:  false,
	}
}

// registerWorkspaceOps adds workspace-lifetime operations to the registry.
// Reload (semantic_reload) stays hand-wired: reply-before-reexec cannot fit
// Dispatch's return-a-value contract, and the liveReload gate is server-process state.
func registerWorkspaceOps(registry *Registry) error {
	if err := Register(registry, snapshotDef()); err != nil {
		return err
	}
	return Register(registry, undoDef())
}
