// Package operation shares structural source planning before publication so
// retry alternatives obey the same revision, type, and layout proofs as execution.
package operation

import (
	"context"
	"fmt"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
)

func prepareStructuralPlanInput(ctx context.Context, cc CallContext, raw map[string]any) (*structuralSnapshot, error) {
	snapshot, err := prepareStructuralSnapshot(ctx, cc, stringParam(raw, "file"))
	if err != nil {
		return nil, err
	}
	if expected, ok := raw["expected_revision"]; ok {
		revision, valid := expected.(string)
		if !valid || revision == "" {
			return nil, fmt.Errorf("expected_revision must be a non-empty string")
		}
		if revision != structuralRevision(snapshot.Sources[snapshot.File]) {
			return nil, fmt.Errorf("expected_revision does not match the current selected-file revision")
		}
	}
	return snapshot, nil
}

func planStructuralSource(snapshot *structuralSnapshot, raw map[string]any, spec structuralPlanSpec) (astedit.SourcePlan, error) {
	plan, err := spec.Plan(snapshot, raw)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	if plan.Changed {
		if err := snapshot.validateOutput(plan.Source); err != nil {
			return astedit.SourcePlan{}, fmt.Errorf("validate structural plan output: %w", err)
		}
	}
	return plan, nil
}

func insertElementConstructRaw(req InsertConstructReq) map[string]any {
	raw := map[string]any{
		"file": req.File, wireInDeclaration: req.InDeclaration, wireDiscriminator: req.Discriminator,
		"source": req.Source, "placement": req.Placement,
	}
	if req.ExpectedRevision != "" {
		raw[wireExpectedRevision] = req.ExpectedRevision
	}
	return raw
}

func insertFieldConstructRaw(req InsertConstructReq) (map[string]any, error) {
	fieldSymbol, err := constructFieldSymbol(req.TypeName, req.Source)
	if err != nil {
		return nil, err
	}
	fieldName := strings.TrimPrefix(fieldSymbol, req.TypeName+".")
	if err := astedit.ValidateAccess(astedit.GolangBackend{}, astedit.AccessModifier(req.AccessModifier), fieldName); err != nil {
		return nil, fmt.Errorf("validate field access modifier: %w: %w", err, ErrInvalidParams)
	}
	raw := map[string]any{
		"file": req.File, "in_type": req.TypeName, "symbol": fieldSymbol,
		"source": req.Source, "placement": req.Placement,
	}
	if req.TargetSymbol != "" {
		raw["target_symbol"] = req.TargetSymbol
	}
	if req.ExpectedRevision != "" {
		raw[wireExpectedRevision] = req.ExpectedRevision
	}
	return raw, nil
}

func validateInsertConstructTarget(ctx context.Context, cc CallContext, raw map[string]any) error {
	request, err := parseInsertConstruct(raw)
	if err != nil {
		return err
	}
	switch request.Kind {
	case backend.ConstructElement:
		selected := insertElementConstructRaw(request)
		snapshot, err := prepareStructuralPlanInput(ctx, cc, selected)
		if err != nil {
			return err
		}
		_, err = planStructuralSource(snapshot, selected, insertElementPlanSpec())
		return err
	case backend.ConstructField:
		selected, err := insertFieldConstructRaw(request)
		if err != nil {
			return err
		}
		return insertFieldStructuralDef().TargetPreflight(ctx, cc, selected)
	default:
		return backend.ErrUnsupportedOperation
	}
}
