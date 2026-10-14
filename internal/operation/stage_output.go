// Package operation declares the closed result shape and continuation graph for retained staging.
package operation

import (
	"slices"

	"github.com/spockz/semantic-editor/internal/backend"
)

// StageReceiptSchema describes the compact status and remaining-step inventory returned by staging controls.
func StageReceiptSchema() *ValueSchema {
	integer := ValueSchema{Kind: SchemaInteger}
	text := ValueSchema{Kind: SchemaString}
	step := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"step_id": text,
	}, Required: []string{"step_id"}}
	steps := ValueSchema{Kind: SchemaArray, Items: &step}
	limits := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"steps": integer, "request_bytes": integer, "dirty_source_bytes": integer, "max_age": integer,
	}, Required: []string{"steps", "request_bytes", "dirty_source_bytes", "max_age"}}
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"status": text, "stage_id": text, "step_id": text, "steps": steps, "step_count": integer,
		"request_bytes": integer, "dirty_source_bytes": integer, "limits": limits,
	}, Required: []string{"status", "stage_id", "steps", "step_count", "request_bytes", "dirty_source_bytes", "limits"}}
}

// StageOutputContinuations declares the staged step selector on a buffer-safe Go or Java mutation result.
func StageOutputContinuations(entry Entry) []OutputContinuation {
	if !entry.BufferedBatch || entry.ReadOnly || (!slices.Contains(entry.Languages, backend.LanguageGo) && !slices.Contains(entry.Languages, backend.LanguageJava)) {
		return nil
	}
	return []OutputContinuation{{
		Path:      "stage_receipt.step_id",
		Schema:    ValueSchema{Kind: SchemaString},
		Role:      RoleSelector,
		Consumers: []string{"semantic_stage_drop"},
		Optional:  true,
	}}
}

// StageDropContinuations exposes each remaining stable step ID as a selector for later drops.
func StageDropContinuations() []OutputContinuation {
	return []OutputContinuation{{
		Path:      "result.steps[].step_id",
		Schema:    ValueSchema{Kind: SchemaString},
		Role:      RoleSelector,
		Consumers: []string{"semantic_stage_drop"},
		Optional:  true,
	}}
}

// StageDropSourceFields lists registered step selectors and the stage-drop remaining-step selector.
func StageDropSourceFields(entries []Entry) []string {
	fields := []string{"semantic_stage_drop.result.steps[].step_id"}
	for _, entry := range entries {
		if len(StageOutputContinuations(entry)) == 0 || entry.MCPName == "" {
			continue
		}
		fields = append(fields, entry.MCPName+".stage_receipt.step_id")
	}
	slices.Sort(fields)
	return slices.Compact(fields)
}
