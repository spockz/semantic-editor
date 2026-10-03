// Package astedit provides AST-level code transformation routines including unified structural construct insertion.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
)

// InsertConstructResult reports the source change produced by a generic insertion.
type InsertConstructResult struct {
	Diff string
}

// InsertConstructOptions configures one language construct insertion.
type InsertConstructOptions struct {
	Kind                ConstructKind
	AccessModifier      AccessModifier
	Placement           Placement
	TargetSymbol        string
	Group               string
	Function            string
	SwitchOn            string
	SwitchPath          string
	TypeName            string
	Anchor              string
	Discriminator       string
	ConstructPath       string
	CasePlacement       CasePlacement
	AnchorCase          string
	Visibility          string
	AutoOrganizeImports bool
}

// GoInsertConstructKinds lists the insertion kinds implemented by the Go handler.
var GoInsertConstructKinds = []ConstructKind{
	backend.ConstructFunction, backend.ConstructMethod, backend.ConstructType,
	backend.ConstructConst, backend.ConstructVar, backend.ConstructDecl,
	backend.ConstructDeclaration, backend.ConstructCase, backend.ConstructField,
	backend.ConstructStatementOrExpr,
}

// InsertConstruct injects one declaration, field, case, or statement into filePath.
func InsertConstruct(ctx context.Context, filePath string, source string, opts InsertConstructOptions) (InsertConstructResult, error) {
	kind := opts.Kind
	if kind == "" {
		kind = inferInsertConstructKind(source, opts)
	} else if err := validateInsertConstructKind(source, kind, opts); err != nil {
		return InsertConstructResult{}, err
	}

	switch kind {
	case backend.ConstructFunction, backend.ConstructMethod:
		err := insertFunctionImpl(ctx, filePath, source, FunctionOptions{
			AccessModifier:      opts.AccessModifier,
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return InsertConstructResult{}, err

	case backend.ConstructType:
		err := insertTypeImpl(ctx, filePath, source, TypeOptions{
			AccessModifier:      opts.AccessModifier,
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return InsertConstructResult{}, err

	case backend.ConstructDecl, backend.ConstructConst, backend.ConstructVar:
		err := insertDeclImpl(ctx, filePath, source, DeclOptions{
			AccessModifier:      opts.AccessModifier,
			Group:               opts.Group,
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return InsertConstructResult{}, err

	case backend.ConstructCase:
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
		discriminator := opts.SwitchOn
		if discriminator == "" {
			discriminator = opts.Discriminator
		}
		diff, err := insertCaseImpl(ctx, filePath, opts.Function, discriminator, source, CaseOptions{
			Placement:           placement,
			AnchorCase:          anchorCase,
			SwitchPath:          opts.SwitchPath,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return InsertConstructResult{Diff: diff}, err

	case backend.ConstructDeclaration:
		err := insertDeclarationImpl(ctx, filePath, source, Options{
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			Visibility:          opts.Visibility,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return InsertConstructResult{}, err

	case backend.ConstructField:
		return InsertConstructResult{}, insertFieldConstruct(ctx, filePath, source, opts)

	case backend.ConstructStatementOrExpr:
		return InsertConstructResult{}, insertStatementConstruct(ctx, filePath, source, opts)

	default:
		return InsertConstructResult{}, fmt.Errorf("unsupported Go construct insertion kind: %q", kind)
	}
}

func validateInsertConstructKind(source string, kind ConstructKind, opts InsertConstructOptions) error {
	trimmed := strings.TrimSpace(source)
	switch kind {
	case backend.ConstructFunction, backend.ConstructMethod:
		candidate := trimmed
		if !strings.HasPrefix(candidate, "package ") {
			candidate = "package p\n" + candidate
		}
		file, err := parser.ParseFile(token.NewFileSet(), "insert.go", []byte(candidate), parser.ParseComments|parser.AllErrors)
		if err != nil || len(file.Decls) != 1 {
			return fmt.Errorf("%s insertion requires exactly one function declaration", kind)
		}
		fn, ok := file.Decls[0].(*ast.FuncDecl)
		if !ok || (kind == backend.ConstructMethod) != (fn.Recv != nil) {
			return fmt.Errorf("source declaration does not match construct kind %q", kind)
		}
	case backend.ConstructType:
		if !strings.HasPrefix(trimmed, "type ") {
			return fmt.Errorf("type insertion requires a type declaration")
		}
	case backend.ConstructConst:
		if !strings.HasPrefix(trimmed, "const ") {
			return fmt.Errorf("const insertion requires a const declaration")
		}
	case backend.ConstructVar:
		if !strings.HasPrefix(trimmed, "var ") {
			return fmt.Errorf("var insertion requires a var declaration")
		}
	case backend.ConstructField:
		if opts.TypeName == "" {
			return fmt.Errorf("field insertion requires a type selector")
		}
	case backend.ConstructCase:
	case backend.ConstructStatementOrExpr:
		if opts.Function == "" {
			return fmt.Errorf("statement insertion requires an owner function")
		}
	case backend.ConstructDeclaration, backend.ConstructDecl:
	default:
		return fmt.Errorf("unsupported Go construct insertion kind: %q", kind)
	}
	return nil
}

func inferInsertConstructKind(source string, opts InsertConstructOptions) ConstructKind {
	trimmed := strings.TrimSpace(source)
	if opts.TypeName != "" {
		return backend.ConstructField
	}
	if opts.Function != "" && (strings.HasPrefix(trimmed, "case ") || strings.HasPrefix(trimmed, "default:")) {
		return backend.ConstructCase
	}
	if opts.Function != "" {
		return backend.ConstructStatementOrExpr
	}
	if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "func(") {
		return backend.ConstructFunction
	}
	if strings.HasPrefix(trimmed, "type ") {
		return backend.ConstructType
	}
	if strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "var ") {
		return backend.ConstructDecl
	}
	return backend.ConstructDeclaration
}

func insertFieldConstruct(ctx context.Context, filePath, source string, opts InsertConstructOptions) error {
	fieldName, err := inferInsertedFieldName(source)
	if err != nil {
		return err
	}
	placement := FieldPlacement(opts.Placement)
	if placement == "" {
		placement = FieldAtEnd
	}
	content, err := sourcebuf.ReadFile(ctx, filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}
	plan, err := PlanInsertField(content, InsertFieldIntent{
		TypeName: opts.TypeName, FieldName: fieldName, FieldSource: source,
		Placement: placement, TargetField: opts.TargetSymbol,
	})
	if err != nil {
		return fmt.Errorf("plan field insertion: %w", err)
	}
	return writeConstructSource(ctx, filePath, plan.Source, opts.AutoOrganizeImports)
}

func insertStatementConstruct(ctx context.Context, filePath, source string, opts InsertConstructOptions) error {
	placement := StatementPlacement(opts.Placement)
	content, err := sourcebuf.ReadFile(ctx, filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}
	plan, err := PlanInsertStatement(content, InsertStatementIntent{
		Function: opts.Function, Anchor: opts.Anchor, Discriminator: opts.Discriminator, ConstructPath: OptionalString{Present: opts.ConstructPath != "", Value: opts.ConstructPath},
		Source: source, Placement: placement,
	})
	if err != nil {
		return fmt.Errorf("plan statement insertion: %w", err)
	}
	return writeConstructSource(ctx, filePath, plan.Source, opts.AutoOrganizeImports)
}

func inferInsertedFieldName(source string) (string, error) {
	wrapped := "package p\ntype _ struct {\n" + source + "\n}"
	file, err := parser.ParseFile(token.NewFileSet(), "field.go", []byte(wrapped), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse inserted field: %w", err)
	}
	decl, ok := file.Decls[0].(*ast.GenDecl)
	if !ok || len(decl.Specs) != 1 {
		return "", fmt.Errorf("field source must contain exactly one field")
	}
	typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return "", fmt.Errorf("field source must contain exactly one field")
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || len(structType.Fields.List) != 1 || len(structType.Fields.List[0].Names) != 1 {
		return "", fmt.Errorf("field source must contain exactly one named field")
	}
	return structType.Fields.List[0].Names[0].Name, nil
}

func writeConstructSource(ctx context.Context, filePath string, source []byte, organizeImports bool) error {
	formatted, err := format.Source(source)
	if err != nil {
		return fmt.Errorf("format planned insertion: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), filePath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return fmt.Errorf("validate planned insertion: %w", err)
	}
	cleanPath := filepath.Clean(filePath)
	if err := pipeline.WriteAtomicContext(ctx, cleanPath, formatted); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}
	dir := filepath.Dir(cleanPath)
	if organizeImports {
		if err := pipeline.OrganizeImports(ctx, dir, cleanPath); err != nil {
			_ = pipeline.Format(ctx, dir, cleanPath)
			return fmt.Errorf("organize imports: %w", err)
		}
		return nil
	}
	if err := pipeline.Format(ctx, dir, cleanPath); err != nil {
		return fmt.Errorf("format file: %w", err)
	}
	return nil
}
