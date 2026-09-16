// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// CasePlacement specifies relative placement for a newly inserted switch case.
type CasePlacement string

const (
	// CasePlacementFirst inserts as the first case in the switch.
	CasePlacementFirst CasePlacement = "first"
	// CasePlacementLast inserts as the last case in the switch.
	CasePlacementLast CasePlacement = "last"
	// CasePlacementBeforeDefault inserts immediately before the default clause, or as the last case if absent.
	CasePlacementBeforeDefault CasePlacement = "before_default"
	// CasePlacementBefore inserts before the anchor case.
	CasePlacementBefore CasePlacement = "before"
	// CasePlacementAfter inserts after the anchor case.
	CasePlacementAfter CasePlacement = "after"
)

// CaseOptions configures case insertion behavior.
type CaseOptions struct {
	Placement           CasePlacement
	AnchorCase          string
	AutoOrganizeImports bool
}

// InsertCase injects a case clause into an existing switch statement.
func InsertCase(ctx context.Context, filePath, funcName, switchOn, caseSource string, opts CaseOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}

	recv, name, err := symbol.ParseIdentifier(funcName)
	if err != nil {
		return "", &symbol.SymbolError{
			Op:     "insert_case",
			File:   filePath,
			Symbol: funcName,
			Err:    err,
		}
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments)
	if err != nil {
		pos := extractSyntaxPosition(fset, err)
		return "", &SyntaxError{
			File:  cleanPath,
			Pos:   pos,
			Cause: err,
			Err:   ErrSyntax,
		}
	}

	var targetFunc *ast.FuncDecl
	for _, decl := range fileNode.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if extractReceiverName(fn.Recv) == recv && fn.Name.Name == name {
			targetFunc = fn
			break
		}
	}

	if targetFunc == nil {
		return "", &symbol.SymbolError{
			Op:     "insert_case",
			File:   filePath,
			Symbol: funcName,
			Err:    symbol.ErrNotFound,
		}
	}

	if targetFunc.Body == nil {
		return "", &symbol.SymbolError{
			Op:     "insert_case",
			File:   filePath,
			Symbol: funcName,
			Pos:    fset.Position(targetFunc.Pos()),
			Err:    ErrNoBody,
		}
	}

	switchBody, err := locateSwitch(fset, targetFunc.Body, strings.TrimSpace(switchOn))
	if err != nil {
		return "", fmt.Errorf("%w: %s in %s", ErrSwitchNotFound, switchOn, funcName)
	}

	caseTrimmed := strings.TrimSpace(caseSource)
	synthetic := "package dummy\n\nfunc _() {\nswitch {\n" + caseTrimmed + "\n}\n}\n"
	synthFset := token.NewFileSet()
	if _, err := parser.ParseFile(synthFset, "snippet", synthetic, parser.ParseComments); err != nil {
		pos := extractSyntaxPosition(synthFset, err)
		if pos.Line > 4 {
			pos.Line -= 4
		}
		return "", &SyntaxError{
			File:    "snippet",
			Snippet: caseSource,
			Pos:     pos,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	offset, err := calculateCaseOffset(fset, content, switchBody, opts)
	if err != nil {
		return "", err
	}

	var newContent bytes.Buffer
	newContent.Write(content[:offset])
	newContent.WriteString("\n")
	newContent.WriteString(caseTrimmed)
	newContent.WriteString("\n")
	newContent.Write(content[offset:])

	formatted, err := format.Source(newContent.Bytes())
	if err != nil {
		return "", &SyntaxError{
			File:    cleanPath,
			Snippet: caseSource,
			Cause:   err,
			Err:     ErrSyntax,
		}
	}

	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}

	dir := filepath.Dir(cleanPath)
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, dir, cleanPath); err != nil {
			_ = pipeline.Format(ctx, dir, cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	} else {
		if err := pipeline.Format(ctx, dir, cleanPath); err != nil {
			return "", fmt.Errorf("format file: %w", err)
		}
	}

	diff := UnifiedDiff(funcName, "", caseTrimmed)
	return diff, nil
}

