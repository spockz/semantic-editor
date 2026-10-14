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

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
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
	ExpectedRevision    string
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
	backend.ConstructElement, backend.ConstructStatementOrExpr,
}

// InsertConstruct injects one declaration, field, case, or statement into filePath.
func InsertConstruct(ctx context.Context, filePath string, source string, opts InsertConstructOptions) (InsertConstructResult, error) {
	if sourcebuf.FromContext(ctx) == nil {
		var result InsertConstructResult
		err := runBufferedFile(ctx, filePath, func(editCtx context.Context) error {
			var err error
			result, err = InsertConstruct(editCtx, filePath, source, opts)
			return err
		})
		return result, err
	}
	kind := opts.Kind
	if kind == "" {
		kind = inferInsertConstructKind(source, opts)
	} else if err := validateInsertConstructKind(source, kind, opts); err != nil {
		return InsertConstructResult{}, err
	}
	return insertConstructByKind(ctx, filePath, source, opts, kind)
}

func insertConstructByKind(ctx context.Context, filePath, source string, opts InsertConstructOptions, kind ConstructKind) (InsertConstructResult, error) {
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
		return insertConstructCase(ctx, filePath, source, opts)

	case backend.ConstructDeclaration:
		err := insertDeclarationImpl(ctx, filePath, source, Options{
			Placement:           opts.Placement,
			TargetSymbol:        opts.TargetSymbol,
			Visibility:          opts.Visibility,
			AutoOrganizeImports: opts.AutoOrganizeImports,
		})
		return InsertConstructResult{}, err

	case backend.ConstructStatementOrExpr:
		return InsertConstructResult{}, insertStatementConstruct(ctx, filePath, source, opts)

	default:
		return InsertConstructResult{}, fmt.Errorf("unsupported Go construct insertion kind: %q", kind)
	}
}

func insertConstructCase(ctx context.Context, filePath, source string, opts InsertConstructOptions) (InsertConstructResult, error) {
	placement := opts.CasePlacement
	if placement == "" {
		placement = casePlacementFromGeneric(opts.Placement)
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
}

func casePlacementFromGeneric(placement Placement) CasePlacement {
	switch placement {
	case "first":
		return CasePlacementFirst
	case "last":
		return CasePlacementLast
	case "before", "before_symbol":
		return CasePlacementBefore
	case "after", "after_symbol":
		return CasePlacementAfter
	case "before_default":
		return CasePlacementBeforeDefault
	default:
		return ""
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
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "insert.go", []byte(candidate), parser.ParseComments|parser.AllErrors)
		if err != nil {
			return &SyntaxError{File: "snippet", Pos: extractSyntaxPosition(fset, err), Snippet: source, Cause: err, Err: ErrSyntax}
		}
		if len(file.Decls) != 1 {
			return fmt.Errorf("%s insertion requires exactly one function declaration", kind)
		}
		fn, ok := file.Decls[0].(*ast.FuncDecl)
		if !ok || (kind == backend.ConstructMethod) != (fn.Recv != nil) {
			return fmt.Errorf("source declaration does not match construct kind %q", kind)
		}
	case backend.ConstructType:
		return validateSingleDeclarationSnippet(source, kind, token.TYPE)
	case backend.ConstructConst:
		return validateSingleDeclarationSnippet(source, kind, token.CONST)
	case backend.ConstructVar:
		return validateSingleDeclarationSnippet(source, kind, token.VAR)
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

func insertStatementConstruct(ctx context.Context, filePath, source string, opts InsertConstructOptions) error {
	placement := StatementPlacement(opts.Placement)
	content, err := sourcebuf.ReadFile(ctx, filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}
	plan, err := PlanInsertStatement(content, InsertStatementIntent{
		Function: opts.Function, Anchor: opts.Anchor, Discriminator: opts.Discriminator, ConstructPath: OptionalString{Present: opts.ConstructPath != "", Value: opts.ConstructPath},
		ExpectedRevision: OptionalString{Present: opts.ExpectedRevision != "", Value: opts.ExpectedRevision},
		Source:           source, Placement: placement,
	})
	if err != nil {
		return fmt.Errorf("plan statement insertion: %w", err)
	}
	return writeConstructSource(ctx, filePath, plan.Source, opts.AutoOrganizeImports)
}

// InferInsertedFieldName validates and returns the name of one inserted struct field.
func InferInsertedFieldName(source string) (string, error) {
	wrapped := "package p\ntype _ struct {\n" + source + "\n}"
	file, err := parser.ParseFile(token.NewFileSet(), "field.go", []byte(wrapped), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse inserted field: %w", err)
	}
	if len(file.Decls) != 1 {
		return "", fmt.Errorf("field source must contain exactly one declaration")
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

func validateSingleDeclarationSnippet(source string, kind ConstructKind, expected token.Token) error {
	snippet, err := parseDeclarationSnippet(source)
	if err != nil {
		return fmt.Errorf("%s insertion requires a valid %s declaration: %w", kind, expected, err)
	}
	if len(snippet.file.Decls) != 1 {
		return fmt.Errorf("%s insertion requires exactly one %s declaration", kind, expected)
	}
	declaration, ok := snippet.file.Decls[0].(*ast.GenDecl)
	if !ok || declaration.Tok != expected {
		return fmt.Errorf("%s insertion requires a %s declaration", kind, expected)
	}
	return nil
}
