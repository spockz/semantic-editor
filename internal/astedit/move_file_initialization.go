// Package astedit proves import relocation does not introduce observable package initialization effects during file moves.
package astedit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// validateMoveFileInitialization proves every affected package variant keeps
// the same ordered package-initialization effects after relocation.
func validateMoveFileInitialization(before, after []*packages.Package, sourcePath, destinationPath, companionSource, companionDestination string) error {
	beforeRoots, err := moveFileInitializationRoots(before, sourcePath, destinationPath, companionSource)
	if err != nil {
		return err
	}
	afterRoots, err := moveFileInitializationRoots(after, sourcePath, destinationPath, companionDestination)
	if err != nil {
		return err
	}
	beforePackages := moveFilePackageIndex(before)
	afterPackages := moveFilePackageIndex(after)
	for key := range beforeRoots {
		if _, ok := afterRoots[key]; !ok {
			return fmt.Errorf("move_file cannot compare initialization root %s after relocation", key)
		}
	}
	for key := range afterRoots {
		if _, ok := beforeRoots[key]; !ok {
			return fmt.Errorf("move_file cannot compare initialization root %s before relocation", key)
		}
	}
	for key, beforeRoot := range beforeRoots {
		afterRoot := afterRoots[key]
		if err := validateMoveFileInitializationEvents(key, beforeRoot, afterRoot, beforePackages, afterPackages); err != nil {
			return err
		}
	}
	return nil
}

func validateMoveFileInitializationEvents(root string, before, after []*types.Package, beforePackages, afterPackages map[*types.Package]*packages.Package) error {
	beforeEvents, err := moveFileInitializationEventsForRoots(before, beforePackages)
	if err != nil {
		return fmt.Errorf("move_file cannot prove initialization before relocation for %s: %w", root, err)
	}
	afterEvents, err := moveFileInitializationEventsForRoots(after, afterPackages)
	if err != nil {
		return fmt.Errorf("move_file cannot prove initialization after relocation for %s: %w", root, err)
	}
	if !slices.Equal(beforeEvents, afterEvents) {
		return fmt.Errorf("move_file refuses initialization-changing import graph for %s", root)
	}
	return nil
}

func moveFileInitializationEvents(root *types.Package, index map[*types.Package]*packages.Package) ([]string, error) {
	return moveFileInitializationEventsForRoots([]*types.Package{root}, index)
}

func moveFileInitializationEventsForRoots(roots []*types.Package, index map[*types.Package]*packages.Package) ([]string, error) {
	packagesByPath, imports, dependents, err := moveFileInitializationGraph(roots, index)
	if err != nil {
		return nil, err
	}
	ready := make([]*types.Package, 0, len(packagesByPath))
	for pkg := range packagesByPath {
		if len(imports[pkg]) == 0 {
			ready = append(ready, pkg)
		}
	}
	slices.SortFunc(ready, compareMoveFilePackagePaths)
	events := make([]string, 0)
	processed := 0
	for len(ready) != 0 {
		current := ready[0]
		ready = ready[1:]
		processed++
		pkg := index[current]
		effectful, err := moveFilePackageHasInitializationEffect(pkg)
		if err != nil {
			return nil, err
		}
		if effectful {
			events = append(events, current.Path())
		}
		for dependent := range dependents[current] {
			delete(imports[dependent], current)
			if len(imports[dependent]) == 0 {
				ready = append(ready, dependent)
			}
		}
		slices.SortFunc(ready, compareMoveFilePackagePaths)
	}
	if processed != len(packagesByPath) {
		return nil, fmt.Errorf("cannot order initialization packages")
	}
	return events, nil
}

func moveFileInitializationGraph(roots []*types.Package, index map[*types.Package]*packages.Package) (map[*types.Package]bool, map[*types.Package]map[*types.Package]bool, map[*types.Package]map[*types.Package]bool, error) {
	packagesByPath := make(map[*types.Package]bool)
	firstByPath := make(map[string]*types.Package)
	imports := make(map[*types.Package]map[*types.Package]bool)
	dependents := make(map[*types.Package]map[*types.Package]bool)
	seen := make(map[*types.Package]bool)
	var collect func(*types.Package) error
	collect = func(current *types.Package) error {
		if current == types.Unsafe || seen[current] {
			return nil
		}
		seen[current] = true
		if current.Path() == "" {
			return fmt.Errorf("cannot uniquely identify %s", current.Path())
		}
		pkg := index[current]
		if err := moveFileValidateInitializationPackage(current, pkg, index, firstByPath); err != nil {
			return err
		}
		packagesByPath[current] = true
		imports[current] = make(map[*types.Package]bool)
		return moveFileCollectInitializationImports(current, imports, dependents, collect)
	}
	for _, root := range roots {
		if err := collect(root); err != nil {
			return nil, nil, nil, err
		}
	}
	return packagesByPath, imports, dependents, nil
}

