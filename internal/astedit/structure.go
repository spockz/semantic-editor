// Package astedit provides AST-level code transformation routines including unified structural construct insertion.
package astedit

import (
	"context"
	"fmt"
	"strings"

	"semedit/internal/backend"
)

// StructureResult reports the output of an InsertStructure operation.
type StructureResult struct {
	Diff string
}

// StructureOptions configures structural construct insertion behavior.
type StructureOptions struct {
	Kind                StructureKind
	AccessModifier      AccessModifier
	Placement           Placement
	TargetSymbol        string
	Group               string
	Overwrite           bool
	Function            string
	SwitchOn            string
	SwitchPath          string
	CasePlacement       CasePlacement
	AnchorCase          string
	Visibility          string
	AutoOrganizeImports bool
}

// StructureKind aliases the canonical cross-language syntax taxonomy for insertion.
type StructureKind = backend.StructureKind

// Go structure kind constants name syntax forms handled by InsertStructure.
const (
	StructureKindFunction    = backend.StructureFunction
	StructureKindMethod      = backend.StructureMethod
	StructureKindType        = backend.StructureType
	StructureKindConst       = backend.StructureConst
	StructureKindVar         = backend.StructureVar
	StructureKindDecl        = backend.StructureDecl
	StructureKindDeclaration = backend.StructureDeclaration
	StructureKindCase        = backend.StructureCase
)

// GoStructureKinds lists the structure kinds implemented by the Go handler.
var GoStructureKinds = []StructureKind{
	StructureKindFunction,
	StructureKindMethod,
	StructureKindType,
	StructureKindConst,
	StructureKindVar,
	StructureKindDecl,
	StructureKindDeclaration,
	StructureKindCase,
}

// InsertStructure injects a structural construct (function, method, type, decl, declaration, or case) into filePath.
func InsertStructure(ctx context.Context, filePath string, source string, opts StructureOptions) (StructureResult, error) {
	kind := opts.Kind
	if kind == "" {
		kind = inferStructureKind(source, opts)
	}

	switch kind {
	case StructureKindFunction, StructureKindMethod:
		err := insertFunctionImpl(ctx, filePath, source, FunctionOptions{
			AccessModifier:      opts.AccessModifier,
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return StructureResult{}, err

	case StructureKindType:
		err := insertTypeImpl(ctx, filePath, source, TypeOptions{
			AccessModifier:      opts.AccessModifier,
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return StructureResult{}, err

	case StructureKindDecl, StructureKindConst, StructureKindVar:
		err := insertDeclImpl(ctx, filePath, source, DeclOptions{
			AccessModifier:      opts.AccessModifier,
			Group:               opts.Group,
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			Overwrite:           opts.Overwrite,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return StructureResult{}, err

	case StructureKindCase:
		placement := opts.CasePlacement
		if placement == "" {
			switch opts.Placement {
			case "first":
				placement = CasePlacementFirst
			case "last":
				placement = CasePlacementLast
			case "before", "before_symbol":
				placement = CasePlacementBefore
			case "after", "after_symbol":
				placement = CasePlacementAfter
			case "before_default":
				placement = CasePlacementBeforeDefault
			default:
				placement = ""
			}
		}
		anchorCase := opts.AnchorCase
		if anchorCase == "" {
			anchorCase = opts.TargetSymbol
		}
		diff, err := insertCaseImpl(ctx, filePath, opts.Function, opts.SwitchOn, source, CaseOptions{
			Placement:           placement,
			AnchorCase:          anchorCase,
			SwitchPath:          opts.SwitchPath,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return StructureResult{Diff: diff}, err

	case StructureKindDeclaration:
		err := insertDeclarationImpl(ctx, filePath, source, Options{
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			Visibility:          opts.Visibility,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return StructureResult{}, err

	default:
		return StructureResult{}, fmt.Errorf("unsupported structure kind: %q", kind)
	}
}

func inferStructureKind(source string, opts StructureOptions) StructureKind {
	trimmed := strings.TrimSpace(source)
	if opts.Function != "" || strings.HasPrefix(trimmed, "case ") || strings.HasPrefix(trimmed, "default:") {
		return StructureKindCase
	}
	if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "func(") {
		return StructureKindFunction
	}
	if strings.HasPrefix(trimmed, "type ") {
		return StructureKindType
	}
	if strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "var ") {
		return StructureKindDecl
	}
	return StructureKindDeclaration
}