func locateSwitch(fset *token.FileSet, body *ast.BlockStmt, targetSwitchOn string) (*ast.BlockStmt, error) {
	var canonicalTarget string
	if targetSwitchOn != "" {
		if parsed, err := parser.ParseExpr(targetSwitchOn); err == nil {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, parsed); err == nil {
				canonicalTarget = buf.String()
			}
		}
		if canonicalTarget == "" {
			canonicalTarget = targetSwitchOn
		}
	}

	var foundBody *ast.BlockStmt
	ast.Inspect(body, func(n ast.Node) bool {
		if foundBody != nil {
			return false
		}
		switch s := n.(type) {
		case *ast.SwitchStmt:
			if targetSwitchOn == "" {
				if s.Tag == nil {
					foundBody = s.Body
					return false
				}
			} else if s.Tag != nil {
				var buf bytes.Buffer
				if err := format.Node(&buf, fset, s.Tag); err == nil {
					if buf.String() == canonicalTarget || strings.TrimSpace(buf.String()) == canonicalTarget {
						foundBody = s.Body
						return false
					}
				}
			}
		case *ast.TypeSwitchStmt:
			if targetSwitchOn != "" && s.Assign != nil {
				var buf bytes.Buffer
				if err := format.Node(&buf, fset, s.Assign); err == nil {
					if strings.Contains(buf.String(), canonicalTarget) || buf.String() == canonicalTarget {
						foundBody = s.Body
						return false
					}
				}
			}
		}
		return true
	})

	if foundBody == nil {
		return nil, ErrSwitchNotFound
	}
	return foundBody, nil
}

func calculateCaseOffset(fset *token.FileSet, content []byte, switchBody *ast.BlockStmt, opts CaseOptions) (int, error) {
	lbraceOffset := fset.Position(switchBody.Lbrace).Offset
	rbraceOffset := fset.Position(switchBody.Rbrace).Offset

	var clauses []*ast.CaseClause
	for _, stmt := range switchBody.List {
		if cc, ok := stmt.(*ast.CaseClause); ok {
			clauses = append(clauses, cc)
		}
	}

	placement := opts.Placement
	if placement == "" {
		placement = CasePlacementBeforeDefault
	}

	switch placement {
	case CasePlacementFirst:
		if len(clauses) > 0 {
			return fset.Position(clauses[0].Pos()).Offset, nil
		}
		return lbraceOffset + 1, nil

	case CasePlacementLast:
		return rbraceOffset, nil

	case CasePlacementBeforeDefault:
		for _, cc := range clauses {
			if cc.List == nil {
				return fset.Position(cc.Pos()).Offset, nil
			}
		}
		return rbraceOffset, nil

	case CasePlacementBefore:
		cc, err := findAnchorCase(fset, content, clauses, opts.AnchorCase)
		if err != nil {
			return 0, err
		}
		return fset.Position(cc.Pos()).Offset, nil

	case CasePlacementAfter:
		cc, err := findAnchorCase(fset, content, clauses, opts.AnchorCase)
		if err != nil {
			return 0, err
		}
		return fset.Position(cc.End()).Offset, nil

	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedPlacement, placement)
	}
}

func findAnchorCase(fset *token.FileSet, content []byte, clauses []*ast.CaseClause, rawAnchor string) (*ast.CaseClause, error) {
	cleanAnchor := strings.TrimSpace(rawAnchor)
	cleanAnchor = strings.TrimPrefix(cleanAnchor, "case ")
	cleanAnchor = strings.TrimSuffix(cleanAnchor, ":")
	cleanAnchor = strings.TrimSpace(cleanAnchor)

	if cleanAnchor == "" {
		return nil, fmt.Errorf("%w: empty anchor case", ErrAnchorNotFound)
	}

	var canonicalAnchor string
	if expr, err := parser.ParseExpr(cleanAnchor); err == nil {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, expr); err == nil {
			canonicalAnchor = buf.String()
		}
	}
	if canonicalAnchor == "" {
		canonicalAnchor = cleanAnchor
	}

	for _, cc := range clauses {
		if cleanAnchor == "default" && cc.List == nil {
			return cc, nil
		}
		for _, expr := range cc.List {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, expr); err == nil {
				if buf.String() == canonicalAnchor || buf.String() == cleanAnchor {
					return cc, nil
				}
			}
			rawExpr := strings.TrimSpace(string(content[fset.Position(expr.Pos()).Offset:fset.Position(expr.End()).Offset]))
			if rawExpr == cleanAnchor || rawExpr == canonicalAnchor {
				return cc, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrAnchorNotFound, rawAnchor)
}