func moveFileValidateInitializationPackage(current *types.Package, pkg *packages.Package, index map[*types.Package]*packages.Package, firstByPath map[string]*types.Package) error {
	if pkg == nil || pkg.TypesInfo == nil || len(pkg.Errors) != 0 || len(pkg.Syntax) == 0 {
		return fmt.Errorf("cannot inspect %s", current.Path())
	}
	first := firstByPath[current.Path()]
	if first == nil {
		firstByPath[current.Path()] = current
		return nil
	}
	firstEffect, err := moveFilePackageHasInitializationEffect(index[first])
	if err != nil {
		return err
	}
	currentEffect, err := moveFilePackageHasInitializationEffect(pkg)
	if err != nil {
		return err
	}
	if firstEffect || currentEffect {
		return fmt.Errorf("cannot uniquely identify effectful package %s", current.Path())
	}
	return nil
}

func moveFileCollectInitializationImports(current *types.Package, imports, dependents map[*types.Package]map[*types.Package]bool, collect func(*types.Package) error) error {
	for _, imported := range current.Imports() {
		if imported == nil {
			return fmt.Errorf("cannot inspect an import of %s", current.Path())
		}
		if imported != types.Unsafe {
			imports[current][imported] = true
			if dependents[imported] == nil {
				dependents[imported] = make(map[*types.Package]bool)
			}
			dependents[imported][current] = true
		}
		if err := collect(imported); err != nil {
			return err
		}
	}
	return nil
}

func compareMoveFilePackagePaths(left, right *types.Package) int {
	return strings.Compare(left.Path(), right.Path())
}

