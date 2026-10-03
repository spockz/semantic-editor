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

	"golang.org/x/tools/go/packages"
)

// PlanAliasTypeMigrationOverlay changes one private struct field to a type with exact identity after alias resolution.
func PlanAliasTypeMigrationOverlay(ctx context.Context, workspaceRoot, file, symbol, desiredType, expectedRevision string) (SemanticEditPlan, error) {
	target, err := resolveTypeMigrationTarget(workspaceRoot, file, symbol, desiredType)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	inputs, err := captureTypeMigrationInputs(ctx, target, expectedRevision)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	change, after, err := buildTypeMigrationChange(inputs)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if validationErr := validateTypeMigrationChange(ctx, target, after); validationErr != nil {
		return SemanticEditPlan{}, validationErr
	}
	inputs.readSet, err = completeRefactorReadSet(ctx, target.moduleRoot, inputs.readSet, inputs.packages)
	if err != nil {
		return SemanticEditPlan{}, err
	}
	if err := assertRefactorReadSet(ctx, inputs.readSet); err != nil {
		return SemanticEditPlan{}, err
	}
	return SemanticEditPlan{
		Change:  change,
		ReadSet: inputs.readSet,
		Diff:    UnifiedDiff(target.path, string(inputs.source), string(after)),
	}, nil
}

type typeMigrationTarget struct {
	path        string
	moduleRoot  string
	ownerName   string
	fieldName   string
	desiredType string
}

func resolveTypeMigrationTarget(workspaceRoot, file, symbolName, desiredType string) (typeMigrationTarget, error) {
	ownerName, fieldName, ok := strings.Cut(symbolName, ".")
	if !ok || !token.IsIdentifier(ownerName) || !token.IsIdentifier(fieldName) || strings.Contains(fieldName, ".") {
		return typeMigrationTarget{}, fmt.Errorf("type migration target must be one qualified struct field, such as Settings.timeout")
	}
	if !token.IsIdentifier(desiredType) {
		return typeMigrationTarget{}, fmt.Errorf("bounded type migration accepts one unqualified Go type name")
	}
	workspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return typeMigrationTarget{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	targetPath, err := filepath.Abs(filepath.Join(workspaceRoot, file))
	if err != nil {
		return typeMigrationTarget{}, fmt.Errorf("resolve type migration file: %w", err)
	}
	moduleRoot, err := moveModuleRoot(targetPath)
	if err != nil {
		return typeMigrationTarget{}, err
	}
	if !moveWithin(workspaceRoot, moduleRoot) || !moveWithin(moduleRoot, targetPath) {
		return typeMigrationTarget{}, fmt.Errorf("type migration requires an existing Go file in the selected workspace module")
	}
	if pathErr := rejectMoveSymlinkPath(moduleRoot, targetPath); pathErr != nil {
		return typeMigrationTarget{}, pathErr
	}
	return typeMigrationTarget{path: targetPath, moduleRoot: moduleRoot, ownerName: ownerName, fieldName: fieldName, desiredType: desiredType}, nil
}

type typeMigrationInputs struct {
	target   typeMigrationTarget
	readSet  map[string][]byte
	source   []byte
	packages []*packages.Package
	pkg      *packages.Package
	field    *ast.Field
}

func captureTypeMigrationInputs(ctx context.Context, target typeMigrationTarget, expectedRevision string) (typeMigrationInputs, error) {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if authorizationErr := buffer.AuthorizeWrites(ctx, target.moduleRoot); authorizationErr != nil {
			return typeMigrationInputs{}, authorizationErr
		}
	}
	readSet, err := inspectMoveModuleFiles(ctx, target.moduleRoot, target.path, target.path, target.ownerName+"."+target.fieldName)
	if err != nil {
		return typeMigrationInputs{}, err
	}
	source, ok := readSet[target.path]
	if !ok {
		return typeMigrationInputs{}, fmt.Errorf("type migration target is not an existing Go source file")
	}
	if expectedRevision != "" && revisionOf(source) != expectedRevision {
		return typeMigrationInputs{}, fmt.Errorf("expected_revision does not match captured type source")
	}
	pkgSet, err := loadMovePackages(ctx, target.moduleRoot, readSet)
	if err != nil {
		return typeMigrationInputs{}, fmt.Errorf("load type-migration package types: %w", err)
	}
	if packageErr := validateRefactorPackages(pkgSet, "type migration"); packageErr != nil {
		return typeMigrationInputs{}, packageErr
	}
	readSet, err = completeRefactorReadSet(ctx, target.moduleRoot, readSet, pkgSet)
	if err != nil {
		return typeMigrationInputs{}, err
	}
	pkgSet, err = loadMovePackages(ctx, target.moduleRoot, readSet)
	if err != nil {
		return typeMigrationInputs{}, fmt.Errorf("reload type-migration package types from captured inputs: %w", err)
	}
	if packageErr := validateRefactorPackages(pkgSet, "type migration"); packageErr != nil {
		return typeMigrationInputs{}, packageErr
	}
	pkg := packageForMoveFile(pkgSet, target.path)
	if pkg == nil {
		return typeMigrationInputs{}, fmt.Errorf("type migration target is not in an active Go package")
	}
	fileNode := syntaxForMoveFile(pkg, target.path)
	if fileNode == nil {
		return typeMigrationInputs{}, fmt.Errorf("type migration target file is not active in the loaded package")
	}
	field, err := findPrivateStructField(fileNode, target.path, target.ownerName, target.fieldName, pkg.Fset)
	if err != nil {
		return typeMigrationInputs{}, err
	}
	return typeMigrationInputs{target: target, readSet: readSet, source: source, packages: pkgSet, pkg: pkg, field: field}, nil
}

