// WHY: This file owns post-edit package loading, test-variant validation, and final type checks for contract edits.

package operation

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"semedit/internal/sourcebuf"
)

type contractPreflightState struct {
	snapshot             *contractSnapshot
	destination          string
	candidate            []byte
	destinationDirectory string
	overlay              map[string][]byte
	parsedTests          map[string][]byte
	parsedTestsMu        sync.Mutex
	capturedSource       map[string][]byte
	capturedSourceMu     sync.Mutex
}

func newContractPreflightState(snapshot *contractSnapshot, destination string, candidate []byte) (*contractPreflightState, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("contract analysis snapshot is required")
	}
	if snapshot.packageForFile(destination) == nil {
		return nil, fmt.Errorf("destination file %q is not in the analyzed package set", destination)
	}
	destinationDirectory, err := canonicalContractDirectory(filepath.Dir(destination))
	if err != nil {
		return nil, fmt.Errorf("resolve contract destination directory: %w", err)
	}
	overlay := sourcebuf.Overlay(snapshot.Context)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[destination] = bytes.Clone(candidate)
	return &contractPreflightState{
		snapshot:             snapshot,
		destination:          destination,
		candidate:            candidate,
		destinationDirectory: destinationDirectory,
		overlay:              overlay,
		parsedTests:          make(map[string][]byte),
		capturedSource:       make(map[string][]byte),
	}, nil
}

func loadContractPreflightPackages(ctx context.Context, state *contractPreflightState) ([]*packages.Package, error) {
	loaded, err := packages.Load(&packages.Config{
		Context:    ctx,
		Dir:        state.snapshot.ModuleRoot,
		Env:        state.snapshot.Env,
		BuildFlags: []string{"-mod=readonly"},
		Mode:       contractPackageLoadMode(),
		Overlay:    state.overlay,
		Tests:      true,
		ParseFile:  state.parseSource,
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("type-check generated contract output: %w", err)
	}
	return loaded, nil
}

func contractPackageLoadMode() packages.LoadMode {
	return packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax |
		packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedModule
}

func (state *contractPreflightState) parseSource(fset *token.FileSet, filename string, source []byte) (*ast.File, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolve preflight Go source %q: %w", filename, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("canonicalize preflight Go source %q: %w", filename, err)
	}
	if canonical == state.destination {
		return state.parseCapturedSource(fset, filename, state.candidate)
	}
	if symlinkErr := state.rejectTestSymlink(filename); symlinkErr != nil {
		return nil, symlinkErr
	}
	current, insideRoot, err := state.currentWorkspaceSource(canonical, source)
	if err != nil {
		return nil, err
	}
	if !insideRoot {
		return parser.ParseFile(fset, filename, source, parser.ParseComments|parser.AllErrors)
	}
	if strings.HasSuffix(canonical, "_test.go") {
		state.recordTestSource(canonical, current)
	}
	return state.parseCapturedSource(fset, filename, current)
}

func (state *contractPreflightState) rejectTestSymlink(filename string) error {
	if !strings.HasSuffix(filename, "_test.go") {
		return nil
	}
	logicalDirectory, err := canonicalContractDirectory(filepath.Dir(filename))
	if err != nil {
		return fmt.Errorf("resolve logical test source directory %q: %w", filename, err)
	}
	if logicalDirectory != state.destinationDirectory {
		return nil
	}
	return contractTestSymlinkError(state.snapshot.Root, filename)
}

func (state *contractPreflightState) currentWorkspaceSource(canonical string, source []byte) ([]byte, bool, error) {
	relative, err := filepath.Rel(state.snapshot.Root, canonical)
	if err != nil {
		return nil, false, fmt.Errorf("check preflight Go source %q against snapshot root: %w", canonical, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, false, nil
	}
	current, err := sourcebuf.ReadFile(state.snapshot.Context, canonical)
	if err != nil {
		return nil, false, fmt.Errorf("read preflight Go source %q: %w", canonical, err)
	}
	if source != nil && !bytes.Equal(current, source) {
		return nil, false, fmt.Errorf("preflight Go source %q changed while packages.Load analyzed it", canonical)
	}
	return current, true, nil
}

func (state *contractPreflightState) recordTestSource(path string, source []byte) {
	state.parsedTestsMu.Lock()
	defer state.parsedTestsMu.Unlock()
	state.parsedTests[path] = bytes.Clone(source)
}

func (state *contractPreflightState) parseCapturedSource(fset *token.FileSet, filename string, source []byte) (*ast.File, error) {
	state.capturedSourceMu.Lock()
	defer state.capturedSourceMu.Unlock()
	return parseContractSource(state.snapshot.Context, state.snapshot.Root, fset, filename, source, state.capturedSource, nil)
}

func validateContractPreflightPackages(snapshot *contractSnapshot, state *contractPreflightState, loaded []*packages.Package) (map[string]*packages.Package, *packages.Package, error) {
	if err := validateContractTestVariants(loaded, state.destinationDirectory); err != nil {
		return nil, nil, err
	}
	finalPackages := collectPrimaryContractPackages(snapshot, loaded)
	originalPackage := snapshot.packageForFile(state.destination)
	requiredPackages := requiredContractPackages(snapshot, originalPackage)
	if err := validateContractPackageGraph(finalPackages, requiredPackages); err != nil {
		return nil, nil, fmt.Errorf("validate final contract package graph: %w", err)
	}
	finalPackage := finalPackages[originalPackage.PkgPath]
	if err := validateGeneratedContractPackage(finalPackage, originalPackage.PkgPath); err != nil {
		return nil, nil, err
	}
	return finalPackages, finalPackage, nil
}

func collectPrimaryContractPackages(snapshot *contractSnapshot, loaded []*packages.Package) map[string]*packages.Package {
	primaryPackagePaths := make(map[string]string, len(snapshot.ByPath))
	for path, pkg := range snapshot.ByPath {
		if pkg != nil {
			primaryPackagePaths[pkg.ID] = path
		}
	}
	finalPackages := make(map[string]*packages.Package)
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if pkg == nil {
			return true
		}
		path, primary := primaryPackagePaths[pkg.ID]
		if primary && pkg.PkgPath == path {
			finalPackages[path] = pkg
		}
		return true
	}, nil)
	return finalPackages
}

func requiredContractPackages(snapshot *contractSnapshot, original *packages.Package) map[string]bool {
	required := make(map[string]bool, len(snapshot.RequiredPkgPaths)+1)
	required[original.PkgPath] = true
	for path := range snapshot.RequiredPkgPaths {
		required[path] = true
	}
	return required
}

func validateGeneratedContractPackage(pkg *packages.Package, packagePath string) error {
	if pkg == nil || pkg.Types == nil {
		return fmt.Errorf("generated output package %q did not type-check", packagePath)
	}
	if len(pkg.Errors) > 0 {
		return fmt.Errorf("generated output package %q has type errors: %s", packagePath, pkg.Errors[0].Msg)
	}
	return nil
}

func recordContractPreflightTests(snapshot *contractSnapshot, parsedTests map[string][]byte) error {
	for path, data := range parsedTests {
		relative, err := filepath.Rel(snapshot.Root, path)
		if err != nil {
			return fmt.Errorf("relativize preflight test source %q: %w", path, err)
		}
		snapshot.ReadSet = append(snapshot.ReadSet, contractRevision{Path: filepath.ToSlash(relative), Revision: contractDigest(data)})
	}
	slices.SortFunc(snapshot.ReadSet, func(left, right contractRevision) int {
		return strings.Compare(left.Path, right.Path)
	})
	return nil
}