func moveFilePackageHasInitializationEffect(pkg *packages.Package) (bool, error) {
	for _, file := range pkg.Syntax {
		for _, spec := range file.Imports {
			if spec.Path.Value == "\"C\"" {
				return true, nil
			}
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "init" {
				return true, nil
			}
			if group, ok := decl.(*ast.GenDecl); ok && group.Tok == token.VAR {
				for _, spec := range group.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						return false, fmt.Errorf("cannot inspect package variable in %s", pkg.PkgPath)
					}
					if len(value.Values) != 0 {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func moveFileInitializationRoots(pkgs []*packages.Package, sourcePath, destinationPath, companionPath string) (map[string][]*types.Package, error) {
	roots := make(map[string][]*types.Package)
	sourcePackagePath := moveFilePackagePathForFile(pkgs, sourcePath)
	destinationPackagePath := moveFilePackagePathForFile(pkgs, destinationPath)
	byID := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		byID[pkg.ID] = pkg
	}
	for _, pkg := range pkgs {
		if pkg.Name == "main" && strings.HasSuffix(pkg.ID, ".test") {
			base := strings.TrimSuffix(pkg.ID, ".test")
			if base == sourcePackagePath || base == destinationPackagePath {
				continue
			}
		}
		if pkg.ID == pkg.PkgPath || !moveFileAffectedTestVariant(pkg, sourcePackagePath, destinationPackagePath) {
			roots["package:"+pkg.ID] = []*types.Package{pkg.Types}
			continue
		}
		if err := moveFileAddTestVariantRoots(roots, pkg, byID, sourcePackagePath, destinationPackagePath, companionPath); err != nil {
			return nil, err
		}
	}
	for _, base := range []string{sourcePackagePath, destinationPackagePath} {
		var combined []*types.Package
		for _, pkg := range pkgs {
			if moveFileAffectedTestVariant(pkg, sourcePackagePath, destinationPackagePath) && moveFileTestVariantBase(pkg, sourcePackagePath, destinationPackagePath) == base {
				combined = append(combined, roots["test:"+pkg.ID]...)
			}
		}
		if len(combined) != 0 {
			roots["test-program:"+base] = combined
			companion := make([]*types.Package, 0, len(roots["matching-test:internal"])+len(roots["matching-test:external"]))
			companion = append(companion, roots["matching-test:internal"]...)
			companion = append(companion, roots["matching-test:external"]...)
			if len(companion) != 0 {
				roots["test-program-with-companion:"+base] = append(slices.Clone(combined), companion...)
			}
		}
	}
	return roots, nil
}

func moveFileAddTestVariantRoots(roots map[string][]*types.Package, pkg *packages.Package, byID map[string]*packages.Package, sourcePackagePath, destinationPackagePath, companionPath string) error {
	base := moveFileTestVariantBase(pkg, sourcePackagePath, destinationPackagePath)
	production := byID[base]
	if production == nil || production.Types == nil {
		return fmt.Errorf("move_file cannot inspect production package for %s", pkg.ID)
	}
	support, err := moveFileTestRunnerSupport(byID[base+".test"], base)
	if err != nil {
		return err
	}
	stable := append([]*types.Package{production.Types}, support...)
	remaining := false
	for _, file := range pkg.Syntax {
		path := filepath.Clean(pkg.Fset.Position(file.Pos()).Filename)
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		imports, importErr := moveFileSyntaxImports(pkg, file)
		if importErr != nil {
			return importErr
		}
		if companionPath != "" && path == filepath.Clean(companionPath) {
			if err := moveFileAddMatchingTestRoot(roots, file, production.Types, support, imports); err != nil {
				return err
			}
			continue
		}
		stable = append(stable, imports...)
		remaining = remaining || moveFileCompanionHasDeclarations(file)
	}
	if remaining {
		roots["test:"+pkg.ID] = stable
	}
	return nil
}

func moveFileAddMatchingTestRoot(roots map[string][]*types.Package, file *ast.File, production *types.Package, support, imports []*types.Package) error {
	if !moveFileCompanionHasDeclarations(file) {
		return nil
	}
	key := "matching-test:internal"
	if strings.HasSuffix(file.Name.Name, "_test") {
		key = "matching-test:external"
	}
	if _, duplicate := roots[key]; duplicate {
		return fmt.Errorf("move_file cannot uniquely identify initialization root %s", key)
	}
	roots[key] = append(append([]*types.Package{production}, support...), imports...)
	return nil
}

func moveFilePackagePathForFile(pkgs []*packages.Package, path string) string {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if filepath.Clean(pkg.Fset.Position(file.Pos()).Filename) == filepath.Clean(path) {
				return pkg.PkgPath
			}
		}
	}
	return ""
}

func moveFileAffectedTestVariant(pkg *packages.Package, sourcePackagePath, destinationPackagePath string) bool {
	if pkg.ID == pkg.PkgPath {
		return false
	}
	for _, base := range []string{sourcePackagePath, destinationPackagePath} {
		if base != "" && strings.Contains(pkg.ID, "["+base+".test]") && (pkg.PkgPath == base || pkg.PkgPath == base+"_test") {
			return true
		}
	}
	return false
}

func moveFileTestVariantBase(pkg *packages.Package, sourcePackagePath, destinationPackagePath string) string {
	if pkg.PkgPath == sourcePackagePath || pkg.PkgPath == sourcePackagePath+"_test" {
		return sourcePackagePath
	}
	return destinationPackagePath
}

func moveFileTestRunnerSupport(runner *packages.Package, base string) ([]*types.Package, error) {
	if runner == nil || runner.Types == nil {
		return nil, fmt.Errorf("move_file cannot inspect test runner for %s", base)
	}
	var support []*types.Package
	for _, imported := range runner.Types.Imports() {
		if imported.Path() != base && imported.Path() != base+"_test" {
			support = append(support, imported)
		}
	}
	return support, nil
}

func moveFileSyntaxImports(pkg *packages.Package, file *ast.File) ([]*types.Package, error) {
	imports := make([]*types.Package, 0, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("move_file cannot inspect import in %s: %w", pkg.ID, err)
		}
		imported := pkg.Imports[path]
		if imported == nil || imported.Types == nil {
			return nil, fmt.Errorf("move_file cannot inspect import %s in %s", path, pkg.ID)
		}
		imports = append(imports, imported.Types)
	}
	return imports, nil
}

func moveFileCompanionHasDeclarations(file *ast.File) bool {
	for _, decl := range file.Decls {
		if group, ok := decl.(*ast.GenDecl); !ok || group.Tok != token.IMPORT {
			return true
		}
	}
	return false
}

func moveFilePackageIndex(pkgs []*packages.Package) map[*types.Package]*packages.Package {
	index := make(map[*types.Package]*packages.Package)
	for _, root := range pkgs {
		packages.Visit([]*packages.Package{root}, func(pkg *packages.Package) bool {
			if pkg.Types == nil {
				return true
			}
			index[pkg.Types] = pkg
			return true
		}, nil)
	}
	return index
}
