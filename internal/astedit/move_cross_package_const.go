// Package astedit isolates constant relocation because its compile-time value and type must survive reference and package changes exactly.
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

func resolveCrossPackageConstant(snapshot crossPackageMoveSnapshot, symbolQuery string, selected crossPackagePackages) (*crossPackageConstant, bool, error) {
	for _, declaration := range selected.sourceFile.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
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
			constantMove, err := validateCrossPackageConstant(snapshot, symbolQuery, selected, group, spec)
			if err != nil {
				return nil, true, err
			}
			return constantMove, true, nil
		}
	}
	return nil, false, nil
}

func validateCrossPackageConstant(snapshot crossPackageMoveSnapshot, name string, selected crossPackagePackages, declaration *ast.GenDecl, spec *ast.ValueSpec) (*crossPackageConstant, error) {
	if declaration.Lparen.IsValid() || len(declaration.Specs) != 1 || len(spec.Names) != 1 || len(spec.Values) != 1 {
		return nil, fmt.Errorf("cross-package constant move requires one standalone ungrouped constant")
	}
	identifier := spec.Names[0]
	if identifier.Name != name || !ast.IsExported(name) {
		return nil, fmt.Errorf("cross-package constant move requires one exported constant")
	}
	object, ok := selected.source.TypesInfo.Defs[identifier].(*types.Const)
	if !ok || object == nil || object.Pkg() == nil || object.Pkg().Path() != selected.source.PkgPath || filepath.Clean(selected.source.Fset.Position(object.Pos()).Filename) != snapshot.sourcePath {
		return nil, fmt.Errorf("cross-package constant move cannot resolve the selected declaration")
	}
	if declarationNameCollision(selected.destFile, declaration) {
		return nil, fmt.Errorf("destination already declares %q", name)
	}
	if err := rejectCrossPackageConstantTypeDependencies(selected.source, spec.Type); err != nil {
		return nil, err
	}
	if _, ok := object.Type().(*types.Basic); !ok {
		return nil, fmt.Errorf("cross-package constant move supports only predeclared basic constant types")
	}
	basic := object.Type().Underlying().(*types.Basic)
	if basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) == 0 {
		return nil, fmt.Errorf("cross-package constant move supports only predeclared basic constant types")
	}
	if object.Val() == nil {
		return nil, fmt.Errorf("cross-package constant move cannot prove the source constant type and value")
	}
	if err := validateCrossPackagePureInitializer(selected.source, spec); err != nil {
		return nil, err
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
	return &crossPackageConstant{declaration: declaration, spec: spec, object: object}, nil
}

func rejectCrossPackageConstantTypeDependencies(pkg *packages.Package, expression ast.Expr) error {
	if expression == nil {
		return nil
	}
	var dependency string
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok || dependency != "" {
			return true
		}
		if isMovePackageName(pkg.TypesInfo.Uses[identifier]) {
			dependency = identifier.Name
			return false
		}
		return true
	})
	if dependency != "" {
		return fmt.Errorf("cross-package constant move refuses package or import type dependency %q", dependency)
	}
	return nil
}

func isMovePackageName(object types.Object) bool {
	_, ok := object.(*types.PkgName)
	return ok
}

func buildCrossPackageConstantMoveChanges(ctx context.Context, snapshot crossPackageMoveSnapshot, target crossPackageMoveTarget, opts MoveConstructOptions) (crossPackageMoveChangeSet, error) {
	return buildCrossPackageValueMoveChanges(ctx, snapshot, target, opts, moveReferenceConstant, validateCrossPackageConstantMoveOverlay)
}

func validateCrossPackageConstantMoveOverlay(ctx context.Context, snapshot crossPackageMoveSnapshot, target crossPackageMoveTarget, changes map[string][]byte, expected map[moveReferenceKey][]moveReferenceExpectation) error {
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
	if err := validateCrossPackageInitializationGraph(target.packages, after); err != nil {
		return err
	}
	destination := packageByPath(after, target.destinationPackage.PkgPath)
	if destination == nil {
		return fmt.Errorf("cross-package constant move cannot identify the destination package")
	}
	moved, ok := destination.Types.Scope().Lookup(target.constant.object.Name()).(*types.Const)
	if !ok || moved.Pkg() == nil || moved.Pkg().Path() != target.destinationPackage.PkgPath || filepath.Clean(destination.Fset.Position(moved.Pos()).Filename) != snapshot.destinationPath {
		return fmt.Errorf("cross-package constant move cannot identify moved constant %q", target.constant.object.Name())
	}
	if !types.Identical(target.constant.object.Type(), moved.Type()) || !constant.Compare(target.constant.object.Val(), token.EQL, moved.Val()) {
		return fmt.Errorf("cross-package constant move changed the constant type or value")
	}
	return validateCrossPackageMoveReferenceBindings(after, target.destinationPackage.PkgPath, moved, expected)
}

func packageByPath(pkgs []*packages.Package, path string) *packages.Package {
	for _, pkg := range pkgs {
		if pkg.PkgPath == path {
			return pkg
		}
	}
	return nil
}
