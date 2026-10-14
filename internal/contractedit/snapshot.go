// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file loads package data and captures a stable source view for type checks.
package contractedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// LoadSnapshot captures the selected module's package graph and source read set.
func LoadSnapshot(ctx context.Context, workDir string, selectedPaths []string) (*Snapshot, error) {
	root, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve contract workspace: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("canonicalize contract workspace: %w", err)
	}
	buffer := sourcebuf.FromContext(ctx)
	analysisCtx := ctx
	if buffer == nil {
		buffer, err = sourcebuf.New(root)
		if err != nil {
			return nil, fmt.Errorf("bind contract source snapshot: %w", err)
		}
		analysisCtx = sourcebuf.WithBuffer(ctx, buffer)
	}
	moduleRoot, err := FindModuleRoot(analysisCtx, root, selectedPaths)
	if err != nil {
		return nil, err
	}
	if authorizeErr := buffer.AuthorizeWrites(analysisCtx, moduleRoot); authorizeErr != nil {
		return nil, fmt.Errorf("authorize contract module writes: %w", authorizeErr)
	}
	environment, err := gocache.Environment(analysisCtx, moduleRoot)
	if err != nil {
		return nil, fmt.Errorf("prepare contract Go environment: %w", err)
	}
	readContents, err := readContractModuleFiles(analysisCtx, moduleRoot)
	if err != nil {
		return nil, err
	}
	parsedContents := make(map[string][]byte)
	var parsedContentsMu sync.Mutex
	loaded, err := loadContractPackages(analysisCtx, root, moduleRoot, environment, parsedContents, &parsedContentsMu)
	if err != nil {
		return nil, err
	}
	if err := rejectLocalReplacementModules(loaded); err != nil {
		return nil, err
	}
	if err := analysisCtx.Err(); err != nil {
		return nil, fmt.Errorf("load current contract source overlay: %w", err)
	}
	snapshot := newSnapshot(analysisCtx, buffer, root, moduleRoot, environment)
	visitErr := captureContractPackages(loaded, snapshot, root, parsedContents, readContents)
	if visitErr != nil {
		return nil, visitErr
	}
	if len(snapshot.Contents) == 0 {
		return nil, fmt.Errorf("contract module has no readable Go source files")
	}
	if err := recheckContractInputs(analysisCtx, readContents); err != nil {
		return nil, err
	}
	if err := recordContractReadSet(snapshot, root, readContents); err != nil {
		return nil, err
	}
	if err := requireContractPackages(snapshot, root, selectedPaths); err != nil {
		return nil, err
	}
	if err := ValidatePackageGraph(snapshot.ByPath, snapshot.RequiredPkgPaths); err != nil {
		return nil, fmt.Errorf("validate required contract package graph: %w", err)
	}
	return snapshot, nil
}

func rejectLocalReplacementModules(loaded []*packages.Package) error {
	var replacementErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if replacementErr != nil || pkg == nil || pkg.Module == nil || pkg.Module.Replace == nil {
			return replacementErr == nil
		}
		replacement := pkg.Module.Replace
		if replacement.Dir != "" && replacement.Version == "" {
			replacementErr = fmt.Errorf("contract edits do not support mutable local replacement module %q outside the captured module", pkg.Module.Path)
			return false
		}
		return true
	}, nil)
	return replacementErr
}

func readContractModuleFiles(ctx context.Context, moduleRoot string) (map[string][]byte, error) {
	contents := make(map[string][]byte)
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(moduleRoot, name)
		data, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			if name == "go.sum" && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read contract module file %q: %w", name, err)
		}
		contents[path] = data
	}
	return contents, nil
}