func buildTypeMigrationChange(inputs typeMigrationInputs) (sourcebuf.Change, []byte, error) {
	field := inputs.field
	if identifier, ok := field.Type.(*ast.Ident); ok && identifier.Name == inputs.target.desiredType {
		return sourcebuf.Change{}, nil, fmt.Errorf("type migration target already uses the requested spelling")
	}
	currentType := inputs.pkg.TypesInfo.TypeOf(field.Type)
	desired, err := types.Eval(inputs.pkg.Fset, inputs.pkg.Types, field.Type.Pos(), inputs.target.desiredType)
	if err != nil || !desired.IsType() {
		if err != nil {
			return sourcebuf.Change{}, nil, fmt.Errorf("resolve desired type %q in package scope: %w", inputs.target.desiredType, err)
		}
		return sourcebuf.Change{}, nil, fmt.Errorf("desired expression %q does not denote a Go type", inputs.target.desiredType)
	}
	if _, validationErr := PlanAliasTypeMigration(currentType.String(), inputs.target.desiredType, currentType, desired.Type); validationErr != nil {
		return sourcebuf.Change{}, nil, validationErr
	}
	start, end := inputs.pkg.Fset.Position(field.Type.Pos()).Offset, inputs.pkg.Fset.Position(field.Type.End()).Offset
	if start < 0 || end > len(inputs.source) || start >= end {
		return sourcebuf.Change{}, nil, fmt.Errorf("type migration field selector has an invalid source range")
	}
	after := make([]byte, 0, len(inputs.source)+len(inputs.target.desiredType)-(end-start))
	after = append(after, inputs.source[:start]...)
	after = append(after, inputs.target.desiredType...)
	after = append(after, inputs.source[end:]...)
	change := sourcebuf.Change{Path: inputs.target.path, Before: bytes.Clone(inputs.source), After: after}
	return change, after, nil
}

func validateTypeMigrationChange(ctx context.Context, target typeMigrationTarget, after []byte) error {
	postPackages, err := typecheckMoveOverlay(ctx, target.moduleRoot, map[string][]byte{target.path: after})
	if err != nil {
		return fmt.Errorf("type migration preflight: %w", err)
	}
	postPkg := packageForMoveFile(postPackages, target.path)
	postFile := syntaxForMoveFile(postPkg, target.path)
	if postPkg == nil || postFile == nil {
		return fmt.Errorf("type migration target disappeared during preflight")
	}
	postField, err := findPrivateStructField(postFile, target.path, target.ownerName, target.fieldName, postPkg.Fset)
	if err != nil {
		return fmt.Errorf("type migration target disappeared during preflight: %w", err)
	}
	postType := postPkg.TypesInfo.TypeOf(postField.Type)
	postDesired, evalErr := types.Eval(postPkg.Fset, postPkg.Types, postField.Type.Pos(), target.desiredType)
	if postType == nil || evalErr != nil || !postDesired.IsType() || !types.Identical(types.Unalias(postType), types.Unalias(postDesired.Type)) {
		return fmt.Errorf("type migration result does not retain the resolved desired type")
	}
	return nil
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
			result, err := matchingPrivateStructField(structure, fieldName)
			if err != nil {
				return nil, err
			}
			if result.found {
				if found != nil {
					return nil, fmt.Errorf("type migration field %s.%s is ambiguous", ownerName, fieldName)
				}
				found = result.field
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("type migration field %s.%s was not found in %s", ownerName, fieldName, path)
	}
	return found, nil
}

type privateStructFieldResult struct {
	field *ast.Field
	found bool
}

func matchingPrivateStructField(structure *ast.StructType, fieldName string) (privateStructFieldResult, error) {
	for _, field := range structure.Fields.List {
		if len(field.Names) != 1 && containsFieldName(field.Names, fieldName) {
			return privateStructFieldResult{}, fmt.Errorf("type migration refuses grouped struct fields")
		}
		for _, name := range field.Names {
			if name.Name != fieldName {
				continue
			}
			if ast.IsExported(name.Name) {
				return privateStructFieldResult{}, fmt.Errorf("type migration supports only unexported struct fields")
			}
			return privateStructFieldResult{field: field, found: true}, nil
		}
	}
	return privateStructFieldResult{}, nil
}

func containsFieldName(names []*ast.Ident, expected string) bool {
	for _, name := range names {
		if name.Name == expected {
			return true
		}
	}
	return false
}
