// This file resolves alias-equivalent type migrations against the current Go overlay and applies one validated edit.

package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"semedit/internal/sourcebuf"
)

// PlanAliasTypeMigrationOverlay changes one private struct field to a type with exact identity after alias resolution.
func PlanAliasTypeMigrationOverlay(ctx context.Context, workspaceRoot, file, symbol, desiredType, expectedRevision string) (SemanticEditPlan, error) {
	ownerName, fieldName, ok := strings.Cut(symbol, ".")
	if !ok || !token.IsIdentifier(ownerName) || !token.IsIdentifier(fieldName) || strings.Contains(fieldName, ".") {
		return SemanticEditPlan{}, fmt.Errorf("type migration target must be one qualified struct field, such as Settings.timeout")
	}
	if !token.IsIdentifier(desiredType) {
		return SemanticEditPlan{}, fmt.Errorf("bounded type migration accepts one unqualified Go type name")
	}
	workspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	targetPath, err := filepath.Abs(filepath.Join(workspaceRoot, file))
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("resolve type migration file: %w", err)
	}
	moduleRoot, err := moveModuleRoot(targetPath)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if !moveWithin(workspaceRoot, moduleRoot) || !moveWithin(moduleRoot, targetPath) {
		return SemanticEditPlan{}, fmt.Errorf("type migration requires an existing Go file in the selected workspace module")
	}
	if err := rejectMoveSymlinkPath(moduleRoot, targetPath); err != nil {
		return SemanticEditPlan{}, err
	}
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if err := buffer.AuthorizeWrites(ctx, moduleRoot); err != nil {
			return SemanticEditPlan{}, err
		}
	}
	readSet, err := inspectMoveModuleFiles(ctx, moduleRoot, targetPath, targetPath, symbol)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	source, ok := readSet[targetPath]
	if !ok {
		return SemanticEditPlan{}, fmt.Errorf("type migration target is not an existing Go source file")
	}
	if expectedRevision != "" && revisionOf(source) != expectedRevision {
		return SemanticEditPlan{}, fmt.Errorf("expected_revision does not match captured type source")
	}
	pkgs, err := loadMovePackages(ctx, moduleRoot, readSet)
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("load type-migration package types: %w", err)
	}
	if err := validateRefactorPackages(pkgs, "type migration"); err != nil {
		return SemanticEditPlan{}, err
	}
	readSet, err = completeRefactorReadSet(ctx, moduleRoot, readSet, pkgs)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	pkgs, err = loadMovePackages(ctx, moduleRoot, readSet)
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("reload type-migration package types from captured inputs: %w", err)
	}
	if err := validateRefactorPackages(pkgs, "type migration"); err != nil {
		return SemanticEditPlan{}, err
	}
	pkg := packageForMoveFile(pkgs, targetPath)
	if pkg == nil {
		return SemanticEditPlan{}, fmt.Errorf("type migration target is not in an active Go package")
	}
	fileNode := syntaxForMoveFile(pkg, targetPath)
	if fileNode == nil {
		return SemanticEditPlan{}, fmt.Errorf("type migration target file is not active in the loaded package")
	}
	field, err := findPrivateStructField(fileNode, targetPath, ownerName, fieldName, pkg.Fset)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if identifier, ok := field.Type.(*ast.Ident); ok && identifier.Name == desiredType {
		return SemanticEditPlan{}, fmt.Errorf("type migration target already uses the requested spelling")
	}
	currentType := pkg.TypesInfo.TypeOf(field.Type)
	desired, err := types.Eval(pkg.Fset, pkg.Types, field.Type.Pos(), desiredType)
	if err != nil || !desired.IsType() {
		if err != nil {
			return SemanticEditPlan{}, fmt.Errorf("resolve desired type %q in package scope: %w", desiredType, err)
		}
		return SemanticEditPlan{}, fmt.Errorf("desired expression %q does not denote a Go type", desiredType)
	}
	if _, err := PlanAliasTypeMigration(currentType.String(), desiredType, currentType, desired.Type); err != nil {
		return SemanticEditPlan{}, err
	}
	start, end := pkg.Fset.Position(field.Type.Pos()).Offset, pkg.Fset.Position(field.Type.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		return SemanticEditPlan{}, fmt.Errorf("type migration field selector has an invalid source range")
	}
	after := make([]byte, 0, len(source)+len(desiredType)-(end-start))
	after = append(after, source[:start]...)
	after = append(after, desiredType...)
	after = append(after, source[end:]...)
	change := sourcebuf.Change{Path: targetPath, Before: bytes.Clone(source), After: after}
	postPackages, err := typecheckMoveOverlay(ctx, moduleRoot, map[string][]byte{targetPath: after})
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("type migration preflight: %w", err)
	}
	postPkg := packageForMoveFile(postPackages, targetPath)
	postFile := syntaxForMoveFile(postPkg, targetPath)
	if postPkg == nil || postFile == nil {
		return SemanticEditPlan{}, fmt.Errorf("type migration target disappeared during preflight")
	}
	postField, err := findPrivateStructField(postFile, targetPath, ownerName, fieldName, postPkg.Fset)
	if err != nil {
		return SemanticEditPlan{}, fmt.Errorf("type migration target disappeared during preflight: %w", err)
	}
	postType := postPkg.TypesInfo.TypeOf(postField.Type)
	postDesired, evalErr := types.Eval(postPkg.Fset, postPkg.Types, postField.Type.Pos(), desiredType)
	if postType == nil || evalErr != nil || !postDesired.IsType() || !types.Identical(types.Unalias(postType), types.Unalias(postDesired.Type)) {
		return SemanticEditPlan{}, fmt.Errorf("type migration result does not retain the resolved desired type")
	}
	readSet, err = completeRefactorReadSet(ctx, moduleRoot, readSet, pkgs)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if err := assertRefactorReadSet(ctx, readSet); err != nil {
		return SemanticEditPlan{}, err
	}
	return SemanticEditPlan{Change: change, ReadSet: readSet, Diff: UnifiedDiff(targetPath, string(source), string(after))}, nil
}

func findPrivateStructField(file *ast.File, path, ownerName, fieldName string, fset *token.FileSet) (*ast.Field, error) {
	var found *ast.Field
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, spec := range group.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != ownerName || filepath.Clean(fset.Position(typeSpec.Pos()).Filename) != filepath.Clean(path) {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("type migration owner %q is not a struct", ownerName)
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) != 1 && containsFieldName(field.Names, fieldName) {
					return nil, fmt.Errorf("type migration refuses grouped struct fields")
				}
				for _, name := range field.Names {
					if name.Name != fieldName {
						continue
					}
					if ast.IsExported(name.Name) {
						return nil, fmt.Errorf("type migration supports only unexported struct fields")
					}
					if found != nil {
						return nil, fmt.Errorf("type migration field %s.%s is ambiguous", ownerName, fieldName)
					}
					found = field
				}
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("type migration field %s.%s was not found in %s", ownerName, fieldName, path)
	}
	return found, nil
}

func containsFieldName(names []*ast.Ident, expected string) bool {
	for _, name := range names {
		if name.Name == expected {
			return true
		}
	}
	return false
}
