// Package astedit keeps package-variable relocation separate because mutable storage and package initialization order need stronger proof than a declaration-only rewrite.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

type crossPackageVariable struct {
	declaration *ast.GenDecl
	spec        *ast.ValueSpec
	object      *types.Var
	initial     constant.Value
}

func resolveCrossPackageVariable(snapshot crossPackageMoveSnapshot, symbolQuery string, selected crossPackagePackages) (*crossPackageVariable, bool, error) {
	for _, declaration := range selected.sourceFile.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, raw := range group.Specs {
			spec, ok := raw.(*ast.ValueSpec)
			if !ok {
				continue
			}
			selectedName := false
			for _, name := range spec.Names {
				selectedName = selectedName || name.Name == symbolQuery
			}
			if !selectedName {
				continue
			}
			variable, err := validateCrossPackageVariable(snapshot, symbolQuery, selected, group, spec)
			if err != nil {
				return nil, true, err
			}
			return variable, true, nil
		}
	}
	return nil, false, nil
}

func validateCrossPackageVariable(snapshot crossPackageMoveSnapshot, name string, selected crossPackagePackages, declaration *ast.GenDecl, spec *ast.ValueSpec) (*crossPackageVariable, error) {
	if declaration.Lparen.IsValid() || len(declaration.Specs) != 1 || len(spec.Names) != 1 || len(spec.Values) > 1 {
		return nil, fmt.Errorf("cross-package variable move requires one standalone ungrouped variable")
	}
	identifier := spec.Names[0]
	if identifier.Name != name || !ast.IsExported(name) {
		return nil, fmt.Errorf("cross-package variable move requires one exported variable")
	}
	object, ok := selected.source.TypesInfo.Defs[identifier].(*types.Var)
	if !ok || object == nil || object.Pkg() == nil || object.Pkg().Path() != selected.source.PkgPath || filepath.Clean(selected.source.Fset.Position(object.Pos()).Filename) != snapshot.sourcePath || object.Parent() != selected.source.Types.Scope() {
		return nil, fmt.Errorf("cross-package variable move cannot resolve the selected package variable")
	}
	if existing := selected.destination.Types.Scope().Lookup(name); existing != nil {
		return nil, fmt.Errorf("destination already declares %q", name)
	}
	basic, ok := object.Type().(*types.Basic)
	if len(spec.Values) > 0 && (!ok || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) == 0) {
		return nil, fmt.Errorf("cross-package variable move supports only predeclared basic scalar types")
	}
	if err := validateCrossPackagePredeclaredTypeExpression(selected.source, spec.Type); err != nil {
		return nil, err
	}
	var initial constant.Value
	if len(spec.Values) == 1 {
		value, exists := selected.source.TypesInfo.Types[spec.Values[0]]
		if !exists || value.Value == nil {
			return nil, fmt.Errorf("cross-package variable move requires an omitted or compile-time constant initializer")
		}
		initial = value.Value
		if err := validateCrossPackagePureInitializer(selected.source, spec); err != nil {
			return nil, err
		}
	}
	if err := validateCrossPackageSelectedFile(snapshot.sourcePath, selected.sourceFile); err != nil {
		return nil, err
	}
	if err := validateCrossPackageSelectedFile(snapshot.destinationPath, selected.destFile); err != nil {
		return nil, err
	}
	if err := rejectCrossPackageTestReferences(snapshot.files, selected.all, object); err != nil {
		return nil, err
	}
	start, end := moveDeclOffsets(selected.source.Fset, declaration, selected.sourceFile, snapshot.files[snapshot.sourcePath])
	if start < 0 || end < start || end > len(snapshot.files[snapshot.sourcePath]) {
		return nil, fmt.Errorf("invalid source declaration bounds for %q", name)
	}
	return &crossPackageVariable{declaration: declaration, spec: spec, object: object, initial: initial}, nil
}

func buildCrossPackageVariableMoveChanges(ctx context.Context, snapshot crossPackageMoveSnapshot, target crossPackageMoveTarget, opts MoveConstructOptions) (crossPackageMoveChangeSet, error) {
	return buildCrossPackageValueMoveChanges(ctx, snapshot, target, opts, moveReferenceVariable, validateCrossPackageVariableMoveOverlay)
}

func validateCrossPackageVariableMoveOverlay(ctx context.Context, snapshot crossPackageMoveSnapshot, target crossPackageMoveTarget, changes map[string][]byte, expected map[moveReferenceKey][]moveReferenceExpectation) error {
	for path := range changes {
		if _, ok := snapshot.files[path]; !ok {
			return fmt.Errorf("reference file is outside the preflight module snapshot: %s", path)
		}
	}
	if err := validateMoveSourceAndDestination(changes, snapshot.sourcePath, snapshot.destinationPath); err != nil {
		return err
	}
	after, err := typecheckMoveOverlay(ctx, snapshot.moduleRoot, changes)
	if err != nil {
		return err
	}
	if err := validateCrossPackagePackageInitialization(target.packages, after, map[*packages.Package]bool{}); err != nil {
		return err
	}
	destination := packageByPath(after, target.destinationPackage.PkgPath)
	if destination == nil {
		return fmt.Errorf("cross-package variable move cannot identify the destination package")
	}
	moved, ok := destination.Types.Scope().Lookup(target.variable.object.Name()).(*types.Var)
	if !ok || moved.Pkg() == nil || moved.Pkg().Path() != target.destinationPackage.PkgPath || filepath.Clean(destination.Fset.Position(moved.Pos()).Filename) != snapshot.destinationPath {
		return fmt.Errorf("cross-package variable move cannot identify moved variable %q", target.variable.object.Name())
	}
	if !types.Identical(target.variable.object.Type(), moved.Type()) {
		return fmt.Errorf("cross-package variable move changed the variable type")
	}
	var currentInitial constant.Value
	if len(target.variable.spec.Values) == 1 {
		declaration, spec := findMovedVariableDeclaration(destination, snapshot.destinationPath, target.variable.object.Name())
		if declaration == nil || spec == nil || len(spec.Values) != 1 {
			return fmt.Errorf("cross-package variable move cannot prove the moved initializer")
		}
		for _, file := range destination.Syntax {
			if filepath.Clean(destination.Fset.Position(file.Pos()).Filename) == snapshot.destinationPath {
				currentInitial = destination.TypesInfo.Types[spec.Values[0]].Value
			}
		}
		if currentInitial == nil || target.variable.initial == nil || !constant.Compare(target.variable.initial, token.EQL, currentInitial) {
			return fmt.Errorf("cross-package variable move changed the initial value")
		}
	} else if target.variable.initial != nil {
		return fmt.Errorf("cross-package variable move changed the zero initializer")
	}
	return validateCrossPackageMoveReferenceBindings(after, target.destinationPackage.PkgPath, moved, expected)
}

func findMovedVariableDeclaration(pkg *packages.Package, path, name string) (*ast.GenDecl, *ast.ValueSpec) {
	for _, file := range pkg.Syntax {
		if filepath.Clean(pkg.Fset.Position(file.Pos()).Filename) != filepath.Clean(path) {
			continue
		}
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR {
				continue
			}
			for _, raw := range group.Specs {
				spec, ok := raw.(*ast.ValueSpec)
				if !ok || len(spec.Names) != 1 || spec.Names[0].Name != name {
					continue
				}
				return group, spec
			}
		}
	}
	return nil, nil
}
