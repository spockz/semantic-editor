// WHY: Keyed field extraction needs typed literal admission and source-order plans separate from delegate proofs.

package operation

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type extractTypeSourceLocation struct {
	file   string
	offset int
}

func extractTypeLocation(fset *token.FileSet, position token.Pos) (extractTypeSourceLocation, error) {
	if fset == nil || !position.IsValid() {
		return extractTypeSourceLocation{}, fmt.Errorf("field extraction requires physical source coordinates")
	}
	physical := fset.PositionFor(position, false)
	if physical.Filename == "" {
		return extractTypeSourceLocation{}, fmt.Errorf("field extraction source filename is unavailable")
	}
	path, err := filepath.Abs(physical.Filename)
	if err != nil {
		return extractTypeSourceLocation{}, fmt.Errorf("resolve field extraction source path: %w", err)
	}
	return extractTypeSourceLocation{file: filepath.Clean(path), offset: physical.Offset}, nil
}

func collectExtractTypeKeyedUses(files []*ast.File) map[*ast.Ident]bool {
	keyed := make(map[*ast.Ident]bool)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, element := range literal.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := pair.Key.(*ast.Ident); ok {
					keyed[key] = true
				}
			}
			return true
		})
	}
	return keyed
}

func extractTypeKeyedLiteralEdits(snapshot *structuralSnapshot, selection extractTypeSelection) ([]sourceEdit, error) {
	var edits []sourceEdit
	for _, file := range selection.Owner.Syntax {
		location, err := extractTypeLocation(selection.Owner.Fset, file.Pos())
		if err != nil {
			return nil, err
		}
		if location.file != snapshot.File {
			continue
		}
		fileEdits, err := extractTypeFileLiteralEdits(snapshot, selection, file)
		if err != nil {
			return nil, err
		}
		edits = append(edits, fileEdits...)
	}
	return edits, nil
}

func extractTypeFileLiteralEdits(snapshot *structuralSnapshot, selection extractTypeSelection, file *ast.File) ([]sourceEdit, error) {
	var edits []sourceEdit
	var visitErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if visitErr != nil {
			return false
		}
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		plan, err := extractTypeKeyedLiteralEdit(snapshot, selection, file, literal)
		if err != nil {
			visitErr = err
			return false
		}
		if plan.changed {
			edits = append(edits, plan.edit)
		}
		return true
	})
	return edits, visitErr
}

type extractTypeLiteralRun struct {
	first ast.Expr
	last  ast.Expr
}

func selectedExtractTypeLiteralRun(pkg *packages.Package, literal *ast.CompositeLit, selected []extractTypeField) (extractTypeLiteralRun, error) {
	var run extractTypeLiteralRun
	closed := false
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return extractTypeLiteralRun{}, fmt.Errorf("field extraction refuses positional owner literals")
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok || !isSelectedExtractField(selected, pkg.TypesInfo.Uses[key]) {
			if run.first != nil {
				closed = true
			}
			continue
		}
		if closed {
			return extractTypeLiteralRun{}, fmt.Errorf("selected keyed fields must form one contiguous source-order run")
		}
		if run.first == nil {
			run.first = pair
		}
		run.last = pair
	}
	return run, nil
}

type extractTypeLiteralPlan struct {
	edit    sourceEdit
	changed bool
}

func extractTypeKeyedLiteralEdit(snapshot *structuralSnapshot, selection extractTypeSelection, file *ast.File, literal *ast.CompositeLit) (extractTypeLiteralPlan, error) {
	pkg := selection.Owner
	owner := pkg.Types.Scope().Lookup(selection.TypeName)
	if owner == nil || !types.Identical(types.Unalias(pkg.TypesInfo.TypeOf(literal)), owner.Type()) {
		return extractTypeLiteralPlan{}, nil
	}
	run, err := selectedExtractTypeLiteralRun(pkg, literal, selection.Selected)
	if err != nil || run.first == nil {
		return extractTypeLiteralPlan{}, err
	}
	name, explicit := literal.Type.(*ast.Ident)
	if !explicit || pkg.TypesInfo.Uses[name] != owner {
		return extractTypeLiteralPlan{}, fmt.Errorf("keyed owner literals require the explicit selected owner type")
	}
	if bindingErr := validateExtractTypeLiteralBinding(pkg, literal, selection.NewType); bindingErr != nil {
		return extractTypeLiteralPlan{}, bindingErr
	}
	if commentErr := validateExtractTypeLiteralComments(pkg.Fset, file, literal); commentErr != nil {
		return extractTypeLiteralPlan{}, commentErr
	}
	start, err := extractTypeLocation(pkg.Fset, run.first.Pos())
	if err != nil {
		return extractTypeLiteralPlan{}, err
	}
	end, err := extractTypeLocation(pkg.Fset, run.last.End())
	if err != nil {
		return extractTypeLiteralPlan{}, err
	}
	source := snapshot.Sources[snapshot.File]
	if start.file != snapshot.File || end.file != snapshot.File || start.offset < 0 || end.offset > len(source) || end.offset <= start.offset {
		return extractTypeLiteralPlan{}, fmt.Errorf("keyed field extraction range is outside the selected source")
	}
	replacement := bytes.Join([][]byte{[]byte(selection.FieldName + ": " + selection.NewType + "{"), source[start.offset:end.offset], []byte("}")}, nil)
	return extractTypeLiteralPlan{edit: sourceEdit{start: start.offset, end: end.offset, text: replacement}, changed: true}, nil
}

func validateExtractTypeLiteralBinding(pkg *packages.Package, literal *ast.CompositeLit, newType string) error {
	scope := pkg.Types.Scope().Innermost(literal.Pos())
	if scope == nil {
		return fmt.Errorf("keyed field extraction cannot establish the literal scope")
	}
	if _, object := scope.LookupParent(newType, literal.Pos()); object != nil {
		return fmt.Errorf("generated type %q is shadowed at a keyed owner literal", newType)
	}
	return nil
}

func validateExtractTypeLiteralComments(fset *token.FileSet, file *ast.File, literal *ast.CompositeLit) error {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//line ") || strings.HasPrefix(comment.Text, "/*line ") {
				return fmt.Errorf("keyed field extraction refuses source-line directives")
			}
		}
		if group.Pos() < literal.End() && group.End() > literal.Pos() {
			return fmt.Errorf("keyed field extraction refuses comments inside owner literals at %s", fset.PositionFor(group.Pos(), false))
		}
	}
	return nil
}