func loadContractPackages(ctx context.Context, root, moduleRoot string, environment []string, parsed map[string][]byte, parsedMu *sync.Mutex) ([]*packages.Package, error) {
	mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedModule
	loaded, err := packages.Load(&packages.Config{
		Context: ctx, Dir: moduleRoot, Env: environment, BuildFlags: []string{"-mod=readonly"}, Mode: mode,
		Overlay: sourcebuf.Overlay(ctx),
		ParseFile: func(fset *token.FileSet, filename string, source []byte) (*ast.File, error) {
			parsedMu.Lock()
			defer parsedMu.Unlock()
			return ParseSource(ctx, root, fset, filename, source, parsed, nil)
		},
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("load current contract source overlay: %w", err)
	}
	return loaded, nil
}

func newSnapshot(ctx context.Context, buffer *sourcebuf.Buffer, root, moduleRoot string, environment []string) *Snapshot {
	return &Snapshot{
		Context: ctx, Buffer: buffer, Root: root, ModuleRoot: moduleRoot, Env: environment,
		Packages: make([]*packages.Package, 0), ByPath: make(map[string]*packages.Package),
		Syntax: make(map[string]*ast.File), Contents: make(map[string][]byte),
	}
}

func captureContractPackages(loaded []*packages.Package, snapshot *Snapshot, root string, parsed, readContents map[string][]byte) error {
	var visitErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if visitErr != nil {
			return false
		}
		if pkg == nil {
			return true
		}
		visitErr = captureContractPackage(snapshot, root, pkg, parsed, readContents)
		return visitErr == nil
	}, nil)
	return visitErr
}

func captureContractPackage(snapshot *Snapshot, root string, pkg *packages.Package, parsed, readContents map[string][]byte) error {
	snapshot.Packages = append(snapshot.Packages, pkg)
	if pkg.PkgPath != "" && snapshot.ByPath[pkg.PkgPath] == nil {
		snapshot.ByPath[pkg.PkgPath] = pkg
	}
	for index, compiled := range pkg.CompiledGoFiles {
		if index >= len(pkg.Syntax) {
			continue
		}
		canonical, err := canonicalCapturedContractFile(compiled, "compiled Go file")
		if err != nil {
			return err
		}
		snapshot.Syntax[canonical] = pkg.Syntax[index]
	}
	for _, file := range pkg.GoFiles {
		canonical, err := canonicalCapturedContractFile(file, "package Go file")
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, canonical)
		if err != nil {
			return fmt.Errorf("relativize package Go file %q: %w", canonical, err)
		}
		if isContractOutsideRoot(relative) {
			continue
		}
		contents, exists := parsed[canonical]
		if !exists {
			return fmt.Errorf("go source %q was not captured by the package parser", canonical)
		}
		readContents[canonical] = bytes.Clone(contents)
		snapshot.Contents[canonical] = bytes.Clone(contents)
	}
	return nil
}

func canonicalCapturedContractFile(path, description string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", description, path, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize %s %q: %w", description, path, err)
	}
	return canonical, nil
}

func isContractOutsideRoot(relative string) bool {
	return relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func recheckContractInputs(ctx context.Context, readContents map[string][]byte) error {
	for path, expected := range readContents {
		actual, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return fmt.Errorf("recheck analyzed contract input %q: %w", path, err)
		}
		if !bytes.Equal(actual, expected) {
			return fmt.Errorf("contract input %q changed while packages.Load analyzed it", path)
		}
	}
	return nil
}

func recordContractReadSet(snapshot *Snapshot, root string, readContents map[string][]byte) error {
	for path, data := range readContents {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relativize contract read-set path %q: %w", path, err)
		}
		if isContractOutsideRoot(relative) {
			continue
		}
		snapshot.ReadSet = append(snapshot.ReadSet, Revision{Path: filepath.ToSlash(relative), Revision: Digest(data)})
	}
	slices.SortFunc(snapshot.ReadSet, func(left, right Revision) int {
		return strings.Compare(left.Path, right.Path)
	})
	return nil
}

func requireContractPackages(snapshot *Snapshot, root string, selectedPaths []string) error {
	for _, selected := range selectedPaths {
		path, err := CanonicalPath(root, selected)
		if err != nil {
			return err
		}
		pkg := snapshot.PackageForFile(path)
		if pkg == nil {
			return fmt.Errorf("selected Go file %q is outside the loaded module package set", selected)
		}
		if snapshot.RequiredPkgPaths == nil {
			snapshot.RequiredPkgPaths = make(map[string]bool)
		}
		snapshot.RequiredPkgPaths[pkg.PkgPath] = true
	}
	return nil
}
